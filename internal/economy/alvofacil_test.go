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
	peek := func(_, _, jackpot int64) (bool, int64) { seen = jackpot; return true, 100 }

	if err := s.noteJackpot(ctx, chat, gambler, jackpotMin-1, now); err != nil {
		t.Fatal(err)
	}
	if err := s.noteJackpot(ctx, chat, gambler, 3_000, now); err != nil {
		t.Fatal(err)
	}
	if err := s.noteJackpot(ctx, chat, gambler, 2_000, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	robbedAt := now.Add(2 * time.Minute)
	r, err := s.rob(ctx, chat, thief, gambler, cd, nil, crimeRisk{}, flagrant{}, robbedAt, robMinVictim, peek)
	if err != nil || seen != 5_000 || r.jackpot != 5_000 {
		t.Fatalf("os lucros grandes seguidos somam 5.000 e chegam no sorteio do roubo, chegou %d (%+v, %v)", seen, r, err)
	}

	shield := jackpotShield(r.jackpot)
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
	if err != nil || r.shield != 0 || seen != 5_000 || r.jackpot != 5_000 {
		t.Fatalf("passado o escudo curto, a marca de 5.000 continua valendo, chegou %d (%+v, %v)", seen, r, err)
	}

	if err := s.noteJackpot(ctx, chat, other, 4_000, now); err != nil {
		t.Fatal(err)
	}
	seen = -1
	r, err = s.rob(ctx, chat, late, other, cd, nil, crimeRisk{}, flagrant{}, now.Add(jackpotWindow+time.Second), robMinVictim, peek)
	if err != nil || seen != 0 {
		t.Fatalf("passada a janela de %s, o prêmio não conta mais, chegou %d (%+v, %v)", jackpotWindow, seen, r, err)
	}
}

func TestLootedTakesJackpotWhole(t *testing.T) {
	if got, want := looted(100_000, 0, 20), exposed(100_000, 20); got != want {
		t.Errorf("sem prêmio, o roubo segue as faixas (%d), deu %d", want, got)
	}
	if got, want := looted(100_000, 40_000, 20), exposed(60_000, 20)+8_000; got != want {
		t.Errorf("os 40.000 do prêmio saem com os 20%% cheios e o resto pelas faixas (%d), deu %d", want, got)
	}
	if jackpotOdds(0.4, 0) != 0.4 || jackpotOdds(0.4, 1) != clampChance(0.4+jackpotRobBonus) {
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
	miss := func(_, _, _ int64) (bool, int64) { return false, 0 }

	if _, err := s.rob(ctx, chat, thief, first, cd, nil, crimeRisk{}, flagrant{}, now, robMinVictim, miss); err != nil {
		t.Fatal(err)
	}
	if err := s.noteJackpot(ctx, chat, gambler, 5_000, now); err != nil {
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
	if err != nil || r.wait != 0 || r.jackpot != 5_000 {
		t.Fatalf("contra quem acabou de ganhar na aposta, dá pra roubar depois de %s (%+v, %v)", short, r, err)
	}
}
