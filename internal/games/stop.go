package games

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	stopCategoryCount = 5
	stopDuration      = 2 * time.Minute
	stopGrace         = 15 * time.Second
	stopMaxAnswer     = 40
	stopReward        = 60
	stopGameKind      = "stop"
	stopClosedNotice  = "⏰ Esse stop já fechou, espera o resultado"
)

var stopNumbering = regexp.MustCompile(`^\d+\s*[.)\-:]\s*`)

type stopSheet struct {
	key     string
	player  types.JID
	answers []string
}

type stopRound struct {
	letter     string
	categories []string
	starter    string
	stopped    chan struct{}
	ends       time.Time

	mu        sync.Mutex
	sheets    []stopSheet
	stopper   types.JID
	stoppedAt time.Time
	closed    bool
	over      bool
}

type stopState struct {
	Letter     string           `json:"letter"`
	Categories []string         `json:"categories"`
	Starter    string           `json:"starter"`
	Ends       time.Time        `json:"ends"`
	Sheets     []stopSavedSheet `json:"sheets"`
	Stopper    types.JID        `json:"stopper"`
	StoppedAt  time.Time        `json:"stopped_at"`
	Closed     bool             `json:"closed"`
}

type stopSavedSheet struct {
	Key     string    `json:"key"`
	Player  types.JID `json:"player"`
	Answers []string  `json:"answers"`
}

type stopScore struct {
	sheet  stopSheet
	marks  []string
	points int
}

func (b *bot) stop(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return b.SendText(ctx, chat, "O !stop só funciona em grupo")
	}
	answers := stopAnswers(core.MessageText(evt.Message))
	value, active := b.stopGames.Load(chat.String())
	if active && value.(*stopRound).isClosed() {
		return b.SendText(ctx, chat, stopClosedNotice)
	}
	if len(answers) == 0 && active {
		return b.SendText(ctx, chat, value.(*stopRound).status("📝 *Stop rolando!*"))
	}
	if len(answers) == 0 {
		return b.startStop(ctx, evt)
	}
	if !active {
		return b.SendText(ctx, chat, "Não tem stop rolando. Manda !stop pra começar")
	}
	return b.answerStop(ctx, evt, value.(*stopRound), answers)
}

func (b *bot) startStop(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	round := newStopRound(evt.Info.Sender.User)
	if value, loaded := b.stopGames.LoadOrStore(chat.String(), round); loaded {
		return b.SendText(ctx, chat, value.(*stopRound).status("📝 *Já tem um stop rolando!*"))
	}
	b.syncStop(ctx, chat, round)
	go b.runStop(ctx, chat, round)
	return b.SendText(ctx, chat, round.status("📝 *Stop!*"))
}

func newStopRound(starter string) *stopRound {
	letters := strings.Split("ABCDEFGIJLMNOPRSTV", "")
	categories := []string{
		"Nome", "Cidade, estado ou país", "Fruta", "Animal", "Objeto", "Cor", "Marca", "Profissão",
		"Comida", "Filme ou série", "Parte do corpo", "Verbo", "Time de futebol", "Carro",
		"Coisa que tem em casa", "Cantor ou banda", "Esporte", "Minha sogra é...",
	}
	return &stopRound{
		letter:     core.PickOne(letters),
		categories: core.PickSome(categories, stopCategoryCount),
		starter:    starter,
		stopped:    make(chan struct{}),
		ends:       time.Now().Add(stopDuration),
	}
}

func (s stopState) round() *stopRound {
	round := &stopRound{
		letter:     s.Letter,
		categories: s.Categories,
		starter:    s.Starter,
		stopped:    make(chan struct{}),
		ends:       s.Ends,
		stopper:    s.Stopper,
		stoppedAt:  s.StoppedAt,
		closed:     s.Closed,
	}
	for _, sheet := range s.Sheets {
		round.sheets = append(round.sheets, stopSheet{key: sheet.Key, player: sheet.Player, answers: sheet.Answers})
	}
	if !round.stopper.IsEmpty() {
		close(round.stopped)
	}
	return round
}

