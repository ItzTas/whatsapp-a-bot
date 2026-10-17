package economy

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/bits"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const betPoolMaxOptions = 10

var (
	errNoBetPool     = errors.New("nenhum bolão aberto")
	errBetPoolOpen   = errors.New("já tem bolão aberto")
	errBetPoolClosed = errors.New("apostas do bolão fechadas")
	errBetPoolOption = errors.New("opção do bolão não existe")
)

type betPoolQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type betPoolOption struct {
	number int64
	text   string
	total  int64
	mine   int64
}

type betPool struct {
	creator  core.Account
	question string
	closed   bool
	options  []betPoolOption
}

type betPoolBet struct {
	owner  core.Account
	option int64
	amount int64
}

type betPoolPayout struct {
	owner  core.Account
	stake  int64
	payout int64
}

type betPoolResult struct {
	pool     betPool
	winner   int64
	payouts  []betPoolPayout
	refunded bool
}

const betPoolMissing = "Não tem bolão aberto aqui 🎯 Abre um com !bolao criar <pergunta> | <opção 1> | <opção 2>"

func (b *bot) bolao(ctx context.Context, evt *events.Message) error {
	_, args := cutPoolWord(core.MessageText(evt.Message))
	action, rest := cutPoolWord(args)
	actions := map[string]func() error{
		"criar":     func() error { return b.createBetPool(ctx, evt, rest) },
		"abrir":     func() error { return b.createBetPool(ctx, evt, rest) },
		"fechar":    func() error { return b.closeBetPool(ctx, evt) },
		"resultado": func() error { return b.resolveBetPool(ctx, evt, rest) },
		"cancelar":  func() error { return b.cancelBetPool(ctx, evt) },
	}
	if run, ok := actions[core.FoldLetters(action)]; ok {
		return run()
	}
	if action == "" {
		return b.betPoolStatus(ctx, evt)
	}
	return b.betOnPool(ctx, evt, action, rest)
}

func (b *bot) createBetPool(ctx context.Context, evt *events.Message, rest string) error {
	chat := evt.Info.Chat.ToNonAD()
	parts := strings.Split(rest, "|")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	parts = slices.DeleteFunc(parts, func(part string) bool { return part == "" })
	if len(parts) < 3 {
		return b.SendText(ctx, chat, "Use assim: !bolao criar <pergunta> | <opção 1> | <opção 2>, tipo !bolao criar Quem ganha o clássico? | Flamengo | Empate | Vasco")
	}
	if len(parts)-1 > betPoolMaxOptions {
		return b.SendText(ctx, chat, fmt.Sprintf("O bolão pode ter no máximo %d opções", betPoolMaxOptions))
	}
	creator := b.Account(ctx, evt.Info.Sender)
	err := b.Wallets.openBetPool(ctx, chat, creator, parts[0], parts[1:], time.Now())
	if errors.Is(err, errBetPoolOpen) {
		return b.SendText(ctx, chat, "Já tem um bolão aberto aqui 🎯 Manda !bolao pra ver. Só dá pra abrir outro depois do !bolao resultado ou do !bolao cancelar")
	}
	if err != nil {
		return fmt.Errorf("abrir bolão: %w", err)
	}

	lines := []string{fmt.Sprintf("🎯 *@%s abriu um bolão!*", creator.JID.User), "❓ " + parts[0], ""}
	for i, option := range parts[1:] {
		lines = append(lines, fmt.Sprintf("*%d.* %s", i+1, option))
	}
	lines = append(lines, "", "_Aposta com !bolao <número> <valor>, tipo !bolao 1 100. Quem acertar divide o pote todo, proporcional ao que apostou_")
	_, err = b.SendMentions(ctx, chat, strings.Join(lines, "\n"), []types.JID{creator.JID})
	return err
}

