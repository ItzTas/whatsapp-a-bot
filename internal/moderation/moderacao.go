package moderation

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) restricted(ctx context.Context, evt *events.Message, store *core.RestrictionStore) bool {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer || evt.Info.IsFromMe {
		return false
	}
	found, err := store.Has(ctx, chat, b.SenderIdentities(ctx, evt))
	if err != nil {
		b.Client.Log.Errorf("checar se %s tá em %s: %v", evt.Info.Sender, store.Table, err)
		return false
	}
	if !found {
		return false
	}
	on, err := b.Modes.Enabled(ctx, chat, core.ModeModeration)
	if err != nil {
		b.Client.Log.Errorf("checar modo %s em %s: %v", core.ModeModeration, chat, err)
		return false
	}
	return on
}

func (b *bot) deleteIfMuted(ctx context.Context, evt *events.Message) bool {
	if !b.restricted(ctx, evt, b.mutes) {
		return false
	}
	chat := evt.Info.Chat.ToNonAD()
	if revoked := evt.Message.GetProtocolMessage(); revoked.GetType() == waE2E.ProtocolMessage_REVOKE {
		b.markRevoked(chat, types.MessageID(revoked.GetKey().GetID()))
		return true
	}
	if evt.IsEdit || !core.Revocable(evt.Message) || b.markRevoked(chat, evt.Info.ID) {
		return true
	}
	b.revokeMuted(ctx, chat, evt)
	return true
}

type revokedMessage struct {
	chat types.JID
	id   types.MessageID
}

func (b *bot) markRevoked(chat types.JID, id types.MessageID) bool {
	key := revokedMessage{chat: chat, id: id}
	if _, already := b.revokedMessages.LoadOrStore(key, struct{}{}); already {
		return true
	}
	time.AfterFunc(10*time.Minute, func() { b.revokedMessages.Delete(key) })
	return false
}

func (b *bot) revokeMuted(ctx context.Context, chat types.JID, evt *events.Message) {
	revoke := b.Client.BuildRevoke(chat, evt.Info.Sender, evt.Info.ID)
	_, err := b.Client.SendMessage(ctx, chat, revoke)
	if errors.Is(err, whatsmeow.ErrMessageTimedOut) {
		_, err = b.Client.SendMessage(ctx, chat, revoke)
	}
	if err == nil {
		return
	}
	if errors.Is(err, whatsmeow.ErrServerReturnedError) {
		b.Client.Log.Infof("o WhatsApp não deixou apagar a mensagem %s de %s, que tá mutado (ela pode já ter sido apagada): %v", evt.Info.ID, evt.Info.Sender, err)
		return
	}
	b.Client.Log.Warnf("apagar mensagem %s de %s, que tá mutado: %v", evt.Info.ID, evt.Info.Sender, err)
}

type moderation struct {
	chat   types.JID
	group  *types.GroupInfo
	people []types.JID
}

func (b *bot) moderate(ctx context.Context, evt *events.Message, name, verb string, needsBotAdmin, ownerAllowed bool) (moderation, string, error) {
	m, problem, err := b.moderateGroup(ctx, evt, name, ownerAllowed)
	if err != nil || problem != "" {
		return moderation{}, problem, err
	}
	if needsBotAdmin && !b.botIsAdmin(m.group) {
		return moderation{}, fmt.Sprintf("Eu preciso ser admin do grupo pra %s alguém 😅", verb), nil
	}
	m.people = b.moderationTargets(evt)
	if len(m.people) == 0 {
		return moderation{}, fmt.Sprintf("Marca quem você quer %s: !%s @pessoa (ou responda uma mensagem da pessoa com !%s)", verb, name, name), nil
	}
	return m, "", nil
}

func (b *bot) moderateGroup(ctx context.Context, evt *events.Message, name string, ownerAllowed bool) (moderation, string, error) {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return moderation{}, fmt.Sprintf("O !%s só funciona em grupo", name), nil
	}
	group, err := b.Client.GetGroupInfo(ctx, chat)
	if err != nil {
		return moderation{}, "", err
	}
	sender, found := core.FindParticipant(group.Participants, core.UsersOf(b.SenderIdentities(ctx, evt)))
	ownerOverride := ownerAllowed && b.IsOwner(ctx, evt)
	if (!found || !core.IsGroupAdmin(sender)) && !ownerOverride {
		return moderation{}, fmt.Sprintf("Só admin do grupo pode usar o !%s 🚫", name), nil
	}
	return moderation{chat: chat, group: group}, "", nil
}

