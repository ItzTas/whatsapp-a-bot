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
	heistSize    = 3
	heistWindow  = 2 * time.Minute
	heistChance  = 0.25
	heistLootMin = 2000
	heistLootMax = 6000
	heistKind    = "assalto"
)

var heistCooldown = cooldown{column: "assalto_em", every: 30 * time.Minute}

type heistCrew struct {
	mu      sync.Mutex
	opened  time.Time
	members []core.Account
	closed  bool
}

type heistCrewState struct {
	Chat    types.JID            `json:"chat"`
	Opened  time.Time            `json:"opened"`
	Members []core.StoredAccount `json:"members"`
}

type robber struct {
	acc      core.Account
	job      Profession
	caught   bool
	shielded bool
	sentence time.Duration
	fine     int64
	injury   injury
}

type heist struct {
	robbers []robber
	gear    []item
	backer  firm
	chance  float64
	success bool
	share   int64
}

func (b *bot) assalto(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	acc := b.Account(ctx, evt.Info.Sender.ToNonAD())
	wait, err := b.Wallets.cooldownLeft(ctx, chat, acc, heistCooldown, time.Now())
	if err != nil {
		return fmt.Errorf("assalto: %w", err)
	}
	if wait > 0 {
		text := fmt.Sprintf("🚓 @%s ainda tá sendo procurado pelo último assalto. Espera %s pra entrar em outro", acc.JID.User, core.FormatWait(wait))
		_, err = b.SendMentions(ctx, chat, text, []types.JID{acc.JID})
		return err
	}

	value, joined := b.heistCrews.LoadOrStore(chat.String(), &heistCrew{opened: time.Now()})
	crew := value.(*heistCrew)
	if !joined {
		go b.expireHeist(ctx, chat, crew)
	}
	members, refusal := b.joinCrew(ctx, chat, crew, acc)
	if refusal != "" {
		return b.SendText(ctx, chat, refusal)
	}
	if len(members) < heistSize {
		_, err = b.SendMentions(ctx, chat, crew.recruitText(acc, len(members)), []types.JID{acc.JID})
		return err
	}
	b.heistCrews.CompareAndDelete(chat.String(), crew)
	return b.runHeist(ctx, chat, members)
}

func (b *bot) joinCrew(ctx context.Context, chat types.JID, c *heistCrew, acc core.Account) ([]core.Account, string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, "🏦 A equipe acabou de sair pro banco. Manda !assalto de novo pra montar outra"
	}
	if slices.ContainsFunc(c.members, func(m core.Account) bool { return m.Key == acc.Key }) {
		return nil, fmt.Sprintf("Você já tá na equipe 🦹 Falta %s pra fechar", core.CountLabel(heistSize-len(c.members), "pessoa", "pessoas"))
	}
	c.members = append(c.members, acc)
	c.closed = len(c.members) == heistSize
	b.saveCrew(ctx, chat, c)
	return slices.Clone(c.members), ""
}

func (b *bot) abandonCrew(ctx context.Context, chat types.JID, c *heistCrew) ([]core.Account, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, false
	}
	c.closed = true
	b.DropGame(ctx, heistKind, chat.String())
	return c.members, true
}

func (b *bot) saveCrew(ctx context.Context, chat types.JID, c *heistCrew) {
	if c.closed {
		b.DropGame(ctx, heistKind, chat.String())
		return
	}
	members := make([]core.StoredAccount, len(c.members))
	for i, m := range c.members {
		members[i] = core.StoreAccount(m)
	}
	b.KeepGame(ctx, heistKind, chat.String(), heistCrewState{Chat: chat, Opened: c.opened, Members: members}, c.opened.Add(heistWindow))
}

func (b *bot) restoreHeist(ctx context.Context, saved core.SavedGame) error {
	var state heistCrewState
	if err := json.Unmarshal(saved.State, &state); err != nil {
		return fmt.Errorf("ler assalto salvo: %w", err)
	}
	members := make([]core.Account, len(state.Members))
	for i, m := range state.Members {
		members[i] = m.Account()
	}
	crew := &heistCrew{opened: state.Opened, members: members}
	if _, taken := b.heistCrews.LoadOrStore(state.Chat.String(), crew); taken {
		return fmt.Errorf("já tem outro assalto sendo montado em %s", state.Chat)
	}
	go b.expireHeist(ctx, state.Chat, crew)
	return nil
}

func (c *heistCrew) recruitText(acc core.Account, size int) string {
	missing := core.CountLabel(heistSize-size, "pessoa", "pessoas")
	left := core.FormatWait(heistWindow - time.Since(c.opened))
	if size > 1 {
		return fmt.Sprintf("🦹 @%s entrou no assalto. Falta %s, manda !assalto em até %s pra entrar", acc.JID.User, missing, left)
	}
	return fmt.Sprintf("🏦 *Assalto ao banco!*\n@%s tá montando uma equipe pra assaltar o banco. Falta %s: manda !assalto em até %s pra entrar\n_A profissão de cada um e o equipamento na mochila (%s) mudam a chance. Se der errado, quem não conseguir escapar vai preso_",
		acc.JID.User, missing, left, heistGearList())
}

