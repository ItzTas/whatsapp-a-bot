package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	auctionWindow      = 2 * time.Minute
	auctionExtension   = 30 * time.Second
	auctionMinStep     = 10
	auctionStepPercent = 5
	auctionKind        = "leilao"
)

type auction struct {
	mu       sync.Mutex
	chat     types.JID
	seller   account
	item     item
	quantity int64
	grades   []grade
	minimum  int64
	bid      int64
	bidder   account
	ends     time.Time
	over     bool
}

type auctionState struct {
	Chat     types.JID     `json:"chat"`
	Seller   storedAccount `json:"seller"`
	Item     string        `json:"item"`
	Quantity int64         `json:"quantity"`
	Grades   []gradeState  `json:"grades,omitempty"`
	Minimum  int64         `json:"minimum"`
	Bid      int64         `json:"bid"`
	Bidder   storedAccount `json:"bidder"`
	Ends     time.Time     `json:"ends"`
}

type auctionBid struct {
	previous account
	refunded int64
	extended bool
	next     int64
}

func (b *bot) leilao(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	text := messageText(evt.Message)
	arg := firstPlainArg(text)
	value, open := b.auctions.Load(chat.String())
	if open && arg != "" {
		return b.bidAuction(ctx, evt, value.(*auction), arg)
	}
	if open {
		return b.sendAuctionStatus(ctx, value.(*auction))
	}
	q := parseItemQuery(text)
	if q.name == "" {
		return b.sendText(ctx, chat, fmt.Sprintf("Não tem leilão rolando agora 🔨 Abre um com !leilao <item> [quantidade] [lance mínimo], tipo !leilao espada 2 500. Ele fica aberto por %s", formatWait(auctionWindow)))
	}
	return b.openAuction(ctx, evt, q)
}

func (b *bot) lance(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	value, open := b.auctions.Load(chat.String())
	if !open {
		return b.sendText(ctx, chat, "Não tem leilão rolando agora 🔨 Abre um com !leilao <item>")
	}
	return b.bidAuction(ctx, evt, value.(*auction), firstPlainArg(messageText(evt.Message)))
}

func (b *bot) openAuction(ctx context.Context, evt *events.Message, q itemQuery) error {
	chat := evt.Info.Chat.ToNonAD()
	seller := b.account(ctx, evt.Info.Sender)
	it, have, problem, err := b.heldItem(ctx, chat, seller, q.name)
	if err != nil {
		return err
	}
	if problem != "" {
		return b.sendText(ctx, chat, problem)
	}
	quantity, minimum, priced := auctionTerms(q, have)
	if quantity > have {
		return b.sendText(ctx, chat, "Você só tem "+quantityLabel(it, have))
	}
	grades, err := b.wallets.takeAuctionItem(ctx, chat, seller, it.id, quantity)
	if errors.Is(err, errNoItems) {
		return b.sendText(ctx, chat, "Sua mochila mudou no meio do caminho e não dá mais 🤨 Confere no !mochila")
	}
	if err != nil {
		return fmt.Errorf("guardar item do leilão: %w", err)
	}

	if !priced {
		minimum = marketPayoutOf(it, grades, marketSalePercent)
	}
	a := &auction{chat: chat, seller: seller, item: it, quantity: quantity, grades: grades, minimum: minimum, ends: time.Now().Add(auctionWindow)}
	if !b.storeAuction(ctx, a) {
		if err := b.wallets.putBackAuctionItem(ctx, chat, seller, it.id, grades); err != nil {
			return fmt.Errorf("devolver item do leilão: %w", err)
		}
		return b.sendText(ctx, chat, "Alguém abriu um leilão aqui agora mesmo 🔨 Espera ele acabar pra abrir o seu")
	}
	go b.runAuction(ctx, a)

	text := fmt.Sprintf("🔨 *Leilão aberto!* @%s tá leiloando %s, que vale %s\n💰 Lance mínimo: %s\n⏰ Fecha em %s, e lance no finzinho prorroga por mais %s\n\n_Manda !lance <valor> pra dar lance_",
		seller.jid.User, a.lot(), coins(worthOf(it, grades)), coins(minimum), formatWait(auctionWindow), formatSeconds(auctionExtension))
	_, err = b.sendMentions(ctx, chat, text, []types.JID{seller.jid})
	return err
}

