package art

var houseHeights = [PortLevels]int{20, 32, 42, 58}

func houseSize(l PortLook) size {
	return buildingLikeSize(1, houseHeights[ClampPortLevel(l.Level)-1])
}

func portLook(kind string, age int, id ramp) look {
	l := newLook("port-"+kind, age, 0)
	l.id = id
	return l
}

func paintPortAged(c *canvas, lk look, n int, sz size, paint func(s iso)) {
	gnd, body := newCanvas(sz.w, sz.h), newCanvas(sz.w, sz.h)
	body.enableTags()
	s := isoFrame(body, n, sz.ax, sz.ay).with(lk)
	s.gnd = gnd
	paint(s)
	finishRough(c, gnd, body, lk)
}

func paintPortHouse(c *canvas, l PortLook) {
	paintPortAged(c, portLook("house", l.Age, blueR), 1, houseSize(l), func(s iso) {
		switch l.Level {
		case 1:
			netShed(s)
		case 2:
			harborOffice(s)
		case 3:
			customsHouse(s, false)
		default:
			customsHouse(s, true)
		}
	})
}

func netShed(s iso) {
	s.leanTo(0.1, 0.1, 0.64, 0.62, 9, 14, true)
	s.doorway(sideL(0.62), 0.24, 0.2, 8)
	netRack(s, 0.8, 0.14, 0.8, 0.6)
	s.crate(0.7, 0.7, 0.24)
	s.barrel(0.3, 0.86)
}

func netRack(s iso, x0, y0, x1, y1 float64) {
	s.post(x0, y0, 0, 11)
	s.post(x1, y1, 0, 11)
	a, b := s.p(x0, y0, 10), s.p(x1, y1, 10)
	s.pen(matTrim, func() {
		s.c.line(int(a.x), int(a.y), int(b.x), int(b.y), woodR[1])
		for i := 0; i <= 6; i++ {
			t := float64(i) / 6
			top := pt{a.x + (b.x-a.x)*t, a.y + (b.y-a.y)*t}
			drop := 5 + 2*float64(i%2)
			s.c.line(int(top.x), int(top.y)+1, int(top.x), int(top.y+drop), ropeR[1])
		}
		for k := 0; k < 2; k++ {
			s.c.line(int(a.x), int(a.y)+3+2*k, int(b.x), int(b.y)+3+2*k, withAlpha(ropeR[2], 200))
		}
	})
}

func harborOffice(s iso) {
	x0, y0, x1, y1 := 0.12, 0.12, 0.88, 0.86
	s.walls(x0, y0, x1, y1, 0, 15)
	s.doorway(sideL(y1), 0.36, 0.22, 10)
	s.window(sideR(x1), 0.38, 6)
	s.roofGableX(roofBox{x0, y0, x1, y1, 15, 10, 0.08}, s.roofMat(), s.kit().wall)
	s.lantern(0.95, 0.94)
	s.flag(0.5, 0.46, 24, 7, s.lk.id)
}

func customsHouse(s iso, hall bool) {
	x0, y0, x1, y1 := 0.1, 0.1, 0.9, 0.9
	top := 26.0
	s.walls(x0, y0, x1, y1, 0, top)
	s.doorway(sideL(y1), 0.38, 0.24, 11)
	s.window(sideL(y1), 0.14, 15)
	s.window(sideL(y1), 0.66, 15)
	s.window(sideR(x1), 0.2, 6)
	s.window(sideR(x1), 0.2, 16)
	s.window(sideR(x1), 0.56, 16)
	s.roofHip(roofBox{x0, y0, x1, y1, top, 10, 0.08}, s.roofMat())
	if hall {
		harborCupola(s, top+8)
		return
	}
	s.chimney(0.3, 0.3, top+4, top+11)
	s.flag(0.7, 0.5, top+6, 8, s.lk.id)
}

func harborCupola(s iso, z float64) {
	k := s.kit()
	s.block(0.36, 0.34, 0.64, 0.62, z-4, z+8, k.wall, k.wall, k.wallTag)
	s.window(sideL(0.62), 0.42, z-1)
	s.window(sideR(0.64), 0.4, z-1)
	if s.lk.age >= ageEmpire {
		s.roofDome(0.5, 0.48, 0.2, z+8, 8, s.domeMat())
		s.finial(0.5, 0.48, z+16, 4)
		return
	}
	s.roofPyramid(roofBox{0.34, 0.32, 0.66, 0.64, z + 8, 9, 0.04}, s.roofMat())
	s.flag(0.5, 0.48, z+16, 5, s.lk.id)
}
