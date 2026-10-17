package city

import (
	"sync"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
)

type bot struct {
	*core.Bot
	games sync.Map
}

type Module = bot

func New(c *core.Bot) *Module {
	return &bot{Bot: c}
}

func (b *bot) Register() {
	b.AddCommands(b.cityCommands)
	b.AddRestorers(map[string]core.GameRestorer{cityKind: b.restoreCity})
	b.OnModeOff(core.ModeCity, b.haltCity)
}
