package economy

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	trialVoteWindow     = 2 * time.Minute
	trialVoteLabel      = "2 minutos"
	TrialGuiltyOption   = "⚖️ Culpado"
	trialInnocentOption = "🕊️ Inocente"
	trialFinePercent    = 20
	trialFineCeiling    = 2000
	trialCostsPercent   = 8
	trialCostsCeiling   = 600
	trialEvery          = 45 * time.Minute
	trialCooldownColumn = "processo_em"
	TrialBusy           = "⚖️ Já tem um processo rolando aqui, espera o juiz bater o martelo"
	trialKind           = "processar"
	trialJuryLookups    = 8
)

type TrialScript struct {
	Hearing  string `json:"julgamento"`
	Guilty   string `json:"culpado"`
	Innocent string `json:"inocente"`
}

type trialSides struct {
	economy      bool
	accuserJob   Profession
	defendantJob Profession
}

type TrialCase struct {
	*core.PollAnswer
	Accuser   types.JID
	Defendant types.JID
	sides     trialSides
	Script    TrialScript
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
	Script       TrialScript     `json:"script"`
	Debt         bool            `json:"debt,omitempty"`
	Poll         core.StoredPoll `json:"poll"`
	Closes       time.Time       `json:"closes"`
}

type trialBill struct {
	payer    types.JID
	payee    types.JID
	payerJob Profession
	payeeJob Profession
	percent  int64
	ceiling  int64
	label    string
	debt     bool
}

func (b *bot) processar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return b.SendText(ctx, chat, "⚖️ O !processar só funciona em grupo")
	}
	mentions := core.MentionedJIDs(evt.Message)
	if len(mentions) == 0 {
		return b.ReplyTo(ctx, evt, "⚖️ Marca quem vai pro banco dos réus: !processar @pessoa. "+grievanceRule())
	}
	accuser, defendant := b.Account(ctx, evt.Info.Sender), b.Account(ctx, mentions[0])
	if accuser.Key == defendant.Key {
		return b.ReplyTo(ctx, evt, "🤨 Não dá pra processar a si mesmo, procura um terapeuta")
	}
	if _, busy := b.TrialChats.LoadOrStore(chat.String(), struct{}{}); busy {
		return b.ReplyTo(ctx, evt, TrialBusy)
	}

	opened, err := b.fileLawsuit(ctx, evt, chat, accuser, defendant)
	if !opened {
		b.TrialChats.Delete(chat.String())
	}
	return err
}

func (b *bot) fileLawsuit(ctx context.Context, evt *events.Message, chat types.JID, accuser, defendant core.Account) (bool, error) {
	now := time.Now()
	var (
		complaint grievance
		found     bool
		owed      *debt
		sides     = trialSides{economy: true}
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) {
		complaint, found, err = b.Wallets.grievance(gctx, chat, accuser, defendant, now)
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
		return false, b.ReplyTo(ctx, evt, "📂 Você não tem queixa nenhuma contra essa pessoa. "+grievanceRule())
	}
	wait, err := b.Wallets.stamp(ctx, chat, accuser, sides.accuserJob.trialCooldown(), now)
	if err != nil {
		return false, fmt.Errorf("carimbar !processar: %w", err)
	}
	if wait > 0 {
		return false, b.ReplyTo(ctx, evt, fmt.Sprintf("⏳ O fórum ainda tá analisando teu último processo. Espera %s pra abrir outro", core.FormatWait(wait)))
	}

	round := &TrialCase{PollAnswer: core.NewPollAnswer(TrialGuiltyOption), Accuser: accuser.JID, Defendant: defendant.JID, sides: sides}
	motive := round.charge(complaint, found, owed, now)
	if err := b.OpenTrial(ctx, chat, round, motive); err != nil {
		b.releaseTrialCooldown(ctx, chat, accuser)
		return false, err
	}
	if round.debt {
		return true, nil
	}
	if err := b.Wallets.dropGrievance(ctx, chat, accuser, defendant); err != nil {
		b.Client.Log.Warnf("apagar a queixa de %s contra %s: %v", accuser.Key, defendant.Key, err)
	}
	return true, nil
}

