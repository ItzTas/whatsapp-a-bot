package economy

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"golang.org/x/sync/errgroup"
)

const crimeSector = "crime"

var syndicates = []firm{
	{id: "gangue-da-esquina", sector: crimeSector, name: "Gangue da esquina", emoji: "🔫", edge: 0.08, breakout: 1_000, breakoutOdds: 0.6, stash: 60, rate: 2, floor: 500, aliases: []string{"gangue", "esquina"}},
	{id: "firma-do-157", sector: crimeSector, name: "Firma do 157", emoji: "💼", edge: 0.12, gangs: true, breakout: 2_500, breakoutOdds: 0.75, stash: 45, rate: 4, floor: 1_500, aliases: []string{"firma", "157"}},
	{id: "mafia-do-porto", sector: crimeSector, name: "Máfia do porto", emoji: "🎩", edge: 0.18, gangs: true, breakout: 6_000, breakoutOdds: 0.9, stash: 30, rate: 7, floor: 4_000, aliases: []string{"mafia", "porto"}},
}

func (f firm) backing() string {
	reach := "no !roubar"
	if f.gangs {
		reach = "no !roubar, no !assalto e na !quadrilha"
	}
	return fmt.Sprintf("soma %d%% de chance %s e, se você rodar, oferece te tirar da cadeia por %s, com %d%% de chance de dar certo, e devolver o que a polícia apreender de ilegal por %d%% do valor",
		percent(f.edge), reach, Coins(f.breakout), percent(f.breakoutOdds), f.stash)
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
	chat       types.JID
	acc        core.Account
	contraband []stack
}

func (l *ledgerTx) confiscated(chat types.JID, acc core.Account, seized []stack) {
	i := slices.IndexFunc(l.jailed, func(j jailing) bool { return j.chat == chat && j.acc.Key == acc.Key })
	if i < 0 {
		return
	}
	l.jailed[i].contraband = append(l.jailed[i].contraband, seized...)
}

type jailLog struct {
	mu     sync.Mutex
	jailed map[types.JID]map[string]jailing
}

