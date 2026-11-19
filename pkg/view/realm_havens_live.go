package view

func (s Site) WithGateHaven() Site {
	if len(s.Tiles) == 0 || len(s.Tiles) != s.Width*s.Height {
		return s
	}
	p := newHavenPlanner(s, 1)
	h, ok := p.gateHaven()
	if !ok {
		return s
	}
	h.ID, h.Name, h.Open, h.Share = GateHavenID, gateHavenName, true, 1
	return p.settled([]Haven{h})
}

func (s Site) HarborGround(x, y, margin int) bool {
	for _, h := range s.Havens {
		yards := h.yardRects()
		for _, r := range []Rect{h.QuayRect(), yards[0], yards[1]} {
			if x >= r.X0-margin && x <= r.X1+margin && y >= r.Y0-margin && y <= r.Y1+margin {
				return true
			}
		}
	}
	return false
}
