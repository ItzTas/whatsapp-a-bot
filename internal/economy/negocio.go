package economy

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

var (
	errNoBusiness    = errors.New("sem negócio")
	errBusinessOwned = errors.New("negócio já aberto")
	errBusinessTop   = errors.New("negócio no nível máximo")
)

const (
	bizTop        = 5
	bizShelfHours = 24
	bizResale     = 50
	bizNameMax    = 40
	normalPricing = "normal"
)

const noVentureText = "🪹 Você não tem nenhum negócio. Manda !negocio lista pra ver o que dá pra abrir"

type business struct {
	id         string
	name       string
	emoji      string
	unit       string
	unitPlural string
	price      int64
	cost       int64
	rate       int64
	aliases    []string
}

var businesses = []business{
	{id: "barraca-de-limonada", name: "Barraca de limonada", emoji: "🍋", unit: "copo", unitPlural: "copos", price: 2_000, cost: 5, rate: 4, aliases: []string{"limonada"}},
	{id: "carrinho-de-pipoca", name: "Carrinho de pipoca", emoji: "🍿", unit: "saquinho", unitPlural: "saquinhos", price: 6_000, cost: 10, rate: 5, aliases: []string{"pipoca"}},
	{id: "banca-de-pastel", name: "Banca de pastel", emoji: "🥟", unit: "pastel", unitPlural: "pastéis", price: 15_000, cost: 20, rate: 6, aliases: []string{"pastelaria"}},
	{id: "lava-jato", name: "Lava-jato", emoji: "🚿", unit: "lavagem", unitPlural: "lavagens", price: 40_000, cost: 60, rate: 5, aliases: []string{"lava rapido"}},
	{id: "boteco", name: "Boteco", emoji: "🍺", unit: "cerveja", unitPlural: "cervejas", price: 100_000, cost: 15, rate: 44, aliases: []string{"bar", "buteco"}},
	{id: "padaria", name: "Padaria", emoji: "🥖", unit: "pão", unitPlural: "pães", price: 250_000, cost: 10, rate: 145},
	{id: "farmacia", name: "Farmácia", emoji: "💊", unit: "remédio", unitPlural: "remédios", price: 600_000, cost: 30, rate: 100, aliases: []string{"drogaria"}},
	{id: "posto-de-gasolina", name: "Posto de gasolina", emoji: "⛽", unit: "litro", unitPlural: "litros", price: 1_500_000, cost: 5, rate: 1_200, aliases: []string{"posto"}},
	{id: "supermercado", name: "Supermercado", emoji: "🛒", unit: "compra", unitPlural: "compras", price: 4_000_000, cost: 25, rate: 500, aliases: []string{"mercado"}},
	{id: "concessionaria", name: "Concessionária", emoji: "🚗", unit: "carro", unitPlural: "carros", price: 10_000_000, cost: 25_000, rate: 1},
}

type pricing struct {
	id      string
	emoji   string
	pace    string
	speed   int64
	markup  int64
	aliases []string
}

var pricings = []pricing{
	{id: "barato", emoji: "🏷️", pace: "vende o dobro", speed: 200, markup: 160, aliases: []string{"barata", "baixo", "promocao"}},
	{id: normalPricing, emoji: "💲", pace: "vende no ritmo de sempre", speed: 100, markup: 200, aliases: []string{"medio", "padrao"}},
	{id: "caro", emoji: "💎", pace: "vende a metade", speed: 50, markup: 260, aliases: []string{"cara", "alto", "salgado"}},
}

func businessByID(id string) (business, bool) {
	i := slices.IndexFunc(businesses, func(biz business) bool { return biz.id == id })
	if i < 0 {
		return business{}, false
	}
	return businesses[i], true
}

func findBusiness(query string) (business, bool) {
	return firstMatch(businesses, query, business.keys)
}

func firstMatch[T any](list []T, query string, keys func(T) []string) (T, bool) {
	var none T
	q := normalizeItemName(query)
	if q == "" {
		return none, false
	}
	matchers := []func(key string) bool{
		func(key string) bool { return key == q },
		func(key string) bool { return strings.HasPrefix(key, q) },
		func(key string) bool { return strings.Contains(key, q) },
	}
	for _, match := range matchers {
		if i := slices.IndexFunc(list, func(x T) bool { return slices.ContainsFunc(keys(x), match) }); i >= 0 {
			return list[i], true
		}
	}
	return none, false
}

