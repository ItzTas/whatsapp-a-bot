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
	pins            *pinStore
}

type Module = bot

func (b *bot) Register() {
	b.AddCommands(b.moderationCommands)
	b.AddInterceptor(b.deleteIfMuted)
	b.AddInterceptor(b.forgetUnpinned)
	b.AddGroupInfo(b.welcome)
	b.AddJob(b.runPins)
}

func New(c *core.Bot) *Module {
	return &bot{
		Bot:      c,
		mutes:    &core.RestrictionStore{DB: c.Messages.DB, Table: "mutados"},
		warnings: &warningStore{db: c.Messages.DB},
		welcomes: &welcomeStore{db: c.Messages.DB},
		pins:     &pinStore{db: c.Messages.DB},
	}
}
