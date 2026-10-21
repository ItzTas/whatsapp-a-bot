package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

const tyrantPeopleShare = 20

func (pd percentDecree) rulerColumn() string {
	return pd.column + "_ditador"
}

func rulerColumns() []string {
	var columns []string
	for _, pd := range percentDecrees {
		columns = append(columns, pd.rulerColumn())
	}
	return columns
}

func rulerReset() string {
	return strings.Join(rulerColumns(), " = 0, ") + " = 0"
}

func (d decrees) ruled() decrees {
	ruled := make(decrees, len(d))
	maps.Copy(ruled, d)
	for _, pd := range percentDecrees {
		ruled[pd.column] += d[pd.rulerColumn()]
	}
	return ruled
}

func (pd percentDecree) harshness() int64 {
	if pd.popular {
		return -1
	}
	return 1
}

func (d decrees) split(pd percentDecree, level int64) (people, extra int64) {
	if !d.on(tyrannyDecree) {
		return level, 0
	}
	h := pd.harshness()
	own, crowd, wanted := d.ruled().level(pd)*h, d.level(pd)*h, level*h
	shift := wanted - own
	if shift < 0 {
		shift = shift * tyrantPeopleShare / 100
	} else {
		shift = (shift*tyrantPeopleShare + 99) / 100
	}
	lo, hi := d.limits(pd)
	people = min(max(max(crowd+shift, wanted)*h, lo), hi)
	return people, level - people
}

func (pd percentDecree) state(level int64) string {
	if level == 0 {
		return pd.revoked
	}
	return pd.effect(level)
}

func (d decrees) percentInForce(pd percentDecree, ruled decrees) string {
	level, own := d.level(pd), ruled.level(pd)
	if level == own {
		if level == 0 {
			return ""
		}
		return pd.effect(level)
	}
	return fmt.Sprintf("%s (pro ditador, %s)", pd.state(level), pd.state(own))
}

func (pd percentDecree) ruledText(user string, own, people int64) string {
	return fmt.Sprintf("🪖 *Decreto do ditador!* @%s decretou que %s, mas isso é só pra ele: na ditadura o povo só sente %d%% de cada alívio do ditador, então pro povo %s",
		user, pd.state(own), tyrantPeopleShare, pd.state(people))
}

func (s *walletStore) decreesOf(ctx context.Context, chat types.JID, acc core.Account, now time.Time) (decrees, error) {
	columns := decreeColumns()
	values := make([]int64, len(columns))
	dest := make([]any, len(values), len(values)+1)
	for i := range values {
		dest[i] = &values[i]
	}
	var ruler bool
	err := s.db.QueryRowContext(ctx,
		"SELECT "+strings.Join(columns, ", ")+", pessoa = ? AND mandato_ate > ? FROM presidencias WHERE chat = ? AND (mandato_ate > ? OR renunciou = 1)",
		acc.Key, now.Unix(), chat.String(), now.Unix(),
	).Scan(append(dest, &ruler)...)
	if errors.Is(err, sql.ErrNoRows) {
		return decrees{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ler decretos: %w", err)
	}
	d := zipDecrees(columns, values)
	if ruler {
		return d.ruled(), nil
	}
	return d, nil
}
