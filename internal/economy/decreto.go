package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	presidentTaxPercent   = 5
	presidentAid          = 500
	presidentBoxItem      = "caixa-misteriosa"
	presidentPardons      = 2
	presidentSeizePercent = 7
	workDecreeMaxCut      = 50
	workDecreeMaxRaise    = 20
	decreeSwing           = 50
	shopTaxDecreeMax      = 20
	pixDecreeMax          = 10
	betTaxDecreeMax       = 10
	casinoDecreeMaxRaise  = 20
	interestDecreeMax     = 10
	savingsTaxDecreeMax   = 50
	bailTaxDecreeMax      = 30
	incomeTaxDecreeMax    = 30
	stockDecreeSwing      = 30
	curfewPenalty         = 0.15
	termOverText          = "🏛️ O mandato acabou bem agora, então o decreto não valeu. Só quem ganhar a próxima eleição (!votar) baixa decreto de novo"
)

var errDecreeIdle = errors.New("decreto sem efeito")

type decrees map[string]int64

type percentDecree struct {
	names   []string
	column  string
	lo, hi  int64
	popular bool
	raised  string
	lowered string
	revoked string
	syntax  string
	about   string
}

type toggleDecree struct {
	names   []string
	column  string
	emoji   string
	effect  string
	revoked string
	syntax  string
	about   string
}

type onceDecree struct {
	names  []string
	column string
	limit  int64
	label  string
	spent  string
	syntax string
	about  string
}

type onceOutcome int

const (
	onceDone onceOutcome = iota
	onceSpent
	onceIdle
)

type decreeCall struct {
	chat    types.JID
	p       presidency
	args    []string
	targets []core.Account
	now     time.Time
}

type decree func(ctx context.Context, call decreeCall) (string, error)

type PixLevy struct {
	president core.Account
	percent   int64
}

