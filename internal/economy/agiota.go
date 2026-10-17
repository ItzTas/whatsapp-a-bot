package economy

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

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
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
	debtor   core.Account
	lender   core.Account
	amount   int64
	rate     int64
	lateRate int64
	since    time.Time
	due      time.Time
}

type debtRequest struct {
	mu       sync.Mutex
	chat     types.JID
	borrower core.Account
	lender   core.Account
	amount   int64
	term     time.Duration
	rate     int64
	lateRate int64
	expires  time.Time
}

type debtRequestState struct {
	Chat     types.JID          `json:"chat"`
	Borrower core.StoredAccount `json:"borrower"`
	Lender   core.StoredAccount `json:"lender"`
	Amount   int64              `json:"amount"`
	Term     time.Duration      `json:"term"`
	Rate     int64              `json:"rate"`
	LateRate int64              `json:"late_rate"`
	Expires  time.Time          `json:"expires"`
}

type debtLien struct {
	lender  core.Account
	percent int64
}

type debtGarnish struct {
	lender core.Account
	paid   int64
	left   int64
}

func (r *debtRequest) debt(now time.Time) debt {
	return debt{chat: r.chat, debtor: r.borrower, lender: r.lender, amount: r.amount, rate: r.rate, lateRate: r.lateRate, since: now, due: now.Add(r.term)}
}

func (p Profession) loanRates() (rate, lateRate int64) {
	return debtRate + p.lendRate, debtLateRate + p.lendLateRate
}

func (p Profession) loanRatesPerk() string {
	rate, lateRate := p.loanRates()
	return fmt.Sprintf("🦈 Empresta no !agiota com juros de %d%% ao dia no prazo e %d%% no atraso (o normal é %d%% e %d%%)", rate, lateRate, debtRate, debtLateRate)
}

func (p Profession) collectionEvery() time.Duration {
	if p.collectEvery == 0 {
		return debtArrestEvery
	}
	return p.collectEvery
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

func (b *bot) agiota(ctx context.Context, evt *events.Message) error {
	args := core.PlainArgs(core.MessageText(evt.Message))
	mentions := core.MentionedJIDs(evt.Message)
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
	borrowerAcc, lenderAcc := b.Account(ctx, borrower), b.Account(ctx, lender)
	if borrowerAcc.Key == lenderAcc.Key {
		return b.SendText(ctx, chat, "Pedir emprestado pra você mesmo? Isso se chama mexer no cofrinho 🤡")
	}
	if len(args) == 0 {
		return b.SendText(ctx, chat, usageText)
	}
	days := int64(debtTermDays)
	if len(args) > 1 {
		n, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil || n < 1 || n > debtMaxDays {
			return b.SendText(ctx, chat, usageText)
		}
		days = n
	}

	now := time.Now()
	var (
		funds     int64
		existing  *debt
		lenderJob Profession
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { funds, _, err = b.Wallets.Balance(gctx, chat, lenderAcc); return err })
	g.Go(func() (err error) { lenderJob, err = b.ProfessionOf(gctx, chat, lenderAcc); return err })
	g.Go(func() (err error) {
		existing, err = b.Wallets.debtBetween(gctx, chat, borrowerAcc, lenderAcc, now)
		return err
	})
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler condições do empréstimo: %w", err)
	}
	amount, ok := ParseStake(args[0], funds)
	if !ok {
		return b.SendText(ctx, chat, usageText)
	}
	if existing != nil {
		text := fmt.Sprintf("💸 @%s, você ainda deve %s pra @%s. Paga primeiro com !agiota pagar @%s", borrower.User, Coins(existing.amount), lender.User, lender.User)
		_, err := b.SendMentions(ctx, chat, text, []types.JID{borrower, lender})
		return err
	}
	if amount < debtMin {
		return b.SendText(ctx, chat, "Por menos de "+Coins(debtMin)+" nem vale a pena pedir 🤏")
	}
	if funds < amount {
		text := fmt.Sprintf("@%s só tem %s, não dá pra emprestar %s 🪹", lender.User, Coins(funds), Coins(amount))
		_, err := b.SendMentions(ctx, chat, text, []types.JID{lender})
		return err
	}

	term := time.Duration(days) * debtCompoundEvery
	rate, lateRate := lenderJob.loanRates()
	r := &debtRequest{chat: chat, borrower: borrowerAcc, lender: lenderAcc, amount: amount, term: term, rate: rate, lateRate: lateRate, expires: now.Add(offerWindow)}
	key := OfferKey(chat, lenderAcc)
	if !b.storeDebtRequest(ctx, key, r) {
		text := fmt.Sprintf("@%s já tem um pedido ou proposta esperando resposta. Espera aceitar, recusar ou passar o tempo dele ⏳", lender.User)
		_, err := b.SendMentions(ctx, chat, text, []types.JID{lender})
		return err
	}
	time.AfterFunc(offerWindow, func() { b.expireDebtRequest(ctx, key, r) })

	atDue := r.debt(now)
	text := fmt.Sprintf("🙏 @%s quer pegar %s emprestado com @%s\n"+
		"📈 Juros de %d%% ao dia, compostos, e prazo de %s: no vencimento a dívida vai tá em %s\n"+
		"🔥 Se atrasar, os juros sobem pra %d%% ao dia e @%s pode !matar, !prender, !sequestrar ou !processar @%s pra cobrar, sem ser crime%s\n\n"+
		"@%s, manda !aceitar pra emprestar ou !recusar pra negar. O pedido vale por %s",
		borrower.User, Coins(amount), lender.User,
		rate, core.CountLabel(int(days), "dia", "dias"), Coins(atDue.accrued(atDue.due).amount),
		lateRate, lender.User, borrower.User, garnishWarning(lenderJob),
		lender.User, core.FormatWait(offerWindow))
	_, err := b.SendMentions(ctx, chat, text, []types.JID{borrower, lender})
	return err
}

