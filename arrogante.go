package main

import (
	"context"
	"fmt"
	"math/rand/v2"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) arrogante(ctx context.Context, evt *events.Message) error {
	frases := []string{
		"⚽ @%s, pra que jogar a bola na Mavie? É só uma criança! ARROGANTCHI 🥸",
		"🥸 Cala a boca, @%s, arroganhti",
		"😤 @%s, onde você quer? ¿De onde quieres? ARROGANTCHI",
		"👶 @%s jogou a bola na Mavie. Pra quê, pô? ARROGANTCHI 😭",
		"⚽ @%s, é só uma criança, pô! Pra que isso? ARROGANTCHI 🥸",
	}

	target := firstMentionOrSender(evt)
	text := fmt.Sprintf(frases[rand.IntN(len(frases))], target.User)
	_, err := b.sendMentions(ctx, evt.Info.Chat, text, []types.JID{target})
	return err
}
