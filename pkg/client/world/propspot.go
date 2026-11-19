package world

import "github.com/waxb/kubernites/pkg/view"

func groundClear(site view.Site, h *Hamlet, at Vec) bool {
	t := TileOf(at)
	if t.X < 0 || t.Y < 0 || t.X >= site.Width || t.Y >= site.Height || WetAt(site, at) {
		return false
	}
	if h != nil && h.occAt(t) == occHut {
		return false
	}
	for _, s := range site.Structures {
		if Covers(s, t.X, t.Y) {
			return false
		}
	}
	return true
}

func firstClear(site view.Site, h *Hamlet, spots []Vec) (Vec, bool) {
	for _, at := range spots {
		if groundClear(site, h, at) {
			return at, true
		}
	}
	return Vec{}, false
}

func propDrawable(d Drawable, at Vec, sx, sy float64) Drawable {
	d.stand = stand(at, sx, sy)
	return d
}
