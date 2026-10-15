package main

import (
	"context"
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
	forcaLives   = 6
	forcaTimeout = 15 * time.Minute
)

type forcaWord struct {
	hint string
	word string
}

var forcaWords = []forcaWord{
	{"Comida", "brigadeiro"}, {"Comida", "coxinha"}, {"Comida", "feijoada"}, {"Comida", "pão de queijo"},
	{"Comida", "açaí"}, {"Comida", "pastel"}, {"Comida", "macarrão"}, {"Comida", "churrasco"},
	{"Bicho", "capivara"}, {"Bicho", "jacaré"}, {"Bicho", "tamanduá"}, {"Bicho", "beija-flor"},
	{"Bicho", "ornitorrinco"}, {"Bicho", "pinguim"}, {"Bicho", "tubarão"}, {"Bicho", "camaleão"},
	{"Objeto", "guarda-chuva"}, {"Objeto", "geladeira"}, {"Objeto", "ventilador"}, {"Objeto", "travesseiro"},
	{"Objeto", "liquidificador"}, {"Objeto", "chinelo"}, {"Objeto", "controle remoto"},
	{"Lugar", "praia"}, {"Lugar", "rodoviária"}, {"Lugar", "academia"}, {"Lugar", "padaria"},
	{"Lugar", "cemitério"}, {"Lugar", "aeroporto"}, {"Lugar", "motel"},
	{"Profissão", "bombeiro"}, {"Profissão", "astronauta"}, {"Profissão", "dentista"},
	{"Profissão", "influenciador"}, {"Profissão", "programador"}, {"Profissão", "veterinário"},
	{"Coisa de grupo", "figurinha"}, {"Coisa de grupo", "áudio"}, {"Coisa de grupo", "visualizou"},
	{"Coisa de grupo", "fofoca"}, {"Coisa de grupo", "print"}, {"Coisa de grupo", "cancelado"},
	{"Sentimento", "saudade"}, {"Sentimento", "ciúme"}, {"Sentimento", "vergonha"}, {"Sentimento", "preguiça"},
}

type forcaGame struct {
	mu       sync.Mutex
	word     forcaWord
	tried    []string
	wrong    []string
	misses   int
	over     bool
	lastPlay time.Time
}

func (b *bot) forca(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	guess := strings.Join(strings.Fields(messageText(evt.Message))[1:], " ")
	game, active := b.activeForca(chat)
	if guess == "" && active {
		return b.sendText(ctx, chat, game.status("🪢 *Forca rolando*"))
	}
	if guess == "" {
		return b.startForca(ctx, chat)
	}
	if !active {
		return b.sendText(ctx, chat, "Não tem forca rolando. Manda !forca pra começar")
	}
	return b.playForca(ctx, evt, game, guess)
}

func (b *bot) answerForca(ctx context.Context, evt *events.Message) error {
	text := strings.TrimSpace(messageText(evt.Message))
	first, _ := utf8.DecodeRuneInString(text)
	if utf8.RuneCountInString(text) != 1 || !unicode.IsLetter(first) {
		return nil
	}
	game, active := b.activeForca(evt.Info.Chat.ToNonAD())
	if !active {
		return nil
	}
	return b.playForca(ctx, evt, game, text)
}

func (b *bot) activeForca(chat types.JID) (*forcaGame, bool) {
	value, ok := b.forcaGames.Load(chat.String())
	if !ok {
		return nil, false
	}
	game := value.(*forcaGame)
	if !game.expired() {
		return game, true
	}
	b.forcaGames.CompareAndDelete(chat.String(), game)
	return nil, false
}

func (b *bot) startForca(ctx context.Context, chat types.JID) error {
	game := &forcaGame{word: forcaWords[rand.IntN(len(forcaWords))], lastPlay: time.Now()}
	if value, loaded := b.forcaGames.LoadOrStore(chat.String(), game); loaded {
		game = value.(*forcaGame)
	}
	header := "🪢 *Forca!*\n\n_Manda uma letra pra chutar, ou !forca <palavra> se já sabe a resposta_"
	return b.sendText(ctx, chat, game.status(header))
}

