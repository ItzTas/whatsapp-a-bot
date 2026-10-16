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

const (
	escapeChance    = 0.25
	escapePenalty   = 7 * time.Minute
	breakoutChance  = 0.2
	breakoutPenalty = 7 * time.Minute
	arrestEaseCut   = 50
)

var escapeCooldown = cooldown{column: "fuga_em", every: 5 * time.Minute}

type escape struct {
	free    bool
	captive bool
	wait    time.Duration
	success bool
	left    time.Duration
}

type getawayOdds struct {
	jail      float64
	captivity float64
}

func (b *bot) fugir(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.account(ctx, person)
	job, err := b.professionOf(ctx, chat, acc)
	if err != nil {
		return err
	}
	odds := getawayOdds{jail: job.escapeOdds(), captivity: job.breakoutOdds()}
	e, err := b.wallets.escape(ctx, chat, acc, odds, time.Now())
	if err != nil {
		return fmt.Errorf("fugir: %w", err)
	}
	_, err = b.sendMentions(ctx, chat, escapeText(e, job, person.User), []types.JID{person})
	return err
}

func (p profession) escapeOdds() float64 {
	return clampChance(escapeChance + p.escapeBonus)
}

func (p profession) breakoutOdds() float64 {
	return clampChance(breakoutChance + p.breakoutBonus)
}

func escapeText(e escape, job profession, user string) string {
	if e.free {
		return fmt.Sprintf("🤨 @%s nem tá preso nem em cativeiro. Fugir de quê?", user)
	}
	if e.captive {
		return breakoutText(e, job, user)
	}
	if e.wait > 0 {
		return fmt.Sprintf("👮 Os guardas ainda tão de olho em @%s depois da última tentativa. Espera %s pra tentar fugir de novo (ou sai em %s)", user, formatWait(e.wait), formatWait(e.left))
	}
	edge := escapeEdgeText(job, job.escapeBonus, job.escapeOdds())
	if e.success {
		return fmt.Sprintf("🏃 @%s fugiu da cadeia! Mas fuga também é crime: a polícia pode !prender de novo nas próximas %s", user, formatWait(crimeWindow)) + edge
	}
	return fmt.Sprintf("🚨 @%s tentou fugir e foi pego no muro! Ganhou mais %s de cadeia e agora só sai em %s", user, formatWait(escapePenalty), formatWait(e.left)) + edge
}

func breakoutText(e escape, job profession, user string) string {
	if e.wait > 0 {
		return fmt.Sprintf("🪢 O cativeiro tá de olho em @%s depois da última tentativa. Espera %s pra tentar fugir de novo (ou sai em %s)", user, formatWait(e.wait), formatWait(e.left))
	}
	edge := escapeEdgeText(job, job.breakoutBonus, job.breakoutOdds())
	if e.success {
		getaways := []string{
			"roeu a corda igual rato e saiu pela janela do banheiro",
			"fingiu um desmaio e correu quando abriram a porta",
			"cavou um túnel com a colher do miojo",
			"subornou o vigia do cativeiro com a senha do Wi-Fi",
			"pulou o muro e saiu correndo de chinelo",
			"esperou o sequestrador dormir vendo novela e saiu na ponta do pé",
		}
		return fmt.Sprintf("🏃 @%s %s e fugiu do cativeiro! Tá livre de novo", user, pickOne(getaways)) + edge
	}
	return fmt.Sprintf("🪢 @%s tentou fugir do cativeiro e foi pego(a) no portão! Ficou amarrado(a) mais %s e agora só sai em %s", user, formatWait(breakoutPenalty), formatWait(e.left)) + edge
}

func escapeEdgeText(job profession, bonus, odds float64) string {
	if bonus > 0 {
		return fmt.Sprintf("\n🗝️ Como %s, tinha %d%% de chance", job.title, percent(odds))
	}
	if bonus < 0 {
		return fmt.Sprintf("\n👀 Como %s, tinha só %d%% de chance", job.title, percent(odds))
	}
	return ""
}

func (s *walletStore) escape(ctx context.Context, chat types.JID, acc account, odds getawayOdds, now time.Time) (e escape, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		var jailUntil, holdUntil int64
		err := tx.QueryRowContext(ctx,
			`SELECT preso_ate, COALESCE((SELECT ate FROM sequestros WHERE chat = ? AND refem = ?), 0)
			 FROM carteiras WHERE chat = ? AND pessoa = ?`,
			chat.String(), acc.key, chat.String(), acc.key,
		).Scan(&jailUntil, &holdUntil)
		if err != nil {
			return err
		}
		jail, hold := time.Unix(jailUntil, 0).Sub(now), time.Unix(holdUntil, 0).Sub(now)
		if jail <= 0 && hold <= 0 {
			e.free = true
			return nil
		}
		e.captive, e.left = jail <= 0, jail
		if e.captive {
			e.left = hold
		}
		if e.wait, err = stampCooldown(ctx, tx, chat, acc, escapeCooldown, now); err != nil || e.wait > 0 {
			return err
		}
		if e.captive {
			e.success, e.left, err = breakOut(ctx, tx, chat, acc, e.left, odds.captivity, now)
			return err
		}
		e.success, e.left, err = breakJail(ctx, tx, chat, acc, e.left, odds.jail, now)
		return err
	})
	return e, err
}

func easeEscape(ctx context.Context, tx *sql.Tx, chat types.JID, acc account, now time.Time) error {
	ready := now.Add(-escapeCooldown.every).Unix()
	_, err := tx.ExecContext(ctx,
		"UPDATE carteiras SET fuga_em = fuga_em - (fuga_em - ?) * ? / 100 WHERE chat = ? AND pessoa = ? AND fuga_em > ?",
		ready, arrestEaseCut, chat.String(), acc.key, ready,
	)
	return err
}

func breakJail(ctx context.Context, tx *sql.Tx, chat types.JID, acc account, left time.Duration, chance float64, now time.Time) (bool, time.Duration, error) {
	if rand.Float64() < chance {
		_, err := tx.ExecContext(ctx, "UPDATE carteiras SET preso_ate = 0, crime_em = ? WHERE chat = ? AND pessoa = ?", now.Unix(), chat.String(), acc.key)
		return true, left, err
	}
	left += escapePenalty
	_, err := tx.ExecContext(ctx, "UPDATE carteiras SET preso_ate = ? WHERE chat = ? AND pessoa = ?", now.Add(left).Unix(), chat.String(), acc.key)
	return false, left, err
}

func breakOut(ctx context.Context, tx *sql.Tx, chat types.JID, acc account, left time.Duration, chance float64, now time.Time) (bool, time.Duration, error) {
	if rand.Float64() < chance {
		_, err := tx.ExecContext(ctx, "DELETE FROM sequestros WHERE chat = ? AND refem = ?", chat.String(), acc.key)
		return true, left, err
	}
	left += breakoutPenalty
	_, err := tx.ExecContext(ctx, "UPDATE sequestros SET ate = ? WHERE chat = ? AND refem = ?", now.Add(left).Unix(), chat.String(), acc.key)
	return false, left, err
}
