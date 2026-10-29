package economy

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

func (b *bot) mochila(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := core.FirstMentionOrSender(evt).ToNonAD()
	held, favorites, err := b.shelf(ctx, chat, b.Account(ctx, person))
	if err != nil {
		return err
	}
	_, err = b.SendMentions(ctx, chat, backpackText(person, held, favorites), []types.JID{person})
	return err
}

func backpackText(person types.JID, held []stack, favorites map[string]bool) string {
	if len(held) == 0 {
		return fmt.Sprintf("🎒 A mochila de @%s tá vazia 🪹\n\n_Dá pra conseguir itens com !vasculhar, !pescar, !minerar e !catar, ou comprando na !loja_", person.User)
	}
	type lot struct {
		item  item
		grade grade
	}
	var (
		units, worth, market int64
		lots                 []lot
		graded               bool
	)
	for _, s := range held {
		units += s.quantity
		worth += s.worth()
		market += marketPayoutOf(s.item, s.grades, marketSalePercent)
		for _, g := range s.grades {
			lots = append(lots, lot{s.item, g})
			graded = graded || g.quality != normalQuality
		}
	}
	lines := []string{fmt.Sprintf("🎒 *Mochila de @%s*", person.User)}
	lines = append(lines, categoryGroups(lots, func(l lot) item { return l.item }, func(l lot) string {
		return fmt.Sprintf("#%d %s %dx %s %s%s%s · 🪙 %s", l.item.number(), rarities[l.item.rarity].emoji, l.grade.quantity, l.item.emoji,
			l.item.name, l.grade.quality.tag(), favoriteMark(favorites[l.item.id]), core.Thousands(l.grade.quality.price(l.item.value)))
	})...)
	lines = append(lines, "", rarityLegend())
	if graded {
		lines = append(lines, "Qualidade: "+qualityGuide(), "_Vender, dar ou usar sempre tira primeiro o de pior qualidade_")
	}
	lines = append(lines, "",
		fmt.Sprintf("📦 %s, valendo %s", core.CountLabel(int(units), "item", "itens"), Coins(worth)),
		fmt.Sprintf("_No mercado sairia tudo por %s. Manda !item <nome> pra ver o que cada um faz_", Coins(market)),
		"_Pra vender vários de uma vez, manda os números com #: !vender #12 #40 #87_",
		"_⭐ é favorito, que não sai em nenhuma venda (só as cópias no !vender duplicados). Favorita com !favoritar <item> ou !favoritar tudo_")
	return strings.Join(lines, "\n")
}

func favoriteMark(favorite bool) string {
	if !favorite {
		return ""
	}
	return " ⭐"
}

func (b *bot) heldItem(ctx context.Context, chat types.JID, acc core.Account, name string) (item, int64, string, error) {
	held, err := b.Wallets.inventory(ctx, chat, acc)
	if err != nil {
		return item{}, 0, "", fmt.Errorf("ler mochila: %w", err)
	}
	matches := preferItems(findItems(name), ownedIn(held))
	if problem := itemQueryProblem(name, matches); problem != "" {
		return item{}, 0, problem, nil
	}
	it := matches[0]
	have := quantityOf(held, it.id)
	if have == 0 {
		return it, 0, fmt.Sprintf("Não tem %s na sua mochila 🎒 Manda !mochila pra ver o que você tem", it.label()), nil
	}
	return it, have, "", nil
}

func (b *bot) dar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	usageText := "Use assim: !dar @pessoa todynho (ou !dar @pessoa todynho 3, ou tudo)"
	mentions := core.MentionedJIDs(evt.Message)
	q := parseItemQuery(core.MessageText(evt.Message))
	if len(mentions) == 0 || q.name == "" {
		return b.SendText(ctx, chat, usageText)
	}
	from, to := evt.Info.Sender.ToNonAD(), mentions[0].ToNonAD()
	fromAcc, toAcc := b.Account(ctx, from), b.Account(ctx, to)
	if fromAcc.Key == toAcc.Key {
		return b.SendText(ctx, chat, "Dar presente pra você mesmo é só guardar na mochila 🤨")
	}

	it, have, problem, err := b.heldItem(ctx, chat, fromAcc, q.name)
	if err != nil {
		return err
	}
	if problem != "" {
		return b.SendText(ctx, chat, problem)
	}
	quantity := q.quantity(have)
	if quantity > have {
		return b.SendText(ctx, chat, "Você só tem "+quantityLabel(it, have))
	}
	grades, err := b.Wallets.giveItem(ctx, chat, fromAcc, toAcc, it.id, quantity)
	if errors.Is(err, errBagFull) {
		_, err = b.SendMentions(ctx, chat, carryRefusal(it, "a de @"+to.User), []types.JID{to})
		return err
	}
	if errors.Is(err, errNoItems) {
		return b.SendText(ctx, chat, "Sua mochila mudou no meio do caminho e não dá mais 🤨 Confere no !mochila")
	}
	if err != nil {
		return fmt.Errorf("dar item: %w", err)
	}
	text := fmt.Sprintf("🎁 @%s deu %s%s pra @%s", from.User, quantityLabel(it, quantity), gradesLabel(grades), to.User)
	if it.gift != "" && quantity == 1 {
		text = fmt.Sprintf(it.gift, from.User, to.User)
	}
	_, err = b.SendMentions(ctx, chat, text, []types.JID{from, to})
	return err
}

