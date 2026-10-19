package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	embezzleCaughtBase  = 0.15
	embezzleCaughtScale = 0.5

	notFaithfulText = "Você não é fiel de igreja nenhuma. As do grupo tão no !igreja lista"
	notLeaderText   = "Só o líder da igreja mexe no cofre 👑 Quem é o líder aparece no !igreja"
)

var embezzleCooldown = cooldown{column: "desvio_em", every: 12 * time.Hour}

var (
	errNotLeader   = errors.New("não é o líder da igreja")
	errTempleTop   = errors.New("o templo já tá no máximo")
	errShortCoffer = errors.New("o cofre não tem tudo isso")
	errLonelyFlock = errors.New("a igreja não tem outros fiéis")
	errThinMiracle = errors.New("o valor não dá pra dividir entre os fiéis")
	errLockedUp    = errors.New("quem desvia tá preso ou de refém")
)

type temple struct {
	emoji string
	name  string
	cost  int64
	power int64
}

var temples = []temple{
	{emoji: "🪑", name: "Salinha alugada", power: 100},
	{emoji: "🏗️", name: "Galpão", cost: 15_000, power: 125},
	{emoji: "🏛️", name: "Templo", cost: 50_000, power: 150},
	{emoji: "🏰", name: "Catedral", cost: 150_000, power: 175},
	{emoji: "🏟️", name: "Megatemplo", cost: 500_000, power: 200},
}

type flock struct {
	id       int64
	name     string
	doctrine string
	leader   string
	coffer   int64
	temple   int
	tithe    int64
}

type rebuild struct {
	church   string
	doctrine string
	from     temple
	to       temple
	coffer   int64
}

type miracle struct {
	church    string
	each      int64
	receivers []types.JID
}

type embezzleOdds struct {
	caught float64
	jail   time.Duration
}

type embezzlement struct {
	church string
	wait   time.Duration
	caught bool
	jail   time.Duration
	fine   int64
	heir   types.JID
}

func scaled(base, power int64) int64 {
	return base * power / 100
}

func templeAt(level int) temple {
	return temples[min(max(level, 0), len(temples)-1)]
}

func (t temple) label() string {
	return t.emoji + " *" + t.name + "*"
}

func nextTempleText(level int) string {
	if level+1 >= len(temples) {
		return "🏁 O templo já tá no máximo"
	}
	next := temples[level+1]
	return fmt.Sprintf("🔨 Próxima reforma: %s, por %s do cofre. O líder manda !igreja reformar, e qualquer fiel ajuda com !igreja ofertar <valor>", next.label(), Coins(next.cost))
}

func templeLadder() string {
	steps := make([]string, len(temples))
	for i, t := range temples {
		steps[i] = t.name
	}
	return strings.Join(steps, " → ")
}

