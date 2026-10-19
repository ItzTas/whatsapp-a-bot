package economy

import (
	"fmt"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
)

func (b *bot) economyCommands() []core.Command {
	return []core.Command{
		{
			Name:     "saldo",
			Aliases:  []string{"carteira", "banco"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!saldo", "mostra quantas moedas você tem e sua posição no grupo 💰"},
				{"!saldo @pessoa", "mostra o saldo da pessoa marcada"},
			},
			Run: b.saldo,
		},
		{
			Name:     "daily",
			Aliases:  []string{"diario"},
			Category: core.EconomyCategory,
			Help:     []core.Usage{{"!daily", fmt.Sprintf("pega de %d a %d moedas grátis, uma vez por dia 🎁 Quem tem bichinho do !petshop às vezes ganha um item que ele traz", dailyMin, dailyMax)}},
			Run:      b.daily,
		},
		{
			Name:     "missoes",
			Aliases:  []string{"missao", "metas"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!missoes", fmt.Sprintf("mostra as suas %d missões do dia 🎯 Cada pessoa tem as suas, tipo trabalhar umas vezes, ganhar no !cassino ou vender pro mercado, e elas trocam à meia-noite. A recompensa cai sozinha quando a missão fecha, e quem fecha as %d ainda leva mais %s e %s", missionsPerDay, missionsPerDay, Coins(missionBonus), missionPrize().label())},
				{"!missoes @pessoa", "mostra as missões da pessoa marcada"},
			},
			Run: b.missoes,
		},
		{
			Name:     "eventos",
			Aliases:  []string{"evento", "aventuras"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!eventos", fmt.Sprintf("mostra o seu diário de eventos 📖 Em %d%% das vezes, o !trabalhar, o %s e os crimes soltam um evento: o bot conta uma situação e manda uma enquete, e só quem tá no evento escolhe o que fazer (também dá pra responder a mensagem com o número da opção). Cada escolha pode dar moeda, item, droga, ferimento, cadeia ou até morte, e quem não escolhe em %s fica parado e aceita o que vier. O diário mostra quantos eventos você já descobriu de cada tipo e os raros que já viveu", percent(eventChance), extractionCommands(), core.FormatWait(eventIdle))},
				{"!eventos @pessoa", "mostra o diário de eventos da pessoa marcada"},
			},
			Run: b.eventos,
		},
		{
			Name:         "pegar",
			Aliases:      []string{"agarrar"},
			Category:     core.LiveCategory,
			BarredInJail: true,
			Help: []core.Usage{
				{"!pegar", fmt.Sprintf("só vale na corrida ao vivo 🏁 De vez em quando cai alguma coisa no grupo (um malote, um baú, um pacote do aeroporto...) e quem mandar !pegar primeiro leva. Às vezes é isca e dá ruim. Fica valendo por %s", core.FormatWait(grabWindow))},
				{"!pegar <resposta>", "quando o prêmio tem senha (uma conta, uma palavra embaralhada ou uns emojis pra contar), só leva quem mandar a resposta certa primeiro"},
			},
			Run: b.pegar,
		},
		{
			Name:         "atacar",
			Aliases:      []string{"bater", "lutar"},
			Category:     core.LiveCategory,
			BarredInJail: true,
			Help: []core.Usage{
				{"!atacar", fmt.Sprintf("só vale com um chefão ao vivo no grupo 👹 Cada golpe tira de %d a %d de vida (mais com arma na profissão, e às vezes sai crítico), uma vez a cada %s, e o chefão pode revidar. Se ele cair em até %s, todo mundo que bateu leva prêmio, maior pra quem deu mais dano, e quem deu mais dano ainda leva um troféu. Se escapar, quem bateu paga o pato", bossHitMin, bossHitMax, core.FormatWait(bossRest), core.FormatWait(bossWindow))},
			},
			Run: b.atacar,
		},
		{
			Name:     "aovivo",
			Aliases:  []string{"invocar"},
			Category: core.LiveCategory,
			Help: []core.Usage{
				{"!aovivo", "só pra deus 📡 solta um evento ao vivo agora, sem esperar ele aparecer sozinho"},
				{"!aovivo <tipo>", "escolhe o tipo: !aovivo corrida, !aovivo chefão ou !aovivo enquete"},
			},
			Run: b.invocar,
		},
		{
			Name:      "dica",
			Aliases:   []string{"dicas"},
			Category:  core.EconomyCategory,
			WhileDead: true,
			Help: []core.Usage{
				{"!dica", "manda uma dica da economia na hora 💡 Pode ser um jeito de achar relíquia mais fácil, de escapar da cadeia, de proteger o dinheiro, ou um comando que pouca gente conhece. Com o submodo dicas ligado (!modo ativar dicas), o bot solta uma sozinho de tempos em tempos"},
				{"!dica <assunto>", "manda uma dica sobre o assunto, tipo !dica relíquia, !dica cadeia ou !dica pescar"},
			},
			Run: b.dica,
		},
		{
			Name:     "trabalhar",
			Aliases:  []string{"trabalho", "work"},
			Category: core.EconomyCategory,
			Help:     []core.Usage{{"!trabalhar", "faz um trampo da sua profissão e ganha moedas 💼 Quanto ganha, de quanto em quanto tempo e o que pode acontecer no meio (pagar em dobro, achar item, dar ruim...) depende da profissão (veja !profissao). Algumas profissões são perigosas e dá pra se machucar no trampo: veja o risco da sua no !saude"}},
			Run:      b.trabalhar,
		},
		{
			Name:     "postar",
			Aliases:  []string{"post", "postagem", "gravar"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!postar <rede>", fmt.Sprintf("posta no %s e ganha (ou perde) seguidores 📲 Dá pra postar a cada %s, numa rede por vez, e precisa de um aparelho na mochila (o 📱 Celular posta no TikTok e no Instagram, o 💻 Computador no YouTube e na Twitch, os dois na !loja). Cada rede tem o seu jeito: o TikTok viraliza e cancela mais, o YouTube cresce devagar e firme, o Instagram fica no meio e a Twitch perde seguidor rápido quando você some. Quanto rende depende da profissão: quem é de Arte e mídia alcança mais gente, e youtuber, streamer, influencer e mais alguns têm a rede deles (veja !profissao). Ficar famoso é muito difícil, e rede parada perde seguidor", platformNames(), core.FormatWait(postCooldown.every))},
				{"!postar", "mostra as redes e o jeito de cada uma"},
			},
			Run: b.postar,
		},
		{
			Name:     "fama",
			Aliases:  []string{"seguidores", "famoso", "famosa"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!fama", "mostra seus seguidores em cada rede, o total e o seu nível de fama 🌟 Os níveis são " + fameTierList() + ". A partir de conhecido(a) no bairro já dá pra fazer !publi, mas fama tem preço: quem te sequestra vende mais caro, a polícia acredita mais em denúncia contra você e fugir da cadeia fica mais difícil, cada vez mais conforme o nível"},
				{"!fama @pessoa", "mostra a fama da pessoa marcada"},
			},
			Run: b.fama,
		},
		{
			Name:     "publi",
			Aliases:  []string{"publicidade", "patrocinio", "publipost"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!publi", fmt.Sprintf("faz publicidade pros seus seguidores e ganha moedas 💼 Precisa de pelo menos %s no total (veja !fama). Quanto mais seguidor, maior a marca e o cachê, e o Instagram é a rede que paga melhor por seguidor. O cachê tem teto, que sobe com o nível de fama: vai de %s pra quem é %s até %s pra quem é %s. Dá pra fazer a cada %s, mas tem %d%% de chance da galera achar forçado e você perder de %d%% a %d%% dos seguidores", followerCount(fameTiers[1].min), Coins(fameTiers[1].ceiling), fameTiers[1].name, Coins(fameTiers[len(fameTiers)-1].ceiling), fameTiers[len(fameTiers)-1].name, core.FormatWait(sponsorCooldown.every), percent(sponsorBacklash), backlashMin, backlashMax)},
			},
			Run: b.publi,
		},
		{
			Name:     "profissao",
			Aliases:  []string{"profissoes", "emprego", "carreira"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!profissao", fmt.Sprintf("mostra as %d profissões, separadas por área 💼 Cada uma tem seu efeito: umas roubam melhor, outras pescam sem vara, vendem mais caro, pagam menos pensão...", len(professions))},
				{"!profissao <área>", "mostra quanto paga e o que tem de especial cada profissão da área, tipo !profissao crime"},
				{"!profissao ver <nome>", "mostra os detalhes de uma profissão (só !profissao ver mostra a sua)"},
				{"!profissao <nome>", "vira essa profissão. Dá pra trocar uma vez a cada " + core.FormatWait(professionChangeEvery) + ", e quem tá na cadeia ou em cativeiro não troca"},
			},
			Run: b.profissao,
		},
		{
			Name:     "apostar",
			Aliases:  []string{"aposta", "bet"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!apostar <valor>", "cara ou coroa: dobra a aposta ou perde tudo 🪙"},
				{"!apostar tudo", "aposta todo o saldo (também aceita metade e porcentagem, tipo 10%)"},
			},
			Run:          b.apostar,
			BarredInJail: true,
		},
		{
			Name:     "cassino",
			Aliases:  []string{"slot", "tigrinho"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!cassino <valor>", "gira o caça-níquel: par paga 2x, trinca 5x e 💎💎💎 paga 10x 🎰"},
			},
			Run:          b.cassino,
			BarredInJail: true,
		},
		{
			Name:     "blackjack",
			Aliases:  []string{"bj", "21"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!blackjack <valor>", "joga uma mão de 21 contra a banca: chega mais perto de 21 que ela sem estourar e leva o dobro, e blackjack de cara paga 3 pra 2 🃏"},
				{"!blackjack carta / parar / dobrar", "pede mais uma carta, para, ou dobra a aposta e recebe só mais uma carta (só nas duas primeiras). Dá pra mandar só carta, parar ou dobrar, sem o !"},
				{"!blackjack", fmt.Sprintf("mostra a sua mão aberta. Se ficar %s sem jogar, o bot para por você", core.FormatWait(blackjackIdle))},
			},
			Run:          b.blackjack,
			BarredInJail: true,
		},
		{
			Name:     "crash",
			Aliases:  []string{"foguete"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!crash <valor>", fmt.Sprintf("entra no foguete 🚀 A primeira aposta abre a rodada e o resto tem %s pra entrar. Depois ele decola, o multiplicador sobe até explodir e quem saca antes leva a aposta vezes o multiplicador", formatSeconds(crashWindow))},
				{"!crash <valor> 2x", "entra já com saque automático quando chegar em 2x (também aceita 1.5x ou 1,5x)"},
				{"sacar", "saca na hora com o multiplicador atual, antes de explodir (!sacar e !crash sacar também funcionam)"},
			},
			Run:          b.crash,
			BarredInJail: true,
		},
		{
			Name:     "corrida",
			Aliases:  []string{"turfe"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!corrida", fmt.Sprintf("abre uma corrida de cavalo 🏇 e mostra os %d cavalos com quanto cada um paga. Quem tem mais chance de ganhar paga menos", raceHorses)},
				{"!corrida <valor> <cavalo>", fmt.Sprintf("aposta no cavalo pelo número, tipo !corrida 100 3. Se não tiver corrida aberta, a aposta abre uma, e o resto tem %s pra apostar. Se o seu cavalo ganhar, você leva a aposta vezes o que ele paga", formatSeconds(raceWindow))},
			},
			Run:          b.corrida,
			BarredInJail: true,
		},
		{
			Name:     "roubar",
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!roubar @pessoa", fmt.Sprintf("tenta roubar a pessoa marcada: se der certo leva de %d%% a %d%% do saldo dela, se der errado paga indenização 🦹 A porcentagem vale cheia só nas primeiras 1.000 moedas e cai nas faixas de cima, então rico perde uma fatia bem menor (o valor de roubo aparece no !saldo). Dá pra roubar a cada %s, e a profissão de cada um muda a espera e a chance (veja !profissao). Quem tem 🚨 alarme na mochila tem %d%% de chance de ele disparar e barrar o roubo. Quem acabou de ser roubado(a) fica %s sem poder ser roubado(a) de novo, e ladrão com menos de %s que é pego vai direto pra cadeia, porque não tem como pagar a indenização. Cada %s sem cometer crime dá %d%% de proteção, até %d%% (a ficha limpa aparece no !saldo). Quem lucrou %s ou mais numa aposta fica marcado(a) por %s: a chance de roubar sobe %d%% e o prêmio sai sem o desconto das faixas", robLootMin, robLootMax, core.FormatWait(robEvery), percent(alarmChance), core.FormatWait(robShield), Coins(brokeThief), core.FormatWait(cleanEvery), percent(cleanStep), percent(cleanMax), Coins(jackpotMin), core.FormatWait(jackpotWindow), percent(jackpotRobBonus))},
			},
			Run:          b.roubar,
			BarredInJail: true,
		},
		{
			Name:     "assalto",
			Aliases:  []string{"assaltar", "assaltobanco"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!assalto", fmt.Sprintf("monta uma equipe de %d pra assaltar o banco 🏦 Quem manda primeiro abre, e o resto tem %s pra mandar !assalto e entrar. A chance base é %d%%, e a profissão de cada um soma ou tira (assaltante de banco, hacker e piloto de fuga ajudam, influencer e streamer atrapalham, veja !profissao). O equipamento na mochila de qualquer um da equipe também soma: %s. Cada tipo conta uma vez e é gasto no assalto. Se der certo, a equipe divide de %s a %s moedas. Se der errado, cada um tenta escapar e quem não consegue vai preso por %s (a profissão muda o tempo). Dá pra assaltar a cada %s", heistSize, core.FormatWait(heistWindow), percent(heistChance), heistGearList(), core.Thousands(heistLootMin), core.Thousands(heistLootMax), core.FormatWait(baseJail), core.FormatWait(heistCooldown.every))},
			},
			Run:          b.assalto,
			BarredInJail: true,
		},
		{
			Name:     "quadrilha",
			Aliases:  []string{"gangue", "bonde"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!quadrilha @pessoa", fmt.Sprintf("junta uma quadrilha de até %d pra roubar a pessoa marcada 🦹 Quem manda primeiro abre, e o resto tem %s pra mandar !quadrilha e entrar. Com %d sai na hora, e se o tempo acabar com pelo menos %d a quadrilha parte pra cima assim mesmo. A chance começa igual à do !roubar, soma o bônus de roubo da profissão de cada um e mais %d%% por pessoa a mais, e cai com a proteção da vítima. Se der certo, a quadrilha leva de %d%% a %d%% do saldo da vítima, nas mesmas faixas do valor de roubo, e divide igual. Se der errado, cada um paga indenização pra vítima, e quem tem menos de %s vai direto pra cadeia. Quem acabou de ser roubado(a) não pode ser cercado(a) por %s. Usa a mesma espera do !roubar, marca o crime de todo mundo (dá pra !prender e !denunciar) e o 🚨 alarme da vítima dispara menos: a chance dele cai %d%% por pessoa a mais na quadrilha. Quem acabou de ganhar uma bolada na aposta também fica mais fácil de cercar, igual no !roubar", gangSize, core.FormatWait(gangWindow), gangSize, gangMin, percent(gangHelp), gangLootMin, gangLootMax, Coins(brokeThief), core.FormatWait(robShield), percent(alarmGangCut))},
			},
			Run:          b.quadrilha,
			BarredInJail: true,
		},
		{
			Name:     "prender",
			Aliases:  []string{"prisao", "cadeia"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!prender @pessoa", "prende a pessoa marcada, se ela tentou !roubar, !furtar, !sequestrar, !matar, entrou num !assalto ou numa !quadrilha, negociou no !mercadonegro ou desviou o cofre da igreja nas últimas " + core.FormatWait(crimeWindow) + " 🚔 Cada crime só rende uma prisão, e se ela tiver um refém no cativeiro ele é libertado. Enquanto tá na cadeia ela não pode !apostar, !cassino, !blackjack, !crash, !corrida, !roubar, !furtar, !sequestrar, !assalto, !quadrilha nem trocar de !profissao. Só a polícia prende (" + arrestingProfessions() + "), e a cadeia é de " + core.FormatWait(baseJail) + ". A profissão de quem é preso aumenta ou diminui esse tempo (bandido pega mais, advogado pega menos, veja !profissao). Quem tá preso pode tentar !fugir, ou um juiz, advogado ou delegado pode tentar !libertar. Quem emprestou moedas com o !agiota pode prender quem passou do prazo sem pagar, mesmo sem ser da polícia e sem crime nenhum"},
			},
			Run: b.prender,
		},
		{
			Name:     "fugir",
			Aliases:  []string{"fuga", "escapar"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!fugir", fmt.Sprintf("tenta fugir da cadeia ou do cativeiro 🏃 Na cadeia a chance normal é de %d%%, e a profissão muda ela (piloto de fuga, mágico e chaveiro escapam fácil, e a polícia presa fica marcada), e quem é famoso(a) tem menos chance, porque todo mundo reconhece (veja !fama). Se der certo sai na hora, mas fuga é crime e a polícia pode !prender de novo. Se der errado ganha mais %s de cadeia. No cativeiro a chance normal é de %d%%, e aqui a polícia, o segurança, o mágico e o chaveiro se dão bem (veja !profissao). Fugir de sequestro não é crime, mas se der errado fica mais %s amarrado(a). Dá pra tentar a cada %s, e cada nova prisão corta %d%% da espera que faltava", percent(escapeChance), core.FormatWait(escapePenalty), percent(breakoutChance), core.FormatWait(breakoutPenalty), core.FormatWait(escapeCooldown.every), arrestEaseCut)},
				{"!fugir @pessoa", fmt.Sprintf("tenta tirar a pessoa marcada da cadeia ou do cativeiro 🪚 A chance é a mesma do !fugir, mas quem conta é a sua profissão. Tirar alguém da cadeia é crime: se der certo, a polícia pode !prender os dois, e se der errado você tem %d%% de chance de ser pego(a) na hora e ir pra cadeia (com multa). Tirar do cativeiro não é crime e não dá cadeia. Usa a mesma espera do seu !fugir, e quem tá preso ou em cativeiro não consegue ajudar ninguém", percent(rescueCaught))},
			},
			Run: b.fugir,
		},
		{
			Name:     "libertar",
			Aliases:  []string{"resgatar", "alvara"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!libertar @pessoa", fmt.Sprintf("tenta tirar a pessoa marcada da cadeia ou do cativeiro 🔓 Só algumas profissões conseguem, cada uma com a sua chance: %s. Se a pessoa tiver presa e em cativeiro ao mesmo tempo, a cadeia vem primeiro. Se der certo ela sai na hora, e se der errado tem %d%% de chance de você levar uma multa de %d%% a %d%% do saldo (o presidente paga %d%% menos). Dá pra tentar a cada %s", releasingProfessions(), percent(releaseFineChance), releaseFineMin, releaseFineMax, percent(presidentFineDiscount), core.FormatWait(releaseCooldown.every))},
			},
			Run:          b.libertar,
			BarredInJail: true,
		},
		{
			Name:     "denunciar",
			Aliases:  []string{"denuncia", "dedurar", "x9"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!denunciar @pessoa", fmt.Sprintf("liga pro 190 e dedura a pessoa marcada, se ela tentou !roubar, !furtar, !sequestrar, !matar, entrou num !assalto ou numa !quadrilha, negociou no !mercadonegro ou desviou o cofre da igreja nas últimas %s 📞 Qualquer um pode denunciar, mas a polícia só acredita em %d%% das vezes, e a profissão muda isso (detetive, promotor e porteiro são levados a sério, golpista e cartomante quase nunca, veja !profissao), e contra quem é famoso(a) a polícia acredita mais (veja !fama). Se colar, a pessoa pega %s de cadeia (mais ou menos conforme a profissão dela) e solta o refém se tiver um. Bandido não pode denunciar: se tentar, a quadrilha descobre e sorteia o castigo, que pode ser cadeia, morte ou cativeiro com quem ele dedurou. Dá pra denunciar a cada %s", core.FormatWait(crimeWindow), percent(reportChance), core.FormatWait(baseJail), core.FormatWait(reportCooldown.every))},
			},
			Run: b.denunciar,
		},
		{
			Name:     "processar",
			Aliases:  []string{"processo"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!processar @pessoa", fmt.Sprintf("leva pro tribunal quem te denunciou no !denunciar ou tentou te prender no !prender com a sua ficha limpa, nas últimas %s, ou quem te deve no !agiota e passou do prazo ⚖️ O grupo vota na enquete se a pessoa é culpada ou inocente. Se for condenada, paga %d%% do saldo (até %s) de indenização pra você, e se for absolvida você paga %d%% (até %s) de custas. A profissão de cada lado também pesa: advogado e promotor valem votos a mais no júri, quem é do crime tem ficha suja (veja no !profissao). Se for calote no !agiota, a condenação tira do saldo da pessoa o que der pra pagar a dívida, no lugar da indenização. Cada queixa rende um processo só, e dá pra abrir um a cada %s, menos pra quem é do meio jurídico", core.FormatWait(grievanceWindow), trialFinePercent, Coins(trialFineCeiling), trialCostsPercent, Coins(trialCostsCeiling), core.FormatWait(trialEvery))},
			},
			Run: b.processar,
		},
		{
			Name:     "votar",
			Aliases:  []string{"voto", "eleger"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!votar @pessoa", fmt.Sprintf("vota na pessoa marcada (ou em você mesmo, com !votar eu) pra presidente da conversa 🗳️ Se ninguém é presidente, o primeiro voto abre a eleição, e a urna fica aberta %s: cada um vota uma vez e pode mudar o voto até fechar. Precisa de pelo menos %d votos pra valer. Se der empate, tem segundo turno só entre quem empatou: os votos zeram e a urna abre de novo por %s, e se empatar de novo a sorte decide. Quem ganha fica %s no cargo, e só depois disso tem eleição de novo. Quando o cargo vaga, a conversa tem %s pra eleger alguém; se ninguém for eleito nesse prazo, o bot sorteia o presidente entre quem mandou mensagem nas últimas %s, todo mundo com a mesma chance. Veja os poderes no !presidente", core.FormatWait(electionWindow), electionQuorum, core.FormatWait(runoffWindow), core.FormatWait(termLength), core.FormatWait(vacancyWindow), core.FormatWait(drawActivity))},
			},
			Run: b.votar,
		},
		{
			Name:     "presidente",
			Aliases:  []string{"presidencia", "eleicao", "mandato"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!presidente", "mostra quem é o presidente da conversa, quanto falta do mandato, os abusos de poder que sobram e os decretos em vigor 🏛️ Se tiver eleição, impeachment ou golpe rolando, mostra como tá"},
			},
			Run: b.presidente,
		},
		{
			Name:     "decreto",
			Aliases:  []string{"decretar"},
			Category: core.EconomyCategory,
			Help: append([]core.Usage{
				{"!decreto", "lista os decretos que o presidente pode baixar 📜 Só o presidente usa, e os decretos caem quando o mandato acaba"},
			}, decreeHelp()...),
			Run: b.decreto,
		},
		{
			Name:     "impeachment",
			Aliases:  []string{"impeachar", "impichar", "impeach"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!impeachment", fmt.Sprintf("vota pra tirar o presidente do cargo ⚖️ O primeiro voto abre a votação, que fica aberta %s: cada um vota uma vez e pode mudar o voto até fechar. Pra passar, precisa de pelo menos %d votos a favor e mais votos a favor do que contra, e aí o mandato acaba na hora, junto com os decretos. Dá pra pedir um impeachment a cada %s por mandato. Na ditadura (!decreto ditadura), não tem impeachment", core.FormatWait(impeachmentWindow), impeachmentQuorum, core.FormatWait(impeachmentEvery))},
				{"!impeachment nao", "vota contra o impeachment (sem nada, ou com !impeachment sim, é a favor)"},
			},
			Run: b.impeachment,
		},
		{
			Name:     "golpe",
			Aliases:  []string{"golpedeestado", "conspirar"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!golpe", fmt.Sprintf("trama derrubar o presidente na marra 🪖 O primeiro abre a conspiração e quem quiser entra mandando !golpe em até %s. Precisa de pelo menos %d golpistas: começa com %d%% de chance, cada golpista a mais soma %d%%, arma na mochila ajuda e a proteção do presidente atrapalha. Se der certo, quem abriu vira presidente por %s e os decretos caem; se der errado, quem não escapar vai preso. Dá pra tentar um golpe a cada %s por mandato. Contra ditador, bastam %d golpistas e dá pra tentar a cada %s. Com deus no golpe, não tem como dar errado", core.FormatWait(coupWindow), coupMin, percent(coupChance), percent(coupHelp), core.FormatWait(termLength), core.FormatWait(coupEvery), tyrantCoupMin, core.FormatWait(tyrantCoupEvery))},
				{"!golpe defender", fmt.Sprintf("entra do lado do governo (o presidente também pode): cada defensor tira %d%% da chance do golpe, e arma na mochila tira mais. Com deus defendendo, o golpe não passa", percent(coupHelp))},
			},
			Run:          b.golpe,
			BarredInJail: true,
		},
		{
			Name:     "renunciar",
			Aliases:  []string{"renuncia"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!renunciar", "larga a presidência na hora 🏳️ Só o presidente usa. Os poderes do cargo acabam junto, mas os decretos continuam valendo até o próximo presidente tomar posse (eleito no !votar ou sorteado), e os impostos deles passam a sumir em vez de ir pra alguém"},
			},
			Run: b.renunciar,
		},
		{
			Name:     "sequestrar",
			Aliases:  []string{"sequestro", "raptar"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!sequestrar @pessoa", fmt.Sprintf("tenta levar a pessoa marcada pro cativeiro 🚐 Se der certo ela fica %s sem poder !apostar, !cassino, !blackjack, !crash, !corrida, !roubar, !furtar, !sequestrar nem trocar de !profissao, ou até ela conseguir !fugir ou a polícia !prender quem sequestrou. Se der errado você paga indenização e ainda pode ir preso na hora (bem mais fácil se a vítima for da polícia). Dá pra tentar a cada %s, um refém por vez, e a profissão de cada um muda a espera e a chance (veja !profissao). Se a vítima te deve no !agiota e passou do prazo, é cobrança: não é crime e você não vai preso na hora", core.FormatWait(hostageHold), core.FormatWait(kidnapCooldown.every))},
			},
			Run:          b.sequestrar,
			BarredInJail: true,
		},
		{
			Name:     "venderrefem",
			Aliases:  []string{"vendersequestrado"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!venderrefem", fmt.Sprintf("leva o seu refém pro mercado clandestino 🤝 O negócio fecha em %s, e até lá a polícia ainda pode !prender você e libertar o refém, ou ele pode !fugir. O preço depende da profissão do refém (quem ganha mais vale mais), da fama dele (famoso vale mais, veja !fama) e da sua profissão (contrabandista, cambista e golpista vendem mais caro). Depois de vendido, o refém fica no máximo mais %s trabalhando pra quem comprou, sem poder apostar, a não ser que consiga !fugir, e nem a polícia resgata mais", core.FormatWait(hostageSaleWait), core.FormatWait(soldHostageHold))},
			},
			Run:          b.venderrefem,
			BarredInJail: true,
		},
		{
			Name:     "matar",
			Aliases:  []string{"assassinar", "apagar"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!matar @pessoa", fmt.Sprintf("tenta apagar a pessoa marcada 🔪 Só algumas profissões matam, cada uma com a sua chance: %s, e a proteção da vítima tira um pouco. Com a Foice da Comadre Morte (relíquia), qualquer um mata, o colete não segura, a vítima fica %dx mais tempo morta e o enterro (de %d%% a %d%% do saldo dela) vai pro seu bolso. Se der certo ela fica %s morta, sem poder usar nenhum comando de economia (a não ser que alguém consiga !reviver), o enterro leva um pedaço do saldo dela, e se tiver um refém no cativeiro ele fica livre. Se tiver !recompensa na cabeça dela, ela é toda sua. Se der errado você tem %d%% de chance de ir preso na hora por %s (bem mais se a vítima for da polícia, e a profissão muda o tempo). Matar ou tentar é crime: a polícia pode !prender e qualquer um pode !denunciar. Dá pra tentar a cada %s. Quem emprestou moedas com o !agiota pode matar quem passou do prazo sem pagar, mesmo sem ser dessas profissões, com pelo menos %d%% de chance, sem ser crime e sem ir preso", killingProfessions(), reaperHoldFactor, reapLossMinPercent, reapLossMaxPercent, core.FormatWait(deathHold), percent(killCaughtChance), core.FormatWait(baseJail), core.FormatWait(killCooldown.every), percent(debtKillChance))},
			},
			Run:          b.matar,
			BarredInJail: true,
		},
		{
			Name:     "recompensa",
			Aliases:  []string{"procurado", "procurados", "premio"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!recompensa @pessoa <valor>", "bota moedas na cabeça da pessoa 🎯 O valor sai do teu saldo na hora e soma com o que os outros já botaram. Quem conseguir !matar ela leva tudo (também aceita metade, 10% ou tudo)"},
				{"!recompensa @pessoa", "mostra quanto vale a cabeça da pessoa"},
				{"!recompensa retirar @pessoa", "tira a recompensa que você botou na pessoa e devolve as moedas"},
				{"!recompensa", fmt.Sprintf("mostra as %d cabeças mais valiosas da conversa e o procurado do dia: todo dia às %dh o bot bota de %d a %d moedas na cabeça de alguém que mexeu na economia nos últimos dias", bountyBoardSize, wantedHour, wantedMin, wantedMax)},
			},
			Run: b.recompensa,
		},
		{
			Name:     "reviver",
			Aliases:  []string{"ressuscitar", "ressucitar"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!reviver @pessoa", fmt.Sprintf("tenta trazer de volta quem morreu no !matar 🧟 Só algumas profissões conseguem, cada uma com a sua chance: %s, e o presidente também revive. Com o Livro de São Cipriano (relíquia), qualquer um revive sem falhar, mas a pessoa volta presa num pacto e passa %d%% de tudo que ganhar pra quem fez o ritual por %s. Se der certo a pessoa volta na hora e já pode usar a economia, e se der errado ela continua no caixão até o prazo acabar. Dá pra tentar a cada %s", revivingProfessions(), soulPactPercent, core.FormatWait(soulPactHold), core.FormatWait(reviveCooldown.every))},
			},
			Run: b.reviver,
		},
		{
			Name:     "abencoar",
			Aliases:  []string{"bencao", "bencaos", "benzer"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!abencoar", "mostra todas as bênçãos, quem pode dar cada uma e as que tão em você agora 🙏"},
				{"!abencoar @pessoa <bênção>", fmt.Sprintf("dá uma bênção pra pessoa por um tempo: mais ganho, mais proteção, mais sorte, mais chance de fugir... A Purificação tira na hora as maldições dela. Só as profissões de fé e feitiçaria abençoam, cada uma com as suas (%s), e ninguém se abençoa sozinho, a não ser gastando um item de bênção no !usar (veja no !item encantos). Quem tem o %s (relíquia) lança qualquer bênção ou maldição. Dá pra lançar uma bênção ou maldição a cada %s", mysticProfessions(), mirrorLabel(), core.FormatWait(charmCooldown.every))},
			},
			Run: b.abencoar,
		},
		{
			Name:     "amaldicoar",
			Aliases:  []string{"maldicoes", "praguejar", "rogarpraga"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!amaldicoar", "mostra todas as maldições, quem pode jogar cada uma e as que tão em você agora 🕯️"},
				{"!amaldicoar @pessoa <maldição>", fmt.Sprintf("joga uma maldição na pessoa por um tempo: ganha menos, fica mais fácil de roubar, perde sorte no crime, fica mais tempo preso... Cada profissão de fé e feitiçaria tem as suas e a sua chance de pegar, e quando falha a praga tem %d%% de chance de voltar pra quem jogou. Quem não tem a profissão pode jogar com item, no !usar <item> @pessoa (veja no !item encantos), com %d%% de chance de pegar. Quem tem o %s joga qualquer maldição, e a que jogam nele volta pra quem jogou. Quem tá amaldiçoado se livra com uma !abencoar @pessoa purificacao ou com um !usar sal grosso, menos da Casa assombrada, que só sai com !imovel exorcizar", percent(hexBackfire), percent(itemHexChance), mirrorLabel())},
			},
			Run: b.amaldicoar,
		},
		{
			Name:     "milagre",
			Aliases:  []string{"chover"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!milagre <valor>", fmt.Sprintf("só pra deus ⚡ faz chover moeda nova pra todo mundo do grupo: cada um ganha o valor (até %s), e nada sai do seu saldo", Coins(miracleMax))},
				{"!milagre <item> [quantidade]", fmt.Sprintf("só pra deus ✨ faz chover um item do !item pra todo mundo do grupo: cada um ganha a quantidade (até %d), criada do nada", creationMax)},
				{"!milagre <valor ou item> @pessoa", "só pra deus 🎯 o mesmo milagre, mas só pra quem você marcar"},
			},
			Run: b.milagre,
		},
		{
			Name:     "castigo",
			Aliases:  []string{"castigar", "ira"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!castigo <valor>", "só pra deus 🔥 faz sumir moedas de todo mundo do grupo: cada um perde o valor (ou o que tiver, se for menos), e as moedas não vão pra ninguém. Aceita número, porcentagem (!castigo 10%), metade ou tudo"},
				{"!castigo @pessoa <valor>", "mesma coisa, mas só com quem você marcar"},
			},
			Run: b.castigo,
		},
		{
			Name:     "criar",
			Aliases:  []string{"conjurar", "materializar"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!criar <item> [quantidade]", fmt.Sprintf("só pra deus ✨ cria qualquer item do !item do nada, direto na sua mochila, até %d de uma vez. Nada sai do seu saldo", creationMax)},
			},
			Run: b.criar,
		},
		{
			Name:     "transferir",
			Aliases:  []string{"pagar", "pix"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!transferir @pessoa <valor>", "manda moedas suas pra pessoa marcada 💸"},
			},
			Run: b.transferir,
		},
		{
			Name:     "conjunta",
			Aliases:  []string{"contaconjunta"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!conjunta", "mostra o saldo da conta conjunta com cada pessoa com quem você casou pelo pedido do modo consentimento 🏦"},
				{"!conjunta depositar <valor>", "guarda moedas suas na conta conjunta (aceita tudo, metade e 10%)"},
				{"!conjunta sacar <valor>", "tira moedas da conta conjunta pra sua carteira. Qualquer um dos dois pode sacar, e no divórcio o que sobrar é dividido meio a meio"},
				{"!conjunta depositar <valor> @pessoa", "escolhe com qual cônjuge, se você for casado(a) com mais de uma pessoa"},
			},
			Run: b.conjunta,
		},
		{
			Name:     "poupar",
			Aliases:  []string{"poupanca", "guardar"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!poupar <valor>", fmt.Sprintf("trava as moedas na poupança por %d horas e elas voltam com %d%% de juros, até %s por vez 🏦", int(savingsLock.Hours()), savingsRate, core.Thousands(savingsCap))},
				{"!poupar", "mostra sua poupança e resgata quando já liberou"},
			},
			Run: b.poupar,
		},
		{
			Name:     "agiota",
			Aliases:  []string{"emprestar", "pediremprestado", "divida", "dividas"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!agiota @pessoa <valor> [dias]", fmt.Sprintf("pede moedas emprestadas pra pessoa marcada, que vira seu agiota 🦈 Ela tem %s pra responder com !aceitar ou !recusar. O prazo padrão é de %d dias (dá pra pedir de 1 a %d). Juros de %d%% ao dia, compostos, já dentro do prazo, e depois dele sobem pra %d%% ao dia (quem é agiota de profissão cobra mais). Passou do prazo, o agiota pode !matar, !prender, !sequestrar ou !processar quem deve, sem ser crime e sem precisar da profissão", core.FormatWait(offerWindow), debtTermDays, debtMaxDays, debtRate, debtLateRate)},
				{"!agiota pagar @pessoa [valor|tudo]", "paga parte ou toda a dívida com a pessoa marcada (se você só deve pra uma, nem precisa marcar)"},
				{"!agiota", "mostra quanto você deve e quanto te devem, com os juros e o prazo de cada dívida"},
			},
			Run: b.agiota,
		},
		{
			Name:     "emprestimo",
			Aliases:  []string{"emprestimos", "financiamento"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!emprestimo pegar <valor> [dias]", fmt.Sprintf("pega moedas emprestadas do banco 🏦, até um limite que cresce com o que sua profissão paga no !trabalhar (de %s a %s). O prazo padrão é de %d dias (dá pra pedir de 1 a %d). Juros de %d%% ao dia, compostos, e depois do prazo sobem pra %d%% ao dia e o banco penhora seu saldo a cada %s até quitar", Coins(loanLimitFloor), Coins(loanLimitCap), loanTermDays, loanMaxDays, loanRate, loanLateRate, core.FormatWait(loanSeizeEvery))},
				{"!emprestimo pagar [valor|tudo]", "paga parte ou todo o empréstimo do banco, e as moedas pagas somem"},
				{"!emprestimo", "mostra quanto você deve pro banco, ou quanto ele te empresta se você não deve nada"},
			},
			Run: b.emprestimo,
		},
		{
			Name:     "acoes",
			Aliases:  []string{"acao", "bovespa"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!acoes", fmt.Sprintf("mostra a bolsa do grupo, com os preços que mexem a cada %d minutos, e a sua carteira com lucro ou prejuízo 📈", int(stockStepEvery.Minutes()))},
				{"!acoes comprar <empresa> <quantidade>", fmt.Sprintf("compra ações (ou 500 moedas, metade, tudo), com %d%% de corretagem", stockFeePercent)},
				{"!acoes vender <empresa> <quantidade|tudo>", "vende suas ações pelo preço de agora"},
			},
			Run: b.acoes,
		},
		{
			Name:     "loteria",
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!loteria <quantidade>", fmt.Sprintf("compra bilhetes da loteria do grupo a %s cada, até %d por sorteio. Todo dia às %dh sai um ganhador, com chance proporcional aos bilhetes, que leva o pote inteiro com %d%% de bônus da casa 🎟️", Coins(lotteryTicketPrice), lotteryMaxTickets, lotteryHour, lotteryHouseBonus)},
				{"!loteria", "mostra o pote, quantos bilhetes você tem, sua chance de ganhar e quanto falta pro sorteio"},
			},
			Run: b.loteria,
		},
		{
			Name:     "bolao",
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!bolao criar <pergunta> | <opção 1> | <opção 2>", fmt.Sprintf("abre um bolão na conversa, com até %d opções separadas por | 🎯 Tipo !bolao criar Quem ganha o clássico? | Flamengo | Empate | Vasco", betPoolMaxOptions)},
				{"!bolao <número> <valor>", "aposta na opção (também aceita metade, 10% e tudo). Quem acertar divide o pote todo, proporcional ao que apostou"},
				{"!bolao", "mostra o bolão aberto, quanto tem em cada opção, quanto cada uma tá pagando e quanto você apostou"},
				{"!bolao fechar", "encerra as apostas (só quem abriu, admin do grupo ou dono do bot)"},
				{"!bolao resultado <número>", "diz qual opção ganhou e paga quem acertou. Se ninguém acertou, todo mundo recebe de volta"},
				{"!bolao cancelar", "cancela o bolão e devolve todas as apostas"},
			},
			Run: b.bolao,
		},
		{
			Name:     "bicho",
			Aliases:  []string{"bixo", "jogodobicho", "fezinha"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!bicho <valor> <bicho>", fmt.Sprintf("faz uma fezinha no jogo do bicho 🦁, tipo !bicho 50 vaca (também aceita metade, 10%% e tudo). Tem extração todo dia às %s, e o resultado é o mesmo pra todo mundo. Se der o seu bicho na cabeça, paga %dx, e o prêmio cai no saldo na hora da extração", animalSchedule(), animalGroupBet.odds)},
				{"!bicho <valor> <número>", fmt.Sprintf("joga num número em vez do bicho: %s, tipo !bicho 5 1234. Cada tipo tem uma aposta máxima, e cada pessoa faz até %d apostas por extração. O bicho é clandestino: o decreto do cassino não pega nele", animalNumberOdds(), animalBetsPerDraw)},
				{"!bicho", "mostra a próxima extração, as suas apostas esperando resultado e os últimos resultados"},
				{"!bicho tabela", "mostra os 25 bichos com as dezenas de cada um, quanto cada tipo de aposta paga e a aposta máxima"},
			},
			Run:          b.bicho,
			BarredInJail: true,
		},
		{
			Name:     "ricos",
			Aliases:  []string{"forbes"},
			Category: core.EconomyCategory,
			Help:     []core.Usage{{"!ricos", "mostra os mais ricos do grupo 🏆 e quem caiu no ranking desde o último !ricos, dizendo se empobreceu apostando, sendo roubado ou pagando imposto"}},
			Run:      b.ricos,
		},
		{
			Name:     "imposto",
			Aliases:  []string{"impostos", "leao", "iptu"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!imposto", fmt.Sprintf("mostra as faixas do imposto sobre fortuna e quanto você paga 🦁 Todo dia às %dh o Leão cobra de quem tem mais de %s, mais o IPTU de quem tem imóvel, e as moedas somem. Um contador no !empregados corta parte disso", taxHour, Coins(taxBrackets[0].above))},
				{"!imposto @pessoa", "mostra quanto a pessoa marcada paga"},
			},
			Run: b.imposto,
		},
		{
			Name:     "fianca",
			Aliases:  []string{"pagarfianca"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!fianca", fmt.Sprintf("paga pra sair da cadeia na hora 🔓 O preço é uma parte do seu saldo, que sobe com o tempo que falta (de %d%% a %d%%, mínimo de %s), e o dinheiro some", exitBasePercent, exitMaxPercent, Coins(exitFloor))},
				{"!fianca @pessoa", "paga a fiança de outra pessoa, com o preço calculado em cima do saldo dela"},
			},
			Run: b.fianca,
		},
		{
			Name:     "hospital",
			Aliases:  []string{"medico", "upa"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!hospital", fmt.Sprintf("paga o hospital particular 🏥 Morto(a), volta à vida na hora: custa %d%% do saldo (mínimo de %s) e dá pra usar mesmo morto(a). Vivo(a) e machucado(a), sai com a saúde cheia pagando só a parte da saúde que falta (com a saúde em 50, metade do preço)", hospitalPercent, Coins(hospitalFloor))},
				{"!hospital @pessoa", "paga o hospital pra trazer outra pessoa de volta, ou pra cuidar dela se tiver machucada"},
			},
			Run:       b.hospital,
			WhileDead: true,
		},
		{
			Name:     "saude",
			Aliases:  []string{"vida", "hp"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!saude", fmt.Sprintf("mostra sua saúde ❤️ (de 0 a %d), quanto falta pra sarar e a chance de se machucar no !trabalhar, em cada extração e em cada crime (quem é vítima de crime também pode se machucar, mas não morre disso). Recupera sozinho 1 ponto a cada %s (mais rápido com um enfermeiro no !empregados), ou na hora com o !hospital ou com !usar curativo. Equipamento de proteção (tem na !loja) diminui a chance de se machucar. Se zerar, morre por %s e o enterro leva de %d%% a %d%% do saldo", MaxHealth, core.FormatWait(healthRegenEvery), core.FormatWait(deathHold), deathLossMinPercent, deathLossMaxPercent)},
				{"!saude @pessoa", "mostra a saúde da pessoa marcada"},
			},
			Run: b.saude,
		},
		{
			Name:     "resgate",
			Aliases:  []string{"pagarresgate"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!resgate", fmt.Sprintf("paga o resgate e sai do cativeiro na hora 💼 Mesmo preço da fiança (de %d%% a %d%% do saldo, mínimo de %s), e o dinheiro some com os bandidos", exitBasePercent, exitMaxPercent, Coins(exitFloor))},
				{"!resgate @pessoa", "paga o resgate de outra pessoa"},
			},
			Run: b.resgate,
		},
		{
			Name:     "imovel",
			Aliases:  []string{"imoveis", "casa", "mansao"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!imovel", fmt.Sprintf("mostra os %d imóveis, da barraca de camping à base em Marte, e onde você mora 🏘️ Cada um dá proteção, bônus no !daily e no !trabalhar e cobra IPTU todo dia", len(estates))},
				{"!imovel comprar", "se muda pro próximo imóvel da lista. Paga o preço cheio e o antigo vai pro lixo"},
				{"!imovel @pessoa", "mostra onde a pessoa marcada mora"},
				{"!imovel guardar <item> [quantidade]", "guarda um item da mochila em casa, onde ninguém furta e a polícia não apreende. Cada imóvel tem um tanto de espaço, e item guardado não funciona até voltar pra mochila"},
				{"!imovel tirar <item> [quantidade]", "tira um item de casa e põe de volta na mochila"},
				{"!imovel guardados", "mostra o que você tem guardado em casa (ou de quem for marcado) e quanto espaço sobra"},
				{"!imovel depositar <valor>", "esconde moedas no cofre de casa, onde ninguém rouba e a multa e o enterro não pegam. Cada imóvel tem um cofre de um tamanho, e moeda no cofre não serve pra nada até voltar pro saldo (dá pra mandar metade, 10% ou tudo)"},
				{"!imovel sacar <valor>", "tira moedas do cofre de casa e põe de volta no saldo"},
				{"!imovel cofre", "mostra quanto você tem no cofre de casa (ou quem for marcado) e quanto ainda cabe"},
				{"!imovel exorcizar", fmt.Sprintf("espanta o espírito da Casa assombrada (a maldição do !amaldicoar) pagando %d dias de IPTU pra quem mandou ele 👻 Enquanto a casa tá assombrada, a proteção e os bônus do imóvel viram o contrário e o IPTU vem %dx", hauntRansomDays, hauntTaxFactor)},
			},
			Run: b.imovel,
		},
		{
			Name:     "empregados",
			Aliases:  []string{"empregado", "funcionarios", "funcionario"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!empregados", "mostra seus empregados, o que eles já acharam e quem dá pra contratar 👷 Eles fazem o " + extractionCommands() + " sozinhos, mas nunca acham relíquia, mítico nem lendário. O caçador de bússola só procura a Bússola do relicário e quase nunca acha. O apostador e o corretor trazem moedas, o contador corta o imposto, o enfermeiro faz a saúde voltar mais rápido e o segurança particular dificulta te !roubar, !furtar, !sequestrar e !matar"},
				{"!empregados contratar <nome>", "contrata um empregado. Precisa de imóvel, e quanto melhor o imóvel, mais vagas"},
				{"!empregados coletar", "pega o que os empregados trouxeram e paga o salário deles. Sem saldo pro salário, eles trabalham fiado e a tensão sobe. No limite da tensão eles entram em greve, até você pagar o atrasado. O contador, o enfermeiro e o segurança não têm nada pra coletar"},
				{"!empregados demitir <nome>", "manda um empregado embora, e o que ele achou e você não coletou vai junto"},
			},
			Run: b.empregados,
		},
		{
			Name:     "contratar",
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!contratar <nome>", "o mesmo que !empregados contratar <nome>"},
			},
			Run: b.contratar,
		},
		{
			Name:     "cofre",
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!cofre", "o mesmo que !imovel cofre"},
			},
			Run: b.showVault,
		},
		{
			Name:     "titulo",
			Aliases:  []string{"titulos", "nobreza"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!titulo", "mostra os títulos de nobreza do grupo, quem tem cada um, quanto custa e a vantagem de cada um 👑 Cada título tem um dono só, aparece antes do nome no !saldo e no !ricos e dá uma vantagem leve enquanto tá em uso: um bônus pequeno, uma bênção ou maldição pra lançar ou um abuso de poder"},
				{"!titulo comprar <nome>", fmt.Sprintf("compra o título, ou toma de quem tem pagando %d%% do que a pessoa pagou (ela recebe %d%% de volta)", titleStealPercent, titleRefundPercent)},
				{"!titulo usar <nome>", "troca o título que aparece no seu nome (!titulo usar nenhum tira)"},
			},
			Run: b.titulo,
		},
		{
			Name:     "implante",
			Aliases:  []string{"implantes", "cirurgia", "ciborgue"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!implante", fmt.Sprintf("mostra a clínica de implantes 🦾 Tem %d implantes, um por parte do corpo (cérebro, olhos e braço), que ajudam a ganhar mais no !trabalhar, nas extrações, no !vender, no !acoes, no cassino ou no crime. Eles chegam no !mercadonegro, um por dia, e vão pra mochila de quem comprar", len(implants))},
				{"!implante instalar <nome>", fmt.Sprintf("instala um implante que tá na sua mochila (o !usar faz o mesmo). O corpo pode rejeitar: aí o implante vai pro lixo e você perde até %d de saúde, e se já tiver machucado(a) pode morrer", implantRejectHarm)},
				{"!implante manutencao", fmt.Sprintf("paga a manutenção dos seus implantes por %s. Sem manutenção, o implante desliga e para de dar a vantagem", upkeepSpan(implantUpkeepMax))},
				{"!implante tirar <nome>", fmt.Sprintf("tira um implante (pelo nome ou pela parte do corpo) e devolve pra mochila. Cada saída do corpo desgasta, e na %dª vez ele quebra", implantWearMax)},
				{"!implante ver [@pessoa]", fmt.Sprintf("mostra os implantes da pessoa, a manutenção e o desgaste de cada um. Quem mata no !matar tem %d%% de chance de arrancar um implante da vítima e levar pra mochila, e a polícia arranca os clandestinos de quem vai preso", percent(implantRipChance))},
			},
			Run: b.implante,
		},
		{
			Name:     "igreja",
			Aliases:  []string{"igrejas", "religiao", "seita"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!igreja", "mostra a sua igreja, ou as igrejas do grupo se você ainda não é fiel de nenhuma ⛪ Cada igreja tem uma doutrina, e todo fiel ganha o efeito dela e paga o dízimo da igreja, uma parte do que entra no !trabalhar e no !daily, pro cofre dela"},
				{"!igreja lista", "mostra as igrejas do grupo, com os fiéis, o cofre e o líder de cada uma"},
				{"!igreja doutrinas", "mostra as doutrinas, com o efeito e o dízimo padrão de cada uma"},
				{"!igreja fundar <doutrina> <nome>", fmt.Sprintf("funda uma igreja e vira o líder dela. Só pra quem é de %s, e custa %d%% do saldo (no mínimo %s)", areas[areaFe].name, churchFoundPercent, Coins(churchFoundFloor))},
				{"!igreja entrar <nome>", fmt.Sprintf("vira fiel de uma igreja. Entrar numa igreja (ou fundar uma) tem espera de %s", core.FormatWait(conversionCooldown.every))},
				{"!igreja sair", "larga a igreja. Se o líder sai, quem tá há mais tempo na igreja assume, e se não sobrar ninguém a igreja fecha e o cofre some"},
				{"!igreja ofertar <valor>", "bota moedas suas no cofre da sua igreja (aceita metade, 10% e tudo)"},
				{"!igreja reformar", fmt.Sprintf("o líder usa o cofre pra reformar o templo (%s), e cada reforma deixa a doutrina mais forte, até %d%% no último", templeLadder(), temples[len(temples)-1].power)},
				{"!igreja dizimo <porcentagem>", fmt.Sprintf("o líder escolhe quanto os fiéis dão de dízimo pro cofre, de 0 a %d%%. A igreja começa com o dízimo padrão da doutrina", churchTitheMax)},
				{"!igreja milagre <valor>", "o líder reparte moedas do cofre em partes iguais entre os outros fiéis"},
				{"!igreja desviar <valor>", fmt.Sprintf("o líder põe moedas do cofre no próprio bolso, uma vez a cada %s. É crime: dá pra ser preso depois, e tem de %d%% a %d%% de chance (quanto mais tira, maior) de ser pego na hora, ir pra cadeia, deixar o dinheiro no cofre e perder a liderança pra quem tá há mais tempo na igreja", core.FormatWait(embezzleCooldown.every), percent(embezzleCaughtBase), percent(embezzleCaughtBase+embezzleCaughtScale))},
			},
			Run: b.igreja,
		},
		{
			Name:     "luxo",
			Aliases:  []string{"lojadeluxo", "grife"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!luxo", fmt.Sprintf("mostra a loja de luxo de hoje 💎 São %d itens finos por dia, a %dx o valor de tabela. Cada item fino dá uma vantagem diferente: os que duram (relógio, carro, jatinho...) valem só de estar na mochila, e as comidas e bebidas dão uma bênção no !usar", luxuryStockSize, luxuryMarkup)},
				{"!luxo comprar <item> [quantidade]", "compra um item da loja de luxo (o !comprar também funciona)"},
			},
			Run: b.luxo,
		},
		{
			Name:     "mercadonegro",
			Aliases:  []string{"mn", "submundo", "biqueira"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!mercadonegro", fmt.Sprintf("mostra as armas, drogas e o contrabando do dia 🕶️ São %d itens por dia, e o preço varia de %d%% a %d%% do valor de tabela. Às vezes chega muamba também: no máximo uma ferramenta de extração lendária por dia (%d%% de chance de vir uma) e %s, cada um com %d%% de chance por dia, mas só vem uma de cada e quem comprar primeiro leva. Toda negociação conta como crime pro !prender e pro !denunciar, e tem %d%% de chance de dar batida: cadeia, multa e a mercadoria perdida. Quem é preso ainda perde tudo o que é ilegal da mochila", blackMarketStockSize, blackMarketMinPercent, blackMarketMaxPercent, percent(hotToolChance), hotGearLabels(), percent(hotGearChance), percent(raidChance))},
				{"!mercadonegro comprar <item> [quantidade]", "compra uma arma, droga ou muamba do estoque do dia (o !comprar também funciona). Arma funciona só de estar na mochila, droga é com !usar"},
				{"!mercadonegro vender <item> [quantidade]", fmt.Sprintf("vende arma ou droga pro receptador, a %d%% do preço do dia", blackMarketBuyback)},
			},
			Run:          b.mercadonegro,
			BarredInJail: true,
		},
		{
			Name:     "doar",
			Aliases:  []string{"obra", "obras", "vaquinha"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!doar <valor>", "doa moedas pras obras do grupo 🏗️ Banco da praça, chafariz, estátua, estádio... e quem mais doar pra cada obra é homenageado nela. O dinheiro não volta"},
				{"!doar", "mostra a obra em construção, os maiores benfeitores e as obras prontas"},
				{"!doar @pessoa <item>", "dá um item de presente, igual ao !dar"},
			},
			Run: b.doar,
		},
		{
			Name:     "zoar",
			Aliases:  []string{"zoeira", "trollar"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!zoar", "mostra as zoeiras pagas e quanto cada uma custa pra você 🃏 O preço sobe com o seu saldo e as moedas somem"},
				{"!zoar palhaco @pessoa", "o bot reage com 🤡 em toda mensagem da pessoa por um tempo"},
				{"!zoar apelido @pessoa <apelido>", "dá um apelido e tranca por 24h, ninguém troca no !apelido"},
				{"!zoar maldicao @pessoa", "o !trabalhar da pessoa paga metade por um tempo"},
				{"!zoar outdoor <texto>", "o bot publica um outdoor com o seu texto"},
				{"!zoar limpar", "faz um descarrego e tira toda zoeira que tá em você"},
			},
			Run: b.zoar,
		},
		{
			Name:     "mochila",
			Aliases:  []string{"inventario", "inv", "bolsa"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!mochila", "mostra os itens que você tem e quanto eles valem 🎒"},
				{"!mochila @pessoa", "mostra a mochila da pessoa marcada"},
			},
			Run: b.mochila,
		},
		{
			Name:     "favoritar",
			Aliases:  []string{"fav", "favorito"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!favoritar <item>", "marca um item da mochila como favorito ⭐ e ele não sai em nenhuma venda (!vender, !vender @pessoa, !mercadonegro vender nem nas vendas em lote). Só as cópias repetidas saem no !vender duplicados, que sempre deixa 1"},
				{"!favoritar #<número> [#número...]", "favorita vários pelos números da !mochila, tipo !favoritar #12 #40"},
				{"!favoritar tudo", "favorita tudo o que tá na mochila agora"},
			},
			Run: b.favoritar,
		},
		{
			Name:     "desfavoritar",
			Aliases:  []string{"desfav"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!desfavoritar <item>", "tira o item dos favoritos, e ele volta a poder ser vendido 💔 Também aceita os números da !mochila (!desfavoritar #12 #40)"},
				{"!desfavoritar tudo", "tira todos os itens dos favoritos"},
			},
			Run: b.desfavoritar,
		},
		{
			Name:     "item",
			Aliases:  []string{"itens", "catalogo"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!item", fmt.Sprintf("mostra os %d itens que existem, separados por tipo, com a raridade na bolinha 📖", len(catalog))},
				{"!item <tipo>", "mostra só os itens de um tipo, separados por raridade, tipo !item pesca, !item armas ou !item comida"},
				{"!item <nome>", "mostra os detalhes de um item: raridade, valor, onde conseguir e pra que serve"},
			},
			Run: b.item,
		},
		{
			Name:     "loja",
			Aliases:  []string{"shop"},
			Category: core.EconomyCategory,
			Help:     []core.Usage{{"!loja", fmt.Sprintf("mostra o que dá pra comprar e quanto custa 🛒 Embaixo vêm as ofertas do dia (⏳ Só hoje): %d itens que mudam todo dia, e cada um só aparece em alguns dias. Um deles é sempre uma ferramenta épica de extração, em rodízio, então cada uma volta em até 3 dias. E a Tele Sena só sai aos domingos", rotatingShopSize)}},
			Run:      b.loja,
		},
		{
			Name:     "petshop",
			Aliases:  []string{"pets", "pet", "bichos"},
			Category: core.EconomyCategory,
			Help:     []core.Usage{{"!petshop", "mostra os bichinhos fofos que dá pra levar pra casa 🐾 De vez em quando eles trazem um item no seu !daily"}},
			Run:      b.petshop,
		},
		{
			Name:     "comprar",
			Aliases:  []string{"buy"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!comprar <item>", "compra um item da loja, tipo !comprar vara de pesca"},
				{"!comprar <item> <quantidade>", "compra vários de uma vez, tipo !comprar todynho 5"},
			},
			Run: b.comprar,
		},
		{
			Name:     "vender",
			Aliases:  []string{"sell"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!vender <item> [quantidade]", fmt.Sprintf("vende pro mercado na hora, mas ele só paga %d%% do valor 💸 (item mítico sai por %d%%; também aceita tudo)", marketSalePercent, mythicSalePercent)},
				{"!vender #<número> [#número...]", "vende tudo o que você tem de cada item pelos números da !mochila, sempre com #, tipo !vender #12 #40 #87"},
				{"!vender tipo <tipo>", "vende tudo o que você tem de um tipo da !mochila, tipo !vender tipo pesca (os ilegais ficam de fora)"},
				{"!vender inuteis", "vende tudo da mochila que não tem efeito nenhum (peixe, minério, lixo, inseto...), lendários e míticos inclusos"},
				{"!vender bugigangas", "vende os inúteis e mais o que dá pra !usar mas não faz nada de verdade (comida, presente, Nokia, Tamagotchi...). Os bichinhos ficam"},
				{"!vender duplicados", "vende tudo que tá repetido e deixa 1 de cada item, sempre o de melhor qualidade. É a única venda que mexe nos favoritos ⭐ (só nas cópias)"},
				{"!vender @pessoa <item> <preço>", fmt.Sprintf("oferece o item pra pessoa marcada pelo preço que você quiser, e ela tem %s pra responder com !aceitar ou !recusar 🤝", core.FormatWait(offerWindow))},
				{"!vender @pessoa <item> <quantidade> <preço>", "vende vários de uma vez, com o preço sendo o total"},
			},
			Run: b.vender,
		},
		{
			Name:     "aceitar",
			Aliases:  []string{"aceito"},
			Category: core.EconomyCategory,
			Help:     []core.Usage{{"!aceitar", "fecha a compra que alguém te ofereceu com o !vender @você, ou empresta as moedas que alguém te pediu com o !agiota @você ou diz sim pro pedido de casamento ou namoro ✅"}},
			Run:      b.aceitar,
		},
		{
			Name:     "recusar",
			Aliases:  []string{"recuso"},
			Category: core.EconomyCategory,
			Help:     []core.Usage{{"!recusar", "recusa a proposta, o pedido de empréstimo ou o pedido de casamento ou namoro que te fizeram, ou cancela os que você fez 🙅"}},
			Run:      b.recusar,
		},
		{
			Name:     "leilao",
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!leilao", "mostra o leilão do dia 🔨 Todo dia o bot leiloa um item épico ou lendário diferente (e, bem de vez em quando, uma relíquia com lance mínimo salgado), e quem tiver o maior lance à meia-noite leva. As moedas do lance vencedor somem"},
			},
			Run: b.leilao,
		},
		{
			Name:     "lance",
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!lance <valor>", fmt.Sprintf("dá um lance no leilão do dia (!leilao <valor> também funciona). Tem que cobrir o anterior em pelo menos %d%% (mínimo %d moedas), as moedas saem na hora e voltam se alguém cobrir, e lance nos últimos %s prorroga o leilão", auctionStepPercent, auctionMinStep, core.FormatWait(auctionExtension))},
			},
			Run: b.lance,
		},
		{
			Name:     "dar",
			Aliases:  []string{"presentear"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!dar @pessoa <item> [quantidade]", "dá um item seu de presente pra pessoa marcada 🎁"},
			},
			Run: b.dar,
		},
		{
			Name:     "usar",
			Aliases:  []string{"abrir", "comer", "beber", "tomar"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!usar <item>", "usa um item: come, bebe, abre caixa, raspa raspadinha... cada um faz uma coisa (veja no !item <nome>) ✨ Item de bênção (trevo, terço, olho grego...) abençoa quem usa, item de cura (curativo, kit de primeiros socorros) recupera saúde, e item de maldição se joga em alguém, tipo !usar gato preto @pessoa. Dá pra mandar !comer, !beber ou !abrir também"},
			},
			Run: b.usar,
		},
		{
			Name:     "aprimorar",
			Aliases:  []string{"aprimoramento", "ferreiro", "upgrade"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!aprimorar", "mostra as ferramentas de extração que dá pra aprimorar e quanto custa cada uma 🔧"},
				{"!aprimorar <ferramenta>", "paga o ferreiro e troca a ferramenta da mochila pela versão aprimorada, que acha coisa melhor e se paga mais rápido. Dá pra aprimorar as raras, épicas e lendárias do !pescar, do !minerar e do !catar, uma vez só cada, e o preço é metade do valor da ferramenta. Se a aprimorada quebrar, o aprimoramento vai junto"},
			},
			Run: b.aprimorar,
		},
		{
			Name:         "primordial",
			Aliases:      []string{"primordiais"},
			Category:     core.EconomyCategory,
			BarredInJail: true,
			Help: []core.Usage{
				{"!primordial", "mostra os seus itens primordiais ⚫ (a raridade mais alta que existe) e o que cada um faz. Sem nenhum, mostra como conseguir o primeiro"},
				{"!primordial <item>", fmt.Sprintf("desperta a habilidade do item primordial. Hoje só existe o Meteorito do Bendegó: !primordial bendego abre uma cratera no grupo por %s, uma vez por dia, onde quem mandar !minerar acha mítico mesmo sem relíquia e com mais sorte, mas paga %d%% do valor de cada achado de pedágio pro dono", core.FormatWait(craterOpen), craterToll)},
			},
			Run: b.primordial,
		},
		{
			Name:     "pescar",
			Aliases:  []string{"pesca", "pescaria"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!pescar", fmt.Sprintf("joga a linha e pesca um item, de lambari a escama de sereia 🎣 Precisa de vara de pesca (tem na !loja) e dá pra pescar a cada %s. Em %s das vezes se machuca (a boia salva-vidas protege)", core.FormatWait(fishing.cooldown.every), oddsText(fishing.hazard.chance))},
			},
			Run: b.gather(fishing),
		},
		{
			Name:     "minerar",
			Aliases:  []string{"mina", "mineracao"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!minerar", fmt.Sprintf("desce na mina atrás de minério e pedra preciosa, e com sorte um meteorito ⛏️ Precisa de picareta (tem na !loja) e dá pra minerar a cada %s. Em %s das vezes se machuca (o capacete protege)", core.FormatWait(mining.cooldown.every), oddsText(mining.hazard.chance))},
			},
			Run: b.gather(mining),
		},
		{
			Name:     "vasculhar",
			Aliases:  []string{"lixao", "lixo", "garimpar"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!vasculhar", fmt.Sprintf("revira o lixão atrás de tralha, e às vezes de coisa boa 🗑️ Não precisa de nada e dá pra vasculhar a cada %s. Em %s das vezes se machuca (a luva de borracha protege)", core.FormatWait(scavenging.cooldown.every), oddsText(scavenging.hazard.chance))},
			},
			Run: b.gather(scavenging),
		},
		{
			Name:     "catar",
			Aliases:  []string{"cacar", "cacada", "insetos", "borboletas"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!catar", fmt.Sprintf("sai com a rede atrás de inseto, de pernilongo a fada no pote 🦋 Precisa de rede de insetos (tem na !loja) e dá pra catar a cada %s. Em %s das vezes se machuca (a roupa de apicultor protege)", core.FormatWait(bugging.cooldown.every), oddsText(bugging.hazard.chance))},
			},
			Run: b.gather(bugging),
		},
		{
			Name:         "plantar",
			Aliases:      []string{"semear", "plantio"},
			Category:     core.EconomyCategory,
			BarredInJail: true,
			Help: []core.Usage{
				{"!plantar <semente>", fmt.Sprintf("bota uma semente ou muda num canteiro da !horta 🌱 Ela cresce sozinha, cada uma no tempo dela, e vira verdura, fruta, flor ou madeira pra vender ou dar pra alguém. Tem semente na !loja, e as mais caras só aparecem de quebra no %s", extractionCommands())},
				{"!plantar <semente> 3", fmt.Sprintf("planta várias de uma vez, uma por canteiro (ou !plantar milho tudo). Todo mundo começa com %d canteiros e dá pra comprar mais no !horta expandir. Dá pra escrever o nome do que vai nascer, tipo !plantar tomate", gardenBeds)},
			},
			Run: b.plantar,
		},
		{
			Name:         "colher",
			Aliases:      []string{"colheita"},
			Category:     core.EconomyCategory,
			BarredInJail: true,
			Help: []core.Usage{
				{"!colher", "colhe tudo o que já ficou pronto na !horta 🧺 Cada canteiro dá umas unidades, cada uma com a qualidade dela, e o canteiro fica livre pra !plantar de novo. O que não tá pronto continua crescendo"},
			},
			Run: b.colher,
		},
		{
			Name:     "horta",
			Aliases:  []string{"plantacao", "roca", "canteiros"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!horta", "mostra os seus canteiros, o que tá crescendo, quanto falta pra cada um e as sementes que você tem na mochila 🌾"},
				{"!horta @pessoa", "mostra a horta da pessoa marcada"},
				{"!horta expandir", fmt.Sprintf("compra mais um canteiro pra sua horta. O primeiro sai por %s, cada um custa mais que o anterior e dá pra comprar até %d. As moedas somem", Coins(bedPrices[0]), len(bedPrices))},
			},
			Run: b.horta,
		},
		{
			Name:     "furtar",
			Aliases:  []string{"furto"},
			Category: core.EconomyCategory,
			Help: []core.Usage{
				{"!furtar @pessoa", fmt.Sprintf("tenta furtar um item aleatório da mochila da pessoa marcada 🥷 Quanto mais raro, mais difícil, e se der errado você paga indenização. Uma vez a cada %s, e a profissão de cada um muda a espera e a chance", core.FormatWait(theftCooldown.every))},
				{"!furtar @pessoa <item>", "tenta furtar um item específico, com menos chance de dar certo. Quem tem 🔒 cadeado na mochila tá protegido, e relíquia nunca pode ser furtada"},
			},
			Run:          b.furtar,
			BarredInJail: true,
		},
	}
}

func (b *bot) economyOwnerCommands() []core.Command {
	return []core.Command{
		{
			Name:     "economia",
			Category: core.OwnerCategory,
			Help: []core.Usage{
				{"!economia resetar intervalos", "zera a espera de todos os comandos da economia (daily, trabalhar, roubar, pescar, prender, fugir...) pra todo mundo desta conversa, solta quem tá preso, de refém ou morto, enche a saúde e limpa a ficha de crime. O dinheiro, a mochila e o resto ficam como tão"},
				{"!economia resetar intervalos @pessoa", "zera os intervalos só das pessoas marcadas"},
				{"!economia vincular <nome do grupo>", "este grupo passa a usar a economia do outro: carteira, mochila, presidente, ranking, loteria, cadeia, tudo junto, e os avisos automáticos saem em todos os grupos vinculados. O que este grupo tinha antes fica guardado"},
				{"!economia desvincular", "este grupo sai da economia compartilhada e volta pra economia que tinha antes de vincular"},
				{"!economia vinculos", "mostra quais grupos usam a mesma economia desta conversa"},
			},
			Run: b.economia,
		},
	}
}
