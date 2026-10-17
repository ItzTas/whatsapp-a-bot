package economy

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	killCaughtChance   = 0.5
	deathHold          = 7 * time.Minute
	reaperKillBonus    = 0.40
	reaperHoldFactor   = 3
	reapLossMinPercent = 15
	reapLossMaxPercent = 25
)

var killCooldown = cooldown{column: "assassinato_em", every: 90 * time.Minute}

type killing struct {
	lawful  bool
	dead    time.Duration
	jailed  bool
	wait    time.Duration
	success bool
	vested  bool
	jail    time.Duration
	freed   types.JID
	fine    int64
	bounty  int64
	lost    int64
	hold    time.Duration
	reaped  bool
}

type killOdds struct {
	lawful  bool
	pierce  bool
	reap    bool
	success float64
	caught  float64
	jail    time.Duration
}

func (b *bot) matar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	mentions := core.MentionedJIDs(evt.Message)
	if len(mentions) == 0 {
		return b.SendText(ctx, chat, "Marca quem você vai apagar: !matar @pessoa")
	}
	killer, victim := evt.Info.Sender.ToNonAD(), mentions[0].ToNonAD()
	killerAcc, victimAcc := b.Account(ctx, killer), b.Account(ctx, victim)
	if killerAcc.Key == victimAcc.Key {
		return b.SendText(ctx, chat, "Se matar não 💛 Se a coisa tiver pesada de verdade, o CVV atende no 188, a qualquer hora")
	}

	killerJob, victimJob, owed, err := b.rivalsWithDebt(ctx, chat, killerAcc, victimAcc)
	if err != nil {
		return err
	}
	if killerJob.killChance == 0 && owed == nil {
		return b.SendText(ctx, chat, "🚫 Só quem é do crime pesado pode matar: "+killingProfessions()+". Ou quem anda armado (tem "+lethalWeapons()+" no !mercadonegro), quem tem a Foice da Comadre Morte, ou quem emprestou moedas com o !agiota pra alguém que passou do prazo")
	}
	odds := killOdds{
		success: killerJob.killOdds(victimJob),
		caught:  victimJob.killCaughtOdds(),
		jail:    Profession{jailTime: baseJail}.sentence(killerJob),
		pierce:  killerJob.divine || killerJob.reaper,
		reap:    killerJob.reaper,
	}
	if killerJob.slipsAway() {
		odds.caught = 0
	}
	if owed != nil {
		odds.success = max(odds.success, clampChance(debtKillChance+killerJob.collectionEdge(owed)-victimJob.guard))
		odds.caught, odds.lawful = 0, true
	}
	k, err := b.Wallets.kill(ctx, chat, killerAcc, victimAcc, odds, killerJob.pace(killCooldown), time.Now())
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
	_, err = b.SendMentions(ctx, chat, text, people)
	return err
}

func (p Profession) killOdds(victim Profession) float64 {
	if p.divine {
		return 1
	}
	return clampChance(p.killChance - victim.guard - p.curfew())
}

func (p Profession) killCaughtOdds() float64 {
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
	return core.JoinWithAnd(names)
}

func killText(k killing, killerJob, victimJob Profession, killer, victim types.JID) string {
	if k.dead > 0 {
		return fmt.Sprintf("⚰️ @%s já tá morto(a) e só ressuscita em %s. Deixa a pessoa descansar", victim.User, core.FormatWait(k.dead))
	}
	if k.jailed {
		return fmt.Sprintf("🔒 @%s tá na cadeia, nem dá pra alcançar", victim.User)
	}
	if k.wait > 0 {
		return fmt.Sprintf("🧽 @%s ainda tá sumindo com as provas do último. Espera %s pra tentar de novo", killer.User, core.FormatWait(k.wait))
	}
	if k.success {
		return deathCaption(k, killer, victim)
	}
	if k.vested {
		text := fmt.Sprintf("🦺 @%s acertou @%s em cheio, mas o colete à prova de balas segurou tudo. O colete ficou destruído, e a vítima saiu só com um roxo", killer.User, victim.User)
		if k.lawful {
			return text
		}
		return text + fmt.Sprintf("\n🚔 Tentativa de homicídio também é crime: a polícia pode !prender @%s nas próximas %s", killer.User, core.FormatWait(crimeWindow))
	}
	text := fmt.Sprintf("🩹 @%s tentou apagar @%s, mas errou feio e a vítima saiu só com um susto", killer.User, victim.User)
	if k.lawful {
		return text
	}
	if k.jail == 0 {
		return text + fmt.Sprintf("\n🚔 Tentativa de homicídio também é crime: a polícia pode !prender @%s nas próximas %s", killer.User, core.FormatWait(crimeWindow))
	}
	if victimJob.jailTime > 0 {
		return text + fmt.Sprintf("\n👮 Só que @%s %s, reagiu e deu voz de prisão na hora! @%s pegou %s de cadeia", victim.User, victimJob.title, killer.User, core.FormatWait(k.jail)) + jailFactorText(killer, killerJob) + fineText(k.fine)
	}
	return text + fmt.Sprintf("\n🚓 Alguém chamou a polícia e @%s foi em cana: %s de cadeia", killer.User, core.FormatWait(k.jail)) + jailFactorText(killer, killerJob) + fineText(k.fine)
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
		fmt.Sprintf(core.PickOne(scenes), killer.User, victim.User),
		"",
		fmt.Sprintf("⚰️ @%s tá morto(a) por %s: até ressuscitar, nada de comando de economia (nem !saldo, !trabalhar ou !daily). Um necromante, o presidente ou quem tem o Livro de São Cipriano pode tentar !reviver", victim.User, core.FormatWait(k.hold)),
	}
	lines = append(lines, reapText(k, killer, victim)...)
	if !k.freed.IsEmpty() {
		lines = append(lines, fmt.Sprintf("🔓 O refém de @%s aproveitou o velório pra fugir: @%s tá livre!", victim.User, k.freed.User))
	}
	if k.bounty > 0 {
		lines = append(lines, fmt.Sprintf("🎯 Tinha recompensa na cabeça de @%s: @%s embolsou %s", victim.User, killer.User, Coins(k.bounty)))
	}
	if k.lawful {
		return strings.Join(lines, "\n")
	}
	lines = append(lines, fmt.Sprintf("🚔 Homicídio é crime: a polícia pode !prender @%s nas próximas %s, e qualquer um pode !denunciar", killer.User, core.FormatWait(crimeWindow)))
	return strings.Join(lines, "\n")
}

