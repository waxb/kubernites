package view

import (
	"math"
)

func RoomFor(free, need ResourcePair) int {
	fits := math.MaxInt32
	if need.CPU > 0 {
		fits = min(fits, int(free.CPU/need.CPU))
	}
	if need.Memory > 0 {
		fits = min(fits, int(free.Memory/need.Memory))
	}
	return max(0, fits)
}
