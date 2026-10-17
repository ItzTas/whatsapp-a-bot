package city

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
)

const (
	cityKind       = "cidadedorme"
	cityMinPlayers = 4
	cityMaxPlayers = 16
	cityTimeout    = 2 * time.Hour
	cityIdleLimit  = 2
	cityNobody     = -1
	cityStranger   = -2
	cityPollLimit  = 12
)

type cityPhase int

const (
	cityLobby cityPhase = iota
	cityNight
	cityDay
	cityOver
)

var cityDurations = map[cityPhase]time.Duration{
	cityLobby: 2 * time.Minute,
	cityNight: 2 * time.Minute,
	cityDay:   3 * time.Minute,
}

type cityRole int

const (
	cityCitizen cityRole = iota
	cityAssassin
	cityDoctor
	cityDetective
)

type cityRoleInfo struct {
	name    string
	emoji   string
	plural  string
	command string
	verb    string
	brief   string
}

var cityRoles = map[cityRole]cityRoleInfo{
	cityCitizen: {
		name:   "cidadão",
		emoji:  "🧑‍🌾",
		plural: "cidadãos",
		brief:  "🧑‍🌾 *Você é cidadão*%s.\n\nVocê não tem poder de noite, então dorme tranquilo 😴 De dia, presta atenção na conversa e ajuda a cidade a votar no assassino",
	},
	cityAssassin: {
		name:    "assassino",
		emoji:   "🔪",
		plural:  "assassinos",
		command: "matar",
		verb:    "matar",
		brief:   "🔪 *Você é assassino*%s!\n\nToda noite eu te mando aqui uma enquete com quem dá pra matar, e você toca no nome (ou responde !matar <número>). De dia, finge que é inocente e tenta jogar a culpa em outra pessoa 😈",
	},
	cityDoctor: {
		name:    "médico",
		emoji:   "💉",
		plural:  "médicos",
		command: "salvar",
		verb:    "salvar",
		brief:   "💉 *Você é o médico*%s!\n\nToda noite eu te mando aqui uma enquete pra escolher alguém pra salvar (ou você responde !salvar <número>). Se o assassino for nessa pessoa, ela não morre. Pode ser você mesmo, só não pode ser a mesma pessoa duas noites seguidas",
	},
	cityDetective: {
		name:    "detetive",
		emoji:   "🔎",
		plural:  "detetives",
		command: "investigar",
		verb:    "investigar",
		brief:   "🔎 *Você é o detetive*%s!\n\nToda noite eu te mando aqui uma enquete pra escolher alguém pra investigar (ou você responde !investigar <número>), e de manhã eu te conto se a pessoa é assassina. Usa isso pra guiar a cidade sem se entregar, porque o assassino vai querer te pegar",
	},
}

type cityPlayer struct {
	jid  types.JID
	ids  []types.JID
	role cityRole
	dead bool
}

type cityGame struct {
	mu        sync.Mutex
	chat      types.JID
	host      cityPlayer
	phase     cityPhase
	players   []cityPlayer
	round     int
	deadline  time.Time
	hurry     chan struct{}
	opened    time.Time
	idle      int
	quiet     bool
	kills     map[int]int
	saved     int
	lastSaved int
	probe     int
	votes     map[int]int
	dropped   bool
	polls     map[int][]cityPoll
	prompts   map[int]*core.PrivatePrompt
}

type cityPollDraft struct {
	question string
	options  []string
}

type cityPoll struct {
	ID      types.MessageID `json:"id"`
	Options []string        `json:"options"`
}

type cityAction func(g *cityGame, info types.MessageInfo) (string, []types.JID)

type cityPick struct {
	reply    string
	mentions []types.JID
	allies   []cityPlayer
	tip      string
	taken    bool
}

type cityDawn struct {
	victim int
	sleuth int
	probe  int
}

func newCityGame(chat types.JID, host cityPlayer) *cityGame {
	game := &cityGame{chat: chat, host: host, players: []cityPlayer{host}, opened: time.Now(), saved: cityNobody, lastSaved: cityNobody, probe: cityNobody}
	game.enter(cityLobby)
	return game
}

