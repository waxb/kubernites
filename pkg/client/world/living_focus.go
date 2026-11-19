package world

import "github.com/waxb/kubernites/pkg/view"

func (r *Renderer) CairnCentre() Vec {
	sum, n := Vec{}, 0.0
	for _, c := range r.livingOf().cairns {
		sum, n = Vec{sum.X + c.at.X, sum.Y + c.at.Y}, n+1
	}
	if n == 0 {
		return Vec{}
	}
	return Vec{sum.X / n, sum.Y / n}
}

func (r *Renderer) LaundryCentre(sc Scene) (Vec, bool) {
	for _, line := range r.clothLines(sc) {
		return Vec{(line.a.X + line.b.X) / 2, (line.a.Y + line.b.Y) / 2}, true
	}
	return Vec{}, false
}

func BoatAnchors(site view.Site) []Vec {
	out := []Vec{}
	for _, b := range PlanBoats(site) {
		out = append(out, b.anchor)
	}
	return out
}
