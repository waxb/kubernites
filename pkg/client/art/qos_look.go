package art

import (
	"fmt"
	"image/color"
	"sync"
)

var hutVariantStates = map[string]bool{"ready": true, "guttering": true, "ghost": true}

func hutVariants(state string) int {
	if hutVariantStates[state] {
		return BuildingVariants
	}
	return 1
}

func (h hut) look() look {
	seed := int(NameSeed(fmt.Sprintf("hut/%s/%d", h.qos, h.tier))%1000003)*7 + h.variant*7919
	return look{age: clampAge(h.age), variant: h.variant, seed: seed, id: tierRamp(h.tier)}
}

type hutPaintKey struct {
	h       hut
	burning bool
}

type hutShot struct {
	once sync.Once
	c    *canvas
}

type hutMemo struct {
	mu    sync.Mutex
	shots map[hutPaintKey]*hutShot
}

func newHutMemo() *hutMemo { return &hutMemo{shots: map[hutPaintKey]*hutShot{}} }

func (m *hutMemo) house(s size, h hut, burning bool) *canvas {
	k := hutPaintKey{h, burning}
	m.mu.Lock()
	sh, ok := m.shots[k]
	if !ok {
		sh = &hutShot{}
		m.shots[k] = sh
	}
	m.mu.Unlock()
	sh.once.Do(func() { sh.c = qosHouse(s, h, burning) })
	return sh.c
}

func hutLayers(s size, l look, paint func(fr iso)) (*canvas, *canvas) {
	gnd, body := newCanvas(s.w, s.h), newCanvas(s.w, s.h)
	body.enableTags()
	fr := hearthFrame(body, s).with(l)
	fr.gnd = gnd
	paint(fr)
	return gnd, body
}

func hutRough(s size, l look, paint func(fr iso)) *canvas {
	c := newCanvas(s.w, s.h)
	gnd, body := hutLayers(s, l, paint)
	finishRough(c, gnd, body, l)
	hutClip(c, s)
	return c
}

func hutOnTile(x, y, ax, ay, margin int) bool {
	cy := ay - TileHeight/2
	return y <= ay+1 && abs(x-ax)*TileHeight+abs(y-cy)*TileWidth <= (TileWidth/2+margin)*TileHeight
}

func hutClip(c *canvas, s size) {
	for y := s.ay - TileHeight/2; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if !hutOnTile(x, y, s.ax, s.ay, 1) {
				c.erase(x, y)
			}
		}
	}
}

func (s iso) hutGround() iso {
	g := s
	g.c, g.gnd = s.groundLayer(), nil
	return g
}

func hutRoofMat(age int, r ramp, seed int) mat {
	switch age {
	case ageDark:
		return thatchMat(r, seed)
	case ageFeudal:
		return shingleMat(r, seed)
	case ageCastle:
		return slateMat(r, seed)
	}
	return tileMat(r, seed)
}

func (s iso) hutFrame(f side, u0, u1, z0, z1 float64) {
	col := woodR[0]
	switch s.lk.age {
	case ageCastle:
		col = stoneR[3]
	case ageEmpire:
		col = goldR[1]
	}
	s.paint(f, matTrim, u0, u1, z0, z1, col)
}

func (s iso) hutWindow(f side, u, z float64, lit bool) {
	switch s.lk.age {
	case ageDark:
		s.paint(f, matTrim, u-0.02, u+0.13, z-1, z+4, woodR[0])
		s.pen(matGlass, func() { s.on(f, u+0.01, u+0.1, z, z+3, emberGlass(lit)) })
		s.paint(f, matTrim, u+0.13, u+0.17, z-1, z+4, woodR[2])
		return
	case ageFeudal:
		s.paint(f, matTrim, u-0.06, u-0.02, z-1, z+5, s.lk.id[1])
		s.paint(f, matTrim, u+0.14, u+0.18, z-1, z+5, s.lk.id[2])
	}
	s.hutFrame(f, u-0.02, u+0.14, z-1, z+5)
	if s.lk.age >= ageCastle {
		s.hutFrame(f, u+0.01, u+0.11, z+5, z+6)
	}
	s.pen(matGlass, func() { s.on(f, u+0.02, u+0.12, z, z+4, paneGlass(lit)) })
}

func (s iso) hutFlowerBox(f side, u, z float64) {
	s.paint(f, matNone, u-0.03, u+0.17, z-2, z, woodR[1])
	for i := 0; i < 4; i++ {
		col := bloomColours[hash(i, s.lk.seed, 61)%uint32(len(bloomColours))]
		s.paint(f, matNone, u+float64(i)*0.04, u+float64(i)*0.04+0.03, z, z+1, col)
	}
}

func (s iso) hutLip(y1, x0, x1, z float64, r ramp) {
	s.pen(matRoof, func() {
		s.quad(s.p(x0, y1, z), s.p(x1, y1, z), s.p(x1, y1, z-2), s.p(x0, y1, z-2), flat(r[0]))
	})
}

func (s iso) hutWashLine(a, b [2]float64) {
	p, q := s.p(a[0], a[1], 9), s.p(b[0], b[1], 9)
	s.c.line(int(p.x), int(p.y), int(p.x), int(p.y)+9, woodR[0])
	s.c.line(int(q.x), int(q.y), int(q.x), int(q.y)+9, woodR[0])
	s.c.line(int(p.x), int(p.y), int(q.x), int(q.y), plasterR[1])
	cloths := []color.NRGBA{white, s.lk.id[2], plasterR[3], blueR[2]}
	for i := 1; i < 4; i++ {
		t := float64(i) / 4
		x, y := int(p.x+(q.x-p.x)*t), int(p.y+(q.y-p.y)*t)
		s.c.rect(x, y+1, 2, 3, cloths[(i+s.lk.variant)%len(cloths)])
	}
}

func (s iso) hutWoodpile(gx, gy float64) {
	for i, o := range [][3]float64{{0, 0, 1}, {0.08, 0.02, 1}, {0.04, 0.01, 3}} {
		p := s.p(gx+o[0], gy+o[1], o[2])
		s.c.disc(p.x, p.y, 1.3, woodR[1+i%2])
		s.c.dot(int(p.x), int(p.y), woodR[3])
	}
}

func (s iso) hutBucket(gx, gy float64) {
	p := s.p(gx, gy, 0)
	s.c.rect(int(p.x)-1, int(p.y)-3, 3, 3, greyR[1])
	s.c.rect(int(p.x)-1, int(p.y)-3, 3, 1, greyR[3])
	s.c.dot(int(p.x), int(p.y)-3, waterR[2])
}

func (s iso) hutProp(k int, gx, gy float64) {
	switch k {
	case 0:
		s.barrel(gx, gy)
	case 1:
		s.hutWoodpile(gx, gy)
	case 2:
		sack(s.c, s.p(gx, gy, 0), rgb(176, 150, 108))
	case 3:
		s.hutBucket(gx, gy)
	}
}
