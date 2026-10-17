package meters

import (
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
)

func (b *bot) metersCommands() []core.Command {
	return []core.Command{
		{
			Name:     "medidor",
			Category: core.MetersCategory,
			Help: []core.Usage{
				{"!medidor <coisa> @pessoa", "lê as mensagens da pessoa e diz de 0 a 100% o quanto ela é <coisa>"},
				{"!medidor <coisa>", "mede você"},
			},
			Run: b.medidor,
		},
		{
			Name:     "gado",
			Category: core.MetersCategory,
			Help:     []core.Usage{{"!gado @pessoa", "lê as mensagens da pessoa e diz o quanto ela é gado 🐂"}},
			Run:      b.meter(gadoMeter()),
		},
		{
			Name:     "corno",
			Category: core.MetersCategory,
			Help:     []core.Usage{{"!corno @pessoa", "lê as mensagens da pessoa e diz o quanto ela é corna 🦌"}},
			Run:      b.meter(cornoMeter()),
		},
		{
			Name:     "gay",
			Category: core.MetersCategory,
			Help:     []core.Usage{{"!gay @pessoa", "lê as mensagens da pessoa e diz o quanto ela é gay 🏳️‍🌈"}},
			Run:      b.meter(gayMeter()),
		},
		{
			Name:     "rankinggay",
			Aliases:  []string{"rankgay", "maisgay"},
			Category: core.MetersCategory,
			Help:     []core.Usage{{"!rankinggay", "mede todo mundo do grupo com o !gay e mostra quem é o mais gay 🏳️‍🌈"}},
			Run:      b.meterRanking(gayMeter()),
		},
		{
			Name:     "chato",
			Category: core.MetersCategory,
			Help:     []core.Usage{{"!chato @pessoa", "diz o quanto a pessoa é chata: fala demais, grita em caps, spamma comando, reclama de tudo 🙄"}},
			Run:      b.meter(chatoMeter()),
		},
		{
			Name:     "euanime",
			Category: core.MetersCategory,
			Help: []core.Usage{
				{"!euanime @pessoa", "diz qual personagem de anime a pessoa é, com foto 🎭 É sempre o mesmo pra mesma pessoa no grupo (quer um que combine de verdade? usa o !quemsou)"},
				{"!euanime", "descobre o seu"},
			},
			Run: b.euanime,
		},
	}
}
