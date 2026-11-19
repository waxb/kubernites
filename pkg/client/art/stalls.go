package art

import (
	"image/color"
	"math"
)

var (
	StallKinds = []string{"stall-parcel", "stall-ledger", "stall-crate", "stall-dispatch", "stall-seal", "stall-guildhall", "stall-archive", "stall-herald", "stall-plaque"}
	StallFaces = []string{"sw", "se", "ne", "nw"}
)

const (
	stallHeight     = 32
	tallStallHeight = 38
)

var stallShade = color.NRGBA{20, 16, 24, 64}

var stallHeights = map[string]int{"stall-archive": tallStallHeight, "stall-herald": tallStallHeight}

var stallPainters = map[string]func(t stall){}

func StallArtKind(kind, face string) string { return kind + "-" + face }

func isStallKind(kind string) bool {
	for _, k := range StallKinds {
		if k == kind {
			return true
		}
	}
	return false
}

func StallHeight(kind string) int {
	if h, ok := stallHeights[kind]; ok {
		return h
	}
	return stallHeight
}

func StallFootprint(face string) (int, int) {
	if face == "se" || face == "nw" {
		return 1, 2
	}
	return 2, 1
}

func stallSize(kind, face string) size {
	w, h := StallFootprint(face)
	cw := (w+h)*TileWidth/2 + 4
	ch := (w+h)*TileHeight/2 + StallHeight(kind) + 2 + buildingFoot
	return size{w: cw, h: ch, ax: w*TileWidth/2 + 2, ay: ch - buildingFoot}
}

func isoFrameWH(c *canvas, w, h, ax, ay int) iso {
	return iso{c: c, ox: float64(ax - (w-h)*TileWidth/2), oy: float64(ay - (w+h)*TileHeight/2)}
}

type stall struct {
	iso
	kind, face string
}

func (t stall) g(u, v float64) (float64, float64) {
	switch t.face {
	case "se":
		return v, 2 - u
	case "ne":
		return 2 - u, 1 - v
	case "nw":
		return 1 - v, u
	}
	return u, v
}

func (t stall) at(u, v, z float64) pt {
	gx, gy := t.g(u, v)
	return t.p(gx, gy, z)
}

func (t stall) grid(u0, v0, u1, v1 float64) (float64, float64, float64, float64) {
	ax, ay := t.g(u0, v0)
	bx, by := t.g(u1, v1)
	return math.Min(ax, bx), math.Min(ay, by), math.Max(ax, bx), math.Max(ay, by)
}

func (t stall) slab(u0, v0, u1, v1, z0, z1 float64, wall, top mat, tag uint8) {
	x0, y0, x1, y1 := t.grid(u0, v0, u1, v1)
	t.block(x0, y0, x1, y1, z0, z1, wall, top, tag)
}

func (t stall) flatSlab(u0, v0, u1, v1, z0, z1 float64, r ramp) {
	x0, y0, x1, y1 := t.grid(u0, v0, u1, v1)
	t.pen(matTrim, func() { t.box(x0, y0, x1, y1, z0, z1, faces{top: flat(r[3]), left: flat(r[2]), right: flat(r[1])}) })
}

func (t stall) facesCamera() bool { return t.face == "sw" || t.face == "se" }

func (t stall) dir(du, dv float64) (float64, float64) {
	x0, y0 := t.g(0, 0)
	x1, y1 := t.g(du, dv)
	return x1 - x0, y1 - y0
}

type plane struct {
	du, dv, u, v float64
}

func frontAt(v float64) plane { return plane{dv: 1, v: v} }
func backAt(v float64) plane  { return plane{dv: -1, v: v} }
func endAt(u, du float64) plane {
	return plane{du: du, u: u}
}

func (t stall) span(p plane, a0, a1 float64) (float64, float64, float64, float64) {
	if p.du != 0 {
		ax, ay := t.g(p.u, a0)
		bx, by := t.g(p.u, a1)
		return ax, ay, bx, by
	}
	ax, ay := t.g(a0, p.v)
	bx, by := t.g(a1, p.v)
	return ax, ay, bx, by
}

func (t stall) sees(p plane) bool {
	nx, ny := t.dir(p.du, p.dv)
	return nx > 0.5 || ny > 0.5
}

func (t stall) panel(p plane, a0, a1, z0, z1 float64, sh shader) {
	nx, ny := t.dir(p.du, p.dv)
	ax, ay, bx, by := t.span(p, a0, a1)
	switch {
	case ny > 0.5:
		t.onLeft(ay, math.Min(ax, bx), math.Max(ax, bx), z0, z1, sh)
	case nx > 0.5:
		t.onRight(ax, math.Min(ay, by), math.Max(ay, by), z0, z1, sh)
	}
}

func (t stall) post(u, v, z0, z1 float64) {
	gx, gy := t.g(u, v)
	t.iso.post(gx, gy, z0, z1)
}

func (t stall) pole(u, v, z0, z1 float64, col ramp) {
	a, b := t.at(u, v, z0), t.at(u, v, z1)
	t.pen(matTrim, func() {
		t.c.line(int(a.x), int(a.y), int(b.x), int(b.y), col[1])
		t.c.line(int(a.x)+1, int(a.y), int(b.x)+1, int(b.y), col[2])
	})
}

func (t stall) shadowFloor() {
	g := t.groundLayer()
	x0, y0, x1, y1 := t.grid(0.08, 0.08, 1.92, 0.95)
	a, b, c, d := t.p(x0, y0, 0), t.p(x1, y0, 0), t.p(x1, y1, 0), t.p(x0, y1, 0)
	g.shadePoly([]pt{a, b, c, d}, flat(stallShade))
}
