package economy

var allInEvents = eventBook{
	name:   "🫣 Vai tudo mesmo?",
	places: []string{allInPlace},
	scenarios: []scenario{
		{
			id:    "tudo-mae",
			title: "A ligação da mãe",
			intros: []string{
				"{@} já tava com o dedo no botão de apostar tudo quando o celular tocou. Era a mãe: 'oi, meu filho, tá fazendo o quê?'",
				"Bem na hora que {@} ia botar o saldo inteiro na mesa, chegou um áudio de três minutos da mãe começando com 'eu sei o que você tá fazendo'",
			},
			choices: []choice{
				{label: "📞 Contar a verdade", risk: lowRisk, endings: []ending{
					{weight: 50, texts: []string{"A mãe deu um sermão de meia hora e {@} desligou com a cabeça no lugar e o coração mais leve"}, effects: []effect{heal(10)}},
					{weight: 25, texts: []string{"A mãe ficou tão brava que rogou praga: 'vai perder tudo, você vai ver'. Praga de mãe pega"}, effects: []effect{enchant("azar")}},
					{weight: 25, texts: []string{"A mãe ficou com pena e mandou um Pix 'pra comprar um lanche e ir pra casa'"}, effects: []effect{earn(30, 90)}},
				}},
				{label: "🤥 Dizer que tá no trampo", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Colou. A mãe ainda falou que tem orgulho de {@}, o que doeu mais que qualquer aposta"}},
					{weight: 30, texts: []string{"A mãe ouviu o barulho do cassino no fundo e mandou a tia buscar {@} pela orelha"}, effects: []effect{injure(5, 15)}},
					{weight: 20, texts: []string{"Mentir pra mãe dá azar, todo mundo sabe. {@} sentiu na hora"}, effects: []effect{enchant("azar")}},
				}},
				{label: "📵 Recusar a ligação", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} recusou. Ela ligou mais quatorze vezes e acabou com o clima de qualquer aposta"}},
					{weight: 40, texts: []string{"A mãe contou tudo no grupo da família. Pra abafar o caso, {@} teve que pagar o almoço de domingo pra todo mundo"}, effects: []effect{lose(2, 5)}},
				}},
			},
			idle: ending{texts: []string{"{@} deixou o celular tocar até cair na caixa postal. A mãe mandou um 'tá bom então' que pesou mais que perder tudo"}},
		},
		{
			id:    "tudo-ombros",
			title: "Anjinho e diabinho",
			intros: []string{
				"Na hora de apostar tudo, apareceu um anjinho num ombro de {@} e um diabinho no outro, os dois falando ao mesmo tempo",
				"{@} fechou os olhos pra tomar coragem e, quando abriu, tinha um anjinho e um diabinho discutindo em cima dos ombros",
			},
			choices: []choice{
				{label: "😇 Ouvir o anjinho", risk: lowRisk, endings: []ending{
					{weight: 55, texts: []string{"O anjinho convenceu {@} a respirar fundo e ainda deixou uma bênção de brinde"}, effects: []effect{enchant("protecao")}},
					{weight: 45, texts: []string{"O anjinho falou tanto de juros compostos que {@} dormiu em pé e acordou descansado"}, effects: []effect{heal(5)}},
				}},
				{label: "😈 Ouvir o diabinho", risk: highRisk, endings: []ending{
					{weight: 35, texts: []string{"O diabinho soprou o número certo de uma rifa ali do lado e {@} levou o prêmio"}, effects: []effect{earn(100, 300)}},
					{weight: 45, texts: []string{"O diabinho convenceu {@} a torrar uma parte do saldo em bobagem antes mesmo de apostar"}, effects: []effect{lose(5, 12)}},
					{weight: 20, texts: []string{"Ouvir o diabinho tem preço: ele foi embora, mas deixou um encosto no lugar"}, effects: []effect{enchant("encosto")}},
				}},
				{label: "🤐 Mandar os dois calarem a boca", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Os dois ficaram ofendidos e sumiram. {@} ficou sozinho com a própria cabeça, o que é ainda mais perigoso"}},
					{weight: 50, texts: []string{"O diabinho deu um peteleco na orelha de {@} antes de ir embora. Doeu"}, effects: []effect{injure(3, 8)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} hesitava, o anjinho e o diabinho começaram a brigar entre eles e esqueceram o assunto"}},
		},
		{
			id:    "tudo-gerente",
			title: "Cortesia da casa",
			intros: []string{
				"O gerente viu {@} chegando com o saldo inteiro na mão e mandou estender o tapete vermelho: 'pra quem vai com tudo, a casa oferece um drink'",
				"Um segurança de terno levou {@} pra uma sala VIP: 'o gerente faz questão de conhecer quem vai apostar tudo hoje'",
			},
			choices: []choice{
				{label: "🍸 Aceitar o drink", risk: midRisk, endings: []ending{
					{weight: 40, texts: []string{"O drink era forte demais e {@} saiu cambaleando antes de conseguir apostar qualquer coisa"}, effects: []effect{injure(5, 12)}},
					{weight: 35, texts: []string{"Drink de graça e umas fichas de cortesia. {@} trocou as fichas na saída e foi pra casa no lucro"}, effects: []effect{earn(60, 180)}},
					{weight: 25, texts: []string{"Tinha alguma coisa no drink. {@} acordou na calçada com a carteira mais leve"}, effects: []effect{lose(6, 12)}},
				}},
				{label: "🙅 Recusar e ficar esperto", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} recusou. O gerente fechou a cara, mas pelo menos ninguém dopou ninguém"}},
					{weight: 30, texts: []string{"Ficar esperto valeu: {@} viu o crupiê trocando o baralho e guardou a lição"}, effects: []effect{enchant("sorte")}},
				}},
				{label: "🕵️ Denunciar as máquinas viciadas", risk: midRisk, needs: lawman, endings: []ending{
					{weight: 60, texts: []string{"{@} mostrou o distintivo, lacrou as máquinas e ainda levou a recompensa da operação"}, effects: []effect{earn(80, 220)}},
					{weight: 40, texts: []string{"O gerente era primo do delegado. {@} levou um empurrão dos seguranças e saiu sem nada"}, effects: []effect{injure(8, 18)}},
				}},
			},
			idle: ending{texts: []string{"{@} demorou tanto pra responder que o gerente foi oferecer o drink pra outro ricaço"}},
		},
		{
			id:    "tudo-cigana",
			title: "A cigana da porta",
			intros: []string{
				"Na porta, uma cigana segurou a mão de {@}: 'vejo que você vai apostar tudo hoje. Quer saber como termina?'",
				"Uma cigana cruzou o caminho de {@} e parou: 'não aposta tudo antes de me deixar ler essa mão, meu bem'",
			},
			choices: []choice{
				{label: "🔮 Pagar pra ler a mão", risk: lowRisk, price: cost{percent: 2, floor: 200}, endings: []ending{
					{weight: 45, texts: []string{"A cigana viu fortuna no caminho de {@} e ainda benzeu a mão"}, effects: []effect{enchant("prosperidade")}},
					{weight: 30, texts: []string{"A cigana viu um monte de números na mão de {@}. Era o boleto"}},
					{weight: 25, texts: []string{"A cigana jogou um trevo no bolso de {@} e mandou ir com fé"}, effects: []effect{enchant("sorte")}},
				}},
				{label: "🙄 Dizer que não acredita nisso", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"A cigana deu de ombros e foi embora. {@} ficou com uma pulga atrás da orelha"}},
					{weight: 40, texts: []string{"A cigana não gostou do desdém e jogou um olho gordo em {@} antes de sair"}, effects: []effect{enchant("olhogordo")}},
				}},
				{label: "🏃 Sair de fininho", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} saiu de fininho e a cigana nem percebeu"}},
					{weight: 30, texts: []string{"Na pressa de fugir da cigana, {@} tropeçou no degrau da porta"}, effects: []effect{injure(3, 8)}},
				}},
			},
			idle: ending{texts: []string{"{@} ficou parado e a cigana leu a mão de graça mesmo: 'você demora demais pra decidir as coisas'"}},
		},
		{
			id:     "tudo-velho",
			title:  "O velho de 87",
			rarity: rareEvent,
			intros: []string{
				"Um velho de chapéu, que jura ter perdido tudo nesse mesmo cassino em 1987, segurou o braço de {@}: 'senta aqui que eu te conto como foi'",
				"Antes de {@} apostar tudo, um velhinho na mesa do canto levantou a bengala: 'eu já fui igual a você, garoto. Paga um café que eu te conto'",
			},
			choices: []choice{
				{label: "👂 Ouvir a história toda", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"A história durou duas horas e terminou com o velho benzendo {@}: 'que você não acabe igual a mim'"}, effects: []effect{enchant("protecao"), heal(15)}},
					{weight: 40, texts: []string{"No fim da história o velho pediu dinheiro emprestado. {@} deu, porque ficou com pena"}, effects: []effect{lose(1, 3)}},
				}},
				{label: "☕ Pagar o café do velho", risk: lowRisk, price: cost{percent: 1, floor: 100}, endings: []ending{
					{weight: 65, texts: []string{"O velho agradeceu o café e jogou uma bênção de prosperidade em {@}"}, effects: []effect{enchant("prosperidade")}},
					{weight: 35, texts: []string{"O velho era um milionário excêntrico testando as pessoas. Gostou de {@} e deixou um envelope gordo"}, effects: []effect{earn(250, 600)}},
				}},
				{label: "🚶 Ignorar o velho", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} passou reto. Quando chegou na mesa, ela tinha acabado de fechar"}},
					{weight: 45, texts: []string{"O velho bateu a bengala no chão: 'vai acabar igual a mim'. E a praga pegou"}, effects: []effect{enchant("miseria")}},
				}},
			},
			idle: ending{texts: []string{"O velho dormiu no meio da própria história e {@} saiu na ponta dos pés"}},
		},
	},
}
