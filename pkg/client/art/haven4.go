package art

import (
	"image/color"
	"math"
)

const (
	warehouseHeight  = 34
	lighthouseHeight = 72
	lighthouseFire   = 59
)

var slateR = ramp{rgb(46, 54, 70), rgb(70, 82, 102), rgb(100, 114, 134), rgb(146, 160, 176)}

func timber(u, z, z0, z1 float64, lit bool) color.NRGBA {
	plaster, beam := plasterR[1], woodR[0]
	if lit {
		plaster = plasterR[3]
	}
	f := u*2.5 - math.Floor(u*2.5)
	t := (z - z0) / (z1 - z0)
	switch {
	case f < 0.1, z-z0 < 1.5, z1-z < 1.5, math.Abs(t-0.5) < 0.04:
		return beam
	case t < 0.5 && math.Abs(f-t*2) < 0.12:
		return woodR[1]
	}
	return plaster
}

func paintWarehouse(c *canvas, v int) {
	sz := buildingLikeSize(2, warehouseHeight)
	gnd, body := newCanvas(sz.w, sz.h), newCanvas(sz.w, sz.h)
	body.enableTags()
	s := isoFrame(body, 2, sz.ax, sz.ay).with(newLook("warehouse", ageFeudal, v))
	s.gnd = gnd
	s.with(newLook("warehouse", ageCastle, v)).apron(2, 0.03)
	warehouseBody(s, v)
	finishRough(c, gnd, body, s.lk)
}

func timberMat(z0, z1 float64) mat {
	return func(p surf) color.NRGBA { return timber(p.u/faceU, p.v, z0, z1, p.light > 0) }
}

func warehouseBody(s iso, v int) {
	x0, y0, x1, y1, z0, z1 := 0.25, 0.25, 1.75, 1.75, 4.0, 27.0
	stone := rubbleMat(stoneR, s.lk.seed)
	s.block(x0, y0, x1, y1, 0, z0, stone, stone, matStone)
	s.block(x0, y0, x1, y1, z0, z1, timberMat(z0, z1), flatMat(plasterR[2]), matWall)
	s.doorL(y1, 0.62, 0.5, 13)
	s.window(sideL(y1), 1.35, 16)
	s.window(sideR(x1), 0.55, 16)
	s.window(sideR(x1), 1.2, 16)
	if v == 1 {
		s.roofGableY(roofBox{x0, y0, x1, y1, z1, 15, 0.14}, slateMat(slateR, s.lk.seed), timberMat(z0, z1))
		warehouseHoist(s, s.p(1.0, y1+0.12, z1+9))
	} else {
		s.roofGableX(roofBox{x0, y0, x1, y1, z1, 15, 0.14}, tileMat(redR, s.lk.seed), timberMat(z0, z1))
		warehouseHoist(s, s.p(x1+0.12, 1.0, z1+9))
	}
	s.crateAt(1.3, 1.78, 0.2, 0)
	s.crateAt(0.3, 1.8, 0.16, 0)
}

func warehouseHoist(s iso, p pt) {
	x, y := int(p.x), int(p.y)
	s.c.rect(x, y, 6, 2, woodR[1])
	s.c.rect(x, y, 6, 1, woodR[2])
	s.c.rect(x+5, y+2, 1, 8, ropeR[1])
	s.c.rect(x+4, y+10, 3, 1, greyR[1])
}

func stoneTone(light float64) int {
	switch {
	case light > 0.55:
		return 3
	case light > -0.1:
		return 2
	case light > -0.6:
		return 1
	}
	return 0
}

func (s iso) towerLevel(cx, cy, r float64, z int) {
	s.towerRing(stoneR, cx, cy, r, z)
}

