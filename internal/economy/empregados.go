package economy

import (
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
	"golang.org/x/sync/errgroup"
)

var (
	errNoEstate = errors.New("sem imóvel")
	errNoRoom   = errors.New("sem vaga")
	errNoStaff  = errors.New("sem empregado")
)

type employee struct {
	id        string
	name      string
	emoji     string
	activity  string
	price     int64
	wage      int64
	every     time.Duration
	stock     int64
	luck      int
	gradeLuck int
	quarry    string
	misses    int
	trade     string
	payouts   []payout
	taxCut    int64
	care      int64
	watch     int64
	aliases   []string
}

type payout struct {
	times  int64
	weight int64
}

const (
	payrollGrace    = time.Hour
	payrollPatience = 3
)

var staffKinds = "COALESCE((SELECT GROUP_CONCAT(w.tipo) FROM empregados w WHERE w.chat = c.chat AND w.pessoa = c.pessoa AND (w.coletado_em > unixepoch() OR w.tipo NOT IN (" + payrollIDs() + "))), '')"

var staff = []employee{
	{id: "pescador-aprendiz", name: "Pescador aprendiz", emoji: "🎣", activity: "!pescar", price: 3_300, wage: 10, every: 30 * time.Minute, stock: 24, gradeLuck: -1},
	{id: "pescador-veterano", name: "Pescador veterano", emoji: "🎣", activity: "!pescar", price: 10_500, wage: 20, every: 20 * time.Minute, stock: 36, luck: 1},
	{id: "garimpeiro-aprendiz", name: "Garimpeiro aprendiz", emoji: "⛏️", activity: "!minerar", price: 5_700, wage: 17, every: 30 * time.Minute, stock: 24, gradeLuck: -1},
	{id: "garimpeiro-veterano", name: "Garimpeiro veterano", emoji: "⛏️", activity: "!minerar", price: 18_300, wage: 37, every: 20 * time.Minute, stock: 36, luck: 1},
	{id: "catador-aprendiz", name: "Catador aprendiz", emoji: "🗑️", activity: "!vasculhar", price: 1_400, wage: 4, every: 30 * time.Minute, stock: 24, gradeLuck: -1, aliases: []string{"lixeiro aprendiz"}},
	{id: "catador-veterano", name: "Catador veterano", emoji: "🗑️", activity: "!vasculhar", price: 4_200, wage: 9, every: 20 * time.Minute, stock: 36, luck: 1, aliases: []string{"lixeiro veterano"}},
	{id: "cacador-de-insetos-aprendiz", name: "Caçador de insetos aprendiz", emoji: "🦋", activity: "!catar", price: 1_500, wage: 5, every: 30 * time.Minute, stock: 24, gradeLuck: -1, aliases: []string{"inseteiro aprendiz"}},
	{id: "cacador-de-insetos-veterano", name: "Caçador de insetos veterano", emoji: "🦋", activity: "!catar", price: 5_200, wage: 10, every: 20 * time.Minute, stock: 36, luck: 1, aliases: []string{"inseteiro veterano"}},
	{id: "cacador-de-bussola", name: "Caçador de bússola", emoji: "🧭", price: 600, wage: 2, every: 30 * time.Minute, stock: 24, quarry: "bussola-do-relicario", misses: 249, aliases: []string{"bussoleiro"}},
	{id: "apostador-aprendiz", name: "Apostador aprendiz", emoji: "🎰", trade: "aposta no cassino por você", price: 2_700, wage: 10, every: 30 * time.Minute, stock: 24, payouts: []payout{{0, 60}, {2, 20}, {5, 12}, {15, 6}, {50, 2}}},
	{id: "apostador-veterano", name: "Apostador veterano", emoji: "🎰", trade: "aposta no cassino por você", price: 15_600, wage: 25, every: 20 * time.Minute, stock: 36, payouts: []payout{{0, 50}, {2, 25}, {5, 14}, {15, 8}, {50, 3}}},
	{id: "corretor-aprendiz", name: "Corretor aprendiz", emoji: "📈", trade: "mexe na bolsa por você", price: 3_300, wage: 30, every: time.Hour, stock: 12, payouts: []payout{{1, 20}, {2, 35}, {3, 30}, {5, 15}}},
	{id: "corretor-veterano", name: "Corretor veterano", emoji: "📈", trade: "mexe na bolsa por você", price: 12_000, wage: 60, every: 40 * time.Minute, stock: 18, payouts: []payout{{1, 10}, {2, 35}, {3, 35}, {5, 20}}},
	{id: "contador-aprendiz", name: "Contador aprendiz", emoji: "🧮", price: 6_000, taxCut: 20},
	{id: "contador-veterano", name: "Contador veterano", emoji: "🧮", price: 24_000, taxCut: 40},
	{id: "enfermeiro-aprendiz", name: "Enfermeiro aprendiz", emoji: "🩺", price: 2_500, care: 50, aliases: []string{"enfermeira aprendiz"}},
	{id: "enfermeiro-veterano", name: "Enfermeiro veterano", emoji: "🩺", price: 8_000, care: 100, aliases: []string{"enfermeira veterana"}},
	{id: "seguranca", name: "Segurança particular", emoji: "🕶️", price: 20_000, wage: 1_000, watch: 15, aliases: []string{"guarda-costas"}},
}

