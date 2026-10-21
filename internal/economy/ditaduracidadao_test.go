package economy

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

func TestTyrannyShieldsCitizens(t *testing.T) {
	tyranny := decrees{tyrannyDecree.column: 1_800_000_000}
	cases := map[string]struct {
		job      Profession
		escape   float64
		immunity float64
	}{
		"cidadão com a revolta cheia": {Profession{decrees: tyranny, uprising: 1}, escapeChance + tyrantEscapeBonus, tyrantCitizenImmunity},
		"cidadão sem revolta":         {Profession{decrees: tyranny}, escapeChance, 0},
		"ditador":                     {Profession{president: true, decrees: tyranny, uprising: 1}, escapeChance, 0},
		"cidadão sem ditadura":        {Profession{decrees: decrees{}, uprising: 1}, escapeChance, 0},
		"presidente sem ditadura":     {Profession{president: true, decrees: decrees{}}, escapeChance, presidentImmunity},
	}
	for name, c := range cases {
		if got := c.job.escapeOdds(); math.Abs(got-c.escape) > 1e-9 {
			t.Errorf("%s: chance de fugir = %v, want %v", name, got, c.escape)
		}
		if got := c.job.immunity(); math.Abs(got-c.immunity) > 1e-9 {
			t.Errorf("%s: imunidade = %v, want %v", name, got, c.immunity)
		}
	}
}

func TestCitizensFeelTheRevolt(t *testing.T) {
	start := time.Unix(1_800_000_000, 0)
	tyranny := decrees{tyrannyDecree.column: start.Unix()}
	early := Profession{decrees: tyranny}.roused(start)
	late := Profession{decrees: tyranny}.roused(start.Add(termLength))
	if early.immunity() <= 0 || early.escapeOdds() <= escapeChance {
		t.Errorf("no começo da ditadura o povo já tem alguma vantagem: imunidade %v, fuga %v", early.immunity(), early.escapeOdds())
	}
	if late.immunity() <= early.immunity() || late.escapeOdds() <= early.escapeOdds() {
		t.Errorf("com mais revolta a vantagem cresce: imunidade %v → %v, fuga %v → %v", early.immunity(), late.immunity(), early.escapeOdds(), late.escapeOdds())
	}
	tyrant := Profession{president: true, decrees: tyranny}.roused(start.Add(termLength))
	if tyrant.uprising != 0 || tyrant.tyrantEscape() != 0 {
		t.Errorf("o ditador não ganha nada com a revolta do povo: revolta %v, fuga %v", tyrant.uprising, tyrant.tyrantEscape())
	}
}

func TestTyrannyLengthensCitizenShield(t *testing.T) {
	ctx := context.Background()
	chat := types.NewJID("120363000000000001", types.GroupServer)
	s, people := jackpotWallets(t, ctx, chat, "5511900000001", "5511900000002", "5511900000003", "5511900000004")
	president, citizen := people[0], people[1]
	now := time.Unix(1_800_000_000, 0)
	if err := s.inaugurate(ctx, chat, president, 3, now); err != nil {
		t.Fatal(err)
	}
	if on, set, err := s.flipDecree(ctx, chat, president, tyrannyDecree, now); err != nil || !set || !on {
		t.Fatalf("o !decreto ditadura não ligou: on %v, set %v, err %v", on, set, err)
	}
	succeed := func(int64, int64, easyMark) (bool, int64) { return true, 1_000 }
	cases := map[string]struct {
		thief  core.Account
		victim core.Account
		shield time.Duration
	}{
		"cidadão": {people[2], citizen, robShield + tyrantShelter},
		"ditador": {people[3], president, robShield},
	}
	for name, c := range cases {
		cd := cooldown{column: "roubo_em", every: robEvery}
		r, err := s.rob(ctx, chat, c.thief, c.victim, cd, nil, crimeRisk{}, flagrant{}, now, robMinVictim, succeed)
		if err != nil || !r.success {
			t.Fatalf("%s: o roubo devia dar certo, deu %+v (%v)", name, r, err)
		}
		left, err := s.shieldLeft(ctx, chat, c.victim, now)
		if err != nil {
			t.Fatal(err)
		}
		if left != c.shield {
			t.Errorf("%s: escudo = %v, want %v", name, left, c.shield)
		}
	}
}
