package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	partyPercent  = 2
	partyFloor    = 50_000
	partySpan     = time.Hour
	partyMaxSpan  = 6 * time.Hour
	partyPayBonus = 5
)

type party struct {
	until time.Time
	host  types.JID
	total int64
}

func (p party) on(now time.Time) bool {
	return p.until.After(now)
}

type partyThrow struct {
	party    party
	price    int64
	extended bool
	full     bool
}

func partyPrice(balance int64) int64 {
	return max(partyFloor, balance*partyPercent/100)
}

func (p Profession) partying(until int64, now time.Time) Profession {
	if until <= now.Unix() {
		return p
	}
	p.party = true
	p.min, p.max = p.min*(100+partyPayBonus)/100, p.max*(100+partyPayBonus)/100
	return p
}

func partyPerk() string {
	return fmt.Sprintf("🎉 Tá rolando festa no grupo: +%d%% no !trabalhar enquanto ela durar", partyPayBonus)
}

func (b *bot) festa(ctx context.Context, evt *events.Message) error {
	args := strings.Fields(core.FoldLetters(core.MessageText(evt.Message)))[1:]
	if len(args) > 0 && slices.Contains([]string{"pagar", "bancar", "dar", "fazer", "esticar"}, args[0]) {
		return b.throwParty(ctx, evt)
	}
	return b.partyStatus(ctx, evt)
}

func (b *bot) partyStatus(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	acc := b.Account(ctx, evt.Info.Sender.ToNonAD())
	now := time.Now()
	var (
		p       party
		balance int64
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { p, err = b.Wallets.partyOf(gctx, chat); return err })
	g.Go(func() (err error) { balance, _, err = b.Wallets.Balance(gctx, chat, acc); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler festa: %w", err)
	}
	lines := []string{"🔇 Nenhuma festa rolando agora. O grupo tá precisando de alguém generoso"}
	var people []types.JID
	if p.on(now) {
		lines = []string{
			fmt.Sprintf("🎉 *Festa rolando!* Bancada por @%s, vai até daqui a %s", p.host.User, core.FormatWait(p.until.Sub(now))),
			fmt.Sprintf("Enquanto durar, todo mundo do grupo ganha +%d%% no !trabalhar. Já torraram %s nela", partyPayBonus, Coins(p.total)),
		}
		people = append(people, p.host)
	}
	lines = append(lines, "",
		fmt.Sprintf("_Pra bancar (ou esticar) mais %s de festa: !festa pagar. Pra você sai %s (%d%% do saldo, no mínimo %s). Dá pra esticar até %s de uma vez, e o dinheiro some: vai tudo pro DJ e pro open bar_",
			core.FormatWait(partySpan), Coins(partyPrice(balance)), partyPercent, Coins(partyFloor), core.FormatWait(partyMaxSpan)))
	_, err := b.SendMentions(ctx, chat, strings.Join(lines, "\n"), people)
	return err
}

func (b *bot) throwParty(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	now := time.Now()
	t, err := b.Wallets.throwParty(ctx, chat, acc, now)
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, chat, fmt.Sprintf("💸 Bancar a festa custa %s pra você, e não tem isso no saldo. Festa é coisa de rico", Coins(t.price)))
	}
	if err != nil {
		return fmt.Errorf("bancar festa: %w", err)
	}
	if t.full {
		return b.SendText(ctx, chat, fmt.Sprintf("🎉 A festa já vai até daqui a %s, e não dá pra esticar mais que %s de uma vez. Espera ela baixar um pouco", core.FormatWait(t.party.until.Sub(now)), core.FormatWait(partyMaxSpan)))
	}
	head := fmt.Sprintf("🎉 @%s bancou a festa! Paredão ligado e open bar liberado", person.User)
	if t.extended {
		head = fmt.Sprintf("🎉 @%s esticou a festa! Mais uma rodada pra todo mundo", person.User)
	}
	text := fmt.Sprintf("%s\nAté daqui a %s, todo mundo do grupo ganha +%d%% no !trabalhar\n💸 Custou %s, e a festa já torrou %s",
		head, core.FormatWait(t.party.until.Sub(now)), partyPayBonus, Coins(t.price), Coins(t.party.total))
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (s *walletStore) partyOf(ctx context.Context, chat types.JID) (party, error) {
	return readParty(ctx, s.db, chat)
}

func readParty(ctx context.Context, q rowQuerier, chat types.JID) (party, error) {
	var (
		until int64
		host  string
		p     party
	)
	err := q.QueryRowContext(ctx, "SELECT ate, anfitriao, total FROM festas WHERE chat = ?", chat.String()).Scan(&until, &host, &p.total)
	if errors.Is(err, sql.ErrNoRows) {
		return party{}, nil
	}
	if err != nil {
		return party{}, err
	}
	p.until = time.Unix(until, 0)
	p.host, err = types.ParseJID(host)
	return p, err
}

func (s *walletStore) throwParty(ctx context.Context, chat types.JID, acc core.Account, now time.Time) (t partyThrow, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		var balance int64
		if err := tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&balance); err != nil {
			return err
		}
		t.price = partyPrice(balance)
		current, err := readParty(ctx, tx, chat)
		if err != nil {
			return err
		}
		t.party = current
		if current.until.Sub(now) > partyMaxSpan-partySpan {
			t.full = true
			return nil
		}
		if err := debitBalance(ctx, tx, chat, acc, t.price); err != nil {
			return err
		}
		t.extended = current.on(now)
		start, total := now, t.price
		if t.extended {
			start, total = current.until, current.total+t.price
		}
		t.party = party{until: start.Add(partySpan), host: acc.JID, total: total}
		_, err = tx.ExecContext(ctx,
			`INSERT INTO festas (chat, ate, anfitriao, total) VALUES (?, ?, ?, ?)
			 ON CONFLICT (chat) DO UPDATE SET ate = excluded.ate, anfitriao = excluded.anfitriao, total = excluded.total`,
			chat.String(), t.party.until.Unix(), acc.JID.String(), total,
		)
		return err
	})
	return t, err
}
