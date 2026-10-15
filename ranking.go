package main

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const rankingSize = 10

type rankingPeriod struct {
	label  string
	length time.Duration
}

type senderCount struct {
	sender    string
	senderAlt string
	messages  int
}

type rankedMember struct {
	jid      types.JID
	messages int
}

func (b *bot) ranking(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return b.sendText(ctx, chat, "O !ranking só funciona em grupo")
	}
	period := rankingPeriodFrom(messageText(evt.Message))
	since := max(time.Now().Add(-period.length).Unix(), windowStart())

	var (
		group  *types.GroupInfo
		counts []senderCount
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { group, err = b.client.GetGroupInfo(gctx, chat); return err })
	g.Go(func() (err error) { counts, err = b.messages.countsBySender(gctx, chat, since); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("montar ranking: %w", err)
	}

	ranked := b.rankMembers(group.Participants, counts)
	if len(ranked) == 0 {
		return b.sendText(ctx, chat, fmt.Sprintf("Ninguém falou nada nos %s 🦗", period.label))
	}
	top := ranked[:min(len(ranked), rankingSize)]
	people := make([]types.JID, len(top))
	for i, m := range top {
		people[i] = m.jid
	}
	_, err := b.sendMentions(ctx, chat, rankingText(period, top, ranked), people)
	return err
}

func rankingPeriodFrom(text string) rankingPeriod {
	periods := map[string]rankingPeriod{
		"hoje":   {"últimas 24 horas", 24 * time.Hour},
		"semana": {"últimos 7 dias", 7 * 24 * time.Hour},
		"mes":    {"últimos 30 dias", 30 * 24 * time.Hour},
		"tudo":   {"últimos 90 dias", messageWindow},
	}
	for _, word := range normalizedWords(text) {
		if p, ok := periods[word]; ok {
			return p
		}
	}
	return periods["semana"]
}

func (b *bot) rankMembers(participants []types.GroupParticipant, counts []senderCount) []rankedMember {
	self := map[string]bool{
		b.client.Store.GetJID().User: true,
		b.client.Store.GetLID().User: true,
	}
	owners := map[string]types.JID{}
	for _, p := range participants {
		for _, id := range []types.JID{p.JID, p.LID, p.PhoneNumber} {
			if id.User == "" || self[id.User] {
				continue
			}
			owners[id.User] = p.JID
		}
	}

	totals := map[types.JID]int{}
	for _, c := range counts {
		owner, ok := owners[c.sender]
		if !ok {
			owner, ok = owners[c.senderAlt]
		}
		if !ok {
			continue
		}
		totals[owner] += c.messages
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
	medals := []string{"🥇", "🥈", "🥉"}
	lines := []string{fmt.Sprintf("🏆 *Quem mais falou* nos %s", period.label), ""}
	for i, m := range top {
		position := fmt.Sprintf("%d.", i+1)
		if i < len(medals) {
			position = medals[i]
		}
		lines = append(lines, fmt.Sprintf("%s @%s: %s", position, m.jid.User, countLabel(m.messages, "mensagem", "mensagens")))
	}
	total := 0
	for _, m := range all {
		total += m.messages
	}
	return strings.Join(append(lines, "", fmt.Sprintf("_%s no total_", countLabel(total, "mensagem", "mensagens"))), "\n")
}

func (s *messageStore) countsBySender(ctx context.Context, chat types.JID, since int64) (counts []senderCount, err error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT remetente, remetente_alt, COUNT(*) FROM mensagens
		 WHERE chat = ? AND enviada_em >= ?
		 GROUP BY remetente, remetente_alt`,
		chat.String(), since,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var c senderCount
		if err := rows.Scan(&c.sender, &c.senderAlt, &c.messages); err != nil {
			return nil, err
		}
		counts = append(counts, c)
	}
	return counts, rows.Err()
}
