package economy

import (
	"database/sql"
	"errors"
	"testing"
	"time"
)

func mustHitKind(t *testing.T, id string) hitKind {
	t.Helper()
	k, ok := hitKindIn(id)
	if !ok || k.id != id {
		t.Fatalf("encomenda %s não existe", id)
	}
	return k
}

func TestHitKindIn(t *testing.T) {
	cases := map[string]string{
		"@5511999999999 roubo":      "roubo",
		"fulano sequestrar":         "sequestro",
		"Cativeiro pro @5511999999": "sequestro",
	}
	for text, want := range cases {
		if k, ok := hitKindIn(text); !ok || k.id != want {
			t.Fatalf("%q deveria ser %s, deu %+v", text, want, k)
		}
	}
	if _, ok := hitKindIn("@5511999999999 abraço"); ok {
		t.Fatal("sem o nome do serviço não acha encomenda")
	}
}

func TestHitPrice(t *testing.T) {
	roubo := mustHitKind(t, "roubo")
	cases := []struct {
		client, target, want int64
	}{
		{0, 0, roubo.floor},
		{100_000_000, 0, 100_000_000 * roubo.clientPercent / 100},
		{0, 50_000_000, 50_000_000 * roubo.targetPercent / 100},
	}
	for _, c := range cases {
		if got := roubo.price(c.client, c.target); got != c.want {
			t.Fatalf("preço com saldo %d e alvo %d deveria ser %d, deu %d", c.client, c.target, c.want, got)
		}
	}
}

func TestContractHitRobbery(t *testing.T) {
	fb := newFirmBench(t)
	client, victim := fb.person("5511900000001"), fb.person("5511900000002")
	fb.setBalance(t, client, 10_000_000)
	fb.setBalance(t, victim, 1_000_000)
	roubo := mustHitKind(t, "roubo")
	price := roubo.price(10_000_000, 1_000_000)

	h, err := fb.s.contractHit(fb.ctx, fb.chat, hitJob{kind: roubo, odds: 1, client: client, victim: victim}, fb.now)
	if err != nil || !h.success || h.price != price || h.loot <= 0 {
		t.Fatalf("encomenda certeira cobra e rouba, deu %+v (%v)", h, err)
	}
	if got := fb.balance(t, client); got != 10_000_000-price {
		t.Fatalf("o preço sai do saldo de quem encomendou e não volta, ficou %d", got)
	}
	if got := fb.balance(t, victim); got != 1_000_000-h.loot {
		t.Fatalf("o que os capangas levam sai da vítima, ficou %d", got)
	}
	if shield, err := fb.s.shieldLeft(fb.ctx, fb.chat, victim, fb.now); err != nil || shield != robShield {
		t.Fatalf("a vítima ganha o escudo de quem acabou de ser roubado, deu %s (%v)", shield, err)
	}

	again, err := fb.s.contractHit(fb.ctx, fb.chat, hitJob{kind: roubo, odds: 1, client: client, victim: fb.person("5511900000003")}, fb.now.Add(time.Minute))
	if err != nil || again.wait != hitEvery-time.Minute || again.price != 0 {
		t.Fatalf("dá pra encomendar uma vez a cada %s, deu %+v (%v)", hitEvery, again, err)
	}
}

