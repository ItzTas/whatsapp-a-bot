package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

const backlogSpan = time.Hour

func backlog(idle, every time.Duration) int {
	if every <= 0 || idle < every {
		return 1
	}
	return int(min(idle/every, (backlogSpan+every-1)/every))
}

func stampedAt(ctx context.Context, q rowQuerier, chat types.JID, acc core.Account, column string) (time.Time, error) {
	var last int64
	err := q.QueryRowContext(ctx, "SELECT "+column+" FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&last)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Unix(0, 0), nil
	}
	return time.Unix(last, 0), err
}

func (p Profession) toil(now time.Time, turns int) shift {
	s := p.payFor(p.work(now))
	for range turns - 1 {
		s = s.merge(p.payFor(p.work(now)))
	}
	return s
}

func (s shift) merge(next shift) shift {
	if s.busted && !next.busted {
		s.task, s.peaked = next.task, next.peaked
	}
	s.amount += next.amount
	s.busted = s.busted && next.busted
	s.finds = append(s.finds, next.finds...)
	s.turns += next.turns
	s.busts += next.busts
	s.doubles += next.doubles
	return s
}

func (s shift) backlogNote() string {
	if s.turns <= 1 {
		return ""
	}
	note := fmt.Sprintf("⏱️ Trabalhou %d turnos acumulados de uma vez", s.turns)
	if s.busts > 0 {
		note += fmt.Sprintf(", e %d %s ruim", s.busts, pluralWord(int64(s.busts), "deu", "deram"))
	}
	return note
}

func (s shift) luckNote() string {
	if s.turns <= 1 {
		return "🍀 Dia de sorte: pagaram em dobro!"
	}
	return fmt.Sprintf("🍀 Dia de sorte: %d %s em dobro", s.doubles, pluralWord(int64(s.doubles), "turno pagou", "turnos pagaram"))
}

func backlogReport(extra []haul) string {
	if len(extra) == 0 {
		return ""
	}
	lines := []string{fmt.Sprintf("⏱️ Desde a última vez deu tempo de mais %d %s:", len(extra), pluralWord(int64(len(extra)), "tentativa", "tentativas"))}
	var (
		order  []string
		counts = map[string]int{}
		empty  int
	)
	for _, h := range extra {
		if h.found.id == "" {
			empty++
			continue
		}
		label := h.found.label() + h.grade.tag()
		if counts[label] == 0 {
			order = append(order, label)
		}
		counts[label]++
	}
	for _, label := range order {
		lines = append(lines, fmt.Sprintf("• %dx %s", counts[label], label))
	}
	if empty > 0 {
		lines = append(lines, fmt.Sprintf("• %d de mão vazia", empty))
	}
	return "\n\n" + strings.Join(lines, "\n")
}

func withdrawn(bag []stack, id string) []stack {
	left := make([]stack, 0, len(bag))
	for _, s := range bag {
		if s.item.id == id {
			s.quantity--
		}
		if s.quantity > 0 {
			left = append(left, s)
		}
	}
	return left
}

func backlogHelp() string {
	return fmt.Sprintf(". Ficou mais tempo sem vir? Cada espera que passa guarda uma tentativa (até juntar %s de espera), e todas saem de uma vez na próxima", core.FormatWait(backlogSpan))
}
