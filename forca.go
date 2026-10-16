package main

import (
	"context"
	"encoding/json"
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
	forcaKind    = "forca"
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
	{"Comida", "lasanha"}, {"Comida", "estrogonofe"}, {"Comida", "tapioca"}, {"Comida", "moqueca"},
	{"Comida", "acarajé"}, {"Comida", "cuscuz"}, {"Comida", "farofa"}, {"Comida", "pamonha"},
	{"Comida", "quindim"}, {"Comida", "beijinho"}, {"Comida", "empada"}, {"Comida", "esfiha"},
	{"Comida", "hambúrguer"}, {"Comida", "pipoca"}, {"Comida", "panqueca"}, {"Comida", "rapadura"},
	{"Comida", "paçoca"}, {"Comida", "vatapá"}, {"Comida", "baião de dois"}, {"Comida", "misto quente"},
	{"Bicho", "capivara"}, {"Bicho", "jacaré"}, {"Bicho", "tamanduá"}, {"Bicho", "beija-flor"},
	{"Bicho", "ornitorrinco"}, {"Bicho", "pinguim"}, {"Bicho", "tubarão"}, {"Bicho", "camaleão"},
	{"Bicho", "girafa"}, {"Bicho", "hipopótamo"}, {"Bicho", "rinoceronte"}, {"Bicho", "tartaruga"},
	{"Bicho", "papagaio"}, {"Bicho", "tucano"}, {"Bicho", "morcego"}, {"Bicho", "golfinho"},
	{"Bicho", "escorpião"}, {"Bicho", "minhoca"}, {"Bicho", "lagartixa"}, {"Bicho", "gambá"},
	{"Bicho", "tatu"}, {"Bicho", "onça-pintada"}, {"Bicho", "mico-leão-dourado"}, {"Bicho", "arara"},
	{"Bicho", "polvo"}, {"Bicho", "caranguejo"}, {"Bicho", "coruja"}, {"Bicho", "lhama"},
	{"Objeto", "guarda-chuva"}, {"Objeto", "geladeira"}, {"Objeto", "ventilador"}, {"Objeto", "travesseiro"},
	{"Objeto", "liquidificador"}, {"Objeto", "chinelo"}, {"Objeto", "controle remoto"},
	{"Objeto", "micro-ondas"}, {"Objeto", "espelho"}, {"Objeto", "tesoura"}, {"Objeto", "escova de dentes"},
	{"Objeto", "carregador"}, {"Objeto", "fone de ouvido"}, {"Objeto", "mochila"}, {"Objeto", "panela de pressão"},
	{"Objeto", "abajur"}, {"Objeto", "cobertor"}, {"Objeto", "isqueiro"}, {"Objeto", "calculadora"},
	{"Objeto", "grampeador"}, {"Objeto", "lanterna"}, {"Objeto", "vassoura"}, {"Objeto", "despertador"},
	{"Objeto", "tomada"}, {"Objeto", "relógio"}, {"Objeto", "guardanapo"}, {"Objeto", "cabide"},
	{"Lugar", "praia"}, {"Lugar", "rodoviária"}, {"Lugar", "academia"}, {"Lugar", "padaria"},
	{"Lugar", "cemitério"}, {"Lugar", "aeroporto"}, {"Lugar", "motel"},
	{"Lugar", "farmácia"}, {"Lugar", "biblioteca"}, {"Lugar", "hospital"}, {"Lugar", "delegacia"},
	{"Lugar", "igreja"}, {"Lugar", "supermercado"}, {"Lugar", "borracharia"}, {"Lugar", "lanchonete"},
	{"Lugar", "zoológico"}, {"Lugar", "estádio"}, {"Lugar", "açougue"}, {"Lugar", "feira"},
	{"Lugar", "posto de gasolina"}, {"Lugar", "lava-jato"}, {"Lugar", "praça"}, {"Lugar", "cinema"},
	{"Lugar", "boteco"}, {"Lugar", "lotérica"},
	{"Profissão", "bombeiro"}, {"Profissão", "astronauta"}, {"Profissão", "dentista"},
	{"Profissão", "influenciador"}, {"Profissão", "programador"}, {"Profissão", "veterinário"},
	{"Profissão", "cabeleireiro"}, {"Profissão", "encanador"}, {"Profissão", "eletricista"}, {"Profissão", "padeiro"},
	{"Profissão", "motorista"}, {"Profissão", "advogado"}, {"Profissão", "jornalista"}, {"Profissão", "fotógrafo"},
	{"Profissão", "enfermeiro"}, {"Profissão", "carteiro"}, {"Profissão", "pedreiro"}, {"Profissão", "cozinheiro"},
	{"Profissão", "professor"}, {"Profissão", "psicólogo"}, {"Profissão", "arquiteto"}, {"Profissão", "policial"},
	{"Coisa de grupo", "figurinha"}, {"Coisa de grupo", "áudio"}, {"Coisa de grupo", "visualizou"},
	{"Coisa de grupo", "fofoca"}, {"Coisa de grupo", "print"}, {"Coisa de grupo", "cancelado"},
	{"Coisa de grupo", "administrador"}, {"Coisa de grupo", "enquete"}, {"Coisa de grupo", "bom dia"},
	{"Coisa de grupo", "corrente"}, {"Coisa de grupo", "meme"}, {"Coisa de grupo", "link"},
	{"Coisa de grupo", "removido"}, {"Coisa de grupo", "silenciado"}, {"Coisa de grupo", "encaminhada"},
	{"Coisa de grupo", "apagou"}, {"Coisa de grupo", "digitando"}, {"Coisa de grupo", "mensagem fixada"},
	{"Coisa de grupo", "chamada de vídeo"}, {"Coisa de grupo", "visto por último"},
	{"Sentimento", "saudade"}, {"Sentimento", "ciúme"}, {"Sentimento", "vergonha"}, {"Sentimento", "preguiça"},
	{"Sentimento", "alegria"}, {"Sentimento", "tristeza"}, {"Sentimento", "raiva"}, {"Sentimento", "medo"},
	{"Sentimento", "ansiedade"}, {"Sentimento", "felicidade"}, {"Sentimento", "inveja"}, {"Sentimento", "nostalgia"},
	{"Sentimento", "orgulho"}, {"Sentimento", "carência"}, {"Sentimento", "esperança"}, {"Sentimento", "tédio"},
	{"Fruta", "abacaxi"}, {"Fruta", "melancia"}, {"Fruta", "jabuticaba"}, {"Fruta", "maracujá"},
	{"Fruta", "goiaba"}, {"Fruta", "acerola"}, {"Fruta", "caju"}, {"Fruta", "pitanga"},
	{"Fruta", "cupuaçu"}, {"Fruta", "graviola"}, {"Fruta", "manga"}, {"Fruta", "morango"},
	{"Fruta", "banana"}, {"Fruta", "laranja"}, {"Fruta", "tangerina"}, {"Fruta", "framboesa"},
	{"Esporte", "futebol"}, {"Esporte", "vôlei"}, {"Esporte", "basquete"}, {"Esporte", "natação"},
	{"Esporte", "capoeira"}, {"Esporte", "skate"}, {"Esporte", "surfe"}, {"Esporte", "judô"},
	{"Esporte", "boxe"}, {"Esporte", "ciclismo"}, {"Esporte", "handebol"}, {"Esporte", "xadrez"},
	{"Esporte", "atletismo"}, {"Esporte", "ginástica"},
	{"Bebida", "caipirinha"}, {"Bebida", "guaraná"}, {"Bebida", "cachaça"}, {"Bebida", "chimarrão"},
	{"Bebida", "tereré"}, {"Bebida", "cerveja"}, {"Bebida", "refrigerante"}, {"Bebida", "limonada"},
	{"Bebida", "café com leite"}, {"Bebida", "achocolatado"}, {"Bebida", "água de coco"}, {"Bebida", "vinho"},
	{"Roupa", "camiseta"}, {"Roupa", "bermuda"}, {"Roupa", "moletom"}, {"Roupa", "jaqueta"},
	{"Roupa", "pijama"}, {"Roupa", "sutiã"}, {"Roupa", "cueca"}, {"Roupa", "meia"},
	{"Roupa", "boné"}, {"Roupa", "cachecol"}, {"Roupa", "biquíni"}, {"Roupa", "sunga"},
	{"Roupa", "vestido"}, {"Roupa", "gravata"},
	{"Corpo", "cotovelo"}, {"Corpo", "joelho"}, {"Corpo", "sobrancelha"}, {"Corpo", "umbigo"},
	{"Corpo", "tornozelo"}, {"Corpo", "calcanhar"}, {"Corpo", "bochecha"}, {"Corpo", "queixo"},
	{"Corpo", "costela"}, {"Corpo", "pulmão"}, {"Corpo", "fígado"}, {"Corpo", "unha"},
	{"Corpo", "cílios"}, {"Corpo", "orelha"},
	{"Natureza", "cachoeira"}, {"Natureza", "arco-íris"}, {"Natureza", "vulcão"}, {"Natureza", "tempestade"},
	{"Natureza", "relâmpago"}, {"Natureza", "furacão"}, {"Natureza", "montanha"}, {"Natureza", "floresta"},
	{"Natureza", "deserto"}, {"Natureza", "terremoto"}, {"Natureza", "neblina"}, {"Natureza", "granizo"},
	{"Natureza", "maré"}, {"Natureza", "pôr do sol"},
	{"Instrumento", "violão"}, {"Instrumento", "bateria"}, {"Instrumento", "pandeiro"}, {"Instrumento", "sanfona"},
	{"Instrumento", "cavaquinho"}, {"Instrumento", "berimbau"}, {"Instrumento", "flauta"}, {"Instrumento", "teclado"},
	{"Instrumento", "saxofone"}, {"Instrumento", "triângulo"},
	{"Gíria", "treta"}, {"Gíria", "rolê"}, {"Gíria", "migué"}, {"Gíria", "mancada"},
	{"Gíria", "perrengue"}, {"Gíria", "ranço"}, {"Gíria", "sextou"}, {"Gíria", "biscoiteiro"},
	{"Gíria", "gado"}, {"Gíria", "zoeira"}, {"Gíria", "lacrou"}, {"Gíria", "flopou"},
	{"Gíria", "contatinho"}, {"Gíria", "embuste"}, {"Gíria", "vacilão"}, {"Gíria", "mitou"},
	{"Transporte", "bicicleta"}, {"Transporte", "motocicleta"}, {"Transporte", "helicóptero"}, {"Transporte", "submarino"},
	{"Transporte", "trem"}, {"Transporte", "metrô"}, {"Transporte", "ônibus"}, {"Transporte", "patinete"},
	{"Transporte", "caminhão"}, {"Transporte", "navio"}, {"Transporte", "carroça"}, {"Transporte", "teleférico"},
	{"Festa", "quadrilha"}, {"Festa", "quentão"}, {"Festa", "fogueira"}, {"Festa", "carnaval"},
	{"Festa", "confete"}, {"Festa", "serpentina"}, {"Festa", "réveillon"}, {"Festa", "fogos de artifício"},
	{"Festa", "balão"}, {"Festa", "abadá"}, {"Festa", "bexiga"}, {"Festa", "karaokê"},
}

