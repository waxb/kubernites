package art

import "image/color"

func portraitHat(c *canvas, s roleSpec) {
	switch s.hat {
	case "cap", "cap-red", "leather":
		col := map[string]color.NRGBA{"cap": woodR[1], "cap-red": redR[2], "leather": woodR[0]}[s.hat]
		c.ellipse(24, 13, 9, 4.5, col)
		c.rect(16, 13, 17, 2, darken(col, 0.2))
		c.dot(21, 11, mix(col, white, 0.3))
		if s.hat != "leather" {
			c.rect(28, 14, 8, 2, darken(col, 0.35))
		}
	case "chef":
		c.rect(16, 5, 16, 9, white)
		c.ellipse(24, 5, 9, 4.5, white)
		c.rect(16, 12, 16, 2, plasterR[2])
		c.rect(26, 5, 1, 7, plasterR[2])
	case "hood":
		c.ellipse(24, 12.5, 10, 5, s.tunic[1])
		c.ellipse(21, 11, 5, 2, s.tunic[2])
	default:
		fancyHat(c, s)
	}
}

func fancyHat(c *canvas, s roleSpec) {
	switch s.hat {
	case "straw":
		c.ellipse(24, 13, 15, 3, strawR[2])
		c.rect(17, 6, 14, 7, strawR[2])
		c.rect(17, 6, 4, 4, strawR[3])
		c.rect(17, 10, 14, 2, redR[1])
	case "tricorn":
		c.shadePoly([]pt{{9, 14}, {24, 5}, {39, 14}, {24, 11}}, flat(inkSft))
		c.line(10, 14, 24, 6, goldR[2])
		c.line(24, 6, 38, 14, goldR[1])
	case "crown":
		c.rect(16, 7, 17, 5, goldR[2])
		for _, x := range []float64{16, 21, 26, 31} {
			c.shadePoly([]pt{{x, 8}, {x + 1, 2}, {x + 2, 8}}, flat(goldR[2]))
		}
		c.rect(16, 10, 17, 1, goldR[1])
		c.dot(20, 9, redR[2])
		c.dot(24, 9, blueR[2])
		c.dot(28, 9, redR[2])
	case "feather":
		c.ellipse(22, 11, 11, 4, goldR[1])
		c.ellipse(21, 10, 8, 2.5, goldR[2])
		c.shadePoly([]pt{{28, 10}, {40, 0}, {41, 2}, {30, 11}}, flat(white))
		c.line(29, 10, 40, 1, plasterR[2])
	case "star":
		c.shadePoly([]pt{{13, 13}, {27, -4}, {35, 13}}, litByX(s.tunic, 24))
		c.ellipse(24, 13, 14, 3, s.tunic[0])
		c.dot(22, 6, goldR[3])
		c.dot(28, 9, goldR[3])
		c.dot(25, 2, goldR[3])
	}
}

func impFace(c *canvas) {
	shadedEllipse(c, 24, 50, 15, 13, furR)
	for _, x := range []float64{10, 38} {
		c.disc(x, 21, 6, furR[1])
		c.disc(x, 21, 3.5, skinR[1])
	}
	shadedEllipse(c, 24, 23, 13, 12, furR)
	c.shadePoly([]pt{{20, 12}, {24, 5}, {27, 12}}, flat(furR[2]))
	c.ellipse(24, 27, 9, 7.5, skinR[3])
	c.ellipse(24, 22, 10, 4, skinR[3])
	for _, x := range []float64{19.5, 28.5} {
		c.disc(x, 22, 3.2, white)
		c.disc(x+0.8, 22.3, 1.6, ink)
	}
	c.line(15, 17, 21, 19, furR[0])
	c.line(33, 17, 27, 19, furR[0])
	c.rect(22, 26, 1, 1, ink)
	c.rect(26, 26, 1, 1, ink)
	c.line(17, 29, 31, 29, ink)
	c.line(17, 29, 19, 32, ink)
	c.line(31, 29, 29, 32, ink)
	c.rect(19, 30, 10, 2, redR[0])
	c.rect(20, 30, 2, 1, white)
	c.rect(26, 30, 2, 1, white)
}

func sheepFace(c *canvas) {
	wool := ramp{rgb(170, 166, 156), rgb(214, 210, 200), rgb(236, 232, 222), white}
	for _, p := range [][3]float64{{12, 44, 8}, {24, 46, 10}, {36, 44, 8}, {18, 38, 6}, {30, 38, 6}} {
		shadedEllipse(c, p[0], p[1], p[2], p[2]*0.8, wool)
	}
	for _, e := range [][2]float64{{12, 20}, {36, 20}} {
		c.ellipse(e[0], e[1], 5, 2.5, inkSft)
		c.ellipse(e[0], e[1], 3, 1.2, rgb(150, 110, 110))
	}
	shadedEllipse(c, 24, 25, 8, 12, ramp{ink, inkSft, rgb(82, 70, 76), rgb(112, 100, 104)})
	for _, p := range [][3]float64{{24, 12, 5}, {19, 14, 4}, {29, 14, 4}} {
		shadedEllipse(c, p[0], p[1], p[2], p[2]*0.8, wool)
	}
	for _, x := range []int{20, 27} {
		c.rect(x, 22, 2, 2, white)
		c.dot(x+1, 23, ink)
	}
	c.rect(22, 32, 5, 1, ink)
	c.dot(24, 31, rgb(150, 110, 110))
}
