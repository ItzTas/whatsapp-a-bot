package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

func (b *bot) amante(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := firstMentionOrSender(evt).ToNonAD()

	var (
		lovers  []relationship
		aliases []string
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { lovers, err = b.relationships.inChat(gctx, chat, kindLover); return err })
	g.Go(func() error { aliases = b.userAliases(gctx, person); return nil })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler amantes: %w", err)
	}

	own := relationshipsOf(lovers, aliases)
	if len(own) == 0 {
		text := fmt.Sprintf("😇 @%s não tem amante nenhum(a)... ou esconde muito bem", person.User)
		_, err := b.sendMentions(ctx, chat, text, []types.JID{person})
		return err
	}

	now := time.Now()
	mentioned := []types.JID{person}
	lines := []string{fmt.Sprintf("🤫 *Amantes de @%s*", person.User), ""}
	for _, r := range own {
		partner, _ := r.partnerOf(aliases)
		mentioned = append(mentioned, partner)
		lines = append(lines, fmt.Sprintf("💋 @%s, se pegando escondido há %s", partner.User, marriageLength(now.Sub(r.since))))
	}
	_, err := b.sendMentions(ctx, chat, strings.Join(lines, "\n"), uniqueJIDs(mentioned))
	return err
}