func (b *bot) betOnPool(ctx context.Context, evt *events.Message, action, rest string) error {
	chat := evt.Info.Chat.ToNonAD()
	option, err := strconv.ParseInt(action, 10, 64)
	arg, _ := cutPoolWord(rest)
	if err != nil || arg == "" {
		return b.SendText(ctx, chat, "Use assim: !bolao <número da opção> <valor>, tipo !bolao 1 100. Manda !bolao pra ver as opções")
	}
	acc := b.Account(ctx, evt.Info.Sender)
	stake, refusal, err := b.PlaceStake(ctx, chat, acc, strings.ToLower(arg), fmt.Sprintf("!bolao %d", option))
	if err != nil {
		return err
	}
	if refusal != "" {
		return b.SendText(ctx, chat, refusal)
	}

	pool, balance, err := b.Wallets.betOnPool(ctx, chat, acc, option, stake)
	refusals := map[error]string{
		errNoBetPool:     betPoolMissing,
		errBetPoolClosed: "As apostas desse bolão já fecharam 🔒 Agora é esperar o resultado",
		errBetPoolOption: "Essa opção não existe. Manda !bolao pra ver as opções",
		ErrNoFunds:       StakeChanged,
	}
	for reason, text := range refusals {
		if errors.Is(err, reason) {
			return b.SendText(ctx, chat, text)
		}
	}
	if err != nil {
		return fmt.Errorf("apostar no bolão: %w", err)
	}
	chosen := pool.options[option-1]
	lines := []string{fmt.Sprintf("🎯 @%s apostou %s em *%s*", acc.JID.User, Coins(stake), chosen.text), ""}
	lines = append(lines, pool.optionLines()...)
	lines = append(lines, "",
		"💰 Pote: "+Coins(pool.pot()),
		fmt.Sprintf("🎫 Você tem %s nessa opção. Saldo: %s", Coins(chosen.mine), Coins(balance)))
	_, err = b.SendMentions(ctx, chat, strings.Join(lines, "\n"), []types.JID{acc.JID})
	return err
}

func (b *bot) betPoolStatus(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	var (
		pool      betPool
		found     bool
		nicknames map[string]string
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) {
		pool, found, err = b.Wallets.betPool(gctx, chat, b.Account(gctx, evt.Info.Sender))
		return err
	})
	g.Go(func() error { nicknames = b.ChatNicknames(gctx, chat); return nil })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler bolão: %w", err)
	}
	if !found {
		return b.SendText(ctx, chat, betPoolMissing)
	}

	lines := []string{"🎯 *Bolão:* " + pool.question, "👤 Aberto por " + b.DisplayName(ctx, pool.creator.JID, nicknames)}
	footer := "_Aposta com !bolao <número> <valor>. Quem acertar divide o pote todo, proporcional ao que apostou_"
	if pool.closed {
		lines = append(lines, "🔒 Apostas fechadas, agora é esperar o resultado")
		footer = "_Quem abriu, admin do grupo ou dono do bot manda !bolao resultado <número>_"
	}
	lines = append(lines, "")
	lines = append(lines, pool.optionLines()...)
	lines = append(lines, "", "💰 Pote: "+Coins(pool.pot()), pool.mineLine(), "", footer)
	return b.ReplyTo(ctx, evt, strings.Join(lines, "\n"))
}

func (b *bot) closeBetPool(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	pool, refusal, err := b.managedBetPool(ctx, evt, "fechar as apostas")
	if err != nil {
		return err
	}
	if refusal != "" {
		return b.SendText(ctx, chat, refusal)
	}
	if pool.closed {
		return b.SendText(ctx, chat, "As apostas desse bolão já tão fechadas 🔒 Agora é só mandar !bolao resultado <número>")
	}
	if err := b.Wallets.closeBetPool(ctx, chat); err != nil {
		return fmt.Errorf("fechar bolão: %w", err)
	}
	lines := []string{"🔒 *Apostas do bolão fechadas!*", "❓ " + pool.question, ""}
	lines = append(lines, pool.optionLines()...)
	lines = append(lines, "", "💰 Pote: "+Coins(pool.pot()), "", "_Quando sair o resultado, manda !bolao resultado <número>_")
	return b.SendText(ctx, chat, strings.Join(lines, "\n"))
}

func (b *bot) resolveBetPool(ctx context.Context, evt *events.Message, rest string) error {
	chat := evt.Info.Chat.ToNonAD()
	arg, _ := cutPoolWord(rest)
	winner, err := strconv.ParseInt(arg, 10, 64)
	if err != nil {
		return b.SendText(ctx, chat, "Diz qual opção ganhou: !bolao resultado <número>. Manda !bolao pra ver as opções")
	}
	pool, refusal, err := b.managedBetPool(ctx, evt, "dar o resultado")
	if err != nil {
		return err
	}
	if refusal != "" {
		return b.SendText(ctx, chat, refusal)
	}
	if winner < 1 || winner > int64(len(pool.options)) {
		return b.SendText(ctx, chat, "Essa opção não existe. Manda !bolao pra ver as opções")
	}
	return b.settleBetPool(ctx, chat, winner)
}

