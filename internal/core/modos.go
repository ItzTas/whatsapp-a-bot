package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	ModeWords        = "palavras"
	ModeLaughs       = "risada"
	modeRomance      = "romance"
	modeFamily       = "familia"
	modeGames        = "jogos"
	modeMeters       = "medidores"
	modeStats        = "estatisticas"
	ModeJokes        = "zoeira"
	modeAI           = "ia"
	modeFun          = "diversao"
	ModeUtilities    = "utilidades"
	ModeEconomy      = "economia"
	ModeModeration   = "moderacao"
	modeAIUnlimited  = "iasemlimite"
	ModeAIMeddling   = "iaenxerida"
	ModeAIInvent     = "iainventa"
	ModeAutoDownload = "baixarlinks"
	ModeConsent      = "consentimento"
	ModeCity         = "cidadedorme"
	modeAll          = "tudo"
)

type chatMode struct {
	name        string
	description string
	includes    []string
	submodes    []chatMode
	parent      string
	exclusive   bool
	coexists    []string
}

func (m chatMode) members() []string {
	if len(m.includes) > 0 {
		return m.includes
	}
	return []string{m.name}
}

func (m chatMode) allows(other chatMode) bool {
	return slices.Contains(m.coexists, other.name) || slices.Contains(m.coexists, other.parent)
}

func (m chatMode) clashes(other chatMode) bool {
	if m.name == other.name {
		return false
	}
	return (m.exclusive && !m.allows(other)) || (other.exclusive && !other.allows(m))
}

func (m chatMode) summary() string {
	if !m.exclusive {
		return m.description
	}
	return fmt.Sprintf("%s. É exclusivo: só convive com %s, e os outros modos ficam pausados enquanto ele tá ligado e voltam quando ele desligar", m.description, JoinWithAnd(m.coexists))
}

func modeExclusive(name string) bool {
	return slices.ContainsFunc(flatChatModes(), func(m chatMode) bool { return m.name == name && m.exclusive })
}

