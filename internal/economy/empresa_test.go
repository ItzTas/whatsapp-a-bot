package economy

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

type firmBench struct {
	ctx  context.Context
	s    *walletStore
	chat types.JID
	now  time.Time
}

func newFirmBench(t *testing.T) firmBench {
	t.Helper()
	ctx := context.Background()
	store, err := core.OpenMessageStore(ctx, filepath.Join(t.TempDir(), "mensagens.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return firmBench{
		ctx:  ctx,
		s:    &walletStore{db: &ledgerDB{db: store.DB}},
		chat: types.NewJID("120363000000000001", types.GroupServer),
		now:  time.Unix(time.Now().Unix(), 0),
	}
}

func (fb firmBench) person(n string) core.Account {
	return core.Account{Key: n + "@s.whatsapp.net", JID: types.NewJID(n, types.DefaultUserServer)}
}

func (fb firmBench) exec(t *testing.T, query string, args ...any) {
	t.Helper()
	if _, err := fb.s.db.ExecContext(fb.ctx, query, args...); err != nil {
		t.Fatal(err)
	}
}

func (fb firmBench) setBalance(t *testing.T, acc core.Account, saldo int64) {
	t.Helper()
	err := fb.s.inTx(fb.ctx, func(tx *ledgerTx) error {
		if err := openWallet(fb.ctx, tx, fb.chat, acc); err != nil {
			return err
		}
		_, err := tx.ExecContext(fb.ctx, "UPDATE carteiras SET saldo = ? WHERE chat = ? AND pessoa = ?", saldo, fb.chat.String(), acc.Key)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func (fb firmBench) balance(t *testing.T, acc core.Account) int64 {
	t.Helper()
	var v int64
	if err := fb.s.db.QueryRowContext(fb.ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", fb.chat.String(), acc.Key).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func (fb firmBench) markJackpot(t *testing.T, acc core.Account, prize int64) {
	t.Helper()
	fb.exec(t, "UPDATE carteiras SET premio_aposta = ?, premio_aposta_em = ? WHERE chat = ? AND pessoa = ?", prize, fb.now.Unix(), fb.chat.String(), acc.Key)
	fb.exec(t, "INSERT INTO apostas_lucro (chat, pessoa, dia, lucro) VALUES (?, ?, ?, ?)", fb.chat.String(), acc.Key, fb.now.In(core.BrazilTime).Format(time.DateOnly), prize)
}

func (fb firmBench) contract(t *testing.T, acc core.Account, f firm, boost time.Time) {
	t.Helper()
	fb.exec(t, "INSERT INTO contratos (chat, pessoa, ramo, empresa, ate, reforco_ate) VALUES (?, ?, ?, ?, ?, ?)",
		fb.chat.String(), acc.Key, f.sector, f.id, fb.now.Add(contractSpan).Unix(), boost.Unix())
}

func mustFirm(t *testing.T, id string) firm {
	t.Helper()
	f, ok := firmByID(id)
	if !ok {
		t.Fatalf("empresa %s não existe no catálogo", id)
	}
	return f
}

func TestSignFirm(t *testing.T) {
	fb := newFirmBench(t)
	vigia, escolta := mustFirm(t, "vigia-de-rua"), mustFirm(t, "escolta-armada")

	poor := fb.person("5511900000001")
	fb.setBalance(t, poor, 10_000)
	h, err := fb.s.sign(fb.ctx, fb.chat, poor, vigia, fb.now)
	if err != nil || h.fee != vigia.floor || h.urgency != 0 || !h.until.Equal(fb.now.Add(contractSpan)) {
		t.Fatalf("com 10.000 a diária do Vigia é o piso e vale 24h, deu %+v (%v)", h, err)
	}
	if got := fb.balance(t, poor); got != 10_000-vigia.floor {
		t.Fatalf("o piso sai do saldo, ficou %d", got)
	}

	renewed, err := fb.s.sign(fb.ctx, fb.chat, poor, vigia, fb.now.Add(time.Hour))
	if err != nil || !renewed.until.Equal(h.until.Add(contractSpan)) {
		t.Fatalf("contratar a mesma empresa soma 24h no fim do contrato, deu %+v (%v)", renewed, err)
	}
	switched, err := fb.s.sign(fb.ctx, fb.chat, poor, escolta, fb.now.Add(2*time.Hour))
	c, found, cerr := contractOf(fb.ctx, fb.s.db, fb.chat, poor, fb.now.Add(2*time.Hour))
	if err != nil || cerr != nil || !found || c.firm.id != escolta.id || !switched.until.Equal(fb.now.Add(2*time.Hour+contractSpan)) {
		t.Fatalf("trocar de empresa começa um contrato novo de 24h com ela, deu %+v %+v (%v %v)", switched, c, err, cerr)
	}

	rich := fb.person("5511900000002")
	fb.setBalance(t, rich, 100_000)
	if h, err := fb.s.sign(fb.ctx, fb.chat, rich, escolta, fb.now); err != nil || h.fee != 100_000*escolta.rate/100 {
		t.Fatalf("acima do piso a diária é a porcentagem do saldo, deu %+v (%v)", h, err)
	}

	lucky := fb.person("5511900000003")
	fb.setBalance(t, lucky, 50_000)
	fb.markJackpot(t, lucky, 20_000)
	if h, err := fb.s.sign(fb.ctx, fb.chat, lucky, vigia, fb.now); err != nil || h.fee != 50_000*vigia.rate/100 || h.urgency != 20_000*emergencyCut/100 {
		t.Fatalf("com prêmio fresco a taxa de emergência é uma fatia do prêmio, deu %+v (%v)", h, err)
	}

	broke := fb.person("5511900000004")
	fb.setBalance(t, broke, 100)
	h, err = fb.s.sign(fb.ctx, fb.chat, broke, vigia, fb.now)
	_, found, cerr = contractOf(fb.ctx, fb.s.db, fb.chat, broke, fb.now)
	if !errors.Is(err, ErrNoFunds) || h.total() != vigia.floor || found || cerr != nil || fb.balance(t, broke) != 100 {
		t.Fatalf("sem saldo pra diária não contrata nem cobra, deu %+v found=%v (%v %v)", h, found, err, cerr)
	}
}

func TestCancelContract(t *testing.T) {
	fb := newFirmBench(t)
	acc := fb.person("5511900000001")
	fb.setBalance(t, acc, 10_000)
	escolta, firma := mustFirm(t, "escolta-armada"), mustFirm(t, "firma-do-157")
	fb.contract(t, acc, escolta, time.Time{})
	fb.contract(t, acc, firma, time.Time{})

	f, err := fb.s.cancelContract(fb.ctx, fb.chat, acc, securitySector, fb.now)
	if err != nil || f.id != escolta.id {
		t.Fatalf("cancelar devolve a empresa do contrato daquele ramo, deu %+v (%v)", f, err)
	}
	if fb.balance(t, acc) != 10_000 {
		t.Fatal("cancelar não devolve dinheiro nenhum")
	}
	if _, err := fb.s.cancelContract(fb.ctx, fb.chat, acc, securitySector, fb.now); !errors.Is(err, errNoContract) {
		t.Fatalf("sem contrato ativo no ramo não tem o que cancelar, deu %v", err)
	}
	if contracts, err := fb.s.contracts(fb.ctx, fb.chat, acc, fb.now); err != nil || len(contracts) != 1 || contracts[0].firm.id != firma.id {
		t.Fatalf("o contrato do outro ramo continua, deu %+v (%v)", contracts, err)
	}
}

func TestEscortOdds(t *testing.T) {
	now := time.Now()
	soon := now.Add(time.Minute)
	jackpot, lucky := easyMark{jackpot: 5_000}, easyMark{lucky: true, won: luckyMin}
	escolta, forte := mustFirm(t, "escolta-armada"), mustFirm(t, "carro-forte")
	cases := []struct {
		name string
		c    contract
		m    easyMark
		want float64
	}{
		{"prêmio fresco usa a chance cheia", contract{firm: escolta}, jackpot, escolta.annul},
		{"marca de quem lucrou esses dias usa metade", contract{firm: escolta}, lucky, escolta.annul / 2},
		{"reforço soma na chance", contract{firm: escolta, boost: soon}, jackpot, escolta.annul + boostOdds},
		{"a chance tem teto", contract{firm: forte, boost: soon}, jackpot, escortOddsMax},
	}
	for _, tc := range cases {
		if got := tc.c.odds(tc.m, now); got < tc.want-1e-9 || got > tc.want+1e-9 {
			t.Errorf("%s: queria %.2f, deu %.2f", tc.name, tc.want, got)
		}
	}
}

func TestEscortMark(t *testing.T) {
	fb := newFirmBench(t)
	marked := easyMark{jackpot: 5_000}
	client, stranger := fb.person("5511900000001"), fb.person("5511900000002")
	fb.setBalance(t, client, 10_000)
	fb.setBalance(t, stranger, 10_000)
	forte := mustFirm(t, "carro-forte")
	fb.contract(t, client, forte, fb.now.Add(time.Hour))

	if m, f, err := escortMark(fb.ctx, fb.s.db, fb.chat, client, easyMark{}, false, fb.now); err != nil || m != (easyMark{}) || f.id != "" {
		t.Fatalf("sem marca a empresa não entra, deu %+v %+v (%v)", m, f, err)
	}
	if m, f, err := escortMark(fb.ctx, fb.s.db, fb.chat, stranger, marked, false, fb.now); err != nil || m != marked || f.id != "" {
		t.Fatalf("sem contrato a marca fica, deu %+v %+v (%v)", m, f, err)
	}
	for range 200 {
		m, f, err := escortMark(fb.ctx, fb.s.db, fb.chat, client, marked, true, fb.now)
		if err != nil {
			t.Fatal(err)
		}
		if f.id == "" {
			continue
		}
		if m != (easyMark{}) || f.id != forte.id {
			t.Fatalf("quando a empresa entra a marca some e ela aparece no roubo, deu %+v %+v", m, f)
		}
		return
	}
	t.Fatal("o Carro-forte com reforço nunca tirou a marca em 200 roubos")
}

func TestEscortSkipsGangsForStreetWatch(t *testing.T) {
	fb := newFirmBench(t)
	marked := easyMark{jackpot: 5_000}
	client := fb.person("5511900000001")
	fb.setBalance(t, client, 10_000)
	fb.contract(t, client, mustFirm(t, "vigia-de-rua"), fb.now.Add(time.Hour))
	for range 50 {
		if m, f, err := escortMark(fb.ctx, fb.s.db, fb.chat, client, marked, true, fb.now); err != nil || m != marked || f.id != "" {
			t.Fatalf("o Vigia de rua não cobre !quadrilha, deu %+v %+v (%v)", m, f, err)
		}
	}
}

func TestEventCushion(t *testing.T) {
	fb := newFirmBench(t)
	escolta := mustFirm(t, "escolta-armada")
	const start = 100_000
	run := func(acc core.Account, place string, haul int64) *eventRun {
		return &eventRun{chat: fb.chat, acc: acc, place: place, haul: haul, now: fb.now}
	}
	play := func(t *testing.T, e effect, r *eventRun) string {
		t.Helper()
		var line string
		err := fb.s.inTx(fb.ctx, func(tx *ledgerTx) (err error) {
			line, err = e.apply(fb.ctx, tx, r)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		return line
	}

	client := fb.person("5511900000001")
	fb.setBalance(t, client, start)
	fb.contract(t, client, escolta, time.Time{})
	bolada := run(client, windfallPlace, 20_000)
	lost := int64(float64(start*10/100) * bolada.spot().bite())
	covered := lost * escolta.cover / 100
	line := play(t, lose(10, 10), bolada)
	if got := fb.balance(t, client); got != start-lost+covered || !strings.Contains(line, "segurou "+Coins(covered)) {
		t.Fatalf("na bolada a empresa segura parte do lose, saldo %d e linha %q", got, line)
	}

	fb.setBalance(t, client, start)
	stalk := run(client, stalkPlace, 20_000)
	skimmed := int64(float64(20_000*10/100) * stalk.spot().bite())
	covered = skimmed * escolta.cover / 100
	line = play(t, skim(10, 10), stalk)
	if got := fb.balance(t, client); got != start-skimmed+covered || !strings.Contains(line, "segurou "+Coins(covered)) {
		t.Fatalf("no alvo fácil a empresa segura parte do skim, saldo %d e linha %q", got, line)
	}

	fb.setBalance(t, client, start)
	line = play(t, lose(10, 10), run(client, "!trabalhar", 0))
	if got := fb.balance(t, client); got != start-start*10/100 || strings.Contains(line, "segurou") {
		t.Fatalf("fora dos eventos de quem ganhou na aposta a empresa não cobre nada, saldo %d e linha %q", got, line)
	}

	stranger := fb.person("5511900000002")
	fb.setBalance(t, stranger, start)
	play(t, lose(10, 10), run(stranger, windfallPlace, 20_000))
	if got := fb.balance(t, stranger); got != start-lost {
		t.Fatalf("sem contrato a perda da bolada vem inteira, saldo %d", got)
	}
}

func TestBoostContract(t *testing.T) {
	fb := newFirmBench(t)
	vigia, escolta := mustFirm(t, "vigia-de-rua"), mustFirm(t, "escolta-armada")
	until := fb.now.Add(jackpotWindow)
	offer := func(acc core.Account, f firm) *boostOffer {
		return &boostOffer{chat: fb.chat, acc: acc, firm: f, price: 3_000, until: until, expires: fb.now.Add(offerWindow)}
	}

	client := fb.person("5511900000001")
	fb.setBalance(t, client, 10_000)
	fb.contract(t, client, vigia, time.Time{})
	if err := fb.s.boost(fb.ctx, offer(client, vigia), fb.now); err != nil {
		t.Fatal(err)
	}
	c, found, err := contractOf(fb.ctx, fb.s.db, fb.chat, client, fb.now)
	if err != nil || !found || c.firm.id != vigia.id || !c.until.Equal(fb.now.Add(contractSpan)) || !c.boost.Equal(until) || fb.balance(t, client) != 7_000 {
		t.Fatalf("o reforço de quem já é cliente mantém a empresa e o contrato e vale até a marca acabar, deu %+v found=%v (%v)", c, found, err)
	}

	stranger := fb.person("5511900000002")
	fb.setBalance(t, stranger, 10_000)
	if err := fb.s.boost(fb.ctx, offer(stranger, escolta), fb.now); err != nil {
		t.Fatal(err)
	}
	c, found, err = contractOf(fb.ctx, fb.s.db, fb.chat, stranger, fb.now)
	if err != nil || !found || c.firm.id != escolta.id || !c.until.Equal(until) || !c.boost.Equal(until) {
		t.Fatalf("quem não é cliente ganha a escolta até a marca acabar, deu %+v found=%v (%v)", c, found, err)
	}

	broke := fb.person("5511900000003")
	fb.setBalance(t, broke, 100)
	err = fb.s.boost(fb.ctx, offer(broke, escolta), fb.now)
	_, found, cerr := contractOf(fb.ctx, fb.s.db, fb.chat, broke, fb.now)
	if !errors.Is(err, ErrNoFunds) || found || cerr != nil {
		t.Fatalf("sem saldo o reforço não sai, deu found=%v (%v %v)", found, err, cerr)
	}
}

func TestRecoverRobbery(t *testing.T) {
	fb := newFirmBench(t)
	victim, thief := fb.person("5511900000001"), fb.person("5511900000002")
	o := &recoveryOffer{chat: fb.chat, victim: victim, thief: thief, firm: mustFirm(t, "escolta-armada"), amount: 5_000, price: 1_250}
	outcomes := map[bool]bool{}
	for range 200 {
		fb.setBalance(t, victim, 10_000)
		fb.setBalance(t, thief, 3_000)
		back, found, err := fb.s.recover(fb.ctx, o)
		if err != nil {
			t.Fatal(err)
		}
		want := map[bool]int64{true: 3_000, false: 0}[found]
		if back != want || fb.balance(t, victim) != 10_000-o.price+back || fb.balance(t, thief) != 3_000-back {
			t.Fatalf("a recuperação cobra adiantado e, quando acha o ladrão, traz o que sobrou com ele; found=%v back=%d vítima=%d ladrão=%d",
				found, back, fb.balance(t, victim), fb.balance(t, thief))
		}
		if outcomes[found] = true; len(outcomes) == 2 {
			break
		}
	}
	if !outcomes[true] {
		t.Fatal("a empresa nunca achou o ladrão em 200 tentativas")
	}

	fb.setBalance(t, victim, 100)
	if _, _, err := fb.s.recover(fb.ctx, o); !errors.Is(err, ErrNoFunds) || fb.balance(t, victim) != 100 {
		t.Fatalf("sem saldo pra pagar a recuperação ela não sai, deu %v", err)
	}
}

func TestRecoveryCut(t *testing.T) {
	cuts := map[string]struct {
		m    easyMark
		want int64
	}{
		"sem marca":         {easyMark{}, 25},
		"marca fraca":       {easyMark{lucky: true}, 35},
		"com prêmio fresco": {easyMark{jackpot: 5_000}, 45},
	}
	for name, c := range cuts {
		if got := recoveryCut(c.m); got != c.want {
			t.Errorf("%s: queria %d%%, deu %d%%", name, c.want, got)
		}
	}
}

func TestTakeOfferExpires(t *testing.T) {
	var offers sync.Map
	now := time.Now()
	offers.Store("viva", &boostOffer{expires: now.Add(time.Minute)})
	offers.Store("vencida", &boostOffer{expires: now.Add(-time.Second)})
	if _, ok := takeOffer[*boostOffer](&offers, "viva", now); !ok {
		t.Fatal("oferta dentro do prazo é aceita")
	}
	if _, ok := takeOffer[*boostOffer](&offers, "vencida", now); ok {
		t.Fatal("oferta vencida não vale mais")
	}
	if _, ok := offers.Load("vencida"); ok {
		t.Fatal("oferta vencida sai da lista")
	}
}
