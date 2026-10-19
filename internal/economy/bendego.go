package economy

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	bendegoID      = "meteorito-do-bendego"
	bendegoShardID = "fragmento-do-bendego"
	skyRockID      = "rocha-celeste"

	starfallOdds = 0.001

	craterKind  = "cratera"
	craterOpen  = time.Hour
	craterEvery = 24 * time.Hour
	craterLuck  = 2
	craterToll  = 20

	gravityPercent   = 3
	gravityEvery     = 6 * time.Hour
	gravityChatLimit = 4
)

type primalPower struct {
	command string
	passive string
	active  string
	run     func(*bot, context.Context, *events.Message, item) error
}

var primalPowers = map[string]primalPower{
	bendegoID: {
		command: "bendego",
		passive: fmt.Sprintf("libera os míticos em toda extração, e o campo gravitacional dele puxa %d%% do que os outros ganham no !trabalhar e reparte entre quem tem Bendegó a cada %s",
			gravityPercent, core.FormatWait(gravityEvery)),
		active: fmt.Sprintf("!primordial bendego abre uma cratera no grupo por %s, uma vez por dia: quem mandar !minerar nela minera como se tivesse relíquia (sai mítico) e com mais sorte, e paga %d%% do valor de cada achado de pedágio pra você",
			core.FormatWait(craterOpen), craterToll),
		run: (*bot).openCrater,
	},
}

func (p primalPower) perk(it item) string {
	var parts []string
	if it.gear != nil {
		parts = append(parts, gearPerk(it.gear))
	}
	return strings.Join(append(parts, p.passive, p.active), "; ")
}

func extractionRelic(it item) bool {
	return it.rarity == relic && it.gear != nil && len(it.gear.activities) == 1
}

func extractionRelicNames() string {
	var names []string
	for _, it := range catalog {
		if extractionRelic(it) {
			names = append(names, it.name)
		}
	}
	return core.JoinWithOr(names)
}

func starfallSource() string {
	return fmt.Sprintf("%s (%s de chance por vez, e só com uma relíquia de extração equipada: %s)", extractionCommands(), oddsText(starfallOdds), extractionRelicNames())
}

func starfallNote(rock item) string {
	shard, _ := itemByID(rock.shard)
	whole, _ := itemByID(shard.forms)
	return fmt.Sprintf("🌠 *CAIU DO CÉU!* Isso só aparece pra quem extrai com relíquia, e quase nunca. Abre com !usar %s: em %d%% das vezes sai %s, e %d deles viram %s sozinhos",
		core.FoldLetters(rock.name), percent(rock.shardOdds), shard.label(), shard.shards, whole.label())
}

type crater struct {
	owner core.Account
	until time.Time
}

type craterSave struct {
	Chat  types.JID          `json:"chat"`
	Owner core.StoredAccount `json:"owner"`
	Until time.Time          `json:"until"`
}

func (c crater) open() bool {
	return c.owner.Key != ""
}

func (c crater) charges(acc core.Account) bool {
	return c.open() && c.owner.Key != acc.Key
}

func (c crater) dig(a activity) activity {
	if !c.open() {
		return a
	}
	a.mythics = true
	a.plainLuck += craterLuck
	return a
}

func (c crater) note(g gathering) string {
	if !c.open() || g.noTool || g.wait > 0 {
		return ""
	}
	if g.toll == 0 {
		return "\n\n🕳️ Minerou na cratera do Bendegó: aqui sai mítico mesmo sem relíquia"
	}
	return fmt.Sprintf("\n\n🕳️ Minerou na cratera do Bendegó de @%s e pagou %s de pedágio", c.owner.JID.User, Coins(g.toll))
}

func (b *bot) craterIn(chat types.JID, a activity, now time.Time) crater {
	if a.command != mining.command {
		return crater{}
	}
	v, ok := b.craters.Load(chat.String())
	if !ok {
		return crater{}
	}
	c := v.(crater)
	if now.Before(c.until) {
		return c
	}
	b.craters.CompareAndDelete(chat.String(), c)
	return crater{}
}

