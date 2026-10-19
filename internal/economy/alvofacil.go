package economy

import (
	"context"
	"fmt"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

const (
	jackpotMin      = 1_000
	jackpotWindow   = 30 * time.Minute
	jackpotRobBonus = 0.25
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

func freshJackpot(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, balance int64, now time.Time) (int64, error) {
	var amount, at int64
	err := tx.QueryRowContext(ctx, "SELECT premio_aposta, premio_aposta_em FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&amount, &at)
	if err != nil || time.Unix(at, 0).Before(now.Add(-jackpotWindow)) {
		return 0, err
	}
	return min(amount, balance), nil
}

func spendJackpot(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account) error {
	_, err := tx.ExecContext(ctx, "UPDATE carteiras SET premio_aposta = 0, premio_aposta_em = 0 WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key)
	return err
}

func jackpotOdds(chance float64, jackpot int64) float64 {
	if jackpot == 0 {
		return chance
	}
	return clampChance(chance + jackpotRobBonus)
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

func robbedVictim(ctx context.Context, tx *ledgerTx, chat types.JID, victim core.Account, now time.Time) error {
	if err := spendJackpot(ctx, tx, chat, victim); err != nil {
		return err
	}
	return raiseShield(ctx, tx, chat, victim, now)
}
