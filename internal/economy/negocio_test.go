package economy

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
)

func lemonade(t *testing.T) business {
	t.Helper()
	biz, ok := businessByID("barraca-de-limonada")
	if !ok {
		t.Fatal("barraca de limonada sumiu do catálogo")
	}
	return biz
}

func (fb firmBench) venture(t *testing.T, acc core.Account, id string) venture {
	t.Helper()
	owned, err := fb.s.ventures(fb.ctx, fb.chat, acc)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range owned {
		if v.kind.id == id {
			return v
		}
	}
	t.Fatalf("%s não tem o negócio %s", acc.Key, id)
	return venture{}
}

func (fb firmBench) openLemonade(t *testing.T, acc core.Account) {
	t.Helper()
	fb.setBalance(t, acc, lemonade(t).price)
	if _, err := fb.s.openVenture(fb.ctx, fb.chat, acc, lemonade(t), fb.now); err != nil {
		t.Fatal(err)
	}
}

func TestVentureSellsUntilTheStockRunsOutOrTheTillFills(t *testing.T) {
	biz := lemonade(t)
	start := time.Unix(1_700_000_000, 0)
	normal, cheap, dear := pricingOf("normal"), pricingOf("barato"), pricingOf("caro")
	cases := []struct {
		name    string
		v       venture
		after   time.Duration
		sold    int64
		gross   int64
		idle    time.Duration
		dry     bool
		full    bool
		advance time.Duration
	}{
		{"vende 4 copos por hora no normal", venture{kind: biz, level: 1, stock: 96, pricing: normal, since: start}, time.Hour + 10*time.Minute, 4, 40, 0, false, false, time.Hour},
		{"o barato vende o dobro mais barato", venture{kind: biz, level: 1, stock: 96, pricing: cheap, since: start}, time.Hour, 8, 64, 0, false, false, time.Hour},
		{"o nível 2 vende 50% a mais", venture{kind: biz, level: 2, stock: 144, pricing: normal, since: start}, time.Hour, 6, 60, 0, false, false, time.Hour},
		{"para quando o estoque acaba", venture{kind: biz, level: 1, stock: 3, pricing: normal, since: start}, 2 * time.Hour, 3, 30, 75 * time.Minute, true, false, 2 * time.Hour},
		{"o caro para quando o caixa enche", venture{kind: biz, level: 1, stock: 96, pricing: dear, since: start}, 48 * time.Hour, 73, 949, 48*time.Hour - 73*30*time.Minute, false, true, 48 * time.Hour},
		{"nada vende antes da hora", venture{kind: biz, level: 1, stock: 96, pricing: normal, since: start}, 10 * time.Minute, 0, 0, 0, false, false, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.v.sell(start.Add(c.after))
			if got.sold != c.sold || got.gross != c.gross || got.idle != c.idle || got.dry != c.dry || got.full != c.full {
				t.Fatalf("sell = vendeu %d por %d, parado %s, sem estoque %v, caixa cheio %v; want %d por %d, parado %s, %v, %v",
					got.sold, got.gross, got.idle, got.dry, got.full, c.sold, c.gross, c.idle, c.dry, c.full)
			}
			if want := start.Add(c.advance); !got.until.Equal(want) {
				t.Fatalf("vendido até %s, want %s", got.until, want)
			}
		})
	}
}

func TestEveryBusinessSellsForWholeCoinsAndTakesLongerToPayBackAsItGrows(t *testing.T) {
	normal := pricingOf(normalPricing)
	var last int64
	for _, biz := range businesses {
		for _, p := range pricings {
			if biz.cost*p.markup%100 != 0 {
				t.Errorf("%s no preço %s sai por %d centavos de moeda", biz.name, p.id, biz.cost*p.markup%100)
			}
		}
		profit := biz.hourlyProfit(1, normal)
		if profit <= 0 {
			t.Fatalf("%s não lucra nada no preço normal", biz.name)
		}
		payback := biz.price / profit
		if payback < 90 || payback > 450 {
			t.Errorf("%s se paga em %d horas, want entre 90 e 450", biz.name, payback)
		}
		if payback < last {
			t.Errorf("%s se paga em %d horas, mais rápido que o negócio mais barato antes dele (%d)", biz.name, payback, last)
		}
		last = payback
	}
}