var (
	workDecree = percentDecree{
		names: []string{"trabalho", "trabalhar", "trampo"}, column: "trabalho_pct", lo: -workDecreeMaxCut, hi: workDecreeMaxRaise,
		raised: "o !trabalhar espera %d%% mais", lowered: "o !trabalhar espera %d%% menos",
		revoked: "o !trabalhar volta à espera normal de cada profissão",
		syntax:  "!decreto trabalho -30", about: "muda a espera do !trabalhar de todo mundo",
	}
	payDecree = percentDecree{
		names: []string{"salario", "salarios", "pagamento"}, column: "salario_pct", lo: -decreeSwing, hi: decreeSwing, popular: true,
		raised: "o !trabalhar paga %d%% mais", lowered: "o !trabalhar paga %d%% menos",
		revoked: "o !trabalhar volta a pagar o normal de cada profissão",
		syntax:  "!decreto salario 20", about: "muda quanto o !trabalhar paga pra todo mundo",
	}
	dailyDecree = percentDecree{
		names: []string{"daily", "diario"}, column: "daily_pct", lo: -decreeSwing, hi: decreeSwing, popular: true,
		raised: "o !daily rende %d%% mais", lowered: "o !daily rende %d%% menos",
		revoked: "o !daily volta ao normal",
		syntax:  "!decreto daily 20", about: "muda quanto o !daily rende",
	}
	interestDecree = percentDecree{
		names: []string{"juros", "poupanca", "poupar"}, column: "juros_pct", lo: -savingsRate, hi: interestDecreeMax, popular: true,
		raised: "os juros da poupança sobem %d pontos", lowered: "os juros da poupança caem %d pontos",
		revoked: "a poupança volta aos juros normais",
		syntax:  "!decreto juros 3", about: fmt.Sprintf("soma ou tira pontos dos juros do !poupar: o normal é %d%%, e 3 faz render %d%%", savingsRate, savingsRate+3),
	}
	savingsTaxDecree = percentDecree{
		names: []string{"iof", "impostopoupanca", "impostojuros"}, column: "poupanca_imposto_pct", hi: savingsTaxDecreeMax,
		raised:  "a poupança paga %d%% dos juros de imposto no resgate, e ele vai pro presidente",
		revoked: "a poupança volta a render sem imposto",
		syntax:  "!decreto iof 20", about: "cobra imposto em cima dos juros do !poupar na hora do resgate, e ele vai pro bolso do presidente (dá pra mandar !decreto imposto poupanca também)",
	}
	incomeTaxDecree = percentDecree{
		names: []string{"renda", "impostorenda", "impostotrabalho", "impostotrampo", "impostosalario", "impostodaily", "impostodiario"}, column: "renda_pct", hi: incomeTaxDecreeMax,
		raised:  "o !trabalhar e o !daily pagam %d%% de imposto de renda, e ele vai pro presidente",
		revoked: "o !trabalhar e o !daily voltam a pagar sem imposto",
		syntax:  "!decreto renda 10", about: "cobra imposto em cima do que o !trabalhar e o !daily pagam, e ele vai pro bolso do presidente (dá pra mandar !decreto imposto trabalho ou !decreto imposto daily também)",
	}
	shopTaxDecree = percentDecree{
		names: []string{"loja", "compra", "compras", "impostocompra", "impostocompras", "impostoloja"}, column: "loja_pct", hi: shopTaxDecreeMax,
		raised:  "o !comprar cobra %d%% de imposto em cima da compra, e ele vai pro presidente",
		revoked: "o !comprar volta a cobrar sem imposto",
		syntax:  "!decreto loja 10", about: "cobra imposto em cima de cada !comprar da !loja e do !petshop, e ele vai pro bolso do presidente (dá pra mandar !decreto imposto compra também)",
	}
	pixDecree = percentDecree{
		names: []string{"pix", "transferencia", "taxa"}, column: "pix_pct", hi: pixDecreeMax,
		raised:  "o !transferir cobra mais %d%% de taxa de quem recebe, e ela vai pro presidente",
		revoked: "o !transferir volta à taxa normal",
		syntax:  "!decreto pix 5", about: "cobra uma taxa a mais de quem recebe no !transferir, e ela vai pro bolso do presidente",
	}
	sentenceDecree = percentDecree{
		names: []string{"pena", "cadeia", "prisao"}, column: "pena_pct", lo: -decreeSwing, hi: decreeSwing,
		raised: "a cadeia dura %d%% mais", lowered: "a cadeia dura %d%% menos",
		revoked: "a cadeia volta ao tempo normal",
		syntax:  "!decreto pena 20", about: "muda quanto tempo dura a cadeia de quem for preso",
	}
	bailDecree = percentDecree{
		names: []string{"fianca", "impostofianca", "taxafianca"}, column: "fianca_pct", hi: bailTaxDecreeMax,
		raised:  "o !fianca cobra %d%% de imposto em cima da fiança, e ele vai pro presidente",
		revoked: "o !fianca volta a cobrar sem imposto",
		syntax:  "!decreto fianca 20", about: "cobra imposto em cima de cada !fianca paga, e ele vai pro bolso do presidente (dá pra mandar !decreto imposto fianca também)",
	}
	gatherDecree = percentDecree{
		names: []string{"extracao", "pescar", "pesca", "minerar", "mineracao", "vasculhar", "catar", "cacar", "insetos", "coleta"}, column: "extracao_pct", lo: -decreeSwing, hi: decreeSwing,
		raised: "o !pescar, o !minerar, o !vasculhar, o !catar e os !empregados esperam %d%% mais", lowered: "o !pescar, o !minerar, o !vasculhar, o !catar e os !empregados esperam %d%% menos",
		revoked: "o !pescar, o !minerar, o !vasculhar, o !catar e os !empregados voltam à espera normal",
		syntax:  "!decreto extracao -30", about: "muda a espera do !pescar, do !minerar, do !vasculhar, do !catar e dos !empregados",
	}
	casinoDecree = percentDecree{
		names: []string{"cassino", "premio", "premios", "apostas", "aposta"}, column: "cassino_pct", lo: -betTaxDecreeMax, hi: casinoDecreeMaxRaise, popular: true,
		raised:  "o cassino paga %d%% mais nos prêmios",
		lowered: "o cassino cobra %d%% de taxa pro presidente: quem perde paga em cima da aposta e quem ganha paga em cima de tudo que levou",
		revoked: "o cassino volta ao normal, sem prêmio a mais nem taxa",
		syntax:  "!decreto cassino 20", about: "mexe no " + casinoGames + ": positivo aumenta o lucro de quem ganha e negativo cobra uma taxa que vai pro bolso do presidente, em que quem perde paga em cima da aposta e quem ganha paga em cima de tudo que levou. Vale o decreto da hora da aposta (dá pra mandar !decreto imposto aposta 5 também)",
	}
	percentDecrees = []percentDecree{workDecree, payDecree, dailyDecree, interestDecree, savingsTaxDecree, incomeTaxDecree, shopTaxDecree, pixDecree, sentenceDecree, bailDecree, gatherDecree, casinoDecree}

	curfewDecree = toggleDecree{
		names: []string{"recolher", "toque", "toquederecolher"}, column: "toque_recolher", emoji: "🌃",
		effect:  fmt.Sprintf("vale o toque de recolher (!roubar, !furtar, !sequestrar, !matar e !assalto com %d%% menos chance)", percent(curfewPenalty)),
		revoked: "acabou o toque de recolher 🌅",
		syntax:  "!decreto recolher", about: fmt.Sprintf("liga (ou desliga) o toque de recolher: !roubar, !furtar, !sequestrar, !matar e !assalto ficam com %d%% menos chance", percent(curfewPenalty)),
	}
	toggleDecrees = []toggleDecree{curfewDecree}

	taxOnce = onceDecree{
		names: []string{"imposto", "impostos", "taxar"}, column: "imposto_cobrado", limit: 1, label: "imposto",
		spent:  "💸 O imposto deste mandato já foi cobrado. O povo não aguenta mais",
		syntax: "!decreto imposto", about: fmt.Sprintf("cobra %d%% do saldo de cada um da conversa e põe no bolso do presidente", presidentTaxPercent),
	}
	aidOnce = onceDecree{
		names: []string{"auxilio", "bolsa", "esmola"}, column: "auxilio_feito", limit: 1, label: "auxílio",
		spent:  "💸 O auxílio deste mandato já saiu. Agora é cada um por si",
		syntax: "!decreto auxilio", about: fmt.Sprintf("dá %s de auxílio pra cada um da conversa, com moeda nova", Coins(presidentAid)),
	}
	boxOnce = onceDecree{
		names: []string{"caixa", "caixas", "caixamisteriosa", "presente", "presentes"}, column: "caixa_feita", limit: 1, label: "caixa misteriosa",
		spent:  "📦 As caixas misteriosas deste mandato já foram distribuídas. Acabou o estoque do Planalto",
		syntax: "!decreto caixa", about: "dá uma caixa misteriosa pra cada um da conversa, saída direto do almoxarifado do governo",
	}
	amnestyOnce = onceDecree{
		names: []string{"anistia"}, column: "anistia_feita", limit: 1, label: "anistia",
		spent:  "🕊️ A anistia deste mandato já foi dada",
		syntax: "!decreto anistia", about: "solta todo mundo da cadeia e limpa a ficha de crime da conversa toda",
	}
	pardonOnce = onceDecree{
		names: []string{"indulto", "indultar", "soltar"}, column: "indultos", limit: presidentPardons, label: "indulto",
		spent:  fmt.Sprintf("🕊️ Os %d indultos deste mandato já foram usados", presidentPardons),
		syntax: "!decreto indulto @pessoa", about: "tira a pessoa da cadeia na hora e limpa a ficha dela",
	}
	seizeOnce = onceDecree{
		names: []string{"confisco", "confiscar", "expropriar"}, column: "confisco_feito", limit: 1, label: "confisco",
		spent:  "🏛️ O confisco deste mandato já foi feito",
		syntax: "!decreto confisco @pessoa", about: fmt.Sprintf("tira %d%% do saldo da pessoa e põe no bolso do presidente", presidentSeizePercent),
	}
	stockOnce = onceDecree{
		names: []string{"acoes", "acao", "intervencao", "mercado"}, column: "intervencao_feita", limit: 1, label: "intervenção nas ações",
		spent:  "📈 A intervenção no mercado deste mandato já foi feita",
		syntax: "!decreto acoes FOFC3 20", about: fmt.Sprintf("sobe ou desce na hora o preço de uma ação, de -%d%% a +%d%% (depois o mercado volta a andar sozinho)", stockDecreeSwing, stockDecreeSwing),
	}
	onceDecrees = []onceDecree{taxOnce, aidOnce, boxOnce, amnestyOnce, pardonOnce, seizeOnce, stockOnce}
)

