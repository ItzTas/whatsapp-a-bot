package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

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
	tax        int64
}

var estates = []estate{
	{name: "Barraca de camping", emoji: "⛺", place: "numa barraca de camping", price: 1_000, guard: 0.01, dailyBonus: 5, tax: 20},
	{name: "Barraco", emoji: "🛖", place: "num barraco", price: 5_000, guard: 0.02, dailyBonus: 10, tax: 100},
	{name: "Casinha", emoji: "🏠", place: "numa casinha", price: 20_000, guard: 0.03, dailyBonus: 15, tax: 300},
	{name: "Casa com piscina", emoji: "🏡", place: "numa casa com piscina", price: 80_000, guard: 0.05, dailyBonus: 20, tax: 1_000},
	{name: "Cobertura", emoji: "🏢", place: "numa cobertura", price: 300_000, guard: 0.07, dailyBonus: 30, tax: 3_000},
	{name: "Mansão", emoji: "🏰", place: "numa mansão", price: 1_000_000, guard: 0.1, dailyBonus: 40, tax: 10_000},
	{name: "Ilha particular", emoji: "🏝️", place: "numa ilha particular", price: 5_000_000, guard: 0.12, dailyBonus: 50, tax: 40_000},
	{name: "Base em Marte", emoji: "🪐", place: "numa base em Marte", price: 25_000_000, guard: 0.15, dailyBonus: 60, tax: 150_000},
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
	return fmt.Sprintf("+%d%% de proteção, +%d%% no !daily, IPTU de %s por dia", percent(e.guard), e.dailyBonus, coins(e.tax))
}

func (p profession) housed(level int) profession {
	e, ok := estateAt(level)
	if !ok {
		return p
	}
	p.estate = level
	p.guard += e.guard
	p.dailyBonus += e.dailyBonus
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
	args := strings.Fields(foldLetters(messageText(evt.Message)))[1:]
	if len(args) > 0 && slices.Contains([]string{"comprar", "subir", "upgrade", "mudar"}, args[0]) {
		return b.buyEstate(ctx, evt)
	}
	person := firstMentionOrSender(evt).ToNonAD()
	job, err := b.professionOf(ctx, chat, b.account(ctx, person))
	if err != nil {
		return err
	}
	_, err = b.sendMentions(ctx, chat, estatesMenu(person, job.estate), []types.JID{person})
	return err
}

func estatesMenu(person types.JID, level int) string {
	lines := []string{"🏘️ *Imóveis*", ""}
	for i, e := range estates {
		mark := "🔒"
		if i+1 <= level {
			mark = "✅"
		}
		lines = append(lines, fmt.Sprintf("%s %s: %s\n    _%s_", mark, e.label(), coins(e.price), e.perks()))
	}
	home := fmt.Sprintf("@%s ainda não tem onde cair morto 🪹", person.User)
	if h := homeText(level); h != "" {
		home = fmt.Sprintf("@%s: %s", person.User, h)
	}
	lines = append(lines, "", home)
	if next, ok := estateAt(level + 1); ok {
		lines = append(lines, fmt.Sprintf("_Manda !imovel comprar pra se mudar pra %s por %s. O imóvel antigo vai pro lixo e o IPTU é cobrado todo dia junto com o imposto_", next.label(), coins(next.price)))
	}
	return strings.Join(lines, "\n")
}

func (b *bot) buyEstate(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.account(ctx, person)
	job, err := b.professionOf(ctx, chat, acc)
	if err != nil {
		return err
	}
	next, ok := estateAt(job.estate + 1)
	if !ok {
		return b.sendText(ctx, chat, "🪐 Você já mora em Marte. Não tem mais pra onde subir, só pro espaço sideral")
	}
	balance, err := b.wallets.moveIn(ctx, chat, acc, job.estate+1, next.price)
	if errors.Is(err, errNoFunds) {
		return b.sendText(ctx, chat, fmt.Sprintf("Você não tem moeda pra isso: %s custa %s. Confere no !saldo", next.label(), coins(next.price)))
	}
	if err != nil {
		return fmt.Errorf("comprar imóvel: %w", err)
	}
	text := fmt.Sprintf("🔑 @%s pegou as chaves e agora mora %s! Pagou %s\n✨ %s\n💰 Saldo: %s", person.User, next.place, coins(next.price), next.perks(), coins(balance))
	_, err = b.sendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (s *walletStore) moveIn(ctx context.Context, chat types.JID, acc account, level int, price int64) (balance int64, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		err := tx.QueryRowContext(ctx,
			"UPDATE carteiras SET saldo = saldo - ?, imovel = ? WHERE chat = ? AND pessoa = ? AND imovel = ? AND saldo >= ? RETURNING saldo",
			price, level, chat.String(), acc.key, level-1, price,
		).Scan(&balance)
		if errors.Is(err, sql.ErrNoRows) {
			return errNoFunds
		}
		return err
	})
	return balance, err
}