func findPricing(word string) (pricing, bool) {
	w := core.FoldLetters(word)
	i := slices.IndexFunc(pricings, func(p pricing) bool { return p.id == w || slices.Contains(p.aliases, w) })
	if i < 0 {
		return pricing{}, false
	}
	return pricings[i], true
}

func pricingOf(id string) pricing {
	p, ok := findPricing(id)
	if !ok {
		p, _ = findPricing(normalPricing)
	}
	return p
}

func (p pricing) price(biz business) int64 {
	return biz.cost * p.markup / 100
}

func (p pricing) perks() string {
	edge := p.speed*(p.markup-100)/100 - 100
	profit := "o lucro por hora de sempre"
	if edge > 0 {
		profit = fmt.Sprintf("%d%% mais lucro por hora", edge)
	}
	if edge < 0 {
		profit = fmt.Sprintf("%d%% menos lucro por hora", -edge)
	}
	return fmt.Sprintf("%s, com %s, e o estoque cheio dura %s", p.pace, profit, core.FormatWait(time.Duration(bizShelfHours*100/p.speed)*time.Hour))
}

func (biz business) keys() []string {
	keys := []string{normalizeItemName(biz.name), normalizeItemName(biz.id)}
	for _, alias := range biz.aliases {
		keys = append(keys, normalizeItemName(alias))
	}
	return keys
}

func (biz business) label() string {
	return biz.emoji + " *" + biz.name + "*"
}

func (biz business) count(n int64) string {
	return core.Thousands(n) + " " + pluralWord(n, biz.unit, biz.unitPlural)
}

func (biz business) boost(level int) int64 {
	return 100 + 50*int64(level-1)
}

func (biz business) shelf(level int) int64 {
	return biz.rate * biz.boost(level) * bizShelfHours / 100
}

func (biz business) till(level int) int64 {
	return biz.shelf(level) * pricingOf(normalPricing).price(biz)
}

func (biz business) every(level int, p pricing) time.Duration {
	return time.Hour * 10_000 / time.Duration(biz.rate*p.speed*biz.boost(level))
}

func (biz business) hourlyProfit(level int, p pricing) int64 {
	return biz.rate * p.speed * biz.boost(level) * (p.price(biz) - biz.cost) / 10_000
}

func (biz business) upgradeCost(level int) int64 {
	return biz.price * int64(level) / 2
}

func (biz business) resale(level int) int64 {
	invested := biz.price
	for n := 1; n < level; n++ {
		invested += biz.upgradeCost(n)
	}
	return invested * bizResale / 100
}

func (biz business) pace(level int, p pricing) string {
	if perHour := biz.rate * p.speed * biz.boost(level) / 10_000; perHour >= 1 {
		return biz.count(perHour) + " por hora"
	}
	return fmt.Sprintf("1 %s a cada %s", biz.unit, core.FormatWait(biz.every(level, p)))
}

func (biz business) perks() string {
	p := pricingOf(normalPricing)
	profit := biz.hourlyProfit(1, p)
	days := (biz.price + profit*12) / (profit * 24)
	return fmt.Sprintf("Vende %s e lucra %s por hora no preço normal, com estoque de %s e caixa de %s. Se paga em uns %d dias vendendo sem parar",
		biz.pace(1, p), Coins(profit), biz.count(biz.shelf(1)), Coins(biz.till(1)), days)
}

type venture struct {
	kind    business
	title   string
	level   int
	stock   int64
	pricing pricing
	since   time.Time
}

func (v venture) keys() []string {
	return append(v.kind.keys(), normalizeItemName(v.title))
}

func (v venture) label() string {
	if v.title == "" {
		return v.kind.label()
	}
	return fmt.Sprintf("%s *%s* (%s)", v.kind.emoji, v.title, v.kind.name)
}

func (v venture) unitPrice() int64 {
	return v.pricing.price(v.kind)
}

func (v venture) every() time.Duration {
	return v.kind.every(v.level, v.pricing)
}

type takings struct {
	sold  int64
	gross int64
	until time.Time
	idle  time.Duration
	dry   bool
	full  bool
}

func (t takings) profit(v venture) int64 {
	return t.sold * (v.unitPrice() - v.kind.cost)
}

func (v venture) sell(now time.Time) takings {
	t := takings{until: v.since}
	if !now.After(v.since) {
		return t
	}
	every := v.every()
	due := int64(now.Sub(v.since) / every)
	t.sold = min(due, v.stock, v.kind.till(v.level)/v.unitPrice())
	t.gross = t.sold * v.unitPrice()
	t.until = v.since.Add(time.Duration(t.sold) * every)
	if t.sold == due {
		return t
	}
	t.idle, t.dry, t.full = now.Sub(t.until), t.sold == v.stock, t.sold < v.stock
	t.until = now
	return t
}

