package app

import (
	"strings"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/world"
)

func KeepChosen(sel hud.Selection, crews []model.CrewChain) hud.Selection {
	if sel.Kind != hud.SelectCrew && sel.Kind != hud.SelectForeman {
		return sel
	}
	for _, c := range crews {
		if hud.ChainHolds(c, sel) {
			return sel
		}
	}
	return hud.Selection{}
}

func Chosen(it hud.Intent, pods []model.PodView) (hud.Selection, bool) {
	switch it.Target {
	case hud.SelectCrew, hud.SelectForeman:
		return hud.Selection{Kind: it.Target, ID: it.Command}, true
	case hud.SelectHearth:
		for _, p := range pods {
			if p.Namespace+"/"+p.Name == it.Command {
				return hud.Selection{Kind: world.HitHearth, ID: it.Command, Pod: p}, true
			}
		}
	}
	return hud.Selection{}, false
}

func (s *Session) Choose(it hud.Intent) {
	sel, ok := Chosen(it, s.Pods)
	if !ok {
		return
	}
	s.Selection = sel
	s.Inspect(sel)
	if at, found := s.chosenAt(sel); found {
		s.GlideTo(at.X, at.Y)
	}
}

func (s *Session) chosenAt(sel hud.Selection) (world.Vec, bool) {
	switch sel.Kind {
	case world.HitHearth:
		return ResolveTarget("pod:"+sel.Pod.Name, s.Site, s.Renderer.Home)
	case hud.SelectForeman:
		return s.Renderer.ForemanAt(sel.ID)
	}
	for _, c := range s.Crews {
		if c.ID() == sel.ID && len(c.Foremen) > 0 {
			return s.Renderer.ForemanAt(c.Foremen[0].ID())
		}
	}
	return world.Vec{}, false
}

const ReplaceAct = "replace"

func (a Autonomy) WithoutForemen(crews []model.CrewChain) Autonomy {
	next := a
	next.Bursts = nil
	for _, b := range a.Bursts {
		if b.Kind != ReplaceAct || !foremanPod(crews, b.Object.Name) {
			next.Bursts = append(next.Bursts, b)
		}
	}
	return next
}

func foremanPod(crews []model.CrewChain, pod string) bool {
	for _, c := range crews {
		for _, f := range c.Foremen {
			if !c.Bare && strings.HasPrefix(pod, f.Name+"-") {
				return true
			}
		}
	}
	return false
}