func (b *bot) bidAuction(ctx context.Context, evt *events.Message, a *auction, arg string) error {
	chat := evt.Info.Chat.ToNonAD()
	bidder := b.account(ctx, evt.Info.Sender)
	amount, refusal, err := b.placeStake(ctx, chat, bidder, arg, "!lance")
	if err != nil {
		return err
	}
	if refusal != "" {
		return b.sendText(ctx, chat, refusal)
	}
	bid, refusal, err := b.raiseAuction(ctx, a, bidder, amount, time.Now())
	if err != nil {
		return err
	}
	if refusal != "" {
		return b.sendText(ctx, chat, refusal)
	}

	lines := []string{fmt.Sprintf("💸 @%s deu um lance de %s em %s", bidder.jid.User, coins(amount), a.lot())}
	mentions := []types.JID{bidder.jid}
	if bid.refunded > 0 {
		lines = append(lines, fmt.Sprintf("↩️ @%s foi coberto e recebeu os %s de volta", bid.previous.jid.User, thousands(bid.refunded)))
		mentions = append(mentions, bid.previous.jid)
	}
	if bid.extended {
		lines = append(lines, fmt.Sprintf("⏰ Lance no finzinho! O leilão foi prorrogado e fecha em %s", formatSeconds(auctionExtension)))
	}
	lines = append(lines, "", fmt.Sprintf("_Pra cobrir: !lance %s ou mais_", thousands(bid.next)))
	_, err = b.sendMentions(ctx, chat, strings.Join(lines, "\n"), mentions)
	return err
}

func (b *bot) raiseAuction(ctx context.Context, a *auction, bidder account, amount int64, now time.Time) (auctionBid, string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.over {
		return auctionBid{}, "🔨 Esse leilão acabou de fechar", nil
	}
	if bidder.key == a.seller.key {
		return auctionBid{}, "Dar lance no próprio leilão? Isso se chama inflar o preço 🤡", nil
	}
	if bidder.key == a.bidder.key {
		return auctionBid{}, fmt.Sprintf("Você já tá ganhando com %s 🏆 Espera alguém cobrir", coins(a.bid)), nil
	}
	if next := a.nextBid(); amount < next {
		return auctionBid{}, fmt.Sprintf("O lance tem que ser de pelo menos %s", coins(next)), nil
	}
	held, err := b.holdStake(ctx, a.chat, bidder, amount)
	if err != nil {
		return auctionBid{}, "", fmt.Errorf("segurar lance do leilão: %w", err)
	}
	if !held {
		return auctionBid{}, stakeChanged, nil
	}

	bid := auctionBid{previous: a.bidder, refunded: a.bid}
	if a.bid > 0 {
		if err := b.wallets.credit(ctx, a.chat, []account{a.bidder}, a.bid); err != nil {
			b.client.Log.Errorf("devolver %d moedas do lance coberto de %s no leilão: %v", a.bid, a.bidder.jid, err)
		}
	}
	a.bid, a.bidder = amount, bidder
	if a.ends.Sub(now) < auctionExtension {
		a.ends = now.Add(auctionExtension)
		bid.extended = true
	}
	bid.next = a.nextBid()
	b.keepAuction(ctx, a)
	return bid, "", nil
}

func (b *bot) storeAuction(ctx context.Context, a *auction) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, taken := b.auctions.LoadOrStore(a.chat.String(), a); taken {
		return false
	}
	b.keepAuction(ctx, a)
	return true
}

func (b *bot) keepAuction(ctx context.Context, a *auction) {
	state := auctionState{
		Chat: a.chat, Seller: storeAccount(a.seller), Item: a.item.id, Quantity: a.quantity, Grades: storeGrades(a.grades),
		Minimum: a.minimum, Bid: a.bid, Bidder: storeAccount(a.bidder), Ends: a.ends,
	}
	b.keepGame(ctx, auctionKind, a.chat.String(), state, a.ends)
}

func (a *auction) nextBid() int64 {
	if a.bid == 0 {
		return a.minimum
	}
	return a.bid + max(auctionMinStep, a.bid*auctionStepPercent/100)
}

func (a *auction) closeIfDue(now time.Time) time.Duration {
	a.mu.Lock()
	defer a.mu.Unlock()
	wait := a.ends.Sub(now)
	if wait <= 0 {
		a.over = true
	}
	return wait
}

