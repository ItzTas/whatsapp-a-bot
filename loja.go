package main

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const maxPurchase = 100

func (b *bot) loja(ctx context.Context, evt *events.Message) error {
	stock := slices.DeleteFunc(slices.Clone(catalog), func(it item) bool { return !it.shop || it.pet })
	lines := []string{"🛒 *Loja do grupo*"}
	lines = append(lines, categoryGroups(stock, func(it item) item { return it }, shopLine)...)
	lines = append(lines, "",
		"_Compra com !comprar <item>, tipo !comprar vara de pesca ou !comprar todynho 5. Os bichinhos ficam no !petshop_",
		fmt.Sprintf("_O resto dos itens só sai no !pescar, no !minerar, no !vasculhar, no !cacar ou em caixa, as coisas finas ficam na !luxo e o que é proibido, no !mercadonegro. Vendendo pro mercado, ele paga %d%% do valor_", marketSalePercent))
	return b.sendText(ctx, evt.Info.Chat, strings.Join(lines, "\n"))
}

func shopLine(it item) string {
	line := fmt.Sprintf("• %s: 🪙 %s", it.label(), thousands(it.value))
	if p := perk(it); p != "" {
		line += " (" + p + ")"
	}
	return line
}

func (b *bot) comprar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	q := parseItemQuery(messageText(evt.Message))
	if q.name == "" {
		return b.sendText(ctx, chat, "Diz o que quer comprar: !comprar vara de pesca (ou !comprar todynho 5). Manda !loja pra ver o que tem")
	}
	matches := preferItems(findItems(q.name), func(it item) bool { return it.shop })
	if problem := itemQueryProblem(q.name, matches); problem != "" {
		return b.sendText(ctx, chat, problem)
	}
	it := matches[0]
	quantity := q.quantity(0)
	if !it.shop && slices.ContainsFunc(luxuryStock(chat, time.Now()), func(s item) bool { return s.id == it.id }) {
		return b.buyLuxury(ctx, evt, it, quantity)
	}
	if it.illegal && inStock(blackMarketStock(chat, time.Now()))(it) {
		return b.buyContraband(ctx, evt, it, quantity)
	}
	if !it.shop {
		return b.sendText(ctx, chat, fmt.Sprintf("A loja não vende %s. Dá pra conseguir com %s, ou comprando de alguém do grupo", it.label(), joinWithAnd(sources(it))))
	}
	if quantity <= 0 {
		return b.sendText(ctx, chat, fmt.Sprintf("Diz quantos quer: !comprar %s 3", strings.ToLower(it.name)))
	}
	if quantity > maxPurchase {
		return b.sendText(ctx, chat, fmt.Sprintf("Calma, dá pra comprar até %d de uma vez 😅", maxPurchase))
	}

	acc := b.account(ctx, person)
	job, err := b.professionOf(ctx, chat, acc)
	if err != nil {
		return err
	}
	cost := job.shopPrice(it.value * quantity)
	balance, err := b.wallets.buyItem(ctx, chat, acc, it, quantity, cost)
	if errors.Is(err, errNoFunds) {
		return b.sendText(ctx, chat, fmt.Sprintf("Você não tem moeda pra isso: %s fica %s. Confere no !saldo", quantityLabel(it, quantity), coins(cost)))
	}
	if err != nil {
		return fmt.Errorf("comprar item: %w", err)
	}
	purchase := "🛒 @%s comprou %s por %s"
	if it.pet {
		purchase = "🐾 @%s levou %s do petshop pra casa por %s"
	}
	text := fmt.Sprintf(purchase, person.User, quantityLabel(it, quantity), coins(cost))
	if job.discount > 0 {
		text += fmt.Sprintf(", com %d%% de desconto porque %s", job.discount, job.title)
	}
	text += job.decrees.note(pricesDecree) + "\n💰 Saldo: " + coins(balance)
	_, err = b.sendMentions(ctx, chat, text, []types.JID{person})
	return err
}
