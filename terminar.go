package main

import (
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

	var all []relationship
	aliases := make([][]string, len(people))
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) {
		all, err = b.relationships.inChat(gctx, chat, kindDating, kindMarriage)
		return err
	})
	for i, p := range people {
		g.Go(func() error { aliases[i] = b.userAliases(gctx, p); return nil })
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler relacionamentos: %w", err)
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
	headers := map[[2]bool]string{
		{false, true}: "💔 *Fim do namoro*",
		{true, false}: "💔 *Divórcio*",
		{true, true}:  "💔 *Fim de tudo*",
	}
	templates := map[relationshipKind]string{
		kindDating:   "@%s e @%s terminaram depois de %s de namoro",
		kindMarriage: "@%s e @%s se separaram depois de %s de casamento",
	}
	aftermaths := map[relationshipKind][]string{
		kindDating: {
			"@%[2]s já tá soltando indireta no story 📱",
			"@%[1]s vai passar a noite ouvindo sofrência 😭",
			"@%[2]s ficou com o moletom de @%[1]s e não vai devolver 🧥",
			"Já tão devolvendo os presentes um por um 🎁",
			"Já tem gente no grupo de olho em @%[1]s 👀",
			"Os dois continuam se seguindo só pra stalkear o story 🙈",
		},
		kindMarriage: {
			"A guarda do cachorro ficou com @%[2]s 🐶",
			"@%[1]s ficou com a casa e @%[2]s ficou com as dívidas 💸",
			"@%[2]s levou até a air fryer 🍟",
			"A pensão vai ser paga em figurinha 🃏",
			"@%[1]s já tá no Tinder de novo 📱",
			"Os dois vão fingir que não se conhecem no próximo rolê 🙈",
		},
	}

	var (
		kinds     [2]bool
		lines     []string
		mentioned []types.JID
	)
	for _, br := range breakups {
		kinds[0] = kinds[0] || br.kind == kindMarriage
		kinds[1] = kinds[1] || br.kind == kindDating
		lines = append(lines, fmt.Sprintf(templates[br.kind], br.partners[0].User, br.partners[1].User, marriageLength(now.Sub(br.since))))
		mentioned = append(mentioned, br.partners[0], br.partners[1])
	}
	last := breakups[len(breakups)-1]
	pool := aftermaths[last.kind]
	aftermath := fmt.Sprintf(pool[rand.IntN(len(pool))], last.partners[0].User, last.partners[1].User)
	text := headers[kinds] + "\n\n" + strings.Join(lines, "\n") + "\n\n" + aftermath
	return text, uniqueJIDs(mentioned)
}
