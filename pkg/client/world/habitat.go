package world

import (
	"math"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	blockFree uint8 = iota
	blockNear
	BlockFootprint
)

const (
	townReach   = 7.0
	keepYard    = 1.0
	yardFade    = 3.0
	stumpTown   = 0.6
	wearReach   = 2
	veilRows    = 5
	landShyness = 2
)

var wearNoise = [art.RoadLevels + 1]float32{0, 0.4, 0.62, 0.85}

type habitat struct {
	w, h    int
	blocked []uint8
	town    []float32
	plot    []bool
	veil    []bool
	land    []int16
	shy     []bool
}

const (
	fnvOffset64 fnv64 = 14695981039346656037
	fnvPrime64  fnv64 = 1099511628211
)

type fnv64 uint64

func (h fnv64) str(s string) fnv64 {
	for i := 0; i < len(s); i++ {
		h = (h ^ fnv64(s[i])) * fnvPrime64
	}
	return h
}

func (h fnv64) bytes(b ...byte) fnv64 {
	for _, c := range b {
		h = (h ^ fnv64(c)) * fnvPrime64
	}
	return h
}

func StructureSig(site view.Site) uint64 {
	h := fnvOffset64
	for _, s := range site.Structures {
		h = h.str(s.Kind).bytes(byte(s.X), byte(s.X>>8), byte(s.Y), byte(s.Y>>8), byte(s.W), byte(s.H), byte(s.Plot), byte(s.Territory))
	}
	for _, hv := range site.Havens {
		h = h.str(hv.ID).bytes(byte(hv.Quay.X), byte(hv.Quay.Y), byte(hv.Tier), byte(hv.Pier), boolByte(hv.Open))
	}
	for _, is := range site.Islands {
		h = h.str(is.ID).bytes(boolByte(is.Owned))
	}
	return uint64(h)
}

func boolByte(v bool) byte {
	if v {
		return 1
	}
	return 0
}

func BuildHabitat(site view.Site, land *art.Land) *habitat {
	n := site.Width * site.Height
	hb := &habitat{w: site.Width, h: site.Height, blocked: make([]uint8, n), town: make([]float32, n), plot: make([]bool, n), veil: make([]bool, n),
		land: ownerOf(site, site.Territories())}
	hb.shy = widen(hb.land, site.Width, site.Height, landShyness)
	for _, s := range site.Structures {
		hb.stamp(s)
		hb.shade(s)
	}
	for _, hv := range SiteHavens(site) {
		hb.stampHaven(PlanHaven(site, hv))
	}
	for _, l := range PlanLanes(site) {
		hb.stampClear(append(l.Home.Claims(), l.Away.Claims()...))
	}
	for _, p := range site.Plots {
		if i, ok := hb.index(p.X, p.Y); ok {
			hb.blocked[i] = BlockFootprint
		}
	}
	for i, lv := range land.Road {
		if lv > 0 {
			hb.spreadWear(i%site.Width, i/site.Width, wearNoise[min(lv, art.RoadLevels)])
		}
	}
	return hb
}

func (hb *habitat) index(x, y int) (int, bool) {
	if x < 0 || y < 0 || x >= hb.w || y >= hb.h {
		return 0, false
	}
	return y*hb.w + x, true
}

func (hb *habitat) raise(i int, v float32) {
	hb.town[i] = max(hb.town[i], v)
}

func (hb *habitat) stamp(s view.Structure) {
	lo, hi := Footprint(s)
	reach := int(townReach)
	for y := lo.Y - reach; y <= hi.Y+reach; y++ {
		for x := lo.X - reach; x <= hi.X+reach; x++ {
			i, ok := hb.index(x, y)
			if !ok {
				continue
			}
			gap := FootprintGap(lo, hi, x, y)
			switch {
			case gap == 0:
				hb.blocked[i] = BlockFootprint
			case gap <= 1:
				hb.blocked[i] = max(hb.blocked[i], blockNear)
			}
			hb.raise(i, townShare(s, gap))
			if s.Kind == view.StructKeep && gap <= keepYard {
				hb.plot[i] = true
			}
		}
	}
}