func (b *bot) playForca(ctx context.Context, evt *events.Message, game *forcaGame, guess string) error {
	chat := evt.Info.Chat.ToNonAD()
	player := evt.Info.Sender.ToNonAD()
	text, over := game.play(guess, player)
	if text == "" {
		return nil
	}
	if over {
		b.forcaGames.CompareAndDelete(chat.String(), game)
	}
	_, err := b.sendMentions(ctx, chat, text, []types.JID{player})
	return err
}

func foldLetters(s string) string {
	return accentReplacer.Replace(strings.ToLower(s))
}

func (g *forcaGame) expired() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.over || time.Since(g.lastPlay) > forcaTimeout
}

func (g *forcaGame) play(guess string, player types.JID) (string, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.over {
		return "", false
	}
	g.lastPlay = time.Now()
	folded := foldLetters(strings.TrimSpace(guess))
	if utf8.RuneCountInString(folded) > 1 {
		return g.guessWord(folded, player)
	}
	return g.guessLetter(folded, player)
}

func (g *forcaGame) guessLetter(letter string, player types.JID) (string, bool) {
	shown := strings.ToUpper(letter)
	if slices.Contains(g.tried, letter) {
		return fmt.Sprintf("🙄 A letra *%s* já foi, @%s", shown, player.User), false
	}
	g.tried = append(g.tried, letter)
	if !strings.Contains(foldLetters(g.word.word), letter) {
		g.wrong = append(g.wrong, shown)
		return g.miss(fmt.Sprintf("❌ @%s chutou *%s* e não tem", player.User, shown))
	}
	if g.solved() {
		return g.win(player), true
	}
	return g.board(fmt.Sprintf("✅ @%s achou a letra *%s*!", player.User, shown)), false
}

func (g *forcaGame) guessWord(word string, player types.JID) (string, bool) {
	if word == foldLetters(g.word.word) {
		return g.win(player), true
	}
	return g.miss(fmt.Sprintf("❌ @%s chutou *%s* e errou", player.User, strings.ToUpper(word)))
}

func (g *forcaGame) miss(header string) (string, bool) {
	g.misses++
	if g.misses < forcaLives {
		return g.board(header), false
	}
	g.over = true
	return fmt.Sprintf("%s\n\n```%s```\n💀 *Enforcado!* A palavra era *%s*", header, gallows(g.misses), strings.ToUpper(g.word.word)), true
}

func (g *forcaGame) win(player types.JID) string {
	g.over = true
	return fmt.Sprintf("🎉 @%s acertou! A palavra era *%s*\n\n❤️ Sobraram %d de %d vidas",
		player.User, strings.ToUpper(g.word.word), forcaLives-g.misses, forcaLives)
}

func (g *forcaGame) solved() bool {
	for _, r := range g.word.word {
		if unicode.IsLetter(r) && !slices.Contains(g.tried, foldLetters(string(r))) {
			return false
		}
	}
	return true
}

func (g *forcaGame) masked() string {
	var cells []string
	for _, r := range strings.ToUpper(g.word.word) {
		if !unicode.IsLetter(r) {
			cells = append(cells, string(r))
			continue
		}
		cell := "_"
		if slices.Contains(g.tried, foldLetters(string(r))) {
			cell = string(r)
		}
		cells = append(cells, cell)
	}
	return strings.Join(cells, " ")
}

func (g *forcaGame) status(header string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.board(header)
}

func (g *forcaGame) board(header string) string {
	lines := []string{
		header,
		"",
		"```" + gallows(g.misses) + "```",
		"💡 Dica: " + g.word.hint,
		"🔤 " + g.masked(),
	}
	if len(g.wrong) > 0 {
		lines = append(lines, "🚫 Erradas: "+strings.Join(g.wrong, ", "))
	}
	lines = append(lines, fmt.Sprintf("❤️ Vidas: %d/%d", forcaLives-g.misses, forcaLives))
	return strings.Join(lines, "\n")
}

func gallows(misses int) string {
	bodies := []string{"", "", " |", "/|", "/|\\", "/|\\", "/|\\"}
	legs := []string{"", "", "", "", "", "/", "/ \\"}
	head := " "
	if misses > 0 {
		head = "O"
	}
	stage := min(misses, forcaLives)
	return fmt.Sprintf("  +---+\n  |   |\n  %s   |\n %-3s  |\n %-3s  |\n      |\n=========", head, bodies[stage], legs[stage])
}
