package jokes

import (
	"context"
	"fmt"
	"math/rand/v2"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) ego(ctx context.Context, evt *events.Message) error {
	frases := map[bool][]string{
		true: {
			"🦚 +%[2]d de ego pra @%[1]s. O ego nem cabe mais no grupo",
			"👑 @%[1]s ganhou +%[2]d de ego e já tá se achando o dono do grupo",
			"🪞 +%[2]d ego. @%[1]s passou 3 horas se olhando no espelho",
			"🎈 O ego de @%[1]s inflou +%[2]d. Cuidado que estoura",
			"😤 @%[1]s com +%[2]d de ego: aura e ego no talo, ninguém aguenta",
		},
		false: {
			"🩹 -%[2]d de ego pra @%[1]s. Ego ferido, vai precisar de colo",
			"📉 @%[1]s perdeu %[2]d de ego. Foi humilhado em público",
			"🥀 -%[2]d ego. @%[1]s descobriu que não é tudo isso",
			"😶 O ego de @%[1]s caiu %[2]d. Desceu do salto",
			"💔 @%[1]s, -%[2]d de ego. Sem aura e sem ego, que fase",
		},
	}

	target := core.FirstMentionOrSender(evt)
	options := frases[rand.IntN(2) == 0]
	points := (rand.IntN(100) + 1) * 100
	text := fmt.Sprintf(options[rand.IntN(len(options))], target.User, points)
	_, err := b.SendMentions(ctx, evt.Info.Chat, text, []types.JID{target})
	return err
}
