package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

type loveStory struct {
	title    string
	genre    string
	chapters []string
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

	stories := loveStories()
	story := stories[rand.IntN(len(stories))]
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

func (s loveStory) render(a, b types.JID) string {
	cast := strings.NewReplacer("{a}", "@"+a.User, "{b}", "@"+b.User)
	reactions := []string{
		"😭 Os leitores choraram e pediram segunda temporada",
		"🔥 Virou trending no Wattpad em 2 horas",
		"🫣 A mãe de alguém leu e agora quer conversar",
		"📚 Já tem editora querendo publicar",
		"💀 Os leitores ainda não superaram",
		"🍿 A Netflix já comprou os direitos",
		"✍️ A autora sumiu e nunca postou o epílogo",
		"💌 O grupo inteiro agora shippa esse casal",
	}

	lines := []string{
		fmt.Sprintf("📖 *%s*", s.title),
		fmt.Sprintf("_%s_", s.genre),
		cast.Replace("✍️ Estrelando {a} e {b}"),
	}
	for i, chapter := range s.chapters {
		lines = append(lines, "", fmt.Sprintf("*Capítulo %d*", i+1), cast.Replace(chapter))
	}
	lines = append(lines,
		"",
		"*FIM* 💞",
		"",
		fmt.Sprintf("⭐ Nota dos leitores: %d/10", rand.IntN(5)+6),
		reactions[rand.IntN(len(reactions))],
	)
	return strings.Join(lines, "\n")
}

func loveStories() []loveStory {
	return []loveStory{
		{
			title: "Inimigos no Mesmo Grupo",
			genre: "Enemies to lovers 😤➡️😍",
			chapters: []string{
				"{a} e {b} brigavam todo dia no grupo. Bastava um mandar áudio que o outro respondia \"ninguém vai ouvir isso\".",
				"Até que um dia o grupo ficou em silêncio e só os dois estavam online às 3 da manhã. A briga virou conversa, a conversa virou desabafo.",
				"No dia seguinte {b} mandou figurinha de coração sem querer. Ou era o que {b} dizia.",
				"Hoje os dois ainda brigam no grupo, mas agora no privado se chamam de \"meu bem\". Todo mundo sabe. Ninguém fala nada.",
			},
		},
		{
			title: "O Café da Esquina",
			genre: "Romance clichê ☕",
			chapters: []string{
				"Toda manhã {a} pedia o mesmo café na mesma padaria. E toda manhã {b}, o atendente, errava o nome no copo.",
				"\"Hoje eu acertei\", disse {b} um dia, entregando o copo. Embaixo do nome tinha um número de telefone.",
				"{a} demorou três dias pra criar coragem de mandar \"oi\". {b} respondeu em 4 segundos.",
				"Um ano depois, a padaria pendurou uma foto dos dois na parede. O café continua ruim. O amor, não.",
			},
		},
		{
			title: "Amor em Tempos de Wi-Fi Ruim",
			genre: "Romance à distância 📶",
			chapters: []string{
				"{a} morava num estado, {b} em outro. Se conheceram numa partida online em que os dois perderam feio.",
				"As chamadas de vídeo travavam sempre na pior hora. {a} já tinha visto {b} congelado de boca aberta umas quarenta vezes.",
				"Um dia {b} sumiu por dois dias. {a} achou que tinha levado um vácuo. Na verdade, {b} tava num ônibus de 30 horas.",
				"A campainha tocou. Era {b}, com uma mala e um carregador de celular de presente. \"Pra você nunca mais cair na chamada.\"",
			},
		},
		{
			title: "A Princesa e o Plebeu",
			genre: "Romance de época 👑",
			chapters: []string{
				"No reino de Grupolândia, {a} era da realeza e {b} um simples vendedor de pão de queijo na praça.",
				"Todo dia {a} fugia do castelo disfarçado só pra comprar um pão de queijo. O pão era ruim. A desculpa era ótima.",
				"O rei descobriu e proibiu o romance. {b} foi banido para o reino vizinho, o temido Grupo da Família.",
				"{a} abdicou do trono, atravessou o reino a cavalo e disse: \"prefiro pão de queijo com você do que caviar sem você\". O povo aplaudiu.",
			},
		},
		{
			title: "Colegas de Quarto",
			genre: "Slow burn 🐢🔥",
			chapters: []string{
				"{a} e {b} dividiam apartamento só por causa do aluguel. Regra número um: nada de sentimentos.",
				"Regra número dois, criada depois: ninguém usa a caneca do outro. {a} usou. {b} fingiu ficar bravo, mas guardou a caneca lavada com um bilhete.",
				"Numa noite de tempestade a luz acabou. Os dois ficaram no sofá conversando até amanhecer, dividindo o mesmo cobertor \"por economia\".",
				"Seis meses depois, desistiram do segundo quarto. Viraram um escritório. E a regra número um foi oficialmente revogada.",
			},
		},
		{
			title: "O Feitiço Errado",
			genre: "Fantasia ✨",
			chapters: []string{
				"{a} era aprendiz de bruxaria e tentou fazer uma poção pra passar na prova. Errou um ingrediente.",
				"A poção virou um feitiço do amor e caiu, por acidente, no copo de {b}, o bruxo mais metido da escola.",
				"{b} passou a seguir {a} pra todo lado recitando poesia ruim. {a} passou uma semana procurando o antídoto.",
				"Quando finalmente acharam a cura, {b} tomou, ficou em silêncio e disse: \"o feitiço passou faz três dias. Eu só continuei\".",
			},
		},
		{
			title: "Amor no Apocalipse",
			genre: "Ação e sobrevivência 🧟",
			chapters: []string{
				"O mundo acabou numa terça-feira. {a} se escondeu num supermercado e deu de cara com {b}, armado com uma baguete.",
				"Decidiram sobreviver juntos. {b} cuidava da comida, {a} da estratégia. Os zumbis cuidavam de atrapalhar.",
				"Encurralados num telhado, {a} disse: \"se a gente não sair dessa, eu queria te falar uma coisa\". O helicóptero chegou antes.",
				"Na base dos sobreviventes, {b} cobrou: \"e aquela coisa?\". {a} respondeu com um beijo. O apocalipse nunca foi tão romântico.",
			},
		},
		{
			title: "Match às 2 da Manhã",
			genre: "Comédia romântica 📱",
			chapters: []string{
				"{a} deu like em {b} no aplicativo achando que tava rolando pra passar. Deu match.",
				"A primeira mensagem de {b} foi \"você também tem cara de quem come pizza fria no café da manhã\". Era verdade.",
				"O primeiro encontro foi um desastre: chuva, restaurante fechado, guarda-chuva quebrado. Acabaram comendo cachorro-quente na marquise.",
				"Até hoje discutem quem deu like primeiro. O aplicativo foi desinstalado no dia seguinte. Pelos dois.",
			},
		},
		{
			title: "O Último Verão",
			genre: "Drama de chorar 😭",
			chapters: []string{
				"Era o último verão antes de {a} se mudar pra longe. {b} sabia, mas fingia que não.",
				"Passaram as férias juntos: praia, sorvete derretendo, pôr do sol que nenhum dos dois conseguia fotografar direito.",
				"Na véspera da viagem, {b} entregou uma carta e pediu pra {a} só abrir no avião. {a} abriu no ônibus pro aeroporto.",
				"Dentro da carta só tinha uma frase: \"eu vou junto\". {b} tava sentado no banco de trás, com uma mochila e um sorriso.",
			},
		},
		{
			title: "Detetives do Coração",
			genre: "Mistério 🕵️",
			chapters: []string{
				"Alguém estava roubando os biscoitos da firma. {a} e {b}, os dois mais fofoqueiros do escritório, foram investigar.",
				"Fizeram tocaia, montaram um mural com barbante vermelho e trocaram mais olhares do que pistas.",
				"Descobriram que o ladrão era o chefe. Mas a maior descoberta foi outra: nenhum dos dois queria que o caso acabasse.",
				"Abriram uma agência de detetives juntos. Nunca resolveram outro caso. Mas almoçam juntos todo dia.",
			},
		},
		{
			title: "Rivais de Campeonato",
			genre: "Esportes ⚽",
			chapters: []string{
				"{a} era capitão de um time, {b} do time rival. A final do campeonato do bairro estava marcada.",
				"Nos treinos, um espionava o outro. Nas espionagens, acabavam conversando na arquibancada até escurecer.",
				"Na final, {b} perdeu um pênalti. Dizem que foi sem querer. Dizem que {b} olhou pra {a} antes de chutar.",
				"{a} levantou a taça e, no meio da festa, foi até {b} e entregou a medalha. \"Metade é sua\". O bairro todo gritou.",
			},
		},
		{
			title: "Viagem no Tempo",
			genre: "Ficção científica 🚀",
			chapters: []string{
				"{a} inventou uma máquina do tempo na garagem e, no primeiro teste, foi parar em 1985. Quem estava lá era {b}.",
				"{b} ajudou {a} a consertar a máquina, entre fitas cassete e um walkman que não parava de enrolar.",
				"Quando a máquina ficou pronta, {a} hesitou. Voltar pro presente significava deixar {b} pra trás.",
				"{a} voltou. E no presente encontrou {b}, com 40 anos a mais, esperando no mesmo lugar: \"demorou, hein\".",
			},
		},
	}
}
