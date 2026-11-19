package app

import (
	"maps"
	"slices"
	"strings"

	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	decreeGap     = .8
	unwardGap     = .3
	clerkBeside   = .5
	clerkBehind   = .2
	stewardBefore = .35
	nodePrefix    = "node/"
)

func init() {
	ErrandGaps[world.ErrandDecree] = decreeGap
	ErrandGaps[world.ErrandUnward] = unwardGap
}

func FrictionsOf(views []view.FrictionView) []world.Friction {
	out := make([]world.Friction, 0, len(views))
	for _, v := range views {
		out = append(out, world.Friction{Kind: v.Kind, Object: v.Object, Text: v.Text})
	}
	return out
}

func (s *Session) frictions() []world.Friction {
	return FrictionsOf(s.State.Friction)
}

func WardedNodes(views []view.FrictionView) map[string]bool {
	out := map[string]bool{}
	for _, v := range views {
		if name, ok := strings.CutPrefix(v.Object, nodePrefix); ok && v.Kind == world.FrictionWarded {
			out[name] = true
		}
	}
	return out
}

func (s *Session) scheduleTarget(pod string, keep view.Structure) world.Vec {
	if at, ok := s.Renderer.Camp(pod); ok {
		return world.Vec{X: at.X + clerkBeside, Y: at.Y - clerkBehind}
	}
	return world.Porch(keep)
}

func (s *Session) watchWards() {
	now := WardedNodes(s.State.Friction)
	before := s.Town.wards
	s.Town.wards = now
	if before == nil {
		return
	}
	from, ok := doorFor(s.Site, view.StructStewards)
	if !ok {
		return
	}
	for _, node := range slices.Sorted(maps.Keys(before)) {
		keep, found := world.KeepForNode(s.Site, node)
		if now[node] || !found || keep.State != view.StateReady {
			continue
		}
		ward := world.WardSpot(s.Site, keep)
		to := world.Vec{X: ward.X + stewardBefore, Y: ward.Y + stewardBefore}
		s.Town.send(world.Errand{Kind: world.ErrandUnward, Role: "steward", From: from, To: to}, s.Clocks.World)
	}
}

func (s *Session) decree(writes uint64) {
	from, okC := doorFor(s.Site, view.StructChancery)
	codex, okX := world.FirstOfKind(s.Site, view.StructCodex, true)
	if !okC || !okX || writes == 0 {
		return
	}
	s.Town.send(world.Errand{Kind: world.ErrandDecree, Role: "scribe", From: from, To: world.ClerkSpot(codex)}, s.Clocks.World)
}
