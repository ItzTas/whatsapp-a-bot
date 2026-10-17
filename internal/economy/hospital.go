package economy

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	hospitalFloor   = 300
	hospitalPercent = 8
)

func hospitalPrice(balance int64, _ time.Duration) int64 {
	return max(hospitalFloor, max(balance, 0)*hospitalPercent/100)
}

func carePrice(balance, health int64) int64 {
	return max(hospitalPrice(balance, 0)*(MaxHealth-health)/MaxHealth, 1)
}

func (b *bot) hospital(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	patient := core.FirstMentionOrSender(evt).ToNonAD()
	dead, err := b.Wallets.DeathLeft(ctx, chat, b.Account(ctx, patient), time.Now())
	if err != nil {
		return err
	}
	if dead == 0 {
		return b.treat(ctx, evt)
	}
	return b.buyExit(ctx, evt, exit{
		left:     b.Wallets.DeathLeft,
		price:    hospitalPrice,
		release:  "UPDATE carteiras SET morto_ate = 0 WHERE chat = ? AND pessoa = ? AND morto_ate > ?",
		notHeld:  "🤨 @%s tá vivo(a). O hospital só atende quem morreu ou se machucou",
		paid:     "🏥 @%s pagou %s pro hospital particular e os médicos trouxeram @%s de volta com o desfibrilador! Já pode usar a economia",
		paidSelf: "🏥 @%s pagou %s pro hospital particular direto do caixão e voltou à vida. Já pode usar a economia",
	})
}

func (b *bot) treat(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	payer, patient := evt.Info.Sender.ToNonAD(), core.FirstMentionOrSender(evt).ToNonAD()
	payerAcc, patientAcc := b.Account(ctx, payer), b.Account(ctx, patient)
	price, err := b.Wallets.treat(ctx, chat, payerAcc, patientAcc, time.Now())
	if errors.Is(err, errHealthy) {
		_, err = b.SendMentions(ctx, chat, fmt.Sprintf("🤨 @%s tá vivo(a) e com a saúde cheia. O hospital só atende quem morreu ou se machucou", patient.User), []types.JID{patient})
		return err
	}
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, chat, fmt.Sprintf("O tratamento sai por %s e você não tem tudo isso. Confere no !saldo", Coins(price)))
	}
	if err != nil {
		return fmt.Errorf("pagar o hospital: %w", err)
	}
	text := fmt.Sprintf("🏥 @%s pagou %s pro hospital, tomou soro na veia e saiu novinho em folha: %s", payer.User, Coins(price), healthText(MaxHealth))
	if payerAcc.Key != patientAcc.Key {
		text = fmt.Sprintf("🏥 @%s pagou %s pro hospital cuidar de @%s, que saiu novinho(a) em folha: %s", payer.User, Coins(price), patient.User, healthText(MaxHealth))
	}
	_, err = b.SendMentions(ctx, chat, text, []types.JID{payer, patient})
	return err
}

func (s *walletStore) treat(ctx context.Context, chat types.JID, payer, patient core.Account, now time.Time) (price int64, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		for _, acc := range []core.Account{payer, patient} {
			if err := openWallet(ctx, tx, chat, acc); err != nil {
				return err
			}
		}
		health, err := readHealth(ctx, tx, chat, patient, now)
		if err != nil {
			return err
		}
		if health >= MaxHealth {
			return errHealthy
		}
		var balance int64
		err = tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), patient.Key).Scan(&balance)
		if err != nil {
			return err
		}
		price = carePrice(balance, health)
		if err := debitBalance(ctx, tx, chat, payer, price); err != nil {
			return err
		}
		return setHealth(ctx, tx, chat, patient, MaxHealth, now)
	})
	return price, err
}
