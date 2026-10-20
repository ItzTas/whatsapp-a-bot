package jokes

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const maxNameLength = 60

var typedNumber = regexp.MustCompile(`^[@+]?\d{8,15}$`)

func (b *bot) nome(ctx context.Context, evt *events.Message) error {
	target, name := nameTarget(evt)
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

func nameTarget(evt *events.Message) (types.JID, string) {
	text := core.MessageText(evt.Message)
	if mentions := core.MentionedJIDs(evt.Message); len(mentions) > 0 {
		return mentions[0].ToNonAD(), core.NicknameFrom(text)
	}
	if contact, found := core.QuotedContact(evt.Message); found {
		return contact, core.NicknameFrom(text)
	}
	fields := strings.Fields(text)
	if len(fields) > 1 && typedNumber.MatchString(fields[1]) {
		return types.NewJID(strings.TrimLeft(fields[1], "@+"), types.DefaultUserServer), strings.Join(fields[2:], " ")
	}
	return evt.Info.Sender.ToNonAD(), core.NicknameFrom(text)
}
