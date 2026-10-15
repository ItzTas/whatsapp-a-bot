package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

var divorceSettlements = []string{
	"A guarda do cachorro ficou com @%[2]s 🐶",
	"@%[1]s ficou com a casa e @%[2]s ficou com as dívidas 💸",
	"@%[2]s levou até a air fryer 🍟",
	"A pensão vai ser paga em figurinha 🃏",
	"@%[1]s já tá no Tinder de novo 📱",
	"Os dois vão fingir que não se conhecem no próximo rolê 🙈",
}

func (b *bot) divorciar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	sender := evt.Info.Sender.ToNonAD()
	mentions := uniqueJIDs(mentionedJIDs(evt.Message))
	if len(mentions) > 2 {
		return b.sendText(ctx, chat, "Divórcio é de duas em duas pessoas: !divorciar @pessoa ou !divorciar @pessoa1 @pessoa2")
	}
	people := append([]types.JID{sender}, mentions...)
	if len(mentions) == 2 {
		people = mentions
	}

	var all []marriage
	aliases := make([][]string, len(people))
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { all, err = b.marriages.inChat(gctx, chat); return err })
	for i, p := range people {
		g.Go(func() error { aliases[i] = b.userAliases(gctx, p); return nil })
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler casamentos: %w", err)
	}

	ended := marriagesOf(all, aliases[0])
	if len(people) == 2 {
		ended = slices.DeleteFunc(ended, func(m marriage) bool { return !m.joins(aliases[0], aliases[1]) })
	}
	if len(ended) == 0 {
		return b.notMarried(ctx, chat, people)
	}

	if err := b.marriages.remove(ctx, ended); err != nil {
		return fmt.Errorf("apagar casamentos: %w", err)
	}
	text, mentioned := divorceText(ended, aliases[0], time.Now())
	_, err := b.sendMentions(ctx, chat, text, mentioned)
	return err
}

func (b *bot) notMarried(ctx context.Context, chat types.JID, people []types.JID) error {
	if len(people) == 1 {
		return b.sendText(ctx, chat, "Você nem é casado(a) com ninguém 🤨 Usa !casar @pessoa primeiro")
	}
	_, err := b.sendMentions(ctx, chat, fmt.Sprintf("@%s e @%s nem são casados 🤨", people[0].User, people[1].User), people)
	return err
}

func divorceText(ended []marriage, initiator []string, now time.Time) (string, []types.JID) {
	lines := []string{"💔 *Divórcio*", ""}
	var mentioned []types.JID
	for _, m := range ended {
		first, second := m.spouses[0], m.spouses[1]
		if partner, ok := m.partnerOf(initiator); ok && partner == first {
			first, second = second, first
		}
		lines = append(lines, fmt.Sprintf("@%s e @%s se separaram depois de %s de casamento", first.User, second.User, marriageLength(now.Sub(m.since))))
		mentioned = append(mentioned, first, second)
	}
	last := ended[len(ended)-1].spouses
	settlement := divorceSettlements[rand.IntN(len(divorceSettlements))]
	lines = append(lines, "", fmt.Sprintf(settlement, last[0].User, last[1].User))
	return strings.Join(lines, "\n"), uniqueJIDs(mentioned)
}
