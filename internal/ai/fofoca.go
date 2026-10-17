package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	gossipChatLines   = 60
	gossipPersonQuote = 15
)

type gossipTarget struct {
	name  string
	texts []string
}

func (b *bot) fofoca(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat
	if !b.Assistant.Enabled() {
		return b.SendText(ctx, chat, "🧠 A IA tá desligada, ninguém configurou um modelo")
	}

	mentions := core.MentionedJIDs(evt.Message)
	var (
		lines  []chatLine
		target *gossipTarget
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) {
		lines, err = b.messages.chatLines(gctx, chat.ToNonAD(), "id != ? AND ltrim(texto) NOT LIKE '!%'", []any{evt.Info.ID}, false, gossipChatLines)
		return err
	})
	if len(mentions) > 0 {
		g.Go(func() error {
			name, texts, err := b.personTexts(gctx, chat, mentions[0].ToNonAD())
			target = &gossipTarget{name: name, texts: texts[:min(gossipPersonQuote, len(texts))]}
			return err
		})
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler mensagens pra !fofoca em %s: %w", chat, err)
	}
	if len(lines) == 0 {
		return b.SendText(ctx, chat, "☕ Ainda não tem mensagem guardada nesta conversa pra virar fofoca")
	}

	nicknames := b.ChatNicknames(ctx, chat)
	toolkit := b.iaToolkit(ctx, evt, nicknames)
	stopThinking := b.thinking(ctx, evt)
	answer, err := b.Assistant.Ask(ctx, b.Asker(ctx, chat, evt.Info.Sender.User), func(contextSize int) string {
		recent := lines[max(0, len(lines)-2*contextSize):]
		return gossipPrompt(b.renderChatLines(ctx, recent, nicknames), target)
	})
	stopThinking()
	if err != nil {
		b.Client.Log.Warnf("!fofoca: %v", err)
		return b.ReplyTo(ctx, evt, iaFailure(err))
	}
	text, tagged := toolkit.resolveMentions(stripTags(answer))
	return b.ReplyMentioning(ctx, evt, text, tagged)
}

func gossipPrompt(conversation string, target *gossipTarget) string {
	parts := []string{"<conversa>", conversation, "</conversa>", ""}
	focus := "envolvendo duas ou três pessoas que aparecem na <conversa>"
	if target != nil {
		parts = append(parts, target.context())
		focus = fmt.Sprintf("com %s como protagonista do babado, puxando gente da <conversa> como coadjuvante", target.name)
	}
	parts = append(parts, fmt.Sprintf(`Você é colunista de fofoca de revista de celebridade, só que do grupo. Inventa um babado %s, usando os assuntos e o jeito de falar das mensagens acima pra dar credibilidade (pode citar trechinhos tirados de contexto).
Comece com uma manchete em negrito e escreva uns dois parágrafos curtos no tom de colunista: "fontes próximas", "um passarinho me contou", "não confirmamos mas também não desmentimos".
Deixe claro que é tudo inventado, tipo citando "fontes não confiáveis" ou terminando com um aviso de que a coluna não checou nada.
Quando citar alguém do grupo, escreva @ seguido do nome exatamente como aparece nas mensagens (tipo @Fulano), assim a pessoa é marcada.
É zoeira entre amigos: pode ser maldoso e exagerado, mas nada de conteúdo sexual explícito, preconceito (raça, religião, orientação sexual, gênero, deficiência, corpo) nem acusação de crime de verdade.`, focus))
	return strings.Join(parts, "\n")
}

func (t *gossipTarget) context() string {
	if len(t.texts) == 0 {
		return t.name + " quase nunca fala no grupo, o que por si só já é suspeito.\n"
	}
	return personPrompt(t.name, t.texts, "")
}
