package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	welcomeMaxLength = 1000
	welcomeDelay     = 10 * time.Second
	welcomePerson    = "{pessoa}"
	welcomeGroup     = "{grupo}"
)

type welcomeSetting struct {
	text   string
	author types.JID
}

type welcomeBatch struct {
	mu     sync.Mutex
	done   bool
	people []types.JID
}

type welcomeStore struct {
	db *sql.DB
}

func (b *bot) boasvindas(ctx context.Context, evt *events.Message) error {
	text := textAfterFields(messageText(evt.Message), 1)
	if text == "" {
		return b.showWelcome(ctx, evt)
	}
	removal := []string{"desligar", "apagar", "remover", "tirar", "off"}
	if slices.Contains(removal, strings.ToLower(text)) {
		return b.removeWelcome(ctx, evt)
	}
	return b.setWelcome(ctx, evt, text)
}

func (b *bot) showWelcome(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return b.sendText(ctx, chat, "O !boasvindas só funciona em grupo")
	}
	setting, err := b.welcomes.get(ctx, chat)
	if err != nil {
		return fmt.Errorf("ler boas-vindas: %w", err)
	}
	if setting.text == "" {
		return b.sendText(ctx, chat, "👋 Este grupo não tem mensagem de boas-vindas. Um admin cria com !boasvindas <texto>, tipo !boasvindas Bem-vindo(a) ao {grupo}, {pessoa}! Lê as regras aí")
	}
	return b.sendText(ctx, chat, strings.Join([]string{
		"👋 *Boas-vindas deste grupo*",
		"",
		setting.text,
		"",
		"✍️ Definida por " + b.displayName(ctx, setting.author, b.chatNicknames(ctx, chat)),
		"_Muda com !boasvindas <texto> ou apaga com !boasvindas desligar_",
	}, "\n"))
}

func (b *bot) setWelcome(ctx context.Context, evt *events.Message, text string) error {
	m, problem, err := b.moderateGroup(ctx, evt, "boasvindas", true)
	if err != nil {
		return err
	}
	if problem != "" {
		return b.sendText(ctx, evt.Info.Chat, problem)
	}
	if utf8.RuneCountInString(text) > welcomeMaxLength {
		return b.sendText(ctx, m.chat, fmt.Sprintf("Mensagem grande demais, o máximo é %d letras", welcomeMaxLength))
	}
	if err := b.welcomes.set(ctx, m.chat, text, b.senderIdentities(ctx, evt), time.Now()); err != nil {
		return fmt.Errorf("salvar boas-vindas: %w", err)
	}
	return b.replyTo(ctx, evt, strings.Join([]string{
		"👋 Pronto! Quando alguém entrar no grupo eu mando assim:",
		"",
		renderWelcome(text, "@fulano", m.group.Name),
		"",
		"_Só funciona com o modo moderacao ligado. Use {pessoa} pra marcar quem entrou e {grupo} pro nome do grupo_",
	}, "\n"))
}

func (b *bot) removeWelcome(ctx context.Context, evt *events.Message) error {
	m, problem, err := b.moderateGroup(ctx, evt, "boasvindas", true)
	if err != nil {
		return err
	}
	if problem != "" {
		return b.sendText(ctx, evt.Info.Chat, problem)
	}
	removed, err := b.welcomes.remove(ctx, m.chat)
	if err != nil {
		return fmt.Errorf("apagar boas-vindas: %w", err)
	}
	if !removed {
		return b.sendText(ctx, m.chat, "Este grupo nem tinha mensagem de boas-vindas")
	}
	return b.sendText(ctx, m.chat, "🗑️ Apaguei a mensagem de boas-vindas. Quem entrar agora não ganha recepção")
}

func renderWelcome(template, mention, groupName string) string {
	if !strings.Contains(template, welcomePerson) {
		template = welcomePerson + " " + template
	}
	return strings.NewReplacer(welcomePerson, mention, welcomeGroup, groupName).Replace(template)
}

func (b *bot) welcome(ctx context.Context, evt *events.GroupInfo) {
	own := usersOf(b.ownIdentities())
	joined := make([]types.JID, 0, len(evt.Join))
	for _, person := range evt.Join {
		if !slices.Contains(own, person.User) {
			joined = append(joined, person.ToNonAD())
		}
	}
	if len(joined) == 0 {
		return
	}
	chat := evt.JID.ToNonAD()
	if !b.queueWelcome(chat, joined) {
		return
	}

	timer := time.NewTimer(welcomeDelay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	}
	if err := b.sendWelcome(ctx, chat, b.takeWelcomeBatch(chat)); err != nil {
		b.client.Log.Errorf("mandar boas-vindas em %s: %v", chat, err)
	}
}

