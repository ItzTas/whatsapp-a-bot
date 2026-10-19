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
	termLength            = 2 * time.Hour
	electionWindow        = 10 * time.Minute
	runoffWindow          = 5 * time.Minute
	electionQuorum        = 3
	presidentAbuses       = 1
	presidentGuard        = 0.25
	presidentImmunity     = 0.75
	presidentRelease      = 0.7
	presidentRevive       = 0.5
	presidentArrestEvery  = 30 * time.Minute
	presidentPayBonus     = 50
	presidentDailyBonus   = 50
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
	drawn           bool
	decrees         decrees
}

type election struct {
	mu      sync.Mutex
	opened  time.Time
	runoff  []core.Account
	ballots map[string]core.Account
	weights map[string]int
	closed  bool
}

type electionState struct {
	Chat    types.JID                     `json:"chat"`
	Opened  time.Time                     `json:"opened"`
	Ballots map[string]core.StoredAccount `json:"ballots"`
	Weights map[string]int                `json:"weights,omitempty"`
	Runoff  []core.StoredAccount          `json:"runoff,omitempty"`
}

type candidate struct {
	acc   core.Account
	votes int
}

type ballot struct {
	changed bool
	refused bool
	outside bool
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
	if cast.outside {
		_, err := b.SendMentions(ctx, chat, e.outsideText(), e.runoffJIDs())
		return err
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
	value, running := b.elections.LoadOrStore(chat.String(), newElection(time.Now(), nil))
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
	if !e.admits(choice) {
		return ballot{outside: true}
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
	runoff := make([]core.StoredAccount, len(e.runoff))
	for i, acc := range e.runoff {
		runoff[i] = core.StoreAccount(acc)
	}
	b.KeepGame(ctx, electionKind, chat.String(), electionState{Chat: chat, Opened: e.opened, Ballots: ballots, Weights: e.weights, Runoff: runoff}, e.closes())
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
	runoff := make([]core.Account, len(state.Runoff))
	for i, acc := range state.Runoff {
		runoff[i] = acc.Account()
	}
	e := &election{opened: state.Opened, runoff: runoff, ballots: ballots, weights: weights}
	if _, running := b.elections.LoadOrStore(state.Chat.String(), e); running {
		return fmt.Errorf("já tem outra eleição rolando em %s", state.Chat)
	}
	go b.closeElection(ctx, state.Chat, e)
	return nil
}

func newElection(opened time.Time, runoff []core.Account) *election {
	return &election{opened: opened, runoff: runoff, ballots: map[string]core.Account{}, weights: map[string]int{}}
}

func (e *election) closes() time.Time {
	if len(e.runoff) > 0 {
		return e.opened.Add(runoffWindow)
	}
	return e.opened.Add(electionWindow)
}

func (e *election) admits(choice core.Account) bool {
	return len(e.runoff) == 0 || slices.ContainsFunc(e.runoff, func(acc core.Account) bool { return acc.Key == choice.Key })
}

func (e *election) runoffJIDs() []types.JID {
	jids := make([]types.JID, len(e.runoff))
	for i, acc := range e.runoff {
		jids[i] = acc.JID
	}
	return jids
}

func (e *election) outsideText() string {
	tags := make([]string, len(e.runoff))
	for i, acc := range e.runoff {
		tags[i] = "@" + acc.JID.User
	}
	return fmt.Sprintf("🗳️ É segundo turno: só vale voto em %s", core.JoinWithOr(tags))
}

func (e *election) tally() []candidate {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.count()
}

func (e *election) count() []candidate {
	byKey := map[string]*candidate{}
	for _, acc := range e.runoff {
		byKey[acc.Key] = &candidate{acc: acc}
	}
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
	left := core.FormatWait(time.Until(e.closes()))
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
	case <-time.After(time.Until(e.closes())):
	}
	candidates := b.closeBallots(ctx, chat, e)
	if runoff, tied := e.runoffBetween(candidates); tied {
		b.openRunoff(ctx, chat, e, candidates, runoff)
		return
	}
	b.elections.CompareAndDelete(chat.String(), e)
	text, err := b.electionResult(ctx, chat, e, candidates)
	if err != nil {
		b.Client.Log.Errorf("apurar eleição: %v", err)
		return
	}
	if _, err := b.SendMentions(ctx, chat, text, candidateJIDs(candidates)); err != nil {
		b.Client.Log.Errorf("anunciar eleição: %v", err)
	}
}

func (e *election) runoffBetween(candidates []candidate) ([]core.Account, bool) {
	if len(e.runoff) > 0 || totalVotes(candidates) < electionQuorum {
		return nil, false
	}
	tied := leaders(candidates)
	if len(tied) < 2 {
		return nil, false
	}
	runoff := make([]core.Account, len(tied))
	for i, c := range tied {
		runoff[i] = c.acc
	}
	return runoff, true
}

func (b *bot) openRunoff(ctx context.Context, chat types.JID, first *election, candidates []candidate, runoff []core.Account) {
	e := newElection(time.Now(), runoff)
	b.keepElection(ctx, chat, e)
	if !b.elections.CompareAndSwap(chat.String(), first, e) {
		b.DropGame(ctx, electionKind, chat.String())
		return
	}
	go b.closeElection(ctx, chat, e)
	lines := []string{"🗳️ *Deu empate na eleição!*"}
	lines = append(lines, votesLines(candidates)...)
	lines = append(lines, "", fmt.Sprintf("⚔️ Vai ter segundo turno entre %s. Os votos zeraram: manda !votar @pessoa em até %s, e só vale voto em quem empatou. Se empatar de novo, a sorte decide",
		core.JoinMentions(e.runoffJIDs()), core.FormatWait(runoffWindow)))
	if _, err := b.SendMentions(ctx, chat, strings.Join(lines, "\n"), candidateJIDs(candidates)); err != nil {
		b.Client.Log.Errorf("anunciar segundo turno: %v", err)
	}
}

func totalVotes(candidates []candidate) int {
	total := 0
	for _, c := range candidates {
		total += c.votes
	}
	return total
}

func leaders(candidates []candidate) []candidate {
	return slices.DeleteFunc(slices.Clone(candidates), func(c candidate) bool { return c.votes < candidates[0].votes })
}

func votesLines(candidates []candidate) []string {
	lines := make([]string, len(candidates))
	for i, c := range candidates {
		lines[i] = fmt.Sprintf("@%s: %s", c.acc.JID.User, core.CountLabel(c.votes, "voto", "votos"))
	}
	return lines
}

func (b *bot) electionResult(ctx context.Context, chat types.JID, e *election, candidates []candidate) (string, error) {
	runoff := len(e.runoff) > 0
	now := time.Now()
	if total := totalVotes(candidates); !runoff && total < electionQuorum {
		vacant, err := b.Wallets.seat(ctx, chat, now)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("🗳️ Eleição anulada: só %s, e precisava de %d. Manda !votar @pessoa pra tentar de novo\n\n🎲 %s", core.CountLabel(total, "voto", "votos"), electionQuorum, vacancyNote(vacant.drawAt, now)), nil
	}
	tied := leaders(candidates)
	winner := core.PickOne(tied)
	if err := b.Wallets.inaugurate(ctx, chat, winner.acc, winner.votes, now); err != nil {
		return "", err
	}
	title := "🗳️ *Resultado da eleição*"
	if runoff {
		title = "🗳️ *Resultado do segundo turno*"
	}
	lines := append([]string{title}, votesLines(candidates)...)
	if len(tied) > 1 {
		lines = append(lines, "", "🪙 Empatou de novo no segundo turno e o desempate foi na sorte")
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
		fmt.Sprintf("😈 Abusa do poder %s no mandato em cada um destes: !prender, !trabalhar, !roubar e !sequestrar na hora, mesmo com a espera pendente", core.CountLabel(presidentAbuses, "vez", "vezes")),
		fmt.Sprintf("💼 Salário de presidente: o !trabalhar paga +%d%% e o !daily rende +%d%%", presidentPayBonus, presidentDailyBonus),
		fmt.Sprintf("🚔 Pode !prender mesmo sem ser da polícia (%s de cadeia, a cada %s)", core.FormatWait(baseJail), core.FormatWait(presidentArrestEvery)),
		fmt.Sprintf("🔓 Pode !libertar com pelo menos %d%% de chance, e se der errado a multa sai %d%% mais barata", percent(presidentRelease), percent(presidentFineDiscount)),
		fmt.Sprintf("🧟 Pode !reviver quem foi morto com pelo menos %d%% de chance", percent(presidentRevive)),
		fmt.Sprintf("🛡️ Segurança presidencial: quem tenta !sequestrar, !matar ou dar !golpe no presidente tem mais %d%% de chance de errar. Contra !roubar e !furtar não adianta", percent(presidentGuard)),
		fmt.Sprintf("⚖️ Imunidade: quando tentam prender o presidente (!prender, !denunciar ou pego no flagra), tem %d%% de chance de ele mostrar a imunidade e sair andando. Quem tava preso sai da cadeia ao tomar posse", percent(presidentImmunity)),
		"📜 Baixa !decreto: mexe no !trabalhar, no !daily, cobra imposto de renda, mexe na poupança, nos preços, no pix, na cadeia e na !fianca, mexe nos prêmios e na taxa do cassino, decreta toque de recolher e, uma vez no mandato, cobra imposto, dá auxílio, anistia, indulto, confisco e intervém nas !acoes. Manda !decreto pra ver a lista",
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
	now := time.Now()
	var (
		p      presidency
		found  bool
		vacant seat
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { p, found, err = b.Wallets.presidency(gctx, chat, now); return err })
	g.Go(func() (err error) { vacant, err = b.Wallets.seat(gctx, chat, now); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler presidência: %w", err)
	}
	if !found {
		text := fmt.Sprintf("🏛️ Ninguém é presidente agora. Manda !votar @pessoa pra abrir a eleição: a urna fica aberta %s e quem ganhar fica %s no cargo\n\n🎲 %s", core.FormatWait(electionWindow), core.FormatWait(termLength), vacancyNote(vacant.drawAt, now))
		return b.SendText(ctx, chat, text+p.decrees.leftoverText())
	}
	text := p.text()
	if value, running := b.impeachments.Load(chat.String()); running {
		text += "\n\n" + value.(*impeachment).partialText()
	}
	if value, running := b.coups.Load(chat.String()); running {
		text += "\n\n" + value.(*coup).partialText()
	}
	_, err := b.SendMentions(ctx, chat, text, []types.JID{p.president.JID})
	return err
}

func (e *election) partialText(candidates []candidate) string {
	title := "🗳️ *Eleição rolando!*"
	if len(e.runoff) > 0 {
		title = "🗳️ *Segundo turno rolando!* Só vale voto em " + core.JoinMentions(e.runoffJIDs()) + "."
	}
	lines := append([]string{fmt.Sprintf("%s A urna fecha em %s", title, core.FormatWait(time.Until(e.closes())))}, votesLines(candidates)...)
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
	if p.drawn {
		return fmt.Sprintf("🏛️ *Presidente:* @%s, que saiu no sorteio 🎲", p.president.JID.User)
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
	p.min += p.min * presidentPayBonus / 100
	p.max += p.max * presidentPayBonus / 100
	p.dailyBonus += presidentDailyBonus
	p.cordon += presidentGuard
	p.releaseChance = max(p.releaseChance, presidentRelease)
	p.reviveChance = max(p.reviveChance, presidentRevive)
	if p.jailTime == 0 {
		p.jailTime, p.arrestEvery = baseJail, presidentArrestEvery
	}
	return p
}

func immunityText(accuser, president types.JID) string {
	return fmt.Sprintf("⚖️ @%s mandou a viatura atrás de @%s, mas é o presidente: mostrou a imunidade e saiu pela porta da frente. O crime continua na ficha, então dá pra tentar de novo (a imunidade segura %d%% das vezes)", accuser.User, president.User, percent(presidentImmunity))
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
	dest := []any{&p.president.Key, &jid, &p.votes, &until, &impeachedAt, &coupAt, &p.coup, &p.drawn}
	for i := range used {
		dest = append(dest, &used[i])
	}
	for i := range levels {
		dest = append(dest, &levels[i])
	}
	err = s.db.QueryRowContext(ctx,
		"SELECT pessoa, jid, votos, mandato_ate, impeachment_em, golpe_em, golpista, sorteado, "+strings.Join(append(charges, columns...), ", ")+" FROM presidencias WHERE chat = ? AND (mandato_ate > ? OR renunciou = 1)",
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
	return s.inTx(ctx, func(tx *ledgerTx) error { return swearIn(ctx, tx, chat, acc, votes, false, now) })
}

func swearIn(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, votes int, drawn bool, now time.Time) error {
	if err := openWallet(ctx, tx, chat, acc); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx,
		`INSERT INTO presidencias (chat, pessoa, jid, votos, eleito_em, mandato_ate, sorteado) VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (chat) DO UPDATE SET pessoa = excluded.pessoa, jid = excluded.jid, votos = excluded.votos,
		 eleito_em = excluded.eleito_em, mandato_ate = excluded.mandato_ate, sorteado = excluded.sorteado, impeachment_em = 0, golpe_em = 0, golpista = 0, renunciou = 0, `+strings.Join(append(abuseChargeColumns(), decreeColumns()...), " = 0, ")+` = 0`,
		chat.String(), acc.Key, acc.JID.String(), votes, now.Unix(), now.Add(termLength).Unix(), drawn,
	)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE carteiras SET preso_ate = 0 WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key); err != nil {
		return err
	}
	return closeVacancy(ctx, tx, chat)
}

func (b *bot) workTerms(ctx context.Context, chat types.JID, acc core.Account, now time.Time) (job Profession, p presidency, err error) {
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { job, err = b.ProfessionOf(gctx, chat, acc); return err })
	g.Go(func() (err error) { p, _, err = b.Wallets.presidency(gctx, chat, now); return err })
	err = g.Wait()
	return job, p, err
}
