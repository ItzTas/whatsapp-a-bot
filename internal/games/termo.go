package games

import (
	"cmp"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	termoKind         = "termo"
	termoLetters      = 5
	termoTries        = 6
	termoTimeout      = 15 * time.Minute
	termoRewardPerTry = 15
)

//go:embed palavras/termo.txt
var termoFile string

var (
	termoList  = strings.Fields(termoFile)
	termoWords = termoIndex(termoList)
)

type termoGuess struct {
	word  string
	shown string
	marks string
}

type termoGame struct {
	mu       sync.Mutex
	answer   string
	folded   string
	guesses  []termoGuess
	players  []string
	over     bool
	won      bool
	lastPlay time.Time
}

type termoSnapshot struct {
	Answer   string               `json:"answer"`
	Guesses  []termoGuessSnapshot `json:"guesses"`
	Players  []string             `json:"players"`
	LastPlay time.Time            `json:"last_play"`
}

type termoGuessSnapshot struct {
	Word  string `json:"word"`
	Shown string `json:"shown"`
	Marks string `json:"marks"`
}

func (b *bot) termo(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	guess := strings.Join(strings.Fields(core.MessageText(evt.Message))[1:], " ")
	game, active := b.activeTermo(ctx, chat)
	if guess == "" && active {
		return b.SendText(ctx, chat, game.status("🟩 *Termo rolando*"))
	}
	if guess == "" {
		return b.startTermo(ctx, chat)
	}
	if !active {
		return b.SendText(ctx, chat, "Não tem termo rolando. Manda !termo pra começar")
	}
	if !validTermoGuess(core.FoldLetters(guess)) {
		return b.ReplyTo(ctx, evt, fmt.Sprintf("🙄 Chuta uma palavra de %d letras, sem número nem símbolo", termoLetters))
	}
	return b.playTermo(ctx, evt, game, guess)
}

func (b *bot) answerTermo(ctx context.Context, evt *events.Message) error {
	text := strings.TrimSpace(core.MessageText(evt.Message))
	if _, known := termoWords[core.FoldLetters(text)]; !known {
		return nil
	}
	game, active := b.activeTermo(ctx, evt.Info.Chat.ToNonAD())
	if !active {
		return nil
	}
	return b.playTermo(ctx, evt, game, text)
}

func (b *bot) activeTermo(ctx context.Context, chat types.JID) (*termoGame, bool) {
	key := chat.String()
	value, ok := b.termoGames.Load(key)
	if !ok {
		return nil, false
	}
	game := value.(*termoGame)
	over, timedOut := game.expire()
	if timedOut {
		b.endTermo(ctx, key, game)
	}
	if over {
		return nil, false
	}
	return game, true
}

func (b *bot) startTermo(ctx context.Context, chat types.JID) error {
	answer := core.PickOne(termoList)
	game := &termoGame{answer: answer, folded: core.FoldLetters(answer), lastPlay: time.Now()}
	header := fmt.Sprintf("🟩 *Termo!* Descubram a palavra de %d letras em %d chances\n\n"+
		"🟩 a letra tá no lugar certo\n🟨 a palavra tem a letra, mas em outro lugar\n⬛ a palavra não tem a letra\n\n"+
		"_Manda uma palavra de %d letras pra chutar, ou !termo <palavra>_", termoLetters, termoTries, termoLetters)
	if value, loaded := b.termoGames.LoadOrStore(chat.String(), game); loaded {
		game = value.(*termoGame)
		header = "🟩 *Termo rolando*"
	}
	b.saveTermo(ctx, chat.String(), game)
	return b.SendText(ctx, chat, game.status(header))
}

func (b *bot) playTermo(ctx context.Context, evt *events.Message, game *termoGame, guess string) error {
	chat := evt.Info.Chat.ToNonAD()
	player := evt.Info.Sender.ToNonAD()
	text, over := game.play(guess, player)
	if text == "" {
		return nil
	}
	if over {
		b.endTermo(ctx, chat.String(), game)
		text = core.AppendLine(text, b.rewardTermo(ctx, chat, game, player))
	}
	b.saveTermo(ctx, chat.String(), game)
	_, err := b.SendMentions(ctx, chat, text, []types.JID{player})
	return err
}

func (b *bot) saveTermo(ctx context.Context, key string, game *termoGame) {
	game.mu.Lock()
	defer game.mu.Unlock()
	if game.over {
		return
	}
	snapshot := termoSnapshot{Answer: game.answer, Players: game.players, LastPlay: game.lastPlay}
	for _, guess := range game.guesses {
		snapshot.Guesses = append(snapshot.Guesses, termoGuessSnapshot{Word: guess.word, Shown: guess.shown, Marks: guess.marks})
	}
	b.KeepGame(ctx, termoKind, key, snapshot, game.lastPlay.Add(termoTimeout))
}

func (b *bot) endTermo(ctx context.Context, key string, game *termoGame) {
	b.DropGame(ctx, termoKind, key)
	b.termoGames.CompareAndDelete(key, game)
}

