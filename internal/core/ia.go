package core

import (
	"context"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/llm"
	"go.mau.fi/whatsmeow/types"
)

func (b *Bot) Asker(ctx context.Context, chat types.JID, user string) llm.Asker {
	unlimited, err := b.ChatModeOn(ctx, chat, modeAIUnlimited)
	if err != nil {
		b.Client.Log.Warnf("ler modo %s de %s: %v", modeAIUnlimited, chat, err)
	}
	return llm.Asker{User: user, Unlimited: unlimited}
}
