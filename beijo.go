package main

import (
	"context"
	"fmt"
	"math/rand/v2"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) beijo(ctx context.Context, evt *events.Message) error {
	pair, ok := coupleFrom(evt.Info.Sender, mentionedJIDs(evt.Message))
	if !ok {
		return b.sendText(ctx, evt.Info.Chat, "Marca uma ou duas pessoas: !beijo @pessoa ou !beijo @pessoa1 @pessoa2")
	}

	cenas := []string{
		"💋 @%s puxou @%s pela cintura e tascou um beijão",
		"😘 @%s roubou um beijo de @%s e saiu correndo",
		"💏 @%s e @%s se beijaram no meio do grupo, sem vergonha nenhuma",
		"🌹 @%s beijou @%s igual final de novela das nove",
		"🫦 @%s deu um beijo em @%s que deixou todo mundo constrangido",
		"🌧️ @%s beijou @%s debaixo da chuva, cena de filme",
		"🎆 @%s beijou @%s na virada do ano e soltaram fogos",
		"😳 @%s foi dar um selinho em @%s e virou beijo de língua",
	}
	tipos := []string{
		"selinho tímido", "beijo de cinema", "beijo de língua", "beijo de esquimó",
		"beijo na testa", "beijo roubado", "beijo de desentupidor de pia", "beijo com gosto de Halls",
	}
	reacoes := []string{
		"🥰 O grupo inteiro shippou na hora",
		"🫣 Alguém chama um padre que isso vai dar casamento",
		"😱 A ex viu e já tá digitando...",
		"🔥 Ficaram sem ar por uns bons segundos",
		"🤢 Tinha comido alho no almoço, que pena",
		"💘 Depois dessa já pode trocar o status pra namorando",
		"🦷 Bateram os dentes, mas valeu a intenção",
		"👀 Os vizinhos tão comentando até agora",
	}

	news, err := b.cheatingNews(ctx, evt.Info.Chat, pair[:])
	if err != nil {
		return fmt.Errorf("ler casamentos: %w", err)
	}

	from, to := pair[0], pair[1]
	caption := fmt.Sprintf("%s\n\n💋 Tipo: %s\n⭐ Nota do beijo: %d/10\n\n%s%s",
		fmt.Sprintf(cenas[rand.IntN(len(cenas))], from.User, to.User),
		tipos[rand.IntN(len(tipos))],
		rand.IntN(11),
		reacoes[rand.IntN(len(reacoes))],
		news.caption(),
	)
	return b.sendGIF(ctx, evt.Info.Chat, "beijo", caption, append([]types.JID{from, to}, news.betrayed...))
}
