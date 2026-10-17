package core

import (
	"context"
	"slices"

	"go.mau.fi/whatsmeow/types"
)

type Account struct {
	Key string
	JID types.JID
}

func (b *Bot) Account(ctx context.Context, person types.JID) Account {
	person = person.ToNonAD()
	return Account{Key: slices.Min(b.UserAliases(ctx, person)), JID: person}
}

func IdentityKeys(ids []types.JID) (string, string) {
	return ids[0].String(), ids[len(ids)-1].String()
}
