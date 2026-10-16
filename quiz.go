package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	quizOptionCount  = 4
	quizRevealDelay  = 45 * time.Second
	quizRevealLabel  = "45 segundos"
	quizMaxQuestion  = 250
	quizMaxOptionLen = 100
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
}

func (b *bot) quiz(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat
	if len(b.assistant.tiers) == 0 {
		return b.sendText(ctx, chat, "🧠 A IA tá desligada, ninguém configurou um modelo")
	}

	prompt := quizPrompt(questionFrom(messageText(evt.Message)))
	stopTyping := b.typing(ctx, chat)
	raw, err := b.assistant.ask(ctx, evt.Info.Sender.User, func(int) string { return prompt })
	stopTyping()
	if err != nil {
		b.client.Log.Warnf("!quiz: %v", err)
		return b.replyTo(ctx, evt, iaFailure(err))
	}
	q, err := parseQuiz(raw)
	if err != nil {
		b.client.Log.Warnf("!quiz: pergunta inválida da IA: %v: %q", err, raw)
		return b.replyTo(ctx, evt, "😵 A IA inventou uma pergunta quebrada, tenta de novo")
	}

	answer := q.Options[q.Correct]
	rand.Shuffle(len(q.Options), func(i, j int) { q.Options[i], q.Options[j] = q.Options[j], q.Options[i] })
	resp, err := b.client.SendMessage(ctx, chat, b.client.BuildPollCreation("🧠 "+q.Question, q.Options, 1))
	if err != nil {
		return err
	}

	round := &quizRound{pollAnswer: newPollAnswer(answer), answer: answer, explanation: q.Explanation}
	b.quizRounds.Store(resp.ID, round)
	go b.revealQuiz(ctx, chat, resp.ID, round)
	return b.sendText(ctx, chat, fmt.Sprintf("_Vota na enquete, a resposta sai em %s_ ⏳", quizRevealLabel))
}

func quizPrompt(theme string) string {
	if theme == "" {
		themes := []string{
			"história do Brasil", "história geral", "geografia", "ciência", "corpo humano", "espaço",
			"cinema", "séries", "música brasileira", "música internacional", "futebol", "esportes",
			"videogames", "memes e internet", "comida", "animais", "tecnologia", "novelas brasileiras",
			"línguas e palavras", "curiosidades aleatórias", "mitologia", "desenhos animados",
		}
		theme = themes[rand.IntN(len(themes))]
	}
	difficulties := []string{"fácil", "média", "difícil"}
	return fmt.Sprintf(`Cria uma pergunta de quiz de dificuldade %s sobre: %s.
Precisa ter exatamente %d alternativas curtas, só uma certa, e as erradas têm que ser plausíveis.
Confira o fato antes: a resposta certa tem que ser verdade de verdade.
Responda só com um JSON, sem texto antes ou depois e sem bloco de código, neste formato:
{"pergunta": "...", "opcoes": ["...", "...", "...", "..."], "correta": 0, "explicacao": "uma frase curta e divertida explicando a resposta"}
O campo "correta" é o índice (começando em 0) da alternativa certa dentro de "opcoes".`,
		difficulties[rand.IntN(len(difficulties))], theme, quizOptionCount)
}

func parseQuiz(raw string) (quizQuestion, error) {
	start, end := strings.Index(raw, "{"), strings.LastIndex(raw, "}")
	if start < 0 || end < start {
		return quizQuestion{}, errors.New("sem JSON na resposta")
	}
	var q quizQuestion
	if err := json.Unmarshal([]byte(raw[start:end+1]), &q); err != nil {
		return quizQuestion{}, err
	}
	q.Question = strings.TrimSpace(q.Question)
	q.Explanation = strings.TrimSpace(q.Explanation)
	for i, o := range q.Options {
		q.Options[i] = strings.TrimSpace(o)
	}
	return q, q.validate()
}

func (q quizQuestion) validate() error {
	if q.Question == "" || len(q.Question) > quizMaxQuestion {
		return errors.New("pergunta vazia ou grande demais")
	}
	if len(q.Options) != quizOptionCount {
		return fmt.Errorf("%d alternativas em vez de %d", len(q.Options), quizOptionCount)
	}
	if q.Correct < 0 || q.Correct >= len(q.Options) {
		return fmt.Errorf("índice da certa fora das alternativas: %d", q.Correct)
	}
	if slices.ContainsFunc(q.Options, func(o string) bool { return o == "" || len(o) > quizMaxOptionLen }) {
		return errors.New("alternativa vazia ou grande demais")
	}
	unique := slices.Compact(slices.Sorted(slices.Values(q.Options)))
	if len(unique) != len(q.Options) {
		return errors.New("alternativas repetidas")
	}
	return nil
}

func (b *bot) revealQuiz(ctx context.Context, chat types.JID, pollID types.MessageID, round *quizRound) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(quizRevealDelay):
	}
	b.quizRounds.Delete(pollID)

	voted, winners := round.result()
	lines := []string{fmt.Sprintf("🧠 *Resposta:* %s", round.answer)}
	if round.explanation != "" {
		lines = append(lines, "_"+round.explanation+"_")
	}
	lines = append(lines, "", quizOutcome(voted, winners))
	if reward := b.rewardQuiz(ctx, chat, winners); reward != "" {
		lines = append(lines, reward)
	}
	if _, err := b.sendMentions(ctx, chat, strings.Join(lines, "\n"), winners); err != nil {
		b.client.Log.Errorf("revelar !quiz: %v", err)
	}
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
