package economy

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

type drop struct {
	item   string
	weight int
}

type activity struct {
	command   string
	emoji     string
	cooldown  cooldown
	tool      string
	breaks    float64
	luck      int
	plainLuck int
	gradeLuck int
	rotten    bool
	starfall  bool
	mythics   bool
	hazard    hazard
	safety    int64
	seeds     float64
	loot      []drop
	found     []string
	empty     []string
	tired     string
}

var fishing = activity{
	command:  "!pescar",
	emoji:    "🎣",
	cooldown: cooldown{column: "pesca_em", every: 10 * time.Minute},
	hazard: hazard{chance: 0.04, harm: 20, wounds: []string{
		"escorregou na pedra lisa do rio e bateu a cabeça", "fisgou a própria orelha com o anzol",
		"foi puxado pra dentro do rio por um peixe gigante e quase se afogou", "pisou numa arraia escondida na areia",
	}},
	tool:   "vara-de-pesca",
	breaks: 0.05,
	seeds:  0.03,
	loot: []drop{
		{"", 180}, {"bota-velha", 80}, {"sapo", 60}, {"lambari", 170}, {"tilapia", 140}, {"camarao", 90},
		{"caranguejo", 80}, {"baiacu", 50}, {"peixe-palhaco", 40}, {"tambaqui", 50}, {"lula", 30}, {"polvo", 25},
		{"pirarucu", 20}, {"lagosta", 15}, {"jacare", 10}, {"perola", 10}, {"tubarao", 5}, {"bau-do-tesouro", 4},
		{"escama-de-sereia", 1}, {"sardinha", 90}, {"traira", 70}, {"pacu", 60}, {"estrela-do-mar", 40},
		{"celular-afogado", 40}, {"piranha", 35}, {"agua-viva", 30}, {"atum", 25}, {"arraia", 20},
		{"garrafa-com-mensagem", 12}, {"dourado", 10}, {"peixe-espada", 7}, {"tridente", 1},
		{"dente-de-megalodonte", 1}, {"boto-cor-de-rosa", 1}, {"relicario", 2}, {"bussola-do-relicario", 1},
		{"chave-de-sao-pedro", 3}, {"pente-da-iara", 1}, {"olho-de-boto", 1},
	},
	found: []string{
		"🎣 @%[1]s jogou a linha e fisgou %[2]s",
		"🎣 @%[1]s esperou um tempão na beira do rio e puxou %[2]s",
		"🎣 Depois de muita paciência, @%[1]s pescou %[2]s",
		"🎣 A vara de @%[1]s quase quebrou, mas veio %[2]s",
	},
	empty: []string{
		"🎣 @%s ficou 2 horas na beira do rio e só pegou sol ☀️",
		"🎣 O peixe comeu a isca de @%s e foi embora rindo",
		"🎣 @%s fisgou o próprio pé 🦶",
		"🎣 @%s passou a tarde pescando e só voltou com história de pescador",
	},
	tired: "🎣 @%s acabou de pescar e os peixes tão desconfiados. Volta em %s",
}

