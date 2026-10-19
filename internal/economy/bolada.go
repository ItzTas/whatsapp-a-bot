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
	windfallPlace  = "bolada"
	windfallWindow = 2 * time.Hour
	windfallFloor  = 5_000
	windfallShare  = 30
	windfallChance = 0.3
)

type purse struct {
	base  int64
	since time.Time
}

type windfallBoard struct {
	mu     sync.Mutex
	purses map[string]purse
}

func (w *windfallBoard) struck(key string, balance int64, now time.Time) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.purses == nil {
		w.purses = map[string]purse{}
	}
	p, ok := w.purses[key]
	if !ok || now.Sub(p.since) > windfallWindow {
		w.purses[key] = purse{base: balance, since: now}
		return false
	}
	if balance < p.base {
		w.purses[key] = purse{base: balance, since: p.since}
		return false
	}
	if balance-p.base < max(windfallFloor, p.base*windfallShare/100) {
		return false
	}
	w.purses[key] = purse{base: balance, since: now}
	return true
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
	if !b.windfalls.struck(eventKey(chat, acc), balance, time.Now()) {
		return
	}
	if rand.Float64() >= windfallChance*oracleEventFactor {
		return
	}
	b.stirEventFor(ctx, chat, acc, b.SenderIdentities(ctx, evt), eventSpot{place: windfallPlace, won: true})
}