func (r *TrialCase) charge(complaint grievance, found bool, owed *debt, now time.Time) string {
	if found {
		r.Script = complaint.script(r.Accuser, r.Defendant, now)
		return complaint.kind.motive
	}
	r.debt, r.Script = true, owed.script(now)
	return debtMotive
}

func (b *bot) releaseTrialCooldown(ctx context.Context, chat types.JID, acc core.Account) {
	if err := b.Wallets.clearCooldown(ctx, chat, acc, trialCooldownColumn); err != nil {
		b.Client.Log.Warnf("liberar a espera do !processar de %s: %v", acc.Key, err)
	}
}

func (s trialSides) extraVotes(debt bool) (guilty, innocent int) {
	lean := func(edge int, own, other *int) {
		if edge >= 0 {
			*own += edge
			return
		}
		*other -= edge
	}
	lean(s.accuserJob.trialEdge, &guilty, &innocent)
	lean(s.defendantJob.trialEdge, &innocent, &guilty)
	if debt {
		guilty += s.accuserJob.collectVotes
	}
	return guilty, innocent
}

func (r *TrialCase) edgeLines() []string {
	parties := []struct {
		who   types.JID
		job   Profession
		own   string
		other string
	}{
		{r.Accuser, r.sides.accuserJob, "culpado", "inocente"},
		{r.Defendant, r.sides.defendantJob, "inocente", "culpado"},
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
		lines = append(lines, fmt.Sprintf("%s @%s %s: +%s pra %s", p.job.emoji, p.who.User, p.job.title, core.CountLabel(edge, "voto", "votos"), side))
	}
	if votes := r.sides.accuserJob.collectVotes; r.debt && votes > 0 {
		lines = append(lines, fmt.Sprintf("🦈 @%s cobra calote de profissão: +%s pra culpado", r.Accuser.User, core.CountLabel(votes, "voto", "votos")))
	}
	return lines
}

func (b *bot) OpenTrial(ctx context.Context, chat types.JID, round *TrialCase, motive string) error {
	if motive == "" {
		motive = "não informado, o que já é suspeito"
	}
	lines := []string{
		"⚖️ *TRIBUNAL DO GRUPO*",
		"",
		fmt.Sprintf("👤 Réu: @%s", round.Defendant.User),
		fmt.Sprintf("📢 Autor da ação: @%s", round.Accuser.User),
		"📄 Motivo: " + motive,
	}
	if edges := round.edgeLines(); len(edges) > 0 {
		lines = append(lines, "", "🎓 *Peso das profissões no júri*")
		lines = append(lines, edges...)
	}
	lines = append(lines,
		"",
		round.Script.Hearing,
		"",
		fmt.Sprintf("_O júri é o grupo: vota na enquete, o juiz bate o martelo em %s_ ⏳", trialVoteLabel),
	)
	if _, err := b.SendMentions(ctx, chat, strings.Join(lines, "\n"), []types.JID{round.Defendant, round.Accuser}); err != nil {
		return err
	}
	resp, err := b.Client.SendMessage(ctx, chat, b.Client.BuildPollCreation("🧑‍⚖️ O réu é culpado ou inocente?", []string{TrialGuiltyOption, trialInnocentOption}, 1))
	if err != nil {
		return err
	}
	round.chat, round.pollID, round.closes = chat, resp.ID, time.Now().Add(trialVoteWindow)
	b.trackTrial(ctx, round)
	return nil
}

func (b *bot) trackTrial(ctx context.Context, round *TrialCase) {
	round.save = func() { b.keepTrial(ctx, round) }
	round.stateMu.Lock()
	b.trialPolls.Store(round.pollID, round)
	round.save()
	round.stateMu.Unlock()
	go b.closeTrial(ctx, round.chat, round.pollID, round)
}

