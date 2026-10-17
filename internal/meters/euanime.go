package meters

import (
	"context"
	"hash/fnv"
	"slices"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/ai"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) euanime(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := core.FirstMentionOrSender(evt).ToNonAD()
	seed := fixedCharacterSeed(chat, b.UserAliases(ctx, person))
	rank := int(seed % ai.CharacterPool)

	characters, err := ai.FetchCharacters(ctx, 1+rank/ai.CharacterPageSize, []int{})
	if err != nil {
		b.Client.Log.Warnf("!euanime: buscar personagens: %v", err)
		return b.ReplyTo(ctx, evt, ai.AnilistDown)
	}
	index := rank % ai.CharacterPageSize
	if index >= len(characters) {
		return b.ReplyTo(ctx, evt, ai.AnilistEmpty)
	}
	return b.ai.SendCharacter(ctx, evt, person, ai.CharacterMatch{
		Character: characters[index],
		Reason:    ai.CharacterReasons[seed/ai.CharacterPool%uint32(len(ai.CharacterReasons))],
	})
}

func fixedCharacterSeed(chat types.JID, aliases []string) uint32 {
	hash := fnv.New32a()
	hash.Write([]byte("euanime|" + chat.User + "|" + slices.Min(aliases)))
	return hash.Sum32()
}
