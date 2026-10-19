package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

const (
	jackpotMin        = 1_000
	jackpotWindow     = 30 * time.Minute
	jackpotRobBonus   = 0.35
	jackpotAlarmCut   = 0.2
	jackpotWaitCut    = 0.35
	jackpotLootBonus  = 0.5
	jackpotArrestSlip = 0.3
	luckyDays         = 3
	luckyMin          = 10_000
	luckyRobBonus     = 0.1

	jackpotShieldMinCut = 0.25
	jackpotShieldMaxCut = 0.8
	jackpotShieldTop    = 1_000_000

	jackpotDeathMinPercent = 40
	jackpotDeathMaxPercent = 60
)

func spillJackpot(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, now time.Time) (int64, error) {
	jackpot, err := freshJackpot(ctx, tx, chat, acc, now)
	if err != nil || jackpot <= 0 {
		return 0, err
	}
	spilled := jackpot * (jackpotDeathMinPercent + rand.Int64N(jackpotDeathMaxPercent-jackpotDeathMinPercent+1)) / 100
	_, err = tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo - ? WHERE chat = ? AND pessoa = ?", spilled, chat.String(), acc.Key)
	return spilled, err
}

func slipsArrest(ctx context.Context, q rowQuerier, chat types.JID, acc core.Account, slip float64, now time.Time) (bool, error) {
	if slip == 0 {
		return false, nil
	}
	jackpot, err := freshJackpot(ctx, q, chat, acc, now)
	return jackpot > 0 && rand.Float64() < slip, err
}

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

type easyMark struct {
	jackpot int64
	lucky   bool
	won     int64
}

func (s *walletStore) noteWinnings(ctx context.Context, chat types.JID, acc core.Account, net int64, now time.Time) error {
	if net == 0 {
		return nil
	}
	return s.inTx(ctx, func(tx *ledgerTx) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM apostas_lucro WHERE chat = ? AND pessoa = ? AND dia < ?", chat.String(), acc.Key, luckySince(now))
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx,
			`INSERT INTO apostas_lucro (chat, pessoa, dia, lucro) VALUES (?, ?, ?, ?)
			 ON CONFLICT (chat, pessoa, dia) DO UPDATE SET lucro = lucro + excluded.lucro`,
			chat.String(), acc.Key, now.In(core.BrazilTime).Format(time.DateOnly), net,
		)
		return err
	})
}

func luckySince(now time.Time) string {
	return now.In(core.BrazilTime).AddDate(0, 0, 1-luckyDays).Format(time.DateOnly)
}

func markOf(ctx context.Context, q rowQuerier, chat types.JID, acc core.Account, now time.Time) (m easyMark, err error) {
	if m.jackpot, err = freshJackpot(ctx, q, chat, acc, now); err != nil || m.jackpot > 0 {
		return m, err
	}
	var won int64
	err = q.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(lucro), 0) FROM apostas_lucro WHERE chat = ? AND pessoa = ? AND dia >= ?",
		chat.String(), acc.Key, luckySince(now),
	).Scan(&won)
	if err != nil || won < luckyMin {
		return m, err
	}
	m.lucky, m.won = true, won
	return m, nil
}

func (m easyMark) haul() int64 {
	if m.jackpot > 0 {
		return m.jackpot
	}
	return m.won
}

func (m easyMark) odds(chance float64) float64 {
	if m.jackpot > 0 {
		return clampChance(chance + jackpotRobBonus)
	}
	if m.lucky {
		return clampChance(chance + luckyRobBonus)
	}
	return chance
}

func (m easyMark) text(victim types.JID) string {
	if m.lucky {
		return fmt.Sprintf("🍀 @%s lucrou alto na aposta esses dias e a rua inteira ficou sabendo, ainda tá mais fácil de roubar\n", victim.User)
	}
	return jackpotText(victim, m.jackpot)
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
	taken := exposed(balance, percent)
	if jackpot == 0 {
		return taken
	}
	return int64(float64(taken) * (1 + jackpotLootBonus))
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
	return time.Duration(float64(robShield) * (1 - jackpotShieldCut(jackpot))).Round(time.Second)
}

func jackpotShieldCut(jackpot int64) float64 {
	reach := math.Log(float64(max(jackpot, jackpotMin))/jackpotMin) / math.Log(jackpotShieldTop/jackpotMin)
	return jackpotShieldMinCut + (jackpotShieldMaxCut-jackpotShieldMinCut)*min(reach, 1)
}
