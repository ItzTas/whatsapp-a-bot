package economy

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	dailyMin          = 1_000
	dailyMax          = 2_000
	robEvery          = 30 * time.Minute
	robMinVictim      = 50
	robSuccessChance  = 0.4
	robLootMin        = 10
	robLootMax        = 30
	brokeThief        = 50
	brokeCaughtChance = 0.5
	coinWinChance     = 0.475
	slotHouseCut      = 0.045
	RichestCount      = 10

	QuizReward          = 50
	QuemfalouReward     = 50
	ForcaRewardPerLife  = 15
	prizeMinPlayers     = 2
	AlimonyPercent      = 20
	CheatingFinePercent = 15
)

var dailyCooldown = cooldown{column: "daily_em", every: 24 * time.Hour}

func (b *bot) saldo(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := core.FirstMentionOrSender(evt).ToNonAD()
	acc := b.Account(ctx, person)
	var (
		balance    int64
		rank       int
		saved      Savings
		hasSavings bool
		titleID    string
		job        Profession
		health     int64
		vault      homeVault
		shield     time.Duration
		contracts  []contract
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { balance, rank, err = b.Wallets.Balance(gctx, chat, acc); return err })
	g.Go(func() (err error) { saved, hasSavings, err = b.Wallets.Savings(gctx, chat, acc); return err })
	g.Go(func() (err error) { titleID, err = b.Wallets.titleOf(gctx, chat, acc); return err })
	g.Go(func() (err error) { job, err = b.ProfessionOf(gctx, chat, acc); return err })
	g.Go(func() (err error) { health, err = b.Wallets.Health(gctx, chat, acc, time.Now()); return err })
	g.Go(func() (err error) { vault, err = b.Wallets.vault(gctx, chat, acc); return err })
	g.Go(func() (err error) { shield, err = b.Wallets.shieldLeft(gctx, chat, acc, time.Now()); return err })
	g.Go(func() (err error) { contracts, err = b.Wallets.contracts(gctx, chat, acc, time.Now()); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler saldo: %w", err)
	}
	note := fmt.Sprintf("🏆 %dº mais rico do grupo", rank)
	if balance == 0 {
		note = "_Tá liso. Usa !daily e !trabalhar pra começar_"
	}
	lines := []string{fmt.Sprintf("💰 %s@%s tem %s", titleBadge(titleID), person.User, Coins(balance)), note}
	if hasSavings {
		lines = append(lines, fmt.Sprintf("🔒 Na poupança: %s, que voltam com %s de juros", Coins(saved.Amount), Coins(saved.Interest)))
	}
	if home := homeText(job.estate); home != "" {
		lines = append(lines, home)
	}
	if exposure := exposureText(balance); exposure != "" {
		lines = append(lines, exposure)
	}
	lines = append(lines, cleanText(job))
	if shield > 0 {
		lines = append(lines, shieldText(shield))
	}
	if text := contractText(contracts, time.Now()); text != "" {
		lines = append(lines, text)
	}
	if vault.stored > 0 {
		lines = append(lines, fmt.Sprintf("🔐 No cofre de casa: %s, longe dos ladrões", Coins(vault.stored)))
	}
	if health < MaxHealth {
		lines = append(lines, "🩺 Saúde: "+healthText(health)+" (veja !saude)")
	}
	_, err := b.SendMentions(ctx, chat, strings.Join(lines, "\n"), []types.JID{person})
	return err
}

func (b *bot) daily(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	now := time.Now()
	var (
		job  Profession
		p    presidency
		pets []item
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { job, p, err = b.workTerms(gctx, chat, acc, now); return err })
	g.Go(func() (err error) {
		pets, err = b.Wallets.holding(gctx, chat, acc, func(it item) bool { return it.pet })
		return err
	})
	if err := g.Wait(); err != nil {
		return err
	}
	amount := dailyMin + rand.Int64N(dailyMax-dailyMin+1)
	amount += amount * job.dailyBonus / 100
	amount += amount * job.decrees.level(dailyDecree) / 100
	levy := p.incomeLevy(acc)
	fetched := fetchGift(pets)
	w, err := b.Wallets.claim(ctx, chat, acc, dailyCooldown, amount, fetched.gift, levy, now)
	if err != nil {
		return fmt.Errorf("pegar moedas: %w", err)
	}
	if w.wait > 0 {
		_, err = b.SendMentions(ctx, chat, fmt.Sprintf("⏳ @%s já pegou o daily hoje. Volta em %s", person.User, core.FormatWait(w.wait)), []types.JID{person})
		return err
	}
	text := fmt.Sprintf("🎁 @%s pegou o daily e ganhou %s", person.User, Coins(amount)) + job.decrees.note(dailyDecree) + incomeTaxText(w.tax, levy) + w.tithe.text() + fetched.text()
	people := []types.JID{person}
	if w.tax > 0 {
		people = append(people, levy.payee()...)
	}
	_, err = b.SendMentions(ctx, chat, text, people)
	return err
}

func (b *bot) apostar(ctx context.Context, evt *events.Message) error {
	return b.gamble(ctx, evt, "!apostar", func(stake int64, jinxed bool) (int64, string) {
		if jinxed || rand.Float64() >= coinWinChance {
			return 0, "🪙 Deu coroa! @%s perdeu %s"
		}
		return 2 * stake, "🪙 Deu cara! @%s ganhou %s"
	})
}

func (b *bot) cassino(ctx context.Context, evt *events.Message) error {
	return b.gamble(ctx, evt, "!cassino", func(stake int64, jinxed bool) (int64, string) {
		reels := spinReels()
		rigged := jinxed || rand.Float64() < slotHouseCut
		for rigged && distinctReels(reels) < len(reels) {
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
	acc := b.Account(ctx, person)
	bet, refusal, err := b.PlaceBet(ctx, chat, acc, core.FirstPlainArg(core.MessageText(evt.Message)), syntax)
	if err != nil {
		return err
	}
	if refusal != "" {
		return b.SendText(ctx, chat, refusal)
	}

	stake := bet.Stake
	payout, result := play(stake, rand.Float64() < bet.job.jinx()+bet.Boost.Risk(stake))
	if payout < stake && rand.Float64() < bet.Boost.Favor(stake) {
		payout, result = play(stake, false)
	}
	payout = bet.Boost.Pay(stake, payout)
	saved := bet.Boost.Rescue(stake, payout)
	refund := bet.job.refund(stake, payout+saved)
	balance, err := b.Wallets.SettleBet(ctx, chat, acc, stake, payout+saved+refund, bet.Boost.levy)
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, chat, stakeChanged)
	}
	if err != nil {
		return fmt.Errorf("pagar aposta: %w", err)
	}
	b.advance(chat, acc, betGoal, payout-stake)
	won := max(payout-stake, stake-payout)
	text := fmt.Sprintf(result, person.User, Coins(won)) + bet.Boost.Tag(stake, payout) + SavedBetText(saved) + BetLevyText(stake, bet.Boost.levy)
	if refund > 0 {
		text += fmt.Sprintf("\n🧾 Mas a profissão devolveu %s", Coins(refund))
	}
	text += "\n💰 Saldo: " + Coins(balance)
	if _, err := b.SendMentions(ctx, chat, text, []types.JID{person}); err != nil {
		return err
	}
	b.tallyBet(ctx, chat, acc, syntax, stake, payout+saved+refund, balance)
	return nil
}

type wager struct {
	Stake int64
	Boost PrizeBoost
	job   Profession
}

func (b *bot) PlaceBet(ctx context.Context, chat types.JID, acc core.Account, arg, syntax string) (wager, string, error) {
	var (
		p      presidency
		job    Profession
		heat   int64
		solace int64
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { p, _, err = b.Wallets.presidency(gctx, chat, time.Now()); return err })
	g.Go(func() (err error) { job, err = b.ProfessionOf(gctx, chat, acc); return err })
	g.Go(func() (err error) { heat, solace, err = b.Wallets.betRecord(gctx, chat, acc); return err })
	if err := g.Wait(); err != nil {
		return wager{}, "", err
	}
	stake, refusal, err := b.placeStake(ctx, chat, acc, arg, syntax)
	if err != nil || refusal != "" {
		return wager{}, refusal, err
	}
	if refusal, err = b.markWarning(ctx, chat, acc, syntax, stake); err != nil || refusal != "" {
		return wager{}, refusal, err
	}
	return wager{Stake: stake, Boost: PrizeBoost{decree: max(job.decrees.level(casinoDecree), 0), levy: p.casinoLevy(acc), luck: job.casinoLuck, heat: heat, solace: solace, rescued: rand.Float64() < job.rescueOdds}, job: job}, "", nil
}

func (b *bot) placeStake(ctx context.Context, chat types.JID, acc core.Account, arg, syntax string) (int64, string, error) {
	rule, err := b.betRuleOf(ctx, chat, acc)
	if err != nil {
		return 0, "", err
	}
	stake, whole, refusal, err := b.affordable(ctx, chat, acc, arg, syntax, rule)
	if err != nil || refusal != "" {
		return stake, refusal, err
	}
	if refusal := b.allIn(ctx, chat, acc, stake, whole); refusal != "" {
		return 0, refusal, nil
	}
	return stake, "", nil
}

func (b *bot) affordable(ctx context.Context, chat types.JID, acc core.Account, arg, syntax string, rule stakeRule) (stake, whole int64, refusal string, err error) {
	balance, _, err := b.Wallets.Balance(ctx, chat, acc)
	if err != nil {
		return 0, 0, "", fmt.Errorf("ler saldo: %w", err)
	}
	if balance == 0 {
		return 0, 0, "Você tá liso 🪹 Usa !daily e !trabalhar pra ter o que apostar", nil
	}
	most, whole := rule.most(balance), rule.whole(balance)
	if most == 0 {
		return 0, 0, rule.tooMuch(balance), nil
	}
	stake, ok := ParseStake(arg, balance)
	if !ok {
		return 0, whole, fmt.Sprintf("Diz quanto apostar: %[1]s 100, %[1]s metade, %[1]s 10%% ou %[1]s tudo", syntax), nil
	}
	if arg == "tudo" || arg == "all" {
		stake = whole
	}
	if stake <= most {
		return stake, whole, "", nil
	}
	if stake <= balance {
		return 0, whole, rule.tooMuch(balance), nil
	}
	return 0, whole, "Você só tem " + Coins(balance), nil
}

func (b *bot) roubar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	aim, problem := b.targetOf(ctx, evt)
	if problem != "" {
		return b.SendText(ctx, chat, problem)
	}
	if !aim.found() {
		return b.SendText(ctx, chat, "Marca quem você vai roubar: !roubar @pessoa\n"+targetHint)
	}
	thief, victim := evt.Info.Sender.ToNonAD(), aim.person
	thiefAcc, victimAcc := b.Account(ctx, thief), b.Account(ctx, victim)
	if thiefAcc.Key == victimAcc.Key {
		return b.SendText(ctx, chat, "Roubar de você mesmo? Isso se chama mudar o dinheiro de bolso 🤡")
	}

	now := time.Now()
	var (
		thiefJob, victimJob Profession
		backer              firm
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) {
		thiefJob, victimJob, err = b.rivalProfessions(gctx, chat, thiefAcc, victimAcc)
		return err
	})
	g.Go(func() (err error) { backer, err = syndicateOf(gctx, b.Wallets.db, chat, thiefAcc, now); return err })
	if err := g.Wait(); err != nil {
		return err
	}
	chance := thiefJob.sober(clampChance(robSuccessChance + thiefJob.robBonus + backer.edge - victimJob.guard - thiefJob.curfew()))
	risk := robRisk.by(thiefJob).on(victimJob)
	caught := flagrant{odds: thiefJob.flagrantOdds(brokeCaughtChance), jail: Profession{jailTime: baseJail}.sentence(thiefJob)}
	r, err := b.Wallets.rob(ctx, chat, thiefAcc, victimAcc, thiefJob.robCooldown(), thiefJob.abuse(robAbuse), risk, caught, now, robMinVictim, robOutcome(chance, thiefJob.robHaul))
	if err != nil {
		return fmt.Errorf("roubar: %w", err)
	}
	cover := ""
	if r.success {
		b.advance(chat, thiefAcc, crimeGoal, 1)
		cover = backer.coverText("@" + thief.User)
	}
	hurtText, freed := r.scuffle.text(thief, victim)
	text := r.escort.escortText(victim) + r.mark.text(victim) + cover + robberyText(r, thief, victim) + r.abuse.note("Abusou do poder e roubou sem esperar a polícia esquecer") + hurtText
	_, err = b.SendMentions(ctx, chat, text, append([]types.JID{thief, victim}, freed...))
	if err != nil || r.wait > 0 || r.shield > 0 || r.poor || r.jail > 0 || r.scuffle.criminal.died {
		return err
	}
	if r.success {
		b.offerRecovery(ctx, chat, thiefAcc, victimAcc, r)
	}
	b.stirEvent(ctx, evt, eventSpot{place: "!roubar", won: r.success, victim: victim})
	return nil
}

