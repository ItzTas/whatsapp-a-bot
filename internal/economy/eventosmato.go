package economy

import "time"

var buggingEvents = eventBook{
	name:   "🦋 Mato",
	places: []string{bugging.command},
	scenarios: []scenario{
		{
			id:    "mato-colmeia-monstro",
			title: "A colmeia monstro",
			intros: []string{
				"Pendurada num galho, {@} achou uma colmeia do tamanho de uma melancia, pingando mel. O zumbido dá pra ouvir de longe",
				"{@} encostou numa árvore oca e ela começou a vibrar. Lá dentro tem uma colmeia gigante, e o cheiro de mel tá de outro mundo",
			},
			choices: []choice{
				{label: "🍯 Meter a mão no mel", risk: highRisk, endings: []ending{
					{weight: 35, texts: []string{"{@} enfiou a mão, arrancou um favo inteiro e saiu correndo antes do enxame acordar"}, effects: []effect{gain(2, "favo-de-mel")}},
					{weight: 50, texts: []string{"O enxame acordou. {@} levou umas 40 picadas e ficou com a cara igual bolacha recheada"}, effects: []effect{injure(25, 40), gain(1, "favo-de-mel")}},
					{weight: 15, texts: []string{"{@} descobriu do pior jeito que tem alergia a abelha"}, effects: []effect{injure(50, 80)}},
				}},
				{label: "🐝 Colher do jeito certo", risk: lowRisk, needs: either(holds("roupa-de-apicultor"), isJob("apicultor")), endings: []ending{
					{weight: 100, texts: []string{"Protegido dos pés à cabeça, {@} colheu o mel com toda a calma do mundo e ainda passou um pouco nas picadas antigas"}, effects: []effect{gain(3, "favo-de-mel"), earn(20, 40), heal(10)}},
				}},
				{label: "🔥 Defumar a colmeia", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"A fumaça acalmou as abelhas e {@} saiu com um favo na mão"}, effects: []effect{gain(1, "favo-de-mel")}},
					{weight: 40, texts: []string{"A fumaça virou fogo e quase queimou o mato inteiro. {@} saiu sem sobrancelha e com a ficha suja"}, effects: []effect{injure(10, 25), stain()}},
				}},
				{label: "👀 Admirar de longe", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"De longe, {@} viu uma borboleta pousar no mel e pegou ela com calma"}, effects: []effect{gain(1, "borboleta")}},
					{weight: 30, texts: []string{"Uma abelha curiosa veio ver {@} de perto e picou bem na ponta do nariz"}, effects: []effect{injure(3, 8)}},
				}},
			},
			idle: ending{texts: []string{"{@} demorou tanto que o enxame veio ver o que tava acontecendo"}, effects: []effect{injure(5, 10)}},
		},
		{
			id:    "mato-cobra-na-trilha",
			title: "A cobra na trilha",
			intros: []string{
				"No meio da trilha, uma cobra enrolada levantou a cabeça pra {@}. Pode ser jararaca, pode ser cobra-cipó, e {@} não faz ideia da diferença",
				"{@} ia pegar um grilo quando percebeu que o galho do lado tava se mexendo. Não era galho",
			},
			choices: []choice{
				{label: "🪵 Espantar com um galho", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"A cobra se ofendeu e foi embora pro mato, deixando o caminho livre"}, effects: []effect{findLoot()}},
					{weight: 30, texts: []string{"A cobra deu o bote e pegou de raspão na canela de {@}"}, effects: []effect{injure(15, 30)}},
					{weight: 15, texts: []string{"O bote foi certeiro, e a perna de {@} inchou igual pão de queijo"}, effects: []effect{injure(35, 50), delay(20 * time.Minute)}},
				}},
				{label: "🗿 Ficar parado igual estátua", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"Depois de 10 minutos de estátua, a cobra cansou e foi embora"}, effects: []effect{delay(10 * time.Minute)}},
					{weight: 30, texts: []string{"A cobra subiu na perna de {@}, deu uma volta e desceu. Ninguém nunca vai acreditar nessa história"}, effects: []effect{enchant("sorte")}},
				}},
				{label: "🔄 Voltar pela outra trilha", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"A outra trilha tava cheia de bicho, e {@} saiu no lucro"}, effects: []effect{findLoot()}},
					{weight: 40, texts: []string{"A outra trilha não levava a lugar nenhum, e {@} perdeu um tempão"}, effects: []effect{delay(10 * time.Minute)}},
				}},
				{label: "🔎 Identificar a cobra", risk: lowRisk, needs: either(inArea(areaCampo), isJob("veterinário")), endings: []ending{
					{weight: 100, texts: []string{"{@} reconheceu na hora: cobra-cipó, inofensiva. Pegou com a mão, soltou longe e ainda achou coisa boa no caminho"}, effects: []effect{findLoot(), findLoot()}},
				}},
			},
		},
		{
			id:     "mato-pegadas-ao-contrario",
			title:  "Pegadas ao contrário",
			rarity: rareEvent,
			intros: []string{
				"{@} achou pegadas no barro, mas os pés apontam pro lado contrário de onde elas vêm. Do meio do mato, alguém assobiou",
				"Um assobio fino cortou o mato, e {@} percebeu que as pegadas no chão tavam todas viradas pra trás",
			},
			choices: []choice{
				{label: "👣 Seguir as pegadas", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"As pegadas levaram {@} pra uma clareira cheia de bicho raro"}, effects: []effect{findLoot(), findLoot(), findLoot()}},
					{weight: 45, texts: []string{"As pegadas andavam em círculo, e {@} passou 3 horas rodando no mato"}, effects: []effect{delay(30 * time.Minute), injure(5, 15)}},
					{weight: 15, texts: []string{"As pegadas pararam numa árvore. Atrás dela, uma figura baixinha de cabelo de fogo"}, next: "mato-curupira-em-pessoa"},
				}},
				{label: "🥃 Deixar uma oferenda", risk: lowRisk, uses: "cachaca", endings: []ending{
					{weight: 100, texts: []string{"{@} deixou a cachaça no pé de uma árvore. Quando olhou de novo, a garrafa tava vazia e o mato parecia mais amigo"}, effects: []effect{enchant("maoboa")}},
				}},
				{label: "🗣️ Gritar que respeita o mato", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"O assobio parou. Do nada, uma joaninha pousou no ombro de {@}"}, effects: []effect{enchant("sorte"), gain(1, "joaninha")}},
					{weight: 40, texts: []string{"O assobio virou gargalhada, e {@} tropeçou em todo cipó do caminho"}, effects: []effect{injure(5, 15), enchant("panema")}},
				}},
				{label: "🏃 Sair correndo do mato", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} saiu do mato sem olhar pra trás e só parou de tremer em casa"}},
				}},
			},
		},
		{
			id:     "mato-curupira-em-pessoa",
			title:  "O Curupira em pessoa",
			rarity: rareEvent,
			stage:  true,
			intros: []string{
				"É ele: o Curupira, de cabelo de fogo e pé virado. Ele olhou pra rede de {@}, cruzou os braços e disse que tem gente catando bicho demais no mato dele",
			},
			choices: []choice{
				{label: "🙇 Prometer cuidar do mato", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"O Curupira gostou da promessa, soprou um vento quente, e {@} se sentiu novo em folha"}, effects: []effect{enchant("maoboa"), heal(20)}},
				}},
				{label: "🪙 Pagar um tributo", risk: midRisk, price: cost{percent: 5, floor: 50}, endings: []ending{
					{weight: 70, texts: []string{"O Curupira guardou as moedas no cabelo de fogo e mostrou o caminho dos bichos raros"}, effects: []effect{findLoot(), findLoot(), earn(50, 100)}},
					{weight: 30, texts: []string{"O Curupira pegou as moedas, deu risada e sumiu"}},
				}},
				{label: "😠 Encarar o Curupira", risk: highRisk, endings: []ending{
					{weight: 30, texts: []string{"O Curupira respeitou a coragem de {@} e deixou um presente antes de sumir"}, effects: []effect{gain(1, "mosquito-no-ambar", "besouro-rinoceronte")}},
					{weight: 62, texts: []string{"O Curupira fez {@} andar em círculo até cair de cansaço"}, effects: []effect{injure(20, 35), enchant("panema")}},
					{weight: 8, texts: []string{"Ninguém encara o Curupira. O mato engoliu {@}, e acharam só a rede, dias depois"}, effects: []effect{perish()}},
				}},
			},
			idle: ending{texts: []string{"{@} ficou mudo, e o Curupira sumiu dando risada"}, effects: []effect{enchant("panema")}},
		},
		{
			id:    "mato-cogumelo-brilhante",
			title: "O cogumelo que brilha",
			intros: []string{
				"Numa sombra úmida, {@} achou um cogumelo que brilha no escuro. Tem cheiro de chá de vó e de péssimas decisões",
				"{@} tropeçou num tronco podre coberto de cogumelo colorido. Um deles parece estar sorrindo",
			},
			choices: []choice{
				{label: "🍵 Fazer um chazinho", risk: highRisk, endings: []ending{
					{weight: 60, texts: []string{"O chá bateu, e {@} começou a ouvir as cores e ver os sons"}, effects: []effect{buzz("cogumelo")}},
					{weight: 25, texts: []string{"O chá bateu forte, e {@} passou a tarde abraçando uma árvore e pedindo desculpa pra ela"}, effects: []effect{buzz("cogumelo"), delay(20 * time.Minute)}},
					{weight: 15, texts: []string{"Não era aquele cogumelo. Era o primo venenoso dele"}, effects: []effect{injure(30, 50)}},
				}},
				{label: "🫙 Guardar num potinho", risk: midRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} guardou o cogumelo num potinho, com todo o carinho do mundo"}, effects: []effect{gain(1, "cogumelo")}},
					{weight: 30, texts: []string{"O cogumelo apodreceu na mochila e o cheiro grudou em tudo"}},
				}},
				{label: "🔬 Vender pra uma bióloga", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"Uma bióloga da universidade comprou o cogumelo pra pesquisa e prometeu botar o nome de {@} no artigo"}, effects: []effect{earn(40, 90)}},
				}},
				{label: "🤪 Comer cru mesmo", risk: highRisk, needs: stoned, endings: []ending{
					{weight: 50, texts: []string{"{@} já tava doidão e comeu cru. Agora tá conversando com uma formiga sobre política"}, effects: []effect{buzz("cogumelo"), heal(10)}},
					{weight: 50, texts: []string{"Misturar com o que {@} já tinha usado foi uma péssima ideia"}, effects: []effect{buzz("cogumelo"), injure(20, 40)}},
				}},
			},
		},
		{
			id:    "mato-borboleta-azul",
			title: "A borboleta-azul no galho alto",
			intros: []string{
				"Lá no alto de uma árvore, {@} viu uma borboleta-azul, dessas que colecionador paga uma fortuna. Ela não parece ter pressa de descer",
				"Um brilho azul passou voando por cima de {@} e pousou no galho mais alto da mangueira. Era uma borboleta-azul de verdade",
			},
			choices: []choice{
				{label: "🧗 Subir na árvore", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"{@} subiu igual macaco e pegou a borboleta com a mão"}, effects: []effect{gain(1, "borboleta-azul")}},
					{weight: 45, texts: []string{"O galho quebrou no meio do caminho, e {@} desceu do jeito mais rápido possível"}, effects: []effect{injure(20, 40)}},
					{weight: 15, texts: []string{"A borboleta voou, e {@} passou meia hora lá em cima esperando o bombeiro"}, effects: []effect{delay(30 * time.Minute)}},
				}},
				{label: "🍯 Atrair com melado", risk: lowRisk, uses: "garrafa-de-melado", endings: []ending{
					{weight: 55, texts: []string{"O cheiro do melado foi irresistível, e a borboleta desceu direto pra rede"}, effects: []effect{gain(1, "borboleta-azul")}},
					{weight: 45, texts: []string{"Vieram 300 formigas e nenhuma borboleta"}, effects: []effect{gain(3, "formiga")}},
				}},
				{label: "🥅 Jogar a rede pra cima", risk: midRisk, endings: []ending{
					{weight: 35, texts: []string{"Acertou de primeira! A borboleta caiu direitinho na rede"}, effects: []effect{gain(1, "borboleta-azul")}},
					{weight: 40, texts: []string{"A rede enganchou no galho e lá ficou"}, effects: []effect{breakTool()}},
					{weight: 25, texts: []string{"Errou a azul, mas veio uma mariposa de consolação"}, effects: []effect{gain(1, "mariposa")}},
				}},
				{label: "📸 Só tirar uma foto", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} tirou a foto, postou, e a borboleta rendeu mais curtida que a última selfie"}, effects: []effect{earn(5, 20)}},
				}},
			},
			idle: ending{texts: []string{"A borboleta-azul cansou de esperar e foi embora voando"}},
		},
		{
			id:    "mato-formigueiro",
			title: "O formigueiro do tamanho de um fusca",
			intros: []string{
				"{@} pisou sem querer num formigueiro do tamanho de um fusca. As formigas saíram em fila e parecem bem bravas",
				"O chão começou a se mexer debaixo do pé de {@}. Era um formigueiro gigante, bem na época de tanajura",
			},
			choices: []choice{
				{label: "🏊 Correr pro rio", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} pulou no rio e as formigas desistiram. Só sobrou uma mordida ou outra"}, effects: []effect{injure(5, 10)}},
					{weight: 40, texts: []string{"O rio era longe. {@} chegou lá com 200 picadas de formiga-de-fogo"}, effects: []effect{injure(20, 35)}},
				}},
				{label: "🍳 Catar tanajura pra fritar", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"Era época de tanajura! {@} encheu um pote e já sabe o que vai ter na janta"}, effects: []effect{gain(4, "tanajura")}},
					{weight: 40, texts: []string{"{@} catou umas tanajuras, mas as soldados pegaram de jeito"}, effects: []effect{gain(2, "tanajura"), injure(10, 20)}},
				}},
				{label: "🧪 Comprar formicida", risk: lowRisk, price: cost{percent: 2, floor: 20}, endings: []ending{
					{weight: 100, texts: []string{"O formicida resolveu, e no fundo do formigueiro tinha uma coisa brilhando"}, effects: []effect{findLoot()}},
				}},
				{label: "🧘 Deixar elas passarem", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"As formigas passaram por cima de {@} carregando folha e nem ligaram"}},
					{weight: 30, texts: []string{"Uma formiga passou carregando uma joaninha, e {@} resgatou a coitada"}, effects: []effect{gain(1, "joaninha")}},
				}},
			},
		},
		{
			id:    "mato-chuva-de-granizo",
			title: "Chuva de granizo",
			intros: []string{
				"O céu ficou preto do nada e começou a cair pedra de gelo do tamanho de ovo de codorna na cabeça de {@}",
				"Trovejou forte e, em dois minutos, o mato virou pista de patinação. Tá chovendo granizo em cima de {@}",
			},
			choices: []choice{
				{label: "🌳 Correr pra debaixo da árvore", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"Deu certo, e debaixo da árvore tinha um monte de bicho escondido também"}, effects: []effect{findLoot(), findLoot()}},
					{weight: 30, texts: []string{"Um raio caiu perto, e {@} ficou com o cabelo em pé o resto do dia"}, effects: []effect{injure(10, 25)}},
					{weight: 10, texts: []string{"O raio caiu NA árvore"}, effects: []effect{injure(30, 45)}},
				}},
				{label: "🏠 Ir embora pra casa", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} chegou em casa ensopando o chão inteiro e desistiu do mato por hoje"}, effects: []effect{delay(10 * time.Minute)}},
				}},
				{label: "🧊 Catar gelo pra vender", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} encheu um isopor de granizo e vendeu como gelo de churrasco"}, effects: []effect{earn(30, 70)}},
					{weight: 50, texts: []string{"No meio da colheita, uma pedra de gelo acertou a testa de {@}"}, effects: []effect{injure(10, 20)}},
				}},
				{label: "⛑️ Botar o capacete", risk: lowRisk, needs: holds("capacete"), endings: []ending{
					{weight: 100, texts: []string{"Com o capacete, {@} ficou de boa no meio da chuva e catou os bichos que caíram das árvores"}, effects: []effect{findLoot(), findLoot()}},
				}},
			},
		},
		{
			id:    "mato-cacador-de-passarinho",
			title: "O caçador de passarinho",
			intros: []string{
				"{@} flagrou um sujeito com gaiolas cheias de passarinho e uma espingarda velha. Ele fez sinal de silêncio e ofereceu uma parte do lucro",
				"No meio do mato, {@} deu de cara com um acampamento de caçador ilegal: tatu, papagaio e um monte de gaiola",
			},
			choices: []choice{
				{label: "🤝 Aceitar a parte", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O sujeito pagou em dinheiro vivo e sumiu no mato"}, effects: []effect{earn(80, 160), stain()}},
					{weight: 30, texts: []string{"Era uma operação do Ibama, e os dois foram levados"}, effects: []effect{nabbed()}},
					{weight: 20, texts: []string{"O sujeito pagou com uma nota de três reais e sumiu antes de {@} reparar"}, effects: []effect{gain(1, "nota-de-tres")}},
				}},
				{label: "📞 Denunciar pro Ibama", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O Ibama chegou, soltou os bichos e agradeceu {@} na frente de todo mundo"}, effects: []effect{enchant("justica"), earn(20, 50)}},
					{weight: 30, texts: []string{"O caçador descobriu quem denunciou e jogou uma pedra em {@} na hora da fuga"}, effects: []effect{injure(10, 20), enchant("justica")}},
				}},
				{label: "🕊️ Soltar os passarinhos", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"{@} abriu as gaiolas e a passarinhada saiu cantando. Uma pena dourada caiu bem no ombro de {@}"}, effects: []effect{enchant("sorte")}},
					{weight: 40, texts: []string{"O caçador viu e partiu pra cima de {@}"}, effects: []effect{injure(15, 30)}},
				}},
				{label: "🚔 Dar voz de prisão", risk: midRisk, needs: lawman, endings: []ending{
					{weight: 80, texts: []string{"{@} deu voz de prisão, apreendeu a espingarda e virou herói do bairro"}, effects: []effect{earn(60, 120)}},
					{weight: 20, texts: []string{"O caçador correu mais que a lei e sumiu no mato"}},
				}},
			},
		},
		{
			id:    "mato-perdido-a-noite",
			title: "Perdido no mato de noite",
			when:  atNight,
			intros: []string{
				"Escureceu e {@} percebeu que não sabe mais onde fica a saída do mato. Tem uma luz piscando lá longe e um uivo do outro lado",
				"A lanterna do celular de {@} morreu no meio da trilha. No escuro total, dá pra ouvir um uivo e ver uma luzinha piscando",
			},
			choices: []choice{
				{label: "💡 Seguir a luz", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"Era uma nuvem de vaga-lume! {@} encheu o pote e ainda achou o caminho de volta"}, effects: []effect{gain(3, "vaga-lume")}},
					{weight: 40, texts: []string{"Era um caçador com lanterna, que levou um susto e deu um tiro pro alto. {@} correu pro lado errado"}, effects: []effect{injure(10, 25), delay(20 * time.Minute)}},
					{weight: 20, texts: []string{"Era a Mula sem Cabeça. Ninguém acredita, mas {@} jura que viu, e voltou com um encosto de lembrança"}, effects: []effect{injure(10, 20), enchant("encosto")}},
				}},
				{label: "🐺 Ir na direção do uivo", risk: highRisk, endings: []ending{
					{weight: 35, texts: []string{"Era só um cachorro perdido, que guiou {@} até a estrada"}},
					{weight: 15, texts: []string{"Era um filhote perdido, que guiou {@} até a estrada e não desgrudou mais"}, effects: []effect{gain(1, "filhote-de-cachorro")}},
					{weight: 40, texts: []string{"Era um lobo-guará, que olhou pra {@} com pena e foi embora. No susto, {@} rolou um barranco"}, effects: []effect{injure(15, 30)}},
					{weight: 10, texts: []string{"Era lua cheia, e aquilo não era cachorro"}, effects: []effect{injure(50, 80)}},
				}},
				{label: "🔥 Fazer fogueira e esperar o dia", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} acendeu uma fogueira, esperou o sol nascer e voltou com uns pernilongos de lembrança"}, effects: []effect{delay(30 * time.Minute), gain(3, "pernilongo")}},
				}},
				{label: "🛵 Chamar um mototáxi", risk: lowRisk, price: cost{percent: 2, floor: 30}, endings: []ending{
					{weight: 100, texts: []string{"O mototáxi apareceu no meio do mato em 7 minutos. Ninguém sabe como"}},
				}},
			},
			idle: ending{texts: []string{"{@} ficou parado no escuro até o dia clarear"}, effects: []effect{delay(30 * time.Minute)}},
		},
		{
			id:     "mato-redemoinho-do-saci",
			title:  "O redemoinho do Saci",
			rarity: legendaryEvent,
			intros: []string{
				"Um redemoinho de poeira passou rodando pelo mato e arrancou a rede da mão de {@}. Lá de dentro vem uma risada e cheiro de cachimbo",
				"Do nada, um redemoinho levantou as folhas em volta de {@}. Quem conhece sabe: é o Saci, e ele tá de bom humor",
			},
			choices: []choice{
				{label: "🍾 Jogar uma garrafa no redemoinho", risk: midRisk, endings: []ending{
					{weight: 10, texts: []string{"PEGOU! O Saci tá preso na garrafa, xingando em tupi e pedindo pra sair"}, effects: []effect{gain(1, "saci-na-garrafa")}},
					{weight: 50, texts: []string{"A garrafa voltou girando e acertou a testa de {@}"}, effects: []effect{injure(10, 20)}},
					{weight: 40, texts: []string{"O redemoinho cuspiu a rede de volta, toda amarrada em nó"}, effects: []effect{delay(10 * time.Minute)}},
				}},
				{label: "🏃 Correr atrás da rede", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} correu atrás do redemoinho e recuperou a rede, cheia de bicho dentro"}, effects: []effect{findLoot(), findLoot()}},
					{weight: 50, texts: []string{"O redemoinho sumiu no horizonte com a rede de {@}"}, effects: []effect{breakTool()}},
				}},
				{label: "🙏 Rezar um Pai-Nosso", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"A risada parou e a rede caiu do céu bem no pé de {@}"}},
					{weight: 40, texts: []string{"O Saci achou graça da reza, devolveu a rede e piscou o olho. Parece que ele vai voltar"}, effects: []effect{enchant("sorte"), plant("mato-saci-voltou", 12*time.Hour)}},
				}},
				{label: "🤝 Propor um trato", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"O Saci topou: ficou com a rede e deixou um saco cheio de bicho raro no lugar"}, effects: []effect{breakTool(), findLoot(), findLoot(), findLoot()}},
					{weight: 40, texts: []string{"O Saci ficou com a rede e deixou uma urucubaca de lembrança"}, effects: []effect{breakTool(), enchant("azar")}},
					{weight: 15, texts: []string{"O Saci gostou tanto de {@} que prometeu aparecer de novo"}, effects: []effect{plant("mato-saci-voltou", 6*time.Hour)}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, o redemoinho foi embora levando a rede"}, effects: []effect{breakTool()}},
		},
		{
			id:    "mato-saci-voltou",
			title: "O Saci voltou",
			stage: true,
			intros: []string{
				"O Saci que {@} conheceu no redemoinho voltou, sentado num galho, balançando a única perna e fumando o cachimbo: \"Lembra de mim?\"",
			},
			choices: []choice{
				{label: "🌟 Pedir um desejo", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O Saci estalou os dedos e a rede de {@} encheu de bicho"}, effects: []effect{findLoot(), findLoot(), findLoot()}},
					{weight: 30, texts: []string{"O desejo foi atendido do jeito do Saci: {@} pediu riqueza e ganhou um saco de tampinha"}, effects: []effect{gain(10, "tampinha")}},
					{weight: 20, texts: []string{"O Saci riu, pegou o dinheiro do bolso de {@} e sumiu num redemoinho"}, effects: []effect{lose(3, 8)}},
				}},
				{label: "🍫 Dar um chocolate", risk: lowRisk, uses: "chocolate", endings: []ending{
					{weight: 100, texts: []string{"O Saci amou o chocolate e prometeu dar uma mãozinha pra {@} no mato"}, effects: []effect{enchant("maoboa"), enchant("faro")}},
				}},
				{label: "🙈 Fingir que não viu", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"O Saci ficou ofendido e amarrou os cadarços de {@} um no outro"}, effects: []effect{delay(10 * time.Minute)}},
				}},
			},
		},
		{
			id:    "mato-onca-pintada",
			title: "Os olhos no galho",
			intros: []string{
				"Um silêncio esquisito tomou conta do mato. Em cima de um galho, uma onça-pintada tá olhando fixo pra {@}",
				"Os passarinhos pararam de cantar de uma vez. {@} olhou pra cima e deu de cara com uma onça deitada no tronco, abanando o rabo",
			},
			choices: []choice{
				{label: "🧍 Ficar grande e encarar", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"{@} abriu os braços, estufou o peito e a onça achou que não valia a pena"}},
					{weight: 45, texts: []string{"A onça não se impressionou e deu uma patada de aviso"}, effects: []effect{injure(15, 35)}},
				}},
				{label: "🏃 Sair correndo", risk: highRisk, endings: []ending{
					{weight: 45, favor: func(p Profession) float64 { return p.escapeBonus }, texts: []string{"{@} correu mais que o vento e só parou na estrada, sem a onça"}},
					{weight: 40, texts: []string{"Ninguém corre mais que onça. Ela alcançou {@} em três pulos"}, effects: []effect{injure(25, 45)}},
					{weight: 7, texts: []string{"Na correria, {@} tropeçou e a onça brincou de gato e rato"}, effects: []effect{injure(45, 60)}},
					{weight: 8, texts: []string{"A onça tava com fome, e {@} virou o almoço dela"}, effects: []effect{perish()}},
				}},
				{label: "🔙 Recuar bem devagar", risk: lowRisk, endings: []ending{
					{weight: 80, texts: []string{"{@} foi andando de costas, passinho por passinho, até a onça sumir de vista"}, effects: []effect{delay(10 * time.Minute)}},
					{weight: 20, texts: []string{"Recuando, {@} quase pisou num bicho raro que tava escondido no capim"}, effects: []effect{findLoot()}},
				}},
				{label: "🔫 Dar um tiro pro alto", risk: midRisk, needs: armedUp, endings: []ending{
					{weight: 70, texts: []string{"A onça fugiu, e a bicharada assustada saiu correndo direto pra rede de {@}"}, effects: []effect{findLoot(), stain()}},
					{weight: 30, texts: []string{"O tiro espantou a onça e chamou o Ibama, que tava na trilha ao lado"}, effects: []effect{nabbed()}},
				}},
				{label: "🩺 Ver se ela tá machucada", risk: lowRisk, needs: isJob("veterinário"), endings: []ending{
					{weight: 100, texts: []string{"{@} percebeu que ela tava com espinho na pata, tirou com cuidado e a onça foi embora mansinha. O pessoal da reserva deu uma ajuda de custo"}, effects: []effect{enchant("maoboa"), earn(40, 100)}},
				}},
			},
			idle: ending{texts: []string{"{@} congelou. A onça bocejou, se espreguiçou e foi dormir em outro galho"}},
		},
		{
			id:    "mato-armadilha-de-cacador",
			title: "O laço no tornozelo",
			intros: []string{
				"{@} pisou numa armadilha de caçador escondida debaixo das folhas, e o laço apertou o tornozelo",
				"Um estalo, um puxão e {@} ficou pendurado de cabeça pra baixo num laço amarrado na árvore",
			},
			choices: []choice{
				{label: "🦶 Puxar o pé na marra", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"O pé saiu, mas a meia e um pedaço da pele ficaram no laço"}, effects: []effect{injure(5, 12)}},
					{weight: 45, texts: []string{"O laço apertou mais ainda e o tornozelo de {@} ficou roxo"}, effects: []effect{injure(15, 30)}},
				}},
				{label: "🪢 Desfazer o nó com calma", risk: lowRisk, endings: []ending{
					{weight: 75, texts: []string{"{@} levou meia hora, mas desfez o nó sem se machucar"}, effects: []effect{delay(10 * time.Minute)}},
					{weight: 25, texts: []string{"Do lado tinha um tatu preso em outra armadilha. {@} soltou o bichinho, que saiu agradecido"}, effects: []effect{enchant("sorte")}},
				}},
				{label: "📢 Gritar por socorro", risk: lowRisk, endings: []ending{
					{weight: 50, texts: []string{"Um lenhador que passava ouviu e cortou o laço com o facão"}},
					{weight: 50, texts: []string{"Quem ouviu foi o dono da armadilha, que ainda cobrou o laço estragado"}, effects: []effect{lose(2, 5)}},
				}},
				{label: "🔎 Desarmar as outras armadilhas", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} desarmou cinco armadilhas e achou o esconderijo do caçador, cheio de bicho"}, effects: []effect{findLoot(), findLoot(), enchant("justica")}},
					{weight: 50, texts: []string{"A segunda armadilha era de mola, e pegou {@} na mão"}, effects: []effect{injure(10, 20)}},
				}},
				{label: "🌾 Soltar como gente do campo", risk: lowRisk, needs: inArea(areaCampo), endings: []ending{
					{weight: 100, texts: []string{"Quem é da roça conhece laço de caçador. {@} soltou em dois segundos e ainda usou a armadilha pra pegar bicho"}, effects: []effect{findLoot()}},
				}},
			},
		},
		{
			id:    "mato-trilheiros-perdidos",
			title: "Os trilheiros perdidos",
			intros: []string{
				"Um grupo de trilheiros de bota nova e mochila de marca tá perdido no mato. O guia sumiu e eles olham pra {@} com cara de esperança",
				"Cinco turistas de bastão de caminhada apareceram na frente de {@}, sem sinal no celular e sem água: 'moço, onde fica a cachoeira?'",
			},
			choices: []choice{
				{label: "🧭 Levar até a estrada", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"Na estrada, os trilheiros fizeram uma vaquinha de agradecimento pra {@}"}, effects: []effect{earn(30, 80)}},
					{weight: 30, texts: []string{"Eles não tinham dinheiro, mas dividiram o lanche natural com {@}"}, effects: []effect{heal(10)}},
				}},
				{label: "💸 Cobrar o resgate adiantado", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"Pagaram no Pix sem pensar duas vezes, e {@} achou a estrada rapidinho"}, effects: []effect{earn(60, 140)}},
					{weight: 30, texts: []string{"{@} cobrou, mas também não sabia o caminho. Passaram a noite todos juntos no mato"}, effects: []effect{delay(30 * time.Minute)}},
					{weight: 15, texts: []string{"Os trilheiros pagaram e deixaram uma avaliação de uma estrela pra {@} no mapa da trilha"}, effects: []effect{earn(30, 60), enchant("olhogordo")}},
				}},
				{label: "🥁 Mandar eles fazerem barulho", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"A barulheira espantou os bichos direto pra rede de {@}"}, effects: []effect{findLoot(), findLoot()}},
					{weight: 50, texts: []string{"A barulheira espantou tudo pra longe e ainda irritou um marimbondo"}, effects: []effect{injure(5, 12)}},
				}},
				{label: "🚒 Resgatar como profissional", risk: lowRisk, needs: isJob("bombeiro", "salva-vidas", "socorrista"), endings: []ending{
					{weight: 100, texts: []string{"{@} fez o resgate certinho, com corda e tudo. Os trilheiros postaram a história e mandaram um agrado"}, effects: []effect{earn(80, 160), enchant("abrecaminho")}},
				}},
			},
		},
		{
			id:     "mato-escaravelho-dourado",
			title:  "O brilho dourado",
			rarity: rareEvent,
			intros: []string{
				"Um brilho dourado passou zunindo e pousou num tronco: é um escaravelho de ouro, desses que os antigos diziam que traz fortuna",
				"{@} ouviu um zumbido metálico e viu um besouro dourado andando devagar numa folha, brilhando igual joia",
			},
			choices: []choice{
				{label: "🥅 Jogar a rede", risk: midRisk, endings: []ending{
					{weight: 15, texts: []string{"Na mosca! O escaravelho de ouro caiu direitinho na rede de {@}"}, effects: []effect{gain(1, "escaravelho-de-ouro")}},
					{weight: 55, texts: []string{"O dourado escapou, mas um besouro grandão caiu no lugar dele"}, effects: []effect{gain(1, "besouro-rinoceronte")}},
					{weight: 30, texts: []string{"A rede enganchou no tronco e rasgou de ponta a ponta"}, effects: []effect{breakTool()}},
				}},
				{label: "🤲 Esperar ele pousar na mão", risk: lowRisk, endings: []ending{
					{weight: 10, texts: []string{"Depois de muita paciência, o escaravelho pousou na palma de {@} e ficou"}, effects: []effect{gain(1, "escaravelho-de-ouro")}},
					{weight: 50, texts: []string{"Ele pousou, deu uma voltinha na mão de {@} e foi embora. Dizem que é bênção"}, effects: []effect{enchant("sorte")}},
					{weight: 40, texts: []string{"O escaravelho foi embora sem dar confiança"}},
				}},
				{label: "🍯 Atrair com melado", risk: lowRisk, uses: "garrafa-de-melado", endings: []ending{
					{weight: 40, texts: []string{"O cheiro do melado foi irresistível e o escaravelho de ouro veio direto pro pote"}, effects: []effect{gain(1, "escaravelho-de-ouro")}},
					{weight: 60, texts: []string{"O dourado não veio, mas o melado atraiu bicho do mato inteiro"}, effects: []effect{findLoot(), findLoot()}},
				}},
				{label: "💡 Ligar a armadilha luminosa", risk: lowRisk, needs: holds("armadilha-luminosa"), endings: []ending{
					{weight: 50, texts: []string{"A luz atraiu o escaravelho, que entrou sozinho na armadilha"}, effects: []effect{gain(1, "escaravelho-de-ouro")}},
					{weight: 50, texts: []string{"A luz atraiu um monte de bicho, menos o dourado"}, effects: []effect{findLoot(), findLoot()}},
				}},
			},
			idle: ending{texts: []string{"Enquanto {@} pensava, o escaravelho abriu as asas e sumiu no meio das árvores"}},
		},
	},
}
