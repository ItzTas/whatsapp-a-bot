package games

import (
	"context"
	"fmt"
	"math/rand/v2"
	"regexp"
	"strings"
	"unicode"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

var vddVerdades = []string{
	"Qual a coisa mais ousada que você já fez na cama?",
	"Já ficou com alguém deste grupo? Quem?",
	"Com quem deste grupo você ficaria hoje à noite?",
	"Qual o lugar mais inusitado onde você já transou?",
	"Já mandou nude? Pra quem foi o último?",
	"Qual a sua maior fantasia que você nunca contou pra ninguém?",
	"Já fingiu um orgasmo?",
	"Quem do grupo você acha mais gostoso(a)?",
	"Já teve um sonho picante com alguém daqui? Com quem?",
	"Qual foi a sua pior experiência na cama?",
	"Qual parte do corpo mais te atrai em alguém?",
	"Já foi pego(a) no flagra? Conta como foi.",
	"Qual foi a sua ficada mais vergonhosa?",
	"Quantas pessoas você já beijou numa noite só?",
	"Você prefere comandar ou ser comandado(a)?",
	"Já ficou com o ex de algum amigo(a)?",
	"Qual a mensagem mais safada que você já mandou?",
	"Já stalkeou alguém daqui de madrugada? Quem?",
	"Qual o seu fetiche mais estranho?",
	"Já transou com alguém que conheceu no mesmo dia?",
	"O que você faria se passasse 24 horas trancado(a) num quarto com alguém do grupo? Com quem?",
	"Já beijou alguém do mesmo sexo?",
	"Já mandou mensagem safada pra pessoa errada?",
	"Quanto tempo faz desde a última vez?",
	"Qual a música perfeita pra uma noite quente?",
	"Quem do grupo você acha que é ruim de cama?",
	"Quem daqui você bloquearia se pudesse, sem ninguém saber?",
	"Quem do grupo é o mais falso(a)?",
	"Qual amigo(a) daqui você acha que tem o pior gosto pra parceiro(a)?",
	"Quem do grupo você acha que ainda é virgem de beijo na boca?",
	"Você já falou mal de alguém deste grupo pelas costas? De quem?",
	"Quem daqui você acha que vai casar primeiro, e quem vai morrer sozinho(a)?",
	"Se tivesse que tirar uma pessoa do grupo, quem seria?",
	"Qual a pessoa do grupo mais chata de conversar?",
	"Quem do grupo você acha que manda nude pra todo mundo?",
	"Qual foi a maior mentira que você já contou pra alguém daqui?",
	"Você já ficou com alguém só por pena?",
	"Qual o seu maior red flag?",
	"Qual o nome mais vergonhoso que você tem salvo no celular?",
	"Você já chorou por alguém que não valia a pena? Quem?",
	"Qual o ex que você voltaria hoje se ele(a) chamasse?",
	"Quem do grupo você acha que tem o pior beijo?",
	"Você já mijou na piscina depois dos 15 anos?",
	"Qual foi a coisa mais nojenta que você já fez e ninguém sabe?",
	"Quanto tempo você fica sem tomar banho no fim de semana?",
}

var vddDesafios = []string{
	"Manda um áudio gemendo o seu próprio nome 🎙️",
	"Manda a última foto da sua galeria, sem trapacear 📸",
	"Manda \"tô com saudade do seu beijo\" pro seu ex e printa a resposta",
	"Descreve num áudio, em detalhes, o seu beijo perfeito",
	"Fala quem do grupo você pegaria e marca a pessoa",
	"Manda um áudio de 10 segundos com a sua voz mais sensual",
	"Deixa o grupo escolher a sua foto de perfil e fica com ela por 1 hora",
	"Manda print das últimas 5 pesquisas do seu navegador",
	"Manda print do perfil da última pessoa que você stalkeou",
	"Lê a última mensagem que você recebeu num áudio, com voz de filme adulto",
	"Faz uma declaração de amor bem melosa pra alguém do grupo",
	"Manda uma selfie fazendo a sua cara mais safada 😏",
	"Manda pro seu crush \"sonhei com você essa noite\" e printa a resposta",
	"Manda um áudio imitando um gemido de filme ruim",
	"Posta no status \"tô carente, alguém me chama\" e deixa lá por 30 minutos",
	"Manda pro seu crush \"o que você tá vestindo agora?\" e printa a resposta",
	"Descreve a pessoa mais atraente do grupo sem dizer o nome, e o grupo tenta adivinhar",
	"Escreve a cantada mais safada que você conhece pra última pessoa que mandou mensagem aqui",
	"Dá uma nota de 0 a 10 pra cada pessoa que falou no grupo hoje",
	"Conta a sua fantasia mais picante num áudio de 30 segundos",
	"Manda um áudio cantando Evidências pro seu crush e printa a reação",
	"Deixa o grupo escolher um emoji pra você mandar pro seu crush, sem explicar nada",
	"Fala a pessoa mais falsa do grupo e marca ela",
	"Marca a pessoa do grupo com o pior gosto musical",
	"Muda o seu nome no WhatsApp pra um que o grupo escolher, por 24 horas",
	"Manda print das suas 5 conversas mais recentes",
	"Manda um áudio pedindo desculpas pra última pessoa que você ignorou",
	"Manda um áudio imitando alguém do grupo, e o grupo tenta adivinhar quem é",
	"Manda pro seu pai ou mãe \"preciso te contar uma coisa\" e só explica daqui 10 minutos",
	"Manda um áudio cantando o hino nacional com voz de bêbado",
	"Manda a foto mais feia sua que você tiver na galeria",
	"Escreve um parágrafo romântico pra pessoa do grupo que você menos gosta",
	"Manda \"oi sumido(a)\" pro quinto contato da sua lista e printa a resposta",
	"Fala um defeito de cada pessoa que respondeu no grupo hoje",
	"Manda um áudio narrando o seu dia como se fosse um jogo de futebol",
	"Posta no status uma foto escolhida pelo grupo e deixa por 1 hora",
	"Manda um áudio contando o seu maior mico em menos de 30 segundos",
}

type vddOption struct {
	title   string
	prompts []string
}

const vddHeader = "🎲 *Verdade ou desafio?*"

var vddQuestionPattern = regexp.MustCompile(`^` + regexp.QuoteMeta(vddHeader) + `\s+@(\d+),`)

func (b *bot) vdd(ctx context.Context, evt *events.Message) error {
	target := evt.Info.Sender.ToNonAD()
	if mentions := core.MentionedJIDs(evt.Message); len(mentions) > 0 {
		target = mentions[0].ToNonAD()
	}

	text := fmt.Sprintf("%s\n\n@%s, responde essa mensagem com *verdade* ou *desafio* 👀", vddHeader, target.User)
	_, err := b.SendMentions(ctx, evt.Info.Chat, text, []types.JID{target})
	return err
}

func (b *bot) answerVdd(ctx context.Context, evt *events.Message) error {
	reply := evt.Message.GetExtendedTextMessage().GetContextInfo()
	questionID := reply.GetStanzaID()
	if questionID == "" || !b.isOwnUser(reply.GetParticipant()) {
		return nil
	}
	match := vddQuestionPattern.FindStringSubmatch(core.MessageText(reply.GetQuotedMessage()))
	if match == nil || !core.SentByUser(evt.Info, match[1]) {
		return nil
	}
	if _, answered := b.vddAnswered.Load(questionID); answered {
		return nil
	}

	chat := evt.Info.Chat
	target := evt.Info.Sender.ToNonAD()
	option, ok := vddChoice(core.MessageText(evt.Message))
	if !ok {
		_, err := b.SendMentions(ctx, chat, fmt.Sprintf("@%s, responde *verdade* ou *desafio*", target.User), []types.JID{target})
		return err
	}
	if _, answered := b.vddAnswered.LoadOrStore(questionID, struct{}{}); answered {
		return nil
	}

	prompt := option.prompts[rand.IntN(len(option.prompts))]
	_, err := b.SendMentions(ctx, chat, fmt.Sprintf("%s pra @%s:\n\n%s", option.title, target.User, prompt), []types.JID{target})
	return err
}

func (b *bot) isOwnUser(raw string) bool {
	jid, err := types.ParseJID(raw)
	if err != nil || jid.User == "" {
		return false
	}
	own := b.Client.Store
	return (own.ID != nil && jid.User == own.ID.User) || jid.User == own.LID.User
}

func vddChoice(text string) (vddOption, bool) {
	options := map[string]vddOption{
		"verdade": {title: "🗣️ *Verdade*", prompts: vddVerdades},
		"desafio": {title: "🔥 *Desafio*", prompts: vddDesafios},
	}
	words := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) })
	for _, word := range words {
		if option, ok := options[word]; ok {
			return option, true
		}
	}
	return vddOption{}, false
}