func robOutcome(chance float64, haul int64) func(thiefBalance, victimBalance int64, mark easyMark) (bool, int64) {
	return func(thiefBalance, victimBalance int64, mark easyMark) (bool, int64) {
		if rand.Float64() < mark.odds(chance) {
			return true, min(victimBalance, looted(victimBalance, mark.jackpot, robLootMin+rand.Int64N(robLootMax-robLootMin+1))*(100+haul)/100)
		}
		return false, thiefBalance * (10 + rand.Int64N(11)) / 100
	}
}

func exposed(balance, percent int64) int64 {
	brackets := []struct{ upTo, weight int64 }{
		{1_000, 60},
		{10_000, 45},
		{100_000, 30},
		{1_000_000, 15},
		{math.MaxInt64, 6},
	}
	var take, floor int64
	for _, br := range brackets {
		if balance <= floor {
			break
		}
		take += (min(balance, br.upTo) - floor) * percent * br.weight
		floor = br.upTo
	}
	return take / (100 * brackets[0].weight)
}

func exposureText(balance int64) string {
	if balance < robMinVictim {
		return ""
	}
	most := exposed(balance, robLootMax)
	share := strings.Replace(strconv.FormatFloat(math.Round(float64(most)*1000/float64(balance))/10, 'f', -1, 64), ".", ",", 1)
	return fmt.Sprintf("🎯 Valor de roubo: um !roubar leva no máximo %s%% do saldo (%s)", share, Coins(most))
}

