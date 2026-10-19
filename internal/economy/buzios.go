package economy

import (
	"fmt"
	"math/rand/v2"
)

const (
	buziosItem        = "buzios-encantados"
	oracleEventFactor = 3
)

func (p Profession) catchesEvent() bool {
	return p.oracle || rand.Float64() < 1.0/oracleEventFactor
}

func (ch choice) weights(job Profession) []int {
	weights := make([]int, len(ch.endings))
	for i, e := range ch.endings {
		weights[i] = max(e.weight+e.lean(job), 0)
	}
	return weights
}

func (ch choice) odds(job Profession) (good, bad int) {
	if len(ch.endings) == 0 {
		return 0, 0
	}
	weights := ch.weights(job)
	var total, fair, foul int
	for i, w := range weights {
		total += w
		fair += w * max(ch.endings[i].omen(), 0)
		foul += w * max(-ch.endings[i].omen(), 0)
	}
	if total == 0 {
		omen := ch.endings[0].omen()
		return 100 * max(omen, 0), 100 * max(-omen, 0)
	}
	return percent(float64(fair) / float64(total)), percent(float64(foul) / float64(total))
}

func (ch choice) reading(job Profession) string {
	if !job.oracle {
		return ""
	}
	good, bad := ch.odds(job)
	return fmt.Sprintf(" 🐚 %d%% bom, %d%% ruim", good, bad)
}

func (e ending) omen() int {
	var sum int
	for _, ef := range e.effects {
		sum += ef.fortune()
	}
	return min(max(sum, -1), 1)
}

func (earnEffect) fortune() int     { return 1 }
func (gainEffect) fortune() int     { return 1 }
func (lootEffect) fortune() int     { return 1 }
func (healEffect) fortune() int     { return 1 }
func (refreshEffect) fortune() int  { return 1 }
func (buzzEffect) fortune() int     { return 0 }
func (plantEffect) fortune() int    { return 0 }
func (loseEffect) fortune() int     { return -1 }
func (skimEffect) fortune() int     { return -1 }
func (forfeitEffect) fortune() int  { return -1 }
func (injureEffect) fortune() int   { return -1 }
func (stainEffect) fortune() int    { return -1 }
func (delayEffect) fortune() int    { return -1 }
func (refundEffect) fortune() int   { return -1 }
func (fansEffect) fortune() int     { return 1 }
func (unfollowEffect) fortune() int { return -1 }
func (grudgeEffect) fortune() int   { return -1 }
func (breakEffect) fortune() int    { return -2 }
func (nabEffect) fortune() int      { return -2 }
func (perishEffect) fortune() int   { return -3 }

func (e spellEffect) fortune() int {
	c, ok := charmByID(e.id)
	if !ok {
		return 0
	}
	if c.curse {
		return -1
	}
	return 1
}
