package romance

import (
	"sync"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/economy"
)

type bot struct {
	*core.Bot
	economy          *economy.Module
	ficDecks         sync.Map
	romanceProposals sync.Map
}

type Module = bot

func (b *bot) Register() {
	b.AddCommands(b.romanceCommands)
	b.AddCommandFilter(b.answerProposal)
	b.AddRestorers(map[string]core.GameRestorer{
		proposalKind: b.restoreProposal,
	})
}

func New(c *core.Bot, econ *economy.Module) *Module {
	return &bot{Bot: c, economy: econ}
}
