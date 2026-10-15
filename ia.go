package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

const (
	providerUsesPerHour = 15
	localQueueSize      = 5
	localRest           = 10 * time.Second
	typingRefresh       = 10 * time.Second
	iaSystemPrompt      = `Você é o bot de um grupo de amigos no WhatsApp. Responda em português do Brasil, no tom do grupo: informal, zoeiro e direto, sem soar como assistente de empresa.
Seja curto, no máximo uns três parágrafos, a não ser que peçam mais.
Use só a formatação do WhatsApp: *negrito*, _itálico_ e listas com hífen. Nada de títulos em markdown, tabelas ou blocos de código.
As mensagens recentes do grupo vêm só como contexto. Responda a pergunta que está no fim.
Os nomes que aparecem são os apelidos que o grupo deu pra cada um. Chame as pessoas por esses apelidos, principalmente quem fez a pergunta.`
)

var (
	errQuotaUsed = errors.New("cota do provedor esgotada")
	errQueueFull = errors.New("fila do modelo local cheia")
)

type iaTier struct {
	name        string
	model       chatModel
	contextSize int
	maxTokens   int
	timeout     time.Duration
	admit       func(ctx context.Context, user string) (release func(ok bool), err error)
}

type assistant struct {
	tiers []iaTier
}

func newAssistantFromEnv() (*assistant, error) {
	var provider, local chatModel
	var providerErr, localErr error
	provider, providerErr = modelFromEnv("LLM_PROVIDER", modelSettings{api: "anthropic"})
	local, localErr = modelFromEnv("LLM_LOCAL", modelSettings{api: "openai", url: "http://localhost:11434/v1"})
	if err := errors.Join(providerErr, localErr); err != nil {
		return nil, err
	}

	a := &assistant{}
	if provider != nil {
		quota := &hourlyQuota{limit: providerUsesPerHour, uses: map[string][]time.Time{}}
		a.tiers = append(a.tiers, iaTier{
			name:        "provedor",
			model:       provider,
			contextSize: 30,
			maxTokens:   2048,
			timeout:     time.Minute,
			admit:       quota.admit,
		})
	}
	if local != nil {
		gate := &cpuGate{slot: make(chan struct{}, 1)}
		a.tiers = append(a.tiers, iaTier{
			name:        "local",
			model:       local,
			contextSize: 80,
			maxTokens:   4096,
			timeout:     5 * time.Minute,
			admit:       gate.admit,
		})
	}
	return a, nil
}

func (a *assistant) ask(ctx context.Context, user string, prompt func(contextSize int) string) (string, error) {
	var errs []error
	for _, tier := range a.tiers {
		answer, err := tier.ask(ctx, user, prompt)
		if err == nil {
			return answer, nil
		}
		errs = append(errs, fmt.Errorf("%s: %w", tier.name, err))
	}
	return "", errors.Join(errs...)
}

func (t iaTier) ask(ctx context.Context, user string, prompt func(contextSize int) string) (answer string, err error) {
	release, err := t.admit(ctx, user)
	if err != nil {
		return "", err
	}
	defer func() { release(err == nil) }()

	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()
	answer, err = t.model.reply(ctx, iaSystemPrompt, prompt(t.contextSize), t.maxTokens)
	if err != nil {
		return "", err
	}
	if answer == "" {
		return "", errors.New("resposta vazia")
	}
	return answer, nil
}

type hourlyQuota struct {
	limit int
	mu    sync.Mutex
	uses  map[string][]time.Time
}

func (q *hourlyQuota) admit(_ context.Context, user string) (func(ok bool), error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	cutoff := time.Now().Add(-time.Hour)
	recent := slices.DeleteFunc(q.uses[user], func(t time.Time) bool { return t.Before(cutoff) })
	if len(recent) >= q.limit {
		q.uses[user] = recent
		return nil, errQuotaUsed
	}
	used := time.Now()
	q.uses[user] = append(recent, used)
	return func(ok bool) {
		if ok {
			return
		}
		q.mu.Lock()
		defer q.mu.Unlock()
		q.uses[user] = slices.DeleteFunc(q.uses[user], func(t time.Time) bool { return t.Equal(used) })
	}, nil
}

type cpuGate struct {
	slot    chan struct{}
	waiting atomic.Int32
}

