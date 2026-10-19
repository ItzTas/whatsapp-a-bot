package economy

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

type favoriteScope struct {
	pool    []item
	missing string
	empty   string
}

func (b *bot) favoritar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	held, favorites, err := b.shelf(ctx, chat, acc)
	if err != nil {
		return err
	}
	pool := make([]item, len(held))
	for i, s := range held {
		pool[i] = s.item
	}
	scope := favoriteScope{
		pool:    pool,
		missing: "%s não tá na sua mochila 🎒",
		empty:   "Sua mochila tá vazia, não tem o que favoritar 🪹",
	}
	chosen, problems := pickFavorites(core.MessageText(evt.Message), scope)
	if chosen == nil && problems == nil {
		return b.SendText(ctx, chat, "Diz o que quer favoritar: !favoritar todynho, !favoritar #12 #40 ou !favoritar tudo ⭐\n\n_Item favorito não sai em nenhuma venda. Só as cópias repetidas saem no !vender duplicados, que sempre deixa 1 na mochila_"+favoritesLine(favorites))
	}
	fresh := slices.DeleteFunc(chosen, func(it item) bool { return favorites[it.id] })
	if len(fresh) == 0 && len(chosen) > 0 {
		problems = append(problems, "Já tava tudo nos favoritos ⭐")
	}
	if len(fresh) == 0 {
		return b.SendText(ctx, chat, strings.Join(problems, "\n"))
	}
	if err := b.Wallets.favorite(ctx, chat, acc, fresh); err != nil {
		return fmt.Errorf("favoritar itens: %w", err)
	}
	text := fmt.Sprintf("⭐ @%s favoritou %s\n_Favorito não sai em nenhum !vender. Pra vender só as cópias repetidas, manda !vender duplicados_",
		person.User, itemLabels(fresh))
	_, err = b.SendMentions(ctx, chat, text+problemsNote(problems), []types.JID{person})
	return err
}

func (b *bot) desfavoritar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	favorites, err := b.Wallets.favorites(ctx, chat, acc)
	if err != nil {
		return fmt.Errorf("ler favoritos: %w", err)
	}
	scope := favoriteScope{
		pool:    favoriteItems(favorites),
		missing: "%s não tá nos seus favoritos",
		empty:   "Você não tem nenhum item favorito ⭐",
	}
	chosen, problems := pickFavorites(core.MessageText(evt.Message), scope)
	if chosen == nil && problems == nil {
		return b.SendText(ctx, chat, "Diz o que quer tirar dos favoritos: !desfavoritar todynho, !desfavoritar #12 #40 ou !desfavoritar tudo"+favoritesLine(favorites))
	}
	if len(chosen) == 0 {
		return b.SendText(ctx, chat, strings.Join(problems, "\n"))
	}
	if err := b.Wallets.unfavorite(ctx, chat, acc, chosen); err != nil {
		return fmt.Errorf("desfavoritar itens: %w", err)
	}
	text := fmt.Sprintf("💔 @%s tirou %s dos favoritos, e agora dá pra vender normalmente", person.User, itemLabels(chosen))
	_, err = b.SendMentions(ctx, chat, text+problemsNote(problems), []types.JID{person})
	return err
}

func pickFavorites(text string, scope favoriteScope) ([]item, []string) {
	inPool := func(it item) bool {
		return slices.ContainsFunc(scope.pool, func(p item) bool { return p.id == it.id })
	}
	if numbers, ok := parseItemNumbers(text); ok {
		return numberedFavorites(numbers, inPool, scope.missing)
	}
	q := parseItemQuery(text)
	if q.name == "" && q.all && len(scope.pool) == 0 {
		return []item{}, []string{scope.empty}
	}
	if q.name == "" && q.all {
		return slices.Clone(scope.pool), nil
	}
	if q.name == "" {
		return nil, nil
	}
	matches := preferItems(findItems(q.name), inPool)
	if problem := itemQueryProblem(q.name, matches); problem != "" {
		return []item{}, []string{problem}
	}
	if !inPool(matches[0]) {
		return []item{}, []string{fmt.Sprintf(scope.missing, matches[0].label())}
	}
	return matches[:1], nil
}