func chatModes() []chatMode {
	modes := []chatMode{
		{name: modeFun, description: "liga ou desliga de uma vez romance, familia, jogos, medidores, estatisticas, zoeira e economia", includes: []string{modeRomance, modeFamily, modeGames, modeMeters, modeStats, ModeJokes, ModeEconomy}},
		{name: modeRomance, description: "libera os comandos de romance, tipo !casar, !namorar, !trair, !traircom, !amante, !ship, !beijo, !abraço e !cafune", submodes: []chatMode{
			{name: ModeConsent, description: "o !casar e o !namorar viram pedido, que a pessoa marcada responde com !aceitar ou !recusar, e só quem tá na relação pode !terminar. Com a economia ligada, quem pede casamento paga a festa e pode oferecer dote (no namoro, um presente), o casal ganha uma !conjunta e quem pede divórcio paga pensão"},
		}},
		{name: modeFamily, description: "libera o !adotar, o !engravidar e o !familia"},
		{name: modeGames, description: "libera os jogos, tipo !bmc, !vdd, !forca, !termo, !velha, !roleta, !quiz, !completa, !batata, !stop, !impostor, !espiao, !jokenpo, !2verdades1mentira e !bandeira"},
		{name: ModeCity, description: "transforma a conversa numa partida de Cidade Dorme: o !cidadedorme abre a sala, à noite o assassino usa o !matar, o médico o !salvar e o detetive o !investigar no privado do bot, e de dia a cidade decide no !votar quem eliminar. O !matar e o !votar daqui tomam o lugar dos da economia", exclusive: true, coexists: []string{ModeModeration, ModeUtilities}},
		{name: modeMeters, description: "libera os medidores, tipo !medidor, !gado, !gay, !rankinggay e !euanime"},
		{name: modeStats, description: "libera as estatísticas, tipo !ranking, !palavra, !tu, !palhaço, !retrospectiva, !primeiramsg e !horário"},
		{name: ModeJokes, description: "libera a zoeira, tipo !kkk, !chute, !apelido e !frase"},
		{name: ModeEconomy, description: "libera as moedas do grupo: !saldo, !daily, !trabalhar, !profissao, !apostar, !cassino, !blackjack, !crash, !corrida, !roubar, !assalto, !quadrilha, !sequestrar, !venderrefem, !matar, !reviver, !prender, !fugir, !denunciar, !processar, !votar, !presidente, !decreto, !impeachment, !golpe, !transferir, !poupar, !conjunta, !agiota, !emprestimo, !acoes, !loteria, !bolao, !ricos, !imposto, !fianca, !hospital, !saude, !resgate, !seguranca, !imovel, !empregados, !titulo, !igreja, !doar, !zoar, !abencoar e !amaldicoar, e os itens: !mochila, !item, !loja, !petshop, !luxo, !mercadonegro, !comprar, !vender, !aceitar, !recusar, !leilao, !lance, !dar, !usar, !pescar, !minerar, !vasculhar, !catar e !furtar. Com ele, quem acerta o !quiz, o !quemfalou, a !forca, o !termo, o !completa, o !emoji, a !bandeira ou a mentira do !2verdades1mentira ou vence o !stop, o !impostor ou o !espiao ganha moedas (se pelo menos 2 pessoas jogarem), o !roleta e o !jokenpo @pessoa aceitam aposta, o !tu mostra o saldo, quem casou pelo pedido do consentimento e pede divórcio paga pensão, quem é pego no !trair ou no !traircom paga indenização e quem é preso no !prender ou num !assalto que deu errado fica sem apostar, roubar, furtar, sequestrar e assaltar até sair da cadeia, quem é sequestrado fica igual até sair do cativeiro ou conseguir !fugir, quem morre no !matar, de overdose ou com a saúde zerada (dá pra se machucar no !trabalhar das profissões perigosas e nas extrações) fica sem nenhum comando de economia até ressuscitar ou alguém !reviver e o enterro leva um pedaço do saldo, e quem pega moedas com o !agiota e passa do prazo pode ser morto, preso, sequestrado ou processado por quem emprestou, e quem pega o !emprestimo do banco e passa do prazo tem o saldo penhorado até quitar"},
		{name: modeAI, description: "libera o !ia, o !resumo, o !roast, o !signo, o !quemsou, o !quizia, o !emoji, o !processaria, o !cancelar, o !fofoca e o !ficia", submodes: []chatMode{
			{name: modeAIUnlimited, description: "a IA para de contar quantas perguntas cada um fez na hora e responde sem pausa entre uma pergunta e outra"},
			{name: ModeAIMeddling, description: "libera o !perseguir, que faz a IA responder toda mensagem da pessoa marcada"},
			{name: ModeAIInvent, description: "a IA responde quando alguém manda um comando que não existe, como se ele existisse"},
		}},
		{name: ModeModeration, description: "libera pros admins o !ban, o !promover, o !rebaixar, o !mutar e o !desmutar (o bot também precisa ser admin), o !advertir (na 3ª advertência a pessoa é banida), o !perdoar, o !boasvindas, o !fixar (por um tempo ou pra sempre) e o !desfixar, e pra todo mundo o !advertencias. O bot passa a apagar as mensagens de quem tá mutado e dar boas-vindas a quem entra no grupo"},
		{name: ModeWords, description: "o bot responde algumas palavras mesmo sem o !, tipo arrogante, nojo, aura, ego ou chorar"},
		{name: ModeLaughs, description: "o bot manda uma figurinha de risada quando alguém ri (kkkk, hahaha, rsrs...)"},
		{name: ModeUtilities, description: "libera o !baixar, o !figurinha, o !mp3, o !musica, o !som, o !transcrever, o !lembrete, o !buscar, o !aniversario (com parabéns automáticos no dia), o !cotacao, o !clima e o !ocr", submodes: []chatMode{
			{name: ModeAutoDownload, description: "o bot baixa sozinho e manda os vídeos, imagens e gifs dos links do Instagram, TikTok, X e outros, sem precisar do !baixar"},
		}},
	}
	var all, submodes, exclusives []string
	for _, m := range modes {
		if m.exclusive {
			exclusives = append(exclusives, m.name)
		}
		if len(m.includes) == 0 && !m.exclusive {
			all = append(all, m.name)
		}
		for _, sub := range m.submodes {
			submodes = append(submodes, sub.name)
		}
	}
	description := fmt.Sprintf("liga ou desliga de uma vez todos os modos, menos os submodos (%s) e os exclusivos (%s)", strings.Join(submodes, ", "), strings.Join(exclusives, ", "))
	return append([]chatMode{{name: modeAll, description: description, includes: all}}, modes...)
}

