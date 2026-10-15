package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const minQuoteLength = 12

var brazilTime = time.FixedZone("BRT", -3*60*60)

type quote struct {
	text   string
	sentAt time.Time
}

func (b *bot) frase(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := firstMentionOrSender(evt)
	q, found, err := b.messages.randomQuote(ctx, chat, b.userAliases(ctx, person))
	if err != nil {
		return fmt.Errorf("sortear frase: %w", err)
	}
	if !found {
		_, err := b.sendMentions(ctx, chat, fmt.Sprintf("Ainda não tenho nenhuma frase de @%s guardada aqui 🤷", person.User), []types.JID{person})
		return err
	}
	text := fmt.Sprintf("📜 Como diria @%s em %s:\n\n%s", person.User, brazilianDate(q.sentAt), quoteBlock(q.text))
	_, err = b.sendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func quoteBlock(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	for i, line := range lines {
		lines[i] = "> " + line
	}
	return strings.Join(lines, "\n")
}

func brazilianDate(t time.Time) string {
	months := []string{
		"janeiro", "fevereiro", "março", "abril", "maio", "junho",
		"julho", "agosto", "setembro", "outubro", "novembro", "dezembro",
	}
	local := t.In(brazilTime)
	date := fmt.Sprintf("%d de %s", local.Day(), months[local.Month()-1])
	if local.Year() == time.Now().In(brazilTime).Year() {
		return date
	}
	return fmt.Sprintf("%s de %d", date, local.Year())
}

func (s *messageStore) randomQuote(ctx context.Context, chat types.JID, users []string) (quote, bool, error) {
	first, second := users[0], users[len(users)-1]
	var (
		q      quote
		sentAt int64
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT texto, enviada_em FROM mensagens
		 WHERE chat = ? AND enviada_em >= ?
		   AND (remetente IN (?, ?) OR remetente_alt IN (?, ?))
		   AND texto NOT LIKE '!%' AND length(texto) >= ?
		 ORDER BY RANDOM() LIMIT 1`,
		chat.String(), windowStart(), first, second, first, second, minQuoteLength,
	).Scan(&q.text, &sentAt)
	if errors.Is(err, sql.ErrNoRows) {
		return quote{}, false, nil
	}
	if err != nil {
		return quote{}, false, err
	}
	q.sentAt = time.Unix(sentAt, 0)
	return q, true, nil
}
