package world

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/render"
)

const (
	fogStart    = .19
	fogPeak     = .26
	fogEnd      = .37
	fogAlpha    = .2
	fogCell     = 12.0
	fogSize     = 8.0
	fogDrift    = .06
	cloudCell   = 22.0
	cloudSize   = 9.0
	cloudAlpha  = .05
	cloudOdds   = 3
	softSize    = 64
	softFall    = 1.4
	weatherPad  = 1
	cloudNightF = .35
)

var wind = Vec{.22, .09}

type cloud struct {
	center Vec
	radius float64
	shown  bool
}

func smoothstep(a, b, x float64) float64 {
	t := min(max((x-a)/(b-a), 0), 1)
	return t * t * (3 - 2*t)
}

func FogAmount(phase float64) float64 {
	p := wrap(phase, 1)
	if p <= fogStart || p >= fogEnd {
		return 0
	}
	if p < fogPeak {
		return smoothstep(fogStart, fogPeak, p)
	}
	return 1 - smoothstep(fogPeak, fogEnd, p)
}

func CloudAt(cx, cy int, t float64) cloud {
	h := mix32(uint32(cx)*0x27d4eb2d ^ uint32(cy)*0x165667b1 ^ 0xc10d)
	base := Vec{float64(cx)*cloudCell + jitterAt(h, 2, cloudCell/3), float64(cy)*cloudCell + jitterAt(h, 12, cloudCell/3)}
	return cloud{
		center: Vec{base.X + wind.X*t, base.Y + wind.Y*t},
		radius: cloudSize * (.7 + float64(h>>22%60)/100),
		shown:  h>>8%cloudOdds != 0,
	}
}

func softImage() *ebiten.Image {
	img := image.NewRGBA(image.Rect(0, 0, softSize, softSize))
	c := float64(softSize-1) / 2
	for y := 0; y < softSize; y++ {
		for x := 0; x < softSize; x++ {
			d := math.Hypot(float64(x)-c, float64(y)-c) / c
			v := uint8(255 * math.Pow(max(1-d, 0), softFall))
			img.SetRGBA(x, y, color.RGBA{v, v, v, v})
		}
	}
	return ebiten.NewImageFromImage(img)
}

func (l *life) soft() *ebiten.Image {
	if l.blob == nil {
		l.blob = softImage()
	}
	return l.blob
}

func (r *Renderer) drawWeather(dst *ebiten.Image, cam render.Camera, sc Scene) {
	if sc.Site.Width == 0 {
		return
	}
	t := sc.Time
	if sc.Reduce {
		t = 0
	}
	blob := r.lifeOf().soft()
	strength := cloudAlpha * (1 - cloudNightF*Darkness(sc.DayPhase)/.55)
	r.drawClouds(dst, cam, blob, t, strength)
	if fog := FogAmount(sc.DayPhase); fog > 0 {
		r.drawFog(dst, cam, blob, t, fog*fogAlpha)
	}
}

func gridSpan(cam render.Camera, view image.Rectangle, cell float64, shift Vec) (int, int, int, int) {
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, c := range [4][2]int{{view.Min.X, view.Min.Y}, {view.Max.X, view.Min.Y}, {view.Min.X, view.Max.Y}, {view.Max.X, view.Max.Y}} {
		gx, gy := cam.ScreenToGrid(float64(c[0]), float64(c[1]))
		x0, y0, x1, y1 = min(x0, gx-shift.X), min(y0, gy-shift.Y), max(x1, gx-shift.X), max(y1, gy-shift.Y)
	}
	return int(math.Floor(x0/cell)) - weatherPad, int(math.Floor(y0/cell)) - weatherPad, int(math.Ceil(x1/cell)) + weatherPad, int(math.Ceil(y1/cell)) + weatherPad
}

func (r *Renderer) drawClouds(dst *ebiten.Image, cam render.Camera, blob *ebiten.Image, t, strength float64) {
	x0, y0, x1, y1 := gridSpan(cam, dst.Bounds(), cloudCell, Vec{wind.X * t, wind.Y * t})
	for cy := y0; cy <= y1; cy++ {
		for cx := x0; cx <= x1; cx++ {
			c := CloudAt(cx, cy, t)
			if c.shown {
				stamp(dst, cam, blob, c.center, c.radius, [4]float32{0, 0, 0, float32(strength)})
			}
		}
	}
}

func (r *Renderer) drawFog(dst *ebiten.Image, cam render.Camera, blob *ebiten.Image, t, strength float64) {
	drift := Vec{fogDrift * t, fogDrift * t * .4}
	x0, y0, x1, y1 := gridSpan(cam, dst.Bounds(), fogCell, drift)
	for cy := y0; cy <= y1; cy++ {
		for cx := x0; cx <= x1; cx++ {
			h := mix32(uint32(cx)*0x9e3779b1 ^ uint32(cy)*0x85ebca77 ^ 0xf06)
			at := Vec{float64(cx)*fogCell + jitterAt(h, 3, fogCell/2) + drift.X, float64(cy)*fogCell + jitterAt(h, 13, fogCell/2) + drift.Y}
			a := float32(strength * (.6 + float64(h>>24%40)/100))
			stamp(dst, cam, blob, at, fogSize*(.8+float64(h>>20%40)/100), [4]float32{.92 * a, .94 * a, .97 * a, a})
		}
	}
}

func stamp(dst *ebiten.Image, cam render.Camera, blob *ebiten.Image, at Vec, radius float64, rgba [4]float32) {
	sx, sy := cam.GridToScreen(at.X, at.Y)
	z := cam.Scale()
	w := radius * render.TileWidth * z
	h := radius * render.TileHeight * z
	b := dst.Bounds()
	if sx+w < float64(b.Min.X) || sx-w > float64(b.Max.X) || sy+h < float64(b.Min.Y) || sy-h > float64(b.Max.Y) {
		return
	}
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(2*w/softSize, 2*h/softSize)
	op.GeoM.Translate(sx-w, sy-h)
	op.ColorScale.Scale(rgba[0], rgba[1], rgba[2], rgba[3])
	dst.DrawImage(blob, op)
}
