package economy

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	crashKind      = "crash"
	crashWindow    = 15 * time.Second
	crashTick      = 1500 * time.Millisecond
	crashDoubling  = 6 * time.Second
	crashHouse     = 97
	crashFloor     = 100
	crashCap       = 10000
	crashMinTarget = 101
)

type crashBet struct {
	acc    core.Account
	stake  int64
	boost  PrizeBoost
	levy   heldLevy
	target int64
	cashed int64
}

type crashRound struct {
	mu     sync.Mutex
	opened time.Time
	start  time.Time
	point  int64
	over   bool
	bets   []*crashBet
	record func()
}

type crashSave struct {
	Chat   types.JID      `json:"chat"`
	Opened time.Time      `json:"opened"`
	Start  time.Time      `json:"start"`
	Point  int64          `json:"point"`
	Bets   []crashSaveBet `json:"bets"`
}

type crashSaveBet struct {
	Account core.StoredAccount `json:"account"`
	Stake   int64              `json:"stake"`
	Boost   int64              `json:"boost"`
	Luck    int64              `json:"luck"`
	Heat    int64              `json:"heat"`
	Rescued bool               `json:"rescued"`
	Levy    heldLevySave       `json:"levy"`
	Target  int64              `json:"target"`
	Cashed  int64              `json:"cashed"`
}

func (b *bot) crash(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	args := strings.Fields(strings.ToLower(core.MessageText(evt.Message)))[1:]
	if len(args) == 0 {
		return b.SendText(ctx, chat, b.crashStatus(chat))
	}
	if args[0] != "sacar" {
		return b.betCrash(ctx, evt, args)
	}
	playing, err := b.cashOutCrash(ctx, evt)
	if err != nil || playing {
		return err
	}
	return b.SendText(ctx, chat, "Você não tá em nenhum foguete agora 🚀 Manda !crash <valor> pra entrar")
}

func (b *bot) answerCrash(ctx context.Context, evt *events.Message) error {
	if core.FoldLetters(strings.TrimSpace(core.MessageText(evt.Message))) != "sacar" {
		return nil
	}
	_, err := b.cashOutCrash(ctx, evt)
	return err
}

func (b *bot) answerCrashCommand(ctx context.Context, evt *events.Message, name string) bool {
	if core.FoldLetters(name) != "sacar" {
		return false
	}
	playing, err := b.cashOutCrash(ctx, evt)
	if err != nil {
		b.Client.Log.Errorf("sacar do !crash com !sacar: %v", err)
	}
	return playing
}

func (b *bot) betCrash(ctx context.Context, evt *events.Message, args []string) error {
	chat := evt.Info.Chat.ToNonAD()
	target, ok := crashTarget(args)
	if !ok {
		return b.SendText(ctx, chat, fmt.Sprintf("O saque automático vai de %s a %s, tipo !crash 100 2x", crashX(crashMinTarget), crashX(crashCap-1)))
	}
	acc := b.Account(ctx, evt.Info.Sender)
	placed, refusal, err := b.PlaceBet(ctx, chat, acc, args[0], "!crash")
	if err != nil {
		return err
	}
	if refusal != "" {
		return b.SendText(ctx, chat, refusal)
	}

	value, joined := b.crashRounds.LoadOrStore(chat.String(), b.trackCrash(ctx, chat, newCrashRound()))
	round := value.(*crashRound)
	if !joined {
		go b.runCrash(ctx, chat, round, crashWindow)
	}
	bet := &crashBet{acc: acc, stake: placed.Stake, boost: placed.Boost, levy: placed.held(), target: target}
	refusal, err = b.enterCrash(ctx, chat, round, bet, placed.Levy)
	if err != nil {
		return err
	}
	if refusal != "" {
		return b.SendText(ctx, chat, refusal)
	}
	text := fmt.Sprintf("🚀 *Crash!* Aposta aberta por %s, manda !crash <valor> pra entrar\n_Quando decolar, o multiplicador sobe até o foguete explodir: manda *sacar* antes disso_\n\n%s",
		formatSeconds(crashWindow), bet.entry())
	if joined {
		text = fmt.Sprintf("%s. Decola em %s 🚀", bet.entry(), formatSeconds(crashWindow-time.Since(round.opened)))
	}
	text += BetLevyText(bet.stake, 0, placed.Levy)
	_, err = b.SendMentions(ctx, chat, text, []types.JID{acc.JID})
	return err
}

