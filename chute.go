package main

import (
	"context"
	"fmt"
	"math/rand/v2"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) chute(ctx context.Context, evt *events.Message) error {
	pair, ok := coupleFrom(evt.Info.Sender, mentionedJIDs(evt.Message))
	if !ok {
		return b.sendText(ctx, evt.Info.Chat, "Marca uma ou duas pessoas: !chute @pessoa ou !chute @pessoa1 @pessoa2")
	}

	cenas := []string{
		"🦵 @%s deu uma voadora em @%s",
		"👟 @%s chutou @%s pra fora do grupo (quase)",
		"⚽ @%s confundiu @%s com uma bola e mandou pro gol",
		"🥋 @%s acertou um chute giratório em @%s",
		"💥 @%s chutou @%s com tanta força que deu eco",
		"🍑 @%s deu um bico na bunda de @%s",
		"🦶 @%s chutou a canela de @%s sem dó",
		"🏈 @%s deu um chute de field goal em @%s",
	}
	destinos := []string{
		"na casa do vizinho", "no meio da rua", "na Lua 🌕", "no grupo da família",
		"dentro da piscina", "no Paraguai", "na fila do SUS", "do outro lado do país",
		"na lixeira mais próxima", "no telhado",
	}
	reacoes := []string{
		"😵 Ficou vendo estrelinha",
		"🤕 Vai precisar de gelo e de terapia",
		"😂 O grupo inteiro riu",
		"😤 Já tá planejando a vingança",
		"🚑 O SAMU já tá a caminho",
		"🫠 Levantou fingindo que não doeu",
		"📸 Alguém filmou e já virou meme",
		"🤡 Nem reagiu, só aceitou",
	}

	from, to := pair[0], pair[1]
	caption := fmt.Sprintf("%s\n\n💪 Força do chute: %d%%\n📍 Foi parar %s\n\n%s",
		fmt.Sprintf(cenas[rand.IntN(len(cenas))], from.User, to.User),
		rand.IntN(101),
		destinos[rand.IntN(len(destinos))],
		reacoes[rand.IntN(len(reacoes))],
	)
	return b.sendGIF(ctx, evt.Info.Chat, "chute", caption, []types.JID{from, to})
}
