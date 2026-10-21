package economy

import (
	"math"
	"testing"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
)

func (fb firmBench) tyrant(t *testing.T, acc core.Account) {
	t.Helper()
	err := fb.s.inTx(fb.ctx, func(tx *ledgerTx) error {
		return swearIn(fb.ctx, tx, fb.chat, acc, 3, false, fb.now)
	})
	if err != nil {
		t.Fatal(err)
	}
	fb.exec(t, "UPDATE presidencias SET "+tyrannyDecree.column+" = ? WHERE chat = ?", fb.now.Unix(), fb.chat.String())
}

func (fb firmBench) protege(t *testing.T, acc core.Account) bool {
	t.Helper()
	if err := fb.s.inTx(fb.ctx, func(tx *ledgerTx) error { return openWallet(fb.ctx, tx, fb.chat, acc) }); err != nil {
		t.Fatal(err)
	}
	st, err := fb.s.standing(fb.ctx, fb.chat, acc)
	if err != nil {
		t.Fatal(err)
	}
	return st.protege
}

func (fb firmBench) anoint(t *testing.T, president, acc core.Account) protegeSwap {
	t.Helper()
	swap, err := fb.s.anointProtege(fb.ctx, fb.chat, president, acc, fb.now)
	if err != nil {
		t.Fatal(err)
	}
	return swap
}

func TestProtegeIsOneAtATime(t *testing.T) {
	fb := newFirmBench(t)
	dictator, ana, bia := fb.person("1"), fb.person("2"), fb.person("3")
	fb.tyrant(t, dictator)

	if swap := fb.anoint(t, dictator, ana); swap != protegeNew {
		t.Fatalf("primeiro favorito = %v, want protegeNew", swap)
	}
	if !fb.protege(t, ana) || fb.protege(t, dictator) {
		t.Fatal("só a ana devia ser favorita")
	}
	if swap := fb.anoint(t, dictator, bia); swap != protegeReplaced {
		t.Fatalf("trocar de favorito = %v, want protegeReplaced", swap)
	}
	if fb.protege(t, ana) || !fb.protege(t, bia) {
		t.Fatal("a bia devia ser a única favorita depois da troca")
	}
	if swap := fb.anoint(t, dictator, bia); swap != protegeSame {
		t.Fatalf("favorecer a mesma pessoa = %v, want protegeSame", swap)
	}
}

func TestProtegeLastsOnlyThroughItsTyranny(t *testing.T) {
	fb := newFirmBench(t)
	dictator, ana := fb.person("1"), fb.person("2")
	fb.tyrant(t, dictator)
	fb.anoint(t, dictator, ana)

	fb.exec(t, "UPDATE presidencias SET "+tyrannyDecree.column+" = ? WHERE chat = ?", fb.now.Unix()+1, fb.chat.String())
	if fb.protege(t, ana) {
		t.Fatal("favorito de uma ditadura anterior continua valendo na nova")
	}

	fb.anoint(t, dictator, ana)
	fb.exec(t, "UPDATE presidencias SET mandato_ate = ? WHERE chat = ?", fb.now.Unix()-1, fb.chat.String())
	if fb.protege(t, ana) {
		t.Fatal("favorito continua valendo depois do fim do mandato")
	}
	if swap := fb.anoint(t, dictator, ana); swap != protegeOver {
		t.Fatalf("favorecer sem mandato = %v, want protegeOver", swap)
	}
}

func TestPatronizedPerks(t *testing.T) {
	base := Profession{min: 100, max: 200, dailyBonus: 5, cordon: 0.1}
	if got := base.patronized(false); got.protege || got.min != 100 || got.max != 200 {
		t.Fatalf("quem não é favorito mudou: %+v", got)
	}
	got := base.patronized(true)
	if !got.protege || got.min != 115 || got.max != 230 || got.dailyBonus != 5+protegeDailyBonus {
		t.Fatalf("favorito: min %d, max %d, daily %d", got.min, got.max, got.dailyBonus)
	}
	if math.Abs(got.cordon-(0.1+protegeGuard)) > 1e-9 {
		t.Fatalf("segurança do favorito = %v, want %v", got.cordon, 0.1+protegeGuard)
	}
}
