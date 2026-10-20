package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	securitySector = "seguranca"
	contractSpan   = 24 * time.Hour
	emergencyCut   = 15
	offerFirmID    = "escolta-armada"
)

var errNoContract = errors.New("sem contrato")

type firm struct {
	id           string
	sector       string
	name         string
	emoji        string
	annul        float64
	gangs        bool
	cover        int64
	edge         float64
	breakout     int64
	breakoutOdds float64
	rate         int64
	floor        int64
	aliases      []string
}

var firms = []firm{
	{id: "vigia-de-rua", sector: securitySector, name: "Vigia de rua", emoji: "🔦", annul: 0.4, cover: 10, rate: 2, floor: 500, aliases: []string{"vigia"}},
	{id: "escolta-armada", sector: securitySector, name: "Escolta armada", emoji: "🚓", annul: 0.6, gangs: true, cover: 20, rate: 4, floor: 1_500, aliases: []string{"escolta"}},
	{id: "carro-forte", sector: securitySector, name: "Carro-forte", emoji: "🛡️", annul: 0.8, gangs: true, cover: 35, rate: 7, floor: 4_000},
}

var allFirms = slices.Concat(firms, syndicates)

func firmByID(id string) (firm, bool) {
	i := slices.IndexFunc(allFirms, func(f firm) bool { return f.id == id })
	if i < 0 {
		return firm{}, false
	}
	return allFirms[i], true
}

func findFirm(query string) (firm, bool) {
	q := normalizeItemName(query)
	if q == "" {
		return firm{}, false
	}
	keys := func(f firm) []string {
		keys := []string{normalizeItemName(f.name), f.id}
		for _, alias := range f.aliases {
			keys = append(keys, normalizeItemName(alias))
		}
		return keys
	}
	matchers := []func(key string) bool{
		func(key string) bool { return key == q },
		func(key string) bool { return strings.HasPrefix(key, q) },
		func(key string) bool { return strings.Contains(key, q) },
	}
	for _, match := range matchers {
		if i := slices.IndexFunc(allFirms, func(f firm) bool { return slices.ContainsFunc(keys(f), match) }); i >= 0 {
			return allFirms[i], true
		}
	}
	return firm{}, false
}

func (f firm) label() string {
	return f.emoji + " *" + f.name + "*"
}

func (f firm) fee(balance int64) int64 {
	return max(f.floor, balance*f.rate/100)
}

func (f firm) perks() string {
	if f.sector == crimeSector {
		return f.backing()
	}
	reach := "!roubar"
	if f.gangs {
		reach = "!roubar e !quadrilha"
	}
	return fmt.Sprintf("tem %d%% de chance de tirar sua fama de alvo fácil em cada %s (metade disso pra quem só lucrou alto esses dias) e segura %d%% do que você perder nos eventos de quem ganhou muito na aposta",
		percent(f.annul), reach, f.cover)
}

type contract struct {
	firm  firm
	until time.Time
	boost time.Time
}

func contractOf(ctx context.Context, q rowQuerier, chat types.JID, acc core.Account, now time.Time) (contract, bool, error) {
	return contractIn(ctx, q, chat, acc, securitySector, now)
}

func contractIn(ctx context.Context, q rowQuerier, chat types.JID, acc core.Account, sector string, now time.Time) (c contract, found bool, err error) {
	var id string
	var until, boost int64
	err = q.QueryRowContext(ctx,
		"SELECT empresa, ate, reforco_ate FROM contratos WHERE chat = ? AND pessoa = ? AND ramo = ? AND ate > ?",
		chat.String(), acc.Key, sector, now.Unix(),
	).Scan(&id, &until, &boost)
	if errors.Is(err, sql.ErrNoRows) {
		return c, false, nil
	}
	if err != nil {
		return c, false, err
	}
	f, ok := firmByID(id)
	return contract{firm: f, until: time.Unix(until, 0), boost: time.Unix(boost, 0)}, ok, nil
}

func (s *walletStore) contracts(ctx context.Context, chat types.JID, acc core.Account, now time.Time) ([]contract, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT empresa, ate, reforco_ate FROM contratos WHERE chat = ? AND pessoa = ? AND ate > ? ORDER BY ramo = ?",
		chat.String(), acc.Key, now.Unix(), crimeSector,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var contracts []contract
	for rows.Next() {
		var id string
		var until, boost int64
		if err := rows.Scan(&id, &until, &boost); err != nil {
			return nil, err
		}
		if f, ok := firmByID(id); ok {
			contracts = append(contracts, contract{firm: f, until: time.Unix(until, 0), boost: time.Unix(boost, 0)})
		}
	}
	return contracts, rows.Err()
}

func contractText(contracts []contract, now time.Time) string {
	lines := make([]string, 0, len(contracts))
	for _, c := range contracts {
		text := fmt.Sprintf("📄 Contrato com a %s por mais %s", c.firm.label(), core.FormatWait(c.until.Sub(now)))
		if now.Before(c.boost) {
			text += ", com reforço"
		}
		lines = append(lines, text)
	}
	return strings.Join(lines, "\n")
}

type firmOffer interface {
	expiresAt() time.Time
}

