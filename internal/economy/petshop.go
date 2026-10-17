package economy

import (
	"context"
	"strings"

	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) petshop(ctx context.Context, evt *events.Message) error {
	lines := []string{"🐾 *Petshop*", "_Bichinho fofo pra levar pra casa_", ""}
	for _, it := range catalog {
		if it.shop && it.pet {
			lines = append(lines, shopLine(it))
		}
	}
	lines = append(lines, "",
		"_Leva pra casa com !comprar <bicho>, tipo !comprar filhote ou !comprar gatinho. Depois é só !usar pra dar carinho, ou !dar @pessoa <bicho> pra dar de presente_")
	return b.SendText(ctx, evt.Info.Chat, strings.Join(lines, "\n"))
}
