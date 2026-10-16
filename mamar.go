package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) mamar(ctx context.Context, evt *events.Message) error {
	people, err := b.sexoParticipants(ctx, evt)
	if err != nil {
		return fmt.Errorf("listar participantes: %w", err)
	}
	if len(people) < 2 {
		return b.sendText(ctx, evt.Info.Chat, "Marca uma ou mais pessoas: !mamar @pessoa, !mamar @pessoa1 @pessoa2 @pessoa3... ou !mamar @todos")
	}

	scenes := []string{
		"🍼 %s sumiram atrás da escada e só se ouviu um barulho de canudinho 🥤",
		"😏 %s foram pro banheiro da festa e a fila ficou enorme 🚽",
		"🍭 %s trataram o momento igual pirulito de festa junina 🎪",
		"🫦 %s se ajoelharam e não foi pra rezar 🙏",
		"🚗 %s pararam o carro num lugar escuro e o vidro embaçou 🌫️",
		"🍦 %s chuparam igual sorvete em dia de calor ☀️",
		"🎤 %s pegaram o microfone e não cantaram nenhuma música 🎶",
		"🌽 %s foram \"comer um milho\" e ninguém viu milho nenhum 👀",
		"🛋️ %s pediram licença e foram \"conversar\" no sofá 🤫",
		"🎬 %s apagaram a luz no meio do filme e ninguém viu o final 📺",
		"🏖️ %s foram pra trás da barraca na praia 🏝️",
		"🍬 %s desembrulharam a bala com muito carinho 🎁",
	}
	news, err := b.cheatingNews(ctx, evt.Info.Chat, people)
	if err != nil {
		return fmt.Errorf("ler casamentos: %w", err)
	}
	scene := fmt.Sprintf(scenes[rand.IntN(len(scenes))], joinMentions(people))
	givers, receivers := splitRoles(people)
	duration, verdict := mamarDuration()
	caption := fmt.Sprintf("%s\n\n%s\n%s\n⏱️ Durou %s\n%s%s",
		scene,
		roleLine("👄", "Mamou", "Mamaram", givers),
		roleLine("🍆", "Foi mamado", "Foram mamados", receivers),
		formatDuration(duration),
		verdict,
		news.caption(),
	)
	return b.sendGIF(ctx, evt.Info.Chat, "sexo", caption, slices.Concat(people, news.betrayed))
}

func mamarDuration() (time.Duration, string) {
	ranges := []struct {
		min, max time.Duration
		verdicts []string
	}{
		{5 * time.Second, 59 * time.Second, []string{
			"😂 Nem deu tempo de ajoelhar direito 🧎",
			"😂 Encostou e já acabou 🫠",
			"😂 Mais rápido que gole de cachaça 🥃",
			"😂 Foi só um bitoquinho e pronto 💋",
			"😂 O pirulito durou mais que isso 🍭",
			"😂 Nem precisou tirar a calça inteira 👖",
			"😂 Pediu desculpa e saiu de fininho 🫣",
		}},
		{time.Minute, 5 * time.Minute, []string{
			"😅 Rapidinha no intervalo do trabalho 💼",
			"😅 Durou o tempo de um áudio de 2 minutos 🎙️",
			"😅 Voltaram antes de alguém sentir falta 👀",
			"😅 Nem o gloss saiu da boca 💄",
			"😅 Tipo café expresso: curto e direto ☕",
			"😅 Foi no comercial da novela 📺",
		}},
		{6 * time.Minute, 20 * time.Minute, []string{
			"😌 Trabalho honesto. Nota 8 ⭐",
			"😌 Ninguém reclamou, ninguém pediu bis 👍",
			"😌 Deu pra sentir o maxilar no dia seguinte 🦷",
			"😌 Técnica de quem já fez curso 📚",
			"😌 Aprovado pelo Inmetro ✅",
			"😌 Teve até elogio no final 🥰",
		}},
		{21 * time.Minute, time.Hour, []string{
			"🔥 O maxilar pediu férias 🏖️",
			"🔥 Precisaram de uma pausa pra beber água 💧",
			"🔥 Saíram com o joelho ralado 🩹",
			"🔥 Dedicação de atleta olímpico 🏅",
			"🔥 A boca ficou dormente igual dentista 🦷",
			"🔥 O vizinho perguntou se tava tudo bem 🧱",
		}},
		{time.Hour + time.Minute, 3 * time.Hour, []string{
			"🏆 Isso não é mamada, é maratona 🏃",
			"🏆 Alguém chama um fisioterapeuta pro pescoço 🩺",
			"🏆 Bateram o recorde do grupo 🥇",
			"🏆 Ganharam uma bolsa de estudos em sucção 🎓",
			"🏆 O dentista vai querer estudar esse caso 🔬",
			"🏆 Tiveram que fazer revezamento 🔄",
		}},
	}
	r := ranges[rand.IntN(len(ranges))]
	return r.min + rand.N(r.max-r.min+1), r.verdicts[rand.IntN(len(r.verdicts))]
}