func garnishWarning(lenderJob Profession) string {
	if lenderJob.garnish == 0 {
		return ""
	}
	return fmt.Sprintf(". E como é %s de profissão, %d%% de todo !trabalhar vai direto pra dívida até quitar", lenderJob.name, lenderJob.garnish)
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
		Chat: r.chat, Borrower: core.StoreAccount(r.borrower), Lender: core.StoreAccount(r.lender),
		Amount: r.amount, Term: r.term, Rate: r.rate, LateRate: r.lateRate, Expires: r.expires,
	}
	b.KeepGame(ctx, debtRequestKind, key, state, r.expires)
	return true
}

func (b *bot) forgetDebtRequest(ctx context.Context, key string, r *debtRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()
	b.DropGame(ctx, debtRequestKind, key)
}

func (b *bot) expireDebtRequest(ctx context.Context, key string, r *debtRequest) {
	if !b.debtRequests.CompareAndDelete(key, r) {
		return
	}
	b.forgetDebtRequest(ctx, key, r)
	text := fmt.Sprintf("⌛ @%s não respondeu e o pedido de @%s (%s emprestado) expirou",
		r.lender.JID.User, r.borrower.JID.User, Coins(r.amount))
	if _, err := b.SendMentions(ctx, r.chat, text, []types.JID{r.lender.JID, r.borrower.JID}); err != nil {
		b.Client.Log.Errorf("avisar que o pedido de empréstimo expirou: %v", err)
	}
}

func (b *bot) restoreDebtRequest(ctx context.Context, saved core.SavedGame) error {
	var state debtRequestState
	if err := json.Unmarshal(saved.State, &state); err != nil {
		return fmt.Errorf("ler pedido de empréstimo salvo: %w", err)
	}
	r := &debtRequest{
		chat: state.Chat, borrower: state.Borrower.Account(), lender: state.Lender.Account(),
		amount: state.Amount, term: state.Term, rate: state.Rate, lateRate: state.LateRate, expires: state.Expires,
	}
	if r.rate == 0 {
		r.rate, r.lateRate = debtRate, debtLateRate
	}
	if _, pending := b.debtRequests.LoadOrStore(saved.Key, r); pending {
		return fmt.Errorf("já tem outro pedido de empréstimo pra %s", saved.Key)
	}
	time.AfterFunc(time.Until(r.expires), func() { b.expireDebtRequest(ctx, saved.Key, r) })
	return nil
}

