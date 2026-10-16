package main

import (
	"context"
	"hash/fnv"
	"slices"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) euanime(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := firstMentionOrSender(evt).ToNonAD()
	seed := fixedCharacterSeed(chat, b.userAliases(ctx, person))
	rank := int(seed % characterPool)

	characters, err := fetchCharacters(ctx, 1+rank/characterPageSize, []int{})
	if err != nil {
		b.client.Log.Warnf("!euanime: buscar personagens: %v", err)
		return b.replyTo(ctx, evt, anilistDown)
	}
	index := rank % characterPageSize
	if index >= len(characters) {
		return b.replyTo(ctx, evt, anilistEmpty)
	}
	return b.sendCharacter(ctx, evt, person, characterMatch{
		character: characters[index],
		reason:    characterReasons[seed/characterPool%uint32(len(characterReasons))],
	})
}

func fixedCharacterSeed(chat types.JID, aliases []string) uint32 {
	hash := fnv.New32a()
	hash.Write([]byte("euanime|" + chat.User + "|" + slices.Min(aliases)))
	return hash.Sum32()
}
