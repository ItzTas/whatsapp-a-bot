package main

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	reminderMaxPerChat  = 10
	reminderMaxLength   = 500
	reminderPreviewSize = 120
	reminderTick        = time.Minute
	reminderGrace       = 30 * time.Minute
	reminderSendLimit   = 4
)

var (
	reminderZone        = time.FixedZone("BRT", -3*60*60)
	reminderTimePattern = regexp.MustCompile(`^(\d{1,2})[h:](\d{2})?$`)
	errReminderLimit    = errors.New("limite de lembretes na conversa")
)

type reminder struct {
	id       int64
	chat     types.JID
	hour     int
	minute   int
	text     string
	mentions []types.JID
	creator  []types.JID
	next     time.Time
}

type reminderStore struct {
	db *sql.DB
}

func (b *bot) lembrete(ctx context.Context, evt *events.Message) error {
	fields := strings.Fields(messageText(evt.Message))
	if len(fields) < 2 {
		return b.listReminders(ctx, evt)
	}
	removal := []string{"apagar", "remover", "tirar", "excluir", "cancelar"}
	if slices.Contains(removal, strings.ToLower(fields[1])) {
		return b.removeReminder(ctx, evt)
	}
	return b.addReminder(ctx, evt)
}

func (b *bot) addReminder(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	text := messageText(evt.Message)
	hour, minute, ok := parseClock(strings.Fields(text)[1])
	body := textAfterFields(text, 2)
	if !ok || body == "" {
		return b.sendText(ctx, chat, "⏰ Use assim: !lembrete <hora> <texto>, tipo !lembrete 8h30 tomar água ou !lembrete 22:00 boa noite. Todo dia nessa hora eu mando o texto aqui")
	}
	if utf8.RuneCountInString(body) > reminderMaxLength {
		return b.sendText(ctx, chat, fmt.Sprintf("Lembrete grande demais, o máximo é %d letras", reminderMaxLength))
	}

	now := time.Now()
	r := reminder{
		chat:     chat,
		hour:     hour,
		minute:   minute,
		text:     body,
		mentions: mentionedJIDs(evt.Message),
		creator:  b.senderIdentities(ctx, evt),
		next:     nextReminder(now, hour, minute),
	}
	err := b.reminders.add(ctx, r, now)
	if errors.Is(err, errReminderLimit) {
		return b.sendText(ctx, chat, fmt.Sprintf("Essa conversa já tem %d lembretes, que é o máximo. Apaga algum com !lembrete apagar <número>", reminderMaxPerChat))
	}
	if err != nil {
		return fmt.Errorf("salvar lembrete: %w", err)
	}
	return b.replyTo(ctx, evt, fmt.Sprintf("⏰ Lembrete criado! Todo dia às %s eu mando aqui, e o primeiro sai %s\n\n_Pra ver os lembretes da conversa, manda !lembrete_",
		clockLabel(hour, minute), dayLabel(r.next, now)))
}

func (b *bot) listReminders(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	var (
		reminders []reminder
		nicknames map[string]string
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { reminders, err = b.reminders.inChat(gctx, chat); return err })
	g.Go(func() error { nicknames = b.chatNicknames(gctx, chat); return nil })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler lembretes: %w", err)
	}
	if len(reminders) == 0 {
		return b.sendText(ctx, chat, "⏰ Ninguém criou lembrete nesta conversa ainda. Cria com !lembrete <hora> <texto>, tipo !lembrete 8h30 tomar água")
	}

	lines := []string{"⏰ *Lembretes desta conversa*", ""}
	for i, r := range reminders {
		lines = append(lines,
			fmt.Sprintf("*%d.* Todo dia às %s", i+1, clockLabel(r.hour, r.minute)),
			b.reminderPreview(ctx, r, nicknames),
			"✍️ Criado por "+b.displayName(ctx, r.creator[0], nicknames),
			"",
		)
	}
	lines = append(lines, "_Apaga com !lembrete apagar <número> (só quem criou ou um admin do grupo)_")
	return b.sendText(ctx, chat, strings.Join(lines, "\n"))
}

