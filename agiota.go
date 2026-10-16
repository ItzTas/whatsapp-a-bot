package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	debtRate        = 5
	debtLateRate    = 15
	debtTermDays    = 2
	debtMaxDays     = 7
	debtMin         = 10
	debtKillChance  = 0.5
	debtArrestEvery = time.Hour
	debtRequestKind = "emprestar"

	debtCompoundEvery = 24 * time.Hour
	debtCeiling       = 100_000
)

var (
	errDebtActive = errors.New("dívida já ativa")
	errNoDebt     = errors.New("sem dívida")
)

type debt struct {
	chat     types.JID
	debtor   account
	lender   account
	amount   int64
	rate     int64
	lateRate int64
	since    time.Time
	due      time.Time
}

type debtRequest struct {
	mu       sync.Mutex
	chat     types.JID
	borrower account
	lender   account
	amount   int64
	term     time.Duration
	expires  time.Time
}

type debtRequestState struct {
	Chat     types.JID     `json:"chat"`
	Borrower storedAccount `json:"borrower"`
	Lender   storedAccount `json:"lender"`
	Amount   int64         `json:"amount"`
	Term     time.Duration `json:"term"`
	Expires  time.Time     `json:"expires"`
}

func newDebt(chat types.JID, debtor, lender account, amount int64, term time.Duration, now time.Time) debt {
	return debt{chat: chat, debtor: debtor, lender: lender, amount: amount, rate: debtRate, lateRate: debtLateRate, since: now, due: now.Add(term)}
}

func (d debt) rateAt(t time.Time) int64 {
	if t.After(d.due) {
		return d.lateRate
	}
	return d.rate
}

func (d debt) accrued(now time.Time) debt {
	days := int64(now.Sub(d.since) / debtCompoundEvery)
	for range days {
		d.since = d.since.Add(debtCompoundEvery)
		d.amount = min(d.amount+(d.amount*d.rateAt(d.since)+99)/100, debtCeiling)
	}
	return d
}

func (d debt) overdue(now time.Time) bool {
	return now.After(d.due)
}

func plainArgs(text string) []string {
	fields := strings.Fields(strings.ToLower(text))
	return slices.DeleteFunc(fields[min(1, len(fields)):], func(f string) bool { return strings.HasPrefix(f, "@") })
}

func (b *bot) agiota(ctx context.Context, evt *events.Message) error {
	args := plainArgs(messageText(evt.Message))
	mentions := mentionedJIDs(evt.Message)
	if len(args) > 0 && slices.Contains([]string{"pagar", "quitar"}, args[0]) {
		return b.payDebt(ctx, evt, args[1:], mentions)
	}
	if len(mentions) == 0 {
		return b.debtStatus(ctx, evt)
	}
	return b.requestDebt(ctx, evt, mentions[0].ToNonAD(), args)
}

