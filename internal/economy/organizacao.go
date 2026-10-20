package economy

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"golang.org/x/sync/errgroup"
)

const crimeSector = "crime"

var syndicates = []firm{
	{id: "gangue-da-esquina", sector: crimeSector, name: "Gangue da esquina", emoji: "🔫", edge: 0.08, breakout: 1_000, breakoutOdds: 0.6, rate: 2, floor: 500, aliases: []string{"gangue", "esquina"}},
	{id: "firma-do-157", sector: crimeSector, name: "Firma do 157", emoji: "💼", edge: 0.12, gangs: true, breakout: 2_500, breakoutOdds: 0.75, rate: 4, floor: 1_500, aliases: []string{"firma", "157"}},
	{id: "mafia-do-porto", sector: crimeSector, name: "Máfia do porto", emoji: "🎩", edge: 0.18, gangs: true, breakout: 6_000, breakoutOdds: 0.9, rate: 7, floor: 4_000, aliases: []string{"mafia", "porto"}},
}

func (f firm) backing() string {
	reach := "no !roubar"
	if f.gangs {
		reach = "no !roubar, no !assalto e na !quadrilha"
	}
	return fmt.Sprintf("soma %d%% de chance %s e, se você rodar, oferece te tirar da cadeia por %s, com %d%% de chance de dar certo",
		percent(f.edge), reach, Coins(f.breakout), percent(f.breakoutOdds))
}

func syndicateOf(ctx context.Context, q rowQuerier, chat types.JID, acc core.Account, now time.Time) (firm, error) {
	c, found, err := contractIn(ctx, q, chat, acc, crimeSector, now)
	if err != nil || !found {
		return firm{}, err
	}
	return c.firm, nil
}

func crewSyndicate(backers []firm) firm {
	var best firm
	for _, f := range backers {
		if f.gangs && f.edge > best.edge {
			best = f
		}
	}
	return best
}

func (f firm) coverText(who string) string {
	if f.id == "" {
		return ""
	}
	return fmt.Sprintf("%s A *%s* deu cobertura pra %s (+%d%% de chance)\n", f.emoji, f.name, who, percent(f.edge))
}

type jailing struct {
	chat types.JID
	acc  core.Account
}

type jailLog struct {
	mu     sync.Mutex
	jailed map[types.JID]map[string]core.Account
}

func (l *jailLog) add(jailings []jailing) {
	if len(jailings) == 0 {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.jailed == nil {
		l.jailed = map[types.JID]map[string]core.Account{}
	}
	for _, j := range jailings {
		if l.jailed[j.chat] == nil {
			l.jailed[j.chat] = map[string]core.Account{}
		}
		l.jailed[j.chat][j.acc.Key] = j.acc
	}
}

func (l *jailLog) take(chat types.JID) map[string]core.Account {
	l.mu.Lock()
	defer l.mu.Unlock()
	jailed := l.jailed[chat]
	delete(l.jailed, chat)
	return jailed
}

type breakoutOffer struct {
	chat    types.JID
	acc     core.Account
	firm    firm
	expires time.Time
}

func (o *breakoutOffer) expiresAt() time.Time { return o.expires }

func (b *bot) offerBreakouts(ctx context.Context, chat types.JID) {
	for _, acc := range b.Wallets.jailings.take(chat) {
		if err := b.offerBreakout(ctx, chat, acc); err != nil {
			b.Client.Log.Errorf("oferecer resgate pra %s: %v", acc.JID, err)
		}
	}
}

func (b *bot) offerBreakout(ctx context.Context, chat types.JID, acc core.Account) error {
	now := time.Now()
	var (
		left   time.Duration
		backer firm
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { left, err = b.Wallets.SentenceLeft(gctx, chat, acc, now); return err })
	g.Go(func() (err error) { backer, err = syndicateOf(gctx, b.Wallets.db, chat, acc, now); return err })
	if err := g.Wait(); err != nil {
		return err
	}
	if backer.id == "" || left == 0 {
		return nil
	}
	b.breakoutOffers.Store(OfferKey(chat, acc), &breakoutOffer{chat: chat, acc: acc, firm: backer, expires: now.Add(offerWindow)})
	text := fmt.Sprintf("%s @%s, a *%s* soube que você rodou e pode te tirar da cadeia agora: %d%% de chance, por %s. Manda !aceitar em %s",
		backer.emoji, acc.JID.User, backer.name, percent(backer.breakoutOdds), Coins(backer.breakout), core.FormatWait(offerWindow))
	_, err := b.SendMentions(ctx, chat, text, []types.JID{acc.JID})
	return err
}

func (b *bot) acceptBreakout(ctx context.Context, o *breakoutOffer) error {
	freed, err := b.Wallets.breakout(ctx, o, time.Now())
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, o.chat, fmt.Sprintf("O resgate sai %s e você não tem tudo isso. Confere no !saldo", Coins(o.firm.breakout)))
	}
	if errors.Is(err, errNotHeld) {
		_, err := b.SendMentions(ctx, o.chat, fmt.Sprintf("🤨 @%s já saiu da cadeia, a %s nem precisou ir", o.acc.JID.User, o.firm.label()), []types.JID{o.acc.JID})
		return err
	}
	if err != nil {
		return fmt.Errorf("aceitar resgate: %w", err)
	}
	_, err = b.SendMentions(ctx, o.chat, o.text(freed), []types.JID{o.acc.JID})
	return err
}

func (o *breakoutOffer) text(freed bool) string {
	if !freed {
		return fmt.Sprintf("🚨 A %s tentou tirar @%s da cadeia, mas a polícia segurou o portão. Os %s não voltam", o.firm.label(), o.acc.JID.User, Coins(o.firm.breakout))
	}
	return fmt.Sprintf("🚐 A %s arrombou a cadeia e tirou @%s de lá por %s. Agora a polícia tá procurando de novo", o.firm.label(), o.acc.JID.User, Coins(o.firm.breakout))
}

func (s *walletStore) breakout(ctx context.Context, o *breakoutOffer, now time.Time) (freed bool, err error) {
	err = s.spend(ctx, o.chat, o.acc, o.firm.breakout, func(tx *ledgerTx) error {
		var until int64
		if err := tx.QueryRowContext(ctx, "SELECT preso_ate FROM carteiras WHERE chat = ? AND pessoa = ?", o.chat.String(), o.acc.Key).Scan(&until); err != nil {
			return err
		}
		if until <= now.Unix() {
			return errNotHeld
		}
		if rand.Float64() >= o.firm.breakoutOdds {
			return nil
		}
		freed = true
		_, err := tx.ExecContext(ctx,
			"UPDATE carteiras SET preso_ate = 0, crime_em = ?, ficha_em = ? WHERE chat = ? AND pessoa = ?",
			now.Unix(), now.Unix(), o.chat.String(), o.acc.Key,
		)
		return err
	})
	return freed && err == nil, err
}
