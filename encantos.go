package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	hexBackfire     = 0.25
	itemHexChance   = 0.5
	mirrorHexChance = 0.6
	mirrorItem      = "espelho-da-iara"
)

var charmCooldown = cooldown{column: "encanto_em", every: 40 * time.Minute}

const hasteCommands = "!pescar, !minerar, !vasculhar, !catar, !roubar, !furtar, !quadrilha, !sequestrar, !matar, !prender, !denunciar, !processar, !reviver, !libertar, !abencoar e !amaldicoar"

var errCharmIdle = errors.New("não tinha maldição pra tirar")

type charm struct {
	id      string
	name    string
	emoji   string
	curse   bool
	hold    time.Duration
	effect  string
	aliases []string
	toll    int64
	boon    int64
	boonCap int64
	apply   func(profession) profession
}

var charms = []charm{
	{id: "prosperidade", name: "Prosperidade", emoji: "💰", hold: 6 * time.Hour, aliases: []string{"dinheiro", "riqueza", "grana"},
		effect: "o !trabalhar paga 50% a mais",
		apply: func(p profession) profession {
			p.min, p.max = p.min*3/2, p.max*3/2
			return p
		}},
	{id: "protecao", name: "Proteção", emoji: "🛡️", hold: 6 * time.Hour, aliases: []string{"escudo", "corpo fechado"},
		effect: "quem tenta !roubar, !furtar, !sequestrar ou !matar a pessoa tem 25% menos chance",
		apply: func(p profession) profession {
			p.guard += 0.25
			return p
		}},
	{id: "sorte", name: "Sorte", emoji: "🍀", hold: 6 * time.Hour, aliases: []string{"trevo", "boa sorte"},
		effect: "20% de chance de ganhar em dobro no !trabalhar (não soma com a da profissão nem com a da bolsa, vale a maior) e 20% de volta quando perde no !apostar e no !cassino",
		apply: func(p profession) profession {
			p.doubleChance = max(p.doubleChance, 0.2)
			p.cashback += 20
			return p
		}},
	{id: "fartura", name: "Fartura", emoji: "🍞", hold: 24 * time.Hour, aliases: []string{"abundancia", "pao"},
		effect: "o !daily rende 50% a mais e a poupança rende 2 pontos a mais",
		apply: func(p profession) profession {
			p.dailyBonus += 50
			p.interest += 2
			return p
		}},
	{id: "libertacao", name: "Libertação", emoji: "🕊️", hold: 6 * time.Hour, aliases: []string{"liberdade", "pomba"},
		effect: "+25% de chance no !fugir da cadeia e do cativeiro, e 30% menos tempo quando vai preso",
		apply: func(p profession) profession {
			p.escapeBonus += 0.25
			p.breakoutBonus += 0.25
			p.jailFactor -= 30
			return p
		}},
	{id: "justica", name: "Justiça divina", emoji: "⚖️", hold: 12 * time.Hour, aliases: []string{"justica divina", "juizo"},
		effect: "conta como 2 votos a mais pro lado da pessoa no júri do !processar",
		apply: func(p profession) profession {
			p.trialEdge += 2
			return p
		}},
	{id: "pressa", name: "Pressa", emoji: "⏩", hold: 3 * time.Hour, aliases: []string{"tempo", "rapidez", "agilidade", "ampulheta"},
		effect: "espera 30% menos no !trabalhar e no " + hasteCommands,
		apply: func(p profession) profession {
			p.every = p.every * 70 / 100
			p.haste -= 30
			return p
		}},
	{id: "multiplicacao", name: "Multiplicação", emoji: "🐟", hold: 12 * time.Hour, boon: boonPercent, boonCap: boonCap, aliases: []string{"multiplicar", "milagre", "peixes", "paes e peixes"},
		effect: fmt.Sprintf("na próxima vez que o saldo da pessoa subir (trabalho, aposta, venda, pix, o que for), entra mais %d%% do que entrou, em moedas novas, até %s", boonPercent, coins(boonCap))},
	{id: "purificacao", name: "Purificação", emoji: "🧂", aliases: []string{"descarrego", "limpeza", "limpar", "exorcismo"},
		effect: "tira na hora toda maldição da pessoa, até a praga do !zoar"},

	{id: "miseria", name: "Miséria", emoji: "💸", curse: true, hold: 4 * time.Hour, aliases: []string{"pobreza", "falencia"},
		effect: "o !trabalhar paga 40% menos",
		apply: func(p profession) profession {
			p.min, p.max = p.min*60/100, p.max*60/100
			return p
		}},
	{id: "olhogordo", name: "Olho gordo", emoji: "🧿", curse: true, hold: 4 * time.Hour, aliases: []string{"mau olhado", "quebranto", "inveja"},
		effect: "quem tenta !roubar, !furtar, !sequestrar ou !matar a pessoa tem 25% mais chance",
		apply: func(p profession) profession {
			p.guard -= 0.25
			return p
		}},
	{id: "azar", name: "Azar", emoji: "🐈‍⬛", curse: true, hold: 4 * time.Hour, aliases: []string{"urucubaca", "zica", "gato preto"},
		effect: "-20% de chance no !roubar, no !furtar, no !sequestrar e no !assalto, 20% das vitórias no !apostar e no !cassino viram derrota e o !trabalhar nunca paga em dobro",
		apply: func(p profession) profession {
			p.robBonus -= 0.2
			p.theftBonus -= 0.2
			p.kidnapBonus -= 0.2
			p.heistBonus -= 0.2
			p.betJinx += 0.2
			p.noDouble = true
			return p
		}},
	{id: "jejum", name: "Jejum", emoji: "🍽️", curse: true, hold: 24 * time.Hour, aliases: []string{"fome", "seca"},
		effect: "o !daily rende metade e a poupança rende 2 pontos a menos",
		apply: func(p profession) profession {
			p.dailyBonus -= 50
			p.interest -= 2
			return p
		}},
	{id: "encosto", name: "Encosto", emoji: "👻", curse: true, hold: 4 * time.Hour, aliases: []string{"assombracao", "espirito"},
		effect: "fica 50% mais tempo na cadeia e tem 25% menos chance no !fugir da cadeia e do cativeiro",
		apply: func(p profession) profession {
			p.jailFactor += 50
			p.escapeBonus -= 0.25
			p.breakoutBonus -= 0.25
			return p
		}},
	{id: "lerdeza", name: "Lerdeza", emoji: "🐌", curse: true, hold: 2 * time.Hour, aliases: []string{"tempo", "lentidao", "atraso", "preguica", "relogio parado"},
		effect: "espera 50% mais no !trabalhar e no " + hasteCommands,
		apply: func(p profession) profession {
			p.every = p.every * 3 / 2
			p.haste += 50
			return p
		}},
	{id: "dizimo", name: "Dízimo", emoji: "🧾", curse: true, hold: 12 * time.Hour, toll: tithePercent, aliases: []string{"pedagio", "comissao", "parte", "porcentagem"},
		effect: fmt.Sprintf("na próxima vez que o saldo da pessoa subir (trabalho, aposta, venda, pix, o que for), %d%% do que entrou vai pra quem rogou a praga. Se a praga voltar, quem recebe é quem ia levar ela", tithePercent)},
	{id: "excomunhao", name: "Excomunhão", emoji: "📛", curse: true, hold: 12 * time.Hour, aliases: []string{"excomungar", "heresia"},
		effect: "o júri do !processar ganha 2 votos contra a pessoa",
		apply: func(p profession) profession {
			p.trialEdge -= 2
			return p
		}},
}