func (b *bot) acceptDebtRequest(ctx context.Context, key string, r *debtRequest) error {
	b.forgetDebtRequest(ctx, key, r)
	d := r.debt(time.Now())
	text, err := b.lendText(ctx, d)
	if err != nil {
		return fmt.Errorf("fechar empréstimo: %w", err)
	}
	_, err = b.SendMentions(ctx, r.chat, text, []types.JID{r.lender.JID, r.borrower.JID})
	return err
}

func (b *bot) lendText(ctx context.Context, d debt) (string, error) {
	borrower, lender := d.debtor.JID.User, d.lender.JID.User
	err := b.Wallets.lend(ctx, d)
	cancelled := map[error]string{
		errDebtActive: fmt.Sprintf("❌ @%s ainda deve pra @%s, então o empréstimo foi cancelado. Paga a outra primeiro", borrower, lender),
		ErrNoFunds:    fmt.Sprintf("❌ @%s não tem mais %s, então o empréstimo foi cancelado", lender, Coins(d.amount)),
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
		lender, Coins(d.amount), borrower,
		core.FormatWait(d.due.Sub(d.since)), d.rate, Coins(d.accrued(d.due).amount),
		d.lateRate, lender, borrower,
		lender), nil
}

func (b *bot) refuseDebtRequest(ctx context.Context, key string, r *debtRequest) error {
	b.forgetDebtRequest(ctx, key, r)
	text := fmt.Sprintf("🙅 @%s não quis emprestar %s pra @%s", r.lender.JID.User, Coins(r.amount), r.borrower.JID.User)
	_, err := b.SendMentions(ctx, r.chat, text, []types.JID{r.lender.JID, r.borrower.JID})
	return err
}

func (b *bot) cancelDebtRequests(ctx context.Context, chat types.JID, borrower core.Account) []types.JID {
	var lenders []types.JID
	b.debtRequests.Range(func(key, value any) bool {
		r := value.(*debtRequest)
		if r.chat == chat && r.borrower.Key == borrower.Key && b.debtRequests.CompareAndDelete(key, value) {
			b.forgetDebtRequest(ctx, key.(string), r)
			lenders = append(lenders, r.lender.JID)
		}
		return true
	})
	return lenders
}

func (b *bot) debtStatus(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	now := time.Now()
	owes, lent, err := b.Wallets.debtsOf(ctx, chat, acc, now)
	if err != nil {
		return fmt.Errorf("ler dívidas: %w", err)
	}
	if len(owes)+len(lent) == 0 {
		text := fmt.Sprintf("😇 @%s não deve nada pra ninguém do grupo, e ninguém deve nada pra você\n\n"+
			"_Pra pedir: !agiota @pessoa 500 (prazo de %d dias, ou !agiota @pessoa 500 5 pra escolher). Quem empresta tem que !aceitar_", person.User, debtTermDays)
		_, err := b.SendMentions(ctx, chat, text, []types.JID{person})
		return err
	}

	people := []types.JID{person}
	var lines []string
	if len(owes) > 0 {
		lines = append(lines, fmt.Sprintf("💸 @%s deve:", person.User))
	}
	for _, d := range owes {
		people = append(people, d.lender.JID)
		lines = append(lines, fmt.Sprintf("• %s pra @%s, %s, %s", Coins(d.amount), d.lender.JID.User, d.deadlineText(now), d.rateText(now)))
	}
	if len(lent) > 0 {
		lines = append(lines, "", "🤝 Te devem:")
	}
	for _, d := range lent {
		people = append(people, d.debtor.JID)
		line := fmt.Sprintf("• @%s, %s, %s, %s", d.debtor.JID.User, Coins(d.amount), d.deadlineText(now), d.rateText(now))
		if d.overdue(now) {
			line += ". Pode !matar, !prender, !sequestrar ou !processar pra cobrar"
		}
		lines = append(lines, line)
	}
	lines = append(lines, "", "_Pra pagar: !agiota pagar @pessoa 100 (ou tudo)_")
	_, err = b.SendMentions(ctx, chat, strings.TrimPrefix(strings.Join(lines, "\n"), "\n"), people)
	return err
}