func (b *bot) keepTrial(ctx context.Context, round *TrialCase) {
	state := trialState{
		Chat: round.chat, PollID: round.pollID, Accuser: round.Accuser, Defendant: round.Defendant,
		Economy: round.sides.economy, AccuserJob: round.sides.accuserJob.name, DefendantJob: round.sides.defendantJob.name,
		Script: round.Script, Debt: round.debt, Poll: round.Stored(), Closes: round.closes,
	}
	b.KeepGame(ctx, trialKind, round.chat.String(), state, round.closes)
}

func (r *TrialCase) Vote(voter types.JID, selected [][]byte) {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	if r.over {
		return
	}
	r.PollAnswer.Vote(voter, selected)
	r.save()
}

func (b *bot) endTrial(ctx context.Context, round *TrialCase) {
	round.stateMu.Lock()
	defer round.stateMu.Unlock()
	round.over = true
	b.DropGame(ctx, trialKind, round.chat.String())
}

func (b *bot) restoreTrial(ctx context.Context, saved core.SavedGame) error {
	var state trialState
	if err := json.Unmarshal(saved.State, &state); err != nil {
		return fmt.Errorf("ler processo salvo: %w", err)
	}
	if _, busy := b.TrialChats.LoadOrStore(state.Chat.String(), struct{}{}); busy {
		return fmt.Errorf("já tem outro processo rolando em %s", state.Chat)
	}
	round := &TrialCase{
		PollAnswer: core.RestorePollAnswer(state.Poll), Accuser: state.Accuser, Defendant: state.Defendant,
		sides: restoreTrialSides(state), Script: state.Script, debt: state.Debt, chat: state.Chat, pollID: state.PollID, closes: state.Closes,
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

func (b *bot) closeTrial(ctx context.Context, chat types.JID, pollID types.MessageID, round *TrialCase) {
	defer b.TrialChats.Delete(chat.String())
	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Until(round.closes)):
	}
	b.trialPolls.Delete(pollID)
	b.endTrial(ctx, round)

	people := []types.JID{round.Defendant, round.Accuser}
	if _, err := b.SendMentions(ctx, chat, b.trialVerdict(ctx, chat, round), people); err != nil {
		b.Client.Log.Errorf("ler a sentença do tribunal: %v", err)
	}
}

func (b *bot) trialVerdict(ctx context.Context, chat types.JID, round *TrialCase) string {
	accuserPaid, defendantPaid := b.trialBribes(ctx, chat, round)
	if accuserPaid != defendantPaid {
		return b.riggedVerdict(ctx, chat, round, accuserPaid)
	}
	jury := round.Ballots()
	if len(jury) == 0 {
		return fmt.Sprintf("🦗 Ninguém do júri apareceu. O processo de @%s contra @%s foi arquivado por falta de quórum", round.Accuser.User, round.Defendant.User)
	}

	votedGuilty, votedInnocent, weighted := juryTally(jury, b.juryJobs(ctx, chat, round.sides.economy, jury))
	extraGuilty, extraInnocent := round.sides.extraVotes(round.debt)
	guilty, innocent := votedGuilty+extraGuilty, votedInnocent+extraInnocent
	header := fmt.Sprintf("🧑‍⚖️ *Sentença*\nJúri: %d culpado, %d inocente\n", votedGuilty, votedInnocent)
	if accuserPaid {
		header += "💰 Os dois lados molharam a mão do juiz, ele embolsou tudo e deixou o júri decidir\n"
	}
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
	return core.AppendLine(text, b.trialPenalty(ctx, chat, bill))
}

func (b *bot) juryJobs(ctx context.Context, chat types.JID, economy bool, jury []core.PollVote) []Profession {
	jobs := make([]Profession, len(jury))
	if !economy {
		return jobs
	}
	var g errgroup.Group
	g.SetLimit(trialJuryLookups)
	for i, v := range jury {
		g.Go(func() (err error) { jobs[i], err = b.ProfessionOf(ctx, chat, b.Account(ctx, v.Voter)); return err })
	}
	if err := g.Wait(); err != nil {
		b.Client.Log.Warnf("ler as profissões do júri do !processar, quem falhou vota com peso 1: %v", err)
	}
	return jobs
}

