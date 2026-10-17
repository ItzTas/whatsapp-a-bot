package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	titleStealPercent  = 150
	titleRefundPercent = 50
	titleHexChance     = 0.4
)

var errTitleChanged = errors.New("o título mudou de dono")

type title struct {
	id     string
	emoji  string
	name   string
	price  int64
	flavor string
	bonus  flex
}

var titles = []title{
	{id: "herdeiro", emoji: "🧸", name: "Herdeiro(a)", price: 5_000, flavor: "nunca trabalhou um dia na vida",
		bonus: flex{effect: "o !daily rende 15% a mais", apply: func(p profession) profession {
			p.dailyBonus += 15
			return p
		}}},
	{id: "coach", emoji: "🗣️", name: "Coach Quântico", price: 8_000, flavor: "vende curso de como ficar rico vendendo curso",
		bonus: flex{effect: "o !trabalhar paga 5% a mais", apply: func(p profession) profession {
			p.min, p.max = p.min*105/100, p.max*105/100
			return p
		}}},
	{id: "patrao", emoji: "🤑", name: "Patrão", price: 10_000, flavor: "paga o salário de todo mundo com atraso",
		bonus: titleAbuse(workAbuse)},
	{id: "influencer", emoji: "📸", name: "Influencer Milionário", price: 15_000, flavor: "ficou rico com publi de bet",
		bonus: flex{effect: "recebe 5% de volta quando perde no !apostar e no !cassino", apply: func(p profession) profession {
			p.cashback += 5
			return p
		}}},
	{id: "rei-do-camarote", emoji: "🍾", name: "Rei do Camarote", price: 20_000, flavor: "chega na balada com vela de faísca",
		bonus: flex{effect: "fura a fila: espera 5% menos no !trabalhar e no " + hasteCommands, apply: func(p profession) profession {
			p.every = p.every * 95 / 100
			p.haste -= 5
			return p
		}}},
	{id: "barao", emoji: "🎩", name: "Barão", price: 30_000, flavor: "usa monóculo e fala de dividendos no café da manhã",
		bonus: flex{effect: "a poupança rende 1 ponto a mais", apply: func(p profession) profession {
			p.interest++
			return p
		}}},
	{id: "magnata", emoji: "💼", name: "Magnata", price: 50_000, flavor: "compra empresa por impulso",
		bonus: flex{effect: "o lucro na venda do !acoes rende 10% a mais", apply: func(p profession) profession {
			p.stockProfit += 10
			return p
		}}},
	{id: "chefao", emoji: "🕴️", name: "Chefão da Máfia", price: 75_000, flavor: "faz propostas que ninguém pode recusar",
		bonus: titleAbuse(robAbuse)},
	{id: "sheik", emoji: "🛢️", name: "Sheik do Petróleo", price: 100_000, flavor: "achou petróleo cavando o quintal",
		bonus: flex{effect: "anda com escolta: quem tenta !roubar, !furtar, !sequestrar ou !matar a pessoa tem 10% menos chance", apply: func(p profession) profession {
			p.guard += 0.1
			return p
		}}},
	{id: "lenda", emoji: "🌟", name: "Lenda Viva", price: 250_000, flavor: "já tem estátua em vida",
		bonus: flex{effect: "tem fã-clube: o voto conta 1 a mais no !votar, no !impeachment e no júri do !processar", apply: func(p profession) profession {
			p.voteWeight = p.ballotWeight() + 1
			return p
		}}},
	{id: "farao", emoji: "🔺", name: "Faraó", price: 500_000, flavor: "construiu uma pirâmide só pra guardar o saldo",
		bonus: titleCurse("encosto")},
	{id: "imperador", emoji: "👑", name: "Imperador(a) do Grupo", price: 1_000_000, flavor: "o grupo inteiro trabalha pra essa pessoa",
		bonus: titleCurse("miseria")},
}

