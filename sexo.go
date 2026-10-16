package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func (b *bot) sexo(ctx context.Context, evt *events.Message) error {
	people, err := b.sexoParticipants(ctx, evt)
	if err != nil {
		return fmt.Errorf("listar participantes: %w", err)
	}
	if len(people) < 2 {
		return b.sendText(ctx, evt.Info.Chat, "Marca uma ou mais pessoas: !sexo @pessoa, !sexo @pessoa1 @pessoa2 @pessoa3... ou !sexo @todos")
	}

	scenes := []string{
		"🔥 %s sumiram juntos e a cama tá rangendo até agora 🛏️",
		"😏 %s se trancaram no quarto. Ninguém entra nas próximas horas 🔒",
		"🍑 %s fizeram o vizinho bater na parede três vezes 🧱",
		"🥵 %s se pegaram de jeito e o grupo inteiro ouviu 📢",
		"💦 %s embaçaram todos os vidros do carro 🚗",
		"🌶️ %s quebraram o estrado da cama. Quem paga o conserto? 🛠️",
		"🫦 %s apagaram a luz e pediram pra ninguém acender tão cedo 🕯️",
		"🍆 %s foram \"assistir um filme\" e ninguém sabe o nome do filme 🎬",
		"🌙 %s foram dormir cedo... e ninguém dormiu 🛌",
		"🚿 %s foram tomar banho juntos pra economizar água 💧",
		"🔊 %s colocaram música alta pra disfarçar o barulho 🎶",
		"🏨 %s fizeram check-in no motel sem bagagem nenhuma 🧳",
		"🍷 %s abriram um vinho e as roupas sumiram 👗",
		"👵 %s mandaram as crianças pra casa da vó 🧸",
		"📵 %s desligaram o celular e desapareceram do mapa 🙈",
		"🛋️ %s disseram que iam só descansar no sofá 😴",
	}
	news, err := b.cheatingNews(ctx, evt.Info.Chat, people)
	if err != nil {
		return fmt.Errorf("ler casamentos: %w", err)
	}
	scene := fmt.Sprintf(scenes[rand.IntN(len(scenes))], joinMentions(people))
	tops, bottoms := splitRoles(people)
	duration, verdict := sexoDuration()
	caption := fmt.Sprintf("%s\n\n%s\n%s\n⏱️ Durou %s\n%s%s",
		scene,
		roleLine("🍆", "Ativo", "Ativos", tops),
		roleLine("🍑", "Passivo", "Passivos", bottoms),
		formatDuration(duration),
		verdict,
		news.caption(),
	)

	video, err := b.uploadGIF(ctx, pickGIF(b.gifs["sexo"]))
	if err != nil {
		return fmt.Errorf("enviar gif: %w", err)
	}
	mentioned := make([]string, 0, len(people)+len(news.betrayed))
	for _, p := range slices.Concat(people, news.betrayed) {
		mentioned = append(mentioned, p.String())
	}
	video.Caption = proto.String(b.withNicknames(ctx, evt.Info.Chat, caption))
	video.ContextInfo = &waE2E.ContextInfo{MentionedJID: mentioned}

	_, err = b.client.SendMessage(ctx, evt.Info.Chat, &waE2E.Message{VideoMessage: video})
	return err
}

