package economy

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	liveKind       = "aovivo"
	liveChatKind   = "aovivo-grupo"
	liveRest       = 90 * time.Minute
	liveDue        = 5 * time.Hour
	livePeak       = 0.12
	liveFirstHour  = 9
	liveLastHour   = 23
	livePlayerSpan = 24 * time.Hour
	liveCrowd      = 3
	liveMemory     = 8
	liveSaveEvery  = 10 * time.Minute
	liveGrace      = 30 * time.Second
	liveJobLimit   = 4
)

type liveFormat int

const (
	grabFormat liveFormat = iota
	bossFormat
	voteFormat
)

type liveRules struct {
	weight int
	crowd  int
	window time.Duration
	setup  func(ev *liveEvent, crowd int)
	show   func(b *bot, ctx context.Context, ev *liveEvent, players []types.JID) error
	finish func(b *bot, ctx context.Context, ev *liveEvent) error
}

func (f liveFormat) rules() liveRules {
	return [...]liveRules{
		grabFormat: {weight: 40, crowd: 1, window: grabWindow, setup: setupGrab, show: (*bot).showGrab, finish: (*bot).missGrab},
		bossFormat: {weight: 30, crowd: liveCrowd, window: bossWindow, setup: setupBoss, show: (*bot).showBoss, finish: (*bot).escapeBoss},
		voteFormat: {weight: 30, crowd: liveCrowd, window: voteWindow, setup: setupVote, show: (*bot).showVote, finish: (*bot).tallyVote},
	}[f]
}

var liveFormats = []liveFormat{grabFormat, bossFormat, voteFormat}

type liveEntry struct {
	format liveFormat
	id     string
	rarity eventRarity
}

var liveCatalog = listLive()

func listLive() []liveEntry {
	var entries []liveEntry
	for _, sc := range grabScenarios {
		entries = append(entries, liveEntry{format: grabFormat, id: sc.id, rarity: sc.rarity})
	}
	for _, sc := range bossScenarios {
		entries = append(entries, liveEntry{format: bossFormat, id: sc.id, rarity: sc.rarity})
	}
	for _, sc := range voteScenarios {
		entries = append(entries, liveEntry{format: voteFormat, id: sc.id, rarity: sc.rarity})
	}
	return entries
}

func liveEntryOf(format liveFormat, id string) (liveEntry, bool) {
	i := slices.IndexFunc(liveCatalog, func(e liveEntry) bool { return e.format == format && e.id == id })
	if i < 0 {
		return liveEntry{}, false
	}
	return liveCatalog[i], true
}

func pickLive(crowd int, recent []string) (liveEntry, bool) {
	formats := slices.DeleteFunc(slices.Clone(liveFormats), func(f liveFormat) bool { return crowd < f.rules().crowd })
	if len(formats) == 0 {
		return liveEntry{}, false
	}
	format := formats[weightedIndex(len(formats), func(i int) int { return formats[i].rules().weight })]
	pool := liveScenariosOf(format, func(e liveEntry) bool { return !slices.Contains(recent, e.id) })
	if len(pool) == 0 {
		pool = liveScenariosOf(format, func(liveEntry) bool { return true })
	}
	if len(pool) == 0 {
		return liveEntry{}, false
	}
	return pool[weightedIndex(len(pool), func(i int) int { return pool[i].rarity.weight() })], true
}

func liveScenariosOf(format liveFormat, keep func(e liveEntry) bool) []liveEntry {
	var pool []liveEntry
	for _, e := range liveCatalog {
		if e.format == format && keep(e) {
			pool = append(pool, e)
		}
	}
	return pool
}

func liveOdds(idle time.Duration) float64 {
	if idle >= liveDue {
		return 1
	}
	if idle <= liveRest {
		return 0
	}
	return livePeak * float64(idle-liveRest) / float64(liveDue-liveRest)
}

func liveHour(now time.Time) bool {
	hour := now.In(core.BrazilTime).Hour()
	return hour >= liveFirstHour && hour < liveLastHour
}

type liveBoard struct {
	events sync.Map
	polls  sync.Map
	mu     sync.Mutex
	chats  map[types.JID]*liveChat
}

type liveChat struct {
	last    time.Time
	players map[string]livePlayer
	recent  []string
	saved   time.Time
}

type livePlayer struct {
	JID  types.JID `json:"jid"`
	Seen time.Time `json:"seen"`
}

type liveChatSave struct {
	Last    time.Time             `json:"last"`
	Players map[string]livePlayer `json:"players"`
	Recent  []string              `json:"recent"`
}

