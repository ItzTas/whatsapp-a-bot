package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/games"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) quizia(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat
	if !b.Assistant.Enabled() {
		return b.SendText(ctx, chat, "🧠 A IA tá desligada, ninguém configurou um modelo")
	}
	if !b.games.LockQuiz(chat) {
		return b.ReplyTo(ctx, evt, games.QuizBusy)
	}

	q, failure := b.aiQuizQuestion(ctx, evt)
	if failure != "" {
		b.games.QuizChats.Delete(chat)
		return b.ReplyTo(ctx, evt, failure)
	}
	return b.games.StartQuiz(ctx, chat, "🧠 ", q, false)
}

func (b *bot) aiQuizQuestion(ctx context.Context, evt *events.Message) (games.QuizQuestion, string) {
	chat := evt.Info.Chat
	prompt := quizPrompt(core.QuestionFrom(core.MessageText(evt.Message)))
	stopThinking := b.thinking(ctx, evt)
	raw, err := b.Assistant.Ask(ctx, b.Asker(ctx, chat, evt.Info.Sender.User), func(int) string { return prompt })
	stopThinking()
	if err != nil {
		b.Client.Log.Warnf("!quizia: %v", err)
		return games.QuizQuestion{}, iaFailure(err)
	}
	q, err := parseQuiz(raw)
	if err != nil {
		b.Client.Log.Warnf("!quizia: pergunta inválida da IA: %v: %q", err, raw)
		return games.QuizQuestion{}, "😵 A IA inventou uma pergunta quebrada, tenta de novo"
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
		difficulties[rand.IntN(len(difficulties))], theme, games.QuizOptionCount)
}

func parseQuiz(raw string) (games.QuizQuestion, error) {
	start, end := strings.Index(raw, "{"), strings.LastIndex(raw, "}")
	if start < 0 || end < start {
		return games.QuizQuestion{}, errors.New("sem JSON na resposta")
	}
	var q games.QuizQuestion
	if err := json.Unmarshal([]byte(raw[start:end+1]), &q); err != nil {
		return games.QuizQuestion{}, err
	}
	q.Question = strings.TrimSpace(q.Question)
	q.Explanation = strings.TrimSpace(q.Explanation)
	for i, o := range q.Options {
		q.Options[i] = strings.TrimSpace(o)
	}
	return q, q.Validate()
}
