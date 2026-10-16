package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) engravidar(ctx context.Context, evt *events.Message) error {
	pair, ok := coupleFrom(evt.Info.Sender, mentionedJIDs(evt.Message))
	if !ok {
		return b.sendText(ctx, evt.Info.Chat, "Marca uma ou duas pessoas: !engravidar @pessoa ou !engravidar @pessoa1 @pessoa2")
	}

	scenes := []string{
		"🤰 @%s engravidou @%s",
		"😱 O teste deu positivo: @%s engravidou @%s",
		"🧪 @%s esqueceu a camisinha e agora @%s tá esperando um bebê",
		"🌙 Foi só uma noite, mas @%s engravidou @%s",
		"🎈 Chá revelação no grupo: @%s engravidou @%s",
		"🥒 @%s engravidou @%s e os desejos já começaram",
		"🏥 O médico chamou @%s num canto e contou: @%s tá grávida(o)",
		"📢 Urgente: @%s vai ter um filho com @%s",
	}
	cravings := []string{
		"pastel de feira às 3 da manhã", "sorvete com feijoada", "manga com leite, desafiando a vó",
		"coxinha de rodoviária", "picolé de limão com sal", "pizza de abacaxi", "açaí com ovo cozido",
		"miojo cru com Nescau", "farofa pura de colher",
	}
	reactions := []string{
		"😵 Desmaiou quando viu o teste",
		"🚬 Alguém disse que ia comprar cigarro e ainda não voltou",
		"🧾 O teste de DNA já foi encomendado",
		"🧶 A vó já tá tricotando sapatinho",
		"💸 O orçamento de fralda já estourou",
		"📸 O ultrassom já virou foto de perfil",
		"😭 Chorou de emoção (ou de desespero, ninguém sabe)",
		"🍼 O grupo já tá brigando pra ver quem vai ser padrinho",
	}

	news, err := b.cheatingNews(ctx, evt.Info.Chat, pair[:])
	if err != nil {
		return fmt.Errorf("ler casamentos: %w", err)
	}

	from, to := pair[0], pair[1]
	babies, sex := pregnancyOutcome()
	dueDate := time.Now().AddDate(0, 9, rand.IntN(29)-14)
	caption := fmt.Sprintf("%s\n\n👶 Vem aí: %s\n🎀 Sexo: %s\n📅 Previsão do parto: %s\n🍫 Desejo: %s\n\n%s%s",
		fmt.Sprintf(scenes[rand.IntN(len(scenes))], from.User, to.User),
		babies,
		sex,
		dueDate.Format("02/01/2006"),
		cravings[rand.IntN(len(cravings))],
		reactions[rand.IntN(len(reactions))],
		news.caption(),
	)
	return b.sendGIF(ctx, evt.Info.Chat, "engravidar", caption, append([]types.JID{from, to}, news.betrayed...))
}

func pregnancyOutcome() (string, string) {
	single := []string{"menino 💙", "menina 💗", "surpresa, só no chá revelação 🎈", "o bebê ficou de costas no ultrassom 🙈"}
	multiple := []string{"todos meninos 💙", "todas meninas 💗", "um de cada, pra ninguém brigar 💙💗", "surpresa, só no chá revelação 🎈"}
	outcomes := []struct {
		chance int
		babies string
		sexes  []string
	}{
		{70, "1 bebê", single},
		{20, "gêmeos 👶👶", multiple},
		{8, "trigêmeos 👶👶👶", multiple},
		{2, "sêxtuplos, boa sorte com as fraldas 👶👶👶👶👶👶", multiple},
	}
	roll := rand.IntN(100)
	for _, o := range outcomes {
		if roll < o.chance {
			return o.babies, o.sexes[rand.IntN(len(o.sexes))]
		}
		roll -= o.chance
	}
	return outcomes[0].babies, single[rand.IntN(len(single))]
}
