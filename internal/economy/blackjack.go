package economy

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
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
	blackjackKind      = "blackjack"
	blackjackIdle      = 2 * time.Minute
	blackjackTarget    = 21
	blackjackDealerMin = 17
	blackjackBusy      = "✋ Termina essa mão antes de começar outra"
	stakeChanged       = "Seu saldo mudou no meio da aposta e não dá mais 🤨 Confere no !saldo"
)

var (
	blackjackRanks = [...]string{"A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"}
	blackjackSuits = [...]string{"♠", "♥", "♦", "♣"}
)

type blackjackCard struct {
	rank int
	suit int
}

type blackjackHand struct {
	mu         sync.Mutex
	acc        core.Account
	stake      int64
	boost      PrizeBoost
	deck       []blackjackCard
	player     []blackjackCard
	dealer     []blackjackCard
	doubled    bool
	over       bool
	lastAction time.Time
	timer      *time.Timer
	record     func()
}

type blackjackSave struct {
	Chat       types.JID           `json:"chat"`
	Account    core.StoredAccount  `json:"account"`
	Stake      int64               `json:"stake"`
	Decree     int64               `json:"decree,omitempty"`
	Levy       *levySave           `json:"levy,omitempty"`
	Luck       int64               `json:"luck"`
	Heat       int64               `json:"heat"`
	Solace     int64               `json:"solace"`
	Cheat      int64               `json:"cheat,omitempty"`
	Rescued    bool                `json:"rescued"`
	Deck       []blackjackSaveCard `json:"deck"`
	Player     []blackjackSaveCard `json:"player"`
	Dealer     []blackjackSaveCard `json:"dealer"`
	Doubled    bool                `json:"doubled"`
	LastAction time.Time           `json:"last_action"`
}

type blackjackSaveCard struct {
	Rank int `json:"rank"`
	Suit int `json:"suit"`
}

type blackjackReport struct {
	acc    core.Account
	text   string
	stake  int64
	payout int64
	over   bool
}

type blackjackMove func(hand *blackjackHand) (note string, err error)

func (b *bot) blackjack(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	acc := b.Account(ctx, evt.Info.Sender)
	key := blackjackKey(chat, acc)
	arg := core.FoldLetters(core.FirstPlainArg(core.MessageText(evt.Message)))
	shortcuts := map[string]string{"c": "carta", "p": "parar", "d": "dobrar"}
	move, isMove := b.blackjackMoves(ctx, chat)[cmp.Or(shortcuts[arg], arg)]
	value, open := b.blackjackHands.Load(key)
	if isMove && !open {
		return b.SendText(ctx, chat, "Você não tem mão aberta 🃏 Manda !blackjack <valor> pra jogar")
	}
	if isMove {
		return b.playBlackjack(ctx, chat, key, value.(*blackjackHand), move)
	}
	if open && arg == "" {
		return b.showBlackjack(ctx, chat, value.(*blackjackHand), "")
	}
	if open {
		return b.showBlackjack(ctx, chat, value.(*blackjackHand), blackjackBusy)
	}
	if arg == "" {
		return b.SendText(ctx, chat, "🃏 Blackjack contra a banca: manda !blackjack 100 (ou metade, 10%, tudo) e depois carta, parar ou dobrar")
	}
	return b.dealBlackjack(ctx, chat, acc, arg)
}

func (b *bot) answerBlackjack(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	move, isMove := b.blackjackMoves(ctx, chat)[core.FoldLetters(strings.TrimSpace(core.MessageText(evt.Message)))]
	if !isMove {
		return nil
	}
	key := blackjackKey(chat, b.Account(ctx, evt.Info.Sender))
	value, open := b.blackjackHands.Load(key)
	if !open {
		return nil
	}
	return b.playBlackjack(ctx, chat, key, value.(*blackjackHand), move)
}

