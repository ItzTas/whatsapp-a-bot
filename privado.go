package main

import (
	"context"
	"slices"
	"strings"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

type privateAnswer func(ctx context.Context, evt *events.Message, text string) (done bool, err error)

type privatePrompt struct {
	keys   []string
	answer privateAnswer
}

func (b *bot) awaitPrivate(ids []types.JID, answer privateAnswer) (*privatePrompt, bool) {
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = id.ToNonAD().String()
	}
	prompt := &privatePrompt{keys: keys, answer: answer}
	for _, key := range keys {
		if _, taken := b.privatePrompts.LoadOrStore(key, prompt); taken {
			b.dropPrivate(prompt)
			return nil, false
		}
	}
	return prompt, true
}

func (b *bot) dropPrivate(prompt *privatePrompt) {
	for _, key := range prompt.keys {
		b.privatePrompts.CompareAndDelete(key, prompt)
	}
}

func (b *bot) answerPrivate(ctx context.Context, evt *events.Message) bool {
	chat := evt.Info.Chat
	if evt.Info.IsFromMe || (chat.Server != types.DefaultUserServer && chat.Server != types.HiddenUserServer) {
		return false
	}
	text := strings.TrimSpace(messageText(evt.Message))
	if text == "" || strings.HasPrefix(text, "!") {
		return false
	}
	prompt, pending := b.pendingPrivate(b.senderIdentities(ctx, evt))
	if !pending {
		return false
	}
	done, err := prompt.answer(ctx, evt, text)
	if err != nil {
		b.client.Log.Errorf("responder no privado de %s: %v", evt.Info.Sender, err)
	}
	if done {
		b.dropPrivate(prompt)
	}
	return true
}

func (b *bot) pendingPrivate(ids []types.JID) (*privatePrompt, bool) {
	for _, id := range ids {
		if value, ok := b.privatePrompts.Load(id.ToNonAD().String()); ok {
			return value.(*privatePrompt), true
		}
	}
	return nil, false
}

func (b *bot) sendPrivate(ctx context.Context, ids []types.JID, text string, mentions []types.JID) bool {
	inbox := privateInbox(ids)
	if _, err := b.sendRawMentions(ctx, inbox, text, mentions); err != nil {
		b.client.Log.Errorf("mandar mensagem no privado de %s: %v", inbox, err)
		return false
	}
	return true
}

func privateInbox(ids []types.JID) types.JID {
	i := slices.IndexFunc(ids, func(id types.JID) bool { return id.Server == types.DefaultUserServer })
	if i < 0 {
		return ids[0].ToNonAD()
	}
	return ids[i].ToNonAD()
}

func (b *bot) privateGroupLabel(ctx context.Context, chat types.JID) string {
	info, err := b.client.GetGroupInfo(ctx, chat)
	if err != nil {
		b.client.Log.Warnf("ler nome do grupo %s: %v", chat, err)
		return ""
	}
	if info.Name == "" {
		return ""
	}
	return " no grupo *" + info.Name + "*"
}