func (b *bot) usar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	q := parseItemQuery(core.MessageText(evt.Message))
	if q.name == "" && q.all {
		return b.openAll(ctx, evt)
	}
	if q.name == "" {
		return b.SendText(ctx, chat, "Diz o que quer usar: !usar todynho, !usar caixa, !usar raspadinha... Manda !mochila pra ver o que você tem")
	}
	it, _, problem, err := b.heldItem(ctx, chat, acc, q.name)
	if err != nil {
		return err
	}
	if problem != "" {
		return b.SendText(ctx, chat, problem)
	}
	if it.graft != nil {
		return b.implantSurgery(ctx, evt, *it.graft)
	}
	if it.use == "" {
		return b.SendText(ctx, chat, unusableText(it))
	}
	if woke, err := b.awaken(ctx, evt, it); woke || err != nil {
		return err
	}
	if it.high != nil {
		return b.takeDrug(ctx, evt, it)
	}
	if it.bless != "" {
		return b.savorLuxury(ctx, evt, it)
	}
	if it.hex != "" {
		return b.hurlCurse(ctx, evt, it)
	}
	if it.id == forgedReleaseID {
		return b.forgeRelease(ctx, evt, it)
	}
	if it.heal > 0 {
		return b.healWith(ctx, evt, it)
	}
	text := fmt.Sprintf(it.use, person.User)
	if it.keep {
		_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
		return err
	}

	luck, err := b.boxLuck(ctx, chat, acc, it)
	if err != nil {
		return err
	}
	effect := it.roll(luck)
	wait, err := b.Wallets.consumeItem(ctx, chat, acc, it, effect, time.Now())
	if errors.Is(err, errNoItems) {
		return b.SendText(ctx, chat, fmt.Sprintf("Não tem mais %s na sua mochila 🎒", it.label()))
	}
	if err != nil {
		return fmt.Errorf("usar item: %w", err)
	}
	if wait > 0 {
		text = fmt.Sprintf(it.limit.tired, person.User, it.label(), core.FormatWait(wait))
		_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
		return err
	}
	_, err = b.SendMentions(ctx, chat, text+effectText(it, effect), []types.JID{person})
	return err
}

func unusableText(it item) string {
	if p := perk(it); p != "" {
		return fmt.Sprintf("%s não precisa de !usar, funciona só de estar na mochila: %s", it.label(), p)
	}
	return fmt.Sprintf("%s não dá pra usar. Dá pra guardar, !vender ou !dar pra alguém", it.label())
}

func (b *bot) boxLuck(ctx context.Context, chat types.JID, acc core.Account, it item) (int, error) {
	if it.loot == nil && it.prizes == nil {
		return 0, nil
	}
	var (
		job  Profession
		jinx int
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { job, err = b.ProfessionOf(gctx, chat, acc); return err })
	g.Go(func() (err error) { jinx, err = b.Wallets.boxJinx(gctx, chat, acc, time.Now()); return err })
	if err := g.Wait(); err != nil {
		return 0, err
	}
	return job.boxLuck - jinx, nil
}

func effectText(it item, e itemEffect) string {
	if e.prize.rarity == primordial {
		return fmt.Sprintf("\n\n⚫ *PRIMORDIAL!* Saiu %s, mais raro que qualquer relíquia\n%s\n✨ %s", e.prize.label(), e.prize.summary(), perk(e.prize))
	}
	if e.prize.rarity == relic {
		return fmt.Sprintf("\n\n🏛️ *RELÍQUIA!* Saiu %s, uma das coisas mais raras que existem\n%s", e.prize.label(), e.prize.summary())
	}
	if e.prize.id != "" {
		return fmt.Sprintf("\n\n✨ Saiu %s\n%s", e.prize.label(), e.prize.summary())
	}
	if it.prizes != nil && e.coins == 0 {
		return "\n\n💸 Não ganhou nada. Fica pra próxima"
	}
	if it.prizes != nil {
		return "\n\n💰 Ganhou " + Coins(e.coins)
	}
	if it.resets != nil {
		return "\n\n⏱️ Já dá pra usar o " + resetCommands(it.resets) + " de novo"
	}
	return ""
}
