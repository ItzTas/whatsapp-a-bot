package core

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

func PlainArgs(text string) []string {
	fields := strings.Fields(strings.ToLower(text))
	return slices.DeleteFunc(fields[min(1, len(fields)):], func(f string) bool { return strings.HasPrefix(f, "@") })
}

func AppendLine(text, line string) string {
	if line == "" {
		return text
	}
	return text + "\n" + line
}

func FirstPlainArg(text string) string {
	fields := strings.Fields(text)
	i := slices.IndexFunc(fields[min(1, len(fields)):], func(f string) bool { return !strings.HasPrefix(f, "@") })
	if i < 0 {
		return ""
	}
	return strings.ToLower(fields[i+1])
}

func Thousands(n int64) string {
	digits := strconv.FormatInt(n, 10)
	var sb strings.Builder
	for i, d := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			sb.WriteByte('.')
		}
		sb.WriteRune(d)
	}
	return sb.String()
}

func Levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := range ra {
		curr := make([]int, len(rb)+1)
		curr[0] = i + 1
		for j := range rb {
			cost := 1
			if ra[i] == rb[j] {
				cost = 0
			}
			curr[j+1] = min(prev[j+1]+1, curr[j]+1, prev[j]+cost)
		}
		prev = curr
	}
	return prev[len(rb)]
}

func FoldLetters(s string) string {
	return accentReplacer.Replace(strings.ToLower(s))
}

func meaningfulWords(text string) []string {
	filler := map[string]bool{
		"oi": true, "ola": true, "opa": true, "eae": true, "eai": true, "salve": true,
		"bom": true, "boa": true, "dia": true, "tarde": true, "noite": true,
		"ok": true, "okay": true, "blz": true, "beleza": true, "sim": true, "nao": true,
		"s": true, "n": true, "ss": true, "nn": true, "kk": true, "e": true, "ne": true,
		"ta": true, "to": true, "vlw": true, "valeu": true, "obg": true,
		"obrigado": true, "obrigada": true, "tmj": true,
	}
	stripped := QuoteNoise.ReplaceAllString(FoldLetters(text), " ")
	words := strings.FieldsFunc(stripped, func(r rune) bool { return !unicode.IsLetter(r) })
	return slices.DeleteFunc(words, func(w string) bool { return filler[w] || IsLaughWord(w) })
}

func QuestionFrom(text string) string {
	text = strings.TrimSpace(text)
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(text, fields[0]))
}

func TruncateRunes(text string, limit int) string {
	if utf8.RuneCountInString(text) <= limit {
		return text
	}
	return string([]rune(text)[:limit]) + "…"
}

func TextAfterFields(text string, n int) string {
	for range n {
		text = strings.TrimLeftFunc(text, unicode.IsSpace)
		i := strings.IndexFunc(text, unicode.IsSpace)
		if i < 0 {
			return ""
		}
		text = text[i:]
	}
	return strings.TrimSpace(text)
}

func MeterBasis(messages int) string {
	if messages == 0 {
		return "_ainda não vi nenhuma mensagem dessa pessoa aqui_"
	}
	if messages == 1 {
		return "_com base em 1 mensagem_"
	}
	return fmt.Sprintf("_com base em %d mensagens_", messages)
}

func IsCommand(text string) bool {
	return strings.HasPrefix(strings.TrimSpace(text), "!")
}

var accentReplacer = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "é", "e", "ê", "e", "í", "i",
	"ó", "o", "ô", "o", "õ", "o", "ú", "u", "ü", "u", "ç", "c",
)

func NormalizedWords(text string) []string {
	return strings.FieldsFunc(accentReplacer.Replace(strings.ToLower(text)), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

func MeterBar(percent int) string {
	filled := percent / 10
	return strings.Repeat("▓", filled) + strings.Repeat("░", 10-filled)
}

const (
	minWordLength = 3
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

type spelledWord struct {
	Key      string
	Spelling string
}

func TextWords(text string) []spelledWord {
	var words []spelledWord
	for _, word := range strings.FieldsFunc(strings.ToLower(urlPattern.ReplaceAllString(text, " ")), isWordSeparator) {
		if key, ok := wordKey(word); ok {
			words = append(words, spelledWord{Key: key, Spelling: word})
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
	if IsLaugh(word) {
		return "kkkk", true
	}
	key := accentReplacer.Replace(word)
	if utf8.RuneCountInString(key) < minWordLength || stopWords[key] {
		return "", false
	}
	return key, true
}

func IsLaugh(word string) bool {
	return len(word) >= 3 && strings.Trim(word, "k") == ""
}

func MostCommonSpelling(spellings map[string]int) string {
	best, bestCount := "", 0
	for spelling, count := range spellings {
		if count > bestCount || (count == bestCount && spelling < best) {
			best, bestCount = spelling, count
		}
	}
	return best
}

func CountLabel(n int, singular, plural string) string {
	if n == 1 {
		return "1 " + singular
	}
	return fmt.Sprintf("%d %s", n, plural)
}

func JoinWithAnd(items []string) string {
	return joinWith(items, " e ")
}

func JoinWithOr(items []string) string {
	return joinWith(items, " ou ")
}

func joinWith(items []string, word string) string {
	last := len(items) - 1
	if last < 0 {
		return ""
	}
	if last == 0 {
		return items[0]
	}
	return strings.Join(items[:last], ", ") + word + items[last]
}
