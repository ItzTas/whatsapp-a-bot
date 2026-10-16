package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"unicode/utf8"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const personQuoteMaxRunes = 300

func (b *bot) askAboutPerson(ctx context.Context, evt *events.Message, command string, request func(name string) string) error {
	chat := evt.Info.Chat
	if len(b.assistant.tiers) == 0 {
		return b.sendText(ctx, chat, "🧠 A IA tá desligada, ninguém configurou um modelo")
	}

	person := firstMentionOrSender(evt)
	name, texts, err := b.personTexts(ctx, chat, person)
	if err != nil {
		return err
	}
	if len(texts) == 0 {
		_, err = b.sendMentions(ctx, chat, fmt.Sprintf("Ainda não vi nenhuma mensagem de @%s aqui pra analisar 🤷", person.User), []types.JID{person})
		return err
	}

	stopThinking := b.thinking(ctx, evt)
	answer, err := b.assistant.ask(ctx, b.iaAsker(ctx, chat, evt.Info.Sender.User), func(contextSize int) string {
		return personPrompt(name, texts[:min(contextSize, len(texts))], request(name))
	})
	stopThinking()
	if err != nil {
		b.client.Log.Warnf("!%s: %v", command, err)
		answer = iaFailure(err)
	}
	return b.replyTo(ctx, evt, answer)
}

func (b *bot) personTexts(ctx context.Context, chat, person types.JID) (name string, texts []string, err error) {
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		name = b.displayName(gctx, person, b.chatNicknames(gctx, chat))
		return nil
	})
	g.Go(func() (err error) {
		texts, err = b.messages.texts(gctx, chat.ToNonAD(), b.userAliases(gctx, person))
		return err
	})
	if err := g.Wait(); err != nil {
		return "", nil, fmt.Errorf("ler mensagens de %s: %w", person, err)
	}
	texts = slices.DeleteFunc(texts, isCommand)
	rand.Shuffle(len(texts), func(i, j int) { texts[i], texts[j] = texts[j], texts[i] })
	return name, texts, nil
}

func personPrompt(name string, texts []string, request string) string {
	lines := make([]string, 0, len(texts)+3)
	lines = append(lines, fmt.Sprintf("Mensagens que %s mandou no grupo:", name))
	for _, t := range texts {
		lines = append(lines, "- "+truncateRunes(strings.ReplaceAll(t, "\n", " "), personQuoteMaxRunes))
	}
	return strings.Join(append(lines, "", request), "\n")
}

func truncateRunes(text string, limit int) string {
	if utf8.RuneCountInString(text) <= limit {
		return text
	}
	return string([]rune(text)[:limit]) + "…"
}

func (b *bot) roast(ctx context.Context, evt *events.Message) error {
	return b.askAboutPerson(ctx, evt, "roast", func(name string) string {
		return fmt.Sprintf(`Faz um roast de %s: zoa a pessoa pesado, no nível de amigo que se conhece há anos, usando o jeito que ela escreve, as manias e os assuntos das mensagens acima. Cite um ou dois trechos dela pra provar o ponto.
Pode exagerar e ser ácido, mas nada de preconceito (raça, religião, orientação sexual, gênero, deficiência, corpo) nem coisa que machuque de verdade.
Uns dois parágrafos curtos, terminando com uma frase de efeito.`, name)
	})
}

func (b *bot) signo(ctx context.Context, evt *events.Message) error {
	return b.askAboutPerson(ctx, evt, "signo", func(name string) string {
		return fmt.Sprintf(`Descobre o signo de %s só pelo jeito que a pessoa escreve nas mensagens acima.
Comece com o signo em negrito e o emoji dele, tipo *♏ Escorpião*.
Depois explique, no tom de uma astróloga zoeira de internet, quais mensagens entregaram o signo, citando trechinhos.
Termine com o ascendente e a lua, descobertos com a mesma lógica, em uma linha cada.`, name)
	})
}
