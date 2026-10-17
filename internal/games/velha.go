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
	velhaKind    = "velha"
	velhaTimeout = 10 * time.Minute
)

var velhaLines = [][3]int{
	{0, 1, 2}, {3, 4, 5}, {6, 7, 8},
	{0, 3, 6}, {1, 4, 7}, {2, 5, 8},
	{0, 4, 8}, {2, 4, 6},
}

type velhaPlayer struct {
	jid  types.JID
	ids  []types.JID
	mark string
	bot  bool
}

type velhaGame struct {
	mu       sync.Mutex
	players  [2]velhaPlayer
	board    [9]string
	turn     int
	over     bool
	lastPlay time.Time
}

type velhaSnapshot struct {
	Players  [2]velhaPlayerSnapshot `json:"players"`
	Board    [9]string              `json:"board"`
	Turn     int                    `json:"turn"`
	LastPlay time.Time              `json:"last_play"`
}

type velhaPlayerSnapshot struct {
	JID  types.JID   `json:"jid"`
	IDs  []types.JID `json:"ids"`
	Mark string      `json:"mark"`
	Bot  bool        `json:"bot"`
}

func (b *bot) velha(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	args := strings.Fields(core.MessageText(evt.Message))[1:]
	giveUp := len(args) > 0 && strings.ToLower(args[0]) == "desistir"
	game, active := b.activeVelha(ctx, chat)
	if giveUp && !active {
		return b.SendText(ctx, chat, "Não tem jogo da velha rolando. Manda !velha pra começar")
	}
	if giveUp {
		return b.giveUpVelha(ctx, evt, game)
	}
	if active {
		_, err := b.SendMentions(ctx, chat, game.status("⭕ *Já tem um jogo da velha rolando*"), game.mentions())
		return err
	}
	return b.startVelha(ctx, evt)
}

func (b *bot) answerVelha(ctx context.Context, evt *events.Message) error {
	text := strings.TrimSpace(core.MessageText(evt.Message))
	cell, err := strconv.Atoi(text)
	if err != nil || len(text) != 1 || cell < 1 {
		return nil
	}
	chat := evt.Info.Chat.ToNonAD()
	game, active := b.activeVelha(ctx, chat)
	if !active {
		return nil
	}
	seat := game.seatOf(evt.Info)
	if seat < 0 {
		return nil
	}
	text, over := game.play(seat, cell-1)
	if text == "" {
		return nil
	}
	if over {
		b.endVelha(ctx, chat.String(), game)
	}
	b.saveVelha(ctx, chat.String(), game)
	_, err = b.SendMentions(ctx, chat, text, game.mentions())
	return err
}

func (b *bot) activeVelha(ctx context.Context, chat types.JID) (*velhaGame, bool) {
	key := chat.String()
	value, ok := b.velhaGames.Load(key)
	if !ok {
		return nil, false
	}
	game := value.(*velhaGame)
	over, timedOut := game.expire()
	if timedOut {
		b.endVelha(ctx, key, game)
	}
	if over {
		return nil, false
	}
	return game, true
}

func (b *bot) saveVelha(ctx context.Context, key string, game *velhaGame) {
	game.mu.Lock()
	defer game.mu.Unlock()
	if game.over {
		return
	}
	snapshot := velhaSnapshot{Board: game.board, Turn: game.turn, LastPlay: game.lastPlay}
	for i, p := range game.players {
		snapshot.Players[i] = velhaPlayerSnapshot{JID: p.jid, IDs: p.ids, Mark: p.mark, Bot: p.bot}
	}
	b.KeepGame(ctx, velhaKind, key, snapshot, game.lastPlay.Add(velhaTimeout))
}

func (b *bot) endVelha(ctx context.Context, key string, game *velhaGame) {
	b.DropGame(ctx, velhaKind, key)
	b.velhaGames.CompareAndDelete(key, game)
}

