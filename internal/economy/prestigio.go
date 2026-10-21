package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	prestigeBase = 5_000_000
	prestigeTop  = 5
	prestigeStep = 2
)

var (
	errPrestigeDivine    = errors.New("deus não renasce")
	errPrestigeTop       = errors.New("já tá no prestígio máximo")
	errPrestigePresident = errors.New("presidente não renasce")
	errPrestigeDebt      = errors.New("quem deve não renasce")
	errPrestigeShort     = errors.New("o patrimônio não chega no próximo prestígio")
)

type prestigeRank struct {
	level     int
	worth     int64
	divine    bool
	president bool
	indebted  bool
}

func prestigeCost(level int) int64 {
	return prestigeBase * int64((level+1)*(level+1))
}

func prestigeBonus(level int) int64 {
	return int64(level * prestigeStep)
}

func prestigeBadge(level int) string {
	if level == 0 {
		return ""
	}
	return strings.Repeat("⭐", level) + " "
}

func prestigePerk(level int) string {
	return fmt.Sprintf("🌟 Prestígio %d: +%d%% no !trabalhar e no !daily (já conta nos valores)", level, prestigeBonus(level))
}

func (p Profession) prestiged(level int) Profession {
	p.prestige = level
	if p.divine {
		return p
	}
	bonus := prestigeBonus(level)
	p.min, p.max = p.min*(100+bonus)/100, p.max*(100+bonus)/100
	p.dailyBonus += bonus
	return p
}

func prestigeOf(ctx context.Context, q rowQuerier, chat types.JID, acc core.Account, now time.Time) (r prestigeRank, err error) {
	var profession string
	err = q.QueryRowContext(ctx,
		`SELECT c.profissao, c.prestigio, c.saldo + c.cofre_casa + COALESCE(p.valor, 0),
		        EXISTS (SELECT 1 FROM presidencias x WHERE x.chat = c.chat AND x.pessoa = c.pessoa AND x.mandato_ate > ?),
		        EXISTS (SELECT 1 FROM emprestimos e WHERE e.chat = c.chat AND e.pessoa = c.pessoa)
		         OR EXISTS (SELECT 1 FROM dividas d WHERE d.chat = c.chat AND d.devedor = c.pessoa)
		 FROM carteiras c LEFT JOIN poupancas p ON p.chat = c.chat AND p.pessoa = c.pessoa
		 WHERE c.chat = ? AND c.pessoa = ?`,
		now.Unix(), chat.String(), acc.Key,
	).Scan(&profession, &r.level, &r.worth, &r.president, &r.indebted)
	if errors.Is(err, sql.ErrNoRows) {
		return prestigeRank{}, nil
	}
	if err != nil {
		return prestigeRank{}, err
	}
	job, _ := professionNamed(profession)
	r.divine = job.divine
	return r, nil
}

func (r prestigeRank) blocker() error {
	blockers := []struct {
		on  bool
		err error
	}{
		{r.divine, errPrestigeDivine},
		{r.level >= prestigeTop, errPrestigeTop},
		{r.president, errPrestigePresident},
		{r.indebted, errPrestigeDebt},
		{r.worth < prestigeCost(r.level), errPrestigeShort},
	}
	for _, block := range blockers {
		if block.on {
			return block.err
		}
	}
	return nil
}

func (r prestigeRank) refusal(err error) string {
	refusals := []struct {
		err  error
		text string
	}{
		{errPrestigeDivine, "⚡ Deus não renasce: já nasceu no topo"},
		{errPrestigeTop, fmt.Sprintf("🌟 Você já tá no prestígio máximo (%d). Daqui não tem mais pra onde renascer", prestigeTop)},
		{errPrestigePresident, "🏛️ Presidente não renasce no meio do mandato. Larga o cargo com !renunciar primeiro"},
		{errPrestigeDebt, "🦈 Quem deve não renasce: quita o !emprestimo e o !agiota antes"},
		{errPrestigeShort, fmt.Sprintf("💰 Pro prestígio %d precisa de %s de patrimônio (saldo, poupança e cofre de casa). Você tem %s, falta %s", r.level+1, Coins(prestigeCost(r.level)), Coins(r.worth), Coins(prestigeCost(r.level)-r.worth))},
	}
	for _, refused := range refusals {
		if errors.Is(err, refused.err) {
			return refused.text
		}
	}
	return ""
}

