package art

import (
	"image/color"
	"math"
	"sort"
)

type uv struct{ u, w float64 }

type vec3 struct{ u, w, z float64 }

type hullView struct {
	c              *canvas
	ox, oy         float64
	ux, uy, wx, wy float64
	lift           float64
}

var headingAxes = map[string][4]float64{
	"se": {1, 0, 0, 1},
	"sw": {0, 1, -1, 0},
	"ne": {0, -1, 1, 0},
	"nw": {-1, 0, 0, -1},
}

var shipLight = [3]float64{-0.45, 0.45, 0.77}

const (
	metricGrid = 22.6
	metricZ    = 1.155
)

func newHullView(c *canvas, ax, ay int, dir string, lift float64) hullView {
	a := headingAxes[dir]
	return hullView{c: c, ox: float64(ax), oy: float64(ay), ux: a[0], uy: a[1], wx: a[2], wy: a[3], lift: lift}
}

func (v hullView) grid(u, w float64) (float64, float64) {
	return u*v.ux + w*v.wx, u*v.uy + w*v.wy
}

func (v hullView) p(u, w, z float64) pt {
	gx, gy := v.grid(u, w)
	return pt{v.ox + (gx-gy)*16, v.oy + (gx+gy)*8 - z - v.lift}
}

func (v hullView) pv(q vec3) pt { return v.p(q.u, q.w, q.z) }

func (v hullView) depth(u, w float64) float64 {
	gx, gy := v.grid(u, w)
	return gx + gy
}

func (v hullView) toward(n vec3) float64 {
	gx, gy := v.grid(n.u, n.w)
	return (gx+gy)*0.61 + n.z*0.5
}

func (v hullView) light(n vec3) float64 {
	gx, gy := v.grid(n.u, n.w)
	l := math.Sqrt(gx*gx + gy*gy + n.z*n.z)
	if l == 0 {
		return 0
	}
	return (gx*shipLight[0] + gy*shipLight[1] + n.z*shipLight[2]) / l
}

func toneIndex(b float64) int {
	switch {
	case b > 0.62:
		return 3
	case b > 0.18:
		return 2
	case b > -0.5:
		return 1
	}
	return 0
}

type edge struct {
	a, b   uv
	za, zb float64
}

func (v hullView) wall(top, bot edge, paint func(s, z float64) color.NRGBA) {
	t0, t1 := v.p(top.a.u, top.a.w, top.za), v.p(top.b.u, top.b.w, top.zb)
	b0, b1 := v.p(bot.a.u, bot.a.w, bot.za), v.p(bot.b.u, bot.b.w, bot.zb)
	if math.Abs(t1.x-t0.x) < 1e-9 && math.Abs(b1.x-b0.x) < 1e-9 {
		return
	}
	sh := func(x, y int) color.NRGBA {
		s, r := invertQuad(t0, t1, b0, b1, float64(x)+0.5, float64(y)+0.5)
		zb := bot.za + s*(bot.zb-bot.za)
		zt := top.za + s*(top.zb-top.za)
		return paint(s, zb+r*(zt-zb))
	}
	v.c.shadePoly([]pt{t0, t1, b1, b0}, sh)
}

func clamp01(x float64) float64 { return math.Min(math.Max(x, 0), 1) }

func invertQuad(t0, t1, b0, b1 pt, x, y float64) (float64, float64) {
	s, r := 0.5, 0.5
	for i := 0; i < 3; i++ {
		ty, by := t0.y+s*(t1.y-t0.y), b0.y+s*(b1.y-b0.y)
		if math.Abs(by-ty) > 1e-9 {
			r = (by - y) / (by - ty)
		}
		x0, x1 := b0.x+r*(t0.x-b0.x), b1.x+r*(t1.x-b1.x)
		if math.Abs(x1-x0) > 1e-9 {
			s = (x - x0) / (x1 - x0)
		}
	}
	return clamp01(s), clamp01(r)
}