func (b *bot) cancelBetPool(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	_, refusal, err := b.managedBetPool(ctx, evt, "cancelar o bolão")
	if err != nil {
		return err
	}
	if refusal != "" {
		return b.SendText(ctx, chat, refusal)
	}
	return b.settleBetPool(ctx, chat, 0)
}

func (b *bot) settleBetPool(ctx context.Context, chat types.JID, winner int64) error {
	result, err := b.Wallets.settleBetPool(ctx, chat, winner)
	if errors.Is(err, errNoBetPool) {
		return b.SendText(ctx, chat, betPoolMissing)
	}
	if err != nil {
		return fmt.Errorf("pagar bolão: %w", err)
	}
	text, mentions := result.text()
	if _, err := b.SendMentions(ctx, chat, text, mentions); err != nil {
		return fmt.Errorf("anunciar o bolão (as moedas já foram pagas): %w", err)
	}
	return nil
}

func (b *bot) managedBetPool(ctx context.Context, evt *events.Message, verb string) (betPool, string, error) {
	pool, found, err := b.Wallets.betPool(ctx, evt.Info.Chat.ToNonAD(), core.Account{})
	if err != nil {
		return betPool{}, "", fmt.Errorf("ler bolão: %w", err)
	}
	if !found {
		return betPool{}, betPoolMissing, nil
	}
	allowed, err := b.managesBetPool(ctx, evt, pool.creator)
	if err != nil {
		return betPool{}, "", err
	}
	if !allowed {
		return betPool{}, fmt.Sprintf("Só quem abriu o bolão, admin do grupo ou dono do bot pode %s 🚫", verb), nil
	}
	return pool, "", nil
}

func (b *bot) managesBetPool(ctx context.Context, evt *events.Message, creator core.Account) (bool, error) {
	if b.Account(ctx, evt.Info.Sender).Key == creator.Key || b.IsOwner(ctx, evt) {
		return true, nil
	}
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return false, nil
	}
	group, err := b.Client.GetGroupInfo(ctx, chat)
	if err != nil {
		return false, fmt.Errorf("ler o grupo: %w", err)
	}
	sender, found := core.FindParticipant(group.Participants, core.UsersOf(b.SenderIdentities(ctx, evt)))
	return found && core.IsGroupAdmin(sender), nil
}

func (p betPool) pot() int64 {
	var pot int64
	for _, o := range p.options {
		pot += o.total
	}
	return pot
}

func (p betPool) optionLines() []string {
	pot := p.pot()
	lines := make([]string, len(p.options))
	for i, o := range p.options {
		lines[i] = fmt.Sprintf("*%d.* %s: %s", o.number, o.text, o.odds(pot))
	}
	return lines
}

func (o betPoolOption) odds(pot int64) string {
	if o.total == 0 {
		return "ninguém apostou ainda"
	}
	return fmt.Sprintf("%s · paga %s", Coins(o.total), crashX(int64(float64(pot)*100/float64(o.total))))
}

func (p betPool) mineLine() string {
	var bets []string
	for _, o := range p.options {
		if o.mine > 0 {
			bets = append(bets, fmt.Sprintf("%s na %d", Coins(o.mine), o.number))
		}
	}
	if len(bets) == 0 {
		return "🎫 Você ainda não apostou"
	}
	return "🎫 Você apostou " + core.JoinWithAnd(bets)
}

func (r betPoolResult) text() (string, []types.JID) {
	mentions := make([]types.JID, len(r.payouts))
	for i, p := range r.payouts {
		mentions[i] = p.owner.JID
	}
	lines := []string{r.headline(), "❓ " + r.pool.question}
	if r.winner > 0 {
		lines = append(lines, "✅ Deu *"+r.pool.options[r.winner-1].text+"*")
	}
	lines = append(lines, "", r.summary())
	for _, p := range r.payouts {
		lines = append(lines, r.payoutLine(p))
	}
	return strings.Join(lines, "\n"), mentions
}

func (r betPoolResult) headline() string {
	if r.winner == 0 {
		return "🚫 *Bolão cancelado*"
	}
	return "🏁 *Saiu o resultado do bolão!*"
}

