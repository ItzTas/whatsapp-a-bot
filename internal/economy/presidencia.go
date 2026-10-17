package economy

import (
	"cmp"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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
	termLength            = 72 * time.Hour
	electionWindow        = 10 * time.Minute
	electionQuorum        = 3
	presidentAbuses       = 3
	presidentGuard        = 0.25
	presidentRelease      = 0.7
	presidentRevive       = 0.5
	presidentArrestEvery  = 30 * time.Minute
	presidentPayFactor    = 3
	presidentDailyBonus   = 200
	presidentFineDiscount = 0.95
	electionKind          = "presidencia"
)

type presidency struct {
	president       core.Account
	votes           int
	until           time.Time
	abuses          map[string]int
	lastImpeachment time.Time
	lastCoup        time.Time
	coup            bool
	decrees         decrees
}

type election struct {
	mu      sync.Mutex
	opened  time.Time
	ballots map[string]core.Account
	weights map[string]int
	closed  bool
}

type electionState struct {
	Chat    types.JID                     `json:"chat"`
	Opened  time.Time                     `json:"opened"`
	Ballots map[string]core.StoredAccount `json:"ballots"`
	Weights map[string]int                `json:"weights,omitempty"`
}

type candidate struct {
	acc   core.Account
	votes int
}

type ballot struct {
	changed bool
	refused bool
	weight  int
}

func (b *bot) votar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	choice, found := core.StatsSubjectOf(evt)
	if !found {
		return b.SendText(ctx, chat, "Marca em quem você vota pra presidente: !votar @pessoa (ou !votar eu)")
	}
	voterAcc, candidateAcc := b.Account(ctx, evt.Info.Sender.ToNonAD()), b.Account(ctx, choice)

	var (
		e               *election
		opened, blocked bool
		voterJob        Profession
		g               errgroup.Group
	)
	g.Go(func() (err error) { e, opened, blocked, err = b.openElection(ctx, chat); return err })
	g.Go(func() (err error) { voterJob, err = b.ProfessionOf(ctx, chat, voterAcc); return err })
	if err := g.Wait(); err != nil || blocked {
		return err
	}
	cast := b.castBallot(ctx, chat, e, voterAcc, candidateAcc, voterJob.ballotWeight())
	if cast.refused {
		return b.SendText(ctx, chat, "🗳️ A apuração já começou. Espera o resultado")
	}
	_, err := b.SendMentions(ctx, chat, e.ballotText(voterAcc, candidateAcc, cast, opened), []types.JID{voterAcc.JID, candidateAcc.JID})
	return err
}

func (b *bot) openElection(ctx context.Context, chat types.JID) (e *election, opened, blocked bool, err error) {
	if value, running := b.elections.Load(chat.String()); running {
		return value.(*election), false, false, nil
	}
	p, found, err := b.Wallets.presidency(ctx, chat, time.Now())
	if err != nil {
		return nil, false, false, fmt.Errorf("ler presidência: %w", err)
	}
	if found {
		text := fmt.Sprintf("🏛️ @%s é presidente e o mandato só acaba daqui a %s. Até lá não tem eleição", p.president.JID.User, core.FormatWait(time.Until(p.until)))
		_, err = b.SendMentions(ctx, chat, text, []types.JID{p.president.JID})
		return nil, false, true, err
	}
	value, running := b.elections.LoadOrStore(chat.String(), &election{opened: time.Now(), ballots: map[string]core.Account{}, weights: map[string]int{}})
	e = value.(*election)
	if !running {
		go b.closeElection(ctx, chat, e)
	}
	return e, !running, false, nil
}

func (b *bot) castBallot(ctx context.Context, chat types.JID, e *election, voter, choice core.Account, weight int) ballot {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return ballot{refused: true}
	}
	_, changed := e.ballots[voter.Key]
	e.ballots[voter.Key] = choice
	e.weights[voter.Key] = weight
	b.keepElection(ctx, chat, e)
	return ballot{changed: changed, weight: weight}
}

func (b *bot) closeBallots(ctx context.Context, chat types.JID, e *election) []candidate {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.closed = true
	b.DropGame(ctx, electionKind, chat.String())
	return e.count()
}

func (b *bot) keepElection(ctx context.Context, chat types.JID, e *election) {
	ballots := make(map[string]core.StoredAccount, len(e.ballots))
	for voter, choice := range e.ballots {
		ballots[voter] = core.StoreAccount(choice)
	}
	b.KeepGame(ctx, electionKind, chat.String(), electionState{Chat: chat, Opened: e.opened, Ballots: ballots, Weights: e.weights}, e.opened.Add(electionWindow))
}