type skinFn func(idx int, cross bool, rail func(s float64) float64) func(s, z float64) color.NRGBA

type prism struct {
	pts, low  []uv
	base, top func(u float64) float64
	crossDrop float64
	skin      skinFn
}

func centroid(pts []uv) (float64, float64) {
	cu, cw := 0.0, 0.0
	for _, p := range pts {
		cu, cw = cu+p.u, cw+p.w
	}
	n := float64(max(len(pts), 1))
	return cu / n, cw / n
}

func outward(pts []uv, a, b uv) (vec3, bool) {
	cu, cw := centroid(pts)
	n := vec3{u: b.w - a.w, w: -(b.u - a.u)}
	if n.u == 0 && n.w == 0 {
		return n, false
	}
	if n.u*((a.u+b.u)/2-cu)+n.w*((a.w+b.w)/2-cw) < 0 {
		n.u, n.w = -n.u, -n.w
	}
	return n, true
}

func (v hullView) walls(pr prism) {
	low := pr.low
	if low == nil {
		low = pr.pts
	}
	for i := range pr.pts {
		j := (i + 1) % len(pr.pts)
		a, b := pr.pts[i], pr.pts[j]
		n, ok := outward(pr.pts, a, b)
		if !ok || v.toward(n) <= 1e-6 {
			continue
		}
		cross := math.Abs(b.u-a.u) < 1e-6
		drop := 0.0
		if cross {
			drop = pr.crossDrop
		}
		ta, tb := pr.top(a.u), pr.top(b.u)
		rail := func(s float64) float64 { return ta + s*(tb-ta) }
		top := edge{a: a, b: b, za: ta, zb: tb}
		bot := edge{a: low[i], b: low[j], za: pr.base(a.u) - drop, zb: pr.base(b.u) - drop}
		v.wall(top, bot, pr.skin(toneIndex(v.light(n)), cross, rail))
	}
}

func (v hullView) project(pts []uv, z func(u float64) float64, dz float64) []pt {
	out := make([]pt, len(pts))
	for i, q := range pts {
		out[i] = v.p(q.u, q.w, z(q.u)+dz)
	}
	return out
}

func inset(pts []uv, d float64) []uv {
	cu, _ := centroid(pts)
	hu, hw := 1e-9, 1e-9
	for _, p := range pts {
		hu, hw = math.Max(hu, math.Abs(p.u-cu)), math.Max(hw, math.Abs(p.w))
	}
	su, sw := math.Max(0, 1-d/hu), math.Max(0, 1-d/hw)
	out := make([]uv, len(pts))
	for i, p := range pts {
		out[i] = uv{cu + (p.u-cu)*su, p.w * sw}
	}
	return out
}

func (v hullView) deckShader(r ramp) shader {
	m := int(math.Round((v.ux + v.uy) / (v.ux - v.uy)))
	return func(x, y int) color.NRGBA {
		k := (2*y - m*x) >> 1
		if ((k%4)+4)%4 == 0 {
			return mix(r[0], r[1], 0.45)
		}
		return r[1]
	}
}

func (v hullView) deckTop(pts []uv, top func(u float64) float64, bulwark float64, inner color.NRGBA, deck, wood ramp) {
	v.c.shadePoly(v.project(pts, top, 0), flat(inner))
	v.c.shadePoly(v.project(inset(pts, 0.07), top, -bulwark), v.deckShader(deck))
	for i := range pts {
		a, b := pts[i], pts[(i+1)%len(pts)]
		n, ok := outward(pts, a, b)
		if !ok || v.toward(n) > 1e-6 {
			continue
		}
		pa, pb := v.p(a.u, a.w, top(a.u)), v.p(b.u, b.w, top(b.u))
		v.c.line(int(math.Floor(pa.x)), int(math.Floor(pa.y)), int(math.Floor(pb.x)), int(math.Floor(pb.y)), wood[3])
	}
}

