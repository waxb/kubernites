package view

import (
	"math"
	"strconv"
	"strings"
)

const compactDigits = 2

const unitCarry = 1000

type compactUnit struct {
	at     float64
	suffix string
}

var compactUnits = []compactUnit{{1e12, "T"}, {1e9, "B"}, {1e6, "M"}, {1e3, "K"}}

func Compact(v float64) string {
	return CompactWith(v, compactDigits)
}

func RateText(v float64) string {
	a := math.Abs(v)
	switch {
	case a < .005:
		return "0"
	case a < 1:
		return strconv.FormatFloat(v, 'f', 2, 64)
	case a < 10:
		return strconv.FormatFloat(v, 'f', 1, 64)
	}
	return Compact(v)
}

func CompactWith(v float64, digits int) string {
	if v < 0 {
		return "-" + CompactWith(-v, digits)
	}
	for i, u := range compactUnits {
		if v >= u.at {
			if i > 0 && roundedTo(v/u.at, digits) >= unitCarry {
				u = compactUnits[i-1]
			}
			return trimmedFloat(v/u.at, digits) + u.suffix
		}
	}
	if v >= 100 || v == math.Trunc(v) {
		return strconv.Itoa(int(v))
	}
	return strconv.FormatFloat(v, 'f', 1, 64)
}

func roundedTo(v float64, digits int) float64 {
	scale := math.Pow(10, float64(digits))
	return math.Round(v*scale) / scale
}

func trimmedFloat(v float64, digits int) string {
	s := strconv.FormatFloat(v, 'f', digits, 64)
	return strings.TrimSuffix(strings.TrimRight(s, "0"), ".")
}
