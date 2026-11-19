package world

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	lanternGap    = 2.6
	lanternSide   = .8
	lanternFirst  = 1.6
	lanternBias   = -.02
	lanternGlowOn = .9
)

type roadLamps struct {
	sig   uint64
	grid  landGrid
	have  bool
	roads map[string][]Vec
}

func (rl *roadLamps) lampsTo(site view.Site, depot view.Structure, from Vec) []Vec {
	sig := StructureSig(site)
	if !rl.have || rl.sig != sig {
		rl.sig, rl.grid, rl.have, rl.roads = sig, LandGridOf(site), true, map[string][]Vec{}
	}
	if v, ok := rl.roads[depot.ID]; ok {
		return v
	}
	v := lampSpotsAlong(site, rl.grid.Route(from, Door(depot)))
	rl.roads[depot.ID] = v
	return v
}

func lampSpotsAlong(site view.Site, route []Vec) []Vec {
	out := []Vec{}
	next := lanternFirst
	walked := 0.0
	for i := 1; i < len(route) && len(out) < maxLanterns; i++ {
		a, b := route[i-1], route[i]
		seg := Dist(a, b)
		if seg < 1e-6 {
			continue
		}
		for next-walked <= seg && len(out) < maxLanterns {
			at := Lerp(a, b, (next-walked)/seg)
			if spot, ok := roadside(site, at, Vec{(b.X - a.X) / seg, (b.Y - a.Y) / seg}); ok {
				out = append(out, spot)
			}
			next += lanternGap
		}
		walked += seg
	}
	return out
}

func roadside(site view.Site, at, dir Vec) (Vec, bool) {
	for _, side := range []float64{1, -1} {
		spot := Vec{at.X - dir.Y*lanternSide*side, at.Y + dir.X*lanternSide*side}
		p := TileOf(spot)
		if !WetAt(site, spot) && !covered(site, p.X, p.Y) && site.Tile(p.X, p.Y) != view.TileForest {
			return spot, true
		}
	}
	return Vec{}, false
}

type depotRoll struct {
	desired, ready int
}

func depotRolls(crews []model.CrewChain) map[string]depotRoll {
	out := map[string]depotRoll{}
	for _, c := range crews {
		if c.Depot == "" {
			continue
		}
		d := out[c.Depot]
		d.desired += c.Desired
		d.ready += c.Ready
		out[c.Depot] = d
	}
	return out
}

func nearestQuay(site view.Site, to Vec) (Vec, bool) {
	best, gap, found := Vec{}, math.MaxFloat64, false
	for _, h := range SiteHavens(site) {
		q := VecOf(h.Quay)
		if d := Dist(q, to); h.Open && d < gap {
			best, gap, found = q, d, true
		}
	}
	return best, found
}

type lampPost struct {
	at  Vec
	lit bool
}

func (r *Renderer) lanternItems(out []Drawable, cam render.Camera, sc Scene, l *living) []Drawable {
	rolls := depotRolls(sc.Crews)
	zoom := cam.Scale()
	posts := map[Point]lampPost{}
	keys := []Point{}
	for _, s := range sc.Site.Structures {
		roll, ok := rolls[s.Object.Name]
		if s.Object.Kind != ServiceKind || s.State != view.StateReady || !ok {
			continue
		}
		quay, found := nearestQuay(sc.Site, Door(s))
		if !found {
			continue
		}
		total, lit := LanternsLit(roll.desired, roll.ready)
		for i, at := range l.roads.lampsTo(sc.Site, s, quay) {
			if i >= total {
				break
			}
			k := TileOf(at)
			post, seen := posts[k]
			if !seen {
				keys = append(keys, k)
				post = lampPost{at: at}
			}
			post.lit = post.lit || i < lit
			posts[k] = post
		}
	}
	for _, k := range keys {
		out = r.lanternItem(out, cam, sc, posts[k].at, posts[k].lit, zoom, l)
	}
	return out
}

func (r *Renderer) lanternItem(out []Drawable, cam render.Camera, sc Scene, at Vec, lit bool, zoom float64, l *living) []Drawable {
	sx, sy := ScreenPointOf(cam, sc.Site, at)
	if !onScreen(sx, sy, zoom) {
		return out
	}
	if lit {
		l.glows = append(l.glows, at)
	}
	p := l.piece(fmt.Sprintf("lantern%v", lit), func() art.Piece { return art.PaintLantern(lit) })
	return append(out, Drawable{Depth: render.DepthKey(at.X, at.Y) + lanternBias, Draw: func(dst *ebiten.Image) {
		p.draw(dst, sx, sy, zoom, render.SpriteOptions{})
	}})
}

func (r *Renderer) drawLivingGlow(dst *ebiten.Image, cam render.Camera, sc Scene) {
	dark := Darkness(sc.DayPhase)
	l := r.livingOf()
	if dark <= lampDusk || len(l.glows) == 0 {
		return
	}
	strength := min(dark/.55, 1) * lampGlowMax
	zoom := cam.Scale()
	for _, at := range l.glows {
		sx, sy := ScreenPointOf(cam, sc.Site, at)
		if onScreen(sx, sy, zoom) {
			r.drawGlow(dst, sx, sy-lampHead*zoom, lampGlowR*zoom, strength)
		}
	}
}
