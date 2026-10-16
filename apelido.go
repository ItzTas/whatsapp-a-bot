package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const maxNicknameLength = 40

var mentionTagPattern = regexp.MustCompile(`@\d+`)

type nicknameStore struct {
	db *sql.DB
}

func (s *nicknameStore) set(ctx context.Context, chat types.JID, users []string, nickname string) (err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, tx.Rollback())
		}
	}()
	for _, user := range users {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO apelidos (chat, usuario, apelido) VALUES (?, ?, ?)
			 ON CONFLICT (chat, usuario) DO UPDATE SET apelido = excluded.apelido`,
			chat.ToNonAD().String(), user, nickname,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *nicknameStore) remove(ctx context.Context, chat types.JID, users []string) error {
	first, second := users[0], users[len(users)-1]
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM apelidos WHERE chat = ? AND usuario IN (?, ?)",
		chat.ToNonAD().String(), first, second,
	)
	return err
}

func (s *nicknameStore) inChat(ctx context.Context, chat types.JID) (nicknames map[string]string, err error) {
	rows, err := s.db.QueryContext(ctx, "SELECT usuario, apelido FROM apelidos WHERE chat = ?", chat.ToNonAD().String())
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	nicknames = map[string]string{}
	for rows.Next() {
		var user, nickname string
		if err := rows.Scan(&user, &nickname); err != nil {
			return nil, err
		}
		nicknames[user] = nickname
	}
	return nicknames, rows.Err()
}

func (b *bot) apelido(ctx context.Context, evt *events.Message) error {
	target := firstMentionOrSender(evt)
	nickname := nicknameFrom(messageText(evt.Message))
	if nickname == "" {
		return b.removeNickname(ctx, evt.Info.Chat, target)
	}
	if utf8.RuneCountInString(nickname) > maxNicknameLength {
		return b.sendText(ctx, evt.Info.Chat, fmt.Sprintf("Apelido grande demais, o máximo é %d letras", maxNicknameLength))
	}
	if err := b.nicknames.set(ctx, evt.Info.Chat, b.userAliases(ctx, target), nickname); err != nil {
		return fmt.Errorf("salvar apelido: %w", err)
	}
	_, err := b.sendMentions(ctx, evt.Info.Chat, fmt.Sprintf("🏷️ Apelido salvo! Agora é @%s", target.User), []types.JID{target})
	return err
}

func (b *bot) removeNickname(ctx context.Context, chat, target types.JID) error {
	if err := b.nicknames.remove(ctx, chat, b.userAliases(ctx, target)); err != nil {
		return fmt.Errorf("apagar apelido: %w", err)
	}
	_, err := b.sendMentions(ctx, chat, fmt.Sprintf("🧽 @%s ficou sem apelido", target.User), []types.JID{target})
	return err
}

func nicknameFrom(text string) string {
	fields := strings.Fields(mentionTagPattern.ReplaceAllString(text, ""))
	return strings.Join(fields[1:], " ")
}

func (b *bot) withNicknames(ctx context.Context, chat types.JID, text string) string {
	return tagNicknames(text, b.chatNicknames(ctx, chat))
}

func tagNicknames(text string, nicknames map[string]string) string {
	return mentionTagPattern.ReplaceAllStringFunc(text, func(tag string) string {
		nickname, ok := nicknames[strings.TrimPrefix(tag, "@")]
		if !ok {
			return tag
		}
		return fmt.Sprintf("%s (%s)", tag, nickname)
	})
}
