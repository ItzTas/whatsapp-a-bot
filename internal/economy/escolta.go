package economy

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

const (
	boostOdds     = 0.2
	escortOddsMax = 0.95
)

func (c contract) odds(m easyMark, now time.Time) float64 {
	chance := c.firm.annul
	if m.jackpot == 0 {
		chance /= 2
	}
	if now.Before(c.boost) {
		chance += boostOdds
	}
	chance *= 1 - levelEscortCut*float64(m.steps())
	return min(chance, escortOddsMax)
}

func escortMark(ctx context.Context, q rowQuerier, chat types.JID, victim core.Account, m easyMark, gang bool, now time.Time) (easyMark, firm, error) {
	if m.jackpot == 0 && !m.lucky {
		return m, firm{}, nil
	}
	c, found, err := contractOf(ctx, q, chat, victim, now)
	if err != nil || !found || (gang && !c.firm.gangs) || rand.Float64() >= c.odds(m, now) {
		return m, firm{}, err
	}
	return easyMark{}, c.firm, nil
}

func (f firm) escortText(victim types.JID) string {
	if f.id == "" {
		return ""
	}
	return fmt.Sprintf("%s A *%s* tava colada em @%s, e nesse roubo nada de alvo fácil\n", f.emoji, f.name, victim.User)
}

func (run *eventRun) cushion(ctx context.Context, q rowQuerier, lost int64) (covered int64, note string, err error) {
	if run.place != windfallPlace && run.place != stalkPlace {
		return 0, "", nil
	}
	c, found, err := contractOf(ctx, q, run.chat, run.acc, run.now)
	if err != nil || !found {
		return 0, "", err
	}
	if covered = lost * c.firm.cover / 100; covered == 0 {
		return 0, "", nil
	}
	return covered, fmt.Sprintf(" (%s a *%s* segurou %s)", c.firm.emoji, c.firm.name, Coins(covered)), nil
}
