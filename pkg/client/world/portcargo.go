package world

const seeThroughAlpha = .45

func (r *Renderer) hidesCargo(piece Hit, depth float64, cargo []Drawable) bool {
	for _, c := range cargo {
		if c.Hit == nil || c.Hit.Kind != HitPile || c.Depth >= depth {
			continue
		}
		x, y := c.Hit.Rect.X+c.Hit.Rect.W/2, c.Hit.Rect.Y+c.Hit.Rect.H/2
		if r.Covers(piece, x, y, 0) {
			return true
		}
	}
	return false
}
