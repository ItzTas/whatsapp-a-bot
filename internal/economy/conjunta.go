package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

var errBadAmount = errors.New("valor inválido")

type jointMove struct {
	run     func(ctx context.Context, chat types.JID, self, spouse core.Account, arg string) (amount, balance int64, err error)
	done    string
	noFunds string
}

func (b *bot) conjunta(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	var (
		marriages []core.Relationship
		aliases   []string
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { marriages, err = b.Relationships.InChat(gctx, chat, core.KindMarriage); return err })
	g.Go(func() error { aliases = b.UserAliases(gctx, person); return nil })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler casamentos: %w", err)
	}
	spouses := core.OtherPartners(core.ConsentedOnly(marriages), aliases, nil)
	if len(spouses) == 0 {
		return b.SendText(ctx, chat, "Conta conjunta é só pra quem casou com consentimento 💍 Com o modo consentimento ligado, faz o pedido com !casar @pessoa")
	}
	self := b.Account(ctx, person)
	args := core.PlainArgs(core.MessageText(evt.Message))
	if len(args) == 0 {
		return b.jointStatement(ctx, chat, self, spouses)
	}

	usageText := "Use assim: !conjunta depositar 100 ou !conjunta sacar 100 (aceita tudo, metade e 10%). Se for casado(a) com mais de uma pessoa, marca o cônjuge"
	moves := map[string]jointMove{
		"depositar": {b.depositJoint, "🏦 @%s guardou %s na conta conjunta com @%s, que agora tem %s", "@%s não tem isso tudo na carteira 🪹"},
		"sacar":     {b.withdrawJoint, "🏦 @%s tirou %s da conta conjunta com @%s, que agora tem %s", "A conta conjunta de @%s não tem isso tudo 🪹"},
	}
	moves["guardar"], moves["tirar"] = moves["depositar"], moves["sacar"]
	move, ok := moves[args[0]]
	if !ok || len(args) < 2 {
		return b.SendText(ctx, chat, usageText)
	}
	spouse, ok := b.pickSpouse(ctx, spouses, core.MentionedJIDs(evt.Message))
	if !ok {
		return b.SendText(ctx, chat, usageText)
	}

	amount, balance, err := move.run(ctx, chat, self, spouse, args[1])
	if errors.Is(err, errBadAmount) {
		return b.SendText(ctx, chat, usageText)
	}
	if errors.Is(err, ErrNoFunds) {
		_, err := b.SendMentions(ctx, chat, fmt.Sprintf(move.noFunds, person.User), []types.JID{person})
		return err
	}
	if err != nil {
		return fmt.Errorf("mexer na conta conjunta: %w", err)
	}
	text := fmt.Sprintf(move.done, person.User, Coins(amount), spouse.JID.User, Coins(balance))
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person, spouse.JID})
	return err
}

func (b *bot) pickSpouse(ctx context.Context, spouses, mentions []types.JID) (core.Account, bool) {
	accounts := b.accounts(ctx, spouses)
	if len(mentions) == 0 {
		return accounts[0], len(accounts) == 1
	}
	wanted := b.Account(ctx, mentions[0])
	i := slices.IndexFunc(accounts, func(a core.Account) bool { return a.Key == wanted.Key })
	if i < 0 {
		return core.Account{}, false
	}
	return accounts[i], true
}

func (b *bot) jointStatement(ctx context.Context, chat types.JID, self core.Account, spouses []types.JID) error {
	balances, err := b.Wallets.jointBalances(ctx, chat, self)
	if err != nil {
		return fmt.Errorf("ler contas conjuntas: %w", err)
	}
	lines := []string{fmt.Sprintf("🏦 *Contas conjuntas de @%s*", self.JID.User), ""}
	for _, spouse := range b.accounts(ctx, spouses) {
		lines = append(lines, fmt.Sprintf("💍 Com @%s: %s", spouse.JID.User, Coins(balances[spouse.Key])))
	}
	lines = append(lines, "", "_Pra mexer: !conjunta depositar 100 ou !conjunta sacar 100. No divórcio, o que tiver lá é dividido meio a meio_")
	_, err = b.SendMentions(ctx, chat, strings.Join(lines, "\n"), append([]types.JID{self.JID}, spouses...))
	return err
}

