package world

import (
	"fmt"
	"image"
	"image/color"
	"maps"
	"math"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	FadeSeconds   = 1.4
	RiseSeconds   = .9
	sinkPixels    = 6.0
	gardenBelow   = .35
	gardenW       = 24
	gardenH       = 12
	gardenSprout  = 3
	gardenPlot    = .66
	gardenKinds   = 3
	riseDustShare = .5
)

var (
	gardenInk   = color.NRGBA{44, 34, 24, 255}
	gardenSoil  = color.NRGBA{132, 92, 54, 255}
	gardenTill  = color.NRGBA{98, 66, 38, 255}
	gardenRim   = color.NRGBA{176, 132, 84, 255}
	gardenBack  = color.NRGBA{120, 84, 50, 255}
	gardenLeafs = [gardenKinds][3]color.NRGBA{
		{{58, 112, 46, 255}, {112, 178, 72, 255}, {168, 214, 104, 255}},
		{{58, 112, 46, 255}, {104, 164, 70, 255}, {214, 86, 74, 255}},
		{{58, 112, 46, 255}, {112, 170, 64, 255}, {244, 212, 96, 255}},
	}
	flatScenery = []string{"decor/flowers", "decor/tallgrass", "decor/fern", "decor/lily"}
)

func RiseOf(born, now float64) float64 {
	return min(max((now-born)/RiseSeconds, 0), 1)
}

func FadeOf(since, now float64) float64 {
	return min(max(1-(now-since)/FadeSeconds, 0), 1)
}

func (r *Renderer) hamletOf() *Hamlet {
	if r.hamlet == nil {
		r.hamlet = NewHamlet()
	}
	return r.hamlet
}

func (r *Renderer) coverHamlet(h *Hamlet, site view.Site) {
	if r.Land == nil || r.Land.land == nil {
		r.livingOf().base, r.livingOf().baseSig = nil, 0
		h.Covered(0, nil)
		return
	}
	lc, l := r.Land, r.livingOf()
	hb := lc.habitat(site)
	l.base, l.baseSig = func(x, y int) bool { return tallScenery(sceneryFor(site, hb, x, y)) }, lc.version^lc.hbSig*sigPrime|1
	h.Covered(l.baseSig^l.plans.sig*sigPrime|1, func(x, y int) bool { return l.base(x, y) || l.plans.farm[Point{x, y}] })
}

func tallScenery(items []scenery) bool {
	for _, s := range items {
		if !s.tiny && !flatName(s.name) {
			return true
		}
	}
	return false
}