func flatChatModes() []chatMode {
	var flat []chatMode
	for _, m := range chatModes() {
		flat = append(flat, m)
		for _, sub := range m.submodes {
			sub.parent = m.name
			flat = append(flat, sub)
		}
	}
	return flat
}

func modeActive(active []string, mode string) bool {
	modes := flatChatModes()
	i := slices.IndexFunc(modes, func(m chatMode) bool { return m.name == mode })
	if i < 0 || modes[i].parent == "" {
		return slices.Contains(active, mode)
	}
	return allActive(active, []string{mode, modes[i].parent})
}

type modeStore struct {
	db *sql.DB
}

func (s *modeStore) Enabled(ctx context.Context, chat types.JID, mode string) (bool, error) {
	var on bool
	err := s.db.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT 1 FROM conversa_modos WHERE chat = ? AND modo = ?)",
		chat.String(), mode,
	).Scan(&on)
	return on, err
}

type modeSwitch struct {
	changed []string
	paused  map[string][]string
	stopped []string
	resumed map[string][]string
}

func (s *modeStore) enable(ctx context.Context, chat types.JID, modes []string) (sw modeSwitch, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return modeSwitch{}, err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, tx.Rollback())
		}
	}()

	active, err := activeModes(ctx, tx, chat)
	if err != nil {
		return modeSwitch{}, err
	}
	sw = modeSwitch{paused: map[string][]string{}, resumed: map[string][]string{}}
	sw.changed = slices.DeleteFunc(slices.Clone(modes), func(mode string) bool { return slices.Contains(active, mode) })
	for _, rival := range rivalsOf(active, modes, true) {
		back, err := stopMode(ctx, tx, chat, rival)
		if err != nil {
			return modeSwitch{}, err
		}
		sw.stopped = append(sw.stopped, rival)
		sw.resumed[rival] = back
	}
	if active, err = activeModes(ctx, tx, chat); err != nil {
		return modeSwitch{}, err
	}
	for _, mode := range modes {
		for _, rival := range rivalsOf(active, []string{mode}, false) {
			if err := pauseMode(ctx, tx, chat, rival, mode); err != nil {
				return modeSwitch{}, err
			}
			sw.paused[mode] = append(sw.paused[mode], rival)
		}
	}
	for _, mode := range modes {
		if _, err := tx.ExecContext(ctx, "DELETE FROM conversa_modos_pausados WHERE chat = ? AND modo = ?", chat.String(), mode); err != nil {
			return modeSwitch{}, err
		}
		if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO conversa_modos (chat, modo) VALUES (?, ?)", chat.String(), mode); err != nil {
			return modeSwitch{}, err
		}
	}
	return sw, tx.Commit()
}

func (s *modeStore) disable(ctx context.Context, chat types.JID, modes []string) (sw modeSwitch, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return modeSwitch{}, err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, tx.Rollback())
		}
	}()

	active, err := activeModes(ctx, tx, chat)
	if err != nil {
		return modeSwitch{}, err
	}
	paused, err := pausedModes(ctx, tx, chat)
	if err != nil {
		return modeSwitch{}, err
	}
	sw = modeSwitch{resumed: map[string][]string{}}
	for _, mode := range modes {
		_, held := paused[mode]
		if slices.Contains(active, mode) || held {
			sw.changed = append(sw.changed, mode)
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM conversa_modos_pausados WHERE chat = ? AND modo = ?", chat.String(), mode); err != nil {
			return modeSwitch{}, err
		}
	}
	for _, mode := range modes {
		if !slices.Contains(active, mode) {
			continue
		}
		back, err := stopMode(ctx, tx, chat, mode)
		if err != nil {
			return modeSwitch{}, err
		}
		if len(back) > 0 {
			sw.resumed[mode] = back
		}
	}
	return sw, tx.Commit()
}

