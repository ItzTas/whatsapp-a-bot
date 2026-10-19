package economy

import "time"

var workEvents = eventBook{
	name:   "💼 Trampo",
	places: []string{"!trabalhar"},
	scenarios: []scenario{
		{
			id:    "trampo-bico-caloteiro",
			title: "O cliente caloteiro",
			areas: []int{areaBicos},
			intros: []string{
				"O cliente do bico de hoje bateu no ombro de {@} e mandou: \"Te pago semana que vem, palavra de honra\". Palavra de honra de quem já deve pra metade do bairro",
				"{@} terminou o bico e, na hora de receber, o cliente começou a procurar a carteira em bolsos que nem existiam",
			},
			choices: []choice{
				{label: "🤝 Confiar e esperar", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} anotou o nome do caloteiro no caderninho e foi embora com fé no ser humano"}, effects: []effect{plant("trampo-bico-caloteiro-volta", 6*time.Hour)}},
					{weight: 40, texts: []string{"O cliente sumiu do mapa, bloqueou {@} no zap e mudou até de corte de cabelo"}},
				}},
				{label: "📢 Expor no grupo do bairro", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"A vergonha bateu e o caloteiro pagou na hora, com juros e pedido de desculpa"}, effects: []effect{earn(60, 120)}},
					{weight: 30, texts: []string{"O caloteiro apareceu bufando e deu um empurrão em {@} na frente da padaria"}, effects: []effect{injure(5, 15)}},
					{weight: 20, texts: []string{"O post viralizou, mas o caloteiro era sobrinho do síndico. {@} ainda teve que pagar a advogada"}, effects: []effect{lose(2, 5)}},
				}},
				{label: "🔨 Cobrar na marra", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"{@} encostou o caloteiro no muro e saiu de lá com o dinheiro e mais um pouco de juros"}, effects: []effect{earn(100, 180), stain()}},
					{weight: 35, texts: []string{"O caloteiro era faixa preta de jiu-jitsu. {@} acordou no chão com o bico pago só em hematoma"}, effects: []effect{injure(15, 30), stain()}},
					{weight: 20, texts: []string{"Uma viatura passou bem na hora da cobrança"}, effects: []effect{nabbed()}},
				}},
				{label: "🃏 Cobrar no carteado", risk: midRisk, needs: isJob("crupiê", "apostador profissional"), endings: []ending{
					{weight: 70, texts: []string{"{@} chamou o caloteiro pra uma partidinha amistosa e recuperou tudo no truco, com baralho marcado"}, effects: []effect{earn(120, 200), stain()}},
					{weight: 30, texts: []string{"O caloteiro roubava melhor que {@} no baralho e ainda levou o troco"}, effects: []effect{lose(2, 5)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} decidia, o caloteiro aproveitou a distração e saiu de fininho"}},
		},
		{
			id:    "trampo-bico-caloteiro-volta",
			title: "O caloteiro voltou",
			stage: true,
			intros: []string{
				"Lembra do cliente caloteiro? Ele apareceu na porta de {@} com um sorrisão amarelo e um envelope gordo na mão",
				"Contra todas as previsões, o caloteiro do bico voltou. E trouxe um envelope e uma proposta pra {@}",
			},
			choices: []choice{
				{label: "💰 Pegar o envelope e contar", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"Tava tudo lá, e o caloteiro ainda pagou juros com medo de virar post"}, effects: []effect{earn(80, 160)}},
					{weight: 30, texts: []string{"Metade era dinheiro de brinquedo, mas pelo menos ele voltou"}, effects: []effect{earn(20, 40), gain(1, "dinheiro-de-brinquedo")}},
				}},
				{label: "🎲 Dobro ou nada com ele", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} topou o dobro ou nada no par ou ímpar e o caloteiro pagou em dobro, chorando"}, effects: []effect{earn(160, 300)}},
					{weight: 50, texts: []string{"Deu ímpar. O caloteiro saiu de lá com a dívida quitada e {@} saiu com cara de besta"}},
				}},
				{label: "🙏 Perdoar a dívida", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} perdoou a dívida. O caloteiro chorou, abraçou e jurou acender uma vela em sua homenagem"}, effects: []effect{enchant("sorte")}},
				}},
			},
			idle: ending{texts: []string{"O caloteiro cansou de esperar na porta e levou o envelope de volta"}},
		},
		{
			id:    "trampo-bico-renda-extra",
			title: "Renda extra garantida",
			areas: []int{areaBicos},
			intros: []string{
				"Chegou uma mensagem no zap de {@}: \"Ganhe 500 por dia só curtindo vídeo! Basta pagar a taxa de ativação\". O perfil tem foto de iate e nome de empresa com três Ls",
				"Um primo distante chamou {@} pra uma \"oportunidade de negócio\" que precisa só de um investimento inicial e muita fé",
			},
			choices: []choice{
				{label: "💸 Pagar a taxa de ativação", risk: midRisk, price: cost{percent: 5, floor: 50}, endings: []ending{
					{weight: 80, texts: []string{"O perfil sumiu cinco minutos depois do Pix. {@} curtiu vídeo de gatinho a tarde inteira de graça"}},
					{weight: 20, texts: []string{"Contra toda a lógica, pagaram mesmo. {@} recebeu e agora não sabe se conta pra alguém ou se esconde"}, effects: []effect{earn(150, 300)}},
				}},
				{label: "🕵️ Fingir que topa e caçar o golpista", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} enrolou o golpista por três dias e entregou tudo pra polícia, que pagou uma recompensinha"}, effects: []effect{earn(60, 120)}},
					{weight: 30, texts: []string{"O golpista percebeu a armação e bloqueou {@} depois de um áudio de 4 minutos xingando"}},
					{weight: 20, texts: []string{"O golpista era melhor de lábia e clonou o zap de {@}, que passou o dia avisando a família"}, effects: []effect{lose(3, 6)}},
				}},
				{label: "🗑️ Bloquear e seguir a vida", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} bloqueou, denunciou e voltou pro trampo com a sensação de dever cumprido"}},
					{weight: 30, texts: []string{"{@} ainda salvou a tia, que já tava com o cartão na mão pra pagar a taxa. Ela pagou um almoço de agradecimento"}, effects: []effect{earn(20, 50)}},
				}},
				{label: "🚀 Revender o esquema", risk: highRisk, needs: isJob("influencer de cripto", "coach"), endings: []ending{
					{weight: 50, texts: []string{"{@} gravou um story numa lancha alugada e vendeu 40 \"vagas exclusivas\". Dinheiro no bolso, consciência no lixo"}, effects: []effect{earn(200, 350), stain()}},
					{weight: 30, texts: []string{"Um dos compradores era policial disfarçado"}, effects: []effect{nabbed()}},
					{weight: 20, texts: []string{"Os seguidores descobriram e o cancelamento foi histórico"}, effects: []effect{lose(5, 10), enchant("olhogordo")}},
				}},
			},
			idle: ending{texts: []string{"A mensagem ficou sem resposta e o golpista foi tentar a sorte com outra pessoa do grupo"}},
		},
		{
			id:    "trampo-bico-amostra",
			title: "Amostra grátis duvidosa",
			areas: []int{areaBicos},
			intros: []string{
				"Uma marca nova de energético chamada TURBO MAX 9000 abordou {@} na rua: querem um depoimento em vídeo provando o produto. A latinha tá meio enferrujada",
				"Um laboratório que funciona no fundo de uma lan house quer pagar {@} pra provar um suplemento \"revolucionário\" na frente da câmera",
			},
			choices: []choice{
				{label: "🥤 Provar na câmera", risk: midRisk, endings: []ending{
					{weight: 45, texts: []string{"{@} virou a latinha, fez cara de êxtase e o depoimento rendeu um cachê decente"}, effects: []effect{earn(60, 120)}},
					{weight: 30, texts: []string{"Bateu errado: {@} passou a tarde abraçado com o vaso sanitário"}, effects: []effect{injure(5, 15)}},
					{weight: 25, texts: []string{"Era basicamente cafeína pura. {@} ficou tão elétrico que voltou a trabalhar na hora"}, effects: []effect{refresh()}},
				}},
				{label: "🙅 Recusar com educação", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} recusou. O promotor ficou triste e deu uma latinha de brinde mesmo assim"}, effects: []effect{gain(1, "energetico")}},
					{weight: 40, texts: []string{"{@} recusou e foi embora antes que alguém mais desesperado aceitasse por menos"}},
				}},
				{label: "📦 Vender as amostras no estádio", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} vendeu a caixa inteira na porta do estádio, a 5 reais cada"}, effects: []effect{earn(40, 90)}},
					{weight: 40, texts: []string{"O fiscal da prefeitura pegou {@} vendendo sem alvará e apreendeu tudo"}, effects: []effect{lose(2, 5), delay(20 * time.Minute)}},
				}},
			},
			idle: ending{texts: []string{"O promotor achou outra cobaia, e {@} perdeu o cachê e a chance de virar meme"}},
		},
		{
			id:    "trampo-bico-fantasia",
			title: "A fantasia travou",
			areas: []int{areaBicos},
			intros: []string{
				"No meio do bico, o zíper da fantasia de {@} travou. Lá fora tá fazendo 38 graus e lá dentro deve estar uns 60",
				"{@} tava fantasiado de mascote de loja quando o zíper emperrou de vez. O suor já tá formando um lago dentro da cabeça de pelúcia",
			},
			choices: []choice{
				{label: "✂️ Rasgar a fantasia", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} rasgou a fantasia e saiu respirando igual baleia. O dono da loja descontou o prejuízo"}, effects: []effect{lose(2, 4)}},
				}},
				{label: "🥵 Aguentar até o fim do turno", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} aguentou firme e o gerente ficou tão emocionado com a dedicação que deu um bônus"}, effects: []effect{earn(60, 120)}},
					{weight: 50, texts: []string{"{@} desmaiou abraçado numa criança, que achou que era parte do show"}, effects: []effect{injure(10, 25)}},
				}},
				{label: "📣 Pedir ajuda pro público", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"Uma senhora com tesoura de unha salvou {@}, e o vídeo do resgate viralizou com trilha de novela"}, effects: []effect{earn(20, 60)}},
					{weight: 40, texts: []string{"O público achou que era encenação e ficou só filmando"}, effects: []effect{delay(15 * time.Minute)}},
				}},
			},
			idle: ending{texts: []string{"{@} ficou parado no sol esperando um milagre, que só veio no fim do turno, quando já tava meio cozido"}, effects: []effect{injure(5, 10)}},
		},
		{
			id:    "trampo-bico-dobro-ou-nada",
			title: "Dobro ou nada",
			areas: []int{areaBicos},
			intros: []string{
				"Na saída do bico, um sujeito de chapéu panamá e dente de ouro chamou {@}: \"Te dobro o pagamento de hoje no cara ou coroa. Tá com medo?\"",
				"Um apostador da esquina viu {@} recebendo o pagamento e já veio sacudindo uma moeda: dobro ou nada, agora",
			},
			choices: []choice{
				{label: "🪙 Apostar o pagamento", risk: midRisk, endings: []ending{
					{weight: 45, texts: []string{"Deu cara! {@} saiu com o pagamento dobrado e o apostador saiu resmungando"}, effects: []effect{earn(100, 180)}},
					{weight: 55, texts: []string{"Deu coroa. A moeda tinha coroa dos dois lados, mas {@} só reparou depois"}, effects: []effect{lose(3, 7)}},
				}},
				{label: "🚶 Não apostar", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} guardou o dinheiro no bolso da frente e foi embora sem olhar pra trás"}},
				}},
				{label: "🗣️ Chamar ele de golpista", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"O povo da esquina deu razão pra {@} e o apostador foi embora escorraçado"}},
					{weight: 40, texts: []string{"O apostador era primo do dono do bar e {@} foi expulso da esquina"}, effects: []effect{delay(20 * time.Minute)}},
				}},
				{label: "🃏 Trocar a moeda dele", risk: highRisk, needs: isJob("crupiê", "apostador profissional"), endings: []ending{
					{weight: 70, texts: []string{"{@} trocou a moeda viciada por uma honesta sem ele perceber e ganhou o dobro"}, effects: []effect{earn(150, 250), stain()}},
					{weight: 30, texts: []string{"O apostador viu a troca e resolveu do jeito antigo: na mão"}, effects: []effect{injure(15, 30)}},
				}},
			},
			idle: ending{texts: []string{"O apostador cansou de esperar e foi atrás de um patinho mais rápido"}},
		},
		{
			id:    "trampo-bico-pagamento-alternativo",
			title: "Pagamento alternativo",
			areas: []int{areaBicos},
			intros: []string{
				"O cliente do bico de hoje abriu a carteira e disse que tá sem dinheiro, mas tem uma proposta: pagar {@} com uma raspadinha, dois tazos e um cupom de desconto",
				"\"Pix não, mas aceita uma raspadinha?\". Foi assim que o cliente resolveu pagar o bico de {@}",
			},
			choices: []choice{
				{label: "🎫 Aceitar a raspadinha", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} pegou a raspadinha e foi embora rezando pra sair alguma coisa"}, effects: []effect{gain(1, "raspadinha")}},
				}},
				{label: "💵 Exigir dinheiro vivo", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"O cliente achou dinheiro escondido na meia e pagou tudo, meio úmido"}, effects: []effect{earn(50, 100)}},
					{weight: 40, texts: []string{"O cliente se ofendeu, guardou a raspadinha e ainda falou mal de {@} no Google"}},
				}},
				{label: "🧸 Levar tudo que ele tiver", risk: midRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} saiu de lá com a raspadinha, os tazos e o cupom vencido de brinde"}, effects: []effect{gain(1, "raspadinha"), gain(2, "tazo"), gain(1, "cupom-vencido")}},
					{weight: 30, texts: []string{"O cliente achou abuso e jogou um tazo na testa de {@}"}, effects: []effect{injure(3, 8)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, o cliente guardou tudo de volta e prometeu pagar semana que vem"}},
		},
		{
			id:    "trampo-obra-andaime",
			title: "O andaime balançou",
			areas: []int{areaObras},
			intros: []string{
				"{@} tava no terceiro andar do andaime quando a estrutura deu uma balançada de navio pirata. Lá embaixo, o mestre de obras gritou \"segura aí!\"",
				"Uma rajada de vento pegou o andaime de jeito, e agora {@} tá abraçando um cano de ferro igual criança abraça bicho de pelúcia",
			},
			choices: []choice{
				{label: "🧗 Descer devagar", risk: lowRisk, endings: []ending{
					{weight: 80, texts: []string{"{@} desceu degrau por degrau, rezando um terço inteiro no caminho, e chegou lá embaixo inteiro"}, effects: []effect{delay(10 * time.Minute)}},
					{weight: 20, texts: []string{"Na pressa, {@} ralou a canela no último degrau"}, effects: []effect{injure(3, 8)}},
				}},
				{label: "🪢 Agarrar a corda", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"A corda segurou e o andaime parou de balançar. {@} terminou o serviço e virou lenda no canteiro"}, effects: []effect{earn(40, 90)}},
					{weight: 40, texts: []string{"A corda segurou, mas o andaime não"}, next: "trampo-obra-andaime-pendurado"},
				}},
				{label: "🦘 Pular pro andar de baixo", risk: highRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} pulou igual filme de ação e caiu em pé na laje. A peãozada aplaudiu"}, effects: []effect{earn(20, 50)}},
					{weight: 38, texts: []string{"{@} errou a laje por um metro e caiu no monte de areia, mas antes bateu em tudo que tinha no caminho"}, effects: []effect{injure(30, 50)}},
					{weight: 7, texts: []string{"{@} errou o pulo feio"}, effects: []effect{perish()}},
				}},
				{label: "🦺 Confiar no cinto do Kit EPI", risk: lowRisk, needs: holds(safetyKitID), endings: []ending{
					{weight: 100, texts: []string{"O cinto do Kit EPI segurou {@} no ar igual marionete. Susto grande, arranhão nenhum, e o mestre de obras deu um extra"}, effects: []effect{earn(30, 60)}},
				}},
			},
			idle: ending{texts: []string{"{@} ficou paralisado abraçando o cano até o andaime parar sozinho, uns vinte minutos depois"}, effects: []effect{delay(20 * time.Minute)}},
		},
		{
			id:    "trampo-obra-andaime-pendurado",
			title: "Pendurado no quinto andar",
			stage: true,
			intros: []string{
				"{@} tá pendurado na corda do lado de fora do prédio, no quinto andar. O mestre de obras grita lá de baixo, a vizinha do prédio da frente tá filmando e o vento não ajuda",
				"Balançando no quinto andar igual pêndulo de relógio, {@} precisa decidir rápido como sair dessa",
			},
			choices: []choice{
				{label: "🪟 Entrar pela janela mais perto", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} entrou pela janela bem na hora do almoço da família, que ofereceu um prato de feijoada"}, effects: []effect{heal(15)}},
					{weight: 40, texts: []string{"Era o apartamento de um policial aposentado, que achou que era invasão e atacou com o chinelo"}, effects: []effect{injure(10, 20)}},
				}},
				{label: "🙏 Esperar o resgate", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"Os bombeiros chegaram em 40 minutos e desceram {@} sob aplausos do quarteirão inteiro"}, effects: []effect{delay(30 * time.Minute)}},
				}},
				{label: "🧗 Descer escalando", risk: highRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} desceu escalando a parede igual homem-aranha e ainda cobrou o serviço como hora extra"}, effects: []effect{earn(80, 160)}},
					{weight: 37, texts: []string{"No segundo andar a mão escorregou, e o monte de areia amorteceu só metade da queda"}, effects: []effect{injure(30, 55)}},
					{weight: 8, texts: []string{"A mão escorregou lá do quinto andar"}, effects: []effect{perish()}},
				}},
			},
			idle: ending{texts: []string{"{@} ficou pendurado tanto tempo que os bombeiros tiveram que vir, e a vizinha ganhou 2 milhões de views"}, effects: []effect{delay(30 * time.Minute)}},
		},
		{
			id:    "trampo-obra-patrao-sumiu",
			title: "O patrão sumiu",
			areas: []int{areaObras},
			intros: []string{
				"Dia de pagamento na obra e o patrão desapareceu. O telefone dele dá caixa postal e o carro sumiu da garagem",
				"{@} terminou a parte dele da obra, mas o dono mandou mensagem dizendo que só paga \"quando vender o terreno\"",
			},
			choices: []choice{
				{label: "⏳ Esperar mais uma semana", risk: lowRisk, endings: []ending{
					{weight: 50, texts: []string{"O patrão reapareceu bronzeado de Cancún e pagou, sem graça nenhuma"}, effects: []effect{earn(40, 80)}},
					{weight: 50, texts: []string{"O patrão continua sumido. {@} ficou só com a promessa e a obra pela metade"}},
				}},
				{label: "🧱 Levar material como pagamento", risk: midRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} encheu o carrinho de mão com o que achou e considerou a dívida quitada"}, effects: []effect{gain(3, "tijolo", "ferro", "cobre"), stain()}},
					{weight: 30, texts: []string{"O vizinho fofoqueiro filmou e chamou a polícia achando que era roubo. E era meio que roubo mesmo"}, effects: []effect{nabbed()}},
				}},
				{label: "⚖️ Botar na Justiça do Trabalho", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"O juiz deu ganho de causa e o patrão teve que pagar com correção"}, effects: []effect{earn(80, 160)}},
					{weight: 45, texts: []string{"O processo vai demorar uns 7 anos, e o advogado já cobrou a consulta"}, effects: []effect{lose(2, 5)}},
				}},
				{label: "📢 Fazer greve com a peãozada", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"A obra parou, a vizinhança reclamou e o patrão apareceu com o dinheiro e uma caixa de cerveja"}, effects: []effect{earn(30, 70)}},
					{weight: 40, texts: []string{"A greve virou churrasco na laje e ninguém lembrou de cobrar nada"}, effects: []effect{heal(10)}},
				}},
			},
			idle: ending{texts: []string{"{@} ficou esperando o patrão até o sol se pôr. Ele não veio"}},
		},
		{
			id:    "trampo-obra-material-sumindo",
			title: "O cimento que some",
			areas: []int{areaObras},
			intros: []string{
				"Toda manhã some saco de cimento da obra. O dono já desconfia da equipe inteira, e {@} tá na lista",
				"Alguém tá desviando material da obra, e quem chega cedo como {@} vê o rastro de pó de cimento até o portão",
			},
			choices: []choice{
				{label: "🕵️ Ficar de tocaia à noite", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} pegou o ladrão no flagra: era o filho do dono, que vendia o cimento pra comprar figurinha. O dono pagou uma recompensa pelo silêncio"}, effects: []effect{earn(80, 150)}},
					{weight: 30, texts: []string{"Era o próprio dono desviando pra outra obra dele. {@} fingiu que não viu e ganhou um aumento misterioso"}, effects: []effect{earn(40, 80)}},
					{weight: 20, texts: []string{"{@} cochilou na tocaia e acordou com uma paulada de quem levava o cimento"}, effects: []effect{injure(10, 25)}},
				}},
				{label: "🤫 Entrar no esquema", risk: highRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} virou sócio do desvio e vendeu cimento pela metade do preço pra obra do vizinho"}, effects: []effect{earn(120, 220), stain()}},
					{weight: 25, texts: []string{"A obra do vizinho era de um delegado"}, effects: []effect{nabbed()}},
					{weight: 15, texts: []string{"O dono descobriu e mandou meio mundo embora, inclusive {@}"}, effects: []effect{lose(3, 6), delay(40 * time.Minute)}},
				}},
				{label: "🙈 Fingir que não viu", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} ficou na dele, e o problema se resolveu sozinho quando o ladrão foi preso em outra obra"}},
					{weight: 30, texts: []string{"Sobrou pra {@}, que teve um saco de cimento descontado do pagamento"}, effects: []effect{lose(2, 4)}},
				}},
				{label: "🧰 Trocar a fechadura do depósito", risk: lowRisk, needs: holds("caixa-de-ferramentas"), endings: []ending{
					{weight: 100, texts: []string{"{@} trocou a fechadura com a caixa de ferramentas e o desvio acabou na hora. O dono pagou um extra"}, effects: []effect{earn(60, 120)}},
				}},
			},
			idle: ending{texts: []string{"{@} enrolou pra decidir e o dono descontou dois sacos de cimento de todo mundo"}, effects: []effect{lose(2, 4)}},
		},
		{
			id:    "trampo-obra-traco-errado",
			title: "O traço errado",
			areas: []int{areaObras},
			intros: []string{
				"{@} errou o traço do cimento: era pra ser três de areia pra um de cimento e saiu o contrário. A parede já tá de pé e meio torta",
				"A parede que {@} levantou hoje cedo tá com uma barriga esquisita e o prumo apontando pro vizinho",
			},
			choices: []choice{
				{label: "🔨 Derrubar e fazer de novo", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} derrubou tudo e refez no capricho. Demorou, mas o mestre de obras aprovou e deu um agrado"}, effects: []effect{delay(20 * time.Minute), earn(20, 50)}},
				}},
				{label: "🎨 Disfarçar com massa corrida", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"Duas demãos de massa corrida e uma de tinta, e ninguém percebeu nada"}, effects: []effect{earn(40, 80)}},
					{weight: 45, texts: []string{"No dia da vistoria, a parede caiu inteira em cima do pé de {@}"}, effects: []effect{injure(10, 25), lose(3, 6)}},
				}},
				{label: "👉 Culpar o servente", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O mestre de obras acreditou e descontou do servente. {@} saiu limpo, mas o karma anotou"}, effects: []effect{enchant("azar")}},
					{weight: 50, texts: []string{"O servente tinha filmado tudo pro TikTok, e {@} pagou o cimento do próprio bolso"}, effects: []effect{lose(3, 7)}},
				}},
				{label: "📐 Corrigir com técnica de mestre", risk: lowRisk, needs: isJob("mestre de obras", "pedreiro"), endings: []ending{
					{weight: 100, texts: []string{"{@} escorou, cortou e reforçou com ferro, e a parede torta virou \"design arrojado\". O cliente pagou a mais"}, effects: []effect{earn(60, 120)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, o cliente chegou e viu a parede torta"}, effects: []effect{lose(2, 4)}},
		},
		{
			id:     "trampo-obra-tesouro-na-parede",
			title:  "Tesouro na parede",
			rarity: rareEvent,
			areas:  []int{areaObras},
			intros: []string{
				"{@} tava quebrando uma parede velha numa reforma quando a marreta bateu em algo oco. Lá dentro tem uma caixa de lata enferrujada cheia de moedas antigas",
				"Atrás do azulejo de 1970 que {@} arrancou hoje tinha um esconderijo com uma caixinha de moedas do tempo do Império",
			},
			choices: []choice{
				{label: "🏠 Entregar pro dono da casa", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O dono chorou: era a herança perdida do avô. Deu uma gorjeta generosa pra {@}"}, effects: []effect{earn(150, 300)}},
					{weight: 30, texts: []string{"O dono pegou a caixa e só disse \"obrigado, pode continuar\". Mas o universo viu"}, effects: []effect{enchant("sorte")}},
				}},
				{label: "🤑 Ficar com tudo", risk: highRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} enfiou a caixa na bolsa da marmita e saiu assobiando"}, effects: []effect{gain(2, "moeda-do-imperio"), earn(150, 300), stain()}},
					{weight: 30, texts: []string{"A câmera da casa pegou tudo"}, effects: []effect{nabbed()}},
					{weight: 15, texts: []string{"A caixa tinha uma maldição escrita em latim na tampa, e {@} leu em voz alta"}, effects: []effect{gain(1, "moeda-amaldicoada"), enchant("miseria")}},
				}},
				{label: "🤝 Dividir com a peãozada", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} dividiu as moedas com a equipe e virou o herói do canteiro. Quem divide, multiplica"}, effects: []effect{gain(1, "moeda-do-imperio"), earn(30, 70), enchant("multiplicacao")}},
					{weight: 40, texts: []string{"Na divisão, o servente brigou pela moeda mais bonita e saiu soco"}, effects: []effect{gain(1, "moeda-do-imperio"), injure(5, 15)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} decidia, o servente pegou a caixa e sumiu de bicicleta"}},
		},
		{
			id:    "trampo-obra-fio-desencapado",
			title: "O fio desencapado",
			areas: []int{areaObras},
			intros: []string{
				"No meio da reforma, {@} achou um fio desencapado soltando faísca bem onde precisa furar a parede",
				"O quadro de luz da obra é uma macarronada de fios, e um deles tá desencapado e estalando do lado da betoneira",
			},
			choices: []choice{
				{label: "🔌 Procurar o disjuntor", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} levou meia hora pra achar o disjuntor certo, desligou tudo e terminou o serviço em paz"}, effects: []effect{delay(15 * time.Minute)}},
				}},
				{label: "⚡ Mexer assim mesmo", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"Tremeu a mão, mas deu certo. {@} terminou na metade do tempo"}, effects: []effect{earn(40, 90)}},
					{weight: 42, texts: []string{"O choque foi tão forte que o cabelo de {@} ficou igual o do Einstein"}, effects: []effect{injure(20, 40)}},
					{weight: 8, texts: []string{"O choque foi forte demais"}, effects: []effect{perish()}},
				}},
				{label: "📞 Pagar um eletricista", risk: lowRisk, price: cost{percent: 3, floor: 40}, endings: []ending{
					{weight: 100, texts: []string{"O eletricista resolveu em cinco minutos e cobrou como se fosse uma hora, mas {@} voltou pro trampo na hora"}, effects: []effect{refresh()}},
				}},
				{label: "🧠 Resolver como profissional", risk: lowRisk, needs: isJob("eletricista"), endings: []ending{
					{weight: 100, texts: []string{"{@} isolou o fio de olho fechado, cobrou o reparo à parte e ainda vendeu um quadro de luz novo pro cliente"}, effects: []effect{earn(80, 150)}},
				}},
			},
			idle: ending{texts: []string{"{@} ficou olhando pro fio até alguém tropeçar nele e o disjuntor cair sozinho"}, effects: []effect{delay(10 * time.Minute)}},
		},
		{
			id:    "trampo-transporte-mochila",
			title: "A mochila esquecida",
			areas: []int{areaTransporte},
			intros: []string{
				"Depois do último serviço, {@} achou uma mochila preta esquecida no banco de trás. Tá pesada e o zíper tá meio aberto",
				"Um passageiro desceu correndo e deixou uma mochila pra trás. {@} já viu um bolo de notas aparecendo pelo zíper",
			},
			choices: []choice{
				{label: "📍 Devolver pro dono", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"O dono era um senhorzinho que tava levando o dinheiro da reforma. Ele chorou e deu uma recompensa pra {@}"}, effects: []effect{earn(60, 120), enchant("sorte")}},
					{weight: 40, texts: []string{"O dono pegou a mochila, conferiu nota por nota e nem agradeceu"}},
				}},
				{label: "🔍 Abrir e ver o que tem", risk: midRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} abriu a mochila devagarinho"}, next: "trampo-transporte-mochila-aberta"},
				}},
				{label: "🚔 Levar pra delegacia", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O delegado agradeceu a honestidade e deu um certificado de cidadão de bem. Só isso mesmo"}, effects: []effect{delay(20 * time.Minute)}},
					{weight: 30, texts: []string{"Na delegacia descobriram que a mochila era de um foragido, e {@} levou a recompensa"}, effects: []effect{earn(100, 200)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, o dono voltou correndo e pegou a mochila pela janela"}},
		},
		{
			id:    "trampo-transporte-mochila-aberta",
			title: "O que tinha na mochila",
			stage: true,
			intros: []string{
				"Dinheiro vivo, um celular tocando sem parar com o contato \"CHEFE 🔪\" e um pacotinho branco. {@} tá suando frio no banco do motorista",
				"Na mochila tem um bolo de dinheiro, um pacotinho branco e um celular que não para de tocar. Na tela: \"CHEFE 🔪\". E {@} ainda tá com tudo na mão",
			},
			choices: []choice{
				{label: "💸 Pegar o dinheiro e sumir", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} trocou de cidade, de corte de cabelo e de chip. Ninguém nunca veio cobrar"}, effects: []effect{earn(250, 450), stain()}},
					{weight: 30, texts: []string{"O \"CHEFE\" achou {@} pelo aplicativo. A conversa foi curta e dolorida"}, effects: []effect{injure(30, 50)}},
					{weight: 20, texts: []string{"A polícia tava seguindo a mochila"}, effects: []effect{nabbed()}},
				}},
				{label: "📱 Atender o celular", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Era o dono, morrendo de medo. Ofereceu uma recompensa pra {@} devolver tudo sem fazer pergunta"}, effects: []effect{earn(120, 220)}},
					{weight: 30, texts: []string{"Era o CHEFE. Ele mandou {@} esquecer o que viu e ainda deixou um \"agrado\" pelo silêncio"}, effects: []effect{gain(2, "baseado"), earn(40, 80)}},
					{weight: 20, texts: []string{"Era o CHEFE, e ele não gostou nada da voz de {@}"}, effects: []effect{injure(10, 25), enchant("olhogordo")}},
				}},
				{label: "❄️ Experimentar o pacotinho", risk: highRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} deu uma provinha e saiu dirigindo a 300 por hora, falando sozinho"}, effects: []effect{buzz("po")}},
					{weight: 35, texts: []string{"Era farinha batizada com sei lá o quê. {@} passou mal no acostamento"}, effects: []effect{injure(10, 25)}},
					{weight: 10, texts: []string{"Era coisa forte demais"}, effects: []effect{buzz("po"), injure(30, 50)}},
				}},
				{label: "🌊 Largar tudo no rio", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} jogou a mochila no rio e foi embora fingindo que nunca viu nada. Dormiu mal, mas dormiu solto"}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, um carro preto encostou e alguém pegou a mochila pela janela sem dizer nada"}},
		},
		{
			id:    "trampo-transporte-blitz",
			title: "Blitz na avenida",
			areas: []int{areaTransporte},
			intros: []string{
				"Blitz no meio da avenida, e o documento do veículo de {@} venceu faz três meses",
				"O guarda de trânsito levantou a mão bem na frente de {@}. Tem lanterna, colete refletivo e cara de quem não tá pra conversa",
			},
			choices: []choice{
				{label: "📄 Mostrar os documentos e torcer", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"O guarda tava com pressa e liberou {@} com um \"vai com Deus\""}},
					{weight: 40, texts: []string{"Multa na hora, e ainda teve que esperar o guincho"}, effects: []effect{gain(1, "multa"), lose(2, 4)}},
				}},
				{label: "☕ Oferecer um cafezinho pro guarda", risk: highRisk, price: cost{percent: 4, floor: 50}, endings: []ending{
					{weight: 65, texts: []string{"O guarda aceitou o \"cafezinho\" e de repente o documento ficou em dia"}, effects: []effect{stain()}},
					{weight: 35, texts: []string{"O guarda era da corregedoria"}, effects: []effect{nabbed()}},
				}},
				{label: "🏃 Furar a blitz", risk: highRisk, endings: []ending{
					{weight: 45, favor: func(p Profession) float64 { return p.escapeBonus }, texts: []string{"{@} meteu o pé e sumiu na rua de trás antes da viatura dar a volta"}, effects: []effect{stain()}},
					{weight: 35, texts: []string{"A viatura alcançou {@} três quarteirões depois"}, effects: []effect{nabbed()}},
					{weight: 20, texts: []string{"Na fuga, {@} bateu num poste e o painel virou sanfona"}, effects: []effect{injure(15, 30), breakTool()}},
				}},
				{label: "😇 Chorar e contar a história da vida", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"A história da avó doente comoveu o guarda, que liberou {@} e ainda deu um abraço"}},
					{weight: 50, texts: []string{"O guarda já tinha ouvido essa história hoje umas quinze vezes"}, effects: []effect{gain(1, "multa"), lose(3, 6)}},
				}},
			},
			idle: ending{texts: []string{"{@} enrolou tanto pra responder que o guarda multou por desobediência"}, effects: []effect{gain(1, "multa"), lose(2, 4)}},
		},
		{
			id:    "trampo-transporte-pneu-furado",
			title: "Pneu furado na estrada",
			areas: []int{areaTransporte},
			intros: []string{
				"PÁ! O pneu estourou no meio da estrada, sem acostamento e com o sol rachando. {@} já tá ouvindo o barulho do aro no asfalto",
				"No meio do nada, entre dois postos que ficam a 80 km um do outro, o pneu de {@} furou",
			},
			choices: []choice{
				{label: "🔧 Trocar sozinho", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} trocou o pneu em vinte minutos e saiu todo sujo de graxa"}, effects: []effect{delay(10 * time.Minute)}},
					{weight: 20, texts: []string{"Enquanto trocava o pneu, {@} achou uma coisa largada no acostamento"}, effects: []effect{findLoot()}},
					{weight: 30, texts: []string{"O macaco escorregou e o carro desceu em cima da mão de {@}"}, effects: []effect{injure(10, 20)}},
				}},
				{label: "🚛 Pedir ajuda pro caminhoneiro", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"O caminhoneiro trocou o pneu em 5 minutos, contou a vida inteira e ainda deu um adesivo de Jesus"}},
					{weight: 25, texts: []string{"O caminhoneiro era bom de papo e melhor ainda de golpe: levou o estepe e um dinheiro do porta-luvas"}, effects: []effect{lose(4, 8)}},
					{weight: 15, texts: []string{"O caminhoneiro ajudou e ainda deu um rebite \"pra viagem render\""}, effects: []effect{buzz("rebite")}},
				}},
				{label: "🛞 Seguir rodando no aro", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} chegou no destino rodando no aro, soltando faísca igual carro alegórico"}},
					{weight: 50, texts: []string{"A trepidação foi tanta que soltou peça do carro inteiro, e o que tava no painel caiu e se espatifou"}, effects: []effect{breakTool(), delay(15 * time.Minute)}},
				}},
				{label: "🧭 Achar uma borracharia no GPS", risk: lowRisk, needs: holds("gps"), endings: []ending{
					{weight: 100, texts: []string{"O GPS achou uma borracharia escondida a 2 km, e o borracheiro ainda indicou um cliente pra {@}"}, effects: []effect{earn(30, 60)}},
				}},
			},
			idle: ending{texts: []string{"{@} ficou parado esperando um milagre, que só chegou duas horas depois na forma de um guincho"}, effects: []effect{delay(25 * time.Minute)}},
		},
		{
			id:    "trampo-transporte-passageiro-famoso",
			title: "Passageiro famoso",
			areas: []int{areaTransporte},
			intros: []string{
				"Entrou no veículo de {@} um cantor sertanejo famoso, de óculos escuros e boné, fingindo que ninguém ia reconhecer",
				"A passageira de hoje é uma ex-BBB que tá indo pra um evento. Ela já pediu pra {@} não postar nada",
			},
			choices: []choice{
				{label: "🤳 Pedir uma foto", risk: lowRisk, endings: []ending{
					{weight: 45, texts: []string{"A celebridade tirou a foto e ainda autografou um guardanapo pra {@}"}, effects: []effect{gain(1, "autografo")}},
					{weight: 55, texts: []string{"Saiu a foto, mas com o dedo de {@} na frente da câmera"}},
				}},
				{label: "🗣️ Puxar papo sobre a carreira", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"A conversa rendeu tanto que a gorjeta veio em nota de cem"}, effects: []effect{earn(60, 120)}},
					{weight: 30, texts: []string{"{@} falou demais da própria vida e a estrela dormiu no meio da história"}},
				}},
				{label: "📰 Vender a fofoca pro site", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O site de fofoca pagou bem pela exclusiva da viagem"}, effects: []effect{earn(150, 280)}},
					{weight: 30, texts: []string{"A assessoria descobriu quem vazou e o aplicativo suspendeu {@} por um tempo"}, effects: []effect{delay(40 * time.Minute)}},
					{weight: 20, texts: []string{"O processo por invasão de privacidade chegou rapidinho"}, effects: []effect{lose(5, 10)}},
				}},
				{label: "🗺️ Ir pelo caminho mais longo", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"A celebridade nem percebeu o tour pela cidade e pagou a viagem mais cara do mês"}, effects: []effect{earn(80, 160)}},
					{weight: 50, texts: []string{"Percebeu, deu uma estrela e postou o nome de {@} pros 20 milhões de seguidores"}, effects: []effect{enchant("olhogordo"), delay(20 * time.Minute)}},
				}},
			},
			idle: ending{texts: []string{"A estrela desceu no meio do caminho achando que {@} era fã demais"}},
		},
		{
			id:    "trampo-transporte-corrida-suspeita",
			title: "A corrida suspeita",
			areas: []int{areaTransporte},
			intros: []string{
				"Dois passageiros de touca entraram e pediram pra {@} esperar na frente do banco com o motor ligado, \"rapidinho\"",
				"\"Para aqui, deixa o motor ligado e não desliga por nada\", disse o passageiro, apontando pra uma joalheria. {@} ficou com uma pulga atrás da orelha",
			},
			choices: []choice{
				{label: "⏱️ Esperar com o motor ligado", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"Eles voltaram correndo com sacolas, e a gorjeta foi a maior da vida de {@}. Melhor nem perguntar"}, effects: []effect{earn(200, 350), stain()}},
					{weight: 35, texts: []string{"Era assalto mesmo, e {@} virou motorista de fuga sem querer"}, effects: []effect{nabbed()}},
					{weight: 20, texts: []string{"Teve tiroteio na saída e sobrou estilhaço pra todo lado"}, effects: []effect{injure(15, 35), stain()}},
				}},
				{label: "🚗 Ir embora na hora", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} arrancou antes deles voltarem e foi pegar outro serviço em paz"}},
				}},
				{label: "📞 Chamar a polícia", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"A polícia pegou a quadrilha e deu um prêmio pra {@} pela denúncia"}, effects: []effect{earn(80, 160)}},
					{weight: 40, texts: []string{"Um deles anotou a placa de {@} antes de ser preso. Agora é viver olhando pelo retrovisor"}, effects: []effect{enchant("olhogordo")}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, os dois voltaram, olharam feio e foram embora a pé"}},
		},
		{
			id:    "trampo-transporte-gorjeta-ou-nota",
			title: "Gorjeta ou cinco estrelas",
			areas: []int{areaTransporte},
			intros: []string{
				"No fim da viagem, a passageira perguntou pra {@}: \"Prefere uma gorjeta ou cinco estrelas com elogio?\"",
				"O cliente do frete gostou tanto do serviço que deixou {@} escolher: dinheiro na mão ou avaliação máxima no aplicativo",
			},
			choices: []choice{
				{label: "✉️ Ficar com a gorjeta", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} pegou o envelope e já foi sacudindo pra ouvir o barulho das moedas"}, effects: []effect{gain(1, "gorjeta")}},
				}},
				{label: "⭐ Pedir as cinco estrelas", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"As cinco estrelas subiram a nota de {@} e o aplicativo mandou outro serviço na mesma hora"}, effects: []effect{refresh()}},
				}},
				{label: "🤑 Pedir os dois", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"A passageira riu, achou {@} engraçado e deu os dois"}, effects: []effect{gain(1, "gorjeta"), refresh()}},
					{weight: 50, texts: []string{"A passageira ficou ofendida, não deu nada e ainda avaliou com uma estrela"}, effects: []effect{delay(20 * time.Minute)}},
				}},
			},
			idle: ending{texts: []string{"A passageira cansou de esperar a resposta e desceu sem dar nenhum dos dois"}},
		},
		{
			id:    "trampo-comida-vigilancia",
			title: "A vigilância sanitária",
			areas: []int{areaComida},
			intros: []string{
				"Chegou a vigilância sanitária sem avisar, de jaleco e prancheta, bem na hora que {@} tava pegando a coxinha que caiu no chão",
				"O fiscal da vigilância sanitária entrou na cozinha de {@} passando o dedo em tudo e fazendo careta",
			},
			choices: []choice{
				{label: "🧾 Mostrar tudo com orgulho", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O fiscal aprovou e colou o selo de qualidade na porta. O movimento dobrou"}, effects: []effect{earn(40, 90)}},
					{weight: 50, texts: []string{"O fiscal achou um cabelo no molho e aplicou multa"}, effects: []effect{lose(3, 7)}},
				}},
				{label: "🐀 Esconder o rato no freezer", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"O fiscal não abriu o freezer e foi embora satisfeito. O rato também"}},
					{weight: 45, texts: []string{"O fiscal abriu o freezer e o rato pulou no peito dele. Interditaram a cozinha"}, effects: []effect{lose(5, 10), delay(30 * time.Minute)}},
				}},
				{label: "🍽️ Oferecer um prato caprichado", risk: highRisk, endings: []ending{
					{weight: 55, texts: []string{"O fiscal comeu, repetiu e esqueceu de multar. A caneta ficou na bolsa"}, effects: []effect{stain()}},
					{weight: 30, texts: []string{"Suborno com comida também é suborno, explicou o fiscal, com a boca cheia"}, effects: []effect{lose(8, 15)}},
					{weight: 15, texts: []string{"O fiscal chamou a polícia antes da sobremesa"}, effects: []effect{nabbed()}},
				}},
				{label: "🧽 Mostrar a cozinha brilhando", risk: lowRisk, needs: holds("kit-de-limpeza"), endings: []ending{
					{weight: 100, texts: []string{"{@} passou o kit de limpeza na frente do fiscal, que ficou tão impressionado que pediu o contato da faxina"}, effects: []effect{earn(60, 120)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} decidia, o próprio fiscal achou o rato"}, effects: []effect{lose(3, 7)}},
		},
		{
			id:    "trampo-comida-critico",
			title: "O crítico disfarçado",
			areas: []int{areaComida},
			intros: []string{
				"Tem um cliente comendo sozinho, anotando tudo num caderninho e cheirando o garfo. É crítico gastronômico disfarçado, certeza",
				"O garçom correu pra avisar {@}: tem um crítico de guia gastronômico na mesa 7, de bigode falso",
			},
			choices: []choice{
				{label: "👨‍🍳 Caprichar no prato", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O crítico deu nota 9 e elogiou o tempero. A casa encheu e o patrão deu um bônus pra {@}"}, effects: []effect{earn(60, 140)}},
					{weight: 30, texts: []string{"O crítico achou \"correto, porém sem alma\". Ninguém sabe o que isso quer dizer"}},
				}},
				{label: "🧂 Usar o ingrediente secreto da vó", risk: midRisk, endings: []ending{
					{weight: 45, texts: []string{"O crítico chorou na mesa. Estrela no guia e fila na porta por um mês"}, effects: []effect{earn(200, 350)}},
					{weight: 35, texts: []string{"O ingrediente secreto da vó era pimenta demais. O crítico saiu de ambulância"}, effects: []effect{lose(5, 10)}},
					{weight: 20, texts: []string{"O prato ficou tão bom que a vó mandou uma bênção lá de cima"}, effects: []effect{enchant("prosperidade")}},
				}},
				{label: "🍲 Servir o prato do dia", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} mandou o prato do dia, igualzinho ao de todo dia. O crítico escreveu \"honesto\" e foi embora"}},
				}},
				{label: "🔥 Fazer show na chapa", risk: midRisk, needs: isJob("chef", "churrasqueiro", "sushiman", "pizzaiolo"), endings: []ending{
					{weight: 70, texts: []string{"{@} flambou, girou a faca e jogou a massa pro alto. O crítico aplaudiu de pé"}, effects: []effect{earn(150, 250)}},
					{weight: 30, texts: []string{"O fogo subiu mais que o previsto e levou a sobrancelha de {@}"}, effects: []effect{injure(5, 15)}},
				}},
			},
			idle: ending{texts: []string{"O crítico cansou de esperar o prato e foi embora escrevendo \"atendimento lento\""}, effects: []effect{lose(2, 4)}},
		},
		{
			id:    "trampo-comida-rato-chef",
			title: "O rato chef",
			areas: []int{areaComida},
			intros: []string{
				"Tem um rato na cozinha. Mas não é um rato qualquer: ele tá em cima da panela, mexendo o molho com o rabo, e o molho tá cheirando muito bem",
				"{@} pegou um rato no flagra provando o tempero da panela. E ele fez cara de que tava faltando sal",
			},
			choices: []choice{
				{label: "🧹 Caçar o rato", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"Depois de uma perseguição digna de desenho animado, {@} pegou o bicho"}, effects: []effect{gain(1, "rato")}},
					{weight: 40, texts: []string{"{@} escorregou no óleo atrás do rato e foi parar dentro da lixeira"}, effects: []effect{injure(5, 15)}},
				}},
				{label: "🐭 Deixar ele ajudar", risk: midRisk, endings: []ending{
					{weight: 40, texts: []string{"O molho do rato ficou divino. Os clientes pediram a receita e o patrão deu um bônus"}, effects: []effect{earn(150, 250)}},
					{weight: 40, texts: []string{"Um cliente viu o rato de touca de chef e gravou tudo"}, effects: []effect{lose(5, 10), delay(20 * time.Minute)}},
					{weight: 20, texts: []string{"O rato virou o mascote da cozinha e passou a trazer sorte"}, effects: []effect{enchant("sorte")}},
				}},
				{label: "☎️ Chamar a dedetização", risk: lowRisk, price: cost{percent: 3, floor: 40}, endings: []ending{
					{weight: 100, texts: []string{"A dedetizadora deixou tudo brilhando, e o patrão reembolsou com um extra pela iniciativa"}, effects: []effect{delay(10 * time.Minute), earn(40, 80)}},
				}},
				{label: "🙈 Fingir que não viu", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} virou de costas e deixou o rato terminar o molho. O almoço saiu normal"}},
					{weight: 30, texts: []string{"O rato achou uma moedinha atrás do fogão e deixou de presente no avental de {@}"}, effects: []effect{earn(10, 30)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} decidia, o rato terminou o molho, lavou a panela e foi embora"}},
		},
		{
			id:    "trampo-comida-furar-fila",
			title: "Gorjeta pra furar a fila",
			areas: []int{areaComida},
			intros: []string{
				"A fila tá dobrando o quarteirão e um cliente de terno chegou balançando uma nota de cem pra {@}: \"Me serve primeiro que essa é sua\"",
				"Um cliente apressado ofereceu uma gorjeta gorda pra {@} passar o pedido dele na frente de todo mundo",
			},
			choices: []choice{
				{label: "💵 Aceitar e servir ele primeiro", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"Ninguém percebeu, e a nota de cem foi direto pro bolso de {@}"}, effects: []effect{earn(80, 150)}},
					{weight: 40, texts: []string{"A fila percebeu, e uma senhora acertou {@} com a bolsa. Mas a nota ficou"}, effects: []effect{earn(80, 150), injure(5, 15)}},
				}},
				{label: "🙅 Recusar na frente de todo mundo", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"A fila aplaudiu {@}, e uma cliente deixou um pudim de agradecimento"}, effects: []effect{gain(1, "pudim")}},
					{weight: 40, texts: []string{"O cliente de terno foi embora bufando, e a vida seguiu"}},
				}},
				{label: "🤥 Pegar a gorjeta e não furar", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} pegou o dinheiro e serviu o cliente no fim da fila mesmo. Ele nem entendeu"}, effects: []effect{earn(80, 150), stain()}},
					{weight: 50, texts: []string{"O cliente de terno era advogado e fez um escândalo com direito a processo"}, effects: []effect{lose(4, 8)}},
				}},
			},
			idle: ending{texts: []string{"O cliente de terno cansou e entrou na fila igual todo mundo"}},
		},
		{
			id:    "trampo-comida-fogo",
			title: "Fogo na cozinha",
			areas: []int{areaComida},
			intros: []string{
				"A frigideira de {@} pegou fogo e as chamas já tão lambendo o exaustor",
				"Um respingo de óleo caiu no fogão e, de repente, a cozinha de {@} virou churrasqueira",
			},
			choices: []choice{
				{label: "💧 Jogar água", risk: highRisk, endings: []ending{
					{weight: 55, texts: []string{"Água no óleo quente: a bola de fogo subiu até o teto e chamuscou {@}"}, effects: []effect{injure(20, 40)}},
					{weight: 45, texts: []string{"Por milagre, apagou. Ninguém sabe como"}},
				}},
				{label: "🧣 Abafar com pano molhado", risk: lowRisk, endings: []ending{
					{weight: 85, texts: []string{"{@} abafou o fogo com o pano e a cozinha inteira aplaudiu"}, effects: []effect{earn(20, 50)}},
					{weight: 15, texts: []string{"O pano tava meio seco e chamuscou a mão de {@}"}, effects: []effect{injure(3, 10)}},
				}},
				{label: "🏃 Sair correndo e gritar", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"Os bombeiros vieram, apagaram e interditaram a cozinha pelo resto do turno"}, effects: []effect{delay(30 * time.Minute)}},
				}},
				{label: "🍳 Tampar com a panela de ferro", risk: lowRisk, needs: holds("panela-de-ferro"), endings: []ending{
					{weight: 100, texts: []string{"{@} tampou o fogo com a panela de ferro num movimento só, e ainda salvou o bife"}, effects: []effect{earn(40, 80)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, o alarme de incêndio disparou e molhou a cozinha inteira"}, effects: []effect{delay(20 * time.Minute)}},
		},
		{
			id:    "trampo-comida-sobras",
			title: "As sobras do dia",
			areas: []int{areaComida},
			intros: []string{
				"Fim do expediente e sobrou comida pra caramba: coxinha, marmita, pastel e meio pudim. O patrão disse que vai tudo pro lixo",
				"{@} tá olhando pra uma montanha de comida que vai pro lixo no fim do turno",
			},
			choices: []choice{
				{label: "🥡 Levar pra casa", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} encheu a mochila e jantou bem por três dias"}, effects: []effect{gain(2, "marmita", "coxinha", "pastel")}},
				}},
				{label: "🙏 Doar pra quem precisa", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} distribuiu tudo na praça. Uma senhora rezou um terço inteiro por você"}, effects: []effect{enchant("fartura")}},
				}},
				{label: "💰 Vender por fora", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} vendeu as sobras na saída do metrô e fez um extra"}, effects: []effect{earn(40, 90)}},
					{weight: 40, texts: []string{"O patrão viu tudo pelo story e descontou do salário"}, effects: []effect{lose(3, 6)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, o caminhão do lixo levou tudo"}},
		},
		{
			id:    "trampo-escritorio-hora-extra",
			title: "Hora extra pelo time",
			areas: []int{areaEscritorio},
			intros: []string{
				"Às 17h59 o chefe chegou na mesa de {@}: \"Preciso de você hoje até umas 22h. É pelo time, né?\"",
				"O gerente mandou no grupo da firma: \"Quem ficar até mais tarde hoje mostra que veste a camisa\". E marcou {@}",
			},
			choices: []choice{
				{label: "😮‍💨 Ficar até tarde", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O chefe lembrou de {@} na hora do bônus. Milagre corporativo"}, effects: []effect{earn(100, 200)}},
					{weight: 50, texts: []string{"{@} ficou até meia-noite e ganhou um \"valeu, campeão\" e uma pizza fria"}, effects: []effect{injure(3, 8), gain(1, "pizza")}},
				}},
				{label: "📅 Inventar um compromisso", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"\"Consulta médica\", disse {@}, e foi direto pro boteco"}},
					{weight: 30, texts: []string{"O chefe viu {@} no boteco pelo story e marcou uma reunião de feedback"}, effects: []effect{lose(2, 4)}},
				}},
				{label: "🕐 Negociar banco de horas", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} negociou folga e no dia seguinte chegou descansado, rendendo o dobro"}, effects: []effect{enchant("pressa")}},
				}},
				{label: "📄 Mostrar um atestado", risk: lowRisk, uses: "atestado", endings: []ending{
					{weight: 100, texts: []string{"{@} sacou o atestado na hora certa, foi embora mais cedo e voltou no outro turno novinho"}, effects: []effect{refresh()}},
				}},
			},
			idle: ending{texts: []string{"{@} demorou pra responder e o chefe entendeu como um sim. Até meia-noite"}, effects: []effect{injure(3, 8)}},
		},
		{
			id:    "trampo-escritorio-phishing",
			title: "O e-mail do diretor",
			areas: []int{areaEscritorio},
			intros: []string{
				"Chegou um e-mail do \"diretor-presidente\" pra {@}: ele precisa urgente de 10 cartões-presente e não pode ligar porque está \"em reunião no exterior\"",
				"{@} recebeu um e-mail da \"TI\" pedindo pra confirmar a senha num link chamado sua-empresa-login-seguro.ru",
			},
			choices: []choice{
				{label: "🎁 Comprar os cartões-presente", risk: midRisk, price: cost{percent: 5, floor: 60}, endings: []ending{
					{weight: 90, texts: []string{"Era golpe. Claro que era golpe. O diretor de verdade nem sabe o nome de {@}"}},
					{weight: 10, texts: []string{"Por incrível que pareça, era o diretor mesmo. Ele reembolsou com juros e decorou o nome de {@}"}, effects: []effect{earn(150, 300)}},
				}},
				{label: "🛡️ Encaminhar pro TI", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O TI agradeceu e {@} ganhou um bônus de segurança da informação"}, effects: []effect{earn(40, 80)}},
					{weight: 30, texts: []string{"O TI agradeceu com um treinamento obrigatório de 4 horas sobre golpes"}, effects: []effect{delay(30 * time.Minute)}},
				}},
				{label: "🖱️ Clicar no link", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"Vírus no computador da firma e na conta do banco de {@} ao mesmo tempo"}, effects: []effect{lose(5, 12)}},
					{weight: 30, texts: []string{"Nada aconteceu. Por enquanto"}},
					{weight: 25, texts: []string{"O vírus fritou o computador e levou o teclado junto"}, effects: []effect{breakTool(), delay(20 * time.Minute)}},
				}},
				{label: "😏 Responder trollando o golpista", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} enrolou o golpista por 40 e-mails e o print viralizou no LinkedIn"}, effects: []effect{earn(20, 60)}},
					{weight: 40, texts: []string{"O golpista não achou graça e agora manda spam todo dia"}},
				}},
			},
			idle: ending{texts: []string{"{@} deixou o e-mail sem resposta e um estagiário clicou no link por todo mundo"}},
		},
		{
			id:    "trampo-escritorio-reuniao",
			title: "A reunião que podia ser um e-mail",
			areas: []int{areaEscritorio},
			intros: []string{
				"Reunião de alinhamento de três horas sobre a pauta da próxima reunião. {@} já tá no slide 47 de 212",
				"O gerente chamou todo mundo pra uma call \"rapidinha\" de duas horas. {@} perdeu a vontade de viver no minuto 10",
			},
			choices: []choice{
				{label: "👂 Prestar atenção de verdade", risk: lowRisk, endings: []ending{
					{weight: 40, texts: []string{"{@} foi a única pessoa que entendeu a pauta e virou dona do projeto. Com aumento"}, effects: []effect{earn(60, 120)}},
					{weight: 60, texts: []string{"{@} prestou atenção e descobriu que a reunião não tinha pauta nenhuma"}, effects: []effect{delay(10 * time.Minute)}},
				}},
				{label: "📴 Fingir que a internet caiu", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"\"Tá travando aqui\", disse {@}, e saiu da call pra trabalhar em paz"}},
					{weight: 30, texts: []string{"O gerente ligou no celular pra perguntar se a internet já tinha voltado"}, effects: []effect{delay(15 * time.Minute)}},
				}},
				{label: "😴 Cochilar de câmera desligada", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"Um cochilo restaurador de 45 minutos, de fone de ouvido e tudo"}, effects: []effect{heal(10)}},
					{weight: 40, texts: []string{"Chamaram {@} pra falar e o microfone captou um ronco de motor de caminhão"}, effects: []effect{lose(2, 5)}},
				}},
				{label: "🔚 Encerrar a reunião", risk: lowRisk, needs: isJob("gerente"), endings: []ending{
					{weight: 100, texts: []string{"{@} usou o poder de gerente: \"Isso podia ser um e-mail\". A equipe trabalhou tão feliz que bateu a meta"}, effects: []effect{earn(40, 80)}},
				}},
				{label: "🌀 Viajar na reunião", risk: midRisk, needs: stoned, endings: []ending{
					{weight: 60, texts: []string{"Sob efeito, {@} achou a reunião fascinante, fez 30 perguntas e foi elogiado pelo engajamento"}, effects: []effect{earn(60, 120)}},
					{weight: 40, texts: []string{"{@} riu sozinho do gráfico de pizza por dez minutos e o RH foi chamado"}, effects: []effect{lose(3, 6)}},
				}},
			},
			idle: ending{texts: []string{"{@} ficou parado tanto tempo que a reunião acabou sozinha. Igual sempre"}},
		},
		{
			id:    "trampo-escritorio-credito",
			title: "O colega levou o crédito",
			areas: []int{areaEscritorio},
			intros: []string{
				"Na reunião com a diretoria, o colega apresentou a planilha que {@} fez a semana inteira. E falou \"meu projeto\" onze vezes",
				"O colega da baia do lado mandou o relatório de {@} pro chefe com o próprio nome na capa",
			},
			choices: []choice{
				{label: "📢 Expor na frente da diretoria", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} mostrou o histórico do arquivo com data e hora. O colega ficou roxo e a promoção veio pra você"}, effects: []effect{earn(120, 220)}},
					{weight: 50, texts: []string{"A diretoria achou feio brigar em público e os dois levaram bronca"}, effects: []effect{enchant("olhogordo")}},
				}},
				{label: "😈 Sabotar o próximo dele", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"{@} trocou uma vírgula na fórmula da planilha dele. O colega apresentou um prejuízo de 3 bilhões e você ficou com a vaga"}, effects: []effect{earn(80, 160)}},
					{weight: 35, texts: []string{"A TI rastreou a alteração até o computador de {@}"}, effects: []effect{lose(5, 10)}},
					{weight: 20, texts: []string{"O tiro saiu pela culatra e a planilha errada era a sua"}, effects: []effect{enchant("azar")}},
				}},
				{label: "🧘 Deixar o karma resolver", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{
						"Na semana seguinte, o colega apresentou outra coisa sozinho e passou vergonha. O karma é pontual",
						"O karma tá demorando, mas {@} segue confiante",
					}},
				}},
				{label: "😭 Chorar no banheiro", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} chorou no banheiro, lavou o rosto e voltou se sentindo mais leve"}, effects: []effect{heal(10)}},
				}},
			},
			idle: ending{texts: []string{"{@} ficou quieto, e o colega já tá apresentando o próximo trabalho seu também"}},
		},
		{
			id:    "trampo-escritorio-pendrive",
			title: "O pendrive misterioso",
			areas: []int{areaEscritorio},
			intros: []string{
				"{@} achou um pendrive no estacionamento da firma escrito \"NÃO ABRIR, CONFIDENCIAL\" com caneta vermelha",
				"Tinha um pendrive esquecido na impressora do andar da diretoria. A etiqueta diz \"planilha secreta\"",
			},
			choices: []choice{
				{label: "💻 Plugar no computador da firma", risk: highRisk, endings: []ending{
					{weight: 35, texts: []string{"Abriu uma planilha chamada \"propinas\", com o apelido do chefe em todas as linhas"}, next: "trampo-escritorio-pendrive-propina"},
					{weight: 35, texts: []string{"Era vírus. O computador começou a tocar funk sozinho e travou de vez"}, effects: []effect{breakTool(), delay(20 * time.Minute)}},
					{weight: 30, texts: []string{"Só tinha foto do churrasco da firma de 2012"}, effects: []effect{gain(1, "pendrive")}},
				}},
				{label: "🛡️ Entregar pro TI", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"O TI agradeceu, formatou o pendrive e deu de presente pra {@}"}, effects: []effect{gain(1, "pendrive")}},
				}},
				{label: "🗑️ Jogar no lixo", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} jogou o pendrive no lixo e dormiu tranquilo"}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, a faxineira levou o pendrive embora"}},
		},
		{
			id:    "trampo-escritorio-pendrive-propina",
			title: "A planilha da propina",
			stage: true,
			intros: []string{
				"A planilha tem tudo: valores, datas e o apelido do chefe, \"Chefinho\", em todas as linhas. {@} tá olhando pra tela com o coração disparado",
				"{@} acabou de descobrir que o chefe desvia dinheiro da firma desde 2019, numa planilha com gráfico e tudo",
			},
			choices: []choice{
				{label: "🤫 Chantagear o chefe", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"O chefe pagou pelo silêncio de {@} sem nem discutir"}, effects: []effect{earn(250, 450), stain()}},
					{weight: 30, texts: []string{"O chefe tinha amigos na polícia"}, effects: []effect{nabbed()}},
					{weight: 20, texts: []string{"O chefe tinha amigos de outro tipo. {@} levou um susto no estacionamento"}, effects: []effect{injure(20, 40)}},
				}},
				{label: "📨 Denunciar pro compliance", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"O chefe foi demitido e {@} ganhou o cargo dele, com bônus de integridade"}, effects: []effect{earn(150, 300)}},
					{weight: 40, texts: []string{"O compliance era cunhado do chefe. {@} foi transferido pro almoxarifado"}, effects: []effect{lose(3, 6), delay(40 * time.Minute)}},
				}},
				{label: "❌ Apagar tudo e esquecer", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} apagou o arquivo, formatou o pendrive e decidiu que nunca viu nada"}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} decidia, o chefe passou atrás da cadeira e viu a tela"}, effects: []effect{enchant("olhogordo")}},
		},
		{
			id:     "trampo-escritorio-zero-a-mais",
			title:  "O zero a mais",
			rarity: rareEvent,
			areas:  []int{areaEscritorio},
			intros: []string{
				"{@} digitou um zero a mais numa transferência da firma e caiu 1 milhão na conta errada. Que, por acaso, é a sua",
				"Erro no sistema da folha: o salário de {@} veio com três zeros a mais e o RH ainda não percebeu",
			},
			choices: []choice{
				{label: "↩️ Devolver na hora", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"O RH devolveu o dinheiro pro lugar e deu um prêmio de honestidade pra {@}"}, effects: []effect{earn(100, 200)}},
				}},
				{label: "💸 Sacar e sumir", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} sacou o que deu, comprou uma passagem só de ida e voltou uma semana depois como se nada tivesse acontecido"}, effects: []effect{earn(400, 700), stain()}},
					{weight: 35, texts: []string{"O banco bloqueou tudo na hora e chamou a polícia"}, effects: []effect{nabbed(), lose(5, 10)}},
					{weight: 15, texts: []string{"O dinheiro era de um cliente muito, muito perigoso"}, effects: []effect{injure(30, 50)}},
				}},
				{label: "📈 Investir rapidinho e devolver", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} aplicou por um dia, devolveu o principal e embolsou o rendimento"}, effects: []effect{earn(300, 500)}},
					{weight: 50, texts: []string{"A bolsa caiu justo nesse dia e {@} teve que completar a diferença do próprio bolso"}, effects: []effect{lose(8, 15)}},
				}},
				{label: "📊 Sumir com o rastro na planilha", risk: highRisk, needs: isJob("contador", "trader", "economista", "analista de dados"), endings: []ending{
					{weight: 70, texts: []string{"{@} fez uma mágica contábil e o dinheiro sumiu dos registros. Ninguém nunca vai achar"}, effects: []effect{earn(400, 700), stain()}},
					{weight: 30, texts: []string{"A auditoria achou a mágica contábil"}, effects: []effect{nabbed(), lose(10, 20)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, o RH percebeu o erro e estornou tudo"}},
		},
		{
			id:    "trampo-saude-receita",
			title: "Receita por fora",
			areas: []int{areaSaude},
			intros: []string{
				"Um paciente fechou a porta e cochichou pra {@}: \"Me arruma uma receitinha daquele remédio tarja preta? Pago bem\"",
				"Um cliente de óculos escuros pediu pra {@} liberar um remédio controlado sem receita. Ele já tá com o dinheiro na mão",
			},
			choices: []choice{
				{label: "🙅 Recusar", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} recusou e o paciente saiu resmungando que ia procurar outro"}},
					{weight: 30, texts: []string{"O paciente se vingou com uma avaliação de uma estrela no Google"}, effects: []effect{lose(1, 3)}},
				}},
				{label: "💊 Cobrar caro e liberar", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"O dinheiro veio em espécie, num saquinho de padaria"}, effects: []effect{earn(150, 280), stain()}},
					{weight: 30, texts: []string{"O paciente era fiscal do conselho"}, effects: []effect{nabbed()}},
					{weight: 20, texts: []string{"O paciente voltou dizendo que o remédio \"não bateu\" e exigindo o dinheiro de volta, com juros"}, effects: []effect{lose(5, 10)}},
				}},
				{label: "🍬 Passar placebo e cobrar", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} vendeu pílula de farinha, e o paciente voltou dizendo que foi o melhor remédio da vida"}, effects: []effect{earn(80, 150)}},
					{weight: 40, texts: []string{"O paciente mandou analisar o comprimido"}, effects: []effect{lose(3, 6)}},
				}},
			},
			idle: ending{texts: []string{"O paciente cansou de esperar e foi tentar a sorte na farmácia da esquina"}},
		},
		{
			id:    "trampo-saude-plantao",
			title: "Plantão de 36 horas",
			areas: []int{areaSaude},
			intros: []string{
				"{@} tá na 33ª hora de plantão, já chamou o próprio reflexo pelo nome e tomou café frio de três dias atrás",
				"O plantão de {@} começou ontem de manhã e o substituto mandou mensagem: \"vou atrasar umas 6 horinhas\"",
			},
			choices: []choice{
				{label: "☕ Tomar mais um café", risk: lowRisk, uses: "cafe", endings: []ending{
					{weight: 100, texts: []string{"O café desceu queimando e {@} acordou tão ligado que deu conta de tudo e voltou pro batente"}, effects: []effect{refresh()}},
				}},
				{label: "🛏️ Cochilar na maca", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"Um cochilo de 20 minutos que valeu por 8 horas de sono"}, effects: []effect{heal(15)}},
					{weight: 40, texts: []string{"{@} acordou com um paciente aferindo a sua pressão, e a chefia viu"}, effects: []effect{lose(2, 4)}},
				}},
				{label: "😤 Aguentar no ódio", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} aguentou firme, salvou todo mundo e ganhou plantão extra pago"}, effects: []effect{earn(100, 180)}},
					{weight: 50, texts: []string{"{@} desmaiou no corredor e virou paciente do próprio hospital"}, effects: []effect{injure(10, 20)}},
				}},
				{label: "🏠 Pedir pra ir embora", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"A chefia liberou, mas vai descontar as horas. Pelo menos {@} dormiu"}, effects: []effect{heal(10), delay(30 * time.Minute)}},
				}},
				{label: "🩹 Pedir pra um colega te examinar", risk: lowRisk, needs: bruised, endings: []ending{
					{weight: 100, texts: []string{"O colega de plantão cuidou de {@} entre um paciente e outro, de graça"}, effects: []effect{heal(25)}},
				}},
			},
			idle: ending{texts: []string{"{@} dormiu em pé no meio da escolha"}, effects: []effect{heal(5)}},
		},
		{
			id:    "trampo-saude-prontuario",
			title: "O prontuário trocado",
			areas: []int{areaSaude},
			intros: []string{
				"{@} percebeu que trocou as fichas de dois pacientes: o do joelho ia fazer exame de próstata, e o da próstata ia pra fisioterapia de joelho",
				"Erro no sistema: os exames do Seu Valdir foram parar na ficha da Dona Cleusa. E quem digitou foi {@}",
			},
			choices: []choice{
				{label: "🗣️ Contar pro chefe", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"Corrigiram a tempo e o chefe elogiou a honestidade de {@}"}, effects: []effect{earn(20, 50)}},
					{weight: 30, texts: []string{"Corrigiram a tempo, mas {@} levou uma advertência"}, effects: []effect{delay(20 * time.Minute)}},
				}},
				{label: "🤫 Corrigir escondido", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"Ninguém percebeu nada e os dois pacientes saíram felizes"}},
					{weight: 40, texts: []string{"O Seu Valdir fez um exame que não precisava e processou"}, effects: []effect{lose(5, 10)}},
				}},
				{label: "👉 Culpar o residente", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O residente levou a culpa e {@} ficou com o karma pesado"}, effects: []effect{enchant("azar")}},
					{weight: 50, texts: []string{"O residente tinha print de tudo"}, effects: []effect{lose(4, 8)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, a Dona Cleusa chegou na fisioterapia de joelho sem entender nada"}, effects: []effect{lose(2, 4)}},
		},
		{
			id:     "trampo-saude-paciente-vip",
			title:  "O paciente milionário",
			rarity: rareEvent,
			areas:  []int{areaSaude},
			intros: []string{
				"Um empresário milionário saiu do atendimento encantado com {@} e ofereceu uma fortuna por atendimento exclusivo, 24 horas por dia",
				"O paciente da sala VIP mandou o motorista entregar um envelope pra {@} com uma proposta: cuidar só da família dele",
			},
			choices: []choice{
				{label: "🤝 Aceitar o contrato", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"O primeiro pagamento já veio maior que um mês de trabalho"}, effects: []effect{earn(300, 500)}},
					{weight: 40, texts: []string{"Era hipocondríaco e ligou 47 vezes no primeiro dia"}, effects: []effect{earn(100, 200), delay(60 * time.Minute)}},
				}},
				{label: "🙅 Recusar com educação", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"O milionário respeitou e mandou um presente de agradecimento"}, effects: []effect{gain(1, "caixa-misteriosa")}},
				}},
				{label: "📈 Inflar a conta", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} cobrou até a caneta e o milionário nem piscou"}, effects: []effect{earn(400, 700), stain()}},
					{weight: 30, texts: []string{"O contador do milionário achou a conta inflada"}, effects: []effect{nabbed()}},
					{weight: 20, texts: []string{"O advogado dele chegou antes da polícia"}, effects: []effect{lose(8, 15)}},
				}},
			},
			idle: ending{texts: []string{"O milionário cansou de esperar e contratou outra pessoa"}},
		},
		{
			id:    "trampo-saude-paciente-bravo",
			title: "O paciente que morde",
			areas: []int{areaSaude},
			intros: []string{
				"O paciente de hoje tá bravo, babando e mostrando os dentes. Pode ser um cachorro, pode ser uma criança de 6 anos com medo de injeção. Dá no mesmo",
				"{@} encostou no paciente e ele rosnou. Literalmente",
			},
			choices: []choice{
				{label: "💪 Segurar firme", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} terminou o atendimento com uma mão e segurou o paciente com a outra"}, effects: []effect{earn(60, 120)}},
					{weight: 40, texts: []string{"Levou uma mordida de respeito no antebraço"}, effects: []effect{injure(10, 25)}},
				}},
				{label: "🆘 Chamar reforço", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"Vieram três colegas segurar o paciente, e o atendimento demorou o triplo"}, effects: []effect{delay(15 * time.Minute)}},
				}},
				{label: "💉 Sedar", risk: midRisk, endings: []ending{
					{weight: 70, texts: []string{"Uma picadinha e o paciente virou um anjo"}, effects: []effect{earn(40, 80)}},
					{weight: 30, texts: []string{"Na confusão, a agulha entrou em {@}, que dormiu na maca do lado"}, effects: []effect{heal(10), delay(30 * time.Minute)}},
				}},
				{label: "🍭 Oferecer um pirulito", risk: lowRisk, uses: "pirulito", endings: []ending{
					{weight: 100, texts: []string{"O pirulito acalmou o paciente na hora. Funciona com criança, cachorro e marmanjo"}, effects: []effect{earn(40, 80)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, o paciente fugiu pela janela"}},
		},
		{
			id:    "trampo-saude-acidente",
			title: "Acidente no caminho",
			areas: []int{areaSaude},
			intros: []string{
				"No caminho do trampo, {@} viu um motoqueiro caído no meio da rua, gemendo, e um monte de gente só filmando",
				"Um ciclista se estabacou na frente de {@} e tá sentado no meio-fio com o joelho em carne viva",
			},
			choices: []choice{
				{label: "🩺 Prestar socorro", risk: midRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} imobilizou, acalmou e esperou a ambulância. O acidentado anotou seu nome pra agradecer depois"}, effects: []effect{plant("trampo-saude-gratidao", 8*time.Hour)}},
					{weight: 30, texts: []string{"O acidentado tava agitado e acertou um soco em {@} sem querer"}, effects: []effect{injure(5, 15)}},
				}},
				{label: "🏃 Seguir pro trampo", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"Alguém da multidão fez o socorro, e {@} chegou no horário"}},
				}},
				{label: "📱 Filmar pra postar", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O vídeo viralizou e rendeu umas moedinhas de monetização"}, effects: []effect{earn(30, 80)}},
					{weight: 50, texts: []string{"A internet cancelou {@} por filmar em vez de ajudar"}, effects: []effect{enchant("olhogordo")}},
				}},
				{label: "🚑 Usar o kit de primeiros socorros", risk: lowRisk, uses: "kit-de-primeiros-socorros", endings: []ending{
					{weight: 100, texts: []string{"{@} fez um atendimento digno de série médica. A família pagou um jantar e mais um pouco"}, effects: []effect{earn(100, 200), plant("trampo-saude-gratidao", 8*time.Hour)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} decidia, a ambulância chegou e resolveu tudo"}},
		},
		{
			id:    "trampo-saude-gratidao",
			title: "O acidentado agradecido",
			stage: true,
			intros: []string{
				"Lembra do acidentado que {@} socorreu? Ele apareceu na porta do trampo, de muleta e com uma caixa de presente embaixo do braço",
				"O acidentado que {@} ajudou voltou, já recuperado, querendo retribuir de algum jeito",
			},
			choices: []choice{
				{label: "🎁 Abrir o presente", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"Dentro da caixa tinha outra caixa. E dentro dela, uma surpresa"}, effects: []effect{gain(1, "caixa-misteriosa")}},
				}},
				{label: "🙏 Dispensar o presente", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} dispensou, e ele jurou acender uma vela todo dia em sua homenagem"}, effects: []effect{enchant("protecao")}},
				}},
				{label: "💵 Pedir dinheiro em vez do presente", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"Ele mandou um Pix generoso, rindo da sinceridade de {@}"}, effects: []effect{earn(100, 200)}},
					{weight: 40, texts: []string{"Ele ficou ofendido e levou o presente embora"}},
				}},
			},
			idle: ending{texts: []string{"Ele esperou, esperou e foi embora achando que {@} nem lembrava mais dele"}},
		},
		{
			id:    "trampo-arte-briga-no-show",
			title: "Briga no show",
			areas: []int{areaArte},
			intros: []string{
				"No meio da apresentação de {@}, começou uma briga generalizada na plateia. Voou cadeira, chinelo e um espetinho de frango",
				"Dois fãs começaram a se estapear bem na frente do palco de {@}, e a segurança tá comendo cachorro-quente lá fora",
			},
			choices: []choice{
				{label: "🕊️ Parar tudo e pedir paz", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} pediu paz no microfone, a briga parou e a plateia aplaudiu de pé"}, effects: []effect{earn(40, 90)}},
					{weight: 30, texts: []string{"A briga parou, mas o contratante descontou o tempo parado do cachê"}, effects: []effect{lose(2, 4)}},
				}},
				{label: "🔊 Aumentar o som e seguir", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"O som abafou a briga e a apresentação virou lendária na cidade"}, effects: []effect{earn(80, 160)}},
					{weight: 40, texts: []string{"Uma cadeira voou direto no palco"}, effects: []effect{injure(10, 25)}},
				}},
				{label: "🥊 Descer e entrar na briga", risk: highRisk, endings: []ending{
					{weight: 40, favor: func(p Profession) float64 { return p.guard }, texts: []string{"{@} separou a briga no braço e virou herói da noite"}, effects: []effect{earn(100, 200)}},
					{weight: 45, texts: []string{"Apanhou dos dois lados ao mesmo tempo"}, effects: []effect{injure(20, 40)}},
					{weight: 15, texts: []string{"A polícia chegou e levou todo mundo, inclusive a atração da noite"}, effects: []effect{nabbed()}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} decidia, a briga subiu no palco e derrubou o equipamento"}, effects: []effect{breakTool()}},
		},
		{
			id:    "trampo-arte-contrato",
			title: "O contrato com cláusula estranha",
			areas: []int{areaArte},
			intros: []string{
				"Um empresário de terno brilhante ofereceu um contrato milionário pra {@}. A cláusula 47 diz \"cede todos os direitos de imagem, nome, alma e toque de celular\"",
				"Uma produtora quer {@} no elenco, mas o contrato tem 80 páginas em letra minúscula e um carimbo com formato de chifre",
			},
			choices: []choice{
				{label: "✍️ Assinar sem ler", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"O contrato era bom de verdade e o primeiro cachê já caiu"}, effects: []effect{earn(300, 500)}},
					{weight: 40, texts: []string{"A cláusula 47 era real, e agora o empresário fica com o grosso do que {@} ganha"}, effects: []effect{enchant("miseria")}},
					{weight: 20, texts: []string{"Tinha multa rescisória escondida na página 79"}, effects: []effect{lose(10, 18)}},
				}},
				{label: "🔍 Ler as letras miúdas", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} achou a pegadinha, riscou a cláusula e negociou um contrato justo"}, effects: []effect{earn(60, 120)}},
					{weight: 40, texts: []string{"{@} leu tudo e desistiu. A alma continua sua"}},
				}},
				{label: "⚖️ Contratar um advogado", risk: lowRisk, price: cost{percent: 4, floor: 60}, endings: []ending{
					{weight: 100, texts: []string{"O advogado transformou o contrato num negócio da China"}, effects: []effect{earn(150, 250)}},
				}},
				{label: "🏃 Pegar o adiantamento e sumir", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} pegou o adiantamento e trocou até de nome artístico"}, effects: []effect{earn(150, 250), stain()}},
					{weight: 50, texts: []string{"O empresário tinha capangas, e eles acharam {@} rapidinho"}, effects: []effect{injure(15, 30)}},
				}},
			},
			idle: ending{texts: []string{"O empresário achou outro artista mais desesperado"}},
		},
		{
			id:    "trampo-arte-viralizou",
			title: "Viralizou do jeito errado",
			areas: []int{areaArte},
			intros: []string{
				"O vídeo de {@} tropeçando no palco virou meme nacional. Já tem remix, figurinha e até dancinha",
				"Um corte de {@} falando besteira ao vivo tá com 3 milhões de views e a palavra \"cancelado\" nos comentários",
			},
			choices: []choice{
				{label: "😂 Abraçar o meme", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} fez piada da própria desgraça e fechou três publis de uma vez"}, effects: []effect{earn(100, 220)}},
					{weight: 40, texts: []string{"A internet achou forçado e o meme virou contra {@}"}, effects: []effect{enchant("olhogordo")}},
				}},
				{label: "🫥 Sumir da internet", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} desativou tudo por uns dias e esperou a internet esquecer"}, effects: []effect{delay(30 * time.Minute)}},
				}},
				{label: "⚖️ Processar quem postou", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O juiz mandou tirar do ar e ainda pagar indenização"}, effects: []effect{earn(80, 160)}},
					{weight: 50, texts: []string{"O processo virou meme também, e as custas ficaram com {@}"}, effects: []effect{lose(4, 8)}},
				}},
				{label: "🎵 Lançar música sobre o meme", risk: midRisk, endings: []ending{
					{weight: 40, texts: []string{"A música estourou nas plataformas e {@} virou o hit do verão"}, effects: []effect{earn(150, 300)}},
					{weight: 60, texts: []string{"A música flopou, mas pelo menos ficou boa"}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, o meme virou figurinha em todos os grupos da família"}},
		},
		{
			id:    "trampo-arte-cache-exposicao",
			title: "Pagamento em exposição",
			areas: []int{areaArte},
			intros: []string{
				"O contratante avisou que não tem cachê, mas {@} vai ganhar \"muita exposição\" e um lanche",
				"Uma marca quer {@} numa campanha. O pagamento? \"Visibilidade\" e um cupom de 10% de desconto",
			},
			choices: []choice{
				{label: "🤷 Aceitar a exposição", risk: lowRisk, endings: []ending{
					{weight: 30, texts: []string{"Por milagre, a exposição deu certo e chegaram três trabalhos pagos depois"}, effects: []effect{earn(60, 150)}},
					{weight: 70, texts: []string{"A exposição rendeu dois seguidores, e um deles é a sua mãe. O lanche era um misto frio"}},
				}},
				{label: "💵 Exigir o cachê", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O contratante achou dinheiro no fundo do caixa e pagou"}, effects: []effect{earn(60, 120)}},
					{weight: 50, texts: []string{"O contratante chamou outro artista que aceita lanche"}},
				}},
				{label: "🍔 Aceitar e atacar o buffet", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"Se é pra pagar em comida, {@} jantou por três dias e ainda levou marmita"}, effects: []effect{heal(10), gain(1, "marmita")}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, o contratante achou outro artista e o lanche acabou"}},
		},
		{
			id:    "trampo-arte-equipamento",
			title: "O equipamento pifou",
			areas: []int{areaArte},
			intros: []string{
				"No meio da apresentação, o equipamento de {@} soltou fumaça, fez um barulho de panela de pressão e apagou",
				"A luz caiu bem no meio da apresentação de {@} e o público tá começando a vaiar",
			},
			choices: []choice{
				{label: "🎭 Improvisar no gogó", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} fez a melhor apresentação da vida no improviso, à luz das lanternas dos celulares"}, effects: []effect{earn(80, 150)}},
					{weight: 40, texts: []string{"O improviso não pegou e choveu vaia"}, effects: []effect{lose(1, 3)}},
				}},
				{label: "🔧 Consertar na hora", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Uma fita isolante e muita fé resolveram"}, effects: []effect{earn(20, 50)}},
					{weight: 30, texts: []string{"Levou um choque que acendeu até a alma"}, effects: []effect{injure(10, 20)}},
					{weight: 20, texts: []string{"Terminou de quebrar de vez"}, effects: []effect{breakTool()}},
				}},
				{label: "🎟️ Cancelar e devolver o ingresso", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} devolveu o dinheiro de todo mundo e saiu com a reputação intacta"}, effects: []effect{lose(2, 4)}},
				}},
				{label: "💡 Ligar a ring light reserva", risk: lowRisk, needs: holds("ring-light"), endings: []ending{
					{weight: 100, texts: []string{"A ring light salvou a noite e virou luz de palco. Ficou até mais bonito"}, effects: []effect{earn(60, 120)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, o público foi embora pra outro show"}, effects: []effect{lose(1, 3)}},
		},
		{
			id:     "trampo-arte-olheiro",
			title:  "O olheiro da TV",
			rarity: rareEvent,
			areas:  []int{areaArte},
			intros: []string{
				"No fim da apresentação, um senhor de crachá de emissora de TV procurou {@}: \"Tô montando o elenco de um programa novo. Topa um teste?\"",
				"Um olheiro de gravadora assistiu a tudo da primeira fila e deixou um cartão pra {@} com o recado: \"Me liga\"",
			},
			choices: []choice{
				{label: "🎬 Fazer o teste na hora", risk: midRisk, endings: []ending{
					{weight: 40, texts: []string{"{@} arrasou no teste improvisado e assinou na hora"}, effects: []effect{earn(300, 600), enchant("prosperidade")}},
					{weight: 60, texts: []string{"{@} travou na frente da câmera e esqueceu até o próprio nome"}},
				}},
				{label: "📇 Pegar o contato e se preparar", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} guardou o cartão e vai treinar a semana inteira pro teste"}, effects: []effect{plant("trampo-arte-teste-tv", 6*time.Hour)}},
				}},
				{label: "😎 Fingir que já é famoso", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O olheiro caiu na pose e ofereceu o dobro do cachê"}, effects: []effect{earn(150, 250)}},
					{weight: 50, texts: []string{"O olheiro pesquisou o nome de {@} na internet e achou só a foto do RG"}, effects: []effect{lose(2, 5)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, o olheiro foi falar com a dupla sertaneja do palco ao lado"}},
		},
		{
			id:    "trampo-arte-teste-tv",
			title: "O grande teste",
			stage: true,
			intros: []string{
				"Chegou o dia do teste que o olheiro marcou. {@} tá no camarim, tremendo, com a maquiagem pela metade",
				"O telefone tocou: é a produção daquele olheiro. O teste de {@} é agora, no estúdio 3",
			},
			choices: []choice{
				{label: "🙏 Ir com a cara e a coragem", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} entrou, brilhou e saiu com contrato assinado"}, effects: []effect{earn(300, 600)}},
					{weight: 45, texts: []string{"Deu branco total na hora H"}},
				}},
				{label: "📣 Levar a família de torcida", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"A família fez tanto barulho na plateia que o diretor deu uma participação especial pra {@}"}, effects: []effect{earn(100, 200)}},
				}},
				{label: "💼 Molhar a mão do diretor", risk: midRisk, price: cost{percent: 8, floor: 200}, endings: []ending{
					{weight: 70, texts: []string{"O diretor aceitou o \"cachê de consultoria\" e {@} virou protagonista"}, effects: []effect{earn(400, 700), stain()}},
					{weight: 30, texts: []string{"O diretor embolsou o dinheiro e sumiu"}},
				}},
			},
			idle: ending{texts: []string{"{@} perdeu a hora do teste e a vaga foi pra outra pessoa"}},
		},
		{
			id:    "trampo-bico-mudanca",
			title: "A geladeira duplex",
			areas: []int{areaBicos},
			when:  wentWell,
			intros: []string{
				"O bico de hoje é carregar a mudança de uma senhora do quarto andar, sem elevador. A primeira peça é uma geladeira duplex, e {@} tá sozinho",
				"{@} topou ajudar numa mudança por uma grana rápida. Ninguém avisou que tinha uma geladeira duplex e que a escada é caracol",
			},
			choices: []choice{
				{label: "💪 Carregar nas costas", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} chegou lá embaixo bufando, e a senhora pagou um extra pela coragem"}, effects: []effect{earn(60, 120)}},
					{weight: 35, texts: []string{"A coluna de {@} travou no segundo lance de escada, junto com a geladeira"}, effects: []effect{injure(10, 25)}},
					{weight: 15, texts: []string{"A geladeira escapou e desceu a escada sozinha. Descontaram o conserto do pagamento"}, effects: []effect{lose(3, 6)}},
				}},
				{label: "🤝 Chamar os vizinhos pra ajudar", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"Os vizinhos ajudaram, e {@} dividiu a grana com todo mundo"}, effects: []effect{earn(20, 50)}},
					{weight: 30, texts: []string{"Os vizinhos só ficaram apontando de onde segurar e foram embora na hora de pegar peso"}},
				}},
				{label: "🪢 Descer pela janela na corda", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"A geladeira desceu bonitinha, a rua aplaudiu e a senhora pagou em dobro"}, effects: []effect{earn(100, 200)}},
					{weight: 43, texts: []string{"A corda arrebentou no primeiro andar e a geladeira caiu bem no pé de {@}"}, effects: []effect{injure(20, 40)}},
					{weight: 7, texts: []string{"A corda arrebentou no quarto andar, e {@} tava embaixo"}, effects: []effect{perish()}},
				}},
				{label: "🛏️ Testar o colchão da mudança", risk: lowRisk, needs: isJob("testador de colchão"), endings: []ending{
					{weight: 100, texts: []string{"{@} deitou no colchão pra conferir a qualidade e acordou duas horas depois, descansado. A senhora achou que fazia parte do serviço e pagou"}, effects: []effect{earn(30, 60), heal(15)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} decidia, a senhora desistiu e chamou um caminhão de mudança de verdade"}},
		},
		{
			id:    "trampo-obra-sindico",
			title: "O síndico da furadeira",
			areas: []int{areaObras},
			intros: []string{
				"{@} ligou a furadeira às 7 da manhã e em dois minutos o síndico tava na porta, de pijama, com o regulamento do condomínio na mão",
				"O vizinho de baixo começou a bater no teto com cabo de vassoura assim que {@} começou a quebrar a parede",
			},
			choices: []choice{
				{label: "🤫 Trabalhar bem quietinho", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} trabalhou em câmera lenta, com martelo de borracha. Demorou o dobro"}, effects: []effect{delay(15 * time.Minute)}},
					{weight: 40, texts: []string{"O cliente adorou a delicadeza e deu um extra pro cafezinho"}, effects: []effect{earn(20, 40)}},
				}},
				{label: "🔊 Ignorar e meter a marreta", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} terminou rapidinho e o cliente pagou feliz da vida"}, effects: []effect{earn(50, 100)}},
					{weight: 30, texts: []string{"A multa do condomínio veio no nome do cliente, e o cliente descontou de {@}"}, effects: []effect{lose(3, 7)}},
					{weight: 20, texts: []string{"O vizinho de baixo subiu e resolveu a questão na mão"}, effects: []effect{injure(5, 15)}},
				}},
				{label: "🍰 Levar um bolo pro síndico", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O síndico comeu meio bolo e liberou o barulho até as dez da noite"}, effects: []effect{refresh()}},
					{weight: 30, texts: []string{"O síndico era diabético e ficou mais bravo ainda"}, effects: []effect{delay(20 * time.Minute)}},
				}},
				{label: "🧰 Consertar a porta do síndico", risk: lowRisk, needs: holds("caixa-de-ferramentas"), endings: []ending{
					{weight: 100, texts: []string{"{@} abriu a caixa de ferramentas, consertou a porta emperrada do síndico de graça e virou o pedreiro oficial do prédio"}, effects: []effect{earn(60, 120)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, o síndico chamou a portaria e a obra parou até a segunda"}, effects: []effect{delay(15 * time.Minute)}},
		},
		{
			id:    "trampo-transporte-saideira",
			title: "O passageiro da saideira",
			areas: []int{areaTransporte},
			intros: []string{
				"Entrou no carro de {@} um passageiro trocando as pernas, abraçado numa garrafa e pedindo pra ir \"na casa da ex, que ela vai entender\"",
				"Três da manhã, porta do bar. O passageiro de {@} já entrou cantando Amado Batista e com cara de quem vai passar mal",
			},
			choices: []choice{
				{label: "🏠 Levar ele pra casa dele", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"A mãe dele atendeu a porta e pagou a corrida com gorjeta, de tanta vergonha"}, effects: []effect{earn(40, 80)}},
					{weight: 40, texts: []string{"Ele dormiu no banco de trás e {@} teve que esperar ele acordar pra saber o endereço"}, effects: []effect{delay(15 * time.Minute)}},
				}},
				{label: "💔 Levar na casa da ex", risk: midRisk, endings: []ending{
					{weight: 40, texts: []string{"A ex aceitou ele de volta, e o casal pagou a corrida com uma gorjeta de casamento"}, effects: []effect{earn(80, 160)}},
					{weight: 40, texts: []string{"A ex jogou um vaso de planta pela janela, e acertou {@} em vez dele"}, effects: []effect{injure(5, 15)}},
					{weight: 20, texts: []string{"No meio do caminho ele vomitou no banco inteiro, e a limpeza saiu do bolso de {@}"}, effects: []effect{lose(2, 5)}},
				}},
				{label: "🚪 Mandar ele descer", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"Ele desceu, abraçou um poste e começou a chamar o poste de amor"}},
					{weight: 30, texts: []string{"Ele desceu e deixou cair umas notas no banco. {@} achou e ficou de boca fechada"}, effects: []effect{earn(20, 50), stain()}},
				}},
				{label: "🧭 Dar uma volta com o taxímetro", risk: highRisk, needs: isJob("taxista", "motorista de aplicativo", "mototaxista"), endings: []ending{
					{weight: 60, texts: []string{"{@} deu três voltas no mesmo quarteirão com o taxímetro rodando, e ele pagou sem nem olhar"}, effects: []effect{earn(150, 250), stain()}},
					{weight: 40, texts: []string{"Ele acordou na terceira volta, reconheceu a padaria e ligou pra polícia"}, effects: []effect{nabbed()}},
				}},
			},
			idle: ending{texts: []string{"{@} demorou tanto que o passageiro dormiu no meio-fio, e outro carro levou ele"}},
		},
		{
			id:    "trampo-comida-influencer",
			title: "A influencer da permuta",
			areas: []int{areaComida},
			when:  wentWell,
			intros: []string{
				"Uma influencer de 12 mil seguidores entrou onde {@} trabalha e já foi avisando: \"faço um story se o almoço for na faixa\"",
				"{@} recebeu na cozinha a proposta de uma blogueira: rodízio completo de graça em troca de \"divulgação\"",
			},
			choices: []choice{
				{label: "🆓 Aceitar a permuta", risk: midRisk, endings: []ending{
					{weight: 45, texts: []string{"O story bombou, a casa encheu e o patrão deu um bônus pra {@}"}, effects: []effect{earn(80, 160)}},
					{weight: 35, texts: []string{"O story teve nove visualizações, três da mãe dela. O patrão descontou o almoço de {@}"}, effects: []effect{lose(2, 5)}},
					{weight: 20, texts: []string{"Ela reclamou da comida nos stories, e metade da cidade ficou de olho torto em {@}"}, effects: []effect{enchant("olhogordo")}},
				}},
				{label: "🙅 Cobrar como qualquer cliente", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"Ela pagou de cara feia e ainda deixou a gorjeta, pra não pegar mal"}, effects: []effect{earn(20, 40)}},
					{weight: 40, texts: []string{"Ela foi embora dizendo que ia expor o lugar. Até agora, nada"}},
				}},
				{label: "🌶️ Servir o prato mais ardido", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O vídeo dela chorando de pimenta viralizou e o lugar virou ponto de desafio"}, effects: []effect{earn(60, 120)}},
					{weight: 50, texts: []string{"Ela passou mal e deixou uma estrela na avaliação, com textão"}, effects: []effect{lose(3, 6)}},
				}},
				{label: "🎂 Fazer um doce pra câmera", risk: lowRisk, needs: isJob("confeiteiro", "padeiro", "chef", "barista"), endings: []ending{
					{weight: 100, texts: []string{"{@} fez um doce tão bonito que ela pagou, postou de graça e marcou o perfil da casa"}, effects: []effect{earn(120, 220)}},
				}},
			},
			idle: ending{texts: []string{"A influencer cansou de esperar e foi fazer story no concorrente"}},
		},
		{
			id:    "trampo-escritorio-impressora",
			title: "A lista na impressora",
			areas: []int{areaEscritorio},
			when:  wentWrong,
			intros: []string{
				"A impressora do escritório travou bem quando {@} chegou. Na bandeja tem uma folha do RH com o título \"CORTES DO MÊS\"",
				"{@} foi buscar uma cópia e achou na impressora uma lista de demissões esquecida. Um dos nomes, meio borrado de toner, parece muito com o de {@}",
			},
			choices: []choice{
				{label: "👀 Ler a lista inteira", risk: midRisk, endings: []ending{
					{weight: 40, texts: []string{"Era a lista de promoções, não de demissões. O chefe confirmou o aumento de {@} no mesmo dia"}, effects: []effect{earn(80, 160)}},
					{weight: 35, texts: []string{"O gerente do RH chegou bem na hora e pegou {@} com a folha na mão"}, effects: []effect{delay(20 * time.Minute), lose(2, 5)}},
					{weight: 25, texts: []string{"Era só a lista de quem ia ganhar cesta de fim de ano, e {@} tava nela"}, effects: []effect{gain(1, "vale-refeicao")}},
				}},
				{label: "🗑️ Jogar no triturador", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"O RH perdeu a lista, adiou os cortes e a firma ainda pagou o bônus do trimestre"}, effects: []effect{earn(20, 50)}},
					{weight: 40, texts: []string{"O RH imprimiu de novo dez minutos depois"}},
				}},
				{label: "📣 Espalhar no grupo da firma", risk: midRisk, endings: []ending{
					{weight: 45, texts: []string{"A firma inteira ameaçou greve e a diretoria cancelou os cortes, com bônus de sossego"}, effects: []effect{earn(80, 150)}},
					{weight: 35, texts: []string{"Descobriram quem vazou a lista, e o desconto veio no contracheque de {@}"}, effects: []effect{lose(5, 10)}},
					{weight: 20, texts: []string{"A fofoca virou processo por vazamento de dados, e o advogado não foi barato"}, effects: []effect{lose(8, 15)}},
				}},
				{label: "🖨️ Consertar a impressora", risk: lowRisk, needs: isJob("suporte de ti", "programador"), endings: []ending{
					{weight: 100, texts: []string{"{@} desentupiu a impressora em dois minutos. O chefe ficou tão grato que riscou o nome de {@} da lista na hora"}, effects: []effect{earn(60, 120), refresh()}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, o gerente do RH voltou correndo pra buscar a folha"}},
		},
		{
			id:     "trampo-saude-leito-13",
			title:  "O paciente do leito 13",
			rarity: rareEvent,
			areas:  []int{areaSaude},
			intros: []string{
				"Plantão da madrugada e a campainha do leito 13 não para de tocar. O problema: o leito 13 tá vazio desde 1987",
				"{@} entrou no quarto 13 pra atender o chamado e encontrou um velhinho de pijama listrado pedindo água. A enfermeira-chefe jura que não tem ninguém internado ali",
			},
			choices: []choice{
				{label: "🥛 Levar a água pra ele", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O velhinho bebeu, sorriu e cochichou onde enterrou as economias no jardim do hospital. Tava lá mesmo"}, effects: []effect{earn(150, 400)}},
					{weight: 30, texts: []string{"Ele agradeceu tanto que resolveu acompanhar {@} pra casa"}, effects: []effect{enchant("encosto")}},
					{weight: 20, texts: []string{"Ele bebeu e sumiu. Em cima do travesseiro ficou uma moeda antiga"}, effects: []effect{gain(1, "moeda-do-imperio")}},
				}},
				{label: "🏃 Sair correndo do quarto", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} só parou de correr no estacionamento"}, effects: []effect{delay(15 * time.Minute)}},
					{weight: 30, texts: []string{"{@} escorregou no piso recém-encerado no meio da fuga"}, effects: []effect{injure(3, 10)}},
				}},
				{label: "🗣️ Puxar conversa com ele", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"Ele contou histórias do hospital antigo a noite inteira e {@} terminou o plantão sem sentir o tempo"}, effects: []effect{heal(20), refresh()}},
					{weight: 40, texts: []string{"Ele passou três horas reclamando da gelatina do hospital"}, effects: []effect{delay(10 * time.Minute)}},
				}},
				{label: "🩺 Examinar o paciente", risk: lowRisk, needs: holds("estetoscopio"), endings: []ending{
					{weight: 100, texts: []string{"O estetoscópio não achou batimento nenhum. O velhinho riu, agradeceu o primeiro doutor que deu atenção pra ele em 40 anos e deixou uma bênção"}, effects: []effect{enchant("protecao"), enchant("sorte")}},
				}},
				{label: "💧 Benzer com água benta", risk: lowRisk, uses: "agua-benta", endings: []ending{
					{weight: 100, texts: []string{"O velhinho foi embora em paz e a campainha nunca mais tocou. A diretoria deu um bônus de coragem pra {@}"}, effects: []effect{earn(100, 250), enchant("protecao")}},
				}},
			},
			idle: ending{texts: []string{"A campainha parou sozinha. No dia seguinte, a cama do leito 13 apareceu arrumadinha"}},
		},
		{
			id:    "trampo-arte-casamento",
			title: "O noivo fugiu",
			areas: []int{areaArte},
			intros: []string{
				"{@} foi contratado pra animar um casamento. Na hora do sim, o noivo saiu correndo pela porta lateral e a festa inteira olhou pro palco",
				"O casamento que {@} ia animar travou: o noivo sumiu, a noiva tá chorando no buffet e o pai dela quer saber quem vai pagar o show",
			},
			choices: []choice{
				{label: "🎤 Tocar assim mesmo", risk: midRisk, endings: []ending{
					{weight: 45, texts: []string{"{@} mandou \"Evidências\" e o casamento virou festa de superação. A noiva pagou com gorjeta"}, effects: []effect{earn(60, 140)}},
					{weight: 35, texts: []string{"Os convidados acharam desrespeito e jogaram bem-casado em {@} até acabar a bandeja"}, effects: []effect{injure(3, 10)}},
					{weight: 20, texts: []string{"No meio do show o noivo voltou arrependido, e o casamento rolou com {@} de trilha sonora"}, effects: []effect{earn(80, 160)}},
				}},
				{label: "💸 Cobrar o cachê na hora", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"O pai da noiva pagou só pra se livrar de mais um problema"}, effects: []effect{earn(40, 80)}},
					{weight: 40, texts: []string{"O pai da noiva disse que quem pagava era o noivo. E o noivo sumiu"}},
				}},
				{label: "🏃 Ir atrás do noivo", risk: midRisk, endings: []ending{
					{weight: 45, texts: []string{"{@} achou o noivo no bar da esquina e trouxe ele de volta. Virou padrinho de honra e ganhou o envelope"}, effects: []effect{earn(120, 220)}},
					{weight: 35, texts: []string{"O noivo tava fugindo do irmão da noiva, e o irmão achou que {@} era cúmplice"}, effects: []effect{injure(10, 25)}},
					{weight: 20, texts: []string{"O noivo convenceu {@} a fugir junto pra rodoviária, e a volta demorou"}, effects: []effect{delay(30 * time.Minute)}},
				}},
				{label: "💡 Fazer live do casamento", risk: lowRisk, needs: holds("ring-light"), endings: []ending{
					{weight: 100, texts: []string{"{@} ligou a ring light e fez live do casamento cancelado. Bateu recorde de audiência e choveu patrocínio"}, effects: []effect{earn(100, 200)}},
				}},
				{label: "🎩 Fazer o noivo aparecer", risk: midRisk, needs: isJob("mágico", "palhaço"), endings: []ending{
					{weight: 60, texts: []string{"Abracadabra: {@} abriu o armário do salão e lá tava o noivo, escondido. A festa foi salva"}, effects: []effect{earn(100, 180)}},
					{weight: 40, texts: []string{"Da cartola saiu só um pombo, que fez cocô no bolo dos noivos"}, effects: []effect{lose(2, 5)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} decidia, o DJ do salão assumiu o palco e levou o cachê"}},
		},
	},
}
