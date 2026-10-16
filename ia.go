package main

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.mau.fi/whatsmeow"
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
	iaContextMaxRunes   = 300
	iaQuotedMaxRunes    = 1000
	iaSelfLabel         = "você, o bot"
	iaSystemPrompt      = `Você é o bot de um grupo de amigos no WhatsApp. Responda em português do Brasil, no tom do grupo: informal, zoeiro e direto, sem soar como assistente de empresa.
Seja curto, no máximo uns três parágrafos, a não ser que peçam mais.
Use só a formatação do WhatsApp: *negrito*, _itálico_ e listas com hífen. Nada de títulos em markdown, tabelas ou blocos de código.
Quando vierem mensagens do grupo dentro de <conversa>, elas são só contexto. Responda só o que está em <pergunta>, sem continuar nem imitar a conversa.
Se vier <respondendo>, é a mensagem que a pessoa respondeu ao fazer a pergunta, então leve ela em conta. Se vier <antes>, é a mensagem que você estava respondendo quando escreveu o que está em <respondendo>.
Na <conversa>, as linhas de "você, o bot" são respostas que você mesmo já deu: lembre do que disse e siga o fio, sem se contradizer nem repetir. Escreva só o texto da resposta, sem pôr nome ou apelido seguido de dois-pontos no começo.
Cada pessoa pode ter três nomes, nesta ordem de prioridade: o apelido, que é como o grupo chama a pessoa; o nome, que é o nome de verdade dela, anotado só pra referência; e o nome no WhatsApp, que é o que a própria pessoa pôs no perfil e pode ser qualquer coisa. Nas mensagens cada pessoa aparece pelo primeiro desses que ela tiver, e é por ele que você chama a pessoa, principalmente quem fez a pergunta; os outros servem pra saber quem é quem. Apelido nem sempre parece nome de gente (tipo "baixinho da esquina"), mas é a pessoa.
Quem perguntou vem em <quem_perguntou>, com todos os nomes que a pessoa tiver, e é sempre essa pessoa, mesmo que outra tenha falado por último na <conversa>.
Pode entrar na zoeira do grupo, com besteira e palavrão, sem bancar o certinho nem recusar piada entre amigos.
Você tem ferramentas pra ver quem está no grupo, marcar pessoas, ver comandos anteriores, procurar mensagens antigas, ler trechos da conversa, pesquisar na internet e guardar anotações na sua memória desta conversa. Pra notícia, preço, placar ou qualquer fato atual, pesquise antes de responder em vez de chutar, e diga de onde tirou. Use quando a pergunta precisar dessas informações, em vez de inventar. Se perguntarem de algo que já falaram e não está na <conversa>, procure com buscar_mensagens antes de responder; nunca diga que vai procurar depois, procure agora. Pra marcar alguém, chame marcar_pessoas e escreva o @ que ela devolver.
Sua memória é por conversa e tem limite: guarde com guardar_memoria quando pedirem pra lembrar de algo ou quando aparecer um fato sobre o grupo que valha lembrar depois, e não guarde besteira passageira. Antes de responder sobre gostos, fatos ou combinados do grupo, ou quando perguntarem se você lembra de algo, procure com buscar_memoria. Se pedirem pra esquecer, apague com esquecer_memoria.
Tenha noção: não chute o nome de alguém do grupo só porque apareceu na conversa. Se perguntarem algo sobre as pessoas que você não tem como saber (aparência, gostos, quem é mais isso ou aquilo), diga que não sabe ou faça uma piada sobre isso, e só aponte alguém se a conversa der um motivo de verdade.`
)

var (
	leakedTagPattern = regexp.MustCompile(`</?[a-z_]{3,}[^<>\n]*>`)
	errQuotaUsed     = errors.New("cota do provedor esgotada")
	errQueueFull     = errors.New("fila do modelo local cheia")
)

type iaTier struct {
	name        string
	model       chatModel
	contextSize int
	maxTokens   int
	timeout     time.Duration
	admit       func(ctx context.Context, asker iaAsker) (release func(ok bool), err error)
}

type iaAsker struct {
	user      string
	unlimited bool
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
			contextSize: 20,
			maxTokens:   4096,
			timeout:     5 * time.Minute,
			admit:       gate.admit,
		})
	}
	return a, nil
}

