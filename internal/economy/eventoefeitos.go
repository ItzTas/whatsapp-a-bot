package economy

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

var errEventVoid = errors.New("evento sem efeito")

type effect interface {
	apply(ctx context.Context, tx *ledgerTx, run *eventRun) (string, error)
	fortune() int
}

type eventRun struct {
	chat   types.JID
	acc    core.Account
	victim core.Account
	job    Profession
	place  string
	stake  int64
	now    time.Time
	dead   bool
	freed  []types.JID
}

type eventResult struct {
	lines   []string
	freed   []types.JID
	missing item
	price   int64
	gone    bool
	broke   bool
	died    bool
}

func (r eventResult) played() bool {
	return !r.gone && !r.broke && !r.died && r.missing.id == ""
}

func (r eventResult) void(person types.JID) string {
	notes := []struct {
		on   bool
		text string
	}{
		{r.gone, fmt.Sprintf("⚰️ Só que @%s morreu antes de decidir, e o evento acabou ali", person.User)},
		{r.missing.id != "", fmt.Sprintf("🎒 Só que @%s não tinha mais %s na mochila, e o evento passou batido", person.User, r.missing.label())},
		{r.broke, fmt.Sprintf("🪙 Só que na hora de pagar @%s não tinha %s, e o evento passou batido", person.User, Coins(r.price))},
	}
	for _, n := range notes {
		if n.on {
			return n.text
		}
	}
	return ""
}

func (s *walletStore) playEvent(ctx context.Context, run eventRun, play eventPlay) (res eventResult, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		if err := openWallet(ctx, tx, run.chat, run.acc); err != nil {
			return err
		}
		var dead bool
		err := tx.QueryRowContext(ctx, "SELECT morto_ate > ? FROM carteiras WHERE chat = ? AND pessoa = ?", run.now.Unix(), run.chat.String(), run.acc.Key).Scan(&dead)
		if err != nil {
			return err
		}
		if dead {
			res.gone = true
			return errEventVoid
		}
		if err := play.pay(ctx, tx, run, &res); err != nil {
			return err
		}
		for _, e := range play.ending.effects {
			line, err := e.apply(ctx, tx, &run)
			if err != nil {
				return err
			}
			if line != "" {
				res.lines = append(res.lines, line)
			}
			if run.dead {
				break
			}
		}
		res.died, res.freed = run.dead, run.freed
		return nil
	})
	if errors.Is(err, errEventVoid) {
		return res, nil
	}
	return res, err
}

func (p eventPlay) pay(ctx context.Context, tx *ledgerTx, run eventRun, res *eventResult) error {
	if p.choice == nil {
		return nil
	}
	if it, ok := itemByID(p.choice.uses); ok {
		err := takeItems(ctx, tx, run.chat, run.acc, it.id, 1)
		if errors.Is(err, errNoItems) {
			res.missing = it
			return errEventVoid
		}
		if err != nil {
			return err
		}
		res.lines = append(res.lines, "🎒 Gastou "+it.label())
	}
	if p.price == 0 {
		return nil
	}
	err := debitBalance(ctx, tx, run.chat, run.acc, p.price)
	if errors.Is(err, ErrNoFunds) {
		res.broke, res.price = true, p.price
		return errEventVoid
	}
	if err != nil {
		return err
	}
	res.lines = append(res.lines, "💸 Pagou "+Coins(p.price))
	return nil
}

func between(lo, hi int64) int64 {
	if hi <= lo {
		return lo
	}
	return lo + rand.Int64N(hi-lo+1)
}

func countedItem(it item, n int64) string {
	if n == 1 {
		return it.label()
	}
	return fmt.Sprintf("%d %s", n, it.label())
}

func (run *eventRun) mourn(d death) string {
	run.dead = true
	lines := []string{fmt.Sprintf("💀 @%s morreu! Fica %s sem comando de economia, a não ser que pague o !hospital ou alguém consiga !reviver", run.acc.JID.User, core.FormatWait(d.hold))}
	if funeral := funeralText(run.acc.JID, d.lost); funeral != "" {
		lines = append(lines, funeral)
	}
	escape, freed := hostageEscape(d.freed, "🔓 O refém de @"+run.acc.JID.User+" aproveitou o velório pra fugir: @%s tá livre!")
	if escape != "" {
		lines = append(lines, escape)
		run.freed = append(run.freed, freed...)
	}
	return strings.Join(lines, "\n")
}