type forcaGame struct {
	mu       sync.Mutex
	word     forcaWord
	tried    []string
	wrong    []string
	misses   int
	over     bool
	won      bool
	players  []string
	lastPlay time.Time
}

type forcaSnapshot struct {
	Hint     string    `json:"hint"`
	Word     string    `json:"word"`
	Tried    []string  `json:"tried"`
	Wrong    []string  `json:"wrong"`
	Misses   int       `json:"misses"`
	Players  []string  `json:"players"`
	LastPlay time.Time `json:"last_play"`
}

func (b *bot) forca(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	guess := strings.Join(strings.Fields(messageText(evt.Message))[1:], " ")
	game, active := b.activeForca(ctx, chat)
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
	game, active := b.activeForca(ctx, evt.Info.Chat.ToNonAD())
	if !active {
		return nil
	}
	return b.playForca(ctx, evt, game, text)
}

func (b *bot) activeForca(ctx context.Context, chat types.JID) (*forcaGame, bool) {
	key := chat.String()
	value, ok := b.forcaGames.Load(key)
	if !ok {
		return nil, false
	}
	game := value.(*forcaGame)
	over, timedOut := game.expire()
	if timedOut {
		b.endForca(ctx, key, game)
	}
	if over {
		return nil, false
	}
	return game, true
}