func employeeByID(id string) (employee, bool) {
	i := slices.IndexFunc(staff, func(e employee) bool { return e.id == id })
	if i < 0 {
		return employee{}, false
	}
	return staff[i], true
}

func findEmployee(query string) (employee, bool) {
	q := normalizeItemName(query)
	if q == "" {
		return employee{}, false
	}
	keys := func(e employee) []string {
		keys := []string{normalizeItemName(e.name), normalizeItemName(e.id)}
		for _, alias := range e.aliases {
			keys = append(keys, normalizeItemName(alias))
		}
		return keys
	}
	matchers := []func(key string) bool{
		func(key string) bool { return key == q },
		func(key string) bool { return strings.HasPrefix(key, q) },
		func(key string) bool { return strings.Contains(key, q) },
	}
	for _, match := range matchers {
		if i := slices.IndexFunc(staff, func(e employee) bool { return slices.ContainsFunc(keys(e), match) }); i >= 0 {
			return staff[i], true
		}
	}
	return employee{}, false
}

func (e employee) label() string {
	return e.emoji + " *" + e.name + "*"
}

func (e employee) passive() bool {
	return e.activity == "" && e.quarry == "" && len(e.payouts) == 0
}

func (e employee) payroll() bool {
	return e.passive() && e.wage > 0
}

func (e employee) patience() int64 {
	if e.payroll() {
		return payrollPatience
	}
	return e.stock / 2
}

func payrollIDs() string {
	ids := []string{"''"}
	for _, e := range staff {
		if e.payroll() {
			ids = append(ids, "'"+e.id+"'")
		}
	}
	return strings.Join(ids, ", ")
}

func payday(now time.Time) time.Time {
	return nextTaxDay(now).Add(payrollGrace)
}

func (e employee) hiredAt(now time.Time) time.Time {
	if e.payroll() {
		return payday(now)
	}
	return now
}

func (e employee) pace(decreed decrees) time.Duration {
	if len(e.payouts) > 0 {
		return e.every
	}
	return e.every * time.Duration(100+decreed.level(gatherDecree)) / 100
}

func (e employee) labor(decreed decrees, rounds int64, h *harvest) {
	if len(e.payouts) > 0 {
		for range rounds {
			h.cash(e.wage * e.bet())
		}
		return
	}
	a := e.shift(decreed)
	for range rounds {
		h.keep(a.roll())
	}
}

func (e employee) bet() int64 {
	var total int64
	for _, p := range e.payouts {
		total += p.weight
	}
	roll := rand.Int64N(total)
	for _, p := range e.payouts {
		if roll < p.weight {
			return p.times
		}
		roll -= p.weight
	}
	return 0
}