type faith struct {
	blessings []string
	curses    []string
	hexChance float64
	blessRite string
	curseRite string
}

type heldCharm struct {
	id   string
	left time.Duration
}

type charmOrder struct {
	chat   types.JID
	caster account
	target account
	charm  charm
	odds   float64
	pierce bool
	cd     cooldown
	now    time.Time
}

type spell struct {
	wait      time.Duration
	hit       bool
	backfire  bool
	reflected bool
}

func (c charm) label() string {
	return c.emoji + " *" + c.name + "*"
}

func (c charm) purges() bool {
	return c.id == "purificacao"
}

func (c charm) matchKeys() []string {
	keys := []string{normalizeItemName(c.id), normalizeItemName(c.name)}
	for _, alias := range c.aliases {
		keys = append(keys, normalizeItemName(alias))
	}
	return keys
}

func (c charm) summary() string {
	if c.purges() {
		return fmt.Sprintf("%s: %s", c.label(), c.effect)
	}
	return fmt.Sprintf("%s (%s): %s", c.label(), formatWait(c.hold), c.effect)
}

func purifiers() string {
	c, _ := charmByID("purificacao")
	return strings.Replace(castersOf(c), " e ", " ou ", 1)
}

func charmByID(id string) (charm, bool) {
	i := slices.IndexFunc(charms, func(c charm) bool { return c.id == id })
	if i < 0 {
		return charm{}, false
	}
	return charms[i], true
}

