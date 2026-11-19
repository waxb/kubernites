package world

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	floatRise   = 12.0
	floatLife   = 0.8
	shoreRadius = 30
)

var juiceKinds = map[string]bool{"sparkle": true, "reweave": true, "puff": true, "horn": true, QoSUp: true, QoSDown: true}

var effectFrames = map[string]int{"puff": 4, "sparkle": 6, "reweave": 6, "dust": 6, "void": 6, "horn": 4, QoSUp: 6, QoSDown: 6}

var effectArt = map[string]string{"puff": "dust", QoSUp: "sparkle", QoSDown: "dust"}

func NearestShore(site view.Site, from Vec) Vec {
	cx, cy := int(from.X), int(from.Y)
	for radius := 1; radius <= shoreRadius; radius++ {
		best, found := Vec{}, false
		bestD := 0.0
		for y := cy - radius; y <= cy+radius; y++ {
			for x := cx - radius; x <= cx+radius; x++ {
				if !ShoreTile(site, x, y) {
					continue
				}
				v := Vec{float64(x) + .5, float64(y) + .5}
				if d := Dist(v, from); !found || d < bestD {
					best, bestD, found = v, d, true
				}
			}
		}
		if found {
			return best
		}
	}
	return Vec{from.X + 4, from.Y + 2}
}

func ShoreTile(site view.Site, x, y int) bool {
	if x < 0 || y < 0 || x >= site.Width || y >= site.Height || Wet(site.Tile(x, y)) {
		return false
	}
	return ShoreMask(site, x, y) != 0
}

func (r *Renderer) ShoreOf(site view.Site, pod string, home Vec) Vec {
	if v, ok := r.shores[pod]; ok {
		return v
	}
	v := NearestShore(site, home)
	r.shores = map[string]Vec{pod: v}
	return v
}

func (r *Renderer) omenPose(sc Scene) (Pose, Vec, bool) {
	o := sc.Omen
	if o == nil {
		return Pose{}, Vec{}, false
	}
	home, ok := r.Homes[o.Pod]
	if !ok {
		return Pose{}, Vec{}, false
	}
	target := Vec{home.X + .5, home.Y + .9}
	from := r.ShoreOf(sc.Site, o.Pod, home)
	span := max(o.Arrive-o.Start, .1)
	progress := min(max((sc.Time-o.Start)/span, 0), 1)
	pose := Pose{Pos: Lerp(from, target, progress), Action: "walk", Dir: Facing(from, target), Frame: Frame(sc.Time, 10)}
	if progress >= 1 {
		pose.Action, pose.Frame = "work", Frame(sc.Time, 8)
	}
	return pose, home, true
}

func Blink(t float64) bool {
	return int(math.Floor(t*4+1e-6))%2 == 0
}

func (r *Renderer) OmenItems(cam render.Camera, sc Scene) []Drawable {
	pose, _, ok := r.omenPose(sc)
	if !ok {
		return nil
	}
	return []Drawable{r.walkerDrawable(cam, sc, "imp", pose, nil, false)}
}

func (r *Renderer) DrawOmenMark(dst *ebiten.Image, cam render.Camera, sc Scene) {
	_, home, ok := r.omenPose(sc)
	if !ok || !(sc.Reduce || Blink(sc.Time)) {
		return
	}
	sx, sy := ScreenPointOf(cam, sc.Site, Vec{home.X + .5, home.Y + .5})
	drawMark(dst, sx, sy-36*cam.Scale())
}

func (r *Renderer) DrawOmenXray(dst *ebiten.Image, cam render.Camera, sc Scene) {
	pose, _, ok := r.omenPose(sc)
	if !ok {
		return
	}
	sx, sy := ScreenPointOf(cam, sc.Site, pose.Pos)
	names := gfx.WalkerNames("imp", pose.Action, pose.Dir, pose.Frame)
	r.Lib.DrawFirst(dst, names, sx, sy, cam.Scale(), render.SpriteOptions{Alpha: .45})
}

func drawMark(dst *ebiten.Image, sx, sy float64) {
	var font ui.Font
	box := ui.Rect{X: int(sx) - 6, Y: int(sy) - 18, W: 12, H: 18}
	ui.Fill(dst, box.Offset(1, 1), ui.Palette.Shadow)
	ui.Fill(dst, box, ui.Palette.Ink)
	ui.Stroke(dst, box, ui.Palette.RedText)
	font.Draw(dst, "!", box.X+1, box.Y+2, 2, ui.Palette.RedText)
}

func (r *Renderer) TransientItems(cam render.Camera, sc Scene) []Drawable {
	out := []Drawable{}
	for _, t := range sc.Transients {
		age := sc.Time - t.Start
		if age < 0 || age > t.Life || (sc.Reduce && juiceKinds[t.Kind]) {
			continue
		}
		at := t.At
		if t.Pod != "" {
			home, ok := r.Homes[t.Pod]
			if !ok {
				continue
			}
			at = Vec{home.X + .5, home.Y + .5}
		}
		out = append(out, r.transientDrawable(cam, sc, t, at, age))
	}
	return out
}

func (r *Renderer) transientDrawable(cam render.Camera, sc Scene, t Transient, at Vec, age float64) Drawable {
	sx, sy := ScreenPointOf(cam, sc.Site, at)
	z := cam.Scale()
	depth := render.DepthKey(at.X, at.Y) + .3
	frames := effectFrames[t.Kind]
	if frames == 0 {
		frames = 6
	}
	kind := t.Kind
	if art, ok := effectArt[kind]; ok {
		kind = art
	}
	name := gfx.EffectName(kind, min(int(age/t.Life*float64(frames)), frames-1))
	progress := age / max(t.Life, 1e-9)
	return Drawable{Depth: depth, Draw: func(dst *ebiten.Image) {
		r.Lib.Draw(dst, name, sx, sy, z, render.SpriteOptions{})
		drawShift(dst, t.Kind, sx, sy, z, progress)
	}}
}

func drawShift(dst *ebiten.Image, kind string, sx, sy, zoom, progress float64) {
	px := max(int(zoom), 1)
	switch kind {
	case QoSUp:
		drawChevron(dst, int(sx), int(sy-(18+qosRise*progress)*zoom), px, true, rankHigh)
	case QoSDown:
		drawChevron(dst, int(sx), int(sy-(28-qosRise*progress)*zoom), px, false, rankLow)
	}
}

func drawFloat(dst *ebiten.Image, text string, sx, sy, progress float64, scale int) {
	var font ui.Font
	alpha := uint8(255 * min(max(1-progress, 0), 1))
	w, _ := font.Measure(text, scale)
	x, y := int(sx)-w/2, int(sy-floatRise*progress*float64(scale))
	font.Draw(dst, text, x+scale, y+scale, scale, ui.Tint(color.RGBA{20, 25, 27, 255}, alpha))
	font.Draw(dst, text, x, y, scale, ui.Tint(ui.Palette.GoldBright, alpha))
}

func (r *Renderer) DeliveryItem(cam render.Camera, sc Scene, age float64, bank Vec) (Drawable, bool) {
	if sc.Reduce || age >= floatLife {
		return Drawable{}, false
	}
	sx, sy := ScreenPointOf(cam, sc.Site, bank)
	z := cam.Scale()
	return Drawable{Depth: render.DepthKey(bank.X, bank.Y) + .5, Draw: func(dst *ebiten.Image) {
		drawFloat(dst, "+1", sx, sy-26*z, age/floatLife, 1)
	}}, true
}
