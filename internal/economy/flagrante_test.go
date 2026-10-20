package economy

import "testing"

func TestBotchedCrimeCaughtOdds(t *testing.T) {
	var plain Profession
	report := plain.reportOdds()
	botched := map[string]float64{
		"!roubar e !quadrilha de quem tá liso": plain.flagrantOdds(brokeCaughtChance),
		"!sequestrar":                          plain.flagrantOdds(plain.caughtOdds()),
		"!matar":                               plain.flagrantOdds(plain.killCaughtOdds()),
		"!fugir @pessoa":                       plain.flagrantOdds(rescueCaught),
	}
	for crime, odds := range botched {
		if odds <= report || odds >= 1 {
			t.Errorf("%s que dá errado prende com chance entre a do !denunciar (%v) e 1, deu %v", crime, report, odds)
		}
	}
}
