package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	stockStepEvery   = 30 * time.Minute
	stockMaxSteps    = 7 * 48
	stockFloorCents  = 100
	stockReversion   = 0.02
	stockFeePercent  = 2
	stockMaxOrder    = 1_000_000
	stockDayLayout   = "2006-01-02"
	stockMoneySuffix = "moedas"
)

var errStockShort = errors.New("ações insuficientes")

type stockCompany struct {
	ticker     string
	key        string
	name       string
	emoji      string
	baseCents  int64
	volatility float64
}

var stockCompanies = []stockCompany{
	{"FOFC3", "fofoca", "Fofoca S.A.", "🗣️", 4200, 0.04},
	{"PIXE3", "pix", "Pix Errado Pagamentos", "💸", 1550, 0.03},
	{"FIGU3", "figurinha", "Figurinhas do Zap", "🖼️", 870, 0.06},
	{"BOTE3", "bot", "Bot do Grupo Holding", "🤖", 12000, 0.015},
	{"CAFE4", "cafe", "Café Requentado da Firma", "☕", 2300, 0.025},
	{"KKKK3", "kkk", "Kkkkk Entretenimento", "😂", 640, 0.05},
	{"AGIO3", "agiota", "Agiota do Bot Financeira", "🦈", 5500, 0.035},
	{"DOGE3", "cripto", "Cripto do Primo", "🐕", 300, 0.12},
}

type stockQuote struct {
	company stockCompany
	cents   int64
	open    int64
}

type stockPosition struct {
	ticker string
	shares int64
	cost   int64
}

type stockTrade struct {
	shares int64
	cents  int64
	gross  int64
	fee    int64
	bonus  int64
	cost   int64
}

type stockRow struct {
	cents   int64
	open    int64
	day     string
	updated int64
}

func findStockCompany(query string) (stockCompany, bool) {
	q := normalizeItemName(query)
	if q == "" {
		return stockCompany{}, false
	}
	keys := func(c stockCompany) []string {
		return []string{normalizeItemName(c.ticker), c.key, normalizeItemName(c.name)}
	}
	i := slices.IndexFunc(stockCompanies, func(c stockCompany) bool { return slices.Contains(keys(c), q) })
	if i >= 0 {
		return stockCompanies[i], true
	}
	prefixed := slices.DeleteFunc(slices.Clone(stockCompanies), func(c stockCompany) bool {
		return !slices.ContainsFunc(keys(c), func(k string) bool { return strings.HasPrefix(k, q) })
	})
	if len(prefixed) != 1 {
		return stockCompany{}, false
	}
	return prefixed[0], true
}

func (c stockCompany) step(cents int64) int64 {
	logPrice := math.Log(float64(cents))
	logPrice += stockReversion*(math.Log(float64(c.baseCents))-logPrice) + c.volatility*rand.NormFloat64()
	return max(int64(math.Round(math.Exp(logPrice))), stockFloorCents)
}

func stockDay(t time.Time) string {
	return t.In(lotteryZone).Format(stockDayLayout)
}

func (c stockCompany) advance(r stockRow, now time.Time) stockRow {
	steps := (now.Unix() - r.updated) / int64(stockStepEvery.Seconds())
	if steps <= 0 {
		return r
	}
	start := time.Unix(r.updated, 0).Add(time.Duration(max(steps-stockMaxSteps, 0)) * stockStepEvery)
	steps = min(steps, stockMaxSteps)
	for i := range steps {
		at := start.Add(time.Duration(i+1) * stockStepEvery)
		if day := stockDay(at); day != r.day {
			r.open, r.day = r.cents, day
		}
		r.cents = c.step(r.cents)
	}
	r.updated = start.Add(time.Duration(steps) * stockStepEvery).Unix()
	return r
}

func stockCents(cents int64) string {
	return fmt.Sprintf("🪙 %s,%02d", thousands(cents/100), cents%100)
}

func stockChange(delta, base int64) string {
	pct := 0.0
	if base > 0 {
		pct = float64(delta) * 100 / float64(base)
	}
	arrow := "📈"
	if delta < 0 {
		arrow = "📉"
	}
	return arrow + " " + strings.Replace(fmt.Sprintf("%+.1f%%", pct), ".", ",", 1)
}