func findCharm(query string, curse bool) (charm, bool) {
	q := normalizeItemName(query)
	pool := slices.DeleteFunc(slices.Clone(charms), func(c charm) bool { return c.curse != curse })
	if i := slices.IndexFunc(pool, func(c charm) bool { return slices.Contains(c.matchKeys(), q) }); i >= 0 {
		return pool[i], true
	}
	prefixed := slices.DeleteFunc(pool, func(c charm) bool {
		return !slices.ContainsFunc(c.matchKeys(), func(key string) bool { return strings.HasPrefix(key, q) })
	})
	if len(prefixed) != 1 {
		return charm{}, false
	}
	return prefixed[0], true
}

func charmNames(ids []string) string {
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		c, _ := charmByID(id)
		names = append(names, c.emoji+" "+c.name)
	}
	return joinWithAnd(names)
}

func mirrorLabel() string {
	it, _ := itemByID(mirrorItem)
	return it.label()
}

func (p profession) canCast(c charm) bool {
	if p.divine || p.mirror {
		return true
	}
	if c.curse {
		return slices.Contains(p.faith.curses, c.id)
	}
	return slices.Contains(p.faith.blessings, c.id)
}

func (p profession) castable(curse bool) []charm {
	return slices.DeleteFunc(slices.Clone(charms), func(c charm) bool { return c.curse != curse || !p.canCast(c) })
}

func (p profession) hexOdds() float64 {
	if p.divine {
		return 1
	}
	if p.mirror {
		return clampChance(max(p.faith.hexChance, mirrorHexChance))
	}
	return clampChance(p.faith.hexChance)
}

func (p profession) castingAs() string {
	if p.mirror && !p.divine {
		return "Com o " + mirrorLabel()
	}
	return "Como " + p.title
}

func (p profession) rite(curse bool) string {
	rites := map[bool]struct{ own, plain string }{
		false: {p.faith.blessRite, "rezou por"},
		true:  {p.faith.curseRite, "rogou uma praga em"},
	}
	r := rites[curse]
	if r.own == "" {
		return r.plain
	}
	return r.own
}

func (p profession) charmed(held []heldCharm) profession {
	for _, h := range held {
		c, ok := charmByID(h.id)
		if !ok || c.purges() {
			continue
		}
		if c.apply != nil {
			p = c.apply(p)
		}
		p.charms = append(p.charms, h.id)
	}
	return p
}

func charmPerks(p profession) []string {
	var blessings, curses []string
	for _, id := range p.charms {
		c, _ := charmByID(id)
		if c.curse {
			curses = append(curses, id)
			continue
		}
		blessings = append(blessings, id)
	}
	var lines []string
	if len(blessings) > 0 {
		lines = append(lines, "😇 Tá abençoado com "+charmNames(blessings)+" (já conta nas vantagens acima)")
	}
	if len(curses) > 0 {
		lines = append(lines, "😈 Tá amaldiçoado com "+charmNames(curses)+" (já conta nas vantagens acima)")
	}
	return lines
}