func (v venture) runway(now time.Time, t takings) (time.Duration, bool) {
	left := v.stock - t.sold
	room := (v.kind.till(v.level) - t.gross) / v.unitPrice()
	return t.until.Add(time.Duration(min(left, room)) * v.every()).Sub(now), left <= room
}

func (v venture) summary(now time.Time) string {
	t := v.sell(now)
	return strings.Join([]string{
		fmt.Sprintf("%s · nível %d · preço %s %s", v.label(), v.level, v.pricing.emoji, v.pricing.id),
		fmt.Sprintf("    💵 Caixa: %s de %s", Coins(t.gross), Coins(v.kind.till(v.level))),
		fmt.Sprintf("    📦 Estoque: %s de %s", core.Thousands(v.stock-t.sold), v.kind.count(v.kind.shelf(v.level))),
		"    " + v.pulse(now, t),
	}, "\n")
}

func (v venture) pulse(now time.Time, t takings) string {
	if t.dry || v.stock == t.sold {
		return fmt.Sprintf("🚫 Sem estoque, parou de vender%s. Manda !negocio repor", stalledFor(t.idle))
	}
	if t.full || v.kind.till(v.level)-t.gross < v.unitPrice() {
		return fmt.Sprintf("🔒 Caixa cheio, parou de vender%s. Manda !negocio coletar", stalledFor(t.idle))
	}
	left, dry := v.runway(now, t)
	stop := "o caixa enche"
	if dry {
		stop = "o estoque acaba"
	}
	return fmt.Sprintf("✅ Vendendo %s, com lucro de %s por hora. Em %s %s", v.kind.pace(v.level, v.pricing), Coins(v.kind.hourlyProfit(v.level, v.pricing)), core.FormatWait(left), stop)
}

func stalledFor(idle time.Duration) string {
	if idle < time.Minute {
		return ""
	}
	return " há " + core.FormatWait(idle)
}

func findVenture(owned []venture, query string) (venture, bool) {
	if strings.TrimSpace(query) == "" && len(owned) == 1 {
		return owned[0], true
	}
	return firstMatch(owned, query, venture.keys)
}

func everyVenture(query string) bool {
	return slices.Contains([]string{"", "tudo", "todos"}, core.FoldLetters(strings.TrimSpace(query)))
}

func splitVentureName(owned []venture, words []string) (venture, string, bool) {
	for i := len(words); i > 0; i-- {
		if v, ok := firstMatch(owned, strings.Join(words[:i], " "), venture.keys); ok {
			return v, cleanTitle(words[i:]), true
		}
	}
	return venture{}, "", false
}

func cleanTitle(words []string) string {
	title := strings.Map(func(r rune) rune {
		if strings.ContainsRune("*_~`", r) {
			return -1
		}
		return r
	}, strings.Join(words, " "))
	runes := []rune(strings.TrimSpace(title))
	return strings.TrimSpace(string(runes[:min(len(runes), bizNameMax)]))
}

func bizQuery(evt *events.Message) string {
	return strings.Join(strings.Fields(core.MessageText(evt.Message))[2:], " ")
}

func (b *bot) negocio(ctx context.Context, evt *events.Message) error {
	args := strings.Fields(core.FoldLetters(core.MessageText(evt.Message)))[1:]
	actions := map[string]core.Handler{
		"lista": b.businessCatalog, "catalogo": b.businessCatalog, "todos": b.businessCatalog,
		"comprar": b.openBusiness, "abrir": b.openBusiness,
		"coletar": b.collectVentures, "recolher": b.collectVentures, "caixa": b.collectVentures,
		"repor": b.restockVentures, "estoque": b.restockVentures, "abastecer": b.restockVentures,
		"preco": b.repriceVenture, "precos": b.repriceVenture,
		"melhorar": b.upgradeVenture, "reformar": b.upgradeVenture, "ampliar": b.upgradeVenture,
		"nome": b.nameVenture, "renomear": b.nameVenture,
		"vender": b.closeVenture, "fechar": b.closeVenture,
	}
	if len(args) > 0 {
		if run, ok := actions[args[0]]; ok {
			return run(ctx, evt)
		}
	}
	return b.ventureStatus(ctx, evt)
}

