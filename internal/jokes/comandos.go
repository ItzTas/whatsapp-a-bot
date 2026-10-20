package jokes

import (
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
)

func (b *bot) jokesCommands() []core.Command {
	return []core.Command{
		{
			Name:     "kkk",
			Category: core.JokesCategory,
			Help: []core.Usage{
				{"!kkk", "responda uma mensagem com !kkk e o bot manda a figurinha dos Simpsons rindo em cima dela 😂"},
			},
			Run: b.kkk,
		},
		{
			Name:     "chute",
			Category: core.JokesCategory,
			Help: []core.Usage{
				{"!chute @pessoa", "você dá um chute na pessoa marcada 🦵"},
				{"!chute @pessoa1 @pessoa2", "a primeira pessoa marcada chuta a segunda 🦵"},
			},
			Run: b.chute,
		},
		{
			Name:     "morder",
			Category: core.JokesCategory,
			Help: []core.Usage{
				{"!morder @pessoa", "você morde a pessoa marcada 🦷"},
				{"!morder @pessoa1 @pessoa2", "a primeira pessoa marcada morde a segunda 🦷"},
			},
			Run: b.morder,
		},
		{
			Name:     "arrogante",
			Aliases:  []string{"arroghamty"},
			Category: core.JokesCategory,
			Help: []core.Usage{
				{"!arrogante @pessoa", "zoa a pessoa marcada com o meme do Neymar: pra que jogar a bola na Mavie? 🥸"},
				{"!arroghamty @pessoa", "a mesma coisa, do jeito que o Neymar fala"},
				{"!arrogante", "chama você mesmo de arrogante"},
			},
			Run: b.arrogante,
		},
		{
			Name:     "ambrugui",
			Category: core.JokesCategory,
			Help: []core.Usage{
				{"!ambrugui @pessoa", "a pessoa marcada come um ambrugui com ingredientes sorteados 🍔"},
				{"!ambrugui", "quem come o ambrugui é você"},
			},
			Run: b.ambrugui,
		},
		{
			Name:     "pedirpix",
			Category: core.JokesCategory,
			Help: []core.Usage{
				{"!pedirpix @pessoa", "você pede um pix pra pessoa marcada, com valor e motivo sorteados 💸"},
				{"!pedirpix @pessoa1 @pessoa2", "a primeira pessoa marcada pede pix pra segunda"},
				{"!pedirpix", "você pede pix pra alguém aleatório do grupo"},
			},
			Run: b.pedirpix,
		},
		{
			Name:     "apelido",
			Category: core.JokesCategory,
			Help: []core.Usage{
				{"!apelido @pessoa <apelido>", "dá um apelido pra pessoa marcada, e o bot mostra ele toda vez que marcar ela"},
				{"!apelido <apelido>", "dá um apelido pra você mesmo"},
				{"!apelido @pessoa", "tira o apelido da pessoa marcada"},
			},
			Run: b.apelido,
		},
		{
			Name:     "nome",
			Category: core.JokesCategory,
			Help: []core.Usage{
				{"!nome @pessoa <nome>", "anota o nome de verdade da pessoa marcada, só pra referência: quem tem apelido continua aparecendo pelo apelido, mas a IA sabe o nome. Quem não tem nome ganha o do zap sozinho quando entra ou manda mensagem"},
				{"!nome <nome>", "anota o seu nome"},
				{"!nome @pessoa", "apaga o nome anotado da pessoa marcada"},
			},
			Run: b.nome,
		},
		{
			Name:     "reagir",
			Aliases:  []string{"reacao", "emote"},
			Category: core.JokesCategory,
			Help: []core.Usage{
				{"!reagir @pessoa <emoji>", "o bot reage com o emoji em toda mensagem que a pessoa mandar, tipo !reagir @pessoa 🤡"},
				{"!reagir parar @pessoa", "o bot para de reagir às mensagens da pessoa"},
			},
			Run: b.reagir,
		},
		{
			Name:     "frase",
			Category: core.JokesCategory,
			Help: []core.Usage{
				{"!frase @pessoa", "o bot desenterra uma mensagem antiga da pessoa e manda como citação 📜"},
				{"!frase", "desenterra uma sua"},
			},
			Run: b.frase,
		},
		{
			Name:     "fotogato",
			Aliases:  []string{"gato", "miau"},
			Category: core.JokesCategory,
			Help:     []core.Usage{{"!fotogato", "manda uma foto aleatória de gato 🐱"}},
			Run:      b.fotogato,
		},
	}
}