func stockFee(gross int64, job profession) int64 {
	if job.freeBrokerage || gross == 0 {
		return 0
	}
	return max(gross*stockFeePercent/100, 1)
}

func (t stockTrade) withProfitBonus(soldCost int64, job profession) stockTrade {
	profit := t.cost - soldCost
	if profit <= 0 || job.stockProfit <= 0 {
		return t
	}
	t.bonus = profit * job.stockProfit / 100
	t.cost += t.bonus
	return t
}

func stockBuyTrade(shares, cents int64, job profession) stockTrade {
	gross := (shares*cents + 99) / 100
	fee := stockFee(gross, job)
	return stockTrade{shares: shares, cents: cents, gross: gross, fee: fee, cost: gross + fee}
}

func stockSellTrade(shares, cents int64, job profession) stockTrade {
	gross := shares * cents / 100
	fee := stockFee(gross, job)
	return stockTrade{shares: shares, cents: cents, gross: gross, fee: fee, cost: gross - fee}
}

func affordableShares(budget, cents int64, job profession) int64 {
	shares := min(budget*100/cents, stockMaxOrder)
	for shares > 0 && stockBuyTrade(shares, cents, job).cost > budget {
		shares--
	}
	return shares
}

func (b *bot) acoes(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.account(ctx, person)
	args := strings.Fields(strings.ToLower(messageText(evt.Message)))[1:]

	text, err := b.stockReply(ctx, chat, acc, args)
	if err != nil {
		return err
	}
	_, err = b.sendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) stockReply(ctx context.Context, chat types.JID, acc account, args []string) (string, error) {
	if len(args) == 0 {
		return b.stockOverview(ctx, chat, acc)
	}
	actions := map[string]func(context.Context, types.JID, account, []string) (string, error){
		"comprar": b.buyStock,
		"compra":  b.buyStock,
		"vender":  b.sellStock,
		"venda":   b.sellStock,
	}
	action, ok := actions[args[0]]
	if !ok {
		return b.stockOverview(ctx, chat, acc)
	}
	return action(ctx, chat, acc, args[1:])
}

func (b *bot) stockOverview(ctx context.Context, chat types.JID, acc account) (string, error) {
	var (
		quotes    []stockQuote
		positions []stockPosition
		job       profession
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { quotes, err = b.wallets.stockMarket(gctx, chat, time.Now()); return err })
	g.Go(func() (err error) { positions, err = b.wallets.stockPositions(gctx, chat, acc); return err })
	g.Go(func() (err error) { job, err = b.professionOf(gctx, chat, acc); return err })
	if err := g.Wait(); err != nil {
		return "", fmt.Errorf("ler bolsa: %w", err)
	}

	lines := []string{fmt.Sprintf("📊 *Bolsa do grupo* (os preços mexem a cada %s)", formatWait(stockStepEvery)), ""}
	prices := make(map[string]stockQuote, len(quotes))
	for _, q := range quotes {
		prices[q.company.ticker] = q
		lines = append(lines, fmt.Sprintf("%s *%s* %s: %s %s hoje",
			q.company.emoji, q.company.ticker, q.company.name, stockCents(q.cents), stockChange(q.cents-q.open, q.open)))
	}
	lines = append(lines, "", stockPortfolioText(acc.jid.User, positions, prices))
	fee := fmt.Sprintf("Corretagem de %d%% em cada ordem", stockFeePercent)
	if job.freeBrokerage {
		fee = "Você não paga corretagem porque " + job.title
	}
	lines = append(lines, "",
		"_!acoes comprar <empresa> <quantidade> (ou 500 moedas, metade, tudo)_",
		"_!acoes vender <empresa> <quantidade> (ou metade, tudo)_",
		"_Dá pra chamar a empresa pelo código ou pelo apelido, tipo fofoca ou cripto. "+fee+"_")
	return strings.Join(lines, "\n"), nil
}

