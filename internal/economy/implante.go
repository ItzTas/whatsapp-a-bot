package economy

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	implantUpkeepGrace = 24 * time.Hour
	implantUpkeepMax   = 7 * 24 * time.Hour
	implantUpkeepDay   = 24 * time.Hour
	implantRejectHarm  = 60
	implantRipChance   = 0.05
	implantWearMax     = 3
	corpoSecoItem      = "coracao-do-corpo-seco"
)

var (
	errPartTaken  = errors.New("parte do corpo ocupada")
	errNoImplant  = errors.New("sem implante")
	errUpkeepPaid = errors.New("manutenção em dia")
)

const (
	partBrain = iota
	partEyes
	partArm
)

type bodyPart struct {
	key     string
	name    string
	emoji   string
	place   string
	from    string
	aliases []string
}

var bodyParts = [...]bodyPart{
	partBrain: {key: "cerebro", name: "Cérebro", emoji: "🧠", place: "no cérebro", from: "do cérebro", aliases: []string{"cabeca", "mente"}},
	partEyes:  {key: "olhos", name: "Olhos", emoji: "👁️", place: "nos olhos", from: "dos olhos", aliases: []string{"olho", "visao"}},
	partArm:   {key: "braco", name: "Braço", emoji: "💪", place: "no braço", from: "do braço", aliases: []string{"bracos", "mao", "maos"}},
}

type implant struct {
	id      string
	part    int
	emoji   string
	name    string
	price   int64
	upkeep  int64
	reject  float64
	illegal bool
	aliases []string
	bonus   flex
}

