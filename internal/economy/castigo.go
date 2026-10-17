package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

func (b *bot) castigo(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return b.SendText(ctx, chat, "🔥 Castigo é coisa de grupo, aqui só tem você")
	}
	arg := core.FirstPlainArg(core.MessageText(evt.Message))
	if _, ok := ParseStake(arg, 100); !ok {
		return b.SendText(ctx, chat, "Use assim: !castigo 500, !castigo 10%, !castigo metade ou !castigo tudo (e marque gente pra castigar só ela)")
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
		return fmt.Errorf("preparar castigo: %w", err)
	}
	if !job.divine {
		return b.SendText(ctx, chat, "🚫 Só deus castiga. Você, no máximo, faz um !roubar")
	}

	marked := core.MentionedJIDs(evt.Message)
	sinners := b.believers(ctx, group, godAcc)
	if len(marked) > 0 {
		sinners = b.accountsOf(ctx, marked)
	}
	if len(sinners) == 0 {
		return b.SendText(ctx, chat, "🔥 Não tem ninguém no grupo pra castigar além de você")
	}

	total, punished, err := b.Wallets.smite(ctx, chat, sinners, arg)
	if err != nil {
		return fmt.Errorf("castigo: %w", err)
	}
	mentions := append([]types.JID{god}, marked...)
	if punished == 0 {
		_, err = b.SendMentions(ctx, chat, fmt.Sprintf("🔥 @%s mandou o castigo, mas ninguém tinha moeda pra perder", god.User), mentions)
		return err
	}
	_, err = b.SendMentions(ctx, chat, fmt.Sprintf("🔥 *Castigo divino!* @%s fez sumir %s de %s ⚡💀",
		god.User, Coins(total), core.CountLabel(punished, "pessoa", "pessoas")), mentions)
	return err
}

func (b *bot) accountsOf(ctx context.Context, people []types.JID) []core.Account {
	seen := map[string]bool{}
	accounts := make([]core.Account, 0, len(people))
	for _, person := range people {
		acc := b.Account(ctx, person)
		if seen[acc.Key] {
			continue
		}
		seen[acc.Key] = true
		accounts = append(accounts, acc)
	}
	return accounts
}

func (s *walletStore) smite(ctx context.Context, chat types.JID, accounts []core.Account, arg string) (total int64, punished int, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		for _, acc := range accounts {
			var balance int64
			err := tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&balance)
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			if err != nil {
				return err
			}
			amount, ok := ParseStake(arg, balance)
			amount = min(amount, balance)
			if !ok || amount == 0 {
				continue
			}
			_, err = tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo - ? WHERE chat = ? AND pessoa = ?", amount, chat.String(), acc.Key)
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
