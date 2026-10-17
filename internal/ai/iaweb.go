package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"syscall"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/llm"
	"golang.org/x/net/html"
)

const (
	webSearchEndpoint = "https://lite.duckduckgo.com/lite/"
	webUserAgent      = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0 Safari/537.36"
	webMaxResults     = 5
	webMaxBodyBytes   = 2 << 20
	webPageMaxRunes   = 2500
)

var (
	sharedAddressSpace = netip.MustParsePrefix("100.64.0.0/10")
	webClient          = &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			DialContext:         (&net.Dialer{Timeout: 5 * time.Second, Control: dialPublicOnly}).DialContext,
			TLSHandshakeTimeout: 5 * time.Second,
		},
	}
)

type webResult struct {
	title   string
	link    string
	snippet string
}

func webTools() []llm.Tool {
	return []llm.Tool{
		{
			Name:        "buscar_na_internet",
			Description: "Pesquisa na internet e devolve os primeiros resultados com título, link e resumo. Use pra notícias, preços, resultados de jogos, fatos atuais ou qualquer coisa que você não sabe com certeza.",
			Properties: map[string]any{
				"consulta": map[string]any{"type": "string", "description": "o que pesquisar, do jeito que se digita num buscador"},
			},
			Required: []string{"consulta"},
			Run:      searchWebTool,
		},
		{
			Name:        "abrir_pagina",
			Description: "Abre um link da internet e devolve o texto da página, cortado. Use depois de buscar_na_internet quando o resumo não bastar, ou quando alguém mandar um link.",
			Properties: map[string]any{
				"url": map[string]any{"type": "string", "description": "endereço completo, começando com http:// ou https://"},
			},
			Required: []string{"url"},
			Run:      openPageTool,
		},
	}
}

func searchWebTool(ctx context.Context, input json.RawMessage) (string, error) {
	args, err := decodeToolInput[struct {
		Query string `json:"consulta"`
	}](input)
	if err != nil {
		return "", err
	}
	query := strings.TrimSpace(args.Query)
	if query == "" {
		return "", errors.New("faltou o que pesquisar")
	}
	results, err := searchWeb(ctx, query)
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return fmt.Sprintf("Nenhum resultado pra %q.", query), nil
	}
	lines := make([]string, 0, len(results))
	for i, r := range results {
		lines = append(lines, fmt.Sprintf("%d. %s\n%s\n%s", i+1, r.title, r.link, r.snippet))
	}
	return strings.Join(lines, "\n\n"), nil
}

func searchWeb(ctx context.Context, query string) ([]webResult, error) {
	doc, err := fetchHTML(ctx, webSearchEndpoint+"?"+url.Values{"q": {query}, "kl": {"br-pt"}}.Encode())
	if err != nil {
		return nil, fmt.Errorf("pesquisar: %w", err)
	}
	var results []webResult
	organic := false
	for n := range doc.Descendants() {
		if len(results) > webMaxResults {
			break
		}
		if n.Type != html.ElementNode {
			continue
		}
		if n.Data == "a" && core.HasClass(n, "result-link") {
			var link string
			link, organic = organicLink(core.Attr(n, "href"))
			if organic {
				results = append(results, webResult{title: nodeText(n), link: link})
			}
			continue
		}
		if organic && n.Data == "td" && core.HasClass(n, "result-snippet") {
			results[len(results)-1].snippet = nodeText(n)
		}
	}
	return results[:min(len(results), webMaxResults)], nil
}

func organicLink(href string) (string, bool) {
	parsed, err := url.Parse(href)
	if err != nil {
		return "", false
	}
	target, err := url.Parse(parsed.Query().Get("uddg"))
	if err != nil || target.Host == "" || strings.HasSuffix(target.Hostname(), "duckduckgo.com") {
		return "", false
	}
	return target.String(), true
}

func openPageTool(ctx context.Context, input json.RawMessage) (string, error) {
	args, err := decodeToolInput[struct {
		URL string `json:"url"`
	}](input)
	if err != nil {
		return "", err
	}
	target, err := url.Parse(strings.TrimSpace(args.URL))
	if err != nil || (target.Scheme != "http" && target.Scheme != "https") || target.Host == "" {
		return "", fmt.Errorf("link %q inválido, precisa começar com http:// ou https://", args.URL)
	}
	doc, err := fetchHTML(ctx, target.String())
	if err != nil {
		return "", fmt.Errorf("abrir %s: %w", target, err)
	}
	text := pageText(doc)
	if text == "" {
		return "A página não tem texto que dê pra ler.", nil
	}
	return core.TruncateRunes(text, webPageMaxRunes), nil
}

func fetchHTML(ctx context.Context, address string) (doc *html.Node, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", webUserAgent)
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en;q=0.5")

	resp, err := webClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, resp.Body.Close()) }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("o site respondeu %s", resp.Status)
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && !strings.Contains(contentType, "html") && !strings.HasPrefix(contentType, "text/") {
		return nil, fmt.Errorf("o link não é uma página de texto (%s)", contentType)
	}
	return html.Parse(io.LimitReader(resp.Body, webMaxBodyBytes))
}

func dialPublicOnly(_, address string, _ syscall.RawConn) error {
	addrPort, err := netip.ParseAddrPort(address)
	if err != nil {
		return fmt.Errorf("endereço %q inválido", address)
	}
	ip := addrPort.Addr().Unmap()
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() || sharedAddressSpace.Contains(ip) {
		return errors.New("endereço de rede interna bloqueado")
	}
	return nil
}

func pageText(doc *html.Node) string {
	skipped := map[string]bool{"script": true, "style": true, "noscript": true, "svg": true, "head": true, "nav": true, "footer": true, "form": true, "iframe": true}
	var parts []string
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && skipped[n.Data] {
			return
		}
		if n.Type == html.TextNode {
			if text := strings.Join(strings.Fields(n.Data), " "); text != "" {
				parts = append(parts, text)
			}
		}
		for child := range n.ChildNodes() {
			walk(child)
		}
	}
	walk(doc)
	return strings.Join(parts, " ")
}

func nodeText(n *html.Node) string {
	return strings.Join(strings.Fields(core.TextContent(n)), " ")
}
