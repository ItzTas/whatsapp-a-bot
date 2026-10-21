package economy

import (
	"context"
	"testing"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

func TestSaleDecreeLevel(t *testing.T) {
	const reign = 1_800_000_000
	cases := map[string]struct {
		stored  int64
		tyranny bool
		want    int64
	}{
		"bônus passa do teto":                {50, false, saleDecreeSwing},
		"imposto passa do teto":              {-50, false, -saleDecreeSwing},
		"dentro do limite fica igual":        {-3, false, -3},
		"imposto na ditadura vai mais longe": {-50, true, -tyrantSaleTaxMax},
		"bônus na ditadura não muda o teto":  {50, true, saleDecreeSwing},
	}
	for name, c := range cases {
		d := decrees{saleDecree.column: c.stored}
		if c.tyranny {
			d[tyrannyDecree.column] = reign
		}
		if got := d.level(saleDecree); got != c.want {
			t.Errorf("%s: decreto venda gravado como %d vale %d, want %d", name, c.stored, got, c.want)
		}
	}

	people, extra := (decrees{tyrannyDecree.column: reign}).split(saleDecree, saleDecreeSwing)
	if people != 1 || extra != saleDecreeSwing-1 {
		t.Errorf("bônus de venda do ditador: povo %d e ditador %d, want povo 1 e ditador %d", people, people+extra, saleDecreeSwing)
	}
}

func TestSaleDecreeSale(t *testing.T) {
	ctx := context.Background()
	chat := types.NewJID("120363000000000001", types.GroupServer)
	box, _ := itemByID(presidentBoxItem)
	sell := func(t *testing.T, s *walletStore, seller core.Account, terms saleTerms) (payout, balance int64) {
		t.Helper()
		if err := s.inTx(ctx, func(tx *ledgerTx) error { return addItems(ctx, tx, chat, seller, box.id, 10) }); err != nil {
			t.Fatal(err)
		}
		sold, balance, err := s.sellItems(ctx, chat, seller, []marketSale{{item: box, quantity: 10, percent: marketSalePercent}}, terms)
		if err != nil {
			t.Fatal(err)
		}
		return sold[0].payout, balance
	}
	balanceOf := func(t *testing.T, s *walletStore, acc core.Account) int64 {
		t.Helper()
		balance, _, err := s.Balance(ctx, chat, acc)
		if err != nil {
			t.Fatal(err)
		}
		return balance
	}

	t.Run("bônus paga a mais com moeda nova", func(t *testing.T) {
		s, people := jackpotWallets(t, ctx, chat, "5511900000001")
		payout, balance := sell(t, s, people[0], saleTerms{boost: saleDecreeSwing})
		bonus := payout * saleDecreeSwing / 100
		if bonus == 0 || balance != 50_000+payout+bonus {
			t.Fatalf("venda de %d com bônus de %d%%: saldo %d, want %d", payout, saleDecreeSwing, balance, 50_000+payout+bonus)
		}
	})

	t.Run("imposto vai pro presidente", func(t *testing.T) {
		s, people := jackpotWallets(t, ctx, chat, "5511900000001", "5511900000002")
		seller, president := people[0], people[1]
		now := time.Now()
		if err := s.inaugurate(ctx, chat, president, 3, now); err != nil {
			t.Fatal(err)
		}
		if set, err := s.setDecree(ctx, chat, president, saleDecree, -saleDecreeSwing, 0, now); err != nil || !set {
			t.Fatalf("o !decreto venda não gravou: set %v, err %v", set, err)
		}
		p, _, err := s.presidency(ctx, chat, now)
		if err != nil {
			t.Fatal(err)
		}
		if got := p.taxLevy(saleDecree, president).percent; got != 0 {
			t.Fatalf("o presidente paga %d%% de imposto na própria venda, want 0", got)
		}
		payout, balance := sell(t, s, seller, saleTerms{levy: p.taxLevy(saleDecree, seller)})
		tax := payout * saleDecreeSwing / 100
		if tax == 0 || balance != 50_000+payout-tax {
			t.Fatalf("venda de %d com imposto de %d%%: saldo %d, want %d", payout, saleDecreeSwing, balance, 50_000+payout-tax)
		}
		if got := balanceOf(t, s, president); got != 50_000+tax {
			t.Fatalf("presidente ficou com %d, want %d", got, 50_000+tax)
		}
	})

	t.Run("imposto abusivo da ditadura revolta o povo", func(t *testing.T) {
		s, people := jackpotWallets(t, ctx, chat, "5511900000001", "5511900000002")
		seller, tyrant := people[0], people[1]
		now := time.Now()
		if err := s.inaugurate(ctx, chat, tyrant, 3, now); err != nil {
			t.Fatal(err)
		}
		if on, set, err := s.flipDecree(ctx, chat, tyrant, tyrannyDecree, now); err != nil || !set || !on {
			t.Fatalf("o !decreto ditadura não ligou: on %v, set %v, err %v", on, set, err)
		}
		if set, err := s.setDecree(ctx, chat, tyrant, saleDecree, -tyrantSaleTaxMax, 0, now); err != nil || !set {
			t.Fatalf("o !decreto venda não gravou: set %v, err %v", set, err)
		}
		p, _, err := s.presidency(ctx, chat, now)
		if err != nil {
			t.Fatal(err)
		}
		payout, balance := sell(t, s, seller, saleTerms{levy: p.taxLevy(saleDecree, seller)})
		tax := payout * tyrantSaleTaxMax / 100
		if balance != 50_000+payout-tax {
			t.Fatalf("venda de %d com imposto de %d%% na ditadura: saldo %d, want %d", payout, tyrantSaleTaxMax, balance, 50_000+payout-tax)
		}
		d, err := s.decrees(ctx, chat, now)
		if err != nil {
			t.Fatal(err)
		}
		if want := int64(tyrantSaleTaxMax - saleDecreeSwing); d[revoltColumn] != want {
			t.Fatalf("revolta do imposto de venda = %d pontos, want %d", d[revoltColumn], want)
		}
	})
}
