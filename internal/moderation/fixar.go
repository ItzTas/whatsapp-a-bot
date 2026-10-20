package moderation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	pinDefault                   = 7 * 24 * time.Hour
	pinMax                       = 365 * 24 * time.Hour
	pinForever     time.Duration = 0
	pinRenewMargin               = time.Hour
	pinRetry                     = time.Hour
	pinSendLimit                 = 4
)

var pinLengthPattern = regexp.MustCompile(`^(\d{1,4})(m|min|minutos?|h|horas?|d|dias?|sem|semanas?)$`)

type pin struct {
	chat   types.JID
	id     types.MessageID
	sender types.JID
	until  time.Time
	renew  time.Time
}

type pinStore struct {
	db *sql.DB
}

func (b *bot) fixar(ctx context.Context, evt *events.Message) error {
	text := core.MessageText(evt.Message)
	length, used, ok := parsePinLength(strings.Fields(text)[1:])
	if !ok {
		return b.SendText(ctx, evt.Info.Chat, fmt.Sprintf("Não entendi o tempo 😅 Use de 1min até %d dias, tipo !fixar 2h, !fixar 3d ou !fixar 2sem, ou !fixar sempre pra deixar fixada pra sempre", int(pinMax.Hours()/24)))
	}
	m, problem, err := b.pinGroup(ctx, evt, "fixar")
	if err != nil {
		return err
	}
	if problem != "" {
		return b.SendText(ctx, evt.Info.Chat, problem)
	}

	p, quoted := quotedPin(m.chat, evt)
	body := core.TextAfterFields(text, 1+used)
	if !quoted && body == "" {
		return b.SendText(ctx, m.chat, "📌 Responda a mensagem que você quer fixar com !fixar (ou !fixar 2h, !fixar sempre...). Sem responder nada, manda !fixar <texto> que eu mando o texto e fixo")
	}
	if !quoted {
		id, err := b.SendRawMentions(ctx, m.chat, body, core.MentionedJIDs(evt.Message))
		if err != nil {
			return fmt.Errorf("mandar a mensagem pra fixar: %w", err)
		}
		p = pin{chat: m.chat, id: id, sender: b.Client.Store.GetJID()}
	}

	now := time.Now()
	if length != pinForever {
		p.until = now.Add(length)
	}
	p, whatsappLength := p.scheduled(now)
	err = b.sendPin(ctx, p, whatsappLength)
	if errors.Is(err, whatsmeow.ErrServerReturnedError) {
		return b.ReplyTo(ctx, evt, "😅 O WhatsApp não deixou fixar essa mensagem. Pode ser que ela já tenha sido apagada ou seja antiga demais")
	}
	if err != nil {
		return fmt.Errorf("fixar mensagem: %w", err)
	}
	if err := b.pins.save(ctx, p); err != nil {
		return fmt.Errorf("salvar mensagem fixada: %w", err)
	}
	return b.ReplyTo(ctx, evt, pinnedMessage(p, length))
}

func (b *bot) desfixar(ctx context.Context, evt *events.Message) error {
	m, problem, err := b.pinGroup(ctx, evt, "desfixar")
	if err != nil {
		return err
	}
	if problem != "" {
		return b.SendText(ctx, evt.Info.Chat, problem)
	}
	p, quoted := quotedPin(m.chat, evt)
	if !quoted {
		return b.SendText(ctx, m.chat, "📌 Responda a mensagem fixada com !desfixar")
	}
	err = b.sendUnpin(ctx, p)
	if errors.Is(err, whatsmeow.ErrServerReturnedError) {
		return b.ReplyTo(ctx, evt, "😅 O WhatsApp não deixou desfixar essa mensagem")
	}
	if err != nil {
		return fmt.Errorf("desfixar mensagem: %w", err)
	}
	if _, err := b.pins.remove(ctx, p.chat, p.id); err != nil {
		return fmt.Errorf("esquecer mensagem fixada: %w", err)
	}
	return b.ReplyTo(ctx, evt, "📌 Desfixei a mensagem")
}