type earnEffect struct{ lo, hi int64 }

func earn(lo, hi int64) effect { return earnEffect{lo, hi} }

func (e earnEffect) apply(ctx context.Context, tx *ledgerTx, run *eventRun) (string, error) {
	amount := run.stake * between(e.lo, e.hi) / 100
	if amount <= 0 {
		return "", nil
	}
	_, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", amount, run.chat.String(), run.acc.Key)
	return "💰 Ganhou " + Coins(amount), err
}

type loseEffect struct{ lo, hi int64 }

func lose(lo, hi int64) effect { return loseEffect{lo, hi} }

func (e loseEffect) apply(ctx context.Context, tx *ledgerTx, run *eventRun) (string, error) {
	lost, err := burnBalance(ctx, tx, run.chat, run.acc, e.lo, e.hi, 1)
	if err != nil || lost == 0 {
		return "", err
	}
	return "💸 Perdeu " + Coins(lost), nil
}

type gainEffect struct {
	n   int64
	ids []string
}

func gain(n int64, ids ...string) effect { return gainEffect{n, ids} }

func (e gainEffect) apply(ctx context.Context, tx *ledgerTx, run *eventRun) (string, error) {
	it, ok := itemByID(core.PickOne(e.ids))
	if !ok || e.n <= 0 {
		return "", nil
	}
	if err := addItems(ctx, tx, run.chat, run.acc, it.id, e.n); err != nil {
		return "", err
	}
	return "🎁 Ganhou " + countedItem(it, e.n), nil
}

type forfeitEffect struct {
	id string
	n  int64
}

func forfeit(id string, n int64) effect { return forfeitEffect{id, n} }

func (e forfeitEffect) apply(ctx context.Context, tx *ledgerTx, run *eventRun) (string, error) {
	it, ok := itemByID(e.id)
	if !ok {
		return "", nil
	}
	held, err := gradesIn(ctx, tx, backpackTable, run.chat, run.acc, it.id)
	if err != nil {
		return "", err
	}
	var have int64
	for _, g := range held {
		have += g.quantity
	}
	n := min(e.n, have)
	if n <= 0 {
		return "", nil
	}
	if err := takeItems(ctx, tx, run.chat, run.acc, it.id, n); err != nil {
		return "", err
	}
	return "📦 Perdeu " + countedItem(it, n), nil
}

type lootEffect struct{}

func findLoot() effect { return lootEffect{} }

func (lootEffect) apply(ctx context.Context, tx *ledgerTx, run *eventRun) (string, error) {
	it, ok := run.findable()
	if !ok {
		return "", nil
	}
	if err := addItems(ctx, tx, run.chat, run.acc, it.id, 1); err != nil {
		return "", err
	}
	return "🎁 Achou " + it.label(), nil
}

func (run *eventRun) findable() (item, bool) {
	if i := slices.IndexFunc(activities, func(a activity) bool { return a.command == run.place }); i >= 0 {
		loot := slices.DeleteFunc(slices.Clone(activities[i].loot), func(d drop) bool {
			it, ok := itemByID(d.item)
			return !ok || it.rarity >= mythic || it.relicOdds > 0
		})
		if len(loot) == 0 {
			return item{}, false
		}
		return itemByID(loot[weightedIndex(len(loot), func(i int) int { return loot[i].weight })].item)
	}
	finds, _ := run.job.loot()
	if len(finds) == 0 {
		return item{}, false
	}
	return itemByID(core.PickOne(finds))
}

type injureEffect struct{ lo, hi int64 }

func injure(lo, hi int64) effect { return injureEffect{lo, hi} }

func (e injureEffect) apply(ctx context.Context, tx *ledgerTx, run *eventRun) (string, error) {
	inj, err := hurt(ctx, tx, run.chat, run.acc, wound{damage: between(e.lo, e.hi)}, run.now)
	if err != nil || inj.damage == 0 {
		return "", err
	}
	if inj.died {
		return run.mourn(inj.death), nil
	}
	return fmt.Sprintf("🤕 Perdeu %d de saúde (%s)", inj.damage, healthText(inj.health)), nil
}

type healEffect struct{ amount int64 }

func heal(amount int64) effect { return healEffect{amount} }

