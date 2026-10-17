package jokes

import (
	"context"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	laughStickerCooldown = 15 * time.Second
)

func (b *bot) kkk(ctx context.Context, evt *events.Message) error {
	return b.SendLaughSticker(ctx, evt.Info.Chat, quotedReply(evt.Message))
}

func quotedReply(msg *waE2E.Message) *waE2E.ContextInfo {
	quoted := msg.GetExtendedTextMessage().GetContextInfo()
	if quoted.GetStanzaID() == "" {
		return nil
	}
	return &waE2E.ContextInfo{
		StanzaID:      quoted.StanzaID,
		Participant:   quoted.Participant,
		QuotedMessage: quoted.QuotedMessage,
	}
}

func (b *bot) answerLaugh(ctx context.Context, evt *events.Message) error {
	if !core.HasLaugh(core.MessageText(evt.Message)) {
		return nil
	}
	on, err := b.Modes.Enabled(ctx, evt.Info.Chat.ToNonAD(), core.ModeLaughs)
	if err != nil || !on {
		return err
	}
	if !b.laughCooldownOver(evt.Info.Chat.ToNonAD()) {
		return nil
	}
	sticker, err := core.RandomLaughSticker()
	if err != nil {
		return err
	}
	return b.SendSticker(ctx, evt.Info.Chat, sticker, nil)
}

func (b *bot) laughCooldownOver(chat types.JID) bool {
	now := time.Now()
	last, sent := b.laughStickerSent.LoadOrStore(chat.String(), now)
	if !sent {
		return true
	}
	if now.Sub(last.(time.Time)) < laughStickerCooldown {
		return false
	}
	return b.laughStickerSent.CompareAndSwap(chat.String(), last, now)
}
