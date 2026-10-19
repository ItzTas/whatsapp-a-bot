package economy

import (
	"math"
	"testing"
	"time"
)

func TestWindfallOdds(t *testing.T) {
	cases := map[int64]float64{
		windfallFloor:    windfallChance,
		windfallRiseFrom: windfallChance,
		75_000:           0.45,
		windfallRiseTo:   windfallChanceMax,
		1_000_000:        windfallChanceMax,
	}
	for haul, want := range cases {
		if got := windfallOdds(haul); math.Abs(got-want) > 1e-9 {
			t.Errorf("windfallOdds(%d) = %v, queria %v", haul, got, want)
		}
	}
}

func TestWindfallStruck(t *testing.T) {
	cases := []struct {
		name   string
		base   int64
		gain   int64
		haul   int64
		struck bool
	}{
		{"pobre abaixo do piso", 10_000, 4_000, 0, false},
		{"pobre no piso", 10_000, windfallFloor, windfallFloor, true},
		{"médio abaixo da porcentagem", 100_000, 29_000, 0, false},
		{"médio na porcentagem", 100_000, 30_000, 30_000, true},
		{"rico abaixo do teto", 518_756, 49_000, 0, false},
		{"rico no teto", 518_756, 82_500, 82_500, true},
	}
	now := time.Now()
	for _, c := range cases {
		var w windfallBoard
		w.struck("k", c.base, now)
		haul, struck := w.struck("k", c.base+c.gain, now.Add(time.Minute))
		if haul != c.haul || struck != c.struck {
			t.Errorf("%s: struck = (%d, %v), queria (%d, %v)", c.name, haul, struck, c.haul, c.struck)
		}
	}
}
