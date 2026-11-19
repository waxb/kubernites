package art

import (
	"image"
	"image/color"
	"image/draw"
)

func (l *Land) Bounds() image.Rectangle {
	nx, ny := Chunks(l.W, l.H)
	b := image.Rectangle{}
	for cy := 0; cy < ny; cy++ {
		for cx := 0; cx < nx; cx++ {
			b = b.Union(ChunkBounds(cx, cy))
		}
	}
	return b
}

func (l *Land) Render(frame int) *image.NRGBA {
	b := l.Bounds()
	out := image.NewNRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			out.SetNRGBA(x, y, seaColor(mod(x, SeaWidth), mod(y, SeaHeight), frame))
		}
	}
	nx, ny := Chunks(l.W, l.H)
	for s := 0; s <= nx+ny-2; s++ {
		for cx := 0; cx < nx; cx++ {
			if cy := s - cx; cy >= 0 && cy < ny {
				img := l.Chunk(cx, cy)
				draw.Draw(out, img.Rect, img, img.Rect.Min, draw.Over)
			}
		}
	}
	return out
}

func mod(v, m int) int {
	return ((v % m) + m) % m
}

func Project(gx, gy, lift float64) image.Point {
	return image.Pt(int((gx-gy)*16), int((gx+gy)*8-lift))
}

func Stamp(dst *image.NRGBA, a Atlas, name string, at image.Point, flip bool) {
	r, ok := a.Regions[name]
	if !ok {
		return
	}
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			sx := r.X + x
			if flip {
				sx = r.X + r.W - 1 - x
			}
			c := a.Image.NRGBAAt(sx, r.Y+y)
			p := image.Pt(at.X-r.AnchorX+x, at.Y-r.AnchorY+y)
			if c.A == 0 || !p.In(dst.Rect) {
				continue
			}
			dst.SetNRGBA(p.X, p.Y, over(dst.NRGBAAt(p.X, p.Y), c))
		}
	}
}

func over(dst, src color.NRGBA) color.NRGBA {
	if src.A == 255 || dst.A == 0 {
		return src
	}
	a := float64(src.A) / 255
	m := func(d, s uint8) uint8 { return uint8(float64(d)*(1-a) + float64(s)*a) }
	return color.NRGBA{m(dst.R, src.R), m(dst.G, src.G), m(dst.B, src.B), max(dst.A, src.A)}
}
