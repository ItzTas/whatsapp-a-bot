package main

import (
	"context"
	"fmt"
	"math/rand/v2"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) chorar(ctx context.Context, evt *events.Message) error {
	frases := []string{
		"😭 Chora não, @%s, vem cá que eu te dou um abraço 🫂",
		"🧻 Toma um lencinho, @%s. Pode chorar que ninguém tá vendo (todo mundo tá vendo)",
		"🌊 @%s chorou tanto que alagou o grupo. Chamem a defesa civil",
		"🎻 Tocando a musiquinha triste pra @%s chorar em paz",
		"🥲 Engole o choro, @%s, que amanhã tem mais",
		"💧 @%s tá chorando igual criança que perdeu o pirulito",
	}

	target := firstMentionOrSender(evt)
	text := fmt.Sprintf(frases[rand.IntN(len(frases))], target.User)
	_, err := b.sendMentions(ctx, evt.Info.Chat, text, []types.JID{target})
	return err
}
