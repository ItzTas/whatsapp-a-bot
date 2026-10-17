package utilities

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/net/html"
	"google.golang.org/protobuf/proto"
)

const (
	shoppingPerShop      = 3
	shoppingQueryMax     = 100
	shoppingTitleMax     = 90
	shoppingTimeout      = 20 * time.Second
	shoppingRetryDelay   = 1500 * time.Millisecond
	shoppingPageMaxBytes = 8 << 20
	showcaseTimeout      = 15 * time.Second
	showcaseTile         = "320x320+12+12"
	shoppingUserAgent    = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"
)

var (
	errShopBusy        = errors.New("a loja tá recusando requisições agora")
	errNextDataMissing = errors.New("a página veio sem o __NEXT_DATA__")
	errAmazonCaptcha   = errors.New("a Amazon pediu captcha")
	errNoPhotoLink     = errors.New("o produto veio sem foto")
	errNoProductPhotos = errors.New("nenhuma foto dos produtos baixou")
)

type listing struct {
	title  string
	price  float64
	seller string
	link   string
	photo  string
}

type shop struct {
	name   string
	search func(ctx context.Context, query string) ([]listing, error)
}

type shopResult struct {
	listings []listing
	err      error
}

func (b *bot) buscar(ctx context.Context, evt *events.Message) error {
	query := core.TextAfterFields(core.MessageText(evt.Message), 1)
	if query == "" {
		return b.ReplyTo(ctx, evt, "🛒 Use assim: !buscar <produto>, tipo !buscar iphone 13. Eu procuro no Buscapé, na Amazon e na KaBuM e mando o preço e o link")
	}
	if utf8.RuneCountInString(query) > shoppingQueryMax {
		return b.ReplyTo(ctx, evt, fmt.Sprintf("Busca grande demais, o máximo é %d letras", shoppingQueryMax))
	}

	shops := []shop{
		{"Buscapé", searchBuscape},
		{"Amazon", searchAmazon},
		{"KaBuM!", searchKabum},
	}
	stop := b.Typing(ctx, evt.Info.Chat)
	results, showcases := b.searchWithShowcases(ctx, shops, query)
	stop()

	if failure, failed := shoppingFailure(query, results); failed {
		return b.ReplyTo(ctx, evt, failure)
	}
	reply := core.ReplyTo(evt)
	for i, r := range results {
		section := shopSection(shops[i].name, r)
		if i == 0 {
			section = fmt.Sprintf("🛒 *Resultados pra \"%s\":*\n\n%s", query, section)
		}
		if err := b.sendShopSection(ctx, evt.Info.Chat, section, showcases[i], reply); err != nil {
			return err
		}
		reply = nil
	}
	return nil
}

func (b *bot) searchWithShowcases(ctx context.Context, shops []shop, query string) ([]shopResult, [][]byte) {
	results := searchShops(ctx, shops, query)
	for i, r := range results {
		if r.err != nil {
			b.Client.Log.Warnf("!buscar %q no %s: %v", query, shops[i].name, r.err)
		}
	}
	if _, failed := shoppingFailure(query, results); failed {
		return results, nil
	}
	return results, b.shopShowcases(ctx, shops, results)
}

func searchShops(ctx context.Context, shops []shop, query string) []shopResult {
	ctx, cancel := context.WithTimeout(ctx, shoppingTimeout)
	defer cancel()
	results := make([]shopResult, len(shops))
	var wg sync.WaitGroup
	for i, s := range shops {
		wg.Go(func() {
			listings, err := s.search(ctx, query)
			results[i] = shopResult{listings: listings, err: err}
		})
	}
	wg.Wait()
	return results
}

func shoppingFailure(query string, results []shopResult) (string, bool) {
	failed := 0
	found := 0
	for _, r := range results {
		if r.err != nil {
			failed++
		}
		found += len(r.listings)
	}
	if failed == len(results) {
		return "😵 Nenhuma loja respondeu agora, tenta de novo daqui a pouco", true
	}
	if found == 0 && failed == 0 {
		return fmt.Sprintf("🤷 Não achei nada pra \"%s\" no Buscapé, na Amazon nem na KaBuM", query), true
	}
	return "", false
}

