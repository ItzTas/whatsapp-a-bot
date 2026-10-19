package games

import (
	"cmp"
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
)

const (
	impostorMinPlayers  = 3
	impostorMaxPlayers  = 12
	impostorMaxRounds   = 3
	impostorClueRunes   = 40
	impostorTimeout     = 40 * time.Minute
	impostorGroupReward = 40
	impostorSoloReward  = 120
	impostorStarted     = "😈 O jogo já começou, agora vai até o fim"
	impostorGameKind    = "impostor"
)

type impostorPhase int

const (
	impostorLobby impostorPhase = iota
	impostorClues
	impostorDeciding
	impostorVoting
	impostorOver
)

type impostorStage struct {
	title    string
	roster   string
	question string
	duration time.Duration
	hint     func(g *impostorGame) string
}

var impostorStages = map[impostorPhase]impostorStage{
	impostorLobby: {
		title:    "🚪 *Sala do impostor aberta*",
		roster:   fmt.Sprintf("👥 *Na sala* (máximo %d):", impostorMaxPlayers),
		duration: 90 * time.Second,
		hint: func(g *impostorGame) string {
			return fmt.Sprintf("_Manda !impostor ou responde esta mensagem só com ! pra entrar (mínimo de %d jogadores). O jogo começa sozinho em %s, ou antes com !impostor começar_ ⏳", impostorMinPlayers, core.Countdown(g.deadline))
		},
	},
	impostorClues: {
		title:    "🗣️ *Impostor rolando: hora das dicas*",
		roster:   "🗣️ *Ordem das dicas:*",
		duration: 40 * time.Second,
		hint:     (*impostorGame).turnHint,
	},
	impostorDeciding: {
		title:    "🤔 *Impostor rolando: decidindo se já votam*",
		roster:   "📝 *Dicas até agora:*",
		question: "🤔 Já dá pra votar no impostor?",
		duration: 40 * time.Second,
		hint: func(g *impostorGame) string {
			return fmt.Sprintf("_Vota na enquete: já dá pra descobrir quem é o impostor ou bora mais uma rodada de dicas? Só vale voto de quem tá jogando. A enquete fecha em %s, ou antes se a maioria já tiver decidido_ ⏳", core.Countdown(g.deadline))
		},
	},
	impostorVoting: {
		title:    "🗳️ *Impostor rolando: hora de votar*",
		roster:   "📝 *Dicas de cada um:*",
		question: "🗳️ Quem é o impostor?",
		duration: time.Minute,
		hint: func(g *impostorGame) string {
			return fmt.Sprintf("_Vota na enquete em quem você acha que é o impostor (só vale voto de quem tá jogando). O resultado sai em %s, ou antes se a maioria já tiver decidido_ ⏳", core.Countdown(g.deadline))
		},
	},
	impostorOver: {
		title:  "🏁 *Esse impostor já acabou*",
		roster: "👥 *Jogadores:*",
		hint:   func(*impostorGame) string { return "_Manda !impostor pra abrir outra sala_" },
	},
}

type impostorWord struct {
	word  string
	decoy string
}

type impostorRound struct {
	players  []roomPlayer
	impostor int
	word     impostorWord
	votes    []int
}

type impostorGame struct {
	mu          sync.Mutex
	phase       impostorPhase
	players     []roomPlayer
	impostor    int
	word        impostorWord
	round       int
	turn        int
	clues       [][]string
	deadline    time.Time
	hurry       chan struct{}
	opened      time.Time
	notices     []types.MessageID
	poll        types.MessageID
	pollOptions []string
	ballot      *roomBallot
	dropped     bool
}

type impostorState struct {
	Phase    impostorPhase     `json:"phase"`
	Players  []savedRoomPlayer `json:"players"`
	Impostor int               `json:"impostor"`
	Word     string            `json:"word"`
	Decoy    string            `json:"decoy"`
	Round    int               `json:"round"`
	Turn     int               `json:"turn"`
	Clues    [][]string        `json:"clues"`
	Deadline time.Time         `json:"deadline"`
	Hurried  bool              `json:"hurried"`
	Opened   time.Time         `json:"opened"`
	Notices  []types.MessageID `json:"notices"`
	Poll     types.MessageID   `json:"poll"`
	Options  []string          `json:"options"`
	Choices  map[int]int       `json:"choices"`
}