func faithPerks(p profession) []string {
	if p.mirror && !p.divine {
		return []string{fmt.Sprintf("🧜‍♀️ Pode !abencoar e !amaldicoar com qualquer bênção ou maldição, com %d%% de chance de a praga pegar, e toda maldição que jogam em você volta pra quem jogou", percent(p.hexOdds()))}
	}
	var lines []string
	if len(p.faith.blessings) > 0 {
		lines = append(lines, "🙏 Pode !abencoar os outros com "+charmNames(p.faith.blessings))
	}
	if len(p.faith.curses) > 0 {
		lines = append(lines, fmt.Sprintf("🕯️ Pode !amaldicoar os outros com %s, com %d%% de chance de pegar (se falhar, a praga pode voltar)", charmNames(p.faith.curses), percent(p.hexOdds())))
	}
	return lines
}

func itemsCarrying(c charm) string {
	var names []string
	for _, it := range catalog {
		if it.bless == c.id || it.hex == c.id {
			names = append(names, it.emoji+" "+it.name)
		}
	}
	return joinWithAnd(names)
}

func cureText() string {
	c, _ := charmByID("purificacao")
	return fmt.Sprintf("_Pra se livrar: pede uma Purificação pra %s (!abencoar @pessoa purificacao) ou usa %s_", purifiers(), itemsCarrying(c))
}

func castersOf(c charm) string {
	var names []string
	for _, p := range professions {
		if p.canCast(c) {
			names = append(names, p.emoji+" "+p.name)
		}
	}
	for _, t := range titles {
		if t.bonus.apply(profession{}).canCast(c) {
			names = append(names, t.emoji+" quem usa o título "+t.name)
		}
	}
	return joinWithAnd(names)
}

func mysticProfessions() string {
	var names []string
	for _, p := range professions {
		if len(p.faith.blessings)+len(p.faith.curses) > 0 {
			names = append(names, p.emoji+" "+p.name)
		}
	}
	return joinWithAnd(names)
}

func (b *bot) abencoar(ctx context.Context, evt *events.Message) error {
	return b.castCharm(ctx, evt, false)
}

func (b *bot) amaldicoar(ctx context.Context, evt *events.Message) error {
	return b.castCharm(ctx, evt, true)
}

func (b *bot) castCharm(ctx context.Context, evt *events.Message, curse bool) error {
	chat := evt.Info.Chat.ToNonAD()
	caster := evt.Info.Sender.ToNonAD()
	casterAcc := b.account(ctx, caster)
	job, err := b.professionOf(ctx, chat, casterAcc)
	if err != nil {
		return err
	}
	mentions := mentionedJIDs(evt.Message)
	if len(mentions) == 0 {
		return b.charmMenu(ctx, chat, casterAcc, job, curse)
	}
	target := mentions[0].ToNonAD()
	targetAcc := b.account(ctx, target)
	available := job.castable(curse)
	if len(available) == 0 {
		return b.sendText(ctx, chat, noFaithText(curse))
	}
	query := strings.Join(strings.Fields(mentionTagPattern.ReplaceAllString(messageText(evt.Message), ""))[1:], " ")
	c, ok := pickCharm(query, available, curse)
	if !ok {
		return b.sendText(ctx, chat, charmChoiceText(query, job, available, curse))
	}
	odds := 1.0
	if curse {
		odds = job.hexOdds()
	}
	order := charmOrder{chat: chat, caster: casterAcc, target: targetAcc, charm: c, odds: odds, pierce: job.divine, cd: job.pace(charmCooldown), now: time.Now()}
	s, err := b.wallets.cast(ctx, order)
	if errors.Is(err, errCharmIdle) {
		_, err := b.sendMentions(ctx, chat, fmt.Sprintf("😌 @%s tá com a alma limpinha, não tem maldição nenhuma pra tirar", target.User), []types.JID{target})
		return err
	}
	if err != nil {
		return fmt.Errorf("lançar %s: %w", c.id, err)
	}
	_, err = b.sendMentions(ctx, chat, spellText(s, job, c, caster, target), []types.JID{caster, target})
	return err
}

