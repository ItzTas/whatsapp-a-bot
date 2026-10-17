package economy

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	ciprianoItem    = "livro-de-sao-cipriano"
	soulPactHold    = 3 * time.Hour
	soulPactPercent = 25
)

var reviveCooldown = cooldown{column: "ressurreicao_em", every: 45 * time.Minute}

type revival struct {
	alive   bool
	wait    time.Duration
	success bool
	pact    bool
}

func (b *bot) reviver(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	mentions := core.MentionedJIDs(evt.Message)
	if len(mentions) == 0 {
		return b.SendText(ctx, chat, "Marca quem você vai trazer de volta: !reviver @pessoa")
	}
	healer, dead := evt.Info.Sender.ToNonAD(), mentions[0].ToNonAD()
	healerAcc, deadAcc := b.Account(ctx, healer), b.Account(ctx, dead)
	if healerAcc.Key == deadAcc.Key {
		return b.SendText(ctx, chat, "Você tá vivinho da silva, não precisa de ritual 🧟")
	}
	job, err := b.ProfessionOf(ctx, chat, healerAcc)
	if err != nil {
		return err
	}
	if job.reviveChance == 0 {
		return b.SendText(ctx, chat, "🚫 Só quem mexe com o além consegue reviver alguém: "+revivingProfessions()+", o presidente, ou quem tem o Livro de São Cipriano")
	}
	r, err := b.Wallets.revive(ctx, chat, healerAcc, deadAcc, job.reviveChance, job.soulPact, job.pace(reviveCooldown), time.Now())
	if err != nil {
		return fmt.Errorf("reviver: %w", err)
	}
	_, err = b.SendMentions(ctx, chat, reviveText(r, job, healer, dead), []types.JID{healer, dead})
	return err
}

func revivingProfessions() string {
	var names []string
	for _, p := range professions {
		if p.reviveChance > 0 {
			names = append(names, fmt.Sprintf("%s %s (%d%%)", p.emoji, p.name, percent(p.reviveChance)))
		}
	}
	return core.JoinWithAnd(names)
}

func reviveText(r revival, job Profession, healer, dead types.JID) string {
	if r.alive {
		return fmt.Sprintf("🤨 @%s tá vivo(a). Reviver o quê, a vontade de trabalhar?", dead.User)
	}
	if r.wait > 0 {
		return fmt.Sprintf("🕯️ %s @%s ainda tá recarregando as energias do último ritual. Espera %s pra tentar reviver alguém de novo", job.emoji, healer.User, core.FormatWait(r.wait))
	}
	text := fmt.Sprintf("⚰️ %s @%s acendeu as velas, mas @%s não quis voltar do além. O caixão continua fechado", job.badge(), healer.User, dead.User)
	if r.success {
		text = fmt.Sprintf("🧟 %s @%s fez o ritual e @%s levantou do caixão! Tá vivo(a) de novo e já pode usar a economia", job.badge(), healer.User, dead.User)
	}
	if r.pact {
		text += fmt.Sprintf("\n📕 Só que o ritual foi pelo Livro de São Cipriano, e ninguém volta de graça: por %s, %d%% de tudo que @%s ganhar vai pra @%s", core.FormatWait(soulPactHold), soulPactPercent, dead.User, healer.User)
	}
	if job.soulPact {
		return text + "\n📕 Com o Livro de São Cipriano, o ritual não falha"
	}
	if job.president {
		return text + fmt.Sprintf("\n🏛️ Como presidente, tinha %d%% de chance", percent(job.reviveChance))
	}
	return text + fmt.Sprintf("\n✨ Como %s, tinha %d%% de chance", job.title, percent(job.reviveChance))
}

func (s *walletStore) revive(ctx context.Context, chat types.JID, healer, dead core.Account, chance float64, pact bool, cd cooldown, now time.Time) (r revival, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		if err := openWallet(ctx, tx, chat, dead); err != nil {
			return err
		}
		var deadUntil int64
		err := tx.QueryRowContext(ctx,
			"SELECT morto_ate FROM carteiras WHERE chat = ? AND pessoa = ?",
			chat.String(), dead.Key,
		).Scan(&deadUntil)
		if err != nil {
			return err
		}
		if r.alive = !time.Unix(deadUntil, 0).After(now); r.alive {
			return nil
		}
		if r.wait, err = stampCooldown(ctx, tx, chat, healer, cd, now); err != nil || r.wait > 0 {
			return err
		}
		if r.success = rand.Float64() < chance; !r.success {
			return nil
		}
		_, err = tx.ExecContext(ctx, "UPDATE carteiras SET morto_ate = 0 WHERE chat = ? AND pessoa = ?", chat.String(), dead.Key)
		if err != nil || !pact {
			return err
		}
		r.pact = true
		return grantCharm(ctx, tx, chat, dead, healer, soulPactCharm, now)
	})
	return r, err
}
