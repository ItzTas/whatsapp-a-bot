package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	impeachmentWindow = 10 * time.Minute
	impeachmentQuorum = 3
	impeachmentEvery  = 6 * time.Hour
	impeachmentKind   = "impeachment"
)

type impeachment struct {
	mu        sync.Mutex
	opened    time.Time
	president account
	ballots   map[string]bool
	weights   map[string]int
	closed    bool
}

type impeachmentState struct {
	Chat      types.JID       `json:"chat"`
	Opened    time.Time       `json:"opened"`
	President storedAccount   `json:"president"`
	Ballots   map[string]bool `json:"ballots"`
	Weights   map[string]int  `json:"weights"`
}

func (b *bot) impeachment(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	voter := b.account(ctx, evt.Info.Sender.ToNonAD())

	var (
		i               *impeachment
		opened, blocked bool
		voterJob        profession
		g               errgroup.Group
	)
	g.Go(func() (err error) { i, opened, blocked, err = b.openImpeachment(ctx, chat); return err })
	g.Go(func() (err error) { voterJob, err = b.professionOf(ctx, chat, voter); return err })
	if err := g.Wait(); err != nil || blocked {
		return err
	}
	favor := impeachmentChoice(messageText(evt.Message))
	cast := b.castImpeachmentBallot(ctx, chat, i, voter, favor, voterJob.ballotWeight())
	if cast.refused {
		return b.sendText(ctx, chat, "⚖️ A apuração do impeachment já começou. Espera o resultado")
	}
	_, err := b.sendMentions(ctx, chat, i.ballotText(voter, favor, cast, opened), []types.JID{voter.jid, i.president.jid})
	return err
}

func impeachmentChoice(text string) bool {
	answers := map[string]bool{"sim": true, "favor": true, "fora": true, "nao": false, "contra": false, "fica": false}
	for _, word := range strings.Fields(foldLetters(text))[1:] {
		if favor, ok := answers[word]; ok {
			return favor
		}
	}
	return true
}

func (b *bot) openImpeachment(ctx context.Context, chat types.JID) (i *impeachment, opened, blocked bool, err error) {
	if value, running := b.impeachments.Load(chat.String()); running {
		return value.(*impeachment), false, false, nil
	}
	now := time.Now()
	p, found, err := b.wallets.presidency(ctx, chat, now)
	if err != nil {
		return nil, false, false, fmt.Errorf("ler presidência: %w", err)
	}
	if !found {
		return nil, false, true, b.sendText(ctx, chat, "🏛️ Ninguém é presidente agora, então não tem quem tirar do cargo. Manda !votar @pessoa pra abrir a eleição")
	}
	if wait := p.lastImpeachment.Add(impeachmentEvery).Sub(now); wait > 0 {
		text := fmt.Sprintf("⚖️ Já teve pedido de impeachment contra @%s há pouco. O próximo só daqui a %s", p.president.jid.User, formatWait(wait))
		_, err = b.sendMentions(ctx, chat, text, []types.JID{p.president.jid})
		return nil, false, true, err
	}
	value, running := b.impeachments.LoadOrStore(chat.String(), &impeachment{opened: now, president: p.president, ballots: map[string]bool{}, weights: map[string]int{}})
	i = value.(*impeachment)
	if running {
		return i, false, false, nil
	}
	if err := b.wallets.stampImpeachment(ctx, chat, p.president, now); err != nil {
		b.client.Log.Errorf("marcar impeachment: %v", err)
	}
	go b.closeImpeachment(ctx, chat, i)
	return i, true, false, nil
}

func (b *bot) castImpeachmentBallot(ctx context.Context, chat types.JID, i *impeachment, voter account, favor bool, weight int) ballot {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.closed {
		return ballot{refused: true}
	}
	_, changed := i.ballots[voter.key]
	i.ballots[voter.key] = favor
	i.weights[voter.key] = weight
	b.keepImpeachment(ctx, chat, i)
	return ballot{changed: changed, weight: weight}
}

func (b *bot) keepImpeachment(ctx context.Context, chat types.JID, i *impeachment) {
	state := impeachmentState{Chat: chat, Opened: i.opened, President: storeAccount(i.president), Ballots: i.ballots, Weights: i.weights}
	b.keepGame(ctx, impeachmentKind, chat.String(), state, i.opened.Add(impeachmentWindow))
}

func (b *bot) restoreImpeachment(ctx context.Context, saved savedGame) error {
	var state impeachmentState
	if err := json.Unmarshal(saved.state, &state); err != nil {
		return fmt.Errorf("ler impeachment salvo: %w", err)
	}
	i := &impeachment{opened: state.Opened, president: state.President.account(), ballots: state.Ballots, weights: state.Weights}
	if _, running := b.impeachments.LoadOrStore(state.Chat.String(), i); running {
		return fmt.Errorf("já tem outro impeachment rolando em %s", state.Chat)
	}
	go b.closeImpeachment(ctx, state.Chat, i)
	return nil
}

