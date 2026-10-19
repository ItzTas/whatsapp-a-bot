package economy

import (
	"context"
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
	blackMarketStockSize  = 8
	blackMarketMaxDeal    = 10
	blackMarketBuyback    = 60
	blackMarketMinPercent = 60
	blackMarketMaxPercent = 180
	raidChance            = 0.12
	hotGearChance         = 0.25
	hotToolChance         = 0.5
	drugMixOverdose       = 0.15
)

var errSoldOut = errors.New("muamba já vendida")

type armament struct {
	crime float64
	kill  float64
	guard float64
}

type high struct {
	lasts    time.Duration
	overdose float64
	pay      int64
	crime    float64
	escape   float64
	heist    float64
	cashback int64
	revive   float64
	kidnap   float64
	kill     float64
	guard    float64
	rush     int64
	luck     int
}

type smuggling struct {
	give   func(tx *ledgerTx) error
	get    func(tx *ledgerTx) error
	raided bool
	jail   time.Duration
}

type raid struct {
	raided  bool
	jail    time.Duration
	fine    int64
	seized  []stack
	freed   types.JID
	balance int64
}

type dose struct {
	death
	mixed     item
	overdosed bool
}

func (a armament) perks() []string {
	parts := []struct {
		on   bool
		text string
	}{
		{a.crime > 0, fmt.Sprintf("+%d%% de chance no !roubar, no !furtar e no !sequestrar", percent(a.crime))},
		{a.kill > 0, fmt.Sprintf("+%d%% no !matar (e libera o !matar pra qualquer profissão)", percent(a.kill))},
		{a.guard > 0, fmt.Sprintf("quem tenta te roubar, furtar, sequestrar ou matar tem %d%% menos chance", percent(a.guard))},
	}
	var found []string
	for _, p := range parts {
		if p.on {
			found = append(found, p.text)
		}
	}
	return found
}

func (h *high) effects() []string {
	parts := []struct {
		on   bool
		text string
	}{
		{h.pay > 0, fmt.Sprintf("+%d%% no !trabalhar", h.pay)},
		{h.crime > 0, fmt.Sprintf("+%d%% de chance no !roubar e no !furtar", percent(h.crime))},
		{h.escape > 0, fmt.Sprintf("+%d%% de chance no !fugir", percent(h.escape))},
		{h.heist > 0, fmt.Sprintf("+%d%% pra equipe no !assalto", percent(h.heist))},
		{h.cashback > 0, fmt.Sprintf("%d%% de volta quando perde no !apostar e no !cassino", h.cashback)},
		{h.revive > 0, fmt.Sprintf("poder de !reviver alguém com %d%% de chance", percent(h.revive))},
		{h.kidnap > 0, fmt.Sprintf("+%d%% de chance no !sequestrar", percent(h.kidnap))},
		{h.kill > 0, fmt.Sprintf("+%d%% no !matar (e libera o !matar pra qualquer profissão)", percent(h.kill))},
		{h.guard > 0, fmt.Sprintf("quem tenta te roubar, furtar, sequestrar ou matar tem %d%% menos chance", percent(h.guard))},
		{h.rush > 0, fmt.Sprintf("espera %d%% menos no !trabalhar, nas extrações e nos crimes", h.rush)},
		{h.luck > 0, fmt.Sprintf("+%s no %s (fica com o melhor achado)", core.CountLabel(h.luck, "sorteio extra", "sorteios extras"), extractionCommands())},
	}
	var found []string
	for _, p := range parts {
		if p.on {
			found = append(found, p.text)
		}
	}
	return found
}

func weaponPerk(it item) string {
	text := "só a melhor arma da mochila conta: " + core.JoinWithAnd(it.arms.perks())
	if it.heist > 0 {
		text += fmt.Sprintf(". No !assalto soma +%d%% pra equipe, e é gasta lá", percent(it.heist))
	}
	if !it.illegal {
		return text
	}
	return text + ". A polícia apreende se você for preso"
}

