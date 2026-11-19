package view

const GateHavenID = GatePortID

const QuayHalf = 2

const QuayDepth = 3

const YardSize = 2

func (p Point) Step(d Point, n int) Point {
	return Point{X: p.X + d.X*n, Y: p.Y + d.Y*n}
}

func (h Haven) Along() Point {
	return Point{X: Abs(h.Out.Y), Y: Abs(h.Out.X)}
}

func (h Haven) PierHead() Point {
	return h.Quay.Step(h.Out, h.Pier)
}

func (h Haven) Berths() [3]Point {
	head := h.PierHead()
	return [3]Point{head.Step(h.Out, 1), head.Step(h.Along(), -1), head.Step(h.Along(), 1)}
}

func (h Haven) QuayArea() (Point, Point) {
	q := h.QuayRect()
	return Point{X: q.X0, Y: q.Y0}, Point{X: q.X1, Y: q.Y1}
}

func (h Haven) Yards() [2]Point {
	yards := h.yardRects()
	return [2]Point{{X: yards[0].X0, Y: yards[0].Y0}, {X: yards[1].X0, Y: yards[1].Y0}}
}

func (h Haven) yardRects() [2]Rect {
	return [2]Rect{h.yardRect(-1), h.yardRect(1)}
}

func (h Haven) yardRect(side int) Rect {
	near := h.Quay.Step(h.Along(), side).Step(h.Out, -QuayDepth)
	return spanRect(near, near.Step(h.Along(), side*(YardSize-1)).Step(h.Out, 1-YardSize))
}

func (h Haven) QuayRect() Rect {
	return spanRect(h.Quay.Step(h.Along(), -QuayHalf), h.Quay.Step(h.Along(), QuayHalf).Step(h.Out, 1-QuayDepth))
}

func (h Haven) pierRect() Rect {
	return spanRect(h.Quay.Step(h.Out, 1), h.PierHead())
}

func spanRect(a, b Point) Rect {
	return Rect{X0: min(a.X, b.X), Y0: min(a.Y, b.Y), X1: max(a.X, b.X), Y1: max(a.Y, b.Y)}
}

func (s Site) HavenOver(want Rect) (Haven, bool) {
	for _, h := range s.Havens {
		yards := h.yardRects()
		if h.QuayRect().Gap(want) == 0 || h.Pier > 0 && h.pierRect().Gap(want) == 0 || yards[0].Gap(want) == 0 || yards[1].Gap(want) == 0 {
			return h, true
		}
	}
	return Haven{}, false
}

func (s Site) OnHarborGround(x, y int) bool {
	for _, h := range s.Havens {
		yards := h.yardRects()
		if h.QuayRect().Contains(x, y) || yards[0].Contains(x, y) || yards[1].Contains(x, y) {
			return true
		}
	}
	return false
}