func robberyText(r robbery, thief, victim types.JID) string {
	if r.wait > 0 {
		return fmt.Sprintf("🚓 @%s ainda tá sendo procurado pela polícia. Espera %s pra roubar de novo", thief.User, core.FormatWait(r.wait))
	}
	if r.poor {
		return fmt.Sprintf("🪹 @%s tem menos de %s, nem vale o risco", victim.User, Coins(robMinVictim))
	}
	if r.shield > 0 {
		return shieldedText(victim, r.shield)
	}
	if r.alarmed {
		return fmt.Sprintf("🚨 @%s tentou roubar @%s, mas o alarme disparou! O ladrão fugiu de mãos vazias e o alarme queimou", thief.User, victim.User)
	}
	if r.silent {
		return fmt.Sprintf("🔕 O alarme de @%s falhou e não disparou!\n", victim.User) + robberyText(robbery{success: r.success, amount: r.amount, jail: r.jail, jailFine: r.jailFine}, thief, victim)
	}
	if r.success {
		return fmt.Sprintf("🦹 @%s roubou %s de @%s e saiu correndo!", thief.User, Coins(r.amount), victim.User)
	}
	if r.jail > 0 {
		return fmt.Sprintf("🚔 @%s foi pego tentando roubar @%s e tava liso demais pra pagar a indenização, então foi direto pra cadeia por %s", thief.User, victim.User, core.FormatWait(r.jail)) + fineText(r.jailFine)
	}
	if r.amount == 0 {
		return fmt.Sprintf("🚔 @%s foi pego tentando roubar @%s, mas tava tão liso que saiu só com a vergonha", thief.User, victim.User)
	}
	return fmt.Sprintf("🚔 @%s foi pego tentando roubar @%s e pagou %s de indenização pra vítima", thief.User, victim.User, Coins(r.amount))
}

