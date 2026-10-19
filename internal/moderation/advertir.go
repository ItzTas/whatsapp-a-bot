package moderation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	warningLimit        = 3
	warningReasonLength = 300
	warningPersonFilter = "chat = ? AND (pessoa IN (?, ?) OR pessoa_alt IN (?, ?))"
)

type warning struct {
	reason string
	author types.JID
	at     time.Time
}

type warningCount struct {
	person types.JID
	count  int
}

type warnedMember struct {
	person types.JID
	member types.JID
	ids    []types.JID
}

type warningStore struct {
	db *sql.DB
}

func (b *bot) advertir(ctx context.Context, evt *events.Message) error {
	fields := strings.Fields(core.MessageText(evt.Message))
	actions := map[string]core.Handler{
		"perdoar": b.perdoar,
		"tirar":   b.perdoar,
		"remover": b.perdoar,
		"limpar":  b.clearWarnings,
		"zerar":   b.clearWarnings,
		"ver":     b.advertencias,
		"listar":  b.advertencias,
	}
	if len(fields) > 1 {
		if action, ok := actions[strings.ToLower(fields[1])]; ok {
			return action(ctx, evt)
		}
	}
	return b.addWarnings(ctx, evt)
}

func (b *bot) addWarnings(ctx context.Context, evt *events.Message) error {
	m, problem, err := b.moderate(ctx, evt, "advertir", "advertir", false, true)
	if err != nil {
		return err
	}
	if problem != "" {
		return b.SendText(ctx, evt.Info.Chat, problem)
	}
	reason := warningReason(core.MessageText(evt.Message))
	if utf8.RuneCountInString(reason) > warningReasonLength {
		return b.SendText(ctx, m.chat, fmt.Sprintf("Motivo grande demais, o máximo é %d letras", warningReasonLength))
	}

	author := b.SenderIdentities(ctx, evt)
	now := time.Now()
	var (
		lines  []string
		warned []warnedMember
	)
	if reason != "" {
		lines = append(lines, "📝 Motivo: "+reason, "")
	}
	for _, person := range m.people {
		p, found := core.FindParticipant(m.group.Participants, core.UsersOf(b.Identities(ctx, person)))
		if !found {
			lines = append(lines, fmt.Sprintf("@%s nem tá no grupo", person.User))
			continue
		}
		ids := core.UniqueJIDs(withoutEmptyJIDs([]types.JID{p.JID, p.LID, p.PhoneNumber}))
		count, err := b.warnings.add(ctx, m.chat, ids, reason, author, now)
		if err != nil {
			return fmt.Errorf("salvar advertência: %w", err)
		}
		lines = append(lines, fmt.Sprintf("⚠️ @%s levou uma advertência (%d/%d)", person.User, count, warningLimit))
		if count >= warningLimit {
			warned = append(warned, warnedMember{person: person, member: p.JID, ids: ids})
		}
	}

	banLines, banned, err := b.banWarned(ctx, m, warned)
	if err != nil {
		return err
	}
	lines = append(lines, banLines...)
	_, err = b.SendMentions(ctx, m.chat, strings.Join(lines, "\n"), slices.Concat(m.people, banned))
	return err
}

func (b *bot) banWarned(ctx context.Context, m moderation, warned []warnedMember) ([]string, []types.JID, error) {
	if len(warned) == 0 {
		return nil, nil, nil
	}
	people := make([]types.JID, len(warned))
	members := make([]types.JID, len(warned))
	for i, w := range warned {
		people[i], members[i] = w.person, w.member
	}
	if !b.botIsAdmin(m.group) {
		return []string{"", fmt.Sprintf("😅 %s chegou em %d advertências, mas eu não sou admin do grupo e não consegui banir. Algum admin tira na mão ou me dá admin", core.JoinMentions(people), warningLimit)}, nil, nil
	}

	results, err := b.Client.UpdateGroupParticipants(ctx, m.chat, members, banAction.change)
	if err != nil {
		return nil, nil, fmt.Errorf("banir quem chegou no limite de advertências: %w", err)
	}
	b.Departed(ctx, m.chat, removedMembers(results))
	lines := []string{""}
	banned := make([]types.JID, 0, len(results))
	for _, r := range results {
		lines = append(lines, banAction.resultLine(r))
		banned = append(banned, r.JID)
		if r.Error != 0 {
			continue
		}
		if err := b.clearBannedWarnings(ctx, m.chat, warned, r.JID); err != nil {
			return nil, nil, err
		}
	}
	return lines, banned, nil
}

