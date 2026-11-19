package art

import "image/color"

var cutR = ramp{rgb(120, 114, 108), rgb(170, 164, 152), rgb(214, 206, 188), rgb(240, 234, 216)}

func paintOutcrop(c *canvas, s size, v int) {
	shapes := [][][]pt{
		{{{2, 19}, {4, 10}, {10, 6}, {15, 9}, {16, 19}}, {{12, 19}, {15, 8}, {21, 4}, {26, 9}, {27, 19}}, {{8, 20}, {10, 15}, {15, 13}, {19, 16}, {19, 20}}},
		{{{3, 19}, {6, 8}, {12, 3}, {18, 7}, {19, 19}}, {{15, 20}, {18, 12}, {23, 10}, {27, 14}, {27, 20}}, {{1, 20}, {2, 16}, {6, 15}, {8, 20}}},
		{{{4, 19}, {5, 12}, {11, 9}, {14, 12}, {14, 19}}, {{10, 19}, {12, 6}, {17, 2}, {22, 6}, {23, 19}}, {{20, 20}, {22, 15}, {26, 15}, {28, 20}}},
	}
	for i, sh := range shapes[v%len(shapes)] {
		facet(c, sh, rockR, 300+v*7+i)
	}
	for _, p := range [][2]int{{6, 11}, {18, 7}, {11, 16}} {
		c.dot(p[0]+v, p[1], greenR[2])
		c.dot(p[0]+v+1, p[1], greenR[3])
	}
}

func paintQuarry(c *canvas, s size, v int) {
	facet(c, []pt{{3, 20}, {5, 10}, {11, 4}, {20, 3}, {27, 8}, {29, 20}}, rockR, 320+v)
	cut := func(sh shader, pts ...pt) { c.shadePoly(pts, sh) }
	rows := newCourses(330+v, 2, 3)
	chisel := func(r ramp) shader {
		return func(x, y int) color.NRGBA {
			row, fv := rows.at(float64(y))
			switch n := pick(x, row, 331+v); {
			case fv < 1 && n > 0.2:
				return r[1]
			case n > 0.93:
				return r[3]
			case n < 0.06:
				return r[1]
			}
			return r[2]
		}
	}
	cut(flat(cutR[3]), pt{6, 12}, pt{13, 9}, pt{21, 12}, pt{14, 15})
	cut(chisel(cutR), pt{6, 12}, pt{14, 15}, pt{14, 21}, pt{6, 20})
	cut(chisel(ramp{cutR[0], cutR[0], cutR[1], cutR[2]}), pt{14, 15}, pt{21, 12}, pt{21, 19}, pt{14, 21})
	blocks := iso{c: c, ox: 25, oy: 21}
	for i, b := range [][3]float64{{-0.1, 0.05, 0}, {0.25, -0.2, 0}, {0.05, -0.1, 6}} {
		if i == 2 && v == 0 {
			continue
		}
		blocks.box(b[0]-0.13, b[1]-0.13, b[0]+0.13, b[1]+0.13, b[2], b[2]+6, faces{top: flat(cutR[3]), left: speckled(cutR, 2, v), right: speckled(cutR, 1, v)})
	}
	if v != 1 {
		c.line(4, 22, 9, 15, woodR[2])
		c.rect(7, 14, 5, 1, greyR[1])
		c.dot(11, 15, greyR[0])
	}
	if v == 2 {
		c.dot(17, 6, greenR[2])
		c.dot(18, 6, greenR[3])
	}
}

func speckled(r ramp, base, v int) shader {
	return func(x, y int) color.NRGBA {
		switch n := pick(x, y, 333+v); {
		case n > 0.9:
			return r[min(base+1, 3)]
		case n < 0.08:
			return r[max(base-1, 0)]
		}
		return r[base]
	}
}

var freightIconPainters = map[string]func(c *canvas){
	"wood": iconWood, "stone": iconStone,
	"kind-parcel": func(c *canvas) { kindIcon(c, "parcel", 0) }, "kind-ledger": func(c *canvas) { kindIcon(c, "ledger", 0) },
	"kind-crate": func(c *canvas) { kindIcon(c, "crate", 2) }, "kind-dispatch": func(c *canvas) { kindIcon(c, "dispatch", 0) },
	"sealed": iconSealed, "priority": iconPriority, "fresh": iconFresh, "spoiled": iconSpoiled,
	"seal-smithy": iconSealSmithy, "key-gate": iconKeyGate, "bell-watch": iconBellWatch,
	"cache-store": iconCacheStore, "caravan-yard": iconCaravan, "herald": iconHorn,
}

