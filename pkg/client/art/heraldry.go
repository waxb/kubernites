package art

import "image/color"

const (
	PennantW  = 14
	PennantH  = 21
	PennantAX = 2
	PennantAY = 19
	Crests    = 2
)

var fiefColours = [12]color.NRGBA{
	rgb(150, 146, 140), rgb(196, 58, 52), rgb(62, 112, 196), rgb(64, 150, 74), rgb(236, 190, 52), rgb(140, 72, 170),
	rgb(232, 120, 40), rgb(70, 178, 186), rgb(232, 232, 220), rgb(120, 48, 64), rgb(150, 190, 60), rgb(196, 96, 160),
}

func FiefColour(colour int) color.NRGBA {
	return fiefColours[((colour%len(fiefColours))+len(fiefColours))%len(fiefColours)]
}

func PennantName(crest int) string {
	return "heraldry/pennant/" + string(rune('0'+crest%Crests))
}

func addHeraldry(b *atlasBuilder) {
	for crest := 0; crest < Crests; crest++ {
		c := crest
		b.add(PennantName(c), PennantW, PennantH, PennantAX, PennantAY, func(cv *canvas) { paintPennant(cv, c) })
	}
}

var pennantCloth = ramp{rgb(170, 170, 170), rgb(206, 206, 206), rgb(236, 236, 236), rgb(255, 255, 255)}

func paintPennant(c *canvas, crest int) {
	c.rect(PennantAX, 3, 1, PennantAY-3, woodR[0])
	c.dot(PennantAX, 2, goldR[2])
	top, h, w := 4, 7, PennantW-PennantAX-3
	for y := 0; y < h; y++ {
		run := w
		if crest == 0 && y >= 2 && y <= h-3 {
			run = w - 3 + min(abs(y-h/2), 2)
		}
		if crest == 1 {
			run = w - y/2
		}
		for x := 0; x < run; x++ {
			col := pennantCloth[2]
			switch {
			case y == 0:
				col = pennantCloth[3]
			case y == h-1 || x == run-1:
				col = pennantCloth[0]
			case (x+y)%4 == 0:
				col = pennantCloth[1]
			}
			c.dot(PennantAX+1+x, top+y, col)
		}
	}
	c.outline(ink)
}
