package economy

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const gardenBeds = 4

var (
	errGardenFull  = errors.New("horta cheia")
	errGardenMaxed = errors.New("horta no tamanho máximo")
	bedPrices      = []int64{1_000, 2_500, 5_000, 10_000}
)

type crop struct {
	yields string
	grows  time.Duration
	least  int64
	most   int64
	wild   int
}

var gardenGoods = []item{
	{id: "semente-de-alface", name: "Semente de alface", emoji: "🌱", rarity: common, value: 20, shop: true,
		seed: &crop{yields: "alface", grows: 30 * time.Minute, least: 2, most: 4, wild: 60}},
	{id: "alface", name: "Alface", emoji: "🥬", rarity: common, value: 22},
	{id: "semente-de-milho", name: "Semente de milho", emoji: "🌱", rarity: common, value: 40, shop: true,
		seed: &crop{yields: "espiga-de-milho", grows: time.Hour, least: 2, most: 4, wild: 50}},
	{id: "espiga-de-milho", name: "Espiga de milho", emoji: "🌽", rarity: common, value: 45, aliases: []string{"milho"}},
	{id: "semente-de-girassol", name: "Semente de girassol", emoji: "🌱", rarity: common, value: 30, shop: true,
		seed: &crop{yields: "girassol", grows: 90 * time.Minute, least: 3, most: 6, wild: 40}},
	{id: "semente-de-tomate", name: "Semente de tomate", emoji: "🌱", rarity: uncommon, value: 80, shop: true,
		seed: &crop{yields: "tomate", grows: 2 * time.Hour, least: 3, most: 5, wild: 30}},
	{id: "tomate", name: "Tomate", emoji: "🍅", rarity: uncommon, value: 60},
	{id: "muda-de-roseira", name: "Muda de roseira", emoji: "🪴", rarity: uncommon, value: 60, shop: true, aliases: []string{"roseira"},
		seed: &crop{yields: "rosa", grows: 3 * time.Hour, least: 4, most: 8, wild: 25}},
	{id: "rama-de-mandioca", name: "Rama de mandioca", emoji: "🌿", rarity: uncommon, value: 120, shop: true, aliases: []string{"maniva"},
		seed: &crop{yields: "mandioca", grows: 4 * time.Hour, least: 3, most: 6, wild: 25}},
	{id: "mandioca", name: "Mandioca", emoji: "🍠", rarity: uncommon, value: 90, aliases: []string{"aipim", "macaxeira"}},
	{id: "muda-de-morango", name: "Muda de morango", emoji: "🪴", rarity: rare, value: 250, shop: true, aliases: []string{"morangueiro"},
		seed: &crop{yields: "morango", grows: 3 * time.Hour, least: 4, most: 8, wild: 12}},
	{id: "morango", name: "Morango", emoji: "🍓", rarity: rare, value: 100},
	{id: "semente-de-abobora", name: "Semente de abóbora", emoji: "🌱", rarity: rare, value: 300, shop: true,
		seed: &crop{yields: "abobora", grows: 6 * time.Hour, least: 2, most: 4, wild: 10}},
	{id: "abobora", name: "Abóbora", emoji: "🎃", rarity: rare, value: 280, aliases: []string{"jerimum"}},
	{id: "semente-de-melancia", name: "Semente de melancia", emoji: "🌱", rarity: rare, value: 400, shop: true,
		seed: &crop{yields: "melancia", grows: 8 * time.Hour, least: 2, most: 4, wild: 8}},
	{id: "melancia", name: "Melancia", emoji: "🍉", rarity: rare, value: 380},
	{id: "semente-de-cacau", name: "Semente de cacau", emoji: "🌰", rarity: epic, value: 1_500,
		seed: &crop{yields: "cacau", grows: 8 * time.Hour, least: 3, most: 6, wild: 3}},
	{id: "cacau", name: "Fruto de cacau", emoji: "🍫", rarity: epic, value: 700},
	{id: "muda-de-jabuticabeira", name: "Muda de jabuticabeira", emoji: "🪴", rarity: epic, value: 1_200, aliases: []string{"jabuticabeira"},
		seed: &crop{yields: "jabuticaba", grows: 6 * time.Hour, least: 6, most: 12, wild: 3}},
	{id: "jabuticaba", name: "Jabuticaba", emoji: "🫐", rarity: epic, value: 280},
	{id: "muda-de-pau-brasil", name: "Muda de pau-brasil", emoji: "🌳", rarity: legendary, value: 6_000,
		seed: &crop{yields: "tora-de-pau-brasil", grows: 24 * time.Hour, least: 1, most: 3, wild: 1}},
	{id: "tora-de-pau-brasil", name: "Tora de pau-brasil", emoji: "🪵", rarity: legendary, value: 6_000, aliases: []string{"pau brasil"}},
}

