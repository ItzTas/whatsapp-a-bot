package economy

import (
	"context"
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
	coupWindow = 5 * time.Minute
	coupMin    = 3
	coupChance = 0.5
	coupHelp   = 0.1
	coupTerm   = 24 * time.Hour
	coupEvery  = 6 * time.Hour
	coupKind   = "golpe"
)

var errCoupVacant = errors.New("o presidente já saiu do cargo")

type coup struct {
	mu        sync.Mutex
	opened    time.Time
	president core.Account
	plotters  []core.Account
	loyalists []core.Account
	closed    bool
}

type coupState struct {
	Chat      types.JID            `json:"chat"`
	Opened    time.Time            `json:"opened"`
	President core.StoredAccount   `json:"president"`
	Plotters  []core.StoredAccount `json:"plotters"`
	Loyalists []core.StoredAccount `json:"loyalists"`
}

type coupAttempt struct {
	president core.Account
	plotters  []robber
	loyalists []core.Account
	head      Profession
	guards    []Profession
	chance    float64
	success   bool
}

func (b *bot) golpe(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	acc := b.Account(ctx, evt.Info.Sender.ToNonAD())
	loyal := coupLoyal(core.MessageText(evt.Message))
	c, opened, blocked, err := b.openCoup(ctx, chat, acc, loyal)
	if err != nil || blocked {
		return err
	}
	text, refusal := b.joinCoup(ctx, chat, c, acc, loyal, opened)
	if refusal != "" {
		return b.sendMentionsText(ctx, chat, refusal, acc, c.president)
	}
	return b.sendMentionsText(ctx, chat, text, acc, c.president)
}

func coupLoyal(text string) bool {
	words := strings.Fields(core.FoldLetters(text))
	return slices.ContainsFunc(words[min(1, len(words)):], func(word string) bool {
		return slices.Contains([]string{"defender", "defendo", "defesa", "governo", "contra"}, word)
	})
}

func (b *bot) openCoup(ctx context.Context, chat types.JID, acc core.Account, loyal bool) (c *coup, opened, blocked bool, err error) {
	if value, running := b.coups.Load(chat.String()); running {
		return value.(*coup), false, false, nil
	}
	now := time.Now()
	p, found, err := b.Wallets.presidency(ctx, chat, now)
	if err != nil {
		return nil, false, false, fmt.Errorf("ler presidência: %w", err)
	}
	if !found {
		return nil, false, true, b.SendText(ctx, chat, "🏛️ Ninguém é presidente agora, então não tem governo pra derrubar. Manda !votar @pessoa pra abrir a eleição")
	}
	if loyal {
		text := fmt.Sprintf("🛡️ Ninguém tá tramando contra @%s agora, então não tem golpe pra defender", p.president.JID.User)
		return nil, false, true, b.sendMentionsText(ctx, chat, text, p.president)
	}
	if p.president.Key == acc.Key {
		return nil, false, true, b.SendText(ctx, chat, "🤡 Dar golpe em você mesmo? Se quer largar o cargo, manda !renunciar")
	}
	if wait := p.lastCoup.Add(coupEvery).Sub(now); wait > 0 {
		text := fmt.Sprintf("🪖 Já teve tentativa de golpe contra @%s há pouco e o palácio tá cercado. A próxima só daqui a %s", p.president.JID.User, core.FormatWait(wait))
		return nil, false, true, b.sendMentionsText(ctx, chat, text, p.president)
	}
	value, running := b.coups.LoadOrStore(chat.String(), &coup{opened: now, president: p.president})
	c = value.(*coup)
	if running {
		return c, false, false, nil
	}
	if err := b.Wallets.stampCoup(ctx, chat, p.president, now); err != nil {
		b.Client.Log.Errorf("marcar golpe: %v", err)
	}
	go b.closeCoup(ctx, chat, c)
	return c, true, false, nil
}