var mining = activity{
	command:  "!minerar",
	emoji:    "⛏️",
	cooldown: cooldown{column: "mineracao_em", every: 10 * time.Minute},
	hazard: hazard{chance: 0.07, harm: 30, wounds: []string{
		"levou uma pedra na cabeça quando o teto da mina cedeu", "acertou a picareta no próprio pé",
		"respirou poeira demais e saiu tossindo da mina", "ficou preso num desmoronamento e saiu cavando com a mão",
	}},
	tool:   "picareta",
	breaks: 0.06,
	seeds:  0.03,
	loot: []drop{
		{"", 150}, {"pedra", 200}, {"carvao", 150}, {"cobre", 120}, {"ferro", 110}, {"sal-grosso", 60},
		{"caveira", 40}, {"prata", 60}, {"ametista", 40}, {"ouro", 35}, {"moeda-do-imperio", 15}, {"vaso-antigo", 15},
		{"esmeralda", 15}, {"safira", 12}, {"rubi", 10}, {"fossil", 8}, {"diamante", 6}, {"bau-do-tesouro", 2},
		{"meteorito", 2}, {"areia", 110}, {"granito", 100}, {"quartzo", 70}, {"ouro-de-tolo", 60}, {"topazio", 35},
		{"geodo", 30}, {"opala", 12}, {"jade", 12}, {"petroleo", 10}, {"urna", 8}, {"uranio", 3},
		{"diamante-rosa", 1}, {"idolo-de-ouro", 1}, {"pedra-filosofal", 1}, {"relicario", 2}, {"bussola-do-relicario", 1},
		{"moeda-amaldicoada", 10}, {"ampulheta", 6}, {"machado-de-xango", 2}, {"pepita-do-eldorado", 1},
		{"fogo-da-mae-do-ouro", 1},
	},
	found: []string{
		"⛏️ @%[1]s cavou fundo na mina e achou %[2]s",
		"⛏️ Depois de muita picaretada, @%[1]s tirou %[2]s da parede",
		"⛏️ @%[1]s quase desmoronou a mina, mas saiu com %[2]s",
		"⛏️ No escuro da mina, @%[1]s tropeçou em %[2]s",
	},
	empty: []string{
		"⛏️ @%s cavou o dia inteiro e só achou terra",
		"⛏️ @%s bateu a picareta numa parede de concreto e o braço ficou tremendo por meia hora",
		"⛏️ @%s se perdeu na mina e levou 3 horas pra achar a saída",
		"⛏️ @%s cavou tanto que saiu do outro lado do mundo, mas de mão vazia",
	},
	tired: "⛏️ @%s ainda tá com o braço doendo da última vez. Volta em %s",
}

var scavenging = activity{
	command:  "!vasculhar",
	emoji:    "🗑️",
	cooldown: cooldown{column: "lixao_em", every: 10 * time.Minute},
	hazard: hazard{chance: 0.06, harm: 20, wounds: []string{
		"cortou a mão num caco de vidro escondido no lixo", "levou uma mordida de um rato do tamanho de um gato",
		"pisou num prego enferrujado e vai ter que tomar antitetânica", "brigou com um urubu e perdeu",
	}},
	seeds: 0.06,
	loot: []drop{
		{"", 210}, {"casca-de-banana", 85}, {"boleto", 65}, {"chinelo", 80}, {"cueca", 60}, {"jornal", 55},
		{"todynho-pela-metade", 70}, {"cabo-usb", 55}, {"guarda-chuva", 50}, {"controle-remoto", 50}, {"boneca", 40},
		{"pneu", 40}, {"cd-pirata", 35}, {"pote-de-feijao", 30}, {"rato", 25}, {"pombo", 20}, {"cadeira", 20},
		{"vhs", 15}, {"polystation", 10}, {"tamagotchi", 10}, {"carrinho", 8}, {"cofrinho", 20}, {"carteira-perdida", 25},
		{"nokia", 12}, {"nota-de-tres", 6}, {"gato-preto", 3}, {"iphone", 2}, {"garrafa-pet", 75}, {"anel-de-latinha", 50},
		{"dinheiro-de-brinquedo", 40}, {"panela-de-pressao", 35}, {"dentadura", 30}, {"peruca", 30}, {"colchao", 30},
		{"privada", 25}, {"sofa", 25}, {"tv-de-tubo", 15}, {"radinho", 15}, {"anao-de-jardim", 15},
		{"bilhete-esquecido", 15}, {"mala", 13}, {"bicicleta", 12}, {"album-da-copa", 12}, {"relogio-cuco", 8},
		{"tampinha", 70}, {"pilha-usada", 60}, {"tazo", 45}, {"fita-k7", 30}, {"tenis-furado", 60}, {"ventilador", 30},
		{"barbie", 30}, {"rolex-do-paraguai", 25}, {"manequim", 20}, {"urubu", 15}, {"disco-de-vinil", 15},
		{"maquina-de-escrever", 12}, {"notebook-positivo", 10}, {"game-boy", 8}, {"lata-de-ninho", 15},
		{"brinco-de-ouro", 5}, {"carta-holografica", 4}, {"cofre-esquecido", 2}, {"bilhete-premiado", 1},
		{"mala-de-dinheiro", 1}, {"taca-jules-rimet", 1}, {"relicario", 2}, {"bussola-do-relicario", 1},
		{"espelho-quebrado", 25}, {"despertador-quebrado", 30}, {"marmita-estragada", 30}, {"olho-de-sogra", 15},
		{"moeda-amaldicoada", 5}, {"pacto-com-o-capeta", 2}, {"chupa-cabra", 1}, {"boneco-do-fofao", 1},
	},
	found: []string{
		"🗑️ @%[1]s revirou o lixão e achou %[2]s",
		"🗑️ @%[1]s enfiou o braço inteiro na lixeira e puxou %[2]s",
		"🗑️ No meio do lixo, @%[1]s encontrou %[2]s",
		"🗑️ @%[1]s brigou com um urubu e ganhou %[2]s",
	},
	empty: []string{
		"🗑️ @%s revirou o lixão inteiro e só ganhou um cheiro que não sai mais da roupa",
		"🗑️ Um gato defendeu o lixo de @%s e venceu a briga 🐈",
		"🗑️ O caminhão do lixo passou antes de @%s e levou tudo",
		"🗑️ @%s achou um saco cheio de coisa boa, mas era a mudança do vizinho",
	},
	tired: "🗑️ @%s acabou de revirar o lixão e ainda tá fedendo. Volta em %s",
}

