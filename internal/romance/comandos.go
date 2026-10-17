package romance

import (
	"fmt"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/economy"
)

func (b *bot) romanceCommands() []core.Command {
	return []core.Command{
		{
			Name:     "casar",
			Category: core.RomanceCategory,
			Help: []core.Usage{
				{"!casar @pessoa", "casa você com a pessoa marcada"},
				{"!casar @pessoa1 @pessoa2", "casa as duas pessoas marcadas"},
				{"!casar @pessoa <dote>", fmt.Sprintf("com o modo consentimento ligado, vira um pedido de casamento 💍 A pessoa tem %s pra responder com !aceitar ou !recusar, e cada um só faz o próprio pedido. Com a economia ligada, quem pede paga a festa (%s, ou casa no cartório se não tiver), pode oferecer um dote, ganha uma !conjunta com o cônjuge e, se pedir divórcio depois, paga pensão", core.FormatWait(proposalWindow), economy.Coins(weddingPartyCost))},
			},
			Run: b.casar,
		},
		{
			Name:     "adotar",
			Category: core.FamilyCategory,
			Help:     []core.Usage{{"!adotar @pessoa", "você adota a pessoa marcada como filho(a) 🍼"}},
			Run:      b.adotar,
		},
		{
			Name:     "engravidar",
			Category: core.FamilyCategory,
			Help: []core.Usage{
				{"!engravidar @pessoa", "você engravida a pessoa marcada 🤰"},
				{"!engravidar @pessoa1 @pessoa2", "a primeira pessoa marcada engravida a segunda 🤰"},
			},
			Run: b.engravidar,
		},
		{
			Name:     "familia",
			Category: core.FamilyCategory,
			Help: []core.Usage{
				{"!familia", "mostra a sua árvore: cônjuges, pais, filhos, irmãos, avós e netos 🌳"},
				{"!familia @pessoa", "mostra a família da pessoa marcada"},
			},
			Run: b.familia,
		},
		{
			Name:     "ship",
			Category: core.RomanceCategory,
			Help: []core.Usage{
				{"!ship @pessoa", "diz de 0 a 100% o quanto você combina com a pessoa marcada e inventa o nome do casal 💘"},
				{"!ship @pessoa1 @pessoa2", "faz o ship das duas pessoas marcadas"},
			},
			Run: b.ship,
		},
		{
			Name:     "namoro",
			Aliases:  []string{"namorado", "namorada"},
			Category: core.RomanceCategory,
			Help: []core.Usage{
				{"!namorado", "mostra o seu namorado. Se você não namora ninguém pelo !namorar, mostra quem dá o maior !ship com você 💑"},
				{"!namorada", "igual ao !namorado, mas no feminino"},
				{"!namoro @pessoa", "mostra com quem a pessoa marcada namora (os três comandos aceitam @pessoa)"},
			},
			Run: b.namoro,
		},
		{
			Name:     "namorar",
			Category: core.RomanceCategory,
			Help: []core.Usage{
				{"!namorar @pessoa", "começa a namorar a pessoa marcada 💘 (dá pra namorar mais de uma pessoa)"},
				{"!namorar @pessoa1 @pessoa2", "faz as duas pessoas marcadas namorarem"},
				{"!namorar @pessoa <presente>", "com o modo consentimento ligado, vira um pedido de namoro 💘 A pessoa responde com !aceitar ou !recusar, e com a economia ligada dá pra mandar um presente em moedas junto"},
			},
			Run: b.namorar,
		},
		{
			Name:     "terminar",
			Aliases:  []string{"divorciar"},
			Category: core.RomanceCategory,
			Help: []core.Usage{
				{"!terminar @pessoa", "você termina o namoro, o casamento e o caso de amante com a pessoa marcada 💔"},
				{"!terminar @pessoa1 @pessoa2", "termina o namoro, o casamento e o caso de amante das duas pessoas marcadas (com o modo consentimento ligado, só se você for uma delas)"},
				{"!terminar", "você termina com todo mundo que namora, com quem casou e com quem é amante (!divorciar faz o mesmo)"},
			},
			Run: b.terminar,
		},
		{
			Name:     "almagemea",
			Aliases:  []string{"alma"},
			Category: core.RomanceCategory,
			Help: []core.Usage{
				{"!almagemea", "lê as mensagens do grupo e descobre quem escreve mais parecido com você, a sua alma gêmea 💞"},
				{"!almagemea @pessoa", "procura a alma gêmea da pessoa marcada"},
			},
			Run: b.almagemea,
		},
		{
			Name:     "sexo",
			Category: core.RomanceCategory,
			Help: []core.Usage{
				{"!sexo", "leva você e uma pessoa aleatória do grupo pro quarto 🔥"},
				{"!sexo @pessoa", "leva você e a pessoa marcada pro quarto 🔥"},
				{"!sexo @pessoa1 @pessoa2 @pessoa3...", "leva todo mundo que você marcar pro quarto 🔥"},
				{"!sexo @todos", "leva o grupo inteiro pro quarto 🔥"},
			},
			Run: b.sexo,
		},
		{
			Name:     "mamar",
			Category: core.RomanceCategory,
			Help: []core.Usage{
				{"!mamar", "você e uma pessoa aleatória do grupo vão pra um cantinho escuro 🍼"},
				{"!mamar @pessoa", "você e a pessoa marcada vão pra um cantinho escuro 🍼"},
				{"!mamar @pessoa1 @pessoa2 @pessoa3...", "leva todo mundo que você marcar pro cantinho escuro 🍼"},
				{"!mamar @todos", "leva o grupo inteiro pro cantinho escuro 🍼"},
			},
			Run: b.mamar,
		},
		{
			Name:     "estudar",
			Category: core.RomanceCategory,
			Help: []core.Usage{
				{"!estudar", "você estuda uma pessoa aleatória do grupo, de portas fechadas 📚"},
				{"!estudar @pessoa", "você estuda a pessoa marcada, de portas fechadas 📚"},
				{"!estudar @pessoa1 @pessoa2 @pessoa3...", "junta todo mundo que você marcar num grupo de estudos 📚"},
				{"!estudar @todos", "o grupo inteiro vira um grupo de estudos 📚"},
			},
			Run: b.estudar,
		},
		{
			Name:     "beijo",
			Aliases:  []string{"beijar"},
			Category: core.RomanceCategory,
			Help: []core.Usage{
				{"!beijo @pessoa", "você dá um beijo na pessoa marcada 💋 (!beijar faz o mesmo)"},
				{"!beijo @pessoa1 @pessoa2", "a primeira pessoa marcada beija a segunda 💋"},
			},
			Run: b.beijo,
		},
		{
			Name:     "abracar",
			Aliases:  []string{"abraco"},
			Category: core.RomanceCategory,
			Help: []core.Usage{
				{"!abracar @pessoa", "você dá um abraço na pessoa marcada 🤗 (!abraço faz o mesmo)"},
				{"!abracar @pessoa1 @pessoa2", "a primeira pessoa marcada abraça a segunda 🤗"},
			},
			Run: b.abracar,
		},
		{
			Name:     "cafune",
			Category: core.RomanceCategory,
			Help: []core.Usage{
				{"!cafune @pessoa", "você faz cafuné na pessoa marcada 🤲"},
				{"!cafune @pessoa1 @pessoa2", "a primeira pessoa marcada faz cafuné na segunda 🤲"},
			},
			Run: b.cafune,
		},
		{
			Name:     "trair",
			Category: core.RomanceCategory,
			Help: []core.Usage{
				{"!trair @pessoa", "você trai a pessoa marcada 🐍"},
				{"!trair @pessoa1 @pessoa2", "a primeira pessoa marcada trai a segunda"},
			},
			Run: b.trair,
		},
		{
			Name:     "traircom",
			Category: core.RomanceCategory,
			Help: []core.Usage{
				{"!traircom @pessoa", "você trai quem você namora ou casou com a pessoa marcada 🐍 e vocês dois viram amantes"},
				{"!traircom @pessoa1 @pessoa2", "a primeira pessoa marcada trai quem namora ou casou com a segunda"},
			},
			Run: b.traircom,
		},
		{
			Name:     "amante",
			Aliases:  []string{"amantes"},
			Category: core.RomanceCategory,
			Help: []core.Usage{
				{"!amante", "mostra os seus amantes, que aparecem quando alguém usa o !traircom 🤫"},
				{"!amante @pessoa", "mostra os amantes da pessoa marcada"},
			},
			Run: b.amante,
		},
		{
			Name:     "fic",
			Category: core.RomanceCategory,
			Help: []core.Usage{
				{"!fic", "escreve uma fic de amor entre você e uma pessoa aleatória do grupo 📖"},
				{"!fic @pessoa", "escreve uma fic de amor entre você e a pessoa marcada"},
				{"!fic @pessoa1 @pessoa2", "escreve uma fic de amor entre as duas pessoas marcadas"},
			},
			Run: b.fic,
		},
	}
}
