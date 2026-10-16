package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	proposalWindow   = 5 * time.Minute
	proposalKind     = "pedido"
	weddingPartyCost = 500
)

type romanceProposal struct {
	mu      sync.Mutex
	chat    types.JID
	kind    relationshipKind
	suitor  account
	beloved account
	dowry   int64
	expires time.Time
}

type romanceProposalState struct {
	Chat    types.JID        `json:"chat"`
	Kind    relationshipKind `json:"kind"`
	Suitor  storedAccount    `json:"suitor"`
	Beloved storedAccount    `json:"beloved"`
	Dowry   int64            `json:"dowry"`
	Expires time.Time        `json:"expires"`
}

type proposalWords struct {
	command  string
	noun     string
	ask      string
	together string
	gift     string
	emoji    string
}

var proposalTerms = map[relationshipKind]proposalWords{
	kindMarriage: {"!casar", "casamento", "💍 @%s pediu @%s em casamento!", "💍 @%s e @%s já são casados há %s", "dote", "💝"},
	kindDating:   {"!namorar", "namoro", "💘 @%s pediu @%s em namoro!", "💑 @%s e @%s já namoram há %s", "presente", "🎁"},
}

func (b *bot) propose(ctx context.Context, evt *events.Message, kind relationshipKind) error {
	chat := evt.Info.Chat.ToNonAD()
	terms := proposalTerms[kind]
	mentions := uniqueJIDs(mentionedJIDs(evt.Message))
	if len(mentions) != 1 {
		return b.sendText(ctx, chat, fmt.Sprintf("Com o consentimento ligado, cada um faz o próprio pedido: %s @pessoa, e a pessoa responde com !aceitar ou !recusar", terms.command))
	}
	suitor, beloved := evt.Info.Sender.ToNonAD(), mentions[0].ToNonAD()
	suitorAcc := b.account(ctx, suitor)

	var (
		all       []relationship
		adoptions []adoption
		aliases   [2][]string
		economy   bool
		funds     int64
	)
	couple := [2]types.JID{suitor, beloved}
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { all, err = b.relationships.inChat(gctx, chat, kind); return err })
	g.Go(func() (err error) { adoptions, err = b.adoptions.inChat(gctx, chat); return err })
	g.Go(func() (err error) { funds, _, err = b.wallets.balance(gctx, chat, suitorAcc); return err })
	g.Go(func() error { economy = b.economyOn(gctx, chat); return nil })
	for i, p := range couple {
		g.Go(func() error { aliases[i] = b.userAliases(gctx, p); return nil })
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler condições do pedido: %w", err)
	}

	if slices.ContainsFunc(aliases[0], func(a string) bool { return slices.Contains(aliases[1], a) }) {
		return b.sendText(ctx, chat, "Pedir a si mesmo(a) em "+terms.noun+" não vale 🪞 Marca outra pessoa")
	}
	if child, parent, ok := adoptiveBond(adoptions, couple, aliases); ok {
		_, err := b.sendMentions(ctx, chat, adoptiveBondText(child, parent, terms.command[1:]), couple[:])
		return err
	}
	together := func(r relationship) bool {
		return r.joins(aliases[0], aliases[1]) && (r.consented || kind == kindDating)
	}
	if i := slices.IndexFunc(all, together); i >= 0 {
		text := fmt.Sprintf(terms.together, suitor.User, beloved.User, marriageLength(time.Since(all[i].since)))
		_, err := b.sendMentions(ctx, chat, text, couple[:])
		return err
	}
	dowry := proposalDowry(plainArgs(messageText(evt.Message)), funds, economy)
	if dowry > funds {
		text := fmt.Sprintf("@%s só tem %s, não dá pra oferecer um %s de %s 🪹", suitor.User, coins(funds), terms.gift, coins(dowry))
		_, err := b.sendMentions(ctx, chat, text, []types.JID{suitor})
		return err
	}

	p := &romanceProposal{chat: chat, kind: kind, suitor: suitorAcc, beloved: b.account(ctx, beloved), dowry: dowry, expires: time.Now().Add(proposalWindow)}
	key := offerKey(chat, p.beloved)
	if !b.storeProposal(ctx, key, p) {
		text := fmt.Sprintf("@%s já tem um pedido esperando resposta. Espera aceitar, recusar ou passar o tempo dele ⏳", beloved.User)
		_, err := b.sendMentions(ctx, chat, text, []types.JID{beloved})
		return err
	}
	time.AfterFunc(proposalWindow, func() { b.expireProposal(ctx, key, p) })
	_, err := b.sendMentions(ctx, chat, proposalText(p, economy), couple[:])
	return err
}

