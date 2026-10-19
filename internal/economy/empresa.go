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
	id      string
	name    string
	emoji   string
	annul   float64
	gangs   bool
	cover   int64
	rate    int64
	floor   int64
	aliases []string
}

var firms = []firm{
	{id: "vigia-de-rua", name: "Vigia de rua", emoji: "🔦", annul: 0.4, cover: 10, rate: 2, floor: 500, aliases: []string{"vigia"}},
	{id: "escolta-armada", name: "Escolta armada", emoji: "🚓", annul: 0.6, gangs: true, cover: 20, rate: 4, floor: 1_500, aliases: []string{"escolta"}},
	{id: "carro-forte", name: "Carro-forte", emoji: "🛡️", annul: 0.8, gangs: true, cover: 35, rate: 7, floor: 4_000},
}

func firmByID(id string) (firm, bool) {
	i := slices.IndexFunc(firms, func(f firm) bool { return f.id == id })
	if i < 0 {
		return firm{}, false
	}
	return firms[i], true
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
		if i := slices.IndexFunc(firms, func(f firm) bool { return slices.ContainsFunc(keys(f), match) }); i >= 0 {
			return firms[i], true
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

func contractOf(ctx context.Context, q rowQuerier, chat types.JID, acc core.Account, now time.Time) (c contract, found bool, err error) {
	var id string
	var until, boost int64
	err = q.QueryRowContext(ctx,
		"SELECT empresa, ate, reforco_ate FROM contratos WHERE chat = ? AND pessoa = ? AND ramo = ? AND ate > ?",
		chat.String(), acc.Key, securitySector, now.Unix(),
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

func contractText(c contract, hired bool, now time.Time) string {
	if !hired {
		return ""
	}
	text := fmt.Sprintf("📄 Contrato com a %s por mais %s", c.firm.label(), core.FormatWait(c.until.Sub(now)))
	if now.Before(c.boost) {
		text += ", com reforço"
	}
	return text
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
		jackpot, err := freshJackpot(ctx, tx, chat, acc, now)
		if err != nil {
			return err
		}
		h.fee, h.urgency = f.fee(balance), jackpot*emergencyCut/100
		if err := debitBalance(ctx, tx, chat, acc, h.total()); err != nil {
			return err
		}
		span := int64(contractSpan.Seconds())
		_, err = tx.ExecContext(ctx,
			`INSERT INTO contratos (chat, pessoa, ramo, empresa, ate) VALUES (?, ?, ?, ?, ?)
			 ON CONFLICT (chat, pessoa, ramo) DO UPDATE SET
			   ate = CASE WHEN contratos.empresa = excluded.empresa AND contratos.ate > ? THEN contratos.ate + ? ELSE excluded.ate END,
			   empresa = excluded.empresa`,
			chat.String(), acc.Key, securitySector, f.id, now.Unix()+span, now.Unix(), span,
		)
		if err != nil {
			return err
		}
		var until int64
		err = tx.QueryRowContext(ctx, "SELECT ate FROM contratos WHERE chat = ? AND pessoa = ? AND ramo = ?", chat.String(), acc.Key, securitySector).Scan(&until)
		h.until = time.Unix(until, 0)
		return err
	})
	return h, err
}

func (s *walletStore) cancelContract(ctx context.Context, chat types.JID, acc core.Account, now time.Time) (f firm, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		c, found, err := contractOf(ctx, tx, chat, acc, now)
		if err != nil {
			return err
		}
		if !found {
			return errNoContract
		}
		f = c.firm
		_, err = tx.ExecContext(ctx, "DELETE FROM contratos WHERE chat = ? AND pessoa = ? AND ramo = ?", chat.String(), acc.Key, securitySector)
		return err
	})
	return f, err
}

func (b *bot) empresa(ctx context.Context, evt *events.Message) error {
	text := core.MessageText(evt.Message)
	args := strings.Fields(core.FoldLetters(text))[1:]
	if len(args) == 0 {
		return b.listFirms(ctx, evt, "")
	}
	actions := map[string]func(context.Context, *events.Message, string) error{
		"lista": b.listFirms, "planos": b.listFirms, "empresas": b.listFirms,
		"contratar": b.signFirm, "assinar": b.signFirm, "renovar": b.signFirm,
		"cancelar": b.cancelFirm, "encerrar": b.cancelFirm,
	}
	action, ok := actions[args[0]]
	if !ok {
		return b.SendText(ctx, evt.Info.Chat.ToNonAD(), "Use assim: !empresa, !empresa contratar <nome> ou !empresa cancelar")
	}
	return action(ctx, evt, core.TextAfterFields(text, 2))
}

func (b *bot) listFirms(ctx context.Context, evt *events.Message, _ string) error {
	chat := evt.Info.Chat.ToNonAD()
	acc := b.Account(ctx, evt.Info.Sender.ToNonAD())
	now := time.Now()
	var (
		balance int64
		current contract
		hired   bool
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { balance, _, err = b.Wallets.Balance(gctx, chat, acc); return err })
	g.Go(func() (err error) { current, hired, err = contractOf(gctx, b.Wallets.db, chat, acc, now); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler empresas: %w", err)
	}
	lines := []string{"🏢 *Empresas de segurança*"}
	if text := contractText(current, hired, now); text != "" {
		lines = append(lines, text)
	}
	for _, f := range firms {
		lines = append(lines, "", f.label()+" "+f.perks(),
			fmt.Sprintf("💰 Diária: %d%% do saldo, no mínimo %s. Pra você agora: %s", f.rate, Coins(f.floor), Coins(f.fee(balance))))
	}
	lines = append(lines, "", fmt.Sprintf("_Manda !empresa contratar <nome>. O contrato vale %s, e quem contrata logo depois de ganhar na aposta paga mais %d%% do prêmio de taxa de emergência_",
		core.FormatWait(contractSpan), emergencyCut))
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

func (b *bot) cancelFirm(ctx context.Context, evt *events.Message, _ string) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	f, err := b.Wallets.cancelContract(ctx, chat, b.Account(ctx, person), time.Now())
	if errors.Is(err, errNoContract) {
		return b.SendText(ctx, chat, "Você não tem contrato com nenhuma empresa. Manda !empresa pra ver as opções")
	}
	if err != nil {
		return fmt.Errorf("cancelar empresa: %w", err)
	}
	_, err = b.SendMentions(ctx, chat, fmt.Sprintf("📄 @%s cancelou o contrato com a %s. O que já foi pago não volta", person.User, f.label()), []types.JID{person})
	return err
}
