package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	mentiraWriteWindow = 3 * time.Minute
	mentiraVoteWindow  = 2 * time.Minute
	mentiraReward      = 40
	mentiraMaxRunes    = 100
	mentiraStatements  = 3
	mentiraGameKind    = "mentira"
)

var mentiraMarker = regexp.MustCompile(`^(?:\d+\s*[.)\-:]|[-•*])\s*`)

type mentiraRound struct {
	*pollAnswer
	author types.JID
	lie    string
	saved  func()
}

type mentiraGame struct {
	chat       types.JID
	author     types.JID
	ids        []types.JID
	statements chan []string

	mu       sync.Mutex
	invited  bool
	deadline time.Time
	told     []string
	poll     types.MessageID
	round    *mentiraRound
	over     bool
}

type mentiraState struct {
	Author   types.JID       `json:"author"`
	IDs      []types.JID     `json:"ids"`
	Invited  bool            `json:"invited"`
	Deadline time.Time       `json:"deadline"`
	Told     []string        `json:"told"`
	Poll     types.MessageID `json:"poll"`
	Lie      string          `json:"lie"`
	Votes    *storedPoll     `json:"votes"`
}

func (b *bot) mentira(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return b.sendText(ctx, chat, "O !2verdades1mentira só funciona em grupo")
	}
	game := &mentiraGame{
		chat:       chat,
		author:     evt.Info.Sender.ToNonAD(),
		ids:        b.senderIdentities(ctx, evt),
		statements: make(chan []string, 1),
		deadline:   time.Now().Add(mentiraWriteWindow),
	}
	if _, running := b.mentiraChats.LoadOrStore(chat.String(), game); running {
		return b.replyTo(ctx, evt, "⏳ Já tem um 2 verdades e 1 mentira rolando aqui, espera ele acabar")
	}
	prompt, free := b.awaitPrivate(game.ids, b.mentiraAnswer(game.statements))
	if !free {
		b.mentiraChats.CompareAndDelete(chat.String(), game)
		return b.replyTo(ctx, evt, "⏳ Você já tá com uma resposta pendente no meu privado. Termina ela antes")
	}
	b.syncMentira(ctx, game)
	go b.writeMentira(ctx, game, prompt)
	return nil
}

func (b *bot) restoreMentira(ctx context.Context, saved savedGame) error {
	var state mentiraState
	if err := json.Unmarshal(saved.state, &state); err != nil {
		return fmt.Errorf("ler o 2 verdades e 1 mentira salvo: %w", err)
	}
	chat, err := types.ParseJID(saved.key)
	if err != nil {
		return fmt.Errorf("ler a conversa do 2 verdades e 1 mentira salvo: %w", err)
	}
	game := state.game(chat)
	if _, loaded := b.mentiraChats.LoadOrStore(saved.key, game); loaded {
		return nil
	}
	if game.round != nil {
		game.round.saved = func() { b.syncMentira(ctx, game) }
		b.mentiraPolls.Store(game.poll, game.round)
		go b.playMentira(ctx, game, b.closeMentira)
		return nil
	}
	if len(game.told) > 0 {
		go b.playMentira(ctx, game, b.pollMentira)
		return nil
	}
	prompt, free := b.awaitPrivate(game.ids, b.mentiraAnswer(game.statements))
	if !free {
		b.mentiraChats.CompareAndDelete(saved.key, game)
		return errors.New("quem ia contar as frases já tem outra resposta pendente no privado")
	}
	go b.writeMentira(ctx, game, prompt)
	return nil
}

func (s mentiraState) game(chat types.JID) *mentiraGame {
	game := &mentiraGame{
		chat:       chat,
		author:     s.Author,
		ids:        s.IDs,
		statements: make(chan []string, 1),
		invited:    s.Invited,
		deadline:   s.Deadline,
		told:       s.Told,
		poll:       s.Poll,
	}
	if s.Poll != "" && s.Votes != nil {
		game.round = &mentiraRound{pollAnswer: restorePollAnswer(*s.Votes), author: s.Author, lie: s.Lie}
	}
	return game
}

func (b *bot) playMentira(ctx context.Context, game *mentiraGame, step func(context.Context, *mentiraGame)) {
	defer b.releaseMentira(ctx, game)
	step(ctx, game)
}