func heistGearList() string {
	var gear []string
	for _, it := range catalog {
		if it.heist > 0 {
			gear = append(gear, it.emoji+" "+strings.ToLower(it.name))
		}
	}
	return core.JoinWithAnd(gear)
}

func (b *bot) expireHeist(ctx context.Context, chat types.JID, crew *heistCrew) {
	defer b.Rescue(ctx, chat)
	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Until(crew.opened.Add(heistWindow))):
	}
	members, abandoned := b.abandonCrew(ctx, chat, crew)
	if !abandoned {
		return
	}
	b.heistCrews.CompareAndDelete(chat.String(), crew)
	showed := "apareceu"
	if len(members) > 1 {
		showed = "apareceram"
	}
	text := fmt.Sprintf("🚫 O assalto foi cancelado: só %s %s. Precisa de %d pessoas", mentionList(members), showed, heistSize)
	if _, err := b.SendMentions(ctx, chat, text, accountJIDs(members)); err != nil {
		b.Client.Log.Errorf("cancelar !assalto: %v", err)
	}
}

func (b *bot) runHeist(ctx context.Context, chat types.JID, members []core.Account) error {
	robbers := make([]robber, len(members))
	g, gctx := errgroup.WithContext(ctx)
	for i, acc := range members {
		robbers[i].acc = acc
		g.Go(func() (err error) { robbers[i].job, err = b.ProfessionOf(gctx, chat, acc); return err })
	}
	if err := g.Wait(); err != nil {
		return err
	}
	h, err := b.Wallets.heist(ctx, chat, robbers, time.Now())
	if err != nil {
		return fmt.Errorf("assalto: %w", err)
	}
	hurtText, freed := h.injuries()
	cover := ""
	if h.success {
		cover = h.backer.coverText("a equipe")
	}
	if _, err := b.SendMentions(ctx, chat, cover+h.text()+hurtText, append(accountJIDs(members), freed...)); err != nil {
		return err
	}
	b.stirCrewEvent(ctx, chat, uncaught(h.robbers), eventSpot{place: "!assalto", won: h.success})
	return nil
}

func (s *walletStore) heist(ctx context.Context, chat types.JID, robbers []robber, now time.Time) (h heist, err error) {
	h.robbers = slices.Clone(robbers)
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		backers := make([]firm, len(robbers))
		for i, r := range robbers {
			if err := openWallet(ctx, tx, chat, r.acc); err != nil {
				return err
			}
			backer, err := syndicateOf(ctx, tx, chat, r.acc, now)
			if err != nil {
				return err
			}
			backers[i] = backer
			_, err = tx.ExecContext(ctx,
				"UPDATE carteiras SET assalto_em = ?, crime_em = ?, ficha_em = ? WHERE chat = ? AND pessoa = ?",
				now.Unix(), now.Unix(), now.Unix(), chat.String(), r.acc.Key,
			)
			if err != nil {
				return err
			}
		}
		gear, err := spendGear(ctx, tx, chat, robbers)
		if err != nil {
			return err
		}
		h.gear, h.backer = gear, crewSyndicate(backers)
		h.chance = clampChance(heistChance + crewBonus(robbers) + gearBonus(gear) + h.backer.edge)
		h.success = rand.Float64() < crewHangover(robbers, func(r robber) Profession { return r.job }).sober(h.chance)
		if err := woundCrew(ctx, tx, chat, h.robbers, h.success, now); err != nil {
			return err
		}
		if h.success {
			h.share = (heistLootMin + rand.Int64N(heistLootMax-heistLootMin+1)) / int64(len(robbers))
			return payCrew(ctx, tx, chat, robbers, h.share)
		}
		return jailCrew(ctx, tx, chat, h.robbers, now)
	})
	return h, err
}

func spendGear(ctx context.Context, tx *ledgerTx, chat types.JID, robbers []robber) ([]item, error) {
	var gear []item
	for _, it := range catalog {
		if it.heist == 0 {
			continue
		}
		spent, err := spendFromCrew(ctx, tx, chat, robbers, it.id)
		if err != nil {
			return nil, err
		}
		if spent {
			gear = append(gear, it)
		}
	}
	return gear, nil
}

func spendFromCrew(ctx context.Context, tx *ledgerTx, chat types.JID, robbers []robber, id string) (bool, error) {
	for _, r := range robbers {
		err := takeItems(ctx, tx, chat, r.acc, id, 1)
		if errors.Is(err, errNoItems) {
			continue
		}
		return err == nil, err
	}
	return false, nil
}

func payCrew(ctx context.Context, tx *ledgerTx, chat types.JID, robbers []robber, share int64) error {
	for _, r := range robbers {
		_, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", share, chat.String(), r.acc.Key)
		if err != nil {
			return err
		}
	}
	return nil
}

func woundCrew(ctx context.Context, tx *ledgerTx, chat types.JID, robbers []robber, success bool, now time.Time) (err error) {
	for i := range robbers {
		r := &robbers[i]
		if r.injury, err = hurt(ctx, tx, chat, r.acc, heistRisk.by(r.job).wound(success), now); err != nil {
			return err
		}
	}
	return nil
}

