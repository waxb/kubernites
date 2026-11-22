package world

import (
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

var (
	selectColor     = color.RGBA{255, 218, 105, 255}
	selectEdge      = color.RGBA{255, 218, 105, 110}
	selectLine      = color.RGBA{255, 218, 105, 150}
	ghostGood       = color.RGBA{90, 200, 90, 110}
	ghostBad        = color.RGBA{220, 70, 60, 110}
	placeGhostAlpha = float32(.6)
	nightColor      = color.RGBA{18, 24, 60, 255}
)

func (r *Renderer) drawSelectionGround(dst *ebiten.Image, cam render.Camera, sc Scene) {
	if sc.Selected == "" {
		return
	}
	for _, s := range sc.Site.Structures {
		if s.ID == sc.Selected {
			lo, hi := Footprint(s)
			elev := float64(sc.Site.ElevationAt(hi.X, hi.Y))
			outlineDiamond(dst, cam, float64(lo.X), float64(lo.Y), float64(hi.X), float64(hi.Y), elev, selectEdge)
		}
	}
	for name, home := range r.Homes {
		if selectedPod(name, sc.Selected) {
			outlineDiamond(dst, cam, home.X, home.Y, home.X, home.Y, elevationAt(sc.Site, home), selectEdge)
		}
	}
}

func selectedPod(name, selected string) bool {
	return "default/"+name == selected || name == selected
}

func (r *Renderer) outlineSprite(dst *ebiten.Image, names []string, sx, sy, zoom float64) {
	for _, n := range names {
		if sheet, reg, ok := r.Lib.Lookup(n); ok {
			render.OutlineAt(dst, sheet, reg, sx, sy, zoom, selectLine)
			return
		}
	}
}

func (r *Renderer) drawOverlays(dst *ebiten.Image, cam render.Camera, sc Scene) {
	r.placed = r.placed[:0]
	r.DrawSpots(dst, cam, sc)
	if g := sc.Ghost; g != nil {
		r.drawGhost(dst, cam, sc, *g)
	}
	r.drawBestSpot(dst, cam, sc)
	r.drawQuayOverlay(dst, cam, sc)
	r.drawTradeLabels(dst, cam, sc)
	if front, lift, ok := r.HarborOf().front(); ok && sc.Hint {
		zoom := cam.Scale()
		sx, sy := ScreenPointOf(cam, sc.Site, front)
		r.drawHint(dst, sx, sy-lift*zoom, zoom, sc)
	}
	r.DrawBeacons(dst, cam, sc)
	for _, k := range r.keepPips(cam, sc) {
		r.placed = append(r.placed, k.plaque())
	}
	for _, b := range r.beaconSpots(cam, sc) {
		r.placed = append(r.placed, beaconArrow(b.sx, b.sy, b.zoom))
	}
	r.drawLabels(dst, cam, sc)
	r.drawPlaques(dst, cam, sc)
	r.drawSignpostLabel(dst, cam, sc)
	r.drawGateLabels(dst, cam, sc)
}

func GhostStructure(site view.Site, g Ghost) view.Structure {
	s := view.Structure{Kind: g.Kind, X: g.X, Y: g.Y, W: max(g.W, 1), H: max(g.H, 1), State: view.StateReady}
	if face, ok := site.StallFace(s.X, s.Y, s.W, s.H); ok && view.StallKind(g.Kind) {
		s.Variant = face
	}
	if !IsKeepKind(g.Kind) {
		return s
	}
	s.Kind, s.Variant, s.Class = view.StructKeep, "tower", g.Class
	if _, ok := FirstOfKind(site, view.StructKeep, false); ok || g.Kind != view.StructKeep {
		s.Variant = "stone"
		if k, ok := view.KeepTypeOf(g.Kind); ok {
			s.Variant = k.Variant
		}
	}
	s.Object.Name = gfx.KeepPreviewNode(s.Variant)
	return s
}

func IsKeepKind(kind string) bool {
	_, ok := view.KeepTypeOf(kind)
	return kind == view.StructKeep || (ok && strings.HasSuffix(kind, "-keep"))
}

func visibleTiles(dst *ebiten.Image, cam render.Camera, site view.Site) (int, int, int, int) {
	b := dst.Bounds()
	x0, y0, x1, y1 := site.Width, site.Height, -1, -1
	for _, c := range [][2]int{{b.Min.X, b.Min.Y}, {b.Max.X, b.Min.Y}, {b.Min.X, b.Max.Y}, {b.Max.X, b.Max.Y}} {
		gx, gy := cam.ScreenToGrid(float64(c[0]), float64(c[1]))
		x0, y0 = min(x0, int(math.Floor(gx))-1), min(y0, int(math.Floor(gy))-1)
		x1, y1 = max(x1, int(math.Ceil(gx))+1), max(y1, int(math.Ceil(gy))+1)
	}
	return max(x0, 0), max(y0, 0), min(x1, site.Width-1), min(y1, site.Height-1)
}

func (r *Renderer) drawGhost(dst *ebiten.Image, cam render.Camera, sc Scene, g Ghost) {
	site := sc.Site
	c, tint := ghostBad, [3]float32{1.2, .55, .5}
	if g.Valid {
		c, tint = ghostGood, [3]float32{.7, 1.15, .7}
	}
	s := GhostStructure(site, g)
	lo, hi := Footprint(s)
	fillFootprint(dst, cam, lo.X, lo.Y, hi.X, hi.Y, c)
	a := Anchor(s)
	sx, sy := ScreenPointOf(cam, site, Vec{a.X - .01, a.Y - .01})
	if name, ok := r.Lib.DrawFirst(dst, gfx.PreviewNames(s, sc.Age), sx, sy, cam.Scale(), render.SpriteOptions{Alpha: placeGhostAlpha, Tint: tint}); ok {
		r.drawFlags(dst, name, s, sx, sy, cam.Scale(), sc.Time, placeGhostAlpha)
	}
	outlineDiamond(dst, cam, float64(lo.X), float64(lo.Y), float64(hi.X), float64(hi.Y), 0, c)
}

func (r *Renderer) drawLabels(dst *ebiten.Image, cam render.Camera, sc Scene) {
	var font ui.Font
	for _, s := range sc.Site.Structures {
		if _, plaque := sc.Plaques[s.ID]; plaque || s.ID != sc.Selected && s.State != view.StateConstruction {
			continue
		}
		a := Anchor(s)
		sx, sy := ScreenPointOf(cam, sc.Site, a)
		label := s.Label
		if s.State == view.StateConstruction {
			label = s.Label + " " + Itoa(int(s.Progress*100)) + "%"
		}
		w, _ := font.Measure(label, 1)
		box, clear := r.settleStructureLabel(cam, s, sx, sy, w+6, labelH, sc.Panels)
		if !clear {
			continue
		}
		ui.Fill(dst, box, ui.Tint(ui.Palette.ConsoleEdge, 200))
		font.Draw(dst, label, box.X+3, box.Y+2, 1, ui.Palette.GoldBright)
	}
}

const (
	labelH  = 11
	plaqueH = 22
)

func (r *Renderer) drawnSpan(cam render.Camera, s view.Structure, sy float64) (top, bottom int, below bool) {
	if hit, ok := r.HitOf(HitStructure, s.ID); ok && len(hit.Shapes) > 0 {
		sh := hit.Shapes[0]
		if up, down, ok := r.Lib.Extent(sh.Name); ok {
			return int(math.Round(sh.Y - float64(up)*sh.Zoom)), int(math.Round(sh.Y + float64(down)*sh.Zoom)), true
		}
	}
	return int(sy-float64(s.H*18)*cam.Scale()) - 13 + labelGap, 0, false
}

func (r *Renderer) settleStructureLabel(cam render.Camera, s view.Structure, sx, sy float64, w, h int, panels []ui.Rect) (ui.Rect, bool) {
	top, bottom, below := r.drawnSpan(cam, s, sy)
	box := ui.Rect{X: int(sx) - w/2, Y: top - labelGap - h, W: w, H: h}
	if got, ok := r.settleLabelToward(box, int(sx), panels, -1); ok || !below {
		return got, ok
	}
	box.Y = bottom + labelGap
	return r.settleLabelToward(box, int(sx), panels, 1)
}

func (r *Renderer) drawPlaques(dst *ebiten.Image, cam render.Camera, sc Scene) {
	var font ui.Font
	literal := ui.Font{Literal: true}
	for _, s := range sc.Site.Structures {
		p, ok := sc.Plaques[s.ID]
		if !ok {
			continue
		}
		sx, sy := ScreenPointOf(cam, sc.Site, Anchor(s))
		w1, _ := font.Measure(p.Title, 1)
		w2, _ := literal.Measure(p.Line, 1)
		w := max(w1, w2)
		box, clear := r.settleStructureLabel(cam, s, sx, sy, w+6, plaqueH, sc.Panels)
		if !clear {
			continue
		}
		ui.Fill(dst, box, ui.Tint(ui.Palette.ConsoleEdge, 210))
		if s.ID == sc.Selected {
			ui.Stroke(dst, box, ui.Palette.Gold)
		}
		font.Draw(dst, p.Title, box.X+3+(w-w1)/2, box.Y+2, 1, ui.Palette.GoldBright)
		literal.Draw(dst, p.Line, box.X+3+(w-w2)/2, box.Y+12, 1, ui.Palette.Cream)
	}
}

func Itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	digits := []byte{}
	for v > 0 {
		digits = append([]byte{byte('0' + v%10)}, digits...)
		v /= 10
	}
	if neg {
		return "-" + string(digits)
	}
	return string(digits)
}

