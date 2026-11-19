package art

import "image/color"

var berryBloom = [][]color.NRGBA{{}, {redR[3], redR[2]}, {white, rgb(236, 170, 200)}}

func paintBush(c *canvas, s size, v int) {
	leaf := []ramp{oakLeaf, deepLeaf, birchLeaf}[v]
	cl := []clump{{5, 8, 3.5}, {11, 8, 3.5}, {8, 6, 4}, {8, 9, 3.5}}
	canopy(c, sortClumps(cl), leaf, 8, 7, 6, 80+v)
	for i, col := range berryBloom[v] {
		for k := 0; k < 3; k++ {
			x, y := 4+int(noise(k, i, 81+v)*9), 4+int(noise(i, k, 82+v)*6)
			c.dot(x, y, col)
		}
	}
}

func paintFern(c *canvas, s size, v int) {
	leaf := []ramp{oakLeaf, deepLeaf, rustLeaf}[v]
	base := pt{7, float64(s.ay)}
	for i, tip := range []pt{{1, 3}, {4, 1}, {8, 0}, {11, 2}, {13, 5}} {
		col := leaf[1+i%2]
		c.line(int(base.x), int(base.y), int(tip.x), int(tip.y)+2, col)
		c.dot(int(tip.x), int(tip.y)+1, leaf[3])
		mid := pt{(base.x + tip.x) / 2, (base.y + tip.y) / 2}
		c.dot(int(mid.x)-1, int(mid.y), leaf[2])
		c.dot(int(mid.x)+1, int(mid.y)+1, leaf[0])
	}
}

func paintReeds(c *canvas, s size, v int) {
	for i, x := range []int{2, 4, 6, 8, 10, 12} {
		top := 2 + (i*3+v*2)%6
		lean := (i%3 - 1)
		c.line(x, s.ay, x+lean, top, greenR[1+i%3])
		if (i+v)%3 != 1 {
			c.rect(x+lean, top-2, 1, 3, woodR[0])
			c.dot(x+lean, top-2, woodR[1])
		}
	}
	c.line(3, s.ay, 0, s.ay-5, greenR[2])
	c.line(11, s.ay, 13, s.ay-6, greenR[1])
}

func paintFlowers(c *canvas, s size, v int) {
	colors := [][]color.NRGBA{{goldR[3], white}, {redR[3], goldR[3]}, {rgb(176, 156, 236), white}}[v]
	for i := 0; i < 8; i++ {
		x, y := 1+(i*5+v*3)%11, 1+(i*3+v)%4
		c.dot(x, y+1, greenR[1])
		c.dot(x, y+2, greenR[0])
		c.dot(x, y, colors[i%len(colors)])
	}
}

func paintMushrooms(c *canvas, s size, v int) {
	cap := []ramp{redR, woodR, goldR}[v]
	for i, p := range [][3]int{{3, 5, 2}, {7, 4, 3}, {10, 6, 2}} {
		x, y, r := p[0], p[1], p[2]
		c.rect(x, y, 1, s.ay-y, cream)
		c.ellipse(float64(x)+0.5, float64(y), float64(r), float64(r)*0.6, cap[1+i%2])
		c.dot(x-1, y-1, cap[3])
		if v == 0 {
			c.dot(x+1, y, white)
		}
	}
}

func paintTallgrass(c *canvas, s size, v int) {
	leaf := []ramp{grassR, meadowR, dryR}[v]
	for i := 0; i < 7; i++ {
		x := 1 + i*2 - (i % 2)
		top := 1 + (i*5+v)%4
		c.line(x, s.ay, x+(i%3-1), top, leaf[1+i%2])
		c.dot(x+(i%3-1), top, leaf[3])
	}
}

func paintLily(c *canvas, s size, v int) {
	pad := ramp{rgb(36, 90, 54), rgb(56, 120, 64), rgb(88, 150, 76), rgb(140, 190, 100)}
	for i, p := range [][3]float64{{4, 3, 2.5}, {9, 4, 2}, {7, 1.5, 1.5}} {
		if i == 2 && v == 1 {
			continue
		}
		c.ellipse(p[0], p[1], p[2]*1.4, p[2]*0.7, pad[1])
		c.dot(int(p[0])-1, int(p[1])-1, pad[3])
		c.dot(int(p[0])+1, int(p[1]), pad[0])
	}
	if v != 1 {
		c.dot(9, 3, white)
		c.dot(9, 2, rgb(240, 180, 210))
	}
}
