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
	consent, err := b.chatModeOn(ctx, chat, modeConsent)
	if err != nil {
		return fmt.Errorf("checar modo consentimento: %w", err)
	}
	if consent {
		return b.propose(ctx, evt, kindDating)
	}
	return b.date(ctx, chat, couple, nil)
}

func (b *bot) date(ctx context.Context, chat types.JID, couple [2]types.JID, deal *romanceProposal) error {
	var (
		all       []relationship
		adoptions []adoption
		aliases   [2][]string
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { all, err = b.relationships.inChat(gctx, chat, kindDating); return err })
	g.Go(func() (err error) { adoptions, err = b.adoptions.inChat(gctx, chat); return err })
	for i, p := range couple {
		g.Go(func() error { aliases[i] = b.userAliases(gctx, p); return nil })
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler namoros: %w", err)
	}

	if slices.ContainsFunc(aliases[0], func(a string) bool { return slices.Contains(aliases[1], a) }) {
		return b.sendText(ctx, chat, "Namorar consigo mesmo(a) não vale 🪞 Marca outra pessoa")
	}
	if child, parent, ok := adoptiveBond(adoptions, couple, aliases); ok {
		_, err := b.sendMentions(ctx, chat, adoptiveBondText(child, parent, "namorar"), couple[:])
		return err
	}
	if i := slices.IndexFunc(all, func(d relationship) bool { return d.joins(aliases[0], aliases[1]) }); i >= 0 {
		text := fmt.Sprintf("💑 @%s e @%s já namoram há %s", couple[0].User, couple[1].User, marriageLength(time.Since(all[i].since)))
		_, err := b.sendMentions(ctx, chat, text, couple[:])
		return err
	}

	paid, ok, err := b.payProposal(ctx, deal)
	if err != nil {
		return err
	}
	if !ok {
		_, err := b.sendMentions(ctx, chat, strings.Join(paid, "\n"), couple[:])
		return err
	}
	if err := b.relationships.add(ctx, chat, kindDating, couple, time.Now(), deal != nil); err != nil {
		return fmt.Errorf("salvar namoro: %w", err)
	}
	lines, mentioned := polyamoryNews(all, couple, aliases)
	text := fmt.Sprintf("💘 @%s e @%s agora estão namorando!", couple[0].User, couple[1].User) + paragraph(lines) + paragraph(paid)
	_, err = b.sendMentions(ctx, chat, text, uniqueJIDs(append(couple[:], mentioned...)))
	return err
}

func polyamoryNews(all []relationship, couple [2]types.JID, aliases [2][]string) ([]string, []types.JID) {
	var (
		lines     []string
		mentioned []types.JID
	)
	for i, p := range couple {
		others := otherPartners(all, aliases[i], nil)
		if len(others) == 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf("💞 Poliamor! @%s também namora %s", p.User, joinMentions(others)))
		mentioned = append(mentioned, others...)
	}
	return lines, mentioned
}
