package main

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

type bot struct {
	client *whatsmeow.Client
	gifs   map[string][]gif

	bmcRounds        sync.Map
	vddAnswered      sync.Map
	quemfalouRounds  sync.Map
	quizRounds       sync.Map
	quizChats        sync.Map
	forcaGames       sync.Map
	termoGames       sync.Map
	impostorGames    sync.Map
	impostorPolls    sync.Map
	blackjackHands   sync.Map
	crashRounds      sync.Map
	raceRounds       sync.Map
	velhaGames       sync.Map
	revolvers        sync.Map
	laughStickerSent sync.Map
	revokedMessages  sync.Map
	ficDecks         sync.Map
	guessRounds      sync.Map
	emojiRecent      sync.Map
	batataGames      sync.Map
	stopGames        sync.Map
	itemOffers       sync.Map
	debtRequests     sync.Map
	romanceProposals sync.Map
	privatePrompts   sync.Map
	mentiraChats     sync.Map
	mentiraPolls     sync.Map
	bandeiraRecent   sync.Map
	auctions         sync.Map
	welcomeBatches   sync.Map
	trialChats       sync.Map
	trialPolls       sync.Map
	heistCrews       sync.Map
	hostageSales     sync.Map
	elections        sync.Map
	impeachments     sync.Map
	messages         *messageStore
	nicknames        *labelStore
	names            *labelStore
	relationships    *relationshipStore
	adoptions        *adoptionStore
	assistant        *assistant
	transcriber      *transcriber
	filters          *filterStore
	modes            *modeStore
	wallets          *walletStore
	quizzes          *quizStore
	mutes            *restrictionStore
	handcuffs        *restrictionStore
	persecuted       *restrictionStore
	reminders        *reminderStore
	birthdays        *birthdayStore
	memories         *memoryStore
	warnings         *warningStore
	welcomes         *welcomeStore
	games            *gameStore
	gamesReady       chan struct{}
	owners           []string
}

type command func(ctx context.Context, evt *events.Message) error

type usage struct {
	syntax      string
	description string
}

type category struct {
	title string
	mode  string
}

type botCommand struct {
	name         string
	aliases      []string
	category     category
	help         []usage
	run          command
	barredInJail bool
	whileDead    bool
}

func (c botCommand) matches(name string) bool {
	name = foldLetters(name)
	return slices.ContainsFunc(append([]string{c.name}, c.aliases...), func(n string) bool { return foldLetters(n) == name })
}

