package economy

import "time"

var rareEvents = eventBook{
	name:   "✨ Raros",
	places: nil,
	scenarios: []scenario{
		{
			id:     "raro-mapa-do-tesouro",
			title:  "O mapa do tesouro",
			rarity: legendaryEvent,
			intros: []string{
				"Um papel amarelado voou com o vento e grudou na cara de {@}. Era metade de um mapa do tesouro, com um X vermelho e a assinatura de um tal Capitão Barba-Ruiva",
				"Dentro de uma garrafa largada no caminho, {@} achou um mapa antigo rasgado ao meio. Na borda, alguém escreveu: \"a outra metade aparece pra quem é digno\"",
			},
			choices: []choice{
				{label: "🗺️ Guardar e seguir as pistas", risk: midRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} guardou o mapa no bolso de dentro e jurou achar a outra metade"}, effects: []effect{plant("raro-mapa-outra-metade", 2*time.Hour)}},
					{weight: 30, texts: []string{"Olhando direito, era o mapa de uma pizzaria do bairro com o X no forno. Pelo menos vinha um cupom"}, effects: []effect{gain(1, "pizza")}},
				}},
				{label: "💰 Vender pro colecionador", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"Um colecionador de antiguidades pagou na hora, sem pechinchar. Rápido demais pro gosto de {@}"}, effects: []effect{earn(150, 300)}},
				}},
				{label: "🔥 Queimar o mapa", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"Quando o mapa queimou, a fumaça fez a forma de uma caveira sorrindo. Melhor assim"}, effects: []effect{enchant("protecao")}},
					{weight: 40, texts: []string{"O fantasma do capitão não gostou nada de ver o mapa virar cinza"}, effects: []effect{enchant("encosto")}},
				}},
			},
		},
		{
			id:     "raro-mapa-outra-metade",
			title:  "A outra metade do mapa",
			rarity: legendaryEvent,
			stage:  true,
			intros: []string{
				"No meio da correria, {@} pisou num pedaço de papel velho: era a outra metade do mapa do Capitão Barba-Ruiva! Juntando as duas, o X aponta pra um lugar bem perto",
			},
			choices: []choice{
				{label: "⛏️ Cavar no X", risk: midRisk, endings: []ending{
					{weight: 75, texts: []string{"A pá bateu em madeira. Era a tampa de um baú com uma caveira pintada"}, next: "raro-mapa-bau"},
					{weight: 25, texts: []string{"{@} cavou um buraco de dois metros e achou só um cano da companhia de água. A água jorrou"}, effects: []effect{injure(5, 15), delay(30 * time.Minute)}},
				}},
				{label: "🤝 Chamar a galera e dividir", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"A galera cavou junto e achou o baú. Dividido entre todo mundo, ainda sobrou uma boa grana pra {@}"}, effects: []effect{earn(250, 500)}},
				}},
				{label: "🧐 Vender o mapa completo", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"Com as duas metades juntas, o mapa valia uma fortuna pros colecionadores"}, effects: []effect{earn(300, 600)}},
				}},
			},
		},
		{
			id:     "raro-mapa-bau",
			title:  "O baú do Capitão Barba-Ruiva",
			rarity: legendaryEvent,
			stage:  true,
			intros: []string{
				"O baú tá ali, desenterrado, com cadeado enferrujado e uma inscrição: \"quem abrir sem respeito, morre\". {@} engoliu seco",
			},
			choices: []choice{
				{label: "🔓 Abrir na marra", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"O cadeado cedeu e o baú tava CHEIO de moeda de ouro"}, effects: []effect{earn(800, 1500), gain(1, "bau-do-tesouro")}},
					{weight: 15, texts: []string{"No fundo do baú, embrulhada em veludo, tinha uma caixinha antiga que vibra sozinha"}, effects: []effect{gain(1, "relicario")}},
					{weight: 30, texts: []string{"Era armadilha: uma flecha enferrujada saiu da fechadura direto no ombro de {@}"}, effects: []effect{injure(30, 60)}},
					{weight: 10, texts: []string{"A maldição do capitão era real. {@} abriu o baú e o baú levou {@}"}, effects: []effect{perish()}},
				}},
				{label: "🙏 Rezar antes de abrir", risk: midRisk, endings: []ending{
					{weight: 70, texts: []string{"Com todo o respeito, o baú abriu sozinho. Dentro, um tesouro e um bilhete: \"pra quem tem modos\""}, effects: []effect{earn(600, 1000), enchant("protecao")}},
					{weight: 30, texts: []string{"A reza não adiantou muito: o baú tava cheio de areia e de um caranguejo bravo"}, effects: []effect{gain(1, "caranguejo"), injure(5, 10)}},
				}},
				{label: "🏃 Deixar o baú quieto", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} enterrou o baú de novo e nunca contou pra ninguém. Às vezes, de noite, ainda ouve um \"obrigado\""}, effects: []effect{enchant("sorte")}},
				}},
				{label: "✝️ Benzer o baú primeiro", risk: lowRisk, needs: inArea(areaFe), endings: []ending{
					{weight: 100, texts: []string{"A maldição se desfez em fumaça com a reza forte de {@}, e o tesouro saiu limpinho"}, effects: []effect{earn(900, 1500), gain(1, "bau-do-tesouro")}},
				}},
			},
		},
		{
			id:     "raro-saci",
			title:  "O redemoinho do Saci",
			rarity: legendaryEvent,
			intros: []string{
				"Um redemoinho de poeira passou rodando e, quando parou, tinha um menino de uma perna só, gorro vermelho e cachimbo, rindo da cara de {@}",
				"Os cadarços de {@} se amarraram sozinhos, o café derramou e um assobio fino veio do mato: é o Saci, pulando numa perna só",
			},
			choices: []choice{
				{label: "🎩 Tentar pegar o gorro", risk: highRisk, endings: []ending{
					{weight: 15, texts: []string{"{@} agarrou o gorro! Sem ele, o Saci teve que cumprir um desejo, e o desejo foi dinheiro"}, effects: []effect{earn(800, 1500)}},
					{weight: 50, texts: []string{"O Saci deu um nó no cabelo de {@}, sumiu no redemoinho e deixou uma urucubaca de presente"}, effects: []effect{enchant("azar")}},
					{weight: 35, texts: []string{"O Saci passou uma rasteira e {@} beijou o chão"}, effects: []effect{injure(15, 30)}},
				}},
				{label: "🍾 Prender numa garrafa", risk: highRisk, endings: []ending{
					{weight: 20, texts: []string{"Com uma garrafa e muita rapidez, {@} prendeu o Saci! Ele xinga lá de dentro sem parar"}, effects: []effect{gain(1, "saci-na-garrafa")}},
					{weight: 50, texts: []string{"O Saci entrou na garrafa, bebeu a cachaça que tinha dentro e saiu rindo"}, effects: []effect{enchant("lerdeza")}},
					{weight: 30, texts: []string{"A garrafa estourou na mão de {@}"}, effects: []effect{injure(10, 25)}},
				}},
				{label: "🚬 Oferecer fumo pro cachimbo", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"O Saci acendeu o cachimbo, deu uma piscada e soprou fumaça da sorte em {@}"}, effects: []effect{enchant("sorte"), earn(100, 250)}},
					{weight: 40, texts: []string{"O Saci aceitou o fumo e levou junto um pé de meia de {@}, só de pirraça"}},
				}},
				{label: "🙏 Rezar um pai-nosso", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O Saci tapou os ouvidos e sumiu no redemoinho"}},
					{weight: 30, texts: []string{"O Saci respeitou a reza e deixou {@} de corpo fechado"}, effects: []effect{enchant("protecao")}},
				}},
			},
		},
		{
			id:     "raro-curupira",
			title:  "As pegadas do Curupira",
			rarity: rareEvent,
			intros: []string{
				"Num atalho pelo matagal, {@} achou pegadas viradas pra trás. Antes de entender, um menino de cabelo de fogo apareceu em cima de um galho: o Curupira",
				"{@} pegou um caminho pelo mato pra cortar volta e se perdeu. Um assobio, um vulto ruivo e pegadas ao contrário: é o Curupira",
			},
			choices: []choice{
				{label: "🎁 Deixar um presente", risk: lowRisk, price: cost{percent: 2, floor: 50}, endings: []ending{
					{weight: 60, texts: []string{"O Curupira aceitou o presente e abençoou as mãos de {@} pra qualquer extração"}, effects: []effect{enchant("maoboa"), enchant("achado")}},
					{weight: 40, texts: []string{"Feliz da vida, o Curupira mostrou o caminho de volta e um atalho cheio de coisa boa"}, effects: []effect{enchant("abrecaminho"), findLoot()}},
				}},
				{label: "🏃 Seguir as pegadas", risk: highRisk, endings: []ending{
					{weight: 30, texts: []string{"Seguindo as pegadas ao contrário, {@} achou o esconderijo do Curupira cheio de tesouro da mata"}, effects: []effect{gain(1, "esmeralda", "ouro", "favo-de-mel", "geodo")}},
					{weight: 45, texts: []string{"As pegadas levaram {@} cada vez mais pra dentro do mato. Foram horas no meio dos espinhos"}, effects: []effect{injure(15, 35), delay(45 * time.Minute)}},
					{weight: 25, texts: []string{"O Curupira ficou bravo com a perseguição e rogou panema em {@}"}, effects: []effect{enchant("panema")}},
				}},
				{label: "🙇 Pedir licença pra passar", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O Curupira fez uma reverência e abriu caminho"}},
					{weight: 30, texts: []string{"O Curupira gostou da educação e assobiou um segredo: onde ficam as coisas boas"}, effects: []effect{enchant("faro")}},
				}},
			},
		},
		{
			id:     "raro-mula-sem-cabeca",
			title:  "A Mula sem cabeça",
			rarity: rareEvent,
			when:   atNight,
			intros: []string{
				"Madrugada, barulho de casco e cheiro de queimado: uma mula sem cabeça, soltando fogo pelo pescoço, galopou na direção de {@}",
				"Relincho sem cabeça, fogo no lugar do pescoço e freio de ferro tilintando: a Mula sem cabeça escolheu {@} pra assombrar hoje",
			},
			choices: []choice{
				{label: "🏃 Correr em zigue-zague", risk: midRisk, endings: []ending{
					{weight: 60, favor: func(p Profession) float64 { return p.escapeBonus }, texts: []string{"{@} correu em zigue-zague até a mula se confundir e trombar num poste"}},
					{weight: 40, texts: []string{"A mula passou rente e chamuscou as costas de {@}"}, effects: []effect{injure(15, 35)}},
				}},
				{label: "⛓️ Arrancar o freio de ferro", risk: highRisk, endings: []ending{
					{weight: 35, texts: []string{"{@} arrancou o freio e a maldição quebrou: a mula virou uma moça, que agradeceu com tudo que tinha"}, effects: []effect{earn(400, 800), enchant("prosperidade")}},
					{weight: 45, texts: []string{"A mula deu um coice de fogo em {@} antes de sumir"}, effects: []effect{injure(30, 55)}},
					{weight: 10, texts: []string{"O fogo pegou de vez. Não sobrou muito de {@}"}, effects: []effect{perish()}},
					{weight: 10, texts: []string{"{@} ficou com o freio na mão e uma maldição de brinde"}, effects: []effect{enchant("encosto")}},
				}},
				{label: "🫣 Esconder unhas e dentes", risk: lowRisk, endings: []ending{
					{weight: 80, texts: []string{"Diz a lenda que a mula ataca quem mostra unha e dente. {@} fechou a boca e as mãos, e ela passou reto"}},
					{weight: 20, texts: []string{"{@} escondeu tudo, mas espirrou na hora errada"}, effects: []effect{injure(5, 15)}},
				}},
			},
		},
		{
			id:     "raro-boitata",
			title:  "A cobra de fogo",
			rarity: rareEvent,
			intros: []string{
				"Uma luz serpenteando no escuro chamou a atenção de {@}. Não era vaga-lume: era uma cobra de fogo gigante, de olhos enormes. O Boitatá",
				"O mato começou a brilhar azul e uma cobra de fogo do tamanho de um ônibus levantou a cabeça perto de {@}",
			},
			choices: []choice{
				{label: "🙈 Fechar os olhos e esperar", risk: lowRisk, endings: []ending{
					{weight: 75, texts: []string{"Quem encara o Boitatá fica cego, todo mundo sabe. {@} fechou os olhos, contou até cem e ele foi embora"}},
					{weight: 25, texts: []string{"O Boitatá passou pertinho e deixou o corpo de {@} quente e protegido"}, effects: []effect{enchant("protecao")}},
				}},
				{label: "💧 Jogar água nele", risk: highRisk, endings: []ending{
					{weight: 30, texts: []string{"A água fez o Boitatá espirrar, e ele cuspiu umas pedras brilhantes que tinha engolido"}, effects: []effect{gain(1, "diamante", "rubi", "esmeralda", "safira")}},
					{weight: 55, texts: []string{"O Boitatá não gostou do banho e cuspiu fogo"}, effects: []effect{injure(25, 50)}},
					{weight: 15, texts: []string{"O Boitatá ficou tão ofendido que deixou {@} com cheiro de churrasco e uma lerdeza de cobra"}, effects: []effect{enchant("lerdeza"), injure(10, 20)}},
				}},
				{label: "👀 Olhar bem nos olhos", risk: highRisk, endings: []ending{
					{weight: 35, texts: []string{"{@} encarou o Boitatá e ele, impressionado com a coragem, deixou uma escama que virou ouro no bolso"}, effects: []effect{earn(300, 600)}},
					{weight: 55, texts: []string{"A luz deixou {@} sem enxergar nada por uma hora inteira"}, effects: []effect{delay(time.Hour), injure(10, 20)}},
					{weight: 10, texts: []string{"O Boitatá engoliu {@} de uma vez só"}, effects: []effect{perish()}},
				}},
				{label: "🔥 Pedir fogo pro cigarro", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O Boitatá achou graça na ousadia, acendeu o cigarro e foi embora rindo"}, effects: []effect{enchant("sorte")}},
					{weight: 50, texts: []string{"O fogo do Boitatá acendeu o cigarro, a sobrancelha e o cabelo de {@}"}, effects: []effect{injure(10, 25)}},
				}},
			},
		},
		{
			id:     "raro-disco-voador",
			title:  "Contato imediato",
			rarity: legendaryEvent,
			intros: []string{
				"Uma luz forte, um zumbido e todos os celulares da rua desligaram. Um disco voador pousou na frente de {@} e um ETzinho verde acenou",
				"O céu ficou verde, as vacas do sítio do lado sumiram e uma nave prateada parou flutuando em cima de {@}",
			},
			choices: []choice{
				{label: "👽 Entrar na nave", risk: highRisk, endings: []ending{
					{weight: 15, texts: []string{"Os ETs deram uma volta na galáxia com {@} e, na despedida, deixaram a nave de presente porque iam trocar de modelo"}, effects: []effect{gain(1, "disco-voador")}},
					{weight: 35, texts: []string{"A nave levou {@} por seis horas. Na volta, sem lembrar de nada e com o cabelo cheirando a ozônio, ainda tinha dinheiro no bolso"}, effects: []effect{delay(time.Hour), earn(200, 500)}},
					{weight: 35, texts: []string{"Fizeram exames estranhos em {@}. Muito estranhos"}, effects: []effect{injure(20, 40)}},
					{weight: 15, texts: []string{"Os ETs pagaram {@} em ouro pra servir de guia turístico da Terra"}, effects: []effect{earn(800, 1500)}},
				}},
				{label: "📸 Filmar tudo", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O vídeo de {@} foi parar no programa de domingo à noite e rendeu uma bolada"}, effects: []effect{earn(300, 700)}},
					{weight: 50, texts: []string{"O vídeo saiu todo desfocado e ninguém acreditou. Virou motivo de chacota pra {@}"}, effects: []effect{enchant("olhogordo")}},
				}},
				{label: "☕ Oferecer um cafezinho", risk: lowRisk, endings: []ending{
					{weight: 35, texts: []string{"O ET adorou café brasileiro e deixou de presente uma pedra que caiu do céu"}, effects: []effect{gain(1, "meteorito")}},
					{weight: 65, texts: []string{"O ET derrubou café no painel da nave e foi embora xingando em marciano"}},
				}},
				{label: "🏃 Sair correndo", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} correu tanto que bateu o recorde do bairro. Ninguém acreditou na história depois"}},
				}},
			},
		},
		{
			id:     "raro-bilhete-da-mega",
			title:  "O bilhete da Mega",
			rarity: legendaryEvent,
			intros: []string{
				"{@} achou um bilhete de loteria no chão, conferiu por curiosidade e quase desmaiou: seis números certinhos",
				"Num bolso de calça velha, {@} achou um bilhete da Mega esquecido. O sorteio foi ontem. E bateu",
			},
			choices: []choice{
				{label: "🏦 Correr pra lotérica", risk: midRisk, endings: []ending{
					{weight: 45, texts: []string{"A moça da lotérica conferiu, arregalou os olhos e chamou o gerente. {@} saiu de lá com a vida resolvida"}, effects: []effect{earn(800, 1500)}},
					{weight: 35, texts: []string{"Era do concurso de 2019. A moça riu tanto que chamou os colegas pra ver"}},
					{weight: 20, texts: []string{"O dono do bilhete apareceu na lotérica bem na hora e partiu pra cima de {@}"}, effects: []effect{injure(5, 15)}},
				}},
				{label: "📢 Procurar o dono", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O dono, um porteiro aposentado, chorou abraçado com {@} e fez questão de dividir o prêmio"}, effects: []effect{earn(400, 800), enchant("sorte")}},
					{weight: 30, texts: []string{"Ninguém apareceu e o prazo do resgate venceu. Pelo menos a consciência tá limpa"}, effects: []effect{enchant("protecao")}},
				}},
				{label: "🖼️ Guardar de lembrança", risk: lowRisk, endings: []ending{
					{weight: 40, texts: []string{"{@} emoldurou o bilhete. Colecionador paga caro por um desses"}, effects: []effect{gain(1, "bilhete-premiado")}},
					{weight: 60, texts: []string{"A mãe de {@} lavou a calça com o bilhete no bolso"}},
				}},
			},
		},
		{
			id:     "raro-heranca",
			title:  "A herança do tio-avô",
			rarity: rareEvent,
			intros: []string{
				"Um advogado de terno brilhante procurou {@}: \"meus pêsames, seu tio-avô Belarmino faleceu e deixou tudo pra você\". {@} nunca ouviu falar de nenhum Belarmino",
				"Chegou uma carta registrada pra {@}: herança de um parente distante que morreu sem filhos. Basta assinar uns papéis",
			},
			choices: []choice{
				{label: "✍️ Assinar os papéis", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Tudo certinho: o tio Belarmino deixou uma conta gorda e um papagaio"}, effects: []effect{earn(300, 600), gain(1, "papagaio")}},
					{weight: 30, texts: []string{"A herança veio junto com as dívidas do tio. E eram muitas"}, effects: []effect{lose(5, 12)}},
					{weight: 20, texts: []string{"Era golpe do falso advogado: os papéis eram uma procuração pra mexer na conta de {@}"}, effects: []effect{lose(10, 20)}},
				}},
				{label: "🔍 Pedir os documentos", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"Os documentos eram reais. {@} recebeu a herança com toda a segurança"}, effects: []effect{earn(300, 550)}},
					{weight: 40, texts: []string{"O tal advogado disse que voltava amanhã com os papéis e nunca mais apareceu"}},
				}},
				{label: "🙅 Recusar, é bom demais", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} recusou. Meses depois saiu no jornal que a herança era real e foi toda pro gato do tio"}},
				}},
				{label: "⚖️ Conferir com olho de quem é da lei", risk: lowRisk, needs: inArea(areaSeguranca), endings: []ending{
					{weight: 100, texts: []string{"{@} achou um furo no inventário, garantiu a parte certa e ainda ficou com um terreno do tio no papel"}, effects: []effect{earn(500, 900)}},
				}},
			},
		},
		{
			id:     "raro-encruzilhada",
			title:  "O pacto na encruzilhada",
			rarity: legendaryEvent,
			when:   atNight,
			intros: []string{
				"Meia-noite em ponto, numa encruzilhada deserta, um homem de terno vermelho e chapéu apareceu do nada na frente de {@}: \"vamos fazer negócio?\"",
				"O relógio bateu meia-noite e uma figura elegante de cartola, encostada no poste da encruzilhada, chamou {@} pelo nome. E {@} nunca disse o nome",
			},
			choices: []choice{
				{label: "🤝 Fechar o pacto", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"O homem sorriu, apertou a mão de {@} e o dinheiro apareceu no bolso. Ele avisou que um dia volta pra cobrar"}, effects: []effect{earn(800, 1500), enchant("prosperidade")}},
					{weight: 30, texts: []string{"O pacto tinha letras miúdas. Muito miúdas"}, effects: []effect{enchant("miseria"), enchant("encosto")}},
					{weight: 20, texts: []string{"O homem quis uma prova de lealdade, e a prova doeu"}, effects: []effect{injure(30, 60)}},
					{weight: 10, texts: []string{"O homem cobrou na hora. Com a alma de {@}"}, effects: []effect{perish()}},
				}},
				{label: "✝️ Fazer o sinal da cruz", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O homem fez uma careta, tirou o chapéu num cumprimento irônico e sumiu num cheiro de enxofre"}, effects: []effect{enchant("protecao")}},
					{weight: 30, texts: []string{"O homem riu alto, disse que não era nada pessoal e sumiu"}},
				}},
				{label: "🪗 Desafiar pra um duelo de sanfona", risk: highRisk, endings: []ending{
					{weight: 30, texts: []string{"{@} tocou tão bem que o homem ficou de queixo caído, pagou a aposta e largou a sanfona dele ali mesmo"}, effects: []effect{earn(1000, 1500), gain(1, "sanfona")}},
					{weight: 50, texts: []string{"O homem tocou melhor, claro. {@} perdeu o duelo e a sorte"}, effects: []effect{enchant("azar"), lose(5, 10)}},
					{weight: 10, texts: []string{"No último acorde, uma tecla voou e acertou o olho de {@}"}, effects: []effect{injure(20, 40)}},
					{weight: 10, texts: []string{"Quem perde duelo com ele vai junto. E {@} foi"}, effects: []effect{perish()}},
				}},
				{label: "📿 Expulsar com reza forte", risk: midRisk, needs: inArea(areaFe), endings: []ending{
					{weight: 70, texts: []string{"A reza de {@} foi tão forte que o homem saiu correndo e deixou a carteira cair"}, effects: []effect{earn(500, 1000), enchant("protecao")}},
					{weight: 30, texts: []string{"O homem aguentou a reza e deixou um encosto de recado"}, effects: []effect{enchant("encosto")}},
				}},
			},
		},
		{
			id:     "raro-genio",
			title:  "O gênio da lâmpada",
			rarity: legendaryEvent,
			intros: []string{
				"{@} chutou uma lâmpada velha largada no caminho e dela saiu uma fumaça azul que virou um gênio de bigode: \"um desejo, só um, e rápido que eu tô no horário de almoço\"",
				"Esfregando uma lâmpada achada no meio da tralha, {@} soltou um gênio meio sonolento. Ele bocejou: \"fala o desejo aí\"",
			},
			choices: []choice{
				{label: "💰 Desejar dinheiro", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"PLIM! Choveu dinheiro em cima de {@}"}, effects: []effect{earn(800, 1500)}},
					{weight: 25, texts: []string{"PLIM! Choveu dinheiro... de brinquedo. O gênio riu muito"}, effects: []effect{gain(10, "dinheiro-de-brinquedo")}},
					{weight: 25, texts: []string{"PLIM! O dinheiro apareceu, mas era do cofre do banco do outro lado da rua. A polícia não achou graça"}, effects: []effect{earn(600, 1000), nabbed()}},
				}},
				{label: "❤️ Desejar saúde", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} sentiu o corpo inteiro renovado. Até a dor nas costas de 2014 sumiu"}, effects: []effect{heal(100), enchant("protecao")}},
				}},
				{label: "🍀 Desejar sorte", risk: lowRisk, endings: []ending{
					{weight: 80, texts: []string{"O gênio estalou os dedos e a vida de {@} virou do avesso, pro lado bom"}, effects: []effect{enchant("sorte"), enchant("prosperidade")}},
					{weight: 20, texts: []string{"O gênio entendeu \"morte\" em vez de \"sorte\" e, por via das dúvidas, deu azar"}, effects: []effect{enchant("azar")}},
				}},
				{label: "🪔 Desejar mais desejos", risk: highRisk, endings: []ending{
					{weight: 20, texts: []string{"O gênio, sem argumento, entregou a lâmpada pra {@} e foi se aposentar em Cabo Frio"}, effects: []effect{gain(1, "lampada-magica")}},
					{weight: 60, texts: []string{"\"Engraçadinho\", disse o gênio, e jogou uma praga antes de sumir"}, effects: []effect{enchant("miseria")}},
					{weight: 20, texts: []string{"O gênio prendeu {@} na lâmpada por uma hora, pra aprender a não ser espertinho"}, effects: []effect{delay(time.Hour)}},
				}},
			},
		},
		{
			id:     "raro-maquina-do-tempo",
			title:  "A máquina do tempo do Seu Zé",
			rarity: rareEvent,
			intros: []string{
				"Seu Zé do ferro-velho jura que transformou uma geladeira velha numa máquina do tempo, e tá chamando {@} pra ser a primeira cobaia",
				"\"Entra aí, é seguro\", disse o Seu Zé apontando pra uma geladeira cheia de fios e um volante de Fusca. A máquina do tempo tá pronta e ele quer que {@} teste primeiro",
			},
			choices: []choice{
				{label: "⏪ Voltar no tempo", risk: highRisk, endings: []ending{
					{weight: 35, texts: []string{"{@} voltou dez minutos no tempo e refez o dia com a sabedoria do futuro"}, effects: []effect{refresh(), earn(200, 400)}},
					{weight: 30, texts: []string{"A máquina mandou {@} pra 1998, e na volta veio uma lembrança no bolso"}, effects: []effect{gain(1, "tamagotchi", "album-da-copa", "fita-k7", "vhs", "game-boy")}},
					{weight: 25, texts: []string{"A geladeira deu um choque daqueles e {@} saiu de lá com o cabelo em pé"}, effects: []effect{injure(15, 35)}},
					{weight: 10, texts: []string{"{@} entrou num loop temporal e viveu o mesmo minuto por uma hora inteira"}, effects: []effect{delay(time.Hour)}},
				}},
				{label: "⏩ Ir pro futuro", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"No futuro, {@} viu o resultado da loteria e voltou correndo pra apostar"}, effects: []effect{earn(300, 600)}},
					{weight: 40, texts: []string{"O futuro era só boleto e fila. {@} voltou sem ânimo nenhum pra nada"}, effects: []effect{enchant("lerdeza")}},
					{weight: 20, texts: []string{"A máquina explodiu no meio da viagem"}, effects: []effect{injure(20, 40)}},
				}},
				{label: "🙅 Dizer que é loucura", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"O Seu Zé ficou chateado e testou ele mesmo. Voltou jurando que viu dinossauro"}},
					{weight: 30, texts: []string{"Dez segundos depois a geladeira explodiu. {@} agradeceu o próprio bom senso"}, effects: []effect{enchant("protecao")}},
				}},
				{label: "🔧 Ajustar a fiação antes", risk: midRisk, needs: inArea(areaObras, areaEscritorio), endings: []ending{
					{weight: 70, texts: []string{"Com a fiação arrumada por {@}, a máquina funcionou de verdade e o Seu Zé pagou a consultoria"}, effects: []effect{earn(300, 600), enchant("pressa")}},
					{weight: 30, texts: []string{"{@} trocou o fio azul pelo vermelho. Nunca troque o fio azul pelo vermelho"}, effects: []effect{injure(20, 40)}},
				}},
			},
		},
		{
			id:     "raro-boto",
			title:  "O moço de chapéu branco",
			rarity: rareEvent,
			when:   atNight,
			intros: []string{
				"Na festa, um moço de terno branco e um chapéu que ele não tira de jeito nenhum chamou {@} pra dançar. O sorriso é bonito demais, e ele cheira a rio",
				"Um forasteiro charmoso de chapéu branco apareceu do nada no forró e só tem olhos pra {@}. Dizem que é o Boto",
			},
			choices: []choice{
				{label: "💃 Dançar com ele", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} dançou a noite inteira e acordou com uma sorte esquisita grudada no corpo"}, effects: []effect{enchant("sorte"), earn(100, 300)}},
					{weight: 30, texts: []string{"{@} acordou na beira do rio, de sapato molhado e sem lembrar de nada"}, effects: []effect{delay(time.Hour)}},
					{weight: 20, texts: []string{"O moço sumiu no fim da festa. E a carteira de {@} sumiu junto"}, effects: []effect{lose(5, 10)}},
				}},
				{label: "🎩 Arrancar o chapéu dele", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"Embaixo do chapéu tinha um buraco de respirar de boto! Pra {@} guardar segredo, ele pagou bem"}, effects: []effect{earn(400, 800)}},
					{weight: 40, texts: []string{"O Boto ficou furioso e deu uma rabada de peixe na cara de {@}"}, effects: []effect{injure(10, 25)}},
					{weight: 20, texts: []string{"O Boto rogou praga de pescador: rio nenhum dá peixe pra {@} por um tempo"}, effects: []effect{enchant("panema")}},
				}},
				{label: "🏃 Ir embora cedo", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} foi embora antes da meia-noite. No dia seguinte, metade da festa tava contando história do Boto"}},
				}},
			},
		},
	},
}
