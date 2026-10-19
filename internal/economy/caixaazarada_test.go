package economy

import (
	"context"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/types"
)

func TestBoxJinx(t *testing.T) {
	ctx := context.Background()
	chat := types.NewJID("120363000000000001", types.GroupServer)
	s, people := jackpotWallets(t, ctx, chat, "5511900000001", "5511900000002")
	gambler, loser := people[0], people[1]
	now := time.Unix(time.Now().Unix(), 0)

	steps := []struct {
		acc  int
		net  int64
		at   time.Duration
		want [2]int
	}{
		{0, boxJinxMin * 3, -5 * time.Hour, [2]int{0, 0}},
		{0, boxJinxMin / 2, -3 * time.Hour, [2]int{0, 0}},
		{0, boxJinxMin / 2, -time.Hour, [2]int{boxJinxDraws, 0}},
		{1, boxJinxMin * 2, -2 * time.Hour, [2]int{boxJinxDraws, boxJinxDraws}},
		{1, -boxJinxMin * 2, -time.Minute, [2]int{boxJinxDraws, 0}},
	}
	for i, st := range steps {
		if err := s.noteRecentWinnings(ctx, chat, people[st.acc], st.net, now.Add(st.at)); err != nil {
			t.Fatal(err)
		}
		for j, acc := range []int{0, 1} {
			got, err := s.boxJinx(ctx, chat, people[acc], now)
			if err != nil || got != st.want[j] {
				t.Fatalf("passo %d, pessoa %d: azar = %d, %v; queria %d", i, acc, got, err, st.want[j])
			}
		}
	}

	if got, err := s.boxJinx(ctx, chat, gambler, now.Add(3*time.Hour)); err != nil || got != 0 {
		t.Fatalf("azar depois da janela = %d, %v; queria 0", got, err)
	}
	if got, err := s.boxJinx(ctx, chat, loser, now); err != nil || got != 0 {
		t.Fatalf("quem perdeu o que ganhou ficou com azar %d, %v", got, err)
	}
}

func TestDrawnKeepsWorstWhenJinxed(t *testing.T) {
	sequence := func() func() int {
		values, i := []int{2, 4, 1}, 0
		return func() int { v := values[i%len(values)]; i++; return v }
	}
	cases := []struct {
		luck int
		want int
	}{
		{0, 2},
		{1, 4},
		{2, 4},
		{-1, 2},
		{-2, 1},
	}
	for _, c := range cases {
		if got := drawn(c.luck, sequence()); got != c.want {
			t.Errorf("drawn(%d) = %d, queria %d", c.luck, got, c.want)
		}
	}
	if !jinxedHit(1, -3) || jinxedHit(0, 0) {
		t.Error("jinxedHit errou os extremos")
	}
}
