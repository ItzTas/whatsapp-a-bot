package economy

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"golang.org/x/sync/errgroup"
)

const (
	stakeSharePercent = 70
	allInSharePercent = 60
	betWindowCap      = 500_000
	betWindow         = 2 * time.Hour
	stopLossPercent   = 70
	stopLossWindow    = boxJinxWindow
)

type betOutcome struct {
	net    int64
	leaves time.Duration
}

type stakeRule struct {
	share, allIn, cap int64
	spent             int64
	freed             time.Duration
	stop              int64
	outcomes          []betOutcome
}

var bidRule = stakeRule{share: 100, allIn: 100, cap: math.MaxInt64}

func betRule(spent int64, freed time.Duration, outcomes []betOutcome) stakeRule {
	return stakeRule{share: stakeSharePercent, allIn: allInSharePercent, cap: betWindowCap, spent: spent, freed: freed, stop: stopLossPercent, outcomes: outcomes}
}

func (r stakeRule) most(balance int64) int64 {
	return max(min(balance, max(1, balance*r.share/100), r.cap-r.spent, r.room(balance)), 0)
}

func (r stakeRule) whole(balance int64) int64 {
	return min(r.most(balance), max(1, balance*r.allIn/100))
}

func (r stakeRule) lost() int64 {
	var net int64
	for _, o := range r.outcomes {
		net += o.net
	}
	return -net
}

func (r stakeRule) room(balance int64) int64 {
	return r.roomAfter(balance, r.lost())
}

func (r stakeRule) roomAfter(balance, lost int64) int64 {
	if r.stop == 0 {
		return math.MaxInt64
	}
	return max(1, (balance+lost)*r.stop/100) - lost
}

func (r stakeRule) reopens(balance int64) time.Duration {
	lost := r.lost()
	for _, o := range r.outcomes {
		lost += o.net
		if r.roomAfter(balance, lost) > 0 {
			return o.leaves
		}
	}
	return 0
}

func (r stakeRule) tooMuch(balance int64) string {
	most := r.most(balance)
	if most == 0 && r.room(balance) <= 0 {
		return fmt.Sprintf("🎰 Você já perdeu %s em aposta nas últimas %s, e a casa não deixa ninguém perder mais de %d%% do que tinha nesse tempo. Dá pra apostar de novo daqui a %s", Coins(r.lost()), core.FormatWait(stopLossWindow), r.stop, core.FormatWait(r.reopens(balance)))
	}
	if most == 0 {
		return fmt.Sprintf("🎰 Você já apostou %s nas últimas %s, que é o limite. Dá pra apostar de novo daqui a %s", Coins(r.spent), core.FormatWait(betWindow), core.FormatWait(r.freed))
	}
	if most == r.room(balance) && r.lost() > 0 {
		return fmt.Sprintf("🎰 Você já perdeu %s em aposta nas últimas %s, e a casa não deixa ninguém perder mais de %d%% do que tinha nesse tempo. Agora dá pra apostar no máximo %s", Coins(r.lost()), core.FormatWait(stopLossWindow), r.stop, Coins(most))
	}
	if most == r.cap-r.spent {
		return fmt.Sprintf("🎰 Nas últimas %s você já apostou %s, e o limite é %s nesse tempo. Agora dá pra apostar no máximo %s", core.FormatWait(betWindow), Coins(r.spent), Coins(r.cap), Coins(most))
	}
	return fmt.Sprintf("🎰 A casa só aceita até %d%% do saldo numa aposta só. Agora dá pra apostar no máximo %s de uma vez", r.share, Coins(most))
}

func (b *bot) betRuleOf(ctx context.Context, chat types.JID, acc core.Account) (stakeRule, error) {
	var (
		spent    int64
		freed    time.Duration
		outcomes []betOutcome
	)
	now := time.Now()
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { spent, freed, err = b.Wallets.wagered(gctx, chat, acc, now); return err })
	g.Go(func() (err error) { outcomes, err = b.Wallets.betOutcomes(gctx, chat, acc, now); return err })
	err := g.Wait()
	return betRule(spent, freed, outcomes), err
}

func (s *walletStore) wagered(ctx context.Context, chat types.JID, acc core.Account, now time.Time) (spent int64, freed time.Duration, err error) {
	var oldest int64
	err = s.db.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(valor), 0), COALESCE(MIN(em), 0) FROM apostas_volume WHERE chat = ? AND pessoa = ? AND em > ?",
		chat.String(), acc.Key, now.Add(-betWindow).Unix(),
	).Scan(&spent, &oldest)
	return spent, time.Unix(oldest, 0).Add(betWindow).Sub(now), err
}

func (s *walletStore) betOutcomes(ctx context.Context, chat types.JID, acc core.Account, now time.Time) (outcomes []betOutcome, err error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT em, lucro FROM apostas_recentes WHERE chat = ? AND pessoa = ? AND em >= ? ORDER BY em",
		chat.String(), acc.Key, now.Add(-stopLossWindow).Unix(),
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var at, net int64
		if err := rows.Scan(&at, &net); err != nil {
			return nil, err
		}
		outcomes = append(outcomes, betOutcome{net: net, leaves: time.Unix(at, 0).Add(stopLossWindow).Sub(now)})
	}
	return outcomes, rows.Err()
}

func noteWager(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, stake int64, now time.Time) error {
	_, err := tx.ExecContext(ctx, "DELETE FROM apostas_volume WHERE chat = ? AND pessoa = ? AND em <= ?", chat.String(), acc.Key, now.Add(-betWindow).Unix())
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO apostas_volume (chat, pessoa, em, valor) VALUES (?, ?, ?, ?)", chat.String(), acc.Key, now.Unix(), stake)
	return err
}