func (s iso) towerRing(rmp ramp, cx, cy, r float64, z int) {
	yc := cy - float64(z)
	for y := int(yc - r/2 - 1); y <= int(yc+r/2+1); y++ {
		for x := int(cx - r - 1); x <= int(cx+r+1); x++ {
			dx, dy := (float64(x)+0.5-cx)/r, (float64(y)+0.5-yc)/(r/2)
			if dx*dx+dy*dy > 1 {
				continue
			}
			idx := stoneTone(-dx*1.1 + 0.15)
			phi := math.Acos(math.Max(-1, math.Min(1, dx)))
			joint := phi*r/3.2 + float64((z/4)%2)*0.5
			if z%4 == 0 || joint-math.Floor(joint) < 0.14 {
				idx = max(idx-1, 0)
			}
			s.c.dot(x, y, rmp[idx])
		}
	}
}

func paintLighthouse(c *canvas, _ int) {
	sz := buildingLikeSize(1, lighthouseHeight)
	gnd, body := newCanvas(sz.w, sz.h), newCanvas(sz.w, sz.h)
	body.enableTags()
	s := isoFrame(body, 1, sz.ax, sz.ay).with(newLook("lighthouse", ageCastle, 0))
	s.gnd = gnd
	s.apron(1, 0.02)
	base := s.p(0.5, 0.5, 0)
	s.pen(matStone, func() {
		for z := 0; z <= 52; z++ {
			s.c.penZ = liftByte(float64(z))
			s.towerLevel(base.x, base.y, 10-3*float64(max(z-3, 0))/49, z)
		}
	})
	lighthouseDetails(s, base)
	lighthouseTop(s, base)
	finishRough(c, gnd, body, s.lk)
	lighthouseGlow(c, base.x, base.y-lighthouseFire)
}

func lighthouseDetails(s iso, base pt) {
	x, y := int(base.x), int(base.y)
	s.c.rect(x-4, y-11, 4, 7, ink)
	s.c.rect(x-3, y-10, 2, 6, woodR[1])
	s.c.dot(x-2, y-7, goldR[2])
	for _, z := range []int{22, 38} {
		s.c.rect(x-3, y-z, 2, 4, ink)
		s.c.dot(x-3, y-z+1, lampLit)
	}
}

func lighthouseTop(s iso, base pt) {
	cx, cy := base.x, base.y
	for z := 52; z <= 54; z++ {
		s.c.ellipse(cx, cy-float64(z), 10, 5, stoneR[1+min(z-52, 2)])
	}
	for z := 55; z <= 64; z++ {
		lanternRing(s.c, cx, cy-float64(z), z)
	}
	flame(s.c, cx, cy-56, 7, 2.6, 0.6)
	for i := -9; i <= 9; i += 3 {
		s.c.rect(int(cx)+i, int(cy)-58+int(math.Abs(float64(i))/3), 1, 3, woodR[0])
	}
	roofY := cy - 65
	s.c.ellipse(cx, roofY, 8, 4, redR[0])
	s.c.shadePoly([]pt{{cx - 8, roofY}, {cx, roofY - 10}, {cx, roofY + 4}}, flat(redR[2]))
	s.c.shadePoly([]pt{{cx, roofY - 10}, {cx + 8, roofY}, {cx, roofY + 4}}, flat(redR[1]))
	s.c.rect(int(cx), int(roofY)-13, 1, 3, goldR[2])
}

func lanternRing(c *canvas, cx, yc float64, z int) {
	for x := int(cx - 6); x <= int(cx+6); x++ {
		dx := (float64(x) + 0.5 - cx) / 6
		if math.Abs(dx) > 1 {
			continue
		}
		dy := math.Sqrt(1-dx*dx) * 3
		col := lampLit
		switch {
		case math.Abs(dx) > 0.8 || math.Abs(dx) < 0.1 || z == 64:
			col = inkSft
		case z < 58:
			col = fireR[2]
		}
		c.dot(x, int(yc+dy), col)
		c.dot(x, int(yc-dy), col)
	}
}

func lighthouseGlow(c *canvas, cx, cy float64) {
	for y := int(cy - 14); y <= int(cy+14); y++ {
		for x := int(cx - 16); x <= int(cx+16); x++ {
			d := math.Hypot((float64(x)+0.5-cx)/16, (float64(y)+0.5-cy)/12)
			if d > 1 || c.at(x, y).A != 0 {
				continue
			}
			c.dot(x, y, withAlpha(goldR[3], uint8(70*(1-d))))
		}
	}
}
