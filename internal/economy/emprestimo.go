package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	loanRate        = 3
	loanLateRate    = 10
	loanTermDays    = 3
	loanMaxDays     = 7
	loanMin         = 50
	loanLimitShifts = 8
	loanLimitFloor  = 300
	loanLimitCap    = 20_000
	loanSeizeEvery  = 2 * time.Hour
	loanSendLimit   = 4
)

type loanSeizure struct {
	chat types.JID
	acc  core.Account
}

func (p Profession) loanLimit() int64 {
	return min(max(p.max*loanLimitShifts, loanLimitFloor), loanLimitCap)
}

func (b *bot) emprestimo(ctx context.Context, evt *events.Message) error {
	args := core.PlainArgs(core.MessageText(evt.Message))
	if len(args) == 0 {
		return b.loanStatus(ctx, evt)
	}
	actions := map[string]func(context.Context, *events.Message, []string) error{
		"pegar":  b.takeLoan,
		"pedir":  b.takeLoan,
		"pagar":  b.repayLoan,
		"quitar": b.repayLoan,
	}
	action, ok := actions[args[0]]
	if !ok {
		return b.takeLoan(ctx, evt, args)
	}
	return action(ctx, evt, args[1:])
}

func (b *bot) loanStatus(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	now := time.Now()
	var (
		job  Profession
		owed *debt
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { job, err = b.ProfessionOf(gctx, chat, acc); return err })
	g.Go(func() (err error) { owed, err = b.Wallets.loanOf(gctx, chat, acc, now); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler empréstimo: %w", err)
	}
	if owed == nil {
		text := fmt.Sprintf("🏦 @%s não deve nada pro banco, e ele te empresta até %s agora\n📈 Juros de %d%% ao dia, compostos, e %d%% depois do prazo\n\n_Pra pegar: !emprestimo pegar 500 (prazo de %d dias, ou escolhe de 1 a %d no fim)_",
			person.User, Coins(job.loanLimit()), loanRate, loanLateRate, loanTermDays, loanMaxDays)
		_, err := b.SendMentions(ctx, chat, text, []types.JID{person})
		return err
	}
	text := fmt.Sprintf("🏦 @%s deve %s pro banco, %s, com juros de %d%% ao dia", person.User, Coins(owed.amount), owed.deadlineText(now), owed.rateAt(now))
	if owed.overdue(now) {
		text += fmt.Sprintf("\n🔨 Como o prazo venceu, o banco penhora seu saldo a cada %s até quitar", core.FormatWait(loanSeizeEvery))
	}
	text += "\n\n_Pra pagar: !emprestimo pagar 100 (ou tudo)_"
	_, err := b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) takeLoan(ctx context.Context, evt *events.Message, args []string) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	usageText := fmt.Sprintf("Use assim: !emprestimo pegar 500. O prazo padrão é de %d dias, e dá pra pedir outro (de 1 a %d) no fim: !emprestimo pegar 500 5", loanTermDays, loanMaxDays)
	if len(args) == 0 {
		return b.SendText(ctx, chat, usageText)
	}
	days := int64(loanTermDays)
	if len(args) > 1 {
		n, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil || n < 1 || n > loanMaxDays {
			return b.SendText(ctx, chat, usageText)
		}
		days = n
	}

	now := time.Now()
	var (
		job  Profession
		owed *debt
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { job, err = b.ProfessionOf(gctx, chat, acc); return err })
	g.Go(func() (err error) { owed, err = b.Wallets.loanOf(gctx, chat, acc, now); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler empréstimo: %w", err)
	}
	if owed != nil {
		return b.SendText(ctx, chat, fmt.Sprintf("Você já deve %s pro banco. Quita antes de pegar outro: !emprestimo pagar tudo", Coins(owed.amount)))
	}
	limit := job.loanLimit()
	amount, ok := ParseStake(args[0], limit)
	if !ok {
		return b.SendText(ctx, chat, usageText)
	}
	if amount < loanMin {
		return b.SendText(ctx, chat, fmt.Sprintf("O banco não perde tempo com menos de %s 🙄", Coins(loanMin)))
	}
	if amount > limit {
		return b.SendText(ctx, chat, fmt.Sprintf("O banco só te empresta até %s. O limite cresce com o que sua profissão paga no !trabalhar", Coins(limit)))
	}

	d := debt{chat: chat, debtor: acc, amount: amount, rate: loanRate, lateRate: loanLateRate, since: now, due: now.Add(time.Duration(days) * debtCompoundEvery)}
	err := b.Wallets.borrow(ctx, d)
	if errors.Is(err, errDebtActive) {
		return b.SendText(ctx, chat, "Você já tem um empréstimo aberto no banco. Confere no !emprestimo")
	}
	if err != nil {
		return fmt.Errorf("pegar empréstimo: %w", err)
	}
	text := fmt.Sprintf("🏦 O banco liberou %s pra @%s\n📈 Juros de %d%% ao dia, compostos, e prazo de %s: no vencimento a dívida vai tá em %s\n🔥 Se atrasar, os juros sobem pra %d%% ao dia e o banco penhora seu saldo a cada %s até quitar\n\n_Pra pagar: !emprestimo pagar 100 (ou tudo)_",
		Coins(amount), person.User, loanRate, core.CountLabel(int(days), "dia", "dias"), Coins(d.accrued(d.due).amount), loanLateRate, core.FormatWait(loanSeizeEvery))
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) repayLoan(ctx context.Context, evt *events.Message, args []string) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	now := time.Now()
	var (
		balance int64
		owed    *debt
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { balance, _, err = b.Wallets.Balance(gctx, chat, acc); return err })
	g.Go(func() (err error) { owed, err = b.Wallets.loanOf(gctx, chat, acc, now); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler empréstimo: %w", err)
	}

	text, err := b.repayLoanText(ctx, chat, acc, owed, balance, args, now)
	if err != nil {
		return err
	}
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) repayLoanText(ctx context.Context, chat types.JID, acc core.Account, owed *debt, balance int64, args []string, now time.Time) (string, error) {
	user := acc.JID.User
	if owed == nil {
		return fmt.Sprintf("@%s não deve nada pro banco 😇", user), nil
	}
	if balance == 0 {
		return fmt.Sprintf("@%s tá liso 🪹 e deve %s pro banco", user, Coins(owed.amount)), nil
	}
	arg := "tudo"
	if len(args) > 0 {
		arg = args[0]
	}
	amount, ok := ParseStake(arg, min(balance, owed.amount))
	if !ok {
		return "Diz quanto pagar: !emprestimo pagar 100 (ou metade, tudo)", nil
	}
	if amount > balance {
		return "Você só tem " + Coins(balance), nil
	}

	left, paid, err := b.Wallets.repayLoan(ctx, chat, acc, amount, now)
	if errors.Is(err, errNoDebt) {
		return fmt.Sprintf("@%s não deve nada pro banco 😇", user), nil
	}
	if errors.Is(err, ErrNoFunds) {
		return "Seu saldo mudou no meio do pagamento e não dá mais 🤨 Confere no !saldo", nil
	}
	if err != nil {
		return "", fmt.Errorf("pagar empréstimo: %w", err)
	}
	if left.amount == 0 {
		return fmt.Sprintf("✅ @%s pagou %s pro banco e quitou o empréstimo 🤝", user, Coins(paid)), nil
	}
	text := fmt.Sprintf("💸 @%s pagou %s pro banco. Ainda deve %s", user, Coins(paid), Coins(left.amount))
	if left.overdue(now) {
		return text + ", e como o prazo já venceu, o banco continua penhorando seu saldo", nil
	}
	return text + ", e o prazo acaba em " + core.FormatWait(left.due.Sub(now)), nil
}

func (b *bot) runLoanSeizures(ctx context.Context) {
	ticker := time.NewTicker(core.ReminderTick)
	defer ticker.Stop()
	for {
		if b.Client.IsLoggedIn() {
			b.seizeOverdueLoans(ctx, time.Now())
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (b *bot) seizeOverdueLoans(ctx context.Context, now time.Time) {
	due, err := b.Wallets.dueSeizures(ctx, now)
	if err != nil {
		b.Client.Log.Errorf("procurar empréstimos atrasados: %v", err)
		return
	}
	var g errgroup.Group
	g.SetLimit(loanSendLimit)
	for _, s := range due {
		g.Go(func() error {
			if err := b.seizeLoan(ctx, s, now); err != nil {
				b.Client.Log.Errorf("penhorar empréstimo de %s em %s: %v", s.acc.Key, s.chat, err)
			}
			return nil
		})
	}
	_ = g.Wait()
}

func (b *bot) seizeLoan(ctx context.Context, s loanSeizure, now time.Time) error {
	left, seized, err := b.Wallets.seizeLoan(ctx, s.chat, s.acc, now)
	if err != nil || seized == 0 {
		return err
	}
	text := fmt.Sprintf("🔨 O banco penhorou %s do saldo de @%s pelo empréstimo atrasado e a dívida tá quitada 🤝", Coins(seized), s.acc.JID.User)
	if left.amount > 0 {
		text = fmt.Sprintf("🔨 O banco penhorou %s do saldo de @%s pelo empréstimo atrasado. Ainda falta %s, e a próxima penhora vem em %s",
			Coins(seized), s.acc.JID.User, Coins(left.amount), core.FormatWait(loanSeizeEvery))
	}
	return b.announce(ctx, s.chat, text, s.acc.JID)
}

func (s *walletStore) borrow(ctx context.Context, d debt) error {
	return s.inTx(ctx, func(tx *ledgerTx) error {
		res, err := tx.ExecContext(ctx,
			`INSERT INTO emprestimos (chat, pessoa, jid, divida, juros, juros_atraso, desde, vence_em, penhora_em)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT DO NOTHING`,
			d.chat.String(), d.debtor.Key, d.debtor.JID.String(), d.amount, d.rate, d.lateRate, d.since.Unix(), d.due.Unix(), d.due.Unix(),
		)
		if err != nil {
			return err
		}
		inserted, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if inserted == 0 {
			return errDebtActive
		}
		if err := openWallet(ctx, tx, d.chat, d.debtor); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", d.amount, d.chat.String(), d.debtor.Key)
		return err
	})
}

func (s *walletStore) loanOf(ctx context.Context, chat types.JID, acc core.Account, now time.Time) (*debt, error) {
	row := s.db.QueryRowContext(ctx, loanColumns+" WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key)
	d, err := scanLoan(row, chat)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	d = d.accrued(now)
	return &d, nil
}

func (s *walletStore) repayLoan(ctx context.Context, chat types.JID, acc core.Account, amount int64, now time.Time) (left debt, paid int64, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) (err error) {
		left, paid, err = settleLoan(ctx, tx, chat, acc, amount, now)
		return err
	})
	return left, paid, err
}

func (s *walletStore) dueSeizures(ctx context.Context, now time.Time) (due []loanSeizure, err error) {
	rows, err := s.db.QueryContext(ctx, "SELECT chat, pessoa, jid FROM emprestimos WHERE penhora_em <= ?", now.Unix())
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var (
			chat, jid string
			l         loanSeizure
		)
		if err := rows.Scan(&chat, &l.acc.Key, &jid); err != nil {
			return nil, err
		}
		if l.chat, err = types.ParseJID(chat); err != nil {
			return nil, err
		}
		if l.acc.JID, err = types.ParseJID(jid); err != nil {
			return nil, err
		}
		if ledgers.linked(l.chat) {
			continue
		}
		due = append(due, l)
	}
	return due, rows.Err()
}

func (s *walletStore) seizeLoan(ctx context.Context, chat types.JID, acc core.Account, now time.Time) (left debt, seized int64, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		res, err := tx.ExecContext(ctx,
			"UPDATE emprestimos SET penhora_em = ? WHERE chat = ? AND pessoa = ? AND penhora_em <= ?",
			now.Add(loanSeizeEvery).Unix(), chat.String(), acc.Key, now.Unix(),
		)
		if err != nil {
			return err
		}
		claimed, err := res.RowsAffected()
		if err != nil || claimed == 0 {
			return err
		}
		var balance int64
		err = tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&balance)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil || balance <= 0 {
			return err
		}
		left, seized, err = settleLoan(ctx, tx, chat, acc, balance, now)
		return err
	})
	return left, seized, err
}

