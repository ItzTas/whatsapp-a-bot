package economy

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

const (
	backpackTable = "inventarios"
	stashTable    = "guardados"
)

var errNoItems = errors.New("item insuficiente")

type haul struct {
	found item
	grade quality
	broke bool
}

type gathering struct {
	haul
	pulled []haul
	seed   item
	noTool bool
	wait   time.Duration
	broken item
	worn   []item
	spent  []item
	toll   int64
	injury injury
}

type theft struct {
	wait    time.Duration
	empty   bool
	locked  bool
	item    item
	grade   quality
	success bool
	fine    int64
	scuffle scuffle
}

func (s *walletStore) inventory(ctx context.Context, chat types.JID, acc core.Account) ([]stack, error) {
	return loadInventory(ctx, s.db, chat, acc)
}

func loadInventory(ctx context.Context, q core.RowsQuerier, chat types.JID, acc core.Account) ([]stack, error) {
	return loadStacks(ctx, q, backpackTable, chat, acc)
}

func loadStacks(ctx context.Context, q core.RowsQuerier, table string, chat types.JID, acc core.Account) (stacks []stack, err error) {
	rows, err := q.QueryContext(ctx,
		"SELECT item, qualidade, quantidade FROM "+table+" WHERE chat = ? AND pessoa = ? ORDER BY item, qualidade",
		chat.String(), acc.Key,
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

func (s *walletStore) buyItem(ctx context.Context, chat types.JID, acc core.Account, it item, quantity, cost int64, levy PixLevy) (balance int64, err error) {
	tax := levy.cut(cost)
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		err := tx.QueryRowContext(ctx,
			"UPDATE carteiras SET saldo = saldo - ? WHERE chat = ? AND pessoa = ? AND saldo >= ? RETURNING saldo",
			cost+tax, chat.String(), acc.Key, cost+tax,
		).Scan(&balance)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoFunds
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

func (s *walletStore) sellItems(ctx context.Context, chat types.JID, acc core.Account, sales []marketSale) (sold []marketSale, balance int64, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
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
			payout, chat.String(), acc.Key,
		).Scan(&balance)
	})
	return sold, balance, err
}

func (s *walletStore) giveItem(ctx context.Context, chat types.JID, from, to core.Account, id string, quantity int64) (grades []grade, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) (err error) {
		grades, err = moveItems(ctx, tx, chat, from, to, id, quantity)
		return err
	})
	return grades, err
}

func (s *walletStore) tradeItem(ctx context.Context, o *itemOffer) (grades []grade, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) (err error) {
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

func moveItems(ctx context.Context, tx *ledgerTx, chat types.JID, from, to core.Account, id string, quantity int64) ([]grade, error) {
	wear, err := wearIn(ctx, tx, backpackTable, chat, from, id)
	if err != nil {
		return nil, err
	}
	grades, err := takeGrades(ctx, tx, chat, from, id, quantity)
	if err != nil {
		return nil, err
	}
	if err := addGrades(ctx, tx, chat, to, id, grades); err != nil {
		return nil, err
	}
	return grades, keepWear(ctx, tx, backpackTable, chat, to, id, wear)
}

func wearIn(ctx context.Context, tx *ledgerTx, table string, chat types.JID, acc core.Account, id string) (wear int64, err error) {
	err = tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(usos), 0) FROM "+table+" WHERE chat = ? AND pessoa = ? AND item = ?",
		chat.String(), acc.Key, id).Scan(&wear)
	return wear, err
}

func keepWear(ctx context.Context, tx *ledgerTx, table string, chat types.JID, acc core.Account, id string, wear int64) error {
	_, err := tx.ExecContext(ctx, "UPDATE "+table+" SET usos = MAX(usos, ?) WHERE chat = ? AND pessoa = ? AND item = ?",
		wear, chat.String(), acc.Key, id)
	return err
}

func (s *walletStore) consumeItem(ctx context.Context, chat types.JID, acc core.Account, it item, e itemEffect, now time.Time) (wait time.Duration, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		if it.limit != nil {
			limitWait, err := stampCooldown(ctx, tx, chat, acc, it.limit.cooldown, now)
			if err != nil {
				return err
			}
			if limitWait > 0 {
				wait = limitWait
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
			e.coins, chat.String(), acc.Key,
		)
		return err
	})
	return wait, err
}

func (s *walletStore) gather(ctx context.Context, chat types.JID, acc core.Account, a activity, pit crater, now time.Time) (g gathering, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
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
		for range k.pull {
			g.pulled = append(g.pulled, k.find())
		}
		if g.injury, err = hurt(ctx, tx, chat, acc, k.hazard.roll(k.safety), now); err != nil {
			return err
		}
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
		finds := append([]haul{g.haul}, g.pulled...)
		for _, h := range finds {
			if h.found.id == "" {
				continue
			}
			if err := addGrades(ctx, tx, chat, acc, h.found.id, []grade{{h.grade, 1}}); err != nil {
				return err
			}
		}
		if g.seed = k.stray(); g.seed.id != "" {
			if err := addItems(ctx, tx, chat, acc, g.seed.id, 1); err != nil {
				return err
			}
		}
		if pit.charges(acc) {
			if g.toll, err = payToll(ctx, tx, chat, acc, pit.owner, finds); err != nil {
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
			chat.String(), acc.Key, k.tool)
		return err
	})
	return g, err
}

