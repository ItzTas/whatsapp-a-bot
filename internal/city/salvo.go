package city

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"time"

	"go.mau.fi/whatsmeow/types"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
)

type citySave struct {
	Chat      types.JID          `json:"chat"`
	Host      citySavedPlayer    `json:"host"`
	Phase     cityPhase          `json:"phase"`
	Players   []citySavedPlayer  `json:"players"`
	Round     int                `json:"round"`
	Deadline  time.Time          `json:"deadline"`
	Hurried   bool               `json:"hurried"`
	Opened    time.Time          `json:"opened"`
	Idle      int                `json:"idle"`
	Quiet     bool               `json:"quiet"`
	Kills     map[int]int        `json:"kills"`
	Saved     int                `json:"saved"`
	LastSaved int                `json:"lastSaved"`
	Probe     int                `json:"probe"`
	Votes     map[int]int        `json:"votes"`
	Polls     map[int][]cityPoll `json:"polls"`
}

type citySavedPlayer struct {
	JID  types.JID   `json:"jid"`
	IDs  []types.JID `json:"ids"`
	Role cityRole    `json:"role"`
	Dead bool        `json:"dead"`
}

func (b *bot) syncCity(ctx context.Context, game *cityGame) {
	game.mu.Lock()
	defer game.mu.Unlock()
	if game.phase != cityOver {
		b.KeepGame(ctx, cityKind, game.chat.String(), game.snapshot(), game.opened.Add(cityTimeout))
		return
	}
	if game.dropped {
		return
	}
	game.dropped = true
	b.DropGame(ctx, cityKind, game.chat.String())
}

func (g *cityGame) snapshot() citySave {
	state := citySave{
		Chat:      g.chat,
		Host:      savedCityPlayer(g.host),
		Phase:     g.phase,
		Round:     g.round,
		Deadline:  g.deadline,
		Hurried:   g.hurried(),
		Opened:    g.opened,
		Idle:      g.idle,
		Quiet:     g.quiet,
		Kills:     maps.Clone(g.kills),
		Saved:     g.saved,
		LastSaved: g.lastSaved,
		Probe:     g.probe,
		Votes:     maps.Clone(g.votes),
		Polls:     maps.Clone(g.polls),
	}
	for _, p := range g.players {
		state.Players = append(state.Players, savedCityPlayer(p))
	}
	return state
}

func savedCityPlayer(p cityPlayer) citySavedPlayer {
	return citySavedPlayer{JID: p.jid, IDs: p.ids, Role: p.role, Dead: p.dead}
}

func (s citySavedPlayer) player() cityPlayer {
	return cityPlayer{jid: s.JID, ids: s.IDs, role: s.Role, dead: s.Dead}
}

func (s citySave) game() *cityGame {
	game := &cityGame{
		chat:      s.Chat,
		host:      s.Host.player(),
		phase:     s.Phase,
		round:     s.Round,
		deadline:  s.Deadline,
		hurry:     make(chan struct{}),
		opened:    s.Opened,
		idle:      s.Idle,
		quiet:     s.Quiet,
		kills:     s.Kills,
		saved:     s.Saved,
		lastSaved: s.LastSaved,
		probe:     s.Probe,
		votes:     s.Votes,
		polls:     s.Polls,
	}
	if game.kills == nil {
		game.kills = map[int]int{}
	}
	if game.votes == nil {
		game.votes = map[int]int{}
	}
	if game.polls == nil {
		game.polls = map[int][]cityPoll{}
	}
	for _, p := range s.Players {
		game.players = append(game.players, p.player())
	}
	if s.Hurried {
		game.wake()
	}
	return game
}

func (b *bot) restoreCity(ctx context.Context, saved core.SavedGame) error {
	var state citySave
	if err := json.Unmarshal(saved.State, &state); err != nil {
		return fmt.Errorf("ler o cidadedorme salvo: %w", err)
	}
	game := state.game()
	if _, loaded := b.games.LoadOrStore(saved.Key, game); loaded {
		return nil
	}
	go b.driveCity(ctx, game)
	return nil
}