func drugPerk(h *high) string {
	text := fmt.Sprintf("!usar dá %s por %s", core.JoinWithAnd(h.effects()), core.FormatWait(h.lasts))
	if h.overdose == 0 {
		return text + fmt.Sprintf(". Sozinho não dá overdose, mas misturado com outra coisa tem %d%% de chance", percent(drugMixOverdose))
	}
	return text + fmt.Sprintf(". Overdose: %d%% de chance, ou %d%% se já tiver sob efeito de outra coisa", percent(h.overdose), percent(h.overdose+drugMixOverdose))
}

func contraband() []item {
	return slices.DeleteFunc(slices.Clone(catalog), func(it item) bool { return !it.illegal })
}

func contrabandIDs(keep func(item) bool) []any {
	var ids []any
	for _, it := range contraband() {
		if keep(it) {
			ids = append(ids, it.id)
		}
	}
	return ids
}

func hotTool(it item) bool {
	return it.gear != nil && it.gear.tool && it.rarity == legendary && it.upgrades == ""
}

func hotGear(it item) bool {
	return hotTool(it) || it.hot || (it.gear != nil && it.gear.relicLure > 0)
}

func oneOfAKind(it item) bool {
	return hotGear(it) || it.graft != nil
}

func hotTools() []item {
	var tools []item
	for _, it := range catalog {
		if hotTool(it) {
			tools = append(tools, it)
		}
	}
	return tools
}

func hotGearArrivals(chat types.JID, now time.Time) []item {
	var arrived []item
	for _, it := range catalog {
		if hotGear(it) && !hotTool(it) && dailyRand(chat, now, "muamba:"+it.id).Float64() < hotGearChance {
			arrived = append(arrived, it)
		}
	}
	grafts := implantGoods()
	arrived = append(arrived, grafts[dailyRand(chat, now, "muamba:implante").IntN(len(grafts))])
	r := dailyRand(chat, now, "muamba:ferramenta")
	if r.Float64() >= hotToolChance {
		return arrived
	}
	tools := hotTools()
	return append(arrived, tools[r.IntN(len(tools))])
}

func blackMarketDay(now time.Time) string {
	return now.In(core.ReminderZone).Format(time.DateOnly)
}

func claimHotGear(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, it item, now time.Time) error {
	res, err := tx.ExecContext(ctx,
		"INSERT INTO muambas (chat, dia, item, comprador) VALUES (?, ?, ?, ?) ON CONFLICT DO NOTHING",
		chat.String(), blackMarketDay(now), it.id, acc.Key,
	)
	if err != nil {
		return err
	}
	claimed, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if claimed == 0 {
		return errSoldOut
	}
	return nil
}

func (s *walletStore) soldHotGear(ctx context.Context, chat types.JID, now time.Time) (sold map[string]bool, err error) {
	rows, err := s.db.QueryContext(ctx, "SELECT item FROM muambas WHERE chat = ? AND dia = ?", chat.String(), blackMarketDay(now))
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	sold = map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		sold[id] = true
	}
	return sold, rows.Err()
}

func inStock(stock []item) func(item) bool {
	return func(it item) bool { return slices.ContainsFunc(stock, func(s item) bool { return s.id == it.id }) }
}

func blackMarketStock(chat types.JID, now time.Time) []item {
	pool := slices.DeleteFunc(contraband(), func(it item) bool { return it.graft != nil })
	dailyRand(chat, now, "mercadonegro").Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	stock := append(pool[:min(blackMarketStockSize, len(pool))], hotGearArrivals(chat, now)...)
	slices.SortFunc(stock, compareItems)
	return stock
}

func blackMarketPercent(chat types.JID, it item, now time.Time) int64 {
	return int64(blackMarketMinPercent + dailyRand(chat, now, it.id).IntN(blackMarketMaxPercent-blackMarketMinPercent+1))
}

func blackMarketPrice(chat types.JID, it item, now time.Time) int64 {
	return max(1, it.value*blackMarketPercent(chat, it, now)/100)
}