type impostorPollStep func(ctx context.Context, chat types.JID, game *impostorGame, round impostorRound, id types.MessageID, ballot *roomBallot)

type impostorAction func(g *impostorGame, info types.MessageInfo) (string, []types.JID)

func (b *bot) impostor(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return b.SendText(ctx, chat, "O !impostor só funciona em grupo")
	}
	actions := map[string]impostorAction{
		"comecar":  (*impostorGame).start,
		"sair":     (*impostorGame).leave,
		"cancelar": (*impostorGame).cancel,
	}
	action, named := actions[core.FoldLetters(core.FirstPlainArg(core.MessageText(evt.Message)))]
	game, active := b.activeImpostor(ctx, chat)
	if named && !active {
		return b.SendText(ctx, chat, "Não tem impostor rolando agora. Manda !impostor pra abrir uma sala 🕵️")
	}
	if named {
		text, mentions := action(game, evt.Info)
		return b.postImpostor(ctx, chat, game, text, mentions)
	}
	if !active {
		return b.openImpostor(ctx, chat, roomPlayer{jid: evt.Info.Sender.ToNonAD(), ids: b.SenderIdentities(ctx, evt)}, evt.Info)
	}
	return b.joinImpostor(ctx, evt, game)
}

func (b *bot) answerImpostor(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	game, active := b.activeImpostor(ctx, chat)
	if !active {
		return nil
	}
	text := strings.TrimSpace(core.MessageText(evt.Message))
	if text != "!" {
		if game.clue(evt.Info, text) {
			b.syncImpostor(ctx, chat, game)
		}
		return nil
	}
	quoted := evt.Message.GetExtendedTextMessage().GetContextInfo().GetStanzaID()
	if quoted == "" || !game.announced(quoted) {
		return nil
	}
	return b.joinImpostor(ctx, evt, game)
}

func (b *bot) joinImpostor(ctx context.Context, evt *events.Message, game *impostorGame) error {
	player := roomPlayer{jid: evt.Info.Sender.ToNonAD(), ids: b.SenderIdentities(ctx, evt)}
	text, mentions := game.join(evt.Info, player)
	return b.postImpostor(ctx, evt.Info.Chat.ToNonAD(), game, text, mentions)
}

func (b *bot) activeImpostor(ctx context.Context, chat types.JID) (*impostorGame, bool) {
	value, ok := b.impostorGames.Load(chat.String())
	if !ok {
		return nil, false
	}
	game := value.(*impostorGame)
	if !game.expired() {
		return game, true
	}
	b.endImpostor(ctx, chat, game)
	b.impostorGames.CompareAndDelete(chat.String(), game)
	return nil, false
}

func (b *bot) openImpostor(ctx context.Context, chat types.JID, host roomPlayer, info types.MessageInfo) error {
	game := newImpostorGame(host)
	if value, loaded := b.impostorGames.LoadOrStore(chat.String(), game); loaded {
		open := value.(*impostorGame)
		text, mentions := open.join(info, host)
		return b.postImpostor(ctx, chat, open, text, mentions)
	}
	go b.runImpostor(ctx, chat, game)
	text, mentions := game.status(fmt.Sprintf("🕵️ *Quem é o impostor?*\n\n@%s abriu a sala! Todo mundo recebe a mesma palavra secreta no privado, menos um: o impostor, que recebe uma parecida e nem sabe que é ele. Depois rolam rodadas de dicas aqui no grupo, um de cada vez, e quando vocês decidirem na enquete, todo mundo vota em quem acha que é o impostor", host.jid.User))
	return b.postImpostor(ctx, chat, game, text, mentions)
}