func nightFill(phase float64) color.RGBA {
	dark := Darkness(phase)
	scale := func(v uint8) uint8 { return uint8(math.Round(float64(v) * dark / nightDepth)) }
	return color.RGBA{scale(nightColor.R), scale(nightColor.G), scale(nightColor.B), uint8(dark * 255)}
}

func drawNight(dst *ebiten.Image, phase float64) {
	c := nightFill(phase)
	if c.A <= 2 {
		return
	}
	b := dst.Bounds()
	vector.FillRect(dst, float32(b.Min.X), float32(b.Min.Y), float32(b.Dx()), float32(b.Dy()), c, false)
}

const glowSize = 64

func glowImage() *ebiten.Image {
	img := image.NewRGBA(image.Rect(0, 0, glowSize, glowSize))
	c := float64(glowSize-1) / 2
	for y := 0; y < glowSize; y++ {
		for x := 0; x < glowSize; x++ {
			d := math.Hypot(float64(x)-c, float64(y)-c) / c
			a := math.Pow(max(1-d, 0), 2)
			v := uint8(a * 255)
			img.SetRGBA(x, y, color.RGBA{v, uint8(float64(v) * .78), uint8(float64(v) * .45), v})
		}
	}
	return ebiten.NewImageFromImage(img)
}

