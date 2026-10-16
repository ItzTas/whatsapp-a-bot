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
	"golang.org/x/sync/errgroup"
)

const (
	trialVoteWindow     = 2 * time.Minute
	trialVoteLabel      = "2 minutos"
	trialGuiltyOption   = "⚖️ Culpado"
	trialInnocentOption = "🕊️ Inocente"
	trialFinePercent    = 20
	trialFineCeiling    = 2000
	trialCostsPercent   = 8
	trialCostsCeiling   = 600
	trialEvery          = 45 * time.Minute
	trialCooldownColumn = "processo_em"
	trialBusy           = "⚖️ Já tem um processo rolando aqui, espera o juiz bater o martelo"
	trialKind           = "processar"
	trialJuryLookups    = 8
)

type trialScript struct {
	Hearing  string `json:"julgamento"`
	Guilty   string `json:"culpado"`
	Innocent string `json:"inocente"`
}

type trialSides struct {
	economy      bool
	accuserJob   profession
	defendantJob profession
}

type trialCase struct {
	*pollAnswer
	accuser   types.JID
	defendant types.JID
	sides     trialSides
	script    trialScript
	debt      bool
	stateMu   sync.Mutex
	over      bool
	chat      types.JID
	pollID    types.MessageID
	closes    time.Time
	save      func()
}

type trialState struct {
	Chat         types.JID       `json:"chat"`
	PollID       types.MessageID `json:"pollId"`
	Accuser      types.JID       `json:"accuser"`
	Defendant    types.JID       `json:"defendant"`
	Economy      bool            `json:"economy"`
	AccuserJob   string          `json:"accuserJob"`
	DefendantJob string          `json:"defendantJob"`
	Script       trialScript     `json:"script"`
	Debt         bool            `json:"debt,omitempty"`
	Poll         storedPoll      `json:"poll"`
	Closes       time.Time       `json:"closes"`
}

type trialBill struct {
	payer    types.JID
	payee    types.JID
	payerJob profession
	payeeJob profession
	percent  int64
	ceiling  int64
	label    string
	debt     bool
}

func (b *bot) processar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return b.sendText(ctx, chat, "⚖️ O !processar só funciona em grupo")
	}
	mentions := mentionedJIDs(evt.Message)
	if len(mentions) == 0 {
		return b.replyTo(ctx, evt, "⚖️ Marca quem vai pro banco dos réus: !processar @pessoa. "+grievanceRule())
	}
	accuser, defendant := b.account(ctx, evt.Info.Sender), b.account(ctx, mentions[0])
	if accuser.key == defendant.key {
		return b.replyTo(ctx, evt, "🤨 Não dá pra processar a si mesmo, procura um terapeuta")
	}
	if _, busy := b.trialChats.LoadOrStore(chat.String(), struct{}{}); busy {
		return b.replyTo(ctx, evt, trialBusy)
	}

	opened, err := b.fileLawsuit(ctx, evt, chat, accuser, defendant)
	if !opened {
		b.trialChats.Delete(chat.String())
	}
	return err
}

func (b *bot) fileLawsuit(ctx context.Context, evt *events.Message, chat types.JID, accuser, defendant account) (bool, error) {
	now := time.Now()
	var (
		complaint grievance
		found     bool
		owed      *debt
		sides     = trialSides{economy: true}
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) {
		complaint, found, err = b.wallets.grievance(gctx, chat, accuser, defendant, now)
		return err
	})
	g.Go(func() (err error) {
		sides.accuserJob, sides.defendantJob, owed, err = b.rivalsWithDebt(gctx, chat, accuser, defendant)
		return err
	})
	if err := g.Wait(); err != nil {
		return false, fmt.Errorf("preparar !processar: %w", err)
	}
	if !found && owed == nil {
		return false, b.replyTo(ctx, evt, "📂 Você não tem queixa nenhuma contra essa pessoa. "+grievanceRule())
	}
	wait, err := b.wallets.stamp(ctx, chat, accuser, sides.accuserJob.trialCooldown(), now)
	if err != nil {
		return false, fmt.Errorf("carimbar !processar: %w", err)
	}
	if wait > 0 {
		return false, b.replyTo(ctx, evt, fmt.Sprintf("⏳ O fórum ainda tá analisando teu último processo. Espera %s pra abrir outro", formatWait(wait)))
	}

	round := &trialCase{pollAnswer: newPollAnswer(trialGuiltyOption), accuser: accuser.jid, defendant: defendant.jid, sides: sides}
	motive := round.charge(complaint, found, owed, now)
	if err := b.openTrial(ctx, chat, round, motive); err != nil {
		b.releaseTrialCooldown(ctx, chat, accuser)
		return false, err
	}
	if round.debt {
		return true, nil
	}
	if err := b.wallets.dropGrievance(ctx, chat, accuser, defendant); err != nil {
		b.client.Log.Warnf("apagar a queixa de %s contra %s: %v", accuser.key, defendant.key, err)
	}
	return true, nil
}

