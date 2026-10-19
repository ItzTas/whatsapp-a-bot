package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	auctionExtension    = 5 * time.Minute
	auctionMinStep      = 10
	auctionStepPercent  = 5
	auctionRelicChance  = 0.03
	auctionRelicPercent = 3000
	auctionSendLimit    = 4
)

type dailyAuction struct {
	chat   types.JID
	day    string
	item   item
	bid    int64
	bidder core.Account
	ends   time.Time
}

type auctionBid struct {
	lot      dailyAuction
	previous core.Account
	refunded int64
	extended bool
}

type dueAuction struct {
	chat types.JID
	day  string
}

func auctionGoods(it item) bool {
	return (it.rarity == epic || it.rarity == legendary) && !it.illegal && !it.shop && it.relicOdds == 0 && it.upgrades == "" && it.graft == nil
}

func auctionOpening(it item) int64 {
	if it.rarity == relic {
		return it.value * auctionRelicPercent / 100
	}
	if luxuryGoods(it) {
		return luxuryPrice(it)
	}
	return it.value
}

func auctionLot(chat types.JID, now time.Time) item {
	if dailyRand(chat, now, "leilao").Float64() < auctionRelicChance {
		return auctionRotation(chat, now, relics(), "leilao-reliquia")
	}
	return auctionRotation(chat, now, slices.DeleteFunc(slices.Clone(catalog), func(it item) bool { return !auctionGoods(it) }), "leilao-rodizio")
}

func auctionRotation(chat types.JID, now time.Time, pool []item, salt string) item {
	h := fnv.New64a()
	_, _ = h.Write([]byte(ledgers.of(chat).String() + salt))
	rand.New(rand.NewPCG(h.Sum64(), 0)).Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	return pool[localDay(now)%int64(len(pool))]
}

func todaysAuction(chat types.JID, now time.Time) dailyAuction {
	return dailyAuction{chat: chat, day: now.In(core.ReminderZone).Format(time.DateOnly), item: auctionLot(chat, now), ends: now.Add(nextRestock(now))}
}

func (a dailyAuction) nextBid() int64 {
	if a.bid == 0 {
		return auctionOpening(a.item)
	}
	return a.bid + max(auctionMinStep, a.bid*auctionStepPercent/100)
}

func (a dailyAuction) refusal(bidder core.Account, amount int64) string {
	if bidder.Key == a.bidder.Key {
		return fmt.Sprintf("Você já tá ganhando com %s 🏆 Espera alguém cobrir", Coins(a.bid))
	}
	if next := a.nextBid(); amount < next {
		return fmt.Sprintf("O lance tem que ser de pelo menos %s", Coins(next))
	}
	return ""
}

func (b *bot) leilao(ctx context.Context, evt *events.Message) error {
	if arg := core.FirstPlainArg(core.MessageText(evt.Message)); arg != "" {
		return b.bidAuction(ctx, evt, arg)
	}
	return b.sendAuctionStatus(ctx, evt.Info.Chat.ToNonAD())
}

func (b *bot) lance(ctx context.Context, evt *events.Message) error {
	return b.bidAuction(ctx, evt, core.FirstPlainArg(core.MessageText(evt.Message)))
}

func (b *bot) bidAuction(ctx context.Context, evt *events.Message, arg string) error {
	chat := evt.Info.Chat.ToNonAD()
	bidder := b.Account(ctx, evt.Info.Sender)
	amount, _, refusal, err := b.affordable(ctx, chat, bidder, arg, "!lance", 0)
	if err != nil {
		return err
	}
	if refusal != "" {
		return b.SendText(ctx, chat, refusal)
	}
	bid, refusal, err := b.Wallets.bidAuction(ctx, chat, bidder, amount, time.Now())
	if err != nil {
		return fmt.Errorf("dar lance no leilão: %w", err)
	}
	if refusal != "" {
		return b.SendText(ctx, chat, refusal)
	}

	lines := []string{fmt.Sprintf("💸 @%s deu um lance de %s em %s", bidder.JID.User, Coins(amount), bid.lot.item.label())}
	mentions := []types.JID{bidder.JID}
	if bid.refunded > 0 {
		lines = append(lines, fmt.Sprintf("↩️ @%s foi coberto e recebeu os %s de volta", bid.previous.JID.User, core.Thousands(bid.refunded)))
		mentions = append(mentions, bid.previous.JID)
	}
	if bid.extended {
		lines = append(lines, fmt.Sprintf("⏰ Lance no finzinho! O leilão foi prorrogado e fecha em %s", core.FormatWait(auctionExtension)))
	}
	lines = append(lines, "", fmt.Sprintf("_Pra cobrir: !lance %s ou mais_", core.Thousands(bid.lot.nextBid())))
	_, err = b.SendMentions(ctx, chat, strings.Join(lines, "\n"), mentions)
	return err
}

