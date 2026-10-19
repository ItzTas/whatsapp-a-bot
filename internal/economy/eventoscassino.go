package economy

var casinoWrathEvents = eventBook{
	name: "🎰 Cassino furioso",
	scenarios: []scenario{
		{
			id:     "cassino-matador",
			title:  "Encomenda dos cassinos",
			places: []string{casinoWrathPlace},
			intros: []string{
				"Os donos de cassino se reuniram num porão e chegaram num acordo: o decreto de {@} tá quebrando a banca, e alguém tem que resolver isso. Um matador de aluguel já pegou o adiantamento",
				"Chegou um recado no gabinete de {@}, escrito com letras recortadas de revista: 'ou o cassino volta ao normal, ou o presidente vira estatística'",
			},
			choices: []choice{
				{label: "🕶️ Dobrar a segurança", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"Os seguranças de {@} acharam o matador escondido no estacionamento e mandaram ele de volta com um recado"}},
					{weight: 30, texts: []string{"O matador acertou de raspão antes dos seguranças derrubarem ele"}, effects: []effect{injure(15, 30)}},
				}},
				{label: "🚗 Mudar a rota todo dia", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} passou a semana indo trabalhar de mototáxi, ônibus e patinete. O matador desistiu, tonto"}},
					{weight: 45, texts: []string{"Numa das rotas novas, o carro de {@} levou uma rajada. Saiu do capô, inteiro por milagre"}, effects: []effect{injure(20, 40)}},
				}},
				{label: "📜 Revogar o decreto", risk: lowRisk, endings: []ending{
					{weight: 80, texts: []string{"{@} revogou o decreto e a encomenda foi cancelada. O matador até devolveu o adiantamento pros cassinos"}, effects: []effect{repealCasino(), enchant("protecao")}},
					{weight: 20, texts: []string{"{@} revogou o decreto, mas o matador já tinha recebido e é profissional. Pelo menos errou"}, effects: []effect{repealCasino(), injure(10, 20)}},
				}},
				{label: "😤 Ir pessoalmente no cassino", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"{@} entrou no cassino de peito aberto, fez um discurso na mesa de roleta e os donos amarelaram. Ainda saiu com uma ficha de presente"}, effects: []effect{earn(150, 300)}},
					{weight: 45, texts: []string{"Os seguranças do cassino deram uma surra em {@} na saída dos fundos"}, effects: []effect{injure(30, 50)}},
					{weight: 15, texts: []string{"O matador tava lá dentro, esperando. {@} entrou e não saiu"}, effects: []effect{perish()}},
				}},
			},
			idle: ending{texts: []string{"{@} não deu bola, e o matador, mal pago, foi tentar a vida em outro ramo"}},
		},
		{
			id:     "cassino-mala",
			title:  "A mala preta",
			places: []string{casinoWrathPlace},
			intros: []string{
				"Um homem de chapéu deixou uma mala preta em cima da mesa de {@}: 'presente dos cassinos. É só o decreto sumir'",
				"O dono do maior cassino da cidade convidou {@} pra um café e empurrou um envelope gordo pela mesa: 'a gente pode resolver isso como gente civilizada'",
			},
			choices: []choice{
				{label: "💼 Aceitar e revogar", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} pegou a mala, revogou o decreto e todo mundo saiu sorrindo. Menos quem aposta"}, effects: []effect{earn(150, 300), repealCasino()}},
				}},
				{label: "🙅 Recusar a propina", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} recusou na cara dura, e a história vazou. O povo achou bonito"}, effects: []effect{enchant("sorte")}},
					{weight: 50, texts: []string{"O dono do cassino não gostou da recusa. Na saída, alguém empurrou {@} da escada"}, effects: []effect{injure(15, 35)}},
				}},
				{label: "🤑 Pegar a mala e não revogar", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"{@} sumiu com a mala e deixou o decreto do jeito que tava. Os cassinos tão até agora sem entender"}, effects: []effect{earn(200, 300)}},
					{weight: 45, texts: []string{"Os cassinos foram cobrar a mala com juros e com taco de beisebol"}, effects: []effect{lose(10, 20), injure(30, 50)}},
					{weight: 15, texts: []string{"Passar a perna nos cassinos tem um preço só, e {@} pagou"}, effects: []effect{perish()}},
				}},
				{label: "🚔 Mandar prender o portador", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"A polícia levou o homem da mala e {@} ainda posou pra foto contra a corrupção"}, effects: []effect{enchant("protecao")}},
					{weight: 40, texts: []string{"O homem da mala era primo do delegado. Saiu em dez minutos e prometeu voltar"}, effects: []effect{enchant("olhogordo")}},
				}},
			},
			idle: ending{texts: []string{"{@} ficou olhando a mala tanto tempo que o homem de chapéu pegou ela de volta e foi embora"}},
		},
		{
			id:     "cassino-jantar",
			title:  "Jantar de negócios",
			places: []string{casinoWrathPlace},
			intros: []string{
				"Os donos de cassino convidaram {@} pra um jantar chique 'pra conversar sobre o decreto'. O garçom serviu um peixe com um cheiro meio estranho",
				"Na festa de um empresário do jogo, ofereceram pra {@} uma taça de vinho que ninguém mais tava bebendo",
			},
			choices: []choice{
				{label: "🍽️ Comer pra não fazer desfeita", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"Era só peixe ruim mesmo. {@} passou a noite no banheiro, mas sobreviveu"}, effects: []effect{injure(15, 30)}},
					{weight: 40, texts: []string{"Tinha veneno no prato. {@} foi parar no hospital e escapou por pouco"}, effects: []effect{injure(35, 50)}},
					{weight: 10, texts: []string{"O veneno era dos bons"}, effects: []effect{perish()}},
				}},
				{label: "🧑‍🍳 Mandar o provador comer antes", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O provador comeu, passou mal na hora, e {@} levantou da mesa sem tocar em nada"}},
					{weight: 30, texts: []string{"O provador comeu tudo, achou ótimo e repetiu. Os donos do cassino ficaram com a conta"}, effects: []effect{heal(10)}},
				}},
				{label: "🚪 Inventar uma desculpa e sair", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} disse que tinha reunião com o FMI e saiu antes da sobremesa"}},
					{weight: 30, texts: []string{"Os donos do cassino ficaram ofendidos e rogaram praga em {@} na mesa mesmo"}, effects: []effect{enchant("azar")}},
				}},
			},
			idle: ending{texts: []string{"{@} enrolou tanto pra escolher que o jantar esfriou e todo mundo foi embora"}},
		},
		{
			id:     "cassino-bomba",
			title:  "Tique-taque no carro",
			rarity: rareEvent,
			places: []string{casinoWrathPlace},
			intros: []string{
				"Quando {@} foi abrir o carro oficial, ouviu um tique-taque vindo de baixo do banco. Do lado, um bilhete: 'com carinho, dos cassinos'",
				"Um pacote chegou no gabinete de {@} com uma fita de cassino amarrada e um barulhinho de relógio lá dentro",
			},
			choices: []choice{
				{label: "📞 Chamar o esquadrão antibomba", risk: lowRisk, endings: []ending{
					{weight: 75, texts: []string{"O esquadrão desarmou a bomba em dois minutos e ainda deu uma bronca em quem montou"}},
					{weight: 25, texts: []string{"O esquadrão demorou, a bomba estourou com {@} ainda por perto e o carro virou sucata"}, effects: []effect{injure(20, 40), lose(3, 8)}},
				}},
				{label: "✂️ Cortar o fio vermelho", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"Era o fio certo! {@} desarmou a bomba sozinho e vendeu a peça pro museu do crime"}, effects: []effect{earn(200, 500)}},
					{weight: 40, texts: []string{"Era o fio azul"}, effects: []effect{injure(35, 50)}},
					{weight: 15, texts: []string{"Não tinha fio certo"}, effects: []effect{perish()}},
				}},
				{label: "🏃 Correr pra longe", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} correu dois quarteirões antes de ouvir o estouro. O carro era do governo, então tudo bem"}},
					{weight: 40, texts: []string{"O estouro jogou {@} no chão no meio da corrida"}, effects: []effect{injure(15, 35)}},
				}},
				{label: "📜 Revogar o decreto ali mesmo", risk: midRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} revogou o decreto pelo celular e o tique-taque parou sozinho. Tinha controle remoto"}, effects: []effect{repealCasino()}},
					{weight: 30, texts: []string{"{@} revogou o decreto, mas o relógio não sabia disso"}, effects: []effect{repealCasino(), injure(20, 40)}},
				}},
			},
			idle: ending{texts: []string{"{@} ficou olhando pro carro até o tique-taque parar. Era um despertador velho, e o susto passou"}},
		},
		{
			id:     "cassino-chefao",
			title:  "O chefão do jogo",
			rarity: legendaryEvent,
			places: []string{casinoWrathPlace},
			intros: []string{
				"Uma limusine parou na frente de {@}. Lá dentro, o chefão de todos os cassinos, fumando charuto: 'senta aí, presidente. A gente precisa conversar sobre esse teu decreto'",
				"{@} foi levado de olhos vendados pra um camarote no último andar do maior cassino do país. O chefão do jogo esperava, sorrindo de um jeito ruim",
			},
			choices: []choice{
				{label: "🤝 Fazer um acordo", risk: lowRisk, endings: []ending{
					{weight: 80, texts: []string{"{@} revogou o decreto e o chefão abriu uma conta VIP no nome do presidente. Negócio é negócio"}, effects: []effect{repealCasino(), earn(500, 1500)}},
					{weight: 20, texts: []string{"O acordo saiu, mas o chefão cobrou um 'pedágio' pela viagem de limusine"}, effects: []effect{repealCasino(), lose(5, 10)}},
				}},
				{label: "👊 Peitar o chefão", risk: highRisk, endings: []ending{
					{weight: 30, texts: []string{"{@} encarou o chefão sem piscar. Ele riu, disse que gostou da coragem e mandou um presente pelo respeito"}, effects: []effect{earn(500, 1500), enchant("protecao")}},
					{weight: 50, texts: []string{"O chefão não gosta de quem levanta a voz. {@} saiu do camarote pela escada, rolando"}, effects: []effect{injure(35, 50), lose(5, 15)}},
					{weight: 20, texts: []string{"O chefão fez um sinal com o charuto e as luzes se apagaram pra {@}"}, effects: []effect{perish()}},
				}},
				{label: "🎲 Apostar o decreto numa partida", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} ganhou a partida de pôquer contra o chefão, levou as fichas e o decreto ficou"}, effects: []effect{earn(300, 800)}},
					{weight: 50, texts: []string{"O baralho era marcado. {@} perdeu e o decreto caiu na mesa de jogo"}, effects: []effect{repealCasino(), lose(5, 10)}},
				}},
			},
			idle: ending{texts: []string{"{@} ficou calado tanto tempo que o chefão perdeu a paciência e mandou largar o presidente no meio da estrada"}},
		},
		{
			id:     "cassino-salinha",
			title:  "A salinha dos fundos",
			places: []string{casinoWatchPlace},
			intros: []string{
				"Com o decreto do presidente o cassino tá pagando demais, e a gerência anda de olho em quem ganha. Dois seguranças de terno chamaram {@}: 'acompanha a gente até a salinha'",
				"O gerente do cassino olhou feio pra {@} depois do último prêmio e cochichou no rádio. Um armário de terno já tava vindo",
			},
			choices: []choice{
				{label: "🚶 Ir pra salinha", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Na salinha, o gerente fez umas perguntas, revistou os bolsos de {@} e liberou com um 'tá de olho, hein'"}},
					{weight: 50, texts: []string{"Na salinha, os seguranças 'convenceram' {@} a devolver uma parte dos prêmios"}, effects: []effect{lose(5, 12), injure(5, 15)}},
				}},
				{label: "🏃 Sair de fininho", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} se misturou na fila da roleta e escapou pela porta giratória"}},
					{weight: 40, texts: []string{"A porta giratória travou com {@} dentro, e os seguranças tiveram tempo de dar um corretivo"}, effects: []effect{injure(10, 25)}},
				}},
				{label: "📣 Fazer escândalo", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} gritou que o cassino tava perseguindo quem ganha, e a gerência, com vergonha, deu uma ficha de cortesia"}, effects: []effect{earn(30, 100)}},
					{weight: 40, texts: []string{"O escândalo não comoveu ninguém, e {@} foi banido da mesa por umas horas de olho gordo"}, effects: []effect{enchant("olhogordo")}},
				}},
			},
			idle: ending{texts: []string{"{@} fingiu que não ouviu, e os seguranças acharam outro sortudo pra perturbar"}},
		},
		{
			id:     "cassino-trapaca",
			title:  "Acusado de trapaça",
			places: []string{casinoWatchPlace},
			intros: []string{
				"O crupiê apontou pra {@} na frente de todo mundo: 'esse aí tá contando carta, só pode'. Com o decreto pagando a mais, o cassino não quer sortudo",
				"Um cara de bigode da mesa do lado começou a gritar que {@} tava com ímã no bolso. O cassino adorou a desculpa",
			},
			choices: []choice{
				{label: "🙌 Mostrar que tá limpo", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} virou os bolsos do avesso, só tinha bala de menta. O crupiê pediu desculpa resmungando"}},
					{weight: 30, texts: []string{"Mesmo limpo, o cassino reteve uma 'taxa de investigação' de {@}"}, effects: []effect{lose(3, 8)}},
				}},
				{label: "🥊 Partir pra cima do crupiê", risk: highRisk, endings: []ending{
					{weight: 30, texts: []string{"O crupiê correu, e o cassino, com medo de processo, pagou um extra pra {@} ficar quieto"}, effects: []effect{earn(80, 200)}},
					{weight: 50, texts: []string{"A segurança do cassino caiu em cima de {@} antes do primeiro soco"}, effects: []effect{injure(25, 45)}},
					{weight: 20, texts: []string{"A polícia que fazia ronda no cassino levou {@} por agressão"}, effects: []effect{nabbed()}},
				}},
				{label: "💸 Devolver um pouco pra ficar em paz", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} devolveu umas fichas e o assunto morreu ali. Caro, mas sem hematoma"}, effects: []effect{lose(4, 8)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, o cara de bigode foi pego com ímã no próprio bolso"}},
		},
	},
}