func (b *bot) requestDebt(ctx context.Context, evt *events.Message, lender types.JID, args []string) error {
	chat := evt.Info.Chat.ToNonAD()
	borrower := evt.Info.Sender.ToNonAD()
	usageText := fmt.Sprintf("Use assim: !agiota @pessoa 500. O prazo padrão é de %d dias, e dá pra pedir outro (de 1 a %d) no fim: !agiota @pessoa 500 5", debtTermDays, debtMaxDays)
	borrowerAcc, lenderAcc := b.account(ctx, borrower), b.account(ctx, lender)
	if borrowerAcc.key == lenderAcc.key {
		return b.sendText(ctx, chat, "Pedir emprestado pra você mesmo? Isso se chama mexer no cofrinho 🤡")
	}
	if len(args) == 0 {
		return b.sendText(ctx, chat, usageText)
	}
	days := int64(debtTermDays)
	if len(args) > 1 {
		n, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil || n < 1 || n > debtMaxDays {
			return b.sendText(ctx, chat, usageText)
		}
		days = n
	}

	now := time.Now()
	var (
		funds    int64
		existing *debt
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { funds, _, err = b.wallets.balance(gctx, chat, lenderAcc); return err })
	g.Go(func() (err error) {
		existing, err = b.wallets.debtBetween(gctx, chat, borrowerAcc, lenderAcc, now)
		return err
	})
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler condições do empréstimo: %w", err)
	}
	amount, ok := parseStake(args[0], funds)
	if !ok {
		return b.sendText(ctx, chat, usageText)
	}
	if existing != nil {
		text := fmt.Sprintf("💸 @%s, você ainda deve %s pra @%s. Paga primeiro com !agiota pagar @%s", borrower.User, coins(existing.amount), lender.User, lender.User)
		_, err := b.sendMentions(ctx, chat, text, []types.JID{borrower, lender})
		return err
	}
	if amount < debtMin {
		return b.sendText(ctx, chat, "Por menos de "+coins(debtMin)+" nem vale a pena pedir 🤏")
	}
	if funds < amount {
		text := fmt.Sprintf("@%s só tem %s, não dá pra emprestar %s 🪹", lender.User, coins(funds), coins(amount))
		_, err := b.sendMentions(ctx, chat, text, []types.JID{lender})
		return err
	}

	term := time.Duration(days) * debtCompoundEvery
	r := &debtRequest{chat: chat, borrower: borrowerAcc, lender: lenderAcc, amount: amount, term: term, expires: now.Add(offerWindow)}
	key := offerKey(chat, lenderAcc)
	if !b.storeDebtRequest(ctx, key, r) {
		text := fmt.Sprintf("@%s já tem um pedido ou proposta esperando resposta. Espera aceitar, recusar ou passar o tempo dele ⏳", lender.User)
		_, err := b.sendMentions(ctx, chat, text, []types.JID{lender})
		return err
	}
	time.AfterFunc(offerWindow, func() { b.expireDebtRequest(ctx, key, r) })

	atDue := newDebt(chat, borrowerAcc, lenderAcc, amount, term, now)
	text := fmt.Sprintf("🙏 @%s quer pegar %s emprestado com @%s\n"+
		"📈 Juros de %d%% ao dia, compostos, e prazo de %s: no vencimento a dívida vai tá em %s\n"+
		"🔥 Se atrasar, os juros sobem pra %d%% ao dia e @%s pode !matar, !prender, !sequestrar ou !processar @%s pra cobrar, sem ser crime\n\n"+
		"@%s, manda !aceitar pra emprestar ou !recusar pra negar. O pedido vale por %s",
		borrower.User, coins(amount), lender.User,
		debtRate, countLabel(int(days), "dia", "dias"), coins(atDue.accrued(atDue.due).amount),
		debtLateRate, lender.User, borrower.User,
		lender.User, formatWait(offerWindow))
	_, err := b.sendMentions(ctx, chat, text, []types.JID{borrower, lender})
	return err
}

func (b *bot) storeDebtRequest(ctx context.Context, key string, r *debtRequest) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, pending := b.itemOffers.Load(key); pending {
		return false
	}
	if _, pending := b.debtRequests.LoadOrStore(key, r); pending {
		return false
	}
	state := debtRequestState{
		Chat: r.chat, Borrower: storeAccount(r.borrower), Lender: storeAccount(r.lender),
		Amount: r.amount, Term: r.term, Expires: r.expires,
	}
	b.keepGame(ctx, debtRequestKind, key, state, r.expires)
	return true
}

func (b *bot) forgetDebtRequest(ctx context.Context, key string, r *debtRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()
	b.dropGame(ctx, debtRequestKind, key)
}

func (b *bot) expireDebtRequest(ctx context.Context, key string, r *debtRequest) {
	if !b.debtRequests.CompareAndDelete(key, r) {
		return
	}
	b.forgetDebtRequest(ctx, key, r)
	text := fmt.Sprintf("⌛ @%s não respondeu e o pedido de @%s (%s emprestado) expirou",
		r.lender.jid.User, r.borrower.jid.User, coins(r.amount))
	if _, err := b.sendMentions(ctx, r.chat, text, []types.JID{r.lender.jid, r.borrower.jid}); err != nil {
		b.client.Log.Errorf("avisar que o pedido de empréstimo expirou: %v", err)
	}
}

func (b *bot) restoreDebtRequest(ctx context.Context, saved savedGame) error {
	var state debtRequestState
	if err := json.Unmarshal(saved.state, &state); err != nil {
		return fmt.Errorf("ler pedido de empréstimo salvo: %w", err)
	}
	r := &debtRequest{
		chat: state.Chat, borrower: state.Borrower.account(), lender: state.Lender.account(),
		amount: state.Amount, term: state.Term, expires: state.Expires,
	}
	if _, pending := b.debtRequests.LoadOrStore(saved.key, r); pending {
		return fmt.Errorf("já tem outro pedido de empréstimo pra %s", saved.key)
	}
	time.AfterFunc(time.Until(r.expires), func() { b.expireDebtRequest(ctx, saved.key, r) })
	return nil
}