func (b *bot) postImpostor(ctx context.Context, chat types.JID, game *impostorGame, text string, mentions []types.JID) error {
	defer b.syncImpostor(ctx, chat, game)
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

func (b *bot) restoreImpostor(ctx context.Context, saved core.SavedGame) error {
	var state impostorState
	if err := json.Unmarshal(saved.State, &state); err != nil {
		return fmt.Errorf("ler o impostor salvo: %w", err)
	}
	chat, err := types.ParseJID(saved.Key)
	if err != nil {
		return fmt.Errorf("ler a conversa do impostor salvo: %w", err)
	}
	game := state.game()
	if _, loaded := b.impostorGames.LoadOrStore(saved.Key, game); loaded {
		return nil
	}
	if game.ballot != nil {
		game.ballot.saved = func() { b.syncImpostor(ctx, chat, game) }
		b.impostorPolls.Store(game.poll, game.ballot)
	}
	go b.resumeImpostor(ctx, chat, game)
	return nil
}

func (s impostorState) game() *impostorGame {
	players := loadRoom(s.Players)
	game := &impostorGame{
		phase:    s.Phase,
		players:  players,
		impostor: s.Impostor,
		word:     impostorWord{word: s.Word, decoy: s.Decoy},
		round:    s.Round,
		turn:     s.Turn,
		clues:    s.Clues,
		deadline: s.Deadline,
		hurry:    make(chan struct{}),
		opened:   s.Opened,
		notices:  s.Notices,
	}
	if s.Hurried {
		game.wake()
	}
	if s.Poll == "" {
		return game
	}
	game.poll, game.pollOptions = s.Poll, s.Options
	game.ballot = newRoomBallot(players, s.Options)
	game.ballot.restore(s.Choices)
	return game
}

func (b *bot) resumeImpostor(ctx context.Context, chat types.JID, game *impostorGame) {
	defer b.Rescue(ctx, chat)
	defer b.releaseImpostor(ctx, chat, game)
	resumes := map[impostorPhase]func(context.Context, types.JID, *impostorGame, impostorRound){
		impostorClues:    b.resumeImpostorClues,
		impostorDeciding: b.resumeImpostorPoll(b.impostorDecision),
		impostorVoting:   b.resumeImpostorPoll(b.impostorVerdict),
	}
	resume, dealt := resumes[game.currentPhase()]
	if !dealt {
		b.playImpostor(ctx, chat, game)
		return
	}
	resume(ctx, chat, game, game.dealt())
}

func (b *bot) resumeImpostorClues(ctx context.Context, chat types.JID, game *impostorGame, round impostorRound) {
	turn, number := game.position()
	if turn < 0 && number > 0 {
		b.impostorRoundOver(ctx, chat, game, round)
		return
	}
	if turn >= 0 && !game.wait(ctx, nil) {
		return
	}
	b.impostorTurns(ctx, chat, game, round)
}

func (b *bot) resumeImpostorPoll(step impostorPollStep) func(context.Context, types.JID, *impostorGame, impostorRound) {
	return func(ctx context.Context, chat types.JID, game *impostorGame, round impostorRound) {
		id, ballot := game.openedPoll()
		if ballot == nil {
			b.impostorRoundOver(ctx, chat, game, round)
			return
		}
		step(ctx, chat, game, round, id, ballot)
	}
}

func (b *bot) runImpostor(ctx context.Context, chat types.JID, game *impostorGame) {
	defer b.Rescue(ctx, chat)
	defer b.releaseImpostor(ctx, chat, game)
	b.playImpostor(ctx, chat, game)
}

func (b *bot) playImpostor(ctx context.Context, chat types.JID, game *impostorGame) {
	if !game.wait(ctx, nil) {
		return
	}
	round, dealt := b.dealImpostor(ctx, chat, game)
	if !dealt {
		return
	}
	b.impostorTurns(ctx, chat, game, round)
}

func (b *bot) releaseImpostor(ctx context.Context, chat types.JID, game *impostorGame) {
	b.syncImpostor(ctx, chat, game)
	b.impostorGames.CompareAndDelete(chat.String(), game)
}

func (b *bot) syncImpostor(ctx context.Context, chat types.JID, game *impostorGame) {
	game.mu.Lock()
	defer game.mu.Unlock()
	b.persistImpostor(ctx, chat, game)
}

func (b *bot) endImpostor(ctx context.Context, chat types.JID, game *impostorGame) {
	game.mu.Lock()
	defer game.mu.Unlock()
	game.halt()
	b.persistImpostor(ctx, chat, game)
}

func (b *bot) persistImpostor(ctx context.Context, chat types.JID, game *impostorGame) {
	if game.phase != impostorOver {
		b.KeepGame(ctx, impostorGameKind, chat.String(), game.snapshot(), game.deadline)
		return
	}
	if game.dropped {
		return
	}
	game.dropped = true
	b.DropGame(ctx, impostorGameKind, chat.String())
}

func (b *bot) dealImpostor(ctx context.Context, chat types.JID, game *impostorGame) (impostorRound, bool) {
	round, dealt := game.deal()
	if !dealt {
		b.tellImpostor(ctx, chat, fmt.Sprintf("😴 Não juntou gente suficiente e o impostor foi cancelado (precisa de pelo menos %d jogadores)", impostorMinPlayers), nil)
		return round, false
	}
	b.syncImpostor(ctx, chat, game)
	announcement, mentions := game.status("🕵️ *Começou o impostor!*\n\n📨 Mandei a palavra secreta no privado de cada um. O impostor recebeu uma diferente e nem sabe que é ele")
	unreached := make([]types.JID, len(round.players))
	var wg sync.WaitGroup
	wg.Go(func() { b.tellImpostor(ctx, chat, announcement, mentions) })
	for i, p := range round.players {
		wg.Go(func() {
			if !b.sendImpostorSecret(ctx, p, round.secret(i)) {
				unreached[i] = p.jid
			}
		})
	}
	wg.Wait()
	unreached = slices.DeleteFunc(unreached, func(jid types.JID) bool { return jid.IsEmpty() })
	if len(unreached) > 0 {
		b.tellImpostor(ctx, chat, fmt.Sprintf("📵 Não consegui mandar a palavra no privado de %s. O jogo continua mesmo assim, então é no blefe 😅", core.JoinMentions(unreached)), unreached)
	}
	return round, true
}

func (b *bot) sendImpostorSecret(ctx context.Context, p roomPlayer, secret string) bool {
	inbox := p.inbox()
	if err := b.SendText(ctx, inbox, secret); err != nil {
		b.Client.Log.Errorf("mandar a palavra do !impostor no privado de %s: %v", inbox, err)
		return false
	}
	return true
}

func (b *bot) impostorTurns(ctx context.Context, chat types.JID, game *impostorGame, round impostorRound) {
	if !b.callImpostorTurns(ctx, chat, game) {
		return
	}
	b.impostorRoundOver(ctx, chat, game, round)
}

func (b *bot) callImpostorTurns(ctx context.Context, chat types.JID, game *impostorGame) bool {
	for {
		text, mentions, more := game.nextTurn()
		b.syncImpostor(ctx, chat, game)
		if !more {
			return true
		}
		b.tellImpostor(ctx, chat, text, mentions)
		if !game.wait(ctx, nil) {
			return false
		}
	}
}

func (b *bot) impostorRoundOver(ctx context.Context, chat types.JID, game *impostorGame, round impostorRound) {
	number, spoke := game.roundOutcome()
	if !spoke {
		b.dropImpostor(ctx, chat, game, round, "😴 Ninguém mandou dica nessa rodada, então o impostor foi cancelado")
		return
	}
	if number == impostorMaxRounds {
		b.voteImpostor(ctx, chat, game, round, fmt.Sprintf("🏁 *Fim da rodada %d, a última!* Agora é votar", number))
		return
	}
	options := []string{"✅ Bora votar", "🔁 Mais uma rodada"}
	id, ballot, opened := b.openImpostorPoll(ctx, chat, game, round, impostorDeciding, fmt.Sprintf("🏁 *Fim da rodada %d!*", number), options)
	if !opened {
		return
	}
	b.impostorDecision(ctx, chat, game, round, id, ballot)
}

func (b *bot) impostorDecision(ctx context.Context, chat types.JID, game *impostorGame, round impostorRound, id types.MessageID, ballot *roomBallot) {
	votes, closed := b.awaitImpostorPoll(ctx, game, id, ballot)
	if !closed {
		return
	}
	if votes[0] > votes[1] {
		b.voteImpostor(ctx, chat, game, round, "✅ *Decidido: bora votar!*")
		return
	}
	b.impostorTurns(ctx, chat, game, round)
}

func (b *bot) voteImpostor(ctx context.Context, chat types.JID, game *impostorGame, round impostorRound, header string) {
	names := b.pollNames(ctx, chat, roomJIDs(round.players))
	id, ballot, opened := b.openImpostorPoll(ctx, chat, game, round, impostorVoting, header, names)
	if !opened {
		return
	}
	b.impostorVerdict(ctx, chat, game, round, id, ballot)
}

func (b *bot) impostorVerdict(ctx context.Context, chat types.JID, game *impostorGame, round impostorRound, id types.MessageID, ballot *roomBallot) {
	votes, closed := b.awaitImpostorPoll(ctx, game, id, ballot)
	if !closed {
		return
	}
	b.endImpostor(ctx, chat, game)
	round.votes = votes
	b.revealImpostor(ctx, chat, round)
}

func (b *bot) openImpostorPoll(ctx context.Context, chat types.JID, game *impostorGame, round impostorRound, phase impostorPhase, header string, options []string) (types.MessageID, *roomBallot, bool) {
	text, mentions := game.advance(phase, header)
	b.tellImpostor(ctx, chat, text, mentions)
	resp, err := b.Client.SendMessage(ctx, chat, b.Client.BuildPollCreation(impostorStages[phase].question, options, 1))
	if err != nil && ctx.Err() != nil {
		return "", nil, false
	}
	if err != nil {
		b.Client.Log.Errorf("abrir enquete do !impostor: %v", err)
		b.dropImpostor(ctx, chat, game, round, "😵 Não consegui abrir a enquete e o impostor foi cancelado")
		return "", nil, false
	}
	ballot := newRoomBallot(round.players, options)
	ballot.saved = func() { b.syncImpostor(ctx, chat, game) }
	game.openPoll(resp.ID, ballot, options)
	b.syncImpostor(ctx, chat, game)
	return resp.ID, ballot, true
}

func (b *bot) awaitImpostorPoll(ctx context.Context, game *impostorGame, id types.MessageID, ballot *roomBallot) ([]int, bool) {
	b.impostorPolls.Store(id, ballot)
	defer b.impostorPolls.Delete(id)
	if !game.wait(ctx, ballot.decided) {
		return nil, false
	}
	return ballot.tally(), true
}

func (b *bot) dropImpostor(ctx context.Context, chat types.JID, game *impostorGame, round impostorRound, reason string) {
	b.endImpostor(ctx, chat, game)
	impostor := round.players[round.impostor].jid
	b.tellImpostor(ctx, chat, fmt.Sprintf("%s. O impostor era @%s, a palavra era *%s* e a dele era *%s*", reason, impostor.User, strings.ToUpper(round.word.word), strings.ToUpper(round.word.decoy)), []types.JID{impostor})
}

func (b *bot) revealImpostor(ctx context.Context, chat types.JID, round impostorRound) {
	verdict, crewWon := round.verdict()
	text := core.AppendLine(round.summary()+"\n\n"+verdict, b.rewardImpostor(ctx, chat, round, crewWon))
	b.tellImpostor(ctx, chat, text, roomJIDs(round.players))
}

func (b *bot) rewardImpostor(ctx context.Context, chat types.JID, round impostorRound, crewWon bool) string {
	players := roomJIDs(round.players)
	if !crewWon {
		return b.economy.Reward(ctx, chat, []types.JID{players[round.impostor]}, len(players), impostorSoloReward)
	}
	crew := slices.Delete(players, round.impostor, round.impostor+1)
	return b.economy.Reward(ctx, chat, crew, len(round.players), impostorGroupReward)
}

func (b *bot) replyImpostor(ctx context.Context, chat types.JID, text string, mentions []types.JID) error {
	if text == "" {
		return nil
	}
	_, err := b.SendMentions(ctx, chat, text, mentions)
	return err
}

func (b *bot) tellImpostor(ctx context.Context, chat types.JID, text string, mentions []types.JID) {
	if err := b.replyImpostor(ctx, chat, text, mentions); err != nil {
		b.Client.Log.Errorf("mandar mensagem do !impostor: %v", err)
	}
}

func newImpostorGame(host roomPlayer) *impostorGame {
	game := &impostorGame{players: []roomPlayer{host}, turn: -1, opened: time.Now()}
	game.enter(impostorLobby)
	return game
}

func (g *impostorGame) enter(phase impostorPhase) {
	g.phase = phase
	g.deadline = time.Now().Add(impostorStages[phase].duration)
	g.hurry = make(chan struct{})
	g.poll, g.pollOptions, g.ballot = "", nil, nil
}

func (g *impostorGame) hurried() bool {
	select {
	case <-g.hurry:
		return true
	default:
		return false
	}
}

func (g *impostorGame) snapshot() impostorState {
	state := impostorState{
		Phase:    g.phase,
		Players:  saveRoom(g.players),
		Impostor: g.impostor,
		Word:     g.word.word,
		Decoy:    g.word.decoy,
		Round:    g.round,
		Turn:     g.turn,
		Clues:    g.clues,
		Deadline: g.deadline,
		Hurried:  g.hurried(),
		Opened:   g.opened,
		Notices:  g.notices,
		Poll:     g.poll,
		Options:  g.pollOptions,
	}
	if g.ballot != nil {
		state.Choices = g.ballot.chosen()
	}
	return state
}

func (g *impostorGame) currentPhase() impostorPhase {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.phase
}

func (g *impostorGame) dealt() impostorRound {
	g.mu.Lock()
	defer g.mu.Unlock()
	return impostorRound{players: slices.Clone(g.players), impostor: g.impostor, word: g.word}
}

func (g *impostorGame) position() (int, int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.turn, g.round
}

func (g *impostorGame) openPoll(id types.MessageID, ballot *roomBallot, options []string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.poll, g.ballot, g.pollOptions = id, ballot, options
}

func (g *impostorGame) openedPoll() (types.MessageID, *roomBallot) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.poll, g.ballot
}