func (b *bot) transferir(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	mentions := core.MentionedJIDs(evt.Message)
	arg := core.FirstPlainArg(core.MessageText(evt.Message))
	if len(mentions) == 0 || arg == "" {
		return b.SendText(ctx, chat, "Use assim: !transferir @pessoa 100")
	}
	from, to := evt.Info.Sender.ToNonAD(), mentions[0].ToNonAD()
	fromAcc, toAcc := b.Account(ctx, from), b.Account(ctx, to)
	if fromAcc.Key == toAcc.Key {
		return b.SendText(ctx, chat, "Mandar moeda pra você mesmo não muda nada 🤨")
	}

	var (
		balance int64
		p       presidency
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { balance, _, err = b.Wallets.Balance(gctx, chat, fromAcc); return err })
	g.Go(func() (err error) { p, _, err = b.Wallets.presidency(gctx, chat, time.Now()); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler saldo: %w", err)
	}
	amount, ok := ParseStake(arg, balance)
	if !ok {
		return b.SendText(ctx, chat, "Não entendi o valor. Use assim: !transferir @pessoa 100 (ou metade, 10%, tudo)")
	}
	levy := p.levy(pixDecree, fromAcc, toAcc)
	people := append([]types.JID{from, to}, levy.payee()...)
	pix, err := b.Wallets.transferWithPix(ctx, chat, fromAcc, toAcc, amount, levy)
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, chat, "Você só tem "+Coins(balance))
	}
	if err != nil {
		return fmt.Errorf("transferir: %w", err)
	}
	_, err = b.SendMentions(ctx, chat, fmt.Sprintf("💸 @%s mandou %s pra @%s", from.User, Coins(amount), to.User)+pixText(pix, levy), people)
	return err
}

