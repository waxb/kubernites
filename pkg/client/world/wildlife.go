package world

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	skyDepth    = 1e12
	restingDark = 0.35
	fishLeap    = 1.1
	gullOdds    = 260
	finchOdds   = 1100
	fishOdds    = 14
)

type flock struct {
	role   string
	center Vec
	radius float64
	omega  float64
	alt    float64
	count  int
	seed   uint32
}

type fishSpot struct {
	pos    Vec
	period float64
	phase  float64
}

func coastal(site view.Site, x, y, r int) bool {
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if !Wet(site.Tile(x+dx, y+dy)) {
				return true
			}
		}
	}
	return false
}

func flockFor(site view.Site, x, y int) (flock, bool) {
	h := mix32(tileHash(x, y) ^ 0x51ed270b)
	k := site.Tile(x, y)
	f := flock{center: Vec{float64(x), float64(y)}, seed: h, omega: 0.25 + float64(h%30)/100, count: 3 + int(h>>8%3)}
	if h>>4%2 == 1 {
		f.omega = -f.omega
	}
	switch {
	case Wet(k) && h>>12%gullOdds == 0 && coastal(site, x, y, 3):
		f.role, f.radius, f.alt = "gull", 2.5+float64(h>>20%20)/10, 26
	case (k == view.TileGrass || k == view.TileForest || k == view.TileMeadow) && h>>12%finchOdds == 1:
		f.role, f.radius, f.alt, f.count = "finch", 3.5+float64(h>>20%30)/10, 34, f.count+2
	default:
		return flock{}, false
	}
	return f, true
}

func fishFor(site view.Site, x, y int) (fishSpot, bool) {
	h := mix32(tileHash(x, y) ^ 0x2545f491)
	if !Wet(site.Tile(x, y)) || !landBeside(site, x, y) || h%fishOdds != 0 {
		return fishSpot{}, false
	}
	return fishSpot{pos: Vec{float64(x) + jitterAt(h, 3, 0.3), float64(y) + jitterAt(h, 13, 0.3)}, period: 7 + float64(h>>8%70)/10, phase: float64(h>>16%100) / 10}, true
}

type wing struct {
	pos    Vec
	alt    float64
	action string
	dir    string
	frame  int
}

func (f flock) member(t float64, m int) wing {
	turn := sign(f.omega)
	ang := f.omega*t + float64(f.seed%628)/100 + float64(m)*0.32*turn
	r := f.radius * (0.85 + 0.15*math.Sin(float64(m)*1.7))
	pos := Vec{f.center.X + math.Cos(ang)*r, f.center.Y + math.Sin(ang)*r}
	ahead := Vec{f.center.X + math.Cos(ang+0.1*turn)*r, f.center.Y + math.Sin(ang+0.1*turn)*r}
	w := wing{pos: pos, alt: f.alt + 3*math.Sin(t*1.3+float64(m)), action: "walk", dir: Facing(pos, ahead), frame: Frame(t+float64(m)*0.23, 8)}
	if math.Sin(t*0.7+float64(m)*2) > 0.3 {
		w.action = "carry"
	}
	return w
}

func sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	return 1
}

func (r *Renderer) wildItems(cam render.Camera, sc Scene) []Drawable {
	lc := r.Land
	if lc == nil || lc.land == nil {
		return nil
	}
	resting := Darkness(sc.DayPhase) > restingDark
	out := []Drawable{}
	for _, k := range lc.visible(cam, lc.view) {
		w := lc.wildAt(sc.Site, k)
		for _, a := range w.beasts {
			if cam.Scale() < 1 && a.role == "rabbit" {
				continue
			}
			out = r.appendCritter(out, cam, sc, a.role, a.Pose(sc.Site, lc.hb, sc.Time, resting))
		}
		for _, f := range w.fish {
			out = r.appendFish(out, cam, sc, f)
		}
		if resting {
			continue
		}
		for _, f := range w.flocks {
			out = r.appendFlock(out, cam, sc, f)
		}
	}
	return out
}

func (r *Renderer) appendCritter(out []Drawable, cam render.Camera, sc Scene, role string, pose Pose) []Drawable {
	zoom := cam.Scale()
	sx, sy := ScreenPointOf(cam, sc.Site, pose.Pos)
	if !inView(r.Land.view, sx, sy, zoom) {
		return out
	}
	name := gfx.WalkerName(role, pose.Action, pose.Dir, pose.Frame)
	opts := sceneryOptions(scenery{wild: Wild(sc.Site, int(pose.Pos.X+.5), int(pose.Pos.Y+.5)), lift: elevationAt(sc.Site, pose.Pos)})
	return append(out, Drawable{Depth: render.DepthKey(pose.Pos.X, pose.Pos.Y), Draw: func(dst *ebiten.Image) {
		r.Lib.Draw(dst, name, sx, sy, zoom, opts)
	}})
}

func (r *Renderer) appendFish(out []Drawable, cam render.Camera, sc Scene, f fishSpot) []Drawable {
	local := math.Mod(sc.Time+f.phase, f.period)
	if local > fishLeap {
		return out
	}
	zoom := cam.Scale()
	sx, sy := cam.GridToScreen(f.pos.X, f.pos.Y)
	if !inView(r.Land.view, sx, sy, zoom) {
		return out
	}
	name := gfx.EffectName("fish", int(local/fishLeap*art.FishFrames))
	return append(out, Drawable{Depth: render.DepthKey(f.pos.X, f.pos.Y), Draw: func(dst *ebiten.Image) {
		r.Lib.Draw(dst, name, sx, sy, zoom, render.SpriteOptions{})
	}})
}

func (r *Renderer) appendFlock(out []Drawable, cam render.Camera, sc Scene, f flock) []Drawable {
	zoom := cam.Scale()
	for m := 0; m < f.count; m++ {
		w := f.member(sc.Time, m)
		gx, gy := cam.GridToScreen(w.pos.X, w.pos.Y)
		if !inView(r.Land.view, gx, gy, zoom) {
			continue
		}
		name := gfx.WalkerName(f.role, w.action, w.dir, w.frame)
		blot := art.BlotName(1)
		sy := gy - w.alt*zoom
		out = append(out,
			Drawable{Depth: render.DepthKey(w.pos.X, w.pos.Y) - .2, Draw: func(dst *ebiten.Image) {
				r.Lib.Draw(dst, blot, gx, gy, zoom, render.SpriteOptions{})
			}},
			Drawable{Depth: skyDepth + float64(m), Draw: func(dst *ebiten.Image) {
				r.Lib.Draw(dst, name, gx, sy, zoom, render.SpriteOptions{})
			}})
	}
	return out
}