func (g *impostorGame) wake() {
	select {
	case <-g.hurry:
	default:
		close(g.hurry)
	}
}

func (g *impostorGame) halt() {
	g.phase = impostorOver
	g.wake()
}

func (g *impostorGame) wait(ctx context.Context, decided <-chan struct{}) bool {
	g.mu.Lock()
	hurry, deadline := g.hurry, g.deadline
	g.mu.Unlock()
	select {
	case <-ctx.Done():
		return false
	case <-hurry:
	case <-decided:
	case <-time.After(time.Until(deadline)):
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.phase != impostorOver
}

func (g *impostorGame) expired() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.phase == impostorOver || time.Since(g.opened) > impostorTimeout
}

func (g *impostorGame) remember(id types.MessageID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.notices = append(g.notices, id)
}

func (g *impostorGame) announced(id types.MessageID) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return slices.Contains(g.notices, id)
}

func (g *impostorGame) seatOf(info types.MessageInfo) int {
	return roomSeat(g.players, info)
}

func (g *impostorGame) join(info types.MessageInfo, player roomPlayer) (string, []types.JID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase != impostorLobby {
		return g.overview()
	}
	if g.seatOf(info) >= 0 {
		return g.board(fmt.Sprintf("🙋 Você já tá na sala, @%s", player.jid.User))
	}
	if len(g.players) >= impostorMaxPlayers {
		return fmt.Sprintf("🚫 A sala já tá cheia (%d jogadores), espera a próxima rodada", impostorMaxPlayers), nil
	}
	g.players = append(g.players, player)
	header := fmt.Sprintf("✅ @%s entrou na sala!", player.jid.User)
	if len(g.players) < impostorMaxPlayers {
		return g.board(header)
	}
	g.wake()
	return header + "\n\n🚀 A sala lotou, começando o jogo!", []types.JID{player.jid}
}

