package stats

import (
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
)

func (b *bot) statsCommands() []core.Command {
	return []core.Command{
		{
			Name:     "ranking",
			Category: core.StatsCategory,
			Help: []core.Usage{
				{"!ranking", "mostra quem mais mandou mensagem no grupo na última semana 🏆"},
				{"!ranking hoje", "o ranking das últimas 24 horas"},
				{"!ranking mês", "o ranking dos últimos 30 dias"},
				{"!ranking tudo", "o ranking dos " + core.MessageWindowLabel},
			},
			Run: b.ranking,
		},
		{
			Name:     "palavra",
			Category: core.StatsCategory,
			Help: []core.Usage{
				{"!palavra @pessoa", "mostra as palavras que a pessoa mais usa 🔤"},
				{"!palavra", "mostra as suas"},
			},
			Run: b.palavra,
		},
		{
			Name:     "tu",
			Category: core.StatsCategory,
			Help: []core.Usage{
				{"!tu", "mostra a sua ficha no grupo: nome, recado, ranking, palavra favorita, família e mais 🪪"},
				{"!tu @pessoa", "mostra a ficha da pessoa marcada"},
				{"!tu (com a economia ligada)", "a ficha também traz saldo, profissão, poupança, ações, mochila, dívidas, recompensa, cadeia e o resto da economia 💼"},
			},
			Run: b.tu,
		},
		{
			Name:     "palhaco",
			Category: core.StatsCategory,
			Help: []core.Usage{
				{"!palhaço", "mostra quem mais fez o grupo rir na última semana, contando os kkkk, ksksks e hahaha que a pessoa arrancou 🤡"},
				{"!palhaço hoje", "o ranking das últimas 24 horas (também tem mês e tudo)"},
			},
			Run: b.palhaco,
		},
		{
			Name:     "retrospectiva",
			Aliases:  []string{"wrapped", "retro"},
			Category: core.StatsCategory,
			Help: []core.Usage{
				{"!retrospectiva", "o wrapped do grupo no ano: quantas mensagens, quem mais falou, dia mais movimentado, horário de pico, palavra do ano e o maior palhaço 🎁"},
				{"!retrospectiva @pessoa", "o wrapped da pessoa: mensagens, lugar no ranking, dia e horário favoritos, palavra mais usada, parceria de papo, risadas arrancadas e a primeira mensagem do ano"},
				{"!retrospectiva eu", "o seu wrapped"},
				{"!retrospectiva 2025", "a retrospectiva de outro ano (também dá pra marcar alguém)"},
			},
			Run: b.retrospectiva,
		},
		{
			Name:     "primeiramsg",
			Aliases:  []string{"primeira", "primeiramensagem"},
			Category: core.StatsCategory,
			Help: []core.Usage{
				{"!primeiramsg @pessoa", "mostra a primeira mensagem da pessoa que eu tenho guardada no grupo, com a data 📜 (só guardo as dos " + core.MessageWindowLabel + ")"},
				{"!primeiramsg", "mostra a sua"},
			},
			Run: b.primeiramsg,
		},
		{
			Name:     "horario",
			Aliases:  []string{"horarios", "hora"},
			Category: core.StatsCategory,
			Help: []core.Usage{
				{"!horário", "mostra num gráfico em que hora do dia o grupo mais conversa na última semana ⏰"},
				{"!horário @pessoa", "o mesmo gráfico, só com as mensagens da pessoa (ou !horário eu pro seu)"},
				{"!horário hoje", "o gráfico das últimas 24 horas (também tem mês e tudo)"},
			},
			Run: b.horario,
		},
	}
}