func (g *cityGame) enter(phase cityPhase) {
	g.phase = phase
	g.deadline = time.Now().Add(cityDurations[phase])
	g.hurry = make(chan struct{})
}

func (g *cityGame) wake() {
	select {
	case <-g.hurry:
	default:
		close(g.hurry)
	}
}

func (g *cityGame) hurried() bool {
	select {
	case <-g.hurry:
		return true
	default:
		return false
	}
}

func (g *cityGame) halt() {
	g.phase = cityOver
	g.wake()
}

func (g *cityGame) wait(ctx context.Context) bool {
	g.mu.Lock()
	hurry, deadline := g.hurry, g.deadline
	g.mu.Unlock()
	select {
	case <-ctx.Done():
		return false
	case <-hurry:
	case <-time.After(time.Until(deadline)):
	}
	return !g.over()
}

func (g *cityGame) currentPhase() cityPhase {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.phase
}

func (g *cityGame) over() bool {
	return g.currentPhase() == cityOver
}

func (g *cityGame) timedOut() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return time.Since(g.opened) > cityTimeout
}

func (g *cityGame) seatOf(info types.MessageInfo) int {
	return slices.IndexFunc(g.players, func(p cityPlayer) bool {
		return slices.ContainsFunc(p.ids, func(id types.JID) bool { return core.SentBy(info, id) })
	})
}

func (g *cityGame) seatOfUser(user string) int {
	return slices.IndexFunc(g.players, func(p cityPlayer) bool {
		return slices.ContainsFunc(p.ids, func(id types.JID) bool { return id.User == user })
	})
}

func (g *cityGame) join(info types.MessageInfo, player cityPlayer) (string, []types.JID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase != cityLobby {
		return g.overview()
	}
	if g.seatOf(info) >= 0 {
		return g.board(fmt.Sprintf("🙋 Você já tá na sala, @%s", player.jid.User))
	}
	if len(g.players) >= cityMaxPlayers {
		return fmt.Sprintf("🚫 A sala já tá cheia (%d jogadores), espera a próxima partida", cityMaxPlayers), nil
	}
	g.players = append(g.players, player)
	header := fmt.Sprintf("✅ @%s entrou na cidade!", player.jid.User)
	if len(g.players) < cityMaxPlayers {
		return g.board(header)
	}
	g.wake()
	return header + "\n\n🚀 A sala lotou, começando a partida!", []types.JID{player.jid}
}

func (g *cityGame) start(info types.MessageInfo) (string, []types.JID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase != cityLobby {
		return g.overview()
	}
	if g.seatOf(info) < 0 {
		return "Só quem tá na sala pode fazer isso 🙄", nil
	}
	if len(g.players) < cityMinPlayers {
		return g.board(fmt.Sprintf("⏳ Ainda não dá pra começar: precisa de pelo menos %d jogadores", cityMinPlayers))
	}
	g.wake()
	return "", nil
}

func (g *cityGame) leave(info types.MessageInfo) (string, []types.JID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase != cityLobby {
		return "🌃 A partida já começou, agora vai até o fim", nil
	}
	seat := g.seatOf(info)
	if seat < 0 {
		return "Você nem tá na sala 🤨", nil
	}
	gone := g.players[seat].jid
	g.players = slices.Delete(g.players, seat, seat+1)
	header := fmt.Sprintf("🚪 @%s saiu da sala", gone.User)
	if len(g.players) > 0 {
		text, mentions := g.board(header)
		return text, append(mentions, gone)
	}
	g.halt()
	return header + " e ela ficou vazia, então a partida foi cancelada", []types.JID{gone}
}

