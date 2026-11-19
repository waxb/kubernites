package world

import (
	"github.com/waxb/kubernites/pkg/view"
)

type Point struct{ X, Y int }

type Vec struct{ X, Y float64 }

func (p Point) Center() Vec { return Vec{float64(p.X), float64(p.Y)} }

func Footprint(s view.Structure) (Point, Point) {
	return Point{s.X, s.Y}, Point{s.X + max(s.W, 1) - 1, s.Y + max(s.H, 1) - 1}
}

func Covers(s view.Structure, x, y int) bool {
	lo, hi := Footprint(s)
	return x >= lo.X && x <= hi.X && y >= lo.Y && y <= hi.Y
}

func Door(s view.Structure) Vec {
	if f, ok := stallFrameOf(s); ok {
		return f.door()
	}
	_, hi := Footprint(s)
	return Vec{float64(hi.X) + .75, float64(hi.Y) + .75}
}

func Anchor(s view.Structure) Vec {
	_, hi := Footprint(s)
	return Vec{float64(hi.X) + .5, float64(hi.Y) + .5}
}

func KeepForNode(site view.Site, node string) (view.Structure, bool) {
	for _, s := range site.Structures {
		if s.Kind == view.StructKeep && s.Object.Name == node {
			return s, true
		}
	}
	return view.Structure{}, false
}

func FirstOfKind(site view.Site, kind string, readyOnly bool) (view.Structure, bool) {
	for _, s := range site.Structures {
		if s.Kind == kind && (!readyOnly || s.State == view.StateReady) {
			return s, true
		}
	}
	return view.Structure{}, false
}

func Homeable(k view.TileKind) bool {
	return k == view.TileGrass || k == view.TileMeadow || k == view.TileSand
}

func Occupied(site view.Site, x, y int) bool {
	for _, s := range site.Structures {
		if Covers(s, x, y) {
			return true
		}
	}
	return false
}

func NearAnyStructure(site view.Site, x, y int) bool {
	for _, s := range site.Structures {
		if nearFootprint(s, x, y) {
			return true
		}
	}
	return false
}

func nearFootprint(s view.Structure, x, y int) bool {
	lo, hi := Footprint(s)
	return x >= lo.X-1 && x <= hi.X+1 && y >= lo.Y-1 && y <= hi.Y+1
}

func PendingSpot(site view.Site, index int) Vec {
	return pendingSpotAmong(site, campSpots(site), index)
}

func pendingSpotAmong(site view.Site, spots []Vec, index int) Vec {
	if len(spots) > 0 {
		s, ring := spots[max(index, 0)%len(spots)], max(index, 0)/len(spots)
		d := campJitter[ring%len(campJitter)]
		return Vec{s.X + d.X, s.Y + d.Y}
	}
	base := Vec{float64(site.Plaza.X), float64(site.Plaza.Y)}
	if s, ok := FirstOfKind(site, view.StructKeep, false); ok {
		lo, _ := Footprint(s)
		base = Vec{float64(lo.X) - 2, float64(lo.Y) + 1}
	}
	return Vec{base.X - float64(index%4)*.8, base.Y + float64(index/4)*.8}
}
