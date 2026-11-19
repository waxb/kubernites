package gfx

import (
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
)

func TestGildedSpritesAreUploadedInTheSameOrderEveryTime(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 64, 16))
	regions := map[string]art.Region{}
	for i := 0; i < 8; i++ {
		r := art.Region{X: i * 8, W: 2 + i%6, H: 4 + i%3}
		for y := r.Y; y < r.Y+r.H; y++ {
			for x := r.X; x < r.X+r.W; x++ {
				src.SetNRGBA(x, y, color.NRGBA{uint8(40 * i), 200, 120, 255})
			}
		}
		regions[fmt.Sprintf("ship/%s/%d/%d", GildedClass, GildedVariant, i)] = r
	}
	saved := newImage
	t.Cleanup(func() { newImage = saved })
	var got, first []string
	newImage = func(img image.Image) *ebiten.Image {
		nrgba := img.(*image.NRGBA)
		got = append(got, fmt.Sprintf("%v %x", nrgba.Bounds(), sha256.Sum256(nrgba.Pix)))
		return nil
	}
	for pass := 0; pass < 40; pass++ {
		got = got[:0]
		(&Library{index: map[string]entry{}}).addGilded(art.Atlas{Image: src, Regions: regions})
		if len(got) != len(regions) {
			t.Fatalf("pass %d gilded %d sprites, want %d", pass, len(got), len(regions))
		}
		if first == nil {
			first = slices.Clone(got)
			continue
		}
		if !slices.Equal(got, first) {
			t.Fatalf("pass %d uploaded the gilded sprites in another order:\n%v\nthe first pass:\n%v", pass, got, first)
		}
	}
}
