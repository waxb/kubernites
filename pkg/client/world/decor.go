package world

import (
	"math"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/view"
)

type scenery struct {
	name string
	pos  Vec
	lift float64
	wild bool
	flip bool
	tiny bool
	tile int32
}

func mix32(h uint32) uint32 {
	h = (h ^ (h >> 16)) * 0x7feb352d
	h = (h ^ (h >> 15)) * 0x846ca68b
	return h ^ (h >> 16)
}

func tileHash(x, y int) uint32 {
	return mix32(uint32(x)*374761393 ^ uint32(y)*668265263)
}

func coarse(x, y, cell int, seed uint32) float64 {
	fx, fy := float64(x)/float64(cell), float64(y)/float64(cell)
	x0, y0 := math.Floor(fx), math.Floor(fy)
	tx, ty := fx-x0, fy-y0
	at := func(i, j int) float64 {
		return float64(mix32(uint32(i)*73856093^uint32(j)*19349663^seed)%1000) / 1000
	}
	ix, iy := int(x0), int(y0)
	a, b, c, d := at(ix, iy), at(ix+1, iy), at(ix, iy+1), at(ix+1, iy+1)
	return a + (b-a)*tx + (c-a)*ty + (a-b-c+d)*tx*ty
}

func Wet(k view.TileKind) bool {
	return k == view.TileWater || k == view.TileShallows
}

func ShoreMask(site view.Site, x, y int) int {
	if Wet(site.Tile(x, y)) {
		return 0
	}
	mask := 0
	for i, d := range [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}} {
		if Wet(site.Tile(x+d[0], y+d[1])) {
			mask |= 1 << i
		}
	}
	return mask
}

func jitterAt(h uint32, shift uint, spread float64) float64 {
	return (float64((h>>shift)%1000)/1000 - 0.5) * 2 * spread
}

func place(name string, x, y int, h uint32, spread float64) scenery {
	return scenery{name: name, pos: Vec{float64(x) + jitterAt(h, 4, spread), float64(y) + jitterAt(h, 14, spread)}, flip: h&1 == 1}
}

func decorName(kind string, h uint32) string {
	return gfx.DecorName(kind, int(h>>20%art.DecorVariants))
}

func sceneryFor(site view.Site, hb *habitat, x, y int) []scenery {
	k := site.Tile(x, y)
	if hb.BlockAt(x, y) == BlockFootprint || k == view.TileRoad || k == view.TilePlaza || RoadLevel(site, x, y) > RoadNone || charterSpot(site, x, y) {
		return nil
	}
	h := tileHash(x, y)
	near := hb.BlockAt(x, y) == blockNear
	var out []scenery
	switch k {
	case view.TileForest:
		out = forestScenery(site, hb, x, y, h, near)
	case view.TileGrass, view.TileMeadow:
		out = fieldScenery(site, hb, x, y, h, near)
	case view.TileRock:
		out = rockScenery(x, y, h, near)
	case view.TileSand:
		out = sandScenery(site, x, y, h)
	case view.TileShallows:
		if ShoreMask(site, x, y) == 0 && landBeside(site, x, y) && h%9 == 0 {
			out = []scenery{place(decorName("reeds", h), x, y, h, 0.3)}
		}
	}
	lift := float64(site.ElevationAt(x, y))
	wild := Wild(site, x, y)
	kept := out[:0]
	for _, s := range out {
		if hb.veiled(x, y) && tall(s.name) {
			continue
		}
		s.lift, s.wild, s.tile = lift, wild, int32(y*site.Width+x)
		kept = append(kept, s)
	}
	return kept
}

func tall(name string) bool {
	return len(name) > 11 && name[:11] == "decor/tree-"
}

func charterSpot(site view.Site, x, y int) bool {
	for _, is := range site.Islands {
		if !is.Owned && Abs(is.Center.X-x) <= 1 && Abs(is.Center.Y-y) <= 1 {
			return true
		}
	}
	return false
}

func Abs(v int) int {
	return max(v, -v)
}

func landBeside(site view.Site, x, y int) bool {
	for _, d := range [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}} {
		if !Wet(site.Tile(x+d[0], y+d[1])) {
			return true
		}
	}
	return false
}

