package economy

import (
	"context"
	"encoding/json"
	"fmt"
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
	raceKind   = "corrida"
	raceWindow = 30 * time.Second
	raceTick   = 2 * time.Second
	raceLength = 12
	raceHorses = 5
	raceHouse  = 92
)

var raceNames = []string{
	"Pé de Pano", "Relâmpago", "Pangaré", "Trovão", "Ventania", "Pipoca", "Cometa", "Sardinha",
	"Furacão", "Bolacha", "Zé Ruela", "Tico-Tico", "Pé de Valsa", "Tornado", "Paçoca", "Capivara Veloz",
	"Barão", "Faísca", "Pastel", "Sem Freio", "Mula Manca", "Foguetinho", "Tropeço", "Coxinha",
}

type raceHorse struct {
	name   string
	chance float64
	odds   int64
}

type raceBet struct {
	acc   core.Account
	stake int64
	boost PrizeBoost
	horse int
}

type raceRound struct {
	mu      sync.Mutex
	opened  time.Time
	started bool
	horses  []raceHorse
	winner  int
	frames  [][]int
	bets    []*raceBet
	ends    time.Time
	over    bool
	record  func()
}

type raceSave struct {
	Chat    types.JID       `json:"chat"`
	Opened  time.Time       `json:"opened"`
	Started bool            `json:"started"`
	Horses  []raceSaveHorse `json:"horses"`
	Winner  int             `json:"winner"`
	Frames  [][]int         `json:"frames"`
	Bets    []raceSaveBet   `json:"bets"`
}

type raceSaveHorse struct {
	Name   string  `json:"name"`
	Chance float64 `json:"chance"`
	Odds   int64   `json:"odds"`
}

type raceSaveBet struct {
	Account core.StoredAccount `json:"account"`
	Stake   int64              `json:"stake"`
	Decree  int64              `json:"decree,omitempty"`
	Luck    int64              `json:"luck"`
	Heat    int64              `json:"heat"`
	Solace  int64              `json:"solace"`
	Cheat   int64              `json:"cheat,omitempty"`
	Rescued bool               `json:"rescued"`
	Horse   int                `json:"horse"`
}

func (b *bot) corrida(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	args := strings.Fields(strings.ToLower(core.MessageText(evt.Message)))[1:]
	if len(args) == 0 {
		return b.openRace(ctx, chat)
	}
	return b.betRace(ctx, evt, args)
}

func (b *bot) openRace(ctx context.Context, chat types.JID) error {
	round, opened := b.raceRound(ctx, chat)
	if !opened {
		return b.SendText(ctx, chat, round.status())
	}
	return b.SendText(ctx, chat, round.announcement())
}

func (b *bot) raceRound(ctx context.Context, chat types.JID) (*raceRound, bool) {
	value, found := b.raceRounds.LoadOrStore(chat.String(), b.trackRace(ctx, chat, newRaceRound()))
	round := value.(*raceRound)
	if !found {
		round.save()
		go b.runRace(ctx, chat, round, raceWindow)
	}
	return round, !found
}

func (b *bot) trackRace(ctx context.Context, chat types.JID, round *raceRound) *raceRound {
	round.record = func() { b.recordRace(ctx, chat, round) }
	return round
}

func (b *bot) recordRace(ctx context.Context, chat types.JID, round *raceRound) {
	if round.over {
		b.DropGame(ctx, raceKind, chat.String())
		return
	}
	b.KeepGame(ctx, raceKind, chat.String(), round.saved(chat), round.deadline())
}

func (b *bot) restoreRace(ctx context.Context, saved core.SavedGame) error {
	var state raceSave
	if err := json.Unmarshal(saved.State, &state); err != nil {
		return fmt.Errorf("ler a !corrida: %w", err)
	}
	horses := make([]raceHorse, len(state.Horses))
	for i, horse := range state.Horses {
		horses[i] = raceHorse{name: horse.Name, chance: horse.Chance, odds: horse.Odds}
	}
	bets := make([]*raceBet, len(state.Bets))
	for i, bet := range state.Bets {
		bets[i] = &raceBet{acc: bet.Account.Account(), stake: bet.Stake, boost: PrizeBoost{decree: bet.Decree, luck: bet.Luck, heat: bet.Heat, solace: bet.Solace, cheat: bet.Cheat, rescued: bet.Rescued}, horse: bet.Horse}
	}
	round := b.trackRace(ctx, state.Chat, &raceRound{
		opened:  state.Opened,
		started: state.Started,
		horses:  horses,
		winner:  state.Winner,
		frames:  state.Frames,
		bets:    bets,
	})
	if _, loaded := b.raceRounds.LoadOrStore(saved.Key, round); loaded {
		return nil
	}
	go b.runRace(ctx, state.Chat, round, max(0, raceWindow-time.Since(round.opened)))
	return nil
}