func (b *bot) clearBannedWarnings(ctx context.Context, chat types.JID, warned []warnedMember, removed types.JID) error {
	for _, w := range warned {
		if !core.ContainsAnyUser(w.ids, []string{removed.User}) {
			continue
		}
		if _, _, err := b.warnings.clear(ctx, chat, w.ids); err != nil {
			return fmt.Errorf("zerar advertências de quem foi banido: %w", err)
		}
	}
	return nil
}

func warningReason(text string) string {
	return strings.Join(strings.Fields(core.MentionTagPattern.ReplaceAllString(core.TextAfterFields(text, 1), "")), " ")
}

func (b *bot) perdoar(ctx context.Context, evt *events.Message) error {
	return b.changeWarnings(ctx, evt, "perdoar", "perdoar", b.warnings.removeLast,
		"🕊️ @%[1]s foi perdoado(a): tirei a última advertência e ficou com %[3]d/%[4]d")
}

func (b *bot) clearWarnings(ctx context.Context, evt *events.Message) error {
	return b.changeWarnings(ctx, evt, "advertir limpar", "deixar com a ficha limpa", b.warnings.clear,
		"🧽 Ficha limpa! Tirei %[2]s de @%[1]s")
}

func (b *bot) changeWarnings(ctx context.Context, evt *events.Message, name, verb string, change func(context.Context, types.JID, []types.JID) (int, int, error), done string) error {
	m, problem, err := b.moderate(ctx, evt, name, verb, false, true)
	if err != nil {
		return err
	}
	if problem != "" {
		return b.SendText(ctx, evt.Info.Chat, problem)
	}
	lines := make([]string, len(m.people))
	for i, person := range m.people {
		removed, remaining, err := change(ctx, m.chat, b.memberIdentities(ctx, m.group, person))
		if err != nil {
			return fmt.Errorf("mexer nas advertências: %w", err)
		}
		lines[i] = warningChangeLine(done, person, removed, remaining)
	}
	_, err = b.SendMentions(ctx, m.chat, strings.Join(lines, "\n"), m.people)
	return err
}

func warningChangeLine(done string, person types.JID, removed, remaining int) string {
	if removed == 0 {
		return fmt.Sprintf("@%s não tem nenhuma advertência", person.User)
	}
	return fmt.Sprintf(done, person.User, core.CountLabel(removed, "advertência", "advertências"), remaining, warningLimit)
}

func (b *bot) advertencias(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return b.SendText(ctx, chat, "O !advertencias só funciona em grupo")
	}
	people := b.moderationTargets(evt)
	if len(people) == 0 {
		return b.showChatWarnings(ctx, chat)
	}
	return b.showPersonWarnings(ctx, chat, people)
}

func (b *bot) showChatWarnings(ctx context.Context, chat types.JID) error {
	counts, err := b.warnings.summary(ctx, chat)
	if err != nil {
		return fmt.Errorf("ler advertências: %w", err)
	}
	if len(counts) == 0 {
		return b.SendText(ctx, chat, "✨ Ninguém tem advertência neste grupo")
	}
	lines := []string{"⚠️ *Advertências do grupo*", ""}
	people := make([]types.JID, len(counts))
	for i, c := range counts {
		people[i] = c.person
		lines = append(lines, fmt.Sprintf("• @%s: %d/%d", c.person.User, c.count, warningLimit))
	}
	lines = append(lines, "", fmt.Sprintf("_Na %dª advertência a pessoa é banida. Pra ver os motivos: !advertencias @pessoa_", warningLimit))
	_, err = b.SendMentions(ctx, chat, strings.Join(lines, "\n"), people)
	return err
}

func (b *bot) showPersonWarnings(ctx context.Context, chat types.JID, people []types.JID) error {
	var nicknames map[string]string
	warnings := make([][]warning, len(people))
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error { nicknames = b.ChatNicknames(gctx, chat); return nil })
	for i, person := range people {
		g.Go(func() (err error) {
			warnings[i], err = b.warnings.of(gctx, chat, b.Identities(gctx, person))
			return err
		})
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler advertências: %w", err)
	}

	var sections []string
	for i, person := range people {
		sections = append(sections, b.personWarningsText(ctx, person, warnings[i], nicknames))
	}
	_, err := b.SendMentions(ctx, chat, strings.Join(sections, "\n\n"), people)
	return err
}

