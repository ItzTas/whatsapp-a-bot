package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

type adoptionStore struct {
	db *sql.DB
}

type adoption struct {
	parent types.JID
	child  types.JID
}

func (s *adoptionStore) add(ctx context.Context, chat, parent, child types.JID, at time.Time) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO adocoes (chat, pai, filho, adotado_em) VALUES (?, ?, ?, ?)",
		chat.ToNonAD().String(), parent.ToNonAD().String(), child.ToNonAD().String(), at.Unix(),
	)
	return err
}

func (s *adoptionStore) inChat(ctx context.Context, chat types.JID) (adoptions []adoption, err error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT pai, filho FROM adocoes WHERE chat = ? ORDER BY adotado_em",
		chat.ToNonAD().String(),
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var (
			a             adoption
			parent, child string
		)
		if err := rows.Scan(&parent, &child); err != nil {
			return nil, err
		}
		if a.parent, err = types.ParseJID(parent); err != nil {
			return nil, err
		}
		if a.child, err = types.ParseJID(child); err != nil {
			return nil, err
		}
		adoptions = append(adoptions, a)
	}
	return adoptions, rows.Err()
}

func parentsOf(adoptions []adoption, users []string) []types.JID {
	var parents []types.JID
	for _, a := range adoptions {
		if slices.Contains(users, a.child.User) {
			parents = append(parents, a.parent)
		}
	}
	return uniqueJIDs(parents)
}

func childrenOf(adoptions []adoption, users []string) []types.JID {
	var children []types.JID
	for _, a := range adoptions {
		if slices.Contains(users, a.parent.User) {
			children = append(children, a.child)
		}
	}
	return uniqueJIDs(children)
}

func usersOf(jids []types.JID) []string {
	users := make([]string, len(jids))
	for i, j := range jids {
		users[i] = j.User
	}
	return users
}

func containsAnyUser(jids []types.JID, users []string) bool {
	return slices.ContainsFunc(jids, func(j types.JID) bool { return slices.Contains(users, j.User) })
}

func adoptiveBond(adoptions []adoption, couple [2]types.JID, aliases [2][]string) (child, parent types.JID, ok bool) {
	for i := range couple {
		if containsAnyUser(childrenOf(adoptions, aliases[i]), aliases[1-i]) {
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
	mentions := mentionedJIDs(evt.Message)
	if len(mentions) != 1 {
		return b.sendText(ctx, chat, "Marca quem você quer adotar: !adotar @pessoa")
	}
	parent, child := evt.Info.Sender.ToNonAD(), mentions[0].ToNonAD()

	var (
		relationships []relationship
		adoptions     []adoption
		aliases       [2][]string
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) {
		relationships, err = b.relationships.inChat(gctx, chat, kindMarriage, kindDating, kindLover)
		return err
	})
	g.Go(func() (err error) { adoptions, err = b.adoptions.inChat(gctx, chat); return err })
	for i, p := range []types.JID{parent, child} {
		g.Go(func() error { aliases[i] = b.userAliases(gctx, p); return nil })
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler família: %w", err)
	}

	if refusal, refused := adoptionRefusal(adoptions, relationships, aliases[0], aliases[1]); refused {
		_, err := b.sendMentions(ctx, chat, fmt.Sprintf(refusal, child.User), []types.JID{child})
		return err
	}
	if err := b.adoptions.add(ctx, chat, parent, child, time.Now()); err != nil {
		return fmt.Errorf("guardar adoção: %w", err)
	}

	spouses := otherPartners(ofKind(relationships, kindMarriage), aliases[0], nil)
	text := adoptionText(parent, child, spouses)
	_, err := b.sendMentions(ctx, chat, text, append([]types.JID{parent, child}, spouses...))
	return err
}

func adoptionRefusal(adoptions []adoption, relationships []relationship, parent, child []string) (string, bool) {
	refusals := []struct {
		applies bool
		text    string
	}{
		{slices.ContainsFunc(parent, func(u string) bool { return slices.Contains(child, u) }), "Ninguém pode se adotar, @%s 🤨"},
		{containsAnyUser(childrenOf(adoptions, parent), child), "@%s já é seu filho(a) 👶"},
		{containsAnyUser(parentsOf(adoptions, parent), child), "@%s é seu pai/mãe, não dá pra adotar 🤯"},
		{containsAnyUser(otherPartners(relationships, parent, nil), child), "@%s namora, é casado(a) ou é amante de você, não dá pra adotar 😳"},
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
		lines = append(lines, fmt.Sprintf("💍 %s agora também tem que ajudar a criar", joinMentions(spouses)))
	}
	return strings.Join(lines, "\n")
}