func decreeColumns() []string {
	var columns []string
	for _, d := range percentDecrees {
		columns = append(columns, d.column)
	}
	for _, d := range toggleDecrees {
		columns = append(columns, d.column)
	}
	for _, d := range onceDecrees {
		columns = append(columns, d.column)
	}
	return columns
}

func zipDecrees(columns []string, values []int64) decrees {
	d := make(decrees, len(columns))
	for i, column := range columns {
		d[column] = values[i]
	}
	return d
}

func (d decrees) level(pd percentDecree) int64 {
	return min(max(d[pd.column], pd.lo), pd.hi)
}

func (d decrees) on(td toggleDecree) bool {
	return d[td.column] != 0
}

func (d decrees) left(od onceDecree) int64 {
	return max(od.limit-d[od.column], 0)
}

func (d decrees) note(pd percentDecree) string {
	level := d.level(pd)
	if level == 0 {
		return ""
	}
	return "\n📜 Decreto do presidente: " + pd.effect(level)
}

func (d decrees) inForce() []string {
	var texts []string
	for _, pd := range percentDecrees {
		if level := d.level(pd); level != 0 {
			texts = append(texts, pd.effect(level))
		}
	}
	for _, td := range toggleDecrees {
		if d.on(td) {
			texts = append(texts, td.effect)
		}
	}
	return texts
}

func (d decrees) leftoverText() string {
	inForce := d.inForce()
	if len(inForce) == 0 {
		return ""
	}
	lines := []string{"", "", "📜 *Decretos do último presidente, que renunciou:* valem até alguém tomar posse, e os impostos deles agora somem em vez de ir pra alguém"}
	for _, decree := range inForce {
		lines = append(lines, "📜 "+decree)
	}
	return strings.Join(lines, "\n")
}

func (d decrees) onceLeftText() string {
	var parts []string
	for _, od := range onceDecrees {
		left := d.left(od)
		if left == 0 {
			continue
		}
		if od.limit == 1 {
			parts = append(parts, od.label)
			continue
		}
		parts = append(parts, fmt.Sprintf("%s (%d)", od.label, left))
	}
	if len(parts) == 0 {
		return "nenhum"
	}
	return core.JoinWithAnd(parts)
}

func (pd percentDecree) effect(level int64) string {
	if level < 0 {
		return fmt.Sprintf(pd.lowered, -level)
	}
	return fmt.Sprintf(pd.raised, level)
}

func (pd percentDecree) bounds() string {
	if pd.lo == 0 {
		return fmt.Sprintf("de 0 a %d%%", pd.hi)
	}
	return fmt.Sprintf("de %d%% a +%d%%", pd.lo, pd.hi)
}

func (pd percentDecree) parse(args []string) (int64, string) {
	return decreePercent(args, pd.lo, pd.hi)
}

func (pd percentDecree) mood(level int64) string {
	if (level > 0) == pd.popular {
		return "🎉"
	}
	return "😩"
}

func (od onceDecree) spentText(d decrees) string {
	return od.spent + "\nDecretos de uma vez que ainda sobram no mandato: " + d.onceLeftText()
}

func (od onceDecree) times() string {
	if od.limit == 1 {
		return "uma vez no mandato"
	}
	return fmt.Sprintf("%d vezes no mandato", od.limit)
}

func (p Profession) curfew() float64 {
	if !p.decrees.on(curfewDecree) {
		return 0
	}
	return curfewPenalty
}

const casinoGames = "!apostar, !cassino, !blackjack, !crash, !corrida e !roleta"

type PrizeBoost struct {
	decree  int64
	luck    int64
	rescued bool
}

func (pb PrizeBoost) Rescue(stake, payout int64) int64 {
	if !pb.rescued || payout >= stake {
		return 0
	}
	return stake - payout
}

type rescuedBet struct {
	acc   core.Account
	stake int64
}

func (b *bot) payRescues(ctx context.Context, chat types.JID, game string, bets []rescuedBet) {
	for _, bet := range bets {
		if err := b.Wallets.payPrize(ctx, chat, bet.acc, bet.stake, prizeLevy{}); err != nil {
			b.Client.Log.Errorf("devolver aposta salva do %s pra %s: %v", game, bet.acc.JID, err)
		}
	}
}

func savedBetNote(pb PrizeBoost) string {
	if !pb.rescued {
		return ""
	}
	return ", mas o Baralho do Zé Pelintra devolveu tudo 🃏"
}

func SavedBetText(saved int64) string {
	if saved == 0 {
		return ""
	}
	return "\n🃏 Mas o Baralho do Zé Pelintra devolveu " + Coins(saved)
}

func (pb PrizeBoost) Pay(stake, payout int64) int64 {
	if payout <= stake {
		return payout
	}
	return stake + (payout-stake)*(100+pb.decree+pb.luck)/100
}

