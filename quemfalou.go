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
	quemfalouCandidates  = 80
	quemfalouRevealDelay = time.Minute
)

type storedQuote struct {
	quote
	sender    string
	senderAlt string
	laughs    int
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

	phrase := quemfalouPhrase(messageText(evt.Message))
	var (
		group  *types.GroupInfo
		quotes []storedQuote
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { group, err = b.client.GetGroupInfo(gctx, chat); return err })
	g.Go(func() (err error) {
		quotes, err = b.messages.randomQuotes(gctx, chat, phrase, quemfalouCandidates)
		return err
	})
	if err := g.Wait(); err != nil {
		return fmt.Errorf("preparar !quemfalou: %w", err)
	}

	owners := b.memberOwners(group.Participants)
	picked, author, found := pickQuemfalouQuote(quotes, owners, phrase == "")
	if !found && phrase != "" {
		return b.sendText(ctx, chat, "Não achei nenhuma mensagem de alguém daqui com essa frase 🤷")
	}
	if !found {
		return b.sendText(ctx, chat, "Ainda não tenho mensagem boa guardada de ninguém daqui 🤷")
	}
	options := quemfalouOptions(owners, author)
	if len(options) < 2 {
		return b.sendText(ctx, chat, "O grupo precisa ter pelo menos 2 pessoas além do bot")
	}
	names := b.pollNames(ctx, chat, options)
	answer := names[slices.Index(options, author)]

	intro := fmt.Sprintf("🕵️ *Quem falou?*\n\n%s\n\n_Vota na enquete, a resposta sai em 1 minuto_ ⏳", quoteBlock(picked.text))
	if err := b.sendText(ctx, chat, intro); err != nil {
		return err
	}
	resp, err := b.client.SendMessage(ctx, chat, b.client.BuildPollCreation("🤔 Quem mandou essa?", names, 1))
	if err != nil {
		return err
	}

	round := &quemfalouRound{
		author:     author,
		q:          picked.quote,
		answerHash: sha256.Sum256([]byte(answer)),
		votes:      map[string]quemfalouVote{},
	}
	b.quemfalouRounds.Store(resp.ID, round)
	go b.revealQuemfalou(ctx, chat, resp.ID, round)
	return nil
}

func pickQuemfalouQuote(quotes []storedQuote, owners memberOwners, onlyWorthQuoting bool) (storedQuote, types.JID, bool) {
	var (
		best   storedQuote
		author types.JID
		found  bool
	)
	for _, q := range quotes {
		owner, ok := owners.of(q.sender, q.senderAlt)
		if !ok || (onlyWorthQuoting && !q.worthQuoting()) || (found && q.laughs <= best.laughs) {
			continue
		}
		best, author, found = q, owner, true
	}
	return best, author, found
}

func quemfalouPhrase(text string) string {
	fields := strings.Fields(text)
	if len(fields) < 2 {
		return ""
	}
	return strings.Trim(strings.Join(fields[1:], " "), `"'“”‘’`)
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
	nicknames := b.chatNicknames(ctx, chat)
	text, mentions := round.reveal(b.fullName(ctx, round.author, nicknames), nicknames)
	if _, err := b.sendRawMentions(ctx, chat, text, mentions); err != nil {
		b.client.Log.Errorf("revelar !quemfalou: %v", err)
	}
}

func (r *quemfalouRound) reveal(authorName string, nicknames map[string]string) (string, []types.JID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	author := "@" + r.author.User
	if authorName != "" {
		author = fmt.Sprintf("%s (%s)", author, authorName)
	}
	header := fmt.Sprintf("🕵️ Quem falou foi %s, em %s!", author, brazilianDate(r.q.sentAt))
	body, winners := r.outcome()
	return header + "\n\n" + tagNicknames(body, nicknames), append([]types.JID{r.author}, winners...)
}

func (r *quemfalouRound) outcome() (string, []types.JID) {
	quote := quoteBlock(r.q.text)
	if len(r.votes) == 0 {
		return quote, nil
	}

	var winners []types.JID
	for _, v := range r.votes {
		if v.correct {
			winners = append(winners, v.voter)
		}
	}
	if len(winners) == 0 {
		return quote + "\n\n❌ Ninguém acertou 😂", nil
	}
	slices.SortFunc(winners, func(a, b types.JID) int { return strings.Compare(a.User, b.User) })
	return quote + "\n\n✅ Acertaram: " + joinMentions(winners), winners
}

func (s *messageStore) randomQuotes(ctx context.Context, chat types.JID, phrase string, limit int) (quotes []storedQuote, err error) {
	pattern, minLength := quoteFilter(phrase)
	rows, err := s.db.QueryContext(ctx,
		`SELECT m.remetente, m.remetente_alt, m.texto, m.enviada_em,
		        (SELECT COUNT(DISTINCT r.quem_riu) FROM risadas r WHERE r.chat = m.chat AND r.mensagem_alvo = m.id)
		 FROM mensagens m
		 WHERE m.chat = ? AND m.enviada_em >= ?
		   AND m.texto NOT LIKE '!%' AND length(m.texto) >= ?
		   AND m.texto LIKE ? ESCAPE '\'
		 ORDER BY RANDOM() LIMIT ?`,
		chat.String(), windowStart(), minLength, pattern, limit,
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
		if err := rows.Scan(&q.sender, &q.senderAlt, &q.text, &sentAt, &q.laughs); err != nil {
			return nil, err
		}
		q.sentAt = time.Unix(sentAt, 0)
		quotes = append(quotes, q)
	}
	return quotes, rows.Err()
}

func quoteFilter(phrase string) (string, int) {
	if phrase == "" {
		return "%", minQuoteLength
	}
	escaped := strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(phrase)
	return "%" + escaped + "%", 1
}
