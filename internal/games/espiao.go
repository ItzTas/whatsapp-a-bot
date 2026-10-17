package games

import (
	"cmp"
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
	espiaoMinPlayers   = 3
	espiaoMaxPlayers   = 10
	espiaoBoardPlaces  = 12
	espiaoLobbyTime    = 90 * time.Second
	espiaoTalkTime     = 8 * time.Minute
	espiaoTurnLimit    = 90 * time.Second
	espiaoPollTime     = time.Minute
	espiaoIdleLimit    = 3
	espiaoCrewReward   = 40
	espiaoSpyReward    = 120
	espiaoGameKind     = "espiao"
	espiaoGuessCommand = "chutar"
	espiaoStarted      = "🕶️ O jogo já começou, agora vai até o fim"
)

type espiaoPhase int

const (
	espiaoLobby espiaoPhase = iota
	espiaoTalk
	espiaoAccusing
	espiaoVoting
	espiaoOver
)

type espiaoStage struct {
	title  string
	roster string
	places bool
	hint   func(g *espiaoGame) string
}

var espiaoStages = map[espiaoPhase]espiaoStage{
	espiaoLobby: {
		title:  "🚪 *Sala do espião aberta*",
		roster: fmt.Sprintf("👥 *Na sala* (máximo %d):", espiaoMaxPlayers),
		hint: func(g *espiaoGame) string {
			return fmt.Sprintf("_Manda !espiao ou responde esta mensagem só com ! pra entrar (mínimo de %d jogadores). O jogo começa sozinho em %s, ou antes com !espiao começar_ ⏳", espiaoMinPlayers, core.Countdown(g.deadline))
		},
	},
	espiaoTalk: {
		title:  "🕶️ *Espião rolando: hora das perguntas*",
		roster: "👥 *Jogadores:*",
		places: true,
		hint:   (*espiaoGame).talkHint,
	},
	espiaoAccusing: {
		title:  "🚨 *Espião rolando: acusação na enquete*",
		roster: "👥 *Jogadores:*",
		hint: func(g *espiaoGame) string {
			return fmt.Sprintf("_O relógio parou. Todo mundo menos @%s vota na enquete: se mais da metade disser que é o espião, o jogo acaba e a gente descobre se acertaram. Fecha em %s, ou antes se a maioria já tiver decidido_ ⏳", g.players[g.suspect].jid.User, core.Countdown(g.deadline))
		},
	},
	espiaoVoting: {
		title:  "🗳️ *Espião rolando: votação final*",
		roster: "👥 *Jogadores:*",
		hint: func(g *espiaoGame) string {
			return fmt.Sprintf("_Vota na enquete em quem você acha que é o espião (só vale voto de quem tá jogando). Se o mais votado for o espião, o grupo ganha; se errarem ou der empate, ganha o espião. Fecha em %s, ou antes se a maioria já tiver decidido_ ⏳", core.Countdown(g.deadline))
		},
	},
	espiaoOver: {
		title:  "🏁 *Esse espião já acabou*",
		roster: "👥 *Jogadores:*",
		hint:   func(*espiaoGame) string { return "_Manda !espiao pra abrir outra sala_" },
	},
}

type espiaoGame struct {
	mu          sync.Mutex
	phase       espiaoPhase
	players     []roomPlayer
	spy         int
	places      []string
	place       int
	roles       []string
	asker       int
	previous    int
	asked       int
	active      bool
	idle        int
	accused     []bool
	accuser     int
	suspect     int
	deadline    time.Time
	turnUntil   time.Time
	paused      time.Duration
	notices     []types.MessageID
	poll        types.MessageID
	pollOptions []string
	ballot      *roomBallot
	guessPoll   types.MessageID
	nudge       chan struct{}
	dropped     bool
	prompt      *core.PrivatePrompt
}

type espiaoState struct {
	Phase     espiaoPhase       `json:"phase"`
	Players   []savedRoomPlayer `json:"players"`
	Spy       int               `json:"spy"`
	Places    []string          `json:"places"`
	Place     int               `json:"place"`
	Roles     []string          `json:"roles"`
	Asker     int               `json:"asker"`
	Previous  int               `json:"previous"`
	Asked     int               `json:"asked"`
	Idle      int               `json:"idle"`
	Accused   []bool            `json:"accused"`
	Accuser   int               `json:"accuser"`
	Suspect   int               `json:"suspect"`
	Deadline  time.Time         `json:"deadline"`
	TurnUntil time.Time         `json:"turn_until"`
	Paused    time.Duration     `json:"paused"`
	Notices   []types.MessageID `json:"notices"`
	Poll      types.MessageID   `json:"poll"`
	Options   []string          `json:"options"`
	Choices   map[int]int       `json:"choices"`
	GuessPoll types.MessageID   `json:"guess_poll"`
}

type espiaoEnding struct {
	header string
	spyWon bool
	void   bool
}

type espiaoReveal struct {
	text     string
	mentions []types.JID
	winners  []types.JID
	players  int
	prize    int64
}

type espiaoTick struct {
	text     string
	mentions []types.JID
	idle     bool
}

type espiaoAction func(g *espiaoGame, info types.MessageInfo, mentions []types.JID) (string, []types.JID)