func shopSection(name string, r shopResult) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "*%s*", name)
	writeShopListings(&sb, r)
	return sb.String()
}

func (b *bot) sendShopSection(ctx context.Context, chat types.JID, text string, showcase []byte, reply *waE2E.ContextInfo) error {
	if showcase != nil {
		err := b.SendImage(ctx, chat, core.Outgoing{Data: showcase, Mimetype: "image/jpeg", Caption: text}, reply)
		if err == nil {
			return nil
		}
		b.Client.Log.Warnf("!buscar: mandar as fotos: %v", err)
	}
	_, err := b.Client.SendMessage(ctx, chat, &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{Text: proto.String(text), ContextInfo: reply},
	})
	return err
}

func (b *bot) shopShowcases(ctx context.Context, shops []shop, results []shopResult) [][]byte {
	ctx, cancel := context.WithTimeout(ctx, showcaseTimeout)
	defer cancel()
	showcases := make([][]byte, len(results))
	var wg sync.WaitGroup
	for i, r := range results {
		if len(r.listings) == 0 {
			continue
		}
		wg.Go(func() {
			showcase, err := b.shopShowcase(ctx, shops[i].name, r.listings)
			if err != nil {
				b.Client.Log.Warnf("!buscar: montar as fotos do %s: %v", shops[i].name, err)
				return
			}
			showcases[i] = showcase
		})
	}
	wg.Wait()
	return showcases
}

func (b *bot) shopShowcase(ctx context.Context, shopName string, listings []listing) (showcase []byte, err error) {
	dir, err := os.MkdirTemp("", "buscar-*")
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(dir)) }()

	tiles := make([]string, len(listings))
	errs := make([]error, len(listings))
	var wg sync.WaitGroup
	for i, l := range listings {
		wg.Go(func() { tiles[i], errs[i] = saveProductPhoto(ctx, l.photo, filepath.Join(dir, strconv.Itoa(i))) })
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			b.Client.Log.Warnf("!buscar: baixar a foto %d do %s: %v", i+1, shopName, err)
			tiles[i] = "null:"
		}
	}
	if !slices.ContainsFunc(tiles, func(tile string) bool { return tile != "null:" }) {
		return nil, errNoProductPhotos
	}

	out := filepath.Join(dir, "vitrine.jpg")
	err = core.RunNiceTool(ctx, "magick", slices.Concat(
		[]string{"montage"},
		tiles,
		[]string{"-tile", fmt.Sprintf("%dx1", len(tiles)), "-geometry", showcaseTile, "-background", "white", "-quality", "85", out},
	)...)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(out)
}

func saveProductPhoto(ctx context.Context, link, file string) (string, error) {
	if !strings.HasPrefix(link, "https://") {
		return "", errNoPhotoLink
	}
	photo, err := fetchShopPageOnce(ctx, link)
	if err != nil {
		return "", err
	}
	formats := map[string]string{"image/jpeg": "jpeg", "image/png": "png", "image/webp": "webp"}
	contentType := http.DetectContentType(photo)
	format, ok := formats[contentType]
	if !ok {
		return "", fmt.Errorf("a foto veio num formato inesperado: %s", contentType)
	}
	if err := os.WriteFile(file, photo, 0o600); err != nil {
		return "", err
	}
	return format + ":" + file, nil
}

func writeShopListings(sb *strings.Builder, r shopResult) {
	if r.err != nil {
		sb.WriteString("\n⚠️ Não respondeu agora")
		return
	}
	if len(r.listings) == 0 {
		sb.WriteString("\n😕 Não achei nada aqui")
		return
	}
	for i, l := range r.listings {
		price := brl(l.price)
		if l.seller != "" {
			price += " na " + l.seller
		}
		fmt.Fprintf(sb, "\n%d. %s\n💰 %s\n🔗 %s", i+1, shortTitle(l.title), price, l.link)
	}
}

