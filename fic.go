package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

type loveStory struct {
	title    string
	genre    string
	chapters []string
}

type ficDeck struct {
	mu    sync.Mutex
	order []int
}

func (d *ficDeck) draw(size int) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.order) == 0 {
		d.order = rand.Perm(size)
	}
	last := len(d.order) - 1
	next := d.order[last]
	d.order = d.order[:last]
	return next
}

func pickOne[T any](items []T) T {
	return items[rand.IntN(len(items))]
}

func pickSome[T any](items []T, n int) []T {
	picked := make([]T, 0, n)
	for _, i := range rand.Perm(len(items))[:min(n, len(items))] {
		picked = append(picked, items[i])
	}
	return picked
}

func (b *bot) fic(ctx context.Context, evt *events.Message) error {
	pair, err := b.ficPair(ctx, evt)
	if err != nil {
		return fmt.Errorf("escolher casal da fic: %w", err)
	}
	if len(pair) < 2 {
		return b.sendText(ctx, evt.Info.Chat, "Marca uma ou duas pessoas: !fic @pessoa ou !fic @pessoa1 @pessoa2")
	}
	if rand.IntN(2) == 0 {
		pair[0], pair[1] = pair[1], pair[0]
	}

	story := b.pickLoveStory(evt.Info.Chat)
	_, err = b.sendMentions(ctx, evt.Info.Chat, story.render(pair[0], pair[1]), pair)
	return err
}

func (b *bot) ficPair(ctx context.Context, evt *events.Message) ([]types.JID, error) {
	mentions := mentionedJIDs(evt.Message)
	if len(mentions) == 0 {
		return b.randomPartner(ctx, evt)
	}
	pair, ok := coupleFrom(evt.Info.Sender, mentions)
	if !ok {
		return nil, nil
	}
	return pair[:], nil
}

func (b *bot) pickLoveStory(chat types.JID) loveStory {
	if rand.IntN(2) == 0 {
		return generatedLoveStory()
	}
	stories := loveStories()
	deck, _ := b.ficDecks.LoadOrStore(chat, &ficDeck{})
	return stories[deck.(*ficDeck).draw(len(stories))]
}

func generatedLoveStory() loveStory {
	pieces := loveStoryPieces()
	chapters := []string{pickOne(pieces.openings)}
	chapters = append(chapters, pickSome(pieces.middles, rand.IntN(2)+1)...)
	chapters = append(chapters, pickOne(pieces.crises), pickOne(pieces.endings))
	return loveStory{
		title:    pickOne(pieces.titles),
		genre:    pickOne(pieces.genres),
		chapters: chapters,
	}
}

func (s loveStory) render(a, b types.JID) string {
	extras := loveStoryExtras()
	cast := strings.NewReplacer(append([]string{"{a}", "@" + a.User, "{b}", "@" + b.User}, ficProps()...)...)

	lines := []string{
		fmt.Sprintf("📖 *%s*", s.title),
		fmt.Sprintf("_%s_", s.genre),
		pickOne(extras.credits),
		"🏷️ " + strings.Join(pickSome(extras.tags, rand.IntN(3)+2), " "),
	}
	for i, chapter := range s.chapters {
		lines = append(lines, "", fmt.Sprintf("*Capítulo %d*", i+1), chapter)
	}
	lines = append(lines, "", pickOne(extras.endings))
	if rand.IntN(3) == 0 {
		lines = append(lines, "", "*Cena pós-créditos* 🎬", pickOne(extras.postCredits))
	}
	lines = append(lines, "", ficRating())
	lines = append(lines, pickSome(extras.reactions, rand.IntN(2)+1)...)
	return cast.Replace(strings.Join(lines, "\n"))
}

