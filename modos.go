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
	modeWords  = "palavras"
	modeLaughs = "risada"
)

type chatMode struct {
	name        string
	description string
}

func chatModes() []chatMode {
	return []chatMode{
		{modeWords, "o bot responde algumas palavras mesmo sem o !, tipo arrogante ou nojo"},
		{modeLaughs, "o bot manda uma figurinha de risada quando alguém ri (kkkk, hahaha, rsrs...)"},
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

func (s *modeStore) enable(ctx context.Context, chat types.JID, mode string) (bool, error) {
	res, err := s.db.ExecContext(ctx, "INSERT OR IGNORE INTO conversa_modos (chat, modo) VALUES (?, ?)", chat.String(), mode)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (s *modeStore) disable(ctx context.Context, chat types.JID, mode string) (bool, error) {
	res, err := s.db.ExecContext(ctx, "DELETE FROM conversa_modos WHERE chat = ? AND modo = ?", chat.String(), mode)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
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

func (b *bot) answerModes(ctx context.Context, evt *events.Message) {
	answers := map[string]command{modeWords: b.answerWords, modeLaughs: b.answerLaugh}
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
	usageText := "Use assim: !modo ativar palavras (ou !modo desativar palavras). Os modos são palavras e risada. Pra mexer em outro grupo, coloque o nome dele no fim."
	if len(args) == 0 {
		return b.sendText(ctx, chat, usageText)
	}
	modes := chatModes()
	i := slices.IndexFunc(modes, func(m chatMode) bool { return m.name == args[0] })
	if i < 0 {
		return b.sendText(ctx, chat, usageText)
	}
	mode := modes[i]

	target, place := chat.ToNonAD(), "nesta conversa"
	if len(args) > 1 {
		entries, problem, err := b.groupByName(ctx, strings.Join(args[1:], " "))
		if err != nil {
			return err
		}
		if problem != "" {
			return b.sendText(ctx, chat, problem)
		}
		target, place = entries[0].ids[0], "no grupo *"+entries[0].name+"*"
	}

	change := b.modes.disable
	if on {
		change = b.modes.enable
	}
	changed, err := change(ctx, target, mode.name)
	if err != nil {
		return err
	}
	results := map[[2]bool]string{
		{true, true}:   "✅ Modo *%s* ativado %s: " + mode.description + ".",
		{true, false}:  "O modo *%s* já tava ativado %s.",
		{false, true}:  "✅ Modo *%s* desativado %s. O bot volta a responder só os comandos com !.",
		{false, false}: "O modo *%s* já tava desativado %s.",
	}
	return b.sendText(ctx, chat, fmt.Sprintf(results[[2]bool{on, changed}], mode.name, place))
}

func chatModeLines(active []string) []string {
	lines := []string{"🧩 *Modos desta conversa*"}
	for _, m := range chatModes() {
		status := "desligado"
		if slices.Contains(active, m.name) {
			status = "ligado"
		}
		lines = append(lines, fmt.Sprintf("• %s: *%s* (%s)", m.name, status, m.description))
	}
	return lines
}