func (b *bot) depositJoint(ctx context.Context, chat types.JID, self, spouse core.Account, arg string) (int64, int64, error) {
	funds, _, err := b.Wallets.Balance(ctx, chat, self)
	if err != nil {
		return 0, 0, err
	}
	amount, ok := ParseStake(arg, funds)
	if !ok {
		return 0, 0, errBadAmount
	}
	balance, err := b.Wallets.depositJoint(ctx, chat, self, spouse, amount)
	return amount, balance, err
}

func (b *bot) withdrawJoint(ctx context.Context, chat types.JID, self, spouse core.Account, arg string) (int64, int64, error) {
	balances, err := b.Wallets.jointBalances(ctx, chat, self)
	if err != nil {
		return 0, 0, err
	}
	amount, ok := ParseStake(arg, balances[spouse.Key])
	if !ok {
		return 0, 0, errBadAmount
	}
	balance, err := b.Wallets.withdrawJoint(ctx, chat, self, spouse, amount)
	return amount, balance, err
}

func (b *bot) SplitJointAccounts(ctx context.Context, chat types.JID, marriages []core.Relationship) string {
	var lines []string
	for _, m := range marriages {
		total, err := b.Wallets.splitJoint(ctx, chat, b.Account(ctx, m.Partners[0]), b.Account(ctx, m.Partners[1]))
		if err != nil {
			b.Client.Log.Errorf("dividir conta conjunta de %s e %s: %v", m.Partners[0], m.Partners[1], err)
			continue
		}
		if total == 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf("🏦 A conta conjunta de @%s e @%s tinha %s e foi dividida meio a meio", m.Partners[0].User, m.Partners[1].User, Coins(total)))
	}
	return strings.Join(lines, "\n")
}

func jointPair(one, other core.Account) (string, string) {
	return min(one.Key, other.Key), max(one.Key, other.Key)
}

func (s *walletStore) jointBalances(ctx context.Context, chat types.JID, acc core.Account) (balances map[string]int64, err error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT CASE WHEN pessoa1 = ? THEN pessoa2 ELSE pessoa1 END, saldo
		 FROM contas_conjuntas WHERE chat = ? AND ? IN (pessoa1, pessoa2)`,
		acc.Key, chat.String(), acc.Key,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	balances = map[string]int64{}
	for rows.Next() {
		var (
			partner string
			balance int64
		)
		if err := rows.Scan(&partner, &balance); err != nil {
			return nil, err
		}
		balances[partner] = balance
	}
	return balances, rows.Err()
}

func (s *walletStore) depositJoint(ctx context.Context, chat types.JID, from, spouse core.Account, amount int64) (balance int64, err error) {
	first, second := jointPair(from, spouse)
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		if err := debitBalance(ctx, tx, chat, from, amount); err != nil {
			return err
		}
		return tx.QueryRowContext(ctx,
			`INSERT INTO contas_conjuntas (chat, pessoa1, pessoa2, saldo) VALUES (?, ?, ?, ?)
			 ON CONFLICT (chat, pessoa1, pessoa2) DO UPDATE SET saldo = saldo + excluded.saldo
			 RETURNING saldo`,
			chat.String(), first, second, amount,
		).Scan(&balance)
	})
	return balance, err
}

func (s *walletStore) withdrawJoint(ctx context.Context, chat types.JID, to, spouse core.Account, amount int64) (balance int64, err error) {
	first, second := jointPair(to, spouse)
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx,
			"UPDATE contas_conjuntas SET saldo = saldo - ? WHERE chat = ? AND pessoa1 = ? AND pessoa2 = ? AND saldo >= ? RETURNING saldo",
			amount, chat.String(), first, second, amount,
		).Scan(&balance)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoFunds
		}
		if err != nil {
			return err
		}
		if err := openWallet(ctx, tx, chat, to); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", amount, chat.String(), to.Key)
		return err
	})
	return balance, err
}

func (s *walletStore) splitJoint(ctx context.Context, chat types.JID, one, other core.Account) (total int64, err error) {
	first, second := jointPair(one, other)
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx,
			"DELETE FROM contas_conjuntas WHERE chat = ? AND pessoa1 = ? AND pessoa2 = ? RETURNING saldo",
			chat.String(), first, second,
		).Scan(&total)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		shares := []struct {
			acc    core.Account
			amount int64
		}{{one, total - total/2}, {other, total / 2}}
		for _, share := range shares {
			if err := openWallet(ctx, tx, chat, share.acc); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", share.amount, chat.String(), share.acc.Key)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return total, err
}
