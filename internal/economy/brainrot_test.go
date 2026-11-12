package economy

import (
	"testing"
)

func TestBrainrotsAreOnlyBought(t *testing.T) {
	list := brainrots()
	if len(list) != len(brainrotGoods) {
		t.Fatalf("todo brainrot tá no catálogo, achou %d de %d", len(list), len(brainrotGoods))
	}
	category, ok := categoryNamed("brainrots")
	if !ok {
		t.Fatal("brainrot tem tipo próprio")
	}
	for i, it := range list {
		if i > 0 && it.value <= list[i-1].value {
			t.Fatalf("a lista do !brainrot sobe de preço, %s vem depois de %s", it.name, list[i-1].name)
		}
		if it.flex == nil || it.flex.effect == "" {
			t.Fatalf("%s precisa de uma vantagem só de estar na mochila", it.name)
		}
		if !it.unboxable() || auctionGoods(it) || luxuryGoods(it) || it.gradable() || it.shop {
			t.Fatalf("%s só sai do !brainrot: nada de caixa, leilão, luxo, qualidade ou loja comum", it.name)
		}
		if it.store() != "!brainrot" || it.category() != category {
			t.Fatalf("%s aparece como brainrot e aponta pro !brainrot", it.name)
		}
		if got := marketPercentOf(it, 90); got != brainrotSalePercent {
			t.Fatalf("o mercado paga só %d%% num brainrot, mesmo com bônus de profissão, deu %d", brainrotSalePercent, got)
		}
		if found := findItems(it.id); len(found) != 1 || found[0].id != it.id {
			t.Fatalf("%s é achado pelo id sem confundir com outro item, deu %d itens", it.name, len(found))
		}
	}
}

func TestBrainrotsAreNeverStolen(t *testing.T) {
	tralalero, ok := itemByID("tralalero-tralala")
	if !ok {
		t.Fatal("Tralalero Tralala existe")
	}
	pick := theftChoice(item{}, 1)
	if got, ok := pick([]stack{{item: tralalero, quantity: 3}}); ok || got.id != "" {
		t.Fatalf("o furto sem alvo ignora brainrot, deu %s", got.id)
	}
}