func stockPortfolioText(user string, positions []stockPosition, prices map[string]stockQuote) string {
	if len(positions) == 0 {
		return fmt.Sprintf("💼 @%s não tem nenhuma ação ainda", user)
	}
	lines := []string{fmt.Sprintf("💼 *Carteira de @%s*", user)}
	var totalValue, totalCost int64
	for _, p := range positions {
		q, ok := prices[p.ticker]
		if !ok {
			continue
		}
		value := p.shares * q.cents / 100
		totalValue += value
		totalCost += p.cost
		lines = append(lines, fmt.Sprintf("%s %s: %s %s, valem %s (%s)",
			q.company.emoji, p.ticker, thousands(p.shares), pluralWord(p.shares, "ação", "ações"), coins(value), stockChange(value-p.cost, p.cost)))
	}
	result := "lucro"
	if totalValue < totalCost {
		result = "prejuízo"
	}
	lines = append(lines, fmt.Sprintf("Pagou %s e hoje vale %s: %s de %s", coins(totalCost), coins(totalValue), result, coins(max(totalValue-totalCost, totalCost-totalValue))))
	return strings.Join(lines, "\n")
}

func (b *bot) stockOrderSetup(ctx context.Context, chat types.JID, acc account, query string) (stockQuote, profession, string, error) {
	company, ok := findStockCompany(query)
	if !ok {
		return stockQuote{}, profession{}, "Não conheço essa empresa 🤔 Manda !acoes pra ver a bolsa", nil
	}
	var (
		quotes []stockQuote
		job    profession
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { quotes, err = b.wallets.stockMarket(gctx, chat, time.Now()); return err })
	g.Go(func() (err error) { job, err = b.professionOf(gctx, chat, acc); return err })
	if err := g.Wait(); err != nil {
		return stockQuote{}, profession{}, "", fmt.Errorf("ler bolsa: %w", err)
	}
	i := slices.IndexFunc(quotes, func(q stockQuote) bool { return q.company.ticker == company.ticker })
	return quotes[i], job, "", nil
}

func (b *bot) buyStock(ctx context.Context, chat types.JID, acc account, args []string) (string, error) {
	usage := "Use assim: !acoes comprar fofoca 10 (ações), !acoes comprar fofoca 500 moedas, ou metade/tudo do saldo"
	if len(args) < 2 {
		return usage, nil
	}
	q, job, refusal, err := b.stockOrderSetup(ctx, chat, acc, args[0])
	if err != nil || refusal != "" {
		return refusal, err
	}
	balance, _, err := b.wallets.balance(ctx, chat, acc)
	if err != nil {
		return "", fmt.Errorf("ler saldo: %w", err)
	}
	shares, ok := stockBuyAmount(args[1:], balance, q.cents, job)
	if !ok {
		return usage, nil
	}
	if shares == 0 {
		return fmt.Sprintf("Com %s não dá nem pra uma ação da %s 🪹", coins(balance), q.company.ticker), nil
	}

	t, err := b.wallets.buyStock(ctx, chat, acc, q.company, shares, job, time.Now())
	if errors.Is(err, errNoFunds) {
		return fmt.Sprintf("💸 %s %s da %s custam %s com a corretagem, e você só tem %s",
			thousands(shares), pluralWord(shares, "ação", "ações"), q.company.ticker, coins(stockBuyTrade(shares, q.cents, job).cost), coins(balance)), nil
	}
	if err != nil {
		return "", fmt.Errorf("comprar ações: %w", err)
	}
	return fmt.Sprintf("🛒 @%s comprou %s %s da %s %s a %s cada\n💸 Pagou %s%s",
		acc.jid.User, thousands(t.shares), pluralWord(t.shares, "ação", "ações"), q.company.emoji, q.company.ticker, stockCents(t.cents),
		coins(t.cost), stockFeeNote(t.fee)), nil
}

func stockBuyAmount(args []string, balance, cents int64, job profession) (int64, bool) {
	arg := args[0]
	inCoins := len(args) > 1 && strings.HasPrefix(args[1], "moeda")
	if trimmed, ok := strings.CutSuffix(arg, stockMoneySuffix); ok && trimmed != "" {
		arg, inCoins = trimmed, true
	}
	if slices.Contains([]string{"tudo", "all", "metade"}, arg) || strings.HasSuffix(arg, "%") {
		inCoins = true
	}
	if inCoins {
		budget, ok := parseStake(arg, balance)
		return affordableShares(budget, cents, job), ok
	}
	shares, err := strconv.ParseInt(strings.ReplaceAll(arg, ".", ""), 10, 64)
	if err != nil || shares <= 0 {
		return 0, false
	}
	return min(shares, stockMaxOrder), true
}