var implants = []implant{
	{id: "chip-de-produtividade", part: partBrain, emoji: "💾", name: "Chip de produtividade", price: 10_000, upkeep: 120, reject: 0.10, aliases: []string{"produtividade"},
		bonus: flex{effect: "o !trabalhar paga 20% a mais", apply: func(p Profession) Profession {
			p.min, p.max = p.min*120/100, p.max*120/100
			return p
		}}},
	{id: "coprocessador-quantico", part: partBrain, emoji: "⚛️", name: "Coprocessador quântico", price: 18_000, upkeep: 200, reject: 0.15, aliases: []string{"coprocessador", "quantico"},
		bonus: flex{effect: "o !trabalhar recarrega 20% mais rápido", apply: func(p Profession) Profession {
			p.every = p.every * 80 / 100
			return p
		}}},
	{id: "chip-de-day-trade", part: partBrain, emoji: "📈", name: "Chip de day trade", price: 15_000, upkeep: 150, reject: 0.10, aliases: []string{"day trade", "trader"},
		bonus: flex{effect: fmt.Sprintf("o lucro na venda do !acoes rende 25%% a mais e não paga a corretagem de %d%%", stockFeePercent), apply: func(p Profession) Profession {
			p.stockProfit += 25
			p.freeBrokerage = true
			return p
		}}},
	{id: "chip-do-golpe", part: partBrain, emoji: "🃏", name: "Chip do golpe", price: 12_000, upkeep: 200, reject: 0.20, illegal: true, aliases: []string{"golpe", "golpista"},
		bonus: flex{effect: "+15% de chance no !roubar e no !furtar, e leva 25% a mais do que rouba no !roubar e na !quadrilha", apply: func(p Profession) Profession {
			p.robBonus += 0.15
			p.theftBonus += 0.15
			p.robHaul += 25
			return p
		}}},
	{id: "olho-scanner", part: partEyes, emoji: "🔍", name: "Olho scanner", price: 12_000, upkeep: 150, reject: 0.10, aliases: []string{"scanner"},
		bonus: flex{effect: "+10% de chance de achar item no !trabalhar, e item raro ou acima aparece 25% mais no " + extractionCommands(), apply: func(p Profession) Profession {
			p.findBonus += 0.10
			p.gatherRare += 25
			return p
		}}},
	{id: "lente-avaliadora", part: partEyes, emoji: "🧐", name: "Lente avaliadora", price: 15_000, upkeep: 150, reject: 0.10, aliases: []string{"lente", "avaliadora"},
		bonus: flex{effect: "o mercado paga 10 pontos a mais do valor no !vender (até 95%)", apply: func(p Profession) Profession {
			p.salePercent = min(p.marketPercent()+10, 95)
			return p
		}}},
	{id: "olho-de-aguia", part: partEyes, emoji: "🦅", name: "Olho de águia", price: 35_000, upkeep: 400, reject: 0.20, aliases: []string{"aguia"},
		bonus: flex{effect: "no " + extractionCommands() + ", sorteia duas vezes e fica com o achado mais valioso (caixa de relíquia não conta no sorteio extra)", apply: func(p Profession) Profession {
			p.gatherLuck++
			return p
		}}},
	{id: "visao-de-raio-x", part: partEyes, emoji: "🩻", name: "Visão de raio-x", price: 12_000, upkeep: 200, reject: 0.20, illegal: true, aliases: []string{"raio x", "raiox"},
		bonus: flex{effect: "+20% de chance no !furtar e +10% pra equipe no !assalto", apply: func(p Profession) Profession {
			p.theftBonus += 0.20
			p.heistBonus += 0.10
			return p
		}}},
	{id: "braco-hidraulico", part: partArm, emoji: "🦾", name: "Braço hidráulico", price: 25_000, upkeep: 250, reject: 0.15, aliases: []string{"hidraulico"},
		bonus: flex{effect: "espera 20% menos no " + extractionCommands(), apply: func(p Profession) Profession {
			p.gatherFaster += 20
			return p
		}}},
	{id: "dedos-bionicos", part: partArm, emoji: "🖐️", name: "Dedos biônicos", price: 12_000, upkeep: 150, reject: 0.10, aliases: []string{"dedos", "bionicos"},
		bonus: flex{effect: "leva 15% a mais de lucro quando ganha no " + casinoGames, apply: func(p Profession) Profession {
			p.casinoLuck += 15
			return p
		}}},
	{id: "braco-blindado", part: partArm, emoji: "🛡️", name: "Braço blindado", price: 15_000, upkeep: 150, reject: 0.10, aliases: []string{"blindado"},
		bonus: flex{effect: "quem tenta !roubar, !furtar, !sequestrar ou !matar você tem 15% menos chance", apply: func(p Profession) Profession {
			p.guard += 0.15
			return p
		}}},
	{id: "braco-canhao", part: partArm, emoji: "🔫", name: "Braço-canhão", price: 20_000, upkeep: 300, reject: 0.25, illegal: true, aliases: []string{"canhao"},
		bonus: flex{effect: "+25% de chance no !matar, que fica liberado pra qualquer profissão, e +10% no !roubar", apply: func(p Profession) Profession {
			p.killChance += 0.25
			p.robBonus += 0.10
			return p
		}}},
}

type installed struct {
	implant
	paidUntil int64
	wear      int64
}

type surgery struct {
	occupant implant
	rejected bool
	spared   bool
	injury   injury
}

type unplugging struct {
	implant
	broke bool
	wear  int64
}

type rowsQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func implantGoods() []item {
	goods := make([]item, len(implants))
	for i := range implants {
		im := &implants[i]
		goods[i] = item{id: im.id, name: im.name, emoji: im.emoji, rarity: legendary, value: im.price, illegal: im.illegal, aliases: im.aliases, graft: im}
	}
	return goods
}

func implantByID(id string) (implant, bool) {
	i := slices.IndexFunc(implants, func(im implant) bool { return im.id == id })
	if i < 0 {
		return implant{}, false
	}
	return implants[i], true
}

func (im implant) label() string {
	return im.emoji + " *" + im.name + "*"
}

func (im implant) organ() bodyPart {
	return bodyParts[im.part]
}

func (im implant) matchKeys() []string {
	keys := []string{normalizeItemName(im.name), normalizeItemName(im.id)}
	for _, alias := range im.aliases {
		keys = append(keys, normalizeItemName(alias))
	}
	return keys
}

func (im implant) risk() hazard {
	return hazard{chance: im.reject, harm: implantRejectHarm, wounds: []string{
		"saiu da mesa de cirurgia com uma hemorragia",
		"pegou uma infecção daquelas no pós-operatório",
		"teve febre de 40 graus na recuperação",
	}}
}

