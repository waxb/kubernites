package gfx

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
)

const (
	opaqueAlpha = 128
	maskWord    = 64
)

type Mask struct {
	W, H  int
	words int
	bits  []uint64
}

func MaskOf(img *image.NRGBA, r art.Region) Mask {
	area := image.Rect(r.X, r.Y, r.X+r.W, r.Y+r.H)
	if img == nil || area.Empty() {
		return Mask{}
	}
	words := (r.W + maskWord - 1) / maskWord
	m := Mask{W: r.W, H: r.H, words: words, bits: make([]uint64, words*r.H)}
	in := area.Intersect(img.Bounds())
	for py := in.Min.Y; py < in.Max.Y; py++ {
		row := img.Pix[img.PixOffset(in.Min.X, py):img.PixOffset(in.Max.X, py)]
		bits := m.bits[(py-r.Y)*words:]
		for i, x := 3, in.Min.X-r.X; i < len(row); i, x = i+4, x+1 {
			if row[i] >= opaqueAlpha {
				bits[x/maskWord] |= 1 << uint(x%maskWord)
			}
		}
	}
	return m
}

func (m Mask) Empty() bool {
	for _, w := range m.bits {
		if w != 0 {
			return false
		}
	}
	return true
}

func (m Mask) At(x, y int) bool {
	if x < 0 || y < 0 || x >= m.W || y >= m.H {
		return false
	}
	return m.bits[y*m.words+x/maskWord]&(1<<uint(x%maskWord)) != 0
}

func (m Mask) Near(x, y, reach int) bool {
	for yy := max(y-reach, 0); yy <= min(y+reach, m.H-1); yy++ {
		for xx := max(x-reach, 0); xx <= min(x+reach, m.W-1); xx++ {
			if m.At(xx, yy) {
				return true
			}
		}
	}
	return false
}

func newEntry(sheet *ebiten.Image, img *image.NRGBA, r art.Region) entry {
	return entry{sheet: sheet, region: r, top: ContentTop(img, r), mask: MaskOf(img, r)}
}

func (l *Library) Mask(name string) (Mask, bool) {
	e, ok := l.entry(name)
	if !ok || e.mask.W == 0 {
		return Mask{}, false
	}
	return e.mask, true
}

func (l *Library) Covers(name string, sx, sy, zoom float64, flip bool, x, y int, pad float64) bool {
	e, ok := l.entry(name)
	if !ok || e.mask.W == 0 || zoom <= 0 {
		return false
	}
	r := e.region
	left := math.Round(sx - float64(r.AnchorX)*zoom)
	if flip {
		left = math.Round(sx - float64(r.W-r.AnchorX)*zoom)
	}
	top := math.Round(sy - float64(r.AnchorY)*zoom)
	lx := int(math.Floor((float64(x) + .5 - left) / zoom))
	ly := int(math.Floor((float64(y) + .5 - top) / zoom))
	if flip {
		lx = r.W - 1 - lx
	}
	return e.mask.Near(lx, ly, int(math.Ceil(max(pad, 0)/zoom)))
}
