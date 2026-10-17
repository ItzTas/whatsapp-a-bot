package main

import (
	"context"
	"database/sql"
	"encoding/json"
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
	gangSize    = 3
	gangMin     = 2
	gangWindow  = 2 * time.Minute
	gangHelp    = 0.1
	gangLootMin = 20
	gangLootMax = 45
	gangKind    = "quadrilha"
)

type gang struct {
	mu      sync.Mutex
	opened  time.Time
	victim  account
	members []account
	closed  bool
}

type gangState struct {
	Chat    types.JID       `json:"chat"`
	Opened  time.Time       `json:"opened"`
	Victim  storedAccount   `json:"victim"`
	Members []storedAccount `json:"members"`
}

type gangster struct {
	acc  account
	job  profession
	paid int64
	took int64
}

type gangRobbery struct {
	victim    account
	gangsters []gangster
	chance    float64
	poor      bool
	alarmed   bool
	success   bool
}

func (b *bot) quadrilha(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	acc := b.account(ctx, evt.Info.Sender.ToNonAD())
	var target *account
	if mentions := mentionedJIDs(evt.Message); len(mentions) > 0 {
		victim := b.account(ctx, mentions[0].ToNonAD())
		target = &victim
	}
	if target != nil && target.key == acc.key {
		return b.sendText(ctx, chat, "Montar quadrilha pra roubar você mesmo? Isso se chama mudar o dinheiro de bolso 🤡")
	}
	refusal, err := b.robWait(ctx, chat, acc)
	if err != nil {
		return err
	}
	if refusal != "" {
		return b.sendMentionsText(ctx, chat, refusal, acc)
	}

	g, found := b.gangFor(ctx, chat, target)
	if !found {
		return b.sendText(ctx, chat, "Marca quem a quadrilha vai roubar: !quadrilha @pessoa")
	}
	members, refusal := b.joinGang(ctx, chat, g, acc, target)
	if refusal != "" {
		return b.sendMentionsText(ctx, chat, refusal, acc, g.victim)
	}
	if len(members) < gangSize {
		return b.sendMentionsText(ctx, chat, g.recruitText(acc, len(members)), acc, g.victim)
	}
	b.gangs.CompareAndDelete(chat.String(), g)
	return b.runGang(ctx, chat, g.victim, members)
}

func (b *bot) sendMentionsText(ctx context.Context, chat types.JID, text string, people ...account) error {
	if text == "" {
		return nil
	}
	_, err := b.sendMentions(ctx, chat, text, accountJIDs(people))
	return err
}

func (b *bot) robWait(ctx context.Context, chat types.JID, acc account) (string, error) {
	job, err := b.professionOf(ctx, chat, acc)
	if err != nil {
		return "", err
	}
	wait, err := b.wallets.cooldownLeft(ctx, chat, acc, job.robCooldown(), time.Now())
	if err != nil {
		return "", fmt.Errorf("quadrilha: %w", err)
	}
	if wait == 0 {
		return "", nil
	}
	return fmt.Sprintf("🚓 @%s ainda tá sendo procurado pela polícia. Espera %s pra roubar de novo", acc.jid.User, formatWait(wait)), nil
}

func (b *bot) gangFor(ctx context.Context, chat types.JID, target *account) (*gang, bool) {
	if target == nil {
		value, found := b.gangs.Load(chat.String())
		if !found {
			return nil, false
		}
		return value.(*gang), true
	}
	fresh := &gang{opened: time.Now(), victim: *target}
	value, loaded := b.gangs.LoadOrStore(chat.String(), fresh)
	if !loaded {
		go b.expireGang(ctx, chat, fresh)
	}
	return value.(*gang), true
}

func (b *bot) joinGang(ctx context.Context, chat types.JID, g *gang, acc account, target *account) ([]account, string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return nil, "🦹 A quadrilha acabou de partir pra cima. Manda !quadrilha @pessoa pra montar outra"
	}
	if target != nil && target.key != g.victim.key {
		return nil, fmt.Sprintf("Já tem uma quadrilha atrás de @%s. Manda !quadrilha pra entrar nela", g.victim.jid.User)
	}
	if acc.key == g.victim.key {
		return nil, "Essa quadrilha é pra te roubar, não dá pra entrar nela 🤡"
	}
	if slices.ContainsFunc(g.members, func(m account) bool { return m.key == acc.key }) {
		return nil, fmt.Sprintf("@%s já tá na quadrilha 🦹", acc.jid.User)
	}
	g.members = append(g.members, acc)
	g.closed = len(g.members) == gangSize
	b.saveGang(ctx, chat, g)
	return slices.Clone(g.members), ""
}

func (b *bot) abandonGang(ctx context.Context, chat types.JID, g *gang) ([]account, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return nil, false
	}
	g.closed = true
	b.dropGame(ctx, gangKind, chat.String())
	return g.members, true
}

