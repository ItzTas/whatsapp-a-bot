package main

import (
	"regexp"
	"strings"
	"testing"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/economy"
)

func TestWhitelistGate(t *testing.T) {
	b := newBench(t)
	ana := newPerson("Ana", "5511990000002")
	chat := b.group("Galera", ana)

	wantSilence(t, b.say(chat, ana, "!comandos"))
	wantReply(t, b.say(chat, b.owner, "!whitelist"), "O grupo *Galera* entrou na whitelist")
	wantReply(t, b.say(chat, ana, "!comandos"), "!comandos")
}

func TestOwnerCommandsIgnoreOthers(t *testing.T) {
	b := newBench(t)
	ana := newPerson("Ana", "5511990000002")
	chat := b.group("Galera", ana)

	wantSilence(t, b.say(chat, ana, "!whitelist"))
	wantSilence(t, b.say(chat, ana, "!comandos"))
}

func TestMessagesRecordedOnlyWhereAllowed(t *testing.T) {
	b := newBench(t)
	ana := newPerson("Ana", "5511990000002")
	open := b.group("Galera", ana)
	closed := b.group("Trabalho", ana)
	b.say(open, b.owner, "!whitelist")

	b.say(open, ana, "bom dia")
	b.say(open, ana, "alguém aí?")
	b.say(closed, ana, "ninguém me ouve")

	counts := map[string]int{}
	rows, err := b.db.QueryContext(b.ctx, "SELECT chat, COUNT(*) FROM mensagens WHERE remetente = ? GROUP BY chat", ana.jid.User)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var chat string
		var n int
		if err := rows.Scan(&chat, &n); err != nil {
			t.Fatal(err)
		}
		counts[chat] = n
	}
	if counts[open.String()] != 2 || counts[closed.String()] != 0 {
		t.Fatalf("esperava 2 mensagens no grupo liberado e nenhuma no outro, ficou %v", counts)
	}
}

func TestModeGatesCommands(t *testing.T) {
	b := newBench(t)
	ana := newPerson("Ana", "5511990000002")
	chat := b.group("Galera", ana)
	b.say(chat, b.owner, "!whitelist")

	wantSilence(t, b.say(chat, ana, "!daily"))
	wantReply(t, b.say(chat, b.owner, "!modo ativar economia"), "Modo *economia* ativado")
	wantReply(t, b.say(chat, ana, "!daily"), "@"+ana.jid.User+" pegou o daily")
	wantReply(t, b.say(chat, b.owner, "!modo desativar economia"), "Modo *economia* desativado")
	wantSilence(t, b.say(chat, ana, "!saldo"))
}

func TestDailyOncePerDay(t *testing.T) {
	b := newBench(t)
	ana := newPerson("Ana", "5511990000002")
	chat := b.group("Galera", ana)
	b.setup(chat, "economia")

	wantReply(t, b.say(chat, ana, "!daily"), "pegou o daily e ganhou")
	earned := b.balance(ana)
	if earned <= 0 {
		t.Fatalf("o daily não pôs moeda na carteira, saldo %d", earned)
	}
	wantReply(t, b.say(chat, ana, "!diario"), "já pegou o daily hoje")
	if got := b.balance(ana); got != earned {
		t.Fatalf("o segundo daily mudou o saldo de %d pra %d", earned, got)
	}
}

func TestBetCeiling(t *testing.T) {
	b := newBench(t)
	ana := newPerson("Ana", "5511990000002")
	chat := b.group("Galera", ana)
	b.setup(chat, "economia")
	b.say(chat, ana, "!daily")
	start := b.balance(ana)
	most, whole := start*70/100, start*60/100

	wantReply(t, b.say(chat, ana, "!apostar 80%"), "dá pra apostar no máximo "+economy.Coins(most)+" de uma vez")
	if got := b.balance(ana); got != start {
		t.Fatalf("a aposta recusada mexeu no saldo: %d virou %d", start, got)
	}
	wantReply(t, b.say(chat, ana, "!apostar tudo"), economy.Coins(whole))
	if got := b.balance(ana); got != start-whole && got != start+whole {
		t.Fatalf("o !apostar tudo devia apostar %d de uma vez, mas o saldo foi de %d pra %d", whole, start, got)
	}
}

