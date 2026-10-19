package economy

import (
	"context"
	"maps"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

const (
	allInPlace  = "tudo"
	allInWindow = 2 * time.Minute
	allInFloor  = 10_000
	allInChance = 0.25
)

type allInBoard struct {
	mu    sync.Mutex
	asked map[string]time.Time
}

func (a *allInBoard) confirmed(key string, now time.Time) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.asked == nil {
		a.asked = map[string]time.Time{}
	}
	maps.DeleteFunc(a.asked, func(_ string, at time.Time) bool { return now.Sub(at) > allInWindow })
	if _, ok := a.asked[key]; ok {
		delete(a.asked, key)
		return true
	}
	a.asked[key] = now
	return false
}

func (b *bot) allIn(ctx context.Context, chat types.JID, acc core.Account, stake, whole int64) string {
	if stake < whole {
		return ""
	}
	if stake < allInFloor || !b.interruptAllIn(ctx, chat, acc) {
		return ""
	}
	return "✋ A aposta de tudo ficou pra depois. Resolve isso aí primeiro e, se ainda quiser, manda o comando de novo"
}

func (b *bot) interruptAllIn(ctx context.Context, chat types.JID, acc core.Account) bool {
	if rand.Float64() >= allInChance*oracleEventFactor {
		return false
	}
	key := eventKey(chat, acc)
	if _, busy := b.scenes.pending.Load(key); busy {
		return false
	}
	mark, err := markOf(ctx, b.Wallets.db, chat, acc, time.Now())
	if err != nil {
		b.Client.Log.Warnf("ler o alvo fácil de %s antes de apostar tudo: %v", acc.JID, err)
		return false
	}
	if mark.haul() == 0 {
		return false
	}
	b.stirEventFor(ctx, chat, acc, b.Identities(ctx, acc.JID), eventSpot{place: allInPlace})
	_, opened := b.scenes.pending.Load(key)
	return opened
}
