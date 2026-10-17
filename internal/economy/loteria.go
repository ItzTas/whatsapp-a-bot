package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	lotteryTicketPrice = 50
	lotteryMaxTickets  = 20
	lotteryHouseBonus  = 20
	lotteryHour        = 20
	lotteryTick        = time.Minute
)

var (
	lotteryZone    = time.FixedZone("BRT", -3*60*60)
	errTicketLimit = errors.New("limite de bilhetes por sorteio")
)

type lotteryTicket struct {
	owner core.Account
	count int64
}

type ticketPurchase struct {
	held    int64
	balance int64
}

type lotteryPool struct {
	tickets      int64
	participants int64
	mine         int64
}

type lotteryWin struct {
	winner types.JID
	prize  int64
}

type pendingLottery struct {
	chat types.JID
	at   time.Time
}

type lotteryDraw struct {
	tickets []lotteryTicket
	total   int64
	winner  lotteryTicket
	prize   int64
}

func (b *bot) loteria(ctx context.Context, evt *events.Message) error {
	arg := core.FirstPlainArg(core.MessageText(evt.Message))
	if arg == "" {
		return b.lotteryStatus(ctx, evt)
	}
	return b.buyTickets(ctx, evt, arg)
}

func (b *bot) buyTickets(ctx context.Context, evt *events.Message, arg string) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	count, err := strconv.ParseInt(arg, 10, 64)
	if err != nil || count <= 0 {
		return b.SendText(ctx, chat, fmt.Sprintf("Diz quantos bilhetes quer, de 1 a %d: !loteria 3 (cada um custa %s)", lotteryMaxTickets, Coins(lotteryTicketPrice)))
	}
	if count > lotteryMaxTickets {
		return b.SendText(ctx, chat, fmt.Sprintf("O máximo é %d bilhetes por pessoa em cada sorteio 🎟️", lotteryMaxTickets))
	}

	now := time.Now()
	draw := nextLotteryDraw(now)
	cost := count * lotteryTicketPrice
	p, err := b.Wallets.buyTickets(ctx, chat, b.Account(ctx, person), draw, count)
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, chat, fmt.Sprintf("Você não tem moeda pra isso: %s custam %s. Confere no !saldo", core.CountLabel(int(count), "bilhete", "bilhetes"), Coins(cost)))
	}
	if errors.Is(err, errTicketLimit) {
		return b.SendText(ctx, chat, fmt.Sprintf("Você já tem %s e o máximo é %d por sorteio 🎟️", core.CountLabel(int(p.held), "bilhete", "bilhetes"), lotteryMaxTickets))
	}
	if err != nil {
		return fmt.Errorf("comprar bilhetes: %w", err)
	}
	text := fmt.Sprintf("🎟️ @%s comprou %s por %s e agora tem %d pro sorteio de %s (daqui a %s)\n💰 Saldo: %s",
		person.User, core.CountLabel(int(count), "bilhete", "bilhetes"), Coins(cost), p.held, drawLabel(draw, now), core.FormatWait(draw.Sub(now)), Coins(p.balance))
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) lotteryStatus(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	now := time.Now()
	draw := nextLotteryDraw(now)
	var (
		pool      lotteryPool
		last      lotteryWin
		hasLast   bool
		nicknames map[string]string
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) {
		pool, err = b.Wallets.lotteryPool(gctx, chat, b.Account(gctx, evt.Info.Sender), draw)
		return err
	})
	g.Go(func() (err error) { last, hasLast, err = b.Wallets.lastLotteryWin(gctx, chat); return err })
	g.Go(func() error { nicknames = b.ChatNicknames(gctx, chat); return nil })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler loteria: %w", err)
	}

	lines := append([]string{"🎟️ *Loteria do grupo*", ""}, lotteryPoolLines(pool)...)
	lines = append(lines, fmt.Sprintf("⏰ Sorteio %s, daqui a %s", drawLabel(draw, now), core.FormatWait(draw.Sub(now))))
	if hasLast {
		lines = append(lines, fmt.Sprintf("🏆 Último ganhador: %s, que levou %s", b.DisplayName(ctx, last.winner, nicknames), Coins(last.prize)))
	}
	lines = append(lines, "", fmt.Sprintf("_Compra com !loteria <quantidade>, a %s cada, até %d por sorteio_", Coins(lotteryTicketPrice), lotteryMaxTickets))
	return b.ReplyTo(ctx, evt, strings.Join(lines, "\n"))
}

