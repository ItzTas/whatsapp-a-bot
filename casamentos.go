package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"golang.org/x/sync/errgroup"
)

type marriageStore struct {
	db *sql.DB
}

type marriage struct {
	id      int64
	spouses [2]types.JID
	since   time.Time
}

func (s *marriageStore) add(ctx context.Context, chat types.JID, spouses [2]types.JID, at time.Time) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO casamentos (chat, conjuge1, conjuge2, casados_em) VALUES (?, ?, ?, ?)",
		chat.ToNonAD().String(), spouses[0].ToNonAD().String(), spouses[1].ToNonAD().String(), at.Unix(),
	)
	return err
}

func (s *marriageStore) remove(ctx context.Context, marriages []marriage) error {
	ids := make([]any, len(marriages))
	for i, m := range marriages {
		ids[i] = m.id
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?, ", len(ids)), ", ")
	_, err := s.db.ExecContext(ctx, "DELETE FROM casamentos WHERE rowid IN ("+placeholders+")", ids...)
	return err
}

func (s *marriageStore) inChat(ctx context.Context, chat types.JID) (marriages []marriage, err error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT rowid, conjuge1, conjuge2, casados_em FROM casamentos WHERE chat = ? ORDER BY casados_em",
		chat.ToNonAD().String(),
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var (
			m             marriage
			first, second string
			since         int64
		)
		if err := rows.Scan(&m.id, &first, &second, &since); err != nil {
			return nil, err
		}
		if m.spouses[0], err = types.ParseJID(first); err != nil {
			return nil, err
		}
		if m.spouses[1], err = types.ParseJID(second); err != nil {
			return nil, err
		}
		m.since = time.Unix(since, 0)
		marriages = append(marriages, m)
	}
	return marriages, rows.Err()
}

func (m marriage) partnerOf(aliases []string) (types.JID, bool) {
	if slices.Contains(aliases, m.spouses[0].User) {
		return m.spouses[1], true
	}
	if slices.Contains(aliases, m.spouses[1].User) {
		return m.spouses[0], true
	}
	return types.JID{}, false
}

func (m marriage) joins(first, second []string) bool {
	partner, ok := m.partnerOf(first)
	return ok && slices.Contains(second, partner.User)
}

func marriagesOf(marriages []marriage, aliases []string) []marriage {
	return slices.DeleteFunc(slices.Clone(marriages), func(m marriage) bool {
		_, ok := m.partnerOf(aliases)
		return !ok
	})
}

func marriageLength(d time.Duration) string {
	if d < 24*time.Hour {
		return formatDuration(d)
	}
	return countLabel(int(d.Hours()/24), "dia", "dias")
}

type weddingNews struct {
	renewal  bool
	lines    []string
	mentions []types.JID
}

func (b *bot) weddingNews(ctx context.Context, chat types.JID, couple [2]types.JID) (weddingNews, error) {
	var (
		all     []marriage
		aliases [2][]string
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { all, err = b.marriages.inChat(gctx, chat); return err })
	for i, p := range couple {
		g.Go(func() error { aliases[i] = b.userAliases(gctx, p); return nil })
	}
	if err := g.Wait(); err != nil {
		return weddingNews{}, err
	}

	var news weddingNews
	if slices.ContainsFunc(all, func(m marriage) bool { return m.joins(aliases[0], aliases[1]) }) {
		news.renewal = true
		news.lines = append(news.lines, "💒 Os dois já eram casados, então só renovaram os votos")
	}
	for i, p := range couple {
		partners := otherPartners(marriagesOf(all, aliases[i]), aliases[i], aliases[1-i])
		if len(partners) == 0 {
			continue
		}
		news.lines = append(news.lines, fmt.Sprintf("⚠️ Bigamia! @%s já era casado(a) com %s", p.User, joinMentions(partners)))
		news.mentions = append(news.mentions, partners...)
	}
	return news, nil
}

func otherPartners(marriages []marriage, self, partner []string) []types.JID {
	var others []types.JID
	for _, m := range marriages {
		other, _ := m.partnerOf(self)
		if slices.Contains(partner, other.User) {
			continue
		}
		others = append(others, other)
	}
	return uniqueJIDs(others)
}

func (n weddingNews) caption() string {
	if len(n.lines) == 0 {
		return ""
	}
	return "\n\n" + strings.Join(n.lines, "\n")
}
