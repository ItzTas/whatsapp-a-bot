package main

import (
	"cmp"
	"context"
	"maps"
	"sync"

	"go.mau.fi/whatsmeow/types"
)

func (b *bot) chatNicknames(ctx context.Context, chat types.JID) map[string]string {
	var names, nicknames map[string]string
	var wg sync.WaitGroup
	wg.Go(func() { names = b.chatNames(ctx, chat) })
	wg.Go(func() { nicknames = b.chatOnlyNicknames(ctx, chat) })
	wg.Wait()
	labels := map[string]string{}
	maps.Copy(labels, names)
	maps.Copy(labels, nicknames)
	return labels
}

func (b *bot) chatOnlyNicknames(ctx context.Context, chat types.JID) map[string]string {
	nicknames, err := b.nicknames.inChat(ctx, chat)
	if err != nil {
		b.client.Log.Warnf("ler apelidos de %s: %v", chat, err)
	}
	return nicknames
}

type personNames struct {
	nickname string
	name     string
	whatsapp string
	number   string
}

func (p personNames) label() string {
	return cmp.Or(p.nickname, p.name, p.whatsapp, p.number)
}

func (b *bot) personNamesIn(ctx context.Context, chat, person types.JID) personNames {
	var (
		names, nicknames map[string]string
		identities       []types.JID
		whatsapp         string
	)
	var wg sync.WaitGroup
	wg.Go(func() { names = b.chatNames(ctx, chat) })
	wg.Go(func() { nicknames = b.chatOnlyNicknames(ctx, chat) })
	wg.Go(func() {
		identities = b.identities(ctx, person)
		whatsapp = b.contactName(ctx, identities)
	})
	wg.Wait()
	number := person.User
	for _, id := range identities {
		if id.Server == types.DefaultUserServer {
			number = "+" + id.User
		}
	}
	return personNames{
		nickname: nicknameOf(identities, nicknames),
		name:     nicknameOf(identities, names),
		whatsapp: whatsapp,
		number:   number,
	}
}

func (b *bot) displayName(ctx context.Context, person types.JID, nicknames map[string]string) string {
	identities := b.identities(ctx, person)
	if nickname := nicknameOf(identities, nicknames); nickname != "" {
		return nickname
	}
	if name := b.contactName(ctx, identities); name != "" {
		return name
	}
	for _, id := range identities {
		if id.Server == types.DefaultUserServer {
			return "+" + id.User
		}
	}
	return person.User
}

func (b *bot) fullName(ctx context.Context, person types.JID, nicknames map[string]string) string {
	identities := b.identities(ctx, person)
	nickname, name := nicknameOf(identities, nicknames), b.contactName(ctx, identities)
	if nickname == "" || name == "" || nickname == name {
		return cmp.Or(nickname, name)
	}
	return nickname + " · " + name
}

func nicknameOf(identities []types.JID, nicknames map[string]string) string {
	for _, id := range identities {
		if nickname, ok := nicknames[id.User]; ok {
			return nickname
		}
	}
	return ""
}

func (b *bot) identities(ctx context.Context, person types.JID) []types.JID {
	alt, err := b.client.Store.GetAltJID(ctx, person)
	if err != nil {
		b.client.Log.Warnf("resolver número alternativo de %s: %v", person, err)
	}
	if alt.IsEmpty() {
		return []types.JID{person}
	}
	return []types.JID{person, alt}
}

func (b *bot) contactName(ctx context.Context, identities []types.JID) string {
	for _, id := range identities {
		contact, err := b.client.Store.Contacts.GetContact(ctx, id)
		if err != nil {
			b.client.Log.Warnf("ler contato %s: %v", id, err)
			continue
		}
		if name := cmp.Or(contact.PushName, contact.FullName, contact.FirstName); name != "" {
			return name
		}
	}
	return ""
}
