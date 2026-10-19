package economy

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

const (
	boostPrizeMin = 20_000
	boostCut      = 15
)

type boostOffer struct {
	chat    types.JID
	acc     core.Account
	firm    firm
	hired   bool
	price   int64
	until   time.Time
	expires time.Time
}

func (o *boostOffer) expiresAt() time.Time { return o.expires }

func (b *bot) offerBoost(ctx context.Context, chat types.JID, acc core.Account, prize int64) {
	if prize < boostPrizeMin {
		return
	}
	now := time.Now()
	c, hired, err := contractOf(ctx, b.Wallets.db, chat, acc, now)
	if err != nil {
		b.Client.Log.Errorf("ler contrato de %s: %v", acc.JID, err)
		return
	}
	if !hired {
		c.firm, _ = firmByID(offerFirmID)
	}
	c.boost = now.Add(jackpotWindow)
	o := &boostOffer{chat: chat, acc: acc, firm: c.firm, hired: hired, price: prize * boostCut / 100, until: c.boost, expires: now.Add(offerWindow)}
	b.boostOffers.Store(OfferKey(chat, acc), o)
	what := "te escoltar"
	if hired {
		what = "reforçar a tua escolta"
	}
	text := fmt.Sprintf("%s @%s, a *%s* viu teu prêmio de %s e oferece %s até a marca de alvo fácil acabar: %d%% de chance de tirar ela em cada roubo, por %s (%d%% do prêmio). Manda !aceitar em %s",
		c.firm.emoji, acc.JID.User, c.firm.name, Coins(prize), what, percent(c.odds(easyMark{jackpot: prize}, now)), Coins(o.price), boostCut, core.FormatWait(offerWindow))
	if _, err := b.SendMentions(ctx, chat, text, []types.JID{acc.JID}); err != nil {
		b.Client.Log.Errorf("oferecer reforço pra %s: %v", acc.JID, err)
	}
}

func (b *bot) acceptBoost(ctx context.Context, o *boostOffer) error {
	err := b.Wallets.boost(ctx, o, time.Now())
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, o.chat, fmt.Sprintf("A escolta sai %s e você não tem tudo isso. Confere no !saldo", Coins(o.price)))
	}
	if err != nil {
		return fmt.Errorf("aceitar reforço: %w", err)
	}
	text := fmt.Sprintf("🤝 @%s pagou %s e a %s tá colada até a marca de alvo fácil acabar", o.acc.JID.User, Coins(o.price), o.firm.label())
	_, err = b.SendMentions(ctx, o.chat, text, []types.JID{o.acc.JID})
	return err
}

func (s *walletStore) boost(ctx context.Context, o *boostOffer, now time.Time) error {
	return s.spend(ctx, o.chat, o.acc, o.price, func(tx *ledgerTx) error {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO contratos (chat, pessoa, ramo, empresa, ate, reforco_ate) VALUES (?, ?, ?, ?, ?, ?)
			 ON CONFLICT (chat, pessoa, ramo) DO UPDATE SET
			   empresa = CASE WHEN contratos.ate > ? THEN contratos.empresa ELSE excluded.empresa END,
			   ate = MAX(CASE WHEN contratos.ate > ? THEN contratos.ate ELSE 0 END, excluded.ate),
			   reforco_ate = excluded.reforco_ate`,
			o.chat.String(), o.acc.Key, securitySector, o.firm.id, o.until.Unix(), o.until.Unix(), now.Unix(), now.Unix(),
		)
		return err
	})
}
