package economy

import (
	"context"
	"errors"
	"testing"

	"go.mau.fi/whatsmeow/types"
)

func TestOpenBoxes(t *testing.T) {
	ctx := context.Background()
	chat := types.NewJID("120363000000000001", types.GroupServer)
	s, people := jackpotWallets(t, ctx, chat, "5511900000001")
	acc := people[0]
	held := map[string]int64{"caixa-misteriosa": 5, "bau-do-tesouro": 2, "raspadinha": 1}
	err := s.inTx(ctx, func(tx *ledgerTx) error {
		for id, n := range held {
			if err := addItems(ctx, tx, chat, acc, id, n); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	bag, err := s.inventory(ctx, chat, acc)
	if err != nil {
		t.Fatal(err)
	}
	u := unbox(bag, 0)
	opened := map[string]int64{}
	for _, box := range u.opened {
		opened[box.item.id] = box.quantity
	}
	if len(opened) != 2 || opened["caixa-misteriosa"] != 5 || opened["bau-do-tesouro"] != 2 {
		t.Fatalf("abriu %v; queria as 5 caixas e os 2 baús, sem a raspadinha", opened)
	}
	var won int64
	for _, p := range u.prizes {
		if p.item.loot != nil {
			t.Errorf("%s saiu de prêmio sem ser aberta", p.item.id)
		}
		won += p.quantity
	}
	if won != 7 {
		t.Fatalf("saíram %d prêmios de 7 caixas", won)
	}

	if err := s.openBoxes(ctx, chat, acc, u); err != nil {
		t.Fatal(err)
	}
	after, err := s.inventory(ctx, chat, acc)
	if err != nil {
		t.Fatal(err)
	}
	var total int64
	for _, st := range after {
		if st.item.loot != nil {
			t.Errorf("%s continuou na mochila depois do !abrir tudo", st.item.id)
		}
		total += st.quantity
	}
	if total != won+held["raspadinha"] {
		t.Fatalf("mochila ficou com %d itens; queria os %d prêmios e a raspadinha", total, won)
	}

	if err := s.openBoxes(ctx, chat, acc, u); !errors.Is(err, errNoItems) {
		t.Fatalf("abrir de novo as caixas que já foram = %v; queria errNoItems", err)
	}
}