func (b *bot) pinGroup(ctx context.Context, evt *events.Message, name string) (moderation, string, error) {
	m, problem, err := b.moderateGroup(ctx, evt, name, true)
	if err != nil || problem != "" {
		return moderation{}, problem, err
	}
	if m.group.IsLocked && !b.botIsAdmin(m.group) {
		return moderation{}, fmt.Sprintf("Eu preciso ser admin do grupo pra %s mensagem, porque aqui só admin mexe nas configurações 😅", name), nil
	}
	return m, "", nil
}

func quotedPin(chat types.JID, evt *events.Message) (pin, bool) {
	info := evt.Message.GetExtendedTextMessage().GetContextInfo()
	sender, err := types.ParseJID(info.GetParticipant())
	if info.GetStanzaID() == "" || err != nil {
		return pin{}, false
	}
	return pin{chat: chat, id: info.GetStanzaID(), sender: sender}, true
}

func parsePinLength(args []string) (length time.Duration, used int, ok bool) {
	if len(args) == 0 {
		return pinDefault, 0, true
	}
	first := core.FoldLetters(args[0])
	if (first == "pra" || first == "para") && len(args) > 1 && core.FoldLetters(args[1]) == "sempre" {
		return pinForever, 2, true
	}
	forever := map[string]bool{"sempre": true, "permanente": true, "eterno": true}
	if forever[first] {
		return pinForever, 1, true
	}
	match := pinLengthPattern.FindStringSubmatch(first)
	if match == nil {
		return pinDefault, 0, true
	}
	units := map[string]time.Duration{
		"m": time.Minute, "min": time.Minute, "minuto": time.Minute,
		"h": time.Hour, "hora": time.Hour,
		"d": 24 * time.Hour, "dia": 24 * time.Hour,
		"sem": 7 * 24 * time.Hour, "semana": 7 * 24 * time.Hour,
	}
	n, _ := strconv.Atoi(match[1])
	length = time.Duration(n) * units[strings.TrimSuffix(match[2], "s")]
	return length, 1, length >= time.Minute && length <= pinMax
}

func whatsappPinLength(remaining time.Duration) time.Duration {
	lengths := []time.Duration{24 * time.Hour, 7 * 24 * time.Hour, 30 * 24 * time.Hour}
	for _, l := range lengths {
		if remaining != pinForever && remaining <= l {
			return l
		}
	}
	return lengths[len(lengths)-1]
}

func (p pin) scheduled(now time.Time) (pin, time.Duration) {
	remaining := pinForever
	if !p.until.IsZero() {
		remaining = p.until.Sub(now)
	}
	length := whatsappPinLength(remaining)
	p.renew = now.Add(length - pinRenewMargin)
	if !p.until.IsZero() && !p.until.After(now.Add(length)) {
		p.renew = p.until
	}
	return p, length
}

func (p pin) ended(now time.Time) bool {
	return !p.until.IsZero() && !p.until.After(now)
}

func pinnedMessage(p pin, length time.Duration) string {
	if length == pinForever {
		return "📌 Fixei a mensagem pra sempre! O WhatsApp só deixa fixar por até 30 dias, então eu refixo sozinho antes de vencer. Pra tirar, responda ela com !desfixar"
	}
	until := p.until.In(core.BrazilTime)
	return fmt.Sprintf("📌 Fixei a mensagem por %s. Ela sai sozinha %s às %s", pinLengthLabel(length), core.BrazilianDate(until), until.Format("15:04"))
}

func pinLengthLabel(d time.Duration) string {
	day := 24 * time.Hour
	if d%day == 0 {
		return core.CountLabel(int(d/day), "dia", "dias")
	}
	return core.FormatDuration(d)
}

func (b *bot) sendPin(ctx context.Context, p pin, length time.Duration) error {
	return b.PinMessage(ctx, p.chat, p.sender, p.id, length)
}

func (b *bot) sendUnpin(ctx context.Context, p pin) error {
	return b.UnpinMessage(ctx, p.chat, p.sender, p.id)
}

