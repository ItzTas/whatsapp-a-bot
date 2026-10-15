package main

import (
	"context"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) comandos(ctx context.Context, evt *events.Message) error {
	return b.sendText(ctx, evt.Info.Chat, commandsMenu(b.commands()))
}

func commandsMenu(commands []botCommand) string {
	var categories []string
	byCategory := map[string][]botCommand{}
	for _, c := range commands {
		if _, seen := byCategory[c.category]; !seen {
			categories = append(categories, c.category)
		}
		byCategory[c.category] = append(byCategory[c.category], c)
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
	return sb.String()
}