func (pb PrizeBoost) Tag(stake, payout int64) string {
	if payout <= stake {
		return ""
	}
	var parts []string
	if pb.decree != 0 {
		parts = append(parts, fmt.Sprintf("📜 %+d%% por decreto", pb.decree))
	}
	if pb.luck != 0 {
		parts = append(parts, fmt.Sprintf("🃏 %+d%% de sorte", pb.luck))
	}
	if len(parts) == 0 {
		return ""
	}
	return " (" + strings.Join(parts, ", ") + ")"
}

func (p presidency) workCooldown(every time.Duration) time.Duration {
	return every * time.Duration(100+p.decrees.level(workDecree)) / 100
}

func (p presidency) payFor(s shift) shift {
	if !s.busted {
		s.amount += s.amount * p.decrees.level(payDecree) / 100
	}
	return s
}

func (p presidency) incomeLevy(acc core.Account) PixLevy {
	return p.levy(incomeTaxDecree, acc)
}

func (p presidency) casinoLevy(acc core.Account) PixLevy {
	levy := p.levy(casinoDecree, acc)
	levy.percent = max(-levy.percent, 0)
	return levy
}

func (p presidency) levy(pd percentDecree, payers ...core.Account) PixLevy {
	for _, payer := range payers {
		if payer.Key == p.president.Key {
			return PixLevy{}
		}
	}
	return PixLevy{president: p.president, percent: p.decrees.level(pd)}
}

func (b *bot) decreto(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	acc := b.Account(ctx, evt.Info.Sender.ToNonAD())
	now := time.Now()
	p, found, err := b.Wallets.presidency(ctx, chat, now)
	if err != nil {
		return fmt.Errorf("ler presidência: %w", err)
	}
	if !found {
		return b.SendText(ctx, chat, "🚫 Só o presidente baixa decreto, e a conversa tá sem presidente agora. Pra eleger um, manda !votar @pessoa")
	}
	if p.president.Key != acc.Key {
		text := fmt.Sprintf("🚫 Só o presidente baixa decreto, e quem tá no cargo é @%s. O mandato acaba daqui a %s", p.president.JID.User, core.FormatWait(p.until.Sub(now)))
		_, err := b.SendMentions(ctx, chat, text, []types.JID{p.president.JID})
		return err
	}
	args := strings.Fields(core.FoldLetters(core.MessageText(evt.Message)))[1:]
	if len(args) == 0 {
		return b.SendText(ctx, chat, decreesMenu())
	}
	run, rest, ok := pickDecree(b.decrees(), args)
	if !ok {
		return b.SendText(ctx, chat, fmt.Sprintf("🤔 Não existe decreto chamado \"%s\". Os que existem:\n\n%s", args[0], decreesMenu()))
	}
	var targets []core.Account
	people := []types.JID{p.president.JID}
	for _, jid := range core.MentionedJIDs(evt.Message) {
		target := b.Account(ctx, jid)
		targets = append(targets, target)
		people = append(people, target.JID)
	}
	text, err := run(ctx, decreeCall{chat: chat, p: p, args: rest, targets: targets, now: now})
	if err != nil {
		return err
	}
	_, err = b.SendMentions(ctx, chat, text, people)
	return err
}

func pickDecree(runs map[string]decree, args []string) (decree, []string, bool) {
	if len(args) > 1 {
		if run, ok := runs[args[0]+args[1]]; ok {
			return run, args[2:], true
		}
	}
	run, ok := runs[args[0]]
	return run, args[1:], ok
}

func (b *bot) decrees() map[string]decree {
	runs := map[string]decree{}
	for _, pd := range percentDecrees {
		run := b.percentDecreeRun(pd)
		for _, name := range pd.names {
			runs[name] = run
		}
	}
	taxed := taxedRun(runs[casinoDecree.names[0]])
	for _, name := range []string{"impostoaposta", "impostoapostas", "taxaaposta", "taxaapostas"} {
		runs[name] = taxed
	}
	for _, td := range toggleDecrees {
		run := b.toggleDecreeRun(td)
		for _, name := range td.names {
			runs[name] = run
		}
	}
	once := []struct {
		od  onceDecree
		run decree
	}{
		{taxOnce, b.taxDecree},
		{aidOnce, b.aidDecree},
		{boxOnce, b.boxDecree},
		{amnestyOnce, b.amnestyDecree},
		{pardonOnce, b.pardonDecree},
		{seizeOnce, b.seizeDecree},
		{stockOnce, b.stockDecree},
	}
	for _, o := range once {
		for _, name := range o.od.names {
			runs[name] = o.run
		}
	}
	return runs
}

func taxedRun(run decree) decree {
	return func(ctx context.Context, call decreeCall) (string, error) {
		if len(call.args) > 0 && !strings.HasPrefix(call.args[0], "-") {
			call.args = append([]string{"-" + call.args[0]}, call.args[1:]...)
		}
		return run(ctx, call)
	}
}

func percentDecreeHelp(revokeHint string) []core.Usage {
	var help []core.Usage
	for _, pd := range percentDecrees {
		help = append(help, core.Usage{pd.syntax, fmt.Sprintf("%s (%s%s)", pd.about, pd.bounds(), revokeHint)})
	}
	return help
}

func toggleDecreeHelp() []core.Usage {
	var help []core.Usage
	for _, td := range toggleDecrees {
		help = append(help, core.Usage{td.syntax, td.about})
	}
	return help
}

func onceDecreeHelp() []core.Usage {
	var help []core.Usage
	for _, od := range onceDecrees {
		help = append(help, core.Usage{od.syntax, od.about + ", " + od.times()})
	}
	return help
}

func decreeHelp() []core.Usage {
	return slices.Concat(percentDecreeHelp(", e 0 tira o decreto"), toggleDecreeHelp(), onceDecreeHelp())
}

