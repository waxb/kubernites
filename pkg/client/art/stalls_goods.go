package art

import (
	"image/color"
	"math"
)

var stallEmblems = map[string]func(c *canvas, x, y int, fg, ac color.NRGBA){
	"stall-parcel": parcelGlyph, "stall-ledger": scalesGlyph, "stall-crate": crateGlyph, "stall-dispatch": letterGlyph,
	"stall-seal": sealGlyph, "stall-guildhall": hornGlyph, "stall-archive": bookGlyph,
}

var brassR = ramp{goldR[0], goldR[1], tkGold, tkGoldBright}

var stallForBox = map[string]string{"parcel": "stall-parcel", "ledger": "stall-ledger", "crate": "stall-crate", "dispatch": "stall-dispatch", "sealed": "stall-seal"}

func StallForBox(boxKind string) string {
	if k, ok := stallForBox[boxKind]; ok {
		return k
	}
	return "stall-parcel"
}

var stallGoodsKind = map[string]string{
	"stall-parcel": "parcel", "stall-ledger": "ledger", "stall-crate": "crate", "stall-dispatch": "dispatch",
	"stall-seal": "parcel", "stall-guildhall": "parcel", "stall-archive": "ledger",
}

func (t stall) emblem(c pt) {
	if glyph, ok := stallEmblems[t.kind]; ok {
		fg, ac := t.glyphTones()
		glyph(t.c, int(math.Round(c.x)), int(math.Round(c.y)), fg, ac)
	}
}

func (s iso) kindBox(gx, gy, z float64, n int, kind string, v int) {
	sz := boxCanvas(n)
	box := newCanvas(sz.w, sz.h)
	drawBox(box, float64(sz.ax), float64(sz.ay), worldBoxes[n], kind, v)
	box.outline(ink)
	p := s.p(gx, gy, z)
	s.c.paste(box, int(math.Round(p.x))-sz.ax, int(math.Round(p.y))-sz.ay, false)
}

func (t stall) goods(p stallPlan) {
	kind, ok := stallGoodsKind[t.kind]
	if !ok {
		return
	}
	z := p.counter
	spots := [][2]float64{{0.5, 0.66}, {1.02, 0.7}, {1.5, 0.66}}
	for i, sp := range spots {
		if i == 1 && t.lk.variant == 2 {
			continue
		}
		gx, gy := t.g(sp[0], sp[1])
		t.kindBox(gx, gy, z, 0, kind, (i+t.lk.variant)%BoxVariants)
	}
	if t.kind == "stall-seal" {
		t.sealDots(z)
	}
}

func (t stall) sealDots(z float64) {
	for _, u := range []float64{0.5, 1.5} {
		q := t.at(u, 0.66, z+5)
		t.c.dot(int(q.x), int(q.y), sealR[2])
		t.c.dot(int(q.x)+1, int(q.y), sealR[1])
	}
}

func parcelGlyph(c *canvas, x, y int, fg, ac color.NRGBA) {
	c.rect(x-3, y-2, 7, 5, fg)
	c.rect(x-2, y-1, 5, 3, kraftR[2])
	c.rect(x-2, y-1, 5, 1, kraftR[3])
	c.rect(x, y-2, 1, 5, ac)
	c.rect(x-3, y, 7, 1, ac)
}

func scalesGlyph(c *canvas, x, y int, fg, ac color.NRGBA) {
	c.rect(x, y-3, 1, 6, fg)
	c.rect(x-3, y-2, 7, 1, fg)
	for _, dx := range []int{-3, 3} {
		c.dot(x+dx, y-1, fg)
		c.rect(x+dx-1, y, 3, 1, ac)
		c.rect(x+dx-1, y+1, 3, 1, fg)
	}
	c.rect(x-1, y+3, 3, 1, fg)
}

func crateGlyph(c *canvas, x, y int, fg, ac color.NRGBA) {
	c.rect(x-3, y-3, 7, 6, fg)
	c.rect(x-2, y-2, 5, 4, crateR[2])
	for i := 0; i < 4; i++ {
		c.dot(x-2+i, y+1-i, fg)
	}
	c.dot(x+2, y-2, ac)
}

func letterGlyph(c *canvas, x, y int, fg, ac color.NRGBA) {
	c.rect(x-3, y-2, 7, 5, fg)
	c.rect(x-2, y-1, 5, 3, paperR[3])
	c.dot(x-1, y-1, fg)
	c.dot(x+1, y-1, fg)
	c.dot(x, y, fg)
	c.dot(x, y+1, ac)
}

func sealGlyph(c *canvas, x, y int, fg, _ color.NRGBA) {
	c.disc(float64(x)+0.5, float64(y)+0.5, 3.2, fg)
	c.disc(float64(x)+0.5, float64(y)+0.5, 2.5, sealR[1])
	c.disc(float64(x)+0.2, float64(y)+0.2, 1.4, sealR[2])
	c.dot(x, y, goldR[3])
}

func hornGlyph(c *canvas, x, y int, fg, ac color.NRGBA) {
	c.rect(x-3, y, 4, 1, fg)
	c.shadePoly([]pt{{float64(x) + 0.5, float64(y) - 0.5}, {float64(x) + 3.5, float64(y) - 3}, {float64(x) + 3.5, float64(y) + 3.5}, {float64(x) + 0.5, float64(y) + 1.5}}, flat(fg))
	c.dot(x+3, y, ac)
	c.dot(x-3, y-1, fg)
}

func bookGlyph(c *canvas, x, y int, fg, ac color.NRGBA) {
	c.rect(x-3, y-2, 7, 5, fg)
	c.rect(x-2, y-2, 2, 4, paperR[3])
	c.rect(x+1, y-2, 2, 4, paperR[2])
	c.dot(x-2, y-1, ac)
	c.dot(x+2, y, ac)
}

func (t stall) glyphTones() (color.NRGBA, color.NRGBA) {
	switch t.lk.age {
	case ageDark:
		return inkSft, t.cloth()[1]
	case ageEmpire:
		return goldR[3], goldR[1]
	}
	return t.cloth()[0], goldR[2]
}
