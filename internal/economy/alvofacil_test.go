package economy

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

func jackpotWallets(t *testing.T, ctx context.Context, chat types.JID, people ...string) (*walletStore, []core.Account) {
	t.Helper()
	store, err := core.OpenMessageStore(ctx, filepath.Join(t.TempDir(), "mensagens.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	s := &walletStore{db: &ledgerDB{db: store.DB}}
	accounts := make([]core.Account, len(people))
	for i, n := range people {
		acc := core.Account{Key: n + "@s.whatsapp.net", JID: types.NewJID(n, types.DefaultUserServer)}
		accounts[i] = acc
		err := s.inTx(ctx, func(tx *ledgerTx) error {
			if err := openWallet(ctx, tx, chat, acc); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = 50000 WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return s, accounts
}

func TestJackpotRobbery(t *testing.T) {
	ctx := context.Background()
	chat := types.NewJID("120363000000000001", types.GroupServer)
	s, people := jackpotWallets(t, ctx, chat, "5511900000001", "5511900000002", "5511900000003", "5511900000004", "5511900000005")
	gambler, thief, other, rookie, late := people[0], people[1], people[2], people[3], people[4]
	now := time.Unix(time.Now().Unix(), 0)
	cd := cooldown{column: "roubo_em", every: robEvery}
	seen := int64(-1)
	peek := func(_, _ int64, m easyMark) (bool, int64) { seen = m.jackpot; return true, 100 }

	for _, step := range []struct {
		net    int64
		at     time.Time
		marked int64
	}{
		{jackpotMin - 1, now, 0},
		{3_000, now, 3_000},
		{2_000, now.Add(time.Minute), 0},
	} {
		marked, err := s.noteWinnings(ctx, chat, gambler, step.net, step.at)
		if err != nil {
			t.Fatal(err)
		}
		if marked != step.marked {
			t.Errorf("lucro de %d: o aviso de alvo fácil sai só quando a marca começa (esperava %d, deu %d)", step.net, step.marked, marked)
		}
	}
	robbedAt := now.Add(2 * time.Minute)
	r, err := s.rob(ctx, chat, thief, gambler, cd, nil, crimeRisk{}, flagrant{}, robbedAt, robMinVictim, peek)
	if err != nil || seen != 5_000 || r.mark.jackpot != 5_000 {
		t.Fatalf("os lucros grandes seguidos somam 5.000 e chegam no sorteio do roubo, chegou %d (%+v, %v)", seen, r, err)
	}

	shield := jackpotShield(r.mark.jackpot)
	if shield >= robShield {
		t.Fatalf("o escudo de quem tá marcado(a) (%s) é menor que o normal (%s)", shield, robShield)
	}
	var until int64
	if err := s.db.QueryRowContext(ctx, "SELECT escudo_ate FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), gambler.Key).Scan(&until); err != nil {
		t.Fatal(err)
	}
	if until != robbedAt.Add(shield).Unix() {
		t.Errorf("quem tá marcado(a) fica protegido(a) até %d, ficou até %d", robbedAt.Add(shield).Unix(), until)
	}

	seen = -1
	r, err = s.rob(ctx, chat, rookie, gambler, cd, nil, crimeRisk{}, flagrant{}, robbedAt.Add(shield+time.Second), robMinVictim, peek)
	if err != nil || r.shield != 0 || seen != 5_000 || r.mark.jackpot != 5_000 {
		t.Fatalf("passado o escudo curto, a marca de 5.000 continua valendo, chegou %d (%+v, %v)", seen, r, err)
	}

	if _, err := s.noteWinnings(ctx, chat, other, 4_000, now); err != nil {
		t.Fatal(err)
	}
	seen = -1
	r, err = s.rob(ctx, chat, late, other, cd, nil, crimeRisk{}, flagrant{}, now.Add(jackpotWindow+time.Second), robMinVictim, peek)
	if err != nil || seen != 0 {
		t.Fatalf("passada a janela de %s, o prêmio não conta mais, chegou %d (%+v, %v)", jackpotWindow, seen, r, err)
	}
}

func TestJackpotOnlyInProfit(t *testing.T) {
	ctx := context.Background()
	chat := types.NewJID("120363000000000001", types.GroupServer)
	s, people := jackpotWallets(t, ctx, chat, "5511900000001")
	gambler := people[0]
	now := time.Unix(time.Now().Unix(), 0)

	steps := []struct {
		name   string
		net    int64
		marked int64
		mark   int64
	}{
		{"quem perdeu 20.000 não fica marcado(a)", -20_000, 0, 0},
		{"ganhar 3.000 no prejuízo não marca", 3_000, 0, 0},
		{"ganhar 18.000 e ficar 1.000 no lucro marca com o lucro de verdade", 18_000, 1_000, 1_000},
		{"perder 500 e ficar abaixo do mínimo de lucro tira a marca", -500, 0, 0},
	}
	for i, step := range steps {
		at := now.Add(time.Duration(i) * time.Second)
		marked, err := s.noteWinnings(ctx, chat, gambler, step.net, at)
		if err != nil {
			t.Fatal(err)
		}
		mark, err := markOf(ctx, s.db, chat, gambler, at)
		if err != nil {
			t.Fatal(err)
		}
		if marked != step.marked || mark.jackpot != step.mark {
			t.Errorf("%s: esperava aviso de %d e marca de %d, deu aviso de %d e marca de %d", step.name, step.marked, step.mark, marked, mark.jackpot)
		}
	}
}

func TestLootedFollowsBrackets(t *testing.T) {
	if got, want := looted(100_000, 0, 20), exposed(100_000, 20); got != want {
		t.Errorf("sem prêmio, o roubo segue as faixas (%d), deu %d", want, got)
	}
	if got, want := looted(100_000, 40_000, 20), int64(float64(exposed(100_000, 20))*(1+jackpotLootBonus)); got != want {
		t.Errorf("com prêmio, o saldo todo passa pelas faixas e o total ganha o bônus de alvo fácil (%d), deu %d", want, got)
	}
	if (easyMark{}).odds(0.4) != 0.4 || (easyMark{jackpot: 1}).odds(0.4) != clampChance(0.4+jackpotRobBonus) {
		t.Error("o prêmio fresco soma o bônus na chance, e sem prêmio a chance fica igual")
	}
}

func TestJackpotShortensRobWait(t *testing.T) {
	ctx := context.Background()
	chat := types.NewJID("120363000000000001", types.GroupServer)
	s, people := jackpotWallets(t, ctx, chat, "5511900000001", "5511900000002", "5511900000003", "5511900000004")
	thief, first, plain, gambler := people[0], people[1], people[2], people[3]
	now := time.Unix(time.Now().Unix(), 0)
	cd := cooldown{column: "roubo_em", every: robEvery}
	miss := func(_, _ int64, _ easyMark) (bool, int64) { return false, 0 }

	if _, err := s.rob(ctx, chat, thief, first, cd, nil, crimeRisk{}, flagrant{}, now, robMinVictim, miss); err != nil {
		t.Fatal(err)
	}
	if _, err := s.noteWinnings(ctx, chat, gambler, 5_000, now); err != nil {
		t.Fatal(err)
	}
	short := jackpotPace(cd, 1).every
	if short >= cd.every {
		t.Fatalf("a espera contra quem ganhou na aposta (%s) é menor que a normal (%s)", short, cd.every)
	}
	soon := now.Add(short + time.Second)

	r, err := s.rob(ctx, chat, thief, plain, cd, nil, crimeRisk{}, flagrant{}, soon, robMinVictim, miss)
	if err != nil || r.wait == 0 {
		t.Fatalf("contra quem não ganhou nada, a espera de %s continua valendo (%+v, %v)", cd.every, r, err)
	}
	r, err = s.rob(ctx, chat, thief, gambler, cd, nil, crimeRisk{}, flagrant{}, soon, robMinVictim, miss)
	if err != nil || r.wait != 0 || r.mark.jackpot != 5_000 {
		t.Fatalf("contra quem acabou de ganhar na aposta, dá pra roubar depois de %s (%+v, %v)", short, r, err)
	}
}

func TestJackpotDeath(t *testing.T) {
	ctx := context.Background()
	chat := types.NewJID("120363000000000001", types.GroupServer)
	s, people := jackpotWallets(t, ctx, chat, "5511900000001", "5511900000002", "5511900000003")
	gambler, plain, stale := people[0], people[1], people[2]
	now := time.Unix(time.Now().Unix(), 0)

	for _, acc := range []core.Account{gambler, stale} {
		if _, err := s.noteWinnings(ctx, chat, acc, 20_000, now); err != nil {
			t.Fatal(err)
		}
	}
	bury := func(acc core.Account, at time.Time) int64 {
		var d death
		err := s.inTx(ctx, func(tx *ledgerTx) (err error) {
			d, err = die(ctx, tx, chat, acc, at)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		var balance int64
		if err := s.db.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&balance); err != nil {
			t.Fatal(err)
		}
		if balance != 50_000-d.lost {
			t.Fatalf("o enterro diz que levou %d, mas o saldo ficou %d", d.lost, balance)
		}
		return d.lost
	}

	plainMin, plainMax := int64(50_000*deathLossMinPercent/100), int64(50_000*deathLossMaxPercent/100)
	if lost := bury(plain, now); lost < plainMin || lost > plainMax {
		t.Errorf("sem prêmio, o enterro leva de %d a %d, levou %d", plainMin, plainMax, lost)
	}
	if lost := bury(stale, now.Add(jackpotWindow+time.Second)); lost < plainMin || lost > plainMax {
		t.Errorf("com o prêmio vencido, o enterro é o normal (de %d a %d), levou %d", plainMin, plainMax, lost)
	}
	spilledMin, spilledMax := int64(20_000*jackpotDeathMinPercent/100), int64(20_000*jackpotDeathMaxPercent/100)
	markedMin := spilledMin + (50_000-spilledMax)*deathLossMinPercent/100
	markedMax := spilledMax + (50_000-spilledMin)*deathLossMaxPercent/100
	if lost := bury(gambler, now.Add(time.Minute)); lost < markedMin || lost > markedMax {
		t.Errorf("quem morre como alvo fácil perde parte dos 20.000 do prêmio além do enterro (de %d a %d), levou %d", markedMin, markedMax, lost)
	}
}

func TestLuckyMarkAfterJackpot(t *testing.T) {
	ctx := context.Background()
	chat := types.NewJID("120363000000000001", types.GroupServer)
	s, people := jackpotWallets(t, ctx, chat,
		"5511900000001", "5511900000002", "5511900000003", "5511900000004",
		"5511900000005", "5511900000006", "5511900000007", "5511900000008")
	lucky, faded, loser, fresh := people[0], people[1], people[2], people[3]
	thieves := people[4:]
	now := time.Unix(time.Now().Unix(), 0)
	day := 24 * time.Hour
	cd := cooldown{column: "roubo_em", every: robEvery}

	wins := []struct {
		acc core.Account
		net int64
		at  time.Time
	}{
		{lucky, 6_000, now.Add(-2 * day)},
		{lucky, -500, now.Add(-day)},
		{lucky, 5_000, now.Add(-jackpotWindow - time.Minute)},
		{faded, 20_000, now.Add(-3 * day)},
		{faded, 1_000, now.Add(-jackpotWindow - time.Minute)},
		{loser, 15_000, now.Add(-day)},
		{loser, -6_000, now},
		{fresh, 12_000, now},
	}
	for _, w := range wins {
		if _, err := s.noteWinnings(ctx, chat, w.acc, w.net, w.at); err != nil {
			t.Fatal(err)
		}
	}

	cases := []struct {
		name   string
		victim core.Account
		want   easyMark
	}{
		{"quem lucrou 10.500 em 3 dias fica marcado(a) depois que o prêmio vence", lucky, easyMark{lucky: true, won: 10_500}},
		{"lucro de mais de 3 dias atrás não conta", faded, easyMark{}},
		{"as perdas dos 3 dias descontam do lucro", loser, easyMark{}},
		{"com o prêmio fresco vale só a marca do alvo fácil, no nível 1", fresh, easyMark{jackpot: 12_000, level: 1}},
	}
	for i, c := range cases {
		var got easyMark
		peek := func(_, _ int64, m easyMark) (bool, int64) { got = m; return false, 0 }
		r, err := s.rob(ctx, chat, thieves[i], c.victim, cd, nil, crimeRisk{}, flagrant{}, now.Add(time.Minute), robMinVictim, peek)
		if err != nil || got != c.want || r.mark != c.want {
			t.Errorf("%s: esperava %+v, chegou %+v (%+v, %v)", c.name, c.want, got, r, err)
		}
	}

	if got, want := (easyMark{lucky: true}).odds(0.4), clampChance(0.4+luckyRobBonus); got != want {
		t.Errorf("a marca dos 3 dias soma %.2f na chance (%.2f), deu %.2f", luckyRobBonus, want, got)
	}
	if luckyRobBonus >= jackpotRobBonus {
		t.Errorf("a marca dos 3 dias (%.2f) pesa menos que a do prêmio fresco (%.2f)", luckyRobBonus, jackpotRobBonus)
	}
	if (easyMark{lucky: true}).text(lucky.JID) == "" || (easyMark{}).text(lucky.JID) != "" {
		t.Error("a resposta do roubo avisa da marca dos 3 dias, e sem marca não avisa nada")
	}
}

func TestJackpotArrestSlip(t *testing.T) {
	ctx := context.Background()
	chat := types.NewJID("120363000000000001", types.GroupServer)
	s, people := jackpotWallets(t, ctx, chat, "5511900000001", "5511900000002", "5511900000003")
	cop, gambler, plain := people[0], people[1], people[2]
	now := time.Unix(time.Now().Unix(), 0)

	if _, err := s.noteWinnings(ctx, chat, gambler, 20_000, now); err != nil {
		t.Fatal(err)
	}
	for _, acc := range []core.Account{gambler, plain} {
		if err := s.inTx(ctx, func(tx *ledgerTx) error { return markCrime(ctx, tx, chat, acc, now) }); err != nil {
			t.Fatal(err)
		}
	}
	state := func(acc core.Account) (until, crime int64) {
		err := s.db.QueryRowContext(ctx, "SELECT preso_ate, crime_em FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&until, &crime)
		if err != nil {
			t.Fatal(err)
		}
		return until, crime
	}
	at := now.Add(time.Minute)
	arrestAt := func(suspect core.Account, when time.Time) arrest {
		a, err := s.arrest(ctx, chat, cop, suspect, cooldown{column: "prisao_em"}, nil, baseJail, arrestOdds{chance: 1, slip: 1}, false, true, when)
		if err != nil {
			t.Fatal(err)
		}
		return a
	}

	if a := arrestAt(gambler, at); !a.lawyered {
		t.Fatalf("com o prêmio fresco e o advogado sempre acertando, a prisão não cola (%+v)", a)
	}
	if until, crime := state(gambler); until != 0 || crime != now.Unix() {
		t.Errorf("quem escapou com o advogado continua solto e com o crime na ficha, ficou preso_ate %d e crime_em %d", until, crime)
	}
	if a := arrestAt(plain, at); a.lawyered || a.dismissed {
		t.Fatalf("sem prêmio de aposta, o advogado não aparece e a prisão cola (%+v)", a)
	}
	if until, crime := state(plain); until != at.Add(baseJail).Unix() || crime != 0 {
		t.Errorf("quem não é alvo fácil vai preso por %s e tem o crime zerado, ficou preso_ate %d e crime_em %d", baseJail, until, crime)
	}
	late := now.Add(jackpotWindow + time.Second)
	if err := s.inTx(ctx, func(tx *ledgerTx) error { return markCrime(ctx, tx, chat, gambler, late) }); err != nil {
		t.Fatal(err)
	}
	if a := arrestAt(gambler, late); a.lawyered || a.clean {
		t.Errorf("com o prêmio vencido, o advogado não aparece mais e a prisão cola (%+v)", a)
	}
}

func TestJackpotShieldShrinksWithWinnings(t *testing.T) {
	if jackpotShield(0) != robShield {
		t.Fatalf("sem prêmio o escudo é o normal (%s), deu %s", robShield, jackpotShield(0))
	}
	smallest := time.Duration(float64(robShield) * (1 - jackpotShieldMinCut)).Round(time.Second)
	if got := jackpotShield(jackpotMin); got != smallest {
		t.Errorf("com o prêmio mínimo o escudo cai %.0f%% (%s), deu %s", jackpotShieldMinCut*100, smallest, got)
	}
	floor := time.Duration(float64(robShield) * (1 - jackpotShieldMaxCut)).Round(time.Second)
	for _, jackpot := range []int64{jackpotShieldTop, jackpotShieldTop * 10} {
		if got := jackpotShield(jackpot); got != floor {
			t.Errorf("com %d de prêmio o escudo para no mínimo (%s), deu %s", jackpot, floor, got)
		}
	}
	prev := jackpotShield(jackpotMin)
	for _, jackpot := range []int64{5_000, 10_000, 100_000, 500_000} {
		got := jackpotShield(jackpot)
		if got >= prev || got <= floor {
			t.Errorf("com %d de prêmio o escudo (%s) fica entre o do prêmio menor (%s) e o mínimo (%s)", jackpot, got, prev, floor)
		}
		prev = got
	}
}
