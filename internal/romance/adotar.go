package romance

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

func adoptiveBond(adoptions []core.Adoption, couple [2]types.JID, aliases [2][]string) (child, parent types.JID, ok bool) {
	for i := range couple {
		if core.ContainsAnyUser(core.ChildrenOf(adoptions, aliases[i]), aliases[1-i]) {
			return couple[1-i], couple[i], true
		}
	}
	return types.JID{}, types.JID{}, false
}

func adoptiveBondText(child, parent types.JID, action string) string {
	return fmt.Sprintf("🚫 @%s é filho(a) adotivo(a) de @%s, não dá pra %s 😳", child.User, parent.User, action)
}

func (b *bot) adotar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	mentions := core.MentionedJIDs(evt.Message)
	if len(mentions) != 1 {
		return b.SendText(ctx, chat, "Marca quem você quer adotar: !adotar @pessoa")
	}
	parent, child := evt.Info.Sender.ToNonAD(), mentions[0].ToNonAD()

	var (
		relationships []core.Relationship
		adoptions     []core.Adoption
		aliases       [2][]string
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) {
		relationships, err = b.Relationships.InChat(gctx, chat, core.KindMarriage, kindDating, kindLover)
		return err
	})
	g.Go(func() (err error) { adoptions, err = b.Adoptions.InChat(gctx, chat); return err })
	for i, p := range []types.JID{parent, child} {
		g.Go(func() error { aliases[i] = b.UserAliases(gctx, p); return nil })
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler família: %w", err)
	}

	if refusal, refused := adoptionRefusal(adoptions, relationships, aliases[0], aliases[1]); refused {
		_, err := b.SendMentions(ctx, chat, fmt.Sprintf(refusal, child.User), []types.JID{child})
		return err
	}
	if err := b.Adoptions.Add(ctx, chat, parent, child, time.Now()); err != nil {
		return fmt.Errorf("guardar adoção: %w", err)
	}

	spouses := core.OtherPartners(ofKind(relationships, core.KindMarriage), aliases[0], nil)
	text := adoptionText(parent, child, spouses)
	_, err := b.SendMentions(ctx, chat, text, append([]types.JID{parent, child}, spouses...))
	return err
}

func adoptionRefusal(adoptions []core.Adoption, relationships []core.Relationship, parent, child []string) (string, bool) {
	refusals := []struct {
		applies bool
		text    string
	}{
		{slices.ContainsFunc(parent, func(u string) bool { return slices.Contains(child, u) }), "Ninguém pode se adotar, @%s 🤨"},
		{core.ContainsAnyUser(core.ChildrenOf(adoptions, parent), child), "@%s já é seu filho(a) 👶"},
		{core.ContainsAnyUser(core.ParentsOf(adoptions, parent), child), "@%s é seu pai/mãe, não dá pra adotar 🤯"},
		{core.ContainsAnyUser(core.OtherPartners(relationships, parent, nil), child), "@%s namora, é casado(a) ou é amante de você, não dá pra adotar 😳"},
	}
	for _, r := range refusals {
		if r.applies {
			return r.text, true
		}
	}
	return "", false
}

func adoptionText(parent, child types.JID, spouses []types.JID) string {
	boasVindas := []string{
		"🧸 Já ganhou um quarto com papel de parede de dinossauro",
		"🍼 Mamadeira e fralda já tão na mochila",
		"📚 Já foi matriculado(a) na escola mais cara da cidade",
		"🎮 Ganhou um videogame no primeiro dia, vai ser mimado(a)",
		"🧹 Já tem lista de tarefas pra lavar a louça",
		"🥦 Vai ter que comer verdura todo dia",
		"📵 Toque de recolher às 22h, sem discussão",
	}
	lines := []string{
		"🍼 *Adoção*",
		"",
		fmt.Sprintf("@%s adotou @%s! Bem-vindo(a) à família 👨‍👩‍👧", parent.User, child.User),
		"",
		boasVindas[rand.IntN(len(boasVindas))],
	}
	if len(spouses) > 0 {
		lines = append(lines, fmt.Sprintf("💍 %s agora também tem que ajudar a criar", core.JoinMentions(spouses)))
	}
	return strings.Join(lines, "\n")
}
