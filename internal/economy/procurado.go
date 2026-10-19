package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"golang.org/x/sync/errgroup"
)

const (
	wantedHour      = 18
	wantedAuthor    = "bot"
	wantedMin       = 500
	wantedMax       = 2000
	wantedActive    = 3 * 24 * time.Hour
	wantedChatLimit = 4

	posseKillChance   = 0.5
	posseLampiaoBonus = 0.3
	lampiaoItem       = "chapeu-de-lampiao"
)

type wantedPost struct {
	target types.JID
	prize  int64
	total  int64
}

func (b *bot) runWanted(ctx context.Context) {
	ticker := time.NewTicker(core.ReminderTick)
	defer ticker.Stop()
	for {
		if b.Client.IsLoggedIn() {
			b.postDueWanted(ctx, time.Now())
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (b *bot) postDueWanted(ctx context.Context, now time.Time) {
	local := now.In(core.ReminderZone)
	if local.Hour() < wantedHour {
		return
	}
	day := local.Format(time.DateOnly)
	chats, err := b.Wallets.unwantedChats(ctx, day)
	if err != nil {
		b.Client.Log.Errorf("procurar conversas pro procurado do dia: %v", err)
		return
	}
	var g errgroup.Group
	g.SetLimit(wantedChatLimit)
	for _, chat := range chats {
		g.Go(func() error {
			if err := b.wantedChat(ctx, chat, day, now); err != nil {
				b.Client.Log.Errorf("botar o procurado do dia em %s: %v", chat, err)
			}
			return nil
		})
	}
	_ = g.Wait()
}

func (b *bot) wantedChat(ctx context.Context, chat types.JID, day string, now time.Time) error {
	if len(b.economyGroups(ctx, chat)) == 0 {
		return nil
	}
	w, ok, err := b.Wallets.postWanted(ctx, chat, day, now)
	if err != nil || !ok {
		return err
	}
	text := fmt.Sprintf("🤖🎯 *Procurado do dia*\n\nO bot botou %s na cabeça de @%s! Com o que os outros já tinham botado, a cabeça vale %s, e quem conseguir !matar leva tudo\n\n_Amanhã às %dh sai outro procurado, e o prêmio do bot que ninguém pegou some_",
		Coins(w.prize), w.target.User, Coins(w.total), wantedHour)
	return b.announce(ctx, chat, text, w.target)
}

func (s *walletStore) unwantedChats(ctx context.Context, day string) (chats []types.JID, err error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT DISTINCT chat FROM carteiras WHERE chat NOT IN (SELECT chat FROM procurados WHERE dia = ?)",
		day,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		chat, err := types.ParseJID(raw)
		if err != nil {
			return nil, err
		}
		if ledgers.linked(chat) {
			continue
		}
		chats = append(chats, chat)
	}
	return chats, rows.Err()
}

func (s *walletStore) postWanted(ctx context.Context, chat types.JID, day string, now time.Time) (w wantedPost, ok bool, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		res, err := tx.ExecContext(ctx, "INSERT INTO procurados (chat, dia) VALUES (?, ?) ON CONFLICT DO NOTHING", chat.String(), day)
		if err != nil {
			return err
		}
		claimed, err := res.RowsAffected()
		if err != nil || claimed == 0 {
			return err
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM procurados WHERE chat = ? AND dia < ?", chat.String(), day); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM recompensas WHERE chat = ? AND autor = ?", chat.String(), wantedAuthor); err != nil {
			return err
		}

		var (
			target core.Account
			raw    string
		)
		err = tx.QueryRowContext(ctx,
			`SELECT pessoa, jid FROM carteiras
			 WHERE chat = ? AND morto_ate <= ? AND MAX(daily_em, trabalho_em, roubo_em) >= ?
			 ORDER BY RANDOM() LIMIT 1`,
			chat.String(), now.Unix(), now.Add(-wantedActive).Unix(),
		).Scan(&target.Key, &raw)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if target.JID, err = types.ParseJID(raw); err != nil {
			return err
		}

		w.target, w.prize = target.JID, wantedMin+rand.Int64N(wantedMax-wantedMin+1)
		_, err = tx.ExecContext(ctx,
			"INSERT INTO recompensas (chat, alvo, autor, valor, em) VALUES (?, ?, ?, ?, ?)",
			chat.String(), target.Key, wantedAuthor, w.prize, now.Unix(),
		)
		if err != nil {
			return err
		}
		w.total, err = bountyTotal(ctx, tx, chat, target)
		ok = err == nil
		return err
	})
	return w, ok, err
}

func (s *walletStore) wanted(ctx context.Context, chat types.JID) (w wantedPost, ok bool, err error) {
	var raw string
	err = s.db.QueryRowContext(ctx,
		`SELECT c.jid, r.valor FROM recompensas r
		 JOIN carteiras c ON c.chat = r.chat AND c.pessoa = r.alvo
		 WHERE r.chat = ? AND r.autor = ?`,
		chat.String(), wantedAuthor,
	).Scan(&raw, &w.prize)
	if errors.Is(err, sql.ErrNoRows) {
		return w, false, nil
	}
	if err != nil {
		return w, false, err
	}
	w.target, err = types.ParseJID(raw)
	return w, err == nil, err
}

func wantedPrizeOn(ctx context.Context, q rowQuerier, chat types.JID, acc core.Account) (int64, error) {
	var prize int64
	err := q.QueryRowContext(ctx,
		"SELECT valor FROM recompensas WHERE chat = ? AND alvo = ? AND autor = ?",
		chat.String(), acc.Key, wantedAuthor,
	).Scan(&prize)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return prize, err
}

func (s *walletStore) wantedPrize(ctx context.Context, chat types.JID, acc core.Account) (int64, error) {
	return wantedPrizeOn(ctx, s.db, chat, acc)
}

func posseReward(ctx context.Context, tx *ledgerTx, chat types.JID, victim, killer core.Account, posse bool) (int64, error) {
	if !posse {
		return 0, nil
	}
	prize, err := wantedPrizeOn(ctx, tx, chat, victim)
	if err != nil || prize == 0 {
		return 0, err
	}
	_, err = tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", prize, chat.String(), killer.Key)
	return prize, err
}

func (p Profession) hunts(victim Profession) float64 {
	if !p.posse || !slices.Contains(victim.luxuries, lampiaoItem) {
		return 0
	}
	return posseLampiaoBonus
}

func posseText(victim types.JID) string {
	return fmt.Sprintf("\n🤠 Foi caçada da volante: @%s era o procurado do dia, então não conta como crime", victim.User)
}