func proposalDowry(args []string, funds int64, economy bool) int64 {
	if !economy || len(args) == 0 {
		return 0
	}
	dowry, ok := parseStake(args[0], funds)
	if !ok {
		return 0
	}
	return dowry
}

func proposalText(p *romanceProposal, economy bool) string {
	terms := proposalTerms[p.kind]
	suitor, beloved := p.suitor.jid.User, p.beloved.jid.User
	lines := []string{fmt.Sprintf(terms.ask, suitor, beloved)}
	if p.dowry > 0 {
		lines = append(lines, fmt.Sprintf("%s No pedido vem um %s de %s, que vai pra @%s se aceitar", terms.emoji, terms.gift, coins(p.dowry), beloved))
	}
	if economy && p.kind == kindMarriage {
		lines = append(lines,
			fmt.Sprintf("🎉 A festa custa %s e quem pediu paga. Sem moedas, o casamento sai no cartório, de graça", coins(weddingPartyCost)),
			"⚖️ Casando assim, quem pedir divórcio depois paga pensão, e o casal ganha uma conta conjunta (!conjunta), dividida meio a meio se separar")
	}
	lines = append(lines, "", fmt.Sprintf("@%s, manda !aceitar pra dizer sim ou !recusar pra dizer não. O pedido vale por %s", beloved, formatWait(proposalWindow)))
	return strings.Join(lines, "\n")
}

func (b *bot) payProposal(ctx context.Context, p *romanceProposal) ([]string, bool, error) {
	if p == nil || !b.economyOn(ctx, p.chat) {
		return nil, true, nil
	}
	terms := proposalTerms[p.kind]
	suitor, beloved := p.suitor.jid.User, p.beloved.jid.User
	var lines []string
	if p.dowry > 0 {
		err := b.wallets.transfer(ctx, p.chat, p.suitor, p.beloved, p.dowry)
		if errors.Is(err, errNoFunds) {
			return []string{fmt.Sprintf("❌ @%s não tem mais as moedas do %s, então o pedido caiu", suitor, terms.gift)}, false, nil
		}
		if err != nil {
			return nil, false, fmt.Errorf("pagar %s do pedido: %w", terms.gift, err)
		}
		lines = append(lines, fmt.Sprintf("%s @%s deu um %s de %s pra @%s", terms.emoji, suitor, terms.gift, coins(p.dowry), beloved))
	}
	if p.kind != kindMarriage {
		return lines, true, nil
	}
	_, err := b.wallets.settle(ctx, p.chat, p.suitor, weddingPartyCost, 0)
	if errors.Is(err, errNoFunds) {
		return append(lines, fmt.Sprintf("💒 Casaram no cartório, sem festa: @%s não tinha %s pra pagar", suitor, coins(weddingPartyCost))), true, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("pagar festa do casamento: %w", err)
	}
	return append(lines, fmt.Sprintf("🎉 @%s pagou %s da festa", suitor, coins(weddingPartyCost))), true, nil
}

func (b *bot) answerProposal(ctx context.Context, evt *events.Message, name string) bool {
	answers := map[string]func(context.Context, *events.Message) (bool, error){
		"aceitar": b.acceptProposal,
		"aceito":  b.acceptProposal,
		"recusar": b.refuseProposal,
		"recuso":  b.refuseProposal,
	}
	answer, ok := answers[foldLetters(name)]
	if !ok {
		return false
	}
	handled, err := answer(ctx, evt)
	if err != nil {
		b.client.Log.Errorf("responder pedido com !%s: %v", name, err)
	}
	return handled
}

func (b *bot) acceptProposal(ctx context.Context, evt *events.Message) (bool, error) {
	key := offerKey(evt.Info.Chat.ToNonAD(), b.account(ctx, evt.Info.Sender))
	value, ok := b.romanceProposals.LoadAndDelete(key)
	if !ok {
		return false, nil
	}
	p := value.(*romanceProposal)
	b.forgetProposal(ctx, key, p)
	unions := map[relationshipKind]func(context.Context, types.JID, [2]types.JID, *romanceProposal) error{
		kindMarriage: b.wed,
		kindDating:   b.date,
	}
	return true, unions[p.kind](ctx, p.chat, [2]types.JID{p.suitor.jid, p.beloved.jid}, p)
}