func firstWord(rest string) string {
	fields := strings.Fields(core.FoldLetters(rest))
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func churchRefusal(err error) (string, bool) {
	refusals := map[error]string{
		errNotFaithful: notFaithfulText,
		errNotLeader:   notLeaderText,
		errTempleTop:   "O templo já tá no máximo, não tem mais o que reformar 🏟️",
		errLonelyFlock: "Não tem nenhum outro fiel na igreja pra receber o milagre 😅",
		errLockedUp:    "🔒 Da cadeia (ou do cativeiro) não dá pra mexer no cofre",
	}
	for refused, text := range refusals {
		if errors.Is(err, refused) {
			return text, true
		}
	}
	return "", false
}

func leaderDraw(c church, found bool, acc core.Account, rest, usage string) (int64, string) {
	if !found {
		return 0, notFaithfulText
	}
	if c.leaderKey != acc.Key {
		return 0, notLeaderText
	}
	if c.coffer == 0 {
		return 0, fmt.Sprintf("🏦 O cofre da *%s* tá vazio", c.name)
	}
	amount, ok := ParseStake(firstWord(rest), c.coffer)
	if !ok {
		return 0, fmt.Sprintf("Não entendi o valor. Use assim: %s (ou metade, 10%%, tudo). O cofre tem %s", usage, Coins(c.coffer))
	}
	return amount, ""
}

func embezzleOddsFor(job Profession, amount, coffer int64) embezzleOdds {
	odds := embezzleOdds{
		caught: job.flagrantOdds(embezzleCaughtBase + embezzleCaughtScale*min(float64(amount)/float64(coffer), 1)),
		jail:   Profession{jailTime: baseJail}.sentence(job),
	}
	if job.divine {
		odds.caught = 0
	}
	return odds
}

func mentionLine(people []types.JID) string {
	tags := make([]string, len(people))
	for i, person := range people {
		tags[i] = "@" + person.User
	}
	return strings.Join(tags, " ")
}

func (b *bot) offerChurch(ctx context.Context, evt *events.Message, rest string) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	balance, _, err := b.Wallets.Balance(ctx, chat, acc)
	if err != nil {
		return fmt.Errorf("ler saldo: %w", err)
	}
	amount, ok := ParseStake(firstWord(rest), balance)
	if !ok {
		return b.SendText(ctx, chat, "Não entendi o valor. Use assim: !igreja ofertar 1000 (ou metade, 10%, tudo)")
	}
	f, err := b.Wallets.offer(ctx, chat, acc, amount)
	if text, refused := churchRefusal(err); refused {
		return b.SendText(ctx, chat, text)
	}
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, chat, "Você só tem "+Coins(balance))
	}
	if err != nil {
		return fmt.Errorf("ofertar pra igreja: %w", err)
	}
	text := fmt.Sprintf("🙏 @%s botou %s no cofre da *%s*, que agora tem %s", person.User, Coins(amount), f.name, Coins(f.coffer))
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) rebuildTemple(ctx context.Context, evt *events.Message, _ string) error {
	chat := evt.Info.Chat.ToNonAD()
	r, err := b.Wallets.rebuild(ctx, chat, b.Account(ctx, evt.Info.Sender.ToNonAD()))
	if text, refused := churchRefusal(err); refused {
		return b.SendText(ctx, chat, text)
	}
	if errors.Is(err, errShortCoffer) {
		return b.SendText(ctx, chat, fmt.Sprintf("A reforma pra %s custa %s e o cofre só tem %s. Os fiéis ajudam com !igreja ofertar <valor>", r.to.label(), Coins(r.to.cost), Coins(r.coffer)))
	}
	if err != nil {
		return fmt.Errorf("reformar templo: %w", err)
	}
	d, _ := doctrineByID(r.doctrine)
	text := fmt.Sprintf("🔨 A *%s* reformou e agora é %s (era %s), por %s do cofre!\n✨ A doutrina %s ficou mais forte: %s", r.church, r.to.label(), r.from.label(), Coins(r.to.cost), d.label(), d.effect(r.to.power))
	return b.SendText(ctx, chat, text)
}

func (b *bot) churchMiracle(ctx context.Context, evt *events.Message, rest string) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	c, found, err := b.Wallets.churchOf(ctx, chat, acc)
	if err != nil {
		return fmt.Errorf("ler igreja: %w", err)
	}
	amount, refusal := leaderDraw(c, found, acc, rest, "!igreja milagre 1000")
	if refusal != "" {
		return b.SendText(ctx, chat, refusal)
	}
	m, err := b.Wallets.miracle(ctx, chat, acc, amount)
	if text, refused := churchRefusal(err); refused {
		return b.SendText(ctx, chat, text)
	}
	if errors.Is(err, errShortCoffer) {
		return b.SendText(ctx, chat, "O cofre não tem tudo isso. Confere no !igreja")
	}
	if errors.Is(err, errThinMiracle) {
		return b.SendText(ctx, chat, fmt.Sprintf("Não dá pra dividir %s entre %s 🤏", Coins(amount), core.CountLabel(len(m.receivers), "fiel", "fiéis")))
	}
	if err != nil {
		return fmt.Errorf("fazer milagre: %w", err)
	}
	total := m.each * int64(len(m.receivers))
	text := fmt.Sprintf("✨ *Milagre na %s!* @%s repartiu %s do cofre, e cada fiel recebeu %s\n%s", m.church, person.User, Coins(total), Coins(m.each), mentionLine(m.receivers))
	_, err = b.SendMentions(ctx, chat, text, append([]types.JID{person}, m.receivers...))
	return err
}

