package economy

import (
	"cmp"
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	common = iota
	uncommon
	rare
	epic
	legendary
	mythic
	relic
	primordial
)

const (
	lockItemID      = "cadeado"
	alarmItemID     = "alarme"
	alarmCarry      = 3
	vestItemID      = "colete"
	vestCarry       = 3
	vestChance      = 0.55
	forgedReleaseID = "alvara-falso"
	scannerTipoff   = 0.25
)

type limiter struct {
	cooldown cooldown
	rule     string
	tired    string
}

var (
	caffeine = limiter{
		cooldown: cooldown{column: "cafeina_em", every: 2 * time.Hour},
		rule:     "cafeína só dá pra tomar uma vez a cada %s",
		tired:    "😵‍💫 @%s já tomou cafeína demais e o coração tá pedindo arrego. Guarda %s pra daqui a %s",
	}
	disguise = limiter{
		cooldown: cooldown{column: "disfarce_em", every: 3 * time.Hour},
		rule:     "a polícia decora cara nova rápido, então só dá pra trocar de disfarce uma vez a cada %s",
		tired:    "🕵️ @%s trocou de cara faz pouco tempo e a polícia ainda tá desconfiada. Guarda %s pra daqui a %s",
	}
)

type rarity struct {
	name   string
	plural string
	emoji  string
	steal  float64
}

var rarities = [...]rarity{
	common:     {"comum", "Comuns", "⚪", 0.6},
	uncommon:   {"incomum", "Incomuns", "🟢", 0.5},
	rare:       {"raro", "Raros", "🔵", 0.4},
	epic:       {"épico", "Épicos", "🟣", 0.3},
	legendary:  {"lendário", "Lendários", "🟡", 0.2},
	mythic:     {"mítico", "Míticos", "🟠", 0.15},
	relic:      {"relíquia", "Relíquias", "🔴", 0.1},
	primordial: {"primordial", "Primordiais", "⚫", 0.05},
}

func (r rarity) heading() string {
	return r.emoji + " *" + r.plural + "*"
}

type prize struct {
	weight int
	amount int64
}

type item struct {
	id        string
	name      string
	emoji     string
	rarity    int
	value     int64
	shop      bool
	aliases   []string
	use       string
	gift      string
	keep      bool
	resets    []string
	limit     *limiter
	rotating  bool
	days      []time.Weekday
	prizes    []prize
	loot      []int
	heist     float64
	luxury    bool
	flex      *flex
	bless     string
	hex       string
	pet       bool
	illegal   bool
	arms      *armament
	high      *high
	heal      int64
	doses     int64
	gear      *gear
	trade     *trade
	metal     bool
	relicOdds float64
	shard     string
	shardOdds float64
	forms     string
	shards    int64
	upgrades  string
	graft     *implant
	networks  []string
	hot       bool
	seed      *crop
	carry     int64
	daily     int64
	storeOnly bool
}

type stack struct {
	item     item
	quantity int64
	grades   []grade
}

func (s stack) worth() int64 {
	return worthOf(s.item, s.grades)
}

type itemEffect struct {
	prize item
	coins int64
}

var cooldownCommands = map[string]string{
	"trabalho_em":  "!trabalhar",
	"roubo_em":     "!roubar",
	"furto_em":     "!furtar",
	"sequestro_em": "!sequestrar",
	"pesca_em":     "!pescar",
	"mineracao_em": "!minerar",
	"lixao_em":     "!vasculhar",
	"inseto_em":    "!catar",
}