func (b *bot) enterCrash(ctx context.Context, chat types.JID, round *crashRound, bet *crashBet, levy PixLevy) (string, error) {
	round.mu.Lock()
	defer round.mu.Unlock()
	if round.over || !round.start.IsZero() {
		return "🚀 O foguete já decolou, espera essa rodada acabar pra apostar", nil
	}
	if slices.ContainsFunc(round.bets, func(other *crashBet) bool { return other.acc.Key == bet.acc.Key }) {
		return "Você já tá nesse foguete 🚀 É uma aposta por pessoa", nil
	}
	held, err := b.holdBet(ctx, chat, bet.acc, bet.stake, levy)
	if err != nil {
		return "", fmt.Errorf("apostar no !crash: %w", err)
	}
	if !held {
		return StakeChanged, nil
	}
	round.bets = append(round.bets, bet)
	round.record()
	return "", nil
}

func (b *bot) cashOutCrash(ctx context.Context, evt *events.Message) (bool, error) {
	now := time.Now()
	chat := evt.Info.Chat.ToNonAD()
	value, found := b.crashRounds.Load(chat.String())
	if !found {
		return false, nil
	}
	bet, refusal := value.(*crashRound).cashOut(b.Account(ctx, evt.Info.Sender).Key, now)
	if bet == nil {
		return false, nil
	}
	mentions := []types.JID{bet.acc.JID}
	if refusal != "" {
		_, err := b.SendMentions(ctx, chat, refusal, mentions)
		return true, err
	}
	balance, err := b.payStake(ctx, chat, bet.acc, bet.payout(), bet.prizeLevy())
	if err != nil {
		return true, fmt.Errorf("pagar saque do !crash: %w", err)
	}
	text := fmt.Sprintf("💰 @%s sacou em %s e levou %s%s (saldo: %s)", bet.acc.JID.User, crashX(bet.cashed), Coins(bet.payout()), bet.tag(), Coins(balance))
	_, err = b.SendMentions(ctx, chat, text, mentions)
	return true, err
}

func (b *bot) runCrash(ctx context.Context, chat types.JID, round *crashRound, wait time.Duration) {
	defer b.Rescue(ctx, chat)
	defer b.crashRounds.CompareAndDelete(chat.String(), round)
	select {
	case <-ctx.Done():
		return
	case <-time.After(wait):
	}
	if round.launch(time.Now()) {
		b.flyCrash(ctx, chat, round)
	}
}

func (b *bot) flyCrash(ctx context.Context, chat types.JID, round *crashRound) {
	nicknames := b.ChatNicknames(ctx, chat)
	mentions := round.players()
	board := func() string { return core.TagNicknames(round.board(time.Now()), nicknames) }
	flight, err := b.SendRawMentions(ctx, chat, board(), mentions)
	if err != nil {
		b.Client.Log.Errorf("mandar o foguete do !crash: %v", err)
	}
	boom := time.NewTimer(time.Until(round.start.Add(crashFlight(round.point))))
	defer boom.Stop()
	tick := time.NewTicker(crashTick)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			b.payCrash(ctx, chat, round.autoCashOuts(time.Now()))
			b.editCrash(ctx, chat, flight, board(), mentions)
		case <-boom.C:
			b.payCrash(ctx, chat, round.explode())
			b.payRescues(ctx, chat, "!crash", round.rescues())
			result := board()
			b.editCrash(ctx, chat, flight, result, mentions)
			if _, err := b.SendRawMentions(ctx, chat, result, mentions); err != nil {
				b.Client.Log.Errorf("mandar resultado do !crash: %v", err)
			}
			b.tallyRound(ctx, chat, "!crash", round.settled())
			return
		}
	}
}