func pickCharm(query string, available []charm, curse bool) (charm, bool) {
	if query == "" && len(available) == 1 {
		return available[0], true
	}
	c, ok := findCharm(query, curse)
	return c, ok && slices.ContainsFunc(available, func(a charm) bool { return a.id == c.id })
}

func noFaithText(curse bool) string {
	if curse {
		return "🚫 Só quem mexe com feitiçaria consegue !amaldicoar: " + mysticProfessions() + ". Manda !amaldicoar pra ver as maldições"
	}
	return "🚫 Só quem mexe com fé consegue !abencoar: " + mysticProfessions() + ". Manda !abencoar pra ver as bênçãos"
}

func charmChoiceText(query string, job profession, available []charm, curse bool) string {
	command, kind := "!abencoar", "bênção"
	if curse {
		command, kind = "!amaldicoar", "maldição"
	}
	header := fmt.Sprintf("Qual %s? %s, você tem:", kind, job.castingAs())
	if query != "" {
		header = fmt.Sprintf("Não conheço essa %s 🤔 %s, você tem:", kind, job.castingAs())
	}
	if c, found := findCharm(query, curse); found && query != "" {
		header = fmt.Sprintf("🚫 %s não é com você, quem tem é %s. %s, você tem:", c.label(), castersOf(c), job.castingAs())
	}
	lines := []string{header}
	for _, c := range available {
		lines = append(lines, c.summary())
	}
	return strings.Join(append(lines, "", fmt.Sprintf("_Manda %s @pessoa <nome>_", command)), "\n")
}

func (b *bot) charmMenu(ctx context.Context, chat types.JID, acc account, job profession, curse bool) error {
	held, err := b.wallets.charmsOn(ctx, chat, acc, time.Now())
	if err != nil {
		return fmt.Errorf("ler encantos: %w", err)
	}
	command, title := "!abencoar", "🙏 *Bênçãos*"
	if curse {
		command, title = "!amaldicoar", "🕯️ *Maldições*"
	}
	lines := []string{title, "_Só quem mexe com fé ou feitiçaria lança, e cada profissão tem as suas. Quem tem o " + mirrorLabel() + " lança todas_"}
	for _, c := range charms {
		if c.curse != curse {
			continue
		}
		lines = append(lines, "", c.summary(), "Quem tem: "+castersOf(c))
		if items := itemsCarrying(c); items != "" {
			lines = append(lines, "Vem no item: "+items)
		}
	}
	lines = append(lines, "", ownCharmsText(job, curse, command))
	if len(held) > 0 {
		lines = append(lines, "", "✨ *Em você agora*")
		for _, h := range held {
			c, _ := charmByID(h.id)
			lines = append(lines, fmt.Sprintf("%s (falta %s)", c.label(), formatWait(h.left)))
		}
	}
	return b.sendText(ctx, chat, strings.Join(lines, "\n"))
}

func ownCharmsText(job profession, curse bool, command string) string {
	available := job.castable(curse)
	if len(available) == 0 {
		return fmt.Sprintf("_Como %s, você não lança nenhuma. Profissões que lançam: %s_", job.title, mysticProfessions())
	}
	ids := make([]string, len(available))
	for i, c := range available {
		ids[i] = c.id
	}
	text := fmt.Sprintf("_%s, você tem %s. Manda %s @pessoa <nome>, uma vez a cada %s_", job.castingAs(), charmNames(ids), command, formatWait(job.pace(charmCooldown).every))
	if job.divine {
		text = fmt.Sprintf("_Como deus, você tem todas, sem espera. Manda %s @pessoa <nome>_", command)
	}
	if curse && !job.divine {
		text += fmt.Sprintf("\n_A praga pega em %d%% das vezes, e quando falha tem %d%% de chance de voltar pra você_", percent(job.hexOdds()), percent(hexBackfire))
	}
	return text
}

