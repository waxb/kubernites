package hud

import (
	"image"
	"image/color"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	bookPx       = 2
	bookArtW     = 440
	bookArtH     = 252
	artPageL     = 12
	artPageR     = 427
	artSpineL    = 218
	artSpineR    = 221
	artPageT     = 14
	artPageB     = 232
	artCoverL    = 6
	ArtCoverR    = 433
	artCoverT    = 16
	artCoverB    = 246
	artBandT     = 235
	artBandStep  = 20
	artNotchHalf = 10
	artNotchT    = 229
	artNotchB    = 249
	artBorder    = 4
	artTailX     = 40
	artTailB     = 251
)

var (
	bookOutline   = color.RGBA{22, 12, 14, 255}
	bookPaper     = color.RGBA{230, 212, 178, 255}
	bookPaperLit  = color.RGBA{238, 222, 190, 255}
	bookRule      = color.RGBA{214, 182, 148, 255}
	bookEdgeLight = color.RGBA{214, 180, 146, 255}
	bookEdgeDark  = color.RGBA{196, 150, 112, 255}
	bookEdgeDeep  = color.RGBA{178, 132, 98, 255}
	bookGroove    = color.RGBA{168, 120, 88, 255}
	bookCover     = color.RGBA{96, 42, 40, 255}
	bookCoverLit  = color.RGBA{166, 90, 48, 255}
	bookCoverDeep = color.RGBA{54, 26, 32, 255}
	bookBand      = []color.RGBA{{54, 24, 30, 255}, {100, 48, 48, 255}, {100, 48, 48, 255}, {112, 56, 52, 255}, {112, 56, 52, 255}, {112, 56, 52, 255}, {92, 42, 44, 255}, {92, 42, 44, 255}, {70, 32, 36, 255}, {70, 32, 36, 255}, {58, 26, 30, 255}}
	bookNotch     = color.RGBA{52, 26, 36, 255}
	bookNotchLit  = color.RGBA{84, 42, 56, 255}
	bookTailRed   = color.RGBA{196, 64, 48, 255}
	bookTailDark  = color.RGBA{136, 38, 34, 255}
)

var (
	bookArtOnce  sync.Once
	bookArtImage *ebiten.Image
)

func ArtX(x int) int { return BookBox.X + x*bookPx }
func artY(y int) int { return BookBox.Y + y*bookPx }

func bookOuter(side int) int {
	if side == 0 {
		return ArtX(artPageL)
	}
	return ArtX(artPageR + 1)
}

func PageTopY() int    { return artY(artPageT) }
func pageBottomY() int { return artY(artPageB + 1) }

type artCanvas struct{ img *image.RGBA }

func (c artCanvas) rect(x, y, w, h int, clr color.RGBA) {
	for j := y; j < y+h; j++ {
		for i := x; i < x+w; i++ {
			c.img.SetRGBA(i, j, clr)
		}
	}
}

func bookArt() *ebiten.Image {
	bookArtOnce.Do(func() {
		c := artCanvas{img: image.NewRGBA(image.Rect(0, 0, bookArtW, bookArtH))}
		paintCover(c)
		for side := range 2 {
			paintStack(c, side)
			paintPage(c, side)
		}
		paintGutter(c)
		paintNotch(c)
		paintTail(c)
		bookArtImage = ebiten.NewImageFromImage(c.img)
	})
	return bookArtImage
}

func pageSpan(side int) (int, int) {
	if side == 0 {
		return artPageL, artSpineL
	}
	return artSpineR, artPageR
}

func paintCover(c artCanvas) {
	c.rect(artCoverL-1, artCoverT-1, ArtCoverR-artCoverL+3, artCoverB-artCoverT+3, bookOutline)
	c.rect(artCoverL, artCoverT, ArtCoverR-artCoverL+1, artCoverB-artCoverT+1, bookCover)
	c.rect(artCoverL, artCoverT+2, 1, artBandT-artCoverT-2, bookCoverLit)
	c.rect(ArtCoverR, artCoverT+2, 1, artBandT-artCoverT-2, bookCoverLit)
	c.rect(artCoverL+2, artCoverT, 1, artBandT-artCoverT, bookCoverDeep)
	c.rect(ArtCoverR-2, artCoverT, 1, artBandT-artCoverT, bookCoverDeep)
	for i, clr := range bookBand {
		c.rect(artCoverL, artBandT+i, ArtCoverR-artCoverL+1, 1, clr)
	}
	for x := artCoverL + artBandStep; x < ArtCoverR; x += artBandStep {
		c.rect(x, artBandT, 1, artCoverB-artBandT+1, bookOutline)
	}
}

