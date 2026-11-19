package app

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/sound"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

const hornLife = 1.0

func LedgerShare(l view.LedgerView) float64 {
	if l.Cap <= 0 {
		return 0
	}
	return min(l.Level/l.Cap, 1)
}

func DecorateSite(site view.Site, horns []model.Horn) view.Site {
	silent := map[string]bool{}
	for _, h := range horns {
		if !h.Sounding {
			silent[h.Name] = true
		}
	}
	structures := make([]view.Structure, len(site.Structures))
	for i, st := range site.Structures {
		if st.Kind == view.StructWarHorn && st.State == view.StateReady && silent[st.Object.Name] {
			st.State = view.StateDark
		}
		structures[i] = st
	}
	next := site
	next.Structures = structures
	return next
}

func nodeReady(n *corev1.Node) bool {
	for _, c := range n.Status.Conditions {
		if c.Type == corev1.NodeReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}

func KeepsReady(c *model.Cache) int {
	n := 0
	for _, node := range c.Nodes() {
		if nodeReady(node) {
			n++
		}
	}
	return n
}

func Joining(c *model.Cache) bool {
	for _, node := range c.Nodes() {
		if !nodeReady(node) {
			return true
		}
	}
	return false
}

func HornTargets(horns []model.Horn) map[string]bool {
	out := map[string]bool{}
	for _, h := range horns {
		out[h.Target] = true
	}
	return out
}

func (s *Session) HornNotices() []sound.SFX {
	var out []sound.SFX
	targets := HornTargets(s.Horns)
	next := make(map[string]int, len(s.Crews))
	for _, c := range s.Crews {
		name := hud.CrewName(c)
		next[name] = c.Desired
		if prev, ok := s.replicas[name]; ok && prev != c.Desired && targets[name] {
			s.Sounded()
			out = append(out, sound.SFXHorn)
		}
	}
	s.replicas = next
	return out
}

func (s *Session) Sounded() {
	horn, ok := world.FirstOfKind(s.Site, view.StructWarHorn, false)
	if !ok {
		return
	}
	s.Transients = append(s.Transients, world.Transient{Kind: "horn", At: world.Anchor(horn), Start: s.Clocks.World, Life: hornLife})
}