func (b *bot) acceptDebtRequest(ctx context.Context, key string, r *debtRequest) error {
	b.forgetDebtRequest(ctx, key, r)
	d := newDebt(r.chat, r.borrower, r.lender, r.amount, r.term, time.Now())
	text, err := b.lendText(ctx, d)
	if err != nil {
		return fmt.Errorf("fechar empréstimo: %w", err)
	}
	_, err = b.sendMentions(ctx, r.chat, text, []types.JID{r.lender.jid, r.borrower.jid})
	return err
}

func (b *bot) lendText(ctx context.Context, d debt) (string, error) {
	borrower, lender := d.debtor.jid.User, d.lender.jid.User
	err := b.wallets.lend(ctx, d)
	cancelled := map[error]string{
		errDebtActive: fmt.Sprintf("❌ @%s ainda deve pra @%s, então o empréstimo foi cancelado. Paga a outra primeiro", borrower, lender),
		errNoFunds:    fmt.Sprintf("❌ @%s não tem mais %s, então o empréstimo foi cancelado", lender, coins(d.amount)),
	}
	for reason, text := range cancelled {
		if errors.Is(err, reason) {
			return text, nil
		}
	}
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("✅ @%s emprestou %s pra @%s\n"+
		"⏳ Vence em %s, com juros de %d%% ao dia: no vencimento a dívida vai tá em %s\n"+
		"🔥 Depois do prazo os juros sobem pra %d%% ao dia e @%s pode !matar, !prender, !sequestrar ou !processar @%s pra cobrar\n\n"+
		"_Pra pagar: !agiota pagar @%s 100 (ou tudo)_",
		lender, coins(d.amount), borrower,
		formatWait(d.due.Sub(d.since)), d.rate, coins(d.accrued(d.due).amount),
		d.lateRate, lender, borrower,
		lender), nil
}

func (b *bot) refuseDebtRequest(ctx context.Context, key string, r *debtRequest) error {
	b.forgetDebtRequest(ctx, key, r)
	text := fmt.Sprintf("🙅 @%s não quis emprestar %s pra @%s", r.lender.jid.User, coins(r.amount), r.borrower.jid.User)
	_, err := b.sendMentions(ctx, r.chat, text, []types.JID{r.lender.jid, r.borrower.jid})
	return err
}

func (b *bot) cancelDebtRequests(ctx context.Context, chat types.JID, borrower account) []types.JID {
	var lenders []types.JID
	b.debtRequests.Range(func(key, value any) bool {
		r := value.(*debtRequest)
		if r.chat == chat && r.borrower.key == borrower.key && b.debtRequests.CompareAndDelete(key, value) {
			b.forgetDebtRequest(ctx, key.(string), r)
			lenders = append(lenders, r.lender.jid)
		}
		return true
	})
	return lenders
}

func (b *bot) debtStatus(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.account(ctx, person)
	now := time.Now()
	owes, lent, err := b.wallets.debtsOf(ctx, chat, acc, now)
	if err != nil {
		return fmt.Errorf("ler dívidas: %w", err)
	}
	if len(owes)+len(lent) == 0 {
		text := fmt.Sprintf("😇 @%s não deve nada pra ninguém do grupo, e ninguém deve nada pra você\n\n"+
			"_Pra pedir: !agiota @pessoa 500 (prazo de %d dias, ou !agiota @pessoa 500 5 pra escolher). Quem empresta tem que !aceitar_", person.User, debtTermDays)
		_, err := b.sendMentions(ctx, chat, text, []types.JID{person})
		return err
	}

	people := []types.JID{person}
	var lines []string
	if len(owes) > 0 {
		lines = append(lines, fmt.Sprintf("💸 @%s deve:", person.User))
	}
	for _, d := range owes {
		people = append(people, d.lender.jid)
		lines = append(lines, fmt.Sprintf("• %s pra @%s, %s", coins(d.amount), d.lender.jid.User, d.deadlineText(now)))
	}
	if len(lent) > 0 {
		lines = append(lines, "", "🤝 Te devem:")
	}
	for _, d := range lent {
		people = append(people, d.debtor.jid)
		line := fmt.Sprintf("• @%s, %s, %s", d.debtor.jid.User, coins(d.amount), d.deadlineText(now))
		if d.overdue(now) {
			line += ". Pode !matar, !prender, !sequestrar ou !processar pra cobrar"
		}
		lines = append(lines, line)
	}
	lines = append(lines, "", fmt.Sprintf("_Juros de %d%% ao dia dentro do prazo e %d%% depois. Pra pagar: !agiota pagar @pessoa 100 (ou tudo)_", debtRate, debtLateRate))
	_, err = b.sendMentions(ctx, chat, strings.TrimPrefix(strings.Join(lines, "\n"), "\n"), people)
	return err
}

