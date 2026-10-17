package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/economy"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const trialMaxQuotes = 25

func (b *bot) processaria(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return b.SendText(ctx, chat, "⚖️ O !processaria só funciona em grupo")
	}
	if !b.Assistant.Enabled() {
		return b.SendText(ctx, chat, "🧠 A IA tá desligada, ninguém configurou um modelo")
	}
	mentions := core.MentionedJIDs(evt.Message)
	if len(mentions) == 0 {
		return b.ReplyTo(ctx, evt, "⚖️ Marca quem vai pro banco dos réus: !processaria @pessoa <motivo>")
	}
	defendant := mentions[0].ToNonAD()
	if defendant.User == evt.Info.Sender.User || defendant.User == evt.Info.SenderAlt.User {
		return b.ReplyTo(ctx, evt, "🤨 Não dá pra processar a si mesmo, procura um terapeuta")
	}
	if _, busy := b.economy.TrialChats.LoadOrStore(chat.String(), struct{}{}); busy {
		return b.ReplyTo(ctx, evt, economy.TrialBusy)
	}

	opened, err := b.fileMockTrial(ctx, evt, chat, defendant)
	if !opened {
		b.economy.TrialChats.Delete(chat.String())
	}
	return err
}

func (b *bot) fileMockTrial(ctx context.Context, evt *events.Message, chat, defendant types.JID) (bool, error) {
	motive := core.NicknameFrom(core.MessageText(evt.Message))
	script, failure := b.writeTrial(ctx, evt, defendant, motive)
	if failure != "" {
		return false, b.ReplyTo(ctx, evt, failure)
	}
	round := &economy.TrialCase{PollAnswer: core.NewPollAnswer(economy.TrialGuiltyOption), Accuser: evt.Info.Sender.ToNonAD(), Defendant: defendant, Script: script}
	if err := b.economy.OpenTrial(ctx, chat, round, motive); err != nil {
		return false, err
	}
	return true, nil
}

func (b *bot) writeTrial(ctx context.Context, evt *events.Message, defendant types.JID, motive string) (economy.TrialScript, string) {
	chat := evt.Info.Chat
	var (
		accuserName   string
		defendantName string
		texts         []string
		nicknames     map[string]string
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		nicknames = b.ChatNicknames(gctx, chat)
		accuserName = b.DisplayName(gctx, evt.Info.Sender.ToNonAD(), nicknames)
		return nil
	})
	g.Go(func() (err error) {
		defendantName, texts, err = b.personTexts(gctx, chat, defendant)
		return err
	})
	if err := g.Wait(); err != nil {
		b.Client.Log.Errorf("!processaria: %v", err)
		return economy.TrialScript{}, "😵 Não consegui ler os autos do processo, tenta de novo"
	}

	stopThinking := b.thinking(ctx, evt)
	tools := b.iaToolkit(ctx, evt, nicknames).randomTools()
	raw, err := b.Assistant.AskWithTools(ctx, b.Asker(ctx, chat, evt.Info.Sender.User), tools, func(contextSize int) string {
		return trialPrompt(accuserName, defendantName, motive, texts[:min(contextSize, trialMaxQuotes, len(texts))])
	})
	stopThinking()
	if err != nil {
		b.Client.Log.Warnf("!processaria: %v", err)
		return economy.TrialScript{}, iaFailure(err)
	}
	script, err := parseTrialScript(raw)
	if err != nil {
		b.Client.Log.Warnf("!processaria: julgamento inválido da IA: %v: %q", err, raw)
		return economy.TrialScript{}, "😵 A IA se enrolou nos autos do processo, tenta de novo"
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

func parseTrialScript(raw string) (economy.TrialScript, error) {
	start, end := strings.Index(raw, "{"), strings.LastIndex(raw, "}")
	if start < 0 || end < start {
		return economy.TrialScript{}, errors.New("sem JSON na resposta")
	}
	var s economy.TrialScript
	if err := json.Unmarshal([]byte(raw[start:end+1]), &s); err != nil {
		return economy.TrialScript{}, err
	}
	s.Hearing, s.Guilty, s.Innocent = stripTags(s.Hearing), stripTags(s.Guilty), stripTags(s.Innocent)
	if s.Hearing == "" || s.Guilty == "" || s.Innocent == "" {
		return economy.TrialScript{}, errors.New("faltou o julgamento ou uma das sentenças")
	}
	return s, nil
}
