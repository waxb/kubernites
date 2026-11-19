package art

const (
	propNone = iota
	propBarrel
	propCrate
	propBarrels
	propSacks
	propCart
	propLogs
	propHay
	propTrough
	propLantern
	propKinds
)

func (s iso) cart(gx, gy float64) {
	bed := faces{top: flat(woodR[2]), left: s.plankLeft(woodR, gy+0.3), right: s.plankRight(woodR, gx+0.45)}
	s.box(gx, gy, gx+0.45, gy+0.3, 5, 9, bed)
	wheelAt(s, gx+0.22, gy+0.32)
	a := s.p(gx+0.45, gy+0.15, 6)
	s.c.line(int(a.x), int(a.y), int(a.x)+7, int(a.y)+4, woodR[1])
	s.c.dot(int(a.x)+2, int(a.y)-4, strawR[2])
	s.c.dot(int(a.x)-2, int(a.y)-4, strawR[3])
}

func (s iso) hay(gx, gy float64) {
	p := s.p(gx, gy, 0)
	s.c.ellipse(p.x, p.y-3, 4, 3, strawR[1])
	s.c.ellipse(p.x-1, p.y-4, 3, 2, strawR[2])
	s.c.dot(int(p.x)-2, int(p.y)-6, strawR[3])
	s.c.dot(int(p.x)+2, int(p.y)-3, strawR[0])
}

func (s iso) trough(gx, gy float64) {
	s.box(gx, gy, gx+0.4, gy+0.16, 0, 4, faces{top: flat(waterR[2]), left: flat(woodR[2]), right: flat(woodR[1])})
	p := s.p(gx+0.12, gy+0.08, 4)
	s.c.dot(int(p.x), int(p.y), waterR[3])
}

func (s iso) lantern(gx, gy float64) {
	a, b := s.p(gx, gy, 0), s.p(gx, gy, 14)
	s.edge(a, b, woodR[0])
	s.c.rect(int(b.x)-1, int(b.y)-3, 3, 3, lampLit)
	s.c.dot(int(b.x), int(b.y)-4, ink)
}

func (s iso) sacks(gx, gy float64) {
	sack(s.c, s.p(gx, gy, 0), rgb(176, 150, 108))
	sack(s.c, s.p(gx+0.12, gy+0.08, 0), rgb(150, 124, 88))
}

func (s iso) prop(kind int, gx, gy float64) {
	switch kind {
	case propBarrel:
		s.barrel(gx, gy)
	case propCrate:
		s.crate(gx, gy, 0.26)
	case propBarrels:
		s.barrel(gx, gy)
		s.barrel(gx+0.16, gy+0.1)
	case propSacks:
		s.sacks(gx, gy)
	case propCart:
		s.cart(gx-0.2, gy-0.15)
	case propLogs:
		logPile(s, gx, gy)
	case propHay:
		s.hay(gx, gy)
	case propTrough:
		s.trough(gx-0.2, gy)
	case propLantern:
		s.lantern(gx, gy)
	}
}

func (s iso) props(spots [][2]float64, k int) {
	for i, sp := range spots {
		s.prop(s.lk.pick(k+i*7, propKinds), sp[0], sp[1])
	}
}