func lotteryPoolLines(pool lotteryPool) []string {
	if pool.tickets == 0 {
		return []string{"🪹 Ninguém comprou bilhete pra esse sorteio ainda"}
	}
	lines := []string{
		fmt.Sprintf("💰 Pote: %s (já com %d%% de bônus da casa)", Coins(lotteryPot(pool.tickets)), lotteryHouseBonus),
		fmt.Sprintf("👥 %s, %s no total", core.CountLabel(int(pool.participants), "pessoa", "pessoas"), core.CountLabel(int(pool.tickets), "bilhete", "bilhetes")),
		myTicketsLine(pool),
	}
	if pool.participants == 1 {
		lines = append(lines, "_Se ninguém mais comprar, não tem sorteio e os bilhetes passam pro dia seguinte_")
	}
	return lines
}

func myTicketsLine(pool lotteryPool) string {
	if pool.mine == 0 {
		return "🎫 Você ainda não tem bilhete"
	}
	return fmt.Sprintf("🎫 Você tem %s: %s de chance", core.CountLabel(int(pool.mine), "bilhete", "bilhetes"), chance(pool.mine, pool.tickets))
}

func (b *bot) runLottery(ctx context.Context) {
	ticker := time.NewTicker(lotteryTick)
	defer ticker.Stop()
	for {
		if b.Client.IsLoggedIn() {
			b.drawDueLotteries(ctx)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (b *bot) drawDueLotteries(ctx context.Context) {
	now := time.Now()
	due, err := b.Wallets.dueLotteries(ctx, now)
	if err != nil {
		b.Client.Log.Errorf("procurar sorteios da loteria: %v", err)
		return
	}
	for _, pending := range due {
		if err := b.runDraw(ctx, pending, now); err != nil {
			b.Client.Log.Errorf("sortear a loteria de %s: %v", pending.chat, err)
		}
	}
}

func (b *bot) runDraw(ctx context.Context, pending pendingLottery, now time.Time) error {
	next := nextLotteryDraw(now)
	d, err := b.Wallets.drawLottery(ctx, pending, next)
	if err != nil || len(d.tickets) == 0 {
		return err
	}
	text, featured := lotteryResult(d, drawLabel(next, now))
	if _, err := b.SendMentions(ctx, pending.chat, text, []types.JID{featured}); err != nil {
		return fmt.Errorf("anunciar o resultado (as moedas já foram pagas): %w", err)
	}
	return nil
}

func lotteryResult(d lotteryDraw, nextDraw string) (string, types.JID) {
	if len(d.tickets) == 1 {
		only := d.tickets[0].owner.JID
		return fmt.Sprintf("🎟️ Só @%s comprou bilhete pra loteria, então não teve sorteio. Os bilhetes passam pro próximo, %s. Chama mais gente com !loteria <quantidade>",
			only.User, nextDraw), only
	}
	winner := d.winner.owner.JID
	lines := []string{
		"🎰 *Saiu o resultado da loteria!*",
		"",
		fmt.Sprintf("🏆 @%s levou o pote de %s com %d dos %d bilhetes", winner.User, Coins(d.prize), d.winner.count, d.total),
		fmt.Sprintf("👥 %d pessoas participaram", len(d.tickets)),
		"",
		fmt.Sprintf("_Próximo sorteio %s. Compra com !loteria <quantidade>_", nextDraw),
	}
	return strings.Join(lines, "\n"), winner
}

func nextLotteryDraw(now time.Time) time.Time {
	local := now.In(lotteryZone)
	draw := time.Date(local.Year(), local.Month(), local.Day(), lotteryHour, 0, 0, 0, lotteryZone)
	if draw.After(now) {
		return draw
	}
	return draw.AddDate(0, 0, 1)
}

func drawLabel(draw, now time.Time) string {
	day := "amanhã"
	if draw.In(lotteryZone).Day() == now.In(lotteryZone).Day() {
		day = "hoje"
	}
	return fmt.Sprintf("%s às %dh", day, lotteryHour)
}

func lotteryPot(tickets int64) int64 {
	return tickets * lotteryTicketPrice * (100 + lotteryHouseBonus) / 100
}

func drawWinner(tickets []lotteryTicket, total int64) lotteryTicket {
	n, i := rand.Int64N(total), 0
	for n >= tickets[i].count {
		n -= tickets[i].count
		i++
	}
	return tickets[i]
}

func chance(part, total int64) string {
	percent := math.Round(float64(part)*1000/float64(total)) / 10
	return strings.Replace(strconv.FormatFloat(percent, 'f', -1, 64), ".", ",", 1) + "%"
}

func (s *walletStore) buyTickets(ctx context.Context, chat types.JID, acc core.Account, draw time.Time, count int64) (p ticketPurchase, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		cost := count * lotteryTicketPrice
		err := tx.QueryRowContext(ctx,
			"UPDATE carteiras SET saldo = saldo - ? WHERE chat = ? AND pessoa = ? AND saldo >= ? RETURNING saldo",
			cost, chat.String(), acc.Key, cost,
		).Scan(&p.balance)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoFunds
		}
		if err != nil {
			return err
		}
		err = tx.QueryRowContext(ctx,
			`INSERT INTO loteria_bilhetes (chat, pessoa, jid, sorteio, quantidade) VALUES (?, ?, ?, ?, ?)
			 ON CONFLICT (chat, pessoa, sorteio) DO UPDATE
			 SET quantidade = loteria_bilhetes.quantidade + excluded.quantidade, jid = excluded.jid
			 WHERE loteria_bilhetes.quantidade + excluded.quantidade <= ?
			 RETURNING quantidade`,
			chat.String(), acc.Key, acc.JID.String(), draw.Unix(), count, lotteryMaxTickets,
		).Scan(&p.held)
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		err = tx.QueryRowContext(ctx,
			"SELECT quantidade FROM loteria_bilhetes WHERE chat = ? AND pessoa = ? AND sorteio = ?",
			chat.String(), acc.Key, draw.Unix(),
		).Scan(&p.held)
		if err != nil {
			return err
		}
		return errTicketLimit
	})
	return p, err
}

