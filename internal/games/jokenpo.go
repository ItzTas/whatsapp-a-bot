package games

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
)

const (
	jokenpoWindow      = 2 * time.Minute
	jokenpoGameKind    = "jokenpo"
	jokenpoUnknownMove = "Não entendi 🤔 Manda só pedra, papel ou tesoura (ou ✊ ✋ ✌️)"
)

var jokenpoMoves = [...]string{"✊ pedra", "✋ papel", "✌️ tesoura"}

type jokenpoPlayer struct {
	jid types.JID
	ids []types.JID
	acc core.Account
}

type jokenpoMatch struct {
	key     string
	chat    types.JID
	players [2]jokenpoPlayer
	ready   chan struct{}

	mu       sync.Mutex
	moves    [2]int
	over     bool
	invited  bool
	deadline time.Time
}

type jokenpoState struct {
	Chat     types.JID             `json:"chat"`
	Players  [2]jokenpoSavedPlayer `json:"players"`
	Moves    [2]int                `json:"moves"`
	Invited  bool                  `json:"invited"`
	Deadline time.Time             `json:"deadline"`
}

type jokenpoSavedPlayer struct {
	JID     types.JID          `json:"jid"`
	IDs     []types.JID        `json:"ids"`
	Account core.StoredAccount `json:"account"`
}

type jokenpoResult struct {
	moves [2]int
}

type jokenpoReply int

const (
	jokenpoLate jokenpoReply = iota
	jokenpoRepeated
	jokenpoPlayed
)

func (b *bot) jokenpo(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	mentions := core.MentionedJIDs(evt.Message)
	if len(mentions) > 0 && !b.IsSelf(mentions[0]) {
		return b.challengeJokenpo(ctx, evt, mentions[0].ToNonAD())
	}
	move, ok := jokenpoMoveOf(core.FirstPlainArg(core.MessageText(evt.Message)))
	if !ok {
		return b.SendText(ctx, chat, "✊✋✌️ Manda !jokenpo pedra, papel ou tesoura pra jogar contra mim, ou !jokenpo @pessoa pra desafiar alguém")
	}
	return b.jokenpoAgainstBot(ctx, chat, evt.Info.Sender.ToNonAD(), move)
}

func (b *bot) jokenpoAgainstBot(ctx context.Context, chat, player types.JID, move int) error {
	mine := rand.IntN(len(jokenpoMoves))
	verdicts := [...]string{"🤝 Empate, @%s! Bora de novo?", "🎉 @%s ganhou! Dessa vez passa 😤", "😎 Ganhei, @%s! Tenta de novo"}
	text := fmt.Sprintf("✊✋✌️ *Jokenpô!*\n\n@%s: %s\n🤖 Eu: %s\n\n", player.User, jokenpoMoves[move], jokenpoMoves[mine]) +
		fmt.Sprintf(verdicts[jokenpoDuel(move, mine)], player.User)
	_, err := b.SendMentions(ctx, chat, text, []types.JID{player})
	return err
}

func (b *bot) challengeJokenpo(ctx context.Context, evt *events.Message, rival types.JID) error {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return b.SendText(ctx, chat, "O desafio do !jokenpo só funciona em grupo. Pra jogar contra mim, manda !jokenpo pedra")
	}
	rivalIDs := b.Identities(ctx, rival)
	if slices.ContainsFunc(rivalIDs, func(id types.JID) bool { return core.SentBy(evt.Info, id) }) {
		return b.SendText(ctx, chat, "Jokenpô contra você mesmo? 🤨 Marca outra pessoa ou manda !jokenpo pedra pra jogar contra mim")
	}
	challenger := evt.Info.Sender.ToNonAD()
	players := [2]jokenpoPlayer{
		{jid: challenger, ids: b.SenderIdentities(ctx, evt), acc: b.Account(ctx, challenger)},
		{jid: rival, ids: rivalIDs, acc: b.Account(ctx, rival)},
	}
	match := newJokenpoMatch(evt.Info.ID, chat, players)
	prompts, free := b.awaitJokenpo(match)
	if !free {
		_, err := b.SendMentions(ctx, chat, "⏳ Um de vocês já tá com uma resposta pendente no meu privado. Termina ela antes de começar outro jokenpô", []types.JID{challenger})
		return err
	}
	b.syncJokenpo(ctx, match)
	go b.runJokenpo(ctx, match, prompts)
	return nil
}