var catalog = slices.Concat([]item{
	{id: "vara-de-pesca", name: "Vara de pesca", emoji: "🎣", rarity: uncommon, value: 300, shop: true, aliases: []string{"vara"}},
	{id: "picareta", name: "Picareta", emoji: "⛏️", rarity: uncommon, value: 400, shop: true},
	{id: "rede-de-insetos", name: "Rede de insetos", emoji: "🥅", rarity: uncommon, value: 150, shop: true, aliases: []string{"rede", "puca", "vara de insetos"}},
	{id: "isca", name: "Isca de minhoca", emoji: "🪱", rarity: common, value: 30, shop: true, aliases: []string{"minhoca", "isca"},
		gear: &gear{activities: []string{"!pescar"}, luck: 1, spent: true, skipsRelics: true}},
	{id: "vara-de-carbono", name: "Vara de carbono", emoji: "🎣", rarity: rare, value: 1500, shop: true, aliases: []string{"carbono"},
		gear: &gear{activities: []string{"!pescar"}, tool: true, sturdy: 60, luck: 1, fewerJunk: 25, grade: 1}},
	{id: "picareta-de-diamante", name: "Picareta de diamante", emoji: "💎", rarity: rare, value: 1800, shop: true,
		gear: &gear{activities: []string{"!minerar"}, tool: true, sturdy: 60, grace: 10, fewerJunk: 58, moreRare: 5, grade: 1}},
	{id: "vara-de-titanio", name: "Vara de titânio", emoji: "🎣", rarity: epic, value: 6000, shop: true, rotating: true, aliases: []string{"titanio"},
		gear: &gear{activities: []string{"!pescar"}, tool: true, sturdy: 85, grace: 20, luck: 2, faster: 40, fewerJunk: 57, moreRare: 22, grade: 2}},
	{id: "picareta-de-niobio", name: "Picareta de nióbio", emoji: "⛏️", rarity: epic, value: 7000, shop: true, rotating: true, aliases: []string{"niobio"},
		gear: &gear{activities: []string{"!minerar"}, tool: true, sturdy: 85, grace: 20, luck: 1, faster: 40, fewerJunk: 70, moreRare: 2, grade: 2}},
	{id: "vara-de-grafeno", name: "Vara de grafeno", emoji: "🎣", rarity: legendary, value: 14_000, aliases: []string{"grafeno", "vara lendaria"},
		gear: &gear{activities: []string{"!pescar"}, tool: true, sturdy: 97, grace: 100, luck: 4, faster: 50, fewerJunk: 74, moreRare: 10, grade: 3}},
	{id: "picareta-de-tungstenio", name: "Picareta de tungstênio", emoji: "⛏️", rarity: legendary, value: 15_000, aliases: []string{"tungstenio", "picareta lendaria"},
		gear: &gear{activities: []string{"!minerar"}, tool: true, sturdy: 97, grace: 100, luck: 3, faster: 50, fewerJunk: 62, grade: 3}},
	{id: "rede-de-seda", name: "Rede de seda", emoji: "🥅", rarity: rare, value: 1_200, shop: true, aliases: []string{"seda"},
		gear: &gear{activities: []string{"!catar"}, tool: true, sturdy: 60, luck: 1, fewerJunk: 48, moreRare: 1, grade: 1}},
	{id: "rede-de-nailon", name: "Rede de náilon", emoji: "🥅", rarity: epic, value: 5_000, shop: true, rotating: true, aliases: []string{"nailon", "nylon", "rede de nylon"},
		gear: &gear{activities: []string{"!catar"}, tool: true, sturdy: 85, grace: 20, luck: 2, faster: 40, fewerJunk: 71, moreRare: 15, grade: 2}},
	{id: "rede-entomologica", name: "Rede entomológica", emoji: "🥅", rarity: legendary, value: 12_000, aliases: []string{"entomologica", "rede lendaria"},
		gear: &gear{activities: []string{"!catar"}, tool: true, sturdy: 97, grace: 100, luck: 4, faster: 50, fewerJunk: 76, moreRare: 18, grade: 3}},
	{id: "vara-de-iemanja", name: "Vara de Iemanjá", emoji: "🌊", rarity: relic, value: 25_000, aliases: []string{"iemanja"},
		gear: &gear{activities: []string{"!pescar"}, tool: true, sturdy: 100, luck: 6, faster: 80, fewerJunk: 76, moreRare: 100, grade: 4, alwaysFinds: true}},
	{id: "picareta-de-serra-pelada", name: "Picareta de Serra Pelada", emoji: "⚒️", rarity: relic, value: 25_000, aliases: []string{"serra pelada"},
		gear: &gear{activities: []string{"!minerar"}, tool: true, sturdy: 100, luck: 5, faster: 80, fewerJunk: 69, moreRare: 100, grade: 4, alwaysFinds: true}},
	{id: "rede-do-curupira", name: "Rede do Curupira", emoji: "🍃", rarity: relic, value: 25_000, aliases: []string{"curupira"},
		gear: &gear{activities: []string{"!catar"}, tool: true, sturdy: 100, luck: 6, faster: 80, fewerJunk: 81, moreRare: 100, grade: 4, alwaysFinds: true}},
	{id: "excalibur", name: "Excalibur", emoji: "🗡️", rarity: relic, value: 40_000, aliases: []string{"espada do rei arthur", "espada na pedra"},
		arms: &armament{crime: 0.30, kill: 0.70, guard: 0.45}},
	{id: "saco-do-homem-do-saco", name: "Saco do Homem do Saco", emoji: "🎒", rarity: relic, value: 25_000, aliases: []string{"homem do saco"},
		gear: &gear{activities: []string{"!vasculhar"}, luck: 7, faster: 80, fewerJunk: 83, moreRare: 100, grade: 4, alwaysFinds: true}},
	{id: "vela-do-negrinho-do-pastoreio", name: "Vela do Negrinho do Pastoreio", emoji: "🕯️", rarity: relic, value: 25_000, aliases: []string{"pastoreio", "negrinho do pastoreio"},
		gear: &gear{activities: []string{"!pescar", "!minerar", "!vasculhar", "!catar"}, luck: 3, alwaysFinds: true}},
	{id: "muiraquita", name: "Muiraquitã", emoji: "🪬", rarity: relic, value: 30_000, aliases: []string{"amuleto"},
		flex: &flex{effect: "+35% de chance de ganhar em dobro no !trabalhar, somado com a maior chance que você já tiver", apply: func(p Profession) Profession { p.relicDouble += 0.35; return p }}},
	{id: "gorro-do-saci", name: "Gorro do Saci", emoji: "🧢", rarity: relic, value: 30_000, aliases: []string{"gorro"},
		flex: &flex{effect: "deixa você invisível quando o crime dá errado: nunca é pego(a) na hora no !sequestrar, no !matar, no !fugir @pessoa nem no !assalto, e quando vai pra cadeia fica 50% menos tempo", apply: func(p Profession) Profession {
			p.elusive = true
			p.jailFactor -= 50
			return p
		}}},
	{id: lampiaoItem, name: "Chapéu de Lampião", emoji: "🌵", rarity: relic, value: 30_000, aliases: []string{"lampiao"},
		flex: &flex{effect: "+20% de chance no !roubar, no !furtar e pra equipe no !assalto, e leva 50% a mais do que rouba no !roubar e na !quadrilha", apply: func(p Profession) Profession {
			p.robBonus += 0.20
			p.robHaul += 50
			p.theftBonus += 0.20
			p.heistBonus += 0.20
			return p
		}}},
	{id: "escamas-do-boitata", name: "Escamas do Boitatá", emoji: "🐍", rarity: relic, value: 30_000, aliases: []string{"boitata"},
		flex: &flex{effect: "quem tenta !roubar, !furtar, !sequestrar ou !matar você tem 25% menos chance", apply: func(p Profession) Profession { p.guard += 0.25; return p }}},
	{id: "coroa-de-dom-pedro", name: "Coroa de Dom Pedro", emoji: "🤴", rarity: relic, value: 30_000, aliases: []string{"dom pedro", "coroa imperial"},
		flex: &flex{effect: "o !daily rende 350% a mais e o seu voto conta 3 a mais no !votar, no !impeachment e no júri do !processar", apply: func(p Profession) Profession {
			p.dailyBonus += 350
			p.voteWeight = p.ballotWeight() + 3
			return p
		}}},
	{id: "santo-graal", name: "Santo Graal", emoji: "🏺", rarity: relic, value: 30_000, aliases: []string{"graal", "calice sagrado"},
		flex: &flex{effect: "a poupança rende 5 pontos a mais, guarda 20.000 a mais por vez e libera na metade do tempo, e o lucro na venda do !acoes rende 25% a mais", apply: func(p Profession) Profession {
			p.interest += 5
			p.savingsRoom += 20_000
			p.savingsFaster += 50
			p.stockProfit += 25
			return p
		}}},
	{id: "baralho-do-ze-pelintra", name: "Baralho do Zé Pelintra", emoji: "🃏", rarity: relic, value: 30_000, aliases: []string{"ze pelintra", "baralho"},
		flex: &flex{effect: "leva 30% a mais de lucro quando ganha no " + casinoGames + ", tem 20% de chance de pegar a aposta inteira de volta quando perde neles, e a maldição Azar não estraga suas apostas", apply: func(p Profession) Profession {
			p.casinoLuck += 30
			p.rescueOdds += 0.20
			p.jinxProof = true
			return p
		}}},
	{id: mirrorItem, name: "Espelho da Iara", emoji: "🧜‍♀️", rarity: relic, value: 30_000, aliases: []string{"espelho encantado", "espelho da sereia"},
		flex: &flex{effect: fmt.Sprintf("lança qualquer bênção ou maldição no !abencoar e no !amaldicoar (a praga pega em pelo menos %d%% das vezes), e toda maldição que jogam em você volta pra quem jogou", percent(mirrorHexChance)),
			apply: func(p Profession) Profession { p.mirror = true; return p }}},
	{id: "lamparina-do-lobisomem", name: "Lamparina do Lobisomem", emoji: "🐺", rarity: relic, value: 30_000, aliases: []string{"lamparina", "lobisomem"},
		flex: &flex{effect: "só funciona da meia-noite às 6h (horário de Brasília), quando você vira lobisomem: +30% de chance no !roubar, no !furtar, no !sequestrar e no !matar (que fica liberado pra qualquer profissão), +20% pra equipe no !assalto, 20% menos chance pra quem tenta te pegar, +25% no !fugir, o !trabalhar paga 50% a mais e recarrega 30% mais rápido, espera 30% menos no " + hasteCommands + ", +10% de chance de achar item trabalhando, o !daily rende 50% a mais, a poupança rende 2 pontos a mais, 20% a mais de lucro no " + casinoGames + ", caixa e baú abrem com 2x mais sorte, refém vende 25% mais caro e o lucro na venda do !acoes rende 15% a mais. De dia não faz nada", apply: func(p Profession) Profession {
			if !werewolfHour(time.Now()) {
				return p
			}
			p.robBonus += 0.30
			p.theftBonus += 0.30
			p.kidnapBonus += 0.30
			p.killChance += 0.30
			p.heistBonus += 0.20
			p.guard += 0.20
			p.escapeBonus += 0.25
			p.breakoutBonus += 0.25
			p.min, p.max = p.min*150/100, p.max*150/100
			p.every = p.every * 70 / 100
			p.haste -= 30
			p.findBonus += 0.10
			p.dailyBonus += 50
			p.interest += 2
			p.casinoLuck += 20
			p.boxLuck++
			p.ransomBonus += 25
			p.stockProfit += 15
			return p
		}}},
	{id: "relicario", name: "Relicário perdido", emoji: "🗝️", rarity: legendary, value: 10_000, aliases: []string{"relicario", "caixa de reliquias", "caixa especial"},
		loot: []int{0, 0, 30, 50, 20}, relicOdds: 0.10, use: "🗝️ @%s girou a chave enferrujada do relicário perdido e um vento gelado saiu lá de dentro"},
	{id: "bussola-do-relicario", name: "Bússola do relicário", emoji: "🧭", rarity: epic, value: 2_500, aliases: []string{"bussola", "bussola encantada"},
		gear: &gear{activities: []string{"!pescar", "!minerar", "!vasculhar", "!catar"}, spent: true, relicLure: 15}},
	{id: "capacete", name: "Capacete com lanterna", emoji: "⛑️", rarity: uncommon, value: 600, shop: true, aliases: []string{"lanterna"},
		gear: &gear{activities: []string{"!minerar"}, faster: 30, safety: 60}},
	{id: "luva-de-borracha", name: "Luva de borracha", emoji: "🧤", rarity: uncommon, value: 350, shop: true, aliases: []string{"luva"},
		gear: &gear{activities: []string{"!vasculhar"}, luck: 1, safety: 50}},
	{id: "armadilha-luminosa", name: "Armadilha luminosa", emoji: "💡", rarity: uncommon, value: 400, shop: true, aliases: []string{"armadilha de luz"},
		gear: &gear{activities: []string{"!catar"}, faster: 30}},
	{id: "garrafa-de-melado", name: "Garrafa de melado", emoji: "🍯", rarity: common, value: 25, shop: true, aliases: []string{"melado"},
		gear: &gear{activities: []string{"!catar"}, luck: 1, spent: true, skipsRelics: true}},
	{id: "ima", name: "Ímã", emoji: "🧲", rarity: common, value: 15, shop: true, aliases: []string{"ima"},
		gear: &gear{activities: []string{"!vasculhar", "!minerar"}, spent: true, metal: true}},
	{id: lockItemID, name: "Cadeado", emoji: "🔒", rarity: uncommon, value: 250, shop: true},
	{id: alarmItemID, name: "Alarme", emoji: "🚨", rarity: rare, value: 2_000, shop: true, rotating: true, storeOnly: true, carry: alarmCarry},
	{id: "caixa-misteriosa", name: "Caixa misteriosa", emoji: "📦", rarity: rare, value: 400, shop: true, aliases: []string{"caixa"},
		loot: []int{45, 30, 17, 7, 1}, use: "📦 @%s rasgou a fita da caixa misteriosa sem nem ler o que tava escrito"},
	{id: "caixa-misteriosa-dourada", name: "Caixa misteriosa dourada", emoji: "✨", rarity: epic, value: 1_700, shop: true, rotating: true, aliases: []string{"caixa dourada", "caixa de ouro", "caixa premium"},
		loot: []int{0, 15, 40, 35, 10}, use: "✨ @%s abriu a caixa misteriosa dourada com luva branca e tudo, que nem unboxing de influencer"},
	{id: "raspadinha", name: "Raspadinha", emoji: "🎫", rarity: common, value: 100, shop: true,
		prizes: []prize{{60, 0}, {30, 100}, {8, 300}, {2, 1500}}, use: "🎫 @%s raspou a raspadinha com a unha do dedão"},
	{id: "cafe", name: "Café", emoji: "☕", rarity: common, value: 150, shop: true, resets: []string{"trabalho_em"}, limit: &caffeine,
		use: "☕ @%s tomou um café passado na hora e acordou pra vida"},
	{id: "energetico", name: "Energético", emoji: "⚡", rarity: common, value: 150, shop: true, aliases: []string{"monster", "redbull"},
		resets: []string{"pesca_em", "mineracao_em", "lixao_em", "inseto_em"}, limit: &caffeine, use: "⚡ @%s virou um energético de uma vez e agora tá ouvindo cores"},
	{id: "todynho", name: "Todynho", emoji: "🥛", rarity: common, value: 15, shop: true, aliases: []string{"toddynho", "todyinho", "todinho"},
		use: "🥛 @%s tomou um Todynho geladinho e voltou a ter 8 anos"},
	{id: "coxinha", name: "Coxinha", emoji: "🍗", rarity: common, value: 10, shop: true,
		use: "🍗 @%s comeu uma coxinha de rodoviária e sobreviveu. Por enquanto"},
	{id: "pastel", name: "Pastel de feira", emoji: "🥟", rarity: common, value: 15, shop: true, aliases: []string{"pastel"},
		use: "🥟 @%s comeu um pastel de vento com caldo de cana e se sentiu num domingo de manhã"},
	{id: "pao-de-queijo", name: "Pão de queijo", emoji: "🧀", rarity: common, value: 8, shop: true,
		use: "🧀 @%s comeu um pão de queijo quentinho, uai"},
	{id: "podrao", name: "Podrão", emoji: "🌭", rarity: common, value: 12, shop: true, aliases: []string{"cachorro quente", "hot dog"},
		use: "🌭 @%s comeu um podrão com purê, batata palha, milho, ervilha e fé"},
	{id: "dolly", name: "Dolly", emoji: "🥤", rarity: common, value: 6, shop: true, aliases: []string{"guarana", "refri"},
		use: "🥤 @%s tomou um Dolly. O Dollynho, seu amiguinho, agradece"},
	{id: "latao", name: "Latão", emoji: "🍺", rarity: common, value: 12, shop: true, aliases: []string{"cerveja", "breja"},
		use: "🍺 @%s abriu um latão quente no fim do mês e chamou de happy hour"},
	{id: "pizza", name: "Pizza de calabresa", emoji: "🍕", rarity: common, value: 50, shop: true, aliases: []string{"pizza"},
		use: "🍕 @%s comeu uma pizza de calabresa inteira sozinho e não ofereceu pra ninguém"},
	{id: "chocolate", name: "Barra de chocolate", emoji: "🍫", rarity: common, value: 25, shop: true, aliases: []string{"chocolate"},
		use:  "🍫 @%s comeu uma barra de chocolate inteira e jurou que a dieta começa segunda",
		gift: "🍫 @%[1]s deu uma barra de chocolate pra @%[2]s. Adoçando a relação"},
	{id: "rosa", name: "Rosa", emoji: "🌹", rarity: common, value: 40, shop: true,
		gift: "🌹 @%[1]s deu uma rosa pra @%[2]s. Tá rolando um clima 👀"},
	{id: "buque", name: "Buquê de flores", emoji: "💐", rarity: uncommon, value: 150, shop: true, aliases: []string{"buque", "flores"},
		gift: "💐 @%[1]s chegou com um buquê de flores pra @%[2]s. Alguém chama o padre"},
	{id: "ursinho", name: "Ursinho de pelúcia", emoji: "🧸", rarity: uncommon, value: 200, shop: true, keep: true, aliases: []string{"pelucia"},
		use:  "🧸 @%s abraçou o ursinho de pelúcia bem forte e se sentiu protegido",
		gift: "🧸 @%[1]s deu um ursinho de pelúcia pra @%[2]s. Que fofura"},
	{id: "anel", name: "Anel de compromisso", emoji: "💍", rarity: rare, value: 1200, shop: true, aliases: []string{"anel", "alianca"},
		gift: "💍 @%[1]s deu um anel de compromisso pra @%[2]s. Isso foi um pedido? 👀 O !namorar tá aí pra isso"},
	{id: "chimarrao", name: "Chimarrão", emoji: "🧉", rarity: common, value: 150, shop: true, aliases: []string{"mate"}, resets: []string{"trabalho_em"}, limit: &caffeine,
		use: "🧉 @%s tomou um chimarrão, bah, e voltou pro batente tri disposto"},
	{id: "tele-sena", name: "Tele Sena", emoji: "🎟️", rarity: common, value: 200, shop: true, days: []time.Weekday{time.Sunday},
		prizes: []prize{{60, 0}, {28, 150}, {9, 600}, {3, 2500}}, use: "🎟️ @%s raspou a Tele Sena no domingo à tarde com fé no Silvio"},
	{id: "acai", name: "Açaí de 500 ml", emoji: "🍇", rarity: common, value: 30, shop: true, aliases: []string{"acai"},
		use: "🍇 @%s pediu um açaí com leite ninho, paçoca e granola, e mais nada, porque o resto era cobrado à parte"},
	{id: "brigadeiro", name: "Brigadeiro", emoji: "🍬", rarity: common, value: 5, shop: true, aliases: []string{"negrinho"},
		use:  "🍬 @%s pegou um brigadeiro da mesa antes do parabéns. Ninguém viu. Todo mundo viu",
		gift: "🍬 @%[1]s deu um brigadeiro pra @%[2]s. Gesto pequeno, amor grande"},
	{id: "pamonha", name: "Pamonha", emoji: "🌽", rarity: common, value: 12, shop: true,
		use: "🌽 @%s comeu uma pamonha de Piracicaba, o puro creme do milho verde"},
	{id: "esfiha", name: "Esfiha de um real", emoji: "🫓", rarity: common, value: 3, shop: true, aliases: []string{"esfirra", "esfiha"},
		use: "🫓 @%s comeu 14 esfihas de um real e chamou de almoço"},
	{id: "miojo", name: "Miojo", emoji: "🍜", rarity: common, value: 4, shop: true, aliases: []string{"lamen", "macarrao instantaneo"},
		use: "🍜 @%s fez um miojo de 3 minutos. Levou 15, mas quem tá contando"},
	{id: "acaraje", name: "Acarajé", emoji: "🧆", rarity: common, value: 20, shop: true,
		use: "🧆 @%s pediu o acarajé quente e a baiana entendeu quente de pimenta. Tá chorando até agora"},
	{id: "churros", name: "Churros de doce de leite", emoji: "🥖", rarity: common, value: 12, shop: true, aliases: []string{"churros", "churro"},
		use: "🥖 @%s mordeu o churros e o doce de leite escorreu até o cotovelo"},
	{id: "cuscuz", name: "Cuscuz com ovo", emoji: "🍳", rarity: common, value: 15, shop: true, aliases: []string{"cuscuz"},
		use: "🍳 @%s tomou café da manhã com cuscuz e ovo e agora carrega o país nas costas"},
	{id: "caldo-de-cana", name: "Caldo de cana", emoji: "🎋", rarity: common, value: 8, shop: true, aliases: []string{"garapa"},
		use: "🎋 @%s tomou um caldo de cana com limão, bem gelado, igual domingo de feira"},
	{id: "sacole", name: "Sacolé", emoji: "🧊", rarity: common, value: 3, shop: true, aliases: []string{"geladinho", "dindin", "chup chup"},
		use: "🧊 @%s chupou um sacolé de uva e ficou com a língua roxa o resto do dia"},
	{id: "pipoca", name: "Pipoca de saquinho", emoji: "🍿", rarity: common, value: 5, shop: true, aliases: []string{"pipoca"},
		use: "🍿 @%s pegou uma pipoca pra assistir a próxima briga do grupo"},
	{id: "cachaca", name: "Cachaça de alambique", emoji: "🥃", rarity: common, value: 35, shop: true, aliases: []string{"cachaca", "pinga", "cana"},
		use: "🥃 @%s tomou uma dose de cachaça de alambique e agora tá contando a história da vida pra um poste"},
	{id: "carta-de-amor", name: "Carta de amor", emoji: "💌", rarity: common, value: 20, shop: true, aliases: []string{"carta", "cartinha"},
		gift: "💌 @%[1]s mandou uma carta de amor perfumada pra @%[2]s. Escrita à mão, com erro de português e tudo"},
	{id: "girassol", name: "Girassol", emoji: "🌻", rarity: common, value: 30, shop: true,
		gift: "🌻 @%[1]s deu um girassol pra @%[2]s. Pra iluminar o dia"},
	{id: "pulseira-da-amizade", name: "Pulseira da amizade", emoji: "🧶", rarity: common, value: 15, shop: true, aliases: []string{"pulseira"},
		gift: "🧶 @%[1]s amarrou uma pulseira da amizade no pulso de @%[2]s. Só tira quando arrebentar"},
	{id: "caixa-de-bombom", name: "Caixa de bombom", emoji: "🎁", rarity: uncommon, value: 60, shop: true, aliases: []string{"bombom"},
		use:  "🎁 @%s comeu a caixa de bombom inteira e deixou só o de coco, como manda a tradição",
		gift: "🎁 @%[1]s deu uma caixa de bombom pra @%[2]s. Os de coco vão sobrar"},
	{id: "mascara", name: "Máscara do Dalí", emoji: "🎭", rarity: uncommon, value: 200, shop: true, aliases: []string{"mascara", "dali"}, heist: 0.05},
	{id: "pe-de-cabra", name: "Pé de cabra", emoji: "🔧", rarity: uncommon, value: 250, shop: true, aliases: []string{"pe de cabra"}, heist: 0.07},
	{id: "furadeira", name: "Furadeira de cofre", emoji: "🛠️", rarity: rare, value: 600, shop: true, rotating: true, aliases: []string{"furadeira"}, heist: 0.1},
	{id: "planta-do-cofre", name: "Planta do cofre", emoji: "🗺️", rarity: rare, value: 900, aliases: []string{"planta", "mapa do cofre"}, heist: 0.15},
	{id: "dinamite", name: "Dinamite", emoji: "🧨", rarity: epic, value: 1500, aliases: []string{"tnt"}, heist: 0.2},

	{id: "pintinho", name: "Pintinho", emoji: "🐤", rarity: common, value: 40, shop: true, pet: true, keep: true, aliases: []string{"pintinho amarelinho"},
		use:  "🐤 @%s levou o pintinho pra passear e ele piou o caminho inteiro",
		gift: "🐤 @%[1]s deu um pintinho amarelinho pra @%[2]s. Cabe na mão, tem pé pequenininho"},
	{id: "peixinho-dourado", name: "Peixinho dourado", emoji: "🐟", rarity: common, value: 60, shop: true, pet: true, keep: true, aliases: []string{"peixinho"},
		use:  "🐟 @%s deu ração pro peixinho dourado, que esqueceu em 3 segundos e ficou feliz de novo",
		gift: "🐟 @%[1]s deu um peixinho dourado num saquinho pra @%[2]s. Cuidado pra não derrubar"},
	{id: "hamster", name: "Hamster", emoji: "🐹", rarity: common, value: 120, shop: true, pet: true, keep: true, aliases: []string{"hamister"},
		use:  "🐹 @%s botou o hamster na rodinha e ele correu 4 km sem sair do lugar, igual a gente na esteira",
		gift: "🐹 @%[1]s deu um hamster pra @%[2]s. Ele já guardou 12 sementes na bochecha"},
	{id: "jabuti", name: "Jabuti", emoji: "🐢", rarity: uncommon, value: 250, shop: true, pet: true, keep: true, aliases: []string{"tartaruga"},
		use:  "🐢 @%s levou o jabuti pra passear. Faz 2 horas e eles ainda não saíram do quintal",
		gift: "🐢 @%[1]s deu um jabuti pra @%[2]s. Ele vai viver mais que os dois juntos"},
	{id: "calopsita", name: "Calopsita", emoji: "🐦", rarity: uncommon, value: 300, shop: true, pet: true, keep: true,
		use:  "🐦 @%s assobiou e a calopsita respondeu com a musiquinha do Silvio Santos",
		gift: "🐦 @%[1]s deu uma calopsita pra @%[2]s. Ela já aprendeu a assobiar e não para mais"},
	{id: "coelhinho", name: "Coelhinho", emoji: "🐰", rarity: uncommon, value: 350, shop: true, pet: true, keep: true, aliases: []string{"coelho"},
		use:  "🐰 @%s fez carinho no coelhinho e ele mexeu o nariz de um jeito que derreteu o grupo",
		gift: "🐰 @%[1]s deu um coelhinho pra @%[2]s. Não é Páscoa, mas é fofura"},
	{id: "gatinho", name: "Gatinho", emoji: "🐱", rarity: rare, value: 600, shop: true, pet: true, keep: true, aliases: []string{"gato", "gatinha", "filhote de gato"},
		use:  "🐱 @%s fez carinho no gatinho e ganhou um ronronar de motorzinho. Depois levou uma mordida, porque gato é assim",
		gift: "🐱 @%[1]s deu um gatinho pra @%[2]s. Agora o sofá é dele"},
	{id: "filhote-de-cachorro", name: "Filhote de cachorro", emoji: "🐶", rarity: rare, value: 800, shop: true, pet: true, keep: true, aliases: []string{"cachorro", "cachorrinho", "filhote", "doguinho"},
		use:  "🐶 @%s chegou em casa e o filhote fez festa como se não se vissem há 10 anos. Foram 10 minutos",
		gift: "🐶 @%[1]s deu um filhote de cachorro com laço no pescoço pra @%[2]s. Alguém segura que vai ter choro"},
	{id: "golden", name: "Golden retriever", emoji: "🦮", rarity: epic, value: 1800, shop: true, pet: true, keep: true, aliases: []string{"golden retriever"},
		use:  "🦮 @%s jogou a bolinha pro golden e ele voltou com a bolinha, um chinelo e um amigo novo",
		gift: "🦮 @%[1]s deu um golden retriever pra @%[2]s. Agora a casa tem pelo até no café"},
	{id: "galinha", name: "Galinha caipira", emoji: "🐔", rarity: common, value: 80, shop: true, pet: true, keep: true, aliases: []string{"galinha"},
		use:  "🐔 @%s foi no quintal e a galinha tinha botado um ovo. Café da manhã garantido",
		gift: "🐔 @%[1]s deu uma galinha caipira pra @%[2]s. Agora tem ovo todo dia"},
	{id: "patinho", name: "Patinho", emoji: "🦆", rarity: common, value: 90, shop: true, pet: true, keep: true, aliases: []string{"pato"},
		use:  "🦆 @%s botou o patinho pra nadar na bacia e ele fez quá de felicidade",
		gift: "🦆 @%[1]s deu um patinho pra @%[2]s. Ele já tá seguindo a pessoa pela casa"},
	{id: "ourico", name: "Ouriço", emoji: "🦔", rarity: uncommon, value: 400, shop: true, pet: true, keep: true, aliases: []string{"porco espinho"},
		use:  "🦔 @%s tentou fazer carinho no ouriço e agora tá tirando espinho da mão",
		gift: "🦔 @%[1]s deu um ouriço pra @%[2]s. É fofo, mas não dá pra abraçar"},
	{id: "lhama", name: "Lhama", emoji: "🦙", rarity: rare, value: 1200, shop: true, pet: true, keep: true, aliases: []string{"alpaca"},
		use:  "🦙 @%s chegou perto da lhama e levou uma cuspida bem no meio da cara",
		gift: "🦙 @%[1]s deu uma lhama pra @%[2]s. Cuidado que ela cospe"},
	{id: "ponei", name: "Pônei", emoji: "🐴", rarity: epic, value: 2500, shop: true, pet: true, keep: true,
		use:  "🐴 @%s deu uma volta de pônei no quintal e se sentiu num filme de faroeste",
		gift: "🐴 @%[1]s deu um pônei pra @%[2]s. Realizou o sonho de criança de alguém"},

	{id: "chinelo", name: "Chinelo sem par", emoji: "🩴", rarity: common, value: 8},
	{id: "todynho-pela-metade", name: "Todynho pela metade", emoji: "🧃", rarity: common, value: 3,
		use: "🧃 @%s tomou o resto de um Todynho que achou no lixo. Tava quente. E tinha uma formiga"},
	{id: "guarda-chuva", name: "Guarda-chuva quebrado", emoji: "☂️", rarity: common, value: 12, metal: true},
	{id: "controle-remoto", name: "Controle remoto sem pilha", emoji: "📺", rarity: common, value: 15, aliases: []string{"controle"}},
	{id: "nokia", name: "Nokia tijolão", emoji: "📱", rarity: uncommon, value: 210, keep: true,
		use: "📱 @%s jogou cobrinha no Nokia tijolão por 3 horas e a bateria nem mexeu"},
	{id: "cd-pirata", name: "CD pirata do Calypso", emoji: "💿", rarity: common, value: 20, keep: true, aliases: []string{"cd", "calypso"},
		use: "💿 @%s botou o CD pirata do Calypso pra tocar e a rua inteira dançou"},
	{id: "vhs", name: "Fita VHS da Xuxa", emoji: "📼", rarity: uncommon, value: 100, aliases: []string{"fita", "vhs"}},
	{id: "cueca", name: "Cueca furada", emoji: "🩲", rarity: common, value: 3,
		use:  "🩲 @%s vestiu a cueca furada que achou no lixão. Que coragem",
		gift: "🩲 @%[1]s deu uma cueca furada de presente pra @%[2]s. O que será que isso quer dizer?"},
	{id: "cabo-usb", name: "Cabo USB que não funciona", emoji: "🔌", rarity: common, value: 8, aliases: []string{"cabo", "usb"}},
	{id: "pote-de-feijao", name: "Pote de sorvete com feijão", emoji: "🍨", rarity: common, value: 30, aliases: []string{"sorvete", "feijao"},
		use: "🍨 @%s abriu o pote de sorvete todo feliz e era feijão. De novo 😭"},
	{id: "casca-de-banana", name: "Casca de banana", emoji: "🍌", rarity: common, value: 2, aliases: []string{"banana"},
		use: "🍌 @%s jogou uma casca de banana no chão e o primeiro que passou escorregou"},
	{id: "pneu", name: "Pneu velho", emoji: "🛞", rarity: common, value: 20},
	{id: "polystation", name: "Polystation", emoji: "🎮", rarity: uncommon, value: 130, keep: true,
		use: "🎮 @%s ligou o Polystation e jogou o cartucho de 9999 jogos em 1, que na verdade eram 4"},
	{id: "tamagotchi", name: "Tamagotchi", emoji: "🥚", rarity: uncommon, value: 140, keep: true,
		use: "🥚 @%s deu comida pro Tamagotchi depois de 15 anos. Ele morreu de novo"},
	{id: "boneca", name: "Boneca sem cabeça", emoji: "🪆", rarity: common, value: 15},
	{id: "carrinho", name: "Carrinho de mercado", emoji: "🛒", rarity: uncommon, value: 210, metal: true, keep: true,
		use: "🛒 @%s desceu a ladeira dentro do carrinho de mercado gritando"},
	{id: "jornal", name: "Jornal velho", emoji: "📰", rarity: common, value: 6},
	{id: "cadeira", name: "Cadeira de plástico de bar", emoji: "🪑", rarity: common, value: 40, aliases: []string{"cadeira"}},
	{id: "nota-de-tres", name: "Nota de três reais", emoji: "💸", rarity: rare, value: 430, aliases: []string{"nota"}},
	{id: "boleto", name: "Boleto vencido", emoji: "🧾", rarity: common, value: 1,
		use:  "🧾 @%s pagou o boleto vencido com juros e multa. Que adulto responsável",
		gift: "🧾 @%[1]s passou um boleto vencido pra @%[2]s pagar. Folgado"},
	{id: "cofrinho", name: "Cofrinho de porquinho", emoji: "🐷", rarity: uncommon, value: 150, aliases: []string{"porquinho"},
		prizes: []prize{{1, 50}, {1, 100}, {1, 150}, {1, 200}, {1, 300}}, use: "🐷 @%s quebrou o cofrinho de porquinho com um martelo"},
	{id: "carteira-perdida", name: "Carteira perdida", emoji: "👛", rarity: uncommon, value: 120, aliases: []string{"carteira"},
		prizes: []prize{{50, 20}, {30, 80}, {15, 250}, {5, 800}}, use: "👛 @%s abriu a carteira perdida e achou uma foto 3x4 de um desconhecido sorrindo"},
	{id: "iphone", name: "iPhone de tela rachada", emoji: "📲", rarity: epic, value: 1_800, keep: true,
		use: "📲 @%s tirou uma selfie com o iPhone de tela rachada e saiu com uma teia de aranha na cara"},
	{id: "pombo", name: "Pombo de estimação", emoji: "🐦", rarity: common, value: 30, keep: true,
		use: "🐦 @%s deu migalha de pão pro pombo de estimação e ele voltou com mais 40 amigos"},
	{id: "rato", name: "Rato de estimação", emoji: "🐀", rarity: common, value: 12, keep: true,
		use: "🐀 @%s fez carinho no rato de estimação. Ele se chama Remy e sabe cozinhar"},
	{id: "gato-preto", name: "Gato preto", emoji: "🐈‍⬛", rarity: rare, value: 400, aliases: []string{"gato"},
		hex: "azar",
		use: "🐈‍⬛ @%[1]s soltou o gato preto bem no caminho de @%[2]s"},
	{id: "marmita", name: "Marmita da mãe", emoji: "🍱", rarity: common, value: 30,
		use: "🍱 @%s abriu a marmita da mãe e tinha arroz, feijão, bife e um bilhetinho de amor"},
	{id: "garrafa-pet", name: "Garrafa PET", emoji: "♻️", rarity: common, value: 3, aliases: []string{"garrafa"}},
	{id: "anel-de-latinha", name: "Anel de latinha", emoji: "🥫", rarity: common, value: 1, metal: true, aliases: []string{"latinha"},
		gift: "🥫 @%[1]s pediu @%[2]s em casamento com um anel de latinha. É o que tem"},
	{id: "dinheiro-de-brinquedo", name: "Dinheiro de brinquedo", emoji: "💵", rarity: common, value: 2, aliases: []string{"dinheiro falso"},
		use: "💵 @%s tentou pagar o pão com dinheiro de brinquedo e o padeiro nem riu"},
	{id: "dentadura", name: "Dentadura", emoji: "🦷", rarity: common, value: 18,
		use: "🦷 @%s botou a dentadura que achou no lixo e sorriu pro grupo. Credo"},
	{id: "peruca", name: "Peruca loira", emoji: "💇", rarity: common, value: 20, keep: true, aliases: []string{"peruca"},
		use: "💇 @%s botou a peruca loira e se apresentou como a prima que veio do interior"},
	{id: "panela-de-pressao", name: "Panela de pressão sem válvula", emoji: "🍲", rarity: common, value: 20, metal: true, aliases: []string{"panela"}},
	{id: "privada", name: "Privada", emoji: "🚽", rarity: common, value: 25, aliases: []string{"vaso sanitario"}},
	{id: "colchao", name: "Colchão de mola", emoji: "🛏️", rarity: common, value: 30, aliases: []string{"colchao"},
		use: "🛏️ @%s deitou no colchão de mola do lixão e uma mola furou a alma"},
	{id: "sofa", name: "Sofá rasgado", emoji: "🛋️", rarity: common, value: 45, aliases: []string{"sofa"}},
	{id: "tv-de-tubo", name: "TV de tubo", emoji: "🖥️", rarity: uncommon, value: 85, keep: true, aliases: []string{"tv", "televisao"},
		use: "🖥️ @%s ligou a TV de tubo, deu dois tapas do lado e pegou o canal"},
	{id: "radinho", name: "Radinho de pilha", emoji: "📻", rarity: uncommon, value: 70, keep: true, aliases: []string{"radio", "radinho"},
		use: "📻 @%s ligou o radinho de pilha e só pegou a reprise da final de 94"},
	{id: "bicicleta", name: "Bicicleta sem corrente", emoji: "🚲", rarity: uncommon, value: 100, metal: true, aliases: []string{"bicicleta", "bike", "magrela"}},
	{id: "album-da-copa", name: "Álbum da Copa incompleto", emoji: "📒", rarity: uncommon, value: 110, keep: true, aliases: []string{"album", "figurinhas"},
		use: "📒 @%s folheou o álbum da Copa e faltava só uma figurinha. Sempre falta só uma"},
	{id: "anao-de-jardim", name: "Anão de jardim", emoji: "🧙", rarity: uncommon, value: 65, keep: true, aliases: []string{"anao", "gnomo"},
		use: "🧙 @%s botou o anão de jardim na porta e agora ele vigia a casa. Ninguém entra. Ninguém sai"},
	{id: "relogio-cuco", name: "Relógio cuco", emoji: "🕰️", rarity: uncommon, value: 130, keep: true, aliases: []string{"cuco"},
		use: "🕰️ @%s deu corda no relógio cuco e o passarinho saiu gritando às 3 da manhã"},
	{id: "bilhete-esquecido", name: "Bilhete de loteria esquecido", emoji: "🎰", rarity: uncommon, value: 60, aliases: []string{"bilhete"},
		prizes: []prize{{80, 0}, {15, 100}, {5, 1000}}, use: "🎰 @%s conferiu o bilhete de loteria esquecido com o resultado de 3 anos atrás"},
	{id: "mala", name: "Mala esquecida", emoji: "🧳", rarity: uncommon, value: 150, aliases: []string{"mala"},
		loot: []int{60, 30, 9, 1, 0}, use: "🧳 @%s abriu a mala esquecida torcendo pra não ter nada ilegal dentro"},

	{id: "tampinha", name: "Tampinha de garrafa", emoji: "🔘", rarity: common, value: 2, metal: true, aliases: []string{"tampinha"}},
	{id: "pilha-usada", name: "Pilha usada", emoji: "🔋", rarity: common, value: 3, metal: true, aliases: []string{"pilha"},
		use: "🔋 @%s mordeu a pilha usada pra ver se ainda tinha carga. Tinha um pouquinho"},
	{id: "tazo", name: "Tazo do salgadinho", emoji: "💫", rarity: common, value: 12, keep: true,
		use: "💫 @%s bateu o tazo no chão e virou o do amigo. Agora é seu"},
	{id: "fita-k7", name: "Fita K7 gravada do rádio", emoji: "🎶", rarity: common, value: 18, keep: true, aliases: []string{"k7", "fitinha", "cassete"},
		use: "🎶 @%s botou a fita K7 pra tocar e o locutor falou por cima de todas as músicas"},
	{id: "tenis-furado", name: "Tênis furado", emoji: "👟", rarity: common, value: 9},
	{id: "ventilador", name: "Ventilador sem hélice", emoji: "🌀", rarity: common, value: 20, metal: true, aliases: []string{"ventilador"}},
	{id: "barbie", name: "Barbie descabelada", emoji: "👸", rarity: common, value: 25, aliases: []string{"barbie"}},
	{id: "rolex-do-paraguai", name: "Rolex do Paraguai", emoji: "⌚", rarity: common, value: 40, metal: true, aliases: []string{"rolex falso"},
		use: "⌚ @%s botou o Rolex do Paraguai pra impressionar e o ponteiro caiu"},
	{id: "manequim", name: "Manequim de loja", emoji: "🧍", rarity: uncommon, value: 55,
		use: "🧍 @%s botou o manequim na janela e o vizinho chamou a polícia"},
	{id: "urubu", name: "Filhote de urubu", emoji: "🐦‍⬛", rarity: uncommon, value: 40, keep: true, aliases: []string{"urubu"},
		use: "🐦‍⬛ @%s deu um pedaço de carne pro filhote de urubu e agora ele segue a pessoa até o ponto de ônibus"},
	{id: "disco-de-vinil", name: "Disco de vinil do Roberto Carlos", emoji: "💽", rarity: uncommon, value: 110, keep: true, aliases: []string{"vinil", "disco", "roberto carlos"},
		use: "💽 @%s botou o vinil do Roberto Carlos pra tocar e chorou ouvindo Detalhes"},
	{id: "maquina-de-escrever", name: "Máquina de escrever", emoji: "⌨️", rarity: uncommon, value: 130, keep: true, metal: true, aliases: []string{"maquina"},
		use: "⌨️ @%s escreveu uma carta inteira na máquina de escrever, errou a última letra e começou tudo de novo"},
	{id: "notebook-positivo", name: "Notebook da Positivo", emoji: "💻", rarity: uncommon, value: 140, keep: true, aliases: []string{"notebook", "positivo"},
		use: "💻 @%s ligou o notebook da Positivo e esperou 20 minutos pra abrir o Paint"},
	{id: "game-boy", name: "Game Boy", emoji: "👾", rarity: uncommon, value: 170, keep: true, aliases: []string{"gameboy"},
		use: "👾 @%s jogou Pokémon no Game Boy com a pilha acabando e esqueceu de salvar"},
	{id: "lata-de-ninho", name: "Lata de Ninho cheia de moeda", emoji: "🪙", rarity: uncommon, value: 80, metal: true, aliases: []string{"ninho", "lata"},
		prizes: []prize{{40, 20}, {35, 60}, {20, 150}, {5, 400}}, use: "🪙 @%s abriu a lata de Ninho achando que era leite e era o cofrinho de moedas da vó"},
	{id: "brinco-de-ouro", name: "Brinco de ouro perdido", emoji: "💛", rarity: rare, value: 460, metal: true, aliases: []string{"brinco"}},
	{id: "carta-holografica", name: "Carta holográfica de Pokémon", emoji: "🃏", rarity: rare, value: 580, keep: true, aliases: []string{"pokemon", "charizard"},
		use: "🃏 @%s mostrou a carta holográfica do Charizard e o grupo inteiro ficou com inveja"},
	{id: "cofre-esquecido", name: "Cofre esquecido", emoji: "🔐", rarity: epic, value: 1400, metal: true, aliases: []string{"cofre"},
		loot: []int{0, 30, 45, 20, 5}, use: "🔐 @%s passou a madrugada tentando a senha do cofre esquecido. Era 1234"},
	{id: "bilhete-premiado", name: "Bilhete premiado da Mega-Sena", emoji: "🤑", rarity: legendary, value: 5000, aliases: []string{"mega", "mega sena"},
		prizes: []prize{{50, 2000}, {35, 5000}, {15, 15000}}, use: "🤑 @%s foi na lotérica com o bilhete premiado que achou no lixo e saiu de lá gritando"},

	{id: "mala-de-dinheiro", name: "Mala de dinheiro de político", emoji: "💼", rarity: legendary, value: 6000, aliases: []string{"propina"},
		prizes: []prize{{50, 3000}, {35, 6000}, {15, 12000}}, use: "💼 @%s abriu a mala de dinheiro que achou no lixão e tinha um bilhete dentro: propina do vereador, não mexer"},
	{id: "taca-jules-rimet", name: "Taça Jules Rimet", emoji: "🏆", rarity: legendary, value: 7_700, keep: true, metal: true, aliases: []string{"jules rimet", "rimet"},
		use: "🏆 @%s levantou a taça Jules Rimet que tava no lixão. No fim das contas ninguém derreteu ela"},
	{id: "lambari", name: "Lambari", emoji: "🐟", rarity: common, value: 30},
	{id: "tilapia", name: "Tilápia", emoji: "🐟", rarity: common, value: 50,
		use: "🐟 @%s fritou a tilápia e comeu com limão e farofa"},
	{id: "camarao", name: "Camarão", emoji: "🦐", rarity: common, value: 40,
		use: "🦐 @%s fritou o camarão no alho e óleo e o prédio inteiro sentiu o cheiro"},
	{id: "caranguejo", name: "Caranguejo", emoji: "🦀", rarity: common, value: 45},
	{id: "bota-velha", name: "Bota velha", emoji: "👢", rarity: common, value: 5, aliases: []string{"bota"}},
	{id: "sapo", name: "Sapo", emoji: "🐸", rarity: common, value: 8,
		use: "🐸 @%s beijou o sapo esperando um príncipe, mas ele continuou sendo só um sapo"},
	{id: "tambaqui", name: "Tambaqui", emoji: "🐟", rarity: uncommon, value: 150},
	{id: "peixe-palhaco", name: "Peixe-palhaço", emoji: "🐠", rarity: uncommon, value: 130, keep: true, aliases: []string{"nemo"},
		use: "🐠 @%s olhou bem pro peixe-palhaço e confirmou: é o Nemo. Achamos ele"},
	{id: "baiacu", name: "Baiacu", emoji: "🐡", rarity: uncommon, value: 100,
		use: "🐡 @%s comeu um baiacu preparado em casa e... tá vivo? Tá vivo. Foi por pouco"},
	{id: "lula", name: "Lula", emoji: "🦑", rarity: uncommon, value: 140},
	{id: "polvo", name: "Polvo", emoji: "🐙", rarity: uncommon, value: 180},
	{id: "pirarucu", name: "Pirarucu", emoji: "🐟", rarity: rare, value: 580},
	{id: "lagosta", name: "Lagosta", emoji: "🦞", rarity: rare, value: 650,
		use: "🦞 @%s comeu lagosta na manteiga e agora fala que é da elite"},
	{id: "jacare", name: "Jacaré", emoji: "🐊", rarity: rare, value: 840, keep: true,
		use: "🐊 @%s levou o jacaré pra passear na coleira e a rua inteira trancou o portão"},
	{id: "perola", name: "Pérola", emoji: "🦪", rarity: rare, value: 910},
	{id: "tubarao", name: "Tubarão", emoji: "🦈", rarity: epic, value: 1_550},
	{id: "bau-do-tesouro", name: "Baú do tesouro", emoji: "🏴‍☠️", rarity: epic, value: 1800, aliases: []string{"bau", "tesouro"},
		loot: []int{0, 0, 50, 40, 10}, use: "🏴‍☠️ @%s arrombou o cadeado enferrujado do baú do tesouro"},
	{id: "escama-de-sereia", name: "Escama de sereia", emoji: "🧜‍♀️", rarity: legendary, value: 5_500, aliases: []string{"sereia", "escama"}},
	{id: "sardinha", name: "Sardinha", emoji: "🐟", rarity: common, value: 15,
		use: "🐟 @%s comeu a sardinha com pão e chamou de rodízio de frutos do mar"},
	{id: "traira", name: "Traíra", emoji: "🐟", rarity: common, value: 30,
		gift: "🐟 @%[1]s deu uma traíra pra @%[2]s. Será que foi indireta?"},
	{id: "pacu", name: "Pacu", emoji: "🐟", rarity: common, value: 60,
		use: "🐟 @%s assou o pacu na brasa e chamou a família toda"},
	{id: "estrela-do-mar", name: "Estrela-do-mar", emoji: "⭐", rarity: common, value: 40, keep: true, aliases: []string{"estrela"},
		use: "⭐ @%s colou a estrela-do-mar na testa e virou o Patrick"},
	{id: "celular-afogado", name: "Celular afogado", emoji: "📵", rarity: common, value: 20,
		use: "📵 @%s botou o celular afogado no arroz e rezou. Não voltou"},
	{id: "piranha", name: "Piranha", emoji: "🐟", rarity: uncommon, value: 85,
		use: "🐟 @%s tentou fazer carinho na piranha e agora digita com 9 dedos"},
	{id: "agua-viva", name: "Água-viva", emoji: "🪼", rarity: uncommon, value: 70, aliases: []string{"agua viva", "medusa"},
		use: "🪼 @%s encostou na água-viva e alguém vai ter que fazer xixi em cima. Voluntários?"},
	{id: "atum", name: "Atum", emoji: "🐟", rarity: uncommon, value: 170,
		use: "🐟 @%s fez um sushi de atum em casa e ficou parecendo um acidente"},
	{id: "arraia", name: "Arraia", emoji: "🐟", rarity: uncommon, value: 210, aliases: []string{"raia"}},
	{id: "garrafa-com-mensagem", name: "Garrafa com mensagem", emoji: "📜", rarity: uncommon, value: 130, aliases: []string{"mensagem"},
		use: "📜 @%s abriu a garrafa com mensagem e era uma carta de 1998 pedindo pra ligar num número que não existe mais"},
	{id: "dourado", name: "Dourado", emoji: "🐟", rarity: rare, value: 490},
	{id: "peixe-espada", name: "Peixe-espada", emoji: "🗡️", rarity: rare, value: 390},
	{id: "tridente", name: "Tridente de Netuno", emoji: "🔱", rarity: legendary, value: 6_600, keep: true, aliases: []string{"tridente", "netuno"},
		use: "🔱 @%s levantou o tridente e o mar ficou agitado até na piscina do prédio"},

	{id: "dente-de-megalodonte", name: "Dente de megalodonte", emoji: "🦷", rarity: legendary, value: 6_050, aliases: []string{"megalodonte", "megalodon"}},
	{id: "boto-cor-de-rosa", name: "Boto-cor-de-rosa", emoji: "🐬", rarity: legendary, value: 7_700, keep: true, aliases: []string{"boto"},
		use: "🐬 @%s levou o boto-cor-de-rosa pra festa junina e ele saiu de lá de chapéu, namorando metade da cidade"},
	{id: "pedra", name: "Pedra", emoji: "🪨", rarity: common, value: 3, keep: true,
		use: "🪨 @%s ficou 10 minutos olhando pra pedra. Ela não fez nada. Que pedra boa"},
	{id: "carvao", name: "Carvão", emoji: "⚫", rarity: common, value: 20,
		use: "⚫ @%s acendeu a churrasqueira com o carvão. Agora só falta a carne (e o dinheiro pra carne)"},
	{id: "ferro", name: "Minério de ferro", emoji: "🔩", rarity: common, value: 60, metal: true, aliases: []string{"ferro"}},
	{id: "cobre", name: "Cobre", emoji: "🟤", rarity: common, value: 45, metal: true},
	{id: "sal-grosso", name: "Sal grosso", emoji: "🧂", rarity: common, value: 8, aliases: []string{"sal"},
		bless: "purificacao",
		use:   "🧂 @%s jogou sal grosso nos quatro cantos da casa. Inveja nenhuma entra mais"},
	{id: "prata", name: "Prata", emoji: "🥈", rarity: uncommon, value: 180, metal: true},
	{id: "ametista", name: "Ametista", emoji: "💜", rarity: uncommon, value: 280},
	{id: "caveira", name: "Caveira", emoji: "💀", rarity: uncommon, value: 80,
		hex: "encosto",
		use: "💀 @%[1]s enterrou a caveira no quintal de @%[2]s numa noite de lua cheia"},
	{id: "ouro", name: "Pepita de ouro", emoji: "🥇", rarity: rare, value: 520, metal: true, aliases: []string{"ouro", "pepita"}},
	{id: "esmeralda", name: "Esmeralda", emoji: "💚", rarity: rare, value: 780},
	{id: "rubi", name: "Rubi", emoji: "❤️", rarity: rare, value: 910},
	{id: "safira", name: "Safira", emoji: "💙", rarity: rare, value: 840},
	{id: "vaso-antigo", name: "Vaso antigo", emoji: "🏺", rarity: rare, value: 720, aliases: []string{"vaso"}},
	{id: "moeda-do-imperio", name: "Moeda do Império", emoji: "🏅", rarity: rare, value: 580, metal: true, aliases: []string{"moeda"}},
	{id: "diamante", name: "Diamante", emoji: "💎", rarity: epic, value: 3_000},
	{id: "fossil", name: "Fóssil de dinossauro", emoji: "🦴", rarity: epic, value: 1_900, aliases: []string{"fossil", "dinossauro"}},
	{id: "meteorito", name: "Meteorito", emoji: "☄️", rarity: legendary, value: 4_950, metal: true},
	{id: "areia", name: "Areia", emoji: "🏖️", rarity: common, value: 5,
		use: "🏖️ @%s jogou areia pro alto e agora tem areia até no ouvido"},
	{id: "granito", name: "Granito", emoji: "🪨", rarity: common, value: 15},
	{id: "quartzo", name: "Quartzo", emoji: "🔹", rarity: common, value: 50, keep: true,
		use: "🔹 @%s botou o quartzo embaixo do travesseiro pra limpar as energias"},
	{id: "ouro-de-tolo", name: "Ouro de tolo", emoji: "🟨", rarity: common, value: 8, metal: true, aliases: []string{"pirita"},
		use: "🟨 @%s tentou vender o ouro de tolo pro ourives e foi expulso da loja"},
	{id: "geodo", name: "Geodo", emoji: "🥚", rarity: uncommon, value: 300,
		prizes: []prize{{40, 100}, {35, 250}, {20, 600}, {5, 1500}}, use: "🥚 @%s quebrou o geodo no meio com o martelo"},
	{id: "topazio", name: "Topázio", emoji: "🧡", rarity: uncommon, value: 310},
	{id: "opala", name: "Opala", emoji: "🤍", rarity: rare, value: 650},
	{id: "jade", name: "Jade", emoji: "🟢", rarity: rare, value: 720},
	{id: "petroleo", name: "Barril de petróleo", emoji: "🛢️", rarity: rare, value: 620, aliases: []string{"petroleo", "barril"}},
	{id: "urna", name: "Urna funerária", emoji: "⚱️", rarity: rare, value: 680, aliases: []string{"urna"}},
	{id: "uranio", name: "Urânio", emoji: "☢️", rarity: epic, value: 2_400, metal: true, keep: true,
		use: "☢️ @%s segurou o urânio com a mão. Agora brilha no escuro"},

	{id: "diamante-rosa", name: "Diamante rosa", emoji: "💗", rarity: legendary, value: 6_600},
	{id: "idolo-de-ouro", name: "Ídolo de ouro", emoji: "🗿", rarity: legendary, value: 7_150, metal: true, aliases: []string{"idolo"}},
	{id: "pedra-filosofal", name: "Pedra filosofal", emoji: "🔴", rarity: legendary, value: 8_800, keep: true, aliases: []string{"filosofal"},
		use: "🔴 @%s encostou a pedra filosofal no prato de feijão e ele virou ouro. Agora não dá pra comer"},
	{id: "folha-seca", name: "Folha seca", emoji: "🍂", rarity: common, value: 2, aliases: []string{"folha"}},
	{id: "teia-de-aranha", name: "Teia de aranha", emoji: "🕸️", rarity: common, value: 3, aliases: []string{"teia"}},
	{id: "pernilongo", name: "Pernilongo", emoji: "🦟", rarity: common, value: 2, aliases: []string{"muriçoca", "carapanã"},
		use: "🦟 @%s soltou o pernilongo no quarto de alguém às 3 da manhã. Crueldade"},
	{id: "mosca-varejeira", name: "Mosca varejeira", emoji: "🪰", rarity: common, value: 3, aliases: []string{"mosca"}},
	{id: "formiga", name: "Formiga", emoji: "🐜", rarity: common, value: 5,
		use: "🐜 @%s soltou a formiga no açucareiro e ela voltou com a família inteira"},
	{id: "barata", name: "Barata voadora", emoji: "🪳", rarity: common, value: 3, aliases: []string{"barata"},
		use: "🪳 @%s soltou a barata voadora e ela foi direto na cara de alguém. O grupo inteiro gritou"},
	{id: "mosquito-da-dengue", name: "Mosquito da dengue", emoji: "🦟", rarity: common, value: 2, aliases: []string{"dengue", "aedes"},
		use: "🦟 @%s criou o mosquito da dengue no pratinho da planta. O agente de saúde já tá batendo no portão"},
	{id: "grilo", name: "Grilo", emoji: "🦗", rarity: common, value: 12, keep: true,
		use: "🦗 @%s deixou o grilo do lado da cama e passou a noite inteira ouvindo cri-cri"},
	{id: "caracol", name: "Caracol", emoji: "🐌", rarity: common, value: 9, keep: true, aliases: []string{"lesma"},
		use: "🐌 @%s apostou corrida com o caracol e perdeu"},
	{id: "lagarta", name: "Taturana", emoji: "🐛", rarity: common, value: 11, aliases: []string{"lagarta"},
		use: "🐛 @%s encostou na taturana e a mão tá ardendo até agora"},
	{id: "joaninha", name: "Joaninha", emoji: "🐞", rarity: common, value: 20,
		gift: "🐞 @%[1]s deu uma joaninha pra @%[2]s. Dizem que dá sorte"},
	{id: "mariposa", name: "Mariposa", emoji: "🦋", rarity: common, value: 15, aliases: []string{"bruxa"},
		use: "🦋 @%s soltou a mariposa dentro de casa e a família inteira jurou que era bruxa"},
	{id: "tanajura", name: "Tanajura", emoji: "🐜", rarity: common, value: 20, aliases: []string{"içá"},
		use: "🐜 @%s fritou a tanajura com farinha, igual o vô fazia no interior"},
	{id: "abelha", name: "Abelha", emoji: "🐝", rarity: common, value: 20,
		use: "🐝 @%s tentou fazer carinho na abelha e levou uma ferroada no dedo"},
	{id: "borboleta", name: "Borboleta", emoji: "🦋", rarity: uncommon, value: 60,
		gift: "🦋 @%[1]s deu uma borboleta pra @%[2]s. Tô sentindo um friozinho na barriga"},
	{id: "cigarra", name: "Cigarra", emoji: "🦗", rarity: uncommon, value: 65,
		use: "🦗 @%s botou a cigarra pra cantar e ela gritou até estourar"},
	{id: "vaga-lume", name: "Vaga-lume", emoji: "✨", rarity: uncommon, value: 70, keep: true, aliases: []string{"vagalume", "pirilampo"},
		use: "✨ @%s soltou o vaga-lume no quarto escuro e foi a coisa mais bonita da semana"},
	{id: "casa-de-marimbondo", name: "Casa de marimbondo", emoji: "🪺", rarity: uncommon, value: 55, aliases: []string{"marimbondo", "vespa"},
		use: "🪺 @%s cutucou a casa de marimbondo com um pau. Foi o último dia com o rosto normal"},
	{id: "louva-a-deus", name: "Louva-a-deus", emoji: "🙏", rarity: uncommon, value: 100, keep: true,
		use: "🙏 @%s ficou encarando o louva-a-deus e ele rezou pelo grupo inteiro"},
	{id: "bicho-pau", name: "Bicho-pau", emoji: "🪵", rarity: uncommon, value: 85, keep: true,
		use: "🪵 @%s mandou a foto do bicho-pau no grupo e até agora ninguém achou ele"},
	{id: "favo-de-mel", name: "Favo de mel", emoji: "🍯", rarity: uncommon, value: 130, aliases: []string{"mel"},
		use: "🍯 @%s comeu o favo de mel com cera e tudo e lambeu os dedos"},
	{id: "escorpiao", name: "Escorpião amarelo", emoji: "🦂", rarity: uncommon, value: 110, aliases: []string{"escorpiao"}},
	{id: "caranguejeira", name: "Aranha caranguejeira", emoji: "🕷️", rarity: uncommon, value: 170, keep: true, aliases: []string{"aranha", "tarantula"},
		use: "🕷️ @%s deixou a caranguejeira passear no braço e metade do grupo saiu"},
	{id: "besouro-rinoceronte", name: "Besouro-rinoceronte", emoji: "🪲", rarity: rare, value: 520, aliases: []string{"besouro"}},
	{id: "casulo-de-seda", name: "Casulo de bicho-da-seda", emoji: "🧵", rarity: rare, value: 460, aliases: []string{"casulo", "bicho da seda"}},
	{id: "borboleta-azul", name: "Borboleta-azul", emoji: "🦋", rarity: rare, value: 780, keep: true, aliases: []string{"morpho"},
		use: "🦋 @%s soltou a borboleta-azul e ela pousou bem no nariz de alguém"},
	{id: "abelha-rainha", name: "Abelha-rainha", emoji: "🐝", rarity: epic, value: 1_800, aliases: []string{"rainha"}},
	{id: "mosquito-no-ambar", name: "Mosquito no âmbar", emoji: "🟠", rarity: epic, value: 2_400, keep: true, aliases: []string{"ambar"},
		use: "🟠 @%s olhou o mosquito no âmbar contra a luz e começou a planejar um parque de dinossauros"},
	{id: "escaravelho-de-ouro", name: "Escaravelho de ouro", emoji: "🪲", rarity: legendary, value: 5_500, aliases: []string{"escaravelho"}},
	{id: "fada", name: "Fada no pote", emoji: "🧚", rarity: legendary, value: 7_700, keep: true,
		use: "🧚 @%s abriu o pote e a fada saiu voando e jogou purpurina no grupo inteiro"},

	{id: "grilo-falante", name: "Grilo falante", emoji: "🦗", rarity: legendary, value: 6_600, keep: true,
		use: "🦗 @%s perguntou pro grilo falante se devia mandar mensagem pro ex e ele respondeu que não"},
	{id: "saci-na-garrafa", name: "Saci na garrafa", emoji: "🌪️", rarity: legendary, value: 8_250, keep: true, aliases: []string{"saci"},
		use: "🌪️ @%s abriu a garrafa e o saci saiu num redemoinho, deu nó no cadarço de todo mundo e sumiu com o cachimbo do vô"},
	{id: "perfume", name: "Perfume do Paraguai", emoji: "🧴", rarity: common, value: 25, aliases: []string{"perfume"},
		use: "🧴 @%s passou o perfume do Paraguai e o cheiro durou exatos 4 minutos"},
	{id: "fone", name: "Fone que só funciona de um lado", emoji: "🎧", rarity: common, value: 20, keep: true, aliases: []string{"fone"},
		use: "🎧 @%s botou o fone que só funciona de um lado e ouviu metade da música"},
	{id: "oculos-juliet", name: "Óculos Juliet", emoji: "🕶️", rarity: uncommon, value: 160, keep: true, aliases: []string{"juliet", "oculos"},
		use: "🕶️ @%s botou o óculos Juliet e já se sente o dono do baile"},
	{id: "camisa-do-brasil", name: "Camisa do Brasil", emoji: "👕", rarity: uncommon, value: 120, keep: true, aliases: []string{"camisa"},
		use: "👕 @%s vestiu a camisa do Brasil e já tá chorando no hino"},
	{id: "terco", name: "Terço da vó", emoji: "📿", rarity: uncommon, value: 60, aliases: []string{"terco"},
		bless: "prosperidade",
		use:   "📿 @%s rezou um terço inteiro pedindo pra ficar rico. Confere o !saldo daqui a pouco"},
	{id: "olho-grego", name: "Olho grego", emoji: "🧿", rarity: uncommon, value: 80,
		bless: "protecao",
		use:   "🧿 @%s pendurou o olho grego no pescoço e agora o mau-olhado bate e volta"},
	{id: "pudim", name: "Pudim da vó", emoji: "🍮", rarity: uncommon, value: 90, aliases: []string{"pudim"},
		use: "🍮 @%s comeu o pudim da vó, com furinho e tudo, e chorou de emoção"},
	{id: "videogame", name: "Videogame destravado", emoji: "🕹️", rarity: rare, value: 600, keep: true, aliases: []string{"ps2", "play"},
		use: "🕹️ @%s ligou o videogame destravado e jogou o futebol de 2006 com o elenco atualizado por um fã"},
	{id: "guitarra", name: "Guitarra elétrica", emoji: "🎸", rarity: rare, value: 800, keep: true, aliases: []string{"guitarra"},
		use: "🎸 @%s tocou Wonderwall na guitarra elétrica. Ninguém pediu"},
	{id: "papagaio", name: "Papagaio que xinga", emoji: "🦜", rarity: rare, value: 500, keep: true, aliases: []string{"papagaio"},
		use: "🦜 @%s pediu pro papagaio falar alguma coisa e ele xingou o grupo inteiro"},
	{id: "picanha", name: "Picanha", emoji: "🥩", rarity: rare, value: 350,
		use: "🥩 @%s assou uma picanha no churrasco e virou a pessoa mais amada do bairro"},
	{id: "fusca", name: "Fusca azul", emoji: "🚙", rarity: epic, value: 3000, keep: true, aliases: []string{"fusca"},
		use: "🚙 @%s saiu de Fusca azul e todo mundo que viu deu um soquinho no braço do amigo"},
	{id: "caramelo", name: "Caramelo", emoji: "🐕", rarity: epic, value: 2000, keep: true, aliases: []string{"cachorro", "vira-lata"},
		use: "🐕 @%s fez carinho no Caramelo e ele balançou o rabo tão forte que derrubou a mesa"},
	{id: "taca-do-penta", name: "Réplica da taça do Penta", emoji: "🏆", rarity: epic, value: 1800, keep: true, aliases: []string{"taca", "penta"},
		use: "🏆 @%s levantou a réplica da taça do Penta e gritou É PENTA da janela"},
	{id: "coroa", name: "Coroa do rei do camarote", emoji: "👑", rarity: epic, value: 2200, keep: true, aliases: []string{"coroa", "camarote"},
		use: "👑 @%s botou a coroa do rei do camarote e pediu 3 garrafas de champanhe com vela de faísca"},
	{id: "unicornio", name: "Unicórnio", emoji: "🦄", rarity: legendary, value: 9000, keep: true, aliases: []string{"unicornio"},
		use: "🦄 @%s montou no unicórnio e saiu galopando num arco-íris"},
	{id: "ovo-de-dragao", name: "Ovo de dragão", emoji: "🐉", rarity: legendary, value: 10000, keep: true, aliases: []string{"dragao", "ovo"},
		use: "🐉 @%s tentou chocar o ovo de dragão no calor do sovaco. Ainda não nasceu"},
	{id: "terreno-na-lua", name: "Terreno na lua", emoji: "🌕", rarity: legendary, value: 7000, keep: true, aliases: []string{"lua", "terreno"},
		use: "🌕 @%s apontou pro céu e mostrou pro grupo: tá vendo aquele pedacinho ali? É meu"},
	{id: "pipa", name: "Pipa", emoji: "🪁", rarity: common, value: 10, keep: true, aliases: []string{"papagaio de papel", "raia de papel"},
		use: "🪁 @%s soltou pipa na laje e o vizinho cortou na primeira"},
	{id: "bola", name: "Bola de capotão", emoji: "⚽", rarity: common, value: 30, keep: true, aliases: []string{"bola"},
		use: "⚽ @%s chutou a bola de capotão e quebrou a janela do vizinho. Corre!"},
	{id: "espada-de-sao-jorge", name: "Espada-de-são-jorge", emoji: "🪴", rarity: common, value: 20, keep: true, aliases: []string{"sao jorge"},
		use: "🪴 @%s botou uma espada-de-são-jorge na porta. Mau-olhado não passa"},
	{id: "pandeiro", name: "Pandeiro", emoji: "🪘", rarity: uncommon, value: 180, keep: true,
		use: "🪘 @%s batucou no pandeiro e o grupo virou roda de samba"},
	{id: "skate", name: "Skate", emoji: "🛹", rarity: uncommon, value: 200, keep: true,
		use: "🛹 @%s tentou dar um ollie no skate e beijou o asfalto"},
	{id: "chapeu-de-cangaceiro", name: "Chapéu de cangaceiro", emoji: "🤠", rarity: rare, value: 450, keep: true, aliases: []string{"chapeu", "cangaceiro"},
		use: "🤠 @%s botou o chapéu de cangaceiro e saiu falando arretado"},
	{id: "cartola", name: "Cartola de mágico", emoji: "🎩", rarity: rare, value: 500, keep: true, aliases: []string{"cartola"},
		use: "🎩 @%s enfiou a mão na cartola e tirou um lenço que não acabava nunca"},
	{id: "sanfona", name: "Sanfona", emoji: "🪗", rarity: rare, value: 700, keep: true, aliases: []string{"acordeon", "gaita"},
		use: "🪗 @%s puxou a sanfona e começou um forró no meio do grupo"},
	{id: "trevo", name: "Trevo de quatro folhas", emoji: "🍀", rarity: rare, value: 777, aliases: []string{"trevo"},
		bless: "sorte",
		use:   "🍀 @%s guardou o trevo de quatro folhas na capinha do celular e a sorte já bateu na porta"},
	{id: "bola-de-cristal", name: "Bola de cristal", emoji: "🔮", rarity: epic, value: 1600, keep: true,
		use: "🔮 @%s olhou na bola de cristal e viu o próprio !saldo daqui a um mês. Melhor nem comentar"},
	{id: "varinha", name: "Varinha mágica", emoji: "🪄", rarity: epic, value: 2000, keep: true, aliases: []string{"varinha"},
		use: "🪄 @%s balançou a varinha e falou umas palavras em latim. Nada aconteceu, mas o clima ficou outro"},
	{id: "lampada-magica", name: "Lâmpada mágica", emoji: "🪔", rarity: legendary, value: 8000, keep: true, aliases: []string{"lampada", "genio"},
		use: "🪔 @%s esfregou a lâmpada mágica e o gênio saiu pedindo um Pix"},
	{id: "espada-lendaria", name: "Espada lendária", emoji: "⚔️", rarity: legendary, value: 8500, keep: true, aliases: []string{"espada"},
		use: "⚔️ @%s puxou a espada lendária da pedra e agora se acha o rei do grupo"},
	{id: "disco-voador", name: "Disco voador", emoji: "🛸", rarity: legendary, value: 9500, keep: true, aliases: []string{"ovni", "nave"},
		use: "🛸 @%s deu um rolê de disco voador e voltou com um ET de carona"},

	{id: "pochete", name: "Pochete do corre", emoji: "👝", rarity: rare, value: 1_200, shop: true, aliases: []string{"pochete"},
		trade: &trade{area: areaBicos}},
	{id: "caixa-de-ferramentas", name: "Caixa de ferramentas", emoji: "🧰", rarity: rare, value: 1_800, shop: true,
		trade: &trade{area: areaObras}},
	{id: "gps", name: "GPS de painel", emoji: "🧭", rarity: rare, value: 1_600, shop: true, aliases: []string{"waze"},
		trade: &trade{area: areaTransporte}},
	{id: "panela-de-ferro", name: "Panela de ferro", emoji: "🍳", rarity: rare, value: 1_600, shop: true, aliases: []string{"frigideira"},
		trade: &trade{area: areaComida}},
	{id: "teclado-mecanico", name: "Teclado mecânico", emoji: "⌨️", rarity: rare, value: 2_000, shop: true, aliases: []string{"teclado"},
		trade: &trade{area: areaEscritorio}},
	{id: "estetoscopio", name: "Estetoscópio", emoji: "🩺", rarity: rare, value: 2_500, shop: true,
		trade: &trade{area: areaSaude}},
	{id: "ring-light", name: "Ring light", emoji: "💡", rarity: rare, value: 1_800, shop: true, aliases: []string{"ringlight"},
		trade: &trade{area: areaArte}},
	{id: "chapeu-de-palha", name: "Chapéu de palha", emoji: "👒", rarity: rare, value: 1_200, shop: true,
		trade: &trade{area: areaCampo}},
	{id: "maquininha", name: "Maquininha de cartão", emoji: "💳", rarity: rare, value: 1_600, shop: true, aliases: []string{"maquina de cartao"},
		trade: &trade{area: areaComercio}},
	{id: "radio-comunicador", name: "Rádio comunicador", emoji: "📟", rarity: rare, value: 2_000, shop: true, aliases: []string{"comunicador", "walkie talkie", "ht"},
		trade: &trade{area: areaSeguranca}},
	{id: "touca-ninja", name: "Touca ninja", emoji: "🥷", rarity: rare, value: 2_500, shop: true, aliases: []string{"balaclava", "touca"},
		trade: &trade{area: areaCrime}},
	{id: "kit-de-limpeza", name: "Kit de limpeza", emoji: "🧽", rarity: rare, value: 1_200, shop: true, aliases: []string{"limpeza", "esponja"},
		trade: &trade{area: areaServicos}},
	{id: "vela-de-sete-dias", name: "Vela de sete dias", emoji: "🕯️", rarity: rare, value: 1_200, shop: true, aliases: []string{"vela"},
		trade: &trade{area: areaFe}},

	{id: "cupom-vencido", name: "Cupom de desconto vencido", emoji: "🏷️", rarity: common, value: 1, aliases: []string{"cupom"},
		use: "🏷️ @%s tentou usar o cupom de desconto vencido e o caixa riu na cara"},
	{id: "tijolo", name: "Tijolo baiano", emoji: "🧱", rarity: common, value: 4,
		gift: "🧱 @%[1]s deu um tijolo pra @%[2]s. Já é o começo da casa própria"},
	{id: "multa", name: "Multa de trânsito", emoji: "🚦", rarity: common, value: 1,
		gift: "🚦 @%[1]s passou a multa de trânsito pra @%[2]s pagar. Os pontos na carteira também"},
	{id: "gorjeta", name: "Envelope de gorjeta", emoji: "✉️", rarity: uncommon, value: 60, aliases: []string{"envelope"},
		prizes: []prize{{30, 20}, {40, 50}, {25, 100}, {5, 400}}, use: "✉️ @%s abriu o envelope de gorjeta que o cliente deixou"},
	{id: "vale-refeicao", name: "Vale-refeição", emoji: "🍽️", rarity: uncommon, value: 80, aliases: []string{"vr", "vale"},
		prizes: []prize{{40, 30}, {40, 60}, {15, 120}, {5, 300}}, use: "🍽️ @%s passou o vale-refeição no restaurante por quilo e torceu pra ter saldo"},
	{id: "grampeador", name: "Grampeador vermelho", emoji: "📎", rarity: common, value: 10, keep: true,
		use: "📎 @%s grampeou o próprio dedo tentando provar que o grampeador tava vazio"},
	{id: "pendrive", name: "Pendrive achado na firma", emoji: "💾", rarity: uncommon, value: 50, keep: true,
		use: "💾 @%s abriu o pendrive achado na firma e só tinha foto do churrasco de 2012"},
	{id: "atestado", name: "Atestado médico", emoji: "📄", rarity: uncommon, value: 100,
		gift: "📄 @%[1]s deu um atestado médico pra @%[2]s faltar no trampo amanhã. Ninguém vai saber"},
	{id: "pirulito", name: "Pirulito do consultório", emoji: "🍭", rarity: common, value: 3, aliases: []string{"pirulito"},
		use: "🍭 @%s ganhou um pirulito depois da vacina e parou de chorar na hora"},
	{id: "ingresso-de-camarote", name: "Ingresso de camarote", emoji: "🎪", rarity: rare, value: 400, aliases: []string{"ingresso", "camarote vip"},
		gift: "🎪 @%[1]s deu um ingresso de camarote pra @%[2]s. Vai ter open bar"},
	{id: "autografo", name: "Autógrafo de famoso", emoji: "✍️", rarity: rare, value: 500, keep: true, aliases: []string{"autografo"},
		use: "✍️ @%s mostrou o autógrafo de famoso pro grupo. Ninguém sabe quem é"},
	{id: "saco-de-laranja", name: "Saco de laranja", emoji: "🍊", rarity: common, value: 20, aliases: []string{"laranja"},
		use: "🍊 @%s chupou seis laranjas de uma vez e agora tá com azia"},
	{id: "mercadoria-encalhada", name: "Caixa de mercadoria encalhada", emoji: "🗃️", rarity: uncommon, value: 150, aliases: []string{"encalhada", "mercadoria"},
		loot: []int{60, 30, 9, 1, 0}, use: "🗃️ @%s abriu a caixa de mercadoria encalhada do estoque pra ver por que ninguém comprou"},
	{id: "algema", name: "Algema", emoji: "⛓️", rarity: uncommon, value: 120, keep: true, metal: true,
		use: "⛓️ @%s se algemou de brincadeira e perdeu a chave"},
	{id: "documento-falso", name: "Documento falso", emoji: "🪪", rarity: rare, value: 350, aliases: []string{"rg falso", "identidade falsa"},
		use: "🪪 @%s mostrou o documento falso na balada e o segurança viu que a foto era do Faustão"},
	{id: "agua-benta", name: "Água benta", emoji: "💧", rarity: rare, value: 300, aliases: []string{"agua"},
		bless: "protecao",
		use:   "💧 @%s se benzeu com água benta e sentiu o corpo fechar"},

	{id: "champanhe", name: "Champanhe Cristal", emoji: "🍾", rarity: epic, value: 3_000, luxury: true, aliases: []string{"champagne", "cristal"},
		bless: "sorte",
		use:   "🍾 @%s estourou um Champanhe Cristal e deu banho no camarote inteiro"},
	{id: "cordao-de-ouro", name: "Cordão de ouro", emoji: "🔗", rarity: epic, value: 4_000, keep: true, luxury: true, aliases: []string{"cordao"},
		flex: &flex{effect: "o !daily rende 25% a mais", apply: func(p Profession) Profession { p.dailyBonus += 25; return p }},
		use:  "🔗 @%s botou o cordão de ouro por cima da camisa, do jeito certo"},
	{id: "bolsa-de-grife", name: "Bolsa de grife", emoji: "👜", rarity: epic, value: 5_000, keep: true, luxury: true, aliases: []string{"grife"},
		flex: &flex{effect: "10% de chance de ganhar em dobro no !trabalhar (não soma com a da profissão nem com a da bênção, vale a maior)", apply: func(p Profession) Profession { p.doubleChance = max(p.doubleChance, 0.1); return p }},
		use:  "👜 @%s desfilou com a bolsa de grife e fez questão de falar o preço pra todo mundo"},
	{id: "rolex", name: "Rolex de ouro", emoji: "⌚", rarity: epic, value: 6_000, keep: true, luxury: true, aliases: []string{"relogio"},
		flex: &flex{effect: "o !trabalhar recarrega 25% mais rápido", apply: func(p Profession) Profession { p.every = p.every * 3 / 4; return p }},
		use:  "⌚ @%s olhou as horas no Rolex de ouro só pra todo mundo ver o pulso"},
	{id: "quadro", name: "Quadro original de museu", emoji: "🖼️", rarity: legendary, value: 12_000, keep: true, luxury: true, aliases: []string{"pintura", "obra de arte"},
		flex: &flex{effect: "o !trabalhar paga 15% a mais", apply: func(p Profession) Profession { p.min, p.max = p.min*115/100, p.max*115/100; return p }},
		use:  "🖼️ @%s pendurou o quadro original na sala e agora cobra ingresso pra visita"},
	{id: "lamborghini", name: "Lamborghini", emoji: "🏎️", rarity: legendary, value: 20_000, keep: true, luxury: true, aliases: []string{"lambo"},
		flex: &flex{effect: "+15% de chance no !fugir da cadeia e do cativeiro e de escapar quando o !assalto dá errado", apply: func(p Profession) Profession { p.escapeBonus += 0.15; p.breakoutBonus += 0.15; return p }},
		use:  "🏎️ @%s acelerou a Lamborghini no quebra-molas e ralou o fundo inteiro"},
	{id: "iate", name: "Iate", emoji: "🛥️", rarity: legendary, value: 40_000, keep: true, luxury: true,
		flex: &flex{effect: "a poupança rende 2 pontos a mais", apply: func(p Profession) Profession { p.interest += 2; return p }},
		use:  "🛥️ @%s deu uma festa no iate e descobriu que ninguém sabe nadar"},
	{id: "jatinho", name: "Jatinho particular", emoji: "🛩️", rarity: legendary, value: 60_000, keep: true, luxury: true, aliases: []string{"jato", "aviao"},
		flex: &flex{effect: "o seu voto conta 1 a mais no !votar, no !impeachment e no júri do !processar", apply: func(p Profession) Profession { p.voteWeight = p.ballotWeight() + 1; return p }},
		use:  "🛩️ @%s foi de jatinho comprar pão na padaria da esquina"},
	{id: "caviar", name: "Lata de caviar", emoji: "🫙", rarity: epic, value: 3_500, luxury: true, aliases: []string{"caviar"},
		bless: "prosperidade",
		use:   "🫙 @%s comeu caviar de colher e fez cara de quem gostou"},
	{id: "vinho", name: "Vinho de safra rara", emoji: "🍷", rarity: epic, value: 3_500, luxury: true, aliases: []string{"vinho"},
		bless: "fartura",
		use:   "🍷 @%s girou a taça, cheirou, provou e disse que tinha notas de carvalho. Ninguém sabe o que é isso"},
	{id: "tenis-de-grife", name: "Tênis de grife", emoji: "👟", rarity: epic, value: 4_500, keep: true, luxury: true, aliases: []string{"tenis"},
		flex: &flex{effect: "+8% de chance no !roubar e no !furtar", apply: func(p Profession) Profession { p.robBonus += 0.08; p.theftBonus += 0.08; return p }},
		use:  "👟 @%s pisou numa poça com o tênis de grife e quis processar a chuva"},
	{id: "anel-de-diamante", name: "Anel de diamante", emoji: "💍", rarity: epic, value: 7_000, keep: true, luxury: true,
		flex: &flex{effect: "conta como 1 voto a mais pro seu lado no júri do !processar", apply: func(p Profession) Profession { p.trialEdge++; return p }},
		use:  "💍 @%s ficou girando o anel de diamante pro sol bater e cegar todo mundo",
		gift: "💍 @%[1]s ajoelhou e abriu uma caixinha com um anel de diamante pra @%[2]s. O grupo inteiro prendeu a respiração"},
	{id: "cavalo-de-raca", name: "Cavalo de raça", emoji: "🐎", rarity: legendary, value: 25_000, keep: true, luxury: true, aliases: []string{"cavalo"},
		flex: &flex{effect: "15% de volta quando perde no !apostar e no !cassino", apply: func(p Profession) Profession { p.cashback += 15; return p }},
		use:  "🐎 @%s desfilou no cavalo de raça na frente da padaria"},
	{id: "helicoptero", name: "Helicóptero", emoji: "🚁", rarity: legendary, value: 30_000, keep: true, luxury: true,
		flex: &flex{effect: "+12% de chance pra equipe no !assalto, sem gastar o helicóptero", apply: func(p Profession) Profession { p.heistBonus += 0.12; return p }},
		use:  "🚁 @%s pousou o helicóptero no telhado do prédio e o síndico já tá escrevendo a advertência"},
	{id: "ilha", name: "Ilha particular", emoji: "🏝️", rarity: legendary, value: 80_000, keep: true, luxury: true, aliases: []string{"ilha"},
		flex: &flex{effect: "quem tenta !roubar, !furtar, !sequestrar ou !matar você tem 12% menos chance", apply: func(p Profession) Profession { p.guard += 0.12; return p }},
		use:  "🏝️ @%s foi pra ilha particular e descobriu que lá não pega wi-fi"},

	{id: "peixeira", name: "Peixeira", emoji: "🔪", rarity: uncommon, value: 250, illegal: true, aliases: []string{"faca"},
		arms: &armament{crime: 0.05, kill: 0.10, guard: 0.03}},
	{id: "soco-ingles", name: "Soco inglês", emoji: "🥊", rarity: uncommon, value: 350, illegal: true,
		arms: &armament{crime: 0.07, guard: 0.05}},
	{id: "oitao", name: "Oitão", emoji: "🔫", rarity: rare, value: 1_200, illegal: true, aliases: []string{"revolver", "trezoitao", "berro"}, heist: 0.08,
		arms: &armament{crime: 0.10, kill: 0.20, guard: 0.08}},
	{id: "doze", name: "Escopeta doze", emoji: "💥", rarity: epic, value: 3_000, illegal: true, aliases: []string{"escopeta", "doze"}, heist: 0.12,
		arms: &armament{crime: 0.12, kill: 0.25, guard: 0.12}},
	{id: "fuzil", name: "Fuzil", emoji: "🪖", rarity: epic, value: 7_000, illegal: true, aliases: []string{"ak47", "metralhadora"}, heist: 0.18,
		arms: &armament{crime: 0.15, kill: 0.30, guard: 0.15}},
	{id: "bazuca", name: "Bazuca", emoji: "🚀", rarity: legendary, value: 20_000, illegal: true, aliases: []string{"rpg"}, heist: 0.25,
		arms: &armament{crime: 0.05, kill: 0.40, guard: 0.20}},

	{id: "baseado", name: "Baseado", emoji: "🌿", rarity: common, value: 50, illegal: true, aliases: []string{"maconha", "beck", "erva"},
		high: &high{lasts: time.Hour, cashback: 30},
		use:  "🌿 @%s acendeu um baseado, riu do nada por cinco minutos e pediu um podrão"},
	{id: "lanca-perfume", name: "Lança-perfume", emoji: "🧪", rarity: uncommon, value: 120, illegal: true, aliases: []string{"lanca", "lolo"},
		high: &high{lasts: 30 * time.Minute, overdose: 0.02, escape: 0.20},
		use:  "🧪 @%s deu uma cheirada no lança-perfume e saiu correndo igual bloco de carnaval"},
	{id: "bala", name: "Bala", emoji: "💊", rarity: uncommon, value: 200, illegal: true, aliases: []string{"ecstasy", "md"},
		high: &high{lasts: time.Hour, overdose: 0.03, heist: 0.15},
		use:  "💊 @%s tomou uma bala, abraçou todo mundo do grupo e disse que ama cada um de verdade"},
	{id: "cogumelo", name: "Chá de cogumelo", emoji: "🍄", rarity: rare, value: 300, illegal: true, aliases: []string{"cogumelo"},
		high: &high{lasts: time.Hour, revive: 0.25},
		use:  "🍄 @%s tomou chá de cogumelo e agora jura que tá conversando com os mortos"},
	{id: "po", name: "Pó", emoji: "❄️", rarity: rare, value: 400, illegal: true, aliases: []string{"cocaina", "farinha", "brizola"},
		high: &high{lasts: time.Hour, overdose: 0.06, pay: 25, crime: 0.10},
		use:  "❄️ @%s cheirou uma carreira, trincou o maxilar e agora fala 300 palavras por minuto"},

	{id: "arruda", name: "Galho de arruda", emoji: "🌿", rarity: common, value: 12, aliases: []string{"arruda"},
		bless: "protecao",
		use:   "🌿 @%s botou um galho de arruda atrás da orelha. Mau-olhado nenhum chega perto"},
	{id: "fita-do-bonfim", name: "Fitinha do Bonfim", emoji: "🎗️", rarity: common, value: 15, aliases: []string{"bonfim", "fita do senhor do bonfim"},
		bless: "sorte",
		use:   "🎗️ @%s amarrou a fitinha do Bonfim no pulso com três nós e fez três pedidos"},
	{id: "semente-de-roma", name: "Semente de romã", emoji: "🌰", rarity: common, value: 20, aliases: []string{"roma", "semente"},
		bless: "prosperidade",
		use:   "🌰 @%s guardou três sementes de romã na carteira, igual a vó ensinou no Dia de Reis"},
	{id: "pao-de-santo-antonio", name: "Pãozinho de Santo Antônio", emoji: "🥖", rarity: common, value: 10, aliases: []string{"santo antonio", "paozinho"},
		bless: "fartura",
		use:   "🥖 @%s guardou o pãozinho de Santo Antônio na lata de farinha. Comida nunca mais falta nessa casa"},
	{id: "defumador", name: "Defumador de alecrim", emoji: "💨", rarity: common, value: 20, aliases: []string{"alecrim", "incenso"},
		bless: "purificacao",
		use:   "💨 @%s passou o defumador de alecrim pela casa inteira e abriu as janelas pra energia ruim sair"},
	{id: "chave-de-sao-pedro", name: "Chave de São Pedro", emoji: "🗝️", rarity: rare, value: 450, aliases: []string{"sao pedro"},
		bless: "libertacao",
		use:   "🗝️ @%s rezou pra São Pedro com a chave na mão. Porta nenhuma segura essa pessoa agora"},
	{id: "ampulheta", name: "Ampulheta encantada", emoji: "⏳", rarity: rare, value: 500,
		bless: "pressa",
		use:   "⏳ @%s virou a ampulheta encantada de cabeça pra baixo e o tempo começou a correr a favor"},
	{id: "figa-de-guine", name: "Figa de guiné", emoji: "🤞", rarity: rare, value: 600, shop: true, aliases: []string{"figa", "guine"},
		bless: "faro",
		use:   "🤞 @%s fechou a mão numa figa de guiné e saiu farejando tesouro igual cachorro de garimpeiro"},
	{id: "vela-de-sao-longuinho", name: "Vela pra São Longuinho", emoji: "🕯️", rarity: rare, value: 450, shop: true, aliases: []string{"sao longuinho", "longuinho"},
		bless: "achado",
		use:   "🕯️ @%s acendeu uma vela pra São Longuinho e deu três pulinhos. Agora o que tava perdido vai aparecer"},
	{id: "banho-de-abre-caminho", name: "Banho de abre-caminho", emoji: "🛁", rarity: rare, value: 450, shop: true, aliases: []string{"abre caminho", "banho"},
		bless: "abrecaminho",
		use:   "🛁 @%s tomou um banho de abre-caminho do pescoço pra baixo. Agora só vem coisa boa"},
	{id: "machado-de-xango", name: "Machado de Xangô", emoji: "🪓", rarity: epic, value: 1_500, aliases: []string{"xango", "oxe"},
		bless: "justica",
		use:   "🪓 @%s ergueu o machado de Xangô e pediu justiça. Júri nenhum vai ser injusto com essa pessoa"},

	{id: "espelho-quebrado", name: "Espelho quebrado", emoji: "🪞", rarity: common, value: 15, aliases: []string{"espelho"},
		hex: "azar",
		use: "🪞 @%[1]s quebrou um espelho bem na frente de @%[2]s. Sete anos de azar, só que pra outra pessoa"},
	{id: "despertador-quebrado", name: "Despertador quebrado", emoji: "⏰", rarity: common, value: 15, aliases: []string{"despertador"},
		hex: "lerdeza",
		use: "⏰ @%[1]s trocou o despertador de @%[2]s por um quebrado. Vai perder a hora pra sempre"},
	{id: "marmita-estragada", name: "Marmita estragada", emoji: "🤢", rarity: common, value: 5, aliases: []string{"estragada"},
		hex: "jejum",
		use: "🤢 @%[1]s trocou a marmita de @%[2]s por uma estragada. Hoje não tem almoço, nem amanhã"},
	{id: "olho-de-sogra", name: "Olho de sogra", emoji: "🍬", rarity: uncommon, value: 50, aliases: []string{"sogra"},
		hex: "olhogordo",
		use: "🍬 @%[1]s deu um olho de sogra pra @%[2]s, e junto veio a inveja da família inteira"},
	{id: "moeda-amaldicoada", name: "Moeda amaldiçoada", emoji: "🪙", rarity: rare, value: 350, metal: true, aliases: []string{"amaldicoada"},
		hex: "miseria",
		use: "🪙 @%[1]s escondeu a moeda amaldiçoada no bolso de @%[2]s. O dinheiro vai escorrer pelos dedos"},
	{id: "pacto-com-o-capeta", name: "Pacto com o capeta", emoji: "📜", rarity: epic, value: 1_400, aliases: []string{"pacto", "capeta"},
		hex: "excomunhao",
		use: "📜 @%[1]s assinou um pacto com o capeta no nome de @%[2]s. Júri nenhum fica do lado dessa pessoa agora"},

	{id: "pente-da-iara", name: "Pente de ouro da Iara", emoji: "🪮", rarity: mythic, value: 15_000, aliases: []string{"iara"}},
	{id: "olho-de-boto", name: "Olho de boto", emoji: "👁️", rarity: mythic, value: 12_000, aliases: []string{"olho do boto"}},
	{id: "pepita-do-eldorado", name: "Pepita do Eldorado", emoji: "🌞", rarity: mythic, value: 18_000, aliases: []string{"eldorado"}},
	{id: "fogo-da-mae-do-ouro", name: "Fogo da Mãe do Ouro", emoji: "🔥", rarity: mythic, value: 14_000, aliases: []string{"mae do ouro"}},
	{id: "chupa-cabra", name: "Chupa-cabra empalhado", emoji: "🐐", rarity: mythic, value: 15_000, aliases: []string{"chupacabra"}},
	{id: "boneco-do-fofao", name: "Boneco do Fofão", emoji: "🔪", rarity: mythic, value: 12_000, aliases: []string{"fofao"}},
	{id: "pena-do-uirapuru", name: "Pena do Uirapuru", emoji: "🪶", rarity: mythic, value: 15_000, aliases: []string{"uirapuru"}},
	{id: "pito-da-caipora", name: "Pito da Caipora", emoji: "🚬", rarity: mythic, value: 12_000, aliases: []string{"caipora"}},

	{id: "grampo", name: "Grampo de cabelo", emoji: "🪝", rarity: common, value: 120, shop: true, aliases: []string{"grampo", "gancho"},
		flex: &flex{effect: "+10% de chance de fugir do cativeiro no !fugir", brief: "+10% pra fugir do cativeiro", apply: func(p Profession) Profession { p.breakoutBonus += 0.10; return p }}},
	{id: "cartao-fidelidade", name: "Cartão fidelidade", emoji: "💳", rarity: uncommon, value: 800, shop: true, aliases: []string{"fidelidade"},
		flex: &flex{effect: "5% de desconto na !loja", apply: func(p Profession) Profession { p.discount += 5; return p }}},
	{id: "mapa-geologico", name: "Mapa geológico", emoji: "🗺️", rarity: rare, value: 1_500, shop: true, aliases: []string{"geologico"},
		gear: &gear{activities: []string{"!minerar"}, luck: 1}},

	{id: "lupa", name: "Lupa de joalheiro", emoji: "🔍", rarity: epic, value: 8_000, shop: true, rotating: true, aliases: []string{"lupa", "joalheiro"},
		gear: &gear{activities: []string{"!pescar", "!minerar", "!vasculhar", "!catar"}, grade: 1}},
	{id: "estilete", name: "Estilete de unboxing", emoji: "🔪", rarity: epic, value: 6_000, shop: true, rotating: true, aliases: []string{"estilete", "unboxing"},
		flex: &flex{effect: "abre caixa e baú no !usar com um sorteio a mais, e fica com o item mais raro", brief: "caixa e baú abrem com mais sorte", apply: func(p Profession) Profession { p.boxLuck++; return p }}},
	{id: "cofre-de-parede", name: "Cofre de parede", emoji: "🐷", rarity: rare, value: 3_500, shop: true, rotating: true, aliases: []string{"cofre de parede"},
		flex: &flex{effect: "guarda 5.000 a mais por vez na poupança", apply: func(p Profession) Profession { p.savingsRoom += 5_000; return p }}},
	{id: "desfibrilador", name: "Desfibrilador", emoji: "🩺", rarity: epic, value: 5_000, shop: true, rotating: true, aliases: []string{"choque"},
		flex: &flex{effect: "+20% de chance no !reviver, e libera o !reviver pra qualquer profissão", brief: "+20% no !reviver, pra qualquer profissão", apply: func(p Profession) Profession { p.reviveChance += 0.20; return p }}},
	{id: "vade-mecum", name: "Vade Mecum", emoji: "📕", rarity: rare, value: 2_500, shop: true, rotating: true, aliases: []string{"vade", "codigo penal"},
		flex: &flex{effect: "leva 20% a mais quando ganha um !processar e paga 20% menos quando perde", brief: "ganha mais e perde menos no !processar", apply: func(p Profession) Profession {
			p.trialBonus += 20
			p.trialDiscount = min(p.trialDiscount+20, 90)
			return p
		}}},
	{id: "biblia-de-cela", name: "Bíblia de cela", emoji: "📖", rarity: rare, value: 2_000, shop: true, rotating: true, aliases: []string{"biblia"},
		flex: &flex{effect: "toda ida pra cadeia fica 20% mais curta, por bom comportamento", brief: "cadeia 20% mais curta", apply: func(p Profession) Profession { p.jailFactor -= 20; return p }}},
	{id: "acordo-pre-nupcial", name: "Acordo pré-nupcial", emoji: "📝", rarity: uncommon, value: 900, shop: true, rotating: true, aliases: []string{"pre nupcial", "acordo"},
		flex: &flex{effect: "paga 30% menos de pensão e de indenização por traição", brief: "30% menos de pensão e indenização", apply: func(p Profession) Profession { p.legalDiscount = min(p.legalDiscount+30, 90); return p }}},
	{id: "camera-escondida", name: "Câmera escondida", emoji: "📹", rarity: rare, value: 1_500, shop: true, rotating: true, aliases: []string{"camera"},
		flex: &flex{effect: "+10% de chance da polícia acreditar no !denunciar", brief: "+10% no !denunciar", apply: func(p Profession) Profession { p.reportBonus += 0.10; return p }}},
	{id: "luvas-de-couro", name: "Luvas de couro", emoji: "🧤", rarity: rare, value: 1_800, shop: true, rotating: true, aliases: []string{"luvas de couro"},
		flex: &flex{effect: "o !furtar recarrega 25% mais rápido", apply: func(p Profession) Profession {
			p.theftEvery = cmp.Or(p.theftEvery, theftCooldown.every) * 3 / 4
			return p
		}}},
	{id: "relogio-de-ponto", name: "Relógio de ponto", emoji: "⏱️", rarity: rare, value: 2_500, shop: true, rotating: true, aliases: []string{"ponto"},
		flex: &flex{effect: "o !trabalhar paga 25% a mais das 5h às 9h, se a sua profissão não tiver horário de pico próprio", brief: "+25% no !trabalhar das 5h às 9h", apply: func(p Profession) Profession {
			if p.peak.bonus == 0 {
				p.peak = peak{from: 5, to: 9, bonus: 25}
			}
			return p
		}}},
	{id: "terminal-de-corretora", name: "Terminal de corretora", emoji: "📈", rarity: epic, value: 5_000, shop: true, rotating: true, aliases: []string{"terminal", "corretora"},
		flex: &flex{effect: fmt.Sprintf("não paga a corretagem de %d%% no !acoes", stockFeePercent), brief: "sem corretagem no !acoes", apply: func(p Profession) Profession { p.freeBrokerage = true; return p }}},
	{id: "kit-disfarce", name: "Kit disfarce", emoji: "🥸", rarity: rare, value: 600, shop: true, rotating: true, aliases: []string{"disfarce", "bigode falso", "peruca"},
		resets: []string{"roubo_em", "furto_em", "sequestro_em"}, limit: &disguise,
		use: "🥸 @%s colou um bigode falso, botou uma peruca e um óculos escuro. Ninguém reconhece mais essa pessoa"},

	{id: "cartao-black", name: "Cartão black", emoji: "💳", rarity: epic, value: 6_000, keep: true, luxury: true, aliases: []string{"black"},
		flex: &flex{effect: "a poupança libera 25% mais rápido", apply: func(p Profession) Profession { p.savingsFaster = min(p.savingsFaster+25, 90); return p }},
		use:  "💳 @%s pagou o cafezinho com o cartão black e fez questão de deixar ele em cima da mesa"},

	{id: "balanca-adulterada", name: "Balança adulterada", emoji: "⚖️", rarity: epic, value: 6_000, illegal: true, aliases: []string{"balanca"},
		flex: &flex{effect: "o mercado paga 10 pontos a mais do valor no !vender (até 95%)", apply: func(p Profession) Profession { p.salePercent = min(p.marketPercent()+10, 95); return p }}},
	{id: "dado-viciado", name: "Dado viciado", emoji: "🎲", rarity: epic, value: 7_000, illegal: true, aliases: []string{"dado"},
		flex: &flex{effect: "10% de chance de pegar a aposta inteira de volta quando perde no " + casinoGames, apply: func(p Profession) Profession { p.rescueOdds += 0.10; return p }}},
	{id: vestItemID, name: "Colete à prova de balas", emoji: "🦺", rarity: rare, value: 2_500, illegal: true, carry: vestCarry, aliases: []string{"colete", "colete balistico"}},
	{id: forgedReleaseID, name: "Alvará de soltura falso", emoji: "📜", rarity: epic, value: 3_000, illegal: true, aliases: []string{"alvara", "alvara falso", "habeas corpus"},
		use: "📜 @%s mostrou um alvará de soltura falso"},
	{id: "rebite", name: "Rebite", emoji: "🚚", rarity: uncommon, value: 250, illegal: true, aliases: []string{"anfetamina", "arrebite"},
		high: &high{lasts: 2 * time.Hour, overdose: 0.03, pay: 15},
		use:  "🚚 @%s tomou um rebite igual caminhoneiro na Régis e agora não dorme até sexta"},
	{id: "ayahuasca", name: "Ayahuasca", emoji: "🍵", rarity: epic, value: 2_000, illegal: true, aliases: []string{"daime", "cha do daime"},
		high: &high{lasts: 2 * time.Hour, overdose: 0.02, revive: 0.50, luck: 2},
		use:  "🍵 @%s tomou ayahuasca na roda do pajé, vomitou a alma e voltou enxergando onde tá escondido cada tesouro do mundo"},
	{id: "anabolizante", name: "Anabolizante", emoji: "💉", rarity: epic, value: 2_200, illegal: true, aliases: []string{"bomba", "durateston", "bombado"},
		high: &high{lasts: 2 * time.Hour, overdose: 0.06, crime: 0.15, escape: 0.20, heist: 0.25, kidnap: 0.20, guard: 0.30},
		use:  "💉 @%s aplicou uma bomba na bunda, ganhou dez quilos de músculo e perdeu dez de paciência"},
	{id: "pedra-azul", name: "Pedra azul", emoji: "🔹", rarity: epic, value: 2_500, illegal: true, aliases: []string{"meta", "metanfetamina", "azulzinha"},
		high: &high{lasts: time.Hour, overdose: 0.08, pay: 50, rush: 30},
		use:  "🔹 @%s fumou a pedra azul que veio de um trailer no deserto e agora trabalha igual máquina e fala igual professor de química"},
	{id: "po-de-anjo", name: "Pó de anjo", emoji: "😇", rarity: legendary, value: 7_000, illegal: true, aliases: []string{"anjo", "pcp"},
		high: &high{lasts: time.Hour, overdose: 0.12, crime: 0.30, escape: 0.30, kidnap: 0.30, kill: 0.40, guard: 0.25},
		use:  "😇 @%s cheirou pó de anjo, arrancou a camisa e agora não sente dor, medo nem remorso"},
	{id: "nzt", name: "Pílula NZT", emoji: "🧠", rarity: legendary, value: 8_000, illegal: true, aliases: []string{"nzt", "nzt-48", "limitless", "pilula"},
		high: &high{lasts: time.Hour, overdose: 0.10, pay: 100, cashback: 50, rush: 25, luck: 1},
		use:  "🧠 @%s tomou a pílula NZT e agora usa 100%% do cérebro. Já entendeu o mercado, o cassino e o sentido da vida"},

	{id: "boia", name: "Boia salva-vidas", emoji: "🛟", rarity: uncommon, value: 500, shop: true, aliases: []string{"boia salva vidas", "salva vidas"},
		gear: &gear{activities: []string{"!pescar"}, safety: 60}},
	{id: "roupa-de-apicultor", name: "Roupa de apicultor", emoji: "🐝", rarity: uncommon, value: 500, shop: true, aliases: []string{"roupa de abelha", "macacao de apicultor"},
		gear: &gear{activities: []string{"!catar"}, safety: 60}},
	{id: safetyKitID, name: "Kit EPI", emoji: "👷", rarity: rare, value: 1_500, shop: true, aliases: []string{"epi", "equipamento de protecao"},
		flex: &flex{effect: "se machuca 50% menos no !trabalhar", apply: func(p Profession) Profession { p.safety += 50; return p }}},
	{id: "curativo", name: "Curativo", emoji: "🩹", rarity: common, value: 60, shop: true, aliases: []string{"band-aid", "bandaid", "esparadrapo"}, heal: 20,
		use: "🩹 @%s colou um curativo no machucado e assoprou pra sarar"},
	{id: "kit-de-primeiros-socorros", name: "Kit de primeiros socorros", emoji: "🚑", rarity: uncommon, value: 250, shop: true, aliases: []string{"primeiros socorros", "kit medico"}, heal: 60,
		use: "🚑 @%s abriu o kit de primeiros socorros, passou merthiolate (do que arde) e enfaixou tudo"},
	{id: "garrafada", name: "Garrafada da benzedeira", emoji: "🍶", rarity: rare, value: 600, shop: true, aliases: []string{"benzedeira", "garrafada de raiz"}, heal: 35, doses: 4,
		use: "🍶 @%s tomou um gole da garrafada da benzedeira, fez careta com o amargo e sentiu o corpo fechando"},
	{id: "maleta-de-medico", name: "Maleta de médico", emoji: "⚕️", rarity: epic, value: 1_500, shop: true, rotating: true, aliases: []string{"maleta", "maleta medica"}, heal: 60, doses: 5,
		use: "⚕️ @%s abriu a maleta de médico, deu ponto no corte, tomou um antibiótico e ainda se receitou repouso"},
	{id: "ampola-de-adrenalina", name: "Ampola de adrenalina", emoji: "💉", rarity: epic, value: 1_200, aliases: []string{"adrenalina", "ampola"}, heal: 100, doses: 2,
		use: "💉 @%s cravou a ampola de adrenalina no peito igual Pulp Fiction e levantou novinho(a) em folha"},
	{id: "agua-benta-de-aparecida", name: "Água benta de Aparecida", emoji: "🕊️", rarity: legendary, value: 6_000, aliases: []string{"aparecida"}, heal: 100, doses: 6,
		use: "🕊️ @%s passou água benta de Aparecida nos machucados e rezou uma ave-maria. Sarou tudo, é milagre"},
	{id: "caixa-misteriosa-prateada", name: "Caixa misteriosa prateada", emoji: "🥈", rarity: rare, value: 850, shop: true, aliases: []string{"caixa prateada", "caixa de prata", "prateada"},
		loot: []int{28, 27, 26, 15, 4}, use: "🥈 @%s sacudiu a caixa misteriosa prateada do lado do ouvido antes de abrir, pra ver se tinha coisa boa"},
	{id: "foice-da-comadre-morte", name: "Foice da Comadre Morte", emoji: "💀", rarity: relic, value: 30_000, aliases: []string{"foice", "comadre morte", "ceifador"},
		flex: &flex{effect: fmt.Sprintf("+%d%% de chance no !matar, que fica liberado pra qualquer profissão e passa direto pelo colete. Quem você mata fica %dx mais tempo morto(a), e o enterro leva de %d%% a %d%% do saldo da vítima, que vão pro seu bolso em vez de sumir", percent(reaperKillBonus), reaperHoldFactor, reapLossMinPercent, reapLossMaxPercent),
			apply: func(p Profession) Profession {
				p.killChance += reaperKillBonus
				p.reaper = true
				return p
			}}},
	{id: ciprianoItem, name: "Livro de São Cipriano", emoji: "📕", rarity: relic, value: 30_000, aliases: []string{"sao cipriano", "cipriano", "capa preta", "livro de capa preta", "grimorio"},
		flex: &flex{effect: fmt.Sprintf("o !reviver nunca falha e fica liberado pra qualquer profissão, e quem você revive volta com um pacto: por %s, %d%% de tudo que a pessoa ganhar vai pra você. E quando você morre, volta na metade do tempo", core.FormatWait(soulPactHold), soulPactPercent),
			apply: func(p Profession) Profession {
				p.reviveChance = 1
				p.soulPact = true
				return p
			}}},
	{id: "chapeu-do-boto", name: "Chapéu do Boto", emoji: "🐬", rarity: relic, value: 30_000, aliases: []string{"chapeu branco"},
		flex: &flex{effect: fmt.Sprintf("o seu alcance no !postar ganha mais %s (quem não é de Arte e mídia vai de %s pra %s), nenhum post é cancelado e a galera nunca acha a !publi forçada", multiplier(botoReach), multiplier(baseReach), multiplier(baseReach+botoReach)),
			apply: func(p Profession) Profession {
				p.reach = p.reachPercent() + botoReach
				p.charming = true
				return p
			}}},
	{id: corpoSecoItem, name: "Coração do Corpo-Seco", emoji: "🫀", rarity: relic, value: 30_000, aliases: []string{"corpo seco", "coracao seco"},
		flex: &flex{effect: "o seu corpo aceita qualquer coisa: o implante nunca é rejeitado na instalação, e ninguém arranca os que tão instalados, nem quem te mata no !matar, nem a polícia no !prender e no !denunciar. A manutenção continua igual",
			apply: func(p Profession) Profession { p.sealedBody = true; return p }}},
	{id: "quepe-da-volante", name: "Quepe da Volante", emoji: "🪖", rarity: relic, value: 30_000, aliases: []string{"volante", "quepe"},
		flex: &flex{effect: fmt.Sprintf("o quepe da tropa que caçou o Lampião: deixa você caçar o procurado do dia, que pode levar !matar seu mesmo que você não seja do crime, com pelo menos %d%% de chance (menos a proteção dele), sem contar como crime e sem risco de ir preso, e a recompensa que o bot botou na cabeça dele vem em dobro pra você. Contra quem tem o Chapéu de Lampião, +%d%% de chance no !matar", percent(posseKillChance), percent(posseLampiaoBonus)),
			apply: func(p Profession) Profession { p.posse = true; return p }}},
	{id: carrancaItem, name: "Carranca do Velho Chico", emoji: "🗿", rarity: relic, value: 30_000, aliases: []string{"carranca", "velho chico"},
		flex: &flex{effect: fmt.Sprintf("a cabeça de bicho da proa dos barcos do São Francisco, que espanta mau espírito: enquanto ela tiver na mochila, a Casa assombrada não faz nada com o seu imóvel (nem o IPTU dobra), e a proteção e os bônus do imóvel no !daily e no !trabalhar valem %d%% a mais", carrancaBoost),
			apply: warded}},
	{id: buziosItem, name: "Búzios encantados", emoji: "🐚", rarity: relic, value: 30_000, aliases: []string{"buzios", "jogo de buzios"},
		flex: &flex{effect: fmt.Sprintf("o jogo de búzios lê o seu destino: os eventos do !trabalhar, das extrações e dos crimes aparecem %dx mais pra você (%d%% das vezes em vez de %d%%), e a enquete do evento mostra a chance de cada opção dar bom e dar ruim", oracleEventFactor, percent(eventChance*oracleEventFactor), percent(eventChance)),
			apply: func(p Profession) Profession { p.oracle = true; return p }}},
	{id: "vara-de-carbono-aprimorada", name: "Vara de carbono aprimorada", emoji: "🎣", rarity: rare, value: 2_250, upgrades: "vara-de-carbono",
		gear: &gear{activities: []string{"!pescar"}, tool: true, sturdy: 60, luck: 1, fewerJunk: 74, moreRare: 2, grade: 1}},
	{id: "picareta-de-diamante-aprimorada", name: "Picareta de diamante aprimorada", emoji: "💎", rarity: rare, value: 2_700, upgrades: "picareta-de-diamante",
		gear: &gear{activities: []string{"!minerar"}, tool: true, sturdy: 60, grace: 10, fewerJunk: 89, moreRare: 6, grade: 1}},
	{id: "rede-de-seda-aprimorada", name: "Rede de seda aprimorada", emoji: "🥅", rarity: rare, value: 1_800, upgrades: "rede-de-seda",
		gear: &gear{activities: []string{"!catar"}, tool: true, sturdy: 60, luck: 1, fewerJunk: 79, moreRare: 4, grade: 1}},
	{id: "vara-de-titanio-aprimorada", name: "Vara de titânio aprimorada", emoji: "🎣", rarity: epic, value: 9_000, upgrades: "vara-de-titanio",
		gear: &gear{activities: []string{"!pescar"}, tool: true, sturdy: 85, grace: 20, luck: 3, faster: 40, fewerJunk: 81, moreRare: 24, grade: 2}},
	{id: "picareta-de-niobio-aprimorada", name: "Picareta de nióbio aprimorada", emoji: "⛏️", rarity: epic, value: 10_500, upgrades: "picareta-de-niobio",
		gear: &gear{activities: []string{"!minerar"}, tool: true, sturdy: 85, grace: 20, luck: 2, faster: 40, fewerJunk: 86, moreRare: 2, grade: 2}},
	{id: "rede-de-nailon-aprimorada", name: "Rede de náilon aprimorada", emoji: "🥅", rarity: epic, value: 7_500, upgrades: "rede-de-nailon",
		gear: &gear{activities: []string{"!catar"}, tool: true, sturdy: 85, grace: 20, luck: 3, faster: 40, fewerJunk: 85, moreRare: 15, grade: 2}},
	{id: "vara-de-grafeno-aprimorada", name: "Vara de grafeno aprimorada", emoji: "🎣", rarity: legendary, value: 21_000, upgrades: "vara-de-grafeno",
		gear: &gear{activities: []string{"!pescar"}, tool: true, sturdy: 97, grace: 100, luck: 6, faster: 50, fewerJunk: 94, moreRare: 10, grade: 3}},
	{id: "picareta-de-tungstenio-aprimorada", name: "Picareta de tungstênio aprimorada", emoji: "⛏️", rarity: legendary, value: 22_500, upgrades: "picareta-de-tungstenio",
		gear: &gear{activities: []string{"!minerar"}, tool: true, sturdy: 97, grace: 100, luck: 5, faster: 50, fewerJunk: 85, grade: 3}},
	{id: "rede-entomologica-aprimorada", name: "Rede entomológica aprimorada", emoji: "🥅", rarity: legendary, value: 18_000, upgrades: "rede-entomologica",
		gear: &gear{activities: []string{"!catar"}, tool: true, sturdy: 97, grace: 100, luck: 6, faster: 50, fewerJunk: 88, moreRare: 18, grade: 3}},
	{id: "celular", name: "Celular", emoji: "📱", rarity: uncommon, value: 900, shop: true, aliases: []string{"smartphone", "telefone"},
		networks: []string{"tiktok", "instagram"}},
	{id: "computador", name: "Computador", emoji: "💻", rarity: rare, value: 2_500, shop: true, aliases: []string{"pc", "desktop"},
		networks: []string{"youtube", "twitch"}},
	{id: skyRockID, name: "Rocha celeste", emoji: "🌠", rarity: primordial, value: 20_000, aliases: []string{"rocha", "pedra do ceu", "pedra celeste"},
		loot: []int{0, 0, 0, 0, 70, 30}, shard: bendegoShardID, shardOdds: 0.5, use: "🌠 @%s rachou a rocha celeste com uma marreta, e ela ainda tava quente por dentro"},
	{id: bendegoShardID, name: "Fragmento do Bendegó", emoji: "🪨", rarity: primordial, value: 30_000, aliases: []string{"fragmento", "fragmento do meteorito", "pedaco do bendego"},
		forms: bendegoID, shards: 4},
	{id: bendegoID, name: "Meteorito do Bendegó", emoji: "☄️", rarity: primordial, value: 150_000, aliases: []string{"bendego"},
		gear: &gear{activities: []string{"!pescar", "!minerar", "!vasculhar", "!catar"}, pull: 1, alwaysFinds: true}},
	{id: "scanner-da-policia", name: "Scanner da polícia", emoji: "📡", rarity: epic, value: 6_000, hot: true, aliases: []string{"scanner", "escuta da policia", "radio da policia"},
		flex: &flex{effect: fmt.Sprintf("fica na frequência da polícia: %d%% de chance de ouvir a viatura chegando e escapar quando alguém tenta te !prender", percent(scannerTipoff)), apply: func(p Profession) Profession { p.tipoff += scannerTipoff; return p }}},
}, implantGoods(), gardenGoods, gatherGoods)

