package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) mochila(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := firstMentionOrSender(evt).ToNonAD()
	held, err := b.wallets.inventory(ctx, chat, b.account(ctx, person))
	if err != nil {
		return fmt.Errorf("ler mochila: %w", err)
	}
	_, err = b.sendMentions(ctx, chat, backpackText(person, held), []types.JID{person})
	return err
}

func backpackText(person types.JID, held []stack) string {
	if len(held) == 0 {
		return fmt.Sprintf("🎒 A mochila de @%s tá vazia 🪹\n\n_Dá pra conseguir itens com !vasculhar, !pescar, !minerar e !catar, ou comprando na !loja_", person.User)
	}
	var units, worth, market int64
	for _, s := range held {
		units += s.quantity
		worth += s.quantity * s.item.value
		market += marketPayout(s.item.value, s.quantity)
	}
	lines := []string{fmt.Sprintf("🎒 *Mochila de @%s*", person.User)}
	lines = append(lines, categoryGroups(held, func(s stack) item { return s.item }, func(s stack) string {
		return fmt.Sprintf("%s %dx %s %s · 🪙 %s", rarities[s.item.rarity].emoji, s.quantity, s.item.emoji, s.item.name, thousands(s.item.value))
	})...)
	lines = append(lines, "", rarityLegend(), "",
		fmt.Sprintf("📦 %s, valendo %s", countLabel(int(units), "item", "itens"), coins(worth)),
		fmt.Sprintf("_No mercado sairia tudo por %s. Manda !item <nome> pra ver o que cada um faz_", coins(market)))
	return strings.Join(lines, "\n")
}

func (b *bot) heldItem(ctx context.Context, chat types.JID, acc account, name string) (item, int64, string, error) {
	held, err := b.wallets.inventory(ctx, chat, acc)
	if err != nil {
		return item{}, 0, "", fmt.Errorf("ler mochila: %w", err)
	}
	matches := preferItems(findItems(name), ownedIn(held))
	if problem := itemQueryProblem(name, matches); problem != "" {
		return item{}, 0, problem, nil
	}
	it := matches[0]
	have := quantityOf(held, it.id)
	if have == 0 {
		return it, 0, fmt.Sprintf("Não tem %s na sua mochila 🎒 Manda !mochila pra ver o que você tem", it.label()), nil
	}
	return it, have, "", nil
}

func (b *bot) dar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	usageText := "Use assim: !dar @pessoa todynho (ou !dar @pessoa todynho 3, ou tudo)"
	mentions := mentionedJIDs(evt.Message)
	q := parseItemQuery(messageText(evt.Message))
	if len(mentions) == 0 || q.name == "" {
		return b.sendText(ctx, chat, usageText)
	}
	from, to := evt.Info.Sender.ToNonAD(), mentions[0].ToNonAD()
	fromAcc, toAcc := b.account(ctx, from), b.account(ctx, to)
	if fromAcc.key == toAcc.key {
		return b.sendText(ctx, chat, "Dar presente pra você mesmo é só guardar na mochila 🤨")
	}

	it, have, problem, err := b.heldItem(ctx, chat, fromAcc, q.name)
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
	err = b.wallets.giveItem(ctx, chat, fromAcc, toAcc, it.id, quantity)
	if errors.Is(err, errNoItems) {
		return b.sendText(ctx, chat, "Sua mochila mudou no meio do caminho e não dá mais 🤨 Confere no !mochila")
	}
	if err != nil {
		return fmt.Errorf("dar item: %w", err)
	}
	text := fmt.Sprintf("🎁 @%s deu %s pra @%s", from.User, quantityLabel(it, quantity), to.User)
	if it.gift != "" && quantity == 1 {
		text = fmt.Sprintf(it.gift, from.User, to.User)
	}
	_, err = b.sendMentions(ctx, chat, text, []types.JID{from, to})
	return err
}

func (b *bot) usar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.account(ctx, person)
	q := parseItemQuery(messageText(evt.Message))
	if q.name == "" {
		return b.sendText(ctx, chat, "Diz o que quer usar: !usar todynho, !usar caixa, !usar raspadinha... Manda !mochila pra ver o que você tem")
	}
	it, _, problem, err := b.heldItem(ctx, chat, acc, q.name)
	if err != nil {
		return err
	}
	if problem != "" {
		return b.sendText(ctx, chat, problem)
	}
	if it.use == "" {
		return b.sendText(ctx, chat, unusableText(it))
	}
	if it.high != nil {
		return b.takeDrug(ctx, evt, it)
	}
	if it.bless != "" {
		return b.savorLuxury(ctx, evt, it)
	}
	text := fmt.Sprintf(it.use, person.User)
	if it.keep {
		_, err = b.sendMentions(ctx, chat, text, []types.JID{person})
		return err
	}

	effect := it.roll()
	wait, err := b.wallets.consumeItem(ctx, chat, acc, it, effect, time.Now())
	if errors.Is(err, errNoItems) {
		return b.sendText(ctx, chat, fmt.Sprintf("Não tem mais %s na sua mochila 🎒", it.label()))
	}
	if err != nil {
		return fmt.Errorf("usar item: %w", err)
	}
	if wait > 0 {
		text = fmt.Sprintf("😵‍💫 @%s já tomou cafeína demais e o coração tá pedindo arrego. Guarda %s pra daqui a %s", person.User, it.label(), formatWait(wait))
		_, err = b.sendMentions(ctx, chat, text, []types.JID{person})
		return err
	}
	_, err = b.sendMentions(ctx, chat, text+effectText(it, effect), []types.JID{person})
	return err
}

func unusableText(it item) string {
	if p := perk(it); p != "" {
		return fmt.Sprintf("%s não precisa de !usar, funciona só de estar na mochila: %s", it.label(), p)
	}
	return fmt.Sprintf("%s não dá pra usar. Dá pra guardar, !vender ou !dar pra alguém", it.label())
}

func effectText(it item, e itemEffect) string {
	if e.prize.rarity == relic {
		return fmt.Sprintf("\n\n🏛️ *RELÍQUIA!* Saiu %s, uma das coisas mais raras que existem\n%s", e.prize.label(), e.prize.summary())
	}
	if e.prize.id != "" {
		return fmt.Sprintf("\n\n✨ Saiu %s\n%s", e.prize.label(), e.prize.summary())
	}
	if it.prizes != nil && e.coins == 0 {
		return "\n\n💸 Não ganhou nada. Fica pra próxima"
	}
	if it.prizes != nil {
		return "\n\n💰 Ganhou " + coins(e.coins)
	}
	if it.resets != nil {
		return "\n\n⏱️ Já dá pra usar o " + resetCommands(it.resets) + " de novo"
	}
	return ""
}
