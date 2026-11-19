package world

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	tileBurst   = .3
	propBurst   = .45
	propLift    = .02
	colonnadeAt = -.35
)

func (r *Renderer) forumDecorOf(site view.Site) *forumDecor {
	if sig := forumSig(site); r.decor == nil || r.decor.sig != sig {
		d := planForumDecor(site)
		r.decor = &d
	}
	return r.decor
}

func (r *Renderer) drawForumGround(dst *ebiten.Image, cam render.Camera, sc Scene) {
	r.drawMarketGround(dst, cam, sc)
	if sc.Site.Forum == nil {
		return
	}
	d := r.forumDecorOf(sc.Site)
	clock := r.agesOf()
	clock.track(sc.Age, sc.Time)
	x0, y0, x1, y1 := visibleTiles(dst, cam, sc.Site)
	zoom := cam.Scale()
	for i, t := range d.tiles {
		if t.at.X < x0 || t.at.X > x1 || t.at.Y < y0 || t.at.Y > y1 {
			continue
		}
		age, _ := clock.phase(t.id)
		sx, sy := ScreenPointOf(cam, sc.Site, Vec{float64(t.at.X), float64(t.at.Y)})
		for _, name := range d.tileNames(i, age) {
			r.Lib.Draw(dst, name, sx, sy, zoom, render.SpriteOptions{})
		}
	}
}

func (r *Renderer) forumItems(cam render.Camera, sc Scene) []Drawable {
	if sc.Site.Forum == nil {
		return nil
	}
	d := r.forumDecorOf(sc.Site)
	clock := r.agesOf()
	clock.track(sc.Age, sc.Time)
	zoom := cam.Scale()
	out := make([]Drawable, 0, len(d.kerb.pieces)+len(d.kerb.posts)+len(d.props))
	for _, k := range d.kerb.pieces {
		age, _ := clock.phase(k.id)
		if age >= 1 && !(age == tierProd && d.edges[k.mid]) {
			out = r.kerbItem(out, cam, sc, k.mid, art.ForumKerbName(k.axis, age, k.v), kerbDepth(k.mid)-kerbBias, zoom)
		}
	}
	for _, p := range d.kerb.posts {
		if age, _ := clock.phase(p.id); age >= 1 {
			out = r.kerbItem(out, cam, sc, p.at, art.ForumPostName(age, p.v), kerbDepth(p.at), zoom)
		}
	}
	for _, p := range d.props {
		out = r.propItem(out, cam, sc, p, zoom)
	}
	return append(out, r.forumBursts(cam, sc, d)...)
}

func (r *Renderer) kerbItem(out []Drawable, cam render.Camera, sc Scene, at Vec, name string, depth, zoom float64) []Drawable {
	sx, sy := ScreenPointOf(cam, sc.Site, at)
	if !onScreen(sx, sy, zoom) {
		return out
	}
	return append(out, Drawable{Depth: depth, Draw: func(dst *ebiten.Image) {
		r.Lib.Draw(dst, name, sx, sy, zoom, render.SpriteOptions{})
	}})
}

func propName(p forumProp, age int) string {
	switch p.kind {
	case PropTerminus:
		return art.ForumTerminusName(p.v)
	case PropStatue:
		if age < tierProd {
			return art.ForumPlinthName(age)
		}
		return art.ForumStatueName(p.v)
	case PropFountain:
		return art.ForumFountainName(age)
	case PropRostra:
		return art.ForumRostraName(p.axis)
	case PropArch:
		return art.ForumArchName(p.axis)
	}
	return art.ForumColonnadeName(p.axis, p.v)
}

func (r *Renderer) propItem(out []Drawable, cam render.Camera, sc Scene, p forumProp, zoom float64) []Drawable {
	age, _ := r.agesOf().phase(p.id)
	if age < p.from || age > p.upTo {
		return out
	}
	depth := render.DepthKey(p.at.X, p.at.Y) + propLift
	if p.kind == PropColonnade {
		depth = kerbDepth(p.at) + colonnadeAt
	}
	return r.kerbItem(out, cam, sc, p.at, propName(p, age), depth, zoom)
}

func (r *Renderer) forumBursts(cam render.Camera, sc Scene, d *forumDecor) []Drawable {
	clock := r.agesOf()
	out := []Drawable{}
	for _, t := range d.tiles {
		if _, busy := clock.phase(t.id); busy {
			out = append(out, r.ageBurst(cam, sc, Vec{float64(t.at.X), float64(t.at.Y)}, tileBurst, t.id)...)
		}
	}
	for _, p := range d.props {
		if age, busy := clock.phase(p.id); busy && age >= p.from && age <= p.upTo {
			out = append(out, r.ageBurst(cam, sc, p.at, propBurst, p.id)...)
		}
	}
	return out
}
