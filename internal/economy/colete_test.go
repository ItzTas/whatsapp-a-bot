package economy

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

func TestVest(t *testing.T) {
	ctx := context.Background()
	store, err := core.OpenMessageStore(ctx, filepath.Join(t.TempDir(), "mensagens.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	s := &walletStore{db: &ledgerDB{db: store.DB}}
	chat := types.NewJID("120363000000000001", types.GroupServer)
	victim := core.Account{Key: "5511900000001@s.whatsapp.net", JID: types.NewJID("5511900000001", types.DefaultUserServer)}
	held := func(t *testing.T) int64 {
		t.Helper()
		var have int64
		err := s.inTx(ctx, func(tx *ledgerTx) error {
			grades, err := gradesIn(ctx, tx, backpackTable, chat, victim, vestItemID)
			for _, g := range grades {
				have += g.quantity
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		return have
	}
	within := func(t *testing.T, run func(tx *ledgerTx) error) error {
		t.Helper()
		return s.inTx(ctx, func(tx *ledgerTx) error {
			if err := openWallet(ctx, tx, chat, victim); err != nil {
				return err
			}
			return run(tx)
		})
	}

	err = within(t, func(tx *ledgerTx) error {
		saved, torn, err := wearVest(ctx, tx, chat, victim)
		if saved || torn {
			t.Errorf("sem colete na mochila não tem o que segurar, deu saved=%v torn=%v", saved, torn)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	err = within(t, func(tx *ledgerTx) error { return addItems(ctx, tx, chat, victim, vestItemID, 10) })
	if err != nil {
		t.Fatal(err)
	}
	if have := held(t); have != vestCarry {
		t.Fatalf("a mochila carrega no máximo %d coletes, ficou com %d", vestCarry, have)
	}
	err = within(t, func(tx *ledgerTx) error { return roomFor(ctx, tx, chat, victim, vestItemID, 1) })
	if !errors.Is(err, errBagFull) {
		t.Fatalf("com %d coletes não cabe mais nenhum, deu %v", vestCarry, err)
	}

	var saves, tears int
	for range 400 {
		before := held(t)
		if before == 0 {
			if err := within(t, func(tx *ledgerTx) error { return addItems(ctx, tx, chat, victim, vestItemID, 1) }); err != nil {
				t.Fatal(err)
			}
			before = 1
		}
		var saved, torn bool
		err := within(t, func(tx *ledgerTx) (err error) {
			saved, torn, err = wearVest(ctx, tx, chat, victim)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		if saved == torn {
			t.Fatalf("com colete ele segura ou falha, deu saved=%v torn=%v", saved, torn)
		}
		after := held(t)
		if saved && after != before-1 {
			t.Fatalf("o colete que segura a bala quebra: tinha %d, ficou %d", before, after)
		}
		if torn && after != before {
			t.Fatalf("o colete que falha continua na mochila: tinha %d, ficou %d", before, after)
		}
		if saved {
			saves++
			continue
		}
		tears++
	}
	if rate := float64(saves) / 400; rate < vestChance-0.1 || rate > vestChance+0.1 {
		t.Fatalf("o colete segura perto de %d%% das vezes, segurou %d de 400 (%d falharam)", percent(vestChance), saves, tears)
	}
}
