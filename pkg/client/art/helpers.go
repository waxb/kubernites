package art

import (
	"image/color"
	"math"
)

var HelperKinds = []string{"seal-smithy", "key-gate", "bell-watch", "cache-store", "caravan-yard"}

var HelperArt = map[string]string{
	"cert-manager": "seal-smithy", "coredns": "stall-herald", "keycloak": "key-gate", "goldpinger": "bell-watch",
	"squid": "cache-store", "rebalancer": "caravan-yard", "descheduler": "caravan-yard",
}

func init() {
	agedKind("seal-smithy", paintSealSmithy)
	agedKind("key-gate", paintKeyGate)
	agedKind("bell-watch", paintBellWatch)
	agedKind("cache-store", paintCacheStore)
	agedKind("caravan-yard", paintCaravanYard)
}

func paintSealSmithy(s iso) {
	s.apron(2, 0.05)
	x0, y0, x1, y1 := 0.3, 0.3, 1.5, 1.35
	top := s.hall(hallSpec{x0: x0, y0: y0, x1: x1, y1: y1, h: 22, rise: 12})
	hlpForgeStack(s, top)
	forgeMouth(s, y1, 0.55, 0.45, top)
	s.window(sideR(x1), 0.72, math.Round(top*0.45))
	switch s.lk.variant {
	case 1:
		s.leanTo(x1, 0.35, x1+0.35, 0.95, math.Round(top*0.4), math.Round(top*0.62), false)
	case 2:
		hlpAnvil(s, 0.3, 1.62)
		s.window(sideL(y1), 1.2, math.Round(top*0.45))
	}
	s.aloft(top-8, func() { sealSign(s, x1, 1.1, top-2) })
	screwPress(s, 1.65, 1.62)
	s.props([][2]float64{{0.22, 1.85}, {1.88, 0.3}}, 31)
}

func hlpAnvil(s iso, gx, gy float64) {
	s.box(gx-0.06, gy-0.06, gx+0.06, gy+0.06, 0, 4, faces{top: flat(woodR[3]), left: flat(woodR[2]), right: flat(woodR[0])})
	s.box(gx-0.12, gy-0.05, gx+0.1, gy+0.05, 4, 7, faces{top: flat(greyR[3]), left: flat(greyR[1]), right: flat(greyR[0])})
	p := s.p(gx+0.1, gy, 7)
	s.c.dot(int(p.x)+1, int(p.y), greyR[2])
}

func hlpForgeStack(s iso, top float64) {
	cx, cy, r := 0.55, 0.55, 0.15
	peak := min(top+13, 40)
	switch s.lk.age {
	case ageDark:
		s.block(cx-r, cy-r, cx+r, cy+r, top-4, peak-2, wattleMat(s.lk.seed+7), flatMat(ink), matWall)
	case ageFeudal:
		s.block(cx-r, cy-r, cx+r, cy+r, top-4, peak, rubbleMat(stoneR, s.lk.seed+7), flatMat(ink), matStone)
	default:
		s.turn(lathe{cx: cx, cy: cy, z: top - 4, h: peak - top + 4, r: r, shape: shapeOct}, s.kit().wall, matStone)
		cap := stoneR[3]
		if s.lk.age == ageEmpire {
			cap = goldR[2]
		}
		s.turn(lathe{cx: cx, cy: cy, z: peak, h: 2, r: r + 0.03, shape: shapeOct}, flatMat(cap), matTrim)
	}
	ember := s.p(cx, cy, peak)
	s.aloft(peak, func() {
		s.c.rect(int(ember.x)-2, int(ember.y)-1, 4, 1, fireR[1])
		s.c.dot(int(ember.x), int(ember.y)-2, withAlpha(fireR[2], 180))
		s.c.dot(int(ember.x)+1, int(ember.y)-4, withAlpha(greyR[2], 150))
	})
}

func forgeMouth(s iso, y1, u, w, top float64) {
	h := math.Round(min(top*0.55, 12))
	s.paint(sideL(y1), matTrim, u-0.04, u+w+0.04, 0, h+2, ink)
	s.pen(matTrim, func() {
		s.onLeft(y1, u, u+w, 0, h, func(x, y int) color.NRGBA {
			_, z := s.leftUV(x, y, y1)
			switch {
			case z < 2:
				return fireR[2]
			case z < 5:
				return fireR[1]
			}
			return rgb(60, 30, 26)
		})
	})
	p := s.p(u+w/2, y1+0.25, 0)
	s.c.rect(int(p.x)-3, int(p.y)-4, 6, 2, greyR[1])
	s.c.rect(int(p.x)-2, int(p.y)-2, 3, 2, greyR[0])
	s.c.rect(int(p.x)-3, int(p.y)-4, 6, 1, greyR[3])
}