func (b *bot) ventureStatus(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	sender := evt.Info.Sender.ToNonAD()
	person := core.FirstMentionOrSender(evt).ToNonAD()
	owned, err := b.Wallets.ventures(ctx, chat, b.Account(ctx, person))
	if err != nil {
		return fmt.Errorf("ler negócios: %w", err)
	}
	if len(owned) == 0 && person == sender {
		return b.SendText(ctx, chat, businessMenu(nil)+"\n\n🪹 Você ainda não tem nenhum negócio. Manda !negocio comprar <nome> pra abrir um")
	}
	_, err = b.SendMentions(ctx, chat, venturesText(person, owned, time.Now()), []types.JID{person})
	return err
}

func venturesText(person types.JID, owned []venture, now time.Time) string {
	if len(owned) == 0 {
		return fmt.Sprintf("🪹 @%s não tem nenhum negócio", person.User)
	}
	lines := []string{fmt.Sprintf("🏪 *Negócios de @%s*", person.User)}
	for _, v := range owned {
		lines = append(lines, "", v.summary(now))
	}
	lines = append(lines, "", "_Manda !negocio coletar pra passar o caixa pro saldo, !negocio repor pra encher o estoque e !negocio preco pra mudar o preço. Com o estoque vazio ou o caixa cheio, o negócio para de vender_")
	return strings.Join(lines, "\n")
}

func (b *bot) businessCatalog(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	owned, err := b.Wallets.ventures(ctx, chat, b.Account(ctx, evt.Info.Sender.ToNonAD()))
	if err != nil {
		return fmt.Errorf("ler negócios: %w", err)
	}
	return b.SendText(ctx, chat, businessMenu(owned))
}

func businessMenu(owned []venture) string {
	lines := []string{"🏪 *Negócios*", ""}
	for _, biz := range businesses {
		mark := "🔒"
		if slices.ContainsFunc(owned, func(v venture) bool { return v.kind.id == biz.id }) {
			mark = "✅"
		}
		lines = append(lines, fmt.Sprintf("%s %s: %s\n    _%s_", mark, biz.label(), Coins(biz.price), biz.perks()))
	}
	lines = append(lines, "", "💲 *Preços* (!negocio preco <negócio> <preço>)")
	for _, p := range pricings {
		lines = append(lines, fmt.Sprintf("%s *%s*: %s", p.emoji, p.id, p.perks()))
	}
	normal := pricingOf(normalPricing)
	lines = append(lines, "", fmt.Sprintf("_Cada negócio abre com o estoque cheio e vende sozinho, e o dinheiro junta no caixa até você mandar !negocio coletar. A mercadoria do !negocio repor custa %d%% do preço normal de venda e sai do seu saldo. O !negocio melhorar sobe o nível até o %d, e cada nível vende mais e aumenta o estoque e o caixa_", 10_000/normal.markup, bizTop))
	return strings.Join(lines, "\n")
}

func (b *bot) openBusiness(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	kind, ok := findBusiness(bizQuery(evt))
	if !ok {
		return b.SendText(ctx, chat, "Fala qual negócio você quer abrir, tipo !negocio comprar pastel. Manda !negocio lista pra ver todos")
	}
	balance, err := b.Wallets.openVenture(ctx, chat, b.Account(ctx, person), kind, time.Now())
	if errors.Is(err, errBusinessOwned) {
		return b.SendText(ctx, chat, fmt.Sprintf("Você já tem %s, e é um de cada por pessoa. Pra vender mais, reforma com !negocio melhorar", kind.label()))
	}
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, chat, fmt.Sprintf("Abrir %s custa %s e você não tem tudo isso. Confere no !saldo", kind.label(), Coins(kind.price)))
	}
	if err != nil {
		return fmt.Errorf("abrir negócio: %w", err)
	}
	text := fmt.Sprintf("🎉 @%s abriu %s por %s! Já veio com %s no estoque e começou a vender\n✨ %s\n💰 Saldo: %s\n_Manda !negocio pra ver o caixa e o estoque, e !negocio nome pra pendurar uma placa_",
		person.User, kind.label(), Coins(kind.price), kind.count(kind.shelf(1)), kind.perks(), Coins(balance))
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) collectVentures(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	vis, err := b.Wallets.cashOut(ctx, chat, b.Account(ctx, person), time.Now())
	if errors.Is(err, errNoBusiness) {
		return b.SendText(ctx, chat, noVentureText)
	}
	if err != nil {
		return fmt.Errorf("coletar negócios: %w", err)
	}
	_, err = b.SendMentions(ctx, chat, vis.cashText(person), []types.JID{person})
	return err
}

