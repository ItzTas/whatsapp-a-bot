package core

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

type SenderText struct {
	Sender    string
	SenderAlt string
	Text      string
}

func (s *MessageStore) TextsBySender(ctx context.Context, chat types.JID) (texts []SenderText, err error) {
	rows, err := s.DB.QueryContext(ctx,
		"SELECT remetente, remetente_alt, texto FROM mensagens WHERE chat = ? AND enviada_em >= ?",
		chat.String(), WindowStart(),
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var t SenderText
		if err := rows.Scan(&t.Sender, &t.SenderAlt, &t.Text); err != nil {
			return nil, err
		}
		texts = append(texts, t)
	}
	return texts, rows.Err()
}

const (
	MessageWindow      = 2 * 365 * 24 * time.Hour
	MessageWindowLabel = "últimos 2 anos"
	maxStoredMessages  = 1_000_000
	pruneEvery         = 500
)

type MessageStore struct {
	DB       *sql.DB
	recorded atomic.Int64
}

type ChatActivity struct {
	Messages int
	Senders  int
}

func OpenMessageStore(ctx context.Context, path string) (*MessageStore, error) {
	db, err := sql.Open("sqlite3", "file:"+path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	if err := migrate(ctx, db); err != nil {
		return nil, errors.Join(fmt.Errorf("migrar banco: %w", err), db.Close())
	}
	s := &MessageStore{DB: db}
	if err := s.prune(ctx); err != nil {
		return nil, errors.Join(err, db.Close())
	}
	return s, nil
}

func (s *MessageStore) Close() error {
	return s.DB.Close()
}

func (s *MessageStore) record(ctx context.Context, evt *events.Message) error {
	text := MessageText(evt.Message)
	if text == "" {
		return nil
	}
	_, err := s.DB.ExecContext(ctx,
		"INSERT OR IGNORE INTO mensagens (chat, id, remetente, remetente_alt, texto, enviada_em) VALUES (?, ?, ?, ?, ?, ?)",
		evt.Info.Chat.ToNonAD().String(), evt.Info.ID, evt.Info.Sender.User, evt.Info.SenderAlt.User, text, evt.Info.Timestamp.Unix(),
	)
	if err != nil {
		return err
	}
	if err := s.recordLaugh(ctx, evt, text); err != nil {
		return fmt.Errorf("guardar risada: %w", err)
	}
	if s.recorded.Add(1)%pruneEvery != 0 {
		return nil
	}
	return s.prune(ctx)
}

func (s *MessageStore) prune(ctx context.Context) error {
	if _, err := s.DB.ExecContext(ctx, "DELETE FROM mensagens WHERE enviada_em < ?", WindowStart()); err != nil {
		return fmt.Errorf("apagar mensagens antigas: %w", err)
	}
	if _, err := s.DB.ExecContext(ctx, "DELETE FROM risadas WHERE enviada_em < ?", WindowStart()); err != nil {
		return fmt.Errorf("apagar risadas antigas: %w", err)
	}
	if _, err := s.DB.ExecContext(ctx, "DELETE FROM ia_respostas WHERE enviada_em < ?", WindowStart()); err != nil {
		return fmt.Errorf("apagar respostas antigas da IA: %w", err)
	}
	_, err := s.DB.ExecContext(ctx,
		"DELETE FROM mensagens WHERE enviada_em < (SELECT enviada_em FROM mensagens ORDER BY enviada_em DESC LIMIT 1 OFFSET ?)",
		maxStoredMessages,
	)
	if err != nil {
		return fmt.Errorf("apagar mensagens além do limite: %w", err)
	}
	usedBy := map[string]string{"quemfalou_usadas": "!quemfalou", "completa_usadas": "!completa"}
	for table, command := range usedBy {
		_, err := s.DB.ExecContext(ctx, fmt.Sprintf(
			`DELETE FROM %[1]s
			 WHERE NOT EXISTS (SELECT 1 FROM mensagens m WHERE m.chat = %[1]s.chat AND m.id = %[1]s.mensagem)`,
			table,
		))
		if err != nil {
			return fmt.Errorf("apagar mensagens usadas no %s que já sumiram: %w", command, err)
		}
	}
	return nil
}

func (s *MessageStore) Texts(ctx context.Context, chat types.JID, users []string) (texts []string, err error) {
	first, second := users[0], users[len(users)-1]
	rows, err := s.DB.QueryContext(ctx,
		`SELECT texto FROM mensagens
		 WHERE chat = ? AND enviada_em >= ?
		   AND (remetente IN (?, ?) OR remetente_alt IN (?, ?))`,
		chat.String(), WindowStart(), first, second, first, second,
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

func (s *MessageStore) Activity(ctx context.Context, chat types.JID) (ChatActivity, error) {
	var a ChatActivity
	err := s.DB.QueryRowContext(ctx,
		"SELECT COUNT(*), COUNT(DISTINCT remetente) FROM mensagens WHERE chat = ? AND enviada_em >= ?",
		chat.String(), WindowStart(),
	).Scan(&a.Messages, &a.Senders)
	return a, err
}

func WindowStart() int64 {
	return time.Now().Add(-MessageWindow).Unix()
}

func (b *Bot) RecordHistory(ctx context.Context, evt *events.HistorySync) {
	for _, conv := range evt.Data.GetConversations() {
		chat, err := types.ParseJID(conv.GetID())
		if err != nil {
			b.Client.Log.Warnf("histórico: chat %q inválido: %v", conv.GetID(), err)
			continue
		}
		for _, item := range conv.GetMessages() {
			msg, err := b.Client.ParseWebMessage(chat, item.GetMessage())
			if err != nil {
				b.Client.Log.Warnf("histórico: ler mensagem de %s: %v", chat, err)
				continue
			}
			allowed, err := b.allowed(ctx, msg)
			if err != nil {
				b.Client.Log.Errorf("histórico: checar blacklist/whitelist de %s: %v", chat, err)
				continue
			}
			if !allowed {
				continue
			}
			if err := b.Messages.record(ctx, msg); err != nil {
				b.Client.Log.Errorf("histórico: guardar mensagem de %s: %v", chat, err)
			}
		}
	}
}

const RankingSize = 10

type SenderCount struct {
	Sender    string
	SenderAlt string
	Messages  int
}

func (s *MessageStore) CountsBySender(ctx context.Context, chat types.JID, since int64) (counts []SenderCount, err error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT remetente, remetente_alt, COUNT(*) FROM mensagens
		 WHERE chat = ? AND enviada_em >= ?
		 GROUP BY remetente, remetente_alt`,
		chat.String(), since,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var c SenderCount
		if err := rows.Scan(&c.Sender, &c.SenderAlt, &c.Messages); err != nil {
			return nil, err
		}
		counts = append(counts, c)
	}
	return counts, rows.Err()
}
