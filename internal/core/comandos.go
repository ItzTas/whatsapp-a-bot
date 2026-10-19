package core

import (
	"cmp"
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

var (
	RomanceCategory    = Category{"💞 Romance", modeRomance}
	FamilyCategory     = Category{"👨‍👩‍👧 Família", modeFamily}
	GamesCategory      = Category{"🎲 Jogos", modeGames}
	CityCategory       = Category{"🌃 Cidade Dorme", ModeCity}
	MetersCategory     = Category{"📊 Medidores", modeMeters}
	StatsCategory      = Category{"📈 Estatísticas", modeStats}
	JokesCategory      = Category{"😂 Zoeira", ModeJokes}
	EconomyCategory    = Category{"💰 Economia", ModeEconomy}
	AICategory         = Category{"🧠 IA", modeAI}
	MeddlingCategory   = Category{"🧠 IA", ModeAIMeddling}
	ModerationCategory = Category{"🛡️ Moderação", ModeModeration}
	helpCategory       = Category{"ℹ️ Ajuda", ""}
	UtilitiesCategory  = Category{"🛠️ Utilidades", ModeUtilities}
	OwnerCategory      = Category{title: "🔒 Dono"}
)

const (
	commandsMenuDefaultLifetime = 10 * time.Minute
	commandsMenuMinLifetime     = 10 * time.Second
	commandsMenuMaxLifetime     = 24 * time.Hour
)

func (b *Bot) comandos(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat
	titles, rest := commandsMenuFilter(QuestionFrom(MessageText(evt.Message)))
	lifetime, ok := commandsMenuLifetime(rest)
	if !ok {
		return b.ReplyTo(ctx, evt, commandsMenuUsage(rest))
	}

	var (
		active []string
		owner  bool
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { active, err = b.Modes.active(gctx, chat.ToNonAD()); return err })
	g.Go(func() error { owner = b.IsOwner(gctx, evt); return nil })
	if err := g.Wait(); err != nil {
		return err
	}

	commands := slices.DeleteFunc(b.commands(), func(c Command) bool {
		return c.Category.Mode != "" && !modeActive(active, c.Category.Mode)
	})
	if owner {
		commands = slices.Concat(commands, b.OwnerCommands())
	}
	if len(titles) > 0 {
		commands = slices.DeleteFunc(commands, func(c Command) bool { return !titles[c.Category.title] })
	}
	if len(commands) == 0 {
		return b.ReplyTo(ctx, evt, "nenhuma dessas categorias tá liberada nesta conversa 😕 veja os modos ligados com !modos")
	}
	resp, err := b.Client.SendMessage(ctx, chat, &waE2E.Message{Conversation: proto.String(commandsMenu(commands, lifetime))})
	if err != nil {
		return err
	}
	go b.revokeCommandsMenu(ctx, chat, resp.ID, lifetime)
	return nil
}

func commandsMenuFilter(arg string) (map[string]bool, string) {
	categoryByWord := map[string]string{}
	for _, c := range menuCategories() {
		words := strings.Fields(FoldLetters(c.title))[1:]
		for _, w := range append(words, strings.Join(words, "")) {
			categoryByWord[w] = c.title
		}
	}
	titles := map[string]bool{}
	var rest []string
	for _, word := range strings.Fields(FoldLetters(arg)) {
		title, ok := categoryByWord[word]
		if !ok {
			rest = append(rest, word)
			continue
		}
		titles[title] = true
	}
	return titles, strings.Join(rest, " ")
}

func commandsMenuUsage(unknown string) string {
	var names []string
	for _, c := range menuCategories() {
		if c == OwnerCategory {
			continue
		}
		names = append(names, c.title)
	}
	return fmt.Sprintf("não entendi *%s* 🤔 manda uma categoria e/ou um tempo, tipo `!menu economia 5min`\n\n*Categorias:* %s\n*Tempo:* 30s, 5min, 2h...", unknown, strings.Join(names, ", "))
}

func commandsMenuLifetime(arg string) (time.Duration, bool) {
	if arg == "" {
		return commandsMenuDefaultLifetime, true
	}
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
		return 0, false
	}
	return min(max(time.Duration(n)*unit, commandsMenuMinLifetime), commandsMenuMaxLifetime), true
}

func (b *Bot) revokeCommandsMenu(ctx context.Context, chat types.JID, id types.MessageID, lifetime time.Duration) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(lifetime):
	}
	if _, err := b.Client.SendMessage(ctx, chat, b.Client.BuildRevoke(chat, types.EmptyJID, id)); err != nil {
		b.Client.Log.Errorf("apagar menu de comandos: %v", err)
	}
}

func menuCategories() []Category {
	return []Category{RomanceCategory, FamilyCategory, GamesCategory, CityCategory, MetersCategory, StatsCategory, JokesCategory, EconomyCategory, AICategory, ModerationCategory, helpCategory, UtilitiesCategory, OwnerCategory}
}

func menuRank(c Category) int {
	return slices.IndexFunc(menuCategories(), func(o Category) bool { return o.title == c.title })
}

func commandsMenu(commands []Command, lifetime time.Duration) string {
	slices.SortStableFunc(commands, func(x, y Command) int { return cmp.Compare(menuRank(x.Category), menuRank(y.Category)) })
	var categories []string
	byCategory := map[string][]Command{}
	for _, c := range commands {
		if _, seen := byCategory[c.Category.title]; !seen {
			categories = append(categories, c.Category.title)
		}
		byCategory[c.Category.title] = append(byCategory[c.Category.title], c)
	}

	var sb strings.Builder
	sb.WriteString("🤖 *COMANDOS DO BOT*")
	for _, category := range categories {
		fmt.Fprintf(&sb, "\n\n━━━━━━━━━━━━━━\n*%s*\n━━━━━━━━━━━━━━", category)
		for _, c := range byCategory[category] {
			sb.WriteString("\n")
			for _, u := range c.Help {
				fmt.Fprintf(&sb, "\n▸ `%s`\n     _%s_", u.Syntax, u.Description)
			}
		}
	}
	sb.WriteString("\n\n_Cada categoria só aparece quando o modo dela tá ligado nesta conversa (veja com !modos)_")
	fmt.Fprintf(&sb, "\n_Esse menu some em %s_ ⏳", FormatDuration(lifetime))
	return sb.String()
}

func (b *Bot) helpCommands() []Command {
	return []Command{
		{
			Name:     "modos",
			Category: helpCategory,
			Help:     []Usage{{"!modos", "mostra todos os modos e quais estão ligados nesta conversa 🧩"}},
			Run:      b.modos,
		},
		{
			Name:     "comandos",
			Aliases:  []string{"menu"},
			Category: helpCategory,
			Help: []Usage{
				{"!comandos [categoria] [tempo]", "mostra esta lista (ou só as categorias pedidas, ex.: !comandos economia jogos), que some depois do tempo escolhido (ex.: 30s, 5min, 2h; padrão 10 minutos)"},
				{"!menu [categoria] [tempo]", "mostra esta lista (ou só as categorias pedidas, ex.: !menu economia 5min), que some depois do tempo escolhido (ex.: 30s, 5min, 2h; padrão 10 minutos)"},
			},
			Run: b.comandos,
		},
	}
}
