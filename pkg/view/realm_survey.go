package view

import (
	"errors"
	"fmt"
	"math"

	"k8s.io/apimachinery/pkg/api/resource"
)

const ClassBalanced = "balanced"

const ClassCompute = "compute"

const ClassMemory = "memory"

const ClassRich = "rich"

const treesPerHand = 3.0

const gibPerStone = 1.0

const crewCPUMilli = 1000

const crewMemoryBytes = 1 << 30

const Gib = 1 << 30

const computeBelow = 1.5

const memoryAbove = 3.0

const richOver = 1.5

const yieldEpsilon = 1e-9

type Survey struct {
	Kind      string `json:"kind"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Radius    int    `json:"radius"`
	Wood      int    `json:"wood"`
	Stone     int    `json:"stone"`
	CPU       int    `json:"cpu"`
	MemoryGiB int    `json:"memoryGiB"`
	Class     string `json:"class,omitempty"`
	Instance  string `json:"instance,omitempty"`
	Keep      string `json:"keep,omitempty"`
	OK        bool   `json:"ok"`
	Reason    string `json:"reason,omitempty"`
}

type SurveyKey struct {
	Kind string `json:"-"`
	X, Y int    `json:"-"`
}

type Reserve struct {
	CpuMilli int64 `json:"-"`
	MemBytes int64 `json:"-"`
}

var ReeveShare = Reserve{CpuMilli: 100, MemBytes: 512 << 20}

func HandsFrom(wood int, kt KeepType) int {
	return min(kt.MaxCPU, int(math.Floor(float64(wood)*LandYield(kt)/treesPerHand+yieldEpsilon)))
}

func GibFrom(stone int, kt KeepType) int {
	return min(kt.MaxMemoryGiB, int(math.Floor(float64(stone)*LandYield(kt)*gibPerStone+yieldEpsilon)))
}

func Nominal(kt KeepType) (float64, float64) {
	cpu, err := resource.ParseQuantity(kt.CPU)
	if err != nil {
		return 0, 0
	}
	memory, err := resource.ParseQuantity(kt.Memory)
	if err != nil {
		return 0, 0
	}
	return float64(cpu.MilliValue()) / 1000, float64(memory.Value()) / Gib
}

func ClassOf(cpu, memory int, kt KeepType) string {
	nc, nm := Nominal(kt)
	switch {
	case cpu == 0 && memory == 0:
		return ""
	case nc > 0 && float64(cpu) >= richOver*nc && float64(memory) >= richOver*nm:
		return ClassRich
	case float64(memory) > memoryAbove*float64(cpu):
		return ClassMemory
	case float64(memory) < computeBelow*float64(cpu):
		return ClassCompute
	}
	return ClassBalanced
}

func freeCPUMilli(cpu int) int64 {
	return max(0, int64(cpu)*1000-ReeveShare.CpuMilli)
}

func freeMemoryBytes(memory int) int64 {
	return max(0, int64(memory)*Gib-ReeveShare.MemBytes)
}

func TreesForACrew(kt KeepType) int {
	hands := math.Ceil(float64(crewCPUMilli+ReeveShare.CpuMilli) / 1000)
	return int(math.Ceil(hands*treesPerHand/LandYield(kt) - yieldEpsilon))
}

func StonesForACrew(kt KeepType) int {
	gibs := math.Ceil(float64(crewMemoryBytes+ReeveShare.MemBytes) / Gib)
	return int(math.Ceil(gibs/(LandYield(kt)*gibPerStone) - yieldEpsilon))
}

func Shortfall(sv Survey, kt KeepType) string {
	cpu, memory := freeCPUMilli(sv.CPU), freeMemoryBytes(sv.MemoryGiB)
	lowCPU, lowMemory := cpu < crewCPUMilli, memory < crewMemoryBytes
	switch {
	case lowCPU && lowMemory:
		return fmt.Sprintf("Too little forest and stone here: a crew needs 1,000 rations and 1,024 rooms free after the Reeve's share (%d trees and %d stones)",
			TreesForACrew(kt), StonesForACrew(kt))
	case lowCPU:
		return fmt.Sprintf("Too little forest here: %s free after the Reeve's share, a crew needs 1,000 (%d trees)",
			Rations(float64(cpu)), TreesForACrew(kt))
	case lowMemory:
		return fmt.Sprintf("Too little stone here: %s free after the Reeve's share, a crew needs 1,024 (%d stones)",
			Rooms(float64(memory)/Mebibyte), StonesForACrew(kt))
	}
	return ""
}

func (sv Survey) Err() error {
	if sv.OK {
		return nil
	}
	return errors.New(sv.Reason)
}