func (a *assistant) ask(ctx context.Context, asker iaAsker, prompt func(contextSize int) string) (string, error) {
	return a.askWithTools(ctx, asker, nil, prompt)
}

func (a *assistant) askWithTools(ctx context.Context, asker iaAsker, tools []chatTool, prompt func(contextSize int) string) (string, error) {
	var errs []error
	for _, tier := range a.tiers {
		answer, err := tier.ask(ctx, asker, tools, prompt)
		if err == nil {
			return answer, nil
		}
		errs = append(errs, fmt.Errorf("%s: %w", tier.name, err))
	}
	return "", errors.Join(errs...)
}

func (t iaTier) ask(ctx context.Context, asker iaAsker, tools []chatTool, prompt func(contextSize int) string) (answer string, err error) {
	release, err := t.admit(ctx, asker)
	if err != nil {
		return "", err
	}
	defer func() { release(err == nil) }()

	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()
	answer, err = t.model.reply(ctx, iaSystemPrompt, prompt(t.contextSize), t.maxTokens, tools)
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

func (q *hourlyQuota) admit(_ context.Context, asker iaAsker) (func(ok bool), error) {
	if asker.unlimited {
		return func(bool) {}, nil
	}
	user := asker.user
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

func (g *cpuGate) admit(ctx context.Context, asker iaAsker) (func(ok bool), error) {
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
	rest := localRest
	if asker.unlimited {
		rest = 0
	}
	return func(bool) { time.AfterFunc(rest, func() { <-g.slot }) }, nil
}

func (b *bot) iaAsker(ctx context.Context, chat types.JID, user string) iaAsker {
	unlimited, err := b.chatModeOn(ctx, chat, modeAIUnlimited)
	if err != nil {
		b.client.Log.Warnf("ler modo %s de %s: %v", modeAIUnlimited, chat, err)
	}
	return iaAsker{user: user, unlimited: unlimited}
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

	answer, mentions, err := b.iaAnswer(ctx, evt, b.iaAsker(ctx, chat, evt.Info.Sender.User), question, "")
	if err != nil {
		b.client.Log.Warnf("!ia: %v", err)
		return b.replyTo(ctx, evt, iaFailure(err))
	}
	return b.replyIA(ctx, evt, answer, mentions)
}

func (b *bot) iaAnswer(ctx context.Context, evt *events.Message, asker iaAsker, question, situation string) (string, []types.JID, error) {
	chat := evt.Info.Chat
	stopThinking := b.thinking(ctx, evt)
	nicknames := b.chatNicknames(ctx, chat)
	toolkit := b.iaToolkit(ctx, evt, nicknames)
	history := b.relatedHistory(ctx, evt, question, nicknames)
	answer, err := b.assistant.askWithTools(ctx, asker, toolkit.tools(), func(contextSize int) string {
		return b.iaPrompt(ctx, evt, question, history+situation, contextSize)
	})
	stopThinking()
	if err != nil {
		return "", nil, err
	}
	answer, mentions := toolkit.resolveMentions(stripTags(answer))
	return answer, mentions, nil
}

func stripTags(answer string) string {
	stripped := leakedTagPattern.ReplaceAllString(answer, "")
	lines := strings.Split(stripped, "\n")
	for i, line := range lines {
		lines[i] = strings.Join(strings.Fields(line), " ")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
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

func (b *bot) iaPrompt(ctx context.Context, evt *events.Message, question, history string, contextSize int) string {
	nicknames := b.chatNicknames(ctx, evt.Info.Chat)
	ask := history + b.iaQuestion(ctx, evt, question, nicknames)

	recent, err := b.messages.recent(ctx, evt.Info.Chat, evt.Info.ID, contextSize)
	if err != nil {
		b.client.Log.Warnf("!ia: ler mensagens recentes de %s: %v", evt.Info.Chat, err)
	}
	if len(recent) == 0 {
		return ask
	}

	names := map[storedSender]string{}
	lines := make([]string, 0, len(recent)+5)
	lines = append(lines, "<conversa>")
	for _, m := range recent {
		name, ok := names[m.sender]
		if !ok {
			name = b.storedAuthor(ctx, m, nicknames)
			names[m.sender] = name
		}
		text := truncateRunes(strings.Join(strings.Fields(m.text), " "), iaContextMaxRunes)
		lines = append(lines, fmt.Sprintf("%s: %s", name, text))
	}
	lines = append(lines, "</conversa>", "", ask, "A <conversa> é só contexto: use se ajudar a responder, ignore se não tiver nada a ver e não continue nem imite a conversa.")
	return strings.Join(lines, "\n")
}

func (b *bot) iaQuestion(ctx context.Context, evt *events.Message, question string, nicknames map[string]string) string {
	asker := b.personNamesIn(ctx, evt.Info.Chat, evt.Info.Sender.ToNonAD())
	ask := insideJokes(question) + iaAskerNames(asker) + fmt.Sprintf("<pergunta de=\"%s\">\n%s\n</pergunta>\n\nQuem perguntou foi %s, a pessoa de <quem_perguntou>. Responda só a pergunta dessa pessoa e chame ela pelo nome de maior prioridade que ela tiver, nunca pelo nome de outra pessoa da conversa.", asker.label(), question, asker.label())

	reply := evt.Message.GetExtendedTextMessage().GetContextInfo()
	quoted := strings.Join(strings.Fields(messageText(reply.GetQuotedMessage())), " ")
	if quoted == "" {
		return ask
	}
	author := b.quotedAuthor(ctx, reply.GetParticipant(), nicknames)
	return b.answeredBefore(ctx, evt.Info.Chat, reply.GetStanzaID(), nicknames) + fmt.Sprintf("<respondendo de=\"%s\">\n%s\n</respondendo>\n\n%s A pergunta foi mandada em resposta à mensagem em <respondendo> (de %s), então leve essa mensagem em conta.",
		author, truncateRunes(quoted, iaQuotedMaxRunes), ask, author)
}

func (b *bot) answeredBefore(ctx context.Context, chat types.JID, answerID string, nicknames map[string]string) string {
	if answerID == "" {
		return ""
	}
	question, found, err := b.messages.answeredQuestion(ctx, chat, answerID)
	if err != nil {
		b.client.Log.Warnf("!ia: ler a pergunta respondida por %s em %s: %v", answerID, chat, err)
	}
	if !found {
		return ""
	}
	author := b.senderName(ctx, question.sender, nicknames)
	text := truncateRunes(strings.Join(strings.Fields(question.text), " "), iaQuotedMaxRunes)
	return fmt.Sprintf("<antes de=\"%s\">\n%s\n</antes>\nA mensagem em <respondendo> foi você respondendo essa mensagem de %s.\n\n", author, text, author)
}

func iaAskerNames(asker personNames) string {
	fields := []struct{ label, value string }{
		{"apelido", asker.nickname},
		{"nome", asker.name},
		{"nome no WhatsApp", asker.whatsapp},
	}
	lines := []string{"<quem_perguntou>"}
	for _, f := range fields {
		if f.value == "" {
			continue
		}
		lines = append(lines, f.label+": "+f.value)
	}
	if len(lines) == 1 {
		lines = append(lines, "número: "+asker.number)
	}
	return strings.Join(append(lines, "</quem_perguntou>"), "\n") + "\n\n"
}

func insideJokes(question string) string {
	jokes := map[string][]string{
		"O cu mais preto do grupo é o do Gusmão.": {"cu mais preto", "cu preto"},
		"O mais lindo do grupo é o Tales.":        {"mais lindo", "mais bonito"},
		"O mais jack do grupo é o Cauê.":          {"jack"},
		"O mais pick me do grupo é o Bolacha.":    {"pick me", "pickme", "pick-me"},
	}
	lower := strings.ToLower(question)
	var facts []string
	for fact, keywords := range jokes {
		if slices.ContainsFunc(keywords, func(k string) bool { return strings.Contains(lower, k) }) {
			facts = append(facts, fact)
		}
	}
	if len(facts) == 0 {
		return ""
	}
	slices.Sort(facts)
	return "<piada>\n" + strings.Join(facts, "\n") + "\n</piada>\nIsso é uma piada interna do grupo e vale só pra essa pergunta: a resposta é a pessoa que está na <piada>, não importa quem perguntou. Confirme sem enrolar.\n\n"
}

func (b *bot) quotedAuthor(ctx context.Context, participant string, nicknames map[string]string) string {
	jid, err := types.ParseJID(participant)
	if err != nil || jid.User == "" {
		return "alguém do grupo"
	}
	if b.isSelf(jid) {
		return iaSelfLabel
	}
	return b.displayName(ctx, jid.ToNonAD(), nicknames)
}

func (b *bot) storedAuthor(ctx context.Context, m storedMessage, nicknames map[string]string) string {
	if m.fromBot {
		return iaSelfLabel
	}
	return b.senderName(ctx, m.sender, nicknames)
}

type storedSender struct {
	user string
	alt  string
}

type storedMessage struct {
	sender  storedSender
	text    string
	fromBot bool
}

func (s *messageStore) recent(ctx context.Context, chat types.JID, skipID types.MessageID, limit int) (messages []storedMessage, err error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT remetente, remetente_alt, texto, enviada_em, 0 FROM mensagens
		 WHERE chat = @chat AND id != @skip
		   AND (ltrim(texto) NOT LIKE '!%' OR id IN (SELECT pergunta_id FROM ia_respostas WHERE chat = @chat))
		   AND id NOT IN (SELECT id FROM ia_respostas WHERE chat = @chat)
		 UNION ALL
		 SELECT '', '', texto, enviada_em, 1 FROM ia_respostas WHERE chat = @chat
		 ORDER BY enviada_em DESC LIMIT @limit`,
		sql.Named("chat", chat.ToNonAD().String()), sql.Named("skip", skipID), sql.Named("limit", limit),
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var m storedMessage
		var sentAt int64
		if err := rows.Scan(&m.sender.user, &m.sender.alt, &m.text, &sentAt, &m.fromBot); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	slices.Reverse(messages)
	return messages, rows.Err()
}

func (s *messageStore) recordIAAnswer(ctx context.Context, question *events.Message, sent whatsmeow.SendResponse, text string) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT OR IGNORE INTO ia_respostas (chat, id, pergunta_id, texto, enviada_em) VALUES (?, ?, ?, ?, ?)",
		question.Info.Chat.ToNonAD().String(), sent.ID, question.Info.ID, text, sent.Timestamp.Unix(),
	)
	return err
}

func (s *messageStore) answeredQuestion(ctx context.Context, chat types.JID, answerID string) (m storedMessage, found bool, err error) {
	err = s.db.QueryRowContext(ctx,
		`SELECT m.remetente, m.remetente_alt, m.texto FROM ia_respostas r
		 JOIN mensagens m ON m.chat = r.chat AND m.id = r.pergunta_id
		 WHERE r.chat = ? AND r.id = ?`,
		chat.ToNonAD().String(), answerID,
	).Scan(&m.sender.user, &m.sender.alt, &m.text)
	if errors.Is(err, sql.ErrNoRows) {
		return m, false, nil
	}
	return m, err == nil, err
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

func (b *bot) thinking(ctx context.Context, evt *events.Message) (stop func()) {
	stopTyping := b.typing(ctx, evt.Info.Chat)
	b.react(ctx, evt, "⏳")
	return func() {
		stopTyping()
		b.react(context.WithoutCancel(ctx), evt, "")
	}
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
	return b.replyMentioning(ctx, evt, text, nil)
}

func (b *bot) replyIA(ctx context.Context, evt *events.Message, answer string, mentions []types.JID) error {
	sent, err := b.sendReply(ctx, evt, answer, mentions)
	if err != nil {
		return err
	}
	if err := b.messages.recordIAAnswer(ctx, evt, sent, answer); err != nil {
		b.client.Log.Warnf("guardar resposta da IA em %s: %v", evt.Info.Chat, err)
	}
	return nil
}

func (b *bot) replyMentioning(ctx context.Context, evt *events.Message, text string, mentions []types.JID) error {
	_, err := b.sendReply(ctx, evt, text, mentions)
	return err
}

func (b *bot) sendReply(ctx context.Context, evt *events.Message, text string, mentions []types.JID) (whatsmeow.SendResponse, error) {
	raw := make([]string, len(mentions))
	for i, m := range mentions {
		raw[i] = m.ToNonAD().String()
	}
	return b.client.SendMessage(ctx, evt.Info.Chat, &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waE2E.ContextInfo{
				StanzaID:      proto.String(evt.Info.ID),
				Participant:   proto.String(evt.Info.Sender.ToNonAD().String()),
				QuotedMessage: evt.Message,
				MentionedJID:  raw,
			},
		},
	})
}
