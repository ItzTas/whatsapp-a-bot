package core

import (
	"context"
	"regexp"
	"slices"
	"strings"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func UsersOf(jids []types.JID) []string {
	users := make([]string, len(jids))
	for i, j := range jids {
		users[i] = j.User
	}
	return users
}

func ContainsAnyUser(jids []types.JID, users []string) bool {
	return slices.ContainsFunc(jids, func(j types.JID) bool { return slices.Contains(users, j.User) })
}

func FirstMentionOrSender(evt *events.Message) types.JID {
	if mentions := MentionedJIDs(evt.Message); len(mentions) > 0 {
		return mentions[0]
	}
	return evt.Info.Sender.ToNonAD()
}

var MentionTagPattern = regexp.MustCompile(`@\d+`)

var vcardNumber = regexp.MustCompile(`waid=(\d+)`)

func QuotedContact(msg *waE2E.Message) (types.JID, bool) {
	quoted := msg.GetExtendedTextMessage().GetContextInfo().GetQuotedMessage()
	cards := []string{quoted.GetContactMessage().GetVcard()}
	for _, contact := range quoted.GetContactsArrayMessage().GetContacts() {
		cards = append(cards, contact.GetVcard())
	}
	for _, card := range cards {
		if found := vcardNumber.FindStringSubmatch(card); found != nil {
			return types.NewJID(found[1], types.DefaultUserServer), true
		}
	}
	return types.JID{}, false
}

func (b *Bot) SendMentions(ctx context.Context, chat types.JID, text string, mentions []types.JID) (types.MessageID, error) {
	return b.SendRawMentions(ctx, chat, b.WithNicknames(ctx, chat, text), mentions)
}

func (b *Bot) SendRawMentions(ctx context.Context, chat types.JID, text string, mentions []types.JID) (types.MessageID, error) {
	resp, err := b.Client.SendMessage(ctx, chat, MentionMessage(text, mentions))
	return resp.ID, err
}

func MentionMessage(text string, mentions []types.JID) *waE2E.Message {
	raw := make([]string, len(mentions))
	for i, m := range mentions {
		raw[i] = m.ToNonAD().String()
	}
	return &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text:        proto.String(text),
			ContextInfo: &waE2E.ContextInfo{MentionedJID: raw},
		},
	}
}

func StatsSubjectOf(evt *events.Message) (types.JID, bool) {
	if mentions := MentionedJIDs(evt.Message); len(mentions) > 0 {
		return mentions[0].ToNonAD(), true
	}
	if slices.Contains(NormalizedWords(MessageText(evt.Message)), "eu") {
		return evt.Info.Sender.ToNonAD(), true
	}
	return types.JID{}, false
}

func MentionsEveryone(msg *waE2E.Message) bool {
	if msg.GetExtendedTextMessage().GetContextInfo().GetNonJIDMentions() > 0 {
		return true
	}
	everyone := map[string]bool{"@todos": true, "@all": true, "@everyone": true}
	for _, field := range strings.Fields(strings.ToLower(MessageText(msg))) {
		if everyone[field] {
			return true
		}
	}
	return false
}

func UniqueJIDs(jids []types.JID) []types.JID {
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

func JoinMentions(people []types.JID) string {
	tags := make([]string, len(people))
	for i, p := range people {
		tags[i] = "@" + p.User
	}
	return JoinWithAnd(tags)
}

func SentBy(info types.MessageInfo, jid types.JID) bool {
	return info.Sender.ToNonAD() == jid || (!info.SenderAlt.IsEmpty() && info.SenderAlt.ToNonAD() == jid)
}

func SentByUser(info types.MessageInfo, user string) bool {
	return info.Sender.User == user || info.SenderAlt.User == user
}

func SentByAny(info types.MessageInfo, users []string) bool {
	return slices.ContainsFunc(users, func(user string) bool { return SentByUser(info, user) })
}

func (b *Bot) IsSelf(jid types.JID) bool {
	if jid.User == b.Client.Store.LID.User {
		return true
	}
	return b.Client.Store.ID != nil && jid.User == b.Client.Store.ID.User
}
