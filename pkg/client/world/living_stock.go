package world

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
)

const (
	henFlock       = 3
	henRoam        = .55
	goatRoam       = .8
	livestockRate  = 2.2
	bigHamlet      = 8
	stockBias      = 0
	gullBias       = .01
	gullBob        = 5.0
	gullSpread     = .3
	gullDeckStride = 3
)

type beast struct {
	role   string
	anchor Vec
	roam   float64
	seed   uint32
}

func stockBeasts(f keepFolk, anchors []Vec) []beast {
	out := []beast{}
	hens := []int{henFlock, henFlock - 1, henFlock}
	for i, at := range anchors {
		h := mix32(model.Hash(f.s.ID) + uint32(i)*0x9e3779b9)
		switch {
		case i < 2 || i == 2 && f.huts >= bigHamlet:
			for k := 0; k < hens[i%len(hens)]; k++ {
				out = append(out, beast{role: "chicken", anchor: at, roam: henRoam, seed: mix32(h + uint32(k))})
			}
		default:
			out = append(out, beast{role: "goat", anchor: at, roam: goatRoam, seed: h})
		}
	}
	return out
}

func (b beast) stand(t float64) (Vec, bool, int) {
	phase := float64(b.seed%1000) / 100
	speed := .22 + float64(b.seed>>10%20)/200
	x := b.anchor.X + b.roam*math.Sin(t*speed+phase)
	y := b.anchor.Y + b.roam*.8*math.Sin(t*speed*.77+phase*1.3)
	vx := math.Cos(t*speed + phase)
	vy := .8 * .77 * math.Cos(t*speed*.77+phase*1.3)
	moving := math.Abs(vx)+math.Abs(vy) > .55
	frame := int(math.Floor((t+phase)*livestockRate)) % art.LivestockFrames
	if moving {
		frame = 2 + frame%2
	} else {
		frame %= 2
	}
	return Vec{x, y}, vx-vy < 0, frame
}

func (r *Renderer) stockItems(out []Drawable, cam render.Camera, sc Scene, l *living) []Drawable {
	h := r.hamletOf()
	for _, f := range l.folk {
		if !HoldsLivestock(f.huts) {
			continue
		}
		plan, ok := l.plans.of(f.s)
		if !ok {
			continue
		}
		for _, b := range stockBeasts(f, plan.stock) {
			t := sc.Time
			if sc.Reduce {
				t = 0
			}
			at, flip, frame := b.stand(t)
			if h.occAt(TileOf(at)) == occHut {
				continue
			}
			p := l.livestock(b.role, frame)
			out = r.pieceItem(out, cam, sc, p, at, stockBias, 1, flip)
		}
	}
	return out
}

func (l *living) livestock(role string, frame int) livingPiece {
	return l.piece(fmt.Sprintf("%s%d", role, frame), func() art.Piece {
		if role == "goat" {
			return art.PaintGoat(frame)
		}
		return art.PaintChicken(frame)
	})
}

func (r *Renderer) quayGullItems(out []Drawable, cam render.Camera, sc Scene, l *living) []Drawable {
	h := r.HarborOf()
	for i, plan := range h.Plans {
		if !plan.Open || i >= len(h.Fleets) || len(plan.Deck) == 0 {
			continue
		}
		in := r.shipsIn(plan, h.Fleets[i])
		if in == 0 {
			continue
		}
		p := l.piece("gull", func() art.Piece { return art.PaintGull(0) })
		for k := 0; k < min(1+in, maxQuayGulls); k++ {
			d := plan.Deck[(k*gullDeckStride+int(model.Hash(plan.ID)))%len(plan.Deck)]
			s := mix32(model.Hash(plan.ID) + uint32(k)*0x85ebca6b)
			at := Vec{float64(d.X) + jitterAt(s, 4, gullSpread), float64(d.Y) + jitterAt(s, 14, gullSpread)}
			lift := h.Lift(at)
			bob := 0.0
			if !sc.Reduce && math.Mod(sc.Time*.7+float64(s%50)/10, gullBob) < .35 {
				bob = 1
			}
			out = r.liftedItem(out, cam, sc, p, at, lift-bob, gullBias, s>>9&1 == 1)
		}
	}
	return out
}

func (r *Renderer) liftedItem(out []Drawable, cam render.Camera, sc Scene, p livingPiece, at Vec, lift, bias float64, flip bool) []Drawable {
	zoom := cam.Scale()
	sx, sy := ScreenPointOf(cam, sc.Site, at)
	sy -= lift * zoom
	if !onScreen(sx, sy, zoom) {
		return out
	}
	return append(out, Drawable{Depth: render.DepthKey(at.X, at.Y) + bias, Draw: func(dst *ebiten.Image) {
		p.draw(dst, sx, sy, zoom, render.SpriteOptions{Flip: flip})
	}})
}

func (r *Renderer) shipsIn(plan HavenPlan, f *fleet) int {
	now, n := r.HarborOf().Now, 0
	for _, s := range f.Ships {
		if len(plan.Berths) == 0 {
			break
		}
		pose, ok := ShipAt(s, plan.Berths[s.Berth%len(plan.Berths)], plan.Sea, now)
		if ok && !pose.Moving {
			n++
		}
	}
	return n
}
