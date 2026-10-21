package economy

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"golang.org/x/sync/errgroup"
)

type idleBoard struct {
	job    Profession
	now    time.Time
	busy   string
	bag    []stack
	worked time.Time
	dug    []time.Time
	beds   []bed
	bought int
	stalls int
}

func (b *bot) meanwhile(ctx context.Context, chat types.JID, acc core.Account, job Profession, now time.Time, busy string) (string, error) {
	board := idleBoard{job: job, now: now, busy: busy, dug: make([]time.Time, len(activities))}
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { board.bag, err = b.Wallets.inventory(gctx, chat, acc); return err })
	g.Go(func() (err error) {
		board.worked, err = stampedAt(gctx, b.Wallets.db, chat, acc, "trabalho_em")
		return err
	})
	for i, a := range activities {
		g.Go(func() (err error) {
			board.dug[i], err = stampedAt(gctx, b.Wallets.db, chat, acc, a.cooldown.column)
			return err
		})
	}
	g.Go(func() (err error) { board.beds, err = b.Wallets.beds(gctx, chat, acc); return err })
	g.Go(func() (err error) { board.bought, err = b.Wallets.boughtBeds(gctx, chat, acc); return err })
	g.Go(func() (err error) { board.stalls, err = b.Wallets.rivalStalls(gctx, chat, acc, now); return err })
	if err := g.Wait(); err != nil {
		return "", fmt.Errorf("ver o que dá pra fazer na espera: %w", err)
	}
	return board.text(b.readyGathers(chat, board)), nil
}

func (b *bot) readyGathers(chat types.JID, board idleBoard) []string {
	var ready []string
	for i, base := range activities {
		if base.command == board.busy {
			continue
		}
		a, _ := b.paced(chat, board.job, base, board.now)
		k := a.equip(board.bag)
		if k.armed && !board.now.Before(board.dug[i].Add(k.cooldown.every)) {
			ready = append(ready, a.command)
		}
	}
	return ready
}

func (board idleBoard) text(gathers []string) string {
	ready := gathers
	if board.busy != "!trabalhar" && !board.now.Before(board.worked.Add(board.job.workCooldown())) {
		ready = append([]string{"!trabalhar"}, ready...)
	}
	var lines []string
	if len(ready) == 1 {
		lines = append(lines, "⏱️ o "+ready[0]+" já tá liberado")
	}
	if len(ready) > 1 {
		lines = append(lines, "⏱️ o "+strings.Join(ready[:len(ready)-1], ", o ")+" e o "+ready[len(ready)-1]+" já tão liberados")
	}
	if garden := board.gardenLine(); garden != "" {
		lines = append(lines, garden)
	}
	if board.stalls > 0 {
		lines = append(lines, fmt.Sprintf("🛒 tem %d %s da galera na !feira", board.stalls, pluralWord(int64(board.stalls), "anúncio", "anúncios")))
	}
	if len(lines) == 0 {
		lines = append(lines, core.PickOne([]string{
			"🃏 dá pra arriscar uma mão no !blackjack",
			"🚀 dá pra subir no foguete do !crash",
			"🐎 dá pra apostar num cavalo na !corrida",
			"🎯 dá pra ver como tão as !missoes de hoje",
			"🛒 dá pra anunciar o que sobrou da mochila na !feira",
		}))
	}
	return "\n\n👉 *Enquanto isso:*\n" + strings.Join(lines, "\n")
}

func (board idleBoard) gardenLine() string {
	ripe := len(slices.DeleteFunc(slices.Clone(board.beds), func(b bed) bool { return !b.ripe(board.now) }))
	if ripe > 0 {
		return fmt.Sprintf("🌱 %d %s pro !colher", ripe, pluralWord(int64(ripe), "canteiro pronto", "canteiros prontos"))
	}
	free := board.job.gardenRoom() + board.bought - len(board.beds)
	if free > 0 && slices.ContainsFunc(board.bag, func(s stack) bool { return s.item.seed != nil }) {
		return "🌱 tem canteiro vazio e semente na mochila pro !plantar"
	}
	return ""
}
