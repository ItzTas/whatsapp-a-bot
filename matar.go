package main

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	killCaughtChance = 0.5
	deathHold        = 7 * time.Minute
)

var killCooldown = cooldown{column: "assassinato_em", every: 90 * time.Minute}

type killing struct {
	lawful  bool
	dead    time.Duration
	jailed  bool
	wait    time.Duration
	success bool
	jail    time.Duration
	freed   types.JID
	fine    int64
	bounty  int64
}

type killOdds struct {
	lawful  bool
	success float64
	caught  float64
	jail    time.Duration
}

func (b *bot) matar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	mentions := mentionedJIDs(evt.Message)
	if len(mentions) == 0 {
		return b.sendText(ctx, chat, "Marca quem você vai apagar: !matar @pessoa")
	}
	killer, victim := evt.Info.Sender.ToNonAD(), mentions[0].ToNonAD()
	killerAcc, victimAcc := b.account(ctx, killer), b.account(ctx, victim)
	if killerAcc.key == victimAcc.key {
		return b.sendText(ctx, chat, "Se matar não 💛 Se a coisa tiver pesada de verdade, o CVV atende no 188, a qualquer hora")
	}

	killerJob, victimJob, owed, err := b.rivalsWithDebt(ctx, chat, killerAcc, victimAcc)
	if err != nil {
		return err
	}
	if killerJob.killChance == 0 && owed == nil {
		return b.sendText(ctx, chat, "🚫 Só quem é do crime pesado pode matar: "+killingProfessions()+". Ou quem anda armado (tem "+lethalWeapons()+" no !mercadonegro), ou quem emprestou moedas com o !agiota pra alguém que passou do prazo")
	}
	odds := killOdds{
		success: killerJob.killOdds(victimJob),
		caught:  victimJob.killCaughtOdds(),
		jail:    profession{jailTime: baseJail}.sentence(killerJob),
	}
	if killerJob.untouchable() {
		odds.caught = 0
	}
	if owed != nil {
		odds.success = max(odds.success, clampChance(debtKillChance-victimJob.guard))
		odds.caught, odds.lawful = 0, true
	}
	k, err := b.wallets.kill(ctx, chat, killerAcc, victimAcc, odds, killerJob.pace(killCooldown), time.Now())
	if err != nil {
		return fmt.Errorf("matar: %w", err)
	}
	people := []types.JID{killer, victim}
	if !k.freed.IsEmpty() {
		people = append(people, k.freed)
	}
	text := killText(k, killerJob, victimJob, killer, victim)
	if k.lawful {
		text += debtCollectionText(owed)
	}
	_, err = b.sendMentions(ctx, chat, text, people)
	return err
}

func (p profession) killOdds(victim profession) float64 {
	if p.divine {
		return 1
	}
	return clampChance(p.killChance - victim.guard - p.curfew())
}

func (p profession) killCaughtOdds() float64 {
	if p.jailTime > 0 {
		return killCaughtChance + copVictimCaught
	}
	return killCaughtChance
}

func killingProfessions() string {
	var names []string
	for _, p := range professions {
		if p.killChance > 0 {
			names = append(names, fmt.Sprintf("%s %s (%d%%)", p.emoji, p.name, percent(p.killChance)))
		}
	}
	return joinWithAnd(names)
}

func killText(k killing, killerJob, victimJob profession, killer, victim types.JID) string {
	if k.dead > 0 {
		return fmt.Sprintf("⚰️ @%s já tá morto(a) e só ressuscita em %s. Deixa a pessoa descansar", victim.User, formatWait(k.dead))
	}
	if k.jailed {
		return fmt.Sprintf("🔒 @%s tá na cadeia, nem dá pra alcançar", victim.User)
	}
	if k.wait > 0 {
		return fmt.Sprintf("🧽 @%s ainda tá sumindo com as provas do último. Espera %s pra tentar de novo", killer.User, formatWait(k.wait))
	}
	if k.success {
		return deathCaption(k, killer, victim)
	}
	text := fmt.Sprintf("🩹 @%s tentou apagar @%s, mas errou feio e a vítima saiu só com um susto", killer.User, victim.User)
	if k.lawful {
		return text
	}
	if k.jail == 0 {
		return text + fmt.Sprintf("\n🚔 Tentativa de homicídio também é crime: a polícia pode !prender @%s nas próximas %s", killer.User, formatWait(crimeWindow))
	}
	if victimJob.jailTime > 0 {
		return text + fmt.Sprintf("\n👮 Só que @%s %s, reagiu e deu voz de prisão na hora! @%s pegou %s de cadeia", victim.User, victimJob.title, killer.User, formatWait(k.jail)) + jailFactorText(killer, killerJob) + fineText(k.fine)
	}
	return text + fmt.Sprintf("\n🚓 Alguém chamou a polícia e @%s foi em cana: %s de cadeia", killer.User, formatWait(k.jail)) + jailFactorText(killer, killerJob) + fineText(k.fine)
}