type espiaoStep func(ctx context.Context, chat types.JID, game *espiaoGame) bool

func (b *bot) espiao(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return b.SendText(ctx, chat, "O !espiao só funciona em grupo")
	}
	actions := map[string]espiaoAction{
		"comecar":  (*espiaoGame).start,
		"sair":     (*espiaoGame).leave,
		"cancelar": (*espiaoGame).cancel,
		"acusar":   (*espiaoGame).accuse,
	}
	action, named := actions[core.FoldLetters(core.FirstPlainArg(core.MessageText(evt.Message)))]
	game, active := b.activeEspiao(chat)
	if named && !active {
		return b.SendText(ctx, chat, "Não tem espião rolando agora. Manda !espiao pra abrir uma sala 🕶️")
	}
	if named {
		text, mentions := action(game, evt.Info, core.MentionedJIDs(evt.Message))
		return b.postEspiao(ctx, chat, game, text, mentions)
	}
	if !active {
		return b.openEspiao(ctx, chat, roomPlayer{jid: evt.Info.Sender.ToNonAD(), ids: b.SenderIdentities(ctx, evt)}, evt.Info)
	}
	return b.joinEspiao(ctx, evt, game)
}

func (b *bot) answerEspiao(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	game, active := b.activeEspiao(chat)
	if !active {
		return nil
	}
	text := strings.TrimSpace(core.MessageText(evt.Message))
	if text == "!" {
		return b.joinEspiaoByReply(ctx, evt, game)
	}
	if core.IsCommand(text) || !game.ask(evt.Info, core.MentionedJIDs(evt.Message)) {
		return nil
	}
	b.React(ctx, evt, "🎤")
	b.syncEspiao(ctx, chat, game)
	return nil
}

func (b *bot) joinEspiaoByReply(ctx context.Context, evt *events.Message, game *espiaoGame) error {
	quoted := evt.Message.GetExtendedTextMessage().GetContextInfo().GetStanzaID()
	if quoted == "" || !game.announced(quoted) {
		return nil
	}
	return b.joinEspiao(ctx, evt, game)
}

func (b *bot) joinEspiao(ctx context.Context, evt *events.Message, game *espiaoGame) error {
	player := roomPlayer{jid: evt.Info.Sender.ToNonAD(), ids: b.SenderIdentities(ctx, evt)}
	text, mentions := game.join(evt.Info, player)
	return b.postEspiao(ctx, evt.Info.Chat.ToNonAD(), game, text, mentions)
}

func (b *bot) activeEspiao(chat types.JID) (*espiaoGame, bool) {
	value, ok := b.espiaoGames.Load(chat.String())
	if !ok {
		return nil, false
	}
	game := value.(*espiaoGame)
	if game.over() {
		return nil, false
	}
	return game, true
}

func (b *bot) openEspiao(ctx context.Context, chat types.JID, host roomPlayer, info types.MessageInfo) error {
	game := newEspiaoGame(host)
	if value, loaded := b.espiaoGames.LoadOrStore(chat.String(), game); loaded {
		open := value.(*espiaoGame)
		text, mentions := open.join(info, host)
		return b.postEspiao(ctx, chat, open, text, mentions)
	}
	go b.driveEspiao(ctx, chat, game)
	text, mentions := game.status(fmt.Sprintf("🕶️ *Quem é o espião?*\n\n@%s abriu a sala! Todo mundo recebe no privado o mesmo local secreto e um papel nele, menos um: o espião, que não faz ideia de onde vocês estão. Aí rolam perguntas aqui no grupo pra caçar o espião, enquanto ele tenta descobrir o local sem se entregar", host.jid.User))
	return b.postEspiao(ctx, chat, game, text, mentions)
}

func (b *bot) postEspiao(ctx context.Context, chat types.JID, game *espiaoGame, text string, mentions []types.JID) error {
	defer b.syncEspiao(ctx, chat, game)
	if text == "" {
		return nil
	}
	id, err := b.SendMentions(ctx, chat, text, mentions)
	if err != nil {
		return err
	}
	game.remember(id)
	return nil
}

func (b *bot) tellEspiao(ctx context.Context, chat types.JID, text string, mentions []types.JID) {
	if text == "" {
		return
	}
	if _, err := b.SendMentions(ctx, chat, text, mentions); err != nil {
		b.Client.Log.Errorf("mandar mensagem do !espiao: %v", err)
	}
}

func (b *bot) restoreEspiao(ctx context.Context, saved core.SavedGame) error {
	var state espiaoState
	if err := json.Unmarshal(saved.State, &state); err != nil {
		return fmt.Errorf("ler o espião salvo: %w", err)
	}
	chat, err := types.ParseJID(saved.Key)
	if err != nil {
		return fmt.Errorf("ler a conversa do espião salvo: %w", err)
	}
	game := state.game()
	if _, loaded := b.espiaoGames.LoadOrStore(saved.Key, game); loaded {
		return nil
	}
	if game.ballot != nil {
		game.ballot.saved = func() { b.syncEspiao(ctx, chat, game) }
		b.espiaoPolls.Store(game.poll, game.ballot)
	}
	go b.driveEspiao(ctx, chat, game)
	return nil
}