func (b *bot) startForca(ctx context.Context, chat types.JID) error {
	game := &forcaGame{word: forcaWords[rand.IntN(len(forcaWords))], lastPlay: time.Now()}
	if value, loaded := b.forcaGames.LoadOrStore(chat.String(), game); loaded {
		game = value.(*forcaGame)
	}
	b.saveForca(ctx, chat.String(), game)
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
		b.endForca(ctx, chat.String(), game)
		text = appendLine(text, b.rewardForca(ctx, chat, game, player))
	}
	b.saveForca(ctx, chat.String(), game)
	_, err := b.sendMentions(ctx, chat, text, []types.JID{player})
	return err
}

func (b *bot) saveForca(ctx context.Context, key string, game *forcaGame) {
	game.mu.Lock()
	defer game.mu.Unlock()
	if game.over {
		return
	}
	snapshot := forcaSnapshot{
		Hint:     game.word.hint,
		Word:     game.word.word,
		Tried:    game.tried,
		Wrong:    game.wrong,
		Misses:   game.misses,
		Players:  game.players,
		LastPlay: game.lastPlay,
	}
	b.keepGame(ctx, forcaKind, key, snapshot, game.lastPlay.Add(forcaTimeout))
}

func (b *bot) endForca(ctx context.Context, key string, game *forcaGame) {
	b.dropGame(ctx, forcaKind, key)
	b.forcaGames.CompareAndDelete(key, game)
}

