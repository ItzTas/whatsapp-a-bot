package romance

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	almaMinMessages = 20
	almaRunnersUp   = 2
	almaSharedWords = 3
)

type vocabulary struct {
	counts   map[string]int
	messages int
}

type soulmate struct {
	jid        types.JID
	similarity float64
	shared     []string
}

func (b *bot) almagemea(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return b.SendText(ctx, chat, "O !almagemea só funciona em grupo")
	}
	person := core.FirstMentionOrSender(evt).ToNonAD()

	var (
		group   *types.GroupInfo
		texts   []core.SenderText
		aliases []string
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { group, err = b.Client.GetGroupInfo(gctx, chat); return err })
	g.Go(func() (err error) { texts, err = b.Messages.TextsBySender(gctx, chat); return err })
	g.Go(func() error { aliases = b.UserAliases(gctx, person); return nil })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("procurar alma gêmea: %w", err)
	}

	owners := b.MemberOwners(group.Participants)
	vocabs, spellings := vocabularies(owners, texts)
	self, found := ownerOfAliases(owners, aliases)
	if _, talked := vocabs[self]; !found || !talked {
		text := fmt.Sprintf("🤐 @%s ainda não falou o bastante pra eu achar a alma gêmea", person.User)
		_, err := b.SendMentions(ctx, chat, text, []types.JID{person})
		return err
	}
	matches := soulmates(self, vocabs)
	if len(matches) == 0 {
		text := fmt.Sprintf("🥲 Ninguém mais aqui falou o bastante pra comparar com @%s", person.User)
		_, err := b.SendMentions(ctx, chat, text, []types.JID{person})
		return err
	}

	text, mentions := soulmateText(person, matches, spellings)
	_, err := b.SendMentions(ctx, chat, text, mentions)
	return err
}

func ownerOfAliases(owners core.MemberOwners, aliases []string) (types.JID, bool) {
	for _, alias := range aliases {
		if owner, ok := owners[alias]; ok {
			return owner, true
		}
	}
	return types.JID{}, false
}

func vocabularies(owners core.MemberOwners, texts []core.SenderText) (map[types.JID]*vocabulary, map[string]map[string]int) {
	vocabs := map[types.JID]*vocabulary{}
	spellings := map[string]map[string]int{}
	for _, t := range texts {
		owner, ok := owners.Of(t.Sender, t.SenderAlt)
		if !ok || core.IsCommand(t.Text) {
			continue
		}
		v := vocabs[owner]
		if v == nil {
			v = &vocabulary{counts: map[string]int{}}
			vocabs[owner] = v
		}
		v.messages++
		for _, w := range core.TextWords(t.Text) {
			v.counts[w.Key]++
			if spellings[w.Key] == nil {
				spellings[w.Key] = map[string]int{}
			}
			spellings[w.Key][w.Spelling]++
		}
	}
	for owner, v := range vocabs {
		if v.messages < almaMinMessages {
			delete(vocabs, owner)
		}
	}
	return vocabs, spellings
}

func soulmates(self types.JID, vocabs map[types.JID]*vocabulary) []soulmate {
	weights := wordWeights(vocabs)
	mine := weights[self]
	matches := make([]soulmate, 0, len(weights))
	for owner, theirs := range weights {
		if owner == self {
			continue
		}
		matches = append(matches, soulmate{jid: owner, similarity: cosine(mine, theirs), shared: sharedWords(mine, theirs)})
	}
	slices.SortFunc(matches, func(a, b soulmate) int {
		return cmp.Or(cmp.Compare(b.similarity, a.similarity), strings.Compare(a.jid.User, b.jid.User))
	})
	return matches[:min(len(matches), almaRunnersUp+1)]
}

func wordWeights(vocabs map[types.JID]*vocabulary) map[types.JID]map[string]float64 {
	usedBy := map[string]int{}
	for _, v := range vocabs {
		for key := range v.counts {
			usedBy[key]++
		}
	}
	people := float64(len(vocabs))
	weights := make(map[types.JID]map[string]float64, len(vocabs))
	for owner, v := range vocabs {
		w := make(map[string]float64, len(v.counts))
		for key, count := range v.counts {
			w[key] = (1 + math.Log(float64(count))) * math.Log(1+people/float64(usedBy[key]))
		}
		weights[owner] = w
	}
	return weights
}

func cosine(a, b map[string]float64) float64 {
	var dot, normA, normB float64
	for key, wa := range a {
		dot += wa * b[key]
		normA += wa * wa
	}
	for _, wb := range b {
		normB += wb * wb
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / math.Sqrt(normA*normB)
}

func sharedWords(a, b map[string]float64) []string {
	var shared []string
	for key := range a {
		if _, ok := b[key]; ok {
			shared = append(shared, key)
		}
	}
	score := func(key string) float64 { return min(a[key], b[key]) }
	slices.SortFunc(shared, func(x, y string) int {
		return cmp.Or(cmp.Compare(score(y), score(x)), strings.Compare(x, y))
	})
	return shared[:min(len(shared), almaSharedWords)]
}

func soulmatePercent(similarity float64) int {
	return int(math.Round(math.Sqrt(similarity) * 100))
}

func soulmateVerdict(percent int) string {
	verdicts := []struct {
		min  int
		text string
	}{
		{85, "Vocês escrevem igualzinho, até os erros de português combinam. É o destino 💍"},
		{70, "Os vocabulários de vocês se completam. Já pode marcar o primeiro date 🍷"},
		{55, "Tem sintonia, dá pra puxar assunto e ver no que dá 😏"},
		{40, "Uma faísca tímida, mas toda história de amor começa assim ✨"},
		{0, "Falam línguas diferentes, mas o amor supera tudo... talvez 🥲"},
	}
	for _, v := range verdicts {
		if percent >= v.min {
			return v.text
		}
	}
	return verdicts[len(verdicts)-1].text
}

func soulmateText(person types.JID, matches []soulmate, spellings map[string]map[string]int) (string, []types.JID) {
	best := matches[0]
	percent := soulmatePercent(best.similarity)
	lines := []string{
		fmt.Sprintf("💞 *Alma gêmea de @%s*", person.User),
		"",
		fmt.Sprintf("💘 @%s, com %d%% de sintonia", best.jid.User, percent),
		soulmateVerdict(percent),
	}
	if len(best.shared) > 0 {
		words := make([]string, len(best.shared))
		for i, key := range best.shared {
			words[i] = "*" + core.MostCommonSpelling(spellings[key]) + "*"
		}
		lines = append(lines, "", "🗣️ Vocês dois vivem falando "+core.JoinWithAnd(words))
	}
	mentions := []types.JID{person, best.jid}
	if len(matches) > 1 {
		runnersUp := make([]string, 0, len(matches)-1)
		for _, m := range matches[1:] {
			runnersUp = append(runnersUp, fmt.Sprintf("@%s (%d%%)", m.jid.User, soulmatePercent(m.similarity)))
			mentions = append(mentions, m.jid)
		}
		lines = append(lines, "", "💫 Quase lá: "+strings.Join(runnersUp, ", "))
	}
	return strings.Join(lines, "\n"), mentions
}