func (it item) produce() item {
	p, _ := itemByID(it.seed.yields)
	return p
}

func seedFor(it item) (item, bool) {
	if it.seed != nil {
		return it, true
	}
	i := slices.IndexFunc(catalog, func(s item) bool { return s.seed != nil && s.seed.yields == it.id })
	if i < 0 {
		return item{}, false
	}
	return catalog[i], true
}

func fromGarden(it item) bool {
	_, ok := seedFor(it)
	return ok
}

func seeds() []item {
	return slices.DeleteFunc(slices.Clone(catalog), func(it item) bool { return it.seed == nil })
}

func wildSeed() item {
	pool := seeds()
	return pool[weightedIndex(len(pool), func(i int) int { return pool[i].seed.wild })]
}

func (it item) wildOdds(a activity) float64 {
	total := 0
	for _, s := range seeds() {
		total += s.seed.wild
	}
	return a.seeds * float64(it.seed.wild) / float64(total)
}

func wildSeedSources(it item) []string {
	var spots []string
	for _, a := range activities {
		if a.seeds > 0 {
			spots = append(spots, fmt.Sprintf("%s (de quebra, 1 em %s)", a.command, core.Thousands(int64(1/it.wildOdds(a)+0.5))))
		}
	}
	return spots
}

func seedPerk(it item) string {
	c := it.seed
	return fmt.Sprintf("!plantar bota num canteiro da !horta, e em %s vira de %d a %d %s pra pegar no !colher", core.FormatWait(c.grows), c.least, c.most, it.produce().label())
}

func seedBrief(it item) string {
	c, p := it.seed, it.produce()
	return fmt.Sprintf("planta e vira %d a %d %s %s em %s", c.least, c.most, p.emoji, p.name, core.FormatWait(c.grows))
}

func (p Profession) gardenRoom() int {
	return gardenBeds + p.farmBeds
}

func (p Profession) growth(d time.Duration) time.Duration {
	return d * time.Duration(100-p.farmFaster) / 100
}

type bed struct {
	seed  item
	ready time.Time
}

func (b bed) ripe(now time.Time) bool {
	return !now.Before(b.ready)
}

func (b bed) line(now time.Time) string {
	p := b.seed.produce()
	text := fmt.Sprintf("%s (de %d a %d)", p.label(), b.seed.seed.least, b.seed.seed.most)
	if b.ripe(now) {
		return "✅ " + text + " · pronto pra !colher"
	}
	return "⏳ " + text + " · fica pronto em " + core.FormatWait(b.ready.Sub(now))
}

type plot struct {
	job    Profession
	beds   []bed
	bought int
	bag    []stack
}

func (p plot) room() int {
	return p.job.gardenRoom() + p.bought
}

func (p plot) expansionLine() string {
	if p.bought >= len(bedPrices) {
		return "🧑‍🌾 Já comprou todos os canteiros que dava"
	}
	return fmt.Sprintf("🧑‍🌾 Mais um canteiro sai por %s no !horta expandir", Coins(bedPrices[p.bought]))
}

func (b *bot) plot(ctx context.Context, chat types.JID, acc core.Account) (p plot, err error) {
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { p.job, err = b.ProfessionOf(gctx, chat, acc); return err })
	g.Go(func() (err error) { p.beds, err = b.Wallets.beds(gctx, chat, acc); return err })
	g.Go(func() (err error) { p.bought, err = b.Wallets.boughtBeds(gctx, chat, acc); return err })
	g.Go(func() (err error) { p.bag, err = b.Wallets.inventory(gctx, chat, acc); return err })
	return p, g.Wait()
}