var bugging = activity{
	command:  "!catar",
	emoji:    "🦋",
	cooldown: cooldown{column: "inseto_em", every: 10 * time.Minute},
	hazard: hazard{chance: 0.05, harm: 20, wounds: []string{
		"levou uma ferroada de marimbondo bem no olho", "foi picado pela caranguejeira que tava escondida no tronco",
		"pisou num formigueiro de formiga-de-fogo", "cutucou uma colmeia e levou 30 picadas",
	}},
	tool:   "rede-de-insetos",
	breaks: 0.05,
	seeds:  0.10,
	loot: []drop{
		{"", 170}, {"folha-seca", 120}, {"teia-de-aranha", 80}, {"pernilongo", 120}, {"mosca-varejeira", 100},
		{"formiga", 120}, {"barata", 80}, {"mosquito-da-dengue", 60}, {"isca", 90}, {"grilo", 110}, {"caracol", 80},
		{"lagarta", 80}, {"joaninha", 90}, {"mariposa", 90}, {"tanajura", 70}, {"abelha", 70}, {"borboleta", 60},
		{"cigarra", 50}, {"vaga-lume", 45}, {"casa-de-marimbondo", 35}, {"louva-a-deus", 35}, {"bicho-pau", 30},
		{"favo-de-mel", 30}, {"escorpiao", 30}, {"caranguejeira", 20}, {"besouro-rinoceronte", 12},
		{"casulo-de-seda", 12}, {"borboleta-azul", 10}, {"abelha-rainha", 5}, {"mosquito-no-ambar", 4},
		{"escaravelho-de-ouro", 1}, {"fada", 1}, {"grilo-falante", 1}, {"saci-na-garrafa", 1}, {"relicario", 2}, {"bussola-do-relicario", 1},
		{"arruda", 50}, {"semente-de-roma", 30}, {"pena-do-uirapuru", 1}, {"pito-da-caipora", 1},
	},
	found: []string{
		"🦋 @%[1]s correu no mato com a rede e pegou %[2]s",
		"🦋 @%[1]s balançou a rede no jardim e veio %[2]s",
		"🦋 @%[1]s ficou parado igual estátua e, quando passou %[2]s, foi rede nele",
		"🦋 @%[1]s virou uma pedra e achou %[2]s embaixo",
	},
	empty: []string{
		"🦋 @%s correu meia hora atrás de uma borboleta e ela foi embora rindo",
		"🦋 @%s balançou a rede no vento e só pegou ar",
		"🦋 @%s tropeçou numa raiz e a rede ficou presa no galho",
		"🦋 @%s pisou num formigueiro e voltou pra casa sem nada, só se coçando",
	},
	tired: "🦋 @%s espantou todos os bichos do mato. Volta em %s",
}