func (b *bot) commands() []botCommand {
	var (
		romance      = category{"💞 Romance", modeRomance}
		familia      = category{"👨‍👩‍👧 Família", modeFamily}
		jogos        = category{"🎲 Jogos", modeGames}
		medidores    = category{"📊 Medidores", modeMeters}
		estatisticas = category{"📈 Estatísticas", modeStats}
		zoeira       = category{"😂 Zoeira", modeJokes}
		inteligencia = category{"🧠 IA", modeAI}
		enxerida     = category{"🧠 IA", modeAIMeddling}
		utilidades   = category{"🛠️ Utilidades", modeUtilities}
		economia     = category{"💰 Economia", modeEconomy}
		moderacao    = category{"🛡️ Moderação", modeModeration}
		ajuda        = category{"ℹ️ Ajuda", ""}
	)
	return []botCommand{
		{
			name:     "casar",
			category: romance,
			help: []usage{
				{"!casar @pessoa", "casa você com a pessoa marcada"},
				{"!casar @pessoa1 @pessoa2", "casa as duas pessoas marcadas"},
				{"!casar @pessoa <dote>", fmt.Sprintf("com o modo consentimento ligado, vira um pedido de casamento 💍 A pessoa tem %s pra responder com !aceitar ou !recusar, e cada um só faz o próprio pedido. Com a economia ligada, quem pede paga a festa (%s, ou casa no cartório se não tiver), pode oferecer um dote, ganha uma !conjunta com o cônjuge e, se pedir divórcio depois, paga pensão", formatWait(proposalWindow), coins(weddingPartyCost))},
			},
			run: b.casar,
		},
		{
			name:     "adotar",
			category: familia,
			help:     []usage{{"!adotar @pessoa", "você adota a pessoa marcada como filho(a) 🍼"}},
			run:      b.adotar,
		},
		{
			name:     "engravidar",
			category: familia,
			help: []usage{
				{"!engravidar @pessoa", "você engravida a pessoa marcada 🤰"},
				{"!engravidar @pessoa1 @pessoa2", "a primeira pessoa marcada engravida a segunda 🤰"},
			},
			run: b.engravidar,
		},
		{
			name:     "familia",
			category: familia,
			help: []usage{
				{"!familia", "mostra a sua árvore: cônjuges, pais, filhos, irmãos, avós e netos 🌳"},
				{"!familia @pessoa", "mostra a família da pessoa marcada"},
			},
			run: b.familia,
		},
		{
			name:     "ship",
			category: romance,
			help: []usage{
				{"!ship @pessoa", "diz de 0 a 100% o quanto você combina com a pessoa marcada e inventa o nome do casal 💘"},
				{"!ship @pessoa1 @pessoa2", "faz o ship das duas pessoas marcadas"},
			},
			run: b.ship,
		},
		{
			name:     "namoro",
			aliases:  []string{"namorado", "namorada"},
			category: romance,
			help: []usage{
				{"!namorado", "mostra o seu namorado. Se você não namora ninguém pelo !namorar, mostra quem dá o maior !ship com você 💑"},
				{"!namorada", "igual ao !namorado, mas no feminino"},
				{"!namoro @pessoa", "mostra com quem a pessoa marcada namora (os três comandos aceitam @pessoa)"},
			},
			run: b.namoro,
		},
		{
			name:     "namorar",
			category: romance,
			help: []usage{
				{"!namorar @pessoa", "começa a namorar a pessoa marcada 💘 (dá pra namorar mais de uma pessoa)"},
				{"!namorar @pessoa1 @pessoa2", "faz as duas pessoas marcadas namorarem"},
				{"!namorar @pessoa <presente>", "com o modo consentimento ligado, vira um pedido de namoro 💘 A pessoa responde com !aceitar ou !recusar, e com a economia ligada dá pra mandar um presente em moedas junto"},
			},
			run: b.namorar,
		},
		{
			name:     "terminar",
			aliases:  []string{"divorciar"},
			category: romance,
			help: []usage{
				{"!terminar @pessoa", "você termina o namoro, o casamento e o caso de amante com a pessoa marcada 💔"},
				{"!terminar @pessoa1 @pessoa2", "termina o namoro, o casamento e o caso de amante das duas pessoas marcadas (com o modo consentimento ligado, só se você for uma delas)"},
				{"!terminar", "você termina com todo mundo que namora, com quem casou e com quem é amante (!divorciar faz o mesmo)"},
			},
			run: b.terminar,
		},
		{
			name:     "almagemea",
			aliases:  []string{"alma"},
			category: romance,
			help: []usage{
				{"!almagemea", "lê as mensagens do grupo e descobre quem escreve mais parecido com você, a sua alma gêmea 💞"},
				{"!almagemea @pessoa", "procura a alma gêmea da pessoa marcada"},
			},
			run: b.almagemea,
		},
		{
			name:     "sexo",
			category: romance,
			help: []usage{
				{"!sexo", "leva você e uma pessoa aleatória do grupo pro quarto 🔥"},
				{"!sexo @pessoa", "leva você e a pessoa marcada pro quarto 🔥"},
				{"!sexo @pessoa1 @pessoa2 @pessoa3...", "leva todo mundo que você marcar pro quarto 🔥"},
				{"!sexo @todos", "leva o grupo inteiro pro quarto 🔥"},
			},
			run: b.sexo,
		},
		{
			name:     "mamar",
			category: romance,
			help: []usage{
				{"!mamar", "você e uma pessoa aleatória do grupo vão pra um cantinho escuro 🍼"},
				{"!mamar @pessoa", "você e a pessoa marcada vão pra um cantinho escuro 🍼"},
				{"!mamar @pessoa1 @pessoa2 @pessoa3...", "leva todo mundo que você marcar pro cantinho escuro 🍼"},
				{"!mamar @todos", "leva o grupo inteiro pro cantinho escuro 🍼"},
			},
			run: b.mamar,
		},
		{
			name:     "estudar",
			category: romance,
			help: []usage{
				{"!estudar", "você estuda uma pessoa aleatória do grupo, de portas fechadas 📚"},
				{"!estudar @pessoa", "você estuda a pessoa marcada, de portas fechadas 📚"},
				{"!estudar @pessoa1 @pessoa2 @pessoa3...", "junta todo mundo que você marcar num grupo de estudos 📚"},
				{"!estudar @todos", "o grupo inteiro vira um grupo de estudos 📚"},
			},
			run: b.estudar,
		},
		{
			name:     "beijo",
			category: romance,
			help: []usage{
				{"!beijo @pessoa", "você dá um beijo na pessoa marcada 💋"},
				{"!beijo @pessoa1 @pessoa2", "a primeira pessoa marcada beija a segunda 💋"},
			},
			run: b.beijo,
		},
		{
			name:     "abracar",
			aliases:  []string{"abraco"},
			category: romance,
			help: []usage{
				{"!abracar @pessoa", "você dá um abraço na pessoa marcada 🤗 (!abraço faz o mesmo)"},
				{"!abracar @pessoa1 @pessoa2", "a primeira pessoa marcada abraça a segunda 🤗"},
			},
			run: b.abracar,
		},
		{
			name:     "cafune",
			category: romance,
			help: []usage{
				{"!cafune @pessoa", "você faz cafuné na pessoa marcada 🤲"},
				{"!cafune @pessoa1 @pessoa2", "a primeira pessoa marcada faz cafuné na segunda 🤲"},
			},
			run: b.cafune,
		},
		{
			name:     "trair",
			category: romance,
			help: []usage{
				{"!trair @pessoa", "você trai a pessoa marcada 🐍"},
				{"!trair @pessoa1 @pessoa2", "a primeira pessoa marcada trai a segunda"},
			},
			run: b.trair,
		},
		{
			name:     "traircom",
			category: romance,
			help: []usage{
				{"!traircom @pessoa", "você trai quem você namora ou casou com a pessoa marcada 🐍 e vocês dois viram amantes"},
				{"!traircom @pessoa1 @pessoa2", "a primeira pessoa marcada trai quem namora ou casou com a segunda"},
			},
			run: b.traircom,
		},
		{
			name:     "amante",
			aliases:  []string{"amantes"},
			category: romance,
			help: []usage{
				{"!amante", "mostra os seus amantes, que aparecem quando alguém usa o !traircom 🤫"},
				{"!amante @pessoa", "mostra os amantes da pessoa marcada"},
			},
			run: b.amante,
		},
		{
			name:     "fic",
			category: romance,
			help: []usage{
				{"!fic", "escreve uma fic de amor entre você e uma pessoa aleatória do grupo 📖"},
				{"!fic @pessoa", "escreve uma fic de amor entre você e a pessoa marcada"},
				{"!fic @pessoa1 @pessoa2", "escreve uma fic de amor entre as duas pessoas marcadas"},
			},
			run: b.fic,
		},
		{
			name:     "bmc",
			category: jogos,
			help: []usage{
				{"!bmc", "pergunta pra você quem você beija, mata ou casa entre 3 pessoas do grupo"},
				{"!bmc @pessoa", "faz a pergunta pra pessoa marcada"},
			},
			run: b.bmc,
		},
		{
			name:     "vdd",
			category: jogos,
			help: []usage{
				{"!vdd", "o bot te pergunta verdade ou desafio 🔥"},
				{"!vdd @pessoa", "faz a pergunta pra pessoa marcada"},
			},
			run: b.vdd,
		},
		{
			name:     "eununca",
			category: jogos,
			help:     []usage{{"!euNunca", "abre uma enquete de eu nunca 🙊"}},
			run:      b.euNunca,
		},
		{
			name:     "prefere",
			category: jogos,
			help:     []usage{{"!prefere", "abre uma enquete de você prefere 🤔"}},
			run:      b.prefere,
		},
		{
			name:     "quem",
			category: jogos,
			help: []usage{
				{"!quem", "abre uma enquete de quem é mais provável, com gente do grupo como opção 👀"},
				{"!quem <pergunta>", "a sua pergunta vira a enquete, tipo !quem vai casar primeiro ou !quem é o mais gado daqui"},
			},
			run: b.quem,
		},
		{
			name:     "quemfalou",
			category: jogos,
			help: []usage{
				{"!quemfalou", "o bot desenterra uma mensagem antiga e abre uma enquete pra adivinhar quem mandou 🕵️"},
				{"!quemfalou <frase>", "em vez de sortear, usa uma mensagem que tenha essa frase"},
			},
			run: b.quemfalou,
		},
		{
			name:     "completa",
			aliases:  []string{"completar"},
			category: jogos,
			help: []usage{
				{"!completa", fmt.Sprintf("o bot desenterra uma mensagem antiga e esconde uma palavra: quem mandar ela primeiro acerta, e com o modo economia ligado ganha %s 🧩", coins(completaReward))},
			},
			run: b.completa,
		},
		{
			name:     "forca",
			category: jogos,
			help: []usage{
				{"!forca", "começa um jogo da forca no grupo, e depois é só mandar uma letra pra chutar 🪢"},
				{"!forca <palavra>", "chuta a palavra inteira"},
			},
			run: b.forca,
		},
		{
			name:     "termo",
			category: jogos,
			help: []usage{
				{"!termo", "começa um termo (o wordle em português): 6 chances pro grupo acertar a palavra de 5 letras 🟩🟨"},
				{"!termo <palavra>", "chuta uma palavra (uma palavra de 5 letras mandada sozinha também vale)"},
			},
			run: b.termo,
		},
		{
			name:     "quiz",
			category: jogos,
			help: []usage{
				{"!quiz", fmt.Sprintf("abre uma enquete com uma pergunta de conhecimentos gerais, e com o modo economia ligado quem acerta ganha %s 🎲", coins(quizReward))},
			},
			run: b.quiz,
		},
		{
			name:     "velha",
			aliases:  []string{"jogodavelha"},
			category: jogos,
			help: []usage{
				{"!velha @pessoa", "desafia a pessoa marcada pro jogo da velha, e depois é só mandar o número da casa (1 a 9) pra jogar ❌⭕"},
				{"!velha", "joga contra o bot"},
				{"!velha desistir", "desiste do jogo que tá rolando"},
			},
			run: b.velha,
		},
		{
			name:     "roleta",
			category: jogos,
			help: []usage{
				{"!roleta", "roleta russa: 1 bala, 6 câmaras, e a chance de morrer sobe a cada rodada 🔫"},
				{"!roleta @pessoa", "aponta o revólver pra pessoa marcada"},
				{"!roleta <valor>", "com o modo economia ligado, aposta moedas que quem levar o tiro sobrevive: quanto menos câmaras sobrando, mais paga 💰"},
			},
			run: b.roleta,
		},
		{
			name:     "impostor",
			category: jogos,
			help: []usage{
				{"!impostor", fmt.Sprintf("abre a sala do jogo do impostor, ou entra na que tá aberta (também dá pra entrar respondendo a mensagem da sala só com !) 🕵️ Todo mundo recebe a palavra secreta no privado, menos o impostor, que só sabe a categoria. Aí rolam rodadas de dicas: na sua vez eu te marco e você manda UMA palavra no grupo. No fim de cada rodada uma enquete decide se já dá pra votar em quem é o impostor ou se vai mais uma (no máximo %d rodadas). Com o modo economia ligado, se pegarem o impostor cada um ganha %s, e se ele escapar leva %s", impostorMaxRounds, coins(impostorGroupReward), coins(impostorSoloReward))},
				{"!impostor começar", fmt.Sprintf("começa sem esperar o tempo da sala (mínimo de %d jogadores)", impostorMinPlayers)},
				{"!impostor sair", "sai da sala antes do jogo começar (quem abriu a sala, ou o primeiro da lista se ele sair, pode usar !impostor cancelar)"},
			},
			run: b.impostor,
		},
		{
			name:     "batata",
			category: jogos,
			help: []usage{
				{"!batata", "começa a batata quente com você segurando: pra passar, manda qualquer mensagem marcando alguém, e ela explode numa hora que ninguém sabe 🥔🔥"},
				{"!batata @pessoa", "começa com a batata na mão da pessoa marcada, ou passa a batata pra ela se tiver na sua mão"},
			},
			run: b.batata,
		},
		{
			name:     "stop",
			aliases:  []string{"adedanha"},
			category: jogos,
			help: []usage{
				{"!stop", "começa um stop (adedanha): o bot sorteia uma letra e 5 categorias, e quem completar tudo primeiro pede STOP ✋"},
				{"!stop <respostas>", fmt.Sprintf("manda as 5 respostas na ordem, separadas por vírgula ou uma por linha. Resposta única vale 10 pontos, repetida vale 5, e com o modo economia ligado quem fizer mais pontos ganha %s", coins(stopReward))},
			},
			run: b.stop,
		},
		{
			name:     "jokenpo",
			aliases:  []string{"ppt", "pedrapapeltesoura"},
			category: jogos,
			help: []usage{
				{"!jokenpo pedra", "joga pedra, papel ou tesoura contra o bot na hora ✊✋✌️"},
				{"!jokenpo @pessoa", fmt.Sprintf("desafia a pessoa marcada: o bot chama vocês dois no privado, cada um manda a jogada lá em segredo (vale ✊ ✋ ✌️ também) e o resultado sai no grupo. Quem não jogar em %s perde por W.O.", formatWait(jokenpoWindow))},
				{"!jokenpo @pessoa <valor>", "com o modo economia ligado, vale moedas: a sua aposta sai na hora, a pessoa aceita mandando a jogada no privado, quem ganhar leva o dobro e empate devolve 💰"},
			},
			run: b.jokenpo,
		},
		{
			name:     "2verdades1mentira",
			aliases:  []string{"2v1m", "mentira"},
			category: jogos,
			help: []usage{
				{"!2verdades1mentira", fmt.Sprintf("o bot te chama no privado pra contar 3 coisas sobre você, 2 verdades e 1 mentira, e abre uma enquete no grupo pra descobrirem qual é a mentira 🤥 Com o modo economia ligado, quem acerta ganha %s", coins(mentiraReward))},
			},
			run: b.mentira,
		},
		{
			name:     "bandeira",
			aliases:  []string{"bandeiras", "pais"},
			category: jogos,
			help: []usage{
				{"!bandeira", fmt.Sprintf("o bot manda a bandeira de um país e quem mandar o nome dele primeiro acerta, e com o modo economia ligado ganha %s 🏳️", coins(bandeiraReward))},
			},
			run: b.bandeira,
		},
		{
			name:     "medidor",
			category: medidores,
			help: []usage{
				{"!medidor <coisa> @pessoa", "lê as mensagens da pessoa e diz de 0 a 100% o quanto ela é <coisa>"},
				{"!medidor <coisa>", "mede você"},
			},
			run: b.medidor,
		},
		{
			name:     "gado",
			category: medidores,
			help:     []usage{{"!gado @pessoa", "lê as mensagens da pessoa e diz o quanto ela é gado 🐂"}},
			run:      b.meter(gadoMeter()),
		},
		{
			name:     "corno",
			category: medidores,
			help:     []usage{{"!corno @pessoa", "lê as mensagens da pessoa e diz o quanto ela é corna 🦌"}},
			run:      b.meter(cornoMeter()),
		},
		{
			name:     "gay",
			category: medidores,
			help:     []usage{{"!gay @pessoa", "lê as mensagens da pessoa e diz o quanto ela é gay 🏳️‍🌈"}},
			run:      b.meter(gayMeter()),
		},
		{
			name:     "rankinggay",
			aliases:  []string{"rankgay", "maisgay"},
			category: medidores,
			help:     []usage{{"!rankinggay", "mede todo mundo do grupo com o !gay e mostra quem é o mais gay 🏳️‍🌈"}},
			run:      b.meterRanking(gayMeter()),
		},
		{
			name:     "chato",
			category: medidores,
			help:     []usage{{"!chato @pessoa", "diz o quanto a pessoa é chata: fala demais, grita em caps, spamma comando, reclama de tudo 🙄"}},
			run:      b.meter(chatoMeter()),
		},
		{
			name:     "euanime",
			category: medidores,
			help: []usage{
				{"!euanime @pessoa", "diz qual personagem de anime a pessoa é, com foto 🎭 É sempre o mesmo pra mesma pessoa no grupo (quer um que combine de verdade? usa o !quemsou)"},
				{"!euanime", "descobre o seu"},
			},
			run: b.euanime,
		},
		{
			name:     "ranking",
			category: estatisticas,
			help: []usage{
				{"!ranking", "mostra quem mais mandou mensagem no grupo na última semana 🏆"},
				{"!ranking hoje", "o ranking das últimas 24 horas"},
				{"!ranking mês", "o ranking dos últimos 30 dias"},
				{"!ranking tudo", "o ranking dos " + messageWindowLabel},
			},
			run: b.ranking,
		},
		{
			name:     "palavra",
			category: estatisticas,
			help: []usage{
				{"!palavra @pessoa", "mostra as palavras que a pessoa mais usa 🔤"},
				{"!palavra", "mostra as suas"},
			},
			run: b.palavra,
		},
		{
			name:     "tu",
			category: estatisticas,
			help: []usage{
				{"!tu", "mostra a sua ficha no grupo: nome, recado, ranking, palavra favorita, família e mais 🪪"},
				{"!tu @pessoa", "mostra a ficha da pessoa marcada"},
			},
			run: b.tu,
		},
		{
			name:     "palhaco",
			category: estatisticas,
			help: []usage{
				{"!palhaço", "mostra quem mais fez o grupo rir na última semana, contando os kkkk, ksksks e hahaha que a pessoa arrancou 🤡"},
				{"!palhaço hoje", "o ranking das últimas 24 horas (também tem mês e tudo)"},
			},
			run: b.palhaco,
		},
		{
			name:     "retrospectiva",
			aliases:  []string{"wrapped", "retro"},
			category: estatisticas,
			help: []usage{
				{"!retrospectiva", "o wrapped do grupo no ano: quantas mensagens, quem mais falou, dia mais movimentado, horário de pico, palavra do ano e o maior palhaço 🎁"},
				{"!retrospectiva @pessoa", "o wrapped da pessoa: mensagens, lugar no ranking, dia e horário favoritos, palavra mais usada, parceria de papo, risadas arrancadas e a primeira mensagem do ano"},
				{"!retrospectiva eu", "o seu wrapped"},
				{"!retrospectiva 2025", "a retrospectiva de outro ano (também dá pra marcar alguém)"},
			},
			run: b.retrospectiva,
		},
		{
			name:     "primeiramsg",
			aliases:  []string{"primeira", "primeiramensagem"},
			category: estatisticas,
			help: []usage{
				{"!primeiramsg @pessoa", "mostra a primeira mensagem da pessoa que eu tenho guardada no grupo, com a data 📜 (só guardo as dos " + messageWindowLabel + ")"},
				{"!primeiramsg", "mostra a sua"},
			},
			run: b.primeiramsg,
		},
		{
			name:     "horario",
			aliases:  []string{"horarios", "hora"},
			category: estatisticas,
			help: []usage{
				{"!horário", "mostra num gráfico em que hora do dia o grupo mais conversa na última semana ⏰"},
				{"!horário @pessoa", "o mesmo gráfico, só com as mensagens da pessoa (ou !horário eu pro seu)"},
				{"!horário hoje", "o gráfico das últimas 24 horas (também tem mês e tudo)"},
			},
			run: b.horario,
		},
		{
			name:     "kkk",
			category: zoeira,
			help: []usage{
				{"!kkk", "responda uma mensagem com !kkk e o bot manda a figurinha dos Simpsons rindo em cima dela 😂"},
			},
			run: b.kkk,
		},
		{
			name:     "chute",
			category: zoeira,
			help: []usage{
				{"!chute @pessoa", "você dá um chute na pessoa marcada 🦵"},
				{"!chute @pessoa1 @pessoa2", "a primeira pessoa marcada chuta a segunda 🦵"},
			},
			run: b.chute,
		},
		{
			name:     "morder",
			category: zoeira,
			help: []usage{
				{"!morder @pessoa", "você morde a pessoa marcada 🦷"},
				{"!morder @pessoa1 @pessoa2", "a primeira pessoa marcada morde a segunda 🦷"},
			},
			run: b.morder,
		},
		{
			name:     "arrogante",
			aliases:  []string{"arroghamty"},
			category: zoeira,
			help: []usage{
				{"!arrogante @pessoa", "zoa a pessoa marcada com o meme do Neymar: pra que jogar a bola na Mavie? 🥸"},
				{"!arroghamty @pessoa", "a mesma coisa, do jeito que o Neymar fala"},
				{"!arrogante", "chama você mesmo de arrogante"},
			},
			run: b.arrogante,
		},
		{
			name:     "ambrugui",
			category: zoeira,
			help: []usage{
				{"!ambrugui @pessoa", "a pessoa marcada come um ambrugui com ingredientes sorteados 🍔"},
				{"!ambrugui", "quem come o ambrugui é você"},
			},
			run: b.ambrugui,
		},
		{
			name:     "pedirpix",
			category: zoeira,
			help: []usage{
				{"!pedirpix @pessoa", "você pede um pix pra pessoa marcada, com valor e motivo sorteados 💸"},
				{"!pedirpix @pessoa1 @pessoa2", "a primeira pessoa marcada pede pix pra segunda"},
				{"!pedirpix", "você pede pix pra alguém aleatório do grupo"},
			},
			run: b.pedirpix,
		},
		{
			name:     "apelido",
			category: zoeira,
			help: []usage{
				{"!apelido @pessoa <apelido>", "dá um apelido pra pessoa marcada, e o bot mostra ele toda vez que marcar ela"},
				{"!apelido <apelido>", "dá um apelido pra você mesmo"},
				{"!apelido @pessoa", "tira o apelido da pessoa marcada"},
			},
			run: b.apelido,
		},
		{
			name:     "nome",
			category: zoeira,
			help: []usage{
				{"!nome @pessoa <nome>", "anota o nome de verdade da pessoa marcada, só pra referência: quem tem apelido continua aparecendo pelo apelido, mas a IA sabe o nome"},
				{"!nome <nome>", "anota o seu nome"},
				{"!nome @pessoa", "apaga o nome anotado da pessoa marcada"},
			},
			run: b.nome,
		},
		{
			name:     "frase",
			category: zoeira,
			help: []usage{
				{"!frase @pessoa", "o bot desenterra uma mensagem antiga da pessoa e manda como citação 📜"},
				{"!frase", "desenterra uma sua"},
			},
			run: b.frase,
		},
		{
			name:     "fotogato",
			aliases:  []string{"gato", "miau"},
			category: zoeira,
			help:     []usage{{"!fotogato", "manda uma foto aleatória de gato 🐱"}},
			run:      b.fotogato,
		},
		{
			name:     "saldo",
			aliases:  []string{"carteira", "banco"},
			category: economia,
			help: []usage{
				{"!saldo", "mostra quantas moedas você tem e sua posição no grupo 💰"},
				{"!saldo @pessoa", "mostra o saldo da pessoa marcada"},
			},
			run: b.saldo,
		},
		{
			name:     "daily",
			aliases:  []string{"diario"},
			category: economia,
			help:     []usage{{"!daily", fmt.Sprintf("pega de %d a %d moedas grátis, uma vez por dia 🎁", dailyMin, dailyMax)}},
			run:      b.daily,
		},
		{
			name:     "trabalhar",
			aliases:  []string{"trabalho", "work"},
			category: economia,
			help:     []usage{{"!trabalhar", "faz um trampo da sua profissão e ganha moedas 💼 Quanto ganha, de quanto em quanto tempo e o que pode acontecer no meio (pagar em dobro, achar item, dar ruim...) depende da profissão (veja !profissao)"}},
			run:      b.trabalhar,
		},
		{
			name:     "profissao",
			aliases:  []string{"profissoes", "emprego", "carreira"},
			category: economia,
			help: []usage{
				{"!profissao", fmt.Sprintf("mostra as %d profissões, separadas por área 💼 Cada uma tem seu efeito: umas roubam melhor, outras pescam sem vara, vendem mais caro, pagam menos pensão...", len(professions))},
				{"!profissao <área>", "mostra quanto paga e o que tem de especial cada profissão da área, tipo !profissao crime"},
				{"!profissao ver <nome>", "mostra os detalhes de uma profissão (só !profissao ver mostra a sua)"},
				{"!profissao <nome>", "vira essa profissão. Dá pra trocar uma vez a cada " + formatWait(professionChangeEvery)},
			},
			run: b.profissao,
		},
		{
			name:     "apostar",
			aliases:  []string{"aposta", "bet"},
			category: economia,
			help: []usage{
				{"!apostar <valor>", "cara ou coroa: dobra a aposta ou perde tudo 🪙"},
				{"!apostar tudo", "aposta todo o saldo (também aceita metade e porcentagem, tipo 10%)"},
			},
			run:          b.apostar,
			barredInJail: true,
		},
		{
			name:     "cassino",
			aliases:  []string{"slot", "tigrinho"},
			category: economia,
			help: []usage{
				{"!cassino <valor>", "gira o caça-níquel: par paga 2x, trinca 5x e 💎💎💎 paga 10x 🎰"},
			},
			run:          b.cassino,
			barredInJail: true,
		},
		{
			name:     "blackjack",
			aliases:  []string{"bj", "21"},
			category: economia,
			help: []usage{
				{"!blackjack <valor>", "joga uma mão de 21 contra a banca: chega mais perto de 21 que ela sem estourar e leva o dobro, e blackjack de cara paga 3 pra 2 🃏"},
				{"!blackjack carta / parar / dobrar", "pede mais uma carta, para, ou dobra a aposta e recebe só mais uma carta (só nas duas primeiras). Dá pra mandar só carta, parar ou dobrar, sem o !"},
				{"!blackjack", fmt.Sprintf("mostra a sua mão aberta. Se ficar %s sem jogar, o bot para por você", formatWait(blackjackIdle))},
			},
			run:          b.blackjack,
			barredInJail: true,
		},
		{
			name:     "crash",
			aliases:  []string{"foguete"},
			category: economia,
			help: []usage{
				{"!crash <valor>", fmt.Sprintf("entra no foguete 🚀 A primeira aposta abre a rodada e o resto tem %s pra entrar. Depois ele decola, o multiplicador sobe até explodir e quem saca antes leva a aposta vezes o multiplicador", formatSeconds(crashWindow))},
				{"!crash <valor> 2x", "entra já com saque automático quando chegar em 2x (também aceita 1.5x ou 1,5x)"},
				{"sacar", "saca na hora com o multiplicador atual, antes de explodir (!crash sacar também funciona)"},
			},
			run:          b.crash,
			barredInJail: true,
		},
		{
			name:     "corrida",
			aliases:  []string{"turfe"},
			category: economia,
			help: []usage{
				{"!corrida", fmt.Sprintf("abre uma corrida de cavalo 🏇 e mostra os %d cavalos com quanto cada um paga. Quem tem mais chance de ganhar paga menos", raceHorses)},
				{"!corrida <valor> <cavalo>", fmt.Sprintf("aposta no cavalo pelo número, tipo !corrida 100 3. Se não tiver corrida aberta, a aposta abre uma, e o resto tem %s pra apostar. Se o seu cavalo ganhar, você leva a aposta vezes o que ele paga", formatSeconds(raceWindow))},
			},
			run:          b.corrida,
			barredInJail: true,
		},
		{
			name:     "roubar",
			category: economia,
			help: []usage{
				{"!roubar @pessoa", fmt.Sprintf("tenta roubar a pessoa marcada: se der certo leva de 10%% a 30%% do saldo dela, se der errado paga indenização 🦹 Dá pra roubar a cada %s, e a profissão de cada um muda a espera e a chance (veja !profissao). Quem tem 🚨 alarme na mochila tá protegido", formatWait(robEvery))},
			},
			run:          b.roubar,
			barredInJail: true,
		},
		{
			name:     "assalto",
			aliases:  []string{"assaltar", "assaltobanco"},
			category: economia,
			help: []usage{
				{"!assalto", fmt.Sprintf("monta uma equipe de %d pra assaltar o banco 🏦 Quem manda primeiro abre, e o resto tem %s pra mandar !assalto e entrar. A chance base é %d%%, e a profissão de cada um soma ou tira (assaltante de banco, hacker e piloto de fuga ajudam, influencer e streamer atrapalham, veja !profissao). O equipamento na mochila de qualquer um da equipe também soma: %s. Cada tipo conta uma vez e é gasto no assalto. Se der certo, a equipe divide de %s a %s moedas. Se der errado, cada um tenta escapar e quem não consegue vai preso por %s (a profissão muda o tempo). Dá pra assaltar a cada %s", heistSize, formatWait(heistWindow), percent(heistChance), heistGearList(), thousands(heistLootMin), thousands(heistLootMax), formatWait(baseJail), formatWait(heistCooldown.every))},
			},
			run:          b.assalto,
			barredInJail: true,
		},
		{
			name:     "prender",
			aliases:  []string{"prisao", "cadeia"},
			category: economia,
			help: []usage{
				{"!prender @pessoa", "prende a pessoa marcada, se ela tentou !roubar, !furtar, !sequestrar, !matar, entrou num !assalto ou negociou no !mercadonegro nas últimas " + formatWait(crimeWindow) + " 🚔 Cada crime só rende uma prisão, e se ela tiver um refém no cativeiro ele é libertado. Enquanto tá na cadeia ela não pode !apostar, !cassino, !blackjack, !crash, !corrida, !roubar, !furtar, !sequestrar nem !assalto. Só a polícia prende (" + arrestingProfessions() + "), e a cadeia é de " + formatWait(baseJail) + ". A profissão de quem é preso aumenta ou diminui esse tempo (bandido pega mais, advogado pega menos, veja !profissao). Quem tá preso pode tentar !fugir, ou um juiz, advogado ou delegado pode tentar !libertar. Quem emprestou moedas com o !agiota pode prender quem passou do prazo sem pagar, mesmo sem ser da polícia e sem crime nenhum"},
			},
			run: b.prender,
		},
		{
			name:     "fugir",
			aliases:  []string{"fuga", "escapar"},
			category: economia,
			help: []usage{
				{"!fugir", fmt.Sprintf("tenta fugir da cadeia ou do cativeiro 🏃 Na cadeia a chance normal é de %d%%, e a profissão muda ela (piloto de fuga, mágico e chaveiro escapam fácil, e a polícia presa fica marcada). Se der certo sai na hora, mas fuga é crime e a polícia pode !prender de novo. Se der errado ganha mais %s de cadeia. No cativeiro a chance normal é de %d%%, e aqui a polícia, o segurança, o mágico e o chaveiro se dão bem (veja !profissao). Fugir de sequestro não é crime, mas se der errado fica mais %s amarrado(a). Dá pra tentar a cada %s, e cada nova prisão corta %d%% da espera que faltava", percent(escapeChance), formatWait(escapePenalty), percent(breakoutChance), formatWait(breakoutPenalty), formatWait(escapeCooldown.every), arrestEaseCut)},
			},
			run: b.fugir,
		},
		{
			name:     "libertar",
			aliases:  []string{"resgatar", "alvara"},
			category: economia,
			help: []usage{
				{"!libertar @pessoa", fmt.Sprintf("tenta tirar a pessoa marcada da cadeia ou do cativeiro 🔓 Só algumas profissões conseguem, cada uma com a sua chance: %s. Se a pessoa tiver presa e em cativeiro ao mesmo tempo, a cadeia vem primeiro. Se der certo ela sai na hora, e se der errado tem %d%% de chance de você levar uma multa de %d%% a %d%% do saldo (o presidente paga %d%% menos). Dá pra tentar a cada %s", releasingProfessions(), percent(releaseFineChance), releaseFineMin, releaseFineMax, percent(presidentFineDiscount), formatWait(releaseCooldown.every))},
			},
			run:          b.libertar,
			barredInJail: true,
		},
		{
			name:     "denunciar",
			aliases:  []string{"denuncia", "dedurar", "x9"},
			category: economia,
			help: []usage{
				{"!denunciar @pessoa", fmt.Sprintf("liga pro 190 e dedura a pessoa marcada, se ela tentou !roubar, !furtar, !sequestrar, !matar, entrou num !assalto ou negociou no !mercadonegro nas últimas %s 📞 Qualquer um pode denunciar, mas a polícia só acredita em %d%% das vezes, e a profissão muda isso (detetive, promotor e porteiro são levados a sério, golpista e cartomante quase nunca, veja !profissao). Se colar, a pessoa pega %s de cadeia (mais ou menos conforme a profissão dela) e solta o refém se tiver um. Bandido não pode denunciar: se tentar, a quadrilha entrega ele e quem vai preso é ele. Dá pra denunciar a cada %s", formatWait(crimeWindow), percent(reportChance), formatWait(baseJail), formatWait(reportCooldown.every))},
			},
			run: b.denunciar,
		},
		{
			name:     "processar",
			aliases:  []string{"processo"},
			category: economia,
			help: []usage{
				{"!processar @pessoa", fmt.Sprintf("leva pro tribunal quem te denunciou no !denunciar ou tentou te prender no !prender com a sua ficha limpa, nas últimas %s, ou quem te deve no !agiota e passou do prazo ⚖️ O grupo vota na enquete se a pessoa é culpada ou inocente. Se for condenada, paga %d%% do saldo (até %s) de indenização pra você, e se for absolvida você paga %d%% (até %s) de custas. A profissão de cada lado também pesa: advogado e promotor valem votos a mais no júri, quem é do crime tem ficha suja (veja no !profissao). Se for calote no !agiota, a condenação tira do saldo da pessoa o que der pra pagar a dívida, no lugar da indenização. Cada queixa rende um processo só, e dá pra abrir um a cada %s, menos pra quem é do meio jurídico", formatWait(grievanceWindow), trialFinePercent, coins(trialFineCeiling), trialCostsPercent, coins(trialCostsCeiling), formatWait(trialEvery))},
			},
			run: b.processar,
		},
		{
			name:     "votar",
			aliases:  []string{"voto", "eleger"},
			category: economia,
			help: []usage{
				{"!votar @pessoa", fmt.Sprintf("vota na pessoa marcada (ou em você mesmo, com !votar eu) pra presidente da conversa 🗳️ Se ninguém é presidente, o primeiro voto abre a eleição, e a urna fica aberta %s: cada um vota uma vez e pode mudar o voto até fechar. Precisa de pelo menos %d votos pra valer, e empate se resolve na sorte. Quem ganha fica %s no cargo, e só depois disso tem eleição de novo. Veja os poderes no !presidente", formatWait(electionWindow), electionQuorum, formatWait(termLength))},
			},
			run: b.votar,
		},
		{
			name:     "presidente",
			aliases:  []string{"presidencia", "eleicao", "mandato"},
			category: economia,
			help: []usage{
				{"!presidente", "mostra quem é o presidente da conversa, quanto falta do mandato, os abusos de poder que sobram e os decretos em vigor 🏛️ Se tiver eleição ou impeachment rolando, mostra a apuração parcial"},
			},
			run: b.presidente,
		},
		{
			name:     "decreto",
			aliases:  []string{"decretar"},
			category: economia,
			help: append([]usage{
				{"!decreto", "lista os decretos que o presidente pode baixar 📜 Só o presidente usa, e os decretos caem quando o mandato acaba"},
			}, decreeHelp()...),
			run: b.decreto,
		},
		{
			name:     "impeachment",
			aliases:  []string{"impeachar", "impichar", "impeach"},
			category: economia,
			help: []usage{
				{"!impeachment", fmt.Sprintf("vota pra tirar o presidente do cargo ⚖️ O primeiro voto abre a votação, que fica aberta %s: cada um vota uma vez e pode mudar o voto até fechar. Pra passar, precisa de pelo menos %d votos a favor e mais votos a favor do que contra, e aí o mandato acaba na hora, junto com os decretos. Dá pra pedir um impeachment a cada %s por mandato", formatWait(impeachmentWindow), impeachmentQuorum, formatWait(impeachmentEvery))},
				{"!impeachment nao", "vota contra o impeachment (sem nada, ou com !impeachment sim, é a favor)"},
			},
			run: b.impeachment,
		},
		{
			name:     "sequestrar",
			aliases:  []string{"sequestro", "raptar"},
			category: economia,
			help: []usage{
				{"!sequestrar @pessoa", fmt.Sprintf("tenta levar a pessoa marcada pro cativeiro 🚐 Se der certo ela fica %s sem poder !apostar, !cassino, !blackjack, !crash, !corrida, !roubar, !furtar nem !sequestrar, ou até ela conseguir !fugir ou a polícia !prender quem sequestrou. Se der errado você paga indenização e ainda pode ir preso na hora (bem mais fácil se a vítima for da polícia). Dá pra tentar a cada %s, um refém por vez, e a profissão de cada um muda a espera e a chance (veja !profissao). Se a vítima te deve no !agiota e passou do prazo, é cobrança: não é crime e você não vai preso na hora", formatWait(hostageHold), formatWait(kidnapCooldown.every))},
			},
			run:          b.sequestrar,
			barredInJail: true,
		},
		{
			name:     "venderrefem",
			aliases:  []string{"vendersequestrado"},
			category: economia,
			help: []usage{
				{"!venderrefem", fmt.Sprintf("leva o seu refém pro mercado clandestino 🤝 O negócio fecha em %s, e até lá a polícia ainda pode !prender você e libertar o refém, ou ele pode !fugir. O preço depende da profissão do refém (quem ganha mais vale mais) e da sua (contrabandista, cambista e golpista vendem mais caro). Depois de vendido, o refém fica no máximo mais %s trabalhando pra quem comprou, sem poder apostar, a não ser que consiga !fugir, e nem a polícia resgata mais", formatWait(hostageSaleWait), formatWait(soldHostageHold))},
			},
			run:          b.venderrefem,
			barredInJail: true,
		},
		{
			name:     "matar",
			aliases:  []string{"assassinar", "apagar"},
			category: economia,
			help: []usage{
				{"!matar @pessoa", fmt.Sprintf("tenta apagar a pessoa marcada 🔪 Só algumas profissões matam, cada uma com a sua chance: %s, e a proteção da vítima tira um pouco. Se der certo ela fica %s morta, sem poder usar nenhum comando de economia (a não ser que alguém consiga !reviver), e se tiver um refém no cativeiro ele fica livre. Se der errado você tem %d%% de chance de ir preso na hora por %s (bem mais se a vítima for da polícia, e a profissão muda o tempo). Matar ou tentar é crime: a polícia pode !prender e qualquer um pode !denunciar. Dá pra tentar a cada %s. Quem emprestou moedas com o !agiota pode matar quem passou do prazo sem pagar, mesmo sem ser dessas profissões, com pelo menos %d%% de chance, sem ser crime e sem ir preso", killingProfessions(), formatWait(deathHold), percent(killCaughtChance), formatWait(baseJail), formatWait(killCooldown.every), percent(debtKillChance))},
			},
			run:          b.matar,
			barredInJail: true,
		},
		{
			name:     "reviver",
			aliases:  []string{"ressuscitar", "ressucitar"},
			category: economia,
			help: []usage{
				{"!reviver @pessoa", fmt.Sprintf("tenta trazer de volta quem morreu no !matar 🧟 Só algumas profissões conseguem, cada uma com a sua chance: %s, e o presidente também revive. Se der certo a pessoa volta na hora e já pode usar a economia, e se der errado ela continua no caixão até o prazo acabar. Dá pra tentar a cada %s", revivingProfessions(), formatWait(reviveCooldown.every))},
			},
			run: b.reviver,
		},
		{
			name:     "abencoar",
			aliases:  []string{"bencao", "bencaos", "benzer"},
			category: economia,
			help: []usage{
				{"!abencoar", "mostra todas as bênçãos, quem pode dar cada uma e as que tão em você agora 🙏"},
				{"!abencoar @pessoa <bênção>", fmt.Sprintf("dá uma bênção pra pessoa por um tempo: mais ganho, mais proteção, mais sorte, mais chance de fugir... A Purificação tira na hora as maldições dela. Só as profissões de fé e feitiçaria abençoam, cada uma com as suas (%s), e ninguém se abençoa sozinho. Dá pra lançar uma bênção ou maldição a cada %s", mysticProfessions(), formatWait(charmCooldown.every))},
			},
			run: b.abencoar,
		},
		{
			name:     "amaldicoar",
			aliases:  []string{"maldicoes", "praguejar", "rogarpraga"},
			category: economia,
			help: []usage{
				{"!amaldicoar", "mostra todas as maldições, quem pode jogar cada uma e as que tão em você agora 🕯️"},
				{"!amaldicoar @pessoa <maldição>", fmt.Sprintf("joga uma maldição na pessoa por um tempo: ganha menos, fica mais fácil de roubar, perde sorte no crime, fica mais tempo preso... Cada profissão de fé e feitiçaria tem as suas e a sua chance de pegar, e quando falha a praga tem %d%% de chance de voltar pra quem jogou. Quem tá amaldiçoado se livra com uma !abencoar @pessoa purificacao", percent(hexBackfire))},
			},
			run: b.amaldicoar,
		},
		{
			name:     "milagre",
			aliases:  []string{"chover"},
			category: economia,
			help: []usage{
				{"!milagre <valor>", fmt.Sprintf("só pra deus ⚡ faz chover moeda nova pra todo mundo do grupo: cada um ganha o valor (até %s), e nada sai do seu saldo", coins(miracleMax))},
				{"!milagre <item> [quantidade]", fmt.Sprintf("só pra deus ✨ faz chover um item do !item pra todo mundo do grupo: cada um ganha a quantidade (até %d), criada do nada", creationMax)},
				{"!milagre <valor ou item> @pessoa", "só pra deus 🎯 o mesmo milagre, mas só pra quem você marcar"},
			},
			run: b.milagre,
		},
		{
			name:     "castigo",
			aliases:  []string{"castigar", "ira"},
			category: economia,
			help: []usage{
				{"!castigo <valor>", "só pra deus 🔥 faz sumir moedas de todo mundo do grupo: cada um perde o valor (ou o que tiver, se for menos), e as moedas não vão pra ninguém. Aceita número, porcentagem (!castigo 10%), metade ou tudo"},
				{"!castigo @pessoa <valor>", "mesma coisa, mas só com quem você marcar"},
			},
			run: b.castigo,
		},
		{
			name:     "criar",
			aliases:  []string{"conjurar", "materializar"},
			category: economia,
			help: []usage{
				{"!criar <item> [quantidade]", fmt.Sprintf("só pra deus ✨ cria qualquer item do !item do nada, direto na sua mochila, até %d de uma vez. Nada sai do seu saldo", creationMax)},
			},
			run: b.criar,
		},
		{
			name:     "transferir",
			aliases:  []string{"pagar", "pix"},
			category: economia,
			help: []usage{
				{"!transferir @pessoa <valor>", "manda moedas suas pra pessoa marcada 💸"},
			},
			run: b.transferir,
		},
		{
			name:     "conjunta",
			aliases:  []string{"contaconjunta"},
			category: economia,
			help: []usage{
				{"!conjunta", "mostra o saldo da conta conjunta com cada pessoa com quem você casou pelo pedido do modo consentimento 🏦"},
				{"!conjunta depositar <valor>", "guarda moedas suas na conta conjunta (aceita tudo, metade e 10%)"},
				{"!conjunta sacar <valor>", "tira moedas da conta conjunta pra sua carteira. Qualquer um dos dois pode sacar, e no divórcio o que sobrar é dividido meio a meio"},
				{"!conjunta depositar <valor> @pessoa", "escolhe com qual cônjuge, se você for casado(a) com mais de uma pessoa"},
			},
			run: b.conjunta,
		},
		{
			name:     "poupar",
			aliases:  []string{"poupanca", "guardar"},
			category: economia,
			help: []usage{
				{"!poupar <valor>", fmt.Sprintf("trava as moedas na poupança por %d horas e elas voltam com %d%% de juros, até %s por vez 🏦", int(savingsLock.Hours()), savingsRate, thousands(savingsCap))},
				{"!poupar", "mostra sua poupança e resgata quando já liberou"},
			},
			run: b.poupar,
		},
		{
			name:     "agiota",
			aliases:  []string{"emprestar", "emprestimo", "pediremprestado", "divida", "dividas"},
			category: economia,
			help: []usage{
				{"!agiota @pessoa <valor> [dias]", fmt.Sprintf("pede moedas emprestadas pra pessoa marcada, que vira seu agiota 🦈 Ela tem %s pra responder com !aceitar ou !recusar. O prazo padrão é de %d dias (dá pra pedir de 1 a %d). Juros de %d%% ao dia, compostos, já dentro do prazo, e depois dele sobem pra %d%% ao dia. Passou do prazo, o agiota pode !matar, !prender, !sequestrar ou !processar quem deve, sem ser crime e sem precisar da profissão", formatWait(offerWindow), debtTermDays, debtMaxDays, debtRate, debtLateRate)},
				{"!agiota pagar @pessoa [valor|tudo]", "paga parte ou toda a dívida com a pessoa marcada (se você só deve pra uma, nem precisa marcar)"},
				{"!agiota", "mostra quanto você deve e quanto te devem, com os juros e o prazo de cada dívida"},
			},
			run: b.agiota,
		},
		{
			name:     "acoes",
			aliases:  []string{"acao", "bovespa"},
			category: economia,
			help: []usage{
				{"!acoes", fmt.Sprintf("mostra a bolsa do grupo, com os preços que mexem a cada %d minutos, e a sua carteira com lucro ou prejuízo 📈", int(stockStepEvery.Minutes()))},
				{"!acoes comprar <empresa> <quantidade>", fmt.Sprintf("compra ações (ou 500 moedas, metade, tudo), com %d%% de corretagem", stockFeePercent)},
				{"!acoes vender <empresa> <quantidade|tudo>", "vende suas ações pelo preço de agora"},
			},
			run: b.acoes,
		},
		{
			name:     "loteria",
			category: economia,
			help: []usage{
				{"!loteria <quantidade>", fmt.Sprintf("compra bilhetes da loteria do grupo a %s cada, até %d por sorteio. Todo dia às %dh sai um ganhador, com chance proporcional aos bilhetes, que leva o pote inteiro com %d%% de bônus da casa 🎟️", coins(lotteryTicketPrice), lotteryMaxTickets, lotteryHour, lotteryHouseBonus)},
				{"!loteria", "mostra o pote, quantos bilhetes você tem, sua chance de ganhar e quanto falta pro sorteio"},
			},
			run: b.loteria,
		},
		{
			name:     "bolao",
			category: economia,
			help: []usage{
				{"!bolao criar <pergunta> | <opção 1> | <opção 2>", fmt.Sprintf("abre um bolão na conversa, com até %d opções separadas por | 🎯 Tipo !bolao criar Quem ganha o clássico? | Flamengo | Empate | Vasco", betPoolMaxOptions)},
				{"!bolao <número> <valor>", "aposta na opção (também aceita metade, 10% e tudo). Quem acertar divide o pote todo, proporcional ao que apostou"},
				{"!bolao", "mostra o bolão aberto, quanto tem em cada opção, quanto cada uma tá pagando e quanto você apostou"},
				{"!bolao fechar", "encerra as apostas (só quem abriu, admin do grupo ou dono do bot)"},
				{"!bolao resultado <número>", "diz qual opção ganhou e paga quem acertou. Se ninguém acertou, todo mundo recebe de volta"},
				{"!bolao cancelar", "cancela o bolão e devolve todas as apostas"},
			},
			run: b.bolao,
		},
		{
			name:     "ricos",
			aliases:  []string{"forbes"},
			category: economia,
			help:     []usage{{"!ricos", "mostra os mais ricos do grupo 🏆"}},
			run:      b.ricos,
		},
		{
			name:     "imposto",
			aliases:  []string{"impostos", "leao", "iptu"},
			category: economia,
			help: []usage{
				{"!imposto", fmt.Sprintf("mostra as faixas do imposto sobre fortuna e quanto você paga 🦁 Todo dia às %dh o Leão cobra de quem tem mais de %s, mais o IPTU de quem tem imóvel, e as moedas somem", taxHour, coins(taxBrackets[0].above))},
				{"!imposto @pessoa", "mostra quanto a pessoa marcada paga"},
			},
			run: b.imposto,
		},
		{
			name:     "fianca",
			aliases:  []string{"pagarfianca"},
			category: economia,
			help: []usage{
				{"!fianca", fmt.Sprintf("paga pra sair da cadeia na hora 🔓 O preço é uma parte do seu saldo, que sobe com o tempo que falta (de %d%% a %d%%, mínimo de %s), e o dinheiro some", exitBasePercent, exitMaxPercent, coins(exitFloor))},
				{"!fianca @pessoa", "paga a fiança de outra pessoa, com o preço calculado em cima do saldo dela"},
			},
			run: b.fianca,
		},
		{
			name:     "hospital",
			aliases:  []string{"medico", "upa"},
			category: economia,
			help: []usage{
				{"!hospital", fmt.Sprintf("paga o hospital particular pra voltar à vida na hora 🏥 Custa %d%% do saldo (mínimo de %s) e dá pra usar mesmo morto(a)", hospitalPercent, coins(hospitalFloor))},
				{"!hospital @pessoa", "paga o hospital pra trazer outra pessoa de volta"},
			},
			run:       b.hospital,
			whileDead: true,
		},
		{
			name:     "resgate",
			aliases:  []string{"pagarresgate"},
			category: economia,
			help: []usage{
				{"!resgate", fmt.Sprintf("paga o resgate e sai do cativeiro na hora 💼 Mesmo preço da fiança (de %d%% a %d%% do saldo, mínimo de %s), e o dinheiro some com os bandidos", exitBasePercent, exitMaxPercent, coins(exitFloor))},
				{"!resgate @pessoa", "paga o resgate de outra pessoa"},
			},
			run: b.resgate,
		},
		{
			name:     "seguranca",
			aliases:  []string{"guardacostas"},
			category: economia,
			help: []usage{
				{"!seguranca", "mostra se você tem segurança particular e quanto custa 🕶️"},
				{"!seguranca contratar", fmt.Sprintf("contrata um guarda-costas por %s: +%d%% de proteção contra !roubar, !sequestrar e !matar. Custa %d%% do saldo (mínimo de %s), e contratar de novo soma o tempo", formatWait(bodyguardHold), percent(bodyguardGuard), bodyguardPercent, coins(bodyguardFloor))},
			},
			run: b.seguranca,
		},
		{
			name:     "subornar",
			aliases:  []string{"suborno", "propina"},
			category: economia,
			help: []usage{
				{"!subornar", "mostra se você tem as autoridades no bolso e quanto custa o envelope 💰"},
				{"!subornar pagar", fmt.Sprintf("molha a mão das autoridades por %s: ninguém te prende (!prender, !denunciar, flagrante no !sequestrar, !matar, !assalto e batida no !mercadonegro) e o juiz decide a teu favor no !processar, a não ser que o outro lado também tenha pago. Custa %d%% do saldo (mínimo de %s), o dinheiro some e pagar de novo soma o tempo", formatWait(bribeHold), bribePercent, coins(bribeFloor))},
			},
			run: b.subornar,
		},
		{
			name:     "imovel",
			aliases:  []string{"imoveis", "casa", "mansao"},
			category: economia,
			help: []usage{
				{"!imovel", fmt.Sprintf("mostra os %d imóveis, da barraca de camping à base em Marte, e onde você mora 🏘️ Cada um dá proteção, bônus no !daily e cobra IPTU todo dia", len(estates))},
				{"!imovel comprar", "se muda pro próximo imóvel da lista. Paga o preço cheio e o antigo vai pro lixo"},
				{"!imovel @pessoa", "mostra onde a pessoa marcada mora"},
			},
			run: b.imovel,
		},
		{
			name:     "titulo",
			aliases:  []string{"titulos", "nobreza"},
			category: economia,
			help: []usage{
				{"!titulo", "mostra os títulos de nobreza do grupo, quem tem cada um e quanto custa 👑 Cada título tem um dono só e aparece antes do nome no !saldo e no !ricos"},
				{"!titulo comprar <nome>", fmt.Sprintf("compra o título, ou toma de quem tem pagando %d%% do que a pessoa pagou (ela recebe %d%% de volta)", titleStealPercent, titleRefundPercent)},
				{"!titulo usar <nome>", "troca o título que aparece no seu nome (!titulo usar nenhum tira)"},
			},
			run: b.titulo,
		},
		{
			name:     "luxo",
			aliases:  []string{"lojadeluxo", "grife"},
			category: economia,
			help: []usage{
				{"!luxo", fmt.Sprintf("mostra a loja de luxo de hoje 💎 São %d itens finos por dia, a %dx o valor de tabela. Cada item fino dá uma vantagem diferente: os que duram (relógio, carro, jatinho...) valem só de estar na mochila, e as comidas e bebidas dão uma bênção no !usar", luxuryStockSize, luxuryMarkup)},
				{"!luxo comprar <item> [quantidade]", "compra um item da loja de luxo (o !comprar também funciona)"},
			},
			run: b.luxo,
		},
		{
			name:     "mercadonegro",
			aliases:  []string{"mn", "submundo", "biqueira"},
			category: economia,
			help: []usage{
				{"!mercadonegro", fmt.Sprintf("mostra as armas e drogas do dia 🕶️ São %d itens por dia, e o preço varia de %d%% a %d%% do valor de tabela. Toda negociação conta como crime pro !prender e pro !denunciar, e tem %d%% de chance de dar batida: cadeia, multa e a mercadoria perdida. Quem é preso ainda perde as armas e drogas da mochila", blackMarketStockSize, blackMarketMinPercent, blackMarketMaxPercent, percent(raidChance))},
				{"!mercadonegro comprar <item> [quantidade]", "compra uma arma ou droga do estoque do dia (o !comprar também funciona). Arma funciona só de estar na mochila, droga é com !usar"},
				{"!mercadonegro vender <item> [quantidade]", fmt.Sprintf("vende arma ou droga pro receptador, a %d%% do preço do dia", blackMarketBuyback)},
			},
			run:          b.mercadonegro,
			barredInJail: true,
		},
		{
			name:     "doar",
			aliases:  []string{"obra", "obras", "vaquinha"},
			category: economia,
			help: []usage{
				{"!doar <valor>", "doa moedas pras obras do grupo 🏗️ Banco da praça, chafariz, estátua, estádio... e quem mais doar pra cada obra é homenageado nela. O dinheiro não volta"},
				{"!doar", "mostra a obra em construção, os maiores benfeitores e as obras prontas"},
				{"!doar @pessoa <item>", "dá um item de presente, igual ao !dar"},
			},
			run: b.doar,
		},
		{
			name:     "zoar",
			aliases:  []string{"zoeira", "trollar"},
			category: economia,
			help: []usage{
				{"!zoar", "mostra as zoeiras pagas e quanto cada uma custa pra você 🃏 O preço sobe com o seu saldo e as moedas somem"},
				{"!zoar palhaco @pessoa", "o bot reage com 🤡 em toda mensagem da pessoa por um tempo"},
				{"!zoar apelido @pessoa <apelido>", "dá um apelido e tranca por 24h, ninguém troca no !apelido"},
				{"!zoar maldicao @pessoa", "o !trabalhar da pessoa paga metade por um tempo"},
				{"!zoar outdoor <texto>", "o bot publica um outdoor com o seu texto"},
				{"!zoar limpar", "faz um descarrego e tira toda zoeira que tá em você"},
			},
			run: b.zoar,
		},
		{
			name:     "mochila",
			aliases:  []string{"inventario", "inv", "bolsa"},
			category: economia,
			help: []usage{
				{"!mochila", "mostra os itens que você tem e quanto eles valem 🎒"},
				{"!mochila @pessoa", "mostra a mochila da pessoa marcada"},
			},
			run: b.mochila,
		},
		{
			name:     "item",
			aliases:  []string{"itens", "catalogo"},
			category: economia,
			help: []usage{
				{"!item", fmt.Sprintf("mostra os %d itens que existem, separados por tipo, com a raridade na bolinha 📖", len(catalog))},
				{"!item <tipo>", "mostra só os itens de um tipo, separados por raridade, tipo !item pesca, !item armas ou !item comida"},
				{"!item <nome>", "mostra os detalhes de um item: raridade, valor, onde conseguir e pra que serve"},
			},
			run: b.item,
		},
		{
			name:     "loja",
			aliases:  []string{"shop"},
			category: economia,
			help:     []usage{{"!loja", "mostra o que dá pra comprar e quanto custa 🛒"}},
			run:      b.loja,
		},
		{
			name:     "petshop",
			aliases:  []string{"pets", "pet", "bichos"},
			category: economia,
			help:     []usage{{"!petshop", "mostra os bichinhos fofos que dá pra levar pra casa 🐾"}},
			run:      b.petshop,
		},
		{
			name:     "comprar",
			aliases:  []string{"buy"},
			category: economia,
			help: []usage{
				{"!comprar <item>", "compra um item da loja, tipo !comprar vara de pesca"},
				{"!comprar <item> <quantidade>", "compra vários de uma vez, tipo !comprar todynho 5"},
			},
			run: b.comprar,
		},
		{
			name:     "vender",
			aliases:  []string{"sell"},
			category: economia,
			help: []usage{
				{"!vender <item> [quantidade]", fmt.Sprintf("vende pro mercado na hora, mas ele só paga %d%% do valor 💸 (também aceita tudo)", marketSalePercent)},
				{"!vender @pessoa <item> <preço>", fmt.Sprintf("oferece o item pra pessoa marcada pelo preço que você quiser, e ela tem %s pra responder com !aceitar ou !recusar 🤝", formatWait(offerWindow))},
				{"!vender @pessoa <item> <quantidade> <preço>", "vende vários de uma vez, com o preço sendo o total"},
			},
			run: b.vender,
		},
		{
			name:     "aceitar",
			aliases:  []string{"aceito"},
			category: economia,
			help:     []usage{{"!aceitar", "fecha a compra que alguém te ofereceu com o !vender @você, ou empresta as moedas que alguém te pediu com o !agiota @você ou diz sim pro pedido de casamento ou namoro ✅"}},
			run:      b.aceitar,
		},
		{
			name:     "recusar",
			aliases:  []string{"recuso"},
			category: economia,
			help:     []usage{{"!recusar", "recusa a proposta, o pedido de empréstimo ou o pedido de casamento ou namoro que te fizeram, ou cancela os que você fez 🙅"}},
			run:      b.recusar,
		},
		{
			name:     "leilao",
			aliases:  []string{"leiloar"},
			category: economia,
			help: []usage{
				{"!leilao <item> [quantidade] [lance mínimo]", fmt.Sprintf("leiloa um item da sua mochila pro grupo por %s 🔨 O item fica preso no leilão até acabar. Sem lance mínimo, ele começa no que o mercado pagaria", formatWait(auctionWindow))},
				{"!leilao", "mostra o leilão rolando, o lance atual e quanto tempo falta"},
			},
			run: b.leilao,
		},
		{
			name:     "lance",
			category: economia,
			help: []usage{
				{"!lance <valor>", fmt.Sprintf("dá um lance no leilão aberto (!leilao <valor> também funciona). Tem que cobrir o anterior em pelo menos %d%% (mínimo %d moedas), as moedas saem na hora e voltam se alguém cobrir, e lance no último %s prorroga o leilão", auctionStepPercent, auctionMinStep, formatSeconds(auctionExtension))},
			},
			run: b.lance,
		},
		{
			name:     "dar",
			aliases:  []string{"presentear"},
			category: economia,
			help: []usage{
				{"!dar @pessoa <item> [quantidade]", "dá um item seu de presente pra pessoa marcada 🎁"},
			},
			run: b.dar,
		},
		{
			name:     "usar",
			aliases:  []string{"abrir", "comer", "beber", "tomar"},
			category: economia,
			help: []usage{
				{"!usar <item>", "usa um item: come, bebe, abre caixa, raspa raspadinha... cada um faz uma coisa (veja no !item <nome>) ✨ Dá pra mandar !comer, !beber ou !abrir também"},
			},
			run: b.usar,
		},
		{
			name:     "pescar",
			aliases:  []string{"pesca", "pescaria"},
			category: economia,
			help: []usage{
				{"!pescar", fmt.Sprintf("joga a linha e pesca um item, de lambari a escama de sereia 🎣 Precisa de vara de pesca (tem na !loja) e dá pra pescar a cada %s", formatWait(fishing.cooldown.every))},
			},
			run: b.gather(fishing),
		},
		{
			name:     "minerar",
			aliases:  []string{"mina", "mineracao"},
			category: economia,
			help: []usage{
				{"!minerar", fmt.Sprintf("desce na mina atrás de minério e pedra preciosa, e com sorte um meteorito ⛏️ Precisa de picareta (tem na !loja) e dá pra minerar a cada %s", formatWait(mining.cooldown.every))},
			},
			run: b.gather(mining),
		},
		{
			name:     "vasculhar",
			aliases:  []string{"lixao", "lixo", "garimpar"},
			category: economia,
			help: []usage{
				{"!vasculhar", fmt.Sprintf("revira o lixão atrás de tralha, e às vezes de coisa boa 🗑️ Não precisa de nada e dá pra vasculhar a cada %s", formatWait(scavenging.cooldown.every))},
			},
			run: b.gather(scavenging),
		},
		{
			name:     "cacar",
			aliases:  []string{"cacada", "insetos", "borboletas"},
			category: economia,
			help: []usage{
				{"!cacar", fmt.Sprintf("sai com a rede atrás de inseto, de pernilongo a fada no pote 🦋 Precisa de rede de insetos (tem na !loja) e dá pra caçar a cada %s", formatWait(bugging.cooldown.every))},
			},
			run: b.gather(bugging),
		},
		{
			name:     "furtar",
			aliases:  []string{"furto"},
			category: economia,
			help: []usage{
				{"!furtar @pessoa", fmt.Sprintf("tenta furtar um item aleatório da mochila da pessoa marcada 🥷 Quanto mais raro, mais difícil, e se der errado você paga indenização. Uma vez a cada %s, e a profissão de cada um muda a espera e a chance", formatWait(theftCooldown.every))},
				{"!furtar @pessoa <item>", "tenta furtar um item específico, com menos chance de dar certo. Quem tem 🔒 cadeado na mochila tá protegido"},
			},
			run:          b.furtar,
			barredInJail: true,
		},
		{
			name:     "ia",
			category: inteligencia,
			help: []usage{
				{"!ia <pergunta>", "pergunta qualquer coisa pra IA, que lê as últimas mensagens do grupo pra entender o assunto e pode guardar coisas na memória da conversa (pede pra ela lembrar ou esquecer algo)"},
			},
			run: b.ia,
		},
		{
			name:     "roast",
			category: inteligencia,
			help: []usage{
				{"!roast @pessoa", "a IA lê as mensagens da pessoa e zoa ela sem dó 🔥"},
				{"!roast", "a IA zoa você"},
			},
			run: b.roast,
		},
		{
			name:     "signo",
			category: inteligencia,
			help: []usage{
				{"!signo @pessoa", "a IA descobre o signo da pessoa pelo jeito que ela escreve, com ascendente e lua ♈"},
				{"!signo", "descobre o seu"},
			},
			run: b.signo,
		},
		{
			name:     "quemsou",
			category: inteligencia,
			help: []usage{
				{"!quemsou @pessoa", "a IA lê as mensagens da pessoa e diz qual personagem de anime combina com ela, com foto e tudo 🎭"},
				{"!quemsou", "descobre o seu (nenhum personagem repete no grupo até acabar a lista)"},
			},
			run: b.quemsou,
		},
		{
			name:     "quizia",
			category: inteligencia,
			help: []usage{
				{"!quizia", "a IA inventa uma pergunta de conhecimentos gerais e abre uma enquete pra ver quem acerta 🧠"},
				{"!quizia <tema>", "a pergunta é sobre esse tema, tipo !quizia futebol"},
			},
			run: b.quizia,
		},
		{
			name:     "emoji",
			aliases:  []string{"emojis"},
			category: inteligencia,
			help: []usage{
				{"!emoji", fmt.Sprintf("a IA descreve um filme, série, música ou meme só com emojis e quem mandar o nome primeiro acerta, e com o modo economia ligado ganha %s 🎬", coins(emojiReward))},
				{"!emoji <tema>", "a charada é sobre esse tema, tipo !emoji anime"},
			},
			run: b.emoji,
		},
		{
			name:     "resumo",
			category: inteligencia,
			help: []usage{
				{"!resumo", fmt.Sprintf("a IA lê as últimas %d mensagens e resume o que rolou na conversa 📝", summaryDefaultMessages)},
				{"!resumo <quantidade>", fmt.Sprintf("resume essa quantidade de mensagens, de %d a %d", summaryMinMessages, summaryMaxMessages)},
			},
			run: b.resumo,
		},
		{
			name:     "processaria",
			aliases:  []string{"tribunal", "julgar"},
			category: inteligencia,
			help: []usage{
				{"!processaria @pessoa <motivo>", "leva a pessoa marcada pro tribunal zoeiro do grupo: a IA faz a acusação e a defesa e o grupo vota na enquete se é culpada ou inocente ⚖️ É só zoeira, ninguém paga nada"},
			},
			run: b.processaria,
		},
		{
			name:     "cancelar",
			aliases:  []string{"cancela", "cancelamento"},
			category: inteligencia,
			help: []usage{
				{"!cancelar @pessoa", "a IA lê as mensagens da pessoa e monta uma thread de cancelamento no estilo Twitter, com prints tirados de contexto 📵"},
				{"!cancelar", "cancela você"},
			},
			run: b.cancelar,
		},
		{
			name:     "fofoca",
			aliases:  []string{"fofocar", "babado"},
			category: inteligencia,
			help: []usage{
				{"!fofoca", "a IA lê as últimas mensagens e inventa um babado com gente do grupo, de fontes nada confiáveis ☕"},
				{"!fofoca @pessoa", "o babado é sobre a pessoa marcada"},
			},
			run: b.fofoca,
		},
		{
			name:     "ficia",
			category: inteligencia,
			help: []usage{
				{"!ficia", "a IA escreve uma fic entre você e uma pessoa aleatória do grupo, imitando o jeito que cada um escreve 📖"},
				{"!ficia @pessoa", "a fic é entre você e a pessoa marcada"},
				{"!ficia @pessoa1 @pessoa2", "a fic é entre as duas pessoas marcadas"},
				{"!ficia @pessoa <tema>", "a fic segue o tema, tipo !ficia @pessoa1 @pessoa2 na praia"},
			},
			run: b.ficia,
		},
		{
			name:     "perseguir",
			aliases:  []string{"perseguiria"},
			category: enxerida,
			help: []usage{
				{"!perseguir @pessoa", "a IA passa a responder toda mensagem que a pessoa marcada mandar 🕵️"},
				{"!perseguir parar @pessoa", "a IA larga do pé da pessoa"},
			},
			run: b.perseguir,
		},
		{
			name:     "ban",
			aliases:  []string{"banir", "kick"},
			category: moderacao,
			help: []usage{
				{"!ban @pessoa", "tira a pessoa marcada do grupo 🔨 (só admin, e o bot também precisa ser admin)"},
				{"!ban", "responda uma mensagem com !ban e o bot bane quem mandou ela (também dá pra marcar várias pessoas)"},
			},
			run: b.ban,
		},
		{
			name:     "promover",
			aliases:  []string{"promote", "daradm"},
			category: moderacao,
			help: []usage{
				{"!promover @pessoa", "deixa a pessoa marcada como admin do grupo ⭐ (só admin, e o bot também precisa ser admin)"},
				{"!promover", "responda uma mensagem com !promover e o bot promove quem mandou ela"},
			},
			run: b.promover,
		},
		{
			name:     "rebaixar",
			aliases:  []string{"demote", "tiraradm"},
			category: moderacao,
			help: []usage{
				{"!rebaixar @pessoa", "tira o admin da pessoa marcada ⬇️ (só admin, e o bot também precisa ser admin; o dono do grupo não dá pra rebaixar)"},
				{"!rebaixar", "responda uma mensagem com !rebaixar e o bot tira o admin de quem mandou ela"},
			},
			run: b.rebaixar,
		},
		{
			name:     "mutar",
			aliases:  []string{"mute", "silenciar"},
			category: moderacao,
			help: []usage{
				{"!mutar @pessoa", "muta a pessoa marcada: o bot apaga toda mensagem que ela mandar no grupo 🔇 (só admin, e o bot também precisa ser admin)"},
				{"!mutar", "responda uma mensagem com !mutar e o bot muta quem mandou ela"},
			},
			run: b.mutar,
		},
		{
			name:     "desmutar",
			aliases:  []string{"unmute"},
			category: moderacao,
			help:     []usage{{"!desmutar @pessoa", "desmuta a pessoa e o bot para de apagar as mensagens dela 🔊 (só admin)"}},
			run:      b.desmutar,
		},
		{
			name:     "algemar",
			aliases:  []string{"prender"},
			category: moderacao,
			help: []usage{
				{"!algemar @pessoa", "algema a pessoa marcada: o bot reage com 🔒 em toda mensagem que ela mandar no grupo (só admin ou o dono do bot)"},
				{"!algemar", "responda uma mensagem com !algemar e o bot algema quem mandou ela"},
			},
			run: b.algemar,
		},
		{
			name:     "desalgemar",
			aliases:  []string{"soltar"},
			category: moderacao,
			help:     []usage{{"!desalgemar @pessoa", "tira as algemas da pessoa e o bot para de reagir nas mensagens dela 🔓 (só admin ou o dono do bot)"}},
			run:      b.desalgemar,
		},
		{
			name:     "advertir",
			aliases:  []string{"advertencia", "warn"},
			category: moderacao,
			help: []usage{
				{"!advertir @pessoa <motivo>", fmt.Sprintf("dá uma advertência pra pessoa marcada ⚠️ (o motivo é opcional). Na %dª o bot bane ela do grupo e zera a ficha (só admin ou o dono do bot)", warningLimit)},
				{"!advertir", "responda uma mensagem com !advertir <motivo> e o bot adverte quem mandou ela"},
				{"!advertir limpar @pessoa", "zera todas as advertências da pessoa (!advertir perdoar @pessoa tira só a última)"},
			},
			run: b.advertir,
		},
		{
			name:     "advertencias",
			aliases:  []string{"warns"},
			category: moderacao,
			help: []usage{
				{"!advertencias", "mostra quem tem advertência no grupo e quantas cada um tem 📋"},
				{"!advertencias @pessoa", "mostra as advertências da pessoa, com o motivo, quem deu e quando"},
			},
			run: b.advertencias,
		},
		{
			name:     "perdoar",
			aliases:  []string{"unwarn"},
			category: moderacao,
			help:     []usage{{"!perdoar @pessoa", "tira a última advertência da pessoa 🕊️ (só admin ou o dono do bot)"}},
			run:      b.perdoar,
		},
		{
			name:     "boasvindas",
			aliases:  []string{"bemvindo", "boasvinda"},
			category: moderacao,
			help: []usage{
				{"!boasvindas <texto>", "define a mensagem que o bot manda quando alguém entra no grupo 👋. Use {pessoa} pra marcar quem entrou e {grupo} pro nome do grupo; sem {pessoa}, o bot marca a pessoa no começo (só admin ou o dono do bot)"},
				{"!boasvindas", "mostra a mensagem de boas-vindas atual"},
				{"!boasvindas desligar", "apaga a mensagem e o bot para de dar boas-vindas"},
			},
			run: b.boasvindas,
		},
		{
			name:     "modos",
			category: ajuda,
			help:     []usage{{"!modos", "mostra todos os modos e quais estão ligados nesta conversa 🧩"}},
			run:      b.modos,
		},
		{
			name:     "baixar",
			aliases:  []string{"download", "dl"},
			category: utilidades,
			help: []usage{
				{"!baixar <link>", "baixa o vídeo, a imagem ou o gif do link e manda aqui 📥 (com o modo baixarlinks ligado, os links do Instagram, TikTok, X e outros nem precisam do comando)"},
				{"!baixar", "responda uma mensagem que tenha link e o bot baixa o que tem nele, ou responda uma figurinha e o bot manda ela como imagem (ou gif, se for animada)"},
				{"!baixar @pessoa", "baixa a foto de perfil da pessoa marcada e manda aqui 🖼️"},
			},
			run: b.baixar,
		},
		{
			name:     "mp3",
			aliases:  []string{"audio"},
			category: utilidades,
			help: []usage{
				{"!mp3 <link>", "baixa só o áudio do link e manda como arquivo mp3, com o nome e a capa 🎵"},
				{"!mp3", "responda um vídeo, um áudio ou uma mensagem com link e o bot manda o áudio em mp3"},
			},
			run: b.mp3,
		},
		{
			name:     "musica",
			aliases:  []string{"tocar", "play"},
			category: utilidades,
			help: []usage{
				{"!musica <nome>", "procura a música no YouTube e manda o áudio aqui 🎶 Dica: põe o artista junto, tipo !musica entombed deftones"},
				{"!musica <link>", "manda o áudio do link aqui pra tocar direto 🎧"},
				{"!musica meme <nome>", "manda um áudio de meme do MyInstants, igual o !som"},
			},
			run: b.musica,
		},
		{
			name:     "som",
			aliases:  []string{"sons"},
			category: utilidades,
			help: []usage{
				{"!som <nome>", "procura o áudio de meme no MyInstants e manda aqui 🔊 tipo !som vine boom"},
			},
			run: b.som,
		},
		{
			name:     "transcrever",
			aliases:  []string{"transcricao"},
			category: utilidades,
			help: []usage{
				{"!transcrever", "responda um áudio ou um vídeo com !transcrever e o bot escreve o que foi falado 🎙️"},
			},
			run: b.transcrever,
		},
		{
			name:     "figurinha",
			aliases:  []string{"fig", "s", "sticker"},
			category: utilidades,
			help: []usage{
				{"!figurinha", "mande uma foto, vídeo ou gif com !figurinha na legenda, ou responda um deles, e o bot transforma em figurinha 🖼️"},
			},
			run: b.figurinha,
		},
		{
			name:     "lembrete",
			aliases:  []string{"lembretes", "lembrar"},
			category: utilidades,
			help: []usage{
				{"!lembrete <hora> <texto>", fmt.Sprintf("cria um lembrete que o bot manda nesta conversa todo dia nessa hora, tipo !lembrete 8h30 tomar água ⏰ Dá pra marcar gente no texto, e cada conversa pode ter até %d", reminderMaxPerChat)},
				{"!lembrete", "mostra os lembretes da conversa, com a hora e quem criou cada um"},
				{"!lembrete apagar <número>", "apaga o lembrete com esse número na lista (só quem criou ou um admin do grupo)"},
			},
			run: b.lembrete,
		},
		{
			name:     "buscar",
			aliases:  []string{"procurar"},
			category: utilidades,
			help: []usage{
				{"!buscar <produto>", fmt.Sprintf("procura o produto no Buscapé, na Amazon e na KaBuM e manda os %d primeiros de cada um, com preço e link 🛒", shoppingPerShop)},
			},
			run: b.buscar,
		},
		{
			name:     "aniversario",
			aliases:  []string{"niver", "aniversarios"},
			category: utilidades,
			help: []usage{
				{"!aniversario <dd/mm>", fmt.Sprintf("salva o seu aniversário nesta conversa, e no dia o bot manda os parabéns aqui às %dh 🎂", birthdayHour)},
				{"!aniversario @pessoa <dd/mm>", "salva o aniversário de outra pessoa"},
				{"!aniversario", "mostra os próximos aniversários da conversa"},
				{"!aniversario apagar [@pessoa]", "apaga o seu aniversário, ou o de outra pessoa (só admin do grupo)"},
			},
			run: b.aniversario,
		},
		{
			name:     "cotacao",
			aliases:  []string{"dolar", "cambio"},
			category: utilidades,
			help: []usage{
				{"!cotacao", "mostra quanto tão o dólar, o euro, a libra, o bitcoin e o ethereum em reais, com a variação do dia 💱"},
				{"!cotacao <moeda>", "mostra a cotação de uma moeda pelo código ou pelo nome, tipo !cotacao ARS ou !cotacao iene"},
				{"!cotacao [moeda] em dolar", "mesma coisa, só que em dólar em vez de real, tipo !cotacao em dolar ou !cotacao bitcoin em dolar"},
			},
			run: b.cotacao,
		},
		{
			name:     "clima",
			aliases:  []string{"tempo", "previsao"},
			category: utilidades,
			help: []usage{
				{"!clima <cidade>", "mostra como tá o tempo agora na cidade e a previsão dos próximos dias 🌦️"},
			},
			run: b.clima,
		},
		{
			name:     "ocr",
			aliases:  []string{"ler", "texto"},
			category: utilidades,
			help: []usage{
				{"!ocr", "mande uma foto com !ocr na legenda, ou responda uma foto ou figurinha, e o bot escreve o texto que tiver nela 🔍"},
			},
			run: b.ocr,
		},
		{
			name:     "comandos",
			aliases:  []string{"menu"},
			category: ajuda,
			help: []usage{
				{"!comandos [tempo]", "mostra esta lista, que some depois do tempo escolhido (ex.: 30s, 5min, 2h; padrão 1 minuto)"},
				{"!menu [tempo]", "mostra esta lista, que some depois do tempo escolhido (ex.: 30s, 5min, 2h; padrão 2 minutos)"},
			},
			run: b.comandos,
		},
	}
}

