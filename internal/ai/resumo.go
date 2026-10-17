package ai

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	summaryDefaultMessages = 100
	summaryMinMessages     = 10
	summaryMaxMessages     = 500
	summaryRequest         = "resume o que rolou nas mensagens recentes do grupo: os assuntos principais, quem disse o quê de importante e se ficou algum combinado. Pode usar lista com hífen e passar dos três parágrafos se precisar."
)

func (b *bot) resumo(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat
	if !b.Assistant.Enabled() {
		return b.SendText(ctx, chat, "🧠 A IA tá desligada, ninguém configurou um modelo")
	}
	stored, err := b.messages.recent(ctx, chat, evt.Info.ID, 1)
	if err != nil {
		return fmt.Errorf("ler mensagens recentes de %s: %w", chat, err)
	}
	if len(stored) == 0 {
		return b.SendText(ctx, chat, "📝 Ainda não tem mensagem guardada nesta conversa pra resumir")
	}

	size := summarySize(core.QuestionFrom(core.MessageText(evt.Message)))
	stopThinking := b.thinking(ctx, evt)
	summary, err := b.Assistant.Ask(ctx, b.Asker(ctx, chat, evt.Info.Sender.User), func(int) string {
		return b.iaPrompt(ctx, evt, summaryRequest, "", size)
	})
	stopThinking()
	if err != nil {
		b.Client.Log.Warnf("!resumo: %v", err)
		summary = iaFailure(err)
	}
	return b.ReplyTo(ctx, evt, summary)
}

func summarySize(arg string) int {
	n, err := strconv.Atoi(strings.TrimSpace(arg))
	if err != nil {
		return summaryDefaultMessages
	}
	return min(max(n, summaryMinMessages), summaryMaxMessages)
}
