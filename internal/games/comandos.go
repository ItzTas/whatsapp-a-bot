package games

import (
	"fmt"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/economy"
)

func (b *bot) gamesCommands() []core.Command {
	return []core.Command{
		{
			Name:     "bmc",
			Category: core.GamesCategory,
			Help: []core.Usage{
				{"!bmc", "pergunta pra você quem você beija, mata ou casa entre 3 pessoas do grupo"},
				{"!bmc @pessoa", "faz a pergunta pra pessoa marcada"},
			},
			Run: b.bmc,
		},
		{
			Name:     "vdd",
			Category: core.GamesCategory,
			Help: []core.Usage{
				{"!vdd", "o bot te pergunta verdade ou desafio 🔥"},
				{"!vdd @pessoa", "faz a pergunta pra pessoa marcada"},
			},
			Run: b.vdd,
		},
		{
			Name:     "eununca",
			Category: core.GamesCategory,
			Help:     []core.Usage{{"!euNunca", "abre uma enquete de eu nunca 🙊"}},
			Run:      b.euNunca,
		},
		{
			Name:     "prefere",
			Category: core.GamesCategory,
			Help:     []core.Usage{{"!prefere", "abre uma enquete de você prefere 🤔"}},
			Run:      b.prefere,
		},
		{
			Name:     "quem",
			Category: core.GamesCategory,
			Help: []core.Usage{
				{"!quem", "abre uma enquete de quem é mais provável, com gente do grupo como opção 👀"},
				{"!quem <pergunta>", "a sua pergunta vira a enquete, tipo !quem vai casar primeiro ou !quem é o mais gado daqui"},
			},
			Run: b.quem,
		},
		{
			Name:     "quemfalou",
			Category: core.GamesCategory,
			Help: []core.Usage{
				{"!quemfalou", "o bot desenterra uma mensagem antiga e abre uma enquete pra adivinhar quem mandou 🕵️"},
				{"!quemfalou <frase>", "em vez de sortear, usa uma mensagem que tenha essa frase"},
			},
			Run: b.quemfalou,
		},
		{
			Name:     "completa",
			Aliases:  []string{"completar"},
			Category: core.GamesCategory,
			Help: []core.Usage{
				{"!completa", fmt.Sprintf("o bot desenterra uma mensagem antiga e esconde uma palavra: quem mandar ela primeiro acerta, e com o modo economia ligado ganha %s 🧩", economy.Coins(completaReward))},
			},
			Run: b.completa,
		},
		{
			Name:     "forca",
			Category: core.GamesCategory,
			Help: []core.Usage{
				{"!forca", "começa um jogo da forca no grupo, e depois é só mandar uma letra pra chutar 🪢"},
				{"!forca <palavra>", "chuta a palavra inteira"},
			},
			Run: b.forca,
		},
		{
			Name:     "termo",
			Category: core.GamesCategory,
			Help: []core.Usage{
				{"!termo", "começa um termo (o wordle em português): 6 chances pro grupo acertar a palavra de 5 letras 🟩🟨"},
				{"!termo <palavra>", "chuta uma palavra (uma palavra de 5 letras mandada sozinha também vale)"},
			},
			Run: b.termo,
		},
		{
			Name:     "quiz",
			Category: core.GamesCategory,
			Help: []core.Usage{
				{"!quiz", fmt.Sprintf("abre uma enquete com uma pergunta de conhecimentos gerais, e com o modo economia ligado quem acerta ganha %s 🎲", economy.Coins(economy.QuizReward))},
			},
			Run: b.quiz,
		},
		{
			Name:     "velha",
			Aliases:  []string{"jogodavelha"},
			Category: core.GamesCategory,
			Help: []core.Usage{
				{"!velha @pessoa", "desafia a pessoa marcada pro jogo da velha, e depois é só mandar o número da casa (1 a 9) pra jogar ❌⭕"},
				{"!velha", "joga contra o bot"},
				{"!velha desistir", "desiste do jogo que tá rolando"},
			},
			Run: b.velha,
		},
		{
			Name:     "roleta",
			Category: core.GamesCategory,
			Help: []core.Usage{
				{"!roleta", "roleta russa: 1 bala, 6 câmaras, e a chance de morrer sobe a cada rodada 🔫"},
				{"!roleta @pessoa", "aponta o revólver pra pessoa marcada"},
				{"!roleta <valor>", "com o modo economia ligado, aposta moedas que quem levar o tiro sobrevive: quanto menos câmaras sobrando, mais paga 💰"},
			},
			Run: b.roleta,
		},
		{
			Name:     "impostor",
			Category: core.GamesCategory,
			Help: []core.Usage{
				{"!impostor", fmt.Sprintf("abre a sala do jogo do impostor, ou entra na que tá aberta (também dá pra entrar respondendo a mensagem da sala só com !) 🕵️ Todo mundo recebe a mesma palavra secreta no privado, menos o impostor, que recebe uma parecida e nem sabe que é ele. Aí rolam rodadas de dicas: na sua vez eu te marco e você manda UMA palavra no grupo. No fim de cada rodada uma enquete decide se já dá pra votar em quem é o impostor ou se vai mais uma (no máximo %d rodadas). Com o modo economia ligado, se pegarem o impostor cada um ganha %s, e se ele escapar leva %s", impostorMaxRounds, economy.Coins(impostorGroupReward), economy.Coins(impostorSoloReward))},
				{"!impostor começar", fmt.Sprintf("começa sem esperar o tempo da sala (mínimo de %d jogadores)", impostorMinPlayers)},
				{"!impostor sair", "sai da sala antes do jogo começar (quem abriu a sala, ou o primeiro da lista se ele sair, pode usar !impostor cancelar)"},
			},
			Run: b.impostor,
		},
		{
			Name:     "batata",
			Category: core.GamesCategory,
			Help: []core.Usage{
				{"!batata", "começa a batata quente com você segurando: pra passar, manda qualquer mensagem marcando alguém, e ela explode numa hora que ninguém sabe 🥔🔥"},
				{"!batata @pessoa", "começa com a batata na mão da pessoa marcada, ou passa a batata pra ela se tiver na sua mão"},
			},
			Run: b.batata,
		},
		{
			Name:     "stop",
			Aliases:  []string{"adedanha"},
			Category: core.GamesCategory,
			Help: []core.Usage{
				{"!stop", "começa um stop (adedanha): o bot sorteia uma letra e 5 categorias, e quem completar tudo primeiro pede STOP ✋"},
				{"!stop <respostas>", fmt.Sprintf("manda as 5 respostas na ordem, separadas por vírgula ou uma por linha. Resposta única vale 10 pontos, repetida vale 5, e com o modo economia ligado quem fizer mais pontos ganha %s", economy.Coins(stopReward))},
			},
			Run: b.stop,
		},
		{
			Name:     "jokenpo",
			Aliases:  []string{"ppt", "pedrapapeltesoura"},
			Category: core.GamesCategory,
			Help: []core.Usage{
				{"!jokenpo pedra", "joga pedra, papel ou tesoura contra o bot na hora ✊✋✌️"},
				{"!jokenpo @pessoa", fmt.Sprintf("desafia a pessoa marcada: o bot chama vocês dois no privado, cada um manda a jogada lá em segredo (vale ✊ ✋ ✌️ também) e o resultado sai no grupo. Quem não jogar em %s perde por W.O.", core.FormatWait(jokenpoWindow))},
				{"!jokenpo @pessoa <valor>", "com o modo economia ligado, vale moedas: a sua aposta sai na hora, a pessoa aceita mandando a jogada no privado, quem ganhar leva o dobro e empate devolve 💰"},
			},
			Run: b.jokenpo,
		},
		{
			Name:     "2verdades1mentira",
			Aliases:  []string{"2v1m", "mentira"},
			Category: core.GamesCategory,
			Help: []core.Usage{
				{"!2verdades1mentira", fmt.Sprintf("o bot te chama no privado pra contar 3 coisas sobre você, 2 verdades e 1 mentira, e abre uma enquete no grupo pra descobrirem qual é a mentira 🤥 Com o modo economia ligado, quem acerta ganha %s", economy.Coins(mentiraReward))},
			},
			Run: b.mentira,
		},
		{
			Name:     "bandeira",
			Aliases:  []string{"bandeiras", "pais"},
			Category: core.GamesCategory,
			Help: []core.Usage{
				{"!bandeira", fmt.Sprintf("o bot manda a bandeira de um país e quem mandar o nome dele primeiro acerta, e com o modo economia ligado ganha %s 🏳️", economy.Coins(bandeiraReward))},
			},
			Run: b.bandeira,
		},
	}
}
