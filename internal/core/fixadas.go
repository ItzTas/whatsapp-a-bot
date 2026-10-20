package core

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

func (b *Bot) PinMessage(ctx context.Context, chat, sender types.JID, id types.MessageID, length time.Duration) error {
	msg := b.pinChange(chat, sender, id, waE2E.PinInChatMessage_PIN_FOR_ALL)
	msg.MessageContextInfo = &waE2E.MessageContextInfo{
		MessageAddOnDurationInSecs: proto.Uint32(uint32(length.Seconds())),
	}
	_, err := b.Client.SendMessage(ctx, chat, msg)
	return err
}

func (b *Bot) UnpinMessage(ctx context.Context, chat, sender types.JID, id types.MessageID) error {
	_, err := b.Client.SendMessage(ctx, chat, b.pinChange(chat, sender, id, waE2E.PinInChatMessage_UNPIN_FOR_ALL))
	return err
}

func (b *Bot) pinChange(chat, sender types.JID, id types.MessageID, kind waE2E.PinInChatMessage_Type) *waE2E.Message {
	return &waE2E.Message{
		PinInChatMessage: &waE2E.PinInChatMessage{
			Key:               b.Client.BuildMessageKey(chat, sender, id),
			Type:              kind.Enum(),
			SenderTimestampMS: proto.Int64(time.Now().UnixMilli()),
		},
	}
}
