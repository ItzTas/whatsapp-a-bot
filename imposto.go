package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	taxHour      = 12
	taxSendLimit = 4
)

type taxBracket struct {
	above   int64
	percent int64
}

var taxBrackets = []taxBracket{
	{above: 20_000, percent: 1},
	{above: 100_000, percent: 2},
	{above: 500_000, percent: 3},
	{above: 2_000_000, percent: 5},
}

type taxPayer struct {
	acc  account
	paid int64
}

type taxReport struct {
	claimed bool
	payers  int
	total   int64
	top     taxPayer
}

func wealthTax(balance int64) int64 {
	var tax int64
	for i, br := range taxBrackets {
		if balance <= br.above {
			break
		}
		top := balance
		if i+1 < len(taxBrackets) {
			top = min(balance, taxBrackets[i+1].above)
		}
		tax += (top - br.above) * br.percent / 100
	}
	return tax
}

func dueTax(balance int64, level int) int64 {
	e, _ := estateAt(level)
	return min(max(balance, 0), wealthTax(balance)+e.tax)
}

func nextTaxDay(now time.Time) time.Time {
	local := now.In(reminderZone)
	next := time.Date(local.Year(), local.Month(), local.Day(), taxHour, 0, 0, 0, reminderZone)
	if !local.Before(next) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

func (b *bot) imposto(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := firstMentionOrSender(evt).ToNonAD()
	acc := b.account(ctx, person)
	var (
		balance int64
		job     profession
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { balance, _, err = b.wallets.balance(gctx, chat, acc); return err })
	g.Go(func() (err error) { job, err = b.professionOf(gctx, chat, acc); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler imposto: %w", err)
	}
	lines := []string{"🦁 *Imposto sobre fortuna*", fmt.Sprintf("_Todo dia às %dh o Leão passa e cobra de quem tem muito. As moedas somem_", taxHour), ""}
	lines = append(lines, fmt.Sprintf("• Até %s: isento", coins(taxBrackets[0].above)))
	for _, br := range taxBrackets {
		lines = append(lines, fmt.Sprintf("• O que passar de %s: %d%% ao dia", coins(br.above), br.percent))
	}
	lines = append(lines,
		"🏠 Quem tem imóvel paga o IPTU dele junto (veja no !imovel)",
		"",
		fmt.Sprintf("@%s tem %s e paga %s na próxima passada do Leão, daqui a %s", person.User, coins(balance), coins(dueTax(balance, job.estate)), formatWait(time.Until(nextTaxDay(time.Now())))),
	)
	_, err := b.sendMentions(ctx, chat, strings.Join(lines, "\n"), []types.JID{person})
	return err
}

func (b *bot) runTaxes(ctx context.Context) {
	ticker := time.NewTicker(reminderTick)
	defer ticker.Stop()
	for {
		if b.client.IsLoggedIn() {
			b.collectDueTaxes(ctx, time.Now())
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (b *bot) collectDueTaxes(ctx context.Context, now time.Time) {
	local := now.In(reminderZone)
	if local.Hour() < taxHour {
		return
	}
	day := local.Format(time.DateOnly)
	chats, err := b.wallets.untaxedChats(ctx, day)
	if err != nil {
		b.client.Log.Errorf("procurar conversas pro imposto: %v", err)
		return
	}
	var g errgroup.Group
	g.SetLimit(taxSendLimit)
	for _, chat := range chats {
		g.Go(func() error {
			if err := b.taxChat(ctx, chat, day); err != nil {
				b.client.Log.Errorf("cobrar imposto em %s: %v", chat, err)
			}
			return nil
		})
	}
	_ = g.Wait()
}

func (b *bot) taxChat(ctx context.Context, chat types.JID, day string) error {
	if !b.economyOn(ctx, chat) {
		return nil
	}
	r, err := b.wallets.levyTaxes(ctx, chat, day)
	if err != nil || !r.claimed || r.payers == 0 {
		return err
	}
	allowed, err := b.allowedIn(ctx, chat, []types.JID{r.top.acc.jid})
	if err != nil || !allowed {
		return err
	}
	text := fmt.Sprintf("🦁 *O Leão passou!*\n\n%d pessoa(s) pagaram %s de imposto sobre fortuna e IPTU hoje, e o dinheiro sumiu nos cofres públicos\n💸 Quem mais pagou foi @%s: %s\n\n_Manda !imposto pra ver as faixas_",
		r.payers, coins(r.total), r.top.acc.jid.User, coins(r.top.paid))
	_, err = b.sendMentions(ctx, chat, text, []types.JID{r.top.acc.jid})
	return err
}

func (s *walletStore) untaxedChats(ctx context.Context, day string) (chats []types.JID, err error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT chat FROM carteiras
		 WHERE (saldo > ? OR imovel > 0) AND chat NOT IN (SELECT chat FROM impostos WHERE dia = ?)`,
		taxBrackets[0].above, day,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		chat, err := types.ParseJID(raw)
		if err != nil {
			return nil, err
		}
		chats = append(chats, chat)
	}
	return chats, rows.Err()
}

func (s *walletStore) levyTaxes(ctx context.Context, chat types.JID, day string) (r taxReport, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, "INSERT INTO impostos (chat, dia) VALUES (?, ?) ON CONFLICT DO NOTHING", chat.String(), day)
		if err != nil {
			return err
		}
		claimed, err := res.RowsAffected()
		if err != nil || claimed == 0 {
			return err
		}
		r.claimed = true
		payers, err := taxPayers(ctx, tx, chat)
		if err != nil {
			return err
		}
		for _, p := range payers {
			_, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo - ? WHERE chat = ? AND pessoa = ?", p.paid, chat.String(), p.acc.key)
			if err != nil {
				return err
			}
			r.payers++
			r.total += p.paid
			if p.paid > r.top.paid {
				r.top = p
			}
		}
		_, err = tx.ExecContext(ctx, "UPDATE impostos SET total = ? WHERE chat = ? AND dia = ?", r.total, chat.String(), day)
		return err
	})
	return r, err
}

func taxPayers(ctx context.Context, tx *sql.Tx, chat types.JID) (payers []taxPayer, err error) {
	rows, err := tx.QueryContext(ctx, "SELECT pessoa, jid, saldo, imovel FROM carteiras WHERE chat = ? AND saldo > 0", chat.String())
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var (
			p       taxPayer
			jid     string
			balance int64
			level   int
		)
		if err := rows.Scan(&p.acc.key, &jid, &balance, &level); err != nil {
			return nil, err
		}
		if p.paid = dueTax(balance, level); p.paid == 0 {
			continue
		}
		if p.acc.jid, err = types.ParseJID(jid); err != nil {
			return nil, err
		}
		payers = append(payers, p)
	}
	return payers, rows.Err()
}

