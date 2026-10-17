package ai

import (
	"fmt"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/economy"
)

func (b *bot) aiCommands() []core.Command {
	return []core.Command{
		{
			Name:     "ia",
			Category: core.AICategory,
			Help: []core.Usage{
				{"!ia <pergunta>", "pergunta qualquer coisa pra IA, que lê as últimas mensagens do grupo pra entender o assunto e pode guardar coisas na memória da conversa (pede pra ela lembrar ou esquecer algo)"},
			},
			Run: b.ia,
		},
		{
			Name:     "roast",
			Category: core.AICategory,
			Help: []core.Usage{
				{"!roast @pessoa", "a IA lê as mensagens da pessoa e zoa ela sem dó 🔥"},
				{"!roast", "a IA zoa você"},
			},
			Run: b.roast,
		},
		{
			Name:     "signo",
			Category: core.AICategory,
			Help: []core.Usage{
				{"!signo @pessoa", "a IA descobre o signo da pessoa pelo jeito que ela escreve, com ascendente e lua ♈"},
				{"!signo", "descobre o seu"},
			},
			Run: b.signo,
		},
		{
			Name:     "quemsou",
			Category: core.AICategory,
			Help: []core.Usage{
				{"!quemsou @pessoa", "a IA lê as mensagens da pessoa e diz qual personagem de anime combina com ela, com foto e tudo 🎭"},
				{"!quemsou", "descobre o seu (nenhum personagem repete no grupo até acabar a lista)"},
			},
			Run: b.quemsou,
		},
		{
			Name:     "quizia",
			Category: core.AICategory,
			Help: []core.Usage{
				{"!quizia", "a IA inventa uma pergunta de conhecimentos gerais e abre uma enquete pra ver quem acerta 🧠"},
				{"!quizia <tema>", "a pergunta é sobre esse tema, tipo !quizia futebol"},
			},
			Run: b.quizia,
		},
		{
			Name:     "emoji",
			Aliases:  []string{"emojis"},
			Category: core.AICategory,
			Help: []core.Usage{
				{"!emoji", fmt.Sprintf("a IA descreve um filme, série, música ou meme só com emojis e quem mandar o nome primeiro acerta, e com o modo economia ligado ganha %s 🎬", economy.Coins(emojiReward))},
				{"!emoji <tema>", "a charada é sobre esse tema, tipo !emoji anime"},
			},
			Run: b.emoji,
		},
		{
			Name:     "resumo",
			Category: core.AICategory,
			Help: []core.Usage{
				{"!resumo", fmt.Sprintf("a IA lê as últimas %d mensagens e resume o que rolou na conversa 📝", summaryDefaultMessages)},
				{"!resumo <quantidade>", fmt.Sprintf("resume essa quantidade de mensagens, de %d a %d", summaryMinMessages, summaryMaxMessages)},
			},
			Run: b.resumo,
		},
		{
			Name:     "processaria",
			Aliases:  []string{"tribunal", "julgar"},
			Category: core.AICategory,
			Help: []core.Usage{
				{"!processaria @pessoa <motivo>", "leva a pessoa marcada pro tribunal zoeiro do grupo: a IA faz a acusação e a defesa e o grupo vota na enquete se é culpada ou inocente ⚖️ É só zoeira, ninguém paga nada"},
			},
			Run: b.processaria,
		},
		{
			Name:     "cancelar",
			Aliases:  []string{"cancela", "cancelamento"},
			Category: core.AICategory,
			Help: []core.Usage{
				{"!cancelar @pessoa", "a IA lê as mensagens da pessoa e monta uma thread de cancelamento no estilo Twitter, com prints tirados de contexto 📵"},
				{"!cancelar", "cancela você"},
			},
			Run: b.cancelar,
		},
		{
			Name:     "fofoca",
			Aliases:  []string{"fofocar", "babado"},
			Category: core.AICategory,
			Help: []core.Usage{
				{"!fofoca", "a IA lê as últimas mensagens e inventa um babado com gente do grupo, de fontes nada confiáveis ☕"},
				{"!fofoca @pessoa", "o babado é sobre a pessoa marcada"},
			},
			Run: b.fofoca,
		},
		{
			Name:     "ficia",
			Category: core.AICategory,
			Help: []core.Usage{
				{"!ficia", "a IA escreve uma fic entre você e uma pessoa aleatória do grupo, imitando o jeito que cada um escreve 📖"},
				{"!ficia @pessoa", "a fic é entre você e a pessoa marcada"},
				{"!ficia @pessoa1 @pessoa2", "a fic é entre as duas pessoas marcadas"},
				{"!ficia @pessoa <tema>", "a fic segue o tema, tipo !ficia @pessoa1 @pessoa2 na praia"},
			},
			Run: b.ficia,
		},
		{
			Name:     "perseguir",
			Aliases:  []string{"perseguiria"},
			Category: core.MeddlingCategory,
			Help: []core.Usage{
				{"!perseguir @pessoa", "a IA passa a responder toda mensagem que a pessoa marcada mandar 🕵️"},
				{"!perseguir parar @pessoa", "a IA larga do pé da pessoa"},
			},
			Run: b.perseguir,
		},
	}
}
