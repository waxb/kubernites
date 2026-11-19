package view

import (
	"sort"
	"strings"
)

const FaceSE = "se"

const FaceSW = "sw"

const FaceNW = "nw"

const FaceNE = "ne"

const BackClass = 3

const CrowdedClass = 6

type StallSlot struct {
	X    int    `json:"x"`
	Y    int    `json:"y"`
	W    int    `json:"w"`
	H    int    `json:"h"`
	Face string `json:"face"`
}

func (sl StallSlot) Rect() Rect {
	return Rect{X0: sl.X, Y0: sl.Y, X1: sl.X + sl.W - 1, Y1: sl.Y + sl.H - 1}
}

func StallKind(kind string) bool {
	switch kind {
	case StructGuildhall, StructArchive, StructHerald, StructPlaque:
		return true
	}
	return strings.HasPrefix(kind, DepotPrefix)
}

func IsStall(s Structure) bool { return StallKind(s.Kind) }

func (b Rect) Ring() []Point {
	if b.Area() == 0 {
		return nil
	}
	if b.X0 == b.X1 || b.Y0 == b.Y1 {
		return b.Tiles()
	}
	out := make([]Point, 0, 2*(b.X1-b.X0+b.Y1-b.Y0))
	for x := b.X0; x <= b.X1; x++ {
		out = append(out, Point{X: x, Y: b.Y0})
	}
	for y := b.Y0 + 1; y <= b.Y1; y++ {
		out = append(out, Point{X: b.X1, Y: y})
	}
	for x := b.X1 - 1; x >= b.X0; x-- {
		out = append(out, Point{X: x, Y: b.Y1})
	}
	for y := b.Y1 - 1; y > b.Y0; y-- {
		out = append(out, Point{X: b.X0, Y: y})
	}
	return out
}

func (b Rect) Cyclic() bool { return b.X0 != b.X1 && b.Y0 != b.Y1 }

func runStart(ring []Point, usable func(Point) bool, cyclic bool) int {
	n := len(ring)
	for i := range ring {
		if cyclic && usable(ring[i]) && !usable(ring[(i-1+n)%n]) {
			return i
		}
	}
	return 0
}

func (b Rect) Runs(usable func(Point) bool) [][]Point {
	ring := b.Ring()
	start := runStart(ring, usable, b.Cyclic())
	out, cur := [][]Point{}, []Point{}
	for k := range ring {
		p := ring[(start+k)%len(ring)]
		if usable(p) {
			cur = append(cur, p)
			continue
		}
		if len(cur) > 0 {
			out = append(out, cur)
		}
		cur = []Point{}
	}
	if len(cur) > 0 {
		out = append(out, cur)
	}
	return out
}

func PairRun(run []Point) [][2]Point {
	n := len(run) / 2
	front := (n + 1) / 2
	out := make([][2]Point, 0, n)
	for i := 0; i < front; i++ {
		out = append(out, [2]Point{run[2*i], run[2*i+1]})
	}
	for i := 0; i < n-front; i++ {
		j := len(run) - 1 - 2*i
		out = append(out, [2]Point{run[j], run[j-1]})
	}
	return out
}

func (b Rect) SlotOf(a, c Point) StallSlot {
	sl := StallSlot{X: min(a.X, c.X), Y: min(a.Y, c.Y), W: Abs(a.X-c.X) + 1, H: Abs(a.Y-c.Y) + 1}
	switch {
	case sl.H == 1 && sl.Y == b.Y1 && b.Y1 != b.Y0:
		sl.Face = FaceNE
	case sl.H == 1:
		sl.Face = FaceSW
	case sl.X == b.X1 && b.X1 != b.X0:
		sl.Face = FaceNW
	default:
		sl.Face = FaceSE
	}
	return sl
}

func EdgeFaces(side Point) (string, string) {
	switch {
	case side.X > 0:
		return FaceSE, FaceNW
	case side.X < 0:
		return FaceNW, FaceSE
	case side.Y > 0:
		return FaceSW, FaceNE
	case side.Y < 0:
		return FaceNE, FaceSW
	}
	return "", ""
}

