package hud

import (
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const guideCards = false

func placingGuideTarget(f Frame) bool {
	if f.Placing == "" {
		return false
	}
	goal := ObjectiveGoal(f)
	if f.Introducing() {
		goal = f.Guide.Intro.Target
	}
	return goal == "build:"+f.Placing
}

func (h *HUD) objectiveSpot(f Frame) (ui.Rect, bool) {
	if !guideCards || h.IntroShown(f) || h.Voice(f) != VoiceObjective || !h.CardsDrawn(f) || !h.ObjectiveShown(f) {
		return ui.Rect{}, false
	}
	if o, _ := f.current(); o.Wait && h.Director.waitSeen == o.ID || o.Practiced {
		return ui.Rect{}, false
	}
	s, ok := h.Step(f)
	if !ok || s.Map || researchStep(f) {
		return ui.Rect{}, false
	}
	return s.Rect, true
}

func (h *HUD) noteWaitSeen(in Input, f Frame) {
	if !in.Click {
		return
	}
	if _, lit := h.objectiveSpot(f); !lit {
		return
	}
	if o, _ := f.current(); o.Wait {
		h.Director.waitSeen = o.ID
	}
}

func researchStep(f Frame) bool {
	o, ok := f.current()
	return ok && o.Page != ""
}

func (h *HUD) drawObjectiveSpot(dst *ebiten.Image, f Frame) {
	hole, ok := h.objectiveSpot(f)
	if !ok {
		return
	}
	card, in := h.objectiveGuideCard(f, hole)
	h.strokeHole(dst, hole)
	h.drawClearPointer(dst, f, card, hole)
	h.DrawIntroCard(dst, card, in)
}

func (h *HUD) objectiveGuideCard(f Frame, hole ui.Rect) (ui.Rect, Intro) {
	o, _ := f.current()
	in := Intro{ID: "objective:" + o.ID, Title: guideWhat(f.QuoteObjective(o.Text)), Purpose: o.Purpose, Try: guideTry(f.QuoteObjective(h.readyFrom(f, o.Progress)))}
	return h.placeGuideCard(f, hole, true, IntroW, h.introBodyHeight(in)), in
}

func guideWhat(text string) string {
	what, _, _ := strings.Cut(text, ": ")
	return strings.TrimSpace(what)
}

func guideTry(progress string) string {
	try := strings.TrimSpace(strings.TrimPrefix(progress, "ready:"))
	if try == "" {
		return "Press the highlighted button."
	}
	return strings.ToUpper(try[:1]) + try[1:] + "."
}

const EdgeArrowTip, edgeArrowBack, edgeArrowWing = 13, 7, 5

func (h *HUD) edgeArrowPoints(at ui.Rect, f Frame) ([3][2]float32, bool) {
	s, _ := h.Step(f)
	ax, ay := float64(at.X+at.W/2), float64(at.Y+at.H/2)
	dx, dy := float64(s.Rect.X+s.Rect.W/2)-ax, float64(s.Rect.Y+s.Rect.H/2)-ay
	n := math.Hypot(dx, dy)
	if n == 0 {
		return [3][2]float32{}, false
	}
	ux, uy := float32(dx/n), float32(dy/n)
	cx, cy := float32(ax), float32(ay)
	return [3][2]float32{
		{cx + ux*EdgeArrowTip, cy + uy*EdgeArrowTip},
		{cx - ux*edgeArrowBack - uy*edgeArrowWing, cy - uy*edgeArrowBack + ux*edgeArrowWing},
		{cx - ux*edgeArrowBack + uy*edgeArrowWing, cy - uy*edgeArrowBack - ux*edgeArrowWing},
	}, true
}

func (h *HUD) edgeArrowRect(at ui.Rect, f Frame) ui.Rect {
	pts, ok := h.edgeArrowPoints(at, f)
	if !ok {
		return ui.Rect{X: at.X + at.W/2, Y: at.Y + at.H/2, W: 1, H: 1}
	}
	x0, y0, x1, y1 := pts[0][0], pts[0][1], pts[0][0], pts[0][1]
	for _, p := range pts[1:] {
		x0, y0, x1, y1 = min(x0, p[0]), min(y0, p[1]), max(x1, p[0]), max(y1, p[1])
	}
	left, top := int(math.Floor(float64(x0)))-1, int(math.Floor(float64(y0)))-1
	return ui.Rect{X: left, Y: top, W: int(math.Ceil(float64(x1))) + 2 - left, H: int(math.Ceil(float64(y1))) + 2 - top}
}

func (h *HUD) drawEdgeArrow(dst *ebiten.Image, at ui.Rect, f Frame) {
	pts, ok := h.edgeArrowPoints(at, f)
	if !ok {
		return
	}
	var p vector.Path
	p.MoveTo(pts[0][0], pts[0][1])
	p.LineTo(pts[1][0], pts[1][1])
	p.LineTo(pts[2][0], pts[2][1])
	p.Close()
	op := &vector.DrawPathOptions{}
	op.ColorScale.ScaleWithColor(ui.Tint(ui.Palette.GoldBright, uint8(160+95*pulse(h.Now, h.Reduce))))
	vector.FillPath(dst, &p, &vector.FillOptions{}, op)
}

func (h *HUD) drawWorldDim(dst *ebiten.Image, f Frame) {
	if h.IntroShown(f) {
		target, ok := h.IntroHole(f)
		dimAround(dst, target, ok && !h.IntroHidden(f))
		return
	}
	if hole, ok := h.objectiveSpot(f); ok {
		dimAround(dst, hole, true)
	}
}
