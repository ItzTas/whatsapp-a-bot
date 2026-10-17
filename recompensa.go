package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const bountyBoardSize = 10

type bounty struct {
	target types.JID
	total  int64
}

func (b *bot) recompensa(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	mentions := mentionedJIDs(evt.Message)
	if len(mentions) == 0 {
		return b.bountyBoard(ctx, chat)
	}
	poster, target := evt.Info.Sender.ToNonAD(), mentions[0].ToNonAD()
	posterAcc, targetAcc := b.account(ctx, poster), b.account(ctx, target)
	arg := firstPlainArg(messageText(evt.Message))
	if slices.Contains([]string{"retirar", "cancelar", "tirar"}, arg) {
		return b.withdrawBounty(ctx, chat, poster, target, posterAcc, targetAcc)
	}
	if arg == "" {
		return b.bountyOn(ctx, chat, target, targetAcc)
	}
	if posterAcc.key == targetAcc.key {
		return b.sendText(ctx, chat, "Botar recompensa na própria cabeça? Se quer tanto assim morrer, pelo menos não paga por isso 🤨")
	}

	balance, _, err := b.wallets.balance(ctx, chat, posterAcc)
	if err != nil {
		return fmt.Errorf("ler saldo: %w", err)
	}
	amount, ok := parseStake(arg, balance)
	if !ok {
		return b.sendText(ctx, chat, "Não entendi o valor. Use assim: !recompensa @pessoa 100 (ou metade, 10%, tudo)")
	}
	total, err := b.wallets.postBounty(ctx, chat, posterAcc, targetAcc, amount, time.Now())
	if errors.Is(err, errNoFunds) {
		return b.sendText(ctx, chat, "Você só tem "+coins(balance))
	}
	if err != nil {
		return fmt.Errorf("botar recompensa: %w", err)
	}
	text := fmt.Sprintf("🎯 @%s botou %s na cabeça de @%s! A recompensa agora tá em %s e vai pra quem conseguir !matar @%s",
		poster.User, coins(amount), target.User, coins(total), target.User)
	_, err = b.sendMentions(ctx, chat, text, []types.JID{poster, target})
	return err
}

func (b *bot) bountyBoard(ctx context.Context, chat types.JID) error {
	board, err := b.wallets.bounties(ctx, chat, bountyBoardSize)
	if err != nil {
		return fmt.Errorf("ler recompensas: %w", err)
	}
	if len(board) == 0 {
		return b.sendText(ctx, chat, "Ninguém tem a cabeça a prêmio por aqui. Bota uma com !recompensa @pessoa 100")
	}
	lines := []string{"🎯 *Procurados*", ""}
	people := make([]types.JID, 0, len(board))
	for i, bt := range board {
		lines = append(lines, fmt.Sprintf("%d. @%s: %s", i+1, bt.target.User, coins(bt.total)))
		people = append(people, bt.target)
	}
	lines = append(lines, "", "Quem conseguir !matar leva o prêmio")
	_, err = b.sendMentions(ctx, chat, strings.Join(lines, "\n"), people)
	return err
}

func (b *bot) bountyOn(ctx context.Context, chat, target types.JID, targetAcc account) error {
	total, err := b.wallets.bountyTotal(ctx, chat, targetAcc)
	if err != nil {
		return fmt.Errorf("ler recompensa: %w", err)
	}
	text := fmt.Sprintf("🎯 A cabeça de @%s vale %s. Quem conseguir !matar leva", target.User, coins(total))
	if total == 0 {
		text = fmt.Sprintf("😇 Ninguém botou recompensa na cabeça de @%s. Pra botar: !recompensa @%s 100", target.User, target.User)
	}
	_, err = b.sendMentions(ctx, chat, text, []types.JID{target})
	return err
}

func (b *bot) withdrawBounty(ctx context.Context, chat, poster, target types.JID, posterAcc, targetAcc account) error {
	refund, err := b.wallets.withdrawBounty(ctx, chat, posterAcc, targetAcc)
	if err != nil {
		return fmt.Errorf("retirar recompensa: %w", err)
	}
	text := fmt.Sprintf("🕊️ @%s retirou a recompensa que tinha botado na cabeça de @%s e pegou %s de volta", poster.User, target.User, coins(refund))
	if refund == 0 {
		text = fmt.Sprintf("Você não botou recompensa nenhuma na cabeça de @%s", target.User)
	}
	_, err = b.sendMentions(ctx, chat, text, []types.JID{poster, target})
	return err
}

func (s *walletStore) postBounty(ctx context.Context, chat types.JID, poster, target account, amount int64, now time.Time) (total int64, err error) {
	err = s.spend(ctx, chat, poster, amount, func(tx *sql.Tx) error {
		if err := openWallet(ctx, tx, chat, target); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx,
			`INSERT INTO recompensas (chat, alvo, autor, valor, em) VALUES (?, ?, ?, ?, ?)
			 ON CONFLICT (chat, alvo, autor) DO UPDATE SET valor = valor + excluded.valor, em = excluded.em`,
			chat.String(), target.key, poster.key, amount, now.Unix(),
		)
		if err != nil {
			return err
		}
		total, err = bountyTotal(ctx, tx, chat, target)
		return err
	})
	return total, err
}

func (s *walletStore) withdrawBounty(ctx context.Context, chat types.JID, poster, target account) (refund int64, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx,
			"DELETE FROM recompensas WHERE chat = ? AND alvo = ? AND autor = ? RETURNING valor",
			chat.String(), target.key, poster.key,
		).Scan(&refund)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", refund, chat.String(), poster.key)
		return err
	})
	return refund, err
}

func (s *walletStore) bountyTotal(ctx context.Context, chat types.JID, target account) (int64, error) {
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

func bountyTotal(ctx context.Context, q rowQuerier, chat types.JID, target account) (total int64, err error) {
	err = q.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(valor), 0) FROM recompensas WHERE chat = ? AND alvo = ?",
		chat.String(), target.key,
	).Scan(&total)
	return total, err
}

func claimBounty(ctx context.Context, tx *sql.Tx, chat types.JID, victim, killer account) (int64, error) {
	total, err := bountyTotal(ctx, tx, chat, victim)
	if err != nil || total == 0 {
		return 0, err
	}
	_, err = tx.ExecContext(ctx, "DELETE FROM recompensas WHERE chat = ? AND alvo = ?", chat.String(), victim.key)
	if err != nil {
		return 0, err
	}
	_, err = tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", total, chat.String(), killer.key)
	return total, err
}
