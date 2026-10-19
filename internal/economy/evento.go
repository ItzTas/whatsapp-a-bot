package economy

import (
	"bytes"
	"context"
	"crypto/sha256"
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
	"golang.org/x/sync/errgroup"
)

const (
	eventKind         = "evento"
	eventChance       = 0.06
	eventIdle         = 5 * time.Minute
	eventMemory       = 15
	eventChatMemory   = 3
	eventQuestion     = "🤔 E agora, o que você faz?"
	eventRestoreGrace = 30 * time.Second

	universalEventWeight = 15
	jobEventWeight       = 200
)

type eventRarity int

const (
	commonEvent eventRarity = iota
	rareEvent
	legendaryEvent
)

func (r eventRarity) weight() int {
	return [...]int{commonEvent: 100, rareEvent: 30, legendaryEvent: 8}[r]
}

func (r eventRarity) badge() string {
	return [...]string{commonEvent: "", rareEvent: " 🔵 *Raro!*", legendaryEvent: " 🟡 *Lendário!*"}[r]
}

type risk int

const (
	lowRisk risk = iota
	midRisk
	highRisk
)

func (r risk) badge() string {
	return [...]string{lowRisk: "🟢", midRisk: "🟡", highRisk: "🔴"}[r]
}

type eventBook struct {
	name      string
	places    []string
	scenarios []scenario
}

type scenario struct {
	id      string
	title   string
	rarity  eventRarity
	places  []string
	areas   []int
	jobs    []string
	stage   bool
	when    func(c eventCase) bool
	intros  []string
	choices []choice
	idle    ending
}

type choice struct {
	label   string
	risk    risk
	needs   func(c eventCase) bool
	uses    string
	price   cost
	endings []ending
}

type cost struct {
	percent int64
	floor   int64
}

type ending struct {
	weight  int
	favor   func(p Profession) float64
	texts   []string
	effects []effect
	next    string
}

type eventSpot struct {
	place  string
	won    bool
	victim types.JID
}

type eventCase struct {
	eventSpot
	job     Profession
	bag     []stack
	balance int64
	health  int64
	now     time.Time
}

type eventPast struct {
	recent  []string
	planted []string
}

type listedScenario struct {
	scenario
	book string
}

var eventBooks = []eventBook{
	generalEvents, fishingEvents, miningEvents, scavengingEvents, buggingEvents,
	workEvents, craftEvents, professionEvents, crimeEvents, fameEvents, gamblingEvents, rareEvents,
}

var eventCatalog = listScenarios(eventBooks)

func listScenarios(books []eventBook) []listedScenario {
	var listed []listedScenario
	for _, book := range books {
		for _, sc := range book.scenarios {
			if len(sc.places) == 0 {
				sc.places = book.places
			}
			listed = append(listed, listedScenario{scenario: sc, book: book.name})
		}
	}
	return listed
}

func scenarioByID(id string) (listedScenario, bool) {
	i := slices.IndexFunc(eventCatalog, func(sc listedScenario) bool { return sc.id == id })
	if i < 0 {
		return listedScenario{}, false
	}
	return eventCatalog[i], true
}

func (sc scenario) fits(c eventCase) bool {
	return (len(sc.places) == 0 || slices.Contains(sc.places, c.place)) &&
		(len(sc.areas) == 0 || slices.Contains(sc.areas, c.job.area)) &&
		(len(sc.jobs) == 0 || slices.Contains(sc.jobs, c.job.name)) &&
		(sc.when == nil || sc.when(c)) &&
		len(sc.open(c)) >= 2
}

func (sc scenario) open(c eventCase) []int {
	var open []int
	for i, ch := range sc.choices {
		if ch.available(c) {
			open = append(open, i)
		}
	}
	return open
}

func (sc scenario) timeout() ending {
	if len(sc.idle.texts) > 0 {
		return sc.idle
	}
	return ending{texts: []string{"{@} ficou parado pensando na vida, e a chance passou"}}
}

func (ch choice) available(c eventCase) bool {
	return (ch.needs == nil || ch.needs(c)) &&
		(ch.uses == "" || c.has(ch.uses)) &&
		(ch.price == cost{} || c.balance >= ch.price.of(c.balance))
}