func sexoDuration() (time.Duration, string) {
	ranges := []struct {
		min, max time.Duration
		verdicts []string
	}{
		{10 * time.Second, 59 * time.Second, []string{
			"😂 Nem deu tempo de tirar a meia 🧦",
			"😂 Piscou, perdeu 👀",
			"😂 O micro-ondas demora mais que isso 🍿",
			"😂 Pediu desculpa e foi dormir 😴",
			"😂 Recorde mundial. Nem o Usain Bolt 🏃",
			"😂 Foi tão rápido que a cama nem percebeu 🛏️",
			"😂 O Wi-Fi demorou mais pra conectar 📶",
			"😂 Nem a música passou da introdução 🎵",
			"😂 Quem piscou achou que nem aconteceu 🫣",
			"😂 Durou menos que um áudio do zap 🎙️",
			"😂 Igual propaganda pulável: acabou antes do botão aparecer ⏭️",
			"😂 O vizinho foi bater na parede e já tinha acabado 🧱",
			"😂 Ainda tava tirando o relógio quando acabou ⌚",
			"😂 Precisa de um tutorial no YouTube urgente 📚",
			"😂 E ainda perguntou \"foi bom pra você?\" 🙃",
			"😂 Velozes e Furiosos ficou com inveja 🏎️",
			"😂 O lençol nem amassou 🛏️",
		}},
		{time.Minute, 5 * time.Minute, []string{
			"😅 O miojo nem ficou pronto 🍜",
			"😅 Durou menos que uma música no rádio 📻",
			"😅 Rapidinha de respeito ⚡",
			"😅 Deu tempo de ver três stories e só 📱",
			"😅 Já tava se vestindo antes de alguém perceber 👖",
			"😅 Foi no intervalo do jogo e ainda sobrou tempo ⚽",
			"😅 Mais rápido que fila de pão na padaria 🥖",
			"😅 Nem esquentou o lençol 🥶",
			"😅 Um banho demora mais que isso 🚿",
			"😅 Tipo café expresso: curto e forte ☕",
			"😅 Melhor que nada... né? 🤷",
			"😅 Pausa do almoço bem aproveitada 🍽️",
			"😅 Rápido, mas pelo menos foi sincero 🫡",
			"😅 Voltaram antes da pizza chegar 🍕",
			"😅 O cachorro nem teve tempo de latir na porta 🐕",
		}},
		{6 * time.Minute, 30 * time.Minute, []string{
			"😌 Honesto. Nota 7 ⭐",
			"😌 Dentro da média, ninguém reclamou 👍",
			"😌 Deu pro gasto 🙂",
			"😌 Nada de mais, nada de menos ⚖️",
			"😌 Tempo de um episódio de série de comédia 📺",
			"😌 Feijão com arroz bem feito 🍛",
			"😌 Ninguém pediu reembolso 💸",
			"😌 Aprovado pelo Inmetro ✅",
			"😌 Nem rápido demais, nem cansativo demais 😮‍💨",
			"😌 Clássico de domingo à tarde ☀️",
			"😌 Deu pra suar um pouquinho 💦",
			"😌 Avaliação no Google: 4 estrelas ⭐",
			"😌 Teve começo, meio e fim. Já é alguma coisa 📖",
			"😌 Teria durado mais se o celular não tivesse tocado 📱",
			"😌 Dava pra melhorar, mas tá valendo 👌",
		}},
		{31 * time.Minute, 2 * time.Hour, []string{
			"🔥 Os vizinhos já chamaram a polícia 🚓",
			"🔥 Precisaram de pausa pra hidratar 💧",
			"🔥 Agora sim, desempenho de atleta 🏅",
			"🔥 A cama pediu demissão 🛏️",
			"🔥 Durou mais que um filme inteiro 🎬",
			"🔥 O lençol foi direto pra lavanderia 🧺",
			"🔥 Até o gato saiu do quarto 🐈",
			"🔥 O vizinho de baixo pediu bis 👏",
			"🔥 Isso foi cardio de verdade 🏃",
			"🔥 Tiveram que abrir a janela pra ventilar 🪟",
			"🔥 O ventilador não deu conta 🌀",
			"🔥 Saíram do quarto andando torto 🦵",
			"🔥 A geladeira foi assaltada depois 🍫",
			"🔥 O síndico mandou uma advertência 📝",
			"🔥 Melhor que muita academia por aí 💪",
		}},
		{2*time.Hour + time.Minute, 6 * time.Hour, []string{
			"🏆 Isso não é sexo, é turno de trabalho 🏭",
			"🏆 Alguém chama uma ambulância 🚑",
			"🏆 Bateram o recorde do grupo 🥇",
			"🏆 Tiveram que pedir pizza no meio 🍕",
			"🏆 Fizeram intervalo igual jogo de futebol ⚽",
			"🏆 Merece entrar pro Guinness 📖",
			"🏆 Precisaram de dois carregadores de celular 🔋",
			"🏆 Já tão cobrando hora extra 💼",
			"🏆 O motel cobrou diária 🏨",
			"🏆 A NASA quer estudar esse caso 🚀",
			"🏆 Tiveram que fazer revezamento 🔄",
			"🏆 Alongamento antes era obrigatório 🧘",
			"🏆 Entraram de noite e saíram de dia 🌅",
			"🏆 A cama virou peça de museu 🏛️",
			"🏆 Alguém liga pro SAMU e pro fisioterapeuta 🩺",
		}},
	}
	r := ranges[rand.IntN(len(ranges))]
	return r.min + rand.N(r.max-r.min+1), r.verdicts[rand.IntN(len(r.verdicts))]
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return countLabel(int(d.Seconds()), "segundo", "segundos")
	}
	if d < time.Hour {
		return countLabel(int(d.Minutes()), "minuto", "minutos")
	}
	hours := countLabel(int(d.Hours()), "hora", "horas")
	minutes := int(d.Minutes()) % 60
	if minutes == 0 {
		return hours
	}
	return hours + " e " + countLabel(minutes, "minuto", "minutos")
}

