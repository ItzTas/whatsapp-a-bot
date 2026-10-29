package utilities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

const (
	infoSearchEndpoint  = "https://pt.wikipedia.org/w/rest.php/v1/search/page"
	infoSummaryEndpoint = "https://pt.wikipedia.org/api/rest_v1/page/summary/"
	infoUserAgent       = "whatsapp-a-bot/1.0 (bot de WhatsApp de grupo)"
	infoTimeout         = 20 * time.Second
	infoQueryMax        = 100
	infoCandidates      = 5
	infoOthersShown     = 3
	infoExtractMax      = 1500
	infoImageMaxBytes   = 5 << 20
)

var errInfoNotFound = errors.New("nada encontrado")

type infoPage struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Extract     string `json:"extract"`
	Thumbnail   struct {
		Source string `json:"source"`
	} `json:"thumbnail"`
	ContentURLs struct {
		Desktop struct {
			Page string `json:"page"`
		} `json:"desktop"`
	} `json:"content_urls"`
}

type infoResult struct {
	page   infoPage
	others []string
	image  []byte
}

func (b *bot) info(ctx context.Context, evt *events.Message) error {
	query := infoQuery(evt.Message)
	if query == "" {
		return b.ReplyTo(ctx, evt, "📚 Use assim: !info <assunto>, tipo !info capivara ou !info Santos Dumont. Eu procuro na Wikipédia e mando o resumo. Também dá pra responder uma mensagem com !info")
	}
	if utf8.RuneCountInString(query) > infoQueryMax {
		return b.ReplyTo(ctx, evt, fmt.Sprintf("Assunto grande demais, o máximo é %d letras", infoQueryMax))
	}
	if name, isCommand := core.CommandName(query); isCommand {
		return b.commandInfo(ctx, evt, name)
	}

	stop := b.Typing(ctx, evt.Info.Chat)
	result, err := b.lookUpInfo(ctx, query)
	stop()
	if errors.Is(err, errInfoNotFound) {
		return b.ReplyTo(ctx, evt, fmt.Sprintf("🤷 Não achei nada sobre \"%s\" na Wikipédia. Tenta escrever de outro jeito", query))
	}
	if err != nil {
		b.Client.Log.Warnf("!info %q: %v", query, err)
		return b.ReplyTo(ctx, evt, "A Wikipédia não respondeu agora 😕 Tenta de novo daqui a pouco")
	}
	return b.sendInfo(ctx, evt, result)
}

func (b *bot) commandInfo(ctx context.Context, evt *events.Message, name string) error {
	text, found, err := b.CommandInfo(ctx, evt, name)
	if err != nil {
		return err
	}
	if !found {
		return b.ReplyTo(ctx, evt, fmt.Sprintf("🤷 Não existe nenhum comando !%s. Veja a lista com !comandos", name))
	}
	return b.ReplyTo(ctx, evt, text)
}

func infoQuery(msg *waE2E.Message) string {
	query := core.TextAfterFields(core.MessageText(msg), 1)
	if query != "" {
		return query
	}
	quoted := msg.GetExtendedTextMessage().GetContextInfo().GetQuotedMessage()
	return strings.Join(strings.Fields(core.MessageText(quoted)), " ")
}

func (b *bot) lookUpInfo(ctx context.Context, query string) (infoResult, error) {
	ctx, cancel := context.WithTimeout(ctx, infoTimeout)
	defer cancel()

	titles, err := searchInfoTitles(ctx, query)
	if err != nil {
		return infoResult{}, err
	}
	for i, title := range titles {
		page, err := fetchInfoSummary(ctx, title)
		if err != nil {
			return infoResult{}, err
		}
		if page.Type == "disambiguation" || page.Extract == "" {
			continue
		}
		others := append(titles[:i:i], titles[i+1:]...)
		return infoResult{page: page, others: others[:min(len(others), infoOthersShown)], image: b.infoImage(ctx, page)}, nil
	}
	return infoResult{}, errInfoNotFound
}

func searchInfoTitles(ctx context.Context, query string) ([]string, error) {
	params := url.Values{"q": {query}, "limit": {fmt.Sprint(infoCandidates)}}
	var found struct {
		Pages []struct {
			Key string `json:"key"`
		} `json:"pages"`
	}
	if err := fetchInfoJSON(ctx, infoSearchEndpoint+"?"+params.Encode(), &found); err != nil {
		return nil, fmt.Errorf("pesquisar: %w", err)
	}
	if len(found.Pages) == 0 {
		return nil, errInfoNotFound
	}
	titles := make([]string, 0, len(found.Pages))
	for _, p := range found.Pages {
		titles = append(titles, p.Key)
	}
	return titles, nil
}

func fetchInfoSummary(ctx context.Context, title string) (infoPage, error) {
	var page infoPage
	if err := fetchInfoJSON(ctx, infoSummaryEndpoint+url.PathEscape(title), &page); err != nil {
		return infoPage{}, fmt.Errorf("resumo de %q: %w", title, err)
	}
	return page, nil
}

func (b *bot) infoImage(ctx context.Context, page infoPage) []byte {
	if page.Thumbnail.Source == "" {
		return nil
	}
	data, err := infoGet(ctx, page.Thumbnail.Source, infoImageMaxBytes)
	if err != nil {
		b.Client.Log.Warnf("!info: baixar a foto de %q: %v", page.Title, err)
		return nil
	}
	sendable := map[string]bool{"image/jpeg": true, "image/png": true}
	if !sendable[http.DetectContentType(data)] {
		return nil
	}
	return data
}

func (b *bot) sendInfo(ctx context.Context, evt *events.Message, result infoResult) error {
	text := infoText(result)
	if result.image != nil {
		err := b.SendImage(ctx, evt.Info.Chat, core.Outgoing{Data: result.image, Mimetype: http.DetectContentType(result.image), Caption: text}, core.ReplyTo(evt))
		if err == nil {
			return nil
		}
		b.Client.Log.Warnf("!info: mandar a foto: %v", err)
	}
	_, err := b.Client.SendMessage(ctx, evt.Info.Chat, &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{Text: proto.String(text), ContextInfo: core.ReplyTo(evt)},
	})
	return err
}

func infoText(result infoResult) string {
	page := result.page
	lines := []string{"📚 *" + page.Title + "*"}
	if page.Description != "" {
		lines = append(lines, "_"+page.Description+"_")
	}
	lines = append(lines, "", core.TruncateRunes(page.Extract, infoExtractMax))
	if page.ContentURLs.Desktop.Page != "" {
		lines = append(lines, "", "🔗 "+page.ContentURLs.Desktop.Page)
	}
	if len(result.others) > 0 {
		others := make([]string, 0, len(result.others))
		for _, key := range result.others {
			others = append(others, strings.ReplaceAll(key, "_", " "))
		}
		lines = append(lines, "", "🔎 Também achei: "+strings.Join(others, ", "))
	}
	return strings.Join(lines, "\n")
}

func fetchInfoJSON(ctx context.Context, link string, out any) error {
	data, err := infoGet(ctx, link, publicJSONMaxBytes)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("ler resposta: %w", err)
	}
	return nil
}

func infoGet(ctx context.Context, link string, maxBytes int64) (data []byte, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", infoUserAgent)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, resp.Body.Close()) }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %s", resp.Status)
	}
	data, err = io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("resposta passou de %d bytes", maxBytes)
	}
	return data, nil
}