func (b *bot) personWarningsText(ctx context.Context, person types.JID, warnings []warning, nicknames map[string]string) string {
	if len(warnings) == 0 {
		return fmt.Sprintf("✨ @%s não tem nenhuma advertência", person.User)
	}
	lines := []string{fmt.Sprintf("⚠️ *Advertências de @%s* (%d/%d)", person.User, len(warnings), warningLimit)}
	for i, w := range warnings {
		reason := w.reason
		if reason == "" {
			reason = "_sem motivo_"
		}
		at := w.at.In(core.BrazilTime)
		lines = append(lines,
			fmt.Sprintf("*%d.* %s", i+1, reason),
			fmt.Sprintf("    👮 por %s em %s às %s", b.DisplayName(ctx, w.author, nicknames), at.Format("02/01/2006"), at.Format("15:04")),
		)
	}
	return strings.Join(lines, "\n")
}

func warningPersonArgs(chat types.JID, ids []types.JID) []any {
	first, second := core.IdentityKeys(ids)
	return []any{chat.String(), first, second, first, second}
}

func (s *warningStore) add(ctx context.Context, chat types.JID, ids []types.JID, reason string, author []types.JID, now time.Time) (int, error) {
	person, personAlt := core.IdentityKeys(ids)
	authorID, authorAlt := core.IdentityKeys(author)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO advertencias (chat, pessoa, pessoa_alt, motivo, autor, autor_alt, criada_em)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		chat.String(), person, personAlt, reason, authorID, authorAlt, now.Unix(),
	)
	if err != nil {
		return 0, err
	}
	return s.count(ctx, chat, ids)
}

func (s *warningStore) count(ctx context.Context, chat types.JID, ids []types.JID) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM advertencias WHERE "+warningPersonFilter, warningPersonArgs(chat, ids)...).Scan(&n)
	return n, err
}

func (s *warningStore) removeLast(ctx context.Context, chat types.JID, ids []types.JID) (int, int, error) {
	res, err := s.db.ExecContext(ctx,
		"DELETE FROM advertencias WHERE id = (SELECT id FROM advertencias WHERE "+warningPersonFilter+" ORDER BY id DESC LIMIT 1)",
		warningPersonArgs(chat, ids)...,
	)
	if err != nil {
		return 0, 0, err
	}
	removed, err := res.RowsAffected()
	if err != nil {
		return 0, 0, err
	}
	remaining, err := s.count(ctx, chat, ids)
	return int(removed), remaining, err
}

func (s *warningStore) clear(ctx context.Context, chat types.JID, ids []types.JID) (int, int, error) {
	res, err := s.db.ExecContext(ctx, "DELETE FROM advertencias WHERE "+warningPersonFilter, warningPersonArgs(chat, ids)...)
	if err != nil {
		return 0, 0, err
	}
	removed, err := res.RowsAffected()
	return int(removed), 0, err
}

func (s *warningStore) of(ctx context.Context, chat types.JID, ids []types.JID) (warnings []warning, err error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT motivo, autor, criada_em FROM advertencias WHERE "+warningPersonFilter+" ORDER BY id",
		warningPersonArgs(chat, ids)...,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var (
			w        warning
			author   string
			unixTime int64
		)
		if err := rows.Scan(&w.reason, &author, &unixTime); err != nil {
			return nil, err
		}
		if w.author, err = types.ParseJID(author); err != nil {
			return nil, err
		}
		w.at = time.Unix(unixTime, 0)
		warnings = append(warnings, w)
	}
	return warnings, rows.Err()
}

func (s *warningStore) summary(ctx context.Context, chat types.JID) (counts []warningCount, err error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT pessoa, COUNT(*) FROM advertencias WHERE chat = ? GROUP BY pessoa ORDER BY COUNT(*) DESC, MIN(id)",
		chat.String(),
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var (
			c      warningCount
			person string
		)
		if err := rows.Scan(&person, &c.count); err != nil {
			return nil, err
		}
		if c.person, err = types.ParseJID(person); err != nil {
			return nil, err
		}
		counts = append(counts, c)
	}
	return counts, rows.Err()
}