func (s espiaoState) game() *espiaoGame {
	game := &espiaoGame{
		phase:     s.Phase,
		players:   loadRoom(s.Players),
		spy:       s.Spy,
		places:    s.Places,
		place:     s.Place,
		roles:     s.Roles,
		asker:     s.Asker,
		previous:  s.Previous,
		asked:     s.Asked,
		idle:      s.Idle,
		accused:   s.Accused,
		accuser:   s.Accuser,
		suspect:   s.Suspect,
		deadline:  s.Deadline,
		turnUntil: s.TurnUntil,
		paused:    s.Paused,
		notices:   s.Notices,
		guessPoll: s.GuessPoll,
		nudge:     make(chan struct{}, 1),
	}
	if s.Poll == "" {
		return game
	}
	game.poll, game.pollOptions = s.Poll, s.Options
	game.ballot = newRoomBallot(game.voters(), s.Options)
	game.ballot.restore(s.Choices)
	return game
}

func (b *bot) driveEspiao(ctx context.Context, chat types.JID, game *espiaoGame) {
	defer b.releaseEspiao(ctx, chat, game)
	steps := map[espiaoPhase]espiaoStep{
		espiaoLobby:    b.closeEspiaoLobby,
		espiaoTalk:     b.talkEspiao,
		espiaoAccusing: b.espiaoPollStep(b.openEspiaoAccusation, b.settleEspiaoAccusation),
		espiaoVoting:   b.espiaoPollStep(b.openEspiaoVote, b.settleEspiaoVote),
	}
	for {
		step, running := steps[game.currentPhase()]
		if !running {
			return
		}
		b.claimEspiaoGuess(chat, game)
		if !step(ctx, chat, game) {
			return
		}
	}
}

func (b *bot) releaseEspiao(ctx context.Context, chat types.JID, game *espiaoGame) {
	if game.prompt != nil {
		b.DropPrivate(game.prompt)
	}
	if id, _ := game.openedPoll(); id != "" {
		b.espiaoPolls.Delete(id)
	}
	b.syncEspiao(ctx, chat, game)
	if game.over() {
		b.espiaoGames.CompareAndDelete(chat.String(), game)
	}
}

func (b *bot) syncEspiao(ctx context.Context, chat types.JID, game *espiaoGame) {
	game.mu.Lock()
	defer game.mu.Unlock()
	if game.phase != espiaoOver {
		b.KeepGame(ctx, espiaoGameKind, chat.String(), game.snapshot(), game.deadline)
		return
	}
	if game.dropped {
		return
	}
	game.dropped = true
	b.DropGame(ctx, espiaoGameKind, chat.String())
}

func (b *bot) claimEspiaoGuess(chat types.JID, game *espiaoGame) {
	if game.prompt != nil {
		return
	}
	spy, poll, places, dealt := game.spyTarget()
	if !dealt {
		return
	}
	prompt, free := b.AwaitPrivate(spy, b.espiaoGuess(chat, game), espiaoGuessCommand)
	if !free {
		return
	}
	if poll != "" {
		prompt.WatchPoll(poll, places)
	}
	game.prompt = prompt
}

func (b *bot) espiaoGuess(chat types.JID, game *espiaoGame) core.PrivateAnswer {
	return func(ctx context.Context, evt *events.Message, text string) (bool, error) {
		ending, found := game.guess(text)
		if !found {
			return false, b.SendText(ctx, evt.Info.Chat, fmt.Sprintf("🤔 Não achei esse local. Vota na enquete ou manda só o número dele na lista (1 a %d)", espiaoBoardPlaces))
		}
		reply := "🎯 Chute anotado! O resultado saiu lá no grupo"
		if !b.finishEspiao(ctx, chat, game, ending) {
			reply = "🏁 Esse espião já acabou"
		}
		return true, b.SendText(ctx, evt.Info.Chat, reply)
	}
}

func (b *bot) closeEspiaoLobby(ctx context.Context, chat types.JID, game *espiaoGame) bool {
	if !game.sleep(ctx, nil) {
		return false
	}
	dealt, short := game.closeLobby()
	if short {
		b.tellEspiao(ctx, chat, fmt.Sprintf("😴 Não juntou gente suficiente e o espião foi cancelado (precisa de pelo menos %d jogadores)", espiaoMinPlayers), nil)
	}
	if dealt {
		b.startEspiao(ctx, chat, game)
	}
	return true
}

func (b *bot) startEspiao(ctx context.Context, chat types.JID, game *espiaoGame) {
	b.syncEspiao(ctx, chat, game)
	b.claimEspiaoGuess(chat, game)
	if unreached := b.briefEspiao(ctx, chat, game); len(unreached) > 0 {
		b.finishEspiao(ctx, chat, game, espiaoEnding{header: fmt.Sprintf("📵 Não consegui mandar mensagem no privado de %s, então o espião foi cancelado. Todo mundo que joga precisa receber mensagem minha no privado", core.JoinMentions(unreached)), void: true})
		return
	}
	text, mentions := game.status("🕶️ *Começou o espião!*\n\n📨 Mandei no privado de cada um o local e o seu papel nele. Só o espião não sabe onde vocês estão, e se ele chutar o local certo no meu privado, ganha na hora")
	b.syncEspiao(ctx, chat, game)
	b.tellEspiao(ctx, chat, text, mentions)
}

