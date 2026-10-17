package economy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
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
	president core.Account
	ballots   map[string]bool
	weights   map[string]int
	closed    bool
}

type impeachmentState struct {
	Chat      types.JID          `json:"chat"`
	Opened    time.Time          `json:"opened"`
	President core.StoredAccount `json:"president"`
	Ballots   map[string]bool    `json:"ballots"`
	Weights   map[string]int     `json:"weights"`
}

func (b *bot) impeachment(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	voter := b.Account(ctx, evt.Info.Sender.ToNonAD())

	var (
		i               *impeachment
		opened, blocked bool
		voterJob        Profession
		g               errgroup.Group
	)
	g.Go(func() (err error) { i, opened, blocked, err = b.openImpeachment(ctx, chat); return err })
	g.Go(func() (err error) { voterJob, err = b.ProfessionOf(ctx, chat, voter); return err })
	if err := g.Wait(); err != nil || blocked {
		return err
	}
	favor := impeachmentChoice(core.MessageText(evt.Message))
	cast := b.castImpeachmentBallot(ctx, chat, i, voter, favor, voterJob.ballotWeight())
	if cast.refused {
		return b.SendText(ctx, chat, "⚖️ A apuração do impeachment já começou. Espera o resultado")
	}
	_, err := b.SendMentions(ctx, chat, i.ballotText(voter, favor, cast, opened), []types.JID{voter.JID, i.president.JID})
	return err
}

func impeachmentChoice(text string) bool {
	answers := map[string]bool{"sim": true, "favor": true, "fora": true, "nao": false, "contra": false, "fica": false}
	for _, word := range strings.Fields(core.FoldLetters(text))[1:] {
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
	p, found, err := b.Wallets.presidency(ctx, chat, now)
	if err != nil {
		return nil, false, false, fmt.Errorf("ler presidência: %w", err)
	}
	if !found {
		return nil, false, true, b.SendText(ctx, chat, "🏛️ Ninguém é presidente agora, então não tem quem tirar do cargo. Manda !votar @pessoa pra abrir a eleição")
	}
	if wait := p.lastImpeachment.Add(impeachmentEvery).Sub(now); wait > 0 {
		text := fmt.Sprintf("⚖️ Já teve pedido de impeachment contra @%s há pouco. O próximo só daqui a %s", p.president.JID.User, core.FormatWait(wait))
		_, err = b.SendMentions(ctx, chat, text, []types.JID{p.president.JID})
		return nil, false, true, err
	}
	value, running := b.impeachments.LoadOrStore(chat.String(), &impeachment{opened: now, president: p.president, ballots: map[string]bool{}, weights: map[string]int{}})
	i = value.(*impeachment)
	if running {
		return i, false, false, nil
	}
	if err := b.Wallets.stampImpeachment(ctx, chat, p.president, now); err != nil {
		b.Client.Log.Errorf("marcar impeachment: %v", err)
	}
	go b.closeImpeachment(ctx, chat, i)
	return i, true, false, nil
}

func (b *bot) castImpeachmentBallot(ctx context.Context, chat types.JID, i *impeachment, voter core.Account, favor bool, weight int) ballot {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.closed {
		return ballot{refused: true}
	}
	_, changed := i.ballots[voter.Key]
	i.ballots[voter.Key] = favor
	i.weights[voter.Key] = weight
	b.keepImpeachment(ctx, chat, i)
	return ballot{changed: changed, weight: weight}
}

func (b *bot) keepImpeachment(ctx context.Context, chat types.JID, i *impeachment) {
	state := impeachmentState{Chat: chat, Opened: i.opened, President: core.StoreAccount(i.president), Ballots: i.ballots, Weights: i.weights}
	b.KeepGame(ctx, impeachmentKind, chat.String(), state, i.opened.Add(impeachmentWindow))
}

func (b *bot) restoreImpeachment(ctx context.Context, saved core.SavedGame) error {
	var state impeachmentState
	if err := json.Unmarshal(saved.State, &state); err != nil {
		return fmt.Errorf("ler impeachment salvo: %w", err)
	}
	i := &impeachment{opened: state.Opened, president: state.President.Account(), ballots: state.Ballots, weights: state.Weights}
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
	b.DropGame(ctx, impeachmentKind, chat.String())
	return i.count()
}

func (i *impeachment) ballotText(voter core.Account, favor bool, cast ballot, opened bool) string {
	side := "a favor do impeachment"
	if !favor {
		side = "contra o impeachment"
	}
	vote := fmt.Sprintf("@%s votou %s", voter.JID.User, side)
	if cast.changed {
		vote = fmt.Sprintf("@%s mudou o voto: agora é %s", voter.JID.User, side)
	}
	if cast.weight > 1 {
		vote += fmt.Sprintf(" (e o voto vale %d)", cast.weight)
	}
	left := core.FormatWait(impeachmentWindow - time.Since(i.opened))
	if !opened {
		return fmt.Sprintf("⚖️ %s. A votação fecha em %s", vote, left)
	}
	return fmt.Sprintf("⚖️ *Pedido de impeachment!*\n%s e abriu a votação pra tirar @%s da presidência. Manda !impeachment sim ou !impeachment nao em até %s (dá pra mudar o voto até lá)\n\nPra passar, precisa de pelo menos %d votos a favor e mais votos a favor do que contra",
		vote, i.president.JID.User, left, impeachmentQuorum)
}

func (i *impeachment) partialText() string {
	i.mu.Lock()
	favor, against := i.count()
	i.mu.Unlock()
	return fmt.Sprintf("⚖️ *Impeachment rolando!* A favor: %s. Contra: %s. A votação fecha em %s",
		core.CountLabel(favor, "voto", "votos"), core.CountLabel(against, "voto", "votos"), core.FormatWait(impeachmentWindow-time.Since(i.opened)))
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
		b.Client.Log.Errorf("apurar impeachment: %v", err)
		return
	}
	if _, err := b.SendMentions(ctx, chat, text, []types.JID{i.president.JID}); err != nil {
		b.Client.Log.Errorf("anunciar impeachment: %v", err)
	}
}

