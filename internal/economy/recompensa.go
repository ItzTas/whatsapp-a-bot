package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const bountyBoardSize = 10

type bounty struct {
	target types.JID
	total  int64
}

func (b *bot) recompensa(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	mentions := core.MentionedJIDs(evt.Message)
	if len(mentions) == 0 {
		return b.bountyBoard(ctx, chat)
	}
	poster, target := evt.Info.Sender.ToNonAD(), mentions[0].ToNonAD()
	posterAcc, targetAcc := b.Account(ctx, poster), b.Account(ctx, target)
	arg := core.FirstPlainArg(core.MessageText(evt.Message))
	if slices.Contains([]string{"retirar", "cancelar", "tirar"}, arg) {
		return b.withdrawBounty(ctx, chat, poster, target, posterAcc, targetAcc)
	}
	if arg == "" {
		return b.bountyOn(ctx, chat, target, targetAcc)
	}
	if posterAcc.Key == targetAcc.Key {
		return b.SendText(ctx, chat, "Botar recompensa na própria cabeça? Se quer tanto assim morrer, pelo menos não paga por isso 🤨")
	}

	balance, _, err := b.Wallets.Balance(ctx, chat, posterAcc)
	if err != nil {
		return fmt.Errorf("ler saldo: %w", err)
	}
	amount, ok := ParseStake(arg, balance)
	if !ok {
		return b.SendText(ctx, chat, "Não entendi o valor. Use assim: !recompensa @pessoa 100 (ou metade, 10%, tudo)")
	}
	total, err := b.Wallets.postBounty(ctx, chat, posterAcc, targetAcc, amount, time.Now())
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, chat, "Você só tem "+Coins(balance))
	}
	if err != nil {
		return fmt.Errorf("botar recompensa: %w", err)
	}
	text := fmt.Sprintf("🎯 @%s botou %s na cabeça de @%s! A recompensa agora tá em %s e vai pra quem conseguir !matar @%s",
		poster.User, Coins(amount), target.User, Coins(total), target.User)
	_, err = b.SendMentions(ctx, chat, text, []types.JID{poster, target})
	return err
}

func (b *bot) bountyBoard(ctx context.Context, chat types.JID) error {
	var (
		board    []bounty
		daily    wantedPost
		hasDaily bool
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { board, err = b.Wallets.bounties(gctx, chat, bountyBoardSize); return err })
	g.Go(func() (err error) { daily, hasDaily, err = b.Wallets.wanted(gctx, chat); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler recompensas: %w", err)
	}
	if len(board) == 0 {
		return b.SendText(ctx, chat, "Ninguém tem a cabeça a prêmio por aqui. Bota uma com !recompensa @pessoa 100")
	}
	lines := []string{"🎯 *Procurados*", ""}
	people := make([]types.JID, 0, len(board)+1)
	for i, bt := range board {
		lines = append(lines, fmt.Sprintf("%d. @%s: %s", i+1, bt.target.User, Coins(bt.total)))
		people = append(people, bt.target)
	}
	if hasDaily {
		lines = append(lines, "", fmt.Sprintf("🤖 Procurado do dia: @%s, com %s do bot na cabeça", daily.target.User, Coins(daily.prize)))
		people = append(people, daily.target)
	}
	lines = append(lines, "", "Quem conseguir !matar leva o prêmio")
	_, err := b.SendMentions(ctx, chat, strings.Join(lines, "\n"), people)
	return err
}

func (b *bot) bountyOn(ctx context.Context, chat, target types.JID, targetAcc core.Account) error {
	total, err := b.Wallets.bountyTotal(ctx, chat, targetAcc)
	if err != nil {
		return fmt.Errorf("ler recompensa: %w", err)
	}
	text := fmt.Sprintf("🎯 A cabeça de @%s vale %s. Quem conseguir !matar leva", target.User, Coins(total))
	if total == 0 {
		text = fmt.Sprintf("😇 Ninguém botou recompensa na cabeça de @%s. Pra botar: !recompensa @%s 100", target.User, target.User)
	}
	_, err = b.SendMentions(ctx, chat, text, []types.JID{target})
	return err
}

func (b *bot) withdrawBounty(ctx context.Context, chat, poster, target types.JID, posterAcc, targetAcc core.Account) error {
	refund, err := b.Wallets.withdrawBounty(ctx, chat, posterAcc, targetAcc)
	if err != nil {
		return fmt.Errorf("retirar recompensa: %w", err)
	}
	text := fmt.Sprintf("🕊️ @%s retirou a recompensa que tinha botado na cabeça de @%s e pegou %s de volta", poster.User, target.User, Coins(refund))
	if refund == 0 {
		text = fmt.Sprintf("Você não botou recompensa nenhuma na cabeça de @%s", target.User)
	}
	_, err = b.SendMentions(ctx, chat, text, []types.JID{poster, target})
	return err
}

func (s *walletStore) postBounty(ctx context.Context, chat types.JID, poster, target core.Account, amount int64, now time.Time) (total int64, err error) {
	err = s.spend(ctx, chat, poster, amount, func(tx *ledgerTx) error {
		if err := openWallet(ctx, tx, chat, target); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx,
			`INSERT INTO recompensas (chat, alvo, autor, valor, em) VALUES (?, ?, ?, ?, ?)
			 ON CONFLICT (chat, alvo, autor) DO UPDATE SET valor = valor + excluded.valor, em = excluded.em`,
			chat.String(), target.Key, poster.Key, amount, now.Unix(),
		)
		if err != nil {
			return err
		}
		total, err = bountyTotal(ctx, tx, chat, target)
		return err
	})
	return total, err
}

