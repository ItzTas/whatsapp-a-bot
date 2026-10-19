package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/games"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	emojiDuration  = time.Minute
	emojiHintAfter = 30 * time.Second
	emojiReward    = 50
	emojiMaxBytes  = 200
	emojiMaxAnswer = 80
)

type emojiPuzzle struct {
	Emojis   string   `json:"emojis"`
	Answer   string   `json:"resposta"`
	Category string   `json:"categoria"`
	Accepted []string `json:"aceitas"`
}

type emojiState struct {
	Puzzle   emojiPuzzle         `json:"puzzle"`
	Progress games.GuessProgress `json:"progress"`
}

func (b *bot) emoji(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if !b.Assistant.Enabled() {
		return b.SendText(ctx, chat, "🧠 A IA tá desligada, ninguém configurou um modelo")
	}
	if b.games.GuessRunning(chat) {
		return b.ReplyTo(ctx, evt, games.GuessBusy)
	}

	value, _ := b.emojiRecent.LoadOrStore(chat.String(), &games.RecentAnswers{})
	recent := value.(*games.RecentAnswers)
	puzzle, failure := b.emojiPuzzle(ctx, evt, recent.List())
	if failure != "" {
		return b.ReplyTo(ctx, evt, failure)
	}
	answers := puzzle.answers()
	round := games.NewGuessRound(func(guess string) bool { return games.MatchesTitle(guess, answers) }, nil)
	b.keepEmoji(ctx, chat.String(), round, puzzle)
	if !b.games.StartGuessRound(chat, round) {
		return b.ReplyTo(ctx, evt, games.GuessBusy)
	}
	recent.Add(puzzle.Answer)

	intro := fmt.Sprintf("🎬 *Adivinha pelos emojis!*\n\n%s\n\n🏷️ Categoria: %s\n_Manda o nome. Tem dica em 30 segundos e a resposta sai em 1 minuto_ ⏳",
		puzzle.Emojis, puzzle.Category)
	if err := b.SendText(ctx, chat, intro); err != nil {
		b.games.AbandonGuess(chat, round)
		return err
	}
	round.Keeper.Keep()
	go b.finishEmoji(ctx, chat, round, puzzle)
	return nil
}

func (b *bot) keepEmoji(ctx context.Context, key string, round *games.GuessRound, puzzle emojiPuzzle) {
	b.games.KeepRound(ctx, &round.Keeper, "emoji", key, round.Started.Add(emojiDuration), func() any {
		return emojiState{Puzzle: puzzle, Progress: round.Progress()}
	})
}

func (b *bot) restoreEmoji(ctx context.Context, saved core.SavedGame) error {
	var state emojiState
	if err := json.Unmarshal(saved.State, &state); err != nil {
		return fmt.Errorf("ler rodada do !emoji: %w", err)
	}
	answers := state.Puzzle.answers()
	round := games.RestoreGuessRound(func(guess string) bool { return games.MatchesTitle(guess, answers) }, nil, state.Progress)
	b.keepEmoji(ctx, saved.Key, round, state.Puzzle)
	chat, err := b.games.ResumeGuess(saved, round)
	if err != nil {
		return err
	}
	go b.finishEmoji(ctx, chat, round, state.Puzzle)
	return nil
}

func (b *bot) emojiPuzzle(ctx context.Context, evt *events.Message, avoid []string) (emojiPuzzle, string) {
	chat := evt.Info.Chat
	prompt := emojiPrompt(core.QuestionFrom(core.MessageText(evt.Message)), avoid)
	stopThinking := b.thinking(ctx, evt)
	raw, err := b.Assistant.Ask(ctx, b.Asker(ctx, chat, evt.Info.Sender.User), func(int) string { return prompt })
	stopThinking()
	if err != nil {
		b.Client.Log.Warnf("!emoji: %v", err)
		return emojiPuzzle{}, iaFailure(err)
	}
	puzzle, err := parseEmojiPuzzle(raw)
	if err != nil {
		b.Client.Log.Warnf("!emoji: charada inválida da IA: %v: %q", err, raw)
		return emojiPuzzle{}, "😵 A IA se enrolou nos emojis, tenta de novo"
	}
	return puzzle, ""
}

func (b *bot) finishEmoji(ctx context.Context, chat types.JID, round *games.GuessRound, puzzle emojiPuzzle) {
	defer b.Rescue(ctx, chat)
	hints := []games.GuessHint{{After: emojiHintAfter, Text: "💡 Dica: " + games.TitleHint(puzzle.Answer)}}
	if err := b.games.AwaitGuess(ctx, chat, round, emojiDuration, hints); err != nil {
		return
	}
	winner, solved, players := round.Outcome()
	if !solved {
		if err := b.SendText(ctx, chat, fmt.Sprintf("%s\n\n⏰ Ninguém acertou! Era *%s*", puzzle.Emojis, puzzle.Answer)); err != nil {
			b.Client.Log.Errorf("revelar !emoji: %v", err)
		}
		return
	}
	text := fmt.Sprintf("%s\n\n🎉 @%s acertou! Era *%s*", puzzle.Emojis, winner.User, puzzle.Answer)
	text = core.AppendLine(text, b.economy.Reward(ctx, chat, []types.JID{winner}, players, emojiReward))
	if _, err := b.SendMentions(ctx, chat, text, []types.JID{winner}); err != nil {
		b.Client.Log.Errorf("revelar !emoji: %v", err)
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
	if games.NormalizeGuess(p.Answer) == "" || utf8.RuneCountInString(p.Answer) > emojiMaxAnswer {
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
		title := games.TitleKey(name)
		if title != "" && !slices.Contains(answers, title) {
			answers = append(answers, title)
		}
	}
	return answers
}
