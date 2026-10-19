package economy

import (
	"context"
	"fmt"
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

const professionChangeEvery = 90 * time.Minute

const (
	areaBicos = iota
	areaObras
	areaTransporte
	areaComida
	areaEscritorio
	areaSaude
	areaArte
	areaCampo
	areaComercio
	areaSeguranca
	areaCrime
	areaServicos
	areaFe
)

type area struct {
	key   string
	name  string
	emoji string
	finds []string
}

var areas = [...]area{
	areaBicos: {"bicos", "Bicos", "🎲",
		[]string{"pochete", "dinheiro-de-brinquedo", "bilhete-esquecido", "raspadinha", "tazo", "pulseira-da-amizade", "cupom-vencido"}},
	areaObras: {"obras", "Obras", "🔨",
		[]string{"caixa-de-ferramentas", "tijolo", "pedra", "areia", "ferro", "cobre", "moeda-do-imperio", "marmita"}},
	areaTransporte: {"transporte", "Transporte", "🚚",
		[]string{"gps", "multa", "carteira-perdida", "guarda-chuva", "celular-afogado", "fone", "mala", "gorjeta"}},
	areaComida: {"comida", "Comida", "🍳",
		[]string{"panela-de-ferro", "vale-refeicao", "gorjeta", "coxinha", "pastel", "pudim", "picanha", "marmita", "pao-de-santo-antonio"}},
	areaEscritorio: {"escritorio", "Escritório", "💻",
		[]string{"teclado-mecanico", "grampeador", "pendrive", "vale-refeicao", "cabo-usb", "cafe", "notebook-positivo"}},
	areaSaude: {"saude", "Saúde", "🏥",
		[]string{"estetoscopio", "atestado", "pirulito", "dentadura", "terco", "cafe"}},
	areaArte: {"arte", "Arte e mídia", "🎭",
		[]string{"ring-light", "ingresso-de-camarote", "autografo", "fita-k7", "disco-de-vinil", "oculos-juliet", "pandeiro"}},
	areaCampo: {"campo", "Campo e natureza", "🌾",
		[]string{"chapeu-de-palha", "saco-de-laranja", "favo-de-mel", "trevo", "isca", "sal-grosso", "pamonha", "arruda", "semente-de-roma"}},
	areaComercio: {"comercio", "Comércio", "🛍️",
		[]string{"maquininha", "mercadoria-encalhada", "cupom-vencido", "rolex-do-paraguai", "perfume", "nota-de-tres", "cd-pirata"}},
	areaSeguranca: {"lei", "Lei e segurança", "🚔",
		[]string{"radio-comunicador", "algema", lockItemID, alarmItemID, "carteira-perdida", "celular-afogado", "cafe"}},
	areaCrime: {"crime", "Crime", "🦹",
		[]string{"touca-ninja", "documento-falso", "mascara", "pe-de-cabra", "planta-do-cofre", "carteira-perdida", "rolex-do-paraguai"}},
	areaServicos: {"servicos", "Serviços", "🧹",
		[]string{"kit-de-limpeza", "gorjeta", "brinco-de-ouro", "anel-de-latinha", "tampinha", "pilha-usada", "carteira-perdida"}},
	areaFe: {"fe", "Fé e feitiçaria", "🙏",
		[]string{"vela-de-sete-dias", "agua-benta", "terco", "olho-grego", "sal-grosso", "espada-de-sao-jorge", "quartzo",
			"arruda", "fita-do-bonfim", "defumador", "pao-de-santo-antonio"}},
}

type peak struct {
	from  int
	to    int
	bonus int64
}

func (p peak) active(now time.Time) bool {
	if p.bonus == 0 {
		return false
	}
	hour := now.In(lotteryZone).Hour()
	if p.from < p.to {
		return hour >= p.from && hour < p.to
	}
	return hour >= p.from || hour < p.to
}

type craft struct {
	activity  string
	faster    int64
	luck      int
	fewerJunk int64
	grade     int
	noTool    bool
	noBreak   bool
}

func (c craft) apply(a activity) activity {
	if c.activity != a.command {
		return a
	}
	a.cooldown.every = a.cooldown.every * time.Duration(100-c.faster) / 100
	a.luck = c.luck
	a.gradeLuck = c.grade
	if c.fewerJunk > 0 {
		a.loot = trimJunk(a.loot, c.fewerJunk)
	}
	if c.noTool {
		a.tool = ""
	}
	if c.noBreak {
		a.breaks = 0
	}
	return a
}

func (c craft) describe() string {
	if c.activity == "" {
		return ""
	}
	a := activityNamed(c.activity)
	tool, _ := itemByID(a.tool)
	parts := []struct {
		on   bool
		text string
	}{
		{c.faster > 0, fmt.Sprintf("espera %d%% menos", c.faster)},
		{c.luck > 0, fmt.Sprintf("tem %dx mais sorte no que acha", c.luck+1)},
		{c.fewerJunk > 0, fmt.Sprintf("vem %d%% menos item comum e mão vazia", c.fewerJunk)},
		{c.grade > 0, fmt.Sprintf("o que acha vem com qualidade melhor: fica com a melhor de %d", c.grade+1)},
		{c.grade < 0, fmt.Sprintf("o que acha vem com qualidade pior: fica com a pior de %d", -c.grade+1)},
		{c.noTool && tool.id != "", "nem precisa de " + tool.label()},
		{c.noBreak && !c.noTool && tool.id != "", tool.label() + " nunca quebra"},
	}
	var found []string
	for _, p := range parts {
		if p.on {
			found = append(found, p.text)
		}
	}
	return fmt.Sprintf("%s No %s: %s", a.emoji, a.command, core.JoinWithAnd(found))
}

type Profession struct {
	name    string
	emoji   string
	title   string
	area    int
	aliases []string
	min     int64
	max     int64
	every   time.Duration
	bust    float64
	tasks   []string
	busts   []string

	bustFine     int64
	doubleChance float64
	relicDouble  float64
	noDouble     bool
	finds        []string
	findChance   float64
	findBonus    float64
	fixedFinds   bool
	equipped     bool
	peak         peak
	hazard       hazard
	safety       int64

	robEvery   time.Duration
	robBonus   float64
	robHaul    int64
	guard      float64
	cordon     float64
	theftEvery time.Duration
	theftBonus float64

	kidnapEvery   time.Duration
	kidnapBonus   float64
	ransomBonus   int64
	breakoutBonus float64

	heistBonus float64

	reach    int64
	stage    string
	fans     audience
	charming bool

	jailTime      time.Duration
	arrestEvery   time.Duration
	abuses        []abuseKind
	jailFactor    int64
	escapeBonus   float64
	releaseChance float64
	elusive       bool

	reportBonus float64
	noSnitch    bool

	killChance   float64
	reaper       bool
	posse        bool
	reviveChance float64
	soulPact     bool

	president   bool
	divine      bool
	estate      int
	warded      bool
	cursed      bool
	weapon      string
	drug        string
	charms      []string
	luxuries    []string
	implants    []string
	unpowered   []string
	sealedBody  bool
	titleID     string
	church      string
	doctrineID  string
	templeLevel int
	churchTithe int64

	craft        craft
	gatherFaster int64
	haste        int64
	compassLure  int
	rotten       bool
	barrenHauls  int64
	gatherGrade  int
	gatherLuck   int
	gatherRare   int64
	gatherJunk   int64
	boxLuck      int
	faith        faith
	mirror       bool
	oracle       bool

	salePercent   int64
	discount      int64
	interest      int64
	savingsRoom   int64
	savingsFaster int64
	stockProfit   int64
	dailyBonus    int64
	cashback      int64
	betJinx       float64
	jinxProof     bool
	casinoLuck    int64
	rescueOdds    float64
	animalLuck    int64
	legalDiscount int64
	trialDiscount int64
	trialBonus    int64
	trialEdge     int
	voteWeight    int
	trialEvery    time.Duration
	freeBrokerage bool

	lendRate     int64
	lendLateRate int64
	collectBonus float64
	collectEvery time.Duration
	collectVotes int
	garnish      int64

	decrees decrees
}

type shift struct {
	amount  int64
	task    string
	busted  bool
	doubled bool
	peaked  bool
	found   item
	wound   wound
}

func professionNamed(name string) (Profession, bool) {
	if name == god.name {
		return god, true
	}
	i := slices.IndexFunc(professions, func(p Profession) bool { return p.name == name })
	if i < 0 {
		return professions[0], false
	}
	return professions[i], true
}

func (p Profession) matchKeys() []string {
	keys := []string{normalizeItemName(p.name)}
	for _, alias := range p.aliases {
		keys = append(keys, normalizeItemName(alias))
	}
	return keys
}

func findProfession(query string) (Profession, bool) {
	q := normalizeItemName(query)
	if q == "" {
		return Profession{}, false
	}
	i := slices.IndexFunc(professions, func(p Profession) bool { return slices.Contains(p.matchKeys(), q) })
	if i >= 0 {
		return professions[i], true
	}
	prefixed := slices.DeleteFunc(slices.Clone(professions), func(p Profession) bool {
		return !slices.ContainsFunc(p.matchKeys(), func(key string) bool { return strings.HasPrefix(key, q) })
	})
	if len(prefixed) != 1 {
		return Profession{}, false
	}
	return prefixed[0], true
}

func findArea(query string) (int, bool) {
	q := normalizeItemName(query)
	i := slices.IndexFunc(areas[:], func(a area) bool {
		return q == a.key || q == normalizeItemName(a.name)
	})
	return i, i >= 0
}

func (p Profession) work(now time.Time) shift {
	if rand.Float64() < p.bust {
		s := shift{task: core.PickOne(p.busts), busted: true}
		if p.bustFine > 0 {
			s.amount = -(p.bustFine/2 + rand.Int64N(p.bustFine/2+1))
		}
		return s
	}
	s := shift{amount: p.min + rand.Int64N(p.max-p.min+1), task: core.PickOne(p.tasks)}
	if p.peak.active(now) {
		s.amount += s.amount * p.peak.bonus / 100
		s.peaked = true
	}
	if rand.Float64() < p.doubling() {
		s.amount *= 2
		s.doubled = true
	}
	if finds, chance := p.loot(); rand.Float64() < chance {
		s.found, _ = itemByID(core.PickOne(finds))
	}
	return s
}

func (s shift) text(job Profession, user string, paid int64) string {
	if s.busted && paid < 0 {
		return fmt.Sprintf("%s @%s %s e ainda perdeu %s", job.emoji, user, s.task, Coins(-paid))
	}
	if s.busted {
		return fmt.Sprintf("%s @%s %s e não ganhou nada", job.emoji, user, s.task)
	}
	lines := []string{fmt.Sprintf("%s @%s %s e ganhou %s", job.emoji, user, s.task, Coins(paid))}
	extras := []struct {
		on   bool
		text string
	}{
		{s.peaked, fmt.Sprintf("⏰ Horário de pico: +%d%%", job.peak.bonus)},
		{s.doubled, "🍀 Dia de sorte: pagaram em dobro!"},
		{s.found.id != "", "🎁 No caminho ainda achou " + s.found.label()},
		{job.suits(s.found) && !job.equipped, fmt.Sprintf("🛠️ É o instrumento de trabalho da sua área: com ele na mochila, o !trabalhar paga %d%% a mais", tradePayBonus)},
	}
	for _, e := range extras {
		if e.on {
			lines = append(lines, e.text)
		}
	}
	return strings.Join(lines, "\n")
}

func (p Profession) robCooldown() cooldown {
	if p.robEvery == 0 {
		return p.pace(cooldown{column: "roubo_em", every: robEvery})
	}
	return p.pace(cooldown{column: "roubo_em", every: p.robEvery})
}

func (p Profession) theftCooldown() cooldown {
	if p.theftEvery == 0 {
		return p.pace(theftCooldown)
	}
	return p.pace(cooldown{column: theftCooldown.column, every: p.theftEvery})
}

func (p Profession) trialCooldown() cooldown {
	if p.trialEvery == 0 {
		return p.pace(cooldown{column: trialCooldownColumn, every: trialEvery})
	}
	return p.pace(cooldown{column: trialCooldownColumn, every: p.trialEvery})
}

func (p Profession) marketPercent() int64 {
	if p.salePercent == 0 {
		return marketSalePercent
	}
	return p.salePercent
}

func (p Profession) shopPrice(cost int64) int64 {
	return cost * (100 - p.discount) / 100
}

func (p Profession) doubling() float64 {
	if p.noDouble {
		return 0
	}
	return p.doubleChance + p.relicDouble
}

func clampChance(c float64) float64 {
	return min(max(c, 0.05), 0.95)
}

func fineChance(f float64) string {
	return strings.ReplaceAll(strconv.FormatFloat(f*100, 'f', -1, 64), ".", ",")
}

func percent(f float64) int {
	return int(f*100 + 0.5)
}

func (p Profession) earnings() string {
	return fmt.Sprintf("💰 Ganha de %s a %s a cada %s", core.Thousands(p.min), core.Thousands(p.max), core.FormatWait(p.every))
}

func findsText(ids []string) string {
	finds := make([]string, len(ids))
	for i, id := range ids {
		it, _ := itemByID(id)
		finds[i] = it.label()
	}
	return core.JoinWithAnd(finds)
}

func (p Profession) perks() []string {
	if p.divine {
		return divinePerks()
	}
	finds, findChance := p.loot()
	perks := []struct {
		on   bool
		text string
	}{
		{p.president, fmt.Sprintf("🏛️ É presidente da conversa: segurança contra !sequestrar e !matar, imunidade que segura %d%% das prisões, pode !prender, !libertar e !reviver, abusa do poder %s no mandato no !prender, !trabalhar, !roubar e !sequestrar e baixa !decreto (veja !presidente)", percent(presidentImmunity), core.CountLabel(presidentAbuses, "vez", "vezes"))},
		{p.bust > 0 && p.bustFine == 0, fmt.Sprintf("😬 Em %d%% das vezes dá ruim e não ganha nada", percent(p.bust))},
		{p.bust > 0 && p.bustFine > 0, fmt.Sprintf("💥 Em %d%% das vezes dá ruim e ainda perde até %s", percent(p.bust), Coins(p.bustFine))},
		{p.hazard.harm > 0, hazardPerk(p.hazard, p.safety)},
		{p.safety > 0 && p.hazard.harm > 0, fmt.Sprintf("🦺 Se machuca %d%% menos no !trabalhar", min(p.safety, maxSafety))},
		{p.peak.bonus > 0, fmt.Sprintf("⏰ Ganha +%d%% entre %dh e %dh", p.peak.bonus, p.peak.from, p.peak.to)},
		{p.doubling() > 0, fmt.Sprintf("🍀 %d%% de chance de ganhar em dobro", percent(p.doubling()))},
		{findChance > 0 && !p.fixedFinds, fmt.Sprintf("🎁 %d%% de chance de achar um item trabalhando: %s", percent(findChance), findsText(finds))},
		{p.fixedFinds, fmt.Sprintf("🎁 Trabalhando não acha nada, só tem %s%% de chance de achar %s, e nada aumenta essa chance", fineChance(findChance), findsText(finds))},
		{p.robEvery > 0, "🦹 Rouba a cada " + core.FormatWait(p.robEvery)},
		{p.robBonus > 0, fmt.Sprintf("🦹 +%d%% de chance no !roubar", percent(p.robBonus))},
		{p.robHaul > 0, fmt.Sprintf("💰 Leva %d%% a mais do que rouba no !roubar e na !quadrilha", p.robHaul)},
		{p.theftEvery > 0, "🥷 Furta a cada " + core.FormatWait(p.theftEvery)},
		{p.theftBonus > 0, fmt.Sprintf("🥷 +%d%% de chance no !furtar", percent(p.theftBonus))},
		{p.kidnapEvery > 0, "🚐 Sequestra a cada " + core.FormatWait(p.kidnapEvery)},
		{p.kidnapBonus > 0, fmt.Sprintf("🚐 +%d%% de chance no !sequestrar", percent(p.kidnapBonus))},
		{p.ransomBonus > 0, fmt.Sprintf("🤝 Vende refém %d%% mais caro no !venderrefem", p.ransomBonus)},
		{p.breakoutBonus > 0, fmt.Sprintf("🪢 +%d%% de chance no !fugir do cativeiro quando é sequestrado", percent(p.breakoutBonus))},
		{p.breakoutBonus < 0, fmt.Sprintf("😴 %d%% menos chance no !fugir do cativeiro quando é sequestrado", percent(-p.breakoutBonus))},
		{p.killChance > 0, fmt.Sprintf("🔪 Pode !matar, com %d%% de chance (menos a proteção da vítima)", percent(p.killChance))},
		{p.guard > 0, fmt.Sprintf("🛡️ Quem tenta !roubar, !furtar, !sequestrar ou !matar você tem %d%% menos chance", percent(p.guard))},
		{p.cordon > 0, fmt.Sprintf("🕴️ Segurança presidencial: quem tenta !sequestrar ou !matar você tem mais %d%% de chance de errar", percent(p.cordon))},
		{p.warded, fmt.Sprintf("🗿 A Carranca espanta mau espírito: a Casa assombrada não faz nada com o seu imóvel, nem o IPTU dobra, e a proteção e os bônus do imóvel valem %d%% a mais", carrancaBoost)},
		{p.jailTime > 0, fmt.Sprintf("🚔 Pode !prender quem tentou !roubar, !furtar, !sequestrar, !matar, entrou num !assalto, negociou no !mercadonegro ou desviou o cofre da igreja nas últimas %s, por %s, a cada %s: quem tá preso não aposta, não rouba, não furta, não sequestra e não assalta. Quem tenta te !sequestrar tem bem mais chance de ir preso na hora", core.FormatWait(crimeWindow), core.FormatWait(p.jailTime), core.FormatWait(p.arrestEvery))},
		{len(p.abuses) > 0, p.abusesText()},
		{p.releaseChance > 0, fmt.Sprintf("🔓 Pode !libertar alguém da cadeia ou do cativeiro, com %d%% de chance. Se der errado, pode levar multa", percent(p.releaseChance))},
		{p.reaper, fmt.Sprintf("💀 No !matar, passa direto pelo colete, a vítima fica %dx mais tempo morta e o enterro (de %d%% a %d%% do saldo dela) vai pro seu bolso", reaperHoldFactor, reapLossMinPercent, reapLossMaxPercent)},
		{p.posse, fmt.Sprintf("🤠 Caça o procurado do dia: pode !matar ele mesmo sem ser do crime, com pelo menos %d%% de chance (menos a proteção dele), sem contar como crime, e leva a recompensa do bot em dobro. Contra quem tem o Chapéu de Lampião, +%d%% no !matar", percent(posseKillChance), percent(posseLampiaoBonus))},
		{p.reviveChance > 0, fmt.Sprintf("🧟 Pode !reviver quem foi morto no !matar, com %d%% de chance", percent(p.reviveChance))},
		{p.soulPact, fmt.Sprintf("📕 Quem você revive fica %s passando %d%% de tudo que ganha pra você", core.FormatWait(soulPactHold), soulPactPercent)},
		{p.heistBonus > 0, fmt.Sprintf("🏦 Soma +%d%% de chance pra equipe no !assalto", percent(p.heistBonus))},
		{p.heistBonus < 0, fmt.Sprintf("🤦 Atrapalha o !assalto: a equipe perde %d%% de chance", percent(-p.heistBonus))},
		{p.jailFactor > 0, fmt.Sprintf("⛓️ Quando é preso no !prender ou no !assalto, fica %d%% mais tempo na cadeia", p.jailFactor)},
		{p.jailFactor < 0, fmt.Sprintf("🗝️ Quando é preso no !prender ou no !assalto, fica %d%% menos tempo na cadeia", -p.jailFactor)},
		{p.elusive, "👻 Fica invisível quando o crime dá errado: nunca é pego(a) na hora no !sequestrar, no !matar, no !fugir @pessoa nem no !assalto"},
		{p.escapeBonus > 0, fmt.Sprintf("🏃 +%d%% de chance no !fugir da cadeia e de escapar quando o !assalto dá errado", percent(p.escapeBonus))},
		{p.escapeBonus < 0, fmt.Sprintf("👀 %d%% menos chance no !fugir e de escapar quando o !assalto dá errado: todo mundo fica de olho", percent(-p.escapeBonus))},
		{p.reportBonus > 0, fmt.Sprintf("🕵️ No !denunciar, a polícia acredita em %d%% das vezes (o normal é %d%%)", percent(p.reportOdds()), percent(reportChance))},
		{p.reportBonus < 0, fmt.Sprintf("🙄 No !denunciar, a polícia só acredita em %d%% das vezes (o normal é %d%%)", percent(p.reportOdds()), percent(reportChance))},
		{p.noSnitch, fmt.Sprintf("🐀 Não pode !denunciar: bandido que vira X-9 é pego pela quadrilha, que sorteia o castigo: %s de cadeia, queima de arquivo (morte) ou %s em cativeiro com quem ele dedurou", core.FormatWait(p.snitchSentence()), core.FormatWait(hostageHold))},
		{p.craft.activity != "", p.craft.describe()},
		{p.gatherFaster > 0, fmt.Sprintf("⏱️ Espera %d%% menos no %s", p.gatherFaster, extractionCommands())},
		{p.haste < 0, fmt.Sprintf("⏩ Espera %d%% menos no %s", -p.haste, hasteCommands)},
		{p.haste > 0, fmt.Sprintf("🐌 Espera %d%% mais no %s", p.haste, hasteCommands)},
		{p.boxLuck > 0, fmt.Sprintf("📦 Abre caixa e baú no !usar com %dx mais sorte: fica com o item mais raro de %d sorteios", p.boxLuck+1, p.boxLuck+1)},
		{p.oracle, fmt.Sprintf("🐚 Os búzios leem o destino: os eventos aparecem %dx mais (%d%% das vezes em vez de %d%%) e a enquete mostra a chance de cada opção dar bom e dar ruim", oracleEventFactor, percent(eventChance*oracleEventFactor), percent(eventChance))},
		{p.compassLure > 0, fmt.Sprintf("🧭 Acha a Bússola do relicário %dx mais fácil no %s", p.compassLure, extractionCommands())},
		{p.rotten, fmt.Sprintf("🪰 Tudo que acha no %s vem com qualidade %s (%d%% do valor)", extractionCommands(), worstQuality().tier().name, worstQuality().tier().percent)},
		{p.barrenHauls > 0, fmt.Sprintf("🫙 Volta de mão vazia %d%% mais vezes no %s", p.barrenHauls, extractionCommands())},
		{p.barrenHauls < 0, fmt.Sprintf("🤲 Volta de mão vazia %d%% menos vezes no %s", -p.barrenHauls, extractionCommands())},
		{p.gatherGrade > 0, fmt.Sprintf("✨ O que acha no %s vem com qualidade melhor: fica com a melhor de %d", extractionCommands(), p.gatherGrade+1)},
		{p.gatherGrade < 0, fmt.Sprintf("🥀 O que acha no %s vem com qualidade pior: fica com a pior de %d", extractionCommands(), -p.gatherGrade+1)},
		{p.gatherLuck > 0, fmt.Sprintf("👃 No %s, sorteia %d vezes e fica com o achado mais valioso (caixa de relíquia não conta nos sorteios extras)", extractionCommands(), p.gatherLuck+1)},
		{p.gatherRare > 0, fmt.Sprintf("🔎 No %s, item raro ou acima aparece %d%% mais", extractionCommands(), p.gatherRare)},
		{p.gatherJunk > 0, fmt.Sprintf("🌱 No %s, vem %d%% menos item comum e mão vazia", extractionCommands(), p.gatherJunk)},
		{p.salePercent >= marketSalePercent, fmt.Sprintf("💸 O mercado paga %d%% do valor no !vender (o normal é %d%%)", p.salePercent, marketSalePercent)},
		{p.salePercent > 0 && p.salePercent < marketSalePercent, fmt.Sprintf("🥀 O mercado só paga %d%% do valor no !vender (o normal é %d%%)", p.salePercent, marketSalePercent)},
		{p.discount > 0, fmt.Sprintf("🛒 %d%% de desconto na !loja", p.discount)},
		{p.interest > 0, fmt.Sprintf("🏦 A poupança rende %d%% em vez de %d%%", savingsRate+p.interest, savingsRate)},
		{p.savingsRoom > 0, fmt.Sprintf("🐷 Guarda até %s por vez na poupança (o normal é %s)", Coins(p.savingsLimit()), Coins(savingsCap))},
		{p.savingsFaster > 0, fmt.Sprintf("⏳ A poupança libera em %s em vez de %s", core.FormatWait(p.savingsWait()), core.FormatWait(savingsLock))},
		{p.dailyBonus > 0, fmt.Sprintf("🎁 O !daily rende +%d%%", p.dailyBonus)},
		{p.cashback > 0, fmt.Sprintf("🧾 Recebe %d%% de volta quando perde no !apostar e no !cassino", p.cashback)},
		{p.jinx() > 0, fmt.Sprintf("🐈‍⬛ %d%% das vitórias no !apostar e no !cassino viram derrota", percent(p.jinx()))},
		{p.jinxProof, "🍀 A maldição Azar não estraga suas vitórias no !apostar e no !cassino"},
		{p.casinoLuck > 0, fmt.Sprintf("🃏 Leva %d%% a mais de lucro quando ganha no %s", p.casinoLuck, casinoGames)},
		{p.rescueOdds > 0, fmt.Sprintf("🃏 %d%% de chance de pegar a aposta inteira de volta quando perde no %s", percent(p.rescueOdds), casinoGames)},
		{p.animalLuck > 0, fmt.Sprintf("🦁 Leva %d%% a mais de lucro quando acerta no !bicho", p.animalLuck)},
		{p.legalDiscount > 0, fmt.Sprintf("⚖️ Paga %d%% menos de pensão e de indenização por traição", p.legalDiscount)},
		{p.trialDiscount > 0, fmt.Sprintf("💼 Paga %d%% menos quando perde um !processar", p.trialDiscount)},
		{p.trialBonus > 0, fmt.Sprintf("📜 Leva %d%% a mais quando ganha um !processar", p.trialBonus)},
		{p.trialEdge > 0, fmt.Sprintf("🎓 No !processar, conta como %s a mais pro seu lado no júri", core.CountLabel(p.trialEdge, "voto", "votos"))},
		{p.trialEdge < 0, fmt.Sprintf("🚨 Ficha suja: no !processar, o júri ganha %s contra você", core.CountLabel(-p.trialEdge, "voto", "votos"))},
		{p.voteWeight > 1, fmt.Sprintf("🗳️ Seu voto conta como %d no !votar, no !impeachment e no júri do !processar", p.voteWeight)},
		{p.trialEvery > 0, fmt.Sprintf("⏳ Abre um !processar a cada %s (o normal é %s)", core.FormatWait(p.trialEvery), core.FormatWait(trialEvery))},
		{p.freeBrokerage, fmt.Sprintf("📈 Não paga a corretagem de %d%% no !acoes", stockFeePercent)},
		{p.stockProfit > 0, fmt.Sprintf("💹 O lucro na venda do !acoes rende %d%% a mais", p.stockProfit)},
		{p.lendRate > 0 || p.lendLateRate > 0, p.loanRatesPerk()},
		{p.garnish > 0, fmt.Sprintf("💼 Quem te deve no !agiota com o prazo vencido tem %d%% do !trabalhar descontado direto pra dívida", p.garnish)},
		{p.collectBonus > 0, fmt.Sprintf("💢 Cobrando dívida vencida do !agiota, tem +%d%% de chance no !matar e no !sequestrar", percent(p.collectBonus))},
		{p.collectEvery > 0, fmt.Sprintf("🚓 Manda !prender quem te deve com o prazo vencido a cada %s (o normal é %s)", core.FormatWait(p.collectEvery), core.FormatWait(debtArrestEvery))},
		{p.collectVotes > 0, fmt.Sprintf("📑 No !processar por calote, conta como %s a mais pro seu lado no júri", core.CountLabel(p.collectVotes, "voto", "votos"))},
	}
	var lines []string
	for _, perk := range perks {
		if perk.on {
			lines = append(lines, perk.text)
		}
	}
	lines = append(lines, tradePerks(p)...)
	lines = append(lines, famePerks(p)...)
	lines = append(lines, faithPerks(p)...)
	lines = append(lines, contrabandPerks(p)...)
	lines = append(lines, luxuryPerks(p)...)
	lines = append(lines, implantPerks(p)...)
	lines = append(lines, titlePerks(p)...)
	lines = append(lines, churchPerks(p)...)
	return append(lines, charmPerks(p)...)
}

func (p Profession) jinx() float64 {
	if p.jinxProof {
		return 0
	}
	return p.betJinx
}

func (p Profession) savingsLimit() int64 {
	return savingsCap + p.savingsRoom
}

func (p Profession) savingsWait() time.Duration {
	return savingsLock * time.Duration(100-p.savingsFaster) / 100
}

func (p Profession) ballotWeight() int {
	return max(p.voteWeight, 1)
}

func (p Profession) label() string {
	return p.emoji + " *" + p.name + "*"
}

func (p Profession) Details() string {
	lines := []string{p.label(), p.earnings()}
	lines = append(lines, p.perks()...)
	return strings.Join(lines, "\n")
}

func (b *bot) ProfessionOf(ctx context.Context, chat types.JID, acc core.Account) (Profession, error) {
	var (
		st      standing
		decreed decrees
		weapon  item
		held    []heldCharm
		passive []item
		body    []installed
		fans    audience
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { held, err = b.Wallets.charmsOn(gctx, chat, acc, time.Now()); return err })
	g.Go(func() (err error) { body, err = b.Wallets.body(gctx, chat, acc); return err })
	g.Go(func() (err error) { st, err = b.Wallets.standing(gctx, chat, acc); return err })
	g.Go(func() (err error) { decreed, err = b.Wallets.decrees(gctx, chat, time.Now()); return err })
	g.Go(func() (err error) { weapon, err = b.Wallets.bestWeapon(gctx, chat, acc); return err })
	g.Go(func() (err error) { passive, err = b.Wallets.passives(gctx, chat, acc); return err })
	g.Go(func() (err error) { fans, err = b.Wallets.audience(gctx, chat, acc, time.Now()); return err })
	if err := g.Wait(); err != nil {
		return Profession{}, fmt.Errorf("ler profissão: %w", err)
	}
	job, _ := professionNamed(st.profession)
	job.decrees = decreed
	if st.president {
		job = job.presidential().tyrannical(time.Now())
	}
	return job.housed(st.estate).escorted(strongest(kindsOf(st.staff), watchOf)).hexed(st.cursed).armed(weapon).drugged(st.drug).adorned(passive).implanted(body, time.Now()).tooled(passive).titled(st.title).devoted(st.church, st.doctrine, st.temple, st.tithe).famed(fans).charmed(held), nil
}

func (b *bot) rivalProfessions(ctx context.Context, chat types.JID, attacker, target core.Account) (attackerJob, targetJob Profession, err error) {
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { attackerJob, err = b.ProfessionOf(gctx, chat, attacker); return err })
	g.Go(func() (err error) { targetJob, err = b.ProfessionOf(gctx, chat, target); return err })
	err = g.Wait()
	return attackerJob, targetJob, err
}

