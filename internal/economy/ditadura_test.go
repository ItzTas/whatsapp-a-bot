package economy

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

func TestTyrannyRevoltBrewsWithTime(t *testing.T) {
	ctx := context.Background()
	chat := types.NewJID("120363000000000001", types.GroupServer)
	s, people := jackpotWallets(t, ctx, chat, "5511900000001")
	president := people[0]
	now := time.Unix(1_800_000_000, 0)
	if err := s.inaugurate(ctx, chat, president, 3, now); err != nil {
		t.Fatal(err)
	}
	if on, set, err := s.flipDecree(ctx, chat, president, tyrannyDecree, now); err != nil || !set || !on {
		t.Fatalf("o !decreto ditadura não ligou: on %v, set %v, err %v", on, set, err)
	}
	d, err := s.decrees(ctx, chat, now)
	if err != nil {
		t.Fatal(err)
	}

	cases := map[time.Duration]float64{
		0:                               revoltBase,
		revoltRiseEvery - time.Second:   revoltBase,
		revoltRiseEvery:                 revoltBase + revoltRise,
		2*revoltRiseEvery + time.Minute: revoltBase + 2*revoltRise,
		termLength:                      revoltKnee + (revoltBase+8*revoltRise-revoltKnee)*revoltSlowdown,
	}
	for after, want := range cases {
		if got := d.revolt(now.Add(after)); math.Abs(got-want) > 1e-9 {
			t.Errorf("revolta %v depois da ditadura = %v, want %v", after, got, want)
		}
	}

	d[revoltColumn] = 100
	if got, want := d.revolt(now), revoltBase+100*revoltPerPoint; math.Abs(got-want) > 1e-9 {
		t.Errorf("revolta com 100 pontos de imposto abusivo = %v, want %v", got, want)
	}
}

func TestTyrantHasNoImmunity(t *testing.T) {
	president := Profession{president: true, decrees: decrees{}}
	if got := president.immunity(); got != presidentImmunity {
		t.Errorf("imunidade do presidente = %v, want %v", got, presidentImmunity)
	}
	tyrant := Profession{president: true, decrees: decrees{tyrannyDecree.column: 1_800_000_000}}
	if got := tyrant.immunity(); got != 0 {
		t.Errorf("imunidade do ditador = %v, want 0", got)
	}
}

func TestCoupOddsIgnoreCurfew(t *testing.T) {
	head := Profession{decrees: decrees{curfewDecree.column: 1_900_000_000}}
	attempt := coupAttempt{head: head, plotters: []robber{{}, {}}}
	if got, want := attempt.odds(), coupChance+coupHelp; math.Abs(got-want) > 1e-9 {
		t.Errorf("chance do golpe com toque de recolher = %v, want %v", got, want)
	}
}

func TestRevoltSlowsDownPastTheKneeWithoutLimit(t *testing.T) {
	cases := map[float64]float64{
		0.2:              0.2,
		revoltKnee:       revoltKnee,
		revoltKnee + 0.2: revoltKnee + 0.2*revoltSlowdown,
		5:                revoltKnee + (5-revoltKnee)*revoltSlowdown,
	}
	for raw, want := range cases {
		if got := easedRevolt(raw); math.Abs(got-want) > 1e-9 {
			t.Errorf("revolta bruta %v vale %v, want %v", raw, got, want)
		}
	}
}

func TestRevoltLinesShowLevelAndNextRise(t *testing.T) {
	start := time.Unix(1_800_000_000, 0)
	d := decrees{tyrannyDecree.column: start.Unix()}

	lines := strings.Join(d.revoltLines(start.Add(5*time.Minute)), "\n")
	for _, want := range []string{"▓▓░░░░░░░░ 10%", "Derruba 10% da segurança", "Sobe pra 15% em 10min"} {
		if !strings.Contains(lines, want) {
			t.Errorf("revolta no começo da ditadura sem %q:\n%s", want, lines)
		}
	}
	if strings.Contains(lines, "Vem de") || strings.Contains(lines, "passou de") {
		t.Errorf("revolta baixa e sem imposto abusivo mostra origem ou virada:\n%s", lines)
	}

	d[revoltColumn] = 100
	lines = strings.Join(d.revoltLines(start), "\n")
	if !strings.Contains(lines, "Vem de 10% do tempo de ditadura e 5% de imposto cobrado acima do teto\n") {
		t.Errorf("revolta com imposto abusivo sem a origem:\n%s", lines)
	}

	lines = strings.Join(d.revoltLines(start.Add(termLength)), "\n")
	for _, want := range []string{"▓▓▓▓▓▓▓▓▓▓ 48% (passou de 40%: agora sobe mais devagar, mas não para)", "e o que passa de 40% conta só 50%", "Sobe pra 50% em 15min"} {
		if !strings.Contains(lines, want) {
			t.Errorf("revolta passando de 40%% sem %q:\n%s", want, lines)
		}
	}
}

func TestDecreedTaxWarnsAboutRevolt(t *testing.T) {
	tyrant := core.Account{Key: "5511900000001@s.whatsapp.net"}
	cases := map[string]struct {
		levy PixLevy
		want string
	}{
		"dentro do teto":            {PixLevy{president: tyrant, percent: 20}, "por decreto"},
		"acima do teto":             {PixLevy{president: tyrant, percent: 70, excess: 40}, "por decreto da ditadura, e o povo ficou mais revoltado 🔥"},
		"acima do teto sem ditador": {PixLevy{percent: 70, excess: 40}, "por decreto"},
	}
	for name, c := range cases {
		if got := c.levy.decreed(); got != c.want {
			t.Errorf("%s: %q, want %q", name, got, c.want)
		}
	}
}
