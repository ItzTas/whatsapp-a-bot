package economy

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

func TestBacklog(t *testing.T) {
	cases := []struct {
		name  string
		idle  time.Duration
		every time.Duration
		want  int
	}{
		{"antes da espera acabar é um turno só", 5 * time.Minute, 8 * time.Minute, 1},
		{"sem espera é um turno só", time.Hour, 0, 1},
		{"cada espera parada vira um turno", 25 * time.Minute, 8 * time.Minute, 3},
		{"o teto é uma hora de turnos, pra cima", 10 * time.Hour, 8 * time.Minute, 8},
		{"espera longa ainda junta dois turnos", 10 * time.Hour, 40 * time.Minute, 2},
		{"espera curta junta até uma hora", 10 * time.Hour, 2 * time.Minute, 30},
	}
	for _, c := range cases {
		if got := backlog(c.idle, c.every); got != c.want {
			t.Errorf("%s: backlog(%s, %s) = %d, queria %d", c.name, c.idle, c.every, got, c.want)
		}
	}
}

func TestShiftMerge(t *testing.T) {
	paid := shift{amount: 100, task: "trampou", turns: 1, doubles: 1, peaked: true, finds: []item{catalog[0]}}
	bust := shift{amount: -30, task: "deu ruim", busted: true, turns: 1, busts: 1}

	s := bust.merge(paid).merge(bust)
	if s.busted || s.task != "trampou" || !s.peaked {
		t.Errorf("com um turno pago o trampo conta como pago, com o texto dele: %+v", s)
	}
	if s.amount != 40 || s.turns != 3 || s.busts != 2 || s.doubles != 1 || len(s.finds) != 1 {
		t.Errorf("os turnos somam ganho, contagem e achados: %+v", s)
	}
	if all := bust.merge(bust); !all.busted || all.amount != -60 {
		t.Errorf("só turno que deu ruim continua dando ruim e soma as multas: %+v", all)
	}
}

func TestGatherBacklog(t *testing.T) {
	ctx := context.Background()
	store, err := core.OpenMessageStore(ctx, filepath.Join(t.TempDir(), "mensagens.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	s := &walletStore{db: &ledgerDB{db: store.DB}}
	chat := types.NewJID("120363000000000001", types.GroupServer)
	now := time.Unix(time.Now().Unix(), 0)
	seq := 0
	fresh := func(t *testing.T, column string, idle time.Duration, items map[string]int64) core.Account {
		t.Helper()
		seq++
		n := fmt.Sprintf("55119%08d", seq)
		acc := core.Account{Key: n + "@s.whatsapp.net", JID: types.NewJID(n, types.DefaultUserServer)}
		err := s.inTx(ctx, func(tx *ledgerTx) error {
			if err := openWallet(ctx, tx, chat, acc); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, "UPDATE carteiras SET "+column+" = ? WHERE chat = ? AND pessoa = ?", now.Add(-idle).Unix(), chat.String(), acc.Key); err != nil {
				return err
			}
			for id, n := range items {
				if err := addItems(ctx, tx, chat, acc, id, n); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		return acc
	}
	held := func(t *testing.T, acc core.Account, id string) int64 {
		t.Helper()
		bag, err := s.inventory(ctx, chat, acc)
		if err != nil {
			t.Fatal(err)
		}
		return quantityOf(bag, id)
	}

	t.Run("a espera parada vira tentativas, até uma hora", func(t *testing.T) {
		for idle, want := range map[time.Duration]int{25 * time.Minute: 2, 10 * time.Hour: 6} {
			acc := fresh(t, "lixao_em", idle, nil)
			g, err := s.gather(ctx, chat, acc, scavenging, crater{}, now)
			if err != nil {
				t.Fatal(err)
			}
			if g.turns != want || len(g.backlog) != want-1 {
				t.Errorf("parado(a) há %s: %d tentativas e %d acumuladas, queria %d", idle, g.turns, len(g.backlog), want)
			}
			if again, err := s.gather(ctx, chat, acc, scavenging, crater{}, now); err != nil || again.wait != scavenging.cooldown.every {
				t.Errorf("logo depois a espera volta inteira, deu %s (%v)", again.wait, err)
			}
		}
	})

	t.Run("o consumível é gasto em cada tentativa até acabar", func(t *testing.T) {
		acc := fresh(t, "pesca_em", 10*time.Hour, map[string]int64{"vara-de-titanio": 1, "isca": 2})
		g, err := s.gather(ctx, chat, acc, fishing, crater{}, now)
		if err != nil {
			t.Fatal(err)
		}
		spent := 0
		for _, it := range g.spent {
			if it.id == "isca" {
				spent++
			}
		}
		if g.turns <= 2 || spent != 2 || held(t, acc, "isca") != 0 {
			t.Errorf("%d tentativas gastaram %d iscas e sobraram %d, queria as 2 gastas", g.turns, spent, held(t, acc, "isca"))
		}
	})

	t.Run("sem ferramenta as tentativas param", func(t *testing.T) {
		fragile := fishing
		fragile.breaks = 1
		acc := fresh(t, "pesca_em", 10*time.Hour, map[string]int64{"vara-de-pesca": 2})
		g, err := s.gather(ctx, chat, acc, fragile, crater{}, now)
		if err != nil {
			t.Fatal(err)
		}
		if g.turns != 2 || g.broken.id != "vara-de-pesca" || held(t, acc, "vara-de-pesca") != 0 {
			t.Errorf("com 2 varas que sempre quebram: %d tentativas, quebrou %q, sobraram %d", g.turns, g.broken.id, held(t, acc, "vara-de-pesca"))
		}
	})
}
