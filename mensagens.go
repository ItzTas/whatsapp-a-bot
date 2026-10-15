package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	messageWindow     = 90 * 24 * time.Hour
	maxStoredMessages = 50_000
	pruneEvery        = 500
)

type messageStore struct {
	db       *sql.DB
	recorded atomic.Int64
}

type chatActivity struct {
	messages int
	senders  int
}

func openMessageStore(ctx context.Context, path string) (*messageStore, error) {
	db, err := sql.Open("sqlite3", "file:"+path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	schema := `
		CREATE TABLE IF NOT EXISTS mensagens (
			chat          TEXT    NOT NULL,
			id            TEXT    NOT NULL,
			remetente     TEXT    NOT NULL,
			remetente_alt TEXT    NOT NULL,
			texto         TEXT    NOT NULL,
			enviada_em    INTEGER NOT NULL,
			PRIMARY KEY (chat, id)
		);
		CREATE INDEX IF NOT EXISTS mensagens_por_remetente ON mensagens (chat, remetente);
		CREATE INDEX IF NOT EXISTS mensagens_por_remetente_alt ON mensagens (chat, remetente_alt);
		CREATE INDEX IF NOT EXISTS mensagens_por_data ON mensagens (chat, enviada_em);
		CREATE INDEX IF NOT EXISTS mensagens_por_idade ON mensagens (enviada_em);
	`
	if _, err := db.ExecContext(ctx, schema); err != nil {
		return nil, errors.Join(fmt.Errorf("criar tabela de mensagens: %w", err), db.Close())
	}
	s := &messageStore{db: db}
	if err := s.prune(ctx); err != nil {
		return nil, errors.Join(err, db.Close())
	}
	return s, nil
}

func (s *messageStore) Close() error {
	return s.db.Close()
}

func (s *messageStore) record(ctx context.Context, evt *events.Message) error {
	text := messageText(evt.Message)
	if text == "" {
		return nil
	}
	_, err := s.db.ExecContext(ctx,
		"INSERT OR IGNORE INTO mensagens (chat, id, remetente, remetente_alt, texto, enviada_em) VALUES (?, ?, ?, ?, ?, ?)",
		evt.Info.Chat.ToNonAD().String(), evt.Info.ID, evt.Info.Sender.User, evt.Info.SenderAlt.User, text, evt.Info.Timestamp.Unix(),
	)
	if err != nil {
		return err
	}
	if s.recorded.Add(1)%pruneEvery != 0 {
		return nil
	}
	return s.prune(ctx)
}

func (s *messageStore) prune(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, "DELETE FROM mensagens WHERE enviada_em < ?", windowStart()); err != nil {
		return fmt.Errorf("apagar mensagens antigas: %w", err)
	}
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM mensagens WHERE enviada_em < (SELECT enviada_em FROM mensagens ORDER BY enviada_em DESC LIMIT 1 OFFSET ?)",
		maxStoredMessages,
	)
	if err != nil {
		return fmt.Errorf("apagar mensagens além do limite: %w", err)
	}
	return nil
}

func (s *messageStore) texts(ctx context.Context, chat types.JID, users []string) (texts []string, err error) {
	first, second := users[0], users[len(users)-1]
	rows, err := s.db.QueryContext(ctx,
		`SELECT texto FROM mensagens
		 WHERE chat = ? AND enviada_em >= ?
		   AND (remetente IN (?, ?) OR remetente_alt IN (?, ?))`,
		chat.String(), windowStart(), first, second, first, second,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			return nil, err
		}
		texts = append(texts, text)
	}
	return texts, rows.Err()
}

func (s *messageStore) activity(ctx context.Context, chat types.JID) (chatActivity, error) {
	var a chatActivity
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*), COUNT(DISTINCT remetente) FROM mensagens WHERE chat = ? AND enviada_em >= ?",
		chat.String(), windowStart(),
	).Scan(&a.messages, &a.senders)
	return a, err
}

func windowStart() int64 {
	return time.Now().Add(-messageWindow).Unix()
}

func (b *bot) recordHistory(ctx context.Context, evt *events.HistorySync) {
	for _, conv := range evt.Data.GetConversations() {
		chat, err := types.ParseJID(conv.GetID())
		if err != nil {
			b.client.Log.Warnf("histórico: chat %q inválido: %v", conv.GetID(), err)
			continue
		}
		for _, item := range conv.GetMessages() {
			msg, err := b.client.ParseWebMessage(chat, item.GetMessage())
			if err != nil {
				b.client.Log.Warnf("histórico: ler mensagem de %s: %v", chat, err)
				continue
			}
			if err := b.messages.record(ctx, msg); err != nil {
				b.client.Log.Errorf("histórico: guardar mensagem de %s: %v", chat, err)
			}
		}
	}
}
