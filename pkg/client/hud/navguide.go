package hud

import (
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	SurfaceLoom    = "surface:loom"
	SurfaceReforge = "surface:reforge"
	reforgeDim     = 170
)

type GuideStep struct {
	ID    string
	Rect  ui.Rect
	Map   bool
	Final bool
}

func (h *HUD) NavState(f Frame) NavState {
	st := NavState{Overlay: h.OverlayID, Page: h.PageID, Tab: h.activeTab(f), Council: h.CouncilTab(), Grimoire: h.cardPage(f), Harbor: HavenSelected(f), Chain: h.ChainShown(f)}
	st.Object = h.objectShown(f)
	st.Crew = st.Object && h.crewGrid(f)
	st.Sub = st.Crew && hearthSelected(f)
	if c, ok := ChainOf(f); ok && st.Crew {
		st.Mode = h.CrewMode(c)
	}
	if st.Harbor {
		st.Haven = f.Selection.ID
	}
	return st
}

func ObjectiveGoal(f Frame) string {
	o, ok := f.current()
	if !ok || f.Guide.Held != "" {
		return ""
	}
	if f.Guide.Target != "" {
		return CardGoal(f)
	}
	kind, builds := objectiveBuilds[o.ID]
	switch {
	case !builds:
		return ""
	case strings.HasPrefix(o.Progress, "gold ") && HandsNeeded(f.State.Freight):
		return view.ParcelPile
	case o.Progress == "being built" || strings.HasPrefix(o.Progress, "gold "):
		return ""
	}
	return "build:" + kind
}

func (h *HUD) GuideGoal(f Frame) string {
	switch h.Voice(f) {
	case VoiceIntro:
		return gridGoal(h.chooserGoal(f.Guide.Intro.Target, f), f)
	case VoiceObjective:
		return gridGoal(h.chooserGoal(ObjectiveGoal(f), f), f)
	case VoiceNext:
		return gridGoal(NextGoal(f), f)
	}
	return ""
}

func (h *HUD) chooserGoal(goal string, f Frame) string {
	switch {
	case goal == SurfaceStone && h.OverlayID != OverlayStone && f.SpeakCommand() != "" && h.ObjectiveDrawn(f):
		return ObjSpeak
	case goal == "cmd:grimoire" && learnPage(f) != "":
		return "learn:" + learnPage(f)
	case goal == CmdSummonID && h.PageID == PageSummon:
		if cells := SoulButtons(f, "summon:", "Hire"); len(cells) > 0 {
			return cells[0].ID
		}
	case strings.HasPrefix(goal, CrewModeOpen):
		return h.crewPageGoal(goal, f)
	case goal == view.WatchButton:
		return h.watchGoal(goal, f)
	case goal == "cmd:soulvault" && h.OverlayID == OverlaySoulvault && GuidedClass(f) != "":
		return f.Guide.Button
	case goal == AgeButtonID && h.OverlayID == OverlayAge && f.State.AgeUp.Available:
		return AgeAdvance
	case goal == CmdCrew && h.OverlayID == OverlayRaise:
		return RaiseConfirm
	case goal == CmdCrew && h.tryButtonGoal(f) != "":
		return h.tryButtonGoal(f)
	case goal == CmdCrew && h.crewsShown(f):
		if line, ok := CrewLine(f); ok && line.Action != nil && line.Action.Name != "" {
			return CrewsKey + ":raise:" + line.Action.Name
		}
		if rows := creatableRows(RaiseRows(f)); len(rows) > 0 {
			return CrewsKey + ":raise:" + guidedCrew(f, rows)
		}
	}
	return goal
}

func (h *HUD) GuidePath(f Frame) []string {
	return NavPath(h.GuideGoal(f), h.NavState(f))
}

func (h *HUD) Step(f Frame) (GuideStep, bool) {
	return h.stepOn(h.GuidePath(f), f)
}

func (h *HUD) stepOn(path []string, f Frame) (GuideStep, bool) {
	if len(path) == 0 {
		return GuideStep{}, false
	}
	s := GuideStep{ID: path[0], Final: len(path) == 1, Map: MapTarget(path[0])}
	if s.Map {
		if len(f.Beacons) == 0 {
			return s, false
		}
		s.Rect = f.Beacons[0]
		return s, true
	}
	r, ok := h.StepRect(s.ID, f)
	s.Rect = r
	return s, ok
}

func (h *HUD) StepRect(id string, f Frame) (ui.Rect, bool) {
	if r, ok := h.ageSurfaceRect(id, f); ok {
		return r, true
	}
	if r, ok := h.TargetRect(id, f); ok {
		return r, true
	}
	if b, ok := HitID(h.OverlayButtons(f), id); ok {
		return b.Rect, true
	}
	if b, ok := HitID(h.ObjectiveButtons(f), id); ok {
		return b.Rect, true
	}
	if b, ok := HitID(h.FreightButtons(f), id); ok {
		return b.Rect, true
	}
	if b, ok := HitID(h.crewsButtons(f), id); ok {
		return b.Rect, true
	}
	if b, ok := HitID(h.CrewButtons(f), id); ok {
		return b.Rect, true
	}
	if b, ok := HitID(h.kindButtons(f), id); ok {
		return b.Rect, true
	}
	return ui.Rect{}, false
}

