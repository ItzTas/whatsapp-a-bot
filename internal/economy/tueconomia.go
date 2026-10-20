package economy

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"golang.org/x/sync/errgroup"
)

type EconomyProfile struct {
	balance    int64
	rank       int
	titleID    string
	job        Profession
	saved      Savings
	hasSavings bool
	joint      int64
	stocks     int64
	held       []stack
	owes       []debt
	lent       []debt
	bounty     int64
	jailLeft   time.Duration
	deathLeft  time.Duration
	health     int64
	captive    captivity
	shield     time.Duration
	contracts  []contract
	now        time.Time
}

func (b *bot) EconomyProfile(ctx context.Context, chat types.JID, person types.JID) (e EconomyProfile, err error) {
	acc := b.Account(ctx, person)
	now := time.Now()
	var (
		jointByPartner map[string]int64
		quotes         []stockQuote
		positions      []stockPosition
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { e.balance, e.rank, err = b.Wallets.Balance(gctx, chat, acc); return err })
	g.Go(func() (err error) { e.titleID, err = b.Wallets.titleOf(gctx, chat, acc); return err })
	g.Go(func() (err error) { e.job, err = b.ProfessionOf(gctx, chat, acc); return err })
	g.Go(func() (err error) { e.saved, e.hasSavings, err = b.Wallets.Savings(gctx, chat, acc); return err })
	g.Go(func() (err error) { jointByPartner, err = b.Wallets.jointBalances(gctx, chat, acc); return err })
	g.Go(func() (err error) { positions, err = b.Wallets.stockPositions(gctx, chat, acc); return err })
	g.Go(func() (err error) { quotes, err = b.Wallets.stockMarket(gctx, chat, now); return err })
	g.Go(func() (err error) { e.held, err = b.Wallets.inventory(gctx, chat, acc); return err })
	g.Go(func() (err error) { e.owes, e.lent, err = b.Wallets.debtsOf(gctx, chat, acc, now); return err })
	g.Go(func() (err error) { e.bounty, err = b.Wallets.bountyTotal(gctx, chat, acc); return err })
	g.Go(func() (err error) { e.jailLeft, err = b.Wallets.SentenceLeft(gctx, chat, acc, now); return err })
	g.Go(func() (err error) { e.deathLeft, err = b.Wallets.DeathLeft(gctx, chat, acc, now); return err })
	g.Go(func() (err error) { e.health, err = b.Wallets.Health(gctx, chat, acc, now); return err })
	g.Go(func() (err error) { e.captive, err = b.Wallets.captivity(gctx, chat, acc, now); return err })
	g.Go(func() (err error) { e.shield, err = b.Wallets.shieldLeft(gctx, chat, acc, now); return err })
	g.Go(func() (err error) { e.contracts, err = b.Wallets.contracts(gctx, chat, acc, now); return err })
	if err := g.Wait(); err != nil {
		return EconomyProfile{}, fmt.Errorf("ler economia: %w", err)
	}

	e.now = now
	for balance := range maps.Values(jointByPartner) {
		e.joint += balance
	}
	prices := make(map[string]int64, len(quotes))
	for _, q := range quotes {
		prices[q.company.ticker] = q.cents
	}
	for _, p := range positions {
		e.stocks += p.shares * prices[p.ticker] / 100
	}
	return e, nil
}

func (e EconomyProfile) Lines() []string {
	lines := []string{"", "💼 *Economia*", e.walletLine(), "Profissão: " + e.job.label()}
	if e.job.president {
		lines = append(lines, "🏛️ Presidente do grupo")
	}
	if badge := strings.TrimSpace(titleBadge(e.titleID)); badge != "" {
		lines = append(lines, "🎖️ Título: "+badge)
	}
	if home := homeText(e.job.estate); home != "" {
		lines = append(lines, home)
	}
	if fame := fameLine(e.job); fame != "" {
		lines = append(lines, fame)
	}
	lines = append(lines, e.assetLines()...)
	lines = append(lines, e.debtLines()...)
	lines = append(lines, e.statusLines()...)
	return lines
}

func (e EconomyProfile) walletLine() string {
	if e.balance == 0 {
		return "🪹 Liso, sem nenhuma moeda"
	}
	return fmt.Sprintf("💰 %s (%dº mais rico do grupo)", Coins(e.balance), e.rank)
}

func (e EconomyProfile) assetLines() []string {
	var lines []string
	if exposure := exposureText(e.balance); exposure != "" {
		lines = append(lines, exposure)
	}
	lines = append(lines, cleanText(e.job))
	if e.shield > 0 {
		lines = append(lines, shieldText(e.shield))
	}
	if text := contractText(e.contracts, e.now); text != "" {
		lines = append(lines, text)
	}
	if e.hasSavings {
		lines = append(lines, fmt.Sprintf("🔒 Poupança: %s, que voltam com %s de juros", Coins(e.saved.Amount), Coins(e.saved.Interest)))
	}
	if e.joint > 0 {
		lines = append(lines, "💑 Conta conjunta: "+Coins(e.joint))
	}
	if e.stocks > 0 {
		lines = append(lines, "📊 Ações valendo "+Coins(e.stocks))
	}
	var units, worth int64
	for _, s := range e.held {
		units += s.quantity
		worth += s.worth()
	}
	if units > 0 {
		lines = append(lines, fmt.Sprintf("🎒 Mochila: %s, valendo %s", core.CountLabel(int(units), "item", "itens"), Coins(worth)))
	}
	if weapon, ok := itemByID(e.job.weapon); ok {
		lines = append(lines, "🔫 Arma: "+weapon.label())
	}
	return lines
}

func (e EconomyProfile) debtLines() []string {
	var lines []string
	if len(e.owes) > 0 {
		lines = append(lines, fmt.Sprintf("🦈 Deve %s no agiota%s", Coins(debtTotal(e.owes)), overdueNote(e.owes)))
	}
	if len(e.lent) > 0 {
		lines = append(lines, fmt.Sprintf("🤝 Tem %s pra receber de empréstimo%s", Coins(debtTotal(e.lent)), overdueNote(e.lent)))
	}
	if e.bounty > 0 {
		lines = append(lines, "🎯 Recompensa pela cabeça: "+Coins(e.bounty))
	}
	return lines
}

func debtTotal(debts []debt) int64 {
	var total int64
	for _, d := range debts {
		total += d.amount
	}
	return total
}

func overdueNote(debts []debt) string {
	now := time.Now()
	for _, d := range debts {
		if d.overdue(now) {
			return " (tem coisa vencida)"
		}
	}
	return ""
}

func (e EconomyProfile) statusLines() []string {
	var lines []string
	if e.jailLeft > 0 {
		lines = append(lines, "⛓️ Tá preso, sai em "+core.FormatWait(e.jailLeft))
	}
	if e.deathLeft > 0 {
		lines = append(lines, "💀 Tá morto, ressuscita em "+core.FormatWait(e.deathLeft))
	}
	if e.health < MaxHealth && e.deathLeft == 0 {
		lines = append(lines, "🩺 Saúde: "+healthText(e.health))
	}
	if e.captive.left > 0 {
		lines = append(lines, "🔗 Tá em cativeiro por mais "+core.FormatWait(e.captive.left))
	}
	if drug, ok := itemByID(e.job.drug); ok {
		lines = append(lines, "😵 Sob efeito de "+drug.label())
	}
	if len(e.job.charms) > 0 {
		lines = append(lines, "✨ Encantos: "+charmNames(e.job.charms))
	}
	return lines
}