func (b *bot) joinCoup(ctx context.Context, chat types.JID, c *coup, acc core.Account, loyal, opened bool) (text, refusal string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return "", "🪖 O golpe já tá na rua. Espera o resultado"
	}
	if !loyal && acc.Key == c.president.Key {
		return "", "🤡 Você é o presidente, não dá pra entrar no golpe contra você mesmo. Manda !golpe defender pra defender o governo"
	}
	mine, other := c.sides(loyal)
	isAcc := func(m core.Account) bool { return m.Key == acc.Key }
	if slices.ContainsFunc(*mine, isAcc) {
		return "", fmt.Sprintf("@%s já tá desse lado 🪖", acc.JID.User)
	}
	if slices.ContainsFunc(other, isAcc) {
		return "", fmt.Sprintf("@%s já escolheu o outro lado e não dá pra virar a casaca agora 🐍", acc.JID.User)
	}
	*mine = append(*mine, acc)
	b.saveCoup(ctx, chat, c)
	return c.joinText(acc, loyal, opened), ""
}

func (c *coup) sides(loyal bool) (mine *[]core.Account, other []core.Account) {
	if loyal {
		return &c.loyalists, c.plotters
	}
	return &c.plotters, c.loyalists
}

func (c *coup) joinText(acc core.Account, loyal, opened bool) string {
	left := core.FormatWait(coupWindow - time.Since(c.opened))
	if opened {
		return fmt.Sprintf("🪖 *Golpe de estado!*\n@%s tá tramando derrubar @%s da presidência. Manda !golpe em até %s pra entrar no golpe, ou !golpe defender pra proteger o governo\n_Precisa de pelo menos %d golpistas. Cada golpista a mais soma %d%% de chance e cada defensor tira %d%%, arma na mochila ajuda os dois lados e a proteção do presidente atrapalha o golpe. Se der certo, @%s vira presidente por %s e os decretos caem. Se der errado, quem não escapar vai preso_",
			acc.JID.User, c.president.JID.User, left, coupMin, percent(coupHelp), percent(coupHelp), acc.JID.User, core.FormatWait(coupTerm))
	}
	count := fmt.Sprintf("Golpistas: %d. Defensores: %d. O golpe sai em %s", len(c.plotters), len(c.loyalists), left)
	if loyal {
		return fmt.Sprintf("🛡️ @%s vai defender @%s. %s", acc.JID.User, c.president.JID.User, count)
	}
	return fmt.Sprintf("🪖 @%s entrou no golpe contra @%s. %s", acc.JID.User, c.president.JID.User, count)
}

func (c *coup) close(ctx context.Context, b *bot, chat types.JID) (plotters, loyalists []core.Account) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	b.DropGame(ctx, coupKind, chat.String())
	return slices.Clone(c.plotters), slices.Clone(c.loyalists)
}

func (b *bot) saveCoup(ctx context.Context, chat types.JID, c *coup) {
	state := coupState{Chat: chat, Opened: c.opened, President: core.StoreAccount(c.president), Plotters: storeAccounts(c.plotters), Loyalists: storeAccounts(c.loyalists)}
	b.KeepGame(ctx, coupKind, chat.String(), state, c.opened.Add(coupWindow))
}

func storeAccounts(accounts []core.Account) []core.StoredAccount {
	stored := make([]core.StoredAccount, len(accounts))
	for i, acc := range accounts {
		stored[i] = core.StoreAccount(acc)
	}
	return stored
}

func loadAccounts(stored []core.StoredAccount) []core.Account {
	accounts := make([]core.Account, len(stored))
	for i, s := range stored {
		accounts[i] = s.Account()
	}
	return accounts
}

