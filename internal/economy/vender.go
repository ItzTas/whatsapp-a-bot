package economy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	marketSalePercent = 60
	mythicSalePercent = 90
	offerWindow       = 3 * time.Minute
	offerKind         = "vender"
)

type itemOffer struct {
	mu       sync.Mutex
	chat     types.JID
	seller   core.Account
	buyer    core.Account
	item     item
	quantity int64
	price    int64
	expires  time.Time
}

type itemOfferState struct {
	Chat     types.JID          `json:"chat"`
	Seller   core.StoredAccount `json:"seller"`
	Buyer    core.StoredAccount `json:"buyer"`
	Item     string             `json:"item"`
	Quantity int64              `json:"quantity"`
	Price    int64              `json:"price"`
	Expires  time.Time          `json:"expires"`
}

func OfferKey(chat types.JID, buyer core.Account) string {
	return chat.String() + "|" + buyer.Key
}

func marketPayout(it item, quantity int64) int64 {
	return marketPayoutAt(it.value, quantity, marketPercentOf(it, marketSalePercent))
}

func marketPercentOf(it item, percent int64) int64 {
	if it.rarity == mythic {
		return max(percent, mythicSalePercent)
	}
	return percent
}

func marketPayoutAt(value, quantity, percent int64) int64 {
	return max(quantity, value*quantity*percent/100)
}

func (b *bot) vender(ctx context.Context, evt *events.Message) error {
	if mentions := core.MentionedJIDs(evt.Message); len(mentions) > 0 {
		return b.offerItem(ctx, evt, mentions[0].ToNonAD())
	}
	text := core.MessageText(evt.Message)
	if numbers, ok := parseItemNumbers(text); ok {
		return b.sellNumbered(ctx, evt, numbers)
	}
	if kind, ok := parseCategorySale(text); ok {
		return b.sellCategory(ctx, evt, kind)
	}
	if sale, ok := parseBulkSale(text); ok {
		return b.sellBatch(ctx, evt, sale.pick, sale.hint)
	}
	return b.sellToMarket(ctx, evt)
}

func parseCategorySale(text string) (string, bool) {
	keywords := map[string]bool{"tipo": true, "categoria": true}
	fields := strings.Fields(strings.ToLower(text))
	if len(fields) < 2 || !keywords[fields[1]] {
		return "", false
	}
	return strings.Join(fields[2:], " "), true
}

type salePick func(held []stack, favorites map[string]bool, percent int64) ([]marketSale, []string)

type bulkSale struct {
	pick salePick
	hint string
}

func parseBulkSale(text string) (bulkSale, bool) {
	useless := bulkSale{
		pick: salesWhere(func(it item) bool { return perk(it) == "" && it.gift == "" }, "Você não tem nenhum item sem efeito na mochila"),
		hint: "_Sem efeito é o item que não tem \"Pra que serve\" no !item <nome> e não dá de presente. Pra levar junto o que o !usar só faz graça, manda !vender bugigangas_",
	}
	trinkets := bulkSale{
		pick: salesWhere(func(it item) bool { return effectPerk(it) == "" && !it.pet }, "Você não tem nenhuma bugiganga na mochila"),
		hint: "_Bugiganga é o que não tem efeito de verdade: o que não tem \"Pra que serve\", os presentes e o que o !usar só faz graça (comida, Nokia, Tamagotchi...). Os bichinhos ficam em casa_",
	}
	duplicates := bulkSale{pick: duplicateSales, hint: "_Fica 1 de cada item na mochila, sempre o de melhor qualidade, e isso vale até pros favoritos ⭐_"}
	modes := map[string]bulkSale{
		"inuteis": useless, "inutil": useless, "sobras": useless,
		"bugigangas": trinkets, "bugiganga": trinkets, "tralhas": trinkets,
		"duplicados": duplicates, "duplicadas": duplicates, "repetidos": duplicates, "repetidas": duplicates,
	}
	fields := strings.Fields(text)
	if len(fields) != 2 {
		return bulkSale{}, false
	}
	sale, ok := modes[normalizeItemName(fields[1])]
	return sale, ok
}

