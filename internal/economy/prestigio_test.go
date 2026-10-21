package economy

import (
	"errors"
	"testing"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
)

func (fb firmBench) rebirth(t *testing.T, acc core.Account) error {
	t.Helper()
	_, err := fb.s.rebirth(fb.ctx, fb.chat, acc, fb.now)
	return err
}

func (fb firmBench) count(t *testing.T, table, owner string, acc core.Account) int {
	t.Helper()
	var n int
	err := fb.s.db.QueryRowContext(fb.ctx, "SELECT COUNT(*) FROM "+table+" WHERE chat = ? AND "+owner+" = ?", fb.chat.String(), acc.Key).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func (fb firmBench) prestigeLevel(t *testing.T, acc core.Account) int {
	t.Helper()
	st, err := fb.s.standing(fb.ctx, fb.chat, acc)
	if err != nil {
		t.Fatal(err)
	}
	return st.prestige
}

func TestPrestigeNeedsWorthThatGrowsEachLevel(t *testing.T) {
	fb := newFirmBench(t)
	ana := fb.person("1")

	fb.setBalance(t, ana, prestigeCost(0)-1)
	if err := fb.rebirth(t, ana); !errors.Is(err, errPrestigeShort) {
		t.Fatalf("renascer com 1 moeda a menos = %v, want errPrestigeShort", err)
	}

	fb.setBalance(t, ana, 3_000_000)
	fb.exec(t, "UPDATE carteiras SET cofre_casa = 1000000 WHERE chat = ? AND pessoa = ?", fb.chat.String(), ana.Key)
	fb.exec(t, "INSERT INTO poupancas (chat, pessoa, valor, juros, vence_em) VALUES (?, ?, 1000000, 0, 0)", fb.chat.String(), ana.Key)
	if err := fb.rebirth(t, ana); err != nil {
		t.Fatalf("renascer com saldo, cofre e poupança somando 5M: %v", err)
	}
	if level := fb.prestigeLevel(t, ana); level != 1 {
		t.Fatalf("prestígio depois de renascer = %d, want 1", level)
	}

	fb.setBalance(t, ana, prestigeCost(0))
	if err := fb.rebirth(t, ana); !errors.Is(err, errPrestigeShort) {
		t.Fatalf("segundo prestígio com o preço do primeiro = %v, want errPrestigeShort", err)
	}
	fb.setBalance(t, ana, prestigeCost(1))
	if err := fb.rebirth(t, ana); err != nil {
		t.Fatalf("segundo prestígio com %d: %v", prestigeCost(1), err)
	}
	if level := fb.prestigeLevel(t, ana); level != 2 {
		t.Fatalf("prestígio depois de renascer de novo = %d, want 2", level)
	}
}

func TestPrestigeWipesWealthAndKeepsTheRest(t *testing.T) {
	fb := newFirmBench(t)
	ana, bia := fb.person("1"), fb.person("2")
	chat := fb.chat.String()

	fb.setBalance(t, ana, prestigeCost(0))
	fb.setBalance(t, bia, 100)
	fb.exec(t, "UPDATE carteiras SET imovel = 3, titulo = 'barao', cofre_casa = 500, canteiros_comprados = 2 WHERE chat = ? AND pessoa = ?", chat, ana.Key)
	fb.exec(t, "INSERT INTO titulos (chat, titulo, pessoa, jid, pago, em) VALUES (?, 'barao', ?, ?, 30000, 0)", chat, ana.Key, ana.JID.String())
	fb.exec(t, "INSERT INTO poupancas (chat, pessoa, valor, juros, vence_em) VALUES (?, ?, 100, 0, 0)", chat, ana.Key)
	fb.exec(t, "INSERT INTO acoes_posicoes (chat, pessoa, empresa, quantidade, custo) VALUES (?, ?, 'PETR', 10, 100)", chat, ana.Key)
	fb.exec(t, "INSERT INTO empregados (chat, pessoa, tipo, coletado_em) VALUES (?, ?, 'pescador', 0)", chat, ana.Key)
	fb.exec(t, "INSERT INTO implantes (chat, pessoa, parte, implante, instalado_em, pago_ate) VALUES (?, ?, 'olhos', 'x', 0, 0)", chat, ana.Key)
	fb.exec(t, "INSERT INTO plantacoes (chat, pessoa, semente, pronta_em) VALUES (?, ?, 'x', 0)", chat, ana.Key)
	fb.exec(t, "INSERT INTO feira_anuncios (chat, vendedor, vendedor_jid, item, quantidade, preco, expira_em) VALUES (?, ?, ?, 'x', 1, 10, 0)", chat, ana.Key, ana.JID.String())
	fb.exec(t, "INSERT INTO carreiras (chat, pessoa, profissao, minutos) VALUES (?, ?, 'bico', 600)", chat, ana.Key)
	for _, acc := range []core.Account{ana, bia} {
		fb.exec(t, "INSERT INTO inventarios (chat, pessoa, item, quantidade) VALUES (?, ?, 'x', 1)", chat, acc.Key)
		fb.exec(t, "INSERT INTO guardados (chat, pessoa, item, quantidade) VALUES (?, ?, 'x', 1)", chat, acc.Key)
	}

	if err := fb.rebirth(t, ana); err != nil {
		t.Fatal(err)
	}

	var (
		balance, vault, beds int64
		estate               int
		title                string
	)
	err := fb.s.db.QueryRowContext(fb.ctx, "SELECT saldo, cofre_casa, imovel, titulo, canteiros_comprados FROM carteiras WHERE chat = ? AND pessoa = ?", chat, ana.Key).
		Scan(&balance, &vault, &estate, &title, &beds)
	if err != nil {
		t.Fatal(err)
	}
	if balance != 0 || vault != 0 || estate != 0 || title != "" || beds != 0 {
		t.Fatalf("carteira depois de renascer: saldo %d, cofre %d, imóvel %d, título %q, canteiros %d; want tudo zerado", balance, vault, estate, title, beds)
	}
	wiped := map[string]string{
		"inventarios": "pessoa", "guardados": "pessoa", "poupancas": "pessoa", "acoes_posicoes": "pessoa",
		"empregados": "pessoa", "implantes": "pessoa", "plantacoes": "pessoa", "titulos": "pessoa", "feira_anuncios": "vendedor",
	}
	for table, owner := range wiped {
		if n := fb.count(t, table, owner, ana); n != 0 {
			t.Errorf("%s depois de renascer tem %d linhas, want 0", table, n)
		}
	}
	if n := fb.count(t, "carreiras", "pessoa", ana); n != 1 {
		t.Errorf("carreiras depois de renascer = %d, want 1", n)
	}
	if fb.balance(t, bia) != 100 || fb.count(t, "inventarios", "pessoa", bia) != 1 || fb.count(t, "guardados", "pessoa", bia) != 1 {
		t.Error("renascer mexeu nas coisas de outra pessoa")
	}
	var lost int64
	err = fb.s.db.QueryRowContext(fb.ctx, "SELECT valor FROM prejuizos WHERE chat = ? AND pessoa = ? AND motivo = ?", chat, ana.Key, string(lossReborn)).Scan(&lost)
	if err != nil {
		t.Fatal(err)
	}
	if lost != prestigeCost(0)+600 {
		t.Errorf("prejuízo anotado no prestígio = %d, want %d", lost, prestigeCost(0)+600)
	}
}

func TestPrestigeRefusesGodPresidentDebtorAndTheTop(t *testing.T) {
	fb := newFirmBench(t)
	god, president, debtor, top := fb.person("1"), fb.person("2"), fb.person("3"), fb.person("4")
	chat := fb.chat.String()
	for _, acc := range []core.Account{god, president, debtor, top} {
		fb.setBalance(t, acc, prestigeCost(prestigeTop))
	}
	fb.exec(t, "UPDATE carteiras SET profissao = 'deus' WHERE chat = ? AND pessoa = ?", chat, god.Key)
	fb.tyrant(t, president)
	fb.exec(t, "INSERT INTO dividas (chat, devedor, devedor_jid, credor, credor_jid, divida, juros, juros_atraso, desde, vence_em) VALUES (?, ?, ?, ?, ?, 100, 0, 0, 0, 0)",
		chat, debtor.Key, debtor.JID.String(), top.Key, top.JID.String())
	fb.exec(t, "UPDATE carteiras SET prestigio = ? WHERE chat = ? AND pessoa = ?", prestigeTop, chat, top.Key)

	cases := []struct {
		who  string
		acc  core.Account
		want error
	}{
		{"deus", god, errPrestigeDivine},
		{"presidente", president, errPrestigePresident},
		{"quem deve no agiota", debtor, errPrestigeDebt},
		{"quem tá no máximo", top, errPrestigeTop},
	}
	for _, c := range cases {
		if err := fb.rebirth(t, c.acc); !errors.Is(err, c.want) {
			t.Errorf("%s renascendo = %v, want %v", c.who, err, c.want)
		}
		if fb.balance(t, c.acc) != prestigeCost(prestigeTop) {
			t.Errorf("%s perdeu o saldo num prestígio recusado", c.who)
		}
	}

	fb.exec(t, "DELETE FROM dividas")
	fb.exec(t, "INSERT INTO emprestimos (chat, pessoa, jid, divida, juros, juros_atraso, desde, vence_em, penhora_em) VALUES (?, ?, ?, 100, 0, 0, 0, 0, 0)",
		chat, debtor.Key, debtor.JID.String())
	if err := fb.rebirth(t, debtor); !errors.Is(err, errPrestigeDebt) {
		t.Errorf("quem deve no banco renascendo = %v, want errPrestigeDebt", err)
	}
}

func TestPrestigeBonusIsTwoPercentPerLevel(t *testing.T) {
	job := Profession{min: 1000, max: 2000, dailyBonus: 5}.prestiged(prestigeTop)
	if job.min != 1100 || job.max != 2200 || job.dailyBonus != 15 || job.prestige != prestigeTop {
		t.Fatalf("prestígio %d: min %d, max %d, daily +%d%%; want 1100, 2200 e +15%%", prestigeTop, job.min, job.max, job.dailyBonus)
	}
	deity := god.prestiged(3)
	if deity.min != god.min || deity.max != god.max || deity.dailyBonus != god.dailyBonus || deity.prestige != 3 {
		t.Fatal("deus ganhou bônus de prestígio")
	}
}
