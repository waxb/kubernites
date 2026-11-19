package view

const MaxReachTable = 24

var reachAreas = func() [MaxReachTable + 1]int {
	var out [MaxReachTable + 1]int
	for r := range out {
		out[r] = CountReach(r)
	}
	return out
}()

func CountReach(radius int) int {
	n := 0
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if WithinReach(dx, dy, radius) {
				n++
			}
		}
	}
	return n
}

func ReachArea(radius int) int {
	if radius >= 0 && radius <= MaxReachTable {
		return reachAreas[radius]
	}
	return CountReach(radius)
}

func LandYield(kt KeepType) float64 {
	if kt.Claim <= 0 || kt.Territory <= 0 || kt.Claim == kt.Territory {
		return kt.Yield
	}
	return kt.Yield * float64(ReachArea(kt.Claim)) / float64(ReachArea(kt.Territory))
}