var activities = []activity{fishing, mining, scavenging, bugging}

func extractionCommands() string {
	commands := make([]string, len(activities))
	for i, a := range activities {
		commands[i] = a.command
	}
	return core.JoinWithAnd(commands)
}

func activityUsing(tool string) (activity, bool) {
	i := slices.IndexFunc(activities, func(a activity) bool { return a.tool == tool })
	if i < 0 {
		return activity{}, false
	}
	return activities[i], true
}

func activityNamed(command string) activity {
	return activities[slices.IndexFunc(activities, func(a activity) bool { return a.command == command })]
}

type gear struct {
	activities  []string
	tool        bool
	sturdy      int64
	grace       int64
	luck        int
	faster      int64
	fewerJunk   int64
	moreRare    int64
	grade       int
	safety      int64
	relicLure   int
	skipsRelics bool
	spent       bool
	metal       bool
	alwaysFinds bool
	pull        int
}

func (g *gear) fits(a activity) bool {
	return g != nil && slices.Contains(g.activities, a.command)
}

type kit struct {
	activity
	armed bool
	grace int64
	pull  int
	worn  []item
	spent []item
}

func (a activity) equip(bag []stack) kit {
	k := kit{activity: a, armed: a.tool == "" || slices.ContainsFunc(bag, func(s stack) bool { return s.item.id == a.tool })}
	best, hasTool := a.bestTool(bag)
	var metalOnly, lured bool
	for _, s := range bag {
		g := s.item.gear
		if !g.fits(a) || g.tool && s.item.id != best.id {
			continue
		}
		k.worn = append(k.worn, s.item)
		if g.skipsRelics {
			k.plainLuck += g.luck
		} else {
			k.luck += g.luck
		}
		k.gradeLuck += g.grade
		k.safety += g.safety
		k.pull += g.pull
		k.cooldown.every = k.cooldown.every * time.Duration(100-g.faster) / 100
		if g.spent {
			k.spent = append(k.spent, s.item)
		}
		if g.fewerJunk > 0 {
			k.loot = trimJunk(k.loot, g.fewerJunk)
		}
		if g.moreRare > 0 {
			k.loot = boostRare(k.loot, g.moreRare)
		}
		if g.relicLure > 0 {
			k.loot = lureRelics(k.loot, g.relicLure)
		}
		if g.alwaysFinds {
			k.loot = slices.DeleteFunc(slices.Clone(k.loot), func(d drop) bool { return d.item == "" })
		}
		metalOnly = metalOnly || g.metal
		lured = lured || g.relicLure > 0
	}
	if !lured {
		k.plainLuck += k.luck
		k.luck = 0
	}
	k.starfall = slices.ContainsFunc(k.worn, extractionRelic)
	if !k.mythics && !slices.ContainsFunc(k.worn, func(it item) bool { return it.rarity >= relic }) {
		k.loot = slices.DeleteFunc(slices.Clone(k.loot), func(d drop) bool { it, _ := itemByID(d.item); return it.rarity == mythic })
	}
	if metalOnly {
		k.loot = slices.DeleteFunc(slices.Clone(k.loot), func(d drop) bool { it, _ := itemByID(d.item); return !it.metal })
	}
	if !hasTool || a.tool == "" {
		return k
	}
	k.tool = best.id
	k.breaks = a.breaks * float64(100-best.gear.sturdy) / 100
	k.grace = best.gear.grace
	k.armed = true
	return k
}