func (b *bot) payCrash(ctx context.Context, chat types.JID, bets []*crashBet) {
	for _, bet := range bets {
		if err := b.Wallets.payPrize(ctx, chat, bet.acc, bet.payout(), bet.prizeLevy()); err != nil {
			b.Client.Log.Errorf("pagar saque automático do !crash pra %s: %v", bet.acc.JID, err)
		}
	}
}

func (b *bot) editCrash(ctx context.Context, chat types.JID, flight types.MessageID, text string, mentions []types.JID) {
	if flight == "" {
		return
	}
	if _, err := b.Client.SendMessage(ctx, chat, b.Client.BuildEdit(chat, flight, core.MentionMessage(text, mentions))); err != nil {
		b.Client.Log.Warnf("editar o foguete do !crash: %v", err)
	}
}

func (b *bot) crashStatus(chat types.JID) string {
	usage := "🚀 Crash: manda !crash 100 pra entrar no foguete (ou !crash 100 2x pra sacar sozinho em 2x) e manda sacar antes de ele explodir"
	value, found := b.crashRounds.Load(chat.String())
	if !found {
		return usage
	}
	return value.(*crashRound).status() + "\n\n" + usage
}

func (b *bot) trackCrash(ctx context.Context, chat types.JID, round *crashRound) *crashRound {
	round.record = func() { b.recordCrash(ctx, chat, round) }
	return round
}

func (b *bot) recordCrash(ctx context.Context, chat types.JID, round *crashRound) {
	if round.over {
		b.DropGame(ctx, crashKind, chat.String())
		return
	}
	b.KeepGame(ctx, crashKind, chat.String(), round.saved(chat), round.ends())
}

func (b *bot) restoreCrash(ctx context.Context, saved core.SavedGame) error {
	var state crashSave
	if err := json.Unmarshal(saved.State, &state); err != nil {
		return fmt.Errorf("ler a rodada do !crash: %w", err)
	}
	bets := make([]*crashBet, len(state.Bets))
	for i, bet := range state.Bets {
		bets[i] = &crashBet{acc: bet.Account.Account(), stake: bet.Stake, boost: PrizeBoost{decree: bet.Boost, luck: bet.Luck, heat: bet.Heat, rescued: bet.Rescued}, levy: bet.Levy.held(), target: bet.Target, cashed: bet.Cashed}
	}
	round := b.trackCrash(ctx, state.Chat, &crashRound{opened: state.Opened, start: state.Start, point: state.Point, bets: bets})
	wait := crashWindow - time.Since(round.opened)
	if !round.start.IsZero() || wait <= 0 {
		return b.refundCrash(ctx, state.Chat, round)
	}
	if _, loaded := b.crashRounds.LoadOrStore(saved.Key, round); loaded {
		return nil
	}
	go b.runCrash(ctx, state.Chat, round, wait)
	return nil
}

func (b *bot) refundCrash(ctx context.Context, chat types.JID, round *crashRound) error {
	refunds := round.cancel()
	if len(refunds) == 0 {
		return nil
	}
	lines := []string{"🚀 O bot reiniciou no meio do !crash, então a rodada foi cancelada e as apostas voltaram", ""}
	mentions := make([]types.JID, len(refunds))
	for i, bet := range refunds {
		mentions[i] = bet.acc.JID
		lines = append(lines, fmt.Sprintf("↩️ @%s recebeu %s de volta", bet.acc.JID.User, Coins(bet.stake)))
		if err := b.Wallets.Credit(ctx, chat, []core.Account{bet.acc}, bet.stake); err != nil {
			b.Client.Log.Errorf("devolver aposta do !crash pra %s: %v", bet.acc.JID, err)
		}
	}
	_, err := b.SendMentions(ctx, chat, strings.Join(lines, "\n"), mentions)
	return err
}

func newCrashRound() *crashRound {
	return &crashRound{opened: time.Now(), point: drawCrashPoint()}
}

