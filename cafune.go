package main

import (
	"context"
	"fmt"
	"math/rand/v2"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) cafune(ctx context.Context, evt *events.Message) error {
	pair, ok := coupleFrom(evt.Info.Sender, mentionedJIDs(evt.Message))
	if !ok {
		return b.sendText(ctx, evt.Info.Chat, "Marca uma ou duas pessoas: !cafune @pessoa ou !cafune @pessoa1 @pessoa2")
	}

	scenes := []string{
		"🤲 @%s fez um cafuné caprichado em @%s",
		"😌 @%s passou a mão no cabelo de @%s bem devagarzinho",
		"🛋️ @%s deitou @%s no colo e mandou um cafuné",
		"💆 @%s bagunçou o cabelo de @%s com todo o carinho",
		"🌙 @%s fez cafuné em @%s até pegar no sono",
		"🐱 @%s fez tanto cafuné em @%s que rolou até um ronronado",
		"🫳 @%s deu tapinhas na cabeça de @%s e falou \"muito bem\"",
		"🥺 @%s não resistiu à carinha de pidão e fez cafuné em @%s",
	}
	places := []string{
		"no colo", "no sofá", "na rede", "debaixo da coberta", "no meio da aula",
		"no ônibus lotado", "na fila do banco", "no meio do grupo", "na beira da praia",
	}
	reactions := []string{
		"😴 Dormiu em menos de dois minutos",
		"🥹 Pediu mais cinco minutinhos",
		"🐶 Ficou balançando um rabinho imaginário",
		"😤 Disse que não gostou, mas não saiu do lugar",
		"💤 Babou no travesseiro",
		"🪮 Agora o cabelo tá cheio de nó",
		"🫠 Derreteu igual manteiga",
		"👀 O grupo inteiro tá pedindo um também",
	}

	from, to := pair[0], pair[1]
	caption := fmt.Sprintf("%s\n\n📍 Foi %s\n😌 Nível de relaxamento: %d%%\n\n%s",
		fmt.Sprintf(scenes[rand.IntN(len(scenes))], from.User, to.User),
		places[rand.IntN(len(places))],
		rand.IntN(101),
		reactions[rand.IntN(len(reactions))],
	)
	return b.sendGIF(ctx, evt.Info.Chat, "cafune", caption, []types.JID{from, to})
}
