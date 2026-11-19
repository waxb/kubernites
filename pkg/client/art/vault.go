package art

import "image/color"

const (
	vaultX0, vaultX1 = 0.1, 0.9
	vaultWall        = 13.0
	vaultRoof        = 7.0
)

var (
	waxRed      = redR[1]
	vaultIron   = ramp{rgb(40, 38, 44), rgb(64, 62, 68), rgb(96, 94, 98), rgb(150, 146, 144)}
	bannerTones = []ramp{redR, blueR, greenR, goldR}
)

func vaultFrame(c *canvas) iso {
	return isoFrame(c, 1, VaultAX, VaultAY)
}

func paintVault(c *canvas, state string, shelves int) {
	g := vaultFrame(c).p(0.5, 0.5, 0)
	c.ellipse(g.x, g.y+1, 17, 8, withAlpha(shadow, 70))
	inked(c, func(c *canvas) { paintVaultBody(c, state, shelves) })
}

func paintVaultBody(c *canvas, state string, shelves int) {
	s := vaultFrame(c)
	vaultBody(s)
	vaultShelves(s, shelves, state)
	vaultDoor(s)
	switch state {
	case "bound":
		vaultPlaque(s, true, false)
	case "released":
		vaultPlaque(s, false, true)
	case "terminating":
		vaultPlaque(s, true, false)
		vaultHold(s)
	}
}

func vaultBody(s iso) {
	s.block(0.04, 0.04, 0.96, 0.96, 0, 2, ashlarMat(stoneR, 41), flatMat(stoneR[2]), matStone)
	s.block(vaultX0, vaultX0, vaultX1, vaultX1, 2, vaultWall, ashlarMat(stoneR, 43), flatMat(stoneR[3]), matWall)
	s.block(vaultX0-0.03, vaultX0-0.03, vaultX1+0.03, vaultX1+0.03, vaultWall, vaultWall+2, flatMat(stoneR[1]), flatMat(stoneR[3]), matTrim)
	s.pyramid(vaultX0-0.03, vaultX0-0.03, vaultX1+0.03, vaultX1+0.03, vaultWall+2, vaultRoof, slateR)
	apex := s.p(0.5, 0.5, vaultWall+2+vaultRoof)
	s.c.rect(int(apex.x), int(apex.y)-2, 1, 2, vaultIron[2])
}

func vaultDoor(s iso) {
	s.pen(matTrim, func() {
		s.onLeft(vaultX1, 0.34, 0.58, 2, 10, func(x, y int) color.NRGBA {
			if y%3 == 0 {
				return vaultIron[1]
			}
			return woodR[1]
		})
	})
	ring := s.p(0.52, vaultX1, 6)
	s.c.dot(int(ring.x), int(ring.y), goldR[2])
	arch0, arch1 := s.p(0.32, vaultX1, 11), s.p(0.6, vaultX1, 11)
	s.c.line(int(arch0.x), int(arch0.y), int(arch1.x), int(arch1.y), stoneR[3])
}

func vaultShelves(s iso, level int, state string) {
	s.pen(matGlass, func() { s.onRight(vaultX1, 0.3, 0.72, 4, 12, flat(ink)) })
	books := []int{0, 2, 3, 6}[min(max(level, 0), VaultShelfLevels-1)]
	for row, z := range []float64{4, 8} {
		a, b := s.p(vaultX1, 0.31, z), s.p(vaultX1, 0.71, z)
		s.c.line(int(a.x), int(a.y), int(b.x), int(b.y), woodR[1])
		if state == "released" {
			continue
		}
		for i := 0; i < books; i++ {
			p := s.p(vaultX1, 0.69-float64(i)*0.0625, z+1)
			col := spines[int(hash(i, row, 71))%len(spines)]
			s.c.rect(int(p.x), int(p.y)-3, 1, 3, col)
			s.c.dot(int(p.x), int(p.y)-3, mix(col, white, 0.3))
		}
	}
	vaultBars(s)
}