func itemByID(id string) (item, bool) {
	i := slices.IndexFunc(catalog, func(it item) bool { return it.id == id })
	if i < 0 {
		return item{}, false
	}
	return catalog[i], true
}

func itemByNumber(n int) (item, bool) {
	if n < 1 || n > len(catalog) {
		return item{}, false
	}
	return catalog[n-1], true
}

func (it item) number() int {
	return slices.IndexFunc(catalog, func(other item) bool { return other.id == it.id }) + 1
}

func (it item) store() string {
	if it.pet {
		return "!petshop"
	}
	return "!loja"
}

func (it item) shelf() string {
	if it.days != nil {
		return "!loja (só " + weekdaysText(it.days) + ")"
	}
	if featuredTool(it) {
		return "!loja (nas ofertas do dia: uma ferramenta épica de extração aparece todo dia, em rodízio, e cada uma volta em até 3 dias)"
	}
	if it.rotating {
		return "!loja (só nos dias em que cai nas ofertas do dia)"
	}
	return it.store()
}

func (it item) label() string {
	return it.emoji + " *" + it.name + "*"
}

func (it item) summary() string {
	r := rarities[it.rarity]
	return fmt.Sprintf("%s %s · vale %s", r.emoji, r.name, Coins(it.value))
}

func quantityLabel(it item, quantity int64) string {
	if quantity == 1 {
		return it.label()
	}
	return fmt.Sprintf("%dx %s", quantity, it.label())
}