func blackMarketPayout(chat types.JID, it item, quantity int64, now time.Time) int64 {
	return max(quantity, blackMarketPrice(chat, it, now)*quantity*blackMarketBuyback/100)
}

func (p Profession) raidOdds() float64 {
	if p.divine {
		return 0
	}
	return raidChance * (1 - p.immunity())
}

func (p Profession) armed(w item) Profession {
	if w.arms == nil {
		return p
	}
	p.weapon = w.id
	p.robBonus += w.arms.crime
	p.theftBonus += w.arms.crime
	p.kidnapBonus += w.arms.crime
	p.killChance += w.arms.kill
	p.guard += w.arms.guard
	return p
}

func (p Profession) drugged(id string) Profession {
	it, ok := itemByID(id)
	if !ok || it.high == nil {
		return p
	}
	h := it.high
	p.drug = id
	p.min = p.min * (100 + h.pay) / 100
	p.max = p.max * (100 + h.pay) / 100
	p.robBonus += h.crime
	p.theftBonus += h.crime
	p.escapeBonus += h.escape
	p.breakoutBonus += h.escape
	p.heistBonus += h.heist
	p.cashback += h.cashback
	p.reviveChance += h.revive
	p.kidnapBonus += h.kidnap
	p.killChance += h.kill
	p.guard += h.guard
	p.every = p.every * time.Duration(100-h.rush) / 100
	p.haste -= h.rush
	p.gatherLuck += h.luck
	return p
}

func contrabandPerks(p Profession) []string {
	var lines []string
	if w, ok := itemByID(p.weapon); ok {
		lines = append(lines, fmt.Sprintf("🔫 Tá armado com %s (as chances acima já contam com ela)", w.label()))
	}
	if d, ok := itemByID(p.drug); ok {
		lines = append(lines, fmt.Sprintf("🌀 Tá sob efeito de %s (as chances acima já contam com ele)", d.label()))
	}
	return lines
}

func hostageEscape(freed types.JID, format string) (string, []types.JID) {
	if freed.IsEmpty() {
		return "", nil
	}
	return fmt.Sprintf(format, freed.User), []types.JID{freed}
}

func lethalWeapons() string {
	var labels []string
	for _, it := range contraband() {
		if it.arms != nil && it.arms.kill > 0 {
			labels = append(labels, it.label())
		}
	}
	return core.JoinWithAnd(labels)
}

func lethalDrugs() string {
	var labels []string
	for _, it := range contraband() {
		if it.high != nil && it.high.kill > 0 {
			labels = append(labels, it.label())
		}
	}
	return core.JoinWithAnd(labels)
}

func hotGearLabels() string {
	var labels []string
	for _, it := range catalog {
		if hotGear(it) && !hotTool(it) {
			labels = append(labels, it.label())
		}
	}
	return core.JoinWithAnd(labels)
}

func seizedText(seized []stack) string {
	if len(seized) == 0 {
		return ""
	}
	labels := make([]string, len(seized))
	for i, s := range seized {
		labels[i] = quantityLabel(s.item, s.quantity)
	}
	return "\n🚨 A polícia apreendeu " + core.JoinWithAnd(labels) + " da mochila"
}

func priceTrend(pct int64) string {
	if pct < 100 {
		return fmt.Sprintf("📉 %d%% da tabela", pct)
	}
	return fmt.Sprintf("📈 %d%% da tabela", pct)
}

