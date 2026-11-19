package art

import (
	"image/color"
	"math"
)

var (
	oakLeaf    = ramp{rgb(34, 72, 42), rgb(54, 104, 50), rgb(84, 140, 60), rgb(140, 184, 84)}
	deepLeaf   = ramp{rgb(26, 60, 40), rgb(42, 88, 48), rgb(66, 118, 56), rgb(110, 158, 74)}
	birchLeaf  = ramp{rgb(66, 104, 40), rgb(104, 146, 52), rgb(148, 182, 70), rgb(204, 214, 116)}
	goldLeaf   = ramp{rgb(120, 76, 24), rgb(176, 122, 34), rgb(220, 170, 52), rgb(246, 214, 110)}
	emberLeaf  = ramp{rgb(112, 36, 26), rgb(168, 58, 34), rgb(214, 96, 46), rgb(244, 154, 84)}
	rustLeaf   = ramp{rgb(110, 54, 22), rgb(166, 88, 30), rgb(208, 128, 44), rgb(236, 180, 90)}
	pineLeaf   = ramp{rgb(22, 52, 42), rgb(32, 74, 54), rgb(48, 100, 64), rgb(86, 136, 80)}
	spruceLeaf = ramp{rgb(18, 42, 44), rgb(26, 62, 58), rgb(38, 84, 70), rgb(72, 118, 90)}
	willowLeaf = ramp{rgb(58, 90, 40), rgb(88, 124, 50), rgb(126, 160, 64), rgb(180, 200, 100)}
	barkR      = ramp{rgb(58, 38, 30), rgb(92, 62, 42), rgb(128, 90, 58), rgb(160, 122, 80)}
)

type clump struct{ x, y, r float64 }

func canopy(c *canvas, clumps []clump, leaf ramp, cx, cy, radius float64, seed int) {
	for i, k := range clumps {
		for y := int(k.y - k.r - 1); y <= int(k.y+k.r+1); y++ {
			for x := int(k.x - k.r - 1); x <= int(k.x+k.r+1); x++ {
				dx, dy := float64(x)+0.5-k.x, float64(y)+0.5-k.y
				d := math.Hypot(dx, dy)
				if d > k.r {
					continue
				}
				c.dot(x, y, leafTone(leaf, x, y, dx, dy, d, k.r, cx, cy, radius, seed+i, i > 0))
			}
		}
	}
}

func leafTone(leaf ramp, x, y int, dx, dy, d, r, cx, cy, radius float64, seed int, rim bool) color.NRGBA {
	local := -(dx*0.75 + dy) / (r * 1.25)
	global := -((float64(x)-cx)*0.75 + (float64(y) - cy)) / radius
	n := noise(x, y, seed)
	l := 0.68*local + 0.32*global + (n-0.5)*0.16
	var tone int
	switch {
	case rim && d > r-1.1 && dx+dy > 0.2*r:
		tone = 0
	case l > 0.5:
		tone = 3
	case l > 0.08:
		tone = 2
	case l > -0.4:
		tone = 1
	}
	if n < 0.1 && tone > 0 {
		tone--
	}
	if tone == 3 && n < 0.3 {
		tone = 2
	}
	return leaf[tone]
}

func jitter(cl []clump, v, seed int) []clump {
	out := make([]clump, len(cl))
	for i, k := range cl {
		out[i] = clump{k.x + float64(int(hash(i, v, seed)%3)-1), k.y + float64(int(hash(v, i, seed)%3)-1), k.r}
	}
	return sortClumps(out)
}

func bark(c *canvas, x, top, bottom, w int) {
	for y := top; y < bottom; y++ {
		for i := 0; i < w; i++ {
			col := barkR[1]
			switch {
			case i == 0:
				col = barkR[2]
			case i == w-1:
				col = barkR[0]
			case noise(x+i, y, 3) > 0.8:
				col = barkR[0]
			}
			c.dot(x+i, y, col)
		}
	}
}

func roots(c *canvas, x, y, w int) {
	c.dot(x-1, y-1, barkR[1])
	c.dot(x-2, y, barkR[1])
	c.dot(x+w, y-1, barkR[0])
	c.dot(x+w+1, y, barkR[0])
	c.rect(x-1, y, w+2, 1, barkR[1])
}