func (b *bot) sendAuctionStatus(ctx context.Context, chat types.JID) error {
	now := time.Now()
	a, err := b.Wallets.currentAuction(ctx, chat, now)
	if err != nil {
		return fmt.Errorf("ler leilão do dia: %w", err)
	}
	lines := []string{"🔨 *Leilão do dia*", "_Todo dia o martelo bate num item diferente, e quem der o maior lance leva na virada da meia-noite_", ""}
	if a.item.rarity == relic {
		lines = append(lines, "🔴 *Hoje tem relíquia!* Isso quase nunca aparece por aqui, e o lance mínimo é salgado")
	}
	lines = append(lines, fmt.Sprintf("📦 %s (%s)", a.item.label(), a.item.summary()))
	if p := perk(a.item); p != "" {
		lines = append(lines, "✨ "+p)
	}
	var mentions []types.JID
	price := "💰 Ainda sem lance, o mínimo é " + Coins(a.nextBid())
	if a.bid > 0 {
		price = fmt.Sprintf("💰 Lance atual: %s de @%s", Coins(a.bid), a.bidder.JID.User)
		mentions = append(mentions, a.bidder.JID)
	}
	lines = append(lines, price,
		fmt.Sprintf("⏰ Fecha em %s, e lance nos últimos %s prorroga", core.FormatWait(a.ends.Sub(now)), core.FormatWait(auctionExtension)),
		"",
		fmt.Sprintf("_Manda !lance %s ou mais pra entrar. As moedas saem na hora e voltam se alguém cobrir; as do lance vencedor somem no martelo_", core.Thousands(a.nextBid())),
	)
	_, err = b.SendMentions(ctx, chat, strings.Join(lines, "\n"), mentions)
	return err
}

