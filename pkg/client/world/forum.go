package world

import (
	"fmt"
	"hash/fnv"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	HitForum    = "forum"
	forumMargin = 2
	kerbBias    = .3
	postRhythm  = 3
)

type kerbEdge struct{ x, y, dx, dy int }

type kerbPiece struct {
	axis   string
	mid    Vec
	v      int
	dx, dy int
	id     string
}

type kerbPost struct {
	at     Vec
	v      int
	corner bool
	id     string
}

type kerbPlan struct {
	sig    uint64
	pieces []kerbPiece
	posts  []kerbPost
}

type forumWindow struct{ lo, hi Point }

func forumWindowOf(site view.Site) (forumWindow, bool) {
	lo, hi, ok := site.ForumBox()
	if !ok {
		return forumWindow{}, false
	}
	return forumWindow{lo: Point{lo.X - forumMargin, lo.Y - forumMargin}, hi: Point{hi.X + forumMargin, hi.Y + forumMargin}}, true
}

func (w forumWindow) each(fn func(x, y int)) {
	for y := w.lo.Y; y <= w.hi.Y; y++ {
		for x := w.lo.X; x <= w.hi.X; x++ {
			fn(x, y)
		}
	}
}

func forumSig(site view.Site) uint64 {
	h := fnv.New64a()
	h.Write([]byte{byte(site.Width), byte(site.Width >> 8)})
	if w, ok := forumWindowOf(site); ok {
		h.Write([]byte{byte(w.lo.X), byte(w.lo.Y), byte(w.hi.X), byte(w.hi.Y)})
		w.each(func(x, y int) { h.Write([]byte{byte(site.Tile(x, y)), RoadLevel(site, x, y)}) })
	}
	return h.Sum64() ^ StructureSig(site)
}

func kerbEdges(site view.Site) []kerbEdge {
	out := []kerbEdge{}
	w, ok := forumWindowOf(site)
	if !ok {
		return out
	}
	w.each(func(x, y int) {
		if !site.OnForum(x, y) || Occupied(site, x, y) {
			return
		}
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if !site.OnForum(nx, ny) && RoadLevel(site, nx, ny) < RoadDirt && !Occupied(site, nx, ny) {
				out = append(out, kerbEdge{x: x, y: y, dx: d[0], dy: d[1]})
			}
		}
	})
	return out
}

func (e kerbEdge) axis() string {
	if e.dx != 0 {
		return "x"
	}
	return "y"
}

func (e kerbEdge) vertices() [2]Point {
	switch {
	case e.dx == 1:
		return [2]Point{{e.x + 1, e.y}, {e.x + 1, e.y + 1}}
	case e.dx == -1:
		return [2]Point{{e.x, e.y}, {e.x, e.y + 1}}
	case e.dy == 1:
		return [2]Point{{e.x, e.y + 1}, {e.x + 1, e.y + 1}}
	}
	return [2]Point{{e.x, e.y}, {e.x + 1, e.y}}
}

func PlanKerb(site view.Site) kerbPlan {
	edges := kerbEdges(site)
	plan := kerbPlan{sig: forumSig(site)}
	meets := map[Point][]string{}
	order := []Point{}
	for _, e := range edges {
		plan.pieces = append(plan.pieces, kerbPiece{axis: e.axis(), mid: Vec{float64(e.x) + float64(e.dx)/2, float64(e.y) + float64(e.dy)/2},
			v: int(hash2(e.x, e.y) % art.ForumKerbKinds), dx: e.dx, dy: e.dy, id: fmt.Sprintf("forum/%d/%d", e.x, e.y)})
		for _, p := range e.vertices() {
			if _, seen := meets[p]; !seen {
				order = append(order, p)
			}
			meets[p] = append(meets[p], e.axis())
		}
	}
	for _, p := range order {
		if postAt(p, meets[p]) {
			corner := len(meets[p]) != 2 || meets[p][0] != meets[p][1]
			plan.posts = append(plan.posts, kerbPost{at: Vec{float64(p.X) - .5, float64(p.Y) - .5}, v: int(hash2(p.X, p.Y) % art.ForumPostKinds), corner: corner,
				id: fmt.Sprintf("forum/post/%d/%d", p.X, p.Y)})
		}
	}
	return plan
}

func postAt(p Point, axes []string) bool {
	if len(axes) != 2 || axes[0] != axes[1] {
		return true
	}
	return (p.X+p.Y)%postRhythm == 0
}

func hash2(x, y int) uint32 {
	return uint32(x*73856093) ^ uint32(y*19349663)
}

func ForumTints(site view.Site, kind string) []Tint {
	out := []Tint{}
	if w, ok := forumWindowOf(site); ok {
		w.each(func(x, y int) {
			if site.OnForum(x, y) {
				out = append(out, Tint{X: x, Y: y, Kind: kind})
			}
		})
	}
	return out
}

func OnTheForum(site view.Site, s view.Structure) bool {
	lo, hi := Footprint(s)
	for y := lo.Y; y <= hi.Y; y++ {
		for x := lo.X; x <= hi.X; x++ {
			if site.OnForum(x, y) {
				return true
			}
		}
	}
	return false
}

func kerbDepth(v Vec) float64 {
	return render.DepthKey(v.X, v.Y)
}