type liveCue struct {
	due     bool
	players []types.JID
	recent  []string
	save    *liveChatSave
}

func (l *liveBoard) chat(chat types.JID, now time.Time) *liveChat {
	if l.chats == nil {
		l.chats = map[types.JID]*liveChat{}
	}
	c, ok := l.chats[chat]
	if !ok {
		c = &liveChat{last: now.Add(-liveRest), players: map[string]livePlayer{}}
		l.chats[chat] = c
	}
	return c
}

func (l *liveBoard) cue(chat types.JID, acc core.Account, now time.Time) liveCue {
	l.mu.Lock()
	defer l.mu.Unlock()
	c := l.chat(chat, now)
	_, known := c.players[acc.Key]
	c.players[acc.Key] = livePlayer{JID: acc.JID, Seen: now}
	for key, p := range c.players {
		if now.Sub(p.Seen) > livePlayerSpan {
			delete(c.players, key)
		}
	}
	cue := liveCue{due: rand.Float64() < liveOdds(now.Sub(c.last)), players: c.roster(), recent: slices.Clone(c.recent)}
	if known && now.Sub(c.saved) < liveSaveEvery {
		return cue
	}
	c.saved = now
	state := c.state()
	cue.save = &state
	return cue
}

func (l *liveBoard) opened(chat types.JID, id string, now time.Time) liveChatSave {
	l.mu.Lock()
	defer l.mu.Unlock()
	c := l.chat(chat, now)
	c.last, c.saved = now, now
	recent := append(slices.Clone(c.recent), id)
	c.recent = recent[max(len(recent)-liveMemory, 0):]
	return c.state()
}

func (l *liveBoard) restore(chat types.JID, s liveChatSave) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.chats == nil {
		l.chats = map[types.JID]*liveChat{}
	}
	players := s.Players
	if players == nil {
		players = map[string]livePlayer{}
	}
	l.chats[chat] = &liveChat{last: s.Last, players: players, recent: s.Recent, saved: time.Now()}
}

func (c *liveChat) roster() []types.JID {
	keys := make([]string, 0, len(c.players))
	for key := range c.players {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(a, b string) int { return c.players[b].Seen.Compare(c.players[a].Seen) })
	roster := make([]types.JID, len(keys))
	for i, key := range keys {
		roster[i] = c.players[key].JID
	}
	return roster
}

func (c *liveChat) state() liveChatSave {
	players := make(map[string]livePlayer, len(c.players))
	for key, p := range c.players {
		players[key] = p
	}
	return liveChatSave{Last: c.last, Players: players, Recent: slices.Clone(c.recent)}
}

type liveEvent struct {
	mu       sync.Mutex
	chat     types.JID
	format   liveFormat
	scenario string
	expires  time.Time
	clue     string
	answer   string
	life     int64
	health   int64
	hitters  map[string]*liveHitter
	labels   []string
	poll     types.MessageID
	ballots  map[string]liveBallot
	keep     func()
	timer    *time.Timer
	closed   bool
}

type liveHitter struct {
	acc    core.Account
	damage int64
	hits   int
	last   time.Time
}

type liveBallot struct {
	Voter  types.JID `json:"voter"`
	Choice int       `json:"choice"`
}

type liveSave struct {
	Chat     types.JID       `json:"chat"`
	Format   liveFormat      `json:"format"`
	Scenario string          `json:"scenario"`
	Expires  time.Time       `json:"expires"`
	Clue     string          `json:"clue,omitempty"`
	Answer   string          `json:"answer,omitempty"`
	Life     int64           `json:"life,omitempty"`
	Health   int64           `json:"health,omitempty"`
	Hitters  []hitterSave    `json:"hitters,omitempty"`
	Labels   []string        `json:"labels,omitempty"`
	Poll     types.MessageID `json:"poll,omitempty"`
	Ballots  []liveBallot    `json:"ballots,omitempty"`
}

type hitterSave struct {
	Account core.StoredAccount `json:"account"`
	Damage  int64              `json:"damage"`
	Hits    int                `json:"hits"`
	Last    time.Time          `json:"last"`
}

func newLiveEvent(chat types.JID, entry liveEntry, crowd int, now time.Time) *liveEvent {
	rules := entry.format.rules()
	ev := &liveEvent{
		chat: chat, format: entry.format, scenario: entry.id, expires: now.Add(rules.window),
		hitters: map[string]*liveHitter{}, ballots: map[string]liveBallot{},
	}
	rules.setup(ev, crowd)
	return ev
}

