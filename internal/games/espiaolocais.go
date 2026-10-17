package games

type espiaoLocation struct {
	emoji string
	name  string
	roles []string
}

var espiaoLocations = []espiaoLocation{
	{"🥖", "Padaria", []string{"padeiro", "caixa", "confeiteiro", "atendente do balcão", "entregador de pão", "cliente com pressa", "aposentado lendo jornal", "criança pedindo sonho"}},
	{"🏥", "Hospital", []string{"médico", "enfermeiro", "cirurgião", "recepcionista", "paciente", "acompanhante", "residente sem dormir", "faxineiro"}},
	{"🏫", "Escola", []string{"professor", "diretor", "merendeira", "inspetor", "porteiro", "aluno do fundão", "aluno CDF", "mãe na reunião"}},
	{"🚓", "Delegacia", []string{"delegado", "escrivão", "investigador", "preso", "advogado", "vítima de furto", "testemunha", "jornalista"}},
	{"⛪", "Igreja", []string{"padre", "coroinha", "sacristão", "organista", "beata", "fiel", "noiva", "turista"}},
	{"🏖️", "Praia", []string{"salva-vidas", "surfista", "vendedor de mate", "vendedor de queijo coalho", "turista", "banhista torrando no sol", "pescador", "criança fazendo castelo de areia"}},
	{"⚽", "Estádio de futebol", []string{"jogador", "goleiro", "técnico", "juiz", "gandula", "narrador", "torcedor", "vendedor de cerveja"}},
	{"🛍️", "Shopping", []string{"vendedor de loja", "gerente", "segurança", "cliente", "atendente da praça de alimentação", "adolescente de rolê", "Papai Noel", "faxineiro"}},
	{"🏋️", "Academia", []string{"personal trainer", "recepcionista", "dono da academia", "professor de spinning", "nutricionista", "marombeiro", "iniciante perdido", "aluno que só tira foto"}},
	{"🎬", "Cinema", []string{"bilheteiro", "pipoqueiro", "projecionista", "lanterninha", "crítico de cinema", "casal de namorados", "criança com medo", "pessoa que fala no meio do filme"}},
	{"✈️", "Avião", []string{"piloto", "copiloto", "comissário de bordo", "mecânico", "executivo", "passageiro com medo de voar", "passageiro da janela", "bebê chorando"}},
	{"🚢", "Navio de cruzeiro", []string{"capitão", "marinheiro", "garçom", "cozinheiro", "animador", "músico", "turista", "casal em lua de mel"}},
	{"🎪", "Circo", []string{"palhaço", "mágico", "trapezista", "malabarista", "domador", "dono do circo", "pipoqueiro", "criança na plateia"}},
	{"🦁", "Zoológico", []string{"tratador", "veterinário", "biólogo", "guia", "segurança", "fotógrafo", "visitante", "criança na excursão"}},
	{"🛒", "Supermercado", []string{"caixa", "repositor", "açougueiro", "gerente", "empacotador", "segurança", "moça da degustação", "cliente"}},
	{"💇", "Salão de beleza", []string{"cabeleireiro", "manicure", "maquiador", "dona do salão", "recepcionista", "cliente", "cliente fofoqueira", "noiva"}},
	{"🌽", "Festa junina", []string{"sanfoneiro", "puxador da quadrilha", "noivo da quadrilha", "noiva da quadrilha", "padre do casamento caipira", "vendedor de pamonha", "quem cuida da fogueira", "criança na pescaria"}},
	{"💒", "Casamento", []string{"noiva", "noivo", "padrinho", "madrinha", "fotógrafo", "DJ", "garçom", "tia que pergunta quando é o seu"}},
	{"🎉", "Baile funk", []string{"DJ", "MC", "dançarino", "segurança", "vendedor de bebida", "fotógrafo", "quem foi só pra ver", "vizinho reclamando do som"}},
	{"🚌", "Ônibus", []string{"motorista", "cobrador", "fiscal", "estudante", "idoso", "passageiro em pé", "passageiro dormindo", "vendedor de bala"}},
	{"🏦", "Banco", []string{"gerente", "caixa", "segurança", "atendente", "cliente", "aposentado na fila", "assaltante", "faxineiro"}},
	{"🔒", "Presídio", []string{"carcereiro", "diretor do presídio", "preso", "preso novato", "advogado", "visitante", "médico", "cozinheiro"}},
	{"🚀", "Estação espacial", []string{"astronauta", "comandante", "cientista", "engenheiro", "médico", "piloto", "técnico de comunicação", "turista espacial"}},
	{"🐄", "Fazenda", []string{"fazendeiro", "peão", "veterinário", "tratorista", "cozinheira", "vendedor de gado", "turista do hotel fazenda", "criança da cidade"}},
	{"🎢", "Parque de diversões", []string{"operador da montanha-russa", "vendedor de algodão-doce", "mascote", "segurança", "visitante", "criança", "pai cansado", "adolescente com medo"}},
	{"💊", "Farmácia", []string{"farmacêutico", "balconista", "caixa", "gerente", "entregador", "representante de laboratório", "cliente gripado", "idosa medindo a pressão"}},
	{"🏨", "Hotel", []string{"recepcionista", "camareira", "mensageiro", "gerente", "cozinheiro", "segurança", "hóspede", "casal em lua de mel"}},
	{"🍣", "Restaurante japonês", []string{"sushiman", "garçom", "cozinheiro", "gerente", "caixa", "entregador", "cliente", "cliente que só come hot roll"}},
	{"🎭", "Teatro", []string{"ator", "diretor", "contrarregra", "iluminador", "figurinista", "bilheteiro", "crítico", "plateia"}},
	{"🏴‍☠️", "Navio pirata", []string{"capitão", "imediato", "marujo", "cozinheiro", "vigia do mastro", "grumete", "prisioneiro", "papagaio"}},
	{"🎓", "Universidade", []string{"professor", "reitor", "pesquisador", "bibliotecário", "calouro", "veterano", "segurança", "tia da cantina"}},
	{"⚰️", "Velório", []string{"viúva", "padre", "coveiro", "agente funerário", "amigo do falecido", "parente distante", "quem foi pelo café", "quem chora mais alto"}},
	{"🍖", "Churrascaria", []string{"churrasqueiro", "garçom do rodízio", "gerente", "caixa", "manobrista", "cliente", "aniversariante", "vegetariano arrependido"}},
	{"🎸", "Show de rock", []string{"vocalista", "guitarrista", "baterista", "roadie", "técnico de som", "segurança", "fã na grade", "vendedor de camiseta"}},
	{"🪖", "Base militar", []string{"sargento", "coronel", "recruta", "sentinela", "cozinheiro", "médico", "motorista", "capelão"}},
	{"🐶", "Pet shop", []string{"veterinário", "tosador", "banhista de cachorro", "atendente", "vendedor", "entregador", "dono de cachorro", "dona de gato"}},
	{"🚉", "Rodoviária", []string{"motorista", "vendedor de passagem", "segurança", "faxineiro", "vendedor de salgado", "taxista", "passageiro", "viajante perdido"}},
	{"🏛️", "Museu", []string{"guia", "curador", "restaurador", "segurança", "artista", "vendedor da lojinha", "turista", "estudante na excursão"}},
	{"🏢", "Escritório", []string{"chefe", "estagiário", "secretária", "pessoa do RH", "cara do TI", "vendedor", "faxineiro", "cliente"}},
	{"🚒", "Quartel dos bombeiros", []string{"bombeiro", "comandante", "motorista do caminhão", "socorrista", "telefonista", "mecânico", "estagiário", "vítima resgatada"}},
}

func (l espiaoLocation) label() string {
	return l.emoji + " " + l.name
}
