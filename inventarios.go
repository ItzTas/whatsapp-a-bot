package main

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
)

var errNoItems = errors.New("item insuficiente")

type rowsQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type haul struct {
	found item
	broke bool
}

type gathering struct {
	haul
	noTool bool
	wait   time.Duration
	broken item
	worn   []item
	spent  []item
}

type theft struct {
	wait    time.Duration
	empty   bool
	locked  bool
	item    item
	success bool
	fine    int64
}

func (s *walletStore) inventory(ctx context.Context, chat types.JID, acc account) ([]stack, error) {
	return loadInventory(ctx, s.db, chat, acc)
}

func loadInventory(ctx context.Context, q rowsQuerier, chat types.JID, acc account) (stacks []stack, err error) {
	rows, err := q.QueryContext(ctx, "SELECT item, quantidade FROM inventarios WHERE chat = ? AND pessoa = ?", chat.String(), acc.key)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var (
			id       string
			quantity int64
		)
		if err := rows.Scan(&id, &quantity); err != nil {
			return nil, err
		}
		it, known := itemByID(id)
		if !known {
			continue
		}
		stacks = append(stacks, stack{item: it, quantity: quantity})
	}
	return stacks, rows.Err()
}

func (s *walletStore) buyItem(ctx context.Context, chat types.JID, acc account, it item, quantity, cost int64) (balance int64, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx,
			"UPDATE carteiras SET saldo = saldo - ? WHERE chat = ? AND pessoa = ? AND saldo >= ? RETURNING saldo",
			cost, chat.String(), acc.key, cost,
		).Scan(&balance)
		if errors.Is(err, sql.ErrNoRows) {
			return errNoFunds
		}
		if err != nil {
			return err
		}
		return addItems(ctx, tx, chat, acc, it.id, quantity)
	})
	return balance, err
}

func (s *walletStore) sellItem(ctx context.Context, chat types.JID, acc account, it item, quantity, payout int64) (balance int64, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		if err := takeItems(ctx, tx, chat, acc, it.id, quantity); err != nil {
			return err
		}
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		return tx.QueryRowContext(ctx,
			"UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ? RETURNING saldo",
			payout, chat.String(), acc.key,
		).Scan(&balance)
	})
	return balance, err
}

func (s *walletStore) giveItem(ctx context.Context, chat types.JID, from, to account, id string, quantity int64) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if err := takeItems(ctx, tx, chat, from, id, quantity); err != nil {
			return err
		}
		return addItems(ctx, tx, chat, to, id, quantity)
	})
}

func (s *walletStore) tradeItem(ctx context.Context, o *itemOffer) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if err := takeItems(ctx, tx, o.chat, o.seller, o.item.id, o.quantity); err != nil {
			return err
		}
		if err := addItems(ctx, tx, o.chat, o.buyer, o.item.id, o.quantity); err != nil {
			return err
		}
		if err := openWallet(ctx, tx, o.chat, o.seller); err != nil {
			return err
		}
		return moveCoins(ctx, tx, o.chat, o.buyer, o.seller, o.price)
	})
}

func (s *walletStore) consumeItem(ctx context.Context, chat types.JID, acc account, it item, e itemEffect, now time.Time) (wait time.Duration, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		if it.resets != nil {
			caffeineWait, err := stampCooldown(ctx, tx, chat, acc, caffeineCooldown, now)
			if err != nil {
				return err
			}
			if caffeineWait > 0 {
				wait = caffeineWait
				return nil
			}
		}
		if err := takeItems(ctx, tx, chat, acc, it.id, 1); err != nil {
			return err
		}
		if e.prize.id != "" {
			if err := addItems(ctx, tx, chat, acc, e.prize.id, 1); err != nil {
				return err
			}
		}
		if e.coins == 0 && it.resets == nil {
			return nil
		}
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		assignments := []string{"saldo = saldo + ?"}
		for _, column := range it.resets {
			assignments = append(assignments, column+" = 0")
		}
		_, err := tx.ExecContext(ctx,
			"UPDATE carteiras SET "+strings.Join(assignments, ", ")+" WHERE chat = ? AND pessoa = ?",
			e.coins, chat.String(), acc.key,
		)
		return err
	})
	return wait, err
}