func (a activity) bestTool(bag []stack) (item, bool) {
	var best item
	for _, s := range bag {
		if s.item.gear.fits(a) && s.item.gear.tool && s.item.value > best.value {
			best = s.item
		}
	}
	return best, best.id != ""
}

func (a activity) source(it item) string {
	if it.rarity != mythic {
		return a.command
	}
	var unlockers []string
	for _, r := range catalog {
		if r.rarity >= relic && r.gear.fits(a) {
			unlockers = append(unlockers, r.name)
		}
	}
	return a.command + " usando " + core.JoinWithOr(unlockers)
}

func trimJunk(loot []drop, percent int64) []drop {
	trimmed := slices.Clone(loot)
	for i, d := range trimmed {
		it, _ := itemByID(d.item)
		if d.item != "" && it.rarity != common {
			continue
		}
		trimmed[i].weight = int(int64(d.weight) * (100 - percent) / 100)
	}
	return trimmed
}

func boostRare(loot []drop, percent int64) []drop {
	boosted := slices.Clone(loot)
	for i, d := range boosted {
		factor := int64(100)
		if it, _ := itemByID(d.item); d.item != "" && it.rarity >= rare {
			factor += percent
		}
		boosted[i].weight = int(int64(d.weight) * factor)
	}
	return boosted
}

func lureRelics(loot []drop, factor int) []drop {
	lured := slices.Clone(loot)
	for i, d := range lured {
		if it, _ := itemByID(d.item); it.relicOdds > 0 {
			lured[i].weight *= factor
		}
	}
	return lured
}

func lureCompasses(loot []drop, factor int) []drop {
	lured := slices.Clone(loot)
	for i, d := range lured {
		if it, _ := itemByID(d.item); it.gear != nil && it.gear.relicLure > 0 {
			lured[i].weight *= factor
		}
	}
	return lured
}

func (p Profession) forage(a activity) activity {
	a.cooldown.every = p.hasted(a.cooldown.every * time.Duration(100-p.gatherFaster) / 100)
	if p.compassLure > 0 {
		a.loot = lureCompasses(a.loot, p.compassLure)
	}
	if barren := p.barrenHauls + int64(p.hangover*hangoverBarren); barren != 0 {
		a.loot = barrenLoot(a.loot, barren)
	}
	if p.gatherRare > 0 {
		a.loot = boostRare(a.loot, p.gatherRare)
	}
	if p.gatherJunk > 0 {
		a.loot = trimJunk(a.loot, p.gatherJunk)
	}
	a.plainLuck += p.gatherLuck
	a.gradeLuck += p.gatherGrade
	a.rotten = p.rotten
	return a
}

func barrenLoot(loot []drop, percent int64) []drop {
	barren := slices.Clone(loot)
	for i, d := range barren {
		if d.item == "" {
			barren[i].weight = int(int64(d.weight) * (100 + percent) / 100)
		}
	}
	return barren
}

