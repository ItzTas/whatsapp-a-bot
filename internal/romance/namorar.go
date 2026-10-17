package romance

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

func (b *bot) namorar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	couple, ok := CoupleFrom(evt.Info.Sender, core.MentionedJIDs(evt.Message))
	if !ok {
		return b.SendText(ctx, chat, "Marca uma ou duas pessoas: !namorar @pessoa ou !namorar @pessoa1 @pessoa2")
	}
	consent, err := b.ChatModeOn(ctx, chat, core.ModeConsent)
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
		all       []core.Relationship
		adoptions []core.Adoption
		aliases   [2][]string
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { all, err = b.Relationships.InChat(gctx, chat, kindDating); return err })
	g.Go(func() (err error) { adoptions, err = b.Adoptions.InChat(gctx, chat); return err })
	for i, p := range couple {
		g.Go(func() error { aliases[i] = b.UserAliases(gctx, p); return nil })
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler namoros: %w", err)
	}

	if slices.ContainsFunc(aliases[0], func(a string) bool { return slices.Contains(aliases[1], a) }) {
		return b.SendText(ctx, chat, "Namorar consigo mesmo(a) não vale 🪞 Marca outra pessoa")
	}
	if child, parent, ok := adoptiveBond(adoptions, couple, aliases); ok {
		_, err := b.SendMentions(ctx, chat, adoptiveBondText(child, parent, "namorar"), couple[:])
		return err
	}
	if i := slices.IndexFunc(all, func(d core.Relationship) bool { return d.Joins(aliases[0], aliases[1]) }); i >= 0 {
		text := fmt.Sprintf("💑 @%s e @%s já namoram há %s", couple[0].User, couple[1].User, marriageLength(time.Since(all[i].Since)))
		_, err := b.SendMentions(ctx, chat, text, couple[:])
		return err
	}

	paid, ok, err := b.payProposal(ctx, deal)
	if err != nil {
		return err
	}
	if !ok {
		_, err := b.SendMentions(ctx, chat, strings.Join(paid, "\n"), couple[:])
		return err
	}
	if err := b.Relationships.Add(ctx, chat, kindDating, couple, time.Now(), deal != nil); err != nil {
		return fmt.Errorf("salvar namoro: %w", err)
	}
	lines, mentioned := polyamoryNews(all, couple, aliases)
	text := fmt.Sprintf("💘 @%s e @%s agora estão namorando!", couple[0].User, couple[1].User) + paragraph(lines) + paragraph(paid)
	_, err = b.SendMentions(ctx, chat, text, core.UniqueJIDs(append(couple[:], mentioned...)))
	return err
}

func polyamoryNews(all []core.Relationship, couple [2]types.JID, aliases [2][]string) ([]string, []types.JID) {
	var (
		lines     []string
		mentioned []types.JID
	)
	for i, p := range couple {
		others := core.OtherPartners(all, aliases[i], nil)
		if len(others) == 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf("💞 Poliamor! @%s também namora %s", p.User, core.JoinMentions(others)))
		mentioned = append(mentioned, others...)
	}
	return lines, mentioned
}