func (b *bot) openCrater(ctx context.Context, evt *events.Message, it item) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	now := time.Now()
	if current := b.craterIn(chat, mining, now); current.open() {
		return b.craterTaken(ctx, chat, current, now)
	}
	wait, err := b.Wallets.stamp(ctx, chat, acc, cooldown{column: "cratera_em", every: craterEvery}, now)
	if err != nil {
		return fmt.Errorf("abrir cratera: %w", err)
	}
	if wait > 0 {
		_, err := b.SendMentions(ctx, chat, fmt.Sprintf("🕳️ @%s já abriu a cratera do Bendegó hoje. Ele volta a cavar em %s", person.User, core.FormatWait(wait)), []types.JID{person})
		return err
	}
	pit := crater{owner: acc, until: now.Add(craterOpen)}
	if prev, loaded := b.craters.LoadOrStore(chat.String(), pit); loaded {
		if err := b.Wallets.clearCooldown(ctx, chat, acc, "cratera_em"); err != nil {
			return fmt.Errorf("devolver a cratera: %w", err)
		}
		return b.craterTaken(ctx, chat, prev.(crater), now)
	}
	b.KeepGame(ctx, craterKind, chat.String(), craterSave{Chat: chat, Owner: core.StoreAccount(acc), Until: pit.until}, pit.until)
	text := fmt.Sprintf("%s🕳️ *Cratera aberta!* @%s cravou o %s no chão e abriu uma cratera, que fica aberta por %s\n\nQuem mandar !minerar aqui minera como se tivesse relíquia (sai mítico) e com %dx mais sorte, mas paga %d%% do valor de cada achado de pedágio pra @%s",
		it.emoji, person.User, it.name, core.FormatWait(craterOpen), craterLuck+1, craterToll, person.User)
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) craterTaken(ctx context.Context, chat types.JID, c crater, now time.Time) error {
	text := fmt.Sprintf("🕳️ Já tem uma cratera do Bendegó aberta aqui, a de @%s, que fecha em %s. Aproveita e vai !minerar nela", c.owner.JID.User, core.FormatWait(c.until.Sub(now)))
	_, err := b.SendMentions(ctx, chat, text, []types.JID{c.owner.JID})
	return err
}

func (b *bot) restoreCrater(_ context.Context, saved core.SavedGame) error {
	var state craterSave
	if err := json.Unmarshal(saved.State, &state); err != nil {
		return fmt.Errorf("ler a cratera do Bendegó: %w", err)
	}
	if time.Now().Before(state.Until) {
		b.craters.Store(saved.Key, crater{owner: state.Owner.Account(), until: state.Until})
	}
	return nil
}

func payToll(ctx context.Context, tx *ledgerTx, chat types.JID, miner, owner core.Account, finds []haul) (int64, error) {
	var worth int64
	for _, h := range finds {
		if h.found.id != "" {
			worth += marketPayoutOf(h.found, []grade{{h.grade, 1}}, marketSalePercent)
		}
	}
	var balance int64
	if err := tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), miner.Key).Scan(&balance); err != nil {
		return 0, err
	}
	fee := min(max(balance, 0), worth*craterToll/100)
	if fee == 0 {
		return 0, nil
	}
	if err := openWallet(ctx, tx, chat, owner); err != nil {
		return 0, err
	}
	return fee, moveCoins(ctx, tx, chat, miner, owner, fee)
}

func assembleShards(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, id string) error {
	shard, ok := itemByID(id)
	if !ok || shard.forms == "" {
		return nil
	}
	held, err := gradesIn(ctx, tx, backpackTable, chat, acc, id)
	if err != nil {
		return err
	}
	var have int64
	for _, g := range held {
		have += g.quantity
	}
	wholes := have / shard.shards
	if wholes == 0 {
		return nil
	}
	if _, err := takeGrades(ctx, tx, chat, acc, id, wholes*shard.shards); err != nil {
		return err
	}
	if err := addItems(ctx, tx, chat, acc, shard.forms, wholes); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO montagens_avisos (chat, jid, item, quantidade) VALUES (?, ?, ?, ?)",
		chat.String(), acc.JID.String(), shard.forms, wholes)
	return err
}

type assembly struct {
	owner    types.JID
	whole    item
	quantity int64
}

func (a assembly) text() string {
	i := slices.IndexFunc(catalog, func(it item) bool { return it.forms == a.whole.id })
	shard := catalog[i]
	return fmt.Sprintf("🧩☄️ *Os fragmentos se juntaram!* @%s juntou %s na mochila, e eles se encaixaram sozinhos e viraram %s, um item primordial\n%s\n\n_Manda !item %s pra ver tudo o que ele faz_",
		a.owner.User, quantityLabel(shard, shard.shards*a.quantity), quantityLabel(a.whole, a.quantity), a.whole.summary(), core.FoldLetters(a.whole.name))
}

