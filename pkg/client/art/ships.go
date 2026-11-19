package art

import (
	"fmt"
	"math"
	"sort"
)

const (
	ShipVariants = 8
	ShipFrames   = 4
)

var ShipClasses = []string{"cog", "caravel", "galleon", "carrack"}

var ShipDirs = []string{"ne", "nw", "se", "sw"}

var (
	shipBob    = []float64{0, 1, 1, 0}
	shipBillow = []float64{0.8, 1, 0.9, 0.65}
)

type castleSpec struct{ from, to, height float64 }

type yardSpec struct{ top, foot, span float64 }

type mastSpec struct {
	at, height          float64
	yards               []yardSpec
	lateen              bool
	fore, aft, over     float64
	nest                bool
	width               int
	pennant, flutterAmp float64
}

type shipSpec struct {
	w, h, ax, ay           int
	length, beam           float64
	freeboard, sheer, rise float64
	bulwark, stripe        float64
	bowPow, sternPinch     float64
	castles                []castleSpec
	masts                  []mastSpec
	bowsprit, billow       float64
	gunports, stays        bool
}

func shipName(class string, variant int, dir string, frame int) string {
	return fmt.Sprintf("ship/%s/%d/%s/%d", class, variant, dir, frame)
}

func addShips(b *atlasBuilder) {
	for _, class := range ShipClasses {
		sp := shipSpecs[class]
		for v := 0; v < ShipVariants; v++ {
			lk := lookFor(class, v)
			for _, d := range ShipDirs {
				for f := 0; f < ShipFrames; f++ {
					dir, frame := d, f
					b.add(shipName(class, v, d, f), sp.w, sp.h, sp.ax, sp.ay, func(c *canvas) { paintShip(c, sp, lk, dir, frame) })
				}
			}
		}
	}
}

func (sp shipSpec) half() float64 { return sp.length / 2 }

func (sp shipSpec) halfBeam(t float64) float64 {
	b := sp.beam / 2
	if t >= 0 {
		return b * math.Pow(math.Max(1-math.Pow(t, sp.bowPow), 0), 0.55)
	}
	return b * (1 - sp.sternPinch*t*t)
}

func (sp shipSpec) rail(t float64) float64 {
	return sp.freeboard + sp.sheer*t*t + sp.rise*math.Pow(math.Max(t, 0), 3)
}

func (sp shipSpec) railU(u float64) float64 { return sp.rail(u / sp.half()) }

func (sp shipSpec) plan(t0, t1 float64, n int) []uv {
	out := make([]uv, 0, 2*n+2)
	for i := 0; i <= n; i++ {
		t := t0 + (t1-t0)*float64(i)/float64(n)
		out = append(out, uv{t * sp.half(), sp.halfBeam(t)})
	}
	for i := n; i >= 0; i-- {
		t := t0 + (t1-t0)*float64(i)/float64(n)
		out = append(out, uv{t * sp.half(), -sp.halfBeam(t)})
	}
	return out
}

const hullFlare = 0.8

func rake(u float64) float64 {
	if u > 0 {
		return 0.93
	}
	return 0.97
}

func (sp shipSpec) waterPlan(n int) []uv {
	top := sp.plan(-1, 1, n)
	out := make([]uv, len(top))
	for i, p := range top {
		out[i] = uv{p.u * rake(p.u), p.w * hullFlare}
	}
	return out
}

type shipScene struct {
	v     hullView
	sp    shipSpec
	lk    shipLook
	frame int
}

func paintShip(c *canvas, sp shipSpec, lk shipLook, dir string, frame int) {
	s := shipScene{v: newHullView(c, sp.ax, sp.ay, dir, shipBob[frame]), sp: sp, lk: lk, frame: frame}
	s.hull()
	s.upper()
	s.pennants()
	c.outline(ink)
	s.rigging()
	s.waterline()
}

type layer struct {
	d    float64
	draw func()
}

func (s shipScene) upper() {
	layers := make([]layer, 0, len(s.sp.castles)+len(s.sp.masts)+2)
	for _, cs := range s.sp.castles {
		cs := cs
		layers = append(layers, layer{s.castleDepth(cs), func() { s.castle(cs) }})
	}
	for _, m := range s.sp.masts {
		m := m
		layers = append(layers, layer{s.v.depth(m.at*s.sp.half(), 0), func() { s.mastGroup(m) }})
	}
	layers = append(layers, layer{s.v.depth(s.sp.half(), 0) + 0.01, s.bowsprit})
	layers = append(layers, layer{s.sternDepth(), s.lantern})
	sort.SliceStable(layers, func(i, j int) bool { return layers[i].d < layers[j].d })
	for _, l := range layers {
		l.draw()
	}
}

func (s shipScene) castleDepth(cs castleSpec) float64 {
	d := s.v.depth((cs.from+cs.to)/2*s.sp.half(), 0)
	for _, m := range s.sp.masts {
		if m.at >= cs.from && m.at <= cs.to {
			d = math.Min(d, s.v.depth(m.at*s.sp.half(), 0)-0.01)
		}
	}
	return d
}

func (s shipScene) sternDepth() float64 {
	for _, cs := range s.sp.castles {
		if cs.from <= -0.99 {
			return s.castleDepth(cs) + 0.001
		}
	}
	return s.v.depth(-s.sp.half(), 0) + 0.005
}