func (b *bot) restoreElection(ctx context.Context, saved core.SavedGame) error {
	var state electionState
	if err := json.Unmarshal(saved.State, &state); err != nil {
		return fmt.Errorf("ler eleição salva: %w", err)
	}
	ballots := make(map[string]core.Account, len(state.Ballots))
	for voter, choice := range state.Ballots {
		ballots[voter] = choice.Account()
	}
	weights := state.Weights
	if weights == nil {
		weights = map[string]int{}
	}
	e := &election{opened: state.Opened, ballots: ballots, weights: weights}
	if _, running := b.elections.LoadOrStore(state.Chat.String(), e); running {
		return fmt.Errorf("já tem outra eleição rolando em %s", state.Chat)
	}
	go b.closeElection(ctx, state.Chat, e)
	return nil
}

func (e *election) tally() []candidate {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.count()
}

func (e *election) count() []candidate {
	byKey := map[string]*candidate{}
	for voter, choice := range e.ballots {
		weight := max(e.weights[voter], 1)
		if c, ok := byKey[choice.Key]; ok {
			c.votes += weight
			continue
		}
		byKey[choice.Key] = &candidate{acc: choice, votes: weight}
	}
	candidates := make([]candidate, 0, len(byKey))
	for _, c := range byKey {
		candidates = append(candidates, *c)
	}
	slices.SortFunc(candidates, func(a, b candidate) int { return cmp.Compare(b.votes, a.votes) })
	return candidates
}

func (e *election) ballotText(voter, choice core.Account, cast ballot, opened bool) string {
	left := core.FormatWait(electionWindow - time.Since(e.opened))
	vote := fmt.Sprintf("@%s votou em @%s", voter.JID.User, choice.JID.User)
	if voter.Key == choice.Key {
		vote = fmt.Sprintf("@%s votou em si mesmo 🤡", voter.JID.User)
	}
	if cast.changed {
		vote = fmt.Sprintf("@%s mudou o voto pra @%s", voter.JID.User, choice.JID.User)
	}
	if cast.weight > 1 {
		vote += fmt.Sprintf(" (e o voto vale %d)", cast.weight)
	}
	if !opened {
		return fmt.Sprintf("🗳️ %s. A urna fecha em %s", vote, left)
	}
	return fmt.Sprintf("🗳️ *Eleição pra presidente!*\n%s e abriu a votação. Manda !votar @pessoa em até %s pra votar (dá pra mudar o voto até lá). Precisa de pelo menos %d votos pra valer\n\nQuem ganhar fica %s no cargo e pode:\n%s",
		vote, left, electionQuorum, core.FormatWait(termLength), strings.Join(presidentialPowers(), "\n"))
}

func (b *bot) closeElection(ctx context.Context, chat types.JID, e *election) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Until(e.opened.Add(electionWindow))):
	}
	candidates := b.closeBallots(ctx, chat, e)
	b.elections.CompareAndDelete(chat.String(), e)
	text, err := b.electionResult(ctx, chat, candidates)
	if err != nil {
		b.Client.Log.Errorf("apurar eleição: %v", err)
		return
	}
	if _, err := b.SendMentions(ctx, chat, text, candidateJIDs(candidates)); err != nil {
		b.Client.Log.Errorf("anunciar eleição: %v", err)
	}
}

func (b *bot) electionResult(ctx context.Context, chat types.JID, candidates []candidate) (string, error) {
	total := 0
	for _, c := range candidates {
		total += c.votes
	}
	if total < electionQuorum {
		return fmt.Sprintf("🗳️ Eleição anulada: só %s, e precisava de %d. Manda !votar @pessoa pra tentar de novo", core.CountLabel(total, "voto", "votos"), electionQuorum), nil
	}
	tied := slices.DeleteFunc(slices.Clone(candidates), func(c candidate) bool { return c.votes < candidates[0].votes })
	winner := core.PickOne(tied)
	now := time.Now()
	if err := b.Wallets.inaugurate(ctx, chat, winner.acc, winner.votes, now); err != nil {
		return "", err
	}
	lines := []string{"🗳️ *Resultado da eleição*"}
	for _, c := range candidates {
		lines = append(lines, fmt.Sprintf("@%s: %s", c.acc.JID.User, core.CountLabel(c.votes, "voto", "votos")))
	}
	if len(tied) > 1 {
		lines = append(lines, "", "🪙 Deu empate e o desempate foi na sorte")
	}
	lines = append(lines, "", fmt.Sprintf("🏛️ @%s é o novo presidente! O mandato vai até daqui a %s, e a próxima eleição só depois disso", winner.acc.JID.User, core.FormatWait(termLength)))
	return strings.Join(append(lines, presidentialPowers()...), "\n"), nil
}

func candidateJIDs(candidates []candidate) []types.JID {
	jids := make([]types.JID, len(candidates))
	for i, c := range candidates {
		jids[i] = c.acc.JID
	}
	return jids
}

