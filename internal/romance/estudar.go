package romance

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) estudar(ctx context.Context, evt *events.Message) error {
	people, err := b.sexoParticipants(ctx, evt)
	if err != nil {
		return fmt.Errorf("listar participantes: %w", err)
	}
	if len(people) < 2 {
		return b.SendText(ctx, evt.Info.Chat, "Marca uma ou mais pessoas: !estudar @pessoa, !estudar @pessoa1 @pessoa2 @pessoa3... ou !estudar @todos")
	}

	scenes := []string{
		"📚 %s se trancaram no quarto pra \"revisar a matéria\" e ninguém viu caderno nenhum 🔒",
		"✏️ %s foram fazer trabalho em grupo e a parte prática ficou bem detalhada 😏",
		"🧪 %s foram pro laboratório fazer uma experiência de química 🔥",
		"📖 %s estudaram anatomia na prática, sem livro nenhum 🫦",
		"🏫 %s ficaram depois da aula pra tirar dúvida e as dúvidas eram outras 🍑",
		"🌙 %s viraram a noite estudando e a cama rangeu até de manhã 🛏️",
		"📝 %s marcaram uma aula particular com a porta trancada 🚪",
		"🎓 %s fizeram um intercâmbio de conhecimento no sofá 🛋️",
		"🔬 %s analisaram um ao outro no microscópio, centímetro por centímetro 👀",
		"📐 %s mediram tudo com régua e ainda pediram uma segunda medição 📏",
		"🧠 %s fizeram um grupo de estudos de dois, sem professor olhando 🤫",
		"📚 %s pegaram um livro na biblioteca e foram ler lá no fundo, onde não tem câmera 📹",
	}
	news, err := b.cheatingNews(ctx, evt.Info.Chat, people)
	if err != nil {
		return fmt.Errorf("ler casamentos: %w", err)
	}
	scene := fmt.Sprintf(scenes[rand.IntN(len(scenes))], core.JoinMentions(people))
	students, studied := splitRoles(people)
	duration, verdict := estudarDuration()
	caption := fmt.Sprintf("%s\n\n%s\n%s\n⏱️ Estudaram por %s\n%s%s",
		scene,
		roleLine("🤓", "Estudou", "Estudaram", students),
		roleLine("📖", "Foi estudado", "Foram estudados", studied),
		core.FormatDuration(duration),
		verdict,
		news.caption(),
	)
	return b.SendGIF(ctx, evt.Info.Chat, "sexo", caption, slices.Concat(people, news.betrayed))
}

func estudarDuration() (time.Duration, string) {
	ranges := []struct {
		min, max time.Duration
		verdicts []string
	}{
		{10 * time.Second, 59 * time.Second, []string{
			"😂 Leu o título e já fechou o livro 📕",
			"😂 Nem abriu o caderno direito 📓",
			"😂 Estudou igual véspera de prova: nada 🫠",
			"😂 Copiou a resposta do colega e foi embora 📋",
			"😂 O professor nem terminou a chamada ✋",
			"😂 Nota zero na prova prática 0️⃣",
			"😂 Durou menos que o resumo do resumo 📄",
		}},
		{time.Minute, 5 * time.Minute, []string{
			"😅 Estudou só o que cai na prova 🎯",
			"😅 Revisão de cinco minutos antes da aula ⏰",
			"😅 Passou raspando, com nota 5 📉",
			"😅 Leu só o primeiro capítulo 📖",
			"😅 Recuperação na certa 📝",
			"😅 Estudou no intervalo do recreio 🍎",
		}},
		{6 * time.Minute, 30 * time.Minute, []string{
			"😌 Aluno mediano. Nota 7 ⭐",
			"😌 Fez o dever de casa direitinho ✅",
			"😌 Passou de ano sem recuperação 🎒",
			"😌 Prestou atenção na aula, pelo menos 👀",
			"😌 O professor deu um \"bom trabalho\" no canto da folha 👍",
			"😌 Fez até anotação na margem do livro ✏️",
		}},
		{31 * time.Minute, 2 * time.Hour, []string{
			"🔥 Estudou a matéria inteira, do começo ao fim 📚",
			"🔥 Gabaritou a prova prática 💯",
			"🔥 O caderno ficou todo amassado 📓",
			"🔥 Precisaram de pausa pra hidratar 💧",
			"🔥 Saíram da biblioteca descabelados 💇",
			"🔥 A bibliotecária pediu silêncio três vezes 🤫",
		}},
		{2*time.Hour + time.Minute, 6 * time.Hour, []string{
			"🏆 Isso não é estudo, é mestrado 🎓",
			"🏆 Já dá pra defender a tese 📜",
			"🏆 Viraram a noite e ainda pediram mais matéria 🌅",
			"🏆 Ganharam bolsa integral 💰",
			"🏆 Viraram professores do assunto 👩‍🏫",
			"🏆 A faculdade quer publicar esse estudo 🔬",
		}},
	}
	r := ranges[rand.IntN(len(ranges))]
	return r.min + rand.N(r.max-r.min+1), r.verdicts[rand.IntN(len(r.verdicts))]
}
