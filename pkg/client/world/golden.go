package world

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	HitGolden    = "golden"
	gleamStep    = .95
	gleamJog     = .1
	gleamClear   = .75
	gleamFront   = 1.4
	gleamTries   = 12
	ringWide     = 11.0
	ringTall     = 5.5
	ringSteps    = 32
	gleamSize    = gfx.GildedBoxSize
	crownSize    = 0
	crownNudge   = 2
	gleamGlintHz = 1.4
	gleamRing    = 7.0
	gleamUrgent  = 3.0
	gleamAbove   = .6
	gleamInland  = .8
)

var gleamTrack = color.RGBA{40, 30, 10, 200}

type Gleam struct {
	ID    string
	Fades float64
	Span  float64
}

type Gilded struct {
	Harbor string
	Gleams []Gleam
}

func (h *Harbor) planOf(id string) (HavenPlan, bool) {
	for _, p := range h.Plans {
		if p.ID == id && p.Open {
			return p, true
		}
	}
	for _, p := range h.Plans {
		if p.Open {
			return p, true
		}
	}
	return HavenPlan{}, false
}

func gleamCandidate(p HavenPlan, b Berth, k int) Vec {
	side := gleamJog
	if k%2 == 1 {
		side = -gleamJog
	}
	return offset(offset(b.Plank, p.Out, -gleamInland-float64(k)*gleamStep), p.Side, side)
}

func GleamSpots(p HavenPlan, b Berth, n int) []Vec {
	out := []Vec{}
	used := map[int]bool{}
	for k := 0; len(out) < n && k < gleamTries; k++ {
		if at := gleamCandidate(p, b, k); clearOfProps(p, at) {
			out, used[k] = append(out, at), true
		}
	}
	for k := 0; len(out) < n; k++ {
		if !used[k] {
			out = append(out, gleamCandidate(p, b, k))
		}
	}
	return out
}

func clearOfProps(p HavenPlan, at Vec) bool {
	for _, prop := range p.Props {
		d := Dist(prop.At, at)
		front := prop.At.X+prop.At.Y > at.X+at.Y
		if d < gleamClear || front && d < gleamFront {
			return false
		}
	}
	return true
}

func GleamLeft(g Gleam, now float64) float64 {
	return max(g.Fades-now, 0)
}

func GleamShare(g Gleam, now float64) float64 {
	if g.Span <= 0 {
		return 0
	}
	return min(max(GleamLeft(g, now)/g.Span, 0), 1)
}

func (r *Renderer) goldenItems(cam render.Camera, sc Scene) []Drawable {
	h := r.HarborOf()
	h.gleams = map[string]Vec{}
	g := sc.Golden
	if g == nil || len(g.Gleams) == 0 {
		return nil
	}
	p, ok := h.planOf(g.Harbor)
	if !ok || len(p.Berths) == 0 {
		return nil
	}
	b := p.Berths[len(p.Berths)-1]
	out := []Drawable{r.gildedShip(cam, sc, b)}
	spots := GleamSpots(p, b, len(g.Gleams))
	for i, gl := range g.Gleams {
		if GleamLeft(gl, h.Now) <= 0 || i >= len(spots) {
			continue
		}
		at := spots[i]
		h.gleams[gl.ID] = at
		out = append(out, r.gleamItem(cam, sc, gl, at, i))
	}
	return out
}

func (r *Renderer) GleamSpot(id string) (Vec, bool) {
	v, ok := r.HarborOf().gleams[id]
	return v, ok
}

func (r *Renderer) gildedShip(cam render.Camera, sc Scene, b Berth) Drawable {
	zoom := cam.Scale()
	sx, sy := ScreenPointOf(cam, sc.Site, b.Moor)
	pose := ShipPose{Pos: b.Moor, Dir: ShipHeading(Vec{b.Moor.X - b.Lane.X, b.Moor.Y - b.Lane.Y}), Alpha: 1}
	frame := 0
	if !sc.Reduce {
		frame = Frame(sc.Time, 3)
	}
	plain := gfx.ShipName(gfx.GildedClass, gfx.GildedVariant, pose.Dir, frame)
	age := sc.Time
	return Drawable{Depth: render.DepthKey(b.Moor.X, b.Moor.Y), Draw: func(dst *ebiten.Image) {
		if _, ok := r.Lib.DrawFirst(dst, []string{gfx.GildedName(plain), plain}, sx, sy, zoom, render.SpriteOptions{}); !ok {
			drawShipFallback(dst, gfx.GildedClass, gfx.GildedVariant, pose, sx, sy, zoom)
		}
		if !sc.Reduce {
			r.drawCheer(dst, sx, sy, zoom, age)
		}
	}}
}

