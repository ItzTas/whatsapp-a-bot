package economy

import (
	"fmt"
	"testing"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
)

func (fb firmBench) president(t *testing.T, acc core.Account, now time.Time) {
	t.Helper()
	err := fb.s.inTx(fb.ctx, func(tx *ledgerTx) error {
		return swearIn(fb.ctx, tx, fb.chat, acc, 3, false, now)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func (fb firmBench) restock(t *testing.T, acc core.Account, now time.Time) (time.Duration, bool) {
	t.Helper()
	t.Cleanup(func() { restocks.mark(fb.chat, time.Time{}) })
	wait, over, err := fb.s.restockShops(fb.ctx, fb.chat, acc, now)
	if err != nil {
		t.Fatal(err)
	}
	return wait, over
}

func (fb firmBench) shelves(now time.Time) string {
	var shelves []string
	for _, stock := range [][]item{rotatingStock(fb.chat, now), luxuryStock(fb.chat, now), blackMarketStock(fb.chat, now)} {
		for _, it := range stock {
			shelves = append(shelves, fmt.Sprintf("%s:%d", it.id, blackMarketPercent(fb.chat, it, now)))
		}
	}
	return fmt.Sprint(shelves)
}

func TestRestockSwapsTodaysOffers(t *testing.T) {
	fb := newFirmBench(t)
	ana := fb.person("1")
	fb.president(t, ana, fb.now)
	tomorrow := fb.now.Add(24 * time.Hour)
	today, nextDay := fb.shelves(fb.now), fb.shelves(tomorrow)

	if wait, over := fb.restock(t, ana, fb.now); wait != 0 || over {
		t.Fatalf("repor = (%v, %v), want (0, false)", wait, over)
	}
	if fb.shelves(fb.now) == today {
		t.Fatal("as ofertas de hoje continuaram iguais depois da reposição")
	}
	if fb.shelves(tomorrow) != nextDay {
		t.Fatal("a reposição mexeu nas ofertas do dia seguinte")
	}
}

func TestRestockWaitOutlivesThePresident(t *testing.T) {
	fb := newFirmBench(t)
	ana, bia := fb.person("1"), fb.person("2")
	fb.president(t, ana, fb.now)
	fb.restock(t, ana, fb.now)

	soon := fb.now.Add(time.Hour)
	fb.president(t, bia, soon)
	if wait, over := fb.restock(t, bia, soon); over || wait != restockEvery-time.Hour {
		t.Fatalf("novo presidente repondo cedo = (%v, %v), want (%v, false)", wait, over, restockEvery-time.Hour)
	}

	later := fb.now.Add(restockEvery)
	fb.president(t, bia, later)
	if wait, over := fb.restock(t, bia, later); wait != 0 || over {
		t.Fatalf("repor depois da espera = (%v, %v), want (0, false)", wait, over)
	}
}

func TestRestockNeedsThePresidentInOffice(t *testing.T) {
	fb := newFirmBench(t)
	ana, bia := fb.person("1"), fb.person("2")
	fb.president(t, ana, fb.now)

	if _, over := fb.restock(t, bia, fb.now); !over {
		t.Fatal("quem não é presidente conseguiu repor as lojas")
	}
	if _, over := fb.restock(t, ana, fb.now.Add(termLength)); !over {
		t.Fatal("o presidente repôs as lojas depois do fim do mandato")
	}
}