func shortTitle(title string) string {
	title = strings.Join(strings.Fields(title), " ")
	runes := []rune(title)
	if len(runes) <= shoppingTitleMax {
		return title
	}
	return strings.TrimRightFunc(string(runes[:shoppingTitleMax]), unicode.IsSpace) + "…"
}

func brl(value float64) string {
	cents := int64(math.Round(value * 100))
	return fmt.Sprintf("R$ %s,%02d", core.Thousands(cents/100), cents%100)
}

func parseBRL(text string) (float64, bool) {
	digits := strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) || r == ',' {
			return r
		}
		return -1
	}, text)
	value, err := strconv.ParseFloat(strings.Replace(digits, ",", ".", 1), 64)
	return value, err == nil && value > 0
}

func fetchShopPage(ctx context.Context, link string) ([]byte, error) {
	page, err := fetchShopPageOnce(ctx, link)
	if !errors.Is(err, errShopBusy) {
		return page, err
	}
	select {
	case <-time.After(shoppingRetryDelay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return fetchShopPageOnce(ctx, link)
}

func fetchShopPageOnce(ctx context.Context, link string) (page []byte, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", shoppingUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, resp.Body.Close()) }()
	if slices.Contains([]int{http.StatusServiceUnavailable, http.StatusTooManyRequests}, resp.StatusCode) {
		return nil, fmt.Errorf("%w: %s", errShopBusy, resp.Status)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, shoppingPageMaxBytes))
}

func nextData(page []byte) ([]byte, error) {
	_, after, ok := bytes.Cut(page, []byte(`id="__NEXT_DATA__"`))
	if !ok {
		return nil, errNextDataMissing
	}
	_, after, ok = bytes.Cut(after, []byte(">"))
	if !ok {
		return nil, errNextDataMissing
	}
	data, _, ok := bytes.Cut(after, []byte("</script>"))
	if !ok {
		return nil, errNextDataMissing
	}
	return data, nil
}

type buscapePage struct {
	Props struct {
		InitialReduxState struct {
			Hits struct {
				Hits []buscapeHit `json:"hits"`
			} `json:"hits"`
		} `json:"initialReduxState"`
	} `json:"props"`
}

type buscapeHit struct {
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	URL       string  `json:"url"`
	Image     string  `json:"image"`
	BestOffer struct {
		MerchantName string `json:"merchantName"`
	} `json:"bestOffer"`
}

func searchBuscape(ctx context.Context, query string) ([]listing, error) {
	page, err := fetchShopPage(ctx, "https://www.buscape.com.br/search?q="+url.QueryEscape(query))
	if err != nil {
		return nil, err
	}
	data, err := nextData(page)
	if err != nil {
		return nil, err
	}
	var parsed buscapePage
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("ler o JSON do Buscapé: %w", err)
	}

	var listings []listing
	for _, hit := range parsed.Props.InitialReduxState.Hits.Hits {
		if hit.Price <= 0 || hit.URL == "" {
			continue
		}
		listings = append(listings, listing{
			title:  hit.Name,
			price:  hit.Price,
			seller: hit.BestOffer.MerchantName,
			link:   "https://www.buscape.com.br" + hit.URL,
			photo:  hit.Image,
		})
		if len(listings) == shoppingPerShop {
			break
		}
	}
	return listings, nil
}

type kabumPage struct {
	Props struct {
		PageProps struct {
			Data json.RawMessage `json:"data"`
		} `json:"pageProps"`
	} `json:"props"`
}

type kabumCatalog struct {
	CatalogServer struct {
		Data []kabumProduct `json:"data"`
	} `json:"catalogServer"`
}

type kabumProduct struct {
	Code              int64   `json:"code"`
	Name              string  `json:"name"`
	FriendlyName      string  `json:"friendlyName"`
	Price             float64 `json:"price"`
	PriceWithDiscount float64 `json:"priceWithDiscount"`
	Available         bool    `json:"available"`
	Thumbnail         string  `json:"thumbnail"`
	Photos            struct {
		Large []string `json:"g"`
	} `json:"photos"`
}

func (p kabumProduct) photo() string {
	if len(p.Photos.Large) > 0 {
		return p.Photos.Large[0]
	}
	return p.Thumbnail
}

