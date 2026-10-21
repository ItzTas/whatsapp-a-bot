package economy

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

const (
	loadedCoinID      = "moeda-viciada"
	loadedCoinFavor   = 8
	loadedCoinLuck    = 10
	loadedCoinCaught  = 0.08
	loadedCoinLossMin = 10
	loadedCoinLossMax = 20
	loadedCoinDaily   = 1
)

var errDailyCap = errors.New("limite de compra do dia")

func claimDaily(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, it item, quantity int64, now time.Time) error {
	if it.daily == 0 {
		return nil
	}
	if quantity > it.daily {
		return errDailyCap
	}
	day := blackMarketDay(now)
	if _, err := tx.ExecContext(ctx, "DELETE FROM compras_diarias WHERE chat = ? AND pessoa = ? AND dia <> ?", chat.String(), acc.Key, day); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx,
		`INSERT INTO compras_diarias (chat, dia, pessoa, item, quantidade) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (chat, dia, pessoa, item) DO UPDATE SET quantidade = quantidade + excluded.quantidade
		WHERE quantidade + excluded.quantidade <= ?`,
		chat.String(), day, acc.Key, it.id, quantity, it.daily,
	)
	if err != nil {
		return err
	}
	claimed, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if claimed == 0 {
		return errDailyCap
	}
	return nil
}

func dailyCapText(it item) string {
	return fmt.Sprintf("O contato só passa %s por pessoa a cada dia, pra não dar na vista. Volta amanhã", quantityLabel(it, it.daily))
}

func dailyCapTag(it item) string {
	if it.daily == 0 {
		return ""
	}
	return fmt.Sprintf(", até %d por dia", it.daily)
}

type cheatBust struct {
	payback
	burned int64
}

func (b *bot) bustCheat(ctx context.Context, chat types.JID, acc core.Account, job Profession, syntax string) (bool, error) {
	if job.cheat == 0 || rand.Float64() >= job.flagrantOdds(loadedCoinCaught) {
		return false, nil
	}
	c, err := b.Wallets.bustCheat(ctx, chat, acc, job.snitchSentence(), time.Now())
	if err != nil {
		return false, fmt.Errorf("pegar a moeda viciada: %w", err)
	}
	rescue, freed := arrest{freed: c.freed}.rescue()
	_, err = b.SendMentions(ctx, chat, c.text(acc.JID, syntax)+rescue, append([]types.JID{acc.JID}, freed...))
	return true, err
}

func (s *walletStore) bustCheat(ctx context.Context, chat types.JID, acc core.Account, jail time.Duration, now time.Time) (c cheatBust, err error) {
	killed := rand.IntN(2) == 0
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		if _, err := spendItem(ctx, tx, chat, acc, loadedCoinID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE carteiras SET ficha_em = ? WHERE chat = ? AND pessoa = ?", now.Unix(), chat.String(), acc.Key); err != nil {
			return err
		}
		if c.burned, err = burnBalance(ctx, tx, chat, acc, loadedCoinLossMin, loadedCoinLossMax, 1); err != nil {
			return err
		}
		if killed {
			c.payback, err = killSnitch(ctx, tx, chat, acc, now)
			return err
		}
		c.payback, err = jailSnitch(ctx, tx, chat, acc, jail, now)
		return err
	})
	return c, err
}

func (c cheatBust) text(person types.JID, syntax string) string {
	text := fmt.Sprintf("🚨 Os seguranças do cassino pegaram @%s apostando no %s com uma moeda viciada! A aposta não valeu e a moeda foi apreendida", person.User, syntax)
	if c.burned > 0 {
		text += fmt.Sprintf("\n💸 O cassino ainda tomou %s de @%s pelo prejuízo", Coins(c.burned), person.User)
	}
	if c.fate == snitchJailed {
		return text + fmt.Sprintf("\n🚔 Depois chamaram a polícia: @%s foi preso e só sai em %s", person.User, core.FormatWait(c.left)) + fineText(c.fine)
	}
	text += fmt.Sprintf("\n💀 Depois levaram @%s pros fundos e só voltou o chinelo: morreu e fica %s sem comando de economia, a não ser que pague o !hospital ou alguém consiga !reviver", person.User, core.FormatWait(c.left))
	if funeral := funeralText(person, c.lost); funeral != "" {
		text += "\n" + funeral
	}
	if c.deposed {
		text += "\n" + deposedText(person)
	}
	return text
}
