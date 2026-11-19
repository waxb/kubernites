package app

import (
	"strings"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/soulstone"
	"github.com/waxb/kubernites/pkg/view"
)

func IntroTargets(ids []string) []string { return hud.IntroTargets(ids) }

func GuideOf(st view.GameState) hud.Guide {
	g := hud.Guide{Staged: st.Introduced != nil, Introduced: IntroTargets(st.Introduced), Effects: EffectsOf(st)}
	if in := st.Intro; in != nil {
		g.Intro = &hud.Intro{ID: in.ID, Title: in.Title, Purpose: in.Purpose, Lore: in.Lore, Try: in.Try, Target: in.Target, Via: in.Via}
		g.Queue = IntroTitles(st.IntroQueue)
	}
	for _, o := range st.Objectives {
		if !o.Done {
			g.Why, g.Target, g.Button = o.Why, ObjectiveTarget(o), o.Button
			break
		}
	}
	return g
}

func IntroTitles(ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		for _, in := range view.Content.AllIntros() {
			if in.ID == id {
				out = append(out, in.Title)
			}
		}
	}
	return out
}

func ObjectiveTarget(o view.ObjectiveView) string {
	if o.Build != "" {
		return "build:" + o.Build
	}
	return o.Target
}

func EffectsOf(st view.GameState) map[string]string {
	out := make(map[string]string, len(st.Effects)+len(st.Buildables)+len(st.Soulstones))
	for id, e := range st.Effects {
		out[id] = e
	}
	for _, b := range st.Buildables {
		if b.Effect != "" {
			out["build:"+b.Kind] = b.Effect
		}
	}
	for _, s := range st.Soulstones {
		if s.Effect != "" {
			out["summon:"+s.Image] = s.Effect
		}
	}
	return out
}

func (s *Session) CurrentGuide() hud.Guide {
	if s.GuideFixture != nil {
		return *s.GuideFixture
	}
	return GuideOf(s.State)
}

func ResolveTarget(target string, site view.Site, home func(string) (world.Vec, bool)) (world.Vec, bool) {
	kind, name, _ := strings.Cut(target, ":")
	switch kind {
	case "pod":
		_, pod, found := strings.Cut(name, "/")
		if !found {
			pod = name
		}
		if v, ok := home(pod); ok {
			return world.Vec{X: v.X + .5, Y: v.Y + .5}, true
		}
	case "keep":
		if k, ok := world.KeepForNode(site, name); ok {
			return world.Anchor(k), true
		}
	case "structure":
		return structureSpot(site, name)
	case "crew":
		return crewSpot(site, target)
	case "haven":
		return world.HarborSign(site, name)
	case "pile":
		return kindPile(site, name)
	}
	return world.Vec{}, false
}

func CrewHut(target string, pods []model.PodView) (string, bool) {
	name, ok := strings.CutPrefix(target, "crew:")
	if !ok {
		return "", false
	}
	_, crew, _ := strings.Cut(name, "/")
	for _, p := range pods {
		if p.Crew == crew && crew != "" && !p.Terminating {
			return p.Name, true
		}
	}
	return "", false
}

func crewSpot(site view.Site, target string) (world.Vec, bool) {
	crew, ok := hud.CrewPlace(target)
	if !ok {
		return world.Vec{}, false
	}
	d, ok := soulstone.DepotOfCrew(crew)
	if !ok {
		return world.Vec{}, false
	}
	for _, s := range site.Structures {
		if s.Object.Kind == "Service" && s.Object.Name == d.Service {
			return world.Anchor(s), true
		}
	}
	return world.Vec{}, false
}

func structureSpot(site view.Site, name string) (world.Vec, bool) {
	for _, s := range site.Structures {
		if s.ID == name {
			return world.Anchor(s), true
		}
	}
	if s, ok := world.FirstOfKind(site, name, false); ok {
		return world.Anchor(s), true
	}
	return world.Vec{}, false
}

func (s *Session) Goal() (world.Vec, bool) {
	target := s.GuideGoal()
	if kind, ok := strings.CutPrefix(target, stallHead); ok {
		return s.DebutStall(kind)
	}
	if !hud.MapTarget(target) {
		return world.Vec{}, false
	}
	if v, ok := StorageTarget(target, s.State, s.Site, s.Renderer.Home); ok {
		return v, true
	}
	if pod, ok := CrewHut(target, s.Pods); ok {
		if v, ok := ResolveTarget("pod:"+pod, s.Site, s.Renderer.Home); ok {
			return v, true
		}
	}
	return ResolveTarget(target, s.Site, s.Renderer.Home)
}

func (s *Session) goalOfNext() bool {
	_, next := s.guideTarget()
	return next
}

func (s *Session) goalAt() *[2]float64 {
	v, ok := s.Goal()
	if !ok {
		return nil
	}
	return &[2]float64{v.X, v.Y}
}

func (s *Session) BeaconGoal() (world.Vec, bool) {
	if s.Hud.InChapter() {
		return s.teacherBeacon()
	}
	if s.GuideGoal() == view.ParcelPile {
		return world.Vec{}, false
	}
	if v, ok := s.stoneBeacon(); ok {
		return v, true
	}
	return s.Goal()
}
