package world

import (
	"encoding/binary"
	"hash/fnv"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	TintOwn     = "own"
	TintGood    = "good"
	TintPoor    = "poor"
	TintForum   = "forum"
	markBias    = .25
	flagHz      = 1.6
	flagPhase   = .37
	stoneKinds  = 3
	roadHides   = 2
	cornerCount = 4
)

type BorderMark struct {
	X, Y    int
	Flag    bool
	Variant int
}

type Tint struct {
	X, Y int
	Kind string
}

type Borders struct {
	Sig   uint64
	Width int
	Areas []view.Territory
	Owner []int16
	Marks []BorderMark
}

func BorderSig(site view.Site) uint64 {
	h := fnv.New64a()
	var buf [8]byte
	put := func(vals ...int) {
		for _, v := range vals {
			binary.LittleEndian.PutUint64(buf[:], uint64(v))
			h.Write(buf[:])
		}
	}
	put(site.Width, site.Height, len(site.Tiles), len(site.Havens))
	for _, is := range site.Islands {
		h.Write([]byte(is.ID))
		put(boolInt(is.Owned))
	}
	for _, s := range site.Structures {
		h.Write([]byte(s.ID + "/" + s.Kind + "/" + s.Variant + "/"))
		put(s.X, s.Y, s.W, s.H, s.Territory)
	}
	return h.Sum64()
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func BordersOf(site view.Site) Borders {
	areas := site.Territories()
	owner := ownerOf(site, areas)
	return Borders{
		Sig: BorderSig(site), Width: site.Width, Areas: areas, Owner: owner,
		Marks: markRings(areas, func(i int, p view.Point) bool {
			return FreeTile(site, p.X, p.Y) && !sharedEdge(owner, site.Width, i, p)
		}),
	}
}

func sharedEdge(owner []int16, width, area int, p view.Point) bool {
	me := int16(area + 1)
	for _, d := range [][2]int{{0, 0}, {1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		x, y := p.X+d[0], p.Y+d[1]
		if x < 0 || y < 0 || x >= width || y*width+x >= len(owner) {
			continue
		}
		if o := owner[y*width+x]; o != 0 && o != me {
			return true
		}
	}
	return false
}

func ownerOf(site view.Site, areas []view.Territory) []int16 {
	owner := make([]int16, len(site.Tiles))
	best := make([]int, len(site.Tiles))
	for i, a := range areas {
		for _, p := range site.TerritoryTiles(a.Center.X, a.Center.Y, 1, 1, a.Radius) {
			k := p.Y*site.Width + p.X
			dx, dy := p.X-a.Center.X, p.Y-a.Center.Y
			if d := dx*dx + dy*dy; owner[k] == 0 || d < best[k] {
				owner[k], best[k] = int16(i+1), d
			}
		}
	}
	return owner
}

func FreeTile(site view.Site, x, y int) bool {
	if t := site.Tile(x, y); t == view.TileRoad || t == view.TilePlaza {
		return false
	}
	for _, s := range site.Structures {
		if Covers(s, x, y) {
			return false
		}
	}
	return true
}

func (l Borders) At(x, y int) (int, bool) {
	if l.Width <= 0 || x < 0 || y < 0 || x >= l.Width {
		return 0, false
	}
	k := y*l.Width + x
	if k >= len(l.Owner) || l.Owner[k] == 0 {
		return 0, false
	}
	return int(l.Owner[k]) - 1, true
}

func (l Borders) AreaOf(keep string) (int, bool) {
	for i, a := range l.Areas {
		if a.Keep == keep {
			return i, true
		}
	}
	return 0, false
}

func Corners(a view.Territory) []view.Point {
	if len(a.Border) < cornerCount {
		return nil
	}
	c := a.Center
	keys := []func(p view.Point) (int, int){
		func(p view.Point) (int, int) { return p.X + p.Y, Abs(p.X - p.Y - c.X + c.Y) },
		func(p view.Point) (int, int) { return -(p.X + p.Y), Abs(p.X - p.Y - c.X + c.Y) },
		func(p view.Point) (int, int) { return p.X - p.Y, Abs(p.X + p.Y - c.X - c.Y) },
		func(p view.Point) (int, int) { return p.Y - p.X, Abs(p.X + p.Y - c.X - c.Y) },
	}
	out := []view.Point{}
	seen := map[view.Point]bool{}
	for _, key := range keys {
		best := extreme(a.Border, key)
		if !seen[best] {
			seen[best] = true
			out = append(out, best)
		}
	}
	return out
}

func extreme(ps []view.Point, key func(view.Point) (int, int)) view.Point {
	best := ps[0]
	b1, b2 := key(best)
	for _, p := range ps[1:] {
		k1, k2 := key(p)
		if k1 < b1 || k1 == b1 && (k2 < b2 || k2 == b2 && (p.Y < best.Y || p.Y == best.Y && p.X < best.X)) {
			best, b1, b2 = p, k1, k2
		}
	}
	return best
}

func ringOrder(a view.Territory) []view.Point {
	out := append([]view.Point(nil), a.Border...)
	angle := func(p view.Point) float64 {
		return math.Atan2(float64(p.Y-a.Center.Y), float64(p.X-a.Center.X))
	}
	sort.SliceStable(out, func(i, j int) bool {
		ai, aj := angle(out[i]), angle(out[j])
		if ai != aj {
			return ai < aj
		}
		return out[i].Y < out[j].Y || out[i].Y == out[j].Y && out[i].X < out[j].X
	})
	return out
}

func crowded(taken map[view.Point]bool, p view.Point) bool {
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if taken[view.Point{X: p.X + dx, Y: p.Y + dy}] {
				return true
			}
		}
	}
	return false
}