func (b *bot) briefEspiao(ctx context.Context, chat types.JID, game *espiaoGame) []types.JID {
	label := b.PrivateGroupLabel(ctx, chat)
	players, spy, places := game.cast()
	prompt := game.prompt
	unreached := make([]types.JID, len(players))
	var wg sync.WaitGroup
	for seat, p := range players {
		wg.Go(func() {
			if !b.SendPrivate(ctx, p.ids, game.briefing(seat, label, prompt != nil), nil) {
				unreached[seat] = p.jid
				return
			}
			if seat == spy {
				b.sendEspiaoGuessPoll(ctx, game, p, places, prompt)
			}
		})
	}
	wg.Wait()
	return slices.DeleteFunc(unreached, func(jid types.JID) bool { return jid.IsEmpty() })
}

func (b *bot) sendEspiaoGuessPoll(ctx context.Context, game *espiaoGame, spy roomPlayer, places []string, prompt *core.PrivatePrompt) {
	id, sent := b.SendPrivatePoll(ctx, spy.ids, "🎯 Onde vocês estão? (é um chute só)", places)
	if !sent {
		b.SendPrivate(ctx, spy.ids, "📍 *Locais possíveis:*\n"+strings.Join(espiaoPlaceLines(places), "\n"), nil)
		return
	}
	game.keepGuessPoll(id)
	if prompt != nil {
		prompt.WatchPoll(id, places)
	}
}

func (b *bot) talkEspiao(ctx context.Context, chat types.JID, game *espiaoGame) bool {
	if !game.sleep(ctx, nil) {
		return false
	}
	tick := game.tick()
	if tick.idle {
		b.finishEspiao(ctx, chat, game, espiaoEnding{header: fmt.Sprintf("😴 Ninguém do jogo falou nada em %s, então o espião foi cancelado", core.FormatWait(espiaoIdleLimit*espiaoTurnLimit)), void: true})
		return true
	}
	b.syncEspiao(ctx, chat, game)
	b.tellEspiao(ctx, chat, tick.text, tick.mentions)
	return true
}

func (b *bot) espiaoPollStep(open espiaoStep, settle func(ctx context.Context, chat types.JID, game *espiaoGame, votes []int)) espiaoStep {
	return func(ctx context.Context, chat types.JID, game *espiaoGame) bool {
		id, ballot := game.openedPoll()
		if ballot == nil {
			return open(ctx, chat, game)
		}
		if !game.sleep(ctx, ballot.decided) {
			return false
		}
		if !game.pollDue() {
			return true
		}
		b.espiaoPolls.Delete(id)
		settle(ctx, chat, game, ballot.tally())
		return true
	}
}

func (b *bot) openEspiaoPoll(ctx context.Context, chat types.JID, game *espiaoGame, header, question string, options []string) bool {
	text, mentions, live := game.callVote(header)
	if !live {
		return true
	}
	b.tellEspiao(ctx, chat, text, mentions)
	resp, err := b.Client.SendMessage(ctx, chat, b.Client.BuildPollCreation(question, options, 1))
	if err != nil {
		b.Client.Log.Errorf("abrir enquete do !espiao: %v", err)
		return false
	}
	ballot := game.openPoll(resp.ID, options)
	ballot.saved = func() { b.syncEspiao(ctx, chat, game) }
	b.espiaoPolls.Store(resp.ID, ballot)
	b.syncEspiao(ctx, chat, game)
	return true
}

func (b *bot) openEspiaoAccusation(ctx context.Context, chat types.JID, game *espiaoGame) bool {
	accuser, suspect := game.accusation()
	name := b.pollNames(ctx, chat, []types.JID{suspect})[0]
	header := fmt.Sprintf("🚨 *@%s acusou @%s de ser o espião!*", accuser.User, suspect.User)
	if b.openEspiaoPoll(ctx, chat, game, header, "🚨 "+name+" é o espião?", []string{"✅ É o espião", "❌ Não é"}) {
		return true
	}
	if ctx.Err() != nil {
		return false
	}
	text, mentions := game.dropAccusation()
	b.syncEspiao(ctx, chat, game)
	b.tellEspiao(ctx, chat, text, mentions)
	return true
}

func (b *bot) settleEspiaoAccusation(ctx context.Context, chat types.JID, game *espiaoGame, votes []int) {
	if ending, convicted := game.conviction(votes); convicted {
		b.finishEspiao(ctx, chat, game, ending)
		return
	}
	text, mentions := game.acquit(votes)
	b.syncEspiao(ctx, chat, game)
	b.tellEspiao(ctx, chat, text, mentions)
}

func (b *bot) openEspiaoVote(ctx context.Context, chat types.JID, game *espiaoGame) bool {
	names := b.pollNames(ctx, chat, game.roster())
	if b.openEspiaoPoll(ctx, chat, game, "⏰ *Acabou o tempo das perguntas!*", "🗳️ Quem é o espião?", names) {
		return true
	}
	if ctx.Err() != nil {
		return false
	}
	b.finishEspiao(ctx, chat, game, espiaoEnding{header: "😵 Não consegui abrir a enquete da votação, então o espião foi cancelado", void: true})
	return true
}

