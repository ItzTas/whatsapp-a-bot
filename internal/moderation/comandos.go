package moderation

import (
	"fmt"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
)

func (b *bot) moderationCommands() []core.Command {
	return []core.Command{
		{
			Name:     "ban",
			Aliases:  []string{"banir", "kick"},
			Category: core.ModerationCategory,
			Help: []core.Usage{
				{"!ban @pessoa", "tira a pessoa marcada do grupo 🔨 (só admin, e o bot também precisa ser admin)"},
				{"!ban", "responda uma mensagem com !ban e o bot bane quem mandou ela (também dá pra marcar várias pessoas)"},
			},
			Run: b.ban,
		},
		{
			Name:     "promover",
			Aliases:  []string{"promote", "daradm"},
			Category: core.ModerationCategory,
			Help: []core.Usage{
				{"!promover @pessoa", "deixa a pessoa marcada como admin do grupo ⭐ (só admin, e o bot também precisa ser admin)"},
				{"!promover", "responda uma mensagem com !promover e o bot promove quem mandou ela"},
			},
			Run: b.promover,
		},
		{
			Name:     "rebaixar",
			Aliases:  []string{"demote", "tiraradm"},
			Category: core.ModerationCategory,
			Help: []core.Usage{
				{"!rebaixar @pessoa", "tira o admin da pessoa marcada ⬇️ (só admin, e o bot também precisa ser admin; o dono do grupo não dá pra rebaixar)"},
				{"!rebaixar", "responda uma mensagem com !rebaixar e o bot tira o admin de quem mandou ela"},
			},
			Run: b.rebaixar,
		},
		{
			Name:     "mutar",
			Aliases:  []string{"mute", "silenciar"},
			Category: core.ModerationCategory,
			Help: []core.Usage{
				{"!mutar @pessoa", "muta a pessoa marcada: o bot apaga toda mensagem que ela mandar no grupo 🔇 (só admin, e o bot também precisa ser admin)"},
				{"!mutar @pessoa1 @pessoa2...", "muta todo mundo que foi marcado de uma vez"},
				{"!mutar @todos", "muta o grupo inteiro, menos os admins"},
				{"!mutar", "responda uma mensagem com !mutar e o bot muta quem mandou ela"},
			},
			Run: b.mutar,
		},
		{
			Name:     "desmutar",
			Aliases:  []string{"unmute"},
			Category: core.ModerationCategory,
			Help: []core.Usage{
				{"!desmutar @pessoa", "desmuta a pessoa e o bot para de apagar as mensagens dela 🔊 (dá pra marcar várias de uma vez; só admin)"},
				{"!desmutar @todos", "desmuta todo mundo que tá mutado no grupo"},
			},
			Run: b.desmutar,
		},
		{
			Name:     "advertir",
			Aliases:  []string{"advertencia", "warn"},
			Category: core.ModerationCategory,
			Help: []core.Usage{
				{"!advertir @pessoa <motivo>", fmt.Sprintf("dá uma advertência pra pessoa marcada ⚠️ (o motivo é opcional). Na %dª o bot bane ela do grupo e zera a ficha (só admin ou o dono do bot)", warningLimit)},
				{"!advertir", "responda uma mensagem com !advertir <motivo> e o bot adverte quem mandou ela"},
				{"!advertir limpar @pessoa", "zera todas as advertências da pessoa (!advertir perdoar @pessoa tira só a última)"},
			},
			Run: b.advertir,
		},
		{
			Name:     "advertencias",
			Aliases:  []string{"warns"},
			Category: core.ModerationCategory,
			Help: []core.Usage{
				{"!advertencias", "mostra quem tem advertência no grupo e quantas cada um tem 📋"},
				{"!advertencias @pessoa", "mostra as advertências da pessoa, com o motivo, quem deu e quando"},
			},
			Run: b.advertencias,
		},
		{
			Name:     "perdoar",
			Aliases:  []string{"unwarn"},
			Category: core.ModerationCategory,
			Help:     []core.Usage{{"!perdoar @pessoa", "tira a última advertência da pessoa 🕊️ (só admin ou o dono do bot)"}},
			Run:      b.perdoar,
		},
		{
			Name:     "boasvindas",
			Aliases:  []string{"bemvindo", "boasvinda"},
			Category: core.ModerationCategory,
			Help: []core.Usage{
				{"!boasvindas <texto>", "define a mensagem que o bot manda quando alguém entra no grupo 👋. Use {pessoa} pra marcar quem entrou e {grupo} pro nome do grupo; sem {pessoa}, o bot marca a pessoa no começo (só admin ou o dono do bot)"},
				{"!boasvindas", "mostra a mensagem de boas-vindas atual"},
				{"!boasvindas desligar", "apaga a mensagem e o bot para de dar boas-vindas"},
			},
			Run: b.boasvindas,
		},
	}
}
