package main

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

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
		{a.health <= maxHealth/3, "🤕"},
	}
	for _, s := range statuses {
		if s.on {
			return s.emoji
		}
	}
	return ""
}

func (s *walletStore) ailments(ctx context.Context, chat types.JID, acc account, now time.Time) (a ailments, err error) {
	var stored, since int64
	at := now.Unix()
	err = s.db.QueryRowContext(ctx,
		`SELECT c.morto_ate > ?, c.preso_ate > ?,
		        EXISTS (SELECT 1 FROM sequestros q WHERE q.chat = c.chat AND q.refem = c.pessoa AND q.ate > ?),
		        EXISTS (SELECT 1 FROM zoeiras z WHERE z.chat = c.chat AND z.alvo = c.pessoa AND z.tipo = ? AND z.ate > ?),
		        EXISTS (SELECT 1 FROM zoeiras z WHERE z.chat = c.chat AND z.alvo = c.pessoa AND z.tipo = ? AND z.ate > ?),
		        COALESCE((SELECT GROUP_CONCAT(e.tipo) FROM encantos e WHERE e.chat = c.chat AND e.alvo = c.pessoa AND e.ate > ?), ''),
		        c.droga_ate > ?, c.saude, c.saude_em
		 FROM carteiras c WHERE c.chat = ? AND c.pessoa = ?`,
		at, at, at, clownKind, at, curseKind, at, at, at, chat.String(), acc.key,
	).Scan(&a.dead, &a.jailed, &a.hostage, &a.clown, &a.hexed, &a.charms, &a.drugged, &stored, &since)
	if errors.Is(err, sql.ErrNoRows) {
		return ailments{health: maxHealth}, nil
	}
	if err != nil {
		return ailments{}, err
	}
	a.health = healthAt(stored, since, now)
	return a, nil
}

func (b *bot) reactToAilments(ctx context.Context, evt *events.Message) {
	if evt.IsEdit || !revocable(evt.Message) {
		return
	}
	chat := evt.Info.Chat.ToNonAD()
	if !b.economyOn(ctx, chat) {
		return
	}
	a, err := b.wallets.ailments(ctx, chat, b.account(ctx, evt.Info.Sender), time.Now())
	if err != nil {
		b.client.Log.Warnf("checar status de %s: %v", evt.Info.Sender, err)
		return
	}
	if emoji := a.emoji(); emoji != "" {
		b.react(ctx, evt, emoji)
	}
}