func takeOffer[T firmOffer](offers *sync.Map, key string, now time.Time) (T, bool) {
	var none T
	value, ok := offers.LoadAndDelete(key)
	if !ok {
		return none, false
	}
	o := value.(T)
	if now.After(o.expiresAt()) {
		return none, false
	}
	return o, true
}

func (b *bot) answerFirmOffer(ctx context.Context, key string, accepted bool) (bool, error) {
	now := time.Now()
	if o, ok := takeOffer[*boostOffer](&b.boostOffers, key, now); ok {
		if !accepted {
			return true, b.dismissFirm(ctx, o.chat, o.acc, o.firm)
		}
		return true, b.acceptBoost(ctx, o)
	}
	if o, ok := takeOffer[*recoveryOffer](&b.recoveryOffers, key, now); ok {
		if !accepted {
			return true, b.dismissFirm(ctx, o.chat, o.victim, o.firm)
		}
		return true, b.acceptRecovery(ctx, o)
	}
	if o, ok := takeOffer[*breakoutOffer](&b.breakoutOffers, key, now); ok {
		if !accepted {
			return true, b.dismissFirm(ctx, o.chat, o.acc, o.firm)
		}
		return true, b.acceptBreakout(ctx, o)
	}
	return false, nil
}

func (b *bot) dismissFirm(ctx context.Context, chat types.JID, acc core.Account, f firm) error {
	_, err := b.SendMentions(ctx, chat, fmt.Sprintf("🙅 @%s dispensou a %s", acc.JID.User, f.label()), []types.JID{acc.JID})
	return err
}

type hiring struct {
	fee     int64
	urgency int64
	until   time.Time
}

func (h hiring) total() int64 {
	return h.fee + h.urgency
}

func (s *walletStore) sign(ctx context.Context, chat types.JID, acc core.Account, f firm, now time.Time) (h hiring, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		var balance int64
		if err := tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&balance); err != nil {
			return err
		}
		h.fee = f.fee(balance)
		if h.urgency, err = f.emergency(ctx, tx, chat, acc, now); err != nil {
			return err
		}
		if err := debitBalance(ctx, tx, chat, acc, h.total()); err != nil {
			return err
		}
		span := int64(contractSpan.Seconds())
		_, err = tx.ExecContext(ctx,
			`INSERT INTO contratos (chat, pessoa, ramo, empresa, ate) VALUES (?, ?, ?, ?, ?)
			 ON CONFLICT (chat, pessoa, ramo) DO UPDATE SET
			   ate = CASE WHEN contratos.empresa = excluded.empresa AND contratos.ate > ? THEN contratos.ate + ? ELSE excluded.ate END,
			   empresa = excluded.empresa`,
			chat.String(), acc.Key, f.sector, f.id, now.Unix()+span, now.Unix(), span,
		)
		if err != nil {
			return err
		}
		var until int64
		err = tx.QueryRowContext(ctx, "SELECT ate FROM contratos WHERE chat = ? AND pessoa = ? AND ramo = ?", chat.String(), acc.Key, f.sector).Scan(&until)
		h.until = time.Unix(until, 0)
		return err
	})
	return h, err
}

func (f firm) emergency(ctx context.Context, q rowQuerier, chat types.JID, acc core.Account, now time.Time) (int64, error) {
	if f.sector != securitySector {
		return 0, nil
	}
	jackpot, err := freshJackpot(ctx, q, chat, acc, now)
	return jackpot * emergencyCut / 100, err
}

func (s *walletStore) cancelContract(ctx context.Context, chat types.JID, acc core.Account, sector string, now time.Time) (f firm, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		c, found, err := contractIn(ctx, tx, chat, acc, sector, now)
		if err != nil {
			return err
		}
		if !found {
			return errNoContract
		}
		f = c.firm
		_, err = tx.ExecContext(ctx, "DELETE FROM contratos WHERE chat = ? AND pessoa = ? AND ramo = ?", chat.String(), acc.Key, sector)
		return err
	})
	return f, err
}

type sector struct {
	id     string
	title  string
	firms  []firm
	footer string
}

func sectors() []sector {
	return []sector{
		{
			id: securitySector, title: "🏢 *Empresas de segurança*", firms: firms,
			footer: fmt.Sprintf("Quem contrata logo depois de ganhar na aposta paga mais %d%% do prêmio de taxa de emergência", emergencyCut),
		},
		{
			id: crimeSector, title: "🕴️ *Organizações do crime*", firms: syndicates,
			footer: "Se você rodar com o contrato ativo, a organização oferece o resgate na hora, é só !aceitar",
		},
	}
}