func countLabel(n int, singular, plural string) string {
	if n == 1 {
		return "1 " + singular
	}
	return fmt.Sprintf("%d %s", n, plural)
}

func splitRoles(people []types.JID) ([]types.JID, []types.JID) {
	shuffled := slices.Clone(people)
	rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	cut := rand.IntN(len(shuffled)-1) + 1
	return shuffled[:cut], shuffled[cut:]
}

func roleLine(emoji, singular, plural string, people []types.JID) string {
	label := plural
	if len(people) == 1 {
		label = singular
	}
	return fmt.Sprintf("%s %s: %s", emoji, label, joinMentions(people))
}

func (b *bot) sexoParticipants(ctx context.Context, evt *events.Message) ([]types.JID, error) {
	if mentionsEveryone(evt.Message) {
		return b.groupMembers(ctx, evt.Info.Chat)
	}
	mentions := mentionedJIDs(evt.Message)
	if len(mentions) == 0 {
		return b.randomPartner(ctx, evt)
	}
	if len(mentions) == 1 {
		return uniqueJIDs([]types.JID{evt.Info.Sender.ToNonAD(), mentions[0]}), nil
	}
	return uniqueJIDs(mentions), nil
}

func (b *bot) randomPartner(ctx context.Context, evt *events.Message) ([]types.JID, error) {
	members, err := b.groupMembers(ctx, evt.Info.Chat)
	if err != nil {
		return nil, err
	}
	candidates := slices.DeleteFunc(members, func(m types.JID) bool {
		return m.User == evt.Info.Sender.User || m.User == evt.Info.SenderAlt.User
	})
	if len(candidates) == 0 {
		return nil, nil
	}
	partner := candidates[rand.IntN(len(candidates))]
	return []types.JID{evt.Info.Sender.ToNonAD(), partner}, nil
}

func mentionsEveryone(msg *waE2E.Message) bool {
	if msg.GetExtendedTextMessage().GetContextInfo().GetNonJIDMentions() > 0 {
		return true
	}
	everyone := map[string]bool{"@todos": true, "@all": true, "@everyone": true}
	for _, field := range strings.Fields(strings.ToLower(messageText(msg))) {
		if everyone[field] {
			return true
		}
	}
	return false
}

func (b *bot) groupMembers(ctx context.Context, chat types.JID) ([]types.JID, error) {
	if chat.Server != types.GroupServer {
		return nil, nil
	}
	info, err := b.client.GetGroupInfo(ctx, chat)
	if err != nil {
		return nil, err
	}
	self := map[string]bool{
		b.client.Store.GetJID().User: true,
		b.client.Store.GetLID().User: true,
	}
	members := make([]types.JID, 0, len(info.Participants))
	for _, p := range info.Participants {
		if self[p.PhoneNumber.User] || self[p.LID.User] || self[p.JID.User] {
			continue
		}
		members = append(members, p.JID)
	}
	return members, nil
}

func uniqueJIDs(jids []types.JID) []types.JID {
	seen := make(map[string]bool, len(jids))
	unique := make([]types.JID, 0, len(jids))
	for _, jid := range jids {
		if seen[jid.User] {
			continue
		}
		seen[jid.User] = true
		unique = append(unique, jid)
	}
	return unique
}

func joinMentions(people []types.JID) string {
	tags := make([]string, len(people))
	for i, p := range people {
		tags[i] = "@" + p.User
	}
	return joinWithAnd(tags)
}

func joinWithAnd(items []string) string {
	last := len(items) - 1
	if last == 0 {
		return items[0]
	}
	return strings.Join(items[:last], ", ") + " e " + items[last]
}
