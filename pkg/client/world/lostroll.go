package world

import (
	"math"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
)

const (
	fallenEvery  = 1.8
	fallenPuffs  = 3
	fallenSpread = .22
	fallenBias   = .02
)

var fallenTint = [3]float32{1.2, .75, .7}

func (l *living) keepFallen(sc Scene, live map[string]bool) {
	if l.fallen == nil {
		l.fallen = map[string]Vec{}
	}
	for name, home := range l.bare {
		if !live[name] && slices.Contains(sc.Fallen, name) {
			l.fallen[name] = home
		}
	}
	for name := range l.fallen {
		if !slices.Contains(sc.Fallen, name) {
			delete(l.fallen, name)
		}
	}
}

func FallenPuff(t float64, seed uint32, i int) (float64, bool) {
	phase := math.Mod(t+float64(seed%97)/97*fallenEvery, fallenEvery)
	return BurstPuff(phase, i)
}

func (r *Renderer) lostRollItems(out []Drawable, cam render.Camera, sc Scene, l *living) []Drawable {
	names := make([]string, 0, len(l.fallen))
	for name := range l.fallen {
		names = append(names, name)
	}
	slices.Sort(names)
	zoom := cam.Scale()
	for _, name := range names {
		home, seed := l.fallen[name], art.NameSeed(name)
		at := Vec{home.X + .5, home.Y + .5}
		for i := range fallenPuffs {
			tt, live := FallenPuff(sc.Time, seed, i)
			if !live || sc.Reduce && i > 0 {
				continue
			}
			ang := float64(seed%360)*math.Pi/180 + float64(i)*2.1
			sx, sy := ScreenPointOf(cam, sc.Site, Vec{at.X + math.Cos(ang)*fallenSpread, at.Y + math.Sin(ang)*fallenSpread})
			k := tt / burstLife
			frame := gfx.EffectName("dust", int(k*float64(art.EffectFrames)))
			opts := render.SpriteOptions{Alpha: float32(1 - k*k), Scale: burstScale + k*burstGrow, Tint: fallenTint}
			y := sy - tt*burstRise*zoom
			out = append(out, Drawable{Depth: render.DepthKey(at.X, at.Y) + fallenBias, Draw: func(dst *ebiten.Image) {
				r.Lib.Draw(dst, frame, sx, y, zoom, opts)
			}})
		}
	}
	return out
}
