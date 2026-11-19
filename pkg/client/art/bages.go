package art

import "sync"

var (
	agedPainters = map[string]func(s iso){}
	agedDark     = map[string]func(s iso){}
)

func agedKind(kind string, paint func(s iso)) { agedPainters[kind] = paint }

func agedDarkKind(kind string, paint func(s iso)) { agedDark[kind] = paint }

func isAged(kind string) bool {
	_, ok := agedPainters[kind]
	return ok
}

func agedLayers(kind string, sz size, l look, paint func(s iso)) (*canvas, *canvas) {
	gnd, body := newCanvas(sz.w, sz.h), newCanvas(sz.w, sz.h)
	body.enableTags()
	fr := isoFrame(body, Footprints[kind], sz.ax, sz.ay).with(l)
	fr.gnd = gnd
	paint(fr)
	return gnd, body
}

func paintAged(dst *canvas, kind string, sz size, l look, paint func(s iso)) {
	gnd, body := agedLayers(kind, sz, l, paint)
	finishRough(dst, gnd, body, l)
}

func agedReady(kind string, sz size, l look) *canvas {
	c := newCanvas(sz.w, sz.h)
	paintAged(c, kind, sz, l, agedPainters[kind])
	return c
}

type agedKey struct {
	kind         string
	age, variant int
}

type agedShot struct {
	once      sync.Once
	full, gnd *canvas
}

type agedMemo struct {
	mu    sync.Mutex
	shots map[agedKey]*agedShot
}

func newAgedMemo() *agedMemo { return &agedMemo{shots: map[agedKey]*agedShot{}} }

func (m *agedMemo) shot(kind string, sz size, l look) *agedShot {
	m.mu.Lock()
	k := agedKey{kind, l.age, l.variant}
	sh, ok := m.shots[k]
	if !ok {
		sh = &agedShot{}
		m.shots[k] = sh
	}
	m.mu.Unlock()
	sh.once.Do(func() {
		gnd, body := agedLayers(kind, sz, l, agedPainters[kind])
		sh.gnd, sh.full = newCanvas(sz.w, sz.h), newCanvas(sz.w, sz.h)
		sh.gnd.paste(gnd, 0, 0, false)
		finishRough(sh.full, gnd, body, l)
	})
	return sh
}

func (m *agedMemo) paintDark(dst *canvas, kind string, sz size, l look) {
	if paint, ok := agedDark[kind]; ok {
		paintAged(dst, kind, sz, l, paint)
	} else {
		dst.paste(m.shot(kind, sz, l).full, 0, 0, false)
	}
	dst.mapPixels(dusk)
}

func (m *agedMemo) paintConstruction(dst *canvas, kind string, sz size, l look, stage int) {
	n := Footprints[kind]
	sh := m.shot(kind, sz, l)
	dst.paste(sh.gnd, 0, 0, false)
	fr := isoFrame(dst, n, sz.ax, sz.ay).with(l)
	if stage > 0 {
		keep := []float64{0, 0.4, 0.75}[stage]
		cut := sz.ay - int(float64(n*TileHeight+buildingHeight(kind))*keep)
		for y := cut; y < sz.h; y++ {
			for x := 0; x < sz.w; x++ {
				if col := sh.full.at(x, y); col.A > 0 {
					dst.dot(x, y, col)
				}
			}
		}
		scaffold(fr, n, float64(sz.ay-cut))
	}
	stakes(fr, n)
	if stage == 0 {
		fr.crate(float64(n)*0.55, float64(n)*0.8, 0.3)
		fr.c.outline(ink)
	}
}

var Unbuilt = map[string]bool{"shrine": true, "signpost": true, "foreign": true, "harbor-gate": true, "sea-chancery": true}

var fewVariants = map[string]int{"sea-chancery": 1}

func variantsOf(kind string) int {
	if n, ok := fewVariants[kind]; ok {
		return n
	}
	return BuildingVariants
}

func addAged(b *atlasBuilder, kind string, m *agedMemo) {
	sz := buildingSize(kind)
	add := func(name string, paint func(c *canvas)) { b.add(name, sz.w, sz.h, sz.ax, sz.ay, paint) }
	if !Unbuilt[kind] {
		add(buildingName(kind, "construction-0"), func(c *canvas) { m.paintConstruction(c, kind, sz, newLook(kind, 0, 0), 0) })
	}
	for age := 0; age < AgeTiers; age++ {
		for v := 0; v < variantsOf(kind); v++ {
			l := newLook(kind, age, v)
			add(BuildingLookName(kind, "ready", age, v), func(c *canvas) { c.paste(m.shot(kind, sz, l).full, 0, 0, false) })
		}
		l := newLook(kind, age, 0)
		add(BuildingLookName(kind, "dark", age, 0), func(c *canvas) { m.paintDark(c, kind, sz, l) })
		if Unbuilt[kind] {
			continue
		}
		for stage := 1; stage < 3; stage++ {
			st := stage
			add(BuildingLookName(kind, stateName(st), age, 0), func(c *canvas) { m.paintConstruction(c, kind, sz, l, st) })
		}
	}
}