func (b *bot) empresa(ctx context.Context, evt *events.Message) error {
	text := core.MessageText(evt.Message)
	args := strings.Fields(core.FoldLetters(text))[1:]
	all := sectors()
	listing := func(shown ...sector) func(context.Context, *events.Message, string) error {
		return func(ctx context.Context, evt *events.Message, _ string) error { return b.listFirms(ctx, evt, shown) }
	}
	if len(args) == 0 {
		return b.listFirms(ctx, evt, all)
	}
	actions := map[string]func(context.Context, *events.Message, string) error{
		"lista": listing(all...), "planos": listing(all...), "empresas": listing(all...),
		"seguranca": listing(all[0]), "crime": listing(all[1]), "organizacao": listing(all[1]), "organizacoes": listing(all[1]),
		"contratar": b.signFirm, "assinar": b.signFirm, "renovar": b.signFirm,
		"cancelar": b.cancelFirm, "encerrar": b.cancelFirm,
	}
	action, ok := actions[args[0]]
	if !ok {
		return b.SendText(ctx, evt.Info.Chat.ToNonAD(), "Use assim: !empresa, !empresa crime, !empresa contratar <nome> ou !empresa cancelar")
	}
	return action(ctx, evt, core.TextAfterFields(text, 2))
}

func (b *bot) listFirms(ctx context.Context, evt *events.Message, shown []sector) error {
	chat := evt.Info.Chat.ToNonAD()
	acc := b.Account(ctx, evt.Info.Sender.ToNonAD())
	now := time.Now()
	var (
		balance   int64
		contracts []contract
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { balance, _, err = b.Wallets.Balance(gctx, chat, acc); return err })
	g.Go(func() (err error) { contracts, err = b.Wallets.contracts(gctx, chat, acc, now); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler empresas: %w", err)
	}
	var lines []string
	if text := contractText(contracts, now); text != "" {
		lines = append(lines, text, "")
	}
	for _, sec := range shown {
		lines = append(lines, sec.title)
		for _, f := range sec.firms {
			lines = append(lines, "", f.label()+" "+f.perks(),
				fmt.Sprintf("💰 Diária: %d%% do saldo, no mínimo %s. Pra você agora: %s", f.rate, Coins(f.floor), Coins(f.fee(balance))))
		}
		lines = append(lines, "", "_"+sec.footer+"_", "")
	}
	lines = append(lines, fmt.Sprintf("_Manda !empresa contratar <nome>. O contrato vale %s, e dá pra ter um de segurança e um com o crime ao mesmo tempo_", core.FormatWait(contractSpan)))
	return b.SendText(ctx, chat, strings.Join(lines, "\n"))
}

func (b *bot) signFirm(ctx context.Context, evt *events.Message, query string) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	f, ok := findFirm(query)
	if !ok {
		return b.SendText(ctx, chat, "Fala qual empresa você quer contratar, tipo !empresa contratar escolta armada. Manda !empresa pra ver a lista")
	}
	now := time.Now()
	h, err := b.Wallets.sign(ctx, chat, b.Account(ctx, person), f, now)
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, chat, fmt.Sprintf("A %s sai %s pra você e você não tem tudo isso. Confere no !saldo", f.label(), Coins(h.total())))
	}
	if err != nil {
		return fmt.Errorf("contratar empresa: %w", err)
	}
	text := fmt.Sprintf("🤝 @%s contratou a %s por %s, e o contrato vale por mais %s\n✨ Ela %s",
		person.User, f.label(), Coins(h.fee), core.FormatWait(h.until.Sub(now)), f.perks())
	if h.urgency > 0 {
		text += fmt.Sprintf("\n🚨 Mais %s de taxa de emergência, porque você acabou de ganhar na aposta e já tá marcado(a)", Coins(h.urgency))
	}
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) cancelFirm(ctx context.Context, evt *events.Message, query string) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	now := time.Now()
	sector, refusal, err := b.cancelledSector(ctx, chat, acc, query, now)
	if err != nil {
		return err
	}
	if refusal != "" {
		return b.SendText(ctx, chat, refusal)
	}
	f, err := b.Wallets.cancelContract(ctx, chat, acc, sector, now)
	if errors.Is(err, errNoContract) {
		return b.SendText(ctx, chat, "Você não tem contrato com essa empresa. Manda !empresa pra ver as opções")
	}
	if err != nil {
		return fmt.Errorf("cancelar empresa: %w", err)
	}
	_, err = b.SendMentions(ctx, chat, fmt.Sprintf("📄 @%s cancelou o contrato com a %s. O que já foi pago não volta", person.User, f.label()), []types.JID{person})
	return err
}

func (b *bot) cancelledSector(ctx context.Context, chat types.JID, acc core.Account, query string, now time.Time) (sector, refusal string, err error) {
	if strings.TrimSpace(query) != "" {
		f, ok := findFirm(query)
		if !ok {
			return "", "Não achei essa empresa. Manda !empresa pra ver os nomes", nil
		}
		return f.sector, "", nil
	}
	contracts, err := b.Wallets.contracts(ctx, chat, acc, now)
	if err != nil {
		return "", "", fmt.Errorf("ler contratos: %w", err)
	}
	if len(contracts) == 0 {
		return "", "Você não tem contrato com nenhuma empresa. Manda !empresa pra ver as opções", nil
	}
	if len(contracts) == 1 {
		return contracts[0].firm.sector, "", nil
	}
	return "", fmt.Sprintf("Você tem contrato com a %s e com a %s. Fala qual, tipo !empresa cancelar %s", contracts[0].firm.label(), contracts[1].firm.label(), strings.ToLower(contracts[1].firm.name)), nil
}
