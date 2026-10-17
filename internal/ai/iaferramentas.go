package ai

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/llm"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	iaToolDefaultLines  = 15
	iaToolMaxLines      = 50
	iaToolTimeLayout    = "2006-01-02 15:04"
	iaKeywordMinRunes   = 5
	iaMaxKeywords       = 3
	iaHistoryPerKeyword = 6
	iaHistoryMaxLines   = 12
)

type iaMember struct {
	jid          types.JID
	users        []string
	name         string
	nickname     string
	realName     string
	whatsappName string
	admin        bool
}

func (m iaMember) aliases() []string {
	aliases := slices.DeleteFunc([]string{m.nickname, m.realName, m.whatsappName, m.name}, func(a string) bool { return a == "" })
	slices.Sort(aliases)
	return slices.Compact(aliases)
}

func (m iaMember) describe() string {
	line := "- " + m.name
	if m.realName != "" && !strings.EqualFold(m.realName, m.name) {
		line += " (nome: " + m.realName + ")"
	}
	if m.admin {
		line += " (admin)"
	}
	return line
}

type iaToolkit struct {
	b         *bot
	chat      types.JID
	nicknames map[string]string
	members   func() ([]iaMember, error)

	mu     sync.Mutex
	tagged map[string]types.JID
}

type chatLine struct {
	sender  storedSender
	text    string
	sentAt  time.Time
	fromBot bool
}

func (b *bot) iaToolkit(ctx context.Context, evt *events.Message, nicknames map[string]string) *iaToolkit {
	k := &iaToolkit{b: b, chat: evt.Info.Chat.ToNonAD(), nicknames: nicknames, tagged: map[string]types.JID{}}
	k.members = sync.OnceValues(func() ([]iaMember, error) { return k.loadMembers(ctx) })
	return k
}

func (k *iaToolkit) tools() []llm.Tool {
	quantity := map[string]any{"type": "integer", "description": fmt.Sprintf("quantas mensagens trazer (padrão %d, máximo %d)", iaToolDefaultLines, iaToolMaxLines)}
	return append([]llm.Tool{
		{
			Name:        "membros_do_grupo",
			Description: "Lista quem está no grupo agora, com o apelido ou nome de cada um e quem é admin.",
			Run:         k.listMembers,
		},
		{
			Name:        "marcar_pessoas",
			Description: "Prepara a marcação de pessoas do grupo. Devolve o @ que você tem que escrever na resposta pra cada pessoa ser marcada de verdade.",
			Properties: map[string]any{
				"nomes": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "apelidos ou nomes das pessoas"},
			},
			Required: []string{"nomes"},
			Run:      k.tagPeople,
		},
		{
			Name:        "comandos_recentes",
			Description: "Mostra os últimos comandos com ! que mandaram nesta conversa, com quem mandou e quando.",
			Properties:  map[string]any{"quantidade": quantity},
			Run:         k.recentCommands,
		},
		{
			Name:        "buscar_mensagens",
			Description: "Procura no histórico desta conversa as mensagens que contêm uma palavra. Use sempre que perguntarem o que alguém falou, disse, combinou ou mandou antes, ou sobre um assunto que não está nas mensagens recentes. Procure por uma palavra-chave curta do assunto, tipo churrasco ou viagem. Também acha o que você mesmo já respondeu, que aparece como \"você, o bot\".",
			Properties: map[string]any{
				"texto":      map[string]any{"type": "string", "description": "trecho a procurar, sem diferenciar maiúscula"},
				"quantidade": quantity,
			},
			Required: []string{"texto"},
			Run:      k.searchMessages,
		},
		{
			Name:        "mensagens_da_pessoa",
			Description: "Mostra as últimas mensagens que uma pessoa do grupo mandou nesta conversa.",
			Properties: map[string]any{
				"nome":       map[string]any{"type": "string", "description": "apelido ou nome da pessoa"},
				"quantidade": quantity,
			},
			Required: []string{"nome"},
			Run:      k.personMessages,
		},
		{
			Name:        "ver_conversa",
			Description: "Mostra a conversa a partir de um momento, em ordem. Use depois de buscar_mensagens pra ler o que veio em volta de uma mensagem.",
			Properties: map[string]any{
				"inicio":     map[string]any{"type": "string", "description": "data e hora no horário de Brasília, no formato AAAA-MM-DD HH:MM"},
				"quantidade": quantity,
			},
			Required: []string{"inicio"},
			Run:      k.conversationFrom,
		},
		{
			Name:        "data_e_hora",
			Description: "Diz a data, a hora e o dia da semana de agora no horário de Brasília.",
			Run:         func(context.Context, json.RawMessage) (string, error) { return nowInBrazil(), nil },
		},
	}, slices.Concat(k.memoryTools(), k.randomTools(), k.economyTools(), webTools())...)
}