func (e healEffect) apply(ctx context.Context, tx *ledgerTx, run *eventRun) (string, error) {
	current, err := readHealth(ctx, tx, run.chat, run.acc, run.now)
	if err != nil || current >= MaxHealth {
		return "", err
	}
	health := min(current+e.amount, MaxHealth)
	if err := setHealth(ctx, tx, run.chat, run.acc, health, run.now); err != nil {
		return "", err
	}
	return fmt.Sprintf("❤️‍🩹 Recuperou %d de saúde (%s)", health-current, healthText(health)), nil
}

type buzzEffect struct{ id string }

func buzz(id string) effect { return buzzEffect{id} }

func (e buzzEffect) apply(ctx context.Context, tx *ledgerTx, run *eventRun) (string, error) {
	it, ok := itemByID(e.id)
	if !ok || it.high == nil {
		return "", nil
	}
	d, err := getHigh(ctx, tx, run.chat, run.acc, it, run.now)
	if err != nil {
		return "", err
	}
	if d.overdosed {
		return fmt.Sprintf("🤢 Teve overdose de %s!\n", it.label()) + run.mourn(d.death), nil
	}
	text := fmt.Sprintf("%s Ficou sob efeito de %s por %s: %s", it.emoji, it.label(), core.FormatWait(it.high.lasts), core.JoinWithAnd(it.high.effects()))
	if d.mixed.id != "" {
		text += fmt.Sprintf(". Misturou com %s e por pouco não deu ruim", d.mixed.label())
	}
	return text, nil
}

type nabEffect struct{}

func nabbed() effect { return nabEffect{} }

func (nabEffect) apply(ctx context.Context, tx *ledgerTx, run *eventRun) (string, error) {
	if rand.Float64() >= run.job.flagrantOdds(1) {
		return fmt.Sprintf("🎩 A polícia chegou, mas @%s escapou do flagrante", run.acc.JID.User), nil
	}
	jail := Profession{jailTime: baseJail}.sentence(run.job)
	_, err := tx.ExecContext(ctx,
		"UPDATE carteiras SET preso_ate = MAX(preso_ate, ?), crime_em = 0 WHERE chat = ? AND pessoa = ?",
		run.now.Add(jail).Unix(), run.chat.String(), run.acc.Key,
	)
	if err != nil {
		return "", err
	}
	fine, err := lockUp(ctx, tx, run.chat, run.acc, run.now)
	if err != nil {
		return "", err
	}
	freed, err := freeHostage(ctx, tx, run.chat, run.acc, run.now)
	if err != nil {
		return "", err
	}
	escape, people := hostageEscape(freed, "\n🔓 Com o sequestrador preso, @%s saiu do cativeiro!")
	run.freed = append(run.freed, people...)
	return fmt.Sprintf("🚔 Foi preso na hora: fica %s na cadeia, sem apostar, roubar nem furtar", core.FormatWait(jail)) + fineText(fine) + escape, nil
}

type perishEffect struct{}

func perish() effect { return perishEffect{} }

func (perishEffect) apply(ctx context.Context, tx *ledgerTx, run *eventRun) (string, error) {
	d, err := die(ctx, tx, run.chat, run.acc, run.now)
	if err != nil {
		return "", err
	}
	return run.mourn(d), nil
}

type stainEffect struct{}

func stain() effect { return stainEffect{} }

func (stainEffect) apply(ctx context.Context, tx *ledgerTx, run *eventRun) (string, error) {
	if err := markCrime(ctx, tx, run.chat, run.acc, run.now); err != nil {
		return "", err
	}
	return "🚨 Ficou com a ficha suja: a polícia pode te !prender nas próximas " + core.FormatWait(crimeWindow), nil
}

type spellEffect struct{ id string }

func enchant(id string) effect { return spellEffect{id} }

func (e spellEffect) apply(ctx context.Context, tx *ledgerTx, run *eventRun) (string, error) {
	c, ok := charmByID(e.id)
	if !ok {
		return "", nil
	}
	if err := grantCharm(ctx, tx, run.chat, run.acc, run.acc, c, run.now); err != nil {
		return "", err
	}
	if c.curse {
		return fmt.Sprintf("%s Pegou a maldição %s por %s: %s", c.emoji, c.label(), core.FormatWait(c.hold), c.effect), nil
	}
	return fmt.Sprintf("%s Ganhou a bênção %s por %s: %s", c.emoji, c.label(), core.FormatWait(c.hold), c.effect), nil
}

