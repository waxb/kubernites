package world

import (
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	lifeItemsGuess         = 96
	drawFiguresWithoutPods = false
)

type life struct {
	sig    uint64
	vents  map[string]vent
	folk   []stroll
	stalls []Vec
	road   []Vec
	carts  cartYard
	fan    fanWatch
	camps  map[string]camp
	boats  []fishingBoat
	blob   *ebiten.Image
	ruins  []ruin
	names  map[folkKey]string
	fanNow []fanTarget
	sets   frictionSets
	lamps  []lampSpot
	fanBuf []fanTarget
	fanAt  float64
	fanFor *model.PodView
	people census
	on     plannedSite
}

func (r *Renderer) lifeOf() *life {
	if r.life == nil {
		r.life = &life{vents: map[string]vent{}, camps: map[string]camp{}}
	}
	return r.life
}

func (l *life) hutVent(name string) vent {
	if v, ok := l.vents[name]; ok {
		return v
	}
	v := HutVent(name)
	l.vents[name] = v
	return v
}

func lifeSig(site view.Site, landVersion uint64) uint64 {
	return StructureSig(site) ^ landVersion*1099511628211 ^ uint64(site.Width)<<32
}

func (r *Renderer) syncLife(sc Scene) *life {
	l := r.lifeOf()
	version := uint64(0)
	if r.Land != nil {
		version = r.Land.version
	}
	if sig := lifeSig(sc.Site, version); sig != l.sig || l.road == nil && !l.on.same(sc.Site) {
		l.sig = sig
		l.plan(sc.Site)
	}
	return l
}

func (l *life) plan(site view.Site) {
	g := roadGraphOf(site)
	if drawFiguresWithoutPods {
		l.folk = planFolk(site, g, maxFolk)
	}
	l.stalls = StallSpots(site, maxStalls)
	l.road = CartRoute(site, g)
	l.boats = PlanBoats(site)
	l.on.keep(site)
}

type plannedSite struct {
	site  view.Site
	forum view.Forum
	ok    bool
}

func (p *plannedSite) keep(s view.Site) {
	k := &p.site
	k.Gate, k.Plaza, k.TowerSpot, k.Width, k.Height, k.NoHarbor = s.Gate, s.Plaza, s.TowerSpot, s.Width, s.Height, s.NoHarbor
	k.Tiles = append(k.Tiles[:0], s.Tiles...)
	k.Elevation = append(k.Elevation[:0], s.Elevation...)
	k.Structures = append(k.Structures[:0], s.Structures...)
	k.Islands = append(k.Islands[:0], s.Islands...)
	k.Region = append(k.Region[:0], s.Region...)
	k.Power = append(k.Power[:0], s.Power...)
	k.Wear = append(k.Wear[:0], s.Wear...)
	k.Havens = append(k.Havens[:0], s.Havens...)
	k.Spots = append(k.Spots[:0], s.Spots...)
	k.Plots = append(k.Plots[:0], s.Plots...)
	k.Forum = nil
	if s.Forum != nil {
		p.forum = *s.Forum
		k.Forum = &p.forum
	}
	p.ok = true
}

func (p *plannedSite) same(s view.Site) bool {
	k := &p.site
	return p.ok && k.Gate == s.Gate && k.Plaza == s.Plaza && k.TowerSpot == s.TowerSpot && k.Width == s.Width && k.Height == s.Height && k.NoHarbor == s.NoHarbor &&
		(k.Forum == nil) == (s.Forum == nil) && (s.Forum == nil || *k.Forum == *s.Forum) &&
		slices.Equal(k.Tiles, s.Tiles) && slices.Equal(k.Elevation, s.Elevation) && slices.Equal(k.Structures, s.Structures) &&
		slices.Equal(k.Islands, s.Islands) && slices.Equal(k.Region, s.Region) && slices.Equal(k.Power, s.Power) && slices.Equal(k.Wear, s.Wear) &&
		slices.Equal(k.Havens, s.Havens) && slices.Equal(k.Spots, s.Spots) && slices.Equal(k.Plots, s.Plots)
}

func (r *Renderer) lifeItems(cam render.Camera, sc Scene) []Drawable {
	l := r.syncLife(sc)
	out := make([]Drawable, 0, lifeItemsGuess)
	out = r.frictionItems(out, cam, sc)
	out = r.smokeItems(out, cam, sc)
	out = r.lampItems(out, cam, sc)
	out = r.cartItems(out, cam, sc, l)
	out = r.stallItems(out, cam, sc, l)
	if drawFiguresWithoutPods {
		out = r.folkItems(out, cam, sc, l)
	}
	out = r.dockItems(out, cam, sc)
	out = r.haulerItems(out, cam, sc)
	return r.boatItems(out, cam, sc, l)
}

func (r *Renderer) drawLifeGround(dst *ebiten.Image, cam render.Camera, sc Scene) {
	r.watchFan(sc)
	r.drawFanLines(dst, cam, sc)
}

func (r *Renderer) drawLifeOverlay(dst *ebiten.Image, cam render.Camera, sc Scene) {
	r.drawWeather(dst, cam, sc)
}

func (r *Renderer) drawLifeGlow(dst *ebiten.Image, cam render.Camera, sc Scene) {
	r.drawLampGlow(dst, cam, sc)
	r.drawStoneGlow(dst, cam, sc)
}
