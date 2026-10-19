package economy

import (
	"context"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

const (
	tiltWindow = 3 * time.Hour
	tiltFloor  = 1_000
	tiltShare  = 50
	tiltChance = 0.3
)

type losingStreak struct {
	bankroll int64
	lost     int64
	last     time.Time
}

type tiltBoard struct {
	mu      sync.Mutex
	streaks map[string]losingStreak
}

type settledBet struct {
	acc    core.Account
	stake  int64
	payout int64
}

func (t *tiltBoard) tally(key string, before, net int64, now time.Time) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.streaks == nil {
		t.streaks = map[string]losingStreak{}
	}
	s, ok := t.streaks[key]
	if !ok || now.Sub(s.last) > tiltWindow {
		s = losingStreak{bankroll: before}
	}
	s.lost += net
	s.last = now
	if s.lost <= 0 {
		delete(t.streaks, key)
		return false
	}
	if s.lost < max(tiltFloor, s.bankroll*tiltShare/100) {
		t.streaks[key] = s
		return false
	}
	delete(t.streaks, key)
	return true
}

func (b *bot) tallyBet(ctx context.Context, chat types.JID, acc core.Account, place string, stake, payout, balance int64) {
	b.NoteBetLoss(ctx, chat, acc, stake, payout)
	if !b.tilts.tally(eventKey(chat, acc), balance+stake-payout, stake-payout, time.Now()) {
		return
	}
	if rand.Float64() >= tiltChance*oracleEventFactor {
		return
	}
	b.stirEventFor(ctx, chat, acc, b.Identities(ctx, acc.JID), eventSpot{place: place})
}

func (b *bot) tallyRound(ctx context.Context, chat types.JID, place string, bets []settledBet) {
	for _, bet := range bets {
		balance, _, err := b.Wallets.Balance(ctx, chat, bet.acc)
		if err != nil {
			b.Client.Log.Errorf("ler saldo de %s depois do %s: %v", bet.acc.JID, place, err)
			continue
		}
		b.tallyBet(ctx, chat, bet.acc, place, bet.stake, bet.payout, balance)
	}
}