func sortClumps(in []clump) []clump {
	out := append([]clump(nil), in...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].y < out[j-1].y; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func paintOak(c *canvas, s size, v int) {
	leaf := []ramp{oakLeaf, deepLeaf, oakLeaf}[v]
	bark(c, 12, 28, s.ay, 4)
	roots(c, 12, s.ay-1, 4)
	c.line(13, 31, 9, 26, barkR[1])
	cl := []clump{{9, 14, 5}, {15, 11, 5.5}, {21, 14, 5}, {6, 20, 5}, {13, 18, 6}, {21, 20, 5.5}, {10, 26, 5}, {18, 26, 5}, {14, 7, 4}}
	canopy(c, jitter(cl, v, 11), leaf, 14, 17, 13, 10+v)
	if v == 1 {
		for _, p := range [][2]int{{8, 16}, {18, 13}, {21, 21}, {12, 24}} {
			c.dot(p[0], p[1], redR[3])
		}
	}
	if v == 2 {
		for _, p := range [][2]int{{7, 19}, {19, 12}, {16, 27}} {
			c.dot(p[0], p[1], goldLeaf[2])
		}
	}
}

func paintMaple(c *canvas, s size, v int) {
	leaf := []ramp{emberLeaf, goldLeaf, rustLeaf}[v]
	bark(c, 11, 26, s.ay, 3)
	roots(c, 11, s.ay-1, 3)
	cl := []clump{{7, 16, 5}, {13, 13, 5.5}, {19, 16, 5}, {9, 22, 5}, {16, 22, 5.5}, {12, 8, 4.5}}
	canopy(c, jitter(cl, v, 21), leaf, 12.5, 15, 11, 20+v)
	for i := 0; i < 3; i++ {
		c.dot(4+i*7+v, s.ay-2-(i*3+v)%5, leaf[2])
	}
}

func paintBirch(c *canvas, s size, v int) {
	c.rect(8, 16, 3, s.ay-16, white)
	c.rect(10, 16, 1, s.ay-16, greyR[2])
	for y := 18; y < s.ay; y += 3 {
		c.rect(8+(y/3)%2, y, 2, 1, ink)
	}
	c.line(10, 22, 13, 17, greyR[2])
	cl := []clump{{5, 14, 3.5}, {11, 11, 4}, {15, 16, 3.5}, {8, 19, 3.5}, {12, 5, 3}, {7, 8, 3}}
	canopy(c, jitter(cl, v, 31), birchLeaf, 9.5, 12, 9, 30+v)
	if v == 2 {
		c.mapPixels(func(col color.NRGBA) color.NRGBA {
			if col.G > col.R && col.G > 100 {
				return mix(col, goldLeaf[2], 0.55)
			}
			return col
		})
	}
}

func paintOld(c *canvas, s size, v int) {
	leaf := []ramp{deepLeaf, oakLeaf, goldLeaf}[v]
	bark(c, 18, 36, s.ay, 8)
	c.rect(21, 44, 2, 4, barkR[0])
	for _, r := range [][4]int{{18, s.ay - 1, 13, s.ay}, {25, s.ay - 1, 31, s.ay}, {19, s.ay - 3, 15, s.ay - 1}} {
		c.line(r[0], r[1], r[2], r[3], barkR[1])
	}
	c.line(19, 38, 12, 31, barkR[1])
	c.line(25, 37, 32, 30, barkR[0])
	cl := []clump{{10, 22, 8}, {22, 15, 9}, {34, 22, 8}, {15, 31, 8}, {29, 31, 8}, {22, 25, 9}, {22, 7, 6}, {13, 13, 6}, {31, 13, 6}}
	canopy(c, jitter(cl, v, 41), leaf, 22, 20, 18, 40+v)
	for _, p := range [][2]int{{19, 42}, {24, 48}} {
		c.dot(p[0], p[1], mossR[2])
	}
}

func paintWillow(c *canvas, s size, v int) {
	bark(c, 14, 20, s.ay, 4)
	roots(c, 14, s.ay-1, 4)
	cl := []clump{{10, 12, 6}, {21, 12, 6}, {16, 8, 6.5}, {15.5, 13, 6}}
	canopy(c, sortClumps(cl), willowLeaf, 16, 11, 11, 50+v)
	for i := 0; i < 11; i++ {
		x := 5 + i*2 + v%2
		top := 11 + int(noise(i, v, 51)*4)
		length := 10 + int(noise(i, v, 52)*8)
		for y := top; y < min(top+length, s.ay-3); y++ {
			col := willowLeaf[1+(i+y)%2]
			if x > 16 {
				col = willowLeaf[(i+y)%2]
			}
			c.dot(x+(y-top)/7, y, col)
		}
	}
}
