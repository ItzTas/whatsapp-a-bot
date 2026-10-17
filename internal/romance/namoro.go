package romance

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

type shipCandidate struct {
	jid      types.JID
	identity string
	percent  int
}

func (b *bot) namoro(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	name, _ := core.CommandName(core.MessageText(evt.Message))
	if chat.Server != types.GroupServer {
		return b.SendText(ctx, chat, fmt.Sprintf("O !%s só funciona em grupo", name))
	}
	person := core.FirstMentionOrSender(evt).ToNonAD()

	var (
		members []types.JID
		aliases []string
		datings []core.Relationship
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { members, err = b.GroupMembers(gctx, chat); return err })
	g.Go(func() (err error) { datings, err = b.Relationships.InChat(gctx, chat, kindDating); return err })
	g.Go(func() error { aliases = b.UserAliases(gctx, person); return nil })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("procurar namorado(a): %w", err)
	}

	if own := core.RelationshipsOf(datings, aliases); len(own) > 0 {
		text, mentioned := datingSummary(name, person, own, aliases, time.Now())
		_, err := b.SendMentions(ctx, chat, text, mentioned)
		return err
	}

	partner, ok := b.bestShip(ctx, aliases, members)
	if !ok {
		text := fmt.Sprintf("🥲 Não tem ninguém no grupo pra namorar com @%s", person.User)
		_, err := b.SendMentions(ctx, chat, text, []types.JID{person})
		return err
	}
	text := fmt.Sprintf(datingPhrases[name][0], person.User, "@"+partner.jid.User) +
		fmt.Sprintf("\n\n%s %d%% de ship", core.MeterBar(partner.percent), partner.percent)
	_, err := b.SendMentions(ctx, chat, text, []types.JID{person, partner.jid})
	return err
}

var datingPhrases = map[string][2]string{
	"namorado": {"💑 O namorado de @%s é %s", "💑 Os namorados de @%s são %s"},
	"namorada": {"💑 A namorada de @%s é %s", "💑 As namoradas de @%s são %s"},
	"namoro":   {"💑 @%s namora com %s", "💑 @%s namora com %s"},
}

func datingSummary(name string, person types.JID, own []core.Relationship, aliases []string, now time.Time) (string, []types.JID) {
	partners := make([]types.JID, len(own))
	lines := make([]string, len(own))
	for i, d := range own {
		partners[i], _ = d.PartnerOf(aliases)
		lines[i] = fmt.Sprintf("💘 Com @%s há %s", partners[i].User, marriageLength(now.Sub(d.Since)))
	}
	mentioned := append([]types.JID{person}, partners...)
	if len(own) == 1 {
		text := fmt.Sprintf(datingPhrases[name][0], person.User, "@"+partners[0].User) +
			fmt.Sprintf("\n\n💘 Namorando há %s", marriageLength(now.Sub(own[0].Since)))
		return text, mentioned
	}
	text := fmt.Sprintf(datingPhrases[name][1], person.User, core.JoinMentions(partners)) + "\n\n" + strings.Join(lines, "\n")
	return text, mentioned
}

func (b *bot) bestShip(ctx context.Context, aliases []string, members []types.JID) (shipCandidate, bool) {
	self := slices.Min(aliases)
	candidates := make([]shipCandidate, len(members))
	var wg sync.WaitGroup
	for i, m := range members {
		wg.Go(func() {
			identity := slices.Min(b.UserAliases(ctx, m))
			candidates[i] = shipCandidate{m, identity, shipPercent(self, identity)}
		})
	}
	wg.Wait()

	candidates = slices.DeleteFunc(candidates, func(c shipCandidate) bool {
		return c.identity == self || slices.Contains(aliases, c.jid.User)
	})
	if len(candidates) == 0 {
		return shipCandidate{}, false
	}
	return slices.MaxFunc(candidates, func(a, b shipCandidate) int {
		return cmp.Or(cmp.Compare(a.percent, b.percent), cmp.Compare(b.identity, a.identity))
	}), true
}