func (k *iaToolkit) loadMembers(ctx context.Context) ([]iaMember, error) {
	if k.chat.Server != types.GroupServer {
		return nil, errors.New("essa conversa não é um grupo")
	}
	group, err := k.b.Client.GetGroupInfo(ctx, k.chat)
	if err != nil {
		return nil, fmt.Errorf("ler o grupo: %w", err)
	}
	names := k.b.ChatNames(ctx, k.chat)
	members := make([]iaMember, 0, len(group.Participants))
	for _, p := range group.Participants {
		if k.b.IsSelf(p.JID) || k.b.IsSelf(p.LID) || k.b.IsSelf(p.PhoneNumber) {
			continue
		}
		ids := slices.DeleteFunc([]types.JID{p.JID, p.LID, p.PhoneNumber}, types.JID.IsEmpty)
		nickname := core.NicknameOf(ids, k.nicknames)
		whatsappName := k.b.ContactName(ctx, ids)
		members = append(members, iaMember{
			jid:          p.JID.ToNonAD(),
			users:        core.UsersOf(ids),
			name:         cmp.Or(nickname, whatsappName, k.b.DisplayName(ctx, p.JID.ToNonAD(), nil)),
			nickname:     nickname,
			realName:     core.NicknameOf(ids, names),
			whatsappName: whatsappName,
			admin:        p.IsAdmin || p.IsSuperAdmin,
		})
	}
	return members, nil
}

func (k *iaToolkit) listMembers(context.Context, json.RawMessage) (string, error) {
	members, err := k.members()
	if err != nil {
		return "", err
	}
	lines := []string{fmt.Sprintf("%d pessoas no grupo, pelo apelido de cada um (ou o nome de quem não tem apelido), com o nome de verdade entre parênteses quando anotado:", len(members))}
	for _, m := range members {
		lines = append(lines, m.describe())
	}
	return strings.Join(lines, "\n"), nil
}

func (k *iaToolkit) findMember(name string) (iaMember, error) {
	members, err := k.members()
	if err != nil {
		return iaMember{}, err
	}
	wanted := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(name, "@")))
	if wanted == "" {
		return iaMember{}, errors.New("faltou o nome da pessoa")
	}
	if i := slices.IndexFunc(members, func(m iaMember) bool {
		return slices.ContainsFunc(m.aliases(), func(a string) bool { return strings.ToLower(a) == wanted }) || slices.Contains(m.users, wanted)
	}); i >= 0 {
		return members[i], nil
	}
	partial := slices.DeleteFunc(slices.Clone(members), func(m iaMember) bool {
		return !slices.ContainsFunc(m.aliases(), func(a string) bool { return strings.Contains(strings.ToLower(a), wanted) })
	})
	if len(partial) == 1 {
		return partial[0], nil
	}
	if len(partial) == 0 {
		return iaMember{}, fmt.Errorf("ninguém no grupo se chama %q, veja os nomes com membros_do_grupo", name)
	}
	names := make([]string, len(partial))
	for i, m := range partial {
		names[i] = m.name
	}
	return iaMember{}, fmt.Errorf("mais de uma pessoa bate com %q: %s", name, strings.Join(names, ", "))
}