func spellText(s spell, job profession, c charm, caster, target types.JID) string {
	if s.wait > 0 {
		return fmt.Sprintf("🕯️ %s @%s ainda tá recarregando a fé do último ritual. Espera %s pra lançar outra", job.emoji, caster.User, formatWait(s.wait))
	}
	head := fmt.Sprintf("%s @%s %s @%s", job.badge(), caster.User, job.rite(c.curse), target.User)
	if c.purges() {
		return head + fmt.Sprintf(" e fez a %s: toda maldição saiu do corpo, até a praga do !zoar ✨", c.label())
	}
	if !c.curse {
		return head + fmt.Sprintf(" e deu %s por %s 😇\n%s", c.label(), formatWait(c.hold), c.effect)
	}
	if s.reflected {
		return head + ", mas " + reflectedText(c, caster)
	}
	if s.hit {
		return head + fmt.Sprintf(" e jogou %s por %s 😈\n%s\n%s", c.label(), formatWait(c.hold), c.effect, cureText())
	}
	if s.backfire {
		return head + fmt.Sprintf(", mas a praga voltou! 🔁 Quem ficou com %s por %s foi @%s\n%s", c.label(), formatWait(c.hold), caster.User, c.effect)
	}
	return head + fmt.Sprintf(", mas a praga não pegou 😮‍💨 Tinha %d%% de chance", percent(job.hexOdds()))
}

func (s *walletStore) cast(ctx context.Context, o charmOrder) (sp spell, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		if err := openWallet(ctx, tx, o.chat, o.target); err != nil {
			return err
		}
		if sp.wait, err = stampCooldown(ctx, tx, o.chat, o.caster, o.cd, o.now); err != nil || sp.wait > 0 {
			return err
		}
		if o.charm.purges() {
			return purgeCurses(ctx, tx, o.chat, o.target, o.now)
		}
		sp, err = landCharm(ctx, tx, o)
		return err
	})
	return sp, err
}

func landCharm(ctx context.Context, tx *sql.Tx, o charmOrder) (spell, error) {
	sp := spell{hit: rand.Float64() < o.odds}
	sp.backfire = !sp.hit && rand.Float64() < hexBackfire
	if !sp.hit && !sp.backfire {
		return sp, nil
	}
	reflected, err := o.bouncesOffMirror(ctx, tx, sp)
	if err != nil {
		return sp, err
	}
	sp.reflected = reflected
	bearer, author := o.target, o.caster
	if sp.backfire || sp.reflected {
		bearer, author = o.caster, o.target
	}
	return sp, grantCharm(ctx, tx, o.chat, bearer, author, o.charm, o.now)
}

func (o charmOrder) bouncesOffMirror(ctx context.Context, tx *sql.Tx, sp spell) (bool, error) {
	if !sp.hit || !o.charm.curse || o.pierce || o.caster.key == o.target.key {
		return false, nil
	}
	return holdsItem(ctx, tx, o.chat, o.target, mirrorItem)
}

func reflectedText(c charm, caster types.JID) string {
	return fmt.Sprintf("a praga bateu no %s e voltou! 🔁 Quem ficou com %s por %s foi @%s\n%s\n%s", mirrorLabel(), c.label(), formatWait(c.hold), caster.User, c.effect, cureText())
}

func (b *bot) hurlCurse(ctx context.Context, evt *events.Message, it item) error {
	chat := evt.Info.Chat.ToNonAD()
	thrower := evt.Info.Sender.ToNonAD()
	c, _ := charmByID(it.hex)
	mentions := mentionedJIDs(evt.Message)
	if len(mentions) == 0 {
		return b.sendText(ctx, chat, fmt.Sprintf("%s é pra jogar em alguém: !usar %s @pessoa\nQuem pegar fica com %s", it.label(), strings.ToLower(it.name), c.summary()))
	}
	target := mentions[0].ToNonAD()
	throwerAcc, targetAcc := b.account(ctx, thrower), b.account(ctx, target)
	sp, err := b.wallets.hurl(ctx, chat, throwerAcc, targetAcc, it, c, time.Now())
	if errors.Is(err, errNoItems) {
		return b.sendText(ctx, chat, fmt.Sprintf("Não tem mais %s na sua mochila 🎒", it.label()))
	}
	if err != nil {
		return fmt.Errorf("jogar %s: %w", it.id, err)
	}
	_, err = b.sendMentions(ctx, chat, hurlText(sp, it, c, thrower, target), []types.JID{thrower, target})
	return err
}