type patch struct {
	d   float64
	q   [4]pt
	col color.NRGBA
}

type mesh struct {
	at     func(s, t float64) vec3
	ns, nt int
	cloth  func(s, t float64, idx int) color.NRGBA
}

func metric(a, b vec3) vec3 {
	return vec3{(b.u - a.u) * metricGrid, (b.w - a.w) * metricGrid, (b.z - a.z) * metricZ}
}

func cross3(a, b vec3) vec3 {
	return vec3{a.w*b.z - a.z*b.w, a.z*b.u - a.u*b.z, a.u*b.w - a.w*b.u}
}

func (v hullView) patchOf(m mesh, i, j int) patch {
	s0, s1 := float64(i)/float64(m.ns), float64(i+1)/float64(m.ns)
	t0, t1 := float64(j)/float64(m.nt), float64(j+1)/float64(m.nt)
	a, b, c, d := m.at(s0, t0), m.at(s1, t0), m.at(s1, t1), m.at(s0, t1)
	n := cross3(metric(a, c), metric(b, d))
	if v.toward(n) < 0 {
		n = vec3{-n.u, -n.w, -n.z}
	}
	mid := m.at((s0+s1)/2, (t0+t1)/2)
	gx, gy := v.grid(mid.u, mid.w)
	return patch{
		d:   (gx+gy)*13.8 + mid.z*0.58,
		q:   [4]pt{v.pv(a), v.pv(b), v.pv(c), v.pv(d)},
		col: m.cloth((s0+s1)/2, (t0+t1)/2, toneIndex(v.light(n))),
	}
}

func (v hullView) drawMesh(m mesh) {
	patches := make([]patch, 0, m.ns*m.nt)
	for i := 0; i < m.ns; i++ {
		for j := 0; j < m.nt; j++ {
			patches = append(patches, v.patchOf(m, i, j))
		}
	}
	sort.SliceStable(patches, func(i, j int) bool { return patches[i].d < patches[j].d })
	for i := range patches {
		v.c.shadePoly(patches[i].q[:], flat(patches[i].col))
	}
}

func (v hullView) seg(a, b vec3, col color.NRGBA) {
	pa, pb := v.pv(a), v.pv(b)
	if col.A < 255 {
		v.c.blendLine(int(math.Floor(pa.x)), int(math.Floor(pa.y)), int(math.Floor(pb.x)), int(math.Floor(pb.y)), col)
		return
	}
	v.c.line(int(math.Floor(pa.x)), int(math.Floor(pa.y)), int(math.Floor(pb.x)), int(math.Floor(pb.y)), col)
}

func (v hullView) polyline(m mesh, from, to [2]float64, n int, col color.NRGBA) {
	for k := 0; k < n; k++ {
		f0, f1 := float64(k)/float64(n), float64(k+1)/float64(n)
		a := m.at(from[0]+(to[0]-from[0])*f0, from[1]+(to[1]-from[1])*f0)
		b := m.at(from[0]+(to[0]-from[0])*f1, from[1]+(to[1]-from[1])*f1)
		v.seg(a, b, col)
	}
}

func (c *canvas) blend(x, y int, col color.NRGBA) {
	if !c.in(x, y) || col.A == 0 {
		return
	}
	under := c.img.NRGBAAt(x, y)
	if under.A == 0 {
		c.img.SetNRGBA(x, y, col)
		return
	}
	m := mix(under, col, float64(col.A)/255)
	m.A = max(under.A, col.A)
	c.img.SetNRGBA(x, y, m)
}

func (c *canvas) blendLine(x0, y0, x1, y1 int, col color.NRGBA) {
	n := max(abs(x1-x0), abs(y1-y0))
	for i := 0; i <= n; i++ {
		f := 0.0
		if n > 0 {
			f = float64(i) / float64(n)
		}
		c.blend(x0+int(math.Round(float64(x1-x0)*f)), y0+int(math.Round(float64(y1-y0)*f)), col)
	}
}