func (ch choice) option(price int64) string {
	parts := []string{ch.label}
	if price > 0 {
		parts = append(parts, "("+Coins(price)+")")
	}
	if it, ok := itemByID(ch.uses); ok {
		parts = append(parts, "(gasta "+it.name+")")
	}
	return strings.Join(append(parts, ch.risk.badge()), " ")
}

func (ch choice) roll(job Profession) ending {
	if len(ch.endings) == 0 {
		return ending{}
	}
	weights := ch.weights(job)
	if slices.Max(weights) == 0 {
		return ch.endings[0]
	}
	return ch.endings[weightedIndex(len(weights), func(i int) int { return weights[i] })]
}

func (e ending) lean(job Profession) int {
	if e.favor == nil {
		return 0
	}
	return int(e.favor(job) * 100)
}

func (p cost) of(balance int64) int64 {
	return max(balance*p.percent/100, p.floor)
}

func (p Profession) shiftValue() int64 {
	return (p.min + p.max) / 2
}

func (c eventCase) has(id string) bool {
	return slices.ContainsFunc(c.bag, func(s stack) bool { return s.item.id == id })
}

func holds(id string) func(c eventCase) bool {
	return func(c eventCase) bool { return c.has(id) }
}

func inArea(areas ...int) func(c eventCase) bool {
	return func(c eventCase) bool { return slices.Contains(areas, c.job.area) }
}

func isJob(names ...string) func(c eventCase) bool {
	return func(c eventCase) bool { return slices.Contains(names, c.job.name) }
}

func either(conds ...func(c eventCase) bool) func(c eventCase) bool {
	return func(c eventCase) bool {
		return slices.ContainsFunc(conds, func(cond func(c eventCase) bool) bool { return cond(c) })
	}
}

func armedUp(c eventCase) bool {
	return c.job.weapon != ""
}

func lawman(c eventCase) bool {
	return c.job.jailTime > 0
}

func killer(c eventCase) bool {
	return c.job.killChance > 0
}

func stoned(c eventCase) bool {
	return c.job.drug != ""
}

func bruised(c eventCase) bool {
	return c.health < MaxHealth*2/3
}

func atNight(c eventCase) bool {
	return werewolfHour(c.now)
}

func onSunday(c eventCase) bool {
	return c.now.In(core.BrazilTime).Weekday() == time.Sunday
}

func wentWell(c eventCase) bool {
	return c.won
}

func wentWrong(c eventCase) bool {
	return !c.won
}

func inCharge(c eventCase) bool {
	return c.job.president
}

func homeOwner(c eventCase) bool {
	return c.job.estate > 0
}

type sceneBoard struct {
	pending  sync.Map
	messages sync.Map
	mu       sync.Mutex
	recent   map[types.JID][]string
}

func (s *sceneBoard) shown(chat types.JID) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.recent[chat])
}

func (s *sceneBoard) remember(chat types.JID, id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.recent == nil {
		s.recent = map[types.JID][]string{}
	}
	shown := append(slices.Clone(s.recent[chat]), id)
	s.recent[chat] = shown[max(len(shown)-eventChatMemory, 0):]
}

type pendingEvent struct {
	mu       sync.Mutex
	key      string
	chat     types.JID
	acc      core.Account
	ids      []types.JID
	victim   core.Account
	scenario string
	spot     eventSpot
	stake    int64
	options  []int
	prices   []int64
	labels   []string
	messages []types.MessageID
	expires  time.Time
	foreseen bool
	timer    *time.Timer
	closed   bool
}

type eventSave struct {
	Chat     types.JID          `json:"chat"`
	Account  core.StoredAccount `json:"account"`
	IDs      []types.JID        `json:"ids"`
	Victim   core.StoredAccount `json:"victim"`
	Scenario string             `json:"scenario"`
	Place    string             `json:"place"`
	Won      bool               `json:"won"`
	Target   types.JID          `json:"target"`
	Stake    int64              `json:"stake"`
	Options  []int              `json:"options"`
	Prices   []int64            `json:"prices"`
	Labels   []string           `json:"labels"`
	Messages []types.MessageID  `json:"messages"`
	Expires  time.Time          `json:"expires"`
}