func flatName(name string) bool {
	for _, prefix := range flatScenery {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func (r *Renderer) drawRising(dst *ebiten.Image, name string, sx, sy, zoom float64, o render.SpriteOptions, rise float64) bool {
	sheet, reg, ok := r.Lib.Lookup(name)
	if !ok {
		return false
	}
	if rise < 1 {
		shown := max(int(math.Ceil(float64(reg.H)*rise)), 1)
		cut := reg.H - shown
		reg = render.Region{X: reg.X, Y: reg.Y + cut, W: reg.W, H: shown, AnchorX: reg.AnchorX, AnchorY: reg.AnchorY - cut}
	}
	render.DrawAt(dst, sheet, reg, sx, sy, zoom, o)
	return true
}

func (r *Renderer) riseDust(cam render.Camera, sc Scene, anchor Vec, rise float64) (Drawable, bool) {
	if rise >= riseDustShare || sc.Reduce {
		return Drawable{}, false
	}
	sx, sy := ScreenPointOf(cam, sc.Site, anchor)
	name := gfx.EffectName("dust", int(rise/riseDustShare*6))
	return Drawable{Depth: render.DepthKey(anchor.X, anchor.Y) + .09, Draw: func(dst *ebiten.Image) {
		r.Lib.Draw(dst, name, sx, sy, cam.Scale(), render.SpriteOptions{Alpha: .8})
	}}, true
}

func (r *Renderer) hamletItems(out []Drawable, cam render.Camera, sc Scene, h *Hamlet) []Drawable {
	zoom := cam.Scale()
	for _, g := range h.gone {
		a := Vec{float64(g.spot.At.X) + .5, float64(g.spot.At.Y) + .5}
		sx, sy := ScreenPointOf(cam, sc.Site, Vec{a.X - .01, a.Y - .01})
		fade := FadeOf(g.since, sc.Time)
		if !onScreen(sx, sy, zoom) || fade <= 0 {
			continue
		}
		names := HutNames(max(g.spot.Tier, 1), model.HearthBanished, g.spot.QoS)
		sink := (1 - fade) * sinkPixels * zoom
		out = append(out, Drawable{Depth: render.DepthKey(a.X, a.Y), Draw: func(dst *ebiten.Image) {
			r.Lib.DrawFirst(dst, names, sx, sy+sink, zoom, render.SpriteOptions{Alpha: float32(fade)})
		}})
	}
	if zoom < familiarZoom {
		return out
	}
	return r.gardenItems(out, cam, sc, h)
}

func (r *Renderer) gardenItems(out []Drawable, cam render.Camera, sc Scene, h *Hamlet) []Drawable {
	zoom := cam.Scale()
	for _, name := range slices.Sorted(maps.Keys(h.SpotList)) {
		s := h.SpotList[name]
		if !s.Sown || h.occAt(s.Garden) != occGarden || !r.livingOf().bound[name] {
			continue
		}
		at := Vec{float64(s.Garden.X) + .5, float64(s.Garden.Y) + .5}
		sx, sy := ScreenPointOf(cam, sc.Site, at)
		if !onScreen(sx, sy, zoom) {
			continue
		}
		kind := int(model.Hash(name) % art.GardenKinds)
		bed := r.livingOf().piece(fmt.Sprintf("garden%d", kind), func() art.Piece { return art.PaintTidyGarden(kind) })
		out = append(out, Drawable{Depth: render.DepthKey(at.X, at.Y) - gardenBelow, Draw: func(dst *ebiten.Image) {
			bed.draw(dst, sx, sy, zoom, render.SpriteOptions{})
		}})
	}
	return out
}

func (r *Renderer) gardenBeds() [gardenKinds]*ebiten.Image {
	if r.beds[0] == nil {
		for k := range r.beds {
			r.beds[k] = ebiten.NewImageFromImage(PaintGarden(k))
		}
	}
	return r.beds
}

func PaintGarden(kind int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, gardenW, gardenH+gardenSprout))
	leaf := gardenLeafs[((kind%gardenKinds)+gardenKinds)%gardenKinds]
	cx, cy := float64(gardenW-1)/2, float64(gardenH-1)/2
	for y := 0; y < gardenH; y++ {
		for x := 0; x < gardenW; x++ {
			d := math.Abs(float64(x)-cx)/(cx+.5) + math.Abs(float64(y)-cy)/(cy+.5)
			if d <= 1 {
				img.SetNRGBA(x, y+gardenSprout, bedPixel(x, y, d, float64(y) > cy))
			}
		}
	}
	for y := 0; y < gardenH; y++ {
		for x := 0; x < gardenW; x++ {
			d := math.Abs(float64(x)-cx)/(cx+.5) + math.Abs(float64(y)-cy)/(cy+.5)
			if d < gardenPlot && (x+2*y)%4 == 2 && x%4 == kind%2*2 {
				sprout(img, x, y+gardenSprout, leaf)
			}
		}
	}
	return img
}

func bedPixel(x, y int, d float64, front bool) color.NRGBA {
	switch {
	case d > .9:
		return gardenInk
	case d > .76 && front:
		return gardenRim
	case d > .76:
		return gardenBack
	case (x+2*y)%4 == 0:
		return gardenTill
	}
	return gardenSoil
}

func sprout(img *image.NRGBA, x, y int, leaf [3]color.NRGBA) {
	img.SetNRGBA(x, y, leaf[0])
	img.SetNRGBA(x-1, y-1, leaf[1])
	img.SetNRGBA(x+1, y-1, leaf[1])
	img.SetNRGBA(x, y-1, leaf[0])
	img.SetNRGBA(x, y-2, leaf[2])
}