func (r *Renderer) drawGlow(dst *ebiten.Image, sx, sy, radius, strength float64) {
	if r.glow == nil {
		r.glow = glowImage()
	}
	op := &ebiten.DrawImageOptions{Blend: ebiten.BlendLighter}
	scale := radius * 2 / glowSize
	op.GeoM.Scale(scale, scale*.7)
	op.GeoM.Translate(sx-radius, sy-radius*.7)
	op.ColorScale.ScaleAlpha(float32(strength))
	op.ColorScale.Scale(float32(strength), float32(strength), float32(strength), 1)
	dst.DrawImage(r.glow, op)
}

func (r *Renderer) DrawLamps(dst *ebiten.Image, cam render.Camera, sc Scene) {
	dark := Darkness(sc.DayPhase)
	if dark < .15 {
		return
	}
	strength := min(dark/.55, 1) * .8
	z := cam.Scale()
	for _, p := range sc.Pods {
		home, ok := r.Homes[p.Name]
		if !ok || p.System || p.State != model.HearthReady {
			continue
		}
		sx, sy := ScreenPointOf(cam, sc.Site, Vec{home.X + .5, home.Y + .5})
		r.drawGlow(dst, sx, sy-6*z, 22*z, strength)
	}
	for _, s := range sc.Site.Structures {
		if s.State != view.StateReady {
			continue
		}
		sx, sy := ScreenPointOf(cam, sc.Site, Door(s))
		r.drawGlow(dst, sx, sy-4*z, 30*z, strength)
	}
}

func fallbackBlock(dst *ebiten.Image, cam render.Camera, site view.Site, s view.Structure) {
	lo, hi := Footprint(s)
	c := color.RGBA{150, 140, 128, 255}
	if s.State == view.StateConstruction {
		c = color.RGBA{190, 160, 110, 200}
	}
	fillFootprint(dst, cam, lo.X, lo.Y, hi.X, hi.Y, c)
	a := Anchor(s)
	sx, sy := ScreenPointOf(cam, site, a)
	h := float32(float64(10+6*s.W) * cam.Scale())
	w := float32(float64(s.W*12) * cam.Scale())
	vector.FillRect(dst, float32(sx)-w/2, float32(sy)-h-float32(8*cam.Scale()), w, h, c, false)
}
