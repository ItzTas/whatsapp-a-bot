package economy

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	wealthLogEvery = time.Hour
	wealthKeep     = 7 * 24 * time.Hour
	wealthWindow   = 24 * time.Hour
)

type enrichment struct {
	who        types.JID
	job        Profession
	title      string
	worth      int64
	start      int64
	since      time.Time
	perHour    float64
	sinceStart time.Duration
}

func (b *bot) runWealthLog(ctx context.Context) {
	ticker := time.NewTicker(core.ReminderTick)
	defer ticker.Stop()
	var logged time.Time
	for {
		if slot := time.Now().Truncate(wealthLogEvery); slot.After(logged) {
			if err := b.Wallets.logWealth(ctx, slot); err != nil {
				b.Client.Log.Errorf("guardar o patrimônio de todo mundo: %v", err)
			} else {
				logged = slot
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *walletStore) logWealth(ctx context.Context, slot time.Time) error {
	return s.inTx(ctx, func(tx *ledgerTx) error {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO patrimonios (chat, pessoa, valor, em)
			 SELECT c.chat, c.pessoa, c.saldo + c.cofre_casa + COALESCE(p.valor, 0), ?
			 FROM carteiras c LEFT JOIN poupancas p ON p.chat = c.chat AND p.pessoa = c.pessoa
			 WHERE c.chat NOT IN (SELECT chat FROM economia_vinculos)
			 ON CONFLICT DO NOTHING`,
			slot.Unix(),
		)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "DELETE FROM patrimonios WHERE em < ?", slot.Add(-wealthKeep).Unix())
		return err
	})
}

func (s *walletStore) enrichments(ctx context.Context, chat types.JID, now time.Time, window time.Duration) (list []enrichment, err error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT c.jid, c.profissao, c.titulo, c.saldo + c.cofre_casa + COALESCE(p.valor, 0), h.valor, h.em
		 FROM carteiras c
		 JOIN patrimonios h ON h.chat = c.chat AND h.pessoa = c.pessoa
		  AND h.em = (SELECT MIN(em) FROM patrimonios WHERE chat = c.chat AND pessoa = c.pessoa AND em >= ?)
		 LEFT JOIN poupancas p ON p.chat = c.chat AND p.pessoa = c.pessoa
		 WHERE c.chat = ?`,
		now.Add(-window).Unix(), chat.String(),
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var (
			e     enrichment
			owner string
			job   string
			since int64
		)
		if err := rows.Scan(&owner, &job, &e.title, &e.worth, &e.start, &since); err != nil {
			return nil, err
		}
		if e.who, err = types.ParseJID(owner); err != nil {
			return nil, err
		}
		e.job, _ = professionNamed(job)
		e.since = time.Unix(since, 0)
		e.sinceStart = now.Sub(e.since)
		if e.sinceStart < wealthLogEvery || (e.worth == 0 && e.start == 0) {
			continue
		}
		e.perHour = float64(e.worth-e.start) / e.sinceStart.Hours()
		list = append(list, e)
	}
	slices.SortFunc(list, func(a, b enrichment) int { return cmp.Compare(b.perHour, a.perHour) })
	return list, rows.Err()
}

func (b *bot) showEnrichment(ctx context.Context, evt *events.Message, args []string) error {
	chat := evt.Info.Chat.ToNonAD()
	window, ok := enrichmentWindow(args)
	if !ok {
		return b.SendText(ctx, chat, "Não entendi a janela. Use assim: !economia enriquecimento (últimas 24h), !economia enriquecimento 12h ou !economia enriquecimento 3d (até 7d)")
	}
	list, err := b.Wallets.enrichments(ctx, chat, time.Now(), window)
	if err != nil {
		return fmt.Errorf("calcular a taxa de enriquecimento: %w", err)
	}
	if len(list) == 0 {
		return b.SendText(ctx, chat, "📈 Ainda não tem histórico pra calcular a taxa de enriquecimento. O bot guarda o patrimônio de todo mundo de hora em hora, então volta daqui a pouco")
	}
	_, err = b.SendMentions(ctx, chat, enrichmentText(list, window), enrichmentPeople(list))
	return err
}

func enrichmentWindow(args []string) (time.Duration, bool) {
	if len(args) == 0 {
		return wealthWindow, true
	}
	units := map[string]time.Duration{"h": time.Hour, "d": 24 * time.Hour}
	arg := core.FoldLetters(args[0])
	unit, ok := units[arg[len(arg)-1:]]
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(arg[:len(arg)-1])
	if err != nil || n <= 0 {
		return 0, false
	}
	return min(time.Duration(n)*unit, wealthKeep), true
}

func enrichmentText(list []enrichment, window time.Duration) string {
	lines := []string{fmt.Sprintf("📈 *Taxa de enriquecimento* (janela de %s)", core.FormatDuration(window)), ""}
	var (
		total          float64
		richer, poorer int
	)
	for i, e := range list {
		lines = append(lines, fmt.Sprintf("%d. %s%s @%s (%s): %s/h (%s em %s)",
			i+1, titleBadge(e.title), e.job.emoji, e.who.User, e.job.name,
			signedCoins(int64(math.Round(e.perHour))), signedCoins(e.worth-e.start), core.FormatDuration(e.sinceStart.Round(time.Hour))))
		total += e.perHour
		switch {
		case e.perHour > 0:
			richer++
		case e.perHour < 0:
			poorer++
		}
	}
	lines = append(lines, "",
		fmt.Sprintf("🌍 A economia toda: %s/h, %d enriquecendo e %d empobrecendo", signedCoins(int64(math.Round(total))), richer, poorer),
		"_Conta saldo, poupança e cofre de casa. A mochila fica de fora, e quem entrou depois aparece com o tempo desde a primeira vez que o bot viu_",
	)
	return strings.Join(lines, "\n")
}

func enrichmentPeople(list []enrichment) []types.JID {
	people := make([]types.JID, len(list))
	for i, e := range list {
		people[i] = e.who
	}
	return people
}

func signedCoins(n int64) string {
	if n < 0 {
		return "-" + Coins(-n)
	}
	return "+" + Coins(n)
}
