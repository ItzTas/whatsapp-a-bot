package main

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

func (b *bot) namorar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	couple, ok := coupleFrom(evt.Info.Sender, mentionedJIDs(evt.Message))
	if !ok {
		return b.sendText(ctx, chat, "Marca uma ou duas pessoas: !namorar @pessoa ou !namorar @pessoa1 @pessoa2")
	}

	var (
		all     []dating
		aliases [2][]string
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { all, err = b.datings.inChat(gctx, chat); return err })
	for i, p := range couple {
		g.Go(func() error { aliases[i] = b.userAliases(gctx, p); return nil })
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler namoros: %w", err)
	}

	if slices.ContainsFunc(aliases[0], func(a string) bool { return slices.Contains(aliases[1], a) }) {
		return b.sendText(ctx, chat, "Namorar consigo mesmo(a) não vale 🪞 Marca outra pessoa")
	}
	if i := slices.IndexFunc(all, func(d dating) bool { return d.joins(aliases[0], aliases[1]) }); i >= 0 {
		text := fmt.Sprintf("💑 @%s e @%s já namoram há %s", couple[0].User, couple[1].User, marriageLength(time.Since(all[i].since)))
		_, err := b.sendMentions(ctx, chat, text, couple[:])
		return err
	}

	if err := b.datings.add(ctx, chat, couple, time.Now()); err != nil {
		return fmt.Errorf("salvar namoro: %w", err)
	}
	lines, mentioned := polyamoryNews(all, couple, aliases)
	text := fmt.Sprintf("💘 @%s e @%s agora estão namorando!", couple[0].User, couple[1].User)
	if len(lines) > 0 {
		text += "\n\n" + strings.Join(lines, "\n")
	}
	_, err := b.sendMentions(ctx, chat, text, uniqueJIDs(append(couple[:], mentioned...)))
	return err
}

func polyamoryNews(all []dating, couple [2]types.JID, aliases [2][]string) ([]string, []types.JID) {
	var (
		lines     []string
		mentioned []types.JID
	)
	for i, p := range couple {
		others := datingPartnersOf(all, aliases[i])
		if len(others) == 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf("💞 Poliamor! @%s também namora %s", p.User, joinMentions(others)))
		mentioned = append(mentioned, others...)
	}
	return lines, mentioned
}
