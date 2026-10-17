package jokes

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) apelido(ctx context.Context, evt *events.Message) error {
	target := core.FirstMentionOrSender(evt)
	locked, err := b.economy.NicknameLocked(ctx, evt.Info.Chat, evt.Info.Sender, target)
	if err != nil {
		return fmt.Errorf("checar apelido trancado: %w", err)
	}
	if locked != "" {
		_, err := b.SendMentions(ctx, evt.Info.Chat, locked, []types.JID{target})
		return err
	}
	nickname := core.NicknameFrom(core.MessageText(evt.Message))
	if nickname == "" {
		return b.removeNickname(ctx, evt.Info.Chat, target)
	}
	if utf8.RuneCountInString(nickname) > core.MaxNicknameLength {
		return b.SendText(ctx, evt.Info.Chat, fmt.Sprintf("Apelido grande demais, o máximo é %d letras", core.MaxNicknameLength))
	}
	if err := b.Nicknames.Set(ctx, evt.Info.Chat, b.UserAliases(ctx, target), nickname); err != nil {
		return fmt.Errorf("salvar apelido: %w", err)
	}
	_, err = b.SendMentions(ctx, evt.Info.Chat, fmt.Sprintf("🏷️ Apelido salvo! Agora é @%s", target.User), []types.JID{target})
	return err
}

func (b *bot) removeNickname(ctx context.Context, chat, target types.JID) error {
	if err := b.Nicknames.Remove(ctx, chat, b.UserAliases(ctx, target)); err != nil {
		return fmt.Errorf("apagar apelido: %w", err)
	}
	_, err := b.SendMentions(ctx, chat, fmt.Sprintf("🧽 @%s ficou sem apelido", target.User), []types.JID{target})
	return err
}
