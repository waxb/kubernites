package world

import (
	"cmp"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	wornVariants = 3
	wornAlpha    = .85
	wornCap      = 1.25
)

type wornTiles struct {
	last  float64
	sig   uint64
	fit   map[Point]bool
	ready bool
}

func (l *living) wear(figs []Figure, sc Scene) {
	dt := 0.0
	if l.footed.ready {
		dt = min(max(sc.Time-l.footed.last, 0), 1)
	}
	l.footed.last, l.footed.ready = sc.Time, true
	if dt == 0 {
		return
	}
	for _, f := range figs {
		if f.Pose.Action != "walk" {
			continue
		}
		p := TileOf(f.Pose.Pos)
		if k := sc.Site.Tile(p.X, p.Y); k != view.TileGrass && k != view.TileMeadow && k != view.TileSand {
			continue
		}
		l.worn[p] = min(l.worn[p]+dt, wornFull*wornCap)
	}
}

func (l *living) fits(site view.Site, p Point) bool {
	sig := StructureSig(site)
	if l.footed.fit == nil || l.footed.sig != sig {
		l.footed.sig, l.footed.fit = sig, map[Point]bool{}
	}
	ok, seen := l.footed.fit[p]
	if !seen {
		ok = RoadLevel(site, p.X, p.Y) == RoadNone && !covered(site, p.X, p.Y)
		l.footed.fit[p] = ok
	}
	return ok
}

func (r *Renderer) drawWorn(dst *ebiten.Image, cam render.Camera, sc Scene, l *living, x0, y0, x1, y1 int, h *Hamlet) {
	zoom := cam.Scale()
	for _, p := range wornInView(l.worn, x0, y0, x1, y1) {
		level := WornLevel(l.worn[p])
		if level <= 0 || h.occAt(p) != occFree || !l.fits(sc.Site, p) {
			continue
		}
		v := int(tileHash(p.X, p.Y) % wornVariants)
		piece := l.piece("worn"+string(rune('0'+v)), func() art.Piece {
			return art.Piece{Img: art.PaintWornTile(v), AX: art.LivingTileW / 2, AY: art.LivingTileH / 2}
		})
		sx, sy := cam.GridToScreenElevated(float64(p.X), float64(p.Y), float64(sc.Site.ElevationAt(p.X, p.Y)))
		piece.draw(dst, sx, sy, zoom, render.SpriteOptions{Alpha: float32(level * wornAlpha)})
	}
}

func wornInView(worn map[Point]float64, x0, y0, x1, y1 int) []Point {
	tiles := []Point{}
	for p := range worn {
		if p.X >= x0 && p.X <= x1 && p.Y >= y0 && p.Y <= y1 {
			tiles = append(tiles, p)
		}
	}
	slices.SortFunc(tiles, func(a, b Point) int {
		return cmp.Compare(render.DepthKey(float64(a.X), float64(a.Y)), render.DepthKey(float64(b.X), float64(b.Y)))
	})
	return tiles
}