func (b *bot) blackjackMoves(ctx context.Context, chat types.JID) map[string]blackjackMove {
	return map[string]blackjackMove{
		"carta":  func(h *blackjackHand) (string, error) { h.hit(); return "", nil },
		"parar":  func(h *blackjackHand) (string, error) { h.stand(); return "", nil },
		"dobrar": func(h *blackjackHand) (string, error) { return b.doubleBlackjack(ctx, chat, h) },
	}
}

func (b *bot) dealBlackjack(ctx context.Context, chat types.JID, acc core.Account, arg string) error {
	bet, refusal, err := b.PlaceBet(ctx, chat, acc, arg, "!blackjack")
	if err != nil || bet.Busted {
		return err
	}
	if refusal != "" {
		return b.SendText(ctx, chat, refusal)
	}
	stake := bet.Stake
	held, err := b.holdStake(ctx, chat, acc, stake, bet.Boost.levy)
	if err != nil {
		return fmt.Errorf("apostar no !blackjack: %w", err)
	}
	if !held {
		return b.SendText(ctx, chat, stakeChanged)
	}

	key := blackjackKey(chat, acc)
	hand := b.trackBlackjack(ctx, chat, key, newBlackjackHand(acc, bet))
	report := hand.report(strings.TrimPrefix(BetLevyText(stake, bet.Boost.levy), "\n"))
	if report.over {
		return b.sendBlackjack(ctx, chat, report)
	}
	if value, loaded := b.blackjackHands.LoadOrStore(key, hand); loaded {
		return b.refundBlackjack(ctx, chat, acc, stake, value.(*blackjackHand))
	}
	hand.arm(blackjackIdle, func() { b.expireBlackjack(ctx, chat, key, hand) })
	return b.sendBlackjack(ctx, chat, report)
}

func (b *bot) trackBlackjack(ctx context.Context, chat types.JID, key string, hand *blackjackHand) *blackjackHand {
	hand.record = func() { b.recordBlackjack(ctx, chat, key, hand) }
	return hand
}

func (b *bot) recordBlackjack(ctx context.Context, chat types.JID, key string, hand *blackjackHand) {
	if hand.over {
		b.DropGame(ctx, blackjackKind, key)
		return
	}
	b.KeepGame(ctx, blackjackKind, key, hand.saved(chat), hand.lastAction.Add(blackjackIdle))
}

func (b *bot) restoreBlackjack(ctx context.Context, saved core.SavedGame) error {
	var state blackjackSave
	if err := json.Unmarshal(saved.State, &state); err != nil {
		return fmt.Errorf("ler a mão do !blackjack: %w", err)
	}
	chat, key := state.Chat, saved.Key
	hand := b.trackBlackjack(ctx, chat, key, &blackjackHand{
		acc:        state.Account.Account(),
		stake:      state.Stake,
		boost:      PrizeBoost{decree: state.Decree, levy: state.Levy.levy(), luck: state.Luck, heat: state.Heat, solace: state.Solace, cheat: state.Cheat, rescued: state.Rescued},
		deck:       loadBlackjackCards(state.Deck),
		player:     loadBlackjackCards(state.Player),
		dealer:     loadBlackjackCards(state.Dealer),
		doubled:    state.Doubled,
		lastAction: state.LastAction,
	})
	if _, loaded := b.blackjackHands.LoadOrStore(key, hand); loaded {
		return nil
	}
	hand.arm(blackjackIdle-time.Since(hand.lastAction), func() { b.expireBlackjack(ctx, chat, key, hand) })
	return nil
}

func (b *bot) refundBlackjack(ctx context.Context, chat types.JID, acc core.Account, stake int64, open *blackjackHand) error {
	if _, err := b.payStake(ctx, chat, acc, stake); err != nil {
		return fmt.Errorf("devolver aposta repetida do !blackjack: %w", err)
	}
	return b.showBlackjack(ctx, chat, open, blackjackBusy)
}

