package games

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

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	guessMaxWords = 8
	GuessBusy     = "⏳ Já tem uma rodada de adivinhar rolando aqui, espera ela acabar"
)

type GuessHint struct {
	After time.Duration
	Text  string
}

type GuessRound struct {
	matches  func(guess string) bool
	excluded []string
	solved   chan struct{}
	Started  time.Time
	Keeper   roundKeeper

	mu      sync.Mutex
	players map[string]bool
	winner  types.JID
	over    bool
}

type GuessProgress struct {
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

func (k *roundKeeper) Keep() {
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

func (b *bot) KeepRound(ctx context.Context, k *roundKeeper, kind, key string, expires time.Time, state func() any) {
	k.save = func() { b.KeepGame(ctx, kind, key, state(), expires) }
	k.drop = func() { b.DropGame(ctx, kind, key) }
}

func NewGuessRound(matches func(guess string) bool, excluded []string) *GuessRound {
	return &GuessRound{matches: matches, excluded: excluded, solved: make(chan struct{}), Started: time.Now(), players: map[string]bool{}}
}

func RestoreGuessRound(matches func(guess string) bool, excluded []string, progress GuessProgress) *GuessRound {
	round := NewGuessRound(matches, excluded)
	round.Started = progress.Started
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

func (b *bot) ResumeGuess(saved core.SavedGame, round *GuessRound) (types.JID, error) {
	chat, err := types.ParseJID(saved.Key)
	if err != nil {
		return types.JID{}, fmt.Errorf("ler conversa da rodada: %w", err)
	}
	if !b.StartGuessRound(chat, round) {
		return types.JID{}, errors.New("já tem outra rodada de adivinhar nessa conversa")
	}
	return chat, nil
}

func (b *bot) AbandonGuess(chat types.JID, round *GuessRound) {
	round.end()
	b.guessRounds.CompareAndDelete(chat.ToNonAD().String(), round)
}

func (b *bot) StartGuessRound(chat types.JID, round *GuessRound) bool {
	_, running := b.guessRounds.LoadOrStore(chat.ToNonAD().String(), round)
	return !running
}

func (b *bot) GuessRunning(chat types.JID) bool {
	_, running := b.guessRounds.Load(chat.ToNonAD().String())
	return running
}

func (b *bot) answerGuess(evt *events.Message) {
	value, ok := b.guessRounds.Load(evt.Info.Chat.ToNonAD().String())
	if !ok {
		return
	}
	text := strings.TrimSpace(core.MessageText(evt.Message))
	if text == "" || strings.HasPrefix(text, "!") || len(strings.Fields(text)) > guessMaxWords {
		return
	}
	round := value.(*GuessRound)
	if core.SentByAny(evt.Info, round.excluded) {
		return
	}
	round.try(evt.Info.Sender.ToNonAD(), NormalizeGuess(text))
}

func (b *bot) AwaitGuess(ctx context.Context, chat types.JID, round *GuessRound, duration time.Duration, hints []GuessHint) error {
	defer b.guessRounds.CompareAndDelete(chat.ToNonAD().String(), round)
	solved, err := b.sendGuessHints(ctx, chat, round, hints)
	if err == nil && !solved {
		_, err = round.wait(ctx, duration-time.Since(round.Started))
	}
	if err != nil {
		return err
	}
	if !round.end() {
		return errors.New("rodada já encerrada")
	}
	return nil
}

func (b *bot) sendGuessHints(ctx context.Context, chat types.JID, round *GuessRound, hints []GuessHint) (bool, error) {
	pending := slices.DeleteFunc(slices.Clone(hints), func(h GuessHint) bool { return h.After <= time.Since(round.Started) })
	for _, hint := range pending {
		solved, err := round.wait(ctx, hint.After-time.Since(round.Started))
		if err != nil || solved {
			return solved, err
		}
		if err := b.SendText(ctx, chat, hint.Text); err != nil {
			b.Client.Log.Warnf("mandar dica em %s: %v", chat, err)
		}
	}
	return false, nil
}

func (r *GuessRound) wait(ctx context.Context, d time.Duration) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case <-r.solved:
		return true, nil
	case <-time.After(d):
		return false, nil
	}
}

func (r *GuessRound) try(player types.JID, guess string) {
	if r.record(player, guess) {
		r.Keeper.Keep()
	}
}

func (r *GuessRound) record(player types.JID, guess string) bool {
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

func (r *GuessRound) end() bool {
	r.mu.Lock()
	r.over = true
	r.mu.Unlock()
	return r.Keeper.discard()
}

func (r *GuessRound) Progress() GuessProgress {
	r.mu.Lock()
	defer r.mu.Unlock()
	return GuessProgress{Started: r.Started, Players: slices.Collect(maps.Keys(r.players)), Winner: r.winner}
}

func (r *GuessRound) Outcome() (winner types.JID, solved bool, players int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.winner, r.winner.User != "", len(r.players)
}

func NormalizeGuess(text string) string {
	words := strings.FieldsFunc(core.FoldLetters(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	return strings.Join(words, " ")
}
