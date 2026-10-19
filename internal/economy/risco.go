package economy

import "math"

const (
	stakeRiskFloor = 1_000
	stakeRiskStep  = 0.005
	stakeRiskMax   = 0.02
)

func StakeRisk(stake int64) float64 {
	if stake <= stakeRiskFloor {
		return 0
	}
	return min(stakeRiskStep*math.Log10(float64(stake)/stakeRiskFloor), stakeRiskMax)
}