func (r betPoolResult) summary() string {
	if len(r.payouts) == 0 {
		return "🪹 Ninguém tinha apostado nada"
	}
	if r.winner == 0 {
		return "↩️ Todo mundo recebeu de volta o que apostou:"
	}
	if r.refunded {
		return "🤷 Ninguém apostou nisso, então todo mundo recebeu de volta o que apostou:"
	}
	return fmt.Sprintf("💰 O pote de %s foi dividido entre quem acertou:", Coins(r.pool.pot()))
}

func (r betPoolResult) payoutLine(p betPoolPayout) string {
	if r.refunded {
		return fmt.Sprintf("↩️ @%s: %s", p.owner.JID.User, Coins(p.payout))
	}
	return fmt.Sprintf("🏆 @%s apostou %s e levou %s", p.owner.JID.User, Coins(p.stake), Coins(p.payout))
}

func betPoolPayouts(bets []betPoolBet, winner int64) ([]betPoolPayout, bool) {
	var pot, winning int64
	for _, bet := range bets {
		pot += bet.amount
		if bet.option == winner {
			winning += bet.amount
		}
	}
	if winning == 0 {
		return betPoolRefunds(bets), true
	}
	var (
		payouts []betPoolPayout
		paid    int64
	)
	for _, bet := range bets {
		if bet.option != winner {
			continue
		}
		share := betPoolShare(pot, bet.amount, winning)
		payouts = append(payouts, betPoolPayout{owner: bet.owner, stake: bet.amount, payout: share})
		paid += share
	}
	slices.SortFunc(payouts, compareBetPoolPayouts)
	payouts[0].payout += pot - paid
	return payouts, false
}

func betPoolRefunds(bets []betPoolBet) []betPoolPayout {
	index := map[string]int{}
	var refunds []betPoolPayout
	for _, bet := range bets {
		i, seen := index[bet.owner.Key]
		if !seen {
			i = len(refunds)
			index[bet.owner.Key] = i
			refunds = append(refunds, betPoolPayout{owner: bet.owner})
		}
		refunds[i].stake += bet.amount
		refunds[i].payout += bet.amount
	}
	slices.SortFunc(refunds, compareBetPoolPayouts)
	return refunds
}

func compareBetPoolPayouts(a, b betPoolPayout) int {
	return cmp.Compare(b.stake, a.stake)
}

func betPoolShare(pot, stake, winning int64) int64 {
	hi, lo := bits.Mul64(uint64(pot), uint64(stake))
	share, _ := bits.Div64(hi, lo, uint64(winning))
	return int64(share)
}

func cutPoolWord(s string) (string, string) {
	s = strings.TrimSpace(s)
	i := strings.IndexFunc(s, unicode.IsSpace)
	if i < 0 {
		return s, ""
	}
	return s[:i], strings.TrimSpace(s[i:])
}

func (s *walletStore) openBetPool(ctx context.Context, chat types.JID, creator core.Account, question string, options []string, now time.Time) error {
	return s.inTx(ctx, func(tx *ledgerTx) error {
		res, err := tx.ExecContext(ctx,
			"INSERT INTO boloes (chat, criador, criador_jid, pergunta, criado_em) VALUES (?, ?, ?, ?, ?) ON CONFLICT DO NOTHING",
			chat.String(), creator.Key, creator.JID.String(), question, now.Unix(),
		)
		if err != nil {
			return err
		}
		inserted, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if inserted == 0 {
			return errBetPoolOpen
		}
		for i, option := range options {
			_, err := tx.ExecContext(ctx, "INSERT INTO bolao_opcoes (chat, numero, texto) VALUES (?, ?, ?)", chat.String(), i+1, option)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *walletStore) betPool(ctx context.Context, chat types.JID, acc core.Account) (betPool, bool, error) {
	return loadBetPool(ctx, s.db, chat, acc)
}

func (s *walletStore) betOnPool(ctx context.Context, chat types.JID, acc core.Account, option, amount int64) (p betPool, balance int64, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		pool, found, err := loadBetPool(ctx, tx, chat, acc)
		if err != nil {
			return err
		}
		if !found {
			return errNoBetPool
		}
		if pool.closed {
			return errBetPoolClosed
		}
		if option < 1 || option > int64(len(pool.options)) {
			return errBetPoolOption
		}
		err = tx.QueryRowContext(ctx,
			"UPDATE carteiras SET saldo = saldo - ? WHERE chat = ? AND pessoa = ? AND saldo >= ? RETURNING saldo",
			amount, chat.String(), acc.Key, amount,
		).Scan(&balance)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoFunds
		}
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx,
			`INSERT INTO bolao_apostas (chat, pessoa, jid, opcao, valor) VALUES (?, ?, ?, ?, ?)
			 ON CONFLICT (chat, pessoa, opcao) DO UPDATE SET valor = bolao_apostas.valor + excluded.valor, jid = excluded.jid`,
			chat.String(), acc.Key, acc.JID.String(), option, amount,
		)
		if err != nil {
			return err
		}
		pool.options[option-1].total += amount
		pool.options[option-1].mine += amount
		p = pool
		return nil
	})
	return p, balance, err
}