func numberedFavorites(numbers []int, inPool func(item) bool, missing string) ([]item, []string) {
	chosen := []item{}
	var problems []string
	seen := map[int]bool{}
	for _, n := range numbers {
		if seen[n] {
			continue
		}
		seen[n] = true
		it, ok := itemByNumber(n)
		if !ok {
			problems = append(problems, fmt.Sprintf("#%d não é o número de nenhum item", n))
			continue
		}
		if !inPool(it) {
			problems = append(problems, fmt.Sprintf("#%d "+missing, n, it.label()))
			continue
		}
		chosen = append(chosen, it)
	}
	return chosen, problems
}

func favoriteItems(favorites map[string]bool) []item {
	var owned []item
	for _, it := range catalog {
		if favorites[it.id] {
			owned = append(owned, it)
		}
	}
	return owned
}

func favoritesLine(favorites map[string]bool) string {
	if len(favorites) == 0 {
		return ""
	}
	return "\n\n⭐ Seus favoritos: " + itemLabels(favoriteItems(favorites))
}

func itemLabels(items []item) string {
	labels := make([]string, len(items))
	for i, it := range items {
		labels[i] = it.label()
	}
	return core.JoinWithAnd(labels)
}

func problemsNote(problems []string) string {
	if len(problems) == 0 {
		return ""
	}
	return "\n\n⚠️ Ficou de fora:\n" + strings.Join(problems, "\n")
}

func favoriteRefusal(it item) string {
	return fmt.Sprintf("⭐ %s tá nos seus favoritos, então não vende. Manda !desfavoritar %s antes, ou !vender duplicados pra vender só as cópias repetidas", it.label(), it.name)
}

func favoritesKept(kept []item) []string {
	if len(kept) == 0 {
		return nil
	}
	return []string{"⭐ Os favoritos ficaram na mochila: " + itemLabels(kept)}
}

func (b *bot) shelf(ctx context.Context, chat types.JID, acc core.Account) (held []stack, favorites map[string]bool, err error) {
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { held, err = b.Wallets.inventory(gctx, chat, acc); return err })
	g.Go(func() (err error) { favorites, err = b.Wallets.favorites(gctx, chat, acc); return err })
	if err := g.Wait(); err != nil {
		return nil, nil, fmt.Errorf("ler mochila e favoritos: %w", err)
	}
	return held, favorites, nil
}

func (b *bot) heldForSale(ctx context.Context, chat types.JID, acc core.Account, name string) (item, int64, string, error) {
	var (
		it        item
		have      int64
		problem   string
		favorites map[string]bool
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { it, have, problem, err = b.heldItem(gctx, chat, acc, name); return err })
	g.Go(func() (err error) { favorites, err = b.Wallets.favorites(gctx, chat, acc); return err })
	if err := g.Wait(); err != nil {
		return item{}, 0, "", err
	}
	if problem == "" && favorites[it.id] {
		problem = favoriteRefusal(it)
	}
	return it, have, problem, nil
}

func (s *walletStore) favorites(ctx context.Context, chat types.JID, acc core.Account) (favorites map[string]bool, err error) {
	rows, err := s.db.QueryContext(ctx, "SELECT item FROM favoritos WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	favorites = map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if _, known := itemByID(id); known {
			favorites[id] = true
		}
	}
	return favorites, rows.Err()
}

func (s *walletStore) favorite(ctx context.Context, chat types.JID, acc core.Account, items []item) error {
	return s.inTx(ctx, func(tx *ledgerTx) error {
		for _, it := range items {
			_, err := tx.ExecContext(ctx,
				"INSERT INTO favoritos (chat, pessoa, item) VALUES (?, ?, ?) ON CONFLICT DO NOTHING",
				chat.String(), acc.Key, it.id,
			)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *walletStore) unfavorite(ctx context.Context, chat types.JID, acc core.Account, items []item) error {
	ids := make([]any, len(items))
	for i, it := range items {
		ids[i] = it.id
	}
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM favoritos WHERE chat = ? AND pessoa = ? AND item IN ("+core.SQLPlaceholders(len(ids))+")",
		append([]any{chat.String(), acc.Key}, ids...)...,
	)
	return err
}
