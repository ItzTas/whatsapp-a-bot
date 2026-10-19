package economy

import (
	"context"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	windfallPlace     = "bolada"
	windfallWindow    = 2 * time.Hour
	windfallFloor     = 5_000
	windfallShare     = 30
	windfallCeiling   = 50_000
	windfallChance    = 0.3
	windfallChanceMax = 0.6
	windfallRiseFrom  = 50_000
	windfallRiseTo    = 100_000
)

type purse struct {
	base  int64
	since time.Time
}

type windfallBoard struct {
	mu     sync.Mutex
	purses map[string]purse
}

func (w *windfallBoard) struck(key string, balance int64, now time.Time) (int64, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.purses == nil {
		w.purses = map[string]purse{}
	}
	p, ok := w.purses[key]
	if !ok || now.Sub(p.since) > windfallWindow {
		w.purses[key] = purse{base: balance, since: now}
		return 0, false
	}
	if balance < p.base {
		w.purses[key] = purse{base: balance, since: p.since}
		return 0, false
	}
	haul := balance - p.base
	if haul < min(max(windfallFloor, p.base*windfallShare/100), windfallCeiling) {
		return 0, false
	}
	w.purses[key] = purse{base: balance, since: now}
	return haul, true
}

func windfallOdds(haul int64) float64 {
	rise := float64(haul-windfallRiseFrom) / float64(windfallRiseTo-windfallRiseFrom)
	return windfallChance + (windfallChanceMax-windfallChance)*min(max(rise, 0), 1)
}

func (b *bot) stirWindfall(ctx context.Context, evt *events.Message, cmd core.Command) {
	chat := evt.Info.Chat.ToNonAD()
	if cmd.Category.Mode != core.ModeEconomy || chat.Server != types.GroupServer {
		return
	}
	acc := b.Account(ctx, evt.Info.Sender)
	balance, _, err := b.Wallets.Balance(ctx, chat, acc)
	if err != nil {
		b.Client.Log.Errorf("ler saldo de %s depois do !%s: %v", acc.JID, cmd.Name, err)
		return
	}
	haul, struck := b.windfalls.struck(eventKey(chat, acc), balance, time.Now())
	if !struck {
		return
	}
	if rand.Float64() >= windfallOdds(haul)*oracleEventFactor {
		return
	}
	b.stirEventFor(ctx, chat, acc, b.SenderIdentities(ctx, evt), eventSpot{place: windfallPlace, won: true, haul: haul})
}
