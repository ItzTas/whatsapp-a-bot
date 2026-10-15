package main

import (
	"context"
	"fmt"
	"hash/fnv"
	"slices"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) ship(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	couple, ok := coupleFrom(evt.Info.Sender, mentionedJIDs(evt.Message))
	if !ok {
		return b.sendText(ctx, chat, "Marca uma ou duas pessoas: !ship @pessoa ou !ship @pessoa1 @pessoa2")
	}

	var (
		aliases [2][]string
		names   [2]string
	)
	nicknames := b.chatNicknames(ctx, chat)
	var wg sync.WaitGroup
	for i, p := range couple {
		wg.Go(func() {
			aliases[i] = b.userAliases(ctx, p)
			names[i] = b.displayName(ctx, p, nicknames)
		})
	}
	wg.Wait()

	identities := [2]string{slices.Min(aliases[0]), slices.Min(aliases[1])}
	percent := shipPercent(identities[0], identities[1])
	lines := []string{"💘 *Ship* 💘", ""}
	header := fmt.Sprintf("@%s + @%s", couple[0].User, couple[1].User)
	if name, ok := coupleName(names[0], names[1]); ok {
		header += fmt.Sprintf(" = *%s*", name)
	}
	lines = append(lines, header, "", fmt.Sprintf("%s %d%%", meterBar(percent), percent), shipVerdict(percent, identities[0] == identities[1]))
	_, err := b.sendMentions(ctx, chat, strings.Join(lines, "\n"), couple[:])
	return err
}

func shipPercent(a, b string) int {
	pair := []string{a, b}
	slices.Sort(pair)
	hash := fnv.New32a()
	hash.Write([]byte("ship|" + strings.Join(pair, "|")))
	return int(hash.Sum32() % 101)
}

func shipVerdict(percent int, selfLove bool) string {
	if selfLove {
		return "Amor próprio é tudo, mas assim já é demais 🪞"
	}
	verdicts := []struct {
		min  int
		text string
	}{
		{95, "Almas gêmeas. Já pode marcar o casamento 💍"},
		{80, "Esse casal tem futuro, só falta alguém tomar coragem 😏"},
		{60, "Tem química. Um date resolve 🍷"},
		{40, "Dá pra tentar, mas sem criar expectativa 🤏"},
		{20, "Melhor ficar só na amizade 🫂"},
		{0, "Nem se fossem as últimas pessoas da Terra 💀"},
	}
	for _, v := range verdicts {
		if percent >= v.min {
			return v.text
		}
	}
	return verdicts[len(verdicts)-1].text
}

func coupleName(a, b string) (string, bool) {
	first, second := firstWord(a), firstWord(b)
	if first == "" || second == "" {
		return "", false
	}
	start := []rune(first)
	end := []rune(strings.ToLower(second))
	return string(start[:(len(start)+1)/2]) + string(end[len(end)/2:]), true
}

func firstWord(name string) string {
	for _, word := range strings.Fields(name) {
		letters := strings.TrimFunc(word, func(r rune) bool { return !unicode.IsLetter(r) })
		if utf8.RuneCountInString(letters) >= 2 {
			return letters
		}
	}
	return ""
}