func (b *bot) holdIfDead(ctx context.Context, evt *events.Message) bool {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	left, err := b.Wallets.DeathLeft(ctx, chat, b.Account(ctx, person), time.Now())
	if err != nil {
		b.Client.Log.Errorf("checar se tá morto: %v", err)
		return false
	}
	if left == 0 {
		return false
	}
	text := fmt.Sprintf("⚰️ @%s tá morto(a) e só ressuscita em %s. Até lá nada de comando de economia, a não ser que alguém consiga !reviver ou você pague o !hospital", person.User, core.FormatWait(left))
	if _, err := b.SendMentions(ctx, chat, text, []types.JID{person}); err != nil {
		b.Client.Log.Errorf("avisar que tá morto: %v", err)
	}
	return true
}

func (o killOdds) funeral(killer core.Account) funeral {
	if !o.reap {
		return plainFuneral
	}
	return funeral{hold: deathHold * reaperHoldFactor, minPercent: reapLossMinPercent, maxPercent: reapLossMaxPercent, heir: killer}
}

func reapText(k killing, killer, victim types.JID) []string {
	if !k.reaped {
		if funeral := funeralText(victim, k.lost); funeral != "" {
			return []string{funeral}
		}
		return nil
	}
	if k.lost == 0 {
		return []string{fmt.Sprintf("💀 A Comadre Morte veio buscar @%s pela foice de @%s, mas o caixão tava vazio: não tinha nada no saldo pra levar", victim.User, killer.User)}
	}
	return []string{fmt.Sprintf("💀 A Comadre Morte veio buscar @%s pela foice de @%s, e o enterro de %s foi direto pro bolso de quem matou", victim.User, killer.User, Coins(k.lost))}
}

func (s *walletStore) kill(ctx context.Context, chat types.JID, killer, victim core.Account, odds killOdds, cd cooldown, now time.Time) (k killing, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		for _, acc := range []core.Account{killer, victim} {
			if err := openWallet(ctx, tx, chat, acc); err != nil {
				return err
			}
		}
		var jailUntil, deadUntil int64
		err := tx.QueryRowContext(ctx,
			"SELECT preso_ate, morto_ate FROM carteiras WHERE chat = ? AND pessoa = ?",
			chat.String(), victim.Key,
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
		k.success = rand.Float64() < odds.success
		if k.success && !odds.pierce {
			if k.vested, err = spendItem(ctx, tx, chat, victim, vestItemID); err != nil {
				return err
			}
			k.success = !k.vested
		}
		if k.vested {
			return nil
		}
		if k.success {
			d, err := bury(ctx, tx, chat, victim, odds.funeral(killer), now)
			if err != nil {
				return err
			}
			k.freed, k.lost, k.hold, k.reaped = d.freed, d.lost, d.hold, odds.reap
			k.bounty, err = claimBounty(ctx, tx, chat, victim, killer)
			return err
		}
		if rand.Float64() >= odds.caught {
			return nil
		}
		k.jail = odds.jail
		_, err = tx.ExecContext(ctx,
			"UPDATE carteiras SET preso_ate = ?, crime_em = 0 WHERE chat = ? AND pessoa = ?",
			now.Add(k.jail).Unix(), chat.String(), killer.Key,
		)
		if err != nil {
			return err
		}
		k.fine, err = lockUp(ctx, tx, chat, killer, now)
		return err
	})
	return k, err
}
