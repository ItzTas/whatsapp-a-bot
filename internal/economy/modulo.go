package economy

import (
	"context"
	"fmt"
	"sync"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types/events"
)

type bot struct {
	*core.Bot
	blackjackHands sync.Map
	crashRounds    sync.Map
	raceRounds     sync.Map
	itemOffers     sync.Map
	debtRequests   sync.Map
	TrialChats     sync.Map
	trialPolls     sync.Map
	heistCrews     sync.Map
	gangs          sync.Map
	hostageSales   sync.Map
	elections      sync.Map
	impeachments   sync.Map
	coups          sync.Map
	craters        sync.Map
	missionSteps   missionLog
	scenes         sceneBoard
	tilts          tiltBoard
	live           liveBoard
	tips           tipBoard
	Wallets        *walletStore
}

type Module = bot

func (b *bot) Register() {
	b.AddCommands(b.economyCommands)
	b.AddOwnerCommands(b.economyOwnerCommands)
	b.AddAfterMessage(b.settleMissions)
	b.AddAfterMessage(b.announceAssemblies)
	b.AddAfterMessage(b.announcePayouts)
	b.AddAfterMessage(b.offerTip)
	b.AddReactor(b.reactToAilments)
	b.AddObserver("responder o tribunal", func(ctx context.Context, evt *events.Message) error { return b.AnswerPoll(ctx, evt, &b.trialPolls) })
	b.AddObserver("responder !blackjack", b.answerBlackjack)
	b.AddObserver("responder !crash", b.answerCrash)
	b.AddCommandFilter(b.answerCrashCommand)
	b.AddObserver("responder evento", b.answerEvent)
	b.AddObserver("responder enquete ao vivo", b.answerLiveVote)
	b.AddGuard(b.economyGuard)
	b.AddAfterCommand(b.stirLive)
	b.OnModeOff(core.ModeEconomy, b.haltEvents)
	b.OnModeOff(core.ModeEconomy, b.haltLive)
	b.OnModeOff(core.ModeLive, b.haltLive)
	b.AddJob(b.runLottery)
	b.AddJob(b.runAnimalGame)
	b.AddJob(b.runTaxes)
	b.AddJob(b.runAuctions)
	b.AddJob(b.runLoanSeizures)
	b.AddJob(b.runWanted)
	b.AddJob(b.runGravity)
	b.AddJob(b.runSuccession)
	b.AddRestorers(map[string]core.GameRestorer{
		blackjackKind:   b.restoreBlackjack,
		crashKind:       b.restoreCrash,
		raceKind:        b.restoreRace,
		offerKind:       b.restoreOffer,
		debtRequestKind: b.restoreDebtRequest,
		hostageSaleKind: b.restoreHostageSale,
		heistKind:       b.restoreHeist,
		gangKind:        b.restoreGang,
		trialKind:       b.restoreTrial,
		electionKind:    b.restoreElection,
		impeachmentKind: b.restoreImpeachment,
		coupKind:        b.restoreCoup,
		eventKind:       b.restoreEvent,
		craterKind:      b.restoreCrater,
		liveKind:        b.restoreLive,
		liveChatKind:    b.restoreLiveChat,
	})
}

func (b *bot) economyGuard(ctx context.Context, evt *events.Message, cmd core.Command) bool {
	economic := cmd.Category.Mode == core.ModeEconomy || cmd.Category.Mode == core.ModeLive
	if economic && !cmd.WhileDead && b.holdIfDead(ctx, evt) {
		return true
	}
	return cmd.BarredInJail && (b.holdIfJailed(ctx, evt) || b.holdIfHostage(ctx, evt))
}

func New(ctx context.Context, c *core.Bot) (*Module, error) {
	if err := ledgers.load(ctx, c.Messages.DB); err != nil {
		return nil, fmt.Errorf("ler vínculos de economia: %w", err)
	}
	return &bot{Bot: c, Wallets: &walletStore{db: &ledgerDB{db: c.Messages.DB}}}, nil
}
