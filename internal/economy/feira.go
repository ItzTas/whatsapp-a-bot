package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	stallSpan       = 24 * time.Hour
	stallsPerSeller = 5
	stallFeePercent = 5
	stallShown      = 20
	stallFields     = "id, vendedor, vendedor_jid, item, quantidade, preco, expira_em"
)

var (
	errNoStall    = errors.New("anúncio não existe")
	errOwnStall   = errors.New("anúncio da própria pessoa")
	errStallsFull = errors.New("anúncios demais")
)

type stall struct {
	id       int64
	seller   core.Account
	item     item
	quantity int64
	price    int64
	expires  time.Time
}

type stallSale struct {
	stall  stall
	grades []grade
	fee    int64
}

func (b *bot) feira(ctx context.Context, evt *events.Message) error {
	actions := map[string]func(context.Context, *events.Message, []string) error{
		"vender": b.pitchStall, "anunciar": b.pitchStall,
		"comprar": b.buyStall,
		"tirar":   b.dropStall, "cancelar": b.dropStall, "remover": b.dropStall,
	}
	fields := strings.Fields(core.MessageText(evt.Message))
	if len(fields) > 1 {
		if run, ok := actions[core.FoldLetters(fields[1])]; ok {
			return run(ctx, evt, fields[2:])
		}
	}
	return b.showFair(ctx, evt, strings.Join(fields[1:], " "))
}

func (b *bot) showFair(ctx context.Context, evt *events.Message, search string) error {
	chat := evt.Info.Chat.ToNonAD()
	now := time.Now()
	open, err := b.Wallets.stalls(ctx, chat, now)
	if err != nil {
		return fmt.Errorf("ler a feira: %w", err)
	}
	folded := core.FoldLetters(search)
	shown := slices.DeleteFunc(open, func(st stall) bool { return !strings.Contains(core.FoldLetters(st.item.name), folded) })
	if len(shown) == 0 && search != "" {
		return b.SendText(ctx, chat, fmt.Sprintf("Ninguém tá vendendo *%s* na feira agora 🤷 Manda !feira pra ver tudo que tem", search))
	}
	if len(shown) == 0 {
		return b.SendText(ctx, chat, "🛒 A feira tá vazia. Seja o primeiro: !feira vender <item> [quantidade] <preço>")
	}
	hidden := len(shown) - stallShown
	shown = shown[:min(len(shown), stallShown)]

	nicknames := b.ChatNicknames(ctx, chat)
	names := make([]string, len(shown))
	var wg sync.WaitGroup
	for i, st := range shown {
		wg.Go(func() { names[i] = b.DisplayName(ctx, st.seller.JID, nicknames) })
	}
	wg.Wait()

	lines := []string{"🛒 *Feira da galera*", ""}
	for i, st := range shown {
		lines = append(lines, fmt.Sprintf("*%d* · %s %s · %s · %s · sai em %s",
			st.id, rarities[st.item.rarity].emoji, quantityLabel(st.item, st.quantity), Coins(st.price), names[i], core.FormatWait(st.expires.Sub(now))))
	}
	if hidden > 0 {
		lines = append(lines, fmt.Sprintf("_...e mais %d. Manda !feira <item> pra procurar um_", hidden))
	}
	lines = append(lines, "", fmt.Sprintf("_Pra comprar: !feira comprar <número>. Pra anunciar: !feira vender <item> [quantidade] <preço>. A feira fica com %d%% de quem vende_", stallFeePercent))
	return b.SendText(ctx, chat, strings.Join(lines, "\n"))
}

