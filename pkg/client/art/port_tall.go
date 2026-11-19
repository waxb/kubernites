package art

import "math"

const beaconTop = 44

func paintPortWarehouse(c *canvas, l PortLook) {
	sz := buildingLikeSize(2, warehouseHeight)
	paintPortAged(c, portLook("warehouse", l.Age, redR), 2, sz, func(s iso) {
		s.apron(2, 0.03)
		x0, y0, x1, y1, top := 0.25, 0.25, 1.75, 1.75, 27.0
		s.walls(x0, y0, x1, y1, 0, top)
		s.doorway(sideL(y1), 0.62, 0.5, 13)
		s.window(sideL(y1), 1.35, 15)
		s.window(sideR(x1), 0.55, 15)
		s.window(sideR(x1), 1.2, 15)
		if l.Variant == 1 {
			s.roofGableY(roofBox{x0, y0, x1, y1, top, 15, 0.14}, s.roofMat(), s.kit().wall)
			warehouseHoist(s, s.p(1.0, y1+0.12, top+9))
		} else {
			s.roofGableX(roofBox{x0, y0, x1, y1, top, 15, 0.14}, s.roofMat(), s.kit().wall)
			warehouseHoist(s, s.p(x1+0.12, 1.0, top+9))
		}
		s.crateAt(1.3, 1.78, 0.2, 0)
		s.crateAt(0.3, 1.8, 0.16, 0)
	})
}

func paintPortLighthouse(c *canvas, l PortLook) {
	sz := buildingLikeSize(1, lighthouseHeight)
	var fire float64
	paintPortAged(c, portLook("lighthouse", l.Age, redR), 1, sz, func(s iso) {
		s.apron(1, 0.02)
		base := s.p(0.5, 0.5, 0)
		switch clampAge(l.Age) {
		case ageDark:
			fire = beacon(s, base)
		case ageFeudal:
			fire = lightTower(s, base, rubbleR, 44)
		case ageCastle:
			fire = lightTower(s, base, stoneR, 52)
		default:
			fire = lightTower(s, base, marbleR, 52)
		}
	})
	base := isoFrame(c, 1, sz.ax, sz.ay).p(0.5, 0.5, 0)
	lighthouseGlow(c, base.x, base.y-fire)
}

func beacon(s iso, base pt) float64 {
	s.pen(matTrim, func() {
		for _, c := range [][2]float64{{0.22, 0.22}, {0.78, 0.22}, {0.22, 0.78}, {0.78, 0.78}} {
			foot, head := s.p(c[0], c[1], 0), s.p(0.5+(c[0]-0.5)*0.4, 0.5+(c[1]-0.5)*0.4, beaconTop)
			s.c.line(int(foot.x), int(foot.y), int(head.x), int(head.y), driftR[1])
			s.c.line(int(foot.x)+1, int(foot.y), int(head.x)+1, int(head.y), driftR[2])
		}
		for z := 10.0; z < beaconTop; z += 11 {
			k := z / beaconTop
			a, b := s.p(0.22+0.17*k, 0.78-0.17*k, z), s.p(0.78-0.17*k, 0.78-0.17*k, z)
			s.c.line(int(a.x), int(a.y), int(b.x), int(b.y), driftR[0])
		}
	})
	s.block(0.36, 0.36, 0.64, 0.64, beaconTop, beaconTop+2, logMat(s.lk.seed), flatMat(driftR[2]), matWall)
	top := s.p(0.5, 0.5, beaconTop+2)
	s.pen(matTrim, func() {
		s.c.rect(int(top.x)-4, int(top.y)-4, 9, 4, ironR[1])
		s.c.rect(int(top.x)-4, int(top.y)-4, 9, 1, ironR[2])
	})
	flame(s.c, top.x, top.y-4, 8, 3.2, 0.7)
	return beaconTop + 8
}

func lightTower(s iso, base pt, stone ramp, height int) float64 {
	s.pen(matStone, func() {
		for z := 0; z <= height; z++ {
			s.c.penZ = liftByte(float64(z))
			r := 10 - 3*float64(max(z-3, 0))/float64(height-3)
			rmp := stone
			if s.lk.age >= ageEmpire && z%13 == 12 {
				rmp = goldR
			}
			s.towerRing(rmp, base.x, base.y, r, z)
		}
	})
	lighthouseDetails(s, base)
	lightCrown(s, base, float64(height))
	return float64(height) + 7
}

func lightCrown(s iso, base pt, h float64) {
	cx, cy := base.x, base.y
	rim := portMatOf(s.lk.age).stone
	for z := h; z <= h+2; z++ {
		s.c.ellipse(cx, cy-z, 10, 5, rim[1+min(int(z-h), 2)])
	}
	for z := h + 3; z <= h+12; z++ {
		lanternRing(s.c, cx, cy-z, int(z-h)+52)
	}
	flame(s.c, cx, cy-h-4, 7, 2.6, 0.6)
	frame := woodR[0]
	if s.lk.age >= ageEmpire {
		frame = goldR[1]
	}
	for i := -9; i <= 9; i += 3 {
		s.c.rect(int(cx)+i, int(cy-h)-6+int(math.Abs(float64(i))/3), 1, 3, frame)
	}
	lightRoof(s, cx, cy-h-13)
}

func lightRoof(s iso, cx, roofY float64) {
	switch clampAge(s.lk.age) {
	case ageFeudal:
		s.c.ellipse(cx, roofY, 8, 4, woodR[0])
		s.c.shadePoly([]pt{{cx - 8, roofY}, {cx, roofY - 9}, {cx, roofY + 4}}, flat(woodR[2]))
		s.c.shadePoly([]pt{{cx, roofY - 9}, {cx + 8, roofY}, {cx, roofY + 4}}, flat(woodR[1]))
	case ageEmpire:
		s.c.ellipse(cx, roofY+1, 8, 4, goldR[0])
		s.c.ellipse(cx, roofY-2, 7, 6, goldR[1])
		s.c.ellipse(cx-2, roofY-3, 4, 4, goldR[2])
		s.c.dot(int(cx)-3, int(roofY)-5, goldR[3])
	default:
		s.c.ellipse(cx, roofY, 8, 4, redR[0])
		s.c.shadePoly([]pt{{cx - 8, roofY}, {cx, roofY - 10}, {cx, roofY + 4}}, flat(redR[2]))
		s.c.shadePoly([]pt{{cx, roofY - 10}, {cx + 8, roofY}, {cx, roofY + 4}}, flat(redR[1]))
	}
	s.c.rect(int(cx), int(roofY)-13, 1, 3, goldR[2])
}