func juryTally(jury []core.PollVote, jobs []Profession) (guilty, innocent int, weighted []string) {
	for i, v := range jury {
		weight := jobs[i].ballotWeight()
		if weight > 1 {
			weighted = append(weighted, fmt.Sprintf("%s Voto de quem %s vale %d", jobs[i].emoji, jobs[i].title, weight))
		}
		if v.Correct {
			guilty += weight
			continue
		}
		innocent += weight
	}
	return guilty, innocent, weighted
}

func (r *TrialCase) sentence(convicted bool) (string, trialBill) {
	if convicted {
		bill := trialBill{
			payer: r.Defendant, payee: r.Accuser, payerJob: r.sides.defendantJob, payeeJob: r.sides.accuserJob,
			percent: trialFinePercent, ceiling: trialFineCeiling, label: "💸 Indenização",
		}
		if r.debt {
			bill.label, bill.debt = "🦈 Cobrança da dívida", true
		}
		return fmt.Sprintf("\n🔨 *CULPADO!* @%s foi condenado(a).\n%s", r.Defendant.User, r.Script.Guilty), bill
	}
	return fmt.Sprintf("\n🕊️ *INOCENTE!* @%s foi absolvido(a).\n%s", r.Defendant.User, r.Script.Innocent), trialBill{
		payer: r.Accuser, payee: r.Defendant, payerJob: r.sides.accuserJob, payeeJob: r.sides.defendantJob,
		percent: trialCostsPercent, ceiling: trialCostsCeiling, label: "💸 Custas do processo",
	}
}

func (b *bot) trialPenalty(ctx context.Context, chat types.JID, bill trialBill) string {
	if bill.debt {
		return b.courtDebtCollection(ctx, chat, bill)
	}
	payer, payee := b.Account(ctx, bill.payer), b.Account(ctx, bill.payee)
	balance, _, err := b.Wallets.Balance(ctx, chat, payer)
	if err != nil {
		b.Client.Log.Errorf("preparar %s do !processar: %v", bill.label, err)
		return ""
	}

	amount := min(balance*bill.percent/100, bill.ceiling) * (100 + bill.payeeJob.trialBonus) / 100 * (100 - bill.payerJob.trialDiscount) / 100
	broke := fmt.Sprintf("%s: @%s tá liso e não tem nem como pagar 🪹", bill.label, bill.payer.User)
	if amount <= 0 {
		return broke
	}
	err = b.Wallets.Transfer(ctx, chat, payer, payee, amount)
	if errors.Is(err, ErrNoFunds) {
		return broke
	}
	if err != nil {
		b.Client.Log.Errorf("cobrar %s de %s: %v", bill.label, bill.payer, err)
		return ""
	}

	text := fmt.Sprintf("%s: @%s pagou %s pra @%s", bill.label, bill.payer.User, Coins(amount), bill.payee.User)
	if bill.payeeJob.trialBonus > 0 {
		text += fmt.Sprintf(". Veio %d%% a mais porque @%s %s", bill.payeeJob.trialBonus, bill.payee.User, bill.payeeJob.title)
	}
	if bill.payerJob.trialDiscount > 0 {
		text += fmt.Sprintf(". Saiu %d%% mais barato porque @%s %s", bill.payerJob.trialDiscount, bill.payer.User, bill.payerJob.title)
	}
	return text
}

func (s *walletStore) stamp(ctx context.Context, chat types.JID, acc core.Account, cd cooldown, now time.Time) (wait time.Duration, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) (txErr error) {
		wait, txErr = stampCooldown(ctx, tx, chat, acc, cd, now)
		return txErr
	})
	return wait, err
}

func (s *walletStore) clearCooldown(ctx context.Context, chat types.JID, acc core.Account, column string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE carteiras SET "+column+" = 0 WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key)
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

func (g grievance) script(accuser, defendant types.JID, now time.Time) TrialScript {
	return TrialScript{
		Hearing:  fmt.Sprintf(g.kind.hearing, defendant.User, accuser.User, core.FormatWait(now.Sub(g.at))),
		Guilty:   g.kind.guilty,
		Innocent: g.kind.innocent,
	}
}

