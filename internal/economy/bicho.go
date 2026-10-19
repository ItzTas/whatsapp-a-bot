package economy

import (
	"cmp"
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

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	animalNumbers      = 10_000
	animalGroupSize    = 4
	animalBetsPerDraw  = 10
	animalResultsShown = 5
	animalResultsKept  = 30 * 24 * time.Hour
	animalHint         = "_Aposta com !bicho <valor> <bicho> ou !bicho <valor> <número>: 2 dígitos é dezena, 3 é centena e 4 é milhar. Os bichos e quanto cada aposta paga ficam no !bicho tabela_"
	animalUsage        = "Diz o valor e o palpite: !bicho 50 vaca (o bicho), !bicho 20 07 (dezena), !bicho 10 123 (centena) ou !bicho 5 1234 (milhar). Os bichos e as dezenas de cada um ficam no !bicho tabela"
)

var errAnimalBetLimit = errors.New("limite de apostas por extração")

type animal struct {
	name    string
	emoji   string
	in      string
	aliases []string
}

var animals = [...]animal{
	{name: "Avestruz", emoji: "🦤", in: "no"},
	{name: "Águia", emoji: "🦅", in: "na"},
	{name: "Burro", emoji: "🫏", in: "no", aliases: []string{"jumento", "jegue"}},
	{name: "Borboleta", emoji: "🦋", in: "na"},
	{name: "Cachorro", emoji: "🐕", in: "no", aliases: []string{"cao", "cachorra"}},
	{name: "Cabra", emoji: "🐐", in: "na", aliases: []string{"bode"}},
	{name: "Carneiro", emoji: "🐏", in: "no", aliases: []string{"ovelha"}},
	{name: "Camelo", emoji: "🐫", in: "no"},
	{name: "Cobra", emoji: "🐍", in: "na", aliases: []string{"serpente"}},
	{name: "Coelho", emoji: "🐇", in: "no"},
	{name: "Cavalo", emoji: "🐎", in: "no"},
	{name: "Elefante", emoji: "🐘", in: "no"},
	{name: "Galo", emoji: "🐓", in: "no", aliases: []string{"galinha"}},
	{name: "Gato", emoji: "🐈", in: "no"},
	{name: "Jacaré", emoji: "🐊", in: "no"},
	{name: "Leão", emoji: "🦁", in: "no"},
	{name: "Macaco", emoji: "🐒", in: "no"},
	{name: "Porco", emoji: "🐖", in: "no"},
	{name: "Pavão", emoji: "🦚", in: "no"},
	{name: "Peru", emoji: "🦃", in: "no"},
	{name: "Touro", emoji: "🐂", in: "no", aliases: []string{"boi"}},
	{name: "Tigre", emoji: "🐅", in: "no"},
	{name: "Urso", emoji: "🐻", in: "no"},
	{name: "Veado", emoji: "🦌", in: "no", aliases: []string{"cervo"}},
	{name: "Vaca", emoji: "🐄", in: "na"},
}

type animalBetKind struct {
	id     string
	title  string
	in     string
	about  string
	digits int
	odds   int64
	limit  int64
}

var (
	animalGroupBet = animalBetKind{id: "grupo", title: "Grupo", in: "no", about: "acerta o bicho", odds: 18, limit: 50_000}
	animalBetKinds = []animalBetKind{
		animalGroupBet,
		{id: "dezena", title: "Dezena", in: "na", about: "acerta os 2 últimos números", digits: 2, odds: 60, limit: 15_000},
		{id: "centena", title: "Centena", in: "na", about: "acerta os 3 últimos números", digits: 3, odds: 600, limit: 1_500},
		{id: "milhar", title: "Milhar", in: "no", about: "acerta os 4 números", digits: 4, odds: 4_000, limit: 250},
	}
)

type animalDraw struct {
	hour int
	name string
}

var animalDraws = []animalDraw{{11, "PTM"}, {14, "PT"}, {16, "PTV"}, {18, "PTN"}, {21, "Corujinha"}}

type animalBet struct {
	kind  animalBetKind
	guess int
	stake int64
	luck  int64
}

type animalTicket struct {
	owner core.Account
	at    time.Time
	bet   animalBet
	prize int64
}