func rivalsOf(active, requested []string, exclusive bool) []string {
	modes := flatChatModes()
	byName := make(map[string]chatMode, len(modes))
	for _, m := range modes {
		byName[m.name] = m
	}
	var rivals []string
	for _, name := range active {
		m, known := byName[name]
		if !known || m.exclusive != exclusive || slices.Contains(requested, name) {
			continue
		}
		if slices.ContainsFunc(requested, func(r string) bool { return m.clashes(byName[r]) }) {
			rivals = append(rivals, name)
		}
	}
	slices.SortFunc(rivals, compareModes)
	return rivals
}

func pauseMode(ctx context.Context, tx *sql.Tx, chat types.JID, mode, by string) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM conversa_modos WHERE chat = ? AND modo = ?", chat.String(), mode); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx,
		`INSERT INTO conversa_modos_pausados (chat, modo, pausado_por) VALUES (?, ?, ?)
		 ON CONFLICT (chat, modo) DO UPDATE SET pausado_por = excluded.pausado_por`,
		chat.String(), mode, by,
	)
	return err
}

func stopMode(ctx context.Context, tx *sql.Tx, chat types.JID, mode string) (back []string, err error) {
	if _, err := tx.ExecContext(ctx, "DELETE FROM conversa_modos WHERE chat = ? AND modo = ?", chat.String(), mode); err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx, "DELETE FROM conversa_modos_pausados WHERE chat = ? AND pausado_por = ? RETURNING modo", chat.String(), mode)
	if err != nil {
		return nil, err
	}
	if back, err = scanModes(rows); err != nil {
		return nil, err
	}
	slices.SortFunc(back, compareModes)
	for _, m := range back {
		if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO conversa_modos (chat, modo) VALUES (?, ?)", chat.String(), m); err != nil {
			return nil, err
		}
	}
	return back, nil
}

func compareModes(a, b string) int {
	order := func(name string) int {
		return slices.IndexFunc(flatChatModes(), func(m chatMode) bool { return m.name == name })
	}
	return order(a) - order(b)
}

func (s *modeStore) active(ctx context.Context, chat types.JID) ([]string, error) {
	return activeModes(ctx, s.db, chat)
}

func (s *modeStore) paused(ctx context.Context, chat types.JID) (map[string]string, error) {
	return pausedModes(ctx, s.db, chat)
}

func activeModes(ctx context.Context, q RowsQuerier, chat types.JID) ([]string, error) {
	rows, err := q.QueryContext(ctx, "SELECT modo FROM conversa_modos WHERE chat = ?", chat.String())
	if err != nil {
		return nil, err
	}
	return scanModes(rows)
}

func scanModes(rows *sql.Rows) (modes []string, err error) {
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var mode string
		if err := rows.Scan(&mode); err != nil {
			return nil, err
		}
		modes = append(modes, mode)
	}
	return modes, rows.Err()
}

func pausedModes(ctx context.Context, q RowsQuerier, chat types.JID) (paused map[string]string, err error) {
	rows, err := q.QueryContext(ctx, "SELECT modo, pausado_por FROM conversa_modos_pausados WHERE chat = ?", chat.String())
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	paused = map[string]string{}
	for rows.Next() {
		var mode, by string
		if err := rows.Scan(&mode, &by); err != nil {
			return nil, err
		}
		paused[mode] = by
	}
	return paused, rows.Err()
}

func (b *Bot) ChatModeOn(ctx context.Context, chat types.JID, mode string) (bool, error) {
	active, err := b.Modes.active(ctx, chat.ToNonAD())
	if err != nil {
		return false, err
	}
	return modeActive(active, mode), nil
}

