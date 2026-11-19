package art

import "image/color"

var squidR = ramp{rgb(120, 50, 110), rgb(170, 80, 150), rgb(214, 124, 186), rgb(244, 186, 220)}

func iconSealSmithy(c *canvas) {
	c.rect(2, 10, 12, 2, greyR[2])
	c.rect(4, 12, 8, 2, greyR[1])
	c.rect(6, 14, 4, 1, greyR[0])
	c.rect(1, 10, 3, 1, greyR[3])
	c.disc(8.5, 6, 3.6, sealR[1])
	c.disc(8, 5.5, 2.6, sealR[2])
	c.disc(8.5, 6, 1.1, sealR[0])
	c.dot(6, 4, sealR[3])
	c.dot(13, 3, fireR[3])
	c.dot(14, 2, withAlpha(fireR[2], 170))
	c.line(12, 8, 14, 4, sealR[2])
}

func iconKeyGate(c *canvas) {
	c.disc(4.5, 5.5, 3.6, goldR[1])
	c.disc(4, 5, 2.5, goldR[2])
	c.disc(4.5, 5.5, 1.2, inkSft)
	c.line(7, 7, 13, 13, goldR[2])
	c.line(8, 7, 14, 13, goldR[1])
	c.rect(10, 12, 2, 2, goldR[1])
	c.rect(12, 10, 2, 2, goldR[1])
	c.dot(3, 3, goldR[3])
}

func iconBellWatch(c *canvas) {
	c.line(1, 14, 6, 6, goldR[1])
	c.line(15, 14, 10, 6, goldR[1])
	c.ellipse(8, 7, 3.6, 4.2, goldR[1])
	c.ellipse(7.2, 6, 1.8, 2.6, goldR[3])
	c.rect(3, 10, 10, 1, goldR[0])
	c.rect(7, 1, 2, 2, woodR[1])
	c.dot(8, 12, goldR[2])
	c.dot(1, 14, goldR[3])
	c.dot(14, 14, goldR[3])
}

func iconCacheStore(c *canvas) {
	c.ellipse(8, 5.5, 4.5, 4.5, squidR[2])
	c.ellipse(7, 4.5, 2.2, 2.2, squidR[3])
	for i := 0; i < 5; i++ {
		x := 4 + i*2
		c.line(x, 9, x+[]int{-1, -1, 0, 1, 1}[i], 14, squidR[1])
	}
	c.dot(6, 6, ink)
	c.dot(10, 6, ink)
}

func iconCaravan(c *canvas) {
	c.rect(2, 8, 12, 3, woodR[2])
	c.rect(2, 8, 12, 1, woodR[3])
	c.shadePoly([]pt{{2.5, 8}, {3, 4}, {8, 1.5}, {13, 4}, {13.5, 8}}, func(x, y int) color.NRGBA {
		if x < 7 {
			return cream
		}
		return plasterR[2]
	})
	c.line(5, 7, 5, 3, woodR[1])
	c.line(10, 7, 10, 3, woodR[1])
	for _, x := range []float64{4.5, 11.5} {
		c.disc(x, 12.5, 2.3, woodR[0])
		c.dot(int(x), 12, woodR[3])
	}
	c.line(14, 10, 15, 13, woodR[1])
}

func iconSealPress(c *canvas) {
	c.rect(3, 1, 10, 2, woodR[2])
	c.rect(3, 1, 10, 1, woodR[3])
	c.rect(7, 3, 2, 6, woodR[1])
	c.rect(4, 9, 8, 3, goldR[1])
	c.rect(4, 9, 8, 1, goldR[3])
	c.disc(8, 13.5, 2.2, sealR[2])
	c.dot(7, 13, sealR[3])
}
