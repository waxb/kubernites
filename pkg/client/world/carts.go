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
	cartEvery   = 6.0
	cartMinLoad = 5.0
	cartSpeed   = .9
	maxCarts    = 3
	cartFade    = .8
	cartReach   = 2
	cartTurns   = 4.0
)

var cartNames = func() [art.CartLoads][2][art.CartFrames]string {
	var out [art.CartLoads][2][art.CartFrames]string
	for l := range out {
		for d, dir := range []string{"se", "sw"} {
			for f := range out[l][d] {
				out[l][d][f] = gfx.CartName(l, dir, f)
			}
		}
	}
	return out
}()

type cartRun struct {
	start, life float64
	load        int
}

type cartYard struct {
	primed bool
	banked float64
	lastAt float64
	seen   float64
	rolls  []cartRun
}

type cartPose struct {
	pos   Vec
	dir   string
	alpha float64
	frame int
}

func CartLoadLevel(boxes float64) int {
	switch {
	case boxes < 20:
		return 0
	case boxes < 60:
		return 1
	}
	return 2
}

func (y *cartYard) step(now, served, routeLen float64) {
	if !y.primed || now < y.seen || served < y.banked || routeLen <= 0 {
		*y = cartYard{primed: true, banked: served, lastAt: now, seen: now}
		return
	}
	y.seen = now
	live := y.rolls[:0]
	for _, c := range y.rolls {
		if now <= c.start+c.life {
			live = append(live, c)
		}
	}
	y.rolls = live
	due := served - y.banked
	if due < cartMinLoad || now-y.lastAt < cartEvery || len(y.rolls) >= maxCarts {
		return
	}
	y.rolls = append(y.rolls, cartRun{start: now, life: routeLen / cartSpeed, load: CartLoadLevel(due)})
	y.banked, y.lastAt = served, now
}

func (c cartRun) pose(route []Vec, now float64) (cartPose, bool) {
	age := now - c.start
	if len(route) < 2 || c.life <= 0 || age < 0 || age > c.life {
		return cartPose{}, false
	}
	pos, dir := PathAt(route, age/c.life)
	travelled := age / c.life * PathLength(route)
	return cartPose{pos: pos, dir: Facing(Vec{}, dir), alpha: min(age/cartFade, (c.life-age)/cartFade, 1), frame: int(travelled*cartTurns) % art.CartFrames}, true
}

func CartRoute(site view.Site, g roadGraph) []Vec {
	gh, okG := FirstOfKind(site, view.StructGuildhall, false)
	tr, okT := FirstOfKind(site, view.StructTreasury, false)
	if !okG || !okT {
		return nil
	}
	from, to := Porch(gh), Porch(tr)
	a, okA := nearestWalk(g, TileOf(from))
	b, okB := nearestWalk(g, TileOf(to))
	if !okA || !okB {
		return []Vec{from, to}
	}
	tiles, ok := g.route(a, b)
	if !ok {
		return []Vec{from, to}
	}
	out := make([]Vec, 0, len(tiles)+2)
	out = append(out, from)
	for _, t := range tiles {
		out = append(out, t.Center())
	}
	return append(out, to)
}

func nearestWalk(g roadGraph, at Point) (Point, bool) {
	best, found, bestD := Point{}, false, math.MaxFloat64
	for dy := -cartReach; dy <= cartReach; dy++ {
		for dx := -cartReach; dx <= cartReach; dx++ {
			p := Point{at.X + dx, at.Y + dy}
			if d := math.Hypot(float64(dx), float64(dy)); g.walk(p.X, p.Y) && d < bestD {
				best, found, bestD = p, true, d
			}
		}
	}
	return best, found
}

func (g roadGraph) route(a, b Point) ([]Point, bool) {
	prev := map[Point]Point{a: a}
	queue := []Point{a}
	for len(queue) > 0 {
		at := queue[0]
		queue = queue[1:]
		if at == b {
			return unwind(prev, a, b), true
		}
		for _, d := range [4][2]int{{1, 0}, {0, 1}, {-1, 0}, {0, -1}} {
			p := Point{at.X + d[0], at.Y + d[1]}
			if _, seen := prev[p]; !seen && g.walk(p.X, p.Y) {
				prev[p] = at
				queue = append(queue, p)
			}
		}
	}
	return nil, false
}

func unwind(prev map[Point]Point, a, b Point) []Point {
	out := []Point{b}
	for at := b; at != a; {
		at = prev[at]
		out = append(out, at)
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func guildTreasuryReady(site view.Site) bool {
	_, g := FirstOfKind(site, view.StructGuildhall, true)
	_, t := FirstOfKind(site, view.StructTreasury, true)
	return g && t
}

func (r *Renderer) cartItems(out []Drawable, cam render.Camera, sc Scene, l *life) []Drawable {
	length := 0.0
	if guildTreasuryReady(sc.Site) {
		length = PathLength(l.road)
	}
	l.carts.step(sc.Time, sc.Served-sc.HandServed, length)
	zoom := cam.Scale()
	for _, c := range l.carts.rolls {
		p, ok := c.pose(l.road, sc.Time)
		if !ok {
			continue
		}
		sx, sy := ScreenPointOf(cam, sc.Site, p.pos)
		if !onScreen(sx, sy, zoom) {
			continue
		}
		name := cartNames[min(c.load, art.CartLoads-1)][dirIndex(p.dir)][p.frame]
		alpha := float32(p.alpha)
		out = append(out, Drawable{Depth: render.DepthKey(p.pos.X, p.pos.Y) + .05, stand: stand(p.pos, sx, sy), Draw: func(dst *ebiten.Image) {
			r.Lib.Draw(dst, name, sx, sy, zoom, render.SpriteOptions{Alpha: alpha})
		}})
	}
	return out
}

func dirIndex(dir string) int {
	if dir == "sw" {
		return 1
	}
	return 0
}