func (im implant) menuLine() string {
	tag := ""
	if im.illegal {
		tag = " 🚨 clandestino"
	}
	return fmt.Sprintf("• %s%s: vale %s · manutenção de %s por dia · %d%% de rejeição\n    ✨ %s", im.label(), tag, Coins(im.price), Coins(im.upkeep), percent(im.reject), im.bonus.effect)
}

func (im implant) perk() string {
	text := fmt.Sprintf("!implante instalar (ou !usar) põe ele %s: %s. Manutenção de %s por dia e %d%% de rejeição (rejeitado, o implante se perde). Cada vez que sai do corpo ele desgasta, e na %dª vez quebra",
		im.organ().place, im.bonus.effect, Coins(im.upkeep), percent(im.reject), implantWearMax)
	if !im.illegal {
		return text
	}
	return text + ". É clandestino: a polícia apreende da mochila e arranca do corpo de quem for preso"
}

func findImplant(query string) (implant, bool) {
	q := normalizeItemName(query)
	if q == "" {
		return implant{}, false
	}
	matchers := []func(key string) bool{
		func(key string) bool { return key == q },
		func(key string) bool { return strings.HasPrefix(key, q) },
		func(key string) bool { return strings.Contains(key, q) },
	}
	for _, match := range matchers {
		i := slices.IndexFunc(implants, func(im implant) bool { return slices.ContainsFunc(im.matchKeys(), match) })
		if i >= 0 {
			return implants[i], true
		}
	}
	return implant{}, false
}

func findBodyPart(query string) (int, bool) {
	q := normalizeItemName(query)
	i := slices.IndexFunc(bodyParts[:], func(bp bodyPart) bool {
		return q == bp.key || q == normalizeItemName(bp.name) || slices.ContainsFunc(bp.aliases, func(alias string) bool { return normalizeItemName(alias) == q })
	})
	return i, i >= 0
}

func lethalImplants() string {
	var labels []string
	for _, im := range implants {
		if im.bonus.apply(Profession{}).killChance > 0 {
			labels = append(labels, im.label())
		}
	}
	return core.JoinWithAnd(labels)
}

func implantLabels(ids []string) string {
	labels := make([]string, len(ids))
	for i, id := range ids {
		im, _ := implantByID(id)
		labels[i] = im.label()
	}
	return core.JoinWithAnd(labels)
}

func (in installed) powered(now time.Time) bool {
	return in.paidUntil > now.Unix()
}

func (in installed) upkeepTo(target, now int64) int64 {
	from := max(now, in.paidUntil)
	if from >= target {
		return 0
	}
	return in.upkeep * (target - from) / int64(implantUpkeepDay/time.Second)
}

func (in installed) status(now time.Time) string {
	wear := fmt.Sprintf(", desgaste %d/%d", in.wear, implantWearMax)
	if in.powered(now) {
		return "🔋 ligado, manutenção paga por " + upkeepSpan(time.Unix(in.paidUntil, 0).Sub(now)) + wear
	}
	return "🔌 desligado: falta pagar a !implante manutencao" + wear
}

func upkeepSpan(d time.Duration) string {
	if d < implantUpkeepDay {
		return core.FormatWait(d)
	}
	return core.CountLabel(int(d/implantUpkeepDay), "dia", "dias")
}

func upkeepDue(body []installed, now time.Time) int64 {
	target := now.Add(implantUpkeepMax).Unix()
	var due int64
	for _, in := range body {
		due += in.upkeepTo(target, now.Unix())
	}
	return due
}

func pickInstalled(body []installed, query string) (installed, bool) {
	match := func(installed) bool { return false }
	if im, ok := findImplant(query); ok {
		match = func(in installed) bool { return in.id == im.id }
	}
	if part, ok := findBodyPart(query); ok {
		match = func(in installed) bool { return in.part == part }
	}
	i := slices.IndexFunc(body, match)
	if i < 0 {
		return installed{}, false
	}
	return body[i], true
}

