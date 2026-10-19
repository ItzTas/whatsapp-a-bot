package economy

import (
	"context"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

const (
	boxJinxWindow = 4 * time.Hour
	boxJinxMin    = 25_000
	boxJinxDraws  = 1
)

func (s *walletStore) noteRecentWinnings(ctx context.Context, chat types.JID, acc core.Account, net int64, now time.Time) error {
	if net == 0 {
		return nil
	}
	return s.inTx(ctx, func(tx *ledgerTx) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM apostas_recentes WHERE chat = ? AND pessoa = ? AND em < ?", chat.String(), acc.Key, now.Add(-boxJinxWindow).Unix())
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "INSERT INTO apostas_recentes (chat, pessoa, em, lucro) VALUES (?, ?, ?, ?)", chat.String(), acc.Key, now.Unix(), net)
		return err
	})
}

func (s *walletStore) boxJinx(ctx context.Context, chat types.JID, acc core.Account, now time.Time) (int, error) {
	var won int64
	err := s.db.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(lucro), 0) FROM apostas_recentes WHERE chat = ? AND pessoa = ? AND em >= ?",
		chat.String(), acc.Key, now.Add(-boxJinxWindow).Unix(),
	).Scan(&won)
	if err != nil || won < boxJinxMin {
		return 0, err
	}
	return boxJinxDraws, nil
}
