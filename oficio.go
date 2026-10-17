package main

import (
	"fmt"
	"slices"
)

const (
	areaFindChance = 0.08
	tradePayBonus  = 15
	tradeFindBonus = 0.07
)

type trade struct {
	area int
}

func tradeTool(a int) (item, bool) {
	i := slices.IndexFunc(catalog, func(it item) bool { return it.trade != nil && it.trade.area == a })
	if i < 0 {
		return item{}, false
	}
	return catalog[i], true
}

func (p profession) loot() ([]string, float64) {
	if p.fixedFinds {
		return p.finds, p.findChance
	}
	if len(p.finds) > 0 {
		return p.finds, p.findChance + p.findBonus
	}
	return areas[p.area].finds, areaFindChance + p.findBonus
}

func (p profession) tooled(owned []item) profession {
	for _, it := range owned {
		if !p.suits(it) {
			continue
		}
		p.min, p.max = p.min*(100+tradePayBonus)/100, p.max*(100+tradePayBonus)/100
		p.findBonus += tradeFindBonus
		p.equipped = true
	}
	return p
}

func (p profession) suits(it item) bool {
	return it.trade != nil && it.trade.area == p.area && !p.divine
}

func tradePerks(p profession) []string {
	tool, ok := tradeTool(p.area)
	if !ok || p.divine {
		return nil
	}
	if p.fixedFinds && p.equipped {
		return []string{fmt.Sprintf("🛠️ Trabalha com %s: o !trabalhar paga %d%% a mais (já conta acima)", tool.label(), tradePayBonus)}
	}
	if p.fixedFinds {
		return []string{fmt.Sprintf("🛠️ Com %s na mochila (tem na !loja), o !trabalhar paga %d%% a mais", tool.label(), tradePayBonus)}
	}
	if p.equipped {
		return []string{fmt.Sprintf("🛠️ Trabalha com %s: o !trabalhar paga %d%% a mais e a chance de achar item sobe %d pontos (já conta acima)", tool.label(), tradePayBonus, percent(tradeFindBonus))}
	}
	return []string{fmt.Sprintf("🛠️ Com %s na mochila (tem na !loja), o !trabalhar paga %d%% a mais e a chance de achar item sobe %d pontos", tool.label(), tradePayBonus, percent(tradeFindBonus))}
}

func tradePerk(t *trade) string {
	a := areas[t.area]
	return fmt.Sprintf("vale só de ter na mochila: quem trabalha em %s %s ganha %d%% a mais no !trabalhar e tem +%d%% de chance de achar item no trampo", a.emoji, a.name, tradePayBonus, percent(tradeFindBonus))
}

func foundAtWork(it item) bool {
	return slices.ContainsFunc(areas[:], func(a area) bool { return slices.Contains(a.finds, it.id) }) ||
		slices.ContainsFunc(professions, func(p profession) bool { return slices.Contains(p.finds, it.id) })
}

func workSource(it item) string {
	var who []string
	for _, a := range areas {
		if slices.Contains(a.finds, it.id) {
			who = append(who, "quem é de "+a.name)
		}
	}
	for _, p := range professions {
		if slices.Contains(p.finds, it.id) {
			who = append(who, p.name)
		}
	}
	if len(who) == 0 {
		return ""
	}
	if len(who) > 6 {
		who = append(who[:5], fmt.Sprintf("mais %d profissões", len(who)-5))
	}
	return "!trabalhar (" + joinWithAnd(who) + ")"
}
