package world

import (
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	LensHeraldry   = "heraldry"
	heraldLotAlpha = 112
	heraldHutAlpha = 150
	heraldWash     = .4
	pennantStep    = .55
)

type Heraldry struct {
	Fiefs map[string]int
	Lots  []HeraldLot
	Keeps map[string][]int
}

type HeraldLot struct {
	Order int
	Tiles []Point
}

func NewHeraldry(namespaces []string, pods []model.PodView, lots []HeraldLot) *Heraldry {
	names := append([]string(nil), namespaces...)
	for _, p := range pods {
		if p.Namespace != "" {
			names = append(names, p.Namespace)
		}
	}
	h := &Heraldry{Fiefs: view.Heraldry(names), Lots: lots, Keeps: map[string][]int{}}
	seen := map[string]map[int]bool{}
	for _, p := range pods {
		if p.Node == "" || p.Terminating {
			continue
		}
		if seen[p.Node] == nil {
			seen[p.Node] = map[int]bool{}
		}
		seen[p.Node][h.Fiefs[p.Namespace]] = true
	}
	for node, orders := range seen {
		for o := range orders {
			h.Keeps[node] = append(h.Keeps[node], o)
		}
		sort.Ints(h.Keeps[node])
	}
	return h
}

func (sc Scene) heraldTint(namespace string) ([3]float32, bool) {
	if sc.Lens.Mode != LensHeraldry || sc.Heraldry == nil || namespace == "" {
		return [3]float32{}, false
	}
	order, ok := sc.Heraldry.Fiefs[namespace]
	if !ok {
		return [3]float32{}, false
	}
	c := art.FiefColour(view.FiefColour(order))
	wash := func(v uint8) float32 { return heraldWash + (1-heraldWash)*float32(v)/255 }
	return [3]float32{wash(c.R), wash(c.G), wash(c.B)}, true
}

func FiefTint(order int, alpha uint8) color.NRGBA {
	c := art.FiefColour(view.FiefColour(order))
	c.A = alpha
	return c
}

func (r *Renderer) drawHeraldGround(dst *ebiten.Image, cam render.Camera, sc Scene) {
	h := sc.Heraldry
	if h == nil {
		return
	}
	x0, y0, x1, y1 := visibleTiles(dst, cam, sc.Site)
	seen := func(t Point) bool { return t.X >= x0 && t.X <= x1 && t.Y >= y0 && t.Y <= y1 }
	lots := map[int]*vector.Path{}
	path := func(paths map[int]*vector.Path, order int) *vector.Path {
		p, ok := paths[order]
		if !ok {
			p = &vector.Path{}
			paths[order] = p
		}
		return p
	}
	for _, l := range h.Lots {
		for _, t := range l.Tiles {
			if seen(t) {
				addTile(path(lots, l.Order), cam, sc.Site, t.X, t.Y)
			}
		}
	}
	for _, s := range sc.Site.Structures {
		order, ok := h.Fiefs[s.Object.Namespace]
		if !ok || s.Object.Namespace == "" || s.Kind == view.StructKeep {
			continue
		}
		lo, hi := Footprint(s)
		if span, ok := clipSpan(lo, hi, x0, y0, x1, y1); ok {
			addTiles(path(lots, order), cam, sc.Site, span)
		}
	}
	huts := map[int]*vector.Path{}
	for _, p := range sc.Pods {
		home, ok := r.Homes[p.Name]
		if !ok {
			continue
		}
		t := Point{int(home.X + .5), int(home.Y + .5)}
		if seen(t) {
			addTile(path(huts, h.Fiefs[p.Namespace]), cam, sc.Site, t.X, t.Y)
		}
	}
	fillByOrder(dst, lots, heraldLotAlpha)
	fillByOrder(dst, huts, heraldHutAlpha)
}

func fillByOrder(dst *ebiten.Image, paths map[int]*vector.Path, alpha uint8) {
	orders := make([]int, 0, len(paths))
	for o := range paths {
		orders = append(orders, o)
	}
	sort.Ints(orders)
	for _, o := range orders {
		op := &vector.DrawPathOptions{}
		op.ColorScale.ScaleWithColor(FiefTint(o, alpha))
		vector.FillPath(dst, paths[o], &vector.FillOptions{}, op)
	}
}

func PennantSpots(keep view.Structure, n int) []Vec {
	lo, hi := Footprint(keep)
	front := []Vec{}
	for x := float64(lo.X); x <= float64(hi.X)+1.01; x += pennantStep {
		front = append(front, Vec{x, float64(hi.Y) + 1.25})
	}
	for y := float64(hi.Y) + 1 - pennantStep; y >= float64(lo.Y)-.01; y -= pennantStep {
		front = append(front, Vec{float64(hi.X) + 1.25, y})
	}
	if n > len(front) {
		n = len(front)
	}
	start := float64(len(front)-n) / 2
	out := make([]Vec, n)
	for i := range out {
		out[i] = front[int(math.Round(start))+i]
	}
	return out
}

func (r *Renderer) drawPennants(dst *ebiten.Image, cam render.Camera, sc Scene) {
	h := sc.Heraldry
	if h == nil {
		return
	}
	zoom := cam.Scale()
	for _, k := range Keeps(sc.Site) {
		orders := h.Keeps[k.Object.Name]
		spots := PennantSpots(k, len(orders))
		for i, at := range spots {
			sx, sy := ScreenPointOf(cam, sc.Site, at)
			if !onScreen(sx, sy, zoom) {
				continue
			}
			c := art.FiefColour(view.FiefColour(orders[i]))
			tint := [3]float32{float32(c.R) / 255, float32(c.G) / 255, float32(c.B) / 255}
			r.Lib.Draw(dst, art.PennantName(view.FiefCrest(orders[i])), sx, sy, zoom, render.SpriteOptions{Tint: tint})
		}
	}
}