func (b *bot) saveGang(ctx context.Context, chat types.JID, g *gang) {
	if g.closed {
		b.dropGame(ctx, gangKind, chat.String())
		return
	}
	members := make([]storedAccount, len(g.members))
	for i, m := range g.members {
		members[i] = storeAccount(m)
	}
	state := gangState{Chat: chat, Opened: g.opened, Victim: storeAccount(g.victim), Members: members}
	b.keepGame(ctx, gangKind, chat.String(), state, g.opened.Add(gangWindow))
}

func (b *bot) restoreGang(ctx context.Context, saved savedGame) error {
	var state gangState
	if err := json.Unmarshal(saved.state, &state); err != nil {
		return fmt.Errorf("ler quadrilha salva: %w", err)
	}
	members := make([]account, len(state.Members))
	for i, m := range state.Members {
		members[i] = m.account()
	}
	g := &gang{opened: state.Opened, victim: state.Victim.account(), members: members}
	if _, taken := b.gangs.LoadOrStore(state.Chat.String(), g); taken {
		return fmt.Errorf("já tem outra quadrilha sendo montada em %s", state.Chat)
	}
	go b.expireGang(ctx, state.Chat, g)
	return nil
}

func (g *gang) recruitText(acc account, size int) string {
	room := countLabel(gangSize-size, "vaga", "vagas")
	left := formatWait(gangWindow - time.Since(g.opened))
	if size > 1 {
		return fmt.Sprintf("🦹 @%s entrou na quadrilha contra @%s. Sobra %s: manda !quadrilha em até %s pra entrar. Se ninguém mais vier, a quadrilha parte pra cima quando o tempo acabar", acc.jid.User, g.victim.jid.User, room, left)
	}
	return fmt.Sprintf("🦹 *Quadrilha!*\n@%s tá juntando gente pra roubar @%s. Sobram %s: manda !quadrilha em até %s pra entrar\n_Com %d já dá pra ir, com %d parte na hora. Cada um a mais soma %d%% de chance, e se der errado cada um paga indenização pra vítima_",
		acc.jid.User, g.victim.jid.User, room, left, gangMin, gangSize, percent(gangHelp))
}

func (b *bot) expireGang(ctx context.Context, chat types.JID, g *gang) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Until(g.opened.Add(gangWindow))):
	}
	members, abandoned := b.abandonGang(ctx, chat, g)
	if !abandoned || len(members) == 0 {
		return
	}
	b.gangs.CompareAndDelete(chat.String(), g)
	if len(members) >= gangMin {
		if err := b.runGang(ctx, chat, g.victim, members); err != nil {
			b.client.Log.Errorf("rodar !quadrilha: %v", err)
		}
		return
	}
	text := fmt.Sprintf("🚫 A quadrilha contra @%s foi desfeita: só @%s apareceu. Precisa de pelo menos %d", g.victim.jid.User, members[0].jid.User, gangMin)
	if err := b.sendMentionsText(ctx, chat, text, members[0], g.victim); err != nil {
		b.client.Log.Errorf("desfazer !quadrilha: %v", err)
	}
}

func (b *bot) runGang(ctx context.Context, chat types.JID, victim account, members []account) error {
	gangsters := make([]gangster, len(members))
	var victimJob profession
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { victimJob, err = b.professionOf(gctx, chat, victim); return err })
	for i, acc := range members {
		gangsters[i].acc = acc
		g.Go(func() (err error) { gangsters[i].job, err = b.professionOf(gctx, chat, acc); return err })
	}
	if err := g.Wait(); err != nil {
		return err
	}
	chance := clampChance(robSuccessChance + gangBonus(gangsters) - victimJob.guard)
	r, err := b.wallets.gangRob(ctx, chat, victim, gangsters, chance, time.Now())
	if err != nil {
		return fmt.Errorf("quadrilha: %w", err)
	}
	return b.sendMentionsText(ctx, chat, r.text(), append(members, victim)...)
}

func gangBonus(gangsters []gangster) float64 {
	bonus := gangHelp * float64(len(gangsters)-1)
	var curfew float64
	for _, g := range gangsters {
		bonus += g.job.robBonus
		curfew = g.job.curfew()
	}
	return bonus - curfew
}

func (s *walletStore) gangRob(ctx context.Context, chat types.JID, victim account, gangsters []gangster, chance float64, now time.Time) (r gangRobbery, err error) {
	r = gangRobbery{victim: victim, gangsters: slices.Clone(gangsters), chance: chance}
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		if err := openWallet(ctx, tx, chat, victim); err != nil {
			return err
		}
		var victimBalance int64
		err := tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), victim.key).Scan(&victimBalance)
		if err != nil {
			return err
		}
		if victimBalance < robMinVictim {
			r.poor = true
			return nil
		}
		if err := stampGang(ctx, tx, chat, r.gangsters, now); err != nil {
			return err
		}
		err = takeItems(ctx, tx, chat, victim, alarmItemID, 1)
		if err == nil {
			r.alarmed = true
			return nil
		}
		if !errors.Is(err, errNoItems) {
			return err
		}
		if r.success = rand.Float64() < chance; r.success {
			share := victimBalance * (gangLootMin + rand.Int64N(gangLootMax-gangLootMin+1)) / 100 / int64(len(r.gangsters))
			return payGang(ctx, tx, chat, victim, r.gangsters, share)
		}
		return indemnify(ctx, tx, chat, victim, r.gangsters)
	})
	return r, err
}

