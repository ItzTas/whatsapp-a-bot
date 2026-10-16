package main

import (
	"context"
	"fmt"
	"math/rand/v2"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) nojo(ctx context.Context, evt *events.Message) error {
	frases := []string{
		"💅 Vamos, tesouro @%s, não se misture com essa gentalha!",
		"😤 Gentalha! Gentalha! Pfff! 💨 Né, @%s?",
		"🤢 @%s, vamos embora, tesouro, que aqui só tem gentalha",
		"🤧 Tá vendo, @%s? É por isso que eu não me misturo com essa gentalha! Pfff!",
		"☕ Que nojo, @%s... vou ali tomar um cafezinho com o Professor Girafales 💐",
	}

	target := firstMentionOrSender(evt)
	text := fmt.Sprintf(frases[rand.IntN(len(frases))], target.User)
	_, err := b.sendMentions(ctx, evt.Info.Chat, text, []types.JID{target})
	return err
}
