package economy

import (
	"context"
	"fmt"
	"strings"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const creationMax = 1000

func (b *bot) criar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	god := evt.Info.Sender.ToNonAD()
	q := parseItemQuery(core.MessageText(evt.Message))
	if q.name == "" {
		return b.SendText(ctx, chat, fmt.Sprintf("Diz o que quer criar: !criar espada (ou !criar diamante 10, até %d de uma vez). Manda !item pra ver todos", creationMax))
	}
	matches := findItems(q.name)
	if problem := itemQueryProblem(q.name, matches); problem != "" {
		return b.SendText(ctx, chat, problem)
	}
	it := matches[0]
	quantity := q.quantity(0)
	if quantity <= 0 || quantity > creationMax {
		return b.SendText(ctx, chat, fmt.Sprintf("Diz quantos quer, até %d: !criar %s 3", creationMax, strings.ToLower(it.name)))
	}

	godAcc := b.Account(ctx, god)
	job, err := b.ProfessionOf(ctx, chat, godAcc)
	if err != nil {
		return fmt.Errorf("preparar criação: %w", err)
	}
	if !job.divine {
		return b.SendText(ctx, chat, "🚫 Só deus cria item do nada. Você vai ter que !comprar igual todo mundo")
	}
	if err := b.Wallets.conjureItem(ctx, chat, godAcc, it.id, quantity); err != nil {
		return fmt.Errorf("criar item: %w", err)
	}
	_, err = b.SendMentions(ctx, chat, fmt.Sprintf("⚡ @%s estalou os dedos e criou %s do nada ✨", god.User, quantityLabel(it, quantity)), []types.JID{god})
	return err
}

func (s *walletStore) conjureItem(ctx context.Context, chat types.JID, acc core.Account, id string, quantity int64) error {
	return s.inTx(ctx, func(tx *ledgerTx) error { return addItems(ctx, tx, chat, acc, id, quantity) })
}
