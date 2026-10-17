package economy

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

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
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
	victim  core.Account
	members []core.Account
	closed  bool
}

type gangState struct {
	Chat    types.JID            `json:"chat"`
	Opened  time.Time            `json:"opened"`
	Victim  core.StoredAccount   `json:"victim"`
	Members []core.StoredAccount `json:"members"`
}

type gangster struct {
	acc  core.Account
	job  Profession
	paid int64
	took int64
}

type gangRobbery struct {
	victim    core.Account
	gangsters []gangster
	chance    float64
	poor      bool
	alarmed   bool
	success   bool
}

func (b *bot) quadrilha(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	acc := b.Account(ctx, evt.Info.Sender.ToNonAD())
	var target *core.Account
	if mentions := core.MentionedJIDs(evt.Message); len(mentions) > 0 {
		victim := b.Account(ctx, mentions[0].ToNonAD())
		target = &victim
	}
	if target != nil && target.Key == acc.Key {
		return b.SendText(ctx, chat, "Montar quadrilha pra roubar você mesmo? Isso se chama mudar o dinheiro de bolso 🤡")
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
		return b.SendText(ctx, chat, "Marca quem a quadrilha vai roubar: !quadrilha @pessoa")
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

func (b *bot) sendMentionsText(ctx context.Context, chat types.JID, text string, people ...core.Account) error {
	if text == "" {
		return nil
	}
	_, err := b.SendMentions(ctx, chat, text, accountJIDs(people))
	return err
}

func (b *bot) robWait(ctx context.Context, chat types.JID, acc core.Account) (string, error) {
	job, err := b.ProfessionOf(ctx, chat, acc)
	if err != nil {
		return "", err
	}
	wait, err := b.Wallets.cooldownLeft(ctx, chat, acc, job.robCooldown(), time.Now())
	if err != nil {
		return "", fmt.Errorf("quadrilha: %w", err)
	}
	if wait == 0 {
		return "", nil
	}
	return fmt.Sprintf("🚓 @%s ainda tá sendo procurado pela polícia. Espera %s pra roubar de novo", acc.JID.User, core.FormatWait(wait)), nil
}

func (b *bot) gangFor(ctx context.Context, chat types.JID, target *core.Account) (*gang, bool) {
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

func (b *bot) joinGang(ctx context.Context, chat types.JID, g *gang, acc core.Account, target *core.Account) ([]core.Account, string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return nil, "🦹 A quadrilha acabou de partir pra cima. Manda !quadrilha @pessoa pra montar outra"
	}
	if target != nil && target.Key != g.victim.Key {
		return nil, fmt.Sprintf("Já tem uma quadrilha atrás de @%s. Manda !quadrilha pra entrar nela", g.victim.JID.User)
	}
	if acc.Key == g.victim.Key {
		return nil, "Essa quadrilha é pra te roubar, não dá pra entrar nela 🤡"
	}
	if slices.ContainsFunc(g.members, func(m core.Account) bool { return m.Key == acc.Key }) {
		return nil, fmt.Sprintf("@%s já tá na quadrilha 🦹", acc.JID.User)
	}
	g.members = append(g.members, acc)
	g.closed = len(g.members) == gangSize
	b.saveGang(ctx, chat, g)
	return slices.Clone(g.members), ""
}

func (b *bot) abandonGang(ctx context.Context, chat types.JID, g *gang) ([]core.Account, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return nil, false
	}
	g.closed = true
	b.DropGame(ctx, gangKind, chat.String())
	return g.members, true
}

func (b *bot) saveGang(ctx context.Context, chat types.JID, g *gang) {
	if g.closed {
		b.DropGame(ctx, gangKind, chat.String())
		return
	}
	members := make([]core.StoredAccount, len(g.members))
	for i, m := range g.members {
		members[i] = core.StoreAccount(m)
	}
	state := gangState{Chat: chat, Opened: g.opened, Victim: core.StoreAccount(g.victim), Members: members}
	b.KeepGame(ctx, gangKind, chat.String(), state, g.opened.Add(gangWindow))
}

func (b *bot) restoreGang(ctx context.Context, saved core.SavedGame) error {
	var state gangState
	if err := json.Unmarshal(saved.State, &state); err != nil {
		return fmt.Errorf("ler quadrilha salva: %w", err)
	}
	members := make([]core.Account, len(state.Members))
	for i, m := range state.Members {
		members[i] = m.Account()
	}
	g := &gang{opened: state.Opened, victim: state.Victim.Account(), members: members}
	if _, taken := b.gangs.LoadOrStore(state.Chat.String(), g); taken {
		return fmt.Errorf("já tem outra quadrilha sendo montada em %s", state.Chat)
	}
	go b.expireGang(ctx, state.Chat, g)
	return nil
}

func (g *gang) recruitText(acc core.Account, size int) string {
	room := core.CountLabel(gangSize-size, "vaga", "vagas")
	left := core.FormatWait(gangWindow - time.Since(g.opened))
	if size > 1 {
		return fmt.Sprintf("🦹 @%s entrou na quadrilha contra @%s. Sobra %s: manda !quadrilha em até %s pra entrar. Se ninguém mais vier, a quadrilha parte pra cima quando o tempo acabar", acc.JID.User, g.victim.JID.User, room, left)
	}
	return fmt.Sprintf("🦹 *Quadrilha!*\n@%s tá juntando gente pra roubar @%s. Sobram %s: manda !quadrilha em até %s pra entrar\n_Com %d já dá pra ir, com %d parte na hora. Cada um a mais soma %d%% de chance, e se der errado cada um paga indenização pra vítima_",
		acc.JID.User, g.victim.JID.User, room, left, gangMin, gangSize, percent(gangHelp))
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
			b.Client.Log.Errorf("rodar !quadrilha: %v", err)
		}
		return
	}
	text := fmt.Sprintf("🚫 A quadrilha contra @%s foi desfeita: só @%s apareceu. Precisa de pelo menos %d", g.victim.JID.User, members[0].JID.User, gangMin)
	if err := b.sendMentionsText(ctx, chat, text, members[0], g.victim); err != nil {
		b.Client.Log.Errorf("desfazer !quadrilha: %v", err)
	}
}

