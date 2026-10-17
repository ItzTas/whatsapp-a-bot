package economy

import (
	"context"
	"database/sql"
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

var (
	errNoEstate = errors.New("sem imóvel")
	errNoRoom   = errors.New("sem vaga")
	errNoStaff  = errors.New("sem empregado")
)

type employee struct {
	id        string
	name      string
	emoji     string
	activity  string
	price     int64
	wage      int64
	every     time.Duration
	stock     int64
	luck      int
	gradeLuck int
	aliases   []string
}

var staff = []employee{
	{id: "pescador-aprendiz", name: "Pescador aprendiz", emoji: "🎣", activity: "!pescar", price: 3_300, wage: 10, every: 30 * time.Minute, stock: 24, gradeLuck: -1},
	{id: "pescador-veterano", name: "Pescador veterano", emoji: "🎣", activity: "!pescar", price: 10_500, wage: 20, every: 20 * time.Minute, stock: 36, luck: 1},
	{id: "garimpeiro-aprendiz", name: "Garimpeiro aprendiz", emoji: "⛏️", activity: "!minerar", price: 5_700, wage: 17, every: 30 * time.Minute, stock: 24, gradeLuck: -1},
	{id: "garimpeiro-veterano", name: "Garimpeiro veterano", emoji: "⛏️", activity: "!minerar", price: 18_300, wage: 37, every: 20 * time.Minute, stock: 36, luck: 1},
	{id: "catador-aprendiz", name: "Catador aprendiz", emoji: "🗑️", activity: "!vasculhar", price: 1_400, wage: 4, every: 30 * time.Minute, stock: 24, gradeLuck: -1, aliases: []string{"lixeiro aprendiz"}},
	{id: "catador-veterano", name: "Catador veterano", emoji: "🗑️", activity: "!vasculhar", price: 4_200, wage: 9, every: 20 * time.Minute, stock: 36, luck: 1, aliases: []string{"lixeiro veterano"}},
	{id: "cacador-de-insetos-aprendiz", name: "Caçador de insetos aprendiz", emoji: "🦋", activity: "!catar", price: 1_500, wage: 5, every: 30 * time.Minute, stock: 24, gradeLuck: -1, aliases: []string{"inseteiro aprendiz"}},
	{id: "cacador-de-insetos-veterano", name: "Caçador de insetos veterano", emoji: "🦋", activity: "!catar", price: 5_200, wage: 10, every: 20 * time.Minute, stock: 36, luck: 1, aliases: []string{"inseteiro veterano"}},
}

func employeeByID(id string) (employee, bool) {
	i := slices.IndexFunc(staff, func(e employee) bool { return e.id == id })
	if i < 0 {
		return employee{}, false
	}
	return staff[i], true
}

func findEmployee(query string) (employee, bool) {
	q := normalizeItemName(query)
	if q == "" {
		return employee{}, false
	}
	keys := func(e employee) []string {
		keys := []string{normalizeItemName(e.name), normalizeItemName(e.id)}
		for _, alias := range e.aliases {
			keys = append(keys, normalizeItemName(alias))
		}
		return keys
	}
	matchers := []func(key string) bool{
		func(key string) bool { return key == q },
		func(key string) bool { return strings.HasPrefix(key, q) },
		func(key string) bool { return strings.Contains(key, q) },
	}
	for _, match := range matchers {
		if i := slices.IndexFunc(staff, func(e employee) bool { return slices.ContainsFunc(keys(e), match) }); i >= 0 {
			return staff[i], true
		}
	}
	return employee{}, false
}

func (e employee) label() string {
	return e.emoji + " *" + e.name + "*"
}

func (e employee) pace(decreed decrees) time.Duration {
	return e.every * time.Duration(100+decreed.level(gatherDecree)) / 100
}

func (e employee) shift(decreed decrees) activity {
	a := activityNamed(e.activity)
	a.loot = slices.DeleteFunc(slices.Clone(a.loot), func(d drop) bool {
		it, _ := itemByID(d.item)
		return it.rarity >= legendary || it.unboxable() || it.gear != nil && it.gear.relicLure > 0
	})
	a.tool, a.breaks = "", 0
	a.luck, a.plainLuck, a.gradeLuck, a.rotten = e.luck, 0, e.gradeLuck, false
	a.cooldown.every = e.pace(decreed)
	return a
}

func (e employee) perks() string {
	skill := "traz o que acha com qualidade puxada pra baixo"
	if e.luck > 0 {
		skill = fmt.Sprintf("tem olho bom: escolhe o melhor de %d achados", e.luck+1)
	}
	return fmt.Sprintf("faz o %s a cada %s, cobra %s por vez, guarda até %d achados e %s", e.activity, core.FormatWait(e.every), Coins(e.wage), e.stock, skill)
}

func estateRoom(level int) int {
	e, ok := estateAt(level)
	if !ok {
		return 0
	}
	return e.staff
}

type hire struct {
	id    int64
	kind  employee
	since time.Time
}

func (h hire) ready(every time.Duration, now time.Time) int64 {
	if every <= 0 || now.Before(h.since) {
		return 0
	}
	return min(int64(now.Sub(h.since)/every), h.kind.stock)
}

type harvest struct {
	rounds  int64
	empty   int64
	strikes int64
	wages   int64
	balance int64
	found   map[string][]grade
}

func (b *bot) empregados(ctx context.Context, evt *events.Message) error {
	args := strings.Fields(core.FoldLetters(core.MessageText(evt.Message)))[1:]
	actions := map[string]core.Handler{
		"contratar": b.hireEmployee,
		"comprar":   b.hireEmployee,
		"demitir":   b.fireEmployee,
		"mandar":    b.fireEmployee,
		"coletar":   b.collectStaff,
		"recolher":  b.collectStaff,
		"pegar":     b.collectStaff,
	}
	if len(args) > 0 {
		if run, ok := actions[args[0]]; ok {
			return run(ctx, evt)
		}
	}
	return b.staffStatus(ctx, evt)
}

func (b *bot) staffStatus(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	now := time.Now()
	var (
		crew    []hire
		level   int
		decreed decrees
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { crew, err = b.Wallets.crew(gctx, chat, acc); return err })
	g.Go(func() (err error) { level, err = b.Wallets.estateLevel(gctx, chat, acc); return err })
	g.Go(func() (err error) { decreed, err = b.Wallets.decrees(gctx, chat, now); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler empregados: %w", err)
	}
	lines := []string{"👷 *Empregados*", ""}
	lines = append(lines, crewLines(crew, decreed, now)...)
	lines = append(lines, roomLine(person, len(crew), level), "", "📋 *Quem dá pra contratar*")
	for _, e := range staff {
		lines = append(lines, fmt.Sprintf("%s: %s\n    _%s_", e.label(), Coins(e.price), e.perks()))
	}
	lines = append(lines, "", "_Eles nunca acham relíquia, mítico, lendário, Relicário perdido nem Bússola do relicário. O que trazem vai pra mochila no !empregados coletar, e o salário sai do seu saldo na hora. Sem saldo pro salário, eles fazem greve e o achado se perde_")
	text := strings.Join(lines, "\n") + decreed.note(gatherDecree)
	_, err := b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func crewLines(crew []hire, decreed decrees, now time.Time) []string {
	if len(crew) == 0 {
		return nil
	}
	type tally struct {
		count int64
		ready int64
	}
	tallies := map[string]*tally{}
	for _, h := range crew {
		t, ok := tallies[h.kind.id]
		if !ok {
			t = &tally{}
			tallies[h.kind.id] = t
		}
		t.count++
		t.ready += h.ready(h.kind.pace(decreed), now)
	}
	var lines []string
	for _, e := range staff {
		t, ok := tallies[e.id]
		if !ok {
			continue
		}
		lines = append(lines, fmt.Sprintf("%dx %s: %d %s pra coletar", t.count, e.label(), t.ready, pluralWord(t.ready, "achado", "achados")))
	}
	return append(lines, "")
}

func roomLine(person types.JID, used, level int) string {
	room := estateRoom(level)
	if room == 0 {
		return fmt.Sprintf("🪹 @%s não tem imóvel, e ninguém trabalha pra quem não tem nem onde morar. Compra um no !imovel", person.User)
	}
	e, _ := estateAt(level)
	return fmt.Sprintf("%s @%s mora %s e tem %d de %d %s ocupadas", e.emoji, person.User, e.place, used, room, pluralWord(int64(room), "vaga", "vagas"))
}

func (b *bot) hireEmployee(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	query := strings.Join(strings.Fields(core.MessageText(evt.Message))[2:], " ")
	kind, ok := findEmployee(query)
	if !ok {
		return b.SendText(ctx, chat, "Fala quem você quer contratar, tipo !empregados contratar pescador aprendiz. Manda !empregados pra ver a lista")
	}
	room, err := b.Wallets.hireStaff(ctx, chat, acc, kind, time.Now())
	if errors.Is(err, errNoEstate) {
		return b.SendText(ctx, chat, "🪹 Ninguém trabalha pra quem não tem nem onde morar. Compra um imóvel no !imovel primeiro")
	}
	if errors.Is(err, errNoRoom) {
		return b.SendText(ctx, chat, fmt.Sprintf("🏚️ Sua casa só tem %d %s e tá tudo ocupado. Sobe de imóvel no !imovel ou demite alguém", room, pluralWord(int64(room), "vaga", "vagas")))
	}
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, chat, fmt.Sprintf("Contratar %s custa %s e você não tem tudo isso. Confere no !saldo", kind.label(), Coins(kind.price)))
	}
	if err != nil {
		return fmt.Errorf("contratar empregado: %w", err)
	}
	text := fmt.Sprintf("🤝 @%s contratou %s por %s!\n✨ Ele %s\n_Manda !empregados coletar pra pegar o que ele achar_", person.User, kind.label(), Coins(kind.price), kind.perks())
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) fireEmployee(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	query := strings.Join(strings.Fields(core.MessageText(evt.Message))[2:], " ")
	kind, ok := findEmployee(query)
	if !ok {
		return b.SendText(ctx, chat, "Fala quem você quer demitir, tipo !empregados demitir pescador aprendiz")
	}
	err := b.Wallets.fireStaff(ctx, chat, acc, kind)
	if errors.Is(err, errNoStaff) {
		return b.SendText(ctx, chat, fmt.Sprintf("Você não tem nenhum %s pra demitir", kind.label()))
	}
	if err != nil {
		return fmt.Errorf("demitir empregado: %w", err)
	}
	text := fmt.Sprintf("📦 @%s demitiu um %s. Ele saiu levando o que tinha achado e ainda não foi coletado", person.User, kind.label())
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) collectStaff(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	now := time.Now()
	decreed, err := b.Wallets.decrees(ctx, chat, now)
	if err != nil {
		return fmt.Errorf("ler decretos: %w", err)
	}
	h, err := b.Wallets.collectStaff(ctx, chat, acc, decreed, now)
	if errors.Is(err, errNoStaff) {
		return b.SendText(ctx, chat, "Você não tem empregado nenhum. Manda !empregados pra ver quem dá pra contratar")
	}
	if err != nil {
		return fmt.Errorf("coletar empregados: %w", err)
	}
	_, err = b.SendMentions(ctx, chat, h.text(person)+decreed.note(gatherDecree), []types.JID{person})
	return err
}

func (h harvest) text(person types.JID) string {
	if h.rounds == 0 && h.strikes == 0 {
		return fmt.Sprintf("👷 Os empregados de @%s ainda não acharam nada. Volta daqui a pouco", person.User)
	}
	lines := []string{fmt.Sprintf("👷 @%s foi ver o que os empregados acharam", person.User)}
	var items []string
	for _, it := range catalog {
		grades, ok := h.found[it.id]
		if !ok {
			continue
		}
		var n int64
		for _, g := range grades {
			n += g.quantity
		}
		items = append(items, fmt.Sprintf("%dx %s", n, it.label()))
	}
	if len(items) > 0 {
		lines = append(lines, "", strings.Join(items, "\n"))
	}
	if len(items) == 0 && h.rounds > 0 {
		lines = append(lines, "", "🫙 Voltaram todos de mão vazia")
	}
	if h.empty > 0 && len(items) > 0 {
		lines = append(lines, fmt.Sprintf("🫙 %d %s de mão vazia", h.empty, pluralWord(h.empty, "vez", "vezes")))
	}
	if h.wages > 0 {
		lines = append(lines, "", fmt.Sprintf("💸 Salário: %s · 💰 Saldo: %s", Coins(h.wages), Coins(h.balance)))
	}
	if h.strikes > 0 {
		lines = append(lines, fmt.Sprintf("🪧 Faltou saldo pro salário e eles fizeram greve: %d %s perdidos", h.strikes, pluralWord(h.strikes, "achado", "achados")))
	}
	return strings.Join(lines, "\n")
}

func (s *walletStore) crew(ctx context.Context, chat types.JID, acc core.Account) ([]hire, error) {
	return loadCrew(ctx, s.db, chat, acc)
}

func loadCrew(ctx context.Context, q core.RowsQuerier, chat types.JID, acc core.Account) (crew []hire, err error) {
	rows, err := q.QueryContext(ctx, "SELECT id, tipo, coletado_em FROM empregados WHERE chat = ? AND pessoa = ? ORDER BY id", chat.String(), acc.Key)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var (
			h     hire
			kind  string
			since int64
		)
		if err := rows.Scan(&h.id, &kind, &since); err != nil {
			return nil, err
		}
		var ok bool
		if h.kind, ok = employeeByID(kind); !ok {
			continue
		}
		h.since = time.Unix(since, 0)
		crew = append(crew, h)
	}
	return crew, rows.Err()
}