func parseItemNumbers(text string) ([]int, bool) {
	fields := strings.FieldsFunc(text, func(r rune) bool { return unicode.IsSpace(r) || r == ',' })
	if len(fields) < 2 {
		return nil, false
	}
	numbers := make([]int, 0, len(fields)-1)
	for _, field := range fields[1:] {
		digits, ok := strings.CutPrefix(field, "#")
		if !ok {
			return nil, false
		}
		n, err := strconv.Atoi(digits)
		if err != nil {
			return nil, false
		}
		numbers = append(numbers, n)
	}
	return numbers, true
}

func (b *bot) sellNumbered(ctx context.Context, evt *events.Message, numbers []int) error {
	pick := func(held []stack, favorites map[string]bool, percent int64) ([]marketSale, []string) {
		var (
			sales    []marketSale
			problems []string
		)
		seen := map[int]bool{}
		for _, n := range numbers {
			if seen[n] {
				continue
			}
			seen[n] = true
			sale, problem := numberedSale(n, held, favorites, percent)
			if problem != "" {
				problems = append(problems, problem)
				continue
			}
			sales = append(sales, sale)
		}
		return sales, problems
	}
	return b.sellBatch(ctx, evt, pick, "_Os números ficam do lado de cada item na !mochila_")
}

func (b *bot) sellCategory(ctx context.Context, evt *events.Message, kind string) error {
	c, ok := categoryNamed(kind)
	if !ok {
		names := make([]string, len(itemCategories))
		for i, category := range itemCategories {
			names[i] = category.heading()
		}
		return b.SendText(ctx, evt.Info.Chat.ToNonAD(), "Qual tipo? Manda assim: !vender tipo pesca\n\nOs tipos são os mesmos da !mochila:\n"+strings.Join(names, "\n"))
	}
	pick := func(held []stack, favorites map[string]bool, percent int64) ([]marketSale, []string) {
		return categorySales(c, held, favorites, percent)
	}
	return b.sellBatch(ctx, evt, pick, "_Os tipos são os mesmos da !mochila, e o !item <tipo> mostra o que cai em cada um_")
}

func categorySales(c int, held []stack, favorites map[string]bool, percent int64) ([]marketSale, []string) {
	var (
		sales    []marketSale
		problems []string
		kept     []item
	)
	for _, s := range held {
		if s.item.category() != c {
			continue
		}
		if favorites[s.item.id] {
			kept = append(kept, s.item)
			continue
		}
		if s.item.illegal {
			problems = append(problems, fmt.Sprintf("#%d %s o mercado não compra, só o !mercadonegro vender", s.item.number(), s.item.label()))
			continue
		}
		sales = append(sales, marketSale{item: s.item, quantity: s.quantity, percent: percent})
	}
	if len(sales)+len(problems)+len(kept) == 0 {
		problems = append(problems, "Você não tem nada de "+itemCategories[c].heading()+" na mochila")
	}
	return sales, append(problems, favoritesKept(kept)...)
}

func salesWhere(sellable func(item) bool, empty string) salePick {
	return func(held []stack, favorites map[string]bool, percent int64) ([]marketSale, []string) {
		var (
			sales []marketSale
			kept  []item
		)
		for _, s := range held {
			if !sellable(s.item) || s.item.illegal {
				continue
			}
			if favorites[s.item.id] {
				kept = append(kept, s.item)
				continue
			}
			sales = append(sales, marketSale{item: s.item, quantity: s.quantity, percent: percent})
		}
		if len(sales)+len(kept) == 0 {
			return nil, []string{empty}
		}
		return sales, favoritesKept(kept)
	}
}

func duplicateSales(held []stack, _ map[string]bool, percent int64) ([]marketSale, []string) {
	var (
		sales    []marketSale
		problems []string
	)
	for _, s := range held {
		if s.quantity < 2 {
			continue
		}
		if s.item.illegal {
			problems = append(problems, fmt.Sprintf("#%d %s o mercado não compra, só o !mercadonegro vender", s.item.number(), s.item.label()))
			continue
		}
		sales = append(sales, marketSale{item: s.item, quantity: s.quantity - 1, percent: percent})
	}
	if len(sales)+len(problems) == 0 {
		problems = append(problems, "Você não tem nenhum item repetido na mochila")
	}
	return sales, problems
}

