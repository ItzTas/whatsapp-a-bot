package economy

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	maxPurchase      = 100
	rotatingShopSize = 5
)

func weekdaysText(days []time.Weekday) string {
	names := map[time.Weekday]string{
		time.Sunday:    "aos domingos",
		time.Monday:    "às segundas",
		time.Tuesday:   "às terças",
		time.Wednesday: "às quartas",
		time.Thursday:  "às quintas",
		time.Friday:    "às sextas",
		time.Saturday:  "aos sábados",
	}
	labels := make([]string, len(days))
	for i, d := range days {
		labels[i] = names[d]
	}
	return core.JoinWithAnd(labels)
}

func onSale(it item, now time.Time) bool {
	return it.days == nil || slices.Contains(it.days, now.In(core.ReminderZone).Weekday())
}

func weekdayStock(now time.Time) []item {
	return slices.DeleteFunc(slices.Clone(catalog), func(it item) bool { return !it.shop || it.days == nil || !onSale(it, now) })
}

func featuredTool(it item) bool {
	return it.shop && it.rotating && it.gear != nil && it.gear.tool
}

func rotatingStock(chat types.JID, now time.Time) []item {
	tools := slices.DeleteFunc(slices.Clone(catalog), func(it item) bool { return !featuredTool(it) })
	featured := tools[localDay(now)%int64(len(tools))]
	pool := slices.DeleteFunc(slices.Clone(catalog), func(it item) bool { return !it.shop || !it.rotating || it.id == featured.id })
	dailyRand(chat, now, "loja").Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	stock := append([]item{featured}, pool[:min(rotatingShopSize-1, len(pool))]...)
	slices.SortFunc(stock, compareItems)
	return stock
}

func (b *bot) loja(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	now := time.Now()
	stock := slices.DeleteFunc(slices.Clone(catalog), func(it item) bool { return !it.shop || it.pet || it.rotating || it.days != nil })
	lines := []string{"🛒 *Loja do grupo*"}
	lines = append(lines, categoryGroups(stock, func(it item) item { return it }, shopLine)...)
	lines = append(lines, "", "⏳ *Só hoje*", "_Essas ofertas mudam todo dia, e cada item só aparece em alguns dias. Comprou, é seu pra sempre. Todo dia tem uma ferramenta épica de extração, em rodízio_")
	for _, it := range weekdayStock(now) {
		lines = append(lines, shopLine(it)+" _(só "+weekdaysText(it.days)+")_")
	}
	for _, it := range rotatingStock(chat, now) {
		lines = append(lines, shopLine(it))
	}
	lines = append(lines, fmt.Sprintf("_As ofertas trocam em %s_", core.FormatWait(nextRestock(now))), "",
		"_Pra ver tudo o que um item faz, manda !item <nome>_",
		"_Compra com !comprar <item>, tipo !comprar vara de pesca ou !comprar todynho 5. Os bichinhos ficam no !petshop_",
		fmt.Sprintf("_O resto dos itens só sai no !pescar, no !minerar, no !vasculhar, no !catar ou em caixa, as coisas finas ficam na !luxo e o que é proibido, no !mercadonegro. Vendendo pro mercado, ele paga %d%% do valor_", marketSalePercent))
	return b.SendText(ctx, chat, strings.Join(lines, "\n"))
}

func shopLine(it item) string {
	r := rarities[it.rarity]
	line := fmt.Sprintf("%s %s · %s · 🪙 %s", r.emoji, it.label(), r.name, core.Thousands(it.value))
	if text := brief(it); text != "" {
		line += " · _" + text + "_"
	}
	return line
}

func brief(it item) string {
	if it.seed != nil {
		return seedBrief(it)
	}
	if it.gear != nil {
		return gearBrief(it.gear)
	}
	if a, ok := activityUsing(it.id); ok {
		return "ferramenta do " + a.command
	}
	protections := map[string]string{
		lockItemID:      "protege a mochila do !furtar",
		alarmItemID:     "protege o saldo do !roubar",
		vestItemID:      "segura um !matar",
		forgedReleaseID: "tira da cadeia, mas é crime",
	}
	if text, ok := protections[it.id]; ok {
		return text
	}
	if it.heal > 0 && it.doses > 1 {
		return fmt.Sprintf("recupera %d de saúde, %d doses", it.heal, it.doses)
	}
	if it.heal > 0 {
		return fmt.Sprintf("recupera %d de saúde", it.heal)
	}
	if it.heist > 0 {
		return fmt.Sprintf("+%d%% pra equipe no !assalto, e gasta", percent(it.heist))
	}
	if it.flex != nil {
		return cmp.Or(it.flex.brief, it.flex.effect)
	}
	if it.trade != nil {
		a := areas[it.trade.area]
		return fmt.Sprintf("rende mais no trampo de %s %s", a.emoji, a.name)
	}
	if it.networks != nil {
		return "posta " + networkNames(it.networks)
	}
	if c, ok := charmByID(it.bless); ok {
		return fmt.Sprintf("bênção %s por %s", c.label(), core.FormatWait(c.hold))
	}
	if c, ok := charmByID(it.hex); ok {
		return "joga " + c.label() + " em alguém"
	}
	if it.loot != nil {
		return "sai um item aleatório"
	}
	if it.prizes != nil {
		top := slices.MaxFunc(it.prizes, func(a, b prize) int { return cmp.Compare(a.amount, b.amount) })
		return "prêmio de até " + Coins(top.amount)
	}
	if it.resets != nil {
		return "zera a espera " + spots("do", "das extrações", resetList(it.resets))
	}
	if it.pet {
		return fmt.Sprintf("às vezes traz um item no !daily (%d%%)", percent(it.fetchChance()))
	}
	return ""
}

