package main

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"go.mau.fi/whatsmeow/types"
)

var (
	errNoFunds       = errors.New("saldo insuficiente")
	errSavingsActive = errors.New("poupança já ativa")
)

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
	title   string
}

type cooldown struct {
	column string
	every  time.Duration
}

type savings struct {
	amount   int64
	interest int64
	due      time.Time
}

type robbery struct {
	wait    time.Duration
	abuse   powerAbuse
	poor    bool
	alarmed bool
	success bool
	amount  int64
}

type workday struct {
	wait    time.Duration
	paid    int64
	tax     int64
	abuse   powerAbuse
	injury  injury
	garnish debtGarnish
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
		"SELECT jid, saldo, titulo FROM carteiras WHERE chat = ? AND saldo > 0 ORDER BY saldo DESC LIMIT ?",
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
		if err := rows.Scan(&owner, &w.balance, &w.title); err != nil {
			return nil, err
		}
		if w.owner, err = types.ParseJID(owner); err != nil {
			return nil, err
		}
		wallets = append(wallets, w)
	}
	return wallets, rows.Err()
}

func (s *walletStore) claim(ctx context.Context, chat types.JID, acc account, cd cooldown, amount int64, levy pixLevy, now time.Time) (wait time.Duration, tax int64, err error) {
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
		tax = levy.cut(amount)
		_, err = tx.ExecContext(ctx,
			"UPDATE carteiras SET saldo = saldo + ?, "+cd.column+" = ? WHERE chat = ? AND pessoa = ?",
			amount-tax, now.Unix(), chat.String(), acc.key,
		)
		if err != nil {
			return err
		}
		return payLevy(ctx, tx, chat, levy, tax)
	})
	return wait, tax, err
}

func (s *walletStore) work(ctx context.Context, chat types.JID, acc account, cd cooldown, abuse abuser, sh shift, levy pixLevy, lien debtLien, now time.Time) (w workday, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		var err error
		if w.wait, w.abuse, err = stampOrAbuse(ctx, tx, chat, acc, cd, abuse, now); err != nil || w.wait > 0 {
			return err
		}
		var balance int64
		err = tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.key).Scan(&balance)
		if err != nil {
			return err
		}
		w.paid = max(sh.amount, -balance)
		w.tax = levy.cut(max(w.paid, 0))
		_, err = tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", w.paid-w.tax, chat.String(), acc.key)
		if err != nil {
			return err
		}
		if err := payLevy(ctx, tx, chat, levy, w.tax); err != nil {
			return err
		}
		if w.garnish, err = garnishPay(ctx, tx, chat, acc, lien, w.paid-w.tax, now); err != nil {
			return err
		}
		if sh.found.id != "" {
			if err := addItems(ctx, tx, chat, acc, sh.found.id, 1); err != nil {
				return err
			}
		}
		w.injury, err = hurt(ctx, tx, chat, acc, sh.wound, now)
		return err
	})
	return w, err
}

type standing struct {
	profession string
	president  bool
	estate     int
	guarded    bool
	bribed     bool
	cursed     bool
	drug       string
	title      string
}

func (s *walletStore) standing(ctx context.Context, chat types.JID, acc account) (st standing, err error) {
	now := time.Now().Unix()
	err = s.db.QueryRowContext(ctx,
		`SELECT c.profissao,
		        EXISTS (SELECT 1 FROM presidencias p WHERE p.chat = c.chat AND p.pessoa = c.pessoa AND p.mandato_ate > ?),
		        c.imovel, c.seguranca_ate > ?, c.propina_ate > ?,
		        EXISTS (SELECT 1 FROM zoeiras z WHERE z.chat = c.chat AND z.alvo = c.pessoa AND z.tipo = ? AND z.ate > ?),
		        CASE WHEN c.droga_ate > ? THEN c.droga ELSE '' END,
		        c.titulo
		 FROM carteiras c WHERE c.chat = ? AND c.pessoa = ?`,
		now, now, now, curseKind, now, now, chat.String(), acc.key,
	).Scan(&st.profession, &st.president, &st.estate, &st.guarded, &st.bribed, &st.cursed, &st.drug, &st.title)
	if errors.Is(err, sql.ErrNoRows) {
		return standing{}, nil
	}
	return st, err
}

func (s *walletStore) spend(ctx context.Context, chat types.JID, acc account, amount int64, then func(tx *sql.Tx) error) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		if err := debitBalance(ctx, tx, chat, acc, amount); err != nil {
			return err
		}
		return then(tx)
	})
}