func (b *bot) handleMessage(ctx context.Context, evt *events.Message) {
	if !b.awaitGames(ctx) {
		return
	}
	if b.deleteIfMuted(ctx, evt) {
		return
	}
	name, isCommand := commandName(messageText(evt.Message))
	if isCommand && b.runOwnerCommand(ctx, evt, name) {
		return
	}
	if b.answerPrivate(ctx, evt) {
		return
	}
	allowed, err := b.allowed(ctx, evt)
	if err != nil {
		b.client.Log.Errorf("checar blacklist/whitelist: %v", err)
		return
	}
	if !allowed {
		return
	}
	if err := b.messages.record(ctx, evt); err != nil {
		b.client.Log.Errorf("guardar mensagem: %v", err)
	}
	b.reactIfHandcuffed(ctx, evt)
	b.reactIfClowned(ctx, evt)
	if err := b.answerBmc(ctx, evt); err != nil {
		b.client.Log.Errorf("responder !bmc: %v", err)
	}
	if err := b.answerVdd(ctx, evt); err != nil {
		b.client.Log.Errorf("responder !vdd: %v", err)
	}
	if err := b.answerPoll(ctx, evt, &b.quemfalouRounds); err != nil {
		b.client.Log.Errorf("responder !quemfalou: %v", err)
	}
	if err := b.answerPoll(ctx, evt, &b.quizRounds); err != nil {
		b.client.Log.Errorf("responder !quiz: %v", err)
	}
	if err := b.answerPoll(ctx, evt, &b.mentiraPolls); err != nil {
		b.client.Log.Errorf("responder !2verdades1mentira: %v", err)
	}
	if err := b.answerPoll(ctx, evt, &b.impostorPolls); err != nil {
		b.client.Log.Errorf("responder !impostor: %v", err)
	}
	if err := b.answerPoll(ctx, evt, &b.trialPolls); err != nil {
		b.client.Log.Errorf("responder o tribunal: %v", err)
	}
	if err := b.answerImpostor(ctx, evt); err != nil {
		b.client.Log.Errorf("entrar no !impostor: %v", err)
	}
	if err := b.answerForca(ctx, evt); err != nil {
		b.client.Log.Errorf("responder !forca: %v", err)
	}
	if err := b.answerTermo(ctx, evt); err != nil {
		b.client.Log.Errorf("responder !termo: %v", err)
	}
	b.answerGuess(evt)
	if err := b.answerVelha(ctx, evt); err != nil {
		b.client.Log.Errorf("responder !velha: %v", err)
	}
	if err := b.answerBatata(ctx, evt); err != nil {
		b.client.Log.Errorf("responder !batata: %v", err)
	}
	if err := b.answerBlackjack(ctx, evt); err != nil {
		b.client.Log.Errorf("responder !blackjack: %v", err)
	}
	if err := b.answerCrash(ctx, evt); err != nil {
		b.client.Log.Errorf("responder !crash: %v", err)
	}
	if !isCommand {
		b.answerModes(ctx, evt)
		return
	}
	if b.answerProposal(ctx, evt, name) {
		return
	}
	commands := b.commands()
	i := slices.IndexFunc(commands, func(c botCommand) bool { return c.matches(name) })
	if i < 0 {
		if err := b.answerUnknownCommand(ctx, evt, name); err != nil {
			b.client.Log.Errorf("responder comando inexistente !%s: %v", name, err)
		}
		return
	}
	on, err := b.categoryEnabled(ctx, evt.Info.Chat, commands[i].category)
	if err != nil {
		b.client.Log.Errorf("checar modo do comando !%s: %v", name, err)
		return
	}
	if !on {
		return
	}
	if commands[i].category.mode == modeEconomy && !commands[i].whileDead && b.holdIfDead(ctx, evt) {
		return
	}
	if commands[i].barredInJail && (b.holdIfJailed(ctx, evt) || b.holdIfHostage(ctx, evt)) {
		return
	}
	if err := commands[i].run(ctx, evt); err != nil {
		b.client.Log.Errorf("comando !%s: %v", name, err)
	}
}

