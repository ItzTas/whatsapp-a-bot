package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"regexp"
	"slices"
	"strings"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

const bmcOptionCount = 3

type bmcRound struct {
	target  types.JID
	options []types.GroupParticipant
}

type bmcAction struct {
	stem   string
	verb   string
	result string
}

var bmcActions = []bmcAction{
	{stem: "beij", verb: "beija", result: "💋 Beija: @%s"},
	{stem: "mat", verb: "mata", result: "🔪 Mata: @%s"},
	{stem: "cas", verb: "casa", result: "💍 Casa: @%s"},
}

var (
	bmcAnswerPattern = regexp.MustCompile(`(?i)\b(beij|mat|cas)[ao]\b[^@]*?@(\d+)`)
	bmcVerbPattern   = regexp.MustCompile(`(?i)\b(beij|mat|cas)[ao]\b`)
	errBmcFormat     = errors.New("responde na ordem da lista (tipo: mato, beijo, caso) ou marcando: beija @pessoa mata @pessoa casa @pessoa")
)

func (b *bot) bmc(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat
	if chat.Server != types.GroupServer {
		return b.sendText(ctx, chat, "O !bmc só funciona em grupo")
	}

	group, err := b.client.GetGroupInfo(ctx, chat)
	if err != nil {
		return fmt.Errorf("ler grupo: %w", err)
	}

	target, excluded := b.bmcTarget(evt)
	round := &bmcRound{target: target.ToNonAD(), options: bmcOptions(group.Participants, excluded)}
	if len(round.options) < bmcOptionCount {
		return b.sendText(ctx, chat, "O grupo precisa ter pelo menos 3 pessoas além de quem vai responder")
	}

	id, err := b.sendMentions(ctx, chat, round.question(), append([]types.JID{round.target}, round.optionJIDs()...))
	if err != nil {
		return err
	}
	b.bmcRounds.Store(id, round)
	return nil
}

func (b *bot) answerBmc(ctx context.Context, evt *events.Message) error {
	questionID := evt.Message.GetExtendedTextMessage().GetContextInfo().GetStanzaID()
	if questionID == "" {
		return nil
	}
	value, ok := b.bmcRounds.Load(questionID)
	if !ok {
		return nil
	}
	round := value.(*bmcRound)
	if !sentBy(evt.Info, round.target) {
		return nil
	}

	chat := evt.Info.Chat
	chosen, err := round.choices(messageText(evt.Message))
	if err != nil {
		_, sendErr := b.sendMentions(ctx, chat, fmt.Sprintf("@%s, %s", round.target.User, err), []types.JID{round.target})
		return sendErr
	}
	if !b.bmcRounds.CompareAndDelete(questionID, round) {
		return nil
	}

	reactions := []string{"😂", "👀", "🔥", "😳", "💀", "🫣"}
	reaction := b.client.BuildReaction(chat, evt.Info.Sender, evt.Info.ID, reactions[rand.IntN(len(reactions))])

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error { _, err := b.client.SendMessage(gctx, chat, reaction); return err })
	g.Go(func() error {
		_, err := b.sendMentions(gctx, chat, round.result(chosen), append([]types.JID{round.target}, chosen...))
		return err
	})
	return g.Wait()
}

func sentBy(info types.MessageInfo, jid types.JID) bool {
	return info.Sender.ToNonAD() == jid || (!info.SenderAlt.IsEmpty() && info.SenderAlt.ToNonAD() == jid)
}

func (b *bot) bmcTarget(evt *events.Message) (types.JID, []types.JID) {
	excluded := []types.JID{b.client.Store.LID}
	if b.client.Store.ID != nil {
		excluded = append(excluded, *b.client.Store.ID)
	}
	if mentions := mentionedJIDs(evt.Message); len(mentions) > 0 {
		return mentions[0], append(excluded, mentions[0])
	}
	return evt.Info.Sender, append(excluded, evt.Info.Sender, evt.Info.SenderAlt)
}

func bmcOptions(participants []types.GroupParticipant, excluded []types.JID) []types.GroupParticipant {
	isExcluded := func(jid types.JID) bool {
		return !jid.IsEmpty() && slices.ContainsFunc(excluded, func(e types.JID) bool {
			return !e.IsEmpty() && e.ToNonAD() == jid.ToNonAD()
		})
	}

	options := slices.DeleteFunc(slices.Clone(participants), func(p types.GroupParticipant) bool {
		return isExcluded(p.JID) || isExcluded(p.LID) || isExcluded(p.PhoneNumber)
	})
	rand.Shuffle(len(options), func(i, j int) { options[i], options[j] = options[j], options[i] })
	return options[:min(len(options), bmcOptionCount)]
}

