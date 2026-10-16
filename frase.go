package main

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	minQuoteLength     = 12
	maxQuoteLength     = 300
	minQuoteWords      = 3
	maxQuoteCandidates = 40
)

var (
	brazilTime = time.FixedZone("BRT", -3*60*60)
	quoteNoise = regexp.MustCompile(`https?://\S+|www\.\S+|@\d+`)
)

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
	candidates, err := s.quoteCandidates(ctx, chat, users)
	if err != nil {
		return quote{}, false, err
	}
	if len(candidates) == 0 {
		return quote{}, false, nil
	}
	if i := slices.IndexFunc(candidates, quote.worthQuoting); i >= 0 {
		return candidates[i], true, nil
	}
	return candidates[0], true, nil
}

func (s *messageStore) quoteCandidates(ctx context.Context, chat types.JID, users []string) (candidates []quote, err error) {
	first, second := users[0], users[len(users)-1]
	rows, err := s.db.QueryContext(ctx,
		`SELECT texto, enviada_em FROM mensagens
		 WHERE chat = ? AND enviada_em >= ?
		   AND (remetente IN (?, ?) OR remetente_alt IN (?, ?))
		   AND texto NOT LIKE '!%' AND length(texto) >= ?
		 ORDER BY RANDOM() LIMIT ?`,
		chat.String(), windowStart(), first, second, first, second, minQuoteLength, maxQuoteCandidates,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var (
			q      quote
			sentAt int64
		)
		if err := rows.Scan(&q.text, &sentAt); err != nil {
			return nil, err
		}
		q.sentAt = time.Unix(sentAt, 0)
		candidates = append(candidates, q)
	}
	return candidates, rows.Err()
}

func (q quote) worthQuoting() bool {
	if utf8.RuneCountInString(q.text) > maxQuoteLength {
		return false
	}
	return len(meaningfulWords(q.text)) >= minQuoteWords
}

func meaningfulWords(text string) []string {
	filler := map[string]bool{
		"oi": true, "ola": true, "opa": true, "eae": true, "eai": true, "salve": true,
		"bom": true, "boa": true, "dia": true, "tarde": true, "noite": true,
		"ok": true, "okay": true, "blz": true, "beleza": true, "sim": true, "nao": true,
		"s": true, "n": true, "ss": true, "nn": true, "kk": true, "e": true, "ne": true,
		"ta": true, "to": true, "vlw": true, "valeu": true, "obg": true,
		"obrigado": true, "obrigada": true, "tmj": true,
	}
	stripped := quoteNoise.ReplaceAllString(foldLetters(text), " ")
	words := strings.FieldsFunc(stripped, func(r rune) bool { return !unicode.IsLetter(r) })
	return slices.DeleteFunc(words, func(w string) bool { return filler[w] || isLaughWord(w) })
}
