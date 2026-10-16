package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const miracleMax = 1_000_000

type miracleGift struct {
	label  string
	bestow func(ctx context.Context, s *walletStore, chat types.JID, accounts []account) error
}

func (b *bot) milagre(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return b.sendText(ctx, chat, "⚡ Milagre é coisa de grupo, aqui só tem você")
	}
	gift, problem := parseMiracleGift(parseItemQuery(messageText(evt.Message)))
	if problem != "" {
		return b.sendText(ctx, chat, problem)
	}
	god := evt.Info.Sender.ToNonAD()
	godAcc := b.account(ctx, god)

	var (
		job   profession
		group *types.GroupInfo
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { job, err = b.professionOf(gctx, chat, godAcc); return err })
	g.Go(func() (err error) { group, err = b.client.GetGroupInfo(gctx, chat); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("preparar milagre: %w", err)
	}
	if !job.divine {
		return b.sendText(ctx, chat, "🚫 Só deus faz milagre. Você, no máximo, faz um !pix")
	}

	believers := b.believers(ctx, group, godAcc)
	if len(believers) == 0 {
		return b.sendText(ctx, chat, "⚡ Não tem ninguém no grupo pra receber o milagre além de você")
	}
	if err := gift.bestow(ctx, b.wallets, chat, believers); err != nil {
		return fmt.Errorf("milagre: %w", err)
	}
	_, err := b.sendMentions(ctx, chat, fmt.Sprintf("⚡ *Milagre!* @%s fez chover %s pra cada um, e %s receberam a bênção 🌧️🙏",
		god.User, gift.label, countLabel(len(believers), "pessoa", "pessoas")), []types.JID{god})
	return err
}

func parseMiracleGift(q itemQuery) (miracleGift, string) {
	usage := fmt.Sprintf("Use assim: !milagre 500 (até %s pra cada um) ou !milagre diamante 5 (até %d de cada item pra cada um)", coins(miracleMax), creationMax)
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
		bestow: func(ctx context.Context, s *walletStore, chat types.JID, accounts []account) error {
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
		label: coins(amount),
		bestow: func(ctx context.Context, s *walletStore, chat types.JID, accounts []account) error {
			return s.credit(ctx, chat, accounts, amount)
		},
	}, ""
}

func (s *walletStore) bestowItem(ctx context.Context, chat types.JID, accounts []account, id string, quantity int64) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		for _, acc := range accounts {
			if err := addItems(ctx, tx, chat, acc, id, quantity); err != nil {
				return err
			}
		}
		return nil
	})
}

func (b *bot) believers(ctx context.Context, group *types.GroupInfo, god account) []account {
	seen := map[string]bool{god.key: true}
	believers := make([]account, 0, len(group.Participants))
	for _, p := range group.Participants {
		acc := b.account(ctx, p.JID)
		if seen[acc.key] {
			continue
		}
		seen[acc.key] = true
		believers = append(believers, acc)
	}
	return believers
}
