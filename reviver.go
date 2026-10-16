package main

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand/v2"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

var reviveCooldown = cooldown{column: "ressurreicao_em", every: 45 * time.Minute}

type revival struct {
	alive   bool
	wait    time.Duration
	success bool
}

func (b *bot) reviver(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	mentions := mentionedJIDs(evt.Message)
	if len(mentions) == 0 {
		return b.sendText(ctx, chat, "Marca quem você vai trazer de volta: !reviver @pessoa")
	}
	healer, dead := evt.Info.Sender.ToNonAD(), mentions[0].ToNonAD()
	healerAcc, deadAcc := b.account(ctx, healer), b.account(ctx, dead)
	if healerAcc.key == deadAcc.key {
		return b.sendText(ctx, chat, "Você tá vivinho da silva, não precisa de ritual 🧟")
	}
	job, err := b.professionOf(ctx, chat, healerAcc)
	if err != nil {
		return err
	}
	if job.reviveChance == 0 {
		return b.sendText(ctx, chat, "🚫 Só quem mexe com o além consegue reviver alguém: "+revivingProfessions()+", ou o presidente")
	}
	r, err := b.wallets.revive(ctx, chat, healerAcc, deadAcc, job.reviveChance, job.pace(reviveCooldown), time.Now())
	if err != nil {
		return fmt.Errorf("reviver: %w", err)
	}
	_, err = b.sendMentions(ctx, chat, reviveText(r, job, healer, dead), []types.JID{healer, dead})
	return err
}

func revivingProfessions() string {
	var names []string
	for _, p := range professions {
		if p.reviveChance > 0 {
			names = append(names, fmt.Sprintf("%s %s (%d%%)", p.emoji, p.name, percent(p.reviveChance)))
		}
	}
	return joinWithAnd(names)
}

func reviveText(r revival, job profession, healer, dead types.JID) string {
	if r.alive {
		return fmt.Sprintf("🤨 @%s tá vivo(a). Reviver o quê, a vontade de trabalhar?", dead.User)
	}
	if r.wait > 0 {
		return fmt.Sprintf("🕯️ %s @%s ainda tá recarregando as energias do último ritual. Espera %s pra tentar reviver alguém de novo", job.emoji, healer.User, formatWait(r.wait))
	}
	text := fmt.Sprintf("⚰️ %s @%s acendeu as velas, mas @%s não quis voltar do além. O caixão continua fechado", job.badge(), healer.User, dead.User)
	if r.success {
		text = fmt.Sprintf("🧟 %s @%s fez o ritual e @%s levantou do caixão! Tá vivo(a) de novo e já pode usar a economia", job.badge(), healer.User, dead.User)
	}
	if job.president {
		return text + fmt.Sprintf("\n🏛️ Como presidente, tinha %d%% de chance", percent(job.reviveChance))
	}
	return text + fmt.Sprintf("\n✨ Como %s, tinha %d%% de chance", job.title, percent(job.reviveChance))
}

func (s *walletStore) revive(ctx context.Context, chat types.JID, healer, dead account, chance float64, cd cooldown, now time.Time) (r revival, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		if err := openWallet(ctx, tx, chat, dead); err != nil {
			return err
		}
		var deadUntil int64
		err := tx.QueryRowContext(ctx,
			"SELECT morto_ate FROM carteiras WHERE chat = ? AND pessoa = ?",
			chat.String(), dead.key,
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
		_, err = tx.ExecContext(ctx, "UPDATE carteiras SET morto_ate = 0 WHERE chat = ? AND pessoa = ?", chat.String(), dead.key)
		return err
	})
	return r, err
}
