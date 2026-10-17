package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"unicode"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	modeWords        = "palavras"
	modeLaughs       = "risada"
	modeRomance      = "romance"
	modeFamily       = "familia"
	modeGames        = "jogos"
	modeMeters       = "medidores"
	modeStats        = "estatisticas"
	modeJokes        = "zoeira"
	modeAI           = "ia"
	modeFun          = "diversao"
	modeUtilities    = "utilidades"
	modeEconomy      = "economia"
	modeModeration   = "moderacao"
	modeAIUnlimited  = "iasemlimite"
	modeAIMeddling   = "iaenxerida"
	modeAIInvent     = "iainventa"
	modeAutoDownload = "baixarlinks"
	modeConsent      = "consentimento"
	modeAll          = "tudo"
)

type chatMode struct {
	name        string
	description string
	includes    []string
	submodes    []chatMode
	parent      string
}

func (m chatMode) members() []string {
	if len(m.includes) > 0 {
		return m.includes
	}
	return []string{m.name}
}

func chatModes() []chatMode {
	modes := []chatMode{
		{name: modeFun, description: "liga ou desliga de uma vez romance, familia, jogos, medidores, estatisticas, zoeira e economia", includes: []string{modeRomance, modeFamily, modeGames, modeMeters, modeStats, modeJokes, modeEconomy}},
		{name: modeRomance, description: "libera os comandos de romance, tipo !casar, !namorar, !trair, !traircom, !amante, !ship, !beijo, !abraço e !cafune", submodes: []chatMode{
			{name: modeConsent, description: "o !casar e o !namorar viram pedido, que a pessoa marcada responde com !aceitar ou !recusar, e só quem tá na relação pode !terminar. Com a economia ligada, quem pede casamento paga a festa e pode oferecer dote (no namoro, um presente), o casal ganha uma !conjunta e quem pede divórcio paga pensão"},
		}},
		{name: modeFamily, description: "libera o !adotar, o !engravidar e o !familia"},
		{name: modeGames, description: "libera os jogos, tipo !bmc, !vdd, !forca, !termo, !velha, !roleta, !quiz, !completa, !batata, !stop, !impostor, !jokenpo, !2verdades1mentira e !bandeira"},
		{name: modeMeters, description: "libera os medidores, tipo !medidor, !gado, !gay, !rankinggay e !euanime"},
		{name: modeStats, description: "libera as estatísticas, tipo !ranking, !palavra, !tu, !palhaço, !retrospectiva, !primeiramsg e !horário"},
		{name: modeJokes, description: "libera a zoeira, tipo !kkk, !chute, !apelido e !frase"},
		{name: modeEconomy, description: "libera as moedas do grupo: !saldo, !daily, !trabalhar, !profissao, !apostar, !cassino, !blackjack, !crash, !corrida, !roubar, !assalto, !quadrilha, !sequestrar, !venderrefem, !matar, !reviver, !prender, !fugir, !denunciar, !processar, !votar, !presidente, !decreto, !impeachment, !golpe, !transferir, !poupar, !conjunta, !agiota, !acoes, !loteria, !bolao, !ricos, !imposto, !fianca, !hospital, !saude, !resgate, !seguranca, !imovel, !empregados, !titulo, !doar, !zoar, !abencoar e !amaldicoar, e os itens: !mochila, !item, !loja, !petshop, !luxo, !mercadonegro, !comprar, !vender, !aceitar, !recusar, !leilao, !lance, !dar, !usar, !pescar, !minerar, !vasculhar, !catar e !furtar. Com ele, quem acerta o !quiz, o !quemfalou, a !forca, o !termo, o !completa, o !emoji, a !bandeira ou a mentira do !2verdades1mentira ou vence o !stop ou o !impostor ganha moedas (se pelo menos 2 pessoas jogarem), o !roleta e o !jokenpo @pessoa aceitam aposta, o !tu mostra o saldo, quem casou pelo pedido do consentimento e pede divórcio paga pensão, quem é pego no !trair ou no !traircom paga indenização e quem é preso no !prender ou num !assalto que deu errado fica sem apostar, roubar, furtar, sequestrar e assaltar até sair da cadeia, quem é sequestrado fica igual até sair do cativeiro ou conseguir !fugir, quem morre no !matar, de overdose ou com a saúde zerada (dá pra se machucar no !trabalhar das profissões perigosas e nas extrações) fica sem nenhum comando de economia até ressuscitar ou alguém !reviver e o enterro leva um pedaço do saldo, e quem pega moedas com o !agiota e passa do prazo pode ser morto, preso, sequestrado ou processado por quem emprestou"},
		{name: modeAI, description: "libera o !ia, o !resumo, o !roast, o !signo, o !quemsou, o !quizia, o !emoji, o !processaria, o !cancelar, o !fofoca e o !ficia", submodes: []chatMode{
			{name: modeAIUnlimited, description: "a IA para de contar quantas perguntas cada um fez na hora e responde sem pausa entre uma pergunta e outra"},
			{name: modeAIMeddling, description: "libera o !perseguir, que faz a IA responder toda mensagem da pessoa marcada"},
			{name: modeAIInvent, description: "a IA responde quando alguém manda um comando que não existe, como se ele existisse"},
		}},
		{name: modeModeration, description: "libera pros admins o !ban, o !promover, o !rebaixar, o !mutar e o !desmutar (o bot também precisa ser admin), o !algemar, o !desalgemar, o !advertir (na 3ª advertência a pessoa é banida), o !perdoar e o !boasvindas, e pra todo mundo o !advertencias. O bot passa a apagar as mensagens de quem tá mutado, reagir com 🔒 nas de quem tá algemado e dar boas-vindas a quem entra no grupo"},
		{name: modeWords, description: "o bot responde algumas palavras mesmo sem o !, tipo arrogante, nojo, aura, ego ou chorar"},
		{name: modeLaughs, description: "o bot manda uma figurinha de risada quando alguém ri (kkkk, hahaha, rsrs...)"},
		{name: modeUtilities, description: "libera o !baixar, o !figurinha, o !mp3, o !musica, o !som, o !transcrever, o !lembrete, o !buscar, o !aniversario (com parabéns automáticos no dia), o !cotacao, o !clima e o !ocr", submodes: []chatMode{
			{name: modeAutoDownload, description: "o bot baixa sozinho e manda os vídeos, imagens e gifs dos links do Instagram, TikTok, X e outros, sem precisar do !baixar"},
		}},
	}
	var all, submodes []string
	for _, m := range modes {
		if len(m.includes) == 0 {
			all = append(all, m.name)
		}
		for _, sub := range m.submodes {
			submodes = append(submodes, sub.name)
		}
	}
	return append([]chatMode{{name: modeAll, description: "liga ou desliga de uma vez todos os modos, menos os submodos (" + strings.Join(submodes, ", ") + ")", includes: all}}, modes...)
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

func (s *modeStore) enabled(ctx context.Context, chat types.JID, mode string) (bool, error) {
	var on bool
	err := s.db.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT 1 FROM conversa_modos WHERE chat = ? AND modo = ?)",
		chat.String(), mode,
	).Scan(&on)
	return on, err
}

