package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const ficiaMaxQuotes = 15

type ficiaCharacter struct {
	name     string
	realName string
	texts    []string
}

func (b *bot) ficia(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat
	if len(b.assistant.tiers) == 0 {
		return b.sendText(ctx, chat, "🧠 A IA tá desligada, ninguém configurou um modelo")
	}
	pair, err := b.ficPair(ctx, evt)
	if err != nil {
		return fmt.Errorf("escolher casal da !ficia: %w", err)
	}
	if len(pair) < 2 {
		return b.sendText(ctx, chat, "Marca uma ou duas pessoas: !ficia @pessoa ou !ficia @pessoa1 @pessoa2")
	}
	if rand.IntN(2) == 0 {
		pair[0], pair[1] = pair[1], pair[0]
	}

	cast, err := b.ficiaCast(ctx, chat, pair)
	if err != nil {
		return err
	}
	theme := nicknameFrom(messageText(evt.Message))
	stopThinking := b.thinking(ctx, evt)
	story, err := b.assistant.ask(ctx, b.iaAsker(ctx, chat, evt.Info.Sender.User), func(contextSize int) string {
		return ficiaPrompt(cast, theme, min(contextSize/2, ficiaMaxQuotes))
	})
	stopThinking()
	if err != nil {
		b.client.Log.Warnf("!ficia: %v", err)
		return b.replyTo(ctx, evt, iaFailure(err))
	}
	header := fmt.Sprintf("📖 _Uma fic de @%s e @%s, escrita pela IA_", pair[0].User, pair[1].User)
	_, err = b.sendMentions(ctx, chat, header+"\n\n"+stripTags(story), pair)
	return err
}

func (b *bot) ficiaCast(ctx context.Context, chat types.JID, pair []types.JID) ([]ficiaCharacter, error) {
	cast := make([]ficiaCharacter, len(pair))
	var realNames map[string]string
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		realNames = b.chatNames(gctx, chat)
		return nil
	})
	for i, person := range pair {
		g.Go(func() (err error) {
			cast[i].name, cast[i].texts, err = b.personTexts(gctx, chat, person.ToNonAD())
			return err
		})
	}
	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("ler o elenco da !ficia: %w", err)
	}
	for i, person := range pair {
		cast[i].realName = nicknameOf(b.identities(ctx, person.ToNonAD()), realNames)
	}
	return cast, nil
}

func ficiaPrompt(cast []ficiaCharacter, theme string, quotes int) string {
	parts := make([]string, 0, len(cast)+2)
	for _, c := range cast {
		parts = append(parts, c.describe(quotes))
	}
	setting := "Escolha você o cenário e o gênero (romance, comédia ou drama)."
	if theme != "" {
		setting = fmt.Sprintf("O tema ou cenário pedido é: %s.", theme)
	}
	parts = append(parts, fmt.Sprintf(`Escreve uma fic curtinha estrelando %s e %s. %s
Use os dois nomes acima pra chamar os personagens. Nos diálogos, cada um fala do jeito que escreve nas mensagens (gírias, manias, erros de digitação, emojis), e a história pode puxar os assuntos que eles costumam falar.
Comece com um título em negrito, divida em dois ou três capítulos curtos e termine com um final marcante ou um gancho.
Tamanho máximo: uns 1500 caracteres no total.
Romance, comédia e drama estão liberados, mas nada de conteúdo sexual explícito, e nada de preconceito (raça, religião, orientação sexual, gênero, deficiência, corpo).`, cast[0].name, cast[1].name, setting))
	return strings.Join(parts, "\n\n")
}

func (c ficiaCharacter) describe(quotes int) string {
	who := c.name
	if c.realName != "" && !strings.EqualFold(c.realName, c.name) {
		who = fmt.Sprintf("%s (nome de verdade: %s)", c.name, c.realName)
	}
	if len(c.texts) == 0 {
		return fmt.Sprintf("Personagem: %s. Quase nunca fala no grupo, então invente a personalidade.", who)
	}
	return "Personagem: " + who + "\n" + personPrompt(c.name, c.texts[:min(quotes, len(c.texts))], "")
}