func (b *bot) doubleBlackjack(ctx context.Context, chat types.JID, h *blackjackHand) (string, error) {
	if len(h.player) > 2 {
		return "✋ Só dá pra dobrar com as duas primeiras cartas", nil
	}
	held, err := b.holdStake(ctx, chat, h.acc, h.stake, h.boost.levy)
	if err != nil {
		return "", fmt.Errorf("dobrar no !blackjack: %w", err)
	}
	if !held {
		return "🪹 Você não tem " + Coins(h.stake+h.boost.levy.cut(h.stake)) + " pra dobrar", nil
	}
	note := strings.TrimPrefix(BetLevyText(h.stake, h.boost.levy), "\n")
	h.stake *= 2
	h.doubled = true
	h.player = append(h.player, h.draw())
	h.stand()
	return note, nil
}

func (b *bot) playBlackjack(ctx context.Context, chat types.JID, key string, hand *blackjackHand, move blackjackMove) error {
	report, err := hand.play(move)
	if err != nil {
		return err
	}
	if report.over {
		b.blackjackHands.CompareAndDelete(key, hand)
	}
	return b.sendBlackjack(ctx, chat, report)
}

func (b *bot) expireBlackjack(ctx context.Context, chat types.JID, key string, hand *blackjackHand) {
	if ctx.Err() != nil {
		return
	}
	report := hand.timeout()
	if !report.over {
		return
	}
	b.blackjackHands.CompareAndDelete(key, hand)
	if err := b.sendBlackjack(ctx, chat, report); err != nil {
		b.Client.Log.Errorf("encerrar !blackjack abandonado: %v", err)
	}
}

func (b *bot) showBlackjack(ctx context.Context, chat types.JID, hand *blackjackHand, note string) error {
	text := hand.view(note)
	if text == "" {
		return nil
	}
	_, err := b.SendMentions(ctx, chat, text, []types.JID{hand.acc.JID})
	return err
}

func (b *bot) sendBlackjack(ctx context.Context, chat types.JID, report blackjackReport) error {
	if report.text == "" {
		return nil
	}
	mentions := []types.JID{report.acc.JID}
	if !report.over {
		_, err := b.SendMentions(ctx, chat, report.text, mentions)
		return err
	}
	balance, err := b.payStake(ctx, chat, report.acc, report.payout)
	if err != nil {
		return fmt.Errorf("pagar !blackjack: %w", err)
	}
	if _, err := b.SendMentions(ctx, chat, report.text+"\n💰 Saldo: "+Coins(balance), mentions); err != nil {
		return err
	}
	b.tallyBet(ctx, chat, report.acc, "!blackjack", report.stake, report.payout, balance)
	return nil
}

func (b *bot) holdStake(ctx context.Context, chat types.JID, acc core.Account, stake int64, levy PixLevy) (bool, error) {
	_, err := b.Wallets.SettleBet(ctx, chat, acc, stake, 0, levy)
	if errors.Is(err, ErrNoFunds) {
		return false, nil
	}
	return err == nil, err
}

func (b *bot) payStake(ctx context.Context, chat types.JID, acc core.Account, payout int64) (int64, error) {
	if err := b.Wallets.payPrize(ctx, chat, acc, payout); err != nil {
		return 0, err
	}
	balance, _, err := b.Wallets.Balance(ctx, chat, acc)
	return balance, err
}

func blackjackKey(chat types.JID, acc core.Account) string {
	return chat.String() + "|" + acc.Key
}

func newBlackjackHand(acc core.Account, bet wager) *blackjackHand {
	h := &blackjackHand{acc: acc, stake: bet.Stake, boost: bet.Boost, deck: newBlackjackDeck(), lastAction: time.Now()}
	h.player = []blackjackCard{h.draw(), h.draw()}
	h.dealer = []blackjackCard{h.draw(), h.draw()}
	h.over = isNatural(h.player) || isNatural(h.dealer)
	return h
}

func newBlackjackDeck() []blackjackCard {
	deck := make([]blackjackCard, 0, len(blackjackRanks)*len(blackjackSuits))
	for suit := range blackjackSuits {
		for rank := range blackjackRanks {
			deck = append(deck, blackjackCard{rank: rank, suit: suit})
		}
	}
	rand.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	return deck
}

