package art

func stoneTop() shader { return flat(stoneR[3]) }

func timberUpper(s iso, x0, y0, x1, y1, z0, z1 float64) {
	s.box(x0, y0, x1, y1, z0, z1, faces{top: flat(plasterR[3]), left: flat(plasterR[2]), right: flat(plasterR[1])})
	for u := x0 + 0.02; u < x1; u += 0.4 {
		a, b := s.p(u, y1, z0), s.p(u, y1, z1)
		s.c.line(int(a.x), int(a.y), int(b.x), int(b.y), woodR[0])
	}
	for v := y0 + 0.02; v < y1; v += 0.4 {
		a, b := s.p(x1, v, z0), s.p(x1, v, z1)
		s.c.line(int(a.x), int(a.y), int(b.x), int(b.y), woodR[0])
	}
	a, b := s.p(x0, y1, z0), s.p(x1, y1, z0)
	s.c.line(int(a.x), int(a.y), int(b.x), int(b.y), woodR[1])
	c, d := s.p(x1, y0, z0), s.p(x1, y1, z0)
	s.c.line(int(c.x), int(c.y), int(d.x), int(d.y), woodR[0])
}
