package economy

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"golang.org/x/sync/errgroup"
)

const (
	vacancyWindow       = 30 * time.Minute
	drawActivity        = 24 * time.Hour
	successionChatLimit = 4
)

type seat struct {
	held   bool
	holder types.JID
	ended  time.Time
	drawAt time.Time
}

func (b *bot) runSuccession(ctx context.Context) {
	ticker := time.NewTicker(core.ReminderTick)
	defer ticker.Stop()
	for {
		if b.Client.IsLoggedIn() {
			b.fillSeats(ctx, time.Now())
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (b *bot) fillSeats(ctx context.Context, now time.Time) {
	chats, err := b.Wallets.economies(ctx)
	if err != nil {
		b.Client.Log.Errorf("procurar conversas sem presidente: %v", err)
		return
	}
	var g errgroup.Group
	g.SetLimit(successionChatLimit)
	for _, chat := range chats {
		g.Go(func() error {
			if err := b.fillSeat(ctx, chat, now); err != nil {
				b.Client.Log.Errorf("ver a presidência de %s: %v", chat, err)
			}
			return nil
		})
	}
	_ = g.Wait()
}

func (b *bot) fillSeat(ctx context.Context, chat types.JID, now time.Time) error {
	groups := b.economyGroups(ctx, chat)
	if len(groups) == 0 {
		return nil
	}
	vacant, err := b.Wallets.seat(ctx, chat, now)
	if err != nil || vacant.held {
		return err
	}
	if vacant.drawAt.IsZero() {
		return b.openSeat(ctx, chat, vacant, now)
	}
	if now.Before(vacant.drawAt) || b.electionIn(groups) {
		return nil
	}
	return b.drawSeat(ctx, chat, groups, now)
}

func (b *bot) openSeat(ctx context.Context, chat types.JID, vacant seat, now time.Time) error {
	opened, err := b.Wallets.openVacancy(ctx, chat, now)
	if err != nil || !opened {
		return err
	}
	headline := "🏛️ *A cadeira de presidente tá vaga!*"
	var featured []types.JID
	if !vacant.holder.IsEmpty() && now.Sub(vacant.ended) < vacancyWindow {
		headline = fmt.Sprintf("⏰ *Acabou o mandato de @%s!*", vacant.holder.User)
		featured = append(featured, vacant.holder)
	}
	text := fmt.Sprintf("%s Vocês têm %s pra eleger alguém: manda !votar @pessoa pra abrir a eleição\n\n🎲 %s",
		headline, core.FormatWait(vacancyWindow), vacancyNote(now.Add(vacancyWindow), now))
	return b.announce(ctx, chat, text, featured...)
}

func (b *bot) electionIn(groups []types.JID) bool {
	return slices.ContainsFunc(groups, func(group types.JID) bool {
		_, running := b.elections.Load(group.String())
		return running
	})
}

func (b *bot) drawSeat(ctx context.Context, chat types.JID, groups []types.JID, now time.Time) error {
	pool, err := b.activeMembers(ctx, groups, now)
	if err != nil || len(pool) == 0 {
		return err
	}
	winner := core.PickOne(pool)
	drawn, err := b.Wallets.drawPresident(ctx, chat, winner, now)
	if err != nil || !drawn {
		return err
	}
	text := fmt.Sprintf("🎲 *Sorteio pra presidente!*\nNinguém foi eleito a tempo, então o bot sorteou entre quem mandou mensagem nas últimas %s (%s no sorteio), e deu @%s! 🏛️ O mandato vai até daqui a %s, e a próxima eleição só depois disso\n\n%s",
		core.FormatWait(drawActivity), core.CountLabel(len(pool), "pessoa", "pessoas"), winner.JID.User, core.FormatWait(termLength), strings.Join(presidentialPowers(), "\n"))
	return b.announce(ctx, chat, text, winner.JID)
}

func (b *bot) activeMembers(ctx context.Context, groups []types.JID, now time.Time) ([]core.Account, error) {
	since := now.Add(-drawActivity).Unix()
	found := make([][]types.JID, len(groups))
	g, gctx := errgroup.WithContext(ctx)
	for i, group := range groups {
		g.Go(func() (err error) { found[i], err = b.activeIn(gctx, group, since); return err })
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	byKey := map[string]core.Account{}
	for _, jid := range slices.Concat(found...) {
		acc := b.Account(ctx, jid)
		byKey[acc.Key] = acc
	}
	return slices.Collect(maps.Values(byKey)), nil
}

func (b *bot) activeIn(ctx context.Context, group types.JID, since int64) ([]types.JID, error) {
	if group.Server != types.GroupServer {
		return nil, nil
	}
	counts, err := b.Messages.CountsBySender(ctx, group, since)
	if err != nil || len(counts) == 0 {
		return nil, err
	}
	info, err := b.Client.GetGroupInfo(ctx, group)
	if err != nil {
		return nil, err
	}
	owners := b.MemberOwners(info.Participants)
	active := make([]types.JID, 0, len(counts))
	for _, c := range counts {
		if owner, ok := owners.Of(c.Sender, c.SenderAlt); ok {
			active = append(active, owner)
		}
	}
	return active, nil
}

func vacancyNote(drawAt, now time.Time) string {
	if drawAt.IsZero() {
		drawAt = now.Add(vacancyWindow)
	}
	if !drawAt.After(now) {
		return fmt.Sprintf("O prazo pra eleger já acabou, então se a urna não decidir, o bot sorteia o presidente entre quem mandou mensagem nas últimas %s", core.FormatWait(drawActivity))
	}
	return fmt.Sprintf("Se ninguém for eleito em %s, o bot sorteia o presidente entre quem mandou mensagem nas últimas %s", core.FormatWait(drawAt.Sub(now)), core.FormatWait(drawActivity))
}

func (s *walletStore) economies(ctx context.Context) (chats []types.JID, err error) {
	rows, err := s.db.QueryContext(ctx, "SELECT DISTINCT chat FROM carteiras")
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
		if ledgers.linked(chat) || chat.Server != types.GroupServer {
			continue
		}
		chats = append(chats, chat)
	}
	return chats, rows.Err()
}

func (s *walletStore) seat(ctx context.Context, chat types.JID, now time.Time) (st seat, err error) {
	var (
		holder        string
		until, drawAt int64
	)
	err = s.db.QueryRowContext(ctx,
		`SELECT COALESCE((SELECT jid FROM presidencias WHERE chat = ?), ''),
		        COALESCE((SELECT mandato_ate FROM presidencias WHERE chat = ?), 0),
		        COALESCE((SELECT sorteio_em FROM presidencia_vagas WHERE chat = ?), 0)`,
		chat.String(), chat.String(), chat.String(),
	).Scan(&holder, &until, &drawAt)
	if err != nil {
		return seat{}, err
	}
	st.held, st.ended = until > now.Unix(), time.Unix(until, 0)
	if drawAt > 0 {
		st.drawAt = time.Unix(drawAt, 0)
	}
	if holder == "" {
		return st, nil
	}
	st.holder, err = types.ParseJID(holder)
	return st, err
}

func (s *walletStore) openVacancy(ctx context.Context, chat types.JID, now time.Time) (opened bool, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		opened, err = openVacancyTx(ctx, tx, chat, now)
		return err
	})
	return opened, err
}

func (s *walletStore) vacate(ctx context.Context, chat types.JID, now time.Time, query string, args ...any) (vacated bool, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		res, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil || affected == 0 {
			return err
		}
		vacated = true
		_, err = openVacancyTx(ctx, tx, chat, now)
		return err
	})
	return vacated, err
}