func (b *bot) sellBatch(ctx context.Context, evt *events.Message, pick salePick, hint string) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	var (
		held      []stack
		favorites map[string]bool
		job       Profession
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { held, favorites, err = b.shelf(gctx, chat, acc); return err })
	g.Go(func() (err error) { job, err = b.ProfessionOf(gctx, chat, acc); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("preparar venda em lote: %w", err)
	}

	sales, problems := pick(held, favorites, job.marketPercent())
	if len(sales) == 0 {
		return b.SendText(ctx, chat, "Nada foi vendido 🤷\n"+strings.Join(problems, "\n")+"\n\n"+hint)
	}

	sold, balance, err := b.Wallets.sellItems(ctx, chat, acc, sales)
	if errors.Is(err, errNoItems) {
		return b.SendText(ctx, chat, "Sua mochila mudou no meio da venda e não dá mais 🤨 Confere no !mochila")
	}
	if err != nil {
		return fmt.Errorf("vender itens em lote: %w", err)
	}
	lines := []string{fmt.Sprintf("💸 @%s vendeu pro mercado:", person.User)}
	var total int64
	for _, sale := range sold {
		total += sale.payout
		lines = append(lines, fmt.Sprintf("#%d %s%s · %s", sale.item.number(), quantityLabel(sale.item, sale.quantity), gradesLabel(sale.grades), Coins(sale.payout)))
	}
	b.advance(chat, acc, saleGoal, total)
	lines = append(lines, "", "💵 Total: "+Coins(total)+marketBonus(job, person)+mythicNote(sold), "💰 Saldo: "+Coins(balance))
	if len(problems) > 0 {
		lines = append(lines, "", "⚠️ Ficou de fora:")
		lines = append(lines, problems...)
	}
	_, err = b.SendMentions(ctx, chat, strings.Join(lines, "\n"), []types.JID{person})
	return err
}

func numberedSale(n int, held []stack, favorites map[string]bool, percent int64) (marketSale, string) {
	it, ok := itemByNumber(n)
	if !ok {
		return marketSale{}, fmt.Sprintf("#%d não é o número de nenhum item", n)
	}
	have := quantityOf(held, it.id)
	if have == 0 {
		return marketSale{}, fmt.Sprintf("#%d %s não tá na sua mochila", n, it.label())
	}
	if favorites[it.id] {
		return marketSale{}, fmt.Sprintf("#%d ⭐ %s tá nos favoritos (!desfavoritar pra vender)", n, it.label())
	}
	if it.illegal {
		return marketSale{}, fmt.Sprintf("#%d %s o mercado não compra, só o !mercadonegro vender", n, it.label())
	}
	return marketSale{item: it, quantity: have, percent: percent}, ""
}

func marketBonus(job Profession, person types.JID) string {
	if job.salePercent == 0 {
		return ""
	}
	return fmt.Sprintf("\n%s Como @%s %s, o mercado pagou %d%% em vez de %d%%", job.emoji, person.User, job.title, job.salePercent, marketSalePercent)
}

func mythicNote(sold []marketSale) string {
	if !slices.ContainsFunc(sold, func(s marketSale) bool { return s.item.rarity == mythic }) {
		return ""
	}
	return fmt.Sprintf("\n🟠 Item mítico sai por pelo menos %d%% do valor", mythicSalePercent)
}

