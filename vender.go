package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	marketSalePercent = 60
	offerWindow       = 3 * time.Minute
	offerKind         = "vender"
)

type itemOffer struct {
	mu       sync.Mutex
	chat     types.JID
	seller   account
	buyer    account
	item     item
	quantity int64
	price    int64
	expires  time.Time
}

type itemOfferState struct {
	Chat     types.JID     `json:"chat"`
	Seller   storedAccount `json:"seller"`
	Buyer    storedAccount `json:"buyer"`
	Item     string        `json:"item"`
	Quantity int64         `json:"quantity"`
	Price    int64         `json:"price"`
	Expires  time.Time     `json:"expires"`
}

func offerKey(chat types.JID, buyer account) string {
	return chat.String() + "|" + buyer.key
}

func marketPayout(value, quantity int64) int64 {
	return marketPayoutAt(value, quantity, marketSalePercent)
}

func marketPayoutAt(value, quantity, percent int64) int64 {
	return max(quantity, value*quantity*percent/100)
}

func (b *bot) vender(ctx context.Context, evt *events.Message) error {
	if mentions := mentionedJIDs(evt.Message); len(mentions) > 0 {
		return b.offerItem(ctx, evt, mentions[0].ToNonAD())
	}
	return b.sellToMarket(ctx, evt)
}

func (b *bot) sellToMarket(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.account(ctx, person)
	q := parseItemQuery(messageText(evt.Message))
	if q.name == "" {
		return b.sendText(ctx, chat, fmt.Sprintf("Diz o que quer vender: !vender todynho (ou !vender todynho 3, ou tudo). O mercado paga %d%% do valor, e com !vender @pessoa todynho 50 você vende pra alguém do grupo pelo preço que quiser", marketSalePercent))
	}
	it, have, problem, err := b.heldItem(ctx, chat, acc, q.name)
	if err != nil {
		return err
	}
	if problem != "" {
		return b.sendText(ctx, chat, problem)
	}
	if it.illegal {
		return b.sendText(ctx, chat, fmt.Sprintf("🚫 O mercado não compra %s. Passa no !mercadonegro vender, ou vende pra alguém do grupo com !vender @pessoa", it.label()))
	}
	quantity := q.quantity(have)
	if quantity > have {
		return b.sendText(ctx, chat, "Você só tem "+quantityLabel(it, have))
	}

	job, err := b.professionOf(ctx, chat, acc)
	if err != nil {
		return err
	}
	payout := marketPayoutAt(it.value, quantity, job.marketPercent())
	balance, err := b.wallets.sellItem(ctx, chat, acc, it, quantity, payout)
	if errors.Is(err, errNoItems) {
		return b.sendText(ctx, chat, "Sua mochila mudou no meio da venda e não dá mais 🤨 Confere no !mochila")
	}
	if err != nil {
		return fmt.Errorf("vender item: %w", err)
	}
	buyers := []string{
		"pro Seu Zé do ferro-velho", "pra uma tia do brechó", "pra um gringo perdido na feira", "pro cara da OLX que pediu desconto",
		"pro agiota do bairro", "pra um colecionador meio estranho", "pro dono do bar da esquina", "pra um tiozinho no Marketplace",
	}
	bonus := ""
	if job.salePercent > 0 {
		bonus = fmt.Sprintf("\n%s Como @%s %s, o mercado pagou %d%% em vez de %d%%", job.emoji, person.User, job.title, job.salePercent, marketSalePercent)
	}
	bonus += job.decrees.note(pricesDecree)
	text := fmt.Sprintf("💸 @%s vendeu %s %s por %s%s\n💰 Saldo: %s\n\n_Vendendo pra alguém do grupo com !vender @pessoa dá pra cobrar o valor cheio (%s)_",
		person.User, quantityLabel(it, quantity), pickOne(buyers), coins(payout), bonus, coins(balance), coins(it.value*quantity))
	_, err = b.sendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) offerItem(ctx context.Context, evt *events.Message, buyer types.JID) error {
	chat := evt.Info.Chat.ToNonAD()
	seller := evt.Info.Sender.ToNonAD()
	usageText := "Use assim: !vender @pessoa todynho 50 (o número é o preço). Pra vender vários: !vender @pessoa todynho 3 150, que são 3 por 150 no total"
	sellerAcc, buyerAcc := b.account(ctx, seller), b.account(ctx, buyer)
	if sellerAcc.key == buyerAcc.key {
		return b.sendText(ctx, chat, "Vender pra você mesmo? Isso se chama arrumar a mochila 🤡")
	}
	q := parseItemQuery(messageText(evt.Message))
	if q.name == "" || len(q.numbers) == 0 {
		return b.sendText(ctx, chat, usageText)
	}
	price := q.numbers[len(q.numbers)-1]
	q.numbers = q.numbers[:len(q.numbers)-1]

	it, have, problem, err := b.heldItem(ctx, chat, sellerAcc, q.name)
	if err != nil {
		return err
	}
	if problem != "" {
		return b.sendText(ctx, chat, problem)
	}
	quantity := q.quantity(have)
	if quantity > have {
		return b.sendText(ctx, chat, "Você só tem "+quantityLabel(it, have))
	}
	funds, _, err := b.wallets.balance(ctx, chat, buyerAcc)
	if err != nil {
		return fmt.Errorf("ler saldo: %w", err)
	}
	if funds < price {
		text := fmt.Sprintf("@%s só tem %s, não dá pra pagar %s 🪹", buyer.User, coins(funds), coins(price))
		_, err = b.sendMentions(ctx, chat, text, []types.JID{buyer})
		return err
	}

	offer := &itemOffer{chat: chat, seller: sellerAcc, buyer: buyerAcc, item: it, quantity: quantity, price: price, expires: time.Now().Add(offerWindow)}
	key := offerKey(chat, buyerAcc)
	if !b.storeOffer(ctx, key, offer) {
		text := fmt.Sprintf("@%s já tem uma proposta ou pedido esperando resposta. Espera aceitar, recusar ou passar o tempo dele ⏳", buyer.User)
		_, err = b.sendMentions(ctx, chat, text, []types.JID{buyer})
		return err
	}
	time.AfterFunc(offerWindow, func() { b.expireOffer(ctx, key, offer) })

	text := fmt.Sprintf("🤝 @%s quer vender %s pra @%s por %s\n_No mercado sairia por %s_\n\n@%s, manda !aceitar pra comprar ou !recusar pra dispensar. A proposta vale por %s",
		seller.User, quantityLabel(it, quantity), buyer.User, coins(price), coins(marketPayout(it.value, quantity)), buyer.User, formatWait(offerWindow))
	_, err = b.sendMentions(ctx, chat, text, []types.JID{seller, buyer})
	return err
}