func (e employee) odds() (average, jackpot, bust int64) {
	var total int64
	for _, p := range e.payouts {
		total += p.weight
		average += p.times * p.weight
		jackpot = max(jackpot, p.times)
		if p.times == 0 {
			bust += p.weight
		}
	}
	return e.wage * average / total, e.wage * jackpot, bust * 100 / total
}

func strongest(kinds []employee, perk func(employee) int64) int64 {
	var best int64
	for _, e := range kinds {
		best = max(best, perk(e))
	}
	return best
}

func kindsOf(tipos string) []employee {
	var kinds []employee
	for id := range strings.SplitSeq(tipos, ",") {
		if e, ok := employeeByID(id); ok {
			kinds = append(kinds, e)
		}
	}
	return kinds
}

func crewKinds(crew []hire) []employee {
	kinds := make([]employee, len(crew))
	for i, h := range crew {
		kinds[i] = h.kind
	}
	return kinds
}

func taxCutOf(e employee) int64 { return e.taxCut }

func careOf(e employee) int64 { return e.care }

func watchOf(e employee) int64 { return e.watch }

func (p Profession) escorted(watch int64) Profession {
	p.guard += float64(watch) / 100
	return p
}

func (e employee) shift(decreed decrees) activity {
	if e.quarry != "" {
		return activity{loot: []drop{{"", e.misses}, {e.quarry, 1}}, cooldown: cooldown{every: e.pace(decreed)}}
	}
	a := activityNamed(e.activity)
	a.loot = slices.DeleteFunc(slices.Clone(a.loot), func(d drop) bool {
		it, _ := itemByID(d.item)
		return it.rarity >= legendary || it.unboxable() || it.gear != nil && it.gear.relicLure > 0
	})
	a.tool, a.breaks = "", 0
	a.luck, a.plainLuck, a.gradeLuck, a.rotten = e.luck, 0, e.gradeLuck, false
	a.cooldown.every = e.pace(decreed)
	return a
}

func (e employee) perks() string {
	if e.taxCut > 0 {
		return fmt.Sprintf("corta %d%% do imposto sobre fortuna e do IPTU todo dia, sem cobrar salário. Com mais de um contador, só vale o melhor", e.taxCut)
	}
	if e.care > 0 {
		return fmt.Sprintf("faz sua saúde voltar %d%% mais rápido (de zero a cheia em %s, e não %s), sem cobrar salário. Com mais de um enfermeiro, só vale o melhor", e.care, core.FormatWait(MaxHealth*regenEvery([]employee{e})), core.FormatWait(MaxHealth*healthRegenEvery))
	}
	if e.watch > 0 {
		return fmt.Sprintf("fica na sua cola: quem tenta !roubar, !furtar, !sequestrar ou !matar você tem %d%% menos chance. Cobra %s de salário todo dia às %dh, junto do imposto. Sem saldo pra isso ele trabalha fiado e a tensão sobe, e com %d salários atrasados entra em greve e não protege até receber. Com mais de um segurança, só vale o melhor", e.watch, Coins(e.wage), taxHour, e.patience())
	}
	if len(e.payouts) > 0 {
		average, jackpot, bust := e.odds()
		risk := "nunca volta com menos do que cobrou"
		if bust > 0 {
			risk = fmt.Sprintf("perde tudo em %d%% das vezes", bust)
		}
		return fmt.Sprintf("%s a cada %s: cobra %s por vez e volta com até %s, %s na média (%s). Guarda até %d rodadas e aguenta %d salários atrasados", e.trade, core.FormatWait(e.every), Coins(e.wage), Coins(jackpot), Coins(average), risk, e.stock, e.patience())
	}
	if e.quarry != "" {
		it, _ := itemByID(e.quarry)
		return fmt.Sprintf("caça a %s a cada %s, cobra %s por vez, guarda até %d tentativas e aguenta %d salários atrasados. Quase sempre volta de mão vazia: só 1 em %d tentativas acha alguma coisa", it.label(), core.FormatWait(e.every), Coins(e.wage), e.stock, e.patience(), e.misses+1)
	}
	skill := "traz o que acha com qualidade puxada pra baixo"
	if e.luck > 0 {
		skill = fmt.Sprintf("tem olho bom: escolhe o melhor de %d achados", e.luck+1)
	}
	return fmt.Sprintf("faz o %s a cada %s, cobra %s por vez, guarda até %d achados, aguenta %d salários atrasados e %s", e.activity, core.FormatWait(e.every), Coins(e.wage), e.stock, e.patience(), skill)
}

