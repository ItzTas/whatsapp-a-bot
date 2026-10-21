package main

import (
	"cmp"
	"fmt"
	"maps"
	"math/rand/v2"
	"os"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
)

const (
	paceStep       = time.Hour
	paceAwake      = 16
	paceEventChain = 4
)

type step func(r *rig, p *player)

type role struct {
	label string
	job   string
	steps []step
}

type player struct {
	person
	role
	instrument string
	equipped   bool
	worth      int64
	sources    map[string]int64
}

type clockColumn struct {
	table  string
	column string
}

type rig struct {
	*bench
	chat    types.JID
	players []*player
	clock   []clockColumn
}

func TestEnrichmentPace(t *testing.T) {
	span, err := time.ParseDuration(os.Getenv("ENRIQUECIMENTO"))
	if err != nil || span < paceStep {
		t.Skip("a simulação de enriquecimento só roda com a duração no ambiente: ENRIQUECIMENTO=72h go test -run TestEnrichmentPace -v .")
	}
	worker := []step{daily, equip, work}
	addict := []step{daily, equip, work, gather, save}
	gambler := []step{daily, work, bet}
	thief := []step{daily, work, rob}

	jobs := professionNames(t)
	shards := make([][]role, min(runtime.GOMAXPROCS(0), len(jobs)))
	for i, job := range jobs {
		shards[i%len(shards)] = append(shards[i%len(shards)], role{"trabalhador", job, worker})
	}
	shards = append(shards,
		[]role{{"viciado", "trader", addict}, {"viciado", "bico", addict}},
		[]role{{"apostador", "bico", gambler}, {"apostador", "bico", gambler}},
		[]role{{"ladrão", "bandido", thief}, {"vítima", "bico", worker}, {"vítima", "bico", worker}},
	)
	rigs := make([]*rig, len(shards))
	for i, roles := range shards {
		rigs[i] = newRig(t, i, roles)
	}

	var g errgroup.Group
	for _, r := range rigs {
		g.Go(func() error { return r.live(int(span / paceStep)) })
	}
	if err := g.Wait(); err != nil {
		t.Fatal(err)
	}
	t.Log(paceReport(rigs, span))
}