func (b *bot) queueWelcome(chat types.JID, people []types.JID) bool {
	for {
		actual, loaded := b.welcomeBatches.LoadOrStore(chat, &welcomeBatch{})
		batch := actual.(*welcomeBatch)
		batch.mu.Lock()
		if batch.done {
			batch.mu.Unlock()
			continue
		}
		batch.people = append(batch.people, people...)
		batch.mu.Unlock()
		return !loaded
	}
}

func (b *bot) takeWelcomeBatch(chat types.JID) []types.JID {
	actual, found := b.welcomeBatches.LoadAndDelete(chat)
	if !found {
		return nil
	}
	batch := actual.(*welcomeBatch)
	batch.mu.Lock()
	defer batch.mu.Unlock()
	batch.done = true
	return uniqueJIDs(batch.people)
}

func (b *bot) sendWelcome(ctx context.Context, chat types.JID, people []types.JID) error {
	if len(people) == 0 {
		return nil
	}
	var (
		setting   welcomeSetting
		groupName string
		on        bool
		allowed   []types.JID
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) {
		setting, err = b.welcomes.get(gctx, chat)
		if err != nil || !strings.Contains(setting.text, welcomeGroup) {
			return err
		}
		groupName = b.welcomeGroupName(gctx, chat)
		return nil
	})
	g.Go(func() (err error) { on, err = b.chatModeOn(gctx, chat, modeModeration); return err })
	g.Go(func() (err error) { allowed, err = b.welcomeAllowed(gctx, chat, people); return err })
	if err := g.Wait(); err != nil {
		return err
	}
	if setting.text == "" || !on || len(allowed) == 0 {
		return nil
	}
	_, err := b.sendMentions(ctx, chat, renderWelcome(setting.text, joinMentions(allowed), groupName), allowed)
	return err
}

func (b *bot) welcomeGroupName(ctx context.Context, chat types.JID) string {
	info, err := b.client.GetGroupInfo(ctx, chat)
	if err != nil {
		b.client.Log.Warnf("ler nome do grupo %s pras boas-vindas: %v", chat, err)
		return "o grupo"
	}
	return info.Name
}

func (b *bot) welcomeAllowed(ctx context.Context, chat types.JID, people []types.JID) ([]types.JID, error) {
	verdicts := make([]bool, len(people))
	g, gctx := errgroup.WithContext(ctx)
	for i, person := range people {
		g.Go(func() (err error) { verdicts[i], err = b.allowedIn(gctx, chat, b.identities(gctx, person)); return err })
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	allowed := make([]types.JID, 0, len(people))
	for i, person := range people {
		if verdicts[i] {
			allowed = append(allowed, person)
		}
	}
	return allowed, nil
}

func (s *welcomeStore) set(ctx context.Context, chat types.JID, text string, author []types.JID, now time.Time) error {
	authorID, authorAlt := identityKeys(author)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO boas_vindas (chat, texto, autor, autor_alt, atualizada_em) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT (chat) DO UPDATE SET texto = excluded.texto, autor = excluded.autor,
		   autor_alt = excluded.autor_alt, atualizada_em = excluded.atualizada_em`,
		chat.String(), text, authorID, authorAlt, now.Unix(),
	)
	return err
}

func (s *welcomeStore) get(ctx context.Context, chat types.JID) (welcomeSetting, error) {
	var setting welcomeSetting
	var author string
	err := s.db.QueryRowContext(ctx, "SELECT texto, autor FROM boas_vindas WHERE chat = ?", chat.String()).Scan(&setting.text, &author)
	if errors.Is(err, sql.ErrNoRows) {
		return welcomeSetting{}, nil
	}
	if err != nil {
		return welcomeSetting{}, err
	}
	setting.author, err = types.ParseJID(author)
	return setting, err
}

func (s *welcomeStore) remove(ctx context.Context, chat types.JID) (bool, error) {
	res, err := s.db.ExecContext(ctx, "DELETE FROM boas_vindas WHERE chat = ?", chat.String())
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}
