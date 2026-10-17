package city

import (
	"fmt"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
)

func (b *bot) cityCommands() []core.Command {
	city := core.CityCategory
	return []core.Command{
		{
			Name:     "cidadedorme",
			Aliases:  []string{"cidade"},
			Category: city,
			Help: []core.Usage{
				{"!cidadedorme", fmt.Sprintf("abre a sala do Cidade Dorme, ou entra nela 🌃 Precisa de %d a %d jogadores. Cada um recebe um papel secreto no privado (assassino, médico, detetive ou cidadão), e a partida alterna noite e dia até a cidade pegar todos os assassinos ou eles virarem maioria", cityMinPlayers, cityMaxPlayers)},
				{"!cidadedorme começar", "começa a partida antes do tempo, se já tiver gente suficiente"},
				{"!cidadedorme sair", "sai da sala antes da partida começar"},
				{"!cidadedorme cancelar", "cancela a partida (só quem abriu a sala)"},
				{"!cidadedorme", "com a partida rolando, mostra quem tá vivo, quem morreu e o papel de quem já saiu"},
			},
			Run: b.cidadedorme,
		},
		{
			Name:     "matar",
			Category: city,
			Help:     []core.Usage{{"!matar <número>", fmt.Sprintf("no meu privado, de noite: o assassino escolhe quem matar, tocando na enquete que eu mando ou pelo número (a noite dura %s). Com mais de um assassino, vale a pessoa mais escolhida", core.FormatWait(cityDurations[cityNight]))}},
			Run:      b.nightCommand,
		},
		{
			Name:     "salvar",
			Category: city,
			Help:     []core.Usage{{"!salvar <número>", "no meu privado, de noite: o médico escolhe quem salvar do assassino, na enquete ou pelo número (pode ser ele mesmo, mas não a mesma pessoa duas noites seguidas)"}},
			Run:      b.nightCommand,
		},
		{
			Name:     "investigar",
			Category: city,
			Help:     []core.Usage{{"!investigar <número>", "no meu privado, de noite: o detetive escolhe quem investigar, na enquete ou pelo número, e de manhã descobre se a pessoa é assassina"}},
			Run:      b.nightCommand,
		},
		{
			Name:     "votar",
			Category: city,
			Help: []core.Usage{
				{"!votar @pessoa", fmt.Sprintf("de dia, vota em quem a cidade deve eliminar (também vale o número da lista do !cidadedorme). Só vivos votam e dá pra trocar o voto. A votação dura %s, e o mais votado sai e tem o papel revelado; empate não elimina ninguém", core.FormatWait(cityDurations[cityDay]))},
				{"!votar ninguem", "vota pra cidade não eliminar ninguém hoje"},
			},
			Run: b.votar,
		},
	}
}