func TestOpenVentureChargesThePriceAndFillsTheShelfOncePerPerson(t *testing.T) {
	fb := newFirmBench(t)
	ana, bia := fb.person("1"), fb.person("2")
	biz := lemonade(t)

	fb.setBalance(t, ana, 5_000)
	balance, err := fb.s.openVenture(fb.ctx, fb.chat, ana, biz, fb.now)
	if err != nil {
		t.Fatal(err)
	}
	if balance != 3_000 || fb.balance(t, ana) != 3_000 {
		t.Fatalf("saldo depois de abrir = %d (banco %d), want 3000", balance, fb.balance(t, ana))
	}
	v := fb.venture(t, ana, biz.id)
	if v.level != 1 || v.stock != 96 || v.pricing.id != normalPricing || !v.since.Equal(fb.now) {
		t.Fatalf("negócio aberto = nível %d, estoque %d, preço %s, desde %s; want nível 1, estoque 96, normal, agora", v.level, v.stock, v.pricing.id, v.since)
	}

	if _, err := fb.s.openVenture(fb.ctx, fb.chat, ana, biz, fb.now); !errors.Is(err, errBusinessOwned) {
		t.Fatalf("abrir o mesmo negócio de novo: err = %v, want errBusinessOwned", err)
	}
	if fb.balance(t, ana) != 3_000 {
		t.Fatalf("a segunda compra mexeu no saldo: %d", fb.balance(t, ana))
	}

	fb.setBalance(t, bia, biz.price-1)
	if _, err := fb.s.openVenture(fb.ctx, fb.chat, bia, biz, fb.now); !errors.Is(err, ErrNoFunds) {
		t.Fatalf("abrir sem saldo: err = %v, want ErrNoFunds", err)
	}
	if n := fb.count(t, "negocios", "pessoa", bia); n != 0 {
		t.Fatalf("quem não pagou ficou com %d negócios", n)
	}
}

func TestCashOutMovesEveryTillToTheBalance(t *testing.T) {
	fb := newFirmBench(t)
	ana := fb.person("1")
	fb.openLemonade(t, ana)
	pipoca, _ := businessByID("carrinho-de-pipoca")
	fb.setBalance(t, ana, pipoca.price)
	if _, err := fb.s.openVenture(fb.ctx, fb.chat, ana, pipoca, fb.now); err != nil {
		t.Fatal(err)
	}

	later := fb.now.Add(2 * time.Hour)
	vis, err := fb.s.cashOut(fb.ctx, fb.chat, ana, later)
	if err != nil {
		t.Fatal(err)
	}
	if vis.gross() != 80+200 || vis.balance != 280 || fb.balance(t, ana) != 280 {
		t.Fatalf("coleta = caixa %d, saldo %d (banco %d); want 280 dos dois negócios", vis.gross(), vis.balance, fb.balance(t, ana))
	}
	if v := fb.venture(t, ana, "barraca-de-limonada"); v.stock != 88 || !v.since.Equal(later) {
		t.Fatalf("limonada depois da coleta = estoque %d, desde %s; want 88 e a hora da coleta", v.stock, v.since)
	}

	vis, err = fb.s.cashOut(fb.ctx, fb.chat, ana, later)
	if err != nil {
		t.Fatal(err)
	}
	if vis.gross() != 0 || fb.balance(t, ana) != 280 {
		t.Fatalf("coletar de novo na mesma hora rendeu %d", vis.gross())
	}

	if _, err := fb.s.cashOut(fb.ctx, fb.chat, fb.person("2"), later); !errors.Is(err, errNoBusiness) {
		t.Fatalf("coletar sem negócio: err = %v, want errNoBusiness", err)
	}
}

