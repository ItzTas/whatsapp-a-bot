package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/economy"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/llm"
	"go.mau.fi/whatsmeow/types"
	"golang.org/x/sync/errgroup"
)

var errEconomyOff = errors.New("o modo economia tá desligado nesta conversa, então ninguém tem saldo nem profissão aqui")

func (k *iaToolkit) economyTools() []llm.Tool {
	return []llm.Tool{
		{
			Name:        "economia_da_pessoa",
			Description: "Mostra a profissão de uma pessoa do grupo (com o que ela ganha e as vantagens), o saldo de moedas, a posição no ranking de ricos, o que tem na poupança e se tá presa. Use sempre que perguntarem do dinheiro, do trabalho ou da profissão de alguém, em vez de inventar.",
			Properties: map[string]any{
				"nome": map[string]any{"type": "string", "description": "apelido ou nome da pessoa"},
			},
			Required: []string{"nome"},
			Run:      k.personEconomy,
		},
		{
			Name:        "mais_ricos",
			Description: fmt.Sprintf("Mostra as %d pessoas mais ricas do grupo, com o saldo de cada uma.", economy.RichestCount),
			Run:         k.richestPeople,
		},
	}
}

func (k *iaToolkit) personEconomy(ctx context.Context, input json.RawMessage) (string, error) {
	args, err := decodeToolInput[struct {
		Name string `json:"nome"`
	}](input)
	if err != nil {
		return "", err
	}
	if !k.b.economy.EconomyOn(ctx, k.chat) {
		return "", errEconomyOff
	}
	m, err := k.findMember(args.Name)
	if err != nil {
		return "", err
	}
	acc := k.b.Account(ctx, m.jid)
	var (
		balance    int64
		rank       int
		saved      economy.Savings
		hasSavings bool
		job        economy.Profession
		jailLeft   time.Duration
		deathLeft  time.Duration
		health     int64
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { balance, rank, err = k.b.economy.Wallets.Balance(gctx, k.chat, acc); return err })
	g.Go(func() (err error) {
		saved, hasSavings, err = k.b.economy.Wallets.Savings(gctx, k.chat, acc)
		return err
	})
	g.Go(func() (err error) { job, err = k.b.economy.ProfessionOf(gctx, k.chat, acc); return err })
	g.Go(func() (err error) {
		jailLeft, err = k.b.economy.Wallets.SentenceLeft(gctx, k.chat, acc, time.Now())
		return err
	})
	g.Go(func() (err error) {
		deathLeft, err = k.b.economy.Wallets.DeathLeft(gctx, k.chat, acc, time.Now())
		return err
	})
	g.Go(func() (err error) {
		health, err = k.b.economy.Wallets.Health(gctx, k.chat, acc, time.Now())
		return err
	})
	if err := g.Wait(); err != nil {
		return "", err
	}

	lines := []string{m.name + ":", "Saldo: " + economy.Coins(balance)}
	if balance > 0 {
		lines = append(lines, fmt.Sprintf("Posição no ranking de ricos: %dº", rank))
	}
	if hasSavings {
		lines = append(lines, fmt.Sprintf("Poupança: %s, que voltam com %s de juros", economy.Coins(saved.Amount), economy.Coins(saved.Interest)))
	}
	if jailLeft > 0 {
		lines = append(lines, "Tá preso, sai em "+core.FormatWait(jailLeft))
	}
	if deathLeft > 0 {
		lines = append(lines, "Tá morto, ressuscita em "+core.FormatWait(deathLeft))
	}
	lines = append(lines, fmt.Sprintf("Saúde: %d de %d", health, economy.MaxHealth))
	lines = append(lines, "Profissão: "+job.Details())
	return strings.Join(lines, "\n"), nil
}

func (k *iaToolkit) richestPeople(ctx context.Context, _ json.RawMessage) (string, error) {
	if !k.b.economy.EconomyOn(ctx, k.chat) {
		return "", errEconomyOff
	}
	wallets, err := k.b.economy.Wallets.Richest(ctx, k.chat, economy.RichestCount)
	if err != nil {
		return "", err
	}
	if len(wallets) == 0 {
		return "Ninguém tem moeda ainda nesta conversa.", nil
	}
	members, err := k.members()
	if err != nil {
		return "", err
	}
	lines := make([]string, len(wallets))
	for i, w := range wallets {
		lines[i] = fmt.Sprintf("%dº %s: %s", i+1, k.walletOwnerName(ctx, members, w.Owner), economy.Coins(w.Balance))
	}
	return strings.Join(lines, "\n"), nil
}

func (k *iaToolkit) walletOwnerName(ctx context.Context, members []iaMember, owner types.JID) string {
	i := slices.IndexFunc(members, func(m iaMember) bool { return slices.Contains(m.users, owner.User) })
	if i >= 0 {
		return members[i].name
	}
	return k.b.DisplayName(ctx, owner, k.nicknames) + " (não tá mais no grupo)"
}