func (p Profession) implanted(body []installed, now time.Time) Profession {
	for _, in := range body {
		if !in.powered(now) {
			p.unpowered = append(p.unpowered, in.id)
			continue
		}
		p = in.bonus.apply(p)
		p.implants = append(p.implants, in.id)
	}
	return p
}

func implantPerks(p Profession) []string {
	var lines []string
	if len(p.implants) > 0 {
		lines = append(lines, "🦾 Tem implantado "+implantLabels(p.implants)+" (já conta nas vantagens acima)")
	}
	if len(p.unpowered) > 0 {
		lines = append(lines, fmt.Sprintf("🔌 %s %s por falta de manutenção: manda !implante manutencao", implantLabels(p.unpowered), pluralWord(int64(len(p.unpowered)), "tá desligado", "tão desligados")))
	}
	if p.sealedBody {
		lines = append(lines, "🫀 O corpo aceita qualquer implante: nunca rejeita, e ninguém arranca os que tão instalados, nem quem te mata no !matar, nem a polícia")
	}
	return lines
}

func (op surgery) text(im implant, person types.JID) (string, []types.JID) {
	if op.rejected {
		hurtText, freed := op.injury.text(person)
		text := fmt.Sprintf("🩸 @%s deitou na maca, mas o corpo rejeitou %s e o implante foi pro lixo hospitalar", person.User, im.label())
		return text + hurtText, append([]types.JID{person}, freed...)
	}
	text := fmt.Sprintf("🦾 @%s deitou na maca e acordou com %s %s!\n✨ %s\n🔋 Já vem com %s de manutenção paga. Depois disso, manda !implante manutencao pra ele não desligar",
		person.User, im.label(), im.organ().place, im.bonus.effect, upkeepSpan(implantUpkeepGrace))
	if op.spared {
		text += "\n🫀 O corpo ia rejeitar, mas o Coração do Corpo-Seco bateu mais forte e segurou o implante no lugar"
	}
	if im.illegal {
		text += "\n🚨 É clandestino: se você for preso no !prender ou no !denunciar, a polícia arranca ele"
	}
	return text, []types.JID{person}
}

func (u unplugging) wearText() string {
	if u.broke {
		return "e ele quebrou de vez na saída"
	}
	return fmt.Sprintf("e ele voltou pra mochila com desgaste %d/%d", u.wear, implantWearMax)
}

func (u unplugging) ripText(killer, victim types.JID) string {
	return fmt.Sprintf("🦾 Antes do rabecão chegar, @%s arrancou %s de @%s, %s", killer.User, u.label(), victim.User, u.wearText())
}

func seizedImplantsText(seized []implant) string {
	if len(seized) == 0 {
		return ""
	}
	labels := make([]string, len(seized))
	for i, im := range seized {
		labels[i] = im.label()
	}
	return fmt.Sprintf("\n🩻 Na revista, a polícia ainda arrancou %s, %s", core.JoinWithAnd(labels), pluralWord(int64(len(seized)), "implante clandestino", "implantes clandestinos"))
}

func (b *bot) implante(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	args := strings.Fields(core.FoldLetters(core.MentionTagPattern.ReplaceAllString(core.MessageText(evt.Message), "")))[1:]
	if len(args) == 0 {
		return b.SendText(ctx, chat, implantMenu())
	}
	actions := map[string]func(context.Context, *events.Message, string) error{
		"instalar": b.installImplant, "colocar": b.installImplant, "botar": b.installImplant,
		"tirar": b.removeImplant, "remover": b.removeImplant,
		"manutencao": b.maintainImplants, "manter": b.maintainImplants, "pagar": b.maintainImplants,
		"ver": b.showImplants, "corpo": b.showImplants, "meus": b.showImplants,
	}
	action, ok := actions[args[0]]
	if !ok {
		return b.SendText(ctx, chat, "Use assim: !implante, !implante instalar <nome>, !implante tirar <nome>, !implante manutencao ou !implante ver")
	}
	return action(ctx, evt, strings.Join(args[1:], " "))
}