func (b *bot) botIsAdmin(group *types.GroupInfo) bool {
	self, found := core.FindParticipant(group.Participants, core.UsersOf(b.ownIdentities()))
	return found && core.IsGroupAdmin(self)
}

func (b *bot) moderationTargets(evt *events.Message) []types.JID {
	people := core.MentionedJIDs(evt.Message)
	replied := evt.Message.GetExtendedTextMessage().GetContextInfo().GetParticipant()
	if jid, err := types.ParseJID(replied); len(people) == 0 && replied != "" && err == nil {
		people = append(people, jid)
	}
	own := core.UsersOf(b.ownIdentities())
	targets := make([]types.JID, 0, len(people))
	for _, person := range people {
		if !slices.Contains(own, person.User) {
			targets = append(targets, person.ToNonAD())
		}
	}
	return core.UniqueJIDs(targets)
}

func (b *bot) ownIdentities() []types.JID {
	return withoutEmptyJIDs([]types.JID{b.Client.Store.GetJID(), b.Client.Store.GetLID()})
}

func withoutEmptyJIDs(ids []types.JID) []types.JID {
	return slices.DeleteFunc(ids, func(id types.JID) bool { return id.User == "" })
}

func (b *bot) memberIdentities(ctx context.Context, group *types.GroupInfo, person types.JID) []types.JID {
	ids := b.Identities(ctx, person)
	p, found := core.FindParticipant(group.Participants, core.UsersOf(ids))
	if !found {
		return ids
	}
	return participantIdentities(p)
}

func participantIdentities(p types.GroupParticipant) []types.JID {
	return core.UniqueJIDs(withoutEmptyJIDs([]types.JID{p.JID, p.LID, p.PhoneNumber}))
}

type participantAction struct {
	name    string
	verb    string
	change  whatsmeow.ParticipantChange
	skip    func(types.GroupParticipant) string
	done    string
	refused string
}

func (b *bot) ban(ctx context.Context, evt *events.Message) error {
	return b.changeParticipants(ctx, evt, banAction)
}

var banAction = participantAction{
	name:    "ban",
	verb:    "banir",
	change:  whatsmeow.ParticipantChangeRemove,
	skip:    func(types.GroupParticipant) string { return "" },
	done:    "🔨 @%s foi banido(a) do grupo",
	refused: "❌ Não consegui banir @%s, o WhatsApp não deixou (erro %d)",
}

func (b *bot) promover(ctx context.Context, evt *events.Message) error {
	return b.changeParticipants(ctx, evt, participantAction{
		name:   "promover",
		verb:   "promover",
		change: whatsmeow.ParticipantChangePromote,
		skip: func(p types.GroupParticipant) string {
			if core.IsGroupAdmin(p) {
				return "@%s já é admin"
			}
			return ""
		},
		done:    "⭐ @%s agora é admin do grupo",
		refused: "❌ Não consegui promover @%s, o WhatsApp não deixou (erro %d)",
	})
}

func (b *bot) rebaixar(ctx context.Context, evt *events.Message) error {
	return b.changeParticipants(ctx, evt, participantAction{
		name:   "rebaixar",
		verb:   "rebaixar",
		change: whatsmeow.ParticipantChangeDemote,
		skip: func(p types.GroupParticipant) string {
			if !core.IsGroupAdmin(p) {
				return "@%s nem é admin"
			}
			return ""
		},
		done:    "⬇️ @%s não é mais admin do grupo",
		refused: "❌ Não consegui tirar o admin de @%s, o WhatsApp não deixou (erro %d)",
	})
}

func (b *bot) changeParticipants(ctx context.Context, evt *events.Message, action participantAction) error {
	m, problem, err := b.moderate(ctx, evt, action.name, action.verb, true, false)
	if err != nil {
		return err
	}
	if problem != "" {
		return b.SendText(ctx, evt.Info.Chat, problem)
	}

	var lines []string
	mentions := slices.Clone(m.people)
	members := make([]types.JID, 0, len(m.people))
	for _, person := range m.people {
		p, found := core.FindParticipant(m.group.Participants, core.UsersOf(b.Identities(ctx, person)))
		if !found {
			lines = append(lines, fmt.Sprintf("@%s nem tá no grupo", person.User))
			continue
		}
		if reason := action.skip(p); reason != "" {
			lines = append(lines, fmt.Sprintf(reason, person.User))
			continue
		}
		members = append(members, p.JID)
	}
	if len(members) > 0 {
		results, err := b.Client.UpdateGroupParticipants(ctx, m.chat, members, action.change)
		if err != nil {
			return err
		}
		for _, r := range results {
			lines = append(lines, action.resultLine(r))
			mentions = append(mentions, r.JID)
		}
	}
	_, err = b.SendMentions(ctx, m.chat, strings.Join(lines, "\n"), mentions)
	return err
}

