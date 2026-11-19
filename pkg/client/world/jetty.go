package world

import (
	"math"
	"sort"

	"github.com/waxb/kubernites/pkg/view"
)

const (
	JettyPlanks = 2
	dockAhead   = .9
	JettyClear  = 2
	lanePairs   = 6000
	facingBias  = .05
	nearSlack   = 10
)

var axes = []Vec{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

type Jetty struct {
	Root   Point
	Out    Vec
	Planks []Vec
	Head   Vec
	Dock   Vec
}

type FerryLane struct {
	Isle   string
	Home   Jetty
	Away   Jetty
	Path   []Vec
	Length float64
}

type coastPair struct {
	a, b Point
	d    float64
}

func PlanLanes(site view.Site) []FerryLane {
	blocked := LaneBlocked(site)
	out := []FerryLane{}
	for _, is := range site.Islands {
		if !is.Owned || is.ID == view.HomeIslandID {
			continue
		}
		if lane, ok := planLane(site, is.ID, blocked); ok {
			out = append(out, lane)
			for _, j := range []Jetty{lane.Home, lane.Away} {
				markJetty(blocked, site.Width, j)
			}
		}
	}
	return out
}

func markJetty(blocked []bool, w int, j Jetty) {
	for _, p := range append([]Vec{j.Root.Center()}, j.Planks...) {
		t := TileOf(p)
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if i := (t.Y+dy)*w + t.X + dx; i >= 0 && i < len(blocked) {
					blocked[i] = true
				}
			}
		}
	}
}

func LaneBlocked(site view.Site) []bool {
	blocked := make([]bool, site.Width*site.Height)
	mark := func(lo, hi view.Point, pad int) {
		for y := max(lo.Y-pad, 0); y <= min(hi.Y+pad, site.Height-1); y++ {
			for x := max(lo.X-pad, 0); x <= min(hi.X+pad, site.Width-1); x++ {
				blocked[y*site.Width+x] = true
			}
		}
	}
	for _, s := range site.Structures {
		lo, hi := Footprint(s)
		mark(view.Point{X: lo.X, Y: lo.Y}, view.Point{X: hi.X, Y: hi.Y}, 1)
	}
	for _, h := range SiteHavens(site) {
		lo, hi := h.QuayArea()
		for _, p := range []view.Point{lo, hi, h.PierHead(), h.Yards()[0], h.Yards()[1]} {
			lo, hi = view.Point{X: min(lo.X, p.X), Y: min(lo.Y, p.Y)}, view.Point{X: max(hi.X, p.X), Y: max(hi.Y, p.Y)}
		}
		mark(lo, hi, JettyClear)
	}
	return blocked
}

func coastTiles(site view.Site, isle string) []Point {
	out := []Point{}
	for y := 0; y < site.Height; y++ {
		for x := 0; x < site.Width; x++ {
			if is, ok := site.IslandAt(x, y); !ok || is.ID != isle || Wet(site.Tile(x, y)) {
				continue
			}
			for _, d := range axes {
				if Wet(site.Tile(x+int(d.X), y+int(d.Y))) {
					out = append(out, Point{x, y})
					break
				}
			}
		}
	}
	return out
}

func biases(tiles []Point, c Vec) []float64 {
	out := make([]float64, len(tiles))
	for i, t := range tiles {
		out[i] = facingBias * Dist(t.Center(), c)
	}
	return out
}

func nearestPairs(away, home []Point, ca, ch Vec, slack float64) []coastPair {
	ba, bh := biases(away, ca), biases(home, ch)
	score := func(i, j int) float64 {
		return math.Hypot(float64(away[i].X-home[j].X), float64(away[i].Y-home[j].Y)) + ba[i] + bh[j]
	}
	best := math.Inf(1)
	for i := range away {
		for j := range home {
			best = min(best, score(i, j))
		}
	}
	pairs := []coastPair{}
	for i, a := range away {
		for j, b := range home {
			if d := score(i, j); d <= best+slack {
				pairs = append(pairs, coastPair{a: a, b: b, d: d})
			}
		}
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].d != pairs[j].d {
			return pairs[i].d < pairs[j].d
		}
		return pairs[i].a.Y*4096+pairs[i].a.X+pairs[i].b.Y*16384+pairs[i].b.X < pairs[j].a.Y*4096+pairs[j].a.X+pairs[j].b.Y*16384+pairs[j].b.X
	})
	return pairs[:min(len(pairs), lanePairs)]
}

