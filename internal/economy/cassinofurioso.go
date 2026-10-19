package economy

import (
	"context"
	"math/rand/v2"
	"slices"
	"sync"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	casinoWrathPlace  = "cassino-furioso"
	casinoWrathChance = 0.2
	casinoWrathEvery  = 30 * time.Minute
	casinoWatchPlace  = "cassino-de-olho"
	casinoWatchChance = 0.05
)

type wrathBoard struct {
	mu   sync.Mutex
	last map[types.JID]time.Time
}

func (w *wrathBoard) due(chat types.JID, now time.Time) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.last == nil {
		w.last = map[types.JID]time.Time{}
	}
	if now.Sub(w.last[chat]) < casinoWrathEvery {
		return false
	}
	w.last[chat] = now
	return true
}

func (b *bot) stirCasinoWrath(ctx context.Context, evt *events.Message, cmd core.Command) {
	games := []string{"apostar", "cassino", "blackjack", "crash", "corrida", "roleta"}
	chat := evt.Info.Chat.ToNonAD()
	if !slices.Contains(games, cmd.Name) || chat.Server != types.GroupServer {
		return
	}
	now := time.Now()
	p, found, err := b.Wallets.presidency(ctx, chat, now)
	if err != nil {
		b.Client.Log.Errorf("ler o decreto do cassino em %s: %v", chat, err)
		return
	}
	level := p.decrees.level(casinoDecree)
	if !found || level <= 0 {
		return
	}
	scale := oracleEventFactor * float64(level) / casinoDecreeMaxRaise
	if rand.Float64() < casinoWrathChance*scale && b.casinoWrath.due(chat, now) {
		b.stirEventFor(ctx, chat, p.president, b.Identities(ctx, p.president.JID), eventSpot{place: casinoWrathPlace})
	}
	acc := b.Account(ctx, evt.Info.Sender)
	if acc.Key == p.president.Key || rand.Float64() >= casinoWatchChance*scale {
		return
	}
	b.stirEventFor(ctx, chat, acc, b.SenderIdentities(ctx, evt), eventSpot{place: casinoWatchPlace})
}

type repealEffect struct{}

func repealCasino() effect { return repealEffect{} }

func (repealEffect) apply(ctx context.Context, tx *ledgerTx, run *eventRun) (string, error) {
	res, err := tx.ExecContext(ctx,
		"UPDATE presidencias SET "+casinoDecree.column+" = 0 WHERE chat = ? AND pessoa = ? AND "+casinoDecree.column+" > 0",
		run.chat.String(), run.acc.Key,
	)
	if err != nil {
		return "", err
	}
	repealed, err := res.RowsAffected()
	if err != nil {
		return "", err
	}
	if repealed == 0 {
		return "📜 O decreto do cassino já tinha caído, então os donos ficaram no prejuízo", nil
	}
	return "📜 Decreto revogado: " + casinoDecree.revoked, nil
}
