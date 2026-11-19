package world

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	boatOut     = 4.0
	boatStride  = 2.5
	boatSide    = 3.0
	boatLoop    = 1.3
	boatSquash  = .6
	boatTries   = 6
	boatSamples = 8
	gullLoop    = 1.1
	gullAlt     = 24
	gullOmega   = .7
	gullCount   = 2
)

var boatNames = func() [art.BoatFrames]string {
	var out [art.BoatFrames]string
	for i := range out {
		out[i] = gfx.EffectName("boat", i)
	}
	return out
}()

type fishingBoat struct {
	anchor Vec
	radius float64
	Period float64 `json:"-"`
	phase  float64
	seed   uint32
}

func (b fishingBoat) At(t float64) (Vec, string) {
	ang := 2 * math.Pi * (t + b.phase) / b.Period
	pos := Vec{b.anchor.X + math.Cos(ang)*b.radius, b.anchor.Y + math.Sin(ang)*b.radius*boatSquash}
	ahead := Vec{-math.Sin(ang), math.Cos(ang) * boatSquash}
	return pos, Facing(Vec{}, ahead)
}

func PlanBoats(site view.Site) []fishingBoat {
	out := []fishingBoat{}
	if site.Width == 0 {
		return out
	}
	keep := jettyKeepOut(PlanLanes(site))
	for _, h := range SiteHavens(site) {
		if !BoatsLaunch(h) {
			continue
		}
		for k := 0; k < 1+HavenTier(h)/2; k++ {
			if b, ok := boatFor(site, h, k, keep); ok {
				out = append(out, b)
			}
		}
	}
	return out
}

func jettyKeepOut(lanes []FerryLane) map[Point]bool {
	out := map[Point]bool{}
	for _, l := range lanes {
		for _, j := range []Jetty{l.Home, l.Away} {
			for _, t := range append(j.Claims(), TileOf(j.Dock)) {
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						out[Point{t.X + dx, t.Y + dy}] = true
					}
				}
			}
		}
	}
	return out
}

func boatFor(site view.Site, h view.Haven, k int, keep map[Point]bool) (fishingBoat, bool) {
	seed := mix32(model.Hash(h.ID) ^ uint32(k)*0x9e3779b9)
	quay, out, side := VecOf(h.Quay), VecOf(h.Out), VecOf(h.Along())
	for _, mirror := range []float64{1, -1} {
		for try := 0; try < boatTries; try++ {
			s := mirror * (boatSide + float64(k+try/2))
			if (seed>>uint(try))&1 == 1 {
				s = -s
			}
			anchor := offset(offset(quay, out, float64(h.Pier)+boatOut+boatStride*float64(k+try%2)), side, s)
			b := fishingBoat{anchor: anchor, radius: boatLoop, Period: 50 + float64(seed%40), phase: float64(seed >> 8 % 100), seed: seed}
			if boatClear(site, b, keep) {
				return b, true
			}
		}
	}
	return fishingBoat{}, false
}

func boatClear(site view.Site, b fishingBoat, keep map[Point]bool) bool {
	for i := 0; i < boatSamples; i++ {
		pos, _ := b.At(float64(i) * b.Period / boatSamples)
		for _, v := range []Vec{pos, {pos.X + .6, pos.Y + .6}} {
			if !WetAt(site, v) || keep[TileOf(v)] {
				return false
			}
		}
	}
	return WetAt(site, b.anchor)
}

func (r *Renderer) boatItems(out []Drawable, cam render.Camera, sc Scene, l *life) []Drawable {
	zoom := cam.Scale()
	resting := Darkness(sc.DayPhase) > restingDark
	for _, b := range l.boats {
		pos, dir := b.At(sc.Time)
		sx, sy := cam.GridToScreen(pos.X, pos.Y)
		if !shipOnScreen(sx, sy, zoom) {
			continue
		}
		name := boatNames[Frame(sc.Time+float64(b.seed%7), 2)]
		if sc.Reduce {
			name = boatNames[0]
		}
		flip := dir == "sw"
		out = append(out, Drawable{Depth: render.DepthKey(pos.X, pos.Y), Draw: func(dst *ebiten.Image) {
			r.Lib.Draw(dst, name, sx, sy, zoom, render.SpriteOptions{Flip: flip})
		}})
		if !resting && r.Land != nil {
			out = r.appendFlock(out, cam, sc, flock{role: "gull", center: pos, radius: gullLoop, omega: gullOmega, alt: gullAlt, count: gullCount, seed: b.seed})
		}
	}
	return out
}