func drawCrashPoint() int64 {
	u := 1 - rand.Float64()
	return min(crashCap, max(crashFloor, int64(crashHouse/u)))
}

func (r *crashRound) launch(now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.record()
	if len(r.bets) == 0 {
		r.over = true
		return false
	}
	r.start = now
	r.rig()
	return true
}

func (r *crashRound) rig() {
	riskiest := slices.MaxFunc(r.bets, func(a, b *crashBet) int { return cmp.Compare(a.boost.Risk(a.stake), b.boost.Risk(b.stake)) })
	if rand.Float64() < riskiest.boost.Risk(riskiest.stake) {
		r.point = min(r.point, drawCrashPoint())
	}
}

func (r *crashRound) cancel() []*crashBet {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.over = true
	r.record()
	var refunds []*crashBet
	for _, bet := range r.bets {
		if bet.cashed == 0 {
			refunds = append(refunds, bet)
		}
	}
	return refunds
}

func (r *crashRound) ends() time.Time {
	if r.start.IsZero() {
		return r.opened.Add(crashWindow)
	}
	return r.start.Add(crashFlight(r.point))
}

func (r *crashRound) saved(chat types.JID) crashSave {
	bets := make([]crashSaveBet, len(r.bets))
	for i, bet := range r.bets {
		bets[i] = crashSaveBet{Account: core.StoreAccount(bet.acc), Stake: bet.stake, Boost: bet.boost.decree, Luck: bet.boost.luck, Heat: bet.boost.heat, Rescued: bet.boost.rescued, Levy: bet.levy.saved(), Target: bet.target, Cashed: bet.cashed}
	}
	return crashSave{Chat: chat, Opened: r.opened, Start: r.start, Point: r.point, Bets: bets}
}

func (r *crashRound) players() []types.JID {
	r.mu.Lock()
	defer r.mu.Unlock()
	players := make([]types.JID, len(r.bets))
	for i, bet := range r.bets {
		players[i] = bet.acc.JID
	}
	return players
}

func (r *crashRound) cashOut(key string, now time.Time) (*crashBet, string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	i := slices.IndexFunc(r.bets, func(bet *crashBet) bool { return bet.acc.Key == key })
	if i < 0 {
		return nil, ""
	}
	bet := r.bets[i]
	user := bet.acc.JID.User
	if r.start.IsZero() {
		return bet, fmt.Sprintf("⏳ Calma, @%s, o foguete ainda nem decolou", user)
	}
	if bet.cashed > 0 {
		return bet, fmt.Sprintf("@%s, você já sacou em %s", user, crashX(bet.cashed))
	}
	elapsed := now.Sub(r.start)
	current := crashMultiplier(elapsed)
	if bet.reached(current, r.point) {
		bet.cashed = bet.target
		r.record()
		return bet, ""
	}
	if elapsed >= crashFlight(r.point) {
		return bet, fmt.Sprintf("💥 Tarde demais, @%s, o foguete explodiu em %s", user, crashX(r.point))
	}
	bet.cashed = min(current, r.point-1)
	r.record()
	return bet, ""
}

func (r *crashRound) autoCashOuts(now time.Time) []*crashBet {
	r.mu.Lock()
	defer r.mu.Unlock()
	cashed := r.reachTargets(crashMultiplier(now.Sub(r.start)))
	if len(cashed) > 0 {
		r.record()
	}
	return cashed
}

func (r *crashRound) explode() []*crashBet {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.over = true
	r.record()
	return r.reachTargets(r.point)
}

func (r *crashRound) settled() []settledBet {
	r.mu.Lock()
	defer r.mu.Unlock()
	bets := make([]settledBet, len(r.bets))
	for i, bet := range r.bets {
		bets[i] = settledBet{acc: bet.acc, stake: bet.stake, payout: bet.payout()}
	}
	return bets
}

