package economy

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) plantar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	p, err := b.plot(ctx, chat, acc)
	if err != nil {
		return fmt.Errorf("ler horta: %w", err)
	}
	q := parseItemQuery(core.MessageText(evt.Message))
	if q.name == "" {
		return b.SendText(ctx, chat, "Diz o que quer plantar: !plantar semente de milho, !plantar milho 3 ou !plantar milho tudo 🌱\n\n"+seedsLine(p.bag)+"\n_Manda !horta pra ver os seus canteiros_")
	}
	found := findItems(q.name)
	matches := preferItems(plantable(found), ownedIn(p.bag))
	if len(found) > 0 && len(matches) == 0 {
		return b.SendText(ctx, chat, fmt.Sprintf("Não dá pra plantar *%s* 🤔 Só semente e muda vão pra horta. Manda !item horta pra ver todas", q.name))
	}
	if problem := itemQueryProblem(q.name, matches); problem != "" {
		return b.SendText(ctx, chat, problem)
	}
	seed := matches[0]
	held := quantityOf(p.bag, seed.id)
	if held == 0 {
		return b.SendText(ctx, chat, fmt.Sprintf("Você não tem %s 🎒 Dá pra conseguir com %s", seed.label(), core.JoinWithAnd(sources(seed))))
	}
	now := time.Now()
	room := p.room()
	free := int64(room - len(p.beds))
	if free <= 0 {
		return b.SendText(ctx, chat, fullGardenText(p, now))
	}
	want := q.quantity(held)
	planted := min(want, held, free)
	grows := p.job.growth(seed.seed.grows)
	err = b.Wallets.sow(ctx, chat, acc, seed, planted, room, now.Add(grows))
	if errors.Is(err, errGardenFull) || errors.Is(err, errNoItems) {
		return b.SendText(ctx, chat, "A horta ou a mochila mudou enquanto você plantava 😵 Confere no !horta e tenta de novo")
	}
	if err != nil {
		return fmt.Errorf("plantar: %w", err)
	}
	text := fmt.Sprintf("🌱 @%s plantou %s. Em %s vira %s (de %d a %d por canteiro) pra !colher",
		person.User, quantityLabel(seed, planted), core.FormatWait(grows), seed.produce().label(), seed.seed.least, seed.seed.most)
	if p.job.farmFaster > 0 {
		text += fmt.Sprintf("\n🌿 Cresce %d%% mais rápido porque %s", p.job.farmFaster, p.job.title)
	}
	text += shortfallNote(want, held, free) + fmt.Sprintf("\n🌾 Canteiros: %d/%d", len(p.beds)+int(planted), room)
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func plantable(found []item) []item {
	var picked []item
	for _, it := range found {
		seed, ok := seedFor(it)
		if ok && !slices.ContainsFunc(picked, func(s item) bool { return s.id == seed.id }) {
			picked = append(picked, seed)
		}
	}
	return picked
}

func fullGardenText(p plot, now time.Time) string {
	return fmt.Sprintf("🌾 A sua horta tá cheia (%d/%d canteiros). %s\n%s", len(p.beds), p.room(), nextHarvest(p.beds[0], now), p.expansionLine())
}

func nextHarvest(next bed, now time.Time) string {
	if next.ripe(now) {
		return "Já tem coisa pronta: manda !colher e libera o canteiro"
	}
	return fmt.Sprintf("O próximo fica pronto em %s, aí é só mandar !colher", core.FormatWait(next.ready.Sub(now)))
}

func shortfallNote(want, held, free int64) string {
	if want <= min(held, free) {
		return ""
	}
	if held < free {
		return fmt.Sprintf("\n_Você só tinha %d na mochila_", held)
	}
	return fmt.Sprintf("\n_Só cabia mais %s na horta_", core.CountLabel(int(free), "canteiro", "canteiros"))
}