func messageText(msg *waE2E.Message) string {
	texts := []string{
		msg.GetConversation(),
		msg.GetExtendedTextMessage().GetText(),
		msg.GetImageMessage().GetCaption(),
		msg.GetVideoMessage().GetCaption(),
	}
	i := slices.IndexFunc(texts, func(text string) bool { return text != "" })
	if i < 0 {
		return ""
	}
	return texts[i]
}

func commandName(text string) (string, bool) {
	fields := strings.Fields(text)
	if len(fields) == 0 || !strings.HasPrefix(fields[0], "!") {
		return "", false
	}
	return strings.ToLower(strings.TrimPrefix(fields[0], "!")), true
}

func mentionedJIDs(msg *waE2E.Message) []types.JID {
	raw := msg.GetExtendedTextMessage().GetContextInfo().GetMentionedJID()
	jids := make([]types.JID, 0, len(raw))
	for _, s := range raw {
		jid, err := types.ParseJID(s)
		if err != nil {
			continue
		}
		jids = append(jids, jid)
	}
	return jids
}

func (b *bot) sendText(ctx context.Context, chat types.JID, text string) error {
	_, err := b.client.SendMessage(ctx, chat, &waE2E.Message{Conversation: proto.String(text)})
	return err
}

func (b *bot) sendPoll(ctx context.Context, chat types.JID, question string, options []string) error {
	_, err := b.client.SendMessage(ctx, chat, b.client.BuildPollCreation(question, options, 1))
	return err
}