func deathCaption(k killing, killer, victim types.JID) string {
	scenes := []string{
		"🍌 @%s deixou uma casca de banana no caminho de @%s e foi fatal",
		"🥧 @%s serviu uma torta envenenada pra @%s, que ainda repetiu",
		"🎹 @%s derrubou um piano em cima de @%s, igual desenho animado",
		"🔪 @%s chamou @%s pra um beco escuro e voltou sozinho, assobiando",
		"☕ @%s colocou alguma coisa esquisita no cafezinho de @%s",
		"🛗 @%s empurrou @%s no poço do elevador e fingiu que nem viu",
		"🪤 @%s armou uma emboscada e @%s caiu direitinho",
	}
	lines := []string{
		fmt.Sprintf(pickOne(scenes), killer.User, victim.User),
		"",
		fmt.Sprintf("⚰️ @%s tá morto(a) por %s: até ressuscitar, nada de comando de economia (nem !saldo, !trabalhar ou !daily). Um necromante ou o presidente pode tentar !reviver", victim.User, formatWait(deathHold)),
	}
	if !k.freed.IsEmpty() {
		lines = append(lines, fmt.Sprintf("🔓 O refém de @%s aproveitou o velório pra fugir: @%s tá livre!", victim.User, k.freed.User))
	}
	if k.bounty > 0 {
		lines = append(lines, fmt.Sprintf("🎯 Tinha recompensa na cabeça de @%s: @%s embolsou %s", victim.User, killer.User, coins(k.bounty)))
	}
	if k.lawful {
		return strings.Join(lines, "\n")
	}
	lines = append(lines, fmt.Sprintf("🚔 Homicídio é crime: a polícia pode !prender @%s nas próximas %s, e qualquer um pode !denunciar", killer.User, formatWait(crimeWindow)))
	return strings.Join(lines, "\n")
}

func (b *bot) holdIfDead(ctx context.Context, evt *events.Message) bool {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	left, err := b.wallets.deathLeft(ctx, chat, b.account(ctx, person), time.Now())
	if err != nil {
		b.client.Log.Errorf("checar se tá morto: %v", err)
		return false
	}
	if left == 0 {
		return false
	}
	text := fmt.Sprintf("⚰️ @%s tá morto(a) e só ressuscita em %s. Até lá nada de comando de economia, a não ser que alguém consiga !reviver ou você pague o !hospital", person.User, formatWait(left))
	if _, err := b.sendMentions(ctx, chat, text, []types.JID{person}); err != nil {
		b.client.Log.Errorf("avisar que tá morto: %v", err)
	}
	return true
}

func (s *walletStore) kill(ctx context.Context, chat types.JID, killer, victim account, odds killOdds, cd cooldown, now time.Time) (k killing, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		for _, acc := range []account{killer, victim} {
			if err := openWallet(ctx, tx, chat, acc); err != nil {
				return err
			}
		}
		var jailUntil, deadUntil int64
		err := tx.QueryRowContext(ctx,
			"SELECT preso_ate, morto_ate FROM carteiras WHERE chat = ? AND pessoa = ?",
			chat.String(), victim.key,
		).Scan(&jailUntil, &deadUntil)
		if err != nil {
			return err
		}
		if k.dead = max(time.Unix(deadUntil, 0).Sub(now), 0); k.dead > 0 {
			return nil
		}
		if k.jailed = time.Unix(jailUntil, 0).After(now); k.jailed {
			return nil
		}
		if k.wait, err = stampCooldown(ctx, tx, chat, killer, cd, now); err != nil || k.wait > 0 {
			return err
		}
		if k.lawful = odds.lawful; !k.lawful {
			if err := markCrime(ctx, tx, chat, killer, now); err != nil {
				return err
			}
		}
		if k.success = rand.Float64() < odds.success; k.success {
			_, err := tx.ExecContext(ctx,
				"UPDATE carteiras SET morto_ate = ? WHERE chat = ? AND pessoa = ?",
				now.Add(deathHold).Unix(), chat.String(), victim.key,
			)
			if err != nil {
				return err
			}
			if k.freed, err = freeHostage(ctx, tx, chat, victim, now); err != nil {
				return err
			}
			k.bounty, err = claimBounty(ctx, tx, chat, victim, killer)
			return err
		}
		if rand.Float64() >= odds.caught {
			return nil
		}
		k.jail = odds.jail
		_, err = tx.ExecContext(ctx,
			"UPDATE carteiras SET preso_ate = ?, crime_em = 0 WHERE chat = ? AND pessoa = ?",
			now.Add(k.jail).Unix(), chat.String(), killer.key,
		)
		if err != nil {
			return err
		}
		k.fine, err = lockUp(ctx, tx, chat, killer, now)
		return err
	})
	return k, err
}
