package economy

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const forgeryCaught = 0.25

func (b *bot) forgeRelease(ctx context.Context, evt *events.Message, it item) error {
	chat := evt.Info.Chat.ToNonAD()
	forger, held := evt.Info.Sender.ToNonAD(), core.FirstMentionOrSender(evt).ToNonAD()
	forgerAcc, heldAcc := b.Account(ctx, forger), b.Account(ctx, held)
	caught := rand.Float64() < forgeryCaught
	err := b.Wallets.forgeRelease(ctx, chat, forgerAcc, heldAcc, caught, time.Now())
	people := []types.JID{forger, held}
	if errors.Is(err, errNotHeld) {
		_, err := b.SendMentions(ctx, chat, fmt.Sprintf("🤨 @%s nem tá preso(a). Guarda %s pra quando precisar", held.User, it.label()), []types.JID{held})
		return err
	}
	if errors.Is(err, errNoItems) {
		return b.SendText(ctx, chat, fmt.Sprintf("Não tem mais %s na sua mochila 🎒", it.label()))
	}
	if err != nil {
		return fmt.Errorf("usar alvará falso: %w", err)
	}
	crime := fmt.Sprintf("\n🚔 Falsificar documento é crime: a polícia pode !prender @%s nas próximas %s", forger.User, core.FormatWait(crimeWindow))
	if caught {
		text := fmt.Sprintf("🚨 O carcereiro olhou o alvará contra a luz e viu que o carimbo era de batata. O papel foi pro lixo e @%s ganhou mais %s de cadeia", held.User, core.FormatWait(escapePenalty))
		_, err = b.SendMentions(ctx, chat, text+crime, people)
		return err
	}
	text := fmt.Sprintf("📜 @%s chegou na delegacia com um alvará de soltura falso no nome de @%s, e o carcereiro abriu a cela sem nem ler. Tá livre!", forger.User, held.User)
	if forgerAcc.Key == heldAcc.Key {
		text = fmt.Sprintf("📜 @%s passou um alvará de soltura falso pela grade, e o carcereiro abriu a cela sem nem ler. Tá livre!", forger.User)
	}
	_, err = b.SendMentions(ctx, chat, text+crime, people)
	return err
}

func (s *walletStore) forgeRelease(ctx context.Context, chat types.JID, forger, held core.Account, caught bool, now time.Time) error {
	release := "UPDATE carteiras SET preso_ate = 0 WHERE chat = ? AND pessoa = ? AND preso_ate > ?"
	if caught {
		release = fmt.Sprintf("UPDATE carteiras SET preso_ate = preso_ate + %d WHERE chat = ? AND pessoa = ? AND preso_ate > ?", int64(escapePenalty.Seconds()))
	}
	return s.inTx(ctx, func(tx *ledgerTx) error {
		if err := takeItems(ctx, tx, chat, forger, forgedReleaseID, 1); err != nil {
			return err
		}
		for _, acc := range []core.Account{forger, held} {
			if err := openWallet(ctx, tx, chat, acc); err != nil {
				return err
			}
		}
		if err := releaseHeld(ctx, tx, release, chat, held, now); err != nil {
			return err
		}
		return markCrime(ctx, tx, chat, forger, now)
	})
}
