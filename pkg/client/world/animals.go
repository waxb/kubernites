package world

import (
	"math"

	"github.com/waxb/kubernites/pkg/view"
)

type species struct {
	roles  []string
	herd   int
	radius float64
	speed  float64
	period float64
	shy    float32
	odds   uint32
	graze  float64
}

var (
	deerKind   = species{roles: []string{"deer", "deer", "stag"}, herd: 3, radius: 2.4, speed: 1.1, period: 8, shy: 0.22, odds: 70, graze: 0.7}
	rabbitKind = species{roles: []string{"rabbit"}, herd: 2, radius: 1.4, speed: 2.2, period: 4.5, shy: 0.42, odds: 110, graze: 0.5}
	foxKind    = species{roles: []string{"fox"}, herd: 1, radius: 3, speed: 1.6, period: 6, shy: 0.28, odds: 150, graze: 0.35}
	boarKind   = species{roles: []string{"boar"}, herd: 2, radius: 2, speed: 0.9, period: 8, shy: 0.3, odds: 130, graze: 0.6}
	sheepKind  = species{roles: []string{"sheep"}, herd: 4, radius: 1.5, speed: 0.5, period: 9, shy: 0.85, odds: 190, graze: 0.8}
)

type critter struct {
	role   string
	home   Vec
	kind   *species
	seed   uint32
	phase  float64
	period float64
}

func herdOf(site view.Site, hb *habitat, kind *species, x, y int, h uint32) []critter {
	out := make([]critter, 0, kind.herd)
	for i := 0; i < kind.herd; i++ {
		hi := mix32(h + uint32(i)*0x85ebca6b)
		home := Vec{float64(x) + jitterAt(hi, 3, 0.7), float64(y) + jitterAt(hi, 13, 0.7)}
		if !roams(site, hb, home, kind.shy) {
			home = Vec{float64(x), float64(y)}
		}
		out = append(out, critter{
			role: kind.roles[i%len(kind.roles)], home: home, kind: kind, seed: hi,
			phase: float64(hi%977) / 97.7, period: kind.period * (0.8 + float64(hi>>10%40)/100),
		})
	}
	return out
}

func critterSpecies(site view.Site, hb *habitat, x, y int, h uint32) *species {
	town := hb.townAt(x, y)
	roll := h >> 5
	switch site.Tile(x, y) {
	case view.TileMeadow:
		if town < sheepKind.shy && roll%sheepKind.odds == 0 {
			return &sheepKind
		}
		fallthrough
	case view.TileGrass:
		if town < deerKind.shy && nearForest(site, x, y, 2) && roll%deerKind.odds == 1 {
			return &deerKind
		}
		if town < rabbitKind.shy && roll%rabbitKind.odds == 2 {
			return &rabbitKind
		}
	case view.TileForest:
		if town < boarKind.shy && roll%boarKind.odds == 3 {
			return &boarKind
		}
		if town < foxKind.shy && roll%foxKind.odds == 4 {
			return &foxKind
		}
	}
	return nil
}

func nearForest(site view.Site, x, y, r int) bool {
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if site.Tile(x+dx, y+dy) == view.TileForest {
				return true
			}
		}
	}
	return false
}

func crittersFor(site view.Site, hb *habitat, x, y int) []critter {
	if hb.BlockAt(x, y) != blockFree || hb.tamed(x, y) || RoadLevel(site, x, y) > RoadNone {
		return nil
	}
	h := mix32(tileHash(x, y) ^ 0xa511e9b3)
	kind := critterSpecies(site, hb, x, y, h)
	if kind == nil {
		return nil
	}
	return herdOf(site, hb, kind, x, y, h)
}

func roams(site view.Site, hb *habitat, v Vec, shy float32) bool {
	x, y := int(math.Floor(v.X+0.5)), int(math.Floor(v.Y+0.5))
	k := site.Tile(x, y)
	return !Wet(k) && k != view.TilePlaza && k != view.TileForum && hb.BlockAt(x, y) == blockFree && !hb.tamed(x, y) && hb.townAt(x, y) < shy+0.12
}

func (a critter) waypoint(site view.Site, hb *habitat, k int) Vec {
	h := mix32(a.seed ^ uint32(k)*0x9e3779b9)
	r := a.kind.radius
	p := Vec{a.home.X + jitterAt(h, 2, r), a.home.Y + jitterAt(h, 12, r)}
	if !roams(site, hb, p, a.kind.shy) || !roams(site, hb, Lerp(a.home, p, .5), a.kind.shy) {
		return a.home
	}
	return p
}

func (a critter) Pose(site view.Site, hb *habitat, t float64, resting bool) Pose {
	if resting {
		return Pose{Pos: a.home, Action: "idle", Dir: []string{"se", "sw"}[a.seed%2], Frame: Frame(t+a.phase, 1.5)}
	}
	clock := t + a.phase
	k := int(math.Floor(clock / a.period))
	local := clock - float64(k)*a.period
	from, to := a.waypoint(site, hb, k-1), a.waypoint(site, hb, k)
	travel := Dist(from, to) / a.kind.speed
	dir := Facing(from, to)
	if local < travel {
		return Pose{Pos: Lerp(from, to, local/travel), Action: "walk", Dir: dir, Frame: Frame(clock, 8)}
	}
	action := "idle"
	if float64(mix32(a.seed+uint32(k))%100)/100 < a.kind.graze {
		action = "work"
	}
	return Pose{Pos: to, Action: action, Dir: dir, Frame: Frame(clock, 3)}
}
