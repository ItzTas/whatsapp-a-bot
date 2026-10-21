package economy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

func TestLoadedCoinFavor(t *testing.T) {
	it, ok := itemByID(loadedCoinID)
	if !ok || !it.illegal {
		t.Fatal("a moeda viciada não tá no catálogo como item ilegal")
	}
	coin := it.flex.apply(Profession{})
	if coin.cheat != loadedCoinFavor || coin.casinoLuck != loadedCoinLuck {
		t.Fatalf("moeda viciada deu cheat %d e sorte %d", coin.cheat, coin.casinoLuck)
	}
	boost := PrizeBoost{cheat: coin.cheat, luck: coin.casinoLuck}
	if got := boost.Favor(stakeRiskFloor); got != float64(loadedCoinFavor)/100 {
		t.Fatalf("favor da moeda = %v", got)
	}
	if got := boost.Risk(stakeRiskFloor); got != 0 {
		t.Fatalf("risco com a moeda = %v", got)
	}
	if got := boost.Pay(100, 200); got != 100+100*(100+loadedCoinLuck)/100 {
		t.Fatalf("prêmio com a moeda = %d", got)
	}
	if got := (Profession{cheat: loadedCoinFavor, elusive: true}).flagrantOdds(loadedCoinCaught); got != 0 {
		t.Fatalf("o Gorro do Saci não escapou da moeda: %v", got)
	}
}

func TestLoadedCoinBust(t *testing.T) {
	ctx := context.Background()
	chat := types.NewJID("120363000000000001", types.GroupServer)
	now := time.Unix(time.Now().Unix(), 0)
	fates := map[snitchFate]bool{}
	for range 40 {
		s, people := jackpotWallets(t, ctx, chat, "5511900000001")
		cheater := people[0]
		if err := s.inTx(ctx, func(tx *ledgerTx) error { return addItems(ctx, tx, chat, cheater, loadedCoinID, 2) }); err != nil {
			t.Fatal(err)
		}

		c, err := s.bustCheat(ctx, chat, cheater, baseJail, now)
		if err != nil {
			t.Fatal(err)
		}
		fates[c.fate] = true
		if c.burned < 50_000*loadedCoinLossMin/100 || c.burned > 50_000*loadedCoinLossMax/100 {
			t.Fatalf("o cassino tomou %d de 50000", c.burned)
		}

		var balance, jailed, dead, record, coins int64
		err = s.db.QueryRowContext(ctx,
			"SELECT saldo, preso_ate, morto_ate, ficha_em, (SELECT COALESCE(SUM(quantidade), 0) FROM inventarios WHERE chat = ? AND pessoa = ? AND item = ?) FROM carteiras WHERE chat = ? AND pessoa = ?",
			chat.String(), cheater.Key, loadedCoinID, chat.String(), cheater.Key,
		).Scan(&balance, &jailed, &dead, &record, &coins)
		if err != nil {
			t.Fatal(err)
		}
		if coins != 1 {
			t.Fatalf("sobraram %d moedas de 2", coins)
		}
		if record != now.Unix() {
			t.Fatalf("ficha_em = %d, queria %d", record, now.Unix())
		}
		if balance != 50_000-c.burned-c.fine-c.lost {
			t.Fatalf("saldo %d depois de perder %d + %d + %d", balance, c.burned, c.fine, c.lost)
		}
		punished := map[snitchFate]bool{snitchJailed: jailed == now.Add(baseJail).Unix(), snitchKilled: dead > now.Unix()}
		if !punished[c.fate] {
			t.Fatalf("destino %d sem castigo: preso_ate %d, morto_ate %d", c.fate, jailed, dead)
		}
	}
	if !fates[snitchJailed] || !fates[snitchKilled] {
		t.Fatalf("em 40 flagras os destinos foram só %v", fates)
	}
}

func TestLoadedCoinDailyCap(t *testing.T) {
	ctx := context.Background()
	chat := types.NewJID("120363000000000001", types.GroupServer)
	s, people := jackpotWallets(t, ctx, chat, "5511900000001", "5511900000002")
	coin, _ := itemByID(loadedCoinID)
	today := time.Date(2026, 10, 20, 12, 0, 0, 0, core.ReminderZone)
	claim := func(acc core.Account, quantity int64, at time.Time) error {
		return s.inTx(ctx, func(tx *ledgerTx) error { return claimDaily(ctx, tx, chat, acc, coin, quantity, at) })
	}

	for _, step := range []struct {
		acc      core.Account
		quantity int64
		at       time.Time
		want     error
	}{
		{people[0], loadedCoinDaily, today, nil},
		{people[0], 1, today.Add(time.Hour), errDailyCap},
		{people[1], loadedCoinDaily, today, nil},
		{people[0], loadedCoinDaily, today.Add(24 * time.Hour), nil},
		{people[0], loadedCoinDaily + 1, today.Add(48 * time.Hour), errDailyCap},
	} {
		if err := claim(step.acc, step.quantity, step.at); !errors.Is(err, step.want) {
			t.Fatalf("%s comprando %d em %s: %v, queria %v", step.acc.Key, step.quantity, step.at, err, step.want)
		}
	}

	var days int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM compras_diarias WHERE pessoa = ?", people[0].Key).Scan(&days); err != nil {
		t.Fatal(err)
	}
	if days != 1 {
		t.Fatalf("ficaram %d dias de compra guardados", days)
	}
}
