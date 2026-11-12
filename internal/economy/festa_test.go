package economy

import (
	"errors"
	"testing"
	"time"
)

func TestPartyPrice(t *testing.T) {
	if got := partyPrice(1_000); got != partyFloor {
		t.Fatalf("saldo pequeno paga o piso, deu %d", got)
	}
	if got := partyPrice(100_000_000); got != 100_000_000*partyPercent/100 {
		t.Fatalf("saldo grande paga a porcentagem, deu %d", got)
	}
}

func TestThrowParty(t *testing.T) {
	fb := newFirmBench(t)
	host, guest := fb.person("5511900000001"), fb.person("5511900000002")
	fb.setBalance(t, host, 10_000_000)
	fb.setBalance(t, guest, 10_000)

	first, err := fb.s.throwParty(fb.ctx, fb.chat, host, fb.now)
	price := int64(10_000_000 * partyPercent / 100)
	if err != nil || first.extended || first.full || first.price != price || !first.party.until.Equal(fb.now.Add(partySpan)) || first.party.total != price {
		t.Fatalf("a primeira festa cobra a porcentagem e dura %s, deu %+v (%v)", partySpan, first, err)
	}
	if got := fb.balance(t, host); got != 10_000_000-price {
		t.Fatalf("o dinheiro da festa some do saldo, ficou %d", got)
	}

	if _, err := fb.s.throwParty(fb.ctx, fb.chat, guest, fb.now); !errors.Is(err, ErrNoFunds) {
		t.Fatalf("quem não tem o piso não banca festa, deu %v", err)
	}
	if got := fb.balance(t, guest); got != 10_000 {
		t.Fatalf("a festa recusada não cobra nada, ficou %d", got)
	}

	second, err := fb.s.throwParty(fb.ctx, fb.chat, host, fb.now)
	if err != nil || !second.extended || !second.party.until.Equal(fb.now.Add(2*partySpan)) || second.party.total != first.price+second.price {
		t.Fatalf("pagar com a festa rolando estica ela e soma o gasto, deu %+v (%v)", second, err)
	}
	stored, err := fb.s.partyOf(fb.ctx, fb.chat)
	if err != nil || !stored.until.Equal(second.party.until) || stored.host.User != host.JID.User || stored.total != second.party.total {
		t.Fatalf("a festa fica guardada com quem pagou por último, deu %+v (%v)", stored, err)
	}

	fb.exec(t, "UPDATE festas SET ate = ? WHERE chat = ?", fb.now.Add(partyMaxSpan).Unix(), fb.chat.String())
	before := fb.balance(t, host)
	full, err := fb.s.throwParty(fb.ctx, fb.chat, host, fb.now)
	if err != nil || !full.full || fb.balance(t, host) != before {
		t.Fatalf("festa no limite de %s não estica nem cobra, deu %+v (%v)", partyMaxSpan, full, err)
	}

	later := fb.now.Add(partyMaxSpan + time.Minute)
	fresh, err := fb.s.throwParty(fb.ctx, fb.chat, host, later)
	if err != nil || fresh.extended || !fresh.party.until.Equal(later.Add(partySpan)) || fresh.party.total != fresh.price {
		t.Fatalf("depois que a festa acaba, pagar começa outra do zero, deu %+v (%v)", fresh, err)
	}
}

func TestPartyBoostsEveryoneInTheChat(t *testing.T) {
	fb := newFirmBench(t)
	host, guest := fb.person("5511900000001"), fb.person("5511900000002")
	fb.setBalance(t, host, 1_000_000)
	fb.setBalance(t, guest, 0)
	if _, err := fb.s.throwParty(fb.ctx, fb.chat, host, fb.now); err != nil {
		t.Fatal(err)
	}
	st, err := fb.s.standing(fb.ctx, fb.chat, guest)
	if err != nil || st.party != fb.now.Add(partySpan).Unix() {
		t.Fatalf("o standing de qualquer um do grupo traz o fim da festa, deu %d (%v)", st.party, err)
	}

	base := Profession{min: 1_000, max: 2_000}
	on := base.partying(st.party, fb.now)
	if !on.party || on.min != 1_000*(100+partyPayBonus)/100 || on.max != 2_000*(100+partyPayBonus)/100 {
		t.Fatalf("com a festa rolando o !trabalhar paga %d%% a mais, deu %+v", partyPayBonus, on)
	}
	off := base.partying(st.party, fb.now.Add(partySpan))
	if off.party || off.min != base.min || off.max != base.max {
		t.Fatalf("festa que acabou não dá bônus, deu %+v", off)
	}
}