func (b *Bot) toggleChatMode(ctx context.Context, evt *events.Message, on bool, args []string) error {
	chat := evt.Info.Chat
	modes := flatChatModes()
	byName := make(map[string]chatMode, len(modes))
	names := make([]string, len(modes))
	for i, m := range modes {
		byName[m.name] = m
		names[i] = m.name
	}
	usageText := "Use assim: !modo ativar jogos (ou !modo desativar jogos). Dá pra mexer em vários de uma vez, tipo !modo ativar jogos zoeira ia. Os modos são " + strings.Join(names, ", ") + ". Pra mexer em outro grupo, coloque -- e o nome dele depois dos modos, tipo !modo ativar jogos ia -- Galera."

	modesText, groupName, toGroup := strings.Cut(strings.ReplaceAll(strings.Join(args, " "), ",", " "), "--")
	groupName = strings.TrimSpace(groupName)
	requested := strings.Fields(FoldLetters(modesText))
	if len(requested) == 0 || (toGroup && groupName == "") {
		return b.SendText(ctx, chat, usageText)
	}
	if i := slices.IndexFunc(requested, func(name string) bool { _, isMode := byName[name]; return !isMode }); i >= 0 {
		return b.SendText(ctx, chat, fmt.Sprintf("Não existe o modo *%s*. %s", requested[i], usageText))
	}

	target, place, problem, err := b.modeTarget(ctx, chat, groupName, toGroup)
	if err != nil {
		return err
	}
	if problem != "" {
		return b.SendText(ctx, chat, problem)
	}

	selected := make([]chatMode, len(requested))
	var members []string
	for i, name := range requested {
		selected[i] = byName[name]
		for _, member := range selected[i].members() {
			if !slices.Contains(members, member) {
				members = append(members, member)
			}
		}
	}
	if on {
		if clash := clashingPair(members, byName); clash != "" {
			return b.SendText(ctx, chat, clash)
		}
	}
	change := b.Modes.disable
	if on {
		change = b.Modes.enable
	}
	sw, err := change(ctx, target, members)
	if err != nil {
		return err
	}
	defer b.modesTurnedOff(ctx, target, sw.off(on))
	results := map[[2]bool]string{
		{true, true}:   "✅ Modo *%[1]s* ativado %[2]s: %[3]s.",
		{true, false}:  "O modo *%[1]s* já tava ativado %[2]s.",
		{false, true}:  "✅ Modo *%[1]s* desativado %[2]s.",
		{false, false}: "O modo *%[1]s* já tava desativado %[2]s.",
	}
	lines := make([]string, len(selected))
	for i, m := range selected {
		modeChanged := slices.ContainsFunc(m.members(), func(name string) bool { return slices.Contains(sw.changed, name) })
		lines[i] = fmt.Sprintf(results[[2]bool{on, modeChanged}], m.name, place, m.description)
	}
	lines = append(lines, sw.notes(members)...)
	if !on {
		return b.SendText(ctx, chat, strings.Join(lines, "\n"))
	}
	warnings, err := b.parentWarnings(ctx, target, selected)
	if err != nil {
		return err
	}
	return b.SendText(ctx, chat, strings.Join(append(lines, warnings...), "\n"))
}

func (sw modeSwitch) off(on bool) []string {
	if !on {
		return sw.changed
	}
	off := slices.Clone(sw.stopped)
	for _, paused := range sw.paused {
		off = append(off, paused...)
	}
	return off
}

func clashingPair(members []string, byName map[string]chatMode) string {
	for i, a := range members {
		j := slices.IndexFunc(members[i+1:], func(other string) bool { return byName[a].clashes(byName[other]) })
		if j < 0 {
			continue
		}
		exclusive, other := a, members[i+1+j]
		if !byName[exclusive].exclusive {
			exclusive, other = other, exclusive
		}
		return fmt.Sprintf("O *%s* não convive com o *%s*, então não dá pra ligar os dois juntos. Liga só um deles.", exclusive, other)
	}
	return ""
}