func (b *bot) refuseProposal(ctx context.Context, evt *events.Message) (bool, error) {
	chat := evt.Info.Chat.ToNonAD()
	acc := b.account(ctx, evt.Info.Sender)
	key := offerKey(chat, acc)
	if value, ok := b.romanceProposals.LoadAndDelete(key); ok {
		p := value.(*romanceProposal)
		b.forgetProposal(ctx, key, p)
		text := fmt.Sprintf("💔 @%s recusou o pedido de %s de @%s", p.beloved.jid.User, proposalTerms[p.kind].noun, p.suitor.jid.User)
		_, err := b.sendMentions(ctx, chat, text, []types.JID{p.beloved.jid, p.suitor.jid})
		return true, err
	}
	beloved := b.cancelProposals(ctx, chat, acc)
	if len(beloved) == 0 {
		return false, nil
	}
	text := fmt.Sprintf("🚫 @%s desistiu do pedido pra %s", acc.jid.User, joinMentions(beloved))
	_, err := b.sendMentions(ctx, chat, text, append(beloved, acc.jid))
	return true, err
}

func (b *bot) cancelProposals(ctx context.Context, chat types.JID, suitor account) []types.JID {
	var beloved []types.JID
	b.romanceProposals.Range(func(key, value any) bool {
		p := value.(*romanceProposal)
		if p.chat == chat && p.suitor.key == suitor.key && b.romanceProposals.CompareAndDelete(key, value) {
			b.forgetProposal(ctx, key.(string), p)
			beloved = append(beloved, p.beloved.jid)
		}
		return true
	})
	return beloved
}

func (b *bot) storeProposal(ctx context.Context, key string, p *romanceProposal) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, pending := b.romanceProposals.LoadOrStore(key, p); pending {
		return false
	}
	state := romanceProposalState{
		Chat: p.chat, Kind: p.kind, Suitor: storeAccount(p.suitor), Beloved: storeAccount(p.beloved),
		Dowry: p.dowry, Expires: p.expires,
	}
	b.keepGame(ctx, proposalKind, key, state, p.expires)
	return true
}

func (b *bot) forgetProposal(ctx context.Context, key string, p *romanceProposal) {
	p.mu.Lock()
	defer p.mu.Unlock()
	b.dropGame(ctx, proposalKind, key)
}

func (b *bot) expireProposal(ctx context.Context, key string, p *romanceProposal) {
	if !b.romanceProposals.CompareAndDelete(key, p) {
		return
	}
	b.forgetProposal(ctx, key, p)
	text := fmt.Sprintf("⌛ @%s não respondeu e o pedido de %s de @%s expirou", p.beloved.jid.User, proposalTerms[p.kind].noun, p.suitor.jid.User)
	if _, err := b.sendMentions(ctx, p.chat, text, []types.JID{p.beloved.jid, p.suitor.jid}); err != nil {
		b.client.Log.Errorf("avisar que o pedido de %s expirou: %v", proposalTerms[p.kind].noun, err)
	}
}

func (b *bot) restoreProposal(ctx context.Context, saved savedGame) error {
	var state romanceProposalState
	if err := json.Unmarshal(saved.state, &state); err != nil {
		return fmt.Errorf("ler pedido salvo: %w", err)
	}
	if _, ok := proposalTerms[state.Kind]; !ok {
		return fmt.Errorf("pedido de %q não existe", state.Kind)
	}
	p := &romanceProposal{
		chat: state.Chat, kind: state.Kind, suitor: state.Suitor.account(), beloved: state.Beloved.account(),
		dowry: state.Dowry, expires: state.Expires,
	}
	if _, pending := b.romanceProposals.LoadOrStore(saved.key, p); pending {
		return fmt.Errorf("já tem outro pedido pra %s", saved.key)
	}
	time.AfterFunc(time.Until(p.expires), func() { b.expireProposal(ctx, saved.key, p) })
	return nil
}

func paragraph(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return "\n\n" + strings.Join(lines, "\n")
}
