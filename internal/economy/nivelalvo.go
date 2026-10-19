package economy

import (
	"context"
	"fmt"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"golang.org/x/sync/errgroup"
)

const (
	markLevelMax   = 5
	levelRobStep   = 0.05
	levelEscortCut = 0.15
)

type markLevel struct {
	jackpot int64
	level   int
	wagered int64
}

func markLevelOf(ctx context.Context, q rowQuerier, chat types.JID, acc core.Account, now time.Time) (m markLevel, err error) {
	if m.jackpot, err = freshJackpot(ctx, q, chat, acc, now); err != nil || m.jackpot == 0 {
		return m, err
	}
	err = q.QueryRowContext(ctx, "SELECT alvo_nivel, alvo_apostado FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&m.level, &m.wagered)
	m.level = max(m.level, 1)
	return m, err
}

func (m markLevel) after(stake int64) markLevel {
	if m.jackpot == 0 || m.level >= markLevelMax {
		return m
	}
	m.wagered += stake
	m.level = min(markLevelMax, m.level+int(m.wagered/m.jackpot))
	m.wagered %= m.jackpot
	return m
}

func (s *walletStore) raiseMark(ctx context.Context, chat types.JID, acc core.Account, stake int64, now time.Time) (from, to int, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		m, err := markLevelOf(ctx, tx, chat, acc, now)
		if err != nil || m.jackpot == 0 {
			return err
		}
		next := m.after(stake)
		from, to = m.level, next.level
		_, err = tx.ExecContext(ctx, "UPDATE carteiras SET alvo_nivel = ?, alvo_apostado = ? WHERE chat = ? AND pessoa = ?", next.level, next.wagered, chat.String(), acc.Key)
		return err
	})
	return from, to, err
}

func resetMarkLevel(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account) error {
	_, err := tx.ExecContext(ctx, "UPDATE carteiras SET alvo_nivel = 1, alvo_apostado = 0 WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key)
	return err
}

func (b *bot) noteMarkLevel(ctx context.Context, chat types.JID, acc core.Account, stake int64) {
	from, to, err := b.Wallets.raiseMark(ctx, chat, acc, stake, time.Now())
	if err != nil {
		b.Client.Log.Warnf("subir o nível de alvo fácil de %s: %v", acc.JID, err)
		return
	}
	if to <= from {
		return
	}
	text := fmt.Sprintf("🎯 @%s, você continuou apostando com o prêmio no bolso e subiu pro *nível %d de alvo fácil*: o !roubar e a !quadrilha contra você ficam ainda mais fáceis, e os eventos de quem ganhou muito te acham mais vezes",
		acc.JID.User, to)
	if _, err := b.SendMentions(ctx, chat, text, []types.JID{acc.JID}); err != nil {
		b.Client.Log.Errorf("avisar o nível de alvo fácil de %s: %v", acc.JID, err)
	}
}

func (b *bot) markWarning(ctx context.Context, chat types.JID, acc core.Account, syntax string, stake int64) (string, error) {
	now := time.Now()
	var (
		m     markLevel
		c     contract
		hired bool
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { m, err = markLevelOf(gctx, b.Wallets.db, chat, acc, now); return err })
	g.Go(func() (err error) { c, hired, err = contractOf(gctx, b.Wallets.db, chat, acc, now); return err })
	if err := g.Wait(); err != nil {
		return "", err
	}
	next := m.after(stake)
	if !hired || next.level <= m.level || b.markWarnings.confirmed(eventKey(chat, acc)+"|"+syntax, now) {
		return "", nil
	}
	return fmt.Sprintf("⚠️ A %s avisou: apostar %s agora te leva pro nível %d de alvo fácil, e aí fica mais difícil de te defender. Se for isso mesmo, manda o mesmo comando de novo em até %s",
		c.firm.label(), Coins(stake), next.level, core.FormatDuration(allInWindow)), nil
}

func (m easyMark) steps() int {
	return max(m.level-1, 0)
}
