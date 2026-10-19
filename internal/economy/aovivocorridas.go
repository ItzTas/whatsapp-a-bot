package economy

import "time"

var grabScenarios = []grabScenario{
	{
		id:    "corrida-malote",
		title: "O malote do carro-forte",
		intros: []string{
			"O carro-forte fez a curva rápido demais na avenida e um malote voou pela janela. Tá lá, caído no meio-fio, com o lacre ainda inteiro",
			"Barulho de freada, porta abrindo, gritaria... e um malote do carro-forte rolando sozinho pela calçada. Os seguranças ainda não perceberam",
		},
		endings: []ending{
			{weight: 70, texts: []string{"{@} meteu o malote debaixo da camisa e saiu assobiando. Lá dentro tinha um bolo de nota que dá pra pagar uns boletos", "{@} correu, pegou e sumiu na multidão antes do segurança virar. Dinheiro vivo, cheirando a banco"}, effects: []effect{earn(120, 260)}},
			{weight: 20, texts: []string{"Era isca! O malote tinha rastreador e a polícia tava esperando na esquina. {@} saiu com um trocado no bolso e algemado", "A Operação Malote Fácil pegou {@} de primeira. Pelo menos deu tempo de esconder umas notas na meia"}, effects: []effect{earn(30, 60), nabbed()}},
			{weight: 10, texts: []string{"Dentro do malote só tinha comprovante de depósito e um sanduíche de mortadela. {@} comeu o sanduíche", "{@} abriu o malote: papelada do banco e uma bala de menta. Que decepção"}, effects: []effect{heal(5)}},
		},
		missed: []string{"Os seguranças voltaram, recolheram o malote e ainda olharam feio pro grupo inteiro"},
	},
	{
		id:    "corrida-pix-bugado",
		title: "O aplicativo bugado",
		intros: []string{
			"Um banco digital bugou e tá mandando Pix de verdade pra quem clicar no link primeiro. Já tem gente comentando no Twitter",
			"Corre no grupo do condomínio: o app do banco tá com erro e o primeiro que pedir estorno recebe em dobro",
		},
		endings: []ending{
			{weight: 75, texts: []string{"{@} clicou primeiro e o dinheiro caiu na hora. O banco consertou o bug cinco minutos depois", "O Pix entrou na conta de {@} antes do estagiário do banco acordar"}, effects: []effect{earn(100, 220)}},
			{weight: 25, texts: []string{"Era golpe do link falso. {@} digitou a senha num site que imitava o banco e viu o saldo emagrecer", "O link era de um golpista. {@} ainda recebeu um \"obrigado pela colaboração\""}, effects: []effect{lose(3, 8)}},
		},
		missed: []string{"O banco consertou o bug e mandou um e-mail pedindo desculpas pelo transtorno"},
	},
	{
		id:    "corrida-caminhao-picanha",
		title: "O caminhão de picanha",
		intros: []string{
			"Um caminhão frigorífico tombou na BR e as caixas de picanha tão espalhadas no acostamento. O motorista tá bem, só meio tonto",
			"Notícia urgente no grupo da família: caminhão de carne virou perto do posto, e a picanha tá lá pegando sol",
		},
		endings: []ending{
			{weight: 60, texts: []string{"{@} chegou primeiro e voltou com uma caixa nas costas. O churrasco de domingo tá garantido", "{@} encheu o porta-malas. A picanha ainda tava gelada"}, effects: []effect{gain(2, "picanha")}},
			{weight: 25, texts: []string{"{@} pegou a picanha, mas o repórter da TV local filmou tudo e o vídeo tá no jornal do meio-dia", "Saiu no jornal: \"populares saqueiam carga\". Quem aparece em primeiro plano é {@}"}, effects: []effect{gain(1, "picanha"), stain()}},
			{weight: 15, texts: []string{"O motorista acordou do susto, viu {@} com a caixa e deu uma voadora. A picanha ficou", "{@} escorregou na gordura que vazou da carga e caiu de bunda no asfalto"}, effects: []effect{injure(10, 25)}},
		},
		missed: []string{"A polícia rodoviária chegou e isolou a carga. Hoje ninguém come picanha"},
	},
	{
		id:    "corrida-cofre",
		title: "O cofre no terreno baldio",
		lock:  "O cofre abre com o resultado de",
		riddle: arithmetic,
		intros: []string{
			"Acharam um cofre velho no meio do mato do terreno baldio. Tem um teclado numérico enferrujado e um bilhete colado: \"a senha é a conta\"",
			"Uma retroescavadeira desenterrou um cofre na obra da esquina. O peão jura que ouviu moeda chacoalhando lá dentro",
		},
		endings: []ending{
			{weight: 60, texts: []string{"Clique! O cofre abriu na mão de {@}, cheio de nota velha que o banco ainda troca", "{@} digitou a senha certa e o cofre cuspiu um maço de dinheiro esquecido"}, effects: []effect{earn(150, 300)}},
			{weight: 25, texts: []string{"Lá dentro tinha uma pepita de ouro enrolada num lenço. {@} guardou antes de alguém ver", "{@} abriu e achou uma pepita de ouro e uma carta de amor de 1987"}, effects: []effect{gain(1, "ouro")}},
			{weight: 15, texts: []string{"O cofre tinha tinta antifurto. {@} ficou azul dos pés à cabeça e a polícia já sabe quem mexeu", "Explodiu tinta azul na cara de {@}. Agora não tem como negar"}, effects: []effect{stain()}},
		},
		missed: []string{"Ninguém acertou a senha, e o dono do terreno levou o cofre de caminhonete"},
	},
	{
		id:     "corrida-bau-pirata",
		title:  "O baú do pirata",
		rarity: rareEvent,
		lock:   "O cadeado do baú tem a palavra embaralhada",
		riddle: scrambled("tesouro", "pirata", "caravela", "papagaio", "corsario", "naufragio"),
		intros: []string{
			"A maré baixou mais do que devia e apareceu um baú encravado na areia, com cadeado de letra e tudo",
			"Um pescador puxou a rede e veio um baú de madeira cheio de craca. O cadeado tem umas letras misturadas",
		},
		endings: []ending{
			{weight: 60, texts: []string{"{@} abriu o baú e o brilho cegou meio mundo. Tesouro de verdade", "O cadeado cedeu e {@} saiu arrastando o baú pela praia"}, effects: []effect{gain(1, "bau-do-tesouro")}},
			{weight: 30, texts: []string{"O baú tinha moeda antiga que um colecionador comprou na hora de {@}", "{@} vendeu o conteúdo do baú pro antiquário antes do sol se pôr"}, effects: []effect{earn(200, 400)}},
			{weight: 10, texts: []string{"No fundo do baú tinha um bilhete: \"quem abrir, carrega a maldição do capitão\". {@} sentiu um arrepio", "Uma voz rouca saiu do baú e jogou uma praga em {@}"}, effects: []effect{earn(50, 100), enchant("azar")}},
		},
		missed: []string{"A maré subiu e levou o baú de volta pro fundo do mar"},
	},
	{
		id:    "corrida-xepa",
		title: "A xepa da feira",
		intros: []string{
			"Meio-dia na feira e o feirante gritou: \"é a xepa! O que sobrar é de graça pro primeiro que chegar\"",
			"O moço do pastel tá desmontando a barraca e oferecendo o que sobrou: quem pegar primeiro leva tudo",
		},
		endings: []ending{
			{weight: 50, texts: []string{"{@} chegou primeiro e saiu com as duas mãos cheias", "{@} encheu a sacola de feira antes de qualquer um piscar"}, effects: []effect{gain(4, "pastel", "pamonha", "saco-de-laranja")}},
			{weight: 30, texts: []string{"O feirante gostou da disposição de {@}, deu um caldo de cana e ainda pagou por ajudar a desmontar", "{@} pegou a xepa e revendeu na porta do metrô"}, effects: []effect{earn(60, 140), gain(1, "caldo-de-cana")}},
			{weight: 20, texts: []string{"Só sobrou o tomate amassado, e {@} passou a tarde no banheiro", "{@} comeu o pastel de vento que tava no sol desde as seis da manhã. Erro grave"}, effects: []effect{injure(5, 12)}},
		},
		missed: []string{"Os pombos chegaram primeiro e limparam a xepa"},
	},
	{
		id:     "corrida-iphone",
		title:  "O iPhone esquecido no busão",
		rarity: rareEvent,
		lock:   "Pra desbloquear a tela,",
		riddle: counting("📱", "📞", "💬", "🔋", "📶"),
		intros: []string{
			"Alguém esqueceu um iPhone no banco do ônibus e desceu. Tá tocando sem parar com \"Mozão\" no visor",
			"Um iPhone caiu do bolso de um engravatado no metrô. A tela tá desbloqueando com um desafio esquisito",
		},
		endings: []ending{
			{weight: 40, texts: []string{"{@} desbloqueou e o celular é seu agora. O \"Mozão\" vai ter que ligar pra outro", "{@} enfiou o iPhone no bolso e desceu no próximo ponto"}, effects: []effect{gain(1, "iphone")}},
			{weight: 35, texts: []string{"{@} devolveu pro dono, que era rico e deu uma recompensa gorda", "O dono apareceu desesperado e pagou {@} pra ter as fotos de volta"}, effects: []effect{earn(80, 180)}},
			{weight: 25, texts: []string{"O dono era policial à paisana e viu {@} guardando o celular. Flagrante", "Era celular de isca da polícia. {@} caiu bonito"}, effects: []effect{nabbed()}},
		},
		missed: []string{"O motorista achou o celular primeiro e levou pro achados e perdidos. Ou pra casa, vai saber"},
	},
	{
		id:     "corrida-bilhete",
		title:  "O bilhete premiado no chão",
		rarity: legendaryEvent,
		lock:   "Pra conferir se é o bilhete certo, os números somam",
		riddle: arithmetic,
		intros: []string{
			"Um velhinho saiu da lotérica comemorando, tropeçou e o bilhete voou. Tá no chão, virado pra cima, com os seis números marcados",
			"No chão da lotérica tem um bilhete da Mega-Sena amassado. Do lado, o resultado de ontem pregado na parede bate com ele",
		},
		endings: []ending{
			{weight: 50, texts: []string{"{@} pegou o bilhete e conferiu três vezes. É ele mesmo, premiado", "O bilhete tá na carteira de {@} agora, guardado como se fosse um órgão"}, effects: []effect{gain(1, "bilhete-premiado")}},
			{weight: 30, texts: []string{"O velhinho voltou chorando e {@} devolveu. Ele dividiu o prêmio por gratidão", "{@} vendeu o bilhete pra um agiota na hora, com desconto mas em dinheiro vivo"}, effects: []effect{earn(400, 900)}},
			{weight: 20, texts: []string{"{@} correu pra lotérica e descobriu: era o bilhete do ano passado. O azar grudou", "O bilhete era falso, impresso em casa pra uma pegadinha de YouTube. {@} virou o vídeo"}, effects: []effect{enchant("azar")}},
		},
		missed: []string{"Ventou forte e o bilhete foi parar no bueiro. Alguém lá embaixo tá rico agora"},
	},
	{
		id:    "corrida-pombo-correio",
		title: "O pombo-correio perdido",
		intros: []string{
			"Um pombo de anilha pousou na janela com um bilhetinho amarrado na pata. Parece cansado e tá aceitando carinho",
			"Tá chovendo pombo-correio? Um deles caiu exausto no meio da roda, com um envelope preso na perna",
		},
		endings: []ending{
			{weight: 40, texts: []string{"O envelope tinha dinheiro de uma aposta antiga. {@} ficou com tudo", "{@} abriu o bilhete: era um Pix impresso, e funcionou"}, effects: []effect{earn(80, 180)}},
			{weight: 30, texts: []string{"O pombo gostou de {@} e não foi mais embora. Agora tem um bicho de estimação", "{@} adotou o pombo, que agora entrega recado pra ninguém"}, effects: []effect{gain(1, "pombo")}},
			{weight: 20, texts: []string{"O bilhete dizia só \"boa sorte pra quem me achar\". E não é que funcionou pra {@}?", "Era uma mensagem de um bruxo: quem pegasse ganhava sorte. {@} ganhou"}, effects: []effect{enchant("sorte")}},
			{weight: 10, texts: []string{"O pombo fez o serviço completo na cabeça de {@} e foi embora", "{@} pegou o pombo, o pombo bicou {@} e ainda fez cocô na blusa"}, effects: []effect{injure(2, 6), enchant("azar")}},
		},
		missed: []string{"O pombo descansou, comeu um farelo e voou pra casa"},
	},
	{
		id:     "corrida-mala-deputado",
		title:  "A mala do deputado",
		rarity: rareEvent,
		lock:   "O segredo da mala é a palavra embaralhada",
		riddle: scrambled("propina", "emenda", "laranja", "mensalao", "caixinha", "lobista"),
		intros: []string{
			"Um carro preto parou, um assessor desceu correndo e esqueceu uma mala no ponto de ônibus. O cadeado tem segredo de letra",
			"A PF tá na porta do gabinete e alguém jogou uma mala pela janela do terceiro andar. Caiu bem no meio do grupo",
		},
		endings: []ending{
			{weight: 45, texts: []string{"{@} abriu a mala: dinheiro vivo, separado em elástico. A ficha suja vem de brinde", "A mala tava cheia. {@} sumiu com ela, mas uma câmera pegou o rosto"}, effects: []effect{earn(250, 500), stain()}},
			{weight: 30, texts: []string{"{@} levou a mala inteira pra casa e escondeu debaixo da cama", "{@} pegou a mala, fechou e saiu como se fosse o dono"}, effects: []effect{gain(1, "mala-de-dinheiro"), stain()}},
			{weight: 25, texts: []string{"A PF tava filmando tudo do prédio da frente. {@} foi preso junto com o assessor", "Era a prova do inquérito, e {@} acabou de entrar no processo"}, effects: []effect{earn(80, 150), nabbed()}},
		},
		missed: []string{"O assessor voltou, pegou a mala e ainda pediu voto pra eleição"},
	},
	{
		id:    "corrida-promocao",
		title: "A promoção relâmpago",
		intros: []string{
			"A loja de eletrônicos do centro anunciou no megafone: \"a primeira pessoa que chegar no caixa leva uma TV por um real\"",
			"Black Friday fora de época: a loja tá vendendo o mostruário de graça pro primeiro que gritar",
		},
		endings: []ending{
			{weight: 60, texts: []string{"{@} chegou primeiro, pagou um real e revendeu a TV no Marketplace no mesmo dia", "{@} saiu da loja com a TV na cabeça e um sorriso no rosto"}, effects: []effect{earn(100, 200)}},
			{weight: 25, texts: []string{"A TV já tinha ido, mas {@} levou o prêmio de consolação", "{@} chegou primeiro, mas a promoção era só pro fone de ouvido"}, effects: []effect{gain(1, "fone")}},
			{weight: 15, texts: []string{"Na corrida até o caixa, {@} tomou uma cotovelada de uma senhora e caiu no corredor de panela", "{@} chegou primeiro, mas a fila passou por cima"}, effects: []effect{injure(10, 25)}},
		},
		missed: []string{"A promoção acabou e o gerente admitiu que era só pra atrair cliente"},
	},
	{
		id:    "corrida-cachorro-madame",
		title: "O cachorro da madame",
		lock:  "Pra achar o cachorro no meio da bicharada,",
		riddle: counting("🐶", "🐱", "🐭", "🐰", "🐹"),
		intros: []string{
			"Cartaz em todo poste: \"procura-se Lulu da Pomerânia, recompensa generosa\". E não é que o Lulu tá ali na praça?",
			"A madame do prédio tá chorando na portaria: o cachorrinho fugiu. Dizem que ele se escondeu no pet shop da esquina",
		},
		endings: []ending{
			{weight: 70, texts: []string{"{@} levou o Lulu de volta e a madame pagou a recompensa chorando de alegria", "{@} devolveu o cachorro e ganhou um envelope gordo e um beijo na testa da madame"}, effects: []effect{earn(150, 280)}},
			{weight: 20, texts: []string{"O cachorro gostou tanto de {@} que agora rosna pra quem chega perto. Melhor segurança da cidade", "{@} ganhou um amigo de quatro patas que late pra ladrão"}, effects: []effect{enchant("protecao")}},
			{weight: 10, texts: []string{"O Lulu não quis voltar e mordeu a canela de {@}", "{@} tentou pegar o cachorro, mas era outro cachorro, e bem bravo"}, effects: []effect{injure(8, 20)}},
		},
		missed: []string{"O cachorro voltou pra casa sozinho, todo sujo, e a madame não pagou ninguém"},
	},
	{
		id:     "corrida-caixa-eletronico",
		title:  "O caixa eletrônico bugado",
		rarity: rareEvent,
		intros: []string{
			"O caixa 24h da esquina enlouqueceu e tá cuspindo nota sem parar. Ainda não tem ninguém na fila",
			"Bug no caixa eletrônico do posto: qualquer cartão saca o dobro. A notícia ainda não espalhou",
		},
		endings: []ending{
			{weight: 55, texts: []string{"{@} encheu os bolsos de nota e o caixa travou logo depois", "{@} chegou primeiro e sacou até a máquina pedir água"}, effects: []effect{earn(200, 400)}},
			{weight: 30, texts: []string{"{@} sacou bastante, mas a câmera do caixa registrou o rosto em alta definição", "O dinheiro veio, e a ficha suja junto"}, effects: []effect{earn(100, 200), stain()}},
			{weight: 15, texts: []string{"O banco mandou a viatura na hora. {@} foi preso com nota saindo do bolso", "Era armadilha do banco pra pegar golpista, e {@} caiu"}, effects: []effect{nabbed()}},
		},
		missed: []string{"O técnico do banco chegou, desligou a máquina e colou um papel de \"fora de serviço\""},
	},
	{
		id:    "corrida-carta-pokemon",
		title: "A carta rara no lixo",
		lock:  "Pra provar que entende do assunto, desembaralha o nome do Pokémon",
		riddle: scrambled("pikachu", "charizard", "mewtwo", "snorlax", "gengar", "bulbasaur"),
		intros: []string{
			"Uma mãe jogou fora a coleção de cartas do filho que tá na faculdade. A caixa tá na calçada esperando o lixeiro",
			"Tem um fichário de cartas de Pokémon jogado na lixeira do prédio. Do lado, um bilhete: \"pode pegar\"",
		},
		endings: []ending{
			{weight: 60, texts: []string{"{@} achou uma holográfica no meio das cartas. Vale mais que o aluguel", "Brilhou! {@} saiu com a carta holográfica rara"}, effects: []effect{gain(1, "carta-holografica")}},
			{weight: 30, texts: []string{"{@} vendeu o fichário inteiro pra uma loja de colecionador", "Um colecionador viu {@} com a caixa e comprou ali mesmo"}, effects: []effect{earn(80, 160)}},
			{weight: 10, texts: []string{"Tudo falsificado, impresso em papel sulfite. {@} ainda pagou o Uber pra levar pra casa", "As cartas eram de mentirinha, e {@} perdeu o tempo e a dignidade"}, effects: []effect{lose(1, 3)}},
		},
		missed: []string{"O lixeiro passou e levou a infância de alguém embora"},
	},
	{
		id:    "corrida-sorteio-live",
		title: "O sorteio na live",
		intros: []string{
			"Uma influencer abriu live e prometeu um Pix pro primeiro que comentar \"eu quero\"",
			"Live de sorteio rolando: o primeiro que mandar o código leva o prêmio na hora",
		},
		endings: []ending{
			{weight: 60, texts: []string{"{@} comentou primeiro e o Pix caiu no meio da live", "A influencer leu o nome de {@} e mandou o prêmio na hora"}, effects: []effect{earn(80, 180)}},
			{weight: 25, texts: []string{"{@} ganhou e ainda foi chamado pra live. Agora todo mundo deseja coisa boa pra {@}", "{@} virou o assunto da live e saiu com a energia lá em cima"}, effects: []effect{enchant("prosperidade")}},
			{weight: 15, texts: []string{"Era perfil fake. {@} passou os dados do cartão pra \"receber o prêmio\"", "O sorteio era golpe, e o golpista cobrou uma \"taxa de liberação\" de {@}"}, effects: []effect{lose(2, 6)}},
		},
		missed: []string{"A live caiu e a influencer sumiu com o prêmio"},
	},
	{
		id:     "corrida-meteoro",
		title:  "A pedra que caiu do céu",
		rarity: legendaryEvent,
		lock:   "O contador Geiger mostra",
		riddle: arithmetic,
		intros: []string{
			"Um clarão no céu, um estrondo, e uma pedra fumegando caiu no campinho de futebol. Ainda tá quente",
			"Caiu um meteorito no telhado da padaria! O padeiro jura que a pedra brilha no escuro",
		},
		endings: []ending{
			{weight: 50, texts: []string{"{@} pegou a pedra com uma luva de forno e guardou. É meteorito de verdade", "{@} chegou primeiro e levou a pedra do espaço pra casa"}, effects: []effect{gain(1, "meteorito")}},
			{weight: 30, texts: []string{"Um museu comprou o meteorito de {@} na mesma tarde", "{@} vendeu a pedra pra um cientista que apareceu do nada de jaleco"}, effects: []effect{earn(300, 700)}},
			{weight: 20, texts: []string{"A pedra tava quente demais, e {@} queimou a mão inteira", "{@} pegou a pedra sem luva. O hospital já conhece {@} pelo nome"}, effects: []effect{injure(20, 40)}},
		},
		missed: []string{"O exército chegou de helicóptero e levou a pedra pra uma base secreta"},
	},
	{
		id:    "corrida-lanche-reuniao",
		title: "O lanche que sobrou da reunião",
		intros: []string{
			"A diretoria saiu da reunião e deixou uma bandeja de salgado intocada na copa. Ainda tá quentinho",
			"Sobrou o coffee break do evento da firma, e o segurança disse que quem pegar primeiro pode levar",
		},
		endings: []ending{
			{weight: 50, texts: []string{"{@} levou a bandeja inteira pra mesa e não dividiu com ninguém", "{@} chegou primeiro e encheu o pote de sorvete vazio de salgado"}, effects: []effect{gain(3, "coxinha", "pao-de-queijo", "esfiha")}},
			{weight: 35, texts: []string{"{@} comeu até passar mal de felicidade. Recuperou o ânimo pra semana", "Barriga cheia, coração feliz. {@} tá novo em folha"}, effects: []effect{heal(15), gain(1, "coxinha")}},
			{weight: 15, texts: []string{"A chefe pegou {@} com a boca cheia de coxinha no horário de trabalho", "{@} foi flagrado levando salgado na bolsa e ganhou hora extra de castigo"}, effects: []effect{delay(30 * time.Minute)}},
		},
		missed: []string{"A moça da limpeza levou a bandeja pra casa, e ela tá certa"},
	},
	{
		id:     "corrida-lampada",
		title:  "A lâmpada do brechó",
		rarity: legendaryEvent,
		lock:   "Na lâmpada tem uma palavra embaralhada",
		riddle: scrambled("aladim", "desejo", "genio", "tapete", "agrabah"),
		intros: []string{
			"No fundo do brechó, uma lâmpada de latão começou a tremer sozinha. O dono do brechó jura que não é dele",
			"Uma lâmpada antiga apareceu na feira de antiguidades sem preço e sem dono. Dizem que esquenta quando alguém encosta",
		},
		endings: []ending{
			{weight: 45, texts: []string{"{@} esfregou a lâmpada e ela piscou. Guardou correndo antes de alguém ver", "{@} pegou a lâmpada e sentiu ela vibrar no bolso"}, effects: []effect{gain(1, "lampada-magica")}},
			{weight: 35, texts: []string{"Saiu um gênio sonolento que deu um desejo pequeno pra {@}: dinheiro, claro", "O gênio tava de mau humor, mas pagou {@} pra voltar a dormir"}, effects: []effect{earn(300, 600)}},
			{weight: 20, texts: []string{"O gênio era um encosto disfarçado e agora mora com {@}", "Saiu fumaça preta da lâmpada e grudou em {@}"}, effects: []effect{enchant("encosto")}},
		},
		missed: []string{"A lâmpada sumiu no ar, e o brechó voltou a vender roupa usada"},
	},
}
