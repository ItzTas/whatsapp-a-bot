package stats

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const retroChatReach = 5 * time.Minute

type retroYear struct {
	year    int
	since   int64
	until   int64
	ongoing bool
	clipped bool
}

type retroDay struct {
	day      time.Time
	messages int
}

type retroStats struct {
	ranked   []rankedMember
	clowns   []rankedMember
	partners []rankedMember
	hours    hourlyCounts
	day      retroDay
	word     wordCount
	hasWord  bool
	first    core.Quote
	hasFirst bool
}

func (b *bot) retrospectiva(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return b.SendText(ctx, chat, "A !retrospectiva só funciona em grupo")
	}
	period, problem := retroYearFrom(core.MessageText(evt.Message))
	if problem != "" {
		return b.SendText(ctx, chat, problem)
	}
	person, single := core.StatsSubjectOf(evt)
	if single {
		return b.personRetro(ctx, chat, person, period)
	}
	return b.groupRetro(ctx, chat, period)
}

func retroYearFrom(text string) (retroYear, string) {
	now := time.Now().In(core.BrazilTime)
	year := now.Year()
	for _, word := range core.NormalizedWords(text) {
		if n, err := strconv.Atoi(word); err == nil && len(word) == 4 {
			year = n
		}
	}
	if year > now.Year() {
		return retroYear{}, fmt.Sprintf("%d ainda nem chegou, volta aqui no fim do ano 🔮", year)
	}
	start := time.Date(year, time.January, 1, 0, 0, 0, 0, core.BrazilTime).Unix()
	end := time.Date(year+1, time.January, 1, 0, 0, 0, 0, core.BrazilTime).Unix()
	if end <= core.WindowStart() {
		return retroYear{}, fmt.Sprintf("Só guardo as mensagens dos %s, então de %d eu não lembro de nada 🧠", core.MessageWindowLabel, year)
	}
	return retroYear{
		year:    year,
		since:   max(start, core.WindowStart()),
		until:   end,
		ongoing: year == now.Year(),
		clipped: start < core.WindowStart(),
	}, ""
}

func (b *bot) groupRetro(ctx context.Context, chat types.JID, period retroYear) error {
	var (
		group  *types.GroupInfo
		counts []core.SenderCount
		laughs []core.SenderCount
		s      retroStats
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { group, err = b.Client.GetGroupInfo(gctx, chat); return err })
	g.Go(func() (err error) {
		counts, err = b.messages.sendersBetween(gctx, chat, period.since, period.until)
		return err
	})
	g.Go(func() (err error) {
		laughs, err = b.messages.laughTargetsBetween(gctx, chat, period.since, period.until)
		return err
	})
	g.Go(func() (err error) {
		s.hours, err = b.messages.hourlyCounts(gctx, chat, nil, period.since, period.until)
		return err
	})
	g.Go(func() (err error) {
		s.day, err = b.messages.busiestDay(gctx, chat, nil, period.since, period.until)
		return err
	})
	g.Go(func() error {
		texts, err := b.messages.textsBetween(gctx, chat, nil, period.since, period.until)
		s.word, s.hasWord = retroWordOf(texts)
		return err
	})
	if err := g.Wait(); err != nil {
		return fmt.Errorf("montar retrospectiva do grupo: %w", err)
	}

	if s.hours.total() == 0 {
		return b.SendText(ctx, chat, fmt.Sprintf("Ninguém falou nada em %d 🦗", period.year))
	}
	s.ranked = b.rankMembers(group.Participants, counts)
	s.clowns = b.rankMembers(group.Participants, laughs)
	text, mentions := s.groupCard(period)
	_, err := b.SendMentions(ctx, chat, text, mentions)
	return err
}

func (b *bot) personRetro(ctx context.Context, chat, person types.JID, period retroYear) error {
	var (
		group    *types.GroupInfo
		counts   []core.SenderCount
		laughs   []core.SenderCount
		partners []core.SenderCount
		aliases  []string
		s        retroStats
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { group, err = b.Client.GetGroupInfo(gctx, chat); return err })
	g.Go(func() (err error) {
		counts, err = b.messages.sendersBetween(gctx, chat, period.since, period.until)
		return err
	})
	g.Go(func() (err error) {
		laughs, err = b.messages.laughTargetsBetween(gctx, chat, period.since, period.until)
		return err
	})
	g.Go(func() error {
		aliases = b.UserAliases(gctx, person)
		h, hctx := errgroup.WithContext(gctx)
		h.Go(func() (err error) {
			s.hours, err = b.messages.hourlyCounts(hctx, chat, aliases, period.since, period.until)
			return err
		})
		h.Go(func() (err error) {
			s.day, err = b.messages.busiestDay(hctx, chat, aliases, period.since, period.until)
			return err
		})
		h.Go(func() (err error) {
			partners, err = b.messages.chatPartners(hctx, chat, aliases, period.since, period.until)
			return err
		})
		h.Go(func() (err error) {
			s.first, s.hasFirst, err = b.messages.firstMessageSince(hctx, chat, aliases, period.since)
			return err
		})
		h.Go(func() error {
			texts, err := b.messages.textsBetween(hctx, chat, aliases, period.since, period.until)
			s.word, s.hasWord = retroWordOf(texts)
			return err
		})
		return h.Wait()
	})
	if err := g.Wait(); err != nil {
		return fmt.Errorf("montar retrospectiva de %s: %w", person.User, err)
	}

	if s.hours.total() == 0 {
		_, err := b.SendMentions(ctx, chat, fmt.Sprintf("@%s não mandou nada aqui em %d 🦗", person.User, period.year), []types.JID{person})
		return err
	}
	s.ranked = b.rankMembers(group.Participants, counts)
	s.clowns = b.rankMembers(group.Participants, laughs)
	s.partners = slices.DeleteFunc(b.rankMembers(group.Participants, partners), func(m rankedMember) bool {
		return slices.Contains(aliases, m.jid.User)
	})
	if s.first.SentAt.Unix() >= period.until {
		s.hasFirst = false
	}
	text, mentions := s.personCard(period, person, aliases)
	_, err := b.SendMentions(ctx, chat, text, mentions)
	return err
}