func (s *walletStore) withdrawBounty(ctx context.Context, chat types.JID, poster, target core.Account) (refund int64, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		err := tx.QueryRowContext(ctx,
			"DELETE FROM recompensas WHERE chat = ? AND alvo = ? AND autor = ? RETURNING valor",
			chat.String(), target.Key, poster.Key,
		).Scan(&refund)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", refund, chat.String(), poster.Key)
		return err
	})
	return refund, err
}

func (s *walletStore) bountyTotal(ctx context.Context, chat types.JID, target core.Account) (int64, error) {
	return bountyTotal(ctx, s.db, chat, target)
}

func (s *walletStore) bounties(ctx context.Context, chat types.JID, limit int) (board []bounty, err error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT c.jid, SUM(r.valor) AS total FROM recompensas r
		 JOIN carteiras c ON c.chat = r.chat AND c.pessoa = r.alvo
		 WHERE r.chat = ? GROUP BY r.alvo ORDER BY total DESC LIMIT ?`,
		chat.String(), limit,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var (
			bt     bounty
			target string
		)
		if err := rows.Scan(&target, &bt.total); err != nil {
			return nil, err
		}
		if bt.target, err = types.ParseJID(target); err != nil {
			return nil, err
		}
		board = append(board, bt)
	}
	return board, rows.Err()
}

type rowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func bountyTotal(ctx context.Context, q rowQuerier, chat types.JID, target core.Account) (total int64, err error) {
	err = q.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(valor), 0) FROM recompensas WHERE chat = ? AND alvo = ?",
		chat.String(), target.Key,
	).Scan(&total)
	return total, err
}

func claimBounty(ctx context.Context, tx *ledgerTx, chat types.JID, victim, killer core.Account) (int64, error) {
	total, err := bountyTotal(ctx, tx, chat, victim)
	if err != nil || total == 0 {
		return 0, err
	}
	_, err = tx.ExecContext(ctx, "DELETE FROM recompensas WHERE chat = ? AND alvo = ?", chat.String(), victim.Key)
	if err != nil {
		return 0, err
	}
	_, err = tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", total, chat.String(), killer.Key)
	return total, err
}
