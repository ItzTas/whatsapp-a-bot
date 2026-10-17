package economy

import (
	"context"
	"database/sql"
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
	benefactorCount  = 5
	monumentMaxPower = 40
)

type publicWork struct {
	emoji string
	name  string
	goal  int64
	honor string
}

var publicWorks = []publicWork{
	{emoji: "🪑", name: "Banco da praça", goal: 5_000, honor: "ganhou uma plaquinha: \"doado por @%s\""},
	{emoji: "🚏", name: "Ponto de ônibus com cobertura", goal: 15_000, honor: "foi batizado de Parada @%s"},
	{emoji: "⛲", name: "Chafariz", goal: 40_000, honor: "tem a cara de @%s esculpida no meio, cuspindo água"},
	{emoji: "🗿", name: "Estátua", goal: 100_000, honor: "é uma estátua de @%s em tamanho real, fazendo pose"},
	{emoji: "🏟️", name: "Estádio", goal: 300_000, honor: "foi batizado de Arena @%s"},
	{emoji: "🏥", name: "Hospital", goal: 800_000, honor: "tem uma ala inteira com o nome de @%s"},
	{emoji: "🚀", name: "Programa espacial", goal: 2_000_000, honor: "mandou um foguete pro espaço com @%s pintado do lado"},
	{emoji: "🌉", name: "Ponte Rio-Niterói 2", goal: 5_000_000, honor: "agora se chama Ponte @%s"},
}

type finishedWork struct {
	level   int
	honoree types.JID
}

type benefactor struct {
	jid   types.JID
	total int64
}

type worksProgress struct {
	level  int
	raised int64
}

func workAt(level int) publicWork {
	if level < len(publicWorks) {
		return publicWorks[level]
	}
	extra := level - len(publicWorks) + 1
	last := publicWorks[len(publicWorks)-1]
	return publicWork{
		emoji: "🗽",
		name:  fmt.Sprintf("Monumento nº %d", extra),
		goal:  last.goal << min(extra, monumentMaxPower),
		honor: "tem o nome de @%s gravado em ouro na base",
	}
}

func (w publicWork) label() string {
	return w.emoji + " *" + w.name + "*"
}

func (w publicWork) honorText(honoree types.JID) string {
	return fmt.Sprintf("%s %s", w.label(), fmt.Sprintf(w.honor, honoree.User))
}

func (b *bot) doar(ctx context.Context, evt *events.Message) error {
	if len(core.MentionedJIDs(evt.Message)) > 0 {
		return b.dar(ctx, evt)
	}
	arg := core.FirstPlainArg(core.MessageText(evt.Message))
	if arg == "" {
		return b.worksStatus(ctx, evt.Info.Chat.ToNonAD())
	}
	return b.donate(ctx, evt, arg)
}

func (b *bot) donate(ctx context.Context, evt *events.Message, arg string) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	balance, _, err := b.Wallets.Balance(ctx, chat, acc)
	if err != nil {
		return fmt.Errorf("ler saldo: %w", err)
	}
	amount, ok := ParseStake(arg, balance)
	if !ok {
		return b.SendText(ctx, chat, "Não entendi o valor. Use assim: !doar 1000 (ou metade, 10%, tudo)")
	}
	finished, progress, err := b.Wallets.donate(ctx, chat, acc, amount, time.Now())
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, chat, "Você só tem "+Coins(balance))
	}
	if err != nil {
		return fmt.Errorf("doar pra obra: %w", err)
	}
	current := workAt(progress.level)
	lines := []string{fmt.Sprintf("🏗️ @%s doou %s pras obras do grupo! Que alma generosa", person.User, Coins(amount))}
	people := []types.JID{person}
	for _, f := range finished {
		lines = append(lines, "", fmt.Sprintf("🎉 *Obra concluída!* %s", workAt(f.level).honorText(f.honoree)))
		people = append(people, f.honoree)
	}
	lines = append(lines, "", fmt.Sprintf("Agora em construção: %s\n%s", current.label(), progressLine(progress.raised, current.goal)))
	_, err = b.SendMentions(ctx, chat, strings.Join(lines, "\n"), people)
	return err
}

func progressLine(raised, goal int64) string {
	pct := int(raised * 100 / goal)
	return fmt.Sprintf("[%s] %d%%: %s de %s", core.MeterBar(pct), pct, Coins(raised), Coins(goal))
}