func compareItems(a, b item) int {
	return cmp.Or(cmp.Compare(b.rarity, a.rarity), cmp.Compare(b.value, a.value), strings.Compare(a.name, b.name))
}

func rarityGroups[T any](entries []T, itemOf func(T) item, line func(T) string) []string {
	sorted := slices.SortedFunc(slices.Values(entries), func(a, b T) int { return compareItems(itemOf(a), itemOf(b)) })
	var lines []string
	current := -1
	for _, e := range sorted {
		if r := itemOf(e).rarity; r != current {
			current = r
			lines = append(lines, "", rarities[r].heading())
		}
		lines = append(lines, line(e))
	}
	return lines
}

type itemCategory struct {
	emoji   string
	name    string
	aliases []string
	holds   func(item) bool
}

func (c itemCategory) heading() string {
	return c.emoji + " *" + c.name + "*"
}

func droppedBy(command string) func(item) bool {
	return func(it item) bool {
		return slices.ContainsFunc(activityNamed(command).loot, func(d drop) bool { return d.item == it.id })
	}
}

func isTool(it item) bool {
	_, ok := activityUsing(it.id)
	return ok || it.gear != nil || it.id == lockItemID || it.id == alarmItemID || it.id == vestItemID || it.id == safetyKitID || it.limit == &disguise
}