func (d debt) deadlineText(now time.Time) string {
	if d.overdue(now) {
		return fmt.Sprintf("⚠️ atrasado há %s", core.FormatWait(now.Sub(d.due)))
	}
	return "vence em " + core.FormatWait(d.due.Sub(now))
}

func (d debt) rateText(now time.Time) string {
	return fmt.Sprintf("juros de %d%% ao dia", d.rateAt(now))
}

func (b *bot) payDebt(ctx context.Context, evt *events.Message, args []string, mentions []types.JID) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	now := time.Now()

	lender, problem, err := b.creditorOf(ctx, chat, acc, mentions, now)
	if err != nil {
		return err
	}
	if problem != "" {
		return b.SendText(ctx, chat, problem)
	}
	var (
		balance int64
		owed    *debt
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { balance, _, err = b.Wallets.Balance(gctx, chat, acc); return err })
	g.Go(func() (err error) { owed, err = b.Wallets.debtBetween(gctx, chat, acc, lender, now); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler dívida: %w", err)
	}

	text, err := b.repayText(ctx, chat, acc, lender, owed, balance, args, now)
	if err != nil {
		return err
	}
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person, lender.JID})
	return err
}

func (b *bot) creditorOf(ctx context.Context, chat types.JID, acc core.Account, mentions []types.JID, now time.Time) (core.Account, string, error) {
	if len(mentions) > 0 {
		return b.Account(ctx, mentions[0].ToNonAD()), "", nil
	}
	owes, _, err := b.Wallets.debtsOf(ctx, chat, acc, now)
	if err != nil {
		return core.Account{}, "", fmt.Errorf("ler dívidas: %w", err)
	}
	if len(owes) == 0 {
		return core.Account{}, "Você não deve nada pra ninguém do grupo 😇", nil
	}
	if len(owes) > 1 {
		return core.Account{}, "Você deve pra mais de uma pessoa, marca pra quem vai pagar: !agiota pagar @pessoa 100", nil
	}
	return owes[0].lender, "", nil
}

func (b *bot) repayText(ctx context.Context, chat types.JID, acc, lender core.Account, owed *debt, balance int64, args []string, now time.Time) (string, error) {
	user, lenderUser := acc.JID.User, lender.JID.User
	if owed == nil {
		return fmt.Sprintf("@%s não deve nada pra @%s 😇", user, lenderUser), nil
	}
	if balance == 0 {
		return fmt.Sprintf("@%s tá liso 🪹 e deve %s pra @%s", user, Coins(owed.amount), lenderUser), nil
	}
	arg := "tudo"
	if len(args) > 0 {
		arg = args[0]
	}
	amount, ok := ParseStake(arg, min(balance, owed.amount))
	if !ok {
		return "Diz quanto pagar: !agiota pagar @pessoa 100 (ou metade, tudo)", nil
	}
	if amount > balance {
		return "Você só tem " + Coins(balance), nil
	}

	left, paid, err := b.Wallets.repayDebt(ctx, chat, acc, lender, amount, now)
	if errors.Is(err, errNoDebt) {
		return fmt.Sprintf("@%s não deve nada pra @%s 😇", user, lenderUser), nil
	}
	if errors.Is(err, ErrNoFunds) {
		return "Seu saldo mudou no meio do pagamento e não dá mais 🤨 Confere no !saldo", nil
	}
	if err != nil {
		return "", fmt.Errorf("pagar dívida: %w", err)
	}
	if left.amount == 0 {
		return fmt.Sprintf("✅ @%s pagou %s pra @%s e quitou a dívida 🤝", user, Coins(paid), lenderUser), nil
	}
	text := fmt.Sprintf("💸 @%s pagou %s pra @%s. Ainda deve %s", user, Coins(paid), lenderUser, Coins(left.amount))
	if left.overdue(now) {
		return text + fmt.Sprintf(", e como o prazo já venceu, @%s ainda pode cobrar do jeito ruim", lenderUser), nil
	}
	return text + ", e o prazo acaba em " + core.FormatWait(left.due.Sub(now)), nil
}

