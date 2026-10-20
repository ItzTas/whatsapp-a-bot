package economy

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

func TestBetRule(t *testing.T) {
	cases := []struct {
		balance, spent, most, whole int64
	}{
		{1, 0, 1, 1},
		{2, 0, 1, 1},
		{10, 0, 7, 6},
		{1_000, 0, 700, 600},
		{700_000, 0, 490_000, 420_000},
		{800_000, 0, betWindowCap, 480_000},
		{50_000_000, 0, betWindowCap, betWindowCap},
		{1_000_000, 300_000, 200_000, 200_000},
		{10_000, 495_000, 5_000, 5_000},
		{1_000_000, betWindowCap, 0, 0},
	}
	for _, c := range cases {
		r := betRule(c.spent, time.Hour, nil)
		if got := r.most(c.balance); got != c.most {
			t.Errorf("most(saldo %d, apostado %d) = %d, want %d", c.balance, c.spent, got, c.most)
		}
		if got := r.whole(c.balance); got != c.whole {
			t.Errorf("whole(saldo %d, apostado %d) = %d, want %d", c.balance, c.spent, got, c.whole)
		}
	}
}

func TestBetRuleRefusals(t *testing.T) {
	cases := []struct {
		rule    stakeRule
		balance int64
		want    string
	}{
		{betRule(0, 0, nil), 1_000, "só aceita até 70% do saldo"},
		{betRule(0, 0, []betOutcome{{net: 2_000, leaves: time.Hour}}), 1_000, "só aceita até 70% do saldo"},
		{betRule(400_000, 0, nil), 1_000_000, "já apostou 🪙 400.000 moedas, e o limite é 🪙 500.000 moedas"},
		{betRule(betWindowCap, 30*time.Minute, nil), 1_000_000, "Dá pra apostar de novo daqui a 30min"},
		{betRule(0, 0, []betOutcome{{net: -5_000, leaves: time.Hour}}), 5_000, "já perdeu 🪙 5.000 moedas em aposta nas últimas 2h, e a casa não deixa ninguém perder mais de 70% do que tinha nesse tempo. Agora dá pra apostar no máximo 🪙 2.000 moedas"},
		{betRule(0, 0, []betOutcome{{net: -7_000, leaves: time.Hour}}), 3_000, "perder mais de 70% do que tinha nesse tempo. Dá pra apostar de novo daqui a 1h"},
	}
	for _, c := range cases {
		if got := c.rule.tooMuch(c.balance); !strings.Contains(got, c.want) {
			t.Errorf("tooMuch(apostado %d, saldo %d) = %q, want contendo %q", c.rule.spent, c.balance, got, c.want)
		}
	}
}

func TestStopLoss(t *testing.T) {
	cases := []struct {
		name     string
		balance  int64
		outcomes []betOutcome
		most     int64
		reopens  time.Duration
	}{
		{"sem aposta recente", 10_000, nil, 7_000, 0},
		{"perdeu metade do que tinha", 5_000, []betOutcome{{net: -5_000, leaves: time.Hour}}, 2_000, 0},
		{"perdeu 70% do que tinha", 3_000, []betOutcome{{net: -7_000, leaves: time.Hour}}, 0, time.Hour},
		{"perdeu tudo", 0, []betOutcome{{net: -10_000, leaves: time.Hour}}, 0, time.Hour},
		{"ganhou antes de perder", 3_000, []betOutcome{{net: 4_000, leaves: time.Hour}, {net: -7_000, leaves: 2 * time.Hour}}, 1_200, 0},
		{"reabre quando a perda grande sai da janela", 3_000, []betOutcome{{net: -6_000, leaves: time.Hour}, {net: -1_000, leaves: 90 * time.Minute}}, 0, time.Hour},
		{"reabre só quando sai a perda que segura", 2_000, []betOutcome{{net: -500, leaves: 10 * time.Minute}, {net: -7_500, leaves: 90 * time.Minute}}, 0, 90 * time.Minute},
		{"saldo novo depois da perda", 13_000, []betOutcome{{net: -7_000, leaves: time.Hour}}, 7_000, 0},
		{"lucro na janela não aperta", 10_000, []betOutcome{{net: 50_000, leaves: time.Hour}}, 7_000, 0},
	}
	for _, c := range cases {
		r := betRule(0, 0, c.outcomes)
		if got := r.most(c.balance); got != c.most {
			t.Errorf("%s: most(saldo %d) = %d, want %d", c.name, c.balance, got, c.most)
		}
		if c.most > 0 {
			continue
		}
		if got := r.reopens(c.balance); got != c.reopens {
			t.Errorf("%s: reopens(saldo %d) = %v, want %v", c.name, c.balance, got, c.reopens)
		}
	}
}

func TestBetOutcomesWindow(t *testing.T) {
	ctx := context.Background()
	chat := types.NewJID("120363000000000001", types.GroupServer)
	s, people := jackpotWallets(t, ctx, chat, "5511900000001", "5511900000002")
	gambler, other := people[0], people[1]
	now := time.Now()
	for _, r := range []struct {
		acc core.Account
		net int64
		at  time.Duration
	}{
		{gambler, -9_000, -stopLossWindow - time.Minute},
		{gambler, -3_000, -stopLossWindow + time.Hour},
		{gambler, 1_000, -time.Hour},
		{other, -50_000, -time.Hour},
	} {
		if err := s.noteRecentWinnings(ctx, chat, r.acc, r.net, now.Add(r.at)); err != nil {
			t.Fatal(err)
		}
	}
	outcomes, err := s.betOutcomes(ctx, chat, gambler, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(outcomes) != 2 || outcomes[0].net != -3_000 || outcomes[1].net != 1_000 {
		t.Fatalf("apostas da janela = %+v, want a perda de 3000 e depois o ganho de 1000", outcomes)
	}
	if leaves := outcomes[0].leaves; leaves < 59*time.Minute || leaves > 61*time.Minute {
		t.Fatalf("a perda de 1h atrás sai da janela em %v, want ~1h", leaves)
	}
}

func TestBidRuleTakesWholeBalance(t *testing.T) {
	for _, balance := range []int64{1, 1_000, 50_000_000} {
		if most, whole := bidRule.most(balance), bidRule.whole(balance); most != balance || whole != balance {
			t.Errorf("bidRule com saldo %d: most = %d, whole = %d", balance, most, whole)
		}
	}
}

func TestWageredWindow(t *testing.T) {
	ctx := context.Background()
	chat := types.NewJID("120363000000000001", types.GroupServer)
	s, people := jackpotWallets(t, ctx, chat, "5511900000001")
	gambler := people[0]
	now := time.Now()
	wager := func(stake int64, at time.Time) {
		t.Helper()
		if err := s.inTx(ctx, func(tx *ledgerTx) error { return noteWager(ctx, tx, chat, gambler, stake, at) }); err != nil {
			t.Fatal(err)
		}
	}

	wager(100_000, now.Add(-betWindow-time.Minute))
	wager(30_000, now.Add(-90*time.Minute))
	if _, err := s.SettleBet(ctx, chat, gambler, 20_000, 0, PixLevy{}); err != nil {
		t.Fatal(err)
	}
	spent, freed, err := s.wagered(ctx, chat, gambler, now)
	if err != nil {
		t.Fatal(err)
	}
	if spent != 50_000 {
		t.Fatalf("apostado nas últimas 2h = %d, want 50000 (a aposta de fora da janela não conta)", spent)
	}
	if freed < 29*time.Minute || freed > 31*time.Minute {
		t.Fatalf("a aposta mais velha da janela sai em %v, want ~30min", freed)
	}
}
