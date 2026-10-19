package economy

import (
	"cmp"
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	bossWindow   = 90 * time.Minute
	bossRest     = 10 * time.Minute
	bossHitMin   = 60
	bossHitMax   = 140
	bossArmed    = 150
	bossCrit     = 0.1
	bossShareMin = 50
	bossShareMax = 250
	bossBar      = 10
)

type bossScenario struct {
	id        string
	title     string
	rarity    eventRarity
	toughness int64
	intros    []string
	hits      []string
	counter   ending
	victory   ending
	trophy    []effect
	defeat    ending
}

type blow struct {
	damage int64
	crit   bool
}

type bossHit struct {
	wait   time.Duration
	gone   bool
	felled bool
	dealt  int64
	health int64
	life   int64
}

func bossByID(id string) (bossScenario, bool) {
	i := slices.IndexFunc(bossScenarios, func(sc bossScenario) bool { return sc.id == id })
	if i < 0 {
		return bossScenario{}, false
	}
	return bossScenarios[i], true
}

func setupBoss(ev *liveEvent, crowd int) {
	sc, ok := bossByID(ev.scenario)
	if !ok {
		return
	}
	ev.life = sc.toughness * int64(max(crowd, liveCrowd))
	ev.health = ev.life
}

func lifeBar(health, life int64) string {
	full := int((health*bossBar + life - 1) / life)
	return strings.Repeat("🟥", full) + strings.Repeat("⬛", bossBar-full) + fmt.Sprintf(" %d/%d", health, life)
}

func strikeOf(job Profession) blow {
	damage := between(bossHitMin, bossHitMax)
	if job.weapon != "" {
		damage = damage * bossArmed / 100
	}
	if rand.Float64() >= bossCrit {
		return blow{damage: damage}
	}
	return blow{damage: damage * 2, crit: true}
}

func shareOf(damage, total int64, hitters int) int64 {
	return min(max(damage*int64(hitters)*100/total, bossShareMin), bossShareMax)
}

func medal(rank int) string {
	medals := []string{"🥇", "🥈", "🥉"}
	if rank < len(medals) {
		return medals[rank]
	}
	return "▫️"
}

func (ev *liveEvent) strike(acc core.Account, damage int64, now time.Time) bossHit {
	ev.mu.Lock()
	defer ev.mu.Unlock()
	if ev.closed {
		return bossHit{gone: true}
	}
	h, ok := ev.hitters[acc.Key]
	if ok && now.Sub(h.last) < bossRest {
		return bossHit{wait: bossRest - now.Sub(h.last)}
	}
	if !ok {
		h = &liveHitter{acc: acc}
		ev.hitters[acc.Key] = h
	}
	dealt := min(damage, ev.health)
	h.damage += dealt
	h.hits++
	h.last = now
	ev.health -= dealt
	hit := bossHit{dealt: dealt, health: ev.health, life: ev.life}
	hit.felled = ev.health == 0 && ev.shut()
	return hit
}

func (ev *liveEvent) ranking() ([]liveHitter, int64) {
	ev.mu.Lock()
	defer ev.mu.Unlock()
	hitters := make([]liveHitter, 0, len(ev.hitters))
	for _, h := range ev.hitters {
		hitters = append(hitters, *h)
	}
	slices.SortFunc(hitters, func(a, b liveHitter) int { return cmp.Compare(b.damage, a.damage) })
	return hitters, ev.health
}

func (b *bot) showBoss(ctx context.Context, ev *liveEvent, players []types.JID) error {
	sc, ok := bossByID(ev.scenario)
	if !ok {
		return fmt.Errorf("chefão %s não existe", ev.scenario)
	}
	rules := fmt.Sprintf("❤️ %s\n⚔️ Todo mundo pode *!atacar*, uma vez a cada %s. Se cair em até %s, quem bateu divide o prêmio pelo dano que deu. Se escapar, quem bateu paga o pato",
		lifeBar(ev.health, ev.life), core.FormatWait(bossRest), core.FormatWait(bossWindow))
	text := liveHeader("👹", sc.title, sc.rarity) + "\n\n" + core.PickOne(sc.intros) + "\n\n" + rules + liveCall(players)
	_, err := b.SendMentions(ctx, ev.chat, text, players)
	return err
}