func decreesMenu() string {
	sections := []struct {
		title string
		hint  string
		help  []core.Usage
	}{
		{"📊 POR PORCENTAGEM", "Ficam valendo até o fim do mandato. Manda 0 pra tirar o decreto", percentDecreeHelp("")},
		{"🔀 LIGA E DESLIGA", "Manda o mesmo decreto de novo pra desligar", toggleDecreeHelp()},
		{"⚡ DE UMA VEZ", "Têm efeito na hora e gastam um uso do mandato", onceDecreeHelp()},
	}
	var sb strings.Builder
	sb.WriteString("📜 *DECRETOS DO PRESIDENTE*")
	for _, s := range sections {
		fmt.Fprintf(&sb, "\n\n━━━━━━━━━━━━━━\n*%s*\n━━━━━━━━━━━━━━\n_%s_", s.title, s.hint)
		for _, u := range s.help {
			fmt.Fprintf(&sb, "\n\n▸ `%s`\n     _%s_", u.Syntax, u.Description)
		}
	}
	sb.WriteString("\n\n_Os decretos caem quando o mandato acaba_")
	return sb.String()
}

func decreePercent(args []string, lo, hi int64) (int64, string) {
	if len(args) == 0 {
		return 0, "Faltou o número"
	}
	pct, err := strconv.ParseInt(strings.TrimSuffix(args[0], "%"), 10, 64)
	if err != nil {
		return 0, fmt.Sprintf("\"%s\" não é número", args[0])
	}
	if pct < lo || pct > hi {
		return 0, fmt.Sprintf("%d%% passa do limite", pct)
	}
	return pct, ""
}

func (pd percentDecree) unchanged(level int64) string {
	if level == 0 {
		return fmt.Sprintf("🤷 Não tem nenhum !decreto %s em vigor, então não tem o que revogar", pd.names[0])
	}
	return "📜 Esse decreto já tá em vigor: " + pd.effect(level)
}

func (b *bot) percentDecreeRun(pd percentDecree) decree {
	return func(ctx context.Context, call decreeCall) (string, error) {
		level, problem := pd.parse(call.args)
		if problem != "" {
			return fmt.Sprintf("🤔 %s. O !decreto %s vai %s, e 0 tira o decreto. Exemplo: %s", problem, pd.names[0], pd.bounds(), pd.syntax), nil
		}
		if level == call.p.decrees.level(pd) {
			return pd.unchanged(level), nil
		}
		set, err := b.Wallets.setDecree(ctx, call.chat, call.p.president, pd.column, level, call.now)
		if err != nil {
			return "", fmt.Errorf("decreto %s: %w", pd.names[0], err)
		}
		if !set {
			return termOverText, nil
		}
		user := call.p.president.JID.User
		if level == 0 {
			return fmt.Sprintf("📜 @%s revogou o decreto: %s", user, pd.revoked), nil
		}
		return fmt.Sprintf("📜 *Decreto presidencial!* @%s decretou que %s até o fim do mandato %s", user, pd.effect(level), pd.mood(level)), nil
	}
}

func (b *bot) toggleDecreeRun(td toggleDecree) decree {
	return func(ctx context.Context, call decreeCall) (string, error) {
		on, set, err := b.Wallets.flipDecree(ctx, call.chat, call.p.president, td.column, call.now)
		if err != nil {
			return "", fmt.Errorf("decreto %s: %w", td.names[0], err)
		}
		if !set {
			return termOverText, nil
		}
		user := call.p.president.JID.User
		if !on {
			return fmt.Sprintf("📜 @%s revogou o decreto: %s", user, td.revoked), nil
		}
		return fmt.Sprintf("%s *Decreto presidencial!* @%s decretou que %s até o fim do mandato", td.emoji, user, td.effect), nil
	}
}

func (b *bot) spendDecree(ctx context.Context, call decreeCall, od onceDecree, then func(tx *ledgerTx) error) (onceOutcome, error) {
	if call.p.decrees.left(od) == 0 {
		return onceSpent, nil
	}
	spent, err := b.Wallets.spendDecree(ctx, call.chat, call.p.president, od, call.now, then)
	if errors.Is(err, errDecreeIdle) {
		return onceIdle, nil
	}
	if err != nil {
		return onceSpent, fmt.Errorf("decreto %s: %w", od.label, err)
	}
	if !spent {
		return onceSpent, nil
	}
	return onceDone, nil
}

func (b *bot) taxDecree(ctx context.Context, call decreeCall) (string, error) {
	var total, payers int64
	outcome, err := b.spendDecree(ctx, call, taxOnce, func(tx *ledgerTx) (err error) {
		total, payers, err = collectTax(ctx, tx, call.chat, call.p.president, presidentTaxPercent)
		if err != nil || total > 0 {
			return err
		}
		return errDecreeIdle
	})
	if err != nil {
		return "", err
	}
	user := call.p.president.JID.User
	return map[onceOutcome]string{
		onceSpent: taxOnce.spentText(call.p.decrees),
		onceIdle:  "💸 O povo tá tão liso que não tem de quem cobrar imposto. O imposto do mandato fica guardado pra depois",
		onceDone:  fmt.Sprintf("💸 *Imposto presidencial!* @%s cobrou %d%% do saldo de %s e embolsou %s", user, presidentTaxPercent, core.CountLabel(int(payers), "pessoa", "pessoas"), Coins(total)),
	}[outcome], nil
}

func (b *bot) aidDecree(ctx context.Context, call decreeCall) (string, error) {
	var paid int64
	outcome, err := b.spendDecree(ctx, call, aidOnce, func(tx *ledgerTx) error {
		res, err := tx.ExecContext(ctx,
			"UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa != ?",
			presidentAid, call.chat.String(), call.p.president.Key,
		)
		if err != nil {
			return err
		}
		if paid, err = res.RowsAffected(); err != nil || paid > 0 {
			return err
		}
		return errDecreeIdle
	})
	if err != nil {
		return "", err
	}
	user := call.p.president.JID.User
	return map[onceOutcome]string{
		onceSpent: aidOnce.spentText(call.p.decrees),
		onceIdle:  "💸 Não tem ninguém com carteira na conversa pra receber o auxílio. Fica guardado pra depois",
		onceDone:  fmt.Sprintf("💸 *Auxílio presidencial!* @%s deu %s pra %s. Moeda nova, saída direto da gráfica 🖨️", user, Coins(presidentAid), core.CountLabel(int(paid), "pessoa", "pessoas")),
	}[outcome], nil
}