func (b *bot) restoreCoup(ctx context.Context, saved core.SavedGame) error {
	var state coupState
	if err := json.Unmarshal(saved.State, &state); err != nil {
		return fmt.Errorf("ler golpe salvo: %w", err)
	}
	c := &coup{opened: state.Opened, president: state.President.Account(), plotters: loadAccounts(state.Plotters), loyalists: loadAccounts(state.Loyalists)}
	if _, running := b.coups.LoadOrStore(state.Chat.String(), c); running {
		return fmt.Errorf("já tem outro golpe sendo tramado em %s", state.Chat)
	}
	go b.closeCoup(ctx, state.Chat, c)
	return nil
}

func (c *coup) partialText() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return fmt.Sprintf("🪖 *Golpe sendo tramado!* Golpistas: %d. Defensores: %d. O golpe sai em %s", len(c.plotters), len(c.loyalists), core.FormatWait(coupWindow-time.Since(c.opened)))
}

func (b *bot) closeCoup(ctx context.Context, chat types.JID, c *coup) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Until(c.opened.Add(coupWindow))):
	}
	plotters, loyalists := c.close(ctx, b, chat)
	b.coups.CompareAndDelete(chat.String(), c)
	people := append(append([]core.Account{c.president}, plotters...), loyalists...)
	text, err := b.runCoup(ctx, chat, c.president, plotters, loyalists)
	if err != nil {
		b.Client.Log.Errorf("dar golpe: %v", err)
		return
	}
	if err := b.sendMentionsText(ctx, chat, text, people...); err != nil {
		b.Client.Log.Errorf("anunciar golpe: %v", err)
	}
}

func (b *bot) runCoup(ctx context.Context, chat types.JID, president core.Account, plotters, loyalists []core.Account) (string, error) {
	attempt := coupAttempt{president: president, plotters: make([]robber, len(plotters)), loyalists: loyalists, guards: make([]Profession, len(loyalists))}
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { attempt.head, err = b.ProfessionOf(gctx, chat, president); return err })
	for i, acc := range plotters {
		attempt.plotters[i].acc = acc
		g.Go(func() (err error) { attempt.plotters[i].job, err = b.ProfessionOf(gctx, chat, acc); return err })
	}
	for i, acc := range loyalists {
		g.Go(func() (err error) { attempt.guards[i], err = b.ProfessionOf(gctx, chat, acc); return err })
	}
	if err := g.Wait(); err != nil {
		return "", err
	}
	if len(plotters) < coupMin && !attempt.godPlotting() {
		return fmt.Sprintf("🚫 O golpe contra @%s não saiu: só %s, e precisava de %d", president.JID.User, core.CountLabel(len(plotters), "golpista apareceu", "golpistas apareceram"), coupMin), nil
	}
	attempt.chance = attempt.odds()
	err := b.Wallets.coup(ctx, chat, &attempt, time.Now())
	if errors.Is(err, errCoupVacant) {
		return fmt.Sprintf("🪖 Os golpistas invadiram o palácio, mas @%s já não era mais presidente. Golpe cancelado", president.JID.User), nil
	}
	if err != nil {
		return "", fmt.Errorf("golpe: %w", err)
	}
	return attempt.text(), nil
}

func (a coupAttempt) godPlotting() bool {
	return slices.ContainsFunc(a.plotters, func(r robber) bool { return r.job.divine })
}

func (a coupAttempt) godDefending() bool {
	return a.head.divine || slices.ContainsFunc(a.guards, func(p Profession) bool { return p.divine })
}

func (a coupAttempt) odds() float64 {
	if a.godPlotting() {
		return 1
	}
	if a.godDefending() {
		return 0
	}
	chance := coupChance + coupHelp*float64(len(a.plotters)-len(a.guards)-1) - a.head.violenceGuard() - a.head.curfew()
	for _, r := range a.plotters {
		chance += r.job.arsenal().crime
	}
	for _, p := range a.guards {
		chance -= p.arsenal().guard
	}
	return clampChance(chance)
}

func (p Profession) arsenal() armament {
	w, ok := itemByID(p.weapon)
	if !ok || w.arms == nil {
		return armament{}
	}
	return *w.arms
}