func (s *walletStore) beds(ctx context.Context, chat types.JID, acc core.Account) ([]bed, error) {
	return loadBeds(ctx, s.db, chat, acc)
}

func (s *walletStore) boughtBeds(ctx context.Context, chat types.JID, acc core.Account) (bought int, err error) {
	err = s.db.QueryRowContext(ctx, "SELECT canteiros_comprados FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&bought)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return bought, err
}

func (s *walletStore) expandGarden(ctx context.Context, chat types.JID, acc core.Account) (bought int, price int64, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		err := tx.QueryRowContext(ctx, "SELECT canteiros_comprados FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&bought)
		if err != nil {
			return err
		}
		if bought >= len(bedPrices) {
			return errGardenMaxed
		}
		price = bedPrices[bought]
		if err := debitBalance(ctx, tx, chat, acc, price); err != nil {
			return err
		}
		bought++
		_, err = tx.ExecContext(ctx, "UPDATE carteiras SET canteiros_comprados = ? WHERE chat = ? AND pessoa = ?", bought, chat.String(), acc.Key)
		return err
	})
	return bought, price, err
}

func loadBeds(ctx context.Context, q core.RowsQuerier, chat types.JID, acc core.Account) (beds []bed, err error) {
	rows, err := q.QueryContext(ctx,
		"SELECT semente, pronta_em FROM plantacoes WHERE chat = ? AND pessoa = ? ORDER BY pronta_em, id",
		chat.String(), acc.Key,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var (
			id    string
			ready int64
		)
		if err := rows.Scan(&id, &ready); err != nil {
			return nil, err
		}
		it, known := itemByID(id)
		if !known || it.seed == nil {
			continue
		}
		beds = append(beds, bed{seed: it, ready: time.Unix(ready, 0)})
	}
	return beds, rows.Err()
}

func (s *walletStore) sow(ctx context.Context, chat types.JID, acc core.Account, seed item, quantity int64, room int, ready time.Time) error {
	return s.inTx(ctx, func(tx *ledgerTx) error {
		var used int64
		err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM plantacoes WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&used)
		if err != nil {
			return err
		}
		if used+quantity > int64(room) {
			return errGardenFull
		}
		if err := takeItems(ctx, tx, chat, acc, seed.id, quantity); err != nil {
			return err
		}
		for range quantity {
			_, err := tx.ExecContext(ctx, "INSERT INTO plantacoes (chat, pessoa, semente, pronta_em) VALUES (?, ?, ?, ?)",
				chat.String(), acc.Key, seed.id, ready.Unix())
			if err != nil {
				return err
			}
		}
		return nil
	})
}

type reaping struct {
	crops   []stack
	reaped  int
	growing []bed
}

func (s *walletStore) reap(ctx context.Context, chat types.JID, acc core.Account, gradeLuck int, now time.Time) (r reaping, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		beds, err := loadBeds(ctx, tx, chat, acc)
		if err != nil {
			return err
		}
		cut := slices.IndexFunc(beds, func(b bed) bool { return !b.ripe(now) })
		if cut < 0 {
			cut = len(beds)
		}
		r.reaped, r.growing = cut, beds[cut:]
		if cut == 0 {
			return nil
		}
		_, err = tx.ExecContext(ctx, "DELETE FROM plantacoes WHERE chat = ? AND pessoa = ? AND pronta_em <= ?",
			chat.String(), acc.Key, now.Unix())
		if err != nil {
			return err
		}
		r.crops = gatherCrops(beds[:cut], gradeLuck)
		for _, c := range r.crops {
			if err := addGrades(ctx, tx, chat, acc, c.item.id, c.grades); err != nil {
				return err
			}
		}
		return nil
	})
	return r, err
}

