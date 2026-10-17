package games

import (
	"context"
	"sync"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/economy"
	"go.mau.fi/whatsmeow/types/events"
)

type bot struct {
	*core.Bot
	economy         *economy.Module
	messages        *messageStore
	bmcRounds       sync.Map
	vddAnswered     sync.Map
	quemfalouRounds sync.Map
	quizRounds      sync.Map
	QuizChats       sync.Map
	forcaGames      sync.Map
	termoGames      sync.Map
	impostorGames   sync.Map
	impostorPolls   sync.Map
	velhaGames      sync.Map
	revolvers       sync.Map
	guessRounds     sync.Map
	batataGames     sync.Map
	stopGames       sync.Map
	mentiraChats    sync.Map
	mentiraPolls    sync.Map
	bandeiraRecent  sync.Map
	quizzes         *quizStore
}

type Module = bot

type messageStore struct {
	*core.MessageStore
}

func (b *bot) Register() {
	b.AddCommands(b.gamesCommands)
	b.AddObserver("responder !bmc", b.answerBmc)
	b.AddObserver("responder !vdd", b.answerVdd)
	b.AddObserver("responder !quemfalou", func(ctx context.Context, evt *events.Message) error {
		return b.AnswerPoll(ctx, evt, &b.quemfalouRounds)
	})
	b.AddObserver("responder !quiz", func(ctx context.Context, evt *events.Message) error { return b.AnswerPoll(ctx, evt, &b.quizRounds) })
	b.AddObserver("responder !2verdades1mentira", func(ctx context.Context, evt *events.Message) error { return b.AnswerPoll(ctx, evt, &b.mentiraPolls) })
	b.AddObserver("responder !impostor", func(ctx context.Context, evt *events.Message) error { return b.AnswerPoll(ctx, evt, &b.impostorPolls) })
	b.AddObserver("entrar no !impostor", b.answerImpostor)
	b.AddObserver("responder !forca", b.answerForca)
	b.AddObserver("responder !termo", b.answerTermo)
	b.AddObserver("responder adivinhação", func(_ context.Context, evt *events.Message) error { b.answerGuess(evt); return nil })
	b.AddObserver("responder !velha", b.answerVelha)
	b.AddObserver("responder !batata", b.answerBatata)
	b.AddRestorers(map[string]core.GameRestorer{
		forcaKind:        b.restoreForca,
		termoKind:        b.restoreTermo,
		velhaKind:        b.restoreVelha,
		bmcKind:          b.restoreBmc,
		batataKind:       b.restoreBatata,
		roletaKind:       b.restoreRevolver,
		"quemfalou":      b.restoreQuemfalou,
		"quiz":           b.restoreQuiz,
		"bandeira":       b.restoreBandeira,
		"completa":       b.restoreCompleta,
		impostorGameKind: b.restoreImpostor,
		jokenpoGameKind:  b.restoreJokenpo,
		mentiraGameKind:  b.restoreMentira,
		stopGameKind:     b.restoreStop,
	})
}

func New(c *core.Bot, econ *economy.Module) *Module {
	return &bot{Bot: c, economy: econ, messages: &messageStore{c.Messages}, quizzes: &quizStore{db: c.Messages.DB}}
}
