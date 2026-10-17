package jokes

import (
	"context"
	"fmt"
	"math/rand/v2"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/romance"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) morder(ctx context.Context, evt *events.Message) error {
	pair, ok := romance.CoupleFrom(evt.Info.Sender, core.MentionedJIDs(evt.Message))
	if !ok {
		return b.SendText(ctx, evt.Info.Chat, "Marca uma ou duas pessoas: !morder @pessoa ou !morder @pessoa1 @pessoa2")
	}

	cenas := []string{
		"🦷 @%s cravou os dentes em @%s",
		"🧛 @%s mordeu o pescoço de @%s igual vampiro",
		"🐶 @%s mordeu @%s igual cachorro com osso",
		"😼 @%s deu uma mordidinha de gato em @%s",
		"🍗 @%s confundiu @%s com uma coxinha e mordeu",
		"😈 @%s mordeu a orelha de @%s e saiu rindo",
		"🦈 @%s atacou @%s igual tubarão",
		"🫦 @%s deu uma mordida no lábio de @%s 👀",
	}
	lugares := []string{
		"no braço", "no pescoço", "na bochecha", "na orelha", "no dedo",
		"no ombro", "na bunda", "no nariz", "na canela", "no cangote",
	}
	reacoes := []string{
		"😳 Ficou a marca dos dentes",
		"🩸 Tirou sangue, alguém chama o SAMU",
		"🥴 Gostou mais do que deveria",
		"😭 Gritou tão alto que o vizinho ouviu",
		"🧄 Agora vai precisar de alho pra se proteger",
		"💉 Melhor tomar a antirrábica por garantia",
		"🙄 Nem sentiu, a pele é grossa",
		"🔥 O grupo inteiro ficou com ciúme",
	}

	from, to := pair[0], pair[1]
	caption := fmt.Sprintf("%s\n\n📍 Mordeu %s\n😬 Força da mordida: %d%%\n\n%s",
		fmt.Sprintf(cenas[rand.IntN(len(cenas))], from.User, to.User),
		lugares[rand.IntN(len(lugares))],
		rand.IntN(101),
		reacoes[rand.IntN(len(reacoes))],
	)
	return b.SendGIF(ctx, evt.Info.Chat, "morder", caption, []types.JID{from, to})
}
