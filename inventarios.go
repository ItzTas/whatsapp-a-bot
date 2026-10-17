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
	grade quality
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
	grade   quality
	success bool
	fine    int64
}

func (s *walletStore) inventory(ctx context.Context, chat types.JID, acc account) ([]stack, error) {
	return loadInventory(ctx, s.db, chat, acc)
}

func loadInventory(ctx context.Context, q rowsQuerier, chat types.JID, acc account) (stacks []stack, err error) {
	rows, err := q.QueryContext(ctx,
		"SELECT item, qualidade, quantidade FROM inventarios WHERE chat = ? AND pessoa = ? ORDER BY item, qualidade",
		chat.String(), acc.key,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var (
			id string
			g  grade
		)
		if err := rows.Scan(&id, &g.quality, &g.quantity); err != nil {
			return nil, err
		}
		it, known := itemByID(id)
		if !known {
			continue
		}
		if n := len(stacks); n > 0 && stacks[n-1].item.id == id {
			stacks[n-1].quantity += g.quantity
			stacks[n-1].grades = append(stacks[n-1].grades, g)
			continue
		}
		stacks = append(stacks, stack{item: it, quantity: g.quantity, grades: []grade{g}})
	}
	return stacks, rows.Err()
}

func (s *walletStore) buyItem(ctx context.Context, chat types.JID, acc account, it item, quantity, cost int64, levy pixLevy) (balance int64, err error) {
	tax := levy.cut(cost)
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx,
			"UPDATE carteiras SET saldo = saldo - ? WHERE chat = ? AND pessoa = ? AND saldo >= ? RETURNING saldo",
			cost+tax, chat.String(), acc.key, cost+tax,
		).Scan(&balance)
		if errors.Is(err, sql.ErrNoRows) {
			return errNoFunds
		}
		if err != nil {
			return err
		}
		if err := addItems(ctx, tx, chat, acc, it.id, quantity); err != nil {
			return err
		}
		return payLevy(ctx, tx, chat, levy, tax)
	})
	return balance, err
}

type marketSale struct {
	item     item
	quantity int64
	percent  int64
	grades   []grade
	payout   int64
}

func (s *walletStore) sellItems(ctx context.Context, chat types.JID, acc account, sales []marketSale) (sold []marketSale, balance int64, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		sold = make([]marketSale, len(sales))
		var payout int64
		for i, sale := range sales {
			grades, err := takeGrades(ctx, tx, chat, acc, sale.item.id, sale.quantity)
			if err != nil {
				return err
			}
			sale.grades, sale.payout = grades, marketPayoutOf(sale.item, grades, sale.percent)
			sold[i] = sale
			payout += sale.payout
		}
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		return tx.QueryRowContext(ctx,
			"UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ? RETURNING saldo",
			payout, chat.String(), acc.key,
		).Scan(&balance)
	})
	return sold, balance, err
}

func (s *walletStore) giveItem(ctx context.Context, chat types.JID, from, to account, id string, quantity int64) (grades []grade, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) (err error) {
		grades, err = moveItems(ctx, tx, chat, from, to, id, quantity)
		return err
	})
	return grades, err
}

func (s *walletStore) tradeItem(ctx context.Context, o *itemOffer) (grades []grade, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) (err error) {
		if grades, err = moveItems(ctx, tx, o.chat, o.seller, o.buyer, o.item.id, o.quantity); err != nil {
			return err
		}
		if err := openWallet(ctx, tx, o.chat, o.seller); err != nil {
			return err
		}
		return moveCoins(ctx, tx, o.chat, o.buyer, o.seller, o.price)
	})
	return grades, err
}

