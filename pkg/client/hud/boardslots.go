package hud

import (
	"slices"

	"github.com/waxb/kubernites/pkg/client/model"
)

const (
	SlotStanding = "standing"
	SlotRising   = "rising"
	SlotFits     = "fits"
	SlotWaits    = "waits"
	SlotLeaving  = "leaving"
	PlotOpen     = "open"
	PlotNoLand   = "noland"
)

type Slot struct {
	Kind string
	Pod  model.PodView
	Real bool
}

func crewPods(c model.CrewChain, f Frame) []model.PodView {
	byName := make(map[string]model.PodView, len(f.Pods))
	for _, p := range f.Pods {
		byName[p.Namespace+"/"+p.Name] = p
	}
	out := []model.PodView{}
	for _, fm := range c.Foremen {
		for _, name := range fm.Pods {
			if p, ok := byName[fm.Namespace+"/"+name]; ok && !p.Terminating {
				out = append(out, p)
			}
		}
	}
	slices.SortStableFunc(out, func(a, b model.PodView) int { return waitRank(a) - waitRank(b) })
	return out
}

func waitRank(p model.PodView) int {
	if p.State == model.HearthPending {
		return 1
	}
	return 0
}

func BoardSlots(pods []model.PodView, target, fits int) []Slot {
	out := make([]Slot, 0, max(target, len(pods)))
	room := fits
	for i, p := range pods {
		kind := podSlotKind(p, i < target)
		if kind == SlotWaits && room > 0 {
			kind, room = SlotRising, room-1
		}
		out = append(out, Slot{Kind: kind, Pod: p, Real: true})
	}
	for i := len(pods); i < target; i++ {
		kind := SlotWaits
		if room > 0 {
			kind, room = SlotFits, room-1
		}
		out = append(out, Slot{Kind: kind})
	}
	return out
}

func podSlotKind(p model.PodView, kept bool) string {
	switch {
	case !kept:
		return SlotLeaving
	case p.State == model.HearthPending:
		return SlotWaits
	case p.State == model.HearthReady:
		return SlotStanding
	}
	return SlotRising
}

func CountSlots(slots []Slot, kind string) int {
	n := 0
	for _, s := range slots {
		if s.Kind == kind {
			n++
		}
	}
	return n
}

func placedCount(pods []model.PodView) int {
	n := 0
	for _, p := range pods {
		if p.State != model.HearthPending {
			n++
		}
	}
	return n
}