func sealSign(s iso, gx, gy, z float64) {
	c := s.p(gx, gy, z)
	s.c.rect(int(c.x), int(c.y)-5, 5, 1, woodR[1])
	s.c.disc(c.x+4.5, c.y+1, 3.5, sealR[1])
	s.c.disc(c.x+4, c.y+0.5, 2.4, sealR[2])
	s.c.dot(int(c.x)+4, int(c.y)+1, sealR[0])
	s.c.line(int(c.x)+2, int(c.y)+4, int(c.x)+1, int(c.y)+7, goldR[2])
	s.c.line(int(c.x)+6, int(c.y)+4, int(c.x)+7, int(c.y)+7, goldR[1])
}

func screwPress(s iso, gx, gy float64) {
	frame := faces{top: flat(woodR[3]), left: flat(woodR[2]), right: flat(woodR[0])}
	s.box(gx-0.25, gy-0.12, gx+0.25, gy+0.12, 0, 4, frame)
	for _, dx := range []float64{-0.22, 0.14} {
		s.box(gx+dx, gy-0.04, gx+dx+0.08, gy+0.04, 4, 17, frame)
	}
	s.box(gx-0.25, gy-0.06, gx+0.25, gy+0.06, 17, 20, frame)
	top, bot := s.p(gx, gy, 20), s.p(gx, gy, 8)
	s.c.line(int(top.x), int(top.y)-3, int(bot.x), int(bot.y), greyR[2])
	s.c.rect(int(top.x)-3, int(top.y)-4, 7, 1, greyR[1])
	s.c.rect(int(bot.x)-2, int(bot.y), 5, 2, goldR[1])
	seal := s.p(gx, gy, 4)
	s.c.disc(seal.x, seal.y-1, 1.8, sealR[2])
}

func paintKeyGate(s iso) {
	s.apron(2, 0.04)
	s.tower(towerSpec{cx: 0.47, cy: 0.47, r: 0.26, h: 40, roofH: 12})
	s.tower(towerSpec{cx: 1.53, cy: 0.47, r: 0.26, h: 36, roofH: 12})
	top := hlpKeyBlock(s, 0.35, 0.45, 1.6, 1.65)
	keyDoor(s, 1.65, 0.8, 0.5, top)
	s.window(sideR(1.6), 1.15, math.Round(top*0.6))
	if s.lk.age >= ageFeudal {
		s.banner(sideL(1.65), 0.45, math.Round(top*0.85), 0.16, 9)
		s.banner(sideL(1.65), 1.42, math.Round(top*0.85), 0.16, 9)
	}
	s.props([][2]float64{{0.2, 1.85}, {1.85, 1.9}}, 33)
}

func hlpKeyBlock(s iso, x0, y0, x1, y1 float64) float64 {
	if s.lk.age <= ageFeudal {
		return s.hall(hallSpec{x0: x0, y0: y0, x1: x1, y1: y1, h: 28, rise: 11})
	}
	top := 30 * s.kit().scale
	s.walls(x0, y0, x1, y1, 0, top)
	if s.lk.age == ageEmpire {
		hlpCrenels(s, x0, y0, x1, y1, top)
		s.roofDome((x0+x1)/2, (y0+y1)/2, 0.26, top, 10, s.domeMat())
		s.finial((x0+x1)/2, (y0+y1)/2, top+10, 5)
		return top
	}
	hlpCrenels(s, x0, y0, x1, y1, top)
	return top
}

func hlpCrenels(s iso, x0, y0, x1, y1, z float64) {
	wall, cap := s.kit().wall, flatMat(stoneR[3])
	i := 0
	for x := x0; x < x1-0.05; x += 0.3 {
		if !s.lk.chance(90+i, 0.12) {
			s.block(x, y1-0.14, x+0.16, y1, z, z+4+float64(s.lk.pick(95+i, 2)), wall, cap, matStone)
		}
		i++
	}
	for y := y0; y < y1-0.05; y += 0.3 {
		if !s.lk.chance(90+i, 0.12) {
			s.block(x1-0.14, y, x1, y+0.16, z, z+4+float64(s.lk.pick(95+i, 2)), wall, cap, matStone)
		}
		i++
	}
}

func keyDoor(s iso, y1, u, w, top float64) {
	h := math.Round(min(top*0.55, 16))
	frame := ink
	if s.lk.age == ageEmpire {
		frame = goldR[1]
	}
	s.paint(sideL(y1), matTrim, u-0.05, u+w+0.05, 0, h+2, frame)
	s.pen(matTrim, func() {
		s.onLeft(y1, u, u+w, 0, h, func(x, y int) color.NRGBA {
			if (x+y)%4 == 0 {
				return greyR[2]
			}
			return woodR[1]
		})
	})
	c := s.p(u+w/2, y1, h+6)
	x, y := int(c.x)-2, int(c.y)
	s.c.disc(float64(x)-5, float64(y)+0.5, 3.4, goldR[1])
	s.c.disc(float64(x)-5.5, float64(y), 2.4, goldR[2])
	s.c.disc(float64(x)-5, float64(y)+0.5, 1.2, inkSft)
	s.c.rect(x-2, y, 11, 2, goldR[2])
	s.c.rect(x-2, y, 11, 1, goldR[3])
	s.c.rect(x+5, y+2, 2, 3, goldR[1])
	s.c.rect(x+8, y+2, 1, 2, goldR[1])
}
