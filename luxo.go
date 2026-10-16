package main

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	luxuryStockSize   = 5
	luxuryMarkup      = 4
	luxuryMaxPurchase = 10
)

type flex struct {
	effect string
	apply  func(profession) profession
}

func luxuryGoods(it item) bool {
	return it.luxury || (it.rarity >= epic && it.keep && !it.shop)
}

func dailyRand(chat types.JID, now time.Time, salt string) *rand.Rand {
	local := now.In(reminderZone)
	day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC).Unix() / 86400
	h := fnv.New64a()
	_, _ = h.Write([]byte(chat.String() + salt))
	return rand.New(rand.NewPCG(uint64(day), h.Sum64()))
}

func luxuryStock(chat types.JID, now time.Time) []item {
	pool := slices.DeleteFunc(slices.Clone(catalog), func(it item) bool { return !luxuryGoods(it) })
	dailyRand(chat, now, "").Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	stock := pool[:min(luxuryStockSize, len(pool))]
	slices.SortFunc(stock, func(a, b item) int { return cmp.Compare(a.value, b.value) })
	return stock
}

func luxuryPrice(it item) int64 {
	return it.value * luxuryMarkup
}

func nextRestock(now time.Time) time.Duration {
	local := now.In(reminderZone)
	return time.Date(local.Year(), local.Month(), local.Day()+1, 0, 0, 0, 0, reminderZone).Sub(local)
}

func (b *bot) luxo(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	q := parseItemQuery(messageText(evt.Message))
	words := strings.Fields(q.name)
	if len(words) > 0 && slices.Contains([]string{"comprar", "buy"}, words[0]) {
		q.name = strings.Join(words[1:], " ")
		return b.buyLuxuryNamed(ctx, evt, q)
	}
	now := time.Now()
	lines := []string{"💎 *Loja de luxo*", "_Só coisa fina, com preço de grife. O estoque muda todo dia_", ""}
	for _, it := range luxuryStock(chat, now) {
		line := fmt.Sprintf("• %s: %s", it.label(), coins(luxuryPrice(it)))
		if p := perk(it); p != "" {
			line += " (" + p + ")"
		}
		lines = append(lines, line)
	}
	lines = append(lines, "",
		fmt.Sprintf("_Compra com !luxo comprar <item> (ou !comprar <item>). O estoque muda em %s. Revendendo, o mercado paga só %d%% do valor de tabela, então a grife fica com o resto_", formatWait(nextRestock(now)), marketSalePercent))
	return b.sendText(ctx, chat, strings.Join(lines, "\n"))
}

func (b *bot) buyLuxuryNamed(ctx context.Context, evt *events.Message, q itemQuery) error {
	chat := evt.Info.Chat.ToNonAD()
	if q.name == "" {
		return b.sendText(ctx, chat, "Diz o que quer comprar: !luxo comprar rolex. Manda !luxo pra ver o estoque de hoje")
	}
	stock := luxuryStock(chat, time.Now())
	matches := preferItems(findItems(q.name), func(it item) bool { return slices.ContainsFunc(stock, func(s item) bool { return s.id == it.id }) })
	if problem := itemQueryProblem(q.name, matches); problem != "" {
		return b.sendText(ctx, chat, problem)
	}
	return b.buyLuxury(ctx, evt, matches[0], q.quantity(0))
}

func (b *bot) buyLuxury(ctx context.Context, evt *events.Message, it item, quantity int64) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	if !slices.ContainsFunc(luxuryStock(chat, time.Now()), func(s item) bool { return s.id == it.id }) {
		return b.sendText(ctx, chat, fmt.Sprintf("%s não tá no estoque de luxo hoje. Manda !luxo pra ver o que tem", it.label()))
	}
	if quantity <= 0 || quantity > luxuryMaxPurchase {
		return b.sendText(ctx, chat, fmt.Sprintf("Dá pra levar de 1 a %d de uma vez", luxuryMaxPurchase))
	}
	cost := luxuryPrice(it) * quantity
	balance, err := b.wallets.buyItem(ctx, chat, b.account(ctx, person), it, quantity, cost)
	if errors.Is(err, errNoFunds) {
		return b.sendText(ctx, chat, fmt.Sprintf("Você não tem moeda pra isso: %s fica %s. Confere no !saldo", quantityLabel(it, quantity), coins(cost)))
	}
	if err != nil {
		return fmt.Errorf("comprar item de luxo: %w", err)
	}
	text := fmt.Sprintf("💎 @%s passou o cartão black e levou %s por %s. A vendedora até ofereceu um espumante\n💰 Saldo: %s", person.User, quantityLabel(it, quantity), coins(cost), coins(balance))
	_, err = b.sendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (p profession) adorned(owned []item) profession {
	for _, it := range owned {
		if it.flex == nil {
			continue
		}
		p = it.flex.apply(p)
		p.luxuries = append(p.luxuries, it.id)
	}
	return p
}

func luxuryPerks(p profession) []string {
	if len(p.luxuries) == 0 {
		return nil
	}
	labels := make([]string, len(p.luxuries))
	for i, id := range p.luxuries {
		it, _ := itemByID(id)
		labels[i] = it.label()
	}
	return []string{"💎 Ostenta " + joinWithAnd(labels) + " (já conta nas vantagens acima)"}
}

func (s *walletStore) passives(ctx context.Context, chat types.JID, acc account) (owned []item, err error) {
	var ids []any
	for _, it := range catalog {
		if it.flex != nil || it.trade != nil {
			ids = append(ids, it.id)
		}
	}
	rows, err := s.db.QueryContext(ctx,
		"SELECT item FROM inventarios WHERE chat = ? AND pessoa = ? AND item IN ("+sqlPlaceholders(len(ids))+")",
		append([]any{chat.String(), acc.key}, ids...)...,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if it, ok := itemByID(id); ok {
			owned = append(owned, it)
		}
	}
	return owned, rows.Err()
}

func (b *bot) savorLuxury(ctx context.Context, evt *events.Message, it item) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	c, _ := charmByID(it.bless)
	err := b.wallets.savor(ctx, chat, b.account(ctx, person), it, c, time.Now())
	if errors.Is(err, errNoItems) {
		return b.sendText(ctx, chat, fmt.Sprintf("Não tem mais %s na sua mochila 🎒", it.label()))
	}
	if err != nil {
		return fmt.Errorf("usar item de luxo: %w", err)
	}
	text := fmt.Sprintf(it.use, person.User) + fmt.Sprintf("\n\n😇 Ficou com %s por %s\n%s", c.label(), formatWait(c.hold), c.effect)
	_, err = b.sendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (s *walletStore) savor(ctx context.Context, chat types.JID, acc account, it item, c charm, now time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if err := takeItems(ctx, tx, chat, acc, it.id, 1); err != nil {
			return err
		}
		return grantCharm(ctx, tx, chat, acc, acc, c, now)
	})
}