func estateRoom(level int) int {
	e, ok := estateAt(level)
	if !ok {
		return 0
	}
	return e.staff
}

type hire struct {
	id      int64
	kind    employee
	since   time.Time
	tension int64
}

func (h hire) rebelling() bool {
	return h.tension >= h.kind.patience()
}

func (h hire) ready(every time.Duration, now time.Time) int64 {
	if every <= 0 || now.Before(h.since) {
		return 0
	}
	return min(int64(now.Sub(h.since)/every), h.kind.stock)
}

type harvest struct {
	rounds  int64
	empty   int64
	credit  int64
	strikes int64
	wages   int64
	arrears int64
	owed    int64
	coins   int64
	balance int64
	found   map[string][]grade
}

func (b *bot) empregados(ctx context.Context, evt *events.Message) error {
	args := strings.Fields(core.FoldLetters(core.MessageText(evt.Message)))[1:]
	actions := map[string]core.Handler{
		"contratar": b.hireEmployee,
		"comprar":   b.hireEmployee,
		"demitir":   b.fireEmployee,
		"mandar":    b.fireEmployee,
		"coletar":   b.collectStaff,
		"recolher":  b.collectStaff,
		"pegar":     b.collectStaff,
	}
	if len(args) > 0 {
		if run, ok := actions[args[0]]; ok {
			return run(ctx, evt)
		}
	}
	return b.staffStatus(ctx, evt)
}

