package economy

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	hitEvery        = 6 * time.Hour
	hitTraceOnWin   = 0.25
	hitTraceOnMiss  = 0.5
	hitHostageHold  = 20 * time.Minute
	hitMinVictimPay = robMinVictim
)

type hitKind struct {
	id            string
	name          string
	emoji         string
	aliases       []string
	kidnap        bool
	floor         int64
	clientPercent int64
	targetPercent int64
	odds          float64
}

var hitKinds = []hitKind{
	{id: "roubo", name: "roubo", emoji: "💰", aliases: []string{"roubar", "assalto"}, floor: 30_000, clientPercent: 2, targetPercent: 20, odds: 0.55},
	{id: "sequestro", name: "sequestro", emoji: "⛓️", aliases: []string{"sequestrar", "cativeiro"}, kidnap: true, floor: 60_000, clientPercent: 4, targetPercent: 30, odds: 0.4},
}

func (k hitKind) price(clientBalance, targetBalance int64) int64 {
	return max(k.floor, clientBalance*k.clientPercent/100, targetBalance*k.targetPercent/100)
}

func (k hitKind) terms() string {
	return fmt.Sprintf("%s *%s*: %d%% do seu saldo ou %d%% do saldo do alvo, o que for maior (no mínimo %s), com %d%% de chance base",
		k.emoji, k.name, k.clientPercent, k.targetPercent, Coins(k.floor), percent(k.odds))
}

func hitKindIn(text string) (hitKind, bool) {
	words := strings.Fields(core.FoldLetters(text))
	i := slices.IndexFunc(hitKinds, func(k hitKind) bool {
		return slices.ContainsFunc(words, func(w string) bool { return w == k.id || slices.Contains(k.aliases, w) })
	})
	if i < 0 {
		return hitKind{}, false
	}
	return hitKinds[i], true
}

type hitJob struct {
	kind   hitKind
	odds   float64
	client core.Account
	victim core.Account
}

type hit struct {
	wait    time.Duration
	shield  time.Duration
	poor    bool
	held    bool
	price   int64
	success bool
	loot    int64
	traced  bool
}

func hitMenu() string {
	lines := []string{"🕴️ *Encomendas*", "_Quem tem contrato com uma organização do crime (!empresa crime) pode pagar pra ela atacar alguém_", ""}
	for _, k := range hitKinds {
		lines = append(lines, k.terms())
	}
	lines = append(lines, "",
		fmt.Sprintf("_Manda !encomendar @pessoa roubo ou !encomendar @pessoa sequestro. A organização soma a chance dela, e a proteção do alvo tira. O dinheiro da encomenda some, dando certo ou não, e o que os capangas roubam fica com eles. Dá pra encomendar uma vez a cada %s, e a polícia pode rastrear a encomenda até você (%d%% das vezes quando dá certo, %d%% quando dá errado)_",
			core.FormatWait(hitEvery), percent(hitTraceOnWin), percent(hitTraceOnMiss)),
		targetHint)
	return strings.Join(lines, "\n")
}

func (b *bot) encomendar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	client := evt.Info.Sender.ToNonAD()
	aim, problem := b.targetOf(ctx, evt)
	if problem != "" {
		return b.SendText(ctx, chat, problem)
	}
	if !aim.found() {
		return b.SendText(ctx, chat, hitMenu())
	}
	kind, named := hitKindIn(strings.Join(strings.Fields(aim.rest)[1:], " "))
	if !named {
		return b.SendText(ctx, chat, hitMenu())
	}
	clientAcc, victimAcc := b.Account(ctx, client), b.Account(ctx, aim.person)
	if clientAcc.Key == victimAcc.Key || b.IsSelf(aim.person) {
		return b.SendText(ctx, chat, "🕴️ A organização não aceita encomenda contra o próprio cliente nem contra o bot")
	}
	now := time.Now()
	var (
		backer    firm
		victimJob Profession
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { backer, err = syndicateOf(gctx, b.Wallets.db, chat, clientAcc, now); return err })
	g.Go(func() (err error) { victimJob, err = b.ProfessionOf(gctx, chat, victimAcc); return err })
	if err := g.Wait(); err != nil {
		return err
	}
	if backer.id == "" {
		return b.SendText(ctx, chat, "🕴️ Encomenda só sai pra quem tem contrato com uma organização do crime. Manda !empresa crime pra ver quem aceita o serviço")
	}
	if victimJob.divine {
		return b.SendText(ctx, chat, fmt.Sprintf("🕴️ A %s olhou o nome no papel e devolveu o pedido. Contra deus ninguém trabalha", backer.name))
	}
	job := hitJob{kind: kind, odds: clampChance(kind.odds + backer.edge - victimJob.guard), client: clientAcc, victim: victimAcc}
	h, err := b.Wallets.contractHit(ctx, chat, job, now)
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, chat, fmt.Sprintf("💸 A %s cobra %s por esse %s, e você não tem isso no saldo", backer.name, Coins(h.price), kind.name))
	}
	if err != nil {
		return fmt.Errorf("encomendar: %w", err)
	}
	_, err = b.SendMentions(ctx, chat, hitText(h, job, backer, client, aim.person), []types.JID{client, aim.person})
	return err
}

