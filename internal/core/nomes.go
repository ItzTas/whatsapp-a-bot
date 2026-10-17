package core

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"strings"
	"sync"

	"go.mau.fi/whatsmeow/types"
)

const MaxNicknameLength = 40

type labelStore struct {
	db     *sql.DB
	table  string
	column string
}

func (s *labelStore) Set(ctx context.Context, chat types.JID, users []string, label string) (err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, tx.Rollback())
		}
	}()
	for _, user := range users {
		_, err := tx.ExecContext(ctx,
			fmt.Sprintf(`INSERT INTO %[1]s (chat, usuario, %[2]s) VALUES (?, ?, ?)
			 ON CONFLICT (chat, usuario) DO UPDATE SET %[2]s = excluded.%[2]s`, s.table, s.column),
			chat.ToNonAD().String(), user, label,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *labelStore) Remove(ctx context.Context, chat types.JID, users []string) error {
	first, second := users[0], users[len(users)-1]
	_, err := s.db.ExecContext(ctx,
		fmt.Sprintf("DELETE FROM %s WHERE chat = ? AND usuario IN (?, ?)", s.table),
		chat.ToNonAD().String(), first, second,
	)
	return err
}

func (s *labelStore) inChat(ctx context.Context, chat types.JID) (labels map[string]string, err error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("SELECT usuario, %s FROM %s WHERE chat = ?", s.column, s.table), chat.ToNonAD().String())
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	labels = map[string]string{}
	for rows.Next() {
		var user, label string
		if err := rows.Scan(&user, &label); err != nil {
			return nil, err
		}
		labels[user] = label
	}
	return labels, rows.Err()
}

func NicknameFrom(text string) string {
	fields := strings.Fields(MentionTagPattern.ReplaceAllString(text, ""))
	return strings.Join(fields[1:], " ")
}

func (b *Bot) WithNicknames(ctx context.Context, chat types.JID, text string) string {
	return TagNicknames(text, b.ChatNicknames(ctx, chat))
}

func TagNicknames(text string, nicknames map[string]string) string {
	return MentionTagPattern.ReplaceAllStringFunc(text, func(tag string) string {
		nickname, ok := nicknames[strings.TrimPrefix(tag, "@")]
		if !ok {
			return tag
		}
		return fmt.Sprintf("%s (%s)", tag, nickname)
	})
}

func (b *Bot) UserAliases(ctx context.Context, person types.JID) []string {
	alt, err := b.Client.Store.GetAltJID(ctx, person)
	if err != nil {
		b.Client.Log.Warnf("resolver número alternativo de %s: %v", person, err)
	}
	if alt.User == "" {
		return []string{person.User}
	}
	return []string{person.User, alt.User}
}

func (b *Bot) ChatNames(ctx context.Context, chat types.JID) map[string]string {
	names, err := b.Names.inChat(ctx, chat)
	if err != nil {
		b.Client.Log.Warnf("ler nomes de %s: %v", chat, err)
	}
	return names
}

func (b *Bot) ChatNicknames(ctx context.Context, chat types.JID) map[string]string {
	var names, nicknames map[string]string
	var wg sync.WaitGroup
	wg.Go(func() { names = b.ChatNames(ctx, chat) })
	wg.Go(func() { nicknames = b.chatOnlyNicknames(ctx, chat) })
	wg.Wait()
	labels := map[string]string{}
	maps.Copy(labels, names)
	maps.Copy(labels, nicknames)
	return labels
}

func (b *Bot) chatOnlyNicknames(ctx context.Context, chat types.JID) map[string]string {
	nicknames, err := b.Nicknames.inChat(ctx, chat)
	if err != nil {
		b.Client.Log.Warnf("ler apelidos de %s: %v", chat, err)
	}
	return nicknames
}

type PersonNames struct {
	Nickname string
	Name     string
	Whatsapp string
	Number   string
}

func (p PersonNames) Label() string {
	return cmp.Or(p.Nickname, p.Name, p.Whatsapp, p.Number)
}

func (b *Bot) PersonNamesIn(ctx context.Context, chat, person types.JID) PersonNames {
	var (
		names, nicknames map[string]string
		identities       []types.JID
		whatsapp         string
	)
	var wg sync.WaitGroup
	wg.Go(func() { names = b.ChatNames(ctx, chat) })
	wg.Go(func() { nicknames = b.chatOnlyNicknames(ctx, chat) })
	wg.Go(func() {
		identities = b.Identities(ctx, person)
		whatsapp = b.ContactName(ctx, identities)
	})
	wg.Wait()
	number := person.User
	for _, id := range identities {
		if id.Server == types.DefaultUserServer {
			number = "+" + id.User
		}
	}
	return PersonNames{
		Nickname: NicknameOf(identities, nicknames),
		Name:     NicknameOf(identities, names),
		Whatsapp: whatsapp,
		Number:   number,
	}
}

func (b *Bot) DisplayName(ctx context.Context, person types.JID, nicknames map[string]string) string {
	identities := b.Identities(ctx, person)
	if nickname := NicknameOf(identities, nicknames); nickname != "" {
		return nickname
	}
	if name := b.ContactName(ctx, identities); name != "" {
		return name
	}
	for _, id := range identities {
		if id.Server == types.DefaultUserServer {
			return "+" + id.User
		}
	}
	return person.User
}

func (b *Bot) FullName(ctx context.Context, person types.JID, nicknames map[string]string) string {
	identities := b.Identities(ctx, person)
	nickname, name := NicknameOf(identities, nicknames), b.ContactName(ctx, identities)
	if nickname == "" || name == "" || nickname == name {
		return cmp.Or(nickname, name)
	}
	return nickname + " · " + name
}

func NicknameOf(identities []types.JID, nicknames map[string]string) string {
	for _, id := range identities {
		if nickname, ok := nicknames[id.User]; ok {
			return nickname
		}
	}
	return ""
}

func (b *Bot) Identities(ctx context.Context, person types.JID) []types.JID {
	alt, err := b.Client.Store.GetAltJID(ctx, person)
	if err != nil {
		b.Client.Log.Warnf("resolver número alternativo de %s: %v", person, err)
	}
	if alt.IsEmpty() {
		return []types.JID{person}
	}
	return []types.JID{person, alt}
}

func (b *Bot) ContactName(ctx context.Context, identities []types.JID) string {
	for _, id := range identities {
		contact, err := b.Client.Store.Contacts.GetContact(ctx, id)
		if err != nil {
			b.Client.Log.Warnf("ler contato %s: %v", id, err)
			continue
		}
		if name := cmp.Or(contact.PushName, contact.FullName, contact.FirstName); name != "" {
			return name
		}
	}
	return ""
}
