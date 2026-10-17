package core

import (
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MinQuoteLength = 12
	MaxQuoteLength = 300
	minQuoteWords  = 3
)

var (
	QuoteNoise = regexp.MustCompile(`https?://\S+|www\.\S+|@\d+`)
)

type Quote struct {
	Text   string
	SentAt time.Time
}

func QuoteBlock(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	for i, line := range lines {
		lines[i] = "> " + line
	}
	return strings.Join(lines, "\n")
}

func (q Quote) WorthQuoting() bool {
	if utf8.RuneCountInString(q.Text) > MaxQuoteLength {
		return false
	}
	return len(meaningfulWords(q.Text)) >= minQuoteWords
}
