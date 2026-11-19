package app

import "github.com/waxb/kubernites/pkg/view"

const parcelsDepot = "parcels"

func fallenOnRoll(roll view.RollView) []string {
	if !roll.Hand {
		return nil
	}
	out := []string{}
	for _, e := range roll.Entries {
		if !e.Live && e.Hut != "" {
			out = append(out, e.Hut)
		}
	}
	return out
}

func BadgeFirstAct(prev, cur view.RollView) bool {
	return (prev.Hand || prev.Bare) && !prev.LooksFor && cur.LooksFor && len(cur.Listed) > 0
}

func (s *Session) trackBadgeAct(prev view.RollView) {
	if !BadgeFirstAct(prev, s.State.Roll) {
		return
	}
	if at, ok := DepotCentre(s.Site, parcelsDepot); ok {
		s.GlideTo(at.X, at.Y)
	}
}

func idleOnRoll(roll view.RollView) []string {
	out := []string{}
	for _, e := range roll.Idle {
		if e.Hut != "" {
			out = append(out, e.Hut)
		}
	}
	return out
}
