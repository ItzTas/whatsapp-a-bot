package economy

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	throwHitChance    = 0.6
	throwRetortChance = 0.35
)

var throwCooldown = cooldown{column: "arremesso_em", every: 10 * time.Minute}

type projectile struct {
	item     string
	source   string
	harm     int64
	hit      string
	miss     string
	shrug    string
	wounds   []string
	paybacks []string
}

var (
	stoneThrow = projectile{item: "pedra", source: "!minerar", harm: 20,
		hit:   "🪨 @%s tacou uma pedra em @%s e acertou em cheio!",
		miss:  "🪨 @%s tacou uma pedra em @%s, mas errou feio e acertou o carro do vizinho",
		shrug: "🪨 @%s acertou uma pedrada em @%s, mas a pedra quicou e nem fez cosquinha",
		wounds: []string{
			"levou uma pedrada bem no meio da testa", "tomou uma pedrada no joelho e saiu mancando",
			"levou uma pedrada nas costas e ficou sem ar", "tomou uma pedrada na canela e viu estrelas",
		},
		paybacks: []string{
			"levou a própria pedra de volta na cabeça", "tomou o troco da pedrada bem no nariz",
			"levou a pedra de volta no meio da barriga",
		},
	}
	tomatoThrow = projectile{item: "tomate", source: "!colher (a semente sai no !catar ou na !loja)", harm: 6,
		hit:   "🍅 @%s tacou um tomate em @%s e espirrou molho pra todo lado!",
		miss:  "🍅 @%s tacou um tomate em @%s, mas errou e sujou a parede inteira",
		shrug: "🍅 @%s acertou um tomate em @%s, mas ele tava tão mole que só fez um barulhinho",
		wounds: []string{
			"levou uma tomatada no olho e ficou ardendo", "tomou um tomate na orelha e ficou cheirando a molho",
			"levou um tomate podre na boca e quase vomitou",
		},
		paybacks: []string{
			"levou o tomate de volta bem na cara", "tomou o resto do tomate na nuca",
		},
	}
	projectiles = []projectile{tomatoThrow, stoneThrow}
)

type throwOdds struct {
	hit     float64
	wound   hazard
	payback hazard
}

type throwing struct {
	noItem   bool
	wait     time.Duration
	struck   bool
	retorted bool
	hit      injury
	payback  injury
}

func (b *bot) tacar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	p := aimedProjectile(core.MessageText(evt.Message))
	mentions := core.MentionedJIDs(evt.Message)
	if len(mentions) == 0 {
		return b.SendText(ctx, chat, fmt.Sprintf("Marca em quem você vai tacar: !tacar @pessoa %s", p.item))
	}
	thrower, victim := evt.Info.Sender.ToNonAD(), mentions[0].ToNonAD()
	throwerAcc, victimAcc := b.Account(ctx, thrower), b.Account(ctx, victim)
	if throwerAcc.Key == victimAcc.Key {
		return b.SendText(ctx, chat, "Tacar pra cima e ficar embaixo esperando? Melhor não 🤡")
	}
	throwerJob, victimJob, err := b.rivalProfessions(ctx, chat, throwerAcc, victimAcc)
	if err != nil {
		return err
	}
	t, err := b.Wallets.throw(ctx, chat, throwerAcc, victimAcc, p.item, throwerJob.pace(throwCooldown), p.odds(throwerJob, victimJob), time.Now())
	if err != nil {
		return fmt.Errorf("tacar %s: %w", p.item, err)
	}
	_, err = b.SendMentions(ctx, chat, p.text(t, thrower, victim), []types.JID{thrower, victim})
	return err
}

func aimedProjectile(text string) projectile {
	folded := core.FoldLetters(text)
	for _, p := range projectiles {
		if strings.Contains(folded, p.item) {
			return p
		}
	}
	return stoneThrow
}

func (p projectile) odds(thrower, victim Profession) throwOdds {
	o := throwOdds{
		hit:     clampChance(thrower.sober(throwHitChance - victim.guard)),
		wound:   hazard{chance: 1, harm: p.harm, wounds: p.wounds},
		payback: hazard{chance: 1, harm: p.harm, wounds: p.paybacks},
	}
	if victim.divine {
		o.wound = hazard{}
	}
	if thrower.divine {
		o.payback = hazard{}
	}
	return o
}

func (s *walletStore) throw(ctx context.Context, chat types.JID, thrower, victim core.Account, id string, cd cooldown, o throwOdds, now time.Time) (t throwing, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		wait, err := stampCooldown(ctx, tx, chat, thrower, cd, now)
		if err != nil || wait > 0 {
			t.wait = wait
			return err
		}
		if err := takeItems(ctx, tx, chat, thrower, id, 1); err != nil {
			return err
		}
		if err := openWallet(ctx, tx, chat, victim); err != nil {
			return err
		}
		if t.struck = rand.Float64() < o.hit; t.struck {
			if t.hit, err = bruise(ctx, tx, chat, victim, o.wound.roll(0), now); err != nil {
				return err
			}
		}
		if t.retorted = rand.Float64() < throwRetortChance; !t.retorted {
			return nil
		}
		t.payback, err = bruise(ctx, tx, chat, thrower, o.payback.roll(0), now)
		return err
	})
	if errors.Is(err, errNoItems) {
		return throwing{noItem: true}, nil
	}
	return t, err
}

func (p projectile) text(t throwing, thrower, victim types.JID) string {
	if t.wait > 0 {
		return fmt.Sprintf("💪 @%s ainda tá com o braço doendo do último arremesso. Espera %s pra tacar de novo", thrower.User, core.FormatWait(t.wait))
	}
	if t.noItem {
		return fmt.Sprintf("🎒 @%s não tem %s na mochila pra tacar. Dá pra conseguir no %s", thrower.User, p.label(), p.source)
	}
	return p.hitText(t, thrower, victim) + p.retortText(t, thrower, victim)
}

func (p projectile) hitText(t throwing, thrower, victim types.JID) string {
	if !t.struck {
		return fmt.Sprintf(p.miss, thrower.User, victim.User)
	}
	if t.hit.damage == 0 {
		return fmt.Sprintf(p.shrug, thrower.User, victim.User)
	}
	hurt, _ := t.hit.text(victim)
	return fmt.Sprintf(p.hit, thrower.User, victim.User) + hurt
}

func (p projectile) retortText(t throwing, thrower, victim types.JID) string {
	if !t.retorted {
		return ""
	}
	if t.payback.damage == 0 {
		return fmt.Sprintf("\n\n😤 @%s pegou %s do chão e tacou de volta, mas @%s desviou", victim.User, p.label(), thrower.User)
	}
	hurt, _ := t.payback.text(thrower)
	return fmt.Sprintf("\n\n😤 @%s pegou %s do chão e revidou!", victim.User, p.label()) + hurt
}

func (p projectile) label() string {
	it, _ := itemByID(p.item)
	return it.label()
}
