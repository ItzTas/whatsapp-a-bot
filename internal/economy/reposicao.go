package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"sync"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

const (
	restockEvery  = 12 * time.Hour
	restockSyntax = "!decreto repor"
)

var restockNames = []string{"repor", "reposicao", "reabastecer", "estoque", "lojas", "atualizarlojas"}

type restockBook struct {
	mu   sync.RWMutex
	last map[types.JID]time.Time
}

var restocks = &restockBook{last: map[types.JID]time.Time{}}

func (r *restockBook) load(ctx context.Context, db *sql.DB) (err error) {
	rows, err := db.QueryContext(ctx, "SELECT chat, reposta_em FROM lojas_reposicoes")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	last := map[types.JID]time.Time{}
	for rows.Next() {
		var chat types.JID
		var at int64
		if err := rows.Scan(&chat, &at); err != nil {
			return err
		}
		last[chat] = time.Unix(at, 0)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.last = last
	return nil
}

func (r *restockBook) of(chat types.JID) time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.last[ledgers.of(chat)]
}

func (r *restockBook) mark(chat types.JID, at time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.last[ledgers.of(chat)] = at
}

func shopRand(chat types.JID, now time.Time, salt string) *rand.Rand {
	if at := restocks.of(chat); !at.IsZero() && localDay(at) == localDay(now) {
		salt += ":reposicao:" + strconv.FormatInt(at.Unix(), 10)
	}
	return dailyRand(chat, now, salt)
}

func restockHelp() core.Usage {
	return core.Usage{restockSyntax, fmt.Sprintf("troca na hora as ofertas do dia da !loja (as do \"Só hoje\"), da !luxo e do !mercadonegro. Depois disso, só dá pra repor de novo daqui a %s, mesmo que entre outro presidente", core.FormatWait(restockEvery))}
}

func (b *bot) restockDecree(ctx context.Context, call decreeCall) (string, error) {
	wait, over, err := b.Wallets.restockShops(ctx, call.chat, call.p.president, call.now)
	if err != nil {
		return "", fmt.Errorf("decreto repor: %w", err)
	}
	if over {
		return termOverText, nil
	}
	if wait > 0 {
		return fmt.Sprintf("📦 As lojas foram repostas faz pouco tempo. A próxima reposição só sai daqui a %s, não importa quem estiver no cargo", core.FormatWait(wait)), nil
	}
	return fmt.Sprintf("📦 *Decreto presidencial!* @%s mandou repor as lojas: a !loja (no \"Só hoje\"), a !luxo e o !mercadonegro já tão com ofertas novas 🎉\n_A próxima reposição só pode sair daqui a %s, mesmo que mude o presidente_",
		call.p.president.JID.User, core.FormatWait(restockEvery)), nil
}

func (s *walletStore) restockShops(ctx context.Context, chat types.JID, president core.Account, now time.Time) (wait time.Duration, over bool, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		var last int64
		err := tx.QueryRowContext(ctx,
			"SELECT COALESCE((SELECT reposta_em FROM lojas_reposicoes WHERE chat = ?), 0) FROM presidencias WHERE chat = ? AND pessoa = ? AND mandato_ate > ?",
			chat.String(), chat.String(), president.Key, now.Unix(),
		).Scan(&last)
		if err != nil {
			return err
		}
		if wait = time.Unix(last, 0).Add(restockEvery).Sub(now); wait > 0 {
			return nil
		}
		_, err = tx.ExecContext(ctx,
			`INSERT INTO lojas_reposicoes (chat, reposta_em) VALUES (?, ?)
			 ON CONFLICT (chat) DO UPDATE SET reposta_em = excluded.reposta_em`,
			chat.String(), now.Unix(),
		)
		return err
	})
	if errors.Is(err, sql.ErrNoRows) {
		return 0, true, nil
	}
	if err != nil || wait > 0 {
		return wait, false, err
	}
	restocks.mark(chat, now)
	return 0, false, nil
}
