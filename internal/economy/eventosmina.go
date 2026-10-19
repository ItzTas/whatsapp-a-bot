package economy

import "time"

var miningEvents = eventBook{
	name:   "⛏️ Mina",
	places: []string{mining.command},
	scenarios: []scenario{
		{
			id:    "mina-estalo-no-teto",
			title: "Estalo no teto",
			intros: []string{
				"Um CRACK ecoou pela mina e caiu um punhado de terra na cabeça de {@}. O teto tá reclamando",
				"{@} tava no meio da picaretada quando o teto da galeria rangeu igual porta de filme de terror",
			},
			choices: []choice{
				{label: "🏃 Correr pra saída", risk: lowRisk, endings: []ending{
					{weight: 80, texts: []string{"{@} saiu correndo e o teto aguentou. Ficou a dúvida e o coração na boca"}},
					{weight: 20, texts: []string{"Na correria, {@} tropeçou no trilho e ralou o joelho"}, effects: []effect{injure(4, 10)}},
				}},
				{label: "🪵 Escorar com a picareta", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"A gambiarra segurou o teto e {@} ainda aproveitou pra cavar mais um pouco"}, effects: []effect{findLoot()}},
					{weight: 25, texts: []string{"O teto segurou. A picareta, não"}, effects: []effect{breakTool()}},
					{weight: 15, texts: []string{"Caiu uma pedra bem no cocuruto de {@}"}, effects: []effect{injure(15, 30)}},
				}},
				{label: "⛏️ Continuar cavando", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"{@} ignorou o teto, cavou rápido e saiu com o bolso cheio segundos antes do desmoronamento"}, effects: []effect{findLoot(), findLoot(), earn(20, 60)}},
					{weight: 35, texts: []string{"O teto desabou e {@} saiu engatinhando da poeira"}, effects: []effect{injure(25, 45)}},
					{weight: 12, texts: []string{"{@} ficou preso nos escombros por horas até o resgate chegar"}, effects: []effect{injure(45, 70), delay(30 * time.Minute)}},
					{weight: 8, texts: []string{"O teto veio inteiro. A mina virou o túmulo de {@}"}, effects: []effect{perish()}},
				}},
				{label: "⛑️ Confiar no capacete", risk: midRisk, needs: holds("capacete"), endings: []ending{
					{weight: 75, texts: []string{"A pedra quicou no capacete e {@} nem sentiu. Seguiu cavando e assobiando"}, effects: []effect{findLoot()}},
					{weight: 25, texts: []string{"O capacete segurou a pedra, mas o pescoço sentiu o tranco"}, effects: []effect{injure(5, 15)}},
				}},
			},
			idle: ending{texts: []string{"{@} ficou parado olhando pro teto até ele parar de estalar. Sorte de principiante"}},
		},
		{
			id:    "mina-veio-de-ouro",
			title: "O veio brilhando",
			intros: []string{
				"Atrás de uma parede rachada, {@} viu um brilho amarelo. É um veio de ouro, mas a parede treme só de olhar",
				"A lanterna de {@} bateu numa fresta e voltou dourada. Tem ouro ali atrás, e a rocha em volta tá toda solta",
			},
			choices: []choice{
				{label: "💥 Meter a picareta com força", risk: highRisk, endings: []ending{
					{weight: 40, texts: []string{"Na terceira picaretada a parede abriu e o ouro caiu no colo de {@}"}, effects: []effect{gain(1, "ouro"), gain(1, "prata")}},
					{weight: 30, texts: []string{"A parede desabou e trouxe metade do teto junto"}, effects: []effect{injure(20, 40)}},
					{weight: 22, texts: []string{"A parede não cedeu. Quem cedeu foi a picareta"}, effects: []effect{breakTool()}},
					{weight: 8, texts: []string{"Desmoronou tudo. {@} achou o ouro e o fim ao mesmo tempo"}, effects: []effect{perish()}},
				}},
				{label: "🪨 Cavar devagarinho", risk: midRisk, endings: []ending{
					{weight: 60, texts: []string{"Com paciência de garimpeiro, {@} tirou uma pepita sem derrubar nada"}, effects: []effect{gain(1, "ouro")}},
					{weight: 30, texts: []string{"Demorou tanto que o turno acabou antes do ouro aparecer"}, effects: []effect{delay(15 * time.Minute)}},
					{weight: 10, texts: []string{"Uma lasca de pedra voou no olho de {@}"}, effects: []effect{injure(5, 15)}},
				}},
				{label: "📍 Marcar o lugar e voltar depois", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} riscou um X na parede com a ponta da picareta e prometeu voltar com mais calma"}, effects: []effect{plant("mina-veio-revisitado", 3*time.Hour)}},
				}},
				{label: "🗺️ Consultar o mapa geológico", risk: lowRisk, needs: holds("mapa-geologico"), endings: []ending{
					{weight: 85, texts: []string{"O mapa mostrou o ponto exato e {@} tirou a pedra sem nem suar"}, effects: []effect{gain(1, "ouro", "esmeralda", "rubi")}},
					{weight: 15, texts: []string{"Pelo mapa era ouro. Era ouro de tolo: o mapa tava de ponta-cabeça"}, effects: []effect{gain(2, "ouro-de-tolo")}},
				}},
			},
		},
		{
			id:     "mina-veio-revisitado",
			title:  "De volta ao X",
			rarity: rareEvent,
			stage:  true,
			intros: []string{
				"{@} voltou na galeria do X riscado. O veio de ouro continua lá, brilhando, e a parede parece mais firme",
				"O X que {@} riscou na parede ainda tá lá. Tem pegada no chão de alguém que passou perto, mas o ouro ninguém tirou",
			},
			choices: []choice{
				{label: "🪨 Tirar o ouro com calma", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"Dessa vez foi tranquilo: {@} tirou o ouro todinho, pedra por pedra"}, effects: []effect{gain(2, "ouro")}},
					{weight: 30, texts: []string{"Alguém chegou antes e só deixou o rastro. Sobrou um pouco de prata no cantinho"}, effects: []effect{gain(1, "prata")}},
				}},
				{label: "🧨 Explodir a parede", risk: highRisk, uses: "dinamite", endings: []ending{
					{weight: 60, texts: []string{"BUM! A parede abriu e revelou uma câmara cheia de pedra preciosa"}, effects: []effect{gain(2, "ouro"), gain(1, "diamante")}},
					{weight: 30, texts: []string{"A explosão abriu a parede e o ouvido de {@}. Ainda tá zunindo"}, effects: []effect{gain(1, "ouro"), injure(15, 30)}},
					{weight: 10, texts: []string{"O pavio era mais curto do que parecia"}, effects: []effect{injure(40, 60)}},
				}},
				{label: "🤝 Chamar um sócio pra dividir", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"Os dois tiraram tudo e dividiram meio a meio. Menos ouro, zero dor de cabeça"}, effects: []effect{gain(1, "ouro"), earn(30, 60)}},
				}},
			},
		},
		{
			id:    "mina-cheiro-de-gas",
			title: "Cheiro estranho",
			intros: []string{
				"Um cheiro de ovo podre tomou conta da galeria. {@} sabe que mina com esse cheiro não é coisa boa",
				"{@} começou a ficar tonto e a lanterna piscou. Tem gás vazando em algum lugar da mina",
			},
			choices: []choice{
				{label: "🏃 Sair agora", risk: lowRisk, endings: []ending{
					{weight: 85, texts: []string{"{@} saiu rapidinho, respirou ar puro e esperou a mina ventilar"}, effects: []effect{delay(10 * time.Minute)}},
					{weight: 15, texts: []string{"No caminho da saída, {@} chutou um saco esquecido no chão e tinha coisa boa dentro"}, effects: []effect{findLoot()}},
				}},
				{label: "😤 Prender a respiração e cavar", risk: highRisk, endings: []ending{
					{weight: 42, texts: []string{"{@} cavou de fôlego preso igual mergulhador e saiu roxo, mas com o bolso cheio"}, effects: []effect{findLoot(), earn(20, 50)}},
					{weight: 38, texts: []string{"A tontura bateu e {@} acordou deitado no chão da galeria"}, effects: []effect{injure(15, 30)}},
					{weight: 12, texts: []string{"{@} desmaiou e só foi achado horas depois pela turma do turno da noite"}, effects: []effect{injure(35, 55)}},
					{weight: 8, texts: []string{"O gás venceu {@} antes da saída"}, effects: []effect{perish()}},
				}},
				{label: "🔥 Acender um fósforo pra ver", risk: highRisk, endings: []ending{
					{weight: 20, texts: []string{"Não era gás, era o ovo cozido da marmita de alguém. {@} achou a marmita e um brilho no chão"}, effects: []effect{findLoot(), gain(1, "marmita")}},
					{weight: 62, texts: []string{"Era gás, sim. A explosão jogou {@} lá pra fora da mina"}, effects: []effect{injure(30, 50), breakTool()}},
					{weight: 10, texts: []string{"A explosão abriu um buraco cheio de cristal e chamuscou {@} inteiro"}, effects: []effect{gain(1, "ametista", "topazio"), injure(15, 25)}},
					{weight: 8, texts: []string{"Era muito gás. Só sobrou o capacete"}, effects: []effect{perish()}},
				}},
				{label: "⛑️ Checar com a lanterna", risk: lowRisk, needs: holds("capacete"), endings: []ending{
					{weight: 100, texts: []string{"A lanterna mostrou o cano furado. {@} fechou o registro, virou herói da mina e ganhou uma caixinha da turma"}, effects: []effect{earn(30, 80), findLoot()}},
				}},
			},
			idle: ending{texts: []string{"{@} ficou parado pensando e começou a ver unicórnio. Melhor sair"}, effects: []effect{injure(5, 10), delay(10 * time.Minute)}},
		},
		{
			id:    "mina-morcegos",
			title: "Nuvem de morcegos",
			intros: []string{
				"{@} bateu a picareta numa fenda e saiu uma nuvem de morcegos gritando direto pra cara",
				"Mil morcegos acordaram de uma vez quando a lanterna de {@} iluminou o teto da galeria",
			},
			choices: []choice{
				{label: "🫣 Se jogar no chão", risk: lowRisk, endings: []ending{
					{weight: 80, texts: []string{"{@} deitou no chão e os morcegos passaram por cima. Ficou um presentinho de morcego no ombro"}},
					{weight: 20, texts: []string{"Um morcego ficou preso no cabelo de {@} e foi uma luta pra soltar"}, effects: []effect{injure(3, 8)}},
				}},
				{label: "🦇 Seguir os morcegos", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"Os morcegos levaram {@} até uma caverna escondida com as paredes brilhando"}, effects: []effect{gain(1, "geodo", "ametista", "quartzo")}},
					{weight: 30, texts: []string{"{@} se perdeu seguindo morcego e levou um tempão pra achar a saída"}, effects: []effect{delay(20 * time.Minute)}},
					{weight: 20, texts: []string{"{@} escorregou no guano e saiu deslizando galeria abaixo"}, effects: []effect{injure(5, 15)}},
				}},
				{label: "⛏️ Espantar na picaretada", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"{@} rodou a picareta igual hélice, espantou os bichos e ainda acertou um veio de prata"}, effects: []effect{gain(1, "prata")}},
					{weight: 30, texts: []string{"Um morcego mordeu a mão de {@}. Melhor passar no posto e tomar vacina"}, effects: []effect{injure(8, 18)}},
					{weight: 20, texts: []string{"A picareta escapou da mão e sumiu num buraco sem fundo"}, effects: []effect{breakTool()}},
				}},
			},
		},
		{
			id:    "mina-garimpeiro-armado",
			title: "Disputa pelo veio",
			when:  wentWell,
			intros: []string{
				"Um garimpeiro barbudo apareceu na galeria com um trinta e oito na cintura: 'esse veio é meu, parceiro'",
				"{@} achou um veio bom, mas um sujeito mal-encarado de revólver chegou junto dizendo que viu primeiro",
			},
			choices: []choice{
				{label: "🤝 Dividir o achado", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"Os dois cavaram juntos e dividiram. O barbudo ficou com a melhor parte, mas ninguém se machucou"}, effects: []effect{gain(1, "prata", "cobre")}},
					{weight: 30, texts: []string{"O barbudo dividiu do jeito dele: ficou com o ouro e deixou a terra pra {@}"}, effects: []effect{gain(2, "pedra")}},
				}},
				{label: "🚶 Deixar pra lá", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} levantou as mãos, foi cavar em outro canto e ainda achou alguma coisinha"}, effects: []effect{findLoot()}},
				}},
				{label: "👊 Encarar o barbudo", risk: highRisk, endings: []ending{
					{weight: 35, favor: func(p Profession) float64 { return p.guard }, texts: []string{"{@} partiu pra cima, tomou o revólver e o barbudo saiu correndo pela galeria"}, effects: []effect{gain(1, "ouro", "esmeralda"), earn(30, 80)}},
					{weight: 42, texts: []string{"O barbudo era faixa preta de garimpo. {@} apanhou feio"}, effects: []effect{injure(15, 35)}},
					{weight: 15, texts: []string{"O revólver disparou e pegou de raspão em {@}"}, effects: []effect{injure(35, 55)}},
					{weight: 8, texts: []string{"O barbudo não errou o tiro"}, effects: []effect{perish()}},
				}},
				{label: "🔫 Mostrar o seu ferro", risk: midRisk, needs: armedUp, endings: []ending{
					{weight: 65, texts: []string{"Quando viu a arma de {@}, o barbudo lembrou que tinha deixado o feijão no fogo"}, effects: []effect{gain(1, "ouro", "rubi"), stain()}},
					{weight: 25, texts: []string{"Virou bangue-bangue dentro da mina. {@} saiu ferido, mas com o veio"}, effects: []effect{injure(20, 40), gain(1, "ouro"), stain()}},
					{weight: 10, texts: []string{"O tiroteio chamou a polícia, que levou todo mundo"}, effects: []effect{nabbed()}},
				}},
				{label: "🚔 Dar voz de prisão", risk: midRisk, needs: lawman, endings: []ending{
					{weight: 80, texts: []string{"O barbudo tinha mandado de prisão em aberto. {@} levou ele algemado e ainda ficou com a recompensa"}, effects: []effect{earn(60, 150)}},
					{weight: 20, texts: []string{"O barbudo correu e atirou pra trás na fuga"}, effects: []effect{injure(10, 25)}},
				}},
			},
		},
		{
			id:    "mina-galeria-abandonada",
			title: "A galeria de 1932",
			intros: []string{
				"{@} achou uma passagem tapada com tábua podre e uma placa: PERIGO, GALERIA FECHADA EM 1932",
				"Atrás de um monte de entulho, {@} viu a entrada de uma galeria antiga. Tem uma lamparina velha pendurada e um frio esquisito saindo lá de dentro",
			},
			choices: []choice{
				{label: "🔦 Entrar na galeria", risk: highRisk, endings: []ending{
					{weight: 30, texts: []string{"Lá no fundo, {@} achou um esqueleto sentado, agarrado numa sacola de couro"}, next: "mina-sacola-do-esqueleto"},
					{weight: 25, texts: []string{"Tinha umas coisas antigas largadas pelo chão. {@} encheu o bolso"}, effects: []effect{gain(1, "moeda-do-imperio", "vaso-antigo", "urna")}},
					{weight: 25, texts: []string{"{@} se perdeu no escuro e caiu num poço raso"}, effects: []effect{injure(15, 30)}},
					{weight: 12, texts: []string{"Um espírito de mineiro antigo não gostou da visita e seguiu {@} até a saída"}, effects: []effect{enchant("encosto")}},
					{weight: 8, texts: []string{"A galeria desabou de vez, com {@} lá dentro"}, effects: []effect{perish()}},
				}},
				{label: "🪨 Jogar uma pedra e escutar", risk: lowRisk, endings: []ending{
					{weight: 60, texts: []string{"A pedra quicou por uns dez segundos. {@} achou melhor não entrar"}},
					{weight: 40, texts: []string{"Alguém lá dentro jogou a pedra de volta. {@} saiu correndo e só parou em casa"}, effects: []effect{delay(10 * time.Minute)}},
				}},
				{label: "📞 Avisar a turma da mina", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"A turma agradeceu o aviso e fez uma vaquinha pra {@}"}, effects: []effect{earn(20, 60)}},
				}},
				{label: "🕯️ Benzer a entrada antes", risk: lowRisk, needs: inArea(areaFe), endings: []ending{
					{weight: 100, texts: []string{"Depois da benzedura a galeria até clareou. {@} entrou, saiu com o bolso cheio e o corpo fechado"}, effects: []effect{gain(1, "moeda-do-imperio", "vaso-antigo"), enchant("protecao")}},
				}},
			},
		},
		{
			id:    "mina-sacola-do-esqueleto",
			title: "A sacola do esqueleto",
			stage: true,
			intros: []string{
				"O esqueleto tá de capacete de 1932 e abraçado na sacola de couro. Parece que ele não quer soltar",
				"Dentro da sacola do esqueleto tem um tilintar de moeda. A mão ossuda ainda tá agarrada na alça",
			},
			choices: []choice{
				{label: "🤲 Puxar a sacola", risk: highRisk, endings: []ending{
					{weight: 50, texts: []string{"A mão soltou com um estalo e a sacola veio cheia de moeda do Império"}, effects: []effect{gain(2, "moeda-do-imperio"), earn(40, 100)}},
					{weight: 30, texts: []string{"A sacola veio, e a maldição do antigo dono veio junto"}, effects: []effect{gain(1, "moeda-do-imperio"), enchant("miseria")}},
					{weight: 20, texts: []string{"O esqueleto desmontou em cima de {@}, que saiu gritando e bateu a cabeça na viga"}, effects: []effect{injure(10, 25)}},
				}},
				{label: "🙏 Rezar pelo defunto e deixar", risk: lowRisk, endings: []ending{
					{weight: 70, texts: []string{"{@} fez uma oração e deixou tudo como estava. Saiu de lá com uma paz que não sabe explicar"}, effects: []effect{enchant("protecao")}},
					{weight: 30, texts: []string{"Na saída, {@} achou uma moeda antiga brilhando no chão. Parece que o defunto agradeceu"}, effects: []effect{gain(1, "moeda-do-imperio")}},
				}},
				{label: "📞 Chamar a polícia", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"A polícia veio, o caso virou notícia e {@} deu entrevista no jornal da cidade"}, effects: []effect{earn(30, 80)}},
				}},
			},
		},
		{
			id:     "mina-dinossauro",
			title:  "O osso gigante",
			rarity: rareEvent,
			intros: []string{
				"A picareta de {@} bateu em algo duro e branco. Cavando em volta, apareceu um osso do tamanho de uma geladeira",
				"{@} achou uma costela gigante saindo da parede da mina. Ou é dinossauro, ou é a maior picanha do mundo",
			},
			choices: []choice{
				{label: "🖌️ Desenterrar com cuidado", risk: midRisk, endings: []ending{
					{weight: 55, texts: []string{"Com pincel e paciência, {@} tirou o fóssil inteirinho"}, effects: []effect{gain(1, "fossil"), delay(20 * time.Minute)}},
					{weight: 30, texts: []string{"Quebrou ao meio na última hora. Sobrou um crânio bonito"}, effects: []effect{gain(1, "caveira")}},
					{weight: 15, texts: []string{"Era osso de boi que alguém enterrou depois do churrasco"}, effects: []effect{gain(1, "pedra")}},
				}},
				{label: "⛏️ Arrancar na marra", risk: highRisk, endings: []ending{
					{weight: 30, texts: []string{"Veio inteiro, por milagre"}, effects: []effect{gain(1, "fossil")}},
					{weight: 50, texts: []string{"O fóssil virou farelo e a picareta foi junto"}, effects: []effect{breakTool()}},
					{weight: 20, texts: []string{"Uma vértebra do tamanho de um pneu rolou em cima do pé de {@}"}, effects: []effect{injure(10, 25)}},
				}},
				{label: "🏛️ Ligar pro museu", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"O museu mandou uma equipe, botou o nome de {@} na plaquinha e ainda pagou uma recompensa"}, effects: []effect{earn(150, 300), enchant("sorte")}},
				}},
				{label: "🕶️ Vender por fora", risk: highRisk, endings: []ending{
					{weight: 65, texts: []string{"Um colecionador gringo pagou uma fortuna em dinheiro vivo, sem nota e sem pergunta"}, effects: []effect{earn(250, 450), stain()}},
					{weight: 35, texts: []string{"O 'colecionador' era da Polícia Federal. Vender fóssil é crime"}, effects: []effect{nabbed(), lose(5, 10)}},
				}},
			},
		},
		{
			id:    "mina-dinamite-velha",
			title: "A dinamite esquecida",
			intros: []string{
				"No canto da galeria, {@} achou uma banana de dinamite velha, suando, com o pavio meio roído por rato",
				"{@} chutou sem querer uma caixa de madeira escrito EXPLOSIVOS. Tem uma dinamite antiga lá dentro",
			},
			choices: []choice{
				{label: "💣 Usar pra abrir a parede", risk: highRisk, endings: []ending{
					{weight: 45, texts: []string{"BUM! A parede abriu e choveu pedra boa pra todo lado"}, effects: []effect{findLoot(), findLoot(), gain(1, "ametista", "prata", "topazio")}},
					{weight: 35, texts: []string{"A dinamite explodiu mais cedo do que devia"}, effects: []effect{injure(20, 40), breakTool()}},
					{weight: 12, texts: []string{"Explodiu na mão. {@} vai ficar um tempo sem bater palma"}, effects: []effect{injure(45, 70)}},
					{weight: 8, texts: []string{"Explodiu tudo, inclusive {@}"}, effects: []effect{perish()}},
				}},
				{label: "🧯 Chamar o bombeiro", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"O bombeiro levou a dinamite, deu uma bronca daquelas e uma coxinha de consolação"}, effects: []effect{gain(1, "coxinha")}},
				}},
				{label: "🌊 Jogar no rio lá fora", risk: midRisk, endings: []ending{
					{weight: 65, texts: []string{"Explodiu na água e choveu peixe. Pesca com bomba é crime, mas {@} catou o que deu"}, effects: []effect{gain(2, "lambari", "tilapia"), stain()}},
					{weight: 35, texts: []string{"No caminho o pavio acendeu sozinho. Deu tempo de jogar, mas o tombo foi feio"}, effects: []effect{injure(10, 20)}},
				}},
			},
		},
		{
			id:     "mina-mae-do-ouro",
			title:  "A Mãe do Ouro",
			rarity: rareEvent,
			intros: []string{
				"Uma bola de fogo atravessou a galeria e parou na frente de {@}. Dentro dela tem uma mulher de cabelo em chamas: é a Mãe do Ouro",
				"A mina esquentou de repente e uma luz dourada tomou conta da galeria. A Mãe do Ouro apareceu e olhou bem pra {@}",
			},
			choices: []choice{
				{label: "🙇 Pedir licença e deixar oferenda", risk: lowRisk, price: cost{percent: 3, floor: 100}, endings: []ending{
					{weight: 75, texts: []string{"A Mãe do Ouro aceitou a oferenda e apontou com o dedo de fogo onde cavar"}, effects: []effect{enchant("maoboa"), enchant("achado"), gain(1, "ouro")}},
					{weight: 25, texts: []string{"Ela aceitou a oferenda e sumiu sem dizer nada. Pelo menos não ficou brava"}},
				}},
				{label: "🤑 Pegar o ouro que ela guarda", risk: highRisk, endings: []ending{
					{weight: 30, texts: []string{"Rápido igual raio, {@} pegou um punhado do altar dela e saiu correndo"}, effects: []effect{gain(2, "ouro"), gain(1, "diamante")}},
					{weight: 50, texts: []string{"A Mãe do Ouro queimou a mão gananciosa de {@} e jogou uma praga"}, effects: []effect{injure(20, 40), enchant("panema")}},
					{weight: 20, texts: []string{"Ela não queimou, fez pior: amaldiçoou o bolso de {@}"}, effects: []effect{enchant("miseria"), lose(5, 12)}},
				}},
				{label: "🙈 Abaixar a cabeça e sair", risk: lowRisk, endings: []ending{
					{weight: 100, texts: []string{"{@} saiu de cabeça baixa. A Mãe do Ouro gostou do respeito e abriu o caminho"}, effects: []effect{enchant("abrecaminho")}},
				}},
				{label: "🙏 Rezar pra ela", risk: lowRisk, needs: inArea(areaFe), endings: []ending{
					{weight: 100, texts: []string{"A reza acalmou a Mãe do Ouro, que deixou uma pepita de presente e o corpo de {@} fechado"}, effects: []effect{enchant("protecao"), gain(1, "ouro")}},
				}},
			},
		},
		{
			id:    "mina-carrinho-desgovernado",
			title: "Carrinho sem freio",
			intros: []string{
				"Um carrinho de minério veio descendo o trilho sem freio, direto na direção de {@}",
				"{@} ouviu um barulho de ferro rangendo e, quando virou, tinha um carrinho de mina descendo a toda",
			},
			choices: []choice{
				{label: "↪️ Pular pro lado", risk: lowRisk, endings: []ending{
					{weight: 80, texts: []string{"{@} pulou no último segundo igual goleiro em pênalti"}},
					{weight: 20, texts: []string{"Pulou, mas caiu bem em cima de uma pedra pontuda"}, effects: []effect{injure(4, 12)}},
				}},
				{label: "🛑 Segurar o carrinho", risk: highRisk, endings: []ending{
					{weight: 42, texts: []string{"{@} segurou o carrinho no peito, e ele tava cheio de minério bom"}, effects: []effect{gain(2, "ferro", "cobre", "prata"), earn(20, 50)}},
					{weight: 40, texts: []string{"O carrinho arrastou {@} uns dez metros pelo trilho"}, effects: []effect{injure(20, 40)}},
					{weight: 12, texts: []string{"Bateu de frente com tudo"}, effects: []effect{injure(40, 60)}},
					{weight: 6, texts: []string{"O carrinho não freou nem por educação"}, effects: []effect{perish()}},
				}},
				{label: "🎢 Pular dentro do carrinho", risk: midRisk, endings: []ending{
					{weight: 50, texts: []string{"O carrinho levou {@} até uma galeria que ninguém conhecia, cheia de pedra boa"}, effects: []effect{findLoot(), findLoot()}},
					{weight: 35, texts: []string{"Capotou na curva e {@} saiu rolando"}, effects: []effect{injure(10, 25)}},
					{weight: 15, texts: []string{"O carrinho saiu do outro lado do morro e {@} teve que voltar a pé"}, effects: []effect{delay(20 * time.Minute)}},
				}},
			},
			idle: ending{texts: []string{"{@} travou no meio do trilho e o carrinho passou raspando. Levou só o boné"}},
		},
	},
}
