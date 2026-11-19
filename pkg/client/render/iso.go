package render

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	TileWidth  = 32
	TileHeight = 16
	StepHeight = 5
)

var ZoomLevels = []float64{0.5, 1, 2}

type Camera struct {
	OriginX int
	OriginY int
	X       float64
	Y       float64
	Zoom    float64
}

func NewCamera(originX, originY int) Camera {
	return Camera{OriginX: originX, OriginY: originY, Zoom: 1}
}

func (c Camera) Scale() float64 {
	return min(max(c.Zoom, ZoomLevels[0]), ZoomLevels[len(ZoomLevels)-1])
}

func (c Camera) GridToScreen(gx, gy float64) (float64, float64) {
	z := c.Scale()
	isoX := (gx-gy)*TileWidth/2 - c.X
	isoY := (gx+gy)*TileHeight/2 - c.Y
	return float64(c.OriginX) + isoX*z, float64(c.OriginY) + isoY*z
}

func (c Camera) GridToScreenElevated(gx, gy, elevation float64) (float64, float64) {
	x, y := c.GridToScreen(gx, gy)
	return x, y - elevation*StepHeight*c.Scale()
}

func (c Camera) ScreenToGrid(sx, sy float64) (float64, float64) {
	z := c.Scale()
	isoX := (sx-float64(c.OriginX))/z + c.X
	isoY := (sy-float64(c.OriginY))/z + c.Y
	return isoX/TileWidth + isoY/TileHeight, isoY/TileHeight - isoX/TileWidth
}

func (c Camera) TileAt(sx, sy int) (int, int) {
	gx, gy := c.ScreenToGrid(float64(sx), float64(sy))
	return int(math.Floor(gx + .5)), int(math.Floor(gy + .5))
}

func (c Camera) Pan(dx, dy float64) Camera {
	next := c
	next.X += dx / c.Scale()
	next.Y += dy / c.Scale()
	return next
}

func (c Camera) ZoomAt(zoom, sx, sy float64) Camera {
	gx, gy := c.ScreenToGrid(sx, sy)
	next := c
	next.Zoom = zoom
	ax, ay := next.GridToScreen(gx, gy)
	return next.Pan(ax-sx, ay-sy)
}

func (c Camera) ZoomStep(dir int, sx, sy float64) Camera {
	idx := 0
	for i, z := range ZoomLevels {
		if math.Abs(z-c.Scale()) < 1e-9 {
			idx = i
		}
	}
	idx = min(max(idx+dir, 0), len(ZoomLevels)-1)
	return c.ZoomAt(ZoomLevels[idx], sx, sy)
}

func (c Camera) CenterOn(gx, gy float64, viewW, viewH int) Camera {
	sx, sy := c.GridToScreen(gx, gy)
	return c.Pan(sx-float64(viewW)/2, sy-float64(viewH)/2)
}

func (c Camera) Clamp(tiles, viewW, viewH int) Camera {
	z := c.Scale()
	halfW := float64(tiles * TileWidth / 2)
	bottom := float64(tiles * TileHeight)
	offX := (float64(viewW)/2 - float64(c.OriginX)) / z
	offY := (float64(viewH)/2 - float64(c.OriginY)) / z
	cy := min(max(c.Y+offY, 0), bottom)
	mid := bottom / 2
	span := halfW
	if mid > 0 {
		span *= 1 - math.Abs(cy-mid)/mid
	}
	cx := min(max(c.X+offX, -span), span)
	next := c
	next.X = cx - offX
	next.Y = cy - offY
	return next
}

type Region struct{ X, Y, W, H, AnchorX, AnchorY int }

func (r Region) Rect() image.Rectangle {
	return image.Rect(r.X, r.Y, r.X+r.W, r.Y+r.H)
}

type SpriteOptions struct {
	Flip  bool
	Alpha float32
	Tint  [3]float32
	Scale float64
}

func DrawAt(dst, sheet *ebiten.Image, r Region, sx, sy float64, camZoom float64, o SpriteOptions) {
	if sheet == nil || r.W <= 0 || r.H <= 0 {
		return
	}
	sub := sheet.SubImage(r.Rect()).(*ebiten.Image)
	z := camZoom
	if o.Scale > 0 {
		z *= o.Scale
	}
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	if o.Flip {
		op.GeoM.Scale(-z, z)
		op.GeoM.Translate(float64(r.W)*z, 0)
		op.GeoM.Translate(math.Round(sx-float64(r.W-r.AnchorX)*z), math.Round(sy-float64(r.AnchorY)*z))
	} else {
		op.GeoM.Scale(z, z)
		op.GeoM.Translate(math.Round(sx-float64(r.AnchorX)*z), math.Round(sy-float64(r.AnchorY)*z))
	}
	applyColor(op, o)
	dst.DrawImage(sub, op)
}

func applyColor(op *ebiten.DrawImageOptions, o SpriteOptions) {
	if o.Tint != [3]float32{} {
		op.ColorScale.Scale(o.Tint[0], o.Tint[1], o.Tint[2], 1)
	}
	if o.Alpha > 0 && o.Alpha < 1 {
		op.ColorScale.ScaleAlpha(o.Alpha)
	}
}

func DepthKey(gx, gy float64) float64 {
	return (gx+gy)*1000 + gy
}
