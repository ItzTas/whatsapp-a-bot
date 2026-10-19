package economy

import (
	"context"
	"math"
	"testing"

	"go.mau.fi/whatsmeow/types"
)

func TestBetSolace(t *testing.T) {
	cases := map[int64]int64{
		0:               0,
		solaceFloor - 1: 0,
		solaceFloor:     3,
		40_000:          4,
		100_000:         6,
		1_000_000:       solaceMax,
		10_000_000_000:  solaceMax,
	}
	for lost, want := range cases {
		if got := betSolace(lost); got != want {
			t.Errorf("betSolace(%d) = %d, want %d", lost, got, want)
		}
	}
}

func TestBetRecordSolace(t *testing.T) {
	ctx := context.Background()
	chat := types.NewJID("120363000000000001", types.GroupServer)
	s, people := jackpotWallets(t, ctx, chat, "5511900000001")
	gambler := people[0]
	solace := func() int64 {
		t.Helper()
		_, got, err := s.betRecord(ctx, chat, gambler)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	note := func(net int64) {
		t.Helper()
		if err := s.noteStreak(ctx, chat, gambler, net); err != nil {
			t.Fatal(err)
		}
	}

	note(-60_000)
	note(-40_000)
	if got := solace(); got != 6 {
		t.Fatalf("maré de azar depois de perder 100k = %d, want 6", got)
	}
	note(70_000)
	if got := solace(); got != 4 {
		t.Fatalf("maré de azar com 30k de perda líquida = %d, want 4", got)
	}
	note(500_000)
	if got := solace(); got != 0 {
		t.Fatalf("maré de azar depois de recuperar tudo = %d, want 0", got)
	}
}

func TestPrizeBoostEdge(t *testing.T) {
	near := func(got, want float64) bool { return math.Abs(got-want) < 1e-9 }
	unlucky := PrizeBoost{solace: 6}

	if got := unlucky.Favor(stakeRiskFloor); !near(got, 0.06) {
		t.Fatalf("Favor de aposta pequena com 6 de maré = %v, want 0.06", got)
	}
	if got := unlucky.Risk(stakeRiskFloor); got != 0 {
		t.Fatalf("Risk de quem tá a favor = %v, want 0", got)
	}

	big := unlucky.Risk(1_000_000)
	if !near(big, stakeRisk(1_000_000)-0.06) || unlucky.Favor(1_000_000) != 0 {
		t.Fatalf("aposta de 1M com 6 de maré: Risk = %v, Favor = %v", big, unlucky.Favor(1_000_000))
	}

	hot := PrizeBoost{heat: 9, solace: 6}
	if got := hot.Risk(stakeRiskFloor); !near(got, 0.03) {
		t.Fatalf("Risk com 9 de calor e 6 de maré = %v, want 0.03", got)
	}
}

func TestSolaceNeverCoversStakingTheLoss(t *testing.T) {
	for _, lost := range []int64{solaceFloor, 40_000, 100_000, 1_000_000, 10_000_000} {
		pb := PrizeBoost{solace: betSolace(lost)}
		if pb.Favor(lost) != 0 || pb.Risk(lost) == 0 {
			t.Errorf("apostar os %d perdidos de uma vez: Risk = %v, Favor = %v", lost, pb.Risk(lost), pb.Favor(lost))
		}
	}
}
