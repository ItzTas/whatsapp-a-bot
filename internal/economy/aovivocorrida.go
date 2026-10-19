package economy

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const grabWindow = 20 * time.Minute

type grabScenario struct {
	id      string
	title   string
	rarity  eventRarity
	intros  []string
	lock    string
	riddle  riddle
	endings []ending
	missed  []string
}

type riddle func() (clue, answer string)

func grabByID(id string) (grabScenario, bool) {
	i := slices.IndexFunc(grabScenarios, func(sc grabScenario) bool { return sc.id == id })
	if i < 0 {
		return grabScenario{}, false
	}
	return grabScenarios[i], true
}

func arithmetic() (string, string) {
	a, b, c := 3+rand.IntN(10), 3+rand.IntN(10), 1+rand.IntN(40)
	return fmt.Sprintf("%d × %d + %d", a, b, c), strconv.Itoa(a*b + c)
}

func scrambled(words ...string) riddle {
	return func() (string, string) {
		word := core.PickOne(words)
		return strings.ToUpper(shuffledWord(word)), word
	}
}

func shuffledWord(word string) string {
	letters := []rune(word)
	for range 10 {
		rand.Shuffle(len(letters), func(i, j int) { letters[i], letters[j] = letters[j], letters[i] })
		if string(letters) != word {
			break
		}
	}
	return string(letters)
}

func counting(target string, decoys ...string) riddle {
	return func() (string, string) {
		n := 3 + rand.IntN(5)
		row := slices.Repeat([]string{target}, n)
		for range 6 + rand.IntN(5) {
			row = append(row, core.PickOne(decoys))
		}
		rand.Shuffle(len(row), func(i, j int) { row[i], row[j] = row[j], row[i] })
		return fmt.Sprintf("quantos %s tem aqui? %s", target, strings.Join(row, "")), strconv.Itoa(n)
	}
}

func sameAnswer(guess, answer string) bool {
	fold := func(s string) string { return strings.Join(strings.Fields(core.FoldLetters(s)), "") }
	return fold(guess) == fold(answer)
}

func setupGrab(ev *liveEvent, _ int) {
	sc, ok := grabByID(ev.scenario)
	if !ok || sc.riddle == nil {
		return
	}
	ev.clue, ev.answer = sc.riddle()
}

func (sc grabScenario) call(clue string) string {
	if clue == "" {
		return fmt.Sprintf("👉 Quem mandar *!pegar* primeiro leva! Vale por %s", core.FormatWait(grabWindow))
	}
	return fmt.Sprintf("%s\n👉 Quem mandar *!pegar <resposta>* certo primeiro leva! Vale por %s", sc.lockText(clue), core.FormatWait(grabWindow))
}

func (sc grabScenario) lockText(clue string) string {
	return fmt.Sprintf("🔐 %s *%s*", sc.lock, clue)
}

func (b *bot) showGrab(ctx context.Context, ev *liveEvent, _ []types.JID) error {
	sc, ok := grabByID(ev.scenario)
	if !ok {
		return fmt.Errorf("corrida %s não existe", ev.scenario)
	}
	text := liveHeader("🏁", sc.title, sc.rarity) + "\n\n" + core.PickOne(sc.intros) + "\n\n" + sc.call(ev.clue)
	return b.SendText(ctx, ev.chat, text)
}

func (b *bot) missGrab(ctx context.Context, ev *liveEvent) error {
	sc, ok := grabByID(ev.scenario)
	if !ok {
		return nil
	}
	text := fmt.Sprintf("⌛ *%s*: ninguém pegou a tempo\n\n%s", sc.title, core.PickOne(sc.missed))
	if ev.answer != "" {
		text += fmt.Sprintf("\n\n_A resposta era *%s*_", ev.answer)
	}
	return b.SendText(ctx, ev.chat, text)
}

func (b *bot) pegar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	ev, ok := b.liveIn(chat, grabFormat)
	if !ok {
		return b.ReplyTo(ctx, evt, "👀 não tem nada pra pegar agora. Quando rolar uma corrida ao vivo no grupo, quem mandar !pegar primeiro leva")
	}
	sc, ok := grabByID(ev.scenario)
	if !ok {
		return nil
	}
	guess := core.QuestionFrom(core.MessageText(evt.Message))
	if ev.answer != "" && guess == "" {
		return b.ReplyTo(ctx, evt, sc.lockText(ev.clue)+"\nTem senha! Manda !pegar <resposta>")
	}
	if ev.answer != "" && !sameAnswer(guess, ev.answer) {
		b.React(ctx, evt, "❌")
		return nil
	}
	if !ev.close() {
		return b.ReplyTo(ctx, evt, "🐢 chegou tarde, alguém pegou antes")
	}
	b.releaseLive(ctx, ev)
	return b.winGrab(ctx, evt, ev, sc)
}

func (b *bot) winGrab(ctx context.Context, evt *events.Message, ev *liveEvent, sc grabScenario) error {
	acc := b.Account(ctx, evt.Info.Sender)
	end := choice{endings: sc.endings}.roll(Profession{})
	results, err := b.playLive(ctx, ev.chat, []liveTurn{{acc: acc, weight: 100, effects: end.effects}})
	if err != nil {
		return fmt.Errorf("pagar a corrida %s: %w", sc.id, err)
	}
	who := "@" + acc.JID.User
	lines := []string{fmt.Sprintf("🏆 *%s*: %s pegou primeiro!", sc.title, who), "", strings.ReplaceAll(core.PickOne(end.texts), "{@}", who)}
	if res := results[0]; len(res.lines) > 0 {
		lines = append(lines, "", strings.Join(res.lines, "\n"))
	}
	_, err = b.SendMentions(ctx, ev.chat, strings.Join(lines, "\n"), append([]types.JID{acc.JID}, results[0].freed...))
	return err
}