func (h *blackjackHand) arm(wait time.Duration, expire func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.timer = time.AfterFunc(wait, expire)
	h.record()
}

func (h *blackjackHand) saved(chat types.JID) blackjackSave {
	return blackjackSave{
		Chat:       chat,
		Account:    core.StoreAccount(h.acc),
		Stake:      h.stake,
		Decree:     h.boost.decree,
		Levy:       h.boost.levy.saved(),
		Luck:       h.boost.luck,
		Heat:       h.boost.heat,
		Solace:     h.boost.solace,
		Cheat:      h.boost.cheat,
		Rescued:    h.boost.rescued,
		Deck:       saveBlackjackCards(h.deck),
		Player:     saveBlackjackCards(h.player),
		Dealer:     saveBlackjackCards(h.dealer),
		Doubled:    h.doubled,
		LastAction: h.lastAction,
	}
}

func saveBlackjackCards(cards []blackjackCard) []blackjackSaveCard {
	saved := make([]blackjackSaveCard, len(cards))
	for i, c := range cards {
		saved[i] = blackjackSaveCard{Rank: c.rank, Suit: c.suit}
	}
	return saved
}

func loadBlackjackCards(saved []blackjackSaveCard) []blackjackCard {
	cards := make([]blackjackCard, len(saved))
	for i, c := range saved {
		cards[i] = blackjackCard{rank: c.Rank, suit: c.Suit}
	}
	return cards
}

func (h *blackjackHand) view(note string) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.over {
		return ""
	}
	return h.report(note).text
}

func (h *blackjackHand) play(move blackjackMove) (blackjackReport, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.over {
		return blackjackReport{}, nil
	}
	h.lastAction = time.Now()
	note, err := move(h)
	if err != nil {
		return blackjackReport{}, err
	}
	h.record()
	return h.report(note), nil
}

func (h *blackjackHand) timeout() blackjackReport {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.over {
		return blackjackReport{}
	}
	if wait := blackjackIdle - time.Since(h.lastAction); wait > 0 {
		h.timer.Reset(wait)
		return blackjackReport{}
	}
	h.stand()
	h.record()
	return h.report(fmt.Sprintf("⏰ @%s sumiu por %s, então eu parei a mão", h.acc.JID.User, core.FormatWait(blackjackIdle)))
}

func (h *blackjackHand) draw() blackjackCard {
	card := h.deck[len(h.deck)-1]
	h.deck = h.deck[:len(h.deck)-1]
	return card
}

func (h *blackjackHand) hit() {
	h.player = append(h.player, h.draw())
	if total, _ := blackjackTotal(h.player); total >= blackjackTarget {
		h.stand()
	}
}

func (h *blackjackHand) stand() {
	h.over = true
	if h.timer != nil {
		h.timer.Stop()
	}
	if total, _ := blackjackTotal(h.player); total > blackjackTarget {
		return
	}
	for total, _ := blackjackTotal(h.dealer); total < blackjackDealerMin; total, _ = blackjackTotal(h.dealer) {
		h.dealer = append(h.dealer, h.dealerDraw())
	}
}

func (h *blackjackHand) dealerDraw() blackjackCard {
	busts := h.busts(h.deck[len(h.deck)-1])
	if busts && rand.Float64() < h.boost.Risk(h.stake) {
		h.stackDeck(false)
	}
	if !busts && rand.Float64() < h.boost.Favor(h.stake) {
		h.stackDeck(true)
	}
	return h.draw()
}

func (h *blackjackHand) stackDeck(bust bool) {
	top := len(h.deck) - 1
	if i := slices.IndexFunc(h.deck, func(c blackjackCard) bool { return h.busts(c) == bust }); i >= 0 {
		h.deck[i], h.deck[top] = h.deck[top], h.deck[i]
	}
}

func (h *blackjackHand) busts(card blackjackCard) bool {
	total, _ := blackjackTotal(append(slices.Clip(h.dealer), card))
	return total > blackjackTarget
}

