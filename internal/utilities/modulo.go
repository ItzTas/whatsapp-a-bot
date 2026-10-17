package utilities

import (
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
)

type bot struct {
	*core.Bot
	transcriber *transcriber
	reminders   *reminderStore
	birthdays   *birthdayStore
}

type Module = bot

func (b *bot) Register() {
	b.AddCommands(b.utilitiesCommands)
	b.AddPassive(core.ModeAutoDownload, b.answerLinks)
	b.AddJob(b.runReminders)
	b.AddJob(b.runBirthdays)
}

func New(c *core.Bot) *Module {
	return &bot{
		Bot:         c,
		transcriber: transcriberFromEnv(),
		reminders:   &reminderStore{db: c.Messages.DB},
		birthdays:   &birthdayStore{db: c.Messages.DB},
	}
}
