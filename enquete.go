package main

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
	vote(voter types.JID, selected [][]byte)
}

type pollAnswer struct {
	hash [sha256.Size]byte

	mu    sync.Mutex
	votes map[string]pollVote
}

type pollVote struct {
	voter   types.JID
	correct bool
}

func newPollAnswer(answer string) *pollAnswer {
	return &pollAnswer{hash: sha256.Sum256([]byte(answer)), votes: map[string]pollVote{}}
}

func (p *pollAnswer) vote(voter types.JID, selected [][]byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(selected) == 0 {
		delete(p.votes, voter.User)
		return
	}
	correct := slices.ContainsFunc(selected, func(h []byte) bool { return bytes.Equal(h, p.hash[:]) })
	p.votes[voter.User] = pollVote{voter: voter, correct: correct}
}

func (p *pollAnswer) result() (voted bool, winners []types.JID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, v := range p.votes {
		if v.correct {
			winners = append(winners, v.voter)
		}
	}
	slices.SortFunc(winners, func(a, b types.JID) int { return strings.Compare(a.User, b.User) })
	return len(p.votes) > 0, winners
}

func (b *bot) answerPoll(ctx context.Context, evt *events.Message, rounds *sync.Map) error {
	update := evt.Message.GetPollUpdateMessage()
	if update == nil {
		return nil
	}
	value, ok := rounds.Load(update.GetPollCreationMessageKey().GetID())
	if !ok {
		return nil
	}
	vote, err := b.client.DecryptPollVote(ctx, evt)
	if err != nil {
		return fmt.Errorf("ler voto: %w", err)
	}
	value.(pollVoter).vote(evt.Info.Sender.ToNonAD(), vote.GetSelectedOptions())
	return nil
}
