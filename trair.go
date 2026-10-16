package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

func (b *bot) trair(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	pair, ok := coupleFrom(evt.Info.Sender, mentionedJIDs(evt.Message))
	if !ok {
		return b.sendText(ctx, chat, "Marca quem vai ser traído(a): !trair @pessoa ou !trair @pessoa1 @pessoa2")
	}
	cheater, betrayed := pair[0], pair[1]

	var (
		aliases [2][]string
		wg      sync.WaitGroup
	)
	for i, p := range pair {
		wg.Go(func() { aliases[i] = b.userAliases(ctx, p) })
	}
	wg.Wait()

	if slices.ContainsFunc(aliases[0], func(a string) bool { return slices.Contains(aliases[1], a) }) {
		return b.sendText(ctx, chat, "Trair a si mesmo(a) não dá 🪞 Marca outra pessoa")
	}

	accomplices := []string{
		"com o(a) ex", "com um match do Tinder", "com o(a) vizinho(a)", "com o(a) melhor amigo(a)",
		"com o(a) personal trainer", "com o motoboy do iFood", "com o(a) colega de trabalho", "com o(a) crush da escola",
	}
	deed := fmt.Sprintf("@%s traiu @%s %s %s", cheater.User, betrayed.User, accomplices[rand.IntN(len(accomplices))], randomCheatingPlace())
	caught := rand.IntN(2) == 0
	text := betrayalText(deed, []types.JID{betrayed}, caught)
	if caught && sentByAny(evt.Info, aliases[0]) {
		text = appendLine(text, b.charge(ctx, chat, cheater, []types.JID{betrayed}, cheatingFinePercent, "⚖️ Indenização"))
	}
	return b.sendGIF(ctx, chat, "trair", text, pair[:])
}

func (b *bot) traircom(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	pair, ok := coupleFrom(evt.Info.Sender, mentionedJIDs(evt.Message))
	if !ok {
		return b.sendText(ctx, chat, "Marca com quem vai ser a traição: !traircom @pessoa ou !traircom @pessoa1 @pessoa2")
	}
	cheater, lover := pair[0], pair[1]

	var (
		all       []relationship
		adoptions []adoption
		aliases   [2][]string
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) {
		all, err = b.relationships.inChat(gctx, chat, kindMarriage, kindDating, kindLover)
		return err
	})
	g.Go(func() (err error) { adoptions, err = b.adoptions.inChat(gctx, chat); return err })
	for i, p := range pair {
		g.Go(func() error { aliases[i] = b.userAliases(gctx, p); return nil })
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler relacionamentos: %w", err)
	}

	if slices.ContainsFunc(aliases[0], func(a string) bool { return slices.Contains(aliases[1], a) }) {
		return b.sendText(ctx, chat, "Trair consigo mesmo(a) não vale 🪞 Marca outra pessoa")
	}
	if child, parent, ok := adoptiveBond(adoptions, pair, aliases); ok {
		_, err := b.sendMentions(ctx, chat, adoptiveBondText(child, parent, "virar amante"), pair[:])
		return err
	}
	partners := otherPartners(slices.Concat(ofKind(all, kindMarriage), ofKind(all, kindDating)), aliases[0], nil)
	if len(partners) == 0 {
		text := fmt.Sprintf("💅 @%s nem namora nem é casado(a), então ficar com @%s não é traição, é só pegação 😏", cheater.User, lover.User)
		_, err := b.sendMentions(ctx, chat, text, pair[:])
		return err
	}
	betrayed := slices.DeleteFunc(partners, func(s types.JID) bool { return slices.Contains(aliases[1], s.User) })
	if len(betrayed) == 0 {
		text := fmt.Sprintf("🤨 @%s tentou trair @%s... com @%s mesmo. Isso tem nome: relacionamento sério 💍", cheater.User, lover.User, lover.User)
		_, err := b.sendMentions(ctx, chat, text, pair[:])
		return err
	}

	affair, err := b.startAffair(ctx, chat, all, pair, aliases, time.Now())
	if err != nil {
		return fmt.Errorf("salvar amantes: %w", err)
	}
	caught := rand.IntN(2) == 0
	deed := fmt.Sprintf("@%s traiu %s com @%s %s", cheater.User, joinMentions(betrayed), lover.User, randomCheatingPlace())
	text := appendLine(betrayalText(deed, betrayed, caught), affair)
	if caught && sentByAny(evt.Info, aliases[0]) {
		text = appendLine(text, b.charge(ctx, chat, cheater, betrayed, cheatingFinePercent, "⚖️ Indenização"))
	}
	return b.sendGIF(ctx, chat, "trair", text, append(pair[:], betrayed...))
}

