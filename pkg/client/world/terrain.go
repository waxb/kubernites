package world

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

var tileColors = map[view.TileKind]color.RGBA{
	view.TileWater: {52, 104, 138, 255}, view.TileShallows: {78, 142, 160, 255}, view.TileSand: {214, 196, 140, 255},
	view.TileGrass: {104, 150, 72, 255}, view.TileMeadow: {128, 168, 82, 255}, view.TileForest: {70, 112, 60, 255},
	view.TileRock: {128, 126, 118, 255}, view.TileRoad: {170, 132, 88, 255}, view.TilePlaza: {188, 172, 140, 255},
	view.TileForum: {226, 218, 196, 255},
}

func TileColor(k view.TileKind) color.RGBA {
	if c, ok := tileColors[k]; ok {
		return c
	}
	return tileColors[view.TileGrass]
}

func (r *Renderer) drawTerrain(dst *ebiten.Image, cam render.Camera, sc Scene, screen image.Rectangle) {
	if r.Land == nil {
		r.Land = newLandCache()
	}
	drawSea(dst, r.Lib, cam, screen, sc.Time)
	if sc.Site.Width == 0 {
		return
	}
	r.Land.sync(sc.Site)
	r.Land.view, r.Land.blocking = screen, !r.Lib.Async()
	keys := r.Land.visible(cam, screen)
	for i, img := range r.Land.images(keys) {
		if img != nil {
			drawChunk(dst, img, art.ChunkBounds(keys[i].cx, keys[i].cy), cam)
		}
	}
}

func drawChunk(dst, img *ebiten.Image, bounds image.Rectangle, cam render.Camera) {
	z := cam.Scale()
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	op.GeoM.Scale(z, z)
	x := float64(cam.OriginX) + (float64(bounds.Min.X)-cam.X)*z
	y := float64(cam.OriginY) + (float64(bounds.Min.Y)-cam.Y)*z
	op.GeoM.Translate(math.Round(x), math.Round(y))
	dst.DrawImage(img, op)
}

const seaFPS = 4

func drawSea(dst *ebiten.Image, lib *gfx.Library, cam render.Camera, screen image.Rectangle, t float64) {
	sheet, reg, ok := lib.Lookup(art.SeaName(int(t * seaFPS)))
	if !ok {
		dst.Fill(tileColors[view.TileWater])
		return
	}
	sub := sheet.SubImage(reg.Rect()).(*ebiten.Image)
	z := cam.Scale()
	w := worldRect(cam, screen)
	x0 := int(math.Floor(float64(w.Min.X)/art.SeaWidth)) * art.SeaWidth
	y0 := int(math.Floor(float64(w.Min.Y)/art.SeaHeight)) * art.SeaHeight
	for y := y0; y < w.Max.Y; y += art.SeaHeight {
		for x := x0; x < w.Max.X; x += art.SeaWidth {
			op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
			op.GeoM.Scale(z, z)
			op.GeoM.Translate(math.Round(float64(cam.OriginX)+(float64(x)-cam.X)*z), math.Round(float64(cam.OriginY)+(float64(y)-cam.Y)*z))
			dst.DrawImage(sub, op)
		}
	}
}

func drawDiamond(dst *ebiten.Image, sx, sy, zoom float64, c color.RGBA) {
	hw, hh := float32(render.TileWidth/2*zoom), float32(render.TileHeight/2*zoom)
	var path vector.Path
	x, y := float32(sx), float32(sy)
	path.MoveTo(x, y-hh)
	path.LineTo(x+hw, y)
	path.LineTo(x, y+hh)
	path.LineTo(x-hw, y)
	path.Close()
	op := &vector.FillOptions{}
	dop := &vector.DrawPathOptions{}
	dop.ColorScale.ScaleWithColor(c)
	vector.FillPath(dst, &path, op, dop)
}

func outlineDiamond(dst *ebiten.Image, cam render.Camera, x0, y0, x1, y1 float64, elev float64, c color.RGBA) {
	strokeDiamond(dst, cam, x0, y0, x1, y1, elev, c, 1)
}

func strokeDiamond(dst *ebiten.Image, cam render.Camera, x0, y0, x1, y1 float64, elev float64, c color.RGBA, width float32) {
	corners := [][2]float64{{x0 - .5, y0 - .5}, {x1 + .5, y0 - .5}, {x1 + .5, y1 + .5}, {x0 - .5, y1 + .5}}
	pts := make([][2]float32, len(corners))
	for i, p := range corners {
		sx, sy := cam.GridToScreenElevated(p[0], p[1], elev)
		pts[i] = [2]float32{float32(sx), float32(sy)}
	}
	for i := range pts {
		a, b := pts[i], pts[(i+1)%len(pts)]
		vector.StrokeLine(dst, a[0], a[1], b[0], b[1], width, c, false)
	}
}

func fillFootprint(dst *ebiten.Image, cam render.Camera, x0, y0, x1, y1 int, c color.RGBA) {
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			sx, sy := cam.GridToScreen(float64(x), float64(y))
			drawDiamond(dst, sx, sy, cam.Scale(), c)
		}
	}
}

func MinimapImage(site view.Site, scale int) *ebiten.Image {
	if site.Width == 0 {
		return ebiten.NewImage(1, 1)
	}
	w := (site.Width + site.Height) * scale
	h := (site.Width + site.Height) * scale / 2
	img := image.NewRGBA(image.Rect(0, 0, w+1, h+1))
	for y := 0; y < site.Height; y++ {
		for x := 0; x < site.Width; x++ {
			c := TileColor(site.Tile(x, y))
			px, py := MinimapPoint(site, x, y, scale)
			for dx := 0; dx < scale*2; dx++ {
				for dy := 0; dy < max(scale, 1); dy++ {
					img.SetRGBA(px+dx-scale, py+dy, c)
				}
			}
		}
	}
	return ebiten.NewImageFromImage(img)
}

func MinimapPoint(site view.Site, x, y, scale int) (int, int) {
	return (x - y + site.Height) * scale, (x + y) * scale / 2
}

const hillLift = 0.06

func HillLight(elevation uint8) render.SpriteOptions {
	if elevation == 0 {
		return render.SpriteOptions{}
	}
	b := 1 + hillLift*float32(elevation)
	return render.SpriteOptions{Tint: [3]float32{b, b, b}}
}