func (b *bot) trabalhar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	now := time.Now()
	var (
		job  Profession
		p    presidency
		lien debtLien
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { job, p, err = b.workTerms(gctx, chat, acc, now); return err })
	g.Go(func() (err error) { lien, err = b.salaryLien(gctx, chat, acc, now); return err })
	if err := g.Wait(); err != nil {
		return err
	}
	s := p.payFor(job.work(now))
	s.wound = job.hazard.roll(job.safety)
	levy := p.incomeLevy(acc)
	w, err := b.Wallets.work(ctx, chat, acc, cooldown{column: "trabalho_em", every: p.workCooldown(job.every)}, job.abuse(workAbuse), s, levy, lien, now)
	if err != nil {
		return fmt.Errorf("trabalhar: %w", err)
	}
	if w.wait == 0 {
		b.advance(chat, acc, workGoal, 1)
	}
	text := s.text(job, person.User, w.paid) + incomeTaxText(w.tax, levy) + w.tithe.text() + gravityText(w.pulled) + w.garnish.text() + w.abuse.note("Abusou do poder e trabalhou sem descansar")
	if w.wait > 0 {
		text = fmt.Sprintf("😮‍💨 @%s ainda tá cansado do último trampo. Volta em %s", person.User, core.FormatWait(w.wait))
	}
	text += p.decrees.note(workDecree) + p.decrees.note(payDecree)
	if job.cursed && w.wait == 0 {
		text += "\n🧿 Tem uma praga do !zoar em cima de você: o trampo pagou metade. Pra se livrar: !zoar limpar"
	}
	hurtText, freed := w.injury.text(person)
	people := append(append([]types.JID{person}, freed...), w.garnish.people()...)
	if w.tax > 0 {
		people = append(people, levy.payee()...)
	}
	_, err = b.SendMentions(ctx, chat, text+hurtText, people)
	if err != nil || w.wait > 0 || w.injury.died {
		return err
	}
	b.stirEvent(ctx, evt, eventSpot{place: "!trabalhar", won: !s.busted})
	return nil
}

