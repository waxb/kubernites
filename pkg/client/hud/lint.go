package hud

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const strayAlpha = 8

type Stray struct {
	Layer  string  `json:"layer"`
	Pixels int     `json:"pixels"`
	Box    ui.Rect `json:"box"`
}

func (s Stray) String() string {
	return fmt.Sprintf("layer %s paints %d pixels outside its registered rectangles, within %d,%d %dx%d", s.Layer, s.Pixels, s.Box.X, s.Box.Y, s.Box.W, s.Box.H)
}

func (h *HUD) Strays(f Frame, scratch *ebiten.Image, read func(*ebiten.Image) *image.RGBA) []Stray {
	f = h.Stage(f)
	out := []Stray{}
	for _, l := range HudLayers {
		rects := []ui.Rect{}
		for _, s := range l.Surfaces(h, f) {
			rects = append(rects, s.Rect)
		}
		scratch.Clear()
		l.Draw(h, scratch, f)
		if s, ok := strayIn(read(scratch), rects); ok {
			s.Layer = l.Name
			out = append(out, s)
		}
	}
	return out
}

func strayIn(img *image.RGBA, rects []ui.Rect) (Stray, bool) {
	b := img.Bounds()
	x0, y0, x1, y1, n := b.Max.X, b.Max.Y, -1, -1, 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if img.Pix[img.PixOffset(x, y)+3] < strayAlpha || inAny(rects, x, y) {
				continue
			}
			n++
			x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
		}
	}
	if n == 0 {
		return Stray{}, false
	}
	return Stray{Pixels: n, Box: ui.Rect{X: x0, Y: y0, W: x1 - x0 + 1, H: y1 - y0 + 1}}, true
}

func inAny(rects []ui.Rect, x, y int) bool {
	for _, r := range rects {
		if r.Contains(x, y) {
			return true
		}
	}
	return false
}
