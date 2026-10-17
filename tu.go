package main

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

type profile struct {
	person    types.JID
	name      string
	about     string
	role      string
	texts     []string
	firstSeen time.Time
	hours     hourlyCounts
	partners  []rankedMember
	laughLine string
	position  int
	ranked    int
	economy   *economyProfile
	spouses   []types.JID
	parents   []types.JID
	children  []types.JID
}

func (b *bot) tu(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := firstMentionOrSender(evt).ToNonAD()
	p := profile{person: person}
	inGroup := chat.Server == types.GroupServer
	since, until := windowStart(), time.Now().Unix()

	var (
		identities []types.JID
		group      *types.GroupInfo
		counts     []senderCount
		laughs     []senderCount
		partners   []senderCount
		marriages  []relationship
		adoptions  []adoption
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		identities = b.identities(gctx, person)
		aliases := usersOf(identities)
		h, hctx := errgroup.WithContext(gctx)
		h.Go(func() error { p.name = b.contactName(hctx, identities); return nil })
		h.Go(func() (err error) { p.texts, err = b.messages.texts(hctx, chat, aliases); return err })
		h.Go(func() (err error) { p.firstSeen, err = b.messages.firstMessageAt(hctx, chat, aliases); return err })
		h.Go(func() (err error) {
			p.hours, err = b.messages.hourlyCounts(hctx, chat, aliases, since, until)
			return err
		})
		if inGroup {
			h.Go(func() (err error) {
				partners, err = b.messages.chatPartners(hctx, chat, aliases, since, until)
				return err
			})
		}
		return h.Wait()
	})
	g.Go(func() error { p.about = b.about(gctx, person); return nil })
	g.Go(func() error {
		if !b.economyOn(gctx, chat) {
			return nil
		}
		e, err := b.economyProfile(gctx, chat, person)
		p.economy = &e
		return err
	})
	g.Go(func() (err error) { marriages, err = b.relationships.inChat(gctx, chat, kindMarriage); return err })
	g.Go(func() (err error) { adoptions, err = b.adoptions.inChat(gctx, chat); return err })
	if inGroup {
		g.Go(func() (err error) { group, err = b.client.GetGroupInfo(gctx, chat); return err })
		g.Go(func() (err error) { counts, err = b.messages.countsBySender(gctx, chat, since); return err })
		g.Go(func() (err error) { laughs, err = b.messages.laughsByTarget(gctx, chat, since); return err })
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("montar ficha: %w", err)
	}

	aliases := usersOf(identities)
	for _, m := range relationshipsOf(marriages, aliases) {
		partner, _ := m.partnerOf(aliases)
		p.spouses = append(p.spouses, partner)
	}
	p.parents = parentsOf(adoptions, aliases)
	p.children = childrenOf(adoptions, aliases)
	if group != nil {
		p.role = groupRole(group.Participants, aliases)
		ranked := b.rankMembers(group.Participants, counts)
		p.position = slices.IndexFunc(ranked, func(m rankedMember) bool { return slices.Contains(aliases, m.jid.User) }) + 1
		p.ranked = len(ranked)
		p.laughLine = retroLaughLine(b.rankMembers(group.Participants, laughs), func(m rankedMember) bool { return slices.Contains(aliases, m.jid.User) })
		p.partners = slices.DeleteFunc(b.rankMembers(group.Participants, partners), func(m rankedMember) bool {
			return slices.Contains(aliases, m.jid.User)
		})
	}

	text, mentions := p.card()
	_, err := b.sendMentions(ctx, chat, text, mentions)
	return err
}

func (b *bot) about(ctx context.Context, person types.JID) string {
	info, err := b.client.GetUserInfo(ctx, []types.JID{person})
	if err != nil {
		b.client.Log.Warnf("ler recado de %s: %v", person, err)
		return ""
	}
	return strings.TrimSpace(info[person].Status)
}

func groupRole(participants []types.GroupParticipant, aliases []string) string {
	i := slices.IndexFunc(participants, func(p types.GroupParticipant) bool {
		return containsAnyUser([]types.JID{p.JID, p.LID, p.PhoneNumber}, aliases)
	})
	if i < 0 {
		return "🚪 Saiu do grupo"
	}
	if participants[i].IsSuperAdmin {
		return "👑 Dono do grupo"
	}
	if participants[i].IsAdmin {
		return "⭐ Admin"
	}
	return "🙋 Membro"
}

func (p profile) card() (string, []types.JID) {
	lines := []string{fmt.Sprintf("🪪 *Ficha de @%s*", p.person.User), ""}
	if p.name != "" {
		lines = append(lines, "📛 Nome: "+p.name)
	}
	if p.about != "" {
		lines = append(lines, fmt.Sprintf("💭 Recado: _%s_", p.about))
	}
	if p.role != "" {
		lines = append(lines, p.role)
	}

	lines = append(lines, "", fmt.Sprintf("💬 %s nos %s", countLabel(len(p.texts), "mensagem", "mensagens"), messageWindowLabel))
	if p.position > 0 {
		lines = append(lines, fmt.Sprintf("🏆 %dº lugar no ranking, de %d pessoas", p.position, p.ranked))
	}
	if !p.firstSeen.IsZero() {
		lines = append(lines, "🗓️ Primeira mensagem que eu vi: "+brazilianDate(p.firstSeen))
	}
	if words := favoriteWords(slices.DeleteFunc(slices.Clone(p.texts), isCommand)); len(words) > 0 {
		lines = append(lines, fmt.Sprintf("🔤 Palavra favorita: *%s* (%dx)", words[0].word, words[0].count))
	}
	if p.hours.total() > 0 {
		lines = append(lines, fmt.Sprintf("⏰ Horário favorito: *%s*", hourlyRange(p.hours.peak())))
	}
	var partner []types.JID
	if len(p.partners) > 0 {
		partner = []types.JID{p.partners[0].jid}
		lines = append(lines, fmt.Sprintf("👯 Parceria de papo: @%s", p.partners[0].jid.User))
	}
	if p.laughLine != "" {
		lines = append(lines, p.laughLine)
	}
	if p.economy != nil {
		lines = append(lines, p.economy.lines()...)
	}

	lines = append(lines, "", p.spousesLine())
	if len(p.parents) > 0 {
		lines = append(lines, "🧓 Pais: "+joinMentions(p.parents))
	}
	if len(p.children) > 0 {
		lines = append(lines, "👶 Filhos: "+joinMentions(p.children))
	}

	mentions := slices.Concat([]types.JID{p.person}, partner, p.spouses, p.parents, p.children)
	return strings.Join(lines, "\n"), uniqueJIDs(mentions)
}

func (p profile) spousesLine() string {
	if len(p.spouses) == 0 {
		return "💔 Solteiro(a)"
	}
	return "💍 Casado(a) com " + joinMentions(p.spouses)
}

func (s *messageStore) firstMessageAt(ctx context.Context, chat types.JID, users []string) (time.Time, error) {
	first, second := users[0], users[len(users)-1]
	var sentAt sql.NullInt64
	err := s.db.QueryRowContext(ctx,
		`SELECT MIN(enviada_em) FROM mensagens
		 WHERE chat = ? AND (remetente IN (?, ?) OR remetente_alt IN (?, ?))`,
		chat.String(), first, second, first, second,
	).Scan(&sentAt)
	if err != nil || !sentAt.Valid {
		return time.Time{}, err
	}
	return time.Unix(sentAt.Int64, 0), nil
}
