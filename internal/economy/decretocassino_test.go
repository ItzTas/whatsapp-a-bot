package economy

import (
	"context"
	"errors"
	"math"
	"testing"

	"go.mau.fi/whatsmeow/types"
)

func TestCasinoDecreeLevel(t *testing.T) {
	cases := map[int64]int64{-50: -betTaxDecreeMax, -7: -7, 0: 0, 7: 7, 50: casinoDecreeMaxRaise}
	for stored, want := range cases {
		if got := (decrees{casinoDecree.column: stored}).level(casinoDecree); got != want {
			t.Errorf("decreto cassino gravado como %d vale %d, want %d", stored, got, want)
		}
	}
	tyranny := decrees{casinoDecree.column: -50, tyrannyDecree.column: 1}
	if got := tyranny.level(casinoDecree); got != -tyrantBetTaxMax {
		t.Errorf("taxa do cassino na ditadura gravada como -50 vale %d, want %d", got, -tyrantBetTaxMax)
	}
}

func TestCasinoDecreeBoost(t *testing.T) {
	pb := PrizeBoost{decree: casinoDecreeMaxRaise}

	if got := pb.Pay(1_000, 2_000); got != 2_100 {
		t.Fatalf("Pay de vitória 2x com decreto +10 = %d, want 2100", got)
	}
	if got := pb.Pay(1_000, 0); got != 0 {
		t.Fatalf("Pay de derrota com decreto +10 = %d, want 0", got)
	}
	if got := pb.Risk(stakeRiskFloor); math.Abs(got-0.045) > 1e-9 {
		t.Fatalf("Risk de aposta pequena com decreto +10 = %v, want 0.045", got)
	}
	if tag := pb.Tag(1_000, 2_100); tag != " (📜 +10% por decreto)" {
		t.Fatalf("Tag com decreto +10 = %q", tag)
	}
}

func TestCasinoLevy(t *testing.T) {
	ctx := context.Background()
	chat := types.NewJID("120363000000000001", types.GroupServer)
	s, people := jackpotWallets(t, ctx, chat, "5511900000001", "5511900000002")
	gambler, president := people[0], people[1]
	taxed := presidency{president: president, decrees: decrees{casinoDecree.column: -10}}

	if got := (presidency{president: president, decrees: decrees{casinoDecree.column: 10}}).taxLevy(casinoDecree, gambler).percent; got != 0 {
		t.Fatalf("decreto positivo cobra %d%% de taxa, want 0", got)
	}
	if got := taxed.taxLevy(casinoDecree, president).percent; got != 0 {
		t.Fatalf("o presidente paga %d%% de taxa na própria aposta, want 0", got)
	}

	levy := taxed.taxLevy(casinoDecree, gambler)
	balances := func() (int64, int64) {
		t.Helper()
		g, _, err := s.Balance(ctx, chat, gambler)
		if err != nil {
			t.Fatal(err)
		}
		p, _, err := s.Balance(ctx, chat, president)
		if err != nil {
			t.Fatal(err)
		}
		return g, p
	}

	if _, err := s.SettleBet(ctx, chat, gambler, 1_000, 2_000, levy); err != nil {
		t.Fatal(err)
	}
	if g, p := balances(); g != 50_900 || p != 50_100 {
		t.Fatalf("vitória de 1000 com taxa de 10%%: apostador %d e presidente %d, want 50900 e 50100", g, p)
	}

	if _, err := s.SettleBet(ctx, chat, gambler, 1_000, 0, levy); err != nil {
		t.Fatal(err)
	}
	if g, p := balances(); g != 49_800 || p != 50_200 {
		t.Fatalf("derrota de 1000 com taxa de 10%%: apostador %d e presidente %d, want 49800 e 50200", g, p)
	}

	if _, err := s.SettleBet(ctx, chat, gambler, 49_800, 0, levy); !errors.Is(err, ErrNoFunds) {
		t.Fatalf("aposta que cabe no saldo mas não com a taxa: err = %v, want ErrNoFunds", err)
	}
}