func openVacancyTx(ctx context.Context, tx *ledgerTx, chat types.JID, now time.Time) (bool, error) {
	res, err := tx.ExecContext(ctx,
		`INSERT INTO presidencia_vagas (chat, sorteio_em)
		 SELECT ?, ? WHERE NOT EXISTS (SELECT 1 FROM presidencias WHERE chat = ? AND mandato_ate > ?)
		 ON CONFLICT (chat) DO NOTHING`,
		chat.String(), now.Add(vacancyWindow).Unix(), chat.String(), now.Unix(),
	)
	if err != nil {
		return false, err
	}
	opened, err := res.RowsAffected()
	return opened > 0, err
}

func closeVacancy(ctx context.Context, tx *ledgerTx, chat types.JID) error {
	_, err := tx.ExecContext(ctx, "DELETE FROM presidencia_vagas WHERE chat = ?", chat.String())
	return err
}

func (s *walletStore) drawPresident(ctx context.Context, chat types.JID, acc core.Account, now time.Time) (drawn bool, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		res, err := tx.ExecContext(ctx, "DELETE FROM presidencia_vagas WHERE chat = ? AND sorteio_em <= ?", chat.String(), now.Unix())
		if err != nil {
			return err
		}
		claimed, err := res.RowsAffected()
		if err != nil || claimed == 0 {
			return err
		}
		drawn = true
		return swearIn(ctx, tx, chat, acc, 0, true, now)
	})
	return drawn, err
}