func blackMarketText(chat types.JID, sold map[string]bool, now time.Time) string {
	lines := []string{"🕶️ *Mercado negro*", "_Ninguém viu nada, ninguém sabe de nada. O estoque e os preços mudam todo dia_"}
	stock := blackMarketStock(chat, now)
	sections := []struct {
		title string
		keep  func(item) bool
	}{
		{"🔫 *Armas*", func(it item) bool { return it.arms != nil }},
		{"💊 *Drogas*", func(it item) bool { return it.high != nil }},
		{"🕶️ *Contrabando*", func(it item) bool { return it.illegal && it.arms == nil && it.high == nil && it.graft == nil }},
		{"🧰 *Muamba*", hotGear},
		{"🦾 *Implante do dia*", func(it item) bool { return it.graft != nil }},
	}
	for _, s := range sections {
		goods := slices.DeleteFunc(slices.Clone(stock), func(it item) bool { return !s.keep(it) })
		if len(goods) == 0 {
			continue
		}
		lines = append(lines, "", s.title)
		for _, it := range goods {
			if sold[it.id] {
				lines = append(lines, fmt.Sprintf("• %s: ~esgotado~, alguém já levou", it.label()))
				continue
			}
			lines = append(lines, fmt.Sprintf("• %s: %s (%s)", it.label(), Coins(blackMarketPrice(chat, it, now)), priceTrend(blackMarketPercent(chat, it, now))))
		}
	}
	lines = append(lines, "",
		fmt.Sprintf("_Compra com !mercadonegro comprar <item> e vende com !mercadonegro vender <item>, a %d%% do preço do dia (dá pra vender qualquer coisa ilegal, mesmo fora do estoque). Manda !item <nome> pra ver o que cada um faz_", blackMarketBuyback),
		"",
		fmt.Sprintf("_De vez em quando chega muamba desviada: uma ferramenta de extração lendária por dia no máximo (%d%% de chance de vir uma, sorteada entre as %d) e %s, com %d%% de chance por dia. Só vem uma de cada, e quem comprar primeiro leva. Ela não é ilegal, então a polícia não apreende, mas o receptador não compra de volta: revende no !vender_", percent(hotToolChance), len(hotTools()), hotGearLabels(), percent(hotGearChance)),
		"",
		fmt.Sprintf("_Todo dia chega um implante desviado, sorteado entre os %d, e só vem um: quem comprar primeiro leva. Ele vai pra mochila, e aí é !implante instalar <nome>. O clandestino 🚨 é contrabando, o legal é muamba_", len(implants)),
		"",
		fmt.Sprintf("⚠️ Toda negociação conta como crime (a polícia pode te !prender) e tem %d%% de chance de dar batida: aí você vai preso, perde a mercadoria e paga multa. Quem é preso ainda perde tudo o que é ilegal da mochila", percent(raidChance)),
		"",
		fmt.Sprintf("_O estoque muda em %s_", core.FormatWait(nextRestock(now))))
	return strings.Join(lines, "\n")
}

func (b *bot) mercadonegro(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	q := parseItemQuery(core.MessageText(evt.Message))
	actions := map[string]func(context.Context, *events.Message, itemQuery) error{
		"comprar": b.buyContrabandNamed,
		"buy":     b.buyContrabandNamed,
		"vender":  b.sellContraband,
		"sell":    b.sellContraband,
	}
	words := strings.Fields(q.name)
	if len(words) > 0 {
		if action, ok := actions[words[0]]; ok {
			q.name = strings.Join(words[1:], " ")
			return action(ctx, evt, q)
		}
	}
	now := time.Now()
	sold, err := b.Wallets.soldHotGear(ctx, chat, now)
	if err != nil {
		return fmt.Errorf("ler muamba vendida: %w", err)
	}
	return b.SendText(ctx, chat, blackMarketText(chat, sold, now))
}

func (b *bot) buyContrabandNamed(ctx context.Context, evt *events.Message, q itemQuery) error {
	chat := evt.Info.Chat.ToNonAD()
	if q.name == "" {
		return b.SendText(ctx, chat, "Diz o que quer comprar: !mercadonegro comprar peixeira. Manda !mercadonegro pra ver o estoque de hoje")
	}
	matches := preferItems(findItems(q.name), inStock(blackMarketStock(chat, time.Now())))
	if problem := itemQueryProblem(q.name, matches); problem != "" {
		return b.SendText(ctx, chat, problem)
	}
	return b.buyContraband(ctx, evt, matches[0], q.quantity(0))
}

