package economy

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const targetHint = "Quem saiu do grupo também vale: manda o nome da pessoa, o número dela na economia (#12, que aparece no !saldo e no !ricos), o telefone (@5511999999999) ou responde uma mensagem ou um contato dela"

type target struct {
	person types.JID
	rest   string
}

func (t target) found() bool {
	return t.person.User != ""
}

func (b *bot) targetOf(ctx context.Context, evt *events.Message) (target, string) {
	text := core.MessageText(evt.Message)
	if mentions := core.MentionedJIDs(evt.Message); len(mentions) > 0 {
		return target{person: mentions[0].ToNonAD(), rest: text}, ""
	}
	if contact, found := core.QuotedContact(evt.Message); found {
		return target{person: contact, rest: text}, ""
	}
	quoted := evt.Message.GetExtendedTextMessage().GetContextInfo()
	replied, err := types.ParseJID(quoted.GetParticipant())
	if err == nil && replied.User != "" && !b.IsSelf(replied) {
		return target{person: replied.ToNonAD(), rest: text}, ""
	}
	if tag := core.MentionTagPattern.FindString(text); tag != "" {
		return target{person: types.NewJID(tag[1:], types.DefaultUserServer), rest: text}, ""
	}
	chat := evt.Info.Chat.ToNonAD()
	if aim, problem, typed := b.personNumbered(ctx, chat, text); typed {
		return aim, problem
	}
	return b.personNamed(ctx, chat, text)
}

func (b *bot) personNamed(ctx context.Context, chat types.JID, text string) (target, string) {
	fields := strings.Fields(text)
	if len(fields) < 2 {
		return target{}, ""
	}
	typed := strings.Fields(core.FoldLetters(strings.Join(fields[1:], " ")))
	longest := 0
	people := map[string]types.JID{}
	for user, label := range b.ChatNicknames(ctx, chat) {
		words := strings.Fields(core.FoldLetters(label))
		if len(words) == 0 || len(words) < longest || len(words) > len(typed) || !slices.Equal(typed[:len(words)], words) {
			continue
		}
		if len(words) > longest {
			longest, people = len(words), map[string]types.JID{}
		}
		person := b.userJID(ctx, user)
		people[b.Account(ctx, person).Key] = person
	}
	if len(people) == 0 {
		return target{}, fmt.Sprintf("Ninguém aqui tem o nome ou o apelido %s 🤷 Confere no !nome ou no !apelido, ou digita o número", strings.Join(fields[1:], " "))
	}
	if len(people) > 1 {
		return target{}, fmt.Sprintf("Tem mais de uma pessoa chamada %s por aqui 🤔 Marca, responde uma mensagem dela ou digita o número", strings.Join(fields[1:1+longest], " "))
	}
	return target{person: slices.Collect(maps.Values(people))[0], rest: strings.Join(slices.Concat(fields[:1], fields[1+longest:]), " ")}, ""
}

func (b *bot) userJID(ctx context.Context, user string) types.JID {
	if ids := b.Identities(ctx, types.NewJID(user, types.HiddenUserServer)); len(ids) > 1 {
		return ids[1].ToNonAD()
	}
	return types.NewJID(user, types.DefaultUserServer)
}
