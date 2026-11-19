package world

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	SmokeHeavy  = 3
	handMilli   = 1000.0
	wispRate    = 5.0
	wispSpread  = 2.0
	wispAlpha   = .5
	wispLevelUp = .12
	stackRise   = 10.0
	smokeFar    = .75
	hutSmokeCap = 60
	smokeAbove  = .02
)

var (
	keepSteps = [3]float64{.05, .4, .75}
	podSteps  = [3]float64{.03, .25, .6}
	sootTint  = [3]float32{.46, .44, .45}
	wispNames = func() [art.WispFrames]string {
		var out [art.WispFrames]string
		for i := range out {
			out[i] = gfx.WispName(i)
		}
		return out
	}()
	stackName = gfx.AmbientName("chimney", 0)
)

type vent struct {
	dx, dy float64
	stack  bool
}

var hutVents = map[string]vent{
	"1": {-1, -34, false}, "2": {6, -36, false}, "3": {-2, -48, false},
	"freeman": {5, -40, false}, "landed": {5, -47, false}, "vagrant": {-3, -29, false},
}

var keepVents = map[string]vent{
	"keep-tower": {-19, -72, true}, "keep-stone": {-12, -77, true}, "keep-great": {-22, -80, true}, "keep-citadel": {-23, -92, true},
}

func stepLevel(f float64, steps [3]float64) int {
	for i, s := range steps {
		if f < s {
			return i
		}
	}
	return len(steps)
}

func KeepSmoke(n view.NodeMetrics) int {
	if n.CPUAllocatableMilli <= 0 || n.CPUMilli <= 0 {
		return 0
	}
	return stepLevel(n.CPUMilli/float64(n.CPUAllocatableMilli), keepSteps)
}

func PodSmoke(m view.PodMetrics) int {
	if m.Throttled {
		return SmokeHeavy
	}
	if m.CPUMilli <= 0 {
		return 0
	}
	return stepLevel(m.CPUMilli/handMilli, podSteps)
}

func SmokePuffs(level int, reduce bool) int {
	n := min(max(level, 0), SmokeHeavy)
	if reduce && n > 0 {
		return 1
	}
	return n
}

func HutVent(name string) vent {
	parts := strings.Split(name, "/")
	if len(parts) >= 4 {
		if v, ok := hutVents[parts[3]]; ok {
			return v
		}
	}
	if len(parts) >= 2 {
		if v, ok := hutVents[parts[1]]; ok {
			return v
		}
	}
	return hutVents["1"]
}

func KeepVent(s view.Structure) vent {
	if v, ok := keepVents[gfx.ArtKind(s)]; ok {
		return v
	}
	return keepVents["keep-stone"]
}

func (r *Renderer) smokeItems(out []Drawable, cam render.Camera, sc Scene) []Drawable {
	out = r.keepSmoke(out, cam, sc)
	return r.hutSmoke(out, cam, sc)
}

func (r *Renderer) keepSmoke(out []Drawable, cam render.Camera, sc Scene) []Drawable {
	zoom := cam.Scale()
	for _, k := range Keeps(sc.Site) {
		if k.State != view.StateReady {
			continue
		}
		a := Anchor(k)
		sx, sy := ScreenPointOf(cam, sc.Site, Vec{a.X - .01, a.Y - .01})
		if !onScreen(sx, sy-60*zoom, zoom) {
			continue
		}
		out = append(out, r.plume(cam, sc, r.keepVent(k), sx, sy, render.DepthKey(a.X, a.Y)+smokeAbove, KeepSmoke(sc.Metrics.Nodes[k.Object.Name]), false))
	}
	return out
}

func (r *Renderer) hutSmoke(out []Drawable, cam render.Camera, sc Scene) []Drawable {
	zoom := cam.Scale()
	drawn := 0
	c := r.lifeOf().people.of(sc)
	for i, p := range sc.Pods {
		if drawn >= hutSmokeCap || p.System || p.Terminating || p.State != model.HearthReady {
			continue
		}
		m := c.stats[i].m
		level := PodSmoke(m)
		if level == 0 || (zoom < smokeFar && !m.Throttled) {
			continue
		}
		home, ok := r.Homes[p.Name]
		if !ok {
			continue
		}
		a := Vec{home.X + .5, home.Y + .5}
		sx, sy := ScreenPointOf(cam, sc.Site, Vec{a.X - .01, a.Y - .01})
		if !onScreen(sx, sy-30*zoom, zoom) {
			continue
		}
		out = append(out, r.plume(cam, sc, r.lifeOf().hutVent(r.hutName(p)), sx, sy, render.DepthKey(a.X, a.Y)+smokeAbove, level, m.Throttled))
		drawn++
	}
	return out
}

func (r *Renderer) plume(cam render.Camera, sc Scene, v vent, sx, sy, depth float64, level int, soot bool) Drawable {
	zoom := cam.Scale()
	vx, vy := sx+v.dx*zoom, sy+v.dy*zoom
	top := vy
	if v.stack {
		top -= stackRise * zoom
	}
	puffs := SmokePuffs(level, sc.Reduce)
	opts := render.SpriteOptions{Alpha: float32(wispAlpha + wispLevelUp*float64(level))}
	if soot {
		opts.Tint = sootTint
	}
	tick := int(sc.Time * wispRate)
	if sc.Reduce {
		tick = 2
	}
	return Drawable{Depth: depth, Draw: func(dst *ebiten.Image) {
		if v.stack {
			r.Lib.Draw(dst, stackName, vx, vy, zoom, render.SpriteOptions{})
		}
		for k := 0; k < puffs; k++ {
			x := vx + (float64(k)-float64(puffs-1)/2)*wispSpread*zoom
			r.Lib.Draw(dst, wispNames[(tick+k*2)%art.WispFrames], x, top, zoom, opts)
		}
	}}
}
