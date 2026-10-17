package utilities

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

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	birthdayListMax   = 30
	birthdaySendLimit = 4
)

var birthdayDatePattern = regexp.MustCompile(`^(\d{1,2})[/.-](\d{1,2})(?:[/.-]\d{2,4})?$`)

type birthday struct {
	id     int64
	chat   types.JID
	person []types.JID
	day    int
	month  int
}

type birthdayStore struct {
	db *sql.DB
}

func (b *bot) aniversario(ctx context.Context, evt *events.Message) error {
	fields := strings.Fields(core.MentionTagPattern.ReplaceAllString(core.MessageText(evt.Message), ""))
	if len(fields) < 2 && len(core.MentionedJIDs(evt.Message)) == 0 {
		return b.listBirthdays(ctx, evt)
	}
	removal := []string{"apagar", "remover", "tirar", "excluir"}
	if len(fields) > 1 && slices.Contains(removal, strings.ToLower(fields[1])) {
		return b.removeBirthday(ctx, evt)
	}
	return b.saveBirthday(ctx, evt, fields[1:])
}

func (b *bot) saveBirthday(ctx context.Context, evt *events.Message, args []string) error {
	chat := evt.Info.Chat.ToNonAD()
	now := time.Now()
	day, month, ok := parseBirthdayDate(strings.Join(args, ""), now)
	if !ok {
		return b.SendText(ctx, chat, "🎂 Use assim: !aniversario 25/12 pra salvar o seu, ou !aniversario @pessoa 25/12 pra salvar o de alguém (ou !aniversario @pessoa hoje)")
	}
	person := b.birthdayTarget(ctx, evt)
	if err := b.birthdays.set(ctx, chat, person, day, month, now); err != nil {
		return fmt.Errorf("salvar aniversário: %w", err)
	}
	days := daysUntilBirthday(now, day, month)
	text := fmt.Sprintf("🎂 Anotado! O aniversário de @%s é dia %s (%s), %s 🥳",
		person[0].User, birthdayLabel(day, month), birthdayCountdown(days), birthdayWishWhen(days))
	_, err := b.SendMentions(ctx, chat, text, person[:1])
	return err
}

func (b *bot) listBirthdays(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	var (
		birthdays []birthday
		nicknames map[string]string
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { birthdays, err = b.birthdays.inChat(gctx, chat); return err })
	g.Go(func() error { nicknames = b.ChatNicknames(gctx, chat); return nil })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler aniversários: %w", err)
	}
	if len(birthdays) == 0 {
		return b.SendText(ctx, chat, "🎂 Ninguém salvou aniversário nesta conversa ainda. Salva o seu com !aniversario dd/mm, tipo !aniversario 25/12")
	}

	now := time.Now()
	slices.SortStableFunc(birthdays, func(x, y birthday) int {
		return cmp.Compare(daysUntilBirthday(now, x.day, x.month), daysUntilBirthday(now, y.day, y.month))
	})
	lines := []string{"🎂 *Próximos aniversários*", ""}
	for _, bd := range birthdays[:min(len(birthdays), birthdayListMax)] {
		lines = append(lines, fmt.Sprintf("*%s* · %s (%s)",
			birthdayLabel(bd.day, bd.month), b.DisplayName(ctx, bd.person[0], nicknames), birthdayCountdown(daysUntilBirthday(now, bd.day, bd.month))))
	}
	if extra := len(birthdays) - birthdayListMax; extra > 0 {
		lines = append(lines, fmt.Sprintf("_...e mais %s_", core.CountLabel(extra, "pessoa", "pessoas")))
	}
	lines = append(lines, "", "_Salva o seu com !aniversario dd/mm_")
	return b.SendText(ctx, chat, strings.Join(lines, "\n"))
}

func (b *bot) removeBirthday(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := b.birthdayTarget(ctx, evt)
	allowed, err := b.canRemoveBirthday(ctx, evt, person)
	if err != nil {
		return fmt.Errorf("checar quem pode apagar o aniversário: %w", err)
	}
	if !allowed {
		return b.SendText(ctx, chat, "Só a própria pessoa ou um admin do grupo pode apagar o aniversário dela 🚫")
	}
	removed, err := b.birthdays.remove(ctx, chat, person)
	if err != nil {
		return fmt.Errorf("apagar aniversário: %w", err)
	}
	text := fmt.Sprintf("🗑️ Apaguei o aniversário de @%s", person[0].User)
	if !removed {
		text = fmt.Sprintf("@%s não tem aniversário salvo nesta conversa", person[0].User)
	}
	_, err = b.SendMentions(ctx, chat, text, person[:1])
	return err
}

