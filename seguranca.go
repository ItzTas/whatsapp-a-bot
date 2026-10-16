package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	bodyguardFloor   = 500
	bodyguardPercent = 5
	bodyguardHold    = 2 * time.Hour
	bodyguardGuard   = 0.15
)

func (p profession) escorted(on bool) profession {
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
	acc := b.account(ctx, person)
	now := time.Now()
	balance, _, err := b.wallets.balance(ctx, chat, acc)
	if err != nil {
		return fmt.Errorf("ler saldo: %w", err)
	}
	price := bodyguardPrice(balance)
	args := strings.Fields(foldLetters(messageText(evt.Message)))[1:]
	if len(args) == 0 || !slices.Contains([]string{"contratar", "comprar", "pagar"}, args[0]) {
		return b.bodyguardStatus(ctx, chat, person, acc, price, now)
	}
	until, err := b.wallets.hireBodyguard(ctx, chat, acc, price, now)
	if errors.Is(err, errNoFunds) {
		return b.sendText(ctx, chat, fmt.Sprintf("O segurança cobra %s e você não tem tudo isso. Confere no !saldo", coins(price)))
	}
	if err != nil {
		return fmt.Errorf("contratar segurança: %w", err)
	}
	text := fmt.Sprintf("🕶️ @%s contratou um segurança particular por %s! Ele fica na cola por mais %s, e a proteção contra !roubar, !sequestrar e !matar sobe %d%%",
		person.User, coins(price), formatWait(until.Sub(now)), percent(bodyguardGuard))
	_, err = b.sendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) bodyguardStatus(ctx context.Context, chat, person types.JID, acc account, price int64, now time.Time) error {
	left, err := b.wallets.leftUntil(ctx, chat, acc, "seguranca_ate", now)
	if err != nil {
		return fmt.Errorf("ler segurança: %w", err)
	}
	status := fmt.Sprintf("🕶️ @%s tá sem segurança", person.User)
	if left > 0 {
		status = fmt.Sprintf("🕶️ O segurança de @%s ainda fica %s na cola", person.User, formatWait(left))
	}
	text := fmt.Sprintf("%s\n\n_Manda !seguranca contratar pra pagar %s e ter mais %s de guarda-costas (+%d%% de proteção contra !roubar, !sequestrar e !matar). Contratar de novo soma o tempo_",
		status, coins(price), formatWait(bodyguardHold), percent(bodyguardGuard))
	_, err = b.sendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (s *walletStore) hireBodyguard(ctx context.Context, chat types.JID, acc account, price int64, now time.Time) (until time.Time, err error) {
	err = s.spend(ctx, chat, acc, price, func(tx *sql.Tx) error {
		var end int64
		err := tx.QueryRowContext(ctx,
			"UPDATE carteiras SET seguranca_ate = MAX(seguranca_ate, ?) + ? WHERE chat = ? AND pessoa = ? RETURNING seguranca_ate",
			now.Unix(), int64(bodyguardHold.Seconds()), chat.String(), acc.key,
		).Scan(&end)
		until = time.Unix(end, 0)
		return err
	})
	return until, err
}