func isleCenter(site view.Site, id string) Vec {
	is, _ := site.Island(id)
	return VecOf(is.Center)
}

func planLane(site view.Site, isle string, blocked []bool) (FerryLane, bool) {
	ca, ch := isleCenter(site, isle), isleCenter(site, view.HomeIslandID)
	away, home := coastTiles(site, isle), coastTiles(site, view.HomeIslandID)
	for _, slack := range []float64{nearSlack, math.Inf(1)} {
		if lane, ok := laneFrom(site, isle, blocked, nearestPairs(away, home, ca, ch, slack)); ok {
			return lane, true
		}
	}
	return FerryLane{}, false
}

func laneFrom(site view.Site, isle string, blocked []bool, pairs []coastPair) (FerryLane, bool) {
	for _, pr := range pairs {
		away, ok := jettyAt(site, blocked, pr.a, pr.b.Center())
		if !ok {
			continue
		}
		home, ok := jettyAt(site, blocked, pr.b, pr.a.Center())
		if !ok || !WetLine(site, home.Dock, away.Dock) {
			continue
		}
		path := []Vec{home.Dock, away.Dock}
		lane := FerryLane{Isle: isle, Home: home, Away: away, Path: path, Length: PathLength(path)}
		if lane.clear(site) {
			return lane, true
		}
	}
	return FerryLane{}, false
}

func jettyAt(site view.Site, blocked []bool, root Point, toward Vec) (Jetty, bool) {
	if blocked[root.Y*site.Width+root.X] {
		return Jetty{}, false
	}
	want := Vec{toward.X - float64(root.X), toward.Y - float64(root.Y)}
	dirs := append([]Vec(nil), axes...)
	sort.SliceStable(dirs, func(i, j int) bool { return dot(dirs[i], want) > dot(dirs[j], want) })
	for _, d := range dirs {
		if dot(d, want) <= 0 {
			break
		}
		if j, ok := JettyToward(site, blocked, root, d); ok {
			return j, true
		}
	}
	return Jetty{}, false
}

func JettyToward(site view.Site, blocked []bool, root Point, d Vec) (Jetty, bool) {
	j := Jetty{Root: root, Out: d}
	for k := 1; k <= JettyPlanks; k++ {
		v := offset(root.Center(), d, float64(k))
		t := TileOf(v)
		if t.X < 0 || t.Y < 0 || t.X >= site.Width || t.Y >= site.Height || !Wet(site.Tile(t.X, t.Y)) || blocked[t.Y*site.Width+t.X] {
			break
		}
		j.Planks = append(j.Planks, v)
	}
	if len(j.Planks) == 0 {
		return Jetty{}, false
	}
	j.Head = j.Planks[len(j.Planks)-1]
	j.Dock = offset(j.Head, d, dockAhead)
	return j, WetAt(site, j.Dock)
}

func dot(a, b Vec) float64 { return a.X*b.X + a.Y*b.Y }

func (j Jetty) Landing() Vec {
	return offset(offset(j.Root.Center(), j.Out, landingAhead), Vec{-j.Out.Y, j.Out.X}, landingAside)
}

func (j Jetty) Claims() []Point {
	out := []Point{j.Root, TileOf(j.Landing())}
	for _, p := range j.Planks {
		out = append(out, TileOf(p))
	}
	return out
}

func (l FerryLane) clear(site view.Site) bool {
	steps := int(l.Length/laneStep) + 1
	for i := 0; i <= steps; i++ {
		u := float64(i) / float64(steps)
		for _, side := range []float64{1, -1} {
			if !WetAt(site, l.afloat(l.Path, u, side)) {
				return false
			}
		}
	}
	return true
}