func (b *bot) forgetUnpinned(ctx context.Context, evt *events.Message) bool {
	change := evt.Message.GetPinInChatMessage()
	if change.GetType() != waE2E.PinInChatMessage_UNPIN_FOR_ALL {
		return false
	}
	if _, err := b.pins.remove(ctx, evt.Info.Chat.ToNonAD(), change.GetKey().GetID()); err != nil {
		b.Client.Log.Errorf("esquecer a mensagem %s que desfixaram em %s: %v", change.GetKey().GetID(), evt.Info.Chat, err)
	}
	return false
}

func (b *bot) runPins(ctx context.Context) {
	ticker := time.NewTicker(core.ReminderTick)
	defer ticker.Stop()
	for {
		if b.Client.IsLoggedIn() {
			b.refreshPins(ctx)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (b *bot) refreshPins(ctx context.Context) {
	now := time.Now()
	due, err := b.pins.due(ctx, now)
	if err != nil {
		b.Client.Log.Errorf("procurar mensagens fixadas: %v", err)
		return
	}
	var g errgroup.Group
	g.SetLimit(pinSendLimit)
	for _, p := range due {
		g.Go(func() error {
			if err := b.refreshPin(ctx, p, now); err != nil {
				b.Client.Log.Errorf("cuidar da mensagem fixada %s em %s: %v", p.id, p.chat, err)
			}
			return nil
		})
	}
	_ = g.Wait()
}

func (b *bot) refreshPin(ctx context.Context, p pin, now time.Time) error {
	if p.ended(now) {
		_, err := b.pins.remove(ctx, p.chat, p.id)
		return errors.Join(b.sendUnpin(ctx, p), err)
	}
	next, length := p.scheduled(now)
	err := b.sendPin(ctx, next, length)
	if errors.Is(err, whatsmeow.ErrServerReturnedError) {
		b.Client.Log.Warnf("o WhatsApp não deixou refixar a mensagem %s em %s, então parei de cuidar dela: %v", p.id, p.chat, err)
		_, err = b.pins.remove(ctx, p.chat, p.id)
		return err
	}
	if err != nil {
		next.renew = now.Add(pinRetry)
		return errors.Join(fmt.Errorf("refixar: %w", err), b.pins.reschedule(ctx, next))
	}
	return b.pins.reschedule(ctx, next)
}

func (s *pinStore) save(ctx context.Context, p pin) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO mensagens_fixadas (chat, mensagem, remetente, ate, renovar_em) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT (chat, mensagem) DO UPDATE SET remetente = excluded.remetente, ate = excluded.ate, renovar_em = excluded.renovar_em`,
		p.chat.String(), p.id, p.sender.ToNonAD().String(), pinUntilUnix(p.until), p.renew.Unix(),
	)
	return err
}

func (s *pinStore) reschedule(ctx context.Context, p pin) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE mensagens_fixadas SET renovar_em = ? WHERE chat = ? AND mensagem = ?",
		p.renew.Unix(), p.chat.String(), p.id,
	)
	return err
}

func (s *pinStore) remove(ctx context.Context, chat types.JID, id types.MessageID) (bool, error) {
	res, err := s.db.ExecContext(ctx, "DELETE FROM mensagens_fixadas WHERE chat = ? AND mensagem = ?", chat.String(), id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (s *pinStore) due(ctx context.Context, now time.Time) (pins []pin, err error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT chat, mensagem, remetente, ate, renovar_em FROM mensagens_fixadas WHERE renovar_em <= ?",
		now.Unix(),
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		p, err := scanPin(rows)
		if err != nil {
			return nil, err
		}
		pins = append(pins, p)
	}
	return pins, rows.Err()
}

func scanPin(rows *sql.Rows) (pin, error) {
	var (
		p            pin
		chat, sender string
		until, renew int64
	)
	if err := rows.Scan(&chat, &p.id, &sender, &until, &renew); err != nil {
		return pin{}, err
	}
	var err error
	if p.chat, err = types.ParseJID(chat); err != nil {
		return pin{}, err
	}
	if p.sender, err = types.ParseJID(sender); err != nil {
		return pin{}, err
	}
	if until > 0 {
		p.until = time.Unix(until, 0)
	}
	p.renew = time.Unix(renew, 0)
	return p, nil
}

func pinUntilUnix(until time.Time) int64 {
	if until.IsZero() {
		return 0
	}
	return until.Unix()
}
