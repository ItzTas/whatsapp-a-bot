package core

import (
	"context"
	"slices"

	"go.mau.fi/whatsmeow/types"
)

func IsGroupAdmin(p types.GroupParticipant) bool {
	return p.IsAdmin || p.IsSuperAdmin
}

func FindParticipant(participants []types.GroupParticipant, users []string) (types.GroupParticipant, bool) {
	i := slices.IndexFunc(participants, func(p types.GroupParticipant) bool {
		return ContainsAnyUser([]types.JID{p.JID, p.LID, p.PhoneNumber}, users)
	})
	if i < 0 {
		return types.GroupParticipant{}, false
	}
	return participants[i], true
}

type MemberOwners map[string]types.JID

func (b *Bot) MemberOwners(participants []types.GroupParticipant) MemberOwners {
	self := map[string]bool{
		b.Client.Store.GetJID().User: true,
		b.Client.Store.GetLID().User: true,
	}
	owners := MemberOwners{}
	for _, p := range participants {
		for _, id := range []types.JID{p.JID, p.LID, p.PhoneNumber} {
			if id.User == "" || self[id.User] {
				continue
			}
			owners[id.User] = p.JID
		}
	}
	return owners
}

func (o MemberOwners) Of(sender, senderAlt string) (types.JID, bool) {
	if owner, ok := o[sender]; ok {
		return owner, true
	}
	owner, ok := o[senderAlt]
	return owner, ok
}

func (b *Bot) GroupMembers(ctx context.Context, chat types.JID) ([]types.JID, error) {
	if chat.Server != types.GroupServer {
		return nil, nil
	}
	info, err := b.Client.GetGroupInfo(ctx, chat)
	if err != nil {
		return nil, err
	}
	self := map[string]bool{
		b.Client.Store.GetJID().User: true,
		b.Client.Store.GetLID().User: true,
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
