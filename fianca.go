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
	exitFloor       = 300
	exitBasePercent = 5
	exitMaxPercent  = 40
)

var errNotHeld = errors.New("não tá mais preso")

type exit struct {
	left     func(ctx context.Context, chat types.JID, acc account, now time.Time) (time.Duration, error)
	price    func(balance int64, left time.Duration) int64
	release  string
	notHeld  string
	paid     string
	paidSelf string
	levy     pixLevy
}

func exitPrice(balance int64, left time.Duration) int64 {
	pct := min(exitBasePercent+int64(left.Minutes())/2, exitMaxPercent)
	return max(exitFloor, max(balance, 0)*pct/100)
}

func (b *bot) fianca(ctx context.Context, evt *events.Message) error {
	p, elected, err := b.wallets.presidency(ctx, evt.Info.Chat.ToNonAD(), time.Now())
	if err != nil {
		return fmt.Errorf("ler presidência: %w", err)
	}
	var levy pixLevy
	if elected {
		levy = pixLevy{president: p.president, percent: p.decrees.level(bailDecree)}
	}
	return b.buyExit(ctx, evt, exit{
		left:     b.wallets.sentenceLeft,
		price:    exitPrice,
		levy:     levy,
		release:  "UPDATE carteiras SET preso_ate = 0 WHERE chat = ? AND pessoa = ? AND preso_ate > ?",
		notHeld:  "🤨 @%s nem tá preso(a). Fiança de quê?",
		paid:     "🔓 @%s pagou %s de fiança e @%s saiu da cadeia agora mesmo. O dinheiro foi pro governo e não volta",
		paidSelf: "🔓 @%s pagou %s de fiança e saiu da cadeia agora mesmo. O dinheiro foi pro governo e não volta",
	})
}

func (b *bot) buyExit(ctx context.Context, evt *events.Message, ex exit) error {
	chat := evt.Info.Chat.ToNonAD()
	payer, held := evt.Info.Sender.ToNonAD(), firstMentionOrSender(evt).ToNonAD()
	payerAcc, heldAcc := b.account(ctx, payer), b.account(ctx, held)
	now := time.Now()
	left, err := ex.left(ctx, chat, heldAcc, now)
	if err != nil {
		return err
	}
	if left == 0 {
		_, err := b.sendMentions(ctx, chat, fmt.Sprintf(ex.notHeld, held.User), []types.JID{held})
		return err
	}
	balance, _, err := b.wallets.balance(ctx, chat, heldAcc)
	if err != nil {
		return fmt.Errorf("ler saldo: %w", err)
	}
	if payerAcc.key == ex.levy.president.key {
		ex.levy = pixLevy{}
	}
	price := ex.price(balance, left)
	tax := ex.levy.cut(price)
	err = b.wallets.spend(ctx, chat, payerAcc, price+tax, func(tx *sql.Tx) error {
		if err := releaseHeld(ctx, tx, ex.release, chat, heldAcc, now); err != nil {
			return err
		}
		return payLevy(ctx, tx, chat, ex.levy, tax)
	})
	if errors.Is(err, errNoFunds) {
		return b.sendText(ctx, chat, fmt.Sprintf("Sai por %s%s e você não tem tudo isso. Confere no !saldo", coins(price+tax), exitTaxShare(tax)))
	}
	if errors.Is(err, errNotHeld) {
		_, err := b.sendMentions(ctx, chat, fmt.Sprintf(ex.notHeld, held.User), []types.JID{held})
		return err
	}
	if err != nil {
		return fmt.Errorf("pagar a saída: %w", err)
	}
	text := fmt.Sprintf(ex.paid, payer.User, coins(price), held.User)
	if payerAcc.key == heldAcc.key {
		text = fmt.Sprintf(ex.paidSelf, payer.User, coins(price))
	}
	people := []types.JID{payer, held}
	if tax > 0 {
		people = append(people, ex.levy.president.jid)
	}
	_, err = b.sendMentions(ctx, chat, text+exitTaxText(tax, ex.levy), people)
	return err
}

func exitTaxShare(tax int64) string {
	if tax == 0 {
		return ""
	}
	return fmt.Sprintf(" (contando %s de imposto pro presidente, por decreto)", coins(tax))
}

func exitTaxText(tax int64, levy pixLevy) string {
	if tax == 0 {
		return ""
	}
	return fmt.Sprintf("\n📜 E mais %s (%d%%) de imposto da fiança foram pro bolso de @%s, por decreto", coins(tax), levy.percent, levy.president.jid.User)
}

func releaseHeld(ctx context.Context, tx *sql.Tx, statement string, chat types.JID, acc account, now time.Time) error {
	res, err := tx.ExecContext(ctx, statement, chat.String(), acc.key, now.Unix())
	if err != nil {
		return err
	}
	released, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if released == 0 {
		return errNotHeld
	}
	return nil
}