func implantMenu() string {
	lines := []string{
		"🦾 *Clínica de implantes*",
		"_A clínica só faz a cirurgia: o implante chega no !mercadonegro, um por dia, e vai pra mochila de quem comprar. Cada parte do corpo aguenta um implante só_",
	}
	for i, bp := range bodyParts {
		lines = append(lines, "", bp.emoji+" *"+bp.name+"*")
		for _, im := range implants {
			if im.part == i {
				lines = append(lines, im.menuLine())
			}
		}
	}
	lines = append(lines, "",
		fmt.Sprintf("_Pra instalar o que tá na mochila: !implante instalar <nome>. Se o corpo rejeitar, o implante vai pro lixo e você perde até %d de saúde (se já tiver machucado(a), pode morrer)_", implantRejectHarm),
		fmt.Sprintf("_A instalação vem com %s de manutenção paga. Depois, o implante desliga até você pagar a !implante manutencao, que deixa tudo pago por %s_", upkeepSpan(implantUpkeepGrace), upkeepSpan(implantUpkeepMax)),
		fmt.Sprintf("_O !implante tirar <nome> devolve o implante pra mochila, mas cada saída do corpo desgasta, e na %dª vez ele quebra_", implantWearMax),
		fmt.Sprintf("_Quem te mata no !matar tem %d%% de chance de arrancar um implante seu e levar pra mochila, e a polícia arranca os clandestinos de quem vai preso no !prender ou no !denunciar. Pra ver os seus: !implante ver_", percent(implantRipChance)))
	return strings.Join(lines, "\n")
}

func (b *bot) installImplant(ctx context.Context, evt *events.Message, query string) error {
	im, ok := findImplant(query)
	if !ok {
		return b.SendText(ctx, evt.Info.Chat.ToNonAD(), "Não conheço esse implante 🤔 Manda !implante pra ver todos")
	}
	return b.implantSurgery(ctx, evt, im)
}

func (b *bot) implantSurgery(ctx context.Context, evt *events.Message, im implant) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	op, err := b.Wallets.operate(ctx, chat, b.Account(ctx, person), im, time.Now())
	if errors.Is(err, errNoItems) {
		return b.SendText(ctx, chat, fmt.Sprintf("Você não tem %s na mochila 🎒 Implante chega no !mercadonegro, um por dia", im.label()))
	}
	if errors.Is(err, errPartTaken) {
		return b.SendText(ctx, chat, fmt.Sprintf("Você já tem %s %s, e cada parte do corpo aguenta um implante só. Pra trocar, tira ele antes com !implante tirar %s", op.occupant.label(), im.organ().place, im.organ().key))
	}
	if err != nil {
		return fmt.Errorf("instalar implante: %w", err)
	}
	text, people := op.text(im, person)
	_, err = b.SendMentions(ctx, chat, text, people)
	return err
}

func (b *bot) removeImplant(ctx context.Context, evt *events.Message, query string) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	if query == "" {
		return b.SendText(ctx, chat, "Fala o que tirar: !implante tirar cérebro, olhos, braço ou o nome do implante")
	}
	acc := b.Account(ctx, person)
	body, err := b.Wallets.body(ctx, chat, acc)
	if err != nil {
		return fmt.Errorf("ler implantes: %w", err)
	}
	target, ok := pickInstalled(body, query)
	if !ok {
		return b.SendText(ctx, chat, "Você não tem esse implante 🤔 Manda !implante ver pra ver os seus")
	}
	u, err := b.Wallets.extract(ctx, chat, acc, target.implant)
	if errors.Is(err, errNoImplant) {
		return b.SendText(ctx, chat, "Esse implante já saiu do seu corpo 🤔 Manda !implante ver pra ver os seus")
	}
	if err != nil {
		return fmt.Errorf("tirar implante: %w", err)
	}
	text := fmt.Sprintf("🔧 @%s tirou %s %s, %s. A vaga ficou livre pra outro", person.User, target.label(), target.organ().from, u.wearText())
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) maintainImplants(ctx context.Context, evt *events.Message, _ string) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	cost, err := b.Wallets.maintain(ctx, chat, b.Account(ctx, person), time.Now())
	if errors.Is(err, errNoImplant) {
		return b.SendText(ctx, chat, "Você não tem nenhum implante 🤷 Manda !implante pra ver a clínica")
	}
	if errors.Is(err, errUpkeepPaid) {
		return b.SendText(ctx, chat, fmt.Sprintf("🔋 A manutenção dos seus implantes já tá paga pelos próximos %s", upkeepSpan(implantUpkeepMax)))
	}
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, chat, fmt.Sprintf("Deixar a manutenção paga por %s sai %s e você não tem tudo isso. Confere no !saldo", upkeepSpan(implantUpkeepMax), Coins(cost)))
	}
	if err != nil {
		return fmt.Errorf("pagar manutenção dos implantes: %w", err)
	}
	text := fmt.Sprintf("🔧 @%s pagou %s de manutenção, e os implantes tão ligados e pagos pelos próximos %s", person.User, Coins(cost), upkeepSpan(implantUpkeepMax))
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) showImplants(ctx context.Context, evt *events.Message, _ string) error {
	chat := evt.Info.Chat.ToNonAD()
	person := core.FirstMentionOrSender(evt).ToNonAD()
	body, err := b.Wallets.body(ctx, chat, b.Account(ctx, person))
	if err != nil {
		return fmt.Errorf("ler implantes: %w", err)
	}
	_, err = b.SendMentions(ctx, chat, bodyReport(person, body, time.Now()), []types.JID{person})
	return err
}

