package economy

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

func TestGardenCatalog(t *testing.T) {
	garden := slices.IndexFunc(itemCategories, func(c itemCategory) bool { return c.name == "Horta" })
	for _, seed := range seeds() {
		c := seed.seed
		produce, ok := itemByID(c.yields)
		if !ok {
			t.Errorf("%s: vira %q, que não tá no catálogo", seed.id, c.yields)
			continue
		}
		if produce.seed != nil {
			t.Errorf("%s: vira outra semente (%s)", seed.id, produce.id)
		}
		if c.least < 1 || c.most < c.least || c.grows <= 0 || c.wild <= 0 {
			t.Errorf("%s: crescimento inválido %+v", seed.id, *c)
		}
		if seed.gradable() || !produce.gradable() {
			t.Errorf("%s: semente não tem qualidade e o que ela vira tem", seed.id)
		}
		if seed.category() != garden {
			t.Errorf("%s: semente fora do tipo Horta", seed.id)
		}
		if effectPerk(seed) == "" {
			t.Errorf("%s: semente sem efeito sai no !vender inuteis", seed.id)
		}
		if !slices.Contains(sources(produce), "!colher (plantando "+seed.label()+")") {
			t.Errorf("%s: o Onde conseguir não fala do !colher: %v", produce.id, sources(produce))
		}
		if !seed.shop && seed.rarity < epic {
			t.Errorf("%s: semente comum até rara fica na !loja", seed.id)
		}
	}
	if picked := plantable(findItems("tomate")); len(picked) != 1 || picked[0].id != "semente-de-tomate" {
		t.Errorf("!plantar tomate tem que achar a semente de tomate, achou %v", picked)
	}
	if picked := plantable(findItems("rosa")); len(picked) != 1 || picked[0].id != "muda-de-roseira" {
		t.Errorf("!plantar rosa tem que achar a muda de roseira, achou %v", picked)
	}
}

func TestStraySeed(t *testing.T) {
	sure := bugging
	sure.seeds = 1
	if found := sure.stray(); found.seed == nil {
		t.Errorf("com chance 1 a extração sempre acha semente, achou %q", found.id)
	}
	never := bugging
	never.seeds = 0
	if found := never.stray(); found.id != "" {
		t.Errorf("sem chance a extração não acha semente, achou %q", found.id)
	}
}

