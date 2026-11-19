package hud

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	lensCycle   = "cycle"
	LensCycleID = lensPrefix + ":" + lensCycle
	lensButtonW = 60
	lensButtonH = 14
	lensLabel   = "MAP"
)

func LensButtonShown(f Frame) bool {
	lens := f.Revealed(LensReveal) && f.Shown(SurfaceOverlays)
	herald := f.Revealed(heraldryReveal) && f.Shown(SurfaceHeraldry)
	return (lens || herald) && MinimapShown(f)
}

func lensCycleTip(f Frame) string {
	if !f.Revealed(LensReveal) {
		return "Show the fiefs on the map: the Heraldry view [O]"
	}
	return "Show risk on the map, one overlay at a time [O]"
}

func LensButtonRect() ui.Rect {
	return ui.Rect{X: Minimap.X + 4, Y: Minimap.Y + 4, W: lensButtonW, H: lensButtonH}
}

func (h *HUD) lensCycleButtons(f Frame) []Button {
	if !LensButtonShown(f) {
		return nil
	}
	label := lensLabel
	if info, ok := lensInfos[h.lens]; ok {
		label = info.label
	}
	return []Button{{ID: LensCycleID, Rect: LensButtonRect(), Label: label, Tip: lensCycleTip(f), Selected: h.lens != LensNone}}
}

func (h *HUD) drawLensButton(dst *ebiten.Image, f Frame) {
	for _, b := range h.lensCycleButtons(f) {
		h.drawButton(dst, b)
	}
}