func (k *iaToolkit) tagPeople(_ context.Context, input json.RawMessage) (string, error) {
	args, err := decodeToolInput[struct {
		Names []string `json:"nomes"`
	}](input)
	if err != nil {
		return "", err
	}
	if len(args.Names) == 0 {
		return "", errors.New("faltou a lista de nomes")
	}
	lines := make([]string, 0, len(args.Names)+1)
	for _, name := range args.Names {
		m, err := k.findMember(name)
		if err != nil {
			lines = append(lines, fmt.Sprintf("- %s: %v", name, err))
			continue
		}
		tag := "@" + m.jid.User
		k.mu.Lock()
		k.tagged[tag] = m.jid
		k.mu.Unlock()
		lines = append(lines, fmt.Sprintf("- %s: escreva %s", m.name, tag))
	}
	lines = append(lines, "Escreva o @ exatamente assim na resposta, sem trocar pelo nome, senão a pessoa não é marcada.")
	return strings.Join(lines, "\n"), nil
}

func (k *iaToolkit) resolveMentions(answer string) (string, []types.JID) {
	if !strings.Contains(answer, "@") {
		return answer, nil
	}
	members, err := k.members()
	if err != nil {
		return answer, k.mentionsIn(answer)
	}
	type alias struct {
		name   string
		member iaMember
	}
	var aliases []alias
	for _, m := range members {
		for _, a := range m.aliases() {
			aliases = append(aliases, alias{a, m})
		}
	}
	slices.SortStableFunc(aliases, func(a, b alias) int { return len(b.name) - len(a.name) })
	for _, a := range aliases {
		pattern := regexp.MustCompile(`(?i)@` + regexp.QuoteMeta(a.name) + `([^\p{L}\p{N}]|$)`)
		if !pattern.MatchString(answer) {
			continue
		}
		tag := "@" + a.member.jid.User
		answer = pattern.ReplaceAllString(answer, tag+"${1}")
		k.mu.Lock()
		k.tagged[tag] = a.member.jid
		k.mu.Unlock()
	}
	return answer, k.mentionsIn(answer)
}

func (k *iaToolkit) mentionsIn(answer string) []types.JID {
	k.mu.Lock()
	defer k.mu.Unlock()
	var mentions []types.JID
	for _, tag := range core.MentionTagPattern.FindAllString(answer, -1) {
		jid, ok := k.tagged[tag]
		if ok && !slices.Contains(mentions, jid) {
			mentions = append(mentions, jid)
		}
	}
	return mentions
}

func (k *iaToolkit) recentCommands(ctx context.Context, input json.RawMessage) (string, error) {
	args, err := decodeToolInput[struct {
		Quantity int `json:"quantidade"`
	}](input)
	if err != nil {
		return "", err
	}
	lines, err := k.b.messages.chatLines(ctx, k.chat, "ltrim(texto) LIKE '!%'", nil, false, toolQuantity(args.Quantity))
	if err != nil {
		return "", err
	}
	return k.render(ctx, lines, "Ninguém mandou comando aqui ainda."), nil
}

func (k *iaToolkit) searchMessages(ctx context.Context, input json.RawMessage) (string, error) {
	args, err := decodeToolInput[struct {
		Text     string `json:"texto"`
		Quantity int    `json:"quantidade"`
	}](input)
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(args.Text)
	if text == "" {
		return "", errors.New("faltou o texto pra procurar")
	}
	lines, err := k.b.messages.chatLines(ctx, k.chat, "instr(lower(texto), lower(?)) > 0", []any{text}, false, toolQuantity(args.Quantity))
	if err != nil {
		return "", err
	}
	return k.render(ctx, lines, fmt.Sprintf("Nenhuma mensagem com %q.", text)), nil
}