func (s *walletStore) estateLevel(ctx context.Context, chat types.JID, acc core.Account) (level int, err error) {
	err = s.db.QueryRowContext(ctx, "SELECT imovel FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&level)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return level, err
}

func (s *walletStore) hireStaff(ctx context.Context, chat types.JID, acc core.Account, kind employee, now time.Time) (room int, err error) {
	err = s.spend(ctx, chat, acc, kind.price, func(tx *ledgerTx) error {
		var level, used int
		err := tx.QueryRowContext(ctx,
			"SELECT imovel, (SELECT COUNT(*) FROM empregados e WHERE e.chat = c.chat AND e.pessoa = c.pessoa) FROM carteiras c WHERE c.chat = ? AND c.pessoa = ?",
			chat.String(), acc.Key,
		).Scan(&level, &used)
		if err != nil {
			return err
		}
		room = estateRoom(level)
		if room == 0 {
			return errNoEstate
		}
		if used >= room {
			return errNoRoom
		}
		_, err = tx.ExecContext(ctx, "INSERT INTO empregados (chat, pessoa, tipo, coletado_em) VALUES (?, ?, ?, ?)",
			chat.String(), acc.Key, kind.id, now.Unix())
		return err
	})
	return room, err
}

func (s *walletStore) fireStaff(ctx context.Context, chat types.JID, acc core.Account, kind employee) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM empregados WHERE id = (
		     SELECT id FROM empregados WHERE chat = ? AND pessoa = ? AND tipo = ? ORDER BY coletado_em DESC LIMIT 1
		 )`,
		chat.String(), acc.Key, kind.id,
	)
	if err != nil {
		return err
	}
	fired, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if fired == 0 {
		return errNoStaff
	}
	return nil
}

func (s *walletStore) collectStaff(ctx context.Context, chat types.JID, acc core.Account, decreed decrees, now time.Time) (h harvest, err error) {
	h.found = map[string][]grade{}
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		crew, err := loadCrew(ctx, tx, chat, acc)
		if err != nil {
			return err
		}
		if len(crew) == 0 {
			return errNoStaff
		}
		if err := tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&h.balance); err != nil {
			return err
		}
		for _, hired := range crew {
			if err := h.work(ctx, tx, hired, decreed, now); err != nil {
				return err
			}
		}
		if h.wages > 0 {
			if err := debitBalance(ctx, tx, chat, acc, h.wages); err != nil {
				return err
			}
			h.balance -= h.wages
		}
		for id, grades := range h.found {
			if err := addGrades(ctx, tx, chat, acc, id, grades); err != nil {
				return err
			}
		}
		return nil
	})
	return h, err
}

func (h *harvest) work(ctx context.Context, tx *ledgerTx, hired hire, decreed decrees, now time.Time) error {
	a := hired.kind.shift(decreed)
	ready := hired.ready(a.cooldown.every, now)
	if ready == 0 {
		return nil
	}
	paid := min(ready, max(h.balance-h.wages, 0)/hired.kind.wage)
	h.rounds += paid
	h.strikes += ready - paid
	h.wages += paid * hired.kind.wage
	for range paid {
		h.keep(a.roll())
	}
	next := hired.since.Add(time.Duration(ready) * a.cooldown.every)
	if ready == hired.kind.stock {
		next = now
	}
	_, err := tx.ExecContext(ctx, "UPDATE empregados SET coletado_em = ? WHERE id = ?", next.Unix(), hired.id)
	return err
}

func (h *harvest) keep(found haul) {
	if found.found.id == "" {
		h.empty++
		return
	}
	grades := h.found[found.found.id]
	i := slices.IndexFunc(grades, func(g grade) bool { return g.quality == found.grade })
	if i < 0 {
		h.found[found.found.id] = append(grades, grade{found.grade, 1})
		return
	}
	grades[i].quantity++
}
