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

type labelStore struct {
	db     *sql.DB
	table  string
	column string
}

func (s *labelStore) set(ctx context.Context, chat types.JID, users []string, label string) (err error) {
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
			fmt.Sprintf(`INSERT INTO %[1]s (chat, usuario, %[2]s) VALUES (?, ?, ?)
			 ON CONFLICT (chat, usuario) DO UPDATE SET %[2]s = excluded.%[2]s`, s.table, s.column),
			chat.ToNonAD().String(), user, label,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *labelStore) remove(ctx context.Context, chat types.JID, users []string) error {
	first, second := users[0], users[len(users)-1]
	_, err := s.db.ExecContext(ctx,
		fmt.Sprintf("DELETE FROM %s WHERE chat = ? AND usuario IN (?, ?)", s.table),
		chat.ToNonAD().String(), first, second,
	)
	return err
}

func (s *labelStore) inChat(ctx context.Context, chat types.JID) (labels map[string]string, err error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("SELECT usuario, %s FROM %s WHERE chat = ?", s.column, s.table), chat.ToNonAD().String())
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	labels = map[string]string{}
	for rows.Next() {
		var user, label string
		if err := rows.Scan(&user, &label); err != nil {
			return nil, err
		}
		labels[user] = label
	}
	return labels, rows.Err()
}

func (b *bot) apelido(ctx context.Context, evt *events.Message) error {
	target := firstMentionOrSender(evt)
	locked, err := b.nicknameLocked(ctx, evt.Info.Chat, evt.Info.Sender, target)
	if err != nil {
		return fmt.Errorf("checar apelido trancado: %w", err)
	}
	if locked != "" {
		_, err := b.sendMentions(ctx, evt.Info.Chat, locked, []types.JID{target})
		return err
	}
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
	_, err = b.sendMentions(ctx, evt.Info.Chat, fmt.Sprintf("🏷️ Apelido salvo! Agora é @%s", target.User), []types.JID{target})
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