func (g *cityGame) cancel(info types.MessageInfo) (string, []types.JID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	host := g.host.jid
	if !slices.ContainsFunc(g.host.ids, func(id types.JID) bool { return core.SentBy(info, id) }) {
		return fmt.Sprintf("Só quem abriu a sala (@%s) pode cancelar a partida 🙄", host.User), []types.JID{host}
	}
	header := fmt.Sprintf("❌ @%s cancelou a partida", host.User)
	lobby := g.phase == cityLobby
	g.halt()
	if lobby {
		return header, []types.JID{host}
	}
	return header + "\n\n" + g.reveal(), append(g.jids(), host)
}

func (g *cityGame) deal() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.players) < cityMinPlayers {
		g.phase = cityOver
		return false
	}
	roles := cityCast(len(g.players))
	rand.Shuffle(len(roles), func(i, j int) { roles[i], roles[j] = roles[j], roles[i] })
	for seat := range g.players {
		g.players[seat].role = roles[seat]
	}
	return true
}

func cityCast(players int) []cityRole {
	assassins := 1
	for _, threshold := range []int{7, 11} {
		if players >= threshold {
			assassins++
		}
	}
	roles := append(slices.Repeat([]cityRole{cityAssassin}, assassins), cityDoctor, cityDetective)
	return append(roles, slices.Repeat([]cityRole{cityCitizen}, players-len(roles))...)
}

func (g *cityGame) cast() string {
	counts := map[cityRole]int{}
	for _, p := range g.players {
		counts[p.role]++
	}
	var parts []string
	for _, role := range []cityRole{cityAssassin, cityDoctor, cityDetective, cityCitizen} {
		info := cityRoles[role]
		if counts[role] == 1 {
			parts = append(parts, fmt.Sprintf("%s 1 %s", info.emoji, info.name))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %d %s", info.emoji, counts[role], info.plural))
	}
	return strings.Join(parts, " · ")
}

func (g *cityGame) briefing(seat int, label string) (string, []types.JID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	p := g.players[seat]
	text := fmt.Sprintf(cityRoles[p.role].brief, label)
	if p.role != cityAssassin {
		return text, nil
	}
	allies := g.alliesOf(seat)
	if len(allies) == 0 {
		return text + "\n\n🐺 Você é o único assassino da cidade", nil
	}
	jids := cityJIDs(allies)
	return fmt.Sprintf("%s\n\n🤝 Seu time: %s. Quando vocês escolhem pessoas diferentes, vale a mais escolhida (no empate, a sorte decide)", text, core.JoinMentions(jids)), jids
}

func (g *cityGame) alliesOf(seat int) []cityPlayer {
	var allies []cityPlayer
	for other, p := range g.players {
		if other != seat && !p.dead && p.role == cityAssassin {
			allies = append(allies, p)
		}
	}
	return allies
}

func (g *cityGame) nightfall(header string) (string, []types.JID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.round++
	g.kills = map[int]int{}
	g.saved, g.probe = cityNobody, cityNobody
	g.polls = map[int][]cityPoll{}
	g.enter(cityNight)
	return g.board(fmt.Sprintf("%s\n\n🌙 *Noite %d: a cidade dorme...*", header, g.round))
}

func (g *cityGame) nightSeats() []int {
	g.mu.Lock()
	defer g.mu.Unlock()
	var seats []int
	for seat, p := range g.players {
		if !p.dead && cityRoles[p.role].command != "" {
			seats = append(seats, seat)
		}
	}
	return seats
}

func (g *cityGame) jidOf(seat int) types.JID {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.players[seat].jid
}

func (g *cityGame) inboxOf(seat int) []types.JID {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.players[seat].ids
}

func (g *cityGame) commandOf(seat int) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return cityRoles[g.players[seat].role].command
}

func (g *cityGame) menu(seat int, label string) (string, []types.JID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	info := cityRoles[g.players[seat].role]
	lines := []string{fmt.Sprintf("🌙 *Noite %d*%s: quem você quer %s?", g.round, label, info.verb), ""}
	var mentions []types.JID
	for _, target := range g.targets(seat) {
		p := g.players[target]
		lines = append(lines, fmt.Sprintf("%d. @%s", target+1, p.jid.User))
		mentions = append(mentions, p.jid)
	}
	lines = append(lines, "", g.menuHint(seat))
	return strings.Join(lines, "\n"), mentions
}