func townShare(s view.Structure, gap float64) float32 {
	if s.Kind != view.StructKeep {
		return float32(max(0, 1-gap/townReach))
	}
	return float32(min(1, max(0, 1-(gap-keepYard)/yardFade)))
}

func (hb *habitat) tamed(x, y int) bool {
	i, ok := hb.index(x, y)
	return ok && (hb.plot[i] || i < len(hb.shy) && hb.shy[i])
}

func widen(land []int16, w, h, reach int) []bool {
	out := make([]bool, len(land))
	for i, o := range land {
		if o == 0 {
			continue
		}
		x, y := i%w, i/w
		for dy := max(-reach, -y); dy <= reach && y+dy < h; dy++ {
			for dx := max(-reach, -x); dx <= reach && x+dx < w; dx++ {
				out[(y+dy)*w+x+dx] = true
			}
		}
	}
	return out
}

func (hb *habitat) settled(x, y int) bool {
	i, ok := hb.index(x, y)
	return ok && i < len(hb.land) && hb.land[i] != 0
}

func (hb *habitat) stampHaven(p HavenPlan) {
	tiles := append([]Point{}, p.Deck...)
	for _, v := range p.Pier {
		tiles = append(tiles, TileOf(v))
	}
	hb.stampTiles(tiles)
	if len(p.Deck) > 0 {
		hb.shade(deckBounds(p.Deck))
	}
}

func (hb *habitat) stampClear(tiles []Point) {
	for _, t := range tiles {
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if i, ok := hb.index(t.X+dx, t.Y+dy); ok {
					hb.blocked[i] = BlockFootprint
				}
			}
		}
	}
}

func (hb *habitat) stampTiles(tiles []Point) {
	for _, t := range tiles {
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if i, ok := hb.index(t.X+dx, t.Y+dy); ok {
					hb.blocked[i] = max(hb.blocked[i], blockNear)
				}
			}
		}
	}
	for _, t := range tiles {
		if i, ok := hb.index(t.X, t.Y); ok {
			hb.blocked[i] = BlockFootprint
		}
	}
}

func deckBounds(deck []Point) view.Structure {
	lo, hi := deck[0], deck[0]
	for _, d := range deck[1:] {
		lo = Point{min(lo.X, d.X), min(lo.Y, d.Y)}
		hi = Point{max(hi.X, d.X), max(hi.Y, d.Y)}
	}
	return view.Structure{X: lo.X, Y: lo.Y, W: hi.X - lo.X + 1, H: hi.Y - lo.Y + 1}
}

func (hb *habitat) shade(s view.Structure) {
	lo, hi := Footprint(s)
	front := hi.X + hi.Y
	for d := 1; d <= veilRows; d++ {
		for col := lo.X - hi.Y - 1; col <= hi.X-lo.Y+1; col++ {
			if (front+d+col)%2 != 0 {
				continue
			}
			x, y := (front+d+col)/2, (front+d-col)/2
			if i, ok := hb.index(x, y); ok {
				hb.veil[i] = true
			}
		}
	}
}

func (hb *habitat) veiled(x, y int) bool {
	i, ok := hb.index(x, y)
	return ok && hb.veil[i]
}

func FootprintGap(lo, hi Point, x, y int) float64 {
	dx := max(lo.X-x, 0, x-hi.X)
	dy := max(lo.Y-y, 0, y-hi.Y)
	return math.Hypot(float64(dx), float64(dy))
}

func (hb *habitat) spreadWear(x, y int, v float32) {
	for dy := -wearReach; dy <= wearReach; dy++ {
		for dx := -wearReach; dx <= wearReach; dx++ {
			if i, ok := hb.index(x+dx, y+dy); ok {
				d := math.Hypot(float64(dx), float64(dy))
				hb.raise(i, v*float32(max(0, 1-d/float64(wearReach+1))))
			}
		}
	}
}

func (hb *habitat) townAt(x, y int) float32 {
	if i, ok := hb.index(x, y); ok {
		return hb.town[i]
	}
	return 0
}

func (hb *habitat) BlockAt(x, y int) uint8 {
	if i, ok := hb.index(x, y); ok {
		return hb.blocked[i]
	}
	return blockFree
}

func (hb *habitat) inPlot(x, y int) bool {
	i, ok := hb.index(x, y)
	return ok && hb.plot[i]
}