func ficRating() string {
	if rand.IntN(6) > 0 {
		return fmt.Sprintf("⭐ Nota dos leitores: %d/10", rand.IntN(5)+6)
	}
	return pickOne([]string{
		"⭐ Nota dos leitores: 11/10, quebrou a escala",
		"⭐ Nota dos leitores: 2/10, mas todo mundo leu três vezes",
		"⭐ Nota dos leitores: indefinida, o site caiu de tanto acesso",
		"⭐ Nota dos leitores: 10/10 e uma denúncia por excesso de clichê",
		"⭐ Nota dos leitores: 4/10, o capítulo 2 dividiu o fandom",
	})
}

func ficProps() []string {
	props := []struct {
		keys    []string
		options [][]string
	}{
		{
			keys: []string{"{no lugar}"},
			options: [][]string{
				{"na fila do pão"}, {"na fila do INSS"}, {"na academia"}, {"num show de pagode"},
				{"na rodoviária"}, {"numa lan house"}, {"na festa junina da escola"}, {"na biblioteca"},
				{"no churrasco da firma"}, {"na aula de zumba"}, {"na feira de domingo"}, {"no lava-jato"},
				{"no posto de gasolina"}, {"no cursinho"}, {"no karaokê"}, {"no casamento de um primo"},
				{"no ônibus das 6h40"}, {"na praia lotada"}, {"na sala de espera do dentista"},
				{"na reunião de condomínio"}, {"no velório de um tio distante"}, {"na fila do caixa eletrônico"},
			},
		},
		{
			keys: []string{"{uma comida}", "{a comida}"},
			options: [][]string{
				{"uma coxinha", "a coxinha"}, {"um pastel de feira", "o pastel de feira"}, {"um miojo", "o miojo"},
				{"um pão de queijo", "o pão de queijo"}, {"um açaí", "o açaí"},
				{"uma pizza de calabresa", "a pizza de calabresa"}, {"um brigadeiro", "o brigadeiro"},
				{"uma pamonha", "a pamonha"}, {"uma esfirra", "a esfirra"},
				{"um cachorro-quente prensado", "o cachorro-quente prensado"}, {"uma tapioca", "a tapioca"},
				{"um x-tudo", "o x-tudo"}, {"um bolo de pote", "o bolo de pote"}, {"um churros", "o churros"},
				{"um yakisoba", "o yakisoba"}, {"uma marmita de feijoada", "a marmita de feijoada"},
			},
		},
		{
			keys: []string{"{um objeto}", "{o objeto}"},
			options: [][]string{
				{"um guarda-chuva", "o guarda-chuva"}, {"um fone com um lado só funcionando", "o fone"},
				{"um carregador de celular", "o carregador"}, {"um moletom velho", "o moletom"},
				{"um boné", "o boné"}, {"um chinelo", "o chinelo"}, {"um pen drive", "o pen drive"},
				{"uma marmita", "a marmita"}, {"um caderno cheio de rabiscos", "o caderno"},
				{"uma pulseirinha de miçanga", "a pulseirinha"}, {"um isqueiro", "o isqueiro"},
				{"um ursinho de pelúcia", "o ursinho"},
			},
		},
		{
			keys: []string{"{uma musica}"},
			options: [][]string{
				{"um pagode dos anos 90"}, {"um sertanejo de sofrência"}, {"um funk de 2012"},
				{"uma música do Roberto Carlos"}, {"um forró de vaquejada"}, {"um K-pop que só os dois conheciam"},
				{"um brega romântico"}, {"um arrocha"}, {"uma balada de rock dos anos 80"},
				{"aquela música de abertura de novela"}, {"um piseiro"},
			},
		},
		{
			keys: []string{"{apelido}"},
			options: [][]string{
				{"mozão"}, {"bb"}, {"chuchu"}, {"dengo"}, {"vida"}, {"pitico"}, {"xuxuzinho"},
				{"amorzinho"}, {"bebê"}, {"momo"}, {"meu bem"}, {"docinho"},
			},
		},
	}

	var pairs []string
	for _, prop := range props {
		option := pickOne(prop.options)
		for i, key := range prop.keys {
			pairs = append(pairs, key, option[i])
		}
	}
	return pairs
}

