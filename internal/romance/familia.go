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

type familyBranch struct {
	emoji  string
	label  string
	people []types.JID
}

func (b *bot) familia(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := core.FirstMentionOrSender(evt).ToNonAD()

	var (
		marriages []core.Relationship
		adoptions []core.Adoption
		aliases   []string
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { marriages, err = b.Relationships.InChat(gctx, chat, core.KindMarriage); return err })
	g.Go(func() (err error) { adoptions, err = b.Adoptions.InChat(gctx, chat); return err })
	g.Go(func() error { aliases = b.UserAliases(gctx, person); return nil })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler família: %w", err)
	}

	own := core.RelationshipsOf(marriages, aliases)
	branches := familyBranches(adoptions, aliases)
	if len(own) == 0 && !slices.ContainsFunc(branches, func(br familyBranch) bool { return len(br.people) > 0 }) {
		text := fmt.Sprintf("🥲 @%s ainda não tem família nenhuma por aqui. Usa !casar ou !adotar", person.User)
		_, err := b.SendMentions(ctx, chat, text, []types.JID{person})
		return err
	}

	lines := []string{fmt.Sprintf("🌳 *Família de @%s*", person.User), ""}
	mentions := []types.JID{person}
	if len(own) > 0 {
		spouses, line := spousesLine(own, aliases, time.Now())
		lines = append(lines, line)
		mentions = append(mentions, spouses...)
	}
	for _, br := range branches {
		if len(br.people) == 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s %s: %s", br.emoji, br.label, core.JoinMentions(br.people)))
		mentions = append(mentions, br.people...)
	}
	_, err := b.SendMentions(ctx, chat, strings.Join(lines, "\n"), core.UniqueJIDs(mentions))
	return err
}

func familyBranches(adoptions []core.Adoption, aliases []string) []familyBranch {
	notSelf := func(people []types.JID) []types.JID {
		return slices.DeleteFunc(people, func(p types.JID) bool { return slices.Contains(aliases, p.User) })
	}
	parents := notSelf(core.ParentsOf(adoptions, aliases))
	children := notSelf(core.ChildrenOf(adoptions, aliases))
	return []familyBranch{
		{"👴", "Avós", notSelf(core.ParentsOf(adoptions, core.UsersOf(parents)))},
		{"🧓", "Pais", parents},
		{"👫", "Irmãos", notSelf(core.ChildrenOf(adoptions, core.UsersOf(parents)))},
		{"👶", "Filhos", children},
		{"🍼", "Netos", notSelf(core.ChildrenOf(adoptions, core.UsersOf(children)))},
	}
}

func spousesLine(own []core.Relationship, aliases []string, now time.Time) ([]types.JID, string) {
	spouses := make([]types.JID, 0, len(own))
	parts := make([]string, 0, len(own))
	for _, m := range own {
		partner, _ := m.PartnerOf(aliases)
		spouses = append(spouses, partner)
		parts = append(parts, fmt.Sprintf("@%s (há %s)", partner.User, marriageLength(now.Sub(m.Since))))
	}
	return spouses, "💍 Cônjuges: " + strings.Join(parts, ", ")
}
