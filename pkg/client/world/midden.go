package world

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	alarmFrames = 6
	alarmFPS    = 8
	alarmLift   = 4
	bloatDepth  = .02
	heapDepth   = .01
)

var (
	heapBrown = color.RGBA{112, 84, 58, 255}
	heapDark  = color.RGBA{72, 56, 42, 255}
	alarmRed  = color.RGBA{235, 70, 55, 255}
)

func MiddenName(level, variant int) string { return gfx.MiddenName(level, variant) }

func CodexBloatName(level int) string { return gfx.CodexBloatName(level) }

func middenFree(site view.Site, x, y int) bool {
	return !Wet(site.Tile(x, y)) && !Occupied(site, x, y) && x >= 0 && y >= 0 && x < site.Width && y < site.Height
}

func MiddenSpot(site view.Site, keep view.Structure) (Vec, bool) {
	lo, hi := Footprint(keep)
	for _, p := range []Point{{lo.X - 1, hi.Y + 1}, {hi.X + 1, lo.Y - 1}, {lo.X - 1, lo.Y - 1}} {
		if middenFree(site, p.X, p.Y) {
			return Vec{float64(p.X) + .5, float64(p.Y) + .5}, true
		}
	}
	return Vec{}, false
}

func (r *Renderer) middenItems(cam render.Camera, sc Scene) []Drawable {
	out := []Drawable{}
	for _, k := range Keeps(sc.Site) {
		m, ok := sc.Caesar.Middens[k.Object.Name]
		level := m.Level()
		if !ok || level < 0 {
			continue
		}
		spot, ok := MiddenSpot(sc.Site, k)
		if !ok {
			continue
		}
		sx, sy := ScreenPointOf(cam, sc.Site, spot)
		if !onScreen(sx, sy, cam.Scale()) {
			continue
		}
		name, zoom := MiddenName(level, int(model.Hash(k.ID))), cam.Scale()
		out = append(out, Drawable{Depth: render.DepthKey(spot.X, spot.Y) + heapDepth, Draw: func(dst *ebiten.Image) {
			if !r.Lib.Draw(dst, name, sx, sy, zoom, render.SpriteOptions{}) {
				drawHeapFallback(dst, sx, sy, zoom, level)
			}
		}})
	}
	return out
}

func drawHeapFallback(dst *ebiten.Image, sx, sy, zoom float64, level int) {
	w, h := float32((6+3*level)*int(max(zoom, 1))), float32((3+2*level)*int(max(zoom, 1)))
	vector.FillRect(dst, float32(sx)-w/2, float32(sy)-h, w, h, heapBrown, false)
	vector.FillRect(dst, float32(sx)-w/4, float32(sy)-h-h/2, w/2, h/2, heapDark, false)
}

func (r *Renderer) codexItems(cam render.Camera, sc Scene) []Drawable {
	c := sc.Caesar.Codex
	level := c.Level()
	s, ok := FirstOfKind(sc.Site, view.StructCodex, false)
	if !ok || !c.Known() || level < 0 {
		return nil
	}
	a := Anchor(s)
	sx, sy := ScreenPointOf(cam, sc.Site, Vec{a.X - .01, a.Y - .01})
	zoom, name := cam.Scale(), CodexBloatName(level)
	return []Drawable{{Depth: render.DepthKey(a.X, a.Y) + bloatDepth, Draw: func(dst *ebiten.Image) {
		r.Lib.Draw(dst, name, sx, sy, zoom, render.SpriteOptions{})
	}}}
}

func AlarmFrame(t float64, reduce bool) int {
	if reduce {
		return 0
	}
	return int(t*alarmFPS) % alarmFrames
}

func (r *Renderer) DrawCodexAlarm(dst *ebiten.Image, cam render.Camera, sc Scene) {
	s, ok := FirstOfKind(sc.Site, view.StructCodex, false)
	if !ok || !sc.Caesar.Codex.Alarm {
		return
	}
	zoom := cam.Scale()
	a := Anchor(s)
	sx, sy := ScreenPointOf(cam, sc.Site, a)
	if !onScreen(sx, sy, zoom) {
		return
	}
	top := sy - 40*zoom
	if head, ok := r.Lib.Headroom(r.resolveLook(s, sc.Age)[0]); ok {
		top = sy - float64(head)*zoom
	}
	name := gfx.AlarmName(AlarmFrame(sc.Time, sc.Reduce))
	if !r.Lib.Draw(dst, name, sx, top-alarmLift*zoom, zoom, render.SpriteOptions{}) {
		size := .8 + .3*Wave(sc.Time*ringPulseHz)
		isoRing(dst, sx, sy, zoom, size, alarmRed)
	}
}
