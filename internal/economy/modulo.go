package economy

import (
	"context"
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
	Wallets        *walletStore
}

type Module = bot

func (b *bot) Register() {
	b.AddCommands(b.economyCommands)
	b.AddOwnerCommands(b.economyOwnerCommands)
	b.AddAfterMessage(b.announcePayouts)
	b.AddReactor(b.reactToAilments)
	b.AddObserver("responder o tribunal", func(ctx context.Context, evt *events.Message) error { return b.AnswerPoll(ctx, evt, &b.trialPolls) })
	b.AddObserver("responder !blackjack", b.answerBlackjack)
	b.AddObserver("responder !crash", b.answerCrash)
	b.AddGuard(b.economyGuard)
	b.AddJob(b.runLottery)
	b.AddJob(b.runTaxes)
	b.AddJob(b.runAuctions)
	b.AddJob(b.runLoanSeizures)
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
	})
}

func (b *bot) economyGuard(ctx context.Context, evt *events.Message, cmd core.Command) bool {
	if cmd.Category.Mode == core.ModeEconomy && !cmd.WhileDead && b.holdIfDead(ctx, evt) {
		return true
	}
	return cmd.BarredInJail && (b.holdIfJailed(ctx, evt) || b.holdIfHostage(ctx, evt))
}

func New(c *core.Bot) *Module {
	return &bot{Bot: c, Wallets: &walletStore{db: c.Messages.DB}}
}
