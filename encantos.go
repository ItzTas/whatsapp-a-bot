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

const hexBackfire = 0.25

var charmCooldown = cooldown{column: "encanto_em", every: 40 * time.Minute}

var errCharmIdle = errors.New("não tinha maldição pra tirar")

type charm struct {
	id      string
	name    string
	emoji   string
	curse   bool
	hold    time.Duration
	effect  string
	aliases []string
	apply   func(profession) profession
}

var charms = []charm{
	{id: "prosperidade", name: "Prosperidade", emoji: "💰", hold: 3 * time.Hour, aliases: []string{"dinheiro", "riqueza", "grana"},
		effect: "o !trabalhar paga 50% a mais",
		apply: func(p profession) profession {
			p.min, p.max = p.min*3/2, p.max*3/2
			return p
		}},
	{id: "protecao", name: "Proteção", emoji: "🛡️", hold: 3 * time.Hour, aliases: []string{"escudo", "corpo fechado"},
		effect: "quem tenta !roubar, !furtar, !sequestrar ou !matar a pessoa tem 25% menos chance",
		apply: func(p profession) profession {
			p.guard += 0.25
			return p
		}},
	{id: "sorte", name: "Sorte", emoji: "🍀", hold: 3 * time.Hour, aliases: []string{"trevo", "boa sorte"},
		effect: "20% de chance de ganhar em dobro no !trabalhar (não soma com a da profissão nem com a da bolsa, vale a maior) e 20% de volta quando perde no !apostar e no !cassino",
		apply: func(p profession) profession {
			p.doubleChance = max(p.doubleChance, 0.2)
			p.cashback += 20
			return p
		}},
	{id: "fartura", name: "Fartura", emoji: "🍞", hold: 12 * time.Hour, aliases: []string{"abundancia", "pao"},
		effect: "o !daily rende 50% a mais e a poupança rende 2 pontos a mais",
		apply: func(p profession) profession {
			p.dailyBonus += 50
			p.interest += 2
			return p
		}},
	{id: "libertacao", name: "Libertação", emoji: "🕊️", hold: 3 * time.Hour, aliases: []string{"liberdade", "pomba"},
		effect: "+25% de chance no !fugir da cadeia e do cativeiro, e 30% menos tempo quando vai preso",
		apply: func(p profession) profession {
			p.escapeBonus += 0.25
			p.breakoutBonus += 0.25
			p.jailFactor -= 30
			return p
		}},
	{id: "justica", name: "Justiça divina", emoji: "⚖️", hold: 6 * time.Hour, aliases: []string{"justica divina", "juizo"},
		effect: "conta como 2 votos a mais pro lado da pessoa no júri do !processar",
		apply: func(p profession) profession {
			p.trialEdge += 2
			return p
		}},
	{id: "purificacao", name: "Purificação", emoji: "🧂", aliases: []string{"descarrego", "limpeza", "limpar", "exorcismo"},
		effect: "tira na hora toda maldição da pessoa, até a praga do !zoar"},

	{id: "miseria", name: "Miséria", emoji: "💸", curse: true, hold: 2 * time.Hour, aliases: []string{"pobreza", "falencia"},
		effect: "o !trabalhar paga 40% menos",
		apply: func(p profession) profession {
			p.min, p.max = p.min*60/100, p.max*60/100
			return p
		}},
	{id: "olhogordo", name: "Olho gordo", emoji: "🧿", curse: true, hold: 2 * time.Hour, aliases: []string{"mau olhado", "quebranto", "inveja"},
		effect: "quem tenta !roubar, !furtar, !sequestrar ou !matar a pessoa tem 25% mais chance",
		apply: func(p profession) profession {
			p.guard -= 0.25
			return p
		}},
	{id: "azar", name: "Azar", emoji: "🐈‍⬛", curse: true, hold: 2 * time.Hour, aliases: []string{"urucubaca", "zica", "gato preto"},
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
	{id: "jejum", name: "Jejum", emoji: "🍽️", curse: true, hold: 12 * time.Hour, aliases: []string{"fome", "seca"},
		effect: "o !daily rende metade e a poupança rende 2 pontos a menos",
		apply: func(p profession) profession {
			p.dailyBonus -= 50
			p.interest -= 2
			return p
		}},
	{id: "encosto", name: "Encosto", emoji: "👻", curse: true, hold: 2 * time.Hour, aliases: []string{"assombracao", "espirito"},
		effect: "fica 50% mais tempo na cadeia e tem 25% menos chance no !fugir da cadeia e do cativeiro",
		apply: func(p profession) profession {
			p.jailFactor += 50
			p.escapeBonus -= 0.25
			p.breakoutBonus -= 0.25
			return p
		}},
	{id: "excomunhao", name: "Excomunhão", emoji: "📛", curse: true, hold: 6 * time.Hour, aliases: []string{"excomungar", "heresia"},
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
	cd     cooldown
	now    time.Time
}

type spell struct {
	wait     time.Duration
	hit      bool
	backfire bool
}

func (c charm) label() string {
	return c.emoji + " *" + c.name + "*"
}

func (c charm) purges() bool {
	return c.apply == nil
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

func (p profession) canCast(c charm) bool {
	if p.divine {
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
	return clampChance(p.faith.hexChance)
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
		p = c.apply(p)
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
	var lines []string
	if len(p.faith.blessings) > 0 {
		lines = append(lines, "🙏 Pode !abencoar os outros com "+charmNames(p.faith.blessings))
	}
	if len(p.faith.curses) > 0 {
		lines = append(lines, fmt.Sprintf("🕯️ Pode !amaldicoar os outros com %s, com %d%% de chance de pegar (se falhar, a praga pode voltar)", charmNames(p.faith.curses), percent(p.hexOdds())))
	}
	return lines
}

func castersOf(c charm) string {
	var names []string
	for _, p := range professions {
		if p.canCast(c) {
			names = append(names, p.emoji+" "+p.name)
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
	if targetAcc.key == casterAcc.key {
		return b.sendText(ctx, chat, selfCharmText(curse))
	}
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
	order := charmOrder{chat: chat, caster: casterAcc, target: targetAcc, charm: c, odds: odds, cd: job.pace(charmCooldown), now: time.Now()}
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

func selfCharmText(curse bool) string {
	if curse {
		return "Se amaldiçoar? Isso aí já é caso de terapia 🫂"
	}
	return "🙅 Santo de casa não faz milagre: ninguém consegue se abençoar sozinho. Pede pra outra pessoa"
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
	header := fmt.Sprintf("Qual %s? Como %s, você tem:", kind, job.title)
	if query != "" {
		header = fmt.Sprintf("Não conheço essa %s 🤔 Como %s, você tem:", kind, job.title)
	}
	if c, found := findCharm(query, curse); found && query != "" {
		header = fmt.Sprintf("🚫 %s não é com você, quem tem é %s. Como %s, você tem:", c.label(), castersOf(c), job.title)
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
	lines := []string{title, "_Só quem mexe com fé ou feitiçaria lança, e cada profissão tem as suas_"}
	for _, c := range charms {
		if c.curse != curse {
			continue
		}
		lines = append(lines, "", c.summary(), "Quem tem: "+castersOf(c))
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
	text := fmt.Sprintf("_Como %s, você tem %s. Manda %s @pessoa <nome>, uma vez a cada %s_", job.title, charmNames(ids), command, formatWait(job.pace(charmCooldown).every))
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
	if s.hit {
		return head + fmt.Sprintf(" e jogou %s por %s 😈\n%s\n_Pra se livrar: pede uma Purificação pra %s (!abencoar @pessoa purificacao)_", c.label(), formatWait(c.hold), c.effect, purifiers())
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
		sp.hit = rand.Float64() < o.odds
		sp.backfire = !sp.hit && rand.Float64() < hexBackfire
		if !sp.hit && !sp.backfire {
			return nil
		}
		bearer := o.target
		if sp.backfire {
			bearer = o.caster
		}
		return grantCharm(ctx, tx, o.chat, bearer, o.caster, o.charm, o.now)
	})
	return sp, err
}

func grantCharm(ctx context.Context, tx *sql.Tx, chat types.JID, bearer, author account, c charm, now time.Time) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO encantos (chat, alvo, tipo, autor, ate) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT (chat, alvo, tipo) DO UPDATE SET autor = excluded.autor, ate = MAX(encantos.ate, excluded.ate)`,
		chat.String(), bearer.key, c.id, author.key, now.Add(c.hold).Unix(),
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