var itemCategories = []itemCategory{
	{"⚫", "Primordiais", []string{"primordial"}, func(it item) bool { return it.rarity == primordial }},
	{"🏛️", "Relíquias", []string{"reliquia"}, func(it item) bool { return it.rarity == relic || it.relicOdds > 0 }},
	{"🧰", "Ferramentas e proteção", []string{"ferramentas", "ferramenta", "protecao", "equipamentos"}, isTool},
	{"🩹", "Saúde", []string{"cura", "curas", "curativos", "remedio", "remedios"}, func(it item) bool { return it.heal > 0 }},
	{"🛠️", "Instrumentos de trabalho", []string{"instrumentos", "instrumento", "oficio", "oficios", "trabalho"}, func(it item) bool { return it.trade != nil }},
	{"📲", "Aparelhos", []string{"aparelho", "eletronicos", "eletronico"}, func(it item) bool { return it.networks != nil }},
	{"🦾", "Implantes", []string{"implante", "ciborgue", "cirurgia"}, func(it item) bool { return it.graft != nil }},
	{"🔫", "Armas", []string{"arma"}, func(it item) bool { return it.arms != nil }},
	{"💊", "Drogas", []string{"droga"}, func(it item) bool { return it.high != nil }},
	{"🕶️", "Contrabando", []string{"contrabandos", "ilegal", "ilegais", "muamba"}, func(it item) bool { return it.illegal }},
	{"🦹", "Equipamento de assalto", []string{"assalto", "assaltos"}, func(it item) bool { return it.heist > 0 }},
	{"🐾", "Bichinhos", []string{"pets", "pet", "bichinho"}, func(it item) bool { return it.pet }},
	{"🥂", "Luxo", []string{"luxos"}, func(it item) bool { return it.luxury }},
	{"🎒", "Vantagens", []string{"vantagem", "passivas", "passiva"}, func(it item) bool { return it.flex != nil }},
	{"🔮", "Bênçãos e maldições", []string{"encantos", "encanto", "bencao", "bencaos", "maldicao", "maldicoes", "misticos", "feiticaria"}, func(it item) bool { return it.bless != "" || it.hex != "" }},
	{"🎁", "Pra abrir", []string{"abrir", "caixas", "premios", "sorte"}, func(it item) bool { return it.loot != nil || it.prizes != nil }},
	{"💝", "Presentes", []string{"presente"}, func(it item) bool { return it.shop && it.gift != "" }},
	{"🍔", "Comes e bebes", []string{"comida", "comidas", "bebida", "bebidas", "lanche", "lanches"}, func(it item) bool { return it.shop && it.use != "" }},
	{"🌱", "Horta", []string{"sementes", "mudas", "plantas", "planta", "colheita", "plantacao", "roca"}, fromGarden},
	{"🎣", "Pesca", []string{"pescar", "peixe", "peixes"}, droppedBy("!pescar")},
	{"⛏️", "Mineração", []string{"minerar", "minerio", "minerios"}, droppedBy("!minerar")},
	{"🗑️", "Lixão", []string{"vasculhar", "lixo"}, droppedBy("!vasculhar")},
	{"🦋", "Insetos", []string{"catar", "inseto"}, droppedBy("!catar")},
	{"💼", "Achados do trampo", []string{"achados", "achado", "trampo"}, foundAtWork},
	{"✨", "Raridades", []string{"raridade", "colecionaveis", "colecao"}, func(item) bool { return true }},
}