func (d debt) deadlineText(now time.Time) string {
	if d.overdue(now) {
		return fmt.Sprintf("⚠️ atrasado há %s", formatWait(now.Sub(d.due)))
	}
	return "vence em " + formatWait(d.due.Sub(now))
}

func (b *bot) payDebt(ctx context.Context, evt *events.Message, args []string, mentions []types.JID) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.account(ctx, person)
	now := time.Now()

	lender, problem, err := b.creditorOf(ctx, chat, acc, mentions, now)
	if err != nil {
		return err
	}
	if problem != "" {
		return b.sendText(ctx, chat, problem)
	}
	var (
		balance int64
		owed    *debt
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { balance, _, err = b.wallets.balance(gctx, chat, acc); return err })
	g.Go(func() (err error) { owed, err = b.wallets.debtBetween(gctx, chat, acc, lender, now); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler dívida: %w", err)
	}

	text, err := b.repayText(ctx, chat, acc, lender, owed, balance, args, now)
	if err != nil {
		return err
	}
	_, err = b.sendMentions(ctx, chat, text, []types.JID{person, lender.jid})
	return err
}

func (b *bot) creditorOf(ctx context.Context, chat types.JID, acc account, mentions []types.JID, now time.Time) (account, string, error) {
	if len(mentions) > 0 {
		return b.account(ctx, mentions[0].ToNonAD()), "", nil
	}
	owes, _, err := b.wallets.debtsOf(ctx, chat, acc, now)
	if err != nil {
		return account{}, "", fmt.Errorf("ler dívidas: %w", err)
	}
	if len(owes) == 0 {
		return account{}, "Você não deve nada pra ninguém do grupo 😇", nil
	}
	if len(owes) > 1 {
		return account{}, "Você deve pra mais de uma pessoa, marca pra quem vai pagar: !agiota pagar @pessoa 100", nil
	}
	return owes[0].lender, "", nil
}

func (b *bot) repayText(ctx context.Context, chat types.JID, acc, lender account, owed *debt, balance int64, args []string, now time.Time) (string, error) {
	user, lenderUser := acc.jid.User, lender.jid.User
	if owed == nil {
		return fmt.Sprintf("@%s não deve nada pra @%s 😇", user, lenderUser), nil
	}
	if balance == 0 {
		return fmt.Sprintf("@%s tá liso 🪹 e deve %s pra @%s", user, coins(owed.amount), lenderUser), nil
	}
	arg := "tudo"
	if len(args) > 0 {
		arg = args[0]
	}
	amount, ok := parseStake(arg, min(balance, owed.amount))
	if !ok {
		return "Diz quanto pagar: !agiota pagar @pessoa 100 (ou metade, tudo)", nil
	}
	if amount > balance {
		return "Você só tem " + coins(balance), nil
	}

	left, paid, err := b.wallets.repayDebt(ctx, chat, acc, lender, amount, now)
	if errors.Is(err, errNoDebt) {
		return fmt.Sprintf("@%s não deve nada pra @%s 😇", user, lenderUser), nil
	}
	if errors.Is(err, errNoFunds) {
		return "Seu saldo mudou no meio do pagamento e não dá mais 🤨 Confere no !saldo", nil
	}
	if err != nil {
		return "", fmt.Errorf("pagar dívida: %w", err)
	}
	if left.amount == 0 {
		return fmt.Sprintf("✅ @%s pagou %s pra @%s e quitou a dívida 🤝", user, coins(paid), lenderUser), nil
	}
	text := fmt.Sprintf("💸 @%s pagou %s pra @%s. Ainda deve %s", user, coins(paid), lenderUser, coins(left.amount))
	if left.overdue(now) {
		return text + fmt.Sprintf(", e como o prazo já venceu, @%s ainda pode cobrar do jeito ruim", lenderUser), nil
	}
	return text + ", e o prazo acaba em " + formatWait(left.due.Sub(now)), nil
}

func (b *bot) rivalsWithDebt(ctx context.Context, chat types.JID, attacker, target account) (attackerJob, targetJob profession, owed *debt, err error) {
	now := time.Now()
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) {
		attackerJob, targetJob, err = b.rivalProfessions(gctx, chat, attacker, target)
		return err
	})
	g.Go(func() (err error) { owed, err = b.wallets.debtBetween(gctx, chat, target, attacker, now); return err })
	if err = g.Wait(); err != nil {
		return attackerJob, targetJob, nil, err
	}
	if owed != nil && !owed.overdue(now) {
		owed = nil
	}
	return attackerJob, targetJob, owed, nil
}

