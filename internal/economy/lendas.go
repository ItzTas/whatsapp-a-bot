package economy

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	legendPlace = "!usar"
	legendPerk  = "guarda uma lenda 🪔 o primeiro !usar da economia toda acorda um evento que nunca mais se repete pra ninguém. Fora isso, dá pra !usar quantas vezes quiser, não gasta"
)

func legendOf(id string) (listedScenario, bool) {
	i := slices.IndexFunc(eventCatalog, func(sc listedScenario) bool { return sc.item == id })
	if i < 0 {
		return listedScenario{}, false
	}
	return eventCatalog[i], true
}

func (b *bot) awaken(ctx context.Context, evt *events.Message, it item) (bool, error) {
	sc, ok := legendOf(it.id)
	if !ok {
		return false, nil
	}
	chat := evt.Info.Chat.ToNonAD()
	acc := b.Account(ctx, evt.Info.Sender)
	if _, busy := b.scenes.pending.Load(eventKey(chat, acc)); busy {
		return false, nil
	}
	woke, err := b.Wallets.claimLegend(ctx, chat, acc, sc.id, time.Now())
	if err != nil || !woke {
		return false, err
	}
	shown, err := b.tellLegend(ctx, evt, acc, sc)
	if err != nil || !shown {
		return shown, errors.Join(err, b.Wallets.forgetLegend(ctx, chat, acc, sc.id))
	}
	return true, nil
}

func (b *bot) tellLegend(ctx context.Context, evt *events.Message, acc core.Account, sc listedScenario) (bool, error) {
	chat := evt.Info.Chat.ToNonAD()
	c, _, err := b.eventCaseOf(ctx, chat, acc, eventSpot{place: legendPlace, won: true})
	if err != nil {
		return false, err
	}
	ev := newPendingEvent(chat, acc, b.SenderIdentities(ctx, evt), sc, c)
	if _, taken := b.scenes.pending.LoadOrStore(ev.key, ev); taken {
		return false, nil
	}
	return true, b.unveil(ctx, ev, sc, c.now)
}

func sleepingLegends(woken []string) string {
	var sleeping []string
	for _, sc := range eventCatalog {
		if it, ok := itemByID(sc.item); ok && !slices.Contains(woken, sc.id) {
			sleeping = append(sleeping, it.label())
		}
	}
	if len(sleeping) == 0 {
		return "🪔 Todas as lendas da economia já acordaram"
	}
	return "🪔 Lendas dormindo na economia: " + strings.Join(sleeping, ", ") + ". O primeiro !usar de cada uma acorda um evento que nunca mais se repete"
}

func (s *walletStore) claimLegend(ctx context.Context, chat types.JID, acc core.Account, id string, now time.Time) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO lendas (chat, evento, pessoa, acordada_em) VALUES (?, ?, ?, ?) ON CONFLICT (chat, evento) DO NOTHING",
		chat.String(), id, acc.Key, now.Unix(),
	)
	if err != nil {
		return false, err
	}
	claimed, err := res.RowsAffected()
	return claimed > 0, err
}

func (s *walletStore) forgetLegend(ctx context.Context, chat types.JID, acc core.Account, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM lendas WHERE chat = ? AND evento = ? AND pessoa = ?", chat.String(), id, acc.Key)
	return err
}

func (s *walletStore) wokenLegends(ctx context.Context, chat types.JID) ([]string, error) {
	return s.eventIDs(ctx, "SELECT evento FROM lendas WHERE chat = ?", chat.String())
}
