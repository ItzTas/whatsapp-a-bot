package economy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) colher(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	job, err := b.ProfessionOf(ctx, chat, acc)
	if err != nil {
		return err
	}
	now := time.Now()
	r, err := b.Wallets.reap(ctx, chat, acc, job.farmGrade, now)
	if err != nil {
		return fmt.Errorf("colher: %w", err)
	}
	if r.reaped > 0 {
		b.practice(ctx, chat, acc, job, Profession.farms, 1)
	}
	_, err = b.SendMentions(ctx, chat, r.text(person, job, now), []types.JID{person})
	return err
}

func (r reaping) text(person types.JID, job Profession, now time.Time) string {
	if r.reaped == 0 && len(r.growing) == 0 {
		return fmt.Sprintf("🪹 A horta de @%s tá vazia. Planta com !plantar <semente>, que tem semente na !loja", person.User)
	}
	if r.reaped == 0 {
		next := r.growing[0]
		return fmt.Sprintf("⏳ Nada pronto ainda, @%s. %s fica pronto em %s\n_Manda !horta pra ver todos os canteiros_",
			person.User, next.seed.produce().label(), core.FormatWait(next.ready.Sub(now)))
	}
	lines := []string{fmt.Sprintf("🧺 @%s colheu %s da horta:", person.User, core.CountLabel(r.reaped, "canteiro", "canteiros"))}
	var worth int64
	for _, c := range r.crops {
		lines = append(lines, "• "+quantityLabel(c.item, c.quantity)+gradesLabel(c.grades))
		worth += c.worth()
	}
	lines = append(lines, "", "💰 Tudo junto vale "+Coins(worth)+". Vende no !vender ou dá pra alguém com !dar")
	if job.farmGrade > 0 {
		lines = append(lines, "✨ Veio com qualidade melhor porque "+job.title)
	}
	if len(r.growing) == 0 {
		return strings.Join(append(lines, "🌱 Os canteiros ficaram livres: manda !plantar de novo"), "\n")
	}
	lines = append(lines, fmt.Sprintf("⏳ Ainda tem %s crescendo, e o próximo fica pronto em %s",
		core.CountLabel(len(r.growing), "canteiro", "canteiros"), core.FormatWait(r.growing[0].ready.Sub(now))))
	return strings.Join(lines, "\n")
}
