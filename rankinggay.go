package main

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

type measuredMember struct {
	jid     types.JID
	percent int
}

func (b *bot) meterRanking(m meter) command {
	return func(ctx context.Context, evt *events.Message) error {
		chat := evt.Info.Chat.ToNonAD()
		if chat.Server != types.GroupServer {
			return b.sendText(ctx, chat, "Esse ranking só funciona em grupo")
		}

		var (
			group    *types.GroupInfo
			texts    []senderText
			activity chatActivity
		)
		g, gctx := errgroup.WithContext(ctx)
		g.Go(func() (err error) { group, err = b.client.GetGroupInfo(gctx, chat); return err })
		g.Go(func() (err error) { texts, err = b.messages.textsBySender(gctx, chat); return err })
		g.Go(func() (err error) { activity, err = b.messages.activity(gctx, chat); return err })
		if err := g.Wait(); err != nil {
			return fmt.Errorf("montar ranking de %s: %w", m.subject, err)
		}

		ranked := b.measureMembers(chat, m, group.Participants, texts, activity)
		if len(ranked) == 0 {
			return b.sendText(ctx, chat, "Não tem ninguém pra medir aqui 🦗")
		}
		top := ranked[:min(len(ranked), rankingSize)]
		people := make([]types.JID, len(top))
		for i, member := range top {
			people[i] = member.jid
		}
		_, err := b.sendMentions(ctx, chat, meterRankingText(m, top), people)
		return err
	}
}

func (b *bot) measureMembers(chat types.JID, m meter, participants []types.GroupParticipant, texts []senderText, activity chatActivity) []measuredMember {
	owners := b.memberOwners(participants)
	textsOf := map[types.JID][]string{}
	for _, t := range texts {
		owner, ok := owners.of(t.sender, t.senderAlt)
		if !ok {
			continue
		}
		textsOf[owner] = append(textsOf[owner], t.text)
	}

	ranked := make([]measuredMember, 0, len(participants))
	for _, p := range participants {
		if owner, ok := owners[p.JID.User]; !ok || owner != p.JID {
			continue
		}
		percent := m.measure(chat, participantAliases(p), history{texts: textsOf[p.JID], activity: activity})
		ranked = append(ranked, measuredMember{jid: p.JID, percent: percent})
	}
	slices.SortFunc(ranked, func(a, b measuredMember) int {
		if a.percent != b.percent {
			return b.percent - a.percent
		}
		return strings.Compare(a.jid.User, b.jid.User)
	})
	return ranked
}

func participantAliases(p types.GroupParticipant) []string {
	var aliases []string
	for _, id := range []types.JID{p.PhoneNumber, p.LID, p.JID} {
		if id.User != "" {
			aliases = append(aliases, id.User)
		}
	}
	return aliases
}

func meterRankingText(m meter, top []measuredMember) string {
	medals := []string{"🥇", "🥈", "🥉"}
	lines := []string{fmt.Sprintf("%s *Ranking de %s do grupo*", m.emoji, m.subject), ""}
	for i, member := range top {
		position := fmt.Sprintf("%d.", i+1)
		if i < len(medals) {
			position = medals[i]
		}
		lines = append(lines, fmt.Sprintf("%s @%s: %d%% %s", position, member.jid.User, member.percent, m.subject))
	}
	return strings.Join(append(lines, "",
		fmt.Sprintf("👑 @%s é a pessoa mais %s do grupo", top[0].jid.User, m.subject),
		fmt.Sprintf("_com base nas mensagens dos %s_", messageWindowLabel),
	), "\n")
}
