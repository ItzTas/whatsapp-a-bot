package economy

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
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

func (sc voteScenario) choiceOf(label string) int {
	return slices.IndexFunc(sc.choices, func(ch choice) bool { return ch.option(0) == label })
}

func majority(picks []int, options int) (pick, votes int) {
	counts := make([]int, options)
	for _, p := range picks {
		if p >= 0 && p < options {
			counts[p]++
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

func (b *bot) answerLiveVote(ctx context.Context, evt *events.Message) error {
	update := evt.Message.GetPollUpdateMessage()
	if update == nil {
		return nil
	}
	value, ok := b.live.polls.Load(update.GetPollCreationMessageKey().GetID())
	if !ok {
		return nil
	}
	vote, err := b.Client.DecryptPollVote(ctx, evt)
	if err != nil {
		return fmt.Errorf("ler voto da enquete ao vivo: %w", err)
	}
	ev := value.(*liveEvent)
	if ev.ballot(b.Account(ctx, evt.Info.Sender), vote.GetSelectedOptions()) {
		b.keepLive(ctx, ev)
	}
	return nil
}

func (ev *liveEvent) ballot(acc core.Account, selected [][]byte) bool {
	ev.mu.Lock()
	defer ev.mu.Unlock()
	if ev.closed {
		return false
	}
	if len(selected) == 0 {
		delete(ev.ballots, acc.Key)
		return true
	}
	pick := pickedLabel(ev.labels, selected)
	if pick < 0 {
		return false
	}
	ev.ballots[acc.Key] = liveBallot{Voter: core.StoreAccount(acc), Label: ev.labels[pick]}
	return true
}

func (ev *liveEvent) casts() []liveBallot {
	ev.mu.Lock()
	defer ev.mu.Unlock()
	ballots := slices.Collect(maps.Values(ev.ballots))
	slices.SortFunc(ballots, func(a, b liveBallot) int { return strings.Compare(a.Voter.Key, b.Voter.Key) })
	return ballots
}

func (b *bot) heldNotes(ctx context.Context, chat types.JID, voters []core.Account, now time.Time) ([]string, error) {
	notes := make([]string, len(voters))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(liveJobLimit)
	for i, acc := range voters {
		g.Go(func() (err error) { notes[i], err = b.heldNote(gctx, chat, acc, now); return err })
	}
	return notes, g.Wait()
}

func (b *bot) heldNote(ctx context.Context, chat types.JID, acc core.Account, now time.Time) (string, error) {
	var (
		jail    time.Duration
		hostage bool
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { jail, err = b.Wallets.SentenceLeft(gctx, chat, acc, now); return err })
	g.Go(func() error {
		c, err := b.Wallets.captivity(gctx, chat, acc, now)
		hostage = c.left > 0
		return err
	})
	if err := g.Wait(); err != nil {
		return "", err
	}
	if jail > 0 {
		return "🔒 tava preso e ficou de fora", nil
	}
	if hostage {
		return "🪢 tava em cativeiro e ficou de fora", nil
	}
	return "", nil
}

func (b *bot) tallyVote(ctx context.Context, ev *liveEvent) error {
	sc, ok := voteByID(ev.scenario)
	if !ok {
		return nil
	}
	var (
		voters []core.Account
		picks  []int
	)
	for _, v := range ev.casts() {
		if pick := sc.choiceOf(v.Label); pick >= 0 {
			voters = append(voters, v.Voter.Account())
			picks = append(picks, pick)
		}
	}
	if len(picks) == 0 {
		return b.SendText(ctx, ev.chat, fmt.Sprintf("⌛ *%s*: ninguém votou\n\n%s", sc.title, core.PickOne(sc.idle)))
	}
	pick, votes := majority(picks, len(sc.choices))
	ch := sc.choices[pick]
	end := ch.roll(Profession{})
	outcomes, err := b.settleVoters(ctx, ev.chat, voters, end.effects)
	if err != nil {
		return fmt.Errorf("resolver a enquete %s: %w", sc.id, err)
	}
	head := fmt.Sprintf("🗳️ *%s*: a galera escolheu %s (%d de %s)", sc.title, ch.label, votes, core.CountLabel(len(picks), "voto", "votos"))
	blocks := []string{head, core.PickOne(end.texts)}
	mentions := liveFreed(outcomes)
	for i, acc := range voters {
		blocks = append(blocks, liveLine("▫️ @"+acc.JID.User, outcomes[i]))
		mentions = append(mentions, acc.JID)
	}
	_, err = b.SendMentions(ctx, ev.chat, strings.Join(blocks, "\n\n"), mentions)
	return err
}

func (b *bot) settleVoters(ctx context.Context, chat types.JID, voters []core.Account, effects []effect) ([]liveOutcome, error) {
	notes, err := b.heldNotes(ctx, chat, voters, time.Now())
	if err != nil {
		return nil, fmt.Errorf("ver quem tá preso: %w", err)
	}
	outcomes := make([]liveOutcome, len(voters))
	var (
		turns []liveTurn
		slots []int
	)
	for i, acc := range voters {
		if notes[i] != "" {
			outcomes[i].held = notes[i]
			continue
		}
		turns = append(turns, liveTurn{acc: acc, weight: 100, effects: effects})
		slots = append(slots, i)
	}
	played, err := b.playLive(ctx, chat, turns)
	if err != nil {
		return nil, err
	}
	for j, i := range slots {
		outcomes[i] = played[j]
	}
	return outcomes, nil
}