var FreightIcons = []string{"wood", "stone", "kind-parcel", "kind-ledger", "kind-crate", "kind-dispatch", "sealed", "priority", "fresh", "spoiled",
	"seal-smithy", "key-gate", "bell-watch", "cache-store", "caravan-yard", "herald"}

func iconWood(c *canvas) {
	for i, p := range [][2]float64{{5, 12}, {11, 12}, {8, 7.5}} {
		c.rect(int(p[0])-4, int(p[1])-2, 8, 4, woodR[1+i%2])
		c.rect(int(p[0])-4, int(p[1])-2, 8, 1, woodR[3])
		c.disc(p[0]+3.5, p[1], 2.1, strawR[2])
		c.dot(int(p[0])+3, int(p[1])-1, woodR[1])
	}
	c.dot(13, 3, greenR[3])
	c.dot(12, 4, greenR[2])
}

func iconStone(c *canvas) {
	for _, b := range [][3]int{{0, 9, 8}, {8, 9, 8}, {4, 3, 8}} {
		c.rect(b[0], b[1], b[2], 6, inkSft)
		c.rect(b[0]+1, b[1]+1, b[2]-2, 4, stoneR[2])
		c.rect(b[0]+1, b[1]+1, b[2]-2, 1, stoneR[3])
		c.rect(b[0]+b[2]-2, b[1]+2, 1, 3, stoneR[1])
		c.dot(b[0]+2, b[1]+3, stoneR[1])
	}
}

func kindIcon(c *canvas, kind string, v int) {
	d := boxDims{0.4375, 6}
	drawBox(c, 8, 10.5, d, kind, v)
}

func iconSealed(c *canvas) {
	c.line(6, 11, 4, 15, twineR[2])
	c.line(10, 11, 12, 15, twineR[3])
	c.disc(8, 8, 5.5, sealR[1])
	c.disc(7.5, 7.5, 4.2, sealR[2])
	c.disc(8, 8, 2, sealR[0])
	c.dot(5, 5, sealR[3])
}

func iconPriority(c *canvas) {
	c.shadePoly([]pt{{1, 5}, {5, 9}, {8, 2}, {11, 9}, {15, 5}, {14, 13}, {2, 13}}, litByX(goldR, 8))
	c.rect(2, 11, 12, 3, goldR[1])
	c.rect(2, 11, 12, 1, goldR[3])
	for _, p := range [][2]int{{1, 4}, {8, 1}, {15, 4}} {
		c.disc(float64(p[0])+0.5, float64(p[1])+0.5, 1.2, goldR[3])
	}
	c.dot(5, 12, redR[2])
	c.dot(8, 12, blueR[2])
	c.dot(11, 12, redR[2])
}

func iconFresh(c *canvas) {
	c.rect(3, 1, 10, 2, woodR[1])
	c.rect(3, 13, 10, 2, woodR[1])
	c.shadePoly([]pt{{4, 3}, {12, 3}, {8.5, 8}, {7.5, 8}}, flat(glassR[3]))
	c.shadePoly([]pt{{7.5, 8}, {8.5, 8}, {12, 13}, {4, 13}}, flat(glassR[3]))
	c.shadePoly([]pt{{5.5, 4}, {10.5, 4}, {8.5, 7}, {7.5, 7}}, flat(freshCols[0]))
	c.shadePoly([]pt{{6, 11}, {10, 11}, {11, 13}, {5, 13}}, flat(freshCols[0]))
	c.rect(8, 8, 1, 3, freshCols[0])
}

func iconSpoiled(c *canvas) {
	drawBox(c, 8, 11, boxDims{0.4375, 5}, "crate", 0)
	c.mapPixels(func(col color.NRGBA) color.NRGBA { return mix(col, mouldR[2], 0.6) })
	for _, p := range [][2]int{{3, 2}, {12, 3}, {8, 1}} {
		c.dot(p[0], p[1], ink)
		c.dot(p[0]-1, p[1]-1, withAlpha(white, 200))
	}
}
