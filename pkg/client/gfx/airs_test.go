package gfx

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/render"
)

func TestAirShaderCompiles(t *testing.T) {
	if _, err := AirShader(); err != nil {
		t.Fatal(err)
	}
	if AirSkins() != 4 || AirHairs() != 6 {
		t.Errorf("skin ramps %d, hairs %d", AirSkins(), AirHairs())
	}
}

func TestAiredDrawLandsWhereAPlainDrawDoes(t *testing.T) {
	r := render.Region{X: 3, Y: 4, W: 10, H: 20, AnchorX: 5, AnchorY: 18}
	for _, o := range []render.SpriteOptions{{}, {Flip: true}, {Scale: 2}} {
		geo := SpriteGeo(r, 100.4, 50.6, 2, o)
		x0, y0 := geo.Apply(0, 0)
		x1, y1 := geo.Apply(float64(r.W), float64(r.H))
		lo, hi := min(x0, x1), max(x0, x1)
		z := 2.0
		if o.Scale > 0 {
			z *= o.Scale
		}
		if hi-lo != float64(r.W)*z || y1-y0 != float64(r.H)*z {
			t.Errorf("%+v: quad %v..%v x %v..%v", o, lo, hi, y0, y1)
		}
		if o.Flip && x0 < x1 {
			t.Error("a flipped sprite mirrors")
		}
	}
	dst := ebiten.NewImage(64, 64)
	defer dst.Deallocate()
	sheet := ebiten.NewImage(32, 32)
	defer sheet.Deallocate()
	DrawRegionAir(dst, sheet, r, SpriteGeo(r, 30, 40, 1, render.SpriteOptions{}), [3]float32{}, 1, Air{On: true, Skin: 2, Hair: 3})
}