func (b *bot) restoreStop(ctx context.Context, saved core.SavedGame) error {
	var state stopState
	if err := json.Unmarshal(saved.State, &state); err != nil {
		return fmt.Errorf("ler o stop salvo: %w", err)
	}
	chat, err := types.ParseJID(saved.Key)
	if err != nil {
		return fmt.Errorf("ler a conversa do stop salvo: %w", err)
	}
	round := state.round()
	if _, loaded := b.stopGames.LoadOrStore(saved.Key, round); loaded {
		return nil
	}
	if round.closed {
		go b.finishStop(ctx, chat, round)
		return nil
	}
	go b.runStop(ctx, chat, round)
	return nil
}

func (b *bot) syncStop(ctx context.Context, chat types.JID, round *stopRound) {
	round.mu.Lock()
	defer round.mu.Unlock()
	if round.over {
		return
	}
	b.KeepGame(ctx, stopGameKind, chat.String(), round.snapshot(), round.closesAt())
}

func (b *bot) endStop(ctx context.Context, chat types.JID, round *stopRound) {
	round.mu.Lock()
	defer round.mu.Unlock()
	if round.over {
		return
	}
	round.over = true
	b.DropGame(ctx, stopGameKind, chat.String())
}

func (r *stopRound) snapshot() stopState {
	state := stopState{
		Letter:     r.letter,
		Categories: r.categories,
		Starter:    r.starter,
		Ends:       r.ends,
		Stopper:    r.stopper,
		StoppedAt:  r.stoppedAt,
		Closed:     r.closed,
	}
	for _, sheet := range r.sheets {
		state.Sheets = append(state.Sheets, stopSavedSheet{Key: sheet.key, Player: sheet.player, Answers: sheet.answers})
	}
	return state
}

func (r *stopRound) closesAt() time.Time {
	if r.closed {
		return time.Now()
	}
	if r.stopper.IsEmpty() {
		return r.ends
	}
	return r.stoppedAt.Add(stopGrace)
}

func (r *stopRound) isClosed() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.closed
}

func stopAnswers(text string) []string {
	pieces := strings.FieldsFunc(core.QuestionFrom(text), func(r rune) bool { return r == ',' || r == '\n' })
	answers := make([]string, 0, len(pieces))
	for _, piece := range pieces {
		answer := strings.TrimSpace(stopNumbering.ReplaceAllString(strings.TrimSpace(piece), ""))
		if answer != "" {
			answers = append(answers, answer)
		}
	}
	return answers
}

func (b *bot) answerStop(ctx context.Context, evt *events.Message, round *stopRound, answers []string) error {
	if len(answers) != len(round.categories) {
		return b.ReplyTo(ctx, evt, fmt.Sprintf("📝 Manda as %d respostas na ordem, separadas por vírgula ou uma por linha. Não sabe alguma? Põe um - no lugar", len(round.categories)))
	}
	if slices.ContainsFunc(answers, func(a string) bool { return utf8.RuneCountInString(a) > stopMaxAnswer }) {
		return b.ReplyTo(ctx, evt, fmt.Sprintf("📏 Resposta grande demais, cada uma pode ter no máximo %d letras", stopMaxAnswer))
	}
	player := evt.Info.Sender.ToNonAD()
	reaction, stopped := round.submit(stopSheet{key: b.Account(ctx, player).Key, player: player, answers: answers})
	if reaction != "" {
		b.syncStop(ctx, evt.Info.Chat.ToNonAD(), round)
	}
	if stopped {
		text := fmt.Sprintf("🛑 *STOP!* @%s terminou!\n\n_Quem ainda não mandou tem %d segundos_ ⏳", player.User, int(stopGrace.Seconds()))
		_, err := b.SendMentions(ctx, evt.Info.Chat, text, []types.JID{player})
		return err
	}
	if reaction == "" {
		return b.ReplyTo(ctx, evt, stopClosedNotice)
	}
	b.React(ctx, evt, reaction)
	return nil
}

func (b *bot) runStop(ctx context.Context, chat types.JID, round *stopRound) {
	if !round.wait(ctx) {
		b.stopGames.CompareAndDelete(chat.String(), round)
		return
	}
	b.syncStop(ctx, chat, round)
	b.finishStop(ctx, chat, round)
}

