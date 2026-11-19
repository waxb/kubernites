package world

import (
	"hash/fnv"
	"math"
	"sort"
	"strconv"

	"github.com/waxb/kubernites/pkg/view"
)

const (
	landBaseGiB   = 4.0
	landBaseReach = 2.0
	berryPlots    = 2
	maxFieldPlots = 12
	defaultCores  = 2
)

type FieldPlot struct {
	At   view.Point
	Crop string
}

type LandTile struct {
	At    view.Point
	Share float64
}

type KeepPatch struct {
	Keep   string
	Center Vec
	Reach  float64
	Land   []LandTile
	Fields []FieldPlot
}

func KeepPatches(site view.Site, homes map[string]Vec) []KeepPatch {
	taken := hutTiles(homes)
	out := []KeepPatch{}
	for _, s := range site.Structures {
		if s.Kind != view.StructKeep || s.State != view.StateReady {
			continue
		}
		out = append(out, keepPatch(site, s, taken))
	}
	return out
}

func hutTiles(homes map[string]Vec) map[view.Point]bool {
	out := make(map[view.Point]bool, len(homes))
	for _, h := range homes {
		out[view.Point{X: int(math.Floor(h.X)), Y: int(math.Floor(h.Y))}] = true
	}
	return out
}

func keepPatch(site view.Site, s view.Structure, taken map[view.Point]bool) KeepPatch {
	mem := float64(s.MemoryGiB)
	if mem <= 0 {
		mem = landBaseGiB
	}
	half := float64(max(s.W, s.H, 1)) / 2
	p := KeepPatch{Keep: s.ID, Center: FootprintCentre(s), Reach: half + landBaseReach*math.Sqrt(mem/landBaseGiB)}
	free := []LandTile{}
	span := int(math.Ceil(p.Reach)) + 1
	cx, cy := int(math.Floor(p.Center.X)), int(math.Floor(p.Center.Y))
	for y := cy - span; y <= cy+span; y++ {
		for x := cx - span; x <= cx+span; x++ {
			d := math.Hypot(float64(x)+.5-p.Center.X, float64(y)+.5-p.Center.Y)
			if d > p.Reach || x < 0 || y < 0 || x >= site.Width || y >= site.Height || site.Tile(x, y) == view.TileWater {
				continue
			}
			tile := LandTile{At: view.Point{X: x, Y: y}, Share: d / p.Reach}
			p.Land = append(p.Land, tile)
			if !taken[tile.At] && !covered(site, x, y) && site.Tile(x, y) != view.TileForest {
				free = append(free, tile)
			}
		}
	}
	p.Fields = fieldsOn(free, s)
	return p
}

func covered(site view.Site, x, y int) bool {
	for _, s := range site.Structures {
		if Covers(s, x, y) {
			return true
		}
	}
	return false
}

func fieldsOn(free []LandTile, s view.Structure) []FieldPlot {
	sort.SliceStable(free, func(i, j int) bool {
		if free[i].Share != free[j].Share {
			return free[i].Share < free[j].Share
		}
		return fieldHash(s.ID, free[i].At) < fieldHash(s.ID, free[j].At)
	})
	cores := s.CPU
	if cores <= 0 {
		cores = defaultCores
	}
	n := min(cores, maxFieldPlots, len(free))
	out := make([]FieldPlot, 0, n)
	for i := range n {
		crop := "wheat"
		if i < berryPlots {
			crop = "berries"
		}
		out = append(out, FieldPlot{At: free[i].At, Crop: crop})
	}
	return out
}

func fieldHash(seed string, p view.Point) uint32 {
	h := fnv.New32a()
	h.Write([]byte(seed + ":" + strconv.Itoa(p.X) + "," + strconv.Itoa(p.Y)))
	return h.Sum32()
}
