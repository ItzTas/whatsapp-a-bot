package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

var errStashFull = errors.New("imóvel sem espaço")

type stashing struct {
	grades []grade
	home   estate
	stored int64
}

func (s stashing) free() int64 {
	return max(s.home.storage-s.stored, 0)
}

func stashQuery(text string) itemQuery {
	fields := strings.Fields(text)
	return parseItemQuery(strings.Join(fields[min(1, len(fields)):], " "))
}

func (b *bot) stashItem(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	q := stashQuery(core.MessageText(evt.Message))
	if q.name == "" {
		return b.SendText(ctx, chat, "Diz o que quer guardar: !imovel guardar todynho (ou !imovel guardar todynho 3, ou tudo)")
	}
	it, have, problem, err := b.heldItem(ctx, chat, acc, q.name)
	if err != nil {
		return err
	}
	if problem != "" {
		return b.SendText(ctx, chat, problem)
	}
	quantity := q.quantity(have)
	if quantity > have {
		return b.SendText(ctx, chat, "Você só tem "+quantityLabel(it, have))
	}
	st, err := b.Wallets.stash(ctx, chat, acc, it.id, quantity)
	if errors.Is(err, errNoEstate) {
		return b.SendText(ctx, chat, "🪹 Guardar onde, se você não tem nem onde morar? Compra um imóvel no !imovel primeiro")
	}
	if errors.Is(err, errStashFull) {
		return b.SendText(ctx, chat, fmt.Sprintf("📦 Não cabe: %s só guarda %s e já tem %d lá dentro, então sobra espaço pra %d. Sobe de imóvel no !imovel ou tira alguma coisa com !imovel tirar",
			st.home.label(), core.CountLabel(int(st.home.storage), "item", "itens"), st.stored, st.free()))
	}
	if errors.Is(err, errNoItems) {
		return b.SendText(ctx, chat, "Sua mochila mudou no meio do caminho e não dá mais 🤨 Confere no !mochila")
	}
	if err != nil {
		return fmt.Errorf("guardar item: %w", err)
	}
	text := fmt.Sprintf("🏠 @%s guardou %s%s em casa. Lá ninguém furta e a polícia não apreende, mas também não vale nada enquanto não voltar pra mochila\n📦 %d de %d espaços ocupados",
		person.User, quantityLabel(it, quantity), gradesLabel(st.grades), st.stored, st.home.storage)
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) unstashItem(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	q := stashQuery(core.MessageText(evt.Message))
	if q.name == "" {
		return b.SendText(ctx, chat, "Diz o que quer tirar de casa: !imovel tirar todynho (ou !imovel tirar todynho 3, ou tudo)")
	}
	stored, err := b.Wallets.stashed(ctx, chat, acc)
	if err != nil {
		return fmt.Errorf("ler guardados: %w", err)
	}
	matches := preferItems(findItems(q.name), ownedIn(stored))
	if problem := itemQueryProblem(q.name, matches); problem != "" {
		return b.SendText(ctx, chat, problem)
	}
	it := matches[0]
	have := quantityOf(stored, it.id)
	if have == 0 {
		return b.SendText(ctx, chat, fmt.Sprintf("Não tem %s guardado em casa 🏠 Manda !imovel guardados pra ver o que tem lá", it.label()))
	}
	quantity := q.quantity(have)
	if quantity > have {
		return b.SendText(ctx, chat, "Você só tem "+quantityLabel(it, have)+" guardado em casa")
	}
	grades, err := b.Wallets.unstash(ctx, chat, acc, it.id, quantity)
	if errors.Is(err, errNoItems) {
		return b.SendText(ctx, chat, "O que tinha em casa mudou no meio do caminho 🤨 Confere no !imovel guardados")
	}
	if err != nil {
		return fmt.Errorf("tirar item de casa: %w", err)
	}
	text := fmt.Sprintf("🎒 @%s tirou %s%s de casa e pôs na mochila", person.User, quantityLabel(it, quantity), gradesLabel(grades))
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) showStash(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := core.FirstMentionOrSender(evt).ToNonAD()
	acc := b.Account(ctx, person)
	level, err := b.Wallets.estateLevel(ctx, chat, acc)
	if err != nil {
		return fmt.Errorf("ler imóvel: %w", err)
	}
	stored, err := b.Wallets.stashed(ctx, chat, acc)
	if err != nil {
		return fmt.Errorf("ler guardados: %w", err)
	}
	_, err = b.SendMentions(ctx, chat, stashText(person, level, stored), []types.JID{person})
	return err
}

