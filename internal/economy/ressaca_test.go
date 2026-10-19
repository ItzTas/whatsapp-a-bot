package economy

import (
	"math"
	"testing"
)

func TestHangoverGrowsWithWinnings(t *testing.T) {
	cases := []struct {
		won  int64
		want float64
	}{
		{0, 0},
		{-50_000, 0},
		{hangoverFloor - 1, 0},
		{hangoverFloor, hangoverStep},
		{hangoverFloor * 10, 2 * hangoverStep},
		{hangoverFloor * 1_000_000, hangoverMax},
	}
	for _, c := range cases {
		if got := hangoverOf(c.won); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("hangoverOf(%d) = %v, quero %v", c.won, got, c.want)
		}
	}
}

func TestHangoverSparesGod(t *testing.T) {
	if p := god.hungover(hangoverFloor * 1_000); p.hangover != 0 {
		t.Errorf("deus ficou com hangover %v", p.hangover)
	}
}

func TestSoberCutsAShareOfTheChance(t *testing.T) {
	p := Profession{hangover: 0.1}
	cases := []struct{ chance, want float64 }{
		{0.6, 0.54},
		{0.1, 0.09},
		{0, 0},
	}
	for _, c := range cases {
		if got := p.sober(c.chance); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("sober(%v) = %v, quero %v", c.chance, got, c.want)
		}
	}
}

func TestShakyRaisesAShareOfTheRisk(t *testing.T) {
	p := Profession{hangover: 0.1}
	cases := []struct{ chance, want float64 }{
		{0, 0},
		{0.2, 0.22},
		{0.9, 0.95},
		{0.97, 0.97},
	}
	for _, c := range cases {
		if got := p.shaky(c.chance); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("shaky(%v) = %v, quero %v", c.chance, got, c.want)
		}
	}
}

func TestCrewHangoverTakesTheWorstMember(t *testing.T) {
	crew := []robber{{job: Profession{hangover: 0.05}}, {job: Profession{hangover: 0.1}}, {}}
	got := crewHangover(crew, func(r robber) Profession { return r.job }).hangover
	if math.Abs(got-0.1) > 1e-9 {
		t.Errorf("crewHangover = %v, quero 0.1", got)
	}
}