func (k *iaToolkit) personMessages(ctx context.Context, input json.RawMessage) (string, error) {
	args, err := decodeToolInput[struct {
		Name     string `json:"nome"`
		Quantity int    `json:"quantidade"`
	}](input)
	if err != nil {
		return "", err
	}
	m, err := k.findMember(args.Name)
	if err != nil {
		return "", err
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?, ", len(m.users)), ", ")
	filter := fmt.Sprintf("(remetente IN (%s) OR remetente_alt IN (%s))", placeholders, placeholders)
	users := make([]any, 0, 2*len(m.users))
	for range 2 {
		for _, u := range m.users {
			users = append(users, u)
		}
	}
	lines, err := k.b.messages.chatLines(ctx, k.chat, filter, users, false, toolQuantity(args.Quantity))
	if err != nil {
		return "", err
	}
	return k.render(ctx, lines, m.name+" ainda não mandou nada aqui."), nil
}

func (k *iaToolkit) conversationFrom(ctx context.Context, input json.RawMessage) (string, error) {
	args, err := decodeToolInput[struct {
		Start    string `json:"inicio"`
		Quantity int    `json:"quantidade"`
	}](input)
	if err != nil {
		return "", err
	}
	start, err := time.ParseInLocation(iaToolTimeLayout, strings.TrimSpace(args.Start), core.BrazilTime)
	if err != nil {
		return "", fmt.Errorf("data %q fora do formato AAAA-MM-DD HH:MM", args.Start)
	}
	lines, err := k.b.messages.chatLines(ctx, k.chat, "enviada_em >= ?", []any{start.Unix()}, true, toolQuantity(args.Quantity))
	if err != nil {
		return "", err
	}
	return k.render(ctx, lines, "Não tem mensagem guardada a partir desse momento."), nil
}

func (k *iaToolkit) render(ctx context.Context, lines []chatLine, empty string) string {
	if len(lines) == 0 {
		return empty
	}
	return k.b.renderChatLines(ctx, lines, k.nicknames)
}

func (b *bot) renderChatLines(ctx context.Context, lines []chatLine, nicknames map[string]string) string {
	names := map[storedSender]string{}
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		name, ok := names[l.sender]
		if !ok {
			name = b.storedAuthor(ctx, storedMessage{sender: l.sender, fromBot: l.fromBot}, nicknames)
			names[l.sender] = name
		}
		text := core.TruncateRunes(strings.Join(strings.Fields(l.text), " "), iaContextMaxRunes)
		out = append(out, fmt.Sprintf("[%s] %s: %s", l.sentAt.In(core.BrazilTime).Format("02/01/2006 15:04"), name, text))
	}
	return strings.Join(out, "\n")
}

func (b *bot) relatedHistory(ctx context.Context, evt *events.Message, question string, nicknames map[string]string) string {
	keywords := searchKeywords(question)
	if len(keywords) == 0 {
		return ""
	}
	found := make([][]chatLine, len(keywords))
	g, gctx := errgroup.WithContext(ctx)
	for i, keyword := range keywords {
		g.Go(func() (err error) {
			found[i], err = b.messages.chatLines(gctx, evt.Info.Chat.ToNonAD(),
				"id != ? AND ltrim(texto) NOT LIKE '!%' AND instr(lower(texto), lower(?)) > 0",
				[]any{evt.Info.ID, keyword}, false, iaHistoryPerKeyword)
			return err
		})
	}
	if err := g.Wait(); err != nil {
		b.Client.Log.Warnf("!ia: procurar no histórico de %s: %v", evt.Info.Chat, err)
		return ""
	}

	lines := slices.CompactFunc(slices.SortedFunc(slices.Values(slices.Concat(found...)), func(a, b chatLine) int {
		return cmp.Or(a.sentAt.Compare(b.sentAt), strings.Compare(a.text, b.text))
	}), func(a, b chatLine) bool { return a.sentAt.Equal(b.sentAt) && a.text == b.text })
	if len(lines) == 0 {
		return ""
	}
	lines = lines[max(0, len(lines)-iaHistoryMaxLines):]
	return "<historico>\n" + b.renderChatLines(ctx, lines, nicknames) +
		"\n</historico>\nO <historico> tem mensagens antigas do grupo que citam palavras da pergunta, achadas automaticamente. Use se tiverem a ver com a pergunta e ignore se não tiverem.\n\n"
}