func gearPerk(g *gear) string {
	parts := []struct {
		on   bool
		text string
	}{
		{g.tool, "serve de ferramenta"},
		{g.sturdy >= 100, "nunca quebra"},
		{g.sturdy > 0 && g.sturdy < 100, fmt.Sprintf("quebra %d%% menos", g.sturdy)},
		{g.grace > 0, fmt.Sprintf("não quebra nos primeiros %d usos", g.grace)},
		{g.luck > 0, fmt.Sprintf("dá %dx mais sorte no que acha", g.luck+1)},
		{g.skipsRelics, "mas essa sorte não ajuda a achar o Relicário perdido, nem com a bússola"},
		{g.faster > 0, fmt.Sprintf("espera %d%% menos", g.faster)},
		{g.fewerJunk > 0, fmt.Sprintf("vem %d%% menos item comum e mão vazia", g.fewerJunk)},
		{g.moreRare > 0, fmt.Sprintf("vem %d%% mais item raro, épico e lendário", g.moreRare)},
		{g.grade > 0, fmt.Sprintf("o que acha vem com qualidade melhor: fica com a melhor de %d", g.grade+1)},
		{g.safety > 0, fmt.Sprintf("se machuca %d%% menos", g.safety)},
		{g.relicLure > 0, fmt.Sprintf("deixa %dx mais fácil achar o Relicário perdido, e só com ela a sorte dos outros itens e da profissão ajuda a achar ele", g.relicLure)},
		{g.metal, "só sai item de metal, então nunca vem de mão vazia"},
		{g.alwaysFinds, "nunca vem de mão vazia"},
		{g.pull > 0, fmt.Sprintf("traz %d itens de cada vez", g.pull+1)},
		{g.spent, "gasta 1 a cada vez"},
	}
	var found []string
	for _, p := range parts {
		if p.on {
			found = append(found, p.text)
		}
	}
	return "no " + core.JoinWithAnd(g.activities) + ", " + core.JoinWithAnd(found)
}

func (a activity) drop() item {
	d := a.loot[weightedIndex(len(a.loot), func(i int) int { return a.loot[i].weight })]
	found, _ := itemByID(d.item)
	return found
}

func (a activity) roll() haul {
	h := a.find()
	h.broke = a.tool != "" && rand.Float64() < a.breaks
	return h
}

func (a activity) find() haul {
	h := haul{found: a.pick()}
	if h.found.gradable() {
		h.grade = rollQuality(a.gradeLuck, a.rotten)
	}
	return h
}

func (a activity) stray() item {
	if rand.Float64() >= a.seeds {
		return item{}
	}
	return wildSeed()
}

func (a activity) pick() item {
	if a.starfall && rand.Float64() < starfallOdds {
		rock, _ := itemByID(skyRockID)
		return rock
	}
	found := a.drop()
	for range a.luck {
		if other := a.drop(); other.value > found.value {
			found = other
		}
	}
	for range a.plainLuck {
		if other := a.drop(); other.relicOdds == 0 && other.value > found.value {
			found = other
		}
	}
	return found
}

type extraction struct {
	text   string
	hurt   string
	people []types.JID
	spot   eventSpot
	went   bool
	died   bool
}

func (x extraction) eventful() bool {
	return x.went && !x.died
}

func (b *bot) extract(ctx context.Context, chat, person types.JID, acc core.Account, job Profession, base activity, now time.Time) (extraction, error) {
	pit := b.craterIn(chat, base, now)
	a := pit.dig(job.forage(job.craft.apply(base)))
	a.cooldown.every = a.cooldown.every * time.Duration(100+job.decrees.level(gatherDecree)) / 100
	g, err := b.Wallets.gather(ctx, chat, acc, a, pit, now)
	if err != nil {
		return extraction{}, fmt.Errorf("%s: %w", a.command, err)
	}
	went := !g.noTool && g.wait == 0
	if went {
		b.advance(chat, acc, gatherGoal, int64(g.turns))
		b.practice(ctx, chat, acc, job, func(p Profession) bool { return p.crafts(a.command) }, g.turns)
	}
	hurt, freed := g.injury.text(person)
	people := append([]types.JID{person}, freed...)
	if g.toll > 0 {
		people = append(people, pit.owner.JID)
	}
	return extraction{
		text:   a.report(g, person) + pit.note(g),
		hurt:   hurt,
		people: people,
		spot:   eventSpot{place: a.command, won: slices.ContainsFunc(g.finds(), func(h haul) bool { return h.found.id != "" })},
		went:   went,
		died:   g.injury.died,
	}, nil
}

