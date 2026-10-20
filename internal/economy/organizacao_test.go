package economy

import (
	"errors"
	"testing"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
)

func TestSignSyndicate(t *testing.T) {
	fb := newFirmBench(t)
	vigia, mafia := mustFirm(t, "vigia-de-rua"), mustFirm(t, "mafia-do-porto")
	acc := fb.person("5511900000001")
	fb.setBalance(t, acc, 100_000)
	fb.markJackpot(t, acc, 20_000)

	h, err := fb.s.sign(fb.ctx, fb.chat, acc, mafia, fb.now)
	if err != nil || h.fee != 100_000*mafia.rate/100 || h.urgency != 0 {
		t.Fatalf("a organização cobra só a diária, sem taxa de emergência, deu %+v (%v)", h, err)
	}
	if _, err := fb.s.sign(fb.ctx, fb.chat, acc, vigia, fb.now); err != nil {
		t.Fatal(err)
	}
	contracts, err := fb.s.contracts(fb.ctx, fb.chat, acc, fb.now)
	if err != nil || len(contracts) != 2 || contracts[0].firm.id != vigia.id || contracts[1].firm.id != mafia.id {
		t.Fatalf("dá pra ter um contrato de segurança e um com o crime, a segurança primeiro, deu %+v (%v)", contracts, err)
	}
	backer, err := syndicateOf(fb.ctx, fb.s.db, fb.chat, acc, fb.now)
	if err != nil || backer.id != mafia.id {
		t.Fatalf("o syndicateOf acha a organização contratada, deu %+v (%v)", backer, err)
	}
	if backer, err := syndicateOf(fb.ctx, fb.s.db, fb.chat, fb.person("5511900000002"), fb.now); err != nil || backer.id != "" || backer.edge != 0 {
		t.Fatalf("sem contrato não tem organização nem bônus, deu %+v (%v)", backer, err)
	}
}

func TestCrewSyndicate(t *testing.T) {
	gangue, firma, mafia := mustFirm(t, "gangue-da-esquina"), mustFirm(t, "firma-do-157"), mustFirm(t, "mafia-do-porto")
	if got := crewSyndicate([]firm{{}, firma, mafia, gangue}); got.id != mafia.id {
		t.Fatalf("a equipe fica com a melhor organização que cobre crime em grupo, deu %s", got.id)
	}
	if got := crewSyndicate([]firm{gangue, {}}); got.id != "" || got.edge != 0 {
		t.Fatalf("a Gangue da esquina não cobre crime em grupo, deu %s", got.id)
	}
}

func TestLockUpNotesJailing(t *testing.T) {
	fb := newFirmBench(t)
	acc := fb.person("5511900000001")
	fb.setBalance(t, acc, 1_000)

	failed := errors.New("desfaz")
	err := fb.s.inTx(fb.ctx, func(tx *ledgerTx) error {
		if _, err := lockUp(fb.ctx, tx, fb.chat, acc, fb.now); err != nil {
			return err
		}
		return failed
	})
	if !errors.Is(err, failed) || len(fb.s.jailings.take(fb.chat)) != 0 {
		t.Fatalf("prisão de transação desfeita não fica anotada (%v)", err)
	}

	err = fb.s.inTx(fb.ctx, func(tx *ledgerTx) error {
		_, err := lockUp(fb.ctx, tx, fb.chat, acc, fb.now)
		return err
	})
	jailed := fb.s.jailings.take(fb.chat)
	if err != nil || len(jailed) != 1 || jailed[acc.Key] != acc {
		t.Fatalf("prisão que entrou no banco fica anotada pra oferta, deu %+v (%v)", jailed, err)
	}
	if again := fb.s.jailings.take(fb.chat); len(again) != 0 {
		t.Fatalf("cada prisão é entregue uma vez só, sobrou %+v", again)
	}
}

func TestBreakout(t *testing.T) {
	fb := newFirmBench(t)
	firma := mustFirm(t, "firma-do-157")
	jail := func(acc core.Account) {
		fb.exec(t, "UPDATE carteiras SET preso_ate = ?, crime_em = 0 WHERE chat = ? AND pessoa = ?", fb.now.Add(10*time.Minute).Unix(), fb.chat.String(), acc.Key)
	}
	jailState := func(acc core.Account) (until, crime int64) {
		t.Helper()
		if err := fb.s.db.QueryRowContext(fb.ctx, "SELECT preso_ate, crime_em FROM carteiras WHERE chat = ? AND pessoa = ?", fb.chat.String(), acc.Key).Scan(&until, &crime); err != nil {
			t.Fatal(err)
		}
		return until, crime
	}
	offer := func(acc core.Account, odds float64) *breakoutOffer {
		f := firma
		f.breakoutOdds = odds
		return &breakoutOffer{chat: fb.chat, acc: acc, firm: f, expires: fb.now.Add(offerWindow)}
	}

	lucky := fb.person("5511900000001")
	fb.setBalance(t, lucky, 10_000)
	jail(lucky)
	freed, err := fb.s.breakout(fb.ctx, offer(lucky, 1), fb.now)
	until, crime := jailState(lucky)
	if err != nil || !freed || until != 0 || crime != fb.now.Unix() || fb.balance(t, lucky) != 10_000-firma.breakout {
		t.Fatalf("resgate que dá certo cobra o preço fixo, solta e marca o crime, deu freed=%v preso_ate=%d crime_em=%d saldo=%d (%v)", freed, until, crime, fb.balance(t, lucky), err)
	}

	unlucky := fb.person("5511900000002")
	fb.setBalance(t, unlucky, 10_000)
	jail(unlucky)
	freed, err = fb.s.breakout(fb.ctx, offer(unlucky, 0), fb.now)
	if until, _ := jailState(unlucky); err != nil || freed || until != fb.now.Add(10*time.Minute).Unix() || fb.balance(t, unlucky) != 10_000-firma.breakout {
		t.Fatalf("resgate que dá errado cobra e deixa na cadeia, deu freed=%v preso_ate=%d (%v)", freed, until, err)
	}

	loose := fb.person("5511900000003")
	fb.setBalance(t, loose, 10_000)
	if _, err := fb.s.breakout(fb.ctx, offer(loose, 1), fb.now); !errors.Is(err, errNotHeld) || fb.balance(t, loose) != 10_000 {
		t.Fatalf("quem já tá solto não paga resgate, deu %v", err)
	}

	broke := fb.person("5511900000004")
	fb.setBalance(t, broke, firma.breakout-1)
	jail(broke)
	if _, err := fb.s.breakout(fb.ctx, offer(broke, 1), fb.now); !errors.Is(err, ErrNoFunds) {
		t.Fatalf("sem o preço no saldo não tem resgate, deu %v", err)
	}
	if until, _ := jailState(broke); until == 0 {
		t.Fatal("sem pagar, continua preso")
	}
}
