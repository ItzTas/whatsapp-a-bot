package main

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
)

type relationshipKind string

const (
	kindDating   relationshipKind = "namoro"
	kindMarriage relationshipKind = "casamento"
	kindLover    relationshipKind = "amante"
)

type relationshipStore struct {
	db *sql.DB
}

type relationship struct {
	id        int64
	kind      relationshipKind
	partners  [2]types.JID
	since     time.Time
	consented bool
}

func (s *relationshipStore) add(ctx context.Context, chat types.JID, kind relationshipKind, couple [2]types.JID, at time.Time, consented bool) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO relacionamentos (chat, tipo, pessoa1, pessoa2, desde, consentido) VALUES (?, ?, ?, ?, ?, ?)",
		chat.ToNonAD().String(), string(kind), couple[0].ToNonAD().String(), couple[1].ToNonAD().String(), at.Unix(), consented,
	)
	return err
}

func (s *relationshipStore) confirm(ctx context.Context, r relationship) error {
	_, err := s.db.ExecContext(ctx, "UPDATE relacionamentos SET consentido = 1 WHERE rowid = ?", r.id)
	return err
}

func (s *relationshipStore) remove(ctx context.Context, relationships []relationship) error {
	ids := make([]any, len(relationships))
	for i, r := range relationships {
		ids[i] = r.id
	}
	_, err := s.db.ExecContext(ctx, "DELETE FROM relacionamentos WHERE rowid IN ("+sqlPlaceholders(len(ids))+")", ids...)
	return err
}

func (s *relationshipStore) inChat(ctx context.Context, chat types.JID, kinds ...relationshipKind) (relationships []relationship, err error) {
	args := []any{chat.ToNonAD().String()}
	for _, k := range kinds {
		args = append(args, string(k))
	}
	rows, err := s.db.QueryContext(ctx,
		"SELECT rowid, tipo, pessoa1, pessoa2, desde, consentido FROM relacionamentos WHERE chat = ? AND tipo IN ("+sqlPlaceholders(len(kinds))+") ORDER BY desde",
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var (
			r             relationship
			first, second string
			since         int64
		)
		if err := rows.Scan(&r.id, &r.kind, &first, &second, &since, &r.consented); err != nil {
			return nil, err
		}
		if r.partners[0], err = types.ParseJID(first); err != nil {
			return nil, err
		}
		if r.partners[1], err = types.ParseJID(second); err != nil {
			return nil, err
		}
		r.since = time.Unix(since, 0)
		relationships = append(relationships, r)
	}
	return relationships, rows.Err()
}

func sqlPlaceholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?, ", n), ", ")
}

func (r relationship) partnerOf(aliases []string) (types.JID, bool) {
	if slices.Contains(aliases, r.partners[0].User) {
		return r.partners[1], true
	}
	if slices.Contains(aliases, r.partners[1].User) {
		return r.partners[0], true
	}
	return types.JID{}, false
}

func (r relationship) joins(first, second []string) bool {
	partner, ok := r.partnerOf(first)
	return ok && slices.Contains(second, partner.User)
}

func relationshipsOf(relationships []relationship, aliases []string) []relationship {
	return slices.DeleteFunc(slices.Clone(relationships), func(r relationship) bool {
		_, ok := r.partnerOf(aliases)
		return !ok
	})
}

func ofKind(relationships []relationship, kind relationshipKind) []relationship {
	return slices.DeleteFunc(slices.Clone(relationships), func(r relationship) bool { return r.kind != kind })
}

func consentedOnly(relationships []relationship) []relationship {
	return slices.DeleteFunc(slices.Clone(relationships), func(r relationship) bool { return !r.consented })
}

func otherPartners(relationships []relationship, self, exclude []string) []types.JID {
	var others []types.JID
	for _, r := range relationships {
		other, ok := r.partnerOf(self)
		if !ok || slices.Contains(exclude, other.User) {
			continue
		}
		others = append(others, other)
	}
	return uniqueJIDs(others)
}
