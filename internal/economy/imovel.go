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

type estate struct {
	name       string
	emoji      string
	place      string
	price      int64
	guard      float64
	dailyBonus int64
	payBonus   int64
	tax        int64
	staff      int
	storage    int64
}

var estates = []estate{
	{name: "Barraca de camping", emoji: "⛺", place: "numa barraca de camping", price: 1_000, guard: 0.01, dailyBonus: 5, payBonus: 1, tax: 20, staff: 1, storage: 10},
	{name: "Barraco", emoji: "🛖", place: "num barraco", price: 5_000, guard: 0.02, dailyBonus: 10, payBonus: 2, tax: 100, staff: 1, storage: 25},
	{name: "Casinha", emoji: "🏠", place: "numa casinha", price: 20_000, guard: 0.03, dailyBonus: 15, payBonus: 3, tax: 300, staff: 2, storage: 50},
	{name: "Casa com piscina", emoji: "🏡", place: "numa casa com piscina", price: 80_000, guard: 0.05, dailyBonus: 20, payBonus: 4, tax: 1_000, staff: 3, storage: 100},
	{name: "Cobertura", emoji: "🏢", place: "numa cobertura", price: 300_000, guard: 0.07, dailyBonus: 30, payBonus: 5, tax: 3_000, staff: 4, storage: 200},
	{name: "Mansão", emoji: "🏰", place: "numa mansão", price: 1_000_000, guard: 0.1, dailyBonus: 40, payBonus: 6, tax: 10_000, staff: 5, storage: 400},
	{name: "Ilha particular", emoji: "🏝️", place: "numa ilha particular", price: 5_000_000, guard: 0.12, dailyBonus: 50, payBonus: 8, tax: 40_000, staff: 6, storage: 800},
	{name: "Base em Marte", emoji: "🪐", place: "numa base em Marte", price: 25_000_000, guard: 0.15, dailyBonus: 60, payBonus: 10, tax: 150_000, staff: 8, storage: 1_500},
}

func estateAt(level int) (estate, bool) {
	if level < 1 || level > len(estates) {
		return estate{}, false
	}
	return estates[level-1], true
}

func (e estate) label() string {
	return e.emoji + " *" + e.name + "*"
}

func (e estate) perks() string {
	return fmt.Sprintf("+%d%% de proteção, +%d%% no !daily, +%d%% no !trabalhar, cabe %d %s, guarda %s, IPTU de %s por dia", percent(e.guard), e.dailyBonus, e.payBonus, e.staff, pluralWord(int64(e.staff), "empregado", "empregados"), core.CountLabel(int(e.storage), "item", "itens"), Coins(e.tax))
}

func (p Profession) housed(level int) Profession {
	e, ok := estateAt(level)
	if !ok {
		return p
	}
	p.estate = level
	p.guard += e.guard
	p.dailyBonus += e.dailyBonus
	p.min, p.max = p.min*(100+e.payBonus)/100, p.max*(100+e.payBonus)/100
	return p
}

func homeText(level int) string {
	e, ok := estateAt(level)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%s Mora %s", e.emoji, e.place)
}

func (b *bot) imovel(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	args := strings.Fields(core.FoldLetters(core.MessageText(evt.Message)))[1:]
	actions := map[string]func(context.Context, *events.Message) error{
		"comprar": b.buyEstate, "subir": b.buyEstate, "upgrade": b.buyEstate, "mudar": b.buyEstate,
		"guardar": b.stashItem, "deixar": b.stashItem,
		"tirar": b.unstashItem, "retirar": b.unstashItem, "pegar": b.unstashItem,
		"guardados": b.showStash, "armario": b.showStash, "deposito": b.showStash,
	}
	if len(args) > 0 {
		if run, ok := actions[args[0]]; ok {
			return run(ctx, evt)
		}
	}
	person := core.FirstMentionOrSender(evt).ToNonAD()
	job, err := b.ProfessionOf(ctx, chat, b.Account(ctx, person))
	if err != nil {
		return err
	}
	_, err = b.SendMentions(ctx, chat, estatesMenu(person, job.estate), []types.JID{person})
	return err
}

func estatesMenu(person types.JID, level int) string {
	lines := []string{"🏘️ *Imóveis*", ""}
	for i, e := range estates {
		mark := "🔒"
		if i+1 <= level {
			mark = "✅"
		}
		lines = append(lines, fmt.Sprintf("%s %s: %s\n    _%s_", mark, e.label(), Coins(e.price), e.perks()))
	}
	home := fmt.Sprintf("@%s ainda não tem onde cair morto 🪹", person.User)
	if h := homeText(level); h != "" {
		home = fmt.Sprintf("@%s: %s", person.User, h)
	}
	lines = append(lines, "", home)
	if next, ok := estateAt(level + 1); ok {
		lines = append(lines, fmt.Sprintf("_Manda !imovel comprar pra se mudar pra %s por %s. O imóvel antigo vai pro lixo e o IPTU é cobrado todo dia junto com o imposto_", next.label(), Coins(next.price)))
	}
	return strings.Join(lines, "\n")
}

func (b *bot) buyEstate(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	job, err := b.ProfessionOf(ctx, chat, acc)
	if err != nil {
		return err
	}
	next, ok := estateAt(job.estate + 1)
	if !ok {
		return b.SendText(ctx, chat, "🪐 Você já mora em Marte. Não tem mais pra onde subir, só pro espaço sideral")
	}
	balance, err := b.Wallets.moveIn(ctx, chat, acc, job.estate+1, next.price)
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, chat, fmt.Sprintf("Você não tem moeda pra isso: %s custa %s. Confere no !saldo", next.label(), Coins(next.price)))
	}
	if err != nil {
		return fmt.Errorf("comprar imóvel: %w", err)
	}
	text := fmt.Sprintf("🔑 @%s pegou as chaves e agora mora %s! Pagou %s\n✨ %s\n💰 Saldo: %s", person.User, next.place, Coins(next.price), next.perks(), Coins(balance))
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (s *walletStore) moveIn(ctx context.Context, chat types.JID, acc core.Account, level int, price int64) (balance int64, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		err := tx.QueryRowContext(ctx,
			"UPDATE carteiras SET saldo = saldo - ?, imovel = ? WHERE chat = ? AND pessoa = ? AND imovel = ? AND saldo >= ? RETURNING saldo",
			price, level, chat.String(), acc.Key, level-1, price,
		).Scan(&balance)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoFunds
		}
		return err
	})
	return balance, err
}
