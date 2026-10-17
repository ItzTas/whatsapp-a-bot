package main

import (
	"context"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) renunciar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.account(ctx, person)
	now := time.Now()
	p, found, err := b.wallets.presidency(ctx, chat, now)
	if err != nil {
		return fmt.Errorf("ler presidência: %w", err)
	}
	if !found {
		return b.sendText(ctx, chat, "🏛️ Ninguém é presidente agora, então não tem cargo pra largar")
	}
	if p.president.key != acc.key {
		text := fmt.Sprintf("🚫 Só quem tá no cargo pode renunciar, e quem é presidente é @%s. Se quer tirar do cargo, manda !impeachment", p.president.jid.User)
		_, err := b.sendMentions(ctx, chat, text, []types.JID{p.president.jid})
		return err
	}
	resigned, err := b.wallets.resign(ctx, chat, acc, now)
	if err != nil {
		return fmt.Errorf("renunciar: %w", err)
	}
	if !resigned {
		return b.sendText(ctx, chat, "🏛️ O mandato acabou bem agora, então nem deu tempo de renunciar")
	}
	text := fmt.Sprintf("🏳️ *Renúncia!* @%s largou a presidência e perdeu os poderes do cargo na hora. Manda !votar @pessoa pra abrir uma eleição nova", person.User)
	if len(p.decrees.inForce()) > 0 {
		text += "\n\n📜 Os decretos dele continuam valendo até alguém tomar posse, mas os impostos agora somem em vez de ir pra alguém. Manda !presidente pra ver quais são"
	}
	_, err = b.sendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (s *walletStore) resign(ctx context.Context, chat types.JID, president account, now time.Time) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		"UPDATE presidencias SET mandato_ate = ?, renunciou = 1 WHERE chat = ? AND pessoa = ? AND mandato_ate > ?",
		now.Unix(), chat.String(), president.key, now.Unix(),
	)
	if err != nil {
		return false, err
	}
	resigned, err := res.RowsAffected()
	return resigned > 0, err
}
