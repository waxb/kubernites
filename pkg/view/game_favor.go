package view

import (
	"math"
)

type CrownState struct {
	Favor   float64       `json:"favor"`
	Next    float64       `json:"next,omitempty"`
	Seq     int           `json:"seq,omitempty"`
	Decrees []DecreeState `json:"decrees,omitempty"`
}

const FavorStart = 50.0

const FavorMax = 100.0

const favorFloor = 0.9

const favorCeiling = 1.15

func FavorMultiplier(favor float64) float64 {
	f := math.Max(0, math.Min(FavorMax, favor))
	if f < FavorStart {
		return favorFloor + (1-favorFloor)*f/FavorStart
	}
	return 1 + (favorCeiling-1)*(f-FavorStart)/(FavorMax-FavorStart)
}
