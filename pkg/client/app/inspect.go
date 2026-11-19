package app

import (
	"log"
	"slices"
	"strings"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

func InspectOf(sel hud.Selection) (view.Action, bool) {
	switch {
	case (sel.Kind == world.HitHearth || sel.Kind == world.HitWalker) && sel.Pod.Name != "":
		return view.Action{Action: view.ActInspect, Kind: "pod", Name: sel.Pod.Name}, true
	case sel.Kind == world.HitStructure && sel.Structure.Kind == view.StructKeep && sel.Structure.Object.Name != "":
		return view.Action{Action: view.ActInspect, Kind: "node", Name: sel.Structure.Object.Name}, true
	case sel.Kind == hud.SelHaven && sel.ID != "":
		return view.Action{Action: view.ActInspect, Kind: "port", Name: sel.ID}, true
	}
	return view.Action{}, false
}

func AgeCardLook(was, now string) (view.Action, bool) {
	if now != hud.OverlayAge || was == hud.OverlayAge {
		return view.Action{}, false
	}
	return view.Action{Action: view.ActInspect, Kind: "age"}, true
}

func (s *Session) Inspect(sel hud.Selection) {
	if a, ok := InspectOf(sel); ok {
		s.look(a)
	}
}

func (s *Session) lookAtPanel() {
	now := s.Hud.Overlay()
	if a, ok := AgeCardLook(s.panel, now); ok {
		s.look(a)
	}
	s.panel = now
}

func (s *Session) look(a view.Action) {
	if s.Live {
		return
	}
	if _, err := s.Realm.Perform(a); err != nil {
		log.Printf("inspect %s/%s: %v", a.Kind, a.Name, err)
	}
}

func IncidentMarks(cards []view.CalamityView, list []view.IncidentReport) []world.IncidentMark {
	open := map[string]view.IncidentReport{}
	for _, inc := range list {
		if inc.State == "open" {
			open[inc.ID] = inc
		}
	}
	out := []world.IncidentMark{}
	for _, c := range cards {
		inc, ok := open[c.Incident]
		pod, isPod := strings.CutPrefix(c.Object, "pod/")
		if ok && isPod {
			out = append(out, world.IncidentMark{Pod: pod, Crash: inc.Kind == view.IncidentCrashLoop})
		}
	}
	return out
}

const lookHere = "Go and look at the hut "

func PlaceNews(news []hud.Notice, places map[string][2]float64) []hud.Notice {
	out := make([]hud.Notice, len(news))
	copy(out, news)
	for i, n := range out {
		cut := strings.LastIndex(n.Detail, lookHere)
		if n.At != nil || cut < 0 {
			continue
		}
		if at, ok := hutPlace(places, strings.TrimSuffix(strings.TrimSpace(n.Detail[cut+len(lookHere):]), ".")); ok {
			out[i].At = &at
		}
	}
	return out
}

func hutPlace(places map[string][2]float64, name string) ([2]float64, bool) {
	keys := []string{}
	for k := range places {
		if strings.HasPrefix(k, "pod/") && strings.HasSuffix(k, "/"+name) {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return [2]float64{}, false
	}
	slices.Sort(keys)
	return places[keys[0]], true
}