func (b *bot) finishStop(ctx context.Context, chat types.JID, round *stopRound) {
	defer b.stopGames.CompareAndDelete(chat.String(), round)
	if len(round.sheets) == 0 {
		b.endStop(ctx, chat, round)
		text := fmt.Sprintf("🦗 O stop da letra *%s* acabou e ninguém mandou resposta", round.letter)
		if err := b.SendText(ctx, chat, text); err != nil {
			b.Client.Log.Errorf("fechar !stop: %v", err)
		}
		return
	}
	verdicts, note := b.judgeStop(ctx, chat, round)
	if ctx.Err() != nil {
		return
	}
	scores := round.score(verdicts)
	winners := stopWinners(scores)
	players := make([]types.JID, len(scores))
	for i, s := range scores {
		players[i] = s.sheet.player
	}
	b.endStop(ctx, chat, round)
	text := core.AppendLine(round.result(scores, winners, note), b.economy.Reward(ctx, chat, winners, len(scores), stopReward))
	if _, err := b.SendMentions(ctx, chat, text, players); err != nil {
		b.Client.Log.Errorf("resultado do !stop: %v", err)
	}
}

func (b *bot) judgeStop(ctx context.Context, chat types.JID, round *stopRound) ([][]bool, string) {
	if !b.Assistant.Enabled() {
		return nil, "_🤖 Sem IA configurada, então só conferi a letra_"
	}
	prompt := round.prompt()
	stopTyping := b.Typing(ctx, chat)
	raw, err := b.Assistant.Ask(ctx, b.Asker(ctx, chat, round.starter), func(int) string { return prompt })
	stopTyping()
	if err != nil {
		b.Client.Log.Warnf("!stop: %v", err)
		return nil, "_🤖 A IA não respondeu, então só conferi a letra_"
	}
	verdicts, err := parseStopVerdicts(raw, len(round.sheets), len(round.categories))
	if err != nil {
		b.Client.Log.Warnf("!stop: julgamento inválido da IA: %v: %q", err, raw)
		return nil, "_🤖 A IA se enrolou no julgamento, então só conferi a letra_"
	}
	return verdicts, "_🤖 A IA conferiu se cada resposta combina com a categoria_"
}

func (r *stopRound) prompt() string {
	sheets := make(map[string][]string, len(r.sheets))
	for i, s := range r.sheets {
		sheets[stopPlayerKey(i)] = s.answers
	}
	raw, _ := json.Marshal(sheets)
	categories := make([]string, len(r.categories))
	for i, c := range r.categories {
		categories[i] = fmt.Sprintf("%d. %s", i+1, c)
	}
	return fmt.Sprintf(`Você é o juiz de uma rodada de Stop (adedanha). A letra é %s.
As categorias, na ordem, são: %s.
As respostas de cada jogador, na mesma ordem das categorias, estão neste JSON: %s
Para cada resposta, diga se ela vale: tem que começar com a letra %s (ignorando acento) e ser um exemplo de verdade da categoria. Aceite erro leve de digitação, gíria e nome popular, mas não aceite palavra inventada, traço, resposta vazia ou coisa de outra categoria.
Responda só com um JSON, sem texto antes ou depois e sem bloco de código, com a mesma chave de cada jogador e uma lista de %d valores true ou false na ordem das categorias, neste formato:
{"J1": [true, false, true, true, true]}`,
		r.letter, strings.Join(categories, "; "), raw, r.letter, len(r.categories))
}

func stopPlayerKey(i int) string {
	return fmt.Sprintf("J%d", i+1)
}

func parseStopVerdicts(raw string, players, categories int) ([][]bool, error) {
	start, end := strings.Index(raw, "{"), strings.LastIndex(raw, "}")
	if start < 0 || end < start {
		return nil, errors.New("sem JSON na resposta")
	}
	var byPlayer map[string][]bool
	if err := json.Unmarshal([]byte(raw[start:end+1]), &byPlayer); err != nil {
		return nil, err
	}
	verdicts := make([][]bool, players)
	for i := range verdicts {
		key := stopPlayerKey(i)
		verdicts[i] = byPlayer[key]
		if len(verdicts[i]) != categories {
			return nil, fmt.Errorf("%s veio com %d respostas em vez de %d", key, len(verdicts[i]), categories)
		}
	}
	return verdicts, nil
}