func (b *bot) boxDecree(ctx context.Context, call decreeCall) (string, error) {
	var given int64
	outcome, err := b.spendDecree(ctx, call, boxOnce, func(tx *ledgerTx) error {
		res, err := tx.ExecContext(ctx,
			`INSERT INTO inventarios (chat, pessoa, item, quantidade)
			 SELECT chat, pessoa, ?, 1 FROM carteiras WHERE chat = ? AND pessoa != ?
			 ON CONFLICT (chat, pessoa, item, qualidade) DO UPDATE SET quantidade = inventarios.quantidade + excluded.quantidade`,
			presidentBoxItem, call.chat.String(), call.p.president.Key,
		)
		if err != nil {
			return err
		}
		if given, err = res.RowsAffected(); err != nil || given > 0 {
			return err
		}
		return errDecreeIdle
	})
	if err != nil {
		return "", err
	}
	box, _ := itemByID(presidentBoxItem)
	return map[onceOutcome]string{
		onceSpent: boxOnce.spentText(call.p.decrees),
		onceIdle:  "📦 Não tem ninguém com carteira na conversa pra receber caixa. As caixas ficam guardadas pra depois",
		onceDone: fmt.Sprintf("📦 *Caixas presidenciais!* @%s mandou uma %s pra %s. Tá na mochila, é só dar !abrir caixa 🎉",
			call.p.president.JID.User, box.label(), core.CountLabel(int(given), "pessoa", "pessoas")),
	}[outcome], nil
}

func (b *bot) amnestyDecree(ctx context.Context, call decreeCall) (string, error) {
	var freed, cleared int64
	outcome, err := b.spendDecree(ctx, call, amnestyOnce, func(tx *ledgerTx) (err error) {
		freed, cleared, err = grantAmnesty(ctx, tx, call.chat, call.now)
		if err != nil || freed+cleared > 0 {
			return err
		}
		return errDecreeIdle
	})
	if err != nil {
		return "", err
	}
	user := call.p.president.JID.User
	return map[onceOutcome]string{
		onceSpent: amnestyOnce.spentText(call.p.decrees),
		onceIdle:  "🕊️ Ninguém tá preso nem com a ficha suja. A anistia fica guardada pra depois",
		onceDone:  fmt.Sprintf("🕊️ *Anistia geral!* @%s soltou %s da cadeia e limpou a ficha de %s", user, core.CountLabel(int(freed), "pessoa", "pessoas"), core.CountLabel(int(cleared), "pessoa", "pessoas")),
	}[outcome], nil
}

func (b *bot) pardonDecree(ctx context.Context, call decreeCall) (string, error) {
	if len(call.targets) == 0 {
		return "Marca quem vai receber o indulto: !decreto indulto @pessoa", nil
	}
	acc := call.targets[0]
	outcome, err := b.spendDecree(ctx, call, pardonOnce, func(tx *ledgerTx) error {
		res, err := tx.ExecContext(ctx,
			"UPDATE carteiras SET preso_ate = 0, crime_em = 0 WHERE chat = ? AND pessoa = ? AND preso_ate > ?",
			call.chat.String(), acc.Key, call.now.Unix(),
		)
		if err != nil {
			return err
		}
		released, err := res.RowsAffected()
		if err != nil || released > 0 {
			return err
		}
		return errDecreeIdle
	})
	if err != nil {
		return "", err
	}
	left := call.p.decrees.left(pardonOnce) - 1
	return map[onceOutcome]string{
		onceSpent: pardonOnce.spentText(call.p.decrees),
		onceIdle:  fmt.Sprintf("🤨 @%s nem tá preso(a). Indulto de quê? O indulto não foi gasto", acc.JID.User),
		onceDone: fmt.Sprintf("🕊️ *Indulto presidencial!* @%s perdoou @%s, que saiu da cadeia agora mesmo e com a ficha limpa. Sobrou %s no mandato",
			call.p.president.JID.User, acc.JID.User, core.CountLabel(int(left), "indulto", "indultos")),
	}[outcome], nil
}

func (b *bot) seizeDecree(ctx context.Context, call decreeCall) (string, error) {
	if len(call.targets) == 0 {
		return "Marca de quem vai confiscar: !decreto confisco @pessoa", nil
	}
	acc := call.targets[0]
	if acc.Key == call.p.president.Key {
		return "Confiscar de si mesmo não muda nada 🤨", nil
	}
	var taken int64
	outcome, err := b.spendDecree(ctx, call, seizeOnce, func(tx *ledgerTx) error {
		err := tx.QueryRowContext(ctx,
			"SELECT saldo * ? / 100 FROM carteiras WHERE chat = ? AND pessoa = ?",
			presidentSeizePercent, call.chat.String(), acc.Key,
		).Scan(&taken)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && taken <= 0) {
			return errDecreeIdle
		}
		if err != nil {
			return err
		}
		return moveCoins(ctx, tx, call.chat, acc, call.p.president, taken)
	})
	if err != nil {
		return "", err
	}
	return map[onceOutcome]string{
		onceSpent: seizeOnce.spentText(call.p.decrees),
		onceIdle:  fmt.Sprintf("🪹 @%s tá liso(a), não tem o que confiscar. O confisco do mandato continua guardado", acc.JID.User),
		onceDone: fmt.Sprintf("🏛️ *Confisco presidencial!* @%s tomou %s de @%s (%d%% do saldo) e pôs no próprio bolso",
			call.p.president.JID.User, Coins(taken), acc.JID.User, presidentSeizePercent),
	}[outcome], nil
}