func (b *bot) sellStock(ctx context.Context, chat types.JID, acc account, args []string) (string, error) {
	usage := "Use assim: !acoes vender fofoca 10 (ou metade, tudo)"
	if len(args) < 1 {
		return usage, nil
	}
	q, job, refusal, err := b.stockOrderSetup(ctx, chat, acc, args[0])
	if err != nil || refusal != "" {
		return refusal, err
	}
	arg := "tudo"
	if len(args) > 1 {
		arg = args[1]
	}

	t, sold, err := b.wallets.sellStock(ctx, chat, acc, q.company, arg, job, time.Now())
	if errors.Is(err, errStockShort) {
		return fmt.Sprintf("Você não tem tanta ação da %s assim 🤨 Confere no !acoes", q.company.ticker), nil
	}
	if err != nil {
		return "", fmt.Errorf("vender ações: %w", err)
	}
	if t.shares == 0 {
		return usage, nil
	}
	return fmt.Sprintf("💰 @%s vendeu %s %s da %s %s a %s cada\n💸 Recebeu %s%s\n%s",
		acc.jid.User, thousands(t.shares), pluralWord(t.shares, "ação", "ações"), q.company.emoji, q.company.ticker, stockCents(t.cents),
		coins(t.cost), stockFeeNote(t.fee)+stockBonusNote(t.bonus), stockResultLine(t.cost-sold)), nil
}

func stockFeeNote(fee int64) string {
	if fee == 0 {
		return ""
	}
	return fmt.Sprintf(" (%s de corretagem)", coins(fee))
}

func stockBonusNote(bonus int64) string {
	if bonus == 0 {
		return ""
	}
	return fmt.Sprintf(" (com %s de lucro extra)", coins(bonus))
}

func stockResultLine(profit int64) string {
	if profit > 0 {
		return "🤑 Lucro de " + coins(profit) + " na operação"
	}
	if profit < 0 {
		return "🤡 Prejuízo de " + coins(-profit) + " na operação"
	}
	return "😐 Saiu elas por elas"
}

func (s *walletStore) stockMarket(ctx context.Context, chat types.JID, now time.Time) (quotes []stockQuote, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		quotes, err = refreshStocks(ctx, tx, chat, now)
		return err
	})
	return quotes, err
}

func refreshStocks(ctx context.Context, tx *sql.Tx, chat types.JID, now time.Time) ([]stockQuote, error) {
	stored, err := readStockRows(ctx, tx, chat)
	if err != nil {
		return nil, err
	}
	quotes := make([]stockQuote, len(stockCompanies))
	for i, c := range stockCompanies {
		row, found := stored[c.ticker]
		if !found {
			row = stockRow{cents: c.baseCents, open: c.baseCents, day: stockDay(now), updated: now.Unix()}
		}
		next := c.advance(row, now)
		if err := saveStockRow(ctx, tx, chat, c.ticker, row, next, found); err != nil {
			return nil, err
		}
		quotes[i] = stockQuote{company: c, cents: next.cents, open: next.open}
	}
	return quotes, nil
}

func readStockRows(ctx context.Context, tx *sql.Tx, chat types.JID) (stored map[string]stockRow, err error) {
	rows, err := tx.QueryContext(ctx, "SELECT empresa, centavos, abertura, dia, atualizado_em FROM acoes_precos WHERE chat = ?", chat.String())
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	stored = map[string]stockRow{}
	for rows.Next() {
		var (
			ticker string
			r      stockRow
		)
		if err := rows.Scan(&ticker, &r.cents, &r.open, &r.day, &r.updated); err != nil {
			return nil, err
		}
		stored[ticker] = r
	}
	return stored, rows.Err()
}

