package economy

import "time"

var craftEvents = eventBook{
	name:   "💼 Trampo",
	places: []string{"!trabalhar"},
	scenarios: []scenario{
		{
			id:    "oficio-campo-gafanhotos",
			title: "A nuvem de gafanhotos",
			areas: []int{areaCampo},
			intros: []string{
				"{@} tava no meio do trampo quando o céu escureceu: uma nuvem de gafanhoto vindo direto pra roça",
				"Primeiro pareceu barulho de helicóptero. Era gafanhoto. Uns três milhões, todos com fome, e {@} bem no caminho deles",
			},
			choices: []choice{
				{label: "🔥 Fazer fumaça pra espantar", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"A fumaceira funcionou e os gafanhotos desviaram pra roça do vizinho chato"}, effects: []effect{earn(60, 150)}},
					{weight: 30, texts: []string{"Os bichos foram embora, mas {@} respirou tanta fumaça que saiu tossindo igual chaminé de fogão a lenha"}, effects: []effect{injure(10, 20), earn(20, 50)}},
					{weight: 15, texts: []string{"O fogo escapou, queimou a cerca e quase pegou no galinheiro"}, effects: []effect{lose(3, 8)}},
				}},
				{label: "🐔 Soltar as galinhas em cima deles", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"As galinhas fizeram o banquete da vida delas e a roça ficou intacta"}, effects: []effect{earn(50, 120)}},
					{weight: 40, texts: []string{"As galinhas viram o tamanho da nuvem e foram as primeiras a fugir"}, effects: []effect{lose(2, 5)}},
				}},
				{label: "🌾 Colher o que der na correria", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} salvou o que deu antes da nuvem chegar"}, effects: []effect{findLoot(), earn(20, 50)}},
				}},
				{label: "🙏 Rezar pra São Pedro mandar chuva", risk: lowRisk, endings: []ending{
					{weight: 50, texts: []string{"Caiu um toró do nada e os gafanhotos foram embora. O povo jura que foi milagre"}, effects: []effect{enchant("sorte")}},
					{weight: 50, texts: []string{"São Pedro tava de folga. Os gafanhotos comeram até a palha do chapéu de {@}"}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, a nuvem passou e almoçou metade da plantação"}, effects: []effect{lose(2, 5)}},
		},
		{
			id:    "oficio-campo-vaca-fugiu",
			title: "A Mimosa fugiu",
			areas: []int{areaCampo},
			intros: []string{
				"A Mimosa, a vaca mais mimada da região, arrombou a porteira e saiu trotando pela estrada como se tivesse hora marcada",
				"Alguém deixou a porteira aberta e a Mimosa aproveitou. Ela tá fugindo pela BR e {@} é o único por perto",
			},
			choices: []choice{
				{label: "🤠 Laçar ela no braço", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Na terceira tentativa, o laço pegou e {@} trouxe a Mimosa de volta, toda orgulhosa"}, effects: []effect{earn(60, 130)}},
					{weight: 35, texts: []string{"A Mimosa arrastou {@} uns 200 metros de estrada de terra"}, effects: []effect{injure(15, 30)}},
					{weight: 15, texts: []string{"{@} laçou o bicho errado: era o touro bravo do vizinho"}, effects: []effect{injure(25, 40), delay(20 * time.Minute)}},
				}},
				{label: "🌽 Atrair com um balde de milho", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"A Mimosa não resistiu ao milho e voltou sozinha, abanando o rabo"}, effects: []effect{earn(30, 60)}},
					{weight: 30, texts: []string{"A Mimosa comeu o milho todo e continuou fugindo, agora com energia de sobra"}},
				}},
				{label: "📢 Chamar a vizinhança pra ajudar", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"Meio bairro correu atrás da vaca, virou festa e no fim sobrou pamonha pra todo mundo"}, effects: []effect{gain(2, "pamonha")}},
				}},
				{label: "🐂 Ir atrás montado no touro", risk: midRisk, needs: isJob("peão de rodeio"), endings: []ending{
					{weight: 75, texts: []string{"{@} montou em pelo no touro, alcançou a Mimosa e ainda foi aplaudido na estrada"}, effects: []effect{earn(100, 200)}},
					{weight: 25, texts: []string{"O touro achou melhor derrubar {@} no meio do pasto"}, effects: []effect{injure(20, 35)}},
				}},
			},
			idle: ending{texts: []string{"A Mimosa sumiu no horizonte. Foi vista depois dançando na festa de peão da cidade vizinha"}},
		},
		{
			id:    "oficio-campo-grileiro",
			title: "O grileiro de botina",
			areas: []int{areaCampo},
			intros: []string{
				"Um sujeito de botina e chapéu chegou com um papel 'oficial' dizendo que a terra onde {@} trabalha agora é dele",
				"Parou uma caminhonete preta na porteira. Desceu um grileiro conhecido da região, balançando uma escritura que ninguém nunca viu",
			},
			choices: []choice{
				{label: "📜 Pedir pra ver a escritura", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"O papel era xerox de xerox com carimbo feito de batata. O grileiro foi embora resmungando"}, effects: []effect{earn(20, 50)}},
					{weight: 40, texts: []string{"O papel parecia de verdade e {@} perdeu o resto do dia no cartório"}, effects: []effect{delay(20 * time.Minute)}},
				}},
				{label: "😤 Botar ele pra correr", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"O grileiro assobiou e desceram três capangas da caçamba da caminhonete"}, next: "oficio-campo-grileiro-capangas"},
					{weight: 45, texts: []string{"Na base do grito e da enxada, {@} fez o grileiro sair cantando pneu"}, effects: []effect{earn(40, 90)}},
				}},
				{label: "🚓 Chamar a polícia", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"A viatura chegou e o grileiro tinha três mandados em aberto. O povo fez churrasco pra comemorar"}, effects: []effect{earn(30, 70)}},
					{weight: 30, texts: []string{"A polícia demorou tanto que o grileiro cansou de esperar e foi embora"}},
				}},
				{label: "🔫 Mostrar o ferro", risk: highRisk, needs: armedUp, endings: []ending{
					{weight: 70, texts: []string{"O grileiro olhou pro ferro, olhou pra {@} e lembrou que tinha um compromisso urgente"}, effects: []effect{earn(50, 110), stain()}},
					{weight: 30, texts: []string{"O grileiro também tava armado. Saiu tiro pra todo lado e um pegou de raspão em {@}"}, effects: []effect{injure(25, 45), stain()}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, o grileiro cercou metade do terreno com arame farpado"}, effects: []effect{delay(15 * time.Minute)}},
		},
		{
			id:    "oficio-campo-grileiro-capangas",
			title: "Os capangas do grileiro",
			stage: true,
			intros: []string{
				"Três capangas mal-encarados cercaram {@}. O grileiro sorriu: 'E aí, vai assinar ou vai precisar de ajuda pra segurar a caneta?'",
				"Os capangas estalaram os dedos ao mesmo tempo, igual ensaio de quadrilha junina. O grileiro estendeu a caneta pra {@}",
			},
			choices: []choice{
				{label: "🌽 Sumir no milharal", risk: midRisk, endings: []ending{
					{weight: 65, favor: func(p Profession) float64 { return p.escapeBonus }, texts: []string{"{@} sumiu no milharal igual saci, e os capangas se perderam lá dentro até o anoitecer"}},
					{weight: 35, texts: []string{"Os capangas pegaram {@} na saída do milharal"}, effects: []effect{injure(20, 35)}},
				}},
				{label: "👊 Sair no braço com os três", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"{@} distribuiu bofetada igual novela das nove e os capangas fugiram junto com o grileiro"}, effects: []effect{earn(80, 160)}},
					{weight: 50, texts: []string{"Três contra um não dá. {@} apanhou até da botina do grileiro"}, effects: []effect{injure(30, 50)}},
					{weight: 10, texts: []string{"Um dos capangas puxou a peixeira e a coisa ficou feia"}, effects: []effect{injure(55, 80)}},
				}},
				{label: "✍️ Assinar o papel", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} assinou sem ler. Depois descobriu que tinha vendido a própria sombra"}, effects: []effect{lose(2, 5)}},
				}},
				{label: "🚔 Dar voz de prisão", risk: midRisk, needs: lawman, endings: []ending{
					{weight: 80, texts: []string{"{@} mostrou o distintivo, os capangas largaram tudo e o grileiro foi em cana"}, effects: []effect{earn(100, 200)}},
					{weight: 20, texts: []string{"Os capangas não respeitaram o distintivo e partiram pra cima"}, effects: []effect{injure(20, 35)}},
				}},
			},
			idle: ending{texts: []string{"{@} travou de nervoso. Os capangas pegaram a carteira e foram embora rindo"}, effects: []effect{lose(2, 5)}},
		},
		{
			id:    "oficio-campo-colmeia",
			title: "A colmeia do tamanho de uma geladeira",
			areas: []int{areaCampo},
			intros: []string{
				"Num tronco oco no fundo do terreno, {@} achou uma colmeia do tamanho de uma geladeira, pingando mel",
				"Um zumbido grave vinha da mangueira velha. Dentro dela, a maior colmeia que {@} já viu na vida",
			},
			choices: []choice{
				{label: "🍯 Tirar o mel na mão", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"{@} saiu com favo pra encher um balde e só levou três picadas"}, effects: []effect{gain(2, "favo-de-mel"), injure(5, 10)}},
					{weight: 45, texts: []string{"As abelhas declararam guerra. {@} correu até o rio e mergulhou de roupa e tudo"}, effects: []effect{injure(20, 40)}},
					{weight: 10, texts: []string{"Era abelha africana. {@} virou almofada de alfinete"}, effects: []effect{injure(45, 70)}},
				}},
				{label: "💨 Usar fumaça antes", risk: midRisk, endings: []ending{
					{weight: 70, texts: []string{"A fumaça acalmou o enxame e o mel saiu limpinho"}, effects: []effect{gain(1, "favo-de-mel"), earn(20, 50)}},
					{weight: 30, texts: []string{"A fumaça só deixou as abelhas mais bravas"}, effects: []effect{injure(10, 25)}},
				}},
				{label: "🐝 Vestir a roupa de apicultor", risk: lowRisk, needs: holds("roupa-de-apicultor"), endings: []ending{
					{weight: 85, texts: []string{"Protegido dos pés à cabeça, {@} colheu com a calma de quem tá num spa"}, effects: []effect{gain(3, "favo-de-mel"), earn(30, 60)}},
					{weight: 15, texts: []string{"Tinha uma abelha DENTRO da roupa. Mesmo assim {@} saiu com mel"}, effects: []effect{injure(5, 10), gain(2, "favo-de-mel")}},
				}},
				{label: "🚶 Deixar as abelhas em paz", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"As abelhas agradeceram polinizando a roça inteira, e a colheita veio caprichada"}, effects: []effect{earn(20, 50)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} decidia, um vizinho esperto passou com um balde e levou o mel todo"}},
		},
		{
			id:    "oficio-campo-seca",
			title: "A seca braba",
			areas: []int{areaCampo},
			intros: []string{
				"Faz 40 dias que não cai uma gota. O açude virou poça e a plantação tá pedindo socorro",
				"O sol tá rachando o chão da roça. Se não arrumar água hoje, {@} perde a colheita",
			},
			choices: []choice{
				{label: "🚛 Pagar um caminhão-pipa", risk: midRisk, price: cost{percent: 8, floor: 80}, endings: []ending{
					{weight: 70, texts: []string{"O caminhão-pipa salvou a lavoura e a colheita veio farta"}, effects: []effect{earn(120, 250)}},
					{weight: 30, texts: []string{"Era golpe: o caminhão veio vazio e o motorista sumiu com o dinheiro"}},
				}},
				{label: "🕳️ Cavar um poço", risk: midRisk, endings: []ending{
					{weight: 40, texts: []string{"No segundo metro, brotou água limpinha!"}, effects: []effect{earn(60, 140), refresh()}},
					{weight: 40, texts: []string{"Cavou o dia inteiro e só achou pedra e calo na mão"}, effects: []effect{injure(5, 15), delay(15 * time.Minute)}},
					{weight: 20, texts: []string{"Em vez de água, subiu um líquido preto e fedorento. {@} jura que ficou rico"}, effects: []effect{gain(1, "petroleo")}},
				}},
				{label: "🌧️ Fazer simpatia pra chover", risk: lowRisk, endings: []ending{
					{weight: 40, texts: []string{"Choveu! Ninguém sabe se foi a simpatia, mas {@} agora se acha pajé"}, effects: []effect{enchant("sorte")}},
					{weight: 60, texts: []string{"Não choveu, e a cachaça da simpatia era a última da casa"}},
				}},
				{label: "📞 Ligar pra prefeitura", risk: lowRisk, endings: []ending{
					{weight: 30, texts: []string{"A prefeitura mandou um caminhão-pipa no mesmo dia. Milagre de ano de eleição"}, effects: []effect{earn(40, 90)}},
					{weight: 70, texts: []string{"A prefeitura prometeu mandar água 'semana que vem'. Do ano que vem"}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, o sol fez o resto e a colheita murchou"}, effects: []effect{lose(2, 5)}},
		},
		{
			id:     "oficio-campo-rodeio",
			title:  "O touro Bandido",
			rarity: rareEvent,
			areas:  []int{areaCampo},
			intros: []string{
				"O locutor da festa de peão gritou o nome de {@} no alto-falante: falta um peão pra montar o Bandido, o touro que ninguém nunca parou por 8 segundos",
				"Na festa do peão da cidade, o Bandido já derrubou 40 peões. A arquibancada inteira começou a gritar o nome de {@}",
			},
			choices: []choice{
				{label: "🐂 Montar o Bandido", risk: highRisk, endings: []ending{
					{weight: 35, texts: []string{"OITO SEGUNDOS! {@} foi carregado nos ombros e levou o prêmio do rodeio"}, effects: []effect{earn(400, 800), gain(1, "chapeu-de-palha")}},
					{weight: 45, texts: []string{"O Bandido jogou {@} tão alto que deu pra ver a cidade vizinha"}, effects: []effect{injure(30, 50)}},
					{weight: 15, texts: []string{"{@} ficou pendurado no touro igual roupa no varal"}, effects: []effect{injure(50, 75)}},
					{weight: 5, texts: []string{"O Bandido não perdoou ninguém hoje"}, effects: []effect{perish()}},
				}},
				{label: "🎤 Pegar o microfone do locutor", risk: midRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} narrou o rodeio inteiro, o povo amou e o dono da festa pagou cachê"}, effects: []effect{earn(150, 300)}},
					{weight: 30, texts: []string{"{@} errou o nome de todos os peões e foi vaiado até ir embora"}, effects: []effect{lose(2, 5)}},
				}},
				{label: "🎰 Apostar contra o Bandido", risk: midRisk, price: cost{percent: 5, floor: 50}, endings: []ending{
					{weight: 40, texts: []string{"Um peão aguentou os 8 segundos e a aposta pagou cinco vezes"}, effects: []effect{earn(300, 600)}},
					{weight: 60, texts: []string{"O Bandido derrubou todo mundo de novo e a aposta virou fumaça"}},
				}},
				{label: "🍺 Ficar só na arquibancada", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} curtiu o show, comeu pastel e voltou pra casa feliz da vida"}, effects: []effect{heal(15)}},
				}},
			},
		},
		{
			id:    "oficio-comercio-devolucao",
			title: "A devolução sem vergonha",
			areas: []int{areaComercio},
			intros: []string{
				"Uma cliente quer devolver um vestido 'sem uso'. Ele tem cheiro de churrasco, mancha de vinho e tem foto dela com ele no casamento da prima",
				"Um cliente chegou querendo trocar uma TV 'que veio com defeito'. A tela tem a marca do controle remoto que ele arremessou no jogo do Brasil",
			},
			choices: []choice{
				{label: "📸 Mostrar a prova pro cliente", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"O cliente ficou vermelho igual pimenta, pediu desculpa e ainda levou outra coisa"}, effects: []effect{earn(40, 90)}},
					{weight: 40, texts: []string{"O cliente fez barraco, gravou vídeo e o vídeo viralizou contra {@}"}, effects: []effect{lose(2, 5)}},
				}},
				{label: "🙂 Aceitar a devolução", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} engoliu o sapo e devolveu o dinheiro. O cliente prometeu voltar, tomara que não"}, effects: []effect{lose(2, 4)}},
				}},
				{label: "🔄 Revender como seminovo", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"Vendeu de novo como 'seminovo com história'. Lucro em dobro"}, effects: []effect{earn(60, 140)}},
					{weight: 45, texts: []string{"O novo comprador achou o defeito e chamou o Procon"}, effects: []effect{lose(3, 6), delay(15 * time.Minute)}},
				}},
			},
			idle: ending{texts: []string{"{@} demorou tanto pra decidir que o cliente foi reclamar no gerente do shopping"}},
		},
		{
			id:    "oficio-comercio-nota-falsa",
			title: "A nota de duzentos",
			areas: []int{areaComercio},
			intros: []string{
				"Um cliente pagou com uma nota de 200 que tem o lobo-guará de óculos escuros e escrito 'BRAZIL' com Z",
				"A nota de 100 que o cliente entregou é do tamanho de um guardanapo e a garoupa tá sorrindo pra {@}",
			},
			choices: []choice{
				{label: "🔍 Recusar a nota", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O cliente saiu xingando, mas sem levar nada"}},
					{weight: 30, texts: []string{"Era nota de verdade, só tava velha. O cliente ofendido nunca mais voltou"}, effects: []effect{lose(2, 4)}},
				}},
				{label: "🤑 Aceitar e passar pra frente", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} passou a nota no troco de outro cliente e ninguém percebeu"}, effects: []effect{earn(80, 180), stain()}},
					{weight: 35, texts: []string{"O próximo cliente percebeu e chamou a polícia na hora"}, effects: []effect{nabbed()}},
					{weight: 15, texts: []string{"A nota voltou pras mãos de {@} três vezes na mesma semana, igual maldição"}, effects: []effect{lose(3, 6)}},
				}},
				{label: "🚨 Chamar a polícia", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"O falsificador foi preso em flagrante e a loja saiu no jornal do bairro"}, effects: []effect{earn(40, 100)}},
					{weight: 40, texts: []string{"O cliente fugiu antes da polícia chegar, mas esqueceu a carteira no balcão"}, effects: []effect{gain(1, "carteira-perdida")}},
				}},
			},
			idle: ending{texts: []string{"{@} ficou encarando a nota tanto tempo que o cliente pegou de volta e foi embora"}},
		},
		{
			id:    "oficio-comercio-fiscal",
			title: "O fiscal da Receita",
			areas: []int{areaComercio},
			intros: []string{
				"Um fiscal de crachá e prancheta entrou e pediu as notas fiscais dos últimos cinco anos. A última nota que {@} emitiu foi no tempo do orelhão",
				"Chegou um fiscal da Receita com cara de quem não ri desde 1998, pedindo o livro-caixa de {@}",
			},
			choices: []choice{
				{label: "📂 Mostrar a papelada bagunçada", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O fiscal se perdeu na bagunça, ficou com dor de cabeça e foi embora"}},
					{weight: 50, texts: []string{"Ele achou tudo o que faltava e caprichou na multa"}, effects: []effect{lose(5, 12)}},
				}},
				{label: "✉️ Oferecer café e um envelope", risk: highRisk, price: cost{percent: 6, floor: 60}, endings: []ending{
					{weight: 60, texts: []string{"O fiscal guardou o envelope, tomou o café e escreveu 'tudo certo' na prancheta"}},
					{weight: 40, texts: []string{"Era fiscal honesto. Oferecer propina pra servidor dá cadeia"}, effects: []effect{nabbed()}},
				}},
				{label: "🏃 Baixar a porta e fugir", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} sumiu pelos fundos e o fiscal ficou meia hora batendo no vidro"}, effects: []effect{delay(20 * time.Minute)}},
					{weight: 40, texts: []string{"O fiscal anotou o CNPJ e a multa veio em dobro"}, effects: []effect{lose(8, 15)}},
				}},
				{label: "🪪 Pedir pra ver o crachá dele", risk: lowRisk, endings: []ending{
					{weight: 50, texts: []string{"O crachá era de plástico com foto de ator de novela. Era golpista e saiu correndo"}, effects: []effect{earn(30, 70)}},
					{weight: 50, texts: []string{"Crachá verdadeiro. O fiscal ficou ofendido e caprichou na auditoria"}, effects: []effect{lose(2, 5)}},
				}},
			},
		},
		{
			id:    "oficio-comercio-concorrente",
			title: "A guerra de preços",
			areas: []int{areaComercio},
			intros: []string{
				"A loja do outro lado da rua pendurou uma faixa: 'TUDO PELA METADE DO PREÇO DO VIZINHO'. O vizinho é {@}",
				"Abriu um concorrente do lado, com fila na porta, música alta e um boneco de posto dançando. Os clientes de {@} tão todos lá",
			},
			choices: []choice{
				{label: "📉 Baixar os preços também", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Guerra de preço! O bairro inteiro veio comprar e {@} lucrou no volume"}, effects: []effect{earn(80, 170)}},
					{weight: 50, texts: []string{"Vendeu muito e não lucrou nada. Trabalhou o dia todo de graça"}, effects: []effect{lose(2, 5)}},
				}},
				{label: "📣 Inventar uma promoção maluca", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"'Compre um e leve um sogro' viralizou e a loja lotou"}, effects: []effect{earn(100, 200)}},
					{weight: 40, texts: []string{"A promoção só atraiu gente pedindo fiado"}, effects: []effect{lose(3, 6)}},
				}},
				{label: "🕵️ Espionar o concorrente", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} descobriu que a promoção era maquiagem: os preços tinham subido antes"}, effects: []effect{earn(50, 120)}},
					{weight: 45, texts: []string{"O concorrente pegou {@} fotografando a vitrine e saiu no tapa"}, effects: []effect{injure(10, 20)}},
				}},
				{label: "🐀 Soltar ratos na loja dele", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"A vigilância sanitária interditou o concorrente. Maldade pura, mas funcionou"}, effects: []effect{earn(150, 300), stain()}},
					{weight: 35, texts: []string{"Os ratos acharam a loja de {@} mais aconchegante"}, effects: []effect{lose(4, 8)}},
					{weight: 15, texts: []string{"O concorrente filmou tudo e chamou a polícia"}, effects: []effect{nabbed()}},
				}},
			},
		},
		{
			id:     "oficio-comercio-caixa-heranca",
			title:  "A caixa lacrada da herança",
			rarity: rareEvent,
			areas:  []int{areaComercio},
			intros: []string{
				"Um velhinho chegou com uma caixa lacrada de herança e quer vender pra {@} 'pelo preço que achar justo'. Lá dentro, alguma coisa chacoalha",
				"Uma senhora de chapéu de renda deixou no balcão uma caixa velha amarrada com barbante. 'Era do meu falecido. Não tenho coragem de abrir'",
			},
			choices: []choice{
				{label: "💰 Comprar a caixa", risk: midRisk, price: cost{percent: 10, floor: 100}, endings: []ending{
					{weight: 35, texts: []string{"Dentro tinha um baú de pirata de verdade, com fechadura e tudo!"}, effects: []effect{gain(1, "bau-do-tesouro")}},
					{weight: 30, texts: []string{"Dentro tinha um álbum da Copa completo, com as figurinhas mais raras"}, effects: []effect{gain(1, "album-da-copa"), earn(100, 200)}},
					{weight: 25, texts: []string{"Era só tralha e um bilhete: 'obrigado pela grana, trouxa'"}, effects: []effect{gain(1, "bilhete-esquecido")}},
					{weight: 10, texts: []string{"Era uma caixa de Pandora: saiu uma praga lá de dentro"}, effects: []effect{enchant("azar")}},
				}},
				{label: "🔨 Leiloar pros clientes", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"O povo disputou a caixa no grito e {@} ficou com a comissão"}, effects: []effect{earn(150, 300)}},
					{weight: 40, texts: []string{"Ninguém deu lance e a caixa voltou pra casa com a dona, toda triste"}},
				}},
				{label: "🙅 Recusar a caixa", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"A caixa foi embora. Uma semana depois {@} viu a família dirigindo um carro zero"}},
				}},
				{label: "🔍 Avaliar com olho clínico", risk: lowRisk, needs: isJob("penhorista", "leiloeiro", "dono de brechó"), endings: []ending{
					{weight: 100, texts: []string{"{@} examinou a caixa, pagou uma mixaria e lucrou alto na revenda"}, effects: []effect{earn(300, 500)}},
				}},
			},
		},
		{
			id:    "oficio-comercio-fiado",
			title: "O caderninho do fiado",
			areas: []int{areaComercio},
			intros: []string{
				"O Seu Jorge, cliente há 20 anos, quer levar mais uma no fiado. O caderninho dele já tem mais página que a Bíblia",
				"A Dona Neide chegou com o sorriso de sempre e a frase de sempre: 'Bota na conta, meu bem'. A conta dela já dá um carro",
			},
			choices: []choice{
				{label: "📒 Anotar mais uma", risk: midRisk, endings: []ending{
					{weight: 40, texts: []string{"Ganhou na quina e veio pagar tudo de uma vez, com juros e um bolo de presente!"}, effects: []effect{earn(150, 300)}},
					{weight: 60, texts: []string{"Mais uma página no caderninho. Agradeceu e sumiu por três meses"}, effects: []effect{lose(2, 4)}},
				}},
				{label: "🙅 Negar o fiado", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"Entendeu o recado e pagou à vista, resmungando"}, effects: []effect{earn(20, 50)}},
					{weight: 30, texts: []string{"Espalhou pro bairro que {@} é mão de vaca e o movimento caiu"}, effects: []effect{lose(2, 4)}},
				}},
				{label: "🤝 Perdoar a dívida inteira", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"Teve choro, abraço e o bairro todo ficou sabendo da bondade. Karma bom é investimento"}, effects: []effect{enchant("sorte")}},
					{weight: 30, texts: []string{"Achou lindo, agradeceu, e no dia seguinte já pediu fiado de novo"}},
				}},
			},
		},
		{
			id:    "oficio-lei-envelope",
			title: "O envelope do empresário",
			areas: []int{areaSeguranca},
			intros: []string{
				"Um empresário de gravata deixou um envelope gordo na mesa de {@}: 'É só esquecer aquele caminhão parado no lugar errado, combinado?'",
				"Um sujeito de relógio dourado apertou a mão de {@} e deixou um envelope junto. 'Pra um cafezinho. Você não viu nada hoje, né?'",
			},
			choices: []choice{
				{label: "💼 Aceitar o envelope", risk: highRisk, endings: []ending{
					{weight: 55, texts: []string{"Ninguém viu nada, e o envelope tava recheado"}, effects: []effect{earn(200, 400), stain()}},
					{weight: 30, texts: []string{"Era operação da corregedoria! O empresário era agente disfarçado"}, effects: []effect{nabbed()}},
					{weight: 15, texts: []string{"{@} aceitou, gastou tudo no mesmo dia e ficou com um peso esquisito nas costas"}, effects: []effect{earn(100, 200), enchant("encosto")}},
				}},
				{label: "🙅 Recusar e mandar embora", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} recusou com classe. O chefe soube e elogiou na frente de todo mundo"}, effects: []effect{earn(20, 50)}},
				}},
				{label: "🚔 Dar voz de prisão", risk: midRisk, needs: lawman, endings: []ending{
					{weight: 80, texts: []string{"Corrupção ativa! O empresário saiu algemado e {@} virou herói da delegacia"}, effects: []effect{earn(100, 220)}},
					{weight: 20, texts: []string{"O empresário tinha advogado caro e saiu antes de {@} terminar o café"}, effects: []effect{delay(15 * time.Minute)}},
				}},
				{label: "📸 Filmar e denunciar", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"O vídeo foi parar no jornal e o empresário perdeu a licença"}, effects: []effect{earn(40, 90), enchant("justica")}},
					{weight: 40, texts: []string{"O vídeo ficou tremido e ninguém acreditou"}},
				}},
			},
		},
		{
			id:    "oficio-lei-testemunha",
			title: "A testemunha com medo",
			areas: []int{areaSeguranca},
			intros: []string{
				"Uma senhora chegou tremendo: viu um crime e quer contar, mas tá com medo de ser descoberta. Ela só confia em {@}",
				"Um menino entregou um bilhete pra {@}: 'Eu vi quem foi. Me encontra atrás da padaria às 6h. Sozinho'",
			},
			choices: []choice{
				{label: "🤝 Garantir proteção", risk: midRisk, endings: []ending{
					{weight: 70, texts: []string{"A testemunha contou tudo e o caso foi resolvido. {@} ganhou um bônus"}, effects: []effect{earn(80, 160)}},
					{weight: 30, texts: []string{"Alguém vazou o nome da testemunha e {@} teve que bancar um hotel por uns dias"}, effects: []effect{lose(3, 6)}},
				}},
				{label: "📝 Mandar preencher o formulário", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"A testemunha olhou o formulário de 14 páginas e desistiu. Caso arquivado"}},
				}},
				{label: "🕵️ Investigar por conta própria", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"{@} seguiu a pista e pegou a quadrilha no flagra. Saiu no jornal"}, effects: []effect{earn(200, 350)}},
					{weight: 40, texts: []string{"A quadrilha percebeu e deu uma surra em {@}"}, effects: []effect{injure(25, 45)}},
					{weight: 15, texts: []string{"Era emboscada num galpão abandonado"}, effects: []effect{injure(55, 80)}},
				}},
				{label: "🔫 Ir armado atrás deles", risk: highRisk, needs: armedUp, endings: []ending{
					{weight: 70, texts: []string{"Com o ferro na cintura, {@} enquadrou a quadrilha inteira"}, effects: []effect{earn(200, 350)}},
					{weight: 30, texts: []string{"Os bandidos tinham mais ferro que {@}"}, effects: []effect{injure(30, 50)}},
				}},
			},
			idle: ending{texts: []string{"A testemunha cansou de esperar e mudou de cidade. O caso esfriou"}},
		},
		{
			id:    "oficio-lei-perseguicao",
			title: "Pega ladrão!",
			areas: []int{areaSeguranca},
			intros: []string{
				"Um motoqueiro passou voando na frente de {@} com uma bolsa roubada na mão. A dona gritando: 'PEGA LADRÃO!'",
				"Um trombadinha arrancou o celular de uma turista e saiu correndo. Ele passou a dois metros de {@}",
			},
			choices: []choice{
				{label: "🏃 Correr atrás", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} correu tanto que alcançou o ladrão... num beco sem saída"}, next: "oficio-lei-perseguicao-beco"},
					{weight: 35, texts: []string{"O ladrão sumiu na esquina, mas deixou o roubo cair. {@} devolveu pra dona"}, effects: []effect{earn(40, 90)}},
					{weight: 15, texts: []string{"{@} tropeçou no meio-fio e ralou o joelho inteiro"}, effects: []effect{injure(10, 20)}},
				}},
				{label: "📞 Ligar pro 190", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"A viatura cercou o ladrão duas ruas depois"}, effects: []effect{earn(40, 80)}},
					{weight: 40, texts: []string{"Caiu na musiquinha de espera. Quando atenderam, o ladrão já tava em outro estado"}},
				}},
				{label: "📟 Acionar a central no rádio", risk: lowRisk, needs: holds("radio-comunicador"), endings: []ending{
					{weight: 90, texts: []string{"A central fechou todas as saídas e o ladrão foi pego em cinco minutos"}, effects: []effect{earn(80, 150)}},
					{weight: 10, texts: []string{"O rádio pegou a frequência de uma rádio de pagode e ninguém ouviu nada"}},
				}},
				{label: "🙈 Fingir que não viu", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} olhou pro outro lado. A vítima olhou feio pra {@} por um bom tempo"}},
				}},
			},
		},
		{
			id:    "oficio-lei-perseguicao-beco",
			title: "O beco sem saída",
			stage: true,
			intros: []string{
				"Encurralado no beco, o ladrão largou o roubo, puxou uma faca e ficou encarando {@}",
				"O ladrão parou no fim do beco, ofegante. Na mão, uma faca de cozinha tremendo",
			},
			choices: []choice{
				{label: "🗣️ Negociar a rendição", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"Depois de muita conversa, o ladrão se entregou chorando"}, effects: []effect{earn(100, 200)}},
					{weight: 40, texts: []string{"Enquanto {@} falava, o ladrão pulou o muro e sumiu"}},
				}},
				{label: "🥋 Partir pra cima", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"Um golpe só e o ladrão tava no chão. {@} virou lenda do bairro"}, effects: []effect{earn(150, 300)}},
					{weight: 40, texts: []string{"A faca pegou no braço de {@}"}, effects: []effect{injure(25, 45)}},
					{weight: 10, texts: []string{"Foi feio. A facada pegou fundo"}, effects: []effect{injure(55, 80)}},
				}},
				{label: "🔫 Sacar a arma", risk: midRisk, needs: armedUp, endings: []ending{
					{weight: 85, texts: []string{"O ladrão largou a faca na hora e levantou as mãos"}, effects: []effect{earn(150, 250)}},
					{weight: 15, texts: []string{"O ladrão ainda tentou correr e {@} escorregou no lixo do beco"}, effects: []effect{injure(5, 15)}},
				}},
				{label: "⏳ Recuar e esperar reforço", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"O reforço chegou, prendeu o ladrão e {@} ficou com metade do crédito"}, effects: []effect{earn(40, 80)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, o ladrão escalou o muro e sumiu nos telhados"}},
		},
		{
			id:    "oficio-lei-colega-corrupto",
			title: "O colega corrupto",
			areas: []int{areaSeguranca},
			intros: []string{
				"{@} pegou o colega de turno embolsando o dinheiro de uma apreensão. Ele piscou: 'Metade é sua se ficar de boca fechada'",
				"No fim do expediente, o parceiro de {@} abriu a mochila: maços de dinheiro de uma batida. 'Ninguém conta, ninguém sente falta'",
			},
			choices: []choice{
				{label: "🤐 Pegar a metade", risk: highRisk, endings: []ending{
					{weight: 60, texts: []string{"Dinheiro fácil. {@} dormiu mal, mas dormiu rico"}, effects: []effect{earn(150, 300), stain()}},
					{weight: 25, texts: []string{"A corregedoria tava de olho nos dois"}, effects: []effect{nabbed()}},
					{weight: 15, texts: []string{"O colega deu só um quarto do combinado e ainda ameaçou {@}"}, effects: []effect{earn(50, 100), stain()}},
				}},
				{label: "🗣️ Denunciar pra corregedoria", risk: midRisk, endings: []ending{
					{weight: 65, texts: []string{"O colega foi expulso e {@} ganhou medalha de honra"}, effects: []effect{earn(80, 160), enchant("justica")}},
					{weight: 35, texts: []string{"A corregedoria abafou e {@} foi transferido pra guarita mais longe da cidade"}, effects: []effect{delay(30 * time.Minute)}},
				}},
				{label: "🙈 Fingir que não viu", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} seguiu o turno olhando pro nada e com a consciência coçando"}},
				}},
			},
		},
		{
			id:    "oficio-lei-represa",
			title: "Socorro na represa",
			areas: []int{areaSeguranca},
			intros: []string{
				"Um grito vindo da represa: tem uma criança se debatendo na água e o povo só filmando com o celular",
				"Um carro desgovernado caiu no lago da praça e o motorista não sabe nadar. Todo mundo olhou pra {@}",
			},
			choices: []choice{
				{label: "🏊 Pular na água", risk: highRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} nadou igual golfinho e trouxe a vítima de volta. O vídeo viralizou"}, effects: []effect{earn(100, 220), enchant("protecao")}},
					{weight: 25, texts: []string{"{@} salvou a vítima, mas bebeu meio lago no caminho"}, effects: []effect{injure(15, 30), earn(60, 120)}},
					{weight: 5, texts: []string{"A correnteza puxou {@} pro fundo e quase foram dois pro hospital"}, effects: []effect{injure(55, 80), earn(40, 80)}},
				}},
				{label: "🛟 Jogar a boia", risk: lowRisk, needs: holds("boia"), endings: []ending{
					{weight: 100, texts: []string{"{@} acertou a boia de primeira e tirou a vítima da água sã e salva"}, effects: []effect{earn(100, 200)}},
				}},
				{label: "📞 Chamar os bombeiros", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"Os bombeiros chegaram a tempo e o resgate deu certo"}, effects: []effect{earn(20, 40)}},
					{weight: 40, texts: []string{"Um pescador puxou a vítima antes dos bombeiros chegarem"}},
				}},
				{label: "🪢 Fazer uma corda humana", risk: midRisk, endings: []ending{
					{weight: 75, texts: []string{"O povo largou o celular, deu as mãos e puxou todo mundo junto"}, effects: []effect{earn(60, 120)}},
					{weight: 25, texts: []string{"A corda humana desmontou e meio bairro caiu na água"}, effects: []effect{injure(5, 15)}},
				}},
			},
		},
		{
			id:     "oficio-lei-cantor",
			title:  "O caso do cantor famoso",
			rarity: rareEvent,
			areas:  []int{areaSeguranca},
			intros: []string{
				"Um cantor sertanejo famosíssimo foi pego dirigindo um trator na contramão da avenida, e o caso caiu justo com {@}. A imprensa tá na porta",
				"Uma dupla sertaneja do topo das paradas arrumou confusão num posto de gasolina e quem tem que resolver é {@}. Tem fã chorando na calçada",
			},
			choices: []choice{
				{label: "📺 Dar entrevista coletiva", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} virou celebridade e ganhou cachê pra aparecer no programa da tarde"}, effects: []effect{earn(300, 600)}},
					{weight: 40, texts: []string{"{@} gaguejou ao vivo e virou meme nacional"}, effects: []effect{lose(3, 7)}},
				}},
				{label: "⚖️ Tratar igual qualquer um", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} aplicou a regra sem frescura e ganhou o respeito da cidade"}, effects: []effect{earn(80, 150), enchant("justica")}},
				}},
				{label: "🎤 Aceitar convite pro show", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"Camarote, foto com o cantor e um autógrafo na camiseta"}, effects: []effect{gain(1, "autografo"), stain()}},
					{weight: 50, texts: []string{"Saiu na capa do jornal: 'Autoridade vai em show de investigado'"}, effects: []effect{lose(8, 15), stain()}},
				}},
				{label: "💰 Aceitar o cachê do empresário", risk: highRisk, endings: []ending{
					{weight: 60, texts: []string{"O empresário pagou muito bem pelo esquecimento"}, effects: []effect{earn(400, 800), stain()}},
					{weight: 40, texts: []string{"A imprensa flagrou a entrega do dinheiro"}, effects: []effect{nabbed()}},
				}},
			},
		},
		{
			id:    "oficio-crime-x9",
			title: "O parceiro X-9",
			areas: []int{areaCrime},
			intros: []string{
				"O parceiro de {@} anda estranho: celular novo, tênis de marca e foi visto saindo da delegacia pela porta dos fundos",
				"Rolou um papo na quebrada: tem alguém da equipe de {@} conversando demais com a polícia",
			},
			choices: []choice{
				{label: "🗣️ Confrontar ele", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O parceiro confessou chorando e prometeu pagar o prejuízo"}, effects: []effect{earn(80, 160)}},
					{weight: 35, texts: []string{"Ele negou tudo, ficou ofendido e saiu na mão"}, effects: []effect{injure(15, 30)}},
					{weight: 15, texts: []string{"Era X-9 mesmo, e a polícia apareceu logo depois"}, effects: []effect{nabbed()}},
				}},
				{label: "📵 Sumir do mapa uns dias", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} trocou de chip, de bairro e de corte de cabelo"}, effects: []effect{delay(15 * time.Minute)}},
				}},
				{label: "🔪 Resolver do jeito da rua", risk: highRisk, needs: killer, endings: []ending{
					{weight: 60, texts: []string{"Ninguém mais viu o parceiro. A boca ficou quietinha"}, effects: []effect{earn(100, 200), stain()}},
					{weight: 25, texts: []string{"O parceiro tava armado também"}, effects: []effect{injure(30, 50), stain()}},
					{weight: 15, texts: []string{"Era armadilha da polícia"}, effects: []effect{nabbed()}},
				}},
				{label: "🧀 Plantar uma informação falsa", risk: midRisk, endings: []ending{
					{weight: 65, texts: []string{"A polícia invadiu o endereço errado e o parceiro caiu em desgraça"}, effects: []effect{earn(60, 130)}},
					{weight: 35, texts: []string{"O endereço 'falso' era de um esquema de verdade. Sobrou pra {@}"}, effects: []effect{lose(5, 10), stain()}},
				}},
			},
		},
		{
			id:    "oficio-crime-batizada",
			title: "A mercadoria batizada",
			areas: []int{areaCrime},
			intros: []string{
				"A remessa que {@} ia repassar chegou com um cheiro estranho. Parece que alguém batizou o pó com farinha de rosca",
				"O pacote veio mais leve que o combinado e com cor de talco de bebê. Alguém na cadeia de entrega tá de sacanagem",
			},
			choices: []choice{
				{label: "👃 Provar pra ver", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"Era farinha de rosca mesmo. {@} espirrou por meia hora"}},
					{weight: 45, texts: []string{"Era do bom, e bateu forte"}, effects: []effect{buzz("po")}},
					{weight: 15, texts: []string{"Tava batizado com veneno de rato"}, effects: []effect{injure(30, 55)}},
				}},
				{label: "📦 Repassar assim mesmo", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Ninguém reclamou e o lucro veio gordo"}, effects: []effect{earn(100, 220), stain()}},
					{weight: 50, texts: []string{"Os clientes reclamaram com o chefe, e o chefe mandou chamar {@}"}, next: "oficio-crime-batizada-chefe"},
				}},
				{label: "🗑️ Jogar tudo no rio", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} jogou tudo no rio. Os peixes ficaram doidões e o prejuízo ficou com {@}"}, effects: []effect{lose(2, 5)}},
				}},
				{label: "🎒 Guardar um pouco pra você", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} separou uma parte e escondeu no fundo da mochila"}, effects: []effect{gain(2, "po"), stain()}},
					{weight: 40, texts: []string{"Na hora de esconder, metade foi parar no ventilador"}, effects: []effect{gain(1, "po")}},
				}},
			},
		},
		{
			id:    "oficio-crime-batizada-chefe",
			title: "O chefe tá bravo",
			stage: true,
			intros: []string{
				"O chefe chamou {@} pro fundo do bar. Do lado dele, dois brutamontes estalando os dedos",
				"O chefe da boca tá sentado esperando {@}, girando um isqueiro na mão. 'Senta aí. Vamos conversar sobre qualidade'",
			},
			choices: []choice{
				{label: "🙇 Pedir perdão e pagar o prejuízo", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"O chefe aceitou o dinheiro e um abraço meio forçado"}, effects: []effect{lose(2, 5)}},
				}},
				{label: "🗣️ Botar a culpa no fornecedor", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"O chefe acreditou e foi atrás do fornecedor. {@} saiu ileso"}},
					{weight: 45, texts: []string{"O chefe não acreditou. Os brutamontes deram um 'recado'"}, effects: []effect{injure(20, 40)}},
				}},
				{label: "🏃 Fugir pela janela do banheiro", risk: midRisk, endings: []ending{
					{weight: 60, favor: func(p Profession) float64 { return p.escapeBonus }, texts: []string{"{@} se espremeu pela janelinha e sumiu na noite"}, effects: []effect{delay(20 * time.Minute)}},
					{weight: 40, texts: []string{"Ficou entalado na janela e os brutamontes puxaram pelas pernas"}, effects: []effect{injure(25, 45)}},
				}},
				{label: "🔫 Sacar o ferro", risk: highRisk, needs: armedUp, endings: []ending{
					{weight: 50, texts: []string{"O chefe respeitou a ousadia e ainda ofereceu sociedade"}, effects: []effect{earn(150, 300)}},
					{weight: 40, texts: []string{"Os brutamontes também tinham ferro"}, effects: []effect{injure(40, 65)}},
					{weight: 10, texts: []string{"Foi a última conversa de {@} com o chefe"}, effects: []effect{perish()}},
				}},
			},
			idle: ending{texts: []string{"{@} demorou tanto pra responder que o chefe mandou os brutamontes resolverem"}, effects: []effect{injure(15, 30)}},
		},
		{
			id:    "oficio-crime-viatura",
			title: "A viatura na esquina",
			areas: []int{areaCrime},
			intros: []string{
				"Uma viatura parou bem na esquina onde {@} faz o corre. Os policiais desceram devagar, olhando em volta",
				"Sirene desligada, giroflex ligado. A viatura tá fazendo ronda devagarzinho na rua de {@}",
			},
			choices: []choice{
				{label: "😎 Disfarçar e assobiar", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} assobiou o hino do time e os policiais nem olharam"}},
					{weight: 40, texts: []string{"Os policiais acharam o assobio suspeito demais"}, effects: []effect{nabbed()}},
				}},
				{label: "🏃 Correr pelo beco", risk: midRisk, endings: []ending{
					{weight: 65, favor: func(p Profession) float64 { return p.escapeBonus }, texts: []string{"{@} conhece aquele beco desde criança e sumiu"}, effects: []effect{delay(15 * time.Minute)}},
					{weight: 35, texts: []string{"Tinha outro policial esperando do outro lado do beco"}, effects: []effect{nabbed()}},
				}},
				{label: "☕ Oferecer um café pros homens", risk: highRisk, price: cost{percent: 8, floor: 80}, endings: []ending{
					{weight: 70, texts: []string{"Os policiais aceitaram o 'café' e foram tomar em outro bairro"}, effects: []effect{refresh()}},
					{weight: 30, texts: []string{"Eram da corregedoria"}, effects: []effect{nabbed()}},
				}},
				{label: "🎭 Botar a máscara e sumir", risk: lowRisk, uses: "mascara", endings: []ending{
					{weight: 90, texts: []string{"Ninguém reconheceu {@} no meio do bloco de carnaval improvisado"}, effects: []effect{refresh()}},
					{weight: 10, texts: []string{"A máscara era de político e o povo começou a vaiar. Os policiais olharam"}, effects: []effect{delay(10 * time.Minute)}},
				}},
			},
		},
		{
			id:    "oficio-crime-favor-do-chefe",
			title: "O pacote do patrão",
			areas: []int{areaCrime},
			intros: []string{
				"O patrão do morro mandou chamar {@}. Ele quer um favor: levar um pacote misterioso até o outro lado da cidade, sem perguntas",
				"Um moleque entregou um recado pra {@}: o patrão quer alguém de confiança pra uma entrega rápida. 'Ele disse que é você'",
			},
			choices: []choice{
				{label: "📦 Levar o pacote", risk: highRisk, endings: []ending{
					{weight: 55, texts: []string{"Entrega feita, ninguém perguntou nada e o patrão pagou bem"}, effects: []effect{earn(200, 400), stain()}},
					{weight: 25, texts: []string{"Blitz no meio do caminho!"}, effects: []effect{nabbed()}},
					{weight: 15, texts: []string{"O pacote era de outro patrão, que não gostou nada da história"}, effects: []effect{injure(30, 50)}},
					{weight: 5, texts: []string{"Era cilada de facção rival"}, effects: []effect{perish()}},
				}},
				{label: "🤔 Perguntar o que tem dentro", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O patrão riu, gostou da coragem e liberou {@} do favor"}},
					{weight: 50, texts: []string{"O patrão não gostou nada da pergunta"}, effects: []effect{injure(15, 30)}},
				}},
				{label: "🙏 Recusar com educação", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"O patrão entendeu, mas vai lembrar disso por muito tempo"}, effects: []effect{enchant("olhogordo")}},
					{weight: 40, texts: []string{"O patrão não aceita não como resposta"}, effects: []effect{injure(10, 25)}},
				}},
				{label: "🏎️ Levar voando de carro", risk: midRisk, needs: isJob("piloto de fuga"), endings: []ending{
					{weight: 85, texts: []string{"Ninguém alcança {@} no volante. Entrega em tempo recorde"}, effects: []effect{earn(250, 450), stain()}},
					{weight: 15, texts: []string{"O radar pegou a 180 por hora"}, effects: []effect{lose(3, 6), stain()}},
				}},
			},
		},
		{
			id:    "oficio-crime-golpe-errado",
			title: "O golpe na pessoa errada",
			areas: []int{areaCrime},
			intros: []string{
				"{@} mandou o clássico 'Oi mãe, troquei de número', mas quem respondeu foi uma delegada aposentada que tá adorando a conversa",
				"O golpe do falso sequestro de {@} caiu numa senhora que respondeu: 'Pode ficar com ele, faz tempo que não paga a pensão'",
			},
			choices: []choice{
				{label: "📱 Continuar o golpe", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"A conversa ficou tão boa que ela mandou um Pix de presente"}, effects: []effect{earn(100, 200), stain()}},
					{weight: 35, texts: []string{"Ela mandou um Pix de 1 centavo com a mensagem: TE ACHEI"}, effects: []effect{nabbed()}},
					{weight: 25, texts: []string{"Ela mandou foto da carteirinha da polícia e {@} jogou o chip no rio"}, effects: []effect{delay(20 * time.Minute)}},
				}},
				{label: "📵 Bloquear e sumir", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"Bloqueou, trocou de chip e foi procurar outra vítima"}},
				}},
				{label: "💬 Pedir conselho de vida", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"Ela deu conselhos tão bons que {@} pensou em mudar de vida. Por cinco minutos"}, effects: []effect{enchant("sorte")}},
					{weight: 30, texts: []string{"Ela mandou um áudio de 47 minutos de sermão"}, effects: []effect{delay(15 * time.Minute)}},
				}},
				{label: "💻 Hackear o celular dela", risk: midRisk, needs: isJob("hacker"), endings: []ending{
					{weight: 60, texts: []string{"{@} entrou no celular dela e achou a senha do banco salva nos contatos"}, effects: []effect{earn(150, 300), stain()}},
					{weight: 40, texts: []string{"O celular dela tinha antivírus da Polícia Federal"}, effects: []effect{nabbed()}},
				}},
			},
		},
		{
			id:     "oficio-crime-cofre-antigo",
			title:  "O cofre na parede",
			rarity: rareEvent,
			areas:  []int{areaCrime},
			intros: []string{
				"Numa reforma no centro, os pedreiros acharam um cofre antigo de banco dentro de uma parede. Ninguém sabe a senha, mas todo mundo sabe que {@} tá na área",
				"Um prédio velho do centro vai ser demolido amanhã. No porão, esquecido, tem um cofre de banco de 1950. Hoje à noite não tem ninguém vigiando",
			},
			choices: []choice{
				{label: "🧨 Explodir com dinamite", risk: highRisk, uses: "dinamite", endings: []ending{
					{weight: 60, texts: []string{"BUM! Choveu nota pela rua inteira e {@} pegou o que deu"}, effects: []effect{earn(500, 1000), stain()}},
					{weight: 25, texts: []string{"BUM! O cofre abriu, e {@} saiu chamuscado"}, effects: []effect{injure(30, 55), earn(300, 600), stain()}},
					{weight: 15, texts: []string{"A explosão acordou a polícia do bairro inteiro"}, effects: []effect{nabbed()}},
				}},
				{label: "🔧 Arrombar no pé de cabra", risk: midRisk, needs: holds("pe-de-cabra"), endings: []ending{
					{weight: 50, texts: []string{"Depois de três horas de força, o cofre cedeu"}, effects: []effect{earn(400, 800), stain()}},
					{weight: 50, texts: []string{"O pé de cabra entortou e o cofre nem arranhou"}, effects: []effect{injure(10, 20)}},
				}},
				{label: "👂 Descobrir a senha no ouvido", risk: midRisk, endings: []ending{
					{weight: 30, texts: []string{"Clique, clique, clique... ABRIU! Igual filme de Hollywood"}, effects: []effect{earn(600, 1200), stain()}},
					{weight: 70, texts: []string{"Duas horas grudado no cofre e só deu pra ouvir o próprio coração"}, effects: []effect{delay(20 * time.Minute)}},
				}},
				{label: "🚶 Deixar pra lá", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} achou arriscado demais. No dia seguinte abriram o cofre: só tinha boleto de 1987"}},
				}},
			},
		},
		{
			id:    "oficio-servicos-calote",
			title: "O Pix que nunca chega",
			areas: []int{areaServicos},
			intros: []string{
				"Terminado o serviço, o cliente bateu no bolso, fez cara de surpresa e disse: 'Ih, esqueci a carteira. Te faço um Pix depois, prometo'",
				"A cliente adorou o serviço, tirou foto, postou e na hora de pagar disse que 'tá sem sinal pra fazer o Pix'",
			},
			choices: []choice{
				{label: "🙂 Confiar no Pix", risk: midRisk, endings: []ending{
					{weight: 40, texts: []string{"O Pix caiu! Com gorjeta e um emoji de coração"}, effects: []effect{earn(60, 120)}},
					{weight: 60, texts: []string{"O Pix nunca caiu, e {@} foi bloqueado em todas as redes"}},
				}},
				{label: "📢 Fazer escândalo na rua", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"Com o povo olhando, a carteira 'perdida' apareceu no outro bolso"}, effects: []effect{earn(50, 100)}},
					{weight: 40, texts: []string{"O caloteiro saiu correndo e {@} escorregou atrás dele"}, effects: []effect{injure(5, 15)}},
				}},
				{label: "⌚ Segurar algo como garantia", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"Deixou o relógio de garantia e nunca mais voltou. O relógio era do Paraguai"}, effects: []effect{gain(1, "rolex-do-paraguai")}},
					{weight: 45, texts: []string{"Deixou o celular de garantia e voltou no outro dia com o dinheiro"}, effects: []effect{earn(60, 110)}},
				}},
				{label: "🧾 Expor no grupo do bairro", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O grupo do bairro cobrou mais rápido que qualquer agiota"}, effects: []effect{earn(40, 90)}},
					{weight: 30, texts: []string{"O caloteiro saiu do grupo do bairro e foi morar em outro bairro"}},
				}},
			},
		},
		{
			id:    "oficio-servicos-anel",
			title: "O anel esquecido",
			areas: []int{areaServicos},
			intros: []string{
				"Trabalhando numa mansão, {@} achou um anel de diamante atrás do sofá. A madame nem sabe que perdeu",
				"No bolso de uma calça que veio pro serviço, {@} achou um anel brilhando. O dono é um ricaço meio distraído",
			},
			choices: []choice{
				{label: "💍 Devolver pro dono", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O dono quase chorou de emoção e deu uma gorjeta caprichada"}, effects: []effect{gain(1, "gorjeta"), earn(40, 90)}},
					{weight: 30, texts: []string{"O dono disse 'ah, esse? tenho vinte iguais' e nem agradeceu. Mas o universo anotou"}, effects: []effect{enchant("sorte")}},
				}},
				{label: "🤫 Guardar no bolso", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"Ninguém percebeu. O anel tá na mochila de {@}"}, effects: []effect{gain(1, "anel"), stain()}},
					{weight: 35, texts: []string{"A câmera escondida pegou tudo e chamaram a polícia"}, effects: []effect{nabbed()}},
					{weight: 15, texts: []string{"Era bijuteria, mas o dono achou que era roubo e fez escândalo"}, effects: []effect{lose(4, 8), stain()}},
				}},
				{label: "📸 Postar no grupo do condomínio", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"Apareceram 14 donos. O verdadeiro deu uma recompensa"}, effects: []effect{earn(50, 100)}},
					{weight: 40, texts: []string{"Virou briga no grupo do condomínio e o anel sumiu na confusão"}},
				}},
			},
		},
		{
			id:    "oficio-servicos-bicho-fugiu",
			title: "O bicho de estimação famoso",
			areas: []int{areaServicos},
			intros: []string{
				"O poodle da madame, que tem perfil no Instagram com 80 mil seguidores, escapou da coleira enquanto {@} olhava o celular",
				"O gato persa do cliente VIP fugiu pela janela. Ele vale mais que o carro de {@} e o dono chega em uma hora",
			},
			choices: []choice{
				{label: "🏃 Correr atrás do bicho", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} pegou o bicho no último segundo, antes dele atravessar a avenida"}, effects: []effect{earn(50, 100)}},
					{weight: 30, texts: []string{"O bicho entrou num bueiro e {@} entrou atrás"}, effects: []effect{injure(10, 20), earn(30, 60)}},
					{weight: 15, texts: []string{"O bicho sumiu de vez. Agora {@} vai ter que contar pro dono"}, effects: []effect{lose(5, 10)}},
				}},
				{label: "🍖 Atrair com petisco", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"Bastou um pedaço de linguiça e o bicho voltou todo carinhoso"}, effects: []effect{earn(20, 50)}},
					{weight: 30, texts: []string{"O bicho voltou, mas trouxe três vira-latas amigos junto"}},
				}},
				{label: "🔁 Arrumar um bicho parecido", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O dono não percebeu a troca. Só estranhou o bicho atender por outro nome"}, effects: []effect{earn(30, 60)}},
					{weight: 50, texts: []string{"O dono percebeu na hora: o dele era branco e esse é bege"}, effects: []effect{lose(6, 12)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, o bicho foi visto entrando num ônibus pra outra cidade"}, effects: []effect{lose(3, 6)}},
		},
		{
			id:    "oficio-servicos-celular",
			title: "O celular desbloqueado",
			areas: []int{areaServicos},
			intros: []string{
				"Uma influencer famosa deixou o celular com {@} enquanto era atendida. Ele tá desbloqueado e apitando fofoca",
				"Um jogador de futebol esqueceu o celular no banco enquanto {@} trabalhava. A tela não para de acender com mensagem de 'amor'",
			},
			choices: []choice{
				{label: "🔒 Não mexer em nada", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} guardou o celular e devolveu. O cliente postou elogio e choveu cliente novo"}, effects: []effect{earn(60, 120)}},
				}},
				{label: "👀 Dar só uma olhadinha", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} descobriu um namoro escondido que ia parar o país, e guardou o segredo pra sempre"}},
					{weight: 50, texts: []string{"O dono voltou mais cedo e pegou {@} rolando a galeria"}, effects: []effect{lose(3, 6)}},
				}},
				{label: "📰 Vender a fofoca pra um portal", risk: highRisk, endings: []ending{
					{weight: 55, texts: []string{"A fofoca virou manchete nacional e o portal pagou bem"}, effects: []effect{earn(200, 400), stain()}},
					{weight: 45, texts: []string{"Descobriram quem vazou e processaram {@}"}, effects: []effect{lose(10, 18)}},
				}},
			},
			idle: ending{texts: []string{"O dono voltou, pegou o celular e foi embora sem nem olhar pra {@}"}},
		},
		{
			id:    "oficio-servicos-saco-tilintando",
			title: "O saco que tilinta",
			areas: []int{areaServicos},
			intros: []string{
				"Saindo do serviço, {@} viu na calçada um saco de lixo pesado demais, que tilinta igual cofrinho quando mexe",
				"Na caçamba da esquina, {@} achou uma mala velha amarrada com fita crepe. Dá pra ouvir moeda lá dentro",
			},
			choices: []choice{
				{label: "🔍 Abrir e ver", risk: midRisk, endings: []ending{
					{weight: 40, texts: []string{"Era uma lata de leite em pó cheia de moedas que a vó tinha escondido"}, effects: []effect{earn(80, 160)}},
					{weight: 30, texts: []string{"Era só vidro quebrado, e {@} cortou a mão"}, effects: []effect{injure(10, 20)}},
					{weight: 30, texts: []string{"Tinha umas coisinhas que ainda prestam"}, effects: []effect{findLoot()}},
				}},
				{label: "🗑️ Deixar quieto", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"Melhor não saber. O caminhão do lixo levou o mistério embora"}},
				}},
				{label: "🏠 Bater na casa da frente", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"Era a coleção de moedas do falecido marido da vizinha. Ela deu uma parte pra {@}"}, effects: []effect{earn(50, 100), enchant("sorte")}},
					{weight: 40, texts: []string{"A vizinha disse que era lixo mesmo e bateu a porta na cara"}},
				}},
			},
		},
		{
			id:     "oficio-servicos-terno-branco",
			title:  "O homem de terno branco",
			rarity: rareEvent,
			areas:  []int{areaServicos},
			intros: []string{
				"Um homem de terno branco e óculos escuros contratou {@} e pagou adiantado com uma nota de cem dólares. Disse que tem 'uma proposta irrecusável'",
				"Uma limusine parou na porta. O motorista desceu e disse que o patrão quer os serviços de {@} hoje, 'pelo valor que você pedir'",
			},
			choices: []choice{
				{label: "🤝 Aceitar a proposta", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"Era só um ricaço excêntrico querendo o melhor serviço da cidade. Pagou uma fortuna"}, effects: []effect{earn(500, 1000)}},
					{weight: 30, texts: []string{"Era pra guardar uma maleta... e a Polícia Federal chegou junto"}, effects: []effect{nabbed()}},
					{weight: 25, texts: []string{"Era pegadinha de programa de TV. Tudo filmado, mas pagaram o cachê"}, effects: []effect{earn(150, 300)}},
				}},
				{label: "🙅 Recusar educadamente", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"O homem sorriu, disse 'você passou no teste' e deixou um envelope"}, effects: []effect{gain(1, "gorjeta"), enchant("sorte")}},
				}},
				{label: "💵 Pegar o adiantamento e sumir", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} trocou os dólares na casa de câmbio e ficou com o lucro"}, effects: []effect{earn(200, 400), stain()}},
					{weight: 40, texts: []string{"Os dólares eram falsos, e o cara do câmbio quase chamou a polícia"}, effects: []effect{lose(3, 6), stain()}},
				}},
			},
		},
		{
			id:    "oficio-fe-milagre-encomenda",
			title: "O milagre por encomenda",
			areas: []int{areaFe},
			intros: []string{
				"Um fiel rico chegou com um maço de dinheiro: quer um milagre pro time dele ganhar o clássico de domingo. 'Pago o que for'",
				"Um político em campanha quer que {@} abençoe a candidatura dele. Na mesa, um envelope e um santinho com a cara dele",
			},
			choices: []choice{
				{label: "🙏 Aceitar e rezar com fé", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Deu certo! O cliente virou devoto e pagou dobrado"}, effects: []effect{earn(150, 300)}},
					{weight: 50, texts: []string{"Deu tudo errado e o cliente quer o dinheiro de volta"}, effects: []effect{lose(3, 7)}},
				}},
				{label: "💰 Pegar o dinheiro e não rezar", risk: highRisk, endings: []ending{
					{weight: 60, texts: []string{"Deu certo sozinho e {@} levou o crédito. Mas lá de cima viram tudo"}, effects: []effect{earn(150, 300), enchant("encosto")}},
					{weight: 40, texts: []string{"Deu errado e o cliente espalhou que {@} é charlatão"}, effects: []effect{enchant("excomunhao"), lose(3, 6)}},
				}},
				{label: "🙅 Dizer que fé não se vende", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O cliente ficou sem graça e doou o dinheiro pra caridade. Lá em cima, alguém anotou"}, effects: []effect{enchant("sorte")}},
					{weight: 30, texts: []string{"O cliente foi embora bufando procurar outro"}},
				}},
			},
		},
		{
			id:    "oficio-fe-possessao",
			title: "A possessão de verdade",
			areas: []int{areaFe},
			intros: []string{
				"Chamaram {@} pra benzer uma criança 'meio agitada'. Chegando lá, ela tá falando latim de trás pra frente e a cama tá flutuando",
				"A família jura que a casa tá assombrada. Quando {@} entrou, as luzes piscaram, o rádio ligou sozinho tocando forró e a avó levitou",
			},
			choices: []choice{
				{label: "📿 Encarar o capeta", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"O demônio olhou pra {@} e sorriu: 'Finalmente alguém com coragem. Vamos negociar'"}, next: "oficio-fe-possessao-trato"},
					{weight: 40, texts: []string{"Na primeira reza, o bicho saiu correndo pela janela"}, effects: []effect{earn(100, 200), enchant("protecao")}},
					{weight: 20, texts: []string{"O capeta arremessou {@} contra o guarda-roupa"}, effects: []effect{injure(20, 40)}},
				}},
				{label: "💧 Jogar água benta", risk: midRisk, uses: "agua-benta", endings: []ending{
					{weight: 80, texts: []string{"A água benta chiou igual bife na chapa e o demônio fugiu gritando"}, effects: []effect{earn(150, 300), enchant("protecao")}},
					{weight: 20, texts: []string{"A água benta era da torneira, o vendedor enganou {@}. O demônio gargalhou"}, effects: []effect{injure(10, 25)}},
				}},
				{label: "🏃 Sair correndo", risk: midRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} saiu correndo e só parou na igreja do outro bairro"}, effects: []effect{delay(15 * time.Minute)}},
					{weight: 30, texts: []string{"O encosto pegou carona com {@} até em casa"}, effects: []effect{enchant("encosto")}},
				}},
				{label: "📞 Chamar alguém mais experiente", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"O especialista chegou, resolveu tudo e levou o crédito e o pagamento"}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, a criança desceu da cama, pegou um biscoito e disse: 'Brincadeira!'"}},
		},
		{
			id:    "oficio-fe-possessao-trato",
			title: "O trato com o capeta",
			stage: true,
			intros: []string{
				"O quarto gelou. O demônio cruzou as pernas no ar e propôs: 'Saio daqui se você me der algo em troca'",
				"Com voz de locutor de rodeio, o demônio fez a proposta: 'Saio dessa casa hoje mesmo. Mas quero um agrado'",
			},
			choices: []choice{
				{label: "✝️ Recusar e rezar mais alto", risk: highRisk, endings: []ending{
					{weight: 55, texts: []string{"A fé de {@} foi mais forte e o demônio saiu pela chaminé"}, effects: []effect{earn(250, 450), enchant("protecao")}},
					{weight: 35, texts: []string{"A briga durou a madrugada inteira e {@} saiu esgotado"}, effects: []effect{injure(30, 50), earn(100, 200)}},
					{weight: 10, texts: []string{"O demônio levou um pedacinho da alma de {@} junto"}, effects: []effect{injure(50, 75), enchant("encosto")}},
				}},
				{label: "🤝 Aceitar o trato", risk: highRisk, endings: []ending{
					{weight: 60, texts: []string{"O demônio saiu e deixou de lembrança uma moeda que brilha no escuro"}, effects: []effect{gain(1, "moeda-amaldicoada"), earn(150, 300), enchant("miseria")}},
					{weight: 40, texts: []string{"O demônio saiu da casa... e entrou na vida de {@}"}, effects: []effect{enchant("encosto"), enchant("azar")}},
				}},
				{label: "🐓 Oferecer uma galinha preta", risk: midRisk, endings: []ending{
					{weight: 70, texts: []string{"O demônio aceitou a galinha e foi embora satisfeito"}, effects: []effect{earn(100, 200)}},
					{weight: 30, texts: []string{"O demônio era vegano e ficou ofendido"}, effects: []effect{injure(15, 30)}},
				}},
				{label: "🚪 Sair de fininho", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} saiu de fininho e o demônio ficou falando sozinho"}},
				}},
			},
		},
		{
			id:    "oficio-fe-oferta",
			title: "A urna das doações",
			areas: []int{areaFe},
			intros: []string{
				"Contando a oferta do mês, {@} percebeu que dá pra 'arredondar' umas notas pra baixo e ninguém ia notar",
				"{@} ficou sozinho com a urna das doações depois da festa da padroeira. A tampa tá frouxa",
			},
			choices: []choice{
				{label: "🙏 Contar tudo certinho", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"Contou até a última moeda. A consciência tá limpa e os santos tão sorrindo"}, effects: []effect{enchant("fartura")}},
					{weight: 30, texts: []string{"Contou certinho e ninguém nem agradeceu"}},
				}},
				{label: "🤏 Arredondar pra baixo", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"Ninguém percebeu, e o bolso agradeceu"}, effects: []effect{earn(150, 300), stain()}},
					{weight: 30, texts: []string{"A tesoureira anotava tudo num caderninho"}, effects: []effect{nabbed()}},
					{weight: 20, texts: []string{"Um raio caiu na antena do templo bem na hora. Foi aviso"}, effects: []effect{earn(80, 150), enchant("azar")}},
				}},
				{label: "🍞 Doar uma parte pros pobres", risk: midRisk, endings: []ending{
					{weight: 70, texts: []string{"Os pobres comeram bem e a fama de {@} correu a cidade"}, effects: []effect{earn(30, 60), enchant("sorte")}},
					{weight: 30, texts: []string{"O conselho achou que foi desvio e abriu sindicância"}, effects: []effect{delay(20 * time.Minute)}},
				}},
			},
		},
		{
			id:    "oficio-fe-procissao",
			title: "A procissão",
			areas: []int{areaFe},
			intros: []string{
				"Dia de procissão e o andor do santo tá pesado demais. Faltam carregadores e o padre olhou direto pra {@}",
				"A procissão de São Jorge tá saindo e um dos carregadores torceu o pé. Todo mundo olhou pra {@}",
			},
			choices: []choice{
				{label: "💪 Carregar o andor", risk: midRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} carregou o santo pela cidade inteira e voltou abençoado, e com dor nas costas"}, effects: []effect{enchant("sorte"), injure(5, 15)}},
					{weight: 30, texts: []string{"O andor escorregou e {@} segurou o santo com a cabeça"}, effects: []effect{injure(15, 30)}},
				}},
				{label: "🕯️ Acompanhar com uma vela", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"Rezou a procissão inteira e voltou com a alma lavada"}, effects: []effect{heal(15)}},
				}},
				{label: "🍿 Vender pipoca no caminho", risk: midRisk, endings: []ending{
					{weight: 65, texts: []string{"Vendeu tudo antes do santo chegar na igreja"}, effects: []effect{earn(60, 130)}},
					{weight: 35, texts: []string{"O padre achou desrespeito e expulsou {@} da procissão na frente de todo mundo"}, effects: []effect{lose(2, 4), enchant("excomunhao")}},
				}},
			},
		},
		{
			id:    "oficio-fe-promessa",
			title: "A promessa pro santo",
			areas: []int{areaFe},
			intros: []string{
				"Uma fiel pediu pra {@} fazer uma promessa por ela: se o filho passar no vestibular, {@} sobe a escadaria da igreja de joelhos",
				"Um torcedor desesperado pediu pra {@} prometer pro santo: se o time não cair pra segunda divisão, {@} paga a promessa de joelhos",
			},
			choices: []choice{
				{label: "🙏 Fazer a promessa", risk: midRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} prometeu. Agora é rezar pra dar certo"}, effects: []effect{earn(30, 60), plant("oficio-fe-promessa-cobranca", 6*time.Hour)}},
				}},
				{label: "💸 Cobrar pela promessa", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"Pagaram adiantado pela promessa"}, effects: []effect{earn(80, 150), plant("oficio-fe-promessa-cobranca", 6*time.Hour)}},
					{weight: 40, texts: []string{"Acharam um absurdo cobrar e saíram falando mal de {@}"}, effects: []effect{lose(2, 4)}},
				}},
				{label: "🙅 Recusar", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} disse que joelho não é brinquedo e mandou procurar outro"}},
				}},
			},
		},
		{
			id:    "oficio-fe-promessa-cobranca",
			title: "O santo cobra a promessa",
			stage: true,
			intros: []string{
				"Lembra da promessa? Deu certo! E agora o santo tá esperando {@} subir a escadaria de 300 degraus de joelhos",
				"O pedido da promessa se realizou. Na mesma noite, {@} sonhou com o santo batendo o pé e apontando pra escadaria",
			},
			choices: []choice{
				{label: "🧎 Pagar de joelhos", risk: midRisk, endings: []ending{
					{weight: 80, texts: []string{"{@} subiu os 300 degraus de joelhos. O santo ficou satisfeito"}, effects: []effect{injure(10, 25), enchant("prosperidade")}},
					{weight: 20, texts: []string{"Chegou lá em cima sem joelho nenhum, mas com a alma leve"}, effects: []effect{injure(25, 40), enchant("sorte")}},
				}},
				{label: "🙈 Fingir que esqueceu", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"O santo não esqueceu"}, effects: []effect{enchant("azar")}},
					{weight: 50, texts: []string{"O santo mandou aviso: um raio caiu do lado de {@}"}, effects: []effect{injure(10, 20), enchant("encosto")}},
				}},
				{label: "🕯️ Acender velas no lugar", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"O santo aceitou a troca, meio a contragosto"}},
					{weight: 40, texts: []string{"O santo é dos antigos e não aceita substituição"}, effects: []effect{enchant("lerdeza")}},
				}},
			},
		},
		{
			id:    "oficio-fe-amarracao",
			title: "A amarração",
			areas: []int{areaFe},
			intros: []string{
				"Uma moça chegou com a foto do ex e uma mecha de cabelo dele: quer uma amarração pra ele voltar em 7 dias, rastejando",
				"Um rapaz apaixonado trouxe a camisa suada da crush e implorou: 'Faz ela me notar. Pago o que for'",
			},
			choices: []choice{
				{label: "🕯️ Fazer a amarração", risk: midRisk, endings: []ending{
					{weight: 45, texts: []string{"Em três dias a pessoa voltou rastejando, com flores e um pagode no celular. Pagaram o triplo"}, effects: []effect{earn(150, 300)}},
					{weight: 35, texts: []string{"Funcionou até demais: agora quem recebe as mensagens apaixonadas é {@}"}, effects: []effect{enchant("encosto")}},
					{weight: 20, texts: []string{"A amarração deu nó e ninguém sabe desfazer"}, effects: []effect{lose(2, 5)}},
				}},
				{label: "💔 Dizer que amor não se força", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"Teve choro, desabafo e lencinho. Saíram leves e ainda pagaram pelo conselho"}, effects: []effect{earn(30, 70)}},
				}},
				{label: "🍯 Fazer um adoçamento suave", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O adoçamento funcionou devagarinho e todo mundo saiu feliz"}, effects: []effect{earn(40, 90)}},
					{weight: 30, texts: []string{"O doce atraiu só formiga"}},
				}},
			},
		},
	},
}
