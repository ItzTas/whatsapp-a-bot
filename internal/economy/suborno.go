package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	bribeFloor   = 10_000
	bribePercent = 20
	bribeHold    = 3 * time.Hour
	bribeColumn  = "propina_ate"
)

func (p Profession) paidOff(on bool) Profession {
	p.bribed = on
	return p
}

func (p Profession) untouchable() bool {
	return p.president || p.bribed
}

func (p Profession) slipsAway() bool {
	return p.untouchable() || p.elusive
}

func bribePrice(balance int64) int64 {
	return max(bribeFloor, max(balance, 0)*bribePercent/100)
}

func arrestShield(suspect types.JID, job Profession) string {
	if job.president {
		return immunityText(suspect)
	}
	if job.bribed {
		return fmt.Sprintf("💰 @%s tá com as autoridades no bolso: a viatura chegou, o delegado viu quem era e foi embora tomar um café", suspect.User)
	}
	return ""
}

func (b *bot) subornar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	now := time.Now()
	balance, _, err := b.Wallets.Balance(ctx, chat, acc)
	if err != nil {
		return fmt.Errorf("ler saldo: %w", err)
	}
	price := bribePrice(balance)
	args := strings.Fields(core.FoldLetters(core.MessageText(evt.Message)))[1:]
	if len(args) == 0 || !slices.Contains([]string{"pagar", "molhar", "comprar"}, args[0]) {
		return b.bribeStatus(ctx, chat, person, acc, price, now)
	}
	until, err := b.Wallets.payBribe(ctx, chat, acc, price, now)
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, chat, fmt.Sprintf("O envelope custa %s e você não tem tudo isso. Autoridade nenhuma se vende barato, confere no !saldo", Coins(price)))
	}
	if err != nil {
		return fmt.Errorf("subornar: %w", err)
	}
	text := fmt.Sprintf("💰 @%s passou um envelope de %s por baixo da mesa! Por mais %s ninguém te prende e o juiz decide a teu favor em qualquer !processar. O dinheiro sumiu no bolso das autoridades",
		person.User, Coins(price), core.FormatWait(until.Sub(now)))
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) bribeStatus(ctx context.Context, chat, person types.JID, acc core.Account, price int64, now time.Time) error {
	left, err := b.Wallets.leftUntil(ctx, chat, acc, bribeColumn, now)
	if err != nil {
		return fmt.Errorf("ler propina: %w", err)
	}
	status := fmt.Sprintf("💰 @%s não tem ninguém no bolso", person.User)
	if left > 0 {
		status = fmt.Sprintf("💰 As autoridades ainda ficam %s no bolso de @%s", core.FormatWait(left), person.User)
	}
	text := fmt.Sprintf("%s\n\n_Manda !subornar pagar pra soltar %s e ter mais %s sem ser preso (!prender, !denunciar, flagrante no !sequestrar, !matar, !assalto e batida no !mercadonegro) e com o juiz do teu lado no !processar. Pagar de novo soma o tempo_",
		status, Coins(price), core.FormatWait(bribeHold))
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (s *walletStore) payBribe(ctx context.Context, chat types.JID, acc core.Account, price int64, now time.Time) (until time.Time, err error) {
	err = s.spend(ctx, chat, acc, price, func(tx *sql.Tx) error {
		var end int64
		err := tx.QueryRowContext(ctx,
			"UPDATE carteiras SET propina_ate = MAX(propina_ate, ?) + ? WHERE chat = ? AND pessoa = ? RETURNING propina_ate",
			now.Unix(), int64(bribeHold.Seconds()), chat.String(), acc.Key,
		).Scan(&end)
		until = time.Unix(end, 0)
		return err
	})
	return until, err
}

func (b *bot) trialBribes(ctx context.Context, chat types.JID, round *TrialCase) (accuserPaid, defendantPaid bool) {
	if !round.sides.economy {
		return false, false
	}
	now := time.Now()
	var accuserLeft, defendantLeft time.Duration
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) {
		accuserLeft, err = b.Wallets.leftUntil(gctx, chat, b.Account(gctx, round.Accuser), bribeColumn, now)
		return err
	})
	g.Go(func() (err error) {
		defendantLeft, err = b.Wallets.leftUntil(gctx, chat, b.Account(gctx, round.Defendant), bribeColumn, now)
		return err
	})
	if err := g.Wait(); err != nil {
		b.Client.Log.Warnf("ler as propinas do !processar, o júri decide sozinho: %v", err)
		return false, false
	}
	return accuserLeft > 0, defendantLeft > 0
}

func (b *bot) riggedVerdict(ctx context.Context, chat types.JID, round *TrialCase, accuserPaid bool) string {
	briber := round.Defendant
	if accuserPaid {
		briber = round.Accuser
	}
	text, bill := round.sentence(accuserPaid)
	text = fmt.Sprintf("🧑‍⚖️ *Sentença*\n💰 @%s molhou a mão do juiz, que nem olhou a enquete\n", briber.User) + text
	return core.AppendLine(text, b.trialPenalty(ctx, chat, bill))
}