func (b *bot) restockVentures(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	only := ""
	if query := bizQuery(evt); !everyVenture(query) {
		owned, err := b.Wallets.ventures(ctx, chat, acc)
		if err != nil {
			return fmt.Errorf("ler negócios: %w", err)
		}
		v, ok := findVenture(owned, query)
		if !ok {
			return b.missingVenture(ctx, chat, owned, "!negocio repor pastel")
		}
		only = v.kind.id
	}
	vis, err := b.Wallets.restock(ctx, chat, acc, only, time.Now())
	if errors.Is(err, errNoBusiness) {
		return b.SendText(ctx, chat, noVentureText)
	}
	if err != nil {
		return fmt.Errorf("repor estoque: %w", err)
	}
	_, err = b.SendMentions(ctx, chat, vis.restockText(person), []types.JID{person})
	return err
}

func (b *bot) repriceVenture(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	words := strings.Fields(core.MessageText(evt.Message))[2:]
	help := []string{"Fala o negócio e o preço, tipo !negocio preco pastel caro", ""}
	for _, p := range pricings {
		help = append(help, fmt.Sprintf("%s *%s*: %s", p.emoji, p.id, p.perks()))
	}
	if len(words) == 0 {
		return b.SendText(ctx, chat, strings.Join(help, "\n"))
	}
	p, ok := findPricing(words[len(words)-1])
	if !ok {
		return b.SendText(ctx, chat, strings.Join(help, "\n"))
	}
	owned, err := b.Wallets.ventures(ctx, chat, acc)
	if err != nil {
		return fmt.Errorf("ler negócios: %w", err)
	}
	v, ok := findVenture(owned, strings.Join(words[:len(words)-1], " "))
	if !ok {
		return b.missingVenture(ctx, chat, owned, "!negocio preco pastel caro")
	}
	vis, err := b.Wallets.reprice(ctx, chat, acc, v.kind.id, p, time.Now())
	if errors.Is(err, errNoBusiness) {
		return b.missingVenture(ctx, chat, nil, "")
	}
	if err != nil {
		return fmt.Errorf("mudar preço do negócio: %w", err)
	}
	text := fmt.Sprintf("%s @%s botou %s no preço *%s*: %s%s", p.emoji, person.User, v.label(), p.id, p.perks(), vis.tillNote())
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) upgradeVenture(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	owned, err := b.Wallets.ventures(ctx, chat, acc)
	if err != nil {
		return fmt.Errorf("ler negócios: %w", err)
	}
	v, ok := findVenture(owned, bizQuery(evt))
	if !ok {
		return b.missingVenture(ctx, chat, owned, "!negocio melhorar pastel")
	}
	vis, err := b.Wallets.upgradeVenture(ctx, chat, acc, v.kind.id, time.Now())
	if errors.Is(err, errBusinessTop) {
		return b.SendText(ctx, chat, fmt.Sprintf("%s já tá no nível %d, o máximo. Mais que isso só abrindo outro negócio", v.label(), bizTop))
	}
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, chat, fmt.Sprintf("A reforma de %s pro nível %d custa %s e você não tem tudo isso, nem juntando o caixa dele. Confere no !saldo", v.label(), v.level+1, Coins(v.kind.upgradeCost(v.level))))
	}
	if errors.Is(err, errNoBusiness) {
		return b.missingVenture(ctx, chat, nil, "")
	}
	if err != nil {
		return fmt.Errorf("melhorar negócio: %w", err)
	}
	was := vis.stops[0].venture
	level := was.level + 1
	text := fmt.Sprintf("🔨 @%s reformou %s pro nível %d por %s! Agora vende %s e lucra %s por hora no preço %s, com estoque de %s e caixa de %s\n_O estoque ficou maior, então manda !negocio repor pra encher_%s",
		person.User, was.label(), level, Coins(was.kind.upgradeCost(was.level)), was.kind.pace(level, was.pricing), Coins(was.kind.hourlyProfit(level, was.pricing)), was.pricing.id,
		was.kind.count(was.kind.shelf(level)), Coins(was.kind.till(level)), vis.tillNote())
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) closeVenture(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	owned, err := b.Wallets.ventures(ctx, chat, acc)
	if err != nil {
		return fmt.Errorf("ler negócios: %w", err)
	}
	query := bizQuery(evt)
	if strings.TrimSpace(query) == "" {
		return b.missingVenture(ctx, chat, owned, "!negocio vender pastel")
	}
	v, ok := findVenture(owned, query)
	if !ok {
		return b.missingVenture(ctx, chat, owned, "!negocio vender pastel")
	}
	vis, err := b.Wallets.closeVenture(ctx, chat, acc, v.kind.id, time.Now())
	if errors.Is(err, errNoBusiness) {
		return b.missingVenture(ctx, chat, nil, "")
	}
	if err != nil {
		return fmt.Errorf("fechar negócio: %w", err)
	}
	was := vis.stops[0].venture
	text := fmt.Sprintf("🏚️ @%s fechou %s e vendeu o ponto por %s, %d%% do que gastou nele. O estoque que sobrou foi junto%s",
		person.User, was.label(), Coins(was.kind.resale(was.level)), bizResale, vis.tillNote())
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) nameVenture(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	owned, err := b.Wallets.ventures(ctx, chat, acc)
	if err != nil {
		return fmt.Errorf("ler negócios: %w", err)
	}
	if len(owned) == 0 {
		return b.SendText(ctx, chat, noVentureText)
	}
	v, title, ok := splitVentureName(owned, strings.Fields(core.MessageText(evt.Message))[2:])
	if !ok {
		return b.SendText(ctx, chat, "Fala o negócio e o nome novo, tipo !negocio nome pastel Pastelaria do Zé. Sem nome nenhum, a placa sai")
	}
	err = b.Wallets.renameVenture(ctx, chat, acc, v.kind.id, title)
	if errors.Is(err, errNoBusiness) {
		return b.missingVenture(ctx, chat, nil, "")
	}
	if err != nil {
		return fmt.Errorf("dar nome pro negócio: %w", err)
	}
	text := fmt.Sprintf("🧽 @%s tirou a placa de %s", person.User, v.label())
	if title != "" {
		text = fmt.Sprintf("🪧 @%s pendurou a placa: %s agora se chama *%s*", person.User, v.kind.label(), title)
	}
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) missingVenture(ctx context.Context, chat types.JID, owned []venture, example string) error {
	if len(owned) == 0 {
		return b.SendText(ctx, chat, noVentureText)
	}
	names := make([]string, len(owned))
	for i, v := range owned {
		names[i] = v.label()
	}
	return b.SendText(ctx, chat, fmt.Sprintf("Fala qual dos seus negócios, tipo %s. Você tem: %s", example, strings.Join(names, ", ")))
}