func TestGardenStore(t *testing.T) {
	ctx := context.Background()
	store, err := core.OpenMessageStore(ctx, filepath.Join(t.TempDir(), "mensagens.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	s := &walletStore{db: &ledgerDB{db: store.DB}}
	chat := types.NewJID("120363000000000001", types.GroupServer)
	farmer := core.Account{Key: "5511900000001@s.whatsapp.net", JID: types.NewJID("5511900000001", types.DefaultUserServer)}
	corn, _ := itemByID("semente-de-milho")
	cob := corn.produce()
	now := time.Now()
	held := func(t *testing.T, id string) int64 {
		t.Helper()
		bag, err := s.inventory(ctx, chat, farmer)
		if err != nil {
			t.Fatal(err)
		}
		return quantityOf(bag, id)
	}
	if err := s.inTx(ctx, func(tx *ledgerTx) error { return addItems(ctx, tx, chat, farmer, corn.id, 5) }); err != nil {
		t.Fatal(err)
	}

	if err := s.sow(ctx, chat, farmer, corn, 3, gardenBeds, now.Add(corn.seed.grows)); err != nil {
		t.Fatal(err)
	}
	if got := held(t, corn.id); got != 2 {
		t.Errorf("plantar 3 de 5 sementes deixa 2 na mochila, deixou %d", got)
	}
	if err := s.sow(ctx, chat, farmer, corn, 2, gardenBeds, now); !errors.Is(err, errGardenFull) {
		t.Errorf("3 canteiros usados de %d não cabem mais 2, deu %v", gardenBeds, err)
	}
	if err := s.sow(ctx, chat, farmer, corn, 1, gardenBeds, now); err != nil {
		t.Fatal(err)
	}
	if err := s.inTx(ctx, func(tx *ledgerTx) error { return takeItems(ctx, tx, chat, farmer, corn.id, 1) }); err != nil {
		t.Fatal(err)
	}
	if err := s.sow(ctx, chat, farmer, corn, 1, gardenBeds+1, now); !errors.Is(err, errNoItems) {
		t.Errorf("sem semente na mochila não planta, deu %v", err)
	}

	r, err := s.reap(ctx, chat, farmer, 0, now)
	if err != nil {
		t.Fatal(err)
	}
	if r.reaped != 1 || len(r.growing) != 3 {
		t.Fatalf("na hora só o canteiro que já tava pronto sai, saíram %d e ficaram %d crescendo", r.reaped, len(r.growing))
	}
	if got := held(t, cob.id); got < corn.seed.least || got > corn.seed.most {
		t.Errorf("um canteiro de milho dá de %d a %d espigas, deu %d", corn.seed.least, corn.seed.most, got)
	}

	later, err := s.reap(ctx, chat, farmer, 0, now.Add(corn.seed.grows))
	if err != nil {
		t.Fatal(err)
	}
	if later.reaped != 3 || len(later.growing) != 0 || len(later.crops) != 1 {
		t.Fatalf("depois do tempo os 3 canteiros saem juntos: %+v", later)
	}
	if c := later.crops[0]; c.item.id != cob.id || c.quantity < 3*corn.seed.least || c.quantity > 3*corn.seed.most {
		t.Errorf("3 canteiros de milho dão de %d a %d espigas, deram %d de %s", 3*corn.seed.least, 3*corn.seed.most, c.quantity, c.item.id)
	}
	beds, err := s.beds(ctx, chat, farmer)
	if err != nil {
		t.Fatal(err)
	}
	if len(beds) != 0 {
		t.Errorf("depois de colher tudo a horta fica vazia, ficou com %d", len(beds))
	}
}

func TestGardenExpansion(t *testing.T) {
	ctx := context.Background()
	store, err := core.OpenMessageStore(ctx, filepath.Join(t.TempDir(), "mensagens.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	s := &walletStore{db: &ledgerDB{db: store.DB}}
	chat := types.NewJID("120363000000000001", types.GroupServer)
	farmer := core.Account{Key: "5511900000001@s.whatsapp.net", JID: types.NewJID("5511900000001", types.DefaultUserServer)}
	setBalance := func(t *testing.T, saldo int64) {
		t.Helper()
		err := s.inTx(ctx, func(tx *ledgerTx) error {
			if err := openWallet(ctx, tx, chat, farmer); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = ? WHERE chat = ? AND pessoa = ?", saldo, chat.String(), farmer.Key)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	if bought, err := s.boughtBeds(ctx, chat, farmer); err != nil || bought != 0 {
		t.Fatalf("quem nunca comprou canteiro tem 0 comprados, tem %d (%v)", bought, err)
	}
	setBalance(t, bedPrices[0]-1)
	if _, price, err := s.expandGarden(ctx, chat, farmer); !errors.Is(err, ErrNoFunds) || price != bedPrices[0] {
		t.Errorf("sem saldo pro primeiro canteiro (%d) a compra falha com ErrNoFunds, deu %v cobrando %d", bedPrices[0], err, price)
	}
	var total int64
	for _, p := range bedPrices {
		total += p
	}
	setBalance(t, total)
	for i, want := range bedPrices {
		bought, price, err := s.expandGarden(ctx, chat, farmer)
		if err != nil {
			t.Fatal(err)
		}
		if bought != i+1 || price != want {
			t.Errorf("a compra %d custa %d e deixa %d comprados, custou %d e deixou %d", i+1, want, i+1, price, bought)
		}
	}
	var saldo int64
	if err := store.DB.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), farmer.Key).Scan(&saldo); err != nil {
		t.Fatal(err)
	}
	if saldo != 0 {
		t.Errorf("os canteiros custam %d juntos e as moedas somem, sobraram %d", total, saldo)
	}
	if _, _, err := s.expandGarden(ctx, chat, farmer); !errors.Is(err, errGardenMaxed) {
		t.Errorf("depois de %d canteiros comprados não dá pra comprar mais, deu %v", len(bedPrices), err)
	}
	if bought, err := s.boughtBeds(ctx, chat, farmer); err != nil || bought != len(bedPrices) {
		t.Errorf("a horta fica com %d canteiros comprados, ficou com %d (%v)", len(bedPrices), bought, err)
	}
}
