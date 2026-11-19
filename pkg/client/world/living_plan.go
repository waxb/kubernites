package world

import (
	"math"
	"sort"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	patchMin       = 5
	patchSpan      = 3
	patchReach     = 4
	patchAcross    = 1.8
	patchInset     = .5
	patchInsetStep = .35
	patchTries     = 8
	bushPerKeep    = 3
	stockAnchors   = 4
	stockRingLo    = 2.4
	stockRingHi    = 4.2
)

type wheatTile struct {
	at      Point
	mid     float64
	variant int
	patch   int
}

type bushSpot struct {
	at      Vec
	variant int
}

type keepPlan struct {
	wheat   []wheatTile
	bushes  []bushSpot
	stock   []Vec
	centre  Vec
	patches int
}

type livingPlans struct {
	sig     uint64
	byKeep  map[string]keepPlan
	ground  ground
	borders Borders
	farm    map[Point]bool
}

type ground struct {
	site    view.Site
	blocked func(x, y int) bool
	plots   map[Point]bool
}

type borderPt struct {
	at    Point
	angle float64
}

func (g ground) ok(x, y int) bool {
	k := g.site.Tile(x, y)
	return (k == view.TileGrass || k == view.TileMeadow) && RoadLevel(g.site, x, y) == RoadNone && !covered(g.site, x, y) && (g.blocked == nil || !g.blocked(x, y)) && !g.plots[Point{x, y}]
}

func plotSet(site view.Site) map[Point]bool {
	if len(site.Plots) == 0 {
		return nil
	}
	out := make(map[Point]bool, len(site.Plots))
	for _, p := range site.Plots {
		out[Point{p.X, p.Y}] = true
	}
	return out
}

func PatchCount(cores int) int { return min(max(2+cores/4, 2), 4) }

func (p *livingPlans) sync(sc Scene, cover uint64, blocked func(x, y int) bool) {
	sig := BorderSig(sc.Site) ^ cover*sigPrime
	if p.byKeep != nil && p.sig == sig {
		return
	}
	p.sig, p.byKeep, p.ground, p.farm = sig, map[string]keepPlan{}, ground{site: sc.Site, blocked: blocked, plots: plotSet(sc.Site)}, map[Point]bool{}
	p.borders = BordersOf(sc.Site)
	if sc.Borders != nil && sc.Borders.Sig == BorderSig(sc.Site) {
		p.borders = *sc.Borders
	}
	for _, s := range sc.Site.Structures {
		if s.Kind != view.StructKeep || s.State != view.StateReady {
			continue
		}
		plan, ok := p.of(s)
		for _, t := range plan.wheat {
			p.farm[t.at] = ok
		}
		for _, b := range plan.bushes {
			p.farm[TileOf(b.at)] = ok
		}
	}
}

func (p *livingPlans) of(s view.Structure) (keepPlan, bool) {
	if plan, ok := p.byKeep[s.ID]; ok {
		return plan, true
	}
	i, ok := p.borders.AreaOf(s.ID)
	if !ok {
		return keepPlan{}, false
	}
	plan := planKeep(p.ground, p.borders, i, s)
	p.byKeep[s.ID] = plan
	return plan, true
}

func borderRing(area view.Territory) []borderPt {
	cx, cy := float64(area.Center.X), float64(area.Center.Y)
	out := make([]borderPt, 0, len(area.Border))
	for _, p := range area.Border {
		out = append(out, borderPt{at: Point{p.X, p.Y}, angle: math.Atan2(float64(p.Y)-cy, float64(p.X)-cx)})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].angle != out[j].angle {
			return out[i].angle < out[j].angle
		}
		if out[i].at.X != out[j].at.X {
			return out[i].at.X < out[j].at.X
		}
		return out[i].at.Y < out[j].at.Y
	})
	return out
}

func ringGap(a, b float64) float64 {
	d := math.Mod(math.Abs(a-b), 2*math.Pi)
	return min(d, 2*math.Pi-d)
}

func nearestRing(ring []borderPt, theta float64, ok func(Point) bool) (Point, bool) {
	best, gap, found := Point{}, math.MaxFloat64, false
	for _, b := range ring {
		if g := ringGap(b.angle, theta); g < gap && ok(b.at) {
			best, gap, found = b.at, g, true
		}
	}
	return best, found
}

func planKeep(g ground, b Borders, area int, s view.Structure) keepPlan {
	a := b.Areas[area]
	centre := Vec{float64(a.Center.X), float64(a.Center.Y)}
	cores := s.CPU
	plan := keepPlan{centre: centre, patches: PatchCount(cores)}
	ring := borderRing(a)
	seed := model.Hash(s.ID)
	base := float64(seed%628) / 100
	used := map[Point]bool{}
	inside := func(p Point) bool {
		o, ok := b.At(p.X, p.Y)
		return ok && o == area
	}
	for j := 0; j < plan.patches; j++ {
		theta := base + 2*math.Pi*float64(j)/float64(plan.patches)
		if tiles, ok := wheatPatch(g, ring, theta, centre, used, inside, mix32(seed+uint32(j)*0x9e3779b9)); ok {
			for _, t := range tiles {
				t.patch = j
				plan.wheat = append(plan.wheat, t)
			}
		}
	}
	plan.bushes = bushSpots(g, ring, base, plan.patches, centre, used, seed)
	plan.stock = stockSpots(g, s, used, seed)
	return plan
}

