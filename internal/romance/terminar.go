package romance

import (
	"cmp"
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/economy"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

func (b *bot) terminar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	sender := evt.Info.Sender.ToNonAD()
	mentions := core.UniqueJIDs(core.MentionedJIDs(evt.Message))
	if len(mentions) > 2 {
		return b.SendText(ctx, chat, "Término é de duas em duas pessoas: !terminar @pessoa ou !terminar @pessoa1 @pessoa2")
	}
	people := append([]types.JID{sender}, mentions...)
	if len(mentions) == 2 {
		people = mentions
	}

	var (
		all     []core.Relationship
		consent bool
	)
	aliases := make([][]string, len(people))
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) {
		all, err = b.Relationships.InChat(gctx, chat, kindDating, core.KindMarriage, kindLover)
		return err
	})
	g.Go(func() (err error) { consent, err = b.ChatModeOn(gctx, chat, core.ModeConsent); return err })
	for i, p := range people {
		g.Go(func() error { aliases[i] = b.UserAliases(gctx, p); return nil })
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler relacionamentos: %w", err)
	}
	if consent && !slices.ContainsFunc(aliases, func(a []string) bool { return core.SentByAny(evt.Info, a) }) {
		return b.SendText(ctx, chat, "Com o consentimento ligado, só quem tá na relação pode terminar: !terminar @pessoa")
	}

	ended := core.RelationshipsOf(all, aliases[0])
	if len(people) == 2 {
		ended = slices.DeleteFunc(ended, func(r core.Relationship) bool { return !r.Joins(aliases[0], aliases[1]) })
	}
	if len(ended) == 0 {
		return b.notTogether(ctx, chat, people)
	}
	if err := b.Relationships.Remove(ctx, ended); err != nil {
		return fmt.Errorf("apagar relacionamentos: %w", err)
	}

	for i := range ended {
		ended[i].Partners = initiatorFirst(ended[i].Partners, aliases[0])
	}
	text, mentioned := breakupText(ended, time.Now())
	marriages := core.ConsentedOnly(ofKind(ended, core.KindMarriage))
	text = core.AppendLine(text, b.economy.SplitJointAccounts(ctx, chat, marriages))
	if payer := slices.IndexFunc(aliases, func(a []string) bool { return core.SentByAny(evt.Info, a) }); payer >= 0 {
		exes := core.OtherPartners(marriages, aliases[payer], nil)
		text = core.AppendLine(text, b.economy.Charge(ctx, chat, people[payer], exes, economy.AlimonyPercent, "⚖️ Pensão"))
		mentioned = core.UniqueJIDs(append(mentioned, people[payer]))
	}
	_, err := b.SendMentions(ctx, chat, text, mentioned)
	return err
}

func (b *bot) notTogether(ctx context.Context, chat types.JID, people []types.JID) error {
	if len(people) == 1 {
		return b.SendText(ctx, chat, "Você nem namora nem é casado(a) com ninguém 🤨 Usa !namorar ou !casar primeiro")
	}
	_, err := b.SendMentions(ctx, chat, fmt.Sprintf("@%s e @%s nem namoram nem são casados 🤨", people[0].User, people[1].User), people)
	return err
}

func initiatorFirst(pair [2]types.JID, initiator []string) [2]types.JID {
	if slices.Contains(initiator, pair[1].User) {
		return [2]types.JID{pair[1], pair[0]}
	}
	return pair
}

func breakupText(breakups []core.Relationship, now time.Time) (string, []types.JID) {
	headers := map[[3]bool]string{
		{false, true, false}: "💔 *Fim do namoro*",
		{true, false, false}: "💔 *Divórcio*",
		{false, false, true}: "💔 *Fim do caso*",
	}
	templates := map[core.RelationshipKind]string{
		kindDating:        "@%s e @%s terminaram depois de %s de namoro",
		core.KindMarriage: "@%s e @%s se separaram depois de %s de casamento",
		kindLover:         "@%s e @%s acabaram o caso depois de %s se pegando escondido",
	}
	aftermaths := map[core.RelationshipKind][]string{
		kindDating: {
			"@{2} já tá soltando indireta no story 📱",
			"@{1} vai passar a noite ouvindo sofrência 😭",
			"@{2} ficou com o moletom de @{1} e não vai devolver 🧥",
			"Já tão devolvendo os presentes um por um 🎁",
			"Já tem gente no grupo de olho em @{1} 👀",
			"Os dois continuam se seguindo só pra stalkear o story 🙈",
		},
		core.KindMarriage: {
			"A guarda do cachorro ficou com @{2} 🐶",
			"@{1} ficou com a casa e @{2} ficou com as dívidas 💸",
			"@{2} levou até a air fryer 🍟",
			"A pensão vai ser paga em figurinha 🃏",
			"@{1} já tá no Tinder de novo 📱",
			"Os dois vão fingir que não se conhecem no próximo rolê 🙈",
		},
		kindLover: {
			"@{1} apagou o contato de @{2}, que tava salvo como \"Pedreiro\" 📵",
			"@{2} vai ter que inventar outro álibi pras sextas à noite 🙈",
			"Os dois juram que nunca se viram na vida 🤐",
			"@{1} voltou a ser fiel... até a próxima 😇",
			"@{2} ficou com a chave do motel de lembrança 🏨",
		},
	}

	var (
		kinds     [3]bool
		lines     []string
		mentioned []types.JID
	)
	for _, br := range breakups {
		kinds[0] = kinds[0] || br.Kind == core.KindMarriage
		kinds[1] = kinds[1] || br.Kind == kindDating
		kinds[2] = kinds[2] || br.Kind == kindLover
		lines = append(lines, fmt.Sprintf(templates[br.Kind], br.Partners[0].User, br.Partners[1].User, marriageLength(now.Sub(br.Since))))
		mentioned = append(mentioned, br.Partners[0], br.Partners[1])
	}
	last := breakups[len(breakups)-1]
	pool := aftermaths[last.Kind]
	aftermath := strings.NewReplacer("{1}", last.Partners[0].User, "{2}", last.Partners[1].User).Replace(pool[rand.IntN(len(pool))])
	text := cmp.Or(headers[kinds], "💔 *Fim de tudo*") + "\n\n" + strings.Join(lines, "\n") + "\n\n" + aftermath
	return text, core.UniqueJIDs(mentioned)
}
