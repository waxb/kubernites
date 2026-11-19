package art

import "image/color"

func (s iso) annexWall() (mat, uint8) {
	if s.lk.age <= ageFeudal {
		return plankMat(woodR, s.lk.seed+9), matWall
	}
	k := s.kit()
	return k.wall, k.wallTag
}

func (s iso) leanTo(x0, y0, x1, y1, zLow, zHigh float64, alongX bool) {
	wall, tag := s.annexWall()
	s.block(x0, y0, x1, y1, 0, zLow, wall, wall, tag)
	s.pen(tag, func() {
		if alongX {
			s.c.shadePoly([]pt{s.p(x1, y0, zLow), s.p(x1, y0, zHigh), s.p(x1, y1, zLow)}, s.rightMat(x1, wall))
			return
		}
		s.c.shadePoly([]pt{s.p(x0, y1, zLow), s.p(x0, y1, zHigh), s.p(x1, y1, zLow)}, s.leftMat(y1, wall))
	})
	e := 0.06
	if alongX {
		s.roofShed(x0-e, y0, x1+e, y1+e, zLow-1, zHigh, true, s.roofMat())
		return
	}
	s.roofShed(x0, y0-e, x1+e, y1+e, zLow-1, zHigh, false, s.roofMat())
}

func (s iso) post(gx, gy, z0, z1 float64) {
	if s.lk.age == ageEmpire {
		s.turn(lathe{cx: gx, cy: gy, z: z0, h: z1 - z0, r: 0.05}, toneMat(marbleR), matTrim)
		return
	}
	col := faces{top: flat(woodR[3]), left: flat(woodR[2]), right: flat(woodR[0])}
	if s.lk.age == ageCastle {
		col = faces{top: flat(stoneR[3]), left: flat(stoneR[2]), right: flat(stoneR[1])}
	}
	s.pen(matTrim, func() { s.box(gx-0.04, gy-0.04, gx+0.04, gy+0.04, z0, z1, col) })
}

func (s iso) porch(y1, u0, u1, depth, h float64) {
	for _, u := range []float64{u0, u1} {
		s.post(u, y1+depth, 0, h)
	}
	s.roofShed(u0-0.06, y1-0.02, u1+0.06, y1+depth+0.06, h, h+5, true, s.roofMat())
	if s.lk.age == ageEmpire {
		mid := s.p((u0+u1)/2, y1+depth+0.06, h+2)
		s.pen(matTrim, func() {
			s.c.shadePoly([]pt{{mid.x - 9, mid.y + 2}, {mid.x, mid.y - 4}, {mid.x + 9, mid.y + 2}}, flat(marbleR[2]))
			s.c.line(int(mid.x)-9, int(mid.y)+2, int(mid.x)+9, int(mid.y)+2, goldR[2])
		})
	}
}

func (s iso) arcade(f side, u0, u1, z0, h float64, n int) {
	span := (u1 - u0) / float64(n)
	dark := rgb(44, 34, 36)
	for i := 0; i < n; i++ {
		a := u0 + float64(i)*span + span*0.2
		b := a + span*0.6
		s.paint(f, matTrim, a-0.02, b+0.02, z0+h-2, z0+h+1, stoneR[3])
		s.paint(f, matTrim, a, b, z0, z0+h-2, dark)
		s.paint(f, matTrim, a+span*0.1, b-span*0.1, z0+h-2, z0+h, dark)
	}
}

func (s iso) banner(f side, u, z, w, h float64) {
	cloth := s.lk.id
	s.pen(matTrim, func() {
		s.on(f, u, u+w, z-h, z, func(x, y int) color.NRGBA {
			if x%3 == 0 {
				return cloth[1]
			}
			return cloth[2]
		})
		s.on(f, u+w*0.35, u+w*0.65, z-h*0.6, z-h*0.4, flat(goldR[3]))
		s.on(f, u-0.02, u+w+0.02, z, z+1, flat(woodR[0]))
	})
}