func settleLoan(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, amount int64, now time.Time) (debt, int64, error) {
	row := tx.QueryRowContext(ctx, loanColumns+" WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key)
	d, err := scanLoan(row, chat)
	if errors.Is(err, sql.ErrNoRows) {
		return debt{}, 0, errNoDebt
	}
	if err != nil {
		return debt{}, 0, err
	}
	left := d.accrued(now)
	paid := min(amount, left.amount)
	if err := debitBalance(ctx, tx, chat, acc, paid); err != nil {
		return debt{}, 0, err
	}
	left.amount -= paid
	return left, paid, storeLoan(ctx, tx, left)
}

const loanColumns = "SELECT pessoa, jid, divida, juros, juros_atraso, desde, vence_em FROM emprestimos"

func scanLoan(row interface{ Scan(dest ...any) error }, chat types.JID) (debt, error) {
	var (
		d          debt
		jid        string
		since, due int64
	)
	if err := row.Scan(&d.debtor.Key, &jid, &d.amount, &d.rate, &d.lateRate, &since, &due); err != nil {
		return debt{}, err
	}
	parsed, err := types.ParseJID(jid)
	if err != nil {
		return debt{}, err
	}
	d.chat, d.debtor.JID, d.since, d.due = chat, parsed, time.Unix(since, 0), time.Unix(due, 0)
	return d, nil
}

func storeLoan(ctx context.Context, tx *ledgerTx, d debt) error {
	if d.amount <= 0 {
		_, err := tx.ExecContext(ctx, "DELETE FROM emprestimos WHERE chat = ? AND pessoa = ?", d.chat.String(), d.debtor.Key)
		return err
	}
	_, err := tx.ExecContext(ctx,
		"UPDATE emprestimos SET divida = ?, desde = ? WHERE chat = ? AND pessoa = ?",
		d.amount, d.since.Unix(), d.chat.String(), d.debtor.Key,
	)
	return err
}
