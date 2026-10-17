package games

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/economy"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	quizRevealDelay = 45 * time.Second
	quizRevealLabel = "45 segundos"
	QuizBusy        = "⏳ Já tem um quiz rolando aqui, espera a resposta dele sair"
)

type QuizQuestion struct {
	Question    string   `json:"pergunta"`
	Options     []string `json:"opcoes"`
	Correct     int      `json:"correta"`
	Explanation string   `json:"explicacao"`
}

type quizRound struct {
	*core.PollAnswer
	answer      string
	explanation string
	paid        bool
	deadline    time.Time
	keeper      roundKeeper
}

type quizState struct {
	Chat        types.JID       `json:"chat"`
	Answer      string          `json:"answer"`
	Explanation string          `json:"explanation"`
	Paid        bool            `json:"paid"`
	Poll        core.StoredPoll `json:"poll"`
}

type quizStore struct {
	db *sql.DB
}

func (b *bot) quiz(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat
	if !b.LockQuiz(chat) {
		return b.ReplyTo(ctx, evt, QuizBusy)
	}
	q, err := b.quizzes.next(ctx, chat.ToNonAD())
	if err != nil {
		b.QuizChats.Delete(chat)
		return fmt.Errorf("sortear pergunta do !quiz: %w", err)
	}
	return b.StartQuiz(ctx, chat, "🎲 ", q, true)
}

func (b *bot) LockQuiz(chat types.JID) bool {
	_, running := b.QuizChats.LoadOrStore(chat, struct{}{})
	return !running
}

func (b *bot) StartQuiz(ctx context.Context, chat types.JID, icon string, q QuizQuestion, paid bool) error {
	answer := q.Options[q.Correct]
	rand.Shuffle(len(q.Options), func(i, j int) { q.Options[i], q.Options[j] = q.Options[j], q.Options[i] })
	resp, err := b.Client.SendMessage(ctx, chat, b.Client.BuildPollCreation(icon+q.Question, q.Options, 1))
	if err != nil {
		b.QuizChats.Delete(chat)
		return err
	}

	round := &quizRound{PollAnswer: core.NewPollAnswer(answer), answer: answer, explanation: q.Explanation, paid: paid, deadline: time.Now().Add(quizRevealDelay)}
	b.keepQuiz(ctx, chat, resp.ID, round)
	b.quizRounds.Store(resp.ID, round)
	round.keeper.Keep()
	go b.revealQuiz(ctx, chat, resp.ID, round)
	return b.SendText(ctx, chat, fmt.Sprintf("_Vota na enquete, a resposta sai em %s_ ⏳", quizRevealLabel))
}

func (r *quizRound) Vote(voter types.JID, selected [][]byte) {
	r.PollAnswer.Vote(voter, selected)
	r.keeper.Keep()
}

func (b *bot) keepQuiz(ctx context.Context, chat types.JID, pollID types.MessageID, round *quizRound) {
	b.KeepRound(ctx, &round.keeper, "quiz", pollID, round.deadline, func() any {
		return quizState{Chat: chat, Answer: round.answer, Explanation: round.explanation, Paid: round.paid, Poll: round.Stored()}
	})
}

func (b *bot) restoreQuiz(ctx context.Context, saved core.SavedGame) error {
	var state quizState
	if err := json.Unmarshal(saved.State, &state); err != nil {
		return fmt.Errorf("ler rodada do !quiz: %w", err)
	}
	round := &quizRound{
		PollAnswer:  core.RestorePollAnswer(state.Poll),
		answer:      state.Answer,
		explanation: state.Explanation,
		paid:        state.Paid,
		deadline:    saved.Expires,
	}
	b.keepQuiz(ctx, state.Chat, saved.Key, round)
	if _, running := b.quizRounds.LoadOrStore(saved.Key, round); running {
		return nil
	}
	b.QuizChats.Store(state.Chat, struct{}{})
	go b.revealQuiz(ctx, state.Chat, saved.Key, round)
	return nil
}

func (b *bot) revealQuiz(ctx context.Context, chat types.JID, pollID types.MessageID, round *quizRound) {
	defer b.QuizChats.Delete(chat)
	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Until(round.deadline)):
	}
	b.quizRounds.Delete(pollID)
	if !round.keeper.discard() {
		return
	}

	voted, winners := round.Result()
	lines := []string{fmt.Sprintf("🧠 *Resposta:* %s", round.answer)}
	if round.explanation != "" {
		lines = append(lines, "_"+round.explanation+"_")
	}
	lines = append(lines, "", quizOutcome(voted, winners))
	if reward := b.quizPrize(ctx, chat, round, winners); reward != "" {
		lines = append(lines, reward)
	}
	if _, err := b.SendMentions(ctx, chat, strings.Join(lines, "\n"), winners); err != nil {
		b.Client.Log.Errorf("revelar quiz: %v", err)
	}
}

func (b *bot) quizPrize(ctx context.Context, chat types.JID, round *quizRound, winners []types.JID) string {
	if !round.paid {
		return ""
	}
	return b.economy.Reward(ctx, chat, winners, round.Voters(), economy.QuizReward)
}

func quizOutcome(voted bool, winners []types.JID) string {
	if !voted {
		return "🦗 Ninguém votou"
	}
	if len(winners) == 0 {
		return "❌ Ninguém acertou 😂"
	}
	return "✅ Acertaram: " + core.JoinMentions(winners)
}

func (s *quizStore) next(ctx context.Context, chat types.JID) (QuizQuestion, error) {
	id, q, err := s.unused(ctx, chat)
	if errors.Is(err, sql.ErrNoRows) {
		if _, err := s.db.ExecContext(ctx, "DELETE FROM quiz_usadas WHERE chat = ?", chat.String()); err != nil {
			return QuizQuestion{}, err
		}
		id, q, err = s.unused(ctx, chat)
	}
	if err != nil {
		return QuizQuestion{}, err
	}
	_, err = s.db.ExecContext(ctx, "INSERT OR IGNORE INTO quiz_usadas (chat, pergunta) VALUES (?, ?)", chat.String(), id)
	return q, err
}

func (s *quizStore) unused(ctx context.Context, chat types.JID) (int64, QuizQuestion, error) {
	var (
		id    int64
		q     QuizQuestion
		right string
		wrong [3]string
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT p.id, p.pergunta, p.certa, p.errada1, p.errada2, p.errada3, p.explicacao
		 FROM quiz_perguntas p
		 WHERE NOT EXISTS (SELECT 1 FROM quiz_usadas u WHERE u.chat = ? AND u.pergunta = p.id)
		 ORDER BY RANDOM() LIMIT 1`,
		chat.String(),
	).Scan(&id, &q.Question, &right, &wrong[0], &wrong[1], &wrong[2], &q.Explanation)
	q.Options = []string{right, wrong[0], wrong[1], wrong[2]}
	return id, q, err
}