func (g *cityGame) targets(seat int) []int {
	var targets []int
	for target := range g.players {
		if g.refusal(seat, target) == "" {
			targets = append(targets, target)
		}
	}
	return targets
}

func (g *cityGame) split(seat int) bool {
	return len(g.targets(seat)) > cityPollLimit
}

func (g *cityGame) changeRule(seat int) string {
	if g.split(seat) {
		return "A lista é grande e veio dividida, então vale a primeira escolha, sem troca 🔒"
	}
	return "Dá pra trocar até a noite acabar"
}

func (g *cityGame) ballots(seat int, label string, names []string) []cityPollDraft {
	g.mu.Lock()
	defer g.mu.Unlock()
	info := cityRoles[g.players[seat].role]
	targets := g.targets(seat)
	options := make([]string, len(targets))
	for i, target := range targets {
		options[i] = fmt.Sprintf("%d. %s", target+1, names[target])
	}
	rule := fmt.Sprintf("Toca num nome (ou manda !%s <número>). %s. Você tem %s ⏳", info.command, g.changeRule(seat), core.Countdown(g.deadline))
	if !g.split(seat) {
		return []cityPollDraft{{question: fmt.Sprintf("🌙 Noite %d%s: quem você quer %s?\n\n%s", g.round, label, info.verb, rule), options: options}}
	}
	parts := (len(options) + cityPollLimit - 1) / cityPollLimit
	size := (len(options) + parts - 1) / parts
	var drafts []cityPollDraft
	for chunk := range slices.Chunk(options, size) {
		question := fmt.Sprintf("🌙 Noite %d%s (lista %d de %d): quem você quer %s?\n\n%s", g.round, label, len(drafts)+1, parts, info.verb, rule)
		drafts = append(drafts, cityPollDraft{question: question, options: chunk})
	}
	return drafts
}

func (g *cityGame) keepPoll(seat int, poll cityPoll) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.polls[seat] = append(g.polls[seat], poll)
}

func (g *cityGame) pollsOf(seat int) []cityPoll {
	g.mu.Lock()
	defer g.mu.Unlock()
	return slices.Clone(g.polls[seat])
}

func (g *cityGame) roster() []types.JID {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.jids()
}

func (g *cityGame) menuHint(seat int) string {
	command := cityRoles[g.players[seat].role].command
	return fmt.Sprintf("_Responde aqui com !%s <número> (ou só o número). %s. Você tem %s_ ⏳", command, g.changeRule(seat), core.Countdown(g.deadline))
}

func (g *cityGame) hint(seat int) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.menuHint(seat)
}

func (g *cityGame) refusal(seat, target int) string {
	if target < 0 || target >= len(g.players) || g.players[target].dead {
		return "Esse número não tá na lista 🤔"
	}
	rules := map[cityRole]func() string{
		cityAssassin: func() string {
			if target == seat {
				return "Se matar? Melhor não 😅 Escolhe outra pessoa"
			}
			if g.players[target].role == cityAssassin {
				return "Esse é do seu time 🤝, escolhe outra pessoa"
			}
			return ""
		},
		cityDoctor: func() string {
			if target == g.lastSaved {
				return "Você salvou essa pessoa na noite passada, hoje escolhe outra 💉"
			}
			return ""
		},
		cityDetective: func() string {
			if target == seat {
				return "Investigar você mesmo? Você já sabe que é inocente 😅"
			}
			return ""
		},
	}
	return rules[g.players[seat].role]()
}