func (b *bot) betRace(ctx context.Context, evt *events.Message, args []string) error {
	chat := evt.Info.Chat.ToNonAD()
	horse, ok := raceHorseArg(args)
	if !ok {
		return b.SendText(ctx, chat, fmt.Sprintf("Diz o valor e o número do cavalo (de 1 a %d): !corrida 100 3", raceHorses))
	}
	acc := b.Account(ctx, evt.Info.Sender)
	placed, refusal, err := b.PlaceBet(ctx, chat, acc, args[0], "!corrida")
	if err != nil || placed.Busted {
		return err
	}
	if refusal != "" {
		return b.SendText(ctx, chat, refusal)
	}

	round, opened := b.raceRound(ctx, chat)
	bet := &raceBet{acc: acc, stake: placed.Stake, boost: placed.Boost, horse: horse}
	refusal, err = b.enterRace(ctx, chat, round, bet)
	if err != nil {
		return err
	}
	if refusal != "" {
		return b.SendText(ctx, chat, refusal)
	}
	text := fmt.Sprintf("%s. Largada em %s 🏇", round.entry(bet), formatSeconds(raceWindow-time.Since(round.opened)))
	if opened {
		text = round.announcement() + "\n\n" + round.entry(bet)
	}
	text += BetLevyText(bet.stake, bet.boost.levy)
	_, err = b.SendMentions(ctx, chat, text, []types.JID{acc.JID})
	return err
}

func (b *bot) enterRace(ctx context.Context, chat types.JID, round *raceRound, bet *raceBet) (string, error) {
	round.mu.Lock()
	defer round.mu.Unlock()
	if round.started {
		return "🏇 Os cavalos já largaram, espera essa corrida acabar pra apostar", nil
	}
	if slices.ContainsFunc(round.bets, func(other *raceBet) bool { return other.acc.Key == bet.acc.Key }) {
		return "Você já apostou nessa corrida 🏇 É uma aposta por pessoa", nil
	}
	held, err := b.holdStake(ctx, chat, bet.acc, bet.stake, bet.boost.levy)
	if err != nil {
		return "", fmt.Errorf("apostar na !corrida: %w", err)
	}
	if !held {
		return stakeChanged, nil
	}
	round.bets = append(round.bets, bet)
	round.record()
	return "", nil
}

func (b *bot) runRace(ctx context.Context, chat types.JID, round *raceRound, wait time.Duration) {
	defer b.Rescue(ctx, chat)
	defer b.raceRounds.CompareAndDelete(chat.String(), round)
	select {
	case <-ctx.Done():
		return
	case <-time.After(wait):
	}
	if round.start() {
		b.gallopRace(ctx, chat, round)
		return
	}
	if err := b.SendText(ctx, chat, "🐴 Ninguém apostou, a corrida foi cancelada e os cavalos voltaram pro estábulo"); err != nil {
		b.Client.Log.Errorf("cancelar a !corrida: %v", err)
	}
}

func (b *bot) gallopRace(ctx context.Context, chat types.JID, round *raceRound) {
	nicknames := b.ChatTags(ctx, chat)
	mentions := round.players()
	board := func(frame int) string { return core.TagNicknames(round.board(frame), nicknames) }
	race, err := b.SendRawMentions(ctx, chat, board(0), mentions)
	if err != nil {
		b.Client.Log.Errorf("mandar a pista da !corrida: %v", err)
	}
	tick := time.NewTicker(raceTick)
	defer tick.Stop()
	last := len(round.frames) - 1
	for frame := 1; frame < last; frame++ {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			b.editRace(ctx, chat, race, board(frame), mentions)
		}
	}
	select {
	case <-ctx.Done():
		return
	case <-tick.C:
	}
	b.payRace(ctx, chat, round)
	b.payRescues(ctx, chat, "!corrida", round.rescues())
	result := board(last)
	b.editRace(ctx, chat, race, result, mentions)
	if _, err := b.SendRawMentions(ctx, chat, result, mentions); err != nil {
		b.Client.Log.Errorf("mandar resultado da !corrida: %v", err)
	}
	b.tallyRound(ctx, chat, "!corrida", round.settled())
}