type tended struct {
	venture venture
	takings takings
	bought  int64
}

type visit struct {
	stops   []tended
	balance int64
}

func (vis visit) gross() int64 {
	var total int64
	for _, st := range vis.stops {
		total += st.takings.gross
	}
	return total
}

func (vis visit) spent() int64 {
	var total int64
	for _, st := range vis.stops {
		total += st.bought * st.venture.kind.cost
	}
	return total
}

func (vis visit) tillNote() string {
	if vis.gross() == 0 {
		return "\n💰 Saldo: " + Coins(vis.balance)
	}
	return fmt.Sprintf("\n💵 O caixa tinha %s e foi pro saldo antes · 💰 Saldo: %s", Coins(vis.gross()), Coins(vis.balance))
}

func (vis visit) cashText(person types.JID) string {
	dry := slices.ContainsFunc(vis.stops, func(st tended) bool { return st.venture.stock == st.takings.sold })
	if vis.gross() == 0 && !dry {
		return fmt.Sprintf("💵 Os negócios de @%s ainda não venderam nada desde a última passada no caixa. Volta daqui a pouco", person.User)
	}
	lines := []string{fmt.Sprintf("💵 @%s passou no caixa dos negócios", person.User), ""}
	for _, st := range vis.stops {
		lines = append(lines, st.takingsLine())
	}
	lines = append(lines, "", fmt.Sprintf("💰 Entrou %s no saldo · Saldo: %s", Coins(vis.gross()), Coins(vis.balance)))
	if dry {
		lines = append(lines, "_Tem negócio sem estoque, e ele não vende nada até você mandar !negocio repor_")
	}
	return strings.Join(lines, "\n")
}

func (st tended) takingsLine() string {
	t, v := st.takings, st.venture
	line := fmt.Sprintf("%s: não vendeu nada ainda", v.label())
	if t.sold > 0 {
		line = fmt.Sprintf("%s: vendeu %s, %s (%s de lucro)", v.label(), v.kind.count(t.sold), Coins(t.gross), Coins(t.profit(v)))
	}
	if t.dry && t.idle >= time.Minute {
		line += fmt.Sprintf("\n    🚫 Ficou %s parado sem estoque", core.FormatWait(t.idle))
	}
	if t.full && t.idle >= time.Minute {
		line += fmt.Sprintf("\n    🔒 Ficou %s parado com o caixa cheio", core.FormatWait(t.idle))
	}
	return line
}