func bodyReport(person types.JID, body []installed, now time.Time) string {
	lines := []string{fmt.Sprintf("🦾 *Implantes de @%s*", person.User)}
	for i, bp := range bodyParts {
		j := slices.IndexFunc(body, func(in installed) bool { return in.part == i })
		if j < 0 {
			lines = append(lines, fmt.Sprintf("%s %s: vazio", bp.emoji, bp.name))
			continue
		}
		lines = append(lines, fmt.Sprintf("%s %s: %s (%s)", bp.emoji, bp.name, body[j].label(), body[j].status(now)))
	}
	if len(body) == 0 {
		return strings.Join(append(lines, "", "_Ainda é 100% carne e osso. Manda !implante pra ver os implantes que chegam no !mercadonegro_"), "\n")
	}
	if due := upkeepDue(body, now); due > 0 {
		lines = append(lines, "", fmt.Sprintf("_Pra deixar tudo pago por %s: !implante manutencao (%s)_", upkeepSpan(implantUpkeepMax), Coins(due)))
	}
	return strings.Join(lines, "\n")
}

func loadBody(ctx context.Context, q rowsQuerier, chat types.JID, acc core.Account) (body []installed, err error) {
	rows, err := q.QueryContext(ctx, "SELECT implante, pago_ate, desgaste FROM implantes WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var (
			id string
			in installed
		)
		if err := rows.Scan(&id, &in.paidUntil, &in.wear); err != nil {
			return nil, err
		}
		im, ok := implantByID(id)
		if !ok {
			continue
		}
		in.implant = im
		body = append(body, in)
	}
	slices.SortFunc(body, func(a, b installed) int { return cmp.Compare(a.part, b.part) })
	return body, rows.Err()
}

func (s *walletStore) body(ctx context.Context, chat types.JID, acc core.Account) ([]installed, error) {
	return loadBody(ctx, s.db, chat, acc)
}

func (s *walletStore) operate(ctx context.Context, chat types.JID, acc core.Account, im implant, now time.Time) (op surgery, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		var id string
		err := tx.QueryRowContext(ctx, "SELECT implante FROM implantes WHERE chat = ? AND pessoa = ? AND parte = ?", chat.String(), acc.Key, im.organ().key).Scan(&id)
		if err == nil {
			op.occupant, _ = implantByID(id)
			return errPartTaken
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		wear, err := wearIn(ctx, tx, backpackTable, chat, acc, im.id)
		if err != nil {
			return err
		}
		if err := takeItems(ctx, tx, chat, acc, im.id, 1); err != nil {
			return err
		}
		sealed, err := holdsItem(ctx, tx, chat, acc, corpoSecoItem)
		if err != nil {
			return err
		}
		w := im.risk().roll(0)
		op.spared = sealed && w.damage > 0
		if op.rejected = w.damage > 0 && !sealed; op.rejected {
			op.injury, err = hurt(ctx, tx, chat, acc, w, now)
			return err
		}
		_, err = tx.ExecContext(ctx,
			"INSERT INTO implantes (chat, pessoa, parte, implante, instalado_em, pago_ate, desgaste) VALUES (?, ?, ?, ?, ?, ?, ?)",
			chat.String(), acc.Key, im.organ().key, im.id, now.Unix(), now.Add(implantUpkeepGrace).Unix(), wear,
		)
		return err
	})
	return op, err
}

