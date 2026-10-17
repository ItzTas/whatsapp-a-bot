package core

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"sync"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

type pollVoter interface {
	Vote(voter types.JID, selected [][]byte)
}

type PollAnswer struct {
	hash [sha256.Size]byte

	mu    sync.Mutex
	votes map[string]PollVote
}

type PollVote struct {
	Voter   types.JID
	Correct bool
}

type StoredPoll struct {
	Hash  []byte       `json:"hash"`
	Votes []storedVote `json:"votes"`
}

type storedVote struct {
	Voter   types.JID `json:"voter"`
	Correct bool      `json:"correct"`
}

func NewPollAnswer(answer string) *PollAnswer {
	return &PollAnswer{hash: sha256.Sum256([]byte(answer)), votes: map[string]PollVote{}}
}

func RestorePollAnswer(stored StoredPoll) *PollAnswer {
	p := &PollAnswer{votes: map[string]PollVote{}}
	copy(p.hash[:], stored.Hash)
	for _, v := range stored.Votes {
		p.votes[v.Voter.User] = PollVote(v)
	}
	return p
}

func (p *PollAnswer) Stored() StoredPoll {
	p.mu.Lock()
	defer p.mu.Unlock()
	stored := StoredPoll{Hash: p.hash[:]}
	for _, v := range p.votes {
		stored.Votes = append(stored.Votes, storedVote(v))
	}
	return stored
}

func (p *PollAnswer) Vote(voter types.JID, selected [][]byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(selected) == 0 {
		delete(p.votes, voter.User)
		return
	}
	correct := slices.ContainsFunc(selected, func(h []byte) bool { return bytes.Equal(h, p.hash[:]) })
	p.votes[voter.User] = PollVote{Voter: voter, Correct: correct}
}

func (p *PollAnswer) Result() (voted bool, winners []types.JID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, v := range p.votes {
		if v.Correct {
			winners = append(winners, v.Voter)
		}
	}
	slices.SortFunc(winners, func(a, b types.JID) int { return strings.Compare(a.User, b.User) })
	return len(p.votes) > 0, winners
}

func (p *PollAnswer) Ballots() []PollVote {
	p.mu.Lock()
	defer p.mu.Unlock()
	ballots := make([]PollVote, 0, len(p.votes))
	for _, v := range p.votes {
		ballots = append(ballots, v)
	}
	return ballots
}

func (p *PollAnswer) Voters() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.votes)
}

func (b *Bot) AnswerPoll(ctx context.Context, evt *events.Message, rounds *sync.Map) error {
	update := evt.Message.GetPollUpdateMessage()
	if update == nil {
		return nil
	}
	value, ok := rounds.Load(update.GetPollCreationMessageKey().GetID())
	if !ok {
		return nil
	}
	vote, err := b.Client.DecryptPollVote(ctx, evt)
	if err != nil {
		return fmt.Errorf("ler voto: %w", err)
	}
	value.(pollVoter).Vote(evt.Info.Sender.ToNonAD(), vote.GetSelectedOptions())
	return nil
}