func TestRestockBuysWhatTheBalanceAffordsAfterCashingTheTill(t *testing.T) {
	fb := newFirmBench(t)
	ana := fb.person("1")
	fb.openLemonade(t, ana)
	fb.exec(t, "UPDATE negocios SET estoque = 0 WHERE chat = ? AND pessoa = ?", fb.chat.String(), ana.Key)
	fb.setBalance(t, ana, 100)

	vis, err := fb.s.restock(fb.ctx, fb.chat, ana, "", fb.now.Add(3*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if vis.spent() != 100 || vis.balance != 0 || fb.venture(t, ana, "barraca-de-limonada").stock != 20 {
		t.Fatalf("repor com 100 de saldo gastou %d, sobrou %d e deixou %d no estoque; want 100, 0 e 20", vis.spent(), vis.balance, fb.venture(t, ana, "barraca-de-limonada").stock)
	}

	fb.setBalance(t, ana, 10_000)
	vis, err = fb.s.restock(fb.ctx, fb.chat, ana, "barraca-de-limonada", fb.now.Add(4*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	v := fb.venture(t, ana, "barraca-de-limonada")
	if vis.gross() != 40 || vis.stops[0].bought != 80 || v.stock != 96 {
		t.Fatalf("repor com saldo = caixa %d, comprou %d, estoque %d; want caixa 40, comprou 80 e estoque cheio de 96", vis.gross(), vis.stops[0].bought, v.stock)
	}
	if want := int64(10_000 + 40 - 80*5); vis.balance != want || fb.balance(t, ana) != want {
		t.Fatalf("saldo depois de repor = %d (banco %d), want %d", vis.balance, fb.balance(t, ana), want)
	}
}

func TestUpgradeChargesTheReformWithTheTillHelpingAndStopsAtTheTop(t *testing.T) {
	fb := newFirmBench(t)
	ana := fb.person("1")
	biz := lemonade(t)
	fb.openLemonade(t, ana)

	fb.setBalance(t, ana, biz.upgradeCost(1))
	if _, err := fb.s.upgradeVenture(fb.ctx, fb.chat, ana, biz.id, fb.now); err != nil {
		t.Fatal(err)
	}
	if v := fb.venture(t, ana, biz.id); v.level != 2 || fb.balance(t, ana) != 0 {
		t.Fatalf("depois da reforma = nível %d, saldo %d; want nível 2 e saldo 0", v.level, fb.balance(t, ana))
	}

	fb.setBalance(t, ana, biz.upgradeCost(2)-50)
	if _, err := fb.s.upgradeVenture(fb.ctx, fb.chat, ana, biz.id, fb.now.Add(30*time.Minute)); !errors.Is(err, ErrNoFunds) {
		t.Fatalf("reforma sem saldo nem caixa que chegue: err = %v, want ErrNoFunds", err)
	}
	if v := fb.venture(t, ana, biz.id); v.level != 2 || v.stock != 96 {
		t.Fatalf("a reforma recusada mexeu no negócio: nível %d, estoque %d", v.level, v.stock)
	}
	if _, err := fb.s.upgradeVenture(fb.ctx, fb.chat, ana, biz.id, fb.now.Add(time.Hour)); err != nil {
		t.Fatalf("o caixa de 60 devia completar os 50 que faltavam: %v", err)
	}
	if v := fb.venture(t, ana, biz.id); v.level != 3 || fb.balance(t, ana) != 10 {
		t.Fatalf("reforma paga com o caixa = nível %d, saldo %d; want nível 3 e saldo 10", v.level, fb.balance(t, ana))
	}

	fb.exec(t, "UPDATE negocios SET nivel = ? WHERE chat = ? AND pessoa = ?", bizTop, fb.chat.String(), ana.Key)
	fb.setBalance(t, ana, 1_000_000)
	if _, err := fb.s.upgradeVenture(fb.ctx, fb.chat, ana, biz.id, fb.now); !errors.Is(err, errBusinessTop) {
		t.Fatalf("reformar no nível máximo: err = %v, want errBusinessTop", err)
	}
}

func TestCloseVenturePaysHalfOfThePriceAndReformsPlusTheTill(t *testing.T) {
	fb := newFirmBench(t)
	ana := fb.person("1")
	biz := lemonade(t)
	fb.openLemonade(t, ana)
	fb.exec(t, "UPDATE negocios SET nivel = 3 WHERE chat = ? AND pessoa = ?", fb.chat.String(), ana.Key)

	vis, err := fb.s.closeVenture(fb.ctx, fb.chat, ana, biz.id, fb.now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if want := int64((2_000+1_000+2_000)/2 + 80); vis.balance != want || fb.balance(t, ana) != want {
		t.Fatalf("saldo depois de fechar = %d (banco %d), want %d", vis.balance, fb.balance(t, ana), want)
	}
	if n := fb.count(t, "negocios", "pessoa", ana); n != 0 {
		t.Fatalf("o negócio fechado continua lá: %d linhas", n)
	}
}

func TestRepriceCashesTheTillAndChangesHowItSells(t *testing.T) {
	fb := newFirmBench(t)
	ana := fb.person("1")
	biz := lemonade(t)
	fb.openLemonade(t, ana)

	vis, err := fb.s.reprice(fb.ctx, fb.chat, ana, biz.id, pricingOf("caro"), fb.now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	v := fb.venture(t, ana, biz.id)
	if vis.gross() != 40 || v.pricing.id != "caro" || v.stock != 92 || v.unitPrice() != 13 {
		t.Fatalf("depois de mudar o preço = caixa %d, preço %s a %d, estoque %d; want caixa 40, caro a 13 e estoque 92", vis.gross(), v.pricing.id, v.unitPrice(), v.stock)
	}
}

func TestVentureNameComesAfterTheBusinessAndLosesFormatting(t *testing.T) {
	pastel, _ := businessByID("banca-de-pastel")
	owned := []venture{{kind: lemonade(t)}, {kind: pastel, title: "Pastel do Zé"}}
	cases := []struct {
		text  string
		kind  string
		title string
	}{
		{"pastel Pastelaria do *Zé*", "banca-de-pastel", "Pastelaria do Zé"},
		{"banca de pastel Pastel Master", "banca-de-pastel", "Pastel Master"},
		{"Pastel do Zé Pastel Master", "banca-de-pastel", "Pastel Master"},
		{"limonada", "barraca-de-limonada", ""},
		{"limonada " + strings.Repeat("a", 60), "barraca-de-limonada", strings.Repeat("a", bizNameMax)},
	}
	for _, c := range cases {
		v, title, ok := splitVentureName(owned, strings.Fields(c.text))
		if !ok || v.kind.id != c.kind || title != c.title {
			t.Errorf("splitVentureName(%q) = %s, %q, %v; want %s, %q", c.text, v.kind.id, title, ok, c.kind, c.title)
		}
	}
	if _, _, ok := splitVentureName(owned, strings.Fields("concessionaria Carros do Zé")); ok {
		t.Error("deu nome pra um negócio que a pessoa nem tem")
	}
}
