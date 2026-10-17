package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	dailyMin         = 150
	dailyMax         = 300
	robEvery         = 30 * time.Minute
	robMinVictim     = 50
	robSuccessChance = 0.4
	richestCount     = 10

	quizReward          = 50
	quemfalouReward     = 50
	forcaRewardPerLife  = 15
	prizeMinPlayers     = 2
	alimonyPercent      = 20
	cheatingFinePercent = 15
)

var dailyCooldown = cooldown{column: "daily_em", every: 24 * time.Hour}

func (b *bot) account(ctx context.Context, person types.JID) account {
	person = person.ToNonAD()
	return account{key: slices.Min(b.userAliases(ctx, person)), jid: person}
}

func (b *bot) saldo(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := firstMentionOrSender(evt).ToNonAD()
	acc := b.account(ctx, person)
	var (
		balance       int64
		rank          int
		saved         savings
		hasSavings    bool
		titleID       string
		job           profession
		bodyguardLeft time.Duration
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { balance, rank, err = b.wallets.balance(gctx, chat, acc); return err })
	g.Go(func() (err error) { saved, hasSavings, err = b.wallets.savings(gctx, chat, acc); return err })
	g.Go(func() (err error) { titleID, err = b.wallets.titleOf(gctx, chat, acc); return err })
	g.Go(func() (err error) { job, err = b.professionOf(gctx, chat, acc); return err })
	g.Go(func() (err error) {
		bodyguardLeft, err = b.wallets.leftUntil(gctx, chat, acc, "seguranca_ate", time.Now())
		return err
	})
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler saldo: %w", err)
	}
	note := fmt.Sprintf("🏆 %dº mais rico do grupo", rank)
	if balance == 0 {
		note = "_Tá liso. Usa !daily e !trabalhar pra começar_"
	}
	lines := []string{fmt.Sprintf("💰 %s@%s tem %s", titleBadge(titleID), person.User, coins(balance)), note}
	if hasSavings {
		lines = append(lines, fmt.Sprintf("🔒 Na poupança: %s, que voltam com %s de juros", coins(saved.amount), coins(saved.interest)))
	}
	if home := homeText(job.estate); home != "" {
		lines = append(lines, home)
	}
	if bodyguardLeft > 0 {
		lines = append(lines, fmt.Sprintf("🕶️ Com segurança particular por mais %s", formatWait(bodyguardLeft)))
	}
	_, err := b.sendMentions(ctx, chat, strings.Join(lines, "\n"), []types.JID{person})
	return err
}

func (b *bot) daily(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.account(ctx, person)
	now := time.Now()
	job, p, err := b.workTerms(ctx, chat, acc, now)
	if err != nil {
		return err
	}
	amount := dailyMin + rand.Int64N(dailyMax-dailyMin+1)
	amount += amount * job.dailyBonus / 100
	amount += amount * job.decrees.level(dailyDecree) / 100
	levy := p.incomeLevy(acc)
	wait, tax, err := b.wallets.claim(ctx, chat, acc, dailyCooldown, amount, levy, now)
	if err != nil {
		return fmt.Errorf("pegar moedas: %w", err)
	}
	if wait > 0 {
		_, err = b.sendMentions(ctx, chat, fmt.Sprintf("⏳ @%s já pegou o daily hoje. Volta em %s", person.User, formatWait(wait)), []types.JID{person})
		return err
	}
	text := fmt.Sprintf("🎁 @%s pegou o daily e ganhou %s", person.User, coins(amount)) + job.decrees.note(dailyDecree) + incomeTaxText(tax, levy)
	people := []types.JID{person}
	if tax > 0 {
		people = append(people, levy.payee()...)
	}
	_, err = b.sendMentions(ctx, chat, text, people)
	return err
}

func (b *bot) apostar(ctx context.Context, evt *events.Message) error {
	return b.gamble(ctx, evt, "!apostar", func(stake int64, jinxed bool) (int64, string) {
		if jinxed || rand.IntN(2) == 0 {
			return 0, "🪙 Deu coroa! @%s perdeu %s"
		}
		return 2 * stake, "🪙 Deu cara! @%s ganhou %s"
	})
}