func (b *bot) settleEspiaoVote(ctx context.Context, chat types.JID, game *espiaoGame, votes []int) {
	b.finishEspiao(ctx, chat, game, game.verdict(votes))
}

func (b *bot) finishEspiao(ctx context.Context, chat types.JID, game *espiaoGame, ending espiaoEnding) bool {
	reveal, closed := game.close(ending)
	if !closed {
		return false
	}
	b.syncEspiao(ctx, chat, game)
	text := reveal.text
	if reward := b.economy.Reward(ctx, chat, reveal.winners, reveal.players, reveal.prize); reward != "" {
		text += "\n\n" + reward
	}
	b.tellEspiao(ctx, chat, text, reveal.mentions)
	return true
}

func newEspiaoGame(host roomPlayer) *espiaoGame {
	return &espiaoGame{
		players:  []roomPlayer{host},
		deadline: time.Now().Add(espiaoLobbyTime),
		nudge:    make(chan struct{}, 1),
	}
}

func (g *espiaoGame) poke() {
	select {
	case g.nudge <- struct{}{}:
	default:
	}
}

func (g *espiaoGame) enter(phase espiaoPhase) {
	g.phase = phase
	g.poll, g.pollOptions, g.ballot = "", nil, nil
}

func (g *espiaoGame) halt() {
	g.phase = espiaoOver
	g.poke()
}

func (g *espiaoGame) sleep(ctx context.Context, decided <-chan struct{}) bool {
	timer := time.NewTimer(time.Until(g.alarm()))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-g.nudge:
	case <-decided:
	case <-timer.C:
	}
	return true
}

func (g *espiaoGame) alarm() time.Time {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase == espiaoTalk && g.turnUntil.Before(g.deadline) {
		return g.turnUntil
	}
	return g.deadline
}

func (g *espiaoGame) currentPhase() espiaoPhase {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.phase
}

func (g *espiaoGame) over() bool {
	return g.currentPhase() == espiaoOver
}

func (g *espiaoGame) snapshot() espiaoState {
	state := espiaoState{
		Phase:     g.phase,
		Players:   saveRoom(g.players),
		Spy:       g.spy,
		Places:    g.places,
		Place:     g.place,
		Roles:     g.roles,
		Asker:     g.asker,
		Previous:  g.previous,
		Asked:     g.asked,
		Idle:      g.idle,
		Accused:   g.accused,
		Accuser:   g.accuser,
		Suspect:   g.suspect,
		Deadline:  g.deadline,
		TurnUntil: g.turnUntil,
		Paused:    g.paused,
		Notices:   g.notices,
		Poll:      g.poll,
		Options:   g.pollOptions,
		GuessPoll: g.guessPoll,
	}
	if g.ballot != nil {
		state.Choices = g.ballot.chosen()
	}
	return state
}

func (g *espiaoGame) remember(id types.MessageID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.notices = append(g.notices, id)
}

func (g *espiaoGame) announced(id types.MessageID) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return slices.Contains(g.notices, id)
}

func (g *espiaoGame) join(info types.MessageInfo, player roomPlayer) (string, []types.JID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase != espiaoLobby {
		return g.overview()
	}
	if roomSeat(g.players, info) >= 0 {
		return g.board(fmt.Sprintf("🙋 Você já tá na sala, @%s", player.jid.User))
	}
	if len(g.players) >= espiaoMaxPlayers {
		return fmt.Sprintf("🚫 A sala já tá cheia (%d jogadores), espera a próxima rodada", espiaoMaxPlayers), nil
	}
	g.players = append(g.players, player)
	header := fmt.Sprintf("✅ @%s entrou na sala!", player.jid.User)
	if len(g.players) < espiaoMaxPlayers {
		return g.board(header)
	}
	g.deadline = time.Now()
	g.poke()
	return header + "\n\n🚀 A sala lotou, começando o jogo!", []types.JID{player.jid}
}

func (g *espiaoGame) start(info types.MessageInfo, _ []types.JID) (string, []types.JID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase != espiaoLobby {
		return g.overview()
	}
	if roomSeat(g.players, info) < 0 {
		return "Só quem tá jogando pode fazer isso 🙄", nil
	}
	if len(g.players) < espiaoMinPlayers {
		return g.board(fmt.Sprintf("⏳ Ainda não dá pra começar: precisa de pelo menos %d jogadores", espiaoMinPlayers))
	}
	g.deadline = time.Now()
	g.poke()
	return "", nil
}

func (g *espiaoGame) leave(info types.MessageInfo, _ []types.JID) (string, []types.JID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase != espiaoLobby {
		return espiaoStarted, nil
	}
	seat := roomSeat(g.players, info)
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
	return header + " e ela ficou vazia, então o espião foi cancelado", []types.JID{gone}
}

func (g *espiaoGame) cancel(info types.MessageInfo, _ []types.JID) (string, []types.JID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase != espiaoLobby {
		return espiaoStarted, nil
	}
	host := g.players[0].jid
	if roomSeat(g.players, info) != 0 {
		return fmt.Sprintf("Só o primeiro da lista (@%s) pode cancelar a sala 🙄", host.User), []types.JID{host}
	}
	g.halt()
	return fmt.Sprintf("❌ @%s cancelou o espião", host.User), []types.JID{host}
}