func (g *cityGame) choose(seat, target int) cityPick {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase != cityNight {
		return cityPick{reply: "⏰ A noite já acabou"}
	}
	if g.players[seat].dead {
		return cityPick{reply: "☠️ Você morreu, agora só assiste"}
	}
	if g.split(seat) && g.acted(seat) {
		return cityPick{reply: "🔒 Você já escolheu nessa noite. Como a lista é grande e veio dividida, vale a primeira escolha"}
	}
	if refusal := g.refusal(seat, target); refusal != "" {
		return cityPick{reply: refusal + "\n\n" + g.menuHint(seat)}
	}
	records := map[cityRole]func(){
		cityAssassin:  func() { g.kills[seat] = target },
		cityDoctor:    func() { g.saved = target },
		cityDetective: func() { g.probe = target },
	}
	role := g.players[seat].role
	records[role]()
	chosen := g.players[target].jid
	pick := cityPick{
		reply:    fmt.Sprintf("✅ Anotado: você vai %s @%s. %s", cityRoles[role].verb, chosen.User, g.changeRule(seat)),
		mentions: []types.JID{chosen},
		taken:    true,
	}
	if role == cityAssassin {
		pick.allies = g.alliesOf(seat)
		pick.tip = fmt.Sprintf("🔪 @%s quer matar @%s", g.players[seat].jid.User, chosen.User)
	}
	if g.nightDone() {
		g.wake()
	}
	return pick
}

func (g *cityGame) nightDone() bool {
	return !slices.ContainsFunc(g.seats(), func(seat int) bool { return !g.players[seat].dead && !g.acted(seat) })
}

func (g *cityGame) acted(seat int) bool {
	checks := map[cityRole]func() bool{
		cityAssassin:  func() bool { _, chose := g.kills[seat]; return chose },
		cityDoctor:    func() bool { return g.saved != cityNobody },
		cityDetective: func() bool { return g.probe != cityNobody },
	}
	check, acts := checks[g.players[seat].role]
	return !acts || check()
}

func (g *cityGame) seats() []int {
	seats := make([]int, len(g.players))
	for i := range seats {
		seats[i] = i
	}
	return seats
}

func (g *cityGame) dawn() cityDawn {
	g.mu.Lock()
	defer g.mu.Unlock()
	dawn := cityDawn{victim: cityNobody, sleuth: cityNobody, probe: g.probe}
	g.quiet = len(g.kills) == 0
	if target := g.target(); target != cityNobody && target != g.saved {
		g.players[target].dead = true
		dawn.victim = target
	}
	if g.probe != cityNobody {
		dawn.sleuth = slices.IndexFunc(g.players, func(p cityPlayer) bool { return p.role == cityDetective })
	}
	g.lastSaved = g.saved
	return dawn
}

func (g *cityGame) target() int {
	counts := map[int]int{}
	for _, target := range g.kills {
		counts[target]++
	}
	leaders := cityLeaders(counts)
	if len(leaders) == 0 {
		return cityNobody
	}
	return leaders[rand.IntN(len(leaders))]
}

func cityLeaders(counts map[int]int) []int {
	if len(counts) == 0 {
		return nil
	}
	top := slices.Max(slices.Collect(maps.Values(counts)))
	var leaders []int
	for _, seat := range slices.Sorted(maps.Keys(counts)) {
		if counts[seat] == top {
			leaders = append(leaders, seat)
		}
	}
	return leaders
}

func (g *cityGame) probeReport(dawn cityDawn) (string, []types.JID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	suspect := g.players[dawn.probe]
	if suspect.role == cityAssassin {
		return fmt.Sprintf("🔎 Resultado da investigação da noite %d: @%s *é assassino* 🔪", g.round, suspect.jid.User), []types.JID{suspect.jid}
	}
	return fmt.Sprintf("🔎 Resultado da investigação da noite %d: @%s *não é assassino* ✅", g.round, suspect.jid.User), []types.JID{suspect.jid}
}

func (g *cityGame) dawnText(dawn cityDawn) (string, []types.JID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	header := fmt.Sprintf("☀️ *Amanhece o dia %d*", g.round)
	if dawn.victim == cityNobody {
		return header + "\n\n😮‍💨 Ninguém morreu essa noite", nil
	}
	victim := g.players[dawn.victim]
	return fmt.Sprintf("%s\n\n⚰️ @%s foi encontrado morto. Era %s", header, victim.jid.User, cityRoleTag(victim.role)), []types.JID{victim.jid}
}