type animalResult struct {
	at   time.Time
	draw int
}

type animalDrawResult struct {
	animalResult
	tickets []animalTicket
}

func (b *bot) bicho(ctx context.Context, evt *events.Message) error {
	args := strings.Fields(strings.ToLower(core.MessageText(evt.Message)))[1:]
	if len(args) == 0 {
		return b.animalStatus(ctx, evt)
	}
	if slices.Contains([]string{"tabela", "bichos", "lista"}, args[0]) {
		return b.animalTable(ctx, evt)
	}
	return b.betAnimal(ctx, evt, args)
}

func (b *bot) betAnimal(ctx context.Context, evt *events.Message, args []string) error {
	chat := evt.Info.Chat.ToNonAD()
	bet, stakeArg, ok := animalBetArgs(args)
	if !ok {
		return b.SendText(ctx, chat, animalUsage)
	}
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	var (
		whole   int64
		refusal string
		job     Profession
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		rule, err := b.betRuleOf(gctx, chat, acc)
		if err != nil {
			return err
		}
		bet.stake, whole, refusal, err = b.affordable(gctx, chat, acc, stakeArg, "!bicho", rule)
		return err
	})
	g.Go(func() (err error) { job, err = b.ProfessionOf(gctx, chat, acc); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("apostar no bicho: %w", err)
	}
	if refusal != "" {
		return b.SendText(ctx, chat, refusal)
	}
	if bet.stake > bet.kind.limit {
		return b.SendText(ctx, chat, fmt.Sprintf("A aposta máxima %s %s é %s 🎫 Os limites de cada tipo ficam no !bicho tabela", bet.kind.in, bet.kind.id, Coins(bet.kind.limit)))
	}
	if refusal := b.allIn(ctx, chat, acc, bet.stake, whole); refusal != "" {
		return b.SendText(ctx, chat, refusal)
	}

	bet.luck = job.animalLuck
	now := time.Now()
	at := nextAnimalDraw(now)
	balance, err := b.Wallets.placeAnimalBet(ctx, chat, acc, bet, at)
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, chat, stakeChanged)
	}
	if errors.Is(err, errAnimalBetLimit) {
		return b.SendText(ctx, chat, fmt.Sprintf("Você já fez %d apostas pra %s, que é o máximo por extração 🎫", animalBetsPerDraw, animalDrawLabel(at, now)))
	}
	if err != nil {
		return fmt.Errorf("apostar no bicho: %w", err)
	}
	payout := bet.payout()
	text := fmt.Sprintf("🎫 @%s jogou %s %s pra %s, que sai daqui a %s. Se der, leva %s%s\n💰 Saldo: %s",
		person.User, Coins(bet.stake), bet.target(), animalDrawLabel(at, now), core.FormatWait(at.Sub(now)), Coins(payout), PrizeBoost{luck: bet.luck}.Tag(bet.stake, payout), Coins(balance))
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) animalStatus(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	acc := b.Account(ctx, evt.Info.Sender.ToNonAD())
	now := time.Now()
	var (
		tickets []animalTicket
		results []animalResult
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { tickets, err = b.Wallets.animalTicketsOf(gctx, chat, acc); return err })
	g.Go(func() (err error) { results, err = b.Wallets.animalResults(gctx, now); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler o bicho: %w", err)
	}

	next := nextAnimalDraw(now)
	lines := []string{
		"🦁 *Jogo do bicho*",
		"",
		fmt.Sprintf("⏰ Próxima extração: %s, daqui a %s", animalDrawLabel(next, now), core.FormatWait(next.Sub(now))),
	}
	lines = append(lines, animalTicketLines(tickets, now)...)
	lines = append(lines, animalResultLines(results, now)...)
	lines = append(lines, "", animalHint)
	return b.ReplyTo(ctx, evt, strings.Join(lines, "\n"))
}

func animalTicketLines(tickets []animalTicket, now time.Time) []string {
	if len(tickets) == 0 {
		return []string{"🎫 Você não tem aposta esperando resultado"}
	}
	lines := []string{"🎫 Suas apostas:"}
	for _, t := range tickets {
		lines = append(lines, fmt.Sprintf("• %s %s, na %s: se der, leva %s", Coins(t.bet.stake), t.bet.target(), animalDrawLabel(t.at, now), Coins(t.bet.payout())))
	}
	return lines
}

func animalResultLines(results []animalResult, now time.Time) []string {
	if len(results) == 0 {
		return nil
	}
	lines := []string{"", "📜 Últimos resultados:"}
	for _, r := range results {
		lines = append(lines, fmt.Sprintf("• %s: %s", animalDrawLabel(r.at, now), animalNumber(r.draw)))
	}
	return lines
}

func (b *bot) animalTable(ctx context.Context, evt *events.Message) error {
	lines := []string{"🦁 *Tabela do bicho*", ""}
	for i, a := range animals {
		lines = append(lines, fmt.Sprintf("%02d %s %s: %s", i+1, a.emoji, a.name, strings.Join(animalTens(i+1), " ")))
	}
	lines = append(lines, "", "💰 *Quanto paga, se der na cabeça*")
	for _, k := range animalBetKinds {
		lines = append(lines, fmt.Sprintf("• %s (%s): %dx, aposta até %s", k.title, k.about, k.odds, Coins(k.limit)))
	}
	lines = append(lines, "", "⏰ Extrações todo dia: "+animalSchedule(), "", animalHint)
	return b.ReplyTo(ctx, evt, strings.Join(lines, "\n"))
}

func (b *bot) runAnimalGame(ctx context.Context) {
	ticker := time.NewTicker(core.ReminderTick)
	defer ticker.Stop()
	for {
		if b.Client.IsLoggedIn() {
			b.drawAnimalGame(ctx, time.Now())
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (b *bot) drawAnimalGame(ctx context.Context, now time.Time) {
	var (
		due []pendingDraw
		g   errgroup.Group
	)
	g.Go(func() error { return b.Wallets.recordAnimalDraw(ctx, lastAnimalDraw(now), now) })
	g.Go(func() (err error) { due, err = b.Wallets.dueDraws(ctx, "bicho_apostas", now); return err })
	if err := g.Wait(); err != nil {
		b.Client.Log.Errorf("sortear o bicho: %v", err)
		return
	}
	for _, pending := range due {
		if err := b.settleAnimalDraw(ctx, pending, now); err != nil {
			b.Client.Log.Errorf("pagar o bicho de %s: %v", pending.chat, err)
		}
	}
}

func (b *bot) settleAnimalDraw(ctx context.Context, pending pendingDraw, now time.Time) error {
	r, err := b.Wallets.payAnimalDraw(ctx, pending)
	if err != nil || len(r.tickets) == 0 {
		return err
	}
	text, winners := r.announcement(now)
	if err := b.announce(ctx, pending.chat, text, winners...); err != nil {
		return fmt.Errorf("anunciar o bicho (as moedas já foram pagas): %w", err)
	}
	return nil
}

func (r animalDrawResult) announcement(now time.Time) (string, []types.JID) {
	a := animals[animalOf(r.draw)-1]
	lines := []string{
		fmt.Sprintf("%s *Deu %s no bicho!* Resultado da %s", a.emoji, strings.ToLower(a.name), animalDrawLabel(r.at, now)),
		"",
		"🎲 Milhar " + animalNumber(r.draw),
		"",
	}
	var (
		winners []types.JID
		losers  int
		lost    int64
	)
	for _, t := range r.tickets {
		if t.prize == 0 {
			losers++
			lost += t.bet.stake
			continue
		}
		lines = append(lines, fmt.Sprintf("🏆 @%s jogou %s %s e levou %s%s", t.owner.JID.User, Coins(t.bet.stake), t.bet.target(), Coins(t.prize), PrizeBoost{luck: t.bet.luck}.Tag(t.bet.stake, t.prize)))
		if !slices.Contains(winners, t.owner.JID) {
			winners = append(winners, t.owner.JID)
		}
	}
	if losers > 0 {
		lines = append(lines, animalLossLine(losers, lost, len(winners) > 0))
	}
	lines = append(lines, "", fmt.Sprintf("_Próxima extração: %s. Faz sua fezinha com !bicho <valor> <bicho>_", animalDrawLabel(nextAnimalDraw(now), now)))
	return strings.Join(lines, "\n"), winners
}

func animalLossLine(losers int, lost int64, anyWinner bool) string {
	bets := core.CountLabel(losers, "aposta", "apostas")
	if anyWinner {
		return fmt.Sprintf("😬 Sem acerto: %s, e a banca ficou com %s", bets, Coins(lost))
	}
	return fmt.Sprintf("😬 Ninguém acertou, e a banca ficou com %s de %s", Coins(lost), bets)
}

func animalBetArgs(args []string) (animalBet, string, bool) {
	if len(args) < 2 {
		return animalBet{}, "", false
	}
	stake, guess := args[0], args[1]
	if _, named := animalNamed(stake); named {
		stake, guess = guess, stake
	}
	bet, ok := parseAnimalGuess(guess)
	return bet, stake, ok && looksLikeStake(stake)
}

func looksLikeStake(arg string) bool {
	_, ok := ParseStake(arg, math.MaxInt32)
	return ok
}

func parseAnimalGuess(word string) (animalBet, bool) {
	if group, ok := animalNamed(word); ok {
		return animalBet{kind: animalGroupBet, guess: group}, true
	}
	i := slices.IndexFunc(animalBetKinds, func(k animalBetKind) bool { return k.digits == len(word) })
	guess, err := strconv.Atoi(word)
	if i < 0 || err != nil || strings.Trim(word, "0123456789") != "" {
		return animalBet{}, false
	}
	return animalBet{kind: animalBetKinds[i], guess: guess}, true
}

func animalNamed(word string) (int, bool) {
	folded := core.FoldLetters(word)
	i := slices.IndexFunc(animals[:], func(a animal) bool {
		return core.FoldLetters(a.name) == folded || slices.Contains(a.aliases, folded)
	})
	return i + 1, i >= 0
}

func animalBetKindNamed(id string) (animalBetKind, error) {
	i := slices.IndexFunc(animalBetKinds, func(k animalBetKind) bool { return k.id == id })
	if i < 0 {
		return animalBetKind{}, fmt.Errorf("tipo de aposta do bicho desconhecido: %q", id)
	}
	return animalBetKinds[i], nil
}

func (k animalBetKind) byAnimal() bool {
	return k.digits == 0
}

func (bet animalBet) hits(draw int) bool {
	if bet.kind.byAnimal() {
		return animalOf(draw) == bet.guess
	}
	return draw%int(math.Pow10(bet.kind.digits)) == bet.guess
}

func (bet animalBet) payout() int64 {
	return PrizeBoost{luck: bet.luck}.Pay(bet.stake, bet.stake*bet.kind.odds)
}

func (bet animalBet) prize(draw int) int64 {
	if !bet.hits(draw) {
		return 0
	}
	return bet.payout()
}

func (bet animalBet) target() string {
	if bet.kind.byAnimal() {
		a := animals[bet.guess-1]
		return fmt.Sprintf("%s %s %s", a.in, a.emoji, a.name)
	}
	return fmt.Sprintf("%s %s %0*d", bet.kind.in, bet.kind.id, bet.kind.digits, bet.guess)
}

func animalOf(draw int) int {
	tens := draw % 100
	if tens == 0 {
		return len(animals)
	}
	return (tens-1)/animalGroupSize + 1
}

func animalTens(group int) []string {
	tens := make([]string, animalGroupSize)
	for i := range tens {
		tens[i] = fmt.Sprintf("%02d", ((group-1)*animalGroupSize+i+1)%100)
	}
	return tens
}

func animalNumber(draw int) string {
	group := animalOf(draw)
	a := animals[group-1]
	return fmt.Sprintf("%04d · %s %s (grupo %02d)", draw, a.emoji, a.name, group)
}

func animalSchedule() string {
	hours := make([]string, len(animalDraws))
	for i, d := range animalDraws {
		hours[i] = fmt.Sprintf("%dh (%s)", d.hour, d.name)
	}
	return core.JoinWithAnd(hours)
}

func animalNumberOdds() string {
	var odds []string
	for _, k := range animalBetKinds {
		if k.byAnimal() {
			continue
		}
		odds = append(odds, fmt.Sprintf("%d dígitos é %s (%dx)", k.digits, k.id, k.odds))
	}
	return core.JoinWithAnd(odds)
}

func animalDrawOn(now time.Time, days, hour int) time.Time {
	local := now.In(core.BrazilTime)
	return time.Date(local.Year(), local.Month(), local.Day()+days, hour, 0, 0, 0, core.BrazilTime)
}

func nextAnimalDraw(now time.Time) time.Time {
	for _, d := range animalDraws {
		if at := animalDrawOn(now, 0, d.hour); at.After(now) {
			return at
		}
	}
	return animalDrawOn(now, 1, animalDraws[0].hour)
}

func lastAnimalDraw(now time.Time) time.Time {
	for _, d := range slices.Backward(animalDraws) {
		if at := animalDrawOn(now, 0, d.hour); !at.After(now) {
			return at
		}
	}
	return animalDrawOn(now, -1, animalDraws[len(animalDraws)-1].hour)
}

func animalDrawLabel(at, now time.Time) string {
	local := at.In(core.BrazilTime)
	return fmt.Sprintf("%s (%s às %dh)", animalDrawName(local.Hour()), animalDay(local, now), local.Hour())
}

func animalDrawName(hour int) string {
	i := slices.IndexFunc(animalDraws, func(d animalDraw) bool { return d.hour == hour })
	if i < 0 {
		return "Extração"
	}
	return animalDraws[i].name
}

func animalDay(local, now time.Time) string {
	today := now.In(core.BrazilTime)
	days := map[string]string{
		today.AddDate(0, 0, -1).Format(time.DateOnly): "ontem",
		today.Format(time.DateOnly):                   "hoje",
		today.AddDate(0, 0, 1).Format(time.DateOnly):  "amanhã",
	}
	if day, ok := days[local.Format(time.DateOnly)]; ok {
		return day
	}
	return local.Format("02/01")
}

func (s *walletStore) placeAnimalBet(ctx context.Context, chat types.JID, acc core.Account, bet animalBet, at time.Time) (balance int64, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		var placed int
		err := tx.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM bicho_apostas WHERE chat = ? AND pessoa = ? AND sorteio = ?",
			chat.String(), acc.Key, at.Unix(),
		).Scan(&placed)
		if err != nil {
			return err
		}
		if placed >= animalBetsPerDraw {
			return errAnimalBetLimit
		}
		err = tx.QueryRowContext(ctx,
			"UPDATE carteiras SET saldo = saldo - ? WHERE chat = ? AND pessoa = ? AND saldo >= ? RETURNING saldo",
			bet.stake, chat.String(), acc.Key, bet.stake,
		).Scan(&balance)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoFunds
		}
		if err != nil {
			return err
		}
		if err := noteWager(ctx, tx, chat, acc, bet.stake, time.Now()); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx,
			"INSERT INTO bicho_apostas (chat, pessoa, jid, sorteio, tipo, palpite, valor, sorte) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			chat.String(), acc.Key, acc.JID.String(), at.Unix(), bet.kind.id, bet.guess, bet.stake, bet.luck,
		)
		return err
	})
	return balance, err
}

