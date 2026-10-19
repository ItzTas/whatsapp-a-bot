package economy

import (
	"context"
	"testing"

	"go.mau.fi/whatsmeow/types"
)

func TestBetSolace(t *testing.T) {
	cases := map[int64]int64{
		0:              0,
		solaceFloor:    0,
		40_000:         1,
		100_000:        2,
		1_000_000:      4,
		10_000_000:     6,
		10_000_000_000: solaceMax,
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
	if got := solace(); got != 2 {
		t.Fatalf("consolo depois de perder 100k = %d, want 2", got)
	}
	note(70_000)
	if got := solace(); got != 0 {
		t.Fatalf("consolo com 30k de perda líquida = %d, want 0", got)
	}
	note(500_000)
	note(-20_000)
	if got := solace(); got != 0 {
		t.Fatalf("consolo depois de recuperar tudo e perder 20k = %d, want 0", got)
	}
}

func TestPrizeBoostSolace(t *testing.T) {
	pb := PrizeBoost{solace: 4}
	if got := pb.Pay(1_000, 3_000); got != 3_080 {
		t.Fatalf("Pay com 4%% de consolo = %d, want 3080", got)
	}
	if got := pb.Pay(1_000, 0); got != 0 {
		t.Fatalf("Pay numa derrota = %d, want 0", got)
	}
	if got := pb.Tag(1_000, 3_080); got != " (🩹 +4% de consolo)" {
		t.Fatalf("Tag = %q", got)
	}
}
