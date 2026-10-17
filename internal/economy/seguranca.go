package economy

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	bodyguardFloor   = 500
	bodyguardPercent = 5
	bodyguardHold    = 2 * time.Hour
	bodyguardGuard   = 0.15
)

func (p Profession) escorted(on bool) Profession {
	if !on {
		return p
	}
	p.escort = true
	p.guard += bodyguardGuard
	return p
}

func bodyguardPrice(balance int64) int64 {
	return max(bodyguardFloor, max(balance, 0)*bodyguardPercent/100)
}

func (b *bot) seguranca(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	now := time.Now()
	balance, _, err := b.Wallets.Balance(ctx, chat, acc)
	if err != nil {
		return fmt.Errorf("ler saldo: %w", err)
	}
	price := bodyguardPrice(balance)
	args := strings.Fields(core.FoldLetters(core.MessageText(evt.Message)))[1:]
	if len(args) == 0 || !slices.Contains([]string{"contratar", "comprar", "pagar"}, args[0]) {
		return b.bodyguardStatus(ctx, chat, person, acc, price, now)
	}
	until, err := b.Wallets.hireBodyguard(ctx, chat, acc, price, now)
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, chat, fmt.Sprintf("O segurança cobra %s e você não tem tudo isso. Confere no !saldo", Coins(price)))
	}
	if err != nil {
		return fmt.Errorf("contratar segurança: %w", err)
	}
	text := fmt.Sprintf("🕶️ @%s contratou um segurança particular por %s! Ele fica na cola por mais %s, e a proteção contra !roubar, !sequestrar e !matar sobe %d%%",
		person.User, Coins(price), core.FormatWait(until.Sub(now)), percent(bodyguardGuard))
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) bodyguardStatus(ctx context.Context, chat, person types.JID, acc core.Account, price int64, now time.Time) error {
	left, err := b.Wallets.leftUntil(ctx, chat, acc, "seguranca_ate", now)
	if err != nil {
		return fmt.Errorf("ler segurança: %w", err)
	}
	status := fmt.Sprintf("🕶️ @%s tá sem segurança", person.User)
	if left > 0 {
		status = fmt.Sprintf("🕶️ O segurança de @%s ainda fica %s na cola", person.User, core.FormatWait(left))
	}
	text := fmt.Sprintf("%s\n\n_Manda !seguranca contratar pra pagar %s e ter mais %s de guarda-costas (+%d%% de proteção contra !roubar, !sequestrar e !matar). Contratar de novo soma o tempo_",
		status, Coins(price), core.FormatWait(bodyguardHold), percent(bodyguardGuard))
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (s *walletStore) hireBodyguard(ctx context.Context, chat types.JID, acc core.Account, price int64, now time.Time) (until time.Time, err error) {
	err = s.spend(ctx, chat, acc, price, func(tx *ledgerTx) error {
		var end int64
		err := tx.QueryRowContext(ctx,
			"UPDATE carteiras SET seguranca_ate = MAX(seguranca_ate, ?) + ? WHERE chat = ? AND pessoa = ? RETURNING seguranca_ate",
			now.Unix(), int64(bodyguardHold.Seconds()), chat.String(), acc.Key,
		).Scan(&end)
		until = time.Unix(end, 0)
		return err
	})
	return until, err
}