func (b *bot) embezzle(ctx context.Context, evt *events.Message, rest string) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	var (
		job   Profession
		c     church
		found bool
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { job, err = b.ProfessionOf(gctx, chat, acc); return err })
	g.Go(func() (err error) { c, found, err = b.Wallets.churchOf(gctx, chat, acc); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler igreja: %w", err)
	}
	amount, refusal := leaderDraw(c, found, acc, rest, "!igreja desviar 1000")
	if refusal != "" {
		return b.SendText(ctx, chat, refusal)
	}
	e, err := b.Wallets.embezzle(ctx, chat, acc, amount, embezzleOddsFor(job, amount, c.coffer), job.pace(embezzleCooldown), time.Now())
	if text, refused := churchRefusal(err); refused {
		return b.SendText(ctx, chat, text)
	}
	if errors.Is(err, errShortCoffer) {
		return b.SendText(ctx, chat, "O cofre não tem tudo isso. Confere no !igreja")
	}
	if err != nil {
		return fmt.Errorf("desviar do cofre: %w", err)
	}
	people := []types.JID{person}
	if !e.heir.IsEmpty() {
		people = append(people, e.heir)
	}
	_, err = b.SendMentions(ctx, chat, embezzleText(e, job, amount, person), people)
	return err
}

func embezzleText(e embezzlement, job Profession, amount int64, person types.JID) string {
	if e.wait > 0 {
		return fmt.Sprintf("🤫 Melhor esperar a poeira baixar: @%s só pode desviar de novo daqui a %s", person.User, core.FormatWait(e.wait))
	}
	if !e.caught {
		return fmt.Sprintf("🤫 @%s desviou %s do cofre da *%s* e ninguém pegou no flagra. Mas foi crime: dá pra !prender ou !denunciar nas próximas %s", person.User, Coins(amount), e.church, core.FormatWait(crimeWindow))
	}
	text := fmt.Sprintf("🚨 @%s foi pego no flagra desviando %s do cofre da *%s*! O dinheiro ficou no cofre, e quem desviou foi pra cadeia por %s", person.User, Coins(amount), e.church, core.FormatWait(e.jail)) +
		jailFactorText(person, job) + fineText(e.fine)
	if e.heir.IsEmpty() {
		return text + "\n👑 Como não tem mais ninguém na igreja, continua líder"
	}
	return text + fmt.Sprintf("\n👑 E perdeu a liderança: agora quem lidera a *%s* é @%s, que tava há mais tempo na igreja", e.church, e.heir.User)
}

func flockOf(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account) (flock, error) {
	var f flock
	err := tx.QueryRowContext(ctx,
		"SELECT i.id, i.nome, i.doutrina, i.lider, i.cofre, i.templo, COALESCE(i.dizimo, ?) FROM carteiras c JOIN igrejas i ON i.id = c.igreja WHERE c.chat = ? AND c.pessoa = ?",
		unsetTithe, chat.String(), acc.Key,
	).Scan(&f.id, &f.name, &f.doctrine, &f.leader, &f.coffer, &f.temple, &f.tithe)
	if errors.Is(err, sql.ErrNoRows) {
		return flock{}, errNotFaithful
	}
	if err != nil {
		return flock{}, err
	}
	d, _ := doctrineByID(f.doctrine)
	f.tithe = d.titheOr(f.tithe)
	return f, nil
}

func ledFlock(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account) (flock, error) {
	f, err := flockOf(ctx, tx, chat, acc)
	if err == nil && f.leader != acc.Key {
		return f, errNotLeader
	}
	return f, err
}

func drainCoffer(ctx context.Context, tx *ledgerTx, id, amount int64) error {
	res, err := tx.ExecContext(ctx, "UPDATE igrejas SET cofre = cofre - ? WHERE id = ? AND cofre >= ?", amount, id, amount)
	if err != nil {
		return err
	}
	drained, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if drained == 0 {
		return errShortCoffer
	}
	return nil
}

func flockJIDs(ctx context.Context, tx *ledgerTx, chat types.JID, id int64, except string) (people []types.JID, err error) {
	rows, err := tx.QueryContext(ctx, "SELECT jid FROM carteiras WHERE chat = ? AND igreja = ? AND pessoa <> ? ORDER BY igreja_desde", chat.String(), id, except)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var jid string
		if err := rows.Scan(&jid); err != nil {
			return nil, err
		}
		person, err := types.ParseJID(jid)
		if err != nil {
			return nil, err
		}
		people = append(people, person)
	}
	return people, rows.Err()
}

