package economy

import (
	"context"
	"database/sql"
	"errors"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

const (
	streakHeatStep = 5
	streakHeatMax  = 90
)

func (s *walletStore) betHeat(ctx context.Context, chat types.JID, acc core.Account) (int64, error) {
	var bets int64
	err := s.db.QueryRowContext(ctx,
		"SELECT aposta_serie FROM carteiras WHERE chat = ? AND pessoa = ?",
		chat.String(), acc.Key,
	).Scan(&bets)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return min(bets*streakHeatStep, streakHeatMax), err
}

func (s *walletStore) noteStreak(ctx context.Context, chat types.JID, acc core.Account, net int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE carteiras SET
		   aposta_serie = CASE WHEN aposta_lucro + ? > 0 THEN aposta_serie + 1 ELSE 0 END,
		   aposta_lucro = MAX(aposta_lucro + ?, 0)
		 WHERE chat = ? AND pessoa = ?`,
		net, net, chat.String(), acc.Key,
	)
	return err
}
