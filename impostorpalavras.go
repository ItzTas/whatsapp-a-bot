package main

type impostorCategory struct {
	name  string
	words []string
}

var impostorCategories = []impostorCategory{
	{"🍔 Comida", []string{
		"pizza", "hambúrguer", "coxinha", "brigadeiro", "feijoada", "lasanha", "sushi", "pipoca",
		"pastel", "açaí", "churrasco", "pão de queijo", "tapioca", "cachorro-quente", "sorvete",
	}},
	{"🐾 Bicho", []string{
		"cachorro", "gato", "capivara", "girafa", "elefante", "tubarão", "pinguim", "jacaré",
		"macaco", "cavalo", "galinha", "cobra", "papagaio", "tartaruga", "leão",
	}},
	{"📍 Lugar", []string{
		"praia", "escola", "hospital", "academia", "shopping", "aeroporto", "igreja", "padaria",
		"cemitério", "cinema", "farmácia", "motel", "rodoviária", "supermercado", "estádio",
	}},
	{"📦 Objeto", []string{
		"geladeira", "guarda-chuva", "celular", "escova de dente", "travesseiro", "ventilador",
		"controle remoto", "espelho", "chinelo", "mochila", "relógio", "tesoura", "óculos", "vassoura",
		"micro-ondas",
	}},
	{"👷 Profissão", []string{
		"médico", "professor", "bombeiro", "policial", "dentista", "padeiro", "motorista de aplicativo",
		"advogado", "astronauta", "palhaço", "cabeleireiro", "programador", "pedreiro", "garçom", "veterinário",
	}},
	{"⚽ Esporte", []string{
		"futebol", "vôlei", "basquete", "natação", "tênis", "boxe", "skate", "surfe",
		"judô", "xadrez", "ciclismo", "handebol", "golfe", "Fórmula 1", "MMA",
	}},
	{"🎬 Filme ou desenho", []string{
		"Titanic", "Shrek", "Toy Story", "O Rei Leão", "Harry Potter", "Homem-Aranha", "Bob Esponja",
		"Os Simpsons", "Frozen", "Procurando Nemo", "Vingadores", "Tropa de Elite", "Scooby-Doo",
		"Pokémon", "Turma da Mônica",
	}},
	{"🏷️ Marca", []string{
		"Coca-Cola", "McDonald's", "Nike", "Apple", "Samsung", "Nutella", "iFood", "Havaianas",
		"Netflix", "Adidas", "Nubank", "Guaraná Antarctica", "Lego", "Google", "Burger King",
	}},
	{"🎉 Coisa de festa", []string{
		"bolo", "balão", "vela de aniversário", "confete", "presente", "DJ", "karaokê", "pula-pula",
		"salgadinho", "fantasia", "parabéns pra você", "língua de sogra", "pista de dança", "convite", "docinho",
	}},
	{"🚗 Meio de transporte", []string{
		"avião", "ônibus", "metrô", "bicicleta", "moto", "carro", "navio", "helicóptero",
		"trem", "patinete", "caminhão", "foguete", "balão de ar quente", "táxi", "teleférico",
	}},
}
