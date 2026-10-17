package economy

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	releaseFineChance = 0.5
	releaseFineMin    = 10
	releaseFineMax    = 25
)

var releaseCooldown = cooldown{column: "libertacao_em", every: 30 * time.Minute}

type release struct {
	free    bool
	captive bool
	wait    time.Duration
	success bool
	fine    int64
}

func (b *bot) libertar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	mentions := core.MentionedJIDs(evt.Message)
	if len(mentions) == 0 {
		return b.SendText(ctx, chat, "Marca quem você vai libertar: !libertar @pessoa")
	}
	liberator, prisoner := evt.Info.Sender.ToNonAD(), mentions[0].ToNonAD()
	liberatorAcc, prisonerAcc := b.Account(ctx, liberator), b.Account(ctx, prisoner)
	if liberatorAcc.Key == prisonerAcc.Key {
		return b.SendText(ctx, chat, "Se libertar sozinho tem outro nome: !fugir 🏃")
	}
	job, err := b.ProfessionOf(ctx, chat, liberatorAcc)
	if err != nil {
		return err
	}
	if job.releaseChance == 0 {
		return b.SendText(ctx, chat, "🚫 Só quem tem poder pra isso consegue libertar alguém: "+releasingProfessions()+", ou o presidente")
	}
	r, err := b.Wallets.release(ctx, chat, liberatorAcc, prisonerAcc, job.releaseChance, job.releaseFineShare(), job.pace(releaseCooldown), time.Now())
	if err != nil {
		return fmt.Errorf("libertar: %w", err)
	}
	_, err = b.SendMentions(ctx, chat, releaseText(r, job, liberator, prisoner), []types.JID{liberator, prisoner})
	return err
}

func (p Profession) releaseFineShare() float64 {
	if p.president {
		return 1 - presidentFineDiscount
	}
	return 1
}

func releasingProfessions() string {
	var names []string
	for _, p := range professions {
		if p.releaseChance > 0 {
			names = append(names, fmt.Sprintf("%s %s (%d%%)", p.emoji, p.name, percent(p.releaseChance)))
		}
	}
	return core.JoinWithAnd(names)
}

func releaseText(r release, job Profession, liberator, prisoner types.JID) string {
	if r.free {
		return fmt.Sprintf("🤨 @%s nem tá preso nem em cativeiro. Libertar de quê?", prisoner.User)
	}
	if r.wait > 0 {
		return fmt.Sprintf("📝 %s @%s ainda tá se explicando pela última soltura. Espera %s pra tentar libertar alguém de novo", job.emoji, liberator.User, core.FormatWait(r.wait))
	}
	type outcome struct{ captive, success bool }
	texts := map[outcome]string{
		{captive: false, success: true}:  "🔓 %s @%s assinou o alvará de soltura e tirou @%s da cadeia! Tá livre de novo",
		{captive: true, success: true}:   "🔓 %s @%s achou o cativeiro, negociou com os sequestradores e tirou @%s de lá! Tá livre de novo",
		{captive: false, success: false}: "🚫 %s @%s tentou soltar @%s, mas o pedido foi negado e a corregedoria ficou de olho",
		{captive: true, success: false}:  "🚫 %s @%s tentou resgatar @%s do cativeiro, mas os sequestradores despistaram e mudaram de esconderijo",
	}
	text := fmt.Sprintf(texts[outcome{r.captive, r.success}], job.badge(), liberator.User, prisoner.User)
	if r.fine > 0 {
		text += fmt.Sprintf("\n💸 E ainda levou uma multa de %s", Coins(r.fine))
	}
	if job.president && r.fine > 0 {
		return text + fmt.Sprintf(", com %d%% de desconto de presidente\n⚖️ Como presidente, tinha %d%% de chance", percent(presidentFineDiscount), percent(job.releaseChance))
	}
	if job.president {
		return text + fmt.Sprintf("\n⚖️ Como presidente, tinha %d%% de chance", percent(job.releaseChance))
	}
	return text + fmt.Sprintf("\n⚖️ Como %s, tinha %d%% de chance", job.title, percent(job.releaseChance))
}

func (s *walletStore) release(ctx context.Context, chat types.JID, liberator, prisoner core.Account, chance, fineShare float64, cd cooldown, now time.Time) (r release, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		if err := openWallet(ctx, tx, chat, prisoner); err != nil {
			return err
		}
		var jailUntil, holdUntil int64
		err := tx.QueryRowContext(ctx,
			`SELECT preso_ate, COALESCE((SELECT ate FROM sequestros WHERE chat = ? AND refem = ?), 0)
			 FROM carteiras WHERE chat = ? AND pessoa = ?`,
			chat.String(), prisoner.Key, chat.String(), prisoner.Key,
		).Scan(&jailUntil, &holdUntil)
		if err != nil {
			return err
		}
		jailed, held := time.Unix(jailUntil, 0).After(now), time.Unix(holdUntil, 0).After(now)
		if !jailed && !held {
			r.free = true
			return nil
		}
		r.captive = !jailed
		if r.wait, err = stampCooldown(ctx, tx, chat, liberator, cd, now); err != nil || r.wait > 0 {
			return err
		}
		if r.success = rand.Float64() < chance; r.success {
			return setFree(ctx, tx, chat, prisoner, r.captive)
		}
		if rand.Float64() >= releaseFineChance {
			return nil
		}
		r.fine, err = burnBalance(ctx, tx, chat, liberator, releaseFineMin, releaseFineMax, fineShare)
		return err
	})
	return r, err
}

func setFree(ctx context.Context, tx *ledgerTx, chat types.JID, prisoner core.Account, captive bool) error {
	query := map[bool]string{
		false: "UPDATE carteiras SET preso_ate = 0 WHERE chat = ? AND pessoa = ?",
		true:  "DELETE FROM sequestros WHERE chat = ? AND refem = ?",
	}[captive]
	_, err := tx.ExecContext(ctx, query, chat.String(), prisoner.Key)
	return err
}

func burnBalance(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, minPercent, maxPercent int64, share float64) (int64, error) {
	var balance int64
	err := tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&balance)
	if err != nil {
		return 0, err
	}
	fine := int64(float64(max(balance, 0)*(minPercent+rand.Int64N(maxPercent-minPercent+1))/100) * share)
	if fine == 0 {
		return 0, nil
	}
	_, err = tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo - ? WHERE chat = ? AND pessoa = ?", fine, chat.String(), acc.Key)
	return fine, err
}