func presidentialPowers() []string {
	return []string{
		fmt.Sprintf("😈 Abusa do poder %d vezes no mandato em cada um destes: !prender, !trabalhar, !roubar e !sequestrar na hora, mesmo com a espera pendente", presidentAbuses),
		fmt.Sprintf("💼 Salário de presidente: o !trabalhar paga %d vezes mais e o !daily rende +%d%%", presidentPayFactor, presidentDailyBonus),
		fmt.Sprintf("🚔 Pode !prender mesmo sem ser da polícia (%s de cadeia, a cada %s)", core.FormatWait(baseJail), core.FormatWait(presidentArrestEvery)),
		fmt.Sprintf("🔓 Pode !libertar com pelo menos %d%% de chance, e se der errado a multa sai %d%% mais barata", percent(presidentRelease), percent(presidentFineDiscount)),
		fmt.Sprintf("🧟 Pode !reviver quem foi morto com pelo menos %d%% de chance", percent(presidentRevive)),
		fmt.Sprintf("🛡️ Segurança presidencial: quem tenta !roubar, !furtar, !sequestrar ou !matar o presidente tem mais %d%% de chance de errar", percent(presidentGuard)),
		"⚖️ Imunidade: ninguém prende o presidente enquanto durar o mandato, nem com !prender, nem com !denunciar, nem pego no flagra. Quem tava preso sai da cadeia ao tomar posse",
		"📜 Baixa !decreto: mexe no !trabalhar, no !daily, cobra imposto de renda, mexe na poupança, nos preços, no pix, na cadeia e na !fianca, fecha o cassino, mexe nos prêmios do cassino, decreta toque de recolher e, uma vez no mandato, cobra imposto, dá auxílio, anistia, indulto, confisco e intervém nas !acoes. Manda !decreto pra ver a lista",
	}
}

func (b *bot) presidente(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if value, running := b.elections.Load(chat.String()); running {
		e := value.(*election)
		candidates := e.tally()
		_, err := b.SendMentions(ctx, chat, e.partialText(candidates), candidateJIDs(candidates))
		return err
	}
	p, found, err := b.Wallets.presidency(ctx, chat, time.Now())
	if err != nil {
		return fmt.Errorf("ler presidência: %w", err)
	}
	if !found {
		text := fmt.Sprintf("🏛️ Ninguém é presidente agora. Manda !votar @pessoa pra abrir a eleição: a urna fica aberta %s e quem ganhar fica %s no cargo", core.FormatWait(electionWindow), core.FormatWait(termLength))
		return b.SendText(ctx, chat, text+p.decrees.leftoverText())
	}
	text := p.text()
	if value, running := b.impeachments.Load(chat.String()); running {
		text += "\n\n" + value.(*impeachment).partialText()
	}
	if value, running := b.coups.Load(chat.String()); running {
		text += "\n\n" + value.(*coup).partialText()
	}
	_, err = b.SendMentions(ctx, chat, text, []types.JID{p.president.JID})
	return err
}

func (e *election) partialText(candidates []candidate) string {
	lines := []string{fmt.Sprintf("🗳️ *Eleição rolando!* A urna fecha em %s", core.FormatWait(electionWindow-time.Since(e.opened)))}
	for _, c := range candidates {
		lines = append(lines, fmt.Sprintf("@%s: %s", c.acc.JID.User, core.CountLabel(c.votes, "voto", "votos")))
	}
	return strings.Join(append(lines, "", "_Manda !votar @pessoa pra votar_"), "\n")
}

func (p presidency) text() string {
	lines := []string{
		p.headline(),
		fmt.Sprintf("⏳ O mandato acaba em %s", core.FormatWait(time.Until(p.until))),
		"😈 Abusos de poder sobrando: " + p.abusesLeftText(),
		"📜 Decretos de uma vez sobrando: " + p.decrees.onceLeftText(),
	}
	for _, decree := range p.decrees.inForce() {
		lines = append(lines, "📜 Decreto em vigor: "+decree)
	}
	return strings.Join(append(append(lines, "", "*Poderes do cargo*"), presidentialPowers()...), "\n")
}

func (p presidency) headline() string {
	if p.coup {
		return fmt.Sprintf("🏛️ *Presidente:* @%s, que tomou o poder num golpe 🪖", p.president.JID.User)
	}
	return fmt.Sprintf("🏛️ *Presidente:* @%s, eleito com %s", p.president.JID.User, core.CountLabel(p.votes, "voto", "votos"))
}

func (p presidency) abusesLeftText() string {
	parts := make([]string, len(abuseKinds))
	for i, kind := range abuseKinds {
		parts[i] = fmt.Sprintf("%s %d de %d", kind.name, presidentAbuses-p.abuses[kind.charges], presidentAbuses)
	}
	return strings.Join(parts, ", ")
}