func searchKeywords(question string) []string {
	generic := map[string]bool{
		"sobre": true, "quais": true, "quando": true, "porque": true, "porquê": true, "grupo": true,
		"falou": true, "falaram": true, "disse": true, "disseram": true, "combinaram": true, "combinou": true,
		"mandou": true, "mandaram": true, "alguém": true, "alguma": true, "vocês": true, "aquele": true,
		"aquela": true, "ontem": true, "semana": true, "tinha": true, "estava": true, "lembra": true,
		"coisa": true, "coisas": true, "pessoa": true, "pessoal": true, "gente": true, "depois": true,
		"antes": true, "agora": true, "ainda": true, "também": true, "muito": true, "então": true,
		"falando": true, "conversa": true, "mensagem": true, "mensagens": true, "aconteceu": true,
		"tudo": true, "todos": true, "nesse": true, "nessa": true, "desse": true, "dessa": true,
	}
	var keywords []string
	for _, field := range strings.Fields(strings.ToLower(question)) {
		word := strings.TrimFunc(field, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
		if utf8.RuneCountInString(word) < iaKeywordMinRunes || generic[word] || slices.Contains(keywords, word) {
			continue
		}
		keywords = append(keywords, word)
	}
	slices.SortStableFunc(keywords, func(a, b string) int { return utf8.RuneCountInString(b) - utf8.RuneCountInString(a) })
	return keywords[:min(len(keywords), iaMaxKeywords)]
}

func (s *messageStore) chatLines(ctx context.Context, chat types.JID, filter string, args []any, oldestFirst bool, limit int) (lines []chatLine, err error) {
	order := "DESC"
	if oldestFirst {
		order = "ASC"
	}
	query := fmt.Sprintf(`SELECT remetente, remetente_alt, texto, enviada_em, do_bot FROM (
			SELECT id, remetente, remetente_alt, texto, enviada_em, 0 AS do_bot FROM mensagens
			WHERE chat = ? AND id NOT IN (SELECT id FROM ia_respostas WHERE chat = ?)
			UNION ALL
			SELECT id, '', '', texto, enviada_em, 1 FROM ia_respostas WHERE chat = ?
		) WHERE %s ORDER BY enviada_em %s LIMIT ?`, filter, order)
	params := append(append([]any{chat.String(), chat.String(), chat.String()}, args...), limit)
	rows, err := s.DB.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var l chatLine
		var sentAt int64
		if err := rows.Scan(&l.sender.user, &l.sender.alt, &l.text, &sentAt, &l.fromBot); err != nil {
			return nil, err
		}
		l.sentAt = time.Unix(sentAt, 0)
		lines = append(lines, l)
	}
	if !oldestFirst {
		slices.Reverse(lines)
	}
	return lines, rows.Err()
}

func decodeToolInput[T any](input json.RawMessage) (T, error) {
	var args T
	if len(input) == 0 {
		return args, nil
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return args, fmt.Errorf("argumentos inválidos: %w", err)
	}
	return args, nil
}

func toolQuantity(n int) int {
	if n <= 0 {
		return iaToolDefaultLines
	}
	return min(n, iaToolMaxLines)
}

func nowInBrazil() string {
	weekdays := [...]string{"domingo", "segunda-feira", "terça-feira", "quarta-feira", "quinta-feira", "sexta-feira", "sábado"}
	now := time.Now().In(core.BrazilTime)
	return fmt.Sprintf("%s, %s (horário de Brasília)", weekdays[now.Weekday()], now.Format("02/01/2006 15:04"))
}
