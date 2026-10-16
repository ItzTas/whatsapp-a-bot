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
	forcaGames       sync.Map
	revolvers        sync.Map
	laughStickerSent sync.Map
	messages         *messageStore
	nicknames        *nicknameStore
	relationships    *relationshipStore
	adoptions        *adoptionStore
	assistant        *assistant
	filters          *filterStore
	modes            *modeStore
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
	name     string
	aliases  []string
	category category
	help     []usage
	run      command
}

func (c botCommand) matches(name string) bool {
	return c.name == name || slices.Contains(c.aliases, name)
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
		utilidades   = category{"🛠️ Utilidades", modeUtilities}
		ajuda        = category{"ℹ️ Ajuda", ""}
	)
	return []botCommand{
		{
			name:     "casar",
			category: romance,
			help: []usage{
				{"!casar @pessoa", "casa você com a pessoa marcada"},
				{"!casar @pessoa1 @pessoa2", "casa as duas pessoas marcadas"},
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
			name:     "familia",
			aliases:  []string{"família"},
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
			},
			run: b.namorar,
		},
		{
			name:     "terminar",
			aliases:  []string{"divorciar"},
			category: romance,
			help: []usage{
				{"!terminar @pessoa", "você termina o namoro e o casamento com a pessoa marcada 💔"},
				{"!terminar @pessoa1 @pessoa2", "termina o namoro e o casamento das duas pessoas marcadas"},
				{"!terminar", "você termina com todo mundo que namora ou com quem casou (!divorciar faz o mesmo)"},
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
			name:     "beijo",
			category: romance,
			help: []usage{
				{"!beijo @pessoa", "você dá um beijo na pessoa marcada 💋"},
				{"!beijo @pessoa1 @pessoa2", "a primeira pessoa marcada beija a segunda 💋"},
			},
			run: b.beijo,
		},
		{
			name:     "trair",
			category: romance,
			help: []usage{
				{"!trair @pessoa", "você trai quem você casou com a pessoa marcada 🐍"},
				{"!trair @pessoa1 @pessoa2", "a primeira pessoa marcada trai com a segunda"},
			},
			run: b.trair,
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
				{"!quem <pergunta>", "faz a sua pergunta, tipo !quem vai casar primeiro"},
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
			name:     "forca",
			category: jogos,
			help: []usage{
				{"!forca", "começa um jogo da forca no grupo, e depois é só mandar uma letra pra chutar 🪢"},
				{"!forca <palavra>", "chuta a palavra inteira"},
			},
			run: b.forca,
		},
		{
			name:     "roleta",
			category: jogos,
			help: []usage{
				{"!roleta", "roleta russa: 1 bala, 6 câmaras, e a chance de morrer sobe a cada rodada 🔫"},
				{"!roleta @pessoa", "aponta o revólver pra pessoa marcada"},
			},
			run: b.roleta,
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
			name:     "chato",
			category: medidores,
			help:     []usage{{"!chato @pessoa", "diz o quanto a pessoa é chata: fala demais, grita em caps, spamma comando, reclama de tudo 🙄"}},
			run:      b.meter(chatoMeter()),
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
			aliases:  []string{"palhaço"},
			category: estatisticas,
			help: []usage{
				{"!palhaço", "mostra quem mais fez o grupo rir na última semana, contando os kkkk, ksksks e hahaha que a pessoa arrancou 🤡"},
				{"!palhaço hoje", "o ranking das últimas 24 horas (também tem mês e tudo)"},
			},
			run: b.palhaco,
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
			name:     "frase",
			category: zoeira,
			help: []usage{
				{"!frase @pessoa", "o bot desenterra uma mensagem antiga da pessoa e manda como citação 📜"},
				{"!frase", "desenterra uma sua"},
			},
			run: b.frase,
		},
		{
			name:     "ia",
			category: inteligencia,
			help: []usage{
				{"!ia <pergunta>", "pergunta qualquer coisa pra IA, que lê as últimas mensagens do grupo pra entender o assunto"},
			},
			run: b.ia,
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
				{"!baixar <link>", "baixa o vídeo, a imagem ou o gif do link e manda aqui 📥 (links do Instagram, TikTok, X e outros nem precisam do comando, é só mandar)"},
				{"!baixar", "responda uma mensagem que tenha link e o bot baixa o que tem nele, ou responda uma figurinha e o bot manda ela como imagem (ou gif, se for animada)"},
			},
			run: b.baixar,
		},
		{
			name:     "mp3",
			aliases:  []string{"audio", "áudio"},
			category: utilidades,
			help: []usage{
				{"!mp3 <link>", "baixa só o áudio do link em mp3 🎵"},
				{"!mp3", "responda um vídeo, um áudio ou uma mensagem com link e o bot manda o áudio em mp3"},
			},
			run: b.mp3,
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
			name:     "comandos",
			aliases:  []string{"menu"},
			category: ajuda,
			help: []usage{
				{"!comandos", "mostra esta lista"},
				{"!menu", "mostra esta lista"},
			},
			run: b.comandos,
		},
	}
}

func (b *bot) handleMessage(ctx context.Context, evt *events.Message) {
	name, isCommand := commandName(messageText(evt.Message))
	if isCommand && b.runOwnerCommand(ctx, evt, name) {
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
	if err := b.answerBmc(ctx, evt); err != nil {
		b.client.Log.Errorf("responder !bmc: %v", err)
	}
	if err := b.answerVdd(ctx, evt); err != nil {
		b.client.Log.Errorf("responder !vdd: %v", err)
	}
	if err := b.answerQuemfalou(ctx, evt); err != nil {
		b.client.Log.Errorf("responder !quemfalou: %v", err)
	}
	if err := b.answerForca(ctx, evt); err != nil {
		b.client.Log.Errorf("responder !forca: %v", err)
	}
	if !isCommand {
		b.answerModes(ctx, evt)
		return
	}
	commands := b.commands()
	i := slices.IndexFunc(commands, func(c botCommand) bool { return c.matches(name) })
	if i < 0 {
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