func (a participantAction) resultLine(r types.GroupParticipant) string {
	if r.Error != 0 {
		return fmt.Sprintf(a.refused, r.JID.User, r.Error)
	}
	return fmt.Sprintf(a.done, r.JID.User)
}

func (b *bot) mutar(ctx context.Context, evt *events.Message) error {
	if core.MentionsEveryone(evt.Message) {
		return b.muteEveryone(ctx, evt)
	}
	return b.changeRestrictions(ctx, evt, "mutar", true, false, b.mutes.Add, map[bool]string{
		true:  "🔇 @%s foi mutado(a): toda mensagem que mandar vai ser apagada",
		false: "@%s já tava mutado(a)",
	})
}

func (b *bot) desmutar(ctx context.Context, evt *events.Message) error {
	if core.MentionsEveryone(evt.Message) {
		return b.unmuteEveryone(ctx, evt)
	}
	return b.changeRestrictions(ctx, evt, "desmutar", false, false, b.mutes.Remove, map[bool]string{
		true:  "🔊 @%s foi desmutado(a) e já pode falar de novo",
		false: "@%s nem tava mutado(a)",
	})
}

func (b *bot) muteEveryone(ctx context.Context, evt *events.Message) error {
	m, problem, err := b.moderateGroup(ctx, evt, "mutar", false)
	if err != nil {
		return err
	}
	if problem == "" && !b.botIsAdmin(m.group) {
		problem = "Eu preciso ser admin do grupo pra mutar alguém 😅"
	}
	if problem != "" {
		return b.SendText(ctx, evt.Info.Chat, problem)
	}
	muted := 0
	for _, p := range m.group.Participants {
		if core.IsGroupAdmin(p) {
			continue
		}
		changed, err := b.mutes.Add(ctx, m.chat, participantIdentities(p))
		if err != nil {
			return err
		}
		if changed {
			muted++
		}
	}
	if muted == 0 {
		return b.SendText(ctx, m.chat, "Todo mundo que não é admin já tava mutado 🤷")
	}
	return b.SendText(ctx, m.chat, fmt.Sprintf("🔇 %s: toda mensagem de quem não é admin vai ser apagada (os admins ficaram de fora)", core.CountLabel(muted, "pessoa mutada", "pessoas mutadas")))
}

func (b *bot) unmuteEveryone(ctx context.Context, evt *events.Message) error {
	m, problem, err := b.moderateGroup(ctx, evt, "desmutar", false)
	if err != nil {
		return err
	}
	if problem != "" {
		return b.SendText(ctx, evt.Info.Chat, problem)
	}
	unmuted, err := b.mutes.Clear(ctx, m.chat)
	if err != nil {
		return err
	}
	if unmuted == 0 {
		return b.SendText(ctx, m.chat, "Ninguém tava mutado no grupo 🤷")
	}
	return b.SendText(ctx, m.chat, fmt.Sprintf("🔊 %s: todo mundo já pode falar de novo", core.CountLabel(int(unmuted), "pessoa desmutada", "pessoas desmutadas")))
}

func (b *bot) changeRestrictions(ctx context.Context, evt *events.Message, name string, needsBotAdmin, ownerAllowed bool, change func(context.Context, types.JID, []types.JID) (bool, error), results map[bool]string) error {
	m, problem, err := b.moderate(ctx, evt, name, name, needsBotAdmin, ownerAllowed)
	if err != nil {
		return err
	}
	if problem != "" {
		return b.SendText(ctx, evt.Info.Chat, problem)
	}
	lines := make([]string, len(m.people))
	for i, person := range m.people {
		changed, err := change(ctx, m.chat, b.memberIdentities(ctx, m.group, person))
		if err != nil {
			return err
		}
		lines[i] = fmt.Sprintf(results[changed], person.User)
	}
	_, err = b.SendMentions(ctx, m.chat, strings.Join(lines, "\n"), m.people)
	return err
}
