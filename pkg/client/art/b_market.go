package art

import "image/color"

func post(s iso, gx, gy float64) {
	s.box(gx, gy, gx+0.12, gy+0.12, 4, 24, faces{top: flat(woodR[3]), left: flat(woodR[2]), right: flat(woodR[0])})
}

func arrow(c *canvas, x, y, w float64, variant int) {
	body := []color.NRGBA{woodR[3], woodR[2], plasterR[2]}[variant]
	c.shadePoly([]pt{{x, y}, {x + w, y - 2}, {x + w + 3, y}, {x + w, y + 3}, {x, y + 3}}, flat(body))
	c.dot(int(x)+2, int(y)+1, ink)
	c.dot(int(x)+4, int(y)+1, ink)
}

func mkColumn(s iso, gx, gy, z0, z1 float64) {
	if s.lk.age != ageEmpire {
		s.post(gx, gy, z0, z1)
		return
	}
	s.turn(lathe{cx: gx, cy: gy, z: z0, h: 2, r: 0.075}, toneMat(marbleR), matTrim)
	s.turn(lathe{cx: gx, cy: gy, z: z0 + 2, h: z1 - z0 - 4, r: 0.055}, toneMat(marbleR), matTrim)
	s.turn(lathe{cx: gx, cy: gy, z: z1 - 2, h: 2, r: 0.08}, toneMat(marbleR), matTrim)
}