func (g *espiaoGame) closeLobby() (dealt, short bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase != espiaoLobby || time.Now().Before(g.deadline) {
		return false, false
	}
	if len(g.players) < espiaoMinPlayers {
		g.halt()
		return false, true
	}
	g.deal()
	return true, false
}

func (g *espiaoGame) deal() {
	locations := core.PickSome(espiaoLocations, espiaoBoardPlaces)
	g.places = make([]string, len(locations))
	for i, location := range locations {
		g.places[i] = location.label()
	}
	g.place = rand.IntN(len(locations))
	g.spy = rand.IntN(len(g.players))
	roles := core.PickSome(locations[g.place].roles, len(locations[g.place].roles))
	g.roles = make([]string, len(g.players))
	crew := 0
	for seat := range g.roles {
		if seat == g.spy {
			continue
		}
		g.roles[seat] = roles[crew%len(roles)]
		crew++
	}
	g.accused = make([]bool, len(g.players))
	g.asker, g.previous = rand.IntN(len(g.players)), -1
	now := time.Now()
	g.enter(espiaoTalk)
	g.deadline, g.turnUntil = now.Add(espiaoTalkTime), now.Add(espiaoTurnLimit)
}

func (g *espiaoGame) cast() ([]roomPlayer, int, []string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return slices.Clone(g.players), g.spy, slices.Clone(g.places)
}

func (g *espiaoGame) spyTarget() ([]types.JID, types.MessageID, []string, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.places == nil {
		return nil, "", nil, false
	}
	return g.players[g.spy].ids, g.guessPoll, slices.Clone(g.places), true
}

func (g *espiaoGame) keepGuessPoll(id types.MessageID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.guessPoll = id
}

func (g *espiaoGame) roster() []types.JID {
	g.mu.Lock()
	defer g.mu.Unlock()
	return roomJIDs(g.players)
}

func (g *espiaoGame) briefing(seat int, label string, claimed bool) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	title := "🕶️ *Espião*" + label
	if seat != g.spy {
		return fmt.Sprintf("%s\n\n📍 Local: *%s*\n🎭 Seu papel: *%s*\n\nUm de vocês é o espião e não sabe onde vocês estão. Nas perguntas lá no grupo, mostra que você sabe o local sem entregar demais, senão o espião descobre 🤫", title, g.places[g.place], g.roles[seat])
	}
	guess := "🎯 Achou que sabe o local? Vota nele na enquete aqui embaixo ou me manda o número dele na lista. Acertou, você ganha na hora; errou, perde. É um chute só"
	if !claimed {
		guess = "⚠️ Você tá com outra resposta pendente no meu privado (tipo um !jokenpo). Assim que ela acabar, dá pra chutar o local votando na enquete aqui embaixo"
	}
	return fmt.Sprintf("%s\n\n🕵️ Você é o *espião*! Todo mundo sabe onde vocês estão, menos você. Presta atenção nas perguntas lá no grupo e responde sem deixar ninguém perceber que você tá perdido 🤫\n\n%s", title, guess)
}

func (g *espiaoGame) ask(info types.MessageInfo, mentions []types.JID) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	seat := roomSeat(g.players, info)
	if g.phase != espiaoTalk || seat < 0 {
		return false
	}
	g.active = true
	target := g.otherMember(mentions, seat, g.previous)
	if seat != g.asker || target < 0 {
		return false
	}
	g.previous, g.asker = seat, target
	g.turnUntil = time.Now().Add(espiaoTurnLimit)
	g.asked++
	return true
}

func (g *espiaoGame) otherMember(mentions []types.JID, seat, skip int) int {
	for _, mention := range mentions {
		target := memberSeat(g.players, mention)
		if target >= 0 && target != seat && target != skip {
			return target
		}
	}
	return -1
}

func (g *espiaoGame) tick() espiaoTick {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now()
	if g.phase != espiaoTalk || now.Before(g.turnUntil) && now.Before(g.deadline) {
		return espiaoTick{}
	}
	if !now.Before(g.deadline) {
		g.enter(espiaoVoting)
		return espiaoTick{}
	}
	return g.skipTurn(now)
}

func (g *espiaoGame) skipTurn(now time.Time) espiaoTick {
	g.idle++
	if g.active {
		g.idle = 0
	}
	g.active = false
	if g.idle >= espiaoIdleLimit {
		return espiaoTick{idle: true}
	}
	stalled := g.players[g.asker].jid
	g.asker = (g.asker + 1 + rand.IntN(len(g.players)-1)) % len(g.players)
	g.previous = -1
	g.turnUntil = now.Add(espiaoTurnLimit)
	next := g.players[g.asker].jid
	text := fmt.Sprintf("⏰ @%s demorou demais pra perguntar, então a vez passou pra @%s\n\n_Pra perguntar, marca a pessoa na mensagem. As perguntas acabam em %s_ ⏳", stalled.User, next.User, core.Countdown(g.deadline))
	return espiaoTick{text: text, mentions: []types.JID{stalled, next}}
}