func (b *bot) gather(base activity) core.Handler {
	return func(ctx context.Context, evt *events.Message) error {
		chat := evt.Info.Chat.ToNonAD()
		person := evt.Info.Sender.ToNonAD()
		acc := b.Account(ctx, person)
		job, err := b.ProfessionOf(ctx, chat, acc)
		if err != nil {
			return err
		}
		x, err := b.extract(ctx, chat, person, acc, job, base, time.Now())
		if err != nil {
			return err
		}
		_, err = b.SendMentions(ctx, chat, x.text+job.decrees.note(gatherDecree)+x.hurt, x.people)
		if err != nil || !x.eventful() {
			return err
		}
		b.stirEvent(ctx, evt, x.spot)
		return nil
	}
}

func (a activity) report(g gathering, person types.JID) string {
	tool, _ := itemByID(a.tool)
	if g.noTool {
		return fmt.Sprintf("%s Pra usar o %s, @%s precisa de %s. Tem na !loja por %s", a.emoji, a.command, person.User, tool.label(), Coins(tool.value))
	}
	if g.wait > 0 {
		return fmt.Sprintf(a.tired, person.User, core.FormatWait(g.wait))
	}
	text := fmt.Sprintf(core.PickOne(a.empty), person.User)
	if g.found.id != "" {
		text = fmt.Sprintf(core.PickOne(a.found), person.User, g.found.label()+g.grade.tag()) + "\n" + g.grade.summary(g.found)
	}
	text += backlogReport(g.backlog)
	for _, extra := range g.pulled {
		if extra.found.id != "" {
			text += fmt.Sprintf("\n\n☄️ A gravidade do Bendegó puxou junto %s\n%s", extra.found.label()+extra.grade.tag(), extra.grade.summary(extra.found))
		}
	}
	if len(g.seeds) > 0 {
		labels := make([]string, len(g.seeds))
		for i, seed := range g.seeds {
			labels[i] = seed.label()
		}
		text += fmt.Sprintf("\n\n🌱 De quebra, achou %s no caminho. Dá pra plantar com !plantar", core.JoinWithAnd(labels))
	}
	text += findNotes(g.finds())
	if len(g.worn) > 0 {
		text += "\n\n🧰 Com " + wornList(g.worn, g.spent)
	}
	if g.broken.id != "" {
		text += fmt.Sprintf("\n\n💥 %s quebrou! Compra outra na !loja", g.broken.label())
	}
	return text
}

func findNotes(finds []haul) string {
	notes := []struct {
		hit  func(item) bool
		text func(item) string
	}{
		{func(it item) bool { return it.rarity == mythic }, func(item) string {
			return fmt.Sprintf("🟠 *MÍTICO!* Isso só aparece pra quem extrai com relíquia, e o mercado paga pelo menos %d%% do valor dele no !vender", mythicSalePercent)
		}},
		{func(it item) bool { return it.relicOdds > 0 }, func(it item) string {
			return fmt.Sprintf("🏛️ Achado raríssimo! Abre com !usar %s e reza: em %d%% das vezes sai uma relíquia", core.FoldLetters(it.name), percent(it.relicOdds))
		}},
		{func(it item) bool { return it.id == skyRockID }, starfallNote},
	}
	var text string
	for _, n := range notes {
		if i := slices.IndexFunc(finds, func(h haul) bool { return n.hit(h.found) }); i >= 0 {
			text += "\n\n" + n.text(finds[i].found)
		}
	}
	return text
}

func wornList(worn, spent []item) string {
	labels := make([]string, len(worn))
	for i, it := range worn {
		labels[i] = it.label()
		var used int
		for _, s := range spent {
			if s.id == it.id {
				used++
			}
		}
		if used > 0 {
			labels[i] += fmt.Sprintf(" (gastou %d)", used)
		}
	}
	return core.JoinWithAnd(labels)
}