func (s *walletStore) animalTicketsOf(ctx context.Context, chat types.JID, acc core.Account) ([]animalTicket, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT pessoa, jid, sorteio, tipo, palpite, valor, sorte FROM bicho_apostas WHERE chat = ? AND pessoa = ? ORDER BY sorteio, id",
		chat.String(), acc.Key,
	)
	if err != nil {
		return nil, err
	}
	return scanAnimalTickets(rows)
}

func (s *walletStore) animalResults(ctx context.Context, now time.Time) (results []animalResult, err error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT sorteio, milhar FROM bicho_resultados WHERE sorteio <= ? ORDER BY sorteio DESC LIMIT ?",
		now.Unix(), animalResultsShown,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var (
			r  animalResult
			at int64
		)
		if err := rows.Scan(&at, &r.draw); err != nil {
			return nil, err
		}
		r.at = time.Unix(at, 0)
		results = append(results, r)
	}
	return results, rows.Err()
}

func (s *walletStore) recordAnimalDraw(ctx context.Context, at, now time.Time) error {
	draw, err := s.riggedAnimalDraw(ctx, at)
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO bicho_resultados (sorteio, milhar) VALUES (?, ?) ON CONFLICT (sorteio) DO NOTHING",
		at.Unix(), draw,
	)
	if err != nil {
		return err
	}
	drawn, err := res.RowsAffected()
	if err != nil || drawn == 0 {
		return err
	}
	_, err = s.db.ExecContext(ctx, "DELETE FROM bicho_resultados WHERE sorteio < ?", now.Add(-animalResultsKept).Unix())
	return err
}

