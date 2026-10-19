package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	feedbackMaxLength = 1000
	feedbackShown     = 20
)

type feedback struct {
	id      int64
	author  []types.JID
	text    string
	created time.Time
}

type feedbackStore struct {
	db *sql.DB
}

func (b *Bot) feedback(ctx context.Context, evt *events.Message) error {
	text := MessageText(evt.Message)
	fields := strings.Fields(text)
	if len(fields) < 2 {
		return b.ReplyTo(ctx, evt, "💬 Manda !feedback <ideia> pra sugerir o que adicionar ou mudar no bot, tipo !feedback um jogo de truco. Pra ver o que a galera já mandou, é !feedback ver")
	}
	actions := map[string]Handler{
		"ver":     b.listFeedbacks,
		"lista":   b.listFeedbacks,
		"apagar":  b.removeFeedback,
		"remover": b.removeFeedback,
	}
	if action, ok := actions[FoldLetters(fields[1])]; ok && len(fields) <= 3 {
		return action(ctx, evt)
	}
	return b.addFeedback(ctx, evt, TextAfterFields(text, 1))
}

func (b *Bot) addFeedback(ctx context.Context, evt *events.Message, text string) error {
	if utf8.RuneCountInString(text) > feedbackMaxLength {
		return b.ReplyTo(ctx, evt, fmt.Sprintf("Feedback grande demais, o máximo é %d letras", feedbackMaxLength))
	}
	id, err := b.feedbacks.add(ctx, evt.Info.Chat.ToNonAD(), b.SenderIdentities(ctx, evt), text, time.Now())
	if err != nil {
		return fmt.Errorf("salvar feedback: %w", err)
	}
	return b.ReplyTo(ctx, evt, fmt.Sprintf("💬 Valeu! Anotei o feedback *#%d* pro dono do bot dar uma olhada\n\n_Pra ver o que a galera já sugeriu, manda !feedback ver_", id))
}

func (b *Bot) listFeedbacks(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	var (
		feedbacks []feedback
		total     int
		nicknames map[string]string
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { feedbacks, total, err = b.feedbacks.latest(gctx, chat, feedbackShown); return err })
	g.Go(func() error { nicknames = b.ChatNicknames(gctx, chat); return nil })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler feedbacks: %w", err)
	}
	if total == 0 {
		return b.ReplyTo(ctx, evt, "💬 Ninguém mandou feedback nesta conversa ainda. Seja o primeiro: !feedback <ideia>")
	}

	lines := []string{"💬 *Feedbacks desta conversa*", ""}
	for _, f := range feedbacks {
		lines = append(lines,
			fmt.Sprintf("*#%d* · %s · %s", f.id, b.DisplayName(ctx, f.author[0], nicknames), BrazilianDate(f.created)),
			f.text,
			"",
		)
	}
	if total > len(feedbacks) {
		lines = append(lines, fmt.Sprintf("_Mostrando os %d mais recentes de %d_", len(feedbacks), total))
	}
	lines = append(lines, "_Manda o seu com !feedback <ideia>. Quem mandou (ou o dono do bot) apaga com !feedback apagar <número>_")
	return b.SendText(ctx, chat, strings.Join(lines, "\n"))
}

func (b *Bot) removeFeedback(ctx context.Context, evt *events.Message) error {
	id, err := strconv.ParseInt(strings.TrimPrefix(TextAfterFields(MessageText(evt.Message), 2), "#"), 10, 64)
	if err != nil {
		return b.ReplyTo(ctx, evt, "Diz o número do feedback: !feedback apagar 3 (os números aparecem no !feedback ver)")
	}
	f, err := b.feedbacks.get(ctx, evt.Info.Chat.ToNonAD(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return b.ReplyTo(ctx, evt, fmt.Sprintf("Não tem feedback número %d nesta conversa. Confere no !feedback ver", id))
	}
	if err != nil {
		return fmt.Errorf("ler feedback: %w", err)
	}
	if !ContainsAnyUser(f.author, UsersOf(b.SenderIdentities(ctx, evt))) && !b.IsOwner(ctx, evt) {
		return b.ReplyTo(ctx, evt, "Só quem mandou o feedback ou o dono do bot pode apagar ele 🚫")
	}
	if err := b.feedbacks.remove(ctx, id); err != nil {
		return fmt.Errorf("apagar feedback: %w", err)
	}
	return b.ReplyTo(ctx, evt, fmt.Sprintf("🗑️ Apaguei o feedback *#%d*", id))
}

func (s *feedbackStore) add(ctx context.Context, chat types.JID, author []types.JID, text string, now time.Time) (int64, error) {
	key, alt := IdentityKeys(author)
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO feedbacks (chat, autor, autor_alt, texto, criado_em) VALUES (?, ?, ?, ?, ?)",
		chat.String(), key, alt, text, now.Unix(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *feedbackStore) latest(ctx context.Context, chat types.JID, limit int) (feedbacks []feedback, total int, err error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, autor, autor_alt, texto, criado_em, COUNT(*) OVER ()
		 FROM feedbacks WHERE chat = ? ORDER BY id DESC LIMIT ?`,
		chat.String(), limit,
	)
	if err != nil {
		return nil, 0, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		f, err := scanFeedback(rows, &total)
		if err != nil {
			return nil, 0, err
		}
		feedbacks = append(feedbacks, f)
	}
	return feedbacks, total, rows.Err()
}

func (s *feedbackStore) get(ctx context.Context, chat types.JID, id int64) (feedback, error) {
	return scanFeedback(s.db.QueryRowContext(ctx,
		"SELECT id, autor, autor_alt, texto, criado_em FROM feedbacks WHERE id = ? AND chat = ?",
		id, chat.String(),
	))
}

func scanFeedback(row interface{ Scan(...any) error }, extra ...any) (feedback, error) {
	var (
		f           feedback
		author, alt string
		created     int64
	)
	if err := row.Scan(append([]any{&f.id, &author, &alt, &f.text, &created}, extra...)...); err != nil {
		return feedback{}, err
	}
	jids := make([]types.JID, 0, 2)
	for _, raw := range slices.Compact([]string{author, alt}) {
		jid, err := types.ParseJID(raw)
		if err != nil {
			return feedback{}, err
		}
		jids = append(jids, jid)
	}
	f.author, f.created = jids, time.Unix(created, 0)
	return f, nil
}

func (s *feedbackStore) remove(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM feedbacks WHERE id = ?", id)
	return err
}