func (b *bot) ricos(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	wallets, err := b.Wallets.Richest(ctx, chat, RichestCount)
	if err != nil {
		return fmt.Errorf("ler ranking de saldo: %w", err)
	}
	if len(wallets) == 0 {
		return b.SendText(ctx, chat, "Ninguém tem moeda ainda 🪹 Usa !daily pra começar")
	}
	lines := []string{"🏆 *Os mais ricos do grupo*", ""}
	owners := make([]types.JID, len(wallets))
	for i, w := range wallets {
		lines = append(lines, fmt.Sprintf("%s %s@%s: %s", richPlace(i+1), titleBadge(w.title), w.Owner.User, Coins(w.Balance)))
		owners[i] = w.Owner
	}
	falls, err := b.Wallets.falls(ctx, chat, wallets)
	if err != nil {
		b.Client.Log.Warnf("ver quem caiu no !ricos de %s: %v", chat, err)
	}
	if len(falls) > 0 {
		lines = append(lines, "")
	}
	for _, f := range falls {
		lines = append(lines, f.text())
		owners = append(owners, f.who)
	}
	_, err = b.SendMentions(ctx, chat, strings.Join(lines, "\n"), owners)
	return err
}

func (b *bot) EconomyOn(ctx context.Context, chat types.JID) bool {
	on, err := b.Modes.Enabled(ctx, chat.ToNonAD(), core.ModeEconomy)
	if err != nil {
		b.Client.Log.Warnf("checar modo economia de %s: %v", chat, err)
	}
	return on
}

func (b *bot) accounts(ctx context.Context, people []types.JID) []core.Account {
	accounts := make([]core.Account, len(people))
	for i, p := range people {
		accounts[i] = b.Account(ctx, p)
	}
	return accounts
}

func (b *bot) Reward(ctx context.Context, chat types.JID, winners []types.JID, players int, amount int64) string {
	if len(winners) == 0 || !b.EconomyOn(ctx, chat) {
		return ""
	}
	if players < prizeMinPlayers {
		return fmt.Sprintf("🪹 Sem moedas dessa vez: o prêmio só sai com pelo menos %d pessoas jogando", prizeMinPlayers)
	}
	if err := b.Wallets.Credit(ctx, chat.ToNonAD(), b.accounts(ctx, winners), amount); err != nil {
		b.Client.Log.Errorf("pagar prêmio pra %s: %v", core.JoinMentions(winners), err)
		return ""
	}
	if len(winners) == 1 {
		return fmt.Sprintf("💰 @%s ganhou %s", winners[0].User, Coins(amount))
	}
	return fmt.Sprintf("💰 %s ganharam %s cada", core.JoinMentions(winners), Coins(amount))
}

func (b *bot) Charge(ctx context.Context, chat, payer types.JID, payees []types.JID, percent int64, label string) string {
	if len(payees) == 0 || !b.EconomyOn(ctx, chat) {
		return ""
	}
	payerAcc := b.Account(ctx, payer)
	job, err := b.ProfessionOf(ctx, chat.ToNonAD(), payerAcc)
	if err != nil {
		b.Client.Log.Errorf("ler profissão de %s pra cobrar %s: %v", payer, label, err)
	}
	percent = percent * (100 - job.legalDiscount) / 100
	each, err := b.Wallets.payShare(ctx, chat.ToNonAD(), payerAcc, b.accounts(ctx, payees), percent)
	if err != nil {
		b.Client.Log.Errorf("cobrar %s de %s: %v", label, payer, err)
		return ""
	}
	if each == 0 {
		return fmt.Sprintf("%s: @%s tá liso e não tem nem como pagar 🪹", label, payer.User)
	}
	receivers := core.JoinMentions(payees)
	if len(payees) > 1 {
		receivers = "cada um: " + receivers
	}
	text := fmt.Sprintf("%s de %d%% do saldo: @%s pagou %s pra %s", label, percent, payer.User, Coins(each), receivers)
	if job.legalDiscount > 0 {
		text += fmt.Sprintf(". Saiu %d%% mais barato porque @%s %s", job.legalDiscount, payer.User, job.title)
	}
	return text
}

