package economy

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

func TestJackpotRobbery(t *testing.T) {
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
	gambler, thief, other, rookie := person("5511900000001"), person("5511900000002"), person("5511900000003"), person("5511900000004")
	for _, acc := range []core.Account{gambler, thief, other, rookie} {
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
	r, err := s.rob(ctx, chat, thief, gambler, cd, nil, crimeRisk{}, flagrant{}, now.Add(2*time.Minute), robMinVictim, peek)
	if err != nil || seen != 5_000 || r.jackpot != 5_000 {
		t.Fatalf("os lucros grandes seguidos somam 5.000 e chegam no sorteio do roubo, chegou %d (%+v, %v)", seen, r, err)
	}

	seen = -1
	later := now.Add(robShield + robEvery + 3*time.Minute)
	r, err = s.rob(ctx, chat, thief, gambler, cd, nil, crimeRisk{}, flagrant{}, later, robMinVictim, peek)
	if err != nil || seen != 0 || r.jackpot != 0 {
		t.Fatalf("depois de roubado(a), o prêmio da aposta já foi embora, chegou %d (%+v, %v)", seen, r, err)
	}

	if err := s.noteJackpot(ctx, chat, other, 4_000, now); err != nil {
		t.Fatal(err)
	}
	seen = -1
	r, err = s.rob(ctx, chat, rookie, other, cd, nil, crimeRisk{}, flagrant{}, now.Add(jackpotWindow+time.Second), robMinVictim, peek)
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