func (l *jailLog) add(jailings []jailing) {
	if len(jailings) == 0 {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.jailed == nil {
		l.jailed = map[types.JID]map[string]jailing{}
	}
	for _, j := range jailings {
		if l.jailed[j.chat] == nil {
			l.jailed[j.chat] = map[string]jailing{}
		}
		j.contraband = append(l.jailed[j.chat][j.acc.Key].contraband, j.contraband...)
		l.jailed[j.chat][j.acc.Key] = j
	}
}

func (l *jailLog) take(chat types.JID) map[string]jailing {
	l.mu.Lock()
	defer l.mu.Unlock()
	jailed := l.jailed[chat]
	delete(l.jailed, chat)
	return jailed
}

type breakoutOffer struct {
	chat       types.JID
	acc        core.Account
	firm       firm
	escape     bool
	contraband []stack
	expires    time.Time
}

func (o *breakoutOffer) expiresAt() time.Time { return o.expires }

func (o *breakoutOffer) stashPrice() int64 {
	var worth int64
	for _, s := range o.contraband {
		worth += s.worth()
	}
	return worth * o.firm.stash / 100
}

func (b *bot) offerBreakouts(ctx context.Context, chat types.JID) {
	for _, j := range b.Wallets.jailings.take(chat) {
		if err := b.offerBreakout(ctx, chat, j); err != nil {
			b.Client.Log.Errorf("oferecer resgate pra %s: %v", j.acc.JID, err)
		}
	}
}

func (b *bot) offerBreakout(ctx context.Context, chat types.JID, j jailing) error {
	now := time.Now()
	var (
		left   time.Duration
		backer firm
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { left, err = b.Wallets.SentenceLeft(gctx, chat, j.acc, now); return err })
	g.Go(func() (err error) { backer, err = syndicateOf(gctx, b.Wallets.db, chat, j.acc, now); return err })
	if err := g.Wait(); err != nil {
		return err
	}
	if backer.id == "" || (left == 0 && len(j.contraband) == 0) {
		return nil
	}
	o := &breakoutOffer{chat: chat, acc: j.acc, firm: backer, escape: left > 0, contraband: j.contraband, expires: now.Add(offerWindow)}
	b.breakoutOffers.Store(OfferKey(chat, j.acc), o)
	_, err := b.SendMentions(ctx, chat, o.pitch(), []types.JID{j.acc.JID})
	return err
}

func (o *breakoutOffer) pitch() string {
	head := fmt.Sprintf("%s @%s, a *%s* soube que você rodou", o.firm.emoji, o.acc.JID.User, o.firm.name)
	wait := core.FormatWait(offerWindow)
	if len(o.contraband) == 0 {
		return fmt.Sprintf("%s e pode te tirar da cadeia agora: %d%% de chance, por %s. Manda !aceitar em %s",
			head, percent(o.firm.breakoutOdds), Coins(o.firm.breakout), wait)
	}
	stash := fmt.Sprintf("sumir com as provas e devolver %s pra tua mochila, por %s", stackLabels(o.contraband), Coins(o.stashPrice()))
	if !o.escape {
		return fmt.Sprintf("%s e pode %s. Manda !aceitar em %s", head, stash, wait)
	}
	return fmt.Sprintf("%s e tem duas propostas:\n🚐 Te tirar da cadeia agora: %d%% de chance, por %s. Manda !aceitar fuga\n🎒 %s. Manda !aceitar muamba\nO !aceitar sozinho fecha as duas. Vale por %s",
		head, percent(o.firm.breakoutOdds), Coins(o.firm.breakout), strings.ToUpper(stash[:1])+stash[1:], wait)
}

func (o *breakoutOffer) picked(pick string) (escape, stash bool) {
	parts := map[string]struct{ escape, stash bool }{
		"fuga": {escape: true}, "resgate": {escape: true}, "cadeia": {escape: true},
		"muamba": {stash: true}, "itens": {stash: true}, "provas": {stash: true},
	}
	want, named := parts[core.FoldLetters(pick)]
	if !named {
		want.escape, want.stash = true, true
	}
	return want.escape && o.escape, want.stash && len(o.contraband) > 0
}

func (o *breakoutOffer) without(escape, stash bool) *breakoutOffer {
	rest := *o
	rest.escape = o.escape && !escape
	if stash {
		rest.contraband = nil
	}
	return &rest
}

func (o *breakoutOffer) open() bool {
	return o.escape || len(o.contraband) > 0
}

func (b *bot) acceptBreakout(ctx context.Context, key string, o *breakoutOffer, pick string) error {
	escape, stash := o.picked(pick)
	if !escape && !stash {
		b.breakoutOffers.Store(key, o)
		offered := map[bool]string{true: "a fuga", false: "sumir com as provas"}[o.escape]
		return b.SendText(ctx, o.chat, fmt.Sprintf("🤷 A %s só ofereceu %s. Manda !aceitar", o.firm.label(), offered))
	}
	if rest := o.without(escape, stash); rest.open() {
		b.breakoutOffers.Store(key, rest)
	}
	var lines []string
	if stash {
		line, err := b.reclaimLine(ctx, o)
		if err != nil {
			return err
		}
		lines = append(lines, line)
	}
	if escape {
		line, err := b.escapeLine(ctx, o)
		if err != nil {
			return err
		}
		lines = append(lines, line)
	}
	_, err := b.SendMentions(ctx, o.chat, strings.Join(lines, "\n"), []types.JID{o.acc.JID})
	return err
}

func (b *bot) reclaimLine(ctx context.Context, o *breakoutOffer) (string, error) {
	err := b.Wallets.reclaim(ctx, o)
	if errors.Is(err, ErrNoFunds) {
		return fmt.Sprintf("Sumir com as provas sai %s e você não tem tudo isso. Confere no !saldo", Coins(o.stashPrice())), nil
	}
	if err != nil {
		return "", fmt.Errorf("devolver a muamba: %w", err)
	}
	return fmt.Sprintf("🎒 A %s sumiu com as provas da delegacia e devolveu %s pra mochila de @%s por %s",
		o.firm.label(), stackLabels(o.contraband), o.acc.JID.User, Coins(o.stashPrice())), nil
}

func (b *bot) escapeLine(ctx context.Context, o *breakoutOffer) (string, error) {
	freed, err := b.Wallets.breakout(ctx, o, time.Now())
	if errors.Is(err, ErrNoFunds) {
		return fmt.Sprintf("O resgate sai %s e você não tem tudo isso. Confere no !saldo", Coins(o.firm.breakout)), nil
	}
	if errors.Is(err, errNotHeld) {
		return fmt.Sprintf("🤨 @%s já saiu da cadeia, a %s nem precisou ir", o.acc.JID.User, o.firm.label()), nil
	}
	if err != nil {
		return "", fmt.Errorf("aceitar resgate: %w", err)
	}
	return o.text(freed), nil
}

func (s *walletStore) reclaim(ctx context.Context, o *breakoutOffer) error {
	return s.spend(ctx, o.chat, o.acc, o.stashPrice(), func(tx *ledgerTx) error {
		for _, st := range o.contraband {
			if err := addGrades(ctx, tx, o.chat, o.acc, st.item.id, st.grades); err != nil {
				return err
			}
		}
		return nil
	})
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