type storyExtras struct {
	credits     []string
	tags        []string
	endings     []string
	postCredits []string
	reactions   []string
}

func loveStoryExtras() storyExtras {
	return storyExtras{
		credits: []string{
			"✍️ Estrelando {a} e {b}",
			"💞 Um romance entre {a} e {b}",
			"🎬 Com {a} e {b} nos papéis principais",
			"✍️ Baseada em fatos reais sobre {a} e {b}",
			"📝 Escrita às 4 da manhã sobre {a} e {b}",
			"🕯️ Uma história de amor entre {a} e {b}, que ninguém pediu",
			"📚 Com {a} e {b}, do autor de \"Amor no Grupo\" e \"Amor no Grupo 2\"",
		},
		tags: []string{
			"#clichê", "#slowburn", "#fofinho", "#angst", "#finalfeliz", "#dramalhão",
			"#vergonhaalheia", "#crack", "#ciúmes", "#casamento", "#amizadecolorida",
			"#inimigosamantes", "#sóumacama", "#segundachance", "#amornogrupo", "#plottwist",
			"#chorei", "#morri", "#otp", "#shippo",
		},
		endings: []string{
			"*FIM* 💞",
			"*FIM* 💍",
			"*FIM?* 👀",
			"*Continua na próxima temporada...* 📺",
			"*FIM* (por enquanto) 😏",
			"*E viveram felizes pra sempre* ✨",
			"*FIM* 🥹",
		},
		postCredits: []string{
			"Anos depois, {a} achou {o objeto} guardado numa caixa. {b} tinha guardado tudo desde o primeiro dia.",
			"Plot twist: {b} sabia que ia dar certo desde o começo. Tinha consultado o horóscopo.",
			"Até hoje ninguém sabe quem deu o primeiro passo. {a} jura que foi {b}. {b} jura que foi {a}.",
			"Um gato aparece na janela. É o terceiro gato do casal. Ninguém sabe de onde ele veio.",
			"O grupo ainda manda \"e aí, casal?\" toda vez que {a} ou {b} aparece online.",
			"Dizem que, numa noite de lua cheia, quem passa {no lugar} ainda ouve os dois discutindo quem ama mais.",
			"Em 2047, a história dos dois virou novela das nove. {a} reclamou do ator escolhido pra fazer {b}.",
			"Segunda temporada confirmada: {b} esqueceu o aniversário de namoro. De novo.",
			"{a} ainda chama {b} de {apelido} em público. O grupo pediu pra parar. Não parou.",
			"Revelação final: quem narrou essa história toda era {o objeto}.",
			"{b} encontra um caderno antigo de {a}. Na última página: \"um dia eu caso com {b}\". Data: 2009.",
			"Um ano depois, os dois voltam {no lugar} e dividem {uma comida}, igualzinho ao primeiro dia.",
		},
		reactions: []string{
			"😭 Os leitores choraram e pediram segunda temporada",
			"🔥 Virou trending no Wattpad em 2 horas",
			"🫣 A mãe de alguém leu e agora quer conversar",
			"📚 Já tem editora querendo publicar",
			"💀 Os leitores ainda não superaram",
			"🍿 A Netflix já comprou os direitos",
			"✍️ A autora sumiu e nunca postou o epílogo",
			"💌 O grupo inteiro agora shippa esse casal",
			"🧾 Os dois negam tudo, mas não pediram pra apagar",
			"📸 Já tem fanart circulando no grupo",
			"🗳️ 87% dos leitores votaram pelo casamento",
			"🚨 A fic foi denunciada por ser realista demais",
			"🎤 Virou peça de teatro na escola do bairro",
			"🧑‍⚖️ O advogado de um dos dois entrou em contato",
			"🫠 Um leitor desmaiou no capítulo 3",
			"📱 Printaram e mandaram no grupo da família",
			"👵 A vó leu e já tá perguntando dos bisnetos",
			"🤐 Alguém no grupo sabe que é verdade e tá quieto",
		},
	}
}
