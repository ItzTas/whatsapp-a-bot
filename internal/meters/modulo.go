package meters

import (
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/ai"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
)

type bot struct {
	*core.Bot
	ai *ai.Module
}

type Module = bot

func (b *bot) Register() {
	b.AddCommands(b.metersCommands)
}

func New(c *core.Bot, assistant *ai.Module) *Module {
	return &bot{Bot: c, ai: assistant}
}