type eventPlay struct {
	choice *choice
	price  int64
	label  string
	ending ending
}

func eventKey(chat types.JID, acc core.Account) string {
	return chat.String() + "|" + acc.Key
}

func eventLuck() bool {
	return rand.Float64() < eventChance*oracleEventFactor
}

func (b *bot) stirEvent(ctx context.Context, evt *events.Message, spot eventSpot) {
	if !eventLuck() {
		return
	}
	chat := evt.Info.Chat.ToNonAD()
	b.stirEventFor(ctx, chat, b.Account(ctx, evt.Info.Sender), b.SenderIdentities(ctx, evt), spot)
}

func (b *bot) stirCrewEvent(ctx context.Context, chat types.JID, crew []core.Account, spot eventSpot) {
	if len(crew) == 0 || !eventLuck() {
		return
	}
	acc := core.PickOne(crew)
	b.stirEventFor(ctx, chat, acc, b.Identities(ctx, acc.JID), spot)
}

func uncaught(robbers []robber) []core.Account {
	var free []core.Account
	for _, r := range robbers {
		if !r.caught && !r.injury.died {
			free = append(free, r.acc)
		}
	}
	return free
}

func (b *bot) stirEventFor(ctx context.Context, chat types.JID, acc core.Account, ids []types.JID, spot eventSpot) {
	if _, busy := b.scenes.pending.Load(eventKey(chat, acc)); busy {
		return
	}
	if err := b.openEvent(ctx, chat, acc, ids, spot); err != nil {
		b.Client.Log.Errorf("abrir evento de %s em %s: %v", acc.JID, chat, err)
	}
}

func (b *bot) openEvent(ctx context.Context, chat types.JID, acc core.Account, ids []types.JID, spot eventSpot) error {
	c, past, err := b.eventCaseOf(ctx, chat, acc, spot)
	if err != nil || !c.job.catchesEvent() {
		return err
	}
	sc, ok := b.pickScenario(chat, c, past)
	if !ok {
		return nil
	}
	ev := newPendingEvent(chat, acc, ids, sc, c)
	if !spot.victim.IsEmpty() {
		ev.victim = b.Account(ctx, spot.victim)
	}
	if _, taken := b.scenes.pending.LoadOrStore(ev.key, ev); taken {
		return nil
	}
	if err := b.Wallets.noteEvent(ctx, chat, acc, sc.id, c.now); err != nil {
		b.scenes.pending.CompareAndDelete(ev.key, ev)
		return fmt.Errorf("anotar o evento: %w", err)
	}
	b.scenes.remember(chat, sc.id)
	return b.showEvent(ctx, ev, sc)
}

func (b *bot) eventCaseOf(ctx context.Context, chat types.JID, acc core.Account, spot eventSpot) (eventCase, eventPast, error) {
	c := eventCase{eventSpot: spot, now: time.Now()}
	var past eventPast
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { c.job, err = b.ProfessionOf(gctx, chat, acc); return err })
	g.Go(func() (err error) { c.bag, err = b.Wallets.inventory(gctx, chat, acc); return err })
	g.Go(func() (err error) { c.balance, _, err = b.Wallets.Balance(gctx, chat, acc); return err })
	g.Go(func() (err error) { c.health, err = b.Wallets.Health(gctx, chat, acc, c.now); return err })
	g.Go(func() (err error) { past, err = b.Wallets.eventPast(gctx, chat, acc, c.now); return err })
	if err := g.Wait(); err != nil {
		return eventCase{}, eventPast{}, fmt.Errorf("ler a situação do evento: %w", err)
	}
	return c, past, nil
}

func (b *bot) pickScenario(chat types.JID, c eventCase, past eventPast) (listedScenario, bool) {
	for _, id := range past.planted {
		if sc, ok := scenarioByID(id); ok && sc.fits(c) {
			return sc, true
		}
	}
	shown := b.scenes.shown(chat)
	pool := eligibleScenarios(c, func(sc listedScenario) bool {
		return !slices.Contains(shown, sc.id) && !slices.Contains(past.recent, sc.id)
	})
	if len(pool) == 0 {
		pool = eligibleScenarios(c, func(sc listedScenario) bool { return !slices.Contains(shown, sc.id) })
	}
	if len(pool) == 0 {
		return listedScenario{}, false
	}
	return pool[weightedIndex(len(pool), func(i int) int { return pool[i].weight() })], true
}

