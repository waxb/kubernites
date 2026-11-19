package world

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	fanShow     = 8.0
	fanFlash    = 3.0
	fanDash     = 4.0
	fanAlpha    = 255
	fanWidth    = 2.0
	fanTail     = 2.0
	fanFlow     = 12.0
	maxFanLines = 24
	doorAside   = .55
	doorFront   = .95
)

var (
	fanColor = color.RGBA{255, 222, 120, 255}
	fanShade = color.RGBA{40, 28, 18, 255}
)

type fanTarget struct {
	door   Vec
	weight float64
	name   string
}

type fanWatch struct {
	primed   bool
	answered bool
	since    float64
	joined   map[string]float64
}

func hutDoor(home Vec) Vec {
	return Vec{home.X + doorAside, home.Y + doorFront}
}

func (r *Renderer) fanTargets(sc Scene) []fanTarget {
	if _, ok := FirstOfKind(sc.Site, view.StructGuildhall, true); !ok {
		return nil
	}
	l := r.lifeOf()
	c := l.people.of(sc)
	if l.fanAt == sc.Time && l.fanFor == c.base && len(l.fanBuf) > 0 {
		return l.fanBuf
	}
	out := l.fanBuf[:0]
	for i, p := range sc.Pods {
		if !declaredHearth(p) {
			continue
		}
		rate := c.stats[i].m.ServedRate
		home, ok := r.Homes[p.Name]
		if ok && rate > 0 {
			out = append(out, fanTarget{door: hutDoor(home), weight: rate, name: p.Name})
		}
	}
	l.fanBuf, l.fanAt, l.fanFor = out, sc.Time, c.base
	return out
}

func (w *fanWatch) observe(now float64, members map[string]bool) {
	if !w.primed {
		*w = fanWatch{primed: true, answered: len(members) > 0, since: math.Inf(-1), joined: map[string]float64{}}
		for m := range members {
			w.joined[m] = math.Inf(-1)
		}
		return
	}
	if !w.answered && len(members) > 0 {
		w.answered, w.since = true, now
	}
	for m := range members {
		if _, ok := w.joined[m]; !ok {
			w.joined[m] = now
		}
	}
	for m := range w.joined {
		if !members[m] {
			delete(w.joined, m)
		}
	}
}

func fade(age, span float64) float64 {
	if age < 0 || age >= span {
		return 0
	}
	return min((span-age)/fanTail, 1)
}

func (w *fanWatch) alpha(name string, now float64) float64 {
	a := fade(now-w.since, fanShow)
	if at, ok := w.joined[name]; ok {
		a = max(a, fade(now-at, fanFlash))
	}
	return a
}

func (w *fanWatch) visible(name string, now float64) bool {
	return w.alpha(name, now) > 0
}

func (r *Renderer) watchFan(sc Scene) {
	if len(r.Homes) == 0 {
		return
	}
	fan := r.fanTargets(sc)
	members := make(map[string]bool, len(fan))
	for _, f := range fan {
		members[f.name] = true
	}
	l := r.lifeOf()
	l.fan.observe(sc.Time, members)
	l.fanNow = fan
}

func (r *Renderer) drawFanLines(dst *ebiten.Image, cam render.Camera, sc Scene) {
	gh, ok := FirstOfKind(sc.Site, view.StructGuildhall, true)
	if !ok {
		return
	}
	w := &r.lifeOf().fan
	fx, fy := ScreenPointOf(cam, sc.Site, Porch(gh))
	shift := 0.0
	if !sc.Reduce {
		shift = math.Mod(sc.Time*fanFlow, 2*fanDash)
	}
	var lit, shade vector.Path
	best := 0.0
	for i, f := range r.lifeOf().fanNow {
		a := w.alpha(f.name, sc.Time)
		tx, ty := ScreenPointOf(cam, sc.Site, f.door)
		if a <= 0 || i >= maxFanLines || !onScreen(tx, ty, cam.Scale()) {
			continue
		}
		best = max(best, a)
		dashes(&lit, fx, fy, tx, ty, shift)
		dashes(&shade, fx, fy+1, tx, ty+1, shift)
	}
	if best > 0 {
		width := float32(max(cam.Scale(), 1) * fanWidth)
		strokeTinted(dst, &shade, width+1, ui.Tint(fanShade, uint8(fanAlpha*best)))
		strokeTinted(dst, &lit, width, ui.Tint(fanColor, uint8(fanAlpha*best)))
	}
}

func dashes(p *vector.Path, x0, y0, x1, y1, shift float64) {
	length := math.Hypot(x1-x0, y1-y0)
	if length < 1 {
		return
	}
	ux, uy := (x1-x0)/length, (y1-y0)/length
	for d := shift - 2*fanDash; d < length; d += 2 * fanDash {
		a, e := max(d, 0), min(d+fanDash, length)
		if e > a {
			p.MoveTo(float32(x0+ux*a), float32(y0+uy*a))
			p.LineTo(float32(x0+ux*e), float32(y0+uy*e))
		}
	}
}

func strokeTinted(dst *ebiten.Image, p *vector.Path, width float32, c color.RGBA) {
	op := &vector.DrawPathOptions{}
	op.ColorScale.ScaleWithColor(c)
	vector.StrokePath(dst, p, &vector.StrokeOptions{Width: width}, op)
}