func (s *walletStore) lotteryPool(ctx context.Context, chat types.JID, acc core.Account, draw time.Time) (p lotteryPool, err error) {
	err = s.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(quantidade), 0), COUNT(*), COALESCE(SUM(quantidade) FILTER (WHERE pessoa = ?), 0)
		 FROM loteria_bilhetes WHERE chat = ? AND sorteio = ?`,
		acc.Key, chat.String(), draw.Unix(),
	).Scan(&p.tickets, &p.participants, &p.mine)
	return p, err
}

func (s *walletStore) lastLotteryWin(ctx context.Context, chat types.JID) (w lotteryWin, found bool, err error) {
	var winner string
	err = s.db.QueryRowContext(ctx,
		"SELECT jid, premio FROM loteria_sorteios WHERE chat = ? ORDER BY sorteio DESC LIMIT 1",
		chat.String(),
	).Scan(&winner, &w.prize)
	if errors.Is(err, sql.ErrNoRows) {
		return lotteryWin{}, false, nil
	}
	if err != nil {
		return lotteryWin{}, false, err
	}
	w.winner, err = types.ParseJID(winner)
	return w, err == nil, err
}

func (s *walletStore) dueLotteries(ctx context.Context, now time.Time) (due []pendingLottery, err error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT DISTINCT chat, sorteio FROM loteria_bilhetes WHERE sorteio <= ? ORDER BY sorteio",
		now.Unix(),
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var (
			chat string
			at   int64
		)
		if err := rows.Scan(&chat, &at); err != nil {
			return nil, err
		}
		jid, err := types.ParseJID(chat)
		if err != nil {
			return nil, err
		}
		due = append(due, pendingLottery{chat: jid, at: time.Unix(at, 0)})
	}
	return due, rows.Err()
}

func (s *walletStore) drawLottery(ctx context.Context, pending pendingLottery, next time.Time) (d lotteryDraw, err error) {
	chat := pending.chat.String()
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		tickets, err := takeLotteryTickets(ctx, tx, pending)
		if err != nil || len(tickets) == 0 {
			return err
		}
		d.tickets = tickets
		if len(tickets) == 1 {
			_, err = tx.ExecContext(ctx,
				`INSERT INTO loteria_bilhetes (chat, pessoa, jid, sorteio, quantidade) VALUES (?, ?, ?, ?, ?)
				 ON CONFLICT (chat, pessoa, sorteio) DO UPDATE SET quantidade = loteria_bilhetes.quantidade + excluded.quantidade`,
				chat, tickets[0].owner.Key, tickets[0].owner.JID.String(), next.Unix(), tickets[0].count,
			)
			return err
		}

		for _, t := range tickets {
			d.total += t.count
		}
		d.winner, d.prize = drawWinner(tickets, d.total), lotteryPot(d.total)
		if err := openWallet(ctx, tx, pending.chat, d.winner.owner); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", d.prize, chat, d.winner.owner.Key)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx,
			"INSERT INTO loteria_sorteios (chat, sorteio, vencedor, jid, premio, bilhetes, participantes) VALUES (?, ?, ?, ?, ?, ?, ?)",
			chat, pending.at.Unix(), d.winner.owner.Key, d.winner.owner.JID.String(), d.prize, d.winner.count, len(tickets),
		)
		return err
	})
	return d, err
}

func takeLotteryTickets(ctx context.Context, tx *sql.Tx, pending pendingLottery) (tickets []lotteryTicket, err error) {
	rows, err := tx.QueryContext(ctx,
		"DELETE FROM loteria_bilhetes WHERE chat = ? AND sorteio = ? RETURNING pessoa, jid, quantidade",
		pending.chat.String(), pending.at.Unix(),
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var (
			t   lotteryTicket
			jid string
		)
		if err := rows.Scan(&t.owner.Key, &jid, &t.count); err != nil {
			return nil, err
		}
		if t.owner.JID, err = types.ParseJID(jid); err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}
	return tickets, rows.Err()
}
