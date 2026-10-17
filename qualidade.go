package main

import (
	"fmt"
	"slices"
	"strings"
)

type quality int

const normalQuality quality = 0

type qualityTier struct {
	level   quality
	name    string
	badge   string
	percent int64
	weight  int
}

var qualities = []qualityTier{
	{-3, "horrível", "💀", 10, 10},
	{-2, "podre", "⭐", 40, 95},
	{-1, "ruim", "⭐⭐", 70, 205},
	{normalQuality, "normal", "⭐⭐⭐", 100, 385},
	{1, "boa", "⭐⭐⭐⭐", 130, 200},
	{2, "excelente", "⭐⭐⭐⭐⭐", 160, 100},
	{3, "lendária", "🌟", 250, 5},
}

type grade struct {
	quality  quality
	quantity int64
}

func (q quality) index() int {
	if i := slices.IndexFunc(qualities, func(t qualityTier) bool { return t.level == q }); i >= 0 {
		return i
	}
	return slices.IndexFunc(qualities, func(t qualityTier) bool { return t.level == normalQuality })
}

func (q quality) tier() qualityTier {
	return qualities[q.index()]
}

func (q quality) tag() string {
	if q == normalQuality {
		return ""
	}
	t := q.tier()
	return fmt.Sprintf(" %s _%s_", t.badge, t.name)
}

func (q quality) price(value int64) int64 {
	return value * q.tier().percent / 100
}

func (q quality) summary(it item) string {
	if q == normalQuality {
		return it.summary()
	}
	r := rarities[it.rarity]
	t := q.tier()
	return fmt.Sprintf("%s %s · %s %s (%d%% do valor) · vale %s", r.emoji, r.name, t.badge, t.name, t.percent, coins(q.price(it.value)))
}

func rollQuality(luck int, rotten bool) quality {
	if rotten {
		return worstQuality()
	}
	pick := func() quality {
		return qualities[weightedIndex(len(qualities), func(i int) int { return qualities[i].weight })].level
	}
	keep := func(a, b quality) quality { return max(a, b) }
	if luck < 0 {
		keep = func(a, b quality) quality { return min(a, b) }
	}
	q := pick()
	for range max(luck, -luck) {
		q = keep(q, pick())
	}
	return q
}

func worstQuality() quality {
	return qualities[0].level
}

func (it item) gradable() bool {
	return it.gear == nil && it.prizes == nil && it.loot == nil && it.relicOdds == 0 && it.rarity != relic &&
		!it.luxury && !it.illegal && !it.pet && it.arms == nil && it.high == nil && it.flex == nil &&
		it.bless == "" && it.hex == "" && it.trade == nil && it.resets == nil && it.heist == 0 && it.heal == 0
}

func worthOf(it item, grades []grade) int64 {
	var worth int64
	for _, g := range grades {
		worth += g.quantity * g.quality.price(it.value)
	}
	return worth
}

func marketPayoutOf(it item, grades []grade, percent int64) int64 {
	percent = marketPercentOf(it, percent)
	var payout int64
	for _, g := range grades {
		payout += marketPayoutAt(g.quality.price(it.value), g.quantity, percent)
	}
	return payout
}

func gradesLabel(grades []grade) string {
	if !slices.ContainsFunc(grades, func(g grade) bool { return g.quality != normalQuality }) {
		return ""
	}
	parts := make([]string, len(grades))
	for i, g := range grades {
		parts[i] = fmt.Sprintf("%d %s", g.quantity, g.quality.tier().name)
	}
	return " (" + joinWithAnd(parts) + ")"
}

func qualityGuide() string {
	parts := make([]string, len(qualities))
	for i, t := range qualities {
		parts[i] = fmt.Sprintf("%s %s %d%%", t.badge, t.name, t.percent)
	}
	return strings.Join(parts, " · ")
}
