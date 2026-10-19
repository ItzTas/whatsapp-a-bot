package economy

import "time"

var professionEvents = eventBook{
	name:   "🧑‍💼 Profissões",
	places: []string{"!trabalhar"},
	scenarios: []scenario{
		{
			id:    "prof-medico-plantao",
			title: "O plantão do caos",
			jobs:  []string{"médico"},
			intros: []string{
				"Três da manhã no pronto-socorro: entra um motoqueiro todo estourado, um bêbado com um anzol enfiado no nariz e uma senhora jurando que tá enfartando. {@} é o único médico de plantão",
				"O plantão de {@} tava calmo até a porta abrir de uma vez: motoqueiro sangrando, bêbado com anzol no nariz e uma vovó gritando que vai morrer. Tudo ao mesmo tempo",
			},
			choices: []choice{
				{label: "🏍️ Atender o motoqueiro", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} costurou o motoqueiro inteiro e ele saiu andando. A família fez até vaquinha pra agradecer"}, effects: []effect{earn(80, 160)}},
					{weight: 40, texts: []string{"No meio da sutura, o motoqueiro parou de respirar"}, next: "prof-medico-mesa"},
				}},
				{label: "👵 Atender a vovó", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"Era só gases. A vovó agradeceu com um pote de pudim"}, effects: []effect{gain(1, "pudim"), earn(20, 50)}},
					{weight: 30, texts: []string{"Era gases mesmo, e a vovó soltou todos de uma vez no meio da consulta. {@} vai precisar de um tempo pra se recuperar"}, effects: []effect{delay(10 * time.Minute)}},
				}},
				{label: "🎣 Tirar o anzol do bêbado", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} tirou o anzol com um alicate e o bêbado ainda deu de presente as iscas que tavam no bolso"}, effects: []effect{gain(2, "isca")}},
					{weight: 40, texts: []string{"O bêbado acordou no meio do procedimento, deu um soco em {@} e voltou a dormir"}, effects: []effect{injure(5, 15)}},
				}},
				{label: "☕ Tomar um café antes", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} voltou do café elétrico e resolveu os três casos em vinte minutos"}, effects: []effect{earn(60, 120)}},
					{weight: 50, texts: []string{"Quando {@} voltou, a diretoria tava esperando na porta com a planilha de atrasos"}, effects: []effect{lose(3, 6), delay(20 * time.Minute)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, a enfermeira-chefe resolveu tudo sozinha. Vai ter fofoca no grupo do hospital"}, effects: []effect{delay(10 * time.Minute)}},
		},
		{
			id:    "prof-medico-mesa",
			title: "Parada na mesa",
			stage: true,
			intros: []string{
				"O monitor apitou reto. O motoqueiro parou na mesa de {@} e a sala inteira olhou pro doutor",
			},
			choices: []choice{
				{label: "⚡ Usar o desfibrilador", risk: lowRisk, needs: holds("desfibrilador"), endings: []ending{
					{weight: 85, texts: []string{"CLEAR! O coração voltou no primeiro choque e {@} virou lenda no hospital"}, effects: []effect{earn(150, 300), enchant("prosperidade")}},
					{weight: 15, texts: []string{"Voltou só no terceiro choque, mas voltou"}, effects: []effect{earn(80, 150)}},
				}},
				{label: "💪 Fazer massagem cardíaca", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} fez massagem até o braço tremer, e o coração voltou a bater"}, effects: []effect{earn(100, 200), injure(5, 10)}},
					{weight: 45, texts: []string{"Não teve jeito. A família culpou {@} e abriu processo no CRM"}, effects: []effect{lose(5, 10)}},
				}},
				{label: "🙏 Chamar o capelão", risk: lowRisk, endings: []ending{
					{weight: 30, texts: []string{"O capelão rezou e, do nada, o monitor voltou a apitar. Ninguém no hospital explica até hoje"}, effects: []effect{earn(50, 100), enchant("protecao")}},
					{weight: 70, texts: []string{"O capelão rezou, mas o motoqueiro já tinha ido. E parece que o espírito dele resolveu seguir {@} pra casa"}, effects: []effect{enchant("encosto")}},
				}},
			},
		},
		{
			id:     "prof-medico-vip",
			title:  "O paciente milionário",
			rarity: rareEvent,
			jobs:   []string{"médico"},
			intros: []string{
				"Um empresário coberto de ouro entrou no consultório de {@} com uma dor nas costas e uma proposta: \"Doutor, me dá um atestado de seis meses que eu te faço rico\"",
				"De relógio dourado e motorista na porta, um empresário sentou na frente de {@} e foi direto: \"Seis meses de atestado. Diz o seu preço\"",
			},
			choices: []choice{
				{label: "📝 Assinar o atestado", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"O empresário pagou em dinheiro vivo e sumiu num carro preto"}, effects: []effect{earn(300, 600), stain()}},
					{weight: 35, texts: []string{"Era uma operação da Polícia Federal: o \"empresário\" tinha distintivo no bolso"}, effects: []effect{nabbed(), lose(5, 10)}},
					{weight: 15, texts: []string{"O CRM descobriu o atestado fajuto e {@} pagou uma multa salgada"}, effects: []effect{lose(10, 20)}},
				}},
				{label: "🩻 Examinar direito", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"Era hérnia de disco de verdade. O empresário ficou tão grato que pagou a consulta em dobro"}, effects: []effect{earn(150, 250)}},
					{weight: 30, texts: []string{"Era preguiça mesmo. Ele saiu xingando e deu uma estrela no Google, mas pagou a consulta"}, effects: []effect{earn(20, 40)}},
				}},
				{label: "🚔 Denunciar a proposta", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"A polícia prendeu o espertalhão e {@} ainda levou uma recompensa da Receita"}, effects: []effect{earn(150, 300), enchant("justica")}},
					{weight: 40, texts: []string{"O empresário tinha amigos. Uma semana depois, o carro de {@} amanheceu riscado de ponta a ponta"}, effects: []effect{lose(3, 8)}},
				}},
			},
		},
		{
			id:    "prof-policial-propina",
			title: "O cafezinho da blitz",
			jobs:  []string{"policial"},
			intros: []string{
				"Na blitz, {@} parou um carrão com o motorista sem habilitação, documento vencido e um cheiro suspeito. O cara já abriu a carteira: \"Vamos resolver isso aqui mesmo, chefe?\"",
				"O motorista do carro importado nem desceu: baixou o vidro, mostrou um maço de notas pra {@} e piscou. \"Um cafezinho pro senhor, e eu sigo viagem\"",
			},
			choices: []choice{
				{label: "💰 Aceitar o cafezinho", risk: highRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} embolsou o cafezinho, bateu continência e liberou o carro"}, effects: []effect{earn(100, 250), stain()}},
					{weight: 30, texts: []string{"Era um corregedor disfarçado testando os policiais da blitz"}, effects: []effect{nabbed(), lose(5, 10)}},
					{weight: 15, texts: []string{"O motorista filmou tudo e postou. Agora {@} é meme nacional e ainda tomou advertência"}, effects: []effect{lose(8, 15), enchant("olhogordo")}},
				}},
				{label: "🚔 Levar pra delegacia", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"Na delegacia, acharam um mandado de prisão contra o cara. {@} ganhou elogio do delegado"}, effects: []effect{earn(80, 150)}},
					{weight: 30, texts: []string{"O cara era sobrinho de deputado. {@} foi transferido pra ronda no lixão"}, effects: []effect{delay(30 * time.Minute)}},
				}},
				{label: "🐕 Revistar o carro", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Tinha um pacote de pó escondido no estepe. {@} fez a apreensão do ano e ganhou medalha"}, effects: []effect{earn(100, 200), enchant("justica")}},
					{weight: 30, texts: []string{"No porta-luvas só tinha um pastel velho e um boleto vencido"}, effects: []effect{gain(1, "boleto")}},
					{weight: 20, texts: []string{"O motorista arrancou no meio da revista e arrastou {@} uns bons metros"}, effects: []effect{injure(15, 30)}},
				}},
			},
		},
		{
			id:    "prof-policial-telhado",
			title: "A perseguição no telhado",
			jobs:  []string{"policial"},
			intros: []string{
				"Um ladrão passou correndo com a bolsa de uma senhora e pulou pro telhado. A senhora gritou \"PEGA, SEU POLICIAL!\" e a rua inteira olhou pra {@}",
				"O trombadinha arrancou a bolsa da vovó e subiu pelo muro igual lagartixa. Agora tá correndo pelos telhados e {@} é o único policial na esquina",
			},
			choices: []choice{
				{label: "🏃 Correr pelo telhado", risk: highRisk, endings: []ending{
					{weight: 50, favor: func(p Profession) float64 { return p.guard }, texts: []string{"{@} saltou de telhado em telhado igual filme de ação e algemou o ladrão"}, effects: []effect{earn(100, 200)}},
					{weight: 35, texts: []string{"A telha quebrou e {@} caiu dentro da sala de uma família no meio do almoço"}, effects: []effect{injure(20, 40)}},
					{weight: 15, texts: []string{"{@} escorregou, caiu de três metros e acordou no hospital sem lembrar do próprio nome"}, effects: []effect{injure(50, 70)}},
				}},
				{label: "📻 Pedir reforço no rádio", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"O reforço cercou a quadra e pegou o ladrão. {@} levou o crédito no relatório"}, effects: []effect{earn(50, 100)}},
					{weight: 40, texts: []string{"O reforço chegou uma hora depois, comendo pastel. O ladrão já tava em outro bairro"}, effects: []effect{gain(1, "pastel")}},
				}},
				{label: "🔫 Tiro de advertência", risk: midRisk, needs: armedUp, endings: []ending{
					{weight: 50, texts: []string{"O ladrão se rendeu na hora e devolveu a bolsa com tudo dentro"}, effects: []effect{earn(80, 160)}},
					{weight: 50, texts: []string{"O tiro acertou a caixa d'água do vizinho, que inundou a casa. A conta veio pra {@}"}, effects: []effect{lose(5, 12)}},
				}},
				{label: "🤷 Fingir que não viu", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} foi tomar um café na padaria e deixou a perseguição pra outro dia"}, effects: []effect{gain(1, "cafe")}},
					{weight: 40, texts: []string{"A senhora era mãe do comandante. {@} levou uma bronca histórica no batalhão"}, effects: []effect{lose(3, 6), delay(20 * time.Minute)}},
				}},
			},
		},
		{
			id:    "prof-uber-fuga",
			title: "A corrida da fuga",
			jobs:  []string{"motorista de aplicativo"},
			intros: []string{
				"Um passageiro entrou no carro de {@} com uma mochila pesada, suando frio: \"Acelera, motorista, que eu pago o triplo\". Lá atrás, uma sirene começou a tocar",
				"O passageiro mal fechou a porta e já gritou: \"Vai, vai, VAI!\". No retrovisor de {@}, duas viaturas viraram a esquina",
			},
			choices: []choice{
				{label: "🏎️ Acelerar", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"{@} costurou o trânsito igual filme de ação e o passageiro pagou em notas de cem"}, effects: []effect{earn(150, 300), stain()}},
					{weight: 60, texts: []string{"A viatura grudou na traseira!"}, next: "prof-uber-viatura"},
				}},
				{label: "🛑 Parar o carro", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O passageiro desceu xingando e fugiu a pé. Os policiais agradeceram {@}"}, effects: []effect{earn(20, 60)}},
					{weight: 30, texts: []string{"O passageiro desceu correndo e esqueceu uma mala no banco de trás"}, effects: []effect{gain(1, "mala")}},
				}},
				{label: "📍 Rumo à delegacia", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} fingiu seguir o GPS e parou bem na porta da delegacia. O passageiro tinha recompensa pela cabeça"}, effects: []effect{earn(100, 200)}},
					{weight: 40, texts: []string{"O passageiro percebeu o caminho e pulou do carro andando, arrancando o retrovisor junto"}, effects: []effect{lose(3, 6)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} hesitava, o passageiro abriu a porta no sinal vermelho e sumiu sem pagar"}},
		},
		{
			id:    "prof-uber-viatura",
			title: "Viatura na cola",
			stage: true,
			intros: []string{
				"A viatura tá colada no para-choque de {@} com a sirene ligada. No banco de trás, o passageiro reza um terço com a mochila no colo",
			},
			choices: []choice{
				{label: "🏁 Despistar no beco", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"{@} entrou num beco, apagou os faróis e a viatura passou reto. O passageiro pagou o quádruplo"}, effects: []effect{earn(200, 400), stain()}},
					{weight: 35, texts: []string{"O beco não tinha saída"}, effects: []effect{nabbed(), lose(5, 10)}},
					{weight: 20, texts: []string{"{@} enfiou o carro num poste tentando fugir"}, effects: []effect{injure(30, 50), nabbed()}},
				}},
				{label: "🙌 Encostar e se render", risk: midRisk, endings: []ending{
					{weight: 80, texts: []string{"{@} explicou que era só o motorista. Depois de três horas de depoimento, liberaram"}, effects: []effect{delay(30 * time.Minute)}},
					{weight: 20, texts: []string{"Os policiais não acreditaram e levaram {@} junto no camburão"}, effects: []effect{nabbed()}},
				}},
				{label: "🚪 Mandar o passageiro descer", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"O passageiro pulou do carro andando, a viatura foi atrás dele e {@} saiu de fininho com o Pix da corrida"}, effects: []effect{earn(50, 100)}},
					{weight: 40, texts: []string{"O passageiro desceu, mas a polícia parou {@} mesmo assim e achou o resto da muamba no banco"}, effects: []effect{nabbed()}},
				}},
			},
		},
		{
			id:    "prof-uber-parto",
			title: "Bebê a caminho",
			jobs:  []string{"motorista de aplicativo"},
			intros: []string{
				"A passageira de {@} começou a gritar no meio da corrida: a bolsa estourou e o hospital fica do outro lado da cidade, bem na hora do rush",
				"\"Moço, acho que o bebê vai nascer AGORA\", avisou a passageira, segurando no banco. O GPS de {@} mostra 47 minutos até a maternidade",
			},
			choices: []choice{
				{label: "🚨 Pisar fundo até o hospital", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} chegou na maternidade em tempo recorde. O bebê quase ganhou o nome do aplicativo"}, effects: []effect{earn(80, 160), enchant("sorte")}},
					{weight: 40, texts: []string{"Chegaram a tempo, mas {@} passou em três radares no caminho"}, effects: []effect{gain(3, "multa"), lose(2, 5)}},
				}},
				{label: "👶 Fazer o parto no carro", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"Com um tutorial no volume máximo, {@} fez o parto no banco de trás. Nasceu uma menina linda"}, effects: []effect{earn(100, 200), enchant("fartura")}},
					{weight: 35, texts: []string{"O parto deu certo, mas o banco do carro nunca mais vai ser o mesmo"}, effects: []effect{lose(5, 10)}},
					{weight: 15, texts: []string{"{@} desmaiou ao ver sangue e bateu a cabeça no volante"}, effects: []effect{injure(15, 30)}},
				}},
				{label: "📞 Chamar o SAMU", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O SAMU chegou rapidinho e a passageira ainda deu cinco estrelas"}, effects: []effect{earn(20, 50)}},
					{weight: 30, texts: []string{"O SAMU demorou e a passageira deu uma estrela por \"falta de iniciativa\""}, effects: []effect{delay(15 * time.Minute)}},
				}},
			},
		},
		{
			id:    "prof-coach-brasa",
			title: "Imersão na brasa",
			jobs:  []string{"coach"},
			intros: []string{
				"Na imersão de fim de semana, os alunos de {@} exigem o ponto alto prometido no folheto: andar descalço na brasa pra \"queimar as crenças limitantes\"",
				"Trinta alunos de pulseirinha VIP cercaram a fogueira gritando o nome de {@}. A brasa tá vermelha e o folheto prometia \"travessia do fogo interior\"",
			},
			choices: []choice{
				{label: "🔥 Andar na brasa primeiro", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} atravessou gritando ACREDITA e nem sentiu. Os alunos compraram a mentoria VIP na hora"}, effects: []effect{earn(150, 300)}},
					{weight: 40, texts: []string{"{@} queimou a sola dos dois pés e saiu pulando igual saci, mas vendeu o curso assim mesmo"}, effects: []effect{injure(20, 35), earn(30, 60)}},
					{weight: 10, texts: []string{"A calça de linho pegou fogo no meio da travessia"}, effects: []effect{injure(40, 60)}},
				}},
				{label: "🧊 Trocar por banho de gelo", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"Ninguém percebeu a troca e todo mundo achou ainda mais profundo"}, effects: []effect{earn(60, 120)}},
					{weight: 40, texts: []string{"Um aluno pediu reembolso por propaganda enganosa"}, effects: []effect{lose(2, 5)}},
				}},
				{label: "🍺 Liberar churrasco e cerveja", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"A imersão virou confraternização e todo mundo deu nota dez"}, effects: []effect{earn(80, 150)}},
					{weight: 45, texts: []string{"Alguém postou a \"imersão\" no Instagram e virou piada no país inteiro"}, effects: []effect{enchant("olhogordo"), lose(2, 5)}},
				}},
			},
		},
		{
			id:    "prof-coach-piramide",
			title: "A oportunidade de negócio",
			jobs:  []string{"coach"},
			intros: []string{
				"Um ex-aluno de {@} apareceu de terno brilhante oferecendo sociedade numa \"rede de marketing multinível revolucionária\". \"É só trazer mais três amigos, coach\"",
				"\"Coach, você me ensinou a pensar grande\", disse o ex-aluno, abrindo uma planilha com um triângulo desenhado. \"Agora é sua vez de entrar no topo\"",
			},
			choices: []choice{
				{label: "📈 Virar sócio", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"O esquema deu dinheiro de verdade. Por enquanto"}, effects: []effect{earn(200, 400), stain(), plant("prof-coach-piramide-cai", 8*time.Hour)}},
					{weight: 35, texts: []string{"A pirâmide desabou na primeira semana e levou o investimento de {@} junto"}, effects: []effect{lose(10, 20)}},
					{weight: 25, texts: []string{"A Polícia Federal fez operação na sede bem no dia da reunião de sócios"}, effects: []effect{nabbed(), lose(5, 10)}},
				}},
				{label: "🎤 Vender curso sobre pirâmides", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} lançou o curso \"Como não cair em pirâmide\" e vendeu igual água"}, effects: []effect{earn(100, 200)}},
					{weight: 40, texts: []string{"Sem querer, o curso também virou uma pirâmide"}, effects: []effect{earn(50, 100), stain()}},
				}},
				{label: "🙅 Recusar", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} recusou e ainda postou um textão sobre integridade no LinkedIn. Rendeu três curtidas"}, effects: []effect{earn(10, 30)}},
				}},
			},
		},
		{
			id:    "prof-coach-piramide-cai",
			title: "A pirâmide desabou",
			stage: true,
			intros: []string{
				"Lembra da \"rede revolucionária\" do ex-aluno de {@}? Desabou. Tem uma fila de gente furiosa na porta querendo o dinheiro de volta",
			},
			choices: []choice{
				{label: "🏃 Fugir pela janela", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} pulou a janela e sumiu do mapa por umas horas"}, effects: []effect{delay(30 * time.Minute)}},
					{weight: 40, texts: []string{"{@} pulou a janela e caiu em cima de uma roseira"}, effects: []effect{injure(10, 20)}},
				}},
				{label: "💸 Devolver parte do dinheiro", risk: midRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} devolveu uma parte e saiu com a reputação meio inteira"}, effects: []effect{lose(8, 15)}},
				}},
				{label: "🎭 Dizer que também foi vítima", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"Colou! {@} até deu entrevista no jornal como vítima do golpe"}, effects: []effect{earn(50, 100)}},
					{weight: 50, texts: []string{"Ninguém acreditou e a fila chamou a polícia"}, effects: []effect{nabbed()}},
				}},
			},
		},
		{
			id:    "prof-palhaco-festa",
			title: "A festa do terror",
			jobs:  []string{"palhaço"},
			intros: []string{
				"Na festa de aniversário, trinta crianças cercaram {@}. O aniversariante quer ver o palhaço levar torta na cara e o pai dele, já no quinto latão, quer briga",
				"O bolo ainda nem foi cortado e a criançada já subiu em cima de {@}. No canto, o pai do aniversariante encara o palhaço com cara de quem perdeu dinheiro no jogo",
			},
			choices: []choice{
				{label: "🥧 Levar a torta na cara", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} levou a torta, a criançada gritou e o pai deu uma gorjeta"}, effects: []effect{earn(30, 60), gain(1, "gorjeta")}},
					{weight: 30, texts: []string{"A torta tinha um garfo esquecido dentro"}, effects: []effect{injure(5, 10), earn(20, 40)}},
				}},
				{label: "🎈 Fazer bichinho pro pai", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O bichinho virou uma cobra, o pai rolou de rir e virou melhor amigo de {@}"}, effects: []effect{earn(50, 100), gain(1, "latao")}},
					{weight: 50, texts: []string{"O pai achou que era deboche e empurrou {@} pra dentro da piscina de bolinha"}, effects: []effect{injure(5, 15)}},
				}},
				{label: "🤡 Fazer o número do susto", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"A criançada amou e contratou {@} pra mais cinco festas"}, effects: []effect{earn(80, 160)}},
					{weight: 40, texts: []string{"Três crianças choraram, uma mordeu {@} e a mãe não pagou o cachê"}, effects: []effect{injure(5, 10), lose(2, 5)}},
					{weight: 20, texts: []string{"A vizinha chamou a polícia achando que era o palhaço assassino da internet. {@} passou a tarde explicando na delegacia"}, effects: []effect{delay(30 * time.Minute)}},
				}},
			},
		},
		{
			id:    "prof-palhaco-quadrilha",
			title: "O convite da quadrilha",
			jobs:  []string{"palhaço"},
			intros: []string{
				"Uma quadrilha procurou {@} com uma proposta: entrar fantasiado num assalto a banco, porque \"ninguém desconfia de palhaço\"",
				"Três caras de máscara bateram na porta do circo: \"A gente precisa de um palhaço pro serviço de amanhã. O banco abre às dez\". Todo mundo olhou pra {@}",
			},
			choices: []choice{
				{label: "🎪 Aceitar o papel", risk: highRisk, endings: []ending{
					{weight: 35, texts: []string{"O assalto deu certo! A polícia procurou um palhaço e prendeu o Bozo errado"}, effects: []effect{earn(200, 400), stain()}},
					{weight: 45, texts: []string{"{@} tropeçou no próprio sapatão na porta do banco e a quadrilha inteira foi presa"}, effects: []effect{nabbed(), lose(5, 10)}},
					{weight: 20, texts: []string{"O segurança acertou bem no nariz de borracha de {@}"}, effects: []effect{injure(30, 50), nabbed()}},
				}},
				{label: "📞 Dedurar a quadrilha", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"A polícia pegou todo mundo no flagra e {@} ganhou uma recompensa"}, effects: []effect{earn(100, 200)}},
					{weight: 45, texts: []string{"A quadrilha descobriu e mandou um recado: o carro de {@} amanheceu cheio de torta"}, effects: []effect{lose(3, 8)}},
				}},
				{label: "🙅 Recusar fazendo piada", risk: lowRisk, endings: []ending{
					{weight: 80, texts: []string{"A quadrilha riu tanto que contratou {@} pra animar o aniversário do chefe"}, effects: []effect{earn(40, 80)}},
					{weight: 20, texts: []string{"Ninguém riu. Foi o silêncio mais longo da carreira de {@}"}},
				}},
			},
		},
		{
			id:    "prof-padre-confissao",
			title: "A confissão do bandido",
			jobs:  []string{"padre"},
			intros: []string{
				"No confessionário, um homem todo tatuado sussurrou pra {@}: \"Padre, eu enterrei o dinheiro do assalto do banco atrás da igreja. Me perdoa?\"",
				"\"Perdoa, padre, porque eu pequei. E o dinheiro tá enterrado embaixo da sua horta\". Do outro lado da treliça, {@} ouviu a confissão mais cara da vida",
			},
			choices: []choice{
				{label: "🤐 Guardar o segredo", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} deu a penitência de sempre: vinte ave-marias e devolver tudo pro banco"}, effects: []effect{earn(20, 40), plant("prof-padre-gratidao", 6*time.Hour)}},
				}},
				{label: "⛏️ Cavar atrás da igreja", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"Tinha mesmo uma mala enterrada. {@} \"doou\" tudo pra reforma da igreja. E pra uma moto nova"}, effects: []effect{earn(250, 500), stain()}},
					{weight: 35, texts: []string{"Era só a ossada do cachorro do antigo padre, e {@} ficou com a mão cheia de calo"}, effects: []effect{injure(5, 10), gain(1, "caveira")}},
					{weight: 25, texts: []string{"O bandido tava olhando pela janela da sacristia"}, effects: []effect{injure(30, 50)}},
				}},
				{label: "🚔 Avisar a polícia", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"A polícia achou o dinheiro e {@} ganhou uma placa de honra na delegacia"}, effects: []effect{earn(80, 150), enchant("justica")}},
					{weight: 50, texts: []string{"O bispo descobriu que {@} quebrou o sigilo da confissão e mandou pra uma paróquia no meio do nada"}, effects: []effect{enchant("excomunhao"), delay(30 * time.Minute)}},
				}},
			},
		},
		{
			id:    "prof-padre-gratidao",
			title: "O bandido arrependido",
			stage: true,
			intros: []string{
				"O homem tatuado do confessionário voltou procurando {@}. Dessa vez de terno, com uma caixa nas mãos e os olhos cheios de lágrima",
			},
			choices: []choice{
				{label: "🎁 Aceitar o presente", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"Era uma doação gorda pra igreja, de um homem que mudou de vida"}, effects: []effect{earn(150, 300), enchant("protecao")}},
					{weight: 40, texts: []string{"Era o dinheiro do assalto, e a polícia vinha logo atrás dele"}, effects: []effect{lose(5, 10), stain()}},
				}},
				{label: "🙏 Rezar com ele", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} rezou um terço inteiro com ele, que saiu dali em paz e deixou o terço da avó de presente"}, effects: []effect{gain(1, "terco"), enchant("protecao")}},
				}},
				{label: "🚪 Fingir que não conhece", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} se escondeu atrás do altar até ele desistir e ir embora"}},
				}},
			},
		},
		{
			id:     "prof-padre-santa",
			title:  "A santa chorou",
			rarity: rareEvent,
			jobs:   []string{"padre"},
			intros: []string{
				"Na missa das sete, a imagem da santa da paróquia de {@} começou a chorar. Em uma hora a igreja lotou, e já tem repórter do Fantástico na porta",
				"Uma beata gritou no meio do sermão: a santa do altar tá chorando! Os fiéis caíram de joelhos e todo mundo olhou pra {@} esperando uma explicação",
			},
			choices: []choice{
				{label: "📺 Dar entrevista na TV", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} virou celebridade nacional e a igreja encheu de romeiro e de doação"}, effects: []effect{earn(300, 600)}},
					{weight: 40, texts: []string{"O repórter achou um cano vazando atrás da santa. Vergonha em rede nacional"}, effects: []effect{enchant("olhogordo"), lose(3, 8)}},
				}},
				{label: "🔧 Investigar o milagre", risk: lowRisk, endings: []ending{
					{weight: 50, texts: []string{"Era infiltração mesmo. {@} consertou o telhado e a santa parou de chorar"}, effects: []effect{earn(20, 50)}},
					{weight: 50, texts: []string{"Não tinha cano, não tinha nada. A lágrima era de verdade, e {@} ficou arrepiado o dia inteiro"}, effects: []effect{enchant("sorte"), enchant("protecao")}},
				}},
				{label: "💰 Vender água da santa", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"Garrafinha de água benta a vinte reais. Esgotou em duas horas"}, effects: []effect{earn(300, 500), gain(1, "agua-benta")}},
					{weight: 30, texts: []string{"O bispo apareceu no meio da venda e expulsou os vendilhões do templo, a começar por {@}"}, effects: []effect{enchant("excomunhao"), lose(5, 10)}},
					{weight: 20, texts: []string{"A Vigilância Sanitária multou a \"água benta\", que era da torneira"}, effects: []effect{lose(8, 15)}},
				}},
			},
		},
		{
			id:    "prof-matador-contrato",
			title: "O contrato duplo",
			jobs:  []string{"matador de aluguel"},
			intros: []string{
				"Um velho de chapéu deixou uma foto e um envelope gordo pra {@}: \"O serviço é esse aqui\". Na foto, {@} reconheceu o próprio velho, dez anos mais novo",
				"O cliente novo de {@} pagou adiantado e entregou a foto do alvo. O alvo era ele mesmo, sorrindo, num churrasco de 2014",
			},
			choices: []choice{
				{label: "🔪 Fazer o serviço", risk: highRisk, endings: []ending{
					{weight: 30, texts: []string{"O velho só queria um fim digno e deixou tudo pra quem cumprisse o contrato"}, effects: []effect{earn(200, 400), stain()}},
					{weight: 40, texts: []string{"Quando {@} chegou, o velho tinha um contrato igualzinho, com a foto de {@}"}, next: "prof-matador-duelo"},
					{weight: 30, texts: []string{"Era uma armadilha da polícia, e o velho era delegado aposentado"}, effects: []effect{nabbed()}},
				}},
				{label: "💵 Pegar o envelope e sumir", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} sumiu com o dinheiro e nunca mais pisou naquele bairro"}, effects: []effect{earn(150, 300), stain()}},
					{weight: 45, texts: []string{"O velho contratou outro matador pra cobrar o calote"}, effects: []effect{injure(20, 40)}},
				}},
				{label: "☕ Conversar com o velho", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"Ele só queria companhia. Pagou um café e contou histórias da época do cangaço"}, effects: []effect{earn(30, 60), gain(1, "cafe")}},
					{weight: 30, texts: []string{"Ele contou onde guardava as economias: numa meia de lã. E deu metade pra {@}"}, effects: []effect{earn(80, 150)}},
				}},
			},
		},
		{
			id:    "prof-matador-duelo",
			title: "Duelo ao meio-dia",
			stage: true,
			intros: []string{
				"O velho e {@} estão frente a frente na rua deserta, cada um com o contrato do outro no bolso. Um cachorro passa entre os dois. Ninguém pisca",
			},
			choices: []choice{
				{label: "🔫 Sacar primeiro", risk: highRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} foi mais rápido. O velho caiu rindo: \"Finalmente um profissional\". E deixou a herança"}, effects: []effect{earn(250, 500), stain()}},
					{weight: 35, texts: []string{"O velho era mais rápido do que parecia"}, effects: []effect{injure(40, 70)}},
					{weight: 10, texts: []string{"Os dois sacaram ao mesmo tempo"}, effects: []effect{perish()}},
				}},
				{label: "🤝 Propor um acordo", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"Os dois rasgaram os contratos e abriram uma firma de \"cobrança\" juntos"}, effects: []effect{earn(80, 160)}},
					{weight: 40, texts: []string{"O velho fingiu aceitar e deu uma bengalada em {@}"}, effects: []effect{injure(15, 30)}},
				}},
				{label: "🏃 Correr", risk: lowRisk, endings: []ending{
					{weight: 100, favor: func(p Profession) float64 { return p.escapeBonus }, texts: []string{"{@} correu tanto que chegou em outro estado, deixando a reputação pra trás"}, effects: []effect{delay(30 * time.Minute), lose(2, 5)}},
				}},
			},
		},
		{
			id:    "prof-matador-endereco",
			title: "O endereço errado",
			jobs:  []string{"matador de aluguel"},
			intros: []string{
				"O contratante passou o endereço pra {@}, mas quem abriu a porta foi uma senhorinha de avental oferecendo bolo de fubá quentinho",
				"{@} chegou no endereço do serviço, respirou fundo e tocou a campainha. Abriu uma vovó de touca: \"Entra, meu filho, o café tá passando\"",
			},
			choices: []choice{
				{label: "🍰 Aceitar o bolo", risk: midRisk, endings: []ending{
					{weight: 70, texts: []string{"O bolo tava divino. {@} ligou pro contratante e cancelou o serviço ali mesmo"}, effects: []effect{heal(15)}},
					{weight: 30, texts: []string{"Enquanto {@} comia, a senhorinha chamou o neto, que é delegado"}, effects: []effect{nabbed()}},
				}},
				{label: "🔍 Investigar o endereço", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O alvo era o vizinho de muro. {@} fez o serviço com calma, depois do café"}, effects: []effect{earn(150, 300), stain()}},
					{weight: 50, texts: []string{"O alvo tinha se mudado pra Portugal, e o contratante quer o adiantamento de volta"}, effects: []effect{lose(5, 10)}},
				}},
				{label: "🔫 Cobrar o contratante", risk: midRisk, needs: armedUp, endings: []ending{
					{weight: 60, texts: []string{"O contratante pagou o serviço inteiro só pra {@} ir embora da porta dele"}, effects: []effect{earn(150, 250)}},
					{weight: 40, texts: []string{"O contratante tinha mais segurança que ministro"}, effects: []effect{injure(20, 40)}},
				}},
				{label: "📵 Sumir e trocar de chip", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} bloqueou o contratante, trocou de chip e foi assistir novela"}},
				}},
			},
		},
		{
			id:    "prof-hacker-banco",
			title: "Invasão ao banco",
			jobs:  []string{"hacker"},
			intros: []string{
				"Três da manhã, {@} achou uma porta aberta no sistema do maior banco do país. Na tela, piscando em verde: \"Acesso administrativo concedido\"",
				"Depois de 14 latas de energético, {@} digitou a última linha e apertou enter. O painel de controle do banco abriu inteirinho na tela",
			},
			choices: []choice{
				{label: "💸 Transferir uns trocados", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"Ninguém percebeu uns centavos sumindo de cada conta. Somando, deu uma bolada"}, effects: []effect{earn(300, 600), stain()}},
					{weight: 60, texts: []string{"Um alerta vermelho começou a piscar: RASTREAMENTO INICIADO"}, next: "prof-hacker-rastreado"},
				}},
				{label: "📧 Avisar o banco da falha", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"O banco pagou uma recompensa pelo bug e ainda mandou uma caneca"}, effects: []effect{earn(150, 250)}},
					{weight: 40, texts: []string{"O banco acusou {@} de invasão e soltou o jurídico em cima"}, effects: []effect{lose(2, 5)}},
				}},
				{label: "🎮 Zerar o Free Fire com o acesso", risk: midRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} desbloqueou todas as skins do jogo e virou rei da molecada"}, effects: []effect{earn(30, 60)}},
					{weight: 30, texts: []string{"O banco era patrocinador do campeonato. Ban permanente"}, effects: []effect{delay(20 * time.Minute)}},
				}},
			},
		},
		{
			id:    "prof-hacker-rastreado",
			title: "Rastreamento iniciado",
			stage: true,
			intros: []string{
				"A barra de rastreamento tá em 70%. O endereço de {@} começa a aparecer letra por letra na tela da central do banco",
			},
			choices: []choice{
				{label: "🔌 Puxar o cabo da tomada", risk: midRisk, endings: []ending{
					{weight: 60, favor: func(p Profession) float64 { return p.escapeBonus }, texts: []string{"Desligou bem a tempo. {@} passou duas horas olhando pra parede, tremendo"}, effects: []effect{delay(20 * time.Minute)}},
					{weight: 40, texts: []string{"Tarde demais. Bateram na porta às seis da manhã"}, effects: []effect{nabbed()}},
				}},
				{label: "🌊 Jogar o notebook no rio", risk: midRisk, endings: []ending{
					{weight: 80, texts: []string{"O notebook afundou junto com as provas. E com as fotos de família"}, effects: []effect{lose(3, 6)}},
					{weight: 20, texts: []string{"O notebook boiou, e o pescador que achou entregou pra polícia"}, effects: []effect{nabbed()}},
				}},
				{label: "⌨️ Hackear de volta", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"{@} desviou o rastreamento pro computador do gerente do banco. Prenderam o gerente"}, effects: []effect{earn(200, 400), stain()}},
					{weight: 55, texts: []string{"A Polícia Federal hackeou mais rápido"}, effects: []effect{nabbed(), lose(5, 10)}},
				}},
			},
		},
		{
			id:    "prof-hacker-prefeito",
			title: "O arquivo do prefeito",
			jobs:  []string{"hacker"},
			intros: []string{
				"Fuçando num e-mail qualquer, {@} achou uma pasta chamada NÃO ABRIR. Dentro, fotos do prefeito fantasiado de pônei numa festa",
				"O pendrive que {@} comprou na feira do rolo veio com uma surpresa: fotos do prefeito de fantasia de pônei, rebolando num aniversário",
			},
			choices: []choice{
				{label: "💰 Chantagear o prefeito", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"O prefeito pagou rapidinho pra abafar o caso"}, effects: []effect{earn(250, 500), stain()}},
					{weight: 35, texts: []string{"O prefeito mandou a Guarda Municipal inteira atrás de {@}"}, effects: []effect{nabbed()}},
					{weight: 20, texts: []string{"O prefeito não ligou e postou a foto ele mesmo. Virou sensação na cidade e {@} ficou falando sozinho"}},
				}},
				{label: "📢 Vazar na internet", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"A foto viralizou e {@} ganhou seguidor e convite pra podcast"}, effects: []effect{earn(80, 160)}},
					{weight: 45, texts: []string{"Rastrearam o vazamento até o Wi-Fi da vizinha, que entregou {@} na hora"}, effects: []effect{stain(), lose(3, 6)}},
				}},
				{label: "🗑️ Apagar e esquecer", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} apagou tudo, mas nunca mais vai olhar pra um pônei do mesmo jeito"}},
				}},
			},
		},
		{
			id:    "prof-caminhoneiro-rebite",
			title: "O rebite do posto",
			jobs:  []string{"caminhoneiro"},
			intros: []string{
				"Faltam 900 km pra entrega e o prazo vence amanhã cedo. No posto, um colega de estrada ofereceu pra {@} um \"rebite pra aguentar a noite\"",
				"Meia-noite no Posto Graal, olho pesando, e o prazo da carga apertando. O caminhoneiro do lado sacudiu uma cartelinha na frente de {@}: \"Toma um que tu chega\"",
			},
			choices: []choice{
				{label: "💊 Tomar o rebite", risk: highRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} rodou a noite inteira de olho arregalado e entregou a carga antes do prazo"}, effects: []effect{buzz("rebite"), earn(100, 200)}},
					{weight: 30, texts: []string{"O coração disparou no meio da serra e {@} teve que parar no acostamento"}, effects: []effect{buzz("rebite"), injure(15, 30)}},
					{weight: 15, texts: []string{"A PRF parou {@} de olho esbugalhado e achou o resto da cartela no painel"}, effects: []effect{nabbed()}},
				}},
				{label: "😴 Dormir na boleia", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} dormiu quatro horas e chegou atrasado, mas inteiro. A transportadora descontou o atraso"}, effects: []effect{lose(2, 5)}},
					{weight: 30, texts: []string{"Dormiu tanto que acordou com o caminhão sendo guinchado"}, effects: []effect{delay(30 * time.Minute)}},
				}},
				{label: "☕ Café e modão no talo", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"Com 14 cafés e o CD do Zezé no repeat, {@} chegou na hora certinha"}, effects: []effect{earn(60, 120)}},
					{weight: 40, texts: []string{"{@} cochilou e acordou no acostamento com o pneu estourado"}, effects: []effect{injure(10, 20), gain(1, "pneu")}},
				}},
			},
		},
		{
			id:    "prof-caminhoneiro-carga",
			title: "A carga que some",
			jobs:  []string{"caminhoneiro"},
			intros: []string{
				"Num posto escuro da BR, três sujeitos de boné rodearam o caminhão de {@}: \"Tem duas opções, chefe: você deixa a carga sumir e leva sua parte, ou a carga some do mesmo jeito\"",
				"O frentista piscou pra {@} e apontou pra uma caminhonete sem placa: \"Os rapazes ali querem conversar sobre a sua carga de celular\"",
			},
			choices: []choice{
				{label: "🤝 Deixar a carga sumir", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} registrou o \"roubo\" na delegacia e embolsou a parte combinada"}, effects: []effect{earn(150, 300), stain()}},
					{weight: 30, texts: []string{"A seguradora desconfiou do boletim de ocorrência e mandou investigar"}, effects: []effect{stain(), lose(5, 10)}},
					{weight: 20, texts: []string{"A quadrilha foi pega na mesma noite e entregou {@} no primeiro interrogatório"}, effects: []effect{nabbed()}},
				}},
				{label: "🚛 Arrancar com o caminhão", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} arrancou com tudo e os caras ficaram comendo poeira"}, effects: []effect{earn(80, 150)}},
					{weight: 50, texts: []string{"Um deles se pendurou na porta e acertou {@} antes de cair"}, effects: []effect{injure(20, 40)}},
				}},
				{label: "📻 Chamar no rádio PX", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"Vinte caminhoneiros apareceram buzinando e a quadrilha fugiu de medo"}, effects: []effect{earn(30, 60), enchant("protecao")}},
					{weight: 40, texts: []string{"Ninguém respondeu, só um cara pedindo música. A quadrilha levou parte da carga"}, effects: []effect{lose(3, 5)}},
				}},
			},
		},
		{
			id:    "prof-bombeiro-predio",
			title: "Prédio em chamas",
			jobs:  []string{"bombeiro"},
			intros: []string{
				"Um prédio de cinco andares tá pegando fogo e uma senhora grita lá do último andar que o papagaio dela ficou lá dentro. {@} é o primeiro a chegar",
				"Fumaça preta saindo pela janela do quinto andar e uma vovó acenando um pano de prato lá de cima: \"Salva o Louro!\". A viatura de {@} acabou de encostar",
			},
			choices: []choice{
				{label: "🔥 Entrar no prédio", risk: highRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} saiu da fumaça com a senhora num braço e o papagaio no outro. Ela deu o papagaio de presente, porque ele xinga demais"}, effects: []effect{earn(150, 300), gain(1, "papagaio")}},
					{weight: 35, texts: []string{"O teto desabou do lado de {@}, que saiu chamuscado, mas com todo mundo salvo"}, effects: []effect{injure(25, 45), earn(100, 200)}},
					{weight: 10, texts: []string{"{@} entrou na fumaça e não saiu mais"}, effects: []effect{perish()}},
				}},
				{label: "🪜 Subir pela escada", risk: midRisk, endings: []ending{
					{weight: 70, texts: []string{"Resgate de manual. A senhora mandou um bolo pro quartel no dia seguinte"}, effects: []effect{earn(80, 150)}},
					{weight: 30, texts: []string{"A escada balançou no vento e {@} ficou pendurado um tempão"}, effects: []effect{injure(10, 20)}},
				}},
				{label: "💧 Esperar a equipe", risk: lowRisk, endings: []ending{
					{weight: 80, texts: []string{"O fogo apagou e todo mundo saiu bem. Menos o sofá"}, effects: []effect{earn(30, 60)}},
					{weight: 20, texts: []string{"O papagaio saiu voando sozinho e xingou {@} a tarde inteira"}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, outro bombeiro entrou no prédio e levou toda a fama"}},
		},
		{
			id:    "prof-bombeiro-calendario",
			title: "O calendário da corporação",
			jobs:  []string{"bombeiro"},
			intros: []string{
				"Chegou o ensaio do calendário anual dos bombeiros e o fotógrafo escolheu {@} pra estrelar dezembro: sem camisa, segurando a mangueira",
				"\"Você é o mês de dezembro\", anunciou o capitão, jogando um óleo de bebê na mão de {@}. O fotógrafo já tava montando o ventilador",
			},
			choices: []choice{
				{label: "📸 Posar com tudo", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"O calendário esgotou e {@} virou papel de parede de metade da cidade"}, effects: []effect{earn(100, 200)}},
					{weight: 45, texts: []string{"A foto foi parar no grupo da família e a tia comentou com emoji de foguinho"}, effects: []effect{enchant("olhogordo")}},
				}},
				{label: "🎅 Posar de Papai Noel", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"Ficou fofo. A tia da cantina comprou dez"}, effects: []effect{earn(30, 60)}},
				}},
				{label: "🔥 Posar com fogo de verdade", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"A foto ficou épica e virou capa de revista"}, effects: []effect{earn(150, 300)}},
					{weight: 50, texts: []string{"O fogo de verdade levou a sobrancelha de {@} embora"}, effects: []effect{injure(15, 30)}},
				}},
			},
		},
		{
			id:     "prof-pescador-iara",
			title:  "O canto da Iara",
			jobs:   []string{"pescador"},
			places: []string{"!pescar"},
			intros: []string{
				"No meio do rio, uma voz linda começou a cantar chamando {@} pelo nome. A água ficou parada igual espelho",
				"A lua bateu no rio e um canto doce saiu lá do fundo. O barco de {@} começou a andar sozinho na direção da voz",
			},
			choices: []choice{
				{label: "🏊 Nadar até a voz", risk: highRisk, endings: []ending{
					{weight: 35, texts: []string{"Era a Iara, e ela simpatizou com {@}: encheu o barco de peixe nobre e ainda deixou uma pérola"}, effects: []effect{gain(1, "pirarucu", "dourado", "tambaqui"), gain(1, "perola"), enchant("maoboa")}},
					{weight: 55, texts: []string{"A correnteza puxou {@} pro fundo e só uma raiz de árvore salvou"}, effects: []effect{injure(30, 60)}},
					{weight: 10, texts: []string{"{@} nadou até a voz e o rio nunca mais devolveu"}, effects: []effect{perish()}},
				}},
				{label: "🙉 Tampar os ouvidos", risk: lowRisk, endings: []ending{
					{weight: 80, texts: []string{"{@} tampou os ouvidos com cera de vela, pescou em paz e voltou com o samburá cheio"}, effects: []effect{findLoot()}},
					{weight: 20, texts: []string{"A Iara se ofendeu com a desfeita e espantou todos os peixes do rio"}, effects: []effect{enchant("panema")}},
				}},
				{label: "🎶 Cantar de volta", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"A Iara achou graça no dueto e jogou uma pérola dentro do barco"}, effects: []effect{gain(1, "perola")}},
					{weight: 50, texts: []string{"A voz de {@} era tão ruim que a Iara amaldiçoou o rio inteiro"}, effects: []effect{enchant("panema"), delay(15 * time.Minute)}},
				}},
			},
		},
		{
			id:    "prof-pescador-turista",
			title: "O turista que quer tubarão",
			jobs:  []string{"pescador"},
			intros: []string{
				"Um turista de chapéu de safári contratou {@} pra um passeio de barco e exige pescar um tubarão. \"Pago o que for, mas quero foto com tubarão\"",
				"\"Meu sonho é tirar foto com um tubarão\", disse o turista, já de colete salva-vidas e protetor no nariz, pulando dentro do barco de {@}",
			},
			choices: []choice{
				{label: "🦈 Ir pro mar aberto", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"Apareceu um tubarão de verdade! A foto saiu linda e o turista pagou em dobro"}, effects: []effect{earn(150, 250)}},
					{weight: 40, texts: []string{"O mar virou, o turista vomitou no barco inteiro e foi embora sem pagar"}, effects: []effect{lose(2, 5)}},
					{weight: 20, texts: []string{"O barco virou e {@} teve que nadar até a praia rebocando o turista"}, effects: []effect{injure(20, 40)}},
				}},
				{label: "🐟 Comprar um cação na feira", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"O turista tirou foto com um cação da feira e postou \"meu primeiro tubarão\""}, effects: []effect{earn(80, 150)}},
					{weight: 40, texts: []string{"O turista viu a etiqueta do mercado presa no rabo do peixe"}, effects: []effect{lose(3, 6)}},
				}},
				{label: "🍺 Levar pro bar", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} contou tanta história de pescador que o turista pagou a conta e ainda deu gorjeta"}, effects: []effect{earn(30, 70), gain(1, "gorjeta")}},
				}},
			},
		},
		{
			id:     "prof-garimpeiro-eldorado",
			title:  "O veio do Eldorado",
			rarity: rareEvent,
			jobs:   []string{"garimpeiro"},
			places: []string{"!minerar"},
			intros: []string{
				"Cavando no fundo da mina, a picareta de {@} bateu num veio amarelo que brilha de ponta a ponta. Lá no fundo do túnel, uma luz esquisita pisca no escuro",
				"Um estalo, um brilho, e a parede da mina se abriu num veio de ouro grosso igual braço. Atrás dele, {@} jura que viu uma chama andando sozinha",
			},
			choices: []choice{
				{label: "⛏️ Cavar o veio inteiro", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} encheu o saco de pepita até não aguentar carregar"}, effects: []effect{gain(3, "ouro"), earn(200, 400)}},
					{weight: 40, texts: []string{"A parede cedeu em cima de {@}, que saiu da mina com uma pepita só e muita dor"}, effects: []effect{injure(30, 60), gain(1, "ouro")}},
					{weight: 10, texts: []string{"A Mãe do Ouro apareceu e levou {@} pro fundo da terra"}, effects: []effect{perish()}},
				}},
				{label: "🕯️ Pedir licença à Mãe do Ouro", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"A luz se apagou devagar, em paz. {@} saiu com umas pepitas e uma sensação de proteção"}, effects: []effect{gain(2, "ouro"), enchant("protecao")}},
					{weight: 30, texts: []string{"A luz achou a oferenda pouca e transformou as pepitas em ouro de tolo"}, effects: []effect{gain(3, "ouro-de-tolo")}},
				}},
				{label: "🗺️ Vender o mapa do veio", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} desenhou o mapa e vendeu pra um gringo por uma fortuna"}, effects: []effect{earn(150, 300)}},
					{weight: 40, texts: []string{"O gringo pagou com um cheque que voltou duas vezes"}, effects: []effect{delay(20 * time.Minute)}},
				}},
			},
		},
		{
			id:    "prof-garimpeiro-comprador",
			title: "O comprador de ouro",
			jobs:  []string{"garimpeiro"},
			intros: []string{
				"Um homem de óculos escuros e cordão grosso apareceu no garimpo de {@}: compra toda a produção, sem nota, em dinheiro vivo, pela metade da cotação",
				"Um avião pousou na pista de terra do garimpo e desceu um sujeito com uma maleta. \"Compro tudo que você tiver\", disse pra {@}, \"e não pergunto de onde veio\"",
			},
			choices: []choice{
				{label: "🤝 Vender sem nota", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"O dinheiro veio em notas novinhas, cheirando a tinta"}, effects: []effect{earn(100, 200), stain()}},
					{weight: 25, texts: []string{"Metade das notas era dinheiro de brinquedo"}, effects: []effect{lose(3, 8), gain(1, "dinheiro-de-brinquedo")}},
					{weight: 15, texts: []string{"Era um agente da Polícia Federal disfarçado"}, effects: []effect{nabbed()}},
				}},
				{label: "🏦 Vender na cooperativa", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"Preço justo, nota fiscal e um cafezinho de cortesia"}, effects: []effect{earn(40, 80)}},
				}},
				{label: "💎 Empurrar ouro de tolo", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} vendeu ouro de tolo como pepita rara e o comprador saiu feliz"}, effects: []effect{earn(100, 200), stain()}},
					{weight: 50, texts: []string{"O comprador entendia de pedra e deu uma coronhada em {@}"}, effects: []effect{injure(15, 30)}},
				}},
			},
		},
		{
			id:    "prof-cartomante-morte",
			title: "A carta da Morte",
			jobs:  []string{"cartomante"},
			intros: []string{
				"Embaralhando pra uma cliente, {@} deixou cair uma carta no chão. Era a Morte, virada pra cima, apontando direto pra {@}",
				"A cliente cortou o baralho três vezes e todas as vezes saiu a mesma carta: a Morte. Só que virada pro lado de {@}",
			},
			choices: []choice{
				{label: "🔮 Ler o próprio destino", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"As cartas disseram que o perigo vem do norte. {@} anotou num papel e guardou no bolso"}, effects: []effect{plant("prof-cartomante-profecia", 4*time.Hour)}},
					{weight: 40, texts: []string{"As cartas mostraram uma fortuna chegando, e {@} cobrou a própria consulta de si mesmo"}, effects: []effect{enchant("sorte")}},
				}},
				{label: "🧂 Limpeza com sal grosso", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} jogou sal grosso na sala inteira e o clima mudou na hora"}, effects: []effect{enchant("protecao")}},
					{weight: 30, texts: []string{"Foi tanto sal que a cliente escorregou e quebrou o salto. A consulta saiu de graça"}, effects: []effect{lose(2, 5)}},
				}},
				{label: "🙈 Dizer que a carta é da cliente", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"A cliente apavorou e pagou o triplo por um \"trabalho de proteção\""}, effects: []effect{earn(100, 200)}},
					{weight: 50, texts: []string{"A cliente saiu correndo sem pagar e falou mal de {@} no bairro inteiro"}, effects: []effect{enchant("olhogordo")}},
				}},
			},
		},
		{
			id:    "prof-cartomante-profecia",
			title: "A profecia se cumpre",
			stage: true,
			intros: []string{
				"O papel no bolso de {@} dizia: \"o perigo vem do norte\". Agora mesmo, um vento gelado começou a soprar do norte e um gato preto atravessou a rua",
			},
			choices: []choice{
				{label: "🏠 Correr pra casa", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} se trancou em casa e nada aconteceu. Ou foi justamente por isso"}},
					{weight: 30, texts: []string{"Em casa, um pedaço do forro despencou bem em cima de {@}"}, effects: []effect{injure(5, 15)}},
				}},
				{label: "🐈‍⬛ Seguir o gato preto", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O gato levou {@} até uma carteira caída na calçada"}, effects: []effect{gain(1, "carteira-perdida"), earn(30, 60)}},
					{weight: 50, texts: []string{"O gato deu meia-volta e meteu as unhas em {@}"}, effects: []effect{injure(5, 15)}},
				}},
				{label: "🃏 Desafiar o destino", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"{@} encarou o vento do norte e ele virou brisa. A sorte trocou de lado"}, effects: []effect{enchant("sorte"), earn(80, 150)}},
					{weight: 45, texts: []string{"O destino aceitou o desafio, e jogou pesado"}, effects: []effect{injure(20, 40), enchant("azar")}},
					{weight: 10, texts: []string{"Com o destino não se brinca"}, effects: []effect{perish()}},
				}},
			},
		},
		{
			id:    "prof-cartomante-traicao",
			title: "O marido trai?",
			jobs:  []string{"cartomante"},
			intros: []string{
				"Uma cliente rica entrou na tenda de {@} jogando dinheiro na mesa: \"Quero saber se meu marido me trai. E quero a verdade\"",
				"De óculos escuros e voz tremendo, a dona da maior loja do bairro sentou na frente de {@}: \"Joga as cartas. Ele tá me traindo ou não?\"",
			},
			choices: []choice{
				{label: "💔 Dizer que trai", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Acertou em cheio! A cliente pagou em dobro e foi direto pro advogado"}, effects: []effect{earn(100, 200)}},
					{weight: 50, texts: []string{"Errou. O marido era fiel e veio tirar satisfação com {@}"}, effects: []effect{injure(5, 15), lose(2, 5)}},
				}},
				{label: "💕 Dizer que é fiel", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"A cliente saiu aliviada e deixou uma gorjeta"}, effects: []effect{earn(40, 80)}},
					{weight: 40, texts: []string{"Uma semana depois ela pegou o marido no flagra e voltou pra cobrar a consulta"}, effects: []effect{lose(2, 5)}},
				}},
				{label: "🔮 Oferecer uma amarração", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"A cliente pagou o pacote completo de sete dias"}, effects: []effect{earn(150, 300), stain()}},
					{weight: 50, texts: []string{"O marido era delegado e veio pessoalmente buscar {@}"}, effects: []effect{nabbed()}},
				}},
			},
		},
		{
			id:    "prof-noel-pedido",
			title: "O pedido impossível",
			jobs:  []string{"papai noel de shopping"},
			intros: []string{
				"Um menininho sentou no colo de {@} e cochichou: \"Papai Noel, eu só quero que meu pai volte pra casa no Natal\". A fila inteira ficou em silêncio",
				"A menina não quis brinquedo. Abraçou {@} e pediu baixinho: \"Traz meu pai de volta, Papai Noel?\". A mãe começou a chorar atrás da fila",
			},
			choices: []choice{
				{label: "🎅 Prometer que ele volta", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"No dia seguinte o pai apareceu no shopping procurando a criança. Ninguém sabe como, mas {@} chorou no vestiário"}, effects: []effect{enchant("fartura"), earn(50, 100)}},
					{weight: 50, texts: []string{"O pai não voltou e a mãe foi reclamar com a gerência"}, effects: []effect{lose(3, 6)}},
				}},
				{label: "🤗 Dar um abraço de Noel", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} deu o abraço mais demorado da história do shopping"}, effects: []effect{earn(20, 40)}},
				}},
				{label: "🧸 Dar o ursinho da mochila", risk: lowRisk, uses: "ursinho", endings: []ending{
					{weight: 100, texts: []string{"A criança abraçou o ursinho e sorriu pela primeira vez no dia"}, effects: []effect{enchant("protecao"), enchant("sorte")}},
				}},
				{label: "📞 Ligar pro pai", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"O pai atendeu, chorou e prometeu ir pro Natal. E foi"}, effects: []effect{enchant("sorte"), earn(80, 150)}},
					{weight: 60, texts: []string{"Quem atendeu foi a nova namorada do pai, e o barraco foi parar no Instagram do shopping"}, effects: []effect{enchant("olhogordo"), lose(3, 6)}},
				}},
			},
		},
		{
			id:     "prof-noel-verdadeiro",
			title:  "O Papai Noel de verdade",
			rarity: rareEvent,
			jobs:   []string{"papai noel de shopping"},
			intros: []string{
				"Um velhinho de barba branca de verdade, roupa vermelha surrada e cheiro de rena sentou no trono do lado de {@}: \"Rapaz, eu vou precisar de um substituto essa noite\"",
				"No fim do expediente, um senhor de barba enorme chamou {@} no estacionamento. Atrás dele, nove renas mastigavam a grama do canteiro",
			},
			choices: []choice{
				{label: "🛷 Ser o substituto", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"{@} entregou presente no mundo inteiro num trenó voador e voltou com o saco cheio de sobra"}, effects: []effect{earn(400, 800), gain(1, "caixa-misteriosa-dourada")}},
					{weight: 40, texts: []string{"{@} caiu do trenó dentro de uma chaminé e voltou coberto de carvão"}, effects: []effect{injure(30, 55), gain(1, "carvao")}},
					{weight: 15, texts: []string{"As renas não obedeceram e {@} foi parar no Polo Norte"}, effects: []effect{delay(45 * time.Minute), injure(10, 20)}},
				}},
				{label: "🍪 Oferecer leite e biscoito", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"O velhinho comeu, agradeceu e deixou um presente embaixo do trono"}, effects: []effect{gain(1, "caixa-misteriosa"), enchant("fartura")}},
				}},
				{label: "📸 Pedir uma selfie", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"A selfie viralizou e o shopping triplicou o cachê de {@}"}, effects: []effect{earn(200, 400)}},
					{weight: 50, texts: []string{"Na foto só aparece {@} sozinho no trono. Ninguém mais viu o velhinho"}, effects: []effect{enchant("sorte")}},
				}},
			},
		},
		{
			id:    "prof-streamer-pimenta",
			title: "O desafio da pimenta",
			jobs:  []string{"streamer"},
			intros: []string{
				"No meio da live, alguém mandou um Pix de mil reais: \"Come uma Carolina Reaper inteira ao vivo\". O chat de {@} enlouqueceu e começou a spammar COME COME COME",
				"Doação de 1000 na tela, com a mensagem: \"Pimenta mais forte do mundo, inteira, sem leite\". Doze mil pessoas esperando a reação de {@}",
			},
			choices: []choice{
				{label: "🌶️ Comer a pimenta", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} comeu, chorou, suou e bateu o recorde de audiência do canal"}, effects: []effect{earn(150, 300), injure(5, 15)}},
					{weight: 40, texts: []string{"{@} passou mal ao vivo e a live terminou com a câmera filmando o teto"}, effects: []effect{injure(20, 40), earn(50, 100)}},
					{weight: 10, texts: []string{"A ambulância apareceu na live, e o clipe rodou o mundo"}, effects: []effect{injure(40, 60)}},
				}},
				{label: "🥛 Comer com leite do lado", risk: midRisk, endings: []ending{
					{weight: 70, texts: []string{"Deu certo, mas o chat passou a semana chamando {@} de fraco"}, effects: []effect{earn(60, 120)}},
					{weight: 30, texts: []string{"O leite tava vencido"}, effects: []effect{injure(10, 20)}},
				}},
				{label: "🙅 Devolver o Pix", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"O chat vaiou, mas a mãe de {@} mandou um áudio orgulhosa"}, effects: []effect{earn(10, 30)}},
				}},
			},
		},
		{
			id:    "prof-streamer-raid",
			title: "A raid surpresa",
			jobs:  []string{"streamer"},
			intros: []string{
				"Um streamer gigante mandou uma raid de 40 mil pessoas pra live de {@}, que naquele momento tava comendo miojo de pijama",
				"O contador de viewers pulou de 3 pra 40 mil em um segundo. {@} tava de pijama, com o miojo na boca e o cabelo em pé",
			},
			choices: []choice{
				{label: "🎤 Fazer um show improvisado", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} cantou sertanejo com o garfo de microfone e virou trend"}, effects: []effect{earn(150, 300)}},
					{weight: 45, texts: []string{"Alguém achou um tweet antigo de {@} e começou o cancelamento"}, effects: []effect{enchant("olhogordo"), lose(3, 8)}},
				}},
				{label: "📴 Desligar a câmera", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"A raid foi embora e ficou só o bot de sempre"}},
				}},
				{label: "🍜 Continuar comendo o miojo", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"O chat achou autêntico e assinou em massa"}, effects: []effect{earn(80, 150)}},
					{weight: 40, texts: []string{"Alguém clipou o barulho de {@} sugando o miojo e virou meme eterno"}, effects: []effect{enchant("olhogordo")}},
				}},
			},
		},
		{
			id:    "prof-programador-deploy",
			title: "Deploy na sexta",
			jobs:  []string{"programador"},
			intros: []string{
				"Sexta-feira, 17h58. O chefe pediu pra {@} subir \"só uma alteraçãozinha\" em produção antes do fim de semana",
				"\"É rapidinho, só muda a cor do botão e sobe\", disse o gerente, já de mochila nas costas, às seis da tarde de sexta. {@} encarou a tela em silêncio",
			},
			choices: []choice{
				{label: "🚀 Subir direto em produção", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"Funcionou de primeira. {@} saiu cedo e foi pro bar"}, effects: []effect{earn(80, 150)}},
					{weight: 45, texts: []string{"Caiu tudo. {@} passou o fim de semana no escritório comendo pizza fria"}, effects: []effect{delay(40 * time.Minute), gain(1, "pizza")}},
					{weight: 15, texts: []string{"O deploy apagou o banco de dados de produção. Inteiro"}, effects: []effect{lose(10, 20)}},
				}},
				{label: "🧪 Rodar os testes antes", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"Os testes passaram, o deploy foi tranquilo e ninguém lembrou de agradecer"}, effects: []effect{earn(30, 60)}},
				}},
				{label: "🏃 Fingir que a internet caiu", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"O chefe acreditou e {@} ganhou um fim de semana em paz"}, effects: []effect{earn(20, 40)}},
					{weight: 40, texts: []string{"O chefe viu {@} online jogando no Discord"}, effects: []effect{lose(3, 6), delay(20 * time.Minute)}},
				}},
			},
		},
		{
			id:     "prof-programador-gringa",
			title:  "A proposta da gringa",
			rarity: rareEvent,
			jobs:   []string{"programador"},
			intros: []string{
				"Chegou no LinkedIn de {@} uma proposta de uma empresa gringa: salário em dólar, trabalho remoto e entrevista técnica daqui a uma hora",
				"\"Hi! We loved your GitHub\", dizia o e-mail. Uma empresa gringa quer {@} no time, com salário em dólar, e a entrevista é hoje",
			},
			choices: []choice{
				{label: "🧠 Encarar a entrevista", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} passou! O primeiro salário em dólar caiu na conta"}, effects: []effect{earn(400, 800)}},
					{weight: 50, texts: []string{"Travou no \"tell me about yourself\" e respondeu em português"}, effects: []effect{delay(15 * time.Minute)}},
				}},
				{label: "🤖 Colar com a IA escondido", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"Ninguém percebeu e {@} foi contratado"}, effects: []effect{earn(400, 800)}},
					{weight: 40, texts: []string{"O entrevistador viu o reflexo da tela nos óculos de {@}"}, effects: []effect{enchant("olhogordo")}},
					{weight: 15, texts: []string{"A empresa era golpe e pediu os dados do cartão de {@} \"pra folha de pagamento\""}, effects: []effect{lose(8, 15)}},
				}},
				{label: "😎 Pedir o dobro do salário", risk: highRisk, endings: []ending{
					{weight: 35, texts: []string{"A gringa topou na hora e {@} ficou rico do dia pra noite"}, effects: []effect{earn(600, 1200)}},
					{weight: 65, texts: []string{"A gringa achou caro e contratou outra pessoa pela metade"}},
				}},
			},
		},
		{
			id:    "prof-motoboy-pacote",
			title: "A entrega suspeita",
			jobs:  []string{"motoboy"},
			intros: []string{
				"O pacote da entrega de {@} tá vazando um pó branco e o endereço é um galpão abandonado. O cliente mandou mensagem: \"Chega rápido e não abre\"",
				"Entrega marcada pra um galpão sem número, pagamento em dinheiro e um pacote que faz um barulhinho de pó quando balança. {@} ficou olhando pro baú da moto",
			},
			choices: []choice{
				{label: "📦 Entregar sem perguntar", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"O cliente deu uma gorjeta gorda e disse \"nunca te vi\""}, effects: []effect{earn(100, 200), stain()}},
					{weight: 30, texts: []string{"Tinha uma viatura esperando dentro do galpão"}, effects: []effect{nabbed()}},
					{weight: 15, texts: []string{"Um rival do cliente seguiu {@} e tomou o pacote na base da porrada"}, effects: []effect{injure(15, 30)}},
				}},
				{label: "👃 Dar uma cheirada", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"Era pó mesmo. {@} entregou o resto ligado no 220"}, effects: []effect{buzz("po"), earn(80, 150)}},
					{weight: 30, texts: []string{"{@} cheirou e começou a ouvir a moto conversando"}, effects: []effect{buzz("po"), injure(10, 20)}},
					{weight: 20, texts: []string{"Era farinha de trigo, e o \"cliente\" era a polícia"}, effects: []effect{nabbed()}},
				}},
				{label: "🚔 Levar pra delegacia", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"A polícia agradeceu e o delegado pagou um pastel"}, effects: []effect{earn(30, 60), gain(1, "pastel")}},
					{weight: 40, texts: []string{"O pacote era farinha pro bolo de uma vovó, e ela ficou sem bolo"}},
				}},
			},
		},
		{
			id:    "prof-motoboy-marginal",
			title: "A corrida na marginal",
			jobs:  []string{"motoboy"},
			intros: []string{
				"Faltam 8 minutos pro pedido de {@} atrasar e virar cupom de desconto. A marginal tá parada e o único caminho é o corredor entre os ônibus",
				"O app mostrou o aviso vermelho: \"cliente aguardando há 52 minutos\". Na frente de {@}, um muro de ônibus e caminhão até o horizonte",
			},
			choices: []choice{
				{label: "🏍️ Voar no corredor", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} chegou com um minuto de sobra, ganhou cinco estrelas e já pegou outra entrega"}, effects: []effect{earn(60, 120), refresh()}},
					{weight: 40, texts: []string{"Um ônibus fechou e {@} beijou o asfalto"}, effects: []effect{injure(20, 45)}},
					{weight: 10, texts: []string{"A moto passou embaixo de um caminhão"}, effects: []effect{perish()}},
				}},
				{label: "🛣️ Cortar pela calçada", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"Os pedestres xingaram, mas a comida chegou quentinha"}, effects: []effect{earn(40, 80)}},
					{weight: 40, texts: []string{"O guarda de trânsito tava justamente na calçada"}, effects: []effect{gain(1, "multa"), lose(3, 6)}},
				}},
				{label: "🍔 Comer o lanche e cancelar", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O lanche tava ótimo e o app nem percebeu"}, effects: []effect{heal(10)}},
					{weight: 30, texts: []string{"O app percebeu e bloqueou {@} por umas horas"}, effects: []effect{delay(30 * time.Minute)}},
				}},
			},
		},
		{
			id:    "prof-professor-pix",
			title: "O Pix da recuperação",
			jobs:  []string{"professor"},
			intros: []string{
				"O pior aluno da turma chamou {@} no canto e mostrou o celular: \"Professor, tem um Pix aqui com seu nome se eu passar de ano\"",
				"No fim da aula, o aluno que nunca entregou um trabalho deixou um bilhete na mesa de {@}: \"Te faço um Pix se a minha média virar 6\"",
			},
			choices: []choice{
				{label: "💸 Aceitar o Pix", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"O Pix caiu e o aluno passou. Ninguém nunca soube"}, effects: []effect{earn(150, 300), stain()}},
					{weight: 35, texts: []string{"O aluno printou a conversa e mandou pra diretora"}, effects: []effect{lose(8, 15), delay(30 * time.Minute)}},
					{weight: 15, texts: []string{"O pai do aluno era delegado e tava gravando tudo"}, effects: []effect{nabbed()}},
				}},
				{label: "📚 Dar aula de reforço", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O aluno estudou e passou por mérito próprio. {@} chorou na formatura"}, effects: []effect{earn(30, 60), plant("prof-professor-ex-aluno", 12*time.Hour)}},
					{weight: 30, texts: []string{"O aluno dormiu em todas as aulas de reforço"}},
				}},
				{label: "📢 Denunciar pra direção", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"A diretora deu os parabéns e uma caixa de bombom"}, effects: []effect{gain(1, "caixa-de-bombom")}},
					{weight: 40, texts: []string{"A mãe do aluno fez um barraco na porta da escola e jogou o chinelo em {@}"}, effects: []effect{injure(5, 10)}},
				}},
			},
		},
		{
			id:    "prof-professor-ex-aluno",
			title: "O ex-aluno voltou",
			stage: true,
			intros: []string{
				"Um carrão parou na frente da escola e desceu um homem de terno. Era aquele aluno do reforço, agora empresário, procurando {@}",
			},
			choices: []choice{
				{label: "🤝 Aceitar o agradecimento", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"Ele pagou um almoço caro e deu um presente pra \"quem acreditou nele\""}, effects: []effect{earn(150, 300)}},
				}},
				{label: "🏫 Pedir doação pra escola", risk: midRisk, endings: []ending{
					{weight: 70, texts: []string{"Ele reformou a escola inteira e botou o nome de {@} na biblioteca"}, effects: []effect{earn(100, 200), enchant("prosperidade")}},
					{weight: 30, texts: []string{"Ele prometeu, tirou foto pro Instagram e nunca mais apareceu"}},
				}},
				{label: "😏 Lembrar que ele colava", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Ele riu e confessou que ainda cola na declaração do imposto de renda"}, effects: []effect{earn(50, 100)}},
					{weight: 50, texts: []string{"Ele não achou graça, entrou no carrão e foi embora"}},
				}},
			},
		},
		{
			id:    "prof-professor-feira",
			title: "A feira de ciências",
			jobs:  []string{"professor"},
			intros: []string{
				"Na feira de ciências, um grupo de alunos de {@} montou um vulcão de bicarbonato \"turbinado\" com um líquido que ninguém sabe de onde veio. Já tá borbulhando",
				"\"Professor, a gente melhorou a fórmula\", disse o aluno, orgulhoso. O vulcão do grupo tá soltando uma fumaça verde e o chão do ginásio tá tremendo",
			},
			choices: []choice{
				{label: "🧯 Desarmar o vulcão", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} jogou areia em cima e salvou a feira"}, effects: []effect{earn(40, 80)}},
					{weight: 40, texts: []string{"O vulcão explodiu bem na cara de {@}"}, effects: []effect{injure(15, 30)}},
				}},
				{label: "📸 Deixar explodir pra foto", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"A explosão foi linda, o grupo ganhou o primeiro lugar e {@} levou o prêmio de orientador"}, effects: []effect{earn(100, 200)}},
					{weight: 45, texts: []string{"A explosão derreteu o teto do ginásio e a conta veio pra {@}"}, effects: []effect{lose(8, 15)}},
					{weight: 15, texts: []string{"O estouro jogou {@} do outro lado do ginásio"}, effects: []effect{injure(30, 50)}},
				}},
				{label: "🏃 Evacuar o ginásio", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"Todo mundo saiu, o vulcão fez um pfff triste e a feira continuou"}, effects: []effect{earn(10, 20)}},
					{weight: 30, texts: []string{"Na correria, alguém pisou no pé de {@}"}, effects: []effect{injure(5, 10)}},
				}},
			},
		},
	},
}
