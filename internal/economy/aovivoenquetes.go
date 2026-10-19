package economy

import "time"

var voteScenarios = []voteScenario{
	{
		id:    "enquete-churrasco",
		title: "A vaquinha do churrasco",
		intros: []string{
			"Alguém esqueceu um envelope com a vaquinha do churrasco em cima da mesa. Dá pra fazer muita coisa com esse dinheiro",
			"Juntaram uma grana pro churrasco de sábado, mas o dono da casa sumiu. O dinheiro tá com o grupo agora",
		},
		choices: []choice{
			{label: "🥩 Comprar só picanha", risk: lowRisk, endings: []ending{
				{weight: 70, texts: []string{"Churrasco histórico. Picanha pra todo mundo levar pra casa"}, effects: []effect{gain(1, "picanha")}},
				{weight: 30, texts: []string{"O açougueiro passou a perna no grupo e mandou acém no lugar da picanha. Pelo menos encheu a barriga"}, effects: []effect{heal(10)}},
			}},
			{label: "🍻 Gastar tudo em cerveja", risk: midRisk, endings: []ending{
				{weight: 50, texts: []string{"Foi a melhor festa do ano. Ninguém lembra direito, mas foi"}, effects: []effect{enchant("sorte")}},
				{weight: 50, texts: []string{"Acabou em ressaca coletiva e todo mundo chegou atrasado no dia seguinte"}, effects: []effect{injure(3, 8), delay(30 * time.Minute)}},
			}},
			{label: "💰 Dividir a grana", risk: lowRisk, endings: []ending{
				{weight: 100, texts: []string{"Cada um pegou sua parte e foi embora sem churrasco. Bando de capitalista"}, effects: []effect{earn(40, 90)}},
			}},
		},
		idle: []string{"Ninguém decidiu nada e o dono da casa voltou pra pegar o envelope"},
	},
	{
		id:     "enquete-mapa",
		title:  "O mapa do tesouro",
		rarity: rareEvent,
		intros: []string{
			"Dentro de uma garrafa que chegou na praia tinha um mapa com um X vermelho bem no meio da mata",
			"O avô de alguém morreu e deixou um mapa desenhado à mão com a frase \"o tesouro da família tá aqui\"",
		},
		choices: []choice{
			{label: "🗺️ Seguir o mapa", risk: highRisk, endings: []ending{
				{weight: 50, texts: []string{"O X era de verdade! O grupo cavou e achou um baú cheio de moeda antiga"}, effects: []effect{earn(150, 300), gain(1, "bau-do-tesouro")}},
				{weight: 35, texts: []string{"O mapa levou o grupo direto pra um formigueiro gigante"}, effects: []effect{injure(10, 25)}},
				{weight: 10, texts: []string{"Era o mapa de uma plantação de alguém que não gostou nada da visita"}, effects: []effect{nabbed()}},
				{weight: 5, texts: []string{"O mapa terminava num penhasco, e o X era lá embaixo"}, effects: []effect{perish()}},
			}},
			{label: "🏪 Vender o mapa", risk: lowRisk, endings: []ending{
				{weight: 100, texts: []string{"Um colecionador pagou bem pelo mapa e saiu feliz da vida. Será que tinha tesouro?"}, effects: []effect{earn(50, 100)}},
			}},
			{label: "🔥 Queimar, é armadilha", risk: lowRisk, endings: []ending{
				{weight: 100, texts: []string{"O grupo queimou o mapa e sentiu um alívio estranho, como se tivesse escapado de alguma coisa"}, effects: []effect{enchant("protecao")}},
			}},
		},
		idle: []string{"O mapa ficou na gaveta e virou papel de embrulhar peixe"},
	},
	{
		id:    "enquete-rifa",
		title: "A rifa da tia",
		intros: []string{
			"A tia do grupo tá vendendo rifa de uma moto zero. Cada número é baratinho e ela não aceita não como resposta",
			"\"Compra minha rifa, gente, é pra pagar a formatura do sobrinho!\" A tia tá na porta com o bloquinho na mão",
		},
		choices: []choice{
			{label: "🎟️ Todo mundo compra", risk: midRisk, endings: []ending{
				{weight: 60, texts: []string{"O número sorteado foi de um primo que ninguém conhece. Dinheiro perdido"}, effects: []effect{lose(1, 3)}},
				{weight: 40, texts: []string{"O grupo ganhou a moto! Venderam no mesmo dia e dividiram o dinheiro"}, effects: []effect{earn(100, 220)}},
			}},
			{label: "🙅 Ninguém compra", risk: lowRisk, endings: []ending{
				{weight: 70, texts: []string{"A tia foi embora resmungando, e ninguém gastou nada"}},
				{weight: 30, texts: []string{"A tia ficou ofendida e rogou uma praga de olho gordo no grupo inteiro"}, effects: []effect{enchant("olhogordo")}},
			}},
		},
		idle: []string{"A tia cansou de esperar resposta e foi vender pra outro grupo"},
	},
	{
		id:    "enquete-festa-firma",
		title: "A festa da firma",
		intros: []string{
			"Convite da festa de fim de ano da firma: open bar, karaokê e o chefe já avisou que vai dançar",
			"A empresa alugou um salão com DJ e buffet liberado. A presença não é obrigatória, mas o chefe vai anotar quem faltou",
		},
		choices: []choice{
			{label: "🕺 Ir e beber tudo", risk: midRisk, endings: []ending{
				{weight: 50, texts: []string{"O grupo fez amizade com o diretor no karaokê e saiu com a promessa de aumento"}, effects: []effect{earn(60, 140)}},
				{weight: 50, texts: []string{"Teve vômito no banheiro, briga no estacionamento e uma ressaca de três dias"}, effects: []effect{injure(5, 12)}},
			}},
			{label: "🏠 Ficar em casa", risk: lowRisk, endings: []ending{
				{weight: 100, texts: []string{"O grupo ficou em casa de pijama, comeu pipoca e dormiu cedo. Paz"}, effects: []effect{heal(15)}},
			}},
			{label: "📸 Ir só pra filmar", risk: midRisk, endings: []ending{
				{weight: 60, texts: []string{"O vídeo do chefe dançando viralizou e o grupo ganhou dinheiro com a monetização"}, effects: []effect{earn(60, 140)}},
				{weight: 40, texts: []string{"O chefe viu o vídeo e mandou todo mundo fazer hora extra"}, effects: []effect{delay(time.Hour)}},
			}},
		},
		idle: []string{"Ninguém respondeu o convite e a festa aconteceu sem o grupo"},
	},
	{
		id:    "enquete-vizinho",
		title: "O vizinho do som alto",
		intros: []string{
			"Três da tarde de uma terça e o vizinho tá com o paredão ligado no último volume. Os vidros tão tremendo",
			"O vizinho comprou uma caixa de som nova e resolveu testar com forró eletrônico o dia inteiro",
		},
		choices: []choice{
			{label: "🚔 Chamar a polícia", risk: lowRisk, endings: []ending{
				{weight: 60, texts: []string{"A polícia veio, desligou o som e o grupo sentiu o gostinho da justiça"}, effects: []effect{enchant("justica")}},
				{weight: 40, texts: []string{"A polícia não veio, e o vizinho ainda aumentou o volume"}},
			}},
			{label: "🔊 Revidar com som mais alto", risk: midRisk, endings: []ending{
				{weight: 50, texts: []string{"Virou uma batalha de som que juntou a rua inteira. O grupo vendeu cerveja e saiu no lucro"}, effects: []effect{earn(50, 120)}},
				{weight: 50, texts: []string{"O síndico multou todo mundo pela bagunça"}, effects: []effect{lose(2, 5)}},
			}},
			{label: "🍻 Juntar na festa dele", risk: midRisk, endings: []ending{
				{weight: 60, texts: []string{"O vizinho era gente boa e liberou a cachaça de alambique"}, effects: []effect{gain(1, "cachaca"), heal(5)}},
				{weight: 40, texts: []string{"Começou uma briga por causa do volume e sobrou pra todo mundo"}, effects: []effect{injure(5, 15)}},
			}},
		},
		idle: []string{"O grupo não decidiu nada, e o paredão tocou até as dez da noite"},
	},
	{
		id:     "enquete-piramide",
		title:  "A pirâmide do primo",
		rarity: rareEvent,
		intros: []string{
			"Um primo apareceu de terno brilhante oferecendo um investimento que rende 30% ao mês. \"É só trazer mais três amigos\"",
			"Chegou no grupo um convite pra uma \"oportunidade única\" de criptomoeda, com palestra num hotel e café grátis",
		},
		choices: []choice{
			{label: "📈 Investir todo mundo", risk: highRisk, endings: []ending{
				{weight: 40, texts: []string{"O grupo entrou cedo e saiu antes da pirâmide desabar. Lucro bom e consciência pesada"}, effects: []effect{earn(200, 400)}},
				{weight: 60, texts: []string{"A pirâmide desabou na semana seguinte e o primo fugiu pro Paraguai com o dinheiro de todo mundo"}, effects: []effect{lose(8, 15)}},
			}},
			{label: "🚔 Denunciar o primo", risk: lowRisk, endings: []ending{
				{weight: 100, texts: []string{"A polícia prendeu o primo e o grupo ganhou uma recompensa por ajudar"}, effects: []effect{earn(30, 60), enchant("justica")}},
			}},
			{label: "🙈 Fingir que não viu", risk: lowRisk, endings: []ending{
				{weight: 100, texts: []string{"O grupo ignorou o primo, que foi enganar outra família"}},
			}},
		},
		idle: []string{"O primo se cansou de esperar e foi oferecer pra outro grupo"},
	},
	{
		id:    "enquete-casa-assombrada",
		title: "A casa assombrada",
		intros: []string{
			"A casa abandonada no fim da rua acendeu a luz sozinha. Tem barulho de corrente e uma porta batendo",
			"Dizem que quem passa a noite na casa velha do morro acha o dinheiro que o antigo dono escondeu. Dizem também que ninguém voltou igual",
		},
		choices: []choice{
			{label: "👻 Entrar todo mundo junto", risk: highRisk, endings: []ending{
				{weight: 40, texts: []string{"Era só um gato e uma janela quebrada, e o grupo achou o dinheiro escondido no colchão"}, effects: []effect{earn(150, 300)}},
				{weight: 40, texts: []string{"Tinha um espírito de verdade, e ele gostou tanto do grupo que foi embora junto"}, effects: []effect{enchant("encosto")}},
				{weight: 16, texts: []string{"O assoalho podre cedeu e todo mundo caiu no porão"}, effects: []effect{injure(15, 30)}},
				{weight: 4, texts: []string{"A porta fechou sozinha e ninguém saiu de lá"}, effects: []effect{perish()}},
			}},
			{label: "🧂 Jogar sal grosso na porta", risk: lowRisk, endings: []ending{
				{weight: 100, texts: []string{"O grupo fez um descarrego na porta e a luz apagou na hora. Corpo fechado"}, effects: []effect{enchant("protecao")}},
			}},
			{label: "🏃 Correr pra longe", risk: lowRisk, endings: []ending{
				{weight: 100, texts: []string{"O grupo correu tanto que chegou na outra cidade. Ninguém se machucou"}},
			}},
		},
		idle: []string{"Ninguém teve coragem de decidir, e a luz da casa apagou sozinha"},
	},
	{
		id:    "enquete-enchente",
		title: "Choveu e alagou tudo",
		intros: []string{
			"Choveu em uma hora o que era pra chover no mês inteiro. A rua virou rio e tem sofá boiando",
			"O bueiro entupiu e a água tá na altura do joelho. Tem vizinho no telhado pedindo ajuda",
		},
		choices: []choice{
			{label: "🛶 Resgatar os vizinhos", risk: midRisk, endings: []ending{
				{weight: 60, texts: []string{"O grupo salvou a rua inteira de boia de piscina e virou herói do bairro"}, effects: []effect{earn(40, 90), enchant("sorte")}},
				{weight: 40, texts: []string{"Na correnteza, todo mundo bateu em alguma coisa que não dava pra ver"}, effects: []effect{injure(5, 15)}},
			}},
			{label: "📦 Salvar as próprias coisas", risk: lowRisk, endings: []ending{
				{weight: 100, texts: []string{"O grupo subiu tudo pra laje e passou a noite contando goteira"}},
			}},
			{label: "🎣 Pescar na rua", risk: midRisk, endings: []ending{
				{weight: 50, texts: []string{"Os peixes do lago transbordaram e o grupo encheu o balde"}, effects: []effect{gain(2, "lambari", "traira")}},
				{weight: 50, texts: []string{"Pescar em água de enchente não foi uma boa ideia. Todo mundo pegou uma virose"}, effects: []effect{injure(8, 18)}},
			}},
		},
		idle: []string{"O grupo ficou olhando a chuva pela janela até a água baixar"},
	},
	{
		id:    "enquete-show",
		title: "Show de graça na praça",
		intros: []string{
			"A prefeitura trouxe um cantor famoso pra um show de graça na praça. Já tem gente acampada desde ontem",
			"Show surpresa no centro! O palco tá montado e a multidão tá chegando",
		},
		choices: []choice{
			{label: "🎤 Ir pro gargarejo", risk: midRisk, endings: []ending{
				{weight: 50, texts: []string{"O cantor jogou a toalha suada pro grupo, e veio junto uma energia boa"}, effects: []effect{enchant("sorte")}},
				{weight: 30, texts: []string{"No empurra-empurra todo mundo ficou amassado contra a grade"}, effects: []effect{injure(5, 15)}},
				{weight: 20, texts: []string{"Furtaram os celulares do grupo inteiro no meio da música"}, effects: []effect{lose(2, 5)}},
			}},
			{label: "🍢 Vender espetinho", risk: lowRisk, endings: []ending{
				{weight: 100, texts: []string{"O grupo montou uma churrasqueira na esquina e vendeu tudo antes do bis"}, effects: []effect{earn(60, 140)}},
			}},
			{label: "🏠 Ver a live de casa", risk: lowRisk, endings: []ending{
				{weight: 100, texts: []string{"O grupo viu tudo pela live, de pijama, sem fila de banheiro"}},
			}},
		},
		idle: []string{"O show acabou antes do grupo decidir se ia"},
	},
	{
		id:     "enquete-heranca",
		title:  "A herança do tio-avô",
		rarity: legendaryEvent,
		intros: []string{
			"Um advogado ligou: um tio-avô que ninguém conhecia morreu e deixou tudo pro grupo. Tem dinheiro, tem casa velha e tem parente brigando",
			"Chegou uma carta registrada: \"vocês são os herdeiros do senhor Oswaldo\". Ninguém sabe quem é o Oswaldo, mas a herança é de verdade",
		},
		choices: []choice{
			{label: "⚖️ Brigar na justiça por tudo", risk: midRisk, endings: []ending{
				{weight: 50, texts: []string{"O juiz deu ganho de causa pro grupo, e a herança inteira caiu na conta"}, effects: []effect{earn(300, 600)}},
				{weight: 50, texts: []string{"O processo durou anos e o advogado ficou com quase tudo"}, effects: []effect{lose(3, 8)}},
			}},
			{label: "🤝 Dividir com os parentes", risk: lowRisk, endings: []ending{
				{weight: 100, texts: []string{"O grupo dividiu tudo na paz, e ainda sobrou uma parte boa pra cada um"}, effects: []effect{earn(150, 300)}},
			}},
			{label: "🏚️ Ficar com a casa velha", risk: highRisk, endings: []ending{
				{weight: 40, texts: []string{"Na parede da casa tinha um cofre escondido, cheio do que o tio-avô guardou a vida inteira"}, effects: []effect{earn(400, 800)}},
				{weight: 40, texts: []string{"A casa vinha com o espírito do tio-avô, que não gostou nada dos herdeiros"}, effects: []effect{enchant("encosto")}},
				{weight: 20, texts: []string{"A casa tava caindo aos pedaços e o grupo gastou tudo na reforma"}, effects: []effect{lose(5, 10)}},
			}},
		},
		idle: []string{"Ninguém respondeu o advogado, e a herança foi pro governo"},
	},
	{
		id:    "enquete-greve",
		title: "Greve de ônibus",
		intros: []string{
			"Os motoristas entraram em greve sem avisar e o ponto tá lotado. O trampo não vai esperar",
			"Nenhum ônibus na rua hoje. O aplicativo de transporte tá cobrando o preço de um rim",
		},
		choices: []choice{
			{label: "🚲 Ir de bicicleta", risk: midRisk, endings: []ending{
				{weight: 60, texts: []string{"O grupo pedalou até o trampo e chegou suado, mas com a saúde em dia"}, effects: []effect{heal(10)}},
				{weight: 40, texts: []string{"Um buraco no meio do caminho derrubou a bicicletada inteira"}, effects: []effect{injure(5, 12)}},
			}},
			{label: "🚗 Rachar um carro", risk: lowRisk, endings: []ending{
				{weight: 100, texts: []string{"O grupo rachou a corrida e chegou no horário, mas a tarifa dinâmica doeu"}, effects: []effect{lose(1, 3)}},
			}},
			{label: "🛌 Faltar o trampo", risk: lowRisk, endings: []ending{
				{weight: 60, texts: []string{"Dia de folga não planejada. O grupo dormiu até tarde e acordou renovado"}, effects: []effect{heal(15), delay(time.Hour)}},
				{weight: 40, texts: []string{"O chefe ligou e descontou o dia de todo mundo"}, effects: []effect{lose(1, 3), delay(time.Hour)}},
			}},
		},
		idle: []string{"Enquanto o grupo pensava, a greve acabou"},
	},
	{
		id:    "enquete-gato-preto",
		title: "O gato preto na porta",
		intros: []string{
			"Um gato preto sentou na porta e não sai de jeito nenhum. Fica olhando pra dentro como quem sabe de alguma coisa",
			"Sexta-feira 13 e um gato preto tá miando sem parar no portão do grupo",
		},
		choices: []choice{
			{label: "🐈‍⬛ Adotar o gato", risk: midRisk, endings: []ending{
				{weight: 50, texts: []string{"O gato virou o mascote do grupo e cada um ganhou um filhote da ninhada que veio depois"}, effects: []effect{gain(1, "gato-preto")}},
				{weight: 50, texts: []string{"A vizinha supersticiosa jura que o azar começou no dia que o gato entrou"}, effects: []effect{enchant("azar")}},
			}},
			{label: "🧹 Espantar o gato", risk: lowRisk, endings: []ending{
				{weight: 100, texts: []string{"O gato foi embora ofendido e ninguém mais viu ele"}},
			}},
			{label: "🍀 Fazer uma simpatia", risk: lowRisk, endings: []ending{
				{weight: 100, texts: []string{"A simpatia virou o jogo e o gato trouxe sorte em vez de azar"}, effects: []effect{enchant("sorte")}},
			}},
		},
		idle: []string{"O gato cansou de esperar e foi atrás de outra família"},
	},
	{
		id:    "enquete-blitz",
		title: "A blitz na esquina",
		intros: []string{
			"O grupo inteiro tá apertado no mesmo carro, com documento atrasado, e tem uma blitz logo ali na frente",
			"Voltando da praia todo mundo amontoado no carro, e tem viatura parando quem passa",
		},
		choices: []choice{
			{label: "🚗 Passar reto", risk: highRisk, endings: []ending{
				{weight: 50, texts: []string{"O guarda tava distraído no celular e o carro passou batido"}},
				{weight: 40, texts: []string{"A viatura foi atrás e levou o grupo inteiro pra delegacia"}, effects: []effect{nabbed()}},
				{weight: 10, texts: []string{"Deu tempo de fugir, mas o radar pegou e a multa veio pra todo mundo"}, effects: []effect{lose(3, 6)}},
			}},
			{label: "🔄 Dar a volta no quarteirão", risk: lowRisk, endings: []ending{
				{weight: 100, texts: []string{"O desvio levou meia hora no trânsito, mas ninguém foi parado"}, effects: []effect{delay(30 * time.Minute)}},
			}},
			{label: "📄 Parar e mostrar documento", risk: lowRisk, endings: []ending{
				{weight: 70, texts: []string{"O guarda conferiu tudo e liberou com uma bronca"}},
				{weight: 30, texts: []string{"O guarda era fã do grupo e desejou boa viagem"}, effects: []effect{enchant("protecao")}},
			}},
		},
		idle: []string{"O carro ficou parado no acostamento e a blitz foi embora antes"},
	},
	{
		id:    "enquete-final",
		title: "A final do campeonato",
		intros: []string{
			"Hoje é a final do campeonato e o time do grupo é o azarão. O bar da esquina tá oferecendo aposta na hora",
			"Clássico decisivo na TV e o grupo precisa decidir como vai assistir",
		},
		choices: []choice{
			{label: "🍻 Assistir no bar", risk: midRisk, endings: []ending{
				{weight: 60, texts: []string{"O time ganhou e o dono do bar liberou a rodada pra todo mundo"}, effects: []effect{heal(10), enchant("sorte")}},
				{weight: 40, texts: []string{"O time perdeu e a torcida rival quebrou o bar"}, effects: []effect{injure(5, 15)}},
			}},
			{label: "🎰 Apostar no azarão", risk: highRisk, endings: []ending{
				{weight: 30, texts: []string{"O azarão virou no último minuto e a aposta pagou uma fortuna"}, effects: []effect{earn(200, 400)}},
				{weight: 70, texts: []string{"O azarão levou de quatro e o dinheiro foi junto"}, effects: []effect{lose(4, 9)}},
			}},
			{label: "📺 Ver em casa", risk: lowRisk, endings: []ending{
				{weight: 100, texts: []string{"O grupo viu o jogo do sofá, comendo pipoca, sem confusão"}},
			}},
		},
		idle: []string{"O jogo acabou e o grupo ainda tava discutindo onde ia assistir"},
	},
	{
		id:     "enquete-luzes",
		title:  "Luzes no céu",
		rarity: rareEvent,
		intros: []string{
			"Três luzes coloridas pararam em cima do morro e não saem do lugar. Os cachorros da rua tão todos uivando",
			"Uma luz forte iluminou o campinho no meio da tarde, e o celular de todo mundo ficou sem sinal",
		},
		choices: []choice{
			{label: "🛸 Ir atrás das luzes", risk: highRisk, endings: []ending{
				{weight: 40, texts: []string{"O grupo filmou a nave de pertinho e vendeu o vídeo pro Fantástico"}, effects: []effect{earn(150, 300)}},
				{weight: 40, texts: []string{"Todo mundo foi abduzido e devolvido horas depois, sem lembrar de nada"}, effects: []effect{delay(time.Hour), enchant("lerdeza")}},
				{weight: 20, texts: []string{"No escuro do morro, o grupo rolou ribanceira abaixo"}, effects: []effect{injure(10, 25)}},
			}},
			{label: "📱 Filmar de longe", risk: lowRisk, endings: []ending{
				{weight: 60, texts: []string{"O vídeo saiu tremido, mas um canal de mistério pagou pelos direitos"}, effects: []effect{earn(40, 100)}},
				{weight: 40, texts: []string{"No vídeo só aparece um borrão e a voz de alguém gritando"}},
			}},
			{label: "🙏 Rezar e ficar em casa", risk: lowRisk, endings: []ending{
				{weight: 100, texts: []string{"O grupo rezou o terço inteiro, e as luzes foram embora"}, effects: []effect{enchant("protecao")}},
			}},
		},
		idle: []string{"As luzes sumiram antes de alguém decidir"},
	},
	{
		id:    "enquete-bolao",
		title: "O bolão da Mega",
		intros: []string{
			"A Mega acumulou e a moça da lotérica tá montando um bolão. Ainda dá tempo de entrar",
			"Prêmio acumulado e todo mundo do trampo tá entrando no bolão. Falta só o grupo decidir",
		},
		choices: []choice{
			{label: "🎟️ Entrar no bolão", risk: midRisk, endings: []ending{
				{weight: 85, texts: []string{"Não acertaram nem a quadra. Fica pra próxima"}, effects: []effect{lose(1, 2)}},
				{weight: 15, texts: []string{"Acertaram a quina! Não dá pra largar o emprego, mas dá pra pagar umas contas"}, effects: []effect{earn(300, 600)}},
			}},
			{label: "🙅 Ficar de fora", risk: lowRisk, endings: []ending{
				{weight: 85, texts: []string{"O bolão não deu nada e o grupo economizou"}},
				{weight: 15, texts: []string{"O bolão ganhou a quina, e o grupo ficou de fora remoendo a inveja"}, effects: []effect{enchant("olhogordo")}},
			}},
		},
		idle: []string{"O bolão fechou antes de o grupo decidir"},
	},
}
