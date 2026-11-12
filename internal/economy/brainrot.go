package economy

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const brainrotSalePercent = 40

func brainrotFlex(effect string, apply func(Profession) Profession) *flex {
	return &flex{effect: effect, apply: apply}
}

var brainrotGoods = []item{
	{id: "lirili-larila", name: "Lirilì Larilà", emoji: "🌵", rarity: epic, value: 500_000, brainrot: true, aliases: []string{"lirili", "larila", "elefante cacto"},
		flex: brainrotFlex("+2% de proteção", func(p Profession) Profession { p.guard += 0.02; return p })},
	{id: "brr-brr-patapim", name: "Brr Brr Patapim", emoji: "🌳", rarity: epic, value: 1_000_000, brainrot: true, aliases: []string{"patapim", "brr brr"},
		flex: brainrotFlex("+3% no !daily", func(p Profession) Profession { p.dailyBonus += 3; return p })},
	{id: "chimpanzini-bananini", name: "Chimpanzini Bananini", emoji: "🐒", rarity: epic, value: 2_000_000, brainrot: true, aliases: []string{"chimpanzini", "bananini"},
		flex: brainrotFlex("+2% de chance de achar item no !trabalhar", func(p Profession) Profession { p.findBonus += 0.02; return p })},
	{id: "trippi-troppi", name: "Trippi Troppi", emoji: "🦐", rarity: epic, value: 4_000_000, brainrot: true, aliases: []string{"trippi", "troppi"},
		flex: brainrotFlex("+2% de lucro quando ganha no cassino", func(p Profession) Profession { p.casinoLuck += 2; return p })},
	{id: "boneca-ambalabu", name: "Boneca Ambalabu", emoji: "🐸", rarity: epic, value: 7_000_000, brainrot: true, aliases: []string{"ambalabu", "boneca"},
		flex: brainrotFlex("10% menos chance de se machucar no trampo", func(p Profession) Profession { p.safety += 10; return p })},
	{id: "cappuccino-assassino", name: "Cappuccino Assassino", emoji: "🗡️", rarity: epic, value: 12_000_000, brainrot: true, aliases: []string{"cappuccino", "assassino"},
		flex: brainrotFlex("+3% de chance no !roubar", func(p Profession) Profession { p.robBonus += 0.03; return p })},
	{id: "ballerina-cappuccina", name: "Ballerina Cappuccina", emoji: "🩰", rarity: legendary, value: 20_000_000, brainrot: true, aliases: []string{"ballerina", "cappuccina", "bailarina"},
		flex: brainrotFlex("+5% de chance de fugir da cadeia", func(p Profession) Profession { p.escapeBonus += 0.05; return p })},
	{id: "tung-tung-tung-sahur", name: "Tung Tung Tung Sahur", emoji: "🪵", rarity: legendary, value: 35_000_000, brainrot: true, aliases: []string{"tung tung", "tung", "sahur"},
		flex: brainrotFlex("+3% de proteção", func(p Profession) Profession { p.guard += 0.03; return p })},
	{id: "bombombini-gusini", name: "Bombombini Gusini", emoji: "🪿", rarity: legendary, value: 60_000_000, brainrot: true, aliases: []string{"bombombini", "gusini"},
		flex: brainrotFlex("os poderes (!roubar, !furtar, !sequestrar...) e as extrações recarregam 5% mais rápido", func(p Profession) Profession { p.haste -= 5; return p })},
	{id: "bombardiro-crocodilo", name: "Bombardiro Crocodilo", emoji: "🐊", rarity: legendary, value: 100_000_000, brainrot: true, aliases: []string{"bombardiro", "crocodilo"},
		flex: brainrotFlex("+3% no !trabalhar", func(p Profession) Profession { return p.paidMore(3) })},
	{id: "frigo-camelo", name: "Frigo Camelo", emoji: "🐪", rarity: legendary, value: 175_000_000, brainrot: true, aliases: []string{"frigo", "camelo"},
		flex: brainrotFlex("o !trabalhar recarrega 3% mais rápido", func(p Profession) Profession { p.every = p.every * 97 / 100; return p })},
	{id: "tralalero-tralala", name: "Tralalero Tralala", emoji: "🦈", rarity: legendary, value: 300_000_000, brainrot: true, aliases: []string{"tralalero", "tralala"},
		flex: brainrotFlex("+4% no !trabalhar", func(p Profession) Profession { return p.paidMore(4) })},
	{id: "la-vaca-saturno-saturnita", name: "La Vaca Saturno Saturnita", emoji: "🐄", rarity: legendary, value: 500_000_000, brainrot: true, aliases: []string{"vaca saturno", "saturnita", "vaca"},
		flex: brainrotFlex("+5% no !trabalhar e +5% no !daily", func(p Profession) Profession { p.dailyBonus += 5; return p.paidMore(5) })},
	{id: "la-grande-combinasion", name: "La Grande Combinasion", emoji: "🎰", rarity: legendary, value: 1_000_000_000, brainrot: true, aliases: []string{"grande combinasion", "combinasion"},
		flex: brainrotFlex("+5% no !trabalhar e o !trabalhar recarrega 5% mais rápido", func(p Profession) Profession {
			p.every = p.every * 95 / 100
			return p.paidMore(5)
		})},
}