func (h *blackjackHand) report(note string) blackjackReport {
	stake := Coins(h.stake)
	if h.doubled {
		stake += " (dobrou)"
	}
	dealer := strings.TrimSpace(strings.Repeat("🂠 ", len(h.dealer)))
	if h.over {
		dealer = showCards(h.dealer) + " (" + h.score(h.dealer) + ")"
	}
	board := fmt.Sprintf("🃏 *Blackjack* de @%s · valendo %s\n\n🏦 Banca: %s\n🙋 Mão: %s (%s)",
		h.acc.JID.User, stake, dealer, showCards(h.player), h.score(h.player))
	if note != "" {
		board = note + "\n\n" + board
	}
	if !h.over {
		return blackjackReport{acc: h.acc, text: board + "\n\n👉 Manda " + h.choices()}
	}
	payout, result := h.outcome()
	saved := h.boost.Rescue(h.stake, payout)
	return blackjackReport{acc: h.acc, text: board + "\n\n" + result + SavedBetText(saved), stake: h.stake, payout: payout + saved, over: true}
}

func (h *blackjackHand) choices() string {
	if len(h.player) > 2 {
		return "*carta* ou *parar*"
	}
	return "*carta*, *parar* ou *dobrar*"
}

func (h *blackjackHand) score(cards []blackjackCard) string {
	total, soft := blackjackTotal(cards)
	if h.over || !soft || total == blackjackTarget {
		return strconv.Itoa(total)
	}
	return fmt.Sprintf("%d/%d", total-10, total)
}

func (h *blackjackHand) outcome() (int64, string) {
	player, _ := blackjackTotal(h.player)
	dealer, _ := blackjackTotal(h.dealer)
	playerNatural, dealerNatural := isNatural(h.player), isNatural(h.dealer)
	outcomes := []struct {
		happened bool
		payout   int64
		text     string
	}{
		{playerNatural && dealerNatural, h.stake, "🤝 Os dois com blackjack, @%s pegou de volta %s"},
		{playerNatural, h.stake + h.stake*3/2, "🃏 *BLACKJACK!* @%s ganhou %s"},
		{dealerNatural, 0, "🃏 A banca tinha blackjack. @%s perdeu %s"},
		{player > blackjackTarget, 0, "💥 Estourou! @%s perdeu %s"},
		{dealer > blackjackTarget, 2 * h.stake, "🎉 A banca estourou! @%s ganhou %s"},
		{player > dealer, 2 * h.stake, "🏆 @%s ganhou %s"},
		{player == dealer, h.stake, "🤝 Empate, @%s pegou de volta %s"},
	}
	for _, o := range outcomes {
		if !o.happened {
			continue
		}
		payout := h.boost.Pay(h.stake, o.payout)
		shown := cmp.Or(max(payout-h.stake, h.stake-payout), h.stake)
		return payout, fmt.Sprintf(o.text, h.acc.JID.User, Coins(shown)) + h.boost.Tag(h.stake, payout)
	}
	return 0, fmt.Sprintf("💸 A banca ganhou. @%s perdeu %s", h.acc.JID.User, Coins(h.stake))
}

func (c blackjackCard) String() string {
	return blackjackRanks[c.rank] + blackjackSuits[c.suit]
}

func (c blackjackCard) points() int {
	return min(c.rank+1, 10)
}

func blackjackTotal(cards []blackjackCard) (total int, soft bool) {
	ace := false
	for _, c := range cards {
		total += c.points()
		ace = ace || c.rank == 0
	}
	if ace && total+10 <= blackjackTarget {
		return total + 10, true
	}
	return total, false
}

func isNatural(cards []blackjackCard) bool {
	total, _ := blackjackTotal(cards)
	return len(cards) == 2 && total == blackjackTarget
}

func showCards(cards []blackjackCard) string {
	shown := make([]string, len(cards))
	for i, c := range cards {
		shown[i] = c.String()
	}
	return strings.Join(shown, " ")
}