func (s *walletStore) riggedAnimalDraw(ctx context.Context, at time.Time) (int, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT pessoa, jid, sorteio, tipo, palpite, valor, sorte FROM bicho_apostas WHERE sorteio = ?",
		at.Unix(),
	)
	if err != nil {
		return 0, err
	}
	tickets, err := scanAnimalTickets(rows)
	if err != nil {
		return 0, err
	}
	draw := rand.IntN(animalNumbers)
	for _, t := range tickets {
		if !t.bet.hits(draw) || rand.Float64() >= stakeRisk(t.bet.stake) {
			continue
		}
		for t.bet.hits(draw) {
			draw = rand.IntN(animalNumbers)
		}
	}
	return draw, nil
}

func (s *walletStore) payAnimalDraw(ctx context.Context, pending pendingDraw) (r animalDrawResult, err error) {
	r.at = pending.at
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		draw, err := animalResultOf(ctx, tx, pending.at)
		if err != nil {
			return err
		}
		tickets, err := takeAnimalBets(ctx, tx, pending)
		if err != nil {
			return err
		}
		for i, t := range tickets {
			tickets[i].prize = t.bet.prize(draw)
			if err := creditAnimalPrize(ctx, tx, pending.chat, tickets[i]); err != nil {
				return err
			}
			if err := noteLoss(ctx, tx, pending.chat, t.owner, lossBet, t.bet.stake-tickets[i].prize); err != nil {
				return err
			}
		}
		slices.SortStableFunc(tickets, func(a, b animalTicket) int { return cmp.Compare(b.prize, a.prize) })
		r.draw, r.tickets = draw, tickets
		return nil
	})
	return r, err
}

