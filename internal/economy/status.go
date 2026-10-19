package economy

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

type ailments struct {
	dead    bool
	jailed  bool
	hostage bool
	clown   bool
	hexed   bool
	charms  string
	drugged bool
	health  int64
}

func (a ailments) curseEmoji() string {
	for id := range strings.SplitSeq(a.charms, ",") {
		if c, ok := charmByID(id); ok && c.curse {
			return c.emoji
		}
	}
	if a.hexed {
		return "🧿"
	}
	return ""
}

func (a ailments) emoji() string {
	curse := a.curseEmoji()
	statuses := []struct {
		on    bool
		emoji string
	}{
		{a.dead, "⚰️"},
		{a.jailed, "🔒"},
		{a.hostage, "🪢"},
		{a.clown, "🤡"},
		{curse != "", curse},
		{a.drugged, "😵‍💫"},
		{a.health <= MaxHealth/3, "🤕"},
	}
	for _, s := range statuses {
		if s.on {
			return s.emoji
		}
	}
	return ""
}

func (s *walletStore) ailments(ctx context.Context, chat types.JID, acc core.Account, now time.Time) (a ailments, err error) {
	var (
		stored, since int64
		tipos         string
	)
	at := now.Unix()
	err = s.db.QueryRowContext(ctx,
		`SELECT c.morto_ate > ?, c.preso_ate > ?,
		        EXISTS (SELECT 1 FROM sequestros q WHERE q.chat = c.chat AND q.refem = c.pessoa AND q.ate > ?),
		        EXISTS (SELECT 1 FROM zoeiras z WHERE z.chat = c.chat AND z.alvo = c.pessoa AND z.tipo = ? AND z.ate > ?),
		        EXISTS (SELECT 1 FROM zoeiras z WHERE z.chat = c.chat AND z.alvo = c.pessoa AND z.tipo = ? AND z.ate > ?),
		        COALESCE((SELECT GROUP_CONCAT(e.tipo) FROM encantos e WHERE e.chat = c.chat AND e.alvo = c.pessoa AND e.ate > ?), ''),
		        c.droga_ate > ?, c.saude, c.saude_em, `+staffKinds+`
		 FROM carteiras c WHERE c.chat = ? AND c.pessoa = ?`,
		at, at, at, clownKind, at, curseKind, at, at, at, chat.String(), acc.Key,
	).Scan(&a.dead, &a.jailed, &a.hostage, &a.clown, &a.hexed, &a.charms, &a.drugged, &stored, &since, &tipos)
	if errors.Is(err, sql.ErrNoRows) {
		return ailments{health: MaxHealth}, nil
	}
	if err != nil {
		return ailments{}, err
	}
	a.health = healthAt(stored, since, regenEvery(kindsOf(tipos)), now)
	return a, nil
}

func (b *bot) reactToAilments(ctx context.Context, evt *events.Message) bool {
	if evt.IsEdit || !core.Revocable(evt.Message) {
		return false
	}
	chat := evt.Info.Chat.ToNonAD()
	if !b.EconomyOn(ctx, chat) {
		return false
	}
	a, err := b.Wallets.ailments(ctx, chat, b.Account(ctx, evt.Info.Sender), time.Now())
	if err != nil {
		b.Client.Log.Warnf("checar status de %s: %v", evt.Info.Sender, err)
		return false
	}
	emoji := a.emoji()
	if emoji == "" {
		return false
	}
	b.React(ctx, evt, emoji)
	return true
}
