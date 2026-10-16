package main

import (
	"context"
	"fmt"
	"math/rand/v2"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) abracar(ctx context.Context, evt *events.Message) error {
	pair, ok := coupleFrom(evt.Info.Sender, mentionedJIDs(evt.Message))
	if !ok {
		return b.sendText(ctx, evt.Info.Chat, "Marca uma ou duas pessoas: !abracar @pessoa ou !abracar @pessoa1 @pessoa2")
	}

	scenes := []string{
		"🤗 @%s deu um abraço apertado em @%s",
		"🐻 @%s esmagou @%s num abraço de urso",
		"🏃 @%s saiu correndo e pulou nos braços de @%s",
		"🫂 @%s abraçou @%s por trás, de surpresa",
		"🥹 @%s abraçou @%s e não quer mais soltar",
		"🎬 @%s e @%s se abraçaram igual reencontro de novela",
		"🫶 @%s deu aquele abraço de consolo em @%s",
		"🐙 @%s se enrolou em @%s igual polvo",
	}
	kinds := []string{
		"abraço de urso", "abraço de lado", "abraço de tamanduá", "abraço por trás",
		"abraço de vó", "abraço com três tapinhas nas costas", "abraço de quem tava com saudade", "abraço meio sem jeito",
	}
	reactions := []string{
		"🔋 Recarregou a bateria social",
		"🦴 Estalou umas três costelas",
		"😭 Começou a chorar no ombro",
		"😳 Ficou vermelho(a) na hora",
		"🧸 Saiu com cheirinho de amaciante",
		"🫁 Ficou sem ar, mas valeu a pena",
		"📸 Alguém tirou foto pra postar",
		"🤧 Tava gripado(a), agora são dois",
	}

	from, to := pair[0], pair[1]
	caption := fmt.Sprintf("%s\n\n🫂 Tipo: %s\n⏱️ Durou %s\n\n%s",
		fmt.Sprintf(scenes[rand.IntN(len(scenes))], from.User, to.User),
		kinds[rand.IntN(len(kinds))],
		countLabel(rand.IntN(60)+1, "segundo", "segundos"),
		reactions[rand.IntN(len(reactions))],
	)
	return b.sendGIF(ctx, evt.Info.Chat, "abracar", caption, []types.JID{from, to})
}
