package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
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
	*pollAnswer
	author types.JID
	q      quote
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

	round := &quemfalouRound{pollAnswer: newPollAnswer(answer), author: author, q: picked.quote}
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
	voted, winners := r.result()
	if !voted {
		return quote, nil
	}
	if len(winners) == 0 {
		return quote + "\n\n❌ Ninguém acertou 😂", nil
	}
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