func (b *bot) staffStatus(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	now := time.Now()
	var (
		crew    []hire
		level   int
		decreed decrees
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { crew, err = b.Wallets.crew(gctx, chat, acc); return err })
	g.Go(func() (err error) { level, err = b.Wallets.estateLevel(gctx, chat, acc); return err })
	g.Go(func() (err error) { decreed, err = b.Wallets.decrees(gctx, chat, now); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler empregados: %w", err)
	}
	lines := []string{"👷 *Empregados*", ""}
	lines = append(lines, crewLines(crew, decreed, now)...)
	lines = append(lines, roomLine(person, len(crew), level), "", "📋 *Quem dá pra contratar*")
	for _, e := range staff {
		lines = append(lines, fmt.Sprintf("%s: %s\n    _%s_", e.label(), Coins(e.price), e.perks()))
	}
	lines = append(lines, "", "_Eles nunca acham relíquia, mítico, lendário, Relicário perdido nem Bússola do relicário, que só o caçador de bússola acha, e bem de vez em quando. O que trazem vai pra mochila (as moedas do apostador e do corretor vão pro saldo) no !empregados coletar, e o salário sai do seu saldo na hora. Sem saldo pro salário, eles trabalham fiado e a tensão sobe a cada salário atrasado. Quando ela chega no limite, eles entram em greve e as rodadas se perdem, até você pagar o atrasado. O contador, o enfermeiro e o segurança trabalham sozinhos e não têm nada pra coletar. Só o segurança cobra salário, todo dia junto do imposto_")
	text := strings.Join(lines, "\n") + decreed.note(gatherDecree)
	_, err := b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func crewLines(crew []hire, decreed decrees, now time.Time) []string {
	if len(crew) == 0 {
		return nil
	}
	type tally struct {
		count     int64
		ready     int64
		striking  int64
		rebelling int64
		tension   int64
		owed      int64
	}
	tallies := map[string]*tally{}
	for _, h := range crew {
		t, ok := tallies[h.kind.id]
		if !ok {
			t = &tally{}
			tallies[h.kind.id] = t
		}
		t.count++
		t.ready += h.ready(h.kind.pace(decreed), now)
		t.tension = max(t.tension, h.tension)
		t.owed += h.tension * h.kind.wage
		if h.rebelling() {
			t.rebelling++
		}
		if h.kind.payroll() && !h.since.After(now) {
			t.striking++
		}
	}
	var lines []string
	for _, e := range staff {
		t, ok := tallies[e.id]
		if !ok {
			continue
		}
		mood := tensionNote(e, t.tension, t.owed, t.rebelling)
		if t.striking > 0 {
			lines = append(lines, fmt.Sprintf("%dx %s: %d em greve, esperando o salário atrasado%s", t.count, e.label(), t.striking, mood))
			continue
		}
		if e.passive() {
			lines = append(lines, fmt.Sprintf("%dx %s: trabalhando%s", t.count, e.label(), mood))
			continue
		}
		one, many := "achado", "achados"
		if len(e.payouts) > 0 {
			one, many = "rodada", "rodadas"
		}
		lines = append(lines, fmt.Sprintf("%dx %s: %d %s pra coletar%s", t.count, e.label(), t.ready, pluralWord(t.ready, one, many), mood))
	}
	return append(lines, "")
}

func tensionNote(e employee, tension, owed, rebelling int64) string {
	if tension == 0 {
		return ""
	}
	if rebelling == 0 {
		return fmt.Sprintf("\n    😠 Tensão %d/%d: %s de salário atrasado", tension, e.patience(), Coins(owed))
	}
	if e.payroll() {
		return fmt.Sprintf("\n    🪧 Tensão no limite: %s de salário atrasado", Coins(owed))
	}
	return fmt.Sprintf("\n    🪧 Tensão no limite: %d em greve até receber os %s de salário atrasado", rebelling, Coins(owed))
}

func roomLine(person types.JID, used, level int) string {
	room := estateRoom(level)
	if room == 0 {
		return fmt.Sprintf("🪹 @%s não tem imóvel, e ninguém trabalha pra quem não tem nem onde morar. Compra um no !imovel", person.User)
	}
	e, _ := estateAt(level)
	return fmt.Sprintf("%s @%s mora %s e tem %d de %d %s ocupadas", e.emoji, person.User, e.place, used, room, pluralWord(int64(room), "vaga", "vagas"))
}

func (b *bot) contratar(ctx context.Context, evt *events.Message) error {
	return b.hire(ctx, evt, 1)
}

func (b *bot) coletar(ctx context.Context, evt *events.Message) error {
	args := strings.Fields(core.FoldLetters(core.MessageText(evt.Message)))[1:]
	if len(args) > 0 && slices.Contains([]string{"empregados", "empregado", "funcionarios", "funcionario"}, args[0]) {
		return b.collectStaff(ctx, evt)
	}
	return b.SendText(ctx, evt.Info.Chat.ToNonAD(), "Fala o que você quer coletar, tipo !coletar empregados")
}

func (b *bot) hireEmployee(ctx context.Context, evt *events.Message) error {
	return b.hire(ctx, evt, 2)
}

func (b *bot) hire(ctx context.Context, evt *events.Message, skip int) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	query := strings.Join(strings.Fields(core.MessageText(evt.Message))[skip:], " ")
	kind, ok := findEmployee(query)
	if !ok {
		return b.SendText(ctx, chat, "Fala quem você quer contratar, tipo !empregados contratar pescador aprendiz. Manda !empregados pra ver a lista")
	}
	room, err := b.Wallets.hireStaff(ctx, chat, acc, kind, time.Now())
	if errors.Is(err, errNoEstate) {
		return b.SendText(ctx, chat, "🪹 Ninguém trabalha pra quem não tem nem onde morar. Compra um imóvel no !imovel primeiro")
	}
	if errors.Is(err, errNoRoom) {
		return b.SendText(ctx, chat, fmt.Sprintf("🏚️ Sua casa só tem %d %s e tá tudo ocupado. Sobe de imóvel no !imovel ou demite alguém", room, pluralWord(int64(room), "vaga", "vagas")))
	}
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, chat, fmt.Sprintf("Contratar %s custa %s e você não tem tudo isso. Confere no !saldo", kind.label(), Coins(kind.price)))
	}
	if err != nil {
		return fmt.Errorf("contratar empregado: %w", err)
	}
	hint := "_Manda !empregados coletar pra pegar o que ele trouxer_"
	if kind.passive() {
		hint = "_Ele já começou a trabalhar, não precisa coletar nada_"
	}
	text := fmt.Sprintf("🤝 @%s contratou %s por %s!\n✨ Ele %s\n%s", person.User, kind.label(), Coins(kind.price), kind.perks(), hint)
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) fireEmployee(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	query := strings.Join(strings.Fields(core.MessageText(evt.Message))[2:], " ")
	kind, ok := findEmployee(query)
	if !ok {
		return b.SendText(ctx, chat, "Fala quem você quer demitir, tipo !empregados demitir pescador aprendiz")
	}
	err := b.Wallets.fireStaff(ctx, chat, acc, kind)
	if errors.Is(err, errNoStaff) {
		return b.SendText(ctx, chat, fmt.Sprintf("Você não tem nenhum %s pra demitir", kind.label()))
	}
	if err != nil {
		return fmt.Errorf("demitir empregado: %w", err)
	}
	text := fmt.Sprintf("📦 @%s demitiu um %s. Ele saiu levando o que tinha achado e ainda não foi coletado", person.User, kind.label())
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) collectStaff(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	now := time.Now()
	decreed, err := b.Wallets.decrees(ctx, chat, now)
	if err != nil {
		return fmt.Errorf("ler decretos: %w", err)
	}
	h, err := b.Wallets.collectStaff(ctx, chat, acc, decreed, now)
	if errors.Is(err, errNoStaff) {
		return b.SendText(ctx, chat, "Você não tem empregado que traga coisa pra coletar (o contador, o enfermeiro e o segurança trabalham sozinhos). Manda !empregados pra ver quem dá pra contratar")
	}
	if err != nil {
		return fmt.Errorf("coletar empregados: %w", err)
	}
	_, err = b.SendMentions(ctx, chat, h.text(person)+decreed.note(gatherDecree), []types.JID{person})
	return err
}

