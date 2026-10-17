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
	blackMarketStockSize  = 6
	blackMarketMaxDeal    = 10
	blackMarketBuyback    = 60
	blackMarketMinPercent = 60
	blackMarketMaxPercent = 180
	raidChance            = 0.12
	drugMixOverdose       = 0.15
)

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
}

type smuggling struct {
	give   func(tx *sql.Tx) error
	get    func(tx *sql.Tx) error
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
	mixed     item
	overdosed bool
	freed     types.JID
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
	text := "só a melhor arma da mochila conta: " + joinWithAnd(it.arms.perks())
	if it.heist > 0 {
		text += fmt.Sprintf(". No !assalto soma +%d%% pra equipe, e é gasta lá", percent(it.heist))
	}
	if !it.illegal {
		return text
	}
	return text + ". A polícia apreende se você for preso"
}

func drugPerk(h *high) string {
	text := fmt.Sprintf("!usar dá %s por %s", joinWithAnd(h.effects()), formatWait(h.lasts))
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

func inStock(stock []item) func(item) bool {
	return func(it item) bool { return slices.ContainsFunc(stock, func(s item) bool { return s.id == it.id }) }
}

func blackMarketStock(chat types.JID, now time.Time) []item {
	pool := contraband()
	dailyRand(chat, now, "mercadonegro").Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	stock := pool[:min(blackMarketStockSize, len(pool))]
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

func (p profession) raidOdds() float64 {
	if p.untouchable() || p.divine {
		return 0
	}
	return raidChance
}

func (p profession) armed(w item) profession {
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

func (p profession) drugged(id string) profession {
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
	return p
}

func contrabandPerks(p profession) []string {
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
	return joinWithAnd(labels)
}

func seizedText(seized []stack) string {
	if len(seized) == 0 {
		return ""
	}
	labels := make([]string, len(seized))
	for i, s := range seized {
		labels[i] = quantityLabel(s.item, s.quantity)
	}
	return "\n🚨 A polícia apreendeu " + joinWithAnd(labels) + " da mochila"
}

func priceTrend(pct int64) string {
	if pct < 100 {
		return fmt.Sprintf("📉 %d%% da tabela", pct)
	}
	return fmt.Sprintf("📈 %d%% da tabela", pct)
}

func blackMarketText(chat types.JID, now time.Time) string {
	lines := []string{"🕶️ *Mercado negro*", "_Ninguém viu nada, ninguém sabe de nada. O estoque e os preços mudam todo dia_"}
	stock := blackMarketStock(chat, now)
	sections := []struct {
		title string
		keep  func(item) bool
	}{
		{"🔫 *Armas*", func(it item) bool { return it.arms != nil }},
		{"💊 *Drogas*", func(it item) bool { return it.high != nil }},
	}
	for _, s := range sections {
		goods := slices.DeleteFunc(slices.Clone(stock), func(it item) bool { return !s.keep(it) })
		if len(goods) == 0 {
			continue
		}
		lines = append(lines, "", s.title)
		for _, it := range goods {
			lines = append(lines, fmt.Sprintf("• %s: %s (%s)", it.label(), coins(blackMarketPrice(chat, it, now)), priceTrend(blackMarketPercent(chat, it, now))))
		}
	}
	lines = append(lines, "",
		fmt.Sprintf("_Compra com !mercadonegro comprar <item> e vende com !mercadonegro vender <item>, a %d%% do preço do dia (dá pra vender qualquer arma ou droga, mesmo fora do estoque). Manda !item <nome> pra ver o que cada um faz_", blackMarketBuyback),
		"",
		fmt.Sprintf("⚠️ Toda negociação conta como crime (a polícia pode te !prender) e tem %d%% de chance de dar batida: aí você vai preso, perde a mercadoria e paga multa. Quem é preso ainda perde as armas e drogas da mochila", percent(raidChance)),
		"",
		fmt.Sprintf("_O estoque muda em %s_", formatWait(nextRestock(now))))
	return strings.Join(lines, "\n")
}

func (b *bot) mercadonegro(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	q := parseItemQuery(messageText(evt.Message))
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
	return b.sendText(ctx, chat, blackMarketText(chat, time.Now()))
}

func (b *bot) buyContrabandNamed(ctx context.Context, evt *events.Message, q itemQuery) error {
	chat := evt.Info.Chat.ToNonAD()
	if q.name == "" {
		return b.sendText(ctx, chat, "Diz o que quer comprar: !mercadonegro comprar peixeira. Manda !mercadonegro pra ver o estoque de hoje")
	}
	matches := preferItems(findItems(q.name), inStock(blackMarketStock(chat, time.Now())))
	if problem := itemQueryProblem(q.name, matches); problem != "" {
		return b.sendText(ctx, chat, problem)
	}
	return b.buyContraband(ctx, evt, matches[0], q.quantity(0))
}

func (b *bot) buyContraband(ctx context.Context, evt *events.Message, it item, quantity int64) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	now := time.Now()
	if !inStock(blackMarketStock(chat, now))(it) {
		return b.sendText(ctx, chat, fmt.Sprintf("%s não chegou no mercado negro hoje. Manda !mercadonegro pra ver o que tem", it.label()))
	}
	if quantity <= 0 || quantity > blackMarketMaxDeal {
		return b.sendText(ctx, chat, fmt.Sprintf("O contato só passa de 1 a %d de uma vez, pra não chamar atenção", blackMarketMaxDeal))
	}
	acc := b.account(ctx, person)
	job, err := b.professionOf(ctx, chat, acc)
	if err != nil {
		return err
	}
	cost := blackMarketPrice(chat, it, now) * quantity
	r, err := b.wallets.smuggle(ctx, chat, acc, smuggling{
		give:   func(tx *sql.Tx) error { return debitBalance(ctx, tx, chat, acc, cost) },
		get:    func(tx *sql.Tx) error { return addItems(ctx, tx, chat, acc, it.id, quantity) },
		raided: rand.Float64() < job.raidOdds(),
		jail:   profession{jailTime: baseJail}.sentence(job),
	}, now)
	if errors.Is(err, errNoFunds) {
		return b.sendText(ctx, chat, fmt.Sprintf("Você não tem moeda pra isso: %s fica %s. Confere no !saldo", quantityLabel(it, quantity), coins(cost)))
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
		person.User, pickOne(contacts), quantityLabel(it, quantity), coins(cost), coins(r.balance))
	_, err = b.sendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) sellContraband(ctx context.Context, evt *events.Message, q itemQuery) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.account(ctx, person)
	if q.name == "" {
		return b.sendText(ctx, chat, fmt.Sprintf("Diz o que quer vender: !mercadonegro vender peixeira (ou tudo). O receptador paga %d%% do preço do dia", blackMarketBuyback))
	}
	it, have, problem, err := b.heldItem(ctx, chat, acc, q.name)
	if err != nil {
		return err
	}
	if problem != "" {
		return b.sendText(ctx, chat, problem)
	}
	if !it.illegal {
		return b.sendText(ctx, chat, fmt.Sprintf("O mercado negro só mexe com arma e droga. %s você vende no !vender", it.label()))
	}
	quantity := q.quantity(have)
	if quantity > have {
		return b.sendText(ctx, chat, "Você só tem "+quantityLabel(it, have))
	}
	job, err := b.professionOf(ctx, chat, acc)
	if err != nil {
		return err
	}
	now := time.Now()
	payout := blackMarketPayout(chat, it, quantity, now)
	r, err := b.wallets.smuggle(ctx, chat, acc, smuggling{
		give: func(tx *sql.Tx) error { return takeItems(ctx, tx, chat, acc, it.id, quantity) },
		get: func(tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", payout, chat.String(), acc.key)
			return err
		},
		raided: rand.Float64() < job.raidOdds(),
		jail:   profession{jailTime: baseJail}.sentence(job),
	}, now)
	if errors.Is(err, errNoItems) {
		return b.sendText(ctx, chat, "Sua mochila mudou no meio da venda e não dá mais 🤨 Confere no !mochila")
	}
	if err != nil {
		return fmt.Errorf("vender no mercado negro: %w", err)
	}
	if r.raided {
		return b.sendRaid(ctx, chat, person, job, r, "vendendo "+quantityLabel(it, quantity))
	}
	text := fmt.Sprintf("🕶️ @%s passou %s pra um receptador e saiu com %s no bolso (%s)\n💰 Saldo: %s",
		person.User, quantityLabel(it, quantity), coins(payout), priceTrend(blackMarketPercent(chat, it, now)), coins(r.balance))
	_, err = b.sendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) sendRaid(ctx context.Context, chat types.JID, person types.JID, job profession, r raid, doing string) error {
	escape, freed := hostageEscape(r.freed, "\n🔓 Com o sequestrador preso, @%s saiu do cativeiro!")
	text := fmt.Sprintf("🚨 BATIDA! A polícia tava de campana e pegou @%s %s. Perdeu a mercadoria e vai passar %s na cadeia",
		person.User, doing, formatWait(r.jail)) + jailFactorText(person, job) + fineText(r.fine) + seizedText(r.seized) + escape
	_, err := b.sendMentions(ctx, chat, text, append([]types.JID{person}, freed...))
	return err
}

func (s *walletStore) smuggle(ctx context.Context, chat types.JID, acc account, deal smuggling, now time.Time) (r raid, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		if err := deal.give(tx); err != nil {
			return err
		}
		readBalance := func() error {
			return tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.key).Scan(&r.balance)
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

func jailSmuggler(ctx context.Context, tx *sql.Tx, chat types.JID, acc account, jail time.Duration, now time.Time) (r raid, err error) {
	r.raided, r.jail = true, jail
	_, err = tx.ExecContext(ctx,
		"UPDATE carteiras SET preso_ate = MAX(preso_ate, ?), crime_em = 0 WHERE chat = ? AND pessoa = ?",
		now.Add(jail).Unix(), chat.String(), acc.key,
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

func seizeContraband(ctx context.Context, tx *sql.Tx, chat types.JID, acc account) (seized []stack, err error) {
	ids := contrabandIDs(func(item) bool { return true })
	rows, err := tx.QueryContext(ctx,
		"DELETE FROM inventarios WHERE chat = ? AND pessoa = ? AND item IN ("+sqlPlaceholders(len(ids))+") RETURNING item, quantidade",
		append([]any{chat.String(), acc.key}, ids...)...,
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
			seized = append(seized, stack{item: it, quantity: quantity})
		}
	}
	return seized, rows.Err()
}

func (s *walletStore) bestWeapon(ctx context.Context, chat types.JID, acc account) (best item, err error) {
	var ids []any
	for _, it := range catalog {
		if it.arms != nil {
			ids = append(ids, it.id)
		}
	}
	rows, err := s.db.QueryContext(ctx,
		"SELECT item FROM inventarios WHERE chat = ? AND pessoa = ? AND item IN ("+sqlPlaceholders(len(ids))+")",
		append([]any{chat.String(), acc.key}, ids...)...,
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
	d, err := b.wallets.takeDrug(ctx, chat, b.account(ctx, person), it, time.Now())
	if errors.Is(err, errNoItems) {
		return b.sendText(ctx, chat, fmt.Sprintf("Não tem mais %s na sua mochila 🎒", it.label()))
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
		text += fmt.Sprintf("\n\n💀 OVERDOSE! @%s apagou e vai ficar %s fora do jogo. Dá pra sair antes pagando o !hospital ou com alguém que saiba !reviver", person.User, formatWait(deathHold)) + escape
		_, err = b.sendMentions(ctx, chat, text, append([]types.JID{person}, freed...))
		return err
	}
	text += fmt.Sprintf("\n\n🌀 Por %s: %s", formatWait(it.high.lasts), joinWithAnd(it.high.effects()))
	_, err = b.sendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (s *walletStore) takeDrug(ctx context.Context, chat types.JID, acc account, it item, now time.Time) (d dose, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		if err := takeItems(ctx, tx, chat, acc, it.id, 1); err != nil {
			return err
		}
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		var (
			active string
			until  int64
		)
		err := tx.QueryRowContext(ctx, "SELECT droga, droga_ate FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.key).Scan(&active, &until)
		if err != nil {
			return err
		}
		odds := it.high.overdose
		if mixed, ok := itemByID(active); ok && until > now.Unix() {
			d.mixed = mixed
			odds += drugMixOverdose
		}
		if d.overdosed = rand.Float64() < odds; !d.overdosed {
			_, err = tx.ExecContext(ctx,
				"UPDATE carteiras SET droga = ?, droga_ate = ? WHERE chat = ? AND pessoa = ?",
				it.id, now.Add(it.high.lasts).Unix(), chat.String(), acc.key,
			)
			return err
		}
		_, err = tx.ExecContext(ctx,
			"UPDATE carteiras SET morto_ate = MAX(morto_ate, ?), droga = '', droga_ate = 0 WHERE chat = ? AND pessoa = ?",
			now.Add(deathHold).Unix(), chat.String(), acc.key,
		)
		if err != nil {
			return err
		}
		d.freed, err = freeHostage(ctx, tx, chat, acc, now)
		return err
	})
	return d, err
}
