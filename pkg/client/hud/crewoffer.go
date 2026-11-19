package hud

import (
	"cmp"

	"github.com/waxb/kubernites/pkg/view"
)

func buyOffer(f Frame, a view.Action) (view.CrewKeepOffer, bool) {
	cv, ok := crewByName(f, a.For)
	if a.Action != "buyKeep" || !ok || cv.KeepOffer == nil || cv.KeepOffer.KeepType != cmp.Or(a.KeepType, view.KeepTypeStone) {
		return view.CrewKeepOffer{}, false
	}
	return *cv.KeepOffer, true
}

func buyPayoff(f Frame, a *view.Action) string {
	if a == nil {
		return ""
	}
	o, ok := buyOffer(f, *a)
	if !ok {
		return ""
	}
	cv, _ := crewByName(f, a.For)
	return offerPayoff(o, a.For, cv.Pending, cv.PerReplica.CPU > 0 || cv.PerReplica.Memory > 0)
}

func withPayoff(l CommandLine, f Frame) CommandLine {
	next := l
	next.Payoff = buyPayoff(f, l.Action)
	return next
}
