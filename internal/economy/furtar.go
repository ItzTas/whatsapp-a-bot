package economy

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	theftFinePercent    = 50
	theftMinFine        = 20
	targetedTheftFactor = 0.6
)

var theftCooldown = cooldown{column: "furto_em", every: 30 * time.Minute}

func (b *bot) furtar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	mentions := core.MentionedJIDs(evt.Message)
	if len(mentions) == 0 {
		return b.SendText(ctx, chat, "Marca de quem você vai furtar: !furtar @pessoa (ou !furtar @pessoa todynho pra tentar um item específico)")
	}
	thief, victim := evt.Info.Sender.ToNonAD(), mentions[0].ToNonAD()
	thiefAcc, victimAcc := b.Account(ctx, thief), b.Account(ctx, victim)
	if thiefAcc.Key == victimAcc.Key {
		return b.SendText(ctx, chat, "Furtar a própria mochila? Isso se chama pegar as suas coisas 🤡")
	}

	var wanted item
	if name := parseItemQuery(core.MessageText(evt.Message)).name; name != "" {
		matches := findItems(name)
		if problem := itemQueryProblem(name, matches); problem != "" {
			return b.SendText(ctx, chat, problem)
		}
		wanted = matches[0]
	}
	if wanted.rarity == relic {
		return b.SendText(ctx, chat, fmt.Sprintf("🏛️ %s é relíquia, e relíquia não se furta. Escolhe outro item da mochila", wanted.label()))
	}
	thiefJob, victimJob, err := b.rivalProfessions(ctx, chat, thiefAcc, victimAcc)
	if err != nil {
		return err
	}
	t, err := b.Wallets.stealItem(ctx, chat, thiefAcc, victimAcc, thiefJob.theftCooldown(), time.Now(), theftChoice(wanted, thiefJob.theftBonus-victimJob.guard-thiefJob.curfew()))
	if err != nil {
		return fmt.Errorf("furtar item: %w", err)
	}
	_, err = b.SendMentions(ctx, chat, theftText(t, wanted, thief, victim), []types.JID{thief, victim})
	return err
}

func theftChoice(wanted item, bonus float64) func(loot []stack) (item, bool) {
	if wanted.id != "" {
		return func(loot []stack) (item, bool) {
			if quantityOf(loot, wanted.id) == 0 {
				return item{}, false
			}
			return wanted, rand.Float64() < clampChance(rarities[wanted.rarity].steal*targetedTheftFactor+bonus)
		}
	}
	return func(loot []stack) (item, bool) {
		loot = slices.DeleteFunc(slices.Clone(loot), func(s stack) bool { return s.item.rarity == relic })
		if len(loot) == 0 {
			return item{}, false
		}
		target := loot[weightedIndex(len(loot), func(i int) int { return int(loot[i].quantity) })].item
		return target, rand.Float64() < clampChance(rarities[target.rarity].steal+bonus)
	}
}

func theftText(t theft, wanted item, thief, victim types.JID) string {
	if t.empty && wanted.id != "" {
		return fmt.Sprintf("🎒 @%s não tem %s na mochila pra furtar", victim.User, wanted.label())
	}
	if t.empty {
		return fmt.Sprintf("🎒 A mochila de @%s não tem nada que dê pra furtar (relíquia não conta)", victim.User)
	}
	if t.wait > 0 {
		return fmt.Sprintf("🚓 @%s ainda tá sendo procurado pelo último furto. Espera %s pra tentar de novo", thief.User, core.FormatWait(t.wait))
	}
	if t.locked {
		return fmt.Sprintf("🔒 @%s tentou abrir a mochila de @%s, mas tinha um cadeado! Ele quebrou segurando o ladrão e não saiu nada", thief.User, victim.User)
	}
	if t.success {
		return fmt.Sprintf("🥷 @%s furtou %s%s da mochila de @%s e sumiu na multidão!\n%s", thief.User, t.item.label(), t.grade.tag(), victim.User, t.grade.summary(t.item))
	}
	if t.fine == 0 {
		return fmt.Sprintf("🚨 @%s foi pego tentando furtar %s de @%s, mas tava tão liso que saiu só com a vergonha", thief.User, t.item.label(), victim.User)
	}
	return fmt.Sprintf("🚨 @%s foi pego tentando furtar %s de @%s e pagou %s de indenização pra vítima", thief.User, t.item.label(), victim.User, Coins(t.fine))
}
