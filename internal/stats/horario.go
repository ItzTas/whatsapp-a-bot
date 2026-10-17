package stats

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
)

const hourlyBarWidth = 12

type hourlyCounts [24]int

func (b *bot) horario(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	period := rankingPeriodFrom(core.MessageText(evt.Message))
	now := time.Now()
	since := max(now.Add(-period.length).Unix(), core.WindowStart())

	person, single := core.StatsSubjectOf(evt)
	var users []string
	if single {
		users = b.UserAliases(ctx, person)
	}
	hours, err := b.messages.hourlyCounts(ctx, chat, users, since, now.Unix()+1)
	if err != nil {
		return fmt.Errorf("contar mensagens por hora: %w", err)
	}

	who := "o grupo"
	var mentions []types.JID
	if single {
		who = "@" + person.User
		mentions = []types.JID{person}
	}
	if hours.total() == 0 {
		_, err := b.SendMentions(ctx, chat, fmt.Sprintf("Não vi %s falar nada nos %s 🦗", who, period.label), mentions)
		return err
	}
	text := strings.Join([]string{
		fmt.Sprintf("⏰ *Que horas %s mais fala* nos %s", who, period.label),
		"",
		hours.chart(),
		"",
		fmt.Sprintf("🔥 Pico: *%s* (%s)", hourlyRange(hours.peak()), core.CountLabel(hours[hours.peak()], "mensagem", "mensagens")),
		hourlyComment(hours.peak()),
	}, "\n")
	_, err = b.SendMentions(ctx, chat, text, mentions)
	return err
}

func (h hourlyCounts) total() int {
	total := 0
	for _, n := range h {
		total += n
	}
	return total
}

func (h hourlyCounts) peak() int {
	peak := 0
	for hour, n := range h {
		if n > h[peak] {
			peak = hour
		}
	}
	return peak
}

func (h hourlyCounts) chart() string {
	top := h[h.peak()]
	digits := len(fmt.Sprint(top))
	lines := make([]string, len(h))
	for hour, n := range h {
		bar := strings.Repeat("█", (n*hourlyBarWidth+top-1)/top)
		lines[hour] = fmt.Sprintf("%02dh %-*s %*d", hour, hourlyBarWidth, bar, digits, n)
	}
	return "```\n" + strings.Join(lines, "\n") + "\n```"
}

func hourlyRange(hour int) string {
	return fmt.Sprintf("das %dh às %dh", hour, (hour+1)%24)
}

func hourlyComment(hour int) string {
	type mood struct {
		until int
		text  string
	}
	moods := []mood{
		{6, "🦉 Corujão total: o papo rola de madrugada, ninguém aqui dorme"},
		{9, "🐓 Acorda e já vai pro zap antes até do café"},
		{12, "☕ O expediente começa e a produtividade termina"},
		{14, "🍽️ A fofoca da hora do almoço é sagrada"},
		{18, "💼 Era pra estar trabalhando, né?"},
		{22, "🛋️ Chegou em casa, largou tudo e foi pro grupo"},
		{24, "🌙 Os boêmios da noite, fofocando antes de dormir"},
	}
	i := slices.IndexFunc(moods, func(m mood) bool { return hour < m.until })
	return moods[i].text
}

func statsZoneOffset() int64 {
	_, offset := time.Now().In(core.BrazilTime).Zone()
	return int64(offset)
}

func statsSenderFilter(users []string) (string, []any) {
	if len(users) == 0 {
		return "", nil
	}
	first, second := users[0], users[len(users)-1]
	return " AND (remetente IN (?, ?) OR remetente_alt IN (?, ?))", []any{first, second, first, second}
}

func (s *messageStore) hourlyCounts(ctx context.Context, chat types.JID, users []string, since, until int64) (hours hourlyCounts, err error) {
	filter, filterArgs := statsSenderFilter(users)
	rows, err := s.DB.QueryContext(ctx,
		`SELECT (enviada_em + ?) / 3600 % 24, COUNT(*) FROM mensagens
		 WHERE chat = ? AND enviada_em >= ? AND enviada_em < ?`+filter+`
		 GROUP BY 1`,
		slices.Concat([]any{statsZoneOffset(), chat.String(), since, until}, filterArgs)...,
	)
	if err != nil {
		return hours, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var hour, n int
		if err := rows.Scan(&hour, &n); err != nil {
			return hours, err
		}
		hours[hour] = n
	}
	return hours, rows.Err()
}
