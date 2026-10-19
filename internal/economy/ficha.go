package economy

import (
	"fmt"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
)

const (
	cleanStep  = 0.01
	cleanMax   = 0.07
	cleanEvery = 24 * time.Hour
)

func (p Profession) cleared(lastCrime int64, now time.Time) Profession {
	days := now.Sub(time.Unix(lastCrime, 0)) / cleanEvery
	p.clean = min(cleanMax, float64(days)*cleanStep)
	p.guard += p.clean
	return p
}

func cleanText(p Profession) string {
	if p.clean == 0 {
		return fmt.Sprintf("🦹 Ficha suja: cometeu crime há menos de %s, então não ganha a proteção da ficha limpa", core.FormatWait(cleanEvery))
	}
	if p.clean < cleanMax {
		return fmt.Sprintf("🕊️ Ficha limpa: +%d%% de proteção contra crime, e sobe %d%% a cada %s sem cometer nenhum, até %d%%", percent(p.clean), percent(cleanStep), core.FormatWait(cleanEvery), percent(cleanMax))
	}
	return fmt.Sprintf("🕊️ Ficha limpa: +%d%% de proteção contra crime, o máximo", percent(p.clean))
}
