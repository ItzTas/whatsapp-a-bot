package economy

import "time"

var legendEvents = eventBook{
	name:   "🪔 Lendas",
	places: []string{legendPlace},
	scenarios: []scenario{
		{
			id:     "lenda-lampada",
			title:  "O gênio de verdade",
			rarity: legendaryEvent,
			item:   "lampada-magica",
			intros: []string{
				"{@} esfregou a Lâmpada mágica esperando o gênio do Pix de sempre. Só que dessa vez a lâmpada ficou quente, a fumaça encheu o grupo e saiu o gênio DE VERDADE, de turbante e tudo: \"faz mil anos que ninguém me esfrega direito. Um desejo, e capricha\"",
			},
			choices: []choice{
				{label: "💰 Desejar uma fortuna", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"Choveu dinheiro em cima de {@}, e dessa vez não era de brinquedo"}, effects: []effect{earn(900, 1500)}},
					{weight: 25, texts: []string{"O gênio entendeu \"Fortuna\" como o nome de uma cigana e mandou {@} pra ela. Pelo menos ela leu a mão de graça e acertou tudo"}, effects: []effect{earn(200, 400), enchant("sorte")}},
					{weight: 15, texts: []string{"O dinheiro veio, mas era o do cofre da prefeitura, e o prefeito veio buscar com dois seguranças"}, effects: []effect{earn(300, 600), lose(5, 10)}},
				}},
				{label: "🍀 Desejar sorte pra vida", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"O gênio estalou os dedos e {@} sentiu o universo inteiro mudar de lado"}, effects: []effect{enchant("sorte"), enchant("prosperidade"), enchant("fartura")}},
				}},
				{label: "🕊️ Libertar o gênio", risk: lowRisk, uses: "lampada-magica", endings: []ending{
					{weight: 100, texts: []string{"O gênio chorou de emoção, quebrou a corrente e, antes de sumir no céu, deixou o que guardava fazia mil anos: \"pra quem me soltou\""}, effects: []effect{gain(1, "relicario"), earn(400, 800), enchant("protecao")}},
				}},
				{label: "🪔 Desejar mais desejos", risk: highRisk, endings: []ending{
					{weight: 25, texts: []string{"O gênio coçou o bigode e admitiu que não tem regra nenhuma contra isso. Lá vem mais um"}, next: "lenda-lampada-bis"},
					{weight: 55, texts: []string{"\"Todo mundo tenta essa\", disse o gênio, e jogou uma praga antes de voltar pra lâmpada"}, effects: []effect{enchant("miseria")}},
					{weight: 20, texts: []string{"O gênio prendeu {@} dentro da lâmpada por uma hora, pra aprender a não ser espertinho"}, effects: []effect{delay(time.Hour)}},
				}},
			},
			idle: ending{texts: []string{"A lâmpada esfriou na mão de {@} e o gênio voltou a dormir. Dizem que agora só acorda daqui a mil anos"}},
		},
		{
			id:     "lenda-lampada-bis",
			title:  "O desejo extra",
			rarity: legendaryEvent,
			stage:  true,
			intros: []string{
				"O gênio bufou, olhou o relógio e liberou mais um desejo pra {@}: \"esse é o último MESMO, e sem gracinha\"",
			},
			choices: []choice{
				{label: "💰 Mais dinheiro", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"PLIM! Mais uma chuva de dinheiro, agora com trovão e tudo"}, effects: []effect{earn(500, 900)}},
				}},
				{label: "❤️ Saúde de ferro", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} sentiu o corpo renovado. Até a dor nas costas de 2014 sumiu"}, effects: []effect{heal(100), enchant("protecao")}},
				}},
				{label: "🏴‍☠️ Um baú do tesouro", risk: midRisk, endings: []ending{
					{weight: 70, texts: []string{"Um baú caiu do céu bem do lado de {@}, cheio de cadeado e de promessa"}, effects: []effect{gain(1, "bau-do-tesouro")}},
					{weight: 30, texts: []string{"O baú veio, sim, cheio de areia e com um caranguejo bravo dentro"}, effects: []effect{gain(1, "caranguejo"), injure(5, 10)}},
				}},
			},
			idle: ending{texts: []string{"O gênio cansou de esperar {@} decidir e voltou pra lâmpada resmungando"}},
		},
		{
			id:     "lenda-dragao",
			title:  "O ovo vai chocar",
			rarity: legendaryEvent,
			item:   "ovo-de-dragao",
			intros: []string{
				"O Ovo de dragão começou a tremer na mão de {@}, esquentou até ficar vermelho e apareceu a primeira rachadura. Lá de dentro vem um barulhinho de quem quer sair, e o cheiro de churrasco já tomou conta do grupo",
			},
			choices: []choice{
				{label: "🐣 Esperar chocar e criar", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"Saiu um dragãozinho do tamanho de um gato, que em uma semana começou a trazer coisa brilhante pra casa. Ninguém pergunta de onde"}, effects: []effect{gain(1, "diamante", "idolo-de-ouro", "escaravelho-de-ouro"), earn(300, 600)}},
					{weight: 30, texts: []string{"O dragãozinho espirrou fogo no sofá, na cortina e na sobrancelha de {@}"}, effects: []effect{injure(10, 30), lose(5, 10)}},
					{weight: 15, texts: []string{"O dragãozinho bateu as asas no primeiro dia, deu uma volta no bairro e nunca mais voltou. Deixou um presente embrulhado na porta"}, effects: []effect{gain(1, "caixa-misteriosa-dourada")}},
				}},
				{label: "💼 Vender antes de chocar", risk: lowRisk, uses: "ovo-de-dragao", endings: []ending{
					{weight: 100, texts: []string{"Um colecionador de terno preto apareceu em cinco minutos, pagou em dinheiro vivo e levou o ovo numa caixa de isopor"}, effects: []effect{earn(1000, 1500)}},
				}},
				{label: "🌋 Devolver pra mãe no vulcão", risk: lowRisk, uses: "ovo-de-dragao", endings: []ending{
					{weight: 100, texts: []string{"A mãe dragão saiu da lava, pegou o ovo com o maior cuidado do mundo e deixou cair um presente nos pés de {@}. Desde então, nada de ruim chega perto"}, effects: []effect{gain(1, "diamante-rosa"), enchant("protecao"), enchant("maoboa")}},
				}},
				{label: "🍳 Fazer uma omelete", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"Foi a omelete mais gostosa da história, e {@} nunca se sentiu tão forte e tão rápido"}, effects: []effect{heal(100), enchant("pressa")}},
					{weight: 60, texts: []string{"A mãe dragão sentiu o cheiro de longe, apareceu na janela e não gostou nada do cardápio"}, effects: []effect{injure(30, 60)}},
				}},
			},
			idle: ending{texts: []string{"A rachadura fechou sozinha e o ovo esfriou na mão de {@}. Pelo jeito o dragãozinho desistiu de nascer"}},
		},
		{
			id:     "lenda-pedra-filosofal",
			title:  "A pedra acordou",
			rarity: legendaryEvent,
			item:   "pedra-filosofal",
			intros: []string{
				"{@} encostou a Pedra filosofal no feijão de novo, mas dessa vez ela começou a pulsar vermelha e a falar em latim. Um velho alquimista apareceu no meio da fumaça: \"finalmente alguém digno. Diz o que a pedra vai transformar\"",
			},
			choices: []choice{
				{label: "🪙 Transformar tudo em ouro", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"Toda a tralha de {@} virou ouro maciço, e o ferro-velho comprou na hora"}, effects: []effect{earn(800, 1300)}},
					{weight: 25, texts: []string{"Virou ouro, sim. Ouro de tolo, que brilha bonito e não vale nada"}, effects: []effect{gain(3, "pedra")}},
					{weight: 15, texts: []string{"A transmutação saiu do controle e derreteu a ponta dos dedos de {@}"}, effects: []effect{injure(15, 35)}},
				}},
				{label: "🧪 Fazer o elixir da vida", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} bebeu o elixir e sentiu dez anos saírem das costas"}, effects: []effect{heal(100), enchant("protecao"), enchant("pressa")}},
				}},
				{label: "📜 Vender pro alquimista", risk: lowRisk, uses: "pedra-filosofal", endings: []ending{
					{weight: 100, texts: []string{"O alquimista levou a pedra embrulhada num pano de veludo e pagou com moedas de ouro de 1600"}, effects: []effect{earn(1000, 1500), gain(1, "bau-do-tesouro")}},
				}},
				{label: "🔮 Transformar o azar em sorte", risk: midRisk, endings: []ending{
					{weight: 70, texts: []string{"A pedra sugou tudo de ruim que tava grudado em {@} e devolveu em sorte"}, effects: []effect{enchant("purificacao"), enchant("sorte")}},
					{weight: 30, texts: []string{"A pedra sugou o azar, pensou bem e cuspiu de volta em dobro"}, effects: []effect{enchant("azar")}},
				}},
			},
			idle: ending{texts: []string{"O alquimista cansou de esperar, a pedra parou de pulsar e voltou a ser só um enfeite caro na mochila de {@}"}},
		},
		{
			id:     "lenda-saci",
			title:  "O trato do saci",
			rarity: legendaryEvent,
			item:   "saci-na-garrafa",
			intros: []string{
				"{@} abriu a garrafa do saci pra zoar o grupo, mas dessa vez ele não saiu correndo. Sentou na boca da garrafa, acendeu o cachimbo e propôs: \"me solta de vez que eu te faço um favor. Mas pensa bem, que saci não trabalha de graça\"",
			},
			choices: []choice{
				{label: "🔓 Soltar o saci de vez", risk: lowRisk, uses: "saci-na-garrafa", endings: []ending{
					{weight: 100, texts: []string{"O saci deu um assobio, girou num redemoinho e, antes de sumir, amarrou a sorte no cadarço de {@}"}, effects: []effect{enchant("sorte"), enchant("pressa"), enchant("abrecaminho")}},
				}},
				{label: "🤝 Botar o saci pra trabalhar", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"O saci fez o trampo de {@} na velocidade de um redemoinho e ainda deixou o troco certinho"}, effects: []effect{refresh(), earn(400, 800)}},
					{weight: 40, texts: []string{"O saci trabalhou, sim: escondeu as coisas de {@}, trançou a crina do cavalo do vizinho e sumiu com o troco"}, effects: []effect{lose(5, 15)}},
				}},
				{label: "🍶 Oferecer fumo e cachaça", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O saci ficou alegrinho e contou onde tinha um tesouro enterrado no mato"}, effects: []effect{gain(1, "bau-do-tesouro")}},
					{weight: 30, texts: []string{"O saci tomou a cachaça, fumou o fumo e dormiu na garrafa. Nem obrigado disse"}},
				}},
				{label: "🙅 Tampar a garrafa de novo", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} tampou a garrafa. Lá de dentro veio um assobio bravo, e todos os cadarços do grupo se amarraram sozinhos"}},
				}},
			},
			idle: ending{texts: []string{"O saci cansou de esperar, pulou de volta pra dentro da garrafa e tampou por dentro. Esse trato não volta mais"}},
		},
		{
			id:     "lenda-fada",
			title:  "A fada cansou do pote",
			rarity: legendaryEvent,
			item:   "fada",
			intros: []string{
				"{@} abriu o pote pra fada jogar purpurina no grupo, como sempre. Só que ela não saiu: cruzou os braços, bateu o pé e disse com a voz fininha que cansou de ser enfeite. \"Ou você me solta com respeito, ou a gente negocia\"",
			},
			choices: []choice{
				{label: "🧚 Soltar a fada", risk: lowRisk, uses: "fada", endings: []ending{
					{weight: 100, texts: []string{"A fada deu um beijo na testa de {@} e saiu voando. Por onde ela passou nasceu flor e caiu dinheiro"}, effects: []effect{earn(300, 600), enchant("fartura"), enchant("prosperidade")}},
				}},
				{label: "✨ Pedir pó de pirlimpimpim", risk: midRisk, endings: []ending{
					{weight: 65, texts: []string{"Com uma pitada do pó, {@} voou até o telhado do banco e achou uma mala esquecida lá em cima"}, effects: []effect{gain(1, "mala-de-dinheiro")}},
					{weight: 35, texts: []string{"{@} exagerou no pó e passou uma hora flutuando no teto sem conseguir descer"}, effects: []effect{delay(time.Hour)}},
				}},
				{label: "📸 Vender foto dela pro jornal", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"A foto viralizou e o jornal pagou bem pela exclusiva"}, effects: []effect{earn(300, 600), fans(10, 20)}},
					{weight: 30, texts: []string{"Ninguém acreditou. Disseram que era um vaga-lume com filtro"}},
				}},
				{label: "🔒 Lacrar o pote com fita", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"A fada ficou de mal, mas continua lá dentro, brilhando de raiva"}},
					{weight: 50, texts: []string{"Fada brava é pior que sogra: ela furou a fita e jogou uma praga antes de voltar pro canto dela"}, effects: []effect{enchant("olhogordo")}},
				}},
			},
			idle: ending{texts: []string{"A fada desistiu de negociar, virou de costas e dormiu. Ela nunca mais falou com {@}"}},
		},
		{
			id:     "lenda-disco-voador",
			title:  "Os donos do disco voltaram",
			rarity: legendaryEvent,
			item:   "disco-voador",
			intros: []string{
				"{@} ligou o Disco voador pro rolê de sempre, mas acendeu no painel um botão que nunca tinha acendido. Cinco minutos depois, uma nave muito maior parou em cima da casa: os ETs vieram buscar o disco que perderam em 1977",
			},
			choices: []choice{
				{label: "🛸 Devolver o disco", risk: lowRisk, uses: "disco-voador", endings: []ending{
					{weight: 100, texts: []string{"Os ETs agradeceram com um aperto de mão gelado e deixaram um presente de outro planeta"}, effects: []effect{gain(1, "meteorito"), earn(600, 1000)}},
				}},
				{label: "🤑 Pedir resgate pelo disco", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"Os ETs pagaram em barras de um metal que ninguém conhece, mas o ferro-velho aceitou sem perguntar"}, effects: []effect{earn(700, 1200)}},
					{weight: 45, texts: []string{"ET não negocia: abduziram {@} pra uma conversa séria e devolveram horas depois, de pijama"}, effects: []effect{injure(5, 15), delay(time.Hour)}},
				}},
				{label: "☕ Chamar os ETs pra um café", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"Os ETs adoraram o café passado no coador e ensinaram um truque pra {@}: fazer tudo na metade do tempo"}, effects: []effect{refresh(), enchant("pressa")}},
					{weight: 30, texts: []string{"Os ETs provaram o café, fizeram careta e foram embora sem dizer nada"}},
				}},
				{label: "🏃 Fugir com o disco", risk: highRisk, endings: []ending{
					{weight: 35, texts: []string{"{@} despistou a nave-mãe atrás de uma nuvem e ainda achou um asteroide cheio de coisa brilhante"}, effects: []effect{gain(1, "diamante", "diamante-rosa"), earn(300, 600)}},
					{weight: 65, texts: []string{"A nave-mãe puxou o disco com raio trator e {@} foi junto. Voltou com uma cicatriz esquisita no braço"}, effects: []effect{injure(30, 60)}},
				}},
			},
			idle: ending{texts: []string{"Os ETs cansaram de esperar resposta e foram embora. O botão do painel apagou e nunca mais acendeu"}},
		},
		{
			id:     "lenda-tridente",
			title:  "Netuno quer o tridente",
			rarity: legendaryEvent,
			item:   "tridente",
			intros: []string{
				"{@} levantou o Tridente de Netuno pra agitar a piscina do prédio, mas dessa vez o céu fechou, o mar subiu até a calçada e de dentro de uma onda saiu o próprio Netuno, de barba molhada e cara de poucos amigos: \"isso aí é meu\"",
			},
			choices: []choice{
				{label: "🔱 Devolver o tridente", risk: lowRisk, uses: "tridente", endings: []ending{
					{weight: 100, texts: []string{"Netuno pegou o tridente, sorriu pela primeira vez em séculos e prometeu que o mar ia tratar {@} bem pra sempre"}, effects: []effect{earn(400, 800), gain(1, "escama-de-sereia"), enchant("maoboa")}},
				}},
				{label: "⚔️ Desafiar Netuno", risk: highRisk, endings: []ending{
					{weight: 30, texts: []string{"{@} acertou um golpe de sorte e Netuno, impressionado, pagou o respeito em pérola e ouro do fundo do mar"}, effects: []effect{earn(900, 1500), gain(1, "bau-do-tesouro")}},
					{weight: 70, texts: []string{"Netuno mandou uma onda que levou {@} até a outra praia, junto com metade da carteira"}, effects: []effect{injure(30, 60), lose(5, 15)}},
				}},
				{label: "🎣 Pedir uma pescaria lendária", risk: midRisk, endings: []ending{
					{weight: 65, texts: []string{"Netuno bufou, mas mandou os peixes fazerem fila pra {@}"}, effects: []effect{gain(1, "dente-de-megalodonte"), enchant("maoboa")}},
					{weight: 35, texts: []string{"Netuno mandou um peixe, sim: um baiacu, que espetou a mão de {@}"}, effects: []effect{injure(10, 25)}},
				}},
				{label: "🏃 Correr com o tridente", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} correu tanto que Netuno desistiu na terceira esquina, sem fôlego fora d'água"}},
					{weight: 50, texts: []string{"Netuno não corre, mas manda chuva: choveu só em cima de {@} a semana inteira"}, effects: []effect{enchant("panema")}},
				}},
			},
			idle: ending{texts: []string{"Netuno cansou de esperar, voltou pro mar resmungando e a piscina do prédio baixou. Ele não sobe de novo tão cedo"}},
		},
		{
			id:     "lenda-jules-rimet",
			title:  "A CBF bateu na porta",
			rarity: legendaryEvent,
			item:   "taca-jules-rimet",
			intros: []string{
				"{@} levantou a Taça Jules Rimet na janela de novo, só que dessa vez alguém filmou. Em uma hora a CBF, a Polícia Federal e um repórter do Fantástico tavam batendo na porta, todos atrás da mesma taça",
			},
			choices: []choice{
				{label: "🏆 Devolver em rede nacional", risk: lowRisk, uses: "taca-jules-rimet", endings: []ending{
					{weight: 100, texts: []string{"{@} devolveu a taça ao vivo no Fantástico, chorou, ganhou a recompensa e virou herói nacional"}, effects: []effect{earn(800, 1300), fans(20, 40)}},
				}},
				{label: "🎭 Vender uma réplica pro gringo", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"Um sheik pagou em dólar e levou uma taça de lata pintada de dourado. A original continua com {@}"}, effects: []effect{earn(700, 1200), stain()}},
					{weight: 40, texts: []string{"O gringo mordeu a taça, sentiu gosto de tinta e chamou a polícia. {@} teve que devolver o dinheiro e mais um pouco"}, effects: []effect{lose(10, 20), stain()}},
				}},
				{label: "🙈 Esconder no forro do teto", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"A Federal revirou a casa e não achou nada. {@} ficou com a taça e com fama de lenda no bairro"}, effects: []effect{enchant("protecao")}},
					{weight: 30, texts: []string{"O forro não aguentou o peso e a taça caiu na cabeça de {@} no meio da madrugada"}, effects: []effect{injure(10, 25)}},
				}},
				{label: "🎤 Dar entrevista exclusiva", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} contou no Fantástico que achou a taça no lixão. O Brasil parou pra ouvir"}, effects: []effect{earn(200, 400), fans(15, 30)}},
				}},
			},
			idle: ending{texts: []string{"Ninguém atendeu a porta, e a CBF, a Federal e o Fantástico foram embora achando que era boato. A taça ficou com {@}, mas a história morreu ali"}},
		},
	},
}