func titleCurse(id string) flex {
	c, _ := charmByID(id)
	return flex{
		effect: fmt.Sprintf("pode !amaldicoar os outros com %s (%s), com pelo menos %d%% de chance de pegar", c.label(), c.effect, percent(titleHexChance)),
		apply: func(p profession) profession {
			p.faith.curses = append(slices.Clip(p.faith.curses), id)
			p.faith.hexChance = max(p.faith.hexChance, titleHexChance)
			return p
		},
	}
}

func titleAbuse(kind abuseKind) flex {
	return flex{
		effect: fmt.Sprintf("a cada %s pode abusar do poder: %s", formatWait(kind.every), kind.perk),
		apply: func(p profession) profession {
			if slices.Contains(p.abuses, kind) {
				return p
			}
			p.abuses = append(slices.Clip(p.abuses), kind)
			return p
		},
	}
}

func (p profession) titled(id string) profession {
	t, ok := titleByID(id)
	if !ok {
		return p
	}
	p = t.bonus.apply(p)
	p.titleID = id
	return p
}

func titlePerks(p profession) []string {
	t, ok := titleByID(p.titleID)
	if !ok {
		return nil
	}
	return []string{"👑 Usa o título " + t.label() + " (já conta nas vantagens acima)"}
}

type titleHolder struct {
	acc  account
	paid int64
}

func titleByID(id string) (title, bool) {
	i := slices.IndexFunc(titles, func(t title) bool { return t.id == id })
	if i < 0 {
		return title{}, false
	}
	return titles[i], true
}

func (t title) label() string {
	return t.emoji + " *" + t.name + "*"
}

func (t title) priceFrom(h titleHolder, held bool) int64 {
	if !held {
		return t.price
	}
	return max(t.price, h.paid*titleStealPercent/100)
}

func titleBadge(id string) string {
	t, ok := titleByID(id)
	if !ok {
		return ""
	}
	return t.emoji + " " + t.name + " "
}

func findTitle(query string) (title, bool) {
	q := normalizeItemName(query)
	if q == "" {
		return title{}, false
	}
	matchers := []func(key string) bool{
		func(key string) bool { return key == q },
		func(key string) bool { return strings.HasPrefix(key, q) },
		func(key string) bool { return strings.Contains(key, q) },
	}
	for _, match := range matchers {
		i := slices.IndexFunc(titles, func(t title) bool {
			return match(normalizeItemName(t.name)) || match(normalizeItemName(t.id))
		})
		if i >= 0 {
			return titles[i], true
		}
	}
	return title{}, false
}

func (b *bot) titulo(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	args := strings.Fields(foldLetters(mentionTagPattern.ReplaceAllString(messageText(evt.Message), "")))[1:]
	if len(args) == 0 {
		return b.titlesMenu(ctx, chat)
	}
	actions := map[string]func(context.Context, *events.Message, string) error{
		"comprar": b.buyTitle, "tomar": b.buyTitle, "roubar": b.buyTitle,
		"usar": b.wearTitle, "equipar": b.wearTitle, "trocar": b.wearTitle,
	}
	action, ok := actions[args[0]]
	if !ok {
		return b.sendText(ctx, chat, "Use assim: !titulo, !titulo comprar <nome> ou !titulo usar <nome>")
	}
	return action(ctx, evt, strings.Join(args[1:], " "))
}

