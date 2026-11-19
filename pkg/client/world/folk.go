package world

import (
	"math"
	"sort"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	maxFolk     = 32
	maxStalls   = 4
	stallGap    = 2.5
	stallRim    = 1.5
	folkSide    = .2
	folkRest    = 2.5
	folkSpeed   = .55
	childSpeed  = 1.0
	folkMinStep = 4
	folkMaxStep = 11
	dockReach   = 5.0
)

type roadGraph struct {
	w, h  int
	on    []bool
	tiles []Point
}

func roadGraphOf(site view.Site) roadGraph {
	g := roadGraph{w: site.Width, h: site.Height, on: make([]bool, site.Width*site.Height)}
	blocked := footprintMask(site)
	for y := 0; y < site.Height; y++ {
		for x := 0; x < site.Width; x++ {
			i := y*site.Width + x
			k := site.Tile(x, y)
			if blocked[i] || Wet(k) || (k != view.TilePlaza && site.RoadAt(x, y) < int(RoadTrodden)) {
				continue
			}
			g.on[i] = true
			g.tiles = append(g.tiles, Point{x, y})
		}
	}
	return g
}

func footprintMask(site view.Site) []bool {
	mask := make([]bool, site.Width*site.Height)
	if p, ok := StoneTile(site); ok {
		mask[p.Y*site.Width+p.X] = true
	}
	for _, s := range site.Structures {
		lo, hi := Footprint(s)
		for y := max(lo.Y, 0); y <= min(hi.Y, site.Height-1); y++ {
			for x := max(lo.X, 0); x <= min(hi.X, site.Width-1); x++ {
				mask[y*site.Width+x] = true
			}
		}
	}
	return mask
}

func (g roadGraph) walk(x, y int) bool {
	return x >= 0 && y >= 0 && x < g.w && y < g.h && g.on[y*g.w+x]
}

func (g roadGraph) next(at, prev Point, h uint32) (Point, bool) {
	var opts [4]Point
	n := 0
	for _, d := range [4][2]int{{1, 0}, {0, 1}, {-1, 0}, {0, -1}} {
		p := Point{at.X + d[0], at.Y + d[1]}
		if g.walk(p.X, p.Y) && p != prev {
			opts[n] = p
			n++
		}
	}
	if n == 0 {
		return prev, g.walk(prev.X, prev.Y) && prev != at
	}
	return opts[h%uint32(n)], true
}

type stroll struct {
	role   string
	look   int
	path   []Vec
	length float64
	speed  float64
	rest   float64
	phase  float64
}

func newStroll(role string, look int, path []Vec, speed, rest, phase float64) stroll {
	return stroll{role: role, look: look, path: path, length: PathLength(path), speed: max(speed, .01), rest: rest, phase: phase}
}

func (s stroll) cycle() float64 {
	return 2 * (s.length/s.speed + s.rest)
}

func (s stroll) At(t float64) Pose {
	clock := t + s.phase
	if len(s.path) < 2 || s.length <= 0 {
		return Pose{Pos: s.path[0], Action: s.pause(), Dir: "se", Frame: Frame(clock, 2)}
	}
	walk := s.length / s.speed
	u := wrap(clock, s.cycle())
	switch {
	case u < s.rest:
		return Pose{Pos: s.path[0], Action: s.pause(), Dir: Facing(s.path[1], s.path[0]), Frame: Frame(clock, 3)}
	case u < s.rest+walk:
		pos, dir := PathAt(s.path, (u-s.rest)/walk)
		return Pose{Pos: pos, Action: s.gait(true), Dir: Facing(Vec{}, dir), Frame: Frame(clock, s.speed*9)}
	case u < 2*s.rest+walk:
		end := s.path[len(s.path)-1]
		return Pose{Pos: end, Action: s.pause(), Dir: Facing(s.path[len(s.path)-2], end), Frame: Frame(clock, 3)}
	}
	pos, dir := PathAt(s.path, 1-(u-2*s.rest-walk)/walk)
	return Pose{Pos: pos, Action: s.gait(false), Dir: Facing(dir, Vec{}), Frame: Frame(clock, s.speed*9)}
}

func (s stroll) pause() string {
	switch s.role {
	case "child", "merchant":
		return "work"
	}
	return "idle"
}