func retroWordOf(texts []string) (wordCount, bool) {
	words := slices.DeleteFunc(favoriteWords(texts), func(w wordCount) bool { return core.IsLaugh(w.word) })
	if len(words) == 0 {
		return wordCount{}, false
	}
	return words[0], true
}

func (p retroYear) title(subject string) string {
	title := fmt.Sprintf("🎁 *Retrospectiva %d %s*", p.year, subject)
	if p.ongoing {
		return title + " _(até agora)_"
	}
	return title
}

func (p retroYear) footer() []string {
	if !p.clipped {
		return nil
	}
	return []string{"", fmt.Sprintf("_Só guardo as mensagens dos %s, então o começo de %d ficou de fora_", core.MessageWindowLabel, p.year)}
}

func (s retroStats) groupCard(period retroYear) (string, []types.JID) {
	top := s.ranked[:min(len(s.ranked), 3)]
	lines := []string{
		period.title("do grupo"),
		"",
		fmt.Sprintf("💬 *%s mensagens*, mandadas por %s", core.Thousands(int64(s.hours.total())), core.CountLabel(len(s.ranked), "pessoa", "pessoas")),
	}
	if len(top) > 0 {
		lines = slices.Concat(lines, []string{"", "🏆 *Quem mais falou*"}, rankingLines(top, "mensagem", "mensagens"), []string{""})
	}
	lines = append(lines,
		fmt.Sprintf("📅 Dia mais movimentado: *%s* (%s)", core.BrazilianDate(s.day.day), core.CountLabel(s.day.messages, "mensagem", "mensagens")),
		fmt.Sprintf("⏰ Horário de pico: *%s*", hourlyRange(s.hours.peak())),
	)
	if s.hasWord {
		lines = append(lines, fmt.Sprintf("🔤 Palavra do ano: *%s* (%dx)", s.word.word, s.word.count))
	}

	mentions := make([]types.JID, len(top))
	for i, m := range top {
		mentions[i] = m.jid
	}
	if len(s.clowns) > 0 {
		clown := s.clowns[0]
		lines = append(lines, fmt.Sprintf("🤡 Quem mais fez rir: @%s (%s)", clown.jid.User, core.CountLabel(clown.messages, "risada", "risadas")))
		mentions = append(mentions, clown.jid)
	}
	lines = append(lines, "", hourlyComment(s.hours.peak()))
	return strings.Join(append(lines, period.footer()...), "\n"), core.UniqueJIDs(mentions)
}

func (s retroStats) personCard(period retroYear, person types.JID, aliases []string) (string, []types.JID) {
	isPerson := func(m rankedMember) bool { return slices.Contains(aliases, m.jid.User) }
	lines := []string{
		period.title("de @" + person.User),
		"",
		fmt.Sprintf("💬 *%s mensagens*", core.Thousands(int64(s.hours.total()))),
	}
	if position := slices.IndexFunc(s.ranked, isPerson) + 1; position > 0 {
		lines = append(lines, fmt.Sprintf("🏆 %dº lugar no ranking, de %d pessoas", position, len(s.ranked)))
	}
	lines = append(lines,
		fmt.Sprintf("📅 Dia mais falador: *%s* (%s)", core.BrazilianDate(s.day.day), core.CountLabel(s.day.messages, "mensagem", "mensagens")),
		fmt.Sprintf("⏰ Horário favorito: *%s*", hourlyRange(s.hours.peak())),
	)
	if s.hasWord {
		lines = append(lines, fmt.Sprintf("🔤 Palavra mais usada: *%s* (%dx)", s.word.word, s.word.count))
	}

	mentions := []types.JID{person}
	if len(s.partners) > 0 {
		partner := s.partners[0]
		lines = append(lines, fmt.Sprintf("👯 Parceria de papo: @%s (%s trocando ideia)", partner.jid.User, core.CountLabel(partner.messages, "vez", "vezes")))
		mentions = append(mentions, partner.jid)
	}
	lines = append(lines, retroLaughLine(s.clowns, isPerson))
	if s.hasFirst {
		lines = append(lines, "", fmt.Sprintf("📜 Primeira mensagem do ano, em %s:", core.BrazilianDate(s.first.SentAt)), core.QuoteBlock(s.first.Text))
	}
	return strings.Join(append(lines, period.footer()...), "\n"), core.UniqueJIDs(mentions)
}

