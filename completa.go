package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	completaDuration   = time.Minute
	completaReward     = 40
	completaCandidates = 80
	completaMinWords   = 5
	completaMinLetters = 3
)

type completaState struct {
	Text     string        `json:"text"`
	SentAt   time.Time     `json:"sent_at"`
	Author   types.JID     `json:"author"`
	Word     string        `json:"word"`
	Excluded []string      `json:"excluded"`
	Progress guessProgress `json:"progress"`
}

type completaPuzzle struct {
	storedQuote
	author types.JID
	word   string
	masked string
}

func (b *bot) completa(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return b.sendText(ctx, chat, "O !completa só funciona em grupo")
	}
	if b.guessRunning(chat) {
		return b.replyTo(ctx, evt, guessBusy)
	}

	var (
		group  *types.GroupInfo
		quotes []storedQuote
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { group, err = b.client.GetGroupInfo(gctx, chat); return err })
	g.Go(func() (err error) {
		quotes, err = b.messages.completaQuotes(gctx, chat, completaCandidates)
		return err
	})
	if err := g.Wait(); err != nil {
		return fmt.Errorf("preparar !completa: %w", err)
	}

	puzzle, found := pickCompleta(quotes, b.memberOwners(group.Participants))
	if !found {
		return b.sendText(ctx, chat, "Não sobrou mensagem boa de ninguém daqui pra completar 🤷")
	}
	round := newGuessRound(completaMatcher(puzzle.word), b.userAliases(ctx, puzzle.author))
	b.keepCompleta(ctx, chat.String(), round, puzzle)
	if !b.startGuessRound(chat, round) {
		return b.replyTo(ctx, evt, guessBusy)
	}
	if err := b.messages.markCompletaUsed(ctx, chat, puzzle.id); err != nil {
		b.abandonGuess(chat, round)
		return fmt.Errorf("marcar mensagem do !completa como usada: %w", err)
	}

	intro := fmt.Sprintf("🧩 *Completa a frase!*\n\n%s\n\n_Manda a palavra que falta (%d letras). A resposta sai em 1 minuto_ ⏳",
		quoteBlock(puzzle.masked), utf8.RuneCountInString(puzzle.word))
	if err := b.sendText(ctx, chat, intro); err != nil {
		b.abandonGuess(chat, round)
		return err
	}
	round.keeper.keep()
	go b.finishCompleta(ctx, chat, round, puzzle)
	return nil
}

func completaMatcher(word string) func(guess string) bool {
	answer := normalizeGuess(word)
	return func(guess string) bool { return slices.Contains(strings.Fields(guess), answer) }
}

func (b *bot) keepCompleta(ctx context.Context, key string, round *guessRound, puzzle completaPuzzle) {
	state := completaState{Text: puzzle.text, SentAt: puzzle.sentAt, Author: puzzle.author, Word: puzzle.word, Excluded: round.excluded}
	b.keepRound(ctx, &round.keeper, "completa", key, round.started.Add(completaDuration), func() any {
		state.Progress = round.progress()
		return state
	})
}

func (b *bot) restoreCompleta(ctx context.Context, saved savedGame) error {
	var state completaState
	if err := json.Unmarshal(saved.state, &state); err != nil {
		return fmt.Errorf("ler rodada do !completa: %w", err)
	}
	puzzle := completaPuzzle{storedQuote: storedQuote{quote: quote{text: state.Text, sentAt: state.SentAt}}, author: state.Author, word: state.Word}
	round := restoreGuessRound(completaMatcher(state.Word), state.Excluded, state.Progress)
	b.keepCompleta(ctx, saved.key, round, puzzle)
	chat, err := b.resumeGuess(saved, round)
	if err != nil {
		return err
	}
	go b.finishCompleta(ctx, chat, round, puzzle)
	return nil
}

