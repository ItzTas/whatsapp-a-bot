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
	modeWords     = "palavras"
	modeLaughs    = "risada"
	modeRomance   = "romance"
	modeFamily    = "familia"
	modeGames     = "jogos"
	modeMeters    = "medidores"
	modeStats     = "estatisticas"
	modeJokes     = "zoeira"
	modeAI        = "ia"
	modeFun       = "diversao"
	modeUtilities = "utilidades"
)

type chatMode struct {
	name        string
	description string
	includes    []string
}

func (m chatMode) members() []string {
	if len(m.includes) > 0 {
		return m.includes
	}
	return []string{m.name}
}

func chatModes() []chatMode {
	return []chatMode{
		{name: modeFun, description: "liga ou desliga de uma vez romance, familia, jogos, medidores, estatisticas e zoeira", includes: []string{modeRomance, modeFamily, modeGames, modeMeters, modeStats, modeJokes}},
		{name: modeRomance, description: "libera os comandos de romance, tipo !casar, !namorar, !ship e !beijo"},
		{name: modeFamily, description: "libera o !adotar e o !familia"},
		{name: modeGames, description: "libera os jogos, tipo !bmc, !vdd, !forca e !roleta"},
		{name: modeMeters, description: "libera os medidores, tipo !medidor, !gado e !gay"},
		{name: modeStats, description: "libera as estatísticas, tipo !ranking, !palavra, !tu e !palhaço"},
		{name: modeJokes, description: "libera a zoeira, tipo !kkk, !chute, !apelido e !frase"},
		{name: modeAI, description: "libera o !ia"},
		{name: modeWords, description: "o bot responde algumas palavras mesmo sem o !, tipo arrogante ou nojo"},
		{name: modeLaughs, description: "o bot manda uma figurinha de risada quando alguém ri (kkkk, hahaha, rsrs...)"},
		{name: modeUtilities, description: "o bot baixa e manda os vídeos, imagens e gifs dos links do Instagram, TikTok, X e outros, e libera o !baixar, o !figurinha e o !mp3"},
	}
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
	return b.modes.enabled(ctx, chat.ToNonAD(), c.mode)
}

func (b *bot) answerModes(ctx context.Context, evt *events.Message) {
	answers := map[string]command{modeWords: b.answerWords, modeLaughs: b.answerLaugh, modeUtilities: b.answerLinks}
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
	modes := chatModes()
	byName := make(map[string]chatMode, len(modes))
	names := make([]string, len(modes))
	for i, m := range modes {
		byName[m.name] = m
		names[i] = m.name
	}
	usageText := "Use assim: !modo ativar jogos (ou !modo desativar jogos). Dá pra mexer em vários de uma vez, tipo !modo ativar jogos zoeira ia. Os modos são " + strings.Join(names, ", ") + ". Pra mexer em outro grupo, coloque -- e o nome dele depois dos modos, tipo !modo ativar jogos ia -- Galera."

	modesText, groupName, toGroup := strings.Cut(strings.ReplaceAll(strings.Join(args, " "), ",", " "), "--")
	groupName = strings.TrimSpace(groupName)
	requested := strings.Fields(modesText)
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
	return b.sendText(ctx, chat, strings.Join(lines, "\n"))
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
		status := "desligado"
		if allActive(active, m.members()) {
			status = "ligado"
		}
		lines = append(lines, fmt.Sprintf("• %s: *%s* (%s)", m.name, status, m.description))
	}
	return lines
}

func allActive(active, modes []string) bool {
	return !slices.ContainsFunc(modes, func(mode string) bool { return !slices.Contains(active, mode) })
}
