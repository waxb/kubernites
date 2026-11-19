package world

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	HitStone        = "stone"
	HitDormantStone = "stone-dormant"
	stoneGlowR      = 16.0
	stoneGlowMax    = .75
	stoneGlowRed    = .35
	stoneGlowGreen  = .85
	stoneGlowBlue   = 2.0
	boothFront      = 10
	stoneWakeSecs   = 1.4
	stoneFadeShare  = .7
	stoneFadeFloor  = .02
	stoneWakeBloom  = 1.1
)

func StoneTile(site view.Site) (Point, bool) {
	p := Point{site.Plaza.X, site.Plaza.Y}
	return p, site.Tile(p.X, p.Y) == view.TilePlaza
}

func StoneSpot(site view.Site) (Vec, bool) {
	p, ok := StoneTile(site)
	return p.Center(), ok
}

func stoneBox() ui.Rect {
	return ui.Rect{X: -art.SpeakStoneHalfW, Y: -art.SpeakStoneTall, W: 2 * art.SpeakStoneHalfW, H: art.SpeakStoneTall + art.SpeakStoneFoot}
}

func clearOfStone(site view.Site, v Vec) bool {
	at, ok := StoneSpot(site)
	if !ok {
		return true
	}
	dx := (v.X - at.X - (v.Y - at.Y)) * render.TileWidth / 2
	dy := (v.X - at.X + v.Y - at.Y) * render.TileHeight / 2
	booth := ui.Rect{X: int(math.Round(dx)) - art.StallAX, Y: int(math.Round(dy)) - art.StallAY, W: art.StallW, H: art.StallH + boothFront}
	return !Overlaps(booth, stoneBox())
}

type stoneState struct {
	seen, awake, waking bool
	woke                float64
}

func (r *Renderer) noteStone(awake bool, now float64) {
	st := r.stone
	if st.seen && !st.awake && awake {
		st.waking, st.woke = true, now
	}
	st.seen, st.awake = true, awake
	r.stone = st
}

func (r *Renderer) StoneWaking(now float64) (float64, bool) {
	if !r.stone.waking {
		return 1, false
	}
	p := (now - r.stone.woke) / stoneWakeSecs
	if p >= 1 || p < 0 {
		return 1, false
	}
	return p, true
}

func (r *Renderer) stoneItems(out []Drawable, cam render.Camera, sc Scene) []Drawable {
	at, ok := StoneSpot(sc.Site)
	if !ok {
		return out
	}
	now := sc.RealTime()
	r.noteStone(sc.Stone, now)
	zoom := cam.Scale()
	sx, sy := ScreenPointOf(cam, sc.Site, at)
	if !onScreen(sx, sy, zoom) {
		return out
	}
	look := art.SpeakStoneLook(sc.Age)
	dormant := art.SpeakStoneName(look, art.StoneDormant)
	if !sc.Stone {
		asleep := &Hit{Kind: HitDormantStone, ID: HitStone, Rect: SpriteRect(r.Lib, dormant, sx, sy, zoom, art.SpeakStoneW, art.SpeakStoneH).Inset(-hitPad)}
		return append(out, Drawable{Depth: render.DepthKey(at.X, at.Y), Hit: asleep, Draw: func(dst *ebiten.Image) {
			r.Lib.Draw(dst, dormant, sx, sy, zoom, render.SpriteOptions{})
		}})
	}
	awake := art.SpeakStoneName(look, art.StoneAwake)
	if LampsLit(sc.DayPhase) {
		awake = art.SpeakStoneName(look, art.StoneLit)
	}
	fade := float32(1)
	if p, waking := r.StoneWaking(now); waking {
		fade = float32(max(min(p/stoneFadeShare, 1), stoneFadeFloor))
	}
	hit := &Hit{Kind: HitStone, ID: HitStone, Rect: SpriteRect(r.Lib, awake, sx, sy, zoom, art.SpeakStoneW, art.SpeakStoneH).Inset(-hitPad)}
	return append(out, Drawable{Depth: render.DepthKey(at.X, at.Y), Hit: hit, Draw: func(dst *ebiten.Image) {
		if fade < 1 {
			r.Lib.Draw(dst, dormant, sx, sy, zoom, render.SpriteOptions{})
		}
		r.Lib.Draw(dst, awake, sx, sy, zoom, render.SpriteOptions{Alpha: fade})
	}})
}

func (r *Renderer) stoneBloom(sc Scene) float64 {
	night := 0.0
	if dark := Darkness(sc.DayPhase); dark > lampDusk {
		night = min(dark/.55, 1) * stoneGlowMax
	}
	if p, waking := r.StoneWaking(sc.RealTime()); waking {
		return max(night, stoneWakeBloom*math.Sin(math.Pi*p))
	}
	return night
}

func (r *Renderer) drawStoneGlow(dst *ebiten.Image, cam render.Camera, sc Scene) {
	at, ok := StoneSpot(sc.Site)
	bloom := r.stoneBloom(sc)
	if !sc.Stone || !ok || bloom <= 0 {
		return
	}
	zoom := cam.Scale()
	sx, sy := ScreenPointOf(cam, sc.Site, at)
	if !onScreen(sx, sy, zoom) {
		return
	}
	strength := float32(bloom)
	if r.glow == nil {
		r.glow = glowImage()
	}
	radius := stoneGlowR * zoom
	lift := art.SpeakStoneGlowLift(art.SpeakStoneLook(sc.Age)) * zoom
	op := &ebiten.DrawImageOptions{Blend: ebiten.BlendLighter}
	scale := radius * 2 / glowSize
	op.GeoM.Scale(scale, scale*.7)
	op.GeoM.Translate(sx-radius, sy-lift-radius*.7)
	op.ColorScale.Scale(strength*stoneGlowRed, strength*stoneGlowGreen, strength*stoneGlowBlue, strength)
	dst.DrawImage(r.glow, op)
}
