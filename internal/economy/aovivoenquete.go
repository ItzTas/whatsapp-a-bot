package economy

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

const (
	voteWindow   = 40 * time.Minute
	voteQuestion = "🗳️ O que a galera faz?"
)

type voteScenario struct {
	id       string
	title    string
	rarity   eventRarity
	question string
	intros   []string
	choices  []choice
	idle     []string
}

func voteByID(id string) (voteScenario, bool) {
	i := slices.IndexFunc(voteScenarios, func(sc voteScenario) bool { return sc.id == id })
	if i < 0 {
		return voteScenario{}, false
	}
	return voteScenarios[i], true
}

func (sc voteScenario) ask() string {
	if sc.question != "" {
		return sc.question
	}
	return voteQuestion
}

func setupVote(ev *liveEvent, _ int) {
	sc, ok := voteByID(ev.scenario)
	if !ok {
		return
	}
	for _, ch := range sc.choices {
		ev.labels = append(ev.labels, ch.option(0))
	}
}

func majority(ballots []liveBallot, options int) (pick, votes int) {
	counts := make([]int, options)
	for _, v := range ballots {
		if v.Choice >= 0 && v.Choice < options {
			counts[v.Choice]++
		}
	}
	votes = slices.Max(counts)
	var tied []int
	for i, n := range counts {
		if n == votes {
			tied = append(tied, i)
		}
	}
	return core.PickOne(tied), votes
}

func (b *bot) showVote(ctx context.Context, ev *liveEvent, players []types.JID) error {
	sc, ok := voteByID(ev.scenario)
	if !ok {
		return fmt.Errorf("enquete %s não existe", ev.scenario)
	}
	rules := fmt.Sprintf("_Vota na enquete aqui embaixo em até %s: a opção mais votada vale pra todo mundo que votar_", core.FormatWait(voteWindow))
	text := liveHeader("🗳️", sc.title, sc.rarity) + "\n\n" + core.PickOne(sc.intros) + "\n\n" + rules + liveCall(players)
	if _, err := b.SendMentions(ctx, ev.chat, text, players); err != nil {
		return err
	}
	poll, err := b.Client.SendMessage(ctx, ev.chat, b.Client.BuildPollCreation(sc.ask(), ev.labels, 1))
	if err != nil {
		return fmt.Errorf("mandar a enquete: %w", err)
	}
	ev.setPoll(poll.ID)
	return nil
}

func (ev *liveEvent) setPoll(id types.MessageID) {
	ev.mu.Lock()
	defer ev.mu.Unlock()
	ev.poll = id
}

func (ev *liveEvent) Vote(voter types.JID, selected [][]byte) {
	if keep := ev.ballot(voter, selected); keep != nil {
		keep()
	}
}

func (ev *liveEvent) ballot(voter types.JID, selected [][]byte) func() {
	ev.mu.Lock()
	defer ev.mu.Unlock()
	if ev.closed {
		return nil
	}
	if len(selected) == 0 {
		delete(ev.ballots, voter.User)
		return ev.keep
	}
	pick := pickedLabel(ev.labels, selected)
	if pick < 0 {
		return nil
	}
	ev.ballots[voter.User] = liveBallot{Voter: voter, Choice: pick}
	return ev.keep
}

func (ev *liveEvent) casts() []liveBallot {
	ev.mu.Lock()
	defer ev.mu.Unlock()
	ballots := make([]liveBallot, 0, len(ev.ballots))
	for _, v := range ev.ballots {
		ballots = append(ballots, v)
	}
	slices.SortFunc(ballots, func(a, b liveBallot) int { return strings.Compare(a.Voter.User, b.Voter.User) })
	return ballots
}

func (b *bot) voters(ctx context.Context, ballots []liveBallot) []core.Account {
	var voters []core.Account
	for _, v := range ballots {
		acc := b.Account(ctx, v.Voter)
		if !slices.ContainsFunc(voters, func(o core.Account) bool { return o.Key == acc.Key }) {
			voters = append(voters, acc)
		}
	}
	return voters
}

func (b *bot) tallyVote(ctx context.Context, ev *liveEvent) error {
	sc, ok := voteByID(ev.scenario)
	if !ok {
		return nil
	}
	ballots := ev.casts()
	if len(ballots) == 0 {
		return b.SendText(ctx, ev.chat, fmt.Sprintf("⌛ *%s*: ninguém votou\n\n%s", sc.title, core.PickOne(sc.idle)))
	}
	pick, votes := majority(ballots, len(sc.choices))
	ch := sc.choices[pick]
	end := ch.roll(Profession{})
	voters := b.voters(ctx, ballots)
	turns := make([]liveTurn, len(voters))
	for i, acc := range voters {
		turns[i] = liveTurn{acc: acc, weight: 100, effects: end.effects}
	}
	results, err := b.playLive(ctx, ev.chat, turns)
	if err != nil {
		return fmt.Errorf("resolver a enquete %s: %w", sc.id, err)
	}
	head := fmt.Sprintf("🗳️ *%s*: a galera escolheu %s (%d de %s)", sc.title, ch.label, votes, core.CountLabel(len(ballots), "voto", "votos"))
	blocks := []string{head, core.PickOne(end.texts)}
	mentions := liveFreed(results)
	for i, acc := range voters {
		blocks = append(blocks, liveLine("▫️ @"+acc.JID.User, results[i]))
		mentions = append(mentions, acc.JID)
	}
	_, err = b.SendMentions(ctx, ev.chat, strings.Join(blocks, "\n\n"), mentions)
	return err
}