func (b *bot) payRace(ctx context.Context, chat types.JID, round *raceRound) {
	for _, bet := range round.finish() {
		if err := b.Wallets.payPrize(ctx, chat, bet.acc, round.payout(bet)); err != nil {
			b.Client.Log.Errorf("pagar a !corrida pra %s: %v", bet.acc.JID, err)
		}
	}
}

func (b *bot) editRace(ctx context.Context, chat types.JID, race types.MessageID, text string, mentions []types.JID) {
	if race == "" {
		return
	}
	if _, err := b.Client.SendMessage(ctx, chat, b.Client.BuildEdit(chat, race, core.MentionMessage(text, mentions))); err != nil {
		b.Client.Log.Warnf("editar a pista da !corrida: %v", err)
	}
}

func newRaceRound() *raceRound {
	names := slices.Clone(raceNames)
	rand.Shuffle(len(names), func(i, j int) { names[i], names[j] = names[j], names[i] })
	weights := make([]float64, raceHorses)
	var total float64
	for i := range weights {
		weights[i] = 1 + 4*rand.Float64()
		total += weights[i]
	}
	horses := make([]raceHorse, raceHorses)
	for i, weight := range weights {
		chance := weight / total
		horses[i] = raceHorse{name: names[i], chance: chance, odds: int64(raceHouse / chance)}
	}
	winner := pickRaceWinner(horses)
	return &raceRound{opened: time.Now(), horses: horses, winner: winner, frames: raceFrames(winner)}
}

func pickRaceWinner(horses []raceHorse) int {
	var total float64
	for _, horse := range horses {
		total += horse.chance
	}
	roll := rand.Float64() * total
	for i, horse := range horses {
		roll -= horse.chance
		if roll < 0 {
			return i
		}
	}
	return slices.IndexFunc(horses, func(horse raceHorse) bool { return horse.chance > 0 })
}

func raceFrames(winner int) [][]int {
	positions := make([]int, raceHorses)
	frames := [][]int{slices.Clone(positions)}
	for slices.Max(positions) < raceLength {
		for i := range positions {
			positions[i] += rand.IntN(3)
		}
		frames = append(frames, slices.Clone(positions))
	}
	finish := frames[len(frames)-1]
	leader := slices.Index(finish, slices.Max(finish))
	for _, frame := range frames {
		frame[leader], frame[winner] = frame[winner], frame[leader]
	}
	for i := range finish {
		if i != winner {
			finish[i] = min(finish[i], raceLength-1)
		}
	}
	return frames
}

func (r *raceRound) start() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.started = true
	r.rig()
	r.ends = time.Now().Add(raceTick * time.Duration(len(r.frames)-1))
	r.over = len(r.bets) == 0
	r.record()
	return !r.over
}

func (r *raceRound) rig() {
	for _, bet := range r.bets {
		if !bet.swaysRace(r.winner) {
			continue
		}
		others := slices.Clone(r.horses)
		others[r.winner].chance = 0
		r.winner = pickRaceWinner(others)
		r.frames = raceFrames(r.winner)
		return
	}
}

func (bet *raceBet) swaysRace(winner int) bool {
	if bet.horse == winner {
		return rand.Float64() < bet.boost.Risk(bet.stake)
	}
	return rand.Float64() < bet.boost.Favor(bet.stake)
}

func (r *raceRound) save() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.record()
}

func (r *raceRound) deadline() time.Time {
	if r.started {
		return r.ends
	}
	return r.opened.Add(raceWindow)
}

func (r *raceRound) saved(chat types.JID) raceSave {
	horses := make([]raceSaveHorse, len(r.horses))
	for i, horse := range r.horses {
		horses[i] = raceSaveHorse{Name: horse.name, Chance: horse.chance, Odds: horse.odds}
	}
	bets := make([]raceSaveBet, len(r.bets))
	for i, bet := range r.bets {
		bets[i] = raceSaveBet{Account: core.StoreAccount(bet.acc), Stake: bet.stake, Decree: bet.boost.decree, Luck: bet.boost.luck, Heat: bet.boost.heat, Solace: bet.boost.solace, Cheat: bet.boost.cheat, Rescued: bet.boost.rescued, Horse: bet.horse}
	}
	return raceSave{Chat: chat, Opened: r.opened, Started: r.started, Horses: horses, Winner: r.winner, Frames: r.frames, Bets: bets}
}