func (r *stopRound) status(header string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	lines := []string{fmt.Sprintf("%s Letra *%s*", header, r.letter), ""}
	for i, c := range r.categories {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, c))
	}
	lines = append(lines, "",
		fmt.Sprintf("_Manda *!stop* com as %d respostas na ordem, separadas por vírgula ou uma por linha. Não sabe alguma? Põe um - no lugar_", len(r.categories)),
		fmt.Sprintf("_Quem completar tudo com a letra certa primeiro pede STOP e o resto tem %d segundos. Se ninguém pedir, fecha em %d minutos_ ⏳", int(stopGrace.Seconds()), int(stopDuration.Minutes())),
	)
	if len(r.sheets) > 0 {
		lines = append(lines, "", fmt.Sprintf("📝 Respostas recebidas: %d", len(r.sheets)))
	}
	if !r.stopper.IsEmpty() {
		lines = append(lines, fmt.Sprintf("🛑 @%s já pediu STOP, corre!", r.stopper.User))
	}
	return strings.Join(lines, "\n")
}

func (r *stopRound) submit(sheet stopSheet) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return "", false
	}
	reaction := "✅"
	if i := slices.IndexFunc(r.sheets, func(s stopSheet) bool { return s.key == sheet.key }); i >= 0 {
		r.sheets = slices.Delete(r.sheets, i, i+1)
		reaction = "🔄"
	}
	r.sheets = append(r.sheets, sheet)
	if !r.stopper.IsEmpty() || slices.ContainsFunc(sheet.answers, func(a string) bool { return !r.startsRight(a) }) {
		return reaction, false
	}
	r.stopper = sheet.player
	r.stoppedAt = time.Now()
	close(r.stopped)
	return reaction, true
}

func (r *stopRound) wait(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(time.Until(r.ends)):
	case <-r.stopped:
		select {
		case <-ctx.Done():
			return false
		case <-time.After(time.Until(r.graceEnds())):
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	return true
}

func (r *stopRound) graceEnds() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.stoppedAt.Add(stopGrace)
}

func (r *stopRound) startsRight(answer string) bool {
	folded := core.FoldLetters(answer)
	return utf8.RuneCountInString(folded) > 1 && strings.HasPrefix(folded, strings.ToLower(r.letter))
}

func (r *stopRound) score(verdicts [][]bool) []stopScore {
	valid := make([][]bool, len(r.sheets))
	repeats := make([]map[string]int, len(r.categories))
	for c := range repeats {
		repeats[c] = map[string]int{}
	}
	for i, s := range r.sheets {
		valid[i] = make([]bool, len(s.answers))
		for c, a := range s.answers {
			valid[i][c] = r.startsRight(a) && (verdicts == nil || verdicts[i][c])
			if valid[i][c] {
				repeats[c][stopAnswerKey(a)]++
			}
		}
	}
	scores := make([]stopScore, len(r.sheets))
	for i, s := range r.sheets {
		scores[i] = stopScore{sheet: s, marks: make([]string, len(s.answers))}
		for c, a := range s.answers {
			mark, points := stopMark(valid[i][c], repeats[c][stopAnswerKey(a)])
			scores[i].marks[c] = mark + " " + a
			scores[i].points += points
		}
	}
	slices.SortStableFunc(scores, func(x, y stopScore) int { return cmp.Compare(y.points, x.points) })
	return scores
}

func stopAnswerKey(answer string) string {
	return strings.Join(strings.Fields(core.FoldLetters(answer)), " ")
}

func stopMark(valid bool, sameAnswers int) (string, int) {
	if !valid {
		return "❌", 0
	}
	if sameAnswers > 1 {
		return "👯", 5
	}
	return "✅", 10
}

func stopWinners(scores []stopScore) []types.JID {
	var winners []types.JID
	for _, s := range scores {
		if s.points == 0 || s.points < scores[0].points {
			break
		}
		winners = append(winners, s.sheet.player)
	}
	return winners
}

func (r *stopRound) result(scores []stopScore, winners []types.JID, note string) string {
	lines := []string{
		fmt.Sprintf("🛑 *Fim do stop!* Letra *%s*", r.letter),
		"📋 " + strings.Join(r.categories, " · "),
	}
	for i, s := range scores {
		medal := ""
		if i < len(winners) {
			medal = "🏆 "
		}
		lines = append(lines, "", fmt.Sprintf("%s@%s: *%d pontos*", medal, s.sheet.player.User, s.points), strings.Join(s.marks, " · "))
	}
	if len(winners) == 0 {
		lines = append(lines, "", "😂 Ninguém fez ponto")
	}
	lines = append(lines, "", "_✅ 10 pontos · 👯 repetida, 5 · ❌ não vale_", note)
	return strings.Join(lines, "\n")
}
