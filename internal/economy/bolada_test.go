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
	fading := windfallWindow + windfallFade/2
	cases := []struct {
		name   string
		base   int64
		gain   int64
		after  time.Duration
		haul   int64
		struck bool
	}{
		{"pobre abaixo do piso", 10_000, 4_000, time.Minute, 0, false},
		{"pobre no piso", 10_000, windfallFloor, time.Minute, windfallFloor, true},
		{"médio abaixo da porcentagem", 100_000, 29_000, time.Minute, 0, false},
		{"médio na porcentagem", 100_000, 30_000, time.Minute, 30_000, true},
		{"rico abaixo do teto", 518_756, 49_000, time.Minute, 0, false},
		{"rico no teto", 518_756, 82_500, time.Minute, 82_500, true},
		{"ganho inteiro no fim da janela", 100_000, 30_000, windfallWindow, 30_000, true},
		{"ganho pela metade no meio do sumiço", 100_000, 80_000, fading, 40_000, true},
		{"metade do ganho abaixo da porcentagem", 100_000, 50_000, fading, 0, false},
		{"ganho depois do sumiço", 100_000, 80_000, windfallWindow + windfallFade, 0, false},
	}
	now := time.Now()
	for _, c := range cases {
		var w windfallBoard
		w.struck("k", c.base, now)
		haul, struck := w.struck("k", c.base+c.gain, now.Add(c.after))
		if haul != c.haul || struck != c.struck {
			t.Errorf("%s: struck = (%d, %v), queria (%d, %v)", c.name, haul, struck, c.haul, c.struck)
		}
	}
}

func TestWindfallKept(t *testing.T) {
	cases := map[time.Duration]float64{
		0:                                 1,
		windfallWindow:                    1,
		windfallWindow + windfallFadeStep: 5.0 / 6,
		windfallWindow + windfallFade/2:   0.5,
		windfallWindow + windfallFade - time.Minute: 1.0 / 6,
		windfallWindow + windfallFade:               0,
	}
	for age, want := range cases {
		if got := windfallKept(age); math.Abs(got-want) > 1e-9 {
			t.Errorf("windfallKept(%v) = %v, queria %v", age, got, want)
		}
	}
}

func TestWindfallBite(t *testing.T) {
	cases := map[int64]float64{
		windfallFloor:    1,
		windfallRiseFrom: 1,
		225_000:          2,
		windfallBiteTo:   windfallBiteMax,
		2_000_000:        windfallBiteMax,
	}
	for haul, want := range cases {
		if got := windfallBite(haul); math.Abs(got-want) > 1e-9 {
			t.Errorf("windfallBite(%d) = %v, queria %v", haul, got, want)
		}
	}
}

func TestSpotBite(t *testing.T) {
	cases := []struct {
		spot eventSpot
		want float64
	}{
		{eventSpot{place: windfallPlace, haul: windfallBiteTo}, windfallBiteMax},
		{eventSpot{place: stalkPlace, haul: windfallBiteTo}, windfallBiteMax},
		{eventSpot{place: "trabalhar", haul: windfallBiteTo}, 1},
	}
	for _, c := range cases {
		if got := c.spot.bite(); got != c.want {
			t.Errorf("bite(%+v) = %v, queria %v", c.spot, got, c.want)
		}
	}
}

func TestWindfallStalk(t *testing.T) {
	var w windfallBoard
	now := time.Now()
	steps := []struct {
		name  string
		key   string
		after time.Duration
		want  bool
	}{
		{"o primeiro evento de alvo fácil sai", "a", 0, true},
		{"outro antes do intervalo não sai", "a", stalkEvery - time.Minute, false},
		{"outra pessoa no mesmo intervalo tem o dela", "b", time.Minute, true},
		{"depois do intervalo sai de novo", "a", stalkEvery, true},
	}
	for _, s := range steps {
		if got := w.stalk(s.key, now.Add(s.after)); got != s.want {
			t.Errorf("%s: stalk = %v, queria %v", s.name, got, s.want)
		}
	}
}

func TestEasyMarkHaul(t *testing.T) {
	cases := map[string]struct {
		mark easyMark
		want int64
	}{
		"sem marca":            {easyMark{}, 0},
		"prêmio fresco":        {easyMark{jackpot: 12_000}, 12_000},
		"lucro alto em 3 dias": {easyMark{lucky: true, won: 10_500}, 10_500},
	}
	for name, c := range cases {
		if got := c.mark.haul(); got != c.want {
			t.Errorf("%s: haul = %d, queria %d", name, got, c.want)
		}
	}
}