func (b *bot) atacar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	ev, ok := b.liveIn(chat, bossFormat)
	if !ok {
		return b.ReplyTo(ctx, evt, "🕊️ não tem nenhum chefão no grupo agora. Quando aparecer um num evento ao vivo, todo mundo pode !atacar")
	}
	sc, ok := bossByID(ev.scenario)
	if !ok {
		return nil
	}
	acc := b.Account(ctx, evt.Info.Sender)
	job, err := b.ProfessionOf(ctx, chat, acc)
	if err != nil {
		return err
	}
	hit := ev.strike(acc, strikeOf(job).damage, time.Now())
	if hit.gone {
		return b.ReplyTo(ctx, evt, fmt.Sprintf("🏃 *%s* já foi embora", sc.title))
	}
	if hit.wait > 0 {
		return b.ReplyTo(ctx, evt, "😮‍💨 você ainda tá sem fôlego, dá pra !atacar de novo em "+core.FormatWait(hit.wait))
	}
	if hit.felled {
		b.releaseLive(ctx, ev)
		return b.fellBoss(ctx, ev, sc, acc)
	}
	b.keepLive(ctx, ev)
	return b.answerHit(ctx, ev, sc, acc, hit)
}

func (b *bot) answerHit(ctx context.Context, ev *liveEvent, sc bossScenario, acc core.Account, hit bossHit) error {
	who := "@" + acc.JID.User
	lines := []string{strings.ReplaceAll(core.PickOne(sc.hits), "{@}", who), fmt.Sprintf("⚔️ %d de dano", hit.dealt), "❤️ " + lifeBar(hit.health, hit.life)}
	mentions := []types.JID{acc.JID}
	if rand.IntN(100) < sc.counter.weight {
		results, err := b.playLive(ctx, ev.chat, []liveTurn{{acc: acc, weight: 100, effects: sc.counter.effects}})
		if err != nil {
			return fmt.Errorf("contra-ataque do chefão %s: %w", sc.id, err)
		}
		lines = append(lines, "", strings.ReplaceAll(core.PickOne(sc.counter.texts), "{@}", who))
		lines = append(lines, results[0].lines...)
		mentions = append(mentions, results[0].freed...)
	}
	_, err := b.SendMentions(ctx, ev.chat, strings.Join(lines, "\n"), mentions)
	return err
}

func (b *bot) fellBoss(ctx context.Context, ev *liveEvent, sc bossScenario, finisher core.Account) error {
	hitters, _ := ev.ranking()
	var total int64
	for _, h := range hitters {
		total += h.damage
	}
	turns := make([]liveTurn, len(hitters))
	for i, h := range hitters {
		effects := sc.victory.effects
		if i == 0 {
			effects = slices.Concat(effects, sc.trophy)
		}
		turns[i] = liveTurn{acc: h.acc, weight: shareOf(h.damage, total, len(hitters)), effects: effects}
	}
	results, err := b.playLive(ctx, ev.chat, turns)
	if err != nil {
		return fmt.Errorf("pagar o chefão %s: %w", sc.id, err)
	}
	head := fmt.Sprintf("🏆 *%s* caiu! O golpe final foi de @%s\n\n%s", sc.title, finisher.JID.User, strings.ReplaceAll(core.PickOne(sc.victory.texts), "{@}", "@"+hitters[0].acc.JID.User))
	return b.reportBoss(ctx, ev, head, hitters, results)
}

func (b *bot) escapeBoss(ctx context.Context, ev *liveEvent) error {
	sc, ok := bossByID(ev.scenario)
	if !ok {
		return nil
	}
	hitters, health := ev.ranking()
	head := fmt.Sprintf("🏃 *%s* escapou com %d de vida sobrando!", sc.title, health)
	if len(hitters) == 0 {
		return b.SendText(ctx, ev.chat, head+"\n\nNinguém teve coragem de encarar, e foi embora rindo da cara do grupo")
	}
	turns := make([]liveTurn, len(hitters))
	for i, h := range hitters {
		turns[i] = liveTurn{acc: h.acc, weight: 100, effects: sc.defeat.effects}
	}
	results, err := b.playLive(ctx, ev.chat, turns)
	if err != nil {
		return fmt.Errorf("cobrar a fuga do chefão %s: %w", sc.id, err)
	}
	return b.reportBoss(ctx, ev, head+"\n\n"+core.PickOne(sc.defeat.texts), hitters, results)
}

func (b *bot) reportBoss(ctx context.Context, ev *liveEvent, head string, hitters []liveHitter, results []eventResult) error {
	blocks := []string{head}
	mentions := liveFreed(results)
	for i, h := range hitters {
		blocks = append(blocks, liveLine(fmt.Sprintf("%s @%s (%d de dano)", medal(i), h.acc.JID.User, h.damage), results[i]))
		mentions = append(mentions, h.acc.JID)
	}
	_, err := b.SendMentions(ctx, ev.chat, strings.Join(blocks, "\n\n"), mentions)
	return err
}
