package economy

import (
	"context"
	"math/rand/v2"
	"slices"
	"sync"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	windfallPlace     = "bolada"
	windfallWindow    = 24 * time.Hour
	windfallFade      = 72 * time.Hour
	windfallFadeStep  = 12 * time.Hour
	windfallFloor     = 5_000
	windfallShare     = 30
	windfallCeiling   = 50_000
	windfallChance    = 0.3
	windfallChanceMax = 0.6
	windfallRiseFrom  = 50_000
	windfallRiseTo    = 100_000
	windfallBiteMax   = 3.0
	windfallBiteTo    = 400_000
	stalkPlace        = "alvofacil"
	stalkShare        = 0.25
	stalkEvery        = 4 * time.Hour
)

type purse struct {
	base  int64
	seen  int64
	since time.Time
}

type windfallBoard struct {
	mu     sync.Mutex
	purses map[string]purse
	stalks map[string]time.Time
}

func (w *windfallBoard) struck(key string, balance int64, now time.Time, earned bool) (int64, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.purses == nil {
		w.purses = map[string]purse{}
	}
	p, ok := w.purses[key]
	kept := windfallKept(now.Sub(p.since))
	if !ok || kept == 0 {
		w.purses[key] = purse{base: balance, seen: balance, since: now}
		return 0, false
	}
	if earned {
		p.base += max(balance-p.seen, 0)
	}
	p.seen = balance
	if balance < p.base {
		w.purses[key] = purse{base: balance, seen: balance, since: p.since}
		return 0, false
	}
	haul := int64(float64(balance-p.base) * kept)
	if haul < windfallBar(p.base) {
		w.purses[key] = p
		return 0, false
	}
	w.purses[key] = purse{base: balance, seen: balance, since: now}
	return haul, true
}

func windfallBar(base int64) int64 {
	return min(max(windfallFloor, base*windfallShare/100), windfallCeiling)
}

func (w *windfallBoard) stalk(key string, now time.Time, every time.Duration) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if last, ok := w.stalks[key]; ok && now.Sub(last) < every {
		return false
	}
	if w.stalks == nil {
		w.stalks = map[string]time.Time{}
	}
	w.stalks[key] = now
	return true
}

func windfallKept(age time.Duration) float64 {
	if age <= windfallWindow {
		return 1
	}
	left := windfallWindow + windfallFade - age
	if left <= 0 {
		return 0
	}
	steps := (left + windfallFadeStep - 1) / windfallFadeStep
	return float64(steps) / float64(windfallFade/windfallFadeStep)
}

func windfallOdds(haul int64) float64 {
	rise := float64(haul-windfallRiseFrom) / float64(windfallRiseTo-windfallRiseFrom)
	return windfallChance + (windfallChanceMax-windfallChance)*min(max(rise, 0), 1)
}

func windfallBite(haul int64) float64 {
	rise := float64(haul-windfallRiseFrom) / float64(windfallBiteTo-windfallRiseFrom)
	return 1 + (windfallBiteMax-1)*min(max(rise, 0), 1)
}

func (s eventSpot) bite() float64 {
	if s.place != windfallPlace && s.place != stalkPlace {
		return 1
	}
	return windfallBite(s.haul)
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
	labor := []string{"trabalhar", "daily", "pescar", "minerar", "vasculhar", "catar", "extracao", "coletar", "colher", "vender", "publi", "poupar", "cofre", "imovel", "acoes"}
	now := time.Now()
	haul, struck := b.windfalls.struck(eventKey(chat, acc), balance, now, slices.Contains(labor, cmd.Name))
	if !struck {
		b.stirStalker(ctx, evt, chat, acc, balance, now)
		return
	}
	if rand.Float64() >= windfallOdds(haul)*oracleEventFactor {
		return
	}
	b.stirEventFor(ctx, chat, acc, b.SenderIdentities(ctx, evt), eventSpot{place: windfallPlace, won: true, haul: haul})
}

func (b *bot) stirStalker(ctx context.Context, evt *events.Message, chat types.JID, acc core.Account, balance int64, now time.Time) {
	roll := rand.Float64()
	if roll >= windfallChanceMax*stalkShare*markLevelMax*oracleEventFactor {
		return
	}
	mark, err := markOf(ctx, b.Wallets.db, chat, acc, now)
	if err != nil {
		b.Client.Log.Errorf("ler a marca de alvo fácil de %s: %v", acc.JID, err)
		return
	}
	pressure := float64(mark.steps() + 1)
	haul := min(mark.haul(), balance)
	if haul <= 0 || haul < windfallBar(balance-haul) || roll >= windfallOdds(haul)*stalkShare*pressure*oracleEventFactor {
		return
	}
	if !b.windfalls.stalk(eventKey(chat, acc), now, time.Duration(float64(stalkEvery)/pressure)) {
		return
	}
	b.stirEventFor(ctx, chat, acc, b.SenderIdentities(ctx, evt), eventSpot{place: stalkPlace, won: true, haul: haul})
}
