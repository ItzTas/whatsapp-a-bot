package economy

import "time"

var scavengingEvents = eventBook{
	name:   "🗑️ Lixão",
	places: []string{scavenging.command},
	scenarios: []scenario{
		{
			id:    "lixao-mala-trancada",
			title: "A mala do cadeado",
			intros: []string{
				"{@} achou uma mala de rodinha trancada com cadeado no meio do lixão. Pesada. Muito pesada. E lá de dentro vem um tic-tac baixinho",
				"No fundo de uma caçamba, {@} achou uma mala de couro com cadeado. Quando sacudiu, alguma coisa lá dentro fez barulho de moeda",
			},
			choices: []choice{
				{label: "🔨 Arrombar na pedrada", risk: midRisk, endings: []ending{
					{weight: 45, texts: []string{"O cadeado cedeu na terceira pedrada, e a mala tava cheia de tralha boa", "Depois de muita pedrada a mala abriu, e {@} quase chorou de emoção com o que tinha dentro"}, effects: []effect{findLoot(), findLoot()}},
					{weight: 25, texts: []string{"Dentro só tinha roupa suja de 2009 e um cheiro que nunca mais sai da memória de {@}. Foi preciso um banho de 40 minutos"}, effects: []effect{delay(15 * time.Minute)}},
					{weight: 20, texts: []string{"A pedra escorregou e acertou em cheio o dedo de {@}"}, effects: []effect{injure(5, 15)}},
					{weight: 10, texts: []string{"O tic-tac era só um despertador velho, sozinho na mala. Mas ele veio junto"}, effects: []effect{gain(1, "despertador-quebrado")}},
				}},
				{label: "🗝️ Abrir com dois grampos", risk: lowRisk, needs: isJob("chaveiro"), endings: []ending{
					{weight: 80, texts: []string{"Com dois grampos e um assobio, {@} abriu o cadeado em 3 segundos. Profissional é outra coisa"}, effects: []effect{findLoot(), earn(40, 80)}},
					{weight: 20, texts: []string{"Abriu fácil, mas dentro só tinha um bilhete: \"quem abriu é curioso\""}},
				}},
				{label: "🚓 Levar pros achados e perdidos", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O dono apareceu chorando: era a mala com as fotos do casamento. Deu uma gorjeta gorda pra {@}"}, effects: []effect{earn(40, 90)}},
					{weight: 30, texts: []string{"Ninguém reclamou a mala, e o guarda ficou com ela. {@} ficou só com a consciência limpa e uma moral com o céu"}, effects: []effect{enchant("justica")}},
				}},
				{label: "🙈 Deixar quieto", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} achou melhor não mexer em mala que faz tic-tac e foi revirar outro canto"}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, outro catador passou correndo e levou a mala"}},
		},
		{
			id:    "lixao-cachorro-bravo",
			title: "O guardião do saco preto",
			intros: []string{
				"Um vira-lata caramelo do tamanho de um bezerro tá deitado em cima de um saco preto estufado, rosnando pra {@}",
				"{@} achou um saco preto que parecia promissor, mas um caramelo enorme decidiu que o saco é dele. E ele tá mostrando os dentes",
			},
			choices: []choice{
				{label: "🍱 Oferecer a marmita", risk: lowRisk, uses: "marmita", endings: []ending{
					{weight: 55, texts: []string{"O caramelo comeu a marmita inteira, arrotou e saiu de cima do saco"}, effects: []effect{findLoot(), findLoot()}},
					{weight: 25, texts: []string{"O caramelo comeu, gostou de {@} e voltou com um filhote da ninhada na boca. Agora é seu"}, effects: []effect{gain(1, "filhote-de-cachorro")}},
					{weight: 20, texts: []string{"O caramelo comeu a marmita e continuou rosnando. Era golpe"}},
				}},
				{label: "🏃 Pegar o saco correndo", risk: highRisk, endings: []ending{
					{weight: 40, favor: func(p Profession) float64 { return p.escapeBonus }, texts: []string{"{@} foi mais rápido que o caramelo e saiu voando com o saco"}, effects: []effect{findLoot(), findLoot()}},
					{weight: 45, texts: []string{"O caramelo foi mais rápido e deu uma mordida no tornozelo de {@}"}, effects: []effect{injure(15, 30)}},
					{weight: 15, texts: []string{"A mordida foi tão feia que {@} precisou de ponto e de vacina antirrábica"}, effects: []effect{injure(30, 50), lose(2, 5)}},
				}},
				{label: "🐕 Perguntar se ele é um bom garoto", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Funcionou: o caramelo virou de barriga pra cima e {@} ganhou um amigo e o saco"}, effects: []effect{findLoot()}},
					{weight: 50, texts: []string{"O caramelo não entendeu a proposta e latiu até {@} desistir"}, effects: []effect{delay(10 * time.Minute)}},
				}},
				{label: "🚶 Procurar outro canto", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} respeitou o território do caramelo e foi revirar outra caçamba"}},
					{weight: 30, texts: []string{"{@} respeitou o caramelo e o universo recompensou: na caçamba do lado tinha coisa melhor"}, effects: []effect{findLoot()}},
				}},
			},
			idle: ending{texts: []string{"O caramelo dormiu em cima do saco, roncando, e {@} desistiu"}},
		},
		{
			id:    "lixao-catador-rival",
			title: "Treta no lixão",
			intros: []string{
				"Um catador veterano chegou empurrando o carrinho e gritou que aquele pedaço do lixão é dele desde 1998. Ele quer que {@} vaze",
				"\"Ô novato, esse lixo aqui tem dono!\", gritou um catador de boné do Flamengo, apontando o dedo pra {@}",
			},
			choices: []choice{
				{label: "🥊 Sair na mão", risk: highRisk, endings: []ending{
					{weight: 35, favor: func(p Profession) float64 { return p.guard }, texts: []string{"{@} ganhou a briga e o veterano saiu correndo, deixando o carrinho pra trás"}, effects: []effect{findLoot(), findLoot(), stain()}},
					{weight: 45, texts: []string{"O veterano tinha 70 anos e uma direita de pedreiro. {@} acordou no meio dos sacos"}, effects: []effect{injure(20, 35)}},
					{weight: 20, texts: []string{"A briga chamou a polícia, que levou os dois pra delegacia"}, effects: []effect{nabbed()}},
				}},
				{label: "🤝 Propor sociedade", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"Viraram sócios, e o veterano mostrou onde cai o lixo do bairro rico"}, effects: []effect{findLoot(), refresh()}},
					{weight: 40, texts: []string{"O veterano topou, pediu um adiantamento pra comprar pinga e sumiu no meio do lixo"}, effects: []effect{lose(1, 3)}},
				}},
				{label: "💸 Pagar o pedágio", risk: lowRisk, price: cost{percent: 3, floor: 30}, endings: []ending{
					{weight: 100, texts: []string{"O veterano mordeu uma moeda pra ver se era de verdade e liberou a área nobre do lixão"}, effects: []effect{findLoot(), findLoot()}},
				}},
				{label: "🚶 Vazar", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} achou melhor não discutir com quem tem um carrinho de ferro e muita raiva"}},
				}},
			},
		},
		{
			id:    "lixao-carteira-do-valdomiro",
			title: "A carteira do Seu Valdomiro",
			intros: []string{
				"{@} achou uma carteira com RG, a foto 3x4 de um senhorzinho e um bolo de notas de 50. No documento: Valdomiro, 81 anos",
				"No meio do papelão, {@} encontrou uma carteira com cartão do SUS, foto de neto e um bolinho de dinheiro amarrado com elástico. É do Seu Valdomiro",
			},
			choices: []choice{
				{label: "🫡 Devolver pro dono", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"Seu Valdomiro chorou, abraçou {@} e fez questão de dar uma recompensa"}, effects: []effect{earn(60, 120), plant("lixao-valdomiro-volta", 6*time.Hour)}},
					{weight: 40, texts: []string{"Seu Valdomiro agradeceu, passou um café e rezou por {@} ali mesmo"}, effects: []effect{enchant("protecao"), plant("lixao-valdomiro-volta", 6*time.Hour)}},
				}},
				{label: "💰 Ficar com o dinheiro", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} embolsou as notas e jogou a carteira de volta no lixo. A consciência pesou, o bolso não"}, effects: []effect{earn(80, 160), enchant("olhogordo")}},
					{weight: 25, texts: []string{"Seu Valdomiro tava procurando a carteira ali perto e viu tudo. Chamou a polícia na hora"}, effects: []effect{nabbed()}},
					{weight: 15, texts: []string{"As notas eram falsas, e ainda veio uma urucubaca de brinde"}, effects: []effect{enchant("azar")}},
				}},
				{label: "🪪 Vender o RG pro falsificador", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"O falsificador pagou bem pelo RG e não fez pergunta"}, effects: []effect{earn(100, 200), stain()}},
					{weight: 35, texts: []string{"O falsificador era informante da polícia"}, effects: []effect{nabbed()}},
					{weight: 15, texts: []string{"O falsificador pagou com uma nota de três reais. {@} só percebeu em casa"}, effects: []effect{gain(1, "nota-de-tres"), stain()}},
				}},
			},
		},
		{
			id:    "lixao-valdomiro-volta",
			title: "Seu Valdomiro voltou",
			stage: true,
			intros: []string{
				"Seu Valdomiro, o senhorzinho da carteira, apareceu no lixão procurando {@}. Disse que tem um ferro-velho e quer retribuir a honestidade",
				"Uma buzina de Brasília amarela chamou {@} na porta do lixão. Era o Seu Valdomiro, aquele da carteira, com um embrulho no banco de trás",
			},
			choices: []choice{
				{label: "🎁 Aceitar o presente", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"Seu Valdomiro tirou do bolso uma coisa que guardava havia 40 anos e deu pra {@}"}, effects: []effect{gain(1, "carta-holografica", "brinco-de-ouro", "disco-de-vinil"), earn(30, 60)}},
					{weight: 30, texts: []string{"O presente era uma TV de tubo de 29 polegadas. Pesada, mas é de coração"}, effects: []effect{gain(1, "tv-de-tubo")}},
				}},
				{label: "🙏 Recusar com educação", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"Seu Valdomiro ficou tão comovido que acendeu uma vela pra {@} na igreja"}, effects: []effect{enchant("sorte")}},
				}},
				{label: "🤑 Pedir dinheiro em vez de presente", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Seu Valdomiro abriu a carteira (aquela mesma) e pagou sem reclamar"}, effects: []effect{earn(100, 200)}},
					{weight: 50, texts: []string{"Seu Valdomiro achou feio, resmungou que a juventude tá perdida e foi embora"}, effects: []effect{enchant("olhogordo")}},
				}},
			},
		},
		{
			id:    "lixao-celular-que-liga",
			title: "O celular que ainda liga",
			intros: []string{
				"{@} achou um celular no lixo. A tela tá rachada, mas ele ligou: 3% de bateria e 47 notificações",
				"Um celular vibrou dentro de um saco de lixo. {@} puxou, e a tela mostrava: \"Mãe chamando\"",
			},
			choices: []choice{
				{label: "📱 Fuçar o celular", risk: midRisk, endings: []ending{
					{weight: 40, texts: []string{"O Pix tava aberto e a senha anotada atrás da capinha. {@} fez umas transferências e jogou o celular no rio"}, effects: []effect{earn(60, 150), stain()}},
					{weight: 30, texts: []string{"Era o celular de um golpista, com 300 conversas de golpe do WhatsApp. {@} entregou tudo pra polícia e levou uma recompensa"}, effects: []effect{earn(30, 80)}},
					{weight: 30, texts: []string{"A última mensagem dizia: \"quem achar esse celular já sabe\". Na mesma hora, um carro preto parou no lixão, e {@} saiu correndo e caiu num buraco"}, effects: []effect{injure(10, 25)}},
				}},
				{label: "🔧 Consertar e vender", risk: lowRisk, needs: isJob("técnico de celular"), endings: []ending{
					{weight: 100, texts: []string{"Técnico é técnico: {@} trocou a tela, passou um paninho e vendeu como seminovo"}, effects: []effect{earn(80, 160)}},
				}},
				{label: "🏪 Vender no ferro-velho", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O dono do ferro-velho pagou uns trocados e ainda reclamou da tela"}, effects: []effect{earn(15, 40)}},
					{weight: 30, texts: []string{"Antes de vender, {@} percebeu que era um Nokia tijolão que sobreviveu ao fim do mundo. Esse fica"}, effects: []effect{gain(1, "nokia")}},
				}},
				{label: "📞 Atender a ligação", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"O celular tocou: era a mãe do dono. {@} combinou de devolver e ganhou um bolo de cenoura de agradecimento"}, effects: []effect{heal(15), earn(10, 30)}},
					{weight: 40, texts: []string{"O celular tocou: era a operadora oferecendo um plano. {@} desligou e jogou o celular de volta no lixo"}},
				}},
			},
		},
		{
			id:    "lixao-rato-gigante",
			title: "O rato que parecia capivara",
			intros: []string{
				"Um rato do tamanho de um gato saiu de dentro de um sofá velho e encarou {@}. Ele tá segurando alguma coisa brilhante na boca",
				"{@} levantou uma tampa de privada e deu de cara com um rato gigante sentado em cima de uma coisa dourada, igual dragão guardando tesouro",
			},
			choices: []choice{
				{label: "🥟 Trocar por uma coxinha", risk: lowRisk, uses: "coxinha", endings: []ending{
					{weight: 100, texts: []string{"O rato largou o brilho, pegou a coxinha e foi embora satisfeito. Negócio justo"}, effects: []effect{gain(1, "brinco-de-ouro", "anel-de-latinha")}},
				}},
				{label: "🩴 Dar uma chinelada", risk: midRisk, endings: []ending{
					{weight: 45, texts: []string{"Acertou! O rato soltou o brilho e fugiu xingando em ratês"}, effects: []effect{gain(1, "brinco-de-ouro", "anel-de-latinha", "tampinha")}},
					{weight: 35, texts: []string{"Errou, e o rato mordeu o dedão do pé de {@}"}, effects: []effect{injure(10, 20)}},
					{weight: 20, texts: []string{"O rato chamou os amigos. Eram 40. {@} saiu do lixão correndo e mancando"}, effects: []effect{injure(20, 35), delay(15 * time.Minute)}},
				}},
				{label: "🐀 Adotar o rato", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O rato aceitou o convite e agora mora na mochila de {@}"}, effects: []effect{gain(1, "rato")}},
					{weight: 30, texts: []string{"O rato fingiu que aceitou, deu uma mordidinha de despedida e sumiu"}, effects: []effect{injure(3, 8)}},
				}},
				{label: "😱 Fugir gritando", risk: lowRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} saiu correndo tão rápido que tropeçou em coisa boa no caminho"}, effects: []effect{findLoot()}},
					{weight: 50, texts: []string{"{@} só parou de correr em casa, e o rato ficou com o tesouro"}},
				}},
			},
		},
		{
			id:    "lixao-fiscal-da-prefeitura",
			title: "O fiscal da prefeitura",
			intros: []string{
				"Um fiscal da prefeitura de colete e prancheta flagrou {@} revirando o lixão e avisou que catar ali sem cadastro dá multa",
				"\"Documento e cadastro de catador, por favor\", disse um fiscal suado, de prancheta na mão, olhando {@} de cima a baixo",
			},
			choices: []choice{
				{label: "📋 Pagar a multa", risk: lowRisk, price: cost{percent: 4, floor: 40}, endings: []ending{
					{weight: 100, texts: []string{"O fiscal carimbou um cadastro provisório pra {@} e ainda indicou a caçamba boa"}, effects: []effect{findLoot(), refresh()}},
				}},
				{label: "☕ Oferecer um cafezinho", risk: midRisk, price: cost{percent: 2, floor: 20}, endings: []ending{
					{weight: 70, texts: []string{"O fiscal guardou o cafezinho no bolso e foi olhar o outro lado do lixão"}, effects: []effect{findLoot()}},
					{weight: 30, texts: []string{"O fiscal era honesto, ficou ofendido e chamou a guarda municipal"}, effects: []effect{nabbed()}},
				}},
				{label: "🏃 Pular o muro", risk: midRisk, endings: []ending{
					{weight: 60, favor: func(p Profession) float64 { return p.escapeBonus }, texts: []string{"{@} pulou o muro e o fiscal de prancheta não conseguiu acompanhar"}},
					{weight: 40, texts: []string{"A calça de {@} enganchou no arame farpado, e o fiscal aplicou a multa com gosto"}, effects: []effect{injure(10, 20), lose(3, 6)}},
				}},
				{label: "🗣️ Falar que reciclagem salva o planeta", risk: lowRisk, endings: []ending{
					{weight: 50, texts: []string{"O fiscal se emocionou com o discurso e liberou {@}"}, effects: []effect{enchant("abrecaminho")}},
					{weight: 50, texts: []string{"O fiscal não se comoveu e mandou {@} parar por hoje"}, effects: []effect{delay(20 * time.Minute)}},
				}},
				{label: "♻️ Mostrar a carteirinha", risk: lowRisk, needs: isJob("catador de recicláveis", "gari"), endings: []ending{
					{weight: 100, texts: []string{"{@} mostrou a carteirinha de profissional, o fiscal pediu desculpa e indicou o melhor ponto do lixão"}, effects: []effect{findLoot(), findLoot()}},
				}},
			},
		},
		{
			id:     "lixao-pacote-suspeito",
			title:  "O pacote suspeito",
			rarity: rareEvent,
			intros: []string{
				"Embrulhado em fita silver tape, {@} achou um tijolinho no meio do lixo. Tem um cheiro estranho e um adesivo escrito \"NÃO ABRIR\"",
				"Dentro de uma caixa de sapato, {@} achou um pacote prensado e cheiroso, desses que a polícia mostra no jornal",
			},
			choices: []choice{
				{label: "✂️ Abrir pra ver", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Era muamba: erva prensada, das boas. {@} enfiou na mochila na hora"}, effects: []effect{gain(3, "baseado")}, next: "lixao-dono-da-muamba"},
					{weight: 30, texts: []string{"Era só farinha de trigo, e {@} ficou com cara de tacho"}},
					{weight: 20, texts: []string{"Era pó, e quando o pacote rasgou, a nuvem subiu direto pro nariz de {@}"}, effects: []effect{buzz("po")}},
				}},
				{label: "🚓 Entregar na delegacia", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O delegado agradeceu e deu uma recompensa de cidadão de bem"}, effects: []effect{earn(50, 120), enchant("justica")}},
					{weight: 30, texts: []string{"O delegado desconfiou de {@} e fez um interrogatório de 2 horas"}, effects: []effect{delay(20 * time.Minute)}},
				}},
				{label: "🙈 Fingir que não viu", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} chutou o pacote pra baixo de um sofá e assobiou olhando pro céu"}},
				}},
				{label: "💰 Vender pro receptador", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"O receptador pagou bem e não fez pergunta nenhuma"}, effects: []effect{earn(150, 300), stain()}},
					{weight: 35, texts: []string{"O receptador era polícia disfarçada"}, effects: []effect{nabbed()}},
					{weight: 20, texts: []string{"O dono da muamba descobriu quem vendeu e foi tirar satisfação"}, effects: []effect{injure(30, 50)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, um urubu desceu, pegou o pacote e saiu voando"}},
		},
		{
			id:    "lixao-dono-da-muamba",
			title: "O dono da muamba",
			stage: true,
			intros: []string{
				"Mal {@} guardou o pacote na mochila, dois caras de moto pararam na frente do lixão perguntando quem achou uma encomenda",
			},
			choices: []choice{
				{label: "🙋 Devolver a encomenda", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"Os caras pegaram o pacote, deram um tapinha nas costas de {@} e deixaram um agrado pela honestidade"}, effects: []effect{forfeit("baseado", 3), earn(40, 100)}},
					{weight: 30, texts: []string{"Os caras pegaram o pacote e ainda cobraram um pedágio pelo incômodo"}, effects: []effect{forfeit("baseado", 3), lose(3, 6)}},
				}},
				{label: "🤐 Dizer que não viu nada", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Os caras acreditaram e foram procurar em outro lixão"}},
					{weight: 50, texts: []string{"Um deles viu o pacote aparecendo na mochila de {@}"}, effects: []effect{forfeit("baseado", 3), injure(20, 40)}},
				}},
				{label: "🏍️ Fugir com a muamba", risk: highRisk, endings: []ending{
					{weight: 52, favor: func(p Profession) float64 { return p.escapeBonus }, texts: []string{"{@} sumiu no meio das montanhas de lixo e os caras desistiram"}, effects: []effect{stain()}},
					{weight: 40, texts: []string{"Os caras alcançaram {@} na saída do lixão"}, effects: []effect{forfeit("baseado", 3), injure(30, 50)}},
					{weight: 8, texts: []string{"Os caras não estavam pra brincadeira"}, effects: []effect{perish()}},
				}},
			},
			idle: ending{texts: []string{"{@} travou de medo, e os caras entenderam o recado e pegaram o pacote"}, effects: []effect{forfeit("baseado", 3)}},
		},
		{
			id:    "lixao-gato-preto",
			title: "O dono do lixão",
			intros: []string{
				"Um gato preto de olho amarelo sentou bem no caminho de {@} e ficou encarando. Os catadores dizem que ele é o dono do lixão",
				"{@} sentiu um arrepio: em cima de uma geladeira velha, um gato preto lambia a pata sem tirar o olho de {@}",
			},
			choices: []choice{
				{label: "🫳 Fazer carinho", risk: midRisk, endings: []ending{
					{weight: 35, texts: []string{"O gato ronronou e seguiu {@} até em casa. Agora ele é seu, e o azar dos outros também"}, effects: []effect{gain(1, "gato-preto")}},
					{weight: 40, texts: []string{"O gato aceitou o carinho por 2 segundos e depois arranhou a mão de {@}"}, effects: []effect{injure(5, 10)}},
					{weight: 25, texts: []string{"O gato fechou os olhos devagar. Dizem que isso é bênção de gato"}, effects: []effect{enchant("sorte")}},
				}},
				{label: "🧂 Jogar sal grosso", risk: lowRisk, uses: "sal-grosso", endings: []ending{
					{weight: 100, texts: []string{"{@} jogou sal grosso, fez o sinal da cruz, e o gato saiu andando ofendido. O mau-olhado nem chegou perto"}, effects: []effect{enchant("purificacao"), enchant("protecao")}},
				}},
				{label: "🔄 Dar a volta no lixão", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} deu a volta no lixão inteiro pra não cruzar com o gato. Perdeu tempo, mas tá vivo"}, effects: []effect{delay(10 * time.Minute)}},
				}},
				{label: "😤 Passar por cima", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Nada aconteceu, e {@} riu da superstição. Ainda achou um troço bom logo depois"}, effects: []effect{findLoot()}},
					{weight: 50, texts: []string{"Dois passos depois, {@} pisou num prego enferrujado. O gato nem piscou"}, effects: []effect{injure(10, 20), enchant("azar")}},
				}},
			},
		},
		{
			id:     "lixao-cofre-no-entulho",
			title:  "O cofre no entulho",
			rarity: rareEvent,
			intros: []string{
				"Debaixo de um monte de entulho de obra, {@} achou um cofre de ferro antigo, desses de filme de faroeste. Tá trancado e pesa uma tonelada",
				"A pá de {@} bateu em metal: era a porta de um cofre velho enterrado no lixão, com a fechadura toda enferrujada",
			},
			choices: []choice{
				{label: "🦾 Arrombar com o pé de cabra", risk: lowRisk, uses: "pe-de-cabra", endings: []ending{
					{weight: 70, texts: []string{"O pé de cabra entortou, mas a porta abriu. Lá dentro tinha um cofre menor, ainda fechado"}, effects: []effect{gain(1, "cofre-esquecido")}},
					{weight: 30, texts: []string{"A porta cedeu, e dentro tinha um bolo de dinheiro antigo que ainda vale"}, effects: []effect{earn(200, 400)}},
				}},
				{label: "🔑 Chamar um chaveiro", risk: lowRisk, price: cost{percent: 5, floor: 80}, endings: []ending{
					{weight: 60, texts: []string{"O chaveiro abriu na terceira tentativa. Tinha joia e papel de herança"}, effects: []effect{gain(1, "brinco-de-ouro", "carta-holografica"), earn(100, 200)}},
					{weight: 40, texts: []string{"O chaveiro abriu, e o cofre tava vazio. Ele cobrou do mesmo jeito"}},
				}},
				{label: "💪 Carregar o cofre inteiro", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"{@} carregou o cofre nas costas até em casa. Vai abrir com calma"}, effects: []effect{gain(1, "cofre-esquecido"), injure(10, 20)}},
					{weight: 45, texts: []string{"O cofre escorregou e caiu no pé de {@}"}, effects: []effect{injure(25, 45)}},
					{weight: 15, texts: []string{"No meio do caminho, a coluna de {@} fez um barulho que não devia"}, effects: []effect{injure(40, 60), delay(30 * time.Minute)}},
				}},
				{label: "🔨 Bater até abrir", risk: midRisk, endings: []ending{
					{weight: 35, texts: []string{"Depois de uma hora de marretada, o cofre abriu e tinha dinheiro de verdade"}, effects: []effect{earn(150, 300), delay(20 * time.Minute)}},
					{weight: 45, texts: []string{"O cofre não abriu, mas o barulho chamou a vizinhança inteira pra ver"}, effects: []effect{delay(20 * time.Minute)}},
					{weight: 20, texts: []string{"A marreta voltou e pegou no ombro de {@}"}, effects: []effect{injure(15, 30)}},
				}},
			},
		},
		{
			id:    "lixao-caminhao-despejando",
			title: "O caminhão da coleta",
			intros: []string{
				"O caminhão da coleta deu ré na beira do lixão e começou a despejar. Os catadores tão correndo pra pegar o melhor antes de todo mundo",
				"Buzina, ré e uma montanha de saco preto caindo: o caminhão do bairro rico chegou, e {@} tá bem do lado da caçamba",
			},
			choices: []choice{
				{label: "🏃 Correr pra frente da caçamba", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"{@} chegou primeiro e pegou a nata do lixo do bairro rico"}, effects: []effect{findLoot(), findLoot(), findLoot()}},
					{weight: 35, texts: []string{"Uma avalanche de saco de lixo desceu em cima de {@}"}, effects: []effect{injure(15, 30)}},
					{weight: 12, texts: []string{"O motorista não viu {@} no retrovisor e a ré foi longe demais"}, effects: []effect{injure(35, 50)}},
					{weight: 8, texts: []string{"O caminhão despejou tudo de uma vez, e {@} ficou lá embaixo"}, effects: []effect{perish()}},
				}},
				{label: "⏳ Esperar o caminhão sair", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"Depois da correria, ainda sobrou coisa boa pra quem teve paciência"}, effects: []effect{findLoot()}},
					{weight: 40, texts: []string{"Quando {@} chegou, só tinha sobrado casca de banana e fralda usada"}},
				}},
				{label: "🙋 Ajudar o motorista a manobrar", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O motorista agradeceu e contou de que rua veio a carga. {@} foi direto no saco certo"}, effects: []effect{findLoot(), refresh()}},
					{weight: 30, texts: []string{"O motorista mandou {@} sair da frente e não quis papo"}},
				}},
				{label: "♻️ Separar como profissional", risk: lowRisk, needs: isJob("catador de recicláveis", "gari", "dono de ferro-velho"), endings: []ending{
					{weight: 100, texts: []string{"Olho treinado acha o que presta no meio do lixo. {@} separou o melhor da carga e ainda vendeu o alumínio"}, effects: []effect{findLoot(), findLoot(), earn(20, 50)}},
				}},
			},
		},
		{
			id:    "lixao-colchao-recheado",
			title: "O colchão da vó Zuleide",
			intros: []string{
				"{@} rasgou um colchão velho de mola e caiu um bolo de nota lá de dentro. Num envelope tá escrito: 'poupança da vó Zuleide, não mexer'",
				"Dentro de um colchão jogado no lixão, {@} achou dinheiro costurado no forro. Tem um bilhete: 'é pro enterro, Zuleide'",
			},
			choices: []choice{
				{label: "💰 Embolsar tudo", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} enfiou as notas no bolso e saiu assobiando. A vó Zuleide, lá de onde estiver, viu tudo"}, effects: []effect{earn(100, 200), enchant("olhogordo")}},
					{weight: 30, texts: []string{"O neto da Zuleide tava procurando o colchão e chegou na hora errada pra {@}"}, effects: []effect{injure(10, 25)}},
					{weight: 15, texts: []string{"As notas eram de cruzado novo. Não valem nada, mas o bolso ficou estufado"}, effects: []effect{gain(1, "dinheiro-de-brinquedo")}},
				}},
				{label: "🏠 Procurar a família", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"A família chorou quando viu o dinheiro e deu uma recompensa pra {@}"}, effects: []effect{earn(40, 100), enchant("sorte")}},
					{weight: 40, texts: []string{"A vó Zuleide tava viva e forte. Benzeu {@} com raminho de arruda ali mesmo"}, effects: []effect{enchant("protecao")}},
				}},
				{label: "🔍 Ver se tem mais nas molas", risk: midRisk, endings: []ending{
					{weight: 40, texts: []string{"Tinha mais um rolinho escondido na outra ponta do colchão"}, effects: []effect{earn(50, 120)}},
					{weight: 40, texts: []string{"Uma mola pulou e cortou a mão de {@}"}, effects: []effect{injure(8, 18)}},
					{weight: 20, texts: []string{"Na outra ponta só tinha a dentadura reserva da Zuleide"}, effects: []effect{gain(1, "dentadura")}},
				}},
				{label: "🪡 Costurar de volta", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} achou melhor não mexer em dinheiro de enterro e costurou o colchão do jeito que tava"}},
				}},
			},
		},
		{
			id:    "lixao-urubu-ladrao",
			title: "O urubu ladrão",
			intros: []string{
				"Um urubu desceu do nada, pegou uma coisa brilhante do saco que {@} tava revirando e pousou num poste ali perto",
				"{@} tinha acabado de achar uma coisa dourada quando um urubu passou rasante e levou no bico",
			},
			choices: []choice{
				{label: "🪨 Jogar uma pedra", risk: midRisk, endings: []ending{
					{weight: 45, texts: []string{"Acertou! O urubu soltou o brilho e ele caiu bem no pé de {@}"}, effects: []effect{gain(1, "brinco-de-ouro", "anel-de-latinha", "rolex-do-paraguai")}},
					{weight: 35, texts: []string{"Errou o urubu e acertou a janela do vizinho do lixão, que veio cobrar o vidro"}, effects: []effect{lose(2, 5)}},
					{weight: 20, texts: []string{"O urubu se vingou do jeito que urubu se vinga: vomitou em {@} lá de cima"}, effects: []effect{delay(10 * time.Minute)}},
				}},
				{label: "🍱 Trocar por comida estragada", risk: lowRisk, uses: "marmita-estragada", endings: []ending{
					{weight: 100, texts: []string{"Urubu não resiste a uma marmita passada. Ele largou o brilho e foi comer feliz"}, effects: []effect{gain(1, "brinco-de-ouro", "anel-de-latinha", "rolex-do-paraguai")}},
				}},
				{label: "🐦 Fazer amizade com o urubu", risk: lowRisk, endings: []ending{
					{weight: 30, texts: []string{"O urubu não devolveu nada, mas o filhote dele resolveu seguir {@} pra casa"}, effects: []effect{gain(1, "urubu")}},
					{weight: 70, texts: []string{"O urubu olhou pra {@} com desprezo e foi embora com o brilho"}},
				}},
				{label: "🚶 Deixar pra lá", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} deu tchau pro brilho e voltou a revirar o lixo"}},
					{weight: 30, texts: []string{"{@} desistiu do brilho e, dois sacos depois, achou coisa melhor"}, effects: []effect{findLoot()}},
				}},
			},
		},
		{
			id:     "lixao-quadro-do-museu",
			title:  "O quadro empoeirado",
			rarity: rareEvent,
			intros: []string{
				"Encostado numa geladeira, tem um quadro todo empoeirado com assinatura borrada. Parece pintura de museu, e {@} jura que já viu num livro da escola",
				"Debaixo de um sofá rasgado, {@} achou uma pintura a óleo numa moldura dourada. Atrás tem um carimbo meio apagado de museu",
			},
			choices: []choice{
				{label: "🖼️ Levar num avaliador", risk: lowRisk, price: cost{percent: 2, floor: 40}, endings: []ending{
					{weight: 50, texts: []string{"O avaliador tirou os óculos, olhou de novo e fez um cheque na hora: era de um pintor famoso dos anos 50"}, effects: []effect{earn(200, 500)}},
					{weight: 50, texts: []string{"O avaliador riu: era impressão de loja de 1,99 com moldura de plástico"}},
				}},
				{label: "🏛️ Ligar pro museu", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"Era o quadro roubado do museu da cidade em 1994. A recompensa ainda tava valendo"}, effects: []effect{earn(150, 300), enchant("justica")}},
					{weight: 40, texts: []string{"O museu disse que não falta quadro nenhum e desligou na cara de {@}"}},
				}},
				{label: "🕶️ Vender pro receptador", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"O receptador pagou em dinheiro vivo e sumiu com o quadro numa Kombi"}, effects: []effect{earn(250, 500), stain()}},
					{weight: 35, texts: []string{"O quadro era roubado, e o receptador já tava sendo seguido pela polícia"}, effects: []effect{nabbed()}},
					{weight: 15, texts: []string{"O receptador pegou o quadro pra 'ver na luz' e nunca mais voltou"}},
				}},
				{label: "🔥 Usar de lenha", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} esquentou a marmita com o quadro. Nunca vai saber quanto valia, e a marmita ficou ótima"}, effects: []effect{heal(10)}},
				}},
				{label: "🔍 Avaliar de olho treinado", risk: lowRisk, needs: isJob("dono de brechó", "dono de ferro-velho", "leiloeiro"), endings: []ending{
					{weight: 100, texts: []string{"{@} reconheceu a pincelada na hora e vendeu pra um colecionador pelo preço certo"}, effects: []effect{earn(300, 500)}},
				}},
			},
		},
	},
}
