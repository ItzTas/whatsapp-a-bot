package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

type drop struct {
	item   string
	weight int
}

type activity struct {
	command  string
	emoji    string
	cooldown cooldown
	tool     string
	breaks   float64
	luck     int
	loot     []drop
	found    []string
	empty    []string
	tired    string
}

var fishing = activity{
	command:  "!pescar",
	emoji:    "🎣",
	cooldown: cooldown{column: "pesca_em", every: 10 * time.Minute},
	tool:     "vara-de-pesca",
	breaks:   0.05,
	loot: []drop{
		{"", 180}, {"bota-velha", 80}, {"sapo", 60}, {"lambari", 170}, {"tilapia", 140}, {"camarao", 90},
		{"caranguejo", 80}, {"baiacu", 50}, {"peixe-palhaco", 40}, {"tambaqui", 50}, {"lula", 30}, {"polvo", 25},
		{"pirarucu", 20}, {"lagosta", 15}, {"jacare", 10}, {"perola", 10}, {"tubarao", 5}, {"bau-do-tesouro", 4},
		{"escama-de-sereia", 1}, {"sardinha", 90}, {"traira", 70}, {"pacu", 60}, {"estrela-do-mar", 40},
		{"celular-afogado", 40}, {"piranha", 35}, {"agua-viva", 30}, {"atum", 25}, {"arraia", 20},
		{"garrafa-com-mensagem", 12}, {"dourado", 10}, {"peixe-espada", 7}, {"tridente", 1},
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
	tool:     "picareta",
	breaks:   0.06,
	loot: []drop{
		{"", 150}, {"pedra", 200}, {"carvao", 150}, {"cobre", 120}, {"ferro", 110}, {"sal-grosso", 60},
		{"caveira", 40}, {"prata", 60}, {"ametista", 40}, {"ouro", 35}, {"moeda-do-imperio", 15}, {"vaso-antigo", 15},
		{"esmeralda", 15}, {"safira", 12}, {"rubi", 10}, {"fossil", 8}, {"diamante", 6}, {"bau-do-tesouro", 2},
		{"meteorito", 2}, {"areia", 110}, {"granito", 100}, {"quartzo", 70}, {"ouro-de-tolo", 60}, {"topazio", 35},
		{"geodo", 30}, {"opala", 12}, {"jade", 12}, {"petroleo", 10}, {"urna", 8}, {"uranio", 3},
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
	command:  "!cacar",
	emoji:    "🦋",
	cooldown: cooldown{column: "inseto_em", every: 10 * time.Minute},
	tool:     "rede-de-insetos",
	breaks:   0.05,
	loot: []drop{
		{"", 170}, {"folha-seca", 120}, {"teia-de-aranha", 80}, {"pernilongo", 120}, {"mosca-varejeira", 100},
		{"formiga", 120}, {"barata", 80}, {"mosquito-da-dengue", 60}, {"isca", 90}, {"grilo", 110}, {"caracol", 80},
		{"lagarta", 80}, {"joaninha", 90}, {"mariposa", 90}, {"tanajura", 70}, {"abelha", 70}, {"borboleta", 60},
		{"cigarra", 50}, {"vaga-lume", 45}, {"casa-de-marimbondo", 35}, {"louva-a-deus", 35}, {"bicho-pau", 30},
		{"favo-de-mel", 30}, {"escorpiao", 30}, {"caranguejeira", 20}, {"besouro-rinoceronte", 12},
		{"casulo-de-seda", 12}, {"borboleta-azul", 10}, {"abelha-rainha", 5}, {"mosquito-no-ambar", 4},
		{"escaravelho-de-ouro", 1}, {"fada", 1},
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
	activities []string
	tool       bool
	sturdy     int64
	luck       int
	faster     int64
	fewerJunk  int64
	spent      bool
	metal      bool
}

func (g *gear) fits(a activity) bool {
	return g != nil && slices.Contains(g.activities, a.command)
}

type kit struct {
	activity
	armed bool
	worn  []item
	spent []item
}

func (a activity) equip(bag []stack) kit {
	k := kit{activity: a, armed: a.tool == "" || slices.ContainsFunc(bag, func(s stack) bool { return s.item.id == a.tool })}
	for _, s := range bag {
		g := s.item.gear
		if !g.fits(a) {
			continue
		}
		k.worn = append(k.worn, s.item)
		k.luck += g.luck
		k.cooldown.every = k.cooldown.every * time.Duration(100-g.faster) / 100
		if g.spent {
			k.spent = append(k.spent, s.item)
		}
		if g.fewerJunk > 0 {
			k.loot = trimJunk(k.loot, g.fewerJunk)
		}
		if g.metal {
			k.loot = slices.DeleteFunc(slices.Clone(k.loot), func(d drop) bool { it, _ := itemByID(d.item); return !it.metal })
		}
		if g.tool && a.tool != "" {
			k.tool = s.item.id
			k.breaks = a.breaks * float64(100-g.sturdy) / 100
			k.armed = true
		}
	}
	return k
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

func gearPerk(g *gear) string {
	parts := []struct {
		on   bool
		text string
	}{
		{g.tool, "serve de ferramenta"},
		{g.sturdy > 0, fmt.Sprintf("quebra %d%% menos", g.sturdy)},
		{g.luck > 0, fmt.Sprintf("dá %dx mais sorte no que acha", g.luck+1)},
		{g.faster > 0, fmt.Sprintf("espera %d%% menos", g.faster)},
		{g.fewerJunk > 0, fmt.Sprintf("vem %d%% menos item comum e mão vazia", g.fewerJunk)},
		{g.metal, "só sai item de metal, então nunca vem de mão vazia"},
		{g.spent, "gasta 1 a cada vez"},
	}
	var found []string
	for _, p := range parts {
		if p.on {
			found = append(found, p.text)
		}
	}
	return "no " + joinWithAnd(g.activities) + ", " + joinWithAnd(found)
}

func (a activity) drop() item {
	d := a.loot[weightedIndex(len(a.loot), func(i int) int { return a.loot[i].weight })]
	found, _ := itemByID(d.item)
	return found
}

func (a activity) roll() haul {
	found := a.drop()
	for range a.luck {
		if other := a.drop(); other.value > found.value {
			found = other
		}
	}
	return haul{found: found, broke: a.tool != "" && rand.Float64() < a.breaks}
}

func (b *bot) gather(base activity) command {
	return func(ctx context.Context, evt *events.Message) error {
		chat := evt.Info.Chat.ToNonAD()
		person := evt.Info.Sender.ToNonAD()
		acc := b.account(ctx, person)
		job, err := b.professionOf(ctx, chat, acc)
		if err != nil {
			return err
		}
		a := job.craft.apply(base)
		a.cooldown.every = a.cooldown.every * time.Duration(100+job.decrees.level(gatherDecree)) / 100
		g, err := b.wallets.gather(ctx, chat, acc, a, time.Now())
		if err != nil {
			return fmt.Errorf("%s: %w", a.command, err)
		}
		_, err = b.sendMentions(ctx, chat, a.report(g, person)+job.decrees.note(gatherDecree), []types.JID{person})
		return err
	}
}

func (a activity) report(g gathering, person types.JID) string {
	tool, _ := itemByID(a.tool)
	if g.noTool {
		return fmt.Sprintf("%s Pra usar o %s, @%s precisa de %s. Tem na !loja por %s", a.emoji, a.command, person.User, tool.label(), coins(tool.value))
	}
	if g.wait > 0 {
		return fmt.Sprintf(a.tired, person.User, formatWait(g.wait))
	}
	text := fmt.Sprintf(pickOne(a.empty), person.User)
	if g.found.id != "" {
		text = fmt.Sprintf(pickOne(a.found), person.User, g.found.label()) + "\n" + g.found.summary()
	}
	if len(g.worn) > 0 {
		text += "\n\n🧰 Com " + wornList(g.worn, g.spent)
	}
	if g.broke {
		text += fmt.Sprintf("\n\n💥 %s quebrou! Compra outra na !loja", g.broken.label())
	}
	return text
}

func wornList(worn, spent []item) string {
	labels := make([]string, len(worn))
	for i, it := range worn {
		labels[i] = it.label()
		if slices.ContainsFunc(spent, func(s item) bool { return s.id == it.id }) {
			labels[i] += " (gastou 1)"
		}
	}
	return joinWithAnd(labels)
}