func debtCollectionText(owed *debt) string {
	return fmt.Sprintf("\n💸 Foi cobrança: @%s deve %s pra @%s e o prazo venceu há %s, então não conta como crime",
		owed.debtor.jid.User, coins(owed.amount), owed.lender.jid.User, formatWait(time.Since(owed.due)))
}

func (s *walletStore) lend(ctx context.Context, d debt) error {
	chat := d.chat
	return s.inTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx,
			`INSERT INTO dividas (chat, devedor, devedor_jid, credor, credor_jid, divida, juros, juros_atraso, desde, vence_em)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT DO NOTHING`,
			chat.String(), d.debtor.key, d.debtor.jid.String(), d.lender.key, d.lender.jid.String(),
			d.amount, d.rate, d.lateRate, d.since.Unix(), d.due.Unix(),
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
		for _, acc := range []account{d.debtor, d.lender} {
			if err := openWallet(ctx, tx, chat, acc); err != nil {
				return err
			}
		}
		return moveCoins(ctx, tx, chat, d.lender, d.debtor, d.amount)
	})
}

func (s *walletStore) debtBetween(ctx context.Context, chat types.JID, debtor, lender account, now time.Time) (*debt, error) {
	row := s.db.QueryRowContext(ctx, debtColumns+" WHERE chat = ? AND devedor = ? AND credor = ?", chat.String(), debtor.key, lender.key)
	d, err := scanDebt(row, chat)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	d = d.accrued(now)
	return &d, nil
}

func (s *walletStore) debtsOf(ctx context.Context, chat types.JID, acc account, now time.Time) (owes, lent []debt, err error) {
	rows, err := s.db.QueryContext(ctx, debtColumns+" WHERE chat = ? AND (devedor = ? OR credor = ?) ORDER BY vence_em", chat.String(), acc.key, acc.key)
	if err != nil {
		return nil, nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		d, err := scanDebt(rows, chat)
		if err != nil {
			return nil, nil, err
		}
		d = d.accrued(now)
		if d.debtor.key == acc.key {
			owes = append(owes, d)
			continue
		}
		lent = append(lent, d)
	}
	return owes, lent, rows.Err()
}

func (s *walletStore) repayDebt(ctx context.Context, chat types.JID, debtor, lender account, amount int64, now time.Time) (left debt, paid int64, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		row := tx.QueryRowContext(ctx, debtColumns+" WHERE chat = ? AND devedor = ? AND credor = ?", chat.String(), debtor.key, lender.key)
		d, err := scanDebt(row, chat)
		if errors.Is(err, sql.ErrNoRows) {
			return errNoDebt
		}
		if err != nil {
			return err
		}
		left = d.accrued(now)
		paid = min(amount, left.amount)
		if err := openWallet(ctx, tx, chat, lender); err != nil {
			return err
		}
		if err := moveCoins(ctx, tx, chat, debtor, lender, paid); err != nil {
			return err
		}
		left.amount -= paid
		return storeDebt(ctx, tx, left)
	})
	return left, paid, err
}

const debtColumns = "SELECT devedor, devedor_jid, credor, credor_jid, divida, juros, juros_atraso, desde, vence_em FROM dividas"

func scanDebt(row interface{ Scan(dest ...any) error }, chat types.JID) (debt, error) {
	var (
		d                    debt
		debtorJID, lenderJID string
		since, due           int64
	)
	err := row.Scan(&d.debtor.key, &debtorJID, &d.lender.key, &lenderJID, &d.amount, &d.rate, &d.lateRate, &since, &due)
	if err != nil {
		return debt{}, err
	}
	if d.debtor.jid, err = types.ParseJID(debtorJID); err != nil {
		return debt{}, err
	}
	if d.lender.jid, err = types.ParseJID(lenderJID); err != nil {
		return debt{}, err
	}
	d.chat, d.since, d.due = chat, time.Unix(since, 0), time.Unix(due, 0)
	return d, nil
}

func storeDebt(ctx context.Context, tx *sql.Tx, d debt) error {
	if d.amount <= 0 {
		_, err := tx.ExecContext(ctx, "DELETE FROM dividas WHERE chat = ? AND devedor = ? AND credor = ?", d.chat.String(), d.debtor.key, d.lender.key)
		return err
	}
	_, err := tx.ExecContext(ctx,
		"UPDATE dividas SET divida = ?, desde = ? WHERE chat = ? AND devedor = ? AND credor = ?",
		d.amount, d.since.Unix(), d.chat.String(), d.debtor.key, d.lender.key,
	)
	return err
}
