package art

import (
	"fmt"
	"math"
)

const (
	BoxSizes      = 4
	BoxVariants   = 4
	FreshLevels   = 4
	BoxPileLevels = 4
	SpoilFrames   = EffectFrames
	CarryWidth    = 20
	CarryHeight   = 34
	CarryAnchorX  = 10
	CarryAnchorY  = 32
	boxPileW      = 44
	boxPileH      = 58
	boxPileAX     = 22
	boxPileAY     = 44
)

var BoxKinds = []string{"parcel", "ledger", "crate", "dispatch"}

var BoxSizeNames = []string{"small", "medium", "large", "huge"}

type boxDims struct {
	s, h float64
}

func (d boxDims) hw() float64 { return d.s * 16 }
func (d boxDims) hh() float64 { return d.s * 8 }

var worldBoxes = []boxDims{{0.25, 5}, {0.375, 7}, {0.5, 10}, {0.75, 14}}

var carryBoxes = []boxDims{{0.1875, 4}, {0.25, 5}, {0.3125, 6}, {0.375, 8}}

func clampSize(size int) int { return min(max(size, 0), BoxSizes-1) }

func BoxLift(size int) int { return int(worldBoxes[clampSize(size)].h) }

func boxCanvas(n int) size {
	d := worldBoxes[clampSize(n)]
	w := int(d.s*32) + 12
	ay := 3 + int(math.Ceil(d.hh()+d.h))
	return size{w: w, h: ay + int(math.Ceil(d.hh())) + 3, ax: w / 2, ay: ay}
}

func BoxName(size int, kind string, variant int) string {
	return fmt.Sprintf("box/%d/%s/%d", size, kind, variant)
}

func BoxSpoiledName(size int, kind string, variant int) string {
	return BoxName(size, kind, variant) + "/spoiled"
}

func BoxSealName(size int) string  { return fmt.Sprintf("box/seal/%d", size) }
func BoxCrownName(size int) string { return fmt.Sprintf("box/crown/%d", size) }

func BoxFreshName(size, level int) string { return fmt.Sprintf("box/fresh/%d/%d", size, level) }

func BoxPileName(kind string, level int) string { return fmt.Sprintf("boxpile/%s/%d", kind, level) }

func CarryName(size int, kind, dir string, frame int) string {
	return fmt.Sprintf("carry/%d/%s/%s/%d", size, kind, dir, frame)
}

func addFreight(b *atlasBuilder) {
	for size := 0; size < BoxSizes; size++ {
		addBoxSize(b, size)
		addPaperBoxes(b, size)
	}
	for _, k := range BoxKinds {
		for level := 0; level < BoxPileLevels; level++ {
			kind, lv := k, level
			b.add(BoxPileName(k, level), boxPileW, boxPileH, boxPileAX, boxPileAY, func(c *canvas) { paintBoxPile(c, kind, lv) })
		}
	}
}

func addBoxSize(b *atlasBuilder, size int) {
	sz, d := boxCanvas(size), worldBoxes[size]
	for _, k := range BoxKinds {
		for v := 0; v < BoxVariants; v++ {
			kind, vv := k, v
			b.add(BoxName(size, k, v), sz.w, sz.h, sz.ax, sz.ay, func(c *canvas) { paintBox(c, sz, d, kind, vv, false) })
			b.add(BoxSpoiledName(size, k, v), sz.w, sz.h, sz.ax, sz.ay, func(c *canvas) { paintBox(c, sz, d, kind, vv, true) })
		}
	}
	b.add(BoxSealName(size), sz.w, sz.h, sz.ax, sz.ay, func(c *canvas) { paintSealMark(c, sz, d, size) })
	b.add(BoxCrownName(size), sz.w, sz.h, sz.ax, sz.ay, func(c *canvas) { paintCrownMark(c, sz, d, size) })
	for level := 0; level < FreshLevels; level++ {
		lv := level
		b.add(BoxFreshName(size, level), sz.w, sz.h, sz.ax, sz.ay, func(c *canvas) { paintFreshMark(c, sz, d, size, lv) })
	}
}

func paintBox(c *canvas, sz size, d boxDims, kind string, v int, spoiled bool) {
	body := newCanvas(sz.w, sz.h)
	drawBox(body, float64(sz.ax), float64(sz.ay), d, kind, v)
	if spoiled {
		rot(body, kind, v)
	}
	body.outline(ink)
	propShadow(c, float64(sz.ax), float64(sz.ay)+d.hh()-0.5, d.hw()*0.9)
	c.paste(body, 0, 0, false)
}

func drawBox(c *canvas, ox, oy float64, d boxDims, kind string, v int) {
	if kind == "dispatch" {
		drawCask(c, ox, oy, d, v)
		return
	}
	bx := boxAt(c, ox, oy, d)
	bx.fill(boxMaterials[kind](v))
	boxDecals[kind](bx, v)
}