func (b *bot) buyContraband(ctx context.Context, evt *events.Message, it item, quantity int64) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	now := time.Now()
	if !inStock(blackMarketStock(chat, now))(it) {
		return b.SendText(ctx, chat, fmt.Sprintf("%s não chegou no mercado negro hoje. Manda !mercadonegro pra ver o que tem", it.label()))
	}
	if quantity <= 0 || quantity > blackMarketMaxDeal {
		return b.SendText(ctx, chat, fmt.Sprintf("O contato só passa de 1 a %d de uma vez, pra não chamar atenção", blackMarketMaxDeal))
	}
	if oneOfAKind(it) && quantity > 1 {
		return b.SendText(ctx, chat, fmt.Sprintf("Só chegou um %s, muamba não vem no atacado", it.label()))
	}
	acc := b.Account(ctx, person)
	job, err := b.ProfessionOf(ctx, chat, acc)
	if err != nil {
		return err
	}
	cost := blackMarketPrice(chat, it, now) * quantity
	r, err := b.Wallets.smuggle(ctx, chat, acc, smuggling{
		give: func(tx *ledgerTx) error {
			if oneOfAKind(it) {
				if err := claimHotGear(ctx, tx, chat, acc, it, now); err != nil {
					return err
				}
			}
			return debitBalance(ctx, tx, chat, acc, cost)
		},
		get:    func(tx *ledgerTx) error { return addItems(ctx, tx, chat, acc, it.id, quantity) },
		raided: rand.Float64() < job.raidOdds(),
		jail:   Profession{jailTime: baseJail}.sentence(job),
	}, now)
	if errors.Is(err, errSoldOut) {
		return b.SendText(ctx, chat, fmt.Sprintf("Alguém já levou %s hoje. Muamba só vem uma de cada, volta amanhã pra ver se chega outra", it.label()))
	}
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, chat, fmt.Sprintf("Você não tem moeda pra isso: %s fica %s. Confere no !saldo", quantityLabel(it, quantity), Coins(cost)))
	}
	if err != nil {
		return fmt.Errorf("comprar no mercado negro: %w", err)
	}
	if r.raided {
		return b.sendRaid(ctx, chat, person, job, r, "comprando "+quantityLabel(it, quantity))
	}
	contacts := []string{
		"num beco escuro", "atrás do posto de gasolina", "no estacionamento do mercado", "num carro de vidro fumê",
		"no fundo de uma borracharia", "num baile no meio da semana", "pela janela de um Uno com escada no teto",
	}
	text := fmt.Sprintf("🕶️ @%s encontrou o contato %s e levou %s por %s. Ninguém viu nada\n💰 Saldo: %s",
		person.User, core.PickOne(contacts), quantityLabel(it, quantity), Coins(cost), Coins(r.balance))
	if _, err := b.SendMentions(ctx, chat, text, []types.JID{person}); err != nil {
		return err
	}
	b.stirEvent(ctx, evt, eventSpot{place: "!mercadonegro", won: true})
	return nil
}

