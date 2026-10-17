package ai

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/llm"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	iaContextMaxRunes = 300
	iaQuotedMaxRunes  = 1000
	iaSelfLabel       = "você, o bot"
)

var (
	leakedTagPattern = regexp.MustCompile(`</?[a-z_]{3,}[^<>\n]*>`)
)

func (b *bot) ia(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat
	question := core.QuestionFrom(core.MessageText(evt.Message))
	if question == "" {
		return b.SendText(ctx, chat, "🧠 Manda a pergunta junto: !ia <pergunta>")
	}
	if !b.Assistant.Enabled() {
		return b.SendText(ctx, chat, "🧠 A IA tá desligada, ninguém configurou um modelo")
	}

	answer, mentions, err := b.iaAnswer(ctx, evt, b.Asker(ctx, chat, evt.Info.Sender.User), question, "")
	if err != nil {
		b.Client.Log.Warnf("!ia: %v", err)
		return b.ReplyTo(ctx, evt, iaFailure(err))
	}
	return b.replyIA(ctx, evt, answer, mentions)
}

func (b *bot) iaAnswer(ctx context.Context, evt *events.Message, asker llm.Asker, question, situation string) (string, []types.JID, error) {
	chat := evt.Info.Chat
	stopThinking := b.thinking(ctx, evt)
	nicknames := b.ChatNicknames(ctx, chat)
	toolkit := b.iaToolkit(ctx, evt, nicknames)
	history := b.relatedHistory(ctx, evt, question, nicknames)
	answer, err := b.Assistant.AskWithTools(ctx, asker, toolkit.tools(), func(contextSize int) string {
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

func iaFailure(err error) string {
	if errors.Is(err, llm.ErrQueueFull) {
		return "🥵 O PC tá fervendo com pergunta na fila, tenta de novo daqui a pouco"
	}
	if errors.Is(err, llm.ErrQuotaUsed) {
		return fmt.Sprintf("⏳ Você já gastou suas %d perguntas dessa hora, espera um pouco", llm.ProviderUsesPerHour)
	}
	return "😵 A IA bugou, tenta de novo"
}

func (b *bot) iaPrompt(ctx context.Context, evt *events.Message, question, history string, contextSize int) string {
	nicknames := b.ChatNicknames(ctx, evt.Info.Chat)
	ask := history + b.iaQuestion(ctx, evt, question, nicknames)

	recent, err := b.messages.recent(ctx, evt.Info.Chat, evt.Info.ID, contextSize)
	if err != nil {
		b.Client.Log.Warnf("!ia: ler mensagens recentes de %s: %v", evt.Info.Chat, err)
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
		text := core.TruncateRunes(strings.Join(strings.Fields(m.text), " "), iaContextMaxRunes)
		lines = append(lines, fmt.Sprintf("%s: %s", name, text))
	}
	lines = append(lines, "</conversa>", "", ask, "A <conversa> é só contexto: use se ajudar a responder, ignore se não tiver nada a ver e não continue nem imite a conversa.")
	return strings.Join(lines, "\n")
}

func (b *bot) iaQuestion(ctx context.Context, evt *events.Message, question string, nicknames map[string]string) string {
	asker := b.PersonNamesIn(ctx, evt.Info.Chat, evt.Info.Sender.ToNonAD())
	ask := insideJokes(question) + iaAskerNames(asker) + fmt.Sprintf("<pergunta de=\"%s\">\n%s\n</pergunta>\n\nQuem perguntou foi %s, a pessoa de <quem_perguntou>. Responda só a pergunta dessa pessoa e chame ela pelo nome de maior prioridade que ela tiver, nunca pelo nome de outra pessoa da conversa.", asker.Label(), question, asker.Label())

	reply := evt.Message.GetExtendedTextMessage().GetContextInfo()
	quoted := strings.Join(strings.Fields(core.MessageText(reply.GetQuotedMessage())), " ")
	if quoted == "" {
		return ask
	}
	author := b.quotedAuthor(ctx, reply.GetParticipant(), nicknames)
	return b.answeredBefore(ctx, evt.Info.Chat, reply.GetStanzaID(), nicknames) + fmt.Sprintf("<respondendo de=\"%s\">\n%s\n</respondendo>\n\n%s A pergunta foi mandada em resposta à mensagem em <respondendo> (de %s), então leve essa mensagem em conta.",
		author, core.TruncateRunes(quoted, iaQuotedMaxRunes), ask, author)
}

func (b *bot) answeredBefore(ctx context.Context, chat types.JID, answerID string, nicknames map[string]string) string {
	if answerID == "" {
		return ""
	}
	question, found, err := b.messages.answeredQuestion(ctx, chat, answerID)
	if err != nil {
		b.Client.Log.Warnf("!ia: ler a pergunta respondida por %s em %s: %v", answerID, chat, err)
	}
	if !found {
		return ""
	}
	author := b.senderName(ctx, question.sender, nicknames)
	text := core.TruncateRunes(strings.Join(strings.Fields(question.text), " "), iaQuotedMaxRunes)
	return fmt.Sprintf("<antes de=\"%s\">\n%s\n</antes>\nA mensagem em <respondendo> foi você respondendo essa mensagem de %s.\n\n", author, text, author)
}

func iaAskerNames(asker core.PersonNames) string {
	fields := []struct{ label, value string }{
		{"apelido", asker.Nickname},
		{"nome", asker.Name},
		{"nome no WhatsApp", asker.Whatsapp},
	}
	lines := []string{"<quem_perguntou>"}
	for _, f := range fields {
		if f.value == "" {
			continue
		}
		lines = append(lines, f.label+": "+f.value)
	}
	if len(lines) == 1 {
		lines = append(lines, "número: "+asker.Number)
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
	if b.IsSelf(jid) {
		return iaSelfLabel
	}
	return b.DisplayName(ctx, jid.ToNonAD(), nicknames)
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
	rows, err := s.DB.QueryContext(ctx,
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
	_, err := s.DB.ExecContext(ctx,
		"INSERT OR IGNORE INTO ia_respostas (chat, id, pergunta_id, texto, enviada_em) VALUES (?, ?, ?, ?, ?)",
		question.Info.Chat.ToNonAD().String(), sent.ID, question.Info.ID, text, sent.Timestamp.Unix(),
	)
	return err
}

func (s *messageStore) answeredQuestion(ctx context.Context, chat types.JID, answerID string) (m storedMessage, found bool, err error) {
	err = s.DB.QueryRowContext(ctx,
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
			contact, err := b.Client.Store.Contacts.GetContact(ctx, types.NewJID(u, server))
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
	stopTyping := b.Typing(ctx, evt.Info.Chat)
	b.React(ctx, evt, "⏳")
	return func() {
		stopTyping()
		b.React(context.WithoutCancel(ctx), evt, "")
	}
}

func (b *bot) replyIA(ctx context.Context, evt *events.Message, answer string, mentions []types.JID) error {
	sent, err := b.SendReply(ctx, evt, answer, mentions)
	if err != nil {
		return err
	}
	if err := b.messages.recordIAAnswer(ctx, evt, sent, answer); err != nil {
		b.Client.Log.Warnf("guardar resposta da IA em %s: %v", evt.Info.Chat, err)
	}
	return nil
}
