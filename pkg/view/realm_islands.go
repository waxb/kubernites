package view

import (
	"math"
	"math/rand/v2"
	"strings"
)

const HomeIslandID = "home"

const isleGap = 5

const jitterTries = 16

const shapeCell = 0.38

const detailCell = 4.0

const coastFade = 0.9

const fadeSteep = 8.0

const legacyHome = "Hearthmoor"

type IsleSlot struct {
	Fx, Fy float64 `json:"-"`
	Radius int     `json:"-"`
	Jitter int     `json:"-"`
	name   string
	zone   string
}

type isleSpec struct {
	island Island
	radius float64
	biome  biome
}

func layIslands(rng *rand.Rand, plan WorldLayout) []isleSpec {
	out := []isleSpec{newIsle(rng, plan.Home, homeBiome, plan.Width, plan.Height, nil)}
	out[0].island.ID, out[0].island.Owned = HomeIslandID, true
	order := rng.Perm(len(outerBiomes))
	for i, slot := range plan.Outer {
		out = append(out, newIsle(rng, slot, outerBiomes[order[i%len(order)]], plan.Width, plan.Height, out))
	}
	return out
}

func newIsle(rng *rand.Rand, slot IsleSlot, b biome, w, h int, placed []isleSpec) isleSpec {
	radius := float64(slot.Radius)
	center := Point{X: int(slot.Fx * float64(w)), Y: int(slot.Fy * float64(h))}
	for try := 0; try < jitterTries; try++ {
		p := Point{X: center.X + rng.IntN(2*slot.Jitter+1) - slot.Jitter, Y: center.Y + rng.IntN(2*slot.Jitter+1) - slot.Jitter}
		if isleFits(p, radius, w, h, placed) {
			center = p
			break
		}
	}
	name := b.names[rng.IntN(len(b.names))]
	if slot.name != "" {
		name = slot.name
	}
	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	zone := "isle-" + slug
	if slot.zone != "" {
		zone = slot.zone
	}
	return isleSpec{island: Island{ID: slug, Name: name, Zone: zone, Center: center}, radius: radius, biome: b}
}

func isleFits(p Point, radius float64, w, h int, placed []isleSpec) bool {
	r := int(math.Ceil(radius))
	if p.X-r < 2 || p.Y-r < 2 || p.X+r > w-3 || p.Y+r > h-3 {
		return false
	}
	for _, o := range placed {
		if math.Hypot(float64(p.X-o.island.Center.X), float64(p.Y-o.island.Center.Y)) < radius+o.radius+isleGap {
			return false
		}
	}
	return true
}

func (sp isleSpec) beach() float64 {
	return shallowsTop + sp.biome.beachTiles*1.15/sp.radius
}

func (sp isleSpec) bounds(w, h int) Rect {
	r := int(math.Ceil(sp.radius))
	c := sp.island.Center
	return Rect{X0: max(0, c.X-r), Y0: max(0, c.Y-r), X1: min(w-1, c.X+r), Y1: min(h-1, c.Y+r)}
}

func shapeIslands(site *Site, rng *rand.Rand, isles []isleSpec, shaped bool) []uint8 {
	f := newFields(rng, site.span())
	height := make([]float64, len(site.Tiles))
	owner := make([]uint8, len(site.Tiles))
	for i, sp := range isles {
		sp.stamp(site, rng, height, owner, uint8(i+1), shaped)
	}
	inland := make([][]int, len(isles))
	for i, id := range owner {
		if id != 0 && height[i] >= isles[id-1].beach() {
			inland[id-1] = append(inland[id-1], i)
		}
	}
	all := make([]cuts, len(isles))
	for k, sp := range isles {
		all[k] = sp.cutsFor(inland[k], height, f, site.Width)
	}
	for i, id := range owner {
		if id != 0 {
			fv, rv, mv := f.at(i, site.Width)
			site.Tiles[i] = classify(height[i], fv, rv, mv, all[id-1])
		}
	}
	return owner
}

func (sp isleSpec) stamp(site *Site, rng *rand.Rand, height []float64, owner []uint8, id uint8, shaped bool) {
	coast, wide := roundCoast, sp
	if shaped {
		coast = NewCoast(rng, id == 1)
		wide.radius *= CoastCeiling
	}
	box := wide.bounds(site.Width, site.Height)
	span := int(2*wide.radius) + 4
	shape, detail := newNoise(rng, shapeCell*sp.radius, span), newNoise(rng, detailCell, span)
	c := sp.island.Center
	for y := box.Y0; y <= box.Y1; y++ {
		for x := box.X0; x <= box.X1; x++ {
			dx, dy := float64(x-c.X), float64(y-c.Y)
			d := math.Hypot(dx, dy) / (sp.radius * coast.Reach(math.Atan2(dy, dx)))
			if d >= 1 {
				continue
			}
			lx, ly := float64(x-box.X0), float64(y-box.Y0)
			h := 1.05 - 1.15*d + (shape.at(lx, ly)-0.5)*0.55 + (detail.at(lx, ly)-0.5)*0.18 - max(0, d-coastFade)*fadeSteep
			if i := y*site.Width + x; owner[i] == 0 || h > height[i] {
				height[i], owner[i] = h, id
			}
		}
	}
}

func regionsOf(site Site, owner []uint8) []uint8 {
	out := make([]uint8, len(site.Tiles))
	for i, t := range site.Tiles {
		if !IsLand(t) {
			continue
		}
		out[i] = max(owner[i], 1)
	}
	return out
}

func islandsOf(isles []isleSpec) []Island {
	out := make([]Island, len(isles))
	for i, sp := range isles {
		out[i] = sp.island
	}
	return out
}

func (s Site) Island(id string) (Island, bool) {
	for _, is := range s.Islands {
		if is.ID == id {
			return is, true
		}
	}
	return Island{}, false
}

func (s Site) Home() Island {
	home, _ := s.Island(HomeIslandID)
	return home
}

func legacyIslands(s Site) ([]Island, []uint8) {
	region := make([]uint8, len(s.Tiles))
	sumX, sumY, land := 0, 0, 0
	for i, t := range s.Tiles {
		if IsLand(t) {
			region[i] = 1
			sumX, sumY, land = sumX+i%s.Width, sumY+i/s.Width, land+1
		}
	}
	center := Point{X: sumX / max(land, 1), Y: sumY / max(land, 1)}
	slug := strings.ToLower(legacyHome)
	return []Island{{ID: HomeIslandID, Name: legacyHome, Zone: "isle-" + slug, Owned: true, Center: center}}, region
}
