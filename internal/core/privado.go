package core

import (
	"context"
	"crypto/sha256"
	"slices"
	"strings"
	"sync"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

type PrivateAnswer func(ctx context.Context, evt *events.Message, text string) (done bool, err error)

type PrivatePrompt struct {
	keys     []string
	answer   PrivateAnswer
	commands []string

	mu    sync.Mutex
	polls map[types.MessageID]map[string]string
}

func (p *PrivatePrompt) WatchPoll(id types.MessageID, options []string) {
	hashes := make(map[string]string, len(options))
	for _, option := range options {
		hash := sha256.Sum256([]byte(option))
		hashes[string(hash[:])] = option
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.polls[id] = hashes
}

func (p *PrivatePrompt) pollOptions(id types.MessageID) (map[string]string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	options, watched := p.polls[id]
	return options, watched
}

func (b *Bot) AwaitPrivate(ids []types.JID, answer PrivateAnswer, commands ...string) (*PrivatePrompt, bool) {
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = id.ToNonAD().String()
	}
	prompt := &PrivatePrompt{keys: keys, answer: answer, commands: commands, polls: map[types.MessageID]map[string]string{}}
	for _, key := range keys {
		if _, taken := b.privatePrompts.LoadOrStore(key, prompt); taken {
			b.DropPrivate(prompt)
			return nil, false
		}
	}
	return prompt, true
}

func (b *Bot) DropPrivate(prompt *PrivatePrompt) {
	for _, key := range prompt.keys {
		b.privatePrompts.CompareAndDelete(key, prompt)
	}
}

func (b *Bot) answerPrivate(ctx context.Context, evt *events.Message) bool {
	chat := evt.Info.Chat
	if evt.Info.IsFromMe || (chat.Server != types.DefaultUserServer && chat.Server != types.HiddenUserServer) {
		return false
	}
	update := evt.Message.GetPollUpdateMessage()
	text := strings.TrimSpace(MessageText(evt.Message))
	if text == "" && update == nil {
		return false
	}
	prompt, pending := b.pendingPrivate(b.SenderIdentities(ctx, evt))
	if !pending {
		return false
	}
	if update != nil {
		return b.answerPrivatePoll(ctx, evt, prompt, update.GetPollCreationMessageKey().GetID())
	}
	name, isCommand := CommandName(text)
	if isCommand && !prompt.claims(name) {
		return false
	}
	b.runPrivateAnswer(ctx, evt, prompt, text)
	return true
}

func (b *Bot) answerPrivatePoll(ctx context.Context, evt *events.Message, prompt *PrivatePrompt, pollID types.MessageID) bool {
	options, watched := prompt.pollOptions(pollID)
	if !watched {
		return false
	}
	vote, err := b.Client.DecryptPollVote(ctx, evt)
	if err != nil {
		b.Client.Log.Errorf("ler voto no privado de %s: %v", evt.Info.Sender, err)
		return true
	}
	for _, hash := range vote.GetSelectedOptions() {
		if option, ok := options[string(hash)]; ok {
			b.runPrivateAnswer(ctx, evt, prompt, option)
			return true
		}
	}
	return true
}

func (b *Bot) runPrivateAnswer(ctx context.Context, evt *events.Message, prompt *PrivatePrompt, text string) {
	done, err := prompt.answer(ctx, evt, text)
	if err != nil {
		b.Client.Log.Errorf("responder no privado de %s: %v", evt.Info.Sender, err)
	}
	if done {
		b.DropPrivate(prompt)
	}
}

func (p *PrivatePrompt) claims(name string) bool {
	return slices.ContainsFunc(p.commands, func(command string) bool { return FoldLetters(command) == FoldLetters(name) })
}

func (b *Bot) pendingPrivate(ids []types.JID) (*PrivatePrompt, bool) {
	for _, id := range ids {
		if value, ok := b.privatePrompts.Load(id.ToNonAD().String()); ok {
			return value.(*PrivatePrompt), true
		}
	}
	return nil, false
}

func (b *Bot) SendPrivate(ctx context.Context, ids []types.JID, text string, mentions []types.JID) bool {
	inbox := privateInbox(ids)
	if _, err := b.SendRawMentions(ctx, inbox, text, mentions); err != nil {
		b.Client.Log.Errorf("mandar mensagem no privado de %s: %v", inbox, err)
		return false
	}
	return true
}

func (b *Bot) SendPrivatePoll(ctx context.Context, ids []types.JID, question string, options []string) (types.MessageID, bool) {
	inbox := privateInbox(ids)
	resp, err := b.Client.SendMessage(ctx, inbox, b.Client.BuildPollCreation(question, options, 1))
	if err != nil {
		b.Client.Log.Errorf("mandar enquete no privado de %s: %v", inbox, err)
		return "", false
	}
	return resp.ID, true
}

func privateInbox(ids []types.JID) types.JID {
	i := slices.IndexFunc(ids, func(id types.JID) bool { return id.Server == types.DefaultUserServer })
	if i < 0 {
		return ids[0].ToNonAD()
	}
	return ids[i].ToNonAD()
}

func (b *Bot) PrivateGroupLabel(ctx context.Context, chat types.JID) string {
	info, err := b.Client.GetGroupInfo(ctx, chat)
	if err != nil {
		b.Client.Log.Warnf("ler nome do grupo %s: %v", chat, err)
		return ""
	}
	if info.Name == "" {
		return ""
	}
	return " no grupo *" + info.Name + "*"
}