func (b *bot) runAuctions(ctx context.Context) {
	ticker := time.NewTicker(core.ReminderTick)
	defer ticker.Stop()
	for {
		if b.Client.IsLoggedIn() {
			b.closeDueAuctions(ctx, time.Now())
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (b *bot) closeDueAuctions(ctx context.Context, now time.Time) {
	due, err := b.Wallets.dueAuctions(ctx, now)
	if err != nil {
		b.Client.Log.Errorf("procurar leilões vencidos: %v", err)
		return
	}
	var g errgroup.Group
	g.SetLimit(auctionSendLimit)
	for _, d := range due {
		g.Go(func() error {
			if err := b.closeAuction(ctx, d, now); err != nil {
				b.Client.Log.Errorf("fechar leilão de %s em %s: %v", d.day, d.chat, err)
			}
			return nil
		})
	}
	_ = g.Wait()
}

func (b *bot) closeAuction(ctx context.Context, d dueAuction, now time.Time) error {
	a, closed, err := b.Wallets.settleAuction(ctx, d, now)
	if err != nil || !closed {
		return err
	}
	next := todaysAuction(d.chat, now)
	text := fmt.Sprintf("🔨 *Dou-lhe uma, dou-lhe duas... vendido!*\n\n🏆 @%s arrematou %s no leilão do dia por %s, e as moedas sumiram no martelo\n\n📦 Já tem item novo na mesa: %s, a partir de %s. Manda !leilao pra ver",
		a.bidder.JID.User, a.item.label(), Coins(a.bid), next.item.label(), Coins(next.nextBid()))
	return b.announce(ctx, d.chat, text, a.bidder.JID)
}

func loadAuction(ctx context.Context, tx *ledgerTx, chat types.JID, now time.Time) (dailyAuction, error) {
	a := dailyAuction{chat: chat}
	var (
		id, bidderJID string
		ends          int64
	)
	err := tx.QueryRowContext(ctx,
		"SELECT dia, item, lance, licitante, licitante_jid, fecha_em FROM leiloes WHERE chat = ? AND fecha_em > ? ORDER BY dia LIMIT 1",
		chat.String(), now.Unix(),
	).Scan(&a.day, &id, &a.bid, &a.bidder.Key, &bidderJID, &ends)
	if errors.Is(err, sql.ErrNoRows) {
		return todaysAuction(chat, now), nil
	}
	if err != nil {
		return a, err
	}
	return a.fill(id, bidderJID, ends)
}

func (a dailyAuction) fill(id, bidderJID string, ends int64) (dailyAuction, error) {
	it, ok := itemByID(id)
	if !ok {
		return a, fmt.Errorf("item %q do leilão não existe mais", id)
	}
	jid, err := types.ParseJID(bidderJID)
	a.item, a.bidder.JID, a.ends = it, jid, time.Unix(ends, 0)
	return a, err
}

func (s *walletStore) currentAuction(ctx context.Context, chat types.JID, now time.Time) (a dailyAuction, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) (err error) {
		a, err = loadAuction(ctx, tx, chat, now)
		return err
	})
	return a, err
}

func (s *walletStore) bidAuction(ctx context.Context, chat types.JID, bidder core.Account, amount int64, now time.Time) (bid auctionBid, refusal string, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		if err := openWallet(ctx, tx, chat, bidder); err != nil {
			return err
		}
		a, err := loadAuction(ctx, tx, chat, now)
		if err != nil {
			return err
		}
		if refusal = a.refusal(bidder, amount); refusal != "" {
			return nil
		}
		if err := debitBalance(ctx, tx, chat, bidder, amount); err != nil {
			return err
		}
		if a.bid > 0 {
			_, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", a.bid, chat.String(), a.bidder.Key)
			if err != nil {
				return err
			}
		}
		bid.previous, bid.refunded = a.bidder, a.bid
		if a.ends.Sub(now) < auctionExtension {
			a.ends, bid.extended = now.Add(auctionExtension), true
		}
		a.bid, a.bidder = amount, bidder
		bid.lot = a
		_, err = tx.ExecContext(ctx,
			`INSERT INTO leiloes (chat, dia, item, lance, licitante, licitante_jid, fecha_em) VALUES (?, ?, ?, ?, ?, ?, ?)
			 ON CONFLICT (chat, dia) DO UPDATE SET lance = excluded.lance, licitante = excluded.licitante, licitante_jid = excluded.licitante_jid, fecha_em = excluded.fecha_em`,
			chat.String(), a.day, a.item.id, a.bid, a.bidder.Key, a.bidder.JID.String(), a.ends.Unix(),
		)
		return err
	})
	if errors.Is(err, ErrNoFunds) {
		return bid, StakeChanged, nil
	}
	return bid, refusal, err
}

func (s *walletStore) dueAuctions(ctx context.Context, now time.Time) (due []dueAuction, err error) {
	rows, err := s.db.QueryContext(ctx, "SELECT chat, dia FROM leiloes WHERE fecha_em <= ?", now.Unix())
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var (
			raw string
			d   dueAuction
		)
		if err := rows.Scan(&raw, &d.day); err != nil {
			return nil, err
		}
		if d.chat, err = types.ParseJID(raw); err != nil {
			return nil, err
		}
		if ledgers.linked(d.chat) {
			continue
		}
		due = append(due, d)
	}
	return due, rows.Err()
}

func (s *walletStore) settleAuction(ctx context.Context, d dueAuction, now time.Time) (a dailyAuction, closed bool, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		var (
			id, bidderJID string
			ends          int64
		)
		a = dailyAuction{chat: d.chat, day: d.day}
		err := tx.QueryRowContext(ctx,
			"DELETE FROM leiloes WHERE chat = ? AND dia = ? AND fecha_em <= ? RETURNING item, lance, licitante, licitante_jid, fecha_em",
			d.chat.String(), d.day, now.Unix(),
		).Scan(&id, &a.bid, &a.bidder.Key, &bidderJID, &ends)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if a, err = a.fill(id, bidderJID, ends); err != nil {
			return err
		}
		closed = true
		return addItems(ctx, tx, d.chat, a.bidder, a.item.id, 1)
	})
	return a, closed, err
}