func (b *bot) restoreVelha(ctx context.Context, saved core.SavedGame) error {
	var snapshot velhaSnapshot
	if err := json.Unmarshal(saved.State, &snapshot); err != nil {
		return fmt.Errorf("ler jogo da velha salvo: %w", err)
	}
	if time.Now().After(saved.Expires) {
		b.DropGame(ctx, velhaKind, saved.Key)
		return nil
	}
	game := &velhaGame{board: snapshot.Board, turn: snapshot.Turn, lastPlay: snapshot.LastPlay}
	for i, p := range snapshot.Players {
		game.players[i] = velhaPlayer{jid: p.JID, ids: p.IDs, mark: p.Mark, bot: p.Bot}
	}
	b.velhaGames.LoadOrStore(saved.Key, game)
	return nil
}

func (b *bot) startVelha(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	rival, ok := b.velhaRival(ctx, evt)
	if !ok {
		return b.SendText(ctx, chat, "Não dá pra jogar contra você mesmo 🤨 Marca alguém ou manda só !velha pra jogar contra mim")
	}
	challenger := velhaPlayer{jid: evt.Info.Sender.ToNonAD(), ids: b.SenderIdentities(ctx, evt), mark: "❌"}
	game := &velhaGame{players: [2]velhaPlayer{challenger, rival}, lastPlay: time.Now()}
	header := fmt.Sprintf("⭕ *Jogo da velha!*\n\n%s (❌) x %s (⭕)", challenger.tag(), rival.tag())
	if value, loaded := b.velhaGames.LoadOrStore(chat.String(), game); loaded {
		game = value.(*velhaGame)
		header = "⭕ *Já tem um jogo da velha rolando*"
	}
	b.saveVelha(ctx, chat.String(), game)
	_, err := b.SendMentions(ctx, chat, game.status(header), game.mentions())
	return err
}

func (b *bot) velhaRival(ctx context.Context, evt *events.Message) (velhaPlayer, bool) {
	mentions := core.MentionedJIDs(evt.Message)
	if len(mentions) == 0 || b.IsSelf(mentions[0]) {
		return velhaPlayer{mark: "⭕", bot: true}, true
	}
	rival := mentions[0].ToNonAD()
	ids := b.Identities(ctx, rival)
	if slices.ContainsFunc(ids, func(id types.JID) bool { return core.SentBy(evt.Info, id) }) {
		return velhaPlayer{}, false
	}
	return velhaPlayer{jid: rival, ids: ids, mark: "⭕"}, true
}

func (b *bot) giveUpVelha(ctx context.Context, evt *events.Message, game *velhaGame) error {
	chat := evt.Info.Chat.ToNonAD()
	seat := game.seatOf(evt.Info)
	if seat < 0 {
		return b.SendText(ctx, chat, "Só quem tá jogando pode desistir 🙄")
	}
	text, over := game.forfeit(seat)
	if !over {
		return nil
	}
	b.endVelha(ctx, chat.String(), game)
	_, err := b.SendMentions(ctx, chat, text, game.mentions())
	return err
}

func (p velhaPlayer) tag() string {
	if p.bot {
		return "eu 🤖"
	}
	return "@" + p.jid.User
}

func (p velhaPlayer) victory() string {
	if p.bot {
		return "🤖 *Ganhei!* Treina mais e volta"
	}
	return fmt.Sprintf("🏆 *%s venceu!*", p.tag())
}

func (g *velhaGame) mentions() []types.JID {
	var jids []types.JID
	for _, p := range g.players {
		if !p.bot {
			jids = append(jids, p.jid)
		}
	}
	return jids
}

func (g *velhaGame) seatOf(info types.MessageInfo) int {
	return slices.IndexFunc(g.players[:], func(p velhaPlayer) bool {
		return slices.ContainsFunc(p.ids, func(id types.JID) bool { return core.SentBy(info, id) })
	})
}

func (g *velhaGame) expire() (over, timedOut bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.over {
		return true, false
	}
	if time.Since(g.lastPlay) <= velhaTimeout {
		return false, false
	}
	g.over = true
	return true, true
}

