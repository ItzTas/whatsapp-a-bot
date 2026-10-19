package economy

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"golang.org/x/sync/errgroup"
)

func (b *bot) unseatDeserter(ctx context.Context, chat types.JID, left []types.JID) {
	if err := b.dropDeserter(ctx, chat, left); err != nil {
		b.Client.Log.Errorf("tirar presidente que saiu de %s: %v", chat, err)
	}
}

func (b *bot) dropDeserter(ctx context.Context, chat types.JID, left []types.JID) error {
	now := time.Now()
	p, found, err := b.Wallets.presidency(ctx, chat, now)
	if err != nil || !found {
		return err
	}
	deserted := slices.ContainsFunc(left, func(person types.JID) bool { return b.Account(ctx, person).Key == p.president.Key })
	if !deserted {
		return nil
	}
	root := ledgers.of(chat)
	stays, err := b.memberElsewhere(ctx, root, chat, p.president)
	if err != nil || stays {
		return err
	}
	removed, err := b.Wallets.oust(ctx, chat, p.president, now)
	if err != nil || !removed {
		return err
	}
	text := fmt.Sprintf("🚪 *A presidência ficou vaga!* @%s saiu do grupo e perdeu o cargo, e os poderes e decretos da presidência caíram junto. Manda !votar @pessoa pra abrir uma eleição nova\n\n🎲 %s",
		p.president.JID.User, vacancyNote(now.Add(vacancyWindow), now))
	return b.announce(ctx, root, text, p.president.JID)
}

func (b *bot) memberElsewhere(ctx context.Context, root, left types.JID, acc core.Account) (bool, error) {
	groups := slices.DeleteFunc(b.economyGroups(ctx, root), func(group types.JID) bool { return group == left })
	if len(groups) == 0 {
		return false, nil
	}
	ids := b.Identities(ctx, acc.JID)
	found := make([]bool, len(groups))
	g, gctx := errgroup.WithContext(ctx)
	for i, group := range groups {
		g.Go(func() error {
			info, err := b.Client.GetGroupInfo(gctx, group)
			if err != nil {
				return err
			}
			owners := b.MemberOwners(info.Participants)
			found[i] = slices.ContainsFunc(ids, func(id types.JID) bool { _, ok := owners[id.User]; return ok })
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return false, err
	}
	return slices.Contains(found, true), nil
}
