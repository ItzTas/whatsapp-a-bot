package main

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"go.mau.fi/whatsmeow/types"
)

var errNoFunds = errors.New("saldo insuficiente")

type walletStore struct {
	db *sql.DB
}

type account struct {
	key string
	jid types.JID
}

type wallet struct {
	owner   types.JID
	balance int64
}

type cooldown struct {
	column string
	every  time.Duration
}

type robbery struct {
	wait    time.Duration
	poor    bool
	success bool
	amount  int64
}

func (s *walletStore) balance(ctx context.Context, chat types.JID, acc account) (balance int64, rank int, err error) {
	err = s.db.QueryRowContext(ctx,
		`SELECT c.saldo, (SELECT COUNT(*) FROM carteiras o WHERE o.chat = c.chat AND o.saldo > c.saldo) + 1
		 FROM carteiras c WHERE c.chat = ? AND c.pessoa = ?`,
		chat.String(), acc.key,
	).Scan(&balance, &rank)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, nil
	}
	return balance, rank, err
}

func (s *walletStore) richest(ctx context.Context, chat types.JID, limit int) (wallets []wallet, err error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT jid, saldo FROM carteiras WHERE chat = ? AND saldo > 0 ORDER BY saldo DESC LIMIT ?",
		chat.String(), limit,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var (
			w     wallet
			owner string
		)
		if err := rows.Scan(&owner, &w.balance); err != nil {
			return nil, err
		}
		if w.owner, err = types.ParseJID(owner); err != nil {
			return nil, err
		}
		wallets = append(wallets, w)
	}
	return wallets, rows.Err()
}

func (s *walletStore) claim(ctx context.Context, chat types.JID, acc account, cd cooldown, amount int64, now time.Time) (wait time.Duration, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		var last int64
		err := tx.QueryRowContext(ctx, "SELECT "+cd.column+" FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.key).Scan(&last)
		if err != nil {
			return err
		}
		if next := time.Unix(last, 0).Add(cd.every); now.Before(next) {
			wait = next.Sub(now)
			return nil
		}
		_, err = tx.ExecContext(ctx,
			"UPDATE carteiras SET saldo = saldo + ?, "+cd.column+" = ? WHERE chat = ? AND pessoa = ?",
			amount, now.Unix(), chat.String(), acc.key,
		)
		return err
	})
	return wait, err
}

func (s *walletStore) settle(ctx context.Context, chat types.JID, acc account, stake, payout int64) (balance int64, err error) {
	err = s.db.QueryRowContext(ctx,
		"UPDATE carteiras SET saldo = saldo - ? + ? WHERE chat = ? AND pessoa = ? AND saldo >= ? RETURNING saldo",
		stake, payout, chat.String(), acc.key, stake,
	).Scan(&balance)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errNoFunds
	}
	return balance, err
}

func (s *walletStore) transfer(ctx context.Context, chat types.JID, from, to account, amount int64) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if err := openWallet(ctx, tx, chat, to); err != nil {
			return err
		}
		return moveCoins(ctx, tx, chat, from, to, amount)
	})
}

func (s *walletStore) credit(ctx context.Context, chat types.JID, accounts []account, amount int64) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		for _, acc := range accounts {
			if err := openWallet(ctx, tx, chat, acc); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", amount, chat.String(), acc.key)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *walletStore) rob(ctx context.Context, chat types.JID, thief, victim account, every time.Duration, now time.Time, minVictim int64,
	outcome func(thiefBalance, victimBalance int64) (success bool, amount int64)) (r robbery, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		for _, acc := range []account{thief, victim} {
			if err := openWallet(ctx, tx, chat, acc); err != nil {
				return err
			}
		}
		var lastRobbery, thiefBalance, victimBalance int64
		err := tx.QueryRowContext(ctx, "SELECT roubo_em, saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), thief.key).Scan(&lastRobbery, &thiefBalance)
		if err != nil {
			return err
		}
		if next := time.Unix(lastRobbery, 0).Add(every); now.Before(next) {
			r.wait = next.Sub(now)
			return nil
		}
		err = tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), victim.key).Scan(&victimBalance)
		if err != nil {
			return err
		}
		if victimBalance < minVictim {
			r.poor = true
			return nil
		}

		r.success, r.amount = outcome(thiefBalance, victimBalance)
		if _, err := tx.ExecContext(ctx, "UPDATE carteiras SET roubo_em = ? WHERE chat = ? AND pessoa = ?", now.Unix(), chat.String(), thief.key); err != nil {
			return err
		}
		if r.success {
			return moveCoins(ctx, tx, chat, victim, thief, r.amount)
		}
		return moveCoins(ctx, tx, chat, thief, victim, r.amount)
	})
	return r, err
}

func (s *walletStore) inTx(ctx context.Context, fn func(tx *sql.Tx) error) (err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, tx.Rollback())
		}
	}()
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func openWallet(ctx context.Context, tx *sql.Tx, chat types.JID, acc account) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO carteiras (chat, pessoa, jid) VALUES (?, ?, ?)
		 ON CONFLICT (chat, pessoa) DO UPDATE SET jid = excluded.jid`,
		chat.String(), acc.key, acc.jid.String(),
	)
	return err
}

func moveCoins(ctx context.Context, tx *sql.Tx, chat types.JID, from, to account, amount int64) error {
	res, err := tx.ExecContext(ctx,
		"UPDATE carteiras SET saldo = saldo - ? WHERE chat = ? AND pessoa = ? AND saldo >= ?",
		amount, chat.String(), from.key, amount,
	)
	if err != nil {
		return err
	}
	moved, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if moved == 0 {
		return errNoFunds
	}
	_, err = tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", amount, chat.String(), to.key)
	return err
}
