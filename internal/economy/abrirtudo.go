package economy

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

type unboxing struct {
	opened  []stack
	prizes  []stack
	chained int64
}

func unbox(bag []stack, luck int) unboxing {
	var (
		u       unboxing
		pending []item
	)
	for _, s := range bag {
		if s.item.loot == nil {
			continue
		}
		u.opened = append(u.opened, stack{item: s.item, quantity: s.quantity})
		for range s.quantity {
			pending = append(pending, s.item)
		}
	}
	won := map[string]int{}
	for len(pending) > 0 {
		box := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		prize := box.roll(luck).prize
		if prize.id == "" {
			continue
		}
		if prize.loot != nil {
			u.chained++
			pending = append(pending, prize)
			continue
		}
		if _, ok := won[prize.id]; !ok {
			won[prize.id] = len(u.prizes)
			u.prizes = append(u.prizes, stack{item: prize})
		}
		u.prizes[won[prize.id]].quantity++
	}
	return u
}

func (s *walletStore) openBoxes(ctx context.Context, chat types.JID, acc core.Account, u unboxing) error {
	return s.inTx(ctx, func(tx *ledgerTx) error {
		for _, box := range u.opened {
			if err := takeItems(ctx, tx, chat, acc, box.item.id, box.quantity); err != nil {
				return err
			}
		}
		for _, p := range u.prizes {
			if err := addItems(ctx, tx, chat, acc, p.item.id, p.quantity); err != nil {
				return err
			}
		}
		return nil
	})
}

func (b *bot) openAll(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	bag, err := b.Wallets.inventory(ctx, chat, acc)
	if err != nil {
		return err
	}
	box := -1
	for i, s := range bag {
		if s.item.loot != nil {
			box = i
			break
		}
	}
	if box < 0 {
		return b.SendText(ctx, chat, "Você não tem nenhuma caixa nem baú na mochila pra abrir 📦 Manda !mochila pra ver o que tem")
	}
	luck, err := b.boxLuck(ctx, chat, acc, bag[box].item)
	if err != nil {
		return err
	}
	u := unbox(bag, luck)
	err = b.Wallets.openBoxes(ctx, chat, acc, u)
	if errors.Is(err, errNoItems) {
		return b.SendText(ctx, chat, "Sua mochila mudou no meio do caminho e não dá mais 🤨 Confere no !mochila")
	}
	if err != nil {
		return fmt.Errorf("abrir tudo: %w", err)
	}
	_, err = b.SendMentions(ctx, chat, unboxingText(person, u), []types.JID{person})
	return err
}

func unboxingText(person types.JID, u unboxing) string {
	opened := make([]string, len(u.opened))
	for i, box := range u.opened {
		opened[i] = quantityLabel(box.item, box.quantity)
	}
	lines := []string{fmt.Sprintf("📦 @%s abriu tudo de uma vez: %s", person.User, strings.Join(opened, ", "))}
	if u.chained == 1 {
		lines = append(lines, "🎁 Saiu mais uma caixa lá de dentro, e já foi aberta também")
	}
	if u.chained > 1 {
		lines = append(lines, fmt.Sprintf("🎁 Saíram mais %d caixas lá de dentro, e já foram abertas também", u.chained))
	}
	lines = append(lines, "", "✨ *Saiu:*")
	lines = append(lines, rarityGroups(u.prizes, func(s stack) item { return s.item }, func(s stack) string {
		return quantityLabel(s.item, s.quantity)
	})...)
	return strings.Join(lines, "\n")
}
