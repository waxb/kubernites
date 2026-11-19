package world

import "github.com/waxb/kubernites/pkg/view"

const (
	RoadNone uint8 = iota
	RoadTrodden
	RoadDirt
	RoadCobbled
)

func RoadLevel(site view.Site, x, y int) uint8 {
	switch site.Tile(x, y) {
	case view.TileWater, view.TileShallows, view.TilePlaza, view.TileForum:
		return RoadNone
	}
	return uint8(min(max(site.RoadAt(x, y), 0), int(RoadCobbled)))
}