func (s *walletStore) changeProfession(ctx context.Context, chat types.JID, acc account, name string, every time.Duration, now time.Time) (wait time.Duration, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		var last int64
		err := tx.QueryRowContext(ctx, "SELECT profissao_em FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.key).Scan(&last)
		if err != nil {
			return err
		}
		if next := time.Unix(last, 0).Add(every); now.Before(next) {
			wait = next.Sub(now)
			return nil
		}
		_, err = tx.ExecContext(ctx,
			"UPDATE carteiras SET profissao = ?, profissao_em = ? WHERE chat = ? AND pessoa = ?",
			name, now.Unix(), chat.String(), acc.key,
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

func (s *walletStore) payShare(ctx context.Context, chat types.JID, payer account, payees []account, percent int64) (each int64, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		var balance int64
		err := tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), payer.key).Scan(&balance)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		each = balance * percent / 100 / int64(len(payees))
		if each == 0 {
			return nil
		}
		for _, payee := range payees {
			if err := openWallet(ctx, tx, chat, payee); err != nil {
				return err
			}
			if err := moveCoins(ctx, tx, chat, payer, payee, each); err != nil {
				return err
			}
		}
		return nil
	})
	return each, err
}

func (s *walletStore) rob(ctx context.Context, chat types.JID, thief, victim account, cd cooldown, abuse abuser, now time.Time, minVictim int64,
	outcome func(thiefBalance, victimBalance int64) (success bool, amount int64)) (r robbery, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		for _, acc := range []account{thief, victim} {
			if err := openWallet(ctx, tx, chat, acc); err != nil {
				return err
			}
		}
		var thiefBalance, victimBalance int64
		err := tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), victim.key).Scan(&victimBalance)
		if err != nil {
			return err
		}
		if victimBalance < minVictim {
			r.poor = true
			return nil
		}
		if r.wait, r.abuse, err = stampOrAbuse(ctx, tx, chat, thief, cd, abuse, now); err != nil || r.wait > 0 {
			return err
		}
		if err := markCrime(ctx, tx, chat, thief, now); err != nil {
			return err
		}
		err = tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), thief.key).Scan(&thiefBalance)
		if err != nil {
			return err
		}
		err = takeItems(ctx, tx, chat, victim, alarmItemID, 1)
		if err == nil {
			r.alarmed = true
			return nil
		}
		if !errors.Is(err, errNoItems) {
			return err
		}

		r.success, r.amount = outcome(thiefBalance, victimBalance)
		if r.success {
			return moveCoins(ctx, tx, chat, victim, thief, r.amount)
		}
		return moveCoins(ctx, tx, chat, thief, victim, r.amount)
	})
	return r, err
}

func (s *walletStore) savings(ctx context.Context, chat types.JID, acc account) (sv savings, found bool, err error) {
	var due int64
	err = s.db.QueryRowContext(ctx,
		"SELECT valor, juros, vence_em FROM poupancas WHERE chat = ? AND pessoa = ?",
		chat.String(), acc.key,
	).Scan(&sv.amount, &sv.interest, &due)
	if errors.Is(err, sql.ErrNoRows) {
		return savings{}, false, nil
	}
	sv.due = time.Unix(due, 0)
	return sv, err == nil, err
}

func (s *walletStore) deposit(ctx context.Context, chat types.JID, acc account, sv savings) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx,
			"INSERT INTO poupancas (chat, pessoa, valor, juros, vence_em) VALUES (?, ?, ?, ?, ?) ON CONFLICT DO NOTHING",
			chat.String(), acc.key, sv.amount, sv.interest, sv.due.Unix(),
		)
		if err != nil {
			return err
		}
		inserted, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if inserted == 0 {
			return errSavingsActive
		}
		res, err = tx.ExecContext(ctx,
			"UPDATE carteiras SET saldo = saldo - ? WHERE chat = ? AND pessoa = ? AND saldo >= ?",
			sv.amount, chat.String(), acc.key, sv.amount,
		)
		if err != nil {
			return err
		}
		debited, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if debited == 0 {
			return errNoFunds
		}
		return nil
	})
}

func (s *walletStore) redeem(ctx context.Context, chat types.JID, acc account, now time.Time, levy pixLevy) (sv savings, tax int64, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx,
			"DELETE FROM poupancas WHERE chat = ? AND pessoa = ? AND vence_em <= ? RETURNING valor, juros",
			chat.String(), acc.key, now.Unix(),
		).Scan(&sv.amount, &sv.interest)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		tax = levy.cut(sv.interest)
		_, err = tx.ExecContext(ctx,
			"UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?",
			sv.amount+sv.interest-tax, chat.String(), acc.key,
		)
		if err != nil {
			return err
		}
		return payLevy(ctx, tx, chat, levy, tax)
	})
	return sv, tax, err
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
	if err := debitBalance(ctx, tx, chat, from, amount); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", amount, chat.String(), to.key)
	return err
}

func debitBalance(ctx context.Context, tx *sql.Tx, chat types.JID, acc account, amount int64) error {
	res, err := tx.ExecContext(ctx,
		"UPDATE carteiras SET saldo = saldo - ? WHERE chat = ? AND pessoa = ? AND saldo >= ?",
		amount, chat.String(), acc.key, amount,
	)
	if err != nil {
		return err
	}
	debited, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if debited == 0 {
		return errNoFunds
	}
	return nil
}
