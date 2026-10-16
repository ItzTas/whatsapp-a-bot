package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	dailyMin         = 150
	dailyMax         = 300
	workMin          = 30
	workMax          = 120
	robEvery         = 2 * time.Hour
	robMinVictim     = 50
	robSuccessChance = 0.4
	quizReward       = 50
	richestCount     = 10
)

var (
	dailyCooldown = cooldown{column: "daily_em", every: 20 * time.Hour}
	workCooldown  = cooldown{column: "trabalho_em", every: time.Hour}
)

func (b *bot) account(ctx context.Context, person types.JID) account {
	person = person.ToNonAD()
	return account{key: slices.Min(b.userAliases(ctx, person)), jid: person}
}

func (b *bot) saldo(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := firstMentionOrSender(evt).ToNonAD()
	balance, rank, err := b.wallets.balance(ctx, chat, b.account(ctx, person))
	if err != nil {
		return fmt.Errorf("ler saldo: %w", err)
	}
	note := fmt.Sprintf("🏆 %dº mais rico do grupo", rank)
	if balance == 0 {
		note = "_Tá liso. Usa !daily e !trabalhar pra começar_"
	}
	text := fmt.Sprintf("💰 @%s tem %s\n%s", person.User, coins(balance), note)
	_, err = b.sendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) daily(ctx context.Context, evt *events.Message) error {
	amount := dailyMin + rand.Int64N(dailyMax-dailyMin+1)
	return b.claimCoins(ctx, evt, dailyCooldown, amount,
		"🎁 @%s pegou o daily e ganhou %s",
		"⏳ @%s já pegou o daily hoje. Volta em %s")
}

func (b *bot) trabalhar(ctx context.Context, evt *events.Message) error {
	jobs := []string{
		"vendendo bala no farol", "de motoboy do iFood debaixo de chuva", "de pedreiro na obra do vizinho",
		"de streamer com 2 viewers (um era a mãe)", "de coach quântico", "de flanelinha no estacionamento do mercado",
		"de estagiário não remunerado, mas o chefe pagou um pastel", "vendendo pack do pé", "de DJ em festa de família",
		"de testador de colchão", "consertando o wifi da vizinha", "de figurante de novela", "de garçom em casamento",
		"vendendo curso de como ficar rico", "de palhaço em festa infantil", "de detetive de traição",
		"de cuidador de cachorro de madame", "cantando no ônibus", "de lavador de carro", "de influencer de cripto",
	}
	amount := workMin + rand.Int64N(workMax-workMin+1)
	return b.claimCoins(ctx, evt, workCooldown, amount,
		"💼 @%s trabalhou "+jobs[rand.IntN(len(jobs))]+" e ganhou %s",
		"😮‍💨 @%s ainda tá cansado do último trampo. Volta em %s")
}

func (b *bot) claimCoins(ctx context.Context, evt *events.Message, cd cooldown, amount int64, done, waiting string) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	wait, err := b.wallets.claim(ctx, chat, b.account(ctx, person), cd, amount, time.Now())
	if err != nil {
		return fmt.Errorf("pegar moedas: %w", err)
	}
	text := fmt.Sprintf(done, person.User, coins(amount))
	if wait > 0 {
		text = fmt.Sprintf(waiting, person.User, formatWait(wait))
	}
	_, err = b.sendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) apostar(ctx context.Context, evt *events.Message) error {
	return b.gamble(ctx, evt, "!apostar", func(stake int64) (int64, string) {
		if rand.IntN(2) == 0 {
			return 0, "🪙 Deu coroa! @%s perdeu %s"
		}
		return 2 * stake, "🪙 Deu cara! @%s ganhou %s"
	})
}