func brainrots() []item {
	return slices.DeleteFunc(slices.Clone(catalog), func(it item) bool { return !it.brainrot })
}

func (b *bot) brainrot(ctx context.Context, evt *events.Message) error {
	q := parseItemQuery(core.MessageText(evt.Message))
	words := strings.Fields(q.name)
	if len(words) > 0 && slices.Contains([]string{"comprar", "buy"}, words[0]) {
		q.name = strings.Join(words[1:], " ")
		return b.buyBrainrotNamed(ctx, evt, q)
	}
	lines := []string{"🧠 *Brainrots*", "_Os bichos mais caros (e mais sem sentido) da economia. Coisa de quem já não sabe mais onde enfiar dinheiro_", ""}
	for _, it := range brainrots() {
		lines = append(lines, fmt.Sprintf("%s: %s (%s)", it.label(), Coins(it.value), it.flex.effect))
	}
	lines = append(lines, "",
		fmt.Sprintf("_Compra com !brainrot comprar <nome> (ou !comprar <nome>). Cada um vale só de estar na mochila, e cada tipo conta uma vez. Ninguém consegue furtar, mas revendendo pro mercado só volta %d%% do preço_", brainrotSalePercent))
	return b.SendText(ctx, evt.Info.Chat, strings.Join(lines, "\n"))
}

func (b *bot) buyBrainrotNamed(ctx context.Context, evt *events.Message, q itemQuery) error {
	chat := evt.Info.Chat.ToNonAD()
	if q.name == "" {
		return b.SendText(ctx, chat, "Diz qual brainrot quer: !brainrot comprar tralalero. Manda !brainrot pra ver todos")
	}
	matches := preferItems(findItems(q.name), func(it item) bool { return it.brainrot })
	if problem := itemQueryProblem(q.name, matches); problem != "" {
		return b.SendText(ctx, chat, problem)
	}
	if !matches[0].brainrot {
		return b.SendText(ctx, chat, fmt.Sprintf("%s não é brainrot. Manda !brainrot pra ver os que tem", matches[0].label()))
	}
	return b.buyBrainrot(ctx, evt, matches[0])
}

func (b *bot) buyBrainrot(ctx context.Context, evt *events.Message, it item) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	balance, err := b.Wallets.buyItem(ctx, chat, b.Account(ctx, person), it, 1, it.value, PixLevy{})
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, chat, fmt.Sprintf("💸 %s custa %s, e você não tem isso. Confere no !saldo", it.label(), Coins(it.value)))
	}
	if err != nil {
		return fmt.Errorf("comprar brainrot: %w", err)
	}
	text := fmt.Sprintf("🧠 @%s torrou %s e levou %s pra casa. Ninguém entendeu nada, mas todo mundo respeitou\n✨ %s\n💰 Saldo: %s",
		person.User, Coins(it.value), it.label(), it.flex.effect, Coins(balance))
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}
