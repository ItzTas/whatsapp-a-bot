package economy

import (
	"testing"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
)

func (fb firmBench) overthrow(t *testing.T, president, leader core.Account) coupAttempt {
	t.Helper()
	a := coupAttempt{president: president, plotters: []robber{{acc: leader}}, chance: 1}
	if err := fb.s.coup(fb.ctx, fb.chat, &a, fb.now); err != nil {
		t.Fatal(err)
	}
	if !a.success {
		t.Fatal("golpe com 100% de chance falhou")
	}
	return a
}

func TestCoupLeaderPlundersTyrant(t *testing.T) {
	fb := newFirmBench(t)
	dictator, leader := fb.person("1"), fb.person("2")
	fb.tyrant(t, dictator)
	fb.setBalance(t, dictator, 5_000)
	fb.setBalance(t, leader, 100)

	a := fb.overthrow(t, dictator, leader)

	if a.loot != 500 {
		t.Fatalf("saque = %d, want 500", a.loot)
	}
	if got := fb.balance(t, dictator); got != 4_500 {
		t.Fatalf("saldo do ditador = %d, want 4500", got)
	}
	if got := fb.balance(t, leader); got != 600 {
		t.Fatalf("saldo do líder = %d, want 600", got)
	}
}

func TestCoupLeaderTakesNothingFromElectedPresident(t *testing.T) {
	fb := newFirmBench(t)
	president, leader := fb.person("1"), fb.person("2")
	err := fb.s.inTx(fb.ctx, func(tx *ledgerTx) error {
		return swearIn(fb.ctx, tx, fb.chat, president, 3, false, fb.now)
	})
	if err != nil {
		t.Fatal(err)
	}
	fb.setBalance(t, president, 5_000)
	fb.setBalance(t, leader, 100)

	a := fb.overthrow(t, president, leader)

	if a.loot != 0 {
		t.Fatalf("saque = %d, want 0", a.loot)
	}
	if got := fb.balance(t, president); got != 5_000 {
		t.Fatalf("saldo do presidente = %d, want 5000", got)
	}
	if got := fb.balance(t, leader); got != 100 {
		t.Fatalf("saldo do líder = %d, want 100", got)
	}
}