func (g *impostorGame) start(info types.MessageInfo) (string, []types.JID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase != impostorLobby {
		return g.overview()
	}
	if g.seatOf(info) < 0 {
		return "Só quem tá jogando pode fazer isso 🙄", nil
	}
	if len(g.players) < impostorMinPlayers {
		return g.board(fmt.Sprintf("⏳ Ainda não dá pra começar: precisa de pelo menos %d jogadores", impostorMinPlayers))
	}
	g.wake()
	return "", nil
}

func (g *impostorGame) leave(info types.MessageInfo) (string, []types.JID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase != impostorLobby {
		return impostorStarted, nil
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
	return header + " e ela ficou vazia, então o impostor foi cancelado", []types.JID{gone}
}

func (g *impostorGame) cancel(info types.MessageInfo) (string, []types.JID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase != impostorLobby {
		return impostorStarted, nil
	}
	host := g.players[0].jid
	if g.seatOf(info) != 0 {
		return fmt.Sprintf("Só o primeiro da lista (@%s) pode cancelar a sala 🙄", host.User), []types.JID{host}
	}
	g.halt()
	return fmt.Sprintf("❌ @%s cancelou o impostor", host.User), []types.JID{host}
}

func (g *impostorGame) deal() (impostorRound, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.players) < impostorMinPlayers {
		g.phase = impostorOver
		return impostorRound{}, false
	}
	rand.Shuffle(len(g.players), func(i, j int) { g.players[i], g.players[j] = g.players[j], g.players[i] })
	words := core.PickOne(impostorCategories).words
	picks := rand.Perm(len(words))
	g.word = impostorWord{word: words[picks[0]], decoy: words[picks[1]]}
	g.impostor = rand.IntN(len(g.players))
	g.clues = make([][]string, len(g.players))
	g.enter(impostorClues)
	return impostorRound{players: slices.Clone(g.players), impostor: g.impostor, word: g.word}, true
}