func (b *bot) rivalsWithDebt(ctx context.Context, chat types.JID, attacker, target core.Account) (attackerJob, targetJob Profession, owed *debt, err error) {
	now := time.Now()
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) {
		attackerJob, targetJob, err = b.rivalProfessions(gctx, chat, attacker, target)
		return err
	})
	g.Go(func() (err error) { owed, err = b.Wallets.debtBetween(gctx, chat, target, attacker, now); return err })
	if err = g.Wait(); err != nil {
		return attackerJob, targetJob, nil, err
	}
	if owed != nil && !owed.overdue(now) {
		owed = nil
	}
	return attackerJob, targetJob, owed, nil
}

func (p Profession) collectionEdge(owed *debt) float64 {
	if owed == nil {
		return 0
	}
	return p.collectBonus
}

func (b *bot) salaryLien(ctx context.Context, chat types.JID, debtor core.Account, now time.Time) (debtLien, error) {
	owes, _, err := b.Wallets.debtsOf(ctx, chat, debtor, now)
	if err != nil {
		return debtLien{}, fmt.Errorf("ler dívidas: %w", err)
	}
	owes = slices.DeleteFunc(owes, func(d debt) bool { return !d.overdue(now) })
	jobs := make([]Profession, len(owes))
	g, gctx := errgroup.WithContext(ctx)
	for i, d := range owes {
		g.Go(func() (err error) { jobs[i], err = b.ProfessionOf(gctx, chat, d.lender); return err })
	}
	if err := g.Wait(); err != nil {
		return debtLien{}, err
	}
	i := slices.IndexFunc(jobs, func(job Profession) bool { return job.garnish > 0 })
	if i < 0 {
		return debtLien{}, nil
	}
	return debtLien{lender: owes[i].lender, percent: jobs[i].garnish}, nil
}

func garnishPay(ctx context.Context, tx *sql.Tx, chat types.JID, debtor core.Account, lien debtLien, earned int64, now time.Time) (debtGarnish, error) {
	cut := earned * lien.percent / 100
	if cut <= 0 {
		return debtGarnish{}, nil
	}
	left, paid, err := settleDebt(ctx, tx, chat, debtor, lien.lender, cut, now)
	if errors.Is(err, errNoDebt) || errors.Is(err, ErrNoFunds) {
		return debtGarnish{}, nil
	}
	return debtGarnish{lender: lien.lender, paid: paid, left: left.amount}, err
}

func (g debtGarnish) people() []types.JID {
	if g.paid == 0 {
		return nil
	}
	return []types.JID{g.lender.JID}
}

func (g debtGarnish) text() string {
	if g.paid == 0 {
		return ""
	}
	if g.left == 0 {
		return fmt.Sprintf("\n🦈 @%s tava de olho no seu salário: %s foram direto pra dívida, que agora tá quitada 🤝", g.lender.JID.User, Coins(g.paid))
	}
	return fmt.Sprintf("\n🦈 @%s tava de olho no seu salário: %s foram direto pra dívida, e ainda falta %s", g.lender.JID.User, Coins(g.paid), Coins(g.left))
}

func debtCollectionText(owed *debt) string {
	return fmt.Sprintf("\n💸 Foi cobrança: @%s deve %s pra @%s e o prazo venceu há %s, então não conta como crime",
		owed.debtor.JID.User, Coins(owed.amount), owed.lender.JID.User, core.FormatWait(time.Since(owed.due)))
}

