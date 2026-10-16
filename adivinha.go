package main

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	guessMaxWords = 8
	guessBusy     = "⏳ Já tem uma rodada de adivinhar rolando aqui, espera ela acabar"
)

type guessHint struct {
	after time.Duration
	text  string
}

type guessRound struct {
	matches  func(guess string) bool
	excluded []string
	solved   chan struct{}
	started  time.Time
	keeper   roundKeeper

	mu      sync.Mutex
	players map[string]bool
	winner  types.JID
	over    bool
}

type guessProgress struct {
	Started time.Time `json:"started"`
	Players []string  `json:"players"`
	Winner  types.JID `json:"winner"`
}

type roundKeeper struct {
	mu   sync.Mutex
	gone bool
	save func()
	drop func()
}

func (k *roundKeeper) keep() {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.gone || k.save == nil {
		return
	}
	k.save()
}

func (k *roundKeeper) discard() bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.gone {
		return false
	}
	k.gone = true
	if k.drop != nil {
		k.drop()
	}
	return true
}

func (b *bot) keepRound(ctx context.Context, k *roundKeeper, kind, key string, expires time.Time, state func() any) {
	k.save = func() { b.keepGame(ctx, kind, key, state(), expires) }
	k.drop = func() { b.dropGame(ctx, kind, key) }
}

func newGuessRound(matches func(guess string) bool, excluded []string) *guessRound {
	return &guessRound{matches: matches, excluded: excluded, solved: make(chan struct{}), started: time.Now(), players: map[string]bool{}}
}

func restoreGuessRound(matches func(guess string) bool, excluded []string, progress guessProgress) *guessRound {
	round := newGuessRound(matches, excluded)
	round.started = progress.Started
	for _, player := range progress.Players {
		round.players[player] = true
	}
	if progress.Winner.User == "" {
		return round
	}
	round.winner, round.over = progress.Winner, true
	close(round.solved)
	return round
}

func (b *bot) resumeGuess(saved savedGame, round *guessRound) (types.JID, error) {
	chat, err := types.ParseJID(saved.key)
	if err != nil {
		return types.JID{}, fmt.Errorf("ler conversa da rodada: %w", err)
	}
	if !b.startGuessRound(chat, round) {
		return types.JID{}, errors.New("já tem outra rodada de adivinhar nessa conversa")
	}
	return chat, nil
}

func (b *bot) abandonGuess(chat types.JID, round *guessRound) {
	round.end()
	b.guessRounds.CompareAndDelete(chat.ToNonAD().String(), round)
}

func (b *bot) startGuessRound(chat types.JID, round *guessRound) bool {
	_, running := b.guessRounds.LoadOrStore(chat.ToNonAD().String(), round)
	return !running
}

func (b *bot) guessRunning(chat types.JID) bool {
	_, running := b.guessRounds.Load(chat.ToNonAD().String())
	return running
}

func (b *bot) answerGuess(evt *events.Message) {
	value, ok := b.guessRounds.Load(evt.Info.Chat.ToNonAD().String())
	if !ok {
		return
	}
	text := strings.TrimSpace(messageText(evt.Message))
	if text == "" || strings.HasPrefix(text, "!") || len(strings.Fields(text)) > guessMaxWords {
		return
	}
	round := value.(*guessRound)
	if sentByAny(evt.Info, round.excluded) {
		return
	}
	round.try(evt.Info.Sender.ToNonAD(), normalizeGuess(text))
}

func (b *bot) awaitGuess(ctx context.Context, chat types.JID, round *guessRound, duration time.Duration, hints []guessHint) error {
	defer b.guessRounds.CompareAndDelete(chat.ToNonAD().String(), round)
	solved, err := b.sendGuessHints(ctx, chat, round, hints)
	if err == nil && !solved {
		_, err = round.wait(ctx, duration-time.Since(round.started))
	}
	if err != nil {
		return err
	}
	if !round.end() {
		return errors.New("rodada já encerrada")
	}
	return nil
}

func (b *bot) sendGuessHints(ctx context.Context, chat types.JID, round *guessRound, hints []guessHint) (bool, error) {
	pending := slices.DeleteFunc(slices.Clone(hints), func(h guessHint) bool { return h.after <= time.Since(round.started) })
	for _, hint := range pending {
		solved, err := round.wait(ctx, hint.after-time.Since(round.started))
		if err != nil || solved {
			return solved, err
		}
		if err := b.sendText(ctx, chat, hint.text); err != nil {
			b.client.Log.Warnf("mandar dica em %s: %v", chat, err)
		}
	}
	return false, nil
}

func (r *guessRound) wait(ctx context.Context, d time.Duration) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case <-r.solved:
		return true, nil
	case <-time.After(d):
		return false, nil
	}
}

func (r *guessRound) try(player types.JID, guess string) {
	if r.record(player, guess) {
		r.keeper.keep()
	}
}

func (r *guessRound) record(player types.JID, guess string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.over {
		return false
	}
	joined := !r.players[player.User]
	r.players[player.User] = true
	if !r.matches(guess) {
		return joined
	}
	r.over = true
	r.winner = player
	close(r.solved)
	return true
}

func (r *guessRound) end() bool {
	r.mu.Lock()
	r.over = true
	r.mu.Unlock()
	return r.keeper.discard()
}

func (r *guessRound) progress() guessProgress {
	r.mu.Lock()
	defer r.mu.Unlock()
	return guessProgress{Started: r.started, Players: slices.Collect(maps.Keys(r.players)), Winner: r.winner}
}

func (r *guessRound) outcome() (winner types.JID, solved bool, players int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.winner, r.winner.User != "", len(r.players)
}

func normalizeGuess(text string) string {
	words := strings.FieldsFunc(foldLetters(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	return strings.Join(words, " ")
}
