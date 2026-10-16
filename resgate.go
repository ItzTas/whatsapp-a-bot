package main

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) resgate(ctx context.Context, evt *events.Message) error {
	return b.buyExit(ctx, evt, exit{
		left: func(ctx context.Context, chat types.JID, acc account, now time.Time) (time.Duration, error) {
			c, err := b.wallets.captivity(ctx, chat, acc, now)
			return c.left, err
		},
		price:    exitPrice,
		release:  "DELETE FROM sequestros WHERE chat = ? AND refem = ? AND ate > ?",
		notHeld:  "🤨 @%s não tá em cativeiro. Resgate de quem?",
		paid:     "💼 @%s deixou uma mala com %s no ponto combinado e @%s foi solto(a) do cativeiro! A grana sumiu junto com os bandidos",
		paidSelf: "💼 @%s pagou %s de resgate e foi solto(a) do cativeiro! A grana sumiu junto com os bandidos",
	})
}
