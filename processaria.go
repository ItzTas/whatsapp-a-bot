package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const trialMaxQuotes = 25

func (b *bot) processaria(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return b.sendText(ctx, chat, "⚖️ O !processaria só funciona em grupo")
	}
	if len(b.assistant.tiers) == 0 {
		return b.sendText(ctx, chat, "🧠 A IA tá desligada, ninguém configurou um modelo")
	}
	mentions := mentionedJIDs(evt.Message)
	if len(mentions) == 0 {
		return b.replyTo(ctx, evt, "⚖️ Marca quem vai pro banco dos réus: !processaria @pessoa <motivo>")
	}
	defendant := mentions[0].ToNonAD()
	if defendant.User == evt.Info.Sender.User || defendant.User == evt.Info.SenderAlt.User {
		return b.replyTo(ctx, evt, "🤨 Não dá pra processar a si mesmo, procura um terapeuta")
	}
	if _, busy := b.trialChats.LoadOrStore(chat.String(), struct{}{}); busy {
		return b.replyTo(ctx, evt, trialBusy)
	}

	opened, err := b.fileMockTrial(ctx, evt, chat, defendant)
	if !opened {
		b.trialChats.Delete(chat.String())
	}
	return err
}

func (b *bot) fileMockTrial(ctx context.Context, evt *events.Message, chat, defendant types.JID) (bool, error) {
	motive := nicknameFrom(messageText(evt.Message))
	script, failure := b.writeTrial(ctx, evt, defendant, motive)
	if failure != "" {
		return false, b.replyTo(ctx, evt, failure)
	}
	round := &trialCase{pollAnswer: newPollAnswer(trialGuiltyOption), accuser: evt.Info.Sender.ToNonAD(), defendant: defendant, script: script}
	if err := b.openTrial(ctx, chat, round, motive); err != nil {
		return false, err
	}
	return true, nil
}

func (b *bot) writeTrial(ctx context.Context, evt *events.Message, defendant types.JID, motive string) (trialScript, string) {
	chat := evt.Info.Chat
	var (
		accuserName   string
		defendantName string
		texts         []string
		nicknames     map[string]string
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		nicknames = b.chatNicknames(gctx, chat)
		accuserName = b.displayName(gctx, evt.Info.Sender.ToNonAD(), nicknames)
		return nil
	})
	g.Go(func() (err error) {
		defendantName, texts, err = b.personTexts(gctx, chat, defendant)
		return err
	})
	if err := g.Wait(); err != nil {
		b.client.Log.Errorf("!processaria: %v", err)
		return trialScript{}, "😵 Não consegui ler os autos do processo, tenta de novo"
	}

	stopThinking := b.thinking(ctx, evt)
	tools := b.iaToolkit(ctx, evt, nicknames).randomTools()
	raw, err := b.assistant.askWithTools(ctx, b.iaAsker(ctx, chat, evt.Info.Sender.User), tools, func(contextSize int) string {
		return trialPrompt(accuserName, defendantName, motive, texts[:min(contextSize, trialMaxQuotes, len(texts))])
	})
	stopThinking()
	if err != nil {
		b.client.Log.Warnf("!processaria: %v", err)
		return trialScript{}, iaFailure(err)
	}
	script, err := parseTrialScript(raw)
	if err != nil {
		b.client.Log.Warnf("!processaria: julgamento inválido da IA: %v: %q", err, raw)
		return trialScript{}, "😵 A IA se enrolou nos autos do processo, tenta de novo"
	}
	return script, ""
}

func trialPrompt(accuser, defendant, motive string, texts []string) string {
	if motive == "" {
		motive = "não informada, então invente um crime besta contra o grupo"
	}
	evidence := fmt.Sprintf("%s quase nunca fala no grupo, o que a promotoria vai usar como prova de que esconde algo.", defendant)
	if len(texts) > 0 {
		evidence = personPrompt(defendant, texts, "")
	}
	return fmt.Sprintf(`%s

Monta um julgamento zoeiro no tribunal do grupo.
Réu: %s. Quem abriu o processo: %s. Acusação: %s.
No campo "julgamento", escreve primeiro a fala da promotoria acusando o réu, citando entre aspas coisas que ele falou nas mensagens acima como prova, e depois a fala do advogado de defesa tentando salvar o réu com argumentos absurdos. Tom de tribunal de novela, com "meritíssimo", "data venia" e "protesto!". Um parágrafo curto pra cada lado, começando com *Promotoria:* e *Defesa:* em negrito.
Nos campos "culpado" e "inocente", escreve as duas sentenças possíveis do juiz, uma ou duas frases cada, com uma pena zoeira que não envolva dinheiro: uma pra se o júri condenar e outra pra se absolver.
Você tem ferramentas de sorte (rolar_dados, numero_aleatorio, cara_ou_coroa, sortear, embaralhar, sortear_membros). Use pelo menos uma pra decidir algo do julgamento na sorte, tipo o juiz rolando um d20 pra ver o humor dele, uma testemunha surpresa sorteada do grupo, os dias da pena ou a prova que a promotoria tira da cartola, e conta o resultado no texto. Nunca invente um resultado de sorte sem chamar a ferramenta.
É zoeira entre amigos: pode exagerar, mas nada de preconceito (raça, religião, orientação sexual, gênero, deficiência, corpo) nem acusação de crime de verdade.
Responda só com um JSON, sem texto antes ou depois e sem bloco de código, neste formato:
{"julgamento": "...", "culpado": "...", "inocente": "..."}
Use \n dentro das strings pra quebrar linha.`, evidence, defendant, accuser, motive)
}

func parseTrialScript(raw string) (trialScript, error) {
	start, end := strings.Index(raw, "{"), strings.LastIndex(raw, "}")
	if start < 0 || end < start {
		return trialScript{}, errors.New("sem JSON na resposta")
	}
	var s trialScript
	if err := json.Unmarshal([]byte(raw[start:end+1]), &s); err != nil {
		return trialScript{}, err
	}
	s.Hearing, s.Guilty, s.Innocent = stripTags(s.Hearing), stripTags(s.Guilty), stripTags(s.Innocent)
	if s.Hearing == "" || s.Guilty == "" || s.Innocent == "" {
		return trialScript{}, errors.New("faltou o julgamento ou uma das sentenças")
	}
	return s, nil
}