func grievanceNote(victim, accused types.JID) string {
	return fmt.Sprintf("\n⚖️ Agora @%s pode !processar @%s nas próximas %s", victim.User, accused.User, core.FormatWait(grievanceWindow))
}

func grievanceRule() string {
	return fmt.Sprintf("Só dá pra processar quem te denunciou ou tentou te prender com a ficha limpa nas últimas %s, ou quem te deve no !agiota e passou do prazo", core.FormatWait(grievanceWindow))
}

const debtMotive = "calote no agiota"

func (d debt) script(now time.Time) TrialScript {
	return TrialScript{
		Hearing: fmt.Sprintf("🦈 *Acusação:* @%s pegou moedas com @%s no !agiota e não pagou. O prazo venceu há %s e a dívida, com os juros, já tá em %s",
			d.debtor.JID.User, d.lender.JID.User, core.FormatWait(now.Sub(d.due)), Coins(d.amount)),
		Guilty:   "Pena: pagar o agiota na hora, com o que tiver no saldo, e ficar com fama de caloteiro",
		Innocent: "O júri teve dó do devedor, e o agiota vai ter que cobrar de outro jeito",
	}
}

func (b *bot) courtDebtCollection(ctx context.Context, chat types.JID, bill trialBill) string {
	debtor, lender := b.Account(ctx, bill.payer), b.Account(ctx, bill.payee)
	balance, _, err := b.Wallets.Balance(ctx, chat, debtor)
	if err != nil {
		b.Client.Log.Errorf("preparar %s do !processar: %v", bill.label, err)
		return ""
	}
	broke := fmt.Sprintf("%s: @%s tá liso e não tem nem como pagar 🪹 A dívida continua de pé", bill.label, bill.payer.User)
	if balance <= 0 {
		return broke
	}

	left, paid, err := b.Wallets.repayDebt(ctx, chat, debtor, lender, balance, time.Now())
	if errors.Is(err, errNoDebt) {
		return fmt.Sprintf("%s: @%s já tinha quitado a dívida antes do martelo bater", bill.label, bill.payer.User)
	}
	if errors.Is(err, ErrNoFunds) {
		return broke
	}
	if err != nil {
		b.Client.Log.Errorf("cobrar a dívida de %s no !processar: %v", bill.payer, err)
		return ""
	}

	text := fmt.Sprintf("%s: o juiz tirou %s do saldo de @%s e entregou pra @%s", bill.label, Coins(paid), bill.payer.User, bill.payee.User)
	if left.amount == 0 {
		return text + ", e a dívida tá quitada 🤝"
	}
	return text + fmt.Sprintf(". Ainda falta %s", Coins(left.amount))
}

func (s *walletStore) fileGrievance(ctx context.Context, chat types.JID, victim, accused core.Account, kind string, now time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM queixas WHERE chat = ? AND em < ?", chat.String(), now.Add(-grievanceWindow).Unix()); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx,
			`INSERT INTO queixas (chat, vitima, acusado, motivo, em) VALUES (?, ?, ?, ?, ?)
			 ON CONFLICT (chat, vitima, acusado) DO UPDATE SET motivo = excluded.motivo, em = excluded.em`,
			chat.String(), victim.Key, accused.Key, kind, now.Unix(),
		)
		return err
	})
}

func (s *walletStore) grievance(ctx context.Context, chat types.JID, victim, accused core.Account, now time.Time) (grievance, bool, error) {
	var (
		kind string
		at   int64
	)
	err := s.db.QueryRowContext(ctx,
		"SELECT motivo, em FROM queixas WHERE chat = ? AND vitima = ? AND acusado = ? AND em >= ?",
		chat.String(), victim.Key, accused.Key, now.Add(-grievanceWindow).Unix(),
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

func (s *walletStore) dropGrievance(ctx context.Context, chat types.JID, victim, accused core.Account) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM queixas WHERE chat = ? AND vitima = ? AND acusado = ?", chat.String(), victim.Key, accused.Key)
	return err
}
