package games

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
)

const (
	emojiRecentLimit = 20
	emojiTypoRunes   = 6
)

type RecentAnswers struct {
	mu    sync.Mutex
	items []string
}

func TitleKey(text string) string {
	articles := map[string]bool{"o": true, "a": true, "os": true, "as": true, "um": true, "uma": true, "the": true}
	words := strings.Fields(NormalizeGuess(text))
	if len(words) > 1 && articles[words[0]] {
		words = words[1:]
	}
	return strings.Join(words, " ")
}

func MatchesTitle(guess string, answers []string) bool {
	guess = TitleKey(guess)
	return slices.ContainsFunc(answers, func(answer string) bool {
		if strings.Contains(" "+guess+" ", " "+answer+" ") {
			return true
		}
		if strings.ReplaceAll(guess, " ", "") == strings.ReplaceAll(answer, " ", "") {
			return true
		}
		return core.Levenshtein(guess, answer) <= utf8.RuneCountInString(answer)/emojiTypoRunes
	})
}

func TitleHint(answer string) string {
	words := strings.Fields(answer)
	for i, word := range words {
		masked := []rune(word)
		seenLetter := false
		for j, r := range masked {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
				continue
			}
			if seenLetter {
				masked[j] = '▁'
			}
			seenLetter = true
		}
		words[i] = string(masked)
	}
	return fmt.Sprintf("%s (%s)", strings.Join(words, " "), core.CountLabel(len(words), "palavra", "palavras"))
}

func (r *RecentAnswers) List() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.items)
}

func (r *RecentAnswers) Add(answer string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = append(r.items, answer)
	r.items = r.items[max(0, len(r.items)-emojiRecentLimit):]
}
