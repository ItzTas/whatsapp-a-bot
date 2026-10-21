package economy

import "time"

var tyrannyEvents = eventBook{
	name:   "🪖 Ditadura",
	places: []string{"!trabalhar", "!pescar", "!minerar", "!vasculhar", "!catar", "!roubar", "!furtar", "!sequestrar", "!matar", "!assalto", "!quadrilha", "!mercadonegro", "!postar", "!publi"},
	scenarios: []scenario{
		{
			id:    "ditadura-panelaco",
			title: "O panelaço",
			when:  underTyranny,
			intros: []string{
				"Deu nove da noite e começou um barulho de panela em todas as janelas do bairro. É panelaço contra o ditador, e o vizinho de {@} já tá batendo a frigideira",
				"Alguém mandou no grupo do condomínio: \"PANELAÇO AGORA CONTRA A DITADURA\". {@} olhou pra panela de pressão em cima do fogão",
			},
			choices: []choice{
				{label: "🍳 Bater panela na janela", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} bateu tanto que a rua inteira entrou junto. Dava pra ouvir do palácio"}, effects: []effect{rebel(2, 4)}},
					{weight: 30, texts: []string{"A panela amassou toda, mas o barulho valeu a pena"}, effects: []effect{rebel(1, 2), lose(1, 2)}},
				}},
				{label: "📢 Puxar o grito na rua", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} subiu num banco da praça, puxou o grito e virou a voz do bairro"}, effects: []effect{rebel(4, 7)}},
					{weight: 30, texts: []string{"O grito pegou, mas um soldado anotou a cara de {@}"}, effects: []effect{rebel(3, 5), stain()}},
					{weight: 20, texts: []string{"Os soldados chegaram com bomba de efeito moral e {@} saiu tossindo"}, effects: []effect{injure(5, 15)}},
				}},
				{label: "📺 Fechar a janela e ver novela", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} aumentou o volume da novela e fingiu que não era com ele(a). O ditador agradece o silêncio"}},
				}},
			},
			idle: ending{texts: []string{"O panelaço acabou antes de {@} achar a tampa da panela"}},
		},
		{
			id:    "ditadura-cartaz",
			title: "A cara do ditador no poste",
			when:  underTyranny,
			intros: []string{
				"Amanheceu um cartaz gigante do ditador em cada poste da rua de {@}, com a frase \"O PAI DO POVO\". Embaixo, em letra miúda: proibido arrancar",
				"Os soldados colaram a foto do ditador sorrindo bem na porta de {@}. Ninguém tá olhando agora",
			},
			choices: []choice{
				{label: "🎨 Desenhar um bigode nele", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O bigode ficou tão bom que a foto rodou todos os grupos da cidade"}, effects: []effect{rebel(1, 3)}},
					{weight: 30, texts: []string{"Um turista achou que era arte de rua e pagou pra {@} desenhar outro"}, effects: []effect{earn(20, 60)}},
				}},
				{label: "✂️ Arrancar o cartaz", risk: highRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} arrancou o cartaz e o bairro inteiro arrancou os outros em seguida"}, effects: []effect{rebel(2, 5)}},
					{weight: 30, texts: []string{"Arrancou, mas a câmera do poste filmou tudo"}, effects: []effect{rebel(1, 3), stain()}},
					{weight: 15, texts: []string{"Uma patrulha virou a esquina bem na hora e levou {@} pra delegacia"}, effects: []effect{nabbed()}},
				}},
				{label: "🫡 Bater continência pro cartaz", risk: lowRisk, endings: []ending{
					{weight: 50, texts: []string{"Um fiscal do regime viu a continência e deu um vale lealdade pra {@}"}, effects: []effect{earn(30, 80)}},
					{weight: 50, texts: []string{"Ninguém viu a continência, e {@} ficou com vergonha sozinho(a) na rua"}},
				}},
			},
		},
		{
			id:    "ditadura-fila-racionamento",
			title: "A fila do racionamento",
			when:  underTyranny,
			intros: []string{
				"O ditador decretou racionamento e a fila do mercado dobra o quarteirão. {@} tá com fome e a fila não anda",
				"Cartaz na porta do mercado: \"UMA COXINHA POR CIDADÃO, POR ORDEM DO PAI DO POVO\". A fila já tem três horas",
			},
			choices: []choice{
				{label: "🧍 Esperar na fila", risk: lowRisk, endings: []ending{
					{weight: 65, texts: []string{"Depois de horas em pé, {@} saiu com a marmita do dia"}, effects: []effect{gain(1, "marmita"), delay(10 * time.Minute)}},
					{weight: 35, texts: []string{"Acabou tudo bem na vez de {@}. Horas perdidas"}, effects: []effect{delay(15 * time.Minute)}},
				}},
				{label: "🗣️ Reclamar alto do racionamento", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"A fila inteira concordou com {@} e começou a gritar junto"}, effects: []effect{rebel(2, 5)}},
					{weight: 30, texts: []string{"Um soldado da porta deu uma cacetada em {@}, e a fila ficou ainda mais revoltada"}, effects: []effect{rebel(2, 4), injure(5, 12)}},
					{weight: 20, texts: []string{"O gerente deu uma coxinha pra {@} calar a boca"}, effects: []effect{gain(1, "coxinha")}},
				}},
				{label: "🛒 Comprar no câmbio negro", risk: midRisk, price: cost{percent: 2, floor: 60}, endings: []ending{
					{weight: 80, texts: []string{"O cara do porta-malas tinha de tudo, e {@} saiu cheio(a) de comida"}, effects: []effect{gain(2, "coxinha", "pudim", "marmita")}},
					{weight: 20, texts: []string{"O cara do porta-malas era fiscal disfarçado"}, effects: []effect{stain()}},
				}},
			},
		},
		{
			id:    "ditadura-alistamento",
			title: "O alistamento obrigatório",
			when:  underTyranny,
			intros: []string{
				"Um caminhão do exército parou na rua e um sargento começou a puxar todo mundo pro alistamento obrigatório. Ele tá vindo na direção de {@}",
				"\"O PAI DO POVO PRECISA DE VOCÊ!\" O sargento apontou bem pra {@}",
			},
			choices: []choice{
				{label: "🏃 Fugir pelos fundos", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} pulou dois muros e sumiu antes do sargento terminar a frase"}},
					{weight: 40, texts: []string{"O segundo muro era mais alto do que parecia"}, effects: []effect{injure(5, 15)}},
				}},
				{label: "🪖 Se alistar", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"O soldo até que é bom, mas o treinamento tomou a tarde inteira de {@}"}, effects: []effect{earn(80, 200), delay(20 * time.Minute)}},
					{weight: 45, texts: []string{"{@} passou o dia limpando a bota do sargento e não ganhou nada"}, effects: []effect{delay(20 * time.Minute)}},
				}},
				{label: "💸 Molhar a mão do sargento", risk: lowRisk, price: cost{percent: 3, floor: 100}, endings: []ending{
					{weight: 90, texts: []string{"O sargento guardou o dinheiro no bolso e esqueceu que {@} existia"}},
					{weight: 10, texts: []string{"O sargento era o único honesto do exército"}, effects: []effect{stain()}},
				}},
			},
			idle: ending{texts: []string{"{@} ficou parado(a), e o sargento achou que era estátua e passou reto"}},
		},
		{
			id:     "ditadura-radio-pirata",
			title:  "A rádio pirata",
			rarity: rareEvent,
			when:   underTyranny,
			intros: []string{
				"No porão de um bar, {@} achou um transmissor escondido. Do lado, um bilhete: \"Rádio Liberdade. Fala o que quiser, a cidade toda tá ouvindo\"",
				"Um rapaz de capuz puxou {@} pelo braço: \"A rádio pirata precisa de uma voz nova hoje. Topa?\"",
			},
			choices: []choice{
				{label: "📻 Falar contra o ditador no ar", risk: highRisk, endings: []ending{
					{weight: 55, texts: []string{"O discurso de {@} passou na cidade inteira e o povo foi pra rua"}, effects: []effect{rebel(6, 10)}},
					{weight: 25, texts: []string{"O discurso passou, mas os soldados rastrearam o sinal até o porão"}, effects: []effect{rebel(3, 6), nabbed()}},
					{weight: 15, texts: []string{"Os soldados invadiram o porão no meio da transmissão e baixaram o cacete"}, effects: []effect{injure(15, 35)}},
					{weight: 5, texts: []string{"O exército não quis prender ninguém. Foi a última transmissão de {@}"}, effects: []effect{perish()}},
				}},
				{label: "🎧 Só ouvir escondido(a)", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} ouviu tudo e espalhou as notícias no grupo da família"}, effects: []effect{rebel(1, 2)}},
					{weight: 30, texts: []string{"O sinal caiu e {@} ficou só com o chiado"}},
				}},
				{label: "🚔 Entregar a rádio pro regime", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"O regime pagou bem pela dica, e a resistência perdeu a voz"}, effects: []effect{earn(100, 250), betray(3, 6)}},
					{weight: 40, texts: []string{"A resistência descobriu quem dedurou e foi tirar satisfação com {@}"}, effects: []effect{betray(2, 4), injure(10, 25)}},
				}},
			},
		},
		{
			id:     "ditadura-porao",
			title:  "A reunião no porão",
			rarity: rareEvent,
			when:   underTyranny,
			intros: []string{
				"Um bilhete apareceu no bolso de {@}: \"Meia-noite, no porão da padaria. Traz ninguém\". Lá embaixo, gente planejando derrubar o ditador",
				"{@} desceu pro porão da padaria e deu de cara com metade do bairro em volta de um mapa do quartel",
			},
			choices: []choice{
				{label: "🤝 Entrar na conspiração", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} virou peça chave da resistência e espalhou a palavra pela cidade"}, effects: []effect{rebel(3, 6)}},
					{weight: 30, texts: []string{"Na mesma noite, a turma soltou os cavalos da cavalaria do ditador"}, effects: []effect{sabotage(1), rebel(1, 3)}},
					{weight: 20, texts: []string{"Tinha um X-9 na reunião. A polícia bateu no porão de madrugada"}, effects: []effect{nabbed()}},
				}},
				{label: "💣 Sabotar o quartel na mesma noite", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} soltou um rojão no paiol e meio exército saiu correndo"}, effects: []effect{sabotage(2), rebel(2, 4)}},
					{weight: 30, texts: []string{"O rojão explodiu cedo demais, na mão de {@}"}, effects: []effect{injure(15, 35)}},
					{weight: 20, texts: []string{"A sentinela tava acordada"}, effects: []effect{nabbed()}},
				}},
				{label: "🚪 Sair de fininho", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} disse que ia no banheiro e nunca mais voltou. Ninguém ali vai esquecer"}},
				}},
			},
		},
		{
			id:    "ditadura-protesto-palacio",
			title: "Protesto na frente do palácio",
			when:  tyrantRules,
			intros: []string{
				"Tem uma multidão gritando na frente do palácio, com faixa e boneco de pano de {@}. O chefe da guarda pergunta o que fazer",
				"\"FORA DITADOR!\" O coro tá tão alto que {@} não consegue nem tirar a soneca da tarde",
			},
			choices: []choice{
				{label: "🪖 Mandar o exército dispersar", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Os soldados limparam a praça em dez minutos, e o povo voltou pra casa de cabeça baixa"}, effects: []effect{quell(3, 6)}},
					{weight: 30, texts: []string{"A foto de um soldado empurrando uma velhinha rodou o mundo"}, effects: []effect{outrage(3, 6)}},
					{weight: 20, texts: []string{"A praça esvaziou, mas um batalhão ficou com pena do povo e desertou"}, effects: []effect{quell(1, 3), desert(1)}},
				}},
				{label: "🎤 Fazer um discurso da sacada", risk: midRisk, endings: []ending{
					{weight: 45, texts: []string{"{@} prometeu churrasco pra todo mundo e a multidão foi embora feliz"}, effects: []effect{quell(2, 5)}},
					{weight: 35, texts: []string{"{@} chamou o povo de vagabundo no microfone ligado"}, effects: []effect{outrage(2, 4)}},
					{weight: 20, texts: []string{"Uma pedra voou da multidão bem na testa de {@}"}, effects: []effect{injure(5, 15)}},
				}},
				{label: "🍞 Distribuir pão e circo", risk: lowRisk, price: cost{percent: 5, floor: 300}, endings: []ending{
					{weight: 85, texts: []string{"Pão de graça e show de pagode na praça. Ninguém lembra mais o que tava protestando"}, effects: []effect{quell(4, 8)}},
					{weight: 15, texts: []string{"O povo comeu o pão, curtiu o pagode e continuou gritando"}, effects: []effect{quell(1, 2)}},
				}},
				{label: "🙉 Ignorar e fechar a cortina", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"Ignorar só deixou a multidão maior"}, effects: []effect{outrage(2, 4)}},
					{weight: 40, texts: []string{"Choveu forte e todo mundo foi pra casa sozinho"}},
				}},
			},
		},
		{
			id:     "ditadura-general",
			title:  "O general ambicioso",
			rarity: rareEvent,
			when:   tyrantRules,
			intros: []string{
				"O General Bigodão pediu uma audiência particular com {@}. Ele quer mais dinheiro, mais medalha e, pelo olhar, talvez a cadeira de {@}",
				"Chegou um recado do quartel: o General Bigodão anda dizendo por aí que mandaria melhor que {@}",
			},
			choices: []choice{
				{label: "💰 Dar um aumento pro general", risk: lowRisk, price: cost{percent: 6, floor: 400}, endings: []ending{
					{weight: 80, texts: []string{"Lealdade comprada. O general ficou tão feliz que trouxe mais tropa"}, effects: []effect{enlist(1)}},
					{weight: 20, texts: []string{"O general embolsou o aumento e continua olhando torto"}},
				}},
				{label: "🎖️ Dar uma medalha de lata", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O general nem percebeu que era lata e saiu desfilando com ela"}},
					{weight: 50, texts: []string{"O general mordeu a medalha, viu que era lata e foi embora levando um batalhão"}, effects: []effect{desert(1)}},
				}},
				{label: "⛓️ Mandar prender o general", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"O general foi preso e a tropa dele passou pro lado de {@}"}, effects: []effect{enlist(2)}},
					{weight: 30, texts: []string{"O general fugiu com dois batalhões e foi dar entrevista falando mal de {@}"}, effects: []effect{desert(2), outrage(2, 4)}},
					{weight: 15, texts: []string{"O general resistiu à prisão no braço"}, effects: []effect{injure(15, 40)}},
					{weight: 5, texts: []string{"O general sacou primeiro"}, effects: []effect{perish()}},
				}},
				{label: "🙈 Fingir que não ouviu", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"O general se distraiu com outra coisa e esqueceu o assunto"}},
					{weight: 40, texts: []string{"O general entendeu o silêncio como fraqueza e levou um batalhão pra passear"}, effects: []effect{desert(1)}},
				}},
			},
		},
		{
			id:    "ditadura-bajuladores",
			title: "O baile dos bajuladores",
			when:  tyrantRules,
			intros: []string{
				"Os ministros fizeram um baile em homenagem a {@}, com bolo de cinco andares e a cara de {@} em cima",
				"Tem um baile no palácio e todo convidado chega com um presente e um elogio decorado pra {@}",
			},
			choices: []choice{
				{label: "🥂 Curtir a festa", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"Os ministros encheram o palácio de presentes caros"}, effects: []effect{earn(50, 150)}},
					{weight: 40, texts: []string{"{@} comeu, bebeu e dormiu como um rei"}, effects: []effect{heal(15)}},
				}},
				{label: "🕵️ Desconfiar de todo mundo", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} pegou um ministro conspirando e confiscou os seguranças dele"}, effects: []effect{enlist(1)}},
					{weight: 50, texts: []string{"{@} acusou o ministro errado no meio do baile e a história vazou"}, effects: []effect{outrage(1, 3)}},
				}},
				{label: "🎁 Exigir presentes melhores", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Assustados, os ministros dobraram os presentes"}, effects: []effect{earn(100, 250)}},
					{weight: 50, texts: []string{"Um ministro ofendido contou tudo pra imprensa"}, effects: []effect{outrage(2, 4)}},
				}},
			},
		},
		{
			id:    "ditadura-estatua",
			title: "A estátua do ditador",
			when:  tyrantRules,
			intros: []string{
				"O ministro da cultura chegou com uma maquete: uma estátua de {@} montado(a) num cavalo, no meio da praça principal",
				"\"Todo grande líder tem uma estátua\", disse o puxa-saco oficial, já com o escultor do lado de {@}",
			},
			choices: []choice{
				{label: "🗿 Estátua de ouro gigante", risk: midRisk, price: cost{percent: 8, floor: 500}, endings: []ending{
					{weight: 60, texts: []string{"A estátua ficou tão imponente que a tropa se encheu de orgulho e o povo ficou com medo"}, effects: []effect{enlist(1), quell(1, 3)}},
					{weight: 40, texts: []string{"Estátua de ouro em plena época de racionamento. O povo ficou furioso"}, effects: []effect{outrage(2, 4)}},
				}},
				{label: "🪨 Estátua de isopor", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"De longe ninguém percebe que é isopor"}},
					{weight: 40, texts: []string{"Choveu, a cabeça da estátua derreteu e virou meme na cidade toda"}, effects: []effect{outrage(1, 2)}},
				}},
				{label: "🙅 Recusar a estátua", risk: lowRisk, endings: []ending{
					{weight: 40, texts: []string{"O povo achou a humildade de {@} bonita"}, effects: []effect{quell(1, 2)}},
					{weight: 60, texts: []string{"Ninguém ficou sabendo que {@} recusou"}},
				}},
			},
		},
		{
			id:     "ditadura-espiao",
			title:  "O espião no palácio",
			rarity: rareEvent,
			when:   tyrantRules,
			intros: []string{
				"A guarda pegou um garçom do palácio fotografando os papéis de {@}. Ele é espião da resistência",
				"O cozinheiro do palácio confessou: passava tudo o que ouvia pra resistência. Agora tá amarrado na frente de {@}",
			},
			choices: []choice{
				{label: "⚖️ Julgamento público", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O julgamento serviu de exemplo, e o povo ficou com medo de conspirar"}, effects: []effect{quell(3, 6)}},
					{weight: 50, texts: []string{"O espião fez um discurso emocionante no tribunal e virou herói nacional"}, effects: []effect{outrage(3, 6)}},
				}},
				{label: "🤝 Virar o espião pro seu lado", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"O espião entregou o esconderijo da resistência"}, effects: []effect{quell(4, 8)}},
					{weight: 40, texts: []string{"Era agente duplo: sumiu levando um batalhão junto"}, effects: []effect{desert(1)}},
				}},
				{label: "🚪 Expulsar do país", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O espião foi pra fronteira e ninguém ouviu mais falar dele"}},
					{weight: 30, texts: []string{"Do exílio, o espião começou um podcast falando mal de {@}"}, effects: []effect{outrage(1, 2)}},
				}},
			},
		},
		{
			id:     "ditadura-cofre-secreto",
			title:  "O cofre do governo anterior",
			rarity: legendaryEvent,
			when:   tyrantRules,
			intros: []string{
				"Reformando o porão do palácio, os pedreiros acharam um cofre do governo anterior, cheio até a boca. Só {@} sabe",
				"Atrás de um quadro do palácio tinha um cofre. Dentro, o tesouro que o presidente anterior esqueceu de levar. {@} tá olhando pra ele",
			},
			choices: []choice{
				{label: "💰 Embolsar tudo", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} encheu os bolsos. O pedreiro contou pra todo mundo, mas o dinheiro já tava guardado"}, effects: []effect{earn(300, 900), outrage(2, 5)}},
					{weight: 40, texts: []string{"{@} embolsou tudo e ninguém nunca soube"}, effects: []effect{earn(150, 400)}},
				}},
				{label: "🏗️ Gastar em obras pro povo", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"Hospital novo, praça nova e asfalto na periferia. O povo até esqueceu que é ditadura"}, effects: []effect{quell(6, 10)}},
				}},
				{label: "🪖 Contratar soldados com o dinheiro", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"O tesouro virou farda, fuzil e coturno"}, effects: []effect{enlist(2)}},
				}},
			},
		},
	},
}