func (b *bot) sellToMarket(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	q := parseItemQuery(core.MessageText(evt.Message))
	if q.name == "" {
		return b.SendText(ctx, chat, fmt.Sprintf("Diz o que quer vender: !vender todynho (ou !vender todynho 3, ou tudo). Pra vender vários de uma vez, usa os números da !mochila com #: !vender #12 #40 #87, ou vende um tipo inteiro: !vender tipo pesca, ou tudo que não serve pra nada: !vender inuteis (ou !vender bugigangas, que leva junto o que o !usar só faz graça), ou o que tá repetido: !vender duplicados. Item favorito (!favoritar) só sai no !vender duplicados. O mercado paga %d%% do valor, e com !vender @pessoa todynho 50 você vende pra alguém do grupo pelo preço que quiser", marketSalePercent))
	}
	it, have, problem, err := b.heldForSale(ctx, chat, acc, q.name)
	if err != nil {
		return err
	}
	if problem != "" {
		return b.SendText(ctx, chat, problem)
	}
	if it.illegal {
		return b.SendText(ctx, chat, fmt.Sprintf("🚫 O mercado não compra %s. Passa no !mercadonegro vender, ou vende pra alguém do grupo com !vender @pessoa", it.label()))
	}
	quantity := q.quantity(have)
	if quantity > have {
		return b.SendText(ctx, chat, "Você só tem "+quantityLabel(it, have))
	}

	job, err := b.ProfessionOf(ctx, chat, acc)
	if err != nil {
		return err
	}
	sold, balance, err := b.Wallets.sellItems(ctx, chat, acc, []marketSale{{item: it, quantity: quantity, percent: job.marketPercent()}})
	if errors.Is(err, errNoItems) {
		return b.SendText(ctx, chat, "Sua mochila mudou no meio da venda e não dá mais 🤨 Confere no !mochila")
	}
	if err != nil {
		return fmt.Errorf("vender item: %w", err)
	}
	b.advance(chat, acc, saleGoal, sold[0].payout)
	buyers := []string{
		"pro Seu Zé do ferro-velho", "pra uma tia do brechó", "pra um gringo perdido na feira", "pro cara da OLX que pediu desconto",
		"pro agiota do bairro", "pra um colecionador meio estranho", "pro dono do bar da esquina", "pra um tiozinho no Marketplace",
	}
	sale := sold[0]
	text := fmt.Sprintf("💸 @%s vendeu %s%s %s por %s%s%s\n💰 Saldo: %s\n\n_Vendendo pra alguém do grupo com !vender @pessoa dá pra cobrar o valor cheio (%s)_",
		person.User, quantityLabel(it, quantity), gradesLabel(sale.grades), core.PickOne(buyers), Coins(sale.payout), marketBonus(job, person), mythicNote(sold), Coins(balance), Coins(worthOf(it, sale.grades)))
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) offerItem(ctx context.Context, evt *events.Message, buyer types.JID) error {
	chat := evt.Info.Chat.ToNonAD()
	seller := evt.Info.Sender.ToNonAD()
	usageText := "Use assim: !vender @pessoa todynho 50 (o número é o preço). Pra vender vários: !vender @pessoa todynho 3 150, que são 3 por 150 no total"
	sellerAcc, buyerAcc := b.Account(ctx, seller), b.Account(ctx, buyer)
	if sellerAcc.Key == buyerAcc.Key {
		return b.SendText(ctx, chat, "Vender pra você mesmo? Isso se chama arrumar a mochila 🤡")
	}
	q := parseItemQuery(core.MessageText(evt.Message))
	if q.name == "" || len(q.numbers) == 0 {
		return b.SendText(ctx, chat, usageText)
	}
	price := q.numbers[len(q.numbers)-1]
	q.numbers = q.numbers[:len(q.numbers)-1]

	it, have, problem, err := b.heldForSale(ctx, chat, sellerAcc, q.name)
	if err != nil {
		return err
	}
	if problem != "" {
		return b.SendText(ctx, chat, problem)
	}
	quantity := q.quantity(have)
	if quantity > have {
		return b.SendText(ctx, chat, "Você só tem "+quantityLabel(it, have))
	}
	funds, _, err := b.Wallets.Balance(ctx, chat, buyerAcc)
	if err != nil {
		return fmt.Errorf("ler saldo: %w", err)
	}
	if funds < price {
		text := fmt.Sprintf("@%s só tem %s, não dá pra pagar %s 🪹", buyer.User, Coins(funds), Coins(price))
		_, err = b.SendMentions(ctx, chat, text, []types.JID{buyer})
		return err
	}

	offer := &itemOffer{chat: chat, seller: sellerAcc, buyer: buyerAcc, item: it, quantity: quantity, price: price, expires: time.Now().Add(offerWindow)}
	key := OfferKey(chat, buyerAcc)
	if !b.storeOffer(ctx, key, offer) {
		text := fmt.Sprintf("@%s já tem uma proposta ou pedido esperando resposta. Espera aceitar, recusar ou passar o tempo dele ⏳", buyer.User)
		_, err = b.SendMentions(ctx, chat, text, []types.JID{buyer})
		return err
	}
	time.AfterFunc(offerWindow, func() { b.expireOffer(ctx, key, offer) })

	text := fmt.Sprintf("🤝 @%s quer vender %s pra @%s por %s\n_%s, e no mercado sairia por %s_\n\n@%s, manda !aceitar pra comprar ou !recusar pra dispensar. A proposta vale por %s",
		seller.User, quantityLabel(it, quantity), buyer.User, Coins(price), storePriceNote(it, quantity), Coins(marketPayout(it, quantity)), buyer.User, core.FormatWait(offerWindow))
	_, err = b.SendMentions(ctx, chat, text, []types.JID{seller, buyer})
	return err
}

