package main

import (
	"context"
	"errors"
	"fmt"

	"go.mau.fi/whatsmeow/types"
)

const (
	tithePercent = 30
	boonPercent  = 50
	boonCap      = 2000
)

type charmPayout struct {
	charm  charm
	bearer types.JID
	author types.JID
	amount int64
}

func (s *walletStore) collectPayouts(ctx context.Context, chat types.JID) (payouts []charmPayout, err error) {
	rows, err := s.db.QueryContext(ctx, "DELETE FROM encantos_avisos WHERE chat = ? RETURNING tipo, alvo_jid, autor_jid, valor", chat.String())
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var (
			p              charmPayout
			id             string
			bearer, author string
		)
		if err := rows.Scan(&id, &bearer, &author, &p.amount); err != nil {
			return nil, err
		}
		p.charm, _ = charmByID(id)
		if p.bearer, err = types.ParseJID(bearer); err != nil {
			return nil, err
		}
		if p.author, err = types.ParseJID(author); err != nil {
			return nil, err
		}
		payouts = append(payouts, p)
	}
	return payouts, rows.Err()
}

func (b *bot) announcePayouts(ctx context.Context, chat types.JID) {
	payouts, err := b.wallets.collectPayouts(ctx, chat)
	if err != nil {
		b.client.Log.Errorf("ler encantos cobrados em %s: %v", chat, err)
		return
	}
	for _, p := range payouts {
		if _, err := b.sendMentions(ctx, chat, payoutText(p), []types.JID{p.bearer, p.author}); err != nil {
			b.client.Log.Errorf("avisar %s em %s: %v", p.charm.id, chat, err)
		}
	}
}

func payoutText(p charmPayout) string {
	if p.charm.curse {
		return fmt.Sprintf("😈 A praga cobrou! @%s tava com %s, ganhou dinheiro e teve que passar %s pra @%s", p.bearer.User, p.charm.label(), coins(p.amount), p.author.User)
	}
	if p.bearer == p.author {
		return fmt.Sprintf("😇 Milagre! @%s tava com %s e o dinheiro que entrou se multiplicou: mais %s", p.bearer.User, p.charm.label(), coins(p.amount))
	}
	return fmt.Sprintf("😇 Milagre! @%s tava com %s, que @%s deu, e o dinheiro que entrou se multiplicou: mais %s", p.bearer.User, p.charm.label(), p.author.User, coins(p.amount))
}