func gearBrief(g *gear) string {
	if g.tool {
		return "ferramenta melhor " + spots("do", "das extrações", g.activities)
	}
	tags := []struct {
		on   bool
		text string
	}{
		{g.metal, "só vem metal"},
		{g.relicLure > 0, "acha mais o Relicário perdido"},
		{g.luck > 0, "acha coisa melhor"},
		{g.faster > 0, "espera menos"},
		{g.fewerJunk > 0, "vem menos lixo"},
		{g.moreRare > 0, "vem mais item raro"},
		{g.grade > 0, "vem com qualidade melhor"},
		{g.safety > 0, "se machuca menos"},
		{g.alwaysFinds, "nunca vem vazio"},
		{g.spent, "gasta 1 por vez"},
	}
	var found []string
	for _, t := range tags {
		if t.on {
			found = append(found, t.text)
		}
	}
	return spots("no", "nas extrações", g.activities) + ": " + core.JoinWithAnd(found)
}

func spots(prep, everywhere string, commands []string) string {
	extraction := func(command string) bool {
		return slices.ContainsFunc(activities, func(a activity) bool { return a.command == command })
	}
	if len(commands) == len(activities) && !slices.ContainsFunc(commands, func(c string) bool { return !extraction(c) }) {
		return everywhere
	}
	return prep + " " + core.JoinWithAnd(commands)
}

func (b *bot) comprar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	q := parseItemQuery(core.MessageText(evt.Message))
	if q.name == "" {
		return b.SendText(ctx, chat, "Diz o que quer comprar: !comprar vara de pesca (ou !comprar todynho 5). Manda !loja pra ver o que tem")
	}
	matches := preferItems(findItems(q.name), func(it item) bool { return it.shop })
	if problem := itemQueryProblem(q.name, matches); problem != "" {
		return b.SendText(ctx, chat, problem)
	}
	it := matches[0]
	quantity := q.quantity(0)
	if !it.shop && slices.ContainsFunc(luxuryStock(chat, time.Now()), func(s item) bool { return s.id == it.id }) {
		return b.buyLuxury(ctx, evt, it, quantity)
	}
	if !it.shop && inStock(blackMarketStock(chat, time.Now()))(it) {
		return b.buyContraband(ctx, evt, it, quantity)
	}
	if !it.shop {
		return b.SendText(ctx, chat, fmt.Sprintf("A loja não vende %s. Dá pra conseguir com %s, ou comprando de alguém do grupo", it.label(), core.JoinWithAnd(sources(it))))
	}
	if !onSale(it, time.Now()) {
		return b.SendText(ctx, chat, fmt.Sprintf("📅 %s só sai %s. Volta no dia certo", it.label(), weekdaysText(it.days)))
	}
	if it.rotating && !inStock(rotatingStock(chat, time.Now()))(it) {
		return b.SendText(ctx, chat, fmt.Sprintf("⏳ %s não tá nas ofertas de hoje. Ele some e volta, então fica de olho na !loja", it.label()))
	}
	if quantity <= 0 {
		return b.SendText(ctx, chat, fmt.Sprintf("Diz quantos quer: !comprar %s 3", strings.ToLower(it.name)))
	}
	if quantity > maxPurchase {
		return b.SendText(ctx, chat, fmt.Sprintf("Calma, dá pra comprar até %d de uma vez 😅", maxPurchase))
	}

	acc := b.Account(ctx, person)
	var (
		job Profession
		p   presidency
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { job, err = b.ProfessionOf(gctx, chat, acc); return err })
	g.Go(func() (err error) { p, _, err = b.Wallets.presidency(gctx, chat, time.Now()); return err })
	if err := g.Wait(); err != nil {
		return err
	}
	levy := p.levy(shopTaxDecree, acc)
	people := append([]types.JID{person}, levy.payee()...)
	cost := job.shopPrice(it.value * quantity)
	tax := levy.cut(cost)
	balance, err := b.Wallets.buyItem(ctx, chat, acc, it, quantity, cost, levy)
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, chat, fmt.Sprintf("Você não tem moeda pra isso: %s fica %s. Confere no !saldo", quantityLabel(it, quantity), Coins(cost+tax)))
	}
	if err != nil {
		return fmt.Errorf("comprar item: %w", err)
	}
	b.advance(chat, acc, shopGoal, cost)
	purchase := "🛒 @%s comprou %s por %s"
	if it.pet {
		purchase = "🐾 @%s levou %s do petshop pra casa por %s"
	}
	text := fmt.Sprintf(purchase, person.User, quantityLabel(it, quantity), Coins(cost))
	if job.discount > 0 {
		text += fmt.Sprintf(", com %d%% de desconto porque %s", job.discount, job.title)
	}
	text += shopTaxText(tax, levy) + "\n💰 Saldo: " + Coins(balance)
	_, err = b.SendMentions(ctx, chat, text, people)
	return err
}

func shopTaxText(tax int64, levy PixLevy) string {
	if tax == 0 {
		return ""
	}
	return fmt.Sprintf("\n📜 E mais %s (%d%%) de imposto da compra foram %s, por decreto", Coins(tax), levy.percent, levy.pocket())
}