func placeCooldown(place string) (column, command string) {
	if i := slices.IndexFunc(activities, func(a activity) bool { return a.command == place }); i >= 0 {
		return activities[i].cooldown.column, place
	}
	columns := map[string]string{
		"!roubar":     "roubo_em",
		"!quadrilha":  "roubo_em",
		"!furtar":     theftCooldown.column,
		"!sequestrar": "sequestro_em",
		"!matar":      "assassinato_em",
		"!assalto":    "assalto_em",
	}
	if column, ok := columns[place]; ok {
		return column, place
	}
	return "trabalho_em", "!trabalhar"
}

type refreshEffect struct{}

func refresh() effect { return refreshEffect{} }

func (refreshEffect) apply(ctx context.Context, tx *ledgerTx, run *eventRun) (string, error) {
	column, command := placeCooldown(run.place)
	_, err := tx.ExecContext(ctx, "UPDATE carteiras SET "+column+" = 0 WHERE chat = ? AND pessoa = ?", run.chat.String(), run.acc.Key)
	return fmt.Sprintf("⏩ O %s já tá liberado de novo", command), err
}

type delayEffect struct{ by time.Duration }

func delay(by time.Duration) effect { return delayEffect{by} }

func (e delayEffect) apply(ctx context.Context, tx *ledgerTx, run *eventRun) (string, error) {
	column, command := placeCooldown(run.place)
	_, err := tx.ExecContext(ctx, "UPDATE carteiras SET "+column+" = "+column+" + ? WHERE chat = ? AND pessoa = ?", int64(e.by.Seconds()), run.chat.String(), run.acc.Key)
	return fmt.Sprintf("🐌 O %s vai demorar mais %s pra liberar", command, core.FormatWait(e.by)), err
}

type breakEffect struct{}

func breakTool() effect { return breakEffect{} }

func (breakEffect) apply(ctx context.Context, tx *ledgerTx, run *eventRun) (string, error) {
	bag, err := loadInventory(ctx, tx, run.chat, run.acc)
	if err != nil {
		return "", err
	}
	tool, ok := run.tool(bag)
	if !ok {
		return "", nil
	}
	if err := takeItems(ctx, tx, run.chat, run.acc, tool.id, 1); err != nil {
		return "", err
	}
	return "💥 " + tool.label() + " quebrou", nil
}

func (run *eventRun) tool(bag []stack) (item, bool) {
	if i := slices.IndexFunc(activities, func(a activity) bool { return a.command == run.place }); i >= 0 {
		best, ok := activities[i].bestTool(bag)
		return best, ok && best.gear.sturdy < 100
	}
	trade, ok := tradeTool(run.job.area)
	if run.place != "!trabalhar" || !ok {
		return item{}, false
	}
	return trade, slices.ContainsFunc(bag, func(s stack) bool { return s.item.id == trade.id })
}

type plantEffect struct {
	id    string
	after time.Duration
}

func plant(id string, after time.Duration) effect { return plantEffect{id, after} }

func (e plantEffect) apply(ctx context.Context, tx *ledgerTx, run *eventRun) (string, error) {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO eventos_marcados (chat, pessoa, evento, a_partir) VALUES (?, ?, ?, ?)
		 ON CONFLICT (chat, pessoa, evento) DO UPDATE SET a_partir = excluded.a_partir`,
		run.chat.String(), run.acc.Key, e.id, run.now.Add(e.after).Unix(),
	)
	return "🔮 _Isso ainda vai dar o que falar..._", err
}

type refundEffect struct{ lo, hi int64 }

func refund(lo, hi int64) effect { return refundEffect{lo, hi} }

func (e refundEffect) apply(ctx context.Context, tx *ledgerTx, run *eventRun) (string, error) {
	if run.victim.Key == "" {
		return "", nil
	}
	var balance int64
	err := tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", run.chat.String(), run.acc.Key).Scan(&balance)
	if err != nil {
		return "", err
	}
	amount := max(balance, 0) * between(e.lo, e.hi) / 100
	if amount <= 0 {
		return "", nil
	}
	if err := openWallet(ctx, tx, run.chat, run.victim); err != nil {
		return "", err
	}
	if err := moveCoins(ctx, tx, run.chat, run.acc, run.victim, amount); err != nil {
		return "", err
	}
	return fmt.Sprintf("🤝 Devolveu %s pra @%s", Coins(amount), run.victim.JID.User), nil
}