func hitText(h hit, job hitJob, backer firm, client, victim types.JID) string {
	refusals := []struct {
		on   bool
		text string
	}{
		{h.wait > 0, fmt.Sprintf("🕴️ A %s ainda tá cuidando da última encomenda de @%s. Volta em %s", backer.name, client.User, core.FormatWait(h.wait))},
		{h.shield > 0, shieldedText(victim, h.shield)},
		{h.poor, fmt.Sprintf("🕴️ @%s não tem nem %s pra roubar. A organização não sai de casa por tão pouco", victim.User, Coins(hitMinVictimPay))},
		{h.held, fmt.Sprintf("🕴️ @%s já tá preso(a), morto(a) ou num cativeiro. Não tem como sequestrar agora", victim.User)},
	}
	for _, r := range refusals {
		if r.on {
			return r.text
		}
	}
	lines := []string{fmt.Sprintf("🕴️ @%s encomendou um %s contra @%s pra %s *%s* e pagou %s", client.User, job.kind.name, victim.User, backer.emoji, backer.name, Coins(h.price))}
	outcomes := map[string]string{
		"roubo":     fmt.Sprintf("💰 Deu certo: os capangas levaram %s de @%s. A organização ficou com tudo", Coins(h.loot), victim.User),
		"sequestro": fmt.Sprintf("⛓️ Deu certo: @%s tá no cativeiro da organização por %s. A polícia não solta, mas dá pra tentar !fugir ou pagar o !resgate", victim.User, core.FormatWait(hitHostageHold)),
	}
	if h.success {
		lines = append(lines, outcomes[job.kind.id])
	}
	if !h.success {
		lines = append(lines, fmt.Sprintf("❌ Os capangas fizeram besteira e @%s escapou. O dinheiro da encomenda não volta", victim.User))
	}
	if h.traced {
		lines = append(lines, fmt.Sprintf("🚨 A polícia rastreou a encomenda até @%s: agora tá com crime na ficha e pode ir em cana com !prender ou !denunciar", client.User))
	}
	return strings.Join(lines, "\n")
}

func (s *walletStore) contractHit(ctx context.Context, chat types.JID, job hitJob, now time.Time) (h hit, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		for _, acc := range []core.Account{job.client, job.victim} {
			if err := openWallet(ctx, tx, chat, acc); err != nil {
				return err
			}
		}
		var (
			last, clientBalance, victimBalance int64
			blocked                            bool
		)
		err := tx.QueryRowContext(ctx,
			`SELECT c.encomenda_em, c.saldo, v.saldo,
			        v.preso_ate > ? OR v.morto_ate > ? OR EXISTS (SELECT 1 FROM sequestros s WHERE s.chat = v.chat AND s.refem = v.pessoa AND s.ate > ?)
			 FROM carteiras c JOIN carteiras v ON v.chat = c.chat AND v.pessoa = ?
			 WHERE c.chat = ? AND c.pessoa = ?`,
			now.Unix(), now.Unix(), now.Unix(), job.victim.Key, chat.String(), job.client.Key,
		).Scan(&last, &clientBalance, &victimBalance, &blocked)
		if err != nil {
			return err
		}
		if next := time.Unix(last, 0).Add(hitEvery); now.Before(next) {
			h.wait = next.Sub(now)
			return nil
		}
		if done, err := h.vet(ctx, tx, chat, job, victimBalance, blocked, now); err != nil || done {
			return err
		}
		h.price = job.kind.price(clientBalance, victimBalance)
		if err := debitBalance(ctx, tx, chat, job.client, h.price); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE carteiras SET encomenda_em = ? WHERE chat = ? AND pessoa = ?", now.Unix(), chat.String(), job.client.Key); err != nil {
			return err
		}
		h.success = rand.Float64() < job.odds
		h.traced = rand.Float64() < hitTraceOdds(h.success)
		if h.traced {
			if err := markCrime(ctx, tx, chat, job.client, now); err != nil {
				return err
			}
		}
		if !h.success {
			return nil
		}
		return h.strike(ctx, tx, chat, job, victimBalance, now)
	})
	return h, err
}

func hitTraceOdds(success bool) float64 {
	if success {
		return hitTraceOnWin
	}
	return hitTraceOnMiss
}

func (h *hit) vet(ctx context.Context, tx *ledgerTx, chat types.JID, job hitJob, victimBalance int64, blocked bool, now time.Time) (bool, error) {
	if job.kind.kidnap {
		h.held = blocked
		return h.held, nil
	}
	shield, err := shieldOn(ctx, tx, chat, job.victim, now)
	if err != nil {
		return false, err
	}
	h.shield = shield
	h.poor = victimBalance < hitMinVictimPay
	return h.shield > 0 || h.poor, nil
}

func (h *hit) strike(ctx context.Context, tx *ledgerTx, chat types.JID, job hitJob, victimBalance int64, now time.Time) error {
	if job.kind.kidnap {
		_, err := tx.ExecContext(ctx,
			"INSERT INTO sequestros (chat, refem, refem_jid, sequestrador, ate) VALUES (?, ?, ?, NULL, ?)",
			chat.String(), job.victim.Key, job.victim.JID.String(), now.Add(hitHostageHold).Unix(),
		)
		return err
	}
	h.loot = min(victimBalance, exposed(victimBalance, robLootMin+rand.Int64N(robLootMax-robLootMin+1)))
	if _, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo - ? WHERE chat = ? AND pessoa = ?", h.loot, chat.String(), job.victim.Key); err != nil {
		return err
	}
	if err := raiseShield(ctx, tx, chat, job.victim, now, robShield); err != nil {
		return err
	}
	return noteLoss(ctx, tx, chat, job.victim, lossRobbed, h.loot)
}
