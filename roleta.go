package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	roletaChambers = 6
	roletaSuspense = 2 * time.Second
)

type revolver struct {
	mu       sync.Mutex
	chambers int
	bullet   int
}

func newRevolver() *revolver {
	return &revolver{chambers: roletaChambers, bullet: rand.IntN(roletaChambers)}
}

func (r *revolver) pull() (fired bool, chambersLeft int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.bullet == 0 {
		r.chambers, r.bullet = roletaChambers, rand.IntN(roletaChambers)
		return true, r.chambers
	}
	r.chambers--
	r.bullet--
	return false, r.chambers
}

func (b *bot) roleta(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	shooter := evt.Info.Sender.ToNonAD()
	target := firstMentionOrSender(evt).ToNonAD()

	intro := fmt.Sprintf("🔫 @%s colocou o revólver na cabeça e puxou o gatilho...", shooter.User)
	if target.User != shooter.User {
		intro = fmt.Sprintf("🔫 @%s apontou o revólver pra @%s e puxou o gatilho...", shooter.User, target.User)
	}
	if _, err := b.sendMentions(ctx, chat, intro, []types.JID{shooter, target}); err != nil {
		return err
	}

	value, _ := b.revolvers.LoadOrStore(chat.String(), newRevolver())
	fired, chambersLeft := value.(*revolver).pull()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(roletaSuspense):
	}
	_, err := b.sendMentions(ctx, chat, roletaResult(target, fired, chambersLeft), []types.JID{target})
	return err
}

func roletaResult(target types.JID, fired bool, chambersLeft int) string {
	if !fired {
		return fmt.Sprintf("*click* 😮‍💨 @%s sobreviveu!\n\n🎯 Chance da próxima: 1 em %d", target.User, chambersLeft)
	}
	funerais := []string{
		"⚰️ O velório vai ser no grupo mesmo",
		"🕯️ Alguém acende uma vela",
		"🌹 Vai deixar saudade... ou não",
		"📜 O testamento deixou tudo pro grupo",
		"👻 Agora vai assombrar o grupo pra sempre",
		"🎺 Tocaram funk no enterro, como era o desejo",
	}
	return fmt.Sprintf("💥 *BANG!* @%s morreu 💀\n\n%s\n🔄 Revólver recarregado com 1 bala em %d câmaras",
		target.User, funerais[rand.IntN(len(funerais))], roletaChambers)
}