func (b *bot) birthdayTarget(ctx context.Context, evt *events.Message) []types.JID {
	mentions := core.MentionedJIDs(evt.Message)
	if len(mentions) == 0 {
		return b.SenderIdentities(ctx, evt)
	}
	return b.Identities(ctx, mentions[0].ToNonAD())
}

func (b *bot) canRemoveBirthday(ctx context.Context, evt *events.Message, person []types.JID) (bool, error) {
	sender := core.UsersOf(b.SenderIdentities(ctx, evt))
	if core.ContainsAnyUser(person, sender) || b.IsOwner(ctx, evt) {
		return true, nil
	}
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return false, nil
	}
	group, err := b.Client.GetGroupInfo(ctx, chat)
	if err != nil {
		return false, err
	}
	p, found := core.FindParticipant(group.Participants, sender)
	return found && core.IsGroupAdmin(p), nil
}

func (b *bot) runBirthdays(ctx context.Context) {
	ticker := time.NewTicker(core.ReminderTick)
	defer ticker.Stop()
	for {
		if b.Client.IsLoggedIn() {
			b.sendDueBirthdays(ctx)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (b *bot) sendDueBirthdays(ctx context.Context) {
	due, err := b.birthdays.claimDue(ctx, time.Now())
	if err != nil {
		b.Client.Log.Errorf("procurar aniversários: %v", err)
		return
	}
	byChat := map[types.JID][]birthday{}
	for _, bd := range due {
		byChat[bd.chat] = append(byChat[bd.chat], bd)
	}
	var g errgroup.Group
	g.SetLimit(birthdaySendLimit)
	for chat, birthdays := range byChat {
		g.Go(func() error {
			if err := b.celebrateBirthdays(ctx, chat, birthdays); err != nil {
				b.Client.Log.Errorf("mandar os parabéns em %s: %v", chat, err)
			}
			return nil
		})
	}
	_ = g.Wait()
}

func (b *bot) celebrateBirthdays(ctx context.Context, chat types.JID, birthdays []birthday) error {
	var on bool
	allowed := make([]bool, len(birthdays))
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { on, err = b.Modes.Enabled(gctx, chat, core.ModeUtilities); return err })
	for i, bd := range birthdays {
		g.Go(func() (err error) { allowed[i], err = b.AllowedIn(gctx, chat, bd.person); return err })
	}
	if err := g.Wait(); err != nil || !on {
		return err
	}

	var people []types.JID
	for i, bd := range birthdays {
		if allowed[i] {
			people = append(people, bd.person[0])
		}
	}
	if len(people) == 0 {
		return nil
	}
	_, err := b.SendMentions(ctx, chat, birthdayMessage(people), people)
	return err
}

func birthdayMessage(people []types.JID) string {
	wish := "Parabéns! Muitos anos de vida, saúde e grana no bolso"
	if len(people) > 1 {
		wish = "Parabéns pra vocês! Muitos anos de vida, saúde e grana no bolso"
	}
	return fmt.Sprintf("🎂🎉 *Hoje é aniversário de %s!*\n\n%s 🥳🎈🎁\n\n_Bora todo mundo dar os parabéns!_", core.JoinMentions(people), wish)
}

func birthdayWishWhen(days int) string {
	if days == 0 {
		return "e os parabéns saem aqui já já"
	}
	return "e eu mando os parabéns aqui assim que der meia-noite do dia"
}

func parseBirthdayDate(arg string, now time.Time) (day, month int, ok bool) {
	if core.FoldLetters(arg) == "hoje" {
		local := now.In(core.ReminderZone)
		return local.Day(), int(local.Month()), true
	}
	m := birthdayDatePattern.FindStringSubmatch(arg)
	if m == nil {
		return 0, 0, false
	}
	day, _ = strconv.Atoi(m[1])
	month, _ = strconv.Atoi(m[2])
	check := time.Date(2024, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	return day, month, check.Day() == day && int(check.Month()) == month
}

func birthdayLabel(day, month int) string {
	return fmt.Sprintf("%02d/%02d", day, month)
}

func birthdayCountdown(days int) string {
	labels := map[int]string{0: "é hoje! 🎉", 1: "é amanhã"}
	if label, ok := labels[days]; ok {
		return label
	}
	return "daqui a " + core.CountLabel(days, "dia", "dias")
}

func birthdayLeapYear(year int) bool {
	return time.Date(year, time.December, 31, 0, 0, 0, 0, time.UTC).YearDay() == 366
}

func birthdayIn(year, day, month int) time.Time {
	if month == 2 && day == 29 && !birthdayLeapYear(year) {
		day = 28
	}
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, core.ReminderZone)
}

func daysUntilBirthday(now time.Time, day, month int) int {
	local := now.In(core.ReminderZone)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, core.ReminderZone)
	next := birthdayIn(local.Year(), day, month)
	if next.Before(today) {
		next = birthdayIn(local.Year()+1, day, month)
	}
	return int(next.Sub(today).Hours() / 24)
}

func (s *birthdayStore) set(ctx context.Context, chat types.JID, person []types.JID, day, month int, now time.Time) error {
	first, second := core.IdentityKeys(person)
	res, err := s.db.ExecContext(ctx,
		`UPDATE aniversarios
		 SET ultimo_ano = CASE WHEN dia = ? AND mes = ? THEN ultimo_ano ELSE 0 END, dia = ?, mes = ?
		 WHERE chat = ? AND (pessoa IN (?, ?) OR pessoa_alt IN (?, ?))`,
		day, month, day, month, chat.String(), first, second, first, second,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil || n > 0 {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO aniversarios (chat, pessoa, pessoa_alt, dia, mes, criado_em) VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT (chat, pessoa) DO UPDATE SET dia = excluded.dia, mes = excluded.mes`,
		chat.String(), first, second, day, month, now.Unix(),
	)
	return err
}

func (s *birthdayStore) remove(ctx context.Context, chat types.JID, person []types.JID) (bool, error) {
	first, second := core.IdentityKeys(person)
	res, err := s.db.ExecContext(ctx,
		"DELETE FROM aniversarios WHERE chat = ? AND (pessoa IN (?, ?) OR pessoa_alt IN (?, ?))",
		chat.String(), first, second, first, second,
	)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (s *birthdayStore) inChat(ctx context.Context, chat types.JID) ([]birthday, error) {
	return s.query(ctx, "SELECT id, chat, pessoa, pessoa_alt, dia, mes FROM aniversarios WHERE chat = ?", chat.String())
}

func (s *birthdayStore) claimDue(ctx context.Context, now time.Time) ([]birthday, error) {
	local := now.In(core.ReminderZone)
	leapFallback := local.Month() == time.February && local.Day() == 28 && !birthdayLeapYear(local.Year())
	return s.query(ctx,
		`UPDATE aniversarios SET ultimo_ano = ?
		 WHERE ultimo_ano < ? AND mes = ? AND (dia = ? OR (? AND dia = 29))
		 RETURNING id, chat, pessoa, pessoa_alt, dia, mes`,
		local.Year(), local.Year(), int(local.Month()), local.Day(), leapFallback,
	)
}

func (s *birthdayStore) query(ctx context.Context, statement string, args ...any) (birthdays []birthday, err error) {
	rows, err := s.db.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		bd, err := scanBirthday(rows)
		if err != nil {
			return nil, err
		}
		birthdays = append(birthdays, bd)
	}
	return birthdays, rows.Err()
}

func scanBirthday(rows *sql.Rows) (birthday, error) {
	var (
		bd                      birthday
		chat, person, personAlt string
	)
	if err := rows.Scan(&bd.id, &chat, &person, &personAlt, &bd.day, &bd.month); err != nil {
		return birthday{}, err
	}
	jids, err := parseJIDs([]string{chat, person, personAlt})
	if err != nil {
		return birthday{}, err
	}
	bd.chat, bd.person = jids[0], core.UniqueJIDs(jids[1:])
	return bd, nil
}
