package stats

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	favoriteWordCount = 5
)

type wordCount struct {
	word  string
	count int
}

func (b *bot) palavra(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := core.FirstMentionOrSender(evt)
	texts, err := b.messages.Texts(ctx, chat, b.UserAliases(ctx, person))
	if err != nil {
		return fmt.Errorf("ler mensagens: %w", err)
	}
	texts = slices.DeleteFunc(texts, core.IsCommand)

	words := favoriteWords(texts)
	if len(words) == 0 {
		_, err := b.SendMentions(ctx, chat, fmt.Sprintf("@%s ainda não falou o bastante pra eu saber 🤐", person.User), []types.JID{person})
		return err
	}

	lines := []string{fmt.Sprintf("🔤 *Palavras favoritas de @%s*", person.User), ""}
	for i, w := range words {
		lines = append(lines, fmt.Sprintf("%d. *%s* (%dx)", i+1, w.word, w.count))
	}
	lines = append(lines, "", core.MeterBasis(len(texts)))
	_, err = b.SendMentions(ctx, chat, strings.Join(lines, "\n"), []types.JID{person})
	return err
}

func favoriteWords(texts []string) []wordCount {
	totals := map[string]int{}
	spellings := map[string]map[string]int{}
	for _, text := range texts {
		for _, w := range core.TextWords(text) {
			totals[w.Key]++
			if spellings[w.Key] == nil {
				spellings[w.Key] = map[string]int{}
			}
			spellings[w.Key][w.Spelling]++
		}
	}

	words := make([]wordCount, 0, len(totals))
	for key, count := range totals {
		words = append(words, wordCount{word: core.MostCommonSpelling(spellings[key]), count: count})
	}
	slices.SortFunc(words, func(a, b wordCount) int {
		if a.count != b.count {
			return b.count - a.count
		}
		return strings.Compare(a.word, b.word)
	})
	return words[:min(len(words), favoriteWordCount)]
}