func (b *bot) cassino(ctx context.Context, evt *events.Message) error {
	return b.gamble(ctx, evt, "!cassino", func(stake int64, jinxed bool) (int64, string) {
		reels := spinReels()
		for jinxed && distinctReels(reels) < len(reels) {
			reels = spinReels()
		}
		machine := "🎰 [ " + strings.Join(reels, " | ") + " ]\n\n"
		distinct := distinctReels(reels)
		if distinct == 1 && reels[0] == "💎" {
			return 10 * stake, machine + "💎 *JACKPOT!* @%s ganhou %s"
		}
		if distinct == 1 {
			return 5 * stake, machine + "🔥 Trinca! @%s ganhou %s"
		}
		if distinct == 2 {
			return 2 * stake, machine + "✨ Deu par! @%s ganhou %s"
		}
		return 0, machine + "💸 Nada. @%s perdeu %s"
	})
}

func spinReels() []string {
	symbols := []string{"🍒", "🍋", "🍇", "🔔", "⭐", "💎"}
	reels := make([]string, 3)
	for i := range reels {
		reels[i] = symbols[rand.IntN(len(symbols))]
	}
	return reels
}

func distinctReels(reels []string) int {
	return len(slices.Compact(slices.Sorted(slices.Values(reels))))
}

func (b *bot) gamble(ctx context.Context, evt *events.Message, syntax string, play func(stake int64, jinxed bool) (payout int64, result string)) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.account(ctx, person)
	var (
		bet     wager
		refusal string
		job     profession
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) {
		bet, refusal, err = b.placeBet(gctx, chat, acc, firstPlainArg(messageText(evt.Message)), syntax)
		return err
	})
	g.Go(func() (err error) { job, err = b.professionOf(gctx, chat, acc); return err })
	if err := g.Wait(); err != nil {
		return err
	}
	if refusal != "" {
		return b.sendText(ctx, chat, refusal)
	}

	stake := bet.stake
	payout, result := play(stake, rand.Float64() < job.betJinx)
	payout = casinoPrize(stake, payout, bet.boost)
	refund := job.refund(stake, payout)
	balance, err := b.wallets.settleBet(ctx, chat, acc, stake, payout+refund, bet.levy)
	if errors.Is(err, errNoFunds) {
		return b.sendText(ctx, chat, stakeChanged)
	}
	if err != nil {
		return fmt.Errorf("pagar aposta: %w", err)
	}
	won := max(payout-stake, stake-payout)
	text := fmt.Sprintf(result, person.User, coins(won)) + prizeTag(stake, payout, bet.boost)
	if refund > 0 {
		text += fmt.Sprintf("\n🧾 Mas a profissão devolveu %s", coins(refund))
	}
	text += betLevyText(stake, payout, bet.levy) + "\n💰 Saldo: " + coins(balance)
	_, err = b.sendMentions(ctx, chat, text, []types.JID{person})
	return err
}

type wager struct {
	stake int64
	boost int64
	levy  pixLevy
}

func (b *bot) placeBet(ctx context.Context, chat types.JID, acc account, arg, syntax string) (wager, string, error) {
	p, _, err := b.wallets.presidency(ctx, chat, time.Now())
	if err != nil {
		return wager{}, "", err
	}
	if p.decrees.on(betsDecree) {
		return wager{}, casinoClosedText, nil
	}
	levy := p.levy(betTaxDecree, acc)
	stake, refusal, err := b.stakeAfterLevy(ctx, chat, acc, arg, syntax, levy.percent)
	return wager{stake: stake, boost: p.decrees.level(casinoDecree), levy: levy}, refusal, err
}

func (b *bot) placeStake(ctx context.Context, chat types.JID, acc account, arg, syntax string) (int64, string, error) {
	return b.stakeAfterLevy(ctx, chat, acc, arg, syntax, 0)
}

func (b *bot) stakeAfterLevy(ctx context.Context, chat types.JID, acc account, arg, syntax string, percent int64) (int64, string, error) {
	balance, _, err := b.wallets.balance(ctx, chat, acc)
	if err != nil {
		return 0, "", fmt.Errorf("ler saldo: %w", err)
	}
	if balance == 0 {
		return 0, "Você tá liso 🪹 Usa !daily e !trabalhar pra ter o que apostar", nil
	}
	budget := balance * 100 / (100 + percent)
	stake, ok := parseStake(arg, budget)
	if !ok {
		return 0, fmt.Sprintf("Diz quanto apostar: %[1]s 100, %[1]s metade, %[1]s 10%% ou %[1]s tudo", syntax), nil
	}
	if stake <= budget {
		return stake, "", nil
	}
	if percent == 0 {
		return 0, "Você só tem " + coins(balance), nil
	}
	return 0, fmt.Sprintf("Você tem %s, mas toda aposta paga %d%% de taxa por decreto, então dá pra apostar no máximo %s", coins(balance), percent, coins(budget)), nil
}