func (b *bot) pitchStall(ctx context.Context, evt *events.Message, args []string) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	q := parseItemQuery("!feira " + strings.Join(args, " "))
	if q.name == "" || len(q.numbers) == 0 {
		return b.SendText(ctx, chat, "Use assim: !feira vender peixe dourado 300 (o número é o preço). Pra vender vários: !feira vender peixe dourado 3 900, que são 3 por 900 no total")
	}
	price := q.numbers[len(q.numbers)-1]
	q.numbers = q.numbers[:len(q.numbers)-1]

	it, have, problem, err := b.heldForSale(ctx, chat, acc, q.name)
	if err != nil {
		return err
	}
	if problem != "" {
		return b.SendText(ctx, chat, problem)
	}
	if it.illegal {
		return b.SendText(ctx, chat, fmt.Sprintf("🚫 %s não entra na feira, que aqui é tudo honesto. Passa no !mercadonegro vender, ou vende escondido com !vender @pessoa", it.label()))
	}
	quantity := q.quantity(have)
	if quantity > have {
		return b.SendText(ctx, chat, "Você só tem "+quantityLabel(it, have))
	}

	now := time.Now()
	id, err := b.Wallets.pitch(ctx, chat, acc, it, quantity, price, now.Add(stallSpan), now)
	if errors.Is(err, errStallsFull) {
		return b.SendText(ctx, chat, fmt.Sprintf("Você já tem %d anúncios na feira, que é o máximo 🧺 Tira um com !feira tirar <número> ou espera alguém comprar", stallsPerSeller))
	}
	if err != nil {
		return fmt.Errorf("anunciar na feira: %w", err)
	}
	text := fmt.Sprintf("🛒 @%s botou na feira %s por %s\n_No mercado sairia por %s. %s_\n\nPra comprar: !feira comprar %d\nO anúncio fica %s e o item continua na mochila até alguém comprar. A feira fica com %d%% da venda",
		person.User, quantityLabel(it, quantity), Coins(price), Coins(marketPayout(it, quantity)), storePriceNote(it, quantity), id, core.FormatWait(stallSpan), stallFeePercent)
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func stallNumber(args []string) (int64, bool) {
	if len(args) == 0 {
		return 0, false
	}
	id, err := strconv.ParseInt(strings.TrimPrefix(args[0], "#"), 10, 64)
	return id, err == nil && id > 0
}

func (b *bot) buyStall(ctx context.Context, evt *events.Message, args []string) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	id, ok := stallNumber(args)
	if !ok {
		return b.SendText(ctx, chat, "Qual anúncio? Manda !feira pra ver os números e depois !feira comprar 12")
	}
	sale, err := b.Wallets.buyStall(ctx, chat, b.Account(ctx, person), id, time.Now())
	st := sale.stall
	if errors.Is(err, errNoItems) {
		if err := b.Wallets.discardStall(ctx, chat, st.id); err != nil {
			return fmt.Errorf("tirar anúncio sem item da feira: %w", err)
		}
		text := fmt.Sprintf("❌ @%s não tem mais %s, então o anúncio saiu da feira", st.seller.JID.User, quantityLabel(st.item, st.quantity))
		_, err = b.SendMentions(ctx, chat, text, []types.JID{st.seller.JID})
		return err
	}
	refusals := map[error]string{
		errNoStall:  fmt.Sprintf("O anúncio %d não tá mais na feira (alguém comprou, o dono tirou ou venceu) 🤷 Manda !feira pra ver os que tão abertos", id),
		errOwnStall: fmt.Sprintf("Comprar o próprio anúncio? Pra tirar da feira é !feira tirar %d 🤡", id),
		ErrNoFunds:  fmt.Sprintf("Você não tem %s pra pagar 🪹", Coins(st.price)),
		errBagFull:  carryRefusal(st.item, "a sua"),
	}
	for reason, text := range refusals {
		if errors.Is(err, reason) {
			return b.SendText(ctx, chat, text)
		}
	}
	if err != nil {
		return fmt.Errorf("comprar na feira: %w", err)
	}
	text := fmt.Sprintf("✅ @%s comprou %s%s de @%s na feira por %s\n🧾 A feira ficou com %s de comissão e @%s recebeu %s",
		person.User, quantityLabel(st.item, st.quantity), gradesLabel(sale.grades), st.seller.JID.User, Coins(st.price), Coins(sale.fee), st.seller.JID.User, Coins(st.price-sale.fee))
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person, st.seller.JID})
	return err
}

func (b *bot) dropStall(ctx context.Context, evt *events.Message, args []string) error {
	chat := evt.Info.Chat.ToNonAD()
	id, ok := stallNumber(args)
	if !ok {
		return b.SendText(ctx, chat, "Qual anúncio? Manda !feira pra ver os números e depois !feira tirar 12")
	}
	st, err := b.Wallets.dropStall(ctx, chat, b.Account(ctx, evt.Info.Sender), id)
	if errors.Is(err, errNoStall) {
		return b.SendText(ctx, chat, fmt.Sprintf("Você não tem nenhum anúncio %d na feira 🤷 Manda !feira pra ver os números", id))
	}
	if err != nil {
		return fmt.Errorf("tirar anúncio da feira: %w", err)
	}
	return b.SendText(ctx, chat, fmt.Sprintf("🧺 Tirou %s da feira", quantityLabel(st.item, st.quantity)))
}