func (b *bot) restoreTermo(ctx context.Context, saved core.SavedGame) error {
	var snapshot termoSnapshot
	if err := json.Unmarshal(saved.State, &snapshot); err != nil {
		return fmt.Errorf("ler termo salvo: %w", err)
	}
	if time.Now().After(saved.Expires) {
		b.DropGame(ctx, termoKind, saved.Key)
		return nil
	}
	game := &termoGame{
		answer:   snapshot.Answer,
		folded:   core.FoldLetters(snapshot.Answer),
		players:  snapshot.Players,
		lastPlay: snapshot.LastPlay,
	}
	for _, guess := range snapshot.Guesses {
		game.guesses = append(game.guesses, termoGuess{word: guess.Word, shown: guess.Shown, marks: guess.Marks})
	}
	b.termoGames.LoadOrStore(saved.Key, game)
	return nil
}

func (b *bot) rewardTermo(ctx context.Context, chat types.JID, game *termoGame, winner types.JID) string {
	amount, players := game.prize()
	if amount == 0 {
		return ""
	}
	return b.economy.Reward(ctx, chat, []types.JID{winner}, players, amount)
}

func termoIndex(words []string) map[string]string {
	index := make(map[string]string, len(words))
	for _, word := range words {
		index[core.FoldLetters(word)] = word
	}
	return index
}

func validTermoGuess(folded string) bool {
	return len(folded) == termoLetters && !strings.ContainsFunc(folded, func(r rune) bool { return r < 'a' || r > 'z' })
}

func termoMarks(answer, guess string) string {
	marks := make([]string, len(guess))
	left := map[byte]int{}
	for i := range guess {
		if guess[i] == answer[i] {
			marks[i] = "🟩"
			continue
		}
		left[answer[i]]++
	}
	for i := range guess {
		if marks[i] != "" {
			continue
		}
		marks[i] = "⬛"
		if left[guess[i]] > 0 {
			left[guess[i]]--
			marks[i] = "🟨"
		}
	}
	return strings.Join(marks, "")
}

func (g *termoGame) expire() (over, timedOut bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.over {
		return true, false
	}
	if time.Since(g.lastPlay) <= termoTimeout {
		return false, false
	}
	g.over = true
	return true, true
}

func (g *termoGame) play(guess string, player types.JID) (string, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.over {
		return "", false
	}
	word := core.FoldLetters(guess)
	shown := strings.ToUpper(cmp.Or(termoWords[word], guess))
	if slices.ContainsFunc(g.guesses, func(t termoGuess) bool { return t.word == word }) {
		return fmt.Sprintf("🙄 *%s* já foi, @%s", shown, player.User), false
	}
	g.lastPlay = time.Now()
	if !slices.Contains(g.players, player.User) {
		g.players = append(g.players, player.User)
	}
	g.guesses = append(g.guesses, termoGuess{word: word, shown: shown, marks: termoMarks(g.folded, word)})
	if word == g.folded {
		return g.win(player), true
	}
	header := fmt.Sprintf("🔤 @%s chutou *%s*", player.User, shown)
	if len(g.guesses) < termoTries {
		return g.board(header), false
	}
	g.over = true
	return g.final(header, fmt.Sprintf("💀 Acabaram as chances! A palavra era *%s*", strings.ToUpper(g.answer))), true
}

func (g *termoGame) win(player types.JID) string {
	g.over = true
	g.won = true
	header := fmt.Sprintf("🎉 @%s acertou! A palavra era *%s*", player.User, strings.ToUpper(g.answer))
	return g.final(header, fmt.Sprintf("🎯 Acertou na %dª de %d tentativas", len(g.guesses), termoTries))
}

func (g *termoGame) prize() (int64, int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.won {
		return 0, 0
	}
	return termoRewardPerTry * int64(termoTries-len(g.guesses)+1), len(g.players)
}

func (g *termoGame) status(header string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.board(header)
}

func (g *termoGame) rows() []string {
	rows := make([]string, len(g.guesses))
	for i, guess := range g.guesses {
		rows[i] = guess.marks + "  *" + guess.shown + "*"
	}
	return rows
}

func (g *termoGame) board(header string) string {
	lines := append([]string{header, ""}, g.rows()...)
	for range termoTries - len(g.guesses) {
		lines = append(lines, strings.Repeat("⬜", termoLetters))
	}
	lines = append(lines, "")
	if out := g.ruledOut(); len(out) > 0 {
		lines = append(lines, "🚫 Fora: "+strings.Join(out, " "))
	}
	lines = append(lines, fmt.Sprintf("🎯 Chances: %d/%d", termoTries-len(g.guesses), termoTries))
	return strings.Join(lines, "\n")
}

func (g *termoGame) final(header, footer string) string {
	return header + "\n\n" + strings.Join(g.rows(), "\n") + "\n\n" + footer
}

func (g *termoGame) ruledOut() []string {
	var out []string
	for _, guess := range g.guesses {
		for _, r := range guess.word {
			letter := strings.ToUpper(string(r))
			if strings.ContainsRune(g.folded, r) || slices.Contains(out, letter) {
				continue
			}
			out = append(out, letter)
		}
	}
	slices.Sort(out)
	return out
}
