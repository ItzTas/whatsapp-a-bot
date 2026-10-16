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

type datingStore struct {
	db *sql.DB
}

type dating struct {
	id       int64
	partners [2]types.JID
	since    time.Time
}

func (s *datingStore) add(ctx context.Context, chat types.JID, couple [2]types.JID, at time.Time) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO namoros (chat, namorado1, namorado2, namorando_desde) VALUES (?, ?, ?, ?)",
		chat.ToNonAD().String(), couple[0].ToNonAD().String(), couple[1].ToNonAD().String(), at.Unix(),
	)
	return err
}

func (s *datingStore) remove(ctx context.Context, datings []dating) error {
	ids := make([]any, len(datings))
	for i, d := range datings {
		ids[i] = d.id
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?, ", len(ids)), ", ")
	_, err := s.db.ExecContext(ctx, "DELETE FROM namoros WHERE rowid IN ("+placeholders+")", ids...)
	return err
}

func (s *datingStore) inChat(ctx context.Context, chat types.JID) (datings []dating, err error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT rowid, namorado1, namorado2, namorando_desde FROM namoros WHERE chat = ? ORDER BY namorando_desde",
		chat.ToNonAD().String(),
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var (
			d             dating
			first, second string
			since         int64
		)
		if err := rows.Scan(&d.id, &first, &second, &since); err != nil {
			return nil, err
		}
		if d.partners[0], err = types.ParseJID(first); err != nil {
			return nil, err
		}
		if d.partners[1], err = types.ParseJID(second); err != nil {
			return nil, err
		}
		d.since = time.Unix(since, 0)
		datings = append(datings, d)
	}
	return datings, rows.Err()
}

func (d dating) partnerOf(aliases []string) (types.JID, bool) {
	if slices.Contains(aliases, d.partners[0].User) {
		return d.partners[1], true
	}
	if slices.Contains(aliases, d.partners[1].User) {
		return d.partners[0], true
	}
	return types.JID{}, false
}

func (d dating) joins(first, second []string) bool {
	partner, ok := d.partnerOf(first)
	return ok && slices.Contains(second, partner.User)
}

func datingsOf(datings []dating, aliases []string) []dating {
	return slices.DeleteFunc(slices.Clone(datings), func(d dating) bool {
		_, ok := d.partnerOf(aliases)
		return !ok
	})
}

func datingPartnersOf(datings []dating, aliases []string) []types.JID {
	partners := make([]types.JID, 0, len(datings))
	for _, d := range datingsOf(datings, aliases) {
		partner, _ := d.partnerOf(aliases)
		partners = append(partners, partner)
	}
	return uniqueJIDs(partners)
}
