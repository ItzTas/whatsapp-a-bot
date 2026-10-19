package economy

import (
	"context"
	"math"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

const (
	hangoverFloor  = 10_000
	hangoverStep   = 0.03
	hangoverMax    = 0.12
	hangoverBarren = 250
)

func hangoverOf(won int64) float64 {
	if won < hangoverFloor {
		return 0
	}
	return min(hangoverStep*(1+math.Log10(float64(won)/hangoverFloor)), hangoverMax)
}

func (s *walletStore) recentWinnings(ctx context.Context, chat types.JID, acc core.Account, now time.Time) (int64, error) {
	var won int64
	err := s.db.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(lucro), 0) FROM apostas_recentes WHERE chat = ? AND pessoa = ? AND em >= ?",
		chat.String(), acc.Key, now.Add(-boxJinxWindow).Unix(),
	).Scan(&won)
	return won, err
}

func (p Profession) hungover(won int64) Profession {
	if p.divine {
		return p
	}
	p.hangover = hangoverOf(won)
	return p
}

func (p Profession) sober(chance float64) float64 {
	return chance * (1 - p.hangover)
}

func (p Profession) shaky(chance float64) float64 {
	return max(chance, min(chance*(1+p.hangover), 0.95))
}

func crewHangover[T any](crew []T, job func(T) Profession) Profession {
	var p Profession
	for _, member := range crew {
		p.hangover = max(p.hangover, job(member).hangover)
	}
	return p
}