func creditAnimalPrize(ctx context.Context, tx *ledgerTx, chat types.JID, t animalTicket) error {
	if t.prize == 0 {
		return nil
	}
	if err := openWallet(ctx, tx, chat, t.owner); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", t.prize, chat.String(), t.owner.Key)
	return err
}

func animalResultOf(ctx context.Context, tx *ledgerTx, at time.Time) (draw int, err error) {
	err = tx.QueryRowContext(ctx,
		`INSERT INTO bicho_resultados (sorteio, milhar) VALUES (?, ?)
		 ON CONFLICT (sorteio) DO UPDATE SET milhar = bicho_resultados.milhar
		 RETURNING milhar`,
		at.Unix(), rand.IntN(animalNumbers),
	).Scan(&draw)
	return draw, err
}

func takeAnimalBets(ctx context.Context, tx *ledgerTx, pending pendingDraw) ([]animalTicket, error) {
	rows, err := tx.QueryContext(ctx,
		"DELETE FROM bicho_apostas WHERE chat = ? AND sorteio = ? RETURNING pessoa, jid, sorteio, tipo, palpite, valor, sorte",
		pending.chat.String(), pending.at.Unix(),
	)
	if err != nil {
		return nil, err
	}
	return scanAnimalTickets(rows)
}

func scanAnimalTickets(rows *sql.Rows) (tickets []animalTicket, err error) {
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var (
			t       animalTicket
			jid, id string
			at      int64
		)
		if err := rows.Scan(&t.owner.Key, &jid, &at, &id, &t.bet.guess, &t.bet.stake, &t.bet.luck); err != nil {
			return nil, err
		}
		if t.owner.JID, err = types.ParseJID(jid); err != nil {
			return nil, err
		}
		if t.bet.kind, err = animalBetKindNamed(id); err != nil {
			return nil, err
		}
		t.at = time.Unix(at, 0)
		tickets = append(tickets, t)
	}
	return tickets, rows.Err()
}