func stoneVariant(p view.Point) int {
	return Abs(p.X*7+p.Y*13) % stoneKinds
}

func BorderMarks(areas []view.Territory, free func(x, y int) bool) []BorderMark {
	return markRings(areas, func(_ int, p view.Point) bool { return free(p.X, p.Y) })
}

func markRings(areas []view.Territory, keep func(i int, p view.Point) bool) []BorderMark {
	taken := map[view.Point]bool{}
	out := []BorderMark{}
	for i, a := range areas {
		for _, p := range Corners(a) {
			if keep(i, p) && !taken[p] {
				out = append(out, BorderMark{X: p.X, Y: p.Y, Flag: true, Variant: stoneVariant(p)})
				taken[p] = true
			}
		}
		for _, p := range ringOrder(a) {
			if !keep(i, p) || crowded(taken, p) {
				continue
			}
			out = append(out, BorderMark{X: p.X, Y: p.Y, Variant: stoneVariant(p)})
			taken[p] = true
		}
	}
	return out
}

func AreaTints(l Borders, areas []int, kind string) []Tint {
	if len(areas) == 0 || l.Width <= 0 {
		return nil
	}
	want := map[int16]bool{}
	for _, i := range areas {
		want[int16(i+1)] = true
	}
	out := []Tint{}
	for k, o := range l.Owner {
		if o != 0 && want[o] {
			out = append(out, Tint{X: k % l.Width, Y: k / l.Width, Kind: kind})
		}
	}
	return out
}

func DiscTints(tiles []view.Point, kind string) []Tint {
	out := make([]Tint, 0, len(tiles))
	for _, p := range tiles {
		out = append(out, Tint{X: p.X, Y: p.Y, Kind: kind})
	}
	return out
}

func MergeTints(layers ...[]Tint) []Tint {
	at := map[view.Point]int{}
	out := []Tint{}
	for _, layer := range layers {
		for _, t := range layer {
			p := view.Point{X: t.X, Y: t.Y}
			if i, ok := at[p]; ok {
				out[i].Kind = t.Kind
				continue
			}
			at[p] = len(out)
			out = append(out, t)
		}
	}
	return out
}

func MarkName(m BorderMark, now float64, reduce bool) string {
	if !m.Flag {
		return gfx.BorderStoneName(m.Variant)
	}
	frame := 0
	if !reduce {
		frame = int(math.Floor(now*flagHz + float64(m.X+m.Y)*flagPhase))
	}
	return gfx.BorderFlagName(frame)
}

func (r *Renderer) DrawTints(dst *ebiten.Image, cam render.Camera, sc Scene) {
	if len(sc.Tints) == 0 {
		return
	}
	x0, y0, x1, y1 := visibleTiles(dst, cam, sc.Site)
	zoom := cam.Scale()
	for _, t := range sc.Tints {
		if t.X < x0 || t.X > x1 || t.Y < y0 || t.Y > y1 {
			continue
		}
		sx, sy := cam.GridToScreenElevated(float64(t.X), float64(t.Y), float64(sc.Site.ElevationAt(t.X, t.Y)))
		r.Lib.Draw(dst, gfx.GroundTintName(t.Kind), sx, sy, zoom, render.SpriteOptions{})
	}
}

func (r *Renderer) BorderItems(cam render.Camera, sc Scene) []Drawable {
	out := []Drawable{}
	if sc.Borders != nil {
		out = append(out, r.markItems(cam, sc, sc.Borders.Marks, 1)...)
	}
	return append(out, r.markItems(cam, sc, sc.Preview, previewAlpha)...)
}
