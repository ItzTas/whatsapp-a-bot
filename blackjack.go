package main

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"sync"
	"time"

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
	acc        account
	stake      int64
	boost      prizeBoost
	levy       heldLevy
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
	Account    storedAccount       `json:"account"`
	Stake      int64               `json:"stake"`
	Boost      int64               `json:"boost"`
	Luck       int64               `json:"luck"`
	Rescued    bool                `json:"rescued"`
	Levy       heldLevySave        `json:"levy"`
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
	acc    account
	text   string
	payout int64
	levy   prizeLevy
	over   bool
}

type blackjackMove func(hand *blackjackHand) (note string, err error)

func (b *bot) blackjack(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	acc := b.account(ctx, evt.Info.Sender)
	key := blackjackKey(chat, acc)
	arg := foldLetters(firstPlainArg(messageText(evt.Message)))
	shortcuts := map[string]string{"c": "carta", "p": "parar", "d": "dobrar"}
	move, isMove := b.blackjackMoves(ctx, chat)[cmp.Or(shortcuts[arg], arg)]
	value, open := b.blackjackHands.Load(key)
	if isMove && !open {
		return b.sendText(ctx, chat, "Você não tem mão aberta 🃏 Manda !blackjack <valor> pra jogar")
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
		return b.sendText(ctx, chat, "🃏 Blackjack contra a banca: manda !blackjack 100 (ou metade, 10%, tudo) e depois carta, parar ou dobrar")
	}
	return b.dealBlackjack(ctx, chat, acc, arg)
}

func (b *bot) answerBlackjack(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	move, isMove := b.blackjackMoves(ctx, chat)[foldLetters(strings.TrimSpace(messageText(evt.Message)))]
	if !isMove {
		return nil
	}
	key := blackjackKey(chat, b.account(ctx, evt.Info.Sender))
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

func (b *bot) dealBlackjack(ctx context.Context, chat types.JID, acc account, arg string) error {
	bet, refusal, err := b.placeBet(ctx, chat, acc, arg, "!blackjack")
	if err != nil {
		return err
	}
	if refusal != "" {
		return b.sendText(ctx, chat, refusal)
	}
	stake := bet.stake
	held, err := b.holdBet(ctx, chat, acc, stake, bet.levy)
	if err != nil {
		return fmt.Errorf("apostar no !blackjack: %w", err)
	}
	if !held {
		return b.sendText(ctx, chat, stakeChanged)
	}

	key := blackjackKey(chat, acc)
	hand := b.trackBlackjack(ctx, chat, key, newBlackjackHand(acc, bet))
	report := hand.report("")
	report.text += betLevyText(stake, 0, bet.levy)
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
		b.dropGame(ctx, blackjackKind, key)
		return
	}
	b.keepGame(ctx, blackjackKind, key, hand.saved(chat), hand.lastAction.Add(blackjackIdle))
}

