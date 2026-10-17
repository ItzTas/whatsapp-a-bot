package jokes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const maxReactionRunes = 16

type reactionStore struct {
	*core.RestrictionStore
}

func (s *reactionStore) set(ctx context.Context, chat types.JID, ids []types.JID, emoji string) (err error) {
	first, second := core.IdentityKeys(ids)
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, tx.Rollback())
		}
	}()
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM reacoes WHERE chat = ? AND (id IN (?, ?) OR id_alt IN (?, ?))",
		chat.String(), first, second, first, second,
	); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO reacoes (chat, id, id_alt, emoji) VALUES (?, ?, ?, ?)",
		chat.String(), first, second, emoji,
	); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *reactionStore) emoji(ctx context.Context, chat types.JID, ids []types.JID) (string, error) {
	first, second := core.IdentityKeys(ids)
	var emoji string
	err := s.DB.QueryRowContext(ctx,
		"SELECT emoji FROM reacoes WHERE chat = ? AND (id IN (?, ?) OR id_alt IN (?, ?)) LIMIT 1",
		chat.String(), first, second, first, second,
	).Scan(&emoji)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return emoji, err
}

func isReactionEmoji(s string) bool {
	if utf8.RuneCountInString(s) > maxReactionRunes {
		return false
	}
	return !strings.ContainsFunc(s, func(r rune) bool {
		return r < 0x2000 || unicode.IsLetter(r) || unicode.IsDigit(r)
	})
}

func (b *bot) reactIfMarked(ctx context.Context, evt *events.Message) bool {
	if evt.IsEdit || evt.Info.IsFromMe || !core.Revocable(evt.Message) {
		return false
	}
	chat := evt.Info.Chat.ToNonAD()
	emoji, err := b.reactions.emoji(ctx, chat, b.SenderIdentities(ctx, evt))
	if err != nil {
		b.Client.Log.Warnf("checar reação de %s em %s: %v", evt.Info.Sender, chat, err)
		return false
	}
	if emoji == "" {
		return false
	}
	on, err := b.ChatModeOn(ctx, chat, core.ModeJokes)
	if err != nil {
		b.Client.Log.Warnf("checar modo %s em %s: %v", core.ModeJokes, chat, err)
		return false
	}
	if !on {
		return false
	}
	b.React(ctx, evt, emoji)
	return true
}

func (b *bot) reagir(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	people := slices.DeleteFunc(core.MentionedJIDs(evt.Message), b.IsSelf)
	if len(people) == 0 {
		return b.SendText(ctx, chat, "😜 Marca quem vai levar reação: !reagir @pessoa 😂 (pra parar, !reagir parar @pessoa)")
	}
	words := slices.DeleteFunc(strings.Fields(core.QuestionFrom(core.MessageText(evt.Message))), func(word string) bool {
		return strings.HasPrefix(word, "@")
	})
	if len(words) > 0 && strings.EqualFold(words[0], "parar") {
		return b.stopReacting(ctx, chat, people)
	}
	if len(words) == 0 || !isReactionEmoji(words[0]) {
		return b.SendText(ctx, chat, "😜 Manda o emoji junto: !reagir @pessoa 😂")
	}
	emoji := words[0]
	lines := make([]string, len(people))
	for i, person := range people {
		if err := b.reactions.set(ctx, chat, b.Identities(ctx, person.ToNonAD()), emoji); err != nil {
			return err
		}
		lines[i] = fmt.Sprintf("%s Agora toda mensagem de @%s ganha um %s", emoji, person.User, emoji)
	}
	_, err := b.SendMentions(ctx, chat, strings.Join(lines, "\n"), people)
	return err
}

func (b *bot) stopReacting(ctx context.Context, chat types.JID, people []types.JID) error {
	results := map[bool]string{
		true:  "😮‍💨 Parei de reagir às mensagens de @%s",
		false: "Eu nem tava reagindo às mensagens de @%s",
	}
	lines := make([]string, len(people))
	for i, person := range people {
		removed, err := b.reactions.Remove(ctx, chat, b.Identities(ctx, person.ToNonAD()))
		if err != nil {
			return err
		}
		lines[i] = fmt.Sprintf(results[removed], person.User)
	}
	_, err := b.SendMentions(ctx, chat, strings.Join(lines, "\n"), people)
	return err
}