func (s *walletStore) extract(ctx context.Context, chat types.JID, acc core.Account, im implant) (u unplugging, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		var wear int64
		err := tx.QueryRowContext(ctx,
			"DELETE FROM implantes WHERE chat = ? AND pessoa = ? AND parte = ? AND implante = ? RETURNING desgaste",
			chat.String(), acc.Key, im.organ().key, im.id,
		).Scan(&wear)
		if errors.Is(err, sql.ErrNoRows) {
			return errNoImplant
		}
		if err != nil {
			return err
		}
		u, err = shelveImplant(ctx, tx, chat, acc, im, wear)
		return err
	})
	return u, err
}

func shelveImplant(ctx context.Context, tx *ledgerTx, chat types.JID, to core.Account, im implant, wear int64) (unplugging, error) {
	u := unplugging{implant: im, wear: wear + 1}
	if u.broke = u.wear >= implantWearMax; u.broke {
		return u, nil
	}
	if err := addItems(ctx, tx, chat, to, im.id, 1); err != nil {
		return u, err
	}
	return u, keepWear(ctx, tx, backpackTable, chat, to, im.id, u.wear)
}

func (s *walletStore) maintain(ctx context.Context, chat types.JID, acc core.Account, now time.Time) (cost int64, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		body, err := loadBody(ctx, tx, chat, acc)
		if err != nil {
			return err
		}
		if len(body) == 0 {
			return errNoImplant
		}
		if cost = upkeepDue(body, now); cost == 0 {
			return errUpkeepPaid
		}
		if err := debitBalance(ctx, tx, chat, acc, cost); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx,
			"UPDATE implantes SET pago_ate = MAX(pago_ate, ?) WHERE chat = ? AND pessoa = ?",
			now.Add(implantUpkeepMax).Unix(), chat.String(), acc.Key,
		)
		return err
	})
	return cost, err
}

func ripImplant(ctx context.Context, tx *ledgerTx, chat types.JID, victim, killer core.Account) (u unplugging, err error) {
	if rand.Float64() >= implantRipChance {
		return u, nil
	}
	if sealed, err := holdsItem(ctx, tx, chat, victim, corpoSecoItem); err != nil || sealed {
		return u, err
	}
	var (
		id   string
		wear int64
	)
	err = tx.QueryRowContext(ctx,
		"DELETE FROM implantes WHERE rowid = (SELECT rowid FROM implantes WHERE chat = ? AND pessoa = ? ORDER BY RANDOM() LIMIT 1) RETURNING implante, desgaste",
		chat.String(), victim.Key,
	).Scan(&id, &wear)
	if errors.Is(err, sql.ErrNoRows) {
		return u, nil
	}
	if err != nil {
		return u, err
	}
	im, ok := implantByID(id)
	if !ok {
		return u, nil
	}
	return shelveImplant(ctx, tx, chat, killer, im, wear)
}

func seizeImplants(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account) (seized []implant, err error) {
	if sealed, err := holdsItem(ctx, tx, chat, acc, corpoSecoItem); err != nil || sealed {
		return nil, err
	}
	var ids []any
	for _, im := range implants {
		if im.illegal {
			ids = append(ids, im.id)
		}
	}
	rows, err := tx.QueryContext(ctx,
		"DELETE FROM implantes WHERE chat = ? AND pessoa = ? AND implante IN ("+core.SQLPlaceholders(len(ids))+") RETURNING implante",
		append([]any{chat.String(), acc.Key}, ids...)...,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if im, ok := implantByID(id); ok {
			seized = append(seized, im)
		}
	}
	return seized, rows.Err()
}