func storePriceNote(it item, quantity int64) string {
	if it.shop {
		return fmt.Sprintf("Na %s custa %s", it.store(), Coins(it.value*quantity))
	}
	if luxuryGoods(it) {
		return fmt.Sprintf("Na !luxo custa %s", Coins(luxuryPrice(it)*quantity))
	}
	return "A loja não vende"
}

func (b *bot) expireOffer(ctx context.Context, key string, o *itemOffer) {
	if !b.itemOffers.CompareAndDelete(key, o) {
		return
	}
	b.forgetOffer(ctx, key, o)
	text := fmt.Sprintf("⌛ @%s não respondeu e a proposta de @%s (%s por %s) expirou",
		o.buyer.JID.User, o.seller.JID.User, quantityLabel(o.item, o.quantity), Coins(o.price))
	if _, err := b.SendMentions(ctx, o.chat, text, []types.JID{o.buyer.JID, o.seller.JID}); err != nil {
		b.Client.Log.Errorf("avisar que a proposta de venda expirou: %v", err)
	}
}

func (b *bot) storeOffer(ctx context.Context, key string, o *itemOffer) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	if _, pending := b.debtRequests.Load(key); pending {
		return false
	}
	if _, pending := b.itemOffers.LoadOrStore(key, o); pending {
		return false
	}
	state := itemOfferState{
		Chat: o.chat, Seller: core.StoreAccount(o.seller), Buyer: core.StoreAccount(o.buyer), Item: o.item.id,
		Quantity: o.quantity, Price: o.price, Expires: o.expires,
	}
	b.KeepGame(ctx, offerKind, key, state, o.expires)
	return true
}

func (b *bot) forgetOffer(ctx context.Context, key string, o *itemOffer) {
	o.mu.Lock()
	defer o.mu.Unlock()
	b.DropGame(ctx, offerKind, key)
}

func (b *bot) restoreOffer(ctx context.Context, saved core.SavedGame) error {
	var state itemOfferState
	if err := json.Unmarshal(saved.State, &state); err != nil {
		return fmt.Errorf("ler proposta de venda salva: %w", err)
	}
	it, ok := itemByID(state.Item)
	if !ok {
		return fmt.Errorf("item %q da proposta não existe mais", state.Item)
	}
	o := &itemOffer{
		chat: state.Chat, seller: state.Seller.Account(), buyer: state.Buyer.Account(), item: it,
		quantity: state.Quantity, price: state.Price, expires: state.Expires,
	}
	if _, pending := b.itemOffers.LoadOrStore(saved.Key, o); pending {
		return fmt.Errorf("já tem outra proposta pra %s", saved.Key)
	}
	time.AfterFunc(time.Until(o.expires), func() { b.expireOffer(ctx, saved.Key, o) })
	return nil
}

