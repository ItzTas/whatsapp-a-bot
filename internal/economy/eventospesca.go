package economy

import "time"

var fishingEvents = eventBook{
	name:   "🎣 Pesca",
	places: []string{fishing.command},
	scenarios: []scenario{
		{
			id:    "pesca-monstro-do-rio",
			title: "O monstro do rio",
			intros: []string{
				"{@} sentiu um puxão tão forte que a vara envergou igual arco. Tem alguma coisa ENORME do outro lado da linha",
				"A linha de {@} começou a cantar e o molinete girou sozinho. O que fisgou ali não é lambari, não",
				"A boia de {@} sumiu de uma vez e a água borbulhou igual panela de pressão. Veio bicho grande",
			},
			choices: []choice{
				{label: "💪 Puxar com tudo", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{
						"Depois de meia hora de cabo de guerra, {@} arrastou o bicho pra margem e ainda tirou foto pro grupo da família",
						"Foi braço, foi perna, foi reza, mas {@} venceu o monstro do rio",
					}, effects: []effect{gain(1, "pirarucu", "tambaqui", "dourado")}},
					{weight: 30, texts: []string{"A linha arrebentou e {@} caiu sentado na lama"}, effects: []effect{injure(5, 15)}},
					{weight: 20, texts: []string{"O peixe puxou tão forte que levou a vara junto pro fundo do rio"}, effects: []effect{breakTool()}},
				}},
				{label: "✂️ Cortar a linha", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{
						"{@} cortou a linha e ficou só imaginando o tamanho do que escapou. A história vai crescer a cada churrasco",
						"{@} cortou a linha. Na roda de pescador, o peixe já tem dois metros e meio",
					}},
				}},
				{label: "🙋 Chamar o vizinho de barranco", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"Os dois puxaram juntos e trouxeram o peixe. Na divisão, {@} ficou com um pedaço e uns trocados"}, effects: []effect{gain(1, "tambaqui", "pacu"), earn(10, 30)}},
					{weight: 30, texts: []string{"O vizinho puxou sozinho, levou o peixe embora e ainda contou pra todo mundo que foi ele que fisgou"}},
				}},
				{label: "🛟 Pular na água com a boia", risk: midRisk, needs: holds("boia"), endings: []ending{
					{weight: 70, texts: []string{"Agarrado na boia, {@} abraçou o peixe e voltou nadando com ele"}, effects: []effect{gain(1, "pirarucu"), earn(30, 60)}},
					{weight: 30, texts: []string{"O peixe arrastou {@} rio abaixo, mas a boia segurou firme"}, effects: []effect{injure(5, 10), delay(15 * time.Minute)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, o peixe cansou de esperar e arrebentou a linha"}},
		},
		{
			id:    "pesca-jacare-na-margem",
			title: "Jacaré de olho",
			intros: []string{
				"{@} tava concentrado na boia quando viu dois olhinhos parados na água, bem pertinho do balde. É um jacaré, e ele tá com fome",
				"Um jacaré de uns três metros subiu no barranco e ficou encarando {@} e o balde da pescaria",
			},
			choices: []choice{
				{label: "🐟 Jogar a pescaria pra ele", risk: lowRisk, endings: []ending{
					{weight: 75, texts: []string{"{@} virou o balde pro jacaré, que engoliu tudo e foi embora de barriga cheia"}, effects: []effect{forfeit("lambari", 3), forfeit("tilapia", 2), forfeit("sardinha", 3)}},
					{weight: 25, texts: []string{"O jacaré comeu, gostou e resolveu morar ali. {@} teve que procurar outro ponto"}, effects: []effect{delay(15 * time.Minute)}},
				}},
				{label: "🏃 Sair de fininho", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} foi andando de costas bem devagar, e o jacaré nem ligou"}},
					{weight: 30, texts: []string{"Saindo de fininho, {@} pisou no barro e escorregou barranco abaixo"}, effects: []effect{injure(5, 12)}},
				}},
				{label: "🥊 Encarar o bicho", risk: highRisk, endings: []ending{
					{weight: 30, favor: func(p Profession) float64 { return p.guard }, texts: []string{
						"{@} pulou no lombo do jacaré igual documentário do Globo Repórter e voltou com ele amarrado",
					}, effects: []effect{gain(1, "jacare")}},
					{weight: 45, texts: []string{"O jacaré deu uma bocada na perna de {@} e só soltou porque não gostou do gosto"}, effects: []effect{injure(20, 40)}},
					{weight: 17, texts: []string{"O jacaré mordeu a vara e levou embora. Pra {@}, sobrou o susto e um arranhão"}, effects: []effect{breakTool(), injure(5, 10)}},
					{weight: 8, texts: []string{"O jacaré arrastou {@} pro fundo do rio e não teve Globo Repórter que salvasse"}, effects: []effect{perish()}},
				}},
				{label: "🔫 Dar um tiro pra cima", risk: midRisk, needs: armedUp, endings: []ending{
					{weight: 70, texts: []string{"O estampido espantou o jacaré e metade dos peixes do rio. A vizinhança anotou a placa do carro de {@}"}, effects: []effect{findLoot(), stain()}},
					{weight: 30, texts: []string{"O tiro espantou o jacaré, mas um vizinho ligou pro 190 na hora"}, effects: []effect{nabbed()}},
				}},
			},
			idle: ending{texts: []string{"{@} travou de medo. O jacaré achou a cena sem graça e voltou pra água"}},
		},
		{
			id:    "pesca-dono-do-pesqueiro",
			title: "O dono do pesqueiro",
			intros: []string{
				"Chegou um senhor de chapéu de palha e balde na mão e disse pra {@}: 'esse ponto aí é meu faz trinta anos'",
				"{@} mal tinha jogado a linha quando o Seu Zé da beira do rio apareceu bufando: 'aí é o MEU lugar'",
			},
			choices: []choice{
				{label: "🤝 Ceder o lugar pro Seu Zé", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} deu o lugar e o Seu Zé, todo agradecido, ensinou o segredo da isca dele e disse que um dia ia retribuir"}, effects: []effect{gain(2, "isca"), plant("pesca-seu-ze-volta", 6*time.Hour)}},
					{weight: 40, texts: []string{"{@} cedeu o lugar e o Seu Zé fisgou um tambaqui na primeira jogada, só pra humilhar"}},
				}},
				{label: "🗣️ Bater boca", risk: midRisk, endings: []ending{
					{weight: 45, texts: []string{"{@} falou mais alto e o Seu Zé foi embora resmungando. O ponto era bom mesmo"}, effects: []effect{findLoot()}},
					{weight: 35, texts: []string{"O Seu Zé chamou os netos, e os netos chamaram {@} pra conversar atrás da árvore"}, effects: []effect{injure(5, 15)}},
					{weight: 20, texts: []string{"Na discussão, o Seu Zé rogou praga: 'aqui você não pesca nem resfriado'"}, effects: []effect{enchant("panema")}},
				}},
				{label: "💸 Pagar um aluguel do ponto", risk: lowRisk, price: cost{percent: 2, floor: 40}, endings: []ending{
					{weight: 70, texts: []string{"Negócio fechado. O Seu Zé sabia mesmo onde o peixe se esconde"}, effects: []effect{findLoot(), findLoot()}},
					{weight: 30, texts: []string{"{@} pagou o aluguel e o peixe tirou folga justo hoje"}},
				}},
				{label: "🥃 Oferecer uma cachaça", risk: lowRisk, uses: "cachaca", endings: []ending{
					{weight: 100, texts: []string{"{@} e o Seu Zé dividiram a pinga, contaram mentira de pescador a tarde toda e viraram amigos de infância"}, effects: []effect{enchant("maoboa"), plant("pesca-seu-ze-volta", 4*time.Hour)}},
				}},
			},
		},
		{
			id:     "pesca-seu-ze-volta",
			title:  "O presente do Seu Zé",
			rarity: rareEvent,
			stage:  true,
			intros: []string{
				"O Seu Zé reapareceu na beira do rio com um embrulho debaixo do braço: 'trouxe uma coisa pra quem me respeitou'",
				"{@} tava pescando quando ouviu um 'ô, meu amigo!'. Era o Seu Zé, todo sorridente, com um presente na mão",
			},
			choices: []choice{
				{label: "🎁 Aceitar o presente", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"Era um baú velho que o pai do Seu Zé tirou do rio nos anos setenta. Ninguém da família teve coragem de abrir"}, effects: []effect{gain(1, "bau-do-tesouro")}},
					{weight: 40, texts: []string{"Era um pote de minhoca gorda e uma garrafa com um mapa rabiscado que o Seu Zé nunca entendeu"}, effects: []effect{gain(5, "isca"), gain(1, "garrafa-com-mensagem")}},
				}},
				{label: "🙅 Recusar com educação", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"O Seu Zé ficou emocionado com a humildade e benzeu a vara de {@} ali mesmo, com reza de pescador"}, effects: []effect{enchant("maoboa"), enchant("sorte")}},
				}},
				{label: "🎣 Chamar ele pra pescar junto", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"Os dois pescaram a tarde inteira. O Seu Zé mostrou o pesqueiro secreto e {@} voltou com o balde cheio"}, effects: []effect{findLoot(), findLoot(), refresh()}},
				}},
			},
		},
		{
			id:    "pesca-garrafa-boiando",
			title: "A garrafa boiando",
			intros: []string{
				"Uma garrafa de vidro veio boiando e enroscou bem no anzol de {@}. Tem um papel enrolado lá dentro",
				"{@} puxou a linha e, em vez de peixe, veio uma garrafa fechada com rolha e um papel amarelado dentro",
			},
			choices: []choice{
				{label: "📜 Abrir e ler", risk: midRisk, endings: []ending{
					{weight: 40, texts: []string{"Era um mapa! Um X vermelho marcado numa ilhota no meio do rio"}, next: "pesca-ilhota-do-x"},
					{weight: 35, texts: []string{"Era uma carta de amor de 1987 pra uma tal de Rosângela. {@} chorou um pouquinho e guardou de lembrança"}, effects: []effect{gain(1, "garrafa-com-mensagem")}},
					{weight: 25, texts: []string{"Era uma corrente: 'repasse pra dez pessoas ou terá sete anos de azar'. {@} não repassou"}, effects: []effect{enchant("azar")}},
				}},
				{label: "🌊 Jogar de volta no rio", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} devolveu a garrafa pro rio. Vai que é de alguém, né"}},
					{weight: 30, texts: []string{"O rio pareceu agradecer: na mesma hora o peixe começou a morder"}, effects: []effect{findLoot()}},
				}},
				{label: "💰 Vender pro turista", risk: lowRisk, endings: []ending{
					{weight: 80, texts: []string{"Um turista achou a coisa mais linda do mundo e pagou sem nem abrir"}, effects: []effect{earn(20, 60)}},
					{weight: 20, texts: []string{"O turista abriu na frente de {@}, achou um mapa e saiu correndo. Ficou o gostinho de 'e se'"}},
				}},
			},
		},
		{
			id:     "pesca-ilhota-do-x",
			title:  "O X na ilhota",
			rarity: rareEvent,
			stage:  true,
			intros: []string{
				"O X do mapa fica numa ilhota no meio do rio, uns duzentos metros de água barrenta. Dá pra ver uma pedra marcada lá",
				"Pelo mapa, o tesouro tá enterrado na ilhota bem ali na frente. O rio tá cheio e a correnteza tá forte",
			},
			choices: []choice{
				{label: "🏊 Atravessar nadando", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"{@} chegou na ilhota sem fôlego, cavou embaixo da pedra e achou um baú enferrujado"}, effects: []effect{gain(1, "bau-do-tesouro")}},
					{weight: 35, texts: []string{"A correnteza levou {@} uns dois quilômetros rio abaixo. A volta foi a pé, molhada e sem tesouro"}, effects: []effect{injure(10, 25), delay(20 * time.Minute)}},
					{weight: 17, texts: []string{"Embaixo da pedra só tinha uma caixa de isopor com uma latinha e um bilhete: 'trouxa'"}, effects: []effect{gain(1, "latao")}},
					{weight: 8, texts: []string{"{@} afundou no meio do caminho e nunca chegou na ilhota"}, effects: []effect{perish()}},
				}},
				{label: "🚣 Alugar um barquinho", risk: midRisk, price: cost{percent: 4, floor: 80}, endings: []ending{
					{weight: 55, texts: []string{"Remando com calma, {@} chegou na ilhota e desenterrou um baú de verdade"}, effects: []effect{gain(1, "bau-do-tesouro")}},
					{weight: 30, texts: []string{"O tesouro era um pote com uma pérola e uma moeda antiga. Pequeno, mas verdadeiro"}, effects: []effect{gain(1, "perola"), gain(1, "moeda-do-imperio")}},
					{weight: 15, texts: []string{"Alguém chegou antes: só tinha um buraco e uma pá esquecida"}},
				}},
				{label: "🗑️ Desistir, deve ser pegadinha", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} rasgou o mapa. Nunca vai saber se era verdade, e vai pensar nisso toda noite antes de dormir"}},
				}},
			},
		},
		{
			id:    "pesca-piracema",
			title: "Fiscal da piracema",
			when:  wentWell,
			intros: []string{
				"Chegou um fiscal do Ibama de colete e prancheta: 'tá sabendo que é época de piracema? Pesca proibida, cidadão'",
				"{@} tava com o balde cheio quando um barco do Ibama encostou na margem. O fiscal já veio olhando pro balde",
			},
			choices: []choice{
				{label: "🙈 Esconder o balde no mato", risk: midRisk, endings: []ending{
					{weight: 55, favor: func(p Profession) float64 { return p.escapeBonus }, texts: []string{"O fiscal olhou pro outro lado bem na hora e {@} escapou com a pescaria"}},
					{weight: 45, texts: []string{"O fiscal achou o balde atrás da moita, aplicou a multa e ainda levou os peixes de prova"}, effects: []effect{lose(4, 10)}},
				}},
				{label: "😇 Soltar tudo de volta no rio", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} devolveu os peixes pro rio. O fiscal liberou com um sermão, e os peixes agradeceram do jeito deles"}, effects: []effect{enchant("maoboa")}},
				}},
				{label: "💸 Molhar a mão do fiscal", risk: highRisk, price: cost{percent: 3, floor: 60}, endings: []ending{
					{weight: 65, texts: []string{"O fiscal contou o dinheiro, guardou no colete e jurou que não viu nada"}, effects: []effect{stain()}},
					{weight: 35, texts: []string{"O fiscal era honesto e levou {@} direto pra delegacia por tentativa de suborno"}, effects: []effect{nabbed()}},
				}},
				{label: "🪪 Mostrar a carteira de pescador", risk: lowRisk, needs: isJob("pescador"), endings: []ending{
					{weight: 100, texts: []string{"Pescador profissional tem licença. O fiscal pediu desculpa e ainda contou onde o peixe tá mordendo"}, effects: []effect{findLoot(), refresh()}},
				}},
			},
		},
		{
			id:    "pesca-barco-furado",
			title: "Água no barco",
			intros: []string{
				"{@} tava no meio do rio quando sentiu o pé molhado. O barquinho alugado tá furado e enchendo rápido",
				"Começou a entrar água pelo fundo do barco de {@}. Olhando bem, o remendo era de chiclete",
			},
			choices: []choice{
				{label: "🚣 Remar feito doido", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} remou tão rápido que chegou na margem antes de afundar. Sem fôlego, mas inteiro"}},
					{weight: 40, texts: []string{"O barco afundou a três metros da margem. {@} chegou nadando e a vara ficou no fundo do rio"}, effects: []effect{breakTool(), injure(3, 10)}},
				}},
				{label: "🍬 Tapar o furo com chiclete", risk: lowRisk, endings: []ending{
					{weight: 50, texts: []string{"Funcionou! Engenharia de pescador. {@} ainda pescou mais um pouquinho"}, effects: []effect{findLoot()}},
					{weight: 50, texts: []string{"O chiclete não segurou. {@} chegou na margem a nado, encharcado de vergonha"}, effects: []effect{injure(3, 8), delay(10 * time.Minute)}},
				}},
				{label: "🪣 Jogar a pescaria fora", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"Sem o peso dos peixes, o barco boiou até a margem. {@} salvou a pele e perdeu o almoço"}, effects: []effect{forfeit("tilapia", 2), forfeit("lambari", 3), forfeit("pacu", 1)}},
				}},
				{label: "🛟 Vestir a boia e seguir pescando", risk: lowRisk, needs: holds("boia"), endings: []ending{
					{weight: 100, texts: []string{"Com a boia no peito, {@} deixou o barco afundar devagarinho e pescou até a água bater no queixo"}, effects: []effect{findLoot(), findLoot()}},
				}},
			},
		},
		{
			id:     "pesca-boto",
			title:  "O moço do chapéu branco",
			rarity: rareEvent,
			intros: []string{
				"Escureceu e apareceu na beira do rio um moço bonito de chapéu branco, todo de linho, chamando {@} pra dançar forró",
				"Do nada, um rapaz de terno branco e chapéu saiu da água sequinho, sorriu pra {@} e estendeu a mão: 'vamos dançar?'",
			},
			choices: []choice{
				{label: "💃 Aceitar a dança", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"{@} dançou a noite inteira e acordou na margem com o bolso cheio e um sorriso bobo"}, effects: []effect{earn(80, 200), enchant("sorte")}},
					{weight: 35, texts: []string{"{@} acordou na beira do rio sem a carteira e sem lembrar de nada. O moço sumiu"}, effects: []effect{lose(5, 12), delay(30 * time.Minute)}},
					{weight: 15, texts: []string{"O moço quis continuar a dança lá no fundo do rio. {@} foi resgatado engolindo água"}, effects: []effect{injure(25, 45)}},
					{weight: 10, texts: []string{"No meio do xote o chapéu caiu e revelou um buraco no alto da cabeça: era o boto! Antes de mergulhar, ele deixou um presente"}, effects: []effect{gain(1, "perola"), earn(100, 200)}},
				}},
				{label: "🙅 Recusar com educação", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"O moço tirou o chapéu, fez uma reverência e mergulhou no rio. Deixou uma pérola na areia"}, effects: []effect{gain(1, "perola")}},
					{weight: 40, texts: []string{"O moço ficou ofendido, pulou no rio e espantou todos os peixes da região"}, effects: []effect{delay(15 * time.Minute)}},
				}},
				{label: "🎩 Arrancar o chapéu dele", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"Debaixo do chapéu tinha um buraco de respirar: era o boto! Ele fugiu pro rio e deixou cair umas coisas do bolso"}, effects: []effect{gain(1, "perola"), earn(50, 120)}},
					{weight: 45, texts: []string{"O boto ficou bravo e encantou {@}: peixe nenhum vai chegar perto tão cedo"}, effects: []effect{enchant("panema")}},
				}},
				{label: "🙏 Rezar um pai-nosso", risk: lowRisk, needs: inArea(areaFe), endings: []ending{
					{weight: 100, texts: []string{"{@} rezou alto e o moço se desfez em espuma. Ficou na margem um cheiro de perfume e uma bênção"}, effects: []effect{enchant("protecao"), enchant("maoboa")}},
				}},
			},
		},
		{
			id:    "pesca-tempestade",
			title: "Nuvem preta",
			intros: []string{
				"O céu ficou preto em cinco minutos e já tá trovejando longe. {@} tá com a linha na água e o peixe começando a morder",
				"Ventania, trovão e cheiro de chuva forte. Os outros pescadores já foram embora e {@} ficou sozinho no barranco",
			},
			choices: []choice{
				{label: "🏠 Recolher tudo e ir embora", risk: lowRisk, endings: []ending{
					{weight: 65, texts: []string{"{@} foi embora e a chuva passou em vinte minutos. Deu pra voltar logo depois"}, effects: []effect{refresh()}},
					{weight: 35, texts: []string{"Choveu a tarde inteira. {@} viu tudo da janela, comendo pipoca"}},
				}},
				{label: "🌧️ Pescar na chuva", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Peixe adora chuva: {@} encheu o balde enquanto todo mundo fugia"}, effects: []effect{findLoot(), findLoot()}},
					{weight: 35, texts: []string{"{@} ficou encharcado, pegou um resfriado e só pescou gripe"}, effects: []effect{injure(5, 12)}},
					{weight: 15, texts: []string{"Um raio caiu do outro lado do rio e a vara de {@} soltou faísca"}, effects: []effect{injure(15, 30), breakTool()}},
				}},
				{label: "🌳 Se abrigar debaixo da árvore", risk: highRisk, endings: []ending{
					{weight: 72, texts: []string{"A árvore aguentou firme e {@} ficou sequinho, filmando a tempestade pro status"}},
					{weight: 20, texts: []string{"Um galho quebrou com o vento e acertou {@} em cheio"}, effects: []effect{injure(10, 25)}},
					{weight: 8, texts: []string{"Caiu um raio bem na árvore. Ninguém avisou {@} que árvore em tempestade é para-raios"}, effects: []effect{perish()}},
				}},
			},
			idle: ending{texts: []string{"{@} ficou parado decidindo e tomou a chuva toda na cabeça"}, effects: []effect{injure(2, 6)}},
		},
		{
			id:    "pesca-piranha-no-pe",
			title: "Piranha no pé",
			intros: []string{
				"{@} tava pescando com o pé na água pra refrescar quando sentiu uma beliscada. Depois outra. Depois várias",
				"Alguma coisa mordeu o dedão de {@} dentro do rio. Olhando melhor, a água tá fervilhando de piranha",
			},
			choices: []choice{
				{label: "🦶 Tirar o pé correndo", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} tirou o pé a tempo. Ficou só a marquinha de dente e a lição"}, effects: []effect{injure(2, 6)}},
					{weight: 30, texts: []string{"{@} tirou o pé tão rápido que caiu de costas no barranco"}, effects: []effect{injure(4, 10)}},
				}},
				{label: "🐟 Agarrar uma com a mão", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} pegou a danada no tapa e levou de troféu, com o dedo inchado"}, effects: []effect{gain(1, "piranha"), injure(5, 10)}},
					{weight: 45, texts: []string{"A piranha mordeu de volta e chamou as amigas pra festa"}, effects: []effect{injure(10, 22)}},
				}},
				{label: "🥪 Jogar o lanche pra distrair", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"As piranhas foram atrás do sanduíche e {@} pescou tranquilo enquanto elas brigavam"}, effects: []effect{findLoot()}},
					{weight: 30, texts: []string{"As piranhas comeram o lanche em três segundos e voltaram com mais fome"}, effects: []effect{injure(3, 8)}},
				}},
				{label: "🩹 Fazer um curativo na hora", risk: lowRisk, uses: "curativo", endings: []ending{
					{weight: 100, texts: []string{"{@} colou o curativo, ficou novo em folha e ainda fisgou uma piranha distraída"}, effects: []effect{heal(15), gain(1, "piranha")}},
				}},
			},
		},
		{
			id:     "pesca-tilapia-falante",
			title:  "A tilápia que fala",
			rarity: rareEvent,
			intros: []string{
				"{@} tirou uma tilápia da água e ela abriu a boca: 'me solta que eu realizo um desejo seu'",
				"A tilápia que {@} acabou de pescar piscou, tossiu e falou com voz de locutor de rádio: 'faço um pedido seu se você me devolver'",
			},
			choices: []choice{
				{label: "💰 Pedir dinheiro", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"'Feito!', disse a tilápia, e caiu um Pix na conta de {@} antes dela sumir na água"}, effects: []effect{earn(250, 500)}},
					{weight: 30, texts: []string{"A tilápia riu, disse que era pegadinha e deixou umas moedinhas de consolação"}, effects: []effect{earn(40, 90)}},
				}},
				{label: "🍀 Pedir sorte", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"A tilápia soprou umas bolhas na cara de {@}. Dizem que é bênção de peixe"}, effects: []effect{enchant("sorte"), enchant("maoboa")}},
				}},
				{label: "🍳 Fritar ela mesmo assim", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Ficou deliciosa, mas a vó da tilápia rogou uma praga lá do fundo do rio"}, effects: []effect{enchant("panema"), heal(10)}},
					{weight: 50, texts: []string{"Na frigideira ela gritou 'vai se arrepender!'. {@} comeu assim mesmo e, por enquanto, nada aconteceu"}, effects: []effect{heal(15)}},
				}},
				{label: "🤝 Soltar sem pedir nada", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} soltou a tilápia sem pedir nada. Ela voltou com um presente na boca e a família toda pra agradecer"}, effects: []effect{gain(1, "perola"), findLoot()}},
				}},
			},
		},
	},
}
