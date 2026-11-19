package art

import "math"

const SealedCarry = "sealed"

func addSealedCarry(b *atlasBuilder) {
	for size := 0; size < BoxSizes; size++ {
		for f := 0; f < WalkerFrames; f++ {
			src := newCanvas(CarryWidth, CarryHeight)
			paintSealedCarry(src, size, f)
			b.add(CarryName(size, SealedCarry, "se", f), CarryWidth, CarryHeight, CarryAnchorX, CarryAnchorY, func(c *canvas) { c.paste(src, 0, 0, false) })
			b.add(CarryName(size, SealedCarry, "sw", f), CarryWidth, CarryHeight, CarryAnchorX, CarryAnchorY, func(c *canvas) { c.paste(src, 0, 0, true) })
		}
	}
}

func paintSealedCarry(c *canvas, size, frame int) {
	x, y := carryBase(size, frame)
	d := carryBoxes[clampSize(size)]
	drawBox(c, x, y, d, BoxKinds[0], 0)
	cy := y + d.hh() - d.h*0.45
	r := math.Max(1.4, d.h*0.3)
	c.disc(x, cy, r, sealR[1])
	c.disc(x-0.4, cy-0.4, math.Max(r-0.9, 0.8), sealR[2])
	c.dot(int(x-r*0.5), int(cy-r*0.5), sealR[3])
	c.outline(ink)
}
