package world

const PickPad = 2

type Shape struct {
	Name string
	X, Y float64
	Zoom float64
	Flip bool
}

var pickClasses = []func(string) bool{
	func(k string) bool { return k == HitGolden },
	func(k string) bool { return k == HitPile },
	func(string) bool { return true },
}

func pickClass(kind string) int {
	for i, in := range pickClasses {
		if in(kind) {
			return i
		}
	}
	return len(pickClasses) - 1
}

func (r *Renderer) Pick(x, y int) (Hit, bool) {
	all := r.PickAll(x, y)
	if len(all) == 0 {
		return Hit{}, false
	}
	return all[0], true
}

func (r *Renderer) PickAll(x, y int) []Hit {
	out := r.gather(x, y, 0)
	for _, h := range r.gather(x, y, PickPad) {
		if !picked(out, h) {
			out = append(out, h)
		}
	}
	return out
}

func (r *Renderer) gather(x, y int, pad float64) []Hit {
	classes := make([][]Hit, len(pickClasses))
	for i := len(r.HitList) - 1; i >= 0; i-- {
		h := r.HitList[i]
		c := pickClass(h.Kind)
		if picked(classes[c], h) || !r.Covers(h, x, y, pad) {
			continue
		}
		classes[c] = append(classes[c], h)
	}
	out := []Hit{}
	for _, c := range classes {
		out = append(out, c...)
	}
	return out
}

func picked(list []Hit, h Hit) bool {
	for _, o := range list {
		if o.Kind == h.Kind && o.ID == h.ID && o.Serial == h.Serial {
			return true
		}
	}
	return false
}

func (r *Renderer) Covers(h Hit, x, y int, pad float64) bool {
	if len(h.Shapes) == 0 {
		return h.Rect.Contains(x, y)
	}
	pad += h.Grow
	if !h.Rect.Inset(-int(pad)).Contains(x, y) {
		return false
	}
	for _, s := range h.Shapes {
		if _, masked := r.Lib.Mask(s.Name); !masked {
			return h.Rect.Contains(x, y)
		}
		if r.Lib.Covers(s.Name, s.X, s.Y, s.Zoom, s.Flip, x, y, pad) {
			return true
		}
	}
	return false
}

func (r *Renderer) ShownName(names []string) string {
	for _, n := range names {
		if r.Lib.Has(n) {
			return n
		}
	}
	if len(names) > 0 {
		return names[0]
	}
	return ""
}

func ShapeAt(name string, sx, sy, zoom float64) []Shape {
	return []Shape{{Name: name, X: sx, Y: sy, Zoom: zoom}}
}

func (r *Renderer) HitOf(kind, id string) (Hit, bool) {
	for i := len(r.HitList) - 1; i >= 0; i-- {
		if r.HitList[i].Kind == kind && r.HitList[i].ID == id {
			return r.HitList[i], true
		}
	}
	return Hit{}, false
}

func (r *Renderer) Hits() []Hit {
	return append([]Hit(nil), r.HitList...)
}
