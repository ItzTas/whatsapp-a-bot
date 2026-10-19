package economy

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

func TestRobberyGuards(t *testing.T) {
	ctx := context.Background()
	store, err := core.OpenMessageStore(ctx, filepath.Join(t.TempDir(), "mensagens.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	s := &walletStore{db: &ledgerDB{db: store.DB}}
	chat := types.NewJID("120363000000000001", types.GroupServer)
	person := func(n string) core.Account {
		return core.Account{Key: n + "@s.whatsapp.net", JID: types.NewJID(n, types.DefaultUserServer)}
	}
	victim, rich, broke, elusive := person("5511900000001"), person("5511900000002"), person("5511900000003"), person("5511900000004")
	setBalance := func(t *testing.T, acc core.Account, saldo int64) {
		t.Helper()
		err := s.inTx(ctx, func(tx *ledgerTx) error {
			if err := openWallet(ctx, tx, chat, acc); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = ? WHERE chat = ? AND pessoa = ?", saldo, chat.String(), acc.Key)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	column := func(t *testing.T, acc core.Account, name string) int64 {
		t.Helper()
		var v int64
		if err := s.db.QueryRowContext(ctx, "SELECT "+name+" FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	cd := cooldown{column: "roubo_em", every: robEvery}
	jail := flagrant{odds: 1, jail: baseJail}
	succeed := func(int64, int64) (bool, int64) { return true, 1_000 }
	fail := func(thief, _ int64) (bool, int64) { return false, thief / 10 }
	setBalance(t, victim, 80_000)
	setBalance(t, rich, 5_000)
	setBalance(t, broke, brokeThief-1)
	setBalance(t, elusive, brokeThief-1)
	now := time.Unix(time.Now().Unix(), 0)

	r, err := s.rob(ctx, chat, rich, victim, cd, nil, crimeRisk{}, jail, now, robMinVictim, succeed)
	if err != nil || !r.success || r.amount != 1_000 {
		t.Fatalf("o primeiro roubo dá certo e leva 1.000, deu %+v (%v)", r, err)
	}
	if got := column(t, victim, "escudo_ate"); got != now.Add(robShield).Unix() {
		t.Errorf("a vítima fica protegida até %d, ficou até %d", now.Add(robShield).Unix(), got)
	}

	later := now.Add(robShield / 2)
	r, err = s.rob(ctx, chat, broke, victim, cd, nil, crimeRisk{}, jail, later, robMinVictim, succeed)
	if err != nil || r.shield != robShield/2 || r.success {
		t.Fatalf("durante o escudo o roubo é barrado com %s restantes, deu %+v (%v)", robShield/2, r, err)
	}
	if got := column(t, victim, "saldo"); got != 79_000 {
		t.Errorf("o roubo barrado não mexe no saldo da vítima (79.000), ficou %d", got)
	}
	if column(t, broke, "roubo_em") != 0 || column(t, broke, "crime_em") != 0 {
		t.Error("o roubo barrado pelo escudo não gasta a espera nem marca crime do ladrão")
	}

	after := now.Add(robShield + time.Second)
	r, err = s.rob(ctx, chat, rich, victim, cd, nil, crimeRisk{}, jail, after.Add(robEvery), robMinVictim, fail)
	if err != nil || r.shield != 0 || r.success || r.jail != 0 || r.amount != 600 {
		t.Fatalf("depois do escudo, ladrão com saldo que falha paga 10%% do saldo (600) e não vai preso, deu %+v (%v)", r, err)
	}

	r, err = s.rob(ctx, chat, broke, victim, cd, nil, crimeRisk{}, jail, after, robMinVictim, fail)
	if err != nil || r.jail != baseJail {
		t.Fatalf("ladrão com menos de %d que falha vai preso por %s, deu %+v (%v)", brokeThief, baseJail, r, err)
	}
	if got := column(t, broke, "preso_ate"); got != after.Add(baseJail).Unix() {
		t.Errorf("o ladrão liso fica preso até %d, ficou até %d", after.Add(baseJail).Unix(), got)
	}
	if column(t, broke, "crime_em") != 0 || column(t, broke, "ficha_em") != after.Unix() {
		t.Error("a prisão zera o crime_em, e a ficha guarda a hora do crime")
	}

	r, err = s.rob(ctx, chat, elusive, victim, cd, nil, crimeRisk{}, flagrant{odds: 0, jail: baseJail}, after, robMinVictim, fail)
	if err != nil || r.jail != 0 || column(t, elusive, "preso_ate") != 0 {
		t.Errorf("ladrão liso que escapa do flagrante não vai preso, deu %+v (%v)", r, err)
	}
}

func TestCleanRecord(t *testing.T) {
	now := time.Now()
	cases := []struct {
		since time.Duration
		clean float64
	}{
		{0, 0},
		{cleanEvery - time.Minute, 0},
		{cleanEvery, cleanStep},
		{3*cleanEvery + time.Hour, 3 * cleanStep},
		{30 * cleanEvery, cleanMax},
	}
	for _, c := range cases {
		p := Profession{guard: 0.05}.cleared(now.Add(-c.since).Unix(), now)
		if percent(p.clean) != percent(c.clean) || percent(p.guard) != percent(0.05+c.clean) {
			t.Errorf("sem crime há %s a ficha dá %d%% e soma no guard, deu %d%% (guard %d%%)", c.since, percent(c.clean), percent(p.clean), percent(p.guard))
		}
	}
	if p := (Profession{}).cleared(0, now); p.clean != cleanMax {
		t.Errorf("quem nunca cometeu crime tem a ficha cheia (%d%%), tem %d%%", percent(cleanMax), percent(p.clean))
	}
}