func (b *bot) writeMentira(ctx context.Context, game *mentiraGame, prompt *privatePrompt) {
	defer b.releaseMentira(ctx, game)
	defer b.dropPrivate(prompt)
	if !game.wasInvited() && !b.inviteMentira(ctx, game) {
		return
	}
	var told []string
	select {
	case <-ctx.Done():
		return
	case told = <-game.statements:
	case <-time.After(time.Until(game.until())):
		b.tellMentira(ctx, game.chat, fmt.Sprintf("⏰ @%s não mandou as frases a tempo, então o 2 verdades e 1 mentira foi cancelado", game.author.User), []types.JID{game.author})
		return
	}
	b.dropPrivate(prompt)
	game.hear(told)
	b.syncMentira(ctx, game)
	b.pollMentira(ctx, game)
}

func (b *bot) inviteMentira(ctx context.Context, game *mentiraGame) bool {
	author := game.author
	invite := fmt.Sprintf("🤥 *2 verdades e 1 mentira*%s\n\nMe manda aqui 3 coisas sobre você: *a MENTIRA primeiro* e depois as 2 verdades, uma por linha (ou separadas por |). Eu embaralho e o grupo vota em qual é a mentira 🤫\n\n_Você tem %s_ ⏳",
		b.privateGroupLabel(ctx, game.chat), formatWait(mentiraWriteWindow))
	if !b.sendPrivate(ctx, game.ids, invite, nil) {
		b.tellMentira(ctx, game.chat, fmt.Sprintf("📵 Não consegui te mandar mensagem no privado, @%s, então o 2 verdades e 1 mentira foi cancelado", author.User), []types.JID{author})
		return false
	}
	b.tellMentira(ctx, game.chat, fmt.Sprintf("🤥 *2 verdades e 1 mentira!*\n\n📨 Chamei @%s no privado pra contar 3 coisas sobre si: 2 verdades e 1 mentira. Assim que chegar, abro a enquete aqui pra vocês descobrirem qual é a mentira", author.User), []types.JID{author})
	game.invite()
	b.syncMentira(ctx, game)
	return true
}

func (b *bot) pollMentira(ctx context.Context, game *mentiraGame) {
	chat, author, told := game.chat, game.author, game.statementsTold()
	round := &mentiraRound{pollAnswer: newPollAnswer(told[0]), author: author, lie: told[0]}
	options := slices.Clone(told)
	rand.Shuffle(len(options), func(i, j int) { options[i], options[j] = options[j], options[i] })
	intro := fmt.Sprintf("🤥 *2 verdades e 1 mentira!*\n\n@%s contou 3 coisas sobre si, e uma delas é mentira. Vota na enquete, a resposta sai em %s ⏳", author.User, formatWait(mentiraVoteWindow))
	b.tellMentira(ctx, chat, intro, []types.JID{author})
	question := fmt.Sprintf("🤥 Qual é a mentira de %s?", b.pollNames(ctx, chat, []types.JID{author})[0])
	resp, err := b.client.SendMessage(ctx, chat, b.client.BuildPollCreation(question, options, 1))
	if err != nil {
		b.client.Log.Errorf("abrir enquete do !2verdades1mentira: %v", err)
		b.tellMentira(ctx, chat, "😵 Não consegui abrir a enquete, então o 2 verdades e 1 mentira foi cancelado", nil)
		return
	}
	round.saved = func() { b.syncMentira(ctx, game) }
	game.open(resp.ID, round)
	b.syncMentira(ctx, game)
	b.mentiraPolls.Store(resp.ID, round)
	b.closeMentira(ctx, game)
}

func (b *bot) closeMentira(ctx context.Context, game *mentiraGame) {
	poll, round, deadline := game.ballot()
	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Until(deadline)):
	}
	b.mentiraPolls.Delete(poll)
	b.endMentira(ctx, game)
	text, mentions := b.revealMentira(ctx, game.chat, round)
	b.tellMentira(ctx, game.chat, text, mentions)
}

func (b *bot) releaseMentira(ctx context.Context, game *mentiraGame) {
	if ctx.Err() == nil {
		b.endMentira(ctx, game)
	}
	b.mentiraChats.CompareAndDelete(game.chat.String(), game)
}

func (b *bot) syncMentira(ctx context.Context, game *mentiraGame) {
	game.mu.Lock()
	defer game.mu.Unlock()
	if game.over {
		return
	}
	b.keepGame(ctx, mentiraGameKind, game.chat.String(), game.snapshot(), game.expiresAt())
}

func (b *bot) endMentira(ctx context.Context, game *mentiraGame) {
	game.mu.Lock()
	defer game.mu.Unlock()
	if game.over {
		return
	}
	game.over = true
	b.dropGame(ctx, mentiraGameKind, game.chat.String())
}