func (sc listedScenario) weight() int {
	if len(sc.jobs) > 0 {
		return sc.rarity.weight() * jobEventWeight / 100
	}
	if len(sc.places) == 0 && len(sc.areas) == 0 {
		return max(sc.rarity.weight()*universalEventWeight/100, 1)
	}
	return sc.rarity.weight()
}

func eligibleScenarios(c eventCase, keep func(sc listedScenario) bool) []listedScenario {
	var pool []listedScenario
	for _, sc := range eventCatalog {
		if !sc.stage && keep(sc) && sc.fits(c) {
			pool = append(pool, sc)
		}
	}
	return pool
}

func newPendingEvent(chat types.JID, acc core.Account, ids []types.JID, sc listedScenario, c eventCase) *pendingEvent {
	ev := &pendingEvent{
		key: eventKey(chat, acc), chat: chat, acc: acc, ids: ids, scenario: sc.id, spot: c.eventSpot,
		stake: c.job.shiftValue(), expires: c.now.Add(eventIdle), foreseen: c.job.oracle,
	}
	for _, i := range sc.open(c) {
		price := sc.choices[i].price.of(max(c.balance, 0))
		ev.options = append(ev.options, i)
		ev.prices = append(ev.prices, price)
		ev.labels = append(ev.labels, sc.choices[i].option(price)+sc.choices[i].reading(c.job))
	}
	return ev
}

func (b *bot) showEvent(ctx context.Context, ev *pendingEvent, sc listedScenario) error {
	text := ev.opening(sc)
	narrative, err := b.SendMentions(ctx, ev.chat, text, ev.mentioned(text))
	if err != nil {
		b.scenes.pending.CompareAndDelete(ev.key, ev)
		return fmt.Errorf("contar o evento: %w", err)
	}
	poll, err := b.Client.SendMessage(ctx, ev.chat, b.Client.BuildPollCreation(eventQuestion, ev.labels, 1))
	if err != nil {
		b.scenes.pending.CompareAndDelete(ev.key, ev)
		return fmt.Errorf("mandar a enquete do evento: %w", err)
	}
	b.track(ctx, ev, []types.MessageID{narrative, poll.ID}, time.Until(ev.expires))
	return nil
}

func (b *bot) track(ctx context.Context, ev *pendingEvent, messages []types.MessageID, wait time.Duration) {
	ev.mu.Lock()
	defer ev.mu.Unlock()
	if ev.closed {
		return
	}
	ev.messages = messages
	for _, id := range messages {
		b.scenes.messages.Store(id, ev)
	}
	b.KeepGame(ctx, eventKind, ev.key, ev.saved(), ev.expires)
	ev.timer = time.AfterFunc(wait, func() { b.expireEvent(ctx, ev) })
}

func (ev *pendingEvent) sent() []types.MessageID {
	ev.mu.Lock()
	defer ev.mu.Unlock()
	return ev.messages
}

func (ev *pendingEvent) opening(sc listedScenario) string {
	text := fmt.Sprintf("⚡ *Evento: %s*%s\n\n%s\n\n_Vota na enquete aqui embaixo em até %s (só vale o voto de @%s), ou responde essa mensagem com o número da opção_",
		sc.title, sc.rarity.badge(), ev.render(core.PickOne(sc.intros)), core.FormatWait(eventIdle), ev.acc.JID.User)
	if ev.foreseen {
		text += fmt.Sprintf("\n🐚 Os búzios de @%s já leram o destino: cada opção mostra a chance de dar bom e de dar ruim", ev.acc.JID.User)
	}
	return text
}

func (ev *pendingEvent) render(text string) string {
	target := "a vítima"
	if !ev.victim.JID.IsEmpty() {
		target = "@" + ev.victim.JID.User
	}
	return strings.NewReplacer("{@}", "@"+ev.acc.JID.User, "{alvo}", target).Replace(text)
}