func (vis visit) restockText(person types.JID) string {
	if vis.spent() == 0 && vis.gross() == 0 && !slices.ContainsFunc(vis.stops, tended.short) {
		return fmt.Sprintf("📦 O estoque de @%s já tá cheio, não tem nada pra repor", person.User)
	}
	lines := []string{fmt.Sprintf("📦 @%s foi repor o estoque", person.User), ""}
	for _, st := range vis.stops {
		v := st.venture
		left, shelf := st.left(), v.kind.shelf(v.level)
		line := fmt.Sprintf("%s: +%s por %s (estoque %s de %s)", v.label(), v.kind.count(st.bought), Coins(st.bought*v.kind.cost), core.Thousands(left), core.Thousands(shelf))
		if st.bought == 0 {
			line = fmt.Sprintf("%s: faltou saldo pra repor (estoque %s de %s)", v.label(), core.Thousands(left), core.Thousands(shelf))
		}
		if st.bought == 0 && left == shelf {
			line = fmt.Sprintf("%s: o estoque já tava cheio", v.label())
		}
		if st.takings.gross > 0 {
			line += fmt.Sprintf("\n    💵 %s do caixa foram pro saldo", Coins(st.takings.gross))
		}
		lines = append(lines, line)
	}
	lines = append(lines, "", fmt.Sprintf("🧾 Mercadoria: %s · 💵 Caixa: %s · 💰 Saldo: %s", Coins(vis.spent()), Coins(vis.gross()), Coins(vis.balance)))
	if slices.ContainsFunc(vis.stops, tended.short) {
		lines = append(lines, "⚠️ Faltou saldo pra encher tudo, então comprou o que deu")
	}
	return strings.Join(lines, "\n")
}

func (st tended) left() int64 {
	return st.venture.stock - st.takings.sold + st.bought
}

func (st tended) short() bool {
	return st.left() < st.venture.kind.shelf(st.venture.level)
}

func (s *walletStore) ventures(ctx context.Context, chat types.JID, acc core.Account) ([]venture, error) {
	return loadVentures(ctx, s.db, chat, acc)
}

func loadVentures(ctx context.Context, q core.RowsQuerier, chat types.JID, acc core.Account) (ventures []venture, err error) {
	rows, err := q.QueryContext(ctx, "SELECT tipo, nome, nivel, estoque, preco, vendido_em FROM negocios WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	owned := map[string]venture{}
	for rows.Next() {
		var (
			v           venture
			kind, price string
			since       int64
		)
		if err := rows.Scan(&kind, &v.title, &v.level, &v.stock, &price, &since); err != nil {
			return nil, err
		}
		var ok bool
		if v.kind, ok = businessByID(kind); !ok {
			continue
		}
		v.pricing, v.since = pricingOf(price), time.Unix(since, 0)
		owned[kind] = v
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, biz := range businesses {
		if v, ok := owned[biz.id]; ok {
			ventures = append(ventures, v)
		}
	}
	return ventures, nil
}

func (s *walletStore) openVenture(ctx context.Context, chat types.JID, acc core.Account, kind business, now time.Time) (balance int64, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx,
			"INSERT INTO negocios (chat, pessoa, tipo, estoque, vendido_em) VALUES (?, ?, ?, ?, ?) ON CONFLICT DO NOTHING",
			chat.String(), acc.Key, kind.id, kind.shelf(1), now.Unix(),
		)
		if err != nil {
			return err
		}
		opened, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if opened == 0 {
			return errBusinessOwned
		}
		if err := debitBalance(ctx, tx, chat, acc, kind.price); err != nil {
			return err
		}
		return tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&balance)
	})
	return balance, err
}

func (vis *visit) cash(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, owned []venture, now time.Time) error {
	for _, v := range owned {
		t := v.sell(now)
		_, err := tx.ExecContext(ctx, "UPDATE negocios SET estoque = estoque - ?, vendido_em = ? WHERE chat = ? AND pessoa = ? AND tipo = ?",
			t.sold, t.until.Unix(), chat.String(), acc.Key, v.kind.id)
		if err != nil {
			return err
		}
		vis.stops = append(vis.stops, tended{venture: v, takings: t})
	}
	if err := openWallet(ctx, tx, chat, acc); err != nil {
		return err
	}
	return vis.settle(ctx, tx, chat, acc, vis.gross())
}

func (vis *visit) settle(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, delta int64) error {
	if delta < 0 {
		if err := debitBalance(ctx, tx, chat, acc, -delta); err != nil {
			return err
		}
	}
	return tx.QueryRowContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ? RETURNING saldo",
		max(delta, 0), chat.String(), acc.Key).Scan(&vis.balance)
}

