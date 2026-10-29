package economy

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/types"
)

func TestLegendCatalog(t *testing.T) {
	items := map[string]bool{}
	for _, sc := range eventCatalog {
		if sc.item == "" {
			continue
		}
		it, ok := itemByID(sc.item)
		if !ok {
			t.Errorf("%s: lenda de item desconhecido %q", sc.id, sc.item)
			continue
		}
		if items[it.id] {
			t.Errorf("%s: %s já tem outra lenda", sc.id, it.id)
		}
		items[it.id] = true
		if !it.keep || it.use == "" || it.graft != nil {
			t.Errorf("%s: o !usar de %s nunca chega na lenda (precisa de keep e use, sem graft)", sc.id, it.id)
		}
		if effectPerk(it) != legendPerk {
			t.Errorf("%s: o !item de %s não fala da lenda: %q", sc.id, it.id, effectPerk(it))
		}
		if sc.stage || !slices.Equal(sc.places, []string{legendPlace}) {
			t.Errorf("%s: lenda abre só pelo %s e não é etapa", sc.id, legendPlace)
		}
		if sc.dreadful() {
			t.Errorf("%s: lenda não castiga quem fica parado", sc.id)
		}
	}
	if len(items) == 0 {
		t.Error("nenhum item tem lenda")
	}
}

func TestLegendOncePerEconomy(t *testing.T) {
	ctx := context.Background()
	chat := types.NewJID("120363000000000001", types.GroupServer)
	s, people := jackpotWallets(t, ctx, chat, "5511900000001", "5511900000002")
	first, second := people[0], people[1]
	now := time.Now()

	claim := func(t *testing.T, who int, id string, want bool) {
		t.Helper()
		woke, err := s.claimLegend(ctx, chat, people[who], id, now)
		if err != nil || woke != want {
			t.Fatalf("pessoa %d acordando %s: acordou=%v (queria %v), erro %v", who, id, woke, want, err)
		}
	}
	claim(t, 0, "lenda-lampada", true)
	claim(t, 1, "lenda-lampada", false)
	claim(t, 0, "lenda-lampada", false)
	claim(t, 1, "lenda-dragao", true)

	woken, err := s.wokenLegends(ctx, chat)
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(woken)
	if !slices.Equal(woken, []string{"lenda-dragao", "lenda-lampada"}) {
		t.Fatalf("lendas acordadas %v; queria a da lâmpada e a do dragão", woken)
	}

	if err := s.forgetLegend(ctx, chat, second, "lenda-lampada"); err != nil {
		t.Fatal(err)
	}
	claim(t, 1, "lenda-lampada", false)
	if err := s.forgetLegend(ctx, chat, first, "lenda-lampada"); err != nil {
		t.Fatal(err)
	}
	claim(t, 1, "lenda-lampada", true)
}

func TestSleepingLegends(t *testing.T) {
	var all []string
	for _, sc := range eventCatalog {
		if sc.item != "" {
			all = append(all, sc.id)
		}
	}
	lamp, _ := itemByID("lampada-magica")
	egg, _ := itemByID("ovo-de-dragao")
	text := sleepingLegends([]string{"lenda-lampada"})
	if strings.Contains(text, lamp.label()) || !strings.Contains(text, egg.label()) {
		t.Errorf("com a lâmpada acordada, a lista devia ter o ovo e não a lâmpada: %q", text)
	}
	if text := sleepingLegends(all); text != "🪔 Todas as lendas da economia já acordaram" {
		t.Errorf("com todas acordadas: %q", text)
	}
}
