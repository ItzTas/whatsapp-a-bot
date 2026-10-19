package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

type lossCause string

const (
	lossBet    lossCause = "aposta"
	lossRobbed lossCause = "roubo"
	lossTax    lossCause = "imposto"
)

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type richFall struct {
	who   types.JID
	was   int
	now   int
	cause lossCause
}

func noteLoss(ctx context.Context, db execer, chat types.JID, acc core.Account, cause lossCause, amount int64) error {
	if amount <= 0 {
		return nil
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO prejuizos (chat, pessoa, motivo, valor) VALUES (?, ?, ?, ?)
		 ON CONFLICT (chat, pessoa, motivo) DO UPDATE SET valor = valor + excluded.valor`,
		chat.String(), acc.Key, string(cause), amount,
	)
	return err
}

func (b *bot) NoteBet(ctx context.Context, chat types.JID, acc core.Account, stake, payout int64) {
	if err := noteLoss(ctx, b.Wallets.db, chat, acc, lossBet, stake-payout); err != nil {
		b.Client.Log.Warnf("anotar prejuízo de aposta de %s: %v", acc.JID, err)
	}
	if err := b.Wallets.noteStreak(ctx, chat, acc, payout-stake); err != nil {
		b.Client.Log.Warnf("anotar série de apostas de %s: %v", acc.JID, err)
	}
	if err := b.Wallets.noteJackpot(ctx, chat, acc, payout-stake, time.Now()); err != nil {
		b.Client.Log.Warnf("anotar prêmio de aposta de %s: %v", acc.JID, err)
	}
	if err := b.Wallets.noteWinnings(ctx, chat, acc, payout-stake, time.Now()); err != nil {
		b.Client.Log.Warnf("anotar lucro do dia na aposta de %s: %v", acc.JID, err)
	}
	if err := b.Wallets.noteRecentWinnings(ctx, chat, acc, payout-stake, time.Now()); err != nil {
		b.Client.Log.Warnf("anotar lucro recente na aposta de %s: %v", acc.JID, err)
	}
}

func (s *walletStore) falls(ctx context.Context, chat types.JID, ranking []wallet) (falls []richFall, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		causes, err := lossCauses(ctx, tx, chat)
		if err != nil {
			return err
		}
		if falls, err = fallenRich(ctx, tx, chat, ranking, causes); err != nil {
			return err
		}
		return snapshotRich(ctx, tx, chat, ranking)
	})
	return falls, err
}

func lossCauses(ctx context.Context, tx *ledgerTx, chat types.JID) (causes map[string]lossCause, err error) {
	rows, err := tx.QueryContext(ctx, "SELECT pessoa, motivo FROM prejuizos WHERE chat = ? ORDER BY valor", chat.String())
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	causes = map[string]lossCause{}
	for rows.Next() {
		var key, cause string
		if err := rows.Scan(&key, &cause); err != nil {
			return nil, err
		}
		causes[key] = lossCause(cause)
	}
	return causes, rows.Err()
}

func fallenRich(ctx context.Context, tx *ledgerTx, chat types.JID, ranking []wallet, causes map[string]lossCause) (falls []richFall, err error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT r.pessoa, r.jid, r.posicao, r.saldo, COALESCE(c.saldo, 0)
		 FROM ricos_ranking r LEFT JOIN carteiras c ON c.chat = r.chat AND c.pessoa = r.pessoa
		 WHERE r.chat = ? ORDER BY r.posicao`,
		chat.String(),
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	places := make(map[string]int, len(ranking))
	for i, w := range ranking {
		places[w.key] = i + 1
	}
	for rows.Next() {
		var (
			key, jid      string
			f             richFall
			before, after int64
		)
		if err := rows.Scan(&key, &jid, &f.was, &before, &after); err != nil {
			return nil, err
		}
		f.now = places[key]
		if after >= before || (f.now != 0 && f.now <= f.was) {
			continue
		}
		if f.who, err = types.ParseJID(jid); err != nil {
			return nil, err
		}
		f.cause = causes[key]
		falls = append(falls, f)
	}
	return falls, rows.Err()
}

func snapshotRich(ctx context.Context, tx *ledgerTx, chat types.JID, ranking []wallet) error {
	for _, query := range []string{"DELETE FROM ricos_ranking WHERE chat = ?", "DELETE FROM prejuizos WHERE chat = ?"} {
		if _, err := tx.ExecContext(ctx, query, chat.String()); err != nil {
			return err
		}
	}
	for i, w := range ranking {
		_, err := tx.ExecContext(ctx,
			"INSERT INTO ricos_ranking (chat, pessoa, jid, posicao, saldo) VALUES (?, ?, ?, ?, ?)",
			chat.String(), w.key, w.Owner.String(), i+1, w.Balance,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f richFall) text() string {
	why := map[lossCause]string{
		lossBet:    "empobreceu apostando 🎰",
		lossRobbed: "empobreceu sendo roubado 🥷",
		lossTax:    "empobreceu pagando imposto pro Leão 🦁",
	}[f.cause]
	if why == "" {
		why = "empobreceu torrando dinheiro por aí 💸"
	}
	if f.now == 0 {
		return fmt.Sprintf("📉 @%s era %s e saiu do ranking: %s", f.who.User, richPlace(f.was), why)
	}
	return fmt.Sprintf("📉 @%s caiu de %s pra %s: %s", f.who.User, richPlace(f.was), richPlace(f.now), why)
}

func richPlace(place int) string {
	medals := []string{"🥇", "🥈", "🥉"}
	if place <= len(medals) {
		return medals[place-1]
	}
	return fmt.Sprintf("%dº", place)
}
