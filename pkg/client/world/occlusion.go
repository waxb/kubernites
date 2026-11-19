package world

import (
	"math"

	"github.com/waxb/kubernites/pkg/client/gfx"
)

const (
	hideShare  = .1
	hideMemory = 4096
)

type hideKey struct {
	back, front string
	dx, dy      int
}

func HiddenShare(back, front gfx.Mask, ox, oy int) float64 {
	seen, hid := 0, 0
	for y := range back.H {
		for x := range back.W {
			if !back.At(x, y) {
				continue
			}
			seen++
			if front.At(x-ox, y-oy) {
				hid++
			}
		}
	}
	if seen == 0 {
		return 0
	}
	return float64(hid) / float64(seen)
}

func (r *Renderer) hides(zoom float64) func(back, front Placed) bool {
	return func(back, front Placed) bool {
		return r.hiddenShare(back, front, zoom) >= hideShare
	}
}

func (r *Renderer) hiddenShare(back, front Placed, zoom float64) float64 {
	key := hideKey{back: r.ShownName(back.names), front: r.ShownName(front.names),
		dx: int(math.Round((front.sx - back.sx) / zoom)), dy: int(math.Round((front.sy - back.sy) / zoom))}
	if share, ok := r.shares[key]; ok {
		return share
	}
	if r.shares == nil || len(r.shares) >= hideMemory {
		r.shares = map[hideKey]float64{}
	}
	share := r.spriteShare(key)
	r.shares[key] = share
	return share
}

func (r *Renderer) spriteShare(k hideKey) float64 {
	_, back, okBack := r.Lib.Lookup(k.back)
	_, front, okFront := r.Lib.Lookup(k.front)
	backMask, hasBack := r.Lib.Mask(k.back)
	frontMask, hasFront := r.Lib.Mask(k.front)
	if !okBack || !okFront || !hasBack || !hasFront {
		return 0
	}
	return HiddenShare(backMask, frontMask, k.dx+back.AnchorX-front.AnchorX, k.dy+back.AnchorY-front.AnchorY)
}
