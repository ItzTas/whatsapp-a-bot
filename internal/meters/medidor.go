package meters

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

type history struct {
	texts    []string
	activity core.ChatActivity
}

type meter struct {
	emoji    string
	subject  string
	evidence func(h history) float64
}

func (b *bot) medidor(ctx context.Context, evt *events.Message) error {
	subject := meterSubject(core.MessageText(evt.Message))
	if subject == "" {
		return b.SendText(ctx, evt.Info.Chat, "Diz o que medir: !medidor <coisa> @pessoa")
	}
	return b.sendMeter(ctx, evt, meter{"📊", subject, matchRate(subjectTerms(subject)...)})
}

func (b *bot) meter(m meter) core.Handler {
	return func(ctx context.Context, evt *events.Message) error {
		return b.sendMeter(ctx, evt, m)
	}
}

func gadoMeter() meter {
	return meter{"🐂", "gado", matchRate(
		"lind", "maravilhos", "princes", "rainha", "gostos", "perfeit", "deusa", "musa", "gata", "gatinh",
		"amor", "mozao", "fofa", "te amo", "saudade", "bom dia", "boa noite", "sonhe com",
		"❤", "😍", "🥰", "😘", "🥺", "💕", "💖", "🌹",
	)}
}

func cornoMeter() meter {
	return meter{"🦌", "corno", matchRate(
		"chifr", "trai", "corn", "galhud", "minha ex ", "meu ex ", "vacuo", "visualizou", "ciume",
		"cade ela", "cade ele", "terminou comigo", "me deixou", "solteir", "sofrend", "fiel",
		"🦌", "💔", "😭", "🥲",
	)}
}

func gayMeter() meter {
	return meter{"🏳️‍🌈", "gay", matchRate(
		"amiga", "miga", "diva", "divo", "babado", "bafo", "close", "arras", "lacr", "mona", "bicha",
		"viad", "gay", "tombou", "bee ", "maravilhos",
		"🏳️‍🌈", "💅", "✨", "💖",
	)}
}

func chatoMeter() meter {
	complaints := matchRate("aff", "pqp", "que saco", "ninguem perguntou", "ngm perguntou", "🙄", "😒", "😤")
	return meter{"🙄", "chato", func(h history) float64 {
		return anyOf(complaints(h), rateOf(h.texts, isShouting), rateOf(h.texts, core.IsCommand), yapping(h))
	}}
}

func (b *bot) sendMeter(ctx context.Context, evt *events.Message, m meter) error {
	people := core.UniqueJIDs(core.MentionedJIDs(evt.Message))
	if len(people) == 0 {
		people = []types.JID{evt.Info.Sender.ToNonAD()}
	}

	chat := evt.Info.Chat.ToNonAD()
	aliases := make([][]string, len(people))
	texts := make([][]string, len(people))
	var activity core.ChatActivity
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { activity, err = b.Messages.Activity(gctx, chat); return err })
	for i, p := range people {
		g.Go(func() (err error) {
			aliases[i] = b.UserAliases(gctx, p)
			texts[i], err = b.Messages.Texts(gctx, chat, aliases[i])
			return err
		})
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler mensagens: %w", err)
	}

	lines := []string{fmt.Sprintf("%s *Medidor de %s*", m.emoji, m.subject)}
	mentioned := make([]string, len(people))
	for i, p := range people {
		percent := m.measure(chat, aliases[i], history{texts: texts[i], activity: activity})
		lines = append(lines, "",
			fmt.Sprintf("@%s é %d%% %s", p.User, percent, m.subject),
			fmt.Sprintf("%s %d%%", core.MeterBar(percent), percent),
			core.MeterBasis(len(texts[i])),
		)
		mentioned[i] = p.String()
	}

	_, err := b.Client.SendMessage(ctx, evt.Info.Chat, &waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{
		Text:        proto.String(b.WithNicknames(ctx, chat, strings.Join(lines, "\n"))),
		ContextInfo: &waE2E.ContextInfo{MentionedJID: mentioned},
	}})
	return err
}

func (m meter) measure(chat types.JID, aliases []string, h history) int {
	hash := fnv.New32a()
	hash.Write([]byte(chat.User + "|" + slices.Min(aliases) + "|" + strings.ToLower(m.subject)))
	baseline := float64(hash.Sum32() % 41)

	confidence := min(float64(len(h.texts))/20, 1)
	signal := m.evidence(h) * confidence
	return int(math.Round(baseline + (100-baseline)*signal))
}

func matchRate(terms ...string) func(history) float64 {
	return func(h history) float64 {
		texts := slices.DeleteFunc(slices.Clone(h.texts), core.IsCommand)
		return rateOf(texts, func(text string) bool { return containsAnyTerm(text, terms) })
	}
}

func rateOf(texts []string, match func(string) bool) float64 {
	if len(texts) == 0 {
		return 0
	}
	hits := 0
	for _, t := range texts {
		if match(t) {
			hits++
		}
	}
	return 1 - math.Exp(-8*float64(hits)/float64(len(texts)))
}

func anyOf(scores ...float64) float64 {
	miss := 1.0
	for _, s := range scores {
		miss *= 1 - s
	}
	return 1 - miss
}

func yapping(h history) float64 {
	if h.activity.Messages == 0 || h.activity.Senders < 2 {
		return 0
	}
	fairShare := float64(h.activity.Messages) / float64(h.activity.Senders)
	return min(max((float64(len(h.texts))/fairShare-1)/3, 0), 1)
}

func isShouting(text string) bool {
	letters := 0
	for _, r := range text {
		if !unicode.IsLetter(r) {
			continue
		}
		if unicode.IsLower(r) {
			return false
		}
		letters++
	}
	return letters >= 6
}

func containsAnyTerm(text string, terms []string) bool {
	words := " " + strings.Join(core.NormalizedWords(text), " ") + " "
	return slices.ContainsFunc(terms, func(term string) bool {
		if strings.ContainsFunc(term, unicode.IsLetter) {
			return strings.Contains(words, " "+term)
		}
		return strings.Contains(text, term)
	})
}

func subjectTerms(subject string) []string {
	words := core.NormalizedWords(subject)
	for i, w := range words {
		if utf8.RuneCountInString(w) > 4 {
			_, last := utf8.DecodeLastRuneInString(w)
			words[i] = w[:len(w)-last]
		}
	}
	return words
}

func meterSubject(text string) string {
	fields := strings.Fields(text)
	words := make([]string, 0, len(fields))
	for _, f := range fields[1:] {
		if strings.HasPrefix(f, "@") {
			continue
		}
		words = append(words, f)
	}
	return strings.Join(words, " ")
}