func (b *bot) cassino(ctx context.Context, evt *events.Message) error {
	return b.gamble(ctx, evt, "!cassino", func(stake int64) (int64, string) {
		symbols := []string{"🍒", "🍋", "🍇", "🔔", "⭐", "💎"}
		reels := make([]string, 3)
		for i := range reels {
			reels[i] = symbols[rand.IntN(len(symbols))]
		}
		machine := "🎰 [ " + strings.Join(reels, " | ") + " ]\n\n"
		distinct := len(slices.Compact(slices.Sorted(slices.Values(reels))))
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

func (b *bot) gamble(ctx context.Context, evt *events.Message, syntax string, play func(stake int64) (payout int64, result string)) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.account(ctx, person)
	balance, _, err := b.wallets.balance(ctx, chat, acc)
	if err != nil {
		return fmt.Errorf("ler saldo: %w", err)
	}
	if balance == 0 {
		return b.sendText(ctx, chat, "Você tá liso 🪹 Usa !daily e !trabalhar pra ter o que apostar")
	}
	stake, ok := parseStake(firstPlainArg(messageText(evt.Message)), balance)
	if !ok {
		return b.sendText(ctx, chat, fmt.Sprintf("Diz quanto apostar: %[1]s 100, %[1]s metade, %[1]s 10%% ou %[1]s tudo", syntax))
	}
	if stake > balance {
		return b.sendText(ctx, chat, "Você só tem "+coins(balance))
	}

	payout, result := play(stake)
	balance, err = b.wallets.settle(ctx, chat, acc, stake, payout)
	if errors.Is(err, errNoFunds) {
		return b.sendText(ctx, chat, "Seu saldo mudou no meio da aposta e não dá mais 🤨 Confere no !saldo")
	}
	if err != nil {
		return fmt.Errorf("pagar aposta: %w", err)
	}
	won := max(payout-stake, stake-payout)
	text := fmt.Sprintf(result, person.User, coins(won)) + "\n💰 Saldo: " + coins(balance)
	_, err = b.sendMentions(ctx, chat, text, []types.JID{person})
	return err
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

	r, err := b.wallets.rob(ctx, chat, thiefAcc, victimAcc, robEvery, time.Now(), robMinVictim, robOutcome)
	if err != nil {
		return fmt.Errorf("roubar: %w", err)
	}
	_, err = b.sendMentions(ctx, chat, robberyText(r, thief, victim), []types.JID{thief, victim})
	return err
}

func robOutcome(thiefBalance, victimBalance int64) (bool, int64) {
	if rand.Float64() < robSuccessChance {
		return true, victimBalance * (10 + rand.Int64N(21)) / 100
	}
	return false, thiefBalance * (10 + rand.Int64N(11)) / 100
}

func robberyText(r robbery, thief, victim types.JID) string {
	if r.wait > 0 {
		return fmt.Sprintf("🚓 @%s ainda tá sendo procurado pela polícia. Espera %s pra roubar de novo", thief.User, formatWait(r.wait))
	}
	if r.poor {
		return fmt.Sprintf("🪹 @%s tem menos de %s, nem vale o risco", victim.User, coins(robMinVictim))
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

	balance, _, err := b.wallets.balance(ctx, chat, fromAcc)
	if err != nil {
		return fmt.Errorf("ler saldo: %w", err)
	}
	amount, ok := parseStake(arg, balance)
	if !ok {
		return b.sendText(ctx, chat, "Não entendi o valor. Use assim: !transferir @pessoa 100 (ou metade, 10%, tudo)")
	}
	err = b.wallets.transfer(ctx, chat, fromAcc, toAcc, amount)
	if errors.Is(err, errNoFunds) {
		return b.sendText(ctx, chat, "Você só tem "+coins(balance))
	}
	if err != nil {
		return fmt.Errorf("transferir: %w", err)
	}
	_, err = b.sendMentions(ctx, chat, fmt.Sprintf("💸 @%s mandou %s pra @%s", from.User, coins(amount), to.User), []types.JID{from, to})
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
		lines = append(lines, fmt.Sprintf("%s @%s: %s", place, w.owner.User, coins(w.balance)))
		owners[i] = w.owner
	}
	_, err = b.sendMentions(ctx, chat, strings.Join(lines, "\n"), owners)
	return err
}

func (b *bot) rewardQuiz(ctx context.Context, chat types.JID, winners []types.JID) string {
	if len(winners) == 0 {
		return ""
	}
	on, err := b.modes.enabled(ctx, chat.ToNonAD(), modeEconomy)
	if err != nil {
		b.client.Log.Warnf("checar modo economia de %s: %v", chat, err)
		return ""
	}
	if !on {
		return ""
	}
	accounts := make([]account, len(winners))
	for i, w := range winners {
		accounts[i] = b.account(ctx, w)
	}
	if err := b.wallets.credit(ctx, chat.ToNonAD(), accounts, quizReward); err != nil {
		b.client.Log.Errorf("pagar prêmio do !quiz: %v", err)
		return ""
	}
	return "💰 Quem acertou ganhou " + coins(quizReward)
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
