package core

import (
	"context"
	"database/sql"
	"fmt"

	"go.mau.fi/whatsmeow/types"
)

type RestrictionStore struct {
	DB    *sql.DB
	Table string
}

func (s *RestrictionStore) Add(ctx context.Context, chat types.JID, ids []types.JID) (bool, error) {
	first, second := IdentityKeys(ids)
	res, err := s.DB.ExecContext(ctx,
		fmt.Sprintf(`INSERT INTO %[1]s (chat, id, id_alt)
		 SELECT ?, ?, ?
		 WHERE NOT EXISTS (
		   SELECT 1 FROM %[1]s WHERE chat = ? AND (id IN (?, ?) OR id_alt IN (?, ?))
		 )`, s.Table),
		chat.String(), first, second, chat.String(), first, second, first, second,
	)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (s *RestrictionStore) Remove(ctx context.Context, chat types.JID, ids []types.JID) (bool, error) {
	first, second := IdentityKeys(ids)
	res, err := s.DB.ExecContext(ctx,
		fmt.Sprintf("DELETE FROM %s WHERE chat = ? AND (id IN (?, ?) OR id_alt IN (?, ?))", s.Table),
		chat.String(), first, second, first, second,
	)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (s *RestrictionStore) Clear(ctx context.Context, chat types.JID) (int64, error) {
	res, err := s.DB.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE chat = ?", s.Table), chat.String())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *RestrictionStore) Has(ctx context.Context, chat types.JID, ids []types.JID) (bool, error) {
	first, second := IdentityKeys(ids)
	var found bool
	err := s.DB.QueryRowContext(ctx,
		fmt.Sprintf("SELECT EXISTS (SELECT 1 FROM %s WHERE chat = ? AND (id IN (?, ?) OR id_alt IN (?, ?)))", s.Table),
		chat.String(), first, second, first, second,
	).Scan(&found)
	return found, err
}
