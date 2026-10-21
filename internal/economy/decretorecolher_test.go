package economy

import (
	"context"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/types"
)

func TestCurfewDecreeLasts(t *testing.T) {
	ctx := context.Background()
	chat := types.NewJID("120363000000000001", types.GroupServer)
	s, people := jackpotWallets(t, ctx, chat, "5511900000001")
	president := people[0]
	now := time.Unix(1_800_000_000, 0)
	if err := s.inaugurate(ctx, chat, president, 3, now); err != nil {
		t.Fatal(err)
	}
	curfewAt := func(at time.Time) bool {
		t.Helper()
		d, err := s.decrees(ctx, chat, at)
		if err != nil {
			t.Fatal(err)
		}
		return d.on(curfewDecree)
	}
	flip := func(td toggleDecree, at time.Time) bool {
		t.Helper()
		on, set, err := s.flipDecree(ctx, chat, president, td, at)
		if err != nil || !set {
			t.Fatalf("decreto %s às %v: set %v, err %v", td.names[0], at, set, err)
		}
		return on
	}

	if !flip(curfewDecree, now) {
		t.Fatal("o !decreto recolher não ligou o toque de recolher")
	}
	if !curfewAt(now.Add(curfewTime - time.Second)) {
		t.Fatal("o toque de recolher não vale até o fim dos 15 minutos")
	}
	if curfewAt(now.Add(curfewTime)) {
		t.Fatal("o toque de recolher vale depois dos 15 minutos")
	}

	later := now.Add(curfewTime + time.Minute)
	if !flip(curfewDecree, later) {
		t.Fatal("o !decreto recolher depois do toque vencido não ligou de novo")
	}
	if flip(curfewDecree, later.Add(time.Minute)) {
		t.Fatal("o !decreto recolher com o toque valendo não desligou")
	}
	if curfewAt(later.Add(time.Minute)) {
		t.Fatal("o toque de recolher desligado continua valendo")
	}

	if !flip(tyrannyDecree, now) {
		t.Fatal("o !decreto ditadura não ligou a ditadura")
	}
	d, err := s.decrees(ctx, chat, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if !d.on(tyrannyDecree) {
		t.Fatal("a ditadura não vale até o fim do mandato")
	}
}