func (g *impostorGame) nextTurn() (string, []types.JID, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.turn < 0 {
		g.round++
	}
	missed, skipped := g.closeTurn()
	g.turn++
	if g.turn == len(g.players) {
		g.turn = -1
		return "", nil, false
	}
	g.enter(impostorClues)
	speaker := g.players[g.turn].jid
	var lines []string
	mentions := []types.JID{speaker}
	if skipped {
		lines = append(lines, fmt.Sprintf("⏰ @%s não mandou a dica a tempo e perdeu a vez", missed.User), "")
		mentions = append(mentions, missed)
	}
	if g.turn == 0 && g.round > 1 {
		lines = append(lines, "🔁 *Não deu maioria pra votar, então bora mais uma rodada de dicas!*", "")
	}
	lines = append(lines,
		fmt.Sprintf("🎤 *Rodada %d de %d* · vez de @%s (%d/%d)", g.round, impostorMaxRounds, speaker.User, g.turn+1, len(g.players)),
		"",
		fmt.Sprintf("_Manda UMA palavra sobre o segredo aqui no grupo, sem entregar demais. Você tem %s_ ⏳", core.Countdown(g.deadline)),
	)
	return strings.Join(lines, "\n"), mentions, true
}

func (g *impostorGame) closeTurn() (types.JID, bool) {
	if g.turn < 0 || g.answered() {
		return types.JID{}, false
	}
	g.clues[g.turn] = append(g.clues[g.turn], "")
	return g.players[g.turn].jid, true
}

