package economy

import (
	"testing"
	"time"
)

func TestCivicAppliesEveryFinishedWork(t *testing.T) {
	base := Profession{min: 10_000, max: 20_000, every: time.Hour}
	if got := base.civic(0); got.works != 0 || got.min != base.min || got.every != base.every || got.guard != 0 {
		t.Fatalf("sem obra pronta nada muda, deu %+v", got)
	}

	all := base.civic(len(publicWorks))
	wantPay := base.min * 103 / 100 * 103 / 100
	wantEvery := base.every * 97 / 100 * 95 / 100
	if all.works != len(publicWorks) || all.min != wantPay || all.every != wantEvery || all.dailyBonus != 2 ||
		all.findBonus != 0.02 || all.guard != 0.02 || all.safety != 10 {
		t.Fatalf("todas as obras somam o efeito de cada uma, deu %+v", all)
	}
}

func TestMonumentsPayUpToTheTop(t *testing.T) {
	works := len(publicWorks)
	base := Profession{min: 10_000, max: 10_000, every: time.Hour}
	before := base.civic(works)
	capped := base.civic(works + monumentPayTop)
	want := before.min
	for range monumentPayTop {
		want = want * (100 + monumentPayStep) / 100
	}
	if capped.min != want {
		t.Fatalf("os %d primeiros monumentos dão +%d%% no !trabalhar cada, deu %d, queria %d", monumentPayTop, monumentPayStep, capped.min, want)
	}
	if beyond := base.civic(works + monumentPayTop + 3); beyond.min != capped.min {
		t.Fatalf("monumento depois do %dº é só orgulho, deu %d", monumentPayTop, beyond.min)
	}
	if goal := workAt(works).goal; goal != publicWorks[works-1].goal*2 {
		t.Fatalf("o primeiro monumento custa o dobro da última obra, deu %d", goal)
	}
}

func TestWorksEffectsText(t *testing.T) {
	got := worksEffects(len(publicWorks))
	want := "+6% no !trabalhar, o !trabalhar recarrega 8% mais rápido, +2% no !daily, +2% de chance de achar item no !trabalhar, +2% de proteção e 10% menos chance de se machucar no trampo"
	if got != want {
		t.Fatalf("o resumo junta os efeitos das obras prontas\n deu: %s\nquer: %s", got, want)
	}
}

func TestStandingReadsFinishedWorks(t *testing.T) {
	fb := newFirmBench(t)
	donor := fb.person("5511900000001")
	fb.setBalance(t, donor, 1_000_000)
	if _, _, err := fb.s.donate(fb.ctx, fb.chat, donor, publicWorks[0].goal+publicWorks[1].goal, fb.now); err != nil {
		t.Fatal(err)
	}
	other := fb.person("5511900000002")
	fb.setBalance(t, other, 0)
	st, err := fb.s.standing(fb.ctx, fb.chat, other)
	if err != nil || st.works != 2 {
		t.Fatalf("as obras prontas valem pra todo mundo do grupo, deu %d (%v)", st.works, err)
	}
}
