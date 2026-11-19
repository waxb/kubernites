package world

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	AgeStagger  = 3.0
	AgeWork     = .9
	ageDustAt   = .3
	burstPuffs  = 5
	burstLife   = .6
	burstStep   = .07
	burstRise   = 22.0
	burstScale  = 1.3
	burstGrow   = 1.1
	hutBurst    = .35
	structBurst = .45
)

var burstTint = [3]float32{1.5, 1.45, 1.4}

type ageClock struct {
	from, to int
	at, now  float64
	primed   bool
}

type lookKey struct {
	kind, variant, state string
	age, look            int
}

func (a *ageClock) track(age int, now float64) {
	a.now = now
	switch {
	case !a.primed:
		a.from, a.to, a.at, a.primed = age, age, now, true
	case age < a.to:
		a.from, a.to, a.at = age, age, now
	case age > a.to:
		a.from, a.to, a.at = a.to, age, now
	case a.from != a.to && now-a.at > AgeStagger+AgeWork:
		a.from = a.to
	}
}

func AgeDelay(id string) float64 {
	return float64(art.NameSeed(id)%1000) / 1000 * AgeStagger
}

func (a *ageClock) phase(id string) (int, bool) {
	if a.from == a.to {
		return a.to, false
	}
	switch t := a.now - a.at - AgeDelay(id); {
	case t < 0:
		return a.from, false
	case t < AgeWork:
		return a.to, true
	}
	return a.to, false
}

func (r *Renderer) agesOf() *ageClock {
	if r.ages == nil {
		r.ages = &ageClock{}
	}
	return r.ages
}

func (r *Renderer) AgePhase(id string) (int, bool) { return r.agesOf().phase(id) }

func (r *Renderer) structureLook(s view.Structure, sc Scene) ([]string, bool) {
	clock := r.agesOf()
	clock.track(sc.Age, sc.Time)
	age, busy := clock.phase(s.ID)
	if busy && s.State == view.StateReady {
		s.State, s.Progress = view.StateConstruction, .9
	}
	if s.Kind == view.StructKeep {
		return gfx.KeepNames(s, age), busy
	}
	if s.Kind == view.StructVault {
		return []string{vaultStructureLook(s, sc)}, busy
	}
	return r.resolveLook(s, age), busy
}

func (r *Renderer) resolveLook(s view.Structure, age int) []string {
	key := lookKey{kind: s.Kind, variant: s.Variant, state: gfx.BuildingState(s), age: age, look: gfx.LookVariant(s.ID)}
	if names, ok := r.looks[key]; ok {
		return names
	}
	if r.looks == nil {
		r.looks = map[lookKey][]string{}
	}
	all := gfx.StructureLookNames(s, age)
	names := make([]string, 0, 2)
	for _, n := range all {
		if r.Lib.Has(n) {
			names = append(names, n)
		}
	}
	if len(names) == 0 {
		names = all
	}
	r.looks[key] = names
	return names
}

func (a *ageClock) elapsed(id string) float64 { return a.now - a.at - AgeDelay(id) }

func BurstPuff(t float64, i int) (float64, bool) {
	tt := t - float64(i)*burstStep
	return tt, tt >= 0 && tt < burstLife
}

func (r *Renderer) ageBurst(cam render.Camera, sc Scene, at Vec, spread float64, id string) []Drawable {
	t := r.agesOf().elapsed(id)
	zoom, seed := cam.Scale(), art.NameSeed(id)
	out := make([]Drawable, 0, burstPuffs)
	for i := 0; i < burstPuffs; i++ {
		tt, live := BurstPuff(t, i)
		if !live {
			continue
		}
		ang := float64(seed%360)*math.Pi/180 + float64(i)*2.4
		sx, sy := ScreenPointOf(cam, sc.Site, Vec{at.X + math.Cos(ang)*spread, at.Y + math.Sin(ang)*spread})
		k := tt / burstLife
		name := gfx.EffectName("dust", int(k*float64(art.EffectFrames)))
		opts := render.SpriteOptions{Alpha: float32(1 - k*k), Scale: burstScale + k*burstGrow, Tint: burstTint}
		y := sy - tt*burstRise*zoom
		out = append(out, Drawable{Depth: render.DepthKey(at.X, at.Y) + ageDustAt, Draw: func(dst *ebiten.Image) {
			r.Lib.Draw(dst, name, sx, y, zoom, opts)
		}})
	}
	return out
}

func FootprintCentre(s view.Structure) Vec {
	return Vec{float64(s.X) + float64(s.W)/2, float64(s.Y) + float64(s.H)/2}
}
