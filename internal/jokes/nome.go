package jokes

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const maxNameLength = 60

func (b *bot) nome(ctx context.Context, evt *events.Message) error {
	target := core.FirstMentionOrSender(evt)
	name := core.NicknameFrom(core.MessageText(evt.Message))
	if name == "" {
		return b.removeName(ctx, evt.Info.Chat, target)
	}
	if utf8.RuneCountInString(name) > maxNameLength {
		return b.SendText(ctx, evt.Info.Chat, fmt.Sprintf("Nome grande demais, o máximo é %d letras", maxNameLength))
	}
	if err := b.Names.Set(ctx, evt.Info.Chat, b.UserAliases(ctx, target), name); err != nil {
		return fmt.Errorf("salvar nome: %w", err)
	}
	_, err := b.SendMentions(ctx, evt.Info.Chat, fmt.Sprintf("📛 Anotado, o nome de @%s é %s", target.User, name), []types.JID{target})
	return err
}

func (b *bot) removeName(ctx context.Context, chat, target types.JID) error {
	if err := b.Names.Remove(ctx, chat, b.UserAliases(ctx, target)); err != nil {
		return fmt.Errorf("apagar nome: %w", err)
	}
	_, err := b.SendMentions(ctx, chat, fmt.Sprintf("🧽 Esqueci o nome de @%s", target.User), []types.JID{target})
	return err
}