func (h harvest) text(person types.JID) string {
	if h.rounds == 0 && h.strikes == 0 && h.wages == 0 && h.owed == 0 {
		return fmt.Sprintf("👷 Os empregados de @%s ainda não acharam nada. Volta daqui a pouco", person.User)
	}
	lines := []string{fmt.Sprintf("👷 @%s foi ver o que os empregados acharam", person.User)}
	var items []string
	for _, it := range catalog {
		grades, ok := h.found[it.id]
		if !ok {
			continue
		}
		var n int64
		for _, g := range grades {
			n += g.quantity
		}
		items = append(items, fmt.Sprintf("%dx %s", n, it.label()))
	}
	if len(items) > 0 {
		lines = append(lines, "", strings.Join(items, "\n"))
	}
	if h.coins > 0 {
		lines = append(lines, "", fmt.Sprintf("🎲 Trouxeram %s em moedas", Coins(h.coins)))
	}
	brought := len(items) > 0 || h.coins > 0
	if !brought && h.rounds > 0 {
		lines = append(lines, "", "🫙 Voltaram todos de mão vazia")
	}
	if h.empty > 0 && brought {
		lines = append(lines, fmt.Sprintf("🫙 %d %s de mão vazia", h.empty, pluralWord(h.empty, "vez", "vezes")))
	}
	if h.wages > 0 {
		lines = append(lines, "", fmt.Sprintf("💸 Salário: %s · 💰 Saldo: %s", Coins(h.wages), Coins(h.balance)))
	}
	if h.arrears > 0 {
		lines = append(lines, fmt.Sprintf("🤝 %s disso era salário atrasado, e a tensão baixou", Coins(h.arrears)))
	}
	if h.credit > 0 {
		lines = append(lines, fmt.Sprintf("😠 Faltou saldo pro salário de %d %s: eles trabalharam fiado, mas a tensão subiu", h.credit, pluralWord(h.credit, "rodada", "rodadas")))
	}
	if h.strikes > 0 {
		lines = append(lines, fmt.Sprintf("🪧 A tensão chegou no limite e eles fizeram greve: %d %s perdidas", h.strikes, pluralWord(h.strikes, "rodada", "rodadas")))
	}
	if h.owed > 0 {
		lines = append(lines, fmt.Sprintf("📒 Salário atrasado: %s. Eles cobram isso antes de qualquer rodada nova", Coins(h.owed)))
	}
	return strings.Join(lines, "\n")
}

