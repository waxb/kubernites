package view

import (
	"sort"
)

const CitadelFief = "kube-system"

const FiefColours = 12

const citadelOrder = 0

func Heraldry(namespaces []string) map[string]int {
	names := append([]string(nil), namespaces...)
	sort.Strings(names)
	out := map[string]int{}
	next := citadelOrder + 1
	for _, n := range names {
		if _, seen := out[n]; seen {
			continue
		}
		if n == CitadelFief {
			out[n] = citadelOrder
			continue
		}
		out[n] = next
		next++
	}
	return out
}

func FiefColour(order int) int {
	return order % FiefColours
}

func FiefCrest(order int) int {
	return order / FiefColours % 2
}