func (s *walletStore) lend(ctx context.Context, d debt) error {
	chat := d.chat
	return s.inTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx,
			`INSERT INTO dividas (chat, devedor, devedor_jid, credor, credor_jid, divida, juros, juros_atraso, desde, vence_em)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT DO NOTHING`,
			chat.String(), d.debtor.Key, d.debtor.JID.String(), d.lender.Key, d.lender.JID.String(),
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
		for _, acc := range []core.Account{d.debtor, d.lender} {
			if err := openWallet(ctx, tx, chat, acc); err != nil {
				return err
			}
		}
		return moveCoins(ctx, tx, chat, d.lender, d.debtor, d.amount)
	})
}

func (s *walletStore) debtBetween(ctx context.Context, chat types.JID, debtor, lender core.Account, now time.Time) (*debt, error) {
	row := s.db.QueryRowContext(ctx, debtColumns+" WHERE chat = ? AND devedor = ? AND credor = ?", chat.String(), debtor.Key, lender.Key)
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

func (s *walletStore) debtsOf(ctx context.Context, chat types.JID, acc core.Account, now time.Time) (owes, lent []debt, err error) {
	rows, err := s.db.QueryContext(ctx, debtColumns+" WHERE chat = ? AND (devedor = ? OR credor = ?) ORDER BY vence_em", chat.String(), acc.Key, acc.Key)
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
		if d.debtor.Key == acc.Key {
			owes = append(owes, d)
			continue
		}
		lent = append(lent, d)
	}
	return owes, lent, rows.Err()
}

func (s *walletStore) repayDebt(ctx context.Context, chat types.JID, debtor, lender core.Account, amount int64, now time.Time) (left debt, paid int64, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) (err error) {
		left, paid, err = settleDebt(ctx, tx, chat, debtor, lender, amount, now)
		return err
	})
	return left, paid, err
}

func settleDebt(ctx context.Context, tx *sql.Tx, chat types.JID, debtor, lender core.Account, amount int64, now time.Time) (debt, int64, error) {
	row := tx.QueryRowContext(ctx, debtColumns+" WHERE chat = ? AND devedor = ? AND credor = ?", chat.String(), debtor.Key, lender.Key)
	d, err := scanDebt(row, chat)
	if errors.Is(err, sql.ErrNoRows) {
		return debt{}, 0, errNoDebt
	}
	if err != nil {
		return debt{}, 0, err
	}
	left := d.accrued(now)
	paid := min(amount, left.amount)
	if err := openWallet(ctx, tx, chat, lender); err != nil {
		return debt{}, 0, err
	}
	if err := moveCoins(ctx, tx, chat, debtor, lender, paid); err != nil {
		return debt{}, 0, err
	}
	left.amount -= paid
	return left, paid, storeDebt(ctx, tx, left)
}

const debtColumns = "SELECT devedor, devedor_jid, credor, credor_jid, divida, juros, juros_atraso, desde, vence_em FROM dividas"

func scanDebt(row interface{ Scan(dest ...any) error }, chat types.JID) (debt, error) {
	var (
		d                    debt
		debtorJID, lenderJID string
		since, due           int64
	)
	err := row.Scan(&d.debtor.Key, &debtorJID, &d.lender.Key, &lenderJID, &d.amount, &d.rate, &d.lateRate, &since, &due)
	if err != nil {
		return debt{}, err
	}
	if d.debtor.JID, err = types.ParseJID(debtorJID); err != nil {
		return debt{}, err
	}
	if d.lender.JID, err = types.ParseJID(lenderJID); err != nil {
		return debt{}, err
	}
	d.chat, d.since, d.due = chat, time.Unix(since, 0), time.Unix(due, 0)
	return d, nil
}

func storeDebt(ctx context.Context, tx *sql.Tx, d debt) error {
	if d.amount <= 0 {
		_, err := tx.ExecContext(ctx, "DELETE FROM dividas WHERE chat = ? AND devedor = ? AND credor = ?", d.chat.String(), d.debtor.Key, d.lender.Key)
		return err
	}
	_, err := tx.ExecContext(ctx,
		"UPDATE dividas SET divida = ?, desde = ? WHERE chat = ? AND devedor = ? AND credor = ?",
		d.amount, d.since.Unix(), d.chat.String(), d.debtor.Key, d.lender.Key,
	)
	return err
}