func (g *mentiraGame) snapshot() mentiraState {
	state := mentiraState{Author: g.author, IDs: g.ids, Invited: g.invited, Deadline: g.deadline, Told: g.told, Poll: g.poll}
	if g.round != nil {
		votes := g.round.stored()
		state.Lie, state.Votes = g.round.lie, &votes
	}
	return state
}

func (g *mentiraGame) expiresAt() time.Time {
	if len(g.told) > 0 && g.round == nil {
		return time.Now()
	}
	return g.deadline
}

func (g *mentiraGame) wasInvited() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.invited
}

func (g *mentiraGame) invite() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.invited = true
	g.deadline = time.Now().Add(mentiraWriteWindow)
}

func (g *mentiraGame) until() time.Time {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.deadline
}

func (g *mentiraGame) hear(told []string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.told = told
}

func (g *mentiraGame) statementsTold() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.told
}

func (g *mentiraGame) open(poll types.MessageID, round *mentiraRound) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.poll, g.round = poll, round
	g.deadline = time.Now().Add(mentiraVoteWindow)
}

func (g *mentiraGame) ballot() (types.MessageID, *mentiraRound, time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.poll, g.round, g.deadline
}

func (r *mentiraRound) vote(voter types.JID, selected [][]byte) {
	r.pollAnswer.vote(voter, selected)
	r.saved()
}

func (b *bot) revealMentira(ctx context.Context, chat types.JID, round *mentiraRound) (string, []types.JID) {
	voted, winners := round.result()
	authorAliases := b.userAliases(ctx, round.author)
	winners = slices.DeleteFunc(winners, func(w types.JID) bool { return slices.Contains(authorAliases, w.User) })
	header := fmt.Sprintf("🤥 A mentira de @%s era:\n\n%s\n\n", round.author.User, quoteBlock(round.lie))
	mentions := append([]types.JID{round.author}, winners...)
	if !voted {
		return header + "🦗 Ninguém votou", mentions
	}
	if len(winners) == 0 {
		return header + fmt.Sprintf("😈 Ninguém acertou, @%s enganou todo mundo", round.author.User), mentions
	}
	text := header + "✅ Acertaram: " + joinMentions(winners)
	return appendLine(text, b.reward(ctx, chat, winners, round.voters(), mentiraReward)), mentions
}

func (b *bot) tellMentira(ctx context.Context, chat types.JID, text string, mentions []types.JID) {
	if _, err := b.sendMentions(ctx, chat, text, mentions); err != nil {
		b.client.Log.Errorf("mandar mensagem do !2verdades1mentira: %v", err)
	}
}

func (b *bot) mentiraAnswer(statements chan<- []string) privateAnswer {
	return func(ctx context.Context, evt *events.Message, text string) (bool, error) {
		told, problem := parseMentira(text)
		if problem != "" {
			return false, b.sendText(ctx, evt.Info.Chat, problem)
		}
		select {
		case statements <- told:
			return true, b.sendText(ctx, evt.Info.Chat, "✅ Recebi! A enquete já vai abrir lá no grupo 👀")
		default:
			return true, nil
		}
	}
}

func parseMentira(text string) ([]string, string) {
	var told []string
	for _, part := range strings.FieldsFunc(text, func(r rune) bool { return r == '\n' || r == '|' }) {
		statement := strings.TrimSpace(mentiraMarker.ReplaceAllString(strings.TrimSpace(part), ""))
		if statement != "" {
			told = append(told, statement)
		}
	}
	if len(told) != mentiraStatements {
		return nil, fmt.Sprintf("Preciso de exatamente 3 frases, e você mandou %d 🤔 Manda a mentira primeiro e depois as 2 verdades, uma por linha (ou separadas por |)", len(told))
	}
	if slices.ContainsFunc(told, func(s string) bool { return utf8.RuneCountInString(s) > mentiraMaxRunes }) {
		return nil, fmt.Sprintf("Cada frase pode ter no máximo %d letras, senão não cabe na enquete ✂️ Manda de novo mais curtinho", mentiraMaxRunes)
	}
	folded := make([]string, len(told))
	for i, s := range told {
		folded[i] = foldLetters(s)
	}
	slices.Sort(folded)
	if len(slices.Compact(folded)) != len(told) {
		return nil, "Tem frase repetida aí 🤨 Manda 3 frases diferentes"
	}
	return told, ""
}
