package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"slices"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

const (
	characterPool          = 2000
	characterPageSize      = 50
	characterCandidates    = 5
	characterReasonMax     = 400
	characterImageMaxBytes = 5 << 20
	anilistTimeout         = 20 * time.Second
	anilistEndpoint        = "https://graphql.anilist.co"
	anilistDown            = "😵 A AniList não respondeu, tenta de novo daqui a pouco"
	anilistEmpty           = "🤷 A AniList não devolveu nenhum personagem, tenta de novo"
	anilistQuery           = `query ($page: Int, $perPage: Int, $exclude: [Int]) {
  Page(page: $page, perPage: $perPage) {
    characters(sort: FAVOURITES_DESC, id_not_in: $exclude) {
      id
      name { full }
      image { large }
      media(perPage: 1, sort: POPULARITY_DESC) { nodes { title { romaji english } } }
    }
  }
}`
)

type character struct {
	id    int
	name  string
	work  string
	image string
}

type characterMatch struct {
	character
	reason string
}

type anilistTitle struct {
	Romaji  string `json:"romaji"`
	English string `json:"english"`
}

type anilistResponse struct {
	Data struct {
		Page struct {
			Characters []struct {
				ID   int `json:"id"`
				Name struct {
					Full string `json:"full"`
				} `json:"name"`
				Image struct {
					Large string `json:"large"`
				} `json:"image"`
				Media struct {
					Nodes []struct {
						Title anilistTitle `json:"title"`
					} `json:"nodes"`
				} `json:"media"`
			} `json:"characters"`
		} `json:"Page"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func (b *bot) quemsou(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := firstMentionOrSender(evt).ToNonAD()

	var (
		candidates    []character
		candidatesErr error
		name          string
		texts         []string
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error { candidates, candidatesErr = b.characterCandidates(gctx, chat); return nil })
	g.Go(func() error { name = b.displayName(gctx, person, b.chatNicknames(gctx, chat)); return nil })
	g.Go(func() (err error) { texts, err = b.messages.texts(gctx, chat, b.userAliases(gctx, person)); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler mensagens de %s: %w", person, err)
	}
	if candidatesErr != nil {
		b.client.Log.Warnf("!quemsou: buscar personagens: %v", candidatesErr)
		return b.replyTo(ctx, evt, anilistDown)
	}
	if len(candidates) == 0 {
		return b.replyTo(ctx, evt, anilistEmpty)
	}

	stopThinking := b.thinking(ctx, evt)
	match := b.matchCharacter(ctx, evt, name, slices.DeleteFunc(texts, isCommand), candidates)
	stopThinking()

	if err := b.messages.markCharacterUsed(ctx, chat, match.id); err != nil {
		b.client.Log.Warnf("!quemsou: marcar personagem %d como usado: %v", match.id, err)
	}
	return b.sendCharacter(ctx, evt, person, match)
}

func (b *bot) sendCharacter(ctx context.Context, evt *events.Message, person types.JID, match characterMatch) error {
	chat := evt.Info.Chat.ToNonAD()
	image, imageErr := downloadCharacterImage(ctx, match.image)
	caption := b.withNicknames(ctx, chat, characterCaption(person, match))
	reply := &waE2E.ContextInfo{
		StanzaID:      proto.String(evt.Info.ID),
		Participant:   proto.String(evt.Info.Sender.ToNonAD().String()),
		QuotedMessage: evt.Message,
		MentionedJID:  []string{person.String()},
	}
	if imageErr == nil {
		return b.sendImage(ctx, chat, outgoing{data: image, mimetype: http.DetectContentType(image), caption: caption}, reply)
	}
	b.client.Log.Warnf("baixar foto de %s: %v", match.name, imageErr)
	_, err := b.client.SendMessage(ctx, chat, &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{Text: proto.String(caption), ContextInfo: reply},
	})
	return err
}

func (b *bot) matchCharacter(ctx context.Context, evt *events.Message, name string, texts []string, candidates []character) characterMatch {
	if len(texts) == 0 || len(b.assistant.tiers) == 0 {
		return randomCharacterMatch(candidates)
	}
	rand.Shuffle(len(texts), func(i, j int) { texts[i], texts[j] = texts[j], texts[i] })
	raw, err := b.assistant.ask(ctx, b.iaAsker(ctx, evt.Info.Chat, evt.Info.Sender.User), func(contextSize int) string {
		return personPrompt(name, texts[:min(contextSize, len(texts))], characterRequest(name, candidates))
	})
	if err != nil {
		b.client.Log.Warnf("!quemsou: %v", err)
		return randomCharacterMatch(candidates)
	}
	match, err := parseCharacterMatch(raw, candidates)
	if err != nil {
		b.client.Log.Warnf("!quemsou: escolha inválida da IA: %v: %q", err, raw)
		return randomCharacterMatch(candidates)
	}
	return match
}

func characterRequest(name string, candidates []character) string {
	lines := make([]string, 0, len(candidates)+3)
	lines = append(lines, fmt.Sprintf("Qual destes personagens mais combina com %s, pelo jeito que a pessoa escreve, pelas manias e pelos assuntos das mensagens acima?", name))
	for i, c := range candidates {
		lines = append(lines, fmt.Sprintf("%d. %s (%s)", i+1, c.name, c.work))
	}
	return strings.Join(append(lines, "", `Responda só com um JSON, sem texto antes ou depois e sem bloco de código, neste formato:
{"personagem": 1, "motivo": "..."}
O campo "personagem" é o número do escolhido na lista.
O "motivo" tem uma ou duas frases zoeiras ligando a pessoa ao personagem, citando um trechinho ou uma mania dela. Exemplo: "fala 'tô cansado' umas 40 vezes por dia, só podia ser o mais preguiçoso da lista".`), "\n")
}

func parseCharacterMatch(raw string, candidates []character) (characterMatch, error) {
	start, end := strings.Index(raw, "{"), strings.LastIndex(raw, "}")
	if start < 0 || end < start {
		return characterMatch{}, errors.New("sem JSON na resposta")
	}
	var choice struct {
		Character int    `json:"personagem"`
		Reason    string `json:"motivo"`
	}
	if err := json.Unmarshal([]byte(raw[start:end+1]), &choice); err != nil {
		return characterMatch{}, err
	}
	if choice.Character < 1 || choice.Character > len(candidates) {
		return characterMatch{}, fmt.Errorf("personagem %d fora da lista", choice.Character)
	}
	reason := strings.TrimSpace(choice.Reason)
	if reason == "" {
		return characterMatch{}, errors.New("motivo vazio")
	}
	return characterMatch{character: candidates[choice.Character-1], reason: truncateRunes(reason, characterReasonMax)}, nil
}

var characterReasons = []string{
	"Os astros decidiram e não cabe recurso ✨",
	"Nem precisei pensar muito, é a sua cara 😌",
	"A semelhança chega a assustar 👀",
	"Confia, o bot sabe das coisas 🔮",
	"Foi a primeira coisa que veio na cabeça, e a primeira impressão é a que fica 🤝",
}

func randomCharacterMatch(candidates []character) characterMatch {
	return characterMatch{
		character: candidates[rand.IntN(len(candidates))],
		reason:    characterReasons[rand.IntN(len(characterReasons))],
	}
}

func characterCaption(person types.JID, match characterMatch) string {
	lines := []string{fmt.Sprintf("🎭 Personagem de @%s: *%s*", person.User, match.name)}
	if match.work != "" {
		lines = append(lines, fmt.Sprintf("📺 _%s_", match.work))
	}
	return strings.Join(append(lines, "", match.reason), "\n")
}

func (b *bot) characterCandidates(ctx context.Context, chat types.JID) ([]character, error) {
	used, err := b.messages.usedCharacters(ctx, chat)
	if err != nil {
		return nil, fmt.Errorf("ler personagens já usados: %w", err)
	}
	if len(used) > characterPool-characterCandidates {
		if err := b.messages.resetUsedCharacters(ctx, chat); err != nil {
			return nil, fmt.Errorf("zerar personagens já usados: %w", err)
		}
		used = []int{}
	}
	pages := max(1, (characterPool-len(used))/characterPageSize)
	characters, err := fetchCharacters(ctx, 1+rand.IntN(pages), used)
	if err != nil {
		return nil, err
	}
	rand.Shuffle(len(characters), func(i, j int) { characters[i], characters[j] = characters[j], characters[i] })
	return characters[:min(characterCandidates, len(characters))], nil
}

func fetchCharacters(ctx context.Context, page int, exclude []int) (characters []character, err error) {
	ctx, cancel := context.WithTimeout(ctx, anilistTimeout)
	defer cancel()
	body, err := json.Marshal(map[string]any{
		"query":     anilistQuery,
		"variables": map[string]any{"page": page, "perPage": characterPageSize, "exclude": exclude},
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anilistEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, resp.Body.Close()) }()

	var parsed anilistResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("AniList respondeu %s: %w", resp.Status, err)
	}
	if len(parsed.Errors) > 0 {
		return nil, fmt.Errorf("AniList respondeu %s: %s", resp.Status, parsed.Errors[0].Message)
	}
	for _, c := range parsed.Data.Page.Characters {
		var titles []anilistTitle
		for _, n := range c.Media.Nodes {
			titles = append(titles, n.Title)
		}
		characters = append(characters, character{id: c.ID, name: c.Name.Full, work: workTitle(titles), image: c.Image.Large})
	}
	return characters, nil
}

func workTitle(titles []anilistTitle) string {
	if len(titles) == 0 {
		return ""
	}
	if titles[0].English != "" {
		return titles[0].English
	}
	return titles[0].Romaji
}

func downloadCharacterImage(ctx context.Context, link string) (data []byte, err error) {
	ctx, cancel := context.WithTimeout(ctx, anilistTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, resp.Body.Close()) }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, characterImageMaxBytes))
}

func (s *messageStore) usedCharacters(ctx context.Context, chat types.JID) (ids []int, err error) {
	rows, err := s.db.QueryContext(ctx, "SELECT personagem FROM personagens_usados WHERE chat = ?", chat.String())
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	ids = []int{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *messageStore) markCharacterUsed(ctx context.Context, chat types.JID, id int) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT OR IGNORE INTO personagens_usados (chat, personagem) VALUES (?, ?)",
		chat.String(), id,
	)
	return err
}

func (s *messageStore) resetUsedCharacters(ctx context.Context, chat types.JID) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM personagens_usados WHERE chat = ?", chat.String())
	return err
}
