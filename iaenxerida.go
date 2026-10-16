package main

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"go.mau.fi/whatsmeow/types/events"
)

const persecutionSituation = `<perseguicao>
O grupo mandou você ficar na cola de quem mandou a <pergunta>. Ela não necessariamente falou com você, é só uma mensagem que essa pessoa mandou no grupo. Se meta nela mesmo assim: comente, zoe ou responda, curto, como um amigo chato que não larga do pé.
</perseguicao>

`

func (b *bot) answerPersecuted(ctx context.Context, evt *events.Message) error {
	text := strings.TrimSpace(messageText(evt.Message))
	if text == "" || evt.IsEdit || evt.Info.IsFromMe || len(b.assistant.tiers) == 0 {
		return nil
	}
	chat := evt.Info.Chat.ToNonAD()
	found, err := b.persecuted.has(ctx, chat, b.senderIdentities(ctx, evt))
	if err != nil || !found {
		return err
	}
	on, err := b.chatModeOn(ctx, chat, modeAIMeddling)
	if err != nil || !on {
		return err
	}
	answer, mentions, err := b.iaAnswer(ctx, evt, b.iaAsker(ctx, chat, "perseguir:"+chat.String()), text, persecutionSituation)
	if err != nil {
		b.client.Log.Warnf("!perseguir: responder %s em %s: %v", evt.Info.Sender, chat, err)
		return nil
	}
	return b.replyIA(ctx, evt, answer, mentions)
}

func (b *bot) perseguir(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	people := slices.DeleteFunc(mentionedJIDs(evt.Message), b.isSelf)
	if len(people) == 0 {
		return b.sendText(ctx, chat, "🕵️ Marca quem a IA vai perseguir: !perseguir @pessoa (pra largar do pé, !perseguir parar @pessoa)")
	}
	change, results := b.persecuted.add, map[bool]string{
		true:  "🕵️ A IA agora vai responder toda mensagem de @%s 👀",
		false: "A IA já tava na cola de @%s",
	}
	if args := strings.Fields(questionFrom(messageText(evt.Message))); len(args) > 0 && strings.EqualFold(args[0], "parar") {
		change, results = b.persecuted.remove, map[bool]string{
			true:  "😮‍💨 A IA largou do pé de @%s",
			false: "A IA nem tava perseguindo @%s",
		}
	}
	lines := make([]string, len(people))
	for i, person := range people {
		changed, err := change(ctx, chat, b.identities(ctx, person.ToNonAD()))
		if err != nil {
			return err
		}
		lines[i] = fmt.Sprintf(results[changed], person.User)
	}
	_, err := b.sendMentions(ctx, chat, strings.Join(lines, "\n"), people)
	return err
}