func paintStack(c artCanvas, side int) {
	l, r := pageSpan(side)
	outer, step := l-1, -1
	if side == 1 {
		outer, step = r+1, 1
	}
	for k, clr := range []color.RGBA{bookEdgeLight, bookEdgeDark, bookEdgeDeep} {
		c.rect(outer+k*step, artPageT+3+k, 1, artPageB-artPageT-2, clr)
	}
	for k, clr := range []color.RGBA{bookEdgeLight, bookEdgeDark, bookEdgeLight, bookEdgeDeep} {
		c.rect(min(l, outer+2*step), artPageB+1+k, r-l+3, 1, clr)
	}
}

func paintPage(c artCanvas, side int) {
	l, r := pageSpan(side)
	c.rect(l, artPageT, r-l+1, artPageB-artPageT+1, bookPaper)
	c.rect(l, artPageT, r-l+1, 1, bookEdgeDark)
	c.rect(l, artPageT+1, r-l+1, 1, bookPaperLit)
	outer := l
	if side == 1 {
		outer = r
	}
	c.rect(outer, artPageT, 1, artPageB-artPageT+1, bookEdgeLight)
	liftCorner(c, side, outer)
	c.rect(l, artPageB-1, r-l+1, 1, bookRule)
	c.rect(l, artPageB, r-l+1, 1, bookEdgeLight)
	paintBorder(c, l+artBorder, artPageT+artBorder, r-l+1-2*artBorder, artPageB-artPageT+1-2*artBorder)
}

func liftCorner(c artCanvas, side, outer int) {
	dir := 1
	if side == 1 {
		dir = -1
	}
	for k := range 3 {
		c.rect(outer+dir*k, artPageT+2-k, 1, 1, bookCover)
		c.rect(outer+dir*(k+1), artPageT+2-k, 1, 1, bookEdgeDark)
	}
	c.rect(outer, artPageT+3, 1, 1, bookEdgeDark)
}

func paintBorder(c artCanvas, x, y, w, h int) {
	c.rect(x, y, w, 1, bookRule)
	c.rect(x, y+h-1, w, 1, bookRule)
	c.rect(x, y, 1, h, bookRule)
	c.rect(x+w-1, y, 1, h, bookRule)
	for _, corner := range [][2]int{{x + 2, y + 2}, {x + w - 4, y + 2}, {x + 2, y + h - 4}, {x + w - 4, y + h - 4}} {
		c.rect(corner[0], corner[1], 2, 2, bookRule)
	}
}

func paintGutter(c artCanvas) {
	c.rect(artSpineL-2, artPageT, 2, artPageB-artPageT+1, bookEdgeLight)
	c.rect(artSpineR+1, artPageT, 2, artPageB-artPageT+1, bookEdgeLight)
	c.rect(artSpineL, artPageT, 1, artPageB-artPageT+1, bookEdgeDark)
	c.rect(artSpineR, artPageT, 1, artPageB-artPageT+1, bookEdgeDark)
	c.rect(artSpineL+1, artPageT+2, 2, artPageB-artPageT-1, bookGroove)
	c.rect(artSpineL-1, artPageT, 6, 2, bookCover)
	for _, x := range []int{artSpineL - 6, artSpineR + 6} {
		for y := artPageT + 8; y < artPageB-8; y++ {
			if (y/9)%3 != 1 {
				c.rect(x, y, 1, 1, bookRule)
			}
		}
	}
}

func paintNotch(c artCanvas) {
	mid := (artSpineL + artSpineR + 1) / 2
	for y := artNotchT; y <= artNotchB; y++ {
		half := artNotchHalf
		if k := y - artNotchT; k < 4 {
			half = artNotchHalf - (4-k)*2
		}
		c.rect(mid-half-1, y, 2*half+2, 1, bookOutline)
		if y < artNotchB {
			c.rect(mid-half, y, 2*half, 1, bookNotch)
		}
	}
	c.rect(mid-artNotchHalf+3, artNotchT+6, 2*artNotchHalf-6, 1, bookNotchLit)
}

func paintTail(c artCanvas) {
	c.rect(artTailX-1, artPageB-2, 5, artTailB-artPageB+2, bookOutline)
	c.rect(artTailX, artPageB-2, 3, artTailB-artPageB+1, bookTailRed)
	c.rect(artTailX+2, artPageB-2, 1, artTailB-artPageB+1, bookTailDark)
	c.rect(artTailX+1, artTailB-2, 1, 2, bookOutline)
	c.rect(artTailX-1, artTailB, 5, 1, color.RGBA{})
	c.rect(artTailX, artTailB-1, 1, 1, bookOutline)
	c.rect(artTailX+2, artTailB-1, 1, 1, bookOutline)
}

func tone(c color.RGBA, shade, lift float64) color.RGBA {
	ch := func(v uint8) uint8 { return uint8(min(max(float64(v)*shade+lift, 0), 255)) }
	return color.RGBA{ch(c.R), ch(c.G), ch(c.B), 255}
}
