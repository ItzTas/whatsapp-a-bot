package main

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	favoriteWordCount = 5
	minWordLength     = 3
)

var urlPattern = regexp.MustCompile(`(?i)\b(?:https?://|www\.)\S+`)

var stopWords = map[string]bool{
	"que": true, "nao": true, "voce": true, "pra": true, "para": true, "com": true, "mas": true,
	"uma": true, "uns": true, "umas": true, "dos": true, "das": true, "nos": true, "nas": true,
	"isso": true, "isto": true, "esse": true, "essa": true, "este": true, "esta": true, "aquele": true,
	"aquela": true, "aqui": true, "ali": true, "mais": true, "muito": true, "muita": true, "tem": true,
	"foi": true, "ser": true, "ter": true, "vai": true, "vou": true, "sim": true, "entao": true,
	"quando": true, "como": true, "por": true, "porque": true, "pois": true, "ate": true, "tava": true,
	"estou": true, "estava": true, "era": true, "ainda": true, "sei": true, "bem": true, "agora": true,
	"tambem": true, "mim": true, "meu": true, "minha": true, "seu": true, "sua": true, "ele": true,
	"ela": true, "eles": true, "elas": true, "nem": true, "pode": true, "fazer": true, "faz": true,
	"tudo": true, "nada": true, "algo": true, "dele": true, "dela": true, "pro": true, "pela": true,
	"pelo": true, "num": true, "numa": true, "sem": true, "sobre": true, "depois": true, "antes": true,
	"onde": true, "qual": true, "quem": true, "acho": true, "ver": true, "fica": true, "ficar": true,
	"tipo": true, "tbm": true, "vcs": true, "voces": true, "gente": true, "eae": true, "the": true,
}

type wordCount struct {
	word  string
	count int
}

func (b *bot) palavra(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := firstMentionOrSender(evt)
	texts, err := b.messages.texts(ctx, chat, b.userAliases(ctx, person))
	if err != nil {
		return fmt.Errorf("ler mensagens: %w", err)
	}
	texts = slices.DeleteFunc(texts, isCommand)

	words := favoriteWords(texts)
	if len(words) == 0 {
		_, err := b.sendMentions(ctx, chat, fmt.Sprintf("@%s ainda não falou o bastante pra eu saber 🤐", person.User), []types.JID{person})
		return err
	}

	lines := []string{fmt.Sprintf("🔤 *Palavras favoritas de @%s*", person.User), ""}
	for i, w := range words {
		lines = append(lines, fmt.Sprintf("%d. *%s* (%dx)", i+1, w.word, w.count))
	}
	lines = append(lines, "", meterBasis(len(texts)))
	_, err = b.sendMentions(ctx, chat, strings.Join(lines, "\n"), []types.JID{person})
	return err
}

func favoriteWords(texts []string) []wordCount {
	totals := map[string]int{}
	spellings := map[string]map[string]int{}
	for _, text := range texts {
		for _, w := range textWords(text) {
			totals[w.key]++
			if spellings[w.key] == nil {
				spellings[w.key] = map[string]int{}
			}
			spellings[w.key][w.spelling]++
		}
	}

	words := make([]wordCount, 0, len(totals))
	for key, count := range totals {
		words = append(words, wordCount{word: mostCommonSpelling(spellings[key]), count: count})
	}
	slices.SortFunc(words, func(a, b wordCount) int {
		if a.count != b.count {
			return b.count - a.count
		}
		return strings.Compare(a.word, b.word)
	})
	return words[:min(len(words), favoriteWordCount)]
}

type spelledWord struct {
	key      string
	spelling string
}

func textWords(text string) []spelledWord {
	var words []spelledWord
	for _, word := range strings.FieldsFunc(strings.ToLower(urlPattern.ReplaceAllString(text, " ")), isWordSeparator) {
		if key, ok := wordKey(word); ok {
			words = append(words, spelledWord{key: key, spelling: word})
		}
	}
	return words
}

func isWordSeparator(r rune) bool {
	return !unicode.IsLetter(r) && !unicode.IsDigit(r)
}

func wordKey(word string) (string, bool) {
	if !strings.ContainsFunc(word, unicode.IsLetter) {
		return "", false
	}
	if isLaugh(word) {
		return "kkkk", true
	}
	key := accentReplacer.Replace(word)
	if utf8.RuneCountInString(key) < minWordLength || stopWords[key] {
		return "", false
	}
	return key, true
}

func isLaugh(word string) bool {
	return len(word) >= 3 && strings.Trim(word, "k") == ""
}

func mostCommonSpelling(spellings map[string]int) string {
	best, bestCount := "", 0
	for spelling, count := range spellings {
		if count > bestCount || (count == bestCount && spelling < best) {
			best, bestCount = spelling, count
		}
	}
	return best
}
