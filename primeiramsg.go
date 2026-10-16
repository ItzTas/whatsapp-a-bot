package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) primeiramsg(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := firstMentionOrSender(evt).ToNonAD()
	first, found, err := b.messages.firstMessageSince(ctx, chat, b.userAliases(ctx, person), windowStart())
	if err != nil {
		return fmt.Errorf("buscar primeira mensagem: %w", err)
	}

	note := fmt.Sprintf("_Só guardo as mensagens dos %s, então pode ter coisa mais antiga que eu não vi_", messageWindowLabel)
	if !found {
		_, err := b.sendMentions(ctx, chat, fmt.Sprintf("Não tenho nenhuma mensagem de @%s guardada aqui 🤷\n\n%s", person.User, note), []types.JID{person})
		return err
	}
	text := fmt.Sprintf("📜 *Primeira mensagem de @%s* que eu tenho guardada, de %s às %s:\n\n%s\n\n%s",
		person.User, brazilianDate(first.sentAt), first.sentAt.In(brazilTime).Format("15:04"), quoteBlock(first.text), note,
	)
	_, err = b.sendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (s *messageStore) firstMessageSince(ctx context.Context, chat types.JID, users []string, since int64) (quote, bool, error) {
	filter, filterArgs := statsSenderFilter(users)
	var (
		text   string
		sentAt int64
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT texto, enviada_em FROM mensagens
		 WHERE chat = ? AND enviada_em >= ? AND ltrim(texto) NOT LIKE '!%'`+filter+`
		 ORDER BY enviada_em LIMIT 1`,
		append([]any{chat.String(), since}, filterArgs...)...,
	).Scan(&text, &sentAt)
	if errors.Is(err, sql.ErrNoRows) {
		return quote{}, false, nil
	}
	if err != nil {
		return quote{}, false, err
	}
	return quote{text: text, sentAt: time.Unix(sentAt, 0)}, true, nil
}
