package economy

import (
	"context"
	"database/sql"
	"errors"
	"math/rand/v2"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

const (
	alarmChance  = 0.55
	alarmGangCut = 0.15
)

var (
	ErrNoFunds       = errors.New("saldo insuficiente")
	errSavingsActive = errors.New("poupança já ativa")
)

type walletStore struct {
	db *ledgerDB
}

type wallet struct {
	Owner   types.JID
	Balance int64
	title   string
	key     string
}

type cooldown struct {
	column string
	every  time.Duration
}

type Savings struct {
	Amount   int64
	Interest int64
	due      time.Time
}

type robbery struct {
	wait     time.Duration
	shield   time.Duration
	abuse    powerAbuse
	poor     bool
	alarmed  bool
	silent   bool
	mark     easyMark
	escort   firm
	success  bool
	amount   int64
	jail     time.Duration
	jailFine int64
	scuffle  scuffle
}

type flagrant struct {
	odds float64
	jail time.Duration
}

type workday struct {
	wait    time.Duration
	paid    int64
	tax     int64
	tithe   tithe
	pulled  int64
	abuse   powerAbuse
	injury  injury
	garnish debtGarnish
}

func (s *walletStore) Balance(ctx context.Context, chat types.JID, acc core.Account) (balance int64, rank int, err error) {
	err = s.db.QueryRowContext(ctx,
		`SELECT c.saldo, (SELECT COUNT(*) FROM carteiras o WHERE o.chat = c.chat AND o.saldo > c.saldo) + 1
		 FROM carteiras c WHERE c.chat = ? AND c.pessoa = ?`,
		chat.String(), acc.Key,
	).Scan(&balance, &rank)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, nil
	}
	return balance, rank, err
}

func (s *walletStore) Richest(ctx context.Context, chat types.JID, limit int) (wallets []wallet, err error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT pessoa, jid, saldo, titulo FROM carteiras WHERE chat = ? AND saldo > 0 ORDER BY saldo DESC LIMIT ?",
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
		if err := rows.Scan(&w.key, &owner, &w.Balance, &w.title); err != nil {
			return nil, err
		}
		if w.Owner, err = types.ParseJID(owner); err != nil {
			return nil, err
		}
		wallets = append(wallets, w)
	}
	return wallets, rows.Err()
}

func (s *walletStore) claim(ctx context.Context, chat types.JID, acc core.Account, cd cooldown, amount int64, gift item, levy PixLevy, now time.Time) (w workday, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		var last int64
		err := tx.QueryRowContext(ctx, "SELECT "+cd.column+" FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&last)
		if err != nil {
			return err
		}
		if next := time.Unix(last, 0).Add(cd.every); now.Before(next) {
			w.wait = next.Sub(now)
			return nil
		}
		w.paid, w.tax = amount, levy.cut(amount)
		if w.tithe, err = payTithe(ctx, tx, chat, acc, amount); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx,
			"UPDATE carteiras SET saldo = saldo + ?, "+cd.column+" = ? WHERE chat = ? AND pessoa = ?",
			amount-w.tax-w.tithe.paid, now.Unix(), chat.String(), acc.Key,
		)
		if err != nil {
			return err
		}
		if gift.id != "" {
			if err := addItems(ctx, tx, chat, acc, gift.id, 1); err != nil {
				return err
			}
		}
		return payLevy(ctx, tx, chat, levy, w.tax)
	})
	return w, err
}