func (g *espiaoGame) accuse(info types.MessageInfo, mentions []types.JID) (string, []types.JID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	refusals := map[espiaoPhase]string{
		espiaoLobby:    "⏳ O jogo nem começou ainda, não tem ninguém pra acusar",
		espiaoAccusing: "⏳ Já tem uma acusação na enquete, espera ela fechar",
		espiaoVoting:   "🗳️ Agora é a votação final, vota na enquete",
		espiaoOver:     "🏁 Esse espião já acabou",
	}
	if refusal, refused := refusals[g.phase]; refused {
		return refusal, nil
	}
	seat := roomSeat(g.players, info)
	if seat < 0 {
		return "Só quem tá jogando pode acusar 🙄", nil
	}
	if g.accused[seat] {
		return "🚫 Você já usou a sua acusação nesse jogo", nil
	}
	suspect := g.otherMember(mentions, seat, -1)
	if suspect < 0 {
		return "Marca quem você quer acusar, tipo !espiao acusar @pessoa (tem que ser alguém que tá jogando, e não vale você mesmo)", nil
	}
	g.accused[seat] = true
	g.accuser, g.suspect = seat, suspect
	g.paused = time.Until(g.deadline)
	g.enter(espiaoAccusing)
	g.deadline = time.Now().Add(espiaoPollTime)
	g.poke()
	return "", nil
}

func (g *espiaoGame) accusation() (types.JID, types.JID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.players[g.accuser].jid, g.players[g.suspect].jid
}

func (g *espiaoGame) dropAccusation() (string, []types.JID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase != espiaoAccusing {
		return "", nil
	}
	accuser := g.players[g.accuser].jid
	g.accused[g.accuser] = false
	g.resumeTalk()
	return fmt.Sprintf("😵 Não consegui abrir a enquete, então a acusação não valeu e @%s pode acusar de novo\n\n%s", accuser.User, g.clock()), []types.JID{accuser, g.players[g.asker].jid}
}

func (g *espiaoGame) conviction(votes []int) (espiaoEnding, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase != espiaoAccusing || votes[0]*2 <= len(g.players)-1 {
		return espiaoEnding{}, false
	}
	verdict := fmt.Sprintf("🚨 A acusação de @%s contra @%s passou (%d a %d)", g.players[g.accuser].jid.User, g.players[g.suspect].jid.User, votes[0], votes[1])
	if g.suspect == g.spy {
		return espiaoEnding{header: verdict + " e acertaram em cheio: era o espião! *Vitória do grupo* 🎉"}, true
	}
	return espiaoEnding{header: verdict + ", mas era inocente 😬 *Vitória do espião!* 😈", spyWon: true}, true
}

func (g *espiaoGame) acquit(votes []int) (string, []types.JID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase != espiaoAccusing {
		return "", nil
	}
	suspect := g.players[g.suspect].jid
	g.resumeTalk()
	return fmt.Sprintf("😮‍💨 A acusação contra @%s não passou (%d a favor, %d contra)\n\n%s", suspect.User, votes[0], votes[1], g.clock()), []types.JID{suspect, g.players[g.asker].jid}
}

func (g *espiaoGame) resumeTalk() {
	now := time.Now()
	g.enter(espiaoTalk)
	g.deadline, g.turnUntil, g.paused = now.Add(g.paused), now.Add(espiaoTurnLimit), 0
}

func (g *espiaoGame) clock() string {
	return fmt.Sprintf("▶️ O relógio voltou a andar: vez de @%s perguntar, e as perguntas acabam em %s ⏳", g.players[g.asker].jid.User, core.Countdown(g.deadline))
}

func (g *espiaoGame) voters() []roomPlayer {
	if g.phase != espiaoAccusing {
		return g.players
	}
	return slices.Delete(slices.Clone(g.players), g.suspect, g.suspect+1)
}

func (g *espiaoGame) callVote(header string) (string, []types.JID, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase == espiaoOver {
		return "", nil, false
	}
	g.deadline = time.Now().Add(espiaoPollTime)
	text, mentions := g.board(header)
	return text, mentions, true
}

func (g *espiaoGame) openPoll(id types.MessageID, options []string) *roomBallot {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.poll, g.pollOptions = id, options
	g.ballot = newRoomBallot(g.voters(), options)
	return g.ballot
}

func (g *espiaoGame) openedPoll() (types.MessageID, *roomBallot) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.poll, g.ballot
}

func (g *espiaoGame) pollDue() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.ballot == nil || g.phase == espiaoOver {
		return false
	}
	select {
	case <-g.ballot.decided:
		return true
	default:
		return !time.Now().Before(g.deadline)
	}
}

func (g *espiaoGame) verdict(votes []int) espiaoEnding {
	g.mu.Lock()
	defer g.mu.Unlock()
	seats := make([]int, len(g.players))
	for i := range seats {
		seats[i] = i
	}
	slices.SortStableFunc(seats, func(a, b int) int { return cmp.Compare(votes[b], votes[a]) })
	lines := []string{"🗳️ *Fim da votação!*", ""}
	for _, seat := range seats {
		lines = append(lines, fmt.Sprintf("• @%s: %s", g.players[seat].jid.User, core.CountLabel(votes[seat], "voto", "votos")))
	}
	outcome, spyWon := g.outcome(votes)
	lines = append(lines, "", outcome)
	return espiaoEnding{header: strings.Join(lines, "\n"), spyWon: spyWon}
}

