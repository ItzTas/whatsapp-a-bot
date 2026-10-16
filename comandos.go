package main

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

const commandsMenuLifetime = time.Minute

func (b *bot) comandos(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat
	var (
		active []string
		owner  bool
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { active, err = b.modes.active(gctx, chat.ToNonAD()); return err })
	g.Go(func() error { owner = b.isOwner(gctx, evt); return nil })
	if err := g.Wait(); err != nil {
		return err
	}

	commands := slices.DeleteFunc(b.commands(), func(c botCommand) bool {
		return c.category.mode != "" && !slices.Contains(active, c.category.mode)
	})
	if owner {
		commands = slices.Concat(commands, b.ownerCommands())
	}
	resp, err := b.client.SendMessage(ctx, chat, &waE2E.Message{Conversation: proto.String(commandsMenu(commands))})
	if err != nil {
		return err
	}
	go b.revokeCommandsMenu(ctx, chat, resp.ID)
	return nil
}

func (b *bot) revokeCommandsMenu(ctx context.Context, chat types.JID, id types.MessageID) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(commandsMenuLifetime):
	}
	if _, err := b.client.SendMessage(ctx, chat, b.client.BuildRevoke(chat, types.EmptyJID, id)); err != nil {
		b.client.Log.Errorf("apagar menu de comandos: %v", err)
	}
}

func commandsMenu(commands []botCommand) string {
	var categories []string
	byCategory := map[string][]botCommand{}
	for _, c := range commands {
		if _, seen := byCategory[c.category.title]; !seen {
			categories = append(categories, c.category.title)
		}
		byCategory[c.category.title] = append(byCategory[c.category.title], c)
	}

	var sb strings.Builder
	sb.WriteString("🤖 *COMANDOS DO BOT*")
	for _, category := range categories {
		fmt.Fprintf(&sb, "\n\n━━━━━━━━━━━━━━\n*%s*\n━━━━━━━━━━━━━━", category)
		for _, c := range byCategory[category] {
			sb.WriteString("\n")
			for _, u := range c.help {
				fmt.Fprintf(&sb, "\n▸ `%s`\n     _%s_", u.syntax, u.description)
			}
		}
	}
	sb.WriteString("\n\n_Cada categoria só aparece quando o modo dela tá ligado nesta conversa (veja com !modos)_")
	sb.WriteString("\n_Esse menu some em 1 minuto_ ⏳")
	return sb.String()
}