func (b *bot) aceitar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	key := OfferKey(chat, b.Account(ctx, evt.Info.Sender))
	if value, ok := b.itemOffers.LoadAndDelete(key); ok {
		return b.acceptOffer(ctx, key, value.(*itemOffer))
	}
	if value, ok := b.debtRequests.LoadAndDelete(key); ok {
		return b.acceptDebtRequest(ctx, key, value.(*debtRequest))
	}
	return b.SendText(ctx, chat, "Ninguém te ofereceu nada nem te pediu emprestado agora 🤷 Quando alguém mandar !vender @você ou !agiota @você, é só !aceitar")
}

func (b *bot) acceptOffer(ctx context.Context, key string, o *itemOffer) error {
	b.forgetOffer(ctx, key, o)
	text, err := b.closeDeal(ctx, o)
	if err != nil {
		return fmt.Errorf("fechar venda de item: %w", err)
	}
	_, err = b.SendMentions(ctx, o.chat, text, []types.JID{o.seller.JID, o.buyer.JID})
	return err
}

func (b *bot) closeDeal(ctx context.Context, o *itemOffer) (string, error) {
	grades, err := b.Wallets.tradeItem(ctx, o)
	cancelled := map[error]string{
		errNoItems: fmt.Sprintf("❌ @%s não tem mais %s, então a venda foi cancelada", o.seller.JID.User, quantityLabel(o.item, o.quantity)),
		ErrNoFunds: fmt.Sprintf("❌ @%s não tem %s pra pagar, então a venda foi cancelada", o.buyer.JID.User, Coins(o.price)),
		errBagFull: carryRefusal(o.item, "a de @"+o.buyer.JID.User) + ", então a venda foi cancelada",
	}
	for reason, text := range cancelled {
		if errors.Is(err, reason) {
			return text, nil
		}
	}
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("✅ Negócio fechado! @%s comprou %s%s de @%s por %s",
		o.buyer.JID.User, quantityLabel(o.item, o.quantity), gradesLabel(grades), o.seller.JID.User, Coins(o.price)), nil
}

func (b *bot) recusar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	acc := b.Account(ctx, evt.Info.Sender)
	key := OfferKey(chat, acc)
	if value, ok := b.itemOffers.LoadAndDelete(key); ok {
		o := value.(*itemOffer)
		b.forgetOffer(ctx, key, o)
		text := fmt.Sprintf("🙅 @%s recusou a proposta de @%s", o.buyer.JID.User, o.seller.JID.User)
		_, err := b.SendMentions(ctx, chat, text, []types.JID{o.buyer.JID, o.seller.JID})
		return err
	}
	if value, ok := b.debtRequests.LoadAndDelete(key); ok {
		return b.refuseDebtRequest(ctx, key, value.(*debtRequest))
	}

	buyers, lenders := b.cancelOffers(ctx, chat, acc), b.cancelDebtRequests(ctx, chat, acc)
	if len(buyers)+len(lenders) == 0 {
		return b.SendText(ctx, chat, "Não tem nenhuma proposta ou pedido pra recusar ou cancelar 🤷")
	}
	var lines []string
	if len(buyers) > 0 {
		lines = append(lines, fmt.Sprintf("🚫 @%s desistiu de vender pra %s", acc.JID.User, core.JoinMentions(buyers)))
	}
	if len(lenders) > 0 {
		lines = append(lines, fmt.Sprintf("🚫 @%s desistiu de pedir emprestado pra %s", acc.JID.User, core.JoinMentions(lenders)))
	}
	people := append(append(buyers, lenders...), acc.JID)
	_, err := b.SendMentions(ctx, chat, strings.Join(lines, "\n"), people)
	return err
}

func (b *bot) cancelOffers(ctx context.Context, chat types.JID, seller core.Account) []types.JID {
	var buyers []types.JID
	b.itemOffers.Range(func(key, value any) bool {
		o := value.(*itemOffer)
		if o.chat == chat && o.seller.Key == seller.Key && b.itemOffers.CompareAndDelete(key, value) {
			b.forgetOffer(ctx, key.(string), o)
			buyers = append(buyers, o.buyer.JID)
		}
		return true
	})
	return buyers
}
