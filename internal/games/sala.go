package games

import (
	"bytes"
	"crypto/sha256"
	"maps"
	"slices"
	"sync"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

type roomPlayer struct {
	jid types.JID
	ids []types.JID
}

type savedRoomPlayer struct {
	JID types.JID   `json:"jid"`
	IDs []types.JID `json:"ids"`
}

type roomBallot struct {
	mu      sync.Mutex
	players []roomPlayer
	options [][sha256.Size]byte
	choices map[int]int
	decided chan struct{}
	saved   func()
}

func saveRoom(players []roomPlayer) []savedRoomPlayer {
	saved := make([]savedRoomPlayer, len(players))
	for i, p := range players {
		saved[i] = savedRoomPlayer{JID: p.jid, IDs: p.ids}
	}
	return saved
}

func loadRoom(saved []savedRoomPlayer) []roomPlayer {
	players := make([]roomPlayer, len(saved))
	for i, p := range saved {
		players[i] = roomPlayer{jid: p.JID, ids: p.IDs}
	}
	return players
}

func roomJIDs(players []roomPlayer) []types.JID {
	jids := make([]types.JID, len(players))
	for i, p := range players {
		jids[i] = p.jid
	}
	return jids
}

func roomSeat(players []roomPlayer, info types.MessageInfo) int {
	return slices.IndexFunc(players, func(p roomPlayer) bool {
		return slices.ContainsFunc(p.ids, func(id types.JID) bool { return core.SentBy(info, id) })
	})
}

func memberSeat(players []roomPlayer, person types.JID) int {
	return slices.IndexFunc(players, func(p roomPlayer) bool {
		return slices.ContainsFunc(p.ids, func(id types.JID) bool { return id.User == person.User })
	})
}

func (p roomPlayer) inbox() types.JID {
	i := slices.IndexFunc(p.ids, func(id types.JID) bool { return id.Server == types.DefaultUserServer })
	if i < 0 {
		return p.jid
	}
	return p.ids[i]
}

func newRoomBallot(players []roomPlayer, options []string) *roomBallot {
	hashes := make([][sha256.Size]byte, len(options))
	for i, option := range options {
		hashes[i] = sha256.Sum256([]byte(option))
	}
	return &roomBallot{players: players, options: hashes, choices: map[int]int{}, decided: make(chan struct{})}
}

func (p *roomBallot) Vote(voter types.JID, selected [][]byte) {
	if p.record(voter, selected) {
		p.saved()
	}
}

func (p *roomBallot) record(voter types.JID, selected [][]byte) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	seat := memberSeat(p.players, voter)
	if seat < 0 {
		return false
	}
	if len(selected) == 0 {
		delete(p.choices, seat)
		return true
	}
	choice := slices.IndexFunc(p.options, func(h [sha256.Size]byte) bool { return bytes.Equal(h[:], selected[0]) })
	if choice < 0 {
		return false
	}
	p.choices[seat] = choice
	p.settle()
	return true
}

func (p *roomBallot) restore(choices map[int]int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	maps.Copy(p.choices, choices)
	p.settle()
}

func (p *roomBallot) settle() {
	if !p.settled() {
		return
	}
	select {
	case <-p.decided:
	default:
		close(p.decided)
	}
}

func (p *roomBallot) chosen() map[int]int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return maps.Clone(p.choices)
}

func (p *roomBallot) settled() bool {
	return len(p.choices) == len(p.players) || slices.Max(p.count())*2 > len(p.players)
}

func (p *roomBallot) tally() []int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.count()
}

func (p *roomBallot) count() []int {
	votes := make([]int, len(p.options))
	for _, choice := range p.choices {
		votes[choice]++
	}
	return votes
}