func (ev *pendingEvent) mentioned(text string, extra ...types.JID) []types.JID {
	people := []types.JID{ev.acc.JID}
	for _, jid := range append([]types.JID{ev.victim.JID}, extra...) {
		if !jid.IsEmpty() && strings.Contains(text, "@"+jid.User) {
			people = append(people, jid)
		}
	}
	return people
}

func (ev *pendingEvent) close() bool {
	ev.mu.Lock()
	defer ev.mu.Unlock()
	if ev.closed {
		return false
	}
	ev.closed = true
	if ev.timer != nil {
		ev.timer.Stop()
	}
	return true
}

func (ev *pendingEvent) owns(info types.MessageInfo) bool {
	return slices.ContainsFunc(ev.ids, func(id types.JID) bool { return core.SentByUser(info, id.User) })
}

func (ev *pendingEvent) picked(selected [][]byte) int {
	return pickedLabel(ev.labels, selected)
}

func pickedLabel(labels []string, selected [][]byte) int {
	if len(selected) == 0 {
		return -1
	}
	return slices.IndexFunc(labels, func(label string) bool {
		hash := sha256.Sum256([]byte(label))
		return bytes.Equal(hash[:], selected[0])
	})
}

func (ev *pendingEvent) play(sc listedScenario, pick int, job Profession) eventPlay {
	if pick < 0 || pick >= len(ev.options) || ev.options[pick] >= len(sc.choices) {
		return eventPlay{ending: sc.timeout()}
	}
	ch := sc.choices[ev.options[pick]]
	if !strings.HasPrefix(ev.labels[pick], ch.label) {
		return eventPlay{ending: sc.timeout()}
	}
	return eventPlay{choice: &ch, price: ev.prices[pick], label: ch.label, ending: ch.roll(job)}
}

func (ev *pendingEvent) run(job Profession) eventRun {
	return eventRun{chat: ev.chat, acc: ev.acc, victim: ev.victim, job: job, place: ev.spot.place, stake: ev.stake, now: time.Now()}
}

func (ev *pendingEvent) saved() eventSave {
	return eventSave{
		Chat: ev.chat, Account: core.StoreAccount(ev.acc), IDs: ev.ids, Victim: core.StoreAccount(ev.victim),
		Scenario: ev.scenario, Place: ev.spot.place, Won: ev.spot.won, Target: ev.spot.victim, Stake: ev.stake,
		Options: ev.options, Prices: ev.prices, Labels: ev.labels, Messages: ev.messages, Expires: ev.expires,
	}
}

func (s eventSave) pending(key string) *pendingEvent {
	return &pendingEvent{
		key: key, chat: s.Chat, acc: s.Account.Account(), ids: s.IDs, victim: s.Victim.Account(), scenario: s.Scenario,
		spot: eventSpot{place: s.Place, won: s.Won, victim: s.Target}, stake: s.Stake, options: s.Options,
		prices: s.Prices, labels: s.Labels, messages: s.Messages, expires: s.Expires,
	}
}

func (b *bot) restoreEvent(ctx context.Context, saved core.SavedGame) error {
	var state eventSave
	if err := json.Unmarshal(saved.State, &state); err != nil {
		return fmt.Errorf("ler o evento: %w", err)
	}
	ev := state.pending(saved.Key)
	if _, taken := b.scenes.pending.LoadOrStore(ev.key, ev); taken {
		return nil
	}
	b.track(ctx, ev, ev.messages, max(time.Until(ev.expires), eventRestoreGrace))
	return nil
}

func (b *bot) answerEvent(ctx context.Context, evt *events.Message) error {
	if update := evt.Message.GetPollUpdateMessage(); update != nil {
		return b.voteEvent(ctx, evt, update.GetPollCreationMessageKey().GetID())
	}
	ev, ok := b.eventAt(evt.Message.GetExtendedTextMessage().GetContextInfo().GetStanzaID())
	if !ok || !ev.owns(evt.Info) {
		return nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(core.MessageText(evt.Message)))
	if err != nil || n < 1 || n > len(ev.labels) {
		return nil
	}
	return b.settleEvent(ctx, ev, n-1)
}