func (b *bot) runGang(ctx context.Context, chat types.JID, victim core.Account, members []core.Account) error {
	gangsters := make([]gangster, len(members))
	var victimJob Profession
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { victimJob, err = b.ProfessionOf(gctx, chat, victim); return err })
	for i, acc := range members {
		gangsters[i].acc = acc
		g.Go(func() (err error) { gangsters[i].job, err = b.ProfessionOf(gctx, chat, acc); return err })
	}
	if err := g.Wait(); err != nil {
		return err
	}
	chance := clampChance(robSuccessChance + gangBonus(gangsters) - victimJob.guard)
	r, err := b.Wallets.gangRob(ctx, chat, victim, gangsters, chance, time.Now())
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

func (s *walletStore) gangRob(ctx context.Context, chat types.JID, victim core.Account, gangsters []gangster, chance float64, now time.Time) (r gangRobbery, err error) {
	r = gangRobbery{victim: victim, gangsters: slices.Clone(gangsters), chance: chance}
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		if err := openWallet(ctx, tx, chat, victim); err != nil {
			return err
		}
		var victimBalance int64
		err := tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), victim.Key).Scan(&victimBalance)
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
			now.Unix(), now.Unix(), chat.String(), g.acc.Key,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func payGang(ctx context.Context, tx *sql.Tx, chat types.JID, victim core.Account, gangsters []gangster, share int64) error {
	for i := range gangsters {
		g := &gangsters[i]
		g.took = share * (100 + g.job.robHaul) / 100
		if err := moveCoins(ctx, tx, chat, victim, g.acc, g.took); err != nil {
			return err
		}
	}
	return nil
}

func indemnify(ctx context.Context, tx *sql.Tx, chat types.JID, victim core.Account, gangsters []gangster) error {
	for i := range gangsters {
		g := &gangsters[i]
		var balance int64
		err := tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), g.acc.Key).Scan(&balance)
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
		return fmt.Sprintf("🪹 A quadrilha cercou @%s, mas a pessoa tem menos de %s. Nem valeu o risco, e ninguém ficou procurado", r.victim.JID.User, Coins(robMinVictim))
	}
	lines := []string{"🦹 *Quadrilha!*", "👥 " + r.crewText(), fmt.Sprintf("🎯 Chance de dar certo: %d%%", percent(r.chance)), ""}
	if r.alarmed {
		return strings.Join(append(lines, fmt.Sprintf("🚨 A quadrilha cercou @%s, mas o alarme disparou! Saíram correndo de mãos vazias e o alarme queimou", r.victim.JID.User)), "\n")
	}
	if r.success {
		lines = append(lines, r.lootLines()...)
		return strings.Join(append(lines, fmt.Sprintf("🚓 Mas a polícia tá de olho: dá pra !prender quem tava na quadrilha nas próximas %s", core.FormatWait(crimeWindow))), "\n")
	}
	lines = append(lines, fmt.Sprintf("🚔 @%s reagiu, juntou gente e a quadrilha se deu mal! Cada um teve que pagar indenização:", r.victim.JID.User))
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
	head := fmt.Sprintf("💰 Cercaram @%s e levaram %s!", r.victim.JID.User, Coins(total))
	first := r.gangsters[0].took
	if !slices.ContainsFunc(r.gangsters, func(g gangster) bool { return g.took != first }) {
		return []string{head + " Cada um ficou com " + Coins(first)}
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
		members[i] = fmt.Sprintf("@%s (%s %s%s)", g.acc.JID.User, g.job.emoji, g.job.name, signedPercent(g.job.robBonus))
	}
	return core.JoinWithAnd(members)
}

func (g gangster) loot() string {
	if g.job.robHaul > 0 {
		return fmt.Sprintf("💰 @%s ficou com %s (%d%% a mais)", g.acc.JID.User, Coins(g.took), g.job.robHaul)
	}
	return fmt.Sprintf("💰 @%s ficou com %s", g.acc.JID.User, Coins(g.took))
}

func (g gangster) fate() string {
	if g.paid == 0 {
		return fmt.Sprintf("🪹 @%s tava tão liso que saiu só com a vergonha", g.acc.JID.User)
	}
	return fmt.Sprintf("💸 @%s pagou %s", g.acc.JID.User, Coins(g.paid))
}