func (s stroll) gait(out bool) string {
	switch {
	case s.role == "carrier" && out, s.role == "farmer" && !out, s.role == "dockhand" && out:
		return "carry"
	}
	return "walk"
}

func planFolk(site view.Site, g roadGraph, n int) []stroll {
	out := make([]stroll, 0, n)
	if len(g.tiles) == 0 {
		return out
	}
	docks := dockAnchors(site)
	for i := 0; i < n; i++ {
		seed := mix32(uint32(i)*0x9e3779b9 ^ 0x51f0c3a7)
		start := g.tiles[seed%uint32(len(g.tiles))]
		role := folkRole(seed, start, docks)
		steps := folkMinStep + int(seed>>8%uint32(folkMaxStep-folkMinStep+1))
		speed := folkSpeed + float64(seed>>12%30)/100
		if role == "child" {
			steps, speed = max(steps/2, 2), childSpeed
		}
		path := strollPath(g, start, steps, seed)
		out = append(out, newStroll(role, int(seed>>20%uint32(gfx.FolkLooks)), path, speed, folkRest+float64(seed>>24%20)/10, float64(seed%997)/13))
	}
	return out
}

func strollPath(g roadGraph, start Point, steps int, seed uint32) []Vec {
	off := Vec{jitterAt(seed, 3, folkSide), jitterAt(seed, 13, folkSide)}
	path := []Vec{{float64(start.X) + off.X, float64(start.Y) + off.Y}}
	at, prev := start, start
	for k := 0; k < steps; k++ {
		nxt, ok := g.next(at, prev, mix32(seed^uint32(k)*0x85ebca6b))
		if !ok {
			break
		}
		prev, at = at, nxt
		path = append(path, Vec{float64(at.X) + off.X, float64(at.Y) + off.Y})
	}
	return path
}

func folkRole(seed uint32, start Point, docks []Vec) string {
	for _, d := range docks {
		if Dist(d, start.Center()) < dockReach && seed>>4%3 == 0 {
			return "dockhand"
		}
	}
	switch roll := seed >> 16 % 100; {
	case roll < 34:
		return "farmer"
	case roll < 60:
		return "carrier"
	case roll < 88:
		return "child"
	}
	return "merchant"
}

func dockAnchors(site view.Site) []Vec {
	out := []Vec{}
	for _, h := range SiteHavens(site) {
		if h.Open {
			out = append(out, VecOf(h.Quay))
		}
	}
	return out
}

func FolkCount(hearths int, served float64, keeps int) int {
	if keeps <= 0 {
		return 0
	}
	return min(2+hearths/2+int(math.Round(max(served, 0)*3)), maxFolk)
}

func FolkBudget(zoom float64) int {
	switch {
	case zoom < .75:
		return 10
	case zoom < 1.5:
		return 20
	}
	return 28
}

func readyKeeps(site view.Site) int {
	n := 0
	for _, k := range Keeps(site) {
		if k.State == view.StateReady {
			n++
		}
	}
	return n
}

func StallCount(rate float64) int {
	switch {
	case rate <= 0:
		return 0
	case rate < .6:
		return 1
	case rate < 1.5:
		return 2
	case rate < 3:
		return 3
	}
	return maxStalls
}

func StallSpots(site view.Site, n int) []Vec {
	plaza := Vec{float64(site.Plaza.X), float64(site.Plaza.Y)}
	camps := campSpots(site)
	cands := []Vec{}
	for y := 0; y < site.Height; y++ {
		for x := 0; x < site.Width; x++ {
			v := Vec{float64(x), float64(y)}
			if site.Tile(x, y) == view.TilePlaza && !NearAnyStructure(site, x, y) && Dist(v, plaza) >= stallRim && clearOfAll(v, camps, 1) && clearOfStone(site, v) {
				cands = append(cands, v)
			}
		}
	}
	sort.SliceStable(cands, func(i, j int) bool { return Dist(cands[i], plaza) > Dist(cands[j], plaza) })
	out := []Vec{}
	for _, c := range cands {
		if len(out) < n && clearOfAll(c, out, stallGap) {
			out = append(out, c)
		}
	}
	return out
}

func clearOfAll(v Vec, others []Vec, gap float64) bool {
	for _, o := range others {
		if Dist(v, o) < gap {
			return false
		}
	}
	return true
}
