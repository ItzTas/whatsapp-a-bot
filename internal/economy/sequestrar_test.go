package economy

import (
	"testing"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
)

func TestSoldTyrantLosesOffice(t *testing.T) {
	fb := newFirmBench(t)
	seat := func(acc core.Account, tyranny bool) {
		t.Helper()
		err := fb.s.inTx(fb.ctx, func(tx *ledgerTx) error {
			return swearIn(fb.ctx, tx, fb.chat, acc, 3, false, fb.now)
		})
		if err != nil {
			t.Fatal(err)
		}
		if tyranny {
			fb.exec(t, "UPDATE presidencias SET "+tyrannyDecree.column+" = ? WHERE chat = ?", fb.now.Unix(), fb.chat.String())
		}
	}
	term := func() (until int64, vacant bool) {
		t.Helper()
		err := fb.s.db.QueryRowContext(fb.ctx,
			"SELECT mandato_ate, EXISTS (SELECT 1 FROM presidencia_vagas WHERE chat = ?) FROM presidencias WHERE chat = ?",
			fb.chat.String(), fb.chat.String(),
		).Scan(&until, &vacant)
		if err != nil {
			t.Fatal(err)
		}
		return until, vacant
	}
	inOffice := func(why string) {
		t.Helper()
		if until, vacant := term(); until <= fb.now.Unix() || vacant {
			t.Fatalf("%s, mandato_ate=%d vaga=%v", why, until, vacant)
		}
	}
	kidnap := func(kidnapper, victim core.Account) {
		t.Helper()
		k, err := fb.s.kidnap(fb.ctx, fb.chat, kidnapper, victim, kidnapCooldown, nil, kidnapOdds{success: 1}, fb.now)
		if err != nil || !k.success {
			t.Fatalf("sequestro garantido deu %+v (%v)", k, err)
		}
	}
	sell := func(seller, sold core.Account) hostageSold {
		t.Helper()
		h, err := fb.s.sellHostage(fb.ctx, fb.chat, seller, sold, 100, fb.now)
		if err != nil || h.left <= 0 {
			t.Fatalf("venda do refém deu %+v (%v)", h, err)
		}
		return h
	}

	tyrant, kidnapper := fb.person("5511900000001"), fb.person("5511900000002")
	seat(tyrant, true)
	kidnap(kidnapper, tyrant)
	inOffice("o ditador sequestrado continua no cargo enquanto tá no cativeiro")
	if h := sell(kidnapper, tyrant); !h.deposed {
		t.Fatalf("o ditador vendido perde o cargo, deu %+v", h)
	}
	if until, vacant := term(); until != fb.now.Unix() || !vacant {
		t.Fatalf("o mandato do ditador vendido acaba agora e a vaga abre, deu mandato_ate=%d vaga=%v", until, vacant)
	}

	president, other := fb.person("5511900000003"), fb.person("5511900000004")
	seat(president, false)
	kidnap(other, president)
	if h := sell(other, president); h.deposed {
		t.Fatalf("presidente sem ditadura continua no cargo mesmo vendido, deu %+v", h)
	}
	inOffice("o mandato do presidente sem ditadura segue depois da venda")

	snitch, captor := fb.person("5511900000005"), fb.person("5511900000006")
	seat(snitch, true)
	var p payback
	err := fb.s.inTx(fb.ctx, func(tx *ledgerTx) (err error) {
		p, err = captureSnitch(fb.ctx, tx, fb.chat, snitch, captor, baseJail, fb.now)
		return err
	})
	if err != nil || p.fate != snitchCaptured {
		t.Fatalf("o ditador X-9 vai pro cativeiro, deu %+v (%v)", p, err)
	}
	inOffice("o ditador X-9 levado de refém continua no cargo")
	if h := sell(captor, snitch); !h.deposed {
		t.Fatalf("o ditador X-9 vendido por quem ficou com ele perde o cargo, deu %+v", h)
	}
}