func (b *bot) roubar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	mentions := mentionedJIDs(evt.Message)
	if len(mentions) == 0 {
		return b.sendText(ctx, chat, "Marca quem você vai roubar: !roubar @pessoa")
	}
	thief, victim := evt.Info.Sender.ToNonAD(), mentions[0].ToNonAD()
	thiefAcc, victimAcc := b.account(ctx, thief), b.account(ctx, victim)
	if thiefAcc.key == victimAcc.key {
		return b.sendText(ctx, chat, "Roubar de você mesmo? Isso se chama mudar o dinheiro de bolso 🤡")
	}

	thiefJob, victimJob, err := b.rivalProfessions(ctx, chat, thiefAcc, victimAcc)
	if err != nil {
		return err
	}
	chance := clampChance(robSuccessChance + thiefJob.robBonus - victimJob.guard - thiefJob.curfew())
	r, err := b.wallets.rob(ctx, chat, thiefAcc, victimAcc, thiefJob.robCooldown(), thiefJob.abuse(robAbuse), time.Now(), robMinVictim, robOutcome(chance))
	if err != nil {
		return fmt.Errorf("roubar: %w", err)
	}
	_, err = b.sendMentions(ctx, chat, robberyText(r, thief, victim)+r.abuse.note("Abusou do poder e roubou sem esperar a polícia esquecer"), []types.JID{thief, victim})
	return err
}

func robOutcome(chance float64) func(thiefBalance, victimBalance int64) (bool, int64) {
	return func(thiefBalance, victimBalance int64) (bool, int64) {
		if rand.Float64() < chance {
			return true, victimBalance * (10 + rand.Int64N(21)) / 100
		}
		return false, thiefBalance * (10 + rand.Int64N(11)) / 100
	}
}

func robberyText(r robbery, thief, victim types.JID) string {
	if r.wait > 0 {
		return fmt.Sprintf("🚓 @%s ainda tá sendo procurado pela polícia. Espera %s pra roubar de novo", thief.User, formatWait(r.wait))
	}
	if r.poor {
		return fmt.Sprintf("🪹 @%s tem menos de %s, nem vale o risco", victim.User, coins(robMinVictim))
	}
	if r.alarmed {
		return fmt.Sprintf("🚨 @%s tentou roubar @%s, mas o alarme disparou! O ladrão fugiu de mãos vazias e o alarme queimou", thief.User, victim.User)
	}
	if r.success {
		return fmt.Sprintf("🦹 @%s roubou %s de @%s e saiu correndo!", thief.User, coins(r.amount), victim.User)
	}
	if r.amount == 0 {
		return fmt.Sprintf("🚔 @%s foi pego tentando roubar @%s, mas tava tão liso que saiu só com a vergonha", thief.User, victim.User)
	}
	return fmt.Sprintf("🚔 @%s foi pego tentando roubar @%s e pagou %s de indenização pra vítima", thief.User, victim.User, coins(r.amount))
}

