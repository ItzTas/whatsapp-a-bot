package ai

import (
	"sync"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/economy"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/games"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/romance"
)

type bot struct {
	*core.Bot
	economy     *economy.Module
	games       *games.Module
	romance     *romance.Module
	messages    *messageStore
	emojiRecent sync.Map
	persecuted  *core.RestrictionStore
	memories    *memoryStore
}

type Module = bot

type messageStore struct {
	*core.MessageStore
}

func (b *bot) Register() {
	b.AddCommands(b.aiCommands)
	b.AddPassive(core.ModeAIMeddling, b.answerPersecuted)
	b.AddUnknownCommand(b.answerUnknownCommand)
	b.AddRestorers(map[string]core.GameRestorer{
		"emoji": b.restoreEmoji,
	})
}

func New(c *core.Bot, econ *economy.Module, gms *games.Module, rom *romance.Module) *Module {
	return &bot{
		Bot:        c,
		economy:    econ,
		games:      gms,
		romance:    rom,
		messages:   &messageStore{c.Messages},
		persecuted: &core.RestrictionStore{DB: c.Messages.DB, Table: "perseguidos"},
		memories:   &memoryStore{db: c.Messages.DB},
	}
}
