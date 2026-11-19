package hud

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	DecoCompass   = "compass"
	decoStepX     = 3
	decoStepY     = 2
	decoOdds      = 7
	decoClear     = 6
	FogTextHeight = 106
)

var decoKinds = []string{"hills", "forest", "waves", "mountains", "hills", "waves"}

type DecoSpot struct {
	Kind string
	X, Y int
}

func (d DecoSpot) Rect(size func(string) (int, int)) ui.Rect {
	w, h := size(d.Kind)
	return ui.Rect{X: d.X, Y: d.Y, W: w, H: h}
}

func mapHash(x, y int) int {
	v := uint32(x)*73856093 ^ uint32(y)*19349663 ^ 0x9e3779b9
	v ^= v >> 13
	v *= 0x5bd1e995
	v ^= v >> 15
	return int(v & 0x7fffffff)
}

func fogZone(l KMapLayout, era int) ui.Rect {
	e := l.Eras[era]
	mid := l.Height * KMapTile / 2
	return ui.Rect{X: e.Start*KMapTile + KMapTile, Y: mid - FogTextHeight/2, W: e.Size*KMapTile - 2*KMapTile, H: FogTextHeight}
}

func DecorBlocks(l KMapLayout, roads []KMapRoadTile) []ui.Rect {
	out := []ui.Rect{}
	for _, p := range l.Plaques {
		out = append(out, p.Rect().Inset(-decoClear))
	}
	for _, t := range roads {
		out = append(out, ui.Rect{X: t.X * KMapTile, Y: t.Y * KMapTile, W: KMapTile, H: KMapTile})
	}
	for i := range l.Eras {
		if l.Empty(i) {
			out = append(out, fogZone(l, i))
		}
	}
	return out
}

func decorFits(r ui.Rect, blocks []ui.Rect, l KMapLayout) bool {
	if r.W <= 0 || r.X < KMapTile || r.Y < 4 || r.X+r.W > (l.Width-1)*KMapTile || r.Y+r.H > l.Height*KMapTile-4 {
		return false
	}
	for _, b := range blocks {
		if Overlaps(r, b) {
			return false
		}
	}
	return true
}

func KnowledgeDecor(l KMapLayout, roads []KMapRoadTile, size func(string) (int, int)) []DecoSpot {
	blocks := DecorBlocks(l, roads)
	out := []DecoSpot{}
	if c, ok := placeCompass(l, blocks, size); ok {
		out = append(out, c)
		blocks = append(blocks, c.Rect(size).Inset(-decoClear))
	}
	for ty := 1; ty < l.Height-1; ty += decoStepY {
		for tx := 1; tx < l.Width-1; tx += decoStepX {
			hsh := mapHash(tx, ty)
			if hsh%decoOdds != 0 {
				continue
			}
			d := DecoSpot{Kind: decoKinds[hsh/decoOdds%len(decoKinds)], X: tx*KMapTile + hsh/31%8, Y: ty*KMapTile + hsh/53%8}
			if r := d.Rect(size); decorFits(r, blocks, l) {
				out = append(out, d)
				blocks = append(blocks, r.Inset(-decoClear))
			}
		}
	}
	return out
}

func placeCompass(l KMapLayout, blocks []ui.Rect, size func(string) (int, int)) (DecoSpot, bool) {
	last := l.Eras[len(l.Eras)-1]
	for ty := l.Height - 4; ty >= 1; ty-- {
		for tx := last.Start + last.Size - 4; tx >= last.Start; tx-- {
			d := DecoSpot{Kind: DecoCompass, X: tx * KMapTile, Y: ty * KMapTile}
			if decorFits(d.Rect(size), blocks, l) {
				return d, true
			}
		}
	}
	return DecoSpot{}, false
}

func (h *HUD) decoSize(kind string) (int, int) {
	return h.Lib.Size(gfx.KMapDecorName(kind))
}

func (h *HUD) drawDecor(dst *ebiten.Image, l KMapLayout, roads []KMapRoadTile, p pen) {
	for _, d := range KnowledgeDecor(l, roads, h.decoSize) {
		r := p.at(d.Rect(h.decoSize))
		h.Lib.Draw(dst, gfx.KMapDecorName(d.Kind), float64(r.X), float64(r.Y), 1, render.SpriteOptions{Alpha: 0.8})
	}
}