func (g *cityGame) daybreak(header string) (string, []types.JID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.votes = map[int]int{}
	g.enter(cityDay)
	return g.board(header)
}

func (g *cityGame) vote(info types.MessageInfo, target int) (string, []types.JID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase == cityLobby {
		return "⏳ A partida nem começou ainda", nil
	}
	if g.phase != cityDay {
		return "🌙 Agora é de noite, a votação é só de dia", nil
	}
	voter := g.seatOf(info)
	if voter < 0 {
		return "Você não tá jogando essa partida 🙄", nil
	}
	if g.players[voter].dead {
		return "☠️ Morto não vota", nil
	}
	if target == cityStranger || (target != cityNobody && (target < 0 || target >= len(g.players) || g.players[target].dead)) {
		return "Essa pessoa não tá viva na partida 🤔 Vota em alguém da lista do !cidadedorme", nil
	}
	if target == voter {
		return "Votar em você mesmo? 🤨", nil
	}
	g.votes[voter] = target
	if len(g.votes) == g.aliveCount() {
		g.wake()
	}
	who := g.players[voter].jid
	mentions := []types.JID{who}
	choice := "ninguém"
	if target != cityNobody {
		choice = "@" + g.players[target].jid.User
		mentions = append(mentions, g.players[target].jid)
	}
	line, counted := g.tally()
	progress := "todo mundo votou"
	if missing := g.aliveCount() - len(g.votes); missing > 0 {
		progress = "faltam " + cityVotes(missing)
	}
	return fmt.Sprintf("🗳️ @%s votou em %s\n%s · %s", who.User, choice, line, progress), append(mentions, counted...)
}

func (g *cityGame) counts() map[int]int {
	counts := map[int]int{}
	for _, target := range g.votes {
		counts[target]++
	}
	return counts
}

func (g *cityGame) tally() (string, []types.JID) {
	counts := g.counts()
	targets := slices.Sorted(maps.Keys(counts))
	slices.SortStableFunc(targets, func(a, b int) int { return cmp.Compare(counts[b], counts[a]) })
	parts := make([]string, len(targets))
	var mentions []types.JID
	for i, target := range targets {
		if target == cityNobody {
			parts[i] = fmt.Sprintf("ninguém: %d", counts[target])
			continue
		}
		parts[i] = fmt.Sprintf("@%s: %d", g.players[target].jid.User, counts[target])
		mentions = append(mentions, g.players[target].jid)
	}
	return "📊 " + strings.Join(parts, " · "), mentions
}

func (g *cityGame) verdict() (string, []types.JID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.quiet && len(g.votes) == 0 {
		g.idle++
	} else {
		g.idle = 0
	}
	header := fmt.Sprintf("🗳️ *Fim da votação do dia %d*", g.round)
	if len(g.votes) == 0 {
		return header + "\n\n🦗 Ninguém votou, então ninguém foi eliminado", nil
	}
	line, mentions := g.tally()
	leaders := cityLeaders(g.counts())
	text := header + "\n\n" + line
	if len(leaders) > 1 {
		return text + "\n\n🤝 Deu empate, então ninguém foi eliminado", mentions
	}
	if leaders[0] == cityNobody {
		return text + "\n\n🙅 A cidade decidiu não eliminar ninguém hoje", mentions
	}
	out := &g.players[leaders[0]]
	out.dead = true
	return fmt.Sprintf("%s\n\n⚖️ A cidade eliminou @%s, que era %s", text, out.jid.User, cityRoleTag(out.role)), append(mentions, out.jid)
}