func stampGang(ctx context.Context, tx *sql.Tx, chat types.JID, gangsters []gangster, now time.Time) error {
	for _, g := range gangsters {
		if err := openWallet(ctx, tx, chat, g.acc); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx,
			"UPDATE carteiras SET "+g.job.robCooldown().column+" = ?, crime_em = ? WHERE chat = ? AND pessoa = ?",
			now.Unix(), now.Unix(), chat.String(), g.acc.key,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func payGang(ctx context.Context, tx *sql.Tx, chat types.JID, victim account, gangsters []gangster, share int64) error {
	for i := range gangsters {
		g := &gangsters[i]
		g.took = share * (100 + g.job.robHaul) / 100
		if err := moveCoins(ctx, tx, chat, victim, g.acc, g.took); err != nil {
			return err
		}
	}
	return nil
}

func indemnify(ctx context.Context, tx *sql.Tx, chat types.JID, victim account, gangsters []gangster) error {
	for i := range gangsters {
		g := &gangsters[i]
		var balance int64
		err := tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), g.acc.key).Scan(&balance)
		if err != nil {
			return err
		}
		g.paid = balance * (10 + rand.Int64N(11)) / 100
		if err := moveCoins(ctx, tx, chat, g.acc, victim, g.paid); err != nil {
			return err
		}
	}
	return nil
}

func (r gangRobbery) text() string {
	if r.poor {
		return fmt.Sprintf("🪹 A quadrilha cercou @%s, mas a pessoa tem menos de %s. Nem valeu o risco, e ninguém ficou procurado", r.victim.jid.User, coins(robMinVictim))
	}
	lines := []string{"🦹 *Quadrilha!*", "👥 " + r.crewText(), fmt.Sprintf("🎯 Chance de dar certo: %d%%", percent(r.chance)), ""}
	if r.alarmed {
		return strings.Join(append(lines, fmt.Sprintf("🚨 A quadrilha cercou @%s, mas o alarme disparou! Saíram correndo de mãos vazias e o alarme queimou", r.victim.jid.User)), "\n")
	}
	if r.success {
		lines = append(lines, r.lootLines()...)
		return strings.Join(append(lines, fmt.Sprintf("🚓 Mas a polícia tá de olho: dá pra !prender quem tava na quadrilha nas próximas %s", formatWait(crimeWindow))), "\n")
	}
	lines = append(lines, fmt.Sprintf("🚔 @%s reagiu, juntou gente e a quadrilha se deu mal! Cada um teve que pagar indenização:", r.victim.jid.User))
	for _, g := range r.gangsters {
		lines = append(lines, g.fate())
	}
	return strings.Join(lines, "\n")
}

func (r gangRobbery) lootLines() []string {
	var total int64
	for _, g := range r.gangsters {
		total += g.took
	}
	head := fmt.Sprintf("💰 Cercaram @%s e levaram %s!", r.victim.jid.User, coins(total))
	first := r.gangsters[0].took
	if !slices.ContainsFunc(r.gangsters, func(g gangster) bool { return g.took != first }) {
		return []string{head + " Cada um ficou com " + coins(first)}
	}
	lines := []string{head}
	for _, g := range r.gangsters {
		lines = append(lines, g.loot())
	}
	return lines
}

func (r gangRobbery) crewText() string {
	members := make([]string, len(r.gangsters))
	for i, g := range r.gangsters {
		members[i] = fmt.Sprintf("@%s (%s %s%s)", g.acc.jid.User, g.job.emoji, g.job.name, signedPercent(g.job.robBonus))
	}
	return joinWithAnd(members)
}

func (g gangster) loot() string {
	if g.job.robHaul > 0 {
		return fmt.Sprintf("💰 @%s ficou com %s (%d%% a mais)", g.acc.jid.User, coins(g.took), g.job.robHaul)
	}
	return fmt.Sprintf("💰 @%s ficou com %s", g.acc.jid.User, coins(g.took))
}

func (g gangster) fate() string {
	if g.paid == 0 {
		return fmt.Sprintf("🪹 @%s tava tão liso que saiu só com a vergonha", g.acc.jid.User)
	}
	return fmt.Sprintf("💸 @%s pagou %s", g.acc.jid.User, coins(g.paid))
}
