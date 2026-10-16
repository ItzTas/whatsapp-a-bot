package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	emojiDuration    = time.Minute
	emojiHintAfter   = 30 * time.Second
	emojiReward      = 50
	emojiRecentLimit = 20
	emojiMaxBytes    = 200
	emojiMaxAnswer   = 80
	emojiTypoRunes   = 6
)

type emojiPuzzle struct {
	Emojis   string   `json:"emojis"`
	Answer   string   `json:"resposta"`
	Category string   `json:"categoria"`
	Accepted []string `json:"aceitas"`
}

type emojiState struct {
	Puzzle   emojiPuzzle   `json:"puzzle"`
	Progress guessProgress `json:"progress"`
}

type recentAnswers struct {
	mu    sync.Mutex
	items []string
}

func (b *bot) emoji(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if len(b.assistant.tiers) == 0 {
		return b.sendText(ctx, chat, "🧠 A IA tá desligada, ninguém configurou um modelo")
	}
	if b.guessRunning(chat) {
		return b.replyTo(ctx, evt, guessBusy)
	}

	value, _ := b.emojiRecent.LoadOrStore(chat.String(), &recentAnswers{})
	recent := value.(*recentAnswers)
	puzzle, failure := b.emojiPuzzle(ctx, evt, recent.list())
	if failure != "" {
		return b.replyTo(ctx, evt, failure)
	}
	answers := puzzle.answers()
	round := newGuessRound(func(guess string) bool { return matchesTitle(guess, answers) }, nil)
	b.keepEmoji(ctx, chat.String(), round, puzzle)
	if !b.startGuessRound(chat, round) {
		return b.replyTo(ctx, evt, guessBusy)
	}
	recent.add(puzzle.Answer)

	intro := fmt.Sprintf("🎬 *Adivinha pelos emojis!*\n\n%s\n\n🏷️ Categoria: %s\n_Manda o nome. Tem dica em 30 segundos e a resposta sai em 1 minuto_ ⏳",
		puzzle.Emojis, puzzle.Category)
	if err := b.sendText(ctx, chat, intro); err != nil {
		b.abandonGuess(chat, round)
		return err
	}
	round.keeper.keep()
	go b.finishEmoji(ctx, chat, round, puzzle)
	return nil
}

func (b *bot) keepEmoji(ctx context.Context, key string, round *guessRound, puzzle emojiPuzzle) {
	b.keepRound(ctx, &round.keeper, "emoji", key, round.started.Add(emojiDuration), func() any {
		return emojiState{Puzzle: puzzle, Progress: round.progress()}
	})
}

func (b *bot) restoreEmoji(ctx context.Context, saved savedGame) error {
	var state emojiState
	if err := json.Unmarshal(saved.state, &state); err != nil {
		return fmt.Errorf("ler rodada do !emoji: %w", err)
	}
	answers := state.Puzzle.answers()
	round := restoreGuessRound(func(guess string) bool { return matchesTitle(guess, answers) }, nil, state.Progress)
	b.keepEmoji(ctx, saved.key, round, state.Puzzle)
	chat, err := b.resumeGuess(saved, round)
	if err != nil {
		return err
	}
	go b.finishEmoji(ctx, chat, round, state.Puzzle)
	return nil
}

func (b *bot) emojiPuzzle(ctx context.Context, evt *events.Message, avoid []string) (emojiPuzzle, string) {
	chat := evt.Info.Chat
	prompt := emojiPrompt(questionFrom(messageText(evt.Message)), avoid)
	stopThinking := b.thinking(ctx, evt)
	raw, err := b.assistant.ask(ctx, b.iaAsker(ctx, chat, evt.Info.Sender.User), func(int) string { return prompt })
	stopThinking()
	if err != nil {
		b.client.Log.Warnf("!emoji: %v", err)
		return emojiPuzzle{}, iaFailure(err)
	}
	puzzle, err := parseEmojiPuzzle(raw)
	if err != nil {
		b.client.Log.Warnf("!emoji: charada inválida da IA: %v: %q", err, raw)
		return emojiPuzzle{}, "😵 A IA se enrolou nos emojis, tenta de novo"
	}
	return puzzle, ""
}

func (b *bot) finishEmoji(ctx context.Context, chat types.JID, round *guessRound, puzzle emojiPuzzle) {
	hints := []guessHint{{after: emojiHintAfter, text: "💡 Dica: " + titleHint(puzzle.Answer)}}
	if err := b.awaitGuess(ctx, chat, round, emojiDuration, hints); err != nil {
		return
	}
	winner, solved, players := round.outcome()
	if !solved {
		if err := b.sendText(ctx, chat, fmt.Sprintf("%s\n\n⏰ Ninguém acertou! Era *%s*", puzzle.Emojis, puzzle.Answer)); err != nil {
			b.client.Log.Errorf("revelar !emoji: %v", err)
		}
		return
	}
	text := fmt.Sprintf("%s\n\n🎉 @%s acertou! Era *%s*", puzzle.Emojis, winner.User, puzzle.Answer)
	text = appendLine(text, b.reward(ctx, chat, []types.JID{winner}, players, emojiReward))
	if _, err := b.sendMentions(ctx, chat, text, []types.JID{winner}); err != nil {
		b.client.Log.Errorf("revelar !emoji: %v", err)
	}
}

