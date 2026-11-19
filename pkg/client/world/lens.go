package world

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	LensLabor    = "labor"
	LensUnhoused = "unhoused"
	LensMidden   = "midden"
	LensCodex    = "codex"
	minPlot      = 3
	labelDrop    = 6
	unhousedRing = .7
	ringPulseHz  = 2.0
)

type Lens struct {
	Mode   string
	Heat   map[string]float64
	Labels map[string]string
}

var heatSteps = []float64{.25, .5, .75, .9}

var heatTints = []color.NRGBA{
	{70, 170, 80, 70}, {150, 190, 70, 80}, {220, 180, 60, 90}, {230, 120, 50, 105}, {215, 55, 45, 120},
}

func HeatLevel(v float64) int {
	level := 0
	for _, s := range heatSteps {
		if v >= s {
			level++
		}
	}
	return level
}

func HeatTint(v float64) color.NRGBA { return heatTints[HeatLevel(v)] }

func HeatEdge(v float64) color.RGBA {
	c := HeatTint(v)
	return color.RGBA{c.R, c.G, c.B, 255}
}

func Wave(t float64) float64 { return .5 + .5*math.Sin(2*math.Pi*t) }

func LensArea(s view.Structure) (Point, Point) {
	lo, hi := Footprint(s)
	pad := 1
	if s.Kind == view.StructKeep {
		pad = max(s.Plot, minPlot)
	}
	return Point{lo.X - pad, lo.Y - pad}, Point{hi.X + pad, hi.Y + pad}
}

type tileSpan struct{ x0, y0, x1, y1 int }

func clipSpan(lo, hi Point, x0, y0, x1, y1 int) (tileSpan, bool) {
	s := tileSpan{max(lo.X, x0), max(lo.Y, y0), min(hi.X, x1), min(hi.Y, y1)}
	return s, s.x0 <= s.x1 && s.y0 <= s.y1
}

func (r *Renderer) drawLens(dst *ebiten.Image, cam render.Camera, sc Scene) {
	if sc.Lens.Mode == LensHeraldry && sc.Ghost == nil {
		r.drawHeraldGround(dst, cam, sc)
		return
	}
	if sc.Lens.Mode == "" || sc.Ghost != nil || len(sc.Lens.Heat) == 0 {
		return
	}
	paths := map[color.NRGBA]*vector.Path{}
	tints := []color.NRGBA{}
	x0, y0, x1, y1 := visibleTiles(dst, cam, sc.Site)
	for _, s := range sc.Site.Structures {
		heat, ok := sc.Lens.Heat[s.ID]
		if !ok {
			continue
		}
		lo, hi := LensArea(s)
		span, ok := clipSpan(lo, hi, x0, y0, x1, y1)
		if !ok {
			continue
		}
		c := HeatTint(heat)
		p, ok := paths[c]
		if !ok {
			p = &vector.Path{}
			paths[c] = p
			tints = append(tints, c)
		}
		addTiles(p, cam, sc.Site, span)
	}
	for _, c := range tints {
		op := &vector.DrawPathOptions{}
		op.ColorScale.ScaleWithColor(c)
		vector.FillPath(dst, paths[c], &vector.FillOptions{}, op)
	}
}

func addTiles(p *vector.Path, cam render.Camera, site view.Site, span tileSpan) {
	for y := span.y0; y <= span.y1; y++ {
		for x := span.x0; x <= span.x1; x++ {
			addTile(p, cam, site, x, y)
		}
	}
}

func addTile(p *vector.Path, cam render.Camera, site view.Site, x, y int) {
	if Wet(site.Tile(x, y)) {
		return
	}
	hw, hh := float32(render.TileWidth/2*cam.Scale()), float32(render.TileHeight/2*cam.Scale())
	sx, sy := cam.GridToScreenElevated(float64(x), float64(y), float64(site.ElevationAt(x, y)))
	cx, cy := float32(sx), float32(sy)
	p.MoveTo(cx, cy-hh)
	p.LineTo(cx+hw, cy)
	p.LineTo(cx, cy+hh)
	p.LineTo(cx-hw, cy)
	p.Close()
}

func (r *Renderer) DrawLensMarks(dst *ebiten.Image, cam render.Camera, sc Scene) {
	if sc.Lens.Mode == "" || sc.Ghost != nil {
		return
	}
	if sc.Lens.Mode == LensUnhoused {
		r.drawUnhousedRings(dst, cam, sc)
	}
	if sc.Lens.Mode == LensHeraldry {
		r.drawPennants(dst, cam, sc)
		return
	}
	var font ui.Font
	for _, s := range sc.Site.Structures {
		text, ok := sc.Lens.Labels[s.ID]
		if !ok {
			continue
		}
		sx, sy := ScreenPointOf(cam, sc.Site, Anchor(s))
		if !onScreen(sx, sy, cam.Scale()) {
			continue
		}
		w, _ := font.Measure(text, 1)
		box := lensLabelBox(sx, sy, w)
		ui.Fill(dst, box.Offset(1, 1), ui.Palette.Shadow)
		ui.Fill(dst, box, ui.Tint(ui.Palette.ConsoleEdge, 225))
		ui.Stroke(dst, box, HeatEdge(sc.Lens.Heat[s.ID]))
		font.Draw(dst, text, box.X+3, box.Y+2, 1, ui.Palette.Cream)
	}
}

func lensLabelBox(sx, sy float64, w int) ui.Rect {
	return keepInView(ui.Rect{X: int(sx) - w/2 - 3, Y: int(sy) + labelDrop, W: w + 6, H: 11})
}

func (r *Renderer) drawUnhousedRings(dst *ebiten.Image, cam render.Camera, sc Scene) {
	pulse := 1.0
	if !sc.Reduce {
		pulse = .6 + .4*Wave(sc.Time*ringPulseHz)
	}
	c := color.RGBA{240, 90, 70, uint8(255 * pulse)}
	for _, p := range sc.Pods {
		if p.System || p.State != model.HearthPending {
			continue
		}
		home, ok := r.Homes[p.Name]
		if !ok {
			continue
		}
		sx, sy := ScreenPointOf(cam, sc.Site, Vec{home.X + .5, home.Y + .5})
		isoRing(dst, sx, sy, cam.Scale(), unhousedRing, c)
	}
}