func (r *trialCase) charge(complaint grievance, found bool, owed *debt, now time.Time) string {
	if found {
		r.script = complaint.script(r.accuser, r.defendant, now)
		return complaint.kind.motive
	}
	r.debt, r.script = true, owed.script(now)
	return debtMotive
}

func (b *bot) releaseTrialCooldown(ctx context.Context, chat types.JID, acc account) {
	if err := b.wallets.clearCooldown(ctx, chat, acc, trialCooldownColumn); err != nil {
		b.client.Log.Warnf("liberar a espera do !processar de %s: %v", acc.key, err)
	}
}

func (s trialSides) extraVotes() (guilty, innocent int) {
	lean := func(edge int, own, other *int) {
		if edge >= 0 {
			*own += edge
			return
		}
		*other -= edge
	}
	lean(s.accuserJob.trialEdge, &guilty, &innocent)
	lean(s.defendantJob.trialEdge, &innocent, &guilty)
	return guilty, innocent
}

func (r *trialCase) edgeLines() []string {
	parties := []struct {
		who   types.JID
		job   profession
		own   string
		other string
	}{
		{r.accuser, r.sides.accuserJob, "culpado", "inocente"},
		{r.defendant, r.sides.defendantJob, "inocente", "culpado"},
	}
	var lines []string
	for _, p := range parties {
		edge, side := p.job.trialEdge, p.own
		if edge == 0 {
			continue
		}
		if edge < 0 {
			edge, side = -edge, p.other
		}
		lines = append(lines, fmt.Sprintf("%s @%s %s: +%s pra %s", p.job.emoji, p.who.User, p.job.title, countLabel(edge, "voto", "votos"), side))
	}
	return lines
}

func (b *bot) openTrial(ctx context.Context, chat types.JID, round *trialCase, motive string) error {
	if motive == "" {
		motive = "não informado, o que já é suspeito"
	}
	lines := []string{
		"⚖️ *TRIBUNAL DO GRUPO*",
		"",
		fmt.Sprintf("👤 Réu: @%s", round.defendant.User),
		fmt.Sprintf("📢 Autor da ação: @%s", round.accuser.User),
		"📄 Motivo: " + motive,
	}
	if edges := round.edgeLines(); len(edges) > 0 {
		lines = append(lines, "", "🎓 *Peso das profissões no júri*")
		lines = append(lines, edges...)
	}
	lines = append(lines,
		"",
		round.script.Hearing,
		"",
		fmt.Sprintf("_O júri é o grupo: vota na enquete, o juiz bate o martelo em %s_ ⏳", trialVoteLabel),
	)
	if _, err := b.sendMentions(ctx, chat, strings.Join(lines, "\n"), []types.JID{round.defendant, round.accuser}); err != nil {
		return err
	}
	resp, err := b.client.SendMessage(ctx, chat, b.client.BuildPollCreation("🧑‍⚖️ O réu é culpado ou inocente?", []string{trialGuiltyOption, trialInnocentOption}, 1))
	if err != nil {
		return err
	}
	round.chat, round.pollID, round.closes = chat, resp.ID, time.Now().Add(trialVoteWindow)
	b.trackTrial(ctx, round)
	return nil
}

func (b *bot) trackTrial(ctx context.Context, round *trialCase) {
	round.save = func() { b.keepTrial(ctx, round) }
	round.stateMu.Lock()
	b.trialPolls.Store(round.pollID, round)
	round.save()
	round.stateMu.Unlock()
	go b.closeTrial(ctx, round.chat, round.pollID, round)
}

func (b *bot) keepTrial(ctx context.Context, round *trialCase) {
	state := trialState{
		Chat: round.chat, PollID: round.pollID, Accuser: round.accuser, Defendant: round.defendant,
		Economy: round.sides.economy, AccuserJob: round.sides.accuserJob.name, DefendantJob: round.sides.defendantJob.name,
		Script: round.script, Debt: round.debt, Poll: round.stored(), Closes: round.closes,
	}
	b.keepGame(ctx, trialKind, round.chat.String(), state, round.closes)
}

