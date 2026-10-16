package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

func (b *bot) trair(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	pair, ok := coupleFrom(evt.Info.Sender, mentionedJIDs(evt.Message))
	if !ok {
		return b.sendText(ctx, chat, "Marca com quem vai ser a traição: !trair @pessoa ou !trair @pessoa1 @pessoa2")
	}
	cheater, lover := pair[0], pair[1]

	var (
		all     []marriage
		aliases [2][]string
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { all, err = b.marriages.inChat(gctx, chat); return err })
	for i, p := range pair {
		g.Go(func() error { aliases[i] = b.userAliases(gctx, p); return nil })
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler casamentos: %w", err)
	}

	spouses := otherPartners(marriagesOf(all, aliases[0]), aliases[0], nil)
	if len(spouses) == 0 {
		text := fmt.Sprintf("💅 @%s nem é casado(a), então ficar com @%s não é traição, é só pegação 😏", cheater.User, lover.User)
		_, err := b.sendMentions(ctx, chat, text, pair[:])
		return err
	}
	betrayed := slices.DeleteFunc(spouses, func(s types.JID) bool { return slices.Contains(aliases[1], s.User) })
	if len(betrayed) == 0 {
		text := fmt.Sprintf("🤨 @%s tentou trair @%s... com @%s mesmo. Isso tem nome: casamento 💍", cheater.User, lover.User, lover.User)
		_, err := b.sendMentions(ctx, chat, text, pair[:])
		return err
	}

	_, err := b.sendMentions(ctx, chat, betrayalText(cheater, lover, betrayed), append(pair[:], betrayed...))
	return err
}

type cheatingNews struct {
	lines    []string
	betrayed []types.JID
}

func (b *bot) cheatingNews(ctx context.Context, chat types.JID, people []types.JID) (cheatingNews, error) {
	var (
		marriages []marriage
		datings   []dating
		aliases   = make([][]string, len(people))
	)
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(16)
	g.Go(func() (err error) { marriages, err = b.marriages.inChat(gctx, chat); return err })
	g.Go(func() (err error) { datings, err = b.datings.inChat(gctx, chat); return err })
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
			{"é casado(a) com", otherPartners(marriagesOf(marriages, aliases[i]), aliases[i], nil)},
			{"namora", datingPartnersOf(datings, aliases[i])},
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
	lines := slices.Concat(n.lines, betrayalOutcome(betrayedTags), []string{fmt.Sprintf("🦌 Chifres novos pra %s", betrayedTags)})
	return "\n\n" + strings.Join(lines, "\n")
}

func betrayalText(cheater, lover types.JID, betrayed []types.JID) string {
	locais := []string{
		"no motel da esquina 🏨", "no banheiro da festa 🚽", "dentro do Uber 🚗", "na casa da sogra 🏠",
		"no churrasco da firma 🍖", "atrás da igreja ⛪", "no provador da Renner 👗", "na fila do show 🎤",
	}

	betrayedTags := joinMentions(betrayed)
	lines := []string{
		"🐍 *Traição*",
		"",
		fmt.Sprintf("@%s traiu %s com @%s %s", cheater.User, betrayedTags, lover.User, locais[rand.IntN(len(locais))]),
		"",
	}
	lines = append(lines, betrayalOutcome(betrayedTags)...)
	lines = append(lines, "", fmt.Sprintf("🦌 Chifres novos pra %s", betrayedTags))
	return strings.Join(lines, "\n")
}

func betrayalOutcome(betrayedTags string) []string {
	disfarces := []string{
		"🤫 Ninguém ficou sabendo... por enquanto",
		"😇 Voltou pra casa com cara de santo(a)",
		"🧼 Apagou as conversas e limpou o histórico",
		"🙈 Disse que tava na casa de um amigo",
	}
	if rand.IntN(2) == 0 {
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
