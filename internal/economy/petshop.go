package economy

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types/events"
)

const petFetchChance = 0.01

type petFetch struct {
	pet  item
	gift item
}

func (b *bot) petshop(ctx context.Context, evt *events.Message) error {
	lines := []string{"🐾 *Petshop*", "_Bichinho fofo pra levar pra casa_", ""}
	for _, it := range catalog {
		if it.shop && it.pet {
			lines = append(lines, shopLine(it))
		}
	}
	lines = append(lines, "",
		"_Leva pra casa com !comprar <bicho>, tipo !comprar filhote ou !comprar gatinho. Depois é só !usar pra dar carinho, ou !dar @pessoa <bicho> pra dar de presente. De vez em quando o bicho aparece com um item no seu !daily_")
	return b.SendText(ctx, evt.Info.Chat, strings.Join(lines, "\n"))
}

func (it item) fetchChance() float64 {
	return petFetchChance * float64(it.rarity+1)
}

func petPerk(it item) string {
	return fmt.Sprintf("tem %d%% de chance de trazer um item comum no seu !daily, e dá pra !usar quantas vezes quiser, não gasta", percent(it.fetchChance()))
}

func fetchGift(pets []item) petFetch {
	for _, pet := range pets {
		if rand.Float64() < pet.fetchChance() {
			return petFetch{pet: pet, gift: core.PickOne(petGifts())}
		}
	}
	return petFetch{}
}

func petGifts() []item {
	return slices.DeleteFunc(slices.Clone(catalog), func(it item) bool { return it.rarity != common || it.pet || it.unboxable() })
}

func (f petFetch) text() string {
	if f.gift.id == "" {
		return ""
	}
	return fmt.Sprintf("\n🐾 De quebra, %s apareceu com %s e deu pra você", f.pet.label(), f.gift.label())
}
