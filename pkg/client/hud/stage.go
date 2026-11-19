package hud

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	AppearFor  = .5
	keyStrip   = "panel:strip"
	KeyMinimap = "panel:minimap"
	KeyCard    = "panel:card"
	keyWell    = "panel:well"
	framePad   = 4
)

type appearances struct {
	primed  bool
	instant bool
	at      map[string]float64
}

func MinimapShown(f Frame) bool { return f.Shown(SurfaceMinimap) }

func CardShown(f Frame) bool {
	return f.Shown(SurfaceGate) && !f.State.Tutorial || !f.Selection.Empty()
}

func (h *HUD) WellShown(f Frame) bool {
	_, council := councilWellButton(f)
	return len(h.visibleTabs(f)) > 0 || council
}

func (h *HUD) StripShown(f Frame) bool {
	return MinimapShown(f) && CardShown(f) && h.WellShown(f)
}

func (h *HUD) shownKeys(f Frame) []string {
	out := []string{}
	for key, on := range map[string]bool{keyStrip: h.StripShown(f), KeyMinimap: MinimapShown(f), KeyCard: CardShown(f), keyWell: h.WellShown(f)} {
		if on {
			out = append(out, key)
		}
	}
	if !h.fullGrid(f) {
		for _, b := range h.CommandButtons(f) {
			out = append(out, b.ID)
		}
	}
	for _, st := range h.TopStats(f) {
		if st.Meter != "" {
			out = append(out, st.Meter)
		}
	}
	if BoardShown(f) {
		out = append(out, SurfaceFreight)
	}
	if LossesShown(f) {
		out = append(out, SurfaceLosses)
	}
	return out
}

func (h *HUD) NoteAppearances(f Frame) {
	if h.appear.at == nil {
		h.appear.at = map[string]float64{}
	}
	for _, key := range h.shownKeys(f) {
		if _, seen := h.appear.at[key]; seen {
			continue
		}
		h.appear.at[key] = f.Clock
		if !h.appear.primed {
			h.appear.at[key] = math.Inf(-1)
		}
	}
	h.appear.primed = true
}

func (h *HUD) SetAppearInstant(on bool) { h.appear.instant = on }

func (h *HUD) AppearProgress(key string, now float64) float64 {
	at, ok := h.appear.at[key]
	if !ok || h.Reduce || h.appear.instant {
		return 1
	}
	return min(max((now-at)/AppearFor, 0), 1)
}

func (h *HUD) withAppear(dst *ebiten.Image, key string, f Frame, draw func(*ebiten.Image)) {
	p := h.AppearProgress(key, f.Clock)
	if p >= 1 || h.fading || f.Reduce {
		draw(dst)
		return
	}
	if h.scratch == nil {
		h.scratch = ebiten.NewImage(Width, Height)
	}
	h.scratch.Clear()
	h.fading = true
	draw(h.scratch)
	h.fading = false
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.ScaleAlpha(float32(p))
	dst.DrawImage(h.scratch, op)
}

func CommandFrame(buttons []Button) ui.Rect {
	if len(buttons) == 0 {
		return ui.Rect{}
	}
	x0, y0 := Command.X+Command.W, Command.Y+Command.H
	for _, b := range buttons {
		x0, y0 = min(x0, b.Rect.X-framePad), min(y0, b.Rect.Y-framePad)
	}
	x0, y0 = max(x0, Command.X), max(y0, Command.Y)
	return ui.Rect{X: x0, Y: y0, W: Command.X + Command.W - x0, H: Command.Y + Command.H - y0}
}

func (h *HUD) CommandFrame(f Frame) ui.Rect {
	if h.fullGrid(f) {
		return Command
	}
	return CommandFrame(h.CommandButtons(f))
}

func (h *HUD) consoleRects(f Frame) []ui.Rect {
	if h.StripShown(f) {
		return append([]ui.Rect{ConsoleBox}, h.CrewRects(f)...)
	}
	out := h.CrewRects(f)
	for _, p := range []struct {
		r  ui.Rect
		on bool
	}{{Minimap, MinimapShown(f)}, {CardBox, CardShown(f)}, {InfoWell, h.WellShown(f)}} {
		if p.on {
			out = append(out, p.r)
		}
	}
	if r := h.CommandFrame(f); r.W > 0 {
		out = append(out, r)
	}
	return out
}

func (h *HUD) consoleCaptures(x, y int, f Frame) bool {
	for _, r := range h.consoleRects(f) {
		if r.Contains(x, y) {
			return true
		}
	}
	return false
}
