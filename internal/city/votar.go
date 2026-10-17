package city

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"go.mau.fi/whatsmeow/types/events"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
)

func (b *bot) votar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	game, active := b.activeCity(chat)
	if !active {
		return b.SendText(ctx, chat, "Não tem partida de Cidade Dorme rolando agora. Manda !cidadedorme pra abrir uma sala 🌃")
	}
	target, ok := cityBallot(evt, game)
	if !ok {
		return b.SendText(ctx, chat, "Use assim: !votar @pessoa, !votar <número da lista do !cidadedorme> ou !votar ninguem")
	}
	text, mentions := game.vote(evt.Info, target)
	return b.postCity(ctx, game, text, mentions)
}

func cityBallot(evt *events.Message, game *cityGame) (int, bool) {
	if mentions := core.MentionedJIDs(evt.Message); len(mentions) > 0 {
		game.mu.Lock()
		defer game.mu.Unlock()
		if seat := game.seatOfUser(mentions[0].User); seat >= 0 {
			return seat, true
		}
		return cityStranger, true
	}
	arg := core.FoldLetters(core.FirstPlainArg(core.MessageText(evt.Message)))
	if slices.Contains([]string{"ninguem", "pular", "branco"}, arg) {
		return cityNobody, true
	}
	n, err := strconv.Atoi(strings.TrimPrefix(arg, "#"))
	if err != nil {
		return 0, false
	}
	if n < 1 {
		return cityStranger, true
	}
	return n - 1, true
}