func (s *walletStore) collectAssemblies(ctx context.Context, chat types.JID) (built []assembly, err error) {
	rows, err := s.db.QueryContext(ctx, "DELETE FROM montagens_avisos WHERE chat = ? RETURNING jid, item, quantidade", chat.String())
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var (
			a       assembly
			jid, id string
		)
		if err := rows.Scan(&jid, &id, &a.quantity); err != nil {
			return nil, err
		}
		if a.owner, err = types.ParseJID(jid); err != nil {
			return nil, err
		}
		if a.whole, _ = itemByID(id); a.whole.id != "" {
			built = append(built, a)
		}
	}
	return built, rows.Err()
}

func (b *bot) announceAssemblies(ctx context.Context, chat types.JID) {
	built, err := b.Wallets.collectAssemblies(ctx, chat)
	if err != nil {
		b.Client.Log.Errorf("ler montagens em %s: %v", chat, err)
		return
	}
	for _, a := range built {
		if _, err := b.SendMentions(ctx, chat, a.text(), []types.JID{a.owner}); err != nil {
			b.Client.Log.Errorf("avisar a montagem de %s em %s: %v", a.whole.id, chat, err)
		}
	}
}

func pullGravity(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, earned int64, now time.Time) (int64, error) {
	pull := earned * gravityPercent / 100
	if pull == 0 {
		return 0, nil
	}
	var held, others bool
	err := tx.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM inventarios WHERE chat = ? AND item = ? AND pessoa = ?),
		        EXISTS (SELECT 1 FROM inventarios WHERE chat = ? AND item = ? AND pessoa <> ?)`,
		chat.String(), bendegoID, acc.Key, chat.String(), bendegoID, acc.Key,
	).Scan(&held, &others)
	if err != nil || held || !others {
		return 0, err
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO campo_gravitacional (chat, acumulado, pago_em) VALUES (?, ?, ?)
		 ON CONFLICT (chat) DO UPDATE SET
		     pago_em = CASE WHEN campo_gravitacional.acumulado = 0 THEN excluded.pago_em ELSE campo_gravitacional.pago_em END,
		     acumulado = campo_gravitacional.acumulado + excluded.acumulado`,
		chat.String(), pull, now.Unix(),
	)
	return pull, err
}

func gravityText(pulled int64) string {
	if pulled == 0 {
		return ""
	}
	return fmt.Sprintf("\n☄️ O campo gravitacional do Bendegó puxou %s (%d%%) do seu trampo, e quem tem Bendegó divide isso a cada %s", Coins(pulled), gravityPercent, core.FormatWait(gravityEvery))
}

type gravityPayout struct {
	pool    int64
	share   int64
	holders []types.JID
}

func (b *bot) runGravity(ctx context.Context) {
	ticker := time.NewTicker(core.ReminderTick)
	defer ticker.Stop()
	for {
		if b.Client.IsLoggedIn() {
			b.settleGravity(ctx, time.Now())
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (b *bot) settleGravity(ctx context.Context, now time.Time) {
	chats, err := b.Wallets.gravityDue(ctx, now)
	if err != nil {
		b.Client.Log.Errorf("procurar campos gravitacionais pra pagar: %v", err)
		return
	}
	var g errgroup.Group
	g.SetLimit(gravityChatLimit)
	for _, chat := range chats {
		g.Go(func() error {
			if err := b.gravityChat(ctx, chat, now); err != nil {
				b.Client.Log.Errorf("pagar o campo gravitacional em %s: %v", chat, err)
			}
			return nil
		})
	}
	_ = g.Wait()
}

func (b *bot) gravityChat(ctx context.Context, chat types.JID, now time.Time) error {
	p, err := b.Wallets.payGravity(ctx, chat, now)
	if err != nil || p.share == 0 {
		return err
	}
	owners := make([]string, len(p.holders))
	for i, h := range p.holders {
		owners[i] = "@" + h.User
	}
	text := fmt.Sprintf("☄️ *Campo gravitacional*\n\nO Bendegó puxou %s do !trabalhar do pessoal nas últimas horas, e cada dono de Bendegó levou %s: %s\n\n_Quem tem Bendegó recebe %d%% do trampo dos outros a cada %s_",
		Coins(p.pool), Coins(p.share), core.JoinWithAnd(owners), gravityPercent, core.FormatWait(gravityEvery))
	return b.announce(ctx, chat, text, p.holders...)
}

func (s *walletStore) gravityDue(ctx context.Context, now time.Time) (chats []types.JID, err error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT chat FROM campo_gravitacional WHERE acumulado > 0 AND pago_em <= ?",
		now.Add(-gravityEvery).Unix(),
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		chat, err := types.ParseJID(raw)
		if err != nil {
			return nil, err
		}
		if ledgers.linked(chat) {
			continue
		}
		chats = append(chats, chat)
	}
	return chats, rows.Err()
}

func (s *walletStore) payGravity(ctx context.Context, chat types.JID, now time.Time) (p gravityPayout, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		err := tx.QueryRowContext(ctx,
			"SELECT acumulado FROM campo_gravitacional WHERE chat = ? AND acumulado > 0 AND pago_em <= ?",
			chat.String(), now.Add(-gravityEvery).Unix(),
		).Scan(&p.pool)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		keys, err := gravityHolders(ctx, tx, chat, &p)
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			p.share = p.pool / int64(len(keys))
		}
		for _, key := range keys {
			if _, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", p.share, chat.String(), key); err != nil {
				return err
			}
		}
		_, err = tx.ExecContext(ctx, "UPDATE campo_gravitacional SET acumulado = acumulado - ?, pago_em = ? WHERE chat = ?",
			p.share*int64(len(keys)), now.Unix(), chat.String())
		return err
	})
	return p, err
}