func (b *bot) restoreJokenpo(ctx context.Context, saved core.SavedGame) error {
	var state jokenpoState
	if err := json.Unmarshal(saved.State, &state); err != nil {
		return fmt.Errorf("ler o jokenpô salvo: %w", err)
	}
	match := state.match(saved.Key)
	go b.runJokenpo(ctx, match, b.reclaimJokenpo(match))
	return nil
}

func (s jokenpoState) match(key string) *jokenpoMatch {
	var players [2]jokenpoPlayer
	for seat, p := range s.Players {
		players[seat] = jokenpoPlayer{jid: p.JID, ids: p.IDs, acc: p.Account.Account()}
	}
	match := &jokenpoMatch{
		key:      key,
		chat:     s.Chat,
		players:  players,
		ready:    make(chan struct{}),
		moves:    s.Moves,
		invited:  s.Invited,
		deadline: s.Deadline,
	}
	if s.Moves[0] >= 0 && s.Moves[1] >= 0 {
		match.signal()
	}
	return match
}

func (b *bot) reclaimJokenpo(match *jokenpoMatch) [2]*core.PrivatePrompt {
	var prompts [2]*core.PrivatePrompt
	for seat, p := range match.players {
		if match.moves[seat] >= 0 {
			continue
		}
		prompt, free := b.AwaitPrivate(p.ids, b.jokenpoAnswer(match, seat))
		if !free {
			b.Client.Log.Warnf("retomar !jokenpo %s: @%s já tem outra resposta pendente no privado", match.key, p.jid.User)
			continue
		}
		prompts[seat] = prompt
	}
	return prompts
}

func (b *bot) syncJokenpo(ctx context.Context, match *jokenpoMatch) {
	match.mu.Lock()
	defer match.mu.Unlock()
	b.keepJokenpo(ctx, match)
}

func (b *bot) keepJokenpo(ctx context.Context, match *jokenpoMatch) {
	if match.over {
		return
	}
	b.KeepGame(ctx, jokenpoGameKind, match.key, match.snapshot(), match.deadline)
}

func (b *bot) endJokenpo(ctx context.Context, match *jokenpoMatch) jokenpoResult {
	match.mu.Lock()
	defer match.mu.Unlock()
	if !match.over {
		b.DropGame(ctx, jokenpoGameKind, match.key)
	}
	match.over = true
	return match.result()
}

func (b *bot) markJokenpoInvited(ctx context.Context, match *jokenpoMatch) {
	match.mu.Lock()
	defer match.mu.Unlock()
	match.invited = true
	match.deadline = time.Now().Add(jokenpoWindow)
	b.keepJokenpo(ctx, match)
}

func (b *bot) awaitJokenpo(match *jokenpoMatch) ([2]*core.PrivatePrompt, bool) {
	var prompts [2]*core.PrivatePrompt
	for seat, p := range match.players {
		prompt, free := b.AwaitPrivate(p.ids, b.jokenpoAnswer(match, seat))
		if !free {
			b.dropJokenpo(prompts)
			return prompts, false
		}
		prompts[seat] = prompt
	}
	return prompts, true
}

func (b *bot) dropJokenpo(prompts [2]*core.PrivatePrompt) {
	for _, prompt := range prompts {
		if prompt != nil {
			b.DropPrivate(prompt)
		}
	}
}

func (b *bot) runJokenpo(ctx context.Context, match *jokenpoMatch, prompts [2]*core.PrivatePrompt) {
	defer b.Rescue(ctx, match.chat)
	defer b.dropJokenpo(prompts)
	if !match.wasInvited() && !b.startJokenpo(ctx, match) {
		return
	}
	select {
	case <-ctx.Done():
		return
	case <-match.ready:
	case <-time.After(time.Until(match.until())):
	}
	result := b.endJokenpo(ctx, match)
	winner := result.winner()
	b.tellJokenpo(ctx, match, match.board(result)+"\n\n"+match.outcome(result, winner))
}