func (g *impostorGame) answered() bool {
	return len(g.clues[g.turn]) == g.round
}

func (g *impostorGame) clue(info types.MessageInfo, text string) bool {
	if text == "" || strings.HasPrefix(text, "!") {
		return false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase != impostorClues || g.turn < 0 || g.seatOf(info) != g.turn || g.answered() {
		return false
	}
	g.clues[g.turn] = append(g.clues[g.turn], core.TruncateRunes(strings.Join(strings.Fields(text), " "), impostorClueRunes))
	g.wake()
	return true
}

func (g *impostorGame) roundOutcome() (int, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	spoke := slices.ContainsFunc(g.clues, func(said []string) bool { return said[g.round-1] != "" })
	return g.round, spoke
}

func (g *impostorGame) advance(phase impostorPhase, header string) (string, []types.JID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.enter(phase)
	return g.board(header)
}

func (g *impostorGame) status(header string) (string, []types.JID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.board(header)
}

func (g *impostorGame) overview() (string, []types.JID) {
	return g.board(impostorStages[g.phase].title)
}

func (g *impostorGame) board(header string) (string, []types.JID) {
	stage := impostorStages[g.phase]
	lines := []string{header, "", stage.roster}
	for seat := range g.players {
		lines = append(lines, g.seatLine(seat))
	}
	lines = append(lines, "", stage.hint(g))
	return strings.Join(lines, "\n"), roomJIDs(g.players)
}

func (g *impostorGame) seatLine(seat int) string {
	line := fmt.Sprintf("%d. @%s", seat+1, g.players[seat].jid.User)
	if seat < len(g.clues) && len(g.clues[seat]) > 0 {
		said := make([]string, len(g.clues[seat]))
		for i, clue := range g.clues[seat] {
			said[i] = cmp.Or(clue, "🤐")
		}
		line += ": " + strings.Join(said, " · ")
	}
	if g.phase == impostorClues && seat == g.turn {
		line += " 🎤"
	}
	return line
}

func (g *impostorGame) turnHint() string {
	if g.turn < 0 {
		return fmt.Sprintf("_Quando chegar sua vez eu te marco aqui, aí você manda UMA palavra sobre o segredo, sem entregar demais. No fim de cada rodada uma enquete decide se já dá pra votar no impostor ou se rola mais uma (no máximo %d rodadas)_ 🎤", impostorMaxRounds)
	}
	return fmt.Sprintf("_Rodada %d de %d: agora é a vez de @%s, que tem %s pra mandar a dica_ ⏳", g.round, impostorMaxRounds, g.players[g.turn].jid.User, core.Countdown(g.deadline))
}

func (r impostorRound) secret(seat int) string {
	word := r.word.word
	if seat == r.impostor {
		word = r.word.decoy
	}
	return fmt.Sprintf("🔑 Sua palavra secreta é *%s*\n\n_Um de vocês recebeu uma palavra diferente e nem sabe que é o impostor, pode até ser você. Quando chegar sua vez lá no grupo, manda UMA palavra sobre a sua sem entregar demais e fica de olho em quem parece estar falando de outra coisa_ 👀", strings.ToUpper(word))
}

func (r impostorRound) summary() string {
	seats := make([]int, len(r.players))
	for i := range seats {
		seats[i] = i
	}
	slices.SortStableFunc(seats, func(a, b int) int { return cmp.Compare(r.votes[b], r.votes[a]) })
	lines := []string{"🗳️ *Fim da votação!*", ""}
	for _, seat := range seats {
		lines = append(lines, fmt.Sprintf("• @%s: %s", r.players[seat].jid.User, impostorVoteCount(r.votes[seat])))
	}
	lines = append(lines, "",
		fmt.Sprintf("🕵️ O impostor era @%s", r.players[r.impostor].jid.User),
		fmt.Sprintf("🔑 A palavra era *%s* e a do impostor era *%s*", strings.ToUpper(r.word.word), strings.ToUpper(r.word.decoy)),
	)
	return strings.Join(lines, "\n")
}

func (r impostorRound) verdict() (string, bool) {
	top := slices.Max(r.votes)
	var leaders []types.JID
	for seat, n := range r.votes {
		if n == top {
			leaders = append(leaders, r.players[seat].jid)
		}
	}
	caught := r.votes[r.impostor] == top
	if top == 0 {
		return "🦗 Ninguém votou, então o impostor ganhou de lavada 😈", false
	}
	if caught && len(leaders) == 1 {
		return "🎉 *Pegaram o impostor!* O grupo ganhou", true
	}
	if caught {
		return "🤝 Deu empate na votação e o impostor escapou. *Vitória do impostor!* 😈", false
	}
	return fmt.Sprintf("😈 *Vitória do impostor!* Vocês votaram em %s", core.JoinMentions(leaders)), false
}

func impostorVoteCount(n int) string {
	if n == 1 {
		return "1 voto"
	}
	return fmt.Sprintf("%d votos", n)
}