func (r *Renderer) drawCheer(dst *ebiten.Image, sx, sy, zoom, age float64) {
	for i := 0; i < 3; i++ {
		k := float64(i)
		name := gfx.EffectName("sparkle", int(age*10)+i*2)
		r.Lib.Draw(dst, name, sx+(k-1)*12*zoom, sy-(cheerLift+6*k)*zoom, zoom, render.SpriteOptions{})
	}
}

func (r *Renderer) gleamItem(cam render.Camera, sc Scene, g Gleam, at Vec, i int) Drawable {
	zoom := cam.Scale()
	sx, sy := ScreenPointOf(cam, sc.Site, at)
	sy -= art.HavenDeck * zoom
	box := gfx.BoxName(gleamSize, KindName(0), 0)
	layers := []string{gfx.GildedName(box), box, gfx.BoxCrownName(crownSize)}
	crownY := sy - float64(gfx.BoxLift(gleamSize)-gfx.BoxLift(crownSize)+crownNudge)*zoom
	rect := SpriteRect(r.Lib, layers[0], sx, sy, zoom, 10, 9)
	hit := &Hit{Kind: HitGolden, ID: g.ID, Rect: rect.Inset(-hitPad - 2)}
	share := GleamShare(g, r.HarborOf().Now)
	left := GleamLeft(g, r.HarborOf().Now)
	t := sc.RealTime()
	return Drawable{Depth: render.DepthKey(at.X, at.Y) + gleamAbove, Hit: hit, Draw: func(dst *ebiten.Image) {
		drawFadeRing(dst, sx, sy, zoom, share, left < gleamUrgent)
		r.Lib.DrawFirst(dst, layers[:2], sx, sy, zoom, render.SpriteOptions{})
		r.Lib.Draw(dst, layers[2], sx, crownY, zoom, render.SpriteOptions{})
		if !sc.Reduce {
			glint := gfx.EffectName("sparkle", int(t*gleamGlintHz*6)+i*2)
			r.Lib.Draw(dst, glint, sx+3*zoom, float64(rect.Y)+2*zoom, zoom, render.SpriteOptions{})
		}
	}}
}

func ringPath(cx, cy, rx, ry, from, to float64) *vector.Path {
	var p vector.Path
	steps := max(int(math.Ceil(ringSteps*(to-from)/(2*math.Pi))), 1)
	for i := 0; i <= steps; i++ {
		a := from + (to-from)*float64(i)/float64(steps)
		x, y := float32(cx+rx*math.Cos(a)), float32(cy+ry*math.Sin(a))
		if i == 0 {
			p.MoveTo(x, y)
			continue
		}
		p.LineTo(x, y)
	}
	return &p
}

func drawFadeRing(dst *ebiten.Image, sx, sy, zoom, share float64, urgent bool) {
	rx, ry := ringWide*zoom, ringTall*zoom
	width := &vector.StrokeOptions{Width: float32(max(zoom, 1) + .5)}
	track := &vector.DrawPathOptions{AntiAlias: true}
	track.ColorScale.ScaleWithColor(gleamTrack)
	vector.StrokePath(dst, ringPath(sx, sy, rx, ry, 0, 2*math.Pi), width, track)
	if share <= 0 {
		return
	}
	clr := ui.Palette.GoldBright
	if urgent {
		clr = ui.Palette.RedText
	}
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(clr)
	start := -math.Pi / 2
	vector.StrokePath(dst, ringPath(sx, sy, rx, ry, start, start+2*math.Pi*share), width, op)
}