func moveItems(ctx context.Context, tx *sql.Tx, chat types.JID, from, to account, id string, quantity int64) ([]grade, error) {
	grades, err := takeGrades(ctx, tx, chat, from, id, quantity)
	if err != nil {
		return nil, err
	}
	return grades, addGrades(ctx, tx, chat, to, id, grades)
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
		if k.grace > 0 {
			uses, err := wearTool(ctx, tx, chat, acc, k.tool)
			if err != nil {
				return err
			}
			g.broke = g.broke && uses > k.grace
		}
		for _, it := range k.spent {
			if err := takeItems(ctx, tx, chat, acc, it.id, 1); err != nil {
				return err
			}
		}
		if g.found.id != "" {
			if err := addGrades(ctx, tx, chat, acc, g.found.id, []grade{{g.grade, 1}}); err != nil {
				return err
			}
		}
		if !g.broke {
			return nil
		}
		g.broken, _ = itemByID(k.tool)
		if err := takeItems(ctx, tx, chat, acc, k.tool, 1); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "UPDATE inventarios SET usos = 0 WHERE chat = ? AND pessoa = ? AND item = ?",
			chat.String(), acc.key, k.tool)
		return err
	})
	return g, err
}

func wearTool(ctx context.Context, tx *sql.Tx, chat types.JID, acc account, id string) (uses int64, err error) {
	err = tx.QueryRowContext(ctx,
		"UPDATE inventarios SET usos = usos + 1 WHERE chat = ? AND pessoa = ? AND item = ? RETURNING usos",
		chat.String(), acc.key, id,
	).Scan(&uses)
	return uses, err
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
			grades, err := moveItems(ctx, tx, chat, victim, thief, t.item.id, 1)
			if err != nil {
				return err
			}
			t.grade = grades[0].quality
			return nil
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
	return addGrades(ctx, tx, chat, acc, id, []grade{{normalQuality, quantity}})
}

func addGrades(ctx context.Context, tx *sql.Tx, chat types.JID, acc account, id string, grades []grade) error {
	for _, g := range grades {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO inventarios (chat, pessoa, item, qualidade, quantidade) VALUES (?, ?, ?, ?, ?)
			 ON CONFLICT (chat, pessoa, item, qualidade) DO UPDATE SET quantidade = inventarios.quantidade + excluded.quantidade`,
			chat.String(), acc.key, id, g.quality, g.quantity,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func takeItems(ctx context.Context, tx *sql.Tx, chat types.JID, acc account, id string, quantity int64) error {
	_, err := takeGrades(ctx, tx, chat, acc, id, quantity)
	return err
}

func takeGrades(ctx context.Context, tx *sql.Tx, chat types.JID, acc account, id string, quantity int64) ([]grade, error) {
	held, err := heldGrades(ctx, tx, chat, acc, id)
	if err != nil {
		return nil, err
	}
	var have int64
	for _, g := range held {
		have += g.quantity
	}
	if have < quantity {
		return nil, errNoItems
	}
	var taken []grade
	for _, g := range held {
		if quantity == 0 {
			break
		}
		n := min(quantity, g.quantity)
		if err := takeGrade(ctx, tx, chat, acc, id, grade{g.quality, n}, n == g.quantity); err != nil {
			return nil, err
		}
		taken = append(taken, grade{g.quality, n})
		quantity -= n
	}
	return taken, nil
}

func takeGrade(ctx context.Context, tx *sql.Tx, chat types.JID, acc account, id string, g grade, whole bool) error {
	if whole {
		_, err := tx.ExecContext(ctx, "DELETE FROM inventarios WHERE chat = ? AND pessoa = ? AND item = ? AND qualidade = ?",
			chat.String(), acc.key, id, g.quality)
		return err
	}
	_, err := tx.ExecContext(ctx,
		"UPDATE inventarios SET quantidade = quantidade - ? WHERE chat = ? AND pessoa = ? AND item = ? AND qualidade = ?",
		g.quantity, chat.String(), acc.key, id, g.quality,
	)
	return err
}

func heldGrades(ctx context.Context, tx *sql.Tx, chat types.JID, acc account, id string) (grades []grade, err error) {
	rows, err := tx.QueryContext(ctx,
		"SELECT qualidade, quantidade FROM inventarios WHERE chat = ? AND pessoa = ? AND item = ? ORDER BY qualidade",
		chat.String(), acc.key, id,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var g grade
		if err := rows.Scan(&g.quality, &g.quantity); err != nil {
			return nil, err
		}
		grades = append(grades, g)
	}
	return grades, rows.Err()
}
