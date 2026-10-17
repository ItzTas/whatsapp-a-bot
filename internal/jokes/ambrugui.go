package jokes

import (
	"context"
	"fmt"
	"math/rand/v2"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) ambrugui(ctx context.Context, evt *events.Message) error {
	paes := []string{
		"brioche tostado", "pão de queijo gigante", "pão francês amassado",
		"pão com gergelim", "pão dormido de ontem", "dois pães de forma",
	}
	carnes := []string{
		"smash duplo", "picanha", "frango empanado", "costela desfiada",
		"carne de soja", "uma carne que ninguém sabe de onde veio",
	}
	recheios := []string{
		"bacon", "cheddar derretido", "ovo frito", "abacaxi", "batata palha", "Nutella",
		"jujuba", "requeijão", "picles", "farofa", "vinagrete", "catupiry",
	}
	reacoes := []string{
		"😋 Comeu em três mordidas e já pediu outro",
		"🤤 Fechou os olhos e esqueceu que o mundo existe",
		"🤢 Deu uma mordida e devolveu pro chapeiro",
		"🥵 Sujou a cara inteira de molho e nem ligou",
		"😭 Chorou de tão bom que tava",
		"💸 Pagou R$ 89 e ainda achou barato",
		"🫃 Agora tá deitado no sofá sem conseguir levantar",
		"🚽 Daqui a meia hora vai sumir do grupo",
	}

	target := core.FirstMentionOrSender(evt)
	extras := rand.Perm(len(recheios))
	text := fmt.Sprintf("🍔 *@%s comeu um ambrugui!*\n\n🍞 Pão: %s\n🥩 Carne: %s\n🧀 Recheio: %s e %s\n\n%s",
		target.User,
		paes[rand.IntN(len(paes))],
		carnes[rand.IntN(len(carnes))],
		recheios[extras[0]],
		recheios[extras[1]],
		reacoes[rand.IntN(len(reacoes))],
	)
	_, err := b.SendMentions(ctx, evt.Info.Chat, text, []types.JID{target})
	return err
}
