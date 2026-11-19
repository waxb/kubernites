package app

import (
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/soulstone"
	"github.com/waxb/kubernites/pkg/view"
)

func kindPile(site view.Site, kind string) (world.Vec, bool) {
	if "pile:"+kind == view.ParcelPile {
		return world.HarborFocus(site)
	}
	return world.YardFocus(site, kind)
}

func (s *Session) prideNotice(p view.Proclamation) {
	switch {
	case p.Kind == view.ProclaimTrade && p.Chest > 0:
		s.Hud.CelebrateChest(s.Frame(true), p.Title, p.Gain, p.Chest)
	case p.Kind == view.ProclaimTrade:
		s.Hud.Celebrate(s.Frame(true), p.Title, p.Gain)
	case p.Kind == view.ProclaimTried:
		s.Hud.Announce(p.Kind, p.Text, false)
	}
}

func WithTry(st view.GameState) view.GameState {
	if st.Try == nil {
		return st
	}
	out := st
	out.Objectives = make([]view.ObjectiveView, 0, len(st.Objectives)+1)
	placed := false
	for _, o := range st.Objectives {
		if !o.Done && !placed {
			out.Objectives, placed = append(out.Objectives, *st.Try), true
		}
		out.Objectives = append(out.Objectives, o)
	}
	if !placed {
		out.Objectives = append(out.Objectives, *st.Try)
	}
	return out
}

const stallHead = "stallspot:"

func (s *Session) stallSpot(kind string) (world.Vec, bool) {
	if d, ok := soulstone.DepotFor(kind); ok {
		if st, stands := world.DepotOf(s.Site, d.Service); stands {
			return world.Anchor(st), true
		}
	}
	stall := view.DepotStructure(kind)
	x, y, ok := s.Realm.SuggestPlacement(stall)
	if !ok {
		return world.Vec{}, false
	}
	g := s.GhostAt(stall, x, y)
	return world.Vec{X: float64(g.X) + float64(g.W-1)/2, Y: float64(g.Y) + float64(g.H-1)/2}, true
}

func (s *Session) DebutStall(kind string) (world.Vec, bool) {
	for _, d := range s.debuts {
		if d.Kind == kind && d.HasStall {
			return d.Stall, true
		}
	}
	return s.stallSpot(kind)
}

func (s *Session) debutsNow() []world.Debut {
	found := world.DebutsOf(s.State.Reveals, s.State.Freight)
	out := make([]world.Debut, 0, len(found))
	for _, d := range found {
		next := d
		next.Stall, next.HasStall = s.stallSpot(d.Kind)
		out = append(out, next)
	}
	return out
}

func (s *Session) TrackRollouts() {
	if s.State.Calm > 0 {
		return
	}
	next, done := s.Rollouts.Step(s.Crews, s.State.Freight)
	s.Rollouts = next
	for _, d := range done {
		s.Hud.CelebrateRollout(s.Frame(true), d)
	}
}
