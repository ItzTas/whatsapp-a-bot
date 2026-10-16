package main

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

const (
	commandsMenuDefaultLifetime = 2 * time.Minute
	commandsMenuMinLifetime     = 10 * time.Second
	commandsMenuMaxLifetime     = 24 * time.Hour
)

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
		return c.category.mode != "" && !modeActive(active, c.category.mode)
	})
	if owner {
		commands = slices.Concat(commands, b.ownerCommands())
	}
	lifetime := commandsMenuLifetime(questionFrom(messageText(evt.Message)))
	resp, err := b.client.SendMessage(ctx, chat, &waE2E.Message{Conversation: proto.String(commandsMenu(commands, lifetime))})
	if err != nil {
		return err
	}
	go b.revokeCommandsMenu(ctx, chat, resp.ID, lifetime)
	return nil
}

func commandsMenuLifetime(arg string) time.Duration {
	units := map[string]time.Duration{
		"":         time.Minute,
		"s":        time.Second,
		"seg":      time.Second,
		"segundo":  time.Second,
		"segundos": time.Second,
		"m":        time.Minute,
		"min":      time.Minute,
		"minuto":   time.Minute,
		"minutos":  time.Minute,
		"h":        time.Hour,
		"hora":     time.Hour,
		"horas":    time.Hour,
	}
	arg = strings.ToLower(strings.Join(strings.Fields(arg), ""))
	digits := strings.TrimRightFunc(arg, unicode.IsLetter)
	unit, known := units[arg[len(digits):]]
	n, err := strconv.Atoi(digits)
	if !known || err != nil || n <= 0 {
		return commandsMenuDefaultLifetime
	}
	return min(max(time.Duration(n)*unit, commandsMenuMinLifetime), commandsMenuMaxLifetime)
}

func (b *bot) revokeCommandsMenu(ctx context.Context, chat types.JID, id types.MessageID, lifetime time.Duration) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(lifetime):
	}
	if _, err := b.client.SendMessage(ctx, chat, b.client.BuildRevoke(chat, types.EmptyJID, id)); err != nil {
		b.client.Log.Errorf("apagar menu de comandos: %v", err)
	}
}

func commandsMenu(commands []botCommand, lifetime time.Duration) string {
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
	fmt.Fprintf(&sb, "\n_Esse menu some em %s_ ⏳", formatDuration(lifetime))
	return sb.String()
}
