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
	exitFloor       = 300
	exitBasePercent = 5
	exitMaxPercent  = 40
)

var errNotHeld = errors.New("não tá mais preso")

type exit struct {
	left     func(ctx context.Context, chat types.JID, acc core.Account, now time.Time) (time.Duration, error)
	price    func(balance int64, left time.Duration) int64
	release  string
	notHeld  string
	paid     string
	paidSelf string
	levy     PixLevy
}

func exitPrice(balance int64, left time.Duration) int64 {
	pct := min(exitBasePercent+int64(left.Minutes())/2, exitMaxPercent)
	return max(exitFloor, max(balance, 0)*pct/100)
}

func (b *bot) fianca(ctx context.Context, evt *events.Message) error {
	p, _, err := b.Wallets.presidency(ctx, evt.Info.Chat.ToNonAD(), time.Now())
	if err != nil {
		return fmt.Errorf("ler presidência: %w", err)
	}
	levy := p.levy(bailDecree)
	return b.buyExit(ctx, evt, exit{
		left:     b.Wallets.SentenceLeft,
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
	payer, held := evt.Info.Sender.ToNonAD(), core.FirstMentionOrSender(evt).ToNonAD()
	payerAcc, heldAcc := b.Account(ctx, payer), b.Account(ctx, held)
	now := time.Now()
	left, err := ex.left(ctx, chat, heldAcc, now)
	if err != nil {
		return err
	}
	if left == 0 {
		_, err := b.SendMentions(ctx, chat, fmt.Sprintf(ex.notHeld, held.User), []types.JID{held})
		return err
	}
	balance, _, err := b.Wallets.Balance(ctx, chat, heldAcc)
	if err != nil {
		return fmt.Errorf("ler saldo: %w", err)
	}
	if ex.levy.collected() && payerAcc.Key == ex.levy.president.Key {
		ex.levy = PixLevy{}
	}
	price := ex.price(balance, left)
	tax := ex.levy.cut(price)
	err = b.Wallets.spend(ctx, chat, payerAcc, price+tax, func(tx *ledgerTx) error {
		if err := releaseHeld(ctx, tx, ex.release, chat, heldAcc, now); err != nil {
			return err
		}
		return payLevy(ctx, tx, chat, ex.levy, tax)
	})
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, chat, fmt.Sprintf("Sai por %s%s e você não tem tudo isso. Confere no !saldo", Coins(price+tax), exitTaxShare(tax, ex.levy)))
	}
	if errors.Is(err, errNotHeld) {
		_, err := b.SendMentions(ctx, chat, fmt.Sprintf(ex.notHeld, held.User), []types.JID{held})
		return err
	}
	if err != nil {
		return fmt.Errorf("pagar a saída: %w", err)
	}
	text := fmt.Sprintf(ex.paid, payer.User, Coins(price), held.User)
	if payerAcc.Key == heldAcc.Key {
		text = fmt.Sprintf(ex.paidSelf, payer.User, Coins(price))
	}
	people := []types.JID{payer, held}
	if tax > 0 {
		people = append(people, ex.levy.payee()...)
	}
	_, err = b.SendMentions(ctx, chat, text+exitTaxText(tax, ex.levy), people)
	return err
}

func exitTaxShare(tax int64, levy PixLevy) string {
	if tax == 0 {
		return ""
	}
	return fmt.Sprintf(" (contando %s de imposto %s, por decreto)", Coins(tax), levy.office())
}

func exitTaxText(tax int64, levy PixLevy) string {
	if tax == 0 {
		return ""
	}
	return fmt.Sprintf("\n📜 E mais %s (%d%%) de imposto da fiança foram %s, por decreto", Coins(tax), levy.percent, levy.pocket())
}

func releaseHeld(ctx context.Context, tx *ledgerTx, statement string, chat types.JID, acc core.Account, now time.Time) error {
	res, err := tx.ExecContext(ctx, statement, chat.String(), acc.Key, now.Unix())
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
