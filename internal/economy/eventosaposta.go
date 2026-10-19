package economy

var gamblingEvents = eventBook{
	name:   "🎰 Fundo do poço",
	places: []string{"!apostar", "!cassino", "!blackjack", "!crash", "!corrida"},
	scenarios: []scenario{
		{
			id:    "aposta-dica-quente",
			title: "A dica quente",
			intros: []string{
				"{@} olhou o saldo e bateu o desespero. Um cara de terno brilhante chegou no ouvido: 'tenho uma dica quente, essa não tem erro'",
				"Depois de tanto perder, {@} recebeu no privado uma mensagem de um número desconhecido: 'sei o horário pagante, quer saber?'",
			},
			choices: []choice{
				{label: "🔥 Seguir a dica", risk: midRisk, endings: []ending{
					{weight: 35, texts: []string{"Não é que a dica era boa? {@} recuperou uma parte e saiu se achando gênio"}, effects: []effect{earn(80, 250)}},
					{weight: 45, texts: []string{"A dica era tão quente que queimou o resto do dinheiro de {@}"}, effects: []effect{lose(8, 15)}},
					{weight: 20, texts: []string{"O cara de terno pediu uma 'taxa de liberação' antes de passar a dica. Sumiu com o Pix e bloqueou {@}"}, effects: []effect{lose(15, 25)}},
				}},
				{label: "🛏️ Ir pra casa dormir", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} desligou o celular e dormiu doze horas seguidas. Acordou pobre, mas descansado"}, effects: []effect{heal(10)}},
					{weight: 30, texts: []string{"Uma noite de sono e {@} acordou com a cabeça no lugar e a sensação de que a maré vai virar"}, effects: []effect{enchant("sorte")}},
				}},
				{label: "🙏 Pedir um sinal pro santo", risk: lowRisk, endings: []ending{
					{weight: 40, texts: []string{"Um trevo de quatro folhas caiu do céu na frente de {@}. Sinal mais claro impossível"}, effects: []effect{enchant("sorte")}},
					{weight: 30, texts: []string{"O santo respondeu com um gato preto atravessando a rua. {@} entendeu o recado"}, effects: []effect{enchant("azar")}},
					{weight: 30, texts: []string{"O santo deixou {@} no vácuo. Deve estar ocupado com quem trabalha"}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, o cara de terno já tava dando a mesma dica pra outro desesperado"}},
		},
		{
			id:    "aposta-agiota",
			title: "O agiota da porta",
			intros: []string{
				"Na saída, um homem de óculos escuros encostou em {@}: 'tá precisando de um dinheirinho pra virar o jogo? Juros camaradas'",
				"O agiota do bairro, que fica sempre na porta do cassino, já conhecia {@} de vista: 'vai uma ajuda aí, campeão?'",
			},
			choices: []choice{
				{label: "💸 Pegar emprestado e jogar tudo", risk: highRisk, endings: []ending{
					{weight: 30, texts: []string{"{@} virou o jogo, pagou o agiota na hora e ainda sobrou. Saiu de lá sem acreditar"}, effects: []effect{earn(150, 300)}},
					{weight: 45, texts: []string{"Perdeu o emprestado também. O agiota cobrou com juros e com a mão pesada"}, effects: []effect{lose(15, 25), injure(20, 40)}},
					{weight: 20, texts: []string{"{@} perdeu tudo e passou a semana fugindo do agiota pelos becos"}, effects: []effect{lose(10, 20), injure(10, 25)}},
					{weight: 5, texts: []string{"{@} não pagou o agiota. O agiota não costuma dar segunda chance"}, effects: []effect{perish()}},
				}},
				{label: "🙅 Recusar educadamente", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"O agiota deu de ombros e foi atrás do próximo desesperado"}},
					{weight: 40, texts: []string{"O agiota não gostou do não e deu um empurrão em {@} na saída"}, effects: []effect{injure(5, 15)}},
				}},
				{label: "🏃 Sair pela porta dos fundos", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} escapou pela cozinha do cassino e ainda levou um salgado de brinde"}, effects: []effect{gain(1, "coxinha")}},
					{weight: 40, texts: []string{"A porta dos fundos dava num beco escuro, e {@} tropeçou numa caixa no escuro"}, effects: []effect{injure(10, 25)}},
				}},
				{label: "🔫 Mostrar que tá armado", risk: midRisk, needs: armedUp, endings: []ending{
					{weight: 60, texts: []string{"O agiota levantou as mãos, pediu desculpa e ainda largou uns trocados pra {@} esquecer o assunto"}, effects: []effect{earn(40, 120)}},
					{weight: 40, texts: []string{"O segurança viu a arma e chamou a polícia. Ficou anotado na ficha de {@}"}, effects: []effect{stain()}},
				}},
			},
			idle: ending{texts: []string{"{@} ficou parado tanto tempo que o agiota achou que era surdo e foi embora"}},
		},
		{
			id:    "aposta-penhora",
			title: "Só mais uma",
			intros: []string{
				"{@} tá encarando o celular e pensando seriamente em penhorar ele pra fazer só mais uma aposta. Só mais uma",
				"O saldo de {@} tá no vermelho emocional. Uma voz lá dentro diz: 'a próxima vira, confia'",
			},
			choices: []choice{
				{label: "📱 Penhorar e apostar", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"A próxima virou mesmo! {@} recuperou uma boa parte e resgatou o celular"}, effects: []effect{earn(100, 300)}},
					{weight: 45, texts: []string{"A próxima não virou. Nem a outra, nem a outra"}, effects: []effect{lose(10, 20)}},
					{weight: 15, texts: []string{"Perdeu a aposta e a loja de penhores ainda cobrou a taxa do contrato. {@} saiu com um boleto na mão"}, effects: []effect{lose(15, 25), gain(1, "boleto")}},
				}},
				{label: "🗑️ Desinstalar o app de aposta", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} apagou o app e sentiu um peso sair das costas, como se tivesse tirado um encosto"}, effects: []effect{enchant("purificacao")}},
					{weight: 40, texts: []string{"{@} desinstalou com toda a convicção do mundo e reinstalou cinco minutos depois"}},
				}},
				{label: "📲 Pedir Pix pra família", risk: midRisk, endings: []ending{
					{weight: 45, texts: []string{"A tia mandou o Pix junto com um áudio de oito minutos de sermão"}, effects: []effect{earn(30, 100)}},
					{weight: 35, texts: []string{"O grupo da família inteiro ficou sabendo do vício de {@}. Os olhares no almoço de domingo vão pesar"}, effects: []effect{enchant("olhogordo")}},
					{weight: 20, texts: []string{"Visualizaram e não responderam. A família de {@} já conhece essa história"}},
				}},
			},
			idle: ending{texts: []string{"{@} ficou tanto tempo encarando o celular que a bateria acabou e a decisão se tomou sozinha"}},
		},
		{
			id:     "aposta-gerente",
			title:  "O gerente do cassino",
			places: []string{"!cassino", "!blackjack"},
			intros: []string{
				"O gerente do cassino viu {@} perdendo sem parar e veio com um sorriso de vendedor: 'cliente VIP merece um drink por conta da casa'",
				"Depois de tanto prejuízo, {@} começou a desconfiar que a máquina é roubada. O gerente tá olhando de longe",
			},
			choices: []choice{
				{label: "🍸 Aceitar o drink da casa", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O drink era bom e {@} relaxou pela primeira vez no dia"}, effects: []effect{heal(15)}},
					{weight: 30, texts: []string{"O drink tava batizado. {@} acordou na calçada sem saber como chegou lá"}, effects: []effect{injure(10, 25)}},
					{weight: 20, texts: []string{"Brindou com o gerente, e o gerente sorriu de um jeito estranho. Aquele brinde trouxe urucubaca"}, effects: []effect{enchant("azar")}},
				}},
				{label: "😤 Reclamar que a máquina é roubada", risk: midRisk, endings: []ending{
					{weight: 35, texts: []string{"Pra evitar o barraco, o gerente devolveu uma parte do que {@} perdeu"}, effects: []effect{earn(50, 150)}},
					{weight: 40, texts: []string{"O segurança pegou {@} pelo colarinho e jogou na rua"}, effects: []effect{injure(15, 35)}},
					{weight: 25, texts: []string{"{@} chutou a máquina, ela quebrou, e a polícia anotou tudo"}, effects: []effect{stain()}},
				}},
				{label: "🚪 Sair de fininho", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} pegou o casaco e saiu sem olhar pra trás. O cassino sempre ganha, mas hoje não leva mais nada"}},
				}},
				{label: "🪪 Dar uma carteirada", risk: lowRisk, needs: lawman, endings: []ending{
					{weight: 60, texts: []string{"O gerente ficou pálido com o distintivo e devolveu uma parte pra {@} não investigar as máquinas"}, effects: []effect{earn(80, 200)}},
					{weight: 40, texts: []string{"O gerente era cunhado do delegado. A carteirada não deu em nada"}},
				}},
			},
			idle: ending{texts: []string{"O gerente cansou de esperar e foi oferecer o drink pra outro perdedor"}},
		},
		{
			id:     "aposta-descarrego",
			title:  "A rezadeira da esquina",
			rarity: rareEvent,
			intros: []string{
				"Na saída, uma senhora com um galho de arruda segurou o braço de {@}: 'meu filho, isso aí não é azar, é encosto'",
				"Uma rezadeira olhou pra {@} e fez o sinal da cruz três vezes: 'tem coisa ruim grudada em você, por isso não ganha nada'",
			},
			choices: []choice{
				{label: "🕯️ Pagar o descarrego", risk: lowRisk, price: cost{percent: 5, floor: 100}, endings: []ending{
					{weight: 60, texts: []string{"A rezadeira passou a arruda, cuspiu cachaça e {@} saiu de lá se sentindo outra pessoa"}, effects: []effect{enchant("purificacao"), enchant("sorte")}},
					{weight: 40, texts: []string{"Foi só uma reza rápida e um banho de sal grosso. Pelo menos {@} saiu cheiroso"}, effects: []effect{enchant("purificacao")}},
				}},
				{label: "😂 Rir da cara dela", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"A rezadeira deu de ombros e seguiu o caminho dela"}},
					{weight: 50, texts: []string{"Ela parou de andar, olhou pra trás e falou baixinho alguma coisa. {@} sentiu um arrepio na nuca"}, effects: []effect{enchant("encosto")}},
				}},
				{label: "🔢 Pedir os números da sorte", risk: midRisk, endings: []ending{
					{weight: 30, texts: []string{"Os números que ela soprou deram certo na hora! {@} recuperou uma parte"}, effects: []effect{earn(100, 300)}},
					{weight: 40, texts: []string{"Ela deu os números do azar de propósito, pra {@} aprender a largar o vício"}, effects: []effect{enchant("azar")}},
					{weight: 30, texts: []string{"Ela disse que número da sorte é trabalhar. {@} não gostou da resposta"}},
				}},
			},
			idle: ending{texts: []string{"A rezadeira esperou um pouco, balançou a cabeça e foi embora rezando por {@}"}},
		},
		{
			id:     "aposta-velho-baralho",
			title:  "O velho do baralho",
			rarity: legendaryEvent,
			intros: []string{
				"Um velhinho de chapéu e terno branco sentou do lado de {@} e embaralhou as cartas: 'uma última partida, tudo ou nada. Se ganhar, recupera tudo'",
				"Ninguém viu de onde ele veio. Um velho de dente de ouro chamou {@} pra mesa do canto: 'te devolvo o que perdeu hoje, se você me vencer'",
			},
			choices: []choice{
				{label: "🃏 Aceitar o tudo ou nada", risk: highRisk, endings: []ending{
					{weight: 35, texts: []string{"{@} venceu o velho na última carta! Ele riu, pagou e sumiu na fumaça do charuto"}, effects: []effect{earn(600, 1500)}},
					{weight: 50, texts: []string{"O velho virou um ás que ninguém tinha visto entrar no baralho. {@} perdeu de novo"}, effects: []effect{lose(15, 25)}},
					{weight: 10, texts: []string{"Na hora de pagar, o velho mostrou um rabo pontudo saindo do terno. Levou o dinheiro e deixou uma praga"}, effects: []effect{lose(10, 15), enchant("miseria")}},
					{weight: 5, texts: []string{"O velho cobrou a dívida do jeito antigo. Ninguém viu {@} sair daquela mesa"}, effects: []effect{perish()}},
				}},
				{label: "🎓 Pedir pra ele te ensinar", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O velho ensinou uns truques de verdade e {@} ganhou umas mãos ali mesmo"}, effects: []effect{earn(100, 400)}},
					{weight: 50, texts: []string{"A aula foi cara: o velho cobrou cada lição em fichas"}, effects: []effect{lose(5, 10)}},
				}},
				{label: "🚶 Recusar e ir embora", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} levantou e foi embora. Quando olhou pra trás, a mesa tava vazia"}},
					{weight: 40, texts: []string{"O velho sorriu: 'sabedoria vale mais que sorte'. E deixou um trevo no bolso de {@}"}, effects: []effect{enchant("sorte")}},
				}},
			},
			idle: ending{texts: []string{"O velho esperou, guardou o baralho e sumiu. Até hoje {@} não sabe se foi sonho"}},
		},
	},
}
