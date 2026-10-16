package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	quizRevealDelay = 45 * time.Second
	quizRevealLabel = "45 segundos"
	quizBusy        = "⏳ Já tem um quiz rolando aqui, espera a resposta dele sair"
)

type quizQuestion struct {
	Question    string   `json:"pergunta"`
	Options     []string `json:"opcoes"`
	Correct     int      `json:"correta"`
	Explanation string   `json:"explicacao"`
}

type quizRound struct {
	*pollAnswer
	answer      string
	explanation string
	paid        bool
	deadline    time.Time
	keeper      roundKeeper
}

type quizState struct {
	Chat        types.JID  `json:"chat"`
	Answer      string     `json:"answer"`
	Explanation string     `json:"explanation"`
	Paid        bool       `json:"paid"`
	Poll        storedPoll `json:"poll"`
}

type quizStore struct {
	db *sql.DB
}

func (b *bot) quiz(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat
	if !b.lockQuiz(chat) {
		return b.replyTo(ctx, evt, quizBusy)
	}
	q, err := b.quizzes.next(ctx, chat.ToNonAD())
	if err != nil {
		b.quizChats.Delete(chat)
		return fmt.Errorf("sortear pergunta do !quiz: %w", err)
	}
	return b.startQuiz(ctx, chat, "🎲 ", q, true)
}

func (b *bot) lockQuiz(chat types.JID) bool {
	_, running := b.quizChats.LoadOrStore(chat, struct{}{})
	return !running
}

func (b *bot) startQuiz(ctx context.Context, chat types.JID, icon string, q quizQuestion, paid bool) error {
	answer := q.Options[q.Correct]
	rand.Shuffle(len(q.Options), func(i, j int) { q.Options[i], q.Options[j] = q.Options[j], q.Options[i] })
	resp, err := b.client.SendMessage(ctx, chat, b.client.BuildPollCreation(icon+q.Question, q.Options, 1))
	if err != nil {
		b.quizChats.Delete(chat)
		return err
	}

	round := &quizRound{pollAnswer: newPollAnswer(answer), answer: answer, explanation: q.Explanation, paid: paid, deadline: time.Now().Add(quizRevealDelay)}
	b.keepQuiz(ctx, chat, resp.ID, round)
	b.quizRounds.Store(resp.ID, round)
	round.keeper.keep()
	go b.revealQuiz(ctx, chat, resp.ID, round)
	return b.sendText(ctx, chat, fmt.Sprintf("_Vota na enquete, a resposta sai em %s_ ⏳", quizRevealLabel))
}

func (r *quizRound) vote(voter types.JID, selected [][]byte) {
	r.pollAnswer.vote(voter, selected)
	r.keeper.keep()
}

func (b *bot) keepQuiz(ctx context.Context, chat types.JID, pollID types.MessageID, round *quizRound) {
	b.keepRound(ctx, &round.keeper, "quiz", pollID, round.deadline, func() any {
		return quizState{Chat: chat, Answer: round.answer, Explanation: round.explanation, Paid: round.paid, Poll: round.stored()}
	})
}

func (b *bot) restoreQuiz(ctx context.Context, saved savedGame) error {
	var state quizState
	if err := json.Unmarshal(saved.state, &state); err != nil {
		return fmt.Errorf("ler rodada do !quiz: %w", err)
	}
	round := &quizRound{
		pollAnswer:  restorePollAnswer(state.Poll),
		answer:      state.Answer,
		explanation: state.Explanation,
		paid:        state.Paid,
		deadline:    saved.expires,
	}
	b.keepQuiz(ctx, state.Chat, saved.key, round)
	if _, running := b.quizRounds.LoadOrStore(saved.key, round); running {
		return nil
	}
	b.quizChats.Store(state.Chat, struct{}{})
	go b.revealQuiz(ctx, state.Chat, saved.key, round)
	return nil
}

func (b *bot) revealQuiz(ctx context.Context, chat types.JID, pollID types.MessageID, round *quizRound) {
	defer b.quizChats.Delete(chat)
	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Until(round.deadline)):
	}
	b.quizRounds.Delete(pollID)
	if !round.keeper.discard() {
		return
	}

	voted, winners := round.result()
	lines := []string{fmt.Sprintf("🧠 *Resposta:* %s", round.answer)}
	if round.explanation != "" {
		lines = append(lines, "_"+round.explanation+"_")
	}
	lines = append(lines, "", quizOutcome(voted, winners))
	if reward := b.quizPrize(ctx, chat, round, winners); reward != "" {
		lines = append(lines, reward)
	}
	if _, err := b.sendMentions(ctx, chat, strings.Join(lines, "\n"), winners); err != nil {
		b.client.Log.Errorf("revelar quiz: %v", err)
	}
}

func (b *bot) quizPrize(ctx context.Context, chat types.JID, round *quizRound, winners []types.JID) string {
	if !round.paid {
		return ""
	}
	return b.reward(ctx, chat, winners, round.voters(), quizReward)
}

func quizOutcome(voted bool, winners []types.JID) string {
	if !voted {
		return "🦗 Ninguém votou"
	}
	if len(winners) == 0 {
		return "❌ Ninguém acertou 😂"
	}
	return "✅ Acertaram: " + joinMentions(winners)
}

func (s *quizStore) next(ctx context.Context, chat types.JID) (quizQuestion, error) {
	id, q, err := s.unused(ctx, chat)
	if errors.Is(err, sql.ErrNoRows) {
		if _, err := s.db.ExecContext(ctx, "DELETE FROM quiz_usadas WHERE chat = ?", chat.String()); err != nil {
			return quizQuestion{}, err
		}
		id, q, err = s.unused(ctx, chat)
	}
	if err != nil {
		return quizQuestion{}, err
	}
	_, err = s.db.ExecContext(ctx, "INSERT OR IGNORE INTO quiz_usadas (chat, pergunta) VALUES (?, ?)", chat.String(), id)
	return q, err
}

func (s *quizStore) unused(ctx context.Context, chat types.JID) (int64, quizQuestion, error) {
	var (
		id    int64
		q     quizQuestion
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