func (i *impeachment) count() (favor, against int) {
	for voter, yes := range i.ballots {
		weight := max(i.weights[voter], 1)
		if yes {
			favor += weight
			continue
		}
		against += weight
	}
	return favor, against
}

func (i *impeachment) closeBallots(ctx context.Context, b *bot, chat types.JID) (favor, against int) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.closed = true
	b.dropGame(ctx, impeachmentKind, chat.String())
	return i.count()
}

func (i *impeachment) ballotText(voter account, favor bool, cast ballot, opened bool) string {
	side := "a favor do impeachment"
	if !favor {
		side = "contra o impeachment"
	}
	vote := fmt.Sprintf("@%s votou %s", voter.jid.User, side)
	if cast.changed {
		vote = fmt.Sprintf("@%s mudou o voto: agora é %s", voter.jid.User, side)
	}
	if cast.weight > 1 {
		vote += fmt.Sprintf(" (e o voto vale %d)", cast.weight)
	}
	left := formatWait(impeachmentWindow - time.Since(i.opened))
	if !opened {
		return fmt.Sprintf("⚖️ %s. A votação fecha em %s", vote, left)
	}
	return fmt.Sprintf("⚖️ *Pedido de impeachment!*\n%s e abriu a votação pra tirar @%s da presidência. Manda !impeachment sim ou !impeachment nao em até %s (dá pra mudar o voto até lá)\n\nPra passar, precisa de pelo menos %d votos a favor e mais votos a favor do que contra",
		vote, i.president.jid.User, left, impeachmentQuorum)
}

func (i *impeachment) partialText() string {
	i.mu.Lock()
	favor, against := i.count()
	i.mu.Unlock()
	return fmt.Sprintf("⚖️ *Impeachment rolando!* A favor: %s. Contra: %s. A votação fecha em %s",
		countLabel(favor, "voto", "votos"), countLabel(against, "voto", "votos"), formatWait(impeachmentWindow-time.Since(i.opened)))
}

func (b *bot) closeImpeachment(ctx context.Context, chat types.JID, i *impeachment) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Until(i.opened.Add(impeachmentWindow))):
	}
	favor, against := i.closeBallots(ctx, b, chat)
	b.impeachments.CompareAndDelete(chat.String(), i)
	text, err := b.impeachmentResult(ctx, chat, i.president, favor, against)
	if err != nil {
		b.client.Log.Errorf("apurar impeachment: %v", err)
		return
	}
	if _, err := b.sendMentions(ctx, chat, text, []types.JID{i.president.jid}); err != nil {
		b.client.Log.Errorf("anunciar impeachment: %v", err)
	}
}

func (b *bot) impeachmentResult(ctx context.Context, chat types.JID, president account, favor, against int) (string, error) {
	tally := fmt.Sprintf("A favor: %s\nContra: %s", countLabel(favor, "voto", "votos"), countLabel(against, "voto", "votos"))
	if favor < impeachmentQuorum || favor <= against {
		return fmt.Sprintf("⚖️ *Impeachment rejeitado!*\n%s\n\n🏛️ @%s continua no cargo. Pra passar, precisava de pelo menos %d votos a favor e mais votos a favor do que contra. Outro pedido só daqui a %s",
			tally, president.jid.User, impeachmentQuorum, formatWait(impeachmentEvery-impeachmentWindow)), nil
	}
	removed, err := b.wallets.impeach(ctx, chat, president, time.Now())
	if err != nil {
		return "", fmt.Errorf("tirar presidente: %w", err)
	}
	if !removed {
		return fmt.Sprintf("⚖️ O impeachment passou, mas o mandato de @%s já tinha acabado", president.jid.User), nil
	}
	return fmt.Sprintf("⚖️ *Impeachment aprovado!*\n%s\n\n🏛️ @%s perdeu o cargo, e os poderes e decretos da presidência caíram junto. Manda !votar @pessoa pra abrir uma eleição nova", tally, president.jid.User), nil
}

func (s *walletStore) stampImpeachment(ctx context.Context, chat types.JID, president account, now time.Time) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE presidencias SET impeachment_em = ? WHERE chat = ? AND pessoa = ? AND mandato_ate > ?",
		now.Unix(), chat.String(), president.key, now.Unix(),
	)
	return err
}

func (s *walletStore) impeach(ctx context.Context, chat types.JID, president account, now time.Time) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		"UPDATE presidencias SET mandato_ate = ? WHERE chat = ? AND pessoa = ? AND mandato_ate > ?",
		now.Unix(), chat.String(), president.key, now.Unix(),
	)
	if err != nil {
		return false, err
	}
	removed, err := res.RowsAffected()
	return removed > 0, err
}
