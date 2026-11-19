package app

import (
	"cmp"
	"fmt"
	"strings"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/sound"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/soulstone"
	"github.com/waxb/kubernites/pkg/view"
)

type bestPlot struct {
	kind, crew string
	x, y       int
}

func PlacementOf(a view.Action) (string, bool) {
	switch a.Action {
	case "buyKeep":
		kt, ok := view.KeepTypeOf(cmp.Or(a.KeepType, view.KeepTypeStone))
		return kt.Kind, ok
	case "openDepot":
		kind := cmp.Or(a.Kind, a.ID)
		if _, ok := soulstone.DepotFor(kind); ok {
			return view.DepotStructure(kind), true
		}
	}
	return "", false
}

func (s *Session) actOrPlace(it hud.Intent) []sound.SFX {
	if kind, ok := PlacementOf(it.Action); ok {
		return s.PlaceFor(kind, it.Action.For)
	}
	from := hud.EchoSource(it)
	a := it.Action
	a.Fix = a.Fix || hud.FixSource(from)
	return s.ActFrom(a, from)
}

func (s *Session) PlaceFor(kind, crew string) []sound.SFX {
	s.StartPlacingFor(kind, cmp.Or(crew, s.keepCrew(kind)))
	s.showPlot()
	return []sound.SFX{sound.SFXClick}
}

func (s *Session) StartPlacingFor(kind, crew string) {
	s.Placing, s.Ghost = kind, nil
	s.Selection = hud.Selection{}
	s.Land.best, s.Land.suggested = bestPlot{}, nil
	x, y, ok := s.suggestFor(kind, crew)
	if !ok {
		return
	}
	s.Ghost = s.GhostAt(kind, x, y)
	s.Land.suggested = s.Ghost
	if crew != "" {
		s.Land.best = bestPlot{kind: kind, crew: crew, x: x, y: y}
	}
}

func (s *Session) FirstKeepPlot() *world.Ghost {
	if s.Placing != view.StructKeep || hasKeep(s.Site) {
		return nil
	}
	return s.Land.suggested
}

func (s *Session) suggestFor(kind, crew string) (int, int, bool) {
	if crew == "" {
		return s.Realm.SuggestPlacement(kind)
	}
	return s.Realm.SuggestKeepFor(kind, crew)
}

func (s *Session) keepCrew(kind string) string {
	if !world.IsKeepKind(kind) || !hasKeep(s.Site) {
		return ""
	}
	kinds := make([]string, 0, len(s.State.Freight.Kinds))
	for _, k := range s.State.Freight.Kinds {
		kinds = append(kinds, k.CrewName)
	}
	return view.Content.KeepCrew(s.State.Economy.Fix, s.State.Crews, kinds)
}

func (b bestPlot) crewFor(kind string) string {
	if b.kind != kind {
		return ""
	}
	return b.crew
}

func (b bestPlot) marked(spots []world.SpotMark, placing string, footprint func(string) (int, int)) []world.SpotMark {
	if b.kind != placing || b.crew == "" {
		return spots
	}
	out := make([]world.SpotMark, 0, len(spots)+1)
	note, found := BestNote(b.crew), false
	for _, m := range spots {
		if m.X == b.x && m.Y == b.y {
			m.Note, found = note, true
		}
		out = append(out, m)
	}
	if !found {
		w, h := footprint(placing)
		out = append(out, world.SpotMark{X: b.x, Y: b.y, W: w, H: h, Note: note})
	}
	return out
}

func BestNote(crew string) string {
	return ui.Bridge("best for the "+crew, "best Node for "+crew)
}

func (s *Session) machineMarks(marks []world.SpotMark) []world.SpotMark {
	out := make([]world.SpotMark, 0, len(marks))
	for _, m := range marks {
		next := m
		next.Machine = MachineNote(s.Realm.PlotSurvey(s.Placing, m.X, m.Y))
		out = append(out, next)
	}
	return out
}

func MachineNote(sv view.Survey) string {
	if sv.Instance == "" {
		return ""
	}
	return ui.Bridge(fmt.Sprintf("%s: %s, %s", sv.Instance, view.Rations(float64(sv.CPU)*view.RationsPerCPU), view.Rooms(float64(sv.MemoryGiB)*1024)), fmt.Sprintf("%s: %d CPU, %d GiB memory", sv.Instance, sv.CPU, sv.MemoryGiB))
}

func bestSpot(marks []world.SpotMark) (world.SpotMark, bool) {
	for _, m := range marks {
		if m.Note != "" {
			return m, true
		}
	}
	return world.SpotMark{}, false
}

func depotOfKind(kind string) (soulstone.Depot, bool) {
	box, ok := strings.CutPrefix(kind, view.DepotStructure(""))
	if !ok {
		return soulstone.Depot{}, false
	}
	return soulstone.DepotFor(box)
}

func DepotName(kind string) (string, bool) {
	d, ok := depotOfKind(kind)
	return ui.Bridge("The "+d.Service+" depot", "Service "+d.Service), ok
}

func DepotNote(kind string, f hud.Frame) string {
	d, ok := depotOfKind(kind)
	if !ok {
		return ""
	}
	if hud.HandDepot(d.Kind, f) {
		return ui.Bridge("Free: it hands each box to a hut on the roll you keep by hand.",
			"Free: a Service with no selector, so it routes these requests to the addresses on the EndpointSlice you keep by hand.")
	}
	return fmt.Sprintf(ui.Bridge("Free: it sends every box of its kind to the %s.", "Free: it routes these requests to the Pods labelled app=%s."), d.Crew)
}

func (s *Session) rememberHome() {
	if s.placeHome == nil {
		home := s.ViewTarget()
		s.placeHome = &home
	}
}

func (s *Session) returnHome() {
	home := s.placeHome
	s.placeHome = nil
	if home != nil {
		s.GlideTo(home.X, home.Y)
	}
}
