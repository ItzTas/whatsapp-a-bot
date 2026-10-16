package main

import (
	"context"
	"fmt"
	"math/rand/v2"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) aura(ctx context.Context, evt *events.Message) error {
	frases := map[bool][]string{
		true: {
			"✨ +%[2]d de aura pra @%[1]s. Isso foi aura farming nível máximo 🗿",
			"🗿 @%[1]s acabou de ganhar +%[2]d de aura. Ninguém segura",
			"😎 +%[2]d aura. @%[1]s nem precisou falar nada, só existiu",
			"🔥 @%[1]s tá com a aura no talo: +%[2]d. Mogou geral",
			"🌌 Aura de @%[1]s: +%[2]d. Personagem principal detectado",
		},
		false: {
			"📉 -%[2]d de aura pra @%[1]s. Que vergonha alheia 😬",
			"💀 @%[1]s perdeu %[2]d de aura com essa. Não tem volta",
			"🤡 -%[2]d aura. @%[1]s zerou a aura e ainda ficou devendo",
			"😭 @%[1]s, você tava indo tão bem... -%[2]d de aura",
			"🚮 Aura de @%[1]s: -%[2]d. Vai ter que farmar muito pra recuperar",
		},
	}

	target := firstMentionOrSender(evt)
	options := frases[rand.IntN(2) == 0]
	points := (rand.IntN(100) + 1) * 100
	text := fmt.Sprintf(options[rand.IntN(len(options))], target.User, points)
	_, err := b.sendMentions(ctx, evt.Info.Chat, text, []types.JID{target})
	return err
}
