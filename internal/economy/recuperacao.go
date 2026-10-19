package economy

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

const (
	recoveryOfferChance = 0.25
	recoveryOdds        = 0.8
)

type recoveryOffer struct {
	chat    types.JID
	victim  core.Account
	thief   core.Account
	firm    firm
	amount  int64
	price   int64
	expires time.Time
}

func (o *recoveryOffer) expiresAt() time.Time { return o.expires }

func recoveryCut(m easyMark) int64 {
	if m.jackpot > 0 {
		return 45
	}
	if m.lucky {
		return 35
	}
	return 25
}

func (b *bot) offerRecovery(ctx context.Context, chat types.JID, thief, victim core.Account, r robbery) {
	if rand.Float64() >= recoveryOfferChance {
		return
	}
	now := time.Now()
	c, hired, err := contractOf(ctx, b.Wallets.db, chat, victim, now)
	if err != nil {
		b.Client.Log.Errorf("ler contrato de %s: %v", victim.JID, err)
		return
	}
	if !hired {
		c.firm, _ = firmByID(offerFirmID)
	}
	o := &recoveryOffer{chat: chat, victim: victim, thief: thief, firm: c.firm, amount: r.amount, price: r.amount * recoveryCut(r.mark) / 100, expires: now.Add(offerWindow)}
	if o.price == 0 {
		return
	}
	b.recoveryOffers.Store(OfferKey(chat, victim), o)
	text := fmt.Sprintf("%s @%s, a *%s* viu @%s fugindo com %s e pode ir atrás: %d%% de chance de trazer de volta, por %s pagos agora. Manda !aceitar em %s",
		c.firm.emoji, victim.JID.User, c.firm.name, thief.JID.User, Coins(r.amount), percent(recoveryOdds), Coins(o.price), core.FormatWait(offerWindow))
	if _, err := b.SendMentions(ctx, chat, text, []types.JID{victim.JID, thief.JID}); err != nil {
		b.Client.Log.Errorf("oferecer recuperação pra %s: %v", victim.JID, err)
	}
}

func (b *bot) acceptRecovery(ctx context.Context, o *recoveryOffer) error {
	back, found, err := b.Wallets.recover(ctx, o)
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, o.chat, fmt.Sprintf("A recuperação sai %s e você não tem tudo isso. Confere no !saldo", Coins(o.price)))
	}
	if err != nil {
		return fmt.Errorf("aceitar recuperação: %w", err)
	}
	_, err = b.SendMentions(ctx, o.chat, o.text(back, found), []types.JID{o.victim.JID, o.thief.JID})
	return err
}

func (o *recoveryOffer) text(back int64, found bool) string {
	if !found {
		return fmt.Sprintf("🏃 @%s sumiu no mundo e a %s voltou de mãos vazias. Os %s que @%s pagou não voltam", o.thief.JID.User, o.firm.label(), Coins(o.price), o.victim.JID.User)
	}
	if back == o.amount {
		return fmt.Sprintf("🚔 A %s achou @%s e trouxe de volta os %s de @%s", o.firm.label(), o.thief.JID.User, Coins(back), o.victim.JID.User)
	}
	if back == 0 {
		return fmt.Sprintf("🚔 A %s achou @%s, mas já não tinha sobrado nada dos %s de @%s", o.firm.label(), o.thief.JID.User, Coins(o.amount), o.victim.JID.User)
	}
	return fmt.Sprintf("🚔 A %s achou @%s e trouxe de volta %s dos %s de @%s, o resto já tinha sumido", o.firm.label(), o.thief.JID.User, Coins(back), Coins(o.amount), o.victim.JID.User)
}

func (s *walletStore) recover(ctx context.Context, o *recoveryOffer) (back int64, found bool, err error) {
	err = s.spend(ctx, o.chat, o.victim, o.price, func(tx *ledgerTx) error {
		if rand.Float64() >= recoveryOdds {
			return nil
		}
		found = true
		var balance int64
		if err := tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", o.chat.String(), o.thief.Key).Scan(&balance); err != nil {
			return err
		}
		if back = min(o.amount, max(balance, 0)); back == 0 {
			return nil
		}
		return moveCoins(ctx, tx, o.chat, o.thief, o.victim, back)
	})
	return back, found, err
}