func emojiPrompt(theme string, avoid []string) string {
	if theme == "" {
		themes := []string{
			"filme", "série", "desenho animado", "anime", "música brasileira", "música internacional",
			"novela brasileira", "videogame", "personagem famoso de ficção", "meme brasileiro", "ditado popular brasileiro",
			"comida típica", "programa de TV brasileiro", "super-herói",
		}
		theme = themes[rand.IntN(len(themes))]
	}
	avoidLine := ""
	if len(avoid) > 0 {
		avoidLine = "\nNão use nenhum destes, que já saíram: " + strings.Join(avoid, "; ") + "."
	}
	return fmt.Sprintf(`Escolhe um exemplo bem conhecido no Brasil deste tema: %s. Tem que ser algo que um grupo de amigos jovens conheça.
Descreve ele SÓ com emojis, de 3 a 8, sem letras nem números. Os emojis têm que dar pra adivinhar: represente as palavras do nome ou o que é mais marcante nele.%s
Responda só com um JSON, sem texto antes ou depois e sem bloco de código, neste formato:
{"emojis": "🦁👑", "resposta": "O Rei Leão", "categoria": "filme", "aceitas": ["rei leão", "the lion king"]}
"resposta" é o nome do jeito que é conhecido no Brasil, "categoria" diz o que é em uma ou duas palavras e "aceitas" traz outros nomes que também valem (título original, apelido popular), podendo ser uma lista vazia.`,
		theme, avoidLine)
}

func parseEmojiPuzzle(raw string) (emojiPuzzle, error) {
	start, end := strings.Index(raw, "{"), strings.LastIndex(raw, "}")
	if start < 0 || end < start {
		return emojiPuzzle{}, errors.New("sem JSON na resposta")
	}
	var p emojiPuzzle
	if err := json.Unmarshal([]byte(raw[start:end+1]), &p); err != nil {
		return emojiPuzzle{}, err
	}
	p.Emojis = strings.TrimSpace(p.Emojis)
	p.Answer = strings.TrimSpace(p.Answer)
	p.Category = strings.ToLower(strings.TrimSpace(p.Category))
	return p, p.validate()
}

func (p emojiPuzzle) validate() error {
	if p.Emojis == "" || len(p.Emojis) > emojiMaxBytes {
		return errors.New("emojis vazios ou grandes demais")
	}
	if strings.ContainsFunc(p.Emojis, func(r rune) bool { return r < utf8.RuneSelf && unicode.IsLetter(r) }) {
		return errors.New("emojis com letras")
	}
	if normalizeGuess(p.Answer) == "" || utf8.RuneCountInString(p.Answer) > emojiMaxAnswer {
		return errors.New("resposta vazia ou grande demais")
	}
	if p.Category == "" {
		return errors.New("sem categoria")
	}
	return nil
}

func (p emojiPuzzle) answers() []string {
	var answers []string
	for _, name := range append([]string{p.Answer}, p.Accepted...) {
		title := titleKey(name)
		if title != "" && !slices.Contains(answers, title) {
			answers = append(answers, title)
		}
	}
	return answers
}

func titleKey(text string) string {
	articles := map[string]bool{"o": true, "a": true, "os": true, "as": true, "um": true, "uma": true, "the": true}
	words := strings.Fields(normalizeGuess(text))
	if len(words) > 1 && articles[words[0]] {
		words = words[1:]
	}
	return strings.Join(words, " ")
}

func matchesTitle(guess string, answers []string) bool {
	guess = titleKey(guess)
	return slices.ContainsFunc(answers, func(answer string) bool {
		if strings.Contains(" "+guess+" ", " "+answer+" ") {
			return true
		}
		if strings.ReplaceAll(guess, " ", "") == strings.ReplaceAll(answer, " ", "") {
			return true
		}
		return levenshtein(guess, answer) <= utf8.RuneCountInString(answer)/emojiTypoRunes
	})
}

func levenshtein(a, b string) int {
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

func titleHint(answer string) string {
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
	return fmt.Sprintf("%s (%s)", strings.Join(words, " "), countLabel(len(words), "palavra", "palavras"))
}

func (r *recentAnswers) list() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.items)
}

func (r *recentAnswers) add(answer string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = append(r.items, answer)
	r.items = r.items[max(0, len(r.items)-emojiRecentLimit):]
}
