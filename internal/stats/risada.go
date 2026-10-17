package stats

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

func (b *bot) palhaco(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return b.SendText(ctx, chat, "O !palhaço só funciona em grupo")
	}
	period := rankingPeriodFrom(core.MessageText(evt.Message))
	since := max(time.Now().Add(-period.length).Unix(), core.WindowStart())

	var (
		group  *types.GroupInfo
		counts []core.SenderCount
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { group, err = b.Client.GetGroupInfo(gctx, chat); return err })
	g.Go(func() (err error) { counts, err = b.messages.LaughsByTarget(gctx, chat, since); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("montar ranking de palhaços: %w", err)
	}

	ranked := b.rankMembers(group.Participants, counts)
	if len(ranked) == 0 {
		return b.SendText(ctx, chat, fmt.Sprintf("Ninguém fez o grupo rir nos %s 😐", period.label))
	}
	top := ranked[:min(len(ranked), core.RankingSize)]
	people := make([]types.JID, len(top))
	for i, m := range top {
		people[i] = m.jid
	}
	lines := slices.Concat(
		[]string{fmt.Sprintf("🤡 *Palhaços do grupo* nos %s", period.label), ""},
		rankingLines(top, "risada", "risadas"),
	)
	if _, err := b.SendMentions(ctx, chat, strings.Join(lines, "\n"), people); err != nil {
		return err
	}
	return b.SendLaughSticker(ctx, chat, nil)
}
