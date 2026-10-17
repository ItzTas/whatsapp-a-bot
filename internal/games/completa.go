package games

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

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
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
	Progress GuessProgress `json:"progress"`
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
		return b.SendText(ctx, chat, "O !completa só funciona em grupo")
	}
	if b.GuessRunning(chat) {
		return b.ReplyTo(ctx, evt, GuessBusy)
	}

	var (
		group  *types.GroupInfo
		quotes []storedQuote
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { group, err = b.Client.GetGroupInfo(gctx, chat); return err })
	g.Go(func() (err error) {
		quotes, err = b.messages.completaQuotes(gctx, chat, completaCandidates)
		return err
	})
	if err := g.Wait(); err != nil {
		return fmt.Errorf("preparar !completa: %w", err)
	}

	puzzle, found := pickCompleta(quotes, b.MemberOwners(group.Participants))
	if !found {
		return b.SendText(ctx, chat, "Não sobrou mensagem boa de ninguém daqui pra completar 🤷")
	}
	round := NewGuessRound(completaMatcher(puzzle.word), b.UserAliases(ctx, puzzle.author))
	b.keepCompleta(ctx, chat.String(), round, puzzle)
	if !b.StartGuessRound(chat, round) {
		return b.ReplyTo(ctx, evt, GuessBusy)
	}
	if err := b.messages.markCompletaUsed(ctx, chat, puzzle.id); err != nil {
		b.AbandonGuess(chat, round)
		return fmt.Errorf("marcar mensagem do !completa como usada: %w", err)
	}

	intro := fmt.Sprintf("🧩 *Completa a frase!*\n\n%s\n\n_Manda a palavra que falta (%d letras). A resposta sai em 1 minuto_ ⏳",
		core.QuoteBlock(puzzle.masked), utf8.RuneCountInString(puzzle.word))
	if err := b.SendText(ctx, chat, intro); err != nil {
		b.AbandonGuess(chat, round)
		return err
	}
	round.Keeper.Keep()
	go b.finishCompleta(ctx, chat, round, puzzle)
	return nil
}

func completaMatcher(word string) func(guess string) bool {
	answer := NormalizeGuess(word)
	return func(guess string) bool { return slices.Contains(strings.Fields(guess), answer) }
}

func (b *bot) keepCompleta(ctx context.Context, key string, round *GuessRound, puzzle completaPuzzle) {
	state := completaState{Text: puzzle.Text, SentAt: puzzle.SentAt, Author: puzzle.author, Word: puzzle.word, Excluded: round.excluded}
	b.KeepRound(ctx, &round.Keeper, "completa", key, round.Started.Add(completaDuration), func() any {
		state.Progress = round.Progress()
		return state
	})
}

func (b *bot) restoreCompleta(ctx context.Context, saved core.SavedGame) error {
	var state completaState
	if err := json.Unmarshal(saved.State, &state); err != nil {
		return fmt.Errorf("ler rodada do !completa: %w", err)
	}
	puzzle := completaPuzzle{storedQuote: storedQuote{Quote: core.Quote{Text: state.Text, SentAt: state.SentAt}}, author: state.Author, word: state.Word}
	round := RestoreGuessRound(completaMatcher(state.Word), state.Excluded, state.Progress)
	b.keepCompleta(ctx, saved.Key, round, puzzle)
	chat, err := b.ResumeGuess(saved, round)
	if err != nil {
		return err
	}
	go b.finishCompleta(ctx, chat, round, puzzle)
	return nil
}

func (b *bot) finishCompleta(ctx context.Context, chat types.JID, round *GuessRound, puzzle completaPuzzle) {
	if err := b.AwaitGuess(ctx, chat, round, completaDuration, nil); err != nil {
		return
	}
	winner, solved, players := round.Outcome()
	header := fmt.Sprintf("⏰ Ninguém acertou! A palavra era *%s*", puzzle.word)
	mentions := []types.JID{puzzle.author}
	if solved {
		header = fmt.Sprintf("🎉 @%s acertou! A palavra era *%s*", winner.User, puzzle.word)
		mentions = append(mentions, winner)
	}
	text := fmt.Sprintf("%s\n\n%s\n— @%s, em %s", header, core.QuoteBlock(puzzle.Text), puzzle.author.User, core.BrazilianDate(puzzle.SentAt))
	if solved {
		text = core.AppendLine(text, b.economy.Reward(ctx, chat, []types.JID{winner}, players, completaReward))
	}
	if _, err := b.SendMentions(ctx, chat, text, mentions); err != nil {
		b.Client.Log.Errorf("revelar !completa: %v", err)
	}
}

func pickCompleta(quotes []storedQuote, owners core.MemberOwners) (completaPuzzle, bool) {
	for _, q := range quotes {
		author, ok := owners.Of(q.sender, q.senderAlt)
		if !ok || core.QuoteNoise.MatchString(q.Text) || utf8.RuneCountInString(q.Text) > core.MaxQuoteLength {
			continue
		}
		word, masked, ok := maskLastWord(q.Text)
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
	if utf8.RuneCountInString(word) < completaMinLetters || core.IsLaughWord(core.FoldLetters(word)) {
		return false
	}
	return !strings.ContainsFunc(word, func(r rune) bool { return !unicode.IsLetter(r) })
}

func appearsElsewhere(cores []string, i int) bool {
	folded := core.FoldLetters(cores[i])
	for j, stem := range cores {
		if j != i && core.FoldLetters(stem) == folded {
			return true
		}
	}
	return false
}

func (s *messageStore) completaQuotes(ctx context.Context, chat types.JID, limit int) (quotes []storedQuote, err error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT m.id, m.remetente, m.remetente_alt, m.texto, m.enviada_em
		 FROM mensagens m
		 WHERE m.chat = ? AND m.enviada_em >= ?
		   AND m.texto NOT LIKE '!%' AND length(m.texto) >= ?
		   AND NOT EXISTS (SELECT 1 FROM completa_usadas u WHERE u.chat = m.chat AND u.mensagem = m.id)
		 ORDER BY RANDOM() LIMIT ?`,
		chat.String(), core.WindowStart(), core.MinQuoteLength, limit,
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
		if err := rows.Scan(&q.id, &q.sender, &q.senderAlt, &q.Text, &sentAt); err != nil {
			return nil, err
		}
		q.SentAt = time.Unix(sentAt, 0)
		quotes = append(quotes, q)
	}
	return quotes, rows.Err()
}

func (s *messageStore) markCompletaUsed(ctx context.Context, chat types.JID, messageID string) error {
	_, err := s.DB.ExecContext(ctx,
		"INSERT OR IGNORE INTO completa_usadas (chat, mensagem) VALUES (?, ?)",
		chat.String(), messageID,
	)
	return err
}