func (r *crashRound) rescues() []rescuedBet {
	r.mu.Lock()
	defer r.mu.Unlock()
	var rescued []rescuedBet
	for _, bet := range r.bets {
		if bet.cashed == 0 && bet.boost.rescued {
			rescued = append(rescued, rescuedBet{acc: bet.acc, stake: bet.stake})
		}
	}
	return rescued
}

func (r *crashRound) reachTargets(current int64) []*crashBet {
	var cashed []*crashBet
	for _, bet := range r.bets {
		if bet.cashed > 0 || !bet.reached(current, r.point) {
			continue
		}
		bet.cashed = bet.target
		cashed = append(cashed, bet)
	}
	return cashed
}

func (r *crashRound) status() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.start.IsZero() {
		return fmt.Sprintf("🚀 Tem rodada aberta com %d no foguete, decola em %s", len(r.bets), formatSeconds(crashWindow-time.Since(r.opened)))
	}
	return fmt.Sprintf("🚀 O foguete tá no ar em %s, quem tá dentro manda sacar", crashX(min(crashMultiplier(time.Since(r.start)), r.point)))
}

func (r *crashRound) board(now time.Time) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	header := fmt.Sprintf("🚀 *%s* 📈\n_Manda *sacar* antes de explodir_", crashX(min(crashMultiplier(now.Sub(r.start)), r.point)))
	if r.over {
		header = fmt.Sprintf("💥 *O foguete explodiu em %s!*", crashX(r.point))
	}
	lines := []string{header, ""}
	for _, bet := range r.bets {
		lines = append(lines, bet.line(r.over))
	}
	return strings.Join(lines, "\n")
}

func (bet *crashBet) reached(current, point int64) bool {
	return bet.target > 0 && bet.target < point && bet.target <= current
}

func (bet *crashBet) payout() int64 {
	return bet.boost.Pay(bet.stake, bet.stake*bet.cashed/100)
}

func (bet *crashBet) prizeLevy() prizeLevy {
	return bet.levy.onPrize(bet.stake, bet.payout())
}

func (bet *crashBet) tag() string {
	return bet.boost.Tag(bet.stake, bet.payout()) + bet.prizeLevy().text()
}

func (bet *crashBet) entry() string {
	return fmt.Sprintf("✅ @%s entrou com %s%s", bet.acc.JID.User, Coins(bet.stake), bet.autoNote())
}

func (bet *crashBet) line(exploded bool) string {
	user := bet.acc.JID.User
	if bet.cashed > 0 {
		return fmt.Sprintf("✅ @%s sacou em %s e levou %s%s", user, crashX(bet.cashed), Coins(bet.payout()), bet.tag())
	}
	if exploded {
		return fmt.Sprintf("💸 @%s perdeu %s", user, Coins(bet.stake)) + savedBetNote(bet.boost)
	}
	return fmt.Sprintf("🧑‍🚀 @%s tá no foguete com %s%s", user, Coins(bet.stake), bet.autoNote())
}

func (bet *crashBet) autoNote() string {
	if bet.target == 0 {
		return ""
	}
	return " (saca sozinho em " + crashX(bet.target) + ")"
}

func crashTarget(args []string) (int64, bool) {
	if len(args) < 2 {
		return 0, true
	}
	number := strings.ReplaceAll(strings.TrimSuffix(args[1], "x"), ",", ".")
	value, err := strconv.ParseFloat(number, 64)
	target := int64(math.Round(value * 100))
	return target, err == nil && target >= crashMinTarget && target < crashCap
}

func crashMultiplier(elapsed time.Duration) int64 {
	return int64(crashFloor * math.Exp2(elapsed.Seconds()/crashDoubling.Seconds()))
}

func crashFlight(point int64) time.Duration {
	return time.Duration(math.Log2(float64(point)/crashFloor) * float64(crashDoubling))
}

func crashX(multiplier int64) string {
	return fmt.Sprintf("%d,%02dx", multiplier/100, multiplier%100)
}

func formatSeconds(d time.Duration) string {
	return fmt.Sprintf("%ds", max(1, int(math.Ceil(d.Seconds()))))
}