func (r *trialCase) vote(voter types.JID, selected [][]byte) {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	if r.over {
		return
	}
	r.pollAnswer.vote(voter, selected)
	r.save()
}

func (b *bot) endTrial(ctx context.Context, round *trialCase) {
	round.stateMu.Lock()
	defer round.stateMu.Unlock()
	round.over = true
	b.dropGame(ctx, trialKind, round.chat.String())
}

func (b *bot) restoreTrial(ctx context.Context, saved savedGame) error {
	var state trialState
	if err := json.Unmarshal(saved.state, &state); err != nil {
		return fmt.Errorf("ler processo salvo: %w", err)
	}
	if _, busy := b.trialChats.LoadOrStore(state.Chat.String(), struct{}{}); busy {
		return fmt.Errorf("já tem outro processo rolando em %s", state.Chat)
	}
	round := &trialCase{
		pollAnswer: restorePollAnswer(state.Poll), accuser: state.Accuser, defendant: state.Defendant,
		sides: restoreTrialSides(state), script: state.Script, debt: state.Debt, chat: state.Chat, pollID: state.PollID, closes: state.Closes,
	}
	b.trackTrial(ctx, round)
	return nil
}

func restoreTrialSides(state trialState) trialSides {
	if !state.Economy {
		return trialSides{}
	}
	accuserJob, _ := professionNamed(state.AccuserJob)
	defendantJob, _ := professionNamed(state.DefendantJob)
	return trialSides{economy: true, accuserJob: accuserJob, defendantJob: defendantJob}
}

func (b *bot) closeTrial(ctx context.Context, chat types.JID, pollID types.MessageID, round *trialCase) {
	defer b.trialChats.Delete(chat.String())
	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Until(round.closes)):
	}
	b.trialPolls.Delete(pollID)
	b.endTrial(ctx, round)

	people := []types.JID{round.defendant, round.accuser}
	if _, err := b.sendMentions(ctx, chat, b.trialVerdict(ctx, chat, round), people); err != nil {
		b.client.Log.Errorf("ler a sentença do tribunal: %v", err)
	}
}

func (b *bot) trialVerdict(ctx context.Context, chat types.JID, round *trialCase) string {
	jury := round.ballots()
	if len(jury) == 0 {
		return fmt.Sprintf("🦗 Ninguém do júri apareceu. O processo de @%s contra @%s foi arquivado por falta de quórum", round.accuser.User, round.defendant.User)
	}

	votedGuilty, votedInnocent, weighted := juryTally(jury, b.juryJobs(ctx, chat, round.sides.economy, jury))
	extraGuilty, extraInnocent := round.sides.extraVotes()
	guilty, innocent := votedGuilty+extraGuilty, votedInnocent+extraInnocent
	header := fmt.Sprintf("🧑‍⚖️ *Sentença*\nJúri: %d culpado, %d inocente\n", votedGuilty, votedInnocent)
	if len(weighted) > 0 {
		header += strings.Join(weighted, "\n") + "\n"
	}
	if edges := round.edgeLines(); len(edges) > 0 {
		header += strings.Join(edges, "\n") + fmt.Sprintf("\nPlacar final: %d culpado, %d inocente\n", guilty, innocent)
	}

	text, bill := round.sentence(guilty > innocent)
	text = header + text
	if !round.sides.economy {
		return text
	}
	return appendLine(text, b.trialPenalty(ctx, chat, bill))
}

func (b *bot) juryJobs(ctx context.Context, chat types.JID, economy bool, jury []pollVote) []profession {
	jobs := make([]profession, len(jury))
	if !economy {
		return jobs
	}
	var g errgroup.Group
	g.SetLimit(trialJuryLookups)
	for i, v := range jury {
		g.Go(func() (err error) { jobs[i], err = b.professionOf(ctx, chat, b.account(ctx, v.voter)); return err })
	}
	if err := g.Wait(); err != nil {
		b.client.Log.Warnf("ler as profissões do júri do !processar, quem falhou vota com peso 1: %v", err)
	}
	return jobs
}

func juryTally(jury []pollVote, jobs []profession) (guilty, innocent int, weighted []string) {
	for i, v := range jury {
		weight := jobs[i].ballotWeight()
		if weight > 1 {
			weighted = append(weighted, fmt.Sprintf("%s Voto de quem %s vale %d", jobs[i].emoji, jobs[i].title, weight))
		}
		if v.correct {
			guilty += weight
			continue
		}
		innocent += weight
	}
	return guilty, innocent, weighted
}