func (b *bot) expireOffer(ctx context.Context, key string, o *itemOffer) {
	if !b.itemOffers.CompareAndDelete(key, o) {
		return
	}
	b.forgetOffer(ctx, key, o)
	text := fmt.Sprintf("⌛ @%s não respondeu e a proposta de @%s (%s por %s) expirou",
		o.buyer.jid.User, o.seller.jid.User, quantityLabel(o.item, o.quantity), coins(o.price))
	if _, err := b.sendMentions(ctx, o.chat, text, []types.JID{o.buyer.jid, o.seller.jid}); err != nil {
		b.client.Log.Errorf("avisar que a proposta de venda expirou: %v", err)
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
		Chat: o.chat, Seller: storeAccount(o.seller), Buyer: storeAccount(o.buyer), Item: o.item.id,
		Quantity: o.quantity, Price: o.price, Expires: o.expires,
	}
	b.keepGame(ctx, offerKind, key, state, o.expires)
	return true
}

func (b *bot) forgetOffer(ctx context.Context, key string, o *itemOffer) {
	o.mu.Lock()
	defer o.mu.Unlock()
	b.dropGame(ctx, offerKind, key)
}

func (b *bot) restoreOffer(ctx context.Context, saved savedGame) error {
	var state itemOfferState
	if err := json.Unmarshal(saved.state, &state); err != nil {
		return fmt.Errorf("ler proposta de venda salva: %w", err)
	}
	it, ok := itemByID(state.Item)
	if !ok {
		return fmt.Errorf("item %q da proposta não existe mais", state.Item)
	}
	o := &itemOffer{
		chat: state.Chat, seller: state.Seller.account(), buyer: state.Buyer.account(), item: it,
		quantity: state.Quantity, price: state.Price, expires: state.Expires,
	}
	if _, pending := b.itemOffers.LoadOrStore(saved.key, o); pending {
		return fmt.Errorf("já tem outra proposta pra %s", saved.key)
	}
	time.AfterFunc(time.Until(o.expires), func() { b.expireOffer(ctx, saved.key, o) })
	return nil
}