func vaultBars(s iso) {
	for _, v := range []float64{0.3, 0.72} {
		a, b := s.p(vaultX1, v, 4), s.p(vaultX1, v, 12)
		s.c.line(int(a.x), int(a.y), int(b.x), int(b.y), vaultIron[2])
	}
	a, b := s.p(vaultX1, 0.3, 12), s.p(vaultX1, 0.72, 12)
	s.c.line(int(a.x), int(a.y), int(b.x), int(b.y), stoneR[3])
}

func vaultPlaque(s iso, sealed, broken bool) {
	p := s.p(0.72, vaultX1, 10)
	x, y := int(p.x)-2, int(p.y)-3
	s.c.rect(x, y, 5, 5, tkParchment)
	s.c.rect(x, y+4, 5, 1, tkParchmentDark)
	s.c.rect(x+1, y+1, 3, 1, inkSft)
	switch {
	case sealed:
		s.c.rect(x+1, y+3, 3, 2, waxRed)
		s.c.dot(x+2, y+3, redR[2])
		s.c.dot(x+2, y+5, redR[1])
	case broken:
		s.c.dot(x+1, y+3, redR[0])
		s.c.dot(x+3, y+4, redR[0])
		s.c.dot(x+2, y+3, inkSft)
		s.c.dot(x+4, y+1, tkParchmentDark)
	}
}

func vaultHold(s iso) {
	a, b := s.p(0.3, vaultX1, 8), s.p(0.62, vaultX1, 4)
	s.c.line(int(a.x), int(a.y), int(b.x), int(b.y), ropeR[2])
	s.c.line(int(a.x), int(a.y)+1, int(b.x), int(b.y)+1, ropeR[0])
	m := s.p(0.46, vaultX1, 6)
	s.c.rect(int(m.x)-1, int(m.y)+1, 3, 3, greyR[2])
	s.c.dot(int(m.x), int(m.y)+2, greyR[0])
	s.c.dot(int(m.x)-1, int(m.y)+1, greyR[3])
}

func paintVaultDetach(c *canvas, banner int) {
	s := vaultFrame(c)
	for _, pair := range [][2]float64{{10, 2}, {2, 10}} {
		a, b := s.p(0.3, vaultX1, pair[0]), s.p(0.62, vaultX1, pair[1])
		stampChain(c, a, b, false)
	}
	lock := s.p(0.46, vaultX1, 6)
	s.c.rect(int(lock.x)-2, int(lock.y), 4, 3, vaultIron[1])
	s.c.dot(int(lock.x)-1, int(lock.y)-1, vaultIron[2])
	s.c.dot(int(lock.x), int(lock.y)-1, vaultIron[2])
	s.c.dot(int(lock.x)-1, int(lock.y)+1, goldR[2])
	keepBanner(s, bannerTones[((banner%len(bannerTones))+len(bannerTones))%len(bannerTones)])
}

func keepBanner(s iso, tone ramp) {
	pole := s.p(0.2, 0.96, 0)
	top := s.p(0.2, 0.96, 19)
	s.c.line(int(pole.x), int(pole.y), int(top.x), int(top.y), woodR[2])
	x, y := int(top.x)+1, int(top.y)+1
	s.c.rect(x, y, 5, 7, tone[2])
	s.c.rect(x, y, 1, 7, tone[1])
	s.c.rect(x+1, y+2, 3, 2, tone[3])
	s.c.dot(x+1, y+7, tone[1])
	s.c.dot(x+3, y+7, tone[1])
	s.c.dot(int(top.x), int(top.y)-1, goldR[2])
}

func stampChain(c *canvas, a, b pt, red bool) {
	dx, dy := b.x-a.x, b.y-a.y
	n := int(max(abs(int(dx)), abs(int(dy)))/2) + 1
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		x, y := int(a.x+dx*t), int(a.y+dy*t)
		tone := chainTone(red)
		if i%2 == 0 {
			c.dot(x, y, tone[1])
			c.dot(x+1, y, tone[3])
			continue
		}
		c.dot(x, y, tone[2])
	}
}

func chainTone(red bool) ramp {
	if red {
		return ramp{redR[1], redR[2], redR[3], orangeR[3]}
	}
	return vaultIron
}