func (b *bot) sellContraband(ctx context.Context, evt *events.Message, q itemQuery) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	if q.name == "" {
		return b.SendText(ctx, chat, fmt.Sprintf("Diz o que quer vender: !mercadonegro vender peixeira (ou tudo). O receptador paga %d%% do preço do dia", blackMarketBuyback))
	}
	it, have, problem, err := b.heldForSale(ctx, chat, acc, q.name)
	if err != nil {
		return err
	}
	if problem != "" {
		return b.SendText(ctx, chat, problem)
	}
	if !it.illegal {
		return b.SendText(ctx, chat, fmt.Sprintf("O receptador só compra o que é ilegal (arma, droga e contrabando). %s você vende no !vender", it.label()))
	}
	quantity := q.quantity(have)
	if quantity > have {
		return b.SendText(ctx, chat, "Você só tem "+quantityLabel(it, have))
	}
	job, err := b.ProfessionOf(ctx, chat, acc)
	if err != nil {
		return err
	}
	now := time.Now()
	payout := blackMarketPayout(chat, it, quantity, now)
	r, err := b.Wallets.smuggle(ctx, chat, acc, smuggling{
		give: func(tx *ledgerTx) error { return takeItems(ctx, tx, chat, acc, it.id, quantity) },
		get: func(tx *ledgerTx) error {
			_, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", payout, chat.String(), acc.Key)
			return err
		},
		raided: rand.Float64() < job.raidOdds(),
		jail:   Profession{jailTime: baseJail}.sentence(job),
	}, now)
	if errors.Is(err, errNoItems) {
		return b.SendText(ctx, chat, "Sua mochila mudou no meio da venda e não dá mais 🤨 Confere no !mochila")
	}
	if err != nil {
		return fmt.Errorf("vender no mercado negro: %w", err)
	}
	if r.raided {
		return b.sendRaid(ctx, chat, person, job, r, "vendendo "+quantityLabel(it, quantity))
	}
	text := fmt.Sprintf("🕶️ @%s passou %s pra um receptador e saiu com %s no bolso (%s)\n💰 Saldo: %s",
		person.User, quantityLabel(it, quantity), Coins(payout), priceTrend(blackMarketPercent(chat, it, now)), Coins(r.balance))
	if _, err := b.SendMentions(ctx, chat, text, []types.JID{person}); err != nil {
		return err
	}
	b.stirEvent(ctx, evt, eventSpot{place: "!mercadonegro", won: true})
	return nil
}

func (b *bot) sendRaid(ctx context.Context, chat types.JID, person types.JID, job Profession, r raid, doing string) error {
	escape, freed := hostageEscape(r.freed, "\n🔓 Com o sequestrador preso, @%s saiu do cativeiro!")
	text := fmt.Sprintf("🚨 BATIDA! A polícia tava de campana e pegou @%s %s. Perdeu a mercadoria e vai passar %s na cadeia",
		person.User, doing, core.FormatWait(r.jail)) + jailFactorText(person, job) + fineText(r.fine) + seizedText(r.seized) + escape
	_, err := b.SendMentions(ctx, chat, text, append([]types.JID{person}, freed...))
	return err
}

func (s *walletStore) smuggle(ctx context.Context, chat types.JID, acc core.Account, deal smuggling, now time.Time) (r raid, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		if err := deal.give(tx); err != nil {
			return err
		}
		readBalance := func() error {
			return tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&r.balance)
		}
		if deal.raided {
			var err error
			if r, err = jailSmuggler(ctx, tx, chat, acc, deal.jail, now); err != nil {
				return err
			}
			return readBalance()
		}
		if err := markCrime(ctx, tx, chat, acc, now); err != nil {
			return err
		}
		if err := deal.get(tx); err != nil {
			return err
		}
		return readBalance()
	})
	return r, err
}

func jailSmuggler(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, jail time.Duration, now time.Time) (r raid, err error) {
	r.raided, r.jail = true, jail
	_, err = tx.ExecContext(ctx,
		"UPDATE carteiras SET preso_ate = MAX(preso_ate, ?), crime_em = 0 WHERE chat = ? AND pessoa = ?",
		now.Add(jail).Unix(), chat.String(), acc.Key,
	)
	if err != nil {
		return r, err
	}
	if r.fine, err = lockUp(ctx, tx, chat, acc, now); err != nil {
		return r, err
	}
	if r.seized, err = seizeContraband(ctx, tx, chat, acc); err != nil {
		return r, err
	}
	r.freed, err = freeHostage(ctx, tx, chat, acc, now)
	return r, err
}

func seizeContraband(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account) (seized []stack, err error) {
	ids := contrabandIDs(func(item) bool { return true })
	rows, err := tx.QueryContext(ctx,
		"DELETE FROM inventarios WHERE chat = ? AND pessoa = ? AND item IN ("+core.SQLPlaceholders(len(ids))+") RETURNING item, quantidade",
		append([]any{chat.String(), acc.Key}, ids...)...,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var (
			id       string
			quantity int64
		)
		if err := rows.Scan(&id, &quantity); err != nil {
			return nil, err
		}
		if it, ok := itemByID(id); ok {
			seized = append(seized, stack{item: it, quantity: quantity, grades: []grade{{normalQuality, quantity}}})
		}
	}
	return seized, rows.Err()
}