func (s *walletStore) work(ctx context.Context, chat types.JID, acc core.Account, cd cooldown, abuse abuser, sh shift, levy PixLevy, lien debtLien, now time.Time) (w workday, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		var err error
		if w.wait, w.abuse, err = stampOrAbuse(ctx, tx, chat, acc, cd, abuse, now); err != nil || w.wait > 0 {
			return err
		}
		var balance int64
		err = tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&balance)
		if err != nil {
			return err
		}
		w.paid = max(sh.amount, -balance)
		w.tax = levy.cut(max(w.paid, 0))
		if w.tithe, err = payTithe(ctx, tx, chat, acc, max(w.paid, 0)); err != nil {
			return err
		}
		if w.pulled, err = pullGravity(ctx, tx, chat, acc, max(w.paid, 0), now); err != nil {
			return err
		}
		net := w.paid - w.tax - w.tithe.paid - w.pulled
		_, err = tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", net, chat.String(), acc.Key)
		if err != nil {
			return err
		}
		if err := payLevy(ctx, tx, chat, levy, w.tax); err != nil {
			return err
		}
		if w.garnish, err = garnishPay(ctx, tx, chat, acc, lien, net, now); err != nil {
			return err
		}
		for _, it := range sh.finds {
			if err := addItems(ctx, tx, chat, acc, it.id, 1); err != nil {
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
	staff      string
	cursed     bool
	drug       string
	title      string
	church     string
	doctrine   string
	temple     int
	tithe      int64
	record     int64
}

func (s *walletStore) standing(ctx context.Context, chat types.JID, acc core.Account) (st standing, err error) {
	now := time.Now().Unix()
	err = s.db.QueryRowContext(ctx,
		`SELECT c.profissao,
		        EXISTS (SELECT 1 FROM presidencias p WHERE p.chat = c.chat AND p.pessoa = c.pessoa AND p.mandato_ate > ?),
		        c.imovel, `+staffKinds+`,
		        EXISTS (SELECT 1 FROM zoeiras z WHERE z.chat = c.chat AND z.alvo = c.pessoa AND z.tipo = ? AND z.ate > ?),
		        CASE WHEN c.droga_ate > ? THEN c.droga ELSE '' END,
		        c.titulo, COALESCE(i.nome, ''), COALESCE(i.doutrina, ''), COALESCE(i.templo, 0), COALESCE(i.dizimo, ?), c.ficha_em
		 FROM carteiras c LEFT JOIN igrejas i ON i.id = c.igreja
		 WHERE c.chat = ? AND c.pessoa = ?`,
		now, curseKind, now, now, unsetTithe, chat.String(), acc.Key,
	).Scan(&st.profession, &st.president, &st.estate, &st.staff, &st.cursed, &st.drug, &st.title, &st.church, &st.doctrine, &st.temple, &st.tithe, &st.record)
	if errors.Is(err, sql.ErrNoRows) {
		return standing{}, nil
	}
	return st, err
}

func (s *walletStore) spend(ctx context.Context, chat types.JID, acc core.Account, amount int64, then func(tx *ledgerTx) error) error {
	return s.inTx(ctx, func(tx *ledgerTx) error {
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		if err := debitBalance(ctx, tx, chat, acc, amount); err != nil {
			return err
		}
		return then(tx)
	})
}

func (s *walletStore) changeProfession(ctx context.Context, chat types.JID, acc core.Account, name string, every time.Duration, now time.Time) (wait time.Duration, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		var last int64
		err := tx.QueryRowContext(ctx, "SELECT profissao_em FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&last)
		if err != nil {
			return err
		}
		if next := time.Unix(last, 0).Add(every); now.Before(next) {
			wait = next.Sub(now)
			return nil
		}
		_, err = tx.ExecContext(ctx,
			"UPDATE carteiras SET profissao = ?, profissao_em = ? WHERE chat = ? AND pessoa = ?",
			name, now.Unix(), chat.String(), acc.Key,
		)
		return err
	})
	return wait, err
}

func (s *walletStore) Settle(ctx context.Context, chat types.JID, acc core.Account, stake, payout int64) (balance int64, err error) {
	err = s.db.QueryRowContext(ctx,
		"UPDATE carteiras SET saldo = saldo - ? + ? WHERE chat = ? AND pessoa = ? AND saldo >= ? RETURNING saldo",
		stake, payout, chat.String(), acc.Key, stake,
	).Scan(&balance)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNoFunds
	}
	return balance, err
}

