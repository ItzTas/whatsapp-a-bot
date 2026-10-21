package games

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/economy"
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
		b.KeepGame(ctx, roletaKind, chat.String(), revolverSave{Chat: chat, Chambers: gun.chambers, Bullet: gun.bullet}, time.Now().Add(roletaKeep))
	}
	return gun
}

func (b *bot) restoreRevolver(ctx context.Context, saved core.SavedGame) error {
	if time.Now().After(saved.Expires) {
		b.DropGame(ctx, roletaKind, saved.Key)
		return nil
	}
	var state revolverSave
	if err := json.Unmarshal(saved.State, &state); err != nil {
		return fmt.Errorf("ler o revólver do !roleta: %w", err)
	}
	b.revolvers.LoadOrStore(saved.Key, b.trackRevolver(ctx, state.Chat, &revolver{chambers: state.Chambers, bullet: state.Bullet}))
	return nil
}

func newRevolver() *revolver {
	return &revolver{chambers: roletaChambers, bullet: rand.IntN(roletaChambers)}
}

func (r *revolver) pull(rig, spare float64) (fired bool, chambersLeft int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.bullet == 0 && r.chambers > 1 && rand.Float64() < spare {
		r.bullet = 1 + rand.IntN(r.chambers-1)
	}
	if r.bullet == 0 || rand.Float64() < rig {
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
	acc     core.Account
	stake   int64
	boost   economy.PrizeBoost
	busted  bool
}

func (b *bot) roleta(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	shooter := evt.Info.Sender.ToNonAD()
	target := core.FirstMentionOrSender(evt).ToNonAD()
	value, _ := b.revolvers.LoadOrStore(chat.String(), b.trackRevolver(ctx, chat, newRevolver()))
	gun := value.(*revolver)
	bet, refusal, err := b.roletaBet(ctx, chat, shooter, core.FirstPlainArg(core.MessageText(evt.Message)), gun)
	if err != nil || bet.busted {
		return err
	}
	if refusal != "" {
		return b.SendText(ctx, chat, refusal)
	}

	intro := fmt.Sprintf("🔫 @%s colocou o revólver na cabeça e puxou o gatilho...", shooter.User)
	if target.User != shooter.User {
		intro = fmt.Sprintf("🔫 @%s apontou o revólver pra @%s e puxou o gatilho...", shooter.User, target.User)
	}
	if bet.stake > 0 {
		intro += "\n💰 Valendo " + economy.Coins(bet.stake)
	}
	if _, err := b.SendMentions(ctx, chat, intro, []types.JID{shooter, target}); err != nil {
		return err
	}

	fired, chambersLeft := gun.pull(bet.boost.Risk(bet.stake), bet.boost.Favor(bet.stake))
	payoff, paid, settled := b.payRoletaBet(ctx, chat, bet, fired, chambersLeft)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(roletaSuspense):
	}
	text := core.AppendLine(roletaResult(target, fired, chambersLeft), payoff)
	if _, err := b.SendMentions(ctx, chat, text, []types.JID{target, shooter}); err != nil {
		return err
	}
	if settled {
		b.economy.NoteBet(ctx, chat, bet.acc, bet.stake, paid)
	}
	return nil
}

func (b *bot) roletaBet(ctx context.Context, chat, shooter types.JID, arg string, gun *revolver) (roletaBet, string, error) {
	if arg == "" || !b.economy.EconomyOn(ctx, chat) {
		return roletaBet{}, "", nil
	}
	if gun.remaining() == 1 {
		return roletaBet{}, "💀 Só sobrou 1 câmara e a bala tá nela, aí não tem aposta. Manda !roleta sem valor se tiver coragem", nil
	}
	acc := b.Account(ctx, shooter)
	bet, refusal, err := b.economy.PlaceBet(ctx, chat, acc, arg, "!roleta")
	return roletaBet{shooter: shooter, acc: acc, stake: bet.Stake, boost: bet.Boost, busted: bet.Busted}, refusal, err
}

func (b *bot) payRoletaBet(ctx context.Context, chat types.JID, bet roletaBet, fired bool, chambersLeft int) (text string, paid int64, settled bool) {
	if bet.stake == 0 {
		return "", 0, false
	}
	payout := bet.boost.Pay(bet.stake, roletaPayout(bet.stake, fired, chambersLeft))
	saved := bet.boost.Rescue(bet.stake, payout)
	balance, err := b.economy.Wallets.SettleBet(ctx, chat, bet.acc, bet.stake, payout+saved, bet.boost.Levy())
	if errors.Is(err, economy.ErrNoFunds) {
		return fmt.Sprintf("🤨 O saldo de @%s mudou no meio e a aposta não valeu", bet.shooter.User), 0, false
	}
	if err != nil {
		b.Client.Log.Errorf("pagar aposta do !roleta: %v", err)
		return "", 0, false
	}
	result := fmt.Sprintf("🤑 @%s ganhou %s%s", bet.shooter.User, economy.Coins(payout-bet.stake), bet.boost.Tag(bet.stake, payout))
	if fired {
		result = fmt.Sprintf("💸 @%s perdeu %s", bet.shooter.User, economy.Coins(bet.stake)) + economy.SavedBetText(saved)
	}
	return result + economy.BetLevyText(bet.stake, bet.boost.Levy()) + "\n💰 Saldo: " + economy.Coins(balance), payout + saved, true
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