func (b *bot) finishCompleta(ctx context.Context, chat types.JID, round *guessRound, puzzle completaPuzzle) {
	if err := b.awaitGuess(ctx, chat, round, completaDuration, nil); err != nil {
		return
	}
	winner, solved, players := round.outcome()
	header := fmt.Sprintf("⏰ Ninguém acertou! A palavra era *%s*", puzzle.word)
	mentions := []types.JID{puzzle.author}
	if solved {
		header = fmt.Sprintf("🎉 @%s acertou! A palavra era *%s*", winner.User, puzzle.word)
		mentions = append(mentions, winner)
	}
	text := fmt.Sprintf("%s\n\n%s\n— @%s, em %s", header, quoteBlock(puzzle.text), puzzle.author.User, brazilianDate(puzzle.sentAt))
	if solved {
		text = appendLine(text, b.reward(ctx, chat, []types.JID{winner}, players, completaReward))
	}
	if _, err := b.sendMentions(ctx, chat, text, mentions); err != nil {
		b.client.Log.Errorf("revelar !completa: %v", err)
	}
}

func pickCompleta(quotes []storedQuote, owners memberOwners) (completaPuzzle, bool) {
	for _, q := range quotes {
		author, ok := owners.of(q.sender, q.senderAlt)
		if !ok || quoteNoise.MatchString(q.text) || utf8.RuneCountInString(q.text) > maxQuoteLength {
			continue
		}
		word, masked, ok := maskLastWord(q.text)
		if ok {
			return completaPuzzle{storedQuote: q, author: author, word: word, masked: masked}, true
		}
	}
	return completaPuzzle{}, false
}

func maskLastWord(text string) (word, masked string, ok bool) {
	tokens := strings.Fields(text)
	if len(tokens) < completaMinWords {
		return "", "", false
	}
	cores := make([]string, len(tokens))
	starts := make([]int, len(tokens))
	cursor := 0
	for i, token := range tokens {
		cores[i] = strings.TrimFunc(token, func(r rune) bool { return !unicode.IsLetter(r) })
		starts[i] = cursor + strings.Index(text[cursor:], token)
		cursor = starts[i] + len(token)
	}
	for i := len(cores) - 1; i >= 0; i-- {
		if !hideable(cores[i]) || appearsElsewhere(cores, i) {
			continue
		}
		at := starts[i] + strings.Index(tokens[i], cores[i])
		blank := strings.Repeat("▁", utf8.RuneCountInString(cores[i]))
		return cores[i], text[:at] + blank + text[at+len(cores[i]):], true
	}
	return "", "", false
}

func hideable(word string) bool {
	if utf8.RuneCountInString(word) < completaMinLetters || isLaughWord(foldLetters(word)) {
		return false
	}
	return !strings.ContainsFunc(word, func(r rune) bool { return !unicode.IsLetter(r) })
}

func appearsElsewhere(cores []string, i int) bool {
	folded := foldLetters(cores[i])
	for j, core := range cores {
		if j != i && foldLetters(core) == folded {
			return true
		}
	}
	return false
}

func (s *messageStore) completaQuotes(ctx context.Context, chat types.JID, limit int) (quotes []storedQuote, err error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT m.id, m.remetente, m.remetente_alt, m.texto, m.enviada_em
		 FROM mensagens m
		 WHERE m.chat = ? AND m.enviada_em >= ?
		   AND m.texto NOT LIKE '!%' AND length(m.texto) >= ?
		   AND NOT EXISTS (SELECT 1 FROM completa_usadas u WHERE u.chat = m.chat AND u.mensagem = m.id)
		 ORDER BY RANDOM() LIMIT ?`,
		chat.String(), windowStart(), minQuoteLength, limit,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var (
			q      storedQuote
			sentAt int64
		)
		if err := rows.Scan(&q.id, &q.sender, &q.senderAlt, &q.text, &sentAt); err != nil {
			return nil, err
		}
		q.sentAt = time.Unix(sentAt, 0)
		quotes = append(quotes, q)
	}
	return quotes, rows.Err()
}

func (s *messageStore) markCompletaUsed(ctx context.Context, chat types.JID, messageID string) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT OR IGNORE INTO completa_usadas (chat, mensagem) VALUES (?, ?)",
		chat.String(), messageID,
	)
	return err
}
