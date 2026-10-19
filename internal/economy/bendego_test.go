package economy

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

func TestItemPagesRender(t *testing.T) {
	for _, it := range catalog {
		if it.details() == "" {
			t.Errorf("%s: página do !item vazia", it.id)
		}
		if len(sources(it)) == 0 && it.rarity >= mythic {
			t.Errorf("%s: sem lugar pra conseguir", it.id)
		}
	}
}

func TestSkyRockPrizes(t *testing.T) {
	rock, _ := itemByID(skyRockID)
	shards := 0
	seen := map[int]bool{}
	const opens = 4000
	for range opens {
		prize := rock.roll(0).prize
		if prize.id == bendegoShardID {
			shards++
			continue
		}
		if prize.rarity != legendary && prize.rarity != mythic {
			t.Fatalf("a Rocha celeste deu %s (%s), e ela só dá fragmento, lendário ou mítico", prize.id, rarities[prize.rarity].name)
		}
		seen[prize.rarity] = true
	}
	if shards < opens*40/100 || shards > opens*60/100 {
		t.Errorf("a Rocha celeste deu fragmento em %d de %d aberturas, longe dos %d%%", shards, opens, percent(rock.shardOdds))
	}
	if !seen[legendary] || !seen[mythic] {
		t.Errorf("a Rocha celeste tem que dar lendário e mítico quando não dá fragmento: %v", seen)
	}
}

func TestBendegoExtraction(t *testing.T) {
	bag := func(ids ...string) []stack {
		var held []stack
		for _, id := range ids {
			it, _ := itemByID(id)
			held = append(held, stack{item: it, quantity: 1})
		}
		return held
	}
	hasMythic := func(k kit) bool {
		return slices.ContainsFunc(k.loot, func(d drop) bool { it, _ := itemByID(d.item); return it.rarity == mythic })
	}
	hasEmpty := func(k kit) bool {
		return slices.ContainsFunc(k.loot, func(d drop) bool { return d.item == "" })
	}
	owner := core.Account{Key: "dono", JID: types.NewJID("5511900000009", types.DefaultUserServer)}
	cases := []struct {
		name     string
		a        activity
		held     []stack
		starfall bool
		mythics  bool
		pull     int
	}{
		{"picareta comum", mining, bag("picareta"), false, false, 0},
		{"relíquia de extração", mining, bag("picareta", "picareta-de-serra-pelada"), true, true, 0},
		{"vela não é relíquia de extração", mining, bag("picareta", "vela-do-negrinho-do-pastoreio"), false, true, 0},
		{"bendegó sem relíquia", mining, bag("picareta", bendegoID), false, true, 1},
		{"bendegó com relíquia no !pescar", fishing, bag("vara-de-pesca", "vara-de-iemanja", bendegoID), true, true, 1},
		{"cratera do bendegó", crater{owner: owner, until: time.Now().Add(time.Hour)}.dig(mining), bag("picareta"), false, true, 0},
	}
	for _, c := range cases {
		k := c.a.equip(c.held)
		if k.starfall != c.starfall {
			t.Errorf("%s: chance da Rocha celeste = %v, queria %v", c.name, k.starfall, c.starfall)
		}
		if hasMythic(k) != c.mythics {
			t.Errorf("%s: mítico no sorteio = %v, queria %v", c.name, hasMythic(k), c.mythics)
		}
		if k.pull != c.pull {
			t.Errorf("%s: itens a mais por extração = %d, queria %d", c.name, k.pull, c.pull)
		}
		if c.pull > 0 && hasEmpty(k) {
			t.Errorf("%s: com o Bendegó a extração nunca vem de mão vazia", c.name)
		}
	}
}

