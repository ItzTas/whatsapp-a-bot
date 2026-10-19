package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow/types"
)

const (
	gameConnectWait = 2 * time.Minute
	gameStaleAfter  = 24 * time.Hour
)

type gameStore struct {
	db *sql.DB
}

type SavedGame struct {
	Key     string
	State   []byte
	Expires time.Time
}

type StoredAccount struct {
	Key string    `json:"key"`
	JID types.JID `json:"jid"`
}

type GameRestorer func(ctx context.Context, saved SavedGame) error

func StoreAccount(acc Account) StoredAccount {
	return StoredAccount(acc)
}

func (s StoredAccount) Account() Account {
	return Account(s)
}

func (b *Bot) RestoreGames(ctx context.Context) {
	defer b.Rescue(ctx, types.EmptyJID)
	defer close(b.gamesReady)
	if !b.Client.WaitForConnection(gameConnectWait) {
		b.Client.Log.Warnf("o bot não conectou em %s, retomando os jogos assim mesmo", gameConnectWait)
	}
	for kind, restore := range b.hooks.restorers {
		b.restoreKind(ctx, kind, restore)
	}
	if err := b.games.sweep(ctx, time.Now().Add(-gameStaleAfter)); err != nil {
		b.Client.Log.Errorf("limpar jogos vencidos: %v", err)
	}
}

func (b *Bot) restoreKind(ctx context.Context, kind string, restore GameRestorer) {
	saved, err := b.games.load(ctx, kind)
	if err != nil {
		b.Client.Log.Errorf("ler jogos de %s: %v", kind, err)
		return
	}
	for _, game := range saved {
		if err := restore(ctx, game); err != nil {
			b.Client.Log.Errorf("retomar %s em %s: %v", kind, game.Key, err)
			b.DropGame(ctx, kind, game.Key)
		}
	}
}

func (b *Bot) awaitGames(ctx context.Context) bool {
	select {
	case <-b.gamesReady:
		return true
	case <-ctx.Done():
		return false
	}
}

func (b *Bot) KeepGame(ctx context.Context, kind, key string, state any, expires time.Time) {
	if err := b.games.save(context.WithoutCancel(ctx), kind, key, state, expires); err != nil {
		b.Client.Log.Errorf("salvar %s em %s: %v", kind, key, err)
	}
}

func (b *Bot) DropGame(ctx context.Context, kind, key string) {
	if err := b.games.remove(context.WithoutCancel(ctx), kind, key); err != nil {
		b.Client.Log.Errorf("apagar %s em %s: %v", kind, key, err)
	}
}

func (s *gameStore) save(ctx context.Context, kind, key string, state any, expires time.Time) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("serializar estado: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO jogos (tipo, chave, estado, expira_em) VALUES (?, ?, ?, ?)
		ON CONFLICT (tipo, chave) DO UPDATE SET estado = excluded.estado, expira_em = excluded.expira_em`,
		kind, key, string(data), expires.Unix(),
	)
	return err
}

func (s *gameStore) remove(ctx context.Context, kind, key string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM jogos WHERE tipo = ? AND chave = ?", kind, key)
	return err
}

func (s *gameStore) load(ctx context.Context, kind string) (saved []SavedGame, err error) {
	rows, err := s.db.QueryContext(ctx, "SELECT chave, estado, expira_em FROM jogos WHERE tipo = ?", kind)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var (
			game    SavedGame
			state   string
			expires int64
		)
		if err := rows.Scan(&game.Key, &state, &expires); err != nil {
			return nil, err
		}
		game.State, game.Expires = []byte(state), time.Unix(expires, 0)
		saved = append(saved, game)
	}
	return saved, rows.Err()
}

func (s *gameStore) sweep(ctx context.Context, before time.Time) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM jogos WHERE expira_em < ?", before.Unix())
	return err
}