func (b *bot) transferir(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	mentions := mentionedJIDs(evt.Message)
	arg := firstPlainArg(messageText(evt.Message))
	if len(mentions) == 0 || arg == "" {
		return b.sendText(ctx, chat, "Use assim: !transferir @pessoa 100")
	}
	from, to := evt.Info.Sender.ToNonAD(), mentions[0].ToNonAD()
	fromAcc, toAcc := b.account(ctx, from), b.account(ctx, to)
	if fromAcc.key == toAcc.key {
		return b.sendText(ctx, chat, "Mandar moeda pra você mesmo não muda nada 🤨")
	}

	var (
		balance int64
		p       presidency
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { balance, _, err = b.wallets.balance(gctx, chat, fromAcc); return err })
	g.Go(func() (err error) { p, _, err = b.wallets.presidency(gctx, chat, time.Now()); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler saldo: %w", err)
	}
	amount, ok := parseStake(arg, balance)
	if !ok {
		return b.sendText(ctx, chat, "Não entendi o valor. Use assim: !transferir @pessoa 100 (ou metade, 10%, tudo)")
	}
	levy := p.levy(pixDecree, fromAcc, toAcc)
	people := append([]types.JID{from, to}, levy.payee()...)
	pix, err := b.wallets.transferWithPix(ctx, chat, fromAcc, toAcc, amount, levy)
	if errors.Is(err, errNoFunds) {
		return b.sendText(ctx, chat, "Você só tem "+coins(balance))
	}
	if err != nil {
		return fmt.Errorf("transferir: %w", err)
	}
	_, err = b.sendMentions(ctx, chat, fmt.Sprintf("💸 @%s mandou %s pra @%s", from.User, coins(amount), to.User)+pixText(pix, levy), people)
	return err
}

func (b *bot) ricos(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	wallets, err := b.wallets.richest(ctx, chat, richestCount)
	if err != nil {
		return fmt.Errorf("ler ranking de saldo: %w", err)
	}
	if len(wallets) == 0 {
		return b.sendText(ctx, chat, "Ninguém tem moeda ainda 🪹 Usa !daily pra começar")
	}
	medals := []string{"🥇", "🥈", "🥉"}
	lines := []string{"🏆 *Os mais ricos do grupo*", ""}
	owners := make([]types.JID, len(wallets))
	for i, w := range wallets {
		place := fmt.Sprintf("%dº", i+1)
		if i < len(medals) {
			place = medals[i]
		}
		lines = append(lines, fmt.Sprintf("%s %s@%s: %s", place, titleBadge(w.title), w.owner.User, coins(w.balance)))
		owners[i] = w.owner
	}
	_, err = b.sendMentions(ctx, chat, strings.Join(lines, "\n"), owners)
	return err
}

func (b *bot) economyOn(ctx context.Context, chat types.JID) bool {
	on, err := b.modes.enabled(ctx, chat.ToNonAD(), modeEconomy)
	if err != nil {
		b.client.Log.Warnf("checar modo economia de %s: %v", chat, err)
	}
	return on
}

func (b *bot) accounts(ctx context.Context, people []types.JID) []account {
	accounts := make([]account, len(people))
	for i, p := range people {
		accounts[i] = b.account(ctx, p)
	}
	return accounts
}

func (b *bot) reward(ctx context.Context, chat types.JID, winners []types.JID, players int, amount int64) string {
	if len(winners) == 0 || !b.economyOn(ctx, chat) {
		return ""
	}
	if players < prizeMinPlayers {
		return fmt.Sprintf("🪹 Sem moedas dessa vez: o prêmio só sai com pelo menos %d pessoas jogando", prizeMinPlayers)
	}
	if err := b.wallets.credit(ctx, chat.ToNonAD(), b.accounts(ctx, winners), amount); err != nil {
		b.client.Log.Errorf("pagar prêmio pra %s: %v", joinMentions(winners), err)
		return ""
	}
	if len(winners) == 1 {
		return fmt.Sprintf("💰 @%s ganhou %s", winners[0].User, coins(amount))
	}
	return fmt.Sprintf("💰 %s ganharam %s cada", joinMentions(winners), coins(amount))
}

func (b *bot) charge(ctx context.Context, chat, payer types.JID, payees []types.JID, percent int64, label string) string {
	if len(payees) == 0 || !b.economyOn(ctx, chat) {
		return ""
	}
	payerAcc := b.account(ctx, payer)
	job, err := b.professionOf(ctx, chat.ToNonAD(), payerAcc)
	if err != nil {
		b.client.Log.Errorf("ler profissão de %s pra cobrar %s: %v", payer, label, err)
	}
	percent = percent * (100 - job.legalDiscount) / 100
	each, err := b.wallets.payShare(ctx, chat.ToNonAD(), payerAcc, b.accounts(ctx, payees), percent)
	if err != nil {
		b.client.Log.Errorf("cobrar %s de %s: %v", label, payer, err)
		return ""
	}
	if each == 0 {
		return fmt.Sprintf("%s: @%s tá liso e não tem nem como pagar 🪹", label, payer.User)
	}
	receivers := joinMentions(payees)
	if len(payees) > 1 {
		receivers = "cada um: " + receivers
	}
	text := fmt.Sprintf("%s de %d%% do saldo: @%s pagou %s pra %s", label, percent, payer.User, coins(each), receivers)
	if job.legalDiscount > 0 {
		text += fmt.Sprintf(". Saiu %d%% mais barato porque @%s %s", job.legalDiscount, payer.User, job.title)
	}
	return text
}

func (p profession) refund(stake, payout int64) int64 {
	if payout > 0 {
		return 0
	}
	return stake * p.cashback / 100
}

func appendLine(text, line string) string {
	if line == "" {
		return text
	}
	return text + "\n" + line
}

func firstPlainArg(text string) string {
	fields := strings.Fields(text)
	i := slices.IndexFunc(fields[min(1, len(fields)):], func(f string) bool { return !strings.HasPrefix(f, "@") })
	if i < 0 {
		return ""
	}
	return strings.ToLower(fields[i+1])
}

func parseStake(arg string, balance int64) (int64, bool) {
	named := map[string]int64{"tudo": balance, "all": balance, "metade": balance / 2}
	if amount, ok := named[arg]; ok {
		return amount, amount > 0
	}
	if percent, ok := strings.CutSuffix(arg, "%"); ok {
		n, err := strconv.ParseInt(percent, 10, 64)
		if err != nil || n <= 0 || n > 100 {
			return 0, false
		}
		amount := balance * n / 100
		return amount, amount > 0
	}
	amount, err := strconv.ParseInt(strings.ReplaceAll(arg, ".", ""), 10, 64)
	return amount, err == nil && amount > 0
}

func coins(n int64) string {
	unit := "moedas"
	if n == 1 {
		unit = "moeda"
	}
	return "🪙 " + thousands(n) + " " + unit
}

func thousands(n int64) string {
	digits := strconv.FormatInt(n, 10)
	var sb strings.Builder
	for i, d := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			sb.WriteByte('.')
		}
		sb.WriteRune(d)
	}
	return sb.String()
}