func (r *trialCase) sentence(convicted bool) (string, trialBill) {
	if convicted {
		bill := trialBill{
			payer: r.defendant, payee: r.accuser, payerJob: r.sides.defendantJob, payeeJob: r.sides.accuserJob,
			percent: trialFinePercent, ceiling: trialFineCeiling, label: "💸 Indenização",
		}
		if r.debt {
			bill.label, bill.debt = "🦈 Cobrança da dívida", true
		}
		return fmt.Sprintf("\n🔨 *CULPADO!* @%s foi condenado(a).\n%s", r.defendant.User, r.script.Guilty), bill
	}
	return fmt.Sprintf("\n🕊️ *INOCENTE!* @%s foi absolvido(a).\n%s", r.defendant.User, r.script.Innocent), trialBill{
		payer: r.accuser, payee: r.defendant, payerJob: r.sides.accuserJob, payeeJob: r.sides.defendantJob,
		percent: trialCostsPercent, ceiling: trialCostsCeiling, label: "💸 Custas do processo",
	}
}

func (b *bot) trialPenalty(ctx context.Context, chat types.JID, bill trialBill) string {
	if bill.debt {
		return b.courtDebtCollection(ctx, chat, bill)
	}
	payer, payee := b.account(ctx, bill.payer), b.account(ctx, bill.payee)
	balance, _, err := b.wallets.balance(ctx, chat, payer)
	if err != nil {
		b.client.Log.Errorf("preparar %s do !processar: %v", bill.label, err)
		return ""
	}

	amount := min(balance*bill.percent/100, bill.ceiling) * (100 + bill.payeeJob.trialBonus) / 100 * (100 - bill.payerJob.trialDiscount) / 100
	broke := fmt.Sprintf("%s: @%s tá liso e não tem nem como pagar 🪹", bill.label, bill.payer.User)
	if amount <= 0 {
		return broke
	}
	err = b.wallets.transfer(ctx, chat, payer, payee, amount)
	if errors.Is(err, errNoFunds) {
		return broke
	}
	if err != nil {
		b.client.Log.Errorf("cobrar %s de %s: %v", bill.label, bill.payer, err)
		return ""
	}

	text := fmt.Sprintf("%s: @%s pagou %s pra @%s", bill.label, bill.payer.User, coins(amount), bill.payee.User)
	if bill.payeeJob.trialBonus > 0 {
		text += fmt.Sprintf(". Veio %d%% a mais porque @%s %s", bill.payeeJob.trialBonus, bill.payee.User, bill.payeeJob.title)
	}
	if bill.payerJob.trialDiscount > 0 {
		text += fmt.Sprintf(". Saiu %d%% mais barato porque @%s %s", bill.payerJob.trialDiscount, bill.payer.User, bill.payerJob.title)
	}
	return text
}

func (s *walletStore) stamp(ctx context.Context, chat types.JID, acc account, cd cooldown, now time.Time) (wait time.Duration, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) (txErr error) {
		wait, txErr = stampCooldown(ctx, tx, chat, acc, cd, now)
		return txErr
	})
	return wait, err
}

func (s *walletStore) clearCooldown(ctx context.Context, chat types.JID, acc account, column string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE carteiras SET "+column+" = 0 WHERE chat = ? AND pessoa = ?", chat.String(), acc.key)
	return err
}

const (
	grievanceWindow = 24 * time.Hour
	reportGrievance = "denuncia"
	arrestGrievance = "prisao"
)

type grievanceKind struct {
	motive   string
	hearing  string
	guilty   string
	innocent string
}

var grievanceKinds = map[string]grievanceKind{
	reportGrievance: {
		motive:   "denúncia caluniosa",
		hearing:  "📞 *Acusação:* @%[1]s ligou pro 190 e dedurou @%[2]s %[3]s atrás, mas a ficha de @%[2]s tava limpinha. A polícia perdeu a viagem e @%[2]s ficou com fama de bandido no grupo",
		guilty:   "Pena: indenização por denúncia caluniosa e o título de X-9 mentiroso oficial do grupo",
		innocent: "O júri achou que foi engano honesto, e o caso foi pro arquivo",
	},
	arrestGrievance: {
		motive:   "tentativa de prisão sem crime",
		hearing:  "🚔 *Acusação:* @%[1]s tentou prender @%[2]s %[3]s atrás sem crime nenhum na ficha. Abuso de autoridade na frente de todo mundo",
		guilty:   "Pena: indenização por abuso de autoridade e pedir desculpas com o distintivo na mão",
		innocent: "O júri entendeu que foi só excesso de zelo, e o caso foi pro arquivo",
	},
}

type grievance struct {
	kind grievanceKind
	at   time.Time
}

