package ai

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types/events"
)

const persecutionSituation = `<perseguicao>
O grupo mandou você ficar na cola de quem mandou a <pergunta>. Ela não necessariamente falou com você, é só uma mensagem que essa pessoa mandou no grupo. Se meta nela mesmo assim: comente, zoe ou responda, curto, como um amigo chato que não larga do pé.
</perseguicao>

`

func (b *bot) answerPersecuted(ctx context.Context, evt *events.Message) error {
	text := strings.TrimSpace(core.MessageText(evt.Message))
	if text == "" || evt.IsEdit || evt.Info.IsFromMe || !b.Assistant.Enabled() {
		return nil
	}
	chat := evt.Info.Chat.ToNonAD()
	found, err := b.persecuted.Has(ctx, chat, b.SenderIdentities(ctx, evt))
	if err != nil || !found {
		return err
	}
	on, err := b.ChatModeOn(ctx, chat, core.ModeAIMeddling)
	if err != nil || !on {
		return err
	}
	answer, mentions, err := b.iaAnswer(ctx, evt, b.Asker(ctx, chat, "perseguir:"+chat.String()), text, persecutionSituation)
	if err != nil {
		b.Client.Log.Warnf("!perseguir: responder %s em %s: %v", evt.Info.Sender, chat, err)
		return nil
	}
	return b.replyIA(ctx, evt, answer, mentions)
}

func (b *bot) perseguir(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	people := slices.DeleteFunc(core.MentionedJIDs(evt.Message), b.IsSelf)
	if len(people) == 0 {
		return b.SendText(ctx, chat, "🕵️ Marca quem a IA vai perseguir: !perseguir @pessoa (pra largar do pé, !perseguir parar @pessoa)")
	}
	change, results := b.persecuted.Add, map[bool]string{
		true:  "🕵️ A IA agora vai responder toda mensagem de @%s 👀",
		false: "A IA já tava na cola de @%s",
	}
	if args := strings.Fields(core.QuestionFrom(core.MessageText(evt.Message))); len(args) > 0 && strings.EqualFold(args[0], "parar") {
		change, results = b.persecuted.Remove, map[bool]string{
			true:  "😮‍💨 A IA largou do pé de @%s",
			false: "A IA nem tava perseguindo @%s",
		}
	}
	lines := make([]string, len(people))
	for i, person := range people {
		changed, err := change(ctx, chat, b.Identities(ctx, person.ToNonAD()))
		if err != nil {
			return err
		}
		lines[i] = fmt.Sprintf(results[changed], person.User)
	}
	_, err := b.SendMentions(ctx, chat, strings.Join(lines, "\n"), people)
	return err
}
