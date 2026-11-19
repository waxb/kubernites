package art

import (
	"math"
	"sort"
)

type boxStack struct {
	gx, gy float64
	sizes  []int
}

var boxPileLayouts = [][]boxStack{
	{{-0.18, 0.05, []int{1}}, {0.22, -0.14, []int{0}}},
	{{-0.2, -0.2, []int{2, 0}}, {0.24, 0.02, []int{1}}, {-0.08, 0.3, []int{0}}, {0.3, -0.3, []int{0}}},
	{{-0.24, -0.24, []int{2, 1, 0}}, {0.24, -0.22, []int{2, 1}}, {-0.22, 0.24, []int{1, 1}}, {0.24, 0.26, []int{1, 0}}},
	{{-0.12, -0.12, []int{3, 2, 1}}, {0.45, -0.25, []int{1, 0, 1}}, {-0.25, 0.45, []int{1, 0}}, {0.5, 0.5, []int{2, 1, 0}}},
}

type pileItem struct {
	gx, gy, z float64
	size, v   int
}

func pileItems(kind string, level int) []pileItem {
	out := []pileItem{}
	seed := int(roleSeed(kind) % 991)
	for i, st := range boxPileLayouts[min(max(level, 0), BoxPileLevels-1)] {
		z := 0.0
		for j, n := range st.sizes {
			v := int(hash(level*7+i, j, seed) % BoxVariants)
			out = append(out, pileItem{gx: st.gx, gy: st.gy, z: z, size: n, v: v})
			z += worldBoxes[n].h
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		di, dj := out[i].gx+out[i].gy, out[j].gx+out[j].gy
		if math.Abs(di-dj) > 1e-9 {
			return di < dj
		}
		return out[i].z < out[j].z
	})
	return out
}

func paintBoxPile(c *canvas, kind string, level int) {
	s := iso{c: c, ox: boxPileAX, oy: boxPileAY}
	for _, it := range pileItems(kind, level) {
		sz := boxCanvas(it.size)
		box := newCanvas(sz.w, sz.h)
		drawBox(box, float64(sz.ax), float64(sz.ay), worldBoxes[it.size], kind, it.v)
		box.outline(ink)
		p := s.p(it.gx, it.gy, it.z)
		c.paste(box, int(math.Round(p.x))-sz.ax, int(math.Round(p.y))-sz.ay, false)
	}
	propShadow(c, boxPileAX, boxPileAY+1, 11+2*float64(level))
}
