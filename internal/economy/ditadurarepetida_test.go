package economy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

type termStep struct {
	president int
	coup      bool
	tyranny   bool
	resign    bool
}

func TestTyrannyNeedsAPreviousTermWithoutIt(t *testing.T) {
	cases := map[string]struct {
		history []termStep
		next    termStep
		barred  bool
	}{
		"ditador reeleito logo depois": {
			history: []termStep{{president: 0, tyranny: true}},
			next:    termStep{president: 0},
			barred:  true,
		},
		"ditador que voltou depois de outro presidente": {
			history: []termStep{{president: 0, tyranny: true}, {president: 1}},
			next:    termStep{president: 0},
			barred:  true,
		},
		"ditador que renunciou e voltou": {
			history: []termStep{{president: 0, tyranny: true, resign: true}},
			next:    termStep{president: 0},
			barred:  true,
		},
		"ditador que voltou no golpe": {
			history: []termStep{{president: 0, tyranny: true}, {president: 1}},
			next:    termStep{president: 0, coup: true},
			barred:  true,
		},
		"ditador depois de um mandato sem ditadura": {
			history: []termStep{{president: 0, tyranny: true}, {president: 0}},
			next:    termStep{president: 0},
		},
		"outro presidente depois do ditador": {
			history: []termStep{{president: 0, tyranny: true}},
			next:    termStep{president: 1},
		},
		"presidente sem ditadura antes": {
			history: []termStep{{president: 0}},
			next:    termStep{president: 0},
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			chat := types.NewJID("120363000000000001", types.GroupServer)
			s, people := jackpotWallets(t, ctx, chat, "5511900000001", "5511900000002")
			now := time.Unix(1_800_000_000, 0)
			for _, step := range c.history {
				now = startTerm(t, ctx, s, chat, people, step, now)
				if !step.tyranny {
					continue
				}
				if declared, err := s.declareTyranny(ctx, chat, people[step.president], now); err != nil || !declared {
					t.Fatalf("ditadura do histórico não saiu: declared %v, err %v", declared, err)
				}
				if !step.resign {
					continue
				}
				if _, err := s.resign(ctx, chat, people[step.president], now.Add(time.Minute)); err != nil {
					t.Fatal(err)
				}
			}
			now = startTerm(t, ctx, s, chat, people, c.next, now)
			declared, err := s.declareTyranny(ctx, chat, people[c.next.president], now)
			if barred := errors.Is(err, errTyrannyRepeat); barred != c.barred {
				t.Fatalf("ditadura barrada = %v, want %v (err %v)", barred, c.barred, err)
			}
			if !c.barred && (err != nil || !declared) {
				t.Fatalf("ditadura não saiu: declared %v, err %v", declared, err)
			}
		})
	}
}

func startTerm(t *testing.T, ctx context.Context, s *walletStore, chat types.JID, people []core.Account, step termStep, now time.Time) time.Time {
	t.Helper()
	if !step.coup {
		now = now.Add(termLength + time.Minute)
		if err := s.inaugurate(ctx, chat, people[step.president], 3, now); err != nil {
			t.Fatal(err)
		}
		return now
	}
	now = now.Add(time.Hour)
	if err := s.inaugurate(ctx, chat, people[1-step.president], 3, now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	err := s.inTx(ctx, func(tx *ledgerTx) error {
		return seize(ctx, tx, chat, people[1-step.president], people[step.president], now)
	})
	if err != nil {
		t.Fatal(err)
	}
	return now
}
