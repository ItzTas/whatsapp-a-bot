package core

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"time"

	"go.mau.fi/whatsmeow/types"
)

type adoptionStore struct {
	db *sql.DB
}

type Adoption struct {
	parent types.JID
	child  types.JID
}

func (s *adoptionStore) Add(ctx context.Context, chat, parent, child types.JID, at time.Time) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO adocoes (chat, pai, filho, adotado_em) VALUES (?, ?, ?, ?)",
		chat.ToNonAD().String(), parent.ToNonAD().String(), child.ToNonAD().String(), at.Unix(),
	)
	return err
}

func (s *adoptionStore) InChat(ctx context.Context, chat types.JID) (adoptions []Adoption, err error) {
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
			a             Adoption
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

func ParentsOf(adoptions []Adoption, users []string) []types.JID {
	var parents []types.JID
	for _, a := range adoptions {
		if slices.Contains(users, a.child.User) {
			parents = append(parents, a.parent)
		}
	}
	return UniqueJIDs(parents)
}

func ChildrenOf(adoptions []Adoption, users []string) []types.JID {
	var children []types.JID
	for _, a := range adoptions {
		if slices.Contains(users, a.parent.User) {
			children = append(children, a.child)
		}
	}
	return UniqueJIDs(children)
}

type RelationshipKind string

const (
	KindMarriage RelationshipKind = "casamento"
)

type relationshipStore struct {
	db *sql.DB
}

type Relationship struct {
	id        int64
	Kind      RelationshipKind
	Partners  [2]types.JID
	Since     time.Time
	Consented bool
}

func (s *relationshipStore) Add(ctx context.Context, chat types.JID, kind RelationshipKind, couple [2]types.JID, at time.Time, consented bool) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO relacionamentos (chat, tipo, pessoa1, pessoa2, desde, consentido) VALUES (?, ?, ?, ?, ?, ?)",
		chat.ToNonAD().String(), string(kind), couple[0].ToNonAD().String(), couple[1].ToNonAD().String(), at.Unix(), consented,
	)
	return err
}

func (s *relationshipStore) Confirm(ctx context.Context, r Relationship) error {
	_, err := s.db.ExecContext(ctx, "UPDATE relacionamentos SET consentido = 1 WHERE rowid = ?", r.id)
	return err
}

func (s *relationshipStore) Remove(ctx context.Context, relationships []Relationship) error {
	ids := make([]any, len(relationships))
	for i, r := range relationships {
		ids[i] = r.id
	}
	_, err := s.db.ExecContext(ctx, "DELETE FROM relacionamentos WHERE rowid IN ("+SQLPlaceholders(len(ids))+")", ids...)
	return err
}

func (s *relationshipStore) InChat(ctx context.Context, chat types.JID, kinds ...RelationshipKind) (relationships []Relationship, err error) {
	args := []any{chat.ToNonAD().String()}
	for _, k := range kinds {
		args = append(args, string(k))
	}
	rows, err := s.db.QueryContext(ctx,
		"SELECT rowid, tipo, pessoa1, pessoa2, desde, consentido FROM relacionamentos WHERE chat = ? AND tipo IN ("+SQLPlaceholders(len(kinds))+") ORDER BY desde",
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var (
			r             Relationship
			first, second string
			since         int64
		)
		if err := rows.Scan(&r.id, &r.Kind, &first, &second, &since, &r.Consented); err != nil {
			return nil, err
		}
		if r.Partners[0], err = types.ParseJID(first); err != nil {
			return nil, err
		}
		if r.Partners[1], err = types.ParseJID(second); err != nil {
			return nil, err
		}
		r.Since = time.Unix(since, 0)
		relationships = append(relationships, r)
	}
	return relationships, rows.Err()
}

func (r Relationship) PartnerOf(aliases []string) (types.JID, bool) {
	if slices.Contains(aliases, r.Partners[0].User) {
		return r.Partners[1], true
	}
	if slices.Contains(aliases, r.Partners[1].User) {
		return r.Partners[0], true
	}
	return types.JID{}, false
}

func (r Relationship) Joins(first, second []string) bool {
	partner, ok := r.PartnerOf(first)
	return ok && slices.Contains(second, partner.User)
}

func RelationshipsOf(relationships []Relationship, aliases []string) []Relationship {
	return slices.DeleteFunc(slices.Clone(relationships), func(r Relationship) bool {
		_, ok := r.PartnerOf(aliases)
		return !ok
	})
}

func ConsentedOnly(relationships []Relationship) []Relationship {
	return slices.DeleteFunc(slices.Clone(relationships), func(r Relationship) bool { return !r.Consented })
}

func OtherPartners(relationships []Relationship, self, exclude []string) []types.JID {
	var others []types.JID
	for _, r := range relationships {
		other, ok := r.PartnerOf(self)
		if !ok || slices.Contains(exclude, other.User) {
			continue
		}
		others = append(others, other)
	}
	return UniqueJIDs(others)
}
