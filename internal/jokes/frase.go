package jokes

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	maxQuoteCandidates = 40
)

func (b *bot) frase(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := core.FirstMentionOrSender(evt)
	q, found, err := b.messages.randomQuote(ctx, chat, b.UserAliases(ctx, person))
	if err != nil {
		return fmt.Errorf("sortear frase: %w", err)
	}
	if !found {
		_, err := b.SendMentions(ctx, chat, fmt.Sprintf("Ainda não tenho nenhuma frase de @%s guardada aqui 🤷", person.User), []types.JID{person})
		return err
	}
	text := fmt.Sprintf("📜 Como diria @%s em %s:\n\n%s", person.User, core.BrazilianDate(q.SentAt), core.QuoteBlock(q.Text))
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (s *messageStore) randomQuote(ctx context.Context, chat types.JID, users []string) (core.Quote, bool, error) {
	candidates, err := s.quoteCandidates(ctx, chat, users)
	if err != nil {
		return core.Quote{}, false, err
	}
	if len(candidates) == 0 {
		return core.Quote{}, false, nil
	}
	if i := slices.IndexFunc(candidates, core.Quote.WorthQuoting); i >= 0 {
		return candidates[i], true, nil
	}
	return candidates[0], true, nil
}

func (s *messageStore) quoteCandidates(ctx context.Context, chat types.JID, users []string) (candidates []core.Quote, err error) {
	first, second := users[0], users[len(users)-1]
	rows, err := s.DB.QueryContext(ctx,
		`SELECT texto, enviada_em FROM mensagens
		 WHERE chat = ? AND enviada_em >= ?
		   AND (remetente IN (?, ?) OR remetente_alt IN (?, ?))
		   AND texto NOT LIKE '!%' AND length(texto) >= ?
		 ORDER BY RANDOM() LIMIT ?`,
		chat.String(), core.WindowStart(), first, second, first, second, core.MinQuoteLength, maxQuoteCandidates,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var (
			q      core.Quote
			sentAt int64
		)
		if err := rows.Scan(&q.Text, &sentAt); err != nil {
			return nil, err
		}
		q.SentAt = time.Unix(sentAt, 0)
		candidates = append(candidates, q)
	}
	return candidates, rows.Err()
}