func (b *bot) voteEvent(ctx context.Context, evt *events.Message, poll types.MessageID) error {
	ev, ok := b.eventAt(poll)
	if !ok || !ev.owns(evt.Info) {
		return nil
	}
	vote, err := b.Client.DecryptPollVote(ctx, evt)
	if err != nil {
		return fmt.Errorf("ler voto do evento: %w", err)
	}
	pick := ev.picked(vote.GetSelectedOptions())
	if pick < 0 {
		return nil
	}
	return b.settleEvent(ctx, ev, pick)
}

func (b *bot) eventAt(id types.MessageID) (*pendingEvent, bool) {
	if id == "" {
		return nil, false
	}
	value, ok := b.scenes.messages.Load(id)
	if !ok {
		return nil, false
	}
	return value.(*pendingEvent), true
}

func (b *bot) expireEvent(ctx context.Context, ev *pendingEvent) {
	if ctx.Err() != nil {
		return
	}
	if err := b.settleEvent(ctx, ev, -1); err != nil {
		b.Client.Log.Errorf("encerrar o evento de %s em %s: %v", ev.acc.JID, ev.chat, err)
	}
}

func (b *bot) settleEvent(ctx context.Context, ev *pendingEvent, pick int) error {
	if !ev.close() {
		return nil
	}
	b.releaseEvent(ctx, ev)
	sc, ok := scenarioByID(ev.scenario)
	if !ok {
		return nil
	}
	job, err := b.ProfessionOf(ctx, ev.chat, ev.acc)
	if err != nil {
		return err
	}
	play := ev.play(sc, pick, job)
	res, err := b.Wallets.playEvent(ctx, ev.run(job), play)
	if err != nil {
		return fmt.Errorf("resolver o evento %s: %w", sc.id, err)
	}
	text := ev.outcome(sc, play, res)
	if _, err := b.SendMentions(ctx, ev.chat, text, ev.mentioned(text, res.freed...)); err != nil {
		return err
	}
	if play.ending.next == "" || !res.played() {
		return nil
	}
	return b.followEvent(ctx, ev, play.ending.next)
}

func (b *bot) releaseEvent(ctx context.Context, ev *pendingEvent) {
	b.scenes.pending.CompareAndDelete(ev.key, ev)
	for _, id := range ev.sent() {
		b.scenes.messages.CompareAndDelete(id, ev)
	}
	b.DropGame(ctx, eventKind, ev.key)
}

func (b *bot) followEvent(ctx context.Context, ev *pendingEvent, id string) error {
	sc, ok := scenarioByID(id)
	if !ok {
		return nil
	}
	on, err := b.ChatModeOn(ctx, ev.chat, core.ModeEconomy)
	if err != nil || !on {
		return err
	}
	c, _, err := b.eventCaseOf(ctx, ev.chat, ev.acc, ev.spot)
	if err != nil {
		return err
	}
	if len(sc.open(c)) < 2 {
		return nil
	}
	next := newPendingEvent(ev.chat, ev.acc, ev.ids, sc, c)
	next.victim = ev.victim
	if _, taken := b.scenes.pending.LoadOrStore(next.key, next); taken {
		return nil
	}
	return b.showEvent(ctx, next, sc)
}

func (b *bot) haltEvents(ctx context.Context, chat types.JID) {
	b.scenes.pending.Range(func(_, value any) bool {
		ev := value.(*pendingEvent)
		if ev.chat == chat && ev.close() {
			b.releaseEvent(ctx, ev)
		}
		return true
	})
}

func (ev *pendingEvent) outcome(sc listedScenario, play eventPlay, res eventResult) string {
	head := fmt.Sprintf("⌛ *%s*: @%s não escolheu a tempo", sc.title, ev.acc.JID.User)
	if play.choice != nil {
		head = fmt.Sprintf("🎲 *%s*: @%s escolheu %s", sc.title, ev.acc.JID.User, play.label)
	}
	if note := res.void(ev.acc.JID); note != "" {
		return head + "\n\n" + note
	}
	lines := []string{head}
	if len(play.ending.texts) > 0 {
		lines = append(lines, "", ev.render(core.PickOne(play.ending.texts)))
	}
	if len(res.lines) > 0 {
		lines = append(lines, "", strings.Join(res.lines, "\n"))
	}
	return strings.Join(lines, "\n")
}