func (s *walletStore) cashOut(ctx context.Context, chat types.JID, acc core.Account, now time.Time) (vis visit, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		owned, err := loadVentures(ctx, tx, chat, acc)
		if err != nil {
			return err
		}
		if len(owned) == 0 {
			return errNoBusiness
		}
		return vis.cash(ctx, tx, chat, acc, owned, now)
	})
	return vis, err
}

func (s *walletStore) restock(ctx context.Context, chat types.JID, acc core.Account, only string, now time.Time) (vis visit, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		owned, err := loadVentures(ctx, tx, chat, acc)
		if err != nil {
			return err
		}
		if only != "" {
			owned = slices.DeleteFunc(owned, func(v venture) bool { return v.kind.id != only })
		}
		if len(owned) == 0 {
			return errNoBusiness
		}
		if err := vis.cash(ctx, tx, chat, acc, owned, now); err != nil {
			return err
		}
		budget := vis.balance
		for i := range vis.stops {
			st := &vis.stops[i]
			v := st.venture
			st.bought = min(v.kind.shelf(v.level)-(v.stock-st.takings.sold), max(budget, 0)/v.kind.cost)
			if st.bought == 0 {
				continue
			}
			budget -= st.bought * v.kind.cost
			_, err := tx.ExecContext(ctx, "UPDATE negocios SET estoque = estoque + ? WHERE chat = ? AND pessoa = ? AND tipo = ?",
				st.bought, chat.String(), acc.Key, v.kind.id)
			if err != nil {
				return err
			}
		}
		return vis.settle(ctx, tx, chat, acc, -vis.spent())
	})
	return vis, err
}

func (s *walletStore) tend(ctx context.Context, chat types.JID, acc core.Account, id string, now time.Time, then func(tx *ledgerTx, v venture) (int64, error)) (vis visit, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		owned, err := loadVentures(ctx, tx, chat, acc)
		if err != nil {
			return err
		}
		i := slices.IndexFunc(owned, func(v venture) bool { return v.kind.id == id })
		if i < 0 {
			return errNoBusiness
		}
		if err := vis.cash(ctx, tx, chat, acc, owned[i:i+1], now); err != nil {
			return err
		}
		delta, err := then(tx, owned[i])
		if err != nil {
			return err
		}
		return vis.settle(ctx, tx, chat, acc, delta)
	})
	return vis, err
}

func (s *walletStore) reprice(ctx context.Context, chat types.JID, acc core.Account, id string, p pricing, now time.Time) (visit, error) {
	return s.tend(ctx, chat, acc, id, now, func(tx *ledgerTx, v venture) (int64, error) {
		_, err := tx.ExecContext(ctx, "UPDATE negocios SET preco = ? WHERE chat = ? AND pessoa = ? AND tipo = ?", p.id, chat.String(), acc.Key, v.kind.id)
		return 0, err
	})
}

func (s *walletStore) upgradeVenture(ctx context.Context, chat types.JID, acc core.Account, id string, now time.Time) (visit, error) {
	return s.tend(ctx, chat, acc, id, now, func(tx *ledgerTx, v venture) (int64, error) {
		if v.level >= bizTop {
			return 0, errBusinessTop
		}
		_, err := tx.ExecContext(ctx, "UPDATE negocios SET nivel = nivel + 1 WHERE chat = ? AND pessoa = ? AND tipo = ?", chat.String(), acc.Key, v.kind.id)
		return -v.kind.upgradeCost(v.level), err
	})
}

func (s *walletStore) closeVenture(ctx context.Context, chat types.JID, acc core.Account, id string, now time.Time) (visit, error) {
	return s.tend(ctx, chat, acc, id, now, func(tx *ledgerTx, v venture) (int64, error) {
		_, err := tx.ExecContext(ctx, "DELETE FROM negocios WHERE chat = ? AND pessoa = ? AND tipo = ?", chat.String(), acc.Key, v.kind.id)
		return v.kind.resale(v.level), err
	})
}

func (s *walletStore) renameVenture(ctx context.Context, chat types.JID, acc core.Account, id, title string) error {
	res, err := s.db.ExecContext(ctx, "UPDATE negocios SET nome = ? WHERE chat = ? AND pessoa = ? AND tipo = ?", title, chat.String(), acc.Key, id)
	if err != nil {
		return err
	}
	renamed, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if renamed == 0 {
		return errNoBusiness
	}
	return nil
}