func (p Profession) refund(stake, payout int64) int64 {
	if payout > 0 {
		return 0
	}
	return stake * p.cashback / 100
}

func ParseStake(arg string, balance int64) (int64, bool) {
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

func Coins(n int64) string {
	unit := "moedas"
	if n == 1 {
		unit = "moeda"
	}
	return "🪙 " + core.Thousands(n) + " " + unit
}

func (b *bot) economia(ctx context.Context, evt *events.Message) error {
	args := strings.Fields(core.MessageText(evt.Message))[1:]
	actions := map[string]func(context.Context, *events.Message, []string) error{
		"resetar":     b.resetEconomy,
		"vincular":    b.linkEconomy,
		"desvincular": b.unlinkEconomy,
		"vinculos":    b.showEconomyLinks,
	}
	if len(args) > 0 {
		if action, ok := actions[core.FoldLetters(args[0])]; ok {
			return action(ctx, evt, args[1:])
		}
	}
	return b.SendText(ctx, evt.Info.Chat.ToNonAD(), "Use assim: !economia resetar intervalos, !economia vincular <nome do grupo>, !economia desvincular ou !economia vinculos")
}

func (b *bot) resetEconomy(ctx context.Context, evt *events.Message, args []string) error {
	chat := evt.Info.Chat.ToNonAD()
	if len(args) == 0 {
		return b.SendText(ctx, chat, "Use assim: !economia resetar intervalos (ou !economia resetar intervalos @pessoa)")
	}
	if !slices.Contains([]string{"intervalos", "intervalo", "esperas"}, core.FoldLetters(args[0])) {
		return b.SendText(ctx, chat, "Só dá pra resetar os intervalos. Dinheiro, mochila e o resto ninguém reseta, nem o dono")
	}
	people := core.MentionedJIDs(evt.Message)
	accounts := make([]core.Account, len(people))
	for i, person := range people {
		accounts[i] = b.Account(ctx, person)
	}
	reset, err := b.Wallets.resetCooldowns(ctx, chat, accounts)
	if err != nil {
		return fmt.Errorf("resetar intervalos: %w", err)
	}
	if len(people) > 0 {
		_, err = b.SendMentions(ctx, chat, "⏱️ Intervalos zerados pra "+mentionList(accounts)+", quem tava preso, de refém ou morto tá livre e todo mundo tá com a saúde cheia. O dinheiro continua igualzinho", people)
		return err
	}
	return b.SendText(ctx, chat, fmt.Sprintf("⏱️ Intervalos zerados pra %d pessoa(s) desta conversa, quem tava preso, de refém ou morto tá livre e todo mundo tá com a saúde cheia. O dinheiro continua igualzinho", reset))
}

func (s *walletStore) resetCooldowns(ctx context.Context, chat types.JID, accounts []core.Account) (reset int64, err error) {
	columns := []string{
		"daily_em", "trabalho_em", "roubo_em", "profissao_em", "pesca_em", "mineracao_em", "lixao_em",
		"inseto_em", "furto_em", "cafeina_em", "prisao_em", "abuso_em", "processo_em", "fuga_em", "sequestro_em",
		"assalto_em", "denuncia_em", "assassinato_em", "libertacao_em", "abuso_trabalho_em", "abuso_roubo_em",
		"abuso_sequestro_em", "ressurreicao_em", "encanto_em", "disfarce_em", "preso_ate", "morto_ate", "crime_em",
		"saude_em", "conversao_em", "desvio_em", "postagem_em", "publi_em", "cratera_em", "escudo_ate", "ficha_em",
		"arremesso_em",
	}
	args := []any{chat.String()}
	for _, acc := range accounts {
		args = append(args, acc.Key)
	}
	where := func(column string) string {
		if len(accounts) == 0 {
			return " WHERE chat = ?"
		}
		return " WHERE chat = ? AND " + column + " IN (?" + strings.Repeat(", ?", len(accounts)-1) + ")"
	}
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		res, err := tx.ExecContext(ctx, "UPDATE carteiras SET "+strings.Join(columns, " = 0, ")+fmt.Sprintf(" = 0, saude = %d", MaxHealth)+where("pessoa"), args...)
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