func (b *bot) runAuction(ctx context.Context, a *auction) {
	for wait := a.closeIfDue(time.Now()); wait > 0; wait = a.closeIfDue(time.Now()) {
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
	b.auctions.CompareAndDelete(a.chat.String(), a)
	if err := b.finishAuction(ctx, a); err != nil {
		b.client.Log.Errorf("fechar leilão de %s: %v", a.chat, err)
	}
}

func (b *bot) finishAuction(ctx context.Context, a *auction) error {
	lot := a.lot()
	if a.bidder.key == "" {
		if err := b.wallets.returnAuctionItem(ctx, a); err != nil {
			return fmt.Errorf("devolver item sem lance: %w", err)
		}
		text := fmt.Sprintf("🔨 O leilão de %s acabou sem nenhum lance, então voltou pra mochila de @%s", lot, a.seller.jid.User)
		_, err := b.sendMentions(ctx, a.chat, text, []types.JID{a.seller.jid})
		return err
	}
	if err := b.wallets.settleAuction(ctx, a); err != nil {
		return fmt.Errorf("pagar leilão: %w", err)
	}
	text := fmt.Sprintf("🔨 *Dou-lhe uma, dou-lhe duas... vendido!*\n\n🏆 @%s arrematou %s de @%s por %s",
		a.bidder.jid.User, lot, a.seller.jid.User, coins(a.bid))
	_, err := b.sendMentions(ctx, a.chat, text, []types.JID{a.bidder.jid, a.seller.jid})
	return err
}

func (b *bot) sendAuctionStatus(ctx context.Context, a *auction) error {
	a.mu.Lock()
	seller, bidder, bid, next, left := a.seller, a.bidder, a.bid, a.nextBid(), time.Until(a.ends)
	a.mu.Unlock()

	lines := []string{
		fmt.Sprintf("🔨 *Leilão de @%s*", seller.jid.User),
		fmt.Sprintf("📦 %s, que vale %s", a.lot(), coins(worthOf(a.item, a.grades))),
	}
	price, mentions := "💰 Ainda sem lance, o mínimo é "+coins(next), []types.JID{seller.jid}
	if bid > 0 {
		price = fmt.Sprintf("💰 Lance atual: %s de @%s", coins(bid), bidder.jid.User)
		mentions = append(mentions, bidder.jid)
	}
	lines = append(lines, price, "⏰ Fecha em "+formatSeconds(left), "", fmt.Sprintf("_Manda !lance %s ou mais pra entrar_", thousands(next)))
	_, err := b.sendMentions(ctx, a.chat, strings.Join(lines, "\n"), mentions)
	return err
}

func auctionTerms(q itemQuery, have int64) (quantity, minimum int64, priced bool) {
	if len(q.numbers) == 0 || (len(q.numbers) == 1 && !q.all && q.count == 0) {
		return q.quantity(have), 0, false
	}
	minimum = q.numbers[len(q.numbers)-1]
	q.numbers = q.numbers[:len(q.numbers)-1]
	return q.quantity(have), minimum, true
}

func (a *auction) lot() string {
	return quantityLabel(a.item, a.quantity) + gradesLabel(a.grades)
}

func (s *walletStore) takeAuctionItem(ctx context.Context, chat types.JID, acc account, id string, quantity int64) (grades []grade, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) (err error) {
		grades, err = takeGrades(ctx, tx, chat, acc, id, quantity)
		return err
	})
	return grades, err
}

func (s *walletStore) putBackAuctionItem(ctx context.Context, chat types.JID, acc account, id string, grades []grade) error {
	return s.inTx(ctx, func(tx *sql.Tx) error { return addGrades(ctx, tx, chat, acc, id, grades) })
}

func (s *walletStore) returnAuctionItem(ctx context.Context, a *auction) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if err := addGrades(ctx, tx, a.chat, a.seller, a.item.id, a.grades); err != nil {
			return err
		}
		return dropGameTx(ctx, tx, auctionKind, a.chat.String())
	})
}

func (s *walletStore) settleAuction(ctx context.Context, a *auction) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if err := dropGameTx(ctx, tx, auctionKind, a.chat.String()); err != nil {
			return err
		}
		if err := addGrades(ctx, tx, a.chat, a.bidder, a.item.id, a.grades); err != nil {
			return err
		}
		if err := openWallet(ctx, tx, a.chat, a.seller); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", a.bid, a.chat.String(), a.seller.key)
		return err
	})
}

func (b *bot) restoreAuction(ctx context.Context, saved savedGame) error {
	var state auctionState
	if err := json.Unmarshal(saved.state, &state); err != nil {
		return fmt.Errorf("ler leilão salvo: %w", err)
	}
	it, ok := itemByID(state.Item)
	if !ok {
		return fmt.Errorf("item %q do leilão não existe mais", state.Item)
	}
	a := &auction{
		chat: state.Chat, seller: state.Seller.account(), item: it, quantity: state.Quantity, grades: restoreGrades(state.Grades, state.Quantity),
		minimum: state.Minimum, bid: state.Bid, bidder: state.Bidder.account(), ends: state.Ends,
	}
	if _, taken := b.auctions.LoadOrStore(a.chat.String(), a); taken {
		return fmt.Errorf("já tem outro leilão rolando em %s", a.chat)
	}
	go b.runAuction(ctx, a)
	return nil
}
