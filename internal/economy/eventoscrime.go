package economy

import "time"

var crimeEvents = eventBook{
	name:   "🦹 Crime",
	places: []string{"!roubar", "!furtar", "!sequestrar", "!matar", "!assalto", "!quadrilha", "!mercadonegro"},
	scenarios: []scenario{
		{
			id:     "crime-traficante-do-beco",
			title:  "O traficante do beco",
			places: []string{"!roubar", "!furtar", "!sequestrar", "!matar", "!assalto", "!quadrilha"},
			intros: []string{
				"Fugindo da cena do crime, {@} entrou num beco escuro e deu de cara com um traficante encostado na parede: \"Tá com cara de quem precisa de um gás, parceiro\"",
				"{@} cortou caminho por um beco pra despistar e um cara de boné aba reta saiu da sombra balançando um saquinho: \"A primeira é por conta da casa\"",
			},
			choices: []choice{
				{label: "❄️ Aceitar o teco", risk: highRisk, endings: []ending{
					{weight: 52, texts: []string{"O teco bateu forte e {@} saiu do beco se sentindo o dono da cidade", "{@} deu uma fungada, arregalou o olho e agora quer assaltar o Banco Central sozinho"}, effects: []effect{buzz("po")}},
					{weight: 25, texts: []string{"{@} passou mal na hora, vomitou atrás da lixeira e saiu do beco tropeçando"}, effects: []effect{injure(15, 30)}},
					{weight: 8, texts: []string{"O pó era batizado com veneno de rato. {@} caiu duro ali mesmo"}, effects: []effect{perish()}},
					{weight: 15, texts: []string{"Era um policial disfarçado. Ele tirou o boné, mostrou o distintivo e algemou {@} com um sorrisinho"}, effects: []effect{nabbed()}},
				}},
				{label: "💸 Comprar o pacote inteiro", risk: midRisk, price: cost{percent: 8, floor: 300}, endings: []ending{
					{weight: 60, texts: []string{"{@} pagou, enfiou o pacote na meia e saiu do beco assobiando"}, effects: []effect{gain(2, "baseado", "lanca-perfume"), gain(1, "po", "bala", "rebite")}},
					{weight: 30, texts: []string{"Na hora da troca, uma viatura apareceu no começo do beco"}, effects: []effect{gain(2, "baseado", "lanca-perfume"), gain(1, "po", "bala", "rebite")}, next: "crime-traficante-viatura"},
					{weight: 10, texts: []string{"Era flagrante armado. A polícia saiu de trás do muro e levou {@} junto com o pacote"}, effects: []effect{nabbed()}},
				}},
				{label: "🔪 Encarar o traficante", risk: highRisk, needs: armedUp, endings: []ending{
					{weight: 50, texts: []string{"{@} mostrou o ferro, o traficante saiu correndo e largou a mochila pra trás"}, effects: []effect{gain(2, "po", "bala"), earn(60, 150)}},
					{weight: 35, texts: []string{"Ele também tava armado. Os dois saíram no tapa e {@} levou a pior"}, effects: []effect{injure(20, 45)}},
					{weight: 8, texts: []string{"O tiroteio foi feio e {@} não saiu vivo do beco"}, effects: []effect{perish()}},
					{weight: 7, texts: []string{"O barulho chamou a polícia e {@} foi o único que não conseguiu correr"}, effects: []effect{nabbed()}},
				}},
				{label: "🏃 Sair vazado", risk: lowRisk, endings: []ending{
					{weight: 80, texts: []string{"{@} recusou com educação e saiu de fininho, olhando pra trás a cada passo"}},
					{weight: 20, texts: []string{"{@} saiu tão rápido que deixou um pé do chinelo no beco. Prioridades"}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, o traficante perdeu a paciência e foi oferecer pra outro"}},
		},
		{
			id:    "crime-traficante-viatura",
			title: "A sirene na esquina",
			stage: true,
			intros: []string{
				"Com o pacote ainda quente no bolso, {@} ouviu uma sirene. A viatura vem devagarinho, com o farol apontado pro beco",
			},
			choices: []choice{
				{label: "🕳️ Jogar o pacote no bueiro", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} jogou tudo no bueiro e passou pela viatura com cara de santo"}, effects: []effect{forfeit("baseado", 2), forfeit("lanca-perfume", 2), forfeit("po", 1), forfeit("bala", 1), forfeit("rebite", 1)}},
				}},
				{label: "🏃 Correr com o pacote", risk: midRisk, endings: []ending{
					{weight: 55, favor: func(p Profession) float64 { return p.escapeBonus }, texts: []string{"{@} pulou o muro do cemitério e sumiu entre os túmulos com o pacote inteirinho"}},
					{weight: 45, texts: []string{"O policial era maratonista. Alcançou {@} em dois quarteirões"}, effects: []effect{nabbed()}},
				}},
				{label: "😮 Engolir o pacote", risk: highRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} engoliu tudo de uma vez. A polícia revistou, não achou nada, e o efeito bateu na hora"}, effects: []effect{buzz("po"), injure(10, 25)}},
					{weight: 32, texts: []string{"O saquinho estourou no estômago e {@} foi parar no pronto-socorro passando muito mal"}, effects: []effect{injure(40, 70)}},
					{weight: 8, texts: []string{"Overdose. {@} não aguentou"}, effects: []effect{perish()}},
				}},
			},
			idle: ending{texts: []string{"{@} congelou no lugar e a viatura passou reto. Dessa vez deu sorte"}},
		},
		{
			id:     "crime-roubar-contando-grana",
			title:  "Contando a grana na calçada",
			places: []string{"!roubar"},
			when:   wentWell,
			intros: []string{
				"{@} parou debaixo de um poste pra contar a grana que tirou de {alvo} e só então viu a viatura estacionada do outro lado da rua",
				"Ainda com o dinheiro de {alvo} na mão, {@} virou a esquina e deu de cara com dois PMs comendo pastel",
			},
			choices: []choice{
				{label: "🏃 Sair correndo", risk: midRisk, endings: []ending{
					{weight: 60, favor: func(p Profession) float64 { return p.escapeBonus }, texts: []string{"{@} pulou dois muros, atravessou um quintal com cachorro e sumiu na quebrada"}},
					{weight: 40, texts: []string{"{@} tropeçou num carrinho de pipoca e caiu bem no pé do policial"}, effects: []effect{nabbed()}},
				}},
				{label: "🧍 Fingir que é turista", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} pediu informação de como chegar no Cristo Redentor e o policial ainda desenhou um mapa"}},
					{weight: 45, texts: []string{"O sotaque de gringo não convenceu ninguém. O PM anotou a cara de {@} no caderninho"}, effects: []effect{stain()}},
				}},
				{label: "💸 Molhar a mão do guarda", risk: midRisk, price: cost{percent: 10, floor: 200}, endings: []ending{
					{weight: 80, texts: []string{"O guarda contou as notas devagar, guardou no colete e mandou {@} ir com Deus"}},
					{weight: 20, texts: []string{"Era a corregedoria fazendo teste. Propina pra polícia dá cadeia, sabia?"}, effects: []effect{nabbed()}},
				}},
				{label: "⛪ Entrar na igreja", risk: lowRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} sentou no último banco, cantou dois hinos e saiu quando a viatura foi embora"}},
					{weight: 15, texts: []string{"O pastor impôs a mão na testa de {@} e profetizou: \"grade nenhuma vai te segurar, irmão\". A viatura foi embora"}, effects: []effect{enchant("libertacao")}},
					{weight: 30, texts: []string{"O pastor passou a sacolinha bem na frente de {@} e não tinha como negar"}, effects: []effect{lose(2, 4)}},
				}},
			},
		},
		{
			id:     "crime-roubar-status-chorando",
			title:  "O status chorando",
			places: []string{"!roubar"},
			when:   wentWell,
			intros: []string{
				"{alvo} postou um status chorando: o dinheiro que {@} levou era do aluguel, e o dono da casa é o pastor da igreja",
				"{@} viu {alvo} contando no grupo da família que perdeu o dinheiro do gás. A tia já mandou até corrente de oração",
			},
			choices: []choice{
				{label: "🤝 Devolver uma parte escondido", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} deixou um envelope debaixo da porta de {alvo} sem assinar. A vó de {alvo} rezou por quem devolveu"}, effects: []effect{refund(5, 10), enchant("protecao")}},
					{weight: 40, texts: []string{"{@} devolveu um pouco e se sentiu tão bem que até dormiu cedo"}, effects: []effect{refund(5, 10)}},
				}},
				{label: "😎 Comentar chora mais", risk: midRisk, endings: []ending{
					{weight: 45, texts: []string{"O comentário viralizou e a inveja do grupo inteiro caiu em cima de {@}"}, effects: []effect{enchant("olhogordo")}},
					{weight: 22, texts: []string{"A tia de {alvo} é benzedeira e não gostou nada do comentário"}, effects: []effect{enchant("azar")}},
					{weight: 25, texts: []string{"Ninguém viu o comentário. {@} saiu ileso e ainda deu risada"}},
					{weight: 8, texts: []string{"{alvo} printou o comentário e levou no terreiro. Agora uma parte do que {@} ganhar volta pra quem foi roubado"}, effects: []effect{grudge("dizimo")}},
				}},
				{label: "🙈 Fingir que não viu", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} deixou o celular de lado e foi dormir. Amanhã é outro dia"}},
				}},
			},
		},
		{
			id:     "crime-roubar-vitima-reage",
			title:  "A revanche do chinelo",
			places: []string{"!roubar"},
			when:   wentWrong,
			intros: []string{
				"{alvo} correu atrás de {@} com uma Havaianas na mão e a vizinhança inteira acordou pra assistir",
				"Depois do roubo que deu errado, {alvo} chamou os primos e agora tem cinco marmanjos olhando pra {@}",
			},
			choices: []choice{
				{label: "🥊 Sair no braço", risk: highRisk, endings: []ending{
					{weight: 45, favor: func(p Profession) float64 { return p.guard }, texts: []string{"{@} desviou de três chineladas, revidou e ainda achou umas notas que caíram do bolso de {alvo}"}, effects: []effect{earn(30, 80)}},
					{weight: 40, texts: []string{"{@} levou uma surra de chinelo que vai ficar marcada até o Natal"}, effects: []effect{injure(20, 40)}},
					{weight: 10, texts: []string{"A vizinha filmou tudo e mandou pro 190"}, effects: []effect{nabbed()}},
					{weight: 5, texts: []string{"A chinelada pegou bem na têmpora. {@} apagou e não acordou mais"}, effects: []effect{perish()}},
				}},
				{label: "🏃 Correr igual o vento", risk: midRisk, endings: []ending{
					{weight: 65, favor: func(p Profession) float64 { return p.escapeBonus }, texts: []string{"{@} correu tanto que chegou no bairro vizinho antes de {alvo} calçar o outro pé do chinelo"}},
					{weight: 35, texts: []string{"{@} tropeçou no meio-fio e ralou a cara inteira no asfalto"}, effects: []effect{injure(5, 15)}},
				}},
				{label: "🙏 Pedir desculpa de joelhos", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{alvo} aceitou as desculpas, mas fez {@} lavar a louça da família inteira"}, effects: []effect{delay(10 * time.Minute)}},
					{weight: 30, texts: []string{"{@} pediu desculpa e ainda pagou um açaí pra fazer as pazes"}, effects: []effect{lose(2, 4)}},
				}},
				{label: "🔪 Mostrar o ferro", risk: midRisk, needs: armedUp, endings: []ending{
					{weight: 60, texts: []string{"{alvo} viu o ferro, recuou devagar e {@} saiu andando como se nada tivesse acontecido"}, effects: []effect{stain()}},
					{weight: 40, texts: []string{"A vizinha filmou {@} com a arma na mão e mandou direto pra polícia"}, effects: []effect{nabbed()}},
				}},
			},
		},
		{
			id:     "crime-roubar-parte-do-tonhao",
			title:  "A parte do Tonhão",
			places: []string{"!roubar"},
			when:   wentWell,
			intros: []string{
				"O Tonhão, que segurou a moto na fuga, chegou cobrando metade do que {@} tirou de {alvo}. Ele tá estalando os dedos",
				"{@} mal guardou o dinheiro de {alvo} e o Tonhão já apareceu: \"Combinado é combinado, cadê a minha parte?\"",
			},
			choices: []choice{
				{label: "🤝 Dividir certinho", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} pagou a parte do Tonhão até o último centavo. Ele saiu dizendo que nunca vai esquecer isso"}, effects: []effect{lose(3, 5), plant("crime-tonhao-volta", 6*time.Hour)}},
				}},
				{label: "😏 Dar o cano", risk: midRisk, endings: []ending{
					{weight: 45, texts: []string{"O Tonhão contou as notas, achou pouco, mas não falou nada. Ainda"}, effects: []effect{plant("crime-tonhao-vinganca", 4*time.Hour)}},
					{weight: 35, texts: []string{"O Tonhão nem percebeu que levou nota de dois no lugar de cinquenta"}, effects: []effect{earn(20, 50)}},
					{weight: 20, texts: []string{"O Tonhão percebeu na hora e deu um tapa na nuca de {@}"}, effects: []effect{injure(10, 20)}},
				}},
				{label: "🍻 Pagar uma rodada no bar", risk: lowRisk, price: cost{percent: 3, floor: 50}, endings: []ending{
					{weight: 75, texts: []string{"Uma cerveja virou três, o Tonhão chorou falando da ex e jurou amizade eterna"}, effects: []effect{enchant("sorte")}},
					{weight: 25, texts: []string{"A rodada virou um churrasco inteiro, e quem pagou foi {@}"}, effects: []effect{lose(2, 5)}},
				}},
			},
		},
		{
			id:     "crime-tonhao-volta",
			title:  "O presente do Tonhão",
			stage:  true,
			places: []string{"!pescar", "!minerar", "!vasculhar", "!catar", "!trabalhar", "!roubar", "!furtar", "!sequestrar", "!matar", "!assalto", "!quadrilha", "!mercadonegro"},
			intros: []string{
				"O Tonhão apareceu do nada com uma sacola na mão: \"Você foi honesto comigo naquela divisão. Trouxe um agrado\"",
			},
			choices: []choice{
				{label: "🎁 Aceitar o presente", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"Dentro da sacola tinha um presente de verdade, com laço e tudo"}, effects: []effect{gain(1, "pe-de-cabra", "mascara", "kit-disfarce")}},
					{weight: 30, texts: []string{"O presente era um pudim que a mãe do Tonhão fez. Melhor pudim da vida de {@}"}, effects: []effect{gain(1, "pudim"), heal(15)}},
				}},
				{label: "🤔 Desconfiar e recusar", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} recusou e o Tonhão ficou meio chateado, mas entendeu"}},
				}},
				{label: "🦹 Chamar ele pra um serviço", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"Os dois fizeram um bico juntos e o serviço rendeu bem"}, effects: []effect{earn(80, 200), stain()}},
					{weight: 40, texts: []string{"O serviço deu ruim e a polícia apareceu antes da hora"}, effects: []effect{nabbed()}},
				}},
			},
			idle: ending{texts: []string{"O Tonhão cansou de esperar resposta e deixou a sacola na porta mesmo. Era um pudim"}, effects: []effect{gain(1, "pudim")}},
		},
		{
			id:     "crime-tonhao-vinganca",
			title:  "O Tonhão não esqueceu",
			stage:  true,
			places: []string{"!pescar", "!minerar", "!vasculhar", "!catar", "!trabalhar", "!roubar", "!furtar", "!sequestrar", "!matar", "!assalto", "!quadrilha", "!mercadonegro"},
			intros: []string{
				"O Tonhão descobriu o cano que levou naquela divisão e agora tá na frente de {@} com mais dois amigos e um taco de beisebol",
			},
			choices: []choice{
				{label: "💸 Pagar o que deve com juros", risk: lowRisk, price: cost{percent: 10, floor: 200}, endings: []ending{
					{weight: 100, texts: []string{"{@} pagou com juros e o Tonhão foi embora rindo"}},
				}},
				{label: "🏃 Correr pro ponto de ônibus", risk: midRisk, endings: []ending{
					{weight: 55, favor: func(p Profession) float64 { return p.escapeBonus }, texts: []string{"{@} correu até o ponto e entrou no primeiro ônibus que passou, sem nem ver o destino"}},
					{weight: 45, texts: []string{"O taco de beisebol alcançou {@} antes do ônibus"}, effects: []effect{injure(20, 35)}},
				}},
				{label: "🥊 Encarar os três", risk: highRisk, endings: []ending{
					{weight: 35, favor: func(p Profession) float64 { return p.guard }, texts: []string{"{@} derrubou o Tonhão com um golpe só e os amigos dele fugiram deixando a carteira cair"}, effects: []effect{earn(40, 100)}},
					{weight: 55, texts: []string{"Três contra um não dá. {@} apanhou até o Tonhão cansar"}, effects: []effect{injure(30, 50)}},
					{weight: 10, texts: []string{"Uma tacada pegou feio e {@} não levantou mais"}, effects: []effect{perish()}},
				}},
				{label: "😭 Chorar e implorar", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O Tonhão ficou com dó, mas levou um dinheirinho de {@} pela humilhação"}, effects: []effect{lose(3, 6)}},
					{weight: 30, texts: []string{"{@} chorou tanto que o Tonhão chorou junto e os dois fizeram as pazes"}},
				}},
			},
			idle: ending{texts: []string{"{@} fingiu que não viu e levou um tapa na orelha pra aprender"}, effects: []effect{injure(5, 10)}},
		},
		{
			id:     "crime-roubar-camera-padaria",
			title:  "A câmera da padaria",
			places: []string{"!roubar"},
			intros: []string{
				"{@} percebeu tarde demais que tinha uma câmera da padaria apontada bem pro lugar onde rolou o B.O. com {alvo}",
				"O dono da padaria da esquina tá rebobinando a gravação da câmera com um sorriso esquisito, e {@} aparece no vídeo",
			},
			choices: []choice{
				{label: "🎭 Voltar disfarçado e apagar", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} entrou de peruca, pediu um pão na chapa e apagou a gravação enquanto o padeiro virava o pão"}, effects: []effect{refresh(), gain(1, "pao-de-queijo")}},
					{weight: 45, texts: []string{"A peruca caiu bem na hora e o padeiro chamou a polícia"}, effects: []effect{nabbed()}},
				}},
				{label: "💸 Comprar o silêncio do padeiro", risk: midRisk, price: cost{percent: 6, floor: 150}, endings: []ending{
					{weight: 85, texts: []string{"O padeiro apagou o vídeo e ainda deu um sonho de brinde"}},
					{weight: 15, texts: []string{"O padeiro pegou o dinheiro e mandou o vídeo pro grupo do bairro assim mesmo"}, effects: []effect{stain()}},
				}},
				{label: "🤷 Torcer pra ser de enfeite", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"Era câmera de enfeite mesmo, daquelas de vinte reais no camelô. Ufa"}},
					{weight: 40, texts: []string{"Era de verdade, e o vídeo de {@} já tá no grupo do condomínio com 300 visualizações"}, effects: []effect{stain(), enchant("olhogordo")}},
				}},
				{label: "💻 Hackear o gravador", risk: lowRisk, needs: isJob("hacker"), endings: []ending{
					{weight: 90, texts: []string{"{@} invadiu o gravador pelo wifi da padaria e trocou o vídeo por um de gatinho"}, effects: []effect{refresh()}},
					{weight: 10, texts: []string{"O wifi da padaria caiu no meio do ataque. Pelo menos ninguém percebeu nada"}},
				}},
			},
		},
		{
			id:     "crime-roubar-mala-do-vereador",
			title:  "A mala do vereador",
			rarity: legendaryEvent,
			places: []string{"!roubar"},
			when:   wentWell,
			intros: []string{
				"No meio do que {@} tirou de {alvo} tinha uma mala de couro com um adesivo: PROPRIEDADE DO VEREADOR. NÃO ABRIR",
				"{alvo} tava carregando, sem saber, uma mala que o vereador mandou guardar. Agora ela tá com {@}",
			},
			choices: []choice{
				{label: "💼 Ficar com a mala", risk: highRisk, endings: []ending{
					{weight: 55, texts: []string{"A mala tava cheia de dinheiro de campanha. {@} nunca viu tanta nota junta"}, effects: []effect{gain(1, "mala-de-dinheiro"), stain()}},
					{weight: 30, texts: []string{"Os seguranças do vereador seguiram {@} e deram um susto daqueles"}, effects: []effect{injure(25, 45)}},
					{weight: 15, texts: []string{"A Polícia Federal tava investigando a mala e levou {@} junto"}, effects: []effect{nabbed()}},
				}},
				{label: "📰 Entregar pro jornal", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"A reportagem saiu no jornal da noite e {@} ganhou uma recompensa de um empresário anônimo"}, effects: []effect{earn(300, 600), enchant("justica")}},
					{weight: 40, texts: []string{"O jornal era do vereador. A mala sumiu e {@} ficou marcado"}, effects: []effect{enchant("azar")}},
				}},
				{label: "🚮 Largar a mala no rio", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} jogou a mala no rio e fingiu que nunca viu nada"}},
				}},
			},
		},
		{
			id:     "crime-furtar-foto-de-familia",
			title:  "A foto da família",
			places: []string{"!furtar"},
			when:   wentWell,
			intros: []string{
				"No meio do que {@} furtou de {alvo} tinha uma foto 3x4 da vó com uma dedicatória: \"pro meu netinho que eu amo\"",
				"{@} achou no furto um bilhete escrito à mão por {alvo}: \"remédio da mãe, não esquecer\"",
			},
			choices: []choice{
				{label: "📮 Devolver pelo correio", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} mandou de volta sem remetente. A vó de {alvo} fez uma oração pra quem devolveu"}, effects: []effect{enchant("protecao")}},
					{weight: 40, texts: []string{"{@} devolveu e ganhou de brinde uma paz de espírito que não tem preço"}, effects: []effect{heal(10)}},
				}},
				{label: "🛒 Vender no camelô", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"O camelô pagou bem e nem perguntou de onde veio"}, effects: []effect{earn(20, 60)}},
					{weight: 40, texts: []string{"O camelô era primo de {alvo} e reconheceu tudo na hora"}, effects: []effect{stain(), injure(5, 15)}},
				}},
				{label: "🗄️ Guardar de lembrança", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} guardou na gaveta e toda noite sente que alguém tá olhando"}, effects: []effect{enchant("encosto")}},
					{weight: 40, texts: []string{"{@} guardou e esqueceu que tinha"}},
				}},
			},
		},
		{
			id:     "crime-furtar-caramelo",
			title:  "O vira-lata caramelo",
			places: []string{"!furtar"},
			intros: []string{
				"Saindo do furto, {@} percebeu que um vira-lata caramelo tá seguindo de perto, abanando o rabo e olhando pro bolso",
				"Um cachorro caramelo viu {@} mexendo nas coisas de {alvo} e agora tá sentado na frente, esperando alguma coisa em troca do silêncio",
			},
			choices: []choice{
				{label: "🥟 Dar a coxinha pra ele", risk: lowRisk, uses: "coxinha", endings: []ending{
					{weight: 100, texts: []string{"O caramelo comeu a coxinha, lambeu a mão de {@} e virou guarda-costas pelo resto do dia"}, effects: []effect{enchant("protecao")}},
				}},
				{label: "🐕 Fazer carinho", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O caramelo deitou de barriga pra cima e {@} ganhou um amigo pra vida"}, effects: []effect{enchant("sorte")}},
					{weight: 30, texts: []string{"O caramelo achou que era brincadeira e mordeu a mão de {@} de leve"}, effects: []effect{injure(5, 10)}},
				}},
				{label: "🏃 Correr dele", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O caramelo achou que era corrida e ganhou. Ainda mordeu a canela de {@}"}, effects: []effect{injure(5, 15)}},
					{weight: 50, texts: []string{"{@} correu e o caramelo ficou pra trás, latindo a história pra rua inteira"}, effects: []effect{stain()}},
				}},
			},
		},
		{
			id:     "crime-furtar-entalado",
			title:  "Entalado na janela",
			places: []string{"!furtar"},
			when:   wentWrong,
			intros: []string{
				"Tentando sair pela janela da casa de {alvo}, {@} ficou entalado pela cintura. Metade pra dentro, metade pra fora",
				"{@} tentou fugir pelo basculante do banheiro de {alvo} e agora tá preso igual rolha de garrafa",
			},
			choices: []choice{
				{label: "💪 Forçar até sair", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"Com um puxão, {@} se soltou e caiu de cara no canteiro de cebolinha"}, effects: []effect{injure(5, 15)}},
					{weight: 40, texts: []string{"A janela quebrou junto e {@} saiu todo cortado"}, effects: []effect{injure(15, 30)}},
				}},
				{label: "📞 Chamar o vizinho pra ajudar", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O vizinho ajudou e ainda achou que {@} era o técnico da internet"}},
					{weight: 50, texts: []string{"O vizinho ajudou, tirou uma foto e mandou pra polícia logo depois"}, effects: []effect{stain(), delay(15 * time.Minute)}},
				}},
				{label: "⏳ Esperar a noite cair", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} esperou entalado até escurecer. Saiu com dor nas costas e o próximo furto atrasado"}, effects: []effect{delay(20 * time.Minute)}},
				}},
				{label: "🔧 Usar o pé de cabra", risk: lowRisk, uses: "pe-de-cabra", endings: []ending{
					{weight: 100, texts: []string{"{@} alavancou a janela com o pé de cabra e saiu limpinho, mas a ferramenta entortou de vez"}},
				}},
			},
		},
		{
			id:     "crime-furtar-receptador",
			title:  "O receptador do Paraguai",
			places: []string{"!furtar"},
			when:   wentWell,
			intros: []string{
				"Um receptador de camisa florida viu {@} com o furto de {alvo} e ofereceu um preço bom demais pra ser verdade",
				"\"Isso aí que você pegou de {alvo} eu levo agora, pagamento à vista\", disse um sujeito com sotaque de Ciudad del Este",
			},
			choices: []choice{
				{label: "💰 Vender na hora", risk: midRisk, endings: []ending{
					{weight: 65, texts: []string{"O receptador pagou em dinheiro vivo e sumiu numa moto sem placa"}, effects: []effect{earn(60, 150), stain()}},
					{weight: 35, texts: []string{"As notas que ele pagou eram de brinquedo. {@} só percebeu na padaria"}, effects: []effect{gain(3, "dinheiro-de-brinquedo")}},
				}},
				{label: "🤝 Pechinchar mais", risk: midRisk, endings: []ending{
					{weight: 45, texts: []string{"{@} chorou o preço e o receptador cedeu: pagou o dobro"}, effects: []effect{earn(120, 250), stain()}},
					{weight: 55, texts: []string{"O receptador se ofendeu, cuspiu no chão e foi embora"}},
				}},
				{label: "🙅 Recusar", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} recusou. Melhor não se meter com gente de camisa florida"}},
				}},
			},
		},
		{
			id:     "crime-furtar-bilhete",
			title:  "O bilhete no bolso",
			rarity: rareEvent,
			places: []string{"!furtar"},
			when:   wentWell,
			intros: []string{
				"No bolso de {alvo}, junto com o furto, veio um bilhete de loteria dobradinho. O sorteio foi ontem",
				"{@} achou um bilhete de loteria escondido no que furtou de {alvo}. Os números são as datas de aniversário da família toda",
			},
			choices: []choice{
				{label: "🎰 Conferir na lotérica", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Deu quina! A moça da lotérica gritou e {@} saiu de lá rico"}, effects: []effect{earn(400, 900)}},
					{weight: 50, texts: []string{"Não deu nada. Nem uma quadra. Só a vergonha na fila"}},
				}},
				{label: "🤝 Devolver o bilhete", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"{alvo} conferiu, ganhou uma quadra e ficou tão feliz que deu um agrado pra {@}"}, effects: []effect{earn(100, 250), enchant("sorte")}},
					{weight: 40, texts: []string{"{alvo} conferiu e não ganhou nada, mas agradeceu a honestidade"}, effects: []effect{enchant("protecao")}},
				}},
				{label: "🔥 Queimar o bilhete", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} queimou o bilhete pra não deixar prova. Nunca vai saber se era o premiado"}},
				}},
			},
		},
		{
			id:     "crime-furtar-vizinha-fofoqueira",
			title:  "A vizinha fofoqueira",
			places: []string{"!furtar"},
			intros: []string{
				"A vizinha de {alvo} viu tudo pela janela e já tá digitando no grupo do condomínio com o dedo tremendo de emoção",
				"Dona Neide, a fofoqueira da rua, cruzou com {@} carregando as coisas de {alvo} e arregalou o olho",
			},
			choices: []choice{
				{label: "🍰 Levar um bolo pra ela", risk: lowRisk, price: cost{percent: 2, floor: 60}, endings: []ending{
					{weight: 85, texts: []string{"Dona Neide aceitou o bolo, contou a vida dos outros por uma hora e esqueceu do furto"}},
					{weight: 15, texts: []string{"Dona Neide comeu o bolo e contou pro grupo do mesmo jeito"}, effects: []effect{stain()}},
				}},
				{label: "📱 Pegar o celular dela", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"{@} arrancou o celular da mão dela antes do áudio ir. E o celular ainda era bom"}, effects: []effect{gain(1, "nokia")}},
					{weight: 40, texts: []string{"Dona Neide é faixa preta de jiu-jitsu da terceira idade"}, effects: []effect{injure(20, 40)}},
					{weight: 20, texts: []string{"Dona Neide gritou tão alto que a viatura que passava parou"}, effects: []effect{nabbed()}},
				}},
				{label: "🥸 Fingir que é entregador", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"\"Entrega pra {alvo}\", disse {@}. Dona Neide acreditou e ainda quis assinar o recebimento"}},
					{weight: 40, texts: []string{"Dona Neide conhece todos os entregadores pelo nome. Gravou um áudio de 4 minutos"}, effects: []effect{stain()}},
				}},
				{label: "😂 Sair rindo", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} acenou pra Dona Neide e saiu rindo. Ela ia contar de qualquer jeito mesmo"}},
				}},
			},
		},
		{
			id:     "crime-sequestrar-refem-cantor",
			title:  "O refém cantor",
			places: []string{"!sequestrar"},
			when:   wentWell,
			intros: []string{
				"{alvo} não para de cantar no porta-malas. Já foi \"Detalhes\" três vezes e agora tá emendando \"Emoções\"",
				"Desde que foi sequestrado, {alvo} contou a história da vida inteira pra {@}, com direito a choro na parte do divórcio",
			},
			choices: []choice{
				{label: "🎧 Botar o fone no máximo", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} botou o fone e ouviu funk por duas horas. A cantoria passou batida"}},
				}},
				{label: "🌿 Oferecer um baseado pro refém", risk: lowRisk, uses: "baseado", endings: []ending{
					{weight: 100, texts: []string{"{alvo} fumou, riu sozinho e dormiu no porta-malas. Paz, finalmente"}, effects: []effect{heal(10)}},
				}},
				{label: "🎤 Cantar junto", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"Os dois fizeram um dueto tão bonito que {@} até se emocionou"}, effects: []effect{enchant("sorte")}},
					{weight: 45, texts: []string{"{@} desafinou tanto que o próprio refém pediu silêncio. Ficou uma dor de cabeça"}, effects: []effect{injure(5, 10)}},
				}},
				{label: "😤 Mandar calar a boca", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{alvo} ficou quietinho. Pelo menos por cinco minutos"}},
					{weight: 30, texts: []string{"{alvo} começou a rezar alto, e a reza pegou em quem não devia"}, effects: []effect{enchant("encosto")}},
					{weight: 10, texts: []string{"{alvo} rezou pedindo o resgate de volta, e o santo atendeu do jeito dele"}, effects: []effect{grudge("dizimo")}},
				}},
			},
		},
		{
			id:     "crime-sequestrar-pneu-furado",
			title:  "Pneu furado na blitz",
			places: []string{"!sequestrar"},
			when:   wentWell,
			intros: []string{
				"No meio da fuga, com {alvo} amarrado no banco de trás, o pneu do carro de {@} furou bem em frente a um posto da PRF",
				"O carro com {alvo} dentro começou a fazer barulho de pneu furado justo quando {@} passava por uma blitz",
			},
			choices: []choice{
				{label: "🔧 Trocar o pneu rapidinho", risk: midRisk, endings: []ending{
					{weight: 60, favor: func(p Profession) float64 { return p.escapeBonus }, texts: []string{"{@} trocou o pneu em 40 segundos, tempo de Fórmula 1, e saiu cantando pneu"}},
					{weight: 40, texts: []string{"O macaco quebrou e o guarda veio oferecer ajuda. Viu o refém"}, effects: []effect{nabbed()}},
				}},
				{label: "🙋 Pedir ajuda pro guarda", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"O guarda trocou o pneu, ouviu o barulho no porta-malas e achou que era o rádio"}, effects: []effect{enchant("sorte")}},
					{weight: 55, texts: []string{"O guarda abriu o porta-malas pra pegar o estepe"}, effects: []effect{nabbed()}},
				}},
				{label: "🛞 Seguir no aro", risk: midRisk, endings: []ending{
					{weight: 65, texts: []string{"{@} seguiu soltando faísca no asfalto até o cativeiro, mas chegou"}, effects: []effect{delay(15 * time.Minute)}},
					{weight: 35, texts: []string{"O carro rodou numa curva e {@} bateu a cabeça no volante"}, effects: []effect{injure(15, 30)}},
				}},
				{label: "📟 Chamar reforço no rádio", risk: lowRisk, needs: holds("radio-comunicador"), endings: []ending{
					{weight: 100, texts: []string{"{@} chamou um comparsa pelo rádio, que chegou com outro carro em 5 minutos"}},
				}},
			},
		},
		{
			id:     "crime-sequestrar-resgate-vr",
			title:  "O resgate em vale-refeição",
			places: []string{"!sequestrar"},
			when:   wentWell,
			intros: []string{
				"A família de {alvo} mandou um áudio de 7 minutos oferecendo o resgate. Só que é em vale-refeição",
				"O cunhado de {alvo} ligou pra negociar e jura que só tem vale-refeição e um Uno 2004 com a porta amassada",
			},
			choices: []choice{
				{label: "🍽️ Aceitar o vale-refeição", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} aceitou. Vai comer de graça no restaurante por quilo até o fim do mês"}, effects: []effect{gain(3, "vale-refeicao")}},
				}},
				{label: "📲 Exigir Pix", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"Depois de muito choro, a família fez o Pix"}, effects: []effect{earn(80, 180)}},
					{weight: 45, texts: []string{"O Pix só caiu depois que o banco pediu o CPF de {@}. Agora tá tudo registrado"}, effects: []effect{stain(), lose(2, 5)}},
				}},
				{label: "📉 Fazer um desconto", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} fez um preço camarada e a família pagou na hora"}, effects: []effect{earn(30, 70)}},
				}},
				{label: "🚗 Aceitar o Uno", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O Uno pegou no tranco e ainda tinha um troco esquecido no porta-luvas"}, effects: []effect{earn(50, 120)}},
					{weight: 50, texts: []string{"O Uno não pegou nem empurrando. {@} ganhou uma carroça de lata e perdeu a tarde"}, effects: []effect{delay(10 * time.Minute)}},
				}},
			},
		},
		{
			id:     "crime-sequestrar-senhora-do-rodo",
			title:  "A senhora do rodo",
			places: []string{"!sequestrar"},
			when:   wentWrong,
			intros: []string{
				"{alvo} gritou tão alto que acordou o quarteirão. Agora tem uma senhora de bobes apontando um rodo pra {@}",
				"O sequestro de {alvo} deu errado e a tia que vende geladinho na esquina saiu correndo atrás de {@} com um rodo molhado",
			},
			choices: []choice{
				{label: "🥋 Encarar a senhora", risk: highRisk, endings: []ending{
					{weight: 25, favor: func(p Profession) float64 { return p.guard }, texts: []string{"{@} desviou do rodo com um mortal pra trás e ainda achou uma nota caída no chão"}, effects: []effect{earn(20, 50)}},
					{weight: 60, texts: []string{"A senhora do rodo foi campeã de vôlei de praia. {@} apanhou bonito"}, effects: []effect{injure(25, 45)}},
					{weight: 10, texts: []string{"Um vizinho chamou a polícia no meio da surra"}, effects: []effect{nabbed()}},
					{weight: 5, texts: []string{"O cabo do rodo acertou a nuca. {@} não acordou mais"}, effects: []effect{perish()}},
				}},
				{label: "🏃 Fugir pelo telhado", risk: midRisk, endings: []ending{
					{weight: 60, favor: func(p Profession) float64 { return p.escapeBonus }, texts: []string{"{@} subiu no telhado igual gato e sumiu pela laje do vizinho"}},
					{weight: 40, texts: []string{"A telha quebrou e {@} caiu dentro da casa de alguém no meio do almoço de domingo"}, effects: []effect{injure(10, 25), stain()}},
				}},
				{label: "🤪 Se fazer de doido", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} começou a latir e a senhora ficou com tanto dó que deixou ir embora"}, effects: []effect{delay(10 * time.Minute)}},
					{weight: 30, texts: []string{"{@} se fez de doido tão bem que a senhora chamou a ambulância"}, effects: []effect{delay(20 * time.Minute)}},
				}},
			},
		},
		{
			id:     "crime-sequestrar-sindrome",
			title:  "Síndrome de Estocolmo",
			places: []string{"!sequestrar"},
			when:   wentWell,
			intros: []string{
				"Depois de uma hora juntos no cativeiro, {alvo} começou a achar {@} até gente boa e propôs uma sociedade no crime",
				"{alvo} olhou pra {@} e disse: \"Sabe que você é bom nisso? Tenho um plano pra nós dois\"",
			},
			choices: []choice{
				{label: "🤝 Ouvir a proposta", risk: midRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} puxou uma cadeira e {alvo} começou a desenhar o plano num guardanapo"}, next: "crime-sequestrar-sociedade"},
				}},
				{label: "🙄 Ignorar o papo", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} ignorou e foi esquentar um miojo. Refém é refém"}},
				}},
				{label: "🧹 Mandar varrer o cativeiro", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{alvo} varreu tudo e ainda passou pano. O cativeiro nunca esteve tão limpo"}, effects: []effect{heal(5)}},
					{weight: 30, texts: []string{"{alvo} varreu a poeira bem na cara de {@}, de propósito"}, effects: []effect{injure(1, 5)}},
				}},
			},
		},
		{
			id:     "crime-sequestrar-sociedade",
			title:  "A sociedade com o refém",
			stage:  true,
			places: []string{"!sequestrar"},
			intros: []string{
				"O plano de {alvo} é dar um golpe no próprio cunhado, que é agiota. Mas quem vai dar a cara é {@}",
			},
			choices: []choice{
				{label: "💼 Topar o golpe", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"O golpe deu certo e o cunhado agiota nem percebeu"}, effects: []effect{earn(150, 300), stain()}},
					{weight: 45, texts: []string{"O cunhado agiota era mais esperto e mandou os capangas"}, effects: []effect{injure(15, 30)}},
				}},
				{label: "🐍 Trair o refém", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} pegou o plano, deu o golpe sozinho e ficou com tudo. A consciência pesou depois"}, effects: []effect{earn(200, 400), enchant("encosto")}},
					{weight: 40, texts: []string{"{alvo} desconfiou e contou tudo pro cunhado. Os capangas vieram atrás de {@}"}, effects: []effect{injure(25, 45)}},
					{weight: 10, texts: []string{"O cunhado agiota não perdoa. Mandou dar um fim em {@}"}, effects: []effect{perish()}},
				}},
				{label: "🚪 Desistir", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} achou o plano bom demais pra ser verdade e voltou pro miojo"}},
				}},
			},
			idle: ending{texts: []string{"{@} pensou tanto que {alvo} dormiu no meio da explicação"}},
		},
		{
			id:     "crime-sequestrar-cativeiro-vizinho",
			title:  "O vizinho do cativeiro",
			places: []string{"!sequestrar"},
			when:   wentWell,
			intros: []string{
				"O cativeiro que {@} alugou fica colado numa igreja, e o culto com banda e bateria começa em 10 minutos",
				"Só depois de trancar {alvo} no cativeiro {@} descobriu que o vizinho de parede é um terreiro em dia de gira",
			},
			choices: []choice{
				{label: "🙏 Ir lá disfarçado", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} cantou, dançou e saiu de lá abençoado, com a certeza de que cadeia nenhuma segura"}, effects: []effect{enchant("libertacao")}},
					{weight: 40, texts: []string{"Na hora do apelo, {@} quase confessou o sequestro no microfone"}, effects: []effect{stain()}},
				}},
				{label: "📢 Torcer pro barulho abafar", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Os gritos de {alvo} sumiram no meio da cantoria. Ninguém ouviu nada"}},
					{weight: 50, texts: []string{"Bem na hora do silêncio da oração, {alvo} gritou socorro. Todo mundo ouviu"}, effects: []effect{nabbed()}},
				}},
				{label: "🧳 Mudar de cativeiro", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} levou {alvo} pra outro lugar, mas perdeu um tempão na mudança"}, effects: []effect{delay(20 * time.Minute)}},
				}},
			},
		},
		{
			id:     "crime-matar-testemunha",
			title:  "A testemunha do picolé",
			places: []string{"!matar"},
			when:   wentWell,
			intros: []string{
				"Um menino de bicicleta viu tudo que aconteceu com {alvo} e tá encarando {@}, chupando um picolé de uva bem devagar",
				"Tinha uma testemunha do que rolou com {alvo}: um menino de uns 10 anos, com picolé na mão e cara de quem vai contar pra mãe",
			},
			choices: []choice{
				{label: "🍦 Comprar outro picolé pra ele", risk: lowRisk, price: cost{percent: 1, floor: 20}, endings: []ending{
					{weight: 80, texts: []string{"O menino aceitou o picolé, piscou o olho e foi embora pedalando"}},
					{weight: 20, texts: []string{"O menino aceitou, mas contou pra mãe mesmo assim. A mãe é escrivã"}, effects: []effect{stain()}},
				}},
				{label: "😠 Fazer cara feia", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"O menino se assustou e saiu correndo. Esqueceu até do picolé"}},
					{weight: 45, texts: []string{"O menino fez cara feia de volta e contou pro pai, que é delegado"}, effects: []effect{stain(), delay(15 * time.Minute)}},
				}},
				{label: "🤷 Ignorar e ir embora", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O menino só queria mesmo era terminar o picolé em paz"}},
					{weight: 50, texts: []string{"O menino desenhou o rosto de {@} e entregou na delegacia. Ficou idêntico"}, effects: []effect{stain()}},
				}},
			},
		},
		{
			id:     "crime-matar-homem-de-terno",
			title:  "O homem de terno",
			places: []string{"!matar"},
			when:   wentWell,
			intros: []string{
				"Um homem de terno preto aplaudiu {@} de dentro de um carro de vidro fumê: \"Trabalho limpo. Tenho um serviço pra você\"",
				"Depois do que aconteceu com {alvo}, um cara de óculos escuros deixou um cartão no bolso de {@}: \"Pago bem. Me liga\"",
			},
			choices: []choice{
				{label: "📞 Ouvir a proposta", risk: midRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} entrou no carro de vidro fumê. O ar-condicionado tava no talo"}, next: "crime-matar-contrato"},
				}},
				{label: "🙅 Recusar", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} rasgou o cartão. Tem coisa que é melhor nem saber"}},
				}},
				{label: "🚔 Entregar ele pra polícia", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"A polícia prendeu o homem de terno e ainda agradeceu {@}"}, effects: []effect{earn(60, 150), enchant("justica")}},
					{weight: 50, texts: []string{"A polícia prendeu o homem de terno e quis saber como {@} conhecia ele"}, effects: []effect{nabbed()}},
				}},
			},
		},
		{
			id:    "crime-matar-contrato",
			title: "O contrato",
			stage: true,
			intros: []string{
				"O homem de terno abriu uma maleta: \"O alvo é um político que deve dinheiro pra muita gente. Metade agora, metade depois\"",
			},
			choices: []choice{
				{label: "💰 Aceitar o serviço", risk: highRisk, endings: []ending{
					{weight: 50, favor: func(p Profession) float64 { return p.killChance }, texts: []string{"Serviço feito sem deixar rastro. A maleta veio cheia, como prometido"}, effects: []effect{earn(250, 500), stain()}},
					{weight: 30, texts: []string{"O político tinha seguranças. {@} voltou de lá todo estropiado"}, effects: []effect{injure(30, 55)}},
					{weight: 12, texts: []string{"Era uma armadilha da Polícia Federal"}, effects: []effect{nabbed()}},
					{weight: 8, texts: []string{"Os seguranças do político foram mais rápidos que {@}"}, effects: []effect{perish()}},
				}},
				{label: "🤝 Pegar o adiantamento e sumir", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} pegou a metade adiantada e desapareceu do mapa"}, effects: []effect{earn(100, 200), enchant("encosto")}},
					{weight: 45, texts: []string{"O homem de terno mandou alguém atrás de {@} pra cobrar"}, effects: []effect{injure(20, 35)}},
				}},
				{label: "🚪 Desistir", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} devolveu a maleta e saiu do carro. O ar-condicionado vai deixar saudade"}},
				}},
			},
			idle: ending{texts: []string{"O homem de terno cansou de esperar, destravou a porta e mandou {@} descer"}},
		},
		{
			id:     "crime-matar-arma-travou",
			title:  "Na hora H",
			places: []string{"!matar"},
			when:   wentWrong,
			intros: []string{
				"Na hora H, a arma de {@} travou e {alvo} saiu correndo pra rua gritando o nome completo de {@}",
				"{@} errou feio, {alvo} escapou e agora tá ligando pro 190 escondido atrás de uma banca de jornal",
			},
			choices: []choice{
				{label: "🏃 Correr atrás", risk: highRisk, endings: []ending{
					{weight: 35, favor: func(p Profession) float64 { return p.killChance }, texts: []string{"{@} alcançou {alvo} e tomou o celular antes da ligação completar"}, effects: []effect{gain(1, "nokia")}},
					{weight: 40, texts: []string{"{alvo} jogou o celular na cara de {@} e fugiu do mesmo jeito"}, effects: []effect{injure(10, 25)}},
					{weight: 25, texts: []string{"A polícia chegou bem no meio da perseguição"}, effects: []effect{nabbed()}},
				}},
				{label: "🚌 Pegar o primeiro ônibus", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} pegou um ônibus pra rodoviária e sumiu por umas horas"}, effects: []effect{delay(30 * time.Minute)}},
					{weight: 40, texts: []string{"O ônibus parou numa blitz duas quadras depois"}, effects: []effect{nabbed()}},
				}},
				{label: "🙌 Se entregar", risk: midRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} se entregou na delegacia e o delegado ficou tão surpreso que até ofereceu café"}, effects: []effect{nabbed()}},
					{weight: 30, texts: []string{"{@} foi se entregar, mas o delegado tava de folga. Voltou pra casa"}},
				}},
				{label: "🥸 Vestir o kit disfarce", risk: lowRisk, uses: "kit-disfarce", endings: []ending{
					{weight: 100, texts: []string{"Bigode falso, peruca e óculos. Nem a mãe de {@} reconheceria agora"}},
				}},
			},
		},
		{
			id:     "crime-matar-praga-da-avo",
			title:  "A praga da vó",
			places: []string{"!matar"},
			when:   wentWell,
			intros: []string{
				"A vó de {alvo} é benzedeira das brabas. Ficou sabendo do ocorrido e rogou uma praga em {@} na porta da igreja",
				"{@} passou na frente da casa de {alvo} e a vó tava na janela com um galho de arruda, murmurando o nome de {@}",
			},
			choices: []choice{
				{label: "💐 Mandar flores", risk: lowRisk, price: cost{percent: 3, floor: 80}, endings: []ending{
					{weight: 100, texts: []string{"A vó aceitou as flores, respirou fundo e retirou a praga"}, effects: []effect{enchant("protecao")}},
				}},
				{label: "🧂 Fazer um contra-feitiço", risk: midRisk, needs: either(inArea(areaFe), holds("sal-grosso")), endings: []ending{
					{weight: 60, texts: []string{"{@} jogou sal grosso pro alto e a praga voltou pra quem mandou, levando junto tudo de ruim que tava grudado"}, effects: []effect{enchant("purificacao"), enchant("sorte")}},
					{weight: 40, texts: []string{"O contra-feitiço saiu pela culatra"}, effects: []effect{enchant("azar")}},
				}},
				{label: "😂 Rir da praga", risk: midRisk, endings: []ending{
					{weight: 30, texts: []string{"{@} riu e nada aconteceu. Praga de vó é lenda"}},
					{weight: 45, texts: []string{"A praga pegou com força"}, effects: []effect{enchant("encosto")}},
					{weight: 17, texts: []string{"A praga pegou com força e ainda trouxe uma dor nas costas"}, effects: []effect{enchant("azar"), injure(5, 15)}},
					{weight: 8, texts: []string{"A vó cobrou em dinheiro: o próximo ganho de {@} vai ter a parte da família de {alvo}"}, effects: []effect{grudge("dizimo")}},
				}},
				{label: "🚶 Mudar de calçada pra sempre", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} agora atravessa a rua toda vez que passa por ali. Na dúvida, melhor não arriscar"}},
				}},
			},
		},
		{
			id:     "crime-matar-alma-penada",
			title:  "A alma penada",
			rarity: rareEvent,
			places: []string{"!matar"},
			when:   wentWell,
			intros: []string{
				"Na volta pra casa, um vulto branco sussurrou o nome de {@} no ouvido. Tinha a voz igualzinha à de {alvo}",
				"A luz do poste piscou três vezes e o fantasma de {alvo} apareceu sentado no banco da praça, olhando pra {@}",
			},
			choices: []choice{
				{label: "📿 Rezar um terço", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} rezou o terço inteiro e o vulto foi embora em paz"}, effects: []effect{enchant("protecao")}},
				}},
				{label: "🎰 Pedir os números da Mega", risk: highRisk, endings: []ending{
					{weight: 35, texts: []string{"O fantasma soprou seis números. {@} jogou e acertou a quina"}, effects: []effect{earn(500, 1200)}},
					{weight: 37, texts: []string{"O fantasma deu risada e grudou nas costas de {@}"}, effects: []effect{enchant("encosto"), enchant("azar")}},
					{weight: 8, texts: []string{"O fantasma soprou os números escritos numa página do Livro de São Cipriano. Não era aposta, era pacto"}, effects: []effect{grudge("pacto")}},
					{weight: 20, texts: []string{"O fantasma não gostou do pedido e deu um susto tão grande que {@} desmaiou"}, effects: []effect{injure(20, 40)}},
				}},
				{label: "🏃 Correr sem olhar pra trás", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} correu até em casa e dormiu de luz acesa"}},
					{weight: 35, texts: []string{"{@} olhou pra trás. Erro clássico. Tropeçou e caiu numa vala"}, effects: []effect{injure(10, 25)}},
					{weight: 10, texts: []string{"{@} não olhou pra trás, mas o fantasma de {alvo} veio junto até a porta de casa"}, effects: []effect{grudge("assombracao")}},
				}},
				{label: "🍄 Trocar uma ideia com ele", risk: midRisk, needs: stoned, endings: []ending{
					{weight: 70, texts: []string{"Sob efeito, {@} conversou horas com o fantasma, que ensinou o caminho de um dinheiro enterrado"}, effects: []effect{earn(200, 500)}},
					{weight: 30, texts: []string{"O fantasma não existia. {@} passou a noite conversando com um poste"}, effects: []effect{delay(20 * time.Minute)}},
				}},
			},
		},
		{
			id:     "crime-despacho-encruzilhada",
			title:  "O despacho na encruzilhada",
			rarity: rareEvent,
			places: []string{"!roubar", "!furtar", "!sequestrar", "!matar"},
			when:   wentWell,
			intros: []string{
				"{alvo} foi num terreiro de madrugada e mandou fazer um trabalho pra {@}. Agora tem um despacho com o nome de {@} escrito a carvão bem na porta de casa",
				"Na encruzilhada perto da casa de {@} apareceu uma galinha preta, sete velas e uma foto 3x4 de {@}. Do lado, um bilhete com a letra de {alvo}",
			},
			choices: []choice{
				{label: "🧂 Pagar um descarrego", risk: lowRisk, price: cost{percent: 5, floor: 150}, endings: []ending{
					{weight: 75, texts: []string{"A mãe de santo passou arruda, sal grosso e defumador, e o trabalho de {alvo} desmanchou na fumaça"}, effects: []effect{enchant("purificacao"), enchant("protecao")}},
					{weight: 25, texts: []string{"O descarrego limpou tudo, mas a mãe de santo avisou que {alvo} não desiste fácil"}, effects: []effect{enchant("purificacao")}},
				}},
				{label: "🦶 Chutar o despacho", risk: highRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} chutou tudo longe e nada aconteceu. A galinha ainda virou janta"}, effects: []effect{heal(15)}},
					{weight: 20, texts: []string{"O santo de {alvo} não gostou do chute e mandou a conta"}, effects: []effect{grudge("dizimo")}},
					{weight: 15, texts: []string{"Um vulto saiu da fumaça das velas e seguiu {@} até em casa"}, effects: []effect{grudge("assombracao")}},
					{weight: 10, texts: []string{"Embaixo da galinha tinha um livro de capa preta. Era o Livro de São Cipriano, com o nome de {@} já assinado num pacto"}, effects: []effect{grudge("pacto")}},
				}},
				{label: "🙏 Pedir pra desfazer o trabalho", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{alvo} aceitou desfazer o trabalho em troca de um pedido de desculpas no grupo da família"}, effects: []effect{delay(15 * time.Minute)}},
					{weight: 35, texts: []string{"{alvo} riu na cara de {@}: \"agora você vai me pagar dízimo\""}, effects: []effect{grudge("dizimo")}},
					{weight: 15, texts: []string{"{alvo} desfez o trabalho e ainda acendeu uma vela pra {@} nunca ver grade de cadeia"}, effects: []effect{enchant("libertacao")}},
				}},
				{label: "🕯️ Desmanchar o trabalho", risk: lowRisk, needs: inArea(areaFe), endings: []ending{
					{weight: 80, texts: []string{"{@} conhece o ofício: desmanchou o trabalho, devolveu as velas pra encruzilhada e saiu com o corpo leve"}, effects: []effect{enchant("purificacao"), enchant("libertacao")}},
					{weight: 20, texts: []string{"{@} desmanchou o trabalho, mas a galinha fugiu e levou junto a paciência"}, effects: []effect{enchant("purificacao")}},
				}},
				{label: "🙈 Passar por cima", risk: lowRisk, endings: []ending{
					{weight: 85, texts: []string{"{@} passou por cima e seguiu a vida. Às vezes não acreditar é a melhor proteção"}},
					{weight: 15, texts: []string{"O trabalho pegou devagarinho: era de fome, e desde então nada para no prato de {@}"}, effects: []effect{enchant("jejum")}},
				}},
			},
			idle: ending{texts: []string{"{@} demorou tanto pra decidir que a chuva levou o despacho embora"}},
		},
		{
			id:     "crime-matar-sobreviveu",
			title:  "Colete à prova de balas",
			places: []string{"!matar"},
			when:   wentWrong,
			intros: []string{
				"{alvo} levantou do chão, bateu a poeira do colete e agora tá correndo atrás de {@}, vermelho de raiva",
				"Deu errado: {alvo} saiu ileso e foi buscar os irmãos. São quatro, todos do jiu-jitsu",
			},
			choices: []choice{
				{label: "🌳 Correr pro mato", risk: midRisk, endings: []ending{
					{weight: 60, favor: func(p Profession) float64 { return p.escapeBonus }, texts: []string{"{@} se enfiou no mato e só saiu quando escureceu, todo picado de mosquito"}, effects: []effect{injure(1, 5)}},
					{weight: 40, texts: []string{"{@} se enfiou no mato e caiu sentado num formigueiro"}, effects: []effect{injure(10, 20)}},
				}},
				{label: "🥊 Encarar de frente", risk: highRisk, endings: []ending{
					{weight: 35, favor: func(p Profession) float64 { return p.guard }, texts: []string{"{@} segurou a onda e {alvo} desistiu da briga"}},
					{weight: 55, texts: []string{"{alvo} e os irmãos deram uma surra histórica em {@}"}, effects: []effect{injure(30, 50)}},
					{weight: 10, texts: []string{"A briga foi feia demais e {@} não resistiu"}, effects: []effect{perish()}},
				}},
				{label: "🏳️ Propor uma trégua", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{alvo} aceitou a trégua em troca de um churrasco pago por {@}"}, effects: []effect{lose(3, 6)}},
					{weight: 40, texts: []string{"{alvo} aceitou a trégua, mas ficou secando {@} com um olhar que dá medo"}, effects: []effect{enchant("olhogordo")}},
				}},
				{label: "💸 Pagar uma indenização", risk: lowRisk, price: cost{percent: 8, floor: 150}, endings: []ending{
					{weight: 100, texts: []string{"{@} pagou e {alvo} foi embora resmungando. Paz comprada"}},
				}},
			},
		},
		{
			id:     "crime-assalto-tinta-azul",
			title:  "A bomba de tinta",
			places: []string{"!assalto"},
			when:   wentWell,
			intros: []string{
				"A mochila de dinheiro tinha uma bomba de tinta escondida. Explodiu na fuga e {@} ficou azul da cabeça aos pés",
				"No meio da fuga, um saco de dinheiro explodiu tinta azul em {@}. Agora parece um Smurf foragido",
			},
			choices: []choice{
				{label: "⛲ Lavar no chafariz da praça", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"A tinta saiu quase toda. Ficou só um azulzinho atrás da orelha"}},
					{weight: 45, texts: []string{"Um guarda municipal viu um Smurf tomando banho no chafariz e achou suspeito"}, effects: []effect{stain(), delay(15 * time.Minute)}},
				}},
				{label: "🕺 Assumir o visual e ir pro baile", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} chegou azul no baile e virou a sensação da noite. Foi coroado rei do baile"}, effects: []effect{enchant("sorte")}},
					{weight: 40, texts: []string{"No baile tinha um policial de folga que tinha visto a notícia do assalto"}, effects: []effect{nabbed()}},
				}},
				{label: "👕 Queimar a roupa", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} queimou a roupa num terreno baldio e voltou pra casa de cueca, mas sem prova nenhuma"}},
				}},
			},
		},
		{
			id:     "crime-assalto-gerente",
			title:  "O gerente prestativo",
			places: []string{"!assalto"},
			when:   wentWell,
			intros: []string{
				"Na saída do banco, o gerente correu atrás de {@} oferecendo o cofre do segundo andar \"pra acabar logo com isso\"",
				"O gerente do banco entregou um bilhete pra {@}: \"Tem um cofre maior na agência do centro. Eu odeio meu chefe\"",
			},
			choices: []choice{
				{label: "💰 Voltar pro segundo cofre", risk: highRisk, endings: []ending{
					{weight: 45, favor: func(p Profession) float64 { return p.heistBonus }, texts: []string{"O segundo cofre tava cheio e o gerente ainda ajudou a carregar"}, effects: []effect{earn(250, 500), stain()}},
					{weight: 35, texts: []string{"Era armadilha. Os seguranças tavam esperando no segundo andar"}, effects: []effect{injure(20, 40)}},
					{weight: 20, texts: []string{"A polícia já tava no prédio quando {@} voltou"}, effects: []effect{nabbed()}},
				}},
				{label: "🤝 Dar uma gorjeta pro gerente", risk: lowRisk, price: cost{percent: 3, floor: 100}, endings: []ending{
					{weight: 100, texts: []string{"O gerente agradeceu emocionado e passou a senha do wifi do banco"}, effects: []effect{enchant("sorte")}},
				}},
				{label: "🏃 Recusar e sumir", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} agradeceu a gentileza e sumiu antes que o gerente mudasse de ideia"}},
				}},
				{label: "🤨 Desconfiar do gerente", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} desconfiou e saiu pela porta dos fundos. A frente tava cercada. O próximo assalto já pode ser planejado"}, effects: []effect{refresh()}},
					{weight: 40, texts: []string{"Desconfiar levou tempo demais. A polícia chegou"}, effects: []effect{nabbed()}},
				}},
			},
		},
		{
			id:     "crime-assalto-cercado",
			title:  "Cercado no banco",
			places: []string{"!assalto"},
			when:   wentWrong,
			intros: []string{
				"O alarme tocou e {@} ficou trancado no banco com um caixa, três idosos e uma criança chorando por pirulito",
				"A polícia cercou o banco. Lá dentro, {@} e uma fila de aposentados que só queria receber a aposentadoria",
			},
			choices: []choice{
				{label: "📢 Negociar com a polícia", risk: midRisk, endings: []ending{
					{weight: 45, texts: []string{"{@} pediu pizza e um carro. Veio a pizza e, enquanto todo mundo comia, {@} fugiu pela janela"}, effects: []effect{heal(10)}},
					{weight: 55, texts: []string{"O negociador da polícia era bom. {@} saiu algemado"}, effects: []effect{nabbed()}},
				}},
				{label: "🕳️ Fugir pelo duto de ar", risk: highRisk, endings: []ending{
					{weight: 50, favor: func(p Profession) float64 { return p.escapeBonus }, texts: []string{"{@} rastejou pelo duto e saiu na padaria do lado, todo empoeirado"}},
					{weight: 40, texts: []string{"{@} ficou entalado no duto e precisou ser resgatado pelos bombeiros"}, effects: []effect{injure(10, 25), nabbed()}},
					{weight: 10, texts: []string{"O duto desabou com {@} dentro"}, effects: []effect{perish()}},
				}},
				{label: "🧓 Se misturar com os aposentados", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} pegou uma bengala, entrou na fila e saiu junto com os idosos"}},
					{weight: 45, texts: []string{"Um dos aposentados dedurou {@} pro policial"}, effects: []effect{nabbed()}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, a polícia entrou pela porta da frente"}, effects: []effect{nabbed()}},
		},
		{
			id:     "crime-assalto-piloto-sumiu",
			title:  "O piloto de fuga sumiu",
			places: []string{"!assalto"},
			intros: []string{
				"O piloto de fuga sumiu com o carro e deixou {@} na calçada do banco segurando um saco de moedas de dez centavos",
				"{@} saiu correndo do banco e cadê o carro? O motorista parou pra abastecer e mandou um áudio pedindo cinco minutinhos",
			},
			choices: []choice{
				{label: "🚲 Roubar uma bicicleta", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} saiu pedalando uma bicicleta de criança, com rodinha e tudo"}, effects: []effect{stain()}},
					{weight: 40, texts: []string{"A bicicleta era do policial que tava comendo pastel na esquina"}, effects: []effect{nabbed()}},
				}},
				{label: "🚌 Pegar o ônibus", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} pagou a passagem com as moedas do assalto e o cobrador nem estranhou"}},
					{weight: 45, texts: []string{"O ônibus quebrou bem na frente do batalhão da PM"}, effects: []effect{nabbed()}},
				}},
				{label: "🗑️ Se esconder na caçamba", risk: lowRisk, endings: []ending{
					{weight: 75, texts: []string{"{@} passou uma hora na caçamba de entulho e saiu fedendo, mas livre"}, effects: []effect{delay(15 * time.Minute)}},
					{weight: 25, texts: []string{"O caminhão da caçamba chegou antes da hora e chacoalhou {@} lá dentro"}, effects: []effect{injure(5, 15)}},
				}},
				{label: "🏎️ Pilotar a fuga sozinho", risk: lowRisk, needs: isJob("piloto de fuga"), endings: []ending{
					{weight: 100, texts: []string{"{@} fez ligação direta num carro da esquina e saiu cantando pneu. Profissional é outra coisa"}, effects: []effect{refresh()}},
				}},
			},
		},
		{
			id:     "crime-assalto-gaveta-secreta",
			title:  "A gaveta secreta",
			rarity: legendaryEvent,
			places: []string{"!assalto"},
			when:   wentWell,
			intros: []string{
				"Fuçando o cofre, {@} achou uma gaveta secreta com uma mala de couro e um envelope com o brasão da República",
				"Atrás do cofre principal tinha uma porta escondida. Lá dentro: uma mala, um envelope lacrado e um cheiro de mofo",
			},
			choices: []choice{
				{label: "💼 Levar a mala", risk: highRisk, endings: []ending{
					{weight: 55, texts: []string{"A mala era de um político e tava cheia"}, effects: []effect{gain(1, "mala-de-dinheiro"), stain()}},
					{weight: 30, texts: []string{"A mala tinha rastreador. A Federal achou {@} em meia hora"}, effects: []effect{nabbed()}},
					{weight: 15, texts: []string{"A mala era pesada demais e {@} travou a coluna carregando"}, effects: []effect{injure(20, 35)}},
				}},
				{label: "📜 Levar o envelope", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"No envelope tinha documentos que valem uma fortuna pra um jornalista"}, effects: []effect{earn(300, 600)}},
					{weight: 40, texts: []string{"O envelope só tinha boleto do condomínio do gerente"}, effects: []effect{gain(2, "boleto")}},
				}},
				{label: "🔐 Levar o cofrinho do fundo", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} pegou só um cofrinho esquecido no fundo da gaveta. Pequeno, mas pesado"}, effects: []effect{gain(1, "cofre-esquecido")}},
				}},
				{label: "🚪 Não mexer em nada", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} fechou a porta secreta. Tem coisa que é melhor não saber"}},
				}},
			},
		},
		{
			id:     "crime-assalto-pao-de-queijo",
			title:  "O cheiro do pão de queijo",
			places: []string{"!assalto"},
			when:   wentWell,
			intros: []string{
				"Saindo do banco com a mochila cheia, {@} sentiu o cheiro do pão de queijo saindo do forno da padaria do lado",
				"A padaria colada no banco tava anunciando: pão de queijo quentinho, saindo agora. {@} parou na calçada babando",
			},
			choices: []choice{
				{label: "🧀 Parar pra comprar", risk: midRisk, endings: []ending{
					{weight: 65, texts: []string{"Dez pães de queijo e um café pingado. Foi o melhor assalto da vida de {@}"}, effects: []effect{gain(3, "pao-de-queijo"), heal(10)}},
					{weight: 35, texts: []string{"Na fila do pão de queijo tinha dois policiais"}, effects: []effect{nabbed()}},
				}},
				{label: "🏃 Seguir a fuga", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} seguiu correndo, com o cheiro do pão de queijo na memória pra sempre"}},
				}},
				{label: "🦹 Assaltar a padaria também", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"{@} levou o caixa da padaria e uma fornada inteira"}, effects: []effect{earn(60, 150), gain(5, "pao-de-queijo")}},
					{weight: 40, texts: []string{"O padeiro tava armado com uma pá de pizza"}, effects: []effect{injure(15, 35)}},
					{weight: 15, texts: []string{"O dono da padaria é policial aposentado e chamou os amigos"}, effects: []effect{nabbed()}},
				}},
			},
		},
		{
			id:     "crime-quadrilha-divisao",
			title:  "A divisão do bonde",
			places: []string{"!quadrilha"},
			when:   wentWell,
			intros: []string{
				"Na hora de dividir o que tiraram de {alvo}, a quadrilha começou a discutir quem trabalhou mais. O Magrão jura que foi ele",
				"Cada um da quadrilha acha que merece a maior parte do que levaram de {alvo}. A discussão já tá no volume máximo",
			},
			choices: []choice{
				{label: "🔫 Impor respeito", risk: midRisk, needs: armedUp, endings: []ending{
					{weight: 70, texts: []string{"Todo mundo ficou quietinho e a maior parte ficou com {@}"}, effects: []effect{earn(60, 150), stain()}},
					{weight: 30, texts: []string{"O Magrão também tava armado. Ficou um clima ruim e sobrou pra {@}"}, effects: []effect{injure(10, 25)}},
				}},
				{label: "✊ Resolver no par ou ímpar", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} ganhou no par ou ímpar e levou a parte do Magrão"}, effects: []effect{earn(40, 100)}},
					{weight: 50, texts: []string{"{@} perdeu no par ou ímpar e teve que inteirar a parte dos outros"}, effects: []effect{lose(3, 7)}},
				}},
				{label: "🤝 Aceitar a divisão", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} aceitou a parte que deram. Melhor pouco do que confusão"}},
				}},
			},
		},
		{
			id:     "crime-quadrilha-x9",
			title:  "O X-9 do bonde",
			places: []string{"!quadrilha"},
			when:   wentWell,
			intros: []string{
				"Um dos comparsas tá digitando muito no celular desde o roubo de {alvo} e olhando pra {@} de canto de olho",
				"O Pezão saiu pra \"comprar cigarro\" logo depois do roubo de {alvo}. Faz meia hora e ele não volta",
			},
			choices: []choice{
				{label: "📱 Pegar o celular dele", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"Era X-9 mesmo! {@} desmascarou ele na frente de todo mundo e ganhou moral"}, effects: []effect{earn(40, 100), enchant("protecao")}},
					{weight: 40, texts: []string{"Ele só tava mandando mensagem pra mãe. O clima ficou péssimo e rolou briga"}, effects: []effect{injure(10, 25)}},
					{weight: 15, texts: []string{"Era X-9, e a polícia já tava chegando"}, effects: []effect{nabbed()}},
				}},
				{label: "👀 Ficar de olho", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} ficou de olho e não aconteceu nada. Talvez fosse só paranoia"}},
					{weight: 40, texts: []string{"Enquanto {@} olhava, o X-9 já tinha mandado a localização"}, effects: []effect{stain()}},
				}},
				{label: "🏃 Sumir antes que dê ruim", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} inventou uma desculpa e sumiu. Melhor perder a festa do que a liberdade"}, effects: []effect{delay(15 * time.Minute)}},
				}},
			},
		},
		{
			id:     "crime-quadrilha-churrasco",
			title:  "Churrasco na laje",
			places: []string{"!quadrilha"},
			when:   wentWell,
			intros: []string{
				"A quadrilha resolveu comemorar o roubo de {alvo} com um churrasco na laje, com pagode e tudo",
				"O Magrão chamou a quadrilha toda pra comemorar o serviço em {alvo}. Só que a laje dele fica de frente pro batalhão",
			},
			choices: []choice{
				{label: "🍖 Ir pro churrasco", risk: midRisk, endings: []ending{
					{weight: 65, texts: []string{"{@} comeu picanha, dançou pagode e voltou de barriga cheia"}, effects: []effect{heal(20), enchant("fartura")}},
					{weight: 35, texts: []string{"A polícia seguiu a fumaça do churrasco e subiu na laje"}, effects: []effect{nabbed()}},
				}},
				{label: "🥩 Levar a carne", risk: lowRisk, price: cost{percent: 3, floor: 80}, endings: []ending{
					{weight: 100, texts: []string{"{@} levou picanha da boa e virou o herói do churrasco"}, effects: []effect{heal(20), enchant("sorte")}},
				}},
				{label: "🏠 Ficar em casa", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} ficou em casa vendo novela. Prudência é tudo"}},
				}},
				{label: "🌿 Levar o baseado", risk: midRisk, uses: "baseado", endings: []ending{
					{weight: 70, texts: []string{"O baseado rodou a laje inteira e o pagode ficou ainda melhor"}, effects: []effect{heal(10), enchant("sorte")}},
					{weight: 30, texts: []string{"A fumaça chamou a atenção do vizinho, que chamou a polícia"}, effects: []effect{stain()}},
				}},
			},
		},
		{
			id:     "crime-quadrilha-abandonado",
			title:  "Abandonado pela quadrilha",
			places: []string{"!quadrilha"},
			when:   wentWrong,
			intros: []string{
				"Quando deu ruim, a quadrilha inteira correu e deixou {@} sozinho, cara a cara com {alvo}",
				"Os comparsas sumiram na primeira dificuldade. Agora é só {@}, {alvo} e um silêncio constrangedor",
			},
			choices: []choice{
				{label: "🙏 Pedir desculpa e devolver algo", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} pediu desculpa e devolveu um trocado. {alvo} até ficou com pena"}, effects: []effect{refund(3, 8)}},
				}},
				{label: "🥊 Sair no braço", risk: highRisk, endings: []ending{
					{weight: 35, favor: func(p Profession) float64 { return p.guard }, texts: []string{"{@} venceu a briga e {alvo} pediu arrego, deixando a carteira pra trás"}, effects: []effect{earn(30, 80)}},
					{weight: 50, texts: []string{"{alvo} sabia lutar. {@} apanhou sozinho pela quadrilha inteira"}, effects: []effect{injure(25, 45)}},
					{weight: 10, texts: []string{"{alvo} chamou a polícia no meio da briga"}, effects: []effect{nabbed()}},
					{weight: 5, texts: []string{"A briga terminou da pior forma possível pra {@}"}, effects: []effect{perish()}},
				}},
				{label: "🎭 Fingir que foi sequestrado pelo bonde", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{alvo} acreditou e ainda ofereceu um copo d'água"}},
					{weight: 50, texts: []string{"{alvo} não acreditou nem um pouco e tirou foto de {@} pra polícia"}, effects: []effect{stain()}},
				}},
			},
		},
		{
			id:     "crime-quadrilha-uniforme",
			title:  "O uniforme do bonde",
			places: []string{"!quadrilha"},
			intros: []string{
				"A quadrilha quer um nome e um uniforme. Já sugeriram \"Bonde do Pix\", \"Os Sem Boleto\" e \"Tropa do Calote\"",
				"O Magrão mandou fazer camiseta pra quadrilha toda, com o nome do bonde e a foto de {@} estampada no peito",
			},
			choices: []choice{
				{label: "👕 Pagar as camisetas", risk: lowRisk, price: cost{percent: 2, floor: 60}, endings: []ending{
					{weight: 100, texts: []string{"As camisetas ficaram lindas. O bonde nunca esteve tão unido"}, effects: []effect{enchant("sorte")}},
				}},
				{label: "📛 Sugerir um nome melhor", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"Todo mundo amou o nome que {@} deu. Virou lenda no bairro"}, effects: []effect{enchant("abrecaminho")}},
					{weight: 40, texts: []string{"Ninguém gostou do nome. {@} levou zoação a semana inteira"}},
				}},
				{label: "🙅 Achar brega", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} achou brega e o bonde ficou sem uniforme mesmo"}},
					{weight: 40, texts: []string{"O Magrão ficou ofendido e cobrou de {@} o dinheiro da gráfica"}, effects: []effect{lose(2, 4)}},
				}},
				{label: "🚔 Lembrar que uniforme ajuda a polícia", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"Todo mundo concordou que uniforme de bandido é burrice. {@} salvou o bonde e já pode planejar o próximo"}, effects: []effect{refresh()}},
				}},
			},
		},
		{
			id:     "crime-quadrilha-chefao",
			title:  "O chamado do chefão",
			rarity: rareEvent,
			places: []string{"!quadrilha"},
			when:   wentWell,
			intros: []string{
				"O chefão do morro ficou sabendo do que a quadrilha fez com {alvo} e mandou chamar {@} pessoalmente",
				"Um moleque de bicicleta entregou um recado pra {@}: o dono do morro quer conversar sobre o serviço em {alvo}",
			},
			choices: []choice{
				{label: "🏚️ Subir o morro", risk: highRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} subiu o morro escoltado por dois seguranças de óculos escuros"}, next: "crime-quadrilha-reuniao"},
				}},
				{label: "🙅 Não ir", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} não foi. O chefão ficou chateado, mas deixou quieto"}},
					{weight: 40, texts: []string{"O chefão não gosta de ser ignorado e mandou um recado bem dolorido"}, effects: []effect{injure(15, 30)}},
				}},
				{label: "🎁 Mandar um presente", risk: lowRisk, price: cost{percent: 5, floor: 200}, endings: []ending{
					{weight: 100, texts: []string{"O chefão gostou do presente e mandou dizer que {@} tá com moral"}, effects: []effect{enchant("protecao")}},
				}},
			},
		},
		{
			id:    "crime-quadrilha-reuniao",
			title: "A reunião no morro",
			stage: true,
			intros: []string{
				"O chefão do morro serviu um uísque e foi direto ao ponto: \"Gostei de você. Quer trabalhar pra mim?\"",
			},
			choices: []choice{
				{label: "🤝 Aceitar entrar pro bonde", risk: highRisk, endings: []ending{
					{weight: 55, texts: []string{"O chefão deu as boas-vindas com um presente e um adiantamento"}, effects: []effect{earn(300, 600), gain(1, "oitao", "peixeira", "soco-ingles"), stain()}},
					{weight: 30, texts: []string{"No primeiro serviço pro chefão deu ruim e a polícia pegou {@}"}, effects: []effect{nabbed()}},
					{weight: 15, texts: []string{"O chefão desconfiou que {@} era X-9 infiltrado"}, effects: []effect{injure(30, 50)}},
				}},
				{label: "💸 Pedir pra trabalhar por conta", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"O chefão respeitou e liberou {@} com a bênção dele"}, effects: []effect{enchant("abrecaminho")}},
					{weight: 40, texts: []string{"O chefão cobrou uma taxa pra deixar {@} trabalhar no território"}, effects: []effect{lose(5, 10)}},
				}},
				{label: "🚪 Agradecer e ir embora", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} agradeceu o uísque e desceu o morro inteiro rezando"}},
				}},
			},
			idle: ending{texts: []string{"O chefão não gosta de esperar. Mandou descer {@} sem resposta e sem o uísque"}},
		},
		{
			id:     "crime-mercado-caixa-do-porto",
			title:  "A caixa do porto",
			places: []string{"!mercadonegro"},
			intros: []string{
				"Um muambeiro novo chamou {@} no canto: \"Chegou um negócio do porto agora. Caixa fechada, preço de amigo\"",
				"Um sujeito de óculos escuros dentro de uma Kombi oferece pra {@} uma caixa lacrada \"que caiu de um contêiner\"",
			},
			choices: []choice{
				{label: "📦 Comprar no escuro", risk: midRisk, price: cost{percent: 5, floor: 150}, endings: []ending{
					{weight: 55, texts: []string{"Na caixa tinha coisa boa de verdade"}, effects: []effect{gain(1, "kit-disfarce", "documento-falso", "soco-ingles", "peixeira")}},
					{weight: 30, texts: []string{"A caixa tinha uma pedra embrulhada em jornal. Clássico"}, effects: []effect{gain(1, "jornal")}},
					{weight: 15, texts: []string{"A caixa era isca da polícia"}, effects: []effect{nabbed()}},
				}},
				{label: "🔍 Pedir pra abrir antes", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"O muambeiro abriu: era só perfume do Paraguai. {@} levou um de brinde"}, effects: []effect{gain(1, "perfume")}},
					{weight: 40, texts: []string{"O muambeiro se ofendeu e sumiu com a Kombi"}},
				}},
				{label: "🙅 Dispensar", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} dispensou. Caixa fechada só traz problema"}},
				}},
				{label: "🚔 Dar voz de prisão", risk: midRisk, needs: lawman, endings: []ending{
					{weight: 65, texts: []string{"{@} prendeu o muambeiro e ainda ganhou elogio do delegado"}, effects: []effect{earn(80, 200), enchant("justica")}},
					{weight: 35, texts: []string{"O muambeiro tinha amigos escondidos na Kombi"}, effects: []effect{injure(15, 30)}},
				}},
			},
		},
		{
			id:     "crime-mercado-federal",
			title:  "É a Federal!",
			places: []string{"!mercadonegro"},
			intros: []string{
				"Um olheiro gritou \"É A FEDERAL!\" e o mercado negro inteiro saiu correndo, com mercadoria voando pra todo lado",
				"Sirene, helicóptero e um megafone: \"Ninguém se mexe!\". {@} tá bem no meio do mercado negro",
			},
			choices: []choice{
				{label: "🏃 Largar tudo e correr", risk: midRisk, endings: []ending{
					{weight: 60, favor: func(p Profession) float64 { return p.escapeBonus }, texts: []string{"{@} pulou uma barraca de CD pirata e sumiu na multidão"}},
					{weight: 40, texts: []string{"{@} escorregou numa poça e caiu nos pés de um agente federal"}, effects: []effect{nabbed()}},
				}},
				{label: "🛢️ Se esconder num barril", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} passou uma hora dentro de um barril de azeitona e saiu com cheiro de pizza"}, effects: []effect{delay(15 * time.Minute)}},
					{weight: 45, texts: []string{"O cachorro da Federal farejou o barril"}, effects: []effect{nabbed()}},
				}},
				{label: "📦 Pegar mercadoria largada", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"No meio da confusão, {@} encheu a mochila com o que ficou pra trás"}, effects: []effect{gain(2, "baseado", "lanca-perfume", "rebite"), earn(40, 100)}},
					{weight: 40, texts: []string{"{@} encheu a mochila e foi preso com ela cheia"}, effects: []effect{nabbed()}},
					{weight: 15, texts: []string{"{@} levou uma bala de borracha na perna no meio da confusão"}, effects: []effect{injure(20, 40)}},
				}},
				{label: "📝 Fingir que é jornalista", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} pegou um caderno, fez cara de repórter e os federais até deram entrevista"}},
					{weight: 50, texts: []string{"O delegado pediu o crachá de imprensa"}, effects: []effect{nabbed()}},
				}},
			},
			idle: ending{texts: []string{"{@} ficou parado e os federais passaram reto, achando que era um manequim"}},
		},
		{
			id:     "crime-mercado-cartao-vip",
			title:  "O cartão VIP",
			places: []string{"!mercadonegro"},
			when:   wentWell,
			intros: []string{
				"O dono do mercado negro gostou de {@} e ofereceu um cartão de cliente VIP: \"Desconto, prioridade e cafezinho\"",
				"\"Você é bom cliente\", disse o dono do mercado negro, entregando um cartão preto com uma caveirinha dourada",
			},
			choices: []choice{
				{label: "💳 Aceitar o cartão", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} virou VIP. O dono prometeu mandar uma surpresa qualquer dia desses"}, effects: []effect{plant("crime-mercado-surpresa", 6*time.Hour)}},
					{weight: 40, texts: []string{"O cartão VIP vinha com uma anuidade salgada"}, effects: []effect{lose(3, 6)}},
				}},
				{label: "🙅 Recusar o cartão", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} recusou. Cliente fiel de mercado negro é o primeiro que a polícia conhece"}},
				}},
				{label: "☕ Aceitar só o cafezinho", risk: midRisk, endings: []ending{
					{weight: 80, texts: []string{"O cafezinho do mercado negro é surpreendentemente bom"}, effects: []effect{heal(5)}},
					{weight: 20, texts: []string{"O cafezinho tinha alguma coisa a mais"}, effects: []effect{buzz("rebite")}},
				}},
			},
		},
		{
			id:     "crime-mercado-surpresa",
			title:  "A surpresa do mercado negro",
			stage:  true,
			places: []string{"!pescar", "!minerar", "!vasculhar", "!catar", "!trabalhar", "!roubar", "!furtar", "!sequestrar", "!matar", "!assalto", "!quadrilha", "!mercadonegro"},
			intros: []string{
				"Um motoboy parou do lado de {@} com um embrulho: \"Lembrança do mercado negro pro cliente VIP\"",
			},
			choices: []choice{
				{label: "🎁 Abrir o embrulho", risk: midRisk, endings: []ending{
					{weight: 70, texts: []string{"Era um presente de verdade, com cartãozinho e tudo"}, effects: []effect{gain(1, "kit-disfarce", "documento-falso", "furadeira")}},
					{weight: 30, texts: []string{"Era um presente, e também uma escuta da polícia escondida no laço"}, effects: []effect{stain()}},
				}},
				{label: "🚮 Jogar fora sem abrir", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} jogou o embrulho no lixo. Presente de bandido nunca é de graça"}},
				}},
				{label: "💸 Revender lacrado", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} revendeu o embrulho lacrado pra um curioso na esquina"}, effects: []effect{earn(40, 100)}},
				}},
			},
			idle: ending{texts: []string{"O motoboy cansou de esperar e largou o embrulho no chão. Dentro tinha um documento falso"}, effects: []effect{gain(1, "documento-falso")}},
		},
		{
			id:     "crime-mercado-troco-suspeito",
			title:  "O troco suspeito",
			places: []string{"!mercadonegro"},
			intros: []string{
				"O vendedor do mercado negro deu o troco de {@} com uma nota de três reais e cara de paisagem",
				"Na hora do troco, o muambeiro empurrou umas notas amassadas pra {@}. Uma delas tem a cara de um papagaio",
			},
			choices: []choice{
				{label: "🤑 Aceitar a nota", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} guardou a nota de três reais. Pelo menos é rara, e colecionador paga bem"}, effects: []effect{gain(1, "nota-de-tres")}},
				}},
				{label: "😤 Reclamar", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O vendedor pediu desculpa e deu um desconto pra compensar"}, effects: []effect{earn(20, 60)}},
					{weight: 50, texts: []string{"O vendedor tinha três primos grandes atrás da barraca"}, effects: []effect{injure(10, 25)}},
				}},
				{label: "🔄 Passar a nota pra frente", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} passou a nota no pastel e ainda pegou o troco certinho"}, effects: []effect{earn(10, 30), gain(1, "pastel")}},
					{weight: 40, texts: []string{"O pasteleiro conferiu a nota contra a luz e chamou a polícia"}, effects: []effect{stain()}},
				}},
			},
		},
		{
			id:     "crime-mercado-oitao",
			title:  "O Oitão de procedência",
			places: []string{"!mercadonegro"},
			intros: []string{
				"Um sujeito de capuz oferece pra {@} um Oitão \"quase novo, só tem um B.O. e meio\"",
				"No fundo do mercado negro, um vendedor desembrulha um revólver de um pano de prato: \"Esse aqui é de confiança\"",
			},
			choices: []choice{
				{label: "🔫 Comprar o Oitão", risk: midRisk, price: cost{percent: 10, floor: 700}, endings: []ending{
					{weight: 70, texts: []string{"Negócio fechado. O Oitão tá na mochila de {@}"}, effects: []effect{gain(1, "oitao"), stain()}},
					{weight: 30, texts: []string{"O vendedor era policial disfarçado e prendeu {@} na hora"}, effects: []effect{nabbed()}},
				}},
				{label: "🔪 Levar a peixeira, mais barata", risk: midRisk, price: cost{percent: 3, floor: 150}, endings: []ending{
					{weight: 85, texts: []string{"A peixeira veio até com bainha de couro"}, effects: []effect{gain(1, "peixeira")}},
					{weight: 15, texts: []string{"A peixeira era de plástico. Golpe dentro do golpe"}},
				}},
				{label: "❓ Perguntar a procedência", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O vendedor contou a história da arma por meia hora e {@} desistiu de comprar"}, effects: []effect{delay(10 * time.Minute)}},
					{weight: 50, texts: []string{"Pergunta errada. O vendedor achou que {@} era X-9"}, effects: []effect{injure(10, 25)}},
				}},
				{label: "🙅 Recusar", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} recusou. Arma com B.O. e meio é encrenca e meia"}},
				}},
			},
		},
		{
			id:     "crime-mercado-cachorro-farejador",
			title:  "O cachorro farejador",
			places: []string{"!mercadonegro"},
			intros: []string{
				"Saindo do mercado negro, um cachorro farejador da polícia começou a cheirar a mochila de {@} com muito interesse",
				"Um pastor alemão de colete da PM sentou na frente de {@} e começou a latir pra mochila",
			},
			choices: []choice{
				{label: "🥟 Dar a coxinha pro cachorro", risk: lowRisk, uses: "coxinha", endings: []ending{
					{weight: 100, texts: []string{"O cachorro comeu a coxinha, ganhou o dia e esqueceu da mochila"}},
				}},
				{label: "🤧 Fingir alergia a cachorro", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} espirrou tanto que o policial ficou com dó e puxou o cachorro"}},
					{weight: 45, texts: []string{"O policial não acreditou e revistou a mochila"}, effects: []effect{forfeit("baseado", 3), forfeit("po", 2), stain()}},
				}},
				{label: "🏃 Correr", risk: midRisk, endings: []ending{
					{weight: 50, favor: func(p Profession) float64 { return p.escapeBonus }, texts: []string{"{@} correu e o cachorro preferiu ficar com o policial"}},
					{weight: 50, texts: []string{"O cachorro correu mais e mordeu a bunda de {@}"}, effects: []effect{injure(10, 20)}},
				}},
				{label: "🧍 Ficar tranquilo", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"O cachorro cheirou, cheirou e foi embora. Era só cheiro de pastel"}},
					{weight: 40, texts: []string{"O cachorro achou alguma coisa e o policial confiscou"}, effects: []effect{forfeit("baseado", 2), forfeit("lanca-perfume", 1)}},
				}},
			},
		},
		{
			id:     "crime-roubar-uber-da-fuga",
			title:  "O Uber da fuga",
			places: []string{"!roubar"},
			when:   wentWell,
			intros: []string{
				"{@} chamou um carro de aplicativo pra fugir e só no banco de trás percebeu: o motorista é o cunhado de {alvo}",
				"O motorista do aplicativo olhou pelo retrovisor e falou: \"Engraçado, minha cunhada {alvo} acabou de ser roubada aqui perto\"",
			},
			choices: []choice{
				{label: "🤫 Ficar quietinho no banco", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} fingiu que tava no telefone a viagem inteira e desceu duas ruas antes"}},
					{weight: 40, texts: []string{"O motorista avaliou {@} com uma estrela e escreveu \"suspeito\" no comentário"}, effects: []effect{stain()}},
				}},
				{label: "💸 Dar uma gorjeta gorda", risk: lowRisk, price: cost{percent: 3, floor: 60}, endings: []ending{
					{weight: 85, texts: []string{"Com a gorjeta, o motorista esqueceu a cunhada, a família e o próprio nome"}},
					{weight: 15, texts: []string{"O motorista agradeceu a gorjeta e contou tudo pra {alvo} no almoço de domingo"}, effects: []effect{stain()}},
				}},
				{label: "🚪 Pular do carro andando", risk: highRisk, endings: []ending{
					{weight: 55, favor: func(p Profession) float64 { return p.escapeBonus }, texts: []string{"{@} rolou no asfalto igual filme de ação e sumiu num beco"}, effects: []effect{injure(3, 10)}},
					{weight: 37, texts: []string{"O carro tava a 60 por hora. {@} ralou até a alma"}, effects: []effect{injure(20, 40)}},
					{weight: 8, texts: []string{"{@} pulou bem na frente de um ônibus"}, effects: []effect{perish()}},
				}},
				{label: "🔫 Tomar o volante", risk: highRisk, needs: armedUp, endings: []ending{
					{weight: 55, texts: []string{"{@} mostrou o ferro, o motorista desceu correndo e ainda largou a pochete no banco"}, effects: []effect{earn(40, 120), stain()}},
					{weight: 45, texts: []string{"O carro tinha rastreador e a polícia fechou a rua em cinco minutos"}, effects: []effect{nabbed()}},
				}},
			},
			idle: ending{texts: []string{"{@} demorou tanto pra decidir que a corrida acabou e o app cobrou a taxa de espera"}},
		},
		{
			id:     "crime-roubar-dinheiro-do-dizimo",
			title:  "O dinheiro do dízimo",
			rarity: rareEvent,
			places: []string{"!roubar"},
			when:   wentWell,
			intros: []string{
				"As notas que {@} tirou de {alvo} tão todas carimbadas: DÍZIMO DA IGREJA DO PASTOR VALDECIR",
				"{alvo} tava levando o envelope do dízimo pra igreja. Agora o envelope tá com {@}, e o pastor já tá rezando contra",
			},
			choices: []choice{
				{label: "⛪ Devolver pro pastor", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"O pastor recebeu o envelope de volta, chorou e orou por {@} no microfone"}, effects: []effect{refund(8, 15), enchant("protecao")}},
					{weight: 40, texts: []string{"{@} devolveu, e {alvo} ganhou fama de abençoado no bairro"}, effects: []effect{refund(8, 15), enchant("justica")}},
				}},
				{label: "🛍️ Torrar tudo no shopping", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} gastou tudo numa tarde e voltou cheio de sacola"}, effects: []effect{earn(60, 150), gain(1, "rolex-do-paraguai", "perfume")}},
					{weight: 50, texts: []string{"O pastor descobriu e expulsou {@} da igreja na frente de todo mundo"}, effects: []effect{enchant("excomunhao")}},
				}},
				{label: "🧺 Lavar na lavanderia do primo", risk: highRisk, price: cost{percent: 5, floor: 100}, endings: []ending{
					{weight: 65, texts: []string{"A lavanderia do primo lavou as notas e devolveu sem carimbo, limpinhas e cheirosas"}, effects: []effect{earn(150, 300), stain()}},
					{weight: 35, texts: []string{"A lavanderia do primo era investigada pela Receita"}, effects: []effect{nabbed()}},
				}},
				{label: "🖨️ Tirar o carimbo na gráfica", risk: midRisk, needs: isJob("falsificador"), endings: []ending{
					{weight: 80, texts: []string{"Trabalho de mestre. Ninguém nunca vai saber que aquilo era dízimo"}, effects: []effect{earn(150, 300)}},
					{weight: 20, texts: []string{"O carimbo saiu, mas a cara do pastor ficou marcada d'água em todas as notas"}, effects: []effect{enchant("azar")}},
				}},
			},
		},
		{
			id:     "crime-furtar-alarme-do-gol",
			title:  "O alarme do Gol",
			places: []string{"!furtar"},
			when:   wentWrong,
			intros: []string{
				"{@} encostou no Gol de {alvo} e o alarme disparou tocando \"Evidências\" no volume máximo",
				"O Gol bolinha de {alvo} começou a apitar e piscar o farol assim que {@} encostou na maçaneta. A rua inteira olhou",
			},
			choices: []choice{
				{label: "🎶 Cantar junto pra disfarçar", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} soltou a voz no refrão e a vizinhança achou que era karaokê"}},
					{weight: 50, texts: []string{"{@} errou a letra, e quem erra Evidências não merece perdão. Alguém filmou"}, effects: []effect{stain()}},
				}},
				{label: "🏃 Correr pra esquina", risk: midRisk, endings: []ending{
					{weight: 60, favor: func(p Profession) float64 { return p.escapeBonus }, texts: []string{"{@} virou a esquina antes do alarme chegar no segundo refrão"}},
					{weight: 40, texts: []string{"{@} correu olhando pra trás e bateu de cara num poste"}, effects: []effect{injure(5, 15)}},
				}},
				{label: "🫣 Se esconder debaixo do carro", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} ficou deitado embaixo do Gol até o alarme cansar"}, effects: []effect{delay(15 * time.Minute)}},
					{weight: 30, texts: []string{"O Gol pingava óleo bem na cara de {@}"}, effects: []effect{injure(3, 8)}},
				}},
				{label: "🔧 Arrancar o rádio no pé de cabra", risk: highRisk, needs: holds("pe-de-cabra"), endings: []ending{
					{weight: 60, texts: []string{"Já que era pra dar escândalo, {@} arrancou o rádio do painel e saiu correndo"}, effects: []effect{gain(1, "radinho"), stain()}},
					{weight: 40, texts: []string{"{alvo} saiu de casa de pijama e chinelo e segurou {@} até a polícia chegar"}, effects: []effect{nabbed()}},
				}},
			},
		},
		{
			id:     "crime-furtar-bolso-do-magico",
			title:  "O bolso do mágico",
			places: []string{"!furtar"},
			when:   wentWell,
			intros: []string{
				"Junto com o furto de {alvo}, saiu do bolso um lenço colorido que não acaba nunca. E tem alguma coisa se mexendo lá dentro",
				"{@} enfiou a mão no bolso de {alvo} e puxou um lenço. Depois outro. Depois outro. Lá no fundo, uma pomba arrulhou",
			},
			choices: []choice{
				{label: "🕊️ Soltar a pomba", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"A pomba voou, deu uma volta em cima de {@} e não fez sujeira. Dizem que é sinal bom"}, effects: []effect{enchant("libertacao")}},
				}},
				{label: "🧣 Puxar o lenço até o fim", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"No fim de quarenta metros de lenço tinha uma nota dobradinha"}, effects: []effect{earn(30, 80)}},
					{weight: 30, texts: []string{"{@} puxou lenço por meia hora e ele não acabou"}, effects: []effect{delay(15 * time.Minute)}},
					{weight: 20, texts: []string{"O lenço enroscou no pescoço de {@} e quase deu ruim"}, effects: []effect{injure(5, 12)}},
				}},
				{label: "🐦 Vender a pomba na feira", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"Um senhor da feira pagou bem pela pomba amestrada"}, effects: []effect{earn(20, 60)}},
					{weight: 40, texts: []string{"Pomba de mágico é bicho sagrado. {alvo} jogou uma praga lá de longe"}, effects: []effect{enchant("azar")}},
				}},
				{label: "🪄 Reconhecer o truque", risk: lowRisk, needs: isJob("mágico"), endings: []ending{
					{weight: 100, texts: []string{"{@} conhecia o truque e achou o fundo falso do bolso, com as moedas da apresentação"}, effects: []effect{earn(60, 150), enchant("sorte")}},
				}},
			},
		},
		{
			id:     "crime-sequestrar-refem-influencer",
			title:  "O refém influencer",
			places: []string{"!sequestrar"},
			when:   wentWell,
			intros: []string{
				"{alvo} pediu, com a maior naturalidade, pra fazer uma live do cativeiro: \"Gente, vocês não vão acreditar onde eu tô\"",
				"Mal chegou no cativeiro e {alvo} já tava procurando o melhor ângulo pra gravar um vídeo de \"arrume-se comigo, versão sequestro\"",
			},
			choices: []choice{
				{label: "📵 Confiscar o celular", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} tomou o celular. Era um tijolão velho, mas ainda liga"}, effects: []effect{gain(1, "nokia")}},
					{weight: 40, texts: []string{"{alvo} chorou por uma hora a perda do engajamento"}},
				}},
				{label: "🎥 Deixar a live e cobrar Pix", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"Os seguidores mandaram Pix pra pagar o resgate ao vivo. Bateu meta"}, effects: []effect{earn(100, 250)}},
					{weight: 40, texts: []string{"Um seguidor reconheceu a parede do cativeiro e mandou a localização pra polícia"}, effects: []effect{nabbed()}},
					{weight: 15, texts: []string{"A live viralizou, e {@} virou o vilão da internet"}, effects: []effect{stain(), enchant("olhogordo")}},
				}},
				{label: "🎭 Aparecer de máscara na live", risk: midRisk, needs: holds("mascara"), endings: []ending{
					{weight: 70, texts: []string{"O sequestrador mascarado virou meme e os seguidores pagaram pra ver o próximo episódio"}, effects: []effect{earn(80, 200), enchant("sorte")}},
					{weight: 30, texts: []string{"A máscara escorregou bem no meio da live"}, effects: []effect{stain()}},
				}},
				{label: "🙄 Ignorar", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} fingiu que não ouviu. {alvo} acabou gravando um vídeo pros stories, sem sinal, e desistiu"}},
				}},
			},
		},
		{
			id:     "crime-sequestrar-familia-nao-quer",
			title:  "A família não quer de volta",
			rarity: rareEvent,
			places: []string{"!sequestrar"},
			when:   wentWell,
			intros: []string{
				"{@} ligou pra família de {alvo} pedindo o resgate e a sogra respondeu: \"Pode ficar, meu filho. E obrigada\"",
				"A família de {alvo} mandou um áudio comemorando: \"Finalmente a casa vai ter paz. Não precisa devolver não\"",
			},
			choices: []choice{
				{label: "😢 Soltar de pena", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} soltou {alvo} e ainda deu o dinheiro da passagem. Coitado"}, effects: []effect{refund(2, 5), enchant("protecao")}},
				}},
				{label: "🧾 Cobrar a estadia da família", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"A família pagou uma mensalidade pra {@} ficar com {alvo} mais um mês"}, effects: []effect{earn(60, 150)}},
					{weight: 50, texts: []string{"A família devolveu a cobrança com a conta do almoço que {alvo} comeu no cativeiro"}, effects: []effect{lose(2, 5)}},
				}},
				{label: "🧹 Botar o refém pra trabalhar", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{alvo} limpou a casa, lavou o carro e vendeu um bolo na rua. O lucro ficou com {@}"}, effects: []effect{earn(40, 100)}},
					{weight: 40, texts: []string{"{alvo} não sai mais da casa de {@}, e agora tem um encosto morando lá"}, effects: []effect{enchant("encosto")}},
				}},
				{label: "📞 Renegociar com a sogra", risk: lowRisk, needs: isJob("sequestrador"), endings: []ending{
					{weight: 100, texts: []string{"{@} conhece o ofício: a sogra pagou pra {alvo} NÃO voltar pra casa"}, effects: []effect{earn(150, 300)}},
				}},
			},
		},
		{
			id:     "crime-matar-velorio",
			title:  "O velório",
			places: []string{"!matar"},
			when:   wentWell,
			intros: []string{
				"A família de {alvo} tá fazendo o velório no salão da igreja, com café, bolo de fubá e muita tia chorando. {@} tá passando na porta",
				"{@} recebeu no grupo do bairro o convite do velório de {alvo}: \"Vai ter café e pão com mortadela\"",
			},
			choices: []choice{
				{label: "☕ Entrar pro café", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} tomou café, comeu três fatias de bolo e deu os pêsames pra família inteira"}, effects: []effect{heal(10)}},
					{weight: 40, texts: []string{"Uma tia reconheceu {@} da hora do ocorrido e começou a gritar no meio do salão"}, effects: []effect{stain()}},
				}},
				{label: "💐 Mandar uma coroa de flores", risk: lowRisk, price: cost{percent: 2, floor: 50}, endings: []ending{
					{weight: 100, texts: []string{"A coroa chegou sem nome, escrito só \"saudades\". A família achou lindo"}, effects: []effect{enchant("protecao")}},
				}},
				{label: "😭 Chorar mais que todo mundo", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} chorou tanto que a família fez uma vaquinha pra consolar o amigo inconsolável"}, effects: []effect{earn(20, 60)}},
					{weight: 50, texts: []string{"O teatro foi demais e {alvo}, lá do além, não gostou nada"}, effects: []effect{enchant("encosto")}},
				}},
				{label: "🙈 Passar longe", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} deu a volta no quarteirão. Velório é lugar de parente"}},
				}},
			},
		},
		{
			id:     "crime-matar-banheiro-do-bar",
			title:  "Trancado no banheiro do bar",
			places: []string{"!matar"},
			when:   wentWrong,
			intros: []string{
				"{@} errou o golpe e {alvo} se trancou no banheiro do bar. Lá de dentro, dá pra ouvir {alvo} ligando pra mãe",
				"Depois do erro, {alvo} se enfiou no banheiro do boteco e trancou a porta. O dono do bar já tá olhando torto pra {@}",
			},
			choices: []choice{
				{label: "🚪 Arrombar a porta", risk: highRisk, endings: []ending{
					{weight: 35, favor: func(p Profession) float64 { return p.killChance }, texts: []string{"A porta cedeu e {alvo} pulou pela janelinha, mas deixou a carteira cair"}, effects: []effect{earn(20, 60)}},
					{weight: 45, texts: []string{"{alvo} tava esperando atrás da porta com o desodorante do banheiro e um isqueiro"}, effects: []effect{injure(15, 35)}},
					{weight: 15, texts: []string{"O dono do bar já tinha chamado a polícia"}, effects: []effect{nabbed()}},
					{weight: 5, texts: []string{"A porta caiu em cima de {@}, que bateu a cabeça na pia"}, effects: []effect{perish()}},
				}},
				{label: "🍺 Pedir uma cerveja e esperar", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{alvo} saiu pela janelinha. {@} ficou só com a cerveja quente e a conta"}, effects: []effect{lose(1, 3)}},
					{weight: 50, texts: []string{"O dono do bar fingiu que ia buscar a cerveja e voltou com dois PMs"}, effects: []effect{nabbed()}},
				}},
				{label: "🗣️ Dizer que foi um engano", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} jurou que foi brincadeira e pagou uma rodada pro bar inteiro pra provar"}, effects: []effect{lose(2, 5)}},
					{weight: 30, texts: []string{"Ninguém acreditou, e o bar inteiro ficou secando {@}"}, effects: []effect{enchant("olhogordo")}},
				}},
				{label: "🎯 Esperar na saída dos fundos", risk: highRisk, needs: killer, endings: []ending{
					{weight: 60, texts: []string{"Profissional não desiste. {@} pegou {alvo} saindo pela janelinha e levou tudo que tinha no bolso"}, effects: []effect{earn(60, 150), stain()}},
					{weight: 40, texts: []string{"Nos fundos do bar tava estacionada uma viatura"}, effects: []effect{nabbed()}},
				}},
			},
		},
		{
			id:     "crime-assalto-mascara-errada",
			title:  "A máscara errada",
			places: []string{"!assalto"},
			when:   wentWell,
			intros: []string{
				"O combinado era todo mundo de Máscara do Dalí, mas {@} chegou de máscara do Chaves. As câmeras do banco amaram",
				"A notícia do assalto já tá no jornal da tarde: \"Bando de Dalís e um Chaves\". O Chaves é {@}",
			},
			choices: []choice{
				{label: "🌊 Jogar a máscara no rio", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} jogou a máscara do Chaves no rio. Foi sem querer querendo"}},
				}},
				{label: "📺 Assumir e virar meme", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O Chaves do assalto virou meme nacional e ninguém leva a sério um bandido de meme"}, effects: []effect{enchant("sorte")}},
					{weight: 50, texts: []string{"Virou meme, e o meme tinha o tênis de {@} bem nítido"}, effects: []effect{stain()}},
				}},
				{label: "👕 Trocar de roupa no shopping", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} entrou no provador de Chaves e saiu de camisa da seleção, irreconhecível"}, effects: []effect{gain(1, "camisa-do-brasil")}},
					{weight: 40, texts: []string{"O segurança do shopping achou estranho alguém entrar no provador com um saco de dinheiro"}, effects: []effect{stain(), delay(15 * time.Minute)}},
				}},
				{label: "🥸 Vestir o kit disfarce", risk: lowRisk, uses: "kit-disfarce", endings: []ending{
					{weight: 100, texts: []string{"Bigode, peruca e óculos por cima do Chaves. Agora nem a polícia sabe quem é quem, e o próximo assalto já pode sair"}, effects: []effect{refresh()}},
				}},
			},
		},
		{
			id:     "crime-assalto-refem-voluntario",
			title:  "O refém voluntário",
			places: []string{"!assalto"},
			intros: []string{
				"No meio do assalto, um senhorzinho levantou a mão: \"Me leva de refém? Eu sempre quis aparecer no Cidade Alerta\"",
				"Uma senhora da fila do banco se ofereceu pra ser refém de {@}, já ajeitando o cabelo pra câmera do helicóptero",
			},
			choices: []choice{
				{label: "🧓 Levar junto na fuga", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"O refém conhecia todos os atalhos do bairro e ainda indicou um esconderijo bom"}, effects: []effect{earn(80, 200)}},
					{weight: 40, texts: []string{"O refém deu entrevista pro helicóptero e falou o nome completo de {@}"}, effects: []effect{nabbed()}},
					{weight: 15, texts: []string{"O refém se empolgou e deu uma bengalada em {@} achando que era cena"}, effects: []effect{injure(10, 25)}},
				}},
				{label: "🙅 Recusar com educação", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} recusou e o senhorzinho ficou chateado, reclamando na fila"}},
				}},
				{label: "✍️ Dar um autógrafo e um trocado", risk: lowRisk, price: cost{percent: 1, floor: 30}, endings: []ending{
					{weight: 100, texts: []string{"O senhorzinho guardou o autógrafo no bolso da camisa, emocionado, e desejou boa sorte"}, effects: []effect{enchant("sorte")}},
				}},
				{label: "🏦 Explicar que refém atrasa", risk: lowRisk, needs: isJob("assaltante de banco"), endings: []ending{
					{weight: 100, texts: []string{"{@} explicou com calma, como bom profissional, e o senhorzinho ainda segurou a porta na saída. A fuga foi limpa"}, effects: []effect{earn(30, 80), refresh()}},
				}},
			},
		},
		{
			id:     "crime-quadrilha-novato",
			title:  "O novato do bonde",
			places: []string{"!quadrilha"},
			when:   wentWell,
			intros: []string{
				"O sobrinho do Magrão entrou pro bonde hoje e filmou o serviço inteiro em {alvo} com o celular na vertical",
				"O novato da quadrilha postou nos stories: \"Primeiro dia no trampo com os parças\". No fundo, dá pra ver {alvo}",
			},
			choices: []choice{
				{label: "📱 Apagar o vídeo na marra", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} apagou o vídeo, a lixeira do celular e o backup na nuvem"}},
					{weight: 40, texts: []string{"O novato mordeu a mão de {@} pra salvar o vídeo"}, effects: []effect{injure(5, 15)}},
				}},
				{label: "🎓 Ensinar o novato", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} deu uma aula sobre discrição e o novato anotou tudo no caderninho"}, effects: []effect{enchant("abrecaminho")}},
					{weight: 40, texts: []string{"A aula durou a tarde toda e o novato ainda não entendeu"}, effects: []effect{delay(15 * time.Minute)}},
				}},
				{label: "🚫 Expulsar do bonde", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"O novato foi embora chorando, mas apagou o vídeo antes"}},
					{weight: 45, texts: []string{"O novato expulso postou o vídeo de vingança"}, effects: []effect{stain()}},
				}},
				{label: "💸 Pagar pro novato sumir", risk: lowRisk, price: cost{percent: 3, floor: 60}, endings: []ending{
					{weight: 100, texts: []string{"O novato pegou o dinheiro, apagou o vídeo e foi comprar um tênis"}},
				}},
			},
		},
		{
			id:     "crime-quadrilha-dica-do-cofre",
			title:  "A dica do comparsa",
			places: []string{"!quadrilha"},
			when:   wentWell,
			intros: []string{
				"Na saída do serviço, um comparsa cochichou pra {@}: \"Sabia que {alvo} guarda um cofre na parede do quarto?\"",
				"O Pezão contou que já foi na casa de {alvo} consertar o chuveiro e viu um cofre atrás do quadro da Santa Ceia",
			},
			choices: []choice{
				{label: "🏠 Ir atrás do cofre agora", risk: highRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} e o Pezão foram direto pra casa de {alvo}, que ainda não tinha chegado"}, next: "crime-quadrilha-cofre-de-casa"},
				}},
				{label: "🤐 Deixar quieto", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} achou que já tava bom por hoje. Ganância é o que mais prende bandido"}},
				}},
				{label: "📍 Vender a dica", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"Um bonde rival pagou bem pela informação"}, effects: []effect{earn(30, 80)}},
					{weight: 40, texts: []string{"O bonde rival foi pego e contou de quem veio a dica"}, effects: []effect{stain()}},
				}},
				{label: "🤝 Devolver um trocado pra vítima", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} achou que tava pesando a mão e devolveu um trocado escondido pra {alvo}"}, effects: []effect{refund(3, 8), enchant("protecao")}},
				}},
			},
		},
		{
			id:     "crime-quadrilha-cofre-de-casa",
			title:  "O cofre atrás da Santa Ceia",
			rarity: rareEvent,
			stage:  true,
			places: []string{"!quadrilha"},
			intros: []string{
				"O quadro da Santa Ceia tá torto e atrás dele tem um cofre. No pé da cama, um rottweiler dorme roncando",
			},
			choices: []choice{
				{label: "🔨 Arrombar no braço", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"Depois de vinte marteladas, o cofre abriu. Tava cheio"}, effects: []effect{earn(150, 300), stain()}},
					{weight: 35, texts: []string{"O barulho acordou o rottweiler"}, effects: []effect{injure(15, 35)}},
					{weight: 20, texts: []string{"{alvo} chegou em casa bem na hora, com a polícia junto"}, effects: []effect{nabbed()}},
					{weight: 5, texts: []string{"O rottweiler acordou de mau humor e {@} não saiu da casa"}, effects: []effect{perish()}},
				}},
				{label: "🦴 Distrair o cachorro e abrir", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"O rottweiler ficou com o osso e {@} acertou a senha: era o aniversário de {alvo}"}, effects: []effect{earn(60, 150)}},
					{weight: 45, texts: []string{"O rottweiler preferiu a perna de {@} ao osso"}, effects: []effect{injure(10, 25)}},
				}},
				{label: "🛠️ Usar a furadeira de cofre", risk: midRisk, needs: holds("furadeira"), endings: []ending{
					{weight: 80, texts: []string{"A furadeira abriu o cofre em dois minutos, sem acordar ninguém"}, effects: []effect{earn(150, 300), stain()}},
					{weight: 20, texts: []string{"A furadeira fez barulho de dentista e o vizinho chamou a polícia"}, effects: []effect{nabbed()}},
				}},
				{label: "🚪 Desistir na porta", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} olhou pro tamanho do rottweiler e achou melhor ir embora"}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, o Pezão abriu o cofre sozinho e sumiu com tudo"}},
		},
		{
			id:     "crime-mercado-leilao-de-muamba",
			title:  "O leilão de muamba",
			rarity: rareEvent,
			places: []string{"!mercadonegro"},
			intros: []string{
				"No fundo do galpão tá rolando um leilão clandestino. O lote da vez é uma caixa coberta com lona: \"Ninguém sabe o que tem, nem eu\"",
				"Um leiloeiro de terno brilhante subiu num caixote: \"Lote misterioso, direto da Receita Federal, quem dá mais?\". {@} tá na primeira fila",
			},
			choices: []choice{
				{label: "🪧 Dar o lance", risk: midRisk, price: cost{percent: 6, floor: 200}, endings: []ending{
					{weight: 50, texts: []string{"{@} arrematou. Debaixo da lona tinha coisa boa"}, effects: []effect{gain(1, "videogame")}},
					{weight: 30, texts: []string{"{@} arrematou uma caixa de Rolex do Paraguai e perfume falsificado"}, effects: []effect{gain(2, "rolex-do-paraguai"), gain(2, "perfume")}},
					{weight: 20, texts: []string{"O leiloeiro era da Receita Federal mesmo, e o lote era isca"}, effects: []effect{nabbed()}},
				}},
				{label: "👀 Só assistir", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} assistiu o leilão inteiro comendo amendoim. Melhor que novela"}},
					{weight: 30, texts: []string{"No chão do galpão, {@} achou umas notas que caíram do bolso de um arrematante"}, effects: []effect{earn(20, 50)}},
				}},
				{label: "🔨 Assumir o martelo", risk: midRisk, needs: isJob("leiloeiro"), endings: []ending{
					{weight: 80, texts: []string{"{@} tomou o martelo e inflou os lances com uma lábia de profissional. A comissão foi gorda"}, effects: []effect{earn(100, 250), stain()}},
					{weight: 20, texts: []string{"Um arrematante percebeu os lances fantasmas e veio tirar satisfação"}, effects: []effect{injure(10, 20)}},
				}},
				{label: "🚪 Sair antes da batida", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} sentiu cheiro de polícia e saiu de fininho. Dez minutos depois, o galpão tava cercado"}, effects: []effect{refresh()}},
				}},
			},
		},
		{
			id:     "crime-mercado-farmacia-do-beco",
			title:  "A farmácia do beco",
			places: []string{"!mercadonegro"},
			intros: []string{
				"Uma barraquinha com uma cruz verde pintada à mão oferece pra {@} \"remédio sem receita, bomba e vitamina de cavalo\"",
				"O farmacêutico do mercado negro, de jaleco encardido, chacoalhou uma ampola pra {@}: \"Vai sair daqui igual o Hulk\"",
			},
			choices: []choice{
				{label: "💉 Comprar a bomba e aplicar", risk: midRisk, price: cost{percent: 4, floor: 150}, endings: []ending{
					{weight: 70, texts: []string{"Dez minutos depois, a camisa de {@} já não fechava mais"}, effects: []effect{buzz("anabolizante")}},
					{weight: 30, texts: []string{"A ampola era vencida desde 2009 e a bunda de {@} inchou igual melancia"}, effects: []effect{injure(10, 25)}},
				}},
				{label: "🐴 Provar a vitamina de cavalo", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"A vitamina de cavalo deu um gás que {@} nunca sentiu na vida"}, effects: []effect{buzz("rebite")}},
					{weight: 42, texts: []string{"{@} passou mal na hora e relinchou a tarde inteira"}, effects: []effect{injure(15, 35)}},
					{weight: 8, texts: []string{"Era vitamina de cavalo mesmo, dose de cavalo. O coração de {@} não aguentou"}, effects: []effect{perish()}},
				}},
				{label: "🩹 Pedir uma amostra grátis", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"O farmacêutico deu um curativo de brinde e um tapinha nas costas"}, effects: []effect{gain(1, "curativo")}},
					{weight: 40, texts: []string{"\"Amostra grátis é na farmácia do shopping\", disse ele, virando a cara"}},
				}},
				{label: "🩺 Denunciar o charlatão", risk: midRisk, needs: inArea(areaSaude), endings: []ending{
					{weight: 65, texts: []string{"{@} mostrou o CRM e o charlatão fugiu largando a barraca. Ficou um dinheirinho no caixa"}, effects: []effect{earn(40, 120), enchant("justica")}},
					{weight: 35, texts: []string{"Os seguranças do mercado negro não gostaram nada de ver médico por ali"}, effects: []effect{injure(10, 25)}},
				}},
			},
		},
	},
}
