package model

import (
	"math"
	"sort"

	"github.com/waxb/kubernites/pkg/view"
)

const (
	UnservedNoDepot = "no-depot"
	UnservedNoCrew  = "no-crew"
	UnservedFew     = "few"

	unservedFloor   = 0.05
	unservedShare   = 0.05
	unservedWaiting = 1.0
)

type Unserved struct {
	Kind   string
	Name   string
	Crew   string
	Depot  string
	Cause  string
	Rate   float64
	Gold   float64
	Queued bool
}

func (u Unserved) Stranded() bool { return u.Cause != UnservedFew }

func quayLost(k view.KindView) float64 {
	l := k.Losses
	return l.Spoiled + l.NoDepot + l.NoCrew + l.QuayFull
}

func pileGrowth(k view.KindView) float64 {
	if k.Waiting < unservedWaiting {
		return 0
	}
	return max(0, k.ArrivalRate-k.ServedRate)
}

func UnservedRate(k view.KindView) float64 {
	return max(quayLost(k), pileGrowth(k))
}

func unservedCause(k view.KindView) string {
	switch {
	case !k.DepotOpen && !k.Door:
		return UnservedNoDepot
	case k.Crew == 0 && !k.Door:
		return UnservedNoCrew
	}
	return UnservedFew
}

func UnservedOf(k view.KindView) (Unserved, bool) {
	rate, bar := UnservedRate(k), max(unservedFloor, unservedShare*k.ArrivalRate)
	if rate < bar {
		return Unserved{}, false
	}
	cause := unservedCause(k)
	queued := quayLost(k) < bar || cause == UnservedFew && k.SpoiledRate < bar
	if !queued {
		rate = quayLost(k)
	}
	return Unserved{Kind: k.Kind, Name: k.Name, Crew: k.CrewName, Depot: k.Depot, Cause: cause,
		Rate: round3(rate), Gold: round3(rate * k.Value), Queued: queued}, true
}

func UnservedKinds(fv view.FreightView) []Unserved {
	out := []Unserved{}
	for _, k := range fv.Kinds {
		if u, ok := UnservedOf(k); ok {
			out = append(out, u)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Gold > out[j].Gold })
	return out
}

func WorstUnserved(fv view.FreightView) (Unserved, bool) {
	all := UnservedKinds(fv)
	if len(all) == 0 {
		return Unserved{}, false
	}
	return all[0], true
}

func round3(v float64) float64 {
	return math.Round(v*1000) / 1000
}