func (b *bot) stockDecree(ctx context.Context, call decreeCall) (string, error) {
	usageText := fmt.Sprintf("Dá pra mexer de -%d%% a +%d%% no preço. Exemplo: %s", stockDecreeSwing, stockDecreeSwing, stockOnce.syntax)
	if len(call.args) == 0 {
		return "🤔 Faltou dizer a ação. " + usageText, nil
	}
	company, found := findStockCompany(call.args[0])
	if !found {
		var tickers []string
		for _, c := range stockCompanies {
			tickers = append(tickers, c.ticker)
		}
		return fmt.Sprintf("🤔 Não achei a ação \"%s\". As que existem: %s", call.args[0], core.JoinWithAnd(tickers)), nil
	}
	pct, problem := decreePercent(call.args[1:], -stockDecreeSwing, stockDecreeSwing)
	if problem == "" && pct == 0 {
		problem = "0% não mexe em nada"
	}
	if problem != "" {
		return "🤔 " + problem + ". " + usageText, nil
	}
	var before, after int64
	outcome, err := b.spendDecree(ctx, call, stockOnce, func(tx *ledgerTx) (err error) {
		before, after, err = pushStock(ctx, tx, call.chat, company, pct, call.now)
		return err
	})
	if err != nil {
		return "", err
	}
	direction := "subir"
	if pct < 0 {
		direction = "descer"
	}
	return map[onceOutcome]string{
		onceSpent: stockOnce.spentText(call.p.decrees),
		onceDone: fmt.Sprintf("📈 *Intervenção no mercado!* @%s mandou a %s %s (%s) %s %d%% na canetada: foi de %s pra %s",
			call.p.president.JID.User, company.emoji, company.name, company.ticker, direction, max(pct, -pct), stockCents(before), stockCents(after)),
	}[outcome], nil
}

func grantAmnesty(ctx context.Context, tx *ledgerTx, chat types.JID, now time.Time) (freed, cleared int64, err error) {
	res, err := tx.ExecContext(ctx, "UPDATE carteiras SET preso_ate = 0 WHERE chat = ? AND preso_ate > ?", chat.String(), now.Unix())
	if err != nil {
		return 0, 0, err
	}
	if freed, err = res.RowsAffected(); err != nil {
		return 0, 0, err
	}
	res, err = tx.ExecContext(ctx, "UPDATE carteiras SET crime_em = 0 WHERE chat = ? AND crime_em > ?", chat.String(), now.Add(-crimeWindow).Unix())
	if err != nil {
		return 0, 0, err
	}
	cleared, err = res.RowsAffected()
	return freed, cleared, err
}

func pushStock(ctx context.Context, tx *ledgerTx, chat types.JID, company stockCompany, pct int64, now time.Time) (before, after int64, err error) {
	quotes, err := refreshStocks(ctx, tx, chat, now)
	if err != nil {
		return 0, 0, err
	}
	for _, q := range quotes {
		if q.company.ticker == company.ticker {
			before = q.cents
		}
	}
	after = max(before*(100+pct)/100, stockFloorCents)
	_, err = tx.ExecContext(ctx, "UPDATE acoes_precos SET centavos = ? WHERE chat = ? AND empresa = ?", after, chat.String(), company.ticker)
	return before, after, err
}

func collectTax(ctx context.Context, tx *ledgerTx, chat types.JID, president core.Account, pct int64) (total, payers int64, err error) {
	err = tx.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(saldo * ? / 100), 0), COUNT(*) FROM carteiras WHERE chat = ? AND pessoa != ? AND saldo * ? / 100 > 0",
		pct, chat.String(), president.Key, pct,
	).Scan(&total, &payers)
	if err != nil || total == 0 {
		return total, payers, err
	}
	_, err = tx.ExecContext(ctx,
		"UPDATE carteiras SET saldo = saldo - saldo * ? / 100 WHERE chat = ? AND pessoa != ? AND saldo > 0",
		pct, chat.String(), president.Key,
	)
	if err != nil {
		return 0, 0, err
	}
	_, err = tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", total, chat.String(), president.Key)
	return total, payers, err
}

func (s *walletStore) decrees(ctx context.Context, chat types.JID, now time.Time) (decrees, error) {
	columns := decreeColumns()
	values := make([]int64, len(columns))
	dest := make([]any, len(values))
	for i := range values {
		dest[i] = &values[i]
	}
	err := s.db.QueryRowContext(ctx,
		"SELECT "+strings.Join(columns, ", ")+" FROM presidencias WHERE chat = ? AND (mandato_ate > ? OR renunciou = 1)",
		chat.String(), now.Unix(),
	).Scan(dest...)
	if errors.Is(err, sql.ErrNoRows) {
		return decrees{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ler decretos: %w", err)
	}
	return zipDecrees(columns, values), nil
}

func (s *walletStore) setDecree(ctx context.Context, chat types.JID, president core.Account, column string, value int64, now time.Time) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		"UPDATE presidencias SET "+column+" = ? WHERE chat = ? AND pessoa = ? AND mandato_ate > ?",
		value, chat.String(), president.Key, now.Unix(),
	)
	if err != nil {
		return false, err
	}
	set, err := res.RowsAffected()
	return set > 0, err
}

func (s *walletStore) flipDecree(ctx context.Context, chat types.JID, president core.Account, column string, now time.Time) (on, set bool, err error) {
	var value int64
	err = s.db.QueryRowContext(ctx,
		"UPDATE presidencias SET "+column+" = 1 - MIN(MAX("+column+", 0), 1) WHERE chat = ? AND pessoa = ? AND mandato_ate > ? RETURNING "+column,
		chat.String(), president.Key, now.Unix(),
	).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return false, false, nil
	}
	return value != 0, err == nil, err
}

