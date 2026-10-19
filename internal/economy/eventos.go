package economy

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const diaryLatest = 3

type seenEvent struct {
	id    string
	times int
}

type diaryShelf struct {
	name  string
	found int
	total int
}

func (b *bot) eventos(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := core.FirstMentionOrSender(evt).ToNonAD()
	acc := b.Account(ctx, person)
	seen, err := b.Wallets.seenEvents(ctx, chat, acc)
	if err != nil {
		return fmt.Errorf("ler o diário de eventos: %w", err)
	}
	_, waiting := b.scenes.pending.Load(eventKey(chat, acc))
	_, err = b.SendMentions(ctx, chat, diaryText(person, seen, waiting), []types.JID{person})
	return err
}

func diaryText(person types.JID, seen []seenEvent, waiting bool) string {
	found := map[string]bool{}
	lived := 0
	for _, s := range seen {
		found[s.id] = true
		lived += s.times
	}
	lines := []string{fmt.Sprintf("📖 *Diário de eventos de @%s*", person.User), ""}
	var discovered, total int
	for _, shelf := range diaryShelves(found) {
		lines = append(lines, fmt.Sprintf("%s: %d/%d", shelf.name, shelf.found, shelf.total))
		discovered += shelf.found
		total += shelf.total
	}
	lines = append(lines, "", fmt.Sprintf("🧭 Descobriu %d de %d eventos e já passou por %s", discovered, total, core.CountLabel(lived, "evento", "eventos")))
	if rare := rareTitles(seen); len(rare) > 0 {
		lines = append(lines, "🏆 Raros que já viveu: "+strings.Join(rare, ", "))
	}
	if latest := latestTitles(seen); len(latest) > 0 {
		lines = append(lines, "🕰️ Últimos: "+strings.Join(latest, ", "))
	}
	if waiting {
		lines = append(lines, "⏳ Tem um evento esperando a sua escolha aqui: vota na enquete!")
	}
	lines = append(lines, "", fmt.Sprintf("_Os eventos aparecem do nada em %d%% das vezes no !trabalhar, no %s e nos crimes, e cada escolha pode dar moeda, item, droga, ferimento, cadeia ou até morte_", percent(eventChance), extractionCommands()))
	return strings.Join(lines, "\n")
}

func diaryShelves(found map[string]bool) []diaryShelf {
	var shelves []diaryShelf
	for _, sc := range eventCatalog {
		if sc.stage {
			continue
		}
		i := slices.IndexFunc(shelves, func(s diaryShelf) bool { return s.name == sc.book })
		if i < 0 {
			shelves = append(shelves, diaryShelf{name: sc.book})
			i = len(shelves) - 1
		}
		shelves[i].total++
		if found[sc.id] {
			shelves[i].found++
		}
	}
	return shelves
}

func rareTitles(seen []seenEvent) []string {
	var titles []string
	for _, s := range seen {
		if sc, ok := scenarioByID(s.id); ok && sc.rarity > commonEvent {
			titles = append(titles, strings.TrimSpace(sc.rarity.badge())+" _"+sc.title+"_")
		}
	}
	return titles
}

func latestTitles(seen []seenEvent) []string {
	var titles []string
	for _, s := range seen {
		if sc, ok := scenarioByID(s.id); ok && len(titles) < diaryLatest {
			titles = append(titles, "_"+sc.title+"_")
		}
	}
	return titles
}

func (s *walletStore) seenEvents(ctx context.Context, chat types.JID, acc core.Account) (seen []seenEvent, err error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT evento, vezes FROM eventos_vistos WHERE chat = ? AND pessoa = ? ORDER BY visto_em DESC",
		chat.String(), acc.Key,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var e seenEvent
		if err := rows.Scan(&e.id, &e.times); err != nil {
			return nil, err
		}
		seen = append(seen, e)
	}
	return seen, rows.Err()
}

func (s *walletStore) eventPast(ctx context.Context, chat types.JID, acc core.Account, now time.Time) (past eventPast, err error) {
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) {
		past.recent, err = s.eventIDs(gctx,
			"SELECT evento FROM eventos_vistos WHERE chat = ? AND pessoa = ? ORDER BY visto_em DESC LIMIT ?",
			chat.String(), acc.Key, eventMemory)
		return err
	})
	g.Go(func() (err error) {
		past.planted, err = s.eventIDs(gctx,
			"SELECT evento FROM eventos_marcados WHERE chat = ? AND pessoa = ? AND a_partir <= ? ORDER BY a_partir",
			chat.String(), acc.Key, now.Unix())
		return err
	})
	if err := g.Wait(); err != nil {
		return eventPast{}, err
	}
	return past, nil
}

func (s *walletStore) eventIDs(ctx context.Context, query string, args ...any) (ids []string, err error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *walletStore) noteEvent(ctx context.Context, chat types.JID, acc core.Account, id string, now time.Time) error {
	return s.inTx(ctx, func(tx *ledgerTx) error {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO eventos_vistos (chat, pessoa, evento, vezes, visto_em) VALUES (?, ?, ?, 1, ?)
			 ON CONFLICT (chat, pessoa, evento) DO UPDATE SET vezes = eventos_vistos.vezes + 1, visto_em = excluded.visto_em`,
			chat.String(), acc.Key, id, now.Unix(),
		)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "DELETE FROM eventos_marcados WHERE chat = ? AND pessoa = ? AND evento = ?", chat.String(), acc.Key, id)
		return err
	})
}