func (b *bot) restoreBlackjack(ctx context.Context, saved savedGame) error {
	var state blackjackSave
	if err := json.Unmarshal(saved.state, &state); err != nil {
		return fmt.Errorf("ler a mão do !blackjack: %w", err)
	}
	chat, key := state.Chat, saved.key
	hand := b.trackBlackjack(ctx, chat, key, &blackjackHand{
		acc:        state.Account.account(),
		stake:      state.Stake,
		boost:      prizeBoost{decree: state.Boost, luck: state.Luck, rescued: state.Rescued},
		levy:       state.Levy.held(),
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

func (b *bot) refundBlackjack(ctx context.Context, chat types.JID, acc account, stake int64, open *blackjackHand) error {
	if _, err := b.payStake(ctx, chat, acc, stake, prizeLevy{}); err != nil {
		return fmt.Errorf("devolver aposta repetida do !blackjack: %w", err)
	}
	return b.showBlackjack(ctx, chat, open, blackjackBusy)
}

func (b *bot) doubleBlackjack(ctx context.Context, chat types.JID, h *blackjackHand) (string, error) {
	if len(h.player) > 2 {
		return "✋ Só dá pra dobrar com as duas primeiras cartas", nil
	}
	held, err := b.holdStake(ctx, chat, h.acc, h.stake)
	if err != nil {
		return "", fmt.Errorf("dobrar no !blackjack: %w", err)
	}
	if !held {
		return "🪹 Você não tem " + coins(h.stake) + " pra dobrar", nil
	}
	h.stake *= 2
	h.doubled = true
	h.player = append(h.player, h.draw())
	h.stand()
	return "", nil
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
		b.client.Log.Errorf("encerrar !blackjack abandonado: %v", err)
	}
}

func (b *bot) showBlackjack(ctx context.Context, chat types.JID, hand *blackjackHand, note string) error {
	text := hand.view(note)
	if text == "" {
		return nil
	}
	_, err := b.sendMentions(ctx, chat, text, []types.JID{hand.acc.jid})
	return err
}

func (b *bot) sendBlackjack(ctx context.Context, chat types.JID, report blackjackReport) error {
	if report.text == "" {
		return nil
	}
	mentions := []types.JID{report.acc.jid}
	if !report.over {
		_, err := b.sendMentions(ctx, chat, report.text, mentions)
		return err
	}
	balance, err := b.payStake(ctx, chat, report.acc, report.payout, report.levy)
	if err != nil {
		return fmt.Errorf("pagar !blackjack: %w", err)
	}
	_, err = b.sendMentions(ctx, chat, report.text+"\n💰 Saldo: "+coins(balance), mentions)
	return err
}

func (b *bot) holdStake(ctx context.Context, chat types.JID, acc account, stake int64) (bool, error) {
	return b.holdBet(ctx, chat, acc, stake, pixLevy{})
}

func (b *bot) holdBet(ctx context.Context, chat types.JID, acc account, stake int64, levy pixLevy) (bool, error) {
	_, err := b.wallets.settleBet(ctx, chat, acc, stake, 0, levy)
	if errors.Is(err, errNoFunds) {
		return false, nil
	}
	return err == nil, err
}

func (b *bot) payStake(ctx context.Context, chat types.JID, acc account, payout int64, levy prizeLevy) (int64, error) {
	if err := b.wallets.payPrize(ctx, chat, acc, payout, levy); err != nil {
		return 0, err
	}
	balance, _, err := b.wallets.balance(ctx, chat, acc)
	return balance, err
}

func blackjackKey(chat types.JID, acc account) string {
	return chat.String() + "|" + acc.key
}

func newBlackjackHand(acc account, bet wager) *blackjackHand {
	h := &blackjackHand{acc: acc, stake: bet.stake, boost: bet.boost, levy: bet.held(), deck: newBlackjackDeck(), lastAction: time.Now()}
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
		Account:    storeAccount(h.acc),
		Stake:      h.stake,
		Boost:      h.boost.decree,
		Luck:       h.boost.luck,
		Rescued:    h.boost.rescued,
		Levy:       h.levy.saved(),
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
	return h.report(fmt.Sprintf("⏰ @%s sumiu por %s, então eu parei a mão", h.acc.jid.User, formatWait(blackjackIdle)))
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
		h.dealer = append(h.dealer, h.draw())
	}
}

func (h *blackjackHand) report(note string) blackjackReport {
	stake := coins(h.stake)
	if h.doubled {
		stake += " (dobrou)"
	}
	dealer := strings.TrimSpace(strings.Repeat("🂠 ", len(h.dealer)))
	if h.over {
		dealer = showCards(h.dealer) + " (" + h.score(h.dealer) + ")"
	}
	board := fmt.Sprintf("🃏 *Blackjack* de @%s · valendo %s\n\n🏦 Banca: %s\n🙋 Mão: %s (%s)",
		h.acc.jid.User, stake, dealer, showCards(h.player), h.score(h.player))
	if note != "" {
		board = note + "\n\n" + board
	}
	if !h.over {
		return blackjackReport{acc: h.acc, text: board + "\n\n👉 Manda " + h.choices()}
	}
	payout, result := h.outcome()
	saved := h.boost.rescue(h.stake, payout)
	levy := h.levy.onPrize(h.stake, payout)
	return blackjackReport{acc: h.acc, text: board + "\n\n" + result + levy.text() + savedBetText(saved), payout: payout + saved, levy: levy, over: true}
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
		payout := h.boost.pay(h.stake, o.payout)
		shown := cmp.Or(max(payout-h.stake, h.stake-payout), h.stake)
		return payout, fmt.Sprintf(o.text, h.acc.jid.User, coins(shown)) + h.boost.tag(h.stake, payout)
	}
	return 0, fmt.Sprintf("💸 A banca ganhou. @%s perdeu %s", h.acc.jid.User, coins(h.stake))
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