func (s *walletStore) spendDecree(ctx context.Context, chat types.JID, president core.Account, od onceDecree, now time.Time, then func(tx *ledgerTx) error) (spent bool, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		res, err := tx.ExecContext(ctx,
			"UPDATE presidencias SET "+od.column+" = "+od.column+" + 1 WHERE chat = ? AND pessoa = ? AND mandato_ate > ? AND "+od.column+" < ?",
			chat.String(), president.Key, now.Unix(), od.limit,
		)
		if err != nil {
			return err
		}
		marked, err := res.RowsAffected()
		if err != nil || marked == 0 {
			return err
		}
		spent = true
		return then(tx)
	})
	return spent, err
}

func (s *walletStore) transferWithPix(ctx context.Context, chat types.JID, from, to core.Account, amount int64, levy PixLevy) (pix int64, err error) {
	pix = amount * levy.percent / 100
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		if err := openWallet(ctx, tx, chat, to); err != nil {
			return err
		}
		if err := moveCoins(ctx, tx, chat, from, to, amount); err != nil {
			return err
		}
		if pix == 0 {
			return nil
		}
		if !levy.collected() {
			return debitBalance(ctx, tx, chat, to, pix)
		}
		return moveCoins(ctx, tx, chat, to, levy.president, pix)
	})
	return pix, err
}

func (l PixLevy) collected() bool {
	return l.president.Key != ""
}

func (l PixLevy) payee() []types.JID {
	if !l.collected() {
		return nil
	}
	return []types.JID{l.president.JID}
}

func (l PixLevy) pocket() string {
	if !l.collected() {
		return "pro ralo (o presidente renunciou)"
	}
	return "pro bolso de @" + l.president.JID.User
}

func (l PixLevy) office() string {
	if !l.collected() {
		return "pro ralo (o presidente renunciou)"
	}
	return "pro bolso do presidente"
}

func (l PixLevy) cut(amount int64) int64 {
	return amount * l.percent / 100
}

func (l PixLevy) onBet(stake, payout int64) int64 {
	if payout <= stake {
		return l.cut(stake)
	}
	return l.cut(payout)
}

type heldLevy struct {
	PixLevy
	paid int64
}

type heldLevySave struct {
	President core.StoredAccount `json:"president"`
	Percent   int64              `json:"percent"`
	Paid      int64              `json:"paid"`
}

type prizeLevy struct {
	PixLevy
	fee int64
}

func (w wager) held() heldLevy {
	return heldLevy{PixLevy: w.Levy, paid: w.Levy.cut(w.Stake)}
}

func (h heldLevy) saved() heldLevySave {
	return heldLevySave{President: core.StoreAccount(h.president), Percent: h.percent, Paid: h.paid}
}

func (s heldLevySave) held() heldLevy {
	return heldLevy{PixLevy: PixLevy{president: s.President.Account(), percent: s.Percent}, paid: s.Paid}
}

func (h heldLevy) onPrize(stake, payout int64) prizeLevy {
	if payout <= stake {
		return prizeLevy{PixLevy: h.PixLevy}
	}
	return prizeLevy{PixLevy: h.PixLevy, fee: max(h.cut(payout)-h.paid, 0)}
}

func (p prizeLevy) text() string {
	if p.fee == 0 {
		return ""
	}
	return fmt.Sprintf(" (📜 menos %s de taxa %s, por decreto)", Coins(p.fee), p.office())
}

func (s *walletStore) payPrize(ctx context.Context, chat types.JID, acc core.Account, payout int64, levy prizeLevy) error {
	return s.inTx(ctx, func(tx *ledgerTx) error {
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", payout-levy.fee, chat.String(), acc.Key)
		if err != nil || levy.fee == 0 || !levy.collected() {
			return err
		}
		if err := openWallet(ctx, tx, chat, levy.president); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", levy.fee, chat.String(), levy.president.Key)
		return err
	})
}

func (s *walletStore) SettleBet(ctx context.Context, chat types.JID, acc core.Account, stake, payout int64, levy PixLevy) (balance int64, err error) {
	due := stake + levy.cut(stake)
	fee := levy.onBet(stake, payout)
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		err := tx.QueryRowContext(ctx,
			"UPDATE carteiras SET saldo = saldo - ? + ? WHERE chat = ? AND pessoa = ? AND saldo >= ? RETURNING saldo",
			stake+fee, payout, chat.String(), acc.Key, due,
		).Scan(&balance)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoFunds
		}
		if err != nil {
			return err
		}
		return payLevy(ctx, tx, chat, levy, fee)
	})
	return balance, err
}

func payLevy(ctx context.Context, tx *ledgerTx, chat types.JID, levy PixLevy, fee int64) error {
	if fee == 0 || !levy.collected() {
		return nil
	}
	if err := openWallet(ctx, tx, chat, levy.president); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", fee, chat.String(), levy.president.Key)
	return err
}

func incomeTaxText(tax int64, levy PixLevy) string {
	if tax == 0 {
		return ""
	}
	return fmt.Sprintf("\n📜 Desse valor, %s (%d%%) foram de imposto de renda %s, por decreto", Coins(tax), levy.percent, levy.pocket())
}

func BetLevyText(stake, payout int64, levy PixLevy) string {
	fee := levy.onBet(stake, payout)
	if fee == 0 {
		return ""
	}
	if payout > stake {
		return fmt.Sprintf("\n📜 Desse prêmio, %s (%d%% de tudo que levou) foram de taxa %s, por decreto", Coins(fee), levy.percent, levy.office())
	}
	return fmt.Sprintf("\n📜 E mais %s (%d%%) de taxa da aposta foram %s, por decreto", Coins(fee), levy.percent, levy.office())
}

func pixText(pix int64, levy PixLevy) string {
	if pix == 0 {
		return ""
	}
	return fmt.Sprintf("\n📜 E mais %s (%d%%) de taxa do pix foram %s, por decreto", Coins(pix), levy.percent, levy.pocket())
}
