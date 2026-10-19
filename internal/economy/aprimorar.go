package economy

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) aprimorar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	held, err := b.Wallets.inventory(ctx, chat, acc)
	if err != nil {
		return fmt.Errorf("ler mochila: %w", err)
	}
	query := parseItemQuery(core.MessageText(evt.Message)).name
	if query == "" {
		return b.SendText(ctx, chat, upgradeMenu(held))
	}
	matches := preferItems(preferItems(findItems(query), upgradable), ownedIn(held))
	if problem := itemQueryProblem(query, matches); problem != "" {
		return b.SendText(ctx, chat, problem)
	}
	base := matches[0]
	if problem := upgradeProblem(base, held); problem != "" {
		return b.SendText(ctx, chat, problem)
	}
	upgraded, _ := upgradeOf(base)
	err = b.Wallets.upgradeTool(ctx, chat, acc, base, upgraded)
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, chat, fmt.Sprintf("Aprimorar %s custa %s e você não tem tudo isso. Confere no !saldo", base.label(), Coins(upgraded.upgradeCost())))
	}
	if errors.Is(err, errNoItems) {
		return b.SendText(ctx, chat, fmt.Sprintf("%s saiu da sua mochila antes de chegar no ferreiro 🤔", base.label()))
	}
	if err != nil {
		return fmt.Errorf("aprimorar ferramenta: %w", err)
	}
	text := fmt.Sprintf("🔧 @%s levou %s pro ferreiro, pagou %s e voltou com %s\n\n%s\n\n_Ela se paga mais rápido que a original. Não dá pra aprimorar de novo, e se quebrar o aprimoramento vai junto_",
		person.User, base.label(), Coins(upgraded.upgradeCost()), upgraded.label(), upgradeGains(base, upgraded))
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func upgradable(it item) bool {
	_, ok := upgradeOf(it)
	return ok
}

func upgradeProblem(it item, held []stack) string {
	problems := []struct {
		on   bool
		text string
	}{
		{it.upgrades != "", fmt.Sprintf("%s já é aprimorada, e cada ferramenta só dá pra aprimorar uma vez 🔧", it.label())},
		{!upgradable(it), fmt.Sprintf("Não dá pra aprimorar %s 🤔 Só as ferramentas de extração raras, épicas e lendárias. Manda !aprimorar pra ver a lista", it.label())},
		{quantityOf(held, it.id) == 0, fmt.Sprintf("Você não tem %s na mochila, e ela tem que tá com você pra ir pro ferreiro", it.label())},
	}
	for _, p := range problems {
		if p.on {
			return p.text
		}
	}
	return ""
}

func upgradeGains(base, upgraded item) string {
	stats := []struct {
		label    string
		from, to string
	}{
		{"🍀 Sorte no que acha", fmt.Sprintf("%dx", base.gear.luck+1), fmt.Sprintf("%dx", upgraded.gear.luck+1)},
		{"🧹 Menos item comum e mão vazia", fmt.Sprintf("%d%%", base.gear.fewerJunk), fmt.Sprintf("%d%%", upgraded.gear.fewerJunk)},
		{"💎 Mais item raro, épico e lendário", fmt.Sprintf("%d%%", base.gear.moreRare), fmt.Sprintf("%d%%", upgraded.gear.moreRare)},
	}
	var gains []string
	for _, s := range stats {
		if s.from != s.to {
			gains = append(gains, fmt.Sprintf("%s: %s → %s", s.label, s.from, s.to))
		}
	}
	return strings.Join(gains, "\n")
}

func upgradeMenu(held []stack) string {
	lines := []string{
		"🔧 *Ferreiro*",
		"_Cada ferramenta de extração rara, épica ou lendária dá pra aprimorar uma vez só. A aprimorada acha coisa melhor e se paga mais rápido, mas continua sendo uma ferramenta só: se quebrar, o aprimoramento vai junto_",
	}
	for _, a := range activities {
		var tools []string
		for _, it := range catalog {
			up, ok := upgradeOf(it)
			if !ok || !it.gear.fits(a) {
				continue
			}
			line := fmt.Sprintf("%s %s %s · 🪙 %s", rarities[it.rarity].emoji, it.emoji, it.name, core.Thousands(up.upgradeCost()))
			if quantityOf(held, it.id) > 0 {
				line += " ✅"
			}
			tools = append(tools, line)
		}
		if len(tools) > 0 {
			lines = append(append(lines, "", a.emoji+" *"+a.command+"*"), tools...)
		}
	}
	lines = append(lines, "", "_✅ = tá na sua mochila. Pra aprimorar: !aprimorar <ferramenta>, tipo !aprimorar vara de carbono_")
	return strings.Join(lines, "\n")
}

func (s *walletStore) upgradeTool(ctx context.Context, chat types.JID, acc core.Account, base, upgraded item) error {
	return s.spend(ctx, chat, acc, upgraded.upgradeCost(), func(tx *ledgerTx) error {
		var wear int64
		err := tx.QueryRowContext(ctx,
			"SELECT COALESCE(MAX(usos), 0) FROM inventarios WHERE chat = ? AND pessoa = ? AND item = ?",
			chat.String(), acc.Key, base.id,
		).Scan(&wear)
		if err != nil {
			return err
		}
		if err := takeItems(ctx, tx, chat, acc, base.id, 1); err != nil {
			return err
		}
		if err := addItems(ctx, tx, chat, acc, upgraded.id, 1); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "UPDATE inventarios SET usos = MAX(usos, ?) WHERE chat = ? AND pessoa = ? AND item = ?",
			wear, chat.String(), acc.Key, upgraded.id)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx,
			"INSERT OR IGNORE INTO favoritos (chat, pessoa, item) SELECT chat, pessoa, ? FROM favoritos WHERE chat = ? AND pessoa = ? AND item = ?",
			upgraded.id, chat.String(), acc.Key, base.id)
		return err
	})
}