func TestTransfer(t *testing.T) {
	b := newBench(t)
	ana, bia := newPerson("Ana", "5511990000002"), newPerson("Bia", "5511990000003")
	chat := b.group("Galera", ana, bia)
	b.setup(chat, "economia")
	b.say(chat, ana, "!daily")
	start := b.balance(ana)

	wantReply(t, b.say(chat, ana, "!pix "+mentionTag(bia)+" 10", bia), "mandou")
	if got := b.balance(ana); got != start-10 {
		t.Fatalf("Ana devia ficar com %d, ficou com %d", start-10, got)
	}
	if got := b.balance(bia); got != 10 {
		t.Fatalf("Bia devia receber 10, tem %d", got)
	}
	wantReply(t, b.say(chat, bia, "!transferir "+mentionTag(ana)+" 999999", ana), "Você só tem")
	wantReply(t, b.say(chat, bia, "!saldo"), "@"+bia.jid.User+" tem")
}

func TestExclusiveModePausesEconomy(t *testing.T) {
	b := newBench(t)
	ana := newPerson("Ana", "5511990000002")
	chat := b.group("Galera", ana)
	b.setup(chat, "economia")

	wantReply(t, b.say(chat, b.owner, "!modo ativar cidadedorme"), "Modo *cidadedorme* ativado")
	wantSilence(t, b.say(chat, ana, "!daily"))
	wantReply(t, b.say(chat, b.owner, "!modo desativar cidadedorme"), "Modo *cidadedorme* desativado")
	wantReply(t, b.say(chat, ana, "!daily"), "pegou o daily")
}

func TestMarriageWithConsent(t *testing.T) {
	b := newBench(t)
	ana, bia := newPerson("Ana", "5511990000002"), newPerson("Bia", "5511990000003")
	chat := b.group("Galera", ana, bia)
	b.setup(chat, "romance consentimento")

	wantReply(t, b.say(chat, ana, "!casar "+mentionTag(bia), bia), mentionTag(ana)+" pediu "+mentionTag(bia)+" em casamento")
	wantReply(t, b.say(chat, bia, "!aceitar"), "acabaram de se casar")
	wantReply(t, b.say(chat, ana, "!casar "+mentionTag(bia), bia), "já são casados")
}

func TestTicTacToe(t *testing.T) {
	b := newBench(t)
	ana, bia := newPerson("Ana", "5511990000002"), newPerson("Bia", "5511990000003")
	chat := b.group("Galera", ana, bia)
	b.setup(chat, "jogos")

	wantReply(t, b.say(chat, ana, "!velha "+mentionTag(bia), bia), "Jogo da velha!")
	wantReply(t, b.say(chat, bia, "1"), "agora é a vez de "+mentionTag(ana))
	b.say(chat, ana, "1")
	wantReply(t, b.say(chat, bia, "1"), "A casa 1 já tá ocupada")
	b.say(chat, bia, "4")
	b.say(chat, ana, "2")
	b.say(chat, bia, "5")
	wantReply(t, b.say(chat, ana, "3"), mentionTag(ana)+" venceu!")
	wantSilence(t, b.say(chat, bia, "6"))
}

func TestGameSurvivesRestart(t *testing.T) {
	b := newBench(t)
	ana, bia := newPerson("Ana", "5511990000002"), newPerson("Bia", "5511990000003")
	chat := b.group("Galera", ana, bia)
	b.setup(chat, "jogos")
	b.say(chat, ana, "!velha "+mentionTag(bia), bia)
	b.say(chat, ana, "5")

	b.restart()

	wantReply(t, b.say(chat, ana, "1"), "agora é a vez de "+mentionTag(bia))
	wantReply(t, b.say(chat, bia, "5"), "A casa 5 já tá ocupada")
	wantReply(t, b.say(chat, bia, "1"), mentionTag(bia)+" jogou na casa 1")
}

func TestThrowNeedsTheItemAndWaits(t *testing.T) {
	b := newBench(t)
	ana, bia := newPerson("Ana", "5511990000002"), newPerson("Bia", "5511990000003")
	chat := b.group("Galera", ana, bia)
	b.setup(chat, "economia")

	wantReply(t, b.say(chat, ana, "!tacarpedra "+mentionTag(bia), bia), "não tem 🪨 *Pedra* na mochila")
	wantReply(t, b.say(chat, ana, "!tacartomate "+mentionTag(bia), bia), "não tem 🍅 *Tomate* na mochila")

	b.say(chat, b.owner, "!profissao deus")
	b.say(chat, b.owner, "!milagre pedra 2 "+mentionTag(ana), ana)
	wantReply(t, b.say(chat, ana, "!tacar "+mentionTag(bia)+" pedra", bia), "tacou uma pedra em @"+bia.jid.User)
	wantReply(t, b.say(chat, ana, "!tacarpedra "+mentionTag(bia), bia), "ainda tá com o braço doendo")
	wantReply(t, b.say(chat, ana, "!mochila"), "Pedra")
}