func (b *bot) startAffair(ctx context.Context, chat types.JID, all []relationship, pair [2]types.JID, aliases [2][]string, now time.Time) (string, error) {
	together := func(r relationship) bool { return r.joins(aliases[0], aliases[1]) }
	lovers := ofKind(all, kindLover)
	if i := slices.IndexFunc(lovers, together); i >= 0 {
		return fmt.Sprintf("💋 O caso de @%s e @%s já dura %s", pair[0].User, pair[1].User, marriageLength(now.Sub(lovers[i].since))), nil
	}
	if slices.ContainsFunc(all, together) {
		return "", nil
	}
	if err := b.relationships.add(ctx, chat, kindLover, pair, now, false); err != nil {
		return "", err
	}
	return fmt.Sprintf("💋 Agora @%s e @%s são amantes", pair[0].User, pair[1].User), nil
}

type cheatingNews struct {
	lines    []string
	betrayed []types.JID
}

func (b *bot) cheatingNews(ctx context.Context, chat types.JID, people []types.JID) (cheatingNews, error) {
	var (
		all     []relationship
		aliases = make([][]string, len(people))
	)
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(16)
	g.Go(func() (err error) {
		all, err = b.relationships.inChat(gctx, chat, kindMarriage, kindDating)
		return err
	})
	for i, p := range people {
		g.Go(func() error { aliases[i] = b.userAliases(gctx, p); return nil })
	}
	if err := g.Wait(); err != nil {
		return cheatingNews{}, err
	}

	present := slices.Concat(aliases...)
	var news cheatingNews
	for i, p := range people {
		relations := []struct {
			verb     string
			partners []types.JID
		}{
			{"é casado(a) com", otherPartners(ofKind(all, kindMarriage), aliases[i], nil)},
			{"namora", otherPartners(ofKind(all, kindDating), aliases[i], nil)},
		}
		for _, r := range relations {
			betrayed := slices.DeleteFunc(r.partners, func(s types.JID) bool { return slices.Contains(present, s.User) })
			if len(betrayed) == 0 {
				continue
			}
			news.lines = append(news.lines, fmt.Sprintf("🐍 *Traição!* @%s %s %s", p.User, r.verb, joinMentions(betrayed)))
			news.betrayed = append(news.betrayed, betrayed...)
		}
	}
	news.betrayed = uniqueJIDs(news.betrayed)
	return news, nil
}

func (n cheatingNews) caption() string {
	if len(n.betrayed) == 0 {
		return ""
	}
	betrayedTags := joinMentions(n.betrayed)
	lines := slices.Concat(n.lines, betrayalOutcome(betrayedTags, rand.IntN(2) == 0), []string{fmt.Sprintf("🦌 Chifres novos pra %s", betrayedTags)})
	return "\n\n" + strings.Join(lines, "\n")
}

func randomCheatingPlace() string {
	places := []string{
		"no motel da esquina 🏨", "no banheiro da festa 🚽", "dentro do Uber 🚗", "na casa da sogra 🏠",
		"no churrasco da firma 🍖", "atrás da igreja ⛪", "no provador da Renner 👗", "na fila do show 🎤",
	}
	return places[rand.IntN(len(places))]
}

func betrayalText(deed string, betrayed []types.JID, caught bool) string {
	betrayedTags := joinMentions(betrayed)
	lines := []string{"🐍 *Traição*", "", deed, ""}
	lines = append(lines, betrayalOutcome(betrayedTags, caught)...)
	lines = append(lines, "", fmt.Sprintf("🦌 Chifres novos pra %s", betrayedTags))
	return strings.Join(lines, "\n")
}

func betrayalOutcome(betrayedTags string, caught bool) []string {
	disfarces := []string{
		"🤫 Ninguém ficou sabendo... por enquanto",
		"😇 Voltou pra casa com cara de santo(a)",
		"🧼 Apagou as conversas e limpou o histórico",
		"🙈 Disse que tava na casa de um amigo",
	}
	if !caught {
		return []string{disfarces[rand.IntN(len(disfarces))]}
	}

	flagras := []string{
		"🚨 *FLAGRA!* %s chegou na hora e viu tudo",
		"📸 *FLAGRA!* Mandaram a foto pra %s no privado",
		"📱 *FLAGRA!* %s viu a notificação aparecendo na tela",
		"🕵️ *FLAGRA!* %s contratou detetive e descobriu tudo",
	}
	vingancas := []string{
		"🧳 As roupas já foram jogadas pela janela",
		"🔪 Já tá afiando a faca",
		"📢 Vai expor tudo no story",
		"🚗 Riscou o carro inteiro com a chave",
		"😭 Chorou ouvindo sofrência a noite toda",
		"🫡 Fingiu que não viu e seguiu a vida",
	}
	return []string{
		fmt.Sprintf(flagras[rand.IntN(len(flagras))], betrayedTags),
		vingancas[rand.IntN(len(vingancas))],
	}
}