func (s *walletStore) coup(ctx context.Context, chat types.JID, a *coupAttempt, now time.Time) error {
	return s.inTx(ctx, func(tx *ledgerTx) error {
		for _, r := range a.plotters {
			if err := openWallet(ctx, tx, chat, r.acc); err != nil {
				return err
			}
			if err := markCrime(ctx, tx, chat, r.acc, now); err != nil {
				return err
			}
		}
		if a.success = rand.Float64() < a.chance; !a.success {
			return jailCrew(ctx, tx, chat, a.plotters, now)
		}
		return seize(ctx, tx, chat, a.president, a.plotters[0].acc, now)
	})
}

func seize(ctx context.Context, tx *ledgerTx, chat types.JID, president, leader core.Account, now time.Time) error {
	res, err := tx.ExecContext(ctx,
		`UPDATE presidencias SET pessoa = ?, jid = ?, votos = 0, eleito_em = ?, mandato_ate = ?, golpe_em = ?, golpista = 1,
		 impeachment_em = 0, renunciou = 0, `+strings.Join(append(abuseChargeColumns(), decreeColumns()...), " = 0, ")+` = 0
		 WHERE chat = ? AND pessoa = ? AND mandato_ate > ?`,
		leader.Key, leader.JID.String(), now.Unix(), now.Add(coupTerm).Unix(), now.Unix(),
		chat.String(), president.Key, now.Unix(),
	)
	if err != nil {
		return err
	}
	seized, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if seized == 0 {
		return errCoupVacant
	}
	_, err = tx.ExecContext(ctx, "UPDATE carteiras SET preso_ate = 0 WHERE chat = ? AND pessoa = ?", chat.String(), leader.Key)
	return err
}

func (s *walletStore) stampCoup(ctx context.Context, chat types.JID, president core.Account, now time.Time) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE presidencias SET golpe_em = ? WHERE chat = ? AND pessoa = ? AND mandato_ate > ?",
		now.Unix(), chat.String(), president.Key, now.Unix(),
	)
	return err
}

func (a coupAttempt) text() string {
	lines := []string{"🪖 *Golpe de estado!*", "👥 Golpistas: " + mentionList(a.plotterAccounts())}
	if len(a.loyalists) > 0 {
		lines = append(lines, "🛡️ Defensores: "+mentionList(a.loyalists))
	}
	lines = append(lines, fmt.Sprintf("🎯 Chance de dar certo: %d%%", percent(a.chance)), "")
	if a.godPlotting() {
		lines = append(lines, "⚡ Deus tava no golpe, então não tinha como dar errado")
	}
	if !a.godPlotting() && a.godDefending() {
		lines = append(lines, "⚡ Deus tava do lado do governo, então não tinha como dar certo")
	}
	if a.success {
		leader := a.plotters[0].acc
		return strings.Join(append(lines,
			fmt.Sprintf("🏛️ Os tanques tomaram o palácio! @%s derrubou @%s e é o novo presidente por %s. Os decretos do governo anterior caíram, e o !votar só volta quando o mandato acabar", leader.JID.User, a.president.JID.User, core.FormatWait(coupTerm)),
			fmt.Sprintf("🚓 Mas golpe é crime: dá pra !prender quem participou (menos o novo presidente) nas próximas %s", core.FormatWait(crimeWindow)),
		), "\n")
	}
	lines = append(lines, fmt.Sprintf("🚨 O golpe fracassou e @%s continua no cargo!", a.president.JID.User))
	for _, r := range a.plotters {
		lines = append(lines, r.fate())
	}
	return strings.Join(lines, "\n")
}

func (a coupAttempt) plotterAccounts() []core.Account {
	accounts := make([]core.Account, len(a.plotters))
	for i, r := range a.plotters {
		accounts[i] = r.acc
	}
	return accounts
}