func retroLaughLine(clowns []rankedMember, isPerson func(rankedMember) bool) string {
	i := slices.IndexFunc(clowns, isPerson)
	if i < 0 {
		return "😐 Não arrancou nenhuma risada do grupo"
	}
	return fmt.Sprintf("😂 Arrancou %s do grupo (%dº maior palhaço)", core.CountLabel(clowns[i].messages, "risada", "risadas"), i+1)
}

func (s *messageStore) sendersBetween(ctx context.Context, chat types.JID, since, until int64) ([]core.SenderCount, error) {
	return s.retroSenderCounts(ctx,
		`SELECT remetente, remetente_alt, COUNT(*) FROM mensagens
		 WHERE chat = ? AND enviada_em >= ? AND enviada_em < ?
		 GROUP BY remetente, remetente_alt`,
		chat.String(), since, until,
	)
}

func (s *messageStore) laughTargetsBetween(ctx context.Context, chat types.JID, since, until int64) ([]core.SenderCount, error) {
	return s.retroSenderCounts(ctx,
		`SELECT alvo, alvo_alt, COUNT(DISTINCT quem_riu || ' ' || mensagem_alvo) FROM risadas
		 WHERE chat = ? AND enviada_em >= ? AND enviada_em < ?
		 GROUP BY alvo, alvo_alt`,
		chat.String(), since, until,
	)
}

func (s *messageStore) chatPartners(ctx context.Context, chat types.JID, users []string, since, until int64) ([]core.SenderCount, error) {
	return s.retroSenderCounts(ctx,
		`WITH seq AS (
		   SELECT remetente, remetente_alt, enviada_em,
		          LAG(remetente) OVER w AS antes,
		          LAG(remetente_alt) OVER w AS antes_alt,
		          LAG(enviada_em) OVER w AS antes_em
		   FROM mensagens
		   WHERE chat = :chat AND enviada_em >= :since AND enviada_em < :until
		   WINDOW w AS (ORDER BY enviada_em)
		 ), pares AS (
		   SELECT remetente, remetente_alt, antes, antes_alt,
		          (remetente IN (:a, :b) OR remetente_alt IN (:a, :b)) AS eu_agora,
		          (antes IN (:a, :b) OR antes_alt IN (:a, :b)) AS eu_antes
		   FROM seq
		   WHERE antes IS NOT NULL AND enviada_em - antes_em <= :reach
		 )
		 SELECT CASE WHEN eu_agora THEN antes ELSE remetente END,
		        CASE WHEN eu_agora THEN antes_alt ELSE remetente_alt END,
		        COUNT(*)
		 FROM pares
		 WHERE eu_agora <> eu_antes
		 GROUP BY 1, 2`,
		sql.Named("chat", chat.String()), sql.Named("since", since), sql.Named("until", until),
		sql.Named("a", users[0]), sql.Named("b", users[len(users)-1]), sql.Named("reach", int64(retroChatReach.Seconds())),
	)
}

func (s *messageStore) retroSenderCounts(ctx context.Context, query string, args ...any) (counts []core.SenderCount, err error) {
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var c core.SenderCount
		if err := rows.Scan(&c.Sender, &c.SenderAlt, &c.Messages); err != nil {
			return nil, err
		}
		counts = append(counts, c)
	}
	return counts, rows.Err()
}

func (s *messageStore) busiestDay(ctx context.Context, chat types.JID, users []string, since, until int64) (retroDay, error) {
	filter, filterArgs := statsSenderFilter(users)
	offset := statsZoneOffset()
	var (
		day int64
		d   retroDay
	)
	err := s.DB.QueryRowContext(ctx,
		`SELECT (enviada_em + ?) / 86400 AS dia, COUNT(*) FROM mensagens
		 WHERE chat = ? AND enviada_em >= ? AND enviada_em < ?`+filter+`
		 GROUP BY dia ORDER BY COUNT(*) DESC, dia LIMIT 1`,
		slices.Concat([]any{offset, chat.String(), since, until}, filterArgs)...,
	).Scan(&day, &d.messages)
	if errors.Is(err, sql.ErrNoRows) {
		return retroDay{}, nil
	}
	if err != nil {
		return retroDay{}, err
	}
	d.day = time.Unix(day*86400-offset, 0)
	return d, nil
}

func (s *messageStore) textsBetween(ctx context.Context, chat types.JID, users []string, since, until int64) (texts []string, err error) {
	filter, filterArgs := statsSenderFilter(users)
	rows, err := s.DB.QueryContext(ctx,
		`SELECT texto FROM mensagens
		 WHERE chat = ? AND enviada_em >= ? AND enviada_em < ? AND ltrim(texto) NOT LIKE '!%'`+filter,
		slices.Concat([]any{chat.String(), since, until}, filterArgs)...,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			return nil, err
		}
		texts = append(texts, text)
	}
	return texts, rows.Err()
}
