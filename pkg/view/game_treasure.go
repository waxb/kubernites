package view

import (
	"fmt"
)

type TreasureView struct {
	Kind  string `json:"kind"`
	Boxes string `json:"boxes"`
	Crew  string `json:"crew"`
}

func TreasureLine(chest float64) string {
	return fmt.Sprintf("The first ship's treasure chest opens: +%s gold", Compact(chest))
}
