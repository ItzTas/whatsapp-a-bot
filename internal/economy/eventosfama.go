package economy

import "time"

var fameEvents = eventBook{
	name:   "🌟 Fama",
	places: []string{"!postar"},
	scenarios: []scenario{
		{
			id:    "fama-hater",
			title: "O hater do post",
			intros: []string{
				"Mal saiu o post e o primeiro comentário já foi de um perfil sem foto: '{@} é a pessoa mais sem graça da internet'",
				"Um tal de @verdade_doa_a_quem_doer_77 comentou em todos os posts de {@} na última hora. Nenhum comentário foi elogio",
			},
			choices: []choice{
				{label: "😏 Responder com deboche", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"A resposta de {@} foi tão boa que virou print e rodou tudo que é página de meme"}, effects: []effect{fans(3, 10)}},
					{weight: 30, texts: []string{"O hater respondeu melhor ainda e a galera ficou do lado dele"}, effects: []effect{unfollow(2, 6)}},
					{weight: 20, texts: []string{"Virou uma treta de 300 comentários e ninguém lembra quem começou"}},
				}},
				{label: "🙉 Ignorar e seguir a vida", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} nem respondeu. O hater cansou sozinho e foi encher o saco de outro"}},
					{weight: 30, texts: []string{"Os seguidores de {@} defenderam por conta própria e o post ainda bombou com a confusão"}, effects: []effect{fans(1, 4)}},
				}},
				{label: "📢 Expor o hater nos stories", risk: midRisk, endings: []ending{
					{weight: 45, texts: []string{"Os seguidores de {@} caíram em cima do perfil e o hater apagou a conta. Mas ele vai voltar, eles sempre voltam"}, effects: []effect{fans(2, 6), plant("fama-hater-volta", 6*time.Hour)}},
					{weight: 35, texts: []string{"O hater era um menino de 12 anos e a internet achou {@} exagerado demais"}, effects: []effect{unfollow(3, 8)}},
					{weight: 20, texts: []string{"O hater tinha mais seguidor que {@} e o exposed voltou dobrado"}, effects: []effect{unfollow(4, 10), enchant("olhogordo")}},
				}},
				{label: "🎬 Transformar em conteúdo", risk: lowRisk, needs: inArea(areaArte), endings: []ending{
					{weight: 75, texts: []string{"{@} leu os comentários de ódio em voz alta com fundo de música triste e o vídeo explodiu"}, effects: []effect{fans(4, 12)}},
					{weight: 25, texts: []string{"O vídeo ficou bom, mas o hater amou a atenção e agora comenta em dobro"}, effects: []effect{fans(1, 3)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava na resposta perfeita, o post já tinha sumido do feed de todo mundo"}},
		},
		{
			id:     "fama-hater-volta",
			title:  "O hater voltou",
			rarity: rareEvent,
			stage:  true,
			intros: []string{
				"Chegou uma mensagem de um perfil novo: 'sou eu, o do comentário. Fiz terapia e virei seu fã. Desculpa, {@}'",
				"O hater que {@} expôs voltou com outra conta, e dessa vez veio com textão de desculpa e uma proposta",
			},
			choices: []choice{
				{label: "🤝 Perdoar e fazer um vídeo juntos", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O vídeo da reconciliação fez a internet chorar. Até a mãe de {@} comentou"}, effects: []effect{fans(5, 12), enchant("sorte")}},
					{weight: 30, texts: []string{"No meio da gravação ele voltou a criticar o corte de cabelo de {@}. Velhos hábitos"}, effects: []effect{fans(1, 4)}},
				}},
				{label: "⚖️ Mandar pro advogado", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"O advogado fez acordo e o ex-hater pagou os danos morais em suaves parcelas no Pix"}, effects: []effect{earn(60, 160)}},
					{weight: 40, texts: []string{"O advogado cobrou a consulta e o processo não deu em nada"}, effects: []effect{lose(2, 5)}},
				}},
				{label: "🚫 Bloquear de novo", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} bloqueou sem ler. Paz de espírito não tem preço"}, effects: []effect{heal(10)}},
				}},
			},
		},
		{
			id:    "fama-famoso-comentou",
			title: "Um famoso comentou",
			when:  wentWell,
			intros: []string{
				"Um famoso de verdade, com selinho azul e tudo, comentou 'kkkkkkk' no post de {@}. O celular não para de vibrar",
				"{@} abriu as notificações e quase caiu da cadeira: um cantor famoso curtiu e comentou o post",
			},
			choices: []choice{
				{label: "📸 Printar e postar", risk: lowRisk, endings: []ending{
					{weight: 80, texts: []string{"O print com 'MEU DEUS' em caixa alta rendeu uns seguidores curiosos"}, effects: []effect{fans(2, 5)}},
					{weight: 20, texts: []string{"O famoso apagou o comentário logo depois e {@} ficou com cara de quem inventou tudo"}, effects: []effect{unfollow(1, 3)}},
				}},
				{label: "💬 Mandar DM pedindo parceria", risk: midRisk, endings: []ending{
					{weight: 30, texts: []string{"O famoso respondeu! Gravaram um vídeo juntos e o perfil de {@} subiu igual foguete"}, effects: []effect{fans(8, 20)}},
					{weight: 45, texts: []string{"Visualizou e não respondeu. Faz parte"}},
					{weight: 25, texts: []string{"A assessoria respondeu com um orçamento de parceria. {@} pagou um pouco pra ouvir um não educado"}, effects: []effect{lose(2, 6)}},
				}},
				{label: "😎 Fingir que nem viu", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"A pose de quem não liga funcionou: virou o perfil misterioso do momento"}, effects: []effect{fans(1, 4)}},
					{weight: 40, texts: []string{"Ninguém percebeu o comentário e {@} ficou sem ter com quem comemorar"}},
				}},
				{label: "✍️ Pedir um autógrafo de presente", risk: lowRisk, needs: inArea(areaArte), endings: []ending{
					{weight: 100, texts: []string{"Colega de profissão se ajuda: o famoso mandou um autógrafo pelo correio com dedicatória pra {@}"}, effects: []effect{gain(1, "autografo")}},
				}},
			},
		},
		{
			id:    "fama-trend-perigosa",
			title: "A trend perigosa",
			intros: []string{
				"A trend da semana é pular de um telhado pra uma piscina de plástico. Tem 40 milhões de visualizações e {@} já tá olhando pro telhado da vizinha",
				"Todo mundo tá fazendo o desafio de comer a pimenta mais ardida do mundo e lamber uma bateria logo depois. {@} comprou as duas coisas",
			},
			choices: []choice{
				{label: "🤸 Fazer o desafio de verdade", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"Deu certo do jeito mais bizarro possível e o vídeo de {@} foi o mais assistido da trend"}, effects: []effect{fans(8, 20)}},
					{weight: 40, texts: []string{"Deu tudo errado e {@} foi parar no hospital. O vídeo, pelo menos, rendeu umas visualizações"}, effects: []effect{injure(20, 45), fans(1, 4)}},
					{weight: 10, texts: []string{"{@} se machucou feio e ainda foi chamado de irresponsável no jornal da noite"}, effects: []effect{injure(25, 45), unfollow(4, 10)}},
					{weight: 5, texts: []string{"Foi o último vídeo de {@}. A trend foi proibida no dia seguinte"}, effects: []effect{perish()}},
				}},
				{label: "🧯 Fazer a versão segura", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} pulou de um banquinho numa bacia. Ninguém achou graça, mas também ninguém se machucou"}},
					{weight: 40, texts: []string{"A versão segura ficou tão ridícula que virou uma trend nova"}, effects: []effect{fans(2, 6)}},
				}},
				{label: "🎭 Fazer uma paródia da trend", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"A paródia zoando quem faz o desafio agradou todo mundo que tava de saco cheio da trend"}, effects: []effect{fans(3, 9)}},
					{weight: 45, texts: []string{"Os fãs da trend acharam que {@} tava se achando melhor que eles"}, effects: []effect{unfollow(2, 6)}},
				}},
				{label: "🚨 Denunciar a trend", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} gravou um alerta sério sobre o desafio. Poucos assistiram, mas as mães do grupo agradeceram"}, effects: []effect{enchant("protecao")}},
				}},
			},
		},
		{
			id:    "fama-live-caiu",
			title: "A internet caiu",
			intros: []string{
				"No melhor momento da live, com o chat bombando, a internet de {@} caiu. O roteador tá piscando vermelho",
				"{@} tava ao vivo quando a tela congelou numa cara horrível. A operadora derrubou o sinal do bairro inteiro",
			},
			choices: []choice{
				{label: "📶 Pegar o Wi-Fi da vizinha", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"A senha era o nome do cachorro dela. A live voltou e o chat comemorou"}, effects: []effect{fans(2, 6)}},
					{weight: 45, texts: []string{"A vizinha viu o nome do celular de {@} conectado, bateu na porta e cobrou a conta de internet do mês"}, effects: []effect{lose(2, 6)}},
				}},
				{label: "📱 Voltar pelo 4G", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"A imagem ficou de batata, mas o chat fiel continuou ali"}, effects: []effect{fans(1, 3)}},
					{weight: 40, texts: []string{"A franquia acabou em dez minutos e {@} teve que pagar pacote extra"}, effects: []effect{lose(1, 4)}},
				}},
				{label: "😤 Xingar a operadora no ar", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O desabafo de {@} contra a operadora virou o grito de guerra de todo mundo com internet ruim"}, effects: []effect{fans(4, 10)}},
					{weight: 50, texts: []string{"Falou tanto palavrão que a plataforma derrubou o vídeo"}, effects: []effect{unfollow(2, 5), delay(20 * time.Minute)}},
				}},
				{label: "😴 Desistir e ir dormir", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} desligou tudo e dormiu doze horas. Acordou novo"}, effects: []effect{heal(15)}},
				}},
			},
			idle: ending{texts: []string{"{@} ficou olhando pro roteador esperando um milagre. O chat foi embora"}, effects: []effect{unfollow(1, 2)}},
		},
		{
			id:    "fama-celular-na-privada",
			title: "Caiu na privada",
			intros: []string{
				"{@} tava gravando um story no banheiro e o celular escorregou direto pra privada. Ainda tá gravando lá dentro",
				"Plop. O celular de {@} caiu na privada no meio de um vídeo. A tela ainda acende debaixo d'água",
			},
			choices: []choice{
				{label: "🖐️ Enfiar a mão e pegar", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"Resgate feito. O celular sobreviveu, a dignidade de {@} nem tanto"}},
					{weight: 25, texts: []string{"{@} puxou o celular, mas ele deu a última piscada e morreu"}, effects: []effect{forfeit("celular", 1)}},
					{weight: 20, texts: []string{"O vídeo continuou gravando e ficou o melhor conteúdo da semana"}, effects: []effect{fans(3, 8)}},
				}},
				{label: "🍚 Colocar no arroz", risk: lowRisk, endings: []ending{
					{weight: 65, texts: []string{"Um dia inteiro no arroz e o celular ligou de novo. Ninguém sabe se o arroz ajuda, mas funcionou"}, effects: []effect{delay(30 * time.Minute)}},
					{weight: 35, texts: []string{"O arroz não salvou e agora o celular tem cheiro de banheiro e de arroz"}, effects: []effect{forfeit("celular", 1)}},
				}},
				{label: "🎥 Postar a desgraça", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"O vídeo do celular caindo, gravado do ponto de vista dele, foi parar em todas as páginas de humor"}, effects: []effect{fans(4, 12)}},
					{weight: 45, texts: []string{"A galera achou nojento e uns seguidores foram embora com ânsia"}, effects: []effect{unfollow(2, 5)}},
				}},
			},
		},
		{
			id:     "fama-empresario",
			title:  "O empresário de famosos",
			rarity: rareEvent,
			intros: []string{
				"Um homem de óculos escuros e corrente de ouro mandou DM: 'sou empresário de famoso, faço você estourar em uma semana. Só preciso de uma taxinha'",
				"Um tal de Marcelinho Produções ligou pra {@}: 'te vi no feed, você tem estrela. Investe comigo que eu te boto no Faustão'",
			},
			choices: []choice{
				{label: "💸 Pagar a taxinha", risk: highRisk, price: cost{percent: 5, floor: 100}, endings: []ending{
					{weight: 55, texts: []string{"O Marcelinho sumiu com o dinheiro e mudou de nome no Instagram. Agora é Marcelão Eventos"}},
					{weight: 30, texts: []string{"Era golpe, mas os seguidores comprados vieram. Todos robôs, mas contam"}, effects: []effect{fans(6, 15)}},
					{weight: 15, texts: []string{"Por incrível que pareça era sério: o empresário arrumou uma entrevista num programa de TV"}, effects: []effect{fans(12, 25)}},
				}},
				{label: "📄 Pedir contrato assinado", risk: lowRisk, endings: []ending{
					{weight: 80, texts: []string{"Na hora do contrato o empresário parou de responder. Golpe evitado"}},
					{weight: 20, texts: []string{"O contrato veio, com cláusula de 90% de tudo que {@} ganhar até 2050. {@} recusou e postou o print"}, effects: []effect{fans(2, 6)}},
				}},
				{label: "📢 Expor o golpe", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"O vídeo expondo o Marcelinho salvou um monte de gente e a internet agradeceu"}, effects: []effect{fans(4, 10), enchant("justica")}},
					{weight: 40, texts: []string{"O Marcelinho mandou um áudio ameaçador e uns amigos dele apareceram na porta de {@}"}, effects: []effect{injure(10, 25)}},
				}},
				{label: "👮 Dar voz de prisão", risk: midRisk, needs: lawman, endings: []ending{
					{weight: 75, texts: []string{"{@} marcou um encontro, chegou de viatura e ainda ganhou a recompensa da delegacia de golpes"}, effects: []effect{earn(100, 250)}},
					{weight: 25, texts: []string{"O Marcelinho percebeu a armadilha e fugiu pela janela do café"}},
				}},
			},
		},
		{
			id:     "fama-fake-news",
			title:  "A fake news",
			rarity: rareEvent,
			intros: []string{
				"Uma página de fofoca postou que {@} foi visto saindo de um motel com o técnico da seleção. É mentira, mas já tem 50 mil curtidas",
				"Espalharam no grupo da família que {@} morreu. Os seguidores tão postando vela e foto de luto",
			},
			choices: []choice{
				{label: "🎥 Gravar vídeo desmentindo", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"O desmentido fez sucesso e ainda veio uma onda de gente nova curiosa"}, effects: []effect{fans(3, 8)}},
					{weight: 45, texts: []string{"Ninguém acreditou no desmentido e a fofoca virou verdade na internet"}, effects: []effect{unfollow(3, 8)}},
				}},
				{label: "😂 Abraçar o meme", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} entrou na brincadeira e virou o assunto do dia no Brasil inteiro"}, effects: []effect{fans(6, 15)}},
					{weight: 50, texts: []string{"A piada passou do ponto e marcas começaram a cancelar contrato"}, effects: []effect{unfollow(5, 12)}},
				}},
				{label: "⚖️ Processar a página", risk: lowRisk, price: cost{percent: 3, floor: 60}, endings: []ending{
					{weight: 65, texts: []string{"O juiz mandou a página pagar indenização e postar um pedido de desculpas"}, effects: []effect{earn(150, 300)}},
					{weight: 35, texts: []string{"A página era de um perfil fake no exterior. O advogado ficou com o dinheiro e ninguém foi punido"}},
				}},
				{label: "🫥 Sumir uns dias", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} deu um tempo das redes. A fofoca morreu sozinha, junto com uns seguidores"}, effects: []effect{unfollow(1, 3), heal(15)}},
				}},
			},
		},
		{
			id:     "fama-fantastico",
			title:  "O vídeo no Fantástico",
			rarity: legendaryEvent,
			when:   wentWell,
			intros: []string{
				"O vídeo de {@} foi parar no Fantástico, com direito a musiquinha e repórter na porta de casa",
				"O post de {@} rodou o mundo: passou no jornal da noite, num programa gringo e até na rádio do interior",
			},
			choices: []choice{
				{label: "🎤 Dar entrevista pro repórter", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} falou bonito na TV e a galera do Brasil inteiro foi seguir"}, effects: []effect{fans(15, 30), earn(100, 300)}},
					{weight: 30, texts: []string{"{@} travou na frente da câmera e virou meme de novo, que também conta"}, effects: []effect{fans(8, 18)}},
				}},
				{label: "💰 Vender os direitos do vídeo", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"Uma agência gringa comprou os direitos do vídeo e pagou em dólar"}, effects: []effect{earn(500, 1200)}},
				}},
				{label: "🤑 Cobrar de todo portal que usou", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} mandou cobrança pra todo site que usou o vídeo e metade pagou"}, effects: []effect{earn(600, 1500)}},
					{weight: 45, texts: []string{"Os portais pararam de usar o vídeo e a fama esfriou no mesmo dia"}, effects: []effect{earn(50, 150)}},
				}},
				{label: "🙏 Agradecer e seguir a vida", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} postou um agradecimento humilde e o universo gostou da atitude"}, effects: []effect{fans(5, 12), enchant("sorte"), enchant("prosperidade")}},
				}},
			},
		},
		{
			id:     "fama-publi-produto-ruim",
			title:  "O produto horrível",
			places: []string{"!publi"},
			intros: []string{
				"A marca mandou o shampoo da publi e, depois de testar, caiu um tufo do cabelo de {@}. O vídeo tem que sair hoje",
				"O produto da publi de {@} é um tênis que descolou a sola na primeira caminhada. O roteiro manda dizer que é 'o mais confortável da vida'",
			},
			choices: []choice{
				{label: "🗣️ Falar a verdade no vídeo", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"A sinceridade de {@} virou exemplo e a galera agradeceu"}, effects: []effect{fans(3, 9)}},
					{weight: 50, texts: []string{"A marca não gostou nada e cobrou a multa do contrato"}, effects: []effect{lose(3, 8)}},
				}},
				{label: "😁 Sorrir e dizer que ama", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"Ninguém percebeu nada e a marca mandou um bônus pela atuação"}, effects: []effect{earn(30, 80)}},
					{weight: 40, texts: []string{"Os seguidores compraram, odiaram e vieram cobrar nos comentários"}, effects: []effect{unfollow(2, 6)}},
				}},
				{label: "📦 Devolver tudo pra marca", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} devolveu o produto com um bilhete educado. Não ganhou nada além da consciência tranquila"}, effects: []effect{enchant("protecao")}},
				}},
				{label: "🎭 Atuar como se fosse novela", risk: lowRisk, needs: isJob("ator", "dublador"), endings: []ending{
					{weight: 100, texts: []string{"{@} chorou de emoção falando do produto. Atuação digna de prêmio, e a marca renovou o contrato"}, effects: []effect{earn(80, 180)}},
				}},
			},
		},
		{
			id:     "fama-publi-concorrente",
			title:  "A proposta da concorrente",
			places: []string{"!publi"},
			intros: []string{
				"Depois da publi, a marca concorrente mandou mensagem pra {@}: 'pagamos o dobro pra você falar mal deles amanhã'",
				"Um gerente da concorrência ligou pra {@} cochichando: 'sabe aquela publi? Grava um vídeo dizendo que deu alergia e o Pix cai na hora'",
			},
			choices: []choice{
				{label: "🤑 Aceitar e falar mal", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"O vídeo saiu, o Pix caiu e ninguém juntou os pontos"}, effects: []effect{earn(150, 300), stain()}},
					{weight: 35, texts: []string{"A primeira marca descobriu o esquema e processou {@} por quebra de contrato"}, effects: []effect{lose(8, 18)}},
					{weight: 20, texts: []string{"Os seguidores perceberam que {@} faz publi pra qualquer um e foram embora"}, effects: []effect{unfollow(6, 15)}},
				}},
				{label: "🙅 Recusar a proposta", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} recusou e a marca original ficou sabendo. Ganhou fama de pessoa honesta no mercado"}, effects: []effect{enchant("justica")}},
					{weight: 30, texts: []string{"{@} recusou e ninguém nunca soube. Honestidade às vezes é só isso"}},
				}},
				{label: "🔨 Fazer leilão entre as duas", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"As duas marcas brigaram pela exclusividade e {@} saiu com um cachê extra"}, effects: []effect{earn(100, 220)}},
					{weight: 50, texts: []string{"As duas marcas descobriram o leilão e cortaram {@} da lista"}},
				}},
				{label: "📲 Contar tudo pra marca", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"A marca agradeceu a lealdade com um bônus e um kit de produtos"}, effects: []effect{earn(40, 100)}},
				}},
			},
		},
		{
			id:     "fama-publi-tigrinho",
			title:  "A publi do joguinho",
			rarity: rareEvent,
			places: []string{"!publi"},
			intros: []string{
				"Um site de aposta ofereceu pra {@} um cachê gordo pra divulgar um joguinho do bicho com animação de tigre. O contrato pede pra dizer que 'é dinheiro fácil'",
				"A plataforma de apostas quer {@} gravando um vídeo ganhando no joguinho, com a conta de demonstração que sempre ganha",
			},
			choices: []choice{
				{label: "🐯 Divulgar o joguinho", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"O cachê caiu e o vídeo rodou. Ninguém perguntou de onde veio o dinheiro"}, effects: []effect{earn(200, 300), stain()}},
					{weight: 35, texts: []string{"A polícia fez uma operação contra quem divulgava o jogo e bateu na porta de {@}"}, effects: []effect{nabbed()}},
					{weight: 20, texts: []string{"Os seguidores perderam dinheiro no jogo e vieram cobrar de {@} em massa"}, effects: []effect{unfollow(8, 15)}},
				}},
				{label: "🙅 Recusar e contar pra galera", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} recusou e ainda postou o print da proposta. Virou exemplo e ganhou respeito"}, effects: []effect{fans(3, 8)}},
					{weight: 30, texts: []string{"{@} recusou e o site chamou outro influencer no mesmo dia"}},
				}},
				{label: "🔍 Divulgar com aviso miúdo", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"O aviso de 'jogue com responsabilidade' em letra minúscula salvou a pele de {@} na justiça"}, effects: []effect{earn(80, 180)}},
					{weight: 45, texts: []string{"O juiz não achou o aviso minúsculo suficiente e aplicou multa"}, effects: []effect{lose(5, 12), stain()}},
				}},
				{label: "👮 Prender o dono do site", risk: midRisk, needs: lawman, endings: []ending{
					{weight: 70, texts: []string{"{@} marcou a assinatura do contrato e chegou com mandado. Operação com coletiva de imprensa e tudo"}, effects: []effect{earn(150, 300), fans(4, 10), enchant("justica")}},
					{weight: 30, texts: []string{"O dono do site tava em outro país. Só deu pra prender o estagiário"}, effects: []effect{earn(30, 80)}},
				}},
			},
		},
		{
			id:     "fama-publi-ao-vivo",
			title:  "A publi ao vivo",
			places: []string{"!publi"},
			intros: []string{
				"A publi de {@} era ao vivo, fritando um ovo na panela antiaderente da marca. O ovo grudou na panela inteira",
				"{@} tava ao vivo mostrando o aspirador da marca quando ele soltou fumaça e apagou",
			},
			choices: []choice{
				{label: "🤹 Improvisar como se fosse de propósito", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O improviso foi tão bom que a marca usou o vídeo como comercial"}, effects: []effect{fans(3, 8), earn(40, 100)}},
					{weight: 50, texts: []string{"Ninguém acreditou e o corte da falha rodou a internet"}, effects: []effect{unfollow(2, 6)}},
				}},
				{label: "🔥 Culpar o fogão", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"O fogão levou a culpa e a marca ficou aliviada"}},
					{weight: 40, texts: []string{"A marca do fogão viu a live e mandou uma notificação extrajudicial"}, effects: []effect{lose(2, 6)}},
				}},
				{label: "🍳 Comer o ovo grudado assim mesmo", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} raspou o ovo com a espátula e comeu sorrindo. O chat amou a coragem"}, effects: []effect{fans(3, 9)}},
					{weight: 45, texts: []string{"O ovo veio com pedaço de teflon e {@} passou a noite com dor de barriga"}, effects: []effect{injure(5, 15)}},
				}},
				{label: "🎩 Transformar em número de mágica", risk: lowRisk, needs: isJob("mágico", "palhaço"), endings: []ending{
					{weight: 100, texts: []string{"{@} fez a panela 'sumir' atrás do balcão e voltou com uma nova. Aplausos no chat e bônus da marca"}, effects: []effect{earn(60, 150), fans(2, 6)}},
				}},
			},
		},
	},
}
