package economy

import (
	"time"
)

var professions = []Profession{
	{
		name: "bico", emoji: "🎲", title: "faz bico", area: areaBicos, min: 30, max: 120, every: 8 * time.Minute,
		tasks: []string{
			"trabalhou vendendo bala no farol", "trabalhou de flanelinha no estacionamento do mercado",
			"trabalhou de estagiário não remunerado, mas o chefe pagou um pastel", "trabalhou vendendo pack do pé",
			"trabalhou de DJ em festa de família", "trabalhou de testador de colchão", "trabalhou consertando o wifi da vizinha",
			"trabalhou de figurante de novela", "trabalhou de garçom em casamento", "trabalhou de palhaço em festa infantil",
			"trabalhou de detetive de traição", "trabalhou de cuidador de cachorro de madame", "trabalhou cantando no ônibus",
			"trabalhou de lavador de carro", "trabalhou de influencer de cripto",
		},
	},
	{
		name: "coach", emoji: "🧘", title: "é coach", area: areaBicos, min: 200, max: 600, every: 28 * time.Minute, bust: 0.4,
		tasks: []string{
			"vendeu uma mentoria de mindset milionário", "gritou ACREDITA em cima de um palco", "fez um workshop de como acordar às 4h",
			"vendeu um curso de como vender curso", "ensinou respiração quântica pra 40 pessoas",
		},
		busts: []string{
			"deu uma palestra pra 3 pessoas, todas da família", "postou motivação no LinkedIn e ninguém comprou nada",
			"fez imersão de fim de semana e só apareceu o cara do som",
		},
	},
	{
		name: "testador de colchão", emoji: "🛏️", title: "é testador de colchão", area: areaBicos, aliases: []string{"testador"},
		min: 80, max: 160, every: 20 * time.Minute,
		dailyBonus:    50,
		breakoutBonus: -0.1,
		tasks: []string{
			"dormiu 4 horas seguidas e chamou isso de expediente", "testou um colchão ortopédico e deu nota 7 pela babinha",
			"deitou de conchinha com o travesseiro da loja", "roncou tão alto que o gerente achou que era a máquina de lavar",
			"avaliou a mola do colchão pulando igual criança", "cochilou no colchão de casal e acordou com cliente olhando",
			"escreveu um laudo de 3 páginas sobre o travesseiro de pena",
		},
	},
	{
		name: "figurante", emoji: "🎬", title: "é figurante", area: areaBicos, aliases: []string{"extra"},
		min: 90, max: 170, every: 20 * time.Minute,
		doubleChance: 0.1,
		finds:        []string{"marmita", "coxinha", "cafe", "pao-de-queijo"}, findChance: 0.15,
		tasks: []string{
			"passou de bicicleta no fundo da cena da novela das nove", "fez o cliente do bar que não fala nada",
			"tomou um café de mentira por 6 horas", "apareceu de costas no Jornal Nacional",
			"fez cara de espanto no fundo do barraco da protagonista", "foi o defunto no velório da novela e quase dormiu de verdade",
			"acenou pra câmera e teve que gravar tudo de novo",
		},
	},
	{
		name: "papai noel de shopping", emoji: "🎅", title: "é papai noel de shopping", area: areaBicos,
		aliases: []string{"papai noel", "noel"},
		min:     180, max: 300, every: 28 * time.Minute,
		bust: 0.15, busts: []string{
			"levou um xixi no colo de uma criança de 3 anos", "teve a barba puxada e descobriram que era falsa",
			"prometeu um PlayStation 5 e a mãe da criança foi reclamar na gerência",
		},
		finds: []string{"ursinho", "boneca", "carrinho", "chocolate", "tamagotchi"}, findChance: 0.2,
		tasks: []string{
			"ouviu 80 pedidos de celular e um de pai voltar pra casa", "tirou foto com 200 crianças e um tiozão bêbado",
			"suou dentro da roupa de veludo a 38 graus", "gritou HO HO HO até ficar rouco",
			"ganhou desenho de uma criança e chorou no intervalo", "disse que o trenó estava na oficina",
		},
	},
	{
		name: "cartomante", emoji: "🔮", title: "é cartomante", area: areaBicos, aliases: []string{"vidente", "taróloga"},
		min: 100, max: 200, every: 20 * time.Minute,
		doubleChance: 0.15, reportBonus: -0.2,
		tasks: []string{
			"previu que a cliente ia reencontrar o ex e acertou no mesmo dia", "tirou a carta da Morte e disse que era coisa boa",
			"jogou búzios pelo WhatsApp", "fez simpatia pra amarrar o boy em 7 dias",
			"leu a mão de um cara e só viu calo de academia", "previu chuva e choveu, virou lenda no bairro",
			"cobrou consulta extra porque Mercúrio estava retrógrado",
		},
	},
	{
		name: "necromante", emoji: "🧟", title: "é necromante", area: areaBicos, aliases: []string{"necromancer", "ressuscitador"},
		min: 90, max: 180, every: 20 * time.Minute,
		bust: 0.1, busts: []string{
			"errou o feitiço e levantou o gato do vizinho em vez do cliente", "acendeu as velas erradas e o cemitério chamou a polícia",
		},
		reviveChance: 0.6,
		tasks: []string{
			"conversou com o falecido e descobriu onde tava o testamento", "fez um ritual no cemitério à meia-noite e cobrou adicional noturno",
			"trouxe o vô de volta só pra ele dizer a senha do Wi-Fi", "levantou um defunto que pediu pra voltar pro caixão",
			"vendeu vela preta com desconto de Finados", "fez sessão espírita e o espírito deixou avaliação de 5 estrelas",
		},
	},
	{
		name: "homem-placa", emoji: "🪧", title: "é homem-placa", area: areaBicos, aliases: []string{"homem placa", "placa"},
		min: 50, max: 110, every: 16 * time.Minute,
		discount: 15,
		tasks: []string{
			"segurou a placa de COMPRO OURO debaixo do sol", "anunciou apartamento decorado girando a placa igual malabarista",
			"ficou 6 horas na esquina com a seta do feirão de carros", "dormiu em pé encostado na placa",
			"usou a placa de guarda-sol", "decorou todas as promoções do bairro",
		},
	},
	{
		name: "fiscal de obra", emoji: "👀", title: "é fiscal de obra", area: areaBicos, aliases: []string{"fiscal"},
		min: 60, max: 120, every: 16 * time.Minute,
		guard: 0.15,
		tasks: []string{
			"ficou 3 horas de mãos pra trás olhando o buraco da Sabesp", "falou que no tempo dele a laje era mais grossa",
			"apontou o erro do pedreiro sem nunca ter pegado numa colher", "acompanhou a betoneira girando do começo ao fim",
			"deu palpite na massa e foi ignorado com sucesso", "contou quantos tijolos a obra usou, errou e contou de novo",
		},
	},
	{
		name: "influencer de cripto", emoji: "🚀", title: "é influencer de cripto", area: areaBicos, aliases: []string{"cripto"},
		min: 300, max: 900, every: 36 * time.Minute, reach: 150, stage: "tiktok",
		bust: 0.45, bustFine: 300, busts: []string{
			"divulgou uma moeda de cachorro que virou pó em 10 minutos", "foi pego no golpe da pirâmide que ele mesmo divulgou",
			"esqueceu a senha da carteira com tudo dentro", "fez live gritando TO THE MOON bem na hora do crash",
		},
		doubleChance: 0.1,
		tasks: []string{
			"vendeu um curso de day trade com print de lucro do Paint", "postou foto alugando Lamborghini por uma hora",
			"divulgou uma moeda nova e saiu antes de todo mundo", "fez live de madrugada explicando blockchain errado",
			"vendeu NFT de macaco pra um tio do zap", "falou HODL tantas vezes que perdeu a voz",
		},
	},
	{
		name: "guardador de fila", emoji: "🧍", title: "é guardador de fila", area: areaBicos, aliases: []string{"fila"},
		min: 60, max: 110, every: 16 * time.Minute,
		theftEvery: 10 * time.Minute, theftBonus: 0.1,
		tasks: []string{
			"guardou lugar na fila do INSS desde as 4h da manhã", "segurou a fila do show por 2 dias com um banquinho",
			"ficou na fila do lançamento do iPhone pra um playboy", "guardou lugar na fila do posto de saúde e fez amizade com todo mundo",
			"enfrentou a fila do Detran sem ir no banheiro", "dormiu na fila da matrícula enrolado num cobertor",
		},
	},
	{
		name: "degustador", emoji: "😋", title: "é degustador", area: areaBicos, aliases: []string{"provador"},
		min: 100, max: 180, every: 20 * time.Minute,
		finds: []string{"coxinha", "pastel", "pao-de-queijo", "pudim", "pizza", "chocolate"}, findChance: 0.25,
		peak: peak{from: 11, to: 14, bonus: 40},
		tasks: []string{
			"provou 30 sabores de sorvete e pediu mais um", "deu nota pra coxinha da padaria com cara de crítico",
			"comeu amostra grátis no mercado o dia inteiro", "avaliou o pudim pela quantidade de furinho",
			"testou a pimenta nova e chorou profissionalmente", "experimentou o pastel de vento e reclamou do vento",
			"comeu todo o buffet de degustação do casamento antes dos noivos",
		},
	},
	{
		name: "apostador profissional", emoji: "🎰", title: "é apostador profissional", area: areaBicos, aliases: []string{"apostador"},
		min: 150, max: 350, every: 28 * time.Minute,
		bust: 0.3, busts: []string{
			"apostou no time da casa e tomou de 7", "foi no jogo do tigrinho só pra ver e saiu sem a camisa",
			"perdeu a aposta no último minuto dos acréscimos", "jurou que tinha um método infalível e o método falhou",
		},
		cashback: 25,
		tasks: []string{
			"acertou o placar exato do jogo da Série B", "ganhou no bicho sonhando com a sogra",
			"fez uma múltipla de 12 jogos e bateu tudo", "apostou no escanteio aos 47 do segundo tempo e deu certo",
			"leu as estatísticas do time da Moldávia por 4 horas", "virou a banca na roleta e saiu antes de devolver",
		},
	},
	{
		name: "crupiê", emoji: "🃏", title: "é crupiê", area: areaBicos, aliases: []string{"croupier", "dealer"},
		min: 80, max: 160, every: 20 * time.Minute,
		peak:     peak{from: 22, to: 4, bonus: 50},
		cashback: 15,
		tasks: []string{
			"embaralhou as cartas igual mágico de festa", "girou a roleta com cara de quem sabe onde vai cair",
			"recolheu as fichas de um tiozão que apostou a aposentadoria", "segurou a cara de paisagem num blefe de 5 mil",
			"pagou um blackjack com cara de quem não gostou", "expulsou um cara contando carta no dedo",
		},
	},
	{
		name: "pedreiro", emoji: "🧱", title: "é pedreiro", area: areaObras, min: 120, max: 220, every: 20 * time.Minute,
		hazard: hazard{chance: 0.1, harm: 30, wounds: []string{"deixou um tijolo cair no pé", "caiu do andaime do segundo andar", "levou uma marretada no dedão"}},
		tasks: []string{
			"levantou um muro torto", "bateu laje no domingo", "assentou piso sem nível nenhum", "fez um puxadinho na casa da sogra",
			"rebocou a parede e o próprio pé", "terminou a obra só 3 meses atrasado",
		},
	},
	{
		name: "servente", emoji: "🪣", title: "é servente de pedreiro", area: areaObras, aliases: []string{"ajudante"},
		min: 80, max: 140, every: 16 * time.Minute,
		finds: []string{"pedra", "cafe", "marmita", "pao-de-queijo", "chinelo"}, findChance: 0.15,
		hazard: hazard{chance: 0.1, harm: 25, wounds: []string{"travou a coluna carregando saco de cimento", "pisou num prego enferrujado da obra", "caiu dentro da vala da fundação"}},
		tasks: []string{
			"carregou 200 tijolos subindo a escada", "virou massa na enxada até a mão criar calo",
			"buscou o café do mestre 14 vezes", "subiu saco de cimento pro terceiro andar no ombro",
			"peneirou areia o dia inteiro", "escutou o pedreiro reclamar da vida por 8 horas",
		},
	},
	{
		name: "mestre de obras", emoji: "👷", title: "é mestre de obras", area: areaObras, aliases: []string{"mestre"},
		min: 180, max: 300, every: 36 * time.Minute,
		discount: 15, abuses: []abuseKind{workAbuse},
		hazard: hazard{chance: 0.05, harm: 25, wounds: []string{"foi conferir a laje e levou uma telha na cabeça", "tropeçou num vergalhão no meio da obra"}},
		tasks: []string{
			"gritou com a equipe inteira antes das 7h", "conseguiu cimento mais barato com o primo do depósito",
			"jurou pro cliente que a obra entrega semana que vem", "leu a planta de cabeça pra baixo e ninguém percebeu",
			"fiscalizou o reboco com a régua e um olhar de decepção", "fez o orçamento e esqueceu de cobrar o próprio serviço",
		},
	},
	{
		name: "eletricista", emoji: "⚡", title: "é eletricista", area: areaObras, min: 110, max: 200, every: 20 * time.Minute,
		bust: 0.15, bustFine: 60, busts: []string{
			"tomou um choque no disjuntor e ficou com o cabelo em pé", "ligou o fio errado e queimou a geladeira do cliente",
			"derrubou a energia da rua inteira no meio da novela",
		},
		peak:       peak{from: 18, to: 22, bonus: 40},
		heistBonus: 0.05,
		hazard:     hazard{chance: 0.12, harm: 40, wounds: []string{"levou um choque de 220 que deixou o cabelo em pé", "esqueceu de desligar o disjuntor", "caiu do poste mexendo num gato de luz"}},
		tasks: []string{
			"trocou o chuveiro da vizinha e ainda queimou a resistência", "fez um gato na rede e chamou de instalação provisória",
			"passou fiação nova na casa inteira", "consertou a tomada que soltava faísca desde 2015",
			"instalou um lustre que ninguém sabe se vai cair", "trocou o padrão de luz em cima da escada bamba",
		},
	},
	{
		name: "encanador", emoji: "🪠", title: "é encanador", area: areaObras, min: 110, max: 210, every: 20 * time.Minute,
		finds: []string{"bota-velha", "rato", "cueca", "carteira-perdida", "sapo", "anel"}, findChance: 0.12,
		hazard: hazard{chance: 0.07, harm: 20, wounds: []string{"levou um jato de esgoto na cara", "cortou a mão num cano quebrado"}},
		tasks: []string{
			"desentupiu a privada de uma festa de aniversário", "trocou o sifão da pia e molhou a cozinha inteira",
			"achou o vazamento depois de quebrar meio banheiro", "consertou a caixa d'água no telhado debaixo de sol",
			"desentupiu o ralo e tirou uma peruca inteira", "mostrou o cofrinho pra família toda debaixo da pia",
		},
	},
	{
		name: "pintor", emoji: "🎨", title: "é pintor", area: areaObras, min: 90, max: 170, every: 20 * time.Minute,
		doubleChance: 0.1,
		hazard:       hazard{chance: 0.08, harm: 25, wounds: []string{"caiu da escada com a lata de tinta junto", "ficou tonto com o cheiro do thinner e caiu do andaime"}},
		tasks: []string{
			"pintou a fachada de verde-limão a pedido da cliente", "deu duas demãos e cobrou por três",
			"pintou a parede e o rodapé junto", "fez textura grafiato que arranha quem encosta",
			"pintou o quarto da criança de rosa e o pai pediu azul", "pisou na lata de tinta e deixou pegada no corredor todo",
		},
	},
	{
		name: "marceneiro", emoji: "🪚", title: "é marceneiro", area: areaObras, min: 130, max: 230, every: 28 * time.Minute,
		salePercent: 75,
		hazard:      hazard{chance: 0.1, harm: 35, wounds: []string{"passou o dedo perto demais da serra circular", "martelou o próprio dedo"}},
		tasks: []string{
			"montou um guarda-roupa sem sobrar parafuso nenhum", "fez uma mesa de jantar que balança só um pouquinho",
			"lixou uma cadeira até ela virar banquinho", "fez armário planejado que não coube na parede",
			"restaurou a cristaleira da avó", "consertou a porta que rangia desde a Copa de 2014",
		},
	},
	{
		name: "serralheiro", emoji: "🔩", title: "é serralheiro", area: areaObras, min: 120, max: 200, every: 28 * time.Minute,
		theftEvery: 15 * time.Minute, theftBonus: 0.15, escapeBonus: 0.15,
		heistBonus: 0.08,
		hazard:     hazard{chance: 0.08, harm: 30, wounds: []string{"deixou o portão de ferro cair no pé", "levou uma faísca da esmerilhadeira no olho"}},
		tasks: []string{
			"fez um portão de ferro que pesa mais que um carro", "abriu um cadeado com um clipe em 10 segundos",
			"instalou grade em todas as janelas do prédio", "soldou a grade do vizinho por engano",
			"fez uma escada caracol que deixa todo mundo tonto", "consertou o portão eletrônico que só abria pela metade",
		},
	},
	{
		name: "gesseiro", emoji: "🧊", title: "é gesseiro", area: areaObras, aliases: []string{"drywall"},
		min: 100, max: 180, every: 20 * time.Minute,
		peak: peak{from: 6, to: 10, bonus: 50},
		tasks: []string{
			"fez uma sanca com LED que a cliente viu no Pinterest", "rebaixou o teto da sala e quase não sobrou sala",
			"levantou uma parede de drywall em uma tarde", "deixou a casa inteira branca de pó de gesso",
			"fez moldura de gesso igual de palácio", "consertou a trinca do teto com fé e massa corrida",
		},
	},
	{
		name: "azulejista", emoji: "🧩", title: "é azulejista", area: areaObras, aliases: []string{"ladrilheiro"},
		min: 110, max: 190, every: 20 * time.Minute,
		finds: []string{"pedra", "carteira-perdida", "caveira", "moeda-do-imperio", "vaso-antigo"}, findChance: 0.08,
		tasks: []string{
			"assentou porcelanato com junta de 1 milímetro", "rejuntou o banheiro inteiro de joelho",
			"quebrou o piso antigo e achou um piso mais antigo embaixo", "fez mosaico de pastilha na piscina",
			"cortou azulejo na mão e só quebrou três", "deixou o piso tão liso que a sogra escorregou",
		},
	},
	{
		name: "soldador", emoji: "👨‍🏭", title: "é soldador", area: areaObras, min: 130, max: 230, every: 28 * time.Minute,
		bust: 0.1, busts: []string{
			"esqueceu a máscara e ficou vendo estrelinha o dia todo", "soldou a própria bota no portão",
			"queimou a sobrancelha com uma faísca",
		},
		craft:  craft{activity: "!minerar", noBreak: true},
		hazard: hazard{chance: 0.1, harm: 30, wounds: []string{"soldou sem máscara e queimou a vista", "levou uma faísca dentro da bota"}},
		tasks: []string{
			"soldou a estrutura de um galpão inteiro", "consertou o carrinho de mão com solda de emergência",
			"fez churrasqueira de tambor pro churrasco de domingo", "remendou a lataria de um Gol 94",
			"soldou o portão da escola no último dia de aula", "fez faísca igual fogos de réveillon",
		},
	},
	{
		name: "vidraceiro", emoji: "🪟", title: "é vidraceiro", area: areaObras, min: 120, max: 220, every: 20 * time.Minute,
		bust: 0.2, bustFine: 100, busts: []string{
			"deixou cair a chapa de vidro temperado na calçada", "trincou o box do banheiro na hora de encaixar",
			"instalou o espelho e o espelho caiu na primeira batida de porta", "cortou o vidro 2 centímetros menor",
		},
		doubleChance: 0.12,
		hazard:       hazard{chance: 0.1, harm: 30, wounds: []string{"deixou a vidraça escorregar e se cortou todo", "atravessou uma porta de vidro achando que tava aberta"}},
		tasks: []string{
			"instalou um box de banheiro sem nenhuma bolha de silicone", "trocou a janela que o filho do vizinho quebrou com a bola",
			"colocou espelho na academia inteira", "fez fachada de vidro numa loja de shopping",
			"carregou um vidro de 3 metros sem respirar", "instalou a porta de vidro e o cliente bateu a cara nela",
		},
	},
	{
		name: "operador de retroescavadeira", emoji: "🚜", title: "é operador de retroescavadeira", area: areaObras,
		aliases: []string{"retroescavadeira", "retro", "operador"},
		min:     150, max: 250, every: 36 * time.Minute,
		craft:  craft{activity: "!minerar", faster: 30, luck: 1, grade: -1, noTool: true},
		hazard: hazard{chance: 0.05, harm: 45, wounds: []string{"tombou a retroescavadeira dentro da vala", "bateu a cabeça no teto da cabine num buraco"}},
		tasks: []string{
			"cavou uma vala e achou um cano da Sabesp do jeito mais caro", "abriu um buraco do tamanho de uma piscina olímpica",
			"fez manobra de retro pra plateia de criança na calçada", "derrubou um muro que era pra derrubar, dessa vez",
			"terraplanou um terreno inteiro antes do almoço", "tirou um carro do atoleiro com a pá da retro",
		},
	},

	{
		name: "motoboy", emoji: "🛵", title: "é motoboy", area: areaTransporte, min: 50, max: 110, every: 8 * time.Minute,
		robEvery: 15 * time.Minute,
		hazard:   hazard{chance: 0.12, harm: 45, wounds: []string{"levou uma fechada de ônibus e beijou o asfalto", "derrapou no óleo da avenida", "passou no corredor e arrancou um retrovisor com o joelho"}},
		tasks: []string{
			"entregou um iFood debaixo de chuva", "fez 14 entregas e levou 3 multas", "entregou uma pizza de cabeça pra baixo e ainda ganhou gorjeta",
			"cortou caminho pela calçada", "entregou um lanche frio com um sorriso quente", "subiu 12 andares de escada porque o elevador quebrou",
			"marcou o pedido como entregue ainda na esquina", "buzinou no corredor da marginal por 40 minutos seguidos",
		},
	},
	{
		name: "motorista de aplicativo", emoji: "🚗", title: "é motorista de aplicativo", area: areaTransporte, aliases: []string{"uber", "99"},
		min: 110, max: 210, every: 20 * time.Minute,
		bust: 0.1, peak: peak{from: 17, to: 20, bonus: 50},
		hazard: hazard{chance: 0.04, harm: 30, wounds: []string{"levou uma coronhada de um passageiro mal-intencionado", "bateu o carro seguindo o GPS pelo meio de uma feira"}},
		tasks: []string{
			"levou um passageiro que contou a vida inteira em 15 minutos", "ofereceu balinha e água pra garantir as 5 estrelas",
			"pegou uma corrida com tarifa dinâmica no meio da chuva", "fingiu que não ouviu o passageiro pedindo pra trocar de música",
			"levou um bêbado pra casa e ele ainda pagou certinho", "fez a mesma corrida do aeroporto três vezes no dia",
			"seguiu o Waze por uma rua de terra e chegou antes do previsto",
		},
		busts: []string{
			"levou um passageiro que vomitou no banco de trás", "teve a conta bloqueada por uma avaliação de 1 estrela",
			"rodou duas horas sem nenhuma corrida aceita", "pegou um passageiro que cancelou quando ele chegou na porta",
		},
	},
	{
		name: "taxista", emoji: "🚕", title: "é taxista", area: areaTransporte, min: 110, max: 190, every: 20 * time.Minute,
		peak: peak{from: 22, to: 5, bonus: 60}, finds: []string{"guarda-chuva", "carteira-perdida", "nokia", "cd-pirata"}, findChance: 0.1,
		tasks: []string{
			"reclamou do Uber com todos os passageiros do dia", "deu a volta no quarteirão três vezes com o taxímetro ligado",
			"explicou como consertaria o país em uma corrida de 10 minutos", "levou um gringo ao aeroporto pela rota mais longa possível",
			"disse que a maquininha de cartão estava sem sinal", "ouviu o jogo no rádio no volume máximo com o passageiro dentro",
			"furou a fila do ponto de táxi na rodoviária",
		},
	},
	{
		name: "caminhoneiro", emoji: "🚛", title: "é caminhoneiro", area: areaTransporte, min: 300, max: 525, every: 40 * time.Minute,
		bust: 0.15, bustFine: 110,
		finds: []string{"pneu", "marmita", "energetico", "cd-pirata"}, findChance: 0.15,
		hazard: hazard{chance: 0.05, harm: 50, wounds: []string{"cochilou no volante e acordou no acostamento", "pegou neblina na serra e bateu no guard-rail"}},
		tasks: []string{
			"cruzou o país ouvindo o mesmo CD de modão", "comeu um PF de posto que deixou saudade", "levou uma carga de melancia de Goiás até a Bahia",
			"buzinou pras crianças que fizeram sinal na estrada", "dormiu na boleia e acordou com o caminhão carregado",
			"passou pela balança sem ninguém perceber o excesso", "pintou uma frase de para-choque nova na carroceria",
		},
		busts: []string{
			"furou dois pneus no meio da BR", "foi parado pela PRF com o tacógrafo zoado",
			"ficou três dias parado numa greve na rodovia", "errou a entrada e foi parar no Paraguai",
		},
	},
	{
		name: "motorista de ônibus", emoji: "🚌", title: "é motorista de ônibus", area: areaTransporte, min: 100, max: 160, every: 20 * time.Minute,
		dailyBonus: 40,
		tasks: []string{
			"passou direto pelo ponto fingindo que não viu ninguém", "fez a curva tão rápido que o ônibus inteiro foi pro lado",
			"esperou a velhinha correr até o ponto e virou herói", "freou do nada e todo mundo agradeceu ao santo de devoção",
			"fechou a porta na cara de um atrasado", "fez a linha inteira com o rádio na Jovem Pan",
			"parou fora do ponto pra pegar um pastel na feira",
		},
	},
	{
		name: "cobrador de ônibus", emoji: "🎫", title: "é cobrador de ônibus", area: areaTransporte, min: 70, max: 130, every: 16 * time.Minute,
		interest: 2, finds: []string{"boleto", "jornal", "carteira-perdida", "nota-de-tres"}, findChance: 0.08,
		tasks: []string{
			"deu troco em moedas de 5 centavos pra um cara com nota de 50", "gritou \"vai descer, motorista!\" umas vinte vezes",
			"liberou a catraca pra um estudante sem carteirinha", "contou o caixa do dia três vezes e bateu na segunda",
			"cochilou na cadeirinha entre um ponto e outro", "arrumou briga com quem tentou passar por baixo da catraca",
		},
	},
	{
		name: "piloto de avião", emoji: "✈️", title: "é piloto de avião", area: areaTransporte, min: 255, max: 390, every: 40 * time.Minute,
		bust: 0.05, discount: 15,
		hazard: hazard{chance: 0.02, harm: 90, wounds: []string{"fez um pouso de barriga em Congonhas", "pegou uma turbulência e bateu a cabeça no painel"}},
		tasks: []string{
			"pousou tão macio que ninguém bateu palma", "pousou tão duro que todo mundo bateu palma", "avisou da turbulência e foi tomar café",
			"fez um voo de Congonhas pro Galeão e reclamou do trânsito", "anunciou o tempo de voo com aquela voz de rádio FM",
			"deu uma volta a mais em Brasília só pra ver a Esplanada", "acumulou mais milhas que todo o grupo junto",
		},
		busts: []string{
			"teve o voo cancelado por causa do tempo", "ficou preso no aeroporto por overbooking da própria tripulação",
			"esqueceu o passaporte em casa num voo internacional",
		},
	},
	{
		name: "carteiro", emoji: "📮", title: "é carteiro", area: areaTransporte, min: 80, max: 140, every: 16 * time.Minute,
		bust: 0.1, finds: []string{"jornal", "boleto", "caixa-misteriosa", "carteira-perdida"}, findChance: 0.18,
		tasks: []string{
			"entregou uma conta de luz atrasada com a maior cara de pau", "deixou o aviso de \"não havia ninguém em casa\" sem tocar a campainha",
			"subiu a ladeira mais íngreme do bairro com a bolsa cheia", "entregou uma encomenda da Shein que demorou três meses",
			"leu o cartão-postal dos outros no caminho", "fez a rota inteira de bicicleta debaixo do sol de meio-dia",
		},
		busts: []string{
			"foi mordido pelo pinscher da rua 4", "foi perseguido por um vira-lata por três quarteirões",
			"perdeu um pacote e passou o dia procurando",
		},
	},
	{
		name: "mototaxista", emoji: "🏍️", title: "é mototaxista", area: areaTransporte, min: 40, max: 90, every: 8 * time.Minute,
		robBonus: 0.1,
		hazard:   hazard{chance: 0.1, harm: 40, wounds: []string{"caiu da moto com passageiro e tudo num buraco", "foi fechado por um carro que não deu seta"}},
		tasks: []string{
			"levou uma senhora com três sacolas de feira na garupa", "subiu o morro empinando com passageiro e tudo",
			"emprestou o capacete que não fecha", "cortou o engarrafamento pelo meio dos carros",
			"levou o passageiro até a porta e ainda esperou ele pegar a carteira", "ficou no ponto jogando dominó entre uma corrida e outra",
		},
	},
	{
		name: "frentista", emoji: "⛽", title: "é frentista", area: areaTransporte, min: 90, max: 150, every: 16 * time.Minute,
		bust: 0.1, bustFine: 80,
		finds: []string{"energetico", "cafe", "latao", "todynho"}, findChance: 0.12,
		tasks: []string{
			"perguntou \"completa?\" pra todo mundo que só queria 20 reais", "limpou o para-brisa sem ninguém pedir",
			"empurrou um aditivo que ninguém sabe pra que serve", "calibrou os pneus e encheu o tanque ao mesmo tempo",
			"vendeu um pacote de bolacha na loja de conveniência por 15 reais", "deu o golpe da bomba arredondando pra cima",
		},
		busts: []string{
			"pôs gasolina num carro a diesel", "saiu com o bico da bomba ainda no tanque",
			"deixou o motoqueiro ir embora sem pagar",
		},
	},
	{
		name: "manobrista", emoji: "🔑", title: "é manobrista", area: areaTransporte, min: 45, max: 85, every: 8 * time.Minute,
		theftEvery: 10 * time.Minute, theftBonus: 0.15,
		tasks: []string{
			"deu uma voltinha no carro do cliente antes de estacionar", "estacionou uma SUV numa vaga de Fusca",
			"ajustou o banco e o retrovisor do cliente todo", "achou a chave certa num quadro com 80 chaves iguais",
			"devolveu o carro com o rádio numa estação de pagode", "ganhou gorjeta só porque abriu a porta pra madame",
		},
	},
	{
		name: "entregador de gás", emoji: "🔥", title: "é entregador de gás", area: areaTransporte, min: 110, max: 190, every: 20 * time.Minute,
		doubleChance: 0.08, peak: peak{from: 8, to: 12, bonus: 50},
		hazard: hazard{chance: 0.08, harm: 30, wounds: []string{"travou a lombar subindo 10 andares com o botijão nas costas", "deixou o botijão cair no pé"}},
		tasks: []string{
			"passou na rua tocando Pour Elise no último volume", "subiu quatro andares com o botijão no ombro",
			"trocou o botijão e ainda testou o fogão com sabão", "chegou bem na hora em que a vizinha estava no meio do almoço",
			"deixou o botijão vazio de recordação na área de serviço", "fez a entrega numa casa em cima do morro de primeira",
			"trocou ideia com a dona de casa enquanto ela procurava o dinheiro",
		},
	},
	{
		name: "chef", emoji: "👨‍🍳", title: "é chef", area: areaComida, min: 170, max: 290, every: 28 * time.Minute,
		peak: peak{from: 19, to: 23, bonus: 40}, finds: []string{"pizza", "pudim", "chocolate", "picanha"}, findChance: 0.1,
		hazard: hazard{chance: 0.07, harm: 20, wounds: []string{"se queimou no óleo da fritura", "cortou o dedo picando cebola rápido demais"}},
		tasks: []string{
			"gritou \"sim, chef!\" pra si mesmo", "montou um prato de 3 garfadas por 180 reais",
			"jogou o risoto do estagiário no lixo e mandou refazer", "passou uma redução de balsâmico em tudo",
			"chamou feijão com arroz de \"leguminosas sobre cama de grãos\"", "provou o molho com o dedo quando ninguém estava olhando",
			"virou uma frigideira com fogo pra plateia",
		},
	},
	{
		name: "padeiro", emoji: "🥖", title: "é padeiro", area: areaComida, min: 110, max: 170, every: 20 * time.Minute,
		peak: peak{from: 5, to: 9, bonus: 60}, finds: []string{"pao-de-queijo", "cafe", "todynho"}, findChance: 0.1,
		tasks: []string{
			"acordou às 4h pra sovar a massa", "tirou a fornada de pão francês na hora em que a fila virou a esquina",
			"vendeu pão de ontem como se fosse de hoje", "fez um sonho com o recheio todo de um lado só",
			"queimou o fundo de uma fornada e chamou de \"tostadinho\"", "separou o pão mais moreninho pro cliente chato",
		},
	},
	{
		name: "confeiteiro", emoji: "🎂", title: "é confeiteiro", area: areaComida, min: 130, max: 230, every: 20 * time.Minute,
		bust: 0.12, bustFine: 60,
		finds: []string{"chocolate", "pudim"}, findChance: 0.15,
		tasks: []string{
			"fez um bolo de aniversário com o nome do cliente escrito errado", "vendeu 200 brigadeiros pra uma festa de criança",
			"lambeu a colher do chocolate quando ninguém viu", "montou um bolo de três andares de brigadeiro e ninho com morango",
			"postou o bolo no Instagram com filtro de padaria francesa", "fez um bolo de pote e cobrou como se fosse obra de arte",
		},
		busts: []string{
			"deixou o bolo de casamento desabar na entrada do salão", "trocou o açúcar pelo sal na massa inteira",
			"esqueceu o pudim no forno e fez um tijolo", "deixou o glacê derreter no sol do caminho",
		},
	},
	{
		name: "pizzaiolo", emoji: "🍕", title: "é pizzaiolo", area: areaComida, min: 120, max: 200, every: 20 * time.Minute,
		doubleChance: 0.06, peak: peak{from: 19, to: 23, bonus: 60},
		hazard: hazard{chance: 0.07, harm: 20, wounds: []string{"encostou o braço no forno a lenha", "deixou a pá de pizza cair no pé"}},
		tasks: []string{
			"girou a massa no ar e ela grudou no teto", "fez uma pizza de strogonoff e jurou que é tradicional",
			"colocou ketchup na pizza e fingiu que não foi ele", "assou 40 pizzas numa sexta de jogo do Brasil",
			"economizou na calabresa e caprichou no orégano", "fez uma meia portuguesa meia doce de leite a pedido do cliente",
		},
	},
	{
		name: "churrasqueiro", emoji: "🥩", title: "é churrasqueiro", area: areaComida, min: 130, max: 230, every: 20 * time.Minute,
		bust: 0.1, finds: []string{"latao", "picanha", "dolly"}, findChance: 0.08,
		hazard: hazard{chance: 0.08, harm: 20, wounds: []string{"jogou álcool na brasa e a churrasqueira explodiu na cara", "pegou o espeto pela ponta quente"}},
		tasks: []string{
			"passou a tarde inteira na churrasqueira e não comeu nada", "serviu o pão de alho primeiro pra ganhar tempo",
			"cortou a picanha na espessura certa e foi aplaudido", "acendeu o carvão com secador de cabelo",
			"jurou que o sal grosso é segredo de família", "ficou de dono da churrasqueira com uma lata na mão",
		},
		busts: []string{
			"deixou a carne virar sola de sapato", "deixou o cachorro roubar a linguiça da grelha",
			"choveu e o churrasco virou feijoada",
		},
	},
	{
		name: "barista", emoji: "☕", title: "é barista", area: areaComida, min: 55, max: 95, every: 8 * time.Minute,
		peak: peak{from: 7, to: 10, bonus: 40}, finds: []string{"cafe", "pao-de-queijo", "chocolate"}, findChance: 0.12,
		tasks: []string{
			"desenhou um coração no cappuccino que parecia um rim", "escreveu o nome do cliente errado no copo de propósito",
			"explicou as notas frutadas de um café de 3 reais", "fez um latte com leite de aveia pra um coach",
			"tirou um espresso tão forte que o cliente viu Deus", "julgou em silêncio quem pediu café com açúcar",
		},
	},
	{
		name: "garçom", emoji: "🍽️", title: "é garçom", area: areaComida, min: 90, max: 160, every: 16 * time.Minute,
		bust: 0.1, bustFine: 50, doubleChance: 0.12,
		tasks: []string{
			"anotou o pedido de 12 pessoas de cabeça e acertou tudo", "trouxe a saideira que ninguém pediu",
			"disse \"já vai sair\" pra um prato que nem tinha entrado no forno", "equilibrou cinco chopes numa bandeja só",
			"cobrou os 10% e ainda ganhou caixinha", "fingiu que não viu a mesa chamando por meia hora",
			"recomendou o prato mais caro com a maior cara de amigo",
		},
		busts: []string{
			"derrubou a bandeja inteira em cima do cliente", "trouxe o pedido da mesa errada três vezes seguidas",
			"levou um calote de uma mesa de oito pessoas",
		},
	},
	{
		name: "sushiman", emoji: "🍣", title: "é sushiman", area: areaComida, min: 140, max: 220, every: 28 * time.Minute,
		craft: craft{activity: "!pescar", luck: 1, grade: 1},
		tasks: []string{
			"enrolou um hot roll com cream cheese até no talo", "cortou um salmão com precisão de cirurgião",
			"botou cebolinha e cream cheese num temaki e chamou de japonês", "montou uma barca de rodízio pra seis pessoas",
			"gritou \"irasshaimase\" sem saber o que significa", "fez um sushi de morango com Nutella a pedido do cliente",
		},
	},
	{
		name: "açougueiro", emoji: "🔪", title: "é açougueiro", area: areaComida, min: 90, max: 150, every: 20 * time.Minute,
		guard: 0.15, finds: []string{"picanha", "sal-grosso"}, findChance: 0.06, killChance: 0.25,
		hazard: hazard{chance: 0.1, harm: 30, wounds: []string{"errou a machadinha e quase levou o dedo junto", "escorregou na câmara fria e caiu de costas"}},
		tasks: []string{
			"moeu a carne de segunda e chamou de patinho", "jogou um bife no balcão com aquela força desnecessária",
			"perguntou \"mais alguma coisa?\" pra fila inteira", "vendeu uma picanha que era maminha",
			"afiou o cutelo olhando fixo pro cliente que pechinchou", "separou o osso pro cachorro da vizinha",
		},
	},
	{
		name: "pipoqueiro", emoji: "🍿", title: "é pipoqueiro", area: areaComida, min: 60, max: 100, every: 8 * time.Minute,
		bust: 0.1, peak: peak{from: 17, to: 21, bonus: 50},
		tasks: []string{
			"vendeu pipoca doce na porta da escola", "botou bacon na pipoca e dobrou o preço",
			"estacionou o carrinho na saída da missa", "deu um saquinho de brinde pra criança que chorou",
			"fez pipoca colorida que deixou a língua de todo mundo azul", "vendeu tudo na porta do estádio antes do apito",
		},
		busts: []string{
			"queimou o milho inteiro e a rua toda ficou com cheiro", "foi expulso da praça pelo fiscal da prefeitura",
			"virou o carrinho no buraco da calçada",
		},
	},
	{
		name: "sorveteiro", emoji: "🍦", title: "é sorveteiro", area: areaComida, min: 80, max: 140, every: 16 * time.Minute,
		bust: 0.15, peak: peak{from: 12, to: 16, bonus: 70},
		tasks: []string{
			"empurrou o carrinho na areia da praia o dia todo", "vendeu picolé de limão pra uma escola inteira",
			"tocou a buzininha do carrinho e juntou uma fila de criança", "deu a última bola de flocos de brinde",
			"vendeu um açaí com mais granola do que açaí", "aproveitou o calor de 40 graus pra subir o preço",
		},
		busts: []string{
			"choveu o dia inteiro e ninguém quis sorvete", "deixou o freezer desligado e o estoque virou sopa",
			"foi atropelado por uma bola de futevôlei e derrubou tudo",
		},
	},
	{
		name: "bartender", emoji: "🍸", title: "é bartender", area: areaComida, min: 90, max: 150, every: 20 * time.Minute,
		peak: peak{from: 22, to: 3, bonus: 70}, cashback: 15,
		tasks: []string{
			"jogou a garrafa pro alto e pegou nas costas", "ouviu o desabafo de um cliente sobre a ex por três horas",
			"fez uma caipirinha de cachaça barata e cobrou como importada", "inventou um drink com nome de signo",
			"serviu a última dose e fingiu que o bar já tinha fechado", "colocou mais gelo do que bebida no copo",
		},
	},
	{
		name: "marmiteiro", emoji: "🍱", title: "é marmiteiro", area: areaComida, min: 90, max: 150, every: 20 * time.Minute,
		salePercent: 70, finds: []string{"marmita", "pote-de-feijao"}, findChance: 0.12,
		tasks: []string{
			"vendeu 50 marmitas na porta da obra", "colocou uma batata frita a mais pra cliente fiel",
			"fez a mesma marmita de frango e chamou de cardápio do dia", "entregou marmita de bicicleta em três bairros",
			"divulgou o cardápio no grupo da família", "cobrou a mais pela marmita com ovo frito em cima",
		},
	},

	{
		name: "programador", emoji: "👨‍💻", title: "é programador", area: areaEscritorio, aliases: []string{"dev"}, min: 250, max: 450, every: 36 * time.Minute,
		tasks: []string{
			"corrigiu um bug e criou mais três", "fez deploy na sexta às 18h", "copiou uma resposta do Stack Overflow e funcionou",
			"passou 4 horas numa reunião que podia ser um e-mail", "explicou pro cliente que não é só um botãozinho",
			"reiniciou o servidor e chamou de manutenção", "culpou o cache e foi tomar um café", "disse que na máquina dele funciona",
		},
	},
	{
		name: "estagiário", emoji: "🧑‍🎓", title: "é estagiário", area: areaEscritorio, aliases: []string{"estagiario", "estagiária"}, min: 40, max: 100,
		every:      8 * time.Minute,
		finds:      []string{"cafe", "marmita", "todynho", "energetico", "pao-de-queijo"},
		findChance: 0.2,
		heistBonus: -0.05,
		tasks: []string{
			"passou o café da firma inteira", "tirou xerox de 300 páginas sem saber pra quê", "foi buscar o almoço do chefe a pé",
			"formatou a planilha e apagou metade sem querer", "ficou até as 20h sem ganhar hora extra",
			"levou a culpa pelo erro do sênior", "atualizou o LinkedIn dizendo que é CEO de si mesmo",
		},
	},
	{
		name: "analista de dados", emoji: "📊", title: "é analista de dados", area: areaEscritorio, aliases: []string{"analista"}, min: 140, max: 240,
		every: 28 * time.Minute,
		craft: craft{activity: "!vasculhar", luck: 1},
		tasks: []string{
			"fez um gráfico de pizza que ninguém entendeu", "descobriu que as vendas caem quando chove",
			"passou o dia limpando uma planilha cheia de célula mesclada", "torturou os dados até eles confessarem",
			"montou um dashboard que só ele abre", "provou com regressão linear que segunda-feira é ruim",
			"rodou um PROCV de 40 minutos que travou o Excel",
		},
	},
	{
		name: "designer", emoji: "🎨", title: "é designer", area: areaEscritorio, min: 160, max: 300, every: 24 * time.Minute,
		bust: 0.25, busts: []string{
			"ouviu que o logo ficou bonito mas quer mais vida", "mandou o arquivo e o cliente abriu no Paint",
			"teve que aumentar o logo até não caber mais na tela", "perdeu o PSD de 3 GB porque o Photoshop fechou sozinho",
		},
		peak: peak{from: 22, to: 3, bonus: 50},
		tasks: []string{
			"mudou a fonte pra Comic Sans a pedido do cliente", "fez 15 versões do mesmo azul",
			"entregou o projeto final_v2_agora_vai_FINAL.png", "trocou o fundo branco por um branco mais quente",
			"fez um flyer de pagode com efeito de brilho", "explicou que fazer logo não é só abrir o Canva",
		},
	},
	{
		name: "analista de rh", emoji: "🧑‍💼", title: "é analista de rh", area: areaEscritorio, aliases: []string{"rh"}, min: 110, max: 190,
		every:      20 * time.Minute,
		finds:      []string{"caixa-misteriosa", "chocolate", "ursinho", "cafe", "pudim"},
		findChance: 0.15,
		tasks: []string{
			"organizou o amigo secreto da firma", "fez uma dinâmica de grupo com bexiga", "mandou um e-mail sobre o uso correto da geladeira",
			"perguntou onde o candidato se vê daqui a 5 anos", "reprovou alguém por usar Comic Sans no currículo",
			"montou o mural dos aniversariantes do mês", "chamou demissão de oportunidade de recomeço",
		},
	},
	{
		name: "contador", emoji: "🧮", title: "é contador", area: areaEscritorio, aliases: []string{"contadora"}, min: 120, max: 200,
		every:         24 * time.Minute,
		legalDiscount: 50,
		tasks: []string{
			"fez o imposto de renda do bairro inteiro", "achou R$ 0,03 de diferença e passou o dia caçando",
			"explicou pro MEI que a conta da empresa não é a dele", "entregou a declaração 23h59 do último dia",
			"carimbou 200 guias seguidas", "fechou o balanço com a calculadora de fita",
			"achou uma nota fiscal de 2014 na gaveta",
		},
	},
	{
		name: "secretário", emoji: "📋", title: "é secretário", area: areaEscritorio, aliases: []string{"secretario", "secretária", "secretaria"},
		min: 70, max: 130, every: 16 * time.Minute,
		theftEvery: 15 * time.Minute, theftBonus: 0.12,
		tasks: []string{
			"falou que o chefe estava em reunião enquanto ele dormia", "atendeu o telefone com voz de rádio",
			"remarcou a mesma reunião cinco vezes", "achou o grampeador que todo mundo procurava",
			"anotou o recado com a letra que só ele entende", "segurou o cliente chato na recepção com café requentado",
			"descobriu a senha do Wi-Fi da diretoria",
		},
	},
	{
		name: "gerente", emoji: "👔", title: "é gerente", area: areaEscritorio, min: 200, max: 350, every: 36 * time.Minute,
		dailyBonus: 50, abuses: []abuseKind{workAbuse},
		tasks: []string{
			"marcou uma reunião pra falar sobre reuniões", "pediu pra equipe vestir a camisa da empresa",
			"falou sinergia três vezes na mesma frase", "cobrou o relatório que ele mesmo esqueceu de pedir",
			"mandou mensagem no grupo da firma às 23h", "chamou corte de bônus de reestruturação",
			"chegou às 10h pra reclamar de atraso",
		},
	},
	{
		name: "bancário", emoji: "🏦", title: "é bancário", area: areaEscritorio, aliases: []string{"bancario", "bancária"}, min: 140, max: 220,
		every:      28 * time.Minute,
		interest:   4,
		heistBonus: 0.12,
		tasks: []string{
			"vendeu um título de capitalização pra vovó", "falou que o sistema caiu e foi almoçar",
			"empurrou um seguro junto com o empréstimo", "pediu pra cliente voltar amanhã com o comprovante de residência",
			"bateu a meta de cartão de crédito no último dia", "atendeu a fila do INSS no dia do pagamento",
			"explicou pela décima vez o que é o Pix",
		},
	},
	{
		name: "economista", emoji: "📉", title: "é economista", area: areaEscritorio, min: 130, max: 210, every: 28 * time.Minute,
		interest: 2, cashback: 15,
		tasks: []string{
			"previu a crise que já tinha acontecido", "explicou a inflação usando o preço do tomate",
			"deu entrevista no jornal falando que depende", "fez uma planilha provando que o churrasco ficou caro",
			"errou a previsão do dólar com muita confiança", "culpou o mercado externo por tudo",
			"falou que o certo era ter comprado dólar em 2010",
		},
	},
	{
		name: "trader", emoji: "📈", title: "é trader", area: areaEscritorio, min: 250, max: 550, every: 28 * time.Minute,
		bust: 0.4, bustFine: 250, busts: []string{
			"comprou no topo e vendeu no fundo", "fez all-in numa moeda com nome de cachorro", "foi liquidado na alavancagem de 100x",
			"seguiu a dica de um coach do Instagram",
		},
		doubleChance: 0.15, freeBrokerage: true,
		tasks: []string{
			"acertou o day trade e postou print no grupo", "vendeu antes da queda por pura sorte",
			"ficou olhando gráfico de vela até ver um rosto", "fez a análise técnica olhando o horóscopo",
			"comprou na baixa e jurou que foi estratégia", "lucrou com o dólar e contou pra família toda",
		},
	},
	{
		name: "suporte de ti", emoji: "🖥️", title: "é do suporte de ti", area: areaEscritorio, aliases: []string{"ti", "suporte"}, min: 90, max: 160,
		every:      16 * time.Minute,
		finds:      []string{"cabo-usb", "controle-remoto", "nokia", "fone", "cd-pirata", "polystation"},
		findChance: 0.2,
		tasks: []string{
			"perguntou se já tentaram desligar e ligar de novo", "descobriu que o computador estava fora da tomada",
			"tirou 3 barras de ferramentas do navegador da diretora", "trocou o toner e saiu com a mão preta",
			"formatou o PC do tio e instalou o Windows pirata", "consertou a impressora com um tapa",
			"explicou que a internet não é o ícone azul do Chrome",
		},
	},
	{
		name: "corretor de imóveis", emoji: "🏠", title: "é corretor de imóveis", area: areaEscritorio, aliases: []string{"corretor", "corretora"},
		min: 180, max: 380, every: 28 * time.Minute,
		bust: 0.3, busts: []string{
			"mostrou o apê e o cliente viu a infiltração", "perdeu a venda pro primo do comprador que também é corretor",
			"levou o cliente no imóvel errado", "ficou 2 horas esperando um cliente que nunca apareceu",
		},
		salePercent: 75,
		tasks: []string{
			"chamou um quarto de 9 m² de aconchegante", "vendeu um terreno em área de mangue como vista pro mar",
			"anunciou a kitnet como studio conceito aberto", "disse que o barulho da avenida é sinal de valorização",
			"alugou uma casa sem garagem dizendo que a rua é tranquila", "fechou contrato e ainda pegou a comissão do fiador",
		},
	},
	{
		name: "médico", emoji: "🩺", title: "é médico", area: areaSaude, aliases: []string{"medico", "médica", "doutor"}, min: 270, max: 440,
		every:        40 * time.Minute,
		doubleChance: 0.12,
		tasks: []string{
			"receitou dipirona e repouso", "atendeu 40 pessoas num plantão de 12 horas", "escreveu uma receita que só a farmácia entende",
			"mandou o paciente parar de pesquisar sintoma no Google", "deu atestado pra quem estava só com preguiça",
			"falou que é virose e mandou beber água", "foi chamado de doutor no churrasco pra olhar uma pinta",
		},
	},
	{
		name: "enfermeiro", emoji: "💉", title: "é enfermeiro", area: areaSaude, aliases: []string{"enfermeira"}, min: 120, max: 200,
		every: 24 * time.Minute,
		peak:  peak{from: 19, to: 7, bonus: 40},
		tasks: []string{
			"achou a veia de primeira", "segurou o plantão enquanto o médico dormia", "deu injeção na bunda de marmanjo chorão",
			"trocou soro, curativo e ainda fez o café da equipe", "aguentou acompanhante gritando na triagem",
			"aferiu a pressão de um tio que jurou que estava ótimo",
		},
	},
	{
		name: "dentista", emoji: "🦷", title: "é dentista", area: areaSaude, min: 200, max: 340, every: 32 * time.Minute,
		finds:      []string{"chocolate", "pudim", "todynho", "ouro"},
		findChance: 0.1,
		tasks: []string{
			"fez pergunta pro paciente com a boca cheia de algodão", "arrancou um siso que estava deitado",
			"perguntou se o paciente usa fio dental e ouviu uma mentira", "fez clareamento em quem toma café o dia todo",
			"ligou o motorzinho e a sala de espera esvaziou", "colocou aparelho num adulto de 40 anos",
			"vendeu lente de contato dental pra influencer",
		},
	},
	{
		name: "veterinário", emoji: "🐶", title: "é veterinário", area: areaSaude, aliases: []string{"veterinario", "veterinária", "vet"},
		min: 150, max: 260, every: 28 * time.Minute,
		finds:      []string{"pombo", "rato", "sapo", "gato-preto", "papagaio"},
		findChance: 0.12,
		hazard:     hazard{chance: 0.07, harm: 20, wounds: []string{"levou uma mordida de um chihuahua estressado", "levou um coice de um cavalo nervoso"}},
		tasks: []string{
			"vacinou um gato que lutou como se fosse o UFC", "tirou um carrapato do tamanho de uma uva",
			"atendeu um vira-lata caramelo que comeu uma meia", "castrou o cachorro da vizinha sem ela saber o que era",
			"tomou mordida de um chihuahua de 2 kg", "explicou pra dona que o pet não pode comer chocolate",
			"cuidou de um papagaio que só sabia xingar",
		},
	},
	{
		name: "farmacêutico", emoji: "💊", title: "é farmacêutico", area: areaSaude, aliases: []string{"farmaceutico", "farmacêutica"}, min: 110, max: 180,
		every:    24 * time.Minute,
		discount: 15,
		tasks: []string{
			"pediu o CPF pra dar desconto", "decifrou a letra do médico de primeira",
			"vendeu Engov pra galera na sexta à noite", "empurrou o genérico dizendo que é a mesma coisa",
			"mediu a pressão do aposentado que vai lá todo dia só pra conversar", "repôs a prateleira de camisinha e de teste de gravidez",
		},
	},
	{
		name: "psicólogo", emoji: "🧠", title: "é psicólogo", area: areaSaude, aliases: []string{"psicologo", "psicóloga", "psi"}, min: 140, max: 240,
		every: 28 * time.Minute,
		guard: 0.15,
		tasks: []string{
			"perguntou e como você se sente sobre isso", "ouviu uma hora de reclamação sobre a sogra",
			"descobriu que o problema era a mãe", "deu alta pra um paciente que ainda manda áudio de 10 minutos",
			"atendeu o casal que brigou dentro do consultório", "analisou o grupo da família e ficou preocupado",
			"falou que vamos trabalhar isso na próxima sessão",
		},
	},
	{
		name: "fisioterapeuta", emoji: "🦵", title: "é fisioterapeuta", area: areaSaude, aliases: []string{"fisio"}, min: 110, max: 190,
		every: 24 * time.Minute,
		craft: craft{activity: "!minerar", faster: 25},
		tasks: []string{
			"estalou as costas de um tio e ele saiu andando reto", "passou exercício que ninguém vai fazer em casa",
			"colocou bolsa de gelo no joelho do jogador de pelada", "corrigiu a postura de quem trabalha deitado na cama",
			"atendeu o paciente que piorou fazendo crossfit", "usou o choquinho e o paciente achou que era tortura",
		},
	},
	{
		name: "nutricionista", emoji: "🥗", title: "é nutricionista", area: areaSaude, aliases: []string{"nutri"}, min: 55, max: 100, every: 8 * time.Minute,
		finds:      []string{"coxinha", "pastel", "podrao", "pizza", "chocolate", "pao-de-queijo"},
		findChance: 0.18,
		tasks: []string{
			"cortou o pão francês da dieta do paciente", "montou um cardápio com chia em tudo",
			"pegou o paciente comendo coxinha escondido", "mandou trocar o refrigerante por água com limão",
			"explicou que suco de caixinha não é fruta", "pesou o paciente que jurou que a balança estava errada",
			"liberou um dia do lixo e o paciente fez uma semana",
		},
	},
	{
		name: "cirurgião plástico", emoji: "👃", title: "é cirurgião plástico", area: areaSaude, aliases: []string{"cirurgiao plastico", "cirurgião", "plástica"},
		min: 285, max: 465, every: 40 * time.Minute,
		bust: 0.15, bustFine: 225, busts: []string{
			"deixou o nariz do paciente torto pro outro lado", "foi processado porque a harmonização ficou harmoniosa demais",
			"esqueceu a pinça e teve que abrir de novo", "operou a pessoa errada da agenda",
		},
		cashback: 20,
		tasks: []string{
			"fez a terceira rinoplastia da mesma influencer", "colocou silicone e postou o antes e depois",
			"fez harmonização facial em metade do Big Brother", "tirou a papada de um empresário em 40 minutos",
			"fez uma lipo e cobrou o preço de um carro", "deu entrevista dizendo que beleza vem de dentro",
		},
	},
	{
		name: "socorrista", emoji: "🚑", title: "é socorrista", area: areaSaude, aliases: []string{"samu"}, min: 100, max: 170, every: 20 * time.Minute,
		peak: peak{from: 0, to: 6, bonus: 80},
		tasks: []string{
			"atendeu um bêbado que caiu do próprio banco do bar", "chegou na ocorrência antes da polícia",
			"socorreu um tio que tentou consertar o telhado sozinho", "fez massagem cardíaca no ritmo de Stayin' Alive",
			"tirou a criança que prendeu a cabeça na grade", "atravessou a cidade de sirene ligada às 3 da manhã",
		},
	},
	{
		name: "personal trainer", emoji: "🏋️", title: "é personal trainer", area: areaSaude, aliases: []string{"personal"}, min: 90, max: 150,
		every:         20 * time.Minute,
		peak:          peak{from: 5, to: 9, bonus: 60},
		robBonus:      0.1,
		breakoutBonus: 0.1,
		tasks: []string{
			"falou que é só mais uma repetição pela quinta vez", "contou as séries olhando o celular",
			"filmou o aluno fazendo agachamento e postou no stories", "vendeu um pote de whey e um de creatina",
			"mandou o aluno sair da academia e correr na rua", "explicou que não existe dia de pular perna",
			"fez a aula toda de regata e óculos escuros",
		},
	},
	{
		name: "massagista", emoji: "💆", title: "é massagista", area: areaSaude, min: 80, max: 140, every: 16 * time.Minute,
		doubleChance: 0.1,
		peak:         peak{from: 18, to: 22, bonus: 40},
		tasks: []string{
			"desfez um nó nas costas do tamanho de uma bola de gude", "fez uma massagem com pedra quente que esfriou no meio",
			"botou música de spa e o cliente roncou", "massageou o pé de um corredor de maratona",
			"tirou a tensão de um gerente em 50 minutos", "usou óleo de lavanda que deixou a sala cheirando a vó",
		},
	},

	{
		name: "streamer", emoji: "🎮", title: "é streamer", area: areaArte, min: 80, max: 260, every: 16 * time.Minute,
		bust:       0.3,
		heistBonus: -0.1,
		reach:      250, stage: "twitch",
		tasks: []string{
			"fez live de 6 horas jogando Minecraft", "recebeu um pix escrito \"te amo\" na live", "fez react de react",
			"ganhou um sub de um bot", "chorou na live e bombou", "pediu pix na live pra comprar uma cadeira gamer nova",
		},
		busts: []string{"ficou 4 horas em live com 0 viewers", "perdeu a internet no meio da live", "tomou ban de 7 dias na Twitch"},
	},
	{
		name: "youtuber", emoji: "📹", title: "é youtuber", area: areaArte, min: 140, max: 300, every: 28 * time.Minute,
		doubleChance: 0.12,
		heistBonus:   -0.1,
		reach:        250, stage: "youtube",
		tasks: []string{
			"gravou \"testei o produto mais barato da Shopee\"", "fez thumbnail com a boca aberta e uma seta vermelha",
			"pediu pra deixar o like e se inscrever 14 vezes no mesmo vídeo", "fez vlog indo na padaria",
			"postou um vídeo de 40 minutos que podia ser um áudio de 2", "fez \"24 horas dentro do carro\" e dormiu na hora 3",
			"pediu desculpa num vídeo chorando com fundo cinza", "fechou publi de curso de inglês que nunca fez",
		},
	},
	{
		name: "youtuber de unboxing", emoji: "📦", title: "é youtuber de unboxing", area: areaArte,
		aliases: []string{"unboxer", "unboxing"}, min: 70, max: 150, every: 20 * time.Minute,
		bust: 0.15, heistBonus: -0.1, boxLuck: 1, reach: 200, stage: "youtube",
		finds: []string{"caixa-misteriosa", "mercadoria-encalhada", "ring-light", "cupom-vencido"}, findChance: 0.12,
		tasks: []string{
			"abriu uma caixa da Shopee em câmera lenta com música épica de fundo", "gastou 20 minutos de vídeo só tirando o plástico bolha",
			"fez \"abri 100 caixas misteriosas e olha o que veio\"", "ganhou um recebido da marca e fingiu surpresa",
			"cheirou a caixa nova na frente da câmera e disse que cheirava a sucesso", "fez ASMR de fita adesiva e bombou",
		},
		busts: []string{
			"abriu a caixa misteriosa ao vivo e só tinha um tijolo", "cortou o produto junto com a fita do estilete",
			"a encomenda foi taxada e ficou presa na alfândega", "o vídeo levou strike porque aparecia a logo da marca",
		},
	},
	{
		name: "influencer", emoji: "🤳", title: "é influencer", area: areaArte, aliases: []string{"blogueiro", "blogueira"},
		min: 90, max: 210, every: 24 * time.Minute,
		discount:    20,
		reportBonus: 0.1,
		heistBonus:  -0.1,
		reach:       250, stage: "instagram",
		tasks: []string{
			"postou foto do café da manhã que esfriou enquanto tirava 80 fotos", "fez publi de chá detox segurando a garrafa de ponta-cabeça",
			"recebeu um recebido e fez unboxing em 12 stories", "postou \"gente, sumi\" depois de 3 horas sem story",
			"fez dancinha do TikTok no meio do supermercado", "vendeu curso de como ficar rico vendendo curso",
			"tirou foto no espelho da academia sem treinar", "divulgou um cupom de desconto que não funcionava",
		},
	},
	{
		name: "ator", emoji: "🎬", title: "é ator", area: areaArte, aliases: []string{"atriz"}, min: 130, max: 230, every: 28 * time.Minute,
		bust: 0.1, robEvery: 25 * time.Minute, robBonus: 0.1,
		tasks: []string{
			"fez figuração de defunto na novela das 9", "gravou comercial de remédio e falou \"se persistirem os sintomas\" bem rapidinho",
			"fez teatro infantil vestido de árvore", "chorou de verdade numa cena porque tinha cebola no set",
			"fez o papel de policial número 3 numa série de streaming", "ensaiou o monólogo no ônibus e assustou o cobrador",
			"fez Shakespeare na praça pra três pombos e um bêbado",
		},
		busts: []string{
			"esqueceu o texto no meio da peça e começou a cantar", "foi reprovado no teste pro papel de \"pessoa na fila\"",
			"fez a peça pra uma plateia de 2 pessoas e uma era a mãe dele",
		},
	},
	{
		name: "cantor sertanejo", emoji: "🤠", title: "é cantor sertanejo", area: areaArte, aliases: []string{"sertanejo"},
		min: 130, max: 230, every: 28 * time.Minute,
		legalDiscount: 50,
		tasks: []string{
			"cantou modão de corno no rodeio até o povo chorar", "fez uma música sobre a ex e outra sobre a ex da ex",
			"lançou o hit \"Bebi Pinga e Liguei pra Ela\"", "fez live de churrasco com cerveja de patrocínio",
			"cantou música de traição num casamento sem perceber", "fez show na exposição agropecuária em cima de um trator",
			"gravou DVD na pracinha com 15 pessoas e editou pra parecer 15 mil",
		},
	},
	{
		name: "mc de funk", emoji: "🎤", title: "é mc de funk", area: areaArte, aliases: []string{"mc", "funkeiro"},
		min: 90, max: 210, every: 20 * time.Minute,
		bust: 0.15, peak: peak{from: 22, to: 4, bonus: 80}, reach: 150, stage: "tiktok",
		tasks: []string{
			"lançou um passinho novo e o bairro inteiro já sabia no dia seguinte", "fez show em 4 bailes na mesma noite",
			"gravou clipe com uma moto alugada e um cordão de ouro de bijuteria", "botou o nome da quebrada no refrão e virou hino",
			"fez feat com o primo que só falava \"ahn\"", "cantou no churrasco da firma e até o chefe desceu até o chão",
			"fez um proibidão sobre o preço do gás",
		},
		busts: []string{
			"esqueceu a letra do próprio hit", "o som estourou e cantou à capela pra ninguém",
			"o contratante pagou com divulgação", "o baile acabou antes do MC chegar",
		},
	},
	{
		name: "dj", emoji: "🎧", title: "é dj", area: areaArte, min: 100, max: 200, every: 20 * time.Minute,
		findChance: 0.12, finds: []string{"fone", "cabo-usb", "energetico", "latao"},
		peak: peak{from: 23, to: 5, bonus: 50},
		tasks: []string{
			"tocou \"Evidências\" às 4 da manhã e salvou a festa", "levantou os braços pro alto depois de apertar play no pendrive",
			"fez remix de funk com sertanejo e de algum jeito deu certo", "aceitou pedido de música do tio bêbado no casamento",
			"soltou o drop na hora certa e a pista explodiu", "fez um set de 5 horas numa rave no meio do mato",
			"tocou o \"parabéns\" 3 vezes na mesma formatura",
		},
	},
	{
		name: "palhaço", emoji: "🤡", title: "é palhaço", area: areaArte, min: 60, max: 130, every: 16 * time.Minute,
		cashback:   20,
		heistBonus: -0.05,
		tasks: []string{
			"fez bichinho de bexiga e todos viraram cobra", "tomou torta na cara 12 vezes no mesmo aniversário",
			"andou de monociclo no farol fazendo malabarismo com limão", "apertou o nariz de buzina e fez um bebê parar de chorar",
			"fez o número do fusca cheio de palhaço", "ajudou a armar a lona do circo antes do show",
			"fez a criançada rir com a piada do pavê",
		},
	},
	{
		name: "mágico", emoji: "🎩", title: "é mágico", area: areaArte, aliases: []string{"ilusionista"},
		min: 80, max: 180, every: 20 * time.Minute,
		bust: 0.1, theftEvery: 20 * time.Minute, theftBonus: 0.15, escapeBonus: 0.3,
		heistBonus:    0.05,
		breakoutBonus: 0.3,
		hazard:        hazard{chance: 0.05, harm: 30, wounds: []string{"errou o truque de serrar a assistente e serrou a própria mão", "engoliu uma espada que era de verdade"}},
		tasks: []string{
			"tirou um coelho da cartola, mas era o gato do vizinho", "adivinhou a carta da plateia na terceira tentativa",
			"serrou a assistente ao meio e colou de volta com durex", "sumiu com o celular de um tio e devolveu do outro lado da festa",
			"tirou uma moeda da orelha de todas as crianças da festa", "fez truque de cartas no bar e adivinhou a carta errada com toda a confiança",
			"fez a mesa flutuar com um fio de nylon bem visível",
		},
		busts: []string{
			"deixou o pombo fugir de dentro da manga no meio do show", "sumiu com a carteira de um cara e não conseguiu fazer voltar",
			"entregou o truque porque a carta caiu da manga",
		},
	},
	{
		name: "tatuador", emoji: "🖋️", title: "é tatuador", area: areaArte, min: 160, max: 300, every: 28 * time.Minute,
		bust: 0.12, bustFine: 100, doubleChance: 0.1,
		tasks: []string{
			"tatuou o nome da namorada num cara que terminou na semana seguinte", "fez um leão que ficou parecendo um gato",
			"cobriu o nome de uma ex com uma rosa gigante", "fez o símbolo do infinito pela milésima vez",
			"tatuou uma frase em chinês e torceu pra significar \"força\"", "fechou um braço inteiro em 6 horas de sessão",
			"fez tatuagem de casal com a pata do cachorro",
		},
		busts: []string{
			"tatuou \"Familia\" sem acento e teve que refazer de graça", "viu o cliente desmaiar na primeira agulhada e ir embora",
			"errou o traço e a tatuagem virou um borrão", "deixou a máquina pifar no meio do contorno",
		},
	},
	{
		name: "fotógrafo", emoji: "📸", title: "é fotógrafo", area: areaArte, min: 100, max: 200, every: 24 * time.Minute,
		salePercent: 75,
		tasks: []string{
			"fotografou um casamento e ainda pegou o buquê sem querer", "fez ensaio de gestante na cachoeira",
			"fez book de 15 anos com fumaça colorida", "editou a foto até a pessoa não se reconhecer",
			"fotografou o cardápio e fez o pastel parecer um banquete", "tirou 800 fotos de um batizado e entregou 12",
			"pegou a hora dourada perfeita no pôr do sol", "fez foto 3x4 com a pessoa piscando e cobrou igual",
		},
	},
	{
		name: "dublador", emoji: "🎙️", title: "é dublador", area: areaArte, min: 120, max: 240, every: 28 * time.Minute,
		dailyBonus: 50,
		tasks: []string{
			"dublou o vilão do desenho com voz de locutor de rádio", "fez a voz do trailer: \"num mundo onde...\"",
			"dublou anime e gritou o nome do golpe por 40 segundos", "gravou a voz da URA do banco: \"digite 1 para...\"",
			"fez 15 vozes no mesmo episódio e ninguém percebeu", "dublou um filme que passa toda semana na Sessão da Tarde",
			"gravou a voz do GPS falando \"recalculando rota\"",
		},
	},
	{
		name: "estátua viva", emoji: "🗿", title: "é estátua viva", area: areaArte, aliases: []string{"estatua"},
		min: 40, max: 120, every: 8 * time.Minute,
		bust: 0.15, bustFine: 30, findChance: 0.15, finds: []string{"chinelo", "casca-de-banana", "coxinha", "raspadinha", "nota-de-tres"}, escapeBonus: 0.1,
		tasks: []string{
			"ficou 4 horas parado pintado de prata na praça", "assustou um turista mexendo só o olho",
			"fez pose de pensador e um pombo pousou na cabeça", "fez o Carlitos paradão e assustou uma criança quando piscou",
			"só piscou quando caiu moeda no chapéu", "segurou um espirro sem mexer um músculo",
		},
		busts: []string{
			"coçou o nariz e o povo pediu a moeda de volta", "pegou chuva e a tinta prata escorreu toda",
			"foi levado pela guarda municipal, que achou que era decoração",
		},
	},
	{
		name: "pescador", emoji: "🎣", title: "é pescador", area: areaCampo, min: 60, max: 140, every: 20 * time.Minute,
		craft:  craft{activity: "!pescar", faster: 50, fewerJunk: 30, grade: 2},
		hazard: hazard{chance: 0.05, harm: 25, wounds: []string{"caiu do barco e quase se afogou", "fisgou a própria mão com o anzol"}},
		tasks: []string{
			"pescou uma tilápia e contou pra todo mundo que era um tubarão", "passou a madrugada no pesqueiro e vendeu tudo na feira",
			"consertou a rede dos pescadores do porto", "levou turista pra pescar e o turista só pegou insolação",
			"pescou um lambari e tirou 30 fotos com ele", "vendeu minhoca na beira do rio",
			"contou história de pescador no bar e o peixe cresceu 2 metros na narração",
		},
	},
	{
		name: "fazendeiro", emoji: "🚜", title: "é fazendeiro", area: areaCampo, aliases: []string{"agro"},
		min: 220, max: 380, every: 36 * time.Minute,
		bust: 0.1, bustFine: 120, findChance: 0.12, finds: []string{"pote-de-feijao", "pao-de-queijo", "cafe", "picanha"},
		farmBeds: 2, farmFaster: 30,
		hazard: hazard{chance: 0.05, harm: 25, wounds: []string{"levou um coice da vaca", "teve o pé atropelado pelo trator"}},
		tasks: []string{
			"tirou leite da vaca às 4 da manhã", "colheu milho e vendeu pamonha na beira da estrada",
			"consertou a cerca que o boi derrubou", "foi pra cidade de caminhonete só pra comprar pão",
			"vacinou o gado inteiro e levou um coice", "botou no espantalho a camisa do time rival",
			"vendeu a safra de feijão e ainda deu uns sacos pra vizinhança",
		},
		busts: []string{
			"pegou geada e perdeu o cafezal inteiro", "deixou a vaca fugir pra fazenda do vizinho",
			"atolou a caminhonete depois de três dias de chuva",
		},
	},
	{
		name: "garimpeiro", emoji: "💎", title: "é garimpeiro", area: areaCampo, min: 80, max: 180, every: 20 * time.Minute,
		bust: 0.1, bustFine: 50, craft: craft{activity: "!minerar", luck: 1, fewerJunk: 40, grade: 2},
		hazard: hazard{chance: 0.1, harm: 35, wounds: []string{"ficou embaixo do barranco do garimpo que desmoronou", "afundou na lama até o pescoço"}},
		tasks: []string{
			"bateou cascalho o dia inteiro e achou umas pepitas", "jurou que uma pedra amarela era ouro e conseguiu vender",
			"achou um cristal e disse que ele tinha energia", "voltou do rio com o bolso cheio de areia brilhante",
			"trocou uma pepitinha por um prato feito e uma pinga", "achou uma ametista e fez colar pra vender na feira hippie",
		},
		busts: []string{
			"achou ouro de tolo e comemorou antes da hora", "viu a bateia descer rio abaixo",
			"perdeu as ferramentas quando o barranco desabou",
		},
	},
	{
		name: "minerador", emoji: "⛏️", title: "é minerador", area: areaCampo, min: 80, max: 160, every: 20 * time.Minute,
		craft:  craft{activity: "!minerar", faster: 30, fewerJunk: 30, grade: 1, noBreak: true},
		hazard: hazard{chance: 0.1, harm: 40, wounds: []string{"ficou preso num desabamento da mina", "respirou gás da mina e desmaiou lá embaixo"}},
		tasks: []string{
			"quebrou pedra o turno inteiro na mina", "tirou carvão suficiente pra um churrasco de 50 pessoas",
			"achou um veio de cobre e vendeu pro ferro-velho", "desceu na mina de capacete com lanterna e voltou preto de pó",
			"explodiu a parede certa pela primeira vez", "empurrou o carrinho de minério sem descarrilhar",
			"achou um osso e jurou que era de dinossauro",
		},
	},
	{
		name: "lenhador", emoji: "🪓", title: "é lenhador", area: areaCampo, min: 160, max: 280, every: 28 * time.Minute,
		bust: 0.15, bustFine: 90, doubleChance: 0.1,
		hazard: hazard{chance: 0.12, harm: 35, wounds: []string{"derrubou a árvore pro lado errado", "levou um coice da motosserra"}},
		tasks: []string{
			"derrubou uma árvore e gritou \"madeiraaa\"", "rachou lenha pro forno da pizzaria",
			"vendeu lenha pra fogueira da festa junina do bairro", "cortou o eucalipto que tava em cima da fiação",
			"fez um banquinho de toco e vendeu como móvel rústico", "serrou tronco no braço porque a motosserra não pegava",
		},
		busts: []string{
			"derrubou a árvore em cima do próprio carro", "viu a motosserra engasgar e não pegar mais",
			"foi multado porque a árvore era protegida",
		},
	},
	{
		name: "apicultor", emoji: "🐝", title: "é apicultor", area: areaCampo, min: 80, max: 160, every: 20 * time.Minute,
		bust: 0.1, guard: 0.2, craft: craft{activity: "!catar", luck: 1, fewerJunk: 25, grade: 2},
		hazard: hazard{chance: 0.12, harm: 20, wounds: []string{"levou 40 picadas de abelha", "abriu a colmeia sem a roupa de proteção"}},
		tasks: []string{
			"colheu mel e vendeu na feira em pote de maionese", "tirou um enxame do telhado da vizinha",
			"vendeu própolis dizendo que cura até dor de cotovelo", "fez vela de cera e vendeu como artesanato",
			"conversou com a abelha rainha e jura que ela entendeu", "vestiu a roupa de astronauta e foi trabalhar",
		},
		busts: []string{
			"levou 40 ferroadas e ficou com a cara parecendo uma bexiga", "viu o enxame inteiro fugir pro terreno do vizinho",
			"chegou tarde e as formigas já tinham acabado com o mel",
		},
	},
	{
		name: "peão de rodeio", emoji: "🐂", title: "é peão de rodeio", area: areaCampo, aliases: []string{"peão", "cowboy"},
		min: 240, max: 440, every: 28 * time.Minute,
		bust: 0.4, bustFine: 100, doubleChance: 0.15,
		hazard: hazard{chance: 0.18, harm: 40, wounds: []string{"foi jogado longe pelo touro, que ainda pisou em cima", "ficou pendurado no touro com a mão presa na corda"}},
		tasks: []string{
			"ficou 8 segundos em cima do touro e levou uma fivela do tamanho de um prato", "montou o touro mais bravo da festa e saiu andando",
			"fez a dança do peão com o chapéu na mão", "domou um cavalo xucro na frente da arquibancada",
			"laçou um bezerro de primeira", "desfilou na arena com a bandeira do patrocinador",
		},
		busts: []string{
			"voou do touro no primeiro segundo", "caiu de cara no barro na frente do crush",
			"levou uma chifrada no traseiro e foi parar no posto de saúde", "viu o touro se recusar a sair do brete",
		},
	},
	{
		name: "mergulhador", emoji: "🤿", title: "é mergulhador", area: areaCampo, min: 60, max: 130, every: 20 * time.Minute,
		craft:  craft{activity: "!pescar", luck: 1, fewerJunk: 30, grade: 1, noTool: true},
		hazard: hazard{chance: 0.08, harm: 40, wounds: []string{"subiu rápido demais e pegou a doença do mergulhador", "levou um susto de um tubarão e bateu a cabeça no coral"}},
		tasks: []string{
			"soldou o casco de um navio debaixo d'água", "achou a aliança que o noivo deixou cair do barco",
			"limpou o aquário gigante do shopping", "levou turista pra ver tartaruga e não apareceu nenhuma",
			"achou um celular no fundo da piscina do clube", "fez vistoria no píer e voltou com um polvo grudado no braço",
			"ensinou um turista a respirar no snorkel",
		},
	},
	{
		name: "caçador de tesouro", emoji: "🗺️", title: "é caçador de tesouro", area: areaCampo, aliases: []string{"caçador"},
		min: 130, max: 250, every: 28 * time.Minute,
		bust: 0.2, bustFine: 80, craft: craft{activity: "!vasculhar", luck: 2, fewerJunk: 40, grade: 1},
		hazard: hazard{chance: 0.08, harm: 35, wounds: []string{"caiu numa armadilha de pedra igual Indiana Jones", "levou uma flecha de uma armadilha antiga"}},
		tasks: []string{
			"seguiu um mapa do tesouro desenhado num guardanapo", "passou o detector de metal na praia e achou 3 tampinhas e um brinco",
			"cavou o quintal da avó atrás das moedas que o vô escondeu", "achou um baú no sótão cheio de foto antiga",
			"vendeu uma moeda velha pra um colecionador", "decifrou uma charada numa igreja do interior",
			"saiu correndo de uma pedra rolando numa ruína",
		},
		busts: []string{
			"cavou um buraco de 3 metros e achou o cano de esgoto", "descobriu que o mapa do tesouro era um cardápio de pizzaria",
			"caiu numa armadilha e foi parar no rio", "abriu o baú e só tinha um bilhete escrito \"chegou tarde\"",
		},
	},
	{
		name: "caçador de bússola", emoji: "🧭", title: "é caçador de bússola do relicário", area: areaCampo,
		aliases: []string{"caçador de bússola do relicário", "bussoleiro"}, min: 20, max: 60, every: 28 * time.Minute,
		bust: 0.45, bustFine: 120, salePercent: 40, betJinx: 0.15,
		jailFactor: 50, escapeBonus: -0.15, breakoutBonus: -0.15, reportBonus: -0.2, heistBonus: -0.1, trialEdge: -1,
		gatherFaster: 50, compassLure: 16, rotten: true, finds: []string{"bussola-do-relicario"}, findChance: 0.01, fixedFinds: true,
		tasks: []string{
			"passou o dia girando no meio da praça esperando a agulha apontar pra algum lugar",
			"vendeu um mapa \"original\" do relicário pra um turista", "seguiu a agulha de uma bússola até a casa da sogra",
			"jurou que a bússola tremeu perto do bueiro e cavou a calçada inteira",
			"contou pra roda do bar a lenda do Relicário perdido e ganhou umas cervejas",
		},
		busts: []string{
			"seguiu a bússola até o meio do rio e teve que voltar nadando", "descobriu que a bússola tava apontando pro ímã da geladeira",
			"invadiu um sítio atrás do relicário e levou multa do dono", "comprou uma bússola falsa no camelô e ainda pagou caro",
		},
	},
	{
		name: "jardineiro", emoji: "🌷", title: "é jardineiro", area: areaCampo, min: 70, max: 150, every: 16 * time.Minute,
		findChance: 0.2, finds: []string{"rosa", "buque", "sapo", "casca-de-banana"}, craft: craft{activity: "!catar", faster: 30},
		tasks: []string{
			"podou a cerca viva em formato de dinossauro", "cortou a grama do condomínio inteiro",
			"salvou a samambaia da avó que tava morrendo", "plantou um pé de manga que só vai dar fruta daqui 5 anos",
			"regou as plantas cantando pra elas crescerem", "arrancou a erva daninha que o síndico jurava ser flor",
			"montou um jardim vertical numa varanda de apartamento",
		},
	},
	{
		name: "agrônomo", emoji: "🌱", title: "é agrônomo", area: areaCampo, min: 130, max: 230, every: 28 * time.Minute,
		interest: 4, farmFaster: 20, farmGrade: 2,
		tasks: []string{
			"fez análise de solo e mandou botar mais calcário", "salvou a soja de uma praga com um laudo de 40 páginas",
			"calculou a safra e errou só por 3 toneladas", "foi na feira agropecuária e voltou com 12 bonés de brinde",
			"receitou adubo pra horta da tia", "pilotou drone em cima da lavoura de milho",
			"explicou rotação de cultura pra um fazendeiro que já fazia isso há 40 anos",
		},
	},
	{
		name: "criador de galinha", emoji: "🐔", title: "cria galinha", area: areaCampo, aliases: []string{"granjeiro"},
		min: 70, max: 150, every: 16 * time.Minute,
		doubleChance: 0.08, peak: peak{from: 4, to: 8, bonus: 80},
		tasks: []string{
			"acordou com o galo às 4 da manhã e recolheu os ovos", "vendeu ovo caipira na feira a preço de ouro",
			"correu atrás da galinha que fugiu do galinheiro", "deu nome pra todas as 40 galinhas",
			"chocou ovo na lâmpada e nasceram 12 pintinhos", "vendeu galinha caipira pro almoço de domingo",
			"trocou o galo que cantava às 3 da manhã por um que canta às 6",
		},
	},

	{
		name: "camelô", emoji: "🧺", title: "é camelô", area: areaComercio, min: 60, max: 120, every: 16 * time.Minute,
		bust: 0.1, bustFine: 50, salePercent: 80,
		tasks: []string{
			"vendeu capinha de celular que não serve em celular nenhum", "gritou \"é dez, é dez, é dez\" a manhã inteira na 25 de Março",
			"vendeu guarda-chuva a 30 reais no primeiro pingo de chuva", "empurrou um carregador que carrega 1% por hora",
			"vendeu fone \"original\" que veio sem a caixinha", "montou a banca na calçada antes do sol nascer",
			"vendeu água gelada no engarrafamento", "fez o \"leva três, paga dois\" e ainda lucrou",
		},
		busts: []string{
			"correu do rapa e largou a mercadoria toda na calçada", "teve a banca levada pela fiscalização da prefeitura",
			"vendeu fiado pro vizinho de banca", "ficou o dia todo debaixo de chuva sem vender nem um isqueiro",
		},
	},
	{
		name: "feirante", emoji: "🍅", title: "é feirante", area: areaComercio, min: 120, max: 200, every: 20 * time.Minute,
		peak:       peak{from: 6, to: 11, bonus: 60},
		finds:      []string{"pastel", "pao-de-queijo", "casca-de-banana", "coxinha"},
		findChance: 0.15,
		tasks: []string{
			"gritou \"olha a banana, freguesa\" até ficar rouco", "deu um pedaço de melancia pra provar e vendeu três",
			"fez a xepa e vendeu tudo pela metade do preço no fim da feira", "chamou toda cliente de \"minha linda\"",
			"montou uma pirâmide de laranja perfeita", "vendeu tomate amassado dizendo que era pra molho",
			"deu um pastel de brinde pro cliente mais chato", "pesou o quilo de uva com o dedão na balança",
		},
	},
	{
		name: "comerciante", emoji: "🏪", title: "é comerciante", area: areaComercio, min: 150, max: 260, every: 28 * time.Minute,
		bust: 0.1, bustFine: 120, dailyBonus: 50,
		tasks: []string{
			"pendurou a placa \"fiado só amanhã\" e salvou o caixa", "abriu a loja às 7h e fechou às 22h",
			"vendeu o último pacote de arroz pro vizinho desesperado", "fez promoção de queima de estoque que nunca acaba",
			"cobrou a caderneta do fiado de todo mundo do bairro", "reajustou o preço da Coca dizendo que foi o fornecedor",
			"vendeu o pão do dia anterior como \"pão da casa\"",
		},
		busts: []string{
			"levou multa da vigilância sanitária por causa do gato no balcão", "teve a maquininha de cartão sem sinal o dia inteiro",
			"descobriu que o funcionário estava dando desconto pros amigos", "pagou o aluguel da loja e não sobrou nada",
		},
	},
	{
		name: "sacoleiro", emoji: "🧳", title: "é sacoleiro", area: areaComercio, aliases: []string{"muambeiro"}, min: 160, max: 300,
		every: 28 * time.Minute,
		bust:  0.2, bustFine: 150, discount: 20,
		tasks: []string{
			"atravessou a Ponte da Amizade com quinze sacolas", "vendeu perfume \"importado\" que cheira a álcool",
			"revendeu os eletrônicos do Paraguai no grupo da família", "pegou o ônibus de excursão das 3h da manhã",
			"escondeu dez celulares dentro do travesseiro", "vendeu tênis de marca com o logo meio torto",
			"encheu a mala de videogame e voltou sorrindo",
		},
		busts: []string{
			"foi parado pela Receita Federal e perdeu a muamba toda", "teve a sacola rasgada no meio da rodoviária",
			"descobriu que o fornecedor mandou caixa cheia de tijolo", "perdeu o ônibus de volta com a mercadoria dentro",
		},
	},
	{
		name: "dono de brechó", emoji: "👗", title: "tem um brechó", area: areaComercio, min: 90, max: 170, every: 20 * time.Minute,
		finds:      []string{"vhs", "cd-pirata", "oculos-juliet", "nokia", "tamagotchi", "boneca"},
		findChance: 0.25,
		tasks: []string{
			"vendeu a jaqueta jeans dos anos 90 como peça \"vintage\"", "cobrou 80 reais numa camisa de time com furo",
			"chamou roupa velha de \"peça de garimpo\"", "garimpou um vestido de festa no bazar da igreja",
			"vendeu o casaco da tia falecida pra uma blogueira", "postou look do dia com a arara da loja",
			"tirou um cheiro de naftalina de uma saia com perfume",
		},
	},
	{
		name: "dono de ferro-velho", emoji: "🔩", title: "tem um ferro-velho", area: areaComercio, min: 70, max: 130, every: 20 * time.Minute,
		salePercent: 75,
		craft:       craft{activity: "!vasculhar", luck: 1, fewerJunk: 25},
		hazard:      hazard{chance: 0.07, harm: 25, wounds: []string{"cortou a mão numa lata enferrujada", "ficou embaixo de uma pilha de sucata que desabou"}},
		tasks: []string{
			"comprou geladeira velha a 20 reais e vendeu o motor a 200", "desmontou um fusca inteiro com uma chave de fenda",
			"pesou uma pilha de latinha na balança viciada", "achou peça de carro rara no meio da sucata",
			"vendeu fio de cobre de procedência duvidosa", "comprou um fogão sem as bocas de um cara na carroça",
			"soltou o cachorro em quem tentou pular o muro",
		},
	},
	{
		name: "leiloeiro", emoji: "🔨", title: "é leiloeiro", area: areaComercio, min: 100, max: 200, every: 28 * time.Minute,
		salePercent: 85, doubleChance: 0.1,
		tasks: []string{
			"gritou \"dou-lhe uma, dou-lhe duas\" em velocidade recorde", "leiloou um sofá rasgado como peça de colecionador",
			"fez dois ricaços brigarem por um quadro feio", "vendeu um carro apreendido pelo triplo do valor",
			"bateu o martelo e quebrou a mesa", "fingiu que o lance da vovó era de outra pessoa",
			"leiloou a coleção de tampinhas de um falecido",
		},
	},
	{
		name: "caixa de supermercado", emoji: "🧾", title: "é caixa de supermercado", area: areaComercio, aliases: []string{"caixa"},
		min: 80, max: 140, every: 20 * time.Minute,
		discount: 15,
		peak:     peak{from: 17, to: 20, bonus: 30},
		tasks: []string{
			"perguntou \"CPF na nota?\" trezentas vezes", "chamou o fiscal porque o código de barras não passava",
			"passou as compras do mês de uma família de doze pessoas", "pediu pra cliente trocar a nota de 100",
			"pesou a banana errada e ninguém percebeu", "aguentou cliente contando moedinha na fila de dez itens",
			"perguntou \"débito ou crédito?\" pela milésima vez",
		},
	},
	{
		name: "vendedor de loja", emoji: "🛍️", title: "é vendedor de loja", area: areaComercio, aliases: []string{"vendedor"},
		min: 100, max: 220, every: 20 * time.Minute,
		bust: 0.15, doubleChance: 0.15,
		tasks: []string{
			"disse \"ficou ótimo em você\" pra todo cliente", "empurrou o cartão da loja pra uma senhora desavisada",
			"buscou o número 38 no estoque e voltou com o 40", "vendeu um tênis e mais três pares de meia",
			"bateu a meta do mês no último dia", "convenceu o cara a levar a camisa em quatro cores",
			"falou \"essa é a última peça\" com a pilha cheia atrás",
		},
		busts: []string{
			"atendeu cliente que só estava dando uma olhadinha", "levou a venda roubada pelo colega da comissão",
			"passou o turno dobrando a mesma pilha de camiseta", "vendeu uma calça e a cliente devolveu no mesmo dia",
		},
	},
	{
		name: "dono de bar", emoji: "🍺", title: "tem um bar", area: areaComercio, min: 120, max: 210, every: 20 * time.Minute,
		peak:       peak{from: 19, to: 2, bonus: 60},
		finds:      []string{"latao", "podrao", "cadeira", "controle-remoto"},
		findChance: 0.12,
		tasks: []string{
			"anotou mais um fiado no caderninho", "botou o jogo do Flamengo na TV de tubo",
			"vendeu porção de calabresa que estava ali desde ontem", "separou uma briga por causa de dominó",
			"ouviu a mesma história triste do bêbado pela quinta vez", "fechou a porta de enrolar com a turma ainda lá dentro",
			"vendeu o último latão gelado pelo dobro do preço", "fez karaokê de sertanejo até o vizinho chamar a polícia",
		},
	},
	{
		name: "revendedor de cosméticos", emoji: "💄", title: "revende cosméticos", area: areaComercio, aliases: []string{"revendedora"},
		min: 120, max: 240, every: 20 * time.Minute,
		bust: 0.25, doubleChance: 0.12,
		finds:      []string{"perfume", "olho-grego", "rosa"},
		findChance: 0.1,
		tasks: []string{
			"passou o catálogo no grupo das mães da escola", "vendeu batom pra todas as tias do almoço de domingo",
			"entregou encomenda no portão de casa em casa", "fez a amostra grátis render pra vinte clientes",
			"vendeu um perfume dizendo que é igualzinho ao importado", "parcelou um hidratante em seis vezes",
			"convenceu a sogra a fazer a linha completa de pele",
		},
		busts: []string{
			"vendeu fiado pra colega de trabalho que sumiu do mapa", "teve a encomenda extraviada pelos Correios",
			"descobriu que a cliente pagou com um Pix agendado que nunca caiu", "deixou o batom derreter no carro",
		},
	},
	{
		name: "penhorista", emoji: "💍", title: "é penhorista", area: areaComercio, min: 100, max: 180, every: 28 * time.Minute,
		salePercent: 80, interest: 4,
		tasks: []string{
			"avaliou a aliança do divórcio de alguém pela metade do valor", "pegou uma televisão em troca de um empréstimo",
			"mordeu uma moeda pra ver se era ouro mesmo", "cobrou juros sobre os juros sobre os juros",
			"guardou um videogame que nunca vai ser resgatado", "olhou um relógio com a lupinha e fez cara de desprezo",
			"vendeu a guitarra que o dono esqueceu de buscar",
		},
	},
	{
		name: "policial", emoji: "👮", title: "é policial", area: areaSeguranca, aliases: []string{"pm"}, min: 120, max: 220,
		every: 28 * time.Minute,
		guard: 0.15, jailTime: baseJail, arrestEvery: 30 * time.Minute, jailFactor: -25, escapeBonus: -0.1,
		breakoutBonus: 0.2,
		hazard:        hazard{chance: 0.1, harm: 40, wounds: []string{"levou um tiro de raspão numa troca de tiro", "correu atrás de um suspeito e caiu do muro"}},
		tasks: []string{
			"fez uma blitz e só achou carro com IPVA atrasado", "comeu coxinha na padaria de graça",
			"correu atrás de um ladrão de celular até perder o fôlego", "parou a viatura na calçada pra tomar um café",
			"atendeu ocorrência de briga de vizinho por causa de som alto", "fez ronda no bairro com a sirene ligada só por diversão",
			"registrou o boletim de ocorrência de um pato desaparecido",
		},
	},
	{
		name: "segurança", emoji: "🕶️", title: "é segurança", area: areaSeguranca, min: 80, max: 140, every: 20 * time.Minute,
		guard:         0.25,
		heistBonus:    0.1,
		breakoutBonus: 0.15,
		hazard:        hazard{chance: 0.08, harm: 30, wounds: []string{"separou uma briga de bar e levou uma garrafada", "levou uma cadeirada de um bêbado"}},
		tasks: []string{
			"seguiu um cliente suspeito pelo mercado inteiro", "ficou de braços cruzados na porta da balada a noite toda",
			"barrou um cara de chinelo na festa", "fingiu falar no rádio sem ninguém do outro lado",
			"revistou bolsa na entrada do show", "usou óculos escuros dentro do shopping às 22h",
			"pegou um adolescente tentando sair com chocolate no bolso",
		},
	},
	{
		name: "delegado", emoji: "🚨", title: "é delegado", area: areaSeguranca, min: 160, max: 280, every: 36 * time.Minute,
		guard: 0.1, legalDiscount: 40, jailTime: baseJail, arrestEvery: time.Hour, abuses: []abuseKind{arrestAbuse}, jailFactor: -25, trialEdge: 1, escapeBonus: -0.1, releaseChance: 0.3,
		breakoutBonus: 0.15,
		tasks: []string{
			"deu entrevista no jornal das seis com cara de sério", "assinou uma pilha de inquérito sem ler",
			"mandou o escrivão digitar o depoimento em caixa alta", "interrogou o suspeito com a luz na cara",
			"montou um quadro de investigação com barbante vermelho", "fechou o caso do roubo de galinha do bairro",
			"pediu pizza pra delegacia inteira no plantão",
		},
	},
	{
		name: "detetive particular", emoji: "🕵️", title: "é detetive particular", area: areaSeguranca, aliases: []string{"detetive"},
		min: 80, max: 150, every: 20 * time.Minute,
		theftEvery: 15 * time.Minute, theftBonus: 0.15, reportBonus: 0.35,
		finds:         []string{"carteira-perdida", "jornal", "anel"},
		findChance:    0.1,
		breakoutBonus: 0.1,
		tasks: []string{
			"seguiu um marido suspeito até o motel", "tirou foto escondido atrás de um jornal furado",
			"descobriu que o gato da cliente estava na casa do vizinho", "passou a madrugada na campana comendo pastel frio",
			"vasculhou o lixo de um empresário", "confirmou a traição pelo story que o cara esqueceu de apagar",
			"usou um bigode falso pra entrar num casamento",
		},
	},
	{
		name: "bombeiro", emoji: "🚒", title: "é bombeiro", area: areaSeguranca, min: 310, max: 455, every: 40 * time.Minute,
		finds:         []string{"gato-preto", "papagaio", "pombo"},
		findChance:    0.08,
		breakoutBonus: 0.15,
		hazard:        hazard{chance: 0.12, harm: 40, wounds: []string{"entrou num prédio pegando fogo e saiu com o braço queimado", "caiu da escada magirus resgatando um gato"}},
		tasks: []string{
			"tirou um gato de cima do poste pela terceira vez na semana", "apagou um incêndio começado por uma panela de pressão",
			"tirou a mão de uma criança presa no pote de biscoito", "fez plantão de 24 horas e ainda lavou o caminhão",
			"tirou a aliança presa no dedo inchado de um noivo", "resgatou uma senhora do elevador parado",
			"pegou uma cobra dentro do sanitário de alguém", "posou pro calendário anual da corporação",
		},
	},
	{
		name: "vigia noturno", emoji: "🔦", title: "é vigia noturno", area: areaSeguranca, aliases: []string{"vigia"}, min: 100,
		max: 160, every: 20 * time.Minute,
		peak:        peak{from: 22, to: 6, bonus: 80},
		reportBonus: 0.15,
		guard:       0.1,
		heistBonus:  0.08,
		hazard:      hazard{chance: 0.05, harm: 20, wounds: []string{"foi atrás de um barulho no escuro e caiu num buraco", "levou uma mordida do cachorro da firma"}},
		tasks: []string{
			"passou a noite acordado na base do café de garrafa térmica", "assustou um gato achando que era ladrão",
			"fez a ronda com a lanterna quase sem pilha", "ouviu o jogo inteiro no radinho de pilha",
			"cochilou na cadeira de plástico e ninguém viu", "bateu o ponto em todas as rondas do condomínio",
			"espantou a molecada que pulava o muro da obra",
		},
	},
	{
		name: "guarda de trânsito", emoji: "🚦", title: "é guarda de trânsito", area: areaSeguranca, aliases: []string{"marronzinho"},
		min: 100, max: 200, every: 20 * time.Minute,
		bust: 0.2, doubleChance: 0.15,
		peak:   peak{from: 17, to: 20, bonus: 40},
		hazard: hazard{chance: 0.05, harm: 30, wounds: []string{"foi atropelado por um motorista sem CNH", "levou um retrovisor no braço no meio do cruzamento"}},
		tasks: []string{
			"multou um carro parado em fila dupla na porta da escola", "apitou no cruzamento até perder a voz",
			"guinchou o carro de um influencer", "fingiu que não viu o carro do chefe na vaga de idoso",
			"organizou o trânsito no dia que o semáforo apagou", "multou um motoboy na contramão",
			"escreveu a multa devagarinho enquanto o motorista implorava",
		},
		busts: []string{
			"foi xingado de tudo quanto é nome no cruzamento", "teve o talão de multa molhado pela chuva",
			"descobriu que a placa era de carro oficial e teve que cancelar a multa", "foi atropelado por um patinete elétrico",
		},
	},
	{
		name: "juiz", emoji: "⚖️", title: "é juiz", area: areaSeguranca, aliases: []string{"juíza"}, min: 225, max: 315,
		every:         40 * time.Minute,
		legalDiscount: 50, dailyBonus: 30, jailTime: baseJail, arrestEvery: 2 * time.Hour, abuses: []abuseKind{arrestAbuse}, jailFactor: -50, trialEdge: 1, trialEvery: 25 * time.Minute, escapeBonus: -0.15, releaseChance: 0.5,
		tasks: []string{
			"bateu o martelo mais forte que o necessário", "deu uma sentença de quarenta páginas cheia de latim",
			"adiou a audiência porque o ar-condicionado quebrou", "recebeu auxílio-paletó e auxílio-gravata no mesmo mês",
			"mandou o advogado tirar o boné na audiência", "tirou férias de sessenta dias",
			"julgou briga de vizinho por causa de uma mangueira",
		},
	},
	{
		name: "advogado", emoji: "💼", title: "é advogado", area: areaSeguranca, aliases: []string{"advogada"}, min: 150, max: 270,
		every: 28 * time.Minute,
		bust:  0.2, legalDiscount: 75, trialDiscount: 50, jailFactor: -50, trialEdge: 2, trialEvery: 15 * time.Minute, releaseChance: 0.4,
		tasks: []string{
			"cobrou consulta pra responder um áudio no WhatsApp", "achou uma brecha na lei e salvou o cliente",
			"falou \"data venia\" sem nenhum motivo", "protocolou um processo um minuto antes do prazo",
			"tirou o tio do SPC com uma petição", "ganhou uma causa trabalhista contra o patrão do primo",
			"respondeu tudo com \"depende\" e cobrou caro por isso",
		},
		busts: []string{
			"perdeu o prazo do recurso por causa do sistema fora do ar", "levou o cliente pra audiência e ele confessou tudo",
			"perdeu a causa e o cliente não pagou os honorários", "esqueceu a OAB em casa e não pôde entrar no fórum",
		},
	},
	{
		name: "promotor", emoji: "📜", title: "é promotor", area: areaSeguranca, aliases: []string{"promotora"}, min: 140, max: 220,
		every:         28 * time.Minute,
		legalDiscount: 35, doubleChance: 0.1, trialBonus: 50, jailFactor: -25, trialEdge: 2, trialEvery: 15 * time.Minute,
		reportBonus: 0.25,
		tasks: []string{
			"fez um discurso emocionante pro júri", "apontou o dedo pro réu e gritou \"culpado\"",
			"montou uma apresentação de PowerPoint com 200 slides de denúncia", "pediu a pena máxima pra um ladrão de galinha",
			"leu o processo inteiro na madrugada antes do júri", "encontrou a prova escondida no anexo 47",
			"deu entrevista na porta do fórum com cara de quem ganhou",
		},
	},
	{
		name: "salva-vidas", emoji: "🛟", title: "é salva-vidas", area: areaSeguranca, min: 80, max: 140, every: 20 * time.Minute,
		craft:      craft{activity: "!pescar", faster: 25, noBreak: true},
		finds:      []string{"bota-velha", "chinelo", "oculos-juliet", "perola"},
		findChance: 0.1,
		hazard:     hazard{chance: 0.08, harm: 25, wounds: []string{"foi salvar um banhista e levou um caldo", "pegou uma água-viva no peito"}},
		tasks: []string{
			"apitou pra galera sair da área de corrente", "puxou um turista que achou que sabia nadar",
			"correu na areia em câmera lenta pra impressionar", "fincou a bandeira vermelha e ninguém respeitou",
			"tirou uma criança do mar e devolveu pra mãe errada", "passou protetor solar só no nariz",
			"resgatou uma boia de unicórnio indo pra alto-mar",
		},
	},
	{
		name: "porteiro", emoji: "🚪", title: "é porteiro", area: areaSeguranca, min: 70, max: 130, every: 20 * time.Minute,
		guard:       0.2,
		reportBonus: 0.2,
		finds:       []string{"caixa-misteriosa", "jornal", "boleto", "cabo-usb"},
		findChance:  0.12,
		tasks: []string{
			"segurou a encomenda do 302 por três dias", "contou a fofoca do prédio inteiro pro entregador",
			"interfonou pra avisar que a pizza chegou fria", "abriu o portão sem perguntar quem era",
			"assistiu novela na televisãozinha da guarita", "deu bronca no morador que deixou o carro torto",
			"anotou quem saiu tarde e quem voltou de madrugada",
		},
	},

	{
		name: "bandido", emoji: "🦹", title: "é bandido", area: areaCrime, aliases: []string{"vagabundo"}, min: 40, max: 150, every: 8 * time.Minute,
		bust: 0.35, robEvery: 5 * time.Minute, robBonus: 0.25, jailFactor: 50, trialEdge: -1, escapeBonus: 0.15, kidnapBonus: 0.1,
		heistBonus: 0.1, noSnitch: true, killChance: 0.35,
		hazard: hazard{chance: 0.1, harm: 40, wounds: []string{"trocou tiro com a polícia e levou um de raspão", "pulou o muro errado e caiu no quintal de um rottweiler"}},
		tasks: []string{
			"vendeu celular de procedência duvidosa na feira", "passou a perna num turista", "vendeu um Rolex de camelô como original",
			"aplicou o golpe do pix errado", "revendeu ingresso falso de show", "vendeu gato de TV a cabo pra rua inteira",
		},
		busts: []string{
			"foi enquadrado pela polícia e teve que largar tudo", "tentou aplicar golpe num golpista e saiu no prejuízo",
			"passou o dia escondido no mato esperando a viatura ir embora", "foi reconhecido pela vítima na fila da lotérica",
		},
	},
	{
		name: "batedor de carteira", emoji: "🤏", title: "é batedor de carteira", area: areaCrime, aliases: []string{"trombadinha", "punguista"},
		min: 30, max: 110, every: 8 * time.Minute,
		bust: 0.3, theftEvery: 10 * time.Minute, theftBonus: 0.1, jailFactor: 25, trialEdge: -1,
		finds: []string{"carteira-perdida", "nokia", "boleto", "raspadinha"}, findChance: 0.1, escapeBonus: 0.1,
		tasks: []string{
			"aliviou a carteira de um tiozinho no 25 de Março", "trabalhou o vagão lotado da Linha Vermelha às 18h",
			"tirou um celular do bolso de trás de um turista distraído", "aproveitou o empurra-empurra do bloquinho de carnaval",
			"passou a mão leve na saída do show de pagode", "fingiu que tropeçou e levou a carteira junto com o pedido de desculpa",
		},
		busts: []string{
			"abriu a carteira e só tinha cupom fiscal e foto 3x4", "meteu a mão no bolso de um policial à paisana",
			"foi pego no flagra e saiu correndo de chinelo", "levou a carteira de outro batedor e teve que devolver",
		},
	},
	{
		name: "golpista", emoji: "🎭", title: "é golpista", area: areaCrime, aliases: []string{"171", "estelionatário"},
		min: 80, max: 220, every: 20 * time.Minute,
		bust: 0.3, bustFine: 100, legalDiscount: 50, robBonus: 0.1, jailFactor: 50, trialEdge: 1, escapeBonus: 0.15, ransomBonus: 25,
		reportBonus: -0.25,
		heistBonus:  0.05,
		tasks: []string{
			"ligou fingindo ser o gerente do banco e pediu o código do SMS", "vendeu curso de como ficar rico vendendo curso",
			"aplicou o golpe do falso sobrinho no zap da tia", "vendeu um terreno na Lua pra um coach",
			"fez rifa de uma moto que nunca existiu", "anunciou apartamento na praia que era um terreno baldio",
			"montou pirâmide financeira com nome de startup",
		},
		busts: []string{
			"tentou aplicar golpe na avó, que já tinha visto no Fantástico", "caiu no golpe de outro golpista mais esperto",
			"virou reportagem do Cidade Alerta", "mandou o áudio do golpe no grupo da família por engano",
		},
	},
	{
		name: "hacker", emoji: "👨‍💻", title: "é hacker", area: areaCrime, min: 180, max: 450, every: 28 * time.Minute,
		bust: 0.3, bustFine: 150, doubleChance: 0.12, jailFactor: 25, trialEdge: -1, escapeBonus: 0.15,
		heistBonus: 0.12,
		tasks: []string{
			"invadiu o Wi-Fi do vizinho com a senha 12345678", "derrubou o site da prefeitura apertando F5 muito rápido",
			"vendeu lista de senhas que eram todas \"senha123\"", "minerou bitcoin no computador da escola",
			"clonou o perfil do Instagram de um influencer de academia", "hackeou a conta do Free Fire de um moleque de 12 anos",
			"digitou muito rápido numa tela verde e saiu dizendo \"estou dentro\"",
		},
		busts: []string{
			"esqueceu a VPN desligada e a polícia bateu na porta", "caiu num phishing enquanto preparava um phishing",
			"teve o PC formatado pela mãe pra \"tirar o vírus\"", "deixou o próprio CPF no código e foi achado em dez minutos",
		},
	},
	{
		name: "contrabandista", emoji: "📦", title: "é contrabandista", area: areaCrime,
		min: 150, max: 330, every: 28 * time.Minute,
		bust: 0.25, bustFine: 120, discount: 15, jailFactor: 50, trialEdge: -1, escapeBonus: 0.1, ransomBonus: 50,
		tasks: []string{
			"trouxe trinta videogames do Paraguai escondidos no estepe", "atravessou a Ponte da Amizade com doze perfumes na cueca",
			"revendeu whisky de Ciudad del Este no churrasco da firma", "encheu o porta-malas de pneu importado sem nota",
			"fez a rota do sacoleiro com três malas e uma mochila", "vendeu iPhone \"lacrado\" que veio dentro de uma caixa de sapato",
		},
		busts: []string{
			"foi parado pela Receita na fronteira com o carro rebaixado de tanta muamba", "teve a muamba apreendida e só sobrou o chaveiro",
			"descobriu que os perfumes importados eram água com corante", "o ônibus de compras quebrou no meio da BR",
		},
	},
	{
		name: "assaltante de banco", emoji: "🏦", title: "assalta banco", area: areaCrime, aliases: []string{"assaltante"},
		min: 300, max: 750, every: 40 * time.Minute,
		bust: 0.45, bustFine: 225, robBonus: 0.35, jailFactor: 100, trialEdge: -2, escapeBonus: 0.2, kidnapBonus: 0.15,
		heistBonus: 0.2, killChance: 0.4, abuses: []abuseKind{robAbuse},
		hazard: hazard{chance: 0.1, harm: 45, wounds: []string{"levou chumbo do segurança do banco", "tava perto demais quando a bomba do cofre explodiu antes da hora"}},
		tasks: []string{
			"cavou um túnel até o cofre igual no filme", "fez o banco inteiro deitar no chão e levou até as canetas presas na corrente",
			"rendeu o carro-forte com uma banana dentro da blusa", "planejou o assalto perfeito com maquete de massinha",
			"esvaziou o caixa eletrônico e ainda tirou o extrato", "saiu do banco de máscara do Dalí cantando Bella Ciao",
		},
		busts: []string{
			"ficou preso na porta giratória com o bolso cheio de moeda", "escolheu bem o dia de greve dos bancários",
			"o túnel saiu no meio da delegacia", "esqueceu de tirar o crachá do trabalho antes de entrar",
		},
	},
	{
		name: "falsificador", emoji: "🖨️", title: "é falsificador", area: areaCrime, min: 120, max: 280, every: 20 * time.Minute,
		bust: 0.25, bustFine: 100, jailFactor: 50, trialEdge: -1,
		finds: []string{"nota-de-tres", "boleto", "jornal", "cd-pirata"}, findChance: 0.12, escapeBonus: 0.1,
		tasks: []string{
			"imprimiu diploma de medicina numa impressora jato de tinta", "fez carteirinha de estudante pra um tiozinho de 50 anos",
			"falsificou atestado médico pro time de várzea inteiro", "fez a assinatura do chefe melhor que o próprio chefe",
			"estampou bolsa de grife que escreveram \"Gucy\"", "carimbou documento com um carimbo feito de batata",
		},
		busts: []string{
			"imprimiu a nota com a cara do Faustão", "a impressora acabou a tinta no meio do lote",
			"escreveu \"República Federativa do Brazil\" com Z", "o cliente pagou com uma nota falsa que ele mesmo tinha feito",
		},
	},
	{
		name: "bicheiro", emoji: "🎰", title: "é bicheiro", area: areaCrime, aliases: []string{"jogo do bicho"}, min: 80, max: 200,
		every: 20 * time.Minute,
		bust:  0.15, cashback: 20, jailFactor: 25, trialEdge: -1, animalLuck: 25,
		tasks: []string{
			"anotou a fezinha da padaria inteira num caderninho", "explicou pra tia que sonhar com cobra é dar na cabeça",
			"pagou o prêmio do avestruz e ficou com o troco", "abriu ponto de bicho do lado da igreja",
			"tirou o resultado do bicho antes do almoço", "recebeu aposta no milhar da placa do carro do vizinho",
		},
		busts: []string{
			"deu jacaré e ele tinha prometido que não dava", "a polícia passou e ele teve que engolir o caderninho",
			"o gato comeu as anotações do dia", "deu a milhar que a rua inteira jogou",
		},
	},
	{
		name: "agiota", emoji: "💰", title: "é agiota", area: areaCrime, min: 120, max: 280, every: 28 * time.Minute,
		bust: 0.2, interest: 3, jailFactor: 50, trialEdge: -1,
		lendRate: 3, lendLateRate: 5, garnish: 30, collectBonus: 0.15, collectEvery: 20 * time.Minute, collectVotes: 2,
		tasks: []string{
			"emprestou cinquenta e cobrou duzentos na semana seguinte", "cobrou juros sobre os juros dos juros",
			"aceitou a TV do cliente como pagamento da parcela", "bateu na porta no dia do pagamento com a calculadora na mão",
			"refinanciou a dívida de um cara que já devia pra outro agiota", "mandou a cobrança pelo grupo da família do devedor",
		},
		busts: []string{
			"o devedor se mudou de estado e trocou de nome", "emprestou pro próprio cunhado e nunca mais viu o dinheiro",
			"o cliente pagou com cheque sem fundo", "ouviu \"semana que vem eu te pago\" pela décima vez",
		},
	},
	{
		name: "cambista", emoji: "🎟️", title: "é cambista", area: areaCrime, min: 80, max: 190, every: 16 * time.Minute,
		bust: 0.2, bustFine: 60, peak: peak{from: 18, to: 23, bonus: 60}, jailFactor: 25, trialEdge: -1, ransomBonus: 30,
		tasks: []string{
			"vendeu ingresso da final pelo triplo na porta do Maracanã", "gritou \"compro e vendo\" até ficar rouco",
			"revendeu meia-entrada pra um senhor de bengala", "vendeu ingresso do show de sertanejo pelo preço de um rim",
			"comprou cem ingressos no site antes de travar", "vendeu lugar na fila do show da madrugada",
		},
		busts: []string{
			"ficou com quarenta ingressos de um show que foi cancelado", "o jogo foi de portões fechados",
			"foi enquadrado pelo Procon na porta do estádio", "vendeu um ingresso pro próprio fiscal do evento",
		},
	},
	{
		name: "piloto de fuga", emoji: "🏎️", title: "é piloto de fuga", area: areaCrime, aliases: []string{"motorista de fuga"},
		min: 90, max: 230, every: 20 * time.Minute,
		bust: 0.25, bustFine: 80, guard: 0.15, robBonus: 0.2, jailFactor: -25, trialEdge: -1, escapeBonus: 0.4,
		kidnapBonus: 0.2, kidnapEvery: 30 * time.Minute,
		heistBonus:    0.12,
		breakoutBonus: 0.25,
		hazard:        hazard{chance: 0.12, harm: 45, wounds: []string{"capotou o carro na perseguição", "bateu num poste fugindo da viatura"}},
		tasks: []string{
			"fez um cavalo de pau na rotatória e despistou a viatura", "fugiu pela contramão cantando Velozes e Furiosos",
			"deu fuga num Gol bolinha com a lataria amarrada de arame", "cortou caminho por dentro do shopping",
			"esperou o comparsa na porta com o motor ligado e o pisca-alerta aceso", "subiu o morro de ré pra despistar",
		},
		busts: []string{
			"parou no sinal vermelho por costume", "o carro de fuga não pegou de jeito nenhum",
			"acabou a gasolina no meio da perseguição", "ficou preso no engarrafamento da Marginal com a polícia logo atrás",
		},
	},
	{
		name: "sequestrador", emoji: "🚐", title: "é sequestrador", area: areaCrime, aliases: []string{"sequestradora", "raptor"},
		min: 70, max: 200, every: 20 * time.Minute,
		bust: 0.3, bustFine: 90, jailFactor: 75, trialEdge: -2, escapeBonus: 0.1,
		kidnapBonus: 0.25, kidnapEvery: 25 * time.Minute, ransomBonus: 25, killChance: 0.3, abuses: []abuseKind{kidnapAbuse},
		breakoutBonus: 0.2,
		tasks: []string{
			"pintou \"FRETES E MUDANÇAS\" na van pra disfarçar", "recortou letra de revista pra montar o bilhete de resgate",
			"alugou um cativeiro com garagem e área de lazer", "treinou voz grossa no espelho pra ligar pra família",
			"vendeu o kit sequestro completo pra um iniciante: saco, corda e bala de goma", "fez curso de nó de marinheiro no YouTube",
		},
		busts: []string{
			"sequestrou o cara errado e a família agradeceu", "o refém falou tanto que ele pagou pra devolver",
			"ligou pedindo resgate e caiu na caixa postal", "deixou o próprio CPF no bilhete de resgate",
		},
	},
	{
		name: "matador de aluguel", emoji: "🔪", title: "é matador de aluguel", area: areaCrime, aliases: []string{"matador", "pistoleiro", "assassino"},
		min: 150, max: 350, every: 36 * time.Minute,
		bust: 0.3, bustFine: 150, jailFactor: 100, trialEdge: -2, escapeBonus: 0.15, noSnitch: true,
		killChance: 0.6,
		hazard:     hazard{chance: 0.08, harm: 40, wounds: []string{"descobriu tarde demais que o alvo também tava armado", "se cortou limpando a faca"}},
		tasks: []string{
			"deu um susto num devedor que pagou tudo na hora", "fez plantão de três dias numa kombi com binóculo",
			"limpou a arma ouvindo pagode na varanda", "apareceu no aniversário do alvo só pra encarar ele no parabéns",
			"cobrou uma dívida antiga só com o olhar", "vendeu silenciador feito de garrafa PET pra um iniciante",
		},
		busts: []string{
			"errou o endereço e assustou uma senhora que chamou a polícia", "o alvo era primo dele e a família ficou sabendo",
			"espirrou na hora H e entregou o esconderijo", "o contratante pagou com cheque sem fundo",
		},
	},
	{
		name: "ladrão de galinha", emoji: "🐔", title: "é ladrão de galinha", area: areaCrime, aliases: []string{"ladrão"}, min: 20, max: 80,
		every: 8 * time.Minute,
		bust:  0.3, theftBonus: 0.08, robBonus: 0.1, jailFactor: 25, trialEdge: -1, escapeBonus: 0.1,
		hazard: hazard{chance: 0.1, harm: 15, wounds: []string{"levou uma mordida na canela do cachorro do sítio", "levou uma bicada do galo de briga"}},
		tasks: []string{
			"pulou o muro do sítio e saiu com duas galinhas debaixo do braço", "roubou o varal da vizinha com as roupas e tudo",
			"levou a manga do pé do vizinho", "saiu do bar com o saleiro e o paliteiro", "roubou o sinal de Wi-Fi da padaria",
			"pegou um botijão vazio e revendeu como cheio",
		},
		busts: []string{
			"foi bicado pela galinha até largar ela", "o cachorro do sítio correu atrás dele até a estrada",
			"o galo cantou e acordou a fazenda inteira", "ficou preso na cerca com a calça rasgada",
		},
	},
	{
		name: "vendedor de dvd pirata", emoji: "📀", title: "vende dvd pirata", area: areaCrime, aliases: []string{"dvd pirata", "pirateiro"},
		min: 50, max: 140, every: 16 * time.Minute,
		bust: 0.15, bustFine: 50, salePercent: 75, jailFactor: 25, trialEdge: -1,
		finds: []string{"cd-pirata", "vhs", "controle-remoto"}, findChance: 0.15,
		tasks: []string{
			"vendeu o filme que ainda tá no cinema com gente levantando na frente", "fez o kit 3 por 10 na porta do metrô",
			"gravou o CD \"As Mais Tocadas do Arrocha 2009\"", "vendeu DVD de show com o áudio de outro show",
			"montou a banquinha com toalha no chão em dois segundos", "vendeu box de novela mexicana completa em 47 discos",
			"gritou \"lançamento, lançamento\" o dia inteiro",
		},
		busts: []string{
			"recolheu a banquinha correndo quando o rapa chegou", "o cliente voltou porque o filme era outro",
			"queimou cem DVDs e todos saíram em branco", "a chuva molhou toda a capa xerocada",
		},
	},
	{
		name: "professor", emoji: "👩‍🏫", title: "é professor", area: areaServicos, aliases: []string{"professora", "prof"}, min: 130, max: 240,
		every: 28 * time.Minute,
		bust:  0.1, dailyBonus: 40,
		tasks: []string{
			"corrigiu cento e vinte provas no domingo à noite", "explicou equação do segundo grau pela quinta vez",
			"confiscou um celular e ficou vendo os stories da turma", "deu aula pra sala de 40 alunos com o ventilador quebrado",
			"passou trabalho em grupo e um aluno fez tudo sozinho", "dividiu a sala em fila de menino e menina e ainda deu conversa",
			"pegou um aluno colando com a resposta escrita na borracha",
		},
		busts: []string{
			"o pincel acabou no meio da aula e ninguém tinha outro", "a turma bagunçou tanto que a coordenação mandou todo mundo pra casa",
			"o salário atrasou de novo", "foi substituído por um vídeo do YouTube",
		},
	},
	{
		name: "diarista", emoji: "🧽", title: "é diarista", area: areaServicos, aliases: []string{"faxineira", "faxineiro"}, min: 100, max: 200,
		every: 20 * time.Minute,
		finds: []string{"carteira-perdida", "controle-remoto", "chinelo", "cueca", "guarda-chuva", "nota-de-tres"}, findChance: 0.15,
		tasks: []string{
			"faxinou um apartamento que não via vassoura desde 2019", "tirou cabelo de um ralo que parecia uma peruca",
			"limpou a geladeira e achou um pote de feijão do ano passado", "passou pano no chão e o dono pisou logo em seguida",
			"lavou a louça de uma república de estudantes", "arrumou o quarto de um adolescente e achou meia perdida de 2017",
			"desentupiu a pia com a mão e muita fé",
		},
	},
	{
		name: "babá", emoji: "🍼", title: "é babá", area: areaServicos, aliases: []string{"baba"}, min: 90, max: 170, every: 20 * time.Minute,
		peak: peak{from: 19, to: 1, bonus: 50},
		tasks: []string{
			"cantou a música da Galinha Pintadinha quarenta vezes seguidas", "convenceu a criança a comer brócolis dizendo que era árvore",
			"trocou uma fralda de nível catastrófico", "fez a criança dormir e dormiu junto no sofá",
			"assistiu Patrulha Canina o dia inteiro", "separou briga de dois irmãos pelo controle da TV",
			"montou um castelo de almofada na sala",
		},
	},
	{
		name: "cabeleireiro", emoji: "💇", title: "é cabeleireiro", area: areaServicos, aliases: []string{"cabeleireira"}, min: 110, max: 230,
		every: 20 * time.Minute,
		bust:  0.1, doubleChance: 0.1,
		tasks: []string{
			"fez progressiva que deixou o salão inteiro chorando de formol", "cortou só as pontinhas e tirou dez centímetros",
			"fez luzes na cliente e ouviu a vida inteira dela", "fez o penteado da noiva duas horas antes do casamento",
			"descoloriu o cabelo de um moleque pro platinado da Copa", "escovou o cabelo da cliente e choveu na saída",
		},
		busts: []string{
			"a cliente queria o corte da foto e saiu parecendo o Chitãozinho", "o secador queimou no meio da escova",
			"a tinta saiu verde em vez de loiro mel",
		},
	},
	{
		name: "manicure", emoji: "💅", title: "é manicure", area: areaServicos, min: 50, max: 110, every: 8 * time.Minute,
		peak: peak{from: 14, to: 19, bonus: 40},
		tasks: []string{
			"fez unha de gel com desenho de oncinha", "tirou a cutícula sem tirar sangue nenhuma vez",
			"pintou de vermelho e ainda contou a fofoca da rua toda", "fez pé e mão da noiva e das cinco madrinhas",
			"colocou unha postiça que dava pra furar parede", "atendeu a cliente que sempre leva o próprio esmalte",
		},
	},
	{
		name: "barbeiro", emoji: "💈", title: "é barbeiro", area: areaServicos, min: 90, max: 180, every: 20 * time.Minute,
		cashback: 15,
		tasks: []string{
			"fez o degradê navalhado do jogador da várzea", "desenhou o raio na lateral da cabeça de um moleque",
			"aparou a barba e discutiu futebol por uma hora", "fez o pezinho do tiozinho que vai lá toda sexta",
			"fez a sobrancelha com duas riscas de navalha", "cortou o cabelo enquanto via o jogo na TV da barbearia",
		},
	},
	{
		name: "costureira", emoji: "🧵", title: "é costureira", area: areaServicos, aliases: []string{"costureiro"}, min: 90, max: 170,
		every: 20 * time.Minute,
		finds: []string{"cueca", "chinelo", "camisa-do-brasil", "guarda-chuva"}, findChance: 0.12,
		tasks: []string{
			"fez a barra da calça de um cara de 1,90 e um de 1,50", "apertou o vestido da madrinha que jurou que ia emagrecer",
			"costurou o rasgo do short no fundilho", "fez a fantasia de festa junina da escola inteira",
			"pregou um botão e cobrou como se fosse um terno", "reformou a camisa da seleção de 2002",
		},
	},
	{
		name: "passeador de cães", emoji: "🐕", title: "passeia com cachorro", area: areaServicos, aliases: []string{"passeador", "dog walker"},
		min: 60, max: 120, every: 16 * time.Minute,
		guard:  0.15,
		hazard: hazard{chance: 0.07, harm: 15, wounds: []string{"foi arrastado no asfalto pelos 8 cachorros puxando pro mesmo lado", "levou uma mordida de um poodle bravo"}},
		tasks: []string{
			"passeou com oito cachorros ao mesmo tempo sem soltar nenhuma guia", "recolheu um cocô do tamanho de um pão francês",
			"foi arrastado por um golden atrás de um pombo", "levou um pinscher que latiu pra um caminhão",
			"deu volta no quarteirão com um buldogue que deitou no meio da calçada", "fez amizade com todos os porteiros do bairro",
		},
	},
	{
		name: "gari", emoji: "🧹", title: "é gari", area: areaServicos, min: 80, max: 150, every: 20 * time.Minute,
		craft:  craft{activity: "!vasculhar", faster: 40},
		hazard: hazard{chance: 0.07, harm: 25, wounds: []string{"caiu do caminhão do lixo andando", "cortou a mão num caco de vidro no saco de lixo"}},
		tasks: []string{
			"varreu a avenida no dia seguinte ao Carnaval", "sambou com a vassoura na Sapucaí",
			"correu atrás do caminhão de lixo e pulou no estribo", "limpou a praia depois do Réveillon",
			"juntou folha seca e o vento espalhou tudo de novo", "recolheu um sofá inteiro largado na esquina",
		},
	},
	{
		name: "catador de recicláveis", emoji: "♻️", title: "cata recicláveis", area: areaServicos, aliases: []string{"catador", "reciclador"},
		min: 40, max: 100, every: 16 * time.Minute,
		craft:  craft{activity: "!vasculhar", luck: 1, fewerJunk: 30},
		hazard: hazard{chance: 0.07, harm: 15, wounds: []string{"cortou a mão numa lata de alumínio", "virou a carroça na ladeira"}},
		tasks: []string{
			"juntou três sacos de latinha depois do show", "empurrou a carroça cheia de papelão ladeira acima",
			"separou o lixo do condomínio que ninguém separa", "achou uma geladeira inteira jogada no terreno",
			"vendeu dez quilos de cobre no ferro-velho", "recolheu as garrafas pet do churrasco da rua",
		},
	},
	{
		name: "técnico de celular", emoji: "📱", title: "é técnico de celular", area: areaServicos, aliases: []string{"técnico"}, min: 110,
		max: 220, every: 20 * time.Minute,
		bust:  0.1,
		finds: []string{"nokia", "cabo-usb", "fone", "controle-remoto", "iphone"}, findChance: 0.06,
		tasks: []string{
			"trocou a tela de um celular que caiu na privada", "colocou película 3D em quinze celulares seguidos",
			"tirou o arroz de dentro de um celular molhado", "desbloqueou o celular de uma tia que esqueceu o padrão",
			"trocou a bateria estufada de um Moto G", "consertou um celular dando um tapa nele",
			"recuperou as fotos do casamento de um celular que não ligava",
		},
		busts: []string{
			"sobrou um parafuso depois de montar o celular", "o celular voltou a não ligar logo depois do cliente pagar",
			"quebrou a tela nova enquanto colocava a película",
		},
	},
	{
		name: "chaveiro", emoji: "🔑", title: "é chaveiro", area: areaServicos, min: 80, max: 150, every: 20 * time.Minute,
		theftBonus: 0.12, escapeBonus: 0.3,
		heistBonus:    0.08,
		breakoutBonus: 0.3,
		tasks: []string{
			"abriu um carro com um arame e um sorriso", "fez cópia da chave da casa da sogra sem ela saber",
			"tirou o cliente trancado fora de casa de cueca", "abriu um cofre antigo que só tinha boleto vencido",
			"trocou o segredo da fechadura depois de um término", "atendeu chamado de emergência às três da manhã",
		},
	},
	{
		name: "garoto de programa", emoji: "💋", title: "é garoto de programa", area: areaServicos,
		aliases: []string{"garota de programa", "gp", "acompanhante"},
		min:     100, max: 220, every: 20 * time.Minute,
		bust: 0.15, doubleChance: 0.15, reportBonus: 0.2,
		peak: peak{from: 22, to: 4, bonus: 90},
		tasks: []string{
			"atendeu um deputado que pediu nota fiscal no nome do gabinete", "foi acompanhante de um tiozinho no casamento da ex dele",
			"ouviu duas horas de desabafo do cliente sobre o chefe", "aceitou pix, cartão e vale-refeição na mesma noite",
			"atualizou o anúncio com foto de dez anos atrás", "fingiu que não reconheceu o vizinho casado",
			"ganhou um perfume importado de um cliente fiel",
		},
		busts: []string{
			"o cliente mandou o comprovante de um pix agendado que nunca caiu", "o cliente era o pastor da igreja da mãe",
			"ficou a noite toda na esquina debaixo de chuva sem ninguém aparecer", "a esposa do cliente chegou antes da hora",
		},
	},
	{
		name: "padre", emoji: "⛪", title: "é padre", area: areaFe, aliases: []string{"sacerdote", "vigario", "frei"},
		min: 110, max: 190, every: 20 * time.Minute,
		peak: peak{from: 7, to: 10, bonus: 40},
		faith: faith{
			blessings: []string{"protecao", "justica", "purificacao", "multiplicacao"}, curses: []string{"excomunhao"}, hexChance: 0.5,
			blessRite: "jogou água benta em", curseRite: "excomungou",
		},
		tasks: []string{
			"rezou a missa das sete pra três velhinhas e um cachorro", "ouviu a confissão de um cara que só queria desabafar do Corinthians",
			"batizou um bebê que chorou a cerimônia inteira", "benzeu o carro novo do prefeito",
			"fez quermesse e vendeu duzentos pastéis", "casou um casal que já tá no terceiro casamento",
		},
	},
	{
		name: "pastor", emoji: "📖", title: "é pastor", area: areaFe, aliases: []string{"pastora", "bispo", "missionario"},
		min: 80, max: 260, every: 22 * time.Minute, doubleChance: 0.2,
		bust: 0.1, busts: []string{
			"o fiel pediu o dízimo de volta no Procon", "a live do culto caiu bem na hora da oferta",
		},
		faith: faith{
			blessings: []string{"prosperidade", "fartura", "multiplicacao"}, curses: []string{"miseria", "excomunhao", "dizimo"}, hexChance: 0.55,
			blessRite: "orou em línguas por", curseRite: "repreendeu em nome de Jesus",
		},
		tasks: []string{
			"passou a sacolinha do dízimo três vezes no mesmo culto", "fez culto de libertação ao vivo no YouTube",
			"vendeu água ungida do Rio Jordão que saiu da torneira", "pregou duas horas sobre humildade e foi embora de BMW",
			"fez a campanha da fogueira santa", "batizou meio bairro numa piscina de plástico",
		},
	},
	{
		name: "mãe de santo", emoji: "🕯️", title: "é mãe de santo", area: areaFe,
		aliases: []string{"mae de santo", "pai de santo", "macumbeira", "macumbeiro", "ialorixa", "babalorixa"},
		min:     100, max: 200, every: 20 * time.Minute,
		faith: faith{
			blessings: []string{"protecao", "sorte", "purificacao", "maoboa", "abrecaminho"}, curses: []string{"olhogordo", "azar", "encosto", "lerdeza"}, hexChance: 0.7,
			blessRite: "acendeu uma vela branca pra", curseRite: "fez um despacho na encruzilhada pra",
		},
		tasks: []string{
			"jogou búzios e acertou o nome do ex do cliente", "fez um trabalho pra arrumar emprego e o cliente passou na entrevista",
			"deu passe numa fila de vinte pessoas", "limpou a energia de uma loja que não vendia nada",
			"tocou atabaque até o dia amanhecer", "fez banho de ervas pra uma cliente que tava com encosto",
		},
	},
	{
		name: "bruxa", emoji: "🧙", title: "é bruxa", area: areaFe, aliases: []string{"bruxo", "feiticeira", "feiticeiro", "wicca"},
		min: 90, max: 220, every: 22 * time.Minute,
		bust: 0.12, busts: []string{
			"errou a poção e o cliente ficou três dias com soluço", "a vassoura quebrou no meio do voo",
		},
		faith: faith{
			blessings: []string{"sorte", "prosperidade"}, curses: []string{"miseria", "azar", "jejum", "encosto", "lerdeza", "dizimo", "panema", "assombracao"}, hexChance: 0.75,
			blessRite: "fez uma poção da sorte pra", curseRite: "mexeu o caldeirão pensando em",
		},
		tasks: []string{
			"vendeu poção do amor com validade de sete dias", "fez um boneco de pano do chefe de um cliente",
			"leu o futuro na borra do café e só viu boleto", "fez feitiço pro crush responder e ele respondeu 'kkk'",
			"colheu erva na lua cheia e cobrou adicional noturno", "abriu um curso online de feitiçaria em 12x",
		},
	},
	{
		name: "benzedeira", emoji: "🌿", title: "é benzedeira", area: areaFe, aliases: []string{"benzedeiro", "rezadeira", "rezadeiro", "curandeira"},
		min: 60, max: 130, every: 14 * time.Minute,
		faith: faith{
			blessings: []string{"purificacao", "protecao", "fartura", "achado"}, curses: []string{"olhogordo"}, hexChance: 0.4,
			blessRite: "passou um raminho de arruda em", curseRite: "rogou uma praga em",
		},
		tasks: []string{
			"tirou quebranto de um bebê que não parava de chorar", "curou espinhela caída com três ave-marias",
			"benzeu uma criança com mau-olhado e cobrou um bolo de fubá", "fez chá de boldo pro vizinho de ressaca",
			"rezou o terço por uma vaca doente", "tirou o olho gordo do carro novo do sobrinho",
		},
	},
	{
		name: "xamã", emoji: "🪶", title: "é xamã", area: areaFe, aliases: []string{"xama", "paje", "druida"},
		min: 100, max: 190, every: 22 * time.Minute, breakoutBonus: 0.15,
		faith: faith{
			blessings: []string{"libertacao", "fartura", "sorte", "pressa", "maoboa", "faro"}, curses: []string{"jejum", "encosto", "panema"}, hexChance: 0.6,
			blessRite: "fez uma roda de tambor por", curseRite: "invocou os espíritos da floresta contra",
		},
		tasks: []string{
			"fez ritual de chuva e alagou a rua do lado", "conduziu uma cerimônia pra executivos estressados da Faria Lima",
			"conversou com o espírito do rio e ele pediu pra pararem de jogar lixo", "defumou o escritório inteiro e disparou o alarme de incêndio",
			"ensinou a galera a meditar no tambor", "descobriu que o animal de poder de um coach era um pombo",
		},
	},
	{
		name: "exorcista", emoji: "✝️", title: "é exorcista", area: areaFe, aliases: []string{"exorcizador", "caca fantasma"},
		min: 130, max: 240, every: 28 * time.Minute, guard: 0.1,
		bust: 0.15, busts: []string{
			"o demônio era só o Wi-Fi com defeito", "o possuído era um adolescente fingindo pra faltar na escola",
		},
		faith: faith{
			blessings: []string{"purificacao", "libertacao"}, curses: []string{"encosto", "assombracao"}, hexChance: 0.65,
			blessRite: "fez um exorcismo em", curseRite: "mandou um encosto pra cima de",
		},
		hazard: hazard{chance: 0.1, harm: 25, wounds: []string{"levou uma rasteira do demônio que saiu do possuído", "levou uma cabeçada de um possuído"}},
		tasks: []string{
			"tirou um demônio de uma casa e ainda cobrou a mudança", "exorcizou uma boneca que andava sozinha (era um carrinho de controle remoto)",
			"jogou água benta na sogra a pedido do genro", "fez exorcismo por chamada de vídeo",
			"expulsou um espírito que só queria a senha do Wi-Fi", "botou uma Bíblia debaixo do travesseiro do cliente e cobrou consulta",
		},
	},
	{
		name: "monge", emoji: "📿", title: "é monge", area: areaFe, aliases: []string{"monja", "budista"},
		min: 50, max: 110, every: 12 * time.Minute, interest: 2, jailFactor: -30,
		faith: faith{
			blessings: []string{"justica", "libertacao", "protecao", "pressa"}, curses: []string{"jejum"}, hexChance: 0.45,
			blessRite: "meditou três horas por", curseRite: "impôs um voto de jejum a",
		},
		tasks: []string{
			"meditou oito horas e esqueceu de almoçar", "varreu o templo três vezes sem reclamar",
			"fez voto de silêncio e ganhou a discussão no grupo da família", "ensinou respiração pra um gerente surtado",
			"pintou uma mandala de areia e um cachorro passou em cima", "vendeu incenso artesanal na feira",
		},
	},
}
