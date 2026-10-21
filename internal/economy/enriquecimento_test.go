package economy

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

func TestEnrichments(t *testing.T) {
	ctx := context.Background()
	store, err := core.OpenMessageStore(ctx, filepath.Join(t.TempDir(), "mensagens.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	s := &walletStore{db: &ledgerDB{db: store.DB}}
	chat := types.NewJID("120363000000000001", types.GroupServer)
	frozen := types.NewJID("120363000000000002", types.GroupServer)
	person := func(n string) core.Account {
		return core.Account{Key: n + "@s.whatsapp.net", JID: types.NewJID(n, types.DefaultUserServer)}
	}
	worker, loser, idle, late := person("5511900000001"), person("5511900000002"), person("5511900000003"), person("5511900000004")
	exec := func(t *testing.T, query string, args ...any) {
		t.Helper()
		if _, err := store.DB.ExecContext(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	setWorth := func(t *testing.T, in types.JID, acc core.Account, saldo, cofre int64) {
		t.Helper()
		err := s.inTx(ctx, func(tx *ledgerTx) error {
			if err := openWallet(ctx, tx, in, acc); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = ?, cofre_casa = ? WHERE chat = ? AND pessoa = ?", saldo, cofre, in.String(), acc.Key)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	start := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	setWorth(t, chat, worker, 1_000, 0)
	setWorth(t, chat, loser, 5_000, 0)
	setWorth(t, chat, idle, 0, 0)
	setWorth(t, frozen, worker, 9_000, 0)
	exec(t, "INSERT INTO economia_vinculos (chat, economia) VALUES (?, ?)", frozen.String(), chat.String())
	if err := s.logWealth(ctx, start.Add(-10*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.logWealth(ctx, start); err != nil {
		t.Fatal(err)
	}

	var old, frozenRows int64
	if err := store.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM patrimonios WHERE em < ?", start.Unix()).Scan(&old); err != nil || old != 0 {
		t.Fatalf("o retrato de mais de 7 dias é apagado, sobraram %d (%v)", old, err)
	}
	if err := store.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM patrimonios WHERE chat = ?", frozen.String()).Scan(&frozenRows); err != nil || frozenRows != 0 {
		t.Fatalf("grupo vinculado não ganha retrato, ganhou %d (%v)", frozenRows, err)
	}

	setWorth(t, chat, worker, 2_000, 500)
	exec(t, "INSERT INTO poupancas (chat, pessoa, valor, juros, vence_em) VALUES (?, ?, ?, 0, 0)", chat.String(), worker.Key, 500)
	setWorth(t, chat, loser, 3_000, 0)
	if err := s.logWealth(ctx, start.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	setWorth(t, chat, late, 800, 0)
	if err := s.logWealth(ctx, start.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}

	list, err := s.enrichments(ctx, chat, start.Add(2*time.Hour), wealthWindow)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("entram só quem tem retrato de pelo menos 1h e patrimônio, deu %+v", list)
	}
	if list[0].who != worker.JID || list[0].worth != 3_000 || list[0].start != 1_000 || list[0].perHour != 1_000 {
		t.Fatalf("quem mais enriquece vem primeiro, contando saldo, cofre e poupança: %+v", list[0])
	}
	if list[1].who != loser.JID || list[1].perHour != -1_000 {
		t.Fatalf("quem perdeu 2.000 em 2h perde 1.000 por hora: %+v", list[1])
	}

	recent, err := s.enrichments(ctx, chat, start.Add(2*time.Hour), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if len(recent) != 2 || recent[0].start != 3_000 || recent[0].perHour != 0 || recent[1].perHour != 0 {
		t.Fatalf("a janela de 1h compara com o retrato de 1h atrás: %+v", recent)
	}
}

func TestEnrichmentWindow(t *testing.T) {
	cases := map[string]struct {
		args   []string
		window time.Duration
		ok     bool
	}{
		"sem janela":    {nil, wealthWindow, true},
		"horas":         {[]string{"12h"}, 12 * time.Hour, true},
		"dias":          {[]string{"3D"}, 72 * time.Hour, true},
		"acima do teto": {[]string{"30d"}, wealthKeep, true},
		"sem unidade":   {[]string{"12"}, 0, false},
		"zero":          {[]string{"0h"}, 0, false},
		"texto":         {[]string{"ontem"}, 0, false},
	}
	for name, c := range cases {
		window, ok := enrichmentWindow(c.args)
		if window != c.window || ok != c.ok {
			t.Errorf("%s: esperava %s %v, deu %s %v", name, c.window, c.ok, window, ok)
		}
	}
}
