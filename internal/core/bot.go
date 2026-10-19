package core

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/llm"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

type Bot struct {
	Client         *whatsmeow.Client
	GIFs           map[string][]GIF
	privatePrompts sync.Map
	Messages       *MessageStore
	Nicknames      *labelStore
	Names          *labelStore
	Relationships  *relationshipStore
	Adoptions      *adoptionStore
	Assistant      *llm.Assistant
	filters        *filterStore
	Modes          *modeStore
	games          *gameStore
	versions       *versionStore
	update         *botUpdate
	gamesReady     chan struct{}
	owners         []string
	hooks          hooks
}

type Handler func(ctx context.Context, evt *events.Message) error

type Usage = struct {
	Syntax      string
	Description string
}

type Category struct {
	title string
	Mode  string
}

type Command struct {
	Name         string
	Aliases      []string
	Category     Category
	Help         []Usage
	Run          Handler
	BarredInJail bool
	WhileDead    bool
}

func (c Command) Matches(name string) bool {
	name = FoldLetters(name)
	return slices.ContainsFunc(append([]string{c.Name}, c.Aliases...), func(n string) bool { return FoldLetters(n) == name })
}

func (b *Bot) HandleMessage(ctx context.Context, evt *events.Message) {
	if !b.awaitGames(ctx) {
		return
	}
	if b.intercepted(ctx, evt) {
		return
	}
	name, isCommand := CommandName(MessageText(evt.Message))
	if isCommand && b.runOwnerCommand(ctx, evt, name) {
		return
	}
	if b.answerPrivate(ctx, evt) {
		return
	}
	allowed, err := b.allowed(ctx, evt)
	if err != nil {
		b.Client.Log.Errorf("checar blacklist/whitelist: %v", err)
		return
	}
	if !allowed {
		return
	}
	defer b.afterMessage(ctx, evt.Info.Chat.ToNonAD())
	if err := b.Messages.record(ctx, evt); err != nil {
		b.Client.Log.Errorf("guardar mensagem: %v", err)
	}
	b.reactToMessage(ctx, evt)
	b.observeMessage(ctx, evt)
	if !isCommand {
		b.answerModes(ctx, evt)
		return
	}
	if b.filteredCommand(ctx, evt, name) {
		return
	}
	matching := slices.DeleteFunc(b.commands(), func(c Command) bool { return !c.Matches(name) })
	if len(matching) == 0 {
		b.answerMissingCommand(ctx, evt, name)
		return
	}
	cmd, on, err := b.enabledCommand(ctx, evt.Info.Chat, matching)
	if err != nil {
		b.Client.Log.Errorf("checar modo do comando !%s: %v", name, err)
		return
	}
	if !on {
		return
	}
	if b.guarded(ctx, evt, cmd) {
		return
	}
	if err := cmd.Run(ctx, evt); err != nil {
		b.Client.Log.Errorf("comando !%s: %v", name, err)
	}
	b.commandRan(ctx, evt, cmd)
}

func (b *Bot) enabledCommand(ctx context.Context, chat types.JID, matching []Command) (Command, bool, error) {
	active, err := b.Modes.active(ctx, chat.ToNonAD())
	if err != nil {
		return Command{}, false, err
	}
	enabled := slices.DeleteFunc(matching, func(c Command) bool { return c.Category.Mode != "" && !modeActive(active, c.Category.Mode) })
	if len(enabled) == 0 {
		return Command{}, false, nil
	}
	if i := slices.IndexFunc(enabled, func(c Command) bool { return modeExclusive(c.Category.Mode) }); i >= 0 {
		return enabled[i], true, nil
	}
	return enabled[0], true, nil
}

func MessageText(msg *waE2E.Message) string {
	texts := []string{
		msg.GetConversation(),
		msg.GetExtendedTextMessage().GetText(),
		msg.GetImageMessage().GetCaption(),
		msg.GetVideoMessage().GetCaption(),
	}
	i := slices.IndexFunc(texts, func(text string) bool { return text != "" })
	if i < 0 {
		return ""
	}
	return texts[i]
}

func CommandName(text string) (string, bool) {
	fields := strings.Fields(text)
	if len(fields) == 0 || !strings.HasPrefix(fields[0], "!") {
		return "", false
	}
	return strings.ToLower(strings.TrimPrefix(fields[0], "!")), true
}

func MentionedJIDs(msg *waE2E.Message) []types.JID {
	raw := msg.GetExtendedTextMessage().GetContextInfo().GetMentionedJID()
	jids := make([]types.JID, 0, len(raw))
	for _, s := range raw {
		jid, err := types.ParseJID(s)
		if err != nil {
			continue
		}
		jids = append(jids, jid)
	}
	return jids
}

func (b *Bot) SendText(ctx context.Context, chat types.JID, text string) error {
	_, err := b.Client.SendMessage(ctx, chat, &waE2E.Message{Conversation: proto.String(text)})
	return err
}

func (b *Bot) SendPoll(ctx context.Context, chat types.JID, question string, options []string) error {
	_, err := b.Client.SendMessage(ctx, chat, b.Client.BuildPollCreation(question, options, 1))
	return err
}

func New(client *whatsmeow.Client, gifs map[string][]GIF, messages *MessageStore, assistant *llm.Assistant) *Bot {
	b := &Bot{
		Client:        client,
		GIFs:          gifs,
		Messages:      messages,
		Nicknames:     &labelStore{db: messages.DB, table: "apelidos", column: "apelido"},
		Names:         &labelStore{db: messages.DB, table: "nomes", column: "nome"},
		Relationships: &relationshipStore{db: messages.DB},
		Adoptions:     &adoptionStore{db: messages.DB},
		Assistant:     assistant,
		filters:       &filterStore{db: messages.DB},
		Modes:         &modeStore{db: messages.DB},
		games:         &gameStore{db: messages.DB},
		versions:      &versionStore{db: messages.DB},
		gamesReady:    make(chan struct{}),
		owners:        OwnersFromEnv(),
	}
	if len(b.owners) == 0 {
		client.Log.Warnf("BOT_DONO não definido: ninguém consegue usar !modo, !blacklist e !whitelist")
	}
	b.registerCore()
	return b
}
