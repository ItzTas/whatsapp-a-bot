package main

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

const (
	hospitalFloor   = 300
	hospitalPercent = 8
)

func hospitalPrice(balance int64, _ time.Duration) int64 {
	return max(hospitalFloor, max(balance, 0)*hospitalPercent/100)
}

func (b *bot) hospital(ctx context.Context, evt *events.Message) error {
	return b.buyExit(ctx, evt, exit{
		left:     b.wallets.deathLeft,
		price:    hospitalPrice,
		release:  "UPDATE carteiras SET morto_ate = 0 WHERE chat = ? AND pessoa = ? AND morto_ate > ?",
		notHeld:  "🤨 @%s tá vivo(a). O hospital só atende quem morreu",
		paid:     "🏥 @%s pagou %s pro hospital particular e os médicos trouxeram @%s de volta com o desfibrilador! Já pode usar a economia",
		paidSelf: "🏥 @%s pagou %s pro hospital particular direto do caixão e voltou à vida. Já pode usar a economia",
	})
}
