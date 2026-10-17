package moderation

import (
	"sync"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
)

type bot struct {
	*core.Bot
	revokedMessages sync.Map
	welcomeBatches  sync.Map
	mutes           *core.RestrictionStore
	warnings        *warningStore
	welcomes        *welcomeStore
}

type Module = bot

func (b *bot) Register() {
	b.AddCommands(b.moderationCommands)
	b.AddInterceptor(b.deleteIfMuted)
	b.AddGroupInfo(b.welcome)
}

func New(c *core.Bot) *Module {
	return &bot{
		Bot:      c,
		mutes:    &core.RestrictionStore{DB: c.Messages.DB, Table: "mutados"},
		warnings: &warningStore{db: c.Messages.DB},
		welcomes: &welcomeStore{db: c.Messages.DB},
	}
}
