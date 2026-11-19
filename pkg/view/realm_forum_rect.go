package view

func (a Rect) Area() int {
	if a.X1 < a.X0 || a.Y1 < a.Y0 {
		return 0
	}
	return (a.X1 - a.X0 + 1) * (a.Y1 - a.Y0 + 1)
}

func (a Rect) Tiles() []Point {
	out := make([]Point, 0, a.Area())
	for y := a.Y0; y <= a.Y1; y++ {
		for x := a.X0; x <= a.X1; x++ {
			out = append(out, Point{X: x, Y: y})
		}
	}
	return out
}

func (a Rect) grownBy(d Point) Rect {
	switch {
	case d.X > 0:
		return Rect{X0: a.X1 + 1, Y0: a.Y0, X1: a.X1 + 1, Y1: a.Y1}
	case d.X < 0:
		return Rect{X0: a.X0 - 1, Y0: a.Y0, X1: a.X0 - 1, Y1: a.Y1}
	case d.Y > 0:
		return Rect{X0: a.X0, Y0: a.Y1 + 1, X1: a.X1, Y1: a.Y1 + 1}
	}
	return Rect{X0: a.X0, Y0: a.Y0 - 1, X1: a.X1, Y1: a.Y0 - 1}
}

func union(a, b Rect) Rect {
	return Rect{X0: min(a.X0, b.X0), Y0: min(a.Y0, b.Y0), X1: max(a.X1, b.X1), Y1: max(a.Y1, b.Y1)}
}

func openSides(side Point) [3]Point {
	return [3]Point{{X: -side.Y, Y: -side.X}, {X: side.Y, Y: side.X}, side}
}

func (g forumGround) Grow(cur Rect, side Point) Rect {
	next := cur
	for _, d := range openSides(side) {
		if strip := next.grownBy(d); g.fits(strip) {
			next = union(next, strip)
		}
	}
	return next
}

func (g forumGround) grown(base Rect, side Point, rings int) Rect {
	out := base
	for i := 0; i < rings; i++ {
		out = g.Grow(out, side)
	}
	return out
}

func (g forumGround) bestBase(codex Rect, side Point) (Rect, bool) {
	best, found := Rect{}, false
	for depth := ForumDepth; depth >= 1; depth-- {
		for pad := ForumPad; pad >= 0; pad-- {
			r := ForumRect(codex, side, 0)
			r = shrink(r, codex, side, ForumPad-pad, ForumDepth-depth)
			if r.Area() > best.Area() && g.fits(r) {
				best, found = r, true
			}
		}
	}
	return best, found
}

func shrink(r, codex Rect, side Point, pad, depth int) Rect {
	if side.X != 0 {
		r.Y0, r.Y1 = r.Y0+pad, r.Y1-pad
	} else {
		r.X0, r.X1 = r.X0+pad, r.X1-pad
	}
	switch {
	case side.X > 0:
		r.X1 -= depth
	case side.X < 0:
		r.X0 += depth
	case side.Y > 0:
		r.Y1 -= depth
	default:
		r.Y0 += depth
	}
	return r
}

func (s Site) ForumOption(codex Rect, side Point) (forumChoice, bool) {
	g, ok := s.ForumGroundFor(codex, ForumRect(codex, side, ForumPlanRing))
	if !ok {
		return forumChoice{}, false
	}
	base, whole := ForumRect(codex, side, 0), true
	if !g.fits(base) {
		base, _ = g.bestBase(codex, side)
		whole = false
	}
	away := side.X*(codex.X0+codex.X1-2*g.centre.X) + side.Y*(codex.Y0+codex.Y1-2*g.centre.Y)
	c := forumChoice{Side: side, Base: base, Laid: base.Area() > 0, whole: whole && away >= 0, Away: away}
	if base.Area() > 0 {
		c.plan = g.grown(base, side, ForumPlanRing)
	}
	c.Score = c.plan.Area() + forumAway*away + ForumFront*btoi(side.X > 0 || side.Y > 0)
	return c, true
}

func (s Site) ForumPlacement(codex Rect) (forumChoice, bool) {
	best, found := forumChoice{}, false
	for _, side := range ForumSides {
		c, ok := s.ForumOption(codex, side)
		if ok && c.whole && (!found || c.Score > best.Score) {
			best, found = c, true
		}
	}
	return best, found
}

func (s Site) ForumFallback(codex Rect) forumChoice {
	best, found := forumChoice{Side: ForumSides[0]}, false
	for _, side := range ForumSides {
		c, ok := s.ForumOption(codex, side)
		if ok && c.Laid && (!found || c.BetterFallback(best)) {
			best, found = c, true
		}
	}
	return best
}

func (c forumChoice) BetterFallback(than forumChoice) bool {
	if (c.Away >= 0) != (than.Away >= 0) {
		return c.Away >= 0
	}
	return c.Score > than.Score
}

func (s Site) PreviewForum(x, y, w, h int) (ForumPreview, bool) {
	if s.Forum != nil {
		return ForumPreview{}, false
	}
	c, ok := s.ForumPlacement(Rect{X0: x, Y0: y, X1: x + w - 1, Y1: y + h - 1})
	if !ok {
		return ForumPreview{}, false
	}
	return ForumPreview{Side: c.Side, Base: c.Base.Tiles(), Plan: c.plan.Tiles()}, true
}