func (sw modeSwitch) notes(members []string) []string {
	var notes []string
	for _, mode := range sw.stopped {
		note := fmt.Sprintf("⏹️ O *%s* não convive com o que você ligou, então foi desligado.", mode)
		if back := sw.resumed[mode]; len(back) > 0 {
			note += fmt.Sprintf(" Os modos que ele tinha pausado voltaram: %s.", boldModes(back))
		}
		notes = append(notes, note)
	}
	for _, mode := range members {
		if paused := sw.paused[mode]; len(paused) > 0 {
			notes = append(notes, fmt.Sprintf("⏸️ Pausei %s enquanto o *%s* tiver ligado, porque eles não convivem. Quando ele desligar, eles voltam.", boldModes(paused), mode))
		}
		if back := sw.resumed[mode]; len(back) > 0 && !slices.Contains(sw.stopped, mode) {
			notes = append(notes, fmt.Sprintf("▶️ Voltaram os modos que o *%s* tinha pausado: %s.", mode, boldModes(back)))
		}
	}
	return notes
}

func boldModes(modes []string) string {
	bold := make([]string, len(modes))
	for i, mode := range modes {
		bold[i] = "*" + mode + "*"
	}
	return JoinWithAnd(bold)
}

func (b *Bot) parentWarnings(ctx context.Context, target types.JID, selected []chatMode) ([]string, error) {
	if !slices.ContainsFunc(selected, func(m chatMode) bool { return m.parent != "" }) {
		return nil, nil
	}
	active, err := b.Modes.active(ctx, target)
	if err != nil {
		return nil, err
	}
	var warnings []string
	for _, m := range selected {
		if m.parent != "" && !slices.Contains(active, m.parent) {
			warnings = append(warnings, fmt.Sprintf("⚠️ O *%s* é um submodo do *%s* e só funciona quando o *%s* também tá ativado.", m.name, m.parent, m.parent))
		}
	}
	return warnings, nil
}

func (b *Bot) modeTarget(ctx context.Context, chat types.JID, groupName string, toGroup bool) (target types.JID, place, problem string, err error) {
	if !toGroup {
		return chat.ToNonAD(), "nesta conversa", "", nil
	}
	entries, problem, err := b.groupByName(ctx, groupName)
	if err != nil || problem != "" {
		return types.EmptyJID, "", problem, err
	}
	return entries[0].ids[0], "no grupo *" + entries[0].name + "*", "", nil
}

func (b *Bot) modos(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat
	var (
		active []string
		paused map[string]string
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { active, err = b.Modes.active(gctx, chat.ToNonAD()); return err })
	g.Go(func() (err error) { paused, err = b.Modes.paused(gctx, chat.ToNonAD()); return err })
	if err := g.Wait(); err != nil {
		return err
	}
	lines := append(chatModeLines(active, paused), "", "_O dono do bot liga e desliga com !modo ativar <modo> e !modo desativar <modo>_")
	return b.SendText(ctx, chat, strings.Join(lines, "\n"))
}

func chatModeLines(active []string, paused map[string]string) []string {
	lines := []string{"🧩 *Modos desta conversa*"}
	for _, m := range chatModes() {
		lines = append(lines, fmt.Sprintf("• %s: *%s* (%s)", m.name, modeStatus(active, paused, m), m.summary()))
		for _, sub := range m.submodes {
			lines = append(lines, fmt.Sprintf("    ↳ %s: *%s* (%s)", sub.name, modeStatus(active, paused, sub), sub.summary()))
		}
	}
	return lines
}

func modeStatus(active []string, paused map[string]string, m chatMode) string {
	if allActive(active, m.members()) {
		return "ligado"
	}
	if by, held := paused[m.name]; held {
		return "pausado pelo " + by
	}
	return "desligado"
}

func allActive(active, modes []string) bool {
	return !slices.ContainsFunc(modes, func(mode string) bool { return !slices.Contains(active, mode) })
}