func (s *walletStore) closeBetPool(ctx context.Context, chat types.JID) error {
	_, err := s.db.ExecContext(ctx, "UPDATE boloes SET fechado = 1 WHERE chat = ?", chat.String())
	return err
}

func (s *walletStore) settleBetPool(ctx context.Context, chat types.JID, winner int64) (r betPoolResult, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		pool, found, err := loadBetPool(ctx, tx, chat, core.Account{})
		if err != nil {
			return err
		}
		if !found {
			return errNoBetPool
		}
		if winner > int64(len(pool.options)) {
			return errBetPoolOption
		}
		bets, err := takeBetPoolBets(ctx, tx, chat)
		if err != nil {
			return err
		}
		for _, query := range []string{"DELETE FROM bolao_opcoes WHERE chat = ?", "DELETE FROM boloes WHERE chat = ?"} {
			if _, err := tx.ExecContext(ctx, query, chat.String()); err != nil {
				return err
			}
		}
		r = betPoolResult{pool: pool, winner: winner}
		r.payouts, r.refunded = betPoolPayouts(bets, winner)
		for _, p := range r.payouts {
			if err := openWallet(ctx, tx, chat, p.owner); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", p.payout, chat.String(), p.owner.Key)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return r, err
}

func loadBetPool(ctx context.Context, q betPoolQuerier, chat types.JID, acc core.Account) (p betPool, found bool, err error) {
	var creator string
	err = q.QueryRowContext(ctx,
		"SELECT criador, criador_jid, pergunta, fechado FROM boloes WHERE chat = ?",
		chat.String(),
	).Scan(&p.creator.Key, &creator, &p.question, &p.closed)
	if errors.Is(err, sql.ErrNoRows) {
		return betPool{}, false, nil
	}
	if err != nil {
		return betPool{}, false, err
	}
	if p.creator.JID, err = types.ParseJID(creator); err != nil {
		return betPool{}, false, err
	}
	p.options, err = loadBetPoolOptions(ctx, q, chat, acc)
	return p, err == nil, err
}

func loadBetPoolOptions(ctx context.Context, q betPoolQuerier, chat types.JID, acc core.Account) (options []betPoolOption, err error) {
	rows, err := q.QueryContext(ctx,
		`SELECT o.numero, o.texto, COALESCE(SUM(a.valor), 0), COALESCE(SUM(a.valor) FILTER (WHERE a.pessoa = ?), 0)
		 FROM bolao_opcoes o LEFT JOIN bolao_apostas a ON a.chat = o.chat AND a.opcao = o.numero
		 WHERE o.chat = ? GROUP BY o.numero, o.texto ORDER BY o.numero`,
		acc.Key, chat.String(),
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var o betPoolOption
		if err := rows.Scan(&o.number, &o.text, &o.total, &o.mine); err != nil {
			return nil, err
		}
		options = append(options, o)
	}
	return options, rows.Err()
}

func takeBetPoolBets(ctx context.Context, tx *ledgerTx, chat types.JID) (bets []betPoolBet, err error) {
	rows, err := tx.QueryContext(ctx, "DELETE FROM bolao_apostas WHERE chat = ? RETURNING pessoa, jid, opcao, valor", chat.String())
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var (
			bet betPoolBet
			jid string
		)
		if err := rows.Scan(&bet.owner.Key, &jid, &bet.option, &bet.amount); err != nil {
			return nil, err
		}
		if bet.owner.JID, err = types.ParseJID(jid); err != nil {
			return nil, err
		}
		bets = append(bets, bet)
	}
	return bets, rows.Err()
}