func (r *raceRound) players() []types.JID {
	r.mu.Lock()
	defer r.mu.Unlock()
	players := make([]types.JID, len(r.bets))
	for i, bet := range r.bets {
		players[i] = bet.acc.JID
	}
	return players
}

func (r *raceRound) finish() []*raceBet {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.over = true
	r.record()
	var winners []*raceBet
	for _, bet := range r.bets {
		if bet.horse == r.winner {
			winners = append(winners, bet)
		}
	}
	return winners
}

func (r *raceRound) settled() []settledBet {
	r.mu.Lock()
	defer r.mu.Unlock()
	bets := make([]settledBet, len(r.bets))
	for i, bet := range r.bets {
		bets[i] = settledBet{acc: bet.acc, stake: bet.stake}
		if bet.horse == r.winner {
			bets[i].payout = r.payout(bet)
		}
	}
	return bets
}

func (r *raceRound) rescues() []rescuedBet {
	r.mu.Lock()
	defer r.mu.Unlock()
	var rescued []rescuedBet
	for _, bet := range r.bets {
		if bet.horse != r.winner && bet.boost.rescued {
			rescued = append(rescued, rescuedBet{acc: bet.acc, stake: bet.stake})
		}
	}
	return rescued
}

func (r *raceRound) payout(bet *raceBet) int64 {
	return bet.boost.Pay(bet.stake, bet.stake*r.horses[bet.horse].odds/100)
}

func (r *raceRound) announcement() string {
	return fmt.Sprintf("🏇 *Corrida de cavalo!* Apostas abertas por %s, manda !corrida <valor> <cavalo> pra apostar\n_Quem tem mais chance paga menos_\n\n%s",
		formatSeconds(raceWindow), r.lineup())
}

func (r *raceRound) lineup() string {
	lines := make([]string, len(r.horses))
	for i, horse := range r.horses {
		lines[i] = fmt.Sprintf("%d. %s: paga %s", i+1, horse.name, crashX(horse.odds))
	}
	return strings.Join(lines, "\n")
}

func (r *raceRound) status() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.started {
		return "🏇 Já tem corrida rolando, espera ela acabar pra abrir outra"
	}
	return fmt.Sprintf("🏇 Tem corrida aberta com %d aposta(s), a largada é em %s. Manda !corrida <valor> <cavalo>\n\n%s",
		len(r.bets), formatSeconds(raceWindow-time.Since(r.opened)), r.lineup())
}

func (r *raceRound) entry(bet *raceBet) string {
	return fmt.Sprintf("✅ @%s apostou %s no %d (%s)", bet.acc.JID.User, Coins(bet.stake), bet.horse+1, r.horses[bet.horse].name)
}

func (r *raceRound) board(frame int) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	finished := frame == len(r.frames)-1
	header := "🏇 *E foi dada a largada!*"
	if finished {
		header = fmt.Sprintf("🏆 *%d. %s venceu!*", r.winner+1, r.horses[r.winner].name)
	}
	lines := []string{header, "", "```" + r.track(r.frames[frame]) + "```", ""}
	for _, bet := range r.bets {
		lines = append(lines, r.betLine(bet, finished))
	}
	return strings.Join(lines, "\n")
}

func (r *raceRound) track(positions []int) string {
	lanes := make([]string, len(r.horses))
	for i, horse := range r.horses {
		run := min(positions[i], raceLength)
		lanes[i] = fmt.Sprintf("%d 🏁%s🏇%s %s", i+1, strings.Repeat("·", raceLength-run), strings.Repeat("·", run), horse.name)
	}
	return strings.Join(lanes, "\n")
}

func (r *raceRound) betLine(bet *raceBet, finished bool) string {
	user := bet.acc.JID.User
	horse := r.horses[bet.horse].name
	if !finished {
		return fmt.Sprintf("🎟️ @%s tá no %s com %s", user, horse, Coins(bet.stake))
	}
	if bet.horse == r.winner {
		payout := r.payout(bet)
		return fmt.Sprintf("✅ @%s apostou no %s e levou %s%s", user, horse, Coins(payout), bet.boost.Tag(bet.stake, payout))
	}
	return fmt.Sprintf("💸 @%s perdeu %s no %s", user, Coins(bet.stake), horse) + savedBetNote(bet.boost)
}

func raceHorseArg(args []string) (int, bool) {
	if len(args) < 2 {
		return 0, false
	}
	horse, err := strconv.Atoi(args[1])
	return horse - 1, err == nil && horse >= 1 && horse <= raceHorses
}
