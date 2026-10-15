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

func openAdoptionStore(ctx context.Context, db *sql.DB) (*adoptionStore, error) {
	schema := `
		CREATE TABLE IF NOT EXISTS adocoes (
			chat       TEXT    NOT NULL,
			pai        TEXT    NOT NULL,
			filho      TEXT    NOT NULL,
			adotado_em INTEGER NOT NULL
		);
		CREATE INDEX IF NOT EXISTS adocoes_por_chat ON adocoes (chat);
	`
	if _, err := db.ExecContext(ctx, schema); err != nil {
		return nil, fmt.Errorf("criar tabela de adoções: %w", err)
	}
	return &adoptionStore{db: db}, nil
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

func (b *bot) adotar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	mentions := mentionedJIDs(evt.Message)
	if len(mentions) != 1 {
		return b.sendText(ctx, chat, "Marca quem você quer adotar: !adotar @pessoa")
	}
	parent, child := evt.Info.Sender.ToNonAD(), mentions[0].ToNonAD()

	var (
		marriages []marriage
		adoptions []adoption
		aliases   [2][]string
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { marriages, err = b.marriages.inChat(gctx, chat); return err })
	g.Go(func() (err error) { adoptions, err = b.adoptions.inChat(gctx, chat); return err })
	for i, p := range []types.JID{parent, child} {
		g.Go(func() error { aliases[i] = b.userAliases(gctx, p); return nil })
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler família: %w", err)
	}

	if refusal, refused := adoptionRefusal(adoptions, aliases[0], aliases[1]); refused {
		_, err := b.sendMentions(ctx, chat, fmt.Sprintf(refusal, child.User), []types.JID{child})
		return err
	}
	if err := b.adoptions.add(ctx, chat, parent, child, time.Now()); err != nil {
		return fmt.Errorf("guardar adoção: %w", err)
	}

	spouses := otherPartners(marriagesOf(marriages, aliases[0]), aliases[0], nil)
	text := adoptionText(parent, child, spouses, aliases[1])
	_, err := b.sendMentions(ctx, chat, text, append([]types.JID{parent, child}, spouses...))
	return err
}

func adoptionRefusal(adoptions []adoption, parent, child []string) (string, bool) {
	refusals := []struct {
		applies bool
		text    string
	}{
		{slices.ContainsFunc(parent, func(u string) bool { return slices.Contains(child, u) }), "Ninguém pode se adotar, @%s 🤨"},
		{containsAnyUser(childrenOf(adoptions, parent), child), "@%s já é seu filho(a) 👶"},
		{containsAnyUser(parentsOf(adoptions, parent), child), "@%s é seu pai/mãe, não dá pra adotar 🤯"},
	}
	for _, r := range refusals {
		if r.applies {
			return r.text, true
		}
	}
	return "", false
}

func adoptionText(parent, child types.JID, spouses []types.JID, childAliases []string) string {
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
	if containsAnyUser(spouses, childAliases) {
		lines = append(lines, fmt.Sprintf("⚠️ Peraí... @%s também é seu cônjuge 😳", child.User))
	}
	coParents := slices.DeleteFunc(slices.Clone(spouses), func(s types.JID) bool { return slices.Contains(childAliases, s.User) })
	if len(coParents) > 0 {
		lines = append(lines, fmt.Sprintf("💍 %s agora também tem que ajudar a criar", joinMentions(coParents)))
	}
	return strings.Join(lines, "\n")
}
