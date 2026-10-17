package jokes

import (
	"sync"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/economy"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/romance"
)

type bot struct {
	*core.Bot
	economy          *economy.Module
	romance          *romance.Module
	messages         *messageStore
	laughStickerSent sync.Map
	reactions        *reactionStore
}

type Module = bot

type messageStore struct {
	*core.MessageStore
}

func (b *bot) Register() {
	b.AddCommands(b.jokesCommands)
	b.AddReactor(b.reactIfMarked)
	b.AddPassive(core.ModeWords, b.answerWords)
	b.AddPassive(core.ModeLaughs, b.answerLaugh)
}

func New(c *core.Bot, econ *economy.Module, rom *romance.Module) *Module {
	return &bot{
		Bot:       c,
		economy:   econ,
		romance:   rom,
		messages:  &messageStore{c.Messages},
		reactions: &reactionStore{&core.RestrictionStore{DB: c.Messages.DB, Table: "reacoes"}},
	}
}
