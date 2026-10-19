package economy

import (
	"context"
	"slices"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) extracao(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	job, err := b.ProfessionOf(ctx, chat, acc)
	if err != nil {
		return err
	}
	now := time.Now()
	var (
		texts  []string
		people []types.JID
		spots  []eventSpot
	)
	for _, a := range activities {
		x, err := b.extract(ctx, chat, person, acc, job, a, now)
		if err != nil {
			return err
		}
		texts = append(texts, x.text+x.hurt)
		for _, p := range x.people {
			if !slices.Contains(people, p) {
				people = append(people, p)
			}
		}
		if x.died {
			spots = nil
			break
		}
		if x.eventful() {
			spots = append(spots, x.spot)
		}
	}
	if _, err := b.SendMentions(ctx, chat, strings.Join(texts, "\n\n")+job.decrees.note(gatherDecree), people); err != nil {
		return err
	}
	for _, spot := range spots {
		b.stirEvent(ctx, evt, spot)
	}
	return nil
}