func (r *bmcRound) optionJIDs() []types.JID {
	jids := make([]types.JID, len(r.options))
	for i, p := range r.options {
		jids[i] = p.JID
	}
	return jids
}

func (r *bmcRound) question() string {
	lines := []string{
		"💋🔪💍 *Beija, mata ou casa?*",
		"",
		fmt.Sprintf("@%s, entre essas três pessoas, quem você beija, quem você mata e com quem você casa?", r.target.User),
		"",
	}
	for i, p := range r.options {
		lines = append(lines, fmt.Sprintf("%d. @%s", i+1, p.JID.User))
	}
	return strings.Join(append(lines, "", "Responde essa mensagem na ordem da lista 👀", "Tipo: mato, beijo, caso", "Ou marcando: beija @pessoa mata @pessoa casa @pessoa"), "\n")
}

func (r *bmcRound) choices(text string) ([]types.JID, error) {
	if !strings.Contains(text, "@") {
		return r.choicesInOrder(text)
	}

	picked := map[string][]string{}
	for _, m := range bmcAnswerPattern.FindAllStringSubmatch(text, -1) {
		stem := strings.ToLower(m[1])
		picked[stem] = append(picked[stem], m[2])
	}
	if len(picked) == 0 {
		return nil, errBmcFormat
	}

	chosen := make([]types.JID, 0, len(bmcActions))
	var missing []string
	for _, a := range bmcActions {
		users := picked[a.stem]
		if len(users) == 0 {
			missing = append(missing, a.verb)
			continue
		}
		if len(users) > 1 {
			return nil, errors.New("escolhe só uma pessoa pra cada: beija, mata e casa")
		}
		jid, ok := r.option(users[0])
		if !ok {
			return nil, errors.New("só vale escolher entre as 3 pessoas da pergunta")
		}
		if slices.Contains(chosen, jid) {
			return nil, errors.New("cada pessoa só pode ser escolhida uma vez")
		}
		chosen = append(chosen, jid)
	}
	if len(missing) > 0 {
		return nil, errors.New("faltou dizer quem você " + strings.Join(missing, " e "))
	}
	return chosen, nil
}

func (r *bmcRound) choicesInOrder(text string) ([]types.JID, error) {
	verbs := bmcVerbPattern.FindAllStringSubmatch(text, -1)
	if len(verbs) != len(r.options) {
		return nil, errBmcFormat
	}

	byStem := map[string]types.JID{}
	for i, m := range verbs {
		byStem[strings.ToLower(m[1])] = r.options[i].JID
	}
	if len(byStem) != len(bmcActions) {
		return nil, errors.New("cada um só vale uma vez: beija, mata e casa")
	}

	chosen := make([]types.JID, len(bmcActions))
	for i, a := range bmcActions {
		chosen[i] = byStem[a.stem]
	}
	return chosen, nil
}

func (r *bmcRound) option(user string) (types.JID, bool) {
	i := slices.IndexFunc(r.options, func(p types.GroupParticipant) bool {
		return user == p.JID.User || user == p.LID.User || user == p.PhoneNumber.User
	})
	if i < 0 {
		return types.JID{}, false
	}
	return r.options[i].JID, true
}

func (r *bmcRound) result(chosen []types.JID) string {
	lines := []string{fmt.Sprintf("*@%s escolheu!*", r.target.User), ""}
	for i, a := range bmcActions {
		lines = append(lines, fmt.Sprintf(a.result, chosen[i].User))
	}
	return strings.Join(lines, "\n")
}

func (b *bot) sendMentions(ctx context.Context, chat types.JID, text string, mentions []types.JID) (types.MessageID, error) {
	return b.sendRawMentions(ctx, chat, b.withNicknames(ctx, chat, text), mentions)
}

func (b *bot) sendRawMentions(ctx context.Context, chat types.JID, text string, mentions []types.JID) (types.MessageID, error) {
	raw := make([]string, len(mentions))
	for i, m := range mentions {
		raw[i] = m.ToNonAD().String()
	}
	resp, err := b.client.SendMessage(ctx, chat, &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text:        proto.String(text),
			ContextInfo: &waE2E.ContextInfo{MentionedJID: raw},
		},
	})
	return resp.ID, err
}
