package economy

import (
	"context"
	"database/sql"
	"errors"
	"math"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

const (
	streakHeatStep = 3
	streakHeatMax  = 25
	solaceFloor    = 10_000
	solaceStep     = 2
	solaceMax      = 6
)

func betSolace(lost int64) int64 {
	if lost <= solaceFloor {
		return 0
	}
	return min(int64(solaceStep*math.Log10(float64(lost)/solaceFloor)), solaceMax)
}

func (s *walletStore) betRecord(ctx context.Context, chat types.JID, acc core.Account) (heat, solace int64, err error) {
	var bets, lost int64
	err = s.db.QueryRowContext(ctx,
		"SELECT aposta_serie, aposta_perda FROM carteiras WHERE chat = ? AND pessoa = ?",
		chat.String(), acc.Key,
	).Scan(&bets, &lost)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, nil
	}
	return min(bets*streakHeatStep, streakHeatMax), betSolace(lost), err
}

func (s *walletStore) noteStreak(ctx context.Context, chat types.JID, acc core.Account, net int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE carteiras SET
		   aposta_serie = CASE WHEN aposta_lucro + ? > 0 THEN aposta_serie + 1 ELSE 0 END,
		   aposta_lucro = MAX(aposta_lucro + ?, 0),
		   aposta_perda = MAX(aposta_perda - ?, 0)
		 WHERE chat = ? AND pessoa = ?`,
		net, net, net, chat.String(), acc.Key,
	)
	return err
}
