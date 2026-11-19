package art

import (
	"image/color"
	"sync"
)

type stallKey struct {
	kind, face   string
	age, variant int
}

type stallMemo struct {
	mu    sync.Mutex
	shots map[stallKey]*agedShot
}

func newStallMemo() *stallMemo { return &stallMemo{shots: map[stallKey]*agedShot{}} }

func stallLayers(kind, face string, l look) (*canvas, *canvas) {
	sz := stallSize(kind, face)
	w, h := StallFootprint(face)
	gnd, body := newCanvas(sz.w, sz.h), newCanvas(sz.w, sz.h)
	body.enableTags()
	fr := isoFrameWH(body, w, h, sz.ax, sz.ay).with(l)
	fr.gnd = gnd
	drawStall(stall{iso: fr, kind: kind, face: face})
	return gnd, body
}

func drawStall(t stall) {
	if paint, ok := stallPainters[t.kind]; ok {
		paint(t)
		return
	}
	paintStallBody(t)
}

func paintStallBody(t stall) {
	p := t.plan()
	t.shadowFloor()
	t.footing()
	if t.facesCamera() {
		t.backWall(p)
		t.counter(p)
		t.goods(p)
		t.frontPosts(p)
		t.roof(p)
	} else {
		t.frontPosts(p)
		t.counter(p)
		t.goods(p)
		t.roof(p)
		t.backWall(p)
	}
}

func (m *stallMemo) shot(kind, face string, l look) *agedShot {
	m.mu.Lock()
	k := stallKey{kind, face, l.age, l.variant}
	sh, ok := m.shots[k]
	if !ok {
		sh = &agedShot{}
		m.shots[k] = sh
	}
	m.mu.Unlock()
	sh.once.Do(func() {
		sz := stallSize(kind, face)
		gnd, body := stallLayers(kind, face, l)
		sh.gnd, sh.full = newCanvas(sz.w, sz.h), newCanvas(sz.w, sz.h)
		sh.gnd.paste(gnd, 0, 0, false)
		finishRough(sh.full, gnd, body, l)
	})
	return sh
}

func stallReady(kind, face string, age, variant int) *canvas {
	return newStallMemo().shot(kind, face, newLook(kind, age, variant)).full
}

func addStalls(b *atlasBuilder) {
	m := newStallMemo()
	for _, kind := range StallKinds {
		for _, face := range StallFaces {
			addStallFace(b, m, kind, face)
		}
	}
}

func addStallFace(b *atlasBuilder, m *stallMemo, kind, face string) {
	sz, art := stallSize(kind, face), StallArtKind(kind, face)
	add := func(name string, paint func(c *canvas)) { b.add(name, sz.w, sz.h, sz.ax, sz.ay, paint) }
	add(buildingName(art, "construction-0"), func(c *canvas) { m.paintRaising(c, kind, face, newLook(kind, 0, 0), 0) })
	for age := 0; age < AgeTiers; age++ {
		for v := 0; v < BuildingVariants; v++ {
			l := newLook(kind, age, v)
			add(BuildingLookName(art, "ready", age, v), func(c *canvas) { c.paste(m.shot(kind, face, l).full, 0, 0, false) })
		}
		l := newLook(kind, age, 0)
		add(BuildingLookName(art, "dark", age, 0), func(c *canvas) {
			c.paste(m.shot(kind, face, l).full, 0, 0, false)
			c.mapPixels(dusk)
		})
		for stage := 1; stage < 3; stage++ {
			st := stage
			add(BuildingLookName(art, stateName(st), age, 0), func(c *canvas) { m.paintRaising(c, kind, face, l, st) })
		}
	}
}

func (m *stallMemo) paintRaising(dst *canvas, kind, face string, l look, stage int) {
	sz := stallSize(kind, face)
	w, h := StallFootprint(face)
	sh := m.shot(kind, face, l)
	dst.paste(sh.gnd, 0, 0, false)
	t := stall{iso: isoFrameWH(dst, w, h, sz.ax, sz.ay).with(l), kind: kind, face: face}
	if stage > 0 {
		keep := []float64{0, 0.45, 0.8}[stage]
		cut := sz.ay - int(float64((w+h)*TileHeight/2+StallHeight(kind))*keep)
		for y := cut; y < sz.h; y++ {
			for x := 0; x < sz.w; x++ {
				if col := sh.full.at(x, y); col.A > 0 {
					dst.dot(x, y, col)
				}
			}
		}
		t.scaffold(min(float64(sz.ay-cut), float64(StallHeight(kind))))
	}
	t.stakes()
	if stage == 0 {
		t.crate(t.nearGrid(1.0, 0.6))
		t.c.outline(ink)
	}
}

func (t stall) nearGrid(u, v float64) (float64, float64, float64) {
	gx, gy := t.g(u, v)
	return gx - 0.12, gy - 0.12, 0.26
}

func (t stall) stakes() {
	for _, c := range [][2]float64{{0.06, 0.06}, {1.94, 0.06}, {0.06, 0.94}, {1.94, 0.94}} {
		a := t.at(c[0], c[1], 0)
		t.c.rect(int(a.x), int(a.y)-5, 1, 5, woodR[2])
		t.c.dot(int(a.x), int(a.y)-5, color.NRGBA{230, 70, 60, 255})
	}
}

func (t stall) scaffold(height float64) {
	top := height * 0.9
	for _, c := range [][2]float64{{0.1, 0.9}, {1.0, 0.95}, {1.9, 0.9}, {1.9, 0.1}} {
		a, b := t.at(c[0], c[1], 0), t.at(c[0], c[1], top)
		t.c.line(int(a.x), int(a.y), int(b.x), int(b.y), woodR[2])
	}
	for z := 7.0; z < top; z += 10 {
		a, b := t.at(0.1, 0.9, z), t.at(1.9, 0.9, z)
		t.c.line(int(a.x), int(a.y), int(b.x), int(b.y), woodR[1])
	}
}