func (s *walletStore) Transfer(ctx context.Context, chat types.JID, from, to core.Account, amount int64) error {
	return s.inTx(ctx, func(tx *ledgerTx) error {
		if err := openWallet(ctx, tx, chat, to); err != nil {
			return err
		}
		return moveCoins(ctx, tx, chat, from, to, amount)
	})
}

func (s *walletStore) Credit(ctx context.Context, chat types.JID, accounts []core.Account, amount int64) error {
	return s.inTx(ctx, func(tx *ledgerTx) error {
		for _, acc := range accounts {
			if err := openWallet(ctx, tx, chat, acc); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", amount, chat.String(), acc.Key)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *walletStore) payShare(ctx context.Context, chat types.JID, payer core.Account, payees []core.Account, percent int64) (each int64, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		var balance int64
		err := tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), payer.Key).Scan(&balance)
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

func (s *walletStore) rob(ctx context.Context, chat types.JID, thief, victim core.Account, cd cooldown, abuse abuser, risk crimeRisk, caught flagrant, now time.Time, minVictim int64,
	outcome func(thiefBalance, victimBalance int64, mark easyMark) (success bool, amount int64)) (r robbery, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		for _, acc := range []core.Account{thief, victim} {
			if err := openWallet(ctx, tx, chat, acc); err != nil {
				return err
			}
		}
		var thiefBalance, victimBalance int64
		err := tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), victim.Key).Scan(&victimBalance)
		if err != nil {
			return err
		}
		if victimBalance < minVictim {
			r.poor = true
			return nil
		}
		if r.shield, err = shieldOn(ctx, tx, chat, victim, now); err != nil || r.shield > 0 {
			return err
		}
		mark, err := markOf(ctx, tx, chat, victim, now)
		if err != nil {
			return err
		}
		mark, escort, err := escortMark(ctx, tx, chat, victim, mark, false, now)
		if err != nil {
			return err
		}
		if r.wait, r.abuse, err = stampOrAbuse(ctx, tx, chat, thief, jackpotPace(cd, mark.jackpot), abuse, now); err != nil || r.wait > 0 {
			return err
		}
		r.mark, r.escort = mark, escort
		if err := markCrime(ctx, tx, chat, thief, now); err != nil {
			return err
		}
		err = tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), thief.Key).Scan(&thiefBalance)
		if err != nil {
			return err
		}
		if r.alarmed, r.silent, err = soundAlarm(ctx, tx, chat, victim, jackpotAlarm(alarmChance, r.mark.jackpot)); err != nil {
			return err
		}
		if err := r.settle(ctx, tx, chat, thief, victim, thiefBalance, victimBalance, outcome); err != nil {
			return err
		}
		if r.scuffle, err = risk.clash(ctx, tx, chat, thief, victim, r.success, r.success, now); err != nil {
			return err
		}
		if r.success {
			return raiseShield(ctx, tx, chat, victim, now.Add(jackpotShield(r.mark.jackpot)))
		}
		if r.alarmed || r.scuffle.criminal.died || thiefBalance >= brokeThief || rand.Float64() >= caught.odds {
			return nil
		}
		r.jail = caught.jail
		r.jailFine, err = jailRobber(ctx, tx, chat, thief, r.jail, now)
		return err
	})
	return r, err
}

func jailRobber(ctx context.Context, tx *ledgerTx, chat types.JID, thief core.Account, jail time.Duration, now time.Time) (int64, error) {
	_, err := tx.ExecContext(ctx,
		"UPDATE carteiras SET preso_ate = MAX(preso_ate, ?), crime_em = 0 WHERE chat = ? AND pessoa = ?",
		now.Add(jail).Unix(), chat.String(), thief.Key,
	)
	if err != nil {
		return 0, err
	}
	return lockUp(ctx, tx, chat, thief, now)
}