func (b *bot) titlesMenu(ctx context.Context, chat types.JID) error {
	holders, err := b.wallets.titleHolders(ctx, chat)
	if err != nil {
		return fmt.Errorf("ler títulos: %w", err)
	}
	lines := []string{
		"👑 *Títulos do grupo*",
		fmt.Sprintf("_Cada título tem um dono só. Pra tomar de alguém, paga %d%% do que a pessoa pagou, e ela recebe %d%% do que tinha pago de consolo_", titleStealPercent, titleRefundPercent),
		"",
	}
	var mentions []types.JID
	for _, t := range titles {
		h, held := holders[t.id]
		status := "livre"
		if held {
			status = fmt.Sprintf("de @%s, pra tomar", h.acc.jid.User)
			mentions = append(mentions, h.acc.jid)
		}
		lines = append(lines, fmt.Sprintf("%s: %s · %s\n    _%s_\n    ✨ %s", t.label(), coins(t.priceFrom(h, held)), status, t.flavor, t.bonus.effect))
	}
	lines = append(lines, "", "_Manda !titulo comprar <nome>. Quem compra já sai usando, e o título aparece antes do nome no !saldo e no !ricos. Só o título que tá usando dá a vantagem. Pra trocar entre os seus: !titulo usar <nome> (ou nenhum)_")
	_, err = b.sendMentions(ctx, chat, strings.Join(lines, "\n"), mentions)
	return err
}

func (b *bot) buyTitle(ctx context.Context, evt *events.Message, query string) error {
	chat := evt.Info.Chat.ToNonAD()
	buyer := evt.Info.Sender.ToNonAD()
	t, ok := findTitle(query)
	if !ok {
		return b.sendText(ctx, chat, "Não conheço esse título 🤔 Manda !titulo pra ver todos")
	}
	acc := b.account(ctx, buyer)
	holders, err := b.wallets.titleHolders(ctx, chat)
	if err != nil {
		return fmt.Errorf("ler títulos: %w", err)
	}
	h, held := holders[t.id]
	if held && h.acc.key == acc.key {
		return b.sendText(ctx, chat, fmt.Sprintf("%s já é seu 😎 Pra usar: !titulo usar %s", t.label(), strings.ToLower(t.name)))
	}
	price := t.priceFrom(h, held)
	refund, err := b.wallets.claimTitle(ctx, chat, acc, t, price, h, held, time.Now())
	if errors.Is(err, errNoFunds) {
		return b.sendText(ctx, chat, fmt.Sprintf("%s custa %s e você não tem tudo isso. Confere no !saldo", t.label(), coins(price)))
	}
	if errors.Is(err, errTitleChanged) {
		return b.sendText(ctx, chat, "Alguém foi mais rápido e o título mudou de dono agora mesmo. Manda !titulo pra ver o preço novo")
	}
	if err != nil {
		return fmt.Errorf("comprar título: %w", err)
	}
	if !held {
		text := fmt.Sprintf("👑 @%s pagou %s e agora é %s do grupo! Alguém que %s, finalmente reconhecido(a)", buyer.User, coins(price), t.label(), t.flavor)
		_, err = b.sendMentions(ctx, chat, text, []types.JID{buyer})
		return err
	}
	text := fmt.Sprintf("⚔️ *GOLPE DE ESTADO!* @%s pagou %s e tomou o título de %s de @%s!\n💸 @%s levou %s de consolo. Agora quem quiser tomar de volta paga %s",
		buyer.User, coins(price), t.label(), h.acc.jid.User, h.acc.jid.User, coins(refund), coins(price*titleStealPercent/100))
	_, err = b.sendMentions(ctx, chat, text, []types.JID{buyer, h.acc.jid})
	return err
}

func (b *bot) wearTitle(ctx context.Context, evt *events.Message, query string) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.account(ctx, person)
	if slices.Contains([]string{"nenhum", "nada", "tirar"}, query) {
		if err := b.wallets.wearTitle(ctx, chat, acc, ""); err != nil {
			return fmt.Errorf("tirar título: %w", err)
		}
		_, err := b.sendMentions(ctx, chat, fmt.Sprintf("🫥 @%s guardou o título na gaveta e voltou a ser gente comum", person.User), []types.JID{person})
		return err
	}
	t, ok := findTitle(query)
	if !ok {
		return b.sendText(ctx, chat, "Não conheço esse título 🤔 Manda !titulo pra ver todos")
	}
	err := b.wallets.wearTitle(ctx, chat, acc, t.id)
	if errors.Is(err, errTitleChanged) {
		return b.sendText(ctx, chat, fmt.Sprintf("%s não é seu. Dá pra comprar ou tomar com !titulo comprar %s", t.label(), strings.ToLower(t.name)))
	}
	if err != nil {
		return fmt.Errorf("usar título: %w", err)
	}
	_, err = b.sendMentions(ctx, chat, fmt.Sprintf("%s Agora @%s desfila como %s", t.emoji, person.User, t.label()), []types.JID{person})
	return err
}

