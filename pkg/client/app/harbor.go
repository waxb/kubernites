package app

import (
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/sound"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

func GateHaven(site view.Site) (view.Haven, bool) {
	for _, h := range world.SiteHavens(site) {
		if h.ID == world.GateHavenID {
			return h, true
		}
	}
	return view.Haven{}, false
}

func (s *Session) SelectHarbor() []sound.SFX {
	return s.SelectHaven("")
}

func (s *Session) goTo(target string) []sound.SFX {
	v, ok := ResolveTarget(target, s.Site, s.Renderer.Home)
	if !ok {
		return nil
	}
	s.GlideTo(v.X, v.Y)
	return []sound.SFX{sound.SFXClick}
}

func havenOf(site view.Site, id string) (view.Haven, bool) {
	if id == "" {
		return GateHaven(site)
	}
	for _, h := range world.SiteHavens(site) {
		if h.ID == id {
			return h, true
		}
	}
	return view.Haven{}, false
}

func (s *Session) SelectHaven(id string) []sound.SFX {
	h, ok := havenOf(s.Site, id)
	if !ok {
		return nil
	}
	s.Selection = hud.Selection{Kind: hud.SelHaven, ID: h.ID}
	s.Inspect(s.Selection)
	s.GlideTo(float64(h.Quay.X), float64(h.Quay.Y))
	s.harborOpened()
	return []sound.SFX{sound.SFXClick}
}

func (s *Session) harborOpened() {
	if in := s.shownGuide().Intro; in != nil && in.Target == hud.SurfaceBerths {
		s.Act(hud.Ack(in.ID))
	}
}

func (s *Session) GuideGoal() string {
	target, _ := s.guideTarget()
	return hud.HarborStep(target, s.Selection)
}

func (s *Session) guideTarget() (string, bool) {
	g := s.shownGuide()
	switch in := g.Intro; {
	case in != nil && in.Target == hud.SurfaceQoS && s.Selection.Pod.Name == "":
		return s.qosHut(), false
	case in != nil && (in.Target == hud.SurfaceBerths || hud.MapTarget(in.Target)):
		return in.Target, false
	case g.Target == "" && g.Held == "" && s.State.Next != nil && s.State.Next.Ready:
		return s.State.Next.Target, true
	}
	return g.Target, false
}

func (s *Session) qosHut() string {
	for _, p := range s.Pods {
		if _, ok := s.Renderer.Home(p.Name); ok && !p.System && p.Namespace == "default" {
			return "pod:" + p.Namespace + "/" + p.Name
		}
	}
	return ""
}