func gatherCrops(ripe []bed, gradeLuck int) []stack {
	counts := map[string]map[quality]int64{}
	for _, b := range ripe {
		c := b.seed.seed
		if counts[c.yields] == nil {
			counts[c.yields] = map[quality]int64{}
		}
		for range c.least + rand.Int64N(c.most-c.least+1) {
			counts[c.yields][rollQuality(gradeLuck, false)]++
		}
	}
	crops := make([]stack, 0, len(counts))
	for id, byQuality := range counts {
		it, _ := itemByID(id)
		s := stack{item: it}
		for _, q := range slices.Sorted(maps.Keys(byQuality)) {
			s.grades = append(s.grades, grade{q, byQuality[q]})
			s.quantity += byQuality[q]
		}
		crops = append(crops, s)
	}
	slices.SortFunc(crops, func(a, b stack) int { return compareItems(a.item, b.item) })
	return crops
}

func (b *bot) horta(ctx context.Context, evt *events.Message) error {
	args := strings.Fields(core.FoldLetters(core.MessageText(evt.Message)))[1:]
	actions := map[string]func(context.Context, *events.Message) error{
		"expandir": b.expandirHorta, "aumentar": b.expandirHorta, "comprar": b.expandirHorta, "canteiro": b.expandirHorta,
	}
	if len(args) > 0 {
		if run, ok := actions[args[0]]; ok {
			return run(ctx, evt)
		}
	}
	chat := evt.Info.Chat.ToNonAD()
	person := core.FirstMentionOrSender(evt).ToNonAD()
	p, err := b.plot(ctx, chat, b.Account(ctx, person))
	if err != nil {
		return fmt.Errorf("ler horta: %w", err)
	}
	_, err = b.SendMentions(ctx, chat, gardenText(person, p, time.Now()), []types.JID{person})
	return err
}

func gardenText(person types.JID, p plot, now time.Time) string {
	room := p.room()
	lines := []string{fmt.Sprintf("🌾 *Horta de @%s* · %d/%d canteiros", person.User, len(p.beds), room)}
	for _, b := range p.beds {
		lines = append(lines, b.line(now))
	}
	if free := room - len(p.beds); free > 0 {
		lines = append(lines, "🟫 "+core.CountLabel(free, "canteiro vazio", "canteiros vazios"))
	}
	lines = append(lines, "", seedsLine(p.bag), p.expansionLine(),
		"_Planta com !plantar <semente> e colhe com !colher. Tem semente na !loja, e de vez em quando acha uma de quebra no !catar, no !vasculhar, no !pescar e no !minerar_")
	return strings.Join(lines, "\n")
}

func (b *bot) expandirHorta(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	job, err := b.ProfessionOf(ctx, chat, acc)
	if err != nil {
		return err
	}
	bought, price, err := b.Wallets.expandGarden(ctx, chat, acc)
	if errors.Is(err, errGardenMaxed) {
		return b.SendText(ctx, chat, fmt.Sprintf("🌾 A sua horta já tá no tamanho máximo, com %d canteiros. Não dá pra comprar mais", job.gardenRoom()+bought))
	}
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, chat, fmt.Sprintf("Você não tem moeda pra isso: o próximo canteiro custa %s. Confere no !saldo", Coins(price)))
	}
	if err != nil {
		return fmt.Errorf("expandir horta: %w", err)
	}
	text := fmt.Sprintf("🧑‍🌾 @%s comprou mais um canteiro por %s, e agora a horta tem %d", person.User, Coins(price), job.gardenRoom()+bought)
	text += "\n" + plot{job: job, bought: bought}.expansionLine()
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func seedsLine(bag []stack) string {
	held := slices.DeleteFunc(slices.Clone(bag), func(s stack) bool { return s.item.seed == nil })
	if len(held) == 0 {
		return "🎒 Nenhuma semente na mochila"
	}
	slices.SortFunc(held, func(a, b stack) int { return cmp.Compare(a.item.seed.grows, b.item.seed.grows) })
	labels := make([]string, len(held))
	for i, s := range held {
		labels[i] = quantityLabel(s.item, s.quantity)
	}
	return "🎒 Sementes na mochila: " + core.JoinWithAnd(labels)
}