func (g *espiaoGame) outcome(votes []int) (string, bool) {
	top := slices.Max(votes)
	var leaders []types.JID
	for seat, n := range votes {
		if n == top {
			leaders = append(leaders, g.players[seat].jid)
		}
	}
	if top == 0 {
		return "🦗 Ninguém votou, então o espião ganhou de lavada 😈", true
	}
	if votes[g.spy] == top && len(leaders) == 1 {
		return "🎉 *Pegaram o espião!* Vitória do grupo", false
	}
	if votes[g.spy] == top {
		return "🤝 Deu empate na votação e o espião escapou. *Vitória do espião!* 😈", true
	}
	return fmt.Sprintf("😈 *Vitória do espião!* Vocês votaram em %s", core.JoinMentions(leaders)), true
}

func (g *espiaoGame) guess(text string) (espiaoEnding, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	pick := espiaoPlaceIndex(g.places, text)
	if pick < 0 {
		return espiaoEnding{}, false
	}
	reveal := fmt.Sprintf("🕶️ *@%s se revelou: era o espião!* Chutou que vocês tavam em *%s*", g.players[g.spy].jid.User, g.places[pick])
	if pick == g.place {
		return espiaoEnding{header: reveal + "... e acertou! *Vitória do espião* 😎", spyWon: true}, true
	}
	return espiaoEnding{header: reveal + ", mas errou. *Vitória do grupo* 🎉"}, true
}

func (g *espiaoGame) close(ending espiaoEnding) (espiaoReveal, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase == espiaoOver {
		return espiaoReveal{}, false
	}
	g.halt()
	lines := []string{
		ending.header,
		"",
		fmt.Sprintf("📍 O local era *%s*", g.places[g.place]),
		fmt.Sprintf("🕶️ O espião era @%s", g.players[g.spy].jid.User),
	}
	if g.asked > 0 {
		lines = append(lines, fmt.Sprintf("❓ %s no total", core.CountLabel(g.asked, "pergunta", "perguntas")))
	}
	lines = append(lines, "", "🎭 *Papéis:*")
	for seat, p := range g.players {
		lines = append(lines, fmt.Sprintf("• @%s: %s", p.jid.User, cmp.Or(g.roles[seat], "🕶️ espião")))
	}
	reveal := espiaoReveal{text: strings.Join(lines, "\n"), mentions: roomJIDs(g.players), players: len(g.players)}
	if ending.void {
		return reveal, true
	}
	reveal.winners, reveal.prize = g.winners(ending.spyWon)
	return reveal, true
}

func (g *espiaoGame) winners(spyWon bool) ([]types.JID, int64) {
	players := roomJIDs(g.players)
	if spyWon {
		return []types.JID{players[g.spy]}, espiaoSpyReward
	}
	return slices.Delete(players, g.spy, g.spy+1), espiaoCrewReward
}

func (g *espiaoGame) status(header string) (string, []types.JID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.board(header)
}

func (g *espiaoGame) overview() (string, []types.JID) {
	return g.board(espiaoStages[g.phase].title)
}

func (g *espiaoGame) board(header string) (string, []types.JID) {
	stage := espiaoStages[g.phase]
	lines := []string{header}
	if stage.places {
		lines = append(lines, "", "📍 *Locais possíveis:*")
		lines = append(lines, espiaoPlaceLines(g.places)...)
	}
	lines = append(lines, "", stage.roster)
	for seat, p := range g.players {
		line := fmt.Sprintf("%d. @%s", seat+1, p.jid.User)
		if g.phase == espiaoTalk && seat == g.asker {
			line += " 🎤"
		}
		lines = append(lines, line)
	}
	lines = append(lines, "", stage.hint(g))
	return strings.Join(lines, "\n"), roomJIDs(g.players)
}

func (g *espiaoGame) talkHint() string {
	return fmt.Sprintf("_Vez de @%s perguntar: marca alguém e faz uma pergunta sobre o local. Quem for marcado responde e faz a próxima pergunta pra outra pessoa (não vale devolver pra quem te perguntou). Desconfiou de alguém? Manda !espiao acusar @pessoa, cada um pode acusar uma vez. As perguntas acabam em %s_ ⏳", g.players[g.asker].jid.User, core.Countdown(g.deadline))
}

func espiaoPlaceLines(places []string) []string {
	lines := make([]string, len(places))
	for i, place := range places {
		lines[i] = fmt.Sprintf("%d. %s", i+1, place)
	}
	return lines
}

func espiaoPlaceIndex(places []string, text string) int {
	fields := strings.Fields(text)
	if len(fields) > 0 && core.IsCommand(fields[0]) {
		fields = fields[1:]
	}
	answer := core.FoldLetters(strings.Join(fields, " "))
	if n, err := strconv.Atoi(answer); err == nil && n >= 1 && n <= len(places) {
		return n - 1
	}
	return slices.IndexFunc(places, func(place string) bool {
		_, name, _ := strings.Cut(place, " ")
		return answer == core.FoldLetters(place) || answer == core.FoldLetters(name)
	})
}