func (b *bot) removeReminder(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	n, err := strconv.Atoi(strings.TrimPrefix(textAfterFields(messageText(evt.Message), 2), "#"))
	if err != nil {
		return b.sendText(ctx, chat, "Diz o número do lembrete: !lembrete apagar 2 (os números aparecem no !lembrete)")
	}
	reminders, err := b.reminders.inChat(ctx, chat)
	if err != nil {
		return fmt.Errorf("ler lembretes: %w", err)
	}
	if n < 1 || n > len(reminders) {
		return b.sendText(ctx, chat, fmt.Sprintf("Não tem lembrete número %d nesta conversa. Confere no !lembrete", n))
	}

	r := reminders[n-1]
	allowed, err := b.canRemoveReminder(ctx, evt, r)
	if err != nil {
		return fmt.Errorf("checar quem pode apagar o lembrete: %w", err)
	}
	if !allowed {
		return b.sendText(ctx, chat, "Só quem criou o lembrete ou um admin do grupo pode apagar ele 🚫")
	}
	if err := b.reminders.remove(ctx, r.id); err != nil {
		return fmt.Errorf("apagar lembrete: %w", err)
	}
	return b.sendText(ctx, chat, fmt.Sprintf("🗑️ Apaguei o lembrete das %s: %s", clockLabel(r.hour, r.minute), b.reminderPreview(ctx, r, b.chatNicknames(ctx, chat))))
}

func (b *bot) canRemoveReminder(ctx context.Context, evt *events.Message, r reminder) (bool, error) {
	sender := usersOf(b.senderIdentities(ctx, evt))
	if containsAnyUser(r.creator, sender) || b.isOwner(ctx, evt) {
		return true, nil
	}
	if r.chat.Server != types.GroupServer {
		return false, nil
	}
	group, err := b.client.GetGroupInfo(ctx, r.chat)
	if err != nil {
		return false, err
	}
	p, found := findParticipant(group.Participants, sender)
	return found && isGroupAdmin(p), nil
}

func (b *bot) reminderPreview(ctx context.Context, r reminder, nicknames map[string]string) string {
	text := mentionTagPattern.ReplaceAllStringFunc(r.text, func(tag string) string {
		i := slices.IndexFunc(r.mentions, func(m types.JID) bool { return "@"+m.User == tag })
		if i < 0 {
			return tag
		}
		return "@" + b.displayName(ctx, r.mentions[i], nicknames)
	})
	return truncateRunes(strings.ReplaceAll(text, "\n", " "), reminderPreviewSize)
}

