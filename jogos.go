package main

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

type savedGame struct {
	key     string
	state   []byte
	expires time.Time
}

type storedAccount struct {
	Key string    `json:"key"`
	JID types.JID `json:"jid"`
}

type gameRestorer func(ctx context.Context, saved savedGame) error

func storeAccount(acc account) storedAccount {
	return storedAccount{Key: acc.key, JID: acc.jid}
}

func (s storedAccount) account() account {
	return account{key: s.Key, jid: s.JID}
}

func (b *bot) gameRestorers() map[string]gameRestorer {
	return map[string]gameRestorer{
		forcaKind:        b.restoreForca,
		termoKind:        b.restoreTermo,
		velhaKind:        b.restoreVelha,
		bmcKind:          b.restoreBmc,
		blackjackKind:    b.restoreBlackjack,
		crashKind:        b.restoreCrash,
		raceKind:         b.restoreRace,
		batataKind:       b.restoreBatata,
		roletaKind:       b.restoreRevolver,
		"quemfalou":      b.restoreQuemfalou,
		"quiz":           b.restoreQuiz,
		"bandeira":       b.restoreBandeira,
		"emoji":          b.restoreEmoji,
		"completa":       b.restoreCompleta,
		auctionKind:      b.restoreAuction,
		offerKind:        b.restoreOffer,
		debtRequestKind:  b.restoreDebtRequest,
		proposalKind:     b.restoreProposal,
		hostageSaleKind:  b.restoreHostageSale,
		heistKind:        b.restoreHeist,
		gangKind:         b.restoreGang,
		trialKind:        b.restoreTrial,
		electionKind:     b.restoreElection,
		impeachmentKind:  b.restoreImpeachment,
		impostorGameKind: b.restoreImpostor,
		jokenpoGameKind:  b.restoreJokenpo,
		mentiraGameKind:  b.restoreMentira,
		stopGameKind:     b.restoreStop,
	}
}

func (b *bot) restoreGames(ctx context.Context) {
	defer close(b.gamesReady)
	if !b.client.WaitForConnection(gameConnectWait) {
		b.client.Log.Warnf("o bot não conectou em %s, retomando os jogos assim mesmo", gameConnectWait)
	}
	for kind, restore := range b.gameRestorers() {
		b.restoreKind(ctx, kind, restore)
	}
	if err := b.games.sweep(ctx, time.Now().Add(-gameStaleAfter)); err != nil {
		b.client.Log.Errorf("limpar jogos vencidos: %v", err)
	}
}

func (b *bot) restoreKind(ctx context.Context, kind string, restore gameRestorer) {
	saved, err := b.games.load(ctx, kind)
	if err != nil {
		b.client.Log.Errorf("ler jogos de %s: %v", kind, err)
		return
	}
	for _, game := range saved {
		if err := restore(ctx, game); err != nil {
			b.client.Log.Errorf("retomar %s em %s: %v", kind, game.key, err)
			b.dropGame(ctx, kind, game.key)
		}
	}
}

func (b *bot) awaitGames(ctx context.Context) bool {
	select {
	case <-b.gamesReady:
		return true
	case <-ctx.Done():
		return false
	}
}

func (b *bot) keepGame(ctx context.Context, kind, key string, state any, expires time.Time) {
	if err := b.games.save(context.WithoutCancel(ctx), kind, key, state, expires); err != nil {
		b.client.Log.Errorf("salvar %s em %s: %v", kind, key, err)
	}
}

func (b *bot) dropGame(ctx context.Context, kind, key string) {
	if err := b.games.remove(context.WithoutCancel(ctx), kind, key); err != nil {
		b.client.Log.Errorf("apagar %s em %s: %v", kind, key, err)
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

func dropGameTx(ctx context.Context, tx *sql.Tx, kind, key string) error {
	_, err := tx.ExecContext(ctx, "DELETE FROM jogos WHERE tipo = ? AND chave = ?", kind, key)
	return err
}

func (s *gameStore) load(ctx context.Context, kind string) (saved []savedGame, err error) {
	rows, err := s.db.QueryContext(ctx, "SELECT chave, estado, expira_em FROM jogos WHERE tipo = ?", kind)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var (
			game    savedGame
			state   string
			expires int64
		)
		if err := rows.Scan(&game.key, &state, &expires); err != nil {
			return nil, err
		}
		game.state, game.expires = []byte(state), time.Unix(expires, 0)
		saved = append(saved, game)
	}
	return saved, rows.Err()
}

func (s *gameStore) sweep(ctx context.Context, before time.Time) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM jogos WHERE expira_em < ?", before.Unix())
	return err
}