func (b *bot) startJokenpo(ctx context.Context, match *jokenpoMatch) bool {
	unreached := b.inviteJokenpo(ctx, match)
	if ctx.Err() != nil {
		return false
	}
	if len(unreached) > 0 {
		text := fmt.Sprintf("📵 Não consegui mandar mensagem no privado de %s, então o jokenpô foi cancelado", core.JoinMentions(unreached))
		b.endJokenpo(ctx, match)
		b.tellJokenpo(ctx, match, text)
		return false
	}
	b.tellJokenpo(ctx, match, match.announcement())
	b.markJokenpoInvited(ctx, match)
	return true
}

func (b *bot) inviteJokenpo(ctx context.Context, match *jokenpoMatch) []types.JID {
	texts := match.invitations(b.PrivateGroupLabel(ctx, match.chat))
	unreached := make([]types.JID, len(match.players))
	var wg sync.WaitGroup
	for seat, p := range match.players {
		wg.Go(func() {
			if !b.SendPrivate(ctx, p.ids, texts[seat], []types.JID{match.players[1-seat].jid}) {
				unreached[seat] = p.jid
			}
		})
	}
	wg.Wait()
	return slices.DeleteFunc(unreached, func(jid types.JID) bool { return jid.IsEmpty() })
}

func (b *bot) tellJokenpo(ctx context.Context, match *jokenpoMatch, text string) {
	if _, err := b.SendMentions(ctx, match.chat, text, match.jids()); err != nil {
		b.Client.Log.Errorf("mandar mensagem do !jokenpo: %v", err)
	}
}

func (b *bot) jokenpoAnswer(match *jokenpoMatch, seat int) core.PrivateAnswer {
	return func(ctx context.Context, evt *events.Message, text string) (bool, error) {
		move, ok := jokenpoMoveOf(text)
		if !ok {
			return false, b.SendText(ctx, evt.Info.Chat, jokenpoUnknownMove)
		}
		reply, err := b.recordJokenpo(ctx, match, seat, move)
		if err != nil {
			return false, err
		}
		replies := map[jokenpoReply]string{
			jokenpoLate:     "⏰ Esse jokenpô já acabou",
			jokenpoRepeated: "Você já jogou, agora é esperar o resultado lá no grupo 😉",
			jokenpoPlayed:   fmt.Sprintf("✅ Anotado: %s. O resultado sai lá no grupo 👀", jokenpoMoves[move]),
		}
		return true, b.SendText(ctx, evt.Info.Chat, replies[reply])
	}
}

func (b *bot) recordJokenpo(ctx context.Context, match *jokenpoMatch, seat, move int) (jokenpoReply, error) {
	match.mu.Lock()
	defer match.mu.Unlock()
	if match.over {
		return jokenpoLate, nil
	}
	if match.moves[seat] >= 0 {
		return jokenpoRepeated, nil
	}
	match.moves[seat] = move
	b.keepJokenpo(ctx, match)
	if match.moves[1-seat] >= 0 {
		match.signal()
	}
	return jokenpoPlayed, nil
}

func newJokenpoMatch(key string, chat types.JID, players [2]jokenpoPlayer) *jokenpoMatch {
	return &jokenpoMatch{
		key:      key,
		chat:     chat,
		players:  players,
		ready:    make(chan struct{}),
		moves:    [2]int{-1, -1},
		deadline: time.Now().Add(jokenpoWindow),
	}
}

func (m *jokenpoMatch) snapshot() jokenpoState {
	state := jokenpoState{
		Chat:     m.chat,
		Moves:    m.moves,
		Invited:  m.invited,
		Deadline: m.deadline,
	}
	for seat, p := range m.players {
		state.Players[seat] = jokenpoSavedPlayer{JID: p.jid, IDs: p.ids, Account: core.StoreAccount(p.acc)}
	}
	return state
}

func (m *jokenpoMatch) wasInvited() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.invited
}

func (m *jokenpoMatch) until() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.deadline
}

func (m *jokenpoMatch) result() jokenpoResult {
	return jokenpoResult{moves: m.moves}
}