func jailCrew(ctx context.Context, tx *ledgerTx, chat types.JID, robbers []robber, now time.Time) error {
	for i := range robbers {
		r := &robbers[i]
		if r.injury.died {
			continue
		}
		if r.shielded = rand.Float64() >= r.job.flagrantOdds(1); r.shielded {
			continue
		}
		if rand.Float64() < r.job.sober(r.job.escapeOdds()) {
			continue
		}
		r.caught = true
		r.sentence = Profession{jailTime: baseJail}.sentence(r.job)
		_, err := tx.ExecContext(ctx,
			"UPDATE carteiras SET preso_ate = MAX(preso_ate, ?), crime_em = 0 WHERE chat = ? AND pessoa = ?",
			now.Add(r.sentence).Unix(), chat.String(), r.acc.Key,
		)
		if err != nil {
			return err
		}
		if r.fine, err = lockUp(ctx, tx, chat, r.acc, now); err != nil {
			return err
		}
	}
	return nil
}

func crewBonus(robbers []robber) float64 {
	var bonus, curfew float64
	for _, r := range robbers {
		bonus += r.job.heistBonus
		curfew = r.job.curfew()
	}
	return bonus - curfew
}

func gearBonus(gear []item) float64 {
	var bonus float64
	for _, it := range gear {
		bonus += it.heist
	}
	return bonus
}

func (h heist) text() string {
	lines := []string{"🏦 *Assalto ao banco!*", "👥 Equipe: " + h.crewText()}
	if len(h.gear) > 0 {
		lines = append(lines, "🧰 Equipamento gasto: "+h.gearText())
	}
	lines = append(lines, fmt.Sprintf("🎯 Chance de dar certo: %d%%", percent(h.chance)), "")
	if h.success {
		return strings.Join(append(lines,
			fmt.Sprintf("💰 Renderam os seguranças, abriram o cofre e sumiram com %s! Cada um levou %s", Coins(h.share*int64(len(h.robbers))), Coins(h.share)),
			fmt.Sprintf("🚓 Mas a polícia tá atrás: dá pra !prender quem tava no assalto nas próximas %s", core.FormatWait(crimeWindow)),
		), "\n")
	}
	lines = append(lines, "🚨 O alarme disparou e a polícia cercou o banco!")
	for _, r := range h.robbers {
		if !r.injury.died {
			lines = append(lines, r.fate())
		}
	}
	return strings.Join(lines, "\n")
}

func (h heist) injuries() (text string, freed []types.JID) {
	for _, r := range h.robbers {
		hurtText, escaped := r.injury.text(r.acc.JID)
		text += hurtText
		freed = append(freed, escaped...)
	}
	return text, freed
}

func (h heist) crewText() string {
	members := make([]string, len(h.robbers))
	for i, r := range h.robbers {
		members[i] = fmt.Sprintf("@%s (%s %s%s)", r.acc.JID.User, r.job.emoji, r.job.name, signedPercent(r.job.heistBonus))
	}
	return core.JoinWithAnd(members)
}

func (h heist) gearText() string {
	gear := make([]string, len(h.gear))
	for i, it := range h.gear {
		gear[i] = fmt.Sprintf("%s %s%s", it.emoji, it.name, signedPercent(it.heist))
	}
	return core.JoinWithAnd(gear)
}

func (r robber) fate() string {
	if r.caught && r.fine > 0 {
		return fmt.Sprintf("🚔 @%s foi preso, fica %s na cadeia e pagou %s de multa", r.acc.JID.User, core.FormatWait(r.sentence), Coins(r.fine))
	}
	if r.caught {
		return fmt.Sprintf("🚔 @%s foi preso e fica %s na cadeia", r.acc.JID.User, core.FormatWait(r.sentence))
	}
	if r.shielded && r.job.president {
		return fmt.Sprintf("⚖️ @%s foi pego, mas é presidente: mostrou a imunidade e saiu pela porta da frente", r.acc.JID.User)
	}
	return fmt.Sprintf("🏃 @%s escapou pela janela do banheiro, mas a polícia ainda pode !prender nas próximas %s", r.acc.JID.User, core.FormatWait(crimeWindow))
}

func signedPercent(f float64) string {
	if f == 0 {
		return ""
	}
	return fmt.Sprintf(", %+d%%", percent(f))
}

func mentionList(accounts []core.Account) string {
	names := make([]string, len(accounts))
	for i, acc := range accounts {
		names[i] = "@" + acc.JID.User
	}
	return core.JoinWithAnd(names)
}

func accountJIDs(accounts []core.Account) []types.JID {
	jids := make([]types.JID, len(accounts))
	for i, acc := range accounts {
		jids[i] = acc.JID
	}
	return jids
}

func (s *walletStore) cooldownLeft(ctx context.Context, chat types.JID, acc core.Account, cd cooldown, now time.Time) (time.Duration, error) {
	var last int64
	err := s.db.QueryRowContext(ctx, "SELECT "+cd.column+" FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&last)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return max(time.Unix(last, 0).Add(cd.every).Sub(now), 0), nil
}
