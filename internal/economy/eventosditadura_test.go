package economy

import (
	"context"
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

func TestTyrannyEventsReachTheirAudience(t *testing.T) {
	tyranny := decrees{tyrannyDecree.column: 1_800_000_000}
	audiences := map[tyrannySide]eventCase{
		tyrantSide: {job: Profession{president: true, decrees: tyranny}},
		peopleSide: {job: Profession{decrees: tyranny}},
	}
	outsiders := []eventCase{
		{job: Profession{president: true, decrees: decrees{}}},
		{job: Profession{decrees: decrees{}}},
		{job: Profession{protege: true, decrees: tyranny}},
	}
	for _, sc := range tyrannyEvents.scenarios {
		var side []tyrannySide
		for s, c := range audiences {
			if sc.when != nil && sc.when(c) {
				side = append(side, s)
			}
		}
		if len(side) != 1 {
			t.Errorf("%s: precisa sair só pro ditador ou só pro povo, sai pra %v", sc.id, side)
			continue
		}
		for _, c := range outsiders {
			if sc.when(c) {
				t.Errorf("%s: sai fora da ditadura ou pro favorito (%+v)", sc.id, c.job)
			}
		}
		for _, fx := range scenarioEffects(sc) {
			if s, ok := tyrannySideOf(fx); ok && s != side[0] {
				t.Errorf("%s: efeito %+v é do outro lado da ditadura", sc.id, fx)
			}
		}
	}
	for _, sc := range eventCatalog {
		if sc.book == tyrannyEvents.name {
			continue
		}
		for _, fx := range scenarioEffects(sc.scenario) {
			if _, ok := tyrannySideOf(fx); ok {
				t.Errorf("%s: efeito de ditadura fora do livro da ditadura", sc.id)
			}
		}
	}
}

func scenarioEffects(sc scenario) []effect {
	var effects []effect
	for _, ch := range sc.choices {
		for _, e := range ch.endings {
			effects = append(effects, e.effects...)
		}
	}
	return effects
}

func tyrannySideOf(fx effect) (tyrannySide, bool) {
	switch v := fx.(type) {
	case revoltEffect:
		return v.side, v.lo > 0 && v.hi >= v.lo
	case armyEffect:
		return v.side, v.battalions > 0
	}
	return 0, false
}

func TestTyrannyEffectsFortune(t *testing.T) {
	cases := map[string]struct {
		fx   effect
		want int
	}{
		"povo revolta":     {rebel(1, 2), 1},
		"povo dedura":      {betray(1, 2), -1},
		"povo sabota":      {sabotage(1), 1},
		"ditador revolta":  {outrage(1, 2), -1},
		"ditador reprime":  {quell(1, 2), 1},
		"ditador alista":   {enlist(1), 1},
		"ditador deserção": {desert(1), -1},
	}
	for name, c := range cases {
		if got := c.fx.fortune(); got != c.want {
			t.Errorf("%s: fortune %d, want %d", name, got, c.want)
		}
	}
}

func playTyrannyEffect(t *testing.T, ctx context.Context, s *walletStore, chat types.JID, now time.Time, fx effect) string {
	t.Helper()
	res, err := s.playEvent(ctx, eventRun{chat: chat, acc: tyrannyCitizen, place: "!trabalhar", now: now}, eventPlay{ending: ending{effects: []effect{fx}}})
	if err != nil {
		t.Fatal(err)
	}
	return strings.Join(res.lines, "\n")
}

var tyrannyCitizen = core.Account{Key: "5511900000002@s.whatsapp.net", JID: types.NewJID("5511900000002", types.DefaultUserServer)}

func TestRevoltEffectsMoveTheRevolt(t *testing.T) {
	ctx := context.Background()
	s, chat, _, reign := tyrantArmy(t, ctx, 0)
	revolt := func() float64 {
		t.Helper()
		d, err := s.decrees(ctx, chat, reign)
		if err != nil {
			t.Fatal(err)
		}
		return d.revolt(reign)
	}
	steps := []struct {
		fx     effect
		revolt float64
		line   string
	}{
		{rebel(5, 5), revoltBase + 0.05, "subiu pra 15%"},
		{quell(8, 8), revoltBase - 0.03, "caiu pra 7%"},
		{betray(20, 20), 0, "caiu pra 0%"},
		{quell(1, 1), 0, "a repressão já abafou tudo o que dava"},
		{rebel(60, 60), revoltBase + 0.25, "subiu pra 35%"},
		{outrage(1, 1), revoltBase + 0.25, "os protestos já empurraram tudo o que dava"},
	}
	for _, step := range steps {
		line := playTyrannyEffect(t, ctx, s, chat, reign, step.fx)
		if got := revolt(); math.Abs(got-step.revolt) > 1e-9 {
			t.Errorf("%+v: revolta %v, want %v", step.fx, got, step.revolt)
		}
		if !strings.Contains(line, step.line) {
			t.Errorf("%+v: resposta sem %q: %q", step.fx, step.line, line)
		}
	}
	d, err := s.decrees(ctx, chat, reign)
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Join(d.revoltLines(reign), "\n"); !strings.Contains(lines, "Vem de 10% do tempo de ditadura e 25% dos protestos do povo") {
		t.Errorf("a origem da revolta não mostra os protestos:\n%s", lines)
	}
	d[agitationColumn] = agitationFloor
	if lines := strings.Join(d.revoltLines(reign), "\n"); !strings.Contains(lines, "Vem de 10% do tempo de ditadura e menos 15% que a repressão abafou") {
		t.Errorf("a origem da revolta não mostra a repressão:\n%s", lines)
	}
}

func TestArmyEffectsHireAndLoseBattalions(t *testing.T) {
	ctx := context.Background()
	s, chat, _, reign := tyrantArmy(t, ctx, 0)
	steps := []struct {
		fx   effect
		army float64
		line string
	}{
		{desert(1), 0, "já tava vazio"},
		{enlist(2), 2 * battalionGuard, "ganhou 2 batalhões"},
		{desert(1), battalionGuard, "perdeu 1 batalhão"},
		{enlist(armyBattalions * 2), armyMax, "ganhou 5 batalhões"},
		{enlist(1), armyMax, "já tava cheio"},
		{sabotage(armyBattalions), 0, "perdeu 6 batalhões"},
	}
	for _, step := range steps {
		line := playTyrannyEffect(t, ctx, s, chat, reign, step.fx)
		if got := hiredArmy(t, ctx, s, chat, reign).army(reign); math.Abs(got-step.army) > 1e-9 {
			t.Errorf("%+v: exército %v, want %v", step.fx, got, step.army)
		}
		if !strings.Contains(line, step.line) {
			t.Errorf("%+v: resposta sem %q: %q", step.fx, step.line, line)
		}
	}
}

func TestTyrannyEffectsNeedTheDictatorship(t *testing.T) {
	ctx := context.Background()
	chat := types.NewJID("120363000000000001", types.GroupServer)
	s, _ := jackpotWallets(t, ctx, chat, "5511900000002")
	now := time.Unix(1_800_000_000, 0)
	for _, fx := range []effect{rebel(5, 5), quell(5, 5), enlist(1), sabotage(1)} {
		if line := playTyrannyEffect(t, ctx, s, chat, now, fx); line != tyrannyOverText {
			t.Errorf("%+v sem ditadura respondeu %q", fx, line)
		}
	}
	if slices.ContainsFunc(eventCatalog, func(sc listedScenario) bool {
		return sc.book == tyrannyEvents.name && sc.fits(eventCase{eventSpot: eventSpot{place: "!trabalhar"}, job: Profession{president: true, decrees: decrees{}}})
	}) {
		t.Error("evento da ditadura saiu sem ditadura")
	}
}