func (s *walletStore) titleHolders(ctx context.Context, chat types.JID) (holders map[string]titleHolder, err error) {
	rows, err := s.db.QueryContext(ctx, "SELECT titulo, pessoa, jid, pago FROM titulos WHERE chat = ?", chat.String())
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	holders = map[string]titleHolder{}
	for rows.Next() {
		var (
			id, jid string
			h       titleHolder
		)
		if err := rows.Scan(&id, &h.acc.key, &jid, &h.paid); err != nil {
			return nil, err
		}
		if h.acc.jid, err = types.ParseJID(jid); err != nil {
			return nil, err
		}
		holders[id] = h
	}
	return holders, rows.Err()
}

func (s *walletStore) claimTitle(ctx context.Context, chat types.JID, buyer account, t title, price int64, previous titleHolder, held bool, now time.Time) (refund int64, err error) {
	err = s.spend(ctx, chat, buyer, price, func(tx *sql.Tx) error {
		var current string
		err := tx.QueryRowContext(ctx, "SELECT pessoa FROM titulos WHERE chat = ? AND titulo = ?", chat.String(), t.id).Scan(&current)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if current != previous.acc.key {
			return errTitleChanged
		}
		if held {
			if refund, err = dethrone(ctx, tx, chat, t.id, previous); err != nil {
				return err
			}
		}
		_, err = tx.ExecContext(ctx,
			`INSERT INTO titulos (chat, titulo, pessoa, jid, pago, em) VALUES (?, ?, ?, ?, ?, ?)
			 ON CONFLICT (chat, titulo) DO UPDATE SET pessoa = excluded.pessoa, jid = excluded.jid, pago = excluded.pago, em = excluded.em`,
			chat.String(), t.id, buyer.key, buyer.jid.String(), price, now.Unix(),
		)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "UPDATE carteiras SET titulo = ? WHERE chat = ? AND pessoa = ?", t.id, chat.String(), buyer.key)
		return err
	})
	return refund, err
}

func dethrone(ctx context.Context, tx *sql.Tx, chat types.JID, id string, h titleHolder) (int64, error) {
	refund := h.paid * titleRefundPercent / 100
	if err := openWallet(ctx, tx, chat, h.acc); err != nil {
		return 0, err
	}
	_, err := tx.ExecContext(ctx,
		"UPDATE carteiras SET saldo = saldo + ?, titulo = CASE WHEN titulo = ? THEN '' ELSE titulo END WHERE chat = ? AND pessoa = ?",
		refund, id, chat.String(), h.acc.key,
	)
	return refund, err
}

func (s *walletStore) wearTitle(ctx context.Context, chat types.JID, acc account, id string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE carteiras SET titulo = ? WHERE chat = ? AND pessoa = ?
		 AND (? = '' OR EXISTS (SELECT 1 FROM titulos t WHERE t.chat = carteiras.chat AND t.titulo = ? AND t.pessoa = carteiras.pessoa))`,
		id, chat.String(), acc.key, id, id,
	)
	if err != nil {
		return err
	}
	changed, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if changed == 0 && id != "" {
		return errTitleChanged
	}
	return nil
}

func (s *walletStore) titleOf(ctx context.Context, chat types.JID, acc account) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx, "SELECT titulo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.key).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return id, err
}
