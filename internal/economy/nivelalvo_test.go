package economy

import (
	"testing"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

func TestMarkLevelAfterStake(t *testing.T) {
	cases := []struct {
		name  string
		from  markLevel
		stake int64
		want  markLevel
	}{
		{"aposta menor que o prêmio só soma", markLevel{jackpot: 10_000, level: 1}, 4_000, markLevel{jackpot: 10_000, level: 1, wagered: 4_000}},
		{"somar o tamanho do prêmio sobe um nível", markLevel{jackpot: 10_000, level: 1, wagered: 4_000}, 7_000, markLevel{jackpot: 10_000, level: 2, wagered: 1_000}},
		{"aposta de várias vezes o prêmio sobe vários", markLevel{jackpot: 10_000, level: 1}, 35_000, markLevel{jackpot: 10_000, level: 4, wagered: 5_000}},
		{"o nível para no máximo", markLevel{jackpot: 10_000, level: markLevelMax - 1}, 90_000, markLevel{jackpot: 10_000, level: markLevelMax, wagered: 0}},
		{"no máximo nada muda", markLevel{jackpot: 10_000, level: markLevelMax, wagered: 3_000}, 50_000, markLevel{jackpot: 10_000, level: markLevelMax, wagered: 3_000}},
		{"sem marca nada muda", markLevel{}, 50_000, markLevel{}},
	}
	for _, tc := range cases {
		if got := tc.from.after(tc.stake); got != tc.want {
			t.Errorf("%s: queria %+v, deu %+v", tc.name, tc.want, got)
		}
	}
}

func TestMarkLevelOdds(t *testing.T) {
	now := time.Now()
	escolta := mustFirm(t, "escolta-armada")
	for level := 1; level <= markLevelMax; level++ {
		m := easyMark{jackpot: 5_000, level: level}
		steps := float64(level - 1)
		if got, want := m.odds(robSuccessChance), clampChance(robSuccessChance+jackpotRobBonus+levelRobStep*steps); got != want {
			t.Errorf("nível %d: o roubo devia ter %.2f de chance, deu %.2f", level, want, got)
		}
		if got, want := (contract{firm: escolta}).odds(m, now), escolta.annul*(1-levelEscortCut*steps); got < want-1e-9 || got > want+1e-9 {
			t.Errorf("nível %d: a escolta devia ter %.2f de chance, deu %.2f", level, want, got)
		}
	}
}

func TestMarkLevelLifecycle(t *testing.T) {
	fb := newFirmBench(t)
	acc := fb.person("5511900000001")
	fb.setBalance(t, acc, 100_000)

	if _, err := fb.s.noteWinnings(fb.ctx, fb.chat, acc, 10_000, fb.now); err != nil {
		t.Fatal(err)
	}
	m, err := markOf(fb.ctx, fb.s.db, fb.chat, acc, fb.now)
	if err != nil || m.jackpot != 10_000 || m.level != 1 {
		t.Fatalf("quem vira alvo fácil começa no nível 1, deu %+v (%v)", m, err)
	}

	from, to, err := fb.s.raiseMark(fb.ctx, fb.chat, acc, 25_000, fb.now)
	if err != nil || from != 1 || to != 3 {
		t.Fatalf("apostar duas vezes e meia o prêmio sobe dois níveis, deu %d→%d (%v)", from, to, err)
	}
	if m, err = markOf(fb.ctx, fb.s.db, fb.chat, acc, fb.now); err != nil || m.level != 3 {
		t.Fatalf("a marca lida no roubo mostra o nível novo, deu %+v (%v)", m, err)
	}

	later := fb.now.Add(jackpotWindow + time.Minute)
	if from, to, err := fb.s.raiseMark(fb.ctx, fb.chat, acc, 50_000, later); err != nil || from != 0 || to != 0 {
		t.Fatalf("depois que a marca vence apostar não sobe nível, deu %d→%d (%v)", from, to, err)
	}
	fb.exec(t, "DELETE FROM apostas_lucro WHERE chat = ? AND pessoa = ?", fb.chat.String(), acc.Key)
	if _, err := fb.s.noteWinnings(fb.ctx, fb.chat, acc, 10_000, later); err != nil {
		t.Fatal(err)
	}
	if m, err = markOf(fb.ctx, fb.s.db, fb.chat, acc, later); err != nil || m.jackpot == 0 || m.level != 1 {
		t.Fatalf("uma marca nova começa de novo no nível 1, deu %+v (%v)", m, err)
	}
}

func TestStalkWaitShrinksWithLevel(t *testing.T) {
	var w windfallBoard
	now := time.Now()
	key := eventKey(types.NewJID("120363000000000001", types.GroupServer), core.Account{Key: "5511900000001@s.whatsapp.net"})
	if !w.stalk(key, now, stalkEvery) {
		t.Fatal("o primeiro evento do alvo fácil sai")
	}
	every := time.Duration(float64(stalkEvery) / markLevelMax)
	if w.stalk(key, now.Add(every-time.Second), every) {
		t.Fatal("antes da espera do nível acabar não sai outro")
	}
	if !w.stalk(key, now.Add(every), every) {
		t.Fatal("no nível máximo a espera entre eventos é dividida pelo nível")
	}
}
