package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

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
	return max(hospitalPrice(balance, 0)*(maxHealth-health)/maxHealth, 1)
}

func (b *bot) hospital(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	patient := firstMentionOrSender(evt).ToNonAD()
	dead, err := b.wallets.deathLeft(ctx, chat, b.account(ctx, patient), time.Now())
	if err != nil {
		return err
	}
	if dead == 0 {
		return b.treat(ctx, evt)
	}
	return b.buyExit(ctx, evt, exit{
		left:     b.wallets.deathLeft,
		price:    hospitalPrice,
		release:  "UPDATE carteiras SET morto_ate = 0 WHERE chat = ? AND pessoa = ? AND morto_ate > ?",
		notHeld:  "🤨 @%s tá vivo(a). O hospital só atende quem morreu ou se machucou",
		paid:     "🏥 @%s pagou %s pro hospital particular e os médicos trouxeram @%s de volta com o desfibrilador! Já pode usar a economia",
		paidSelf: "🏥 @%s pagou %s pro hospital particular direto do caixão e voltou à vida. Já pode usar a economia",
	})
}

func (b *bot) treat(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	payer, patient := evt.Info.Sender.ToNonAD(), firstMentionOrSender(evt).ToNonAD()
	payerAcc, patientAcc := b.account(ctx, payer), b.account(ctx, patient)
	price, err := b.wallets.treat(ctx, chat, payerAcc, patientAcc, time.Now())
	if errors.Is(err, errHealthy) {
		_, err = b.sendMentions(ctx, chat, fmt.Sprintf("🤨 @%s tá vivo(a) e com a saúde cheia. O hospital só atende quem morreu ou se machucou", patient.User), []types.JID{patient})
		return err
	}
	if errors.Is(err, errNoFunds) {
		return b.sendText(ctx, chat, fmt.Sprintf("O tratamento sai por %s e você não tem tudo isso. Confere no !saldo", coins(price)))
	}
	if err != nil {
		return fmt.Errorf("pagar o hospital: %w", err)
	}
	text := fmt.Sprintf("🏥 @%s pagou %s pro hospital, tomou soro na veia e saiu novinho em folha: %s", payer.User, coins(price), healthText(maxHealth))
	if payerAcc.key != patientAcc.key {
		text = fmt.Sprintf("🏥 @%s pagou %s pro hospital cuidar de @%s, que saiu novinho(a) em folha: %s", payer.User, coins(price), patient.User, healthText(maxHealth))
	}
	_, err = b.sendMentions(ctx, chat, text, []types.JID{payer, patient})
	return err
}

func (s *walletStore) treat(ctx context.Context, chat types.JID, payer, patient account, now time.Time) (price int64, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		for _, acc := range []account{payer, patient} {
			if err := openWallet(ctx, tx, chat, acc); err != nil {
				return err
			}
		}
		health, err := readHealth(ctx, tx, chat, patient, now)
		if err != nil {
			return err
		}
		if health >= maxHealth {
			return errHealthy
		}
		var balance int64
		err = tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), patient.key).Scan(&balance)
		if err != nil {
			return err
		}
		price = carePrice(balance, health)
		if err := debitBalance(ctx, tx, chat, payer, price); err != nil {
			return err
		}
		return setHealth(ctx, tx, chat, patient, maxHealth, now)
	})
	return price, err
}
