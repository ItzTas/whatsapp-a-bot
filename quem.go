package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"unicode/utf8"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	maxPollOptions     = 12
	maxQuemQuestionLen = 200
)

var quemPerguntas = []string{
	"Quem é mais provável de casar primeiro?",
	"Quem é mais provável de ser preso(a)?",
	"Quem é mais provável de ficar rico(a)?",
	"Quem é mais provável de voltar com o ex?",
	"Quem é mais provável de morrer primeiro num filme de terror?",
	"Quem é mais provável de mandar mensagem bêbado(a) pro ex?",
	"Quem é mais provável de ter um perfil fake?",
	"Quem é mais provável de chorar vendo filme da Disney?",
	"Quem é mais provável de sumir do grupo por uma semana?",
	"Quem é mais provável de chegar atrasado(a) no próprio casamento?",
	"Quem é mais provável de virar coach?",
	"Quem é mais provável de ficar com alguém daqui?",
	"Quem é mais provável de ser cancelado(a) na internet?",
	"Quem é mais provável de esquecer o aniversário do(a) namorado(a)?",
	"Quem é mais provável de gastar o salário todo no primeiro dia?",
	"Quem é mais provável de mandar áudio de 5 minutos?",
	"Quem é mais provável de stalkear o crush de madrugada?",
	"Quem é mais provável de ganhar no BBB?",
	"Quem é mais provável de dar perdido depois do primeiro encontro?",
	"Quem é mais provável de dormir na festa?",
}

func (b *bot) quem(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return b.sendText(ctx, chat, "O !quem só funciona em grupo")
	}
	question, ok := quemQuestion(messageText(evt.Message))
	if !ok {
		return b.sendText(ctx, chat, fmt.Sprintf("Pergunta grande demais, o máximo é %d letras", maxQuemQuestionLen))
	}

	members, err := b.groupMembers(ctx, chat)
	if err != nil {
		return fmt.Errorf("ler grupo: %w", err)
	}
	if len(members) < 2 {
		return b.sendText(ctx, chat, "O grupo precisa ter pelo menos 2 pessoas além do bot")
	}
	rand.Shuffle(len(members), func(i, j int) { members[i], members[j] = members[j], members[i] })
	members = members[:min(len(members), maxPollOptions)]

	return b.sendPoll(ctx, chat, "🤔 "+question, b.pollNames(ctx, chat, members))
}

func quemQuestion(text string) (string, bool) {
	fields := strings.Fields(text)
	if len(fields) < 2 {
		return quemPerguntas[rand.IntN(len(quemPerguntas))], true
	}
	question := "Quem " + strings.TrimRight(strings.Join(fields[1:], " "), "?") + "?"
	return question, utf8.RuneCountInString(question) <= maxQuemQuestionLen
}

func (b *bot) pollNames(ctx context.Context, chat types.JID, people []types.JID) []string {
	nicknames := b.chatNicknames(ctx, chat)
	names := make([]string, len(people))
	var wg sync.WaitGroup
	for i, p := range people {
		wg.Go(func() { names[i] = b.displayName(ctx, p, nicknames) })
	}
	wg.Wait()

	seen := map[string]int{}
	for i, name := range names {
		seen[name]++
		if seen[name] > 1 {
			names[i] = fmt.Sprintf("%s (%d)", name, seen[name])
		}
	}
	return names
}
