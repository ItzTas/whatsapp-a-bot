package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	quemfalouOptionCount = 4
	quemfalouCandidates  = 30
	quemfalouRevealDelay = time.Minute
)

type storedQuote struct {
	quote
	sender    string
	senderAlt string
}

type quemfalouRound struct {
	author     types.JID
	q          quote
	answerHash [sha256.Size]byte

	mu    sync.Mutex
	votes map[string]quemfalouVote
}

type quemfalouVote struct {
	voter   types.JID
	correct bool
}

func (b *bot) quemfalou(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return b.sendText(ctx, chat, "O !quemfalou só funciona em grupo")
	}

	var (
		group  *types.GroupInfo
		quotes []storedQuote
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { group, err = b.client.GetGroupInfo(gctx, chat); return err })
	g.Go(func() (err error) { quotes, err = b.messages.randomQuotes(gctx, chat, quemfalouCandidates); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("preparar !quemfalou: %w", err)
	}

	owners := b.memberOwners(group.Participants)
	i := slices.IndexFunc(quotes, func(q storedQuote) bool { _, ok := owners.of(q.sender, q.senderAlt); return ok })
	if i < 0 {
		return b.sendText(ctx, chat, "Ainda não tenho mensagem guardada de ninguém daqui 🤷")
	}
	author, _ := owners.of(quotes[i].sender, quotes[i].senderAlt)
	options := quemfalouOptions(owners, author)
	if len(options) < 2 {
		return b.sendText(ctx, chat, "O grupo precisa ter pelo menos 2 pessoas além do bot")
	}
	names := b.pollNames(ctx, chat, options)
	answer := names[slices.Index(options, author)]

	intro := fmt.Sprintf("🕵️ *Quem falou?*\n\n%s\n\n_Vota na enquete, a resposta sai em 1 minuto_ ⏳", quoteBlock(quotes[i].text))
	if err := b.sendText(ctx, chat, intro); err != nil {
		return err
	}
	resp, err := b.client.SendMessage(ctx, chat, b.client.BuildPollCreation("🤔 Quem mandou essa?", names, 1))
	if err != nil {
		return err
	}

	round := &quemfalouRound{
		author:     author,
		q:          quotes[i].quote,
		answerHash: sha256.Sum256([]byte(answer)),
		votes:      map[string]quemfalouVote{},
	}
	b.quemfalouRounds.Store(resp.ID, round)
	go b.revealQuemfalou(ctx, chat, resp.ID, round)
	return nil
}

func quemfalouOptions(owners memberOwners, author types.JID) []types.JID {
	var others []types.JID
	for _, member := range owners {
		if member == author || slices.Contains(others, member) {
			continue
		}
		others = append(others, member)
	}
	rand.Shuffle(len(others), func(i, j int) { others[i], others[j] = others[j], others[i] })
	options := append(others[:min(len(others), quemfalouOptionCount-1)], author)
	rand.Shuffle(len(options), func(i, j int) { options[i], options[j] = options[j], options[i] })
	return options
}

func (b *bot) answerQuemfalou(ctx context.Context, evt *events.Message) error {
	update := evt.Message.GetPollUpdateMessage()
	if update == nil {
		return nil
	}
	value, ok := b.quemfalouRounds.Load(update.GetPollCreationMessageKey().GetID())
	if !ok {
		return nil
	}
	vote, err := b.client.DecryptPollVote(ctx, evt)
	if err != nil {
		return fmt.Errorf("ler voto: %w", err)
	}
	value.(*quemfalouRound).vote(evt.Info.Sender.ToNonAD(), vote.GetSelectedOptions())
	return nil
}

func (r *quemfalouRound) vote(voter types.JID, selected [][]byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(selected) == 0 {
		delete(r.votes, voter.User)
		return
	}
	correct := slices.ContainsFunc(selected, func(h []byte) bool { return bytes.Equal(h, r.answerHash[:]) })
	r.votes[voter.User] = quemfalouVote{voter: voter, correct: correct}
}

func (b *bot) revealQuemfalou(ctx context.Context, chat types.JID, pollID types.MessageID, round *quemfalouRound) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(quemfalouRevealDelay):
	}
	b.quemfalouRounds.Delete(pollID)
	text, mentions := round.reveal()
	if _, err := b.sendMentions(ctx, chat, text, mentions); err != nil {
		b.client.Log.Errorf("revelar !quemfalou: %v", err)
	}
}

func (r *quemfalouRound) reveal() (string, []types.JID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	lines := []string{fmt.Sprintf("🕵️ Quem falou foi @%s, em %s!", r.author.User, brazilianDate(r.q.sentAt)), "", quoteBlock(r.q.text)}
	mentions := []types.JID{r.author}
	if len(r.votes) == 0 {
		return strings.Join(lines, "\n"), mentions
	}

	var winners []types.JID
	for _, v := range r.votes {
		if v.correct {
			winners = append(winners, v.voter)
		}
	}
	if len(winners) == 0 {
		return strings.Join(append(lines, "", "❌ Ninguém acertou 😂"), "\n"), mentions
	}
	slices.SortFunc(winners, func(a, b types.JID) int { return strings.Compare(a.User, b.User) })
	lines = append(lines, "", "✅ Acertaram: "+joinMentions(winners))
	return strings.Join(lines, "\n"), append(mentions, winners...)
}

func (s *messageStore) randomQuotes(ctx context.Context, chat types.JID, limit int) (quotes []storedQuote, err error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT remetente, remetente_alt, texto, enviada_em FROM mensagens
		 WHERE chat = ? AND enviada_em >= ?
		   AND texto NOT LIKE '!%' AND length(texto) >= ?
		 ORDER BY RANDOM() LIMIT ?`,
		chat.String(), windowStart(), minQuoteLength, limit,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var (
			q      storedQuote
			sentAt int64
		)
		if err := rows.Scan(&q.sender, &q.senderAlt, &q.text, &sentAt); err != nil {
			return nil, err
		}
		q.sentAt = time.Unix(sentAt, 0)
		quotes = append(quotes, q)
	}
	return quotes, rows.Err()
}