func (r *robbery) settle(ctx context.Context, tx *ledgerTx, chat types.JID, thief, victim core.Account, thiefBalance, victimBalance int64,
	outcome func(thiefBalance, victimBalance int64, mark easyMark) (success bool, amount int64)) error {
	if r.alarmed {
		return nil
	}
	if r.success, r.amount = outcome(thiefBalance, victimBalance, r.mark); r.success {
		if err := moveCoins(ctx, tx, chat, victim, thief, r.amount); err != nil {
			return err
		}
		return noteLoss(ctx, tx, chat, victim, lossRobbed, r.amount)
	}
	return moveCoins(ctx, tx, chat, thief, victim, r.amount)
}

func soundAlarm(ctx context.Context, tx *ledgerTx, chat types.JID, victim core.Account, chance float64) (fired, silent bool, err error) {
	held, err := gradesIn(ctx, tx, backpackTable, chat, victim, alarmItemID)
	if err != nil || len(held) == 0 {
		return false, false, err
	}
	if rand.Float64() >= chance {
		return false, true, nil
	}
	return true, false, takeItems(ctx, tx, chat, victim, alarmItemID, 1)
}

func (s *walletStore) Savings(ctx context.Context, chat types.JID, acc core.Account) (sv Savings, found bool, err error) {
	var due int64
	err = s.db.QueryRowContext(ctx,
		"SELECT valor, juros, vence_em FROM poupancas WHERE chat = ? AND pessoa = ?",
		chat.String(), acc.Key,
	).Scan(&sv.Amount, &sv.Interest, &due)
	if errors.Is(err, sql.ErrNoRows) {
		return Savings{}, false, nil
	}
	sv.due = time.Unix(due, 0)
	return sv, err == nil, err
}

func (s *walletStore) deposit(ctx context.Context, chat types.JID, acc core.Account, sv Savings) error {
	return s.inTx(ctx, func(tx *ledgerTx) error {
		res, err := tx.ExecContext(ctx,
			"INSERT INTO poupancas (chat, pessoa, valor, juros, vence_em) VALUES (?, ?, ?, ?, ?) ON CONFLICT DO NOTHING",
			chat.String(), acc.Key, sv.Amount, sv.Interest, sv.due.Unix(),
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
			sv.Amount, chat.String(), acc.Key, sv.Amount,
		)
		if err != nil {
			return err
		}
		debited, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if debited == 0 {
			return ErrNoFunds
		}
		return nil
	})
}

func (s *walletStore) redeem(ctx context.Context, chat types.JID, acc core.Account, now time.Time, levy PixLevy) (sv Savings, tax int64, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		err := tx.QueryRowContext(ctx,
			"DELETE FROM poupancas WHERE chat = ? AND pessoa = ? AND vence_em <= ? RETURNING valor, juros",
			chat.String(), acc.Key, now.Unix(),
		).Scan(&sv.Amount, &sv.Interest)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		tax = levy.cut(sv.Interest)
		_, err = tx.ExecContext(ctx,
			"UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?",
			sv.Amount+sv.Interest-tax, chat.String(), acc.Key,
		)
		if err != nil {
			return err
		}
		return payLevy(ctx, tx, chat, levy, tax)
	})
	return sv, tax, err
}

func (s *walletStore) inTx(ctx context.Context, fn func(tx *ledgerTx) error) (err error) {
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

func openWallet(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO carteiras (chat, pessoa, jid) VALUES (?, ?, ?)
		 ON CONFLICT (chat, pessoa) DO UPDATE SET jid = excluded.jid`,
		chat.String(), acc.Key, acc.JID.String(),
	)
	return err
}

func moveCoins(ctx context.Context, tx *ledgerTx, chat types.JID, from, to core.Account, amount int64) error {
	if err := debitBalance(ctx, tx, chat, from, amount); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", amount, chat.String(), to.Key)
	return err
}

func debitBalance(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, amount int64) error {
	res, err := tx.ExecContext(ctx,
		"UPDATE carteiras SET saldo = saldo - ? WHERE chat = ? AND pessoa = ? AND saldo >= ?",
		amount, chat.String(), acc.Key, amount,
	)
	if err != nil {
		return err
	}
	debited, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if debited == 0 {
		return ErrNoFunds
	}
	return nil
}
