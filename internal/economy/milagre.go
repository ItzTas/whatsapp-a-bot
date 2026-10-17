package economy

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const miracleMax = 1_000_000

type miracleGift struct {
	label  string
	bestow func(ctx context.Context, s *walletStore, chat types.JID, accounts []core.Account) error
}

func (b *bot) milagre(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return b.SendText(ctx, chat, "⚡ Milagre é coisa de grupo, aqui só tem você")
	}
	gift, problem := parseMiracleGift(parseItemQuery(core.MessageText(evt.Message)))
	if problem != "" {
		return b.SendText(ctx, chat, problem)
	}
	god := evt.Info.Sender.ToNonAD()
	godAcc := b.Account(ctx, god)

	var (
		job   Profession
		group *types.GroupInfo
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { job, err = b.ProfessionOf(gctx, chat, godAcc); return err })
	g.Go(func() (err error) { group, err = b.Client.GetGroupInfo(gctx, chat); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("preparar milagre: %w", err)
	}
	if !job.divine {
		return b.SendText(ctx, chat, "🚫 Só deus faz milagre. Você, no máximo, faz um !pix")
	}

	marked := core.MentionedJIDs(evt.Message)
	if len(marked) > 0 {
		return b.chosenMiracle(ctx, chat, god, godAcc, gift, marked)
	}
	believers := b.believers(ctx, group, godAcc)
	if len(believers) == 0 {
		return b.SendText(ctx, chat, "⚡ Não tem ninguém no grupo pra receber o milagre além de você")
	}
	if err := gift.bestow(ctx, b.Wallets, chat, believers); err != nil {
		return fmt.Errorf("milagre: %w", err)
	}
	_, err := b.SendMentions(ctx, chat, fmt.Sprintf("⚡ *Milagre!* @%s fez chover %s pra cada um, e %s receberam a bênção 🌧️🙏",
		god.User, gift.label, core.CountLabel(len(believers), "pessoa", "pessoas")), []types.JID{god})
	return err
}

func (b *bot) chosenMiracle(ctx context.Context, chat, god types.JID, godAcc core.Account, gift miracleGift, marked []types.JID) error {
	chosen := slices.DeleteFunc(b.accountsOf(ctx, marked), func(acc core.Account) bool { return acc.Key == godAcc.Key })
	if len(chosen) == 0 {
		return b.SendText(ctx, chat, "⚡ Milagre pra si mesmo não rola, deus. Pra isso tem o !criar")
	}
	if err := gift.bestow(ctx, b.Wallets, chat, chosen); err != nil {
		return fmt.Errorf("milagre: %w", err)
	}
	_, err := b.SendMentions(ctx, chat, fmt.Sprintf("⚡ *Milagre!* @%s fez chover %s pra %s 🌧️🙏",
		god.User, gift.label, mentionList(chosen)), append([]types.JID{god}, accountJIDs(chosen)...))
	return err
}

func parseMiracleGift(q itemQuery) (miracleGift, string) {
	usage := fmt.Sprintf("Use assim: !milagre 500 (até %s pra cada um) ou !milagre diamante 5 (até %d de cada item pra cada um)", Coins(miracleMax), creationMax)
	if q.name == "" {
		return coinMiracle(q, usage)
	}
	matches := findItems(q.name)
	if problem := itemQueryProblem(q.name, matches); problem != "" {
		return miracleGift{}, problem
	}
	it := matches[0]
	quantity := q.quantity(0)
	if quantity <= 0 || quantity > creationMax {
		return miracleGift{}, fmt.Sprintf("Diz quantos cada um ganha, até %d: !milagre %s 3", creationMax, strings.ToLower(it.name))
	}
	return miracleGift{
		label: quantityLabel(it, quantity),
		bestow: func(ctx context.Context, s *walletStore, chat types.JID, accounts []core.Account) error {
			return s.bestowItem(ctx, chat, accounts, it.id, quantity)
		},
	}, ""
}

func coinMiracle(q itemQuery, usage string) (miracleGift, string) {
	if len(q.numbers) == 0 || q.numbers[0] > miracleMax {
		return miracleGift{}, usage
	}
	amount := q.numbers[0]
	return miracleGift{
		label: Coins(amount),
		bestow: func(ctx context.Context, s *walletStore, chat types.JID, accounts []core.Account) error {
			return s.Credit(ctx, chat, accounts, amount)
		},
	}, ""
}

func (s *walletStore) bestowItem(ctx context.Context, chat types.JID, accounts []core.Account, id string, quantity int64) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		for _, acc := range accounts {
			if err := addItems(ctx, tx, chat, acc, id, quantity); err != nil {
				return err
			}
		}
		return nil
	})
}

func (b *bot) believers(ctx context.Context, group *types.GroupInfo, god core.Account) []core.Account {
	seen := map[string]bool{god.Key: true}
	believers := make([]core.Account, 0, len(group.Participants))
	for _, p := range group.Participants {
		acc := b.Account(ctx, p.JID)
		if seen[acc.Key] {
			continue
		}
		seen[acc.Key] = true
		believers = append(believers, acc)
	}
	return believers
}
