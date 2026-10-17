package ai

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types/events"
)

const unknownCommandSituation = `<comando>
Quem mandou a <pergunta> usou o comando !%s, que não existe no bot. Finja que ele existe: entenda pelo nome o que ele faria, use o que veio depois do nome como entrada e responda como se fosse o resultado do comando. Não diga que o comando não existe nem que está fingindo.
</comando>

`

func (b *bot) answerUnknownCommand(ctx context.Context, evt *events.Message, name string) error {
	if !commandWord(name) || !b.Assistant.Enabled() || slices.ContainsFunc(b.OwnerCommands(), func(c core.Command) bool { return c.Matches(name) }) {
		return nil
	}
	chat := evt.Info.Chat
	on, err := b.ChatModeOn(ctx, chat, core.ModeAIInvent)
	if err != nil || !on {
		return err
	}
	question := strings.TrimSpace(core.MessageText(evt.Message))
	answer, mentions, err := b.iaAnswer(ctx, evt, b.Asker(ctx, chat, evt.Info.Sender.User), question, fmt.Sprintf(unknownCommandSituation, name))
	if err != nil {
		b.Client.Log.Warnf("comando inexistente !%s: %v", name, err)
		return b.ReplyTo(ctx, evt, iaFailure(err))
	}
	return b.replyIA(ctx, evt, answer, mentions)
}

func commandWord(name string) bool {
	first, _ := utf8.DecodeRuneInString(name)
	return unicode.IsLetter(first)
}
