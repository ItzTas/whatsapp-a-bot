package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	MaxHealth           = 100
	healthRegenEvery    = 3 * time.Minute
	maxSafety           = 90
	deathLossMinPercent = 3
	deathLossMaxPercent = 6
	safetyKitID         = "kit-epi"
)

var errHealthy = errors.New("saúde cheia")

type hazard struct {
	chance float64
	harm   int64
	wounds []string
}

func (h hazard) odds(safety int64) float64 {
	return h.chance * float64(100-min(safety, maxSafety)) / 100
}

func (h hazard) roll(safety int64) wound {
	if h.harm == 0 || rand.Float64() >= h.odds(safety) {
		return wound{}
	}
	return wound{damage: h.harm/2 + rand.Int64N(h.harm/2+1), scene: core.PickOne(h.wounds)}
}

type wound struct {
	damage int64
	scene  string
}

type death struct {
	freed types.JID
	lost  int64
	hold  time.Duration
}

type funeral struct {
	hold       time.Duration
	minPercent int64
	maxPercent int64
	heir       core.Account
}

var plainFuneral = funeral{hold: deathHold, minPercent: deathLossMinPercent, maxPercent: deathLossMaxPercent}

type injury struct {
	wound
	death
	health int64
	died   bool
}

func healthAt(stored, since int64, now time.Time) int64 {
	regen := int64(now.Sub(time.Unix(since, 0)) / healthRegenEvery)
	return min(MaxHealth, stored+max(regen, 0))
}

func readHealth(ctx context.Context, q rowQuerier, chat types.JID, acc core.Account, now time.Time) (int64, error) {
	var stored, since int64
	err := q.QueryRowContext(ctx, "SELECT saude, saude_em FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&stored, &since)
	if errors.Is(err, sql.ErrNoRows) {
		return MaxHealth, nil
	}
	if err != nil {
		return 0, err
	}
	return healthAt(stored, since, now), nil
}

func (s *walletStore) Health(ctx context.Context, chat types.JID, acc core.Account, now time.Time) (int64, error) {
	return readHealth(ctx, s.db, chat, acc, now)
}

func setHealth(ctx context.Context, tx *sql.Tx, chat types.JID, acc core.Account, health int64, now time.Time) error {
	_, err := tx.ExecContext(ctx, "UPDATE carteiras SET saude = ?, saude_em = ? WHERE chat = ? AND pessoa = ?", health, now.Unix(), chat.String(), acc.Key)
	return err
}

func hurt(ctx context.Context, tx *sql.Tx, chat types.JID, acc core.Account, w wound, now time.Time) (inj injury, err error) {
	inj.wound = w
	if w.damage == 0 {
		return inj, nil
	}
	current, err := readHealth(ctx, tx, chat, acc, now)
	if err != nil {
		return inj, err
	}
	if inj.health = max(current-w.damage, 0); inj.health > 0 {
		return inj, setHealth(ctx, tx, chat, acc, inj.health, now)
	}
	inj.died = true
	inj.death, err = die(ctx, tx, chat, acc, now)
	return inj, err
}

func die(ctx context.Context, tx *sql.Tx, chat types.JID, acc core.Account, now time.Time) (death, error) {
	return bury(ctx, tx, chat, acc, plainFuneral, now)
}

func bury(ctx context.Context, tx *sql.Tx, chat types.JID, acc core.Account, f funeral, now time.Time) (d death, err error) {
	revenant, err := holdsItem(ctx, tx, chat, acc, ciprianoItem)
	if err != nil {
		return d, err
	}
	if d.hold = f.hold; revenant {
		d.hold /= 2
	}
	_, err = tx.ExecContext(ctx,
		"UPDATE carteiras SET morto_ate = MAX(morto_ate, ?), saude = ?, saude_em = ? WHERE chat = ? AND pessoa = ?",
		now.Add(d.hold).Unix(), MaxHealth, now.Unix(), chat.String(), acc.Key,
	)
	if err != nil {
		return d, err
	}
	if d.freed, err = freeHostage(ctx, tx, chat, acc, now); err != nil {
		return d, err
	}
	if d.lost, err = burnBalance(ctx, tx, chat, acc, f.minPercent, f.maxPercent, 1); err != nil || d.lost == 0 || f.heir.Key == "" {
		return d, err
	}
	_, err = tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", d.lost, chat.String(), f.heir.Key)
	return d, err
}

func funeralText(person types.JID, lost int64) string {
	if lost == 0 {
		return ""
	}
	return fmt.Sprintf("💸 O enterro de @%s levou %s do saldo", person.User, Coins(lost))
}

func (inj injury) text(person types.JID) (string, []types.JID) {
	if inj.damage == 0 {
		return "", nil
	}
	if !inj.died {
		return fmt.Sprintf("\n\n🤕 @%s %s: perdeu %d de saúde (%s). Sara sozinho com o tempo, ou na hora com !usar curativo", person.User, inj.scene, inj.damage, healthText(inj.health)), nil
	}
	lines := []string{
		"",
		fmt.Sprintf("💀 @%s %s e não resistiu: a saúde zerou e morreu. Fica %s sem comando de economia, a não ser que pague o !hospital ou alguém consiga !reviver", person.User, inj.scene, core.FormatWait(inj.hold)),
	}
	if funeral := funeralText(person, inj.lost); funeral != "" {
		lines = append(lines, funeral)
	}
	escape, freed := hostageEscape(inj.freed, "🔓 O refém de @"+person.User+" aproveitou o velório pra fugir: @%s tá livre!")
	if escape != "" {
		lines = append(lines, escape)
	}
	return "\n" + strings.Join(lines, "\n"), freed
}

func healthText(h int64) string {
	heart := "💔"
	if h > MaxHealth/3 {
		heart = "🧡"
	}
	if h > MaxHealth*2/3 {
		heart = "❤️"
	}
	return fmt.Sprintf("%s %d/%d", heart, h, MaxHealth)
}

func oddsText(f float64) string {
	return strings.ReplaceAll(strconv.FormatFloat(math.Round(f*1000)/10, 'f', -1, 64), ".", ",") + "%"
}

func hazardPerk(h hazard, safety int64) string {
	text := fmt.Sprintf("🤕 Trampo perigoso: %s de chance de se machucar no !trabalhar e perder até %d de saúde", oddsText(h.odds(safety)), h.harm)
	if safety > 0 {
		text += " (já conta a proteção)"
	}
	return text
}

func (s *walletStore) heal(ctx context.Context, chat types.JID, acc core.Account, it item, now time.Time) (health int64, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		current, err := readHealth(ctx, tx, chat, acc, now)
		if err != nil {
			return err
		}
		if current >= MaxHealth {
			return errHealthy
		}
		if err := takeItems(ctx, tx, chat, acc, it.id, 1); err != nil {
			return err
		}
		health = min(current+it.heal, MaxHealth)
		return setHealth(ctx, tx, chat, acc, health, now)
	})
	return health, err
}

