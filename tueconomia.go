package main

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"golang.org/x/sync/errgroup"
)

type economyProfile struct {
	balance       int64
	rank          int
	titleID       string
	job           profession
	saved         savings
	hasSavings    bool
	joint         int64
	stocks        int64
	held          []stack
	owes          []debt
	lent          []debt
	bounty        int64
	jailLeft      time.Duration
	deathLeft     time.Duration
	captive       captivity
	bodyguardLeft time.Duration
	bribeLeft     time.Duration
}

func (b *bot) economyProfile(ctx context.Context, chat types.JID, person types.JID) (e economyProfile, err error) {
	acc := b.account(ctx, person)
	now := time.Now()
	var (
		jointByPartner map[string]int64
		quotes         []stockQuote
		positions      []stockPosition
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { e.balance, e.rank, err = b.wallets.balance(gctx, chat, acc); return err })
	g.Go(func() (err error) { e.titleID, err = b.wallets.titleOf(gctx, chat, acc); return err })
	g.Go(func() (err error) { e.job, err = b.professionOf(gctx, chat, acc); return err })
	g.Go(func() (err error) { e.saved, e.hasSavings, err = b.wallets.savings(gctx, chat, acc); return err })
	g.Go(func() (err error) { jointByPartner, err = b.wallets.jointBalances(gctx, chat, acc); return err })
	g.Go(func() (err error) { positions, err = b.wallets.stockPositions(gctx, chat, acc); return err })
	g.Go(func() (err error) { quotes, err = b.wallets.stockMarket(gctx, chat, now); return err })
	g.Go(func() (err error) { e.held, err = b.wallets.inventory(gctx, chat, acc); return err })
	g.Go(func() (err error) { e.owes, e.lent, err = b.wallets.debtsOf(gctx, chat, acc, now); return err })
	g.Go(func() (err error) { e.bounty, err = b.wallets.bountyTotal(gctx, chat, acc); return err })
	g.Go(func() (err error) { e.jailLeft, err = b.wallets.sentenceLeft(gctx, chat, acc, now); return err })
	g.Go(func() (err error) { e.deathLeft, err = b.wallets.deathLeft(gctx, chat, acc, now); return err })
	g.Go(func() (err error) { e.captive, err = b.wallets.captivity(gctx, chat, acc, now); return err })
	g.Go(func() (err error) {
		e.bodyguardLeft, err = b.wallets.leftUntil(gctx, chat, acc, "seguranca_ate", now)
		return err
	})
	g.Go(func() (err error) {
		e.bribeLeft, err = b.wallets.leftUntil(gctx, chat, acc, "propina_ate", now)
		return err
	})
	if err := g.Wait(); err != nil {
		return economyProfile{}, fmt.Errorf("ler economia: %w", err)
	}

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

func (e economyProfile) lines() []string {
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
	lines = append(lines, e.assetLines()...)
	lines = append(lines, e.debtLines()...)
	lines = append(lines, e.statusLines()...)
	return lines
}

func (e economyProfile) walletLine() string {
	if e.balance == 0 {
		return "🪹 Liso, sem nenhuma moeda"
	}
	return fmt.Sprintf("💰 %s (%dº mais rico do grupo)", coins(e.balance), e.rank)
}

func (e economyProfile) assetLines() []string {
	var lines []string
	if e.hasSavings {
		lines = append(lines, fmt.Sprintf("🔒 Poupança: %s, que voltam com %s de juros", coins(e.saved.amount), coins(e.saved.interest)))
	}
	if e.joint > 0 {
		lines = append(lines, "💑 Conta conjunta: "+coins(e.joint))
	}
	if e.stocks > 0 {
		lines = append(lines, "📊 Ações valendo "+coins(e.stocks))
	}
	var units, worth int64
	for _, s := range e.held {
		units += s.quantity
		worth += s.worth()
	}
	if units > 0 {
		lines = append(lines, fmt.Sprintf("🎒 Mochila: %s, valendo %s", countLabel(int(units), "item", "itens"), coins(worth)))
	}
	if weapon, ok := itemByID(e.job.weapon); ok {
		lines = append(lines, "🔫 Arma: "+weapon.label())
	}
	return lines
}

func (e economyProfile) debtLines() []string {
	var lines []string
	if len(e.owes) > 0 {
		lines = append(lines, fmt.Sprintf("🦈 Deve %s no agiota%s", coins(debtTotal(e.owes)), overdueNote(e.owes)))
	}
	if len(e.lent) > 0 {
		lines = append(lines, fmt.Sprintf("🤝 Tem %s pra receber de empréstimo%s", coins(debtTotal(e.lent)), overdueNote(e.lent)))
	}
	if e.bounty > 0 {
		lines = append(lines, "🎯 Recompensa pela cabeça: "+coins(e.bounty))
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

func (e economyProfile) statusLines() []string {
	var lines []string
	if e.jailLeft > 0 {
		lines = append(lines, "⛓️ Tá preso, sai em "+formatWait(e.jailLeft))
	}
	if e.deathLeft > 0 {
		lines = append(lines, "💀 Tá morto, ressuscita em "+formatWait(e.deathLeft))
	}
	if e.captive.left > 0 {
		lines = append(lines, "🔗 Tá em cativeiro por mais "+formatWait(e.captive.left))
	}
	if e.bodyguardLeft > 0 {
		lines = append(lines, "🕶️ Com segurança particular por mais "+formatWait(e.bodyguardLeft))
	}
	if e.bribeLeft > 0 {
		lines = append(lines, "💵 Com a polícia subornada por mais "+formatWait(e.bribeLeft))
	}
	if drug, ok := itemByID(e.job.drug); ok {
		lines = append(lines, "😵 Sob efeito de "+drug.label())
	}
	if len(e.job.charms) > 0 {
		lines = append(lines, "✨ Encantos: "+charmNames(e.job.charms))
	}
	return lines
}
