package stats

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) primeiramsg(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := core.FirstMentionOrSender(evt).ToNonAD()
	first, found, err := b.messages.firstMessageSince(ctx, chat, b.UserAliases(ctx, person), core.WindowStart())
	if err != nil {
		return fmt.Errorf("buscar primeira mensagem: %w", err)
	}

	note := fmt.Sprintf("_Só guardo as mensagens dos %s, então pode ter coisa mais antiga que eu não vi_", core.MessageWindowLabel)
	if !found {
		_, err := b.SendMentions(ctx, chat, fmt.Sprintf("Não tenho nenhuma mensagem de @%s guardada aqui 🤷\n\n%s", person.User, note), []types.JID{person})
		return err
	}
	text := fmt.Sprintf("📜 *Primeira mensagem de @%s* que eu tenho guardada, de %s às %s:\n\n%s\n\n%s",
		person.User, core.BrazilianDate(first.SentAt), first.SentAt.In(core.BrazilTime).Format("15:04"), core.QuoteBlock(first.Text), note,
	)
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (s *messageStore) firstMessageSince(ctx context.Context, chat types.JID, users []string, since int64) (core.Quote, bool, error) {
	filter, filterArgs := statsSenderFilter(users)
	var (
		text   string
		sentAt int64
	)
	err := s.DB.QueryRowContext(ctx,
		`SELECT texto, enviada_em FROM mensagens
		 WHERE chat = ? AND enviada_em >= ? AND ltrim(texto) NOT LIKE '!%'`+filter+`
		 ORDER BY enviada_em LIMIT 1`,
		append([]any{chat.String(), since}, filterArgs...)...,
	).Scan(&text, &sentAt)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Quote{}, false, nil
	}
	if err != nil {
		return core.Quote{}, false, err
	}
	return core.Quote{Text: text, SentAt: time.Unix(sentAt, 0)}, true, nil
}
