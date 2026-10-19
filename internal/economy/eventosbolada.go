package economy

var windfallEvents = eventBook{
	name:   "💰 Dinheiro chama ladrão",
	places: []string{windfallPlace},
	scenarios: []scenario{
		{
			id:    "bolada-moto",
			title: "Perdeu, playboy",
			intros: []string{
				"A notícia de que {@} encheu o bolso correu rápido. Na esquina, dois caras numa moto sem placa encostaram: 'passa tudo aí, ricaço'",
				"{@} saiu todo pimpão depois de ganhar uma bolada e nem viu a moto chegando por trás. O garupa já desceu com a mão na cintura",
			},
			choices: []choice{
				{label: "💸 Entregar a carteira", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} entregou a carteira sem discutir. Os caras levaram um pedaço da bolada e ainda agradeceram"}, effects: []effect{lose(5, 10)}},
					{weight: 30, texts: []string{"{@} entregou a carteira velha, a de enganar ladrão. Os caras foram embora felizes com trinta conto e um cartão do SUS"}},
				}},
				{label: "🏃 Sair correndo", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} correu igual nunca correu na vida e se enfiou numa padaria. A moto passou reto"}},
					{weight: 30, texts: []string{"{@} tropeçou no meio-fio, ralou tudo e os caras pegaram o dinheiro caído no chão"}, effects: []effect{lose(8, 15), injure(10, 20)}},
					{weight: 20, texts: []string{"Na correria, {@} esbarrou num vendedor ambulante e derrubou a banca. Pagou o prejuízo dele, mas salvou a bolada"}, effects: []effect{lose(2, 5)}},
				}},
				{label: "👊 Reagir na mão", risk: highRisk, endings: []ending{
					{weight: 35, texts: []string{"{@} acertou um soco no garupa, que caiu da moto e largou a pochete de outros assaltos. Virou o dia"}, effects: []effect{earn(100, 250)}},
					{weight: 45, texts: []string{"Os dois deram uma surra em {@} e levaram o dinheiro de troco"}, effects: []effect{lose(10, 20), injure(25, 45)}},
					{weight: 15, texts: []string{"{@} reagiu, levou umas pancadas, mas o povo da rua apareceu e os caras fugiram de mão vazia"}, effects: []effect{injure(10, 20)}},
					{weight: 5, texts: []string{"O garupa não tava blefando. {@} não vai gastar essa bolada"}, effects: []effect{perish()}},
				}},
				{label: "🔫 Sacar a arma primeiro", risk: midRisk, needs: armedUp, endings: []ending{
					{weight: 60, texts: []string{"Quando viu a arma de {@}, o piloto acelerou tanto que o garupa ficou pra trás pedindo desculpa"}},
					{weight: 40, texts: []string{"Os caras fugiram, mas uma vizinha filmou {@} de arma na mão e mandou pra polícia"}, effects: []effect{stain()}},
				}},
				{label: "🚔 Dar voz de prisão", risk: midRisk, needs: lawman, endings: []ending{
					{weight: 60, texts: []string{"{@} mostrou o distintivo, algemou os dois no poste e ainda ganhou a recompensa que tinha na cabeça deles"}, effects: []effect{earn(80, 200)}},
					{weight: 40, texts: []string{"O piloto arrancou e arrastou {@} uns metros pela calçada antes de fugir"}, effects: []effect{injure(15, 30)}},
				}},
			},
			idle: ending{texts: []string{"{@} ficou paralisado de medo, e os caras, achando que era doido, foram procurar outro"}},
		},
		{
			id:    "bolada-central",
			title: "Ligação da central",
			intros: []string{
				"Nem deu tempo de comemorar: o celular de {@} tocou. 'Aqui é da central de segurança do banco, detectamos uma movimentação alta na sua conta'",
				"Chegou um SMS no celular de {@}: 'Sua conta foi bloqueada por movimentação suspeita. Clique no link pra liberar'. Logo depois da bolada, que coincidência",
			},
			choices: []choice{
				{label: "📲 Passar o código que chegou", risk: highRisk, endings: []ending{
					{weight: 70, texts: []string{"Não era o banco. Em dois minutos a conta de {@} tava mais leve e o número já não existia"}, effects: []effect{lose(15, 25)}},
					{weight: 30, texts: []string{"O golpista era tão ruim que errou o Pix e mandou dinheiro pra {@} em vez de tirar"}, effects: []effect{earn(50, 150)}},
				}},
				{label: "📵 Desligar na cara", risk: lowRisk, endings: []ending{
					{weight: 80, texts: []string{"{@} desligou e bloqueou o número. Banco nenhum liga desse jeito"}},
					{weight: 20, texts: []string{"{@} desligou, mas ficou tão nervoso que passou o resto do dia vendo o extrato"}, effects: []effect{injure(2, 5)}},
				}},
				{label: "🤡 Enrolar o golpista", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} fingiu ser a vovó Neusa por quarenta minutos. O golpista desistiu e mandou um Pix de pena"}, effects: []effect{earn(20, 80)}},
					{weight: 50, texts: []string{"{@} se empolgou no teatro e acabou digitando a senha de verdade sem querer"}, effects: []effect{lose(5, 12)}},
				}},
				{label: "🕵️ Rastrear a ligação", risk: midRisk, needs: lawman, endings: []ending{
					{weight: 70, texts: []string{"{@} rastreou o número até uma central de golpes num quartinho e levou um celular apreendido pra casa"}, effects: []effect{gain(1, "celular")}},
					{weight: 30, texts: []string{"O número era de um presídio. {@} preencheu três formulários e não deu em nada"}},
				}},
			},
			idle: ending{texts: []string{"O golpista cansou de esperar e foi ligar pra outro ricaço"}},
		},
		{
			id:    "bolada-primo",
			title: "O primo de terceiro grau",
			intros: []string{
				"Depois da bolada, apareceu na porta de {@} um primo de terceiro grau que ninguém via desde o batizado: 'lembra de mim? Tô com um investimento imperdível'",
				"O grupo da família, que vivia morto, acordou de repente. Um tio de {@} chamou no privado: 'soube da tua sorte, tenho um negócio pra te propor'",
			},
			choices: []choice{
				{label: "🤝 Investir no negócio", risk: highRisk, endings: []ending{
					{weight: 25, texts: []string{"Contra todas as expectativas, a barraca de açaí do primo bombou e ele devolveu com juros"}, effects: []effect{earn(150, 300)}},
					{weight: 55, texts: []string{"O primo sumiu com o dinheiro e mudou o nome no Instagram. {@} só soube depois, no almoço de Natal"}, effects: []effect{lose(10, 20)}},
					{weight: 20, texts: []string{"O negócio era vender pirâmide. {@} perdeu um pouco e ganhou um kit de produtos que ninguém quer"}, effects: []effect{lose(3, 8), gain(1, "boleto")}},
				}},
				{label: "🙅 Dizer que gastou tudo", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} jurou que tava duro, e o primo saiu levando só uma coxinha da geladeira"}},
					{weight: 40, texts: []string{"O primo não acreditou e espalhou pra família inteira que {@} ficou metido depois de rico"}, effects: []effect{enchant("olhogordo")}},
				}},
				{label: "🍗 Chamar pra um churrasco", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O churrasco juntou a família toda, ninguém pediu dinheiro e {@} saiu de alma lavada"}, effects: []effect{heal(15), enchant("protecao")}},
					{weight: 50, texts: []string{"Apareceram vinte parentes, todos com um 'negocinho'. A conta da carne saiu do bolso de {@}"}, effects: []effect{lose(3, 8)}},
				}},
			},
			idle: ending{texts: []string{"{@} deixou o primo no vácuo até ele desistir e mandar um 'tá bom, metido'"}},
		},
		{
			id:    "bolada-arrastao",
			title: "Arrastão na comemoração",
			intros: []string{
				"{@} resolveu comemorar a bolada num barzinho cheio. No meio do pagode, um grito: 'é arrastão!'",
				"Na festa que {@} pagou pra todo mundo depois da bolada, as luzes apagaram e começou um empurra-empurra suspeito",
			},
			choices: []choice{
				{label: "🫣 Esconder o dinheiro na meia", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"Ninguém procura dinheiro numa meia suada. {@} saiu do arrastão com tudo"}},
					{weight: 30, texts: []string{"O ladrão era do tipo caprichoso e revistou até as meias de {@}"}, effects: []effect{lose(5, 10)}},
				}},
				{label: "🪑 Se jogar debaixo da mesa", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Debaixo da mesa, {@} achou a carteira que algum ladrão deixou cair na pressa"}, effects: []effect{earn(40, 120)}},
					{weight: 50, texts: []string{"Pisaram na mão de {@} umas oito vezes antes da polícia chegar"}, effects: []effect{injure(10, 25)}},
				}},
				{label: "🦸 Bancar o herói", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"{@} derrubou um dos ladrões com uma cadeira de plástico e o bar inteiro pagou a conta dele"}, effects: []effect{earn(80, 200), enchant("protecao")}},
					{weight: 50, texts: []string{"A cadeira de plástico quebrou na primeira pancada. Os ladrões não acharam graça"}, effects: []effect{injure(25, 45), lose(5, 15)}},
					{weight: 10, texts: []string{"A polícia chegou, viu {@} batendo em todo mundo e levou junto"}, effects: []effect{nabbed()}},
				}},
			},
			idle: ending{texts: []string{"Quando {@} decidiu o que fazer, o arrastão já tinha passado e só sobrou o garçom cobrando a conta"}},
		},
		{
			id:    "bolada-casa",
			title: "Barulho na sala",
			when:  homeOwner,
			intros: []string{
				"Às três da manhã, {@} acordou com um barulho na sala. Alguém ficou sabendo da bolada e veio buscar pessoalmente",
				"{@} chegou em casa e a porta tava encostada. Lá de dentro, uma lanterna mexia nas gavetas",
			},
			choices: []choice{
				{label: "😴 Fingir que tá dormindo", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"O ladrão levou uns trocados da gaveta e a televisão. {@} fingiu tão bem que dormiu de verdade"}, effects: []effect{lose(3, 8)}},
					{weight: 40, texts: []string{"O ladrão não achou nada que prestasse e foi embora decepcionado com a decoração"}},
				}},
				{label: "📞 Ligar pra polícia", risk: midRisk, endings: []ending{
					{weight: 40, texts: []string{"A viatura chegou rápido dessa vez e pegou o ladrão no quintal. Ele ainda devolveu o que tinha na mochila"}, effects: []effect{earn(30, 100)}},
					{weight: 60, texts: []string{"A polícia chegou duas horas depois, anotou tudo e foi embora. O ladrão tinha ido bem antes, com uma parte da bolada"}, effects: []effect{lose(8, 15)}},
				}},
				{label: "🧹 Pegar o cabo de vassoura", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"{@} desceu a vassoura no ladrão, que pulou o muro e largou tudo pra trás, até o celular dele"}, effects: []effect{gain(1, "celular")}},
					{weight: 45, texts: []string{"O ladrão era maior que a porta. A vassoura não ajudou em nada"}, effects: []effect{injure(25, 45), lose(5, 12)}},
					{weight: 10, texts: []string{"O ladrão tava armado, e {@} levou uma vassoura pra um tiroteio"}, effects: []effect{perish()}},
				}},
				{label: "🔫 Esperar na escada armado", risk: midRisk, needs: armedUp, endings: []ending{
					{weight: 70, texts: []string{"O ladrão viu {@} na escada, largou tudo e saiu pela janela que nem entrou"}},
					{weight: 30, texts: []string{"O vizinho ouviu o tiro pro alto e chamou a polícia, que acabou fichando {@} também"}, effects: []effect{stain()}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} decidia, o ladrão tropeçou no tapete, fez barulho demais e fugiu sozinho"}},
		},
		{
			id:     "bolada-hacker",
			title:  "Conta invadida",
			rarity: rareEvent,
			intros: []string{
				"Pouco depois da bolada, chegou um e-mail pra {@}: 'invadimos sua conta. Pague um resgate ou a gente esvazia tudo'. Tinha até uma caveira em ASCII",
				"O celular de {@} começou a abrir aplicativo sozinho. Alguém tava lá dentro, de olho na bolada",
			},
			choices: []choice{
				{label: "🔑 Trocar todas as senhas", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} trocou tudo pra senhas enormes e o hacker ficou do lado de fora chorando"}},
					{weight: 30, texts: []string{"Trocou as senhas, mas o hacker já tinha feito um Pix antes. Pelo menos foi só um"}, effects: []effect{lose(5, 10)}},
				}},
				{label: "💰 Pagar o resgate", risk: midRisk, price: cost{percent: 5, floor: 300}, endings: []ending{
					{weight: 60, texts: []string{"O hacker, por incrível que pareça, cumpriu a palavra e ainda deixou uma avaliação cinco estrelas"}, effects: []effect{enchant("protecao")}},
					{weight: 40, texts: []string{"{@} pagou o resgate e o hacker pediu mais. Hacker não tem palavra"}, effects: []effect{lose(5, 15)}},
				}},
				{label: "🧑‍💻 Hackear o hacker de volta", risk: highRisk, endings: []ending{
					{weight: 35, texts: []string{"{@} abriu o terminal de tela preta, digitou rápido e esvaziou a conta do hacker"}, effects: []effect{earn(200, 500)}},
					{weight: 50, texts: []string{"{@} digitou 'hackear' no Google e baixou um vírus pior ainda"}, effects: []effect{lose(15, 25)}},
					{weight: 15, texts: []string{"A invasão chamou atenção da Polícia Federal, que achou que {@} era o hacker"}, effects: []effect{nabbed()}},
				}},
			},
			idle: ending{texts: []string{"{@} ignorou o e-mail e no dia seguinte descobriu que era só spam"}},
		},
		{
			id:     "bolada-relampago",
			title:  "Sequestro relâmpago",
			rarity: rareEvent,
			intros: []string{
				"Uma van preta parou do lado de {@}, a porta abriu e três mãos puxaram pra dentro. 'Vamo dar uma voltinha até o caixa eletrônico, ricaço'",
				"{@} entrou num carro de aplicativo pra ir comemorar a bolada. O motorista travou as portas e mudou a rota: 'primeiro a gente passa no banco'",
			},
			choices: []choice{
				{label: "🏧 Sacar o que eles pedirem", risk: midRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} sacou o limite do dia e foi largado num posto de gasolina, inteiro e bem mais pobre"}, effects: []effect{lose(12, 20)}},
					{weight: 30, texts: []string{"O caixa tava sem dinheiro. Os sequestradores ficaram tão frustrados que soltaram {@} na hora"}},
				}},
				{label: "🚪 Pular do carro andando", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} pulou num sinal fechado, rolou na calçada e sumiu num beco. Dolorido, mas com o dinheiro"}, effects: []effect{injure(20, 35)}},
					{weight: 40, texts: []string{"O carro não tava tão devagar assim"}, effects: []effect{injure(35, 50)}},
					{weight: 10, texts: []string{"{@} pulou bem na frente de um ônibus"}, effects: []effect{perish()}},
				}},
				{label: "🗣️ Convencer que é pobre", risk: midRisk, endings: []ending{
					{weight: 40, texts: []string{"{@} mostrou o extrato de um mês atrás e chorou. Os sequestradores ficaram com pena e deixaram dez conto pro ônibus"}, effects: []effect{earn(10, 30)}},
					{weight: 60, texts: []string{"Eles tinham o print do !ricos. Não colou"}, effects: []effect{lose(15, 25), injure(5, 15)}},
				}},
				{label: "🔪 Virar o jogo dentro da van", risk: highRisk, needs: killer, endings: []ending{
					{weight: 60, texts: []string{"Os sequestradores escolheram a pessoa errada. {@} saiu da van com a carteira de todos eles"}, effects: []effect{earn(200, 500), stain()}},
					{weight: 40, texts: []string{"Deu briga feia dentro da van, e {@} saiu de lá na ambulância"}, effects: []effect{injure(30, 50)}},
				}},
			},
			idle: ending{texts: []string{"{@} ficou tão quieto que os sequestradores acharam que tinham pegado a pessoa errada e largaram na esquina"}},
		},
		{
			id:     "bolada-quadrilha",
			title:  "O plano da quadrilha",
			rarity: legendaryEvent,
			intros: []string{
				"{@} ouviu sem querer, na mesa do lado, uma quadrilha conhecida combinando o assalto da semana. O alvo, pelo jeito, era {@}",
				"Um bilhete anônimo apareceu embaixo da porta de {@}: 'a quadrilha do Zé Pequeno vai atrás da sua bolada amanhã. Um amigo'",
			},
			choices: []choice{
				{label: "🛡️ Contratar segurança armada", risk: lowRisk, price: cost{percent: 8, floor: 1000}, endings: []ending{
					{weight: 80, texts: []string{"Com dois armários de terno do lado, {@} passou a semana em paz. A quadrilha desistiu"}, effects: []effect{enchant("protecao")}},
					{weight: 20, texts: []string{"Os seguranças eram da quadrilha. {@} pagou pra ser assaltado com escolta"}, effects: []effect{lose(10, 20)}},
				}},
				{label: "🧳 Sumir uns dias", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} passou uns dias na casa da avó no interior, comeu bem e voltou descansado"}, effects: []effect{heal(30)}},
					{weight: 40, texts: []string{"A quadrilha não achou {@} e levou o que tinha em casa no lugar"}, effects: []effect{lose(8, 15)}},
				}},
				{label: "🪤 Montar uma armadilha", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"A armadilha funcionou: a quadrilha caiu numa rede, a polícia levou todos e a recompensa ficou com {@}"}, effects: []effect{earn(500, 1500)}},
					{weight: 45, texts: []string{"A quadrilha desconfiou, desviou da armadilha e pegou {@} pelo outro lado"}, effects: []effect{lose(15, 25), injure(25, 45)}},
					{weight: 15, texts: []string{"A quadrilha não perdoa quem tenta fazer graça"}, effects: []effect{perish()}},
				}},
				{label: "🤝 Oferecer sociedade", risk: midRisk, needs: killer, endings: []ending{
					{weight: 60, texts: []string{"O chefe da quadrilha reconheceu {@} do meio e ofereceu uma parte do último serviço pela amizade"}, effects: []effect{earn(300, 800), stain()}},
					{weight: 40, texts: []string{"O chefe não gostou da concorrência e mandou recado com os punhos"}, effects: []effect{injure(20, 40)}},
				}},
			},
			idle: ending{texts: []string{"{@} não fez nada, e a quadrilha, por sorte, foi presa por outro crime antes de chegar"}},
		},
		{
			id:    "bolada-leao",
			title: "O leão quer a parte dele",
			intros: []string{
				"Chegou uma carta com o carimbo do leão na casa de {@}: 'identificamos um aumento patrimonial repentino. Favor acertar a mordida'",
				"Mal deu tempo de gastar e o fiscal já tava batendo na porta de {@}, de prancheta na mão: 'essa bolada aí foi declarada?'",
			},
			choices: []choice{
				{label: "🧾 Declarar tudo direitinho", risk: lowRisk, endings: []ending{
					{weight: 75, texts: []string{"{@} pagou o imposto da bolada certinho e ainda ganhou um 'parabéns pela honestidade' por carta"}, effects: []effect{skim(15, 25)}},
					{weight: 25, texts: []string{"O fiscal fez a conta errado a favor de {@} e foi embora sem cobrar nada. Ninguém vai avisar ele"}},
				}},
				{label: "🙈 Fingir que não viu a carta", risk: midRisk, endings: []ending{
					{weight: 45, texts: []string{"A carta foi pro lixo, o fiscal esqueceu e {@} seguiu a vida com a bolada inteira"}},
					{weight: 35, texts: []string{"O leão não esquece. Veio a multa por atraso, e com juros"}, effects: []effect{skim(30, 45)}},
					{weight: 20, texts: []string{"O leão bloqueou a conta de {@} e só soltou depois de morder um pedação da bolada"}, effects: []effect{skim(50, 65)}},
				}},
				{label: "🧮 Chamar um contador esperto", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O contador achou dedução até no pão de queijo. {@} pagou só uma mordidinha"}, effects: []effect{skim(5, 10)}},
					{weight: 30, texts: []string{"O contador achou uma restituição esquecida de anos atrás e {@} saiu ganhando"}, effects: []effect{earn(50, 150)}},
					{weight: 20, texts: []string{"O contador inventou tanta dedução que a Receita caiu em cima de {@} com multa"}, effects: []effect{skim(30, 50)}},
				}},
				{label: "🏝️ Mandar a bolada pras Ilhas Cayman", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"A bolada sumiu num paraíso fiscal e o leão ficou só olhando. {@} agora tem conta offshore"}},
					{weight: 40, texts: []string{"O doleiro sumiu com uma parte no caminho, e o leão ainda mordeu o resto por evasão"}, effects: []effect{skim(55, 75), stain()}},
					{weight: 20, texts: []string{"A Federal pegou a remessa no meio do caminho e {@} saiu do banco algemado"}, effects: []effect{skim(40, 60), nabbed()}},
				}},
				{label: "🕴️ Dar uma carteirada", risk: midRisk, needs: lawman, endings: []ending{
					{weight: 60, texts: []string{"{@} mostrou o distintivo, o fiscal engoliu seco e arquivou o processo"}},
					{weight: 40, texts: []string{"O fiscal anotou o nome de {@} e mandou pra corregedoria junto com a cobrança"}, effects: []effect{skim(15, 25), stain()}},
				}},
			},
			idle: ending{texts: []string{"{@} deixou a carta na mesa tanto tempo que ela virou descanso de copo. O leão, por enquanto, não voltou"}},
		},
	},
}