func (b *bot) aceitar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	key := offerKey(chat, b.account(ctx, evt.Info.Sender))
	if value, ok := b.itemOffers.LoadAndDelete(key); ok {
		return b.acceptOffer(ctx, key, value.(*itemOffer))
	}
	if value, ok := b.debtRequests.LoadAndDelete(key); ok {
		return b.acceptDebtRequest(ctx, key, value.(*debtRequest))
	}
	return b.sendText(ctx, chat, "Ninguém te ofereceu nada nem te pediu emprestado agora 🤷 Quando alguém mandar !vender @você ou !agiota @você, é só !aceitar")
}

func (b *bot) acceptOffer(ctx context.Context, key string, o *itemOffer) error {
	b.forgetOffer(ctx, key, o)
	text, err := b.closeDeal(ctx, o)
	if err != nil {
		return fmt.Errorf("fechar venda de item: %w", err)
	}
	_, err = b.sendMentions(ctx, o.chat, text, []types.JID{o.seller.jid, o.buyer.jid})
	return err
}

func (b *bot) closeDeal(ctx context.Context, o *itemOffer) (string, error) {
	err := b.wallets.tradeItem(ctx, o)
	cancelled := map[error]string{
		errNoItems: fmt.Sprintf("❌ @%s não tem mais %s, então a venda foi cancelada", o.seller.jid.User, quantityLabel(o.item, o.quantity)),
		errNoFunds: fmt.Sprintf("❌ @%s não tem %s pra pagar, então a venda foi cancelada", o.buyer.jid.User, coins(o.price)),
	}
	for reason, text := range cancelled {
		if errors.Is(err, reason) {
			return text, nil
		}
	}
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("✅ Negócio fechado! @%s comprou %s de @%s por %s",
		o.buyer.jid.User, quantityLabel(o.item, o.quantity), o.seller.jid.User, coins(o.price)), nil
}

func (b *bot) recusar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	acc := b.account(ctx, evt.Info.Sender)
	key := offerKey(chat, acc)
	if value, ok := b.itemOffers.LoadAndDelete(key); ok {
		o := value.(*itemOffer)
		b.forgetOffer(ctx, key, o)
		text := fmt.Sprintf("🙅 @%s recusou a proposta de @%s", o.buyer.jid.User, o.seller.jid.User)
		_, err := b.sendMentions(ctx, chat, text, []types.JID{o.buyer.jid, o.seller.jid})
		return err
	}
	if value, ok := b.debtRequests.LoadAndDelete(key); ok {
		return b.refuseDebtRequest(ctx, key, value.(*debtRequest))
	}

	buyers, lenders := b.cancelOffers(ctx, chat, acc), b.cancelDebtRequests(ctx, chat, acc)
	if len(buyers)+len(lenders) == 0 {
		return b.sendText(ctx, chat, "Não tem nenhuma proposta ou pedido pra recusar ou cancelar 🤷")
	}
	var lines []string
	if len(buyers) > 0 {
		lines = append(lines, fmt.Sprintf("🚫 @%s desistiu de vender pra %s", acc.jid.User, joinMentions(buyers)))
	}
	if len(lenders) > 0 {
		lines = append(lines, fmt.Sprintf("🚫 @%s desistiu de pedir emprestado pra %s", acc.jid.User, joinMentions(lenders)))
	}
	people := append(append(buyers, lenders...), acc.jid)
	_, err := b.sendMentions(ctx, chat, strings.Join(lines, "\n"), people)
	return err
}

func (b *bot) cancelOffers(ctx context.Context, chat types.JID, seller account) []types.JID {
	var buyers []types.JID
	b.itemOffers.Range(func(key, value any) bool {
		o := value.(*itemOffer)
		if o.chat == chat && o.seller.key == seller.key && b.itemOffers.CompareAndDelete(key, value) {
			b.forgetOffer(ctx, key.(string), o)
			buyers = append(buyers, o.buyer.jid)
		}
		return true
	})
	return buyers
}