func forestScenery(site view.Site, hb *habitat, x, y int, h uint32, near bool) []scenery {
	town := hb.townAt(x, y)
	switch {
	case hb.inPlot(x, y) || near:
		return nil
	case town > stumpTown:
		if h%3 == 0 {
			return []scenery{place(decorName("stump", h), x, y, h, 0.3)}
		}
		return nil
	}
	if h%83 == 0 && forestInterior(site, x, y) {
		return []scenery{place(decorName("tree-old", h), x, y, h, 0.15)}
	}
	count := 1 + int(h>>3%2)
	out := make([]scenery, 0, count+1)
	for i := 0; i < count; i++ {
		hi := mix32(h + uint32(i)*0x9e3779b9)
		s := place(treeSpecies(x, y, hi, site.ElevationAt(x, y)), x, y, hi, 0.34)
		out = append(out, s)
	}
	if h%11 == 0 {
		out = append(out, place(decorName([]string{"log", "stump", "boulder", "bush"}[h>>7%4], h), x, y, mix32(h), 0.4))
	}
	return out
}

func forestInterior(site view.Site, x, y int) bool {
	for _, d := range [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}} {
		if site.Tile(x+d[0], y+d[1]) != view.TileForest {
			return false
		}
	}
	return true
}

func treeSpecies(x, y int, h uint32, elevation uint8) string {
	conifer := coarse(x, y, 9, 1) + float64(elevation)*0.12
	autumn := coarse(x, y, 11, 2)
	birch := coarse(x, y, 7, 3)
	r := float64(h>>9%1000) / 1000
	switch {
	case r < (conifer-0.38)*2.2:
		if h>>2%10 < 3 {
			return decorName("tree-spruce", h)
		}
		return decorName("tree-pine", h)
	case birch > 0.64 && r < 0.7:
		return decorName("tree-birch", h)
	case autumn > 0.68 && r < 0.75:
		return decorName("tree-maple", h)
	}
	if autumn > 0.6 {
		return gfx.DecorName("tree-oak", 2)
	}
	return gfx.DecorName("tree-oak", int(h>>21%2))
}

func fieldScenery(site view.Site, hb *habitat, x, y int, h uint32, near bool) []scenery {
	if hb.inPlot(x, y) {
		return nil
	}
	fringe := nextToForest(site, x, y)
	roll := h >> 8 % 1000
	switch {
	case hb.settled(x, y):
		return settledScenery(x, y, h, roll)
	case !near && roll < 2 && hb.townAt(x, y) < 0.3:
		return []scenery{place(decorName("tree-old", h), x, y, h, 0.1)}
	case !near && fringe && roll < 130:
		return []scenery{place(treeSpecies(x, y, h, site.ElevationAt(x, y)), x, y, h, 0.35)}
	case fringe && roll < 260:
		return []scenery{place(decorName("bush", h), x, y, h, 0.35)}
	case !near && roll < 14:
		return []scenery{place(treeSpecies(x, y, h, site.ElevationAt(x, y)), x, y, h, 0.3)}
	case roll < 40:
		return []scenery{place(decorName("bush", h), x, y, h, 0.35)}
	case roll < 52:
		return []scenery{tinyAt(place(decorName("rock", h), x, y, h, 0.35))}
	case !near && roll < 55:
		return []scenery{place(decorName("boulder", h), x, y, h, 0.25)}
	}
	return nil
}

func settledScenery(x, y int, h, roll uint32) []scenery {
	if roll >= 40 && roll < 52 {
		return []scenery{tinyAt(place(decorName("rock", h), x, y, h, 0.35))}
	}
	return nil
}

func tinyAt(s scenery) scenery {
	s.tiny = true
	return s
}

func nextToForest(site view.Site, x, y int) bool {
	for _, d := range [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}} {
		if site.Tile(x+d[0], y+d[1]) == view.TileForest {
			return true
		}
	}
	return false
}

func rockScenery(x, y int, h uint32, near bool) []scenery {
	roll := h >> 8 % 100
	switch {
	case near && roll < 30:
		return []scenery{tinyAt(place(decorName("rock", h), x, y, h, 0.3))}
	case near:
		return nil
	case roll < 26:
		return []scenery{place(decorName("boulder", h), x, y, h, 0.25)}
	case roll < 50:
		return []scenery{tinyAt(place(decorName("rock", h), x, y, h, 0.35))}
	case roll < 58:
		return []scenery{place(decorName("tree-pine", h), x, y, h, 0.3)}
	case roll < 62:
		return []scenery{place(decorName("tree-spruce", h), x, y, h, 0.3)}
	}
	return nil
}

func sandScenery(site view.Site, x, y int, h uint32) []scenery {
	roll := h >> 8 % 100
	switch {
	case ShoreMask(site, x, y) != 0 && roll < 18:
		return []scenery{place(decorName("reeds", h), x, y, h, 0.3)}
	case roll < 3:
		return []scenery{tinyAt(place(decorName("rock", h), x, y, h, 0.3))}
	case roll < 5:
		return []scenery{place(gfx.DecorName("log", 0), x, y, h, 0.3)}
	}
	return nil
}