func formatWait(d time.Duration) string {
	d = d.Round(time.Minute)
	hours, minutes := int(d.Hours()), int(d.Minutes())%60
	if hours > 0 && minutes > 0 {
		return fmt.Sprintf("%dh %dmin", hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dmin", max(minutes, 1))
}

func (b *bot) economia(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	args := strings.Fields(foldLetters(messageText(evt.Message)))[1:]
	if len(args) < 2 || args[0] != "resetar" {
		return b.sendText(ctx, chat, "Use assim: !economia resetar intervalos (ou !economia resetar intervalos @pessoa)")
	}
	if !slices.Contains([]string{"intervalos", "intervalo", "esperas"}, args[1]) {
		return b.sendText(ctx, chat, "Só dá pra resetar os intervalos. Dinheiro, mochila e o resto ninguém reseta, nem o dono")
	}
	people := mentionedJIDs(evt.Message)
	accounts := make([]account, len(people))
	for i, person := range people {
		accounts[i] = b.account(ctx, person)
	}
	reset, err := b.wallets.resetCooldowns(ctx, chat, accounts)
	if err != nil {
		return fmt.Errorf("resetar intervalos: %w", err)
	}
	if len(people) > 0 {
		_, err = b.sendMentions(ctx, chat, "⏱️ Intervalos zerados pra "+mentionList(accounts)+", e quem tava preso, de refém ou morto tá livre. O dinheiro continua igualzinho", people)
		return err
	}
	return b.sendText(ctx, chat, fmt.Sprintf("⏱️ Intervalos zerados pra %d pessoa(s) desta conversa, e quem tava preso, de refém ou morto tá livre. O dinheiro continua igualzinho", reset))
}

func (s *walletStore) resetCooldowns(ctx context.Context, chat types.JID, accounts []account) (reset int64, err error) {
	columns := []string{
		"daily_em", "trabalho_em", "roubo_em", "profissao_em", "pesca_em", "mineracao_em", "lixao_em",
		"inseto_em", "furto_em", "cafeina_em", "prisao_em", "abuso_em", "processo_em", "fuga_em", "sequestro_em",
		"assalto_em", "denuncia_em", "assassinato_em", "libertacao_em", "abuso_trabalho_em", "abuso_roubo_em",
		"abuso_sequestro_em", "ressurreicao_em", "encanto_em", "preso_ate", "morto_ate", "crime_em",
	}
	args := []any{chat.String()}
	for _, acc := range accounts {
		args = append(args, acc.key)
	}
	where := func(column string) string {
		if len(accounts) == 0 {
			return " WHERE chat = ?"
		}
		return " WHERE chat = ? AND " + column + " IN (?" + strings.Repeat(", ?", len(accounts)-1) + ")"
	}
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, "UPDATE carteiras SET "+strings.Join(columns, " = 0, ")+" = 0"+where("pessoa"), args...)
		if err != nil {
			return err
		}
		if reset, err = res.RowsAffected(); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "DELETE FROM sequestros"+where("refem"), args...)
		return err
	})
	return reset, err
}