func (s *walletStore) offer(ctx context.Context, chat types.JID, acc core.Account, amount int64) (f flock, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		var err error
		if f, err = flockOf(ctx, tx, chat, acc); err != nil {
			return err
		}
		if err := debitBalance(ctx, tx, chat, acc, amount); err != nil {
			return err
		}
		f.coffer += amount
		_, err = tx.ExecContext(ctx, "UPDATE igrejas SET cofre = cofre + ? WHERE id = ?", amount, f.id)
		return err
	})
	return f, err
}

func (s *walletStore) rebuild(ctx context.Context, chat types.JID, acc core.Account) (r rebuild, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		f, err := ledFlock(ctx, tx, chat, acc)
		if err != nil {
			return err
		}
		if f.temple+1 >= len(temples) {
			return errTempleTop
		}
		r = rebuild{church: f.name, doctrine: f.doctrine, from: templeAt(f.temple), to: temples[f.temple+1], coffer: f.coffer}
		if err := drainCoffer(ctx, tx, f.id, r.to.cost); err != nil {
			return err
		}
		r.coffer -= r.to.cost
		_, err = tx.ExecContext(ctx, "UPDATE igrejas SET templo = templo + 1 WHERE id = ?", f.id)
		return err
	})
	return r, err
}

func (s *walletStore) miracle(ctx context.Context, chat types.JID, acc core.Account, amount int64) (m miracle, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		f, err := ledFlock(ctx, tx, chat, acc)
		if err != nil {
			return err
		}
		m.church = f.name
		if m.receivers, err = flockJIDs(ctx, tx, chat, f.id, acc.Key); err != nil {
			return err
		}
		if len(m.receivers) == 0 {
			return errLonelyFlock
		}
		if m.each = amount / int64(len(m.receivers)); m.each == 0 {
			return errThinMiracle
		}
		if err := drainCoffer(ctx, tx, f.id, m.each*int64(len(m.receivers))); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx,
			"UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND igreja = ? AND pessoa <> ?",
			m.each, chat.String(), f.id, acc.Key,
		)
		return err
	})
	return m, err
}

func (s *walletStore) embezzle(ctx context.Context, chat types.JID, acc core.Account, amount int64, odds embezzleOdds, cd cooldown, now time.Time) (e embezzlement, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		f, err := ledFlock(ctx, tx, chat, acc)
		if err != nil {
			return err
		}
		e.church = f.name
		jailUntil, holdUntil, err := confinement(ctx, tx, chat, acc)
		if err != nil {
			return err
		}
		if time.Unix(jailUntil, 0).After(now) || time.Unix(holdUntil, 0).After(now) {
			return errLockedUp
		}
		if f.coffer < amount {
			return errShortCoffer
		}
		if e.wait, err = stampCooldown(ctx, tx, chat, acc, cd, now); err != nil || e.wait > 0 {
			return err
		}
		if e.caught = rand.Float64() < odds.caught; !e.caught {
			return pocketCoffer(ctx, tx, chat, acc, f.id, amount, now)
		}
		e.jail = odds.jail
		_, err = tx.ExecContext(ctx,
			"UPDATE carteiras SET preso_ate = ?, crime_em = 0 WHERE chat = ? AND pessoa = ?",
			now.Add(e.jail).Unix(), chat.String(), acc.Key,
		)
		if err != nil {
			return err
		}
		if e.fine, err = lockUp(ctx, tx, chat, acc, now); err != nil {
			return err
		}
		e.heir, err = depose(ctx, tx, chat, acc, f.id, now)
		return err
	})
	return e, err
}

func pocketCoffer(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, id, amount int64, now time.Time) error {
	if err := drainCoffer(ctx, tx, id, amount); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", amount, chat.String(), acc.Key)
	if err != nil {
		return err
	}
	return markCrime(ctx, tx, chat, acc, now)
}

func depose(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, id int64, now time.Time) (types.JID, error) {
	heir, found, err := crownSuccessor(ctx, tx, chat, id, acc.Key)
	if err != nil || !found {
		return types.JID{}, err
	}
	_, err = tx.ExecContext(ctx, "UPDATE carteiras SET igreja_desde = ? WHERE chat = ? AND pessoa = ?", now.Unix(), chat.String(), acc.Key)
	return heir.JID, err
}