func professionNames(t *testing.T) []string {
	t.Helper()
	b := newBench(t)
	ana := newPerson("Ana", "5511980000001")
	chat := b.group("Profissões", ana)
	b.setup(chat, "economia")
	lines := strings.Split(wantReply(t, b.say(chat, ana, "!profissao"), "profissões*"), "\n")
	var names []string
	for i, line := range lines[:len(lines)-1] {
		if !strings.Contains(line, "(!profissao ") {
			continue
		}
		for entry := range strings.SplitSeq(lines[i+1], ", ") {
			_, name, _ := strings.Cut(entry, " ")
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		t.Fatalf("o menu do !profissao não trouxe nenhuma profissão: %q", lines)
	}
	return names
}

func newRig(t *testing.T, n int, roles []role) *rig {
	t.Helper()
	instrumentNamed := regexp.MustCompile(`🛠️ Com \S+ \*([^*]+)\* na mochila`)
	r := &rig{bench: newBench(t)}
	people := make([]person, len(roles))
	for i := range roles {
		people[i] = newPerson(fmt.Sprintf("Jogador %d", i+1), fmt.Sprintf("5511%03d%05d", n, i+1))
	}
	r.chat = r.group(fmt.Sprintf("Simulação %d", n+1), people...)
	r.setup(r.chat, "economia")
	for i, role := range roles {
		p := &player{person: people[i], role: role, sources: map[string]int64{}}
		reply := wantReply(t, r.say(r.chat, p.person, "!profissao "+role.job), role.job)
		if found := instrumentNamed.FindStringSubmatch(reply); found != nil {
			p.instrument = found[1]
		}
		r.players = append(r.players, p)
	}
	r.clock = clockColumns(t, r.bench)
	return r
}

func clockColumns(t *testing.T, b *bench) []clockColumn {
	t.Helper()
	rows, err := b.db.QueryContext(b.ctx,
		`SELECT m.name, c.name FROM sqlite_master m, pragma_table_info(m.name) c
		 WHERE m.type = 'table' AND c.type LIKE 'INT%'
		  AND (c.name = 'em' OR c.name LIKE '%\_em' ESCAPE '\' OR c.name LIKE '%\_ate' ESCAPE '\')`)
	if err != nil {
		t.Fatalf("achar as colunas de tempo: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var columns []clockColumn
	for rows.Next() {
		var c clockColumn
		if err := rows.Scan(&c.table, &c.column); err != nil {
			t.Fatalf("ler as colunas de tempo: %v", err)
		}
		columns = append(columns, c)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("ler as colunas de tempo: %v", err)
	}
	return columns
}

func (r *rig) live(hours int) error {
	for hour := range hours {
		if hour%24 < paceAwake {
			for _, p := range r.players {
				for _, s := range p.steps {
					s(r, p)
				}
			}
		}
		if err := r.elapse(paceStep); err != nil {
			return err
		}
	}
	for _, p := range r.players {
		worth, err := r.worthOf(p.person)
		if err != nil {
			return err
		}
		p.worth = worth
		var explained int64
		for _, gain := range p.sources {
			explained += gain
		}
		p.sources["outros"] = worth - explained
	}
	return nil
}

func (r *rig) elapse(d time.Duration) error {
	tx, err := r.db.BeginTx(r.ctx, nil)
	if err != nil {
		return fmt.Errorf("adiantar o relógio: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, c := range r.clock {
		query := fmt.Sprintf("UPDATE %[1]q SET %[2]q = %[2]q - ? WHERE %[2]q > 0", c.table, c.column)
		if _, err := tx.ExecContext(r.ctx, query, int64(d.Seconds())); err != nil {
			return fmt.Errorf("adiantar o relógio em %s.%s: %w", c.table, c.column, err)
		}
	}
	return tx.Commit()
}

func (r *rig) worthOf(p person) (int64, error) {
	var worth int64
	err := r.db.QueryRowContext(r.ctx,
		`SELECT COALESCE(SUM(c.saldo + c.cofre_casa + COALESCE(s.valor, 0)), 0)
		 FROM carteiras c LEFT JOIN poupancas s ON s.chat = c.chat AND s.pessoa = c.pessoa
		 WHERE c.pessoa = ?`,
		p.jid.User,
	).Scan(&worth)
	if err != nil {
		return 0, fmt.Errorf("ler o patrimônio de %s: %w", p.name, err)
	}
	return worth, nil
}

func (r *rig) act(p *player, text string, mentions ...person) []string {
	command, _, _ := strings.Cut(text, " ")
	before := r.worth(p)
	start := r.net.count()
	replies := r.say(r.chat, p.person, text, mentions...)
	after := r.worth(p)
	p.sources[command] += after - before
	for range paceEventChain {
		poll, ok := r.pollSince(start)
		if !ok {
			break
		}
		start, before = r.net.count(), after
		pick := strconv.Itoa(rand.IntN(len(poll.message.GetPollCreationMessage().GetOptions())) + 1)
		replies = append(replies, r.quote(r.chat, p.person, pick, &waE2E.ContextInfo{StanzaID: proto.String(string(poll.id))})...)
		after = r.worth(p)
		p.sources["evento"] += after - before
	}
	return replies
}

func (r *rig) worth(p *player) int64 {
	worth, err := r.worthOf(p.person)
	if err != nil {
		r.t.Error(err)
	}
	return worth
}

func (r *rig) pollSince(start int) (sentMessage, bool) {
	sent := r.net.since(start)
	i := slices.IndexFunc(sent, func(m sentMessage) bool { return m.chat == r.chat && m.message.GetPollCreationMessage() != nil })
	if i < 0 {
		return sentMessage{}, false
	}
	return sent[i], true
}

func daily(r *rig, p *player) {
	r.act(p, "!daily")
}

func work(r *rig, p *player) {
	r.act(p, "!trabalhar")
}

func equip(r *rig, p *player) {
	if p.instrument == "" || p.equipped {
		return
	}
	p.equipped = slices.ContainsFunc(r.act(p, "!comprar "+p.instrument), purchased)
}

func purchased(reply string) bool {
	return strings.HasPrefix(reply, "🛒 @") && strings.Contains(reply, " comprou ")
}

func gather(r *rig, p *player) {
	tools := map[string]string{"vara-de-pesca": "vara de pesca", "picareta": "picareta", "rede-de-insetos": "rede de insetos"}
	for id, name := range tools {
		if r.held(p.person, id) == 0 {
			r.act(p, "!comprar "+name)
		}
	}
	for _, command := range []string{"!pescar", "!minerar", "!vasculhar", "!catar", "!vender bugigangas"} {
		r.act(p, command)
	}
}

func save(r *rig, p *player) {
	r.act(p, "!poupar")
	r.act(p, "!poupar metade")
}

func bet(r *rig, p *player) {
	r.act(p, "!apostar tudo")
}

func rob(r *rig, p *player) {
	var (
		target *player
		best   int64 = -1
	)
	for _, other := range r.players {
		if other == p {
			continue
		}
		if balance := r.balance(other.person); balance > best {
			target, best = other, balance
		}
	}
	r.act(p, "!roubar "+mentionTag(target.person), target.person)
	r.act(p, "!furtar "+mentionTag(target.person), target.person)
}

func paceReport(rigs []*rig, span time.Duration) string {
	var all []*player
	for _, r := range rigs {
		all = append(all, r.players...)
	}
	slices.SortFunc(all, func(a, b *player) int { return cmp.Compare(b.worth, a.worth) })
	hours, days := span.Hours(), span.Hours()/24
	lines := []string{
		fmt.Sprintf("Patrimônio (saldo + cofre + poupança) depois de %s, acordado %dh por dia e passando de hora em hora:", span, paceAwake),
		fmt.Sprintf("%4s  %-12s %-24s %12s %9s %10s", "#", "perfil", "profissão", "total", "por hora", "por dia"),
	}
	byLabel := map[string][]*player{}
	for i, p := range all {
		lines = append(lines, fmt.Sprintf("%4d  %-12s %-24s %12s %9s %10s", i+1, p.label, p.job,
			paceCoins(p.worth), paceCoins(int64(float64(p.worth)/hours)), paceCoins(int64(float64(p.worth)/days))))
		byLabel[p.label] = append(byLabel[p.label], p)
	}
	lines = append(lines, "", "Por perfil, por dia: a mediana do patrimônio e, em média por jogador, de onde ele veio:")
	for _, label := range slices.Sorted(maps.Keys(byLabel)) {
		players := byLabel[label]
		sources := map[string]int64{}
		for _, p := range players {
			for source, gain := range p.sources {
				sources[source] += gain
			}
		}
		perDay := func(n int64) string { return paceCoins(int64(float64(n) / float64(len(players)) / days)) }
		var parts []string
		for _, source := range slices.SortedFunc(maps.Keys(sources), func(a, b string) int { return cmp.Compare(sources[b], sources[a]) }) {
			if sources[source] != 0 {
				parts = append(parts, source+" "+perDay(sources[source]))
			}
		}
		lines = append(lines, fmt.Sprintf("  %-12s %10s (%d jogadores): %s", label,
			paceCoins(int64(float64(players[len(players)/2].worth)/days)), len(players), strings.Join(parts, ", ")))
	}
	return "\n" + strings.Join(lines, "\n")
}

func paceCoins(n int64) string {
	if n < 0 {
		return "-" + core.Thousands(-n)
	}
	return core.Thousands(n)
}
