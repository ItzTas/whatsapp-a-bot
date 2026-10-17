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
	rescueCaught    = 0.5
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

type rescue struct {
	locked  bool
	free    bool
	captive bool
	wait    time.Duration
	success bool
	caught  bool
	jail    time.Duration
	fine    int64
}

type rescueOdds struct {
	getaway getawayOdds
	caught  float64
	jail    time.Duration
}

func (b *bot) fugir(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.account(ctx, person)
	job, err := b.professionOf(ctx, chat, acc)
	if err != nil {
		return err
	}
	if mentions := mentionedJIDs(evt.Message); len(mentions) > 0 {
		friend := mentions[0].ToNonAD()
		if friendAcc := b.account(ctx, friend); friendAcc.key != acc.key {
			return b.helpEscape(ctx, chat, job, person, friend, acc, friendAcc)
		}
	}
	odds := getawayOdds{jail: job.escapeOdds(), captivity: job.breakoutOdds()}
	e, err := b.wallets.escape(ctx, chat, acc, odds, time.Now())
	if err != nil {
		return fmt.Errorf("fugir: %w", err)
	}
	_, err = b.sendMentions(ctx, chat, escapeText(e, job, person.User), []types.JID{person})
	return err
}

func (b *bot) helpEscape(ctx context.Context, chat types.JID, job profession, helper, friend types.JID, helperAcc, friendAcc account) error {
	odds := rescueOdds{
		getaway: getawayOdds{jail: job.escapeOdds(), captivity: job.breakoutOdds()},
		caught:  rescueCaught,
		jail:    profession{jailTime: baseJail}.sentence(job),
	}
	if job.slipsAway() {
		odds.caught = 0
	}
	r, err := b.wallets.rescue(ctx, chat, helperAcc, friendAcc, odds, time.Now())
	if err != nil {
		return fmt.Errorf("ajudar a fugir: %w", err)
	}
	_, err = b.sendMentions(ctx, chat, rescueText(r, job, helper, friend), []types.JID{helper, friend})
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

func rescueText(r rescue, job profession, helper, friend types.JID) string {
	if r.locked {
		return fmt.Sprintf("🔒 @%s tá trancado(a) e não tem como ajudar ninguém a fugir. Tenta !fugir sozinho primeiro", helper.User)
	}
	if r.free {
		return fmt.Sprintf("🤨 @%s nem tá preso nem em cativeiro. Ajudar a fugir de quê?", friend.User)
	}
	if r.wait > 0 {
		return fmt.Sprintf("👮 Os guardas ainda tão de olho em @%s depois da última fuga. Espera %s pra tentar de novo", helper.User, formatWait(r.wait))
	}
	if r.captive {
		return captiveRescueText(r, job, helper, friend)
	}
	edge := escapeEdgeText(job, job.escapeBonus, job.escapeOdds())
	if r.success {
		plans := []string{
			"escondeu uma lima dentro de um bolo de fubá e serrou as grades de",
			"cavou um túnel com uma colher até a cela de",
			"distraiu o guarda com um pastel de feira e abriu a cela de",
			"chegou de helicóptero alugado no pátio e puxou",
			"se vestiu de advogado, entrou na visita e saiu de braço dado com",
		}
		return fmt.Sprintf("🪚 @%s %s @%s e os dois saíram da cadeia! Mas ajudar fuga é crime: a polícia pode !prender os dois nas próximas %s", helper.User, pickOne(plans), friend.User, formatWait(crimeWindow)) + edge
	}
	if r.caught {
		return fmt.Sprintf("🚔 @%s tentou tirar @%s da cadeia e foi pego(a) no pátio! Agora os dois tão presos: @%s fica %s na cadeia", helper.User, friend.User, helper.User, formatWait(r.jail)) + jailFactorText(helper, job) + fineText(r.fine) + edge
	}
	return fmt.Sprintf("🚨 @%s tentou tirar @%s da cadeia, mas o alarme tocou e teve que vazar correndo. Escapou dessa vez, mas a ficha ficou suja: a polícia pode !prender nas próximas %s", helper.User, friend.User, formatWait(crimeWindow)) + edge
}

func captiveRescueText(r rescue, job profession, helper, friend types.JID) string {
	edge := escapeEdgeText(job, job.breakoutBonus, job.breakoutOdds())
	if r.success {
		return fmt.Sprintf("🏃 @%s achou o cativeiro, cortou as cordas e tirou @%s de lá! Tá livre de novo", helper.User, friend.User) + edge
	}
	return fmt.Sprintf("🪢 @%s tentou tirar @%s do cativeiro, mas os sequestradores viram e teve que vazar", helper.User, friend.User) + edge
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
		jailUntil, holdUntil, err := confinement(ctx, tx, chat, acc)
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

func (s *walletStore) rescue(ctx context.Context, chat types.JID, helper, friend account, odds rescueOdds, now time.Time) (r rescue, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		helperJail, helperHold, err := confinement(ctx, tx, chat, helper)
		if err != nil {
			return err
		}
		if time.Unix(helperJail, 0).After(now) || time.Unix(helperHold, 0).After(now) {
			r.locked = true
			return nil
		}
		friendJail, friendHold, err := confinement(ctx, tx, chat, friend)
		if err != nil {
			return err
		}
		jailed, held := time.Unix(friendJail, 0).After(now), time.Unix(friendHold, 0).After(now)
		if !jailed && !held {
			r.free = true
			return nil
		}
		r.captive = !jailed
		if r.wait, err = stampCooldown(ctx, tx, chat, helper, escapeCooldown, now); err != nil || r.wait > 0 {
			return err
		}
		if r.captive {
			if r.success = rand.Float64() < odds.getaway.captivity; r.success {
				return setFree(ctx, tx, chat, friend, true)
			}
			return nil
		}
		if r.success = rand.Float64() < odds.getaway.jail; r.success {
			return bustOut(ctx, tx, chat, helper, friend, now)
		}
		if r.caught = rand.Float64() < odds.caught; !r.caught {
			return markCrime(ctx, tx, chat, helper, now)
		}
		r.jail = odds.jail
		_, err = tx.ExecContext(ctx,
			"UPDATE carteiras SET preso_ate = ?, crime_em = 0 WHERE chat = ? AND pessoa = ?",
			now.Add(r.jail).Unix(), chat.String(), helper.key,
		)
		if err != nil {
			return err
		}
		r.fine, err = lockUp(ctx, tx, chat, helper, now)
		return err
	})
	return r, err
}

func bustOut(ctx context.Context, tx *sql.Tx, chat types.JID, helper, friend account, now time.Time) error {
	_, err := tx.ExecContext(ctx, "UPDATE carteiras SET preso_ate = 0, crime_em = ? WHERE chat = ? AND pessoa = ?", now.Unix(), chat.String(), friend.key)
	if err != nil {
		return err
	}
	return markCrime(ctx, tx, chat, helper, now)
}

func confinement(ctx context.Context, tx *sql.Tx, chat types.JID, acc account) (jailUntil, holdUntil int64, err error) {
	if err := openWallet(ctx, tx, chat, acc); err != nil {
		return 0, 0, err
	}
	err = tx.QueryRowContext(ctx,
		`SELECT preso_ate, COALESCE((SELECT ate FROM sequestros WHERE chat = ? AND refem = ?), 0)
		 FROM carteiras WHERE chat = ? AND pessoa = ?`,
		chat.String(), acc.key, chat.String(), acc.key,
	).Scan(&jailUntil, &holdUntil)
	return jailUntil, holdUntil, err
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