func (f Forum) door(p Point) bool {
	c := f.CodexRect()
	inX, inY := p.X >= c.X0 && p.X <= c.X1, p.Y >= c.Y0 && p.Y <= c.Y1
	return (inX && (p.Y == c.Y0-1 || p.Y == c.Y1+1)) || (inY && (p.X == c.X0-1 || p.X == c.X1+1))
}

func (s Site) StallSlots() []StallSlot {
	lo, hi, ok := s.ForumBox()
	if !ok {
		return nil
	}
	box := Rect{X0: lo.X, Y0: lo.Y, X1: hi.X, Y1: hi.Y}
	usable := func(p Point) bool { return s.OnForum(p.X, p.Y) && !s.Forum.door(p) }
	out := []StallSlot{}
	for _, run := range box.Runs(usable) {
		for _, pr := range PairRun(run) {
			out = append(out, box.SlotOf(pr[0], pr[1]))
		}
	}
	return s.orderedSlots(out)
}

type slotRank struct {
	class, key, index int
}

func (s Site) CrowdsStall(sl StallSlot) bool {
	fp := sl.Rect()
	for _, st := range s.Structures {
		if IsStall(st) || st.Kind == StructWarHorn || st.Kind == StructWayfinder {
			continue
		}
		if Footprint(st).Gap(fp) < 2 || Hides(Footprint(st), StructureLift(st.Kind, st.Variant), fp, tallStallLift) {
			return true
		}
	}
	return false
}

func (f Forum) rankOf(sl StallSlot, index int) slotRank {
	codexFace, farFace := EdgeFaces(f.Side)
	c := f.CodexRect()
	cx, cy := 2*sl.X+sl.W-1-(c.X0+c.X1), 2*sl.Y+sl.H-1-(c.Y0+c.Y1)
	along, across := f.Side.X*cx+f.Side.Y*cy, Abs(f.Side.Y*cx+f.Side.X*cy)
	r := slotRank{key: along, index: index}
	switch sl.Face {
	case codexFace:
		r.class, r.key = 1, -across
	case farFace:
		r.class, r.key = 2, -across
	}
	if sl.Face == FaceNE || sl.Face == FaceNW {
		r.class += BackClass
	}
	return r
}

func (a slotRank) before(b slotRank) bool {
	switch {
	case a.class != b.class:
		return a.class < b.class
	case a.key != b.key:
		return a.key < b.key
	}
	return a.index < b.index
}

func (s Site) orderedSlots(slots []StallSlot) []StallSlot {
	ranks := make([]slotRank, len(slots))
	idx := make([]int, len(slots))
	for i, sl := range slots {
		ranks[i], idx[i] = s.Forum.rankOf(sl, i), i
		if s.CrowdsStall(sl) {
			ranks[i].class += CrowdedClass
		}
	}
	sort.SliceStable(idx, func(i, j int) bool { return ranks[idx[i]].before(ranks[idx[j]]) })
	out := make([]StallSlot, len(slots))
	for i, k := range idx {
		out[i] = slots[k]
	}
	return out
}

func (s Site) OverlapsStructure(fp Rect) bool {
	for _, st := range s.Structures {
		if Footprint(st).Gap(fp) == 0 {
			return true
		}
	}
	return false
}

func (s Site) FreeStallSlots() []StallSlot {
	out := []StallSlot{}
	for _, sl := range s.StallSlots() {
		if !s.OverlapsStructure(sl.Rect()) {
			out = append(out, sl)
		}
	}
	return out
}

func (s Site) StallSlotNear(x, y float64) (StallSlot, bool) {
	best, found, bestD := StallSlot{}, false, 0.0
	for _, sl := range s.FreeStallSlots() {
		dx, dy := float64(sl.X)+float64(sl.W-1)/2-x, float64(sl.Y)+float64(sl.H-1)/2-y
		if d := dx*dx + dy*dy; !found || d < bestD {
			best, found, bestD = sl, true, d
		}
	}
	return best, found
}

func (s Site) StallFace(x, y, w, h int) (string, bool) {
	for _, sl := range s.StallSlots() {
		if sl.X == x && sl.Y == y && sl.W == w && sl.H == h {
			return sl.Face, true
		}
	}
	return "", false
}

func (s Site) StallSlotAt(x, y int) (StallSlot, bool) {
	for _, sl := range s.StallSlots() {
		if sl.X == x && sl.Y == y {
			return sl, true
		}
	}
	return StallSlot{}, false
}