func (s *walletStore) gather(ctx context.Context, chat types.JID, acc account, a activity, now time.Time) (g gathering, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		bag, err := loadInventory(ctx, tx, chat, acc)
		if err != nil {
			return err
		}
		k := a.equip(bag)
		if !k.armed {
			g.noTool = true
			return nil
		}
		wait, err := stampCooldown(ctx, tx, chat, acc, k.cooldown, now)
		if err != nil {
			return err
		}
		if wait > 0 {
			g.wait = wait
			return nil
		}
		g.haul, g.worn, g.spent = k.roll(), k.worn, k.spent
		for _, it := range k.spent {
			if err := takeItems(ctx, tx, chat, acc, it.id, 1); err != nil {
				return err
			}
		}
		if g.found.id != "" {
			if err := addItems(ctx, tx, chat, acc, g.found.id, 1); err != nil {
				return err
			}
		}
		if !g.broke {
			return nil
		}
		g.broken, _ = itemByID(k.tool)
		return takeItems(ctx, tx, chat, acc, k.tool, 1)
	})
	return g, err
}

func (s *walletStore) stealItem(ctx context.Context, chat types.JID, thief, victim account, cd cooldown, now time.Time,
	choose func(loot []stack) (target item, success bool)) (t theft, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		loot, err := loadInventory(ctx, tx, chat, victim)
		if err != nil {
			return err
		}
		t.item, t.success = choose(loot)
		if t.item.id == "" {
			t.empty = true
			return nil
		}
		wait, err := stampCooldown(ctx, tx, chat, thief, cd, now)
		if err != nil {
			return err
		}
		if wait > 0 {
			t.wait = wait
			return nil
		}
		if err := markCrime(ctx, tx, chat, thief, now); err != nil {
			return err
		}
		if quantityOf(loot, lockItemID) > 0 {
			t.locked = true
			return takeItems(ctx, tx, chat, victim, lockItemID, 1)
		}
		if t.success {
			if err := takeItems(ctx, tx, chat, victim, t.item.id, 1); err != nil {
				return err
			}
			return addItems(ctx, tx, chat, thief, t.item.id, 1)
		}
		return payTheftFine(ctx, tx, chat, thief, victim, &t)
	})
	return t, err
}

func payTheftFine(ctx context.Context, tx *sql.Tx, chat types.JID, thief, victim account, t *theft) error {
	var balance int64
	err := tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), thief.key).Scan(&balance)
	if err != nil {
		return err
	}
	t.fine = min(balance, max(t.item.value*theftFinePercent/100, theftMinFine))
	if t.fine == 0 {
		return nil
	}
	if err := openWallet(ctx, tx, chat, victim); err != nil {
		return err
	}
	return moveCoins(ctx, tx, chat, thief, victim, t.fine)
}

func stampCooldown(ctx context.Context, tx *sql.Tx, chat types.JID, acc account, cd cooldown, now time.Time) (time.Duration, error) {
	if err := openWallet(ctx, tx, chat, acc); err != nil {
		return 0, err
	}
	var last int64
	err := tx.QueryRowContext(ctx, "SELECT "+cd.column+" FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.key).Scan(&last)
	if err != nil {
		return 0, err
	}
	if next := time.Unix(last, 0).Add(cd.every); now.Before(next) {
		return next.Sub(now), nil
	}
	_, err = tx.ExecContext(ctx, "UPDATE carteiras SET "+cd.column+" = ? WHERE chat = ? AND pessoa = ?", now.Unix(), chat.String(), acc.key)
	return 0, err
}

func addItems(ctx context.Context, tx *sql.Tx, chat types.JID, acc account, id string, quantity int64) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO inventarios (chat, pessoa, item, quantidade) VALUES (?, ?, ?, ?)
		 ON CONFLICT (chat, pessoa, item) DO UPDATE SET quantidade = inventarios.quantidade + excluded.quantidade`,
		chat.String(), acc.key, id, quantity,
	)
	return err
}

func takeItems(ctx context.Context, tx *sql.Tx, chat types.JID, acc account, id string, quantity int64) error {
	res, err := tx.ExecContext(ctx,
		"UPDATE inventarios SET quantidade = quantidade - ? WHERE chat = ? AND pessoa = ? AND item = ? AND quantidade > ?",
		quantity, chat.String(), acc.key, id, quantity,
	)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil || n > 0 {
		return err
	}
	res, err = tx.ExecContext(ctx,
		"DELETE FROM inventarios WHERE chat = ? AND pessoa = ? AND item = ? AND quantidade = ?",
		chat.String(), acc.key, id, quantity,
	)
	if err != nil {
		return err
	}
	removed, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if removed == 0 {
		return errNoItems
	}
	return nil
}