func (m *jokenpoMatch) signal() {
	select {
	case <-m.ready:
	default:
		close(m.ready)
	}
}

func (m *jokenpoMatch) jids() []types.JID {
	return []types.JID{m.players[0].jid, m.players[1].jid}
}

func (m *jokenpoMatch) invitations(group string) [2]string {
	wait := fmt.Sprintf("\n\n_Você tem %s_ ⏳", core.FormatWait(jokenpoWindow))
	howTo := "Manda aqui a sua jogada: pedra, papel ou tesoura (ou ✊ ✋ ✌️). Ninguém fica sabendo até o resultado sair no grupo 🤫"
	texts := [2]string{
		fmt.Sprintf("✊✋✌️ *Jokenpô*%s\n\nVocê desafiou @%s! %s", group, m.players[1].jid.User, howTo),
		fmt.Sprintf("✊✋✌️ *Jokenpô*%s\n\n@%s te desafiou! %s", group, m.players[0].jid.User, howTo),
	}
	return [2]string{texts[0] + wait, texts[1] + wait}
}

func (m *jokenpoMatch) announcement() string {
	return fmt.Sprintf("✊✋✌️ *Jokenpô!* @%s desafiou @%s\n\n📨 Mandei mensagem no privado de vocês dois: respondam lá com pedra, papel ou tesoura. O resultado sai aqui quando os dois jogarem, ou em %s ⏳",
		m.players[0].jid.User, m.players[1].jid.User, core.FormatWait(jokenpoWindow))
}

func (m *jokenpoMatch) board(r jokenpoResult) string {
	lines := []string{fmt.Sprintf("✊✋✌️ *Jokenpô: @%s x @%s*", m.players[0].jid.User, m.players[1].jid.User), ""}
	for seat, p := range m.players {
		played := "🤐 não jogou"
		if r.moves[seat] >= 0 {
			played = jokenpoMoves[r.moves[seat]]
		}
		lines = append(lines, fmt.Sprintf("@%s: %s", p.jid.User, played))
	}
	return strings.Join(lines, "\n")
}

func (m *jokenpoMatch) outcome(r jokenpoResult, winner int) string {
	tags := [2]string{"@" + m.players[0].jid.User, "@" + m.players[1].jid.User}
	if winner < 0 && r.moves[0] < 0 {
		return "⏰ Ninguém jogou a tempo e o jokenpô foi cancelado"
	}
	if winner < 0 {
		return "🤝 Deu empate!"
	}
	if r.moves[1-winner] < 0 {
		return fmt.Sprintf("⏰ %s não jogou a tempo, então %s ganhou por W.O.", tags[1-winner], tags[winner])
	}
	return fmt.Sprintf("🎉 %s ganhou!", tags[winner])
}

func (r jokenpoResult) winner() int {
	if r.moves[0] < 0 && r.moves[1] < 0 {
		return -1
	}
	if r.moves[0] < 0 {
		return 1
	}
	if r.moves[1] < 0 {
		return 0
	}
	return [...]int{-1, 0, 1}[jokenpoDuel(r.moves[0], r.moves[1])]
}

func jokenpoDuel(a, b int) int {
	return (a - b + len(jokenpoMoves)) % len(jokenpoMoves)
}

func jokenpoMoveOf(text string) (int, bool) {
	cleaned := strings.Map(func(r rune) rune {
		if r == '\uFE0F' || (r >= 0x1F3FB && r <= 0x1F3FF) {
			return -1
		}
		return r
	}, core.FoldLetters(strings.Trim(strings.TrimSpace(text), ".!?")))
	moves := map[string]int{
		"pedra": 0, "pe": 0, "pd": 0, "rock": 0, "✊": 0, "👊": 0, "🤜": 0, "🤛": 0, "🪨": 0,
		"papel": 1, "pa": 1, "pp": 1, "paper": 1, "✋": 1, "🖐": 1, "🤚": 1, "📄": 1,
		"tesoura": 2, "te": 2, "ts": 2, "t": 2, "scissors": 2, "✌": 2, "✂": 2,
	}
	move, ok := moves[cleaned]
	return move, ok
}
