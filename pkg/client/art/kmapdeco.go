package art

import (
	"image/color"
	"math"
)

var bannerEmblems = map[string]func(c *canvas, x, y int){
	"networking": emblemAnchor, "scheduling": emblemCompass, "reliability": emblemShield, "upkeep": emblemBroomBook,
}

func emblemAnchor(c *canvas, x, y int) {
	c.disc(float64(x)+0.5, float64(y)-7.5, 2.2, goldR[2])
	c.disc(float64(x)+0.5, float64(y)-7.5, 1, bannerCloths["networking"][1])
	c.rect(x, y-5, 2, 13, goldR[2])
	c.rect(x-4, y-3, 10, 2, goldR[2])
	for i := 0; i <= 6; i++ {
		t := float64(i) / 6
		a := math.Pi * t
		c.rect(int(float64(x)+0.5-6*math.Cos(a)), int(float64(y)+3+5*math.Sin(a)), 2, 2, goldR[2])
	}
	c.dot(x-6, y+2, goldR[3])
	c.dot(x+7, y+2, goldR[3])
}

func emblemCompass(c *canvas, x, y int) {
	fx, fy := float64(x)+0.5, float64(y)
	c.disc(fx, fy, 8.5, goldR[1])
	c.disc(fx, fy, 7, bannerCloths["scheduling"][1])
	c.shadePoly([]pt{{fx, fy - 7}, {fx + 2, fy}, {fx, fy + 7}, {fx - 2, fy}}, flat(goldR[3]))
	c.shadePoly([]pt{{fx - 7, fy}, {fx, fy - 2}, {fx + 7, fy}, {fx, fy + 2}}, flat(goldR[2]))
	c.shadePoly([]pt{{fx, fy - 7}, {fx + 2, fy}, {fx - 2, fy}}, flat(redR[2]))
	c.dot(x, y, ink)
}

func emblemShield(c *canvas, x, y int) {
	fx, fy := float64(x)+0.5, float64(y)
	c.shadePoly([]pt{{fx - 7, fy - 8}, {fx + 7, fy - 8}, {fx + 7, fy}, {fx, fy + 9}, {fx - 7, fy}}, flat(goldR[2]))
	c.shadePoly([]pt{{fx - 5, fy - 6}, {fx + 5, fy - 6}, {fx + 5, fy}, {fx, fy + 6.5}, {fx - 5, fy}}, flat(redR[1]))
	c.rect(x, y-6, 1, 12, goldR[3])
	c.rect(x-5, y-2, 11, 1, goldR[3])
}

func emblemBroomBook(c *canvas, x, y int) {
	c.rect(x-7, y-3, 14, 10, cream)
	c.rect(x-7, y-3, 14, 1, white)
	c.rect(x, y-3, 1, 10, tkParchmentDark)
	c.rect(x-8, y+7, 16, 2, goldR[1])
	for _, yy := range []int{y, y + 3} {
		c.rect(x-5, yy, 4, 1, mapInk)
		c.rect(x+2, yy, 4, 1, mapInk)
	}
	c.line(x+6, y-9, x-2, y+1, woodR[2])
	c.shadePoly([]pt{{float64(x) - 4, float64(y) - 1}, {float64(x) - 1, float64(y) + 2}, {float64(x) - 5, float64(y) + 5}, {float64(x) - 7, float64(y) + 2}}, flat(strawR[2]))
}

var kmapDecorPainters = map[string]func(c *canvas){
	"compass": decoCompass, "hills": decoHills, "forest": decoForest, "waves": decoWaves, "mountains": decoMountains,
}

func decoCompass(c *canvas) {
	cx, cy := 20.0, 20.0
	ringArc(c, cx, cy, 13, 14.5, func(float64) bool { return false }, ramp{mapInk, mapInk, mapInk, mapInk})
	for i := 0; i < 8; i++ {
		a := float64(i) * math.Pi / 4
		r := 18.0
		if i%2 == 1 {
			r = 11
		}
		tip := pt{cx + r*math.Sin(a), cy - r*math.Cos(a)}
		l := pt{cx + 2.5*math.Sin(a-math.Pi/2), cy - 2.5*math.Cos(a-math.Pi/2)}
		rr := pt{cx + 2.5*math.Sin(a+math.Pi/2), cy - 2.5*math.Cos(a+math.Pi/2)}
		c.shadePoly([]pt{l, tip, {cx, cy}}, flat(tkParchmentDark))
		c.shadePoly([]pt{{cx, cy}, tip, rr}, flat(mapInk))
	}
	c.shadePoly([]pt{{cx - 2.5, cy}, {cx, cy - 18}, {cx + 2.5, cy}}, flat(tkRed))
	c.disc(cx, cy, 1.8, tkGold)
}

func decoHills(c *canvas) {
	for i, h := range [][3]float64{{8, 12, 7}, {19, 12, 9}} {
		for x := 0; x < c.w; x++ {
			dx := (float64(x) + 0.5 - h[0]) / h[2]
			if math.Abs(dx) > 1 {
				continue
			}
			top := h[1] - (h[2]-1)*math.Sqrt(1-dx*dx)
			c.dot(x, int(top), mapInk)
			if dx > 0.1 && (x+i)%2 == 0 {
				for y := int(top) + 2; y < int(h[1]); y += 2 {
					c.dot(x, y, mapInkLt)
				}
			}
		}
	}
}

func decoForest(c *canvas) {
	for i, p := range [][2]float64{{5, 13}, {11, 11}, {17, 13}, {22, 11}, {8, 8}, {15, 7}} {
		c.shadePoly([]pt{{p[0] - 3, p[1]}, {p[0], p[1] - 7}, {p[0] + 3, p[1]}}, func(x, y int) color.NRGBA {
			if float64(x) > p[0] {
				return mix(greenR[1], tkParchmentDark, 0.3)
			}
			return mix(greenR[2], tkParchment, 0.3)
		})
		c.rect(int(p[0]), int(p[1]), 1, 2, mapInk)
		c.dot(int(p[0])-1+i%2, int(p[1])-3, mapInkLt)
	}
}

func decoWaves(c *canvas) {
	for _, row := range [][2]int{{0, 1}, {6, 4}} {
		for i := 0; i < 8; i++ {
			y := row[1] + []int{0, -1, -1, 0, 1, 1, 0, 0}[i]
			c.dot(row[0]+i, y, withAlpha(blueR[1], 200))
		}
	}
}

func decoMountains(c *canvas) {
	for _, m := range [][3]float64{{9, 17, 8}, {19, 17, 12}, {26, 17, 6}} {
		apex := pt{m[0], m[1] - m[2]}
		c.shadePoly([]pt{{m[0] - m[2]*0.9, m[1]}, apex, {m[0] + m[2]*0.9, m[1]}}, func(x, y int) color.NRGBA {
			if float64(x) > m[0] && (x+y)%2 == 0 {
				return mapInkLt
			}
			return mix(tkParchment, cream, 0.3)
		})
		c.line(int(m[0]-m[2]*0.9), int(m[1]), int(apex.x), int(apex.y), mapInk)
		c.line(int(apex.x), int(apex.y), int(m[0]+m[2]*0.9), int(m[1]), mapInk)
		c.dot(int(apex.x), int(apex.y)+1, white)
	}
}