func (b *bot) healWith(ctx context.Context, evt *events.Message, it item) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	health, err := b.Wallets.heal(ctx, chat, b.Account(ctx, person), it, time.Now())
	if errors.Is(err, errNoItems) {
		return b.SendText(ctx, chat, fmt.Sprintf("Não tem mais %s na sua mochila 🎒", it.label()))
	}
	if errors.Is(err, errHealthy) {
		_, err = b.SendMentions(ctx, chat, fmt.Sprintf("❤️ @%s tá com a saúde cheia. Guarda %s pra quando se machucar", person.User, it.label()), []types.JID{person})
		return err
	}
	if err != nil {
		return fmt.Errorf("usar cura: %w", err)
	}
	text := fmt.Sprintf(it.use, person.User) + fmt.Sprintf("\n\n🩹 Recuperou até %d de saúde: agora tá com %s", it.heal, healthText(health))
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func healPerk(it item) string {
	return fmt.Sprintf("!usar recupera %d de saúde na hora (até %d), e acaba", it.heal, MaxHealth)
}

func (b *bot) saude(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := core.FirstMentionOrSender(evt).ToNonAD()
	acc := b.Account(ctx, person)
	var (
		health int64
		job    Profession
		bag    []stack
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { health, err = b.Wallets.Health(gctx, chat, acc, time.Now()); return err })
	g.Go(func() (err error) { job, err = b.ProfessionOf(gctx, chat, acc); return err })
	g.Go(func() (err error) { bag, err = b.Wallets.inventory(gctx, chat, acc); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler saúde: %w", err)
	}
	_, err := b.SendMentions(ctx, chat, healthReport(person, health, job, bag), []types.JID{person})
	return err
}

func healthReport(person types.JID, health int64, job Profession, bag []stack) string {
	lines := []string{fmt.Sprintf("🩺 Saúde de @%s: %s", person.User, healthText(health))}
	if health < MaxHealth {
		lines = append(lines, fmt.Sprintf("⏳ Recupera 1 ponto a cada %s sozinho: fica 100%% em %s", core.FormatWait(healthRegenEvery), core.FormatWait(time.Duration(MaxHealth-health)*healthRegenEvery)))
	}
	lines = append(lines, "", "*Onde dá pra se machucar*")
	work := fmt.Sprintf("%s !trabalhar (%s): não machuca", job.emoji, job.name)
	if job.hazard.harm > 0 {
		work = fmt.Sprintf("%s !trabalhar (%s): %s de chance, até %d de saúde", job.emoji, job.name, oddsText(job.hazard.odds(job.safety)), job.hazard.harm)
	}
	lines = append(lines, work)
	for _, base := range activities {
		k := job.forage(job.craft.apply(base)).equip(bag)
		line := fmt.Sprintf("%s %s: %s de chance, até %d de saúde", k.emoji, k.command, oddsText(k.hazard.odds(k.safety)), k.hazard.harm)
		if k.safety > 0 {
			line += fmt.Sprintf(" (%d%% menos com a proteção)", min(k.safety, maxSafety))
		}
		lines = append(lines, line)
	}
	lines = append(lines, "",
		fmt.Sprintf("_Se a saúde zerar, morre: fica %s fora da economia e o enterro leva de %d%% a %d%% do saldo. Toda morte leva esse dinheiro, até no !matar e na overdose_", core.FormatWait(deathHold), deathLossMinPercent, deathLossMaxPercent),
		"_Pra se cuidar: !usar curativo ou kit de primeiros socorros, e equipamento de proteção (tem tudo na !loja)_")
	return strings.Join(lines, "\n")
}
