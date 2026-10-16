package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

func (b *bot) castigo(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return b.sendText(ctx, chat, "🔥 Castigo é coisa de grupo, aqui só tem você")
	}
	arg := firstPlainArg(messageText(evt.Message))
	if _, ok := parseStake(arg, 100); !ok {
		return b.sendText(ctx, chat, "Use assim: !castigo 500, !castigo 10%, !castigo metade ou !castigo tudo (e marque gente pra castigar só ela)")
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
		return fmt.Errorf("preparar castigo: %w", err)
	}
	if !job.divine {
		return b.sendText(ctx, chat, "🚫 Só deus castiga. Você, no máximo, faz um !roubar")
	}

	marked := mentionedJIDs(evt.Message)
	sinners := b.believers(ctx, group, godAcc)
	if len(marked) > 0 {
		sinners = b.accountsOf(ctx, marked)
	}
	if len(sinners) == 0 {
		return b.sendText(ctx, chat, "🔥 Não tem ninguém no grupo pra castigar além de você")
	}

	total, punished, err := b.wallets.smite(ctx, chat, sinners, arg)
	if err != nil {
		return fmt.Errorf("castigo: %w", err)
	}
	mentions := append([]types.JID{god}, marked...)
	if punished == 0 {
		_, err = b.sendMentions(ctx, chat, fmt.Sprintf("🔥 @%s mandou o castigo, mas ninguém tinha moeda pra perder", god.User), mentions)
		return err
	}
	_, err = b.sendMentions(ctx, chat, fmt.Sprintf("🔥 *Castigo divino!* @%s fez sumir %s de %s ⚡💀",
		god.User, coins(total), countLabel(punished, "pessoa", "pessoas")), mentions)
	return err
}

func (b *bot) accountsOf(ctx context.Context, people []types.JID) []account {
	seen := map[string]bool{}
	accounts := make([]account, 0, len(people))
	for _, person := range people {
		acc := b.account(ctx, person)
		if seen[acc.key] {
			continue
		}
		seen[acc.key] = true
		accounts = append(accounts, acc)
	}
	return accounts
}

func (s *walletStore) smite(ctx context.Context, chat types.JID, accounts []account, arg string) (total int64, punished int, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		for _, acc := range accounts {
			var balance int64
			err := tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.key).Scan(&balance)
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			if err != nil {
				return err
			}
			amount, ok := parseStake(arg, balance)
			amount = min(amount, balance)
			if !ok || amount == 0 {
				continue
			}
			_, err = tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo - ? WHERE chat = ? AND pessoa = ?", amount, chat.String(), acc.key)
			if err != nil {
				return err
			}
			total += amount
			punished++
		}
		return nil
	})
	return total, punished, err
}
