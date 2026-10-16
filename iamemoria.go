package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"go.mau.fi/whatsmeow/types"
)

const (
	iaMemoryLimit    = 50
	iaMemoryMaxRunes = 300
)

type iaMemory struct {
	id      int64
	text    string
	savedAt time.Time
}

type memoryStore struct {
	db *sql.DB
}

func (s *memoryStore) save(ctx context.Context, chat types.JID, text string) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO ia_memorias (chat, texto, criada_em)
		 SELECT ?, ?, ?
		 WHERE (SELECT COUNT(*) FROM ia_memorias WHERE chat = ?) < ?`,
		chat.String(), text, time.Now().Unix(), chat.String(), iaMemoryLimit,
	)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (s *memoryStore) search(ctx context.Context, chat types.JID, text string) (memories []iaMemory, err error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, texto, criada_em FROM ia_memorias
		 WHERE chat = ? AND instr(lower(texto), lower(?)) > 0
		 ORDER BY criada_em DESC, id DESC`,
		chat.String(), text,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var m iaMemory
		var savedAt int64
		if err := rows.Scan(&m.id, &m.text, &savedAt); err != nil {
			return nil, err
		}
		m.savedAt = time.Unix(savedAt, 0)
		memories = append(memories, m)
	}
	return memories, rows.Err()
}

func (s *memoryStore) count(ctx context.Context, chat types.JID) (n int, err error) {
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM ia_memorias WHERE chat = ?", chat.String()).Scan(&n)
	return n, err
}

func (s *memoryStore) forget(ctx context.Context, chat types.JID, id int64) (bool, error) {
	res, err := s.db.ExecContext(ctx, "DELETE FROM ia_memorias WHERE chat = ? AND id = ?", chat.String(), id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (k *iaToolkit) memoryTools() []chatTool {
	return []chatTool{
		{
			name:        "guardar_memoria",
			description: fmt.Sprintf("Guarda uma anotação na memória desta conversa pra lembrar em outras perguntas, tipo um fato sobre alguém, um combinado ou algo que pediram pra você não esquecer. Escreva a anotação completa e sozinha, com o nome de quem ela fala, em até %d caracteres. A conversa guarda no máximo %d anotações.", iaMemoryMaxRunes, iaMemoryLimit),
			properties: map[string]any{
				"texto": map[string]any{"type": "string", "description": "a anotação a guardar"},
			},
			required: []string{"texto"},
			run:      k.saveMemory,
		},
		{
			name:        "buscar_memoria",
			description: "Procura nas anotações que você guardou nesta conversa. Sem texto, mostra todas. Use antes de responder sobre gostos, fatos ou combinados do grupo, ou quando perguntarem se você lembra de algo.",
			properties: map[string]any{
				"texto": map[string]any{"type": "string", "description": "palavra a procurar, sem diferenciar maiúscula (vazio mostra tudo)"},
			},
			run: k.searchMemory,
		},
		{
			name:        "esquecer_memoria",
			description: "Apaga uma anotação da memória desta conversa pelo número dela (o # que aparece no buscar_memoria). Use quando pedirem pra esquecer, quando a anotação ficou errada ou velha, ou pra abrir espaço quando a memória encher.",
			properties: map[string]any{
				"numero": map[string]any{"type": "integer", "description": "o número da anotação"},
			},
			required: []string{"numero"},
			run:      k.forgetMemory,
		},
	}
}

func (k *iaToolkit) saveMemory(ctx context.Context, input json.RawMessage) (string, error) {
	args, err := decodeToolInput[struct {
		Text string `json:"texto"`
	}](input)
	if err != nil {
		return "", err
	}
	text := strings.Join(strings.Fields(args.Text), " ")
	if text == "" {
		return "", errors.New("faltou o texto da anotação")
	}
	if n := utf8.RuneCountInString(text); n > iaMemoryMaxRunes {
		return "", fmt.Errorf("a anotação tem %d caracteres, resuma em até %d", n, iaMemoryMaxRunes)
	}
	saved, err := k.b.memories.save(ctx, k.chat, text)
	if err != nil {
		return "", err
	}
	if !saved {
		return "", fmt.Errorf("a memória desta conversa tá cheia (%d anotações), apague uma com esquecer_memoria antes ou avise que não dá pra guardar", iaMemoryLimit)
	}
	used, err := k.b.memories.count(ctx, k.chat)
	if err != nil {
		k.b.client.Log.Warnf("!ia: contar memórias de %s: %v", k.chat, err)
		return "Anotação guardada.", nil
	}
	return fmt.Sprintf("Anotação guardada (%d de %d usadas).", used, iaMemoryLimit), nil
}

func (k *iaToolkit) searchMemory(ctx context.Context, input json.RawMessage) (string, error) {
	args, err := decodeToolInput[struct {
		Text string `json:"texto"`
	}](input)
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(args.Text)
	memories, err := k.b.memories.search(ctx, k.chat, text)
	if err != nil {
		return "", err
	}
	if len(memories) == 0 && text == "" {
		return "A memória desta conversa tá vazia.", nil
	}
	if len(memories) == 0 {
		return fmt.Sprintf("Nenhuma anotação com %q.", text), nil
	}
	lines := make([]string, 0, len(memories))
	for _, m := range memories {
		lines = append(lines, fmt.Sprintf("#%d [%s] %s", m.id, m.savedAt.In(brazilTime).Format("02/01/2006"), m.text))
	}
	return strings.Join(lines, "\n"), nil
}

func (k *iaToolkit) forgetMemory(ctx context.Context, input json.RawMessage) (string, error) {
	args, err := decodeToolInput[struct {
		ID int64 `json:"numero"`
	}](input)
	if err != nil {
		return "", err
	}
	forgotten, err := k.b.memories.forget(ctx, k.chat, args.ID)
	if err != nil {
		return "", err
	}
	if !forgotten {
		return "", fmt.Errorf("não tem anotação #%d nesta conversa", args.ID)
	}
	return fmt.Sprintf("Anotação #%d apagada.", args.ID), nil
}