func (s *walletStore) rebirth(ctx context.Context, chat types.JID, acc core.Account, now time.Time) (r prestigeRank, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) (err error) {
		if r, err = prestigeOf(ctx, tx, chat, acc, now); err != nil {
			return err
		}
		if err := r.blocker(); err != nil {
			return err
		}
		wiped := map[string]string{
			"inventarios":    "pessoa",
			"guardados":      "pessoa",
			"poupancas":      "pessoa",
			"acoes_posicoes": "pessoa",
			"empregados":     "pessoa",
			"implantes":      "pessoa",
			"plantacoes":     "pessoa",
			"titulos":        "pessoa",
			"feira_anuncios": "vendedor",
		}
		for table, owner := range wiped {
			if _, err := tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE chat = ? AND "+owner+" = ?", chat.String(), acc.Key); err != nil {
				return err
			}
		}
		_, err = tx.ExecContext(ctx,
			`UPDATE carteiras SET saldo = 0, cofre_casa = 0, imovel = 0, titulo = '', canteiros_comprados = 0, prestigio = prestigio + 1
			 WHERE chat = ? AND pessoa = ?`,
			chat.String(), acc.Key,
		)
		if err != nil {
			return err
		}
		return noteLoss(ctx, tx, chat, acc, lossReborn, r.worth)
	})
	return r, err
}

func (b *bot) prestigio(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	now := time.Now()
	args := strings.Fields(core.FoldLetters(core.MessageText(evt.Message)))[1:]
	if len(args) > 0 && slices.Contains([]string{"confirmar", "confirmo"}, args[0]) {
		return b.reborn(ctx, chat, person, acc, now)
	}
	r, err := prestigeOf(ctx, b.Wallets.db, chat, acc, now)
	if err != nil {
		return fmt.Errorf("ler prestígio: %w", err)
	}
	return b.SendText(ctx, chat, r.menu())
}

func (b *bot) reborn(ctx context.Context, chat, person types.JID, acc core.Account, now time.Time) error {
	r, err := b.Wallets.rebirth(ctx, chat, acc, now)
	if refusal := r.refusal(err); refusal != "" {
		return b.SendText(ctx, chat, refusal)
	}
	if err != nil {
		return fmt.Errorf("renascer no prestígio: %w", err)
	}
	level := r.level + 1
	_, err = b.SendMentions(ctx, chat, fmt.Sprintf(
		"🌟 @%s renasceu! Deixou %s pra trás e agora tá no prestígio %d %s: +%d%% no !trabalhar e no !daily pra sempre\n_Começa do zero: saldo, mochila, imóvel e o resto viraram pó. A profissão e a carreira continuam_",
		person.User, Coins(r.worth), level, strings.TrimSpace(prestigeBadge(level)), prestigeBonus(level),
	), []types.JID{person})
	return err
}

func (r prestigeRank) menu() string {
	lines := []string{
		"🌟 *Prestígio*",
		fmt.Sprintf("_Quem junta patrimônio suficiente (saldo, poupança e cofre de casa) pode renascer: perde tudo que tem e ganha um bônus pequeno pra sempre, que acumula até o prestígio %d_", prestigeTop),
		"",
		r.standingLine(),
	}
	if r.level < prestigeTop {
		lines = append(lines, fmt.Sprintf("Pro prestígio %d: %s de patrimônio. Ele dá +%d%% no !trabalhar e no !daily e mais uma ⭐ no seu nome", r.level+1, Coins(prestigeCost(r.level)), prestigeBonus(r.level+1)))
	}
	lines = append(lines, "",
		"🔥 Renascer apaga: saldo, poupança, cofre e imóvel, mochila e guardados, ações, empregados, implantes, títulos, horta e anúncios da !feira",
		"✅ Fica: profissão e carreira, fama, igreja, romance, saúde e ficha",
		"",
	)
	if refusal := r.refusal(r.blocker()); refusal != "" {
		return strings.Join(append(lines, refusal), "\n")
	}
	return strings.Join(append(lines, "Tá tudo certo pra renascer. Manda !prestigio confirmar"), "\n")
}

func (r prestigeRank) standingLine() string {
	if r.level == 0 {
		return "Você ainda não renasceu nenhuma vez"
	}
	return fmt.Sprintf("Você tá no prestígio %d %s: +%d%% no !trabalhar e no !daily", r.level, strings.TrimSpace(prestigeBadge(r.level)), prestigeBonus(r.level))
}