func (g *cityGame) ending() (string, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	assassins := 0
	for _, p := range g.players {
		if !p.dead && p.role == cityAssassin {
			assassins++
		}
	}
	others := g.aliveCount() - assassins
	if assassins == 0 {
		return "🎉 *A cidade venceu!* Não sobrou nenhum assassino", true
	}
	if assassins >= others {
		return "🔪 *Os assassinos venceram!* Agora eles são maioria na cidade", true
	}
	if g.idle >= cityIdleLimit {
		return "😴 A cidade parou de jogar, então a partida acabou", true
	}
	return "", false
}

func (g *cityGame) aliveCount() int {
	alive := 0
	for _, p := range g.players {
		if !p.dead {
			alive++
		}
	}
	return alive
}

func (g *cityGame) close(header string) (string, []types.JID, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase == cityOver {
		return "", nil, false
	}
	g.halt()
	return header + "\n\n" + g.reveal(), g.jids(), true
}

func (g *cityGame) reveal() string {
	lines := []string{"🎭 *Papéis da partida:*"}
	for _, p := range g.players {
		line := fmt.Sprintf("• @%s: %s %s", p.jid.User, cityRoles[p.role].name, cityRoles[p.role].emoji)
		if p.dead {
			line += " ☠️"
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (g *cityGame) status(header string) (string, []types.JID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.board(header)
}

func (g *cityGame) overview() (string, []types.JID) {
	titles := map[cityPhase]string{
		cityLobby: "🚪 *Sala do Cidade Dorme aberta*",
		cityNight: fmt.Sprintf("🌙 *Cidade Dorme: noite %d*", g.round),
		cityDay:   fmt.Sprintf("☀️ *Cidade Dorme: dia %d*", g.round),
		cityOver:  "🏁 *Essa partida já acabou*",
	}
	return g.board(titles[g.phase])
}

func (g *cityGame) board(header string) (string, []types.JID) {
	lines := []string{header, ""}
	if g.phase == cityLobby {
		lines = append(lines, fmt.Sprintf("👥 *Na sala* (máximo %d):", cityMaxPlayers))
	} else {
		lines = append(lines, g.cast(), "", fmt.Sprintf("🏘️ *Moradores* (%d vivos):", g.aliveCount()))
	}
	for seat, p := range g.players {
		line := fmt.Sprintf("%d. @%s", seat+1, p.jid.User)
		if p.dead {
			line += fmt.Sprintf(" ☠️ (era %s)", cityRoleTag(p.role))
		}
		lines = append(lines, line)
	}
	hints := map[cityPhase]func() string{
		cityLobby: func() string {
			return fmt.Sprintf("_Manda !cidadedorme pra entrar (mínimo de %d jogadores). A partida começa sozinha em %s, ou antes com !cidadedorme começar_ ⏳", cityMinPlayers, core.Countdown(g.deadline))
		},
		cityNight: func() string {
			return fmt.Sprintf("_Assassino, médico e detetive: olhem o meu privado, vocês têm %s pra agir. O resto, boa noite_ 😴", core.Countdown(g.deadline))
		},
		cityDay: func() string {
			return fmt.Sprintf("_☀️ Discutam quem é o assassino e votem com !votar @pessoa (ou o número da lista, ou !votar ninguem). Só vivos votam, e morto não fala 🤐 A votação fecha em %s, ou antes se todo mundo votar_ 🗳️", core.Countdown(g.deadline))
		},
		cityOver: func() string { return "_Manda !cidadedorme pra abrir outra sala_" },
	}
	lines = append(lines, "", hints[g.phase]())
	return strings.Join(lines, "\n"), g.jids()
}

func (g *cityGame) jids() []types.JID {
	return cityJIDs(g.players)
}

func cityJIDs(players []cityPlayer) []types.JID {
	jids := make([]types.JID, len(players))
	for i, p := range players {
		jids[i] = p.jid
	}
	return jids
}

func cityRoleTag(role cityRole) string {
	return fmt.Sprintf("*%s* %s", cityRoles[role].name, cityRoles[role].emoji)
}

func cityVotes(n int) string {
	if n == 1 {
		return "1 voto"
	}
	return fmt.Sprintf("%d votos", n)
}
