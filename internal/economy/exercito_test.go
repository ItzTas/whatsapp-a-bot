package economy

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

func tyrantArmy(t *testing.T, ctx context.Context, taxed int64) (*walletStore, types.JID, core.Account, time.Time) {
	t.Helper()
	chat := types.NewJID("120363000000000001", types.GroupServer)
	s, people := jackpotWallets(t, ctx, chat, "5511900000001")
	president := people[0]
	reign := time.Unix(1_800_000_000, 0)
	if err := s.inaugurate(ctx, chat, president, 3, reign); err != nil {
		t.Fatal(err)
	}
	if on, set, err := s.flipDecree(ctx, chat, president, tyrannyDecree, reign); err != nil || !set || !on {
		t.Fatalf("o !decreto ditadura não ligou: on %v, set %v, err %v", on, set, err)
	}
	_, err := s.db.ExecContext(ctx, "UPDATE presidencias SET "+revoltColumn+" = ? WHERE chat = ?", taxed, chat.String())
	if err != nil {
		t.Fatal(err)
	}
	return s, chat, president, reign
}

func hiredArmy(t *testing.T, ctx context.Context, s *walletStore, chat types.JID, hired time.Time) decrees {
	t.Helper()
	d, err := s.decrees(ctx, chat, hired)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestArmyDesertsAtNormalPaceWhileRevoltIsLow(t *testing.T) {
	ctx := context.Background()
	s, chat, president, reign := tyrantArmy(t, ctx, 0)
	hired := reign.Add(10 * time.Minute)

	r, err := s.raiseArmy(ctx, chat, president, 2, hired)
	if err != nil {
		t.Fatal(err)
	}
	if r.routing || r.battalions != 2 || math.Abs(r.guard-2*battalionGuard) > 1e-9 || !r.until.Equal(hired.Add(2*battalionTime)) {
		t.Fatalf("raiseArmy = %+v, want 2 batalhões com +%v até %v sem deserção acelerada", r, 2*battalionGuard, hired.Add(2*battalionTime))
	}
	d := hiredArmy(t, ctx, s, chat, hired)
	if got := d.army(hired.Add(battalionTime)); math.Abs(got-battalionGuard) > 1e-9 {
		t.Errorf("exército depois de %v = %v, want %v", battalionTime, got, battalionGuard)
	}
}

func TestArmyDesertsFasterPastTheKnee(t *testing.T) {
	ctx := context.Background()
	s, chat, president, reign := tyrantArmy(t, ctx, 50)
	hired := reign.Add(100 * time.Minute)

	r, err := s.raiseArmy(ctx, chat, president, 10, hired)
	if err != nil {
		t.Fatal(err)
	}
	gone := hired.Add(armyBattalions * battalionTime / routPace)
	if !r.routing || r.battalions != armyBattalions || math.Abs(r.guard-armyMax) > 1e-9 || !r.until.Equal(gone) {
		t.Fatalf("raiseArmy = %+v, want %d batalhões com +%v até %v desertando mais rápido", r, armyBattalions, armyMax, gone)
	}
	d := hiredArmy(t, ctx, s, chat, hired)
	if got := d.army(hired.Add(battalionTime / routPace)); math.Abs(got-armyMax+battalionGuard) > 1e-9 {
		t.Errorf("exército depois de %v = %v, want %v", battalionTime/routPace, got, armyMax-battalionGuard)
	}
	if got := d.army(gone); got != 0 {
		t.Errorf("exército em %v = %v, want 0", gone, got)
	}
}

func TestArmySpeedsUpDesertionWhenRevoltCrossesTheKnee(t *testing.T) {
	ctx := context.Background()
	s, chat, president, reign := tyrantArmy(t, ctx, 50)
	routed := reign.Add(6 * revoltRiseEvery)
	hired := routed.Add(-battalionTime / 2)

	r, err := s.raiseArmy(ctx, chat, president, 1, hired)
	if err != nil {
		t.Fatal(err)
	}
	gone := routed.Add(battalionTime / 2 / routPace)
	if r.routing || !r.until.Equal(gone) {
		t.Fatalf("raiseArmy = %+v, want o último batalhão saindo em %v", r, gone)
	}
	d := hiredArmy(t, ctx, s, chat, hired)
	if got := d.army(routed); math.Abs(got-battalionGuard/2) > 1e-9 {
		t.Errorf("exército quando a revolta passa de %v = %v, want %v", revoltKnee, got, battalionGuard/2)
	}
	if got := d.army(gone); got != 0 {
		t.Errorf("exército em %v = %v, want 0", gone, got)
	}
}