func (b *bot) profissao(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	current, err := b.ProfessionOf(ctx, chat, acc)
	if err != nil {
		return err
	}
	args := strings.Fields(strings.ToLower(core.MessageText(evt.Message)))[1:]
	if len(args) == 0 {
		return b.SendText(ctx, chat, professionsMenu(current)+b.godHint(ctx, evt))
	}
	if i, ok := findArea(strings.Join(args, " ")); ok {
		return b.SendText(ctx, chat, areaText(i))
	}
	if slices.Contains([]string{"ver", "info", "detalhes"}, args[0]) {
		return b.SendText(ctx, chat, professionInfo(strings.Join(args[1:], " "), current))
	}

	query := strings.Join(args, " ")
	chosen, ok := findProfession(query)
	if b.summonsGod(ctx, evt, query) {
		chosen, ok = god, true
	}
	if !ok {
		return b.SendText(ctx, chat, "Não conheço essa profissão 🤔 Manda !profissao pra ver todas")
	}
	if chosen.name == current.name {
		return b.SendText(ctx, chat, fmt.Sprintf("Você já %s %s", chosen.title, chosen.emoji))
	}
	if b.holdIfJailed(ctx, evt) || b.holdIfHostage(ctx, evt) {
		return nil
	}
	every := professionChangeEvery
	if chosen.divine || current.divine {
		every = 0
	}
	wait, err := b.Wallets.changeProfession(ctx, chat, acc, chosen.name, every, time.Now())
	if err != nil {
		return fmt.Errorf("trocar profissão: %w", err)
	}
	text := fmt.Sprintf("%s @%s agora %s! Manda !trabalhar pra começar o trampo\n\n%s", chosen.emoji, person.User, chosen.title, chosen.Details())
	if wait > 0 {
		text = fmt.Sprintf("📋 @%s trocou de profissão faz pouco tempo. Espera %s pra virar %s", person.User, core.FormatWait(wait), chosen.name)
	}
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func professionInfo(query string, current Profession) string {
	if query == "" {
		return "Sua profissão hoje:\n\n" + current.Details()
	}
	job, ok := findProfession(query)
	if !ok {
		return "Não conheço essa profissão 🤔 Manda !profissao pra ver todas"
	}
	return job.Details() + fmt.Sprintf("\n\n_Pra virar: !profissao %s_", job.name)
}

func professionsMenu(current Profession) string {
	lines := []string{fmt.Sprintf("💼 *As %d profissões*", len(professions))}
	for i, a := range areas {
		var names []string
		for _, p := range professions {
			if p.area == i {
				names = append(names, p.emoji+" "+p.name)
			}
		}
		lines = append(lines, "", fmt.Sprintf("%s *%s* (!profissao %s)", a.emoji, a.name, a.key), strings.Join(names, ", "))
	}
	lines = append(lines, "",
		fmt.Sprintf("Hoje você %s %s", current.title, current.emoji),
		"_!profissao <área> mostra quanto cada uma paga e o que ela tem de especial, e !profissao ver <nome> mostra uma só_",
		fmt.Sprintf("_Pra trocar: !profissao <nome>. Dá pra trocar uma vez a cada %s_", core.FormatWait(professionChangeEvery)))
	return strings.Join(lines, "\n")
}

func areaText(i int) string {
	a := areas[i]
	lines := []string{fmt.Sprintf("%s *%s*", a.emoji, a.name)}
	for _, p := range professions {
		if p.area != i {
			continue
		}
		lines = append(lines, "", p.label(), p.earnings())
		lines = append(lines, p.perks()...)
	}
	lines = append(lines, "", "_Pra virar uma delas: !profissao <nome>_")
	return strings.Join(lines, "\n")
}
