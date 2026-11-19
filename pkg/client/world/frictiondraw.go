package world

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	ruinLife  = 90.0
	maxRuins  = 4
	ruinAlpha = .8
	wardAhead = 1.0
	pileAside = .8
	pileFront = .1
)

var (
	wardName  = gfx.AmbientName("ward", 0)
	pileNames = func() [gfx.ScrollPiles]string {
		var out [gfx.ScrollPiles]string
		for i := range out {
			out[i] = gfx.AmbientName("scrolls", i)
		}
		return out
	}()
)

type ruin struct {
	home  Vec
	tier  int
	since float64
	pod   string
}

func WardSpot(site view.Site, k view.Structure) Vec {
	lo, hi := Footprint(k)
	front := Vec{float64(hi.X) + wardAhead, float64(hi.Y) + wardAhead}
	at, ok := firstClear(site, nil, []Vec{front,
		{float64(hi.X) + wardAhead, float64(hi.Y)}, {float64(hi.X), float64(hi.Y) + wardAhead},
		{float64(hi.X) + wardAhead, float64(lo.Y)}, {float64(lo.X), float64(hi.Y) + wardAhead}})
	if !ok {
		return front
	}
	return at
}

func (r *Renderer) frictionItems(out []Drawable, cam render.Camera, sc Scene) []Drawable {
	l := r.lifeOf()
	l.noteCamps(sc, r.Homes)
	l.sets.fill(sc, l.people.of(sc))
	out = r.homelessItems(out, cam, sc)
	out = r.wardItems(out, cam, sc, l.sets.warded)
	out = r.decreeItems(out, cam, sc)
	out = r.backlogItems(out, cam, sc)
	return r.ruinItems(out, cam, sc, l.sets.unreplaced)
}

func (r *Renderer) homelessItems(out []Drawable, cam render.Camera, sc Scene) []Drawable {
	zoom := cam.Scale()
	for _, p := range sc.Pods {
		home, ok := r.Homes[p.Name]
		if p.System || p.State != model.HearthPending || !ok {
			continue
		}
		at := CampSpot(home)
		pose := Pose{Pos: at, Action: "idle", Dir: "se", Frame: Frame(sc.Time+seedOffset(model.Hash(p.Name)), 2)}
		d := r.walkerDrawable(cam, sc, p.Sprite, pose, &p, true)
		sx, sy := ScreenPointOf(cam, sc.Site, at)
		if !onScreen(sx, sy, zoom) {
			continue
		}
		inner := d.Draw
		d.Draw = func(dst *ebiten.Image) {
			r.Lib.Draw(dst, bundleName, sx-bundleSide*zoom, sy, zoom, render.SpriteOptions{})
			inner(dst)
		}
		out = append(out, d)
	}
	return out
}

func (r *Renderer) wardItems(out []Drawable, cam render.Camera, sc Scene, warded map[string]bool) []Drawable {
	zoom := cam.Scale()
	for _, k := range Keeps(sc.Site) {
		spot := WardSpot(sc.Site, k)
		lift, lifting := WardLift(sc, spot)
		if !warded[k.Object.Name] && !lifting {
			continue
		}
		sx, sy := ScreenPointOf(cam, sc.Site, spot)
		if !onScreen(sx, sy, zoom) {
			continue
		}
		y := sy - lift*wardRise*zoom
		alpha := float32(max(1-lift, .05))
		out = append(out, propDrawable(Drawable{Depth: render.DepthKey(spot.X, spot.Y), Draw: func(dst *ebiten.Image) {
			r.Lib.Draw(dst, wardName, sx, y, zoom, render.SpriteOptions{Alpha: alpha})
		}}, spot, sx, sy))
	}
	return out
}

func (r *Renderer) backlogItems(out []Drawable, cam render.Camera, sc Scene) []Drawable {
	level := ScrollPile(sc.Ledger)
	archive, ok := FirstOfKind(sc.Site, view.StructArchive, true)
	if level < 0 || !ok {
		return out
	}
	door := Door(archive)
	at, ok := firstClear(sc.Site, r.hamletOf(), []Vec{{door.X - pileAside, door.Y + pileFront}, {door.X + pileAside, door.Y + pileFront}, {door.X, door.Y + pileAside}})
	if !ok {
		return out
	}
	zoom := cam.Scale()
	sx, sy := ScreenPointOf(cam, sc.Site, at)
	if !onScreen(sx, sy, zoom) {
		return out
	}
	name := pileNames[min(level, len(pileNames)-1)]
	return append(out, propDrawable(Drawable{Depth: render.DepthKey(at.X, at.Y), Draw: func(dst *ebiten.Image) {
		r.Lib.Draw(dst, name, sx, sy, zoom, render.SpriteOptions{})
	}}, at, sx, sy))
}

func (l *life) noteRuins(sc Scene, homes map[string]Vec) {
	kept := l.ruins[:0]
	for _, u := range l.ruins {
		if sc.Time-u.since < ruinLife && sc.Time >= u.since && !homeTaken(homes, u.home) {
			kept = append(kept, u)
		}
	}
	l.ruins = kept
	for _, e := range sc.Errands {
		if e.Kind != ErrandFallen || len(l.ruins) >= maxRuins || hasRuin(l.ruins, e.Pod) {
			continue
		}
		l.ruins = append(l.ruins, ruin{home: Vec{e.To.X - .45, e.To.Y - .8}, tier: max(e.Tier, 1), since: e.Start, pod: e.Pod})
	}
}

func homeTaken(homes map[string]Vec, at Vec) bool {
	for _, h := range homes {
		if Dist(h, at) < .5 {
			return true
		}
	}
	return false
}

func hasRuin(ruins []ruin, pod string) bool {
	for _, u := range ruins {
		if u.pod == pod {
			return true
		}
	}
	return false
}

func (r *Renderer) ruinItems(out []Drawable, cam render.Camera, sc Scene, unreplaced bool) []Drawable {
	l := r.lifeOf()
	l.noteRuins(sc, r.Homes)
	if !unreplaced {
		return out
	}
	zoom := cam.Scale()
	for _, u := range l.ruins {
		a := Vec{u.home.X + .5, u.home.Y + .5}
		sx, sy := ScreenPointOf(cam, sc.Site, Vec{a.X - .01, a.Y - .01})
		if !onScreen(sx, sy, zoom) {
			continue
		}
		name := gfx.HearthName(u.tier, model.HearthBanished)
		out = append(out, Drawable{Depth: render.DepthKey(a.X, a.Y), Draw: func(dst *ebiten.Image) {
			r.Lib.Draw(dst, name, sx, sy, zoom, render.SpriteOptions{Alpha: ruinAlpha})
		}})
	}
	return out
}
