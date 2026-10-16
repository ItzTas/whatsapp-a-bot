package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"

	"go.mau.fi/whatsmeow/types/events"
)

const (
	quizOptionCount  = 4
	quizMaxQuestion  = 250
	quizMaxOptionLen = 100
)

func (b *bot) quizia(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat
	if len(b.assistant.tiers) == 0 {
		return b.sendText(ctx, chat, "🧠 A IA tá desligada, ninguém configurou um modelo")
	}
	if !b.lockQuiz(chat) {
		return b.replyTo(ctx, evt, quizBusy)
	}

	q, failure := b.aiQuizQuestion(ctx, evt)
	if failure != "" {
		b.quizChats.Delete(chat)
		return b.replyTo(ctx, evt, failure)
	}
	return b.startQuiz(ctx, chat, "🧠 ", q, false)
}

func (b *bot) aiQuizQuestion(ctx context.Context, evt *events.Message) (quizQuestion, string) {
	chat := evt.Info.Chat
	prompt := quizPrompt(questionFrom(messageText(evt.Message)))
	stopThinking := b.thinking(ctx, evt)
	raw, err := b.assistant.ask(ctx, b.iaAsker(ctx, chat, evt.Info.Sender.User), func(int) string { return prompt })
	stopThinking()
	if err != nil {
		b.client.Log.Warnf("!quizia: %v", err)
		return quizQuestion{}, iaFailure(err)
	}
	q, err := parseQuiz(raw)
	if err != nil {
		b.client.Log.Warnf("!quizia: pergunta inválida da IA: %v: %q", err, raw)
		return quizQuestion{}, "😵 A IA inventou uma pergunta quebrada, tenta de novo"
	}
	return q, ""
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