func (b *bot) stirLive(ctx context.Context, evt *events.Message, cmd core.Command) {
	chat := evt.Info.Chat.ToNonAD()
	if cmd.Category.Mode != core.ModeEconomy || chat.Server != types.GroupServer {
		return
	}
	now := time.Now()
	cue := b.live.cue(chat, b.Account(ctx, evt.Info.Sender), now)
	if cue.save != nil {
		b.keepLiveChat(ctx, chat, *cue.save)
	}
	if !cue.due || !liveHour(now) {
		return
	}
	if _, busy := b.live.events.Load(chat); busy {
		return
	}
	on, err := b.ChatModeOn(ctx, chat, core.ModeLive)
	if err != nil {
		b.Client.Log.Errorf("ver o modo aovivo em %s: %v", chat, err)
		return
	}
	if !on {
		return
	}
	if err := b.openLive(ctx, chat, cue.players, cue.recent, now); err != nil {
		b.Client.Log.Errorf("abrir evento ao vivo em %s: %v", chat, err)
	}
}

func (b *bot) openLive(ctx context.Context, chat types.JID, players []types.JID, recent []string, now time.Time) error {
	entry, ok := pickLive(len(players), recent)
	if !ok {
		return nil
	}
	ev := newLiveEvent(chat, entry, len(players), now)
	if _, taken := b.live.events.LoadOrStore(chat, ev); taken {
		return nil
	}
	b.keepLiveChat(ctx, chat, b.live.opened(chat, entry.id, now))
	if err := entry.format.rules().show(b, ctx, ev, players); err != nil {
		b.live.events.CompareAndDelete(chat, ev)
		return fmt.Errorf("mostrar o evento ao vivo %s: %w", entry.id, err)
	}
	b.trackLive(ctx, ev, time.Until(ev.expires))
	return nil
}

func (b *bot) keepLiveChat(ctx context.Context, chat types.JID, state liveChatSave) {
	b.KeepGame(ctx, liveChatKind, chat.String(), state, time.Now().Add(2*livePlayerSpan))
}

func (b *bot) trackLive(ctx context.Context, ev *liveEvent, wait time.Duration) {
	poll, ok := ev.arm(func() { b.keepLive(ctx, ev) }, func() { b.expireLive(ctx, ev) }, wait)
	if !ok {
		return
	}
	if poll != "" {
		b.live.polls.Store(poll, ev)
	}
	b.keepLive(ctx, ev)
}

func (ev *liveEvent) arm(keep, expire func(), wait time.Duration) (types.MessageID, bool) {
	ev.mu.Lock()
	defer ev.mu.Unlock()
	if ev.closed {
		return "", false
	}
	ev.keep = keep
	ev.timer = time.AfterFunc(wait, expire)
	return ev.poll, true
}

func (ev *liveEvent) pollID() types.MessageID {
	ev.mu.Lock()
	defer ev.mu.Unlock()
	return ev.poll
}

func (b *bot) keepLive(ctx context.Context, ev *liveEvent) {
	ev.mu.Lock()
	defer ev.mu.Unlock()
	if ev.closed {
		return
	}
	b.KeepGame(ctx, liveKind, ev.chat.String(), ev.snapshot(), ev.expires)
}

func (ev *liveEvent) snapshot() liveSave {
	s := liveSave{
		Chat: ev.chat, Format: ev.format, Scenario: ev.scenario, Expires: ev.expires, Clue: ev.clue, Answer: ev.answer,
		Life: ev.life, Health: ev.health, Labels: ev.labels, Poll: ev.poll,
	}
	for _, h := range ev.hitters {
		s.Hitters = append(s.Hitters, hitterSave{Account: core.StoreAccount(h.acc), Damage: h.damage, Hits: h.hits, Last: h.last})
	}
	for _, v := range ev.ballots {
		s.Ballots = append(s.Ballots, v)
	}
	return s
}

func (s liveSave) event() *liveEvent {
	ev := &liveEvent{
		chat: s.Chat, format: s.Format, scenario: s.Scenario, expires: s.Expires, clue: s.Clue, answer: s.Answer,
		life: s.Life, health: s.Health, labels: s.Labels, poll: s.Poll,
		hitters: map[string]*liveHitter{}, ballots: map[string]liveBallot{},
	}
	for _, h := range s.Hitters {
		acc := h.Account.Account()
		ev.hitters[acc.Key] = &liveHitter{acc: acc, damage: h.Damage, hits: h.Hits, last: h.Last}
	}
	for _, v := range s.Ballots {
		ev.ballots[v.Voter.User] = v
	}
	return ev
}

