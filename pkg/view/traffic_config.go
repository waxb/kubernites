package view

import (
	"math"
)

const StartPhase = 0.27

type Config struct {
	BaseArrival   float64
	DayPeriod     float64
	QueueCap      float64
	WindowSeconds int
	EMATau        float64
	Namespace     string
}

func DefaultConfig() Config {
	return Config{
		BaseArrival:   0.45,
		DayPeriod:     720,
		QueueCap:      150,
		WindowSeconds: 300,
		EMATau:        5,
		Namespace:     "default",
	}
}

func DayFactor(phase float64) float64 {
	return 0.35 + 0.65*(0.5-0.5*math.Cos(2*math.Pi*phase))
}
