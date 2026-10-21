package economy

import (
	"context"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/types"
)

func TestTyrantDecreeSplit(t *testing.T) {
	const reign = 1_800_000_000
	cases := map[string]struct {
		pd                 percentDecree
		people, own, level int64
		tyranny            bool
		wantPeople         int64
	}{
		"fora da ditadura o decreto vale igual pra todo mundo": {incomeTaxDecree, 30, 30, 0, false, 0},
		"imposto cortado pelo ditador cai só 20% pro povo":     {incomeTaxDecree, 80, 80, 0, true, 64},
		"repetir o corte não alivia mais o povo":               {incomeTaxDecree, 64, 0, 0, true, 64},
		"subir o imposto do ditador sobe 20% pro povo":         {incomeTaxDecree, 64, 0, 50, true, 74},
		"subir o imposto além do que o povo paga vale inteiro": {incomeTaxDecree, 64, 0, 80, true, 80},
		"salário maior chega 20% pro povo":                     {payDecree, 0, 0, 50, true, 10},
		"salário menor vale inteiro pro povo":                  {payDecree, 0, 0, -30, true, -30},
		"espera menor no !trabalhar chega 20% pro povo":        {workDecree, 0, 0, -50, true, -10},
		"corte pequeno demais não chega no povo":               {incomeTaxDecree, 10, 10, 6, true, 10},
	}
	for name, c := range cases {
		d := decrees{c.pd.column: c.people, c.pd.rulerColumn(): c.own - c.people}
		if c.tyranny {
			d[tyrannyDecree.column] = reign
		}
		people, extra := d.split(c.pd, c.level)
		if people != c.wantPeople || people+extra != c.level {
			t.Errorf("%s: povo %d e ditador %d, want povo %d e ditador %d", name, people, people+extra, c.wantPeople, c.level)
		}
	}
}

func TestTyrantReadsOwnDecrees(t *testing.T) {
	ctx := context.Background()
	chat := types.NewJID("120363000000000001", types.GroupServer)
	s, people := jackpotWallets(t, ctx, chat, "5511900000001", "5511900000002")
	tyrant, citizen := people[0], people[1]
	now := time.Unix(1_800_000_000, 0)
	if err := s.inaugurate(ctx, chat, tyrant, 3, now); err != nil {
		t.Fatal(err)
	}
	if on, set, err := s.flipDecree(ctx, chat, tyrant, tyrannyDecree, now); err != nil || !set || !on {
		t.Fatalf("o !decreto ditadura não ligou: on %v, set %v, err %v", on, set, err)
	}
	if set, err := s.setDecree(ctx, chat, tyrant, payDecree, 10, 40, now); err != nil || !set {
		t.Fatalf("o !decreto salario não gravou: set %v, err %v", set, err)
	}

	reads := map[string]struct {
		d    func() (decrees, error)
		want int64
	}{
		"ditador":            {func() (decrees, error) { return s.decreesOf(ctx, chat, tyrant, now) }, 50},
		"cidadão":            {func() (decrees, error) { return s.decreesOf(ctx, chat, citizen, now) }, 10},
		"leitura sem pessoa": {func() (decrees, error) { return s.decrees(ctx, chat, now) }, 10},
	}
	for who, r := range reads {
		d, err := r.d()
		if err != nil {
			t.Fatal(err)
		}
		if got := d.level(payDecree); got != r.want {
			t.Errorf("salário por decreto lido pelo %s = %d%%, want %d%%", who, got, r.want)
		}
	}

	d, err := s.decrees(ctx, chat, now)
	if err != nil {
		t.Fatal(err)
	}
	inForce := d.inForce(now)
	want := payDecree.effect(10) + " (pro ditador, " + payDecree.effect(50) + ")"
	if len(inForce) == 0 || inForce[0] != want {
		t.Errorf("decretos em vigor = %q, want começando com %q", inForce, want)
	}
}
