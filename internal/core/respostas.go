package core

import (
	"context"
	"slices"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

const (
	typingRefresh = 10 * time.Second
)

func (b *Bot) Typing(ctx context.Context, chat types.JID) (stop func()) {
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer b.Rescue(ctx, chat)
		defer close(done)
		ticker := time.NewTicker(typingRefresh)
		defer ticker.Stop()
		for {
			if err := b.Client.SendChatPresence(ctx, chat, types.ChatPresenceComposing, types.ChatPresenceMediaText); err != nil && ctx.Err() == nil {
				b.Client.Log.Warnf("mostrar digitando em %s: %v", chat, err)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return func() {
		cancel()
		<-done
		if err := b.Client.SendChatPresence(context.WithoutCancel(ctx), chat, types.ChatPresencePaused, types.ChatPresenceMediaText); err != nil {
			b.Client.Log.Warnf("parar digitando em %s: %v", chat, err)
		}
	}
}

func (b *Bot) ReplyTo(ctx context.Context, evt *events.Message, text string) error {
	return b.ReplyMentioning(ctx, evt, text, nil)
}

func (b *Bot) ReplyMentioning(ctx context.Context, evt *events.Message, text string, mentions []types.JID) error {
	_, err := b.SendReply(ctx, evt, text, mentions)
	return err
}

func (b *Bot) SendReply(ctx context.Context, evt *events.Message, text string, mentions []types.JID) (whatsmeow.SendResponse, error) {
	raw := make([]string, len(mentions))
	for i, m := range mentions {
		raw[i] = m.ToNonAD().String()
	}
	return b.Client.SendMessage(ctx, evt.Info.Chat, &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waE2E.ContextInfo{
				StanzaID:      proto.String(evt.Info.ID),
				Participant:   proto.String(evt.Info.Sender.ToNonAD().String()),
				QuotedMessage: evt.Message,
				MentionedJID:  raw,
			},
		},
	})
}

func Revocable(msg *waE2E.Message) bool {
	return !slices.Contains([]bool{
		msg.GetProtocolMessage() != nil,
		msg.GetReactionMessage() != nil,
		msg.GetEncReactionMessage() != nil,
		msg.GetPollUpdateMessage() != nil,
		msg.GetKeepInChatMessage() != nil,
		msg.GetPinInChatMessage() != nil,
	}, true)
}
