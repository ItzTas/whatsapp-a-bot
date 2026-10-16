package main

import (
	"context"
	"fmt"
	"unicode/utf8"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const maxNameLength = 60

func (b *bot) nome(ctx context.Context, evt *events.Message) error {
	target := firstMentionOrSender(evt)
	name := nicknameFrom(messageText(evt.Message))
	if name == "" {
		return b.removeName(ctx, evt.Info.Chat, target)
	}
	if utf8.RuneCountInString(name) > maxNameLength {
		return b.sendText(ctx, evt.Info.Chat, fmt.Sprintf("Nome grande demais, o máximo é %d letras", maxNameLength))
	}
	if err := b.names.set(ctx, evt.Info.Chat, b.userAliases(ctx, target), name); err != nil {
		return fmt.Errorf("salvar nome: %w", err)
	}
	_, err := b.sendMentions(ctx, evt.Info.Chat, fmt.Sprintf("📛 Anotado, o nome de @%s é %s", target.User, name), []types.JID{target})
	return err
}

func (b *bot) removeName(ctx context.Context, chat, target types.JID) error {
	if err := b.names.remove(ctx, chat, b.userAliases(ctx, target)); err != nil {
		return fmt.Errorf("apagar nome: %w", err)
	}
	_, err := b.sendMentions(ctx, chat, fmt.Sprintf("🧽 Esqueci o nome de @%s", target.User), []types.JID{target})
	return err
}

func (b *bot) chatNames(ctx context.Context, chat types.JID) map[string]string {
	names, err := b.names.inChat(ctx, chat)
	if err != nil {
		b.client.Log.Warnf("ler nomes de %s: %v", chat, err)
	}
	return names
}