func saveStockRow(ctx context.Context, tx *sql.Tx, chat types.JID, ticker string, old, next stockRow, found bool) error {
	if !found {
		_, err := tx.ExecContext(ctx,
			"INSERT INTO acoes_precos (chat, empresa, centavos, abertura, dia, atualizado_em) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT DO NOTHING",
			chat.String(), ticker, next.cents, next.open, next.day, next.updated,
		)
		return err
	}
	if next == old {
		return nil
	}
	_, err := tx.ExecContext(ctx,
		"UPDATE acoes_precos SET centavos = ?, abertura = ?, dia = ?, atualizado_em = ? WHERE chat = ? AND empresa = ? AND atualizado_em = ?",
		next.cents, next.open, next.day, next.updated, chat.String(), ticker, old.updated,
	)
	return err
}

func (s *walletStore) stockPositions(ctx context.Context, chat types.JID, acc account) (positions []stockPosition, err error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT empresa, quantidade, custo FROM acoes_posicoes WHERE chat = ? AND pessoa = ? ORDER BY empresa",
		chat.String(), acc.key,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var p stockPosition
		if err := rows.Scan(&p.ticker, &p.shares, &p.cost); err != nil {
			return nil, err
		}
		positions = append(positions, p)
	}
	return positions, rows.Err()
}

func quoteOf(ctx context.Context, tx *sql.Tx, chat types.JID, company stockCompany, now time.Time) (stockQuote, error) {
	quotes, err := refreshStocks(ctx, tx, chat, now)
	if err != nil {
		return stockQuote{}, err
	}
	i := slices.IndexFunc(quotes, func(q stockQuote) bool { return q.company.ticker == company.ticker })
	return quotes[i], nil
}

func (s *walletStore) buyStock(ctx context.Context, chat types.JID, acc account, company stockCompany, shares int64, job profession, now time.Time) (t stockTrade, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		q, err := quoteOf(ctx, tx, chat, company, now)
		if err != nil {
			return err
		}
		t = stockBuyTrade(shares, q.cents, job)
		if err := debitBalance(ctx, tx, chat, acc, t.cost); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx,
			`INSERT INTO acoes_posicoes (chat, pessoa, empresa, quantidade, custo) VALUES (?, ?, ?, ?, ?)
			 ON CONFLICT (chat, pessoa, empresa) DO UPDATE SET quantidade = quantidade + excluded.quantidade, custo = custo + excluded.custo`,
			chat.String(), acc.key, company.ticker, t.shares, t.cost,
		)
		return err
	})
	return t, err
}

func (s *walletStore) sellStock(ctx context.Context, chat types.JID, acc account, company stockCompany, arg string, job profession, now time.Time) (t stockTrade, soldCost int64, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		var held, cost int64
		err := tx.QueryRowContext(ctx,
			"SELECT quantidade, custo FROM acoes_posicoes WHERE chat = ? AND pessoa = ? AND empresa = ?",
			chat.String(), acc.key, company.ticker,
		).Scan(&held, &cost)
		if errors.Is(err, sql.ErrNoRows) {
			return errStockShort
		}
		if err != nil {
			return err
		}
		shares, ok := parseStake(arg, held)
		if !ok {
			return nil
		}
		if shares > held {
			return errStockShort
		}
		q, err := quoteOf(ctx, tx, chat, company, now)
		if err != nil {
			return err
		}
		t = stockSellTrade(shares, q.cents, job)
		soldCost = cost * shares / held
		t = t.withProfitBonus(soldCost, job)
		if err := storeStockPosition(ctx, tx, chat, acc, company.ticker, held-shares, cost-soldCost); err != nil {
			return err
		}
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", t.cost, chat.String(), acc.key)
		return err
	})
	return t, soldCost, err
}

func storeStockPosition(ctx context.Context, tx *sql.Tx, chat types.JID, acc account, ticker string, shares, cost int64) error {
	if shares == 0 {
		_, err := tx.ExecContext(ctx, "DELETE FROM acoes_posicoes WHERE chat = ? AND pessoa = ? AND empresa = ?", chat.String(), acc.key, ticker)
		return err
	}
	_, err := tx.ExecContext(ctx,
		"UPDATE acoes_posicoes SET quantidade = ?, custo = ? WHERE chat = ? AND pessoa = ? AND empresa = ?",
		shares, cost, chat.String(), acc.key, ticker,
	)
	return err
}

func pluralWord(n int64, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