func hurlText(sp spell, it item, c charm, thrower, target types.JID) string {
	head := fmt.Sprintf(it.use, thrower.User, target.User)
	if sp.reflected {
		return head + "\n\nMas " + reflectedText(c, thrower)
	}
	if sp.hit {
		return head + fmt.Sprintf("\n\n😈 Pegou! @%s ficou com %s por %s\n%s\n%s", target.User, c.label(), formatWait(c.hold), c.effect, cureText())
	}
	if sp.backfire {
		return head + fmt.Sprintf("\n\n🔁 Mas a praga voltou! Quem ficou com %s por %s foi @%s\n%s", c.label(), formatWait(c.hold), thrower.User, c.effect)
	}
	return head + fmt.Sprintf("\n\n😮‍💨 Mas a praga não pegou. Tinha %d%% de chance", percent(itemHexChance))
}

func (s *walletStore) hurl(ctx context.Context, chat types.JID, thrower, target account, it item, c charm, now time.Time) (sp spell, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		if err := takeItems(ctx, tx, chat, thrower, it.id, 1); err != nil {
			return err
		}
		if err := openWallet(ctx, tx, chat, target); err != nil {
			return err
		}
		sp, err = landCharm(ctx, tx, charmOrder{chat: chat, caster: thrower, target: target, charm: c, odds: itemHexChance, now: now})
		return err
	})
	return sp, err
}

func grantCharm(ctx context.Context, tx *sql.Tx, chat types.JID, bearer, author account, c charm, now time.Time) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO encantos (chat, alvo, tipo, autor, ate, parte, bonus, teto) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (chat, alvo, tipo) DO UPDATE SET autor = excluded.autor, ate = MAX(encantos.ate, excluded.ate),
		   parte = excluded.parte, bonus = excluded.bonus, teto = excluded.teto`,
		chat.String(), bearer.key, c.id, author.key, now.Add(c.hold).Unix(), c.toll, c.boon, c.boonCap,
	)
	return err
}

func purgeCurses(ctx context.Context, tx *sql.Tx, chat types.JID, acc account, now time.Time) error {
	var curses []any
	for _, c := range charms {
		if c.curse {
			curses = append(curses, c.id)
		}
	}
	statements := []struct {
		query string
		args  []any
	}{
		{"DELETE FROM encantos WHERE chat = ? AND alvo = ? AND ate > ? AND tipo IN (?" + strings.Repeat(", ?", len(curses)-1) + ")", curses},
		{"DELETE FROM zoeiras WHERE chat = ? AND alvo = ? AND ate > ? AND tipo = ?", []any{curseKind}},
	}
	var removed int64
	for _, st := range statements {
		res, err := tx.ExecContext(ctx, st.query, append([]any{chat.String(), acc.key, now.Unix()}, st.args...)...)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		removed += n
	}
	if removed == 0 {
		return errCharmIdle
	}
	return nil
}

func (s *walletStore) charmsOn(ctx context.Context, chat types.JID, acc account, now time.Time) (held []heldCharm, err error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT tipo, ate FROM encantos WHERE chat = ? AND alvo = ? AND ate > ? ORDER BY ate",
		chat.String(), acc.key, now.Unix(),
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var (
			id    string
			until int64
		)
		if err := rows.Scan(&id, &until); err != nil {
			return nil, err
		}
		held = append(held, heldCharm{id: id, left: time.Unix(until, 0).Sub(now)})
	}
	return held, rows.Err()
}