func stashText(person types.JID, level int, stored []stack) string {
	home, ok := estateAt(level)
	if !ok {
		return fmt.Sprintf("🪹 @%s não tem imóvel, então não tem onde guardar nada. Compra um no !imovel", person.User)
	}
	if len(stored) == 0 {
		return fmt.Sprintf("🏠 @%s não guardou nada em casa ainda. %s guarda %s\n\n_Manda !imovel guardar <item> pra deixar em casa, longe do !furtar e da polícia_",
			person.User, home.label(), core.CountLabel(int(home.storage), "item", "itens"))
	}
	type lot struct {
		item  item
		grade grade
	}
	var (
		units, worth int64
		lots         []lot
	)
	for _, s := range stored {
		units += s.quantity
		worth += s.worth()
		for _, g := range s.grades {
			lots = append(lots, lot{s.item, g})
		}
	}
	lines := []string{fmt.Sprintf("🏠 *Guardado na casa de @%s* (%s)", person.User, home.label())}
	lines = append(lines, categoryGroups(lots, func(l lot) item { return l.item }, func(l lot) string {
		return fmt.Sprintf("#%d %s %dx %s %s%s · 🪙 %s", l.item.number(), rarities[l.item.rarity].emoji, l.grade.quantity, l.item.emoji,
			l.item.name, l.grade.quality.tag(), core.Thousands(l.grade.quality.price(l.item.value)))
	})...)
	lines = append(lines, "",
		fmt.Sprintf("📦 %d de %d espaços ocupados, valendo %s", units, home.storage, Coins(worth)),
		"_Item guardado não é furtado nem apreendido, mas só funciona de novo quando volta pra mochila. Manda !imovel tirar <item> pra pegar de volta_")
	return strings.Join(lines, "\n")
}

func (s *walletStore) stashed(ctx context.Context, chat types.JID, acc core.Account) ([]stack, error) {
	return loadStacks(ctx, s.db, stashTable, chat, acc)
}

func (s *walletStore) stash(ctx context.Context, chat types.JID, acc core.Account, id string, quantity int64) (st stashing, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		var level int
		err := tx.QueryRowContext(ctx,
			"SELECT c.imovel, COALESCE((SELECT SUM(g.quantidade) FROM guardados g WHERE g.chat = c.chat AND g.pessoa = c.pessoa), 0) FROM carteiras c WHERE c.chat = ? AND c.pessoa = ?",
			chat.String(), acc.Key,
		).Scan(&level, &st.stored)
		if errors.Is(err, sql.ErrNoRows) {
			return errNoEstate
		}
		if err != nil {
			return err
		}
		home, ok := estateAt(level)
		if !ok {
			return errNoEstate
		}
		st.home = home
		if st.stored+quantity > home.storage {
			return errStashFull
		}
		if st.grades, err = shiftItems(ctx, tx, backpackTable, stashTable, chat, acc, id, quantity); err != nil {
			return err
		}
		st.stored += quantity
		return nil
	})
	return st, err
}

func (s *walletStore) unstash(ctx context.Context, chat types.JID, acc core.Account, id string, quantity int64) (grades []grade, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) (err error) {
		grades, err = shiftItems(ctx, tx, stashTable, backpackTable, chat, acc, id, quantity)
		return err
	})
	return grades, err
}

func shiftItems(ctx context.Context, tx *ledgerTx, from, to string, chat types.JID, acc core.Account, id string, quantity int64) ([]grade, error) {
	var wear int64
	err := tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(usos), 0) FROM "+from+" WHERE chat = ? AND pessoa = ? AND item = ?",
		chat.String(), acc.Key, id).Scan(&wear)
	if err != nil {
		return nil, err
	}
	grades, err := takeGradesFrom(ctx, tx, from, chat, acc, id, quantity)
	if err != nil {
		return nil, err
	}
	if err := addGradesTo(ctx, tx, to, chat, acc, id, grades); err != nil {
		return nil, err
	}
	_, err = tx.ExecContext(ctx, "UPDATE "+to+" SET usos = MAX(usos, ?) WHERE chat = ? AND pessoa = ? AND item = ?",
		wear, chat.String(), acc.Key, id)
	return grades, err
}
