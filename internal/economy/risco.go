package economy

import "math"

const (
	stakeRiskFloor = 500
	stakeRiskStep  = 0.035
	stakeRiskMax   = 0.15
)

func stakeRisk(stake int64) float64 {
	if stake <= stakeRiskFloor {
		return 0
	}
	return min(stakeRiskStep*math.Log10(float64(stake)/stakeRiskFloor), stakeRiskMax)
}

func (pb PrizeBoost) edge(stake int64) float64 {
	return stakeRisk(stake) + float64(min(pb.heat, streakHeatMax)-min(pb.solace, solaceMax))/100
}

func (pb PrizeBoost) Risk(stake int64) float64 {
	return max(pb.edge(stake), 0)
}

func (pb PrizeBoost) Favor(stake int64) float64 {
	return max(-pb.edge(stake), 0)
}
