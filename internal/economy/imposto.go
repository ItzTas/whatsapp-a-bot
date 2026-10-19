package economy

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
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
	acc  core.Account
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

func dueTax(balance int64, level int, haunted bool, cut int64) int64 {
	return min(max(balance, 0), (wealthTax(balance)+iptu(level, haunted))*(100-cut)/100)
}

func nextTaxDay(now time.Time) time.Time {
	local := now.In(core.ReminderZone)
	next := time.Date(local.Year(), local.Month(), local.Day(), taxHour, 0, 0, 0, core.ReminderZone)
	if !local.Before(next) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

func (b *bot) imposto(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := core.FirstMentionOrSender(evt).ToNonAD()
	acc := b.Account(ctx, person)
	var (
		balance int64
		job     Profession
		crew    []hire
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { balance, _, err = b.Wallets.Balance(gctx, chat, acc); return err })
	g.Go(func() (err error) { job, err = b.ProfessionOf(gctx, chat, acc); return err })
	g.Go(func() (err error) { crew, err = b.Wallets.crew(gctx, chat, acc); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler imposto: %w", err)
	}
	cut := strongest(crewKinds(crew), taxCutOf)
	discount := ""
	if cut > 0 {
		discount = fmt.Sprintf(" (já com %d%% a menos do contador)", cut)
	}
	lines := []string{"🦁 *Imposto sobre fortuna*", fmt.Sprintf("_Todo dia às %dh o Leão passa e cobra de quem tem muito. As moedas somem_", taxHour), ""}
	lines = append(lines, fmt.Sprintf("• Até %s: isento", Coins(taxBrackets[0].above)))
	for _, br := range taxBrackets {
		lines = append(lines, fmt.Sprintf("• O que passar de %s: %d%% ao dia", Coins(br.above), br.percent))
	}
	lines = append(lines,
		fmt.Sprintf("🏠 Quem tem imóvel paga o IPTU dele junto (veja no !imovel), e casa assombrada paga %dx", hauntTaxFactor),
		"🧮 Um contador no !empregados corta parte do imposto e do IPTU",
		"",
		fmt.Sprintf("@%s tem %s e paga %s%s na próxima passada do Leão, daqui a %s", person.User, Coins(balance), Coins(dueTax(balance, job.estate, job.hauntedHome(), cut)), discount, core.FormatWait(time.Until(nextTaxDay(time.Now())))),
	)
	_, err := b.SendMentions(ctx, chat, strings.Join(lines, "\n"), []types.JID{person})
	return err
}

func (b *bot) runTaxes(ctx context.Context) {
	ticker := time.NewTicker(core.ReminderTick)
	defer ticker.Stop()
	for {
		if b.Client.IsLoggedIn() {
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
	local := now.In(core.ReminderZone)
	if local.Hour() < taxHour {
		return
	}
	day := local.Format(time.DateOnly)
	chats, err := b.Wallets.untaxedChats(ctx, day)
	if err != nil {
		b.Client.Log.Errorf("procurar conversas pro imposto: %v", err)
		return
	}
	var g errgroup.Group
	g.SetLimit(taxSendLimit)
	for _, chat := range chats {
		g.Go(func() error {
			if err := b.taxChat(ctx, chat, day); err != nil {
				b.Client.Log.Errorf("cobrar imposto em %s: %v", chat, err)
			}
			return nil
		})
	}
	_ = g.Wait()
}

func (b *bot) taxChat(ctx context.Context, chat types.JID, day string) error {
	if len(b.economyGroups(ctx, chat)) == 0 {
		return nil
	}
	r, err := b.Wallets.levyTaxes(ctx, chat, day)
	if err != nil || !r.claimed || r.payers == 0 {
		return err
	}
	text := fmt.Sprintf("🦁 *O Leão passou!*\n\n%d pessoa(s) pagaram %s de imposto sobre fortuna e IPTU hoje, e o dinheiro sumiu nos cofres públicos\n💸 Quem mais pagou foi @%s: %s\n\n_Manda !imposto pra ver as faixas_",
		r.payers, Coins(r.total), r.top.acc.JID.User, Coins(r.top.paid))
	return b.announce(ctx, chat, text, r.top.acc.JID)
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
		if ledgers.linked(chat) {
			continue
		}
		chats = append(chats, chat)
	}
	return chats, rows.Err()
}

func (s *walletStore) levyTaxes(ctx context.Context, chat types.JID, day string) (r taxReport, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
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
			_, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo - ? WHERE chat = ? AND pessoa = ?", p.paid, chat.String(), p.acc.Key)
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

func taxPayers(ctx context.Context, tx *ledgerTx, chat types.JID) (payers []taxPayer, err error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT pessoa, jid, saldo, imovel,
		        EXISTS (SELECT 1 FROM encantos e WHERE e.chat = carteiras.chat AND e.alvo = carteiras.pessoa AND e.tipo = ? AND e.ate > ?)
		        AND NOT EXISTS (SELECT 1 FROM inventarios i WHERE i.chat = carteiras.chat AND i.pessoa = carteiras.pessoa AND i.item = ?),
		        COALESCE((SELECT GROUP_CONCAT(w.tipo) FROM empregados w WHERE w.chat = carteiras.chat AND w.pessoa = carteiras.pessoa), '')
		 FROM carteiras WHERE chat = ? AND saldo > 0`,
		hauntID, time.Now().Unix(), carrancaItem, chat.String(),
	)
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
			haunted bool
			tipos   string
		)
		if err := rows.Scan(&p.acc.Key, &jid, &balance, &level, &haunted, &tipos); err != nil {
			return nil, err
		}
		if p.paid = dueTax(balance, level, haunted, strongest(kindsOf(tipos), taxCutOf)); p.paid == 0 {
			continue
		}
		if p.acc.JID, err = types.ParseJID(jid); err != nil {
			return nil, err
		}
		payers = append(payers, p)
	}
	return payers, rows.Err()
}