func (g *velhaGame) status(header string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return header + "\n\n" + g.grid() + "\n\n" + g.nextTurn()
}

func (g *velhaGame) play(seat, cell int) (string, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.over {
		return "", false
	}
	player := g.players[seat]
	if seat != g.turn {
		return fmt.Sprintf("✋ Calma, %s, agora é a vez de %s", player.tag(), g.players[g.turn].tag()), false
	}
	if g.board[cell] != "" {
		return fmt.Sprintf("🙄 A casa %d já tá ocupada, %s", cell+1, player.tag()), false
	}
	g.lastPlay = time.Now()
	g.place(cell)
	header := fmt.Sprintf("%s jogou na casa %d", player.tag(), cell+1)
	if g.finished() || !g.players[g.turn].bot {
		return g.report(header)
	}
	botCell := g.botCell()
	g.place(botCell)
	return g.report(fmt.Sprintf("%s jogou na casa %d e eu joguei na %d", player.tag(), cell+1, botCell+1))
}

func (g *velhaGame) forfeit(seat int) (string, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.over {
		return "", false
	}
	g.over = true
	return fmt.Sprintf("🏳️ %s desistiu\n\n%s", g.players[seat].tag(), g.players[1-seat].victory()), true
}

func (g *velhaGame) place(cell int) {
	g.board[cell] = g.players[g.turn].mark
	g.turn = 1 - g.turn
}

func (g *velhaGame) report(header string) (string, bool) {
	text := header + "\n\n" + g.grid()
	if winner, won := g.winner(); won {
		g.over = true
		return text + "\n\n" + winner.victory(), true
	}
	if g.full() {
		g.over = true
		return text + "\n\n👵 *Deu velha!* Ninguém ganhou", true
	}
	return text + "\n\n" + g.nextTurn(), false
}

func (g *velhaGame) nextTurn() string {
	player := g.players[g.turn]
	return fmt.Sprintf("👉 Vez de %s (%s): manda o número de uma casa livre", player.tag(), player.mark)
}

func (g *velhaGame) winner() (velhaPlayer, bool) {
	for _, line := range velhaLines {
		mark := g.board[line[0]]
		if mark == "" || mark != g.board[line[1]] || mark != g.board[line[2]] {
			continue
		}
		i := slices.IndexFunc(g.players[:], func(p velhaPlayer) bool { return p.mark == mark })
		return g.players[i], true
	}
	return velhaPlayer{}, false
}

func (g *velhaGame) full() bool {
	return !slices.Contains(g.board[:], "")
}

func (g *velhaGame) finished() bool {
	_, won := g.winner()
	return won || g.full()
}

func (g *velhaGame) botCell() int {
	own, rival := g.players[g.turn].mark, g.players[1-g.turn].mark
	choices := [][]int{g.completing(own), g.completing(rival), {4}, {0, 2, 6, 8}, {1, 3, 5, 7}}
	for _, cells := range choices {
		free := slices.DeleteFunc(slices.Clone(cells), func(c int) bool { return g.board[c] != "" })
		if len(free) > 0 {
			return free[rand.IntN(len(free))]
		}
	}
	return slices.Index(g.board[:], "")
}

func (g *velhaGame) completing(mark string) []int {
	var cells []int
	for _, line := range velhaLines {
		marked, empty := 0, -1
		for _, c := range line {
			switch g.board[c] {
			case mark:
				marked++
			case "":
				empty = c
			}
		}
		if marked == 2 && empty >= 0 {
			cells = append(cells, empty)
		}
	}
	return cells
}

func (g *velhaGame) grid() string {
	digits := []string{"1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣"}
	rows := make([]string, 3)
	for r := range rows {
		cells := make([]string, 3)
		for c := range cells {
			i := r*3 + c
			cells[c] = cmp.Or(g.board[i], digits[i])
		}
		rows[r] = strings.Join(cells, " ")
	}
	return strings.Join(rows, "\n")
}