func wheatPatch(g ground, ring []borderPt, theta float64, centre Vec, used map[Point]bool, inside func(Point) bool, h uint32) ([]wheatTile, bool) {
	size := patchMin + int(h%patchSpan)
	free := func(p Point) bool { return !used[p] && g.ok(p.X, p.Y) }
	for try := 0; try < patchTries; try++ {
		anchor, ok := nearestRing(ring, theta+float64(try)*.18*float64(1-2*(try%2)), free)
		if !ok {
			return nil, false
		}
		inset := patchInset + float64(h>>uint(try)%3)*patchInsetStep
		if tiles := gatherPatch(g, anchor, centre, inset, size, used, inside); len(tiles) >= patchMin {
			for _, t := range tiles {
				used[t.at] = true
			}
			return tiles, true
		}
	}
	return nil, false
}

func gatherPatch(g ground, anchor Point, centre Vec, inset float64, size int, used map[Point]bool, inside func(Point) bool) []wheatTile {
	at := Vec{float64(anchor.X), float64(anchor.Y)}
	u := Vec{at.X - centre.X, at.Y - centre.Y}
	n := math.Hypot(u.X, u.Y)
	if n == 0 {
		return nil
	}
	u = Vec{u.X / n, u.Y / n}
	t := Vec{-u.Y, u.X}
	c := Vec{at.X - u.X*inset, at.Y - u.Y*inset}
	type cand struct {
		at    Point
		score float64
	}
	cands := []cand{}
	for y := anchor.Y - patchReach; y <= anchor.Y+patchReach; y++ {
		for x := anchor.X - patchReach; x <= anchor.X+patchReach; x++ {
			p := Point{x, y}
			if used[p] || !g.ok(x, y) {
				continue
			}
			d := Vec{float64(x) - c.X, float64(y) - c.Y}
			along, across := d.X*t.X+d.Y*t.Y, d.X*u.X+d.Y*u.Y
			cands = append(cands, cand{at: p, score: along*along + patchAcross*across*across})
		}
	}
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].score != cands[j].score {
			return cands[i].score < cands[j].score
		}
		return tileHash(cands[i].at.X, cands[i].at.Y) < tileHash(cands[j].at.X, cands[j].at.Y)
	})
	out, outside, far, near := []wheatTile{}, 0, 0.0, math.MaxFloat64
	for _, cd := range cands {
		if len(out) == size {
			break
		}
		if !inside(cd.at) {
			if outside >= size/2 {
				continue
			}
			outside++
		}
		far, near = math.Max(far, math.Sqrt(cd.score)), math.Min(near, math.Sqrt(cd.score))
		out = append(out, wheatTile{at: cd.at, mid: math.Sqrt(cd.score), variant: int(tileHash(cd.at.X, cd.at.Y) % 2)})
	}
	for i := range out {
		out[i].mid = 1 - (out[i].mid-near)/(far-near+.6)
	}
	return out
}

func bushSpots(g ground, ring []borderPt, base float64, patches int, centre Vec, used map[Point]bool, seed uint32) []bushSpot {
	out := []bushSpot{}
	step := 2 * math.Pi / float64(patches)
	for j := 0; j < bushPerKeep; j++ {
		theta := base + step*(float64(j%patches)+.5) + float64(j/patches)*.4
		anchor, ok := nearestRing(ring, theta, func(p Point) bool { return !used[p] && g.ok(p.X, p.Y) })
		if !ok {
			continue
		}
		used[anchor] = true
		h := mix32(seed ^ uint32(j+1)*0x85ebca6b)
		out = append(out, bushSpot{at: Vec{float64(anchor.X) + jitterAt(h, 3, .25), float64(anchor.Y) + jitterAt(h, 13, .25)}, variant: int(h >> 20 % 2)})
	}
	return out
}

func stockSpots(g ground, s view.Structure, used map[Point]bool, seed uint32) []Vec {
	c := keepMid(s)
	half := float64(max(s.W, s.H, 1)) / 2
	out := []Vec{}
	for j := 0; j < stockAnchors*3 && len(out) < stockAnchors; j++ {
		h := mix32(seed ^ uint32(j+7)*0xc2b2ae35)
		ang := float64(h%628) / 100
		d := half + stockRingLo + float64(h>>12%100)/100*(stockRingHi-stockRingLo)
		at := Vec{c.X + math.Cos(ang)*d, c.Y + math.Sin(ang)*d}
		p := TileOf(at)
		if !used[p] && g.ok(p.X, p.Y) {
			out = append(out, at)
		}
	}
	return out
}

func keepMid(s view.Structure) Vec {
	return Vec{float64(s.X) + float64(max(s.W, 1)-1)/2, float64(s.Y) + float64(max(s.H, 1)-1)/2}
}