func TestBendegoStore(t *testing.T) {
	ctx := context.Background()
	store, err := core.OpenMessageStore(ctx, filepath.Join(t.TempDir(), "mensagens.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	s := &walletStore{db: &ledgerDB{db: store.DB}}
	chat := types.NewJID("120363000000000001", types.GroupServer)
	holder := core.Account{Key: "5511900000001@s.whatsapp.net", JID: types.NewJID("5511900000001", types.DefaultUserServer)}
	worker := core.Account{Key: "5511900000002@s.whatsapp.net", JID: types.NewJID("5511900000002", types.DefaultUserServer)}
	now := time.Now()

	tx := func(t *testing.T, fn func(tx *ledgerTx) error) {
		t.Helper()
		if err := s.inTx(ctx, fn); err != nil {
			t.Fatal(err)
		}
	}
	held := func(t *testing.T, acc core.Account, id string) int64 {
		t.Helper()
		bag, err := s.inventory(ctx, chat, acc)
		if err != nil {
			t.Fatal(err)
		}
		return quantityOf(bag, id)
	}
	balance := func(t *testing.T, acc core.Account) int64 {
		t.Helper()
		var saldo int64
		if err := store.DB.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&saldo); err != nil {
			t.Fatal(err)
		}
		return saldo
	}
	tx(t, func(tx *ledgerTx) error {
		for _, acc := range []core.Account{holder, worker} {
			if err := openWallet(ctx, tx, chat, acc); err != nil {
				return err
			}
		}
		_, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = 100000 WHERE chat = ?", chat.String())
		return err
	})

	t.Run("quatro fragmentos viram o Bendegó", func(t *testing.T) {
		tx(t, func(tx *ledgerTx) error { return addItems(ctx, tx, chat, holder, bendegoShardID, 3) })
		if got := held(t, holder, bendegoShardID); got != 3 {
			t.Fatalf("com 3 fragmentos a mochila tem que ficar com os 3, ficou com %d", got)
		}
		tx(t, func(tx *ledgerTx) error { return addItems(ctx, tx, chat, holder, bendegoShardID, 2) })
		if got := held(t, holder, bendegoShardID); got != 1 {
			t.Errorf("com 5 fragmentos sobra 1 depois de montar, sobraram %d", got)
		}
		if got := held(t, holder, bendegoID); got != 1 {
			t.Errorf("com 5 fragmentos sai 1 Bendegó, saíram %d", got)
		}
		built, err := s.collectAssemblies(ctx, chat)
		if err != nil {
			t.Fatal(err)
		}
		if len(built) != 1 || built[0].owner != holder.JID || built[0].whole.id != bendegoID || built[0].quantity != 1 {
			t.Errorf("aviso da montagem errado: %+v", built)
		}
		if again, _ := s.collectAssemblies(ctx, chat); len(again) != 0 {
			t.Errorf("o aviso da montagem sai uma vez só, saiu de novo: %+v", again)
		}
	})

	t.Run("campo gravitacional puxa do trabalho dos outros e paga a cada 6 horas", func(t *testing.T) {
		var pulled, own int64
		tx(t, func(tx *ledgerTx) (err error) {
			if pulled, err = pullGravity(ctx, tx, chat, worker, 1000, now); err != nil {
				return err
			}
			own, err = pullGravity(ctx, tx, chat, holder, 1000, now)
			return err
		})
		if pulled != 1000*gravityPercent/100 {
			t.Errorf("o campo puxou %d de quem trabalhou 1000, queria %d", pulled, 1000*gravityPercent/100)
		}
		if own != 0 {
			t.Errorf("o campo não puxa do próprio dono do Bendegó, puxou %d", own)
		}
		before := balance(t, holder)
		early, err := s.payGravity(ctx, chat, now.Add(gravityEvery-time.Minute))
		if err != nil {
			t.Fatal(err)
		}
		if early.share != 0 {
			t.Errorf("antes de %s o campo não paga, pagou %d", gravityEvery, early.share)
		}
		p, err := s.payGravity(ctx, chat, now.Add(gravityEvery))
		if err != nil {
			t.Fatal(err)
		}
		if p.share != pulled || len(p.holders) != 1 || p.holders[0] != holder.JID {
			t.Errorf("pagamento do campo errado: %+v", p)
		}
		if got := balance(t, holder) - before; got != pulled {
			t.Errorf("o dono recebeu %d do campo, queria %d", got, pulled)
		}
	})

	t.Run("quem minera na cratera paga pedágio pro dono", func(t *testing.T) {
		tx(t, func(tx *ledgerTx) error {
			for _, id := range []string{"picareta", "vela-do-negrinho-do-pastoreio"} {
				if err := addItems(ctx, tx, chat, worker, id, 1); err != nil {
					return err
				}
			}
			return nil
		})
		pit := crater{owner: holder, until: now.Add(craterOpen)}
		workerBefore, holderBefore := balance(t, worker), balance(t, holder)
		g, err := s.gather(ctx, chat, worker, pit.dig(mining), pit, now)
		if err != nil {
			t.Fatal(err)
		}
		if g.found.id == "" {
			t.Fatal("com a Vela do Negrinho do Pastoreio a extração sempre acha alguma coisa")
		}
		want := marketPayoutOf(g.found, []grade{{g.grade, 1}}, marketSalePercent) * craterToll / 100
		if g.toll != want {
			t.Errorf("pedágio de %d por %s, queria %d", g.toll, g.found.id, want)
		}
		if paid, got := workerBefore-balance(t, worker), balance(t, holder)-holderBefore; paid != g.toll || got != g.toll {
			t.Errorf("o minerador pagou %d e o dono recebeu %d, os dois tinham que ser %d", paid, got, g.toll)
		}
	})
}
