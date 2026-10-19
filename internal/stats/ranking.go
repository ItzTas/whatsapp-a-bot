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

type rankingPeriod struct {
	label  string
	length time.Duration
}

type rankedMember struct {
	jid      types.JID
	messages int
}

func (b *bot) ranking(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return b.SendText(ctx, chat, "O !ranking só funciona em grupo")
	}
	period := rankingPeriodFrom(core.MessageText(evt.Message))
	since := max(time.Now().Add(-period.length).Unix(), core.WindowStart())

	var (
		group  *types.GroupInfo
		counts []core.SenderCount
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { group, err = b.Client.GetGroupInfo(gctx, chat); return err })
	g.Go(func() (err error) { counts, err = b.messages.CountsBySender(gctx, chat, since); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("montar ranking: %w", err)
	}

	ranked := b.rankMembers(group.Participants, counts)
	if len(ranked) == 0 {
		return b.SendText(ctx, chat, fmt.Sprintf("Ninguém falou nada nos %s 🦗", period.label))
	}
	top := ranked[:min(len(ranked), core.RankingSize)]
	people := make([]types.JID, len(top))
	for i, m := range top {
		people[i] = m.jid
	}
	_, err := b.SendMentions(ctx, chat, rankingText(period, top, ranked), people)
	return err
}

func rankingPeriodFrom(text string) rankingPeriod {
	periods := map[string]rankingPeriod{
		"hoje":   {"últimas 24 horas", 24 * time.Hour},
		"semana": {"últimos 7 dias", 7 * 24 * time.Hour},
		"mes":    {"últimos 30 dias", 30 * 24 * time.Hour},
		"tudo":   {core.MessageWindowLabel, core.MessageWindow},
	}
	for _, word := range core.NormalizedWords(text) {
		if p, ok := periods[word]; ok {
			return p
		}
	}
	return periods["semana"]
}

func (b *bot) rankMembers(participants []types.GroupParticipant, counts []core.SenderCount) []rankedMember {
	owners := b.MemberOwners(participants)
	totals := map[types.JID]int{}
	for _, c := range counts {
		owner, ok := owners.Of(c.Sender, c.SenderAlt)
		if !ok {
			continue
		}
		totals[owner] += c.Messages
	}

	ranked := make([]rankedMember, 0, len(totals))
	for jid, messages := range totals {
		ranked = append(ranked, rankedMember{jid: jid, messages: messages})
	}
	slices.SortFunc(ranked, func(a, b rankedMember) int {
		if a.messages != b.messages {
			return b.messages - a.messages
		}
		return strings.Compare(a.jid.User, b.jid.User)
	})
	return ranked
}

func rankingText(period rankingPeriod, top, all []rankedMember) string {
	lines := slices.Concat(
		[]string{fmt.Sprintf("🏆 *Quem mais falou* nos %s", period.label), ""},
		rankingLines(top, "mensagem", "mensagens"),
	)
	total := 0
	for _, m := range all {
		total += m.messages
	}
	return strings.Join(append(lines, "", fmt.Sprintf("_%s no total_", core.CountLabel(total, "mensagem", "mensagens"))), "\n")
}

func rankingLines(top []rankedMember, singular, plural string) []string {
	medals := []string{"🥇", "🥈", "🥉"}
	lines := make([]string, len(top))
	for i, m := range top {
		position := fmt.Sprintf("%d.", i+1)
		if i < len(medals) {
			position = medals[i]
		}
		lines[i] = fmt.Sprintf("%s @%s: %s", position, m.jid.User, core.CountLabel(m.messages, singular, plural))
	}
	return lines
}