func (p kabumProduct) bestPrice() float64 {
	if p.PriceWithDiscount > 0 {
		return p.PriceWithDiscount
	}
	return p.Price
}

func searchKabum(ctx context.Context, query string) ([]listing, error) {
	page, err := fetchShopPage(ctx, "https://www.kabum.com.br/busca/"+url.PathEscape(strings.Join(strings.Fields(query), "-")))
	if err != nil {
		return nil, err
	}
	data, err := nextData(page)
	if err != nil {
		return nil, err
	}
	var parsed kabumPage
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("ler o JSON da KaBuM: %w", err)
	}
	catalog, err := decodeKabumCatalog(parsed.Props.PageProps.Data)
	if err != nil {
		return nil, fmt.Errorf("ler o catálogo da KaBuM: %w", err)
	}

	var listings []listing
	for _, p := range catalog.CatalogServer.Data {
		if !p.Available || p.bestPrice() <= 0 {
			continue
		}
		listings = append(listings, listing{
			title: p.Name,
			price: p.bestPrice(),
			link:  fmt.Sprintf("https://www.kabum.com.br/produto/%d/%s", p.Code, p.FriendlyName),
			photo: p.photo(),
		})
		if len(listings) == shoppingPerShop {
			break
		}
	}
	return listings, nil
}

func decodeKabumCatalog(raw json.RawMessage) (catalog kabumCatalog, err error) {
	if len(raw) == 0 || string(raw) == "null" {
		return catalog, nil
	}
	if raw[0] == '"' {
		var embedded string
		if err := json.Unmarshal(raw, &embedded); err != nil {
			return catalog, err
		}
		raw = json.RawMessage(embedded)
	}
	return catalog, json.Unmarshal(raw, &catalog)
}

func searchAmazon(ctx context.Context, query string) ([]listing, error) {
	page, err := fetchShopPage(ctx, "https://www.amazon.com.br/s?k="+url.QueryEscape(query))
	if err != nil {
		return nil, err
	}
	doc, err := html.Parse(bytes.NewReader(page))
	if err != nil {
		return nil, fmt.Errorf("ler o HTML da Amazon: %w", err)
	}

	var listings []listing
	for n := range doc.Descendants() {
		if core.Attr(n, "data-component-type") != "s-search-result" || core.HasClass(n, "AdHolder") {
			continue
		}
		if l, ok := amazonListing(n); ok {
			listings = append(listings, l)
		}
		if len(listings) == shoppingPerShop {
			break
		}
	}
	if len(listings) == 0 && bytes.Contains(page, []byte("/errors/validateCaptcha")) {
		return nil, errAmazonCaptcha
	}
	return listings, nil
}

func amazonListing(result *html.Node) (listing, bool) {
	asin := core.Attr(result, "data-asin")
	heading := findElement(result, func(n *html.Node) bool { return n.Data == "h2" && core.Attr(n, "aria-label") != "" })
	priceTag := findElement(result, func(n *html.Node) bool { return core.HasClass(n, "a-price") && !core.HasClass(n, "a-text-price") })
	if asin == "" || heading == nil || priceTag == nil {
		return listing{}, false
	}
	offscreen := findElement(priceTag, func(n *html.Node) bool { return core.HasClass(n, "a-offscreen") })
	if offscreen == nil {
		return listing{}, false
	}
	price, ok := parseBRL(core.TextContent(offscreen))
	if !ok {
		return listing{}, false
	}
	l := listing{
		title: core.Attr(heading, "aria-label"),
		price: price,
		link:  "https://www.amazon.com.br/dp/" + asin,
	}
	if photo := findElement(result, func(n *html.Node) bool { return core.HasClass(n, "s-image") }); photo != nil {
		l.photo = core.Attr(photo, "src")
	}
	return l, true
}

func findElement(root *html.Node, match func(*html.Node) bool) *html.Node {
	for n := range root.Descendants() {
		if n.Type == html.ElementNode && match(n) {
			return n
		}
	}
	return nil
}
