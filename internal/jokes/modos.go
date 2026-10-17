package jokes

import (
	"context"
	"slices"
	"strings"
	"unicode"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types/events"
)

type wordTrigger struct {
	prefix string
	run    core.Handler
}

func (b *bot) answerWords(ctx context.Context, evt *events.Message) error {
	triggers := []wordTrigger{
		{"arrog", b.arrogante},
		{"noj", b.nojo},
		{"aura", b.aura},
		{"ego", b.ego},
		{"chor", b.chorar},
	}
	words := strings.FieldsFunc(core.FoldLetters(core.MessageText(evt.Message)), func(r rune) bool { return !unicode.IsLetter(r) })
	i := slices.IndexFunc(triggers, func(t wordTrigger) bool {
		return slices.ContainsFunc(words, func(word string) bool { return strings.HasPrefix(word, t.prefix) })
	})
	if i < 0 {
		return nil
	}
	on, err := b.Modes.Enabled(ctx, evt.Info.Chat.ToNonAD(), core.ModeWords)
	if err != nil || !on {
		return err
	}
	return triggers[i].run(ctx, evt)
}