func (g grievance) script(accuser, defendant types.JID, now time.Time) trialScript {
	return trialScript{
		Hearing:  fmt.Sprintf(g.kind.hearing, defendant.User, accuser.User, formatWait(now.Sub(g.at))),
		Guilty:   g.kind.guilty,
		Innocent: g.kind.innocent,
	}
}

func grievanceNote(victim, accused types.JID) string {
	return fmt.Sprintf("\n⚖️ Agora @%s pode !processar @%s nas próximas %s", victim.User, accused.User, formatWait(grievanceWindow))
}

func grievanceRule() string {
	return fmt.Sprintf("Só dá pra processar quem te denunciou ou tentou te prender com a ficha limpa nas últimas %s, ou quem te deve no !agiota e passou do prazo", formatWait(grievanceWindow))
}

const debtMotive = "calote no agiota"

func (d debt) script(now time.Time) trialScript {
	return trialScript{
		Hearing: fmt.Sprintf("🦈 *Acusação:* @%s pegou moedas com @%s no !agiota e não pagou. O prazo venceu há %s e a dívida, com os juros, já tá em %s",
			d.debtor.jid.User, d.lender.jid.User, formatWait(now.Sub(d.due)), coins(d.amount)),
		Guilty:   "Pena: pagar o agiota na hora, com o que tiver no saldo, e ficar com fama de caloteiro",
		Innocent: "O júri teve dó do devedor, e o agiota vai ter que cobrar de outro jeito",
	}
}

func (b *bot) courtDebtCollection(ctx context.Context, chat types.JID, bill trialBill) string {
	debtor, lender := b.account(ctx, bill.payer), b.account(ctx, bill.payee)
	balance, _, err := b.wallets.balance(ctx, chat, debtor)
	if err != nil {
		b.client.Log.Errorf("preparar %s do !processar: %v", bill.label, err)
		return ""
	}
	broke := fmt.Sprintf("%s: @%s tá liso e não tem nem como pagar 🪹 A dívida continua de pé", bill.label, bill.payer.User)
	if balance <= 0 {
		return broke
	}

	left, paid, err := b.wallets.repayDebt(ctx, chat, debtor, lender, balance, time.Now())
	if errors.Is(err, errNoDebt) {
		return fmt.Sprintf("%s: @%s já tinha quitado a dívida antes do martelo bater", bill.label, bill.payer.User)
	}
	if errors.Is(err, errNoFunds) {
		return broke
	}
	if err != nil {
		b.client.Log.Errorf("cobrar a dívida de %s no !processar: %v", bill.payer, err)
		return ""
	}

	text := fmt.Sprintf("%s: o juiz tirou %s do saldo de @%s e entregou pra @%s", bill.label, coins(paid), bill.payer.User, bill.payee.User)
	if left.amount == 0 {
		return text + ", e a dívida tá quitada 🤝"
	}
	return text + fmt.Sprintf(". Ainda falta %s", coins(left.amount))
}

func (s *walletStore) fileGrievance(ctx context.Context, chat types.JID, victim, accused account, kind string, now time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM queixas WHERE chat = ? AND em < ?", chat.String(), now.Add(-grievanceWindow).Unix()); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx,
			`INSERT INTO queixas (chat, vitima, acusado, motivo, em) VALUES (?, ?, ?, ?, ?)
			 ON CONFLICT (chat, vitima, acusado) DO UPDATE SET motivo = excluded.motivo, em = excluded.em`,
			chat.String(), victim.key, accused.key, kind, now.Unix(),
		)
		return err
	})
}

func (s *walletStore) grievance(ctx context.Context, chat types.JID, victim, accused account, now time.Time) (grievance, bool, error) {
	var (
		kind string
		at   int64
	)
	err := s.db.QueryRowContext(ctx,
		"SELECT motivo, em FROM queixas WHERE chat = ? AND vitima = ? AND acusado = ? AND em >= ?",
		chat.String(), victim.key, accused.key, now.Add(-grievanceWindow).Unix(),
	).Scan(&kind, &at)
	if errors.Is(err, sql.ErrNoRows) {
		return grievance{}, false, nil
	}
	if err != nil {
		return grievance{}, false, err
	}
	known, ok := grievanceKinds[kind]
	return grievance{kind: known, at: time.Unix(at, 0)}, ok, nil
}

func (s *walletStore) dropGrievance(ctx context.Context, chat types.JID, victim, accused account) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM queixas WHERE chat = ? AND vitima = ? AND acusado = ?", chat.String(), victim.key, accused.key)
	return err
}