func (b *bot) restoreLive(ctx context.Context, saved core.SavedGame) error {
	var state liveSave
	if err := json.Unmarshal(saved.State, &state); err != nil {
		return fmt.Errorf("ler o evento ao vivo: %w", err)
	}
	if _, ok := liveEntryOf(state.Format, state.Scenario); !ok {
		return fmt.Errorf("o evento ao vivo %s não existe mais", state.Scenario)
	}
	ev := state.event()
	if _, taken := b.live.events.LoadOrStore(ev.chat, ev); taken {
		return nil
	}
	b.trackLive(ctx, ev, max(time.Until(ev.expires), liveGrace))
	return nil
}

func (b *bot) restoreLiveChat(_ context.Context, saved core.SavedGame) error {
	chat, err := types.ParseJID(saved.Key)
	if err != nil {
		return fmt.Errorf("ler o grupo do evento ao vivo: %w", err)
	}
	var state liveChatSave
	if err := json.Unmarshal(saved.State, &state); err != nil {
		return fmt.Errorf("ler os jogadores do evento ao vivo: %w", err)
	}
	b.live.restore(chat, state)
	return nil
}

func (ev *liveEvent) close() bool {
	ev.mu.Lock()
	defer ev.mu.Unlock()
	return ev.shut()
}

func (ev *liveEvent) shut() bool {
	if ev.closed {
		return false
	}
	ev.closed = true
	if ev.timer != nil {
		ev.timer.Stop()
	}
	return true
}

func (b *bot) releaseLive(ctx context.Context, ev *liveEvent) {
	b.live.events.CompareAndDelete(ev.chat, ev)
	if poll := ev.pollID(); poll != "" {
		b.live.polls.CompareAndDelete(poll, ev)
	}
	b.DropGame(ctx, liveKind, ev.chat.String())
}

func (b *bot) expireLive(ctx context.Context, ev *liveEvent) {
	if ctx.Err() != nil || !ev.close() {
		return
	}
	b.releaseLive(ctx, ev)
	if err := ev.format.rules().finish(b, ctx, ev); err != nil {
		b.Client.Log.Errorf("encerrar o evento ao vivo %s em %s: %v", ev.scenario, ev.chat, err)
	}
}

func (b *bot) haltLive(ctx context.Context, chat types.JID) {
	value, ok := b.live.events.Load(chat.ToNonAD())
	if !ok {
		return
	}
	ev := value.(*liveEvent)
	if ev.close() {
		b.releaseLive(ctx, ev)
	}
}

func (b *bot) liveIn(chat types.JID, format liveFormat) (*liveEvent, bool) {
	value, ok := b.live.events.Load(chat)
	if !ok {
		return nil, false
	}
	ev := value.(*liveEvent)
	return ev, ev.format == format
}

type liveTurn struct {
	acc     core.Account
	weight  int64
	effects []effect
}

func (b *bot) playLive(ctx context.Context, chat types.JID, turns []liveTurn) ([]eventResult, error) {
	jobs := make([]Profession, len(turns))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(liveJobLimit)
	for i, t := range turns {
		g.Go(func() (err error) { jobs[i], err = b.ProfessionOf(gctx, chat, t.acc); return err })
	}
	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("ler as profissões do evento ao vivo: %w", err)
	}
	results := make([]eventResult, len(turns))
	for i, t := range turns {
		run := eventRun{chat: chat, acc: t.acc, job: jobs[i], stake: jobs[i].shiftValue() * t.weight / 100, now: time.Now()}
		res, err := b.Wallets.playEvent(ctx, run, eventPlay{ending: ending{effects: t.effects}})
		if err != nil {
			return nil, err
		}
		results[i] = res
	}
	return results, nil
}

func liveLine(head string, res eventResult) string {
	if res.gone {
		return head + ": ⚰️ tava morto e ficou de fora"
	}
	if len(res.lines) == 0 {
		return head + ": saiu no zero a zero"
	}
	return head + "\n" + strings.Join(res.lines, "\n")
}

func liveFreed(results []eventResult) []types.JID {
	var freed []types.JID
	for _, res := range results {
		freed = append(freed, res.freed...)
	}
	return freed
}

func liveHeader(emoji, title string, rarity eventRarity) string {
	return fmt.Sprintf("%s *Ao vivo: %s*%s", emoji, title, rarity.badge())
}

func liveCall(players []types.JID) string {
	if len(players) == 0 {
		return ""
	}
	return "\n\n📣 Chamando quem jogou hoje: " + core.JoinMentions(players)
}
