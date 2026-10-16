package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

const (
	quoteEndpoint = "https://economia.awesomeapi.com.br/json/last/"
	quoteTimeout  = 15 * time.Second
)

var (
	quoteCodePattern = regexp.MustCompile(`^[A-Z0-9]{2,6}$`)
	errQuoteUnknown  = errors.New("moeda não encontrada")
)

type currencyQuote struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Bid       string `json:"bid"`
	PctChange string `json:"pctChange"`
	High      string `json:"high"`
	Low       string `json:"low"`
	Created   string `json:"create_date"`
}

func (b *bot) cotacao(ctx context.Context, evt *events.Message) error {
	arg, target := quoteArgs(textAfterFields(messageText(evt.Message), 1))
	if arg == "" {
		return b.sendQuotes(ctx, evt, quoteDefaults(target), target, quoteTitle(target))
	}
	code := quoteCode(arg)
	if !quoteCodePattern.MatchString(code) {
		return b.replyTo(ctx, evt, "💱 Use assim: !cotacao pra ver as principais, ou !cotacao <moeda>, tipo !cotacao ARS ou !cotacao iene. Bota \"em dolar\" no fim pra ver em dólar, tipo !cotacao euro em dolar")
	}
	if code == target {
		return b.replyTo(ctx, evt, quoteSelfJoke(target))
	}
	return b.sendQuotes(ctx, evt, []string{code}, target, "💱 *Cotação*")
}

func quoteArgs(arg string) (string, string) {
	fields := strings.Fields(arg)
	n := len(fields)
	if n < 2 || foldLetters(fields[n-2]) != "em" {
		return arg, "BRL"
	}
	targets := map[string]string{
		"dolar":   "USD",
		"dolares": "USD",
		"usd":     "USD",
		"real":    "BRL",
		"reais":   "BRL",
		"brl":     "BRL",
	}
	target, ok := targets[foldLetters(fields[n-1])]
	if !ok {
		return arg, "BRL"
	}
	return strings.Join(fields[:n-2], " "), target
}

func quoteDefaults(target string) []string {
	if target == "USD" {
		return []string{"BRL", "EUR", "GBP", "BTC", "ETH"}
	}
	return []string{"USD", "EUR", "GBP", "BTC", "ETH"}
}

func quoteTitle(target string) string {
	if target == "USD" {
		return "💱 *Cotação de hoje em dólar*"
	}
	return "💱 *Cotação de hoje*"
}

func quoteSelfJoke(target string) string {
	if target == "USD" {
		return "💱 Um dólar vale... um dólar 🤡"
	}
	return "💱 Um real vale... um real 🤡"
}

func (b *bot) sendQuotes(ctx context.Context, evt *events.Message, codes []string, target, title string) error {
	quotes, err := fetchQuotes(ctx, codes, target)
	if errors.Is(err, errQuoteUnknown) {
		return b.replyTo(ctx, evt, fmt.Sprintf("🤷 Não achei a moeda %s. Usa o código de 3 letras, tipo USD, EUR, ARS, JPY ou BTC", codes[0]))
	}
	if err != nil {
		b.client.Log.Warnf("!cotacao %v: %v", codes, err)
		return b.replyTo(ctx, evt, "Não consegui pegar a cotação agora 😕 Tenta de novo daqui a pouco")
	}

	lines := []string{title, ""}
	for _, q := range quotes {
		lines = append(lines, q.lines(target)...)
		lines = append(lines, "")
	}
	lines = append(lines, "_Atualizado em "+quotes[0].updatedAt()+"_")
	return b.replyTo(ctx, evt, strings.Join(lines, "\n"))
}

func quoteCode(arg string) string {
	names := map[string]string{
		"dolar":    "USD",
		"euro":     "EUR",
		"libra":    "GBP",
		"bitcoin":  "BTC",
		"btc":      "BTC",
		"ethereum": "ETH",
		"peso":     "ARS",
		"real":     "BRL",
		"iene":     "JPY",
		"yuan":     "CNY",
		"franco":   "CHF",
	}
	if code, ok := names[foldLetters(arg)]; ok {
		return code
	}
	return strings.ToUpper(arg)
}

func fetchQuotes(ctx context.Context, codes []string, target string) ([]currencyQuote, error) {
	ctx, cancel := context.WithTimeout(ctx, quoteTimeout)
	defer cancel()
	pairs := make([]string, len(codes))
	for i, c := range codes {
		pairs[i] = c + "-" + target
	}
	var raw map[string]currencyQuote
	err := fetchPublicJSON(ctx, quoteEndpoint+strings.Join(pairs, ","), &raw)
	if errors.Is(err, errPublicNotFound) {
		return nil, errQuoteUnknown
	}
	if err != nil {
		return nil, err
	}
	quotes := make([]currencyQuote, 0, len(codes))
	for _, c := range codes {
		q, ok := raw[c+target]
		if !ok {
			return nil, fmt.Errorf("%w: %s", errQuoteUnknown, c)
		}
		quotes = append(quotes, q)
	}
	return quotes, nil
}

func (q currencyQuote) lines(target string) []string {
	name, _, _ := strings.Cut(q.Name, "/")
	return []string{
		fmt.Sprintf("%s *%s* (%s)", quoteEmoji(q.Code), name, q.Code),
		fmt.Sprintf("💵 %s %s", quoteMoney(q.Bid, target), quoteChange(q.PctChange)),
		fmt.Sprintf("📊 Mínima %s · Máxima %s", quoteMoney(q.Low, target), quoteMoney(q.High, target)),
	}
}

func (q currencyQuote) updatedAt() string {
	at, err := time.ParseInLocation(time.DateTime, q.Created, reminderZone)
	if err != nil {
		return q.Created
	}
	return at.Format("02/01 às 15:04")
}

func quoteEmoji(code string) string {
	emojis := map[string]string{
		"USD": "🇺🇸",
		"EUR": "🇪🇺",
		"GBP": "🇬🇧",
		"BTC": "🪙",
		"ETH": "💎",
		"ARS": "🇦🇷",
		"JPY": "🇯🇵",
		"CNY": "🇨🇳",
		"CHF": "🇨🇭",
		"BRL": "🇧🇷",
	}
	if emoji, ok := emojis[code]; ok {
		return emoji
	}
	return "💰"
}

func quoteMoney(raw, target string) string {
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return raw
	}
	symbols := map[string]string{"BRL": "R$", "USD": "US$"}
	symbol := symbols[target]
	if value >= 1 || value <= 0 {
		cents := int64(math.Round(value * 100))
		return fmt.Sprintf("%s %s,%02d", symbol, thousands(cents/100), cents%100)
	}
	decimals := max(4, 2-int(math.Floor(math.Log10(value))))
	return symbol + " " + strings.Replace(strconv.FormatFloat(value, 'f', decimals, 64), ".", ",", 1)
}

func quoteChange(raw string) string {
	pct, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return ""
	}
	pct = math.Round(pct*100) / 100
	if pct == 0 {
		return "➖ 0,00% hoje"
	}
	label := strings.Replace(fmt.Sprintf("%+.2f%%", pct), ".", ",", 1)
	if pct < 0 {
		return "📉 " + label + " hoje"
	}
	return "📈 " + label + " hoje"
}