func (s *walletStore) bestWeapon(ctx context.Context, chat types.JID, acc core.Account) (best item, err error) {
	var ids []any
	for _, it := range catalog {
		if it.arms != nil {
			ids = append(ids, it.id)
		}
	}
	rows, err := s.db.QueryContext(ctx,
		"SELECT item FROM inventarios WHERE chat = ? AND pessoa = ? AND item IN ("+core.SQLPlaceholders(len(ids))+")",
		append([]any{chat.String(), acc.Key}, ids...)...,
	)
	if err != nil {
		return item{}, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return item{}, err
		}
		if it, ok := itemByID(id); ok && it.value > best.value {
			best = it
		}
	}
	return best, rows.Err()
}

func (b *bot) takeDrug(ctx context.Context, evt *events.Message, it item) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	d, err := b.Wallets.takeDrug(ctx, chat, b.Account(ctx, person), it, time.Now())
	if errors.Is(err, errNoItems) {
		return b.SendText(ctx, chat, fmt.Sprintf("Não tem mais %s na sua mochila 🎒", it.label()))
	}
	if err != nil {
		return fmt.Errorf("usar droga: %w", err)
	}
	text := fmt.Sprintf(it.use, person.User)
	if d.mixed.id != "" {
		text += fmt.Sprintf("\n⚠️ Misturou com %s, que ainda tava batendo", d.mixed.label())
	}
	if d.overdosed {
		escape, freed := hostageEscape(d.freed, "\n🔓 Na confusão, o refém @%s fugiu do cativeiro!")
		text += fmt.Sprintf("\n\n💀 OVERDOSE! @%s apagou e vai ficar %s fora do jogo. Dá pra sair antes pagando o !hospital ou com alguém que saiba !reviver", person.User, core.FormatWait(deathHold))
		if funeral := funeralText(person, d.lost); funeral != "" {
			text += "\n" + funeral
		}
		if d.deposed {
			text += "\n" + deposedText(person)
		}
		text += escape
		_, err = b.SendMentions(ctx, chat, text, append([]types.JID{person}, freed...))
		return err
	}
	text += fmt.Sprintf("\n\n🌀 Por %s: %s", core.FormatWait(it.high.lasts), core.JoinWithAnd(it.high.effects()))
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (s *walletStore) takeDrug(ctx context.Context, chat types.JID, acc core.Account, it item, now time.Time) (d dose, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		if err := takeItems(ctx, tx, chat, acc, it.id, 1); err != nil {
			return err
		}
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		var err error
		d, err = getHigh(ctx, tx, chat, acc, it, now)
		return err
	})
	return d, err
}

func getHigh(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, it item, now time.Time) (d dose, err error) {
	var (
		active string
		until  int64
	)
	err = tx.QueryRowContext(ctx, "SELECT droga, droga_ate FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&active, &until)
	if err != nil {
		return d, err
	}
	odds := it.high.overdose
	if mixed, ok := itemByID(active); ok && until > now.Unix() {
		d.mixed = mixed
		odds += drugMixOverdose
	}
	if d.overdosed = rand.Float64() < odds; !d.overdosed {
		_, err = tx.ExecContext(ctx,
			"UPDATE carteiras SET droga = ?, droga_ate = ? WHERE chat = ? AND pessoa = ?",
			it.id, now.Add(it.high.lasts).Unix(), chat.String(), acc.Key,
		)
		return d, err
	}
	_, err = tx.ExecContext(ctx, "UPDATE carteiras SET droga = '', droga_ate = 0 WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key)
	if err != nil {
		return d, err
	}
	d.death, err = die(ctx, tx, chat, acc, now)
	return d, err
}