func (h *HUD) ageSurfaceRect(id string, f Frame) (ui.Rect, bool) {
	if h.OverlayID != OverlayAge {
		return ui.Rect{}, false
	}
	switch id {
	case SurfaceReforge:
		if b, ok := HitID(h.ageButtons(f), "reforge:open"); ok {
			return b.Rect, true
		}
	case SurfaceLoom:
		return loomArea(h.ageButtons(f))
	}
	return ui.Rect{}, false
}

func loomArea(buttons []Button) (ui.Rect, bool) {
	var out ui.Rect
	found := false
	for _, b := range buttons {
		if !strings.HasPrefix(b.ID, "loom:") {
			continue
		}
		out, found = unionRect(out, b.Rect, found), true
	}
	return out, found
}

func unionRect(a, b ui.Rect, have bool) ui.Rect {
	if !have {
		return b
	}
	x, y := min(a.X, b.X), min(a.Y, b.Y)
	return ui.Rect{X: x, Y: y, W: max(a.X+a.W, b.X+b.W) - x, H: max(a.Y+a.H, b.Y+b.H) - y}
}

const edgePointerSize, edgePointerPad = 20, 14

func viewArea() ui.Rect { return ui.Rect{X: 0, Y: TopH, W: Width, H: ConsoleY - TopH} }

func (h *HUD) IntroHidden(f Frame) bool {
	s, ok := h.Step(f)
	return ok && s.Map && !inView(s.Rect)
}

func inView(r ui.Rect) bool {
	v := viewArea()
	return r.X+r.W/2 >= v.X && r.X+r.W/2 < v.X+v.W && r.Y >= v.Y && r.Y+r.H <= v.Y+v.H
}

func edgePointer(r ui.Rect) ui.Rect {
	v := viewArea()
	cx := min(max(r.X+r.W/2, v.X+edgePointerPad), v.X+v.W-edgePointerPad)
	cy := min(max(r.Y+r.H/2, v.Y+edgePointerPad), v.Y+v.H-edgePointerPad)
	if cy >= v.Y+v.H-edgePointerPad && cx+edgePointerSize/2 > Command.X {
		cx = Command.X - edgePointerSize
	}
	return ui.Rect{X: cx - edgePointerSize/2, Y: cy - edgePointerSize/2, W: edgePointerSize, H: edgePointerSize}
}

func (h *HUD) clearOfObjective(r ui.Rect, f Frame) ui.Rect {
	if !h.CardsDrawn(f) || !h.ObjectiveShown(f) {
		return r
	}
	card := h.ObjectiveRectOf(f).Inset(-edgePointerPad)
	if !Overlaps(r, card) {
		return r
	}
	v := viewArea()
	below := ui.Rect{X: r.X, Y: card.Y + card.H, W: r.W, H: r.H}
	beside := ui.Rect{X: card.X + card.W, Y: r.Y, W: r.W, H: r.H}
	for _, c := range []ui.Rect{below, beside} {
		if c.Y+c.H <= v.Y+v.H && c.X+c.W <= v.X+v.W {
			return c
		}
	}
	return r
}

func (h *HUD) IntroHole(f Frame) (ui.Rect, bool) {
	if s, ok := h.Step(f); ok {
		if s.Map && !inView(s.Rect) {
			return h.clearOfObjective(edgePointer(s.Rect), f), true
		}
		return s.Rect, true
	}
	return h.TargetRect(f.Guide.Intro.Target, f)
}

func (h *HUD) IntroHoleFor(f Frame) (ui.Rect, bool) {
	f = h.Stage(f)
	if !h.IntroShown(f) {
		return ui.Rect{}, false
	}
	return h.IntroHole(f)
}

func (h *HUD) MapStep(f Frame) bool {
	s, ok := h.Step(f)
	return ok && s.Map && !f.Introducing()
}

func DimRects(area, hole ui.Rect, cut bool) []ui.Rect {
	if !cut || overlapArea(area, hole) == 0 {
		return []ui.Rect{area}
	}
	x0, y0 := max(hole.X, area.X), max(hole.Y, area.Y)
	x1, y1 := min(hole.X+hole.W, area.X+area.W), min(hole.Y+hole.H, area.Y+area.H)
	out := []ui.Rect{}
	for _, r := range []ui.Rect{
		{X: area.X, Y: area.Y, W: area.W, H: y0 - area.Y},
		{X: area.X, Y: y1, W: area.W, H: area.Y + area.H - y1},
		{X: area.X, Y: y0, W: x0 - area.X, H: y1 - y0},
		{X: x1, Y: y0, W: area.X + area.W - x1, H: y1 - y0},
	} {
		if r.W > 0 && r.H > 0 {
			out = append(out, r)
		}
	}
	return out
}

func (h *HUD) StepID(f Frame) string {
	if s, ok := h.Step(f); ok && !s.Map {
		return s.ID
	}
	return ""
}

func (h *HUD) cardPage(f Frame) string {
	if h.OverlayID != OverlayGrimoire {
		return h.Grimoire
	}
	return h.focusedPage(h.Knowledge(f))
}

func learnPage(f Frame) string {
	if o, ok := f.current(); ok {
		return o.Page
	}
	return ""
}