func TestContractHitRefusals(t *testing.T) {
	fb := newFirmBench(t)
	client, victim := fb.person("5511900000001"), fb.person("5511900000002")
	roubo, sequestro := mustHitKind(t, "roubo"), mustHitKind(t, "sequestro")
	fb.setBalance(t, client, 10_000_000)
	fb.setBalance(t, victim, 1_000_000)

	fb.exec(t, "UPDATE carteiras SET escudo_ate = ? WHERE chat = ? AND pessoa = ?", fb.now.Add(time.Minute).Unix(), fb.chat.String(), victim.Key)
	h, err := fb.s.contractHit(fb.ctx, fb.chat, hitJob{kind: roubo, odds: 1, client: client, victim: victim}, fb.now)
	if err != nil || h.shield != time.Minute || fb.balance(t, client) != 10_000_000 {
		t.Fatalf("vítima com escudo recusa o roubo sem cobrar, deu %+v (%v)", h, err)
	}

	fb.exec(t, "UPDATE carteiras SET preso_ate = ? WHERE chat = ? AND pessoa = ?", fb.now.Add(time.Hour).Unix(), fb.chat.String(), victim.Key)
	h, err = fb.s.contractHit(fb.ctx, fb.chat, hitJob{kind: sequestro, odds: 1, client: client, victim: victim}, fb.now)
	if err != nil || !h.held || fb.balance(t, client) != 10_000_000 {
		t.Fatalf("quem tá preso não pode ser sequestrado e a encomenda não cobra, deu %+v (%v)", h, err)
	}

	poor := fb.person("5511900000003")
	fb.setBalance(t, poor, hitMinVictimPay-1)
	h, err = fb.s.contractHit(fb.ctx, fb.chat, hitJob{kind: roubo, odds: 1, client: client, victim: poor}, fb.now)
	if err != nil || !h.poor || fb.balance(t, client) != 10_000_000 {
		t.Fatalf("quem não tem o mínimo não vale o roubo, deu %+v (%v)", h, err)
	}

	broke, rich := fb.person("5511900000004"), fb.person("5511900000005")
	fb.setBalance(t, broke, roubo.floor-1)
	fb.setBalance(t, rich, 1_000_000)
	_, err = fb.s.contractHit(fb.ctx, fb.chat, hitJob{kind: roubo, odds: 1, client: broke, victim: rich}, fb.now)
	var stamp int64
	if scanErr := fb.s.db.QueryRowContext(fb.ctx, "SELECT encomenda_em FROM carteiras WHERE chat = ? AND pessoa = ?", fb.chat.String(), broke.Key).Scan(&stamp); scanErr != nil {
		t.Fatal(scanErr)
	}
	if !errors.Is(err, ErrNoFunds) || fb.balance(t, broke) != roubo.floor-1 || stamp != 0 {
		t.Fatalf("sem saldo pro preço a encomenda não sai nem gasta a espera, deu %v (espera %d)", err, stamp)
	}
}

func TestContractHitKidnapping(t *testing.T) {
	fb := newFirmBench(t)
	client, victim := fb.person("5511900000001"), fb.person("5511900000002")
	sequestro := mustHitKind(t, "sequestro")
	fb.setBalance(t, client, 1_000_000)
	fb.setBalance(t, victim, 500_000)

	h, err := fb.s.contractHit(fb.ctx, fb.chat, hitJob{kind: sequestro, odds: 1, client: client, victim: victim}, fb.now)
	if err != nil || !h.success || h.price != sequestro.price(1_000_000, 500_000) {
		t.Fatalf("sequestro certeiro sai e cobra, deu %+v (%v)", h, err)
	}
	var (
		until     int64
		kidnapper sql.NullString
	)
	err = fb.s.db.QueryRowContext(fb.ctx, "SELECT ate, sequestrador FROM sequestros WHERE chat = ? AND refem = ?", fb.chat.String(), victim.Key).Scan(&until, &kidnapper)
	if err != nil || until != fb.now.Add(hitHostageHold).Unix() || kidnapper.Valid {
		t.Fatalf("a vítima fica no cativeiro da organização, sem sequestrador pra polícia prender, deu %d %v (%v)", until, kidnapper, err)
	}
	if got := fb.balance(t, victim); got != 500_000 {
		t.Fatalf("sequestro não mexe no saldo da vítima, ficou %d", got)
	}
}

func TestContractHitMiss(t *testing.T) {
	fb := newFirmBench(t)
	client, victim := fb.person("5511900000001"), fb.person("5511900000002")
	roubo := mustHitKind(t, "roubo")
	fb.setBalance(t, client, 1_000_000)
	fb.setBalance(t, victim, 1_000_000)

	h, err := fb.s.contractHit(fb.ctx, fb.chat, hitJob{kind: roubo, odds: 0, client: client, victim: victim}, fb.now)
	if err != nil || h.success || h.loot != 0 {
		t.Fatalf("encomenda que falha não rouba nada, deu %+v (%v)", h, err)
	}
	if fb.balance(t, client) != 1_000_000-h.price || fb.balance(t, victim) != 1_000_000 {
		t.Fatal("a encomenda que falha cobra igual e a vítima fica intacta")
	}
}