func (b *bot) restoreForca(ctx context.Context, saved savedGame) error {
	var snapshot forcaSnapshot
	if err := json.Unmarshal(saved.state, &snapshot); err != nil {
		return fmt.Errorf("ler forca salva: %w", err)
	}
	if time.Now().After(saved.expires) {
		b.dropGame(ctx, forcaKind, saved.key)
		return nil
	}
	game := &forcaGame{
		word:     forcaWord{hint: snapshot.Hint, word: snapshot.Word},
		tried:    snapshot.Tried,
		wrong:    snapshot.Wrong,
		misses:   snapshot.Misses,
		players:  snapshot.Players,
		lastPlay: snapshot.LastPlay,
	}
	b.forcaGames.LoadOrStore(saved.key, game)
	return nil
}

func (b *bot) rewardForca(ctx context.Context, chat types.JID, game *forcaGame, winner types.JID) string {
	amount, players := game.prize()
	if amount == 0 {
		return ""
	}
	return b.reward(ctx, chat, []types.JID{winner}, players, amount)
}

func foldLetters(s string) string {
	return accentReplacer.Replace(strings.ToLower(s))
}

func (g *forcaGame) expire() (over, timedOut bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.over {
		return true, false
	}
	if time.Since(g.lastPlay) <= forcaTimeout {
		return false, false
	}
	g.over = true
	return true, true
}

func (g *forcaGame) play(guess string, player types.JID) (string, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.over {
		return "", false
	}
	g.lastPlay = time.Now()
	if !slices.Contains(g.players, player.User) {
		g.players = append(g.players, player.User)
	}
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

func (g *forcaGame) prize() (int64, int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.won {
		return 0, 0
	}
	return forcaRewardPerLife * int64(forcaLives-g.misses), len(g.players)
}

func (g *forcaGame) win(player types.JID) string {
	g.over = true
	g.won = true
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