func scanStall(row interface{ Scan(dest ...any) error }) (stall, error) {
	var (
		st      stall
		jid, id string
		expires int64
	)
	if err := row.Scan(&st.id, &st.seller.Key, &jid, &id, &st.quantity, &st.price, &expires); err != nil {
		return stall{}, err
	}
	seller, err := types.ParseJID(jid)
	if err != nil {
		return stall{}, err
	}
	st.seller.JID, st.expires = seller, time.Unix(expires, 0)
	st.item, _ = itemByID(id)
	return st, nil
}

func (s *walletStore) stalls(ctx context.Context, chat types.JID, now time.Time) (open []stall, err error) {
	rows, err := s.db.QueryContext(ctx, "SELECT "+stallFields+" FROM feira_anuncios WHERE chat = ? AND expira_em > ? ORDER BY id DESC", chat.String(), now.Unix())
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		st, err := scanStall(rows)
		if err != nil {
			return nil, err
		}
		if st.item.id != "" {
			open = append(open, st)
		}
	}
	return open, rows.Err()
}

func (s *walletStore) rivalStalls(ctx context.Context, chat types.JID, acc core.Account, now time.Time) (count int, err error) {
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM feira_anuncios WHERE chat = ? AND expira_em > ? AND vendedor != ?",
		chat.String(), now.Unix(), acc.Key).Scan(&count)
	return count, err
}

func (s *walletStore) pitch(ctx context.Context, chat types.JID, acc core.Account, it item, quantity, price int64, expires, now time.Time) (id int64, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM feira_anuncios WHERE chat = ? AND expira_em <= ?", chat.String(), now.Unix()); err != nil {
			return err
		}
		var others int
		err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM feira_anuncios WHERE chat = ? AND vendedor = ? AND item != ?",
			chat.String(), acc.Key, it.id).Scan(&others)
		if err != nil {
			return err
		}
		if others >= stallsPerSeller {
			return errStallsFull
		}
		return tx.QueryRowContext(ctx,
			`INSERT INTO feira_anuncios (chat, vendedor, vendedor_jid, item, quantidade, preco, expira_em) VALUES (?, ?, ?, ?, ?, ?, ?)
			 ON CONFLICT (chat, vendedor, item) DO UPDATE SET vendedor_jid = excluded.vendedor_jid, quantidade = excluded.quantidade,
			 preco = excluded.preco, expira_em = excluded.expira_em
			 RETURNING id`,
			chat.String(), acc.Key, acc.JID.String(), it.id, quantity, price, expires.Unix(),
		).Scan(&id)
	})
	return id, err
}

func (s *walletStore) buyStall(ctx context.Context, chat types.JID, buyer core.Account, id int64, now time.Time) (sale stallSale, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		row := tx.QueryRowContext(ctx, "SELECT "+stallFields+" FROM feira_anuncios WHERE chat = ? AND id = ? AND expira_em > ?", chat.String(), id, now.Unix())
		st, err := scanStall(row)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && st.item.id == "") {
			return errNoStall
		}
		if err != nil {
			return err
		}
		sale.stall = st
		if st.seller.Key == buyer.Key {
			return errOwnStall
		}
		if sale.grades, err = moveItems(ctx, tx, chat, st.seller, buyer, st.item.id, st.quantity); err != nil {
			return err
		}
		if err := openWallet(ctx, tx, chat, st.seller); err != nil {
			return err
		}
		if err := moveCoins(ctx, tx, chat, buyer, st.seller, st.price); err != nil {
			return err
		}
		if sale.fee = st.price * stallFeePercent / 100; sale.fee > 0 {
			if err := debitBalance(ctx, tx, chat, st.seller, sale.fee); err != nil {
				return err
			}
		}
		_, err = tx.ExecContext(ctx, "DELETE FROM feira_anuncios WHERE chat = ? AND id = ?", chat.String(), st.id)
		return err
	})
	return sale, err
}

func (s *walletStore) discardStall(ctx context.Context, chat types.JID, id int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM feira_anuncios WHERE chat = ? AND id = ?", chat.String(), id)
	return err
}

func (s *walletStore) dropStall(ctx context.Context, chat types.JID, seller core.Account, id int64) (stall, error) {
	row := s.db.QueryRowContext(ctx, "DELETE FROM feira_anuncios WHERE chat = ? AND id = ? AND vendedor = ? RETURNING "+stallFields,
		chat.String(), id, seller.Key)
	st, err := scanStall(row)
	if errors.Is(err, sql.ErrNoRows) {
		return stall{}, errNoStall
	}
	return st, err
}