func (b *bot) impeachmentResult(ctx context.Context, chat types.JID, president core.Account, favor, against int) (string, error) {
	p, found, err := b.Wallets.presidency(ctx, chat, time.Now())
	if err != nil {
		return "", fmt.Errorf("ler presidência: %w", err)
	}
	if !found || p.president.Key != president.Key {
		return fmt.Sprintf("⚖️ A votação do impeachment fechou, mas @%s já não é mais presidente", president.JID.User), nil
	}
	tally := fmt.Sprintf("A favor: %s\nContra: %s", core.CountLabel(favor, "voto", "votos"), core.CountLabel(against, "voto", "votos"))
	if favor < impeachmentQuorum || favor <= against {
		return fmt.Sprintf("⚖️ *Impeachment rejeitado!*\n%s\n\n🏛️ @%s continua no cargo. Pra passar, precisava de pelo menos %d votos a favor e mais votos a favor do que contra. Outro pedido só daqui a %s",
			tally, president.JID.User, impeachmentQuorum, core.FormatWait(impeachmentEvery-impeachmentWindow)), nil
	}
	removed, err := b.Wallets.impeach(ctx, chat, president, time.Now())
	if err != nil {
		return "", fmt.Errorf("tirar presidente: %w", err)
	}
	if !removed {
		return fmt.Sprintf("⚖️ O impeachment passou, mas o mandato de @%s já tinha acabado", president.JID.User), nil
	}
	return fmt.Sprintf("⚖️ *Impeachment aprovado!*\n%s\n\n🏛️ @%s perdeu o cargo, e os poderes e decretos da presidência caíram junto. Manda !votar @pessoa pra abrir uma eleição nova", tally, president.JID.User), nil
}

func (s *walletStore) stampImpeachment(ctx context.Context, chat types.JID, president core.Account, now time.Time) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE presidencias SET impeachment_em = ? WHERE chat = ? AND pessoa = ? AND mandato_ate > ?",
		now.Unix(), chat.String(), president.Key, now.Unix(),
	)
	return err
}

func (s *walletStore) impeach(ctx context.Context, chat types.JID, president core.Account, now time.Time) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		"UPDATE presidencias SET mandato_ate = ? WHERE chat = ? AND pessoa = ? AND mandato_ate > ?",
		now.Unix(), chat.String(), president.Key, now.Unix(),
	)
	if err != nil {
		return false, err
	}
	removed, err := res.RowsAffected()
	return removed > 0, err
}
