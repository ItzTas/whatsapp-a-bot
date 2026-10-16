package main

import (
	"cmp"
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

func (b *bot) terminar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	sender := evt.Info.Sender.ToNonAD()
	mentions := uniqueJIDs(mentionedJIDs(evt.Message))
	if len(mentions) > 2 {
		return b.sendText(ctx, chat, "Término é de duas em duas pessoas: !terminar @pessoa ou !terminar @pessoa1 @pessoa2")
	}
	people := append([]types.JID{sender}, mentions...)
	if len(mentions) == 2 {
		people = mentions
	}

	var (
		all     []relationship
		consent bool
	)
	aliases := make([][]string, len(people))
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) {
		all, err = b.relationships.inChat(gctx, chat, kindDating, kindMarriage, kindLover)
		return err
	})
	g.Go(func() (err error) { consent, err = b.chatModeOn(gctx, chat, modeConsent); return err })
	for i, p := range people {
		g.Go(func() error { aliases[i] = b.userAliases(gctx, p); return nil })
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler relacionamentos: %w", err)
	}
	if consent && !slices.ContainsFunc(aliases, func(a []string) bool { return sentByAny(evt.Info, a) }) {
		return b.sendText(ctx, chat, "Com o consentimento ligado, só quem tá na relação pode terminar: !terminar @pessoa")
	}

	ended := relationshipsOf(all, aliases[0])
	if len(people) == 2 {
		ended = slices.DeleteFunc(ended, func(r relationship) bool { return !r.joins(aliases[0], aliases[1]) })
	}
	if len(ended) == 0 {
		return b.notTogether(ctx, chat, people)
	}
	if err := b.relationships.remove(ctx, ended); err != nil {
		return fmt.Errorf("apagar relacionamentos: %w", err)
	}

	for i := range ended {
		ended[i].partners = initiatorFirst(ended[i].partners, aliases[0])
	}
	text, mentioned := breakupText(ended, time.Now())
	marriages := consentedOnly(ofKind(ended, kindMarriage))
	text = appendLine(text, b.splitJointAccounts(ctx, chat, marriages))
	if payer := slices.IndexFunc(aliases, func(a []string) bool { return sentByAny(evt.Info, a) }); payer >= 0 {
		exes := otherPartners(marriages, aliases[payer], nil)
		text = appendLine(text, b.charge(ctx, chat, people[payer], exes, alimonyPercent, "⚖️ Pensão"))
		mentioned = uniqueJIDs(append(mentioned, people[payer]))
	}
	_, err := b.sendMentions(ctx, chat, text, mentioned)
	return err
}

func (b *bot) notTogether(ctx context.Context, chat types.JID, people []types.JID) error {
	if len(people) == 1 {
		return b.sendText(ctx, chat, "Você nem namora nem é casado(a) com ninguém 🤨 Usa !namorar ou !casar primeiro")
	}
	_, err := b.sendMentions(ctx, chat, fmt.Sprintf("@%s e @%s nem namoram nem são casados 🤨", people[0].User, people[1].User), people)
	return err
}

func initiatorFirst(pair [2]types.JID, initiator []string) [2]types.JID {
	if slices.Contains(initiator, pair[1].User) {
		return [2]types.JID{pair[1], pair[0]}
	}
	return pair
}

func breakupText(breakups []relationship, now time.Time) (string, []types.JID) {
	headers := map[[3]bool]string{
		{false, true, false}: "💔 *Fim do namoro*",
		{true, false, false}: "💔 *Divórcio*",
		{false, false, true}: "💔 *Fim do caso*",
	}
	templates := map[relationshipKind]string{
		kindDating:   "@%s e @%s terminaram depois de %s de namoro",
		kindMarriage: "@%s e @%s se separaram depois de %s de casamento",
		kindLover:    "@%s e @%s acabaram o caso depois de %s se pegando escondido",
	}
	aftermaths := map[relationshipKind][]string{
		kindDating: {
			"@{2} já tá soltando indireta no story 📱",
			"@{1} vai passar a noite ouvindo sofrência 😭",
			"@{2} ficou com o moletom de @{1} e não vai devolver 🧥",
			"Já tão devolvendo os presentes um por um 🎁",
			"Já tem gente no grupo de olho em @{1} 👀",
			"Os dois continuam se seguindo só pra stalkear o story 🙈",
		},
		kindMarriage: {
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
		kinds[0] = kinds[0] || br.kind == kindMarriage
		kinds[1] = kinds[1] || br.kind == kindDating
		kinds[2] = kinds[2] || br.kind == kindLover
		lines = append(lines, fmt.Sprintf(templates[br.kind], br.partners[0].User, br.partners[1].User, marriageLength(now.Sub(br.since))))
		mentioned = append(mentioned, br.partners[0], br.partners[1])
	}
	last := breakups[len(breakups)-1]
	pool := aftermaths[last.kind]
	aftermath := strings.NewReplacer("{1}", last.partners[0].User, "{2}", last.partners[1].User).Replace(pool[rand.IntN(len(pool))])
	text := cmp.Or(headers[kinds], "💔 *Fim de tudo*") + "\n\n" + strings.Join(lines, "\n") + "\n\n" + aftermath
	return text, uniqueJIDs(mentioned)
}
