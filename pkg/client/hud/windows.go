package hud

import (
	"maps"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	PanelHarbor    = "harbor"
	PanelCrew      = "crew"
	PanelObjective = "objective"
	PanelLesson    = "lesson"
	PanelNotices   = "notices"
	PanelCalamity  = "calamity"
	PanelIntro     = "intro"
	IntentPanels   = "panels"
	panelTitleH    = 16
	dragSlop       = 4
	WindowPrefix   = "window"
	MenuPanels     = "menu:panels"
)

type PanelPin struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type windowSpec struct {
	Rect    func(*HUD, Frame) (ui.Rect, bool) `json:"-"`
	draw    func(*HUD, *ebiten.Image, Frame)
	buttons func(*HUD, Frame) []Button
}

type panelGrab struct {
	Id     string `json:"-"`
	fromX  int
	fromY  int
	start  ui.Rect
	moving bool
}

type windowState struct {
	pins  map[string]PanelPin
	order []string
	Grab  panelGrab `json:"-"`
}

var stackedPanels = []string{PanelHarbor, PanelObjective, PanelLesson, PanelNotices, PanelCalamity, PanelCrews, PanelCrew, PanelKind, PanelWorker}

var WindowSpecs = map[string]windowSpec{}

func KnownPanels(pins map[string]PanelPin) map[string]PanelPin {
	out := map[string]PanelPin{}
	for id, p := range pins {
		if slices.Contains(stackedPanels, id) || id == PanelIntro {
			out[id] = p
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (h *HUD) Panels() map[string]PanelPin { return maps.Clone(h.Win.pins) }

func (h *HUD) SetPanels(pins map[string]PanelPin) { h.Win = windowState{pins: KnownPanels(pins)} }

func clampPanel(r ui.Rect) ui.Rect {
	next := r
	next.X = min(max(r.X, 0), Width-r.W)
	next.Y = min(max(r.Y, TopH), ConsoleY-r.H)
	return next
}

func (f Frame) placed(id string, home ui.Rect) ui.Rect {
	p, ok := f.Panels[id]
	if !ok || home.W == 0 {
		return home
	}
	pinned := clampPanel(ui.Rect{X: p.X, Y: p.Y, W: home.W, H: home.H})
	if id == PanelNotices {
		return pinned
	}
	return clearOf(pinned, f.railBox())
}

func harborWindow(_ *HUD, f Frame) (ui.Rect, bool) {
	return FreightRect(f), BoardShown(f)
}

func ObjectiveWindow(h *HUD, f Frame) (ui.Rect, bool) {
	if !h.CardsDrawn(f) || !h.ObjectiveShown(f) {
		return ui.Rect{}, false
	}
	return h.ObjectiveRectOf(f), true
}

func lessonWindow(h *HUD, f Frame) (ui.Rect, bool) {
	if !h.LessonShown(f) {
		return ui.Rect{}, false
	}
	return h.LessonRect(f), true
}

func railWindow(h *HUD, f Frame) (ui.Rect, bool) {
	r := h.RailRect(f)
	return r, r.W > 0
}

func crewWindow(h *HUD, f Frame) (ui.Rect, bool) {
	rects := h.CrewRects(f)
	if len(rects) == 0 {
		return ui.Rect{}, false
	}
	return rects[0], true
}

func drawObjectiveWindow(h *HUD, dst *ebiten.Image, f Frame) {
	if h.CardsDrawn(f) {
		h.drawObjective(dst, f)
	}
}

func drawLessonWindow(h *HUD, dst *ebiten.Image, f Frame) {
	if h.CardsDrawn(f) {
		h.drawLesson(dst, f)
	}
}

func (w windowState) Stacked() []string {
	if len(w.order) == 0 {
		return stackedPanels
	}
	return w.order
}

func (w windowState) raised(id string) windowState {
	if !slices.Contains(stackedPanels, id) {
		return w
	}
	stack := slices.DeleteFunc(slices.Clone(w.Stacked()), func(s string) bool { return s == id })
	return windowState{pins: w.pins, order: append(stack, id), Grab: w.Grab}
}

func (w windowState) pinned(id string, r ui.Rect) windowState {
	pins := maps.Clone(w.pins)
	if pins == nil {
		pins = map[string]PanelPin{}
	}
	pins[id] = PanelPin{X: r.X, Y: r.Y}
	return windowState{pins: pins, order: w.order, Grab: w.Grab}
}

func (w windowState) homed(id string) windowState {
	pins := maps.Clone(w.pins)
	delete(pins, id)
	return windowState{pins: KnownPanels(pins), order: w.order, Grab: w.Grab}
}

func (w windowState) grabbed(g panelGrab) windowState {
	return windowState{pins: w.pins, order: w.order, Grab: g}
}

func (h *HUD) introWindow(f Frame) (ui.Rect, bool) {
	if !h.IntroShown(f) {
		return ui.Rect{}, false
	}
	return h.IntroCard(f), true
}

func (h *HUD) windowAt(f Frame, x, y int) (string, ui.Rect, bool) {
	if r, ok := h.introWindow(f); ok && r.Contains(x, y) {
		return PanelIntro, r, true
	}
	if h.OverlayID != OverlayNone {
		return "", ui.Rect{}, false
	}
	stack := h.Win.Stacked()
	for i := len(stack) - 1; i >= 0; i-- {
		if r, ok := WindowSpecs[stack[i]].Rect(h, f); ok && r.Contains(x, y) {
			return stack[i], r, true
		}
	}
	return "", ui.Rect{}, false
}

func windowBody(id string, r ui.Rect) Button {
	return Button{ID: WindowPrefix + ":" + id, Rect: r}
}

func (h *HUD) windowButtons(f Frame) []Button {
	out := []Button{}
	for _, id := range h.Win.Stacked() {
		spec := WindowSpecs[id]
		if r, ok := spec.Rect(h, f); ok && h.OverlayID == OverlayNone {
			out = append(out, windowBody(id, r))
		}
		out = append(out, spec.buttons(h, f)...)
	}
	return out
}

func (h *HUD) introWindowButtons(f Frame) []Button {
	r, ok := h.introWindow(f)
	if !ok {
		return nil
	}
	return append([]Button{windowBody(PanelIntro, r)}, h.IntroButtons(f)...)
}

func (h *HUD) PanelRects(f Frame) []ui.Rect {
	out := append([]ui.Rect{TopBar}, h.consoleRects(f)...)
	if h.OverlayID != OverlayNone {
		out = append(out, h.OverlayRect())
	}
	for _, id := range h.Win.Stacked() {
		if r, ok := WindowSpecs[id].Rect(h, f); ok {
			out = append(out, windowClaim(id, r))
		}
	}
	if r, ok := h.introWindow(f); ok {
		out = append(out, r)
	}
	if r, ok := h.TrayRect(f); ok {
		out = append(out, r)
	}
	if h.LegendShown(f) {
		out = append(out, LegendRect(h.lens))
	}
	if SiegeActive(f) {
		out = append(out, SiegeStripRect(f))
	}
	if r, ok := h.SurveyRect(f); ok {
		out = append(out, r)
	}
	if r, ok := h.BannerRect(f); ok {
		out = append(out, r.Inset(-TipClear))
	}
	if h.IntroShown(f) && !h.IntroHidden(f) {
		if r, ok := h.IntroHole(f); ok {
			out = append(out, ringSurface("", r).Rect)
		}
	}
	for i := range out {
		out[i] = Shadowed(out[i])
	}
	return out
}

func windowClaim(id string, r ui.Rect) ui.Rect {
	if id == PanelNotices {
		return r.Inset(-1)
	}
	return r
}

func (h *HUD) drawWindows(dst *ebiten.Image, f Frame) {
	for _, id := range h.Win.Stacked() {
		if _, open := WindowSpecs[id].Rect(h, f); !open {
			continue
		}
		ui.InRegion(WindowRegion+id, func() { WindowSpecs[id].draw(h, dst, f) })
	}
}

func titleBar(id string, r ui.Rect) ui.Rect {
	height := panelTitleH
	if id == PanelNotices {
		height = railGripH
	}
	return ui.Rect{X: r.X, Y: r.Y, W: r.W, H: min(height, r.H)}
}

func isControl(b Button) bool {
	return (b.Label != "" || b.Icon != "") && !strings.HasPrefix(b.ID, WindowPrefix+":")
}

func (h *HUD) resetPanels() []Intent {
	h.Win, h.OverlayID = windowState{}, OverlayNone
	return []Intent{{Kind: IntentPanels}, {Kind: IntentSound, Command: "click"}}
}

func init() {
	prefixHandlers[WindowPrefix] = func(*HUD, string, Frame) []Intent { return nil }
	WindowSpecs[PanelHarbor] = windowSpec{Rect: harborWindow, draw: (*HUD).drawFreight, buttons: (*HUD).FreightButtons}
	WindowSpecs[PanelObjective] = windowSpec{Rect: ObjectiveWindow, draw: drawObjectiveWindow, buttons: (*HUD).ObjectiveButtons}
	WindowSpecs[PanelLesson] = windowSpec{Rect: lessonWindow, draw: drawLessonWindow, buttons: (*HUD).LessonButtons}
	WindowSpecs[PanelNotices] = windowSpec{Rect: railWindow, draw: (*HUD).drawRail, buttons: (*HUD).RailButtons}
	WindowSpecs[PanelCalamity] = windowSpec{Rect: (*HUD).CalamityRect, draw: (*HUD).DrawCalamity, buttons: (*HUD).CalamityButtons}
	WindowSpecs[PanelCrew] = windowSpec{Rect: crewWindow, draw: (*HUD).DrawCrewCard, buttons: (*HUD).CrewButtons}
}
