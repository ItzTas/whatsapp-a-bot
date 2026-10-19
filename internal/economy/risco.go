package economy

import "math"

const (
	stakeRiskFloor = 1_000
	stakeRiskStep  = 0.025
	stakeRiskMax   = 0.1
)

func stakeRisk(stake int64) float64 {
	if stake <= stakeRiskFloor {
		return 0
	}
	return min(stakeRiskStep*math.Log10(float64(stake)/stakeRiskFloor), stakeRiskMax)
}

func (pb PrizeBoost) Risk(stake int64) float64 {
	return stakeRisk(stake) + float64(min(pb.heat, streakHeatMax))/100
}
