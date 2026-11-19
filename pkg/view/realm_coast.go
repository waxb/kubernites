package view

import (
	"math"
	"math/rand/v2"
)

const CoastCeiling = 1.18

const coastWaves = 5

const coastFirst = 2

const coastMean = 0.97

const coastSwing = 0.24

const bayWidth = 0.22

const bayDeep = 0.42

const minBays = 2

const maxBays = 3

const HomeHarbor = math.Pi

const harborClear = 0.7

const baySpacing = 0.9

const reachFloor = 0.45

const bayPlacements = 24

type Coast struct {
	amp, phase []float64
	bays       []float64
	round      bool
}

var roundCoast = Coast{round: true}

func NewCoast(rng *rand.Rand, home bool) Coast {
	c := Coast{amp: make([]float64, coastWaves), phase: make([]float64, coastWaves)}
	sum := 0.0
	for k := range c.amp {
		c.amp[k] = (0.4 + 0.6*rng.Float64()) / math.Pow(float64(k+coastFirst), 0.7)
		c.phase[k] = 2 * math.Pi * rng.Float64()
		sum += c.amp[k]
	}
	for k := range c.amp {
		c.amp[k] /= sum
	}
	n := minBays + rng.IntN(maxBays-minBays+1)
	for try := 0; try < bayPlacements && len(c.bays) < n; try++ {
		a := 2 * math.Pi * rng.Float64()
		if home && arcGap(a, HomeHarbor) < harborClear {
			continue
		}
		apart := true
		for _, b := range c.bays {
			apart = apart && arcGap(a, b) >= baySpacing
		}
		if apart {
			c.bays = append(c.bays, a)
		}
	}
	return c
}

func arcGap(a, b float64) float64 {
	d := math.Mod(math.Abs(a-b), 2*math.Pi)
	return min(d, 2*math.Pi-d)
}

func (c Coast) Reach(theta float64) float64 {
	if c.round {
		return 1
	}
	wave := 0.0
	for k, a := range c.amp {
		wave += a * math.Cos(float64(k+coastFirst)*theta+c.phase[k])
	}
	reach := coastMean + coastSwing*wave
	for _, b := range c.bays {
		g := arcGap(theta, b) / bayWidth
		reach -= bayDeep * math.Exp(-g*g)
	}
	return min(max(reach, reachFloor), CoastCeiling)
}