func (it item) category() int {
	return slices.IndexFunc(itemCategories, func(c itemCategory) bool { return c.holds(it) })
}

func categoryNamed(query string) (int, bool) {
	q := normalizeItemName(query)
	i := slices.IndexFunc(itemCategories, func(c itemCategory) bool {
		return normalizeItemName(c.name) == q || slices.ContainsFunc(c.aliases, func(alias string) bool { return normalizeItemName(alias) == q })
	})
	return i, i >= 0
}

func categoryGroups[T any](entries []T, itemOf func(T) item, line func(T) string) []string {
	groups := make([][]T, len(itemCategories))
	for _, e := range entries {
		c := itemOf(e).category()
		groups[c] = append(groups[c], e)
	}
	var lines []string
	for c, group := range groups {
		if len(group) == 0 {
			continue
		}
		slices.SortStableFunc(group, func(a, b T) int { return compareItems(itemOf(a), itemOf(b)) })
		lines = append(lines, "", itemCategories[c].heading())
		for _, e := range group {
			lines = append(lines, line(e))
		}
	}
	return lines
}

func rarityLegend() string {
	labels := make([]string, len(rarities))
	for i, r := range rarities {
		labels[i] = r.emoji + " " + r.name
	}
	return strings.Join(labels, " · ")
}

