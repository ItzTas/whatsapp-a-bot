package economy

import "time"

var generalEvents = eventBook{
	name:   "🌎 Gerais",
	places: nil,
	scenarios: []scenario{
		{
			id:    "geral-pix-errado",
			title: "O Pix que caiu do céu",
			intros: []string{
				"O celular de {@} vibrou: caiu um Pix gordo de um tal de Josué, que {@} nunca viu na vida. Dois minutos depois chegou a mensagem: \"moço pelo amor de Deus foi engano, era o dinheiro do aluguel\"",
				"{@} abriu o app do banco e quase caiu da cadeira: tinha um Pix de um desconhecido na conta. Logo em seguida veio um áudio chorando pedindo de volta",
			},
			choices: []choice{
				{label: "🤝 Devolver tudo", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} devolveu na hora. O Josué mandou um áudio de 4 minutos agradecendo e ainda deixou uma marmita na portaria"}, effects: []effect{gain(1, "marmita")}},
					{weight: 40, texts: []string{"{@} devolveu e recebeu de volta um \"vlw\" seco. Pelo menos a consciência tá limpa"}},
				}},
				{label: "🤑 Fingir que não viu", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Ninguém nunca mais falou do assunto e {@} pagou o lanche da semana com o dinheiro do Josué"}, effects: []effect{earn(60, 150)}},
					{weight: 30, texts: []string{"O Josué abriu contestação no banco, o Pix voltou e {@} ainda levou uma tarifa de brinde"}, effects: []effect{lose(2, 5)}},
					{weight: 20, texts: []string{"O Josué era primo do delegado. Agora o nome de {@} tá numa ocorrência de apropriação indébita"}, effects: []effect{earn(30, 80), stain()}},
				}},
				{label: "🕵️ Desconfiar e investigar", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"Era o golpe do Pix falso: o comprovante era montagem e o tal Josué queria que {@} mandasse dinheiro de verdade. Golpe evitado"}},
					{weight: 40, texts: []string{"{@} ligou pro banco, descobriu que o Pix era real e o gerente, emocionado com tanta honestidade, liberou um brinde"}, effects: []effect{earn(20, 50)}},
				}},
			},
		},
		{
			id:    "geral-filho-numero-novo",
			title: "Oi, troquei de número",
			intros: []string{
				"Chegou no WhatsApp de {@}: \"oi, é seu sobrinho, troquei de número. Tô num aperto, consegue me fazer um Pix rapidinho? amanhã te devolvo\". A foto de perfil é um pôr do sol",
				"Um número desconhecido mandou pra {@}: \"oi, sou eu, perdi o celular, salva esse número novo. Preciso pagar um boleto urgente, me ajuda?\"",
			},
			choices: []choice{
				{label: "💸 Mandar o Pix", risk: midRisk, price: cost{percent: 5, floor: 50}, endings: []ending{
					{weight: 75, texts: []string{"Pix enviado, número bloqueado. O \"sobrinho\" sumiu mais rápido que o dinheiro, e anotou o nome de {@} na lista de quem cai fácil"}, effects: []effect{plant("geral-golpista-volta", 6*time.Hour)}},
					{weight: 25, texts: []string{"Por milagre era mesmo o sobrinho, que devolveu no dia seguinte com juros e um pedido de desculpas"}, effects: []effect{earn(60, 120)}},
				}},
				{label: "🎙️ Pedir um áudio", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O \"sobrinho\" disse que tava sem microfone e sumiu. Clássico"}},
					{weight: 30, texts: []string{"O golpista mandou um áudio de inteligência artificial com voz de robô. {@} printou, mandou no grupo da família e virou a lenda do dia"}, effects: []effect{enchant("sorte")}},
				}},
				{label: "😈 Enrolar o golpista", risk: midRisk, endings: []ending{
					{weight: 45, texts: []string{"{@} disse que só mandava depois de um Pix de teste. O golpista mandou um trocado pra ganhar confiança e {@} bloqueou na hora"}, effects: []effect{earn(30, 80)}},
					{weight: 35, texts: []string{"O golpista percebeu a zoeira, ficou bravo e jogou uma praga digital em {@}"}, effects: []effect{enchant("olhogordo")}},
					{weight: 20, texts: []string{"Enquanto {@} enrolava, o golpista clonou o WhatsApp e pediu dinheiro pra família inteira em nome de {@}"}, effects: []effect{lose(2, 6), plant("geral-golpista-volta", 6*time.Hour)}},
				}},
				{label: "🚔 Rastrear o golpista", risk: lowRisk, needs: lawman, endings: []ending{
					{weight: 70, texts: []string{"{@} rastreou o número, achou o golpista num quarto cheio de chips e levou a recompensa da operação"}, effects: []effect{earn(100, 220), enchant("justica")}},
					{weight: 30, texts: []string{"O golpista era de outro estado. {@} só conseguiu derrubar a conta dele"}, effects: []effect{earn(20, 50)}},
				}},
			},
		},
		{
			id:    "geral-golpista-volta",
			title: "O golpista voltou",
			stage: true,
			intros: []string{
				"O mesmo golpista que já passou a perna em {@} voltou, agora fingindo ser do banco: \"sua conta foi invadida, preciso que você confirme o código que chegou por SMS\"",
				"Olha ele de novo! O golpista que enganou {@} mandou mensagem como se nada tivesse acontecido: \"lembra de mim? tenho um investimento que rende 30% ao dia\"",
			},
			choices: []choice{
				{label: "📲 Passar o código", risk: highRisk, endings: []ending{
					{weight: 80, texts: []string{"Era golpe de novo, óbvio. O golpista limpou uma parte da conta de {@} e ainda mandou um emoji de risada"}, effects: []effect{lose(8, 15)}},
					{weight: 20, texts: []string{"O golpista digitou o código errado três vezes e o banco bloqueou ele. {@} escapou por pura burrice alheia"}},
				}},
				{label: "🪤 Fingir que caiu e marcar encontro", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} marcou na praça e apareceu com a vizinhança inteira. O golpista devolveu tudo que tinha levado, e mais um pouco, pra sair inteiro"}, effects: []effect{earn(150, 300)}},
					{weight: 30, texts: []string{"O golpista não apareceu, mas mandou um motoboy entregar uma pizza fria no nome de {@}"}, effects: []effect{gain(1, "pizza")}},
					{weight: 20, texts: []string{"O golpista apareceu com dois primos grandões. {@} saiu correndo e tropeçou no meio-fio"}, effects: []effect{injure(10, 25)}},
				}},
				{label: "🚫 Bloquear e denunciar", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} bloqueou, denunciou e mandou o print pro grupo do condomínio. Lição aprendida"}, effects: []effect{enchant("protecao")}},
				}},
				{label: "🚔 Prender em flagrante", risk: lowRisk, needs: lawman, endings: []ending{
					{weight: 100, texts: []string{"{@} armou a cilada, o golpista caiu direitinho e a recompensa pela quadrilha inteira foi pro bolso de {@}"}, effects: []effect{earn(200, 400), enchant("justica")}},
				}},
			},
		},
		{
			id:    "geral-vira-lata",
			title: "O vira-lata caramelo",
			intros: []string{
				"Um vira-lata caramelo começou a seguir {@} abanando o rabo. Para quando {@} para, anda quando {@} anda. Já escolheu o dono",
				"{@} sentiu um focinho gelado na perna: era um cachorro de rua magrinho, olhando com cara de quem não come desde terça",
			},
			choices: []choice{
				{label: "🍞 Dar um pedaço de pão", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"O cachorro comeu, deitou do lado e passou o dia rosnando pra qualquer um que chegasse perto de {@}"}, effects: []effect{enchant("protecao")}},
					{weight: 40, texts: []string{"O cachorro engoliu o pão inteiro e foi embora sem nem olhar pra trás. Interesseiro"}},
				}},
				{label: "🏠 Levar pra casa", risk: midRisk, endings: []ending{
					{weight: 40, texts: []string{"Na manhã seguinte, o cachorro apareceu com um filhote na boca. Agora são dois na família de {@}"}, effects: []effect{gain(1, "filhote-de-cachorro")}},
					{weight: 35, texts: []string{"O cachorro comeu o sofá, o chinelo e o controle da TV. Mas é amor"}, effects: []effect{lose(2, 5)}},
					{weight: 25, texts: []string{"Era o cachorro fujão do vizinho brabo, que veio buscar e ainda cobrou a ração que ele comeu"}, effects: []effect{lose(1, 3)}},
				}},
				{label: "🏃 Tentar despistar", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O cachorro achou que era brincadeira e correu junto. {@} chegou suando e o cachorro chegou feliz"}},
					{weight: 30, texts: []string{"O cachorro ficou ofendido e deu uma mordidinha de leve no calcanhar de {@}"}, effects: []effect{injure(3, 8)}},
				}},
				{label: "🩺 Cuidar das feridas dele", risk: lowRisk, needs: inArea(areaSaude), endings: []ending{
					{weight: 100, texts: []string{"{@} limpou as feridas, deu vermífugo e postou o antes e depois. Choveu doação de gente comovida"}, effects: []effect{earn(60, 140), enchant("sorte")}},
				}},
			},
		},
		{
			id:    "geral-cigana",
			title: "A cigana da praça",
			intros: []string{
				"Uma cigana de saia rodada segurou a mão de {@} do nada: \"deixa eu ler tua sorte, meu bem, tô vendo uma coisa forte aqui\"",
				"\"Psiu, você aí!\" Uma cigana parou {@} no meio da rua jurando que viu um recado do além na palma da mão de {@}",
			},
			choices: []choice{
				{label: "🖐️ Deixar ler a mão", risk: midRisk, price: cost{percent: 2, floor: 20}, endings: []ending{
					{weight: 45, texts: []string{"\"Vejo dinheiro chegando\", disse ela. E não é que {@} sentiu a sorte mudar?"}, effects: []effect{enchant("sorte")}},
					{weight: 30, texts: []string{"Ela olhou a mão, fez o sinal da cruz e saiu andando rápido. {@} ficou com um frio na espinha"}, effects: []effect{enchant("azar")}},
					{weight: 25, texts: []string{"Ela leu a mão direita de {@} enquanto a mão esquerda dela lia a carteira"}, effects: []effect{lose(3, 6)}},
				}},
				{label: "🙅 Recusar educadamente", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"A cigana deu de ombros e foi ler a mão do moço da pamonha"}},
				}},
				{label: "🃏 Pedir as cartas de graça", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Ela riu da cara de pau, tirou a carta da Roda da Fortuna e soprou um segredo no ouvido de {@}"}, effects: []effect{earn(20, 60), enchant("faro")}},
					{weight: 50, texts: []string{"\"De graça? Então leva isso de graça também\", e rogou uma praga que {@} vai sentir na próxima extração"}, effects: []effect{enchant("panema")}},
				}},
				{label: "🔮 Ler a mão dela de volta", risk: midRisk, needs: inArea(areaFe), endings: []ending{
					{weight: 60, texts: []string{"Vidente reconhece vidente: as duas sortes se cruzaram e saiu troca de simpatia, banho de ervas e um caminho aberto pra {@}"}, effects: []effect{enchant("abrecaminho"), earn(30, 80)}},
					{weight: 40, texts: []string{"Deu disputa de vidência no meio da praça. A cigana ganhou e deixou um encosto de lembrança"}, effects: []effect{enchant("encosto")}},
				}},
			},
		},
		{
			id:    "geral-pregador",
			title: "O pregador do busão",
			intros: []string{
				"No ônibus lotado, um pregador de terno pegou o microfone do motorista e começou: \"irmãos, tem alguém aqui precisando de uma palavra\". E olhou fixo pra {@}",
				"Um pregador entrou no busão com uma caixinha de som e uma bíblia enorme e escolheu justo o banco do lado de {@} pra pregar",
			},
			choices: []choice{
				{label: "🙏 Ouvir a palavra", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"A pregação tocou o coração de {@}, que desceu do ônibus com o corpo fechado"}, effects: []effect{enchant("protecao")}},
					{weight: 40, texts: []string{"{@} prestou tanta atenção que perdeu o ponto e desceu três bairros depois"}, effects: []effect{delay(15 * time.Minute)}},
				}},
				{label: "🎤 Pregar junto", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} pegou o embalo, puxou um louvor e o ônibus inteiro cantou junto. Teve até oferta"}, effects: []effect{earn(40, 100)}},
					{weight: 45, texts: []string{"O motorista cansou do show, parou o ônibus e botou os dois pra fora no meio do nada"}, effects: []effect{delay(20 * time.Minute), injure(3, 8)}},
				}},
				{label: "🎧 Botar o fone no máximo", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} sobreviveu à viagem ouvindo pagode no talo"}},
					{weight: 30, texts: []string{"O pregador viu o fone, entendeu como um sinal e orou ainda mais alto. Pelo menos a viagem passou rápido"}, effects: []effect{enchant("fartura")}},
				}},
				{label: "📖 Debater versículo por versículo", risk: midRisk, needs: inArea(areaFe), endings: []ending{
					{weight: 50, texts: []string{"{@} ganhou o debate com três citações decoradas e o pregador ofereceu uma vaga de auxiliar na igreja dele"}, effects: []effect{earn(60, 140), enchant("justica")}},
					{weight: 50, texts: []string{"O debate virou briga de igreja e o pregador excomungou {@} ali mesmo, no corredor do ônibus"}, effects: []effect{enchant("excomunhao")}},
				}},
			},
		},
		{
			id:    "geral-blitz",
			title: "Blitz da Lei Seca",
			intros: []string{
				"Cones laranja, viatura e um guarda acenando com a lanterna: {@} caiu bem no meio de uma blitz da Lei Seca",
				"\"Boa noite, documentos e encosta ali\". A blitz pegou {@} de surpresa na saída da avenida",
			},
			choices: []choice{
				{label: "😇 Soprar o bafômetro", risk: lowRisk, endings: []ending{
					{weight: 75, texts: []string{"Zero vírgula zero. O guarda até elogiou o hálito de {@}"}},
					{weight: 25, favor: func(p Profession) float64 {
						if p.drug != "" {
							return 0.6
						}
						return 0
					}, texts: []string{"O bafômetro apitou por causa do enxaguante bucal. Vai explicar isso pro guarda"}, effects: []effect{lose(2, 5)}},
				}},
				{label: "🏃 Dar meia-volta e fugir", risk: highRisk, endings: []ending{
					{weight: 45, favor: func(p Profession) float64 { return p.escapeBonus }, texts: []string{"{@} fez um cavalo de pau digno de filme e sumiu pelo acostamento. Mas a placa ficou anotada"}, effects: []effect{stain()}},
					{weight: 40, texts: []string{"A viatura alcançou {@} em dois quarteirões"}, effects: []effect{nabbed()}},
					{weight: 15, texts: []string{"Na pressa, {@} deu de cara num poste"}, effects: []effect{injure(20, 40)}},
				}},
				{label: "☕ Oferecer um cafezinho", risk: midRisk, price: cost{percent: 4, floor: 80}, endings: []ending{
					{weight: 60, texts: []string{"O guarda guardou o \"cafezinho\" no bolso e mandou {@} seguir viagem"}},
					{weight: 40, texts: []string{"O guarda era da Corregedoria. Tentativa de suborno filmada e tudo"}, effects: []effect{nabbed()}},
				}},
				{label: "🪪 Mostrar o distintivo", risk: lowRisk, needs: lawman, endings: []ending{
					{weight: 100, texts: []string{"Os colegas reconheceram {@}, liberaram com um tapinha nas costas e ainda deram uma carona"}, effects: []effect{refresh()}},
				}},
			},
		},
		{
			id:    "geral-apagao",
			title: "O apagão",
			intros: []string{
				"Tudo apagou de uma vez: poste, geladeira, wi-fi. O bairro inteiro de {@} ficou no breu e só se ouvia a vizinhança gritando",
				"Um estouro no transformador e pronto: apagão geral. {@} ficou no escuro com o celular em 4% de bateria",
			},
			choices: []choice{
				{label: "🕯️ Acender uma vela e esperar", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} passou o apagão contando história de assombração pra criançada da rua. Virou lenda"}},
					{weight: 30, texts: []string{"Procurando a vela no escuro, {@} achou umas moedas perdidas atrás do sofá"}, effects: []effect{earn(10, 30)}},
				}},
				{label: "🔦 Ver o que houve no poste", risk: midRisk, endings: []ending{
					{weight: 45, texts: []string{"{@} segurou a lanterna pro eletricista da companhia, que dividiu o lanche e a gorjeta"}, effects: []effect{earn(30, 70)}},
					{weight: 35, texts: []string{"{@} encostou no fio errado e levou um choque que deixou o cabelo em pé"}, effects: []effect{injure(10, 25)}},
					{weight: 20, texts: []string{"Embaixo do poste tinha uma caixa que caiu do caminhão da companhia. Achado não é roubado, né?"}, effects: []effect{gain(1, "radinho", "fone", "cabo-usb")}},
				}},
				{label: "🛒 Aproveitar o escuro na venda", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"{@} saiu da venda do Seu Manel com os braços cheios antes de a luz voltar"}, effects: []effect{gain(3, "latao", "coxinha", "chocolate", "miojo"), stain()}},
					{weight: 35, texts: []string{"A luz voltou bem na hora, com {@} abraçado num fardo de latão e cara de paisagem. O Seu Manel chamou a viatura"}, effects: []effect{nabbed()}},
					{weight: 20, texts: []string{"No escuro, {@} tropeçou numa caixa de garrafas e se esborrachou no chão"}, effects: []effect{injure(15, 30)}},
				}},
				{label: "⚡ Consertar o transformador", risk: midRisk, needs: inArea(areaObras), endings: []ending{
					{weight: 70, texts: []string{"Com duas fitas isolantes e muita fé, {@} devolveu a luz pro bairro. A vizinhança fez vaquinha de agradecimento"}, effects: []effect{earn(100, 220)}},
					{weight: 30, texts: []string{"O transformador estourou de novo, agora com {@} pendurado no poste"}, effects: []effect{injure(20, 40)}},
				}},
			},
		},
		{
			id:    "geral-enchente",
			title: "A enchente",
			intros: []string{
				"Choveu em uma hora o que era pra chover no mês. A rua de {@} virou rio, com sofá boiando e tudo",
				"O bueiro entupiu, o córrego subiu e em dez minutos a água tava no joelho de {@}",
			},
			choices: []choice{
				{label: "🏠 Esperar a água baixar", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} esperou no segundo andar comendo pipoca e vendo a rua virar Veneza"}},
					{weight: 40, texts: []string{"A água entrou e levou umas coisinhas, mas nada que doa muito"}, effects: []effect{lose(1, 3)}},
				}},
				{label: "🛶 Ajudar os vizinhos", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} tirou três idosos, dois cachorros e um papagaio de dentro das casas. O bairro inteiro aplaudiu, com direito a vaquinha"}, effects: []effect{earn(60, 140), enchant("fartura")}},
					{weight: 40, texts: []string{"{@} caiu num bueiro sem tampa no meio do resgate"}, effects: []effect{injure(10, 25)}},
				}},
				{label: "🌊 Atravessar a nado", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} atravessou a correnteza igual medalhista de olimpíada e chegou antes de todo mundo"}, effects: []effect{refresh()}},
					{weight: 35, texts: []string{"A correnteza arrastou {@} por meio quarteirão até um poste segurar"}, effects: []effect{injure(20, 40)}},
					{weight: 8, texts: []string{"A correnteza foi mais forte. Acharam {@} três ruas depois"}, effects: []effect{perish()}},
					{weight: 7, texts: []string{"No meio do caminho, {@} agarrou uma mala que boiava cheia de coisa"}, effects: []effect{gain(1, "mala", "cofrinho", "lata-de-ninho")}},
				}},
				{label: "🛟 Usar a boia salva-vidas", risk: lowRisk, needs: holds("boia"), endings: []ending{
					{weight: 100, texts: []string{"Com a boia, {@} montou um serviço de travessia e cobrou passagem simbólica"}, effects: []effect{earn(80, 160), enchant("protecao")}},
				}},
			},
		},
		{
			id:    "geral-sorteio-radio",
			title: "O sorteio da rádio",
			intros: []string{
				"O celular de {@} tocou: \"Alôôô, aqui é da Rádio Tupiniquim FM! Você foi sorteado na pergunta valendo prêmio! Tá preparado?\"",
				"\"Atenção, ouvinte! {@}, você está ao vivo na Rádio Tupiniquim e pode levar um prêmio agora!\"",
			},
			choices: []choice{
				{label: "📻 Responder a pergunta", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"\"Qual o nome do cachorro do Chaves?\" {@} cravou: ele não tem cachorro! Acertou e foi pra grande final"}, next: "geral-radio-final"},
					{weight: 45, texts: []string{"{@} gaguejou ao vivo, errou a capital da Bahia e virou meme na rádio"}, effects: []effect{enchant("olhogordo")}},
				}},
				{label: "🤨 Desconfiar e desligar", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"Era golpe mesmo: logo depois o mesmo número ligou pra vizinha pedindo os dados do cartão"}},
					{weight: 30, texts: []string{"Era sorteio de verdade. O prêmio foi pro próximo ouvinte e {@} ouviu tudo pelo rádio da padaria"}},
				}},
				{label: "🎤 Mandar um alô pra galera", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} mandou alô pra mãe, pra tia, pro grupo do zap e pro Seu Zé da padaria. O locutor achou simpático e mandou um brinde"}, effects: []effect{earn(10, 30)}},
				}},
			},
		},
		{
			id:    "geral-radio-final",
			title: "A pergunta de ouro",
			stage: true,
			intros: []string{
				"\"{@}, você chegou na final! Pode parar agora e levar o prêmio garantido, ou arriscar tudo na pergunta de ouro!\" A rádio inteira prendeu a respiração",
			},
			choices: []choice{
				{label: "💰 Parar e levar o prêmio", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} jogou seguro, levou o prêmio e ainda ganhou um boné da rádio"}, effects: []effect{earn(80, 160)}},
				}},
				{label: "🎰 Arriscar na pergunta de ouro", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"\"Em que ano o Brasil foi tetra?\" {@} respondeu 1994 antes do locutor terminar. PRÊMIO MÁXIMO!"}, effects: []effect{earn(300, 600), gain(1, "ingresso-de-camarote")}},
					{weight: 60, texts: []string{"{@} respondeu com toda a certeza do mundo e errou. Saiu sem nada e com a rádio inteira rindo"}},
				}},
				{label: "📞 Pedir ajuda aos universitários", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"Os universitários acertaram e {@} levou metade do prêmio de ouro"}, effects: []effect{earn(150, 300)}},
					{weight: 45, texts: []string{"Os universitários discordaram entre si e {@} saiu de mãos abanando"}},
				}},
			},
		},
		{
			id:    "geral-celular-bueiro",
			title: "O celular no bueiro",
			intros: []string{
				"Num tropeço besta, o celular de {@} escorregou da mão, quicou duas vezes e PLOFT: direto no bueiro",
				"{@} foi tirar o celular do bolso pra ver a hora e ele mergulhou de cabeça no bueiro mais nojento da cidade",
			},
			choices: []choice{
				{label: "🤿 Enfiar o braço no bueiro", risk: midRisk, endings: []ending{
					{weight: 45, texts: []string{"{@} resgatou o celular e, de brinde, puxou mais alguma coisa que tava lá embaixo"}, effects: []effect{gain(1, "celular-afogado", "carteira-perdida", "brinco-de-ouro", "anel-de-latinha")}},
					{weight: 35, texts: []string{"Uma ratazana do tamanho de um gato mordeu o dedo de {@}"}, effects: []effect{injure(5, 15)}},
					{weight: 20, texts: []string{"O braço de {@} ficou entalado e só os bombeiros resolveram, uma hora depois"}, effects: []effect{delay(30 * time.Minute)}},
				}},
				{label: "🪦 Dar o celular como morto", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} fez um minuto de silêncio pelo celular e seguiu a vida"}},
					{weight: 40, texts: []string{"O tio de {@} ficou com dó e deu um Nokia tijolão que tava na gaveta desde 2004"}, effects: []effect{gain(1, "nokia")}},
				}},
				{label: "🧲 Pescar com ímã e barbante", risk: lowRisk, needs: holds("ima"), endings: []ending{
					{weight: 100, texts: []string{"Com o ímã num barbante, {@} pescou o celular e mais umas tralhas de metal"}, effects: []effect{gain(1, "carteira-perdida", "anel-de-latinha", "tampinha")}},
				}},
			},
		},
		{
			id:    "geral-arraia",
			title: "O arraiá da rua",
			intros: []string{
				"Bandeirinha, fogueira e quentão: a rua de {@} virou arraiá, e a quadrilha tá precisando de gente",
				"\"Olha a cobra! É mentira!\" O arraiá da escolinha tá bombando e puxaram {@} pra dentro da festa",
			},
			choices: []choice{
				{label: "💌 Mandar um correio elegante", risk: lowRisk, endings: []ending{
					{weight: 50, texts: []string{"O correio elegante voltou com resposta e um coração desenhado"}, effects: []effect{gain(1, "carta-de-amor")}},
					{weight: 50, texts: []string{"A resposta veio: \"quem é você?\". Doeu"}},
				}},
				{label: "🎣 Jogar na pescaria", risk: lowRisk, price: cost{floor: 20}, endings: []ending{
					{weight: 65, texts: []string{"{@} pescou um peixinho de papelão e levou um prêmio da barraca"}, effects: []effect{gain(1, "pamonha", "pipoca", "caldo-de-cana", "churros")}},
					{weight: 35, texts: []string{"{@} pescou o prêmio grande da barraca!"}, effects: []effect{gain(1, "ursinho", "caixa-de-bombom")}},
				}},
				{label: "🔥 Pular a fogueira", risk: highRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} pulou a fogueira com estilo e o arraiá inteiro aplaudiu. Dizem que dá sorte o ano todo"}, effects: []effect{enchant("sorte")}},
					{weight: 30, texts: []string{"{@} pulou, mas a sobrancelha ficou pra trás"}, effects: []effect{injure(10, 25)}},
					{weight: 10, texts: []string{"{@} tropeçou na hora do pulo e caiu de bunda na brasa"}, effects: []effect{injure(35, 60)}},
				}},
				{label: "💃 Puxar a quadrilha", risk: midRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} comandou o \"anarriê\" com tanta moral que o noivo da quadrilha pagou um cachê"}, effects: []effect{earn(30, 80)}},
					{weight: 30, texts: []string{"No \"olha a chuva\", {@} escorregou e torceu o tornozelo"}, effects: []effect{injure(8, 18)}},
				}},
			},
		},
		{
			id:    "geral-bloquinho",
			title: "O bloquinho",
			intros: []string{
				"Um bloco de carnaval passou arrastando todo mundo, com marchinha, purpurina e um cara fantasiado de Chapolin. {@} ficou bem no meio",
				"Sem aviso nenhum, um bloquinho de rua dobrou a esquina e engoliu {@} no meio do confete",
			},
			choices: []choice{
				{label: "🎉 Cair na folia", risk: midRisk, endings: []ending{
					{weight: 40, texts: []string{"{@} pulou até de madrugada e ainda achou umas notas amassadas no chão do bloco"}, effects: []effect{earn(30, 90)}},
					{weight: 25, texts: []string{"{@} exagerou na catuaba e acordou na calçada sem saber o próprio nome"}, effects: []effect{injure(10, 20), delay(20 * time.Minute)}},
					{weight: 20, texts: []string{"Na muvuca, alguém passou a mão na carteira de {@}"}, effects: []effect{lose(3, 8)}},
					{weight: 15, texts: []string{"Um folião ofereceu um lança e {@} foi na onda"}, effects: []effect{buzz("lanca-perfume")}},
				}},
				{label: "👀 Só olhar da calçada", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} assistiu tudo da calçada e ainda ganhou uma latinha de um folião generoso"}, effects: []effect{gain(1, "latao")}},
				}},
				{label: "🧊 Vender latão pros foliões", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} montou um isopor na esquina e vendeu tudo em meia hora"}, effects: []effect{earn(60, 150)}},
					{weight: 40, texts: []string{"O rapa da prefeitura passou e levou o isopor, as latas e o orgulho de {@}"}, effects: []effect{lose(2, 5)}},
				}},
			},
		},
		{
			id:    "geral-jogo-no-bar",
			title: "Final no boteco",
			intros: []string{
				"Final de campeonato e o boteco do Seu Zé tá lotado. Metade com uma camisa, metade com a outra, e {@} bem no meio da confusão",
				"{@} parou no bar pra tomar uma água e caiu no meio de uma final de campeonato com aposta rolando no balcão",
			},
			choices: []choice{
				{label: "💰 Apostar no time da casa", risk: midRisk, price: cost{percent: 4, floor: 40}, endings: []ending{
					{weight: 45, texts: []string{"Gol no último minuto! {@} pulou em cima do balcão e foi recolher o dinheiro da aposta"}, effects: []effect{earn(100, 240)}},
					{weight: 55, texts: []string{"O time perdeu de goleada e o dinheiro da aposta foi junto"}},
				}},
				{label: "📺 Só assistir", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"Jogo bom, cerveja gelada e {@} ainda ganhou um petisco de quem venceu a aposta"}, effects: []effect{gain(1, "picanha", "coxinha", "latao")}},
					{weight: 30, texts: []string{"No gol, {@} comemorou tanto que derrubou o copo do vizinho e teve que pagar outro"}, effects: []effect{lose(1, 2)}},
				}},
				{label: "🗣️ Provocar a torcida rival", risk: highRisk, endings: []ending{
					{weight: 35, texts: []string{"A provocação foi tão boa que até a torcida rival riu e pagou uma rodada pra {@}"}, effects: []effect{gain(2, "latao")}},
					{weight: 65, texts: []string{"A torcida rival levantou da mesa toda de uma vez e veio na direção de {@}"}, next: "geral-briga-no-bar"},
				}},
			},
		},
		{
			id:    "geral-briga-no-bar",
			title: "A briga no boteco",
			stage: true,
			intros: []string{
				"Seis torcedores rivais cercaram {@} perto da mesa de sinuca. O Seu Zé já tá escondendo as garrafas",
			},
			choices: []choice{
				{label: "🏃 Correr pela cozinha", risk: midRisk, endings: []ending{
					{weight: 60, favor: func(p Profession) float64 { return p.escapeBonus }, texts: []string{"{@} saiu pela cozinha, pulou o muro e só parou em casa"}},
					{weight: 40, texts: []string{"{@} escorregou no óleo da cozinha e levou uns cascudos antes de fugir"}, effects: []effect{injure(10, 20)}},
				}},
				{label: "👊 Encarar no braço", risk: highRisk, endings: []ending{
					{weight: 30, favor: func(p Profession) float64 { return p.guard }, texts: []string{"{@} encarou os seis e ganhou no grito. Saiu do bar nos ombros da galera"}, effects: []effect{earn(40, 100)}},
					{weight: 50, texts: []string{"Voou cadeira, voou garrafa, e quase tudo acertou {@}"}, effects: []effect{injure(20, 45)}},
					{weight: 20, texts: []string{"A viatura chegou no meio da pancadaria e levou {@} junto"}, effects: []effect{injure(10, 20), nabbed()}},
				}},
				{label: "🍻 Pagar uma rodada de paz", risk: lowRisk, price: cost{percent: 3, floor: 60}, endings: []ending{
					{weight: 100, texts: []string{"Uma rodada pra todo mundo e a guerra virou abraço. Agora {@} tem seis amigos novos e uma dor de cabeça"}},
				}},
				{label: "🔫 Mostrar o que tem na cintura", risk: highRisk, needs: armedUp, endings: []ending{
					{weight: 70, texts: []string{"Bastou {@} levantar a camisa pros seis sentarem quietinhos e pedirem desculpa"}, effects: []effect{stain()}},
					{weight: 30, texts: []string{"Alguém filmou e mandou pra polícia. Arma na cintura na frente do bar inteiro"}, effects: []effect{nabbed()}},
				}},
			},
		},
		{
			id:    "geral-velhinho",
			title: "O velhinho na faixa",
			intros: []string{
				"Um velhinho de boina tava tentando atravessar a avenida com quatro sacolas de feira e uma bengala. Os carros não paravam e {@} viu tudo",
				"Na faixa de pedestre, um senhorzinho de suspensório olhou pra {@} com cara de quem precisava de uma mão",
			},
			choices: []choice{
				{label: "🤝 Ajudar a atravessar", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} parou o trânsito com a mão estendida e levou o velhinho até o outro lado. Ele apertou a mão de {@} e disse: \"não vou esquecer disso\""}, effects: []effect{plant("geral-velhinho-volta", 8*time.Hour)}},
					{weight: 30, texts: []string{"O velhinho agradeceu e deu uma bala de hortelã meio derretida que tava no bolso"}},
				}},
				{label: "🛍️ Levar as sacolas até a casa", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} carregou tudo até a casa dele, ganhou um café passado na hora e um \"Deus te abençoe\" que pareceu sério demais"}, effects: []effect{plant("geral-velhinho-volta", 6*time.Hour), enchant("fartura")}},
					{weight: 40, texts: []string{"Uma das sacolas era de melancia. Duas, na verdade. As costas de {@} não perdoaram"}, effects: []effect{injure(5, 15)}},
				}},
				{label: "🙈 Fingir que não viu", risk: lowRisk, endings: []ending{
					{weight: 80, texts: []string{"{@} olhou pro outro lado e seguiu. Um motoboy parou e ajudou no lugar"}},
					{weight: 20, texts: []string{"A vizinha fofoqueira viu tudo da janela, e agora o bairro inteiro sabe"}, effects: []effect{enchant("olhogordo")}},
				}},
			},
		},
		{
			id:    "geral-velhinho-volta",
			title: "O velhinho voltou",
			stage: true,
			intros: []string{
				"Lembra do velhinho da faixa? Ele achou {@}! Agora de terno, chegou num carro com motorista: \"sou o Seu Arlindo, aposentado e muito rico. Gente boa tá em falta\"",
				"Um senhor de boina bateu na porta de {@}. Era o velhinho da faixa, com um envelope na mão e sorriso de avô",
			},
			choices: []choice{
				{label: "🎁 Aceitar o presente", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"No envelope tinha um maço de notas e um bilhete: \"pra quem ainda tem bom coração\""}, effects: []effect{earn(150, 300)}},
					{weight: 40, texts: []string{"O presente era uma relíquia de família, embrulhada em papel de pão"}, effects: []effect{gain(1, "relogio-cuco", "disco-de-vinil", "trevo", "moeda-do-imperio")}},
				}},
				{label: "🙏 Dizer que não precisa", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"O Seu Arlindo se emocionou com tanta humildade e rezou um terço inteiro por {@} ali mesmo"}, effects: []effect{enchant("sorte"), enchant("protecao")}},
				}},
				{label: "☕ Aceitar um café com histórias", risk: lowRisk, endings: []ending{
					{weight: 65, texts: []string{"Foram três horas de história da época do bonde, mas {@} saiu com a receita secreta do pudim e um pudim inteiro"}, effects: []effect{gain(1, "pudim"), delay(15 * time.Minute)}},
					{weight: 35, texts: []string{"No meio da conversa, o Seu Arlindo contou onde enterrou uma lata cheia de moedas na época do cruzeiro"}, effects: []effect{gain(1, "lata-de-ninho"), earn(50, 120)}},
				}},
			},
		},
		{
			id:    "geral-crianca-perdida",
			title: "A criança perdida",
			intros: []string{
				"Uma criança de uns 5 anos agarrou a perna de {@} chorando: \"cadê minha mãe?\"",
				"No meio da muvuca, um menininho com picolé derretendo na mão perguntou se {@} podia ajudar a achar a mãe dele",
			},
			choices: []choice{
				{label: "🧒 Ajudar a achar a mãe", risk: lowRisk, endings: []ending{
					{weight: 65, texts: []string{"Depois de rodar um quarteirão de mãos dadas, acharam a mãe, que abraçou {@} chorando e insistiu em dar um agrado"}, effects: []effect{earn(20, 60), enchant("protecao")}},
					{weight: 35, texts: []string{"A mãe apareceu gritando \"SOLTA MEU FILHO\" e {@} passou meia hora explicando pro segurança"}, effects: []effect{delay(15 * time.Minute)}},
				}},
				{label: "📢 Chamar o segurança", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"O segurança anunciou no alto-falante e a mãe apareceu em dois minutos. Missão cumprida"}},
				}},
				{label: "🍭 Comprar um doce pra acalmar", risk: lowRisk, price: cost{floor: 10}, endings: []ending{
					{weight: 60, texts: []string{"Com o pirulito na boca, a criança parou de chorar e apontou: \"ali minha mãe!\". A mãe pagou o doce em dobro"}, effects: []effect{earn(10, 30)}},
					{weight: 40, texts: []string{"A criança comeu o doce, pediu outro e saiu correndo sozinha atrás do carrinho de sorvete"}},
				}},
				{label: "🚔 Acionar a ronda", risk: lowRisk, needs: lawman, endings: []ending{
					{weight: 100, texts: []string{"Pelo rádio, {@} achou a mãe em cinco minutos e ganhou elogio do comando"}, effects: []effect{earn(40, 90), enchant("justica")}},
				}},
			},
		},
		{
			id:    "geral-fofoca",
			title: "A vizinha fofoqueira",
			intros: []string{
				"Dona Cleide, a vizinha que sabe de tudo, chamou {@} na janela: \"vem cá, me conta uma coisa...\"",
				"A Dona Cleide largou a novela, debruçou na janela e fez sinal pra {@} chegar perto. Lá vem fofoca",
			},
			choices: []choice{
				{label: "🗣️ Contar tudo da sua vida", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Em dez minutos o bairro inteiro sabia até o saldo de {@}"}, effects: []effect{enchant("olhogordo")}},
					{weight: 50, texts: []string{"Em troca, a Dona Cleide deu um pedaço de bolo e uma fofoca quentíssima que valia ouro"}, effects: []effect{gain(1, "pudim"), earn(10, 40)}},
				}},
				{label: "🤐 Desconversar", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} falou do tempo, do preço do tomate e escapou da entrevista"}},
				}},
				{label: "😈 Inventar uma fofoca", risk: midRisk, endings: []ending{
					{weight: 40, texts: []string{"A fofoca inventada fez tanto sucesso que a Dona Cleide pagou o café de {@}"}, effects: []effect{earn(20, 60)}},
					{weight: 40, texts: []string{"A fofoca chegou no ouvido do vizinho errado, que veio tirar satisfação na mão"}, effects: []effect{injure(5, 15)}},
					{weight: 20, texts: []string{"O vizinho processou {@} por calúnia e ganhou"}, effects: []effect{lose(4, 8)}},
				}},
				{label: "🕵️ Pedir a fofoca do crime do bairro", risk: lowRisk, needs: lawman, endings: []ending{
					{weight: 100, texts: []string{"A Dona Cleide entregou de bandeja quem anda roubando os varais. Caso resolvido e recompensa garantida"}, effects: []effect{earn(60, 140)}},
				}},
			},
		},
		{
			id:    "geral-mendigo",
			title: "O mendigo misterioso",
			intros: []string{
				"Um senhor de barba branca e roupa rasgada estendeu a mão pra {@}: \"um trocado pro café, meu bem?\". Só que o relógio no pulso dele parece de ouro",
				"Sentado na calçada, um homem maltrapilho pediu ajuda a {@}. Ele tem um olhar estranho, de quem tá testando alguém",
			},
			choices: []choice{
				{label: "🪙 Dar um trocado", risk: lowRisk, price: cost{floor: 10}, endings: []ending{
					{weight: 55, texts: []string{"Ele sorriu e disse: \"Deus te pague em dobro\". E parece que vai pagar"}, effects: []effect{enchant("fartura")}},
					{weight: 35, texts: []string{"Era um milionário excêntrico testando as pessoas. Ele tirou um talão de cheque do casaco rasgado"}, effects: []effect{earn(100, 250)}},
					{weight: 10, texts: []string{"Ele foi direto pro bar comprar uma cachaça e brindou à saúde de {@}"}},
				}},
				{label: "🍞 Pagar um lanche", risk: lowRisk, price: cost{floor: 25}, endings: []ending{
					{weight: 70, texts: []string{"Ele comeu devagar, agradeceu e benzeu {@} com três sinais da cruz"}, effects: []effect{enchant("sorte")}},
					{weight: 30, texts: []string{"No fim do lanche ele revelou: era o dono da padaria inteira, disfarçado. Pagou tudo de volta e mais um pouco"}, effects: []effect{earn(150, 300)}},
				}},
				{label: "💬 Puxar conversa", risk: lowRisk, endings: []ending{
					{weight: 50, texts: []string{"Ele contou que já foi dono de banco e deu uma dica de investimento que funcionou"}, effects: []effect{earn(20, 80)}},
					{weight: 50, texts: []string{"Ele falou por duas horas seguidas sobre a ex-mulher"}, effects: []effect{delay(20 * time.Minute)}},
				}},
				{label: "🙄 Ignorar", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"O homem só anotou alguma coisa num caderninho e voltou a olhar o movimento"}},
				}},
			},
		},
		{
			id:    "geral-gato-preto",
			title: "O gato preto",
			intros: []string{
				"Um gato preto atravessou bem na frente de {@}, parou no meio do caminho e ficou encarando sem piscar",
				"Um gato preto pulou do muro e sentou na frente de {@}. Os dois se encararam por um minuto inteiro",
			},
			choices: []choice{
				{label: "🐈‍⬛ Fazer carinho", risk: midRisk, endings: []ending{
					{weight: 45, texts: []string{"O gato ronronou, gostou de {@} e foi junto pra casa"}, effects: []effect{gain(1, "gato-preto")}},
					{weight: 30, texts: []string{"O gato aceitou o carinho por exatos dois segundos e meteu a unha"}, effects: []effect{injure(3, 8)}},
					{weight: 25, texts: []string{"Superstição é superstição: o resto do dia de {@} foi uma desgraça"}, effects: []effect{enchant("azar")}},
				}},
				{label: "↩️ Dar a volta no quarteirão", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} respeitou a tradição e deu a volta no quarteirão. Demorou mais, mas chegou sem azar"}, effects: []effect{delay(10 * time.Minute)}},
				}},
				{label: "🧂 Jogar sal por cima do ombro", risk: lowRisk, uses: "sal-grosso", endings: []ending{
					{weight: 100, texts: []string{"{@} jogou sal grosso por cima do ombro esquerdo e anulou o azar com sobra"}, effects: []effect{enchant("protecao")}},
				}},
				{label: "🙏 Benzer o gato", risk: lowRisk, needs: inArea(areaFe), endings: []ending{
					{weight: 100, texts: []string{"{@} benzeu o gato com três galhos de arruda. Ele miou em aprovação e a sorte virou"}, effects: []effect{enchant("sorte")}},
				}},
			},
		},
		{
			id:    "geral-tombo-motoboy",
			title: "O tombo do motoboy",
			intros: []string{
				"Um motoboy derrapou na areia e caiu bem na frente de {@}. Tá no chão gemendo, com a mochila de entrega aberta e o lanche espalhado",
				"BAM! Uma moto passou reto na curva e o piloto foi parar na calçada, aos pés de {@}",
			},
			choices: []choice{
				{label: "📞 Ligar pro SAMU", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"O SAMU chegou rápido e o motoboy, agradecido, deu pra {@} o envelope de gorjeta do dia"}, effects: []effect{gain(1, "gorjeta")}},
				}},
				{label: "🏍️ Levantar a moto", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} levantou a moto, ajudou o motoboy a terminar as entregas e dividiu a gorjeta"}, effects: []effect{earn(40, 100)}},
					{weight: 40, texts: []string{"A moto escorregou e caiu em cima do pé de {@}"}, effects: []effect{injure(10, 20)}},
				}},
				{label: "📱 Filmar pro grupo", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O vídeo bombou no grupo do bairro e {@} ganhou uns likes. Só likes"}},
					{weight: 50, texts: []string{"O povo chamou {@} de abutre nos comentários"}, effects: []effect{enchant("olhogordo")}},
				}},
				{label: "🩺 Fazer os primeiros socorros", risk: lowRisk, needs: inArea(areaSaude), endings: []ending{
					{weight: 80, texts: []string{"{@} imobilizou o braço, estancou o sangue e o motoboy saiu sorrindo. A empresa de entrega mandou uma recompensa"}, effects: []effect{earn(100, 200), enchant("sorte")}},
					{weight: 20, texts: []string{"O motoboy vomitou de nervoso no meio do atendimento, mas sobreviveu inteirinho graças a {@}"}},
				}},
				{label: "🩹 Usar o kit de primeiros socorros", risk: lowRisk, uses: "kit-de-primeiros-socorros", endings: []ending{
					{weight: 100, texts: []string{"{@} abriu o kit e fez um curativo digno de série médica. O motoboy prometeu nunca esquecer"}, effects: []effect{earn(80, 160), enchant("protecao")}},
				}},
			},
		},
		{
			id:    "geral-festa-estranha",
			title: "A festa estranha",
			when:  atNight,
			intros: []string{
				"Três da manhã e {@} foi parar numa festa estranha com gente esquisita, luz piscando e um DJ de capa",
				"Madrugada adentro, um conhecido de um conhecido arrastou {@} pra um after num galpão abandonado",
			},
			choices: []choice{
				{label: "💊 Aceitar a bala oferecida", risk: highRisk, endings: []ending{
					{weight: 55, texts: []string{"A bala bateu e {@} abraçou todo mundo da festa dizendo que ama cada um"}, effects: []effect{buzz("bala")}},
					{weight: 25, texts: []string{"A bala bateu errado e {@} passou o resto da festa no banheiro"}, effects: []effect{injure(15, 30)}},
					{weight: 20, texts: []string{"Era uma batida policial disfarçada de festa"}, effects: []effect{buzz("bala"), nabbed()}},
				}},
				{label: "🕺 Dançar até o sol nascer", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} dançou tanto que ganhou o concurso de dança da festa"}, effects: []effect{earn(40, 100)}},
					{weight: 45, texts: []string{"{@} dançou até o sol nascer e o corpo cobrou a conta"}, effects: []effect{injure(5, 15), delay(20 * time.Minute)}},
				}},
				{label: "🛏️ Ir pra casa dormir", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} fez a coisa mais sensata da noite e acordou zerado"}, effects: []effect{heal(15)}},
				}},
			},
		},
		{
			id:    "geral-cano-estourado",
			title: "O cano estourado",
			when:  homeOwner,
			intros: []string{
				"O vizinho ligou desesperado: tá vazando água da casa de {@} pra rua inteira. Um cano estourou",
				"{@} chegou em casa e ouviu um barulho de cachoeira. Não era a TV: era um cano estourado dentro da parede",
			},
			choices: []choice{
				{label: "🔧 Consertar sozinho", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Com fita veda-rosca e muito vídeo tutorial, {@} resolveu. Orgulho de quem tem casa própria"}},
					{weight: 50, texts: []string{"{@} apertou o cano errado, levou um jato na cara e o conserto saiu mais caro"}, effects: []effect{injure(3, 10), lose(2, 4)}},
				}},
				{label: "📞 Chamar o encanador", risk: lowRisk, price: cost{percent: 2, floor: 80}, endings: []ending{
					{weight: 100, texts: []string{"O encanador chegou, olhou, coçou a cabeça e resolveu em dez minutos. Cobrou pela sabedoria"}},
				}},
				{label: "🙈 Fingir que não é com você", risk: midRisk, endings: []ending{
					{weight: 40, texts: []string{"A água achou o caminho do bueiro sozinha e o problema se resolveu. Sorte de principiante"}},
					{weight: 60, texts: []string{"O teto do vizinho de baixo desabou e ele mandou a conta pra {@}"}, effects: []effect{lose(4, 9)}},
				}},
				{label: "🪠 Resolver com macete de obra", risk: lowRisk, needs: inArea(areaObras), endings: []ending{
					{weight: 100, texts: []string{"{@} resolveu com um macete que nenhum encanador conhece e ainda cobrou do vizinho o conserto do cano dele"}, effects: []effect{earn(30, 80), refresh()}},
				}},
			},
		},
		{
			id:    "geral-coletiva",
			title: "A coletiva de imprensa",
			when:  inCharge,
			intros: []string{
				"Microfones, câmeras e flashes: a imprensa cercou {@}, presidente da conversa, querendo uma declaração sobre a crise",
				"\"Presidente! Presidente! Uma palavrinha!\" Os repórteres cercaram {@} na saída do palácio",
			},
			choices: []choice{
				{label: "🎤 Dar uma entrevista", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} falou bonito, a aprovação subiu e choveram doações de apoiadores"}, effects: []effect{earn(100, 250)}},
					{weight: 30, texts: []string{"{@} trocou o nome do país ao vivo. O vídeo tá em todos os grupos"}, effects: []effect{enchant("olhogordo")}},
					{weight: 20, texts: []string{"Um repórter perguntou sobre as contas da campanha e {@} gaguejou. Escândalo"}, effects: []effect{lose(4, 9)}},
				}},
				{label: "🚪 Sair pelos fundos", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} saiu pela porta da cozinha e a imprensa ficou entrevistando o segurança"}},
				}},
				{label: "💸 Comprar o silêncio do jornal", risk: midRisk, price: cost{percent: 8, floor: 300}, endings: []ending{
					{weight: 70, texts: []string{"O jornal publicou uma matéria sobre filhotes de panda no lugar. Missão cumprida"}},
					{weight: 30, texts: []string{"O jornal publicou na capa: PRESIDENTE TENTA COMPRAR O SILÊNCIO DO JORNAL"}, effects: []effect{enchant("excomunhao")}},
				}},
			},
		},
		{
			id:    "geral-almoco-domingo",
			title: "Almoço de domingo",
			when:  onSunday,
			intros: []string{
				"Domingo, meio-dia, cheiro de lasanha: a vó de {@} ligou intimando pro almoço de família",
				"A família inteira de {@} se reuniu no domingo: o tio do pavê, a prima que vende cosmético e a vó com a colher de pau",
			},
			choices: []choice{
				{label: "🍗 Comer até passar mal", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} repetiu quatro vezes e ainda levou marmita pra semana"}, effects: []effect{heal(20), gain(1, "marmita")}},
					{weight: 40, texts: []string{"{@} exagerou na maionese e passou a tarde gemendo no sofá"}, effects: []effect{injure(5, 15)}},
				}},
				{label: "🗣️ Discutir política com o tio", risk: midRisk, endings: []ending{
					{weight: 30, texts: []string{"{@} ganhou a discussão e o tio, derrotado, pagou a sobremesa"}, effects: []effect{earn(20, 60)}},
					{weight: 45, texts: []string{"A discussão virou guerra e metade da família parou de falar com {@}"}, effects: []effect{enchant("encosto")}},
					{weight: 25, texts: []string{"O tio jogou uma coxa de frango na testa de {@}"}, effects: []effect{injure(5, 12)}},
				}},
				{label: "🍽️ Lavar a louça", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"A vó viu {@} lavando a louça e deu um dinheirinho escondido da família"}, effects: []effect{earn(30, 80)}},
					{weight: 30, texts: []string{"A vó encheu as mãos de {@} de pudim pra levar"}, effects: []effect{gain(1, "pudim")}},
				}},
			},
		},
		{
			id:    "geral-benzedeira",
			title: "A benzedeira da esquina",
			when:  bruised,
			intros: []string{
				"Mancando, com roxo pra todo lado, {@} passou na frente da casa da Dona Benedita, a benzedeira, que chamou da janela: \"entra, que eu vi esse quebranto de longe\"",
				"A Dona Benedita, benzedeira do bairro, viu os machucados de {@} e já foi pegando os galhos de arruda",
			},
			choices: []choice{
				{label: "🌿 Deixar benzer", risk: lowRisk, endings: []ending{
					{weight: 80, texts: []string{"Três galhos de arruda murcharam e a dor de {@} foi embora junto"}, effects: []effect{heal(25), enchant("protecao")}},
					{weight: 20, texts: []string{"A arruda murchou na hora. A Dona Benedita disse que era coisa forte e mandou voltar outro dia"}, effects: []effect{heal(5)}},
				}},
				{label: "🏥 Dizer que prefere médico", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"A Dona Benedita resmungou que médico não tira olho gordo, mas deu um copo d'água e um pão"}, effects: []effect{heal(5)}},
				}},
				{label: "🍵 Tomar o chá dela", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"O chá de sete ervas fez milagre"}, effects: []effect{heal(35)}},
					{weight: 40, texts: []string{"O chá era laxante. {@} passou a próxima meia hora em lugar nenhum além do banheiro"}, effects: []effect{delay(20 * time.Minute)}},
				}},
			},
		},
		{
			id:    "geral-larica",
			title: "A larica",
			when:  stoned,
			intros: []string{
				"{@} tá sob efeito e bateu uma larica monstra. A geladeira tá vazia, e a do vizinho tá logo ali",
				"A brisa de {@} deu lugar a uma fome de leão. Tudo parece comestível, até o sabonete",
			},
			choices: []choice{
				{label: "🍕 Pedir uma pizza", risk: lowRisk, price: cost{floor: 50}, endings: []ending{
					{weight: 100, texts: []string{"A pizza chegou em 20 minutos, que pareceram três dias. Melhor pizza da vida de {@}"}, effects: []effect{gain(1, "pizza"), heal(5)}},
				}},
				{label: "🧊 Assaltar a geladeira do vizinho", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} voltou com os braços cheios de coisa gostosa"}, effects: []effect{gain(2, "chocolate", "pudim", "coxinha", "brigadeiro")}},
					{weight: 30, texts: []string{"O vizinho acordou e pegou {@} com a cara enfiada no pote de sorvete"}, effects: []effect{lose(2, 5)}},
					{weight: 20, texts: []string{"O vizinho achou que era ladrão e chamou a polícia"}, effects: []effect{stain()}},
				}},
				{label: "😴 Dormir e esquecer", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} dormiu 14 horas seguidas e acordou com outra cara"}, effects: []effect{heal(10)}},
					{weight: 40, texts: []string{"{@} dormiu tanto que perdeu o horário de tudo"}, effects: []effect{delay(15 * time.Minute)}},
				}},
				{label: "🍫 Comer o chocolate da mochila", risk: lowRisk, uses: "chocolate", endings: []ending{
					{weight: 100, texts: []string{"Melhor chocolate já comido na história da humanidade. Fome resolvida"}, effects: []effect{heal(10)}},
				}},
			},
		},
	},
}
