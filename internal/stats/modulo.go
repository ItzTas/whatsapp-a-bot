package stats

import (
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/economy"
)

type bot struct {
	*core.Bot
	economy  *economy.Module
	messages *messageStore
}

type Module = bot

type messageStore struct {
	*core.MessageStore
}

func (b *bot) Register() {
	b.AddCommands(b.statsCommands)
}

func New(c *core.Bot, econ *economy.Module) *Module {
	return &bot{Bot: c, economy: econ, messages: &messageStore{c.Messages}}
}