func gravityHolders(ctx context.Context, tx *ledgerTx, chat types.JID, p *gravityPayout) (keys []string, err error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT DISTINCT c.pessoa, c.jid FROM inventarios i
		 JOIN carteiras c ON c.chat = i.chat AND c.pessoa = i.pessoa
		 WHERE i.chat = ? AND i.item = ? ORDER BY c.pessoa`,
		chat.String(), bendegoID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var key, raw string
		if err := rows.Scan(&key, &raw); err != nil {
			return nil, err
		}
		jid, err := types.ParseJID(raw)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
		p.holders = append(p.holders, jid)
	}
	return keys, rows.Err()
}

func (b *bot) primordial(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	acc := b.Account(ctx, evt.Info.Sender.ToNonAD())
	query := parseItemQuery(core.MessageText(evt.Message)).name
	if query == "" {
		return b.listPrimordials(ctx, chat, acc)
	}
	it, _, problem, err := b.heldItem(ctx, chat, acc, query)
	if err != nil {
		return err
	}
	if problem != "" {
		return b.SendText(ctx, chat, problem)
	}
	power, ok := primalPowers[it.id]
	if !ok {
		return b.SendText(ctx, chat, fmt.Sprintf("%s não tem habilidade pra despertar no !primordial. Manda !primordial pra ver as dos seus itens", it.label()))
	}
	return power.run(b, ctx, evt, it)
}

func (b *bot) listPrimordials(ctx context.Context, chat types.JID, acc core.Account) error {
	owned, err := b.Wallets.holding(ctx, chat, acc, func(it item) bool { _, ok := primalPowers[it.id]; return ok })
	if err != nil {
		return fmt.Errorf("ler primordiais: %w", err)
	}
	if len(owned) == 0 {
		return b.SendText(ctx, chat, primordialHint())
	}
	lines := []string{"⚫ *Seus itens primordiais*"}
	for _, it := range owned {
		power := primalPowers[it.id]
		lines = append(lines, "", it.label(), "✨ "+power.active, "🌀 Sem fazer nada: "+power.passive)
	}
	return b.SendText(ctx, chat, strings.Join(lines, "\n"))
}

func primordialHint() string {
	lines := []string{"⚫ Você ainda não tem nenhum item primordial, a raridade mais alta que existe. Eles têm um poder que vale só de ter na mochila e uma habilidade que se desperta com !primordial <item>", ""}
	for _, it := range catalog {
		if _, ok := primalPowers[it.id]; ok {
			lines = append(lines, fmt.Sprintf("%s se consegue %s", it.label(), core.JoinWithOr(primordialSources(it))))
		}
	}
	shard, _ := itemByID(bendegoShardID)
	rock, _ := itemByID(skyRockID)
	lines = append(lines, fmt.Sprintf("%s sai da %s, que cai no %s", shard.label(), rock.label(), starfallSource()))
	return strings.Join(lines, "\n")
}
