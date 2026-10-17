package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	roletaKind     = "roleta"
	roletaChambers = 6
	roletaSuspense = 2 * time.Second
	roletaKeep     = 7 * 24 * time.Hour
)

type revolver struct {
	mu       sync.Mutex
	chambers int
	bullet   int
	record   func()
}

type revolverSave struct {
	Chat     types.JID `json:"chat"`
	Chambers int       `json:"chambers"`
	Bullet   int       `json:"bullet"`
}

func (b *bot) trackRevolver(ctx context.Context, chat types.JID, gun *revolver) *revolver {
	gun.record = func() {
		b.keepGame(ctx, roletaKind, chat.String(), revolverSave{Chat: chat, Chambers: gun.chambers, Bullet: gun.bullet}, time.Now().Add(roletaKeep))
	}
	return gun
}

func (b *bot) restoreRevolver(ctx context.Context, saved savedGame) error {
	if time.Now().After(saved.expires) {
		b.dropGame(ctx, roletaKind, saved.key)
		return nil
	}
	var state revolverSave
	if err := json.Unmarshal(saved.state, &state); err != nil {
		return fmt.Errorf("ler o revólver do !roleta: %w", err)
	}
	b.revolvers.LoadOrStore(saved.key, b.trackRevolver(ctx, state.Chat, &revolver{chambers: state.Chambers, bullet: state.Bullet}))
	return nil
}

func newRevolver() *revolver {
	return &revolver{chambers: roletaChambers, bullet: rand.IntN(roletaChambers)}
}

func (r *revolver) pull() (fired bool, chambersLeft int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.bullet == 0 {
		r.chambers, r.bullet = roletaChambers, rand.IntN(roletaChambers)
		r.record()
		return true, r.chambers
	}
	r.chambers--
	r.bullet--
	r.record()
	return false, r.chambers
}

func (r *revolver) remaining() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.chambers
}

type roletaBet struct {
	shooter types.JID
	acc     account
	stake   int64
	boost   prizeBoost
	levy    pixLevy
}

func (b *bot) roleta(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	shooter := evt.Info.Sender.ToNonAD()
	target := firstMentionOrSender(evt).ToNonAD()
	value, _ := b.revolvers.LoadOrStore(chat.String(), b.trackRevolver(ctx, chat, newRevolver()))
	gun := value.(*revolver)
	bet, refusal, err := b.roletaBet(ctx, chat, shooter, firstPlainArg(messageText(evt.Message)), gun)
	if err != nil {
		return err
	}
	if refusal != "" {
		return b.sendText(ctx, chat, refusal)
	}

	intro := fmt.Sprintf("🔫 @%s colocou o revólver na cabeça e puxou o gatilho...", shooter.User)
	if target.User != shooter.User {
		intro = fmt.Sprintf("🔫 @%s apontou o revólver pra @%s e puxou o gatilho...", shooter.User, target.User)
	}
	if bet.stake > 0 {
		intro += "\n💰 Valendo " + coins(bet.stake)
	}
	if _, err := b.sendMentions(ctx, chat, intro, []types.JID{shooter, target}); err != nil {
		return err
	}

	fired, chambersLeft := gun.pull()
	payoff := b.payRoletaBet(ctx, chat, bet, fired, chambersLeft)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(roletaSuspense):
	}
	text := appendLine(roletaResult(target, fired, chambersLeft), payoff)
	_, err = b.sendMentions(ctx, chat, text, []types.JID{target, shooter})
	return err
}

func (b *bot) roletaBet(ctx context.Context, chat, shooter types.JID, arg string, gun *revolver) (roletaBet, string, error) {
	if arg == "" || !b.economyOn(ctx, chat) {
		return roletaBet{}, "", nil
	}
	if gun.remaining() == 1 {
		return roletaBet{}, "💀 Só sobrou 1 câmara e a bala tá nela, aí não tem aposta. Manda !roleta sem valor se tiver coragem", nil
	}
	acc := b.account(ctx, shooter)
	bet, refusal, err := b.placeBet(ctx, chat, acc, arg, "!roleta")
	return roletaBet{shooter: shooter, acc: acc, stake: bet.stake, boost: bet.boost, levy: bet.levy}, refusal, err
}

func (b *bot) payRoletaBet(ctx context.Context, chat types.JID, bet roletaBet, fired bool, chambersLeft int) string {
	if bet.stake == 0 {
		return ""
	}
	payout := bet.boost.pay(bet.stake, roletaPayout(bet.stake, fired, chambersLeft))
	balance, err := b.wallets.settleBet(ctx, chat, bet.acc, bet.stake, payout, bet.levy)
	if errors.Is(err, errNoFunds) {
		return fmt.Sprintf("🤨 O saldo de @%s mudou no meio e a aposta não valeu", bet.shooter.User)
	}
	if err != nil {
		b.client.Log.Errorf("pagar aposta do !roleta: %v", err)
		return ""
	}
	result := fmt.Sprintf("🤑 @%s ganhou %s%s", bet.shooter.User, coins(payout-bet.stake), bet.boost.tag(bet.stake, payout))
	if fired {
		result = fmt.Sprintf("💸 @%s perdeu %s", bet.shooter.User, coins(bet.stake))
	}
	return result + betLevyText(bet.stake, payout, bet.levy) + "\n💰 Saldo: " + coins(balance)
}

func roletaPayout(stake int64, fired bool, chambersLeft int) int64 {
	if fired {
		return 0
	}
	return stake + max(1, stake/int64(chambersLeft))
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