func normalizeItemName(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, core.FoldLetters(s))
}

func (it item) matchKeys() []string {
	keys := []string{normalizeItemName(it.name), normalizeItemName(it.id)}
	for _, alias := range it.aliases {
		keys = append(keys, normalizeItemName(alias))
	}
	return keys
}

func findItems(query string) []item {
	q := normalizeItemName(query)
	if q == "" {
		return nil
	}
	matchers := []func(key string) bool{
		func(key string) bool { return key == q },
		func(key string) bool { return strings.HasPrefix(key, q) },
		func(key string) bool { return strings.Contains(key, q) },
	}
	for _, match := range matchers {
		found := slices.DeleteFunc(slices.Clone(catalog), func(it item) bool { return !slices.ContainsFunc(it.matchKeys(), match) })
		if len(found) > 0 {
			return found
		}
	}
	return nil
}

func preferItems(matches []item, keep func(item) bool) []item {
	if len(matches) < 2 {
		return matches
	}
	preferred := slices.DeleteFunc(slices.Clone(matches), func(it item) bool { return !keep(it) })
	if len(preferred) == 0 {
		return matches
	}
	return preferred
}

func ownedIn(held []stack) func(item) bool {
	return func(it item) bool { return quantityOf(held, it.id) > 0 }
}

