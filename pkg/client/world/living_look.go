package world

import (
	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	IvyMossAfter    = 360.0
	IvyFullAfter    = 2520.0
	IvySteps        = 7
	CairnStageSpan  = 70.0
	RubbleStageSpan = 110.0
	LivestockHuts   = 5
	LaundryMinCrew  = 2
	hungryShare     = .85
	hungryFreeNeedy = .3
	wheatBoost      = 1.3
	wheatFloor      = .62
	maxLanterns     = 8
	pennantsPerKeep = 6
	berryRich       = 6.0
	berryPickRate   = .35
	berryRegrowRate = .09
	wornFull        = 16.0
	wornFloor       = 3.0
	boatMinTier     = 1
	maxQuayGulls    = 4
)

func clamp01(v float64) float64 { return min(max(v, 0), 1) }

func FoodPressure(v view.KeepView) float64 {
	if v.Allocatable.CPU <= 0 {
		return .5
	}
	alloc := float64(v.Allocatable.CPU)
	return clamp01(max(float64(v.Requested.CPU), float64(v.Used.CPU)) / alloc)
}

func FoodFree(v view.KeepView) float64 { return 1 - FoodPressure(v) }

func WheatLevel(free, middle float64) int {
	score := clamp01(free*wheatBoost) * (wheatFloor + (1-wheatFloor)*clamp01(middle))
	return min(int(score*art.WheatLevels), art.WheatLevels-1)
}

func Hungry(v view.KeepView, throttled int, stalled bool) bool {
	if v.Allocatable.CPU <= 0 {
		return false
	}
	return FoodPressure(v) >= hungryShare || throttled > 0 || stalled && FoodFree(v) < hungryFreeNeedy
}

func BerryTarget(hungry bool, free float64) float64 {
	if hungry {
		return 0
	}
	return min(clamp01(free)*berryRich, art.BushFruits-1)
}

func BerryFruit(level float64) int {
	return min(max(int(level+.5), 0), art.BushFruits-1)
}

func EaseBerries(level, target, dt float64) float64 {
	switch {
	case level > target:
		return max(level-berryPickRate*dt, target)
	case level < target:
		return min(level+berryRegrowRate*dt, target)
	}
	return level
}

func IvyLevel(quiet float64) int {
	if quiet < IvyMossAfter {
		return 0
	}
	span := clamp01((quiet - IvyMossAfter) / (IvyFullAfter - IvyMossAfter))
	return 1 + min(int(span*float64(IvySteps-1)+.5), IvySteps-1)
}

func CairnStage(age float64) int {
	return min(max(int(age/CairnStageSpan), 0), art.CairnStages-1)
}

func RubbleStage(age float64) int {
	return min(max(int(age/RubbleStageSpan), 0), art.RubbleStages-1)
}

func HasBinding(c model.CrewChain) bool { return c.Resources.MemLimit > 0 }

func LaundryHangs(c model.CrewChain) bool {
	return !c.Bare && c.Desired >= LaundryMinCrew && c.Ready == c.Desired && !c.Rolling()
}

func LanternsLit(desired, ready int) (total, lit int) {
	total = min(max(desired, ready), maxLanterns)
	return total, min(max(ready, 0), total)
}

func HoldsLivestock(huts int) bool { return huts >= LivestockHuts }

func BoatsLaunch(h view.Haven) bool { return h.Open && int(h.Tier) >= boatMinTier }

func BoardParchments(crews, depots int) int { return max(crews, 0) + max(depots, 0) }

func WornLevel(seconds float64) float64 {
	if seconds <= wornFloor {
		return 0
	}
	return clamp01((seconds - wornFloor) / (wornFull - wornFloor))
}

func ScaffoldedPods(crews []model.CrewChain) map[string]bool {
	out := map[string]bool{}
	for _, c := range crews {
		if !c.Rolling() {
			continue
		}
		for _, f := range c.Foremen {
			if f.Newest {
				continue
			}
			for _, name := range f.Pods {
				out[name] = true
			}
		}
	}
	return out
}
