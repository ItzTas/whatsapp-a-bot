package economy

import (
	"context"
	"fmt"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) renunciar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	now := time.Now()
	p, found, err := b.Wallets.presidency(ctx, chat, now)
	if err != nil {
		return fmt.Errorf("ler presidência: %w", err)
	}
	if !found {
		return b.SendText(ctx, chat, "🏛️ Ninguém é presidente agora, então não tem cargo pra largar")
	}
	if p.president.Key != acc.Key {
		text := fmt.Sprintf("🚫 Só quem tá no cargo pode renunciar, e quem é presidente é @%s. Se quer tirar do cargo, manda !impeachment", p.president.JID.User)
		_, err := b.SendMentions(ctx, chat, text, []types.JID{p.president.JID})
		return err
	}
	resigned, err := b.Wallets.resign(ctx, chat, acc, now)
	if err != nil {
		return fmt.Errorf("renunciar: %w", err)
	}
	if !resigned {
		return b.SendText(ctx, chat, "🏛️ O mandato acabou bem agora, então nem deu tempo de renunciar")
	}
	text := fmt.Sprintf("🏳️ *Renúncia!* @%s largou a presidência e perdeu os poderes do cargo na hora. Manda !votar @pessoa pra abrir uma eleição nova\n\n🎲 %s", person.User, vacancyNote(now.Add(vacancyWindow), now))
	if len(p.decrees.inForce()) > 0 {
		text += "\n\n📜 Os decretos dele continuam valendo até alguém tomar posse, mas os impostos agora somem em vez de ir pra alguém. Manda !presidente pra ver quais são"
	}
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (s *walletStore) resign(ctx context.Context, chat types.JID, president core.Account, now time.Time) (bool, error) {
	return s.vacate(ctx, chat, now,
		"UPDATE presidencias SET mandato_ate = ?, renunciou = 1, "+tyrannyDecree.column+" = 0 WHERE chat = ? AND pessoa = ? AND mandato_ate > ?",
		now.Unix(), chat.String(), president.Key, now.Unix(),
	)
}