func (p Profession) presidential() Profession {
	p.president = true
	p.min *= presidentPayFactor
	p.max *= presidentPayFactor
	p.dailyBonus += presidentDailyBonus
	p.guard += presidentGuard
	p.releaseChance = max(p.releaseChance, presidentRelease)
	p.reviveChance = max(p.reviveChance, presidentRevive)
	if p.jailTime == 0 {
		p.jailTime, p.arrestEvery = baseJail, presidentArrestEvery
	}
	return p
}

func immunityText(president types.JID) string {
	return fmt.Sprintf("⚖️ @%s é presidente e tem imunidade: ninguém prende enquanto durar o mandato", president.User)
}

func (p Profession) badge() string {
	if p.president {
		return "🏛️"
	}
	return p.emoji
}

func presidentialAbuse(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, kind abuseKind, now time.Time) (powerAbuse, error) {
	var used int
	err := tx.QueryRowContext(ctx,
		"UPDATE presidencias SET "+kind.charges+" = "+kind.charges+" + 1 WHERE chat = ? AND pessoa = ? AND mandato_ate > ? AND "+kind.charges+" < ? RETURNING "+kind.charges,
		chat.String(), acc.Key, now.Unix(), presidentAbuses,
	).Scan(&used)
	if errors.Is(err, sql.ErrNoRows) {
		return powerAbuse{}, nil
	}
	return powerAbuse{kind: kind, used: err == nil, presidential: true, left: presidentAbuses - used}, err
}

func abuseChargeColumns() []string {
	columns := make([]string, len(abuseKinds))
	for i, kind := range abuseKinds {
		columns[i] = kind.charges
	}
	return columns
}

func (s *walletStore) presidency(ctx context.Context, chat types.JID, now time.Time) (p presidency, found bool, err error) {
	var (
		jid         string
		until       int64
		impeachedAt int64
		coupAt      int64
	)
	charges := abuseChargeColumns()
	used := make([]int, len(charges))
	columns := decreeColumns()
	levels := make([]int64, len(columns))
	dest := []any{&p.president.Key, &jid, &p.votes, &until, &impeachedAt, &coupAt, &p.coup}
	for i := range used {
		dest = append(dest, &used[i])
	}
	for i := range levels {
		dest = append(dest, &levels[i])
	}
	err = s.db.QueryRowContext(ctx,
		"SELECT pessoa, jid, votos, mandato_ate, impeachment_em, golpe_em, golpista, "+strings.Join(append(charges, columns...), ", ")+" FROM presidencias WHERE chat = ? AND (mandato_ate > ? OR renunciou = 1)",
		chat.String(), now.Unix(),
	).Scan(dest...)
	if errors.Is(err, sql.ErrNoRows) {
		return presidency{}, false, nil
	}
	if err != nil {
		return presidency{}, false, err
	}
	p.abuses = make(map[string]int, len(charges))
	for i, column := range charges {
		p.abuses[column] = used[i]
	}
	p.decrees = zipDecrees(columns, levels)
	p.until = time.Unix(until, 0)
	if !p.until.After(now) {
		return presidency{decrees: p.decrees}, false, nil
	}
	p.lastImpeachment = time.Unix(impeachedAt, 0)
	p.lastCoup = time.Unix(coupAt, 0)
	p.president.JID, err = types.ParseJID(jid)
	return p, err == nil, err
}

func (s *walletStore) inaugurate(ctx context.Context, chat types.JID, acc core.Account, votes int, now time.Time) error {
	return s.inTx(ctx, func(tx *ledgerTx) error {
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx,
			`INSERT INTO presidencias (chat, pessoa, jid, votos, eleito_em, mandato_ate) VALUES (?, ?, ?, ?, ?, ?)
			 ON CONFLICT (chat) DO UPDATE SET pessoa = excluded.pessoa, jid = excluded.jid, votos = excluded.votos,
			 eleito_em = excluded.eleito_em, mandato_ate = excluded.mandato_ate, impeachment_em = 0, golpe_em = 0, golpista = 0, renunciou = 0, `+strings.Join(append(abuseChargeColumns(), decreeColumns()...), " = 0, ")+` = 0`,
			chat.String(), acc.Key, acc.JID.String(), votes, now.Unix(), now.Add(termLength).Unix(),
		)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "UPDATE carteiras SET preso_ate = 0 WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key)
		return err
	})
}

func (b *bot) workTerms(ctx context.Context, chat types.JID, acc core.Account, now time.Time) (job Profession, p presidency, err error) {
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { job, err = b.ProfessionOf(gctx, chat, acc); return err })
	g.Go(func() (err error) { p, _, err = b.Wallets.presidency(gctx, chat, now); return err })
	err = g.Wait()
	return job, p, err
}
