package economy

import (
	"context"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const bundlePacePercent = 150

func bundled(a activity) activity {
	a.cooldown.every = a.cooldown.every * bundlePacePercent / 100
	return a
}

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
		x, err := b.extract(ctx, chat, person, acc, job, bundled(a), now)
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
	if len(spots) == 0 {
		return nil
	}
	b.stirEvent(ctx, evt, spots[rand.IntN(len(spots))])
	return nil
}