func (b *bot) worksStatus(ctx context.Context, chat types.JID) error {
	var (
		progress worksProgress
		top      []benefactor
		finished []finishedWork
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { progress, err = b.Wallets.worksProgress(gctx, chat); return err })
	g.Go(func() (err error) { top, err = b.Wallets.benefactors(gctx, chat, benefactorCount); return err })
	g.Go(func() (err error) { finished, err = b.Wallets.finishedWorks(gctx, chat); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler obras: %w", err)
	}
	current := workAt(progress.level)
	lines := []string{"🏗️ *Obras do grupo*", "", "Em construção: " + current.label(), progressLine(progress.raised, current.goal)}
	var people []types.JID
	if len(top) > 0 {
		lines = append(lines, "", "🏆 *Benfeitores*")
		medals := []string{"🥇", "🥈", "🥉"}
		for i, bf := range top {
			place := fmt.Sprintf("%dº", i+1)
			if i < len(medals) {
				place = medals[i]
			}
			lines = append(lines, fmt.Sprintf("%s @%s: %s", place, bf.jid.User, Coins(bf.total)))
			people = append(people, bf.jid)
		}
	}
	if len(finished) > 0 {
		lines = append(lines, "", "🏛️ *Obras prontas*")
		for _, f := range finished {
			lines = append(lines, workAt(f.level).honorText(f.honoree))
			people = append(people, f.honoree)
		}
	}
	lines = append(lines, "", "_Manda !doar <valor> pra ajudar. O dinheiro vai todo pra obra e não volta, e quem mais doar pra cada obra é homenageado nela_")
	_, err := b.SendMentions(ctx, chat, strings.Join(lines, "\n"), people)
	return err
}

func (s *walletStore) worksProgress(ctx context.Context, chat types.JID) (p worksProgress, err error) {
	err = s.db.QueryRowContext(ctx, "SELECT nivel, arrecadado FROM obras WHERE chat = ?", chat.String()).Scan(&p.level, &p.raised)
	if errors.Is(err, sql.ErrNoRows) {
		return worksProgress{}, nil
	}
	return p, err
}

func (s *walletStore) donate(ctx context.Context, chat types.JID, acc core.Account, amount int64, now time.Time) (finished []finishedWork, p worksProgress, err error) {
	err = s.spend(ctx, chat, acc, amount, func(tx *ledgerTx) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO obras (chat) VALUES (?) ON CONFLICT DO NOTHING", chat.String())
		if err != nil {
			return err
		}
		if err := tx.QueryRowContext(ctx, "SELECT nivel, arrecadado FROM obras WHERE chat = ?", chat.String()).Scan(&p.level, &p.raised); err != nil {
			return err
		}
		for left := amount; left > 0; {
			w := workAt(p.level)
			put := min(left, w.goal-p.raised)
			if err := recordDonation(ctx, tx, chat, acc, p.level, put); err != nil {
				return err
			}
			left -= put
			p.raised += put
			if p.raised < w.goal {
				break
			}
			f, err := finishWork(ctx, tx, chat, p.level, now)
			if err != nil {
				return err
			}
			finished = append(finished, f)
			p.level, p.raised = p.level+1, 0
		}
		_, err = tx.ExecContext(ctx, "UPDATE obras SET nivel = ?, arrecadado = ? WHERE chat = ?", p.level, p.raised, chat.String())
		return err
	})
	return finished, p, err
}

func recordDonation(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, level int, amount int64) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO obras_doacoes (chat, nivel, pessoa, jid, valor) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT (chat, nivel, pessoa) DO UPDATE SET valor = valor + excluded.valor, jid = excluded.jid`,
		chat.String(), level, acc.Key, acc.JID.String(), amount,
	)
	return err
}

func finishWork(ctx context.Context, tx *ledgerTx, chat types.JID, level int, now time.Time) (finishedWork, error) {
	var raw string
	err := tx.QueryRowContext(ctx,
		"SELECT jid FROM obras_doacoes WHERE chat = ? AND nivel = ? ORDER BY valor DESC LIMIT 1",
		chat.String(), level,
	).Scan(&raw)
	if err != nil {
		return finishedWork{}, err
	}
	honoree, err := types.ParseJID(raw)
	if err != nil {
		return finishedWork{}, err
	}
	_, err = tx.ExecContext(ctx,
		"INSERT INTO obras_concluidas (chat, nivel, homenageado, em) VALUES (?, ?, ?, ?)",
		chat.String(), level, raw, now.Unix(),
	)
	return finishedWork{level: level, honoree: honoree}, err
}

func (s *walletStore) benefactors(ctx context.Context, chat types.JID, limit int) (top []benefactor, err error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT MAX(jid), SUM(valor) AS total FROM obras_doacoes WHERE chat = ? GROUP BY pessoa ORDER BY total DESC LIMIT ?",
		chat.String(), limit,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var (
			raw string
			bf  benefactor
		)
		if err := rows.Scan(&raw, &bf.total); err != nil {
			return nil, err
		}
		if bf.jid, err = types.ParseJID(raw); err != nil {
			return nil, err
		}
		top = append(top, bf)
	}
	return top, rows.Err()
}

func (s *walletStore) finishedWorks(ctx context.Context, chat types.JID) (works []finishedWork, err error) {
	rows, err := s.db.QueryContext(ctx, "SELECT nivel, homenageado FROM obras_concluidas WHERE chat = ? ORDER BY nivel", chat.String())
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var (
			raw string
			f   finishedWork
		)
		if err := rows.Scan(&f.level, &raw); err != nil {
			return nil, err
		}
		if f.honoree, err = types.ParseJID(raw); err != nil {
			return nil, err
		}
		works = append(works, f)
	}
	return works, rows.Err()
}
