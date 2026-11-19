package world

import (
	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/view"
)

var groundOf = map[view.TileKind]art.Ground{
	view.TileWater: art.GroundWater, view.TileShallows: art.GroundShallows, view.TileSand: art.GroundSand,
	view.TileGrass: art.GroundGrass, view.TileMeadow: art.GroundMeadow, view.TileForest: art.GroundForest,
	view.TileRock: art.GroundRock, view.TileRoad: art.GroundRoad, view.TilePlaza: art.GroundPlaza,
}

func LandOf(site view.Site) *art.Land {
	l := art.NewLand(site.Width, site.Height)
	plan := TownPlan(site)
	for y := 0; y < site.Height; y++ {
		for x := 0; x < site.Width; x++ {
			i := y*site.Width + x
			g, ok := groundOf[site.Tile(x, y)]
			if !ok {
				g = art.GroundGrass
			}
			l.Ground[i] = g
			l.Height[i] = site.ElevationAt(x, y)
			l.Road[i] = plan[i]
			l.Wild[i] = Wild(site, x, y)
		}
	}
	l.Prepare()
	return l
}

func Wild(site view.Site, x, y int) bool {
	island, ok := site.IslandAt(x, y)
	return ok && !island.Owned
}