func (b *bot) runReminders(ctx context.Context) {
	ticker := time.NewTicker(reminderTick)
	defer ticker.Stop()
	for {
		if b.client.IsLoggedIn() {
			b.sendDueReminders(ctx)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (b *bot) sendDueReminders(ctx context.Context) {
	now := time.Now()
	due, err := b.reminders.due(ctx, now)
	if err != nil {
		b.client.Log.Errorf("procurar lembretes: %v", err)
		return
	}
	var g errgroup.Group
	g.SetLimit(reminderSendLimit)
	for _, r := range due {
		g.Go(func() error {
			if err := b.fireReminder(ctx, r, now); err != nil {
				b.client.Log.Errorf("mandar o lembrete %d em %s: %v", r.id, r.chat, err)
			}
			return nil
		})
	}
	_ = g.Wait()
}

func (b *bot) fireReminder(ctx context.Context, r reminder, now time.Time) error {
	claimed, err := b.reminders.reschedule(ctx, r, nextReminder(now, r.hour, r.minute))
	if err != nil || !claimed {
		return err
	}
	if now.Sub(r.next) > reminderGrace {
		b.client.Log.Infof("lembrete %d de %s pulado: era pra sair %s e o bot tava fora do ar", r.id, r.chat, r.next.In(reminderZone).Format(time.DateTime))
		return nil
	}
	active, err := b.reminderActive(ctx, r)
	if err != nil || !active {
		return err
	}
	_, err = b.sendMentions(ctx, r.chat, reminderMessage(r), append(slices.Clone(r.mentions), r.creator[0]))
	return err
}

func (b *bot) reminderActive(ctx context.Context, r reminder) (bool, error) {
	var on, allowed bool
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { on, err = b.modes.enabled(gctx, r.chat, modeUtilities); return err })
	g.Go(func() (err error) { allowed, err = b.allowedIn(gctx, r.chat, r.creator); return err })
	if err := g.Wait(); err != nil {
		return false, err
	}
	return on && allowed, nil
}

func reminderMessage(r reminder) string {
	return fmt.Sprintf("⏰ *Lembrete das %s*\n\n%s\n\n✍️ Criado por @%s", clockLabel(r.hour, r.minute), r.text, r.creator[0].User)
}

func parseClock(arg string) (hour, minute int, ok bool) {
	m := reminderTimePattern.FindStringSubmatch(strings.ToLower(arg))
	if m == nil {
		return 0, 0, false
	}
	hour, _ = strconv.Atoi(m[1])
	minute, _ = strconv.Atoi(cmp.Or(m[2], "0"))
	return hour, minute, hour < 24 && minute < 60
}

func nextReminder(now time.Time, hour, minute int) time.Time {
	local := now.In(reminderZone)
	at := time.Date(local.Year(), local.Month(), local.Day(), hour, minute, 0, 0, reminderZone)
	if at.After(now) {
		return at
	}
	return at.AddDate(0, 0, 1)
}

func clockLabel(hour, minute int) string {
	return fmt.Sprintf("%02d:%02d", hour, minute)
}

func dayLabel(at, now time.Time) string {
	if at.In(reminderZone).Day() == now.In(reminderZone).Day() {
		return "hoje"
	}
	return "amanhã"
}

func textAfterFields(text string, n int) string {
	for range n {
		text = strings.TrimLeftFunc(text, unicode.IsSpace)
		i := strings.IndexFunc(text, unicode.IsSpace)
		if i < 0 {
			return ""
		}
		text = text[i:]
	}
	return strings.TrimSpace(text)
}

func (s *reminderStore) add(ctx context.Context, r reminder, now time.Time) error {
	creator, creatorAlt := identityKeys(r.creator)
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO lembretes (chat, hora, minuto, texto, mencoes, criador, criador_alt, criado_em, proximo)
		 SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?
		 WHERE (SELECT COUNT(*) FROM lembretes WHERE chat = ?) < ?`,
		r.chat.String(), r.hour, r.minute, r.text, jidList(r.mentions), creator, creatorAlt, now.Unix(), r.next.Unix(),
		r.chat.String(), reminderMaxPerChat,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err == nil && n == 0 {
		return errReminderLimit
	}
	return err
}

func (s *reminderStore) inChat(ctx context.Context, chat types.JID) ([]reminder, error) {
	return s.query(ctx, "chat = ? ORDER BY hora, minuto, id", chat.String())
}

func (s *reminderStore) due(ctx context.Context, now time.Time) ([]reminder, error) {
	return s.query(ctx, "proximo <= ? ORDER BY proximo", now.Unix())
}

func (s *reminderStore) query(ctx context.Context, filter string, args ...any) (reminders []reminder, err error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, chat, hora, minuto, texto, mencoes, criador, criador_alt, proximo FROM lembretes WHERE "+filter,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		r, err := scanReminder(rows)
		if err != nil {
			return nil, err
		}
		reminders = append(reminders, r)
	}
	return reminders, rows.Err()
}

func scanReminder(rows *sql.Rows) (reminder, error) {
	var (
		r                                   reminder
		chat, mentions, creator, creatorAlt string
		next                                int64
	)
	if err := rows.Scan(&r.id, &chat, &r.hour, &r.minute, &r.text, &mentions, &creator, &creatorAlt, &next); err != nil {
		return reminder{}, err
	}
	jids, err := parseJIDs(append([]string{chat, creator, creatorAlt}, strings.Fields(mentions)...))
	if err != nil {
		return reminder{}, err
	}
	r.chat, r.creator, r.mentions, r.next = jids[0], uniqueJIDs(jids[1:3]), jids[3:], time.Unix(next, 0)
	return r, nil
}

func (s *reminderStore) reschedule(ctx context.Context, r reminder, next time.Time) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		"UPDATE lembretes SET proximo = ? WHERE id = ? AND proximo = ?",
		next.Unix(), r.id, r.next.Unix(),
	)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (s *reminderStore) remove(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM lembretes WHERE id = ?", id)
	return err
}

func jidList(jids []types.JID) string {
	raw := make([]string, len(jids))
	for i, j := range jids {
		raw[i] = j.ToNonAD().String()
	}
	return strings.Join(raw, " ")
}

func parseJIDs(raw []string) ([]types.JID, error) {
	jids := make([]types.JID, len(raw))
	for i, s := range raw {
		jid, err := types.ParseJID(s)
		if err != nil {
			return nil, err
		}
		jids[i] = jid
	}
	return jids, nil
}
