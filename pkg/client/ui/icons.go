package ui

import (
	"image"
	"image/color"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/waxb/kubernites/pkg/client/art"
)

const (
	IconGold     = ""
	IconWisdom   = ""
	IconFood     = ""
	IconHousing  = ""
	IconRenown   = ""
	IconBox      = ""
	IconParcel   = ""
	IconLedger   = ""
	IconCrate    = ""
	IconDispatch = ""
	IconSealed   = ""
	IconCellar   = ""
	iconGap      = ""
	iconFirst    = ''
	iconLast     = ''
	gapRune      = ''
	iconWidth    = 2*Advance - 1
	iconRows     = GlyphHeight + 2
	iconLift     = 1
	inkAlpha     = 128
)

type spriteSource func() (*image.NRGBA, bool)

func topBarArt(name string) spriteSource {
	return func() (*image.NRGBA, bool) { return art.PaintIcon(name) }
}

func panelBoxArt(kind string, sealed bool) spriteSource {
	return func() (*image.NRGBA, bool) { return art.PaintBoxIcon(kind, sealed) }
}

var iconSources = map[rune]spriteSource{
	'': topBarArt("gold"), '': topBarArt("wisdom"), '': topBarArt("food"),
	'': topBarArt("housing"), '': topBarArt("renown"),
	'': panelBoxArt("parcel", false), '': panelBoxArt("parcel", false), '': panelBoxArt("ledger", false),
	'': panelBoxArt("crate", false), '': panelBoxArt("dispatch", false), '': panelBoxArt("parcel", true),
	'': topBarArt("overlay-midden"),
}

var inlineIcons = sync.OnceValue(func() map[rune]*image.NRGBA {
	out := make(map[rune]*image.NRGBA, len(iconSources))
	for r, paint := range iconSources {
		if src, ok := paint(); ok {
			out[r] = textSized(src)
		}
	}
	return out
})

func inkBounds(src *image.NRGBA) image.Rectangle {
	b := image.Rectangle{}
	for y := src.Rect.Min.Y; y < src.Rect.Max.Y; y++ {
		for x := src.Rect.Min.X; x < src.Rect.Max.X; x++ {
			if src.NRGBAAt(x, y).A >= inkAlpha {
				b = b.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return b
}

func textSized(src *image.NRGBA) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, iconWidth, iconRows))
	b := inkBounds(src)
	if b.Empty() {
		return out
	}
	f := min(float64(iconWidth)/float64(b.Dx()), float64(iconRows)/float64(b.Dy()), 1)
	w, h := int(float64(b.Dx())*f+.5), int(float64(b.Dy())*f+.5)
	ox, oy := (iconWidth-w)/2, (iconRows-h)/2
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := src.NRGBAAt(b.Min.X+int((float64(x)+.5)/f), b.Min.Y+int((float64(y)+.5)/f))
			if c.A >= inkAlpha {
				c.A = 255
				out.SetNRGBA(ox+x, oy+y, c)
			}
		}
	}
	return out
}

var iconWords = map[rune]string{
	'\ue000': "gold", '\ue002': "wisdom", '\ue004': "food", '\ue006': "housing", '\ue008': "renown",
	'\ue00a': "box", '\ue00c': "parcel", '\ue00e': "ledger", '\ue010': "crate", '\ue012': "dispatch", '\ue014': "sealed box", '\ue016': "cellar",
}

func Spelled(text string) string {
	var b strings.Builder
	runes := []rune(text)
	for i, r := range runes {
		switch {
		case iconStart(r):
			b.WriteString("[" + iconWords[r] + "]")
			if i+2 >= len(runes) || runes[i+2] != gapRune {
				b.WriteRune(' ')
			}
		case iconRune(r) && r != gapRune:
		case r == gapRune:
			b.WriteRune(' ')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func Plain(text string) string {
	return strings.Map(func(r rune) rune {
		if iconRune(r) {
			return -1
		}
		return r
	}, text)
}

func IconLead(text string) int {
	runes := []rune(text)
	if len(runes) < 2 || !iconStart(runes[0]) {
		return 0
	}
	return 2
}

func iconStart(r rune) bool {
	return r >= iconFirst && r <= iconLast && (r-iconFirst)%2 == 0
}

func iconRune(r rune) bool {
	return r >= iconFirst && r <= iconLast || r == gapRune
}

func iconPixel(icon string, x, y int) color.Color {
	runes := []rune(icon)
	if len(runes) == 0 || x < 0 || y < 0 || x >= iconWidth || y >= iconRows {
		return nil
	}
	img, ok := inlineIcons()[runes[0]]
	if !ok {
		return nil
	}
	c := img.NRGBAAt(x, y)
	if c.A == 0 {
		return nil
	}
	return color.RGBA{c.R, c.G, c.B, 255}
}

func drawIcon(dst *ebiten.Image, start rune, x, y, scale int, silhouette color.Color) {
	img, ok := inlineIcons()[start]
	if !ok {
		return
	}
	for row := 0; row < iconRows; row++ {
		for col := 0; col < iconWidth; col++ {
			c := img.NRGBAAt(col, row)
			if c.A == 0 {
				continue
			}
			var clr color.Color = c
			if silhouette != nil {
				clr = silhouette
			}
			vector.FillRect(dst, float32(x+col*scale), float32(y+(row-iconLift)*scale), float32(scale), float32(scale), clr, false)
		}
	}
}

func cells(text string) int {
	n := 0
	for _, r := range text {
		if r != gapRune {
			n++
		}
	}
	return n
}

func cutCells(runes []rune, n int) int {
	at, used := 0, 0
	for at < len(runes) && (runes[at] == gapRune || used < n) {
		if runes[at] != gapRune {
			used++
		}
		at++
	}
	at = safeCut(runes, at)
	for at > 0 && (runes[at-1] == gapRune || iconStart(runes[at-1])) {
		at--
	}
	return at
}

func safeCut(runes []rune, n int) int {
	n = min(max(n, 0), len(runes))
	if n > 0 && n < len(runes) && iconStart(runes[n-1]) {
		return n - 1
	}
	return n
}
