package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

const (
	jackpotMin       = 1_000
	jackpotWindow    = 30 * time.Minute
	jackpotRobBonus  = 0.25
	jackpotAlarmCut  = 0.2
	jackpotWaitCut   = 0.5
	jackpotShieldCut = 0.5
)

func (s *walletStore) noteJackpot(ctx context.Context, chat types.JID, acc core.Account, net int64, now time.Time) error {
	if net < jackpotMin {
		return nil
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE carteiras SET
		   premio_aposta = CASE WHEN premio_aposta_em >= ? THEN premio_aposta + ? ELSE ? END,
		   premio_aposta_em = ?
		 WHERE chat = ? AND pessoa = ?`,
		now.Add(-jackpotWindow).Unix(), net, net, now.Unix(), chat.String(), acc.Key,
	)
	return err
}

func freshJackpot(ctx context.Context, q rowQuerier, chat types.JID, acc core.Account, now time.Time) (int64, error) {
	var amount, at int64
	err := q.QueryRowContext(ctx, "SELECT MIN(premio_aposta, saldo), premio_aposta_em FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&amount, &at)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil || time.Unix(at, 0).Before(now.Add(-jackpotWindow)) {
		return 0, err
	}
	return amount, nil
}

func jackpotOdds(chance float64, jackpot int64) float64 {
	if jackpot == 0 {
		return chance
	}
	return clampChance(chance + jackpotRobBonus)
}

func jackpotAlarm(chance float64, jackpot int64) float64 {
	if jackpot == 0 {
		return chance
	}
	return max(0, chance-jackpotAlarmCut)
}

func jackpotPace(cd cooldown, jackpot int64) cooldown {
	if jackpot == 0 {
		return cd
	}
	cd.every = time.Duration(float64(cd.every) * (1 - jackpotWaitCut))
	return cd
}

func looted(balance, jackpot, percent int64) int64 {
	return exposed(balance-jackpot, percent) + jackpot*percent/100
}

func jackpotText(victim types.JID, jackpot int64) string {
	if jackpot == 0 {
		return ""
	}
	return fmt.Sprintf("💸 @%s acabou de ganhar %s na aposta e saiu contando dinheiro na rua, alvo fácil\n", victim.User, Coins(jackpot))
}

func jackpotShield(jackpot int64) time.Duration {
	if jackpot == 0 {
		return robShield
	}
	return time.Duration(float64(robShield) * (1 - jackpotShieldCut))
}