func TestExtractionRunsEveryActivity(t *testing.T) {
	b := newBench(t)
	ana := newPerson("Ana", "5511990000002")
	chat := b.group("Galera", ana)
	b.setup(chat, "economia")

	first := wantReply(t, b.say(chat, ana, "!extração"), "Pra usar o !pescar")
	for _, fragment := range []string{"Pra usar o !minerar", "Pra usar o !catar", "🗑️"} {
		if !strings.Contains(first, fragment) {
			t.Fatalf("esperava %q na resposta do !extracao, veio %q", fragment, first)
		}
	}
	wantReply(t, b.say(chat, ana, "!extrair"), "acabou de revirar o lixão")
}

func TestPostWithoutNetworkPicksOneTheDeviceReaches(t *testing.T) {
	b := newBench(t)
	ana := newPerson("Ana", "5511990000002")
	chat := b.group("Galera", ana)
	b.setup(chat, "economia")

	wantReply(t, b.say(chat, ana, "!postar"), "precisa de um aparelho na mochila")

	b.say(chat, b.owner, "!profissao deus")
	b.say(chat, b.owner, "!milagre celular 1 "+mentionTag(ana), ana)
	reply := wantReply(t, b.say(chat, ana, "!postar"), "Agora são")
	if !strings.Contains(reply, "no TikTok") && !strings.Contains(reply, "no Instagram") {
		t.Fatalf("com só o Celular o !postar devia cair no TikTok ou no Instagram, respondeu %q", reply)
	}
}

func listingNumber(t *testing.T, reply string) string {
	t.Helper()
	found := regexp.MustCompile(`!feira comprar (\d+)`).FindStringSubmatch(reply)
	if found == nil {
		t.Fatalf("o anúncio não disse o número pra comprar: %q", reply)
	}
	return found[1]
}

func TestFairSellsBetweenPlayers(t *testing.T) {
	b := newBench(t)
	ana, bia := newPerson("Ana", "5511990000002"), newPerson("Bia", "5511990000003")
	chat := b.group("Galera", ana, bia)
	b.setup(chat, "economia")
	b.say(chat, b.owner, "!profissao deus")
	b.say(chat, b.owner, "!milagre pedra 3 "+mentionTag(ana), ana)
	b.say(chat, b.owner, "!milagre 1000 "+mentionTag(bia), bia)
	anaStart, biaStart := b.balance(ana), b.balance(bia)

	wantReply(t, b.say(chat, bia, "!feira"), "A feira tá vazia")
	id := listingNumber(t, wantReply(t, b.say(chat, ana, "!feira vender pedra 2 100"), "botou na feira"))
	wantReply(t, b.say(chat, bia, "!feira"), "*"+id+"* · ⚪ 2x 🪨 *Pedra* · "+economy.Coins(100))
	wantReply(t, b.say(chat, bia, "!feira pedra"), "Pedra")
	wantReply(t, b.say(chat, bia, "!feira todynho"), "Ninguém tá vendendo")
	wantReply(t, b.say(chat, ana, "!feira comprar "+id), "Comprar o próprio anúncio?")

	wantReply(t, b.say(chat, bia, "!feira comprar "+id), "comprou 2x 🪨 *Pedra*")
	if got := b.balance(bia); got != biaStart-100 {
		t.Fatalf("Bia devia pagar 100, o saldo foi de %d pra %d", biaStart, got)
	}
	if got := b.balance(ana); got != anaStart+95 {
		t.Fatalf("Ana devia receber 95 (100 menos a comissão de 5%%), o saldo foi de %d pra %d", anaStart, got)
	}
	if ana, bia := b.held(ana, "pedra"), b.held(bia, "pedra"); ana != 1 || bia != 2 {
		t.Fatalf("depois da venda Ana devia ter 1 pedra e Bia 2, ficaram %d e %d", ana, bia)
	}
	wantReply(t, b.say(chat, bia, "!feira comprar "+id), "não tá mais na feira")

	id = listingNumber(t, wantReply(t, b.say(chat, ana, "!feira vender pedra 50"), "botou na feira"))
	b.say(chat, ana, "!vender pedra")
	wantReply(t, b.say(chat, bia, "!feira comprar "+id), "não tem mais 🪨 *Pedra*, então o anúncio saiu da feira")
	wantReply(t, b.say(chat, bia, "!feira"), "A feira tá vazia")

	id = listingNumber(t, wantReply(t, b.say(chat, bia, "!feira vender pedra 2 80"), "botou na feira"))
	wantReply(t, b.say(chat, ana, "!feira tirar "+id), "Você não tem nenhum anúncio "+id)
	wantReply(t, b.say(chat, bia, "!feira tirar "+id), "Tirou 2x 🪨 *Pedra* da feira")
	wantReply(t, b.say(chat, ana, "!feira"), "A feira tá vazia")
}