func quantityOf(held []stack, id string) int64 {
	i := slices.IndexFunc(held, func(s stack) bool { return s.item.id == id })
	if i < 0 {
		return 0
	}
	return held[i].quantity
}

func itemQueryProblem(query string, matches []item) string {
	if len(matches) == 1 {
		return ""
	}
	if len(matches) == 0 {
		return fmt.Sprintf("Não conheço nenhum item chamado *%s* 🤔 Manda !item pra ver todos", query)
	}
	labels := make([]string, 0, len(matches))
	for _, it := range matches[:min(len(matches), 6)] {
		labels = append(labels, it.label())
	}
	return "Achei mais de um item com esse nome: " + core.JoinWithAnd(labels) + ". Escreve o nome mais completo"
}

type itemQuery struct {
	name    string
	numbers []int64
	count   int64
	all     bool
}

func parseItemQuery(text string) itemQuery {
	everything := map[string]bool{"tudo": true, "todos": true, "todas": true, "all": true}
	fillers := map[string]bool{"por": true}
	var (
		q     itemQuery
		words []string
	)
	fields := strings.Fields(strings.ToLower(text))
	for _, field := range fields[min(1, len(fields)):] {
		if strings.HasPrefix(field, "@") || fillers[field] {
			continue
		}
		if everything[field] {
			q.all = true
			continue
		}
		if n, ok := parseAmount(field); ok {
			q.numbers = append(q.numbers, n)
			continue
		}
		if n, ok := parseCount(field); ok {
			q.count = n
			continue
		}
		words = append(words, field)
	}
	q.name = strings.Join(words, " ")
	return q
}

func parseAmount(s string) (int64, bool) {
	n, err := strconv.ParseInt(strings.ReplaceAll(s, ".", ""), 10, 64)
	return n, err == nil && n > 0
}

func parseCount(s string) (int64, bool) {
	digits, ok := strings.CutSuffix(s, "x")
	if !ok {
		return 0, false
	}
	return parseAmount(digits)
}

func (q itemQuery) quantity(held int64) int64 {
	if q.all {
		return held
	}
	if q.count > 0 {
		return q.count
	}
	if len(q.numbers) > 0 {
		return q.numbers[0]
	}
	return 1
}

func weightedIndex(n int, weight func(i int) int) int {
	total := 0
	for i := range n {
		total += weight(i)
	}
	roll := rand.IntN(total)
	for i := range n {
		if roll < weight(i) {
			return i
		}
		roll -= weight(i)
	}
	return n - 1
}

func (it item) unboxable() bool {
	return it.luxury || it.illegal || it.hot || it.storeOnly || it.rarity >= mythic || it.relicOdds > 0 || it.upgrades != "" || it.graft != nil
}

func upgradeOf(base item) (item, bool) {
	i := slices.IndexFunc(catalog, func(it item) bool { return it.upgrades == base.id })
	if i < 0 {
		return item{}, false
	}
	return catalog[i], true
}

func (it item) upgradeCost() int64 {
	base, _ := itemByID(it.upgrades)
	return it.value - base.value
}

func randomItemOf(r int) item {
	pool := slices.DeleteFunc(slices.Clone(catalog), func(it item) bool { return it.rarity != r })
	if r == mythic {
		return core.PickOne(pool)
	}
	return core.PickOne(slices.DeleteFunc(pool, item.unboxable))
}

func (it item) lootWeight(r int) int {
	if r >= len(it.loot) {
		return 0
	}
	return it.loot[r]
}

func relics() []item {
	return slices.DeleteFunc(slices.Clone(catalog), func(it item) bool { return it.rarity != relic })
}

func relicNames() string {
	var names []string
	for _, it := range relics() {
		names = append(names, it.label())
	}
	return core.JoinWithAnd(names)
}

func relicBoxes() []item {
	return slices.DeleteFunc(slices.Clone(catalog), func(it item) bool { return it.relicOdds == 0 })
}

func (it item) roll(luck int) itemEffect {
	var e itemEffect
	if it.loot != nil {
		e.prize = randomItemOf(it.lootRarity(luck))
	}
	if jinxedHit(it.relicOdds, luck) {
		e.prize = core.PickOne(relics())
	}
	if rand.Float64() < it.shardOdds {
		e.prize, _ = itemByID(it.shard)
	}
	if it.prizes != nil {
		e.coins = drawn(min(luck, 0), func() int64 {
			return it.prizes[weightedIndex(len(it.prizes), func(i int) int { return it.prizes[i].weight })].amount
		})
	}
	return e
}

func (it item) lootRarity(luck int) int {
	return drawn(luck, func() int { return weightedIndex(len(it.loot), func(i int) int { return it.loot[i] }) })
}

func drawn[T cmp.Ordered](luck int, draw func() T) T {
	picked := draw()
	for range luck {
		picked = max(picked, draw())
	}
	for range -luck {
		picked = min(picked, draw())
	}
	return picked
}

func jinxedHit(chance float64, luck int) bool {
	hit := rand.Float64() < chance
	for range -luck {
		hit = hit && rand.Float64() < chance
	}
	return hit
}

func resetList(columns []string) []string {
	commands := make([]string, len(columns))
	for i, column := range columns {
		commands[i] = cooldownCommands[column]
	}
	return commands
}

func resetCommands(columns []string) string {
	return core.JoinWithAnd(resetList(columns))
}

func perk(it item) string {
	return cmp.Or(effectPerk(it), usePerk(it))
}

func effectPerk(it item) string {
	if it.seed != nil {
		return seedPerk(it)
	}
	if it.graft != nil {
		return it.graft.perk()
	}
	if power, ok := primalPowers[it.id]; ok {
		return power.perk(it)
	}
	if whole, ok := itemByID(it.forms); ok {
		return fmt.Sprintf("junta %d na mochila e eles viram %s sozinhos", it.shards, whole.label())
	}
	if it.gear != nil {
		return gearPerk(it.gear)
	}
	if a, ok := activityUsing(it.id); ok {
		return fmt.Sprintf("é a ferramenta do %s, e quebra em %d%% das vezes", a.command, int(a.breaks*100))
	}
	protections := map[string]string{
		lockItemID:      "protege a mochila de um !furtar, e quebra segurando o ladrão",
		alarmItemID:     fmt.Sprintf("tem %d%% de chance de disparar num !roubar e salvar o saldo, e queima quando dispara. Contra !quadrilha a chance cai %d%% por pessoa a mais, e quem acabou de ganhar uma bolada na aposta perde mais %d%%. Cada pessoa só carrega %d na mochila", percent(alarmChance), percent(alarmGangCut), percent(jackpotAlarmCut), alarmCarry),
		vestItemID:      fmt.Sprintf("tem %d%% de chance de segurar um !matar que ia dar certo, e fica destruído quando segura a bala (contra deus e a Foice não adianta). Cada pessoa só carrega %d na mochila, e a polícia apreende se você for preso", percent(vestChance), vestCarry),
		forgedReleaseID: fmt.Sprintf("!usar tira você da cadeia na hora (ou !usar alvará @pessoa tira outra pessoa), mas em %d%% das vezes o carcereiro percebe e a pena aumenta %s. Conta como crime, e a polícia apreende se você for preso", percent(forgeryCaught), core.FormatWait(escapePenalty)),
	}
	if text, ok := protections[it.id]; ok {
		return text
	}
	if it.heal > 0 {
		return healPerk(it)
	}
	if it.arms != nil {
		return weaponPerk(it)
	}
	if it.high != nil {
		return drugPerk(it.high)
	}
	if it.heist > 0 {
		return fmt.Sprintf("soma +%d%% de chance pra equipe no !assalto, e é gasto no assalto (cada tipo de equipamento só conta uma vez por equipe)", percent(it.heist))
	}
	if it.flex != nil && it.illegal {
		return "vale só de ter na mochila: " + it.flex.effect + ". A polícia apreende se você for preso"
	}
	if it.flex != nil {
		return "vale só de ter na mochila: " + it.flex.effect
	}
	if it.trade != nil {
		return tradePerk(it.trade)
	}
	if it.networks != nil {
		return "vale só de ter na mochila: libera o !postar " + networkNames(it.networks)
	}
	if c, ok := charmByID(it.bless); ok {
		return "!usar dá " + c.summary()
	}
	if c, ok := charmByID(it.hex); ok {
		return fmt.Sprintf("!usar %s @pessoa joga %s. Pega em %d%% das vezes, e quando falha tem %d%% de chance de voltar pra você", strings.ToLower(it.name), c.summary(), percent(itemHexChance), percent(hexBackfire))
	}
	if it.loot != nil {
		return "!usar abre e sai um item aleatório"
	}
	if it.prizes != nil {
		top := slices.MaxFunc(it.prizes, func(a, b prize) int { return cmp.Compare(a.amount, b.amount) })
		return "!usar dá um prêmio de até " + Coins(top.amount)
	}
	if it.resets != nil {
		return fmt.Sprintf("!usar zera a espera do %s, mas "+it.limit.rule, resetCommands(it.resets), core.FormatWait(it.limit.cooldown.every))
	}
	if it.pet {
		return petPerk(it)
	}
	return ""
}

func usePerk(it item) string {
	if it.keep {
		return "dá pra !usar quantas vezes quiser, não gasta"
	}
	if it.use != "" {
		return "dá pra !usar uma vez, e aí acaba"
	}
	return ""
}

func sources(it item) []string {
	if it.rarity == primordial {
		return append(primordialSources(it), "o !criar ou o !milagre de deus")
	}
	if it.rarity == relic {
		var found []string
		for _, box := range relicBoxes() {
			found = append(found, fmt.Sprintf("%s %s (%d%% de chance)", box.emoji, box.name, percent(box.relicOdds)))
		}
		return append(found, "o !leilao (bem de vez em quando, com lance mínimo salgado)", "o !criar ou o !milagre de deus")
	}
	if base, ok := itemByID(it.upgrades); ok {
		return []string{fmt.Sprintf("!aprimorar %s, por %s", strings.ToLower(base.name), Coins(it.upgradeCost()))}
	}
	var found []string
	if it.shop {
		found = append(found, it.shelf())
	}
	if it.seed != nil {
		found = append(found, wildSeedSources(it)...)
	}
	if seed, ok := seedFor(it); ok && seed.id != it.id {
		found = append(found, "!colher (plantando "+seed.label()+")")
	}
	if luxuryGoods(it) {
		found = append(found, "!luxo (quando tá no estoque do dia)")
	}
	if auctionGoods(it) {
		found = append(found, "!leilao (no dia em que ele vai pro martelo)")
	}
	if it.graft != nil {
		return append(found, fmt.Sprintf("!mercadonegro (chega um implante por dia, sorteado entre os %d)", len(implants)))
	}
	if it.illegal {
		return append(found, "!mercadonegro (quando tá no estoque do dia)")
	}
	if hotTool(it) {
		found = append(found, fmt.Sprintf("!mercadonegro (%d%% de chance de chegar uma ferramenta lendária por dia, sorteada entre as %d)", percent(hotToolChance), len(hotTools())))
	}
	if hotGear(it) && !hotTool(it) {
		found = append(found, fmt.Sprintf("!mercadonegro (%d%% de chance de chegar a cada dia)", percent(hotGearChance)))
	}
	for _, a := range activities {
		if slices.ContainsFunc(a.loot, func(d drop) bool { return d.item == it.id }) {
			found = append(found, a.source(it))
		}
	}
	if work := workSource(it); work != "" {
		found = append(found, work)
	}
	if it.unboxable() && it.rarity != mythic {
		return found
	}
	for _, box := range catalog {
		if box.lootWeight(it.rarity) > 0 {
			found = append(found, box.emoji+" "+box.name)
		}
	}
	return found
}

func primordialSources(it item) []string {
	var found []string
	if it.id == skyRockID {
		found = append(found, starfallSource())
	}
	for _, box := range catalog {
		if box.shard == it.id {
			found = append(found, fmt.Sprintf("%s %s (%d%% de chance)", box.emoji, box.name, percent(box.shardOdds)))
		}
	}
	for _, piece := range catalog {
		if piece.forms == it.id {
			found = append(found, fmt.Sprintf("juntando %d %s (eles se montam sozinhos)", piece.shards, piece.label()))
		}
	}
	return found
}

func lootOdds(weights []int) string {
	total := 0
	for _, w := range weights {
		total += w
	}
	var odds []string
	for r, w := range weights {
		if w > 0 {
			odds = append(odds, fmt.Sprintf("%d%% %s", w*100/total, rarities[r].name))
		}
	}
	return core.JoinWithAnd(odds)
}

func prizeOdds(prizes []prize) string {
	total := 0
	for _, p := range prizes {
		total += p.weight
	}
	odds := make([]string, len(prizes))
	for i, p := range prizes {
		reward := "nada"
		if p.amount > 0 {
			reward = Coins(p.amount)
		}
		odds[i] = fmt.Sprintf("%d%% %s", p.weight*100/total, reward)
	}
	return core.JoinWithAnd(odds)
}

func (it item) resaleText() string {
	if it.illegal {
		return fmt.Sprintf("💸 O mercado não compra. No !mercadonegro, o receptador paga %d%% do preço do dia", blackMarketBuyback)
	}
	if it.rarity == mythic {
		return fmt.Sprintf("💸 O mercado paga %s, porque item mítico sai por pelo menos %d%% do valor", Coins(marketPayout(it, 1)), mythicSalePercent)
	}
	return "💸 O mercado paga " + Coins(marketPayout(it, 1))
}

func (it item) details() string {
	lines := []string{
		fmt.Sprintf("%s · #%d", it.label(), it.number()),
		it.summary(),
		it.resaleText(),
		"",
		"📍 Onde conseguir: " + core.JoinWithAnd(sources(it)),
	}
	if p := perk(it); p != "" {
		lines = append(lines, "✨ Pra que serve: "+p)
	}
	if up, ok := upgradeOf(it); ok {
		lines = append(lines, fmt.Sprintf("🔧 Dá pra !aprimorar uma vez por %s, e ela vira %s, que se paga mais rápido", Coins(up.upgradeCost()), up.label()))
	}
	if it.upgrades != "" {
		lines = append(lines, "🔧 Já é aprimorada, então não dá pra aprimorar de novo. Se quebrar, o aprimoramento vai junto")
	}
	if it.metal {
		lines = append(lines, "🧲 É de metal: o Ímã puxa ele no !vasculhar e no !minerar")
	}
	if it.loot != nil {
		lines = append(lines, "🎲 Chances: "+lootOdds(it.loot))
	}
	if it.relicOdds > 0 {
		lines = append(lines, fmt.Sprintf("🏛️ Em %d%% das vezes, no lugar disso, sai uma relíquia: %s", percent(it.relicOdds), relicNames()))
	}
	if shard, ok := itemByID(it.shard); ok {
		lines = append(lines, fmt.Sprintf("⚫ Em %d%% das vezes, no lugar disso, sai %s", percent(it.shardOdds), shard.label()))
	}
	if it.prizes != nil {
		lines = append(lines, "🎲 Prêmios: "+prizeOdds(it.prizes))
	}
	return strings.Join(lines, "\n")
}

func catalogText() string {
	lines := []string{fmt.Sprintf("📖 *Todos os %d itens*", len(catalog))}
	lines = append(lines, categoryGroups(catalog, func(it item) item { return it }, func(it item) string {
		return fmt.Sprintf("%s %s %s · 🪙 %s", rarities[it.rarity].emoji, it.emoji, it.name, core.Thousands(it.value))
	})...)
	lines = append(lines, "", rarityLegend(),
		"_Manda !item <tipo> pra ver um tipo só (tipo !item pesca) ou !item <nome> pra ver onde achar e o que cada um faz_")
	return strings.Join(lines, "\n")
}

func categoryText(c int) string {
	items := slices.DeleteFunc(slices.Clone(catalog), func(it item) bool { return it.category() != c })
	lines := []string{fmt.Sprintf("%s · %s", itemCategories[c].heading(), core.CountLabel(len(items), "item", "itens"))}
	lines = append(lines, rarityGroups(items, func(it item) item { return it }, func(it item) string {
		return fmt.Sprintf("%s %s · 🪙 %s", it.emoji, it.name, core.Thousands(it.value))
	})...)
	lines = append(lines, "", "_Manda !item <nome> pra ver onde achar e o que cada um faz_")
	return strings.Join(lines, "\n")
}

func (b *bot) item(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	query := parseItemQuery(core.MessageText(evt.Message)).name
	if query == "" {
		return b.SendText(ctx, chat, catalogText())
	}
	if c, ok := categoryNamed(query); ok {
		return b.SendText(ctx, chat, categoryText(c))
	}
	matches := findItems(query)
	if problem := itemQueryProblem(query, matches); problem != "" {
		return b.SendText(ctx, chat, problem)
	}
	return b.SendText(ctx, chat, matches[0].details())
}
