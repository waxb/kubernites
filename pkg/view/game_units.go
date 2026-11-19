package view

import (
	"math"
	"strconv"
	"strings"
)

const RationsPerCPU = 1000

const roomsPerGiB = 1024

func Rations(milli float64) string {
	n := int64(math.Round(milli))
	if n == 1 {
		return "1 ration"
	}
	return Grouped(n) + " rations"
}

func Rooms(mib float64) string {
	n := int64(math.Round(mib))
	if n == 1 {
		return "1 room"
	}
	return Grouped(n) + " rooms"
}

func Barrels(mib float64) string {
	n := int64(math.Round(mib))
	if n == 1 {
		return "1 barrel"
	}
	return Grouped(n) + " barrels"
}

func MilliOf(cpu string) float64 {
	if milli, ok := strings.CutSuffix(cpu, "m"); ok {
		v, _ := strconv.ParseFloat(milli, 64)
		return v
	}
	v, _ := strconv.ParseFloat(cpu, 64)
	return v * RationsPerCPU
}

func RationsOf(cpu string) string { return Rations(MilliOf(cpu)) }

func MebiOf(memory string) float64 {
	for suffix, scale := range map[string]float64{"Gi": roomsPerGiB, "Mi": 1, "Ki": 1.0 / 1024} {
		if n, ok := strings.CutSuffix(memory, suffix); ok {
			v, _ := strconv.ParseFloat(n, 64)
			return v * scale
		}
	}
	v, _ := strconv.ParseFloat(memory, 64)
	return v / (1 << 20)
}

func RoomsOf(memory string) string { return Rooms(MebiOf(memory)) }

func Grouped(n int64) string {
	sign, digits := "", strconv.FormatInt(n, 10)
	if n < 0 {
		sign, digits = "-", digits[1:]
	}
	var b strings.Builder
	for i, d := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(d)
	}
	return sign + b.String()
}