func (s *modeStore) enable(ctx context.Context, chat types.JID, modes []string) ([]string, error) {
	return s.change(ctx, "INSERT OR IGNORE INTO conversa_modos (chat, modo) VALUES (?, ?)", chat, modes)
}

func (s *modeStore) disable(ctx context.Context, chat types.JID, modes []string) ([]string, error) {
	return s.change(ctx, "DELETE FROM conversa_modos WHERE chat = ? AND modo = ?", chat, modes)
}

func (s *modeStore) change(ctx context.Context, query string, chat types.JID, modes []string) (changed []string, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, tx.Rollback())
		}
	}()

	for _, mode := range modes {
		res, err := tx.ExecContext(ctx, query, chat.String(), mode)
		if err != nil {
			return nil, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return nil, err
		}
		if n > 0 {
			changed = append(changed, mode)
		}
	}
	return changed, tx.Commit()
}

func (s *modeStore) active(ctx context.Context, chat types.JID) (modes []string, err error) {
	rows, err := s.db.QueryContext(ctx, "SELECT modo FROM conversa_modos WHERE chat = ?", chat.String())
	if err != nil {
		return nil, err
	}
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

func (b *bot) categoryEnabled(ctx context.Context, chat types.JID, c category) (bool, error) {
	if c.mode == "" {
		return true, nil
	}
	return b.chatModeOn(ctx, chat, c.mode)
}

func (b *bot) chatModeOn(ctx context.Context, chat types.JID, mode string) (bool, error) {
	active, err := b.modes.active(ctx, chat.ToNonAD())
	if err != nil {
		return false, err
	}
	return modeActive(active, mode), nil
}

func (b *bot) answerModes(ctx context.Context, evt *events.Message) {
	answers := map[string]command{modeWords: b.answerWords, modeLaughs: b.answerLaugh, modeAutoDownload: b.answerLinks, modeAIMeddling: b.answerPersecuted}
	var wg sync.WaitGroup
	for mode, answer := range answers {
		wg.Go(func() {
			if err := answer(ctx, evt); err != nil {
				b.client.Log.Errorf("responder modo %s: %v", mode, err)
			}
		})
	}
	wg.Wait()
}

type wordTrigger struct {
	prefix string
	run    command
}

func (b *bot) answerWords(ctx context.Context, evt *events.Message) error {
	triggers := []wordTrigger{
		{"arrog", b.arrogante},
		{"noj", b.nojo},
		{"aura", b.aura},
		{"ego", b.ego},
		{"chor", b.chorar},
	}
	words := strings.FieldsFunc(foldLetters(messageText(evt.Message)), func(r rune) bool { return !unicode.IsLetter(r) })
	i := slices.IndexFunc(triggers, func(t wordTrigger) bool {
		return slices.ContainsFunc(words, func(word string) bool { return strings.HasPrefix(word, t.prefix) })
	})
	if i < 0 {
		return nil
	}
	on, err := b.modes.enabled(ctx, evt.Info.Chat.ToNonAD(), modeWords)
	if err != nil || !on {
		return err
	}
	return triggers[i].run(ctx, evt)
}

func (b *bot) toggleChatMode(ctx context.Context, evt *events.Message, on bool, args []string) error {
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
	requested := strings.Fields(foldLetters(modesText))
	if len(requested) == 0 || (toGroup && groupName == "") {
		return b.sendText(ctx, chat, usageText)
	}
	if i := slices.IndexFunc(requested, func(name string) bool { _, isMode := byName[name]; return !isMode }); i >= 0 {
		return b.sendText(ctx, chat, fmt.Sprintf("Não existe o modo *%s*. %s", requested[i], usageText))
	}

	target, place, problem, err := b.modeTarget(ctx, chat, groupName, toGroup)
	if err != nil {
		return err
	}
	if problem != "" {
		return b.sendText(ctx, chat, problem)
	}

	selected := make([]chatMode, len(requested))
	var members []string
	for i, name := range requested {
		selected[i] = byName[name]
		members = append(members, selected[i].members()...)
	}
	change := b.modes.disable
	if on {
		change = b.modes.enable
	}
	changed, err := change(ctx, target, members)
	if err != nil {
		return err
	}
	results := map[[2]bool]string{
		{true, true}:   "✅ Modo *%[1]s* ativado %[2]s: %[3]s.",
		{true, false}:  "O modo *%[1]s* já tava ativado %[2]s.",
		{false, true}:  "✅ Modo *%[1]s* desativado %[2]s.",
		{false, false}: "O modo *%[1]s* já tava desativado %[2]s.",
	}
	lines := make([]string, len(selected))
	for i, m := range selected {
		modeChanged := slices.ContainsFunc(m.members(), func(name string) bool { return slices.Contains(changed, name) })
		lines[i] = fmt.Sprintf(results[[2]bool{on, modeChanged}], m.name, place, m.description)
	}
	if !on {
		return b.sendText(ctx, chat, strings.Join(lines, "\n"))
	}
	warnings, err := b.parentWarnings(ctx, target, selected)
	if err != nil {
		return err
	}
	return b.sendText(ctx, chat, strings.Join(append(lines, warnings...), "\n"))
}

func (b *bot) parentWarnings(ctx context.Context, target types.JID, selected []chatMode) ([]string, error) {
	if !slices.ContainsFunc(selected, func(m chatMode) bool { return m.parent != "" }) {
		return nil, nil
	}
	active, err := b.modes.active(ctx, target)
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

func (b *bot) modeTarget(ctx context.Context, chat types.JID, groupName string, toGroup bool) (target types.JID, place, problem string, err error) {
	if !toGroup {
		return chat.ToNonAD(), "nesta conversa", "", nil
	}
	entries, problem, err := b.groupByName(ctx, groupName)
	if err != nil || problem != "" {
		return types.EmptyJID, "", problem, err
	}
	return entries[0].ids[0], "no grupo *" + entries[0].name + "*", "", nil
}

func (b *bot) modos(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat
	active, err := b.modes.active(ctx, chat.ToNonAD())
	if err != nil {
		return err
	}
	lines := append(chatModeLines(active), "", "_O dono do bot liga e desliga com !modo ativar <modo> e !modo desativar <modo>_")
	return b.sendText(ctx, chat, strings.Join(lines, "\n"))
}

func chatModeLines(active []string) []string {
	lines := []string{"🧩 *Modos desta conversa*"}
	for _, m := range chatModes() {
		lines = append(lines, fmt.Sprintf("• %s: *%s* (%s)", m.name, modeStatus(active, m), m.description))
		for _, sub := range m.submodes {
			lines = append(lines, fmt.Sprintf("    ↳ %s: *%s* (%s)", sub.name, modeStatus(active, sub), sub.description))
		}
	}
	return lines
}

func modeStatus(active []string, m chatMode) string {
	if allActive(active, m.members()) {
		return "ligado"
	}
	return "desligado"
}

func allActive(active, modes []string) bool {
	return !slices.ContainsFunc(modes, func(mode string) bool { return !slices.Contains(active, mode) })
}