func (g *cpuGate) admit(ctx context.Context, _ string) (func(ok bool), error) {
	if g.waiting.Add(1) > localQueueSize {
		g.waiting.Add(-1)
		return nil, errQueueFull
	}
	defer g.waiting.Add(-1)
	select {
	case g.slot <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return func(bool) { time.AfterFunc(localRest, func() { <-g.slot }) }, nil
}

func (b *bot) ia(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat
	question := questionFrom(messageText(evt.Message))
	if question == "" {
		return b.sendText(ctx, chat, "🧠 Manda a pergunta junto: !ia <pergunta>")
	}
	if len(b.assistant.tiers) == 0 {
		return b.sendText(ctx, chat, "🧠 A IA tá desligada, ninguém configurou um modelo")
	}

	stopTyping := b.typing(ctx, chat)
	answer, err := b.assistant.ask(ctx, evt.Info.Sender.User, func(contextSize int) string {
		return b.iaPrompt(ctx, evt, question, contextSize)
	})
	stopTyping()
	if err != nil {
		b.client.Log.Warnf("!ia: %v", err)
		answer = iaFailure(err)
	}
	return b.replyTo(ctx, evt, answer)
}

func questionFrom(text string) string {
	text = strings.TrimSpace(text)
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(text, fields[0]))
}

func iaFailure(err error) string {
	if errors.Is(err, errQueueFull) {
		return "🥵 O PC tá fervendo com pergunta na fila, tenta de novo daqui a pouco"
	}
	if errors.Is(err, errQuotaUsed) {
		return fmt.Sprintf("⏳ Você já gastou suas %d perguntas dessa hora, espera um pouco", providerUsesPerHour)
	}
	return "😵 A IA bugou, tenta de novo"
}

func (b *bot) iaPrompt(ctx context.Context, evt *events.Message, question string, contextSize int) string {
	nicknames := b.chatNicknames(ctx, evt.Info.Chat)
	asker := b.displayName(ctx, evt.Info.Sender.ToNonAD(), nicknames)
	ask := fmt.Sprintf("Pergunta de %s: %s", asker, question)

	recent, err := b.messages.recent(ctx, evt.Info.Chat, evt.Info.ID, contextSize)
	if err != nil {
		b.client.Log.Warnf("!ia: ler mensagens recentes de %s: %v", evt.Info.Chat, err)
	}
	if len(recent) == 0 {
		return ask
	}

	names := map[storedSender]string{}
	lines := make([]string, 0, len(recent)+3)
	lines = append(lines, "Mensagens recentes do grupo:")
	for _, m := range recent {
		name, ok := names[m.sender]
		if !ok {
			name = b.senderName(ctx, m.sender, nicknames)
			names[m.sender] = name
		}
		lines = append(lines, fmt.Sprintf("%s: %s", name, m.text))
	}
	return strings.Join(append(lines, "", ask), "\n")
}

type storedSender struct {
	user string
	alt  string
}

type storedMessage struct {
	sender storedSender
	text   string
}

func (s *messageStore) recent(ctx context.Context, chat types.JID, skipID types.MessageID, limit int) (messages []storedMessage, err error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT remetente, remetente_alt, texto FROM mensagens
		 WHERE chat = ? AND id != ?
		 ORDER BY enviada_em DESC LIMIT ?`,
		chat.ToNonAD().String(), skipID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var m storedMessage
		if err := rows.Scan(&m.sender.user, &m.sender.alt, &m.text); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	slices.Reverse(messages)
	return messages, rows.Err()
}

func (b *bot) senderName(ctx context.Context, sender storedSender, nicknames map[string]string) string {
	users := slices.DeleteFunc([]string{sender.user, sender.alt}, func(u string) bool { return u == "" })
	for _, u := range users {
		if nickname, ok := nicknames[u]; ok {
			return nickname
		}
	}
	for _, u := range users {
		for _, server := range []string{types.DefaultUserServer, types.HiddenUserServer} {
			contact, err := b.client.Store.Contacts.GetContact(ctx, types.NewJID(u, server))
			if err != nil {
				continue
			}
			if name := cmp.Or(contact.PushName, contact.FullName, contact.FirstName); name != "" {
				return name
			}
		}
	}
	return sender.user
}

func (b *bot) typing(ctx context.Context, chat types.JID) (stop func()) {
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(typingRefresh)
		defer ticker.Stop()
		for {
			if err := b.client.SendChatPresence(ctx, chat, types.ChatPresenceComposing, types.ChatPresenceMediaText); err != nil && ctx.Err() == nil {
				b.client.Log.Warnf("mostrar digitando em %s: %v", chat, err)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return func() {
		cancel()
		<-done
		if err := b.client.SendChatPresence(context.WithoutCancel(ctx), chat, types.ChatPresencePaused, types.ChatPresenceMediaText); err != nil {
			b.client.Log.Warnf("parar digitando em %s: %v", chat, err)
		}
	}
}

func (b *bot) replyTo(ctx context.Context, evt *events.Message, text string) error {
	_, err := b.client.SendMessage(ctx, evt.Info.Chat, &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waE2E.ContextInfo{
				StanzaID:      proto.String(evt.Info.ID),
				Participant:   proto.String(evt.Info.Sender.ToNonAD().String()),
				QuotedMessage: evt.Message,
			},
		},
	})
	return err
}