func (s *walletStore) crew(ctx context.Context, chat types.JID, acc core.Account) ([]hire, error) {
	return loadCrew(ctx, s.db, chat, acc)
}

func loadCrew(ctx context.Context, q core.RowsQuerier, chat types.JID, acc core.Account) (crew []hire, err error) {
	rows, err := q.QueryContext(ctx, "SELECT id, tipo, coletado_em, tensao FROM empregados WHERE chat = ? AND pessoa = ? ORDER BY id", chat.String(), acc.Key)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var (
			h     hire
			kind  string
			since int64
		)
		if err := rows.Scan(&h.id, &kind, &since, &h.tension); err != nil {
			return nil, err
		}
		var ok bool
		if h.kind, ok = employeeByID(kind); !ok {
			continue
		}
		h.since = time.Unix(since, 0)
		crew = append(crew, h)
	}
	return crew, rows.Err()
}

func (s *walletStore) estateLevel(ctx context.Context, chat types.JID, acc core.Account) (level int, err error) {
	err = s.db.QueryRowContext(ctx, "SELECT imovel FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&level)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return level, err
}

func (s *walletStore) hireStaff(ctx context.Context, chat types.JID, acc core.Account, kind employee, now time.Time) (room int, err error) {
	err = s.spend(ctx, chat, acc, kind.price, func(tx *ledgerTx) error {
		var level, used int
		err := tx.QueryRowContext(ctx,
			"SELECT imovel, (SELECT COUNT(*) FROM empregados e WHERE e.chat = c.chat AND e.pessoa = c.pessoa) FROM carteiras c WHERE c.chat = ? AND c.pessoa = ?",
			chat.String(), acc.Key,
		).Scan(&level, &used)
		if err != nil {
			return err
		}
		room = estateRoom(level)
		if room == 0 {
			return errNoEstate
		}
		if used >= room {
			return errNoRoom
		}
		_, err = tx.ExecContext(ctx, "INSERT INTO empregados (chat, pessoa, tipo, coletado_em) VALUES (?, ?, ?, ?)",
			chat.String(), acc.Key, kind.id, kind.hiredAt(now).Unix())
		return err
	})
	return room, err
}

func (s *walletStore) fireStaff(ctx context.Context, chat types.JID, acc core.Account, kind employee) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM empregados WHERE id = (
		     SELECT id FROM empregados WHERE chat = ? AND pessoa = ? AND tipo = ? ORDER BY coletado_em DESC LIMIT 1
		 )`,
		chat.String(), acc.Key, kind.id,
	)
	if err != nil {
		return err
	}
	fired, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if fired == 0 {
		return errNoStaff
	}
	return nil
}

func (s *walletStore) collectStaff(ctx context.Context, chat types.JID, acc core.Account, decreed decrees, now time.Time) (h harvest, err error) {
	h.found = map[string][]grade{}
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		crew, err := loadCrew(ctx, tx, chat, acc)
		if err != nil {
			return err
		}
		crew = slices.DeleteFunc(crew, func(h hire) bool { return h.kind.passive() })
		if len(crew) == 0 {
			return errNoStaff
		}
		if err := tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&h.balance); err != nil {
			return err
		}
		for _, hired := range crew {
			if err := h.work(ctx, tx, hired, decreed, now); err != nil {
				return err
			}
		}
		if h.wages > 0 {
			if err := debitBalance(ctx, tx, chat, acc, h.wages); err != nil {
				return err
			}
			h.balance -= h.wages
		}
		if h.coins > 0 {
			if _, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", h.coins, chat.String(), acc.Key); err != nil {
				return err
			}
			h.balance += h.coins
		}
		for id, grades := range h.found {
			if err := addGrades(ctx, tx, chat, acc, id, grades); err != nil {
				return err
			}
		}
		return nil
	})
	return h, err
}

func (h *harvest) work(ctx context.Context, tx *ledgerTx, hired hire, decreed decrees, now time.Time) error {
	wage := hired.kind.wage
	every := hired.kind.pace(decreed)
	ready := hired.ready(every, now)
	affordable := max(h.balance-h.wages, 0) / wage
	settled := min(hired.tension, affordable)
	tension := hired.tension - settled
	paid := min(ready, affordable-settled)
	credit := min(ready-paid, max(hired.kind.patience()-tension, 0))
	tension += credit
	h.rounds += paid + credit
	h.credit += credit
	h.strikes += ready - paid - credit
	h.wages += (settled + paid) * wage
	h.arrears += settled * wage
	h.owed += tension * wage
	hired.kind.labor(decreed, paid+credit, h)
	next := hired.since.Add(time.Duration(ready) * every)
	if ready == hired.kind.stock {
		next = now
	}
	_, err := tx.ExecContext(ctx, "UPDATE empregados SET coletado_em = ?, tensao = ? WHERE id = ?", next.Unix(), tension, hired.id)
	return err
}

func payStaff(ctx context.Context, tx *ledgerTx, chat types.JID, now time.Time) error {
	crew, err := payrollCrew(ctx, tx, chat)
	if err != nil {
		return err
	}
	for _, p := range crew {
		if err := p.pay(ctx, tx, chat, now); err != nil {
			return err
		}
	}
	return nil
}

func (p payrollHire) pay(ctx context.Context, tx *ledgerTx, chat types.JID, now time.Time) error {
	var balance int64
	err := tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), p.person).Scan(&balance)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	due := p.tension + 1
	paid := min(due, max(balance, 0)/p.kind.wage)
	if paid > 0 {
		if _, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo - ? WHERE chat = ? AND pessoa = ?", paid*p.kind.wage, chat.String(), p.person); err != nil {
			return err
		}
	}
	tension := min(due-paid, p.kind.patience())
	if tension >= p.kind.patience() {
		_, err = tx.ExecContext(ctx, "UPDATE empregados SET tensao = ? WHERE id = ?", tension, p.id)
		return err
	}
	_, err = tx.ExecContext(ctx, "UPDATE empregados SET coletado_em = ?, tensao = ? WHERE id = ?", payday(now).Unix(), tension, p.id)
	return err
}

type payrollHire struct {
	id      int64
	person  string
	kind    employee
	tension int64
}

func payrollCrew(ctx context.Context, tx *ledgerTx, chat types.JID) (crew []payrollHire, err error) {
	rows, err := tx.QueryContext(ctx, "SELECT id, pessoa, tipo, tensao FROM empregados WHERE chat = ? AND tipo IN ("+payrollIDs()+") ORDER BY id", chat.String())
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var (
			p    payrollHire
			kind string
		)
		if err := rows.Scan(&p.id, &p.person, &kind, &p.tension); err != nil {
			return nil, err
		}
		p.kind, _ = employeeByID(kind)
		crew = append(crew, p)
	}
	return crew, rows.Err()
}

func (h *harvest) cash(won int64) {
	if won == 0 {
		h.empty++
		return
	}
	h.coins += won
}

func (h *harvest) keep(found haul) {
	if found.found.id == "" {
		h.empty++
		return
	}
	grades := h.found[found.found.id]
	i := slices.IndexFunc(grades, func(g grade) bool { return g.quality == found.grade })
	if i < 0 {
		h.found[found.found.id] = append(grades, grade{found.grade, 1})
		return
	}
	grades[i].quantity++
}