func wearTool(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, id string) (uses int64, err error) {
	err = tx.QueryRowContext(ctx,
		"UPDATE inventarios SET usos = usos + 1 WHERE chat = ? AND pessoa = ? AND item = ? RETURNING usos",
		chat.String(), acc.Key, id,
	).Scan(&uses)
	return uses, err
}

func (s *walletStore) stealItem(ctx context.Context, chat types.JID, thief, victim core.Account, cd cooldown, risk crimeRisk, now time.Time,
	choose func(loot []stack) (target item, success bool)) (t theft, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
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
		if err := t.settle(ctx, tx, chat, thief, victim, loot); err != nil {
			return err
		}
		t.scuffle, err = risk.clash(ctx, tx, chat, thief, victim, t.success && !t.locked, false, now)
		return err
	})
	return t, err
}

func (t *theft) settle(ctx context.Context, tx *ledgerTx, chat types.JID, thief, victim core.Account, loot []stack) error {
	if quantityOf(loot, lockItemID) > 0 {
		t.locked = true
		return takeItems(ctx, tx, chat, victim, lockItemID, 1)
	}
	if !t.success {
		return payTheftFine(ctx, tx, chat, thief, victim, t)
	}
	grades, err := moveItems(ctx, tx, chat, victim, thief, t.item.id, 1)
	if err != nil {
		return err
	}
	t.grade = grades[0].quality
	return nil
}

func payTheftFine(ctx context.Context, tx *ledgerTx, chat types.JID, thief, victim core.Account, t *theft) error {
	var balance int64
	err := tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), thief.Key).Scan(&balance)
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

func stampCooldown(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, cd cooldown, now time.Time) (time.Duration, error) {
	if err := openWallet(ctx, tx, chat, acc); err != nil {
		return 0, err
	}
	var last int64
	err := tx.QueryRowContext(ctx, "SELECT "+cd.column+" FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&last)
	if err != nil {
		return 0, err
	}
	if next := time.Unix(last, 0).Add(cd.every); now.Before(next) {
		return next.Sub(now), nil
	}
	_, err = tx.ExecContext(ctx, "UPDATE carteiras SET "+cd.column+" = ? WHERE chat = ? AND pessoa = ?", now.Unix(), chat.String(), acc.Key)
	return 0, err
}

func addItems(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, id string, quantity int64) error {
	return addGrades(ctx, tx, chat, acc, id, []grade{{normalQuality, quantity}})
}

func addGrades(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, id string, grades []grade) error {
	return addGradesTo(ctx, tx, backpackTable, chat, acc, id, grades)
}

func addGradesTo(ctx context.Context, tx *ledgerTx, table string, chat types.JID, acc core.Account, id string, grades []grade) error {
	for _, g := range grades {
		_, err := tx.ExecContext(ctx,
			"INSERT INTO "+table+` (chat, pessoa, item, qualidade, quantidade) VALUES (?, ?, ?, ?, ?)
			 ON CONFLICT (chat, pessoa, item, qualidade) DO UPDATE SET quantidade = `+table+`.quantidade + excluded.quantidade`,
			chat.String(), acc.Key, id, g.quality, g.quantity,
		)
		if err != nil {
			return err
		}
	}
	if table != backpackTable {
		return nil
	}
	return assembleShards(ctx, tx, chat, acc, id)
}

func takeItems(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, id string, quantity int64) error {
	_, err := takeGrades(ctx, tx, chat, acc, id, quantity)
	return err
}

func spendItem(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, id string) (bool, error) {
	err := takeItems(ctx, tx, chat, acc, id, 1)
	if errors.Is(err, errNoItems) {
		return false, nil
	}
	return err == nil, err
}

func takeGrades(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, id string, quantity int64) ([]grade, error) {
	return takeGradesFrom(ctx, tx, backpackTable, chat, acc, id, quantity)
}

func takeGradesFrom(ctx context.Context, tx *ledgerTx, table string, chat types.JID, acc core.Account, id string, quantity int64) ([]grade, error) {
	held, err := gradesIn(ctx, tx, table, chat, acc, id)
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
		if err := takeGrade(ctx, tx, table, chat, acc, id, grade{g.quality, n}, n == g.quantity); err != nil {
			return nil, err
		}
		taken = append(taken, grade{g.quality, n})
		quantity -= n
	}
	return taken, nil
}

func takeGrade(ctx context.Context, tx *ledgerTx, table string, chat types.JID, acc core.Account, id string, g grade, whole bool) error {
	if whole {
		_, err := tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE chat = ? AND pessoa = ? AND item = ? AND qualidade = ?",
			chat.String(), acc.Key, id, g.quality)
		return err
	}
	_, err := tx.ExecContext(ctx,
		"UPDATE "+table+" SET quantidade = quantidade - ? WHERE chat = ? AND pessoa = ? AND item = ? AND qualidade = ?",
		g.quantity, chat.String(), acc.Key, id, g.quality,
	)
	return err
}

func holdsItem(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, id string) (held bool, err error) {
	err = tx.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT 1 FROM inventarios WHERE chat = ? AND pessoa = ? AND item = ?)",
		chat.String(), acc.Key, id,
	).Scan(&held)
	return held, err
}

func gradesIn(ctx context.Context, tx *ledgerTx, table string, chat types.JID, acc core.Account, id string) (grades []grade, err error) {
	rows, err := tx.QueryContext(ctx,
		"SELECT qualidade, quantidade FROM "+table+" WHERE chat = ? AND pessoa = ? AND item = ? ORDER BY qualidade",
		chat.String(), acc.Key, id,
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
