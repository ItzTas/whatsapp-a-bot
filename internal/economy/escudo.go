package economy

import (
	"context"
	"fmt"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

const robShield = 15 * time.Minute

func (s *walletStore) shieldLeft(ctx context.Context, chat types.JID, acc core.Account, now time.Time) (time.Duration, error) {
	return s.leftUntil(ctx, chat, acc, "escudo_ate", now)
}

func shieldOn(ctx context.Context, tx *ledgerTx, chat types.JID, victim core.Account, now time.Time) (time.Duration, error) {
	var until int64
	err := tx.QueryRowContext(ctx, "SELECT escudo_ate FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), victim.Key).Scan(&until)
	return max(time.Unix(until, 0).Sub(now), 0), err
}

func raiseShield(ctx context.Context, tx *ledgerTx, chat types.JID, victim core.Account, until time.Time) error {
	_, err := tx.ExecContext(ctx, "UPDATE carteiras SET escudo_ate = ? WHERE chat = ? AND pessoa = ?", until.Unix(), chat.String(), victim.Key)
	return err
}

func shieldText(left time.Duration) string {
	return fmt.Sprintf("🛡️ Acabou de ser roubado(a): ninguém consegue roubar de novo por %s", core.FormatWait(left))
}

func shieldedText(victim types.JID, left time.Duration) string {
	return fmt.Sprintf("🛡️ @%s acabou de ser roubado(a) e a polícia tá de olho. Dá pra roubar de novo em %s", victim.User, core.FormatWait(left))
}
